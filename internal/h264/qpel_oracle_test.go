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

const qpelOracleC = `
#include <stddef.h>
#include <stdint.h>
#include <stdio.h>

#define av_unused
#define pixeltmp int16_t
#define BIT_DEPTH 8
#include "h264qpel_template.c"
#undef BIT_DEPTH
#define BIT_DEPTH 9
#include "h264qpel_template.c"
#undef BIT_DEPTH
#define BIT_DEPTH 10
#include "h264qpel_template.c"
#undef BIT_DEPTH
#undef pixeltmp
#define pixeltmp int32_t
#define BIT_DEPTH 12
#include "h264qpel_template.c"
#undef BIT_DEPTH
#define BIT_DEPTH 14
#include "h264qpel_template.c"
#undef BIT_DEPTH
#undef pixeltmp

typedef void (*qpel_fn)(uint8_t *dst, const uint8_t *src, ptrdiff_t stride);

#define QPEL_FN(prefix, size, suffix, depth) prefix ## size ## _mc ## suffix ## _ ## depth ## _c
#define QPEL_LIST(prefix, size, depth) { \
    QPEL_FN(prefix, size, 00, depth), QPEL_FN(prefix, size, 10, depth), \
    QPEL_FN(prefix, size, 20, depth), QPEL_FN(prefix, size, 30, depth), \
    QPEL_FN(prefix, size, 01, depth), QPEL_FN(prefix, size, 11, depth), \
    QPEL_FN(prefix, size, 21, depth), QPEL_FN(prefix, size, 31, depth), \
    QPEL_FN(prefix, size, 02, depth), QPEL_FN(prefix, size, 12, depth), \
    QPEL_FN(prefix, size, 22, depth), QPEL_FN(prefix, size, 32, depth), \
    QPEL_FN(prefix, size, 03, depth), QPEL_FN(prefix, size, 13, depth), \
    QPEL_FN(prefix, size, 23, depth), QPEL_FN(prefix, size, 33, depth) \
}

#define DECL_QPEL_TABLES(depth) \
static qpel_fn put2_ ## depth[16] = QPEL_LIST(put_h264_qpel, 2, depth); \
static qpel_fn put4_ ## depth[16] = QPEL_LIST(put_h264_qpel, 4, depth); \
static qpel_fn put8_ ## depth[16] = QPEL_LIST(put_h264_qpel, 8, depth); \
static qpel_fn put16_ ## depth[16] = QPEL_LIST(put_h264_qpel, 16, depth); \
static qpel_fn avg4_ ## depth[16] = QPEL_LIST(avg_h264_qpel, 4, depth); \
static qpel_fn avg8_ ## depth[16] = QPEL_LIST(avg_h264_qpel, 8, depth); \
static qpel_fn avg16_ ## depth[16] = QPEL_LIST(avg_h264_qpel, 16, depth)

DECL_QPEL_TABLES(8);
DECL_QPEL_TABLES(9);
DECL_QPEL_TABLES(10);
DECL_QPEL_TABLES(12);
DECL_QPEL_TABLES(14);

static qpel_fn qpel_put_fn(int size, int idx)
{
    switch (size) {
    case 2:
        return put2_8[idx];
    case 4:
        return put4_8[idx];
    case 8:
        return put8_8[idx];
    case 16:
        return put16_8[idx];
    }
    return 0;
}

static qpel_fn qpel_avg_fn(int size, int idx)
{
    switch (size) {
    case 4:
        return avg4_8[idx];
    case 8:
        return avg8_8[idx];
    case 16:
        return avg16_8[idx];
    }
    return 0;
}

static qpel_fn qpel_put_fn_high(int size, int idx, int bit_depth)
{
#define RETURN_PUT(depth) \
    do { \
        switch (size) { \
        case 2: return put2_ ## depth[idx]; \
        case 4: return put4_ ## depth[idx]; \
        case 8: return put8_ ## depth[idx]; \
        case 16: return put16_ ## depth[idx]; \
        } \
    } while (0)
    switch (bit_depth) {
    case 9:
        RETURN_PUT(9);
        break;
    case 10:
        RETURN_PUT(10);
        break;
    case 12:
        RETURN_PUT(12);
        break;
    case 14:
        RETURN_PUT(14);
        break;
    }
#undef RETURN_PUT
    return 0;
}

static qpel_fn qpel_avg_fn_high(int size, int idx, int bit_depth)
{
#define RETURN_AVG(depth) \
    do { \
        switch (size) { \
        case 4: return avg4_ ## depth[idx]; \
        case 8: return avg8_ ## depth[idx]; \
        case 16: return avg16_ ## depth[idx]; \
        } \
    } while (0)
    switch (bit_depth) {
    case 9:
        RETURN_AVG(9);
        break;
    case 10:
        RETURN_AVG(10);
        break;
    case 12:
        RETURN_AVG(12);
        break;
    case 14:
        RETURN_AVG(14);
        break;
    }
#undef RETURN_AVG
    return 0;
}

static void init_qpel_fixture(uint8_t *dst, uint8_t *src, int n)
{
    for (int i = 0; i < n; i++) {
        dst[i] = (20 + i * 11) & 255;
        src[i] = (10 + i * 9) & 255;
    }
}

static void print_qpel_case(const char *label, qpel_fn fn, int size)
{
    const int stride = 48;
    const int offset = 6 * stride + 6;
    uint8_t dst[48 * 48];
    uint8_t src[48 * 48];

    init_qpel_fixture(dst, src, 48 * 48);
    fn(dst + offset, src + offset, stride);

    printf("%s", label);
    for (int y = 0; y < size; y++)
        for (int x = 0; x < size; x++)
            printf(" %u", dst[offset + y * stride + x]);
    printf("\n");
}

static void init_qpel_fixture_high(uint16_t *dst, uint16_t *src, int n, int bit_depth)
{
    int max = (1 << bit_depth) - 1;
    for (int i = 0; i < n; i++) {
        dst[i] = (uint16_t)((20 + i * 37) & max);
        src[i] = (uint16_t)((10 + i * 29) & max);
    }
}

static void print_qpel_case_high(const char *label, qpel_fn fn, int size, int bit_depth)
{
    const int stride = 48;
    const int offset = 6 * stride + 6;
    uint16_t dst[48 * 48];
    uint16_t src[48 * 48];

    init_qpel_fixture_high(dst, src, 48 * 48, bit_depth);
    fn((uint8_t *)(dst + offset), (const uint8_t *)(src + offset),
       stride * (ptrdiff_t)sizeof(uint16_t));

    printf("%s", label);
    for (int y = 0; y < size; y++)
        for (int x = 0; x < size; x++)
            printf(" %u", dst[offset + y * stride + x]);
    printf("\n");
}

int main(void)
{
    static const char *suffix[16] = {
        "00", "10", "20", "30", "01", "11", "21", "31",
        "02", "12", "22", "32", "03", "13", "23", "33"
    };
    const int put_sizes[4] = { 2, 4, 8, 16 };
    const int avg_sizes[3] = { 4, 8, 16 };
    const int high_depths[4] = { 9, 10, 12, 14 };
    char label[64];

    for (int s = 0; s < 4; s++) {
        for (int idx = 0; idx < 16; idx++) {
            snprintf(label, sizeof(label), "putqpel%d_%s", put_sizes[s], suffix[idx]);
            print_qpel_case(label, qpel_put_fn(put_sizes[s], idx), put_sizes[s]);
        }
    }
    for (int s = 0; s < 3; s++) {
        for (int idx = 0; idx < 16; idx++) {
            snprintf(label, sizeof(label), "avgqpel%d_%s", avg_sizes[s], suffix[idx]);
            print_qpel_case(label, qpel_avg_fn(avg_sizes[s], idx), avg_sizes[s]);
        }
    }
    for (int d = 0; d < 4; d++) {
        int depth = high_depths[d];
        for (int s = 0; s < 4; s++) {
            for (int idx = 0; idx < 16; idx++) {
                snprintf(label, sizeof(label), "putqpel%d_%d_%s", depth, put_sizes[s], suffix[idx]);
                print_qpel_case_high(label, qpel_put_fn_high(put_sizes[s], idx, depth), put_sizes[s], depth);
            }
        }
        for (int s = 0; s < 3; s++) {
            for (int idx = 0; idx < 16; idx++) {
                snprintf(label, sizeof(label), "avgqpel%d_%d_%s", depth, avg_sizes[s], suffix[idx]);
                print_qpel_case_high(label, qpel_avg_fn_high(avg_sizes[s], idx, depth), avg_sizes[s], depth);
            }
        }
    }
    return 0;
}
`

const qpelOracleCommonH = `
#ifndef GOH264_QPEL_COMMON_H
#define GOH264_QPEL_COMMON_H
#include <stdint.h>

static inline uint8_t av_clip_uint8(int v)
{
    if (v < 0)
        return 0;
    if (v > 255)
        return 255;
    return (uint8_t)v;
}

static inline unsigned av_clip_uintp2(int v, int p)
{
    const int max = (1 << p) - 1;
    if (v < 0)
        return 0;
    if (v > max)
        return max;
    return (unsigned)v;
}
#endif
`

const qpelOracleIntreadwriteH = `
#ifndef GOH264_QPEL_INTREADWRITE_H
#define GOH264_QPEL_INTREADWRITE_H
#include <stdint.h>

static inline uint16_t AV_RN16(const void *p)
{
    const uint8_t *b = (const uint8_t *)p;
    return (uint16_t)b[0] | ((uint16_t)b[1] << 8);
}

static inline uint32_t AV_RN32(const void *p)
{
    const uint8_t *b = (const uint8_t *)p;
    return (uint32_t)b[0] | ((uint32_t)b[1] << 8) |
           ((uint32_t)b[2] << 16) | ((uint32_t)b[3] << 24);
}

static inline uint64_t AV_RN64(const void *p)
{
    const uint8_t *b = (const uint8_t *)p;
    return (uint64_t)b[0] | ((uint64_t)b[1] << 8) |
           ((uint64_t)b[2] << 16) | ((uint64_t)b[3] << 24) |
           ((uint64_t)b[4] << 32) | ((uint64_t)b[5] << 40) |
           ((uint64_t)b[6] << 48) | ((uint64_t)b[7] << 56);
}

static inline uint32_t AV_RN32A(const void *p)
{
    return AV_RN32(p);
}

static inline uint64_t AV_RN64A(const void *p)
{
    return AV_RN64(p);
}

static inline void AV_WN16(void *p, uint16_t v)
{
    uint8_t *b = (uint8_t *)p;
    b[0] = v;
    b[1] = v >> 8;
}

static inline void AV_WN32(void *p, uint32_t v)
{
    uint8_t *b = (uint8_t *)p;
    b[0] = v;
    b[1] = v >> 8;
    b[2] = v >> 16;
    b[3] = v >> 24;
}

static inline void AV_WN64(void *p, uint64_t v)
{
    uint8_t *b = (uint8_t *)p;
    b[0] = v;
    b[1] = v >> 8;
    b[2] = v >> 16;
    b[3] = v >> 24;
    b[4] = v >> 32;
    b[5] = v >> 40;
    b[6] = v >> 48;
    b[7] = v >> 56;
}

static inline void AV_WN32A(void *p, uint32_t v)
{
    AV_WN32(p, v);
}

static inline void AV_WN64A(void *p, uint64_t v)
{
    AV_WN64(p, v);
}
#endif
`

func TestH264QpelUpstreamOracle(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run pinned FFmpeg H.264 qpel oracle")
	}
	cc, err := exec.LookPath("cc")
	if err != nil {
		t.Skip("cc not available")
	}

	root := h264RepoRoot(t)
	upstreamDir := filepath.Join(root, ".upstream", "ffmpeg-n8.0.1", "libavcodec")
	qpelTemplate, err := os.ReadFile(filepath.Join(upstreamDir, "h264qpel_template.c"))
	if err != nil {
		t.Skipf("pinned upstream H.264 qpel source not available: %v", err)
	}
	hpelTemplate, err := os.ReadFile(filepath.Join(upstreamDir, "hpel_template.c"))
	if err != nil {
		t.Skipf("pinned upstream hpel source not available: %v", err)
	}
	pelTemplate, err := os.ReadFile(filepath.Join(upstreamDir, "pel_template.c"))
	if err != nil {
		t.Skipf("pinned upstream pel source not available: %v", err)
	}
	pixelsH, err := os.ReadFile(filepath.Join(upstreamDir, "pixels.h"))
	if err != nil {
		t.Skipf("pinned upstream pixels header not available: %v", err)
	}
	rndAvgH, err := os.ReadFile(filepath.Join(upstreamDir, "rnd_avg.h"))
	if err != nil {
		t.Skipf("pinned upstream rounded-average header not available: %v", err)
	}
	bitDepthTemplate, err := os.ReadFile(filepath.Join(upstreamDir, "bit_depth_template.c"))
	if err != nil {
		t.Skipf("pinned upstream bit-depth source not available: %v", err)
	}

	dir := t.TempDir()
	writeOracleFile(t, filepath.Join(dir, "oracle.c"), qpelOracleC)
	writeOracleFile(t, filepath.Join(dir, "h264qpel_template.c"), string(qpelTemplate))
	writeOracleFile(t, filepath.Join(dir, "hpel_template.c"), string(hpelTemplate))
	writeOracleFile(t, filepath.Join(dir, "pel_template.c"), string(pelTemplate))
	writeOracleFile(t, filepath.Join(dir, "pixels.h"), string(pixelsH))
	writeOracleFile(t, filepath.Join(dir, "rnd_avg.h"), string(rndAvgH))
	writeOracleFile(t, filepath.Join(dir, "bit_depth_template.c"), string(bitDepthTemplate))
	writeOracleFile(t, filepath.Join(dir, "mathops.h"), "")
	if err := os.Mkdir(filepath.Join(dir, "libavutil"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeOracleFile(t, filepath.Join(dir, "libavutil", "common.h"), qpelOracleCommonH)
	writeOracleFile(t, filepath.Join(dir, "libavutil", "intreadwrite.h"), qpelOracleIntreadwriteH)

	bin := filepath.Join(dir, "oracle")
	cmd := exec.Command(cc, "-std=c99", "-I"+dir, filepath.Join(dir, "oracle.c"), "-o", bin)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("compile H.264 qpel oracle: %v\n%s", err, out)
	}

	out, err := exec.Command(bin).CombinedOutput()
	if err != nil {
		t.Fatalf("run H.264 qpel oracle: %v\n%s", err, out)
	}

	got := strings.TrimSpace(string(out))
	want := strings.TrimSpace(h264QpelOracleWant(t))
	if got != want {
		t.Fatalf("H.264 qpel oracle mismatch\nC oracle:\n%s\nGo:\n%s", got, want)
	}
}

func h264QpelOracleWant(t *testing.T) string {
	t.Helper()
	var b strings.Builder
	for _, size := range []int{2, 4, 8, 16} {
		for idx, suffix := range h264QpelOracleSuffixes {
			printQpelOracleWant(t, &b, fmt.Sprintf("putqpel%d_%s", size, suffix), size, idx, false)
		}
	}
	for _, size := range []int{4, 8, 16} {
		for idx, suffix := range h264QpelOracleSuffixes {
			printQpelOracleWant(t, &b, fmt.Sprintf("avgqpel%d_%s", size, suffix), size, idx, true)
		}
	}
	for _, bitDepth := range []int{9, 10, 12, 14} {
		for _, size := range []int{2, 4, 8, 16} {
			for idx, suffix := range h264QpelOracleSuffixes {
				printQpelOracleWantHigh(t, &b, fmt.Sprintf("putqpel%d_%d_%s", bitDepth, size, suffix), size, idx, false, bitDepth)
			}
		}
		for _, size := range []int{4, 8, 16} {
			for idx, suffix := range h264QpelOracleSuffixes {
				printQpelOracleWantHigh(t, &b, fmt.Sprintf("avgqpel%d_%d_%s", bitDepth, size, suffix), size, idx, true, bitDepth)
			}
		}
	}
	return b.String()
}

var h264QpelOracleSuffixes = [16]string{
	"00", "10", "20", "30",
	"01", "11", "21", "31",
	"02", "12", "22", "32",
	"03", "13", "23", "33",
}

func printQpelOracleWant(t *testing.T, b *strings.Builder, label string, size int, idx int, avg bool) {
	t.Helper()
	const stride = 48
	const offset = 6*stride + 6
	dst, src := makeQpelOracleFixture(stride, 48)
	mx, my := idx%4, idx/4
	var err error
	if avg {
		err = h264AvgH264QpelMC(dst, offset, src, offset, stride, size, mx, my)
	} else {
		err = h264PutH264QpelMC(dst, offset, src, offset, stride, size, mx, my)
	}
	if err != nil {
		t.Fatal(err)
	}
	fmt.Fprint(b, label)
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			fmt.Fprintf(b, " %d", dst[offset+y*stride+x])
		}
	}
	fmt.Fprint(b, "\n")
}

func printQpelOracleWantHigh(t *testing.T, b *strings.Builder, label string, size int, idx int, avg bool, bitDepth int) {
	t.Helper()
	const stride = 48
	const offset = 6*stride + 6
	dst, src := makeQpelOracleFixtureHigh(stride, 48, bitDepth)
	mx, my := idx%4, idx/4
	var err error
	if avg {
		err = h264AvgH264QpelMCHigh(dst, offset, src, offset, stride, size, mx, my, bitDepth)
	} else {
		err = h264PutH264QpelMCHigh(dst, offset, src, offset, stride, size, mx, my, bitDepth)
	}
	if err != nil {
		t.Fatal(err)
	}
	fmt.Fprint(b, label)
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			fmt.Fprintf(b, " %d", dst[offset+y*stride+x])
		}
	}
	fmt.Fprint(b, "\n")
}

func makeQpelOracleFixture(stride int, rows int) ([]uint8, []uint8) {
	dst := make([]uint8, stride*rows)
	src := make([]uint8, stride*rows)
	for i := range dst {
		dst[i] = uint8((20 + i*11) & 255)
		src[i] = uint8((10 + i*9) & 255)
	}
	return dst, src
}

func makeQpelOracleFixtureHigh(stride int, rows int, bitDepth int) ([]uint16, []uint16) {
	dst := make([]uint16, stride*rows)
	src := make([]uint16, stride*rows)
	max := (1 << uint(bitDepth)) - 1
	for i := range dst {
		dst[i] = uint16((20 + i*37) & max)
		src[i] = uint16((10 + i*29) & max)
	}
	return dst, src
}
