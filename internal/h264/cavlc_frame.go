// SPDX-License-Identifier: LGPL-2.1-or-later
//
// Source-shaped frame-MB CAVLC macroblock handoff from FFmpeg n8.0.1
// libavcodec/h264_cavlc.c ff_h264_decode_mb_cavlc. This layer deliberately
// stops at entropy-to-state write-back; reconstruction and deblocking remain
// separate decoder steps.

package h264

type cavlcFrameMacroblockInput struct {
	MBXY                int
	SliceNum            uint16
	SliceType           int32
	SliceTypeNoS        int32
	QScale              int
	RefCount            [2]uint32
	DCT8x8Allowed       bool
	DirectSpatialMVPred bool
	PPS                 *PPS
	SPS                 *SPS
}

type cavlcFrameMacroblockResult struct {
	MBType     uint32
	CBP        int
	CBPTable   int
	QScale     int
	ChromaQP   [2]uint8
	ChromaPred int32
	Neighbors  macroblockDecodeNeighbors
	Intra      cavlcMacroblockSyntax
	Inter      cavlcInterMacroblockSyntax
	IsIntra    bool
	IsInter    bool
	Skipped    bool
}

const cavlcMBSkipRunUnset int32 = -1

type cavlcFrameSliceState struct {
	MBSkipRun int32
}

func newCAVLCFrameSliceState() cavlcFrameSliceState {
	return cavlcFrameSliceState{MBSkipRun: cavlcMBSkipRunUnset}
}

func (m *macroblockTables) decodeCAVLCFrameSliceMacroblock(gb *bitReader, sh *SliceHeader, state *cavlcFrameSliceState, mbXY int, sliceNum uint16) (cavlcFrameMacroblockResult, error) {
	var result cavlcFrameMacroblockResult
	if m == nil || gb == nil || sh == nil || sh.PPS == nil || sh.SPS == nil || state == nil {
		return result, ErrInvalidData
	}
	if sh.PictureStructure != PictureFrame || sh.SPS.MBAFF != 0 {
		return result, ErrUnsupported
	}
	if sh.QScale > qpMaxNum || state.MBSkipRun < cavlcMBSkipRunUnset {
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
			state.MBSkipRun--
			return m.writeBackCAVLCFrameSkipMacroblock(sh, mbXY, sliceNum)
		}
		state.MBSkipRun = cavlcMBSkipRunUnset
	}

	return m.decodeCAVLCFrameMacroblock(gb, cavlcFrameMacroblockInput{
		MBXY:                mbXY,
		SliceNum:            sliceNum,
		SliceType:           sh.SliceType,
		SliceTypeNoS:        sh.SliceTypeNoS,
		QScale:              int(sh.QScale),
		RefCount:            sh.RefCount,
		DCT8x8Allowed:       sh.PPS.Transform8x8Mode != 0,
		DirectSpatialMVPred: sh.DirectSpatialMVPred != 0,
		PPS:                 sh.PPS,
		SPS:                 sh.SPS,
	})
}

func (m *macroblockTables) decodeCAVLCFrameMacroblock(gb *bitReader, in cavlcFrameMacroblockInput) (cavlcFrameMacroblockResult, error) {
	var result cavlcFrameMacroblockResult
	if m == nil || gb == nil || in.PPS == nil || in.SPS == nil {
		return result, ErrInvalidData
	}
	if in.QScale < 0 || in.QScale > qpMaxNum {
		return result, ErrInvalidData
	}

	base, err := decodeCAVLCMBType(gb, in.SliceType, in.SliceTypeNoS)
	if err != nil {
		return result, err
	}
	result.MBType = base.MBType
	if base.MBType&MBTypeIntraPCM != 0 {
		return result, ErrUnsupported
	}

	listCount, err := cavlcFrameListCount(in.SliceTypeNoS)
	if err != nil {
		return result, err
	}

	var intraCache [h264IntraPredModeCacheSize]int8
	var residual cavlcResidualContext
	var motion macroblockMotionCache
	cacheResult, err := m.fillFrameMacroblockDecodeCaches(&intraCache, &residual, &motion, frameMacroblockDecodeCacheInput{
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
		return m.decodeCAVLCFrameIntraMacroblock(gb, in, base, &residual, &intraCache, cacheResult, result)
	}
	return m.decodeCAVLCFrameInterMacroblock(gb, in, base, &residual, &motion, listCount, result)
}

func (m *macroblockTables) writeBackCAVLCFrameSkipMacroblock(sh *SliceHeader, mbXY int, sliceNum uint16) (cavlcFrameMacroblockResult, error) {
	var result cavlcFrameMacroblockResult
	if sh == nil {
		return result, ErrInvalidData
	}
	if sh.SliceTypeNoS != PictureTypeP {
		return result, ErrUnsupported
	}

	mbType := MBType16x16 | MBTypeP0L0 | MBTypeP1L0 | MBTypeSkip
	neighbors, err := m.fillDecodeNeighborsFrame(mbXY, sliceNum, mbType)
	if err != nil {
		return result, err
	}
	if err := m.writeBackPskipMacroblock(mbXY, int(sh.QScale), neighbors.motionNeighbors(mbType, 1, PictureTypeP, false, false), sliceNum); err != nil {
		return result, err
	}

	result.MBType = mbType
	result.CBP = 0
	result.CBPTable = 0
	result.QScale = int(sh.QScale)
	result.Neighbors = neighbors
	result.IsInter = true
	result.Skipped = true
	return result, nil
}

func (m *macroblockTables) decodeCAVLCFrameIntraMacroblock(gb *bitReader, in cavlcFrameMacroblockInput, base cavlcMacroblockSyntax, residual *cavlcResidualContext, intraCache *[h264IntraPredModeCacheSize]int8, cacheResult frameMacroblockDecodeCacheResult, result cavlcFrameMacroblockResult) (cavlcFrameMacroblockResult, error) {
	var pred [16]int8
	var err error
	if isIntra4x4(base.MBType) {
		pred, err = predIntra4x4Modes(intraCache)
		if err != nil {
			return result, err
		}
	}

	mb, err := residual.decodeCAVLCIntraMacroblockAfterType(gb, in.PPS, in.SPS, base, in.QScale, in.DCT8x8Allowed, pred)
	if err != nil {
		return result, err
	}
	if err := validateCAVLCFrameIntraPredModes(&mb, in.SPS, cacheResult.Intra); err != nil {
		return result, err
	}
	if err := m.writeBackCAVLCIntraMacroblock(in.MBXY, &mb, residual, in.SliceNum); err != nil {
		return result, err
	}

	result.MBType = mb.MBType
	result.CBP = mb.CBP
	result.CBPTable = mb.CBPTable
	result.QScale = mb.QScale
	result.ChromaQP = mb.ChromaQP
	result.ChromaPred = mb.ChromaPredMode
	result.Intra = mb
	result.IsIntra = true
	return result, nil
}

func (m *macroblockTables) decodeCAVLCFrameInterMacroblock(gb *bitReader, in cavlcFrameMacroblockInput, base cavlcMacroblockSyntax, residual *cavlcResidualContext, motion *macroblockMotionCache, listCount int, result cavlcFrameMacroblockResult) (cavlcFrameMacroblockResult, error) {
	if isDirect(base.MBType) {
		return result, ErrUnsupported
	}

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

func validateCAVLCFrameIntraPredModes(mb *cavlcMacroblockSyntax, sps *SPS, cacheResult intraPredDecodeCacheResult) error {
	if mb == nil || sps == nil {
		return ErrInvalidData
	}
	if isIntra4x4(mb.MBType) {
		var checkCache [h264IntraPredModeCacheSize]int8
		if err := fillIntra4x4PredModeCacheFromSyntax(&checkCache, &mb.Intra4x4PredMode); err != nil {
			return err
		}
		if err := checkIntra4x4PredModeCache(&checkCache, cacheResult.TopSamplesAvailable, cacheResult.LeftSamplesAvailable); err != nil {
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
