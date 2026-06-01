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

const reconstructOracleC = `
#include <stddef.h>
#include <stdint.h>
#include <stdio.h>
#include <string.h>

#define BIT_DEPTH 8
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

#define BIT_DEPTH 8
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

#define MB_TYPE_INTRA16x16 (1U << 1)

#define MB_WIDTH 4
#define MB_HEIGHT 4
#define LUMA_STRIDE 80
#define CHROMA_STRIDE 48
#define NNZ_SIZE (15 * 8)

enum {
    LUMA_DC_BLOCK_INDEX = 48,
    CHROMA_DC_BLOCK_INDEX = 49,
    GOH264_VERT_PRED = 0,
    GOH264_HOR_PRED = 1,
    GOH264_DC_PRED = 2,
    GOH264_DIAG_DOWN_LEFT_PRED = 3,
    GOH264_DIAG_DOWN_RIGHT_PRED = 4,
    GOH264_VERT_RIGHT_PRED = 5,
    GOH264_HOR_DOWN_PRED = 6,
    GOH264_VERT_LEFT_PRED = 7,
    GOH264_HOR_UP_PRED = 8,
    GOH264_LEFT_DC_PRED = 9,
    GOH264_TOP_DC_PRED = 10,
    GOH264_DC_128_PRED = 11,
};

typedef void (*pred4_fn)(uint8_t *src, const uint8_t *topright, ptrdiff_t stride);
typedef void (*pred8l_fn)(uint8_t *src, int has_topleft, int has_topright, ptrdiff_t stride);

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

static void init_intra_pcm(uint8_t *pcm, int n, int seed)
{
    for (int i = 0; i < n; i++)
        pcm[i] = (uint8_t)((seed + 17 * i + (i >> 3) + 3 * (i >> 6)) & 255);
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

static void init_offsets_high(int *offset, int luma_stride, int chroma_stride)
{
    const int base = scan8[0];
    for (int i = 0; i < 16; i++) {
        int delta = scan8[i] - base;
        int luma_sample = 4 * (delta & 7) + 4 * luma_stride * (delta >> 3);
        int chroma_sample = 4 * (delta & 7) + 4 * chroma_stride * (delta >> 3);
        offset[i] = luma_sample * (int)sizeof(uint16_t);
        offset[16 + i] = chroma_sample * (int)sizeof(uint16_t);
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

static void init_residual_intra4x4(int16_t *mb, int16_t *luma_dc, uint8_t *nnz)
{
    init_residual_420(mb, luma_dc, nnz);
    memset(luma_dc, 0, 16 * sizeof(*luma_dc));
    nnz[scan8[LUMA_DC_BLOCK_INDEX]] = 0;
    nnz[scan8[0]] = 2;
    mb[0] = 9;
    mb[1] = -3;
    nnz[scan8[5]] = 1;
    mb[5 * 16] = 11;
    nnz[scan8[10]] = 2;
    mb[10 * 16] = -8;
    mb[10 * 16 + 3] = 5;
    nnz[scan8[15]] = 1;
    mb[15 * 16] = 6;
}

static void init_residual_intra8x8(int16_t *mb, int16_t *luma_dc, uint8_t *nnz)
{
    init_residual_422(mb, luma_dc, nnz);
    memset(luma_dc, 0, 16 * sizeof(*luma_dc));
    nnz[scan8[LUMA_DC_BLOCK_INDEX]] = 0;
    for (int k = 0; k < 4; k++) {
        int i = 4 * k;
        nnz[scan8[i]] = 1;
        mb[i * 16] = 8 + i;
        mb[i * 16 + 7] = i - 5;
    }
    nnz[scan8[4]] = 2;
    mb[4 * 16 + 1] = -4;
    nnz[scan8[8]] = 2;
    mb[8 * 16 + 9] = 3;
}

static void init_residual_420_high(int32_t *mb, int32_t *luma_dc, uint8_t *nnz)
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

static void init_residual_inter_420_high(int32_t *mb, int32_t *luma_dc, uint8_t *nnz)
{
    memset(mb, 0, 48 * 16 * sizeof(*mb));
    memset(luma_dc, 0, 16 * sizeof(*luma_dc));
    memset(nnz, 0, NNZ_SIZE);

    nnz[scan8[0]] = 2;
    mb[0] = 128;
    mb[1] = -16;

    nnz[scan8[CHROMA_DC_BLOCK_INDEX + 0]] = 1;
    mb[16 * 16 + 0] = 3;
    mb[16 * 16 + 16] = 1;
    mb[16 * 16 + 32] = -2;
    mb[16 * 16 + 48] = 4;
}

static void init_residual_422_high(int32_t *mb, int32_t *luma_dc, uint8_t *nnz)
{
    init_residual_420_high(mb, luma_dc, nnz);
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

static void init_residual_intra4x4_high(int32_t *mb, int32_t *luma_dc, uint8_t *nnz)
{
    init_residual_420_high(mb, luma_dc, nnz);
    memset(luma_dc, 0, 16 * sizeof(*luma_dc));
    nnz[scan8[LUMA_DC_BLOCK_INDEX]] = 0;
    nnz[scan8[0]] = 2;
    mb[0] = 9;
    mb[1] = -3;
    nnz[scan8[5]] = 1;
    mb[5 * 16] = 11;
    nnz[scan8[10]] = 2;
    mb[10 * 16] = -8;
    mb[10 * 16 + 3] = 5;
    nnz[scan8[15]] = 1;
    mb[15 * 16] = 6;
}

static void init_residual_intra8x8_high(int32_t *mb, int32_t *luma_dc, uint8_t *nnz)
{
    init_residual_422_high(mb, luma_dc, nnz);
    memset(luma_dc, 0, 16 * sizeof(*luma_dc));
    nnz[scan8[LUMA_DC_BLOCK_INDEX]] = 0;
    for (int k = 0; k < 4; k++) {
        int i = 4 * k;
        nnz[scan8[i]] = 1;
        mb[i * 16] = 8 + i;
        mb[i * 16 + 7] = i - 5;
    }
    nnz[scan8[4]] = 2;
    mb[4 * 16 + 1] = -4;
    nnz[scan8[8]] = 2;
    mb[8 * 16 + 9] = 3;
}

static void init_intra4x4_pred_cache(int8_t *pred)
{
    static const int8_t modes[16] = {
        GOH264_VERT_PRED, GOH264_HOR_PRED, GOH264_DC_PRED, GOH264_DIAG_DOWN_LEFT_PRED,
        GOH264_DIAG_DOWN_RIGHT_PRED, GOH264_VERT_RIGHT_PRED, GOH264_HOR_DOWN_PRED, GOH264_VERT_LEFT_PRED,
        GOH264_HOR_UP_PRED, GOH264_LEFT_DC_PRED, GOH264_TOP_DC_PRED, GOH264_DC_128_PRED,
        GOH264_VERT_PRED, GOH264_HOR_PRED, GOH264_DC_PRED, GOH264_DIAG_DOWN_LEFT_PRED,
    };
    memset(pred, 0, NNZ_SIZE);
    for (int i = 0; i < 16; i++)
        pred[scan8[i]] = modes[i];
}

static void init_intra8x8_pred_cache(int8_t *pred)
{
    memset(pred, 0, NNZ_SIZE);
    pred[scan8[0]] = GOH264_VERT_PRED;
    pred[scan8[4]] = GOH264_DIAG_DOWN_LEFT_PRED;
    pred[scan8[8]] = GOH264_VERT_RIGHT_PRED;
    pred[scan8[12]] = GOH264_HOR_DOWN_PRED;
}

static pred4_fn pred4x4_func(int dir)
{
    switch (dir) {
    case GOH264_VERT_PRED: return pred4x4_vertical_8_c;
    case GOH264_HOR_PRED: return pred4x4_horizontal_8_c;
    case GOH264_DC_PRED: return pred4x4_dc_8_c;
    case GOH264_DIAG_DOWN_LEFT_PRED: return pred4x4_down_left_8_c;
    case GOH264_DIAG_DOWN_RIGHT_PRED: return pred4x4_down_right_8_c;
    case GOH264_VERT_RIGHT_PRED: return pred4x4_vertical_right_8_c;
    case GOH264_HOR_DOWN_PRED: return pred4x4_horizontal_down_8_c;
    case GOH264_VERT_LEFT_PRED: return pred4x4_vertical_left_8_c;
    case GOH264_HOR_UP_PRED: return pred4x4_horizontal_up_8_c;
    case GOH264_LEFT_DC_PRED: return pred4x4_left_dc_8_c;
    case GOH264_TOP_DC_PRED: return pred4x4_top_dc_8_c;
    case GOH264_DC_128_PRED: return pred4x4_128_dc_8_c;
    }
    return 0;
}

static pred8l_fn pred8x8l_func(int dir)
{
    switch (dir) {
    case GOH264_VERT_PRED: return pred8x8l_vertical_8_c;
    case GOH264_HOR_PRED: return pred8x8l_horizontal_8_c;
    case GOH264_DC_PRED: return pred8x8l_dc_8_c;
    case GOH264_DIAG_DOWN_LEFT_PRED: return pred8x8l_down_left_8_c;
    case GOH264_DIAG_DOWN_RIGHT_PRED: return pred8x8l_down_right_8_c;
    case GOH264_VERT_RIGHT_PRED: return pred8x8l_vertical_right_8_c;
    case GOH264_HOR_DOWN_PRED: return pred8x8l_horizontal_down_8_c;
    case GOH264_VERT_LEFT_PRED: return pred8x8l_vertical_left_8_c;
    case GOH264_HOR_UP_PRED: return pred8x8l_horizontal_up_8_c;
    case GOH264_LEFT_DC_PRED: return pred8x8l_left_dc_8_c;
    case GOH264_TOP_DC_PRED: return pred8x8l_top_dc_8_c;
    case GOH264_DC_128_PRED: return pred8x8l_128_dc_8_c;
    }
    return 0;
}

static pred4_fn pred4x4_func_high_10(int dir)
{
    switch (dir) {
    case GOH264_VERT_PRED: return pred4x4_vertical_10_c;
    case GOH264_HOR_PRED: return pred4x4_horizontal_10_c;
    case GOH264_DC_PRED: return pred4x4_dc_10_c;
    case GOH264_DIAG_DOWN_LEFT_PRED: return pred4x4_down_left_10_c;
    case GOH264_DIAG_DOWN_RIGHT_PRED: return pred4x4_down_right_10_c;
    case GOH264_VERT_RIGHT_PRED: return pred4x4_vertical_right_10_c;
    case GOH264_HOR_DOWN_PRED: return pred4x4_horizontal_down_10_c;
    case GOH264_VERT_LEFT_PRED: return pred4x4_vertical_left_10_c;
    case GOH264_HOR_UP_PRED: return pred4x4_horizontal_up_10_c;
    case GOH264_LEFT_DC_PRED: return pred4x4_left_dc_10_c;
    case GOH264_TOP_DC_PRED: return pred4x4_top_dc_10_c;
    case GOH264_DC_128_PRED: return pred4x4_128_dc_10_c;
    }
    return 0;
}

static pred8l_fn pred8x8l_func_high_12(int dir)
{
    switch (dir) {
    case GOH264_VERT_PRED: return pred8x8l_vertical_12_c;
    case GOH264_HOR_PRED: return pred8x8l_horizontal_12_c;
    case GOH264_DC_PRED: return pred8x8l_dc_12_c;
    case GOH264_DIAG_DOWN_LEFT_PRED: return pred8x8l_down_left_12_c;
    case GOH264_DIAG_DOWN_RIGHT_PRED: return pred8x8l_down_right_12_c;
    case GOH264_VERT_RIGHT_PRED: return pred8x8l_vertical_right_12_c;
    case GOH264_HOR_DOWN_PRED: return pred8x8l_horizontal_down_12_c;
    case GOH264_VERT_LEFT_PRED: return pred8x8l_vertical_left_12_c;
    case GOH264_HOR_UP_PRED: return pred8x8l_horizontal_up_12_c;
    case GOH264_LEFT_DC_PRED: return pred8x8l_left_dc_12_c;
    case GOH264_TOP_DC_PRED: return pred8x8l_top_dc_12_c;
    case GOH264_DC_128_PRED: return pred8x8l_128_dc_12_c;
    }
    return 0;
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

static void run_intra_pcm_420(void)
{
    Pic dst;
    uint8_t pcm[384];
    const int mb_x = 1, mb_y = 1;
    init_pic(&dst, 1, 17);
    init_intra_pcm(pcm, sizeof(pcm), 33);

    for (int i = 0; i < 16; i++)
        memcpy(pic_y(&dst, mb_x, mb_y) + i * LUMA_STRIDE, pcm + i * 16, 16);
    const uint8_t *src_cb = pcm + 256;
    const uint8_t *src_cr = pcm + 256 + 8 * 8;
    for (int i = 0; i < 8; i++) {
        memcpy(pic_cb(&dst, mb_x, mb_y) + i * CHROMA_STRIDE, src_cb + i * 8, 8);
        memcpy(pic_cr(&dst, mb_x, mb_y) + i * CHROMA_STRIDE, src_cr + i * 8, 8);
    }
    print_mb("intra_pcm_420", &dst, mb_x, mb_y);
}

static void run_intra_pcm_422(void)
{
    Pic dst;
    uint8_t pcm[512];
    const int mb_x = 1, mb_y = 1;
    init_pic(&dst, 2, 21);
    init_intra_pcm(pcm, sizeof(pcm), 49);

    for (int i = 0; i < 16; i++)
        memcpy(pic_y(&dst, mb_x, mb_y) + i * LUMA_STRIDE, pcm + i * 16, 16);
    const uint8_t *src_cb = pcm + 256;
    const uint8_t *src_cr = pcm + 256 + 16 * 8;
    for (int i = 0; i < 16; i++) {
        memcpy(pic_cb(&dst, mb_x, mb_y) + i * CHROMA_STRIDE, src_cb + i * 8, 8);
        memcpy(pic_cr(&dst, mb_x, mb_y) + i * CHROMA_STRIDE, src_cr + i * 8, 8);
    }
    print_mb("intra_pcm_422", &dst, mb_x, mb_y);
}

typedef struct PicHigh {
    uint16_t y[LUMA_STRIDE * MB_HEIGHT * 16];
    uint16_t cb[LUMA_STRIDE * MB_HEIGHT * 16];
    uint16_t cr[LUMA_STRIDE * MB_HEIGHT * 16];
    int chroma_idc;
    int chroma_stride;
} PicHigh;

static void fill_plane_high(uint16_t *p, int n, int seed)
{
    for (int i = 0; i < n; i++)
        p[i] = (uint16_t)((seed + i * 13 + (i >> 4) * 7) & 0x3fff);
}

static void init_pic_high(PicHigh *p, int chroma_idc, int seed)
{
    memset(p, 0, sizeof(*p));
    p->chroma_idc = chroma_idc;
    p->chroma_stride = chroma_idc == 3 ? LUMA_STRIDE : CHROMA_STRIDE;
    fill_plane_high(p->y, LUMA_STRIDE * MB_HEIGHT * 16, seed);
    fill_plane_high(p->cb, LUMA_STRIDE * MB_HEIGHT * 16, seed + 29);
    fill_plane_high(p->cr, LUMA_STRIDE * MB_HEIGHT * 16, seed + 71);
}

static uint16_t *pic_high_y(PicHigh *p, int mb_x, int mb_y)
{
    return p->y + mb_y * 16 * LUMA_STRIDE + mb_x * 16;
}

static uint16_t *pic_high_cb(PicHigh *p, int mb_x, int mb_y)
{
    if (p->chroma_idc == 1)
        return p->cb + mb_y * 8 * p->chroma_stride + mb_x * 8;
    if (p->chroma_idc == 2)
        return p->cb + mb_y * 16 * p->chroma_stride + mb_x * 8;
    return p->cb + mb_y * 16 * p->chroma_stride + mb_x * 16;
}

static uint16_t *pic_high_cr(PicHigh *p, int mb_x, int mb_y)
{
    if (p->chroma_idc == 1)
        return p->cr + mb_y * 8 * p->chroma_stride + mb_x * 8;
    if (p->chroma_idc == 2)
        return p->cr + mb_y * 16 * p->chroma_stride + mb_x * 8;
    return p->cr + mb_y * 16 * p->chroma_stride + mb_x * 16;
}

static void put_bit_high(uint8_t *buf, int *bit_pos, int bit)
{
    if (bit)
        buf[*bit_pos >> 3] |= 1U << (7 - (*bit_pos & 7));
    (*bit_pos)++;
}

static void init_intra_pcm_high(uint8_t *pcm, int sample_count, int bit_depth,
                                int seed)
{
    const int max = (1 << bit_depth) - 1;
    int bit_pos = 0;
    memset(pcm, 0, (sample_count * bit_depth + 7) >> 3);
    for (int i = 0; i < sample_count; i++) {
        int sample = (seed + 17 * i + (i >> 3) + 3 * (i >> 6)) & max;
        for (int bit = bit_depth - 1; bit >= 0; bit--)
            put_bit_high(pcm, &bit_pos, (sample >> bit) & 1);
    }
}

static unsigned get_bits_high(const uint8_t *buf, int *bit_pos, int n)
{
    unsigned v = 0;
    for (int i = 0; i < n; i++) {
        v = (v << 1) | ((buf[*bit_pos >> 3] >> (7 - (*bit_pos & 7))) & 1);
        (*bit_pos)++;
    }
    return v;
}

static void decode_high_plane(const uint8_t *pcm, int *bit_pos, uint16_t *dst,
                              int stride, int width, int height, int bit_depth)
{
    for (int y = 0; y < height; y++)
        for (int x = 0; x < width; x++)
            dst[y * stride + x] = get_bits_high(pcm, bit_pos, bit_depth);
}

static void decode_intra_pcm_high(PicHigh *dst, const uint8_t *pcm, int mb_x,
                                  int mb_y, int bit_depth)
{
    int bit_pos = 0;
    int chroma_w = 8;
    int chroma_h = 8;
    decode_high_plane(pcm, &bit_pos, pic_high_y(dst, mb_x, mb_y),
                      LUMA_STRIDE, 16, 16, bit_depth);
    if (!dst->chroma_idc)
        return;
    if (dst->chroma_idc == 2) {
        chroma_h = 16;
    } else if (dst->chroma_idc == 3) {
        chroma_w = 16;
        chroma_h = 16;
    }
    decode_high_plane(pcm, &bit_pos, pic_high_cb(dst, mb_x, mb_y),
                      dst->chroma_stride, chroma_w, chroma_h, bit_depth);
    decode_high_plane(pcm, &bit_pos, pic_high_cr(dst, mb_x, mb_y),
                      dst->chroma_stride, chroma_w, chroma_h, bit_depth);
}

static void print_mb_high(const char *label, PicHigh *p, int mb_x, int mb_y)
{
    uint16_t *y = pic_high_y(p, mb_x, mb_y);
    uint16_t *cb = pic_high_cb(p, mb_x, mb_y);
    uint16_t *cr = pic_high_cr(p, mb_x, mb_y);
    int cw = p->chroma_idc == 3 ? 16 : 8;
    int ch = p->chroma_idc == 1 ? 8 : 16;

    printf("%s y", label);
    for (int yy = 0; yy < 16; yy++)
        for (int x = 0; x < 16; x++)
            printf(" %u", y[yy * LUMA_STRIDE + x]);
    printf("\n");

    printf("%s cb", label);
    for (int yy = 0; yy < ch; yy++)
        for (int x = 0; x < cw; x++)
            printf(" %u", cb[yy * p->chroma_stride + x]);
    printf("\n");

    printf("%s cr", label);
    for (int yy = 0; yy < ch; yy++)
        for (int x = 0; x < cw; x++)
            printf(" %u", cr[yy * p->chroma_stride + x]);
    printf("\n");
}

static void run_intra_pcm_high_420_10(void)
{
    PicHigh dst;
    uint8_t pcm[480];
    const int mb_x = 1, mb_y = 1;
    init_pic_high(&dst, 1, 17);
    init_intra_pcm_high(pcm, 384, 10, 33);
    decode_intra_pcm_high(&dst, pcm, mb_x, mb_y, 10);
    print_mb_high("intra_pcm_high_420_10", &dst, mb_x, mb_y);
}

static void run_intra_pcm_high_422_12(void)
{
    PicHigh dst;
    uint8_t pcm[768];
    const int mb_x = 1, mb_y = 1;
    init_pic_high(&dst, 2, 21);
    init_intra_pcm_high(pcm, 512, 12, 49);
    decode_intra_pcm_high(&dst, pcm, mb_x, mb_y, 12);
    print_mb_high("intra_pcm_high_422_12", &dst, mb_x, mb_y);
}

static void run_intra_pcm_high_444_14(void)
{
    PicHigh dst;
    uint8_t pcm[1344];
    const int mb_x = 1, mb_y = 1;
    init_pic_high(&dst, 3, 23);
    init_intra_pcm_high(pcm, 768, 14, 57);
    decode_intra_pcm_high(&dst, pcm, mb_x, mb_y, 14);
    print_mb_high("intra_pcm_high_444_14", &dst, mb_x, mb_y);
}

static void run_intra16x16_high_420_10(void)
{
    PicHigh dst;
    int offset[48];
    int32_t mb[48 * 16];
    int32_t luma_dc[16];
    uint8_t nnz[NNZ_SIZE];
    uint8_t *dest[2];
    const int mb_x = 1, mb_y = 1;
    const int luma_stride_bytes = LUMA_STRIDE * (int)sizeof(uint16_t);
    init_pic_high(&dst, 1, 17);
    const int chroma_stride_bytes = dst.chroma_stride * (int)sizeof(uint16_t);
    init_offsets_high(offset, LUMA_STRIDE, dst.chroma_stride);
    init_residual_420_high(mb, luma_dc, nnz);

    pred8x8_horizontal_10_c((uint8_t *)pic_high_cb(&dst, mb_x, mb_y),
                            chroma_stride_bytes);
    pred8x8_horizontal_10_c((uint8_t *)pic_high_cr(&dst, mb_x, mb_y),
                            chroma_stride_bytes);
    pred16x16_vertical_10_c((uint8_t *)pic_high_y(&dst, mb_x, mb_y), luma_stride_bytes);
    if (nnz[scan8[LUMA_DC_BLOCK_INDEX]])
        ff_h264_luma_dc_dequant_idct_10_c((int16_t *)mb, (int16_t *)luma_dc, 64);
    ff_h264_idct_add16intra_10_c((uint8_t *)pic_high_y(&dst, mb_x, mb_y),
                                 offset, (int16_t *)mb, luma_stride_bytes, nnz);
    if (nnz[scan8[CHROMA_DC_BLOCK_INDEX + 0]])
        ff_h264_chroma_dc_dequant_idct_10_c((int16_t *)(mb + 16 * 16), 64);
    if (nnz[scan8[CHROMA_DC_BLOCK_INDEX + 1]])
        ff_h264_chroma_dc_dequant_idct_10_c((int16_t *)(mb + 32 * 16), 64);
    dest[0] = (uint8_t *)pic_high_cb(&dst, mb_x, mb_y);
    dest[1] = (uint8_t *)pic_high_cr(&dst, mb_x, mb_y);
    ff_h264_idct_add8_10_c(dest, offset, (int16_t *)mb, chroma_stride_bytes, nnz);
    print_mb_high("intra16x16_high_420_10", &dst, mb_x, mb_y);
}

static void run_inter_p16x16_high_420_10(void)
{
    PicHigh dst, ref;
    int offset[48];
    int32_t mb[48 * 16];
    int32_t luma_dc[16];
    uint8_t nnz[NNZ_SIZE];
    uint8_t *dest[2];
    const int mb_x = 1, mb_y = 1;
    const int luma_stride_bytes = LUMA_STRIDE * (int)sizeof(uint16_t);
    init_pic_high(&dst, 1, 9);
    init_pic_high(&ref, 1, 77);
    const int chroma_stride_bytes = dst.chroma_stride * (int)sizeof(uint16_t);
    init_offsets_high(offset, LUMA_STRIDE, dst.chroma_stride);
    init_residual_inter_420_high(mb, luma_dc, nnz);

    for (int y = 0; y < 16; y++)
        memcpy(pic_high_y(&dst, mb_x, mb_y) + y * LUMA_STRIDE,
               pic_high_y(&ref, mb_x, mb_y) + y * LUMA_STRIDE,
               16 * sizeof(uint16_t));
    for (int y = 0; y < 8; y++) {
        memcpy(pic_high_cb(&dst, mb_x, mb_y) + y * dst.chroma_stride,
               pic_high_cb(&ref, mb_x, mb_y) + y * ref.chroma_stride,
               8 * sizeof(uint16_t));
        memcpy(pic_high_cr(&dst, mb_x, mb_y) + y * dst.chroma_stride,
               pic_high_cr(&ref, mb_x, mb_y) + y * ref.chroma_stride,
               8 * sizeof(uint16_t));
    }

    ff_h264_idct_add16_10_c((uint8_t *)pic_high_y(&dst, mb_x, mb_y),
                            offset, (int16_t *)mb, luma_stride_bytes, nnz);
    if (nnz[scan8[CHROMA_DC_BLOCK_INDEX + 0]])
        ff_h264_chroma_dc_dequant_idct_10_c((int16_t *)(mb + 16 * 16), 64);
    if (nnz[scan8[CHROMA_DC_BLOCK_INDEX + 1]])
        ff_h264_chroma_dc_dequant_idct_10_c((int16_t *)(mb + 32 * 16), 64);
    dest[0] = (uint8_t *)pic_high_cb(&dst, mb_x, mb_y);
    dest[1] = (uint8_t *)pic_high_cr(&dst, mb_x, mb_y);
    ff_h264_idct_add8_10_c(dest, offset, (int16_t *)mb, chroma_stride_bytes, nnz);
    print_mb_high("inter_p16x16_high_420_10", &dst, mb_x, mb_y);
}

static void run_intra4x4_high_420_10(void)
{
    PicHigh dst;
    int offset[48];
    int32_t mb[48 * 16];
    int32_t luma_dc[16];
    uint8_t nnz[NNZ_SIZE];
    int8_t pred_cache[NNZ_SIZE];
    uint8_t *dest[2];
    const int mb_x = 1, mb_y = 1;
    const int top_right_available = 0xeeff;
    const int luma_stride_bytes = LUMA_STRIDE * (int)sizeof(uint16_t);
    init_pic_high(&dst, 1, 17);
    const int chroma_stride_bytes = dst.chroma_stride * (int)sizeof(uint16_t);
    init_offsets_high(offset, LUMA_STRIDE, dst.chroma_stride);
    init_residual_intra4x4_high(mb, luma_dc, nnz);
    init_intra4x4_pred_cache(pred_cache);

    pred8x8_dc_10_c((uint8_t *)pic_high_cb(&dst, mb_x, mb_y), chroma_stride_bytes);
    pred8x8_dc_10_c((uint8_t *)pic_high_cr(&dst, mb_x, mb_y), chroma_stride_bytes);
    for (int i = 0; i < 16; i++) {
        uint16_t *ptr = pic_high_y(&dst, mb_x, mb_y) + offset[i] / (int)sizeof(uint16_t);
        const int dir = pred_cache[scan8[i]];
        const uint8_t *topright = NULL;
        uint16_t unavailable_topright[4];
        pred4_fn pred = pred4x4_func_high_10(dir);
        if (dir == GOH264_DIAG_DOWN_LEFT_PRED || dir == GOH264_VERT_LEFT_PRED) {
            if ((top_right_available << i) & 0x8000) {
                topright = (const uint8_t *)(ptr + 4 - LUMA_STRIDE);
            } else {
                for (int j = 0; j < 4; j++)
                    unavailable_topright[j] = ptr[3 - LUMA_STRIDE];
                topright = (const uint8_t *)unavailable_topright;
            }
        }
        pred((uint8_t *)ptr, topright, luma_stride_bytes);
        if (nnz[scan8[i]]) {
            if (nnz[scan8[i]] == 1 && mb[i * 16])
                ff_h264_idct_dc_add_10_c((uint8_t *)ptr, (int16_t *)(mb + i * 16), luma_stride_bytes);
            else
                ff_h264_idct_add_10_c((uint8_t *)ptr, (int16_t *)(mb + i * 16), luma_stride_bytes);
        }
    }
    if (nnz[scan8[CHROMA_DC_BLOCK_INDEX + 0]])
        ff_h264_chroma_dc_dequant_idct_10_c((int16_t *)(mb + 16 * 16), 64);
    if (nnz[scan8[CHROMA_DC_BLOCK_INDEX + 1]])
        ff_h264_chroma_dc_dequant_idct_10_c((int16_t *)(mb + 32 * 16), 64);
    dest[0] = (uint8_t *)pic_high_cb(&dst, mb_x, mb_y);
    dest[1] = (uint8_t *)pic_high_cr(&dst, mb_x, mb_y);
    ff_h264_idct_add8_10_c(dest, offset, (int16_t *)mb, chroma_stride_bytes, nnz);
    print_mb_high("intra4x4_high_420_10", &dst, mb_x, mb_y);
}

static void run_intra8x8_high_422_12(void)
{
    PicHigh dst;
    int offset[48];
    int32_t mb[48 * 16];
    int32_t luma_dc[16];
    uint8_t nnz[NNZ_SIZE];
    int8_t pred_cache[NNZ_SIZE];
    uint8_t *dest[2];
    const int mb_x = 1, mb_y = 1;
    const int top_left_available = 0xffff;
    const int top_right_available = 0xffff;
    const int luma_stride_bytes = LUMA_STRIDE * (int)sizeof(uint16_t);
    init_pic_high(&dst, 2, 31);
    const int chroma_stride_bytes = dst.chroma_stride * (int)sizeof(uint16_t);
    init_offsets_high(offset, LUMA_STRIDE, dst.chroma_stride);
    init_residual_intra8x8_high(mb, luma_dc, nnz);
    init_intra8x8_pred_cache(pred_cache);

    pred8x16_plane_12_c((uint8_t *)pic_high_cb(&dst, mb_x, mb_y), chroma_stride_bytes);
    pred8x16_plane_12_c((uint8_t *)pic_high_cr(&dst, mb_x, mb_y), chroma_stride_bytes);
    for (int i = 0; i < 16; i += 4) {
        uint16_t *ptr = pic_high_y(&dst, mb_x, mb_y) + offset[i] / (int)sizeof(uint16_t);
        const int dir = pred_cache[scan8[i]];
        pred8l_fn pred = pred8x8l_func_high_12(dir);
        pred((uint8_t *)ptr, (top_left_available << i) & 0x8000,
             (top_right_available << i) & 0x4000, luma_stride_bytes);
        if (nnz[scan8[i]]) {
            if (nnz[scan8[i]] == 1 && mb[i * 16])
                ff_h264_idct8_dc_add_12_c((uint8_t *)ptr, (int16_t *)(mb + i * 16), luma_stride_bytes);
            else
                ff_h264_idct8_add_12_c((uint8_t *)ptr, (int16_t *)(mb + i * 16), luma_stride_bytes);
        }
    }
    if (nnz[scan8[CHROMA_DC_BLOCK_INDEX + 0]])
        ff_h264_chroma422_dc_dequant_idct_12_c((int16_t *)(mb + 16 * 16), 64);
    if (nnz[scan8[CHROMA_DC_BLOCK_INDEX + 1]])
        ff_h264_chroma422_dc_dequant_idct_12_c((int16_t *)(mb + 32 * 16), 64);
    dest[0] = (uint8_t *)pic_high_cb(&dst, mb_x, mb_y);
    dest[1] = (uint8_t *)pic_high_cr(&dst, mb_x, mb_y);
    ff_h264_idct_add8_422_12_c(dest, offset, (int16_t *)mb, chroma_stride_bytes, nnz);
    print_mb_high("intra8x8_high_422_12", &dst, mb_x, mb_y);
}

static void run_intra4x4_420(void)
{
    Pic dst;
    int offset[48];
    int16_t mb[48 * 16];
    int16_t luma_dc[16];
    uint8_t nnz[NNZ_SIZE];
    int8_t pred_cache[NNZ_SIZE];
    uint8_t *dest[2];
    const int mb_x = 1, mb_y = 1;
    const int top_right_available = 0xeeff;
    init_pic(&dst, 1, 17);
    init_offsets(offset, LUMA_STRIDE, CHROMA_STRIDE);
    init_residual_intra4x4(mb, luma_dc, nnz);
    init_intra4x4_pred_cache(pred_cache);

    pred8x8_dc_8_c(pic_cb(&dst, mb_x, mb_y), CHROMA_STRIDE);
    pred8x8_dc_8_c(pic_cr(&dst, mb_x, mb_y), CHROMA_STRIDE);
    for (int i = 0; i < 16; i++) {
        uint8_t *ptr = pic_y(&dst, mb_x, mb_y) + offset[i];
        const int dir = pred_cache[scan8[i]];
        const uint8_t *topright = NULL;
        uint8_t unavailable_topright[4];
        pred4_fn pred = pred4x4_func(dir);
        if (dir == GOH264_DIAG_DOWN_LEFT_PRED || dir == GOH264_VERT_LEFT_PRED) {
            if ((top_right_available << i) & 0x8000) {
                topright = ptr + 4 - LUMA_STRIDE;
            } else {
                memset(unavailable_topright, ptr[3 - LUMA_STRIDE], sizeof(unavailable_topright));
                topright = unavailable_topright;
            }
        }
        pred(ptr, topright, LUMA_STRIDE);
        if (nnz[scan8[i]]) {
            if (nnz[scan8[i]] == 1 && mb[i * 16])
                ff_h264_idct_dc_add_8_c(ptr, mb + i * 16, LUMA_STRIDE);
            else
                ff_h264_idct_add_8_c(ptr, mb + i * 16, LUMA_STRIDE);
        }
    }
    if (nnz[scan8[CHROMA_DC_BLOCK_INDEX + 0]])
        ff_h264_chroma_dc_dequant_idct_8_c(mb + 16 * 16, 64);
    if (nnz[scan8[CHROMA_DC_BLOCK_INDEX + 1]])
        ff_h264_chroma_dc_dequant_idct_8_c(mb + 32 * 16, 64);
    dest[0] = pic_cb(&dst, mb_x, mb_y);
    dest[1] = pic_cr(&dst, mb_x, mb_y);
    ff_h264_idct_add8_8_c(dest, offset, mb, CHROMA_STRIDE, nnz);
    print_mb("intra4x4_420", &dst, mb_x, mb_y);
}

static void run_intra8x8_422(void)
{
    Pic dst;
    int offset[48];
    int16_t mb[48 * 16];
    int16_t luma_dc[16];
    uint8_t nnz[NNZ_SIZE];
    int8_t pred_cache[NNZ_SIZE];
    uint8_t *dest[2];
    const int mb_x = 1, mb_y = 1;
    const int top_left_available = 0xffff;
    const int top_right_available = 0xffff;
    init_pic(&dst, 2, 31);
    init_offsets(offset, LUMA_STRIDE, CHROMA_STRIDE);
    init_residual_intra8x8(mb, luma_dc, nnz);
    init_intra8x8_pred_cache(pred_cache);

    pred8x16_plane_8_c(pic_cb(&dst, mb_x, mb_y), CHROMA_STRIDE);
    pred8x16_plane_8_c(pic_cr(&dst, mb_x, mb_y), CHROMA_STRIDE);
    for (int i = 0; i < 16; i += 4) {
        uint8_t *ptr = pic_y(&dst, mb_x, mb_y) + offset[i];
        const int dir = pred_cache[scan8[i]];
        pred8l_fn pred = pred8x8l_func(dir);
        pred(ptr, (top_left_available << i) & 0x8000,
             (top_right_available << i) & 0x4000, LUMA_STRIDE);
        if (nnz[scan8[i]]) {
            if (nnz[scan8[i]] == 1 && mb[i * 16])
                ff_h264_idct8_dc_add_8_c(ptr, mb + i * 16, LUMA_STRIDE);
            else
                ff_h264_idct8_add_8_c(ptr, mb + i * 16, LUMA_STRIDE);
        }
    }
    if (nnz[scan8[CHROMA_DC_BLOCK_INDEX + 0]])
        ff_h264_chroma422_dc_dequant_idct_8_c(mb + 16 * 16, 64);
    if (nnz[scan8[CHROMA_DC_BLOCK_INDEX + 1]])
        ff_h264_chroma422_dc_dequant_idct_8_c(mb + 32 * 16, 64);
    dest[0] = pic_cb(&dst, mb_x, mb_y);
    dest[1] = pic_cr(&dst, mb_x, mb_y);
    ff_h264_idct_add8_422_8_c(dest, offset, mb, CHROMA_STRIDE, nnz);
    print_mb("intra8x8_422", &dst, mb_x, mb_y);
}

int main(void)
{
    run_intra16x16_420();
    run_intra16x16_422();
    run_intra_pcm_420();
    run_intra_pcm_422();
    run_intra_pcm_high_420_10();
    run_intra_pcm_high_422_12();
    run_intra_pcm_high_444_14();
    run_intra16x16_high_420_10();
    run_inter_p16x16_high_420_10();
    run_intra4x4_high_420_10();
    run_intra8x8_high_422_12();
    run_intra4x4_420();
    run_intra8x8_422();
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

static inline uint16_t goh264_recon_clip_uintp2(int v, int p)
{
    int max = (1 << p) - 1;
    if (v < 0)
        return 0;
    if (v > max)
        return (uint16_t)max;
    return (uint16_t)v;
}

static inline uint64_t goh264_recon_rn8pa(const void *p)
{
    const uint8_t *b = (const uint8_t *)p;
    return (uint64_t)b[0] | ((uint64_t)b[1] << 8) |
           ((uint64_t)b[2] << 16) | ((uint64_t)b[3] << 24) |
           ((uint64_t)b[4] << 32) | ((uint64_t)b[5] << 40) |
           ((uint64_t)b[6] << 48) | ((uint64_t)b[7] << 56);
}

static inline void goh264_recon_wn8pa(void *p, uint64_t v)
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
#undef SUINT
#undef av_unused
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
#if BIT_DEPTH > 8
#define pixel uint16_t
#define pixel2 uint32_t
#define pixel4 uint64_t
#define dctcoef int32_t
#define SUINT unsigned
#define FUNC3(a, b, c)  a ## _ ## b ## c
#define FUNC2(a, b, c)  FUNC3(a, b, c)
#define FUNC(a) FUNC2(a, BIT_DEPTH, _c)
#define FUNCC(a) FUNC2(a, BIT_DEPTH, _c)
#define AV_RN4PA(p) goh264_recon_rn8pa(p)
#define AV_WN4PA(p, v) goh264_recon_wn8pa((void *)(p), (uint64_t)(v))
#define PIXEL_SPLAT_X4(x) ((uint64_t)(uint16_t)(x) * 0x0001000100010001ULL)
#define CLIP(a) goh264_recon_clip_uintp2((a), BIT_DEPTH)
#define av_clip_pixel(a) goh264_recon_clip_uintp2((a), BIT_DEPTH)
#else
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
#endif
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
	appendH264ReconstructOracleIntraPCM(t, &b, "intra_pcm_420", 1, 17, h264ReconstructIntraPCM(1, 33))
	appendH264ReconstructOracleIntraPCM(t, &b, "intra_pcm_422", 2, 21, h264ReconstructIntraPCM(2, 49))
	appendH264ReconstructOracleIntraPCMHigh(t, &b, "intra_pcm_high_420_10", 1, 10, 17, h264ReconstructIntraPCMHigh(1, 10, 33))
	appendH264ReconstructOracleIntraPCMHigh(t, &b, "intra_pcm_high_422_12", 2, 12, 21, h264ReconstructIntraPCMHigh(2, 12, 49))
	appendH264ReconstructOracleIntraPCMHigh(t, &b, "intra_pcm_high_444_14", 3, 14, 23, h264ReconstructIntraPCMHigh(3, 14, 57))
	appendH264ReconstructOracleIntra16x16High(t, &b, "intra16x16_high_420_10", 1, 10, 17, int32(intraPred8x8Horizontal), int8(intraPred8x8Vertical), h264ReconstructResidual420())
	appendH264ReconstructOracleInterP16x16High(t, &b, "inter_p16x16_high_420_10")
	appendH264ReconstructOracleIntra4x4High(t, &b, "intra4x4_high_420_10", 1, 10, 17, int32(intraPred8x8DC), 0xffff, 0xeeff, h264ReconstructResidualIntra4x4())
	appendH264ReconstructOracleIntra8x8High(t, &b, "intra8x8_high_422_12", 2, 12, 31, int32(intraPred8x8Plane), 0xffff, 0xffff, h264ReconstructResidualIntra8x8())
	appendH264ReconstructOracleIntra4x4(t, &b, "intra4x4_420", 1, 17, int32(intraPred8x8DC), 0xffff, 0xeeff, h264ReconstructResidualIntra4x4())
	appendH264ReconstructOracleIntra8x8(t, &b, "intra8x8_422", 2, 31, int32(intraPred8x8Plane), 0xffff, 0xffff, h264ReconstructResidualIntra8x8())
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

func appendH264ReconstructOracleIntraPCM(t *testing.T, b *strings.Builder, label string, chromaFormatIDC int, seed int, pcm []byte) {
	dst := makeH264MotionCompPicture(chromaFormatIDC, seed)
	if err := h264HLDecodeFrameMacroblock(dst, h264FrameMBReconstructInput{
		MBType:   MBTypeIntraPCM,
		MBX:      1,
		MBY:      1,
		IntraPCM: pcm,
	}); err != nil {
		t.Fatal(err)
	}
	printH264MotionCompMB(b, label, dst, 1, 1)
}

func appendH264ReconstructOracleIntraPCMHigh(t *testing.T, b *strings.Builder, label string, chromaFormatIDC int, bitDepth int, seed int, pcm []byte) {
	dst := makeH264ReconstructHighPicture(chromaFormatIDC, seed)
	yOff, cbOff, crOff, err := h264MBDestPartOffsetsHigh(dst, 1, 1, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := h264HLDecodeFrameIntraPCMHigh(dst, yOff, cbOff, crOff, pcm, bitDepth); err != nil {
		t.Fatal(err)
	}
	printH264MotionCompMBHigh(b, label, dst, 1, 1)
}

func appendH264ReconstructOracleIntra16x16High(t *testing.T, b *strings.Builder, label string, chromaFormatIDC int, bitDepth int, seed int, chromaPred int32, lumaPred int8, residual cavlcResidualContext) {
	dst := makeH264ReconstructHighPicture(chromaFormatIDC, seed)
	if err := h264HLDecodeFrameMacroblockHigh(dst, h264FrameMBReconstructInputHigh{
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
		BitDepth:           bitDepth,
	}); err != nil {
		t.Fatal(err)
	}
	printH264MotionCompMBHigh(b, label, dst, 1, 1)
}

func appendH264ReconstructOracleInterP16x16High(t *testing.T, b *strings.Builder, label string) {
	const bitDepth = 10
	dst := makeH264ReconstructHighPicture(1, 9)
	ref := makeH264ReconstructHighPicture(1, 77)
	refs := [2][]*h264PicturePlanesHigh{{ref}}
	var cache macroblockMotionCache
	cache.Ref[0][h264Scan8[0]] = 0
	residual := h264ReconstructResidualInter420()
	if err := h264HLDecodeFrameMacroblockHigh(dst, h264FrameMBReconstructInputHigh{
		MBType:        MBType16x16 | MBTypeP0L0,
		MBX:           1,
		MBY:           1,
		CBP:           0x21,
		QScale:        18,
		ChromaQP:      [2]uint8{18, 18},
		ListCount:     1,
		PPS:           cavlcFlatQMulPPS(),
		Residual:      &residual,
		Motion:        &cache,
		Refs:          refs,
		MotionScratch: makeH264MotionCompScratchHigh(dst),
		BitDepth:      bitDepth,
	}); err != nil {
		t.Fatal(err)
	}
	printH264MotionCompMBHigh(b, label, dst, 1, 1)
}

func appendH264ReconstructOracleIntra4x4(t *testing.T, b *strings.Builder, label string, chromaFormatIDC int, seed int, chromaPred int32, topLeftAvailable uint16, topRightAvailable uint16, residual cavlcResidualContext) {
	dst := makeH264MotionCompPicture(chromaFormatIDC, seed)
	predCache := h264ReconstructIntra4x4PredCache()
	if err := h264HLDecodeFrameMacroblock(dst, h264FrameMBReconstructInput{
		MBType:            MBTypeIntra4x4,
		MBX:               1,
		MBY:               1,
		CBP:               0x31,
		QScale:            20,
		ChromaQP:          [2]uint8{20, 21},
		ChromaPredMode:    chromaPred,
		Intra4x4PredCache: &predCache,
		TopLeftAvailable:  topLeftAvailable,
		TopRightAvailable: topRightAvailable,
		PPS:               cavlcFlatQMulPPS(),
		Residual:          &residual,
	}); err != nil {
		t.Fatal(err)
	}
	printH264MotionCompMB(b, label, dst, 1, 1)
}

func appendH264ReconstructOracleIntra4x4High(t *testing.T, b *strings.Builder, label string, chromaFormatIDC int, bitDepth int, seed int, chromaPred int32, topLeftAvailable uint16, topRightAvailable uint16, residual cavlcResidualContext) {
	dst := makeH264ReconstructHighPicture(chromaFormatIDC, seed)
	predCache := h264ReconstructIntra4x4PredCache()
	if err := h264HLDecodeFrameMacroblockHigh(dst, h264FrameMBReconstructInputHigh{
		MBType:            MBTypeIntra4x4,
		MBX:               1,
		MBY:               1,
		CBP:               0x31,
		QScale:            20,
		ChromaQP:          [2]uint8{20, 21},
		ChromaPredMode:    chromaPred,
		Intra4x4PredCache: &predCache,
		TopLeftAvailable:  topLeftAvailable,
		TopRightAvailable: topRightAvailable,
		PPS:               cavlcFlatQMulPPS(),
		Residual:          &residual,
		BitDepth:          bitDepth,
	}); err != nil {
		t.Fatal(err)
	}
	printH264MotionCompMBHigh(b, label, dst, 1, 1)
}

func appendH264ReconstructOracleIntra8x8(t *testing.T, b *strings.Builder, label string, chromaFormatIDC int, seed int, chromaPred int32, topLeftAvailable uint16, topRightAvailable uint16, residual cavlcResidualContext) {
	dst := makeH264MotionCompPicture(chromaFormatIDC, seed)
	predCache := h264ReconstructIntra8x8PredCache()
	if err := h264HLDecodeFrameMacroblock(dst, h264FrameMBReconstructInput{
		MBType:            MBTypeIntra4x4 | MBType8x8DCT,
		MBX:               1,
		MBY:               1,
		CBP:               0x33,
		QScale:            22,
		ChromaQP:          [2]uint8{22, 23},
		ChromaPredMode:    chromaPred,
		Intra4x4PredCache: &predCache,
		TopLeftAvailable:  topLeftAvailable,
		TopRightAvailable: topRightAvailable,
		PPS:               cavlcFlatQMulPPS(),
		Residual:          &residual,
	}); err != nil {
		t.Fatal(err)
	}
	printH264MotionCompMB(b, label, dst, 1, 1)
}

func appendH264ReconstructOracleIntra8x8High(t *testing.T, b *strings.Builder, label string, chromaFormatIDC int, bitDepth int, seed int, chromaPred int32, topLeftAvailable uint16, topRightAvailable uint16, residual cavlcResidualContext) {
	dst := makeH264ReconstructHighPicture(chromaFormatIDC, seed)
	predCache := h264ReconstructIntra8x8PredCache()
	if err := h264HLDecodeFrameMacroblockHigh(dst, h264FrameMBReconstructInputHigh{
		MBType:            MBTypeIntra4x4 | MBType8x8DCT,
		MBX:               1,
		MBY:               1,
		CBP:               0x33,
		QScale:            22,
		ChromaQP:          [2]uint8{22, 23},
		ChromaPredMode:    chromaPred,
		Intra4x4PredCache: &predCache,
		TopLeftAvailable:  topLeftAvailable,
		TopRightAvailable: topRightAvailable,
		PPS:               cavlcFlatQMulPPS(),
		Residual:          &residual,
		BitDepth:          bitDepth,
	}); err != nil {
		t.Fatal(err)
	}
	printH264MotionCompMBHigh(b, label, dst, 1, 1)
}

func printH264MotionCompMBHigh(b *strings.Builder, label string, p *h264PicturePlanesHigh, mbX int, mbY int) {
	yOff := mbY*16*p.LumaStride + mbX*16
	fmt.Fprintf(b, "%s y", label)
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			fmt.Fprintf(b, " %d", p.Y[yOff+y*p.LumaStride+x])
		}
	}
	b.WriteByte('\n')

	cw, ch := 16, 16
	cOff := mbY*16*p.ChromaStride + mbX*16
	if p.ChromaFormatIDC == 1 {
		cw, ch = 8, 8
		cOff = mbY*8*p.ChromaStride + mbX*8
	} else if p.ChromaFormatIDC == 2 {
		cw, ch = 8, 16
		cOff = mbY*16*p.ChromaStride + mbX*8
	}
	fmt.Fprintf(b, "%s cb", label)
	for y := 0; y < ch; y++ {
		for x := 0; x < cw; x++ {
			fmt.Fprintf(b, " %d", p.Cb[cOff+y*p.ChromaStride+x])
		}
	}
	b.WriteByte('\n')
	fmt.Fprintf(b, "%s cr", label)
	for y := 0; y < ch; y++ {
		for x := 0; x < cw; x++ {
			fmt.Fprintf(b, " %d", p.Cr[cOff+y*p.ChromaStride+x])
		}
	}
	b.WriteByte('\n')
}
