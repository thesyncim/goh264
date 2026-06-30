// SPDX-License-Identifier: LGPL-2.1-or-later
//go:build !purego && arm64

package h264

const h264LoopFilterChromaHighASMEnabled = true

// h264VLoopFilterChromaHigh10ASM mirrors FFmpeg's 10-bit chroma deblock ABI shape:
// uint8_t *pix, ptrdiff_t stride, int alpha/beta, int8_t *tc0.
//
//go:noescape
func h264VLoopFilterChromaHigh10ASM(pix *uint8, stride int, alpha int32, beta int32, tc0 *int8)

// h264HLoopFilterChromaHigh10ASM mirrors FFmpeg's 10-bit chroma deblock ABI shape:
// uint8_t *pix, ptrdiff_t stride, int alpha/beta, int8_t *tc0.
//
//go:noescape
func h264HLoopFilterChromaHigh10ASM(pix *uint8, stride int, alpha int32, beta int32, tc0 *int8)

// h264HLoopFilterChroma422High10ASM mirrors FFmpeg's 10-bit 4:2:2 chroma deblock ABI shape:
// uint8_t *pix, ptrdiff_t stride, int alpha/beta, int8_t *tc0.
//
//go:noescape
func h264HLoopFilterChroma422High10ASM(pix *uint8, stride int, alpha int32, beta int32, tc0 *int8)
