// SPDX-License-Identifier: LGPL-2.1-or-later
//go:build !purego && arm64

package h264

const h264LoopFilterLumaIntraASMEnabled = true

// h264VLoopFilterLumaIntra8ASM mirrors FFmpeg's 8-bit luma intra deblock ABI shape:
// uint8_t *pix, ptrdiff_t stride, int alpha/beta.
//
//go:noescape
func h264VLoopFilterLumaIntra8ASM(pix *uint8, stride int, alpha int32, beta int32)
