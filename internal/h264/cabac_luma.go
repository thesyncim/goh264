// SPDX-License-Identifier: LGPL-2.1-or-later
//
// Source-shaped port of FFmpeg n8.0.1 libavcodec/h264_cabac.c
// decode_cabac_luma_residual. This layer orchestrates CABAC residual syntax
// against the local residual buffers; inverse transforms and macroblock write
// back remain pending.

package h264

var cabacResidualContextCategory = [4][3]int{
	{0, 6, 10},
	{1, 7, 11},
	{2, 8, 12},
	{5, 9, 13},
}

func (c *cavlcResidualContext) decodeCABACLumaResidual(src cabacSyntaxSource, pps *PPS, scan []uint8, scan8x8 []uint8, mbType uint32, cbp int, p int, qscale int, leftCBP int, topCBP int, mbField bool, chroma444 bool) (int, error) {
	return c.decodeCABACLumaResidualTyped(src, pps, scan, scan8x8, mbType, cbp, p, qscale, leftCBP, topCBP, mbField, chroma444, false)
}

func (c *cavlcResidualContext) decodeCABACLumaResidualTyped(src cabacSyntaxSource, pps *PPS, scan []uint8, scan8x8 []uint8, mbType uint32, cbp int, p int, qscale int, leftCBP int, topCBP int, mbField bool, chroma444 bool, narrowDCT bool) (int, error) {
	if p < 0 || p > 2 || qscale < 0 || qscale > qpMaxNum || pps == nil {
		return 0, ErrInvalidData
	}

	if isIntra16x16(mbType) {
		for i := range c.MBLumaDC[p] {
			c.MBLumaDC[p][i] = 0
		}
		dc, err := c.decodeCABACResidualDCTyped(src, c.MBLumaDC[p][:], cabacResidualContextCategory[0][p], lumaDCBlockIndex+p, scan, 16, leftCBP, topCBP, mbField, false, narrowDCT)
		if err != nil {
			return 0, err
		}
		cbpTableBits := dc.CBPTableBits

		if cbp&15 != 0 {
			qmul := pps.Dequant4Buffer[p][qscale][:]
			for i4x4 := 0; i4x4 < 16; i4x4++ {
				index := 16*p + i4x4
				block := c.MB[16*index : 16*index+16]
				if _, err := c.decodeCABACResidualNonDCTyped(src, block, cabacResidualContextCategory[1][p], index, scan[1:], qmul, 15, leftCBP, topCBP, mbField, chroma444, narrowDCT); err != nil {
					return 0, err
				}
			}
			return cbpTableBits, nil
		}

		fillCAVLCNonZero(&c.NonZeroCountCache, int(h264Scan8[16*p]), 4, 4, 8, 0)
		return cbpTableBits, nil
	}

	cqm := p
	if !isIntra(mbType) {
		cqm += 3
	}
	for i8x8 := 0; i8x8 < 4; i8x8++ {
		if cbp&(1<<i8x8) != 0 {
			if is8x8DCT(mbType) {
				index := 16*p + 4*i8x8
				offset := 64*i8x8 + 256*p
				block := c.MB[offset : offset+64]
				if _, err := c.decodeCABACResidualNonDCTyped(src, block, cabacResidualContextCategory[3][p], index, scan8x8, pps.Dequant8Buffer[cqm][qscale][:], 64, leftCBP, topCBP, mbField, chroma444, narrowDCT); err != nil {
					return 0, err
				}
			} else {
				qmul := pps.Dequant4Buffer[cqm][qscale][:]
				for i4x4 := 0; i4x4 < 4; i4x4++ {
					index := 16*p + 4*i8x8 + i4x4
					block := c.MB[16*index : 16*index+16]
					if _, err := c.decodeCABACResidualNonDCTyped(src, block, cabacResidualContextCategory[2][p], index, scan, qmul, 16, leftCBP, topCBP, mbField, chroma444, narrowDCT); err != nil {
						return 0, err
					}
				}
			}
		} else {
			fillCAVLCNonZero(&c.NonZeroCountCache, int(h264Scan8[4*i8x8+16*p]), 2, 2, 8, 0)
		}
	}
	return 0, nil
}
