// SPDX-License-Identifier: LGPL-2.1-or-later
//
// Source-shaped 8-bit H.264 DSP reconstruction helpers from FFmpeg n8.0.1
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
