// SPDX-License-Identifier: LGPL-2.1-or-later
//
// Source-shaped H.264 DSP reconstruction helpers from FFmpeg n8.0.1
// libavcodec/h264addpx_template.c and h264dsp_template.c.

package h264

func h264AddPixels4Clear(dst []uint8, block []int32, stride int) error {
	if err := checkTransformAddArgs(dst, block, 16, stride, 4); err != nil {
		return err
	}
	for y := 0; y < 4; y++ {
		row := y * stride
		src := y * 4
		dst[row+0] += uint8(dctcoef8Value(block[src+0]))
		dst[row+1] += uint8(dctcoef8Value(block[src+1]))
		dst[row+2] += uint8(dctcoef8Value(block[src+2]))
		dst[row+3] += uint8(dctcoef8Value(block[src+3]))
	}
	clearInt32(block[:16])
	return nil
}

func h264AddPixels8Clear(dst []uint8, block []int32, stride int) error {
	if err := checkTransformAddArgs(dst, block, 64, stride, 8); err != nil {
		return err
	}
	for y := 0; y < 8; y++ {
		row := y * stride
		src := y * 8
		dst[row+0] += uint8(dctcoef8Value(block[src+0]))
		dst[row+1] += uint8(dctcoef8Value(block[src+1]))
		dst[row+2] += uint8(dctcoef8Value(block[src+2]))
		dst[row+3] += uint8(dctcoef8Value(block[src+3]))
		dst[row+4] += uint8(dctcoef8Value(block[src+4]))
		dst[row+5] += uint8(dctcoef8Value(block[src+5]))
		dst[row+6] += uint8(dctcoef8Value(block[src+6]))
		dst[row+7] += uint8(dctcoef8Value(block[src+7]))
	}
	clearInt32(block[:64])
	return nil
}

func h264AddPixels4ClearHigh(dst []uint16, block []int32, stride int) error {
	if err := checkTransformAddArgsHigh(dst, block, 16, stride, 4); err != nil {
		return err
	}
	for y := 0; y < 4; y++ {
		row := y * stride
		src := y * 4
		dst[row+0] += uint16(uint32(block[src+0]))
		dst[row+1] += uint16(uint32(block[src+1]))
		dst[row+2] += uint16(uint32(block[src+2]))
		dst[row+3] += uint16(uint32(block[src+3]))
	}
	clearInt32(block[:16])
	return nil
}

func h264AddPixels8ClearHigh(dst []uint16, block []int32, stride int) error {
	if err := checkTransformAddArgsHigh(dst, block, 64, stride, 8); err != nil {
		return err
	}
	for y := 0; y < 8; y++ {
		row := y * stride
		src := y * 8
		dst[row+0] += uint16(uint32(block[src+0]))
		dst[row+1] += uint16(uint32(block[src+1]))
		dst[row+2] += uint16(uint32(block[src+2]))
		dst[row+3] += uint16(uint32(block[src+3]))
		dst[row+4] += uint16(uint32(block[src+4]))
		dst[row+5] += uint16(uint32(block[src+5]))
		dst[row+6] += uint16(uint32(block[src+6]))
		dst[row+7] += uint16(uint32(block[src+7]))
	}
	clearInt32(block[:64])
	return nil
}

func h264WeightPixels(dst []uint8, stride int, height int, log2Denom int, weight int, offset int, width int) error {
	if err := checkWeightedPixelsArgs(dst, nil, stride, height, width, log2Denom); err != nil {
		return err
	}
	scaledOffset := int(int32(uint32(offset) << uint(log2Denom)))
	if log2Denom != 0 {
		scaledOffset += 1 << (log2Denom - 1)
	}

	for y := 0; y < height; y++ {
		row := y * stride
		for x := 0; x < width; x++ {
			dst[row+x] = clipUint8((int(dst[row+x])*weight + scaledOffset) >> uint(log2Denom))
		}
	}
	return nil
}

func h264BiweightPixels(dst []uint8, src []uint8, stride int, height int, log2Denom int, weightd int, weights int, offset int, width int) error {
	if err := checkWeightedPixelsArgs(dst, src, stride, height, width, log2Denom); err != nil {
		return err
	}
	offset = int(int32(uint32(offset)))
	scaledOffset := int(int32(uint32((offset+1)|1) << uint(log2Denom)))

	for y := 0; y < height; y++ {
		row := y * stride
		for x := 0; x < width; x++ {
			dst[row+x] = clipUint8((int(src[row+x])*weights + int(dst[row+x])*weightd + scaledOffset) >> uint(log2Denom+1))
		}
	}
	return nil
}

func h264WeightPixelsHigh(dst []uint16, stride int, height int, log2Denom int, weight int, offset int, width int, bitDepth int) error {
	if err := checkWeightedPixelsHighArgs(dst, nil, stride, height, width, log2Denom, bitDepth); err != nil {
		return err
	}
	shift := bitDepth - 8
	scaledOffset := int(int32(uint32(offset) << uint(log2Denom+shift)))
	if log2Denom != 0 {
		scaledOffset += 1 << (log2Denom - 1)
	}

	for y := 0; y < height; y++ {
		row := y * stride
		for x := 0; x < width; x++ {
			dst[row+x] = clipUintBitDepth((int(dst[row+x])*weight+scaledOffset)>>uint(log2Denom), bitDepth)
		}
	}
	return nil
}

func h264BiweightPixelsHigh(dst []uint16, src []uint16, stride int, height int, log2Denom int, weightd int, weights int, offset int, width int, bitDepth int) error {
	if err := checkWeightedPixelsHighArgs(dst, src, stride, height, width, log2Denom, bitDepth); err != nil {
		return err
	}
	shift := bitDepth - 8
	offset = int(int32(uint32(offset) << uint(shift)))
	scaledOffset := int(int32(uint32((offset+1)|1) << uint(log2Denom)))

	for y := 0; y < height; y++ {
		row := y * stride
		for x := 0; x < width; x++ {
			dst[row+x] = clipUintBitDepth((int(src[row+x])*weights+int(dst[row+x])*weightd+scaledOffset)>>uint(log2Denom+1), bitDepth)
		}
	}
	return nil
}

func h264VLoopFilterLuma(pix []uint8, offset int, stride int, alpha int, beta int, tc0 *[4]int8) error {
	return h264LoopFilterLuma(pix, offset, stride, 1, 4, alpha, beta, tc0)
}

func h264HLoopFilterLuma(pix []uint8, offset int, stride int, alpha int, beta int, tc0 *[4]int8) error {
	return h264LoopFilterLuma(pix, offset, 1, stride, 4, alpha, beta, tc0)
}

func h264HLoopFilterLumaMBAFF(pix []uint8, offset int, stride int, alpha int, beta int, tc0 *[4]int8) error {
	return h264LoopFilterLuma(pix, offset, 1, stride, 2, alpha, beta, tc0)
}

func h264VLoopFilterLumaIntra(pix []uint8, offset int, stride int, alpha int, beta int) error {
	return h264LoopFilterLumaIntra(pix, offset, stride, 1, 4, alpha, beta)
}

func h264HLoopFilterLumaIntra(pix []uint8, offset int, stride int, alpha int, beta int) error {
	return h264LoopFilterLumaIntra(pix, offset, 1, stride, 4, alpha, beta)
}

func h264HLoopFilterLumaMBAFFIntra(pix []uint8, offset int, stride int, alpha int, beta int) error {
	return h264LoopFilterLumaIntra(pix, offset, 1, stride, 2, alpha, beta)
}

func h264VLoopFilterChroma(pix []uint8, offset int, stride int, alpha int, beta int, tc0 *[4]int8) error {
	return h264LoopFilterChroma(pix, offset, stride, 1, 2, alpha, beta, tc0)
}

func h264HLoopFilterChroma(pix []uint8, offset int, stride int, alpha int, beta int, tc0 *[4]int8) error {
	return h264LoopFilterChroma(pix, offset, 1, stride, 2, alpha, beta, tc0)
}

func h264HLoopFilterChromaMBAFF(pix []uint8, offset int, stride int, alpha int, beta int, tc0 *[4]int8) error {
	return h264LoopFilterChroma(pix, offset, 1, stride, 1, alpha, beta, tc0)
}

func h264HLoopFilterChroma422(pix []uint8, offset int, stride int, alpha int, beta int, tc0 *[4]int8) error {
	return h264LoopFilterChroma(pix, offset, 1, stride, 4, alpha, beta, tc0)
}

func h264HLoopFilterChroma422MBAFF(pix []uint8, offset int, stride int, alpha int, beta int, tc0 *[4]int8) error {
	return h264LoopFilterChroma(pix, offset, 1, stride, 2, alpha, beta, tc0)
}

func h264VLoopFilterChromaIntra(pix []uint8, offset int, stride int, alpha int, beta int) error {
	return h264LoopFilterChromaIntra(pix, offset, stride, 1, 2, alpha, beta)
}

func h264HLoopFilterChromaIntra(pix []uint8, offset int, stride int, alpha int, beta int) error {
	return h264LoopFilterChromaIntra(pix, offset, 1, stride, 2, alpha, beta)
}

func h264HLoopFilterChromaMBAFFIntra(pix []uint8, offset int, stride int, alpha int, beta int) error {
	return h264LoopFilterChromaIntra(pix, offset, 1, stride, 1, alpha, beta)
}

func h264HLoopFilterChroma422Intra(pix []uint8, offset int, stride int, alpha int, beta int) error {
	return h264LoopFilterChromaIntra(pix, offset, 1, stride, 4, alpha, beta)
}

func h264HLoopFilterChroma422MBAFFIntra(pix []uint8, offset int, stride int, alpha int, beta int) error {
	return h264LoopFilterChromaIntra(pix, offset, 1, stride, 2, alpha, beta)
}

func h264VLoopFilterLumaHigh(pix []uint16, offset int, stride int, alpha int, beta int, tc0 *[4]int8, bitDepth int) error {
	return h264LoopFilterLumaHigh(pix, offset, stride, 1, 4, alpha, beta, tc0, bitDepth)
}

func h264HLoopFilterLumaHigh(pix []uint16, offset int, stride int, alpha int, beta int, tc0 *[4]int8, bitDepth int) error {
	return h264LoopFilterLumaHigh(pix, offset, 1, stride, 4, alpha, beta, tc0, bitDepth)
}

func h264HLoopFilterLumaMBAFFHigh(pix []uint16, offset int, stride int, alpha int, beta int, tc0 *[4]int8, bitDepth int) error {
	return h264LoopFilterLumaHigh(pix, offset, 1, stride, 2, alpha, beta, tc0, bitDepth)
}

func h264VLoopFilterLumaIntraHigh(pix []uint16, offset int, stride int, alpha int, beta int, bitDepth int) error {
	return h264LoopFilterLumaIntraHigh(pix, offset, stride, 1, 4, alpha, beta, bitDepth)
}

func h264HLoopFilterLumaIntraHigh(pix []uint16, offset int, stride int, alpha int, beta int, bitDepth int) error {
	return h264LoopFilterLumaIntraHigh(pix, offset, 1, stride, 4, alpha, beta, bitDepth)
}

func h264HLoopFilterLumaMBAFFIntraHigh(pix []uint16, offset int, stride int, alpha int, beta int, bitDepth int) error {
	return h264LoopFilterLumaIntraHigh(pix, offset, 1, stride, 2, alpha, beta, bitDepth)
}

func h264VLoopFilterChromaHigh(pix []uint16, offset int, stride int, alpha int, beta int, tc0 *[4]int8, bitDepth int) error {
	return h264LoopFilterChromaHigh(pix, offset, stride, 1, 2, alpha, beta, tc0, bitDepth)
}

func h264HLoopFilterChromaHigh(pix []uint16, offset int, stride int, alpha int, beta int, tc0 *[4]int8, bitDepth int) error {
	return h264LoopFilterChromaHigh(pix, offset, 1, stride, 2, alpha, beta, tc0, bitDepth)
}

func h264HLoopFilterChromaMBAFFHigh(pix []uint16, offset int, stride int, alpha int, beta int, tc0 *[4]int8, bitDepth int) error {
	return h264LoopFilterChromaHigh(pix, offset, 1, stride, 1, alpha, beta, tc0, bitDepth)
}

func h264HLoopFilterChroma422High(pix []uint16, offset int, stride int, alpha int, beta int, tc0 *[4]int8, bitDepth int) error {
	return h264LoopFilterChromaHigh(pix, offset, 1, stride, 4, alpha, beta, tc0, bitDepth)
}

func h264HLoopFilterChroma422MBAFFHigh(pix []uint16, offset int, stride int, alpha int, beta int, tc0 *[4]int8, bitDepth int) error {
	return h264LoopFilterChromaHigh(pix, offset, 1, stride, 2, alpha, beta, tc0, bitDepth)
}

func h264VLoopFilterChromaIntraHigh(pix []uint16, offset int, stride int, alpha int, beta int, bitDepth int) error {
	return h264LoopFilterChromaIntraHigh(pix, offset, stride, 1, 2, alpha, beta, bitDepth)
}

func h264HLoopFilterChromaIntraHigh(pix []uint16, offset int, stride int, alpha int, beta int, bitDepth int) error {
	return h264LoopFilterChromaIntraHigh(pix, offset, 1, stride, 2, alpha, beta, bitDepth)
}

func h264HLoopFilterChromaMBAFFIntraHigh(pix []uint16, offset int, stride int, alpha int, beta int, bitDepth int) error {
	return h264LoopFilterChromaIntraHigh(pix, offset, 1, stride, 1, alpha, beta, bitDepth)
}

func h264HLoopFilterChroma422IntraHigh(pix []uint16, offset int, stride int, alpha int, beta int, bitDepth int) error {
	return h264LoopFilterChromaIntraHigh(pix, offset, 1, stride, 4, alpha, beta, bitDepth)
}

func h264HLoopFilterChroma422MBAFFIntraHigh(pix []uint16, offset int, stride int, alpha int, beta int, bitDepth int) error {
	return h264LoopFilterChromaIntraHigh(pix, offset, 1, stride, 2, alpha, beta, bitDepth)
}

func h264LoopFilterLuma(pix []uint8, offset int, xstride int, ystride int, innerIters int, alpha int, beta int, tc0 *[4]int8) error {
	if tc0 == nil {
		return ErrInvalidData
	}
	if err := checkLoopFilterArgs(pix, offset, xstride, ystride, innerIters, 4, 3, 2); err != nil {
		return err
	}
	pos := offset
	for i := 0; i < 4; i++ {
		tcOrig := int(tc0[i])
		if tcOrig < 0 {
			pos += innerIters * ystride
			continue
		}
		for d := 0; d < innerIters; d++ {
			p0 := int(pix[pos-1*xstride])
			p1 := int(pix[pos-2*xstride])
			p2 := int(pix[pos-3*xstride])
			q0 := int(pix[pos])
			q1 := int(pix[pos+1*xstride])
			q2 := int(pix[pos+2*xstride])

			if absInt(p0-q0) < alpha &&
				absInt(p1-p0) < beta &&
				absInt(q1-q0) < beta {
				tc := tcOrig

				if absInt(p2-p0) < beta {
					if tcOrig != 0 {
						pix[pos-2*xstride] = uint8(p1 + clipInt(((p2+((p0+q0+1)>>1))>>1)-p1, -tcOrig, tcOrig))
					}
					tc++
				}
				if absInt(q2-q0) < beta {
					if tcOrig != 0 {
						pix[pos+xstride] = uint8(q1 + clipInt(((q2+((p0+q0+1)>>1))>>1)-q1, -tcOrig, tcOrig))
					}
					tc++
				}

				delta := clipInt((((q0-p0)*4)+(p1-q1)+4)>>3, -tc, tc)
				pix[pos-xstride] = clipUint8(p0 + delta)
				pix[pos] = clipUint8(q0 - delta)
			}
			pos += ystride
		}
	}
	return nil
}

func h264LoopFilterLumaHigh(pix []uint16, offset int, xstride int, ystride int, innerIters int, alpha int, beta int, tc0 *[4]int8, bitDepth int) error {
	if tc0 == nil {
		return ErrInvalidData
	}
	if err := checkLoopFilterHighArgs(pix, offset, xstride, ystride, innerIters, 4, 3, 2, bitDepth); err != nil {
		return err
	}
	shift := bitDepth - 8
	alpha <<= uint(shift)
	beta <<= uint(shift)
	pos := offset
	for i := 0; i < 4; i++ {
		tcOrig := int(tc0[i]) * (1 << uint(shift))
		if tcOrig < 0 {
			pos += innerIters * ystride
			continue
		}
		for d := 0; d < innerIters; d++ {
			p0 := int(pix[pos-1*xstride])
			p1 := int(pix[pos-2*xstride])
			p2 := int(pix[pos-3*xstride])
			q0 := int(pix[pos])
			q1 := int(pix[pos+1*xstride])
			q2 := int(pix[pos+2*xstride])

			if absInt(p0-q0) < alpha &&
				absInt(p1-p0) < beta &&
				absInt(q1-q0) < beta {
				tc := tcOrig

				if absInt(p2-p0) < beta {
					if tcOrig != 0 {
						pix[pos-2*xstride] = uint16(p1 + clipInt(((p2+((p0+q0+1)>>1))>>1)-p1, -tcOrig, tcOrig))
					}
					tc++
				}
				if absInt(q2-q0) < beta {
					if tcOrig != 0 {
						pix[pos+xstride] = uint16(q1 + clipInt(((q2+((p0+q0+1)>>1))>>1)-q1, -tcOrig, tcOrig))
					}
					tc++
				}

				delta := clipInt((((q0-p0)*4)+(p1-q1)+4)>>3, -tc, tc)
				pix[pos-xstride] = clipUintBitDepth(p0+delta, bitDepth)
				pix[pos] = clipUintBitDepth(q0-delta, bitDepth)
			}
			pos += ystride
		}
	}
	return nil
}

func h264LoopFilterLumaIntra(pix []uint8, offset int, xstride int, ystride int, innerIters int, alpha int, beta int) error {
	if err := checkLoopFilterArgs(pix, offset, xstride, ystride, innerIters, 4, 4, 3); err != nil {
		return err
	}
	pos := offset
	for d := 0; d < 4*innerIters; d++ {
		p2 := int(pix[pos-3*xstride])
		p1 := int(pix[pos-2*xstride])
		p0 := int(pix[pos-1*xstride])
		q0 := int(pix[pos])
		q1 := int(pix[pos+1*xstride])
		q2 := int(pix[pos+2*xstride])

		if absInt(p0-q0) < alpha &&
			absInt(p1-p0) < beta &&
			absInt(q1-q0) < beta {
			if absInt(p0-q0) < ((alpha >> 2) + 2) {
				if absInt(p2-p0) < beta {
					p3 := int(pix[pos-4*xstride])
					pix[pos-1*xstride] = uint8((p2 + 2*p1 + 2*p0 + 2*q0 + q1 + 4) >> 3)
					pix[pos-2*xstride] = uint8((p2 + p1 + p0 + q0 + 2) >> 2)
					pix[pos-3*xstride] = uint8((2*p3 + 3*p2 + p1 + p0 + q0 + 4) >> 3)
				} else {
					pix[pos-1*xstride] = uint8((2*p1 + p0 + q1 + 2) >> 2)
				}
				if absInt(q2-q0) < beta {
					q3 := int(pix[pos+3*xstride])
					pix[pos] = uint8((p1 + 2*p0 + 2*q0 + 2*q1 + q2 + 4) >> 3)
					pix[pos+1*xstride] = uint8((p0 + q0 + q1 + q2 + 2) >> 2)
					pix[pos+2*xstride] = uint8((2*q3 + 3*q2 + q1 + q0 + p0 + 4) >> 3)
				} else {
					pix[pos] = uint8((2*q1 + q0 + p1 + 2) >> 2)
				}
			} else {
				pix[pos-1*xstride] = uint8((2*p1 + p0 + q1 + 2) >> 2)
				pix[pos] = uint8((2*q1 + q0 + p1 + 2) >> 2)
			}
		}
		pos += ystride
	}
	return nil
}

func h264LoopFilterLumaIntraHigh(pix []uint16, offset int, xstride int, ystride int, innerIters int, alpha int, beta int, bitDepth int) error {
	if err := checkLoopFilterHighArgs(pix, offset, xstride, ystride, innerIters, 4, 4, 3, bitDepth); err != nil {
		return err
	}
	shift := bitDepth - 8
	alpha <<= uint(shift)
	beta <<= uint(shift)
	pos := offset
	for d := 0; d < 4*innerIters; d++ {
		p2 := int(pix[pos-3*xstride])
		p1 := int(pix[pos-2*xstride])
		p0 := int(pix[pos-1*xstride])
		q0 := int(pix[pos])
		q1 := int(pix[pos+1*xstride])
		q2 := int(pix[pos+2*xstride])

		if absInt(p0-q0) < alpha &&
			absInt(p1-p0) < beta &&
			absInt(q1-q0) < beta {
			if absInt(p0-q0) < ((alpha >> 2) + 2) {
				if absInt(p2-p0) < beta {
					p3 := int(pix[pos-4*xstride])
					pix[pos-1*xstride] = uint16((p2 + 2*p1 + 2*p0 + 2*q0 + q1 + 4) >> 3)
					pix[pos-2*xstride] = uint16((p2 + p1 + p0 + q0 + 2) >> 2)
					pix[pos-3*xstride] = uint16((2*p3 + 3*p2 + p1 + p0 + q0 + 4) >> 3)
				} else {
					pix[pos-1*xstride] = uint16((2*p1 + p0 + q1 + 2) >> 2)
				}
				if absInt(q2-q0) < beta {
					q3 := int(pix[pos+3*xstride])
					pix[pos] = uint16((p1 + 2*p0 + 2*q0 + 2*q1 + q2 + 4) >> 3)
					pix[pos+1*xstride] = uint16((p0 + q0 + q1 + q2 + 2) >> 2)
					pix[pos+2*xstride] = uint16((2*q3 + 3*q2 + q1 + q0 + p0 + 4) >> 3)
				} else {
					pix[pos] = uint16((2*q1 + q0 + p1 + 2) >> 2)
				}
			} else {
				pix[pos-1*xstride] = uint16((2*p1 + p0 + q1 + 2) >> 2)
				pix[pos] = uint16((2*q1 + q0 + p1 + 2) >> 2)
			}
		}
		pos += ystride
	}
	return nil
}

func h264LoopFilterChroma(pix []uint8, offset int, xstride int, ystride int, innerIters int, alpha int, beta int, tc0 *[4]int8) error {
	if tc0 == nil {
		return ErrInvalidData
	}
	if err := checkLoopFilterArgs(pix, offset, xstride, ystride, innerIters, 4, 2, 1); err != nil {
		return err
	}
	pos := offset
	for i := 0; i < 4; i++ {
		tc := int(tc0[i])
		if tc <= 0 {
			pos += innerIters * ystride
			continue
		}
		for d := 0; d < innerIters; d++ {
			p0 := int(pix[pos-1*xstride])
			p1 := int(pix[pos-2*xstride])
			q0 := int(pix[pos])
			q1 := int(pix[pos+1*xstride])

			if absInt(p0-q0) < alpha &&
				absInt(p1-p0) < beta &&
				absInt(q1-q0) < beta {
				delta := clipInt(((q0-p0)*4+(p1-q1)+4)>>3, -tc, tc)
				pix[pos-xstride] = clipUint8(p0 + delta)
				pix[pos] = clipUint8(q0 - delta)
			}
			pos += ystride
		}
	}
	return nil
}

func h264LoopFilterChromaHigh(pix []uint16, offset int, xstride int, ystride int, innerIters int, alpha int, beta int, tc0 *[4]int8, bitDepth int) error {
	if tc0 == nil {
		return ErrInvalidData
	}
	if err := checkLoopFilterHighArgs(pix, offset, xstride, ystride, innerIters, 4, 2, 1, bitDepth); err != nil {
		return err
	}
	shift := bitDepth - 8
	alpha <<= uint(shift)
	beta <<= uint(shift)
	pos := offset
	for i := 0; i < 4; i++ {
		tc := ((int(tc0[i]) - 1) << uint(shift)) + 1
		if tc <= 0 {
			pos += innerIters * ystride
			continue
		}
		for d := 0; d < innerIters; d++ {
			p0 := int(pix[pos-1*xstride])
			p1 := int(pix[pos-2*xstride])
			q0 := int(pix[pos])
			q1 := int(pix[pos+1*xstride])

			if absInt(p0-q0) < alpha &&
				absInt(p1-p0) < beta &&
				absInt(q1-q0) < beta {
				delta := clipInt(((q0-p0)*4+(p1-q1)+4)>>3, -tc, tc)
				pix[pos-xstride] = clipUintBitDepth(p0+delta, bitDepth)
				pix[pos] = clipUintBitDepth(q0-delta, bitDepth)
			}
			pos += ystride
		}
	}
	return nil
}

func h264LoopFilterChromaIntra(pix []uint8, offset int, xstride int, ystride int, innerIters int, alpha int, beta int) error {
	if err := checkLoopFilterArgs(pix, offset, xstride, ystride, innerIters, 4, 2, 1); err != nil {
		return err
	}
	pos := offset
	for d := 0; d < 4*innerIters; d++ {
		p0 := int(pix[pos-1*xstride])
		p1 := int(pix[pos-2*xstride])
		q0 := int(pix[pos])
		q1 := int(pix[pos+1*xstride])

		if absInt(p0-q0) < alpha &&
			absInt(p1-p0) < beta &&
			absInt(q1-q0) < beta {
			pix[pos-xstride] = uint8((2*p1 + p0 + q1 + 2) >> 2)
			pix[pos] = uint8((2*q1 + q0 + p1 + 2) >> 2)
		}
		pos += ystride
	}
	return nil
}

func h264LoopFilterChromaIntraHigh(pix []uint16, offset int, xstride int, ystride int, innerIters int, alpha int, beta int, bitDepth int) error {
	if err := checkLoopFilterHighArgs(pix, offset, xstride, ystride, innerIters, 4, 2, 1, bitDepth); err != nil {
		return err
	}
	shift := bitDepth - 8
	alpha <<= uint(shift)
	beta <<= uint(shift)
	pos := offset
	for d := 0; d < 4*innerIters; d++ {
		p0 := int(pix[pos-1*xstride])
		p1 := int(pix[pos-2*xstride])
		q0 := int(pix[pos])
		q1 := int(pix[pos+1*xstride])

		if absInt(p0-q0) < alpha &&
			absInt(p1-p0) < beta &&
			absInt(q1-q0) < beta {
			pix[pos-xstride] = uint16((2*p1 + p0 + q1 + 2) >> 2)
			pix[pos] = uint16((2*q1 + q0 + p1 + 2) >> 2)
		}
		pos += ystride
	}
	return nil
}

func checkWeightedPixelsArgs(dst []uint8, src []uint8, stride int, height int, width int, log2Denom int) error {
	if stride <= 0 || height < 0 || log2Denom < 0 {
		return ErrInvalidData
	}
	if width != 2 && width != 4 && width != 8 && width != 16 {
		return ErrInvalidData
	}
	if height == 0 {
		return nil
	}
	needed := (height-1)*stride + width
	if len(dst) < needed {
		return ErrInvalidData
	}
	if src != nil && len(src) < needed {
		return ErrInvalidData
	}
	return nil
}

func checkWeightedPixelsHighArgs(dst []uint16, src []uint16, stride int, height int, width int, log2Denom int, bitDepth int) error {
	if err := checkH264DSPHighBitDepth(bitDepth); err != nil {
		return err
	}
	if stride <= 0 || height < 0 || log2Denom < 0 {
		return ErrInvalidData
	}
	if width != 2 && width != 4 && width != 8 && width != 16 {
		return ErrInvalidData
	}
	if height == 0 {
		return nil
	}
	needed := (height-1)*stride + width
	if len(dst) < needed {
		return ErrInvalidData
	}
	if src != nil && len(src) < needed {
		return ErrInvalidData
	}
	return nil
}

func checkLoopFilterArgs(pix []uint8, offset int, xstride int, ystride int, innerIters int, groups int, before int, after int) error {
	if offset < 0 || xstride <= 0 || ystride <= 0 || innerIters <= 0 || groups <= 0 {
		return ErrInvalidData
	}
	minIndex := offset - before*xstride
	maxIndex := offset + (groups*innerIters-1)*ystride + after*xstride
	if minIndex < 0 || maxIndex >= len(pix) {
		return ErrInvalidData
	}
	return nil
}

func transformBlockDestinationHigh(dst []uint16, offset int, stride int, size int) ([]uint16, error) {
	if offset < 0 {
		return nil, ErrInvalidData
	}
	needed := offset + (size-1)*stride + size
	if stride <= 0 || size <= 0 || len(dst) < needed {
		return nil, ErrInvalidData
	}
	return dst[offset:], nil
}

func checkTransformAddArgsHigh(dst []uint16, block []int32, blockLen int, stride int, size int) error {
	if len(block) < blockLen {
		return ErrInvalidData
	}
	_, err := transformBlockDestinationHigh(dst, 0, stride, size)
	return err
}

func checkLoopFilterHighArgs(pix []uint16, offset int, xstride int, ystride int, innerIters int, groups int, before int, after int, bitDepth int) error {
	if err := checkH264DSPHighBitDepth(bitDepth); err != nil {
		return err
	}
	if offset < 0 || xstride <= 0 || ystride <= 0 || innerIters <= 0 || groups <= 0 {
		return ErrInvalidData
	}
	minIndex := offset - before*xstride
	maxIndex := offset + (groups*innerIters-1)*ystride + after*xstride
	if minIndex < 0 || maxIndex >= len(pix) {
		return ErrInvalidData
	}
	return nil
}

func checkH264DSPHighBitDepth(bitDepth int) error {
	switch bitDepth {
	case 9, 10, 12, 14:
		return nil
	default:
		return ErrUnsupported
	}
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func clipUintBitDepth(v int, bitDepth int) uint16 {
	if v < 0 {
		return 0
	}
	max := (1 << uint(bitDepth)) - 1
	if v > max {
		return uint16(max)
	}
	return uint16(v)
}

func clipInt(v int, lo int, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
