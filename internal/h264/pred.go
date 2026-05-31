// SPDX-License-Identifier: LGPL-2.1-or-later
//
// Source-shaped 8-bit H.264 pixel prediction helpers from FFmpeg n8.0.1
// libavcodec/h264pred_template.c.

package h264

func h264Pred16x16Vertical(pix []uint8, offset int, stride int) error {
	if err := checkPredictionArgs(pix, offset, stride, 16, 16, 0, 1); err != nil {
		return err
	}
	for y := 0; y < 16; y++ {
		copy(pix[offset+y*stride:offset+y*stride+16], pix[offset-stride:offset-stride+16])
	}
	return nil
}

func h264Pred16x16Horizontal(pix []uint8, offset int, stride int) error {
	if err := checkPredictionArgs(pix, offset, stride, 16, 16, 1, 0); err != nil {
		return err
	}
	for y := 0; y < 16; y++ {
		v := pix[offset+y*stride-1]
		fillPredictionRow(pix, offset+y*stride, 16, v)
	}
	return nil
}

func h264Pred16x16DC(pix []uint8, offset int, stride int) error {
	if err := checkPredictionArgs(pix, offset, stride, 16, 16, 1, 1); err != nil {
		return err
	}
	dc := 0
	for i := 0; i < 16; i++ {
		dc += int(pix[offset-1+i*stride])
		dc += int(pix[offset+i-stride])
	}
	fillPredictionBlock(pix, offset, stride, 16, 16, uint8((dc+16)>>5))
	return nil
}

func h264Pred16x16LeftDC(pix []uint8, offset int, stride int) error {
	if err := checkPredictionArgs(pix, offset, stride, 16, 16, 1, 0); err != nil {
		return err
	}
	dc := 0
	for i := 0; i < 16; i++ {
		dc += int(pix[offset-1+i*stride])
	}
	fillPredictionBlock(pix, offset, stride, 16, 16, uint8((dc+8)>>4))
	return nil
}

func h264Pred16x16TopDC(pix []uint8, offset int, stride int) error {
	if err := checkPredictionArgs(pix, offset, stride, 16, 16, 0, 1); err != nil {
		return err
	}
	dc := 0
	for i := 0; i < 16; i++ {
		dc += int(pix[offset+i-stride])
	}
	fillPredictionBlock(pix, offset, stride, 16, 16, uint8((dc+8)>>4))
	return nil
}

func h264Pred16x16DC128(pix []uint8, offset int, stride int) error {
	return h264PredConstant(pix, offset, stride, 16, 16, 128)
}

func h264Pred16x16DC127(pix []uint8, offset int, stride int) error {
	return h264PredConstant(pix, offset, stride, 16, 16, 127)
}

func h264Pred16x16DC129(pix []uint8, offset int, stride int) error {
	return h264PredConstant(pix, offset, stride, 16, 16, 129)
}

func h264Pred16x16Plane(pix []uint8, offset int, stride int) error {
	if err := checkPredictionArgs(pix, offset, stride, 16, 16, 1, 1); err != nil {
		return err
	}
	h := int(pix[offset+8-stride]) - int(pix[offset+6-stride])
	v := int(pix[offset+8*stride-1]) - int(pix[offset+6*stride-1])
	for k := 2; k <= 8; k++ {
		h += k * (int(pix[offset+7+k-stride]) - int(pix[offset+7-k-stride]))
		v += k * (int(pix[offset+(7+k)*stride-1]) - int(pix[offset+(7-k)*stride-1]))
	}
	h = (5*h + 32) >> 6
	v = (5*v + 32) >> 6

	a := 16*(int(pix[offset+15*stride-1])+int(pix[offset+15-stride])+1) - 7*(v+h)
	for y := 0; y < 16; y++ {
		b := a
		a += v
		row := offset + y*stride
		for x := 0; x < 16; x++ {
			pix[row+x] = clipUint8(b >> 5)
			b += h
		}
	}
	return nil
}

func h264Pred8x8Vertical(pix []uint8, offset int, stride int) error {
	if err := checkPredictionArgs(pix, offset, stride, 8, 8, 0, 1); err != nil {
		return err
	}
	for y := 0; y < 8; y++ {
		copy(pix[offset+y*stride:offset+y*stride+8], pix[offset-stride:offset-stride+8])
	}
	return nil
}

func h264Pred8x16Vertical(pix []uint8, offset int, stride int) error {
	if err := checkPredictionArgs(pix, offset, stride, 8, 16, 0, 1); err != nil {
		return err
	}
	for y := 0; y < 16; y++ {
		copy(pix[offset+y*stride:offset+y*stride+8], pix[offset-stride:offset-stride+8])
	}
	return nil
}

func h264Pred8x8Horizontal(pix []uint8, offset int, stride int) error {
	if err := checkPredictionArgs(pix, offset, stride, 8, 8, 1, 0); err != nil {
		return err
	}
	for y := 0; y < 8; y++ {
		fillPredictionRow(pix, offset+y*stride, 8, pix[offset+y*stride-1])
	}
	return nil
}

func h264Pred8x16Horizontal(pix []uint8, offset int, stride int) error {
	if err := checkPredictionArgs(pix, offset, stride, 8, 16, 1, 0); err != nil {
		return err
	}
	for y := 0; y < 16; y++ {
		fillPredictionRow(pix, offset+y*stride, 8, pix[offset+y*stride-1])
	}
	return nil
}

func h264Pred8x8DC(pix []uint8, offset int, stride int) error {
	if err := checkPredictionArgs(pix, offset, stride, 8, 8, 1, 1); err != nil {
		return err
	}
	dc0, dc1, dc2 := 0, 0, 0
	for i := 0; i < 4; i++ {
		dc0 += int(pix[offset-1+i*stride]) + int(pix[offset+i-stride])
		dc1 += int(pix[offset+4+i-stride])
		dc2 += int(pix[offset-1+(i+4)*stride])
	}
	fillPredictionRect(pix, offset, stride, 0, 0, 4, 4, uint8((dc0+4)>>3))
	fillPredictionRect(pix, offset, stride, 4, 0, 4, 4, uint8((dc1+2)>>2))
	fillPredictionRect(pix, offset, stride, 0, 4, 4, 4, uint8((dc2+2)>>2))
	fillPredictionRect(pix, offset, stride, 4, 4, 4, 4, uint8((dc1+dc2+4)>>3))
	return nil
}

func h264Pred8x16DC(pix []uint8, offset int, stride int) error {
	if err := checkPredictionArgs(pix, offset, stride, 8, 16, 1, 1); err != nil {
		return err
	}
	dc0, dc1, dc2, dc3, dc4 := 0, 0, 0, 0, 0
	for i := 0; i < 4; i++ {
		dc0 += int(pix[offset-1+i*stride]) + int(pix[offset+i-stride])
		dc1 += int(pix[offset+4+i-stride])
		dc2 += int(pix[offset-1+(i+4)*stride])
		dc3 += int(pix[offset-1+(i+8)*stride])
		dc4 += int(pix[offset-1+(i+12)*stride])
	}
	fillPredictionRect(pix, offset, stride, 0, 0, 4, 4, uint8((dc0+4)>>3))
	fillPredictionRect(pix, offset, stride, 4, 0, 4, 4, uint8((dc1+2)>>2))
	fillPredictionRect(pix, offset, stride, 0, 4, 4, 4, uint8((dc2+2)>>2))
	fillPredictionRect(pix, offset, stride, 4, 4, 4, 4, uint8((dc1+dc2+4)>>3))
	fillPredictionRect(pix, offset, stride, 0, 8, 4, 4, uint8((dc3+2)>>2))
	fillPredictionRect(pix, offset, stride, 4, 8, 4, 4, uint8((dc1+dc3+4)>>3))
	fillPredictionRect(pix, offset, stride, 0, 12, 4, 4, uint8((dc4+2)>>2))
	fillPredictionRect(pix, offset, stride, 4, 12, 4, 4, uint8((dc1+dc4+4)>>3))
	return nil
}

func h264Pred8x8LeftDC(pix []uint8, offset int, stride int) error {
	if err := checkPredictionArgs(pix, offset, stride, 8, 8, 1, 0); err != nil {
		return err
	}
	dc0, dc2 := 0, 0
	for i := 0; i < 4; i++ {
		dc0 += int(pix[offset-1+i*stride])
		dc2 += int(pix[offset-1+(i+4)*stride])
	}
	fillPredictionRect(pix, offset, stride, 0, 0, 8, 4, uint8((dc0+2)>>2))
	fillPredictionRect(pix, offset, stride, 0, 4, 8, 4, uint8((dc2+2)>>2))
	return nil
}

func h264Pred8x16LeftDC(pix []uint8, offset int, stride int) error {
	if err := checkPredictionArgs(pix, offset, stride, 8, 16, 1, 0); err != nil {
		return err
	}
	if err := h264Pred8x8LeftDC(pix, offset, stride); err != nil {
		return err
	}
	return h264Pred8x8LeftDC(pix, offset+8*stride, stride)
}

func h264Pred8x8TopDC(pix []uint8, offset int, stride int) error {
	if err := checkPredictionArgs(pix, offset, stride, 8, 8, 0, 1); err != nil {
		return err
	}
	dc0, dc1 := 0, 0
	for i := 0; i < 4; i++ {
		dc0 += int(pix[offset+i-stride])
		dc1 += int(pix[offset+4+i-stride])
	}
	fillPredictionRect(pix, offset, stride, 0, 0, 4, 8, uint8((dc0+2)>>2))
	fillPredictionRect(pix, offset, stride, 4, 0, 4, 8, uint8((dc1+2)>>2))
	return nil
}

func h264Pred8x16TopDC(pix []uint8, offset int, stride int) error {
	if err := checkPredictionArgs(pix, offset, stride, 8, 16, 0, 1); err != nil {
		return err
	}
	dc0, dc1 := 0, 0
	for i := 0; i < 4; i++ {
		dc0 += int(pix[offset+i-stride])
		dc1 += int(pix[offset+4+i-stride])
	}
	fillPredictionRect(pix, offset, stride, 0, 0, 4, 16, uint8((dc0+2)>>2))
	fillPredictionRect(pix, offset, stride, 4, 0, 4, 16, uint8((dc1+2)>>2))
	return nil
}

func h264Pred8x8DC128(pix []uint8, offset int, stride int) error {
	return h264PredConstant(pix, offset, stride, 8, 8, 128)
}

func h264Pred8x8DC127(pix []uint8, offset int, stride int) error {
	return h264PredConstant(pix, offset, stride, 8, 8, 127)
}

func h264Pred8x8DC129(pix []uint8, offset int, stride int) error {
	return h264PredConstant(pix, offset, stride, 8, 8, 129)
}

func h264Pred8x16DC128(pix []uint8, offset int, stride int) error {
	return h264PredConstant(pix, offset, stride, 8, 16, 128)
}

func h264Pred8x8Plane(pix []uint8, offset int, stride int) error {
	if err := checkPredictionArgs(pix, offset, stride, 8, 8, 1, 1); err != nil {
		return err
	}
	h := int(pix[offset+4-stride]) - int(pix[offset+2-stride])
	v := int(pix[offset+4*stride-1]) - int(pix[offset+2*stride-1])
	for k := 2; k <= 4; k++ {
		h += k * (int(pix[offset+3+k-stride]) - int(pix[offset+3-k-stride]))
		v += k * (int(pix[offset+(3+k)*stride-1]) - int(pix[offset+(3-k)*stride-1]))
	}
	h = (17*h + 16) >> 5
	v = (17*v + 16) >> 5

	a := 16*(int(pix[offset+7*stride-1])+int(pix[offset+7-stride])+1) - 3*(v+h)
	for y := 0; y < 8; y++ {
		b := a
		a += v
		row := offset + y*stride
		for x := 0; x < 8; x++ {
			pix[row+x] = clipUint8(b >> 5)
			b += h
		}
	}
	return nil
}

func h264Pred8x16Plane(pix []uint8, offset int, stride int) error {
	if err := checkPredictionArgs(pix, offset, stride, 8, 16, 1, 1); err != nil {
		return err
	}
	h := int(pix[offset+4-stride]) - int(pix[offset+2-stride])
	v := int(pix[offset+8*stride-1]) - int(pix[offset+6*stride-1])
	k := 2
	for ; k <= 4; k++ {
		h += k * (int(pix[offset+3+k-stride]) - int(pix[offset+3-k-stride]))
		v += k * (int(pix[offset+(7+k)*stride-1]) - int(pix[offset+(7-k)*stride-1]))
	}
	for ; k <= 8; k++ {
		v += k * (int(pix[offset+(7+k)*stride-1]) - int(pix[offset+(7-k)*stride-1]))
	}
	h = (17*h + 16) >> 5
	v = (5*v + 32) >> 6

	a := 16*(int(pix[offset+15*stride-1])+int(pix[offset+7-stride])+1) - 7*v - 3*h
	for y := 0; y < 16; y++ {
		b := a
		a += v
		row := offset + y*stride
		for x := 0; x < 8; x++ {
			pix[row+x] = clipUint8(b >> 5)
			b += h
		}
	}
	return nil
}

func h264Pred4x4VerticalAdd(pix []uint8, offset int, block []int32, stride int) error {
	if err := checkPredictionArgs(pix, offset, stride, 4, 4, 0, 1); err != nil {
		return err
	}
	if len(block) < 16 {
		return ErrInvalidData
	}
	for x := 0; x < 4; x++ {
		v := pix[offset-stride+x]
		v += uint8(dctcoef8Value(block[x]))
		pix[offset+x] = v
		v += uint8(dctcoef8Value(block[4+x]))
		pix[offset+stride+x] = v
		v += uint8(dctcoef8Value(block[8+x]))
		pix[offset+2*stride+x] = v
		pix[offset+3*stride+x] = v + uint8(dctcoef8Value(block[12+x]))
	}
	clearInt32(block[:16])
	return nil
}

func h264Pred4x4HorizontalAdd(pix []uint8, offset int, block []int32, stride int) error {
	if err := checkPredictionArgs(pix, offset, stride, 4, 4, 1, 0); err != nil {
		return err
	}
	if len(block) < 16 {
		return ErrInvalidData
	}
	for y := 0; y < 4; y++ {
		row := offset + y*stride
		src := y * 4
		v := pix[row-1]
		v += uint8(dctcoef8Value(block[src+0]))
		pix[row+0] = v
		v += uint8(dctcoef8Value(block[src+1]))
		pix[row+1] = v
		v += uint8(dctcoef8Value(block[src+2]))
		pix[row+2] = v
		pix[row+3] = v + uint8(dctcoef8Value(block[src+3]))
	}
	clearInt32(block[:16])
	return nil
}

func h264Pred16x16VerticalAdd(pix []uint8, blockOffset *[48]int, block []int32, stride int) error {
	if blockOffset == nil || len(block) < 16*16 {
		return ErrInvalidData
	}
	for i := 0; i < 16; i++ {
		if err := h264Pred4x4VerticalAdd(pix, blockOffset[i], block[i*16:i*16+16], stride); err != nil {
			return err
		}
	}
	return nil
}

func h264Pred16x16HorizontalAdd(pix []uint8, blockOffset *[48]int, block []int32, stride int) error {
	if blockOffset == nil || len(block) < 16*16 {
		return ErrInvalidData
	}
	for i := 0; i < 16; i++ {
		if err := h264Pred4x4HorizontalAdd(pix, blockOffset[i], block[i*16:i*16+16], stride); err != nil {
			return err
		}
	}
	return nil
}

func h264Pred8x8VerticalAdd(pix []uint8, blockOffset *[48]int, block []int32, stride int) error {
	if blockOffset == nil || len(block) < 4*16 {
		return ErrInvalidData
	}
	for i := 0; i < 4; i++ {
		if err := h264Pred4x4VerticalAdd(pix, blockOffset[i], block[i*16:i*16+16], stride); err != nil {
			return err
		}
	}
	return nil
}

func h264Pred8x8HorizontalAdd(pix []uint8, blockOffset *[48]int, block []int32, stride int) error {
	if blockOffset == nil || len(block) < 4*16 {
		return ErrInvalidData
	}
	for i := 0; i < 4; i++ {
		if err := h264Pred4x4HorizontalAdd(pix, blockOffset[i], block[i*16:i*16+16], stride); err != nil {
			return err
		}
	}
	return nil
}

func h264Pred8x16VerticalAdd(pix []uint8, blockOffset *[48]int, block []int32, stride int) error {
	if blockOffset == nil || len(block) < 8*16 {
		return ErrInvalidData
	}
	for i := 0; i < 4; i++ {
		if err := h264Pred4x4VerticalAdd(pix, blockOffset[i], block[i*16:i*16+16], stride); err != nil {
			return err
		}
	}
	for i := 4; i < 8; i++ {
		if err := h264Pred4x4VerticalAdd(pix, blockOffset[i+4], block[i*16:i*16+16], stride); err != nil {
			return err
		}
	}
	return nil
}

func h264Pred8x16HorizontalAdd(pix []uint8, blockOffset *[48]int, block []int32, stride int) error {
	if blockOffset == nil || len(block) < 8*16 {
		return ErrInvalidData
	}
	for i := 0; i < 4; i++ {
		if err := h264Pred4x4HorizontalAdd(pix, blockOffset[i], block[i*16:i*16+16], stride); err != nil {
			return err
		}
	}
	for i := 4; i < 8; i++ {
		if err := h264Pred4x4HorizontalAdd(pix, blockOffset[i+4], block[i*16:i*16+16], stride); err != nil {
			return err
		}
	}
	return nil
}

func h264PredConstant(pix []uint8, offset int, stride int, width int, height int, value uint8) error {
	if err := checkPredictionArgs(pix, offset, stride, width, height, 0, 0); err != nil {
		return err
	}
	fillPredictionBlock(pix, offset, stride, width, height, value)
	return nil
}

func fillPredictionBlock(pix []uint8, offset int, stride int, width int, height int, value uint8) {
	for y := 0; y < height; y++ {
		fillPredictionRow(pix, offset+y*stride, width, value)
	}
}

func fillPredictionRect(pix []uint8, offset int, stride int, x0 int, y0 int, width int, height int, value uint8) {
	for y := 0; y < height; y++ {
		fillPredictionRow(pix, offset+(y0+y)*stride+x0, width, value)
	}
}

func fillPredictionRow(pix []uint8, offset int, width int, value uint8) {
	for x := 0; x < width; x++ {
		pix[offset+x] = value
	}
}

func checkPredictionArgs(pix []uint8, offset int, stride int, width int, height int, leftMargin int, topMargin int) error {
	if offset < 0 || stride <= 0 || width <= 0 || height <= 0 || leftMargin < 0 || topMargin < 0 {
		return ErrInvalidData
	}
	minIndex := offset - leftMargin - topMargin*stride
	maxIndex := offset + (height-1)*stride + width - 1
	if minIndex < 0 || maxIndex >= len(pix) {
		return ErrInvalidData
	}
	return nil
}
