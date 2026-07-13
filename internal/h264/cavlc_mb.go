// SPDX-License-Identifier: LGPL-2.1-or-later
//
// Source-shaped port of the CAVLC macroblock type, coded-block-pattern,
// qscale, and intra residual orchestration from FFmpeg n8.0.1
// libavcodec/h264_cavlc.c ff_h264_decode_mb_cavlc.

package h264

const intraPredDC1288x8 = 6

type cavlcMacroblockSyntax struct {
	MBType              uint32
	PartitionCount      uint8
	CBP                 int
	CBPTable            int
	QScale              int
	ChromaQP            [2]uint8
	Intra16x16PredMode  int8
	ChromaPredMode      int32
	Intra4x4PredMode    [16]int8
	TransformSize8x8DCT bool
	IntraPCM            []byte
}

func (c *cavlcResidualContext) decodeCAVLCIntraMacroblock(gb *bitReader, pps *PPS, sps *SPS, sliceType int32, sliceTypeNoS int32, qscale int, dct8x8Allowed bool, predIntra4x4 [16]int8) (cavlcMacroblockSyntax, error) {
	var mb cavlcMacroblockSyntax
	mb, err := decodeCAVLCMBType(gb, sliceType, sliceTypeNoS)
	if err != nil {
		return mb, err
	}
	return c.decodeCAVLCIntraMacroblockAfterType(gb, pps, sps, mb, qscale, dct8x8Allowed, predIntra4x4)
}

func (c *cavlcResidualContext) decodeCAVLCIntraMacroblockAfterType(gb *bitReader, pps *PPS, sps *SPS, mb cavlcMacroblockSyntax, qscale int, dct8x8Allowed bool, predIntra4x4 [16]int8) (cavlcMacroblockSyntax, error) {
	if gb == nil || pps == nil || sps == nil {
		return mb, ErrInvalidData
	}
	if !isIntra(mb.MBType) {
		return mb, ErrUnsupported
	}
	if mb.MBType&MBTypeIntraPCM != 0 {
		return mb, ErrUnsupported
	}

	if isIntra4x4(mb.MBType) {
		var err error
		mb, err = decodeCAVLCIntra4x4ModesFromPred(gb, mb, dct8x8Allowed, predIntra4x4)
		if err != nil {
			return mb, err
		}
	}

	return c.decodeCAVLCIntraMacroblockAfterPred(gb, pps, sps, mb, qscale, dct8x8Allowed)
}

func (c *cavlcResidualContext) decodeCAVLCFrameIntraMacroblockAfterType(gb *bitReader, pps *PPS, sps *SPS, mb cavlcMacroblockSyntax, qscale int, dct8x8Allowed bool, intraCache *[h264IntraPredModeCacheSize]int8) (cavlcMacroblockSyntax, error) {
	if gb == nil || pps == nil || sps == nil {
		return mb, ErrInvalidData
	}
	if !isIntra(mb.MBType) {
		return mb, ErrUnsupported
	}
	if mb.MBType&MBTypeIntraPCM != 0 {
		return mb, ErrUnsupported
	}

	if isIntra4x4(mb.MBType) {
		var err error
		mb, err = decodeCAVLCIntra4x4ModesWithCache(gb, mb, dct8x8Allowed, intraCache)
		if err != nil {
			return mb, err
		}
	}

	return c.decodeCAVLCIntraMacroblockAfterPred(gb, pps, sps, mb, qscale, dct8x8Allowed)
}

func decodeCAVLCIntra4x4ModesFromPred(gb *bitReader, mb cavlcMacroblockSyntax, dct8x8Allowed bool, predIntra4x4 [16]int8) (cavlcMacroblockSyntax, error) {
	if gb == nil {
		return mb, ErrInvalidData
	}
	di := 1
	if dct8x8Allowed {
		flag, err := gb.readBit()
		if err != nil {
			return mb, err
		}
		if flag != 0 {
			mb.MBType |= MBType8x8DCT
			mb.TransformSize8x8DCT = true
			di = 4
		}
	}

	for i := 0; i < 16; i += di {
		mode := int(predIntra4x4[i])
		if mode < 0 || mode > 8 {
			return mb, ErrInvalidData
		}
		prevIntra4x4PredModeFlag, err := gb.readBit()
		if err != nil {
			return mb, err
		}
		if prevIntra4x4PredModeFlag == 0 {
			remMode, err := gb.readBits(3)
			if err != nil {
				return mb, err
			}
			mode = int(remMode)
			if mode >= int(predIntra4x4[i]) {
				mode++
			}
		}
		if mode > 8 {
			return mb, ErrInvalidData
		}
		for j := 0; j < di; j++ {
			mb.Intra4x4PredMode[i+j] = int8(mode)
		}
	}
	return mb, nil
}

func decodeCAVLCIntra4x4ModesWithCache(gb *bitReader, mb cavlcMacroblockSyntax, dct8x8Allowed bool, cache *[h264IntraPredModeCacheSize]int8) (cavlcMacroblockSyntax, error) {
	if gb == nil || cache == nil {
		return mb, ErrInvalidData
	}
	di := 1
	if dct8x8Allowed {
		flag, err := gb.readBit()
		if err != nil {
			return mb, err
		}
		if flag != 0 {
			mb.MBType |= MBType8x8DCT
			mb.TransformSize8x8DCT = true
			di = 4
		}
	}

	for i := 0; i < 16; i += di {
		pred, err := predIntraMode(cache, i)
		if err != nil {
			return mb, err
		}
		mode := int(pred)
		prevIntra4x4PredModeFlag, err := gb.readBit()
		if err != nil {
			return mb, err
		}
		if prevIntra4x4PredModeFlag == 0 {
			remMode, err := gb.readBits(3)
			if err != nil {
				return mb, err
			}
			mode = int(remMode)
			if mode >= int(pred) {
				mode++
			}
		}
		if mode < 0 || mode > 8 {
			return mb, ErrInvalidData
		}
		if di == 4 {
			fillIntraPredModeRectangle(cache, int(h264Scan8[i]), 2, 2, 8, int8(mode))
		} else {
			cache[h264Scan8[i]] = int8(mode)
		}
		for j := 0; j < di; j++ {
			mb.Intra4x4PredMode[i+j] = int8(mode)
		}
	}
	return mb, nil
}

func (c *cavlcResidualContext) decodeCAVLCIntraMacroblockAfterPred(gb *bitReader, pps *PPS, sps *SPS, mb cavlcMacroblockSyntax, qscale int, dct8x8Allowed bool) (cavlcMacroblockSyntax, error) {
	if gb == nil || pps == nil || sps == nil {
		return mb, ErrInvalidData
	}
	if !isIntra(mb.MBType) {
		return mb, ErrUnsupported
	}
	if mb.MBType&MBTypeIntraPCM != 0 {
		return mb, ErrUnsupported
	}

	decodeChroma := sps.ChromaFormatIDC == 1 || sps.ChromaFormatIDC == 2
	if decodeChroma {
		predMode, err := gb.readUEGolomb31()
		if err != nil {
			return mb, err
		}
		if predMode > 3 {
			return mb, ErrInvalidData
		}
		mb.ChromaPredMode = int32(predMode)
	} else {
		mb.ChromaPredMode = intraPredDC1288x8
	}

	cbp, err := decodeCAVLCCBP(gb, mb.MBType, decodeChroma, mb.CBP)
	if err != nil {
		return mb, err
	}
	mb.CBP = cbp

	if dct8x8Allowed && (mb.CBP&15) != 0 && !isIntra(mb.MBType) {
		flag, err := gb.readBit()
		if err != nil {
			return mb, err
		}
		if flag != 0 {
			mb.MBType |= MBType8x8DCT
			mb.TransformSize8x8DCT = true
		}
	}

	mb.QScale, mb.ChromaQP, mb.CBPTable, err = c.decodeCAVLCResidualPayload(gb, pps, sps, mb.MBType, mb.CBP, qscale)
	if err != nil {
		return mb, err
	}
	return mb, nil
}

func decodeCAVLCMBType(gb *bitReader, sliceType int32, sliceTypeNoS int32) (cavlcMacroblockSyntax, error) {
	var mb cavlcMacroblockSyntax
	raw, err := gb.readUEGolombLong()
	if err != nil {
		return mb, err
	}

	if sliceTypeNoS == PictureTypeB {
		if raw < 23 {
			info := h264BMBTypeInfo[raw]
			mb.PartitionCount = info.PartitionCount
			mb.MBType = info.Type
			return mb, nil
		}
		raw -= 23
	} else if sliceTypeNoS == PictureTypeP {
		if raw < 5 {
			info := h264PMBTypeInfo[raw]
			mb.PartitionCount = info.PartitionCount
			mb.MBType = info.Type
			return mb, nil
		}
		raw -= 5
	} else {
		if sliceTypeNoS != PictureTypeI {
			return mb, ErrInvalidData
		}
		if sliceType == PictureTypeSI && raw != 0 {
			raw--
		}
	}

	if raw > 25 {
		return mb, ErrInvalidData
	}
	info := h264IMBTypeInfo[raw]
	mb.PartitionCount = 0
	mb.CBP = int(info.CBP)
	mb.Intra16x16PredMode = info.PredMode
	mb.MBType = info.Type
	return mb, nil
}

func writeCAVLCMBType(bw *BitWriter, sliceType int32, sliceTypeNoS int32, mb cavlcMacroblockSyntax) error {
	if bw == nil {
		return ErrInvalidData
	}
	if sliceTypeNoS == PictureTypeB {
		for raw, info := range h264BMBTypeInfo {
			if info.Type == mb.MBType && info.PartitionCount == mb.PartitionCount {
				return bw.WriteUEGolomb(uint32(raw))
			}
		}
		raw, err := cavlcIMBTypeRaw(mb)
		if err != nil {
			return err
		}
		return bw.WriteUEGolomb(uint32(raw + len(h264BMBTypeInfo)))
	}
	if sliceTypeNoS == PictureTypeP {
		for raw, info := range h264PMBTypeInfo {
			if info.Type == mb.MBType && info.PartitionCount == mb.PartitionCount {
				return bw.WriteUEGolomb(uint32(raw))
			}
		}
		raw, err := cavlcIMBTypeRaw(mb)
		if err != nil {
			return err
		}
		return bw.WriteUEGolomb(uint32(raw + len(h264PMBTypeInfo)))
	}
	if sliceTypeNoS != PictureTypeI {
		return ErrInvalidData
	}
	raw, err := cavlcIMBTypeRaw(mb)
	if err != nil {
		return err
	}
	if sliceType == PictureTypeSI && raw > 0 {
		raw++
	}
	return bw.WriteUEGolomb(uint32(raw))
}

func cavlcIMBTypeRaw(mb cavlcMacroblockSyntax) (int, error) {
	for raw, info := range h264IMBTypeInfo {
		if info.Type == mb.MBType && info.PredMode == mb.Intra16x16PredMode && int(info.CBP) == mb.CBP {
			return raw, nil
		}
	}
	return 0, ErrInvalidData
}

func decodeCAVLCCBP(gb *bitReader, mbType uint32, decodeChroma bool, cbp int) (int, error) {
	if !isIntra16x16(mbType) {
		raw, err := gb.readUEGolombLong()
		if err != nil {
			return 0, err
		}
		if decodeChroma {
			if raw > 47 {
				return 0, ErrInvalidData
			}
			if isIntra4x4(mbType) {
				return int(h264GolombToIntra4x4CBP[raw]), nil
			}
			return int(h264GolombToInterCBP[raw]), nil
		}
		if raw > 15 {
			return 0, ErrInvalidData
		}
		if isIntra4x4(mbType) {
			return int(cavlcGolombToIntra4x4CBPGray[raw]), nil
		}
		return int(cavlcGolombToInterCBPGray[raw]), nil
	}

	if !decodeChroma && cbp > 15 {
		return 0, ErrInvalidData
	}
	return cbp, nil
}

func writeCAVLCCBP(bw *BitWriter, mbType uint32, decodeChroma bool, cbp int) error {
	if bw == nil || cbp < 0 {
		return ErrInvalidData
	}
	if isIntra16x16(mbType) {
		if !decodeChroma && cbp > 15 {
			return ErrInvalidData
		}
		return nil
	}

	var table []uint8
	if decodeChroma {
		if cbp > 47 {
			return ErrInvalidData
		}
		if isIntra4x4(mbType) {
			table = h264GolombToIntra4x4CBP[:]
		} else {
			table = h264GolombToInterCBP[:]
		}
	} else {
		if cbp > 15 {
			return ErrInvalidData
		}
		if isIntra4x4(mbType) {
			table = cavlcGolombToIntra4x4CBPGray[:]
		} else {
			table = cavlcGolombToInterCBPGray[:]
		}
	}
	for raw, value := range table {
		if int(value) == cbp {
			return bw.WriteUEGolomb(uint32(raw))
		}
	}
	return ErrInvalidData
}

func (c *cavlcResidualContext) decodeCAVLCResidualPayload(gb *bitReader, pps *PPS, sps *SPS, mbType uint32, cbp int, qscale int) (int, [2]uint8, int, error) {
	var chromaQP [2]uint8
	cbpTable := cbp
	if pps == nil || sps == nil || qscale < 0 {
		return qscale, chromaQP, cbpTable, ErrInvalidData
	}
	if cbp != 0 || isIntra16x16(mbType) {
		clear(c.MB[:])
	}

	if cbp != 0 || isIntra16x16(mbType) {
		dquant, err := gb.readSEGolombLong()
		if err != nil {
			return qscale, chromaQP, cbpTable, err
		}
		maxQP := int32(51 + 6*(sps.BitDepthLuma-8))
		qscale, err = updateCAVLCQScale(qscale, dquant, maxQP)
		if err != nil {
			return qscale, chromaQP, cbpTable, err
		}
		if qscale > qpMaxNum {
			return qscale, chromaQP, cbpTable, ErrInvalidData
		}
		chromaQP[0] = pps.ChromaQPTable[0][qscale]
		chromaQP[1] = pps.ChromaQPTable[1][qscale]

		mbField := mbType&MBTypeInterlaced != 0
		scan, scan8x8 := h264CAVLCScansForQScale(sps, qscale, mbField)
		ret, err := c.decodeLumaResidual(gb, pps, scan, scan8x8, mbType, cbp, 0, qscale)
		if err != nil {
			return qscale, chromaQP, cbpTable, err
		}
		cbpTable |= ret << 12
		if sps.ChromaFormatIDC == 3 {
			if _, err := c.decodeLumaResidual(gb, pps, scan, scan8x8, mbType, cbp, 1, int(chromaQP[0])); err != nil {
				return qscale, chromaQP, cbpTable, err
			}
			if _, err := c.decodeLumaResidual(gb, pps, scan, scan8x8, mbType, cbp, 2, int(chromaQP[1])); err != nil {
				return qscale, chromaQP, cbpTable, err
			}
		} else if err := c.decodeChromaResidual(gb, pps, scan, mbType, cbp, int32(sps.ChromaFormatIDC), chromaQP); err != nil {
			return qscale, chromaQP, cbpTable, err
		}
		return qscale, chromaQP, cbpTable, nil
	}

	clearCAVLCResidualCaches(c)
	return qscale, chromaQP, cbpTable, nil
}

func (c *cavlcResidualContext) writeCAVLCInterResidualPayload(bw *BitWriter, pps *PPS, sps *SPS, mbType uint32, cbp int, qscale int, nextQScale int) (int, error) {
	if bw == nil || pps == nil || sps == nil || c == nil || cbp < 0 || qscale < 0 || nextQScale < 0 {
		return 0, ErrInvalidData
	}
	if isIntra(mbType) || is8x8DCT(mbType) || sps.ChromaFormatIDC != 1 {
		return 0, ErrUnsupported
	}
	if cbp == 0 {
		clearCAVLCResidualCaches(c)
		return 0, nil
	}

	maxQP := int32(51 + 6*(sps.BitDepthLuma-8))
	if err := writeCAVLCDQuantForQScale(bw, qscale, nextQScale, maxQP); err != nil {
		return 0, err
	}
	if nextQScale > qpMaxNum {
		return 0, ErrInvalidData
	}
	chromaQP := [2]uint8{
		pps.ChromaQPTable[0][nextQScale],
		pps.ChromaQPTable[1][nextQScale],
	}
	scan, _ := h264CAVLCScansForQScale(sps, nextQScale, mbType&MBTypeInterlaced != 0)
	ret, err := c.writeCAVLCLumaResidual(bw, scan, mbType, cbp, 0)
	if err != nil {
		return 0, err
	}
	if err := c.writeCAVLCChromaResidual(bw, pps, scan, mbType, cbp, chromaQP); err != nil {
		return 0, err
	}
	return cbp | ret<<12, nil
}

func (c *cavlcResidualContext) writeCAVLCLumaResidual(bw *BitWriter, scan []uint8, mbType uint32, cbp int, p int) (int, error) {
	if bw == nil || c == nil || p < 0 || p > 2 || isIntra16x16(mbType) || is8x8DCT(mbType) {
		return 0, ErrInvalidData
	}
	newCBP := 0
	for i8x8 := 0; i8x8 < 4; i8x8++ {
		if cbp&(1<<i8x8) == 0 {
			nnz := int(h264Scan8[4*i8x8+p*16])
			c.NonZeroCountCache[nnz] = 0
			c.NonZeroCountCache[nnz+1] = 0
			c.NonZeroCountCache[nnz+8] = 0
			c.NonZeroCountCache[nnz+9] = 0
			continue
		}
		for i4x4 := 0; i4x4 < 4; i4x4++ {
			index := i4x4 + 4*i8x8 + p*16
			block := c.MB[16*index : 16*index+16]
			totalCoeff, err := writeCAVLCResidualBounded(bw, block, 0, scan, 16, c.predNonZeroCount(cavlcResidualPredIndex(index)))
			if err != nil {
				return 0, err
			}
			c.NonZeroCountCache[h264Scan8[index]] = uint8(totalCoeff)
			newCBP |= totalCoeff << i8x8
		}
	}
	return newCBP, nil
}

func (c *cavlcResidualContext) writeCAVLCChromaResidual(bw *BitWriter, pps *PPS, scan []uint8, mbType uint32, cbp int, chromaQP [2]uint8) error {
	if bw == nil || pps == nil || c == nil {
		return ErrInvalidData
	}
	if cbp&0x30 != 0 {
		for chromaIdx := 0; chromaIdx < 2; chromaIdx++ {
			offset := 256 + 16*16*chromaIdx
			totalCoeff, err := writeCAVLCResidualBounded(bw, c.MB[offset:], 0, h264ChromaDCScan[:], 4, c.predNonZeroCount(cavlcResidualPredIndex(chromaDCBlockIndex+chromaIdx)))
			if err != nil {
				return err
			}
			c.NonZeroCountCache[h264Scan8[chromaDCBlockIndex+chromaIdx]] = uint8(totalCoeff)
		}
	}
	if cbp&0x20 != 0 {
		for chromaIdx := 0; chromaIdx < 2; chromaIdx++ {
			qp := int(chromaQP[chromaIdx])
			if qp > qpMaxNum {
				return ErrInvalidData
			}
			mbOffset := 16 * (16 + 16*chromaIdx)
			for i4x4 := 0; i4x4 < 4; i4x4++ {
				index := 16 + 16*chromaIdx + i4x4
				block := c.MB[mbOffset : mbOffset+16]
				totalCoeff, err := writeCAVLCResidualBounded(bw, block, 0, scan[1:], 15, c.predNonZeroCount(cavlcResidualPredIndex(index)))
				if err != nil {
					return err
				}
				c.NonZeroCountCache[h264Scan8[index]] = uint8(totalCoeff)
				mbOffset += 16
			}
		}
		return nil
	}
	fillCAVLCNonZero(&c.NonZeroCountCache, int(h264Scan8[16]), 4, 4, 8, 0)
	fillCAVLCNonZero(&c.NonZeroCountCache, int(h264Scan8[32]), 4, 4, 8, 0)
	return nil
}

func cavlcResidualPredIndex(n int) int {
	if n >= lumaDCBlockIndex {
		return (n - lumaDCBlockIndex) * 16
	}
	return n
}

func updateCAVLCQScale(qscale int, dquant int32, maxQP int32) (int, error) {
	q := int32(uint32(int32(qscale)) + uint32(dquant))
	if uint32(q) > uint32(maxQP) {
		if q < 0 {
			q += maxQP + 1
		} else {
			q -= maxQP + 1
		}
		if uint32(q) > uint32(maxQP) {
			return int(maxQP), ErrInvalidData
		}
	}
	return int(q), nil
}

func writeCAVLCDQuantForQScale(bw *BitWriter, qscale int, nextQScale int, maxQP int32) error {
	if bw == nil || qscale < 0 || nextQScale < 0 || maxQP < 0 ||
		qscale > int(maxQP) || nextQScale > int(maxQP) {
		return ErrInvalidData
	}
	qpRange := int(maxQP) + 1
	delta := nextQScale - qscale
	if delta > qpRange/2 {
		delta -= qpRange
	} else if delta < -qpRange/2 {
		delta += qpRange
	}
	got, err := updateCAVLCQScale(qscale, int32(delta), maxQP)
	if err != nil || got != nextQScale {
		return ErrInvalidData
	}
	return bw.WriteSEGolomb(int32(delta))
}

func clearCAVLCResidualCaches(c *cavlcResidualContext) {
	fillCAVLCNonZero(&c.NonZeroCountCache, int(h264Scan8[0]), 4, 4, 8, 0)
	fillCAVLCNonZero(&c.NonZeroCountCache, int(h264Scan8[16]), 4, 4, 8, 0)
	fillCAVLCNonZero(&c.NonZeroCountCache, int(h264Scan8[32]), 4, 4, 8, 0)
}
