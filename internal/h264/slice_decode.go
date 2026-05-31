// SPDX-License-Identifier: LGPL-2.1-or-later
//
// Source-shaped simple frame-MB slice decode/reconstruct loop from FFmpeg
// n8.0.1 libavcodec/h264_slice.c decode_slice. This layer keeps the raw
// "entropy MB, then hl_decode_mb" order for frame pictures while deblocking,
// MBAFF/field pictures, error resilience, and threading remain separate lanes.

package h264

type h264FrameSliceDecodeInput struct {
	SliceNum      uint16
	Refs          [2][]*h264PicturePlanes
	PredWeight    *PredWeightTable
	MotionScratch *h264MotionCompScratch
}

type h264FrameSliceDecodeResult struct {
	Macroblocks int
	LastMBXY    int
	EndOfSlice  bool
	EndOfFrame  bool
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

	state := newCAVLCFrameSliceState()
	for {
		var work frameMacroblockDecodeWork
		mb, err := m.decodeCAVLCFrameSliceMacroblockWithWork(gb, sh, &state, cur.MBXY, in.SliceNum, &work)
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

	state := cabacFrameSliceState{}
	for {
		var work frameMacroblockDecodeWork
		mb, err := m.decodeCABACFrameSliceMacroblockWithWork(src, sh, &state, cur.MBXY, in.SliceNum, &work)
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

func validateSimpleFrameSliceDecodeInputs(m *macroblockTables, dst *h264PicturePlanes, sh *SliceHeader, sliceNum uint16) error {
	if sliceNum == ^uint16(0) {
		return ErrInvalidData
	}
	if sh.PictureStructure != PictureFrame || sh.SPS.MBAFF != 0 {
		return ErrUnsupported
	}
	if sh.DeblockingFilter != 0 || sh.SPS.TransformBypass != 0 || sh.SPS.BitDepthLuma != 8 || sh.SPS.ChromaFormatIDC == 3 {
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
		IntraPCM:           mb.IntraPCM,
	}
}
