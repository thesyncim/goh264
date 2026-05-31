// SPDX-License-Identifier: LGPL-2.1-or-later
//
// Source-shaped state write-back for decoded CAVLC/CABAC macroblocks from
// FFmpeg n8.0.1 libavcodec/h264_cavlc.c ff_h264_decode_mb_cavlc,
// libavcodec/h264_cabac.c ff_h264_decode_mb_cabac, and
// libavcodec/h264_mvpred.h decode_mb_skip.

package h264

func fillIntra4x4PredModeCacheFromSyntax(cache *[h264IntraPredModeCacheSize]int8, modes *[16]int8) error {
	if cache == nil || modes == nil {
		return ErrInvalidData
	}
	for i := 0; i < 16; i++ {
		mode := modes[i]
		if mode < 0 || mode > 8 {
			return ErrInvalidData
		}
		cache[h264Scan8[i]] = mode
	}
	return nil
}

func (m *macroblockTables) writeBackCAVLCIntraMacroblock(mbXY int, mb *cavlcMacroblockSyntax, c *cavlcResidualContext, sliceNum uint16) error {
	if mb == nil || c == nil || !isIntra(mb.MBType) {
		return ErrInvalidData
	}
	if mb.MBType&MBTypeIntraPCM != 0 {
		return ErrUnsupported
	}
	if isIntra4x4(mb.MBType) {
		var cache [h264IntraPredModeCacheSize]int8
		if err := fillIntra4x4PredModeCacheFromSyntax(&cache, &mb.Intra4x4PredMode); err != nil {
			return err
		}
		if err := m.writeBackIntraPredMode(mbXY, &cache); err != nil {
			return err
		}
	}
	if err := m.writeBackMacroblockTables(mbXY, mb.MBType, mb.CBPTable, mb.QScale, sliceNum); err != nil {
		return err
	}
	m.ChromaPred[mbXY] = int8(mb.ChromaPredMode)
	return m.writeBackNonZeroCount(mbXY, &c.NonZeroCountCache)
}

func (m *macroblockTables) writeBackCAVLCIntraPCMMacroblock(mbXY int, sliceNum uint16) error {
	if err := m.writeBackMacroblockTables(mbXY, MBTypeIntraPCM, 0, 0, sliceNum); err != nil {
		return err
	}
	m.ChromaPred[mbXY] = 0
	for i := range m.NonZeroCount[mbXY] {
		m.NonZeroCount[mbXY][i] = 16
	}
	return nil
}

func (m *macroblockTables) writeBackCABACIntraMacroblock(mbXY int, mb *cavlcMacroblockSyntax, c *cavlcResidualContext, intraCache *[h264IntraPredModeCacheSize]int8, sliceNum uint16) error {
	if mb == nil || c == nil || !isIntra(mb.MBType) {
		return ErrInvalidData
	}
	if mb.MBType&MBTypeIntraPCM != 0 {
		return ErrUnsupported
	}
	if isIntra4x4(mb.MBType) {
		if intraCache == nil {
			return ErrInvalidData
		}
		if err := m.writeBackIntraPredMode(mbXY, intraCache); err != nil {
			return err
		}
	}
	if err := m.writeBackMacroblockTables(mbXY, mb.MBType, mb.CBPTable, mb.QScale, sliceNum); err != nil {
		return err
	}
	m.ChromaPred[mbXY] = int8(mb.ChromaPredMode)
	return m.writeBackNonZeroCount(mbXY, &c.NonZeroCountCache)
}

func (m *macroblockTables) writeBackCAVLCInterMacroblock(mbXY int, mb *cavlcInterMacroblockSyntax, c *cavlcResidualContext, cache *macroblockMotionCache, listCount int, sliceTypeNoS int32, sliceNum uint16) error {
	if mb == nil || c == nil || cache == nil || isIntra(mb.MBType) {
		return ErrInvalidData
	}
	if isDirect(mb.MBType) {
		return ErrUnsupported
	}
	if isInter(mb.MBType) {
		if err := fillCAVLCInterMotionCache(cache, mb, listCount); err != nil {
			return err
		}
		if err := m.writeBackMotion(mbXY, mb.MBType, sliceTypeNoS, false, &mb.SubMBType, cache); err != nil {
			return err
		}
	}
	if err := m.writeBackMacroblockTables(mbXY, mb.MBType, mb.CBPTable, mb.QScale, sliceNum); err != nil {
		return err
	}
	m.ChromaPred[mbXY] = 0
	return m.writeBackNonZeroCount(mbXY, &c.NonZeroCountCache)
}

func (m *macroblockTables) writeBackCABACInterMacroblock(mbXY int, mb *cavlcInterMacroblockSyntax, c *cavlcResidualContext, cache *macroblockMotionCache, listCount int, sliceTypeNoS int32, sliceNum uint16) error {
	if mb == nil || c == nil || cache == nil || isIntra(mb.MBType) {
		return ErrInvalidData
	}
	if isDirect(mb.MBType) {
		return ErrUnsupported
	}
	if isInter(mb.MBType) {
		if err := fillCABACInterMotionCache(cache, mb, listCount); err != nil {
			return err
		}
		if err := m.writeBackMotion(mbXY, mb.MBType, sliceTypeNoS, true, &mb.SubMBType, cache); err != nil {
			return err
		}
	}
	if err := m.writeBackMacroblockTables(mbXY, mb.MBType, mb.CBPTable, mb.QScale, sliceNum); err != nil {
		return err
	}
	m.ChromaPred[mbXY] = 0
	return m.writeBackNonZeroCount(mbXY, &c.NonZeroCountCache)
}

func (m *macroblockTables) writeBackPskipMacroblock(mbXY int, qscale int, n motionDecodeNeighbors, sliceNum uint16) error {
	return m.writeBackPskipMacroblockWithCABAC(mbXY, qscale, n, sliceNum, false)
}

func (m *macroblockTables) writeBackCABACPskipMacroblock(mbXY int, qscale int, n motionDecodeNeighbors, sliceNum uint16) error {
	return m.writeBackPskipMacroblockWithCABAC(mbXY, qscale, n, sliceNum, true)
}

func (m *macroblockTables) writeBackPskipMacroblockWithCABAC(mbXY int, qscale int, n motionDecodeNeighbors, sliceNum uint16, cabac bool) error {
	if qscale < 0 || qscale > qpMaxNum {
		return ErrInvalidData
	}
	mbType := MBType16x16 | MBTypeP0L0 | MBTypeP1L0 | MBTypeSkip
	var cache macroblockMotionCache
	n.MBType = mbType
	if err := m.predPSkipMotion(&cache, n); err != nil {
		return err
	}
	if err := m.writeBackMotion(mbXY, mbType, PictureTypeP, cabac, nil, &cache); err != nil {
		return err
	}
	if err := m.writeBackMacroblockTables(mbXY, mbType, 0, qscale, sliceNum); err != nil {
		return err
	}
	clearMacroblockNonZeroCount(&m.NonZeroCount[mbXY])
	m.ChromaPred[mbXY] = 0
	return nil
}

func (m *macroblockTables) writeBackCAVLCMacroblockTables(mbXY int, mbType uint32, cbpTable int, qscale int, sliceNum uint16) error {
	return m.writeBackMacroblockTables(mbXY, mbType, cbpTable, qscale, sliceNum)
}

func (m *macroblockTables) writeBackMacroblockTables(mbXY int, mbType uint32, cbpTable int, qscale int, sliceNum uint16) error {
	if err := m.checkCodedMBXY(mbXY); err != nil {
		return err
	}
	if qscale < 0 || qscale > qpMaxNum {
		return ErrInvalidData
	}
	m.CBPTable[mbXY] = cbpTable
	m.MacroblockTyp[mbXY] = mbType
	m.QScaleTable[mbXY] = uint8(qscale)
	m.SliceTable[mbXY] = sliceNum
	return nil
}

func clearMacroblockNonZeroCount(nnz *[h264MBNonZeroCountSize]uint8) {
	for i := range nnz {
		nnz[i] = 0
	}
}
