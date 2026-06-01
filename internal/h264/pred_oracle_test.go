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

const predOracleC = `
#include <stddef.h>
#include <stdint.h>
#include <stdio.h>
#include <string.h>

#define BIT_DEPTH 8
#include "h264pred_template.c"
#undef BIT_DEPTH
#define BIT_DEPTH 9
#include "h264pred_template.c"
#undef BIT_DEPTH
#define BIT_DEPTH 10
#include "h264pred_template.c"
#undef BIT_DEPTH
#define BIT_DEPTH 12
#include "h264pred_template.c"
#undef BIT_DEPTH
#define BIT_DEPTH 14
#include "h264pred_template.c"
#undef BIT_DEPTH

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

typedef void (*pred_fn)(uint8_t *src, ptrdiff_t stride);
typedef void (*pred4_fn)(uint8_t *src, const uint8_t *topright,
                         ptrdiff_t stride);
typedef void (*pred8l_fn)(uint8_t *src, int has_topleft, int has_topright,
                          ptrdiff_t stride);
typedef void (*pred4_add_fn)(uint8_t *pix, int16_t *block, ptrdiff_t stride);
typedef void (*pred8l_add_fn)(uint8_t *src, int16_t *block, int has_topleft,
                              int has_topright, ptrdiff_t stride);
typedef void (*pred_offset_add_fn)(uint8_t *pix, const int *block_offset,
                                   int16_t *block, ptrdiff_t stride);

static void init_fixture(uint8_t *pix, int stride, int rows)
{
    for (int y = 0; y < rows; y++)
        for (int x = 0; x < stride; x++)
            pix[y * stride + x] = 30 + (x * 5 + y * 7) % 180;
}

static void init_block(int16_t *block, int n)
{
    for (int i = 0; i < n; i++)
        block[i] = (i % 7) * 3 - 9;
}

static int sum_i16(const int16_t *block, int n)
{
    int sum = 0;
    for (int i = 0; i < n; i++)
        sum += block[i];
    return sum;
}

static void init_offsets(int *offset, int stride)
{
    const int base = scan8[0];
    for (int i = 0; i < 16; i++) {
        int delta = scan8[i] - base;
        offset[i] = 4 * (delta & 7) + 4 * stride * (delta >> 3);
        offset[16 + i] = offset[i];
        offset[32 + i] = offset[i];
    }
}

static void print_block(const char *label, const uint8_t *pix, int stride,
                        int offset, int width, int height)
{
    printf("%s", label);
    for (int y = 0; y < height; y++)
        for (int x = 0; x < width; x++)
            printf(" %u", pix[offset + y * stride + x]);
    printf("\n");
}

static void print_pred16(const char *label, pred_fn fn)
{
    const int stride = 24;
    const int offset = 4 * stride + 4;
    uint8_t pix[24 * 24];
    init_fixture(pix, stride, 24);
    fn(pix + offset, stride);
    print_block(label, pix, stride, offset, 16, 16);
}

static void print_pred8(const char *label, pred_fn fn)
{
    const int stride = 16;
    const int offset = 4 * stride + 4;
    uint8_t pix[16 * 16];
    init_fixture(pix, stride, 16);
    fn(pix + offset, stride);
    print_block(label, pix, stride, offset, 8, 8);
}

static void print_pred8x16(const char *label, pred_fn fn)
{
    const int stride = 16;
    const int offset = 4 * stride + 4;
    uint8_t pix[16 * 24];
    init_fixture(pix, stride, 24);
    fn(pix + offset, stride);
    print_block(label, pix, stride, offset, 8, 16);
}

static void print_pred4(const char *label, pred4_fn fn)
{
    const int stride = 12;
    const int offset = 3 * stride + 3;
    uint8_t pix[12 * 12];
    const uint8_t topright[4] = { 91, 123, 155, 177 };
    init_fixture(pix, stride, 12);
    fn(pix + offset, topright, stride);
    print_block(label, pix, stride, offset, 4, 4);
}

static void print_pred8l(const char *label, pred8l_fn fn,
                         int has_topleft, int has_topright)
{
    const int stride = 28;
    const int offset = 5 * stride + 5;
    uint8_t pix[28 * 18];
    init_fixture(pix, stride, 18);
    fn(pix + offset, has_topleft, has_topright, stride);
    print_block(label, pix, stride, offset, 8, 8);
}

static void print_pred8l_cases(const char *label, pred8l_fn fn)
{
    char name[64];
    snprintf(name, sizeof(name), "%s_11", label);
    print_pred8l(name, fn, 1, 1);
    snprintf(name, sizeof(name), "%s_00", label);
    print_pred8l(name, fn, 0, 0);
}

static void print_add4(const char *label, pred4_add_fn fn)
{
    const int stride = 8;
    const int offset = 2 * stride + 2;
    uint8_t pix[8 * 8];
    int16_t block[16];
    init_fixture(pix, stride, 8);
    init_block(block, 16);
    fn(pix + offset, block, stride);
    print_block(label, pix, stride, offset, 4, 4);
    printf("%s_sum %d\n", label, sum_i16(block, 16));
}

static void print_pred8l_add(const char *label, pred8l_add_fn fn,
                             int has_topleft, int has_topright)
{
    const int stride = 28;
    const int offset = 5 * stride + 5;
    uint8_t pix[28 * 18];
    int16_t block[64];
    init_fixture(pix, stride, 18);
    init_block(block, 64);
    fn(pix + offset, block, has_topleft, has_topright, stride);
    print_block(label, pix, stride, offset, 8, 8);
    printf("%s_sum %d\n", label, sum_i16(block, 64));
}

static void print_pred8l_add_cases(const char *label, pred8l_add_fn fn)
{
    char name[64];
    snprintf(name, sizeof(name), "%s_11", label);
    print_pred8l_add(name, fn, 1, 1);
    snprintf(name, sizeof(name), "%s_00", label);
    print_pred8l_add(name, fn, 0, 0);
}

static void print_offset_add(const char *label, pred_offset_add_fn fn,
                             int width, int height, int block_count)
{
    const int stride = 24;
    const int base = 4 * stride + 4;
    uint8_t pix[24 * 24];
    int16_t block[16 * 16];
    int offset[48];
    init_fixture(pix, stride, 24);
    init_block(block, 16 * 16);
    init_offsets(offset, stride);
    fn(pix + base, offset, block, stride);
    print_block(label, pix, stride, base, width, height);
    printf("%s_sum %d\n", label, sum_i16(block, block_count * 16));
}

static void init_fixture_high(uint16_t *pix, int stride, int rows,
                              int bit_depth)
{
    const int mask = (1 << bit_depth) - 1;
    for (int y = 0; y < rows; y++)
        for (int x = 0; x < stride; x++)
            pix[y * stride + x] = (30 + x * 37 + y * 41) & mask;
}

static void init_topright_high(uint16_t *topright, int bit_depth)
{
    const int mask = (1 << bit_depth) - 1;
    topright[0] = 91 & mask;
    topright[1] = 257 & mask;
    topright[2] = 389 & mask;
    topright[3] = ((1 << bit_depth) - 7) & mask;
}

static void init_block_high(int32_t *block, int n)
{
    for (int i = 0; i < n; i++)
        block[i] = (i % 7) * 5 - 13;
}

static int sum_i32(const int32_t *block, int n)
{
    int sum = 0;
    for (int i = 0; i < n; i++)
        sum += block[i];
    return sum;
}

static void init_offsets_high(int *offset, int stride)
{
    const int base = scan8[0];
    for (int i = 0; i < 16; i++) {
        int delta = scan8[i] - base;
        int sample_offset = 4 * (delta & 7) + 4 * stride * (delta >> 3);
        offset[i] = sample_offset * (int)sizeof(uint16_t);
        offset[16 + i] = offset[i];
        offset[32 + i] = offset[i];
    }
}

static void print_block_high(const char *label, const uint16_t *pix,
                             int stride, int offset, int width, int height)
{
    printf("%s", label);
    for (int y = 0; y < height; y++)
        for (int x = 0; x < width; x++)
            printf(" %u", pix[offset + y * stride + x]);
    printf("\n");
}

static void print_pred16_high(const char *label, pred_fn fn, int bit_depth)
{
    const int stride = 24;
    const int offset = 4 * stride + 4;
    uint16_t pix[24 * 24];
    init_fixture_high(pix, stride, 24, bit_depth);
    fn((uint8_t *)(pix + offset), stride * (int)sizeof(uint16_t));
    print_block_high(label, pix, stride, offset, 16, 16);
}

static void print_pred8_high(const char *label, pred_fn fn, int bit_depth)
{
    const int stride = 16;
    const int offset = 4 * stride + 4;
    uint16_t pix[16 * 16];
    init_fixture_high(pix, stride, 16, bit_depth);
    fn((uint8_t *)(pix + offset), stride * (int)sizeof(uint16_t));
    print_block_high(label, pix, stride, offset, 8, 8);
}

static void print_pred8x16_high(const char *label, pred_fn fn, int bit_depth)
{
    const int stride = 16;
    const int offset = 4 * stride + 4;
    uint16_t pix[16 * 24];
    init_fixture_high(pix, stride, 24, bit_depth);
    fn((uint8_t *)(pix + offset), stride * (int)sizeof(uint16_t));
    print_block_high(label, pix, stride, offset, 8, 16);
}

static void print_pred4_high(const char *label, pred4_fn fn, int bit_depth)
{
    const int stride = 12;
    const int offset = 3 * stride + 3;
    uint16_t pix[12 * 12];
    uint16_t topright[4];
    init_fixture_high(pix, stride, 12, bit_depth);
    init_topright_high(topright, bit_depth);
    fn((uint8_t *)(pix + offset), (const uint8_t *)topright,
       stride * (int)sizeof(uint16_t));
    print_block_high(label, pix, stride, offset, 4, 4);
}

static void print_pred8l_high(const char *label, pred8l_fn fn,
                              int has_topleft, int has_topright,
                              int bit_depth)
{
    const int stride = 28;
    const int offset = 5 * stride + 5;
    uint16_t pix[28 * 18];
    init_fixture_high(pix, stride, 18, bit_depth);
    fn((uint8_t *)(pix + offset), has_topleft, has_topright,
       stride * (int)sizeof(uint16_t));
    print_block_high(label, pix, stride, offset, 8, 8);
}

static void print_pred8l_cases_high(const char *label, pred8l_fn fn,
                                    int bit_depth)
{
    char name[64];
    snprintf(name, sizeof(name), "%s_11", label);
    print_pred8l_high(name, fn, 1, 1, bit_depth);
    snprintf(name, sizeof(name), "%s_00", label);
    print_pred8l_high(name, fn, 0, 0, bit_depth);
}

static void print_add4_high(const char *label, pred4_add_fn fn,
                            int bit_depth)
{
    const int stride = 8;
    const int offset = 2 * stride + 2;
    uint16_t pix[8 * 8];
    int32_t block[16];
    init_fixture_high(pix, stride, 8, bit_depth);
    init_block_high(block, 16);
    fn((uint8_t *)(pix + offset), (int16_t *)block,
       stride * (int)sizeof(uint16_t));
    print_block_high(label, pix, stride, offset, 4, 4);
    printf("%s_sum %d\n", label, sum_i32(block, 16));
}

static void print_pred8l_add_high(const char *label, pred8l_add_fn fn,
                                  int has_topleft, int has_topright,
                                  int bit_depth)
{
    const int stride = 28;
    const int offset = 5 * stride + 5;
    uint16_t pix[28 * 18];
    int32_t block[64];
    init_fixture_high(pix, stride, 18, bit_depth);
    init_block_high(block, 64);
    fn((uint8_t *)(pix + offset), (int16_t *)block, has_topleft,
       has_topright, stride * (int)sizeof(uint16_t));
    print_block_high(label, pix, stride, offset, 8, 8);
    printf("%s_sum %d\n", label, sum_i32(block, 64));
}

static void print_pred8l_add_cases_high(const char *label, pred8l_add_fn fn,
                                        int bit_depth)
{
    char name[64];
    snprintf(name, sizeof(name), "%s_11", label);
    print_pred8l_add_high(name, fn, 1, 1, bit_depth);
    snprintf(name, sizeof(name), "%s_00", label);
    print_pred8l_add_high(name, fn, 0, 0, bit_depth);
}

static void print_offset_add_high(const char *label, pred_offset_add_fn fn,
                                  int width, int height, int block_count,
                                  int bit_depth)
{
    const int stride = 24;
    const int base = 4 * stride + 4;
    uint16_t pix[24 * 24];
    int32_t block[16 * 16];
    int offset[48];
    init_fixture_high(pix, stride, 24, bit_depth);
    init_block_high(block, 16 * 16);
    init_offsets_high(offset, stride);
    fn((uint8_t *)(pix + base), offset, (int16_t *)block,
       stride * (int)sizeof(uint16_t));
    print_block_high(label, pix, stride, base, width, height);
    printf("%s_sum %d\n", label, sum_i32(block, block_count * 16));
}

#define RUN_HIGH_PREDICTION(depth) do {                                           \
    print_pred4_high("pred4v_" #depth, pred4x4_vertical_ ## depth ## _c, depth);  \
    print_pred4_high("pred4h_" #depth, pred4x4_horizontal_ ## depth ## _c, depth);\
    print_pred4_high("pred4dc_" #depth, pred4x4_dc_ ## depth ## _c, depth);      \
    print_pred4_high("pred4ldc_" #depth, pred4x4_left_dc_ ## depth ## _c, depth);\
    print_pred4_high("pred4tdc_" #depth, pred4x4_top_dc_ ## depth ## _c, depth); \
    print_pred4_high("pred4dc128_" #depth, pred4x4_128_dc_ ## depth ## _c, depth);\
    print_pred4_high("pred4dr_" #depth, pred4x4_down_right_ ## depth ## _c, depth);\
    print_pred4_high("pred4dl_" #depth, pred4x4_down_left_ ## depth ## _c, depth);\
    print_pred4_high("pred4vr_" #depth, pred4x4_vertical_right_ ## depth ## _c, depth);\
    print_pred4_high("pred4vl_" #depth, pred4x4_vertical_left_ ## depth ## _c, depth);\
    print_pred4_high("pred4hu_" #depth, pred4x4_horizontal_up_ ## depth ## _c, depth);\
    print_pred4_high("pred4hd_" #depth, pred4x4_horizontal_down_ ## depth ## _c, depth);\
    print_pred16_high("pred16v_" #depth, pred16x16_vertical_ ## depth ## _c, depth);\
    print_pred16_high("pred16h_" #depth, pred16x16_horizontal_ ## depth ## _c, depth);\
    print_pred16_high("pred16dc_" #depth, pred16x16_dc_ ## depth ## _c, depth);  \
    print_pred16_high("pred16ldc_" #depth, pred16x16_left_dc_ ## depth ## _c, depth);\
    print_pred16_high("pred16tdc_" #depth, pred16x16_top_dc_ ## depth ## _c, depth);\
    print_pred16_high("pred16dc128_" #depth, pred16x16_128_dc_ ## depth ## _c, depth);\
    print_pred16_high("pred16plane_" #depth, pred16x16_plane_ ## depth ## _c, depth);\
    print_pred8_high("pred8v_" #depth, pred8x8_vertical_ ## depth ## _c, depth); \
    print_pred8_high("pred8h_" #depth, pred8x8_horizontal_ ## depth ## _c, depth);\
    print_pred8_high("pred8dc_" #depth, pred8x8_dc_ ## depth ## _c, depth);      \
    print_pred8_high("pred8ldc_" #depth, pred8x8_left_dc_ ## depth ## _c, depth);\
    print_pred8_high("pred8tdc_" #depth, pred8x8_top_dc_ ## depth ## _c, depth); \
    print_pred8_high("pred8dc128_" #depth, pred8x8_128_dc_ ## depth ## _c, depth);\
    print_pred8_high("pred8mc_l0t_" #depth, pred8x8_mad_cow_dc_l0t_ ## depth ## _c, depth);\
    print_pred8_high("pred8mc_0lt_" #depth, pred8x8_mad_cow_dc_0lt_ ## depth ## _c, depth);\
    print_pred8_high("pred8mc_l00_" #depth, pred8x8_mad_cow_dc_l00_ ## depth ## _c, depth);\
    print_pred8_high("pred8mc_0l0_" #depth, pred8x8_mad_cow_dc_0l0_ ## depth ## _c, depth);\
    print_pred8_high("pred8plane_" #depth, pred8x8_plane_ ## depth ## _c, depth);\
    print_pred8x16_high("pred8x16v_" #depth, pred8x16_vertical_ ## depth ## _c, depth);\
    print_pred8x16_high("pred8x16h_" #depth, pred8x16_horizontal_ ## depth ## _c, depth);\
    print_pred8x16_high("pred8x16dc_" #depth, pred8x16_dc_ ## depth ## _c, depth);\
    print_pred8x16_high("pred8x16ldc_" #depth, pred8x16_left_dc_ ## depth ## _c, depth);\
    print_pred8x16_high("pred8x16tdc_" #depth, pred8x16_top_dc_ ## depth ## _c, depth);\
    print_pred8x16_high("pred8x16dc128_" #depth, pred8x16_128_dc_ ## depth ## _c, depth);\
    print_pred8x16_high("pred8x16mc_l0t_" #depth, pred8x16_mad_cow_dc_l0t_ ## depth ## _c, depth);\
    print_pred8x16_high("pred8x16mc_0lt_" #depth, pred8x16_mad_cow_dc_0lt_ ## depth ## _c, depth);\
    print_pred8x16_high("pred8x16mc_l00_" #depth, pred8x16_mad_cow_dc_l00_ ## depth ## _c, depth);\
    print_pred8x16_high("pred8x16mc_0l0_" #depth, pred8x16_mad_cow_dc_0l0_ ## depth ## _c, depth);\
    print_pred8x16_high("pred8x16plane_" #depth, pred8x16_plane_ ## depth ## _c, depth);\
    print_pred8l_cases_high("pred8ldc128_" #depth, pred8x8l_128_dc_ ## depth ## _c, depth);\
    print_pred8l_cases_high("pred8lldc_" #depth, pred8x8l_left_dc_ ## depth ## _c, depth);\
    print_pred8l_cases_high("pred8ltdc_" #depth, pred8x8l_top_dc_ ## depth ## _c, depth);\
    print_pred8l_cases_high("pred8ldc_" #depth, pred8x8l_dc_ ## depth ## _c, depth);\
    print_pred8l_cases_high("pred8lh_" #depth, pred8x8l_horizontal_ ## depth ## _c, depth);\
    print_pred8l_cases_high("pred8lv_" #depth, pred8x8l_vertical_ ## depth ## _c, depth);\
    print_pred8l_cases_high("pred8ldl_" #depth, pred8x8l_down_left_ ## depth ## _c, depth);\
    print_pred8l_cases_high("pred8ldr_" #depth, pred8x8l_down_right_ ## depth ## _c, depth);\
    print_pred8l_cases_high("pred8lvr_" #depth, pred8x8l_vertical_right_ ## depth ## _c, depth);\
    print_pred8l_cases_high("pred8lhd_" #depth, pred8x8l_horizontal_down_ ## depth ## _c, depth);\
    print_pred8l_cases_high("pred8lvl_" #depth, pred8x8l_vertical_left_ ## depth ## _c, depth);\
    print_pred8l_cases_high("pred8lhu_" #depth, pred8x8l_horizontal_up_ ## depth ## _c, depth);\
    print_add4_high("add4v_" #depth, pred4x4_vertical_add_ ## depth ## _c, depth);\
    print_add4_high("add4h_" #depth, pred4x4_horizontal_add_ ## depth ## _c, depth);\
    print_pred8l_add_cases_high("add8lfv_" #depth, pred8x8l_vertical_filter_add_ ## depth ## _c, depth);\
    print_pred8l_add_cases_high("add8lfh_" #depth, pred8x8l_horizontal_filter_add_ ## depth ## _c, depth);\
    print_offset_add_high("add16v_" #depth, pred16x16_vertical_add_ ## depth ## _c, 16, 16, 16, depth);\
    print_offset_add_high("add16h_" #depth, pred16x16_horizontal_add_ ## depth ## _c, 16, 16, 16, depth);\
    print_offset_add_high("add8v_" #depth, pred8x8_vertical_add_ ## depth ## _c, 8, 8, 4, depth);\
    print_offset_add_high("add8h_" #depth, pred8x8_horizontal_add_ ## depth ## _c, 8, 8, 4, depth);\
    print_offset_add_high("add8x16v_" #depth, pred8x16_vertical_add_ ## depth ## _c, 8, 16, 8, depth);\
    print_offset_add_high("add8x16h_" #depth, pred8x16_horizontal_add_ ## depth ## _c, 8, 16, 8, depth);\
} while (0)

int main(void)
{
    print_pred4("pred4v", pred4x4_vertical_8_c);
    print_pred4("pred4h", pred4x4_horizontal_8_c);
    print_pred4("pred4dc", pred4x4_dc_8_c);
    print_pred4("pred4ldc", pred4x4_left_dc_8_c);
    print_pred4("pred4tdc", pred4x4_top_dc_8_c);
    print_pred4("pred4dc128", pred4x4_128_dc_8_c);
    print_pred4("pred4dr", pred4x4_down_right_8_c);
    print_pred4("pred4dl", pred4x4_down_left_8_c);
    print_pred4("pred4vr", pred4x4_vertical_right_8_c);
    print_pred4("pred4vl", pred4x4_vertical_left_8_c);
    print_pred4("pred4hu", pred4x4_horizontal_up_8_c);
    print_pred4("pred4hd", pred4x4_horizontal_down_8_c);

    print_pred16("pred16v", pred16x16_vertical_8_c);
    print_pred16("pred16h", pred16x16_horizontal_8_c);
    print_pred16("pred16dc", pred16x16_dc_8_c);
    print_pred16("pred16ldc", pred16x16_left_dc_8_c);
    print_pred16("pred16tdc", pred16x16_top_dc_8_c);
    print_pred16("pred16dc128", pred16x16_128_dc_8_c);
    print_pred16("pred16dc127", pred16x16_127_dc_8_c);
    print_pred16("pred16dc129", pred16x16_129_dc_8_c);
    print_pred16("pred16plane", pred16x16_plane_8_c);

    print_pred8("pred8v", pred8x8_vertical_8_c);
    print_pred8("pred8h", pred8x8_horizontal_8_c);
    print_pred8("pred8dc", pred8x8_dc_8_c);
    print_pred8("pred8ldc", pred8x8_left_dc_8_c);
    print_pred8("pred8tdc", pred8x8_top_dc_8_c);
    print_pred8("pred8dc128", pred8x8_128_dc_8_c);
    print_pred8("pred8dc127", pred8x8_127_dc_8_c);
    print_pred8("pred8dc129", pred8x8_129_dc_8_c);
    print_pred8("pred8mc_l0t", pred8x8_mad_cow_dc_l0t_8_c);
    print_pred8("pred8mc_0lt", pred8x8_mad_cow_dc_0lt_8_c);
    print_pred8("pred8mc_l00", pred8x8_mad_cow_dc_l00_8_c);
    print_pred8("pred8mc_0l0", pred8x8_mad_cow_dc_0l0_8_c);
    print_pred8("pred8plane", pred8x8_plane_8_c);

    print_pred8x16("pred8x16v", pred8x16_vertical_8_c);
    print_pred8x16("pred8x16h", pred8x16_horizontal_8_c);
    print_pred8x16("pred8x16dc", pred8x16_dc_8_c);
    print_pred8x16("pred8x16ldc", pred8x16_left_dc_8_c);
    print_pred8x16("pred8x16tdc", pred8x16_top_dc_8_c);
    print_pred8x16("pred8x16dc128", pred8x16_128_dc_8_c);
    print_pred8x16("pred8x16mc_l0t", pred8x16_mad_cow_dc_l0t_8_c);
    print_pred8x16("pred8x16mc_0lt", pred8x16_mad_cow_dc_0lt_8_c);
    print_pred8x16("pred8x16mc_l00", pred8x16_mad_cow_dc_l00_8_c);
    print_pred8x16("pred8x16mc_0l0", pred8x16_mad_cow_dc_0l0_8_c);
    print_pred8x16("pred8x16plane", pred8x16_plane_8_c);

    print_pred8l_cases("pred8ldc128", pred8x8l_128_dc_8_c);
    print_pred8l_cases("pred8lldc", pred8x8l_left_dc_8_c);
    print_pred8l_cases("pred8ltdc", pred8x8l_top_dc_8_c);
    print_pred8l_cases("pred8ldc", pred8x8l_dc_8_c);
    print_pred8l_cases("pred8lh", pred8x8l_horizontal_8_c);
    print_pred8l_cases("pred8lv", pred8x8l_vertical_8_c);
    print_pred8l_cases("pred8ldl", pred8x8l_down_left_8_c);
    print_pred8l_cases("pred8ldr", pred8x8l_down_right_8_c);
    print_pred8l_cases("pred8lvr", pred8x8l_vertical_right_8_c);
    print_pred8l_cases("pred8lhd", pred8x8l_horizontal_down_8_c);
    print_pred8l_cases("pred8lvl", pred8x8l_vertical_left_8_c);
    print_pred8l_cases("pred8lhu", pred8x8l_horizontal_up_8_c);

    print_add4("add4v", pred4x4_vertical_add_8_c);
    print_add4("add4h", pred4x4_horizontal_add_8_c);
    print_pred8l_add_cases("add8lfv", pred8x8l_vertical_filter_add_8_c);
    print_pred8l_add_cases("add8lfh", pred8x8l_horizontal_filter_add_8_c);
    print_offset_add("add16v", pred16x16_vertical_add_8_c, 16, 16, 16);
    print_offset_add("add16h", pred16x16_horizontal_add_8_c, 16, 16, 16);
    print_offset_add("add8v", pred8x8_vertical_add_8_c, 8, 8, 4);
    print_offset_add("add8h", pred8x8_horizontal_add_8_c, 8, 8, 4);
    print_offset_add("add8x16v", pred8x16_vertical_add_8_c, 8, 16, 8);
    print_offset_add("add8x16h", pred8x16_horizontal_add_8_c, 8, 16, 8);
    RUN_HIGH_PREDICTION(9);
    RUN_HIGH_PREDICTION(10);
    RUN_HIGH_PREDICTION(12);
    RUN_HIGH_PREDICTION(14);
    return 0;
}
`

const predOracleBitDepthTemplate = `
#include <stdint.h>

#ifndef GOH264_PRED_BITDEPTH_HELPERS
#define GOH264_PRED_BITDEPTH_HELPERS
static inline uint8_t goh264_pred_clip_uint8(int v)
{
    if (v < 0)
        return 0;
    if (v > 255)
        return 255;
    return (uint8_t)v;
}

static inline uint32_t goh264_pred_rn4pa(const void *p)
{
    const uint8_t *b = (const uint8_t *)p;
    return (uint32_t)b[0] | ((uint32_t)b[1] << 8) |
           ((uint32_t)b[2] << 16) | ((uint32_t)b[3] << 24);
}

static inline void goh264_pred_wn4pa(void *p, uint32_t v)
{
    uint8_t *b = (uint8_t *)p;
    b[0] = v;
    b[1] = v >> 8;
    b[2] = v >> 16;
    b[3] = v >> 24;
}

static inline uint16_t goh264_pred_clip_uintp2(int v, int p)
{
    int max = (1 << p) - 1;
    if (v < 0)
        return 0;
    if (v > max)
        return (uint16_t)max;
    return (uint16_t)v;
}

static inline uint64_t goh264_pred_rn8pa(const void *p)
{
    const uint8_t *b = (const uint8_t *)p;
    return (uint64_t)b[0] | ((uint64_t)b[1] << 8) |
           ((uint64_t)b[2] << 16) | ((uint64_t)b[3] << 24) |
           ((uint64_t)b[4] << 32) | ((uint64_t)b[5] << 40) |
           ((uint64_t)b[6] << 48) | ((uint64_t)b[7] << 56);
}

static inline void goh264_pred_wn8pa(void *p, uint64_t v)
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
#endif

#undef pixel
#undef pixel2
#undef pixel4
#undef dctcoef
#undef av_unused
#undef FUNC3
#undef FUNC2
#undef FUNC
#undef FUNCC
#undef AV_RN4PA
#undef AV_WN4PA
#undef PIXEL_SPLAT_X4
#undef CLIP

#define av_unused
#if BIT_DEPTH > 8
#define pixel uint16_t
#define pixel2 uint32_t
#define pixel4 uint64_t
#define dctcoef int32_t
#define FUNC3(a, b, c)  a ## _ ## b ## c
#define FUNC2(a, b, c)  FUNC3(a, b, c)
#define FUNC(a) FUNC2(a, BIT_DEPTH, _c)
#define FUNCC(a) FUNC2(a, BIT_DEPTH, _c)
#define AV_RN4PA(p) goh264_pred_rn8pa(p)
#define AV_WN4PA(p, v) goh264_pred_wn8pa((void *)(p), (uint64_t)(v))
#define PIXEL_SPLAT_X4(x) ((uint64_t)(uint16_t)(x) * 0x0001000100010001ULL)
#define CLIP(a) goh264_pred_clip_uintp2((a), BIT_DEPTH)
#else
#define pixel uint8_t
#define pixel2 uint16_t
#define pixel4 uint32_t
#define dctcoef int16_t
#define FUNC3(a, b, c)  a ## _ ## b ## c
#define FUNC2(a, b, c)  FUNC3(a, b, c)
#define FUNC(a) FUNC2(a, BIT_DEPTH, _c)
#define FUNCC(a) FUNC2(a, BIT_DEPTH, _c)
#define AV_RN4PA(p) goh264_pred_rn4pa(p)
#define AV_WN4PA(p, v) goh264_pred_wn4pa((void *)(p), (uint32_t)(v))
#define PIXEL_SPLAT_X4(x) ((uint32_t)(uint8_t)(x) * 0x01010101U)
#define CLIP(a) goh264_pred_clip_uint8(a)
#endif
`

func TestH264PredictionUpstreamOracle(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run pinned FFmpeg H.264 prediction oracle")
	}
	cc, err := exec.LookPath("cc")
	if err != nil {
		t.Skip("cc not available")
	}

	root := h264RepoRoot(t)
	upstreamTemplate := filepath.Join(root, ".upstream", "ffmpeg-n8.0.1", "libavcodec", "h264pred_template.c")
	template, err := os.ReadFile(upstreamTemplate)
	if err != nil {
		t.Skipf("pinned upstream H.264 prediction source not available: %v", err)
	}

	dir := t.TempDir()
	writeOracleFile(t, filepath.Join(dir, "oracle.c"), predOracleC)
	writeOracleFile(t, filepath.Join(dir, "h264pred_template.c"), string(template))
	writeOracleFile(t, filepath.Join(dir, "bit_depth_template.c"), predOracleBitDepthTemplate)
	writeOracleFile(t, filepath.Join(dir, "mathops.h"), "")
	if err := os.Mkdir(filepath.Join(dir, "libavutil"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeOracleFile(t, filepath.Join(dir, "libavutil", "intreadwrite.h"), "")

	bin := filepath.Join(dir, "oracle")
	cmd := exec.Command(cc, "-std=c99", "-I"+dir, filepath.Join(dir, "oracle.c"), "-o", bin)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("compile H.264 prediction oracle: %v\n%s", err, out)
	}

	out, err := exec.Command(bin).CombinedOutput()
	if err != nil {
		t.Fatalf("run H.264 prediction oracle: %v\n%s", err, out)
	}

	got := strings.TrimSpace(string(out))
	want := strings.TrimSpace(h264PredictionOracleWant(t))
	if got != want {
		t.Fatalf("H.264 prediction oracle mismatch\nC oracle:\n%s\nGo:\n%s", got, want)
	}
}

func h264PredictionOracleWant(t *testing.T) string {
	t.Helper()
	var b strings.Builder
	for _, c := range []struct {
		label string
		fn    h264Pred4Func
	}{
		{"pred4v", func(pix []uint8, offset int, stride int, topRight []uint8) error {
			return h264Pred4x4Vertical(pix, offset, stride)
		}},
		{"pred4h", func(pix []uint8, offset int, stride int, topRight []uint8) error {
			return h264Pred4x4Horizontal(pix, offset, stride)
		}},
		{"pred4dc", func(pix []uint8, offset int, stride int, topRight []uint8) error {
			return h264Pred4x4DC(pix, offset, stride)
		}},
		{"pred4ldc", func(pix []uint8, offset int, stride int, topRight []uint8) error {
			return h264Pred4x4LeftDC(pix, offset, stride)
		}},
		{"pred4tdc", func(pix []uint8, offset int, stride int, topRight []uint8) error {
			return h264Pred4x4TopDC(pix, offset, stride)
		}},
		{"pred4dc128", func(pix []uint8, offset int, stride int, topRight []uint8) error {
			return h264Pred4x4DC128(pix, offset, stride)
		}},
		{"pred4dr", func(pix []uint8, offset int, stride int, topRight []uint8) error {
			return h264Pred4x4DownRight(pix, offset, stride)
		}},
		{"pred4dl", h264Pred4x4DownLeft},
		{"pred4vr", func(pix []uint8, offset int, stride int, topRight []uint8) error {
			return h264Pred4x4VerticalRight(pix, offset, stride)
		}},
		{"pred4vl", h264Pred4x4VerticalLeft},
		{"pred4hu", func(pix []uint8, offset int, stride int, topRight []uint8) error {
			return h264Pred4x4HorizontalUp(pix, offset, stride)
		}},
		{"pred4hd", func(pix []uint8, offset int, stride int, topRight []uint8) error {
			return h264Pred4x4HorizontalDown(pix, offset, stride)
		}},
	} {
		printPred4OracleWant(t, &b, c.label, c.fn)
	}
	for _, c := range []struct {
		label string
		fn    h264PredFunc
	}{
		{"pred16v", h264Pred16x16Vertical},
		{"pred16h", h264Pred16x16Horizontal},
		{"pred16dc", h264Pred16x16DC},
		{"pred16ldc", h264Pred16x16LeftDC},
		{"pred16tdc", h264Pred16x16TopDC},
		{"pred16dc128", h264Pred16x16DC128},
		{"pred16dc127", h264Pred16x16DC127},
		{"pred16dc129", h264Pred16x16DC129},
		{"pred16plane", h264Pred16x16Plane},
	} {
		printPredOracleWant(t, &b, c.label, c.fn, 24, 24, 4*24+4, 16, 16)
	}
	for _, c := range []struct {
		label string
		fn    h264PredFunc
	}{
		{"pred8v", h264Pred8x8Vertical},
		{"pred8h", h264Pred8x8Horizontal},
		{"pred8dc", h264Pred8x8DC},
		{"pred8ldc", h264Pred8x8LeftDC},
		{"pred8tdc", h264Pred8x8TopDC},
		{"pred8dc128", h264Pred8x8DC128},
		{"pred8dc127", h264Pred8x8DC127},
		{"pred8dc129", h264Pred8x8DC129},
		{"pred8mc_l0t", h264Pred8x8MadCowDCL0T},
		{"pred8mc_0lt", h264Pred8x8MadCowDC0LT},
		{"pred8mc_l00", h264Pred8x8MadCowDCL00},
		{"pred8mc_0l0", h264Pred8x8MadCowDC0L0},
		{"pred8plane", h264Pred8x8Plane},
	} {
		printPredOracleWant(t, &b, c.label, c.fn, 16, 16, 4*16+4, 8, 8)
	}
	for _, c := range []struct {
		label string
		fn    h264PredFunc
	}{
		{"pred8x16v", h264Pred8x16Vertical},
		{"pred8x16h", h264Pred8x16Horizontal},
		{"pred8x16dc", h264Pred8x16DC},
		{"pred8x16ldc", h264Pred8x16LeftDC},
		{"pred8x16tdc", h264Pred8x16TopDC},
		{"pred8x16dc128", h264Pred8x16DC128},
		{"pred8x16mc_l0t", h264Pred8x16MadCowDCL0T},
		{"pred8x16mc_0lt", h264Pred8x16MadCowDC0LT},
		{"pred8x16mc_l00", h264Pred8x16MadCowDCL00},
		{"pred8x16mc_0l0", h264Pred8x16MadCowDC0L0},
		{"pred8x16plane", h264Pred8x16Plane},
	} {
		printPredOracleWant(t, &b, c.label, c.fn, 16, 24, 4*16+4, 8, 16)
	}
	for _, c := range []struct {
		label string
		fn    h264Pred8LFunc
	}{
		{"pred8ldc128", h264Pred8x8LDC128},
		{"pred8lldc", h264Pred8x8LLeftDC},
		{"pred8ltdc", h264Pred8x8LTopDC},
		{"pred8ldc", h264Pred8x8LDC},
		{"pred8lh", h264Pred8x8LHorizontal},
		{"pred8lv", h264Pred8x8LVertical},
		{"pred8ldl", h264Pred8x8LDownLeft},
		{"pred8ldr", h264Pred8x8LDownRight},
		{"pred8lvr", h264Pred8x8LVerticalRight},
		{"pred8lhd", h264Pred8x8LHorizontalDown},
		{"pred8lvl", h264Pred8x8LVerticalLeft},
		{"pred8lhu", h264Pred8x8LHorizontalUp},
	} {
		printPred8LCasesOracleWant(t, &b, c.label, c.fn)
	}
	printPred4AddOracleWant(t, &b, "add4v", h264Pred4x4VerticalAdd)
	printPred4AddOracleWant(t, &b, "add4h", h264Pred4x4HorizontalAdd)
	printPred8LAddCasesOracleWant(t, &b, "add8lfv", h264Pred8x8LVerticalFilterAdd)
	printPred8LAddCasesOracleWant(t, &b, "add8lfh", h264Pred8x8LHorizontalFilterAdd)
	printPredOffsetAddOracleWant(t, &b, "add16v", h264Pred16x16VerticalAdd, 16, 16, 16)
	printPredOffsetAddOracleWant(t, &b, "add16h", h264Pred16x16HorizontalAdd, 16, 16, 16)
	printPredOffsetAddOracleWant(t, &b, "add8v", h264Pred8x8VerticalAdd, 8, 8, 4)
	printPredOffsetAddOracleWant(t, &b, "add8h", h264Pred8x8HorizontalAdd, 8, 8, 4)
	printPredOffsetAddOracleWant(t, &b, "add8x16v", h264Pred8x16VerticalAdd, 8, 16, 8)
	printPredOffsetAddOracleWant(t, &b, "add8x16h", h264Pred8x16HorizontalAdd, 8, 16, 8)
	appendH264PredictionHighOracleWant(t, &b)
	return b.String()
}

type h264PredFunc func([]uint8, int, int) error
type h264Pred4Func func([]uint8, int, int, []uint8) error
type h264Pred8LFunc func([]uint8, int, int, bool, bool) error
type h264Pred4AddFunc func([]uint8, int, []int32, int) error
type h264Pred8LAddFunc func([]uint8, int, []int32, int, bool, bool) error
type h264PredOffsetAddFunc func([]uint8, *[48]int, []int32, int) error
type h264PredFuncHigh func([]uint16, int, int, int) error
type h264Pred4FuncHigh func([]uint16, int, int, []uint16, int) error
type h264Pred8LFuncHigh func([]uint16, int, int, bool, bool, int) error
type h264Pred4AddFuncHigh func([]uint16, int, []int32, int, int) error
type h264Pred8LAddFuncHigh func([]uint16, int, []int32, int, bool, bool, int) error
type h264PredOffsetAddFuncHigh func([]uint16, *[48]int, []int32, int, int) error

func appendH264PredictionHighOracleWant(t *testing.T, b *strings.Builder) {
	t.Helper()
	for _, bitDepth := range []int{9, 10, 12, 14} {
		for _, c := range []struct {
			label string
			fn    h264Pred4FuncHigh
		}{
			{"pred4v", func(pix []uint16, offset int, stride int, topRight []uint16, bitDepth int) error {
				return h264Pred4x4VerticalHigh(pix, offset, stride, bitDepth)
			}},
			{"pred4h", func(pix []uint16, offset int, stride int, topRight []uint16, bitDepth int) error {
				return h264Pred4x4HorizontalHigh(pix, offset, stride, bitDepth)
			}},
			{"pred4dc", func(pix []uint16, offset int, stride int, topRight []uint16, bitDepth int) error {
				return h264Pred4x4DCHigh(pix, offset, stride, bitDepth)
			}},
			{"pred4ldc", func(pix []uint16, offset int, stride int, topRight []uint16, bitDepth int) error {
				return h264Pred4x4LeftDCHigh(pix, offset, stride, bitDepth)
			}},
			{"pred4tdc", func(pix []uint16, offset int, stride int, topRight []uint16, bitDepth int) error {
				return h264Pred4x4TopDCHigh(pix, offset, stride, bitDepth)
			}},
			{"pred4dc128", func(pix []uint16, offset int, stride int, topRight []uint16, bitDepth int) error {
				return h264Pred4x4DC128High(pix, offset, stride, bitDepth)
			}},
			{"pred4dr", func(pix []uint16, offset int, stride int, topRight []uint16, bitDepth int) error {
				return h264Pred4x4DownRightHigh(pix, offset, stride, bitDepth)
			}},
			{"pred4dl", h264Pred4x4DownLeftHigh},
			{"pred4vr", func(pix []uint16, offset int, stride int, topRight []uint16, bitDepth int) error {
				return h264Pred4x4VerticalRightHigh(pix, offset, stride, bitDepth)
			}},
			{"pred4vl", h264Pred4x4VerticalLeftHigh},
			{"pred4hu", func(pix []uint16, offset int, stride int, topRight []uint16, bitDepth int) error {
				return h264Pred4x4HorizontalUpHigh(pix, offset, stride, bitDepth)
			}},
			{"pred4hd", func(pix []uint16, offset int, stride int, topRight []uint16, bitDepth int) error {
				return h264Pred4x4HorizontalDownHigh(pix, offset, stride, bitDepth)
			}},
		} {
			printPred4OracleWantHigh(t, b, h264PredHighLabel(c.label, bitDepth), c.fn, bitDepth)
		}
		for _, c := range []struct {
			label string
			fn    h264PredFuncHigh
		}{
			{"pred16v", h264Pred16x16VerticalHigh},
			{"pred16h", h264Pred16x16HorizontalHigh},
			{"pred16dc", h264Pred16x16DCHigh},
			{"pred16ldc", h264Pred16x16LeftDCHigh},
			{"pred16tdc", h264Pred16x16TopDCHigh},
			{"pred16dc128", h264Pred16x16DC128High},
			{"pred16plane", h264Pred16x16PlaneHigh},
		} {
			printPredOracleWantHigh(t, b, h264PredHighLabel(c.label, bitDepth), c.fn, 24, 24, 4*24+4, 16, 16, bitDepth)
		}
		for _, c := range []struct {
			label string
			fn    h264PredFuncHigh
		}{
			{"pred8v", h264Pred8x8VerticalHigh},
			{"pred8h", h264Pred8x8HorizontalHigh},
			{"pred8dc", h264Pred8x8DCHigh},
			{"pred8ldc", h264Pred8x8LeftDCHigh},
			{"pred8tdc", h264Pred8x8TopDCHigh},
			{"pred8dc128", h264Pred8x8DC128High},
			{"pred8mc_l0t", h264Pred8x8MadCowDCL0THigh},
			{"pred8mc_0lt", h264Pred8x8MadCowDC0LTHigh},
			{"pred8mc_l00", h264Pred8x8MadCowDCL00High},
			{"pred8mc_0l0", h264Pred8x8MadCowDC0L0High},
			{"pred8plane", h264Pred8x8PlaneHigh},
		} {
			printPredOracleWantHigh(t, b, h264PredHighLabel(c.label, bitDepth), c.fn, 16, 16, 4*16+4, 8, 8, bitDepth)
		}
		for _, c := range []struct {
			label string
			fn    h264PredFuncHigh
		}{
			{"pred8x16v", h264Pred8x16VerticalHigh},
			{"pred8x16h", h264Pred8x16HorizontalHigh},
			{"pred8x16dc", h264Pred8x16DCHigh},
			{"pred8x16ldc", h264Pred8x16LeftDCHigh},
			{"pred8x16tdc", h264Pred8x16TopDCHigh},
			{"pred8x16dc128", h264Pred8x16DC128High},
			{"pred8x16mc_l0t", h264Pred8x16MadCowDCL0THigh},
			{"pred8x16mc_0lt", h264Pred8x16MadCowDC0LTHigh},
			{"pred8x16mc_l00", h264Pred8x16MadCowDCL00High},
			{"pred8x16mc_0l0", h264Pred8x16MadCowDC0L0High},
			{"pred8x16plane", h264Pred8x16PlaneHigh},
		} {
			printPredOracleWantHigh(t, b, h264PredHighLabel(c.label, bitDepth), c.fn, 16, 24, 4*16+4, 8, 16, bitDepth)
		}
		for _, c := range []struct {
			label string
			fn    h264Pred8LFuncHigh
		}{
			{"pred8ldc128", h264Pred8x8LDC128High},
			{"pred8lldc", h264Pred8x8LLeftDCHigh},
			{"pred8ltdc", h264Pred8x8LTopDCHigh},
			{"pred8ldc", h264Pred8x8LDCHigh},
			{"pred8lh", h264Pred8x8LHorizontalHigh},
			{"pred8lv", h264Pred8x8LVerticalHigh},
			{"pred8ldl", h264Pred8x8LDownLeftHigh},
			{"pred8ldr", h264Pred8x8LDownRightHigh},
			{"pred8lvr", h264Pred8x8LVerticalRightHigh},
			{"pred8lhd", h264Pred8x8LHorizontalDownHigh},
			{"pred8lvl", h264Pred8x8LVerticalLeftHigh},
			{"pred8lhu", h264Pred8x8LHorizontalUpHigh},
		} {
			printPred8LCasesOracleWantHigh(t, b, h264PredHighLabel(c.label, bitDepth), c.fn, bitDepth)
		}
		printPred4AddOracleWantHigh(t, b, h264PredHighLabel("add4v", bitDepth), h264Pred4x4VerticalAddHigh, bitDepth)
		printPred4AddOracleWantHigh(t, b, h264PredHighLabel("add4h", bitDepth), h264Pred4x4HorizontalAddHigh, bitDepth)
		printPred8LAddCasesOracleWantHigh(t, b, h264PredHighLabel("add8lfv", bitDepth), h264Pred8x8LVerticalFilterAddHigh, bitDepth)
		printPred8LAddCasesOracleWantHigh(t, b, h264PredHighLabel("add8lfh", bitDepth), h264Pred8x8LHorizontalFilterAddHigh, bitDepth)
		printPredOffsetAddOracleWantHigh(t, b, h264PredHighLabel("add16v", bitDepth), h264Pred16x16VerticalAddHigh, 16, 16, 16, bitDepth)
		printPredOffsetAddOracleWantHigh(t, b, h264PredHighLabel("add16h", bitDepth), h264Pred16x16HorizontalAddHigh, 16, 16, 16, bitDepth)
		printPredOffsetAddOracleWantHigh(t, b, h264PredHighLabel("add8v", bitDepth), h264Pred8x8VerticalAddHigh, 8, 8, 4, bitDepth)
		printPredOffsetAddOracleWantHigh(t, b, h264PredHighLabel("add8h", bitDepth), h264Pred8x8HorizontalAddHigh, 8, 8, 4, bitDepth)
		printPredOffsetAddOracleWantHigh(t, b, h264PredHighLabel("add8x16v", bitDepth), h264Pred8x16VerticalAddHigh, 8, 16, 8, bitDepth)
		printPredOffsetAddOracleWantHigh(t, b, h264PredHighLabel("add8x16h", bitDepth), h264Pred8x16HorizontalAddHigh, 8, 16, 8, bitDepth)
	}
}

func h264PredHighLabel(label string, bitDepth int) string {
	return fmt.Sprintf("%s_%d", label, bitDepth)
}

func printPredOracleWant(t *testing.T, b *strings.Builder, label string, fn h264PredFunc, stride int, rows int, offset int, width int, height int) {
	t.Helper()
	pix := makePredictionFixture(stride, rows)
	if err := fn(pix, offset, stride); err != nil {
		t.Fatal(err)
	}
	printPredBlock(b, label, pix, stride, offset, width, height)
}

func printPred4OracleWant(t *testing.T, b *strings.Builder, label string, fn h264Pred4Func) {
	t.Helper()
	const stride = 12
	const offset = 3*stride + 3
	pix := makePredictionFixture(stride, 12)
	topRight := []uint8{91, 123, 155, 177}
	if err := fn(pix, offset, stride, topRight); err != nil {
		t.Fatal(err)
	}
	printPredBlock(b, label, pix, stride, offset, 4, 4)
}

func printPred8LCasesOracleWant(t *testing.T, b *strings.Builder, label string, fn h264Pred8LFunc) {
	t.Helper()
	printPred8LOracleWant(t, b, label+"_11", fn, true, true)
	printPred8LOracleWant(t, b, label+"_00", fn, false, false)
}

func printPred8LOracleWant(t *testing.T, b *strings.Builder, label string, fn h264Pred8LFunc, hasTopLeft bool, hasTopRight bool) {
	t.Helper()
	const stride = 28
	const offset = 5*stride + 5
	pix := makePredictionFixture(stride, 18)
	if err := fn(pix, offset, stride, hasTopLeft, hasTopRight); err != nil {
		t.Fatal(err)
	}
	printPredBlock(b, label, pix, stride, offset, 8, 8)
}

func printPred4AddOracleWant(t *testing.T, b *strings.Builder, label string, fn h264Pred4AddFunc) {
	t.Helper()
	const stride = 8
	const offset = 2*stride + 2
	pix := makePredictionFixture(stride, 8)
	block := makePredictionBlock(16)
	if err := fn(pix, offset, block, stride); err != nil {
		t.Fatal(err)
	}
	printPredBlock(b, label, pix, stride, offset, 4, 4)
	fmt.Fprintf(b, "%s_sum %d\n", label, sumInt32(block))
}

func printPred8LAddCasesOracleWant(t *testing.T, b *strings.Builder, label string, fn h264Pred8LAddFunc) {
	t.Helper()
	printPred8LAddOracleWant(t, b, label+"_11", fn, true, true)
	printPred8LAddOracleWant(t, b, label+"_00", fn, false, false)
}

func printPred8LAddOracleWant(t *testing.T, b *strings.Builder, label string, fn h264Pred8LAddFunc, hasTopLeft bool, hasTopRight bool) {
	t.Helper()
	const stride = 28
	const offset = 5*stride + 5
	pix := makePredictionFixture(stride, 18)
	block := makePredictionBlock(64)
	if err := fn(pix, offset, block, stride, hasTopLeft, hasTopRight); err != nil {
		t.Fatal(err)
	}
	printPredBlock(b, label, pix, stride, offset, 8, 8)
	fmt.Fprintf(b, "%s_sum %d\n", label, sumInt32(block))
}

func printPredOffsetAddOracleWant(t *testing.T, b *strings.Builder, label string, fn h264PredOffsetAddFunc, width int, height int, blockCount int) {
	t.Helper()
	const stride = 24
	const base = 4*stride + 4
	pix := makePredictionFixture(stride, 24)
	block := makePredictionBlock(16 * 16)
	offsets, err := h264FrameBlockOffsets(stride, stride, 0)
	if err != nil {
		t.Fatal(err)
	}
	for i := range offsets {
		offsets[i] += base
	}
	if err := fn(pix, &offsets, block, stride); err != nil {
		t.Fatal(err)
	}
	printPredBlock(b, label, pix, stride, base, width, height)
	fmt.Fprintf(b, "%s_sum %d\n", label, sumInt32(block[:blockCount*16]))
}

func makePredictionBlock(n int) []int32 {
	block := make([]int32, n)
	for i := range block {
		block[i] = int32((i%7)*3 - 9)
	}
	return block
}

func printPredBlock(b *strings.Builder, label string, pix []uint8, stride int, offset int, width int, height int) {
	fmt.Fprint(b, label)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			fmt.Fprintf(b, " %d", pix[offset+y*stride+x])
		}
	}
	fmt.Fprint(b, "\n")
}

func printPredOracleWantHigh(t *testing.T, b *strings.Builder, label string, fn h264PredFuncHigh, stride int, rows int, offset int, width int, height int, bitDepth int) {
	t.Helper()
	pix := makePredictionFixtureHigh(stride, rows, bitDepth)
	if err := fn(pix, offset, stride, bitDepth); err != nil {
		t.Fatal(err)
	}
	printPredBlockHigh(b, label, pix, stride, offset, width, height)
}

func printPred4OracleWantHigh(t *testing.T, b *strings.Builder, label string, fn h264Pred4FuncHigh, bitDepth int) {
	t.Helper()
	const stride = 12
	const offset = 3*stride + 3
	pix := makePredictionFixtureHigh(stride, 12, bitDepth)
	topRight := makePredictionTopRightHigh(bitDepth)
	if err := fn(pix, offset, stride, topRight, bitDepth); err != nil {
		t.Fatal(err)
	}
	printPredBlockHigh(b, label, pix, stride, offset, 4, 4)
}

func printPred8LCasesOracleWantHigh(t *testing.T, b *strings.Builder, label string, fn h264Pred8LFuncHigh, bitDepth int) {
	t.Helper()
	printPred8LOracleWantHigh(t, b, label+"_11", fn, true, true, bitDepth)
	printPred8LOracleWantHigh(t, b, label+"_00", fn, false, false, bitDepth)
}

func printPred8LOracleWantHigh(t *testing.T, b *strings.Builder, label string, fn h264Pred8LFuncHigh, hasTopLeft bool, hasTopRight bool, bitDepth int) {
	t.Helper()
	const stride = 28
	const offset = 5*stride + 5
	pix := makePredictionFixtureHigh(stride, 18, bitDepth)
	if err := fn(pix, offset, stride, hasTopLeft, hasTopRight, bitDepth); err != nil {
		t.Fatal(err)
	}
	printPredBlockHigh(b, label, pix, stride, offset, 8, 8)
}

func printPred4AddOracleWantHigh(t *testing.T, b *strings.Builder, label string, fn h264Pred4AddFuncHigh, bitDepth int) {
	t.Helper()
	const stride = 8
	const offset = 2*stride + 2
	pix := makePredictionFixtureHigh(stride, 8, bitDepth)
	block := makePredictionBlockHigh(16)
	if err := fn(pix, offset, block, stride, bitDepth); err != nil {
		t.Fatal(err)
	}
	printPredBlockHigh(b, label, pix, stride, offset, 4, 4)
	fmt.Fprintf(b, "%s_sum %d\n", label, sumInt32(block))
}

func printPred8LAddCasesOracleWantHigh(t *testing.T, b *strings.Builder, label string, fn h264Pred8LAddFuncHigh, bitDepth int) {
	t.Helper()
	printPred8LAddOracleWantHigh(t, b, label+"_11", fn, true, true, bitDepth)
	printPred8LAddOracleWantHigh(t, b, label+"_00", fn, false, false, bitDepth)
}

func printPred8LAddOracleWantHigh(t *testing.T, b *strings.Builder, label string, fn h264Pred8LAddFuncHigh, hasTopLeft bool, hasTopRight bool, bitDepth int) {
	t.Helper()
	const stride = 28
	const offset = 5*stride + 5
	pix := makePredictionFixtureHigh(stride, 18, bitDepth)
	block := makePredictionBlockHigh(64)
	if err := fn(pix, offset, block, stride, hasTopLeft, hasTopRight, bitDepth); err != nil {
		t.Fatal(err)
	}
	printPredBlockHigh(b, label, pix, stride, offset, 8, 8)
	fmt.Fprintf(b, "%s_sum %d\n", label, sumInt32(block))
}

func printPredOffsetAddOracleWantHigh(t *testing.T, b *strings.Builder, label string, fn h264PredOffsetAddFuncHigh, width int, height int, blockCount int, bitDepth int) {
	t.Helper()
	const stride = 24
	const base = 4*stride + 4
	pix := makePredictionFixtureHigh(stride, 24, bitDepth)
	block := makePredictionBlockHigh(16 * 16)
	offsets, err := h264FrameBlockOffsets(stride, stride, 0)
	if err != nil {
		t.Fatal(err)
	}
	for i := range offsets {
		offsets[i] += base
	}
	if err := fn(pix, &offsets, block, stride, bitDepth); err != nil {
		t.Fatal(err)
	}
	printPredBlockHigh(b, label, pix, stride, base, width, height)
	fmt.Fprintf(b, "%s_sum %d\n", label, sumInt32(block[:blockCount*16]))
}

func makePredictionFixtureHigh(stride int, rows int, bitDepth int) []uint16 {
	pix := make([]uint16, stride*rows)
	mask := uint16((1 << uint(bitDepth)) - 1)
	for y := 0; y < rows; y++ {
		for x := 0; x < stride; x++ {
			pix[y*stride+x] = uint16(30+x*37+y*41) & mask
		}
	}
	return pix
}

func makePredictionTopRightHigh(bitDepth int) []uint16 {
	mask := uint16((1 << uint(bitDepth)) - 1)
	return []uint16{91 & mask, 257 & mask, 389 & mask, uint16((1<<uint(bitDepth))-7) & mask}
}

func makePredictionBlockHigh(n int) []int32 {
	block := make([]int32, n)
	for i := range block {
		block[i] = int32((i%7)*5 - 13)
	}
	return block
}

func printPredBlockHigh(b *strings.Builder, label string, pix []uint16, stride int, offset int, width int, height int) {
	fmt.Fprint(b, label)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			fmt.Fprintf(b, " %d", pix[offset+y*stride+x])
		}
	}
	fmt.Fprint(b, "\n")
}
