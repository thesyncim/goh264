// SPDX-License-Identifier: LGPL-2.1-or-later
//go:build !purego && arm64

package h264

const h264LoopFilterChromaASMEnabled = true

// h264VLoopFilterChroma8ASM mirrors FFmpeg's 8-bit chroma deblock ABI shape:
// uint8_t *pix, ptrdiff_t stride, int alpha/beta, int8_t *tc0.
//
//go:noescape
func h264VLoopFilterChroma8ASM(pix *uint8, stride int, alpha int32, beta int32, tc0 *int8)

// h264HLoopFilterChroma8ASM mirrors FFmpeg's 8-bit chroma deblock ABI shape:
// uint8_t *pix, ptrdiff_t stride, int alpha/beta, int8_t *tc0.
//
//go:noescape
func h264HLoopFilterChroma8ASM(pix *uint8, stride int, alpha int32, beta int32, tc0 *int8)

// h264HLoopFilterChroma4228ASM mirrors FFmpeg's 8-bit 4:2:2 chroma deblock ABI shape:
// uint8_t *pix, ptrdiff_t stride, int alpha/beta, int8_t *tc0.
//
//go:noescape
func h264HLoopFilterChroma4228ASM(pix *uint8, stride int, alpha int32, beta int32, tc0 *int8)

// h264VLoopFilterChromaIntra8ASM mirrors FFmpeg's 8-bit chroma intra deblock ABI shape:
// uint8_t *pix, ptrdiff_t stride, int alpha/beta.
//
//go:noescape
func h264VLoopFilterChromaIntra8ASM(pix *uint8, stride int, alpha int32, beta int32)

// h264HLoopFilterChromaIntra8ASM mirrors FFmpeg's 8-bit chroma intra deblock ABI shape:
// uint8_t *pix, ptrdiff_t stride, int alpha/beta.
//
//go:noescape
func h264HLoopFilterChromaIntra8ASM(pix *uint8, stride int, alpha int32, beta int32)

// h264HLoopFilterChroma422Intra8ASM mirrors FFmpeg's 8-bit 4:2:2 chroma intra deblock ABI shape:
// uint8_t *pix, ptrdiff_t stride, int alpha/beta.
//
//go:noescape
func h264HLoopFilterChroma422Intra8ASM(pix *uint8, stride int, alpha int32, beta int32)
