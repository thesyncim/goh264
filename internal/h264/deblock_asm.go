// SPDX-License-Identifier: LGPL-2.1-or-later
//go:build !purego && (amd64 || arm64)

package h264

const h264LoopFilterLumaASMEnabled = true

// h264VLoopFilterLuma8ASM mirrors FFmpeg's 8-bit luma deblock ABI shape:
// uint8_t *pix, ptrdiff_t stride, int alpha/beta, int8_t *tc0.
//
//go:noescape
func h264VLoopFilterLuma8ASM(pix *uint8, stride int, alpha int32, beta int32, tc0 *int8)

// h264HLoopFilterLuma8ASM mirrors FFmpeg's 8-bit luma deblock ABI shape:
// uint8_t *pix, ptrdiff_t stride, int alpha/beta, int8_t *tc0.
//
//go:noescape
func h264HLoopFilterLuma8ASM(pix *uint8, stride int, alpha int32, beta int32, tc0 *int8)
