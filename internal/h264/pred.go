// SPDX-License-Identifier: LGPL-2.1-or-later
//
// Source-shaped 8-bit H.264 pixel prediction helpers from FFmpeg n8.0.1
// libavcodec/h264pred_template.c.

package h264

func h264Pred4x4Vertical(pix []uint8, offset int, stride int) error {
	if err := checkPredictionArgs(pix, offset, stride, 4, 4, 0, 1); err != nil {
		return err
	}
	for y := 0; y < 4; y++ {
		copy(pix[offset+y*stride:offset+y*stride+4], pix[offset-stride:offset-stride+4])
	}
	return nil
}

func h264Pred4x4Horizontal(pix []uint8, offset int, stride int) error {
	if err := checkPredictionArgs(pix, offset, stride, 4, 4, 1, 0); err != nil {
		return err
	}
	for y := 0; y < 4; y++ {
		fillPredictionRow(pix, offset+y*stride, 4, pix[offset-1+y*stride])
	}
	return nil
}

func h264Pred4x4DC(pix []uint8, offset int, stride int) error {
	if err := checkPredictionArgs(pix, offset, stride, 4, 4, 1, 1); err != nil {
		return err
	}
	dc := int(pix[offset-stride]) + int(pix[offset+1-stride]) + int(pix[offset+2-stride]) + int(pix[offset+3-stride]) +
		int(pix[offset-1]) + int(pix[offset-1+stride]) + int(pix[offset-1+2*stride]) + int(pix[offset-1+3*stride])
	fillPredictionBlock(pix, offset, stride, 4, 4, uint8((dc+4)>>3))
	return nil
}

func h264Pred4x4LeftDC(pix []uint8, offset int, stride int) error {
	if err := checkPredictionArgs(pix, offset, stride, 4, 4, 1, 0); err != nil {
		return err
	}
	dc := int(pix[offset-1]) + int(pix[offset-1+stride]) + int(pix[offset-1+2*stride]) + int(pix[offset-1+3*stride])
	fillPredictionBlock(pix, offset, stride, 4, 4, uint8((dc+2)>>2))
	return nil
}

func h264Pred4x4TopDC(pix []uint8, offset int, stride int) error {
	if err := checkPredictionArgs(pix, offset, stride, 4, 4, 0, 1); err != nil {
		return err
	}
	dc := int(pix[offset-stride]) + int(pix[offset+1-stride]) + int(pix[offset+2-stride]) + int(pix[offset+3-stride])
	fillPredictionBlock(pix, offset, stride, 4, 4, uint8((dc+2)>>2))
	return nil
}

func h264Pred4x4DC128(pix []uint8, offset int, stride int) error {
	return h264PredConstant(pix, offset, stride, 4, 4, 128)
}

func h264Pred4x4DownRight(pix []uint8, offset int, stride int) error {
	if err := checkPredictionArgs(pix, offset, stride, 4, 4, 1, 1); err != nil {
		return err
	}
	lt, t0, t1, t2, t3 := int(pix[offset-1-stride]), int(pix[offset-stride]), int(pix[offset+1-stride]), int(pix[offset+2-stride]), int(pix[offset+3-stride])
	l0, l1, l2, l3 := int(pix[offset-1]), int(pix[offset-1+stride]), int(pix[offset-1+2*stride]), int(pix[offset-1+3*stride])

	pix[offset+3*stride] = uint8((l3 + 2*l2 + l1 + 2) >> 2)
	v := uint8((l2 + 2*l1 + l0 + 2) >> 2)
	pix[offset+2*stride], pix[offset+1+3*stride] = v, v
	v = uint8((l1 + 2*l0 + lt + 2) >> 2)
	pix[offset+stride], pix[offset+1+2*stride], pix[offset+2+3*stride] = v, v, v
	v = uint8((l0 + 2*lt + t0 + 2) >> 2)
	pix[offset], pix[offset+1+stride], pix[offset+2+2*stride], pix[offset+3+3*stride] = v, v, v, v
	v = uint8((lt + 2*t0 + t1 + 2) >> 2)
	pix[offset+1], pix[offset+2+stride], pix[offset+3+2*stride] = v, v, v
	v = uint8((t0 + 2*t1 + t2 + 2) >> 2)
	pix[offset+2], pix[offset+3+stride] = v, v
	pix[offset+3] = uint8((t1 + 2*t2 + t3 + 2) >> 2)
	return nil
}

func h264Pred4x4DownLeft(pix []uint8, offset int, stride int, topRight []uint8) error {
	if err := checkPredictionArgs(pix, offset, stride, 4, 4, 0, 1); err != nil {
		return err
	}
	if len(topRight) < 4 {
		return ErrInvalidData
	}
	t0, t1, t2, t3 := int(pix[offset-stride]), int(pix[offset+1-stride]), int(pix[offset+2-stride]), int(pix[offset+3-stride])
	t4, t5, t6, t7 := int(topRight[0]), int(topRight[1]), int(topRight[2]), int(topRight[3])

	pix[offset] = uint8((t0 + t2 + 2*t1 + 2) >> 2)
	v := uint8((t1 + t3 + 2*t2 + 2) >> 2)
	pix[offset+1], pix[offset+stride] = v, v
	v = uint8((t2 + t4 + 2*t3 + 2) >> 2)
	pix[offset+2], pix[offset+1+stride], pix[offset+2*stride] = v, v, v
	v = uint8((t3 + t5 + 2*t4 + 2) >> 2)
	pix[offset+3], pix[offset+2+stride], pix[offset+1+2*stride], pix[offset+3*stride] = v, v, v, v
	v = uint8((t4 + t6 + 2*t5 + 2) >> 2)
	pix[offset+3+stride], pix[offset+2+2*stride], pix[offset+1+3*stride] = v, v, v
	v = uint8((t5 + t7 + 2*t6 + 2) >> 2)
	pix[offset+3+2*stride], pix[offset+2+3*stride] = v, v
	pix[offset+3+3*stride] = uint8((t6 + 3*t7 + 2) >> 2)
	return nil
}

func h264Pred4x4VerticalRight(pix []uint8, offset int, stride int) error {
	if err := checkPredictionArgs(pix, offset, stride, 4, 4, 1, 1); err != nil {
		return err
	}
	lt, t0, t1, t2, t3 := int(pix[offset-1-stride]), int(pix[offset-stride]), int(pix[offset+1-stride]), int(pix[offset+2-stride]), int(pix[offset+3-stride])
	l0, l1, l2 := int(pix[offset-1]), int(pix[offset-1+stride]), int(pix[offset-1+2*stride])

	v := uint8((lt + t0 + 1) >> 1)
	pix[offset], pix[offset+1+2*stride] = v, v
	v = uint8((t0 + t1 + 1) >> 1)
	pix[offset+1], pix[offset+2+2*stride] = v, v
	v = uint8((t1 + t2 + 1) >> 1)
	pix[offset+2], pix[offset+3+2*stride] = v, v
	pix[offset+3] = uint8((t2 + t3 + 1) >> 1)
	v = uint8((l0 + 2*lt + t0 + 2) >> 2)
	pix[offset+stride], pix[offset+1+3*stride] = v, v
	v = uint8((lt + 2*t0 + t1 + 2) >> 2)
	pix[offset+1+stride], pix[offset+2+3*stride] = v, v
	v = uint8((t0 + 2*t1 + t2 + 2) >> 2)
	pix[offset+2+stride], pix[offset+3+3*stride] = v, v
	pix[offset+3+stride] = uint8((t1 + 2*t2 + t3 + 2) >> 2)
	pix[offset+2*stride] = uint8((lt + 2*l0 + l1 + 2) >> 2)
	pix[offset+3*stride] = uint8((l0 + 2*l1 + l2 + 2) >> 2)
	return nil
}

func h264Pred4x4VerticalLeft(pix []uint8, offset int, stride int, topRight []uint8) error {
	if err := checkPredictionArgs(pix, offset, stride, 4, 4, 0, 1); err != nil {
		return err
	}
	if len(topRight) < 3 {
		return ErrInvalidData
	}
	t0, t1, t2, t3 := int(pix[offset-stride]), int(pix[offset+1-stride]), int(pix[offset+2-stride]), int(pix[offset+3-stride])
	t4, t5, t6 := int(topRight[0]), int(topRight[1]), int(topRight[2])

	pix[offset] = uint8((t0 + t1 + 1) >> 1)
	v := uint8((t1 + t2 + 1) >> 1)
	pix[offset+1], pix[offset+2*stride] = v, v
	v = uint8((t2 + t3 + 1) >> 1)
	pix[offset+2], pix[offset+1+2*stride] = v, v
	v = uint8((t3 + t4 + 1) >> 1)
	pix[offset+3], pix[offset+2+2*stride] = v, v
	pix[offset+3+2*stride] = uint8((t4 + t5 + 1) >> 1)
	pix[offset+stride] = uint8((t0 + 2*t1 + t2 + 2) >> 2)
	v = uint8((t1 + 2*t2 + t3 + 2) >> 2)
	pix[offset+1+stride], pix[offset+3*stride] = v, v
	v = uint8((t2 + 2*t3 + t4 + 2) >> 2)
	pix[offset+2+stride], pix[offset+1+3*stride] = v, v
	v = uint8((t3 + 2*t4 + t5 + 2) >> 2)
	pix[offset+3+stride], pix[offset+2+3*stride] = v, v
	pix[offset+3+3*stride] = uint8((t4 + 2*t5 + t6 + 2) >> 2)
	return nil
}

func h264Pred4x4HorizontalUp(pix []uint8, offset int, stride int) error {
	if err := checkPredictionArgs(pix, offset, stride, 4, 4, 1, 0); err != nil {
		return err
	}
	l0, l1, l2, l3 := int(pix[offset-1]), int(pix[offset-1+stride]), int(pix[offset-1+2*stride]), int(pix[offset-1+3*stride])

	pix[offset] = uint8((l0 + l1 + 1) >> 1)
	pix[offset+1] = uint8((l0 + 2*l1 + l2 + 2) >> 2)
	v := uint8((l1 + l2 + 1) >> 1)
	pix[offset+2], pix[offset+stride] = v, v
	v = uint8((l1 + 2*l2 + l3 + 2) >> 2)
	pix[offset+3], pix[offset+1+stride] = v, v
	v = uint8((l2 + l3 + 1) >> 1)
	pix[offset+2+stride], pix[offset+2*stride] = v, v
	v = uint8((l2 + 3*l3 + 2) >> 2)
	pix[offset+3+stride], pix[offset+1+2*stride] = v, v
	pix[offset+3+2*stride], pix[offset+1+3*stride], pix[offset+3*stride], pix[offset+2+2*stride], pix[offset+2+3*stride], pix[offset+3+3*stride] =
		uint8(l3), uint8(l3), uint8(l3), uint8(l3), uint8(l3), uint8(l3)
	return nil
}

func h264Pred4x4HorizontalDown(pix []uint8, offset int, stride int) error {
	if err := checkPredictionArgs(pix, offset, stride, 4, 4, 1, 1); err != nil {
		return err
	}
	lt, t0, t1, t2 := int(pix[offset-1-stride]), int(pix[offset-stride]), int(pix[offset+1-stride]), int(pix[offset+2-stride])
	l0, l1, l2, l3 := int(pix[offset-1]), int(pix[offset-1+stride]), int(pix[offset-1+2*stride]), int(pix[offset-1+3*stride])

	v := uint8((lt + l0 + 1) >> 1)
	pix[offset], pix[offset+2+stride] = v, v
	v = uint8((l0 + 2*lt + t0 + 2) >> 2)
	pix[offset+1], pix[offset+3+stride] = v, v
	pix[offset+2] = uint8((lt + 2*t0 + t1 + 2) >> 2)
	pix[offset+3] = uint8((t0 + 2*t1 + t2 + 2) >> 2)
	v = uint8((l0 + l1 + 1) >> 1)
	pix[offset+stride], pix[offset+2+2*stride] = v, v
	v = uint8((lt + 2*l0 + l1 + 2) >> 2)
	pix[offset+1+stride], pix[offset+3+2*stride] = v, v
	v = uint8((l1 + l2 + 1) >> 1)
	pix[offset+2*stride], pix[offset+2+3*stride] = v, v
	v = uint8((l0 + 2*l1 + l2 + 2) >> 2)
	pix[offset+1+2*stride], pix[offset+3+3*stride] = v, v
	pix[offset+3*stride] = uint8((l2 + l3 + 1) >> 1)
	pix[offset+1+3*stride] = uint8((l1 + 2*l2 + l3 + 2) >> 2)
	return nil
}

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

func h264Pred8x8MadCowDCL0T(pix []uint8, offset int, stride int) error {
	if err := h264Pred8x8TopDC(pix, offset, stride); err != nil {
		return err
	}
	return h264Pred4x4DC(pix, offset, stride)
}

func h264Pred8x16MadCowDCL0T(pix []uint8, offset int, stride int) error {
	if err := h264Pred8x16TopDC(pix, offset, stride); err != nil {
		return err
	}
	return h264Pred4x4DC(pix, offset, stride)
}

func h264Pred8x8MadCowDC0LT(pix []uint8, offset int, stride int) error {
	if err := h264Pred8x8DC(pix, offset, stride); err != nil {
		return err
	}
	return h264Pred4x4TopDC(pix, offset, stride)
}

func h264Pred8x16MadCowDC0LT(pix []uint8, offset int, stride int) error {
	if err := h264Pred8x16DC(pix, offset, stride); err != nil {
		return err
	}
	return h264Pred4x4TopDC(pix, offset, stride)
}

func h264Pred8x8MadCowDCL00(pix []uint8, offset int, stride int) error {
	if err := h264Pred8x8LeftDC(pix, offset, stride); err != nil {
		return err
	}
	if err := h264Pred4x4DC128(pix, offset+4*stride, stride); err != nil {
		return err
	}
	return h264Pred4x4DC128(pix, offset+4*stride+4, stride)
}

func h264Pred8x16MadCowDCL00(pix []uint8, offset int, stride int) error {
	if err := h264Pred8x16LeftDC(pix, offset, stride); err != nil {
		return err
	}
	if err := h264Pred4x4DC128(pix, offset+4*stride, stride); err != nil {
		return err
	}
	return h264Pred4x4DC128(pix, offset+4*stride+4, stride)
}

func h264Pred8x8MadCowDC0L0(pix []uint8, offset int, stride int) error {
	if err := h264Pred8x8LeftDC(pix, offset, stride); err != nil {
		return err
	}
	if err := h264Pred4x4DC128(pix, offset, stride); err != nil {
		return err
	}
	return h264Pred4x4DC128(pix, offset+4, stride)
}

func h264Pred8x16MadCowDC0L0(pix []uint8, offset int, stride int) error {
	if err := h264Pred8x16LeftDC(pix, offset, stride); err != nil {
		return err
	}
	if err := h264Pred4x4DC128(pix, offset, stride); err != nil {
		return err
	}
	return h264Pred4x4DC128(pix, offset+4, stride)
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

func h264Pred8x8LDC128(pix []uint8, offset int, stride int, hasTopLeft bool, hasTopRight bool) error {
	_ = hasTopLeft
	_ = hasTopRight
	return h264PredConstant(pix, offset, stride, 8, 8, 128)
}

func h264Pred8x8LLeftDC(pix []uint8, offset int, stride int, hasTopLeft bool, hasTopRight bool) error {
	_ = hasTopRight
	if err := checkPrediction8x8LArgs(pix, offset, stride, true, hasTopLeft, -1); err != nil {
		return err
	}
	left := h264Pred8x8LLoadLeft(pix, offset, stride, hasTopLeft)
	dc := 0
	for i := 0; i < 8; i++ {
		dc += left[i]
	}
	fillPredictionBlock(pix, offset, stride, 8, 8, uint8((dc+4)>>3))
	return nil
}

func h264Pred8x8LTopDC(pix []uint8, offset int, stride int, hasTopLeft bool, hasTopRight bool) error {
	if err := checkPrediction8x8LArgs(pix, offset, stride, hasTopLeft, true, h264Pred8x8LTopMaxX(hasTopRight)); err != nil {
		return err
	}
	top := h264Pred8x8LLoadTop(pix, offset, stride, hasTopLeft, hasTopRight)
	dc := 0
	for i := 0; i < 8; i++ {
		dc += top[i]
	}
	fillPredictionBlock(pix, offset, stride, 8, 8, uint8((dc+4)>>3))
	return nil
}

func h264Pred8x8LDC(pix []uint8, offset int, stride int, hasTopLeft bool, hasTopRight bool) error {
	if err := checkPrediction8x8LArgs(pix, offset, stride, true, true, h264Pred8x8LTopMaxX(hasTopRight)); err != nil {
		return err
	}
	left := h264Pred8x8LLoadLeft(pix, offset, stride, hasTopLeft)
	top := h264Pred8x8LLoadTop(pix, offset, stride, hasTopLeft, hasTopRight)
	dc := 0
	for i := 0; i < 8; i++ {
		dc += left[i] + top[i]
	}
	fillPredictionBlock(pix, offset, stride, 8, 8, uint8((dc+8)>>4))
	return nil
}

func h264Pred8x8LHorizontal(pix []uint8, offset int, stride int, hasTopLeft bool, hasTopRight bool) error {
	_ = hasTopRight
	if err := checkPrediction8x8LArgs(pix, offset, stride, true, hasTopLeft, -1); err != nil {
		return err
	}
	left := h264Pred8x8LLoadLeft(pix, offset, stride, hasTopLeft)
	for y := 0; y < 8; y++ {
		fillPredictionRow(pix, offset+y*stride, 8, uint8(left[y]))
	}
	return nil
}

func h264Pred8x8LVertical(pix []uint8, offset int, stride int, hasTopLeft bool, hasTopRight bool) error {
	if err := checkPrediction8x8LArgs(pix, offset, stride, hasTopLeft, true, h264Pred8x8LTopMaxX(hasTopRight)); err != nil {
		return err
	}
	top := h264Pred8x8LLoadTop(pix, offset, stride, hasTopLeft, hasTopRight)
	for x := 0; x < 8; x++ {
		pix[offset+x] = uint8(top[x])
	}
	for y := 1; y < 8; y++ {
		copy(pix[offset+y*stride:offset+y*stride+8], pix[offset:offset+8])
	}
	return nil
}

func h264Pred8x8LDownLeft(pix []uint8, offset int, stride int, hasTopLeft bool, hasTopRight bool) error {
	if err := checkPrediction8x8LArgs(pix, offset, stride, hasTopLeft, true, h264Pred8x8LTopRightMaxX(hasTopRight)); err != nil {
		return err
	}
	top := h264Pred8x8LLoadTop(pix, offset, stride, hasTopLeft, hasTopRight)
	h264Pred8x8LLoadTopRight(pix, offset, stride, hasTopRight, &top)
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			sum := x + y
			if sum < 14 {
				pix[offset+x+y*stride] = h264PredAvg3(top[sum], top[sum+1], top[sum+2])
			} else {
				pix[offset+x+y*stride] = uint8((top[14] + 3*top[15] + 2) >> 2)
			}
		}
	}
	return nil
}

func h264Pred8x8LDownRight(pix []uint8, offset int, stride int, hasTopLeft bool, hasTopRight bool) error {
	if err := checkPrediction8x8LArgs(pix, offset, stride, true, true, h264Pred8x8LTopMaxX(hasTopRight)); err != nil {
		return err
	}
	top := h264Pred8x8LLoadTop(pix, offset, stride, hasTopLeft, hasTopRight)
	left := h264Pred8x8LLoadLeft(pix, offset, stride, hasTopLeft)
	topLeft := h264Pred8x8LLoadTopLeft(pix, offset, stride)
	edge := [17]int{
		left[7], left[6], left[5], left[4], left[3], left[2], left[1], left[0],
		topLeft,
		top[0], top[1], top[2], top[3], top[4], top[5], top[6], top[7],
	}
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			i := x - y + 7
			pix[offset+x+y*stride] = h264PredAvg3(edge[i], edge[i+1], edge[i+2])
		}
	}
	return nil
}

func h264Pred8x8LVerticalRight(pix []uint8, offset int, stride int, hasTopLeft bool, hasTopRight bool) error {
	if err := checkPrediction8x8LArgs(pix, offset, stride, true, true, h264Pred8x8LTopMaxX(hasTopRight)); err != nil {
		return err
	}
	top := h264Pred8x8LLoadTop(pix, offset, stride, hasTopLeft, hasTopRight)
	left := h264Pred8x8LLoadLeft(pix, offset, stride, hasTopLeft)
	topLeft := h264Pred8x8LLoadTopLeft(pix, offset, stride)
	topEdge := [9]int{topLeft, top[0], top[1], top[2], top[3], top[4], top[5], top[6], top[7]}
	leftEdge := [10]int{left[7], left[6], left[5], left[4], left[3], left[2], left[1], left[0], topLeft, top[0]}
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			z := 2*x - y
			if z >= 0 {
				i := z >> 1
				if z&1 == 0 {
					pix[offset+x+y*stride] = h264PredAvg2(topEdge[i], topEdge[i+1])
				} else {
					pix[offset+x+y*stride] = h264PredAvg3(topEdge[i], topEdge[i+1], topEdge[i+2])
				}
			} else {
				i := 8 + z
				pix[offset+x+y*stride] = h264PredAvg3(leftEdge[i], leftEdge[i+1], leftEdge[i+2])
			}
		}
	}
	return nil
}

func h264Pred8x8LHorizontalDown(pix []uint8, offset int, stride int, hasTopLeft bool, hasTopRight bool) error {
	if err := checkPrediction8x8LArgs(pix, offset, stride, true, true, h264Pred8x8LTopMaxX(hasTopRight)); err != nil {
		return err
	}
	top := h264Pred8x8LLoadTop(pix, offset, stride, hasTopLeft, hasTopRight)
	left := h264Pred8x8LLoadLeft(pix, offset, stride, hasTopLeft)
	topLeft := h264Pred8x8LLoadTopLeft(pix, offset, stride)
	leftEdge := [9]int{topLeft, left[0], left[1], left[2], left[3], left[4], left[5], left[6], left[7]}
	topEdge := [9]int{topLeft, top[0], top[1], top[2], top[3], top[4], top[5], top[6], top[7]}
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			z := 2*y - x
			if z >= 0 {
				i := z >> 1
				if z&1 == 0 {
					pix[offset+x+y*stride] = h264PredAvg2(leftEdge[i], leftEdge[i+1])
				} else {
					pix[offset+x+y*stride] = h264PredAvg3(leftEdge[i], leftEdge[i+1], leftEdge[i+2])
				}
			} else if z == -1 {
				pix[offset+x+y*stride] = h264PredAvg3(left[0], topLeft, top[0])
			} else {
				i := -z
				pix[offset+x+y*stride] = h264PredAvg3(topEdge[i], topEdge[i-1], topEdge[i-2])
			}
		}
	}
	return nil
}

func h264Pred8x8LVerticalLeft(pix []uint8, offset int, stride int, hasTopLeft bool, hasTopRight bool) error {
	if err := checkPrediction8x8LArgs(pix, offset, stride, hasTopLeft, true, h264Pred8x8LTopRightMaxX(hasTopRight)); err != nil {
		return err
	}
	top := h264Pred8x8LLoadTop(pix, offset, stride, hasTopLeft, hasTopRight)
	h264Pred8x8LLoadTopRight(pix, offset, stride, hasTopRight, &top)
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			z := y + 2*x
			i := z >> 1
			if z&1 == 0 {
				pix[offset+x+y*stride] = h264PredAvg2(top[i], top[i+1])
			} else {
				pix[offset+x+y*stride] = h264PredAvg3(top[i], top[i+1], top[i+2])
			}
		}
	}
	return nil
}

func h264Pred8x8LHorizontalUp(pix []uint8, offset int, stride int, hasTopLeft bool, hasTopRight bool) error {
	_ = hasTopRight
	if err := checkPrediction8x8LArgs(pix, offset, stride, true, hasTopLeft, -1); err != nil {
		return err
	}
	left := h264Pred8x8LLoadLeft(pix, offset, stride, hasTopLeft)
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			z := x + 2*y
			if z < 13 {
				i := z >> 1
				if z&1 == 0 {
					pix[offset+x+y*stride] = h264PredAvg2(left[i], left[i+1])
				} else {
					pix[offset+x+y*stride] = h264PredAvg3(left[i], left[i+1], left[i+2])
				}
			} else if z == 13 {
				pix[offset+x+y*stride] = uint8((left[6] + 3*left[7] + 2) >> 2)
			} else {
				pix[offset+x+y*stride] = uint8(left[7])
			}
		}
	}
	return nil
}

func h264Pred8x8LVerticalFilterAdd(pix []uint8, offset int, block []int32, stride int, hasTopLeft bool, hasTopRight bool) error {
	if len(block) < 64 {
		return ErrInvalidData
	}
	if err := checkPrediction8x8LArgs(pix, offset, stride, hasTopLeft, true, h264Pred8x8LTopMaxX(hasTopRight)); err != nil {
		return err
	}
	top := h264Pred8x8LLoadTop(pix, offset, stride, hasTopLeft, hasTopRight)
	for x := 0; x < 8; x++ {
		v := uint8(top[x])
		for y := 0; y < 7; y++ {
			v += uint8(dctcoef8Value(block[y*8+x]))
			pix[offset+x+y*stride] = v
		}
		pix[offset+x+7*stride] = v + uint8(dctcoef8Value(block[56+x]))
	}
	clearInt32(block[:64])
	return nil
}

func h264Pred8x8LHorizontalFilterAdd(pix []uint8, offset int, block []int32, stride int, hasTopLeft bool, hasTopRight bool) error {
	_ = hasTopRight
	if len(block) < 64 {
		return ErrInvalidData
	}
	if err := checkPrediction8x8LArgs(pix, offset, stride, true, hasTopLeft, -1); err != nil {
		return err
	}
	left := h264Pred8x8LLoadLeft(pix, offset, stride, hasTopLeft)
	for y := 0; y < 8; y++ {
		row := offset + y*stride
		src := y * 8
		v := uint8(left[y])
		for x := 0; x < 7; x++ {
			v += uint8(dctcoef8Value(block[src+x]))
			pix[row+x] = v
		}
		pix[row+7] = v + uint8(dctcoef8Value(block[src+7]))
	}
	clearInt32(block[:64])
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

func h264Pred8x8LLoadLeft(pix []uint8, offset int, stride int, hasTopLeft bool) [8]int {
	topLeft := pix[offset-1]
	if hasTopLeft {
		topLeft = pix[offset-1-stride]
	}
	var left [8]int
	left[0] = (int(topLeft) + 2*int(pix[offset-1]) + int(pix[offset-1+stride]) + 2) >> 2
	for y := 1; y < 7; y++ {
		left[y] = (int(pix[offset-1+(y-1)*stride]) + 2*int(pix[offset-1+y*stride]) + int(pix[offset-1+(y+1)*stride]) + 2) >> 2
	}
	left[7] = (int(pix[offset-1+6*stride]) + 3*int(pix[offset-1+7*stride]) + 2) >> 2
	return left
}

func h264Pred8x8LLoadTop(pix []uint8, offset int, stride int, hasTopLeft bool, hasTopRight bool) [16]int {
	topLeft := pix[offset-stride]
	if hasTopLeft {
		topLeft = pix[offset-1-stride]
	}
	var top [16]int
	top[0] = (int(topLeft) + 2*int(pix[offset-stride]) + int(pix[offset+1-stride]) + 2) >> 2
	for x := 1; x < 7; x++ {
		top[x] = (int(pix[offset+x-1-stride]) + 2*int(pix[offset+x-stride]) + int(pix[offset+x+1-stride]) + 2) >> 2
	}
	topRight := pix[offset+7-stride]
	if hasTopRight {
		topRight = pix[offset+8-stride]
	}
	top[7] = (int(topRight) + 2*int(pix[offset+7-stride]) + int(pix[offset+6-stride]) + 2) >> 2
	return top
}

func h264Pred8x8LLoadTopRight(pix []uint8, offset int, stride int, hasTopRight bool, top *[16]int) {
	if hasTopRight {
		for x := 8; x < 15; x++ {
			top[x] = (int(pix[offset+x-1-stride]) + 2*int(pix[offset+x-stride]) + int(pix[offset+x+1-stride]) + 2) >> 2
		}
		top[15] = (int(pix[offset+14-stride]) + 3*int(pix[offset+15-stride]) + 2) >> 2
		return
	}
	v := int(pix[offset+7-stride])
	for x := 8; x < 16; x++ {
		top[x] = v
	}
}

func h264Pred8x8LLoadTopLeft(pix []uint8, offset int, stride int) int {
	return (int(pix[offset-1]) + 2*int(pix[offset-1-stride]) + int(pix[offset-stride]) + 2) >> 2
}

func h264PredAvg2(a int, b int) uint8 {
	return uint8((a + b + 1) >> 1)
}

func h264PredAvg3(a int, b int, c int) uint8 {
	return uint8((a + 2*b + c + 2) >> 2)
}

func h264Pred8x8LTopMaxX(hasTopRight bool) int {
	if hasTopRight {
		return 8
	}
	return 7
}

func h264Pred8x8LTopRightMaxX(hasTopRight bool) int {
	if hasTopRight {
		return 15
	}
	return 7
}

func checkPrediction8x8LArgs(pix []uint8, offset int, stride int, left bool, top bool, topRightMaxX int) error {
	leftMargin := 0
	if left {
		leftMargin = 1
	}
	topMargin := 0
	if top {
		topMargin = 1
	}
	if err := checkPredictionArgs(pix, offset, stride, 8, 8, leftMargin, topMargin); err != nil {
		return err
	}
	if top && topRightMaxX >= 0 {
		maxTopIndex := offset - stride + topRightMaxX
		if maxTopIndex < 0 || maxTopIndex >= len(pix) {
			return ErrInvalidData
		}
	}
	return nil
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
