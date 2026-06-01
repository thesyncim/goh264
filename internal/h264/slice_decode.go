// SPDX-License-Identifier: LGPL-2.1-or-later
//
// Source-shaped simple frame-MB slice decode/reconstruct loop from FFmpeg
// n8.0.1 libavcodec/h264_slice.c decode_slice. This layer keeps the raw
// "entropy MB, then hl_decode_mb" order for frame pictures while row-threaded
// deblocking, MBAFF/field pictures, error resilience, and threading remain
// separate lanes.

package h264

type h264FrameSliceDecodeInput struct {
	SliceNum      uint16
	Refs          [2][]*h264PicturePlanes
	Direct        h264DirectMotionContext
	PredWeight    *PredWeightTable
	MotionScratch *h264MotionCompScratch
}

type h264FrameSliceDecodeInputHigh struct {
	SliceNum      uint16
	Refs          [2][]*h264PicturePlanesHigh
	Direct        h264DirectMotionContext
	PredWeight    *PredWeightTable
	MotionScratch *h264MotionCompScratchHigh
}

type h264FrameSliceDecodeResult struct {
	Macroblocks int
	LastMBXY    int
	EndOfSlice  bool
	EndOfFrame  bool
}

type cabacFrameSliceDecoder struct {
	cabac cabacContext
	state [1024]uint8
}

func (d *cabacFrameSliceDecoder) source() cabacSyntaxDecoder {
	return cabacSyntaxDecoder{
		cabac: &d.cabac,
		state: &d.state,
	}
}

func (m *macroblockTables) decodeFrameSliceData(gb *bitReader, dst *h264PicturePlanes, sh *SliceHeader, in h264FrameSliceDecodeInput) (h264FrameSliceDecodeResult, error) {
	var result h264FrameSliceDecodeResult
	if sh == nil || sh.PPS == nil {
		return result, ErrInvalidData
	}
	if sh.PPS.CABAC == 0 {
		return m.decodeCAVLCFrameSlice(gb, dst, sh, in)
	}
	dec, err := initCABACFrameSliceDecoder(gb, sh)
	if err != nil {
		return result, err
	}
	return m.decodeCABACFrameSlice(dec.source(), dst, sh, in)
}

func (m *macroblockTables) decodeFrameSliceDataHigh(gb *bitReader, dst *h264PicturePlanesHigh, sh *SliceHeader, in h264FrameSliceDecodeInputHigh) (h264FrameSliceDecodeResult, error) {
	var result h264FrameSliceDecodeResult
	if m == nil || gb == nil || dst == nil || sh == nil || sh.PPS == nil || sh.SPS == nil {
		return result, ErrInvalidData
	}
	if err := validateSimpleFrameSliceDecodeInputsHigh(m, dst, sh, in.SliceNum); err != nil {
		return result, err
	}
	if sh.PPS.CABAC == 0 {
		return m.decodeCAVLCFrameSliceHigh(gb, dst, sh, in)
	}
	dec, err := initCABACFrameSliceDecoder(gb, sh)
	if err != nil {
		return result, err
	}
	return m.decodeCABACFrameSliceHigh(dec.source(), dst, sh, in)
}

func initCABACFrameSliceDecoder(gb *bitReader, sh *SliceHeader) (cabacFrameSliceDecoder, error) {
	var dec cabacFrameSliceDecoder
	if gb == nil || sh == nil || sh.SPS == nil {
		return dec, ErrInvalidData
	}
	buf, err := gb.remainingAlignedBytes()
	if err != nil {
		return dec, err
	}
	dec.cabac, err = initCABACDecoder(buf)
	if err != nil {
		return dec, err
	}
	dec.state, err = initH264CABACStates(sh.SliceTypeNoS, sh.CABACInitIDC, int32(sh.QScale), int32(sh.SPS.BitDepthLuma))
	if err != nil {
		return dec, err
	}
	return dec, nil
}

func (m *macroblockTables) decodeCAVLCFrameSlice(gb *bitReader, dst *h264PicturePlanes, sh *SliceHeader, in h264FrameSliceDecodeInput) (h264FrameSliceDecodeResult, error) {
	var result h264FrameSliceDecodeResult
	if m == nil || gb == nil || dst == nil || sh == nil || sh.PPS == nil || sh.SPS == nil {
		return result, ErrInvalidData
	}
	if err := validateSimpleFrameSliceDecodeInputs(m, dst, sh, in.SliceNum); err != nil {
		return result, err
	}
	cur, err := newSliceMacroblockCursor(m, sh)
	if err != nil {
		return result, err
	}

	state := newCAVLCFrameSliceState(int(sh.QScale))
	for {
		var work frameMacroblockDecodeWork
		mb, err := m.decodeCAVLCFrameSliceMacroblockWithDirectWork(gb, sh, &state, cur.MBXY, in.SliceNum, in.Direct, &work)
		if err != nil {
			return result, err
		}
		if err := h264HLDecodeFrameMacroblock(dst, h264FrameMBReconstructInputFromCAVLC(sh, cur, mb, &work, in)); err != nil {
			return result, err
		}
		result.Macroblocks++
		result.LastMBXY = cur.MBXY

		if !cur.advanceFrameMB() {
			result.EndOfFrame = true
			result.EndOfSlice = true
			return result, nil
		}
		if gb.bitsLeft() <= 0 && state.MBSkipRun <= 0 {
			result.EndOfSlice = true
			return result, nil
		}
	}
}

func (m *macroblockTables) decodeCAVLCFrameSliceHigh(gb *bitReader, dst *h264PicturePlanesHigh, sh *SliceHeader, in h264FrameSliceDecodeInputHigh) (h264FrameSliceDecodeResult, error) {
	var result h264FrameSliceDecodeResult
	if m == nil || gb == nil || dst == nil || sh == nil || sh.PPS == nil || sh.SPS == nil {
		return result, ErrInvalidData
	}
	if err := validateSimpleFrameSliceDecodeInputsHigh(m, dst, sh, in.SliceNum); err != nil {
		return result, err
	}
	cur, err := newSliceMacroblockCursor(m, sh)
	if err != nil {
		return result, err
	}

	state := newCAVLCFrameSliceState(int(sh.QScale))
	for {
		var work frameMacroblockDecodeWork
		mb, err := m.decodeCAVLCFrameSliceMacroblockWithDirectWork(gb, sh, &state, cur.MBXY, in.SliceNum, in.Direct, &work)
		if err != nil {
			return result, err
		}
		if err := validateHighFrameSliceMacroblockForReconstruct(sh, mb.MBType, mb.CBP, mb.CBPTable); err != nil {
			return result, err
		}
		if err := h264HLDecodeFrameMacroblockHigh(dst, h264FrameMBReconstructInputHighFromCAVLC(sh, cur, mb, &work, in)); err != nil {
			return result, err
		}
		result.Macroblocks++
		result.LastMBXY = cur.MBXY

		if !cur.advanceFrameMB() {
			result.EndOfFrame = true
			result.EndOfSlice = true
			return result, nil
		}
		if gb.bitsLeft() <= 0 && state.MBSkipRun <= 0 {
			result.EndOfSlice = true
			return result, nil
		}
	}
}

func (m *macroblockTables) decodeCABACFrameSlice(src cabacSyntaxSource, dst *h264PicturePlanes, sh *SliceHeader, in h264FrameSliceDecodeInput) (h264FrameSliceDecodeResult, error) {
	var result h264FrameSliceDecodeResult
	if m == nil || src == nil || dst == nil || sh == nil || sh.PPS == nil || sh.SPS == nil {
		return result, ErrInvalidData
	}
	if err := validateSimpleFrameSliceDecodeInputs(m, dst, sh, in.SliceNum); err != nil {
		return result, err
	}
	cur, err := newSliceMacroblockCursor(m, sh)
	if err != nil {
		return result, err
	}

	state := cabacFrameSliceState{QScale: int(sh.QScale)}
	for {
		var work frameMacroblockDecodeWork
		mb, err := m.decodeCABACFrameSliceMacroblockWithDirectWork(src, sh, &state, cur.MBXY, in.SliceNum, in.Direct, &work)
		if err != nil {
			return result, err
		}
		if err := h264HLDecodeFrameMacroblock(dst, h264FrameMBReconstructInputFromCABAC(sh, cur, mb, &work, in)); err != nil {
			return result, err
		}
		result.Macroblocks++
		result.LastMBXY = cur.MBXY

		eos := src.terminate() != 0
		if !cur.advanceFrameMB() {
			result.EndOfFrame = true
			result.EndOfSlice = true
			return result, nil
		}
		if eos {
			result.EndOfSlice = true
			return result, nil
		}
	}
}

func (m *macroblockTables) decodeCABACFrameSliceHigh(src cabacSyntaxSource, dst *h264PicturePlanesHigh, sh *SliceHeader, in h264FrameSliceDecodeInputHigh) (h264FrameSliceDecodeResult, error) {
	var result h264FrameSliceDecodeResult
	if m == nil || src == nil || dst == nil || sh == nil || sh.PPS == nil || sh.SPS == nil {
		return result, ErrInvalidData
	}
	if err := validateSimpleFrameSliceDecodeInputsHigh(m, dst, sh, in.SliceNum); err != nil {
		return result, err
	}
	cur, err := newSliceMacroblockCursor(m, sh)
	if err != nil {
		return result, err
	}

	state := cabacFrameSliceState{QScale: int(sh.QScale)}
	for {
		var work frameMacroblockDecodeWork
		mb, err := m.decodeCABACFrameSliceMacroblockWithDirectWork(src, sh, &state, cur.MBXY, in.SliceNum, in.Direct, &work)
		if err != nil {
			return result, err
		}
		if err := validateHighFrameSliceMacroblockForReconstruct(sh, mb.MBType, mb.CBP, mb.CBPTable); err != nil {
			return result, err
		}
		if err := h264HLDecodeFrameMacroblockHigh(dst, h264FrameMBReconstructInputHighFromCABAC(sh, cur, mb, &work, in)); err != nil {
			return result, err
		}
		result.Macroblocks++
		result.LastMBXY = cur.MBXY

		eos := src.terminate() != 0
		if !cur.advanceFrameMB() {
			result.EndOfFrame = true
			result.EndOfSlice = true
			return result, nil
		}
		if eos {
			result.EndOfSlice = true
			return result, nil
		}
	}
}

func validateSimpleFrameSliceDecodeInputs(m *macroblockTables, dst *h264PicturePlanes, sh *SliceHeader, sliceNum uint16) error {
	if sliceNum == ^uint16(0) {
		return ErrInvalidData
	}
	if sh.PictureStructure != PictureFrame || sh.SPS.MBAFF != 0 {
		return ErrUnsupported
	}
	if !h264SimpleFrameSliceDecodeSupportsBitDepth(sh.SPS.BitDepthLuma) {
		return ErrUnsupported
	}
	if sh.QScale > qpMaxNum {
		return ErrInvalidData
	}
	if _, err := cavlcFrameListCount(sh.SliceTypeNoS); err != nil {
		return err
	}
	if err := dst.validate(); err != nil {
		return err
	}
	if m.MBWidth != dst.MBWidth || m.MBHeight != dst.MBHeight || m.ChromaFormatIDC != dst.ChromaFormatIDC || int(sh.SPS.ChromaFormatIDC) != dst.ChromaFormatIDC {
		return ErrInvalidData
	}
	return nil
}

func validateSimpleFrameSliceDecodeInputsHigh(m *macroblockTables, dst *h264PicturePlanesHigh, sh *SliceHeader, sliceNum uint16) error {
	if sliceNum == ^uint16(0) {
		return ErrInvalidData
	}
	if sh == nil || sh.SPS == nil {
		return ErrInvalidData
	}
	if sh.PictureStructure != PictureFrame || sh.SPS.MBAFF != 0 {
		return ErrUnsupported
	}
	if sh.SPS.BitDepthLuma != 10 {
		return ErrUnsupported
	}
	if err := checkH264DSPHighBitDepth(int(sh.SPS.BitDepthLuma)); err != nil {
		return err
	}
	if sh.SPS.BitDepthChroma != sh.SPS.BitDepthLuma {
		return ErrUnsupported
	}
	if sh.SPS.ChromaFormatIDC != 1 {
		return ErrUnsupported
	}
	switch sh.SliceTypeNoS {
	case PictureTypeI:
	case PictureTypeP:
	default:
		return ErrUnsupported
	}
	if sh.DeblockingFilter != 0 {
		return ErrUnsupported
	}
	if sh.QScale > uint32(h264MaxQPForBitDepth(int(sh.SPS.BitDepthLuma))) {
		return ErrInvalidData
	}
	if _, err := cavlcFrameListCount(sh.SliceTypeNoS); err != nil {
		return err
	}
	if err := dst.validate(); err != nil {
		return err
	}
	if m.MBWidth != dst.MBWidth || m.MBHeight != dst.MBHeight || m.ChromaFormatIDC != dst.ChromaFormatIDC || int(sh.SPS.ChromaFormatIDC) != dst.ChromaFormatIDC {
		return ErrInvalidData
	}
	return nil
}

func validateHighFrameSliceMacroblockForReconstruct(sh *SliceHeader, mbType uint32, cbp int, cbpTable int) error {
	if sh == nil {
		return ErrInvalidData
	}
	switch sh.SliceTypeNoS {
	case PictureTypeI:
		return nil
	case PictureTypeP:
	default:
		return ErrUnsupported
	}
	if cbp < 0 || cbpTable < 0 || isIntra(mbType) {
		return ErrUnsupported
	}
	if isSkip(mbType) {
		if mbType == MBType16x16|MBTypeP0L0|MBTypeP1L0|MBTypeSkip && cbp == 0 && cbpTable == 0 {
			return nil
		}
		return ErrUnsupported
	}
	if mbType == MBType16x16|MBTypeP0L0 {
		return nil
	}
	return ErrUnsupported
}

func h264SimpleFrameSliceDecodeSupportsBitDepth(bitDepth int32) bool {
	// High-depth entropy paths exist, but this simple slice loop still feeds
	// 8-bit reconstruction/loop-filter state.
	return bitDepth == 8
}

func h264FrameMBReconstructInputFromCAVLC(sh *SliceHeader, cur sliceMacroblockCursor, mb cavlcFrameMacroblockResult, work *frameMacroblockDecodeWork, in h264FrameSliceDecodeInput) h264FrameMBReconstructInput {
	listCount, _ := cavlcFrameListCount(sh.SliceTypeNoS)
	return h264FrameMBReconstructInput{
		MBType:             mb.MBType,
		SubMBType:          mb.Inter.SubMBType,
		MBX:                cur.MBX,
		MBY:                cur.MBY,
		CBP:                mb.CBP,
		QScale:             mb.QScale,
		ChromaQP:           mb.ChromaQP,
		ChromaPredMode:     mb.ChromaPred,
		Intra16x16PredMode: mb.Intra.Intra16x16PredMode,
		Intra4x4PredCache:  &work.IntraCache,
		TopLeftAvailable:   mb.TopLeftAvailable,
		TopRightAvailable:  mb.TopRightAvailable,
		ListCount:          listCount,
		PPS:                sh.PPS,
		Residual:           &work.Residual,
		Motion:             &work.Motion,
		Refs:               in.Refs,
		PredWeight:         in.PredWeight,
		MotionScratch:      in.MotionScratch,
		TransformBypass:    sh.SPS.TransformBypass != 0 && mb.QScale == 0,
		IntraPCM:           mb.IntraPCM,
	}
}

func h264FrameMBReconstructInputHighFromCAVLC(sh *SliceHeader, cur sliceMacroblockCursor, mb cavlcFrameMacroblockResult, work *frameMacroblockDecodeWork, in h264FrameSliceDecodeInputHigh) h264FrameMBReconstructInputHigh {
	listCount, _ := cavlcFrameListCount(sh.SliceTypeNoS)
	return h264FrameMBReconstructInputHigh{
		MBType:             mb.MBType,
		SubMBType:          mb.Inter.SubMBType,
		MBX:                cur.MBX,
		MBY:                cur.MBY,
		CBP:                mb.CBP,
		QScale:             mb.QScale,
		ChromaQP:           mb.ChromaQP,
		ChromaPredMode:     mb.ChromaPred,
		Intra16x16PredMode: mb.Intra.Intra16x16PredMode,
		Intra4x4PredCache:  &work.IntraCache,
		TopLeftAvailable:   mb.TopLeftAvailable,
		TopRightAvailable:  mb.TopRightAvailable,
		ListCount:          listCount,
		PPS:                sh.PPS,
		Residual:           &work.Residual,
		Motion:             &work.Motion,
		Refs:               in.Refs,
		PredWeight:         in.PredWeight,
		MotionScratch:      in.MotionScratch,
		TransformBypass:    sh.SPS.TransformBypass != 0 && mb.QScale == 0,
		DeblockingFilter:   sh.DeblockingFilter != 0,
		BitDepth:           int(sh.SPS.BitDepthLuma),
		IntraPCM:           mb.IntraPCM,
	}
}

func h264FrameMBReconstructInputHighFromCABAC(sh *SliceHeader, cur sliceMacroblockCursor, mb cabacFrameMacroblockResult, work *frameMacroblockDecodeWork, in h264FrameSliceDecodeInputHigh) h264FrameMBReconstructInputHigh {
	listCount, _ := cavlcFrameListCount(sh.SliceTypeNoS)
	return h264FrameMBReconstructInputHigh{
		MBType:             mb.MBType,
		SubMBType:          mb.Inter.SubMBType,
		MBX:                cur.MBX,
		MBY:                cur.MBY,
		CBP:                mb.CBP,
		QScale:             mb.QScale,
		ChromaQP:           mb.ChromaQP,
		ChromaPredMode:     mb.ChromaPred,
		Intra16x16PredMode: mb.Intra.Intra16x16PredMode,
		Intra4x4PredCache:  &work.IntraCache,
		TopLeftAvailable:   mb.TopLeftAvailable,
		TopRightAvailable:  mb.TopRightAvailable,
		ListCount:          listCount,
		PPS:                sh.PPS,
		Residual:           &work.Residual,
		Motion:             &work.Motion,
		Refs:               in.Refs,
		PredWeight:         in.PredWeight,
		MotionScratch:      in.MotionScratch,
		TransformBypass:    sh.SPS.TransformBypass != 0 && mb.QScale == 0,
		DeblockingFilter:   sh.DeblockingFilter != 0,
		BitDepth:           int(sh.SPS.BitDepthLuma),
		IntraPCM:           mb.IntraPCM,
	}
}

func h264FrameMBReconstructInputFromCABAC(sh *SliceHeader, cur sliceMacroblockCursor, mb cabacFrameMacroblockResult, work *frameMacroblockDecodeWork, in h264FrameSliceDecodeInput) h264FrameMBReconstructInput {
	listCount, _ := cavlcFrameListCount(sh.SliceTypeNoS)
	return h264FrameMBReconstructInput{
		MBType:             mb.MBType,
		SubMBType:          mb.Inter.SubMBType,
		MBX:                cur.MBX,
		MBY:                cur.MBY,
		CBP:                mb.CBP,
		QScale:             mb.QScale,
		ChromaQP:           mb.ChromaQP,
		ChromaPredMode:     mb.ChromaPred,
		Intra16x16PredMode: mb.Intra.Intra16x16PredMode,
		Intra4x4PredCache:  &work.IntraCache,
		TopLeftAvailable:   mb.TopLeftAvailable,
		TopRightAvailable:  mb.TopRightAvailable,
		ListCount:          listCount,
		PPS:                sh.PPS,
		Residual:           &work.Residual,
		Motion:             &work.Motion,
		Refs:               in.Refs,
		PredWeight:         in.PredWeight,
		MotionScratch:      in.MotionScratch,
		TransformBypass:    sh.SPS.TransformBypass != 0 && mb.QScale == 0,
		IntraPCM:           mb.IntraPCM,
	}
}
