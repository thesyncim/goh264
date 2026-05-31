// SPDX-License-Identifier: LGPL-2.1-or-later
//
// Source-shaped port of the luma residual layer from FFmpeg n8.0.1
// libavcodec/h264_cavlc.c decode_luma_residual.

package h264

const h264NonZeroCountCacheSize = 15 * 8

type cavlcResidualContext struct {
	NonZeroCountCache [h264NonZeroCountCacheSize]uint8
	MB                [48 * 16]int32
	MBLumaDC          [3][16]int32
}

func (c *cavlcResidualContext) predNonZeroCount(n int) int {
	index8 := int(h264Scan8[n])
	left := c.NonZeroCountCache[index8-1]
	top := c.NonZeroCountCache[index8-8]
	i := int(left) + int(top)
	if i < 64 {
		i = (i + 1) >> 1
	}
	return i & 31
}

func (c *cavlcResidualContext) decodeResidual(gb *bitReader, block []int32, n int, scantable []uint8, qmul []uint32, maxCoeff int) (int, error) {
	predN := n
	if n >= lumaDCBlockIndex {
		predN = (n - lumaDCBlockIndex) * 16
	}
	totalCoeff, err := decodeCAVLCResidual(gb, block, n, scantable, qmul, maxCoeff, c.predNonZeroCount(predN))
	if err != nil {
		return 0, err
	}
	c.NonZeroCountCache[h264Scan8[n]] = uint8(totalCoeff)
	return totalCoeff, nil
}

func (c *cavlcResidualContext) decodeLumaResidual(gb *bitReader, pps *PPS, scan []uint8, scan8x8 []uint8, mbType uint32, cbp int, p int, qscale int) (int, error) {
	if p < 0 || p > 2 || qscale < 0 || qscale > qpMaxNum {
		return 0, ErrInvalidData
	}

	if isIntra16x16(mbType) {
		for i := range c.MBLumaDC[p] {
			c.MBLumaDC[p][i] = 0
		}
		if _, err := c.decodeResidual(gb, c.MBLumaDC[p][:], lumaDCBlockIndex+p, scan, nil, 16); err != nil {
			return 0, err
		}

		if cbp&15 != 0 {
			for i8x8 := 0; i8x8 < 4; i8x8++ {
				for i4x4 := 0; i4x4 < 4; i4x4++ {
					index := i4x4 + 4*i8x8 + p*16
					block := c.MB[16*index : 16*index+16]
					if _, err := c.decodeResidual(gb, block, index, scan[1:], pps.Dequant4Buffer[p][qscale][:], 15); err != nil {
						return 0, err
					}
				}
			}
			return 0x0f, nil
		}

		fillCAVLCNonZero(&c.NonZeroCountCache, int(h264Scan8[p*16]), 4, 4, 8, 0)
		return 0, nil
	}

	cqm := p
	if !isIntra(mbType) {
		cqm += 3
	}
	newCBP := 0
	for i8x8 := 0; i8x8 < 4; i8x8++ {
		if cbp&(1<<i8x8) != 0 {
			if is8x8DCT(mbType) {
				offset := 64*i8x8 + 256*p
				buf := c.MB[offset : offset+64]
				for i4x4 := 0; i4x4 < 4; i4x4++ {
					index := i4x4 + 4*i8x8 + p*16
					if _, err := c.decodeResidual(gb, buf, index, scan8x8[16*i4x4:], pps.Dequant8Buffer[cqm][qscale][:], 16); err != nil {
						return 0, err
					}
				}
				nnz := int(h264Scan8[4*i8x8+p*16])
				c.NonZeroCountCache[nnz] += c.NonZeroCountCache[nnz+1] +
					c.NonZeroCountCache[nnz+8] + c.NonZeroCountCache[nnz+9]
				if c.NonZeroCountCache[nnz] != 0 {
					newCBP |= 1 << i8x8
				}
			} else {
				for i4x4 := 0; i4x4 < 4; i4x4++ {
					index := i4x4 + 4*i8x8 + p*16
					block := c.MB[16*index : 16*index+16]
					if _, err := c.decodeResidual(gb, block, index, scan, pps.Dequant4Buffer[cqm][qscale][:], 16); err != nil {
						return 0, err
					}
					newCBP |= int(c.NonZeroCountCache[h264Scan8[index]]) << i8x8
				}
			}
		} else {
			nnz := int(h264Scan8[4*i8x8+p*16])
			c.NonZeroCountCache[nnz] = 0
			c.NonZeroCountCache[nnz+1] = 0
			c.NonZeroCountCache[nnz+8] = 0
			c.NonZeroCountCache[nnz+9] = 0
		}
	}
	return newCBP, nil
}

func fillCAVLCNonZero(cache *[h264NonZeroCountCacheSize]uint8, start int, width int, height int, stride int, value uint8) {
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			cache[start+y*stride+x] = value
		}
	}
}

func isIntra(mbType uint32) bool {
	return mbType&(MBTypeIntra4x4|MBTypeIntra16x16|MBTypeIntraPCM) != 0
}

func isIntra4x4(mbType uint32) bool {
	return mbType&MBTypeIntra4x4 != 0
}

func isIntra16x16(mbType uint32) bool {
	return mbType&MBTypeIntra16x16 != 0
}

func is8x8DCT(mbType uint32) bool {
	return mbType&MBType8x8DCT != 0
}
