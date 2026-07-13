// SPDX-License-Identifier: LGPL-2.1-or-later
//
// Source-shaped port of CABAC residual coefficient syntax from FFmpeg n8.0.1
// libavcodec/h264_cabac.c get_cabac_cbf_ctx and
// decode_cabac_residual_internal. Macroblock prediction, transform execution,
// and write-back remain in the later h264_mb/h264_slice integration layer.

package h264

var cabacCBFBaseContext = [14]int{
	85, 89, 93, 97, 101, 1012, 460, 464, 468, 1016, 472, 476, 480, 1020,
}

var cabacSignificantCoeffFlagOffset = [2][14]int{
	{105 + 0, 105 + 15, 105 + 29, 105 + 44, 105 + 47, 402, 484 + 0, 484 + 15, 484 + 29, 660, 528 + 0, 528 + 15, 528 + 29, 718},
	{277 + 0, 277 + 15, 277 + 29, 277 + 44, 277 + 47, 436, 776 + 0, 776 + 15, 776 + 29, 675, 820 + 0, 820 + 15, 820 + 29, 733},
}

var cabacLastCoeffFlagOffset = [2][14]int{
	{166 + 0, 166 + 15, 166 + 29, 166 + 44, 166 + 47, 417, 572 + 0, 572 + 15, 572 + 29, 690, 616 + 0, 616 + 15, 616 + 29, 748},
	{338 + 0, 338 + 15, 338 + 29, 338 + 44, 338 + 47, 451, 864 + 0, 864 + 15, 864 + 29, 699, 908 + 0, 908 + 15, 908 + 29, 757},
}

var cabacCoeffAbsLevelM1Offset = [14]int{
	227 + 0, 227 + 10, 227 + 20, 227 + 30, 227 + 39, 426, 952 + 0, 952 + 10, 952 + 20, 708, 982 + 0, 982 + 10, 982 + 20, 766,
}

var cabacSignificantCoeffFlagOffset8x8 = [2][63]uint8{
	{
		0, 1, 2, 3, 4, 5, 5, 4, 4, 3, 3, 4, 4, 4, 5, 5,
		4, 4, 4, 4, 3, 3, 6, 7, 7, 7, 8, 9, 10, 9, 8, 7,
		7, 6, 11, 12, 13, 11, 6, 7, 8, 9, 14, 10, 9, 8, 6, 11,
		12, 13, 11, 6, 9, 14, 10, 9, 11, 12, 13, 11, 14, 10, 12,
	},
	{
		0, 1, 1, 2, 2, 3, 3, 4, 5, 6, 7, 7, 7, 8, 4, 5,
		6, 9, 10, 10, 8, 11, 12, 11, 9, 9, 10, 10, 8, 11, 12, 11,
		9, 9, 10, 10, 8, 11, 12, 11, 9, 9, 10, 10, 8, 13, 13, 9,
		9, 10, 10, 8, 13, 13, 9, 9, 10, 10, 14, 14, 14, 14, 14,
	},
}

var cabacSigCoeffOffsetDC = [7]uint8{0, 0, 1, 1, 2, 2, 2}
var cabacCoeffAbsLevel1Context = [8]uint8{1, 2, 3, 4, 0, 0, 0, 0}
var cabacCoeffAbsLevelGT1Context = [2][8]uint8{
	{5, 5, 5, 5, 6, 7, 8, 9},
	{5, 5, 5, 5, 6, 7, 8, 8},
}
var cabacCoeffAbsLevelTransition = [2][8]uint8{
	{1, 2, 3, 3, 4, 5, 6, 7},
	{4, 4, 4, 4, 5, 6, 7, 7},
}

type cabacResidualResult struct {
	Coded        bool
	CoeffCount   int
	CBPTableBits int
}

func (c *cavlcResidualContext) cabacCBFContext(cat int, idx int, maxCoeff int, isDC bool, leftCBP int, topCBP int) (int, error) {
	if cat < 0 || cat >= len(cabacCBFBaseContext) || idx < 0 || idx >= len(h264Scan8) || maxCoeff <= 0 {
		return 0, ErrInvalidData
	}

	nza, nzb := 0, 0
	if isDC {
		if cat == 3 {
			chromaIdx := idx - chromaDCBlockIndex
			if chromaIdx < 0 || chromaIdx >= 2 {
				return 0, ErrInvalidData
			}
			nza = (leftCBP >> (6 + chromaIdx)) & 0x01
			nzb = (topCBP >> (6 + chromaIdx)) & 0x01
		} else {
			lumaIdx := idx - lumaDCBlockIndex
			if lumaIdx < 0 || lumaIdx >= 3 {
				return 0, ErrInvalidData
			}
			nza = leftCBP & (0x100 << lumaIdx)
			nzb = topCBP & (0x100 << lumaIdx)
		}
	} else {
		scan := int(h264Scan8[idx])
		nza = int(c.NonZeroCountCache[scan-1])
		nzb = int(c.NonZeroCountCache[scan-8])
	}

	ctx := 0
	if nza > 0 {
		ctx++
	}
	if nzb > 0 {
		ctx += 2
	}
	return cabacCBFBaseContext[cat] + ctx, nil
}

func (c *cavlcResidualContext) decodeCABACResidualDC(src cabacSyntaxSource, block []int32, cat int, n int, scantable []uint8, maxCoeff int, leftCBP int, topCBP int, mbField bool, chroma422 bool) (cabacResidualResult, error) {
	return c.decodeCABACResidualDCTyped(src, block, cat, n, scantable, maxCoeff, leftCBP, topCBP, mbField, chroma422, false)
}

func (c *cavlcResidualContext) decodeCABACResidualDCTyped(src cabacSyntaxSource, block []int32, cat int, n int, scantable []uint8, maxCoeff int, leftCBP int, topCBP int, mbField bool, chroma422 bool, narrowDCT bool) (cabacResidualResult, error) {
	if dec, ok := src.(*cabacSyntaxDecoder); ok {
		return decodeCABACResidualDCDecoder(c, dec, block, cat, n, scantable, maxCoeff, leftCBP, topCBP, mbField, chroma422, narrowDCT)
	}
	return decodeCABACResidualDCSource(c, src, block, cat, n, scantable, maxCoeff, leftCBP, topCBP, mbField, chroma422, narrowDCT)
}

func decodeCABACResidualDCDecoder(c *cavlcResidualContext, src *cabacSyntaxDecoder, block []int32, cat int, n int, scantable []uint8, maxCoeff int, leftCBP int, topCBP int, mbField bool, chroma422 bool, narrowDCT bool) (cabacResidualResult, error) {
	var result cabacResidualResult
	ctx, err := c.cabacCBFContext(cat, n, maxCoeff, true, leftCBP, topCBP)
	if err != nil {
		return result, err
	}
	if src.get(ctx) == 0 {
		c.NonZeroCountCache[h264Scan8[n]] = 0
		return result, nil
	}
	return decodeCABACResidualInternalDecoder(c, src, block, cat, n, scantable, nil, maxCoeff, true, mbField, chroma422, narrowDCT)
}

func decodeCABACResidualDCSource[S cabacSyntaxSource](c *cavlcResidualContext, src S, block []int32, cat int, n int, scantable []uint8, maxCoeff int, leftCBP int, topCBP int, mbField bool, chroma422 bool, narrowDCT bool) (cabacResidualResult, error) {
	var result cabacResidualResult
	ctx, err := c.cabacCBFContext(cat, n, maxCoeff, true, leftCBP, topCBP)
	if err != nil {
		return result, err
	}
	if src.get(ctx) == 0 {
		c.NonZeroCountCache[h264Scan8[n]] = 0
		return result, nil
	}
	return decodeCABACResidualInternalSource(c, src, block, cat, n, scantable, nil, maxCoeff, true, mbField, chroma422, narrowDCT)
}

func (c *cavlcResidualContext) decodeCABACResidualNonDC(src cabacSyntaxSource, block []int32, cat int, n int, scantable []uint8, qmul []uint32, maxCoeff int, leftCBP int, topCBP int, mbField bool, chroma444 bool) (cabacResidualResult, error) {
	return c.decodeCABACResidualNonDCTyped(src, block, cat, n, scantable, qmul, maxCoeff, leftCBP, topCBP, mbField, chroma444, false)
}

func (c *cavlcResidualContext) decodeCABACResidualNonDCTyped(src cabacSyntaxSource, block []int32, cat int, n int, scantable []uint8, qmul []uint32, maxCoeff int, leftCBP int, topCBP int, mbField bool, chroma444 bool, narrowDCT bool) (cabacResidualResult, error) {
	if dec, ok := src.(*cabacSyntaxDecoder); ok {
		return decodeCABACResidualNonDCDecoder(c, dec, block, cat, n, scantable, qmul, maxCoeff, leftCBP, topCBP, mbField, chroma444, narrowDCT)
	}
	return decodeCABACResidualNonDCSource(c, src, block, cat, n, scantable, qmul, maxCoeff, leftCBP, topCBP, mbField, chroma444, narrowDCT)
}

func decodeCABACResidualNonDCDecoder(c *cavlcResidualContext, src *cabacSyntaxDecoder, block []int32, cat int, n int, scantable []uint8, qmul []uint32, maxCoeff int, leftCBP int, topCBP int, mbField bool, chroma444 bool, narrowDCT bool) (cabacResidualResult, error) {
	var result cabacResidualResult
	if cat < 0 || cat >= len(cabacCBFBaseContext) {
		return result, ErrInvalidData
	}
	if cat != 5 || chroma444 {
		ctx, err := c.cabacCBFContext(cat, n, maxCoeff, false, leftCBP, topCBP)
		if err != nil {
			return result, err
		}
		if src.get(ctx) == 0 {
			if maxCoeff == 64 {
				fillCAVLCNonZero(&c.NonZeroCountCache, int(h264Scan8[n]), 2, 2, 8, 0)
			} else {
				c.NonZeroCountCache[h264Scan8[n]] = 0
			}
			return result, nil
		}
	}
	return decodeCABACResidualInternalDecoder(c, src, block, cat, n, scantable, qmul, maxCoeff, false, mbField, false, narrowDCT)
}

func decodeCABACResidualNonDCSource[S cabacSyntaxSource](c *cavlcResidualContext, src S, block []int32, cat int, n int, scantable []uint8, qmul []uint32, maxCoeff int, leftCBP int, topCBP int, mbField bool, chroma444 bool, narrowDCT bool) (cabacResidualResult, error) {
	var result cabacResidualResult
	if cat < 0 || cat >= len(cabacCBFBaseContext) {
		return result, ErrInvalidData
	}
	if cat != 5 || chroma444 {
		ctx, err := c.cabacCBFContext(cat, n, maxCoeff, false, leftCBP, topCBP)
		if err != nil {
			return result, err
		}
		if src.get(ctx) == 0 {
			if maxCoeff == 64 {
				fillCAVLCNonZero(&c.NonZeroCountCache, int(h264Scan8[n]), 2, 2, 8, 0)
			} else {
				c.NonZeroCountCache[h264Scan8[n]] = 0
			}
			return result, nil
		}
	}
	return decodeCABACResidualInternalSource(c, src, block, cat, n, scantable, qmul, maxCoeff, false, mbField, false, narrowDCT)
}

func (c *cavlcResidualContext) decodeCABACResidualInternal(src cabacSyntaxSource, block []int32, cat int, n int, scantable []uint8, qmul []uint32, maxCoeff int, isDC bool, mbField bool, chroma422 bool, narrowDCT bool) (cabacResidualResult, error) {
	if dec, ok := src.(*cabacSyntaxDecoder); ok {
		return decodeCABACResidualInternalDecoder(c, dec, block, cat, n, scantable, qmul, maxCoeff, isDC, mbField, chroma422, narrowDCT)
	}
	return decodeCABACResidualInternalSource(c, src, block, cat, n, scantable, qmul, maxCoeff, isDC, mbField, chroma422, narrowDCT)
}

func decodeCABACResidualInternalSource[S cabacSyntaxSource](c *cavlcResidualContext, src S, block []int32, cat int, n int, scantable []uint8, qmul []uint32, maxCoeff int, isDC bool, mbField bool, chroma422 bool, narrowDCT bool) (cabacResidualResult, error) {
	var result cabacResidualResult
	if cat < 0 || cat >= len(cabacCBFBaseContext) || n < 0 || n >= len(h264Scan8) || maxCoeff <= 0 || maxCoeff > 64 {
		return result, ErrInvalidData
	}
	if len(scantable) < maxCoeff {
		return result, ErrInvalidData
	}
	if !isDC && len(qmul) < maxCoeff {
		return result, ErrInvalidData
	}

	fieldIdx := 0
	if mbField {
		fieldIdx = 1
	}
	sigCtxBase := cabacSignificantCoeffFlagOffset[fieldIdx][cat]
	lastCtxBase := cabacLastCoeffFlagOffset[fieldIdx][cat]
	absCtxBase := cabacCoeffAbsLevelM1Offset[cat]

	var index [64]int
	coeffCount := 0
	last := 0

	if !isDC && maxCoeff == 64 {
		sigOff := cabacSignificantCoeffFlagOffset8x8[fieldIdx]
		for last = 0; last < 63; last++ {
			if src.get(sigCtxBase+int(sigOff[last])) != 0 {
				index[coeffCount] = last
				coeffCount++
				lastOff := int(h264CABACTables[h264LastCoeffFlagOffset8x8Offset+last])
				if src.get(lastCtxBase+lastOff) != 0 {
					last = maxCoeff
					break
				}
			}
		}
		if last == maxCoeff-1 {
			index[coeffCount] = last
			coeffCount++
		}
	} else {
		coefs := maxCoeff - 1
		for last = 0; last < coefs; last++ {
			sigOff := last
			lastOff := last
			if isDC && chroma422 {
				sigOff = int(cabacSigCoeffOffsetDC[last])
				lastOff = sigOff
			}
			if src.get(sigCtxBase+sigOff) != 0 {
				index[coeffCount] = last
				coeffCount++
				if src.get(lastCtxBase+lastOff) != 0 {
					last = maxCoeff
					break
				}
			}
		}
		if last == maxCoeff-1 {
			index[coeffCount] = last
			coeffCount++
		}
	}
	if coeffCount <= 0 {
		return result, ErrInvalidData
	}

	result.Coded = true
	result.CoeffCount = coeffCount
	if isDC {
		if cat == 3 {
			result.CBPTableBits = 0x40 << (n - chromaDCBlockIndex)
		} else {
			result.CBPTableBits = 0x100 << (n - lumaDCBlockIndex)
		}
		c.NonZeroCountCache[h264Scan8[n]] = uint8(coeffCount)
	} else if maxCoeff == 64 {
		fillCAVLCNonZero(&c.NonZeroCountCache, int(h264Scan8[n]), 2, 2, 8, uint8(coeffCount))
	} else {
		c.NonZeroCountCache[h264Scan8[n]] = uint8(coeffCount)
	}

	nodeCtx := 0
	for coeffCount > 0 {
		coeffCount--
		scanPos := int(scantable[index[coeffCount]])
		if scanPos < 0 || scanPos >= len(block) {
			return result, ErrInvalidData
		}
		if !isDC && scanPos >= len(qmul) {
			return result, ErrInvalidData
		}

		ctx := absCtxBase + int(cabacCoeffAbsLevel1Context[nodeCtx])
		if src.get(ctx) == 0 {
			nodeCtx = int(cabacCoeffAbsLevelTransition[0][nodeCtx])
			if isDC {
				storeCABACResidualCoeff(block, scanPos, src.bypassSign(-1), narrowDCT)
			} else {
				storeCABACResidualCoeff(block, scanPos, (src.bypassSign(-int32(qmul[scanPos]))+32)>>6, narrowDCT)
			}
			continue
		}

		coeffAbs := 2
		gt1Index := 0
		if isDC && chroma422 {
			gt1Index = 1
		}
		ctx = absCtxBase + int(cabacCoeffAbsLevelGT1Context[gt1Index][nodeCtx])
		nodeCtx = int(cabacCoeffAbsLevelTransition[1][nodeCtx])

		for coeffAbs < 15 && src.get(ctx) != 0 {
			coeffAbs++
		}
		if coeffAbs >= 15 {
			j := 0
			for src.bypass() != 0 && j < 16+7 {
				j++
			}
			coeffAbs = 1
			for j > 0 {
				j--
				coeffAbs += coeffAbs + src.bypass()
			}
			coeffAbs += 14
		}

		if isDC {
			storeCABACResidualCoeff(block, scanPos, src.bypassSign(-int32(coeffAbs)), narrowDCT)
		} else {
			storeCABACResidualCoeff(block, scanPos, (src.bypassSign(-int32(coeffAbs))*int32(qmul[scanPos])+32)>>6, narrowDCT)
		}
	}
	return result, nil
}

// decodeCABACResidualInternalDecoder is the production specialization. It is
// deliberately concrete rather than generic: Go's shape instantiation still
// dispatches interface-constrained method calls through a dictionary, while
// the decoder's CABAC bins are hot enough for direct calls to matter.
func decodeCABACResidualInternalDecoder(c *cavlcResidualContext, src *cabacSyntaxDecoder, block []int32, cat int, n int, scantable []uint8, qmul []uint32, maxCoeff int, isDC bool, mbField bool, chroma422 bool, narrowDCT bool) (cabacResidualResult, error) {
	var result cabacResidualResult
	if cat < 0 || cat >= len(cabacCBFBaseContext) || n < 0 || n >= len(h264Scan8) || maxCoeff <= 0 || maxCoeff > 64 {
		return result, ErrInvalidData
	}
	if len(scantable) < maxCoeff {
		return result, ErrInvalidData
	}
	if !isDC && len(qmul) < maxCoeff {
		return result, ErrInvalidData
	}

	fieldIdx := 0
	if mbField {
		fieldIdx = 1
	}
	sigCtxBase := cabacSignificantCoeffFlagOffset[fieldIdx][cat]
	lastCtxBase := cabacLastCoeffFlagOffset[fieldIdx][cat]
	absCtxBase := cabacCoeffAbsLevelM1Offset[cat]

	var index [64]int
	coeffCount := 0
	last := 0

	if !isDC && maxCoeff == 64 {
		sigOff := cabacSignificantCoeffFlagOffset8x8[fieldIdx]
		for last = 0; last < 63; last++ {
			if src.get(sigCtxBase+int(sigOff[last])) != 0 {
				index[coeffCount] = last
				coeffCount++
				lastOff := int(h264CABACTables[h264LastCoeffFlagOffset8x8Offset+last])
				if src.get(lastCtxBase+lastOff) != 0 {
					last = maxCoeff
					break
				}
			}
		}
		if last == maxCoeff-1 {
			index[coeffCount] = last
			coeffCount++
		}
	} else {
		coefs := maxCoeff - 1
		for last = 0; last < coefs; last++ {
			sigOff := last
			lastOff := last
			if isDC && chroma422 {
				sigOff = int(cabacSigCoeffOffsetDC[last])
				lastOff = sigOff
			}
			if src.get(sigCtxBase+sigOff) != 0 {
				index[coeffCount] = last
				coeffCount++
				if src.get(lastCtxBase+lastOff) != 0 {
					last = maxCoeff
					break
				}
			}
		}
		if last == maxCoeff-1 {
			index[coeffCount] = last
			coeffCount++
		}
	}
	if coeffCount <= 0 {
		return result, ErrInvalidData
	}

	result.Coded = true
	result.CoeffCount = coeffCount
	if isDC {
		if cat == 3 {
			result.CBPTableBits = 0x40 << (n - chromaDCBlockIndex)
		} else {
			result.CBPTableBits = 0x100 << (n - lumaDCBlockIndex)
		}
		c.NonZeroCountCache[h264Scan8[n]] = uint8(coeffCount)
	} else if maxCoeff == 64 {
		fillCAVLCNonZero(&c.NonZeroCountCache, int(h264Scan8[n]), 2, 2, 8, uint8(coeffCount))
	} else {
		c.NonZeroCountCache[h264Scan8[n]] = uint8(coeffCount)
	}

	nodeCtx := 0
	for coeffCount > 0 {
		coeffCount--
		scanPos := int(scantable[index[coeffCount]])
		if scanPos < 0 || scanPos >= len(block) {
			return result, ErrInvalidData
		}
		if !isDC && scanPos >= len(qmul) {
			return result, ErrInvalidData
		}

		ctx := absCtxBase + int(cabacCoeffAbsLevel1Context[nodeCtx])
		if src.get(ctx) == 0 {
			nodeCtx = int(cabacCoeffAbsLevelTransition[0][nodeCtx])
			if isDC {
				storeCABACResidualCoeff(block, scanPos, src.bypassSign(-1), narrowDCT)
			} else {
				storeCABACResidualCoeff(block, scanPos, (src.bypassSign(-int32(qmul[scanPos]))+32)>>6, narrowDCT)
			}
			continue
		}

		coeffAbs := 2
		gt1Index := 0
		if isDC && chroma422 {
			gt1Index = 1
		}
		ctx = absCtxBase + int(cabacCoeffAbsLevelGT1Context[gt1Index][nodeCtx])
		nodeCtx = int(cabacCoeffAbsLevelTransition[1][nodeCtx])

		for coeffAbs < 15 && src.get(ctx) != 0 {
			coeffAbs++
		}
		if coeffAbs >= 15 {
			j := 0
			for src.bypass() != 0 && j < 16+7 {
				j++
			}
			coeffAbs = 1
			for j > 0 {
				j--
				coeffAbs += coeffAbs + src.bypass()
			}
			coeffAbs += 14
		}

		if isDC {
			storeCABACResidualCoeff(block, scanPos, src.bypassSign(-int32(coeffAbs)), narrowDCT)
		} else {
			storeCABACResidualCoeff(block, scanPos, (src.bypassSign(-int32(coeffAbs))*int32(qmul[scanPos])+32)>>6, narrowDCT)
		}
	}
	return result, nil
}

func storeCABACResidualCoeff(block []int32, pos int, value int32, narrowDCT bool) {
	if narrowDCT {
		value = dctcoef8(int(value))
	}
	block[pos] = value
}
