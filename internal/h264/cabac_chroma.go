// SPDX-License-Identifier: LGPL-2.1-or-later
//
// Source-shaped port of the CABAC chroma residual tail from FFmpeg n8.0.1
// libavcodec/h264_cabac.c ff_h264_decode_mb_cabac.

package h264

func (c *cavlcResidualContext) decodeCABACChromaResidual(src cabacSyntaxSource, pps *PPS, scan []uint8, mbType uint32, cbp int, chromaFormatIDC int32, chromaQP [2]uint8, leftCBP int, topCBP int, mbField bool) (int, error) {
	return c.decodeCABACChromaResidualTyped(src, pps, scan, mbType, cbp, chromaFormatIDC, chromaQP, leftCBP, topCBP, mbField, false)
}

func (c *cavlcResidualContext) decodeCABACChromaResidualTyped(src cabacSyntaxSource, pps *PPS, scan []uint8, mbType uint32, cbp int, chromaFormatIDC int32, chromaQP [2]uint8, leftCBP int, topCBP int, mbField bool, narrowDCT bool) (int, error) {
	if pps == nil {
		return 0, ErrInvalidData
	}
	if chromaFormatIDC != 1 && chromaFormatIDC != 2 {
		fillCAVLCNonZero(&c.NonZeroCountCache, int(h264Scan8[16]), 4, 4, 8, 0)
		fillCAVLCNonZero(&c.NonZeroCountCache, int(h264Scan8[32]), 4, 4, 8, 0)
		return 0, nil
	}

	numC8x8 := int(chromaFormatIDC)
	dcScan := h264ChromaDCScan[:]
	chroma422 := chromaFormatIDC == 2
	if chroma422 {
		dcScan = h264Chroma422DCScan[:]
	}

	cbpTableBits := 0
	if cbp&0x30 != 0 {
		for chromaIdx := 0; chromaIdx < 2; chromaIdx++ {
			offset := 256 + 16*16*chromaIdx
			dc, err := c.decodeCABACResidualDCTyped(src, c.MB[offset:], 3, chromaDCBlockIndex+chromaIdx, dcScan, 4*numC8x8, leftCBP, topCBP, mbField, chroma422, narrowDCT)
			if err != nil {
				return 0, err
			}
			cbpTableBits |= dc.CBPTableBits
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
				return 0, ErrInvalidData
			}
			qmul := pps.Dequant4Buffer[cqm][qp][:]
			mbOffset := 16 * (16 + 16*chromaIdx)
			for i8x8 := 0; i8x8 < numC8x8; i8x8++ {
				for i4x4 := 0; i4x4 < 4; i4x4++ {
					index := 16 + 16*chromaIdx + 8*i8x8 + i4x4
					block := c.MB[mbOffset : mbOffset+16]
					if _, err := c.decodeCABACResidualNonDCTyped(src, block, 4, index, scan[1:], qmul, 15, leftCBP, topCBP, mbField, false, narrowDCT); err != nil {
						return 0, err
					}
					mbOffset += 16
				}
			}
		}
	} else {
		fillCAVLCNonZero(&c.NonZeroCountCache, int(h264Scan8[16]), 4, 4, 8, 0)
		fillCAVLCNonZero(&c.NonZeroCountCache, int(h264Scan8[32]), 4, 4, 8, 0)
	}

	return cbpTableBits, nil
}
