// SPDX-License-Identifier: LGPL-2.1-or-later
//
// Source-shaped simple frame-MB slice decode/reconstruct loop from FFmpeg
// n8.0.1 libavcodec/h264_slice.c decode_slice. This layer keeps the raw
// "entropy MB, then hl_decode_mb" order for frame pictures while row-threaded
// deblocking, field-coded MBAFF/PAFF pictures, error resilience, and threading
// remain separate lanes.

package h264

import "fmt"

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
	if err := validateSimpleFrameSliceDecodeInputHighRefs(sh, in); err != nil {
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
	buf, err := gb.remainingAlignedRawBytes()
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
	dstView := *dst
	if sh.PictureStructure != PictureFrame {
		applySimpleFieldRefPlane(&dstView, sh.PictureStructure)
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
			return result, fmt.Errorf("cavlc mb_xy=%d bit=%d: %w", cur.MBXY, gb.bitPos, err)
		}
		var refPlanes [2][32]h264PicturePlanes
		var refPtrs [2][32]*h264PicturePlanes
		reconDst, reconMBY, reconRefs, err := h264FrameMBAFFReconstructView(&dstView, cur, mb.MBType, in.Refs, &refPlanes, &refPtrs)
		if err != nil {
			return result, fmt.Errorf("reconstruct cavlc mb_xy=%d view: %w", cur.MBXY, err)
		}
		reconIn := h264FrameMBReconstructInputFromCAVLC(sh, cur, mb, &work, in)
		reconIn.MBY = reconMBY
		reconIn.Refs = reconRefs
		if err := h264HLDecodeFrameMacroblock(&reconDst, reconIn); err != nil {
			return result, fmt.Errorf("reconstruct cavlc mb_xy=%d: %w", cur.MBXY, err)
		}
		result.Macroblocks++
		result.LastMBXY = cur.MBXY
		if cur.FrameMBAFF {
			bottom, err := cur.bottomMBAFFFrameMB()
			if err != nil {
				return result, err
			}
			var bottomWork frameMacroblockDecodeWork
			bottomMB, err := m.decodeCAVLCFrameSliceMacroblockWithDirectWork(gb, sh, &state, bottom.MBXY, in.SliceNum, in.Direct, &bottomWork)
			if err != nil {
				return result, fmt.Errorf("cavlc mb_xy=%d bit=%d: %w", bottom.MBXY, gb.bitPos, err)
			}
			var bottomRefPlanes [2][32]h264PicturePlanes
			var bottomRefPtrs [2][32]*h264PicturePlanes
			bottomDst, bottomMBY, bottomRefs, err := h264FrameMBAFFReconstructView(&dstView, bottom, bottomMB.MBType, in.Refs, &bottomRefPlanes, &bottomRefPtrs)
			if err != nil {
				return result, fmt.Errorf("reconstruct cavlc mb_xy=%d view: %w", bottom.MBXY, err)
			}
			bottomIn := h264FrameMBReconstructInputFromCAVLC(sh, bottom, bottomMB, &bottomWork, in)
			bottomIn.MBY = bottomMBY
			bottomIn.Refs = bottomRefs
			if err := h264HLDecodeFrameMacroblock(&bottomDst, bottomIn); err != nil {
				return result, fmt.Errorf("reconstruct cavlc mb_xy=%d: %w", bottom.MBXY, err)
			}
			result.Macroblocks++
			result.LastMBXY = bottom.MBXY
		}

		if !cur.advanceFrameMB() {
			result.EndOfFrame = true
			result.EndOfSlice = true
			return result, nil
		}
		if cur.FrameMBAFF && cur.MBX == 0 {
			state.MBFieldDecodingFlag = m.predictFrameMBAFFFieldDecodingFlag(cur.MBXY, in.SliceNum)
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
	dstView := *dst
	if sh.PictureStructure != PictureFrame {
		applySimpleFieldRefPlaneHigh(&dstView, sh.PictureStructure)
	}
	if err := validateSimpleFrameSliceDecodeInputHighRefs(sh, in); err != nil {
		return result, err
	}
	cur, err := newSliceMacroblockCursor(m, sh)
	if err != nil {
		return result, err
	}

	state := newCAVLCFrameSliceState(int(sh.QScale))
	for {
		var work frameMacroblockDecodeWork
		mb, err := m.decodeCAVLCFrameSliceMacroblockWithDirectWorkGuard(gb, sh, &state, cur.MBXY, in.SliceNum, in.Direct, &work, sh.SliceTypeNoS == PictureTypeB)
		if err != nil {
			return result, fmt.Errorf("cavlc high mb_xy=%d bit=%d: %w", cur.MBXY, gb.bitPos, err)
		}
		if err := validateHighFrameSliceMacroblockForReconstructWithSubMB(sh, mb.MBType, &mb.Inter.SubMBType, mb.CBP, mb.CBPTable); err != nil {
			return result, fmt.Errorf("validate high cavlc mb_xy=%d: %w", cur.MBXY, err)
		}
		if err := h264HLDecodeFrameMacroblockHigh(&dstView, h264FrameMBReconstructInputHighFromCAVLC(sh, cur, mb, &work, in)); err != nil {
			return result, fmt.Errorf("reconstruct high cavlc mb_xy=%d: %w", cur.MBXY, err)
		}
		result.Macroblocks++
		result.LastMBXY = cur.MBXY

		if !cur.advanceFrameMB() {
			result.EndOfFrame = true
			result.EndOfSlice = true
			return result, nil
		}
		if cur.FrameMBAFF && cur.MBX == 0 {
			state.MBFieldDecodingFlag = m.predictFrameMBAFFFieldDecodingFlag(cur.MBXY, in.SliceNum)
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
	dstView := *dst
	if sh.PictureStructure != PictureFrame {
		applySimpleFieldRefPlane(&dstView, sh.PictureStructure)
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
			return result, fmt.Errorf("cabac mb_xy=%d: %w", cur.MBXY, err)
		}
		var refPlanes [2][32]h264PicturePlanes
		var refPtrs [2][32]*h264PicturePlanes
		reconDst, reconMBY, reconRefs, err := h264FrameMBAFFReconstructView(&dstView, cur, mb.MBType, in.Refs, &refPlanes, &refPtrs)
		if err != nil {
			return result, fmt.Errorf("reconstruct cabac mb_xy=%d view: %w", cur.MBXY, err)
		}
		reconIn := h264FrameMBReconstructInputFromCABAC(sh, cur, mb, &work, in)
		reconIn.MBY = reconMBY
		reconIn.Refs = reconRefs
		if err := h264HLDecodeFrameMacroblock(&reconDst, reconIn); err != nil {
			return result, fmt.Errorf("reconstruct cabac mb_xy=%d: %w", cur.MBXY, err)
		}
		result.Macroblocks++
		result.LastMBXY = cur.MBXY
		if cur.FrameMBAFF {
			bottom, err := cur.bottomMBAFFFrameMB()
			if err != nil {
				return result, err
			}
			var bottomWork frameMacroblockDecodeWork
			bottomMB, err := m.decodeCABACFrameSliceMacroblockWithDirectWork(src, sh, &state, bottom.MBXY, in.SliceNum, in.Direct, &bottomWork)
			if err != nil {
				return result, fmt.Errorf("cabac mb_xy=%d: %w", bottom.MBXY, err)
			}
			var bottomRefPlanes [2][32]h264PicturePlanes
			var bottomRefPtrs [2][32]*h264PicturePlanes
			bottomDst, bottomMBY, bottomRefs, err := h264FrameMBAFFReconstructView(&dstView, bottom, bottomMB.MBType, in.Refs, &bottomRefPlanes, &bottomRefPtrs)
			if err != nil {
				return result, fmt.Errorf("reconstruct cabac mb_xy=%d view: %w", bottom.MBXY, err)
			}
			bottomIn := h264FrameMBReconstructInputFromCABAC(sh, bottom, bottomMB, &bottomWork, in)
			bottomIn.MBY = bottomMBY
			bottomIn.Refs = bottomRefs
			if err := h264HLDecodeFrameMacroblock(&bottomDst, bottomIn); err != nil {
				return result, fmt.Errorf("reconstruct cabac mb_xy=%d: %w", bottom.MBXY, err)
			}
			result.Macroblocks++
			result.LastMBXY = bottom.MBXY
		}

		eos := src.terminate() != 0
		if !cur.advanceFrameMB() {
			result.EndOfFrame = true
			result.EndOfSlice = true
			return result, nil
		}
		if cur.FrameMBAFF && cur.MBX == 0 {
			state.MBFieldDecodingFlag = m.predictFrameMBAFFFieldDecodingFlag(cur.MBXY, in.SliceNum)
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
	dstView := *dst
	if sh.PictureStructure != PictureFrame {
		applySimpleFieldRefPlaneHigh(&dstView, sh.PictureStructure)
	}
	if err := validateSimpleFrameSliceDecodeInputHighRefs(sh, in); err != nil {
		return result, err
	}
	cur, err := newSliceMacroblockCursor(m, sh)
	if err != nil {
		return result, err
	}

	state := cabacFrameSliceState{QScale: int(sh.QScale)}
	for {
		var work frameMacroblockDecodeWork
		mb, err := m.decodeCABACFrameSliceMacroblockWithDirectWorkGuard(src, sh, &state, cur.MBXY, in.SliceNum, in.Direct, &work, sh.SliceTypeNoS == PictureTypeB)
		if err != nil {
			return result, fmt.Errorf("cabac high mb_xy=%d: %w", cur.MBXY, err)
		}
		if err := validateHighFrameSliceMacroblockForReconstructWithSubMB(sh, mb.MBType, &mb.Inter.SubMBType, mb.CBP, mb.CBPTable); err != nil {
			return result, fmt.Errorf("validate high cabac mb_xy=%d: %w", cur.MBXY, err)
		}
		if err := h264HLDecodeFrameMacroblockHigh(&dstView, h264FrameMBReconstructInputHighFromCABAC(sh, cur, mb, &work, in)); err != nil {
			return result, fmt.Errorf("reconstruct high cabac mb_xy=%d: %w", cur.MBXY, err)
		}
		result.Macroblocks++
		result.LastMBXY = cur.MBXY

		eos := src.terminate() != 0
		if !cur.advanceFrameMB() {
			result.EndOfFrame = true
			result.EndOfSlice = true
			return result, nil
		}
		if cur.FrameMBAFF && cur.MBX == 0 {
			state.MBFieldDecodingFlag = m.predictFrameMBAFFFieldDecodingFlag(cur.MBXY, in.SliceNum)
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
	if sh.PictureStructure != PictureFrame && sh.PictureStructure != PictureTopField && sh.PictureStructure != PictureBottomField {
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
	if sh.PictureStructure != PictureFrame {
		return ErrUnsupported
	}
	if !isPublicHighFrameBitDepthCandidate(sh.SPS.BitDepthLuma) {
		return ErrUnsupported
	}
	if sh.SPS.BitDepthLuma != 8 {
		if err := checkH264DSPHighBitDepth(int(sh.SPS.BitDepthLuma)); err != nil {
			return err
		}
	}
	if sh.SPS.BitDepthChroma != sh.SPS.BitDepthLuma {
		return ErrUnsupported
	}
	if sh.SPS.ChromaFormatIDC != 1 && sh.SPS.ChromaFormatIDC != 2 && sh.SPS.ChromaFormatIDC != 3 {
		return ErrUnsupported
	}
	if sh.DeblockingFilter < 0 || sh.DeblockingFilter > 2 {
		return ErrInvalidData
	}
	if !isPublicHighFrameBitDepthScope(sh) {
		return ErrUnsupported
	}
	if err := validateHighFrameSliceDeblockingScope(sh); err != nil {
		return err
	}
	if sh.SPS.ChromaFormatIDC != 1 {
		if sh.SliceTypeNoS != PictureTypeI && sh.SliceTypeNoS != PictureTypeP {
			return ErrUnsupported
		}
		if sh.DeblockingFilter != 1 {
			return ErrUnsupported
		}
		if sh.PPS != nil && sh.PPS.WeightedPred != 0 {
			return ErrUnsupported
		}
		if sh.PredWeightTable.UseWeight != 0 || sh.PredWeightTable.UseWeightChroma != 0 {
			return ErrUnsupported
		}
	}
	switch sh.SliceTypeNoS {
	case PictureTypeI:
	case PictureTypeP:
	case PictureTypeB:
		return validateHighFrameSliceBPredWeight(sh, &sh.PredWeightTable)
	default:
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

func isPublicHighFrameBitDepthCandidate(bitDepth int32) bool {
	switch bitDepth {
	case 8, 10, 12:
		return true
	default:
		return false
	}
}

func isPublicHighFrameBitDepthScope(sh *SliceHeader) bool {
	if sh == nil || sh.SPS == nil {
		return false
	}
	switch sh.SPS.BitDepthLuma {
	case 8:
		return sh.SPS.FrameMBSOnlyFlag == 0 &&
			sh.SPS.MBAFF != 0 &&
			sh.PictureStructure == PictureFrame &&
			sh.SPS.ChromaFormatIDC == 1
	case 10:
		return true
	case 12:
		return sh.SPS.ChromaFormatIDC == 1 &&
			sh.SliceTypeNoS == PictureTypeI &&
			sh.DeblockingFilter == 0
	default:
		return false
	}
}

func validateHighFrameSliceDeblockingScope(sh *SliceHeader) error {
	if sh == nil {
		return ErrInvalidData
	}
	if sh.DeblockingFilter == 2 {
		if sh.SliceTypeNoS == PictureTypeB {
			return ErrUnsupported
		}
	}
	return nil
}

func validateHighFrameSliceBDeblockingMacroblock(sh *SliceHeader, mbType uint32, subMBType *[4]uint32, cbp int, cbpTable int) error {
	if sh == nil || sh.SliceTypeNoS != PictureTypeB || sh.DeblockingFilter == 0 {
		return nil
	}
	if sh.PPS == nil {
		return ErrInvalidData
	}
	if sh.DeblockingFilter == 1 {
		if !isHighBImplicitWeighted(sh) {
			if isHighB16x16ExplicitMacroblock(mbType) || isHighB16x16DirectMacroblock(mbType) {
				return nil
			}
			if isHighB16x16DirectSkipMacroblock(mbType) && cbp == 0 && cbpTable == 0 {
				return nil
			}
			if isHighB16x8Or8x16ExplicitMacroblock(mbType) {
				return nil
			}
			if isHighB8x8ExplicitSubMacroblock(mbType, subMBType) {
				return nil
			}
			if isHighB8x8DirectSubMacroblock(mbType, subMBType) {
				return nil
			}
		}
		if isHighBImplicitWeighted(sh) {
			if isHighB16x16ExplicitMacroblock(mbType) {
				return nil
			}
			if isHighB16x16DirectMacroblock(mbType) {
				return nil
			}
			if isHighB16x16DirectSkipMacroblock(mbType) && cbp == 0 && cbpTable == 0 {
				return nil
			}
			if isHighB16x8Or8x16ExplicitMacroblock(mbType) {
				return nil
			}
			if isHighB8x8ExplicitSubMacroblock(mbType, subMBType) {
				return nil
			}
			if isHighB8x8DirectSubMacroblock(mbType, subMBType) {
				return nil
			}
		}
	}
	return ErrUnsupported
}

func validateHighFrameSliceBMacroblockScope(sh *SliceHeader, mbType uint32, subMBType *[4]uint32, cbp int, cbpTable int) error {
	if err := validateHighFrameSliceBDeblockingMacroblock(sh, mbType, subMBType, cbp, cbpTable); err != nil {
		return err
	}
	return nil
}

func validateSimpleFrameSliceDecodeInputHighRefs(sh *SliceHeader, in h264FrameSliceDecodeInputHigh) error {
	if sh == nil {
		return ErrInvalidData
	}
	return validateHighFrameSliceBPredWeight(sh, in.PredWeight)
}

func validateHighFrameSliceBPredWeight(sh *SliceHeader, pwt *PredWeightTable) error {
	if sh == nil {
		return ErrInvalidData
	}
	if sh.SliceTypeNoS != PictureTypeB || pwt == nil {
		return nil
	}
	if sh.PPS == nil {
		return ErrInvalidData
	}
	switch sh.PPS.WeightedBipredIDC {
	case 0:
		if pwt.UseWeight == 0 && pwt.UseWeightChroma == 0 {
			return nil
		}
	case 2:
		if pwt.UseWeight == 0 && pwt.UseWeightChroma == 0 {
			return nil
		}
		if pwt.UseWeight == 2 && pwt.UseWeightChroma == 2 {
			return nil
		}
	}
	return ErrUnsupported
}

func validateHighFrameSliceBaseMacroblockForDecode(sliceTypeNoS int32, mbType uint32) error {
	if sliceTypeNoS != PictureTypeB {
		return nil
	}
	if isHighB16x16ExplicitMacroblock(mbType) {
		return nil
	}
	if mbType == MBTypeDirect2|MBTypeL0L1 {
		return nil
	}
	if isHighBExplicitPartitionedBaseMacroblock(mbType) {
		return nil
	}
	return ErrUnsupported
}

func validateHighFrameSliceMacroblockForReconstruct(sh *SliceHeader, mbType uint32, cbp int, cbpTable int) error {
	return validateHighFrameSliceMacroblockForReconstructWithSubMB(sh, mbType, nil, cbp, cbpTable)
}

func validateHighFrameSliceMacroblockForReconstructWithSubMB(sh *SliceHeader, mbType uint32, subMBType *[4]uint32, cbp int, cbpTable int) error {
	if sh == nil {
		return ErrInvalidData
	}
	switch sh.SliceTypeNoS {
	case PictureTypeI:
		if sh.SPS != nil && sh.SPS.BitDepthLuma != 10 {
			if mbType == MBTypeIntraPCM {
				return nil
			}
			if sh.SPS.BitDepthLuma == 12 {
				if (mbType == MBTypeIntra4x4 || mbType == MBTypeIntra16x16) && cbp == 0 && cbpTable == 0 {
					return nil
				}
				if mbType == MBTypeIntra16x16 && cbp == 0 && cbpTable == 0x100 {
					return nil
				}
				if mbType == MBTypeIntra16x16 && cbp == 0x10 && (cbpTable == 0x10 || cbpTable == 0x50) {
					return nil
				}
				if mbType == MBTypeIntra16x16 && cbp == 0x20 && cbpTable == 0x20 {
					return nil
				}
				if mbType == MBTypeIntra16x16 && cbp == 0x0f && (cbpTable == 0x0f || cbpTable == 0xf00f) {
					return nil
				}
				if mbType == MBTypeIntra16x16 && cbp == 0x0f && cbpTable == 0x10f {
					return nil
				}
			}
			return ErrUnsupported
		}
		return nil
	case PictureTypeP:
	case PictureTypeB:
	default:
		return ErrUnsupported
	}
	if cbp < 0 || cbpTable < 0 {
		return ErrUnsupported
	}
	if isIntra(mbType) {
		if sh.SliceTypeNoS == PictureTypeP && isHighPIntraMacroblock(mbType) {
			return nil
		}
		return ErrUnsupported
	}
	if sh.SliceTypeNoS == PictureTypeB {
		if err := validateHighFrameSliceBMacroblockScope(sh, mbType, subMBType, cbp, cbpTable); err != nil {
			return err
		}
		if isSkip(mbType) {
			if isHighB16x16DirectSkipMacroblock(mbType) && cbp == 0 && cbpTable == 0 {
				return nil
			}
			return ErrUnsupported
		}
		if isHighB16x16ExplicitMacroblock(mbType) {
			return nil
		}
		if isHighB16x16DirectMacroblock(mbType) {
			return nil
		}
		if isHighB8x8DirectSubMacroblock(mbType, subMBType) {
			return nil
		}
		if isHighBExplicitPartitionedMacroblock(mbType, subMBType) {
			return nil
		}
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
	if isHighPPartitionedMacroblock(sh, mbType, subMBType) {
		return nil
	}
	return ErrUnsupported
}

func isHighPIntraMacroblock(mbType uint32) bool {
	return mbType == MBTypeIntra4x4 || mbType == MBTypeIntra16x16
}

func isHighPPartitionedMacroblock(sh *SliceHeader, mbType uint32, subMBType *[4]uint32) bool {
	if sh == nil || sh.SliceTypeNoS != PictureTypeP || mbType&MBType8x8DCT != 0 {
		return false
	}
	switch mbType {
	case MBType16x8 | MBTypeP0L0 | MBTypeP1L0,
		MBType8x16 | MBTypeP0L0 | MBTypeP1L0:
		return true
	case MBType8x8 | MBTypeP0L0 | MBTypeP1L0,
		MBType8x8 | MBTypeP0L0 | MBTypeP1L0 | MBTypeRef0:
		if subMBType == nil {
			return false
		}
		for i := 0; i < 4; i++ {
			if !isHighPSubMBType(subMBType[i]) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func isHighPWeighted(sh *SliceHeader) bool {
	if sh == nil {
		return false
	}
	if sh.PPS != nil && sh.PPS.WeightedPred != 0 {
		return true
	}
	return sh.PredWeightTable.UseWeight != 0 || sh.PredWeightTable.UseWeightChroma != 0
}

func isHighPSubMBType(subType uint32) bool {
	switch subType {
	case MBType16x16 | MBTypeP0L0,
		MBType16x8 | MBTypeP0L0,
		MBType8x16 | MBTypeP0L0,
		MBType8x8 | MBTypeP0L0:
		return true
	default:
		return false
	}
}

func isHighB8x8DirectSubCarrier(mbType uint32) bool {
	const carrier = MBType8x8 | MBTypeP0L0 | MBTypeP0L1 | MBTypeP1L0 | MBTypeP1L1
	return mbType == carrier
}

func isHighBExplicitPartitionedBaseMacroblock(mbType uint32) bool {
	return isHighB16x8Or8x16ExplicitMacroblock(mbType) || isHighB8x8DirectSubCarrier(mbType)
}

func isHighBExplicitPartitionedMacroblock(mbType uint32, subMBType *[4]uint32) bool {
	return isHighB16x8Or8x16ExplicitMacroblock(mbType) || isHighB8x8ExplicitSubMacroblock(mbType, subMBType)
}

func isHighBImplicitWeighted(sh *SliceHeader) bool {
	return sh != nil && sh.PPS != nil && sh.PPS.WeightedBipredIDC == 2
}

func isHighB16x16ExplicitMacroblock(mbType uint32) bool {
	switch mbType {
	case MBType16x16 | MBTypeP0L1,
		MBType16x16 | MBTypeP0L0 | MBTypeP0L1:
		return true
	default:
		return false
	}
}

func isHighB16x8Or8x16ExplicitMacroblock(mbType uint32) bool {
	if mbType&(MBTypeDirect2|MBTypeSkip|MBType16x16|MBType8x8|MBTypeIntra4x4|MBTypeIntra16x16|MBTypeIntraPCM) != 0 {
		return false
	}
	if mbType&MBType16x8 != 0 {
		return mbType&MBType8x16 == 0 &&
			mbType&(MBTypeP0L0|MBTypeP0L1) != 0 &&
			mbType&(MBTypeP1L0|MBTypeP1L1) != 0
	}
	return mbType&MBType8x16 != 0 &&
		mbType&(MBTypeP0L0|MBTypeP0L1) != 0 &&
		mbType&(MBTypeP1L0|MBTypeP1L1) != 0
}

func isHighB8x8ExplicitSubMacroblock(mbType uint32, subMBType *[4]uint32) bool {
	if subMBType == nil || !isHighB8x8DirectSubCarrier(mbType) {
		return false
	}
	for i := 0; i < 4; i++ {
		if !isHighBExplicitSubMBType(subMBType[i]) {
			return false
		}
	}
	return true
}

func isHighBExplicitSubMBType(subType uint32) bool {
	switch subType {
	case MBType16x16 | MBTypeP0L0,
		MBType16x16 | MBTypeP0L1,
		MBType16x16 | MBTypeP0L0 | MBTypeP0L1,
		MBType16x8 | MBTypeP0L0 | MBTypeP1L0,
		MBType8x16 | MBTypeP0L0 | MBTypeP1L0,
		MBType16x8 | MBTypeP0L1 | MBTypeP1L1,
		MBType8x16 | MBTypeP0L1 | MBTypeP1L1,
		MBType16x8 | MBTypeP0L0 | MBTypeP0L1 | MBTypeP1L0 | MBTypeP1L1,
		MBType8x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeP1L0 | MBTypeP1L1,
		MBType8x8 | MBTypeP0L0 | MBTypeP1L0,
		MBType8x8 | MBTypeP0L1 | MBTypeP1L1,
		MBType8x8 | MBTypeP0L0 | MBTypeP0L1 | MBTypeP1L0 | MBTypeP1L1:
		return true
	default:
		return false
	}
}

func isHighB8x8DirectSubMacroblock(mbType uint32, subMBType *[4]uint32) bool {
	if subMBType == nil || !(isHighB8x8DirectSubCarrier(mbType) || isHighB8x8TopLevelDirectMacroblock(mbType)) {
		return false
	}
	for i := 0; i < 4; i++ {
		if !isHighBResolvedDirectSubMBType(subMBType[i]) {
			return false
		}
	}
	return true
}

func isHighBResolvedDirectSubMBType(subType uint32) bool {
	switch subType {
	case MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2,
		MBType16x16 | MBTypeL0L1 | MBTypeDirect2,
		MBType8x8 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2,
		MBType8x8 | MBTypeL0L1 | MBTypeDirect2:
		return true
	default:
		return false
	}
}

func isHighB8x8TopLevelDirectMacroblock(mbType uint32) bool {
	return mbType == MBType8x8|MBTypeL0L1|MBTypeDirect2
}

func isHighB16x16DirectMacroblock(mbType uint32) bool {
	const spatial = MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2
	const temporal = MBType16x16 | MBTypeL0L1 | MBTypeDirect2
	return mbType == spatial || mbType == temporal
}

func isHighB16x16DirectSkipMacroblock(mbType uint32) bool {
	if mbType&MBTypeSkip == 0 {
		return false
	}
	return isHighB16x16DirectMacroblock(mbType &^ MBTypeSkip)
}

func h264SimpleFrameSliceDecodeSupportsBitDepth(bitDepth int32) bool {
	// High-depth entropy paths exist, but this simple slice loop still feeds
	// 8-bit reconstruction/loop-filter state.
	return bitDepth == 8
}

func h264FrameMBAFFReconstructView(dst *h264PicturePlanes, cur sliceMacroblockCursor, mbType uint32, refs [2][]*h264PicturePlanes, refPlanes *[2][32]h264PicturePlanes, refPtrs *[2][32]*h264PicturePlanes) (h264PicturePlanes, int, [2][]*h264PicturePlanes, error) {
	if dst == nil {
		return h264PicturePlanes{}, 0, refs, ErrInvalidData
	}
	view := *dst
	mbY := cur.PixelMBY
	if !cur.FrameMBAFF || mbType&MBTypeInterlaced == 0 {
		return view, mbY, refs, nil
	}
	if refPlanes == nil || refPtrs == nil || cur.MBY < 0 {
		return h264PicturePlanes{}, 0, refs, ErrInvalidData
	}
	pictureStructure := PictureTopField
	if cur.MBY&1 != 0 {
		pictureStructure = PictureBottomField
	}
	applySimpleFieldRefPlane(&view, pictureStructure)
	fieldRefs, err := h264MBAFFFieldRefViews(refs, cur.MBY&1, refPlanes, refPtrs)
	if err != nil {
		return h264PicturePlanes{}, 0, refs, err
	}
	return view, cur.MBY >> 1, fieldRefs, nil
}

func h264MBAFFFieldRefViews(refs [2][]*h264PicturePlanes, parity int, planes *[2][32]h264PicturePlanes, ptrs *[2][32]*h264PicturePlanes) ([2][]*h264PicturePlanes, error) {
	var out [2][]*h264PicturePlanes
	if planes == nil || ptrs == nil || parity < 0 || parity > 1 {
		return out, ErrInvalidData
	}
	for list := 0; list < 2; list++ {
		if len(refs[list])*2 > len(planes[list]) {
			return out, ErrUnsupported
		}
		for ref := 0; ref < len(refs[list])*2; ref++ {
			mapped := ref ^ parity
			frameIndex := mapped >> 1
			if refs[list][frameIndex] == nil {
				return out, ErrInvalidData
			}
			planes[list][ref] = *refs[list][frameIndex]
			pictureStructure := PictureTopField
			if mapped&1 != 0 {
				pictureStructure = PictureBottomField
			}
			applySimpleFieldRefPlane(&planes[list][ref], pictureStructure)
			ptrs[list][ref] = &planes[list][ref]
		}
		out[list] = ptrs[list][:len(refs[list])*2]
	}
	return out, nil
}

func h264FrameMBReconstructInputFromCAVLC(sh *SliceHeader, cur sliceMacroblockCursor, mb cavlcFrameMacroblockResult, work *frameMacroblockDecodeWork, in h264FrameSliceDecodeInput) h264FrameMBReconstructInput {
	listCount, _ := cavlcFrameListCount(sh.SliceTypeNoS)
	return h264FrameMBReconstructInput{
		MBType:             mb.MBType,
		SubMBType:          mb.Inter.SubMBType,
		MBX:                cur.MBX,
		MBY:                cur.PixelMBY,
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
		MotionWeightMBY:    cur.MBY,
		UseMotionWeightMBY: true,
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
		MBY:                cur.PixelMBY,
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
		MotionWeightMBY:    cur.MBY,
		UseMotionWeightMBY: true,
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
		MBY:                cur.PixelMBY,
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
		MotionWeightMBY:    cur.MBY,
		UseMotionWeightMBY: true,
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
		MBY:                cur.PixelMBY,
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
		MotionWeightMBY:    cur.MBY,
		UseMotionWeightMBY: true,
		MotionScratch:      in.MotionScratch,
		TransformBypass:    sh.SPS.TransformBypass != 0 && mb.QScale == 0,
		IntraPCM:           mb.IntraPCM,
	}
}
