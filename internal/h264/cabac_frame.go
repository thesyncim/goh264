// SPDX-License-Identifier: LGPL-2.1-or-later
//
// Source-shaped frame-MB CABAC macroblock handoff from FFmpeg n8.0.1
// libavcodec/h264_cabac.c ff_h264_decode_mb_cabac. This layer connects the
// translated CABAC syntax, residual, motion-cache, and state write-back pieces
// while still stopping before reconstruction/deblocking.

package h264

type cabacFrameMacroblockInput struct {
	MBXY                   int
	SliceNum               uint16
	SliceType              int32
	SliceTypeNoS           int32
	QScale                 int
	LastQScaleDiff         int
	MBFieldDecodingFlag    int32
	RefCount               [2]uint32
	DCT8x8Allowed          bool
	DirectSpatialMVPred    bool
	DeblockingFilter       int32
	Direct                 h264DirectMotionContext
	PPS                    *PPS
	SPS                    *SPS
	RejectUnsupportedHighB bool
}

type cabacFrameMacroblockResult struct {
	MBType              uint32
	CBP                 int
	CBPTable            int
	QScale              int
	LastQScaleDiff      int
	MBFieldDecodingFlag int32
	ChromaQP            [2]uint8
	ChromaPred          int32
	TopLeftAvailable    uint16
	TopRightAvailable   uint16
	Neighbors           macroblockDecodeNeighbors
	Intra               cavlcMacroblockSyntax
	Inter               cavlcInterMacroblockSyntax
	IntraPCM            []byte
	IsIntra             bool
	IsInter             bool
	Skipped             bool
}

type cabacFrameSliceState struct {
	QScale              int
	LastQScaleDiff      int
	PrevMBSkipped       bool
	NextMBSkipped       bool
	MBFieldDecodingFlag int32
}

func (m *macroblockTables) decodeCABACFrameSliceMacroblock(src cabacSyntaxSource, sh *SliceHeader, state *cabacFrameSliceState, mbXY int, sliceNum uint16) (cabacFrameMacroblockResult, error) {
	var work frameMacroblockDecodeWork
	return m.decodeCABACFrameSliceMacroblockWithWork(src, sh, state, mbXY, sliceNum, &work)
}

func (m *macroblockTables) decodeCABACFrameSliceMacroblockWithWork(src cabacSyntaxSource, sh *SliceHeader, state *cabacFrameSliceState, mbXY int, sliceNum uint16, work *frameMacroblockDecodeWork) (cabacFrameMacroblockResult, error) {
	return m.decodeCABACFrameSliceMacroblockWithDirectWork(src, sh, state, mbXY, sliceNum, h264DirectMotionContext{}, work)
}

func (m *macroblockTables) decodeCABACFrameSliceMacroblockWithDirectWork(src cabacSyntaxSource, sh *SliceHeader, state *cabacFrameSliceState, mbXY int, sliceNum uint16, direct h264DirectMotionContext, work *frameMacroblockDecodeWork) (cabacFrameMacroblockResult, error) {
	return m.decodeCABACFrameSliceMacroblockWithDirectWorkGuard(src, sh, state, mbXY, sliceNum, direct, work, false)
}

func (m *macroblockTables) decodeCABACFrameSliceMacroblockWithDirectWorkGuard(src cabacSyntaxSource, sh *SliceHeader, state *cabacFrameSliceState, mbXY int, sliceNum uint16, direct h264DirectMotionContext, work *frameMacroblockDecodeWork, rejectUnsupportedHighB bool) (cabacFrameMacroblockResult, error) {
	var result cabacFrameMacroblockResult
	if m == nil || src == nil || sh == nil || sh.PPS == nil || sh.SPS == nil || state == nil || work == nil {
		return result, ErrInvalidData
	}
	if sh.PictureStructure != PictureFrame && sh.PictureStructure != PictureTopField && sh.PictureStructure != PictureBottomField {
		return result, ErrUnsupported
	}
	if sh.QScale > qpMaxNum || state.QScale < 0 || state.QScale > qpMaxNum {
		return result, ErrInvalidData
	}

	frameMBAFF := sh.PictureStructure == PictureFrame && sh.SPS.MBAFF != 0
	mbX := mbXY % m.MBStride
	mbY := mbXY / m.MBStride
	if frameMBAFF && (mbY&1) != 0 && state.MBFieldDecodingFlag != 0 {
		return result, ErrUnsupported
	}

	if sh.SliceTypeNoS != PictureTypeI {
		var skip bool
		var err error
		if frameMBAFF && (mbY&1) != 0 && state.PrevMBSkipped {
			skip = state.NextMBSkipped
		} else if frameMBAFF {
			skip, err = m.decodeCABACMBSkipMBAFF(src, mbXY, mbX, mbY, sh.SliceTypeNoS, sliceNum, state.MBFieldDecodingFlag)
		} else {
			skip, err = m.decodeCABACMBSkip(src, mbXY, sh.SliceTypeNoS, sliceNum)
		}
		if err != nil {
			return result, err
		}
		if skip {
			state.PrevMBSkipped = true
			state.LastQScaleDiff = 0
			if frameMBAFF && (mbY&1) == 0 {
				next, err := m.decodeCABACMBSkipMBAFF(src, mbXY+m.MBStride, mbX, mbY+1, sh.SliceTypeNoS, sliceNum, state.MBFieldDecodingFlag)
				if err != nil {
					return result, err
				}
				state.NextMBSkipped = next
				if !next {
					flag, err := m.decodeCABACFieldDecodingFlag(src, mbXY, mbX, sliceNum, state.MBFieldDecodingFlag != 0)
					if err != nil {
						return result, err
					}
					state.MBFieldDecodingFlag = flag
					result.MBFieldDecodingFlag = flag
				}
				if state.MBFieldDecodingFlag != 0 {
					result.MBType = MBTypeInterlaced
					return result, ErrUnsupported
				}
			}
			return m.writeBackCABACFrameSkipMacroblockWithDirectWorkGuard(sh, state.QScale, mbXY, sliceNum, direct, work, rejectUnsupportedHighB)
		}
	}
	state.PrevMBSkipped = false
	if frameMBAFF && (mbY&1) == 0 {
		flag, err := m.decodeCABACFieldDecodingFlag(src, mbXY, mbX, sliceNum, state.MBFieldDecodingFlag != 0)
		if err != nil {
			return result, err
		}
		state.MBFieldDecodingFlag = flag
		result.MBFieldDecodingFlag = flag
		if flag != 0 {
			result.MBType = MBTypeInterlaced
			return result, ErrUnsupported
		}
	}

	result, err := m.decodeCABACFrameMacroblockWithWork(src, cabacFrameMacroblockInput{
		MBXY:                   mbXY,
		SliceNum:               sliceNum,
		SliceType:              sh.SliceType,
		SliceTypeNoS:           sh.SliceTypeNoS,
		QScale:                 state.QScale,
		LastQScaleDiff:         state.LastQScaleDiff,
		MBFieldDecodingFlag:    state.MBFieldDecodingFlag,
		RefCount:               sh.RefCount,
		DCT8x8Allowed:          sh.PPS.Transform8x8Mode != 0,
		DirectSpatialMVPred:    sh.DirectSpatialMVPred != 0,
		DeblockingFilter:       sh.DeblockingFilter,
		Direct:                 direct,
		PPS:                    sh.PPS,
		SPS:                    sh.SPS,
		RejectUnsupportedHighB: rejectUnsupportedHighB,
	}, work)
	if err != nil {
		return result, err
	}
	state.LastQScaleDiff = result.LastQScaleDiff
	if result.MBType&MBTypeIntraPCM == 0 {
		state.QScale = result.QScale
	}
	return result, nil
}

func (m *macroblockTables) decodeCABACFrameMacroblock(src cabacSyntaxSource, in cabacFrameMacroblockInput) (cabacFrameMacroblockResult, error) {
	var work frameMacroblockDecodeWork
	return m.decodeCABACFrameMacroblockWithWork(src, in, &work)
}

func (m *macroblockTables) decodeCABACFrameMacroblockWithWork(src cabacSyntaxSource, in cabacFrameMacroblockInput, work *frameMacroblockDecodeWork) (cabacFrameMacroblockResult, error) {
	var result cabacFrameMacroblockResult
	if m == nil || src == nil || in.PPS == nil || in.SPS == nil || work == nil {
		return result, ErrInvalidData
	}
	if in.QScale < 0 || in.QScale > qpMaxNum {
		return result, ErrInvalidData
	}
	if in.MBFieldDecodingFlag != 0 {
		result.MBFieldDecodingFlag = in.MBFieldDecodingFlag
		result.MBType = MBTypeInterlaced
		return result, ErrUnsupported
	}
	*work = frameMacroblockDecodeWork{}

	neighbors, err := m.fillDecodeNeighborsFrame(in.MBXY, in.SliceNum, 0)
	if err != nil {
		return result, err
	}
	base, err := decodeCABACMBType(src, in.SliceType, in.SliceTypeNoS, neighbors.LeftType[h264LeftTop], neighbors.TopType)
	if err != nil {
		return result, err
	}
	result.MBType = base.MBType
	if in.RejectUnsupportedHighB {
		if err := validateHighFrameSliceBaseMacroblockForDecode(in.SliceTypeNoS, base.MBType); err != nil {
			return result, err
		}
	}
	if base.MBType&MBTypeIntraPCM != 0 {
		return m.decodeCABACFrameIntraPCMMacroblock(src, in, base, result)
	}

	listCount, err := cavlcFrameListCount(in.SliceTypeNoS)
	if err != nil {
		return result, err
	}

	cacheResult, err := m.fillFrameMacroblockDecodeCaches(&work.IntraCache, &work.Residual, &work.Motion, frameMacroblockDecodeCacheInput{
		MBXY:                 in.MBXY,
		SliceNum:             in.SliceNum,
		MBType:               base.MBType,
		ListCount:            listCount,
		SliceTypeNoS:         in.SliceTypeNoS,
		CABAC:                true,
		ConstrainedIntraPred: in.PPS.ConstrainedIntraPred != 0,
		DirectSpatialMVPred:  in.DirectSpatialMVPred,
	})
	if err != nil {
		return result, err
	}
	result.Neighbors = cacheResult.Neighbors

	if isIntra(base.MBType) {
		return m.decodeCABACFrameIntraMacroblock(src, in, base, &work.Residual, &work.IntraCache, cacheResult, result)
	}
	return m.decodeCABACFrameInterMacroblock(src, in, base, &work.Residual, &work.Motion, listCount, cacheResult, result)
}

func (m *macroblockTables) decodeCABACFrameIntraPCMMacroblock(src cabacSyntaxSource, in cabacFrameMacroblockInput, base cavlcMacroblockSyntax, result cabacFrameMacroblockResult) (cabacFrameMacroblockResult, error) {
	pcm, err := readCABACIntraPCMBytes(src, in.SPS)
	if err != nil {
		return result, err
	}
	base.IntraPCM = pcm
	base.QScale = 0
	base.CBPTable = 0xf7ef
	if err := m.writeBackCABACIntraPCMMacroblock(in.MBXY, in.SliceNum); err != nil {
		return result, err
	}
	result.MBType = base.MBType
	result.CBP = 0
	result.CBPTable = 0xf7ef
	result.QScale = 0
	result.LastQScaleDiff = 0
	result.Intra = base
	result.IntraPCM = pcm
	result.IsIntra = true
	return result, nil
}

func (m *macroblockTables) decodeCABACMBSkip(src cabacSyntaxSource, mbXY int, sliceTypeNoS int32, sliceNum uint16) (bool, error) {
	if m == nil || src == nil {
		return false, ErrInvalidData
	}
	if err := m.checkCodedMBXY(mbXY); err != nil {
		return false, err
	}
	ctx := 0
	mbX := mbXY % m.MBStride
	leftXY := -1
	if mbX > 0 {
		leftXY = mbXY - 1
	}
	topXY := mbXY - m.MBStride
	if m.sameSlice(leftXY, sliceNum) && !isSkip(m.MacroblockTyp[leftXY]) {
		ctx++
	}
	if m.sameSlice(topXY, sliceNum) && !isSkip(m.MacroblockTyp[topXY]) {
		ctx++
	}
	if sliceTypeNoS == PictureTypeB {
		ctx += 13
	} else if sliceTypeNoS != PictureTypeP {
		return false, ErrInvalidData
	}
	return src.get(11+ctx) != 0, nil
}

func (m *macroblockTables) decodeCABACMBSkipMBAFF(src cabacSyntaxSource, mbXY int, mbX int, mbY int, sliceTypeNoS int32, sliceNum uint16, mbFieldDecodingFlag int32) (bool, error) {
	if m == nil || src == nil || mbX < 0 || mbY < 0 {
		return false, ErrInvalidData
	}
	if err := m.checkCodedMBXY(mbXY); err != nil {
		return false, err
	}
	if sliceNum == ^uint16(0) {
		return false, ErrInvalidData
	}
	mbPairXY := mbX + (mbY&^1)*m.MBStride
	mbaXY := mbPairXY - 1
	if (mbY&1) != 0 &&
		m.sameSlice(mbaXY, sliceNum) &&
		(mbFieldDecodingFlag != 0) == (m.MacroblockTyp[mbaXY]&MBTypeInterlaced != 0) {
		mbaXY += m.MBStride
	}
	var mbbXY int
	if mbFieldDecodingFlag != 0 {
		mbbXY = mbPairXY - m.MBStride
		if (mbY&1) == 0 &&
			m.sameSlice(mbbXY, sliceNum) &&
			m.MacroblockTyp[mbbXY]&MBTypeInterlaced != 0 {
			mbbXY -= m.MBStride
		}
	} else {
		mbbXY = mbX + (mbY-1)*m.MBStride
	}

	ctx := 0
	if m.sameSlice(mbaXY, sliceNum) && !isSkip(m.MacroblockTyp[mbaXY]) {
		ctx++
	}
	if m.sameSlice(mbbXY, sliceNum) && !isSkip(m.MacroblockTyp[mbbXY]) {
		ctx++
	}
	if sliceTypeNoS == PictureTypeB {
		ctx += 13
	} else if sliceTypeNoS != PictureTypeP {
		return false, ErrInvalidData
	}
	return src.get(11+ctx) != 0, nil
}

func (m *macroblockTables) decodeCABACFieldDecodingFlag(src cabacSyntaxSource, mbXY int, mbX int, sliceNum uint16, prevMBField bool) (int32, error) {
	if m == nil || src == nil || mbX < 0 {
		return 0, ErrInvalidData
	}
	if err := m.checkCodedMBXY(mbXY); err != nil {
		return 0, err
	}
	if sliceNum == ^uint16(0) {
		return 0, ErrInvalidData
	}
	ctx := 0
	if prevMBField && mbX != 0 {
		ctx++
	}
	mbbXY := mbXY - 2*m.MBStride
	if mbbXY >= 0 && mbbXY < len(m.MacroblockTyp) &&
		m.SliceTable[mbbXY] == sliceNum &&
		m.MacroblockTyp[mbbXY]&MBTypeInterlaced != 0 {
		ctx++
	}
	return int32(src.get(70 + ctx)), nil
}

func (m *macroblockTables) writeBackCABACFrameSkipMacroblock(sh *SliceHeader, mbXY int, sliceNum uint16) (cabacFrameMacroblockResult, error) {
	var work frameMacroblockDecodeWork
	if sh == nil {
		return cabacFrameMacroblockResult{}, ErrInvalidData
	}
	return m.writeBackCABACFrameSkipMacroblockWithWork(sh, int(sh.QScale), mbXY, sliceNum, &work)
}

func (m *macroblockTables) writeBackCABACFrameSkipMacroblockWithWork(sh *SliceHeader, qscale int, mbXY int, sliceNum uint16, work *frameMacroblockDecodeWork) (cabacFrameMacroblockResult, error) {
	return m.writeBackCABACFrameSkipMacroblockWithDirectWork(sh, qscale, mbXY, sliceNum, h264DirectMotionContext{}, work)
}

func (m *macroblockTables) writeBackCABACFrameSkipMacroblockWithDirectWork(sh *SliceHeader, qscale int, mbXY int, sliceNum uint16, direct h264DirectMotionContext, work *frameMacroblockDecodeWork) (cabacFrameMacroblockResult, error) {
	return m.writeBackCABACFrameSkipMacroblockWithDirectWorkGuard(sh, qscale, mbXY, sliceNum, direct, work, false)
}

func (m *macroblockTables) writeBackCABACFrameSkipMacroblockWithDirectWorkGuard(sh *SliceHeader, qscale int, mbXY int, sliceNum uint16, direct h264DirectMotionContext, work *frameMacroblockDecodeWork, rejectUnsupportedHighB bool) (cabacFrameMacroblockResult, error) {
	var result cabacFrameMacroblockResult
	if sh == nil || work == nil {
		return result, ErrInvalidData
	}
	if qscale < 0 || qscale > qpMaxNum {
		return result, ErrInvalidData
	}
	if sh.SliceTypeNoS == PictureTypeB {
		return m.writeBackCABACFrameBSkipMacroblockWithDirectWorkGuard(sh, qscale, mbXY, sliceNum, direct, work, rejectUnsupportedHighB)
	}
	if sh.SliceTypeNoS != PictureTypeP {
		return result, ErrUnsupported
	}

	mbType := MBType16x16 | MBTypeP0L0 | MBTypeP1L0 | MBTypeSkip
	neighbors, err := m.fillDecodeNeighborsFrame(mbXY, sliceNum, mbType)
	if err != nil {
		return result, err
	}
	*work = frameMacroblockDecodeWork{}
	if err := m.writeBackCABACPskipMacroblockWithMotion(mbXY, qscale, neighbors.motionNeighbors(mbType, 1, PictureTypeP, true, false), sliceNum, &work.Motion); err != nil {
		return result, err
	}

	result.MBType = mbType
	result.CBP = 0
	result.CBPTable = 0
	result.QScale = qscale
	result.LastQScaleDiff = 0
	result.Neighbors = neighbors
	result.IsInter = true
	result.Skipped = true
	return result, nil
}

func (m *macroblockTables) writeBackCABACFrameBSkipMacroblockWithDirectWork(sh *SliceHeader, qscale int, mbXY int, sliceNum uint16, direct h264DirectMotionContext, work *frameMacroblockDecodeWork) (cabacFrameMacroblockResult, error) {
	return m.writeBackCABACFrameBSkipMacroblockWithDirectWorkGuard(sh, qscale, mbXY, sliceNum, direct, work, false)
}

func (m *macroblockTables) writeBackCABACFrameBSkipMacroblockWithDirectWorkGuard(sh *SliceHeader, qscale int, mbXY int, sliceNum uint16, direct h264DirectMotionContext, work *frameMacroblockDecodeWork, rejectUnsupportedHighB bool) (cabacFrameMacroblockResult, error) {
	var result cabacFrameMacroblockResult
	mbType := MBTypeL0L1 | MBTypeDirect2 | MBTypeSkip
	neighbors, err := m.fillDecodeNeighborsFrame(mbXY, sliceNum, mbType)
	if err != nil {
		return result, err
	}
	*work = frameMacroblockDecodeWork{}
	if direct.DirectSpatialMVPred {
		if err := m.fillMotionDecodeCaches(&work.Motion, neighbors.motionNeighbors(mbType, 2, PictureTypeB, true, true)); err != nil {
			return result, err
		}
	}
	var subMBType [4]uint32
	if err := m.predDirectMotionFrame(&work.Motion, mbXY, &mbType, &subMBType, direct); err != nil {
		return result, err
	}
	if rejectUnsupportedHighB {
		if err := validateHighFrameSliceMacroblockForReconstructWithSubMB(sh, mbType, &subMBType, 0, 0); err != nil {
			return result, err
		}
	}
	if err := m.writeBackBskipMacroblockWithMotion(mbXY, qscale, mbType, true, &subMBType, sliceNum, &work.Motion); err != nil {
		return result, err
	}

	result.MBType = mbType
	result.CBP = 0
	result.CBPTable = 0
	result.QScale = qscale
	result.LastQScaleDiff = 0
	result.Neighbors = neighbors
	result.Inter.SubMBType = subMBType
	result.IsInter = true
	result.Skipped = true
	return result, nil
}

func (m *macroblockTables) decodeCABACFrameIntraMacroblock(src cabacSyntaxSource, in cabacFrameMacroblockInput, base cavlcMacroblockSyntax, residual *cavlcResidualContext, intraCache *[h264IntraPredModeCacheSize]int8, cacheResult frameMacroblockDecodeCacheResult, result cabacFrameMacroblockResult) (cabacFrameMacroblockResult, error) {
	mb := base
	var writeBackIntraCache [h264IntraPredModeCacheSize]int8
	if isIntra4x4(mb.MBType) {
		di := 1
		if in.DCT8x8Allowed && src.get(399+cacheResult.Intra.NeighborTransformSize) != 0 {
			mb.MBType |= MBType8x8DCT
			mb.TransformSize8x8DCT = true
			di = 4
		}
		for i := 0; i < 16; i += di {
			pred, err := predIntraMode(intraCache, i)
			if err != nil {
				return result, err
			}
			mode := decodeCABACMBIntra4x4PredMode(src, int(pred))
			if mode < 0 || mode > 8 {
				return result, ErrInvalidData
			}
			if di == 4 {
				fillIntraPredModeRectangle(intraCache, int(h264Scan8[i]), 2, 2, 8, int8(mode))
			} else {
				intraCache[h264Scan8[i]] = int8(mode)
			}
			for j := 0; j < di; j++ {
				mb.Intra4x4PredMode[i+j] = int8(mode)
			}
		}
		writeBackIntraCache = *intraCache
		if err := validateCABACFrameIntra4x4PredModes(intraCache, cacheResult.Intra); err != nil {
			return result, err
		}
	} else if isIntra16x16(mb.MBType) {
		mode, err := checkIntraPredMode(int(mb.Intra16x16PredMode), cacheResult.Intra.TopSamplesAvailable, cacheResult.Intra.LeftSamplesAvailable, false)
		if err != nil {
			return result, err
		}
		mb.Intra16x16PredMode = int8(mode)
	}

	rawChromaPred := int32(0)
	if in.SPS.ChromaFormatIDC == 1 || in.SPS.ChromaFormatIDC == 2 {
		raw := m.decodeCABACMBChromaPredMode(src, cacheResult.Neighbors)
		rawChromaPred = int32(raw)
		mode, err := checkIntraPredMode(raw, cacheResult.Intra.TopSamplesAvailable, cacheResult.Intra.LeftSamplesAvailable, true)
		if err != nil {
			return result, err
		}
		mb.ChromaPredMode = int32(mode)
	} else {
		mb.ChromaPredMode = intraPredDC1288x8
	}

	if !isIntra16x16(mb.MBType) {
		mb.CBP = decodeCABACMBCBPLuma(src, cacheResult.Residual.LeftCBP, cacheResult.Residual.TopCBP)
		if in.SPS.ChromaFormatIDC == 1 || in.SPS.ChromaFormatIDC == 2 {
			mb.CBP |= decodeCABACMBCBPChroma(src, cacheResult.Residual.LeftCBP, cacheResult.Residual.TopCBP) << 4
		}
	} else if (in.SPS.ChromaFormatIDC != 1 && in.SPS.ChromaFormatIDC != 2) && mb.CBP > 15 {
		return result, ErrInvalidData
	}

	qscale, chromaQP, cbpTable, lastDiff, err := residual.decodeCABACResidualPayload(src, in.PPS, in.SPS, mb.MBType, mb.CBP, in.QScale, in.LastQScaleDiff, cacheResult.Residual)
	if err != nil {
		return result, err
	}
	mb.QScale = qscale
	mb.ChromaQP = chromaQP
	mb.CBPTable = cbpTable
	if err := m.writeBackCABACIntraMacroblockWithChromaPred(in.MBXY, &mb, residual, &writeBackIntraCache, int8(rawChromaPred), in.SliceNum); err != nil {
		return result, err
	}

	result.MBType = mb.MBType
	result.CBP = mb.CBP
	result.CBPTable = mb.CBPTable
	result.QScale = mb.QScale
	result.LastQScaleDiff = lastDiff
	result.ChromaQP = mb.ChromaQP
	result.ChromaPred = mb.ChromaPredMode
	result.TopLeftAvailable = cacheResult.Intra.TopLeftSamplesAvailable
	result.TopRightAvailable = cacheResult.Intra.TopRightSamplesAvailable
	result.Intra = mb
	result.IsIntra = true
	return result, nil
}

func (m *macroblockTables) decodeCABACFrameInterMacroblock(src cabacSyntaxSource, in cabacFrameMacroblockInput, base cavlcMacroblockSyntax, residual *cavlcResidualContext, motion *macroblockMotionCache, listCount int, cacheResult frameMacroblockDecodeCacheResult, result cabacFrameMacroblockResult) (cabacFrameMacroblockResult, error) {
	var mb cavlcInterMacroblockSyntax
	mb.cavlcMacroblockSyntax = base
	if isDirect(base.MBType) {
		if err := m.predDirectMotionFrame(motion, in.MBXY, &mb.MBType, &mb.SubMBType, in.Direct); err != nil {
			return result, err
		}
		fillMVDRectangle(&motion.MVD[0], int(h264Scan8[0]), 4, 4, 8, [2]uint8{})
		fillMVDRectangle(&motion.MVD[1], int(h264Scan8[0]), 4, 4, 8, [2]uint8{})
	} else {
		if err := m.decodeCABACInterMotionSyntax(src, &mb, motion, in.MBXY, in.SliceTypeNoS, listCount, in.RefCount, in.Direct); err != nil {
			return result, err
		}
	}

	mb.CBP = decodeCABACMBCBPLuma(src, cacheResult.Residual.LeftCBP, cacheResult.Residual.TopCBP)
	if in.SPS.ChromaFormatIDC == 1 || in.SPS.ChromaFormatIDC == 2 {
		mb.CBP |= decodeCABACMBCBPChroma(src, cacheResult.Residual.LeftCBP, cacheResult.Residual.TopCBP) << 4
	}
	dct8x8Allowed := in.DCT8x8Allowed
	if mb.PartitionCount == 4 {
		dct8x8Allowed = subMBTypesAllowDCT8x8(dct8x8Allowed, &mb.SubMBType, in.SPS.Direct8x8InferenceFlag != 0)
	}
	if dct8x8Allowed && (mb.CBP&15) != 0 {
		if src.get(399+cabacNeighborTransformSize(cacheResult.Neighbors)) != 0 {
			mb.MBType |= MBType8x8DCT
			mb.TransformSize8x8DCT = true
		}
	}

	qscale, chromaQP, cbpTable, lastDiff, err := residual.decodeCABACResidualPayload(src, in.PPS, in.SPS, mb.MBType, mb.CBP, in.QScale, in.LastQScaleDiff, cacheResult.Residual)
	if err != nil {
		return result, err
	}
	mb.QScale = qscale
	mb.ChromaQP = chromaQP
	mb.CBPTable = cbpTable
	if in.RejectUnsupportedHighB {
		sh := &SliceHeader{
			SliceTypeNoS:     in.SliceTypeNoS,
			DeblockingFilter: in.DeblockingFilter,
			PPS:              in.PPS,
			SPS:              in.SPS,
		}
		if err := validateHighFrameSliceMacroblockForReconstructWithSubMB(sh, mb.MBType, &mb.SubMBType, mb.CBP, mb.CBPTable); err != nil {
			return result, err
		}
	}
	if err := m.writeBackCABACInterMacroblock(in.MBXY, &mb, residual, motion, listCount, in.SliceTypeNoS, in.SliceNum); err != nil {
		return result, err
	}

	result.MBType = mb.MBType
	result.CBP = mb.CBP
	result.CBPTable = mb.CBPTable
	result.QScale = mb.QScale
	result.LastQScaleDiff = lastDiff
	result.ChromaQP = mb.ChromaQP
	result.Inter = mb
	result.IsInter = true
	return result, nil
}

func (m *macroblockTables) decodeCABACMBChromaPredMode(src cabacSyntaxSource, n macroblockDecodeNeighbors) int {
	ctx := 0
	if n.LeftType[h264LeftTop] != 0 && m.ChromaPred[n.LeftXY[h264LeftTop]] != 0 {
		ctx++
	}
	if n.TopType != 0 && m.ChromaPred[n.TopXY] != 0 {
		ctx++
	}
	if src.get(64+ctx) == 0 {
		return 0
	}
	if src.get(64+3) == 0 {
		return 1
	}
	if src.get(64+3) == 0 {
		return 2
	}
	return 3
}

func readCABACIntraPCMBytes(src cabacSyntaxSource, sps *SPS) ([]byte, error) {
	if src == nil || sps == nil || sps.ChromaFormatIDC >= uint32(len(h264IntraPCMSampleCount)) {
		return nil, ErrInvalidData
	}
	n, err := h264IntraPCMByteCount(int(sps.ChromaFormatIDC), int(sps.BitDepthLuma))
	if err != nil {
		return nil, err
	}
	pcmSrc, ok := src.(cabacIntraPCMSource)
	if !ok {
		return nil, ErrUnsupported
	}
	return pcmSrc.intraPCMBytes(n)
}

func (m *macroblockTables) decodeCABACInterMotionSyntax(src cabacSyntaxSource, mb *cavlcInterMacroblockSyntax, motion *macroblockMotionCache, mbXY int, sliceTypeNoS int32, listCount int, refCount [2]uint32, direct h264DirectMotionContext) error {
	if src == nil || mb == nil || motion == nil || listCount < 0 || listCount > 2 || isIntra(mb.MBType) {
		return ErrInvalidData
	}
	for list := 0; list < 2; list++ {
		for i := 0; i < 4; i++ {
			mb.Ref[list][i] = -1
		}
	}
	if isDirect(mb.MBType) {
		return ErrUnsupported
	}

	if mb.PartitionCount == 4 {
		if err := decodeCABACSubMBTypes(src, mb, sliceTypeNoS); err != nil {
			return err
		}
		hasDirectSub := sliceTypeNoS == PictureTypeB && hasDirectSubMBType(&mb.SubMBType)
		if hasDirectSub {
			if m == nil {
				return ErrInvalidData
			}
			if err := m.predDirectMotionFrame(motion, mbXY, &mb.MBType, &mb.SubMBType, direct); err != nil {
				return err
			}
			markDirectSubRefsUnavailable(motion)
			fillDirectCacheFromSubMBTypes(&motion.Direct, &mb.SubMBType)
		}
		for list := 0; list < listCount; list++ {
			for i := 0; i < 4; i++ {
				ref := int32(-1)
				if isDirect(mb.SubMBType[i]) {
					mb.Ref[list][i] = ref
					continue
				}
				if isDir(mb.SubMBType[i], 0, list) {
					var err error
					ref, err = decodeCABACRefForPartition(src, motion, sliceTypeNoS, list, 4*i, refCount[list])
					if err != nil {
						return err
					}
				}
				mb.Ref[list][i] = ref
				start := int(h264Scan8[4*i])
				motion.Ref[list][start+1] = int8(ref)
				motion.Ref[list][start+8] = int8(ref)
				motion.Ref[list][start+9] = int8(ref)
			}
		}
		for list := 0; list < listCount; list++ {
			for i := 0; i < 4; i++ {
				start := int(h264Scan8[4*i])
				motion.Ref[list][start] = motion.Ref[list][start+1]
				if isDirect(mb.SubMBType[i]) {
					fillMVDRectangle(&motion.MVD[list], start, 2, 2, 8, [2]uint8{})
					continue
				}
				if !isDir(mb.SubMBType[i], 0, list) {
					fillMVDRectangle(&motion.MVD[list], start, 2, 2, 8, [2]uint8{})
					fillMotionRectangle(&motion.MV[list], start, 2, 2, 8, [2]int16{})
					continue
				}
				blockWidth := 1
				if mb.SubMBType[i]&(MBType16x16|MBType16x8) != 0 {
					blockWidth = 2
				}
				for j := 0; j < int(mb.SubPartitionCount[i]); j++ {
					index := 4*i + blockWidth*j
					ref := motion.Ref[list][h264Scan8[index]]
					pred, err := predMotion(motion, index, blockWidth, list, ref)
					if err != nil {
						return err
					}
					mvd, mvda, err := decodeCABACMVDForPartition(src, motion, list, index)
					if err != nil {
						return err
					}
					mb.MVD[list][index] = mvd
					writeSubPartitionMVD(motion, list, index, mb.SubMBType[i], mvda)
					writeSubPartitionMV(motion, list, index, mb.SubMBType[i], addMVD(pred, mvd))
				}
			}
		}
		return nil
	}

	if is16x16(mb.MBType) {
		for list := 0; list < listCount; list++ {
			if !isDir(mb.MBType, 0, list) {
				continue
			}
			ref, err := decodeCABACRefForPartition(src, motion, sliceTypeNoS, list, 0, refCount[list])
			if err != nil {
				return err
			}
			mb.Ref[list][0] = ref
			fillRefRectangle(&motion.Ref[list], int(h264Scan8[0]), 4, 4, 8, int8(ref))
		}
		for list := 0; list < listCount; list++ {
			if !isDir(mb.MBType, 0, list) {
				continue
			}
			ref := motion.Ref[list][h264Scan8[0]]
			pred, err := predMotion(motion, 0, 4, list, ref)
			if err != nil {
				return err
			}
			mvd, mvda, err := decodeCABACMVDForPartition(src, motion, list, 0)
			if err != nil {
				return err
			}
			mb.MVD[list][0] = mvd
			fillMVDRectangle(&motion.MVD[list], int(h264Scan8[0]), 4, 4, 8, mvda)
			fillMotionRectangle(&motion.MV[list], int(h264Scan8[0]), 4, 4, 8, addMVD(pred, mvd))
		}
		return nil
	}

	if is16x8(mb.MBType) {
		for list := 0; list < listCount; list++ {
			for i := 0; i < 2; i++ {
				ref := int32(-1)
				if isDir(mb.MBType, i, list) {
					var err error
					ref, err = decodeCABACRefForPartition(src, motion, sliceTypeNoS, list, 8*i, refCount[list])
					if err != nil {
						return err
					}
				}
				mb.Ref[list][i] = ref
				fillRefRectangle(&motion.Ref[list], int(h264Scan8[0])+16*i, 4, 2, 8, int8(ref))
			}
		}
		for list := 0; list < listCount; list++ {
			for i := 0; i < 2; i++ {
				start := int(h264Scan8[0]) + 16*i
				if !isDir(mb.MBType, i, list) {
					fillMVDRectangle(&motion.MVD[list], start, 4, 2, 8, [2]uint8{})
					fillMotionRectangle(&motion.MV[list], start, 4, 2, 8, [2]int16{})
					continue
				}
				index := 8 * i
				ref := motion.Ref[list][start]
				pred, err := pred16x8Motion(motion, index, list, ref)
				if err != nil {
					return err
				}
				mvd, mvda, err := decodeCABACMVDForPartition(src, motion, list, index)
				if err != nil {
					return err
				}
				mb.MVD[list][index] = mvd
				fillMVDRectangle(&motion.MVD[list], start, 4, 2, 8, mvda)
				fillMotionRectangle(&motion.MV[list], start, 4, 2, 8, addMVD(pred, mvd))
			}
		}
		return nil
	}

	if is8x16(mb.MBType) {
		for list := 0; list < listCount; list++ {
			for i := 0; i < 2; i++ {
				ref := int32(-1)
				if isDir(mb.MBType, i, list) {
					var err error
					ref, err = decodeCABACRefForPartition(src, motion, sliceTypeNoS, list, 4*i, refCount[list])
					if err != nil {
						return err
					}
				}
				mb.Ref[list][i] = ref
				fillRefRectangle(&motion.Ref[list], int(h264Scan8[0])+2*i, 2, 4, 8, int8(ref))
			}
		}
		for list := 0; list < listCount; list++ {
			for i := 0; i < 2; i++ {
				start := int(h264Scan8[0]) + 2*i
				if !isDir(mb.MBType, i, list) {
					fillMVDRectangle(&motion.MVD[list], start, 2, 4, 8, [2]uint8{})
					fillMotionRectangle(&motion.MV[list], start, 2, 4, 8, [2]int16{})
					continue
				}
				index := 4 * i
				ref := motion.Ref[list][start]
				pred, err := pred8x16Motion(motion, index, list, ref)
				if err != nil {
					return err
				}
				mvd, mvda, err := decodeCABACMVDForPartition(src, motion, list, index)
				if err != nil {
					return err
				}
				mb.MVD[list][index] = mvd
				fillMVDRectangle(&motion.MVD[list], start, 2, 4, 8, mvda)
				fillMotionRectangle(&motion.MV[list], start, 2, 4, 8, addMVD(pred, mvd))
			}
		}
		return nil
	}
	return ErrUnsupported
}

func decodeCABACSubMBTypes(src cabacSyntaxSource, mb *cavlcInterMacroblockSyntax, sliceTypeNoS int32) error {
	for i := 0; i < 4; i++ {
		var info PMBInfo
		if sliceTypeNoS == PictureTypeB {
			_, info = decodeCABACBSubMBType(src)
		} else if sliceTypeNoS == PictureTypeP {
			_, info = decodeCABACPSubMBType(src)
		} else {
			return ErrInvalidData
		}
		mb.SubPartitionCount[i] = info.PartitionCount
		mb.SubMBType[i] = info.Type
	}
	return nil
}

func markDirectSubRefsUnavailable(motion *macroblockMotionCache) {
	if motion == nil {
		return
	}
	for list := 0; list < 2; list++ {
		motion.Ref[list][h264Scan8[4]] = h264PartNotAvailable
		motion.Ref[list][h264Scan8[12]] = h264PartNotAvailable
	}
}

func fillDirectCacheFromSubMBTypes(cache *[h264MotionCacheSize]uint8, sub *[4]uint32) {
	if cache == nil || sub == nil {
		return
	}
	for i := 0; i < 4; i++ {
		fillDirectRectangle(cache, int(h264Scan8[4*i]), 2, 2, 8, uint8(sub[i]>>1))
	}
}

func decodeCABACRefForPartition(src cabacSyntaxSource, motion *macroblockMotionCache, sliceTypeNoS int32, list int, n int, refTotal uint32) (int32, error) {
	if refTotal == 0 {
		return 0, ErrInvalidData
	}
	if refTotal == 1 {
		return 0, nil
	}
	index := int(h264Scan8[n])
	ref := decodeCABACMBRef(src, sliceTypeNoS, int32(motion.Ref[list][index-1]), int32(motion.Ref[list][index-8]), uint32(motion.Direct[index-1]), uint32(motion.Direct[index-8]))
	if ref < 0 || uint32(ref) >= refTotal {
		return 0, ErrInvalidData
	}
	return ref, nil
}

func decodeCABACMVDForPartition(src cabacSyntaxSource, motion *macroblockMotionCache, list int, n int) ([2]int32, [2]uint8, error) {
	var mvd [2]int32
	var mvda [2]uint8
	index := int(h264Scan8[n])
	x, ax, err := decodeCABACMBMVD(src, 40, int(motion.MVD[list][index-1][0])+int(motion.MVD[list][index-8][0]))
	if err != nil {
		return mvd, mvda, err
	}
	y, ay, err := decodeCABACMBMVD(src, 47, int(motion.MVD[list][index-1][1])+int(motion.MVD[list][index-8][1]))
	if err != nil {
		return mvd, mvda, err
	}
	mvd[0] = x
	mvd[1] = y
	mvda[0] = uint8(ax)
	mvda[1] = uint8(ay)
	return mvd, mvda, nil
}

func (c *cavlcResidualContext) decodeCABACResidualPayload(src cabacSyntaxSource, pps *PPS, sps *SPS, mbType uint32, cbp int, qscale int, lastQScaleDiff int, cacheResult residualDecodeCacheResult) (int, [2]uint8, int, int, error) {
	var chromaQP [2]uint8
	cbpTable := cbp
	if src == nil || pps == nil || sps == nil || qscale < 0 {
		return qscale, chromaQP, cbpTable, lastQScaleDiff, ErrInvalidData
	}

	if cbp != 0 || isIntra16x16(mbType) {
		maxQP := int(51 + 6*(sps.BitDepthLuma-8))
		var err error
		qscale, lastQScaleDiff, err = decodeCABACQScaleDiff(src, qscale, lastQScaleDiff, maxQP)
		if err != nil {
			return qscale, chromaQP, cbpTable, lastQScaleDiff, err
		}
		if qscale > qpMaxNum {
			return qscale, chromaQP, cbpTable, lastQScaleDiff, ErrInvalidData
		}
		chromaQP[0] = pps.ChromaQPTable[0][qscale]
		chromaQP[1] = pps.ChromaQPTable[1][qscale]

		if mbType&MBTypeInterlaced != 0 {
			return qscale, chromaQP, cbpTable, lastQScaleDiff, ErrUnsupported
		}

		scan, scan8x8 := h264CABACScansForQScale(sps, qscale)
		narrowDCT := sps.BitDepthLuma == 8
		ret, err := c.decodeCABACLumaResidualTyped(src, pps, scan, scan8x8, mbType, cbp, 0, qscale, cacheResult.LeftCBP, cacheResult.TopCBP, false, false, narrowDCT)
		if err != nil {
			return qscale, chromaQP, cbpTable, lastQScaleDiff, err
		}
		cbpTable |= ret
		if sps.ChromaFormatIDC == 3 {
			ret, err := c.decodeCABACLumaResidualTyped(src, pps, scan, scan8x8, mbType, cbp, 1, int(chromaQP[0]), cacheResult.LeftCBP, cacheResult.TopCBP, false, true, narrowDCT)
			if err != nil {
				return qscale, chromaQP, cbpTable, lastQScaleDiff, err
			}
			cbpTable |= ret
			ret, err = c.decodeCABACLumaResidualTyped(src, pps, scan, scan8x8, mbType, cbp, 2, int(chromaQP[1]), cacheResult.LeftCBP, cacheResult.TopCBP, false, true, narrowDCT)
			if err != nil {
				return qscale, chromaQP, cbpTable, lastQScaleDiff, err
			}
			cbpTable |= ret
		} else {
			ret, err := c.decodeCABACChromaResidualTyped(src, pps, scan, mbType, cbp, int32(sps.ChromaFormatIDC), chromaQP, cacheResult.LeftCBP, cacheResult.TopCBP, false, narrowDCT)
			if err != nil {
				return qscale, chromaQP, cbpTable, lastQScaleDiff, err
			}
			cbpTable |= ret
		}
		return qscale, chromaQP, cbpTable, lastQScaleDiff, nil
	}

	clearCAVLCResidualCaches(c)
	return qscale, chromaQP, cbpTable, 0, nil
}

func validateCABACFrameIntra4x4PredModes(cache *[h264IntraPredModeCacheSize]int8, cacheResult intraPredDecodeCacheResult) error {
	if cache == nil {
		return ErrInvalidData
	}
	return checkIntra4x4PredModeCache(cache, cacheResult.TopSamplesAvailable, cacheResult.LeftSamplesAvailable)
}

func fillIntraPredModeRectangle(cache *[h264IntraPredModeCacheSize]int8, start int, width int, height int, stride int, value int8) {
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			cache[start+y*stride+x] = value
		}
	}
}

func cabacNeighborTransformSize(n macroblockDecodeNeighbors) int {
	return boolToInt(is8x8DCT(n.TopType)) + boolToInt(is8x8DCT(n.LeftType[h264LeftTop]))
}
