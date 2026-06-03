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

const motionCompOracleC = `
#include <stddef.h>
#include <stdint.h>
#include <stdio.h>
#include <string.h>

#define av_unused
#define pixeltmp int16_t
#define BIT_DEPTH 8
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
#undef pixeltmp

#define BIT_DEPTH 8
#include "h264chroma_template.c"
#undef BIT_DEPTH
#define BIT_DEPTH 10
#include "h264chroma_template.c"
#undef BIT_DEPTH
#define BIT_DEPTH 12
#include "h264chroma_template.c"
#undef BIT_DEPTH

#define av_always_inline inline
#define av_flatten
#define FFABS(a) ((a) >= 0 ? (a) : -(a))
#define FFMIN(a,b) ((a) > (b) ? (b) : (a))
#define FFMAX(a,b) ((a) > (b) ? (a) : (b))
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
#define BIT_DEPTH 10
#include "h264dsp_template.c"
#undef BIT_DEPTH
#define BIT_DEPTH 12
#include "h264dsp_template.c"
#undef BIT_DEPTH

#define BIT_DEPTH 8
#include "videodsp_template.c"
#undef BIT_DEPTH
#define BIT_DEPTH 10
#include "videodsp_template.c"
#undef BIT_DEPTH
#define BIT_DEPTH 12
#include "videodsp_template.c"
#undef BIT_DEPTH

#define MB_TYPE_16x16      (1U << 3)
#define MB_TYPE_16x8       (1U << 4)
#define MB_TYPE_8x16       (1U << 5)
#define MB_TYPE_8x8        (1U << 6)
#define MB_TYPE_DIRECT2    (1U << 8)
#define MB_TYPE_P0L0       (1U << 12)
#define MB_TYPE_P1L0       (1U << 13)
#define MB_TYPE_P0L1       (1U << 14)
#define MB_TYPE_P1L1       (1U << 15)

#define MB_WIDTH 4
#define MB_HEIGHT 4
#define LUMA_STRIDE 80
#define CHROMA_STRIDE_420_422 48
#define MOTION_CACHE_SIZE 40

static const uint8_t scan8[16] = {
    4 + 1 * 8, 5 + 1 * 8, 4 + 2 * 8, 5 + 2 * 8,
    6 + 1 * 8, 7 + 1 * 8, 6 + 2 * 8, 7 + 2 * 8,
    4 + 3 * 8, 5 + 3 * 8, 4 + 4 * 8, 5 + 4 * 8,
    6 + 3 * 8, 7 + 3 * 8, 6 + 4 * 8, 7 + 4 * 8,
};

typedef void (*qpel_fn)(uint8_t *dst, const uint8_t *src, ptrdiff_t stride);
typedef void (*chroma_fn)(uint8_t *dst, const uint8_t *src, ptrdiff_t stride,
                          int h, int x, int y);
typedef void (*weight_fn)(uint8_t *block, ptrdiff_t stride, int height,
                          int log2_denom, int weight, int offset);
typedef void (*biweight_fn)(uint8_t *dst, uint8_t *src, ptrdiff_t stride,
                            int height, int log2_denom, int weightd,
                            int weights, int offset);

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
static qpel_fn put4_ ## depth[16] = QPEL_LIST(put_h264_qpel, 4, depth); \
static qpel_fn put8_ ## depth[16] = QPEL_LIST(put_h264_qpel, 8, depth); \
static qpel_fn put16_ ## depth[16] = QPEL_LIST(put_h264_qpel, 16, depth); \
static qpel_fn avg4_ ## depth[16] = QPEL_LIST(avg_h264_qpel, 4, depth); \
static qpel_fn avg8_ ## depth[16] = QPEL_LIST(avg_h264_qpel, 8, depth); \
static qpel_fn avg16_ ## depth[16] = QPEL_LIST(avg_h264_qpel, 16, depth)

DECL_QPEL_TABLES(8);
DECL_QPEL_TABLES(10);
DECL_QPEL_TABLES(12);

typedef struct Pic {
    uint8_t y[LUMA_STRIDE * MB_HEIGHT * 16];
    uint8_t cb[LUMA_STRIDE * MB_HEIGHT * 16];
    uint8_t cr[LUMA_STRIDE * MB_HEIGHT * 16];
    int chroma_idc;
    int chroma_stride;
} Pic;

typedef struct PicHigh {
    uint16_t y[LUMA_STRIDE * MB_HEIGHT * 16];
    uint16_t cb[LUMA_STRIDE * MB_HEIGHT * 16];
    uint16_t cr[LUMA_STRIDE * MB_HEIGHT * 16];
    int chroma_idc;
    int chroma_stride;
    int bit_depth;
} PicHigh;

typedef struct MotionCtx {
    int16_t mv_cache[2][MOTION_CACHE_SIZE][2];
    int8_t ref_cache[2][MOTION_CACHE_SIZE];
} MotionCtx;

typedef struct PWT {
    int use_weight;
    int use_weight_chroma;
    int luma_log2_weight_denom;
    int chroma_log2_weight_denom;
    int luma_weight[48][2][2];
    int chroma_weight[48][2][2][2];
    int implicit_weight[48][48][2];
} PWT;

static qpel_fn qpel_func(int size, int idx, int avg)
{
    if (avg) {
        switch (size) {
        case 4:  return avg4_8[idx];
        case 8:  return avg8_8[idx];
        case 16: return avg16_8[idx];
        }
    } else {
        switch (size) {
        case 4:  return put4_8[idx];
        case 8:  return put8_8[idx];
        case 16: return put16_8[idx];
        }
    }
    return 0;
}

static qpel_fn qpel_func_high(int size, int idx, int avg, int bit_depth)
{
#define RETURN_QPEL(depth) \
    do { \
        if (avg) { \
            switch (size) { \
            case 4:  return avg4_ ## depth[idx]; \
            case 8:  return avg8_ ## depth[idx]; \
            case 16: return avg16_ ## depth[idx]; \
            } \
        } else { \
            switch (size) { \
            case 4:  return put4_ ## depth[idx]; \
            case 8:  return put8_ ## depth[idx]; \
            case 16: return put16_ ## depth[idx]; \
            } \
        } \
    } while (0)
    switch (bit_depth) {
    case 10:
        RETURN_QPEL(10);
        break;
    case 12:
        RETURN_QPEL(12);
        break;
    }
#undef RETURN_QPEL
    return 0;
}

static chroma_fn chroma_func(int width, int avg)
{
    if (avg) {
        switch (width) {
        case 1: return avg_h264_chroma_mc1_8_c;
        case 2: return avg_h264_chroma_mc2_8_c;
        case 4: return avg_h264_chroma_mc4_8_c;
        case 8: return avg_h264_chroma_mc8_8_c;
        }
    } else {
        switch (width) {
        case 1: return put_h264_chroma_mc1_8_c;
        case 2: return put_h264_chroma_mc2_8_c;
        case 4: return put_h264_chroma_mc4_8_c;
        case 8: return put_h264_chroma_mc8_8_c;
        }
    }
    return 0;
}

static chroma_fn chroma_func_high(int width, int avg, int bit_depth)
{
#define RETURN_CHROMA(depth) \
    do { \
        if (avg) { \
            switch (width) { \
            case 1: return avg_h264_chroma_mc1_ ## depth ## _c; \
            case 2: return avg_h264_chroma_mc2_ ## depth ## _c; \
            case 4: return avg_h264_chroma_mc4_ ## depth ## _c; \
            case 8: return avg_h264_chroma_mc8_ ## depth ## _c; \
            } \
        } else { \
            switch (width) { \
            case 1: return put_h264_chroma_mc1_ ## depth ## _c; \
            case 2: return put_h264_chroma_mc2_ ## depth ## _c; \
            case 4: return put_h264_chroma_mc4_ ## depth ## _c; \
            case 8: return put_h264_chroma_mc8_ ## depth ## _c; \
            } \
        } \
    } while (0)
    switch (bit_depth) {
    case 10:
        RETURN_CHROMA(10);
        break;
    case 12:
        RETURN_CHROMA(12);
        break;
    }
#undef RETURN_CHROMA
    return 0;
}

static weight_fn weight_func(int width)
{
    switch (width) {
    case 2:  return weight_h264_pixels2_8_c;
    case 4:  return weight_h264_pixels4_8_c;
    case 8:  return weight_h264_pixels8_8_c;
    case 16: return weight_h264_pixels16_8_c;
    }
    return 0;
}

static weight_fn weight_func_high(int width, int bit_depth)
{
#define RETURN_WEIGHT(depth) \
    do { \
        switch (width) { \
        case 2:  return weight_h264_pixels2_ ## depth ## _c; \
        case 4:  return weight_h264_pixels4_ ## depth ## _c; \
        case 8:  return weight_h264_pixels8_ ## depth ## _c; \
        case 16: return weight_h264_pixels16_ ## depth ## _c; \
        } \
    } while (0)
    switch (bit_depth) {
    case 10:
        RETURN_WEIGHT(10);
        break;
    case 12:
        RETURN_WEIGHT(12);
        break;
    }
#undef RETURN_WEIGHT
    return 0;
}

static biweight_fn biweight_func(int width)
{
    switch (width) {
    case 2:  return biweight_h264_pixels2_8_c;
    case 4:  return biweight_h264_pixels4_8_c;
    case 8:  return biweight_h264_pixels8_8_c;
    case 16: return biweight_h264_pixels16_8_c;
    }
    return 0;
}

static biweight_fn biweight_func_high(int width, int bit_depth)
{
#define RETURN_BIWEIGHT(depth) \
    do { \
        switch (width) { \
        case 2:  return biweight_h264_pixels2_ ## depth ## _c; \
        case 4:  return biweight_h264_pixels4_ ## depth ## _c; \
        case 8:  return biweight_h264_pixels8_ ## depth ## _c; \
        case 16: return biweight_h264_pixels16_ ## depth ## _c; \
        } \
    } while (0)
    switch (bit_depth) {
    case 10:
        RETURN_BIWEIGHT(10);
        break;
    case 12:
        RETURN_BIWEIGHT(12);
        break;
    }
#undef RETURN_BIWEIGHT
    return 0;
}

static int is_dir(uint32_t mb_type, int part, int list)
{
    if (list == 0) {
        if (part == 0)
            return !!(mb_type & MB_TYPE_P0L0);
        return !!(mb_type & MB_TYPE_P1L0);
    }
    if (part == 0)
        return !!(mb_type & MB_TYPE_P0L1);
    return !!(mb_type & MB_TYPE_P1L1);
}

static void fill_plane(uint8_t *p, int n, int seed)
{
    for (int i = 0; i < n; i++)
        p[i] = (uint8_t)((seed + i * 13 + (i >> 4) * 7) & 255);
}

static void fill_plane_high(uint16_t *p, int n, int seed, int bit_depth)
{
    int max = (1 << bit_depth) - 1;
    for (int i = 0; i < n; i++)
        p[i] = (uint16_t)((seed + i * 13 + (i >> 4) * 7) & max);
}

static void init_pic(Pic *p, int chroma_idc, int seed)
{
    memset(p, 0, sizeof(*p));
    p->chroma_idc = chroma_idc;
    p->chroma_stride = chroma_idc == 3 ? LUMA_STRIDE : CHROMA_STRIDE_420_422;
    fill_plane(p->y, sizeof(p->y), seed);
    fill_plane(p->cb, sizeof(p->cb), seed + 29);
    fill_plane(p->cr, sizeof(p->cr), seed + 71);
}

static void init_pic_high(PicHigh *p, int chroma_idc, int bit_depth, int seed)
{
    memset(p, 0, sizeof(*p));
    p->chroma_idc = chroma_idc;
    p->chroma_stride = chroma_idc == 3 ? LUMA_STRIDE : CHROMA_STRIDE_420_422;
    p->bit_depth = bit_depth;
    fill_plane_high(p->y, sizeof(p->y) / sizeof(p->y[0]), seed, bit_depth);
    fill_plane_high(p->cb, sizeof(p->cb) / sizeof(p->cb[0]), seed + 29, bit_depth);
    fill_plane_high(p->cr, sizeof(p->cr) / sizeof(p->cr[0]), seed + 71, bit_depth);
}

static void init_motion(MotionCtx *ctx)
{
    memset(ctx, 0, sizeof(*ctx));
    for (int list = 0; list < 2; list++)
        for (int i = 0; i < MOTION_CACHE_SIZE; i++)
            ctx->ref_cache[list][i] = -1;
}

static void init_pwt(PWT *pwt)
{
    memset(pwt, 0, sizeof(*pwt));
    for (int ref = 0; ref < 48; ref++) {
        for (int list = 0; list < 2; list++) {
            pwt->luma_weight[ref][list][0] = 1;
            pwt->luma_weight[ref][list][1] = 0;
            for (int c = 0; c < 2; c++) {
                pwt->chroma_weight[ref][list][c][0] = 1;
                pwt->chroma_weight[ref][list][c][1] = 0;
            }
        }
        for (int ref1 = 0; ref1 < 48; ref1++) {
            pwt->implicit_weight[ref][ref1][0] = 32;
            pwt->implicit_weight[ref][ref1][1] = 32;
        }
    }
}

static void set_ref_mv(MotionCtx *ctx, int list, int n, int ref, int mx, int my)
{
    int idx = scan8[n];
    ctx->ref_cache[list][idx] = (int8_t)ref;
    ctx->mv_cache[list][idx][0] = (int16_t)mx;
    ctx->mv_cache[list][idx][1] = (int16_t)my;
}

static uint8_t *pic_cb(Pic *p, int mb_x, int mb_y)
{
    if (p->chroma_idc == 1)
        return p->cb + mb_y * 8 * p->chroma_stride + mb_x * 8;
    if (p->chroma_idc == 2)
        return p->cb + mb_y * 16 * p->chroma_stride + mb_x * 8;
    return p->cb + mb_y * 16 * p->chroma_stride + mb_x * 16;
}

static uint8_t *pic_cr(Pic *p, int mb_x, int mb_y)
{
    if (p->chroma_idc == 1)
        return p->cr + mb_y * 8 * p->chroma_stride + mb_x * 8;
    if (p->chroma_idc == 2)
        return p->cr + mb_y * 16 * p->chroma_stride + mb_x * 8;
    return p->cr + mb_y * 16 * p->chroma_stride + mb_x * 16;
}

static uint16_t *pic_cb_high(PicHigh *p, int mb_x, int mb_y)
{
    if (p->chroma_idc == 1)
        return p->cb + mb_y * 8 * p->chroma_stride + mb_x * 8;
    if (p->chroma_idc == 2)
        return p->cb + mb_y * 16 * p->chroma_stride + mb_x * 8;
    return p->cb + mb_y * 16 * p->chroma_stride + mb_x * 16;
}

static uint16_t *pic_cr_high(PicHigh *p, int mb_x, int mb_y)
{
    if (p->chroma_idc == 1)
        return p->cr + mb_y * 8 * p->chroma_stride + mb_x * 8;
    if (p->chroma_idc == 2)
        return p->cr + mb_y * 16 * p->chroma_stride + mb_x * 8;
    return p->cr + mb_y * 16 * p->chroma_stride + mb_x * 16;
}

static void emulated_edge_mc_high(uint16_t *buf, const uint16_t *src,
                                  ptrdiff_t buf_stride, ptrdiff_t src_stride,
                                  int block_w, int block_h,
                                  int src_x, int src_y, int w, int h,
                                  int bit_depth)
{
    ptrdiff_t buf_stride_bytes = buf_stride * (ptrdiff_t)sizeof(uint16_t);
    ptrdiff_t src_stride_bytes = src_stride * (ptrdiff_t)sizeof(uint16_t);
    if (bit_depth == 10)
        ff_emulated_edge_mc_10((uint8_t *)buf, (const uint8_t *)src,
                               buf_stride_bytes, src_stride_bytes,
                               block_w, block_h, src_x, src_y, w, h);
    else
        ff_emulated_edge_mc_12((uint8_t *)buf, (const uint8_t *)src,
                               buf_stride_bytes, src_stride_bytes,
                               block_w, block_h, src_x, src_y, w, h);
}

static void mc_dir_part(Pic *dst, Pic *ref, MotionCtx *ctx,
                        int n, int square, int height, int delta, int list,
                        uint8_t *dest_y, uint8_t *dest_cb, uint8_t *dest_cr,
                        int src_x_offset, int src_y_offset,
                        int qpel_size, int chroma_width, int avg)
{
    int mx = ctx->mv_cache[list][scan8[n]][0] + src_x_offset * 8;
    int my = ctx->mv_cache[list][scan8[n]][1] + src_y_offset * 8;
    int luma_xy = (mx & 3) + ((my & 3) << 2);
    int offset = (mx >> 2) + (my >> 2) * LUMA_STRIDE;
    uint8_t *src_y = ref->y + offset;
    uint8_t edge_emu_buffer[LUMA_STRIDE * (16 + 5)];
    qpel_fn qpel = qpel_func(qpel_size, luma_xy, avg);
    int extra_width = 0;
    int extra_height = 0;
    int emu = 0;
    int full_mx = mx >> 2;
    int full_my = my >> 2;
    int pic_width = 16 * MB_WIDTH;
    int pic_height = 16 * MB_HEIGHT;

    if (mx & 7)
        extra_width -= 3;
    if (my & 7)
        extra_height -= 3;

    if (full_mx < 0 - extra_width ||
        full_my < 0 - extra_height ||
        full_mx + 16 > pic_width + extra_width ||
        full_my + 16 > pic_height + extra_height) {
        ff_emulated_edge_mc_8(edge_emu_buffer,
                                src_y - 2 - 2 * LUMA_STRIDE,
                                LUMA_STRIDE, LUMA_STRIDE,
                                16 + 5, 16 + 5, full_mx - 2,
                                full_my - 2, pic_width, pic_height);
        src_y = edge_emu_buffer + 2 + 2 * LUMA_STRIDE;
        emu = 1;
    }

    qpel(dest_y, src_y, LUMA_STRIDE);
    if (!square)
        qpel(dest_y + delta, src_y + delta, LUMA_STRIDE);

    if (dst->chroma_idc == 3) {
        uint8_t *src_cb = ref->cb + offset;
        uint8_t *src_cr = ref->cr + offset;
        if (emu) {
            ff_emulated_edge_mc_8(edge_emu_buffer,
                                    src_cb - 2 - 2 * LUMA_STRIDE,
                                    LUMA_STRIDE, LUMA_STRIDE,
                                    16 + 5, 16 + 5, full_mx - 2,
                                    full_my - 2, pic_width, pic_height);
            src_cb = edge_emu_buffer + 2 + 2 * LUMA_STRIDE;
        }
        qpel(dest_cb, src_cb, LUMA_STRIDE);
        if (!square)
            qpel(dest_cb + delta, src_cb + delta, LUMA_STRIDE);
        if (emu) {
            ff_emulated_edge_mc_8(edge_emu_buffer,
                                    src_cr - 2 - 2 * LUMA_STRIDE,
                                    LUMA_STRIDE, LUMA_STRIDE,
                                    16 + 5, 16 + 5, full_mx - 2,
                                    full_my - 2, pic_width, pic_height);
            src_cr = edge_emu_buffer + 2 + 2 * LUMA_STRIDE;
        }
        qpel(dest_cr, src_cr, LUMA_STRIDE);
        if (!square)
            qpel(dest_cr + delta, src_cr + delta, LUMA_STRIDE);
        return;
    }

    int ysh = 3 - (dst->chroma_idc == 2);
    uint8_t *src_cb = ref->cb + (mx >> 3) + (my >> ysh) * dst->chroma_stride;
    uint8_t *src_cr = ref->cr + (mx >> 3) + (my >> ysh) * dst->chroma_stride;
    chroma_fn chroma = chroma_func(chroma_width, avg);
    int chroma_h = height >> (dst->chroma_idc == 1);
    int chroma_y = ((unsigned)my << (dst->chroma_idc == 2)) & 7;

    if (emu) {
        ff_emulated_edge_mc_8(edge_emu_buffer, src_cb,
                                dst->chroma_stride, dst->chroma_stride,
                                9, 8 * dst->chroma_idc + 1, mx >> 3,
                                my >> ysh, pic_width >> 1,
                                pic_height >> (dst->chroma_idc == 1));
        src_cb = edge_emu_buffer;
    }
    chroma(dest_cb, src_cb, dst->chroma_stride, chroma_h, mx & 7, chroma_y);
    if (emu) {
        ff_emulated_edge_mc_8(edge_emu_buffer, src_cr,
                                dst->chroma_stride, dst->chroma_stride,
                                9, 8 * dst->chroma_idc + 1, mx >> 3,
                                my >> ysh, pic_width >> 1,
                                pic_height >> (dst->chroma_idc == 1));
        src_cr = edge_emu_buffer;
    }
    chroma(dest_cr, src_cr, dst->chroma_stride, chroma_h, mx & 7, chroma_y);
}

static void mc_dir_part_high(PicHigh *dst, PicHigh *ref, MotionCtx *ctx,
                             int n, int square, int height, int delta, int list,
                             uint16_t *dest_y, uint16_t *dest_cb, uint16_t *dest_cr,
                             int src_x_offset, int src_y_offset,
                             int qpel_size, int chroma_width, int avg)
{
    int mx = ctx->mv_cache[list][scan8[n]][0] + src_x_offset * 8;
    int my = ctx->mv_cache[list][scan8[n]][1] + src_y_offset * 8;
    int luma_xy = (mx & 3) + ((my & 3) << 2);
    int offset = (mx >> 2) + (my >> 2) * LUMA_STRIDE;
    uint16_t *src_y = ref->y + offset;
    uint16_t edge_emu_buffer[LUMA_STRIDE * (16 + 5)];
    qpel_fn qpel = qpel_func_high(qpel_size, luma_xy, avg, dst->bit_depth);
    ptrdiff_t luma_stride_bytes = LUMA_STRIDE * (ptrdiff_t)sizeof(uint16_t);
    ptrdiff_t chroma_stride_bytes = dst->chroma_stride * (ptrdiff_t)sizeof(uint16_t);
    int extra_width = 0;
    int extra_height = 0;
    int emu = 0;
    int full_mx = mx >> 2;
    int full_my = my >> 2;
    int pic_width = 16 * MB_WIDTH;
    int pic_height = 16 * MB_HEIGHT;

    if (mx & 7)
        extra_width -= 3;
    if (my & 7)
        extra_height -= 3;

    if (full_mx < 0 - extra_width ||
        full_my < 0 - extra_height ||
        full_mx + 16 > pic_width + extra_width ||
        full_my + 16 > pic_height + extra_height) {
        emulated_edge_mc_high(edge_emu_buffer,
                              src_y - 2 - 2 * LUMA_STRIDE,
                              LUMA_STRIDE, LUMA_STRIDE,
                              16 + 5, 16 + 5, full_mx - 2,
                              full_my - 2, pic_width, pic_height,
                              dst->bit_depth);
        src_y = edge_emu_buffer + 2 + 2 * LUMA_STRIDE;
        emu = 1;
    }

    qpel((uint8_t *)dest_y, (const uint8_t *)src_y, luma_stride_bytes);
    if (!square)
        qpel((uint8_t *)(dest_y + delta), (const uint8_t *)(src_y + delta),
             luma_stride_bytes);

    if (dst->chroma_idc == 3) {
        uint16_t *src_cb = ref->cb + offset;
        uint16_t *src_cr = ref->cr + offset;
        if (emu) {
            emulated_edge_mc_high(edge_emu_buffer,
                                  src_cb - 2 - 2 * LUMA_STRIDE,
                                  LUMA_STRIDE, LUMA_STRIDE,
                                  16 + 5, 16 + 5, full_mx - 2,
                                  full_my - 2, pic_width, pic_height,
                                  dst->bit_depth);
            src_cb = edge_emu_buffer + 2 + 2 * LUMA_STRIDE;
        }
        qpel((uint8_t *)dest_cb, (const uint8_t *)src_cb, chroma_stride_bytes);
        if (!square)
            qpel((uint8_t *)(dest_cb + delta), (const uint8_t *)(src_cb + delta),
                 chroma_stride_bytes);
        if (emu) {
            emulated_edge_mc_high(edge_emu_buffer,
                                  src_cr - 2 - 2 * LUMA_STRIDE,
                                  LUMA_STRIDE, LUMA_STRIDE,
                                  16 + 5, 16 + 5, full_mx - 2,
                                  full_my - 2, pic_width, pic_height,
                                  dst->bit_depth);
            src_cr = edge_emu_buffer + 2 + 2 * LUMA_STRIDE;
        }
        qpel((uint8_t *)dest_cr, (const uint8_t *)src_cr, chroma_stride_bytes);
        if (!square)
            qpel((uint8_t *)(dest_cr + delta), (const uint8_t *)(src_cr + delta),
                 chroma_stride_bytes);
        return;
    }

    int ysh = 3 - (dst->chroma_idc == 2);
    uint16_t *src_cb = ref->cb + (mx >> 3) + (my >> ysh) * dst->chroma_stride;
    uint16_t *src_cr = ref->cr + (mx >> 3) + (my >> ysh) * dst->chroma_stride;
    chroma_fn chroma = chroma_func_high(chroma_width, avg, dst->bit_depth);
    int chroma_h = height >> (dst->chroma_idc == 1);
    int chroma_y = ((unsigned)my << (dst->chroma_idc == 2)) & 7;

    if (emu) {
        emulated_edge_mc_high(edge_emu_buffer, src_cb,
                              dst->chroma_stride, dst->chroma_stride,
                              9, 8 * dst->chroma_idc + 1, mx >> 3,
                              my >> ysh, pic_width >> 1,
                              pic_height >> (dst->chroma_idc == 1),
                              dst->bit_depth);
        src_cb = edge_emu_buffer;
    }
    chroma((uint8_t *)dest_cb, (const uint8_t *)src_cb, chroma_stride_bytes,
           chroma_h, mx & 7, chroma_y);
    if (emu) {
        emulated_edge_mc_high(edge_emu_buffer, src_cr,
                              dst->chroma_stride, dst->chroma_stride,
                              9, 8 * dst->chroma_idc + 1, mx >> 3,
                              my >> ysh, pic_width >> 1,
                              pic_height >> (dst->chroma_idc == 1),
                              dst->bit_depth);
        src_cr = edge_emu_buffer;
    }
    chroma((uint8_t *)dest_cr, (const uint8_t *)src_cr, chroma_stride_bytes,
           chroma_h, mx & 7, chroma_y);
}

static void mc_part_std_high(PicHigh *dst, PicHigh *refs[2][2], MotionCtx *ctx,
                             int mb_x, int mb_y, int n, int part, int square,
                             int height, int delta, int x_offset, int y_offset,
                             int qpel_size, int chroma_width, int list0, int list1)
{
    uint16_t *dest_y = dst->y + mb_y * 16 * LUMA_STRIDE + mb_x * 16;
    uint16_t *dest_cb = pic_cb_high(dst, mb_x, mb_y);
    uint16_t *dest_cr = pic_cr_high(dst, mb_x, mb_y);
    int avg = 0;

    dest_y += 2 * x_offset + 2 * y_offset * LUMA_STRIDE;
    if (dst->chroma_idc == 3) {
        dest_cb += 2 * x_offset + 2 * y_offset * LUMA_STRIDE;
        dest_cr += 2 * x_offset + 2 * y_offset * LUMA_STRIDE;
    } else if (dst->chroma_idc == 2) {
        dest_cb += x_offset + 2 * y_offset * dst->chroma_stride;
        dest_cr += x_offset + 2 * y_offset * dst->chroma_stride;
    } else {
        dest_cb += x_offset + y_offset * dst->chroma_stride;
        dest_cr += x_offset + y_offset * dst->chroma_stride;
    }

    x_offset += 8 * mb_x;
    y_offset += 8 * mb_y;

    if (list0) {
        int refn = ctx->ref_cache[0][scan8[n]];
        mc_dir_part_high(dst, refs[0][refn], ctx, n, square, height, delta, 0,
                         dest_y, dest_cb, dest_cr, x_offset, y_offset,
                         qpel_size, chroma_width, avg);
        avg = 1;
    }
    if (list1) {
        int refn = ctx->ref_cache[1][scan8[n]];
        mc_dir_part_high(dst, refs[1][refn], ctx, n, square, height, delta, 1,
                         dest_y, dest_cb, dest_cr, x_offset, y_offset,
                         qpel_size, chroma_width, avg);
    }
}

static void mc_part_weighted_high(PicHigh *dst, PicHigh *refs[2][2],
                                  MotionCtx *ctx, PWT *pwt,
                                  int mb_x, int mb_y, int n, int part,
                                  int square, int height, int delta,
                                  int x_offset, int y_offset, int qpel_size,
                                  int chroma_width, int luma_width,
                                  int list0, int list1)
{
    uint16_t *dest_y = dst->y + mb_y * 16 * LUMA_STRIDE + mb_x * 16;
    uint16_t *dest_cb = pic_cb_high(dst, mb_x, mb_y);
    uint16_t *dest_cr = pic_cr_high(dst, mb_x, mb_y);
    int chroma_height;
    int chroma_weight_width = chroma_width;
    ptrdiff_t luma_stride_bytes = LUMA_STRIDE * (ptrdiff_t)sizeof(uint16_t);
    ptrdiff_t chroma_stride_bytes = dst->chroma_stride * (ptrdiff_t)sizeof(uint16_t);

    dest_y += 2 * x_offset + 2 * y_offset * LUMA_STRIDE;
    if (dst->chroma_idc == 3) {
        chroma_height = height;
        chroma_weight_width = luma_width;
        dest_cb += 2 * x_offset + 2 * y_offset * LUMA_STRIDE;
        dest_cr += 2 * x_offset + 2 * y_offset * LUMA_STRIDE;
    } else if (dst->chroma_idc == 2) {
        chroma_height = height;
        dest_cb += x_offset + 2 * y_offset * dst->chroma_stride;
        dest_cr += x_offset + 2 * y_offset * dst->chroma_stride;
    } else {
        chroma_height = height >> 1;
        dest_cb += x_offset + y_offset * dst->chroma_stride;
        dest_cr += x_offset + y_offset * dst->chroma_stride;
    }
    x_offset += 8 * mb_x;
    y_offset += 8 * mb_y;

    if (list0 && list1) {
        uint16_t tmp_y[LUMA_STRIDE * 16] = {0};
        uint16_t tmp_cb[LUMA_STRIDE * 16] = {0};
        uint16_t tmp_cr[LUMA_STRIDE * 16] = {0};
        int refn0 = ctx->ref_cache[0][scan8[n]];
        int refn1 = ctx->ref_cache[1][scan8[n]];

        mc_dir_part_high(dst, refs[0][refn0], ctx, n, square, height, delta, 0,
                         dest_y, dest_cb, dest_cr, x_offset, y_offset,
                         qpel_size, chroma_width, 0);
        mc_dir_part_high(dst, refs[1][refn1], ctx, n, square, height, delta, 1,
                         tmp_y, tmp_cb, tmp_cr, x_offset, y_offset,
                         qpel_size, chroma_width, 0);

        if (pwt->use_weight == 2) {
            int weight0 = pwt->implicit_weight[refn0][refn1][mb_y & 1];
            int weight1 = 64 - weight0;
            biweight_func_high(luma_width, dst->bit_depth)((uint8_t *)dest_y,
                                                          (uint8_t *)tmp_y,
                                                          luma_stride_bytes, height, 5,
                                                          weight0, weight1, 0);
            biweight_func_high(chroma_weight_width, dst->bit_depth)((uint8_t *)dest_cb,
                                                                    (uint8_t *)tmp_cb,
                                                                    chroma_stride_bytes,
                                                                    chroma_height, 5,
                                                                    weight0, weight1, 0);
            biweight_func_high(chroma_weight_width, dst->bit_depth)((uint8_t *)dest_cr,
                                                                    (uint8_t *)tmp_cr,
                                                                    chroma_stride_bytes,
                                                                    chroma_height, 5,
                                                                    weight0, weight1, 0);
        } else {
            biweight_func_high(luma_width, dst->bit_depth)((uint8_t *)dest_y,
                                                          (uint8_t *)tmp_y,
                                                          luma_stride_bytes, height,
                                                          pwt->luma_log2_weight_denom,
                                                          pwt->luma_weight[refn0][0][0],
                                                          pwt->luma_weight[refn1][1][0],
                                                          pwt->luma_weight[refn0][0][1] +
                                                          pwt->luma_weight[refn1][1][1]);
            biweight_func_high(chroma_weight_width, dst->bit_depth)((uint8_t *)dest_cb,
                                                                    (uint8_t *)tmp_cb,
                                                                    chroma_stride_bytes,
                                                                    chroma_height,
                                                                    pwt->chroma_log2_weight_denom,
                                                                    pwt->chroma_weight[refn0][0][0][0],
                                                                    pwt->chroma_weight[refn1][1][0][0],
                                                                    pwt->chroma_weight[refn0][0][0][1] +
                                                                    pwt->chroma_weight[refn1][1][0][1]);
            biweight_func_high(chroma_weight_width, dst->bit_depth)((uint8_t *)dest_cr,
                                                                    (uint8_t *)tmp_cr,
                                                                    chroma_stride_bytes,
                                                                    chroma_height,
                                                                    pwt->chroma_log2_weight_denom,
                                                                    pwt->chroma_weight[refn0][0][1][0],
                                                                    pwt->chroma_weight[refn1][1][1][0],
                                                                    pwt->chroma_weight[refn0][0][1][1] +
                                                                    pwt->chroma_weight[refn1][1][1][1]);
        }
    } else {
        int list = list1 ? 1 : 0;
        int refn = ctx->ref_cache[list][scan8[n]];
        mc_dir_part_high(dst, refs[list][refn], ctx, n, square, height, delta, list,
                         dest_y, dest_cb, dest_cr, x_offset, y_offset,
                         qpel_size, chroma_width, 0);

        weight_func_high(luma_width, dst->bit_depth)((uint8_t *)dest_y,
                                                     luma_stride_bytes, height,
                                                     pwt->luma_log2_weight_denom,
                                                     pwt->luma_weight[refn][list][0],
                                                     pwt->luma_weight[refn][list][1]);
        if (pwt->use_weight_chroma) {
            weight_func_high(chroma_weight_width, dst->bit_depth)((uint8_t *)dest_cb,
                                                                 chroma_stride_bytes,
                                                                 chroma_height,
                                                                 pwt->chroma_log2_weight_denom,
                                                                 pwt->chroma_weight[refn][list][0][0],
                                                                 pwt->chroma_weight[refn][list][0][1]);
            weight_func_high(chroma_weight_width, dst->bit_depth)((uint8_t *)dest_cr,
                                                                 chroma_stride_bytes,
                                                                 chroma_height,
                                                                 pwt->chroma_log2_weight_denom,
                                                                 pwt->chroma_weight[refn][list][1][0],
                                                                 pwt->chroma_weight[refn][list][1][1]);
        }
    }
}

static void hl_motion_high(PicHigh *dst, PicHigh *refs[2][2], MotionCtx *ctx,
                           uint32_t mb_type, uint32_t sub_mb_type[4],
                           int mb_x, int mb_y)
{
    if (mb_type & MB_TYPE_16x16) {
        mc_part_std_high(dst, refs, ctx, mb_x, mb_y, 0, 0, 1, 16, 0, 0, 0, 16, 8,
                         is_dir(mb_type, 0, 0), is_dir(mb_type, 0, 1));
    } else if (mb_type & MB_TYPE_16x8) {
        mc_part_std_high(dst, refs, ctx, mb_x, mb_y, 0, 0, 0, 8, 8, 0, 0, 8, 8,
                         is_dir(mb_type, 0, 0), is_dir(mb_type, 0, 1));
        mc_part_std_high(dst, refs, ctx, mb_x, mb_y, 8, 1, 0, 8, 8, 0, 4, 8, 8,
                         is_dir(mb_type, 1, 0), is_dir(mb_type, 1, 1));
    } else if (mb_type & MB_TYPE_8x16) {
        mc_part_std_high(dst, refs, ctx, mb_x, mb_y, 0, 0, 0, 16, 8 * LUMA_STRIDE, 0, 0, 8, 4,
                         is_dir(mb_type, 0, 0), is_dir(mb_type, 0, 1));
        mc_part_std_high(dst, refs, ctx, mb_x, mb_y, 4, 1, 0, 16, 8 * LUMA_STRIDE, 4, 0, 8, 4,
                         is_dir(mb_type, 1, 0), is_dir(mb_type, 1, 1));
    } else {
        for (int i = 0; i < 4; i++) {
            uint32_t sub = sub_mb_type[i];
            int n = 4 * i;
            int x_offset = (i & 1) << 2;
            int y_offset = (i & 2) << 1;
            if (sub & MB_TYPE_16x16) {
                mc_part_std_high(dst, refs, ctx, mb_x, mb_y, n, 0, 1, 8, 0, x_offset, y_offset, 8, 4,
                                 is_dir(sub, 0, 0), is_dir(sub, 0, 1));
            } else if (sub & MB_TYPE_16x8) {
                mc_part_std_high(dst, refs, ctx, mb_x, mb_y, n, 0, 0, 4, 4, x_offset, y_offset, 4, 4,
                                 is_dir(sub, 0, 0), is_dir(sub, 0, 1));
                mc_part_std_high(dst, refs, ctx, mb_x, mb_y, n + 2, 0, 0, 4, 4, x_offset, y_offset + 2, 4, 4,
                                 is_dir(sub, 0, 0), is_dir(sub, 0, 1));
            } else if (sub & MB_TYPE_8x16) {
                mc_part_std_high(dst, refs, ctx, mb_x, mb_y, n, 0, 0, 8, 4 * LUMA_STRIDE, x_offset, y_offset, 4, 2,
                                 is_dir(sub, 0, 0), is_dir(sub, 0, 1));
                mc_part_std_high(dst, refs, ctx, mb_x, mb_y, n + 1, 0, 0, 8, 4 * LUMA_STRIDE, x_offset + 2, y_offset, 4, 2,
                                 is_dir(sub, 0, 0), is_dir(sub, 0, 1));
            } else {
                for (int j = 0; j < 4; j++) {
                    int sub_x_offset = x_offset + 2 * (j & 1);
                    int sub_y_offset = y_offset + (j & 2);
                    mc_part_std_high(dst, refs, ctx, mb_x, mb_y, n + j, 0, 1, 4, 0, sub_x_offset, sub_y_offset, 4, 2,
                                     is_dir(sub, 0, 0), is_dir(sub, 0, 1));
                }
            }
        }
    }
}

static void mc_part_std(Pic *dst, Pic *refs[2][2], MotionCtx *ctx,
                        int mb_x, int mb_y, int n, int part, int square,
                        int height, int delta, int x_offset, int y_offset,
                        int qpel_size, int chroma_width, int list0, int list1)
{
    uint8_t *dest_y = dst->y + mb_y * 16 * LUMA_STRIDE + mb_x * 16;
    uint8_t *dest_cb = pic_cb(dst, mb_x, mb_y);
    uint8_t *dest_cr = pic_cr(dst, mb_x, mb_y);
    int avg = 0;

    dest_y += 2 * x_offset + 2 * y_offset * LUMA_STRIDE;
    if (dst->chroma_idc == 3) {
        dest_cb += 2 * x_offset + 2 * y_offset * LUMA_STRIDE;
        dest_cr += 2 * x_offset + 2 * y_offset * LUMA_STRIDE;
    } else if (dst->chroma_idc == 2) {
        dest_cb += x_offset + 2 * y_offset * dst->chroma_stride;
        dest_cr += x_offset + 2 * y_offset * dst->chroma_stride;
    } else {
        dest_cb += x_offset + y_offset * dst->chroma_stride;
        dest_cr += x_offset + y_offset * dst->chroma_stride;
    }

    x_offset += 8 * mb_x;
    y_offset += 8 * mb_y;

    if (list0) {
        int refn = ctx->ref_cache[0][scan8[n]];
        mc_dir_part(dst, refs[0][refn], ctx, n, square, height, delta, 0,
                    dest_y, dest_cb, dest_cr, x_offset, y_offset,
                    qpel_size, chroma_width, avg);
        avg = 1;
    }
    if (list1) {
        int refn = ctx->ref_cache[1][scan8[n]];
        mc_dir_part(dst, refs[1][refn], ctx, n, square, height, delta, 1,
                    dest_y, dest_cb, dest_cr, x_offset, y_offset,
                    qpel_size, chroma_width, avg);
    }
}

static void mc_part_weighted(Pic *dst, Pic *refs[2][2], MotionCtx *ctx, PWT *pwt,
                             int mb_x, int mb_y, int n, int part, int square,
                             int height, int delta, int x_offset, int y_offset,
                             int qpel_size, int chroma_width, int luma_width,
                             int list0, int list1)
{
    uint8_t *dest_y = dst->y + mb_y * 16 * LUMA_STRIDE + mb_x * 16;
    uint8_t *dest_cb = pic_cb(dst, mb_x, mb_y);
    uint8_t *dest_cr = pic_cr(dst, mb_x, mb_y);
    int chroma_height;
    int chroma_weight_width = chroma_width;

    dest_y += 2 * x_offset + 2 * y_offset * LUMA_STRIDE;
    if (dst->chroma_idc == 3) {
        chroma_height = height;
        chroma_weight_width = luma_width;
        dest_cb += 2 * x_offset + 2 * y_offset * LUMA_STRIDE;
        dest_cr += 2 * x_offset + 2 * y_offset * LUMA_STRIDE;
    } else if (dst->chroma_idc == 2) {
        chroma_height = height;
        dest_cb += x_offset + 2 * y_offset * dst->chroma_stride;
        dest_cr += x_offset + 2 * y_offset * dst->chroma_stride;
    } else {
        chroma_height = height >> 1;
        dest_cb += x_offset + y_offset * dst->chroma_stride;
        dest_cr += x_offset + y_offset * dst->chroma_stride;
    }
    x_offset += 8 * mb_x;
    y_offset += 8 * mb_y;

    if (list0 && list1) {
        uint8_t tmp_y[LUMA_STRIDE * 16] = {0};
        uint8_t tmp_cb[LUMA_STRIDE * 16] = {0};
        uint8_t tmp_cr[LUMA_STRIDE * 16] = {0};
        int refn0 = ctx->ref_cache[0][scan8[n]];
        int refn1 = ctx->ref_cache[1][scan8[n]];

        mc_dir_part(dst, refs[0][refn0], ctx, n, square, height, delta, 0,
                    dest_y, dest_cb, dest_cr, x_offset, y_offset,
                    qpel_size, chroma_width, 0);
        mc_dir_part(dst, refs[1][refn1], ctx, n, square, height, delta, 1,
                    tmp_y, tmp_cb, tmp_cr, x_offset, y_offset,
                    qpel_size, chroma_width, 0);

        if (pwt->use_weight == 2) {
            int weight0 = pwt->implicit_weight[refn0][refn1][mb_y & 1];
            int weight1 = 64 - weight0;
            biweight_func(luma_width)(dest_y, tmp_y, LUMA_STRIDE, height, 5,
                                      weight0, weight1, 0);
            biweight_func(chroma_weight_width)(dest_cb, tmp_cb, dst->chroma_stride,
                                               chroma_height, 5, weight0, weight1, 0);
            biweight_func(chroma_weight_width)(dest_cr, tmp_cr, dst->chroma_stride,
                                               chroma_height, 5, weight0, weight1, 0);
        } else {
            biweight_func(luma_width)(dest_y, tmp_y, LUMA_STRIDE, height,
                                      pwt->luma_log2_weight_denom,
                                      pwt->luma_weight[refn0][0][0],
                                      pwt->luma_weight[refn1][1][0],
                                      pwt->luma_weight[refn0][0][1] +
                                      pwt->luma_weight[refn1][1][1]);
            biweight_func(chroma_weight_width)(dest_cb, tmp_cb, dst->chroma_stride,
                                               chroma_height,
                                               pwt->chroma_log2_weight_denom,
                                               pwt->chroma_weight[refn0][0][0][0],
                                               pwt->chroma_weight[refn1][1][0][0],
                                               pwt->chroma_weight[refn0][0][0][1] +
                                               pwt->chroma_weight[refn1][1][0][1]);
            biweight_func(chroma_weight_width)(dest_cr, tmp_cr, dst->chroma_stride,
                                               chroma_height,
                                               pwt->chroma_log2_weight_denom,
                                               pwt->chroma_weight[refn0][0][1][0],
                                               pwt->chroma_weight[refn1][1][1][0],
                                               pwt->chroma_weight[refn0][0][1][1] +
                                               pwt->chroma_weight[refn1][1][1][1]);
        }
    } else {
        int list = list1 ? 1 : 0;
        int refn = ctx->ref_cache[list][scan8[n]];
        mc_dir_part(dst, refs[list][refn], ctx, n, square, height, delta, list,
                    dest_y, dest_cb, dest_cr, x_offset, y_offset,
                    qpel_size, chroma_width, 0);

        weight_func(luma_width)(dest_y, LUMA_STRIDE, height,
                                pwt->luma_log2_weight_denom,
                                pwt->luma_weight[refn][list][0],
                                pwt->luma_weight[refn][list][1]);
        if (pwt->use_weight_chroma) {
            weight_func(chroma_weight_width)(dest_cb, dst->chroma_stride,
                                             chroma_height,
                                             pwt->chroma_log2_weight_denom,
                                             pwt->chroma_weight[refn][list][0][0],
                                             pwt->chroma_weight[refn][list][0][1]);
            weight_func(chroma_weight_width)(dest_cr, dst->chroma_stride,
                                             chroma_height,
                                             pwt->chroma_log2_weight_denom,
                                             pwt->chroma_weight[refn][list][1][0],
                                             pwt->chroma_weight[refn][list][1][1]);
        }
    }
}

static void hl_motion(Pic *dst, Pic *refs[2][2], MotionCtx *ctx,
                      uint32_t mb_type, uint32_t sub_mb_type[4],
                      int mb_x, int mb_y)
{
    if (mb_type & MB_TYPE_16x16) {
        mc_part_std(dst, refs, ctx, mb_x, mb_y, 0, 0, 1, 16, 0, 0, 0, 16, 8,
                    is_dir(mb_type, 0, 0), is_dir(mb_type, 0, 1));
    } else if (mb_type & MB_TYPE_16x8) {
        mc_part_std(dst, refs, ctx, mb_x, mb_y, 0, 0, 0, 8, 8, 0, 0, 8, 8,
                    is_dir(mb_type, 0, 0), is_dir(mb_type, 0, 1));
        mc_part_std(dst, refs, ctx, mb_x, mb_y, 8, 1, 0, 8, 8, 0, 4, 8, 8,
                    is_dir(mb_type, 1, 0), is_dir(mb_type, 1, 1));
    } else if (mb_type & MB_TYPE_8x16) {
        mc_part_std(dst, refs, ctx, mb_x, mb_y, 0, 0, 0, 16, 8 * LUMA_STRIDE, 0, 0, 8, 4,
                    is_dir(mb_type, 0, 0), is_dir(mb_type, 0, 1));
        mc_part_std(dst, refs, ctx, mb_x, mb_y, 4, 1, 0, 16, 8 * LUMA_STRIDE, 4, 0, 8, 4,
                    is_dir(mb_type, 1, 0), is_dir(mb_type, 1, 1));
    } else {
        for (int i = 0; i < 4; i++) {
            uint32_t sub = sub_mb_type[i];
            int n = 4 * i;
            int x_offset = (i & 1) << 2;
            int y_offset = (i & 2) << 1;
            if (sub & MB_TYPE_16x16) {
                mc_part_std(dst, refs, ctx, mb_x, mb_y, n, 0, 1, 8, 0, x_offset, y_offset, 8, 4,
                            is_dir(sub, 0, 0), is_dir(sub, 0, 1));
            } else if (sub & MB_TYPE_16x8) {
                mc_part_std(dst, refs, ctx, mb_x, mb_y, n, 0, 0, 4, 4, x_offset, y_offset, 4, 4,
                            is_dir(sub, 0, 0), is_dir(sub, 0, 1));
                mc_part_std(dst, refs, ctx, mb_x, mb_y, n + 2, 0, 0, 4, 4, x_offset, y_offset + 2, 4, 4,
                            is_dir(sub, 0, 0), is_dir(sub, 0, 1));
            } else if (sub & MB_TYPE_8x16) {
                mc_part_std(dst, refs, ctx, mb_x, mb_y, n, 0, 0, 8, 4 * LUMA_STRIDE, x_offset, y_offset, 4, 2,
                            is_dir(sub, 0, 0), is_dir(sub, 0, 1));
                mc_part_std(dst, refs, ctx, mb_x, mb_y, n + 1, 0, 0, 8, 4 * LUMA_STRIDE, x_offset + 2, y_offset, 4, 2,
                            is_dir(sub, 0, 0), is_dir(sub, 0, 1));
            } else {
                for (int j = 0; j < 4; j++) {
                    int sub_x_offset = x_offset + 2 * (j & 1);
                    int sub_y_offset = y_offset + (j & 2);
                    mc_part_std(dst, refs, ctx, mb_x, mb_y, n + j, 0, 1, 4, 0, sub_x_offset, sub_y_offset, 4, 2,
                                is_dir(sub, 0, 0), is_dir(sub, 0, 1));
                }
            }
        }
    }
}

static void print_mb(const char *label, Pic *dst, int mb_x, int mb_y)
{
    int yoff = mb_y * 16 * LUMA_STRIDE + mb_x * 16;
    printf("%s y", label);
    for (int y = 0; y < 16; y++)
        for (int x = 0; x < 16; x++)
            printf(" %u", dst->y[yoff + y * LUMA_STRIDE + x]);
    printf("\n");

    int cw = dst->chroma_idc == 3 ? 16 : 8;
    int ch = dst->chroma_idc == 1 ? 8 : 16;
    uint8_t *cb = pic_cb(dst, mb_x, mb_y);
    uint8_t *cr = pic_cr(dst, mb_x, mb_y);
    printf("%s cb", label);
    for (int y = 0; y < ch; y++)
        for (int x = 0; x < cw; x++)
            printf(" %u", cb[y * dst->chroma_stride + x]);
    printf("\n");
    printf("%s cr", label);
    for (int y = 0; y < ch; y++)
        for (int x = 0; x < cw; x++)
            printf(" %u", cr[y * dst->chroma_stride + x]);
    printf("\n");
}

static void print_mb_high(const char *label, PicHigh *dst, int mb_x, int mb_y)
{
    int yoff = mb_y * 16 * LUMA_STRIDE + mb_x * 16;
    printf("%s y", label);
    for (int y = 0; y < 16; y++)
        for (int x = 0; x < 16; x++)
            printf(" %u", dst->y[yoff + y * LUMA_STRIDE + x]);
    printf("\n");

    int cw = dst->chroma_idc == 3 ? 16 : 8;
    int ch = dst->chroma_idc == 1 ? 8 : 16;
    uint16_t *cb = pic_cb_high(dst, mb_x, mb_y);
    uint16_t *cr = pic_cr_high(dst, mb_x, mb_y);
    printf("%s cb", label);
    for (int y = 0; y < ch; y++)
        for (int x = 0; x < cw; x++)
            printf(" %u", cb[y * dst->chroma_stride + x]);
    printf("\n");
    printf("%s cr", label);
    for (int y = 0; y < ch; y++)
        for (int x = 0; x < cw; x++)
            printf(" %u", cr[y * dst->chroma_stride + x]);
    printf("\n");
}

static void run_p16x16_420(void)
{
    Pic dst, ref0;
    Pic *refs[2][2] = {{0}};
    MotionCtx ctx;
    uint32_t sub[4] = {0};
    init_pic(&dst, 1, 3);
    init_pic(&ref0, 1, 41);
    refs[0][0] = &ref0;
    init_motion(&ctx);
    set_ref_mv(&ctx, 0, 0, 0, 5, 7);
    hl_motion(&dst, refs, &ctx, MB_TYPE_16x16 | MB_TYPE_P0L0, sub, 1, 1);
    print_mb("p16x16_420", &dst, 1, 1);
}

static void run_b16x8_420(void)
{
    Pic dst, ref0, ref1;
    Pic *refs[2][2] = {{0}};
    MotionCtx ctx;
    uint32_t sub[4] = {0};
    init_pic(&dst, 1, 9);
    init_pic(&ref0, 1, 51);
    init_pic(&ref1, 1, 101);
    refs[0][0] = &ref0;
    refs[1][0] = &ref1;
    init_motion(&ctx);
    set_ref_mv(&ctx, 0, 0, 0, 4, 6);
    set_ref_mv(&ctx, 1, 0, 0, 12, 14);
    set_ref_mv(&ctx, 0, 8, 0, 8, 10);
    set_ref_mv(&ctx, 1, 8, 0, 16, 18);
    hl_motion(&dst, refs, &ctx,
              MB_TYPE_16x8 | MB_TYPE_P0L0 | MB_TYPE_P1L0 | MB_TYPE_P0L1 | MB_TYPE_P1L1,
              sub, 1, 1);
    print_mb("b16x8_420", &dst, 1, 1);
}

static void run_p8x16_422(void)
{
    Pic dst, ref0;
    Pic *refs[2][2] = {{0}};
    MotionCtx ctx;
    uint32_t sub[4] = {0};
    init_pic(&dst, 2, 19);
    init_pic(&ref0, 2, 73);
    refs[0][0] = &ref0;
    init_motion(&ctx);
    set_ref_mv(&ctx, 0, 0, 0, 3, 5);
    set_ref_mv(&ctx, 0, 4, 0, 9, 13);
    hl_motion(&dst, refs, &ctx, MB_TYPE_8x16 | MB_TYPE_P0L0 | MB_TYPE_P1L0, sub, 1, 1);
    print_mb("p8x16_422", &dst, 1, 1);
}

static void run_sub8x8_420(void)
{
    Pic dst, ref0;
    Pic *refs[2][2] = {{0}};
    MotionCtx ctx;
    uint32_t sub[4] = {
        MB_TYPE_16x16 | MB_TYPE_P0L0,
        MB_TYPE_16x8 | MB_TYPE_P0L0,
        MB_TYPE_8x16 | MB_TYPE_P0L0,
        MB_TYPE_8x8 | MB_TYPE_P0L0,
    };
    init_pic(&dst, 1, 27);
    init_pic(&ref0, 1, 111);
    refs[0][0] = &ref0;
    init_motion(&ctx);
    for (int n = 0; n < 16; n++)
        set_ref_mv(&ctx, 0, n, 0, (n % 5) + 1, (n % 7) + 2);
    hl_motion(&dst, refs, &ctx, MB_TYPE_8x8 | MB_TYPE_P0L0 | MB_TYPE_P1L0, sub, 1, 1);
    print_mb("sub8x8_420", &dst, 1, 1);
}

static void run_weighted_p16x16_420(void)
{
    Pic dst, ref0;
    Pic *refs[2][2] = {{0}};
    MotionCtx ctx;
    PWT pwt;
    init_pic(&dst, 1, 23);
    init_pic(&ref0, 1, 91);
    refs[0][0] = &ref0;
    init_motion(&ctx);
    init_pwt(&pwt);
    set_ref_mv(&ctx, 0, 0, 0, 0, 0);
    pwt.use_weight = 1;
    pwt.use_weight_chroma = 1;
    pwt.luma_log2_weight_denom = 2;
    pwt.chroma_log2_weight_denom = 1;
    pwt.luma_weight[0][0][0] = 3;
    pwt.luma_weight[0][0][1] = -2;
    pwt.chroma_weight[0][0][0][0] = 2;
    pwt.chroma_weight[0][0][0][1] = 1;
    pwt.chroma_weight[0][0][1][0] = -1;
    pwt.chroma_weight[0][0][1][1] = 3;

    mc_part_weighted(&dst, refs, &ctx, &pwt, 1, 1, 0, 0, 1, 16, 0, 0, 0,
                     16, 8, 16, 1, 0);
    print_mb("weighted_p16x16_420", &dst, 1, 1);
}

static void run_weighted_implicit_b16x8_422(void)
{
    Pic dst, ref0, ref1;
    Pic *refs[2][2] = {{0}};
    MotionCtx ctx;
    PWT pwt;
    init_pic(&dst, 2, 37);
    init_pic(&ref0, 2, 79);
    init_pic(&ref1, 2, 119);
    refs[0][0] = &ref0;
    refs[1][0] = &ref1;
    init_motion(&ctx);
    init_pwt(&pwt);
    set_ref_mv(&ctx, 0, 0, 0, 4, 2);
    set_ref_mv(&ctx, 1, 0, 0, 8, 6);
    set_ref_mv(&ctx, 0, 8, 0, 12, 10);
    set_ref_mv(&ctx, 1, 8, 0, 16, 14);
    pwt.use_weight = 2;
    pwt.implicit_weight[0][0][1] = 21;

    mc_part_weighted(&dst, refs, &ctx, &pwt, 1, 1, 0, 0, 0, 8, 8, 0, 0,
                     8, 8, 16, 1, 1);
    mc_part_weighted(&dst, refs, &ctx, &pwt, 1, 1, 8, 1, 0, 8, 8, 0, 4,
                     8, 8, 16, 1, 1);
    print_mb("weighted_implicit_b16x8_422", &dst, 1, 1);
}

static void run_edge_p16x16_420(void)
{
    Pic dst, ref0;
    Pic *refs[2][2] = {{0}};
    MotionCtx ctx;
    uint32_t sub[4] = {0};
    init_pic(&dst, 1, 43);
    init_pic(&ref0, 1, 131);
    refs[0][0] = &ref0;
    init_motion(&ctx);
    set_ref_mv(&ctx, 0, 0, 0, 1, 0);
    hl_motion(&dst, refs, &ctx, MB_TYPE_16x16 | MB_TYPE_P0L0, sub, 3, 1);
    print_mb("edge_p16x16_420", &dst, 3, 1);
}

static void run_high10_p16x16_420(void)
{
    PicHigh dst, ref0;
    PicHigh *refs[2][2] = {{0}};
    MotionCtx ctx;
    uint32_t sub[4] = {0};
    init_pic_high(&dst, 1, 10, 203);
    init_pic_high(&ref0, 1, 10, 241);
    refs[0][0] = &ref0;
    init_motion(&ctx);
    set_ref_mv(&ctx, 0, 0, 0, 5, 7);
    hl_motion_high(&dst, refs, &ctx, MB_TYPE_16x16 | MB_TYPE_P0L0, sub, 1, 1);
    print_mb_high("high10_p16x16_420", &dst, 1, 1);
}

static void run_high12_b16x8_444(void)
{
    PicHigh dst, ref0, ref1;
    PicHigh *refs[2][2] = {{0}};
    MotionCtx ctx;
    uint32_t sub[4] = {0};
    init_pic_high(&dst, 3, 12, 309);
    init_pic_high(&ref0, 3, 12, 351);
    init_pic_high(&ref1, 3, 12, 401);
    refs[0][0] = &ref0;
    refs[1][0] = &ref1;
    init_motion(&ctx);
    set_ref_mv(&ctx, 0, 0, 0, 4, 6);
    set_ref_mv(&ctx, 1, 0, 0, 12, 14);
    set_ref_mv(&ctx, 0, 8, 0, 8, 10);
    set_ref_mv(&ctx, 1, 8, 0, 16, 18);
    hl_motion_high(&dst, refs, &ctx,
                   MB_TYPE_16x8 | MB_TYPE_P0L0 | MB_TYPE_P1L0 | MB_TYPE_P0L1 | MB_TYPE_P1L1,
                   sub, 1, 1);
    print_mb_high("high12_b16x8_444", &dst, 1, 1);
}

static void run_high10_weighted_p16x16_420(void)
{
    PicHigh dst, ref0;
    PicHigh *refs[2][2] = {{0}};
    MotionCtx ctx;
    PWT pwt;
    init_pic_high(&dst, 1, 10, 223);
    init_pic_high(&ref0, 1, 10, 291);
    refs[0][0] = &ref0;
    init_motion(&ctx);
    init_pwt(&pwt);
    set_ref_mv(&ctx, 0, 0, 0, 0, 0);
    pwt.use_weight = 1;
    pwt.use_weight_chroma = 1;
    pwt.luma_log2_weight_denom = 2;
    pwt.chroma_log2_weight_denom = 1;
    pwt.luma_weight[0][0][0] = 3;
    pwt.luma_weight[0][0][1] = -2;
    pwt.chroma_weight[0][0][0][0] = 2;
    pwt.chroma_weight[0][0][0][1] = 1;
    pwt.chroma_weight[0][0][1][0] = -1;
    pwt.chroma_weight[0][0][1][1] = 3;

    mc_part_weighted_high(&dst, refs, &ctx, &pwt, 1, 1, 0, 0, 1, 16, 0, 0, 0,
                          16, 8, 16, 1, 0);
    print_mb_high("high10_weighted_p16x16_420", &dst, 1, 1);
}

static void run_high12_weighted_p16x8_420(void)
{
    PicHigh dst, ref0;
    PicHigh *refs[2][2] = {{0}};
    MotionCtx ctx;
    PWT pwt;
    init_pic_high(&dst, 1, 12, 457);
    init_pic_high(&ref0, 1, 12, 509);
    refs[0][0] = &ref0;
    init_motion(&ctx);
    init_pwt(&pwt);
    set_ref_mv(&ctx, 0, 0, 0, 3, 5);
    set_ref_mv(&ctx, 0, 8, 0, 7, 11);
    pwt.use_weight = 1;
    pwt.use_weight_chroma = 1;
    pwt.luma_log2_weight_denom = 2;
    pwt.chroma_log2_weight_denom = 1;
    pwt.luma_weight[0][0][0] = 3;
    pwt.luma_weight[0][0][1] = -2;
    pwt.chroma_weight[0][0][0][0] = 2;
    pwt.chroma_weight[0][0][0][1] = 1;
    pwt.chroma_weight[0][0][1][0] = -1;
    pwt.chroma_weight[0][0][1][1] = 3;

    mc_part_weighted_high(&dst, refs, &ctx, &pwt, 1, 1, 0, 0, 0, 8, 8, 0, 0,
                          8, 8, 16, 1, 0);
    mc_part_weighted_high(&dst, refs, &ctx, &pwt, 1, 1, 8, 1, 0, 8, 8, 0, 4,
                          8, 8, 16, 1, 0);
    print_mb_high("high12_weighted_p16x8_420", &dst, 1, 1);
}

static void run_high12_weighted_implicit_b16x8_422(void)
{
    PicHigh dst, ref0, ref1;
    PicHigh *refs[2][2] = {{0}};
    MotionCtx ctx;
    PWT pwt;
    init_pic_high(&dst, 2, 12, 337);
    init_pic_high(&ref0, 2, 12, 379);
    init_pic_high(&ref1, 2, 12, 419);
    refs[0][0] = &ref0;
    refs[1][0] = &ref1;
    init_motion(&ctx);
    init_pwt(&pwt);
    set_ref_mv(&ctx, 0, 0, 0, 4, 2);
    set_ref_mv(&ctx, 1, 0, 0, 8, 6);
    set_ref_mv(&ctx, 0, 8, 0, 12, 10);
    set_ref_mv(&ctx, 1, 8, 0, 16, 14);
    pwt.use_weight = 2;
    pwt.implicit_weight[0][0][1] = 21;

    mc_part_weighted_high(&dst, refs, &ctx, &pwt, 1, 1, 0, 0, 0, 8, 8, 0, 0,
                          8, 8, 16, 1, 1);
    mc_part_weighted_high(&dst, refs, &ctx, &pwt, 1, 1, 8, 1, 0, 8, 8, 0, 4,
                          8, 8, 16, 1, 1);
    print_mb_high("high12_weighted_implicit_b16x8_422", &dst, 1, 1);
}

static void run_high12_edge_p16x16_420(void)
{
    PicHigh dst, ref0;
    PicHigh *refs[2][2] = {{0}};
    MotionCtx ctx;
    uint32_t sub[4] = {0};
    init_pic_high(&dst, 1, 12, 343);
    init_pic_high(&ref0, 1, 12, 431);
    refs[0][0] = &ref0;
    init_motion(&ctx);
    set_ref_mv(&ctx, 0, 0, 0, 1, 0);
    hl_motion_high(&dst, refs, &ctx, MB_TYPE_16x16 | MB_TYPE_P0L0, sub, 3, 1);
    print_mb_high("high12_edge_p16x16_420", &dst, 3, 1);
}

int main(void)
{
    run_p16x16_420();
    run_b16x8_420();
    run_p8x16_422();
    run_sub8x8_420();
    run_weighted_p16x16_420();
    run_weighted_implicit_b16x8_422();
    run_edge_p16x16_420();
    run_high10_p16x16_420();
    run_high12_b16x8_444();
    run_high10_weighted_p16x16_420();
    run_high12_weighted_p16x8_420();
    run_high12_weighted_implicit_b16x8_422();
    run_high12_edge_p16x16_420();
    return 0;
}
`

func TestH264MotionCompUpstreamOracle(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run pinned FFmpeg H.264 motion-comp call-site oracle")
	}
	cc := os.Getenv("CC")
	if cc == "" {
		cc = "cc"
	}
	root := h264RepoRoot(t)
	upstreamDir := filepath.Join(root, ".upstream", "ffmpeg-n8.0.1", "libavcodec")

	qpelTemplate, err := os.ReadFile(filepath.Join(upstreamDir, "h264qpel_template.c"))
	if err != nil {
		t.Fatal(err)
	}
	hpelTemplate, err := os.ReadFile(filepath.Join(upstreamDir, "hpel_template.c"))
	if err != nil {
		t.Fatal(err)
	}
	pelTemplate, err := os.ReadFile(filepath.Join(upstreamDir, "pel_template.c"))
	if err != nil {
		t.Fatal(err)
	}
	pixelsH, err := os.ReadFile(filepath.Join(upstreamDir, "pixels.h"))
	if err != nil {
		t.Fatal(err)
	}
	rndAvgH, err := os.ReadFile(filepath.Join(upstreamDir, "rnd_avg.h"))
	if err != nil {
		t.Fatal(err)
	}
	bitDepthTemplate, err := os.ReadFile(filepath.Join(upstreamDir, "bit_depth_template.c"))
	if err != nil {
		t.Fatal(err)
	}
	chromaTemplate, err := os.ReadFile(filepath.Join(upstreamDir, "h264chroma_template.c"))
	if err != nil {
		t.Fatal(err)
	}
	dspTemplate, err := os.ReadFile(filepath.Join(upstreamDir, "h264dsp_template.c"))
	if err != nil {
		t.Fatal(err)
	}
	videoDSPTemplate, err := os.ReadFile(filepath.Join(upstreamDir, "videodsp_template.c"))
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	writeOracleFile(t, filepath.Join(dir, "oracle.c"), motionCompOracleC)
	writeOracleFile(t, filepath.Join(dir, "h264qpel_template.c"), string(qpelTemplate))
	writeOracleFile(t, filepath.Join(dir, "hpel_template.c"), string(hpelTemplate))
	writeOracleFile(t, filepath.Join(dir, "pel_template.c"), string(pelTemplate))
	writeOracleFile(t, filepath.Join(dir, "pixels.h"), string(pixelsH))
	writeOracleFile(t, filepath.Join(dir, "rnd_avg.h"), string(rndAvgH))
	writeOracleFile(t, filepath.Join(dir, "bit_depth_template.c"), string(bitDepthTemplate))
	writeOracleFile(t, filepath.Join(dir, "h264chroma_template.c"), string(chromaTemplate))
	writeOracleFile(t, filepath.Join(dir, "h264dsp_template.c"), string(dspTemplate))
	writeOracleFile(t, filepath.Join(dir, "videodsp_template.c"), string(videoDSPTemplate))
	writeOracleFile(t, filepath.Join(dir, "mathops.h"), "")
	if err := os.Mkdir(filepath.Join(dir, "libavutil"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeOracleFile(t, filepath.Join(dir, "libavutil", "common.h"), qpelOracleCommonH)
	writeOracleFile(t, filepath.Join(dir, "libavutil", "intreadwrite.h"), qpelOracleIntreadwriteH)
	writeOracleFile(t, filepath.Join(dir, "libavutil", "avassert.h"), "#define av_assert2(cond) do { } while (0)\n")

	bin := filepath.Join(dir, "oracle")
	cmd := exec.Command(cc, "-std=c99", "-I"+dir, filepath.Join(dir, "oracle.c"), "-o", bin)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("compile H.264 motion-comp oracle: %v\n%s", err, out)
	}
	out, err := exec.Command(bin).CombinedOutput()
	if err != nil {
		t.Fatalf("run H.264 motion-comp oracle: %v\n%s", err, out)
	}

	got := strings.TrimSpace(string(out))
	want := strings.TrimSpace(h264MotionCompOracleWant(t))
	if got != want {
		t.Fatalf("H.264 motion-comp oracle mismatch\nC oracle:\n%s\nGo:\n%s", got, want)
	}
}

func h264MotionCompOracleWant(t *testing.T) string {
	var b strings.Builder
	appendH264MotionCompOracleP16x16(t, &b)
	appendH264MotionCompOracleB16x8(t, &b)
	appendH264MotionCompOracleP8x16(t, &b)
	appendH264MotionCompOracleSub8x8(t, &b)
	appendH264MotionCompOracleWeightedP16x16(t, &b)
	appendH264MotionCompOracleWeightedImplicitB16x8(t, &b)
	appendH264MotionCompOracleEdgeP16x16(t, &b)
	appendH264MotionCompOracleHigh10P16x16(t, &b)
	appendH264MotionCompOracleHigh12B16x8(t, &b)
	appendH264MotionCompOracleHigh10WeightedP16x16(t, &b)
	appendH264MotionCompOracleHigh12WeightedP16x8(t, &b)
	appendH264MotionCompOracleHigh12WeightedImplicitB16x8(t, &b)
	appendH264MotionCompOracleHigh12EdgeP16x16(t, &b)
	return b.String()
}

func appendH264MotionCompOracleP16x16(t *testing.T, b *strings.Builder) {
	dst := makeH264MotionCompPicture(1, 3)
	ref0 := makeH264MotionCompPicture(1, 41)
	refs := [2][]*h264PicturePlanes{{ref0}}
	var cache macroblockMotionCache
	cache.Ref[0][h264Scan8[0]] = 0
	cache.MV[0][h264Scan8[0]] = [2]int16{5, 7}
	if err := h264HLMotionFrame(dst, refs, &cache, MBType16x16|MBTypeP0L0, [4]uint32{}, 1, 1, 1); err != nil {
		t.Fatal(err)
	}
	printH264MotionCompMB(b, "p16x16_420", dst, 1, 1)
}

func appendH264MotionCompOracleB16x8(t *testing.T, b *strings.Builder) {
	dst := makeH264MotionCompPicture(1, 9)
	ref0 := makeH264MotionCompPicture(1, 51)
	ref1 := makeH264MotionCompPicture(1, 101)
	refs := [2][]*h264PicturePlanes{{ref0}, {ref1}}
	var cache macroblockMotionCache
	cache.Ref[0][h264Scan8[0]] = 0
	cache.Ref[1][h264Scan8[0]] = 0
	cache.Ref[0][h264Scan8[8]] = 0
	cache.Ref[1][h264Scan8[8]] = 0
	cache.MV[0][h264Scan8[0]] = [2]int16{4, 6}
	cache.MV[1][h264Scan8[0]] = [2]int16{12, 14}
	cache.MV[0][h264Scan8[8]] = [2]int16{8, 10}
	cache.MV[1][h264Scan8[8]] = [2]int16{16, 18}
	mbType := MBType16x8 | MBTypeP0L0 | MBTypeP1L0 | MBTypeP0L1 | MBTypeP1L1
	if err := h264HLMotionFrame(dst, refs, &cache, mbType, [4]uint32{}, 1, 1, 2); err != nil {
		t.Fatal(err)
	}
	printH264MotionCompMB(b, "b16x8_420", dst, 1, 1)
}

func appendH264MotionCompOracleP8x16(t *testing.T, b *strings.Builder) {
	dst := makeH264MotionCompPicture(2, 19)
	ref0 := makeH264MotionCompPicture(2, 73)
	refs := [2][]*h264PicturePlanes{{ref0}}
	var cache macroblockMotionCache
	cache.Ref[0][h264Scan8[0]] = 0
	cache.Ref[0][h264Scan8[4]] = 0
	cache.MV[0][h264Scan8[0]] = [2]int16{3, 5}
	cache.MV[0][h264Scan8[4]] = [2]int16{9, 13}
	if err := h264HLMotionFrame(dst, refs, &cache, MBType8x16|MBTypeP0L0|MBTypeP1L0, [4]uint32{}, 1, 1, 1); err != nil {
		t.Fatal(err)
	}
	printH264MotionCompMB(b, "p8x16_422", dst, 1, 1)
}

func appendH264MotionCompOracleSub8x8(t *testing.T, b *strings.Builder) {
	dst := makeH264MotionCompPicture(1, 27)
	ref0 := makeH264MotionCompPicture(1, 111)
	refs := [2][]*h264PicturePlanes{{ref0}}
	var cache macroblockMotionCache
	for n := 0; n < 16; n++ {
		cache.Ref[0][h264Scan8[n]] = 0
		cache.MV[0][h264Scan8[n]] = [2]int16{int16(n%5 + 1), int16(n%7 + 2)}
	}
	subMBType := [4]uint32{
		MBType16x16 | MBTypeP0L0,
		MBType16x8 | MBTypeP0L0,
		MBType8x16 | MBTypeP0L0,
		MBType8x8 | MBTypeP0L0,
	}
	if err := h264HLMotionFrame(dst, refs, &cache, MBType8x8|MBTypeP0L0|MBTypeP1L0, subMBType, 1, 1, 1); err != nil {
		t.Fatal(err)
	}
	printH264MotionCompMB(b, "sub8x8_420", dst, 1, 1)
}

func appendH264MotionCompOracleWeightedP16x16(t *testing.T, b *strings.Builder) {
	dst := makeH264MotionCompPicture(1, 23)
	ref0 := makeH264MotionCompPicture(1, 91)
	refs := [2][]*h264PicturePlanes{{ref0}}
	var cache macroblockMotionCache
	cache.Ref[0][h264Scan8[0]] = 0
	pwt := h264MotionCompTestPWT(1)
	pwt.UseWeight = 1
	pwt.UseWeightChroma = 1
	pwt.LumaLog2WeightDenom = 2
	pwt.ChromaLog2WeightDenom = 1
	pwt.LumaWeight[0][0] = [2]int32{3, -2}
	pwt.ChromaWeight[0][0][0] = [2]int32{2, 1}
	pwt.ChromaWeight[0][0][1] = [2]int32{-1, 3}
	if err := h264HLMotionFrameWeighted(dst, refs, &cache, MBType16x16|MBTypeP0L0, [4]uint32{}, 1, 1, 1, &pwt, nil); err != nil {
		t.Fatal(err)
	}
	printH264MotionCompMB(b, "weighted_p16x16_420", dst, 1, 1)
}

func appendH264MotionCompOracleWeightedImplicitB16x8(t *testing.T, b *strings.Builder) {
	dst := makeH264MotionCompPicture(2, 37)
	ref0 := makeH264MotionCompPicture(2, 79)
	ref1 := makeH264MotionCompPicture(2, 119)
	refs := [2][]*h264PicturePlanes{{ref0}, {ref1}}
	var cache macroblockMotionCache
	cache.Ref[0][h264Scan8[0]] = 0
	cache.Ref[1][h264Scan8[0]] = 0
	cache.Ref[0][h264Scan8[8]] = 0
	cache.Ref[1][h264Scan8[8]] = 0
	cache.MV[0][h264Scan8[0]] = [2]int16{4, 2}
	cache.MV[1][h264Scan8[0]] = [2]int16{8, 6}
	cache.MV[0][h264Scan8[8]] = [2]int16{12, 10}
	cache.MV[1][h264Scan8[8]] = [2]int16{16, 14}
	pwt := h264MotionCompTestPWT(2)
	pwt.UseWeight = 2
	pwt.ImplicitWeight[0][0][1] = 21
	mbType := MBType16x8 | MBTypeP0L0 | MBTypeP1L0 | MBTypeP0L1 | MBTypeP1L1
	if err := h264HLMotionFrameWeighted(dst, refs, &cache, mbType, [4]uint32{}, 1, 1, 2, &pwt, makeH264MotionCompScratch(dst)); err != nil {
		t.Fatal(err)
	}
	printH264MotionCompMB(b, "weighted_implicit_b16x8_422", dst, 1, 1)
}

func appendH264MotionCompOracleEdgeP16x16(t *testing.T, b *strings.Builder) {
	dst := makeH264MotionCompPicture(1, 43)
	ref0 := makeH264MotionCompPicture(1, 131)
	refs := [2][]*h264PicturePlanes{{ref0}}
	var cache macroblockMotionCache
	cache.Ref[0][h264Scan8[0]] = 0
	cache.MV[0][h264Scan8[0]] = [2]int16{1, 0}
	if err := h264HLMotionFrameWithScratch(dst, refs, &cache, MBType16x16|MBTypeP0L0, [4]uint32{}, 3, 1, 1, makeH264MotionCompScratch(dst)); err != nil {
		t.Fatal(err)
	}
	printH264MotionCompMB(b, "edge_p16x16_420", dst, 3, 1)
}

func appendH264MotionCompOracleHigh10P16x16(t *testing.T, b *strings.Builder) {
	const bitDepth = 10
	dst := makeH264MotionCompPictureHigh(1, bitDepth, 203)
	ref0 := makeH264MotionCompPictureHigh(1, bitDepth, 241)
	refs := [2][]*h264PicturePlanesHigh{{ref0}}
	var cache macroblockMotionCache
	cache.Ref[0][h264Scan8[0]] = 0
	cache.MV[0][h264Scan8[0]] = [2]int16{5, 7}
	if err := h264HLMotionFrameHighOracle(dst, refs, &cache, MBType16x16|MBTypeP0L0, [4]uint32{}, 1, 1, 1, bitDepth, nil, nil); err != nil {
		t.Fatal(err)
	}
	printH264MotionCompMBHigh(b, "high10_p16x16_420", dst, 1, 1)
}

func appendH264MotionCompOracleHigh12B16x8(t *testing.T, b *strings.Builder) {
	const bitDepth = 12
	dst := makeH264MotionCompPictureHigh(3, bitDepth, 309)
	ref0 := makeH264MotionCompPictureHigh(3, bitDepth, 351)
	ref1 := makeH264MotionCompPictureHigh(3, bitDepth, 401)
	refs := [2][]*h264PicturePlanesHigh{{ref0}, {ref1}}
	var cache macroblockMotionCache
	cache.Ref[0][h264Scan8[0]] = 0
	cache.Ref[1][h264Scan8[0]] = 0
	cache.Ref[0][h264Scan8[8]] = 0
	cache.Ref[1][h264Scan8[8]] = 0
	cache.MV[0][h264Scan8[0]] = [2]int16{4, 6}
	cache.MV[1][h264Scan8[0]] = [2]int16{12, 14}
	cache.MV[0][h264Scan8[8]] = [2]int16{8, 10}
	cache.MV[1][h264Scan8[8]] = [2]int16{16, 18}
	mbType := MBType16x8 | MBTypeP0L0 | MBTypeP1L0 | MBTypeP0L1 | MBTypeP1L1
	if err := h264HLMotionFrameHighOracle(dst, refs, &cache, mbType, [4]uint32{}, 1, 1, 2, bitDepth, nil, nil); err != nil {
		t.Fatal(err)
	}
	printH264MotionCompMBHigh(b, "high12_b16x8_444", dst, 1, 1)
}

func appendH264MotionCompOracleHigh10WeightedP16x16(t *testing.T, b *strings.Builder) {
	const bitDepth = 10
	dst := makeH264MotionCompPictureHigh(1, bitDepth, 223)
	ref0 := makeH264MotionCompPictureHigh(1, bitDepth, 291)
	refs := [2][]*h264PicturePlanesHigh{{ref0}}
	var cache macroblockMotionCache
	cache.Ref[0][h264Scan8[0]] = 0
	pwt := h264MotionCompTestPWT(1)
	pwt.UseWeight = 1
	pwt.UseWeightChroma = 1
	pwt.LumaLog2WeightDenom = 2
	pwt.ChromaLog2WeightDenom = 1
	pwt.LumaWeight[0][0] = [2]int32{3, -2}
	pwt.ChromaWeight[0][0][0] = [2]int32{2, 1}
	pwt.ChromaWeight[0][0][1] = [2]int32{-1, 3}
	if err := h264HLMotionFrameHighOracle(dst, refs, &cache, MBType16x16|MBTypeP0L0, [4]uint32{}, 1, 1, 1, bitDepth, &pwt, nil); err != nil {
		t.Fatal(err)
	}
	printH264MotionCompMBHigh(b, "high10_weighted_p16x16_420", dst, 1, 1)
}

func appendH264MotionCompOracleHigh12WeightedP16x8(t *testing.T, b *strings.Builder) {
	const bitDepth = 12
	dst := makeH264MotionCompPictureHigh(1, bitDepth, 457)
	ref0 := makeH264MotionCompPictureHigh(1, bitDepth, 509)
	refs := [2][]*h264PicturePlanesHigh{{ref0}}
	var cache macroblockMotionCache
	cache.Ref[0][h264Scan8[0]] = 0
	cache.Ref[0][h264Scan8[8]] = 0
	cache.MV[0][h264Scan8[0]] = [2]int16{3, 5}
	cache.MV[0][h264Scan8[8]] = [2]int16{7, 11}
	pwt := h264MotionCompTestPWT(1)
	pwt.UseWeight = 1
	pwt.UseWeightChroma = 1
	pwt.LumaLog2WeightDenom = 2
	pwt.ChromaLog2WeightDenom = 1
	pwt.LumaWeight[0][0] = [2]int32{3, -2}
	pwt.ChromaWeight[0][0][0] = [2]int32{2, 1}
	pwt.ChromaWeight[0][0][1] = [2]int32{-1, 3}
	mbType := MBType16x8 | MBTypeP0L0 | MBTypeP1L0
	if err := h264HLMotionFrameHighOracle(dst, refs, &cache, mbType, [4]uint32{}, 1, 1, 1, bitDepth, &pwt, nil); err != nil {
		t.Fatal(err)
	}
	printH264MotionCompMBHigh(b, "high12_weighted_p16x8_420", dst, 1, 1)
}

func appendH264MotionCompOracleHigh12WeightedImplicitB16x8(t *testing.T, b *strings.Builder) {
	const bitDepth = 12
	dst := makeH264MotionCompPictureHigh(2, bitDepth, 337)
	ref0 := makeH264MotionCompPictureHigh(2, bitDepth, 379)
	ref1 := makeH264MotionCompPictureHigh(2, bitDepth, 419)
	refs := [2][]*h264PicturePlanesHigh{{ref0}, {ref1}}
	var cache macroblockMotionCache
	cache.Ref[0][h264Scan8[0]] = 0
	cache.Ref[1][h264Scan8[0]] = 0
	cache.Ref[0][h264Scan8[8]] = 0
	cache.Ref[1][h264Scan8[8]] = 0
	cache.MV[0][h264Scan8[0]] = [2]int16{4, 2}
	cache.MV[1][h264Scan8[0]] = [2]int16{8, 6}
	cache.MV[0][h264Scan8[8]] = [2]int16{12, 10}
	cache.MV[1][h264Scan8[8]] = [2]int16{16, 14}
	pwt := h264MotionCompTestPWT(2)
	pwt.UseWeight = 2
	pwt.ImplicitWeight[0][0][1] = 21
	mbType := MBType16x8 | MBTypeP0L0 | MBTypeP1L0 | MBTypeP0L1 | MBTypeP1L1
	if err := h264HLMotionFrameHighOracle(dst, refs, &cache, mbType, [4]uint32{}, 1, 1, 2, bitDepth, &pwt, makeH264MotionCompScratchHigh(dst)); err != nil {
		t.Fatal(err)
	}
	printH264MotionCompMBHigh(b, "high12_weighted_implicit_b16x8_422", dst, 1, 1)
}

func appendH264MotionCompOracleHigh12EdgeP16x16(t *testing.T, b *strings.Builder) {
	const bitDepth = 12
	dst := makeH264MotionCompPictureHigh(1, bitDepth, 343)
	ref0 := makeH264MotionCompPictureHigh(1, bitDepth, 431)
	refs := [2][]*h264PicturePlanesHigh{{ref0}}
	var cache macroblockMotionCache
	cache.Ref[0][h264Scan8[0]] = 0
	cache.MV[0][h264Scan8[0]] = [2]int16{1, 0}
	if err := h264HLMotionFrameHighOracle(dst, refs, &cache, MBType16x16|MBTypeP0L0, [4]uint32{}, 3, 1, 1, bitDepth, nil, makeH264MotionCompScratchHigh(dst)); err != nil {
		t.Fatal(err)
	}
	printH264MotionCompMBHigh(b, "high12_edge_p16x16_420", dst, 3, 1)
}

func printH264MotionCompMB(b *strings.Builder, label string, p *h264PicturePlanes, mbX int, mbY int) {
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

func h264HLMotionFrameHighOracle(dst *h264PicturePlanesHigh, refs [2][]*h264PicturePlanesHigh, cache *macroblockMotionCache, mbType uint32, subMBType [4]uint32, mbX int, mbY int, listCount int, bitDepth int, pwt *PredWeightTable, scratch *h264MotionCompScratchHigh) error {
	if pwt != nil {
		return h264HLMotionFrameWeightedHigh(dst, refs, cache, mbType, subMBType, mbX, mbY, listCount, pwt, scratch, bitDepth)
	}
	if scratch != nil {
		return h264HLMotionFrameWithScratchHigh(dst, refs, cache, mbType, subMBType, mbX, mbY, listCount, scratch, bitDepth)
	}
	return h264HLMotionFrameHigh(dst, refs, cache, mbType, subMBType, mbX, mbY, listCount, bitDepth)
}
