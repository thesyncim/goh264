// SPDX-License-Identifier: LGPL-2.1-or-later
//
// Source-shaped port of the chroma residual tail from FFmpeg n8.0.1
// libavcodec/h264_cavlc.c ff_h264_decode_mb_cavlc.

package h264

func (c *cavlcResidualContext) decodeChromaResidual(gb *bitReader, pps *PPS, scan []uint8, mbType uint32, cbp int, chromaFormatIDC int32, chromaQP [2]uint8) error {
	if chromaFormatIDC != 1 && chromaFormatIDC != 2 {
		return nil
	}

	numC8x8 := int(chromaFormatIDC)
	dcScan := h264ChromaDCScan[:]
	if chromaFormatIDC == 2 {
		dcScan = h264Chroma422DCScan[:]
	}

	if cbp&0x30 != 0 {
		for chromaIdx := 0; chromaIdx < 2; chromaIdx++ {
			offset := 256 + 16*16*chromaIdx
			if _, err := c.decodeResidual(gb, c.MB[offset:], chromaDCBlockIndex+chromaIdx, dcScan, nil, 4*numC8x8); err != nil {
				return err
			}
		}
	}

	if cbp&0x20 != 0 {
		for chromaIdx := 0; chromaIdx < 2; chromaIdx++ {
			cqm := chromaIdx + 1
			if !isIntra(mbType) {
				cqm += 3
			}
			qp := int(chromaQP[chromaIdx])
			if qp > qpMaxNum {
				return ErrInvalidData
			}
			qmul := pps.Dequant4Buffer[cqm][qp][:]
			mbOffset := 16 * (16 + 16*chromaIdx)
			for i8x8 := 0; i8x8 < numC8x8; i8x8++ {
				for i4x4 := 0; i4x4 < 4; i4x4++ {
					index := 16 + 16*chromaIdx + 8*i8x8 + i4x4
					block := c.MB[mbOffset : mbOffset+16]
					if _, err := c.decodeResidual(gb, block, index, scan[1:], qmul, 15); err != nil {
						return err
					}
					mbOffset += 16
				}
			}
		}
	} else {
		fillCAVLCNonZero(&c.NonZeroCountCache, int(h264Scan8[16]), 4, 4, 8, 0)
		fillCAVLCNonZero(&c.NonZeroCountCache, int(h264Scan8[32]), 4, 4, 8, 0)
	}

	return nil
}
