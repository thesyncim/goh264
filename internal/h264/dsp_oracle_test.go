// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const dspOracleC = `
#include <stddef.h>
#include <stdint.h>
#include <stdio.h>
#include <string.h>

#define av_always_inline inline
#define av_flatten
#define FFABS(a) ((a) >= 0 ? (a) : -(a))
static inline int av_clip(int v, int amin, int amax)
{
    if (v < amin)
        return amin;
    if (v > amax)
        return amax;
    return v;
}

#define BIT_DEPTH 8
#include "h264dsp_template.c"
#undef BIT_DEPTH

#define BIT_DEPTH 8
#include "h264addpx_template.c"
#undef BIT_DEPTH

static int sum_i16(const int16_t *v, int n)
{
    int sum = 0;
    for (int i = 0; i < n; i++)
        sum += v[i];
    return sum;
}

static void print_add4(void)
{
    uint8_t dst[6 * 4];
    int16_t block[16] = {
        10, -2, 300, -300,
        1, 2, 3, 4,
        -5, -6, -7, -8,
        255, 256, -255, -256,
    };

    for (int i = 0; i < 6 * 4; i++)
        dst[i] = 20 + i * 3;

    ff_h264_add_pixels4_8_c(dst, block, 6);
    printf("add4");
    for (int y = 0; y < 4; y++)
        for (int x = 0; x < 6; x++)
            printf(" %u", dst[y * 6 + x]);
    printf(" blocksum %d\n", sum_i16(block, 16));
}

static void print_add8(void)
{
    uint8_t dst[10 * 8];
    int16_t block[64];

    for (int i = 0; i < 10 * 8; i++)
        dst[i] = 40 + i;
    for (int i = 0; i < 64; i++)
        block[i] = i - 32;

    ff_h264_add_pixels8_8_c(dst, block, 10);
    printf("add8");
    for (int y = 0; y < 8; y++)
        for (int x = 0; x < 10; x++)
            printf(" %u", dst[y * 10 + x]);
    printf(" blocksum %d\n", sum_i16(block, 64));
}

static void call_weight(int width, uint8_t *dst, int stride, int height,
                        int log2_denom, int weight, int offset)
{
    switch (width) {
    case 16:
        weight_h264_pixels16_8_c(dst, stride, height, log2_denom, weight, offset);
        break;
    case 8:
        weight_h264_pixels8_8_c(dst, stride, height, log2_denom, weight, offset);
        break;
    case 4:
        weight_h264_pixels4_8_c(dst, stride, height, log2_denom, weight, offset);
        break;
    case 2:
        weight_h264_pixels2_8_c(dst, stride, height, log2_denom, weight, offset);
        break;
    }
}

static void call_biweight(int width, uint8_t *dst, uint8_t *src, int stride,
                          int height, int log2_denom, int weightd,
                          int weights, int offset)
{
    switch (width) {
    case 16:
        biweight_h264_pixels16_8_c(dst, src, stride, height, log2_denom, weightd, weights, offset);
        break;
    case 8:
        biweight_h264_pixels8_8_c(dst, src, stride, height, log2_denom, weightd, weights, offset);
        break;
    case 4:
        biweight_h264_pixels4_8_c(dst, src, stride, height, log2_denom, weightd, weights, offset);
        break;
    case 2:
        biweight_h264_pixels2_8_c(dst, src, stride, height, log2_denom, weightd, weights, offset);
        break;
    }
}

static void print_weight_case(int width)
{
    uint8_t dst[20 * 3];

    for (int i = 0; i < 20 * 3; i++)
        dst[i] = (13 + i * 17 + width) & 255;

    call_weight(width, dst, 20, 3, 3, -5, -7);
    printf("weight%d", width);
    for (int y = 0; y < 3; y++)
        for (int x = 0; x < width; x++)
            printf(" %u", dst[y * 20 + x]);
    printf("\n");
}

static void print_weight_zero_denom(void)
{
    uint8_t dst[8] = { 0, 1, 16, 63, 64, 127, 200, 255 };

    call_weight(8, dst, 8, 1, 0, 1, -2);
    printf("weight0");
    for (int i = 0; i < 8; i++)
        printf(" %u", dst[i]);
    printf("\n");
}

static void print_biweight_case(int width)
{
    uint8_t dst[20 * 3];
    uint8_t src[20 * 3];

    for (int i = 0; i < 20 * 3; i++) {
        dst[i] = (5 + i * 11 + width * 3) & 255;
        src[i] = (250 - i * 7 + width) & 255;
    }

    call_biweight(width, dst, src, 20, 3, 2, 3, -2, -5);
    printf("biweight%d", width);
    for (int y = 0; y < 3; y++)
        for (int x = 0; x < width; x++)
            printf(" %u", dst[y * 20 + x]);
    printf("\n");
}

typedef void (*loop_filter_tc_fn)(uint8_t *pix, ptrdiff_t stride, int alpha, int beta, int8_t *tc0);
typedef void (*loop_filter_intra_fn)(uint8_t *pix, ptrdiff_t stride, int alpha, int beta);

static void init_loop_fixture(uint8_t *pix, int stride, int rows)
{
    for (int y = 0; y < rows; y++)
        for (int x = 0; x < stride; x++)
            pix[y * stride + x] = 80 + (x * 2 + y * 3) % 64;
}

static void print_loop_window(const char *label, uint8_t *pix, int stride, int offset)
{
    printf("%s", label);
    for (int y = -4; y < 12; y++)
        for (int x = -4; x < 12; x++)
            printf(" %u", pix[offset + y * stride + x]);
    printf("\n");
}

static void print_loop_tc_case(const char *label, loop_filter_tc_fn fn)
{
    const int stride = 32;
    const int offset = 12 * stride + 12;
    uint8_t pix[32 * 32];
    int8_t tc0[4] = { 2, 0, -1, 4 };

    init_loop_fixture(pix, stride, 32);
    fn(pix + offset, stride, 80, 80, tc0);
    print_loop_window(label, pix, stride, offset);
}

static void print_loop_intra_case(const char *label, loop_filter_intra_fn fn)
{
    const int stride = 32;
    const int offset = 12 * stride + 12;
    uint8_t pix[32 * 32];

    init_loop_fixture(pix, stride, 32);
    fn(pix + offset, stride, 80, 80);
    print_loop_window(label, pix, stride, offset);
}

int main(void)
{
    print_add4();
    print_add8();
    print_weight_case(2);
    print_weight_case(4);
    print_weight_case(8);
    print_weight_case(16);
    print_weight_zero_denom();
    print_biweight_case(2);
    print_biweight_case(4);
    print_biweight_case(8);
    print_biweight_case(16);
    print_loop_tc_case("vluma", h264_v_loop_filter_luma_8_c);
    print_loop_tc_case("hluma", h264_h_loop_filter_luma_8_c);
    print_loop_tc_case("hlumambaff", h264_h_loop_filter_luma_mbaff_8_c);
    print_loop_intra_case("vlumai", h264_v_loop_filter_luma_intra_8_c);
    print_loop_intra_case("hlumai", h264_h_loop_filter_luma_intra_8_c);
    print_loop_intra_case("hlumambaffi", h264_h_loop_filter_luma_mbaff_intra_8_c);
    print_loop_tc_case("vchroma", h264_v_loop_filter_chroma_8_c);
    print_loop_tc_case("hchroma", h264_h_loop_filter_chroma_8_c);
    print_loop_tc_case("hchromambaff", h264_h_loop_filter_chroma_mbaff_8_c);
    print_loop_tc_case("hchroma422", h264_h_loop_filter_chroma422_8_c);
    print_loop_tc_case("hchroma422mbaff", h264_h_loop_filter_chroma422_mbaff_8_c);
    print_loop_intra_case("vchromai", h264_v_loop_filter_chroma_intra_8_c);
    print_loop_intra_case("hchromai", h264_h_loop_filter_chroma_intra_8_c);
    print_loop_intra_case("hchromambaffi", h264_h_loop_filter_chroma_mbaff_intra_8_c);
    print_loop_intra_case("hchroma422i", h264_h_loop_filter_chroma422_intra_8_c);
    print_loop_intra_case("hchroma422mbaffi", h264_h_loop_filter_chroma422_mbaff_intra_8_c);
    return 0;
}
`

const dspOracleBitDepthTemplate = `
#include <stdint.h>

#ifndef GOH264_DSP_BITDEPTH_HELPERS
#define GOH264_DSP_BITDEPTH_HELPERS
static inline uint8_t goh264_dsp_clip_uint8(int v)
{
    if (v < 0)
        return 0;
    if (v > 255)
        return 255;
    return (uint8_t)v;
}
#endif

#undef pixel
#undef dctcoef
#undef FUNC3
#undef FUNC2
#undef FUNC
#undef FUNCC
#undef av_clip_pixel

#define pixel uint8_t
#define dctcoef int16_t
#define FUNC3(a, b, c)  a ## _ ## b ## c
#define FUNC2(a, b, c)  FUNC3(a, b, c)
#define FUNC(a) FUNC2(a, BIT_DEPTH, _c)
#define FUNCC(a) FUNC2(a, BIT_DEPTH, _c)
#define av_clip_pixel(a) goh264_dsp_clip_uint8(a)
`

func TestH264DSPUpstreamOracle(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run pinned FFmpeg H.264 DSP oracle")
	}
	cc, err := exec.LookPath("cc")
	if err != nil {
		t.Skip("cc not available")
	}

	root := h264RepoRoot(t)
	upstreamDir := filepath.Join(root, ".upstream", "ffmpeg-n8.0.1", "libavcodec")
	dspTemplate, err := os.ReadFile(filepath.Join(upstreamDir, "h264dsp_template.c"))
	if err != nil {
		t.Skipf("pinned upstream H.264 DSP source not available: %v", err)
	}
	addpxTemplate, err := os.ReadFile(filepath.Join(upstreamDir, "h264addpx_template.c"))
	if err != nil {
		t.Skipf("pinned upstream H.264 add-pixels source not available: %v", err)
	}

	dir := t.TempDir()
	writeOracleFile(t, filepath.Join(dir, "oracle.c"), dspOracleC)
	writeOracleFile(t, filepath.Join(dir, "h264dsp_template.c"), string(dspTemplate))
	writeOracleFile(t, filepath.Join(dir, "h264addpx_template.c"), string(addpxTemplate))
	writeOracleFile(t, filepath.Join(dir, "bit_depth_template.c"), dspOracleBitDepthTemplate)

	bin := filepath.Join(dir, "oracle")
	cmd := exec.Command(cc, "-std=c99", "-I"+dir, filepath.Join(dir, "oracle.c"), "-o", bin)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("compile H.264 DSP oracle: %v\n%s", err, out)
	}

	out, err := exec.Command(bin).CombinedOutput()
	if err != nil {
		t.Fatalf("run H.264 DSP oracle: %v\n%s", err, out)
	}

	got := strings.TrimSpace(string(out))
	want := strings.TrimSpace(h264DSPOracleWant(t))
	if got != want {
		t.Fatalf("H.264 DSP oracle mismatch\nC oracle:\n%s\nGo:\n%s", got, want)
	}
}

func h264DSPOracleWant(t *testing.T) string {
	t.Helper()
	var b strings.Builder
	printDSPAdd4OracleWant(t, &b)
	printDSPAdd8OracleWant(t, &b)
	for _, width := range []int{2, 4, 8, 16} {
		printDSPWeightOracleWant(t, &b, width)
	}
	printDSPWeightZeroDenomOracleWant(t, &b)
	for _, width := range []int{2, 4, 8, 16} {
		printDSPBiweightOracleWant(t, &b, width)
	}
	printDSPLoopTCOracleWant(t, &b, "vluma", h264VLoopFilterLuma)
	printDSPLoopTCOracleWant(t, &b, "hluma", h264HLoopFilterLuma)
	printDSPLoopTCOracleWant(t, &b, "hlumambaff", h264HLoopFilterLumaMBAFF)
	printDSPLoopIntraOracleWant(t, &b, "vlumai", h264VLoopFilterLumaIntra)
	printDSPLoopIntraOracleWant(t, &b, "hlumai", h264HLoopFilterLumaIntra)
	printDSPLoopIntraOracleWant(t, &b, "hlumambaffi", h264HLoopFilterLumaMBAFFIntra)
	printDSPLoopTCOracleWant(t, &b, "vchroma", h264VLoopFilterChroma)
	printDSPLoopTCOracleWant(t, &b, "hchroma", h264HLoopFilterChroma)
	printDSPLoopTCOracleWant(t, &b, "hchromambaff", h264HLoopFilterChromaMBAFF)
	printDSPLoopTCOracleWant(t, &b, "hchroma422", h264HLoopFilterChroma422)
	printDSPLoopTCOracleWant(t, &b, "hchroma422mbaff", h264HLoopFilterChroma422MBAFF)
	printDSPLoopIntraOracleWant(t, &b, "vchromai", h264VLoopFilterChromaIntra)
	printDSPLoopIntraOracleWant(t, &b, "hchromai", h264HLoopFilterChromaIntra)
	printDSPLoopIntraOracleWant(t, &b, "hchromambaffi", h264HLoopFilterChromaMBAFFIntra)
	printDSPLoopIntraOracleWant(t, &b, "hchroma422i", h264HLoopFilterChroma422Intra)
	printDSPLoopIntraOracleWant(t, &b, "hchroma422mbaffi", h264HLoopFilterChroma422MBAFFIntra)
	return b.String()
}

func printDSPAdd4OracleWant(t *testing.T, b *strings.Builder) {
	t.Helper()
	dst := make([]uint8, 6*4)
	block := []int32{
		10, -2, 300, -300,
		1, 2, 3, 4,
		-5, -6, -7, -8,
		255, 256, -255, -256,
	}
	for i := range dst {
		dst[i] = uint8(20 + i*3)
	}

	if err := h264AddPixels4Clear(dst, block, 6); err != nil {
		t.Fatal(err)
	}
	fmt.Fprint(b, "add4")
	for y := 0; y < 4; y++ {
		for x := 0; x < 6; x++ {
			fmt.Fprintf(b, " %d", dst[y*6+x])
		}
	}
	fmt.Fprintf(b, " blocksum %d\n", sumInt32(block))
}

func printDSPAdd8OracleWant(t *testing.T, b *strings.Builder) {
	t.Helper()
	dst := make([]uint8, 10*8)
	block := make([]int32, 64)
	for i := range dst {
		dst[i] = uint8(40 + i)
	}
	for i := range block {
		block[i] = int32(i - 32)
	}

	if err := h264AddPixels8Clear(dst, block, 10); err != nil {
		t.Fatal(err)
	}
	fmt.Fprint(b, "add8")
	for y := 0; y < 8; y++ {
		for x := 0; x < 10; x++ {
			fmt.Fprintf(b, " %d", dst[y*10+x])
		}
	}
	fmt.Fprintf(b, " blocksum %d\n", sumInt32(block))
}

func printDSPWeightOracleWant(t *testing.T, b *strings.Builder, width int) {
	t.Helper()
	dst := make([]uint8, 20*3)
	for i := range dst {
		dst[i] = uint8((13 + i*17 + width) & 255)
	}

	if err := h264WeightPixels(dst, 20, 3, 3, -5, -7, width); err != nil {
		t.Fatal(err)
	}
	fmt.Fprintf(b, "weight%d", width)
	for y := 0; y < 3; y++ {
		for x := 0; x < width; x++ {
			fmt.Fprintf(b, " %d", dst[y*20+x])
		}
	}
	fmt.Fprint(b, "\n")
}

func printDSPWeightZeroDenomOracleWant(t *testing.T, b *strings.Builder) {
	t.Helper()
	dst := []uint8{0, 1, 16, 63, 64, 127, 200, 255}

	if err := h264WeightPixels(dst, 8, 1, 0, 1, -2, 8); err != nil {
		t.Fatal(err)
	}
	fmt.Fprint(b, "weight0")
	for _, value := range dst {
		fmt.Fprintf(b, " %d", value)
	}
	fmt.Fprint(b, "\n")
}

func printDSPBiweightOracleWant(t *testing.T, b *strings.Builder, width int) {
	t.Helper()
	dst := make([]uint8, 20*3)
	src := make([]uint8, 20*3)
	for i := range dst {
		dst[i] = uint8((5 + i*11 + width*3) & 255)
		src[i] = uint8((250 - i*7 + width) & 255)
	}

	if err := h264BiweightPixels(dst, src, 20, 3, 2, 3, -2, -5, width); err != nil {
		t.Fatal(err)
	}
	fmt.Fprintf(b, "biweight%d", width)
	for y := 0; y < 3; y++ {
		for x := 0; x < width; x++ {
			fmt.Fprintf(b, " %d", dst[y*20+x])
		}
	}
	fmt.Fprint(b, "\n")
}

type h264LoopFilterTCFunc func([]uint8, int, int, int, int, *[4]int8) error
type h264LoopFilterIntraFunc func([]uint8, int, int, int, int) error

func printDSPLoopTCOracleWant(t *testing.T, b *strings.Builder, label string, fn h264LoopFilterTCFunc) {
	t.Helper()
	const stride = 32
	const offset = 12*stride + 12
	pix := h264LoopFilterOracleFixture(stride, 32)
	tc0 := [4]int8{2, 0, -1, 4}

	if err := fn(pix, offset, stride, 80, 80, &tc0); err != nil {
		t.Fatal(err)
	}
	printDSPLoopWindow(b, label, pix, stride, offset)
}

func printDSPLoopIntraOracleWant(t *testing.T, b *strings.Builder, label string, fn h264LoopFilterIntraFunc) {
	t.Helper()
	const stride = 32
	const offset = 12*stride + 12
	pix := h264LoopFilterOracleFixture(stride, 32)

	if err := fn(pix, offset, stride, 80, 80); err != nil {
		t.Fatal(err)
	}
	printDSPLoopWindow(b, label, pix, stride, offset)
}

func h264LoopFilterOracleFixture(stride int, rows int) []uint8 {
	pix := make([]uint8, stride*rows)
	for y := 0; y < rows; y++ {
		for x := 0; x < stride; x++ {
			pix[y*stride+x] = uint8(80 + (x*2+y*3)%64)
		}
	}
	return pix
}

func printDSPLoopWindow(b *strings.Builder, label string, pix []uint8, stride int, offset int) {
	fmt.Fprint(b, label)
	for y := -4; y < 12; y++ {
		for x := -4; x < 12; x++ {
			fmt.Fprintf(b, " %d", pix[offset+y*stride+x])
		}
	}
	fmt.Fprint(b, "\n")
}
