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

const idctOracleC = `
#include <stdint.h>
#include <stdio.h>
#include <string.h>

#define BIT_DEPTH 8
#include "h264idct_template.c"
#undef BIT_DEPTH
#define BIT_DEPTH 9
#include "h264idct_template.c"
#undef BIT_DEPTH
#define BIT_DEPTH 10
#include "h264idct_template.c"
#undef BIT_DEPTH
#define BIT_DEPTH 12
#include "h264idct_template.c"
#undef BIT_DEPTH
#define BIT_DEPTH 14
#include "h264idct_template.c"
#undef BIT_DEPTH

static int sum_i16(const int16_t *v, int n)
{
    int sum = 0;
    for (int i = 0; i < n; i++)
        sum += v[i];
    return sum;
}

static long long sum_i32(const int32_t *v, int n)
{
    long long sum = 0;
    for (int i = 0; i < n; i++)
        sum += v[i];
    return sum;
}

static void call_idct4_high(int bit_depth, uint16_t *dst, int32_t *block, int stride)
{
    switch (bit_depth) {
    case 9:
        ff_h264_idct_add_9_c((uint8_t *)dst, (int16_t *)block, stride * (int)sizeof(uint16_t));
        return;
    case 10:
        ff_h264_idct_add_10_c((uint8_t *)dst, (int16_t *)block, stride * (int)sizeof(uint16_t));
        return;
    case 12:
        ff_h264_idct_add_12_c((uint8_t *)dst, (int16_t *)block, stride * (int)sizeof(uint16_t));
        return;
    case 14:
        ff_h264_idct_add_14_c((uint8_t *)dst, (int16_t *)block, stride * (int)sizeof(uint16_t));
        return;
    }
}

static void call_idct8_high(int bit_depth, uint16_t *dst, int32_t *block, int stride)
{
    switch (bit_depth) {
    case 9:
        ff_h264_idct8_add_9_c((uint8_t *)dst, (int16_t *)block, stride * (int)sizeof(uint16_t));
        return;
    case 10:
        ff_h264_idct8_add_10_c((uint8_t *)dst, (int16_t *)block, stride * (int)sizeof(uint16_t));
        return;
    case 12:
        ff_h264_idct8_add_12_c((uint8_t *)dst, (int16_t *)block, stride * (int)sizeof(uint16_t));
        return;
    case 14:
        ff_h264_idct8_add_14_c((uint8_t *)dst, (int16_t *)block, stride * (int)sizeof(uint16_t));
        return;
    }
}

static void call_idct4dc_high(int bit_depth, uint16_t *dst, int32_t *block, int stride)
{
    switch (bit_depth) {
    case 9:
        ff_h264_idct_dc_add_9_c((uint8_t *)dst, (int16_t *)block, stride * (int)sizeof(uint16_t));
        return;
    case 10:
        ff_h264_idct_dc_add_10_c((uint8_t *)dst, (int16_t *)block, stride * (int)sizeof(uint16_t));
        return;
    case 12:
        ff_h264_idct_dc_add_12_c((uint8_t *)dst, (int16_t *)block, stride * (int)sizeof(uint16_t));
        return;
    case 14:
        ff_h264_idct_dc_add_14_c((uint8_t *)dst, (int16_t *)block, stride * (int)sizeof(uint16_t));
        return;
    }
}

static void call_idct8dc_high(int bit_depth, uint16_t *dst, int32_t *block, int stride)
{
    switch (bit_depth) {
    case 9:
        ff_h264_idct8_dc_add_9_c((uint8_t *)dst, (int16_t *)block, stride * (int)sizeof(uint16_t));
        return;
    case 10:
        ff_h264_idct8_dc_add_10_c((uint8_t *)dst, (int16_t *)block, stride * (int)sizeof(uint16_t));
        return;
    case 12:
        ff_h264_idct8_dc_add_12_c((uint8_t *)dst, (int16_t *)block, stride * (int)sizeof(uint16_t));
        return;
    case 14:
        ff_h264_idct8_dc_add_14_c((uint8_t *)dst, (int16_t *)block, stride * (int)sizeof(uint16_t));
        return;
    }
}

static void call_luma_dc_high(int bit_depth, int32_t *output, int32_t *input, int qmul)
{
    switch (bit_depth) {
    case 9:
        ff_h264_luma_dc_dequant_idct_9_c((int16_t *)output, (int16_t *)input, qmul);
        return;
    case 10:
        ff_h264_luma_dc_dequant_idct_10_c((int16_t *)output, (int16_t *)input, qmul);
        return;
    case 12:
        ff_h264_luma_dc_dequant_idct_12_c((int16_t *)output, (int16_t *)input, qmul);
        return;
    case 14:
        ff_h264_luma_dc_dequant_idct_14_c((int16_t *)output, (int16_t *)input, qmul);
        return;
    }
}

static void call_chroma420_dc_high(int bit_depth, int32_t *block, int qmul)
{
    switch (bit_depth) {
    case 9:
        ff_h264_chroma_dc_dequant_idct_9_c((int16_t *)block, qmul);
        return;
    case 10:
        ff_h264_chroma_dc_dequant_idct_10_c((int16_t *)block, qmul);
        return;
    case 12:
        ff_h264_chroma_dc_dequant_idct_12_c((int16_t *)block, qmul);
        return;
    case 14:
        ff_h264_chroma_dc_dequant_idct_14_c((int16_t *)block, qmul);
        return;
    }
}

static void call_chroma422_dc_high(int bit_depth, int32_t *block, int qmul)
{
    switch (bit_depth) {
    case 9:
        ff_h264_chroma422_dc_dequant_idct_9_c((int16_t *)block, qmul);
        return;
    case 10:
        ff_h264_chroma422_dc_dequant_idct_10_c((int16_t *)block, qmul);
        return;
    case 12:
        ff_h264_chroma422_dc_dequant_idct_12_c((int16_t *)block, qmul);
        return;
    case 14:
        ff_h264_chroma422_dc_dequant_idct_14_c((int16_t *)block, qmul);
        return;
    }
}

static void print_idct4(void)
{
    uint8_t dst[8 * 8];
    int16_t block[16] = {
        96, -16, 7, 3,
        -8, 5, 0, 2,
        12, -3, 4, -1,
        1, 0, -2, 6,
    };

    for (int i = 0; i < 8 * 8; i++)
        dst[i] = 90 + (i * 7) % 80;

    ff_h264_idct_add_8_c(dst, block, 8);
    printf("idct4");
    for (int y = 0; y < 4; y++)
        for (int x = 0; x < 4; x++)
            printf(" %u", dst[y * 8 + x]);
    printf(" blocksum %d\n", sum_i16(block, 16));
}

static void print_idct8(void)
{
    uint8_t dst[10 * 10];
    int16_t block[64] = { 0 };

    for (int i = 0; i < 10 * 10; i++)
        dst[i] = 30 + (i * 5) % 160;
    block[0] = 80;
    block[1] = -9;
    block[7] = 4;
    block[9] = 13;
    block[18] = -5;
    block[27] = 3;
    block[63] = -2;

    ff_h264_idct8_add_8_c(dst, block, 10);
    printf("idct8");
    for (int y = 0; y < 8; y++)
        for (int x = 0; x < 8; x++)
            printf(" %u", dst[y * 10 + x]);
    printf(" blocksum %d\n", sum_i16(block, 64));
}

static void print_idct4_edge(void)
{
    uint8_t dst[8 * 8];
    int16_t block[16] = {
        32767, -32768, 30000, -30000,
        -25000, 24000, -23000, 22000,
        21000, -20000, 19000, -18000,
        -17000, 16000, -15000, 14000,
    };

    for (int i = 0; i < 8 * 8; i++)
        dst[i] = (i & 1) ? 255 : 0;

    ff_h264_idct_add_8_c(dst, block, 8);
    printf("idct4edge");
    for (int y = 0; y < 4; y++)
        for (int x = 0; x < 4; x++)
            printf(" %u", dst[y * 8 + x]);
    printf(" blocksum %d\n", sum_i16(block, 16));
}

static void print_idct8_edge(void)
{
    uint8_t dst[10 * 10];
    int16_t block[64] = { 0 };

    for (int i = 0; i < 10 * 10; i++)
        dst[i] = (i * 37) & 255;
    for (int i = 0; i < 64; i++)
        block[i] = (i & 1) ? (int16_t)(-32768 + i * 257) : (int16_t)(32767 - i * 191);

    ff_h264_idct8_add_8_c(dst, block, 10);
    printf("idct8edge");
    for (int y = 0; y < 8; y++)
        for (int x = 0; x < 8; x++)
            printf(" %u", dst[y * 10 + x]);
    printf(" blocksum %d\n", sum_i16(block, 64));
}

static void print_luma_dc(void)
{
    int16_t input[16];
    int16_t output[16 * 16] = { 0 };
    int idx[16] = {
        0, 16, 32, 48,
        64, 80, 96, 112,
        128, 144, 160, 176,
        192, 208, 224, 240,
    };

    for (int i = 0; i < 16; i++)
        input[i] = i * 3 - 20;

    ff_h264_luma_dc_dequant_idct_8_c(output, input, 64);
    printf("luma");
    for (int i = 0; i < 16; i++)
        printf(" %d", output[idx[i]]);
    printf(" untouched %d %d\n", output[1], output[255]);
}

static void print_chroma420_dc(void)
{
    int16_t block[64] = { 0 };

    block[0] = 1;
    block[16] = 2;
    block[32] = 3;
    block[48] = 4;

    ff_h264_chroma_dc_dequant_idct_8_c(block, 64);
    printf("chroma420 %d %d %d %d\n", block[0], block[16], block[32], block[48]);
}

static void print_chroma422_dc(void)
{
    int16_t block[128] = { 0 };
    int idx[8] = { 0, 16, 32, 48, 64, 80, 96, 112 };

    for (int i = 0; i < 8; i++)
        block[idx[i]] = i * 2 - 5;

    ff_h264_chroma422_dc_dequant_idct_8_c(block, 64);
    printf("chroma422");
    for (int i = 0; i < 8; i++)
        printf(" %d", block[idx[i]]);
    printf("\n");
}

static void print_idct4_high(int bit_depth)
{
    uint16_t dst[8 * 8];
    int32_t block[16] = {
        128, -24, 11, 5,
        -14, 9, 0, 3,
        19, -7, 6, -2,
        2, 0, -3, 8,
    };
    int max = (1 << bit_depth) - 1;
    int span = max / 2;

    for (int i = 0; i < 8 * 8; i++)
        dst[i] = (uint16_t)(max / 4 + (i * 97) % span);

    call_idct4_high(bit_depth, dst, block, 8);
    printf("idct4_%d", bit_depth);
    for (int y = 0; y < 4; y++)
        for (int x = 0; x < 4; x++)
            printf(" %u", dst[y * 8 + x]);
    printf(" blocksum %lld\n", sum_i32(block, 16));
}

static void print_idct8_high(int bit_depth)
{
    uint16_t dst[10 * 10];
    int32_t block[64] = { 0 };
    int max = (1 << bit_depth) - 1;
    int span = max / 2;

    for (int i = 0; i < 10 * 10; i++)
        dst[i] = (uint16_t)(max / 5 + (i * 53) % span);
    block[0] = 256;
    block[1] = -21;
    block[7] = 8;
    block[9] = 33;
    block[18] = -15;
    block[27] = 9;
    block[45] = 7;
    block[63] = -4;

    call_idct8_high(bit_depth, dst, block, 10);
    printf("idct8_%d", bit_depth);
    for (int y = 0; y < 8; y++)
        for (int x = 0; x < 8; x++)
            printf(" %u", dst[y * 10 + x]);
    printf(" blocksum %lld\n", sum_i32(block, 64));
}

static void print_idct4dc_high(int bit_depth)
{
    uint16_t dst[6 * 6];
    int32_t block[16] = { 0 };
    int max = (1 << bit_depth) - 1;

    for (int i = 0; i < 6 * 6; i++)
        dst[i] = (i & 1) ? 1 : (uint16_t)(max - 2);
    block[0] = 512;

    call_idct4dc_high(bit_depth, dst, block, 6);
    printf("idct4dc_%d", bit_depth);
    for (int y = 0; y < 4; y++)
        for (int x = 0; x < 4; x++)
            printf(" %u", dst[y * 6 + x]);
    printf(" block0 %d\n", block[0]);
}

static void print_idct8dc_high(int bit_depth)
{
    uint16_t dst[10 * 10];
    int32_t block[64] = { 0 };
    int max = (1 << bit_depth) - 1;

    for (int i = 0; i < 10 * 10; i++)
        dst[i] = (i & 3) ? (uint16_t)(max / 3) : 3;
    block[0] = -256;

    call_idct8dc_high(bit_depth, dst, block, 10);
    printf("idct8dc_%d", bit_depth);
    for (int y = 0; y < 8; y++)
        for (int x = 0; x < 8; x++)
            printf(" %u", dst[y * 10 + x]);
    printf(" block0 %d\n", block[0]);
}

static void print_luma_dc_high(int bit_depth)
{
    int32_t input[16];
    int32_t output[16 * 16] = { 0 };
    int idx[16] = {
        0, 16, 32, 48,
        64, 80, 96, 112,
        128, 144, 160, 176,
        192, 208, 224, 240,
    };

    for (int i = 0; i < 16; i++)
        input[i] = i * 7 - 40;

    call_luma_dc_high(bit_depth, output, input, 96);
    printf("luma_%d", bit_depth);
    for (int i = 0; i < 16; i++)
        printf(" %d", output[idx[i]]);
    printf(" untouched %d %d\n", output[1], output[255]);
}

static void print_chroma420_dc_high(int bit_depth)
{
    int32_t block[64] = { 0 };

    block[0] = 3;
    block[16] = -4;
    block[32] = 8;
    block[48] = -11;

    call_chroma420_dc_high(bit_depth, block, 96);
    printf("chroma420_%d %d %d %d %d\n", bit_depth, block[0], block[16], block[32], block[48]);
}

static void print_chroma422_dc_high(int bit_depth)
{
    int32_t block[128] = { 0 };
    int idx[8] = { 0, 16, 32, 48, 64, 80, 96, 112 };

    for (int i = 0; i < 8; i++)
        block[idx[i]] = i * 3 - 9;

    call_chroma422_dc_high(bit_depth, block, 96);
    printf("chroma422_%d", bit_depth);
    for (int i = 0; i < 8; i++)
        printf(" %d", block[idx[i]]);
    printf("\n");
}

int main(void)
{
    int high_depths[4] = { 9, 10, 12, 14 };

    print_idct4();
    print_idct8();
    print_idct4_edge();
    print_idct8_edge();
    print_luma_dc();
    print_chroma420_dc();
    print_chroma422_dc();
    for (int i = 0; i < 4; i++) {
        print_idct4_high(high_depths[i]);
        print_idct8_high(high_depths[i]);
        print_idct4dc_high(high_depths[i]);
        print_idct8dc_high(high_depths[i]);
        print_luma_dc_high(high_depths[i]);
        print_chroma420_dc_high(high_depths[i]);
        print_chroma422_dc_high(high_depths[i]);
    }
    return 0;
}
`

const idctOracleBitDepthTemplate = `
#include <stdint.h>

#ifndef GOH264_ORACLE_BIT_DEPTH_HELPERS
#define GOH264_ORACLE_BIT_DEPTH_HELPERS

static inline uint8_t goh264_clip_uint8(int v)
{
    if (v < 0)
        return 0;
    if (v > 255)
        return 255;
    return (uint8_t)v;
}

static inline uint16_t goh264_clip_uintp2(int v, int p)
{
    int max = (1 << p) - 1;
    if (v < 0)
        return 0;
    if (v > max)
        return (uint16_t)max;
    return (uint16_t)v;
}

#endif

#undef pixel
#undef dctcoef
#undef SUINT
#undef FUNC3
#undef FUNC2
#undef FUNCC
#undef av_clip_pixel

#if BIT_DEPTH > 8
#define pixel uint16_t
#define dctcoef int32_t
#define av_clip_pixel(a) goh264_clip_uintp2((a), BIT_DEPTH)
#else
#define pixel uint8_t
#define dctcoef int16_t
#define av_clip_pixel(a) goh264_clip_uint8(a)
#endif
#define SUINT unsigned
#define FUNC3(a, b, c)  a ## _ ## b ## c
#define FUNC2(a, b, c)  FUNC3(a, b, c)
#define FUNCC(a) FUNC2(a, BIT_DEPTH, _c)
`

const idctOracleH264Parse = `
#include <stdint.h>

#ifndef GOH264_ORACLE_H264_PARSE_H
#define GOH264_ORACLE_H264_PARSE_H

static const uint8_t scan8[16 * 3 + 3] = {
    4 +  1 * 8, 5 +  1 * 8, 4 +  2 * 8, 5 +  2 * 8,
    6 +  1 * 8, 7 +  1 * 8, 6 +  2 * 8, 7 +  2 * 8,
    4 +  3 * 8, 5 +  3 * 8, 4 +  4 * 8, 5 +  4 * 8,
    6 +  3 * 8, 7 +  3 * 8, 6 +  4 * 8, 7 +  4 * 8,
    4 +  6 * 8, 5 +  6 * 8, 4 +  7 * 8, 5 +  7 * 8,
    6 +  6 * 8, 7 +  6 * 8, 6 +  7 * 8, 7 +  7 * 8,
    4 +  8 * 8, 5 +  8 * 8, 4 +  9 * 8, 5 +  9 * 8,
    6 +  8 * 8, 7 +  8 * 8, 6 +  9 * 8, 7 +  9 * 8,
    4 + 11 * 8, 5 + 11 * 8, 4 + 12 * 8, 5 + 12 * 8,
    6 + 11 * 8, 7 + 11 * 8, 6 + 12 * 8, 7 + 12 * 8,
    4 + 13 * 8, 5 + 13 * 8, 4 + 14 * 8, 5 + 14 * 8,
    6 + 13 * 8, 7 + 13 * 8, 6 + 14 * 8, 7 + 14 * 8,
    0 +  0 * 8, 0 +  5 * 8, 0 + 10 * 8
};

#endif
`

const idctOracleH264IDCTH = `
#include <stdint.h>
`

func TestH264IDCTUpstreamOracle(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run pinned FFmpeg H.264 IDCT oracle")
	}
	cc, err := exec.LookPath("cc")
	if err != nil {
		t.Skip("cc not available")
	}

	root := h264RepoRoot(t)
	upstreamTemplate := filepath.Join(root, ".upstream", "ffmpeg-n8.0.1", "libavcodec", "h264idct_template.c")
	template, err := os.ReadFile(upstreamTemplate)
	if err != nil {
		t.Skipf("pinned upstream H.264 IDCT source not available: %v", err)
	}

	dir := t.TempDir()
	writeOracleFile(t, filepath.Join(dir, "oracle.c"), idctOracleC)
	writeOracleFile(t, filepath.Join(dir, "h264idct_template.c"), string(template))
	writeOracleFile(t, filepath.Join(dir, "bit_depth_template.c"), idctOracleBitDepthTemplate)
	writeOracleFile(t, filepath.Join(dir, "h264_parse.h"), idctOracleH264Parse)
	writeOracleFile(t, filepath.Join(dir, "h264idct.h"), idctOracleH264IDCTH)
	if err := os.Mkdir(filepath.Join(dir, "libavutil"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeOracleFile(t, filepath.Join(dir, "libavutil", "common.h"), "")

	bin := filepath.Join(dir, "oracle")
	cmd := exec.Command(cc, "-std=c99", "-Wall", "-Wextra", "-I"+dir, filepath.Join(dir, "oracle.c"), "-o", bin)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("compile H.264 IDCT oracle: %v\n%s", err, out)
	}

	out, err := exec.Command(bin).CombinedOutput()
	if err != nil {
		t.Fatalf("run H.264 IDCT oracle: %v\n%s", err, out)
	}

	got := strings.TrimSpace(string(out))
	want := strings.TrimSpace(h264IDCTOracleWant(t))
	if got != want {
		t.Fatalf("H.264 IDCT oracle mismatch\nC oracle:\n%s\nGo:\n%s", got, want)
	}
}

func h264IDCTOracleWant(t *testing.T) string {
	t.Helper()
	var b strings.Builder
	printIDCT4OracleWant(t, &b)
	printIDCT8OracleWant(t, &b)
	printIDCT4EdgeOracleWant(t, &b)
	printIDCT8EdgeOracleWant(t, &b)
	printLumaDCOracleWant(t, &b)
	printChroma420DCOracleWant(t, &b)
	printChroma422DCOracleWant(t, &b)
	for _, bitDepth := range []int{9, 10, 12, 14} {
		printIDCT4HighOracleWant(t, &b, bitDepth)
		printIDCT8HighOracleWant(t, &b, bitDepth)
		printIDCT4DCHighOracleWant(t, &b, bitDepth)
		printIDCT8DCHighOracleWant(t, &b, bitDepth)
		printLumaDCHighOracleWant(t, &b, bitDepth)
		printChroma420DCHighOracleWant(t, &b, bitDepth)
		printChroma422DCHighOracleWant(t, &b, bitDepth)
	}
	return b.String()
}

func printIDCT4OracleWant(t *testing.T, b *strings.Builder) {
	t.Helper()
	dst := make([]uint8, 8*8)
	block := []int32{
		96, -16, 7, 3,
		-8, 5, 0, 2,
		12, -3, 4, -1,
		1, 0, -2, 6,
	}
	for i := range dst {
		dst[i] = uint8(90 + (i*7)%80)
	}

	if err := h264IDCTAdd(dst, block, 8); err != nil {
		t.Fatal(err)
	}
	fmt.Fprint(b, "idct4")
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			fmt.Fprintf(b, " %d", dst[y*8+x])
		}
	}
	fmt.Fprintf(b, " blocksum %d\n", sumInt32(block[:16]))
}

func printIDCT8OracleWant(t *testing.T, b *strings.Builder) {
	t.Helper()
	dst := make([]uint8, 10*10)
	block := make([]int32, 64)
	for i := range dst {
		dst[i] = uint8(30 + (i*5)%160)
	}
	block[0] = 80
	block[1] = -9
	block[7] = 4
	block[9] = 13
	block[18] = -5
	block[27] = 3
	block[63] = -2

	if err := h264IDCT8Add(dst, block, 10); err != nil {
		t.Fatal(err)
	}
	fmt.Fprint(b, "idct8")
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			fmt.Fprintf(b, " %d", dst[y*10+x])
		}
	}
	fmt.Fprintf(b, " blocksum %d\n", sumInt32(block[:64]))
}

func printIDCT4EdgeOracleWant(t *testing.T, b *strings.Builder) {
	t.Helper()
	dst := make([]uint8, 8*8)
	block := []int32{
		32767, -32768, 30000, -30000,
		-25000, 24000, -23000, 22000,
		21000, -20000, 19000, -18000,
		-17000, 16000, -15000, 14000,
	}
	for i := range dst {
		if i&1 == 1 {
			dst[i] = 255
		}
	}

	if err := h264IDCTAdd(dst, block, 8); err != nil {
		t.Fatal(err)
	}
	fmt.Fprint(b, "idct4edge")
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			fmt.Fprintf(b, " %d", dst[y*8+x])
		}
	}
	fmt.Fprintf(b, " blocksum %d\n", sumInt32(block[:16]))
}

func printIDCT8EdgeOracleWant(t *testing.T, b *strings.Builder) {
	t.Helper()
	dst := make([]uint8, 10*10)
	block := make([]int32, 64)
	for i := range dst {
		dst[i] = uint8((i * 37) & 255)
	}
	for i := range block {
		if i&1 == 1 {
			block[i] = int32(int16(-32768 + i*257))
		} else {
			block[i] = int32(int16(32767 - i*191))
		}
	}

	if err := h264IDCT8Add(dst, block, 10); err != nil {
		t.Fatal(err)
	}
	fmt.Fprint(b, "idct8edge")
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			fmt.Fprintf(b, " %d", dst[y*10+x])
		}
	}
	fmt.Fprintf(b, " blocksum %d\n", sumInt32(block[:64]))
}

func printLumaDCOracleWant(t *testing.T, b *strings.Builder) {
	t.Helper()
	var input [16]int32
	output := make([]int32, 16*16)
	idx := [...]int{
		0, 16, 32, 48,
		64, 80, 96, 112,
		128, 144, 160, 176,
		192, 208, 224, 240,
	}
	for i := range input {
		input[i] = int32(i*3 - 20)
	}

	if err := h264LumaDCDequantIDCT(output, &input, 64); err != nil {
		t.Fatal(err)
	}
	fmt.Fprint(b, "luma")
	for _, outIdx := range idx {
		fmt.Fprintf(b, " %d", dctcoef8Value(output[outIdx]))
	}
	fmt.Fprintf(b, " untouched %d %d\n", dctcoef8Value(output[1]), dctcoef8Value(output[255]))
}

func printChroma420DCOracleWant(t *testing.T, b *strings.Builder) {
	t.Helper()
	block := make([]int32, 64)
	block[0] = 1
	block[16] = 2
	block[32] = 3
	block[48] = 4

	if err := h264ChromaDCDequantIDCT(block, 64); err != nil {
		t.Fatal(err)
	}
	fmt.Fprintf(b, "chroma420 %d %d %d %d\n",
		dctcoef8Value(block[0]), dctcoef8Value(block[16]),
		dctcoef8Value(block[32]), dctcoef8Value(block[48]))
}

func printChroma422DCOracleWant(t *testing.T, b *strings.Builder) {
	t.Helper()
	block := make([]int32, 128)
	idx := [...]int{0, 16, 32, 48, 64, 80, 96, 112}
	for i, blockIdx := range idx {
		block[blockIdx] = int32(i*2 - 5)
	}

	if err := h264Chroma422DCDequantIDCT(block, 64); err != nil {
		t.Fatal(err)
	}
	fmt.Fprint(b, "chroma422")
	for _, blockIdx := range idx {
		fmt.Fprintf(b, " %d", dctcoef8Value(block[blockIdx]))
	}
	fmt.Fprint(b, "\n")
}

func printIDCT4HighOracleWant(t *testing.T, b *strings.Builder, bitDepth int) {
	t.Helper()
	dst := make([]uint16, 8*8)
	block := []int32{
		128, -24, 11, 5,
		-14, 9, 0, 3,
		19, -7, 6, -2,
		2, 0, -3, 8,
	}
	max := (1 << uint(bitDepth)) - 1
	span := max / 2
	for i := range dst {
		dst[i] = uint16(max/4 + (i*97)%span)
	}

	if err := h264IDCTAddHigh(dst, block, 8, bitDepth); err != nil {
		t.Fatal(err)
	}
	fmt.Fprintf(b, "idct4_%d", bitDepth)
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			fmt.Fprintf(b, " %d", dst[y*8+x])
		}
	}
	fmt.Fprintf(b, " blocksum %d\n", sumInt32Raw(block[:16]))
}

func printIDCT8HighOracleWant(t *testing.T, b *strings.Builder, bitDepth int) {
	t.Helper()
	dst := make([]uint16, 10*10)
	block := make([]int32, 64)
	max := (1 << uint(bitDepth)) - 1
	span := max / 2
	for i := range dst {
		dst[i] = uint16(max/5 + (i*53)%span)
	}
	block[0] = 256
	block[1] = -21
	block[7] = 8
	block[9] = 33
	block[18] = -15
	block[27] = 9
	block[45] = 7
	block[63] = -4

	if err := h264IDCT8AddHigh(dst, block, 10, bitDepth); err != nil {
		t.Fatal(err)
	}
	fmt.Fprintf(b, "idct8_%d", bitDepth)
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			fmt.Fprintf(b, " %d", dst[y*10+x])
		}
	}
	fmt.Fprintf(b, " blocksum %d\n", sumInt32Raw(block[:64]))
}

func printIDCT4DCHighOracleWant(t *testing.T, b *strings.Builder, bitDepth int) {
	t.Helper()
	dst := make([]uint16, 6*6)
	block := make([]int32, 16)
	max := (1 << uint(bitDepth)) - 1
	for i := range dst {
		if i&1 == 1 {
			dst[i] = 1
		} else {
			dst[i] = uint16(max - 2)
		}
	}
	block[0] = 512

	if err := h264IDCTDCAddHigh(dst, block, 6, bitDepth); err != nil {
		t.Fatal(err)
	}
	fmt.Fprintf(b, "idct4dc_%d", bitDepth)
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			fmt.Fprintf(b, " %d", dst[y*6+x])
		}
	}
	fmt.Fprintf(b, " block0 %d\n", block[0])
}

func printIDCT8DCHighOracleWant(t *testing.T, b *strings.Builder, bitDepth int) {
	t.Helper()
	dst := make([]uint16, 10*10)
	block := make([]int32, 64)
	max := (1 << uint(bitDepth)) - 1
	for i := range dst {
		if i&3 == 0 {
			dst[i] = 3
		} else {
			dst[i] = uint16(max / 3)
		}
	}
	block[0] = -256

	if err := h264IDCT8DCAddHigh(dst, block, 10, bitDepth); err != nil {
		t.Fatal(err)
	}
	fmt.Fprintf(b, "idct8dc_%d", bitDepth)
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			fmt.Fprintf(b, " %d", dst[y*10+x])
		}
	}
	fmt.Fprintf(b, " block0 %d\n", block[0])
}

func printLumaDCHighOracleWant(t *testing.T, b *strings.Builder, bitDepth int) {
	t.Helper()
	var input [16]int32
	output := make([]int32, 16*16)
	idx := [...]int{
		0, 16, 32, 48,
		64, 80, 96, 112,
		128, 144, 160, 176,
		192, 208, 224, 240,
	}
	for i := range input {
		input[i] = int32(i*7 - 40)
	}

	if err := h264LumaDCDequantIDCTHigh(output, &input, 96); err != nil {
		t.Fatal(err)
	}
	fmt.Fprintf(b, "luma_%d", bitDepth)
	for _, outIdx := range idx {
		fmt.Fprintf(b, " %d", output[outIdx])
	}
	fmt.Fprintf(b, " untouched %d %d\n", output[1], output[255])
}

func printChroma420DCHighOracleWant(t *testing.T, b *strings.Builder, bitDepth int) {
	t.Helper()
	block := make([]int32, 64)
	block[0] = 3
	block[16] = -4
	block[32] = 8
	block[48] = -11

	if err := h264ChromaDCDequantIDCTHigh(block, 96); err != nil {
		t.Fatal(err)
	}
	fmt.Fprintf(b, "chroma420_%d %d %d %d %d\n", bitDepth, block[0], block[16], block[32], block[48])
}

func printChroma422DCHighOracleWant(t *testing.T, b *strings.Builder, bitDepth int) {
	t.Helper()
	block := make([]int32, 128)
	idx := [...]int{0, 16, 32, 48, 64, 80, 96, 112}
	for i, blockIdx := range idx {
		block[blockIdx] = int32(i*3 - 9)
	}

	if err := h264Chroma422DCDequantIDCTHigh(block, 96); err != nil {
		t.Fatal(err)
	}
	fmt.Fprintf(b, "chroma422_%d", bitDepth)
	for _, blockIdx := range idx {
		fmt.Fprintf(b, " %d", block[blockIdx])
	}
	fmt.Fprint(b, "\n")
}

func sumInt32(v []int32) int {
	sum := 0
	for _, value := range v {
		sum += int(dctcoef8Value(value))
	}
	return sum
}

func sumInt32Raw(v []int32) int64 {
	var sum int64
	for _, value := range v {
		sum += int64(value)
	}
	return sum
}
