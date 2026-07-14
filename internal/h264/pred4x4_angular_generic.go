// SPDX-License-Identifier: LGPL-2.1-or-later
//go:build purego || !arm64

package h264

func h264Pred4x4DownRightKernel(pix []uint8, offset int, stride int) {
	h264Pred4x4DownRightScalar(pix, offset, stride)
}

func h264Pred4x4HorizontalUpKernel(pix []uint8, offset int, stride int) {
	h264Pred4x4HorizontalUpScalar(pix, offset, stride)
}
