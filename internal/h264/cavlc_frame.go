// SPDX-License-Identifier: LGPL-2.1-or-later
//
// Source-shaped frame-MB CAVLC macroblock handoff from FFmpeg n8.0.1
// libavcodec/h264_cavlc.c ff_h264_decode_mb_cavlc. This layer deliberately
// stops at entropy-to-state write-back; reconstruction and deblocking remain
// separate decoder steps.

package h264

type cavlcFrameMacroblockInput struct {
	MBXY                   int
	SliceNum               uint16
	SliceType              int32
	SliceTypeNoS           int32
	QScale                 int
	RefCount               [2]uint32
	DCT8x8Allowed          bool
	DirectSpatialMVPred    bool
	Direct                 h264DirectMotionContext
	PPS                    *PPS
	SPS                    *SPS
	RejectUnsupportedHighB bool
}

type cavlcFrameMacroblockResult struct {
	MBType            uint32
	CBP               int
	CBPTable          int
	QScale            int
	ChromaQP          [2]uint8
	ChromaPred        int32
	TopLeftAvailable  uint16
	TopRightAvailable uint16
	Neighbors         macroblockDecodeNeighbors
	Intra             cavlcMacroblockSyntax
	Inter             cavlcInterMacroblockSyntax
	IntraPCM          []byte
	IsIntra           bool
	IsInter           bool
	Skipped           bool
}

const cavlcMBSkipRunUnset int32 = -1

type cavlcFrameSliceState struct {
	MBSkipRun int32
	QScale    int
}

func newCAVLCFrameSliceState(qscale int) cavlcFrameSliceState {
	return cavlcFrameSliceState{MBSkipRun: cavlcMBSkipRunUnset, QScale: qscale}
}

func (m *macroblockTables) decodeCAVLCFrameSliceMacroblock(gb *bitReader, sh *SliceHeader, state *cavlcFrameSliceState, mbXY int, sliceNum uint16) (cavlcFrameMacroblockResult, error) {
	var work frameMacroblockDecodeWork
	return m.decodeCAVLCFrameSliceMacroblockWithWork(gb, sh, state, mbXY, sliceNum, &work)
}

func (m *macroblockTables) decodeCAVLCFrameSliceMacroblockWithWork(gb *bitReader, sh *SliceHeader, state *cavlcFrameSliceState, mbXY int, sliceNum uint16, work *frameMacroblockDecodeWork) (cavlcFrameMacroblockResult, error) {
	return m.decodeCAVLCFrameSliceMacroblockWithDirectWork(gb, sh, state, mbXY, sliceNum, h264DirectMotionContext{}, work)
}

func (m *macroblockTables) decodeCAVLCFrameSliceMacroblockWithDirectWork(gb *bitReader, sh *SliceHeader, state *cavlcFrameSliceState, mbXY int, sliceNum uint16, direct h264DirectMotionContext, work *frameMacroblockDecodeWork) (cavlcFrameMacroblockResult, error) {
	return m.decodeCAVLCFrameSliceMacroblockWithDirectWorkGuard(gb, sh, state, mbXY, sliceNum, direct, work, false)
}

func (m *macroblockTables) decodeCAVLCFrameSliceMacroblockWithDirectWorkGuard(gb *bitReader, sh *SliceHeader, state *cavlcFrameSliceState, mbXY int, sliceNum uint16, direct h264DirectMotionContext, work *frameMacroblockDecodeWork, rejectUnsupportedHighB bool) (cavlcFrameMacroblockResult, error) {
	var result cavlcFrameMacroblockResult
	if m == nil || gb == nil || sh == nil || sh.PPS == nil || sh.SPS == nil || state == nil || work == nil {
		return result, ErrInvalidData
	}
	if sh.PictureStructure != PictureFrame || sh.SPS.MBAFF != 0 {
		return result, ErrUnsupported
	}
	if sh.QScale > qpMaxNum || state.MBSkipRun < cavlcMBSkipRunUnset || state.QScale < 0 || state.QScale > qpMaxNum {
		return result, ErrInvalidData
	}

	if sh.SliceTypeNoS != PictureTypeI {
		if state.MBSkipRun == cavlcMBSkipRunUnset {
			run, err := gb.readUEGolombLong()
			if err != nil {
				return result, err
			}
			if run > uint32(m.MBWidth*m.MBHeight) {
				return result, ErrInvalidData
			}
			state.MBSkipRun = int32(run)
		}
		if state.MBSkipRun > 0 {
			if rejectUnsupportedHighB && sh.SliceTypeNoS == PictureTypeB {
				return result, ErrUnsupported
			}
			state.MBSkipRun--
			return m.writeBackCAVLCFrameSkipMacroblockWithDirectWork(sh, state.QScale, mbXY, sliceNum, direct, work)
		}
		state.MBSkipRun = cavlcMBSkipRunUnset
	}

	result, err := m.decodeCAVLCFrameMacroblockWithWork(gb, cavlcFrameMacroblockInput{
		MBXY:                   mbXY,
		SliceNum:               sliceNum,
		SliceType:              sh.SliceType,
		SliceTypeNoS:           sh.SliceTypeNoS,
		QScale:                 state.QScale,
		RefCount:               sh.RefCount,
		DCT8x8Allowed:          sh.PPS.Transform8x8Mode != 0,
		DirectSpatialMVPred:    sh.DirectSpatialMVPred != 0,
		Direct:                 direct,
		PPS:                    sh.PPS,
		SPS:                    sh.SPS,
		RejectUnsupportedHighB: rejectUnsupportedHighB,
	}, work)
	if err != nil {
		return result, err
	}
	if result.MBType&MBTypeIntraPCM == 0 {
		state.QScale = result.QScale
	}
	return result, nil
}

func (m *macroblockTables) decodeCAVLCFrameMacroblock(gb *bitReader, in cavlcFrameMacroblockInput) (cavlcFrameMacroblockResult, error) {
	var work frameMacroblockDecodeWork
	return m.decodeCAVLCFrameMacroblockWithWork(gb, in, &work)
}

func (m *macroblockTables) decodeCAVLCFrameMacroblockWithWork(gb *bitReader, in cavlcFrameMacroblockInput, work *frameMacroblockDecodeWork) (cavlcFrameMacroblockResult, error) {
	var result cavlcFrameMacroblockResult
	if m == nil || gb == nil || in.PPS == nil || in.SPS == nil || work == nil {
		return result, ErrInvalidData
	}
	if in.QScale < 0 || in.QScale > qpMaxNum {
		return result, ErrInvalidData
	}
	*work = frameMacroblockDecodeWork{}

	base, err := decodeCAVLCMBType(gb, in.SliceType, in.SliceTypeNoS)
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
		return m.decodeCAVLCFrameIntraPCMMacroblock(gb, in, base, result)
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
		CABAC:                false,
		ConstrainedIntraPred: in.PPS.ConstrainedIntraPred != 0,
		DirectSpatialMVPred:  in.DirectSpatialMVPred,
	})
	if err != nil {
		return result, err
	}
	result.Neighbors = cacheResult.Neighbors

	if isIntra(base.MBType) {
		return m.decodeCAVLCFrameIntraMacroblock(gb, in, base, &work.Residual, &work.IntraCache, cacheResult, result)
	}
	return m.decodeCAVLCFrameInterMacroblock(gb, in, base, &work.Residual, &work.Motion, listCount, result)
}

func (m *macroblockTables) decodeCAVLCFrameIntraPCMMacroblock(gb *bitReader, in cavlcFrameMacroblockInput, base cavlcMacroblockSyntax, result cavlcFrameMacroblockResult) (cavlcFrameMacroblockResult, error) {
	pcm, err := readCAVLCIntraPCMBytes(gb, in.SPS)
	if err != nil {
		return result, err
	}
	base.IntraPCM = pcm
	base.QScale = 0
	if err := m.writeBackCAVLCIntraPCMMacroblock(in.MBXY, in.SliceNum); err != nil {
		return result, err
	}
	result.MBType = base.MBType
	result.CBP = 0
	result.CBPTable = 0
	result.QScale = 0
	result.Intra = base
	result.IntraPCM = pcm
	result.IsIntra = true
	return result, nil
}

func (m *macroblockTables) writeBackCAVLCFrameSkipMacroblock(sh *SliceHeader, mbXY int, sliceNum uint16) (cavlcFrameMacroblockResult, error) {
	var work frameMacroblockDecodeWork
	if sh == nil {
		return cavlcFrameMacroblockResult{}, ErrInvalidData
	}
	return m.writeBackCAVLCFrameSkipMacroblockWithWork(sh, int(sh.QScale), mbXY, sliceNum, &work)
}

func (m *macroblockTables) writeBackCAVLCFrameSkipMacroblockWithWork(sh *SliceHeader, qscale int, mbXY int, sliceNum uint16, work *frameMacroblockDecodeWork) (cavlcFrameMacroblockResult, error) {
	return m.writeBackCAVLCFrameSkipMacroblockWithDirectWork(sh, qscale, mbXY, sliceNum, h264DirectMotionContext{}, work)
}

func (m *macroblockTables) writeBackCAVLCFrameSkipMacroblockWithDirectWork(sh *SliceHeader, qscale int, mbXY int, sliceNum uint16, direct h264DirectMotionContext, work *frameMacroblockDecodeWork) (cavlcFrameMacroblockResult, error) {
	var result cavlcFrameMacroblockResult
	if sh == nil || work == nil {
		return result, ErrInvalidData
	}
	if qscale < 0 || qscale > qpMaxNum {
		return result, ErrInvalidData
	}
	if sh.SliceTypeNoS == PictureTypeB {
		return m.writeBackCAVLCFrameBSkipMacroblockWithDirectWork(sh, qscale, mbXY, sliceNum, direct, work)
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
	if err := m.writeBackPskipMacroblockWithMotion(mbXY, qscale, neighbors.motionNeighbors(mbType, 1, PictureTypeP, false, false), sliceNum, &work.Motion); err != nil {
		return result, err
	}

	result.MBType = mbType
	result.CBP = 0
	result.CBPTable = 0
	result.QScale = qscale
	result.Neighbors = neighbors
	result.IsInter = true
	result.Skipped = true
	return result, nil
}

func (m *macroblockTables) writeBackCAVLCFrameBSkipMacroblockWithDirectWork(sh *SliceHeader, qscale int, mbXY int, sliceNum uint16, direct h264DirectMotionContext, work *frameMacroblockDecodeWork) (cavlcFrameMacroblockResult, error) {
	var result cavlcFrameMacroblockResult
	mbType := MBTypeL0L1 | MBTypeDirect2 | MBTypeSkip
	neighbors, err := m.fillDecodeNeighborsFrame(mbXY, sliceNum, mbType)
	if err != nil {
		return result, err
	}
	*work = frameMacroblockDecodeWork{}
	if direct.DirectSpatialMVPred {
		if err := m.fillMotionDecodeCaches(&work.Motion, neighbors.motionNeighbors(mbType, 2, PictureTypeB, false, true)); err != nil {
			return result, err
		}
	}
	var subMBType [4]uint32
	if err := m.predDirectMotionFrame(&work.Motion, mbXY, &mbType, &subMBType, direct); err != nil {
		return result, err
	}
	if err := m.writeBackBskipMacroblockWithMotion(mbXY, qscale, mbType, false, &subMBType, sliceNum, &work.Motion); err != nil {
		return result, err
	}

	result.MBType = mbType
	result.CBP = 0
	result.CBPTable = 0
	result.QScale = qscale
	result.Neighbors = neighbors
	result.Inter.SubMBType = subMBType
	result.IsInter = true
	result.Skipped = true
	return result, nil
}

func (m *macroblockTables) decodeCAVLCFrameIntraMacroblock(gb *bitReader, in cavlcFrameMacroblockInput, base cavlcMacroblockSyntax, residual *cavlcResidualContext, intraCache *[h264IntraPredModeCacheSize]int8, cacheResult frameMacroblockDecodeCacheResult, result cavlcFrameMacroblockResult) (cavlcFrameMacroblockResult, error) {
	mb, err := residual.decodeCAVLCFrameIntraMacroblockAfterType(gb, in.PPS, in.SPS, base, in.QScale, in.DCT8x8Allowed, intraCache)
	if err != nil {
		return result, err
	}
	rawChromaPred := int32(0)
	if in.SPS.ChromaFormatIDC == 1 || in.SPS.ChromaFormatIDC == 2 {
		rawChromaPred = mb.ChromaPredMode
	}
	if err := validateCAVLCFrameIntraPredModes(&mb, in.SPS, intraCache, cacheResult.Intra); err != nil {
		return result, err
	}
	if err := m.writeBackCAVLCIntraMacroblockWithChromaPred(in.MBXY, &mb, residual, int8(rawChromaPred), in.SliceNum); err != nil {
		return result, err
	}

	result.MBType = mb.MBType
	result.CBP = mb.CBP
	result.CBPTable = mb.CBPTable
	result.QScale = mb.QScale
	result.ChromaQP = mb.ChromaQP
	result.ChromaPred = mb.ChromaPredMode
	result.TopLeftAvailable = cacheResult.Intra.TopLeftSamplesAvailable
	result.TopRightAvailable = cacheResult.Intra.TopRightSamplesAvailable
	result.Intra = mb
	result.IsIntra = true
	return result, nil
}

func (m *macroblockTables) decodeCAVLCFrameInterMacroblock(gb *bitReader, in cavlcFrameMacroblockInput, base cavlcMacroblockSyntax, residual *cavlcResidualContext, motion *macroblockMotionCache, listCount int, result cavlcFrameMacroblockResult) (cavlcFrameMacroblockResult, error) {
	var mb cavlcInterMacroblockSyntax
	mb.cavlcMacroblockSyntax = base
	var err error
	switch in.SliceTypeNoS {
	case PictureTypeP:
		mb, err = residual.decodeCAVLCInterPMacroblockAfterType(gb, in.PPS, in.SPS, mb, in.QScale, in.RefCount, in.DCT8x8Allowed)
	case PictureTypeB:
		mb, err = residual.decodeCAVLCInterBMacroblockAfterType(gb, in.PPS, in.SPS, mb, in.QScale, in.RefCount, in.DCT8x8Allowed)
	default:
		return result, ErrInvalidData
	}
	if err != nil {
		return result, err
	}
	if isDirect(mb.MBType) {
		if err := m.predDirectMotionFrame(motion, in.MBXY, &mb.MBType, &mb.SubMBType, in.Direct); err != nil {
			return result, err
		}
	} else if in.SliceTypeNoS == PictureTypeB && mb.PartitionCount == 4 && hasDirectSubMBType(&mb.SubMBType) {
		if err := m.predDirectMotionFrame(motion, in.MBXY, &mb.MBType, &mb.SubMBType, in.Direct); err != nil {
			return result, err
		}
		markDirectSubRefsUnavailable(motion)
	}
	if in.RejectUnsupportedHighB {
		if err := validateHighFrameSliceMacroblockForReconstruct(&SliceHeader{SliceTypeNoS: in.SliceTypeNoS}, mb.MBType, mb.CBP, mb.CBPTable); err != nil {
			return result, err
		}
	}
	if err := m.writeBackCAVLCInterMacroblock(in.MBXY, &mb, residual, motion, listCount, in.SliceTypeNoS, in.SliceNum); err != nil {
		return result, err
	}

	result.MBType = mb.MBType
	result.CBP = mb.CBP
	result.CBPTable = mb.CBPTable
	result.QScale = mb.QScale
	result.ChromaQP = mb.ChromaQP
	result.Inter = mb
	result.IsInter = true
	return result, nil
}

func validateCAVLCFrameIntraPredModes(mb *cavlcMacroblockSyntax, sps *SPS, cache *[h264IntraPredModeCacheSize]int8, cacheResult intraPredDecodeCacheResult) error {
	if mb == nil || sps == nil {
		return ErrInvalidData
	}
	if isIntra4x4(mb.MBType) {
		if cache == nil {
			return ErrInvalidData
		}
		if err := fillIntra4x4PredModeCacheFromSyntax(cache, &mb.Intra4x4PredMode); err != nil {
			return err
		}
		if err := checkIntra4x4PredModeCache(cache, cacheResult.TopSamplesAvailable, cacheResult.LeftSamplesAvailable); err != nil {
			return err
		}
	} else if isIntra16x16(mb.MBType) {
		mode, err := checkIntraPredMode(int(mb.Intra16x16PredMode), cacheResult.TopSamplesAvailable, cacheResult.LeftSamplesAvailable, false)
		if err != nil {
			return err
		}
		mb.Intra16x16PredMode = int8(mode)
	}

	if sps.ChromaFormatIDC == 1 || sps.ChromaFormatIDC == 2 {
		mode, err := checkIntraPredMode(int(mb.ChromaPredMode), cacheResult.TopSamplesAvailable, cacheResult.LeftSamplesAvailable, true)
		if err != nil {
			return err
		}
		mb.ChromaPredMode = int32(mode)
	} else {
		mb.ChromaPredMode = intraPredDC1288x8
	}
	return nil
}

func readCAVLCIntraPCMBytes(gb *bitReader, sps *SPS) ([]byte, error) {
	if gb == nil || sps == nil || sps.ChromaFormatIDC >= uint32(len(h264IntraPCMSampleCount)) {
		return nil, ErrInvalidData
	}
	n, err := h264IntraPCMByteCount(int(sps.ChromaFormatIDC), int(sps.BitDepthLuma))
	if err != nil {
		return nil, err
	}
	return gb.readAlignedBytes(n)
}

func cavlcFrameListCount(sliceTypeNoS int32) (int, error) {
	switch sliceTypeNoS {
	case PictureTypeI:
		return 0, nil
	case PictureTypeP:
		return 1, nil
	case PictureTypeB:
		return 2, nil
	default:
		return 0, ErrInvalidData
	}
}
