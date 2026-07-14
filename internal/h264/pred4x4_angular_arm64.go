// SPDX-License-Identifier: LGPL-2.1-or-later
//go:build !purego && arm64

package h264

func h264Pred4x4DownRightKernel(pix []uint8, offset int, stride int) {
	h264Pred4x4DownRightASM(&pix[offset], stride)
}

func h264Pred4x4HorizontalUpKernel(pix []uint8, offset int, stride int) {
	h264Pred4x4HorizontalUpASM(&pix[offset], stride)
}

//go:noescape
func h264Pred4x4DownRightASM(dst *uint8, stride int)

//go:noescape
func h264Pred4x4HorizontalUpASM(dst *uint8, stride int)
