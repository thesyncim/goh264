// SPDX-License-Identifier: LGPL-2.1-or-later
//go:build !purego && arm64

package h264

func h264Pred4x4SimpleKernel(pix []uint8, offset int, stride int, mode int) {
	h264Pred4x4SimpleASM(&pix[offset], stride, int32(mode))
}

//go:noescape
func h264Pred4x4SimpleASM(dst *uint8, stride int, mode int32)
