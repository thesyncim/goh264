// SPDX-License-Identifier: LGPL-2.1-or-later
//
// Source-shaped 8-bit H.264 inverse transform kernels from FFmpeg n8.0.1
// libavcodec/h264idct_template.c. These are the reference Go kernels used
// before SIMD or high-bit-depth specialization.

package h264

func h264IDCTAdd(dst []uint8, block []int32, stride int) error {
	if err := checkTransformAddArgs(dst, block, 16, stride, 4); err != nil {
		return err
	}
	block[0] = dctcoef8(int(block[0]) + 1<<5)

	for i := 0; i < 4; i++ {
		z0 := int(dctcoef8Value(block[i+4*0])) + int(dctcoef8Value(block[i+4*2]))
		z1 := int(dctcoef8Value(block[i+4*0])) - int(dctcoef8Value(block[i+4*2]))
		z2 := (int(dctcoef8Value(block[i+4*1])) >> 1) - int(dctcoef8Value(block[i+4*3]))
		z3 := int(dctcoef8Value(block[i+4*1])) + (int(dctcoef8Value(block[i+4*3])) >> 1)

		block[i+4*0] = dctcoef8(z0 + z3)
		block[i+4*1] = dctcoef8(z1 + z2)
		block[i+4*2] = dctcoef8(z1 - z2)
		block[i+4*3] = dctcoef8(z0 - z3)
	}

	for i := 0; i < 4; i++ {
		z0 := int(dctcoef8Value(block[0+4*i])) + int(dctcoef8Value(block[2+4*i]))
		z1 := int(dctcoef8Value(block[0+4*i])) - int(dctcoef8Value(block[2+4*i]))
		z2 := (int(dctcoef8Value(block[1+4*i])) >> 1) - int(dctcoef8Value(block[3+4*i]))
		z3 := int(dctcoef8Value(block[1+4*i])) + (int(dctcoef8Value(block[3+4*i])) >> 1)

		dst[i+0*stride] = clipUint8(int(dst[i+0*stride]) + ((z0 + z3) >> 6))
		dst[i+1*stride] = clipUint8(int(dst[i+1*stride]) + ((z1 + z2) >> 6))
		dst[i+2*stride] = clipUint8(int(dst[i+2*stride]) + ((z1 - z2) >> 6))
		dst[i+3*stride] = clipUint8(int(dst[i+3*stride]) + ((z0 - z3) >> 6))
	}

	clearInt32(block[:16])
	return nil
}

func h264IDCT8Add(dst []uint8, block []int32, stride int) error {
	if err := checkTransformAddArgs(dst, block, 64, stride, 8); err != nil {
		return err
	}
	block[0] = dctcoef8(int(block[0]) + 32)

	for i := 0; i < 8; i++ {
		a0 := int(dctcoef8Value(block[i+0*8])) + int(dctcoef8Value(block[i+4*8]))
		a2 := int(dctcoef8Value(block[i+0*8])) - int(dctcoef8Value(block[i+4*8]))
		a4 := (int(dctcoef8Value(block[i+2*8])) >> 1) - int(dctcoef8Value(block[i+6*8]))
		a6 := (int(dctcoef8Value(block[i+6*8])) >> 1) + int(dctcoef8Value(block[i+2*8]))

		b0 := a0 + a6
		b2 := a2 + a4
		b4 := a2 - a4
		b6 := a0 - a6

		a1 := -int(dctcoef8Value(block[i+3*8])) + int(dctcoef8Value(block[i+5*8])) - int(dctcoef8Value(block[i+7*8])) - (int(dctcoef8Value(block[i+7*8])) >> 1)
		a3 := int(dctcoef8Value(block[i+1*8])) + int(dctcoef8Value(block[i+7*8])) - int(dctcoef8Value(block[i+3*8])) - (int(dctcoef8Value(block[i+3*8])) >> 1)
		a5 := -int(dctcoef8Value(block[i+1*8])) + int(dctcoef8Value(block[i+7*8])) + int(dctcoef8Value(block[i+5*8])) + (int(dctcoef8Value(block[i+5*8])) >> 1)
		a7 := int(dctcoef8Value(block[i+3*8])) + int(dctcoef8Value(block[i+5*8])) + int(dctcoef8Value(block[i+1*8])) + (int(dctcoef8Value(block[i+1*8])) >> 1)

		b1 := (a7 >> 2) + a1
		b3 := a3 + (a5 >> 2)
		b5 := (a3 >> 2) - a5
		b7 := a7 - (a1 >> 2)

		block[i+0*8] = dctcoef8(b0 + b7)
		block[i+7*8] = dctcoef8(b0 - b7)
		block[i+1*8] = dctcoef8(b2 + b5)
		block[i+6*8] = dctcoef8(b2 - b5)
		block[i+2*8] = dctcoef8(b4 + b3)
		block[i+5*8] = dctcoef8(b4 - b3)
		block[i+3*8] = dctcoef8(b6 + b1)
		block[i+4*8] = dctcoef8(b6 - b1)
	}
	for i := 0; i < 8; i++ {
		a0 := int(dctcoef8Value(block[0+i*8])) + int(dctcoef8Value(block[4+i*8]))
		a2 := int(dctcoef8Value(block[0+i*8])) - int(dctcoef8Value(block[4+i*8]))
		a4 := (int(dctcoef8Value(block[2+i*8])) >> 1) - int(dctcoef8Value(block[6+i*8]))
		a6 := (int(dctcoef8Value(block[6+i*8])) >> 1) + int(dctcoef8Value(block[2+i*8]))

		b0 := a0 + a6
		b2 := a2 + a4
		b4 := a2 - a4
		b6 := a0 - a6

		a1 := -int(dctcoef8Value(block[3+i*8])) + int(dctcoef8Value(block[5+i*8])) - int(dctcoef8Value(block[7+i*8])) - (int(dctcoef8Value(block[7+i*8])) >> 1)
		a3 := int(dctcoef8Value(block[1+i*8])) + int(dctcoef8Value(block[7+i*8])) - int(dctcoef8Value(block[3+i*8])) - (int(dctcoef8Value(block[3+i*8])) >> 1)
		a5 := -int(dctcoef8Value(block[1+i*8])) + int(dctcoef8Value(block[7+i*8])) + int(dctcoef8Value(block[5+i*8])) + (int(dctcoef8Value(block[5+i*8])) >> 1)
		a7 := int(dctcoef8Value(block[3+i*8])) + int(dctcoef8Value(block[5+i*8])) + int(dctcoef8Value(block[1+i*8])) + (int(dctcoef8Value(block[1+i*8])) >> 1)

		b1 := (a7 >> 2) + a1
		b3 := a3 + (a5 >> 2)
		b5 := (a3 >> 2) - a5
		b7 := a7 - (a1 >> 2)

		dst[i+0*stride] = clipUint8(int(dst[i+0*stride]) + ((b0 + b7) >> 6))
		dst[i+1*stride] = clipUint8(int(dst[i+1*stride]) + ((b2 + b5) >> 6))
		dst[i+2*stride] = clipUint8(int(dst[i+2*stride]) + ((b4 + b3) >> 6))
		dst[i+3*stride] = clipUint8(int(dst[i+3*stride]) + ((b6 + b1) >> 6))
		dst[i+4*stride] = clipUint8(int(dst[i+4*stride]) + ((b6 - b1) >> 6))
		dst[i+5*stride] = clipUint8(int(dst[i+5*stride]) + ((b4 - b3) >> 6))
		dst[i+6*stride] = clipUint8(int(dst[i+6*stride]) + ((b2 - b5) >> 6))
		dst[i+7*stride] = clipUint8(int(dst[i+7*stride]) + ((b0 - b7) >> 6))
	}

	clearInt32(block[:64])
	return nil
}

func h264IDCTDCAdd(dst []uint8, block []int32, stride int) error {
	if err := checkTransformAddArgs(dst, block, 1, stride, 4); err != nil {
		return err
	}
	dc := (int(dctcoef8Value(block[0])) + 32) >> 6
	block[0] = 0
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			dst[y*stride+x] = clipUint8(int(dst[y*stride+x]) + dc)
		}
	}
	return nil
}

func h264IDCT8DCAdd(dst []uint8, block []int32, stride int) error {
	if err := checkTransformAddArgs(dst, block, 1, stride, 8); err != nil {
		return err
	}
	dc := (int(dctcoef8Value(block[0])) + 32) >> 6
	block[0] = 0
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			dst[y*stride+x] = clipUint8(int(dst[y*stride+x]) + dc)
		}
	}
	return nil
}

func h264IDCTAdd16(dst []uint8, blockOffset *[48]int, block []int32, stride int, nnzc *[h264NonZeroCountCacheSize]uint8) error {
	if blockOffset == nil || nnzc == nil || len(block) < 16*16 {
		return ErrInvalidData
	}
	for i := 0; i < 16; i++ {
		nnz := nnzc[h264Scan8[i]]
		if nnz == 0 {
			continue
		}
		dstBlock, err := transformBlockDestination(dst, blockOffset[i], stride, 4)
		if err != nil {
			return err
		}
		coef := block[i*16 : i*16+16]
		if nnz == 1 && dctcoef8Value(coef[0]) != 0 {
			if err := h264IDCTDCAdd(dstBlock, coef, stride); err != nil {
				return err
			}
		} else if err := h264IDCTAdd(dstBlock, coef, stride); err != nil {
			return err
		}
	}
	return nil
}

func h264IDCTAdd16Intra(dst []uint8, blockOffset *[48]int, block []int32, stride int, nnzc *[h264NonZeroCountCacheSize]uint8) error {
	if blockOffset == nil || nnzc == nil || len(block) < 16*16 {
		return ErrInvalidData
	}
	for i := 0; i < 16; i++ {
		coef := block[i*16 : i*16+16]
		if nnzc[h264Scan8[i]] != 0 {
			dstBlock, err := transformBlockDestination(dst, blockOffset[i], stride, 4)
			if err != nil {
				return err
			}
			if err := h264IDCTAdd(dstBlock, coef, stride); err != nil {
				return err
			}
		} else if dctcoef8Value(coef[0]) != 0 {
			dstBlock, err := transformBlockDestination(dst, blockOffset[i], stride, 4)
			if err != nil {
				return err
			}
			if err := h264IDCTDCAdd(dstBlock, coef, stride); err != nil {
				return err
			}
		}
	}
	return nil
}

func h264IDCT8Add4(dst []uint8, blockOffset *[48]int, block []int32, stride int, nnzc *[h264NonZeroCountCacheSize]uint8) error {
	if blockOffset == nil || nnzc == nil || len(block) < 16*16 {
		return ErrInvalidData
	}
	for i := 0; i < 16; i += 4 {
		nnz := nnzc[h264Scan8[i]]
		if nnz == 0 {
			continue
		}
		dstBlock, err := transformBlockDestination(dst, blockOffset[i], stride, 8)
		if err != nil {
			return err
		}
		coef := block[i*16 : i*16+64]
		if nnz == 1 && dctcoef8Value(coef[0]) != 0 {
			if err := h264IDCT8DCAdd(dstBlock, coef, stride); err != nil {
				return err
			}
		} else if err := h264IDCT8Add(dstBlock, coef, stride); err != nil {
			return err
		}
	}
	return nil
}

func h264IDCTAdd8(dest *[2][]uint8, blockOffset *[48]int, block []int32, stride int, nnzc *[h264NonZeroCountCacheSize]uint8) error {
	if dest == nil || blockOffset == nil || nnzc == nil || len(block) < 48*16 {
		return ErrInvalidData
	}
	for j := 1; j < 3; j++ {
		for i := j * 16; i < j*16+4; i++ {
			if err := h264IDCTAddChromaBlock(dest[j-1], blockOffset[i], block[i*16:i*16+16], stride, nnzc[h264Scan8[i]]); err != nil {
				return err
			}
		}
	}
	return nil
}

func h264IDCTAdd8_422(dest *[2][]uint8, blockOffset *[48]int, block []int32, stride int, nnzc *[h264NonZeroCountCacheSize]uint8) error {
	if dest == nil || blockOffset == nil || nnzc == nil || len(block) < 48*16 {
		return ErrInvalidData
	}
	for j := 1; j < 3; j++ {
		for i := j * 16; i < j*16+4; i++ {
			if err := h264IDCTAddChromaBlock(dest[j-1], blockOffset[i], block[i*16:i*16+16], stride, nnzc[h264Scan8[i]]); err != nil {
				return err
			}
		}
	}
	for j := 1; j < 3; j++ {
		for i := j*16 + 4; i < j*16+8; i++ {
			if err := h264IDCTAddChromaBlock(dest[j-1], blockOffset[i+4], block[i*16:i*16+16], stride, nnzc[h264Scan8[i+4]]); err != nil {
				return err
			}
		}
	}
	return nil
}

func h264LumaDCDequantIDCT(output []int32, input *[16]int32, qmul int) error {
	if input == nil || len(output) < 16*16 {
		return ErrInvalidData
	}
	var temp [16]int
	xOffset := [4]int{0, 2 * 16, 8 * 16, 10 * 16}

	for i := 0; i < 4; i++ {
		z0 := int(dctcoef8Value(input[4*i+0])) + int(dctcoef8Value(input[4*i+1]))
		z1 := int(dctcoef8Value(input[4*i+0])) - int(dctcoef8Value(input[4*i+1]))
		z2 := int(dctcoef8Value(input[4*i+2])) - int(dctcoef8Value(input[4*i+3]))
		z3 := int(dctcoef8Value(input[4*i+2])) + int(dctcoef8Value(input[4*i+3]))

		temp[4*i+0] = z0 + z3
		temp[4*i+1] = z0 - z3
		temp[4*i+2] = z1 - z2
		temp[4*i+3] = z1 + z2
	}

	for i := 0; i < 4; i++ {
		offset := xOffset[i]
		z0 := temp[4*0+i] + temp[4*2+i]
		z1 := temp[4*0+i] - temp[4*2+i]
		z2 := temp[4*1+i] - temp[4*3+i]
		z3 := temp[4*1+i] + temp[4*3+i]

		output[16*0+offset] = dctcoef8(((z0+z3)*qmul + 128) >> 8)
		output[16*1+offset] = dctcoef8(((z1+z2)*qmul + 128) >> 8)
		output[16*4+offset] = dctcoef8(((z1-z2)*qmul + 128) >> 8)
		output[16*5+offset] = dctcoef8(((z0-z3)*qmul + 128) >> 8)
	}
	return nil
}

func h264ChromaDCDequantIDCT(block []int32, qmul int) error {
	if len(block) < 49 {
		return ErrInvalidData
	}
	const stride = 16 * 2
	const xStride = 16
	a := int(dctcoef8Value(block[stride*0+xStride*0]))
	b := int(dctcoef8Value(block[stride*0+xStride*1]))
	c := int(dctcoef8Value(block[stride*1+xStride*0]))
	d := int(dctcoef8Value(block[stride*1+xStride*1]))

	e := a - b
	a = a + b
	b = c - d
	c = c + d

	block[stride*0+xStride*0] = dctcoef8(((a + c) * qmul) >> 7)
	block[stride*0+xStride*1] = dctcoef8(((e + b) * qmul) >> 7)
	block[stride*1+xStride*0] = dctcoef8(((a - c) * qmul) >> 7)
	block[stride*1+xStride*1] = dctcoef8(((e - b) * qmul) >> 7)
	return nil
}

func h264Chroma422DCDequantIDCT(block []int32, qmul int) error {
	if len(block) < 113 {
		return ErrInvalidData
	}
	const stride = 16 * 2
	const xStride = 16
	var temp [8]int
	xOffset := [2]int{0, 16}

	for i := 0; i < 4; i++ {
		temp[2*i+0] = int(dctcoef8Value(block[stride*i+xStride*0])) + int(dctcoef8Value(block[stride*i+xStride*1]))
		temp[2*i+1] = int(dctcoef8Value(block[stride*i+xStride*0])) - int(dctcoef8Value(block[stride*i+xStride*1]))
	}

	for i := 0; i < 2; i++ {
		offset := xOffset[i]
		z0 := temp[2*0+i] + temp[2*2+i]
		z1 := temp[2*0+i] - temp[2*2+i]
		z2 := temp[2*1+i] - temp[2*3+i]
		z3 := temp[2*1+i] + temp[2*3+i]

		block[stride*0+offset] = dctcoef8(((z0+z3)*qmul + 128) >> 8)
		block[stride*1+offset] = dctcoef8(((z1+z2)*qmul + 128) >> 8)
		block[stride*2+offset] = dctcoef8(((z1-z2)*qmul + 128) >> 8)
		block[stride*3+offset] = dctcoef8(((z0-z3)*qmul + 128) >> 8)
	}
	return nil
}

func h264FrameBlockOffsets(lumaStride int, chromaStride int, pixelShift int) ([48]int, error) {
	var offset [48]int
	if lumaStride <= 0 || chromaStride <= 0 || pixelShift < 0 || pixelShift > 1 {
		return offset, ErrInvalidData
	}
	base := int(h264Scan8[0])
	for i := 0; i < 16; i++ {
		delta := int(h264Scan8[i]) - base
		offset[i] = (4 * ((delta) & 7) << pixelShift) + 4*lumaStride*(delta>>3)
		offset[16+i] = (4 * ((delta) & 7) << pixelShift) + 4*chromaStride*(delta>>3)
		offset[32+i] = offset[16+i]
	}
	return offset, nil
}

func h264IDCTAddChromaBlock(dst []uint8, offset int, block []int32, stride int, nnz uint8) error {
	if nnz == 0 && dctcoef8Value(block[0]) == 0 {
		return nil
	}
	dstBlock, err := transformBlockDestination(dst, offset, stride, 4)
	if err != nil {
		return err
	}
	if nnz != 0 {
		return h264IDCTAdd(dstBlock, block, stride)
	}
	return h264IDCTDCAdd(dstBlock, block, stride)
}

func transformBlockDestination(dst []uint8, offset int, stride int, size int) ([]uint8, error) {
	if offset < 0 {
		return nil, ErrInvalidData
	}
	needed := offset + (size-1)*stride + size
	if stride <= 0 || size <= 0 || len(dst) < needed {
		return nil, ErrInvalidData
	}
	return dst[offset:], nil
}

func checkTransformAddArgs(dst []uint8, block []int32, blockLen int, stride int, size int) error {
	if len(block) < blockLen {
		return ErrInvalidData
	}
	_, err := transformBlockDestination(dst, 0, stride, size)
	return err
}

func dctcoef8(v int) int32 {
	return int32(int16(v))
}

func dctcoef8Value(v int32) int16 {
	return int16(v)
}

func clipUint8(v int) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
}

func clearInt32(v []int32) {
	for i := range v {
		v[i] = 0
	}
}
