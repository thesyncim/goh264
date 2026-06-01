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

const chromaOracleC = `
#include <stddef.h>
#include <stdint.h>
#include <stdio.h>

#define BIT_DEPTH 8
#include "h264chroma_template.c"
#undef BIT_DEPTH
#define BIT_DEPTH 9
#include "h264chroma_template.c"
#undef BIT_DEPTH
#define BIT_DEPTH 10
#include "h264chroma_template.c"
#undef BIT_DEPTH
#define BIT_DEPTH 12
#include "h264chroma_template.c"
#undef BIT_DEPTH
#define BIT_DEPTH 14
#include "h264chroma_template.c"
#undef BIT_DEPTH

typedef void (*chroma_fn)(uint8_t *dst, const uint8_t *src, ptrdiff_t stride,
                          int h, int x, int y);

static chroma_fn put_chroma_fn(int width)
{
    switch (width) {
    case 1:
        return put_h264_chroma_mc1_8_c;
    case 2:
        return put_h264_chroma_mc2_8_c;
    case 4:
        return put_h264_chroma_mc4_8_c;
    case 8:
        return put_h264_chroma_mc8_8_c;
    }
    return 0;
}

static chroma_fn avg_chroma_fn(int width)
{
    switch (width) {
    case 1:
        return avg_h264_chroma_mc1_8_c;
    case 2:
        return avg_h264_chroma_mc2_8_c;
    case 4:
        return avg_h264_chroma_mc4_8_c;
    case 8:
        return avg_h264_chroma_mc8_8_c;
    }
    return 0;
}

static chroma_fn put_chroma_fn_high(int width, int bit_depth)
{
    switch (bit_depth) {
    case 9:
        switch (width) {
        case 1:
            return put_h264_chroma_mc1_9_c;
        case 2:
            return put_h264_chroma_mc2_9_c;
        case 4:
            return put_h264_chroma_mc4_9_c;
        case 8:
            return put_h264_chroma_mc8_9_c;
        }
        break;
    case 10:
        switch (width) {
        case 1:
            return put_h264_chroma_mc1_10_c;
        case 2:
            return put_h264_chroma_mc2_10_c;
        case 4:
            return put_h264_chroma_mc4_10_c;
        case 8:
            return put_h264_chroma_mc8_10_c;
        }
        break;
    case 12:
        switch (width) {
        case 1:
            return put_h264_chroma_mc1_12_c;
        case 2:
            return put_h264_chroma_mc2_12_c;
        case 4:
            return put_h264_chroma_mc4_12_c;
        case 8:
            return put_h264_chroma_mc8_12_c;
        }
        break;
    case 14:
        switch (width) {
        case 1:
            return put_h264_chroma_mc1_14_c;
        case 2:
            return put_h264_chroma_mc2_14_c;
        case 4:
            return put_h264_chroma_mc4_14_c;
        case 8:
            return put_h264_chroma_mc8_14_c;
        }
        break;
    }
    return 0;
}

static chroma_fn avg_chroma_fn_high(int width, int bit_depth)
{
    switch (bit_depth) {
    case 9:
        switch (width) {
        case 1:
            return avg_h264_chroma_mc1_9_c;
        case 2:
            return avg_h264_chroma_mc2_9_c;
        case 4:
            return avg_h264_chroma_mc4_9_c;
        case 8:
            return avg_h264_chroma_mc8_9_c;
        }
        break;
    case 10:
        switch (width) {
        case 1:
            return avg_h264_chroma_mc1_10_c;
        case 2:
            return avg_h264_chroma_mc2_10_c;
        case 4:
            return avg_h264_chroma_mc4_10_c;
        case 8:
            return avg_h264_chroma_mc8_10_c;
        }
        break;
    case 12:
        switch (width) {
        case 1:
            return avg_h264_chroma_mc1_12_c;
        case 2:
            return avg_h264_chroma_mc2_12_c;
        case 4:
            return avg_h264_chroma_mc4_12_c;
        case 8:
            return avg_h264_chroma_mc8_12_c;
        }
        break;
    case 14:
        switch (width) {
        case 1:
            return avg_h264_chroma_mc1_14_c;
        case 2:
            return avg_h264_chroma_mc2_14_c;
        case 4:
            return avg_h264_chroma_mc4_14_c;
        case 8:
            return avg_h264_chroma_mc8_14_c;
        }
        break;
    }
    return 0;
}

static void init_chroma_fixture(uint8_t *dst, uint8_t *src, int n)
{
    for (int i = 0; i < n; i++) {
        dst[i] = (20 + i * 11) & 255;
        src[i] = (10 + i * 9) & 255;
    }
}

static void print_chroma_case(const char *label, chroma_fn fn, int width,
                              int h, int x, int y)
{
    const int stride = 24;
    const int offset = 4 * stride + 5;
    uint8_t dst[24 * 17];
    uint8_t src[24 * 17];

    init_chroma_fixture(dst, src, 24 * 17);
    fn(dst + offset, src + offset, stride, h, x, y);

    printf("%s", label);
    for (int row = 0; row < h; row++)
        for (int col = 0; col < width; col++)
            printf(" %u", dst[offset + row * stride + col]);
    printf("\n");
}

static void init_chroma_fixture_high(uint16_t *dst, uint16_t *src, int n, int bit_depth)
{
    int max = (1 << bit_depth) - 1;
    for (int i = 0; i < n; i++) {
        dst[i] = (uint16_t)((20 + i * 37) & max);
        src[i] = (uint16_t)((10 + i * 29) & max);
    }
}

static void print_chroma_case_high(const char *label, chroma_fn fn, int width,
                                   int h, int x, int y, int bit_depth)
{
    const int stride = 24;
    const int offset = 4 * stride + 5;
    uint16_t dst[24 * 17];
    uint16_t src[24 * 17];

    init_chroma_fixture_high(dst, src, 24 * 17, bit_depth);
    fn((uint8_t *)(dst + offset), (const uint8_t *)(src + offset),
       stride * (ptrdiff_t)sizeof(uint16_t), h, x, y);

    printf("%s", label);
    for (int row = 0; row < h; row++)
        for (int col = 0; col < width; col++)
            printf(" %u", dst[offset + row * stride + col]);
    printf("\n");
}

static void print_chroma_suite(const char *prefix, int avg)
{
    const int widths[4] = { 1, 2, 4, 8 };
    const int xy[4][2] = { { 0, 0 }, { 3, 0 }, { 0, 5 }, { 3, 5 } };
    char label[64];

    for (int w = 0; w < 4; w++) {
        for (int c = 0; c < 4; c++) {
            snprintf(label, sizeof(label), "%s%d_%d_%d",
                     prefix, widths[w], xy[c][0], xy[c][1]);
            print_chroma_case(label,
                              avg ? avg_chroma_fn(widths[w])
                                  : put_chroma_fn(widths[w]),
                              widths[w], 5, xy[c][0], xy[c][1]);
        }
    }
}

static void print_chroma_suite_high(const char *prefix, int avg, int bit_depth)
{
    const int widths[4] = { 1, 2, 4, 8 };
    const int xy[4][2] = { { 0, 0 }, { 3, 0 }, { 0, 5 }, { 3, 5 } };
    char label[80];

    for (int w = 0; w < 4; w++) {
        for (int c = 0; c < 4; c++) {
            snprintf(label, sizeof(label), "%s%d_%d_%d_%d",
                     prefix, bit_depth, widths[w], xy[c][0], xy[c][1]);
            print_chroma_case_high(label,
                                   avg ? avg_chroma_fn_high(widths[w], bit_depth)
                                       : put_chroma_fn_high(widths[w], bit_depth),
                                   widths[w], 5, xy[c][0], xy[c][1], bit_depth);
        }
    }
}

int main(void)
{
    const int depths[4] = { 9, 10, 12, 14 };

    print_chroma_suite("putmc", 0);
    print_chroma_suite("avgmc", 1);
    for (int i = 0; i < 4; i++) {
        print_chroma_suite_high("putmc", 0, depths[i]);
        print_chroma_suite_high("avgmc", 1, depths[i]);
    }
    return 0;
}
`

const chromaOracleBitDepthTemplate = `
#include <stdint.h>

#undef pixel
#undef FUNC3
#undef FUNC2
#undef FUNC
#undef FUNCC

#if BIT_DEPTH > 8
#define pixel uint16_t
#else
#define pixel uint8_t
#endif
#define FUNC3(a, b, c)  a ## _ ## b ## c
#define FUNC2(a, b, c)  FUNC3(a, b, c)
#define FUNC(a) FUNC2(a, BIT_DEPTH, _c)
#define FUNCC(a) FUNC2(a, BIT_DEPTH, _c)
`

func TestH264ChromaMCUpstreamOracle(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run pinned FFmpeg H.264 chroma MC oracle")
	}
	cc, err := exec.LookPath("cc")
	if err != nil {
		t.Skip("cc not available")
	}

	root := h264RepoRoot(t)
	upstreamTemplate := filepath.Join(root, ".upstream", "ffmpeg-n8.0.1", "libavcodec", "h264chroma_template.c")
	template, err := os.ReadFile(upstreamTemplate)
	if err != nil {
		t.Skipf("pinned upstream H.264 chroma MC source not available: %v", err)
	}

	dir := t.TempDir()
	writeOracleFile(t, filepath.Join(dir, "oracle.c"), chromaOracleC)
	writeOracleFile(t, filepath.Join(dir, "h264chroma_template.c"), string(template))
	writeOracleFile(t, filepath.Join(dir, "bit_depth_template.c"), chromaOracleBitDepthTemplate)
	if err := os.Mkdir(filepath.Join(dir, "libavutil"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeOracleFile(t, filepath.Join(dir, "libavutil", "avassert.h"), "#define av_assert2(cond) do { } while (0)\n")

	bin := filepath.Join(dir, "oracle")
	cmd := exec.Command(cc, "-std=c99", "-I"+dir, filepath.Join(dir, "oracle.c"), "-o", bin)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("compile H.264 chroma MC oracle: %v\n%s", err, out)
	}

	out, err := exec.Command(bin).CombinedOutput()
	if err != nil {
		t.Fatalf("run H.264 chroma MC oracle: %v\n%s", err, out)
	}

	got := strings.TrimSpace(string(out))
	want := strings.TrimSpace(h264ChromaMCOracleWant(t))
	if got != want {
		t.Fatalf("H.264 chroma MC oracle mismatch\nC oracle:\n%s\nGo:\n%s", got, want)
	}
}

func h264ChromaMCOracleWant(t *testing.T) string {
	t.Helper()
	var b strings.Builder
	for _, avg := range []bool{false, true} {
		for _, width := range []int{1, 2, 4, 8} {
			for _, xy := range [][2]int{{0, 0}, {3, 0}, {0, 5}, {3, 5}} {
				label := "putmc"
				fn := h264ChromaPutFunc(t, width)
				if avg {
					label = "avgmc"
					fn = h264ChromaAvgFunc(t, width)
				}
				printChromaMCOracleWant(t, &b, fmt.Sprintf("%s%d_%d_%d", label, width, xy[0], xy[1]), fn, width, 5, xy[0], xy[1])
			}
		}
	}
	for _, bitDepth := range []int{9, 10, 12, 14} {
		for _, avg := range []bool{false, true} {
			for _, width := range []int{1, 2, 4, 8} {
				for _, xy := range [][2]int{{0, 0}, {3, 0}, {0, 5}, {3, 5}} {
					label := "putmc"
					fn := h264ChromaPutFuncHigh(t, width)
					if avg {
						label = "avgmc"
						fn = h264ChromaAvgFuncHigh(t, width)
					}
					printChromaMCOracleWantHigh(t, &b, fmt.Sprintf("%s%d_%d_%d_%d", label, bitDepth, width, xy[0], xy[1]), fn, width, 5, xy[0], xy[1], bitDepth)
				}
			}
		}
	}
	return b.String()
}

type h264ChromaMCFunc func([]uint8, []uint8, int, int, int, int) error
type h264ChromaMCFuncHigh func([]uint16, []uint16, int, int, int, int, int) error

func h264ChromaPutFunc(t *testing.T, width int) h264ChromaMCFunc {
	t.Helper()
	switch width {
	case 1:
		return h264PutH264ChromaMC1
	case 2:
		return h264PutH264ChromaMC2
	case 4:
		return h264PutH264ChromaMC4
	case 8:
		return h264PutH264ChromaMC8
	default:
		t.Fatalf("unsupported chroma MC width %d", width)
		return nil
	}
}

func h264ChromaAvgFunc(t *testing.T, width int) h264ChromaMCFunc {
	t.Helper()
	switch width {
	case 1:
		return h264AvgH264ChromaMC1
	case 2:
		return h264AvgH264ChromaMC2
	case 4:
		return h264AvgH264ChromaMC4
	case 8:
		return h264AvgH264ChromaMC8
	default:
		t.Fatalf("unsupported chroma MC width %d", width)
		return nil
	}
}

func h264ChromaPutFuncHigh(t *testing.T, width int) h264ChromaMCFuncHigh {
	t.Helper()
	switch width {
	case 1:
		return h264PutH264ChromaMC1High
	case 2:
		return h264PutH264ChromaMC2High
	case 4:
		return h264PutH264ChromaMC4High
	case 8:
		return h264PutH264ChromaMC8High
	default:
		t.Fatalf("unsupported high chroma MC width %d", width)
		return nil
	}
}

func h264ChromaAvgFuncHigh(t *testing.T, width int) h264ChromaMCFuncHigh {
	t.Helper()
	switch width {
	case 1:
		return h264AvgH264ChromaMC1High
	case 2:
		return h264AvgH264ChromaMC2High
	case 4:
		return h264AvgH264ChromaMC4High
	case 8:
		return h264AvgH264ChromaMC8High
	default:
		t.Fatalf("unsupported high chroma MC width %d", width)
		return nil
	}
}

func printChromaMCOracleWant(t *testing.T, b *strings.Builder, label string, fn h264ChromaMCFunc, width int, height int, x int, y int) {
	t.Helper()
	const stride = 24
	const offset = 4*stride + 5
	dst, src := makeChromaMCOracleFixture(stride, 17)

	if err := fn(dst[offset:], src[offset:], stride, height, x, y); err != nil {
		t.Fatal(err)
	}
	fmt.Fprint(b, label)
	for row := 0; row < height; row++ {
		for col := 0; col < width; col++ {
			fmt.Fprintf(b, " %d", dst[offset+row*stride+col])
		}
	}
	fmt.Fprint(b, "\n")
}

func printChromaMCOracleWantHigh(t *testing.T, b *strings.Builder, label string, fn h264ChromaMCFuncHigh, width int, height int, x int, y int, bitDepth int) {
	t.Helper()
	const stride = 24
	const offset = 4*stride + 5
	dst, src := makeChromaMCOracleFixtureHigh(stride, 17, bitDepth)

	if err := fn(dst[offset:], src[offset:], stride, height, x, y, bitDepth); err != nil {
		t.Fatal(err)
	}
	fmt.Fprint(b, label)
	for row := 0; row < height; row++ {
		for col := 0; col < width; col++ {
			fmt.Fprintf(b, " %d", dst[offset+row*stride+col])
		}
	}
	fmt.Fprint(b, "\n")
}

func makeChromaMCOracleFixture(stride int, rows int) ([]uint8, []uint8) {
	dst := make([]uint8, stride*rows)
	src := make([]uint8, stride*rows)
	for i := range dst {
		dst[i] = uint8((20 + i*11) & 255)
		src[i] = uint8((10 + i*9) & 255)
	}
	return dst, src
}

func makeChromaMCOracleFixtureHigh(stride int, rows int, bitDepth int) ([]uint16, []uint16) {
	dst := make([]uint16, stride*rows)
	src := make([]uint16, stride*rows)
	max := (1 << uint(bitDepth)) - 1
	for i := range dst {
		dst[i] = uint16((20 + i*37) & max)
		src[i] = uint16((10 + i*29) & max)
	}
	return dst, src
}
