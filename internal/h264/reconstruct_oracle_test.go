// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const reconstructOracleC = `
#include <stddef.h>
#include <stdint.h>
#include <stdio.h>
#include <string.h>

#define BIT_DEPTH 8
#include "h264pred_template.c"
#undef BIT_DEPTH

#define BIT_DEPTH 8
#include "h264idct_template.c"
#undef BIT_DEPTH

#define MB_TYPE_INTRA16x16 (1U << 1)

#define MB_WIDTH 4
#define MB_HEIGHT 4
#define LUMA_STRIDE 80
#define CHROMA_STRIDE 48
#define NNZ_SIZE (15 * 8)

enum {
    LUMA_DC_BLOCK_INDEX = 48,
    CHROMA_DC_BLOCK_INDEX = 49,
};

typedef struct Pic {
    uint8_t y[LUMA_STRIDE * MB_HEIGHT * 16];
    uint8_t cb[CHROMA_STRIDE * MB_HEIGHT * 16];
    uint8_t cr[CHROMA_STRIDE * MB_HEIGHT * 16];
    int chroma_idc;
    int chroma_stride;
} Pic;

static void fill_plane(uint8_t *p, int n, int seed)
{
    for (int i = 0; i < n; i++)
        p[i] = (uint8_t)((seed + i * 13 + (i >> 4) * 7) & 255);
}

static void init_pic(Pic *p, int chroma_idc, int seed)
{
    memset(p, 0, sizeof(*p));
    p->chroma_idc = chroma_idc;
    p->chroma_stride = CHROMA_STRIDE;
    fill_plane(p->y, sizeof(p->y), seed);
    fill_plane(p->cb, sizeof(p->cb), seed + 29);
    fill_plane(p->cr, sizeof(p->cr), seed + 71);
}

static uint8_t *pic_y(Pic *p, int mb_x, int mb_y)
{
    return p->y + mb_y * 16 * LUMA_STRIDE + mb_x * 16;
}

static uint8_t *pic_cb(Pic *p, int mb_x, int mb_y)
{
    if (p->chroma_idc == 1)
        return p->cb + mb_y * 8 * p->chroma_stride + mb_x * 8;
    return p->cb + mb_y * 16 * p->chroma_stride + mb_x * 8;
}

static uint8_t *pic_cr(Pic *p, int mb_x, int mb_y)
{
    if (p->chroma_idc == 1)
        return p->cr + mb_y * 8 * p->chroma_stride + mb_x * 8;
    return p->cr + mb_y * 16 * p->chroma_stride + mb_x * 8;
}

static void init_offsets(int *offset, int luma_stride, int chroma_stride)
{
    const int base = scan8[0];
    for (int i = 0; i < 16; i++) {
        int delta = scan8[i] - base;
        offset[i] = 4 * (delta & 7) + 4 * luma_stride * (delta >> 3);
        offset[16 + i] = 4 * (delta & 7) + 4 * chroma_stride * (delta >> 3);
        offset[32 + i] = offset[16 + i];
    }
}

static void init_residual_420(int16_t *mb, int16_t *luma_dc, uint8_t *nnz)
{
    memset(mb, 0, 48 * 16 * sizeof(*mb));
    memset(luma_dc, 0, 16 * sizeof(*luma_dc));
    memset(nnz, 0, NNZ_SIZE);

    nnz[scan8[LUMA_DC_BLOCK_INDEX]] = 1;
    luma_dc[0] = 3;
    luma_dc[5] = -2;
    luma_dc[10] = 4;
    luma_dc[15] = 1;

    nnz[scan8[0]] = 2;
    mb[0] = 10;
    mb[1] = -4;
    nnz[scan8[5]] = 1;
    mb[5 * 16] = 12;

    nnz[scan8[CHROMA_DC_BLOCK_INDEX + 0]] = 1;
    mb[16 * 16 + 0] = 2;
    mb[16 * 16 + 16] = -1;
    mb[16 * 16 + 32] = 3;
    mb[16 * 16 + 48] = 1;
    nnz[scan8[16]] = 2;
    mb[16 * 16 + 1] = 5;

    nnz[scan8[CHROMA_DC_BLOCK_INDEX + 1]] = 1;
    mb[32 * 16 + 0] = -2;
    mb[32 * 16 + 16] = 4;
    mb[32 * 16 + 32] = 1;
    mb[32 * 16 + 48] = -3;
    nnz[scan8[32]] = 2;
    mb[32 * 16 + 2] = -6;
}

static void init_residual_422(int16_t *mb, int16_t *luma_dc, uint8_t *nnz)
{
    init_residual_420(mb, luma_dc, nnz);
    nnz[scan8[20]] = 1;
    mb[20 * 16] = 9;
    nnz[scan8[36]] = 1;
    mb[36 * 16] = -7;
    mb[16 * 16 + 64] = 5;
    mb[16 * 16 + 80] = -4;
    mb[16 * 16 + 96] = 2;
    mb[16 * 16 + 112] = 1;
    mb[32 * 16 + 64] = -5;
    mb[32 * 16 + 80] = 3;
    mb[32 * 16 + 96] = 2;
    mb[32 * 16 + 112] = -1;
}

static void print_mb(const char *label, Pic *p, int mb_x, int mb_y)
{
    uint8_t *y = pic_y(p, mb_x, mb_y);
    uint8_t *cb = pic_cb(p, mb_x, mb_y);
    uint8_t *cr = pic_cr(p, mb_x, mb_y);
    int ch = p->chroma_idc == 1 ? 8 : 16;

    printf("%s y", label);
    for (int yy = 0; yy < 16; yy++)
        for (int x = 0; x < 16; x++)
            printf(" %u", y[yy * LUMA_STRIDE + x]);
    printf("\n");

    printf("%s cb", label);
    for (int yy = 0; yy < ch; yy++)
        for (int x = 0; x < 8; x++)
            printf(" %u", cb[yy * p->chroma_stride + x]);
    printf("\n");

    printf("%s cr", label);
    for (int yy = 0; yy < ch; yy++)
        for (int x = 0; x < 8; x++)
            printf(" %u", cr[yy * p->chroma_stride + x]);
    printf("\n");
}

static void run_intra16x16_420(void)
{
    Pic dst;
    int offset[48];
    int16_t mb[48 * 16];
    int16_t luma_dc[16];
    uint8_t nnz[NNZ_SIZE];
    uint8_t *dest[2];
    const int mb_x = 1, mb_y = 1;
    init_pic(&dst, 1, 17);
    init_offsets(offset, LUMA_STRIDE, CHROMA_STRIDE);
    init_residual_420(mb, luma_dc, nnz);

    pred8x8_horizontal_8_c(pic_cb(&dst, mb_x, mb_y), CHROMA_STRIDE);
    pred8x8_horizontal_8_c(pic_cr(&dst, mb_x, mb_y), CHROMA_STRIDE);
    pred16x16_vertical_8_c(pic_y(&dst, mb_x, mb_y), LUMA_STRIDE);
    if (nnz[scan8[LUMA_DC_BLOCK_INDEX]])
        ff_h264_luma_dc_dequant_idct_8_c(mb, luma_dc, 64);
    ff_h264_idct_add16intra_8_c(pic_y(&dst, mb_x, mb_y), offset, mb, LUMA_STRIDE, nnz);
    if (nnz[scan8[CHROMA_DC_BLOCK_INDEX + 0]])
        ff_h264_chroma_dc_dequant_idct_8_c(mb + 16 * 16, 64);
    if (nnz[scan8[CHROMA_DC_BLOCK_INDEX + 1]])
        ff_h264_chroma_dc_dequant_idct_8_c(mb + 32 * 16, 64);
    dest[0] = pic_cb(&dst, mb_x, mb_y);
    dest[1] = pic_cr(&dst, mb_x, mb_y);
    ff_h264_idct_add8_8_c(dest, offset, mb, CHROMA_STRIDE, nnz);
    print_mb("intra16x16_420", &dst, mb_x, mb_y);
}

static void run_intra16x16_422(void)
{
    Pic dst;
    int offset[48];
    int16_t mb[48 * 16];
    int16_t luma_dc[16];
    uint8_t nnz[NNZ_SIZE];
    uint8_t *dest[2];
    const int mb_x = 1, mb_y = 1;
    init_pic(&dst, 2, 31);
    init_offsets(offset, LUMA_STRIDE, CHROMA_STRIDE);
    init_residual_422(mb, luma_dc, nnz);

    pred8x16_vertical_8_c(pic_cb(&dst, mb_x, mb_y), CHROMA_STRIDE);
    pred8x16_vertical_8_c(pic_cr(&dst, mb_x, mb_y), CHROMA_STRIDE);
    pred16x16_dc_8_c(pic_y(&dst, mb_x, mb_y), LUMA_STRIDE);
    if (nnz[scan8[LUMA_DC_BLOCK_INDEX]])
        ff_h264_luma_dc_dequant_idct_8_c(mb, luma_dc, 64);
    ff_h264_idct_add16intra_8_c(pic_y(&dst, mb_x, mb_y), offset, mb, LUMA_STRIDE, nnz);
    if (nnz[scan8[CHROMA_DC_BLOCK_INDEX + 0]])
        ff_h264_chroma422_dc_dequant_idct_8_c(mb + 16 * 16, 64);
    if (nnz[scan8[CHROMA_DC_BLOCK_INDEX + 1]])
        ff_h264_chroma422_dc_dequant_idct_8_c(mb + 32 * 16, 64);
    dest[0] = pic_cb(&dst, mb_x, mb_y);
    dest[1] = pic_cr(&dst, mb_x, mb_y);
    ff_h264_idct_add8_422_8_c(dest, offset, mb, CHROMA_STRIDE, nnz);
    print_mb("intra16x16_422", &dst, mb_x, mb_y);
}

int main(void)
{
    run_intra16x16_420();
    run_intra16x16_422();
    return 0;
}
`

const reconstructOracleBitDepthTemplate = `
#include <stdint.h>

#ifndef GOH264_RECON_BITDEPTH_HELPERS
#define GOH264_RECON_BITDEPTH_HELPERS
static inline uint8_t goh264_recon_clip_uint8(int v)
{
    if (v < 0)
        return 0;
    if (v > 255)
        return 255;
    return (uint8_t)v;
}

static inline uint32_t goh264_recon_rn4pa(const void *p)
{
    const uint8_t *b = (const uint8_t *)p;
    return (uint32_t)b[0] | ((uint32_t)b[1] << 8) |
           ((uint32_t)b[2] << 16) | ((uint32_t)b[3] << 24);
}

static inline void goh264_recon_wn4pa(void *p, uint32_t v)
{
    uint8_t *b = (uint8_t *)p;
    b[0] = v;
    b[1] = v >> 8;
    b[2] = v >> 16;
    b[3] = v >> 24;
}
#endif

#undef pixel
#undef pixel2
#undef pixel4
#undef dctcoef
#undef SUINT
#undef FUNC3
#undef FUNC2
#undef FUNC
#undef FUNCC
#undef AV_RN4PA
#undef AV_WN4PA
#undef PIXEL_SPLAT_X4
#undef CLIP
#undef av_clip_pixel

#define av_unused
#define pixel uint8_t
#define pixel2 uint16_t
#define pixel4 uint32_t
#define dctcoef int16_t
#define SUINT unsigned
#define FUNC3(a, b, c)  a ## _ ## b ## c
#define FUNC2(a, b, c)  FUNC3(a, b, c)
#define FUNC(a) FUNC2(a, BIT_DEPTH, _c)
#define FUNCC(a) FUNC2(a, BIT_DEPTH, _c)
#define AV_RN4PA(p) goh264_recon_rn4pa(p)
#define AV_WN4PA(p, v) goh264_recon_wn4pa((void *)(p), (uint32_t)(v))
#define PIXEL_SPLAT_X4(x) ((uint32_t)(uint8_t)(x) * 0x01010101U)
#define CLIP(a) goh264_recon_clip_uint8(a)
#define av_clip_pixel(a) goh264_recon_clip_uint8(a)
`

func TestH264ReconstructUpstreamOracle(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run pinned FFmpeg H.264 reconstruction oracle")
	}
	cc, err := exec.LookPath("cc")
	if err != nil {
		t.Skip("cc not available")
	}

	root := h264RepoRoot(t)
	upstreamDir := filepath.Join(root, ".upstream", "ffmpeg-n8.0.1", "libavcodec")
	predTemplate, err := os.ReadFile(filepath.Join(upstreamDir, "h264pred_template.c"))
	if err != nil {
		t.Skipf("pinned upstream H.264 prediction source not available: %v", err)
	}
	idctTemplate, err := os.ReadFile(filepath.Join(upstreamDir, "h264idct_template.c"))
	if err != nil {
		t.Skipf("pinned upstream H.264 IDCT source not available: %v", err)
	}

	dir := t.TempDir()
	writeOracleFile(t, filepath.Join(dir, "oracle.c"), reconstructOracleC)
	writeOracleFile(t, filepath.Join(dir, "h264pred_template.c"), string(predTemplate))
	writeOracleFile(t, filepath.Join(dir, "h264idct_template.c"), string(idctTemplate))
	writeOracleFile(t, filepath.Join(dir, "bit_depth_template.c"), reconstructOracleBitDepthTemplate)
	writeOracleFile(t, filepath.Join(dir, "h264_parse.h"), idctOracleH264Parse)
	writeOracleFile(t, filepath.Join(dir, "h264idct.h"), idctOracleH264IDCTH)
	writeOracleFile(t, filepath.Join(dir, "mathops.h"), "")
	if err := os.Mkdir(filepath.Join(dir, "libavutil"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeOracleFile(t, filepath.Join(dir, "libavutil", "common.h"), "")
	writeOracleFile(t, filepath.Join(dir, "libavutil", "intreadwrite.h"), "")

	bin := filepath.Join(dir, "oracle")
	cmd := exec.Command(cc, "-std=c99", "-I"+dir, filepath.Join(dir, "oracle.c"), "-o", bin)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("compile H.264 reconstruction oracle: %v\n%s", err, out)
	}
	out, err := exec.Command(bin).CombinedOutput()
	if err != nil {
		t.Fatalf("run H.264 reconstruction oracle: %v\n%s", err, out)
	}

	got := strings.TrimSpace(string(out))
	want := strings.TrimSpace(h264ReconstructOracleWant(t))
	if got != want {
		t.Fatalf("H.264 reconstruction oracle mismatch\nC oracle:\n%s\nGo:\n%s", got, want)
	}
}

func h264ReconstructOracleWant(t *testing.T) string {
	var b strings.Builder
	appendH264ReconstructOracleIntra16x16(t, &b, "intra16x16_420", 1, 17, int32(intraPred8x8Horizontal), int8(intraPred8x8Vertical), h264ReconstructResidual420())
	appendH264ReconstructOracleIntra16x16(t, &b, "intra16x16_422", 2, 31, int32(intraPred8x8Vertical), int8(intraPred8x8DC), h264ReconstructResidual422())
	return b.String()
}

func appendH264ReconstructOracleIntra16x16(t *testing.T, b *strings.Builder, label string, chromaFormatIDC int, seed int, chromaPred int32, lumaPred int8, residual cavlcResidualContext) {
	dst := makeH264MotionCompPicture(chromaFormatIDC, seed)
	if err := h264HLDecodeFrameMacroblock(dst, h264FrameMBReconstructInput{
		MBType:             MBTypeIntra16x16,
		MBX:                1,
		MBY:                1,
		CBP:                0x31,
		QScale:             20,
		ChromaQP:           [2]uint8{20, 21},
		ChromaPredMode:     chromaPred,
		Intra16x16PredMode: lumaPred,
		PPS:                cavlcFlatQMulPPS(),
		Residual:           &residual,
	}); err != nil {
		t.Fatal(err)
	}
	printH264MotionCompMB(b, label, dst, 1, 1)
}
