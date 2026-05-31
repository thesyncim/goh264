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
#undef pixeltmp

typedef void (*qpel_fn)(uint8_t *dst, const uint8_t *src, ptrdiff_t stride);

#define QPEL_FN(prefix, size, suffix) prefix ## size ## _mc ## suffix ## _8_c
#define QPEL_LIST(prefix, size) { \
    QPEL_FN(prefix, size, 00), QPEL_FN(prefix, size, 10), \
    QPEL_FN(prefix, size, 20), QPEL_FN(prefix, size, 30), \
    QPEL_FN(prefix, size, 01), QPEL_FN(prefix, size, 11), \
    QPEL_FN(prefix, size, 21), QPEL_FN(prefix, size, 31), \
    QPEL_FN(prefix, size, 02), QPEL_FN(prefix, size, 12), \
    QPEL_FN(prefix, size, 22), QPEL_FN(prefix, size, 32), \
    QPEL_FN(prefix, size, 03), QPEL_FN(prefix, size, 13), \
    QPEL_FN(prefix, size, 23), QPEL_FN(prefix, size, 33) \
}

static qpel_fn put2[16] = QPEL_LIST(put_h264_qpel, 2);
static qpel_fn put4[16] = QPEL_LIST(put_h264_qpel, 4);
static qpel_fn put8[16] = QPEL_LIST(put_h264_qpel, 8);
static qpel_fn put16[16] = QPEL_LIST(put_h264_qpel, 16);
static qpel_fn avg4[16] = QPEL_LIST(avg_h264_qpel, 4);
static qpel_fn avg8[16] = QPEL_LIST(avg_h264_qpel, 8);
static qpel_fn avg16[16] = QPEL_LIST(avg_h264_qpel, 16);

static qpel_fn qpel_put_fn(int size, int idx)
{
    switch (size) {
    case 2:
        return put2[idx];
    case 4:
        return put4[idx];
    case 8:
        return put8[idx];
    case 16:
        return put16[idx];
    }
    return 0;
}

static qpel_fn qpel_avg_fn(int size, int idx)
{
    switch (size) {
    case 4:
        return avg4[idx];
    case 8:
        return avg8[idx];
    case 16:
        return avg16[idx];
    }
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

int main(void)
{
    static const char *suffix[16] = {
        "00", "10", "20", "30", "01", "11", "21", "31",
        "02", "12", "22", "32", "03", "13", "23", "33"
    };
    const int put_sizes[4] = { 2, 4, 8, 16 };
    const int avg_sizes[3] = { 4, 8, 16 };
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

func makeQpelOracleFixture(stride int, rows int) ([]uint8, []uint8) {
	dst := make([]uint8, stride*rows)
	src := make([]uint8, stride*rows)
	for i := range dst {
		dst[i] = uint8((20 + i*11) & 255)
		src[i] = uint8((10 + i*9) & 255)
	}
	return dst, src
}
