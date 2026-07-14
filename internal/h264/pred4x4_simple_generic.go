// SPDX-License-Identifier: LGPL-2.1-or-later
//go:build purego || !arm64

package h264

func h264Pred4x4SimpleKernel(pix []uint8, offset int, stride int, mode int) {
	switch int8(mode) {
	case intraPredVertical:
		for y := 0; y < 4; y++ {
			copy(pix[offset+y*stride:offset+y*stride+4], pix[offset-stride:offset-stride+4])
		}
	case intraPredHorizontal:
		for y := 0; y < 4; y++ {
			fillPredictionRow(pix, offset+y*stride, 4, pix[offset-1+y*stride])
		}
	case intraPredDC:
		dc := int(pix[offset-stride]) + int(pix[offset+1-stride]) + int(pix[offset+2-stride]) + int(pix[offset+3-stride]) +
			int(pix[offset-1]) + int(pix[offset-1+stride]) + int(pix[offset-1+2*stride]) + int(pix[offset-1+3*stride])
		fillPredictionBlock(pix, offset, stride, 4, 4, uint8((dc+4)>>3))
	case intraPredLeftDC:
		dc := int(pix[offset-1]) + int(pix[offset-1+stride]) + int(pix[offset-1+2*stride]) + int(pix[offset-1+3*stride])
		fillPredictionBlock(pix, offset, stride, 4, 4, uint8((dc+2)>>2))
	case intraPredTopDC:
		dc := int(pix[offset-stride]) + int(pix[offset+1-stride]) + int(pix[offset+2-stride]) + int(pix[offset+3-stride])
		fillPredictionBlock(pix, offset, stride, 4, 4, uint8((dc+2)>>2))
	case intraPredDC128:
		fillPredictionBlock(pix, offset, stride, 4, 4, 128)
	}
}
