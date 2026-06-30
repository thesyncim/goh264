// SPDX-License-Identifier: LGPL-2.1-or-later
//go:build !purego && amd64

package h264

const h264LoopFilterLumaIntraASMEnabled = true
const h264LoopFilterLumaIntraV8ASMEnabled = true
const h264LoopFilterLumaIntraH8ASMEnabled = true

// h264VLoopFilterLumaIntra8ASM mirrors FFmpeg's 8-bit luma intra deblock ABI shape:
// uint8_t *pix, ptrdiff_t stride, int alpha/beta.
//
//go:noescape
func h264VLoopFilterLumaIntra8ASM(pix *uint8, stride int, alpha int32, beta int32)

// h264HLoopFilterLumaIntra8ASM mirrors FFmpeg's 8-bit luma intra deblock ABI shape:
// uint8_t *pix, ptrdiff_t stride, int alpha/beta.
//
//go:noescape
func h264HLoopFilterLumaIntra8ASM(pix *uint8, stride int, alpha int32, beta int32)
