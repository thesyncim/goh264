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

static int sum_i16(const int16_t *v, int n)
{
    int sum = 0;
    for (int i = 0; i < n; i++)
        sum += v[i];
    return sum;
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

int main(void)
{
    print_idct4();
    print_idct8();
    print_idct4_edge();
    print_idct8_edge();
    print_luma_dc();
    print_chroma420_dc();
    print_chroma422_dc();
    return 0;
}
`

const idctOracleBitDepthTemplate = `
#include <stdint.h>

static inline uint8_t goh264_clip_uint8(int v)
{
    if (v < 0)
        return 0;
    if (v > 255)
        return 255;
    return (uint8_t)v;
}

#define pixel uint8_t
#define dctcoef int16_t
#define SUINT unsigned
#define FUNC3(a, b, c)  a ## _ ## b ## c
#define FUNC2(a, b, c)  FUNC3(a, b, c)
#define FUNCC(a) FUNC2(a, BIT_DEPTH, _c)
#define av_clip_pixel(a) goh264_clip_uint8(a)
`

const idctOracleH264Parse = `
#include <stdint.h>

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

func sumInt32(v []int32) int {
	sum := 0
	for _, value := range v {
		sum += int(dctcoef8Value(value))
	}
	return sum
}
