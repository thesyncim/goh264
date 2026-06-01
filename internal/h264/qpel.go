// SPDX-License-Identifier: LGPL-2.1-or-later
//
// Source-shaped H.264 luma quarter-pel motion compensation helpers from
// FFmpeg n8.0.1 libavcodec/h264qpel_template.c.

package h264

func h264PutH264QpelMC(dst []uint8, dstOffset int, src []uint8, srcOffset int, stride int, size int, mx int, my int) error {
	return h264QpelMC(dst, dstOffset, src, srcOffset, stride, size, mx, my, false)
}

func h264AvgH264QpelMC(dst []uint8, dstOffset int, src []uint8, srcOffset int, stride int, size int, mx int, my int) error {
	return h264QpelMC(dst, dstOffset, src, srcOffset, stride, size, mx, my, true)
}

func h264PutH264QpelMCHigh(dst []uint16, dstOffset int, src []uint16, srcOffset int, stride int, size int, mx int, my int, bitDepth int) error {
	return h264QpelMCHigh(dst, dstOffset, src, srcOffset, stride, size, mx, my, false, bitDepth)
}

func h264AvgH264QpelMCHigh(dst []uint16, dstOffset int, src []uint16, srcOffset int, stride int, size int, mx int, my int, bitDepth int) error {
	return h264QpelMCHigh(dst, dstOffset, src, srcOffset, stride, size, mx, my, true, bitDepth)
}

func h264QpelMC(dst []uint8, dstOffset int, src []uint8, srcOffset int, stride int, size int, mx int, my int, avg bool) error {
	return h264QpelMCStrides(dst, dstOffset, stride, src, srcOffset, stride, size, mx, my, avg)
}

func h264QpelMCHigh(dst []uint16, dstOffset int, src []uint16, srcOffset int, stride int, size int, mx int, my int, avg bool, bitDepth int) error {
	return h264QpelMCStridesHigh(dst, dstOffset, stride, src, srcOffset, stride, size, mx, my, avg, bitDepth)
}

func h264QpelMCStrides(dst []uint8, dstOffset int, dstStride int, src []uint8, srcOffset int, srcStride int, size int, mx int, my int, avg bool) error {
	if err := checkH264QpelArgs(dst, dstOffset, dstStride, src, srcOffset, srcStride, size, mx, my); err != nil {
		return err
	}
	var pred [16 * 16]uint8
	var a [16 * 16]uint8
	var b [16 * 16]uint8

	switch my*4 + mx {
	case 0:
		h264QpelCopyPred(&pred, src, srcOffset, srcStride, size)
	case 1:
		h264QpelHPred(&a, src, srcOffset, srcStride, size, 0)
		h264QpelCopyPred(&b, src, srcOffset, srcStride, size)
		h264QpelAvgPred(&pred, &b, &a, size)
	case 2:
		h264QpelHPred(&pred, src, srcOffset, srcStride, size, 0)
	case 3:
		h264QpelHPred(&a, src, srcOffset, srcStride, size, 0)
		h264QpelCopyPred(&b, src, srcOffset+1, srcStride, size)
		h264QpelAvgPred(&pred, &b, &a, size)
	case 4:
		h264QpelVPred(&a, src, srcOffset, srcStride, size, 0)
		h264QpelCopyPred(&b, src, srcOffset, srcStride, size)
		h264QpelAvgPred(&pred, &b, &a, size)
	case 5:
		h264QpelHPred(&a, src, srcOffset, srcStride, size, 0)
		h264QpelVPred(&b, src, srcOffset, srcStride, size, 0)
		h264QpelAvgPred(&pred, &a, &b, size)
	case 6:
		h264QpelHPred(&a, src, srcOffset, srcStride, size, 0)
		h264QpelHVPred(&b, src, srcOffset, srcStride, size)
		h264QpelAvgPred(&pred, &a, &b, size)
	case 7:
		h264QpelHPred(&a, src, srcOffset, srcStride, size, 0)
		h264QpelVPred(&b, src, srcOffset+1, srcStride, size, 0)
		h264QpelAvgPred(&pred, &a, &b, size)
	case 8:
		h264QpelVPred(&pred, src, srcOffset, srcStride, size, 0)
	case 9:
		h264QpelVPred(&a, src, srcOffset, srcStride, size, 0)
		h264QpelHVPred(&b, src, srcOffset, srcStride, size)
		h264QpelAvgPred(&pred, &a, &b, size)
	case 10:
		h264QpelHVPred(&pred, src, srcOffset, srcStride, size)
	case 11:
		h264QpelVPred(&a, src, srcOffset+1, srcStride, size, 0)
		h264QpelHVPred(&b, src, srcOffset, srcStride, size)
		h264QpelAvgPred(&pred, &a, &b, size)
	case 12:
		h264QpelVPred(&a, src, srcOffset, srcStride, size, 0)
		h264QpelCopyPred(&b, src, srcOffset+srcStride, srcStride, size)
		h264QpelAvgPred(&pred, &b, &a, size)
	case 13:
		h264QpelHPred(&a, src, srcOffset, srcStride, size, 1)
		h264QpelVPred(&b, src, srcOffset, srcStride, size, 0)
		h264QpelAvgPred(&pred, &a, &b, size)
	case 14:
		h264QpelHPred(&a, src, srcOffset, srcStride, size, 1)
		h264QpelHVPred(&b, src, srcOffset, srcStride, size)
		h264QpelAvgPred(&pred, &a, &b, size)
	case 15:
		h264QpelHPred(&a, src, srcOffset, srcStride, size, 1)
		h264QpelVPred(&b, src, srcOffset+1, srcStride, size, 0)
		h264QpelAvgPred(&pred, &a, &b, size)
	default:
		return ErrInvalidData
	}

	h264QpelStorePred(dst, dstOffset, dstStride, &pred, size, avg)
	return nil
}

func h264QpelMCStridesHigh(dst []uint16, dstOffset int, dstStride int, src []uint16, srcOffset int, srcStride int, size int, mx int, my int, avg bool, bitDepth int) error {
	if err := checkH264DSPHighBitDepth(bitDepth); err != nil {
		return err
	}
	if err := checkH264QpelArgsHigh(dst, dstOffset, dstStride, src, srcOffset, srcStride, size, mx, my); err != nil {
		return err
	}
	var pred [16 * 16]uint16
	var a [16 * 16]uint16
	var b [16 * 16]uint16

	switch my*4 + mx {
	case 0:
		h264QpelCopyPredHigh(&pred, src, srcOffset, srcStride, size)
	case 1:
		h264QpelHPredHigh(&a, src, srcOffset, srcStride, size, 0, bitDepth)
		h264QpelCopyPredHigh(&b, src, srcOffset, srcStride, size)
		h264QpelAvgPredHigh(&pred, &b, &a, size)
	case 2:
		h264QpelHPredHigh(&pred, src, srcOffset, srcStride, size, 0, bitDepth)
	case 3:
		h264QpelHPredHigh(&a, src, srcOffset, srcStride, size, 0, bitDepth)
		h264QpelCopyPredHigh(&b, src, srcOffset+1, srcStride, size)
		h264QpelAvgPredHigh(&pred, &b, &a, size)
	case 4:
		h264QpelVPredHigh(&a, src, srcOffset, srcStride, size, 0, bitDepth)
		h264QpelCopyPredHigh(&b, src, srcOffset, srcStride, size)
		h264QpelAvgPredHigh(&pred, &b, &a, size)
	case 5:
		h264QpelHPredHigh(&a, src, srcOffset, srcStride, size, 0, bitDepth)
		h264QpelVPredHigh(&b, src, srcOffset, srcStride, size, 0, bitDepth)
		h264QpelAvgPredHigh(&pred, &a, &b, size)
	case 6:
		h264QpelHPredHigh(&a, src, srcOffset, srcStride, size, 0, bitDepth)
		h264QpelHVPredHigh(&b, src, srcOffset, srcStride, size, bitDepth)
		h264QpelAvgPredHigh(&pred, &a, &b, size)
	case 7:
		h264QpelHPredHigh(&a, src, srcOffset, srcStride, size, 0, bitDepth)
		h264QpelVPredHigh(&b, src, srcOffset+1, srcStride, size, 0, bitDepth)
		h264QpelAvgPredHigh(&pred, &a, &b, size)
	case 8:
		h264QpelVPredHigh(&pred, src, srcOffset, srcStride, size, 0, bitDepth)
	case 9:
		h264QpelVPredHigh(&a, src, srcOffset, srcStride, size, 0, bitDepth)
		h264QpelHVPredHigh(&b, src, srcOffset, srcStride, size, bitDepth)
		h264QpelAvgPredHigh(&pred, &a, &b, size)
	case 10:
		h264QpelHVPredHigh(&pred, src, srcOffset, srcStride, size, bitDepth)
	case 11:
		h264QpelVPredHigh(&a, src, srcOffset+1, srcStride, size, 0, bitDepth)
		h264QpelHVPredHigh(&b, src, srcOffset, srcStride, size, bitDepth)
		h264QpelAvgPredHigh(&pred, &a, &b, size)
	case 12:
		h264QpelVPredHigh(&a, src, srcOffset, srcStride, size, 0, bitDepth)
		h264QpelCopyPredHigh(&b, src, srcOffset+srcStride, srcStride, size)
		h264QpelAvgPredHigh(&pred, &b, &a, size)
	case 13:
		h264QpelHPredHigh(&a, src, srcOffset, srcStride, size, 1, bitDepth)
		h264QpelVPredHigh(&b, src, srcOffset, srcStride, size, 0, bitDepth)
		h264QpelAvgPredHigh(&pred, &a, &b, size)
	case 14:
		h264QpelHPredHigh(&a, src, srcOffset, srcStride, size, 1, bitDepth)
		h264QpelHVPredHigh(&b, src, srcOffset, srcStride, size, bitDepth)
		h264QpelAvgPredHigh(&pred, &a, &b, size)
	case 15:
		h264QpelHPredHigh(&a, src, srcOffset, srcStride, size, 1, bitDepth)
		h264QpelVPredHigh(&b, src, srcOffset+1, srcStride, size, 0, bitDepth)
		h264QpelAvgPredHigh(&pred, &a, &b, size)
	default:
		return ErrInvalidData
	}

	h264QpelStorePredHigh(dst, dstOffset, dstStride, &pred, size, avg)
	return nil
}

func h264QpelCopyPred(out *[16 * 16]uint8, src []uint8, srcOffset int, stride int, size int) {
	for y := 0; y < size; y++ {
		copy(out[y*size:y*size+size], src[srcOffset+y*stride:srcOffset+y*stride+size])
	}
}

func h264QpelCopyPredHigh(out *[16 * 16]uint16, src []uint16, srcOffset int, stride int, size int) {
	for y := 0; y < size; y++ {
		copy(out[y*size:y*size+size], src[srcOffset+y*stride:srcOffset+y*stride+size])
	}
}

func h264QpelHPred(out *[16 * 16]uint8, src []uint8, srcOffset int, stride int, size int, yOffset int) {
	for y := 0; y < size; y++ {
		row := srcOffset + (y+yOffset)*stride
		for x := 0; x < size; x++ {
			v := (int(src[row+x])+int(src[row+x+1]))*20 -
				(int(src[row+x-1])+int(src[row+x+2]))*5 +
				(int(src[row+x-2]) + int(src[row+x+3]))
			out[y*size+x] = clipUint8((v + 16) >> 5)
		}
	}
}

func h264QpelHPredHigh(out *[16 * 16]uint16, src []uint16, srcOffset int, stride int, size int, yOffset int, bitDepth int) {
	for y := 0; y < size; y++ {
		row := srcOffset + (y+yOffset)*stride
		for x := 0; x < size; x++ {
			v := (int(src[row+x])+int(src[row+x+1]))*20 -
				(int(src[row+x-1])+int(src[row+x+2]))*5 +
				(int(src[row+x-2]) + int(src[row+x+3]))
			out[y*size+x] = clipUintBitDepth((v+16)>>5, bitDepth)
		}
	}
}

func h264QpelVPred(out *[16 * 16]uint8, src []uint8, srcOffset int, stride int, size int, xOffset int) {
	for x := 0; x < size; x++ {
		col := srcOffset + x + xOffset
		for y := 0; y < size; y++ {
			row := col + y*stride
			v := (int(src[row])+int(src[row+stride]))*20 -
				(int(src[row-stride])+int(src[row+2*stride]))*5 +
				(int(src[row-2*stride]) + int(src[row+3*stride]))
			out[y*size+x] = clipUint8((v + 16) >> 5)
		}
	}
}

func h264QpelVPredHigh(out *[16 * 16]uint16, src []uint16, srcOffset int, stride int, size int, xOffset int, bitDepth int) {
	for x := 0; x < size; x++ {
		col := srcOffset + x + xOffset
		for y := 0; y < size; y++ {
			row := col + y*stride
			v := (int(src[row])+int(src[row+stride]))*20 -
				(int(src[row-stride])+int(src[row+2*stride]))*5 +
				(int(src[row-2*stride]) + int(src[row+3*stride]))
			out[y*size+x] = clipUintBitDepth((v+16)>>5, bitDepth)
		}
	}
}

func h264QpelHVPred(out *[16 * 16]uint8, src []uint8, srcOffset int, stride int, size int) {
	var tmp [16 * (16 + 5)]int
	for y := -2; y < size+3; y++ {
		row := srcOffset + y*stride
		tmpRow := (y + 2) * size
		for x := 0; x < size; x++ {
			tmp[tmpRow+x] = (int(src[row+x])+int(src[row+x+1]))*20 -
				(int(src[row+x-1])+int(src[row+x+2]))*5 +
				(int(src[row+x-2]) + int(src[row+x+3]))
		}
	}
	for x := 0; x < size; x++ {
		for y := 0; y < size; y++ {
			row := (y + 2) * size
			v := (tmp[row+x]+tmp[row+size+x])*20 -
				(tmp[row-size+x]+tmp[row+2*size+x])*5 +
				(tmp[row-2*size+x] + tmp[row+3*size+x])
			out[y*size+x] = clipUint8((v + 512) >> 10)
		}
	}
}

func h264QpelHVPredHigh(out *[16 * 16]uint16, src []uint16, srcOffset int, stride int, size int, bitDepth int) {
	var tmp [16 * (16 + 5)]int
	for y := -2; y < size+3; y++ {
		row := srcOffset + y*stride
		tmpRow := (y + 2) * size
		for x := 0; x < size; x++ {
			tmp[tmpRow+x] = (int(src[row+x])+int(src[row+x+1]))*20 -
				(int(src[row+x-1])+int(src[row+x+2]))*5 +
				(int(src[row+x-2]) + int(src[row+x+3]))
		}
	}
	for x := 0; x < size; x++ {
		for y := 0; y < size; y++ {
			row := (y + 2) * size
			v := (tmp[row+x]+tmp[row+size+x])*20 -
				(tmp[row-size+x]+tmp[row+2*size+x])*5 +
				(tmp[row-2*size+x] + tmp[row+3*size+x])
			out[y*size+x] = clipUintBitDepth((v+512)>>10, bitDepth)
		}
	}
}

func h264QpelAvgPred(dst *[16 * 16]uint8, a *[16 * 16]uint8, b *[16 * 16]uint8, size int) {
	n := size * size
	for i := 0; i < n; i++ {
		dst[i] = uint8((int(a[i]) + int(b[i]) + 1) >> 1)
	}
}

func h264QpelAvgPredHigh(dst *[16 * 16]uint16, a *[16 * 16]uint16, b *[16 * 16]uint16, size int) {
	n := size * size
	for i := 0; i < n; i++ {
		dst[i] = uint16((int(a[i]) + int(b[i]) + 1) >> 1)
	}
}

func h264QpelStorePred(dst []uint8, dstOffset int, stride int, pred *[16 * 16]uint8, size int, avg bool) {
	for y := 0; y < size; y++ {
		row := dstOffset + y*stride
		for x := 0; x < size; x++ {
			v := pred[y*size+x]
			if avg {
				dst[row+x] = uint8((int(dst[row+x]) + int(v) + 1) >> 1)
			} else {
				dst[row+x] = v
			}
		}
	}
}

func h264QpelStorePredHigh(dst []uint16, dstOffset int, stride int, pred *[16 * 16]uint16, size int, avg bool) {
	for y := 0; y < size; y++ {
		row := dstOffset + y*stride
		for x := 0; x < size; x++ {
			v := pred[y*size+x]
			if avg {
				dst[row+x] = uint16((int(dst[row+x]) + int(v) + 1) >> 1)
			} else {
				dst[row+x] = v
			}
		}
	}
}

func checkH264QpelArgs(dst []uint8, dstOffset int, dstStride int, src []uint8, srcOffset int, srcStride int, size int, mx int, my int) error {
	if dstOffset < 0 || srcOffset < 0 || dstStride <= 0 || srcStride <= 0 || mx < 0 || mx >= 4 || my < 0 || my >= 4 {
		return ErrInvalidData
	}
	if size != 2 && size != 4 && size != 8 && size != 16 {
		return ErrInvalidData
	}
	if dstStride < size || srcStride < size {
		return ErrInvalidData
	}
	dstMax := dstOffset + (size-1)*dstStride + size - 1
	if dstMax >= len(dst) {
		return ErrInvalidData
	}

	minX, minY, maxX, maxY := h264QpelSourceBounds(size, mx, my)
	minIndex := srcOffset + minY*srcStride + minX
	maxIndex := srcOffset + maxY*srcStride + maxX
	if minIndex < 0 || maxIndex >= len(src) {
		return ErrInvalidData
	}
	return nil
}

func checkH264QpelArgsHigh(dst []uint16, dstOffset int, dstStride int, src []uint16, srcOffset int, srcStride int, size int, mx int, my int) error {
	if dstOffset < 0 || srcOffset < 0 || dstStride <= 0 || srcStride <= 0 || mx < 0 || mx >= 4 || my < 0 || my >= 4 {
		return ErrInvalidData
	}
	if size != 2 && size != 4 && size != 8 && size != 16 {
		return ErrInvalidData
	}
	if dstStride < size || srcStride < size {
		return ErrInvalidData
	}
	dstMax := dstOffset + (size-1)*dstStride + size - 1
	if dstMax >= len(dst) {
		return ErrInvalidData
	}

	minX, minY, maxX, maxY := h264QpelSourceBounds(size, mx, my)
	minIndex := srcOffset + minY*srcStride + minX
	maxIndex := srcOffset + maxY*srcStride + maxX
	if minIndex < 0 || maxIndex >= len(src) {
		return ErrInvalidData
	}
	return nil
}

func h264QpelSourceBounds(size int, mx int, my int) (int, int, int, int) {
	minX, minY, maxX, maxY := 0, 0, size-1, size-1
	merge := func(x0 int, y0 int, x1 int, y1 int) {
		if x0 < minX {
			minX = x0
		}
		if y0 < minY {
			minY = y0
		}
		if x1 > maxX {
			maxX = x1
		}
		if y1 > maxY {
			maxY = y1
		}
	}
	includeCurrent := func(xOffset int, yOffset int) {
		merge(xOffset, yOffset, xOffset+size-1, yOffset+size-1)
	}
	includeH := func(yOffset int) {
		merge(-2, yOffset, size+2, yOffset+size-1)
	}
	includeV := func(xOffset int) {
		merge(xOffset, -2, xOffset+size-1, size+2)
	}
	includeHV := func() {
		merge(-2, -2, size+2, size+2)
	}

	switch my*4 + mx {
	case 0:
		includeCurrent(0, 0)
	case 1:
		includeCurrent(0, 0)
		includeH(0)
	case 2:
		includeH(0)
	case 3:
		includeCurrent(1, 0)
		includeH(0)
	case 4:
		includeCurrent(0, 0)
		includeV(0)
	case 5:
		includeH(0)
		includeV(0)
	case 6:
		includeH(0)
		includeHV()
	case 7:
		includeH(0)
		includeV(1)
	case 8:
		includeV(0)
	case 9:
		includeV(0)
		includeHV()
	case 10:
		includeHV()
	case 11:
		includeV(1)
		includeHV()
	case 12:
		includeCurrent(0, 1)
		includeV(0)
	case 13:
		includeH(1)
		includeV(0)
	case 14:
		includeH(1)
		includeHV()
	case 15:
		includeH(1)
		includeV(1)
	}
	return minX, minY, maxX, maxY
}
