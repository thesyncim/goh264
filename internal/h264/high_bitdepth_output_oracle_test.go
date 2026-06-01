// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

import (
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestH264HighBitDepthRawOutputOracleFixtures(t *testing.T) {
	for _, tt := range h264HighOutputOracleCases() {
		t.Run(tt.name, func(t *testing.T) {
			pic := makeH264HighOutputOraclePicture(tt)
			raw, err := appendH264HighBitDepthRawYUVLE(nil, pic, tt.width, tt.height, tt.cropLeft, tt.cropTop, tt.bitDepth)
			if err != nil {
				t.Fatal(err)
			}
			if got, want := len(raw), len(tt.wantHex)/2; got != want {
				t.Fatalf("raw size = %d, want %d", got, want)
			}
			if got := hex.EncodeToString(raw); got != tt.wantHex {
				t.Fatalf("raw hex mismatch\n got %s\nwant %s", got, tt.wantHex)
			}
		})
	}
}

func TestH264HighBitDepthRawOutputPreservesIntraPCMSamples(t *testing.T) {
	tests := []struct {
		name            string
		chromaFormatIDC int
		bitDepth        int
		seed            int
		width           int
		height          int
		wantSamples     []uint16
	}{
		{
			name:            "420_10",
			chromaFormatIDC: 1,
			bitDepth:        10,
			seed:            33,
			width:           16,
			height:          16,
			wantSamples:     []uint16{33, 50, 67, 84, 101, 118, 135, 152, 170, 187, 204, 221},
		},
		{
			name:            "422_12",
			chromaFormatIDC: 2,
			bitDepth:        12,
			seed:            49,
			width:           16,
			height:          16,
			wantSamples:     []uint16{49, 66, 83, 100, 117, 134, 151, 168, 186, 203, 220, 237},
		},
		{
			name:            "444_14",
			chromaFormatIDC: 3,
			bitDepth:        14,
			seed:            57,
			width:           16,
			height:          16,
			wantSamples:     []uint16{57, 74, 91, 108, 125, 142, 159, 176, 194, 211, 228, 245},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pic := makeH264ReconstructHighPicture(tt.chromaFormatIDC, 0)
			yOff, cbOff, crOff, err := h264MBDestPartOffsetsHigh(pic, 1, 1, 0, 0)
			if err != nil {
				t.Fatal(err)
			}
			pcm := h264ReconstructIntraPCMHigh(tt.chromaFormatIDC, tt.bitDepth, tt.seed)
			if err := h264HLDecodeFrameIntraPCMHigh(pic, yOff, cbOff, crOff, pcm, tt.bitDepth); err != nil {
				t.Fatal(err)
			}

			croppedMB := &h264PicturePlanesHigh{
				Y:               pic.Y[yOff:],
				Cb:              pic.Cb[cbOff:],
				Cr:              pic.Cr[crOff:],
				LumaStride:      pic.LumaStride,
				ChromaStride:    pic.ChromaStride,
				MBWidth:         1,
				MBHeight:        1,
				ChromaFormatIDC: pic.ChromaFormatIDC,
			}
			raw, err := appendH264HighBitDepthRawYUVLE(nil, croppedMB, tt.width, tt.height, 0, 0, tt.bitDepth)
			if err != nil {
				t.Fatal(err)
			}
			gotSamples, err := h264HighOutputRawSamplesLE(raw[:len(tt.wantSamples)*2], tt.bitDepth)
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(gotSamples, tt.wantSamples) {
				t.Fatalf("luma samples = %v, want %v", gotSamples, tt.wantSamples)
			}
		})
	}
}

func TestH264HighBitDepthMBDestPartOffsetsMatchFFmpegLayout(t *testing.T) {
	tests := []struct {
		name            string
		chromaFormatIDC int
		wantY           int
		wantC           int
	}{
		{name: "mono", chromaFormatIDC: 0, wantY: 2118, wantC: 0},
		{name: "420", chromaFormatIDC: 1, wantY: 2118, wantC: 643},
		{name: "422", chromaFormatIDC: 2, wantY: 2118, wantC: 1267},
		{name: "444", chromaFormatIDC: 3, wantY: 2118, wantC: 1286},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pic := &h264PicturePlanesHigh{
				Y:               make([]uint16, 80*64),
				Cb:              make([]uint16, 48*64),
				Cr:              make([]uint16, 48*64),
				LumaStride:      80,
				ChromaStride:    48,
				MBWidth:         4,
				MBHeight:        4,
				ChromaFormatIDC: tt.chromaFormatIDC,
			}
			y, cb, cr, err := h264MBDestPartOffsetsHigh(pic, 2, 1, 3, 5)
			if err != nil {
				t.Fatal(err)
			}
			if y != tt.wantY || cb != tt.wantC || cr != tt.wantC {
				t.Fatalf("offsets = y:%d cb:%d cr:%d, want y:%d c:%d", y, cb, cr, tt.wantY, tt.wantC)
			}
		})
	}
}

func TestH264HighBitDepthRawOutputRejectsInvalidFixtures(t *testing.T) {
	valid := h264HighOutputOracleCases()[0]
	tests := []struct {
		name string
		edit func(*h264HighOutputOracleCase, *h264PicturePlanesHigh)
	}{
		{
			name: "unsupported_bit_depth",
			edit: func(tc *h264HighOutputOracleCase, _ *h264PicturePlanesHigh) {
				tc.bitDepth = 11
			},
		},
		{
			name: "sample_exceeds_bit_depth",
			edit: func(tc *h264HighOutputOracleCase, pic *h264PicturePlanesHigh) {
				pic.Y[tc.cropTop*pic.LumaStride+tc.cropLeft] = 1 << uint(tc.bitDepth)
			},
		},
		{
			name: "undersized_luma_stride",
			edit: func(_ *h264HighOutputOracleCase, pic *h264PicturePlanesHigh) {
				pic.LumaStride = pic.MBWidth*16 - 1
			},
		},
		{
			name: "negative_crop",
			edit: func(tc *h264HighOutputOracleCase, _ *h264PicturePlanesHigh) {
				tc.cropLeft = -1
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc := valid
			pic := makeH264HighOutputOraclePicture(tc)
			tt.edit(&tc, pic)
			if _, err := appendH264HighBitDepthRawYUVLE(nil, pic, tc.width, tc.height, tc.cropLeft, tc.cropTop, tc.bitDepth); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestH264HighBitDepthRawOutputCOracle(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run high-bit-depth raw output C oracle")
	}
	cc, err := exec.LookPath("cc")
	if err != nil {
		t.Skip("cc not available")
	}

	dir := t.TempDir()
	writeOracleFile(t, filepath.Join(dir, "high_output_oracle.c"), h264HighOutputOracleC)
	bin := filepath.Join(dir, "high_output_oracle")
	cmd := exec.Command(cc, "-std=c99", filepath.Join(dir, "high_output_oracle.c"), "-o", bin)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("compile high output oracle: %v\n%s", err, out)
	}
	out, err := exec.Command(bin).CombinedOutput()
	if err != nil {
		t.Fatalf("run high output oracle: %v\n%s", err, out)
	}
	got := strings.TrimSpace(string(out))
	want := strings.TrimSpace(h264HighOutputOracleWant(t))
	if got != want {
		t.Fatalf("high-bit-depth raw output oracle mismatch\nC oracle:\n%s\nGo:\n%s", got, want)
	}
}

type h264HighOutputOracleCase struct {
	name            string
	chromaFormatIDC int
	bitDepth        int
	mbWidth         int
	mbHeight        int
	width           int
	height          int
	cropLeft        int
	cropTop         int
	lumaStride      int
	chromaStride    int
	seed            int
	wantHex         string
}

func h264HighOutputOracleCases() []h264HighOutputOracleCase {
	return []h264HighOutputOracleCase{
		{
			name:            "yuv420p10le_odd_crop",
			chromaFormatIDC: 1,
			bitDepth:        10,
			mbWidth:         2,
			mbHeight:        2,
			width:           5,
			height:          3,
			cropLeft:        3,
			cropTop:         5,
			lumaStride:      40,
			chromaStride:    24,
			seed:            19,
			wantHex:         "a202b802de02f7021d03060325033703560385036a0392039d03c503ed03cd03e80313002f00530077008a02a502d002ec0210033403",
		},
		{
			name:            "yuv422p12le_vertical_crop",
			chromaFormatIDC: 2,
			bitDepth:        12,
			mbWidth:         2,
			mbHeight:        1,
			width:           6,
			height:          4,
			cropLeft:        2,
			cropTop:         3,
			lumaStride:      36,
			chromaStride:    20,
			seed:            37,
			wantHex:         "a801cc011002340258027c02230250025d028a02b702e4028e02b402ca02f00209032f03e90218033703490368039703410465048904b304e0040d0515054b0571058705a605d505fe062207460770079d07ca07d20708082e08440863089208",
		},
		{
			name:            "yuv444p14le_full_chroma_crop",
			chromaFormatIDC: 3,
			bitDepth:        14,
			mbWidth:         1,
			mbHeight:        1,
			width:           3,
			height:          2,
			cropLeft:        4,
			cropTop:         2,
			lumaStride:      20,
			chromaStride:    22,
			seed:            53,
			wantHex:         "b301de01f90120024402680270049b04b604dd04010525052d07580773079a07be07e207",
		},
		{
			name:            "yuv400p9le_luma_only_crop",
			chromaFormatIDC: 0,
			bitDepth:        9,
			mbWidth:         1,
			mbHeight:        1,
			width:           4,
			height:          3,
			cropLeft:        1,
			cropTop:         2,
			lumaStride:      18,
			seed:            71,
			wantHex:         "44015f018a01c501a601ca01ee0132001800450072007f00",
		},
	}
}

func makeH264HighOutputOraclePicture(tc h264HighOutputOracleCase) *h264PicturePlanesHigh {
	pic := &h264PicturePlanesHigh{
		Y:               make([]uint16, tc.lumaStride*tc.mbHeight*16),
		LumaStride:      tc.lumaStride,
		MBWidth:         tc.mbWidth,
		MBHeight:        tc.mbHeight,
		ChromaFormatIDC: tc.chromaFormatIDC,
	}
	for y := 0; y < tc.mbHeight*16; y++ {
		for x := 0; x < tc.lumaStride; x++ {
			pic.Y[y*tc.lumaStride+x] = h264HighOutputOracleSample(tc.seed, 0, x, y, tc.bitDepth)
		}
	}
	if tc.chromaFormatIDC == 0 {
		return pic
	}
	_, chromaHeight := h264ChromaFrameSize(tc.mbWidth, tc.mbHeight, tc.chromaFormatIDC)
	pic.ChromaStride = tc.chromaStride
	pic.Cb = make([]uint16, tc.chromaStride*chromaHeight)
	pic.Cr = make([]uint16, tc.chromaStride*chromaHeight)
	for y := 0; y < chromaHeight; y++ {
		for x := 0; x < tc.chromaStride; x++ {
			pic.Cb[y*tc.chromaStride+x] = h264HighOutputOracleSample(tc.seed, 1, x, y, tc.bitDepth)
			pic.Cr[y*tc.chromaStride+x] = h264HighOutputOracleSample(tc.seed, 2, x, y, tc.bitDepth)
		}
	}
	return pic
}

func appendH264HighBitDepthRawYUVLE(dst []byte, pic *h264PicturePlanesHigh, width int, height int, cropLeft int, cropTop int, bitDepth int) ([]byte, error) {
	if err := checkH264DSPHighBitDepth(bitDepth); err != nil {
		return dst, err
	}
	if pic == nil || width <= 0 || height <= 0 || cropLeft < 0 || cropTop < 0 {
		return dst, ErrInvalidData
	}
	if err := pic.validate(); err != nil {
		return dst, err
	}
	if cropLeft+width > pic.MBWidth*16 || cropTop+height > pic.MBHeight*16 {
		return dst, ErrInvalidData
	}
	var err error
	dst, err = appendH264HighBitDepthRawPlaneLE(dst, pic.Y, pic.LumaStride, width, height, cropLeft, cropTop, bitDepth)
	if err != nil {
		return dst, err
	}

	chromaWidth, chromaHeight, err := h264HighOutputChromaDisplaySize(width, height, pic.ChromaFormatIDC)
	if err != nil {
		return dst, err
	}
	if chromaWidth == 0 || chromaHeight == 0 {
		return dst, nil
	}
	chromaCropLeft, chromaCropTop, err := h264HighOutputChromaCrop(cropLeft, cropTop, pic.ChromaFormatIDC)
	if err != nil {
		return dst, err
	}
	if chromaCropLeft+chromaWidth > h264HighOutputChromaStoredWidth(pic) ||
		chromaCropTop+chromaHeight > h264HighOutputChromaStoredHeight(pic) {
		return dst, ErrInvalidData
	}
	dst, err = appendH264HighBitDepthRawPlaneLE(dst, pic.Cb, pic.ChromaStride, chromaWidth, chromaHeight, chromaCropLeft, chromaCropTop, bitDepth)
	if err != nil {
		return dst, err
	}
	return appendH264HighBitDepthRawPlaneLE(dst, pic.Cr, pic.ChromaStride, chromaWidth, chromaHeight, chromaCropLeft, chromaCropTop, bitDepth)
}

func appendH264HighBitDepthRawPlaneLE(dst []byte, plane []uint16, stride int, width int, height int, cropLeft int, cropTop int, bitDepth int) ([]byte, error) {
	if stride <= 0 || width <= 0 || height <= 0 || cropLeft < 0 || cropTop < 0 {
		return dst, ErrInvalidData
	}
	if len(plane) < (cropTop+height-1)*stride+cropLeft+width {
		return dst, ErrInvalidData
	}
	maxSample := uint16((1 << uint(bitDepth)) - 1)
	for y := 0; y < height; y++ {
		row := (cropTop+y)*stride + cropLeft
		for x := 0; x < width; x++ {
			v := plane[row+x]
			if v > maxSample {
				return dst, ErrInvalidData
			}
			dst = append(dst, byte(v), byte(v>>8))
		}
	}
	return dst, nil
}

func h264HighOutputRawSamplesLE(raw []byte, bitDepth int) ([]uint16, error) {
	if len(raw)%2 != 0 {
		return nil, ErrInvalidData
	}
	maxSample := uint16((1 << uint(bitDepth)) - 1)
	out := make([]uint16, len(raw)/2)
	for i := range out {
		v := uint16(raw[2*i]) | uint16(raw[2*i+1])<<8
		if v > maxSample {
			return nil, ErrInvalidData
		}
		out[i] = v
	}
	return out, nil
}

func h264HighOutputChromaDisplaySize(width int, height int, chromaFormatIDC int) (int, int, error) {
	switch chromaFormatIDC {
	case 0:
		return 0, 0, nil
	case 1:
		return (width + 1) >> 1, (height + 1) >> 1, nil
	case 2:
		return (width + 1) >> 1, height, nil
	case 3:
		return width, height, nil
	default:
		return 0, 0, ErrInvalidData
	}
}

func h264HighOutputChromaCrop(cropLeft int, cropTop int, chromaFormatIDC int) (int, int, error) {
	if cropLeft < 0 || cropTop < 0 {
		return 0, 0, ErrInvalidData
	}
	switch chromaFormatIDC {
	case 0, 3:
		return cropLeft, cropTop, nil
	case 1:
		return cropLeft >> 1, cropTop >> 1, nil
	case 2:
		return cropLeft >> 1, cropTop, nil
	default:
		return 0, 0, ErrInvalidData
	}
}

func h264HighOutputChromaStoredWidth(pic *h264PicturePlanesHigh) int {
	switch pic.ChromaFormatIDC {
	case 1, 2:
		return pic.MBWidth * 8
	case 3:
		return pic.MBWidth * 16
	default:
		return 0
	}
}

func h264HighOutputChromaStoredHeight(pic *h264PicturePlanesHigh) int {
	switch pic.ChromaFormatIDC {
	case 1:
		return pic.MBHeight * 8
	case 2, 3:
		return pic.MBHeight * 16
	default:
		return 0
	}
}

func h264HighOutputOracleSample(seed int, plane int, x int, y int, bitDepth int) uint16 {
	max := (1 << uint(bitDepth)) - 1
	return uint16((seed + plane*701 + x*37 + y*101 + ((x ^ y) << 2) + (x*y)%29) & max)
}

func h264HighOutputOracleWant(t *testing.T) string {
	var b strings.Builder
	for _, tc := range h264HighOutputOracleCases() {
		raw, err := appendH264HighBitDepthRawYUVLE(nil, makeH264HighOutputOraclePicture(tc), tc.width, tc.height, tc.cropLeft, tc.cropTop, tc.bitDepth)
		if err != nil {
			t.Fatal(err)
		}
		fmt.Fprintf(&b, "%s %s\n", tc.name, hex.EncodeToString(raw))
	}
	return b.String()
}

const h264HighOutputOracleC = `
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>

typedef struct Fixture {
    const char *name;
    int chroma_idc;
    int bit_depth;
    int mb_width;
    int mb_height;
    int width;
    int height;
    int crop_left;
    int crop_top;
    int luma_stride;
    int chroma_stride;
    int seed;
} Fixture;

typedef struct PictureHigh {
    uint16_t *y;
    uint16_t *cb;
    uint16_t *cr;
} PictureHigh;

static uint16_t oracle_sample(int seed, int plane, int x, int y, int bit_depth)
{
    const int max = (1 << bit_depth) - 1;
    return (uint16_t)((seed + plane * 701 + x * 37 + y * 101 +
                       ((x ^ y) << 2) + (x * y) % 29) & max);
}

static void chroma_frame_size(int mb_width, int mb_height, int chroma_idc,
                              int *width, int *height)
{
    if (chroma_idc == 1) {
        *width = mb_width * 8;
        *height = mb_height * 8;
    } else if (chroma_idc == 2) {
        *width = mb_width * 8;
        *height = mb_height * 16;
    } else if (chroma_idc == 3) {
        *width = mb_width * 16;
        *height = mb_height * 16;
    } else {
        *width = 0;
        *height = 0;
    }
}

static void chroma_display_size(int width, int height, int chroma_idc,
                                int *chroma_width, int *chroma_height)
{
    if (chroma_idc == 1) {
        *chroma_width = (width + 1) >> 1;
        *chroma_height = (height + 1) >> 1;
    } else if (chroma_idc == 2) {
        *chroma_width = (width + 1) >> 1;
        *chroma_height = height;
    } else if (chroma_idc == 3) {
        *chroma_width = width;
        *chroma_height = height;
    } else {
        *chroma_width = 0;
        *chroma_height = 0;
    }
}

static void chroma_crop(int crop_left, int crop_top, int chroma_idc,
                        int *chroma_crop_left, int *chroma_crop_top)
{
    if (chroma_idc == 1) {
        *chroma_crop_left = crop_left >> 1;
        *chroma_crop_top = crop_top >> 1;
    } else if (chroma_idc == 2) {
        *chroma_crop_left = crop_left >> 1;
        *chroma_crop_top = crop_top;
    } else {
        *chroma_crop_left = crop_left;
        *chroma_crop_top = crop_top;
    }
}

static PictureHigh make_picture(Fixture f)
{
    PictureHigh p = {0};
    const int luma_height = f.mb_height * 16;
    p.y = (uint16_t *)calloc((size_t)f.luma_stride * luma_height, sizeof(uint16_t));
    for (int y = 0; y < luma_height; y++) {
        for (int x = 0; x < f.luma_stride; x++)
            p.y[y * f.luma_stride + x] = oracle_sample(f.seed, 0, x, y, f.bit_depth);
    }
    if (!f.chroma_idc)
        return p;

    int chroma_width, chroma_height;
    chroma_frame_size(f.mb_width, f.mb_height, f.chroma_idc, &chroma_width, &chroma_height);
    (void)chroma_width;
    p.cb = (uint16_t *)calloc((size_t)f.chroma_stride * chroma_height, sizeof(uint16_t));
    p.cr = (uint16_t *)calloc((size_t)f.chroma_stride * chroma_height, sizeof(uint16_t));
    for (int y = 0; y < chroma_height; y++) {
        for (int x = 0; x < f.chroma_stride; x++) {
            p.cb[y * f.chroma_stride + x] = oracle_sample(f.seed, 1, x, y, f.bit_depth);
            p.cr[y * f.chroma_stride + x] = oracle_sample(f.seed, 2, x, y, f.bit_depth);
        }
    }
    return p;
}

static void print_plane_hex(const uint16_t *plane, int stride,
                            int crop_left, int crop_top, int width, int height)
{
    for (int y = 0; y < height; y++) {
        const uint16_t *row = plane + (crop_top + y) * stride + crop_left;
        for (int x = 0; x < width; x++)
            printf("%02x%02x", row[x] & 0xff, row[x] >> 8);
    }
}

static void run_fixture(Fixture f)
{
    PictureHigh p = make_picture(f);
    int chroma_width, chroma_height, chroma_crop_left, chroma_crop_top;
    printf("%s ", f.name);
    print_plane_hex(p.y, f.luma_stride, f.crop_left, f.crop_top, f.width, f.height);
    chroma_display_size(f.width, f.height, f.chroma_idc, &chroma_width, &chroma_height);
    if (chroma_width && chroma_height) {
        chroma_crop(f.crop_left, f.crop_top, f.chroma_idc, &chroma_crop_left, &chroma_crop_top);
        print_plane_hex(p.cb, f.chroma_stride, chroma_crop_left, chroma_crop_top,
                        chroma_width, chroma_height);
        print_plane_hex(p.cr, f.chroma_stride, chroma_crop_left, chroma_crop_top,
                        chroma_width, chroma_height);
    }
    printf("\n");
    free(p.y);
    free(p.cb);
    free(p.cr);
}

int main(void)
{
    const Fixture fixtures[] = {
        {"yuv420p10le_odd_crop", 1, 10, 2, 2, 5, 3, 3, 5, 40, 24, 19},
        {"yuv422p12le_vertical_crop", 2, 12, 2, 1, 6, 4, 2, 3, 36, 20, 37},
        {"yuv444p14le_full_chroma_crop", 3, 14, 1, 1, 3, 2, 4, 2, 20, 22, 53},
        {"yuv400p9le_luma_only_crop", 0, 9, 1, 1, 4, 3, 1, 2, 18, 0, 71},
    };
    for (size_t i = 0; i < sizeof(fixtures) / sizeof(fixtures[0]); i++)
        run_fixture(fixtures[i]);
    return 0;
}
`
