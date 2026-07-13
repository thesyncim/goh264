// SPDX-License-Identifier: LGPL-2.1-or-later
//go:build purego || !arm64

package h264

const h264WeightPixels16ASMEnabled = false

func h264WeightPixels16ASM(dst *uint8, stride int, height int32, log2Denom int32, weight int32, offset int32) {
}
