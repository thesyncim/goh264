// SPDX-License-Identifier: LGPL-2.1-or-later
//
// Source-shaped frame-MB CABAC macroblock handoff from FFmpeg n8.0.1
// libavcodec/h264_cabac.c ff_h264_decode_mb_cabac. This layer connects the
// translated CABAC syntax, residual, motion-cache, and state write-back pieces
// while still stopping before reconstruction/deblocking.

package h264

import "fmt"

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
	FrameMBAFF             bool
	FieldPicture           bool
	Direct                 h264DirectMotionContext
	PPS                    *PPS
	SPS                    *SPS
	X264Build              int32
	X264BuildSet           bool
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

type h264X264BuildInfo struct {
	Build int32
	Set   bool
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
	return m.decodeCABACFrameSliceMacroblockWithDirectWorkGuardX264(src, sh, state, mbXY, sliceNum, direct, work, h264X264BuildInfo{}, rejectUnsupportedHighB)
}

func (m *macroblockTables) decodeCABACFrameSliceMacroblockWithDirectWorkGuardX264(src cabacSyntaxSource, sh *SliceHeader, state *cabacFrameSliceState, mbXY int, sliceNum uint16, direct h264DirectMotionContext, work *frameMacroblockDecodeWork, x264 h264X264BuildInfo, rejectUnsupportedHighB bool) (cabacFrameMacroblockResult, error) {
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
	return m.decodeCABACFrameSliceMacroblockWithDirectWorkGuardX264Validated(src, sh, state, mbXY, sliceNum, direct, work, x264, rejectUnsupportedHighB)
}

func (m *macroblockTables) decodeCABACFrameSliceMacroblockWithDirectWorkGuardX264Validated(src cabacSyntaxSource, sh *SliceHeader, state *cabacFrameSliceState, mbXY int, sliceNum uint16, direct h264DirectMotionContext, work *frameMacroblockDecodeWork, x264 h264X264BuildInfo, rejectUnsupportedHighB bool) (cabacFrameMacroblockResult, error) {
	var result cabacFrameMacroblockResult
	frameMBAFF := sh.PictureStructure == PictureFrame && sh.SPS.MBAFF != 0
	mbX := mbXY % m.MBStride
	mbY := mbXY / m.MBStride

	if sh.SliceTypeNoS != PictureTypeI {
		var skip bool
		var err error
		if frameMBAFF && (mbY&1) != 0 && state.PrevMBSkipped {
			skip = state.NextMBSkipped
		} else if frameMBAFF {
			skip, err = m.decodeCABACMBSkipMBAFF(src, mbXY, mbX, mbY, sh.SliceTypeNoS, sliceNum, state.MBFieldDecodingFlag)
		} else {
			skip, err = m.decodeCABACMBSkip(src, mbXY, sh.SliceTypeNoS, sliceNum, sh.PictureStructure != PictureFrame)
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
				}
			}
			result, err := m.writeBackCABACFrameSkipMacroblockWithDirectWorkFieldGuard(sh, state.QScale, mbXY, sliceNum, state.MBFieldDecodingFlag, direct, work, rejectUnsupportedHighB)
			if err != nil {
				return result, fmt.Errorf("skip field=%d: %w", state.MBFieldDecodingFlag, err)
			}
			return result, nil
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
		}
	}

	refCount := sh.RefCount
	if frameMBAFF && state.MBFieldDecodingFlag != 0 {
		refCount = h264MBAFFFieldRefCount(refCount)
	}
	result, err := m.decodeCABACFrameMacroblockWithWorkValidated(src, cabacFrameMacroblockInput{
		MBXY:                   mbXY,
		SliceNum:               sliceNum,
		SliceType:              sh.SliceType,
		SliceTypeNoS:           sh.SliceTypeNoS,
		QScale:                 state.QScale,
		LastQScaleDiff:         state.LastQScaleDiff,
		MBFieldDecodingFlag:    state.MBFieldDecodingFlag,
		RefCount:               refCount,
		DCT8x8Allowed:          sh.PPS.Transform8x8Mode != 0,
		DirectSpatialMVPred:    sh.DirectSpatialMVPred != 0,
		DeblockingFilter:       sh.DeblockingFilter,
		FrameMBAFF:             frameMBAFF,
		FieldPicture:           sh.PictureStructure != PictureFrame || state.MBFieldDecodingFlag != 0,
		Direct:                 direct,
		PPS:                    sh.PPS,
		SPS:                    sh.SPS,
		X264Build:              x264.Build,
		X264BuildSet:           x264.Set,
		RejectUnsupportedHighB: rejectUnsupportedHighB,
	}, work)
	if err != nil {
		return result, fmt.Errorf("field=%d refs=%d/%d: %w", state.MBFieldDecodingFlag, refCount[0], refCount[1], err)
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
	return m.decodeCABACFrameMacroblockWithWorkValidated(src, in, work)
}

func (m *macroblockTables) decodeCABACFrameMacroblockWithWorkValidated(src cabacSyntaxSource, in cabacFrameMacroblockInput, work *frameMacroblockDecodeWork) (cabacFrameMacroblockResult, error) {
	var result cabacFrameMacroblockResult
	result.MBFieldDecodingFlag = in.MBFieldDecodingFlag
	work.resetForDecode()

	fieldPicture := in.FieldPicture || in.MBFieldDecodingFlag != 0
	neighborMBType := uint32(0)
	if in.FrameMBAFF && fieldPicture {
		neighborMBType = MBTypeInterlaced
	}
	neighbors, err := m.fillDecodeNeighborsFrameEntropy(in.MBXY, in.SliceNum, neighborMBType, fieldPicture, in.FrameMBAFF)
	if err != nil {
		return result, fmt.Errorf("neighbors field=%t: %w", fieldPicture, err)
	}
	base, err := decodeCABACMBTypeForSource(src, in.SliceType, in.SliceTypeNoS, neighbors.LeftType[h264LeftTop], neighbors.TopType)
	if err != nil {
		return result, fmt.Errorf("mb_type field=%t left=%#x top=%#x: %w", fieldPicture, neighbors.LeftType[h264LeftTop], neighbors.TopType, err)
	}
	if fieldPicture {
		base.MBType |= MBTypeInterlaced
	}
	result.MBType = base.MBType
	if in.RejectUnsupportedHighB {
		if err := validateHighFrameSliceBaseMacroblockForDecode(in.SliceTypeNoS, base.MBType); err != nil {
			return result, fmt.Errorf("validate_base field=%t type=%#x: %w", fieldPicture, base.MBType, err)
		}
	}
	if base.MBType&MBTypeIntraPCM != 0 {
		return m.decodeCABACFrameIntraPCMMacroblock(src, in, base, result)
	}

	listCount, err := cavlcFrameListCount(in.SliceTypeNoS)
	if err != nil {
		return result, fmt.Errorf("list_count type=%d: %w", in.SliceTypeNoS, err)
	}

	cacheResult, err := m.fillFrameMacroblockDecodeCachesEntropy(&work.IntraCache, &work.Residual, &work.Motion, frameMacroblockDecodeCacheInput{
		MBXY:                 in.MBXY,
		SliceNum:             in.SliceNum,
		MBType:               base.MBType,
		ListCount:            listCount,
		SliceTypeNoS:         in.SliceTypeNoS,
		CABAC:                true,
		FieldPicture:         fieldPicture,
		ConstrainedIntraPred: in.PPS.ConstrainedIntraPred != 0,
		DirectSpatialMVPred:  in.DirectSpatialMVPred,
	}, in.FrameMBAFF)
	if err != nil {
		return result, fmt.Errorf("caches field=%t type=%#x left=%#x top=%#x: %w", fieldPicture, base.MBType, neighbors.LeftType[h264LeftTop], neighbors.TopType, err)
	}
	result.Neighbors = cacheResult.Neighbors

	if isIntra(base.MBType) {
		result, err := m.decodeCABACFrameIntraMacroblock(src, in, base, &work.Residual, &work.IntraCache, cacheResult, result)
		if err != nil {
			return result, fmt.Errorf("intra field=%t type=%#x cbp=%#x: %w", fieldPicture, base.MBType, base.CBP, err)
		}
		return result, nil
	}
	result, err = m.decodeCABACFrameInterMacroblock(src, in, base, &work.Residual, &work.Motion, listCount, cacheResult, result)
	if err != nil {
		return result, fmt.Errorf("inter field=%t type=%#x refs=%d/%d: %w", fieldPicture, base.MBType, in.RefCount[0], in.RefCount[1], err)
	}
	return result, nil
}

func (m *macroblockTables) decodeCABACFrameIntraPCMMacroblock(src cabacSyntaxSource, in cabacFrameMacroblockInput, base cavlcMacroblockSyntax, result cabacFrameMacroblockResult) (cabacFrameMacroblockResult, error) {
	pcm, err := readCABACIntraPCMBytes(src, in.SPS)
	if err != nil {
		return result, err
	}
	base.IntraPCM = pcm
	base.QScale = 0
	base.CBPTable = 0xf7ef
	if err := m.writeBackCABACIntraPCMMacroblock(in.MBXY, base.MBType, in.SliceNum); err != nil {
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

func (m *macroblockTables) decodeCABACMBSkip(src cabacSyntaxSource, mbXY int, sliceTypeNoS int32, sliceNum uint16, fieldPicture bool) (bool, error) {
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
	topStride := m.MBStride
	if fieldPicture {
		topStride <<= 1
	}
	topXY := mbXY - topStride
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
	if dec, ok := src.(*cabacSyntaxDecoder); ok {
		return dec.get(11+ctx) != 0, nil
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
	if dec, ok := src.(*cabacSyntaxDecoder); ok {
		return dec.get(11+ctx) != 0, nil
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
	if dec, ok := src.(*cabacSyntaxDecoder); ok {
		return int32(dec.get(70 + ctx)), nil
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
	return m.writeBackCABACFrameSkipMacroblockWithDirectWorkFieldGuard(sh, qscale, mbXY, sliceNum, 0, direct, work, rejectUnsupportedHighB)
}

func (m *macroblockTables) writeBackCABACFrameSkipMacroblockWithDirectWorkFieldGuard(sh *SliceHeader, qscale int, mbXY int, sliceNum uint16, mbFieldDecodingFlag int32, direct h264DirectMotionContext, work *frameMacroblockDecodeWork, rejectUnsupportedHighB bool) (cabacFrameMacroblockResult, error) {
	var result cabacFrameMacroblockResult
	if sh == nil || work == nil {
		return result, ErrInvalidData
	}
	if qscale < 0 || qscale > qpMaxNum {
		return result, ErrInvalidData
	}
	if sh.SliceTypeNoS == PictureTypeB {
		return m.writeBackCABACFrameBSkipMacroblockWithDirectWorkFieldGuard(sh, qscale, mbXY, sliceNum, mbFieldDecodingFlag, direct, work, rejectUnsupportedHighB)
	}
	if sh.SliceTypeNoS != PictureTypeP {
		return result, ErrUnsupported
	}

	mbType := MBType16x16 | MBTypeP0L0 | MBTypeP1L0 | MBTypeSkip
	fieldPicture := sh.PictureStructure != PictureFrame || mbFieldDecodingFlag != 0
	if fieldPicture {
		mbType |= MBTypeInterlaced
	}
	frameMBAFF := sh.PictureStructure == PictureFrame && sh.SPS != nil && sh.SPS.MBAFF != 0
	neighbors, err := m.fillDecodeNeighborsFrameEntropy(mbXY, sliceNum, mbType, fieldPicture, frameMBAFF)
	if err != nil {
		return result, err
	}
	work.resetForSkip()
	motionNeighbors := neighbors.motionNeighbors(mbType, 1, PictureTypeP, true, false)
	motionNeighbors.FrameMBAFF = frameMBAFF
	if err := m.writeBackCABACPskipMacroblockWithMotion(mbXY, qscale, motionNeighbors, sliceNum, &work.Motion); err != nil {
		return result, err
	}
	if fieldPicture {
		m.MacroblockTyp[mbXY] |= MBTypeInterlaced
	}

	result.MBType = mbType
	result.MBFieldDecodingFlag = mbFieldDecodingFlag
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
	return m.writeBackCABACFrameBSkipMacroblockWithDirectWorkFieldGuard(sh, qscale, mbXY, sliceNum, 0, direct, work, rejectUnsupportedHighB)
}

func (m *macroblockTables) writeBackCABACFrameBSkipMacroblockWithDirectWorkFieldGuard(sh *SliceHeader, qscale int, mbXY int, sliceNum uint16, mbFieldDecodingFlag int32, direct h264DirectMotionContext, work *frameMacroblockDecodeWork, rejectUnsupportedHighB bool) (cabacFrameMacroblockResult, error) {
	var result cabacFrameMacroblockResult
	if sh == nil || work == nil {
		return result, ErrInvalidData
	}
	mbType := MBTypeL0L1 | MBTypeDirect2 | MBTypeSkip
	fieldPicture := sh.PictureStructure != PictureFrame || mbFieldDecodingFlag != 0
	if fieldPicture {
		mbType |= MBTypeInterlaced
	}
	work.resetForSkip()
	frameMBAFF := sh.PictureStructure == PictureFrame && sh.SPS != nil && sh.SPS.MBAFF != 0
	var neighbors macroblockDecodeNeighbors
	var err error
	if direct.DirectSpatialMVPred {
		cacheResult, err := m.fillFrameMacroblockDecodeCachesEntropy(&work.IntraCache, &work.Residual, &work.Motion, frameMacroblockDecodeCacheInput{
			MBXY:                mbXY,
			SliceNum:            sliceNum,
			MBType:              mbType,
			ListCount:           2,
			SliceTypeNoS:        PictureTypeB,
			CABAC:               true,
			FieldPicture:        fieldPicture,
			DirectSpatialMVPred: true,
		}, frameMBAFF)
		if err != nil {
			return result, err
		}
		neighbors = cacheResult.Neighbors
	} else {
		neighbors, err = m.fillDecodeNeighborsFrameEntropy(mbXY, sliceNum, mbType, fieldPicture, frameMBAFF)
		if err != nil {
			return result, err
		}
	}
	var subMBType [4]uint32
	if err := m.predDirectMotionFrame(&work.Motion, mbXY, &mbType, &subMBType, direct); err != nil {
		return result, fmt.Errorf("bskip direct field=%t type=%#x: %w", fieldPicture, mbType, err)
	}
	if rejectUnsupportedHighB {
		if err := validateHighFrameSliceMacroblockForReconstructWithSubMB(sh, mbType, &subMBType, 0, 0); err != nil {
			return result, fmt.Errorf("validate_bskip type=%#x sub=%#x: %w", mbType, subMBType, err)
		}
	}
	if err := m.writeBackBskipMacroblockWithMotion(mbXY, qscale, mbType, true, &subMBType, sliceNum, &work.Motion); err != nil {
		return result, err
	}

	result.MBType = mbType
	result.MBFieldDecodingFlag = mbFieldDecodingFlag
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
		if in.DCT8x8Allowed {
			transform8x8 := 0
			if dec, ok := src.(*cabacSyntaxDecoder); ok {
				transform8x8 = dec.get(399 + cacheResult.Intra.NeighborTransformSize)
			} else {
				transform8x8 = src.get(399 + cacheResult.Intra.NeighborTransformSize)
			}
			if transform8x8 != 0 {
				mb.MBType |= MBType8x8DCT
				mb.TransformSize8x8DCT = true
				di = 4
			}
		}
		for i := 0; i < 16; i += di {
			pred, err := predIntraMode(intraCache, i)
			if err != nil {
				return result, err
			}
			mode := decodeCABACMBIntra4x4PredModeForSource(src, int(pred))
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
		mb.CBP = decodeCABACMBCBPLumaForSource(src, cacheResult.Residual.LeftCBP, cacheResult.Residual.TopCBP)
		if in.SPS.ChromaFormatIDC == 1 || in.SPS.ChromaFormatIDC == 2 {
			mb.CBP |= decodeCABACMBCBPChromaForSource(src, cacheResult.Residual.LeftCBP, cacheResult.Residual.TopCBP) << 4
		}
	} else if (in.SPS.ChromaFormatIDC != 1 && in.SPS.ChromaFormatIDC != 2) && mb.CBP > 15 {
		return result, ErrInvalidData
	}

	adjustCABACChroma444DCT8x8NonZeroCache(residual, in.SPS.ChromaFormatIDC, mb.MBType, cacheResult.Neighbors, in.X264Build, in.X264BuildSet)
	qscale, chromaQP, cbpTable, lastDiff, err := residual.decodeCABACResidualPayload(src, in.PPS, in.SPS, mb.MBType, mb.CBP, in.QScale, in.LastQScaleDiff, cacheResult.Residual)
	if err != nil {
		return result, fmt.Errorf("intra_residual field=%t type=%#x cbp=%#x: %w", in.FieldPicture, mb.MBType, mb.CBP, err)
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
	dct8x8Allowed := in.DCT8x8Allowed
	if isDirect(base.MBType) {
		if err := m.predDirectMotionFrame(motion, in.MBXY, &mb.MBType, &mb.SubMBType, in.Direct); err != nil {
			return result, err
		}
		fillMVDRectangle(&motion.MVD[0], int(h264Scan8[0]), 4, 4, 8, [2]uint8{})
		fillMVDRectangle(&motion.MVD[1], int(h264Scan8[0]), 4, 4, 8, [2]uint8{})
		dct8x8Allowed = dct8x8Allowed && in.SPS.Direct8x8InferenceFlag != 0
	} else {
		predCtx := m.frameMotionPredContext(in.MBXY, in.FrameMBAFF, cacheResult.Neighbors, base.MBType, listCount, in.SliceTypeNoS, true, in.DirectSpatialMVPred)
		if err := m.decodeCABACInterMotionSyntax(src, &mb, motion, in.MBXY, in.SliceTypeNoS, listCount, in.RefCount, in.Direct, predCtx); err != nil {
			return result, fmt.Errorf("inter_motion field=%t type=%#x parts=%d sub=%#x refs=%d/%d: %w",
				in.FieldPicture, base.MBType, base.PartitionCount, mb.SubMBType, in.RefCount[0], in.RefCount[1], err)
		}
	}

	mb.CBP = decodeCABACMBCBPLumaForSource(src, cacheResult.Residual.LeftCBP, cacheResult.Residual.TopCBP)
	if in.SPS.ChromaFormatIDC == 1 || in.SPS.ChromaFormatIDC == 2 {
		mb.CBP |= decodeCABACMBCBPChromaForSource(src, cacheResult.Residual.LeftCBP, cacheResult.Residual.TopCBP) << 4
	}
	if mb.PartitionCount == 4 {
		dct8x8Allowed = subMBTypesAllowDCT8x8(dct8x8Allowed, &mb.SubMBType, in.SPS.Direct8x8InferenceFlag != 0)
	}
	if dct8x8Allowed && (mb.CBP&15) != 0 {
		transform8x8 := 0
		if dec, ok := src.(*cabacSyntaxDecoder); ok {
			transform8x8 = dec.get(399 + cabacNeighborTransformSize(cacheResult.Neighbors))
		} else {
			transform8x8 = src.get(399 + cabacNeighborTransformSize(cacheResult.Neighbors))
		}
		if transform8x8 != 0 {
			mb.MBType |= MBType8x8DCT
			mb.TransformSize8x8DCT = true
		}
	}

	adjustCABACChroma444DCT8x8NonZeroCache(residual, in.SPS.ChromaFormatIDC, mb.MBType, cacheResult.Neighbors, in.X264Build, in.X264BuildSet)
	qscale, chromaQP, cbpTable, lastDiff, err := residual.decodeCABACResidualPayload(src, in.PPS, in.SPS, mb.MBType, mb.CBP, in.QScale, in.LastQScaleDiff, cacheResult.Residual)
	if err != nil {
		return result, fmt.Errorf("inter_residual field=%t type=%#x cbp=%#x: %w", in.FieldPicture, mb.MBType, mb.CBP, err)
	}
	mb.QScale = qscale
	mb.ChromaQP = chromaQP
	mb.CBPTable = cbpTable
	if in.RejectUnsupportedHighB {
		pictureStructure := int32(PictureFrame)
		if in.FieldPicture {
			pictureStructure = PictureTopField
		}
		sh := &SliceHeader{
			SliceTypeNoS:     in.SliceTypeNoS,
			PictureStructure: pictureStructure,
			DeblockingFilter: in.DeblockingFilter,
			PPS:              in.PPS,
			SPS:              in.SPS,
		}
		if err := validateHighFrameSliceMacroblockForReconstructWithSubMB(sh, mb.MBType, &mb.SubMBType, mb.CBP, mb.CBPTable); err != nil {
			return result, fmt.Errorf("validate_inter type=%#x cbp=%#x table=%#x sub=%#x: %w",
				mb.MBType, mb.CBP, mb.CBPTable, mb.SubMBType, err)
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
	if dec, ok := src.(*cabacSyntaxDecoder); ok {
		return decodeCABACMBChromaPredModeSource(dec, ctx)
	}
	return decodeCABACMBChromaPredModeSource(src, ctx)
}

func decodeCABACMBChromaPredModeSource[S cabacSyntaxSource](src S, ctx int) int {
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

func (m *macroblockTables) decodeCABACInterMotionSyntax(src cabacSyntaxSource, mb *cavlcInterMacroblockSyntax, motion *macroblockMotionCache, mbXY int, sliceTypeNoS int32, listCount int, refCount [2]uint32, direct h264DirectMotionContext, predCtx *h264MotionPredContext) error {
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
				return fmt.Errorf("direct_sub type=%#x sub=%#x: %w", mb.MBType, mb.SubMBType, err)
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
						return fmt.Errorf("ref list=%d sub=%d n=%d total=%d: %w", list, i, 4*i, refCount[list], err)
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
					pred, err := predMotionWithContext(motion, index, blockWidth, list, ref, predCtx)
					if err != nil {
						return fmt.Errorf("pred list=%d sub=%d n=%d width=%d ref=%d: %w", list, i, index, blockWidth, ref, err)
					}
					mvd, mvda, err := decodeCABACMVDForPartition(src, motion, list, index)
					if err != nil {
						return fmt.Errorf("mvd list=%d sub=%d n=%d: %w", list, i, index, err)
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
				return fmt.Errorf("ref list=%d n=0 total=%d: %w", list, refCount[list], err)
			}
			mb.Ref[list][0] = ref
			fillRefRectangle(&motion.Ref[list], int(h264Scan8[0]), 4, 4, 8, int8(ref))
		}
		for list := 0; list < listCount; list++ {
			if !isDir(mb.MBType, 0, list) {
				continue
			}
			ref := motion.Ref[list][h264Scan8[0]]
			pred, err := predMotionWithContext(motion, 0, 4, list, ref, predCtx)
			if err != nil {
				return fmt.Errorf("pred list=%d n=0 width=4 ref=%d: %w", list, ref, err)
			}
			mvd, mvda, err := decodeCABACMVDForPartition(src, motion, list, 0)
			if err != nil {
				return fmt.Errorf("mvd list=%d n=0: %w", list, err)
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
						return fmt.Errorf("ref list=%d part=%d n=%d total=%d: %w", list, i, 8*i, refCount[list], err)
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
				pred, err := pred16x8MotionWithContext(motion, index, list, ref, predCtx)
				if err != nil {
					return fmt.Errorf("pred16x8 list=%d part=%d n=%d ref=%d: %w", list, i, index, ref, err)
				}
				mvd, mvda, err := decodeCABACMVDForPartition(src, motion, list, index)
				if err != nil {
					return fmt.Errorf("mvd list=%d part=%d n=%d: %w", list, i, index, err)
				}
				mb.MVD[list][index] = mvd
				fillMVDRectangle(&motion.MVD[list], start, 4, 2, 8, mvda)
				mv := addMVD(pred, mvd)
				fillMotionRectangle(&motion.MV[list], start, 4, 2, 8, mv)
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
						return fmt.Errorf("ref list=%d part=%d n=%d total=%d: %w", list, i, 4*i, refCount[list], err)
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
				pred, err := pred8x16MotionWithContext(motion, index, list, ref, predCtx)
				if err != nil {
					return fmt.Errorf("pred8x16 list=%d part=%d n=%d ref=%d: %w", list, i, index, ref, err)
				}
				mvd, mvda, err := decodeCABACMVDForPartition(src, motion, list, index)
				if err != nil {
					return fmt.Errorf("mvd list=%d part=%d n=%d: %w", list, i, index, err)
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
		return 0, fmt.Errorf("zero refs: %w", ErrInvalidData)
	}
	if refTotal == 1 {
		return 0, nil
	}
	index := int(h264Scan8[n])
	refA := motion.Ref[list][index-1]
	refB := motion.Ref[list][index-8]
	directA := motion.Direct[index-1]
	directB := motion.Direct[index-8]
	ref := decodeCABACMBRefForSource(src, sliceTypeNoS, int32(refA), int32(refB), uint32(directA), uint32(directB))
	if ref < 0 || uint32(ref) >= refTotal {
		cabacState := ""
		if dec, ok := src.(*cabacSyntaxDecoder); ok && dec.cabac != nil {
			cabacState = fmt.Sprintf(" low=%d range=%d bytestream=%d", dec.cabac.low, dec.cabac.rng, dec.cabac.bytestream)
		}
		return 0, fmt.Errorf("decoded ref=%d refa=%d refb=%d dira=%d dirb=%d%s: %w", ref, refA, refB, directA, directB, cabacState, ErrInvalidData)
	}
	return ref, nil
}

func decodeCABACMVDForPartition(src cabacSyntaxSource, motion *macroblockMotionCache, list int, n int) ([2]int32, [2]uint8, error) {
	var mvd [2]int32
	var mvda [2]uint8
	index := int(h264Scan8[n])
	dec, direct := src.(*cabacSyntaxDecoder)
	var x int32
	var ax int
	var err error
	if direct {
		x, ax, err = decodeCABACMBMVD(dec, 40, int(motion.MVD[list][index-1][0])+int(motion.MVD[list][index-8][0]))
	} else {
		x, ax, err = decodeCABACMBMVD(src, 40, int(motion.MVD[list][index-1][0])+int(motion.MVD[list][index-8][0]))
	}
	if err != nil {
		return mvd, mvda, err
	}
	var y int32
	var ay int
	if direct {
		y, ay, err = decodeCABACMBMVD(dec, 47, int(motion.MVD[list][index-1][1])+int(motion.MVD[list][index-8][1]))
	} else {
		y, ay, err = decodeCABACMBMVD(src, 47, int(motion.MVD[list][index-1][1])+int(motion.MVD[list][index-8][1]))
	}
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
		clear(c.MB[:])
	}

	if cbp != 0 || isIntra16x16(mbType) {
		maxQP := int(51 + 6*(sps.BitDepthLuma-8))
		var err error
		qscale, lastQScaleDiff, err = decodeCABACQScaleDiffForSource(src, qscale, lastQScaleDiff, maxQP)
		if err != nil {
			return qscale, chromaQP, cbpTable, lastQScaleDiff, fmt.Errorf("qscale diff: %w", err)
		}
		if qscale > qpMaxNum {
			return qscale, chromaQP, cbpTable, lastQScaleDiff, fmt.Errorf("qscale=%d max=%d: %w", qscale, qpMaxNum, ErrInvalidData)
		}
		chromaQP[0] = pps.ChromaQPTable[0][qscale]
		chromaQP[1] = pps.ChromaQPTable[1][qscale]

		mbField := mbType&MBTypeInterlaced != 0
		scan, scan8x8 := h264CABACScansForQScale(sps, qscale, mbField)
		narrowDCT := sps.BitDepthLuma == 8
		ret, err := c.decodeCABACLumaResidualTyped(src, pps, scan, scan8x8, mbType, cbp, 0, qscale, cacheResult.LeftCBP, cacheResult.TopCBP, mbField, sps.ChromaFormatIDC == 3, narrowDCT)
		if err != nil {
			return qscale, chromaQP, cbpTable, lastQScaleDiff, fmt.Errorf("luma residual p=0: %w", err)
		}
		cbpTable |= ret
		if sps.ChromaFormatIDC == 3 {
			ret, err := c.decodeCABACLumaResidualTyped(src, pps, scan, scan8x8, mbType, cbp, 1, int(chromaQP[0]), cacheResult.LeftCBP, cacheResult.TopCBP, mbField, true, narrowDCT)
			if err != nil {
				return qscale, chromaQP, cbpTable, lastQScaleDiff, fmt.Errorf("luma residual p=1: %w", err)
			}
			cbpTable |= ret
			ret, err = c.decodeCABACLumaResidualTyped(src, pps, scan, scan8x8, mbType, cbp, 2, int(chromaQP[1]), cacheResult.LeftCBP, cacheResult.TopCBP, mbField, true, narrowDCT)
			if err != nil {
				return qscale, chromaQP, cbpTable, lastQScaleDiff, fmt.Errorf("luma residual p=2: %w", err)
			}
			cbpTable |= ret
		} else {
			ret, err := c.decodeCABACChromaResidualTyped(src, pps, scan, mbType, cbp, int32(sps.ChromaFormatIDC), chromaQP, cacheResult.LeftCBP, cacheResult.TopCBP, mbField, narrowDCT)
			if err != nil {
				return qscale, chromaQP, cbpTable, lastQScaleDiff, fmt.Errorf("chroma residual: %w", err)
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

func adjustCABACChroma444DCT8x8NonZeroCache(c *cavlcResidualContext, chromaFormatIDC uint32, mbType uint32, n macroblockDecodeNeighbors, x264Build int32, x264BuildSet bool) {
	if c == nil || chromaFormatIDC != 3 || !is8x8DCT(mbType) {
		return
	}
	oldX264Build := h264X264BuildUsesUnfiltered8x8LAdd(x264Build, x264BuildSet)
	for i := 0; i < 2; i++ {
		leftType := n.LeftType[i]
		if leftType == 0 || is8x8DCT(leftType) {
			continue
		}
		value := uint8(0)
		if oldX264Build {
			if isIntra(mbType) {
				value = 64
			}
		} else if leftType&MBTypeIntraPCM != 0 {
			value = 64
		}
		c.NonZeroCountCache[3+8*1+2*8*i] = value
		c.NonZeroCountCache[3+8*2+2*8*i] = value
		c.NonZeroCountCache[3+8*6+2*8*i] = value
		c.NonZeroCountCache[3+8*7+2*8*i] = value
		c.NonZeroCountCache[3+8*11+2*8*i] = value
		c.NonZeroCountCache[3+8*12+2*8*i] = value
	}
	if n.TopType == 0 || is8x8DCT(n.TopType) {
		return
	}
	value := uint8(0)
	if oldX264Build {
		if isIntra(mbType) {
			value = 64
		}
	} else if n.TopType&MBTypeIntraPCM != 0 {
		value = 64
	}
	fillCAVLCNonZero(&c.NonZeroCountCache, 4+8*0, 4, 1, 8, value)
	fillCAVLCNonZero(&c.NonZeroCountCache, 4+8*5, 4, 1, 8, value)
	fillCAVLCNonZero(&c.NonZeroCountCache, 4+8*10, 4, 1, 8, value)
}
