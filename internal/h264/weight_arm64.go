// SPDX-License-Identifier: LGPL-2.1-or-later
//go:build !purego && arm64

package h264

const h264WeightPixels16ASMEnabled = true

// h264WeightPixels16ASM mirrors FFmpeg's 8-bit weighted prediction ABI shape:
// uint8_t *dst, ptrdiff_t stride, int height/log2_den/weight/offset.
//
//go:noescape
func h264WeightPixels16ASM(dst *uint8, stride int, height int32, log2Denom int32, weight int32, offset int32)
