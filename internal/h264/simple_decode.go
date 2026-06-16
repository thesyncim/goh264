// SPDX-License-Identifier: LGPL-2.1-or-later
//
// Source-shaped simple-frame integration around FFmpeg n8.0.1
// libavcodec/h264dec.c decode_nal_units/ff_h264_queue_decode_slice and
// libavcodec/h264_slice.c decode_slice. This is intentionally limited to the
// simple frame-picture subset whose macroblock decode/reconstruct path is
// already translated.

package h264

import "fmt"

type DecodedFrame struct {
	Y, Cb, Cr                      []uint8
	Y16, Cb16, Cr16                []uint16
	LumaStride                     int
	ChromaStride                   int
	Width                          int
	Height                         int
	CropLeft                       int
	CropTop                        int
	MBWidth                        int
	MBHeight                       int
	ChromaFormatIDC                int
	BitDepthLuma                   int
	BitDepthChroma                 int
	SARNum                         int32
	SARDen                         int32
	VideoFormat                    int32
	VideoFullRangeFlag             int32
	ColorPrimaries                 int32
	ColorTransfer                  int32
	ColorMatrix                    int32
	ChromaLocation                 int32
	ChromaSampleLocTypeTopField    int32
	ChromaSampleLocTypeBottomField int32
	TimingInfoPresentFlag          int32
	NumUnitsInTick                 uint32
	TimeScale                      uint32
	FixedFrameRateFlag             int32
	RepeatPict                     int
	InterlacedFrame                bool
	TopFieldFirst                  bool
	KeyFrame                       bool
	SideData                       DecodedFrameSideData
	frameNum                       uint32
	fieldPOC                       [2]int32
	poc                            int32
	idrKeyFrame                    bool
	mmcoReset                      bool
	recovered                      uint8
	frameMBSOnlyFlag               int32
	fieldPicture                   bool
	mbaff                          bool
	tables                         *macroblockTables
	refEntries                     [2][]simpleRefEntry
	fieldRefEntries                [2][2][]simpleRefEntry
	invalidGap                     bool
}

type DecodedFrameSideData struct {
	UserDataUnregistered [][]uint8
	A53ClosedCaptions    []uint8
	X264Build            int32
	PictureTiming        H264SEIPictureTiming
	S12MTimecodes        []uint32
	RecoveryPoint        H264SEIRecoveryPoint
	BufferingPeriod      H264SEIBufferingPeriod
	GreenMetadata        H264SEIGreenMetadata
	AFD                  H2645SEIAFD
	FramePacking         H2645SEIFramePacking
	Stereo3D             AVStereo3D
	Spherical            AVSphericalMapping
	DisplayMatrix        AVDisplayMatrix
	DisplayOrientation   H2645SEIDisplayOrientation
	AlternativeTransfer  H2645SEIAlternativeTransfer
	AmbientViewing       H2645SEIAmbientViewingEnvironment
	FilmGrain            H2645SEIFilmGrainCharacteristics
	MasteringMetadata    AVMasteringDisplayMetadata
	MasteringDisplay     H2645SEIMasteringDisplay
	ContentLight         H2645SEIContentLight
	ICCProfile           []uint8
	DynamicHDR10Plus     []uint8
	LCEVC                []uint8
	ReferenceDisplays    AV3DReferenceDisplaysInfo
}

type SimpleDecoder struct {
	sps [maxSPSCount]*SPS
	pps [maxPPSCount]*PPS
	dpb simpleFrameDPB
	sei H264SEIContext
	st  simpleDecodeState
}

type simpleDecodeState struct {
	frame                 *DecodedFrame
	tables                *macroblockTables
	motionScratch         *h264MotionCompScratch
	motionScratchHigh     *h264MotionCompScratchHigh
	loopFilterSlices      []h264LoopFilterSliceParams
	loopFilterRefFrameIDs map[*DecodedFrame]int8
	sliceNum              uint16
	haveSlice             bool
	frameComplete         bool
	fieldPairPending      bool
	pendingFieldStructure int32
	pendingFieldFrameNum  uint32
}

func (s *simpleDecodeState) resetPicture() {
	if s == nil {
		return
	}
	s.frame = nil
	s.tables = nil
	s.motionScratch = nil
	s.motionScratchHigh = nil
	s.loopFilterSlices = nil
	s.loopFilterRefFrameIDs = nil
	s.sliceNum = 0
	s.haveSlice = false
	s.frameComplete = false
	s.fieldPairPending = false
	s.pendingFieldStructure = 0
	s.pendingFieldFrameNum = 0
}

func (s *simpleDecodeState) hasPendingComplementaryField() bool {
	return s != nil && s.haveSlice && !s.frameComplete && s.fieldPairPending
}

func (d *SimpleDecoder) StoreAVCDecoderConfiguration(cfg AVCDecoderConfigurationRecord) error {
	if d == nil || cfg.NALLengthSize < 1 || cfg.NALLengthSize > 4 {
		return ErrInvalidData
	}
	return d.StoreParamSets(cfg.SPS, cfg.PPS)
}

func (d *SimpleDecoder) StoreParamSets(sps [maxSPSCount]*SPS, pps [maxPPSCount]*PPS) error {
	if d == nil {
		return ErrInvalidData
	}
	d.sps = sps
	d.pps = pps
	d.dpb.reset()
	d.sei.Reset()
	d.st.resetPicture()
	return nil
}

func (d *SimpleDecoder) UpdateParamSets(sps [maxSPSCount]*SPS, pps [maxPPSCount]*PPS) error {
	if d == nil {
		return ErrInvalidData
	}
	d.sps = sps
	d.pps = pps
	return nil
}

func (d *SimpleDecoder) DecodeNALUnits(nals []NALUnit) ([]*DecodedFrame, error) {
	if d == nil {
		return nil, ErrInvalidData
	}
	return d.DecodeNALUnitsWithSideData(nals, DecodedFrameSideData{})
}

func (d *SimpleDecoder) DecodeNALUnitsWithSideData(nals []NALUnit, packetSideData DecodedFrameSideData) ([]*DecodedFrame, error) {
	if d == nil {
		return nil, ErrInvalidData
	}
	return decodeSimpleNALUnitsWithDecoderState(nals, &d.sps, &d.pps, &d.dpb, &d.sei, &d.st, packetSideData, false)
}

func (d *SimpleDecoder) DecodeAVCFrames(data []byte, nalLengthSize int) ([]*DecodedFrame, error) {
	if d == nil {
		return nil, ErrInvalidData
	}
	nals, err := SplitAVCC(data, nalLengthSize)
	if err != nil {
		return nil, err
	}
	return d.DecodeNALUnits(nals)
}

func (d *SimpleDecoder) FlushDelayedFrames() ([]*DecodedFrame, error) {
	if d == nil {
		return nil, ErrInvalidData
	}
	frames, err := d.dpb.drainOutputFrames(true)
	d.sei.Reset()
	return frames, err
}

func (d *SimpleDecoder) FlushDelayedFrame() (*DecodedFrame, error) {
	if d == nil {
		return nil, ErrInvalidData
	}
	snap := d.dpb.snapshot()
	frames, err := d.dpb.drainOutputFrames(true)
	if len(frames) == 1 {
		return frames[0], err
	}
	d.dpb.restore(snap)
	if err != nil {
		return nil, err
	}
	return nil, ErrUnsupported
}

func (d *SimpleDecoder) DecodeAVCFramesWithConfig(data []byte, cfg AVCDecoderConfigurationRecord) ([]*DecodedFrame, error) {
	if err := d.StoreAVCDecoderConfiguration(cfg); err != nil {
		return nil, err
	}
	nals, err := SplitAVCC(data, cfg.NALLengthSize)
	if err != nil {
		return nil, err
	}
	d.sei.Reset()
	return decodeSimpleNALUnitsWithDecoderState(nals, &d.sps, &d.pps, &d.dpb, &d.sei, &d.st, DecodedFrameSideData{}, true)
}

func DecodeAnnexBSimple(data []byte) (*DecodedFrame, error) {
	frames, err := DecodeAnnexBSimpleFrames(data)
	if err != nil {
		return nil, err
	}
	if len(frames) != 1 {
		return nil, ErrUnsupported
	}
	return frames[0], nil
}

func DecodeAnnexBSimpleFrames(data []byte) ([]*DecodedFrame, error) {
	nals, err := SplitAnnexB(data)
	if err != nil {
		return nil, err
	}
	return DecodeSimpleNALUnits(nals)
}

func DecodeAVCSimpleFrames(data []byte, nalLengthSize int) ([]*DecodedFrame, error) {
	nals, err := SplitAVCC(data, nalLengthSize)
	if err != nil {
		return nil, err
	}
	return DecodeSimpleNALUnits(nals)
}

func DecodeAVCSimpleFramesWithConfigurationRecord(config []byte, data []byte) ([]*DecodedFrame, error) {
	cfg, err := DecodeAVCDecoderConfigurationRecord(config)
	if err != nil {
		return nil, err
	}
	return DecodeAVCSimpleFramesWithConfig(data, cfg)
}

func DecodeAVCSimpleFramesWithConfig(data []byte, cfg AVCDecoderConfigurationRecord) ([]*DecodedFrame, error) {
	var dec SimpleDecoder
	return dec.DecodeAVCFramesWithConfig(data, cfg)
}

func DecodeSimpleNALUnits(nals []NALUnit) ([]*DecodedFrame, error) {
	return DecodeSimpleNALUnitsWithParamSets(nals, [maxSPSCount]*SPS{}, [maxPPSCount]*PPS{})
}

func DecodeSimpleNALUnitsWithParamSets(nals []NALUnit, spsList [maxSPSCount]*SPS, ppsList [maxPPSCount]*PPS) ([]*DecodedFrame, error) {
	var dpb simpleFrameDPB
	var sei H264SEIContext
	dpb.reset()
	sei.Reset()
	return decodeSimpleNALUnitsWithState(nals, &spsList, &ppsList, &dpb, &sei, DecodedFrameSideData{}, true)
}

func decodeSimpleNALUnitsWithState(nals []NALUnit, spsList *[maxSPSCount]*SPS, ppsList *[maxPPSCount]*PPS, dpb *simpleFrameDPB, sei *H264SEIContext, packetSideData DecodedFrameSideData, flushOutput bool) ([]*DecodedFrame, error) {
	var st simpleDecodeState
	return decodeSimpleNALUnitsWithDecoderState(nals, spsList, ppsList, dpb, sei, &st, packetSideData, flushOutput)
}

func decodeSimpleNALUnitsWithDecoderState(nals []NALUnit, spsList *[maxSPSCount]*SPS, ppsList *[maxPPSCount]*PPS, dpb *simpleFrameDPB, sei *H264SEIContext, st *simpleDecodeState, packetSideData DecodedFrameSideData, flushOutput bool) ([]*DecodedFrame, error) {
	if spsList == nil || ppsList == nil || dpb == nil || sei == nil || st == nil {
		return nil, ErrInvalidData
	}
	if st.frameComplete {
		st.resetPicture()
	}
	if flushOutput {
		dpb.primeOutputReorderDelayFromNALs(nals, spsList, ppsList)
	}
	var frames []*DecodedFrame
	decodedFrames := 0
	acceptedStateOnlyNAL := false
	currentIDRSegmentOutputIndex := -1

	for nalIndex, nal := range nals {
		switch nal.Type {
		case NALSPS:
			sps, err := DecodeSPSFromNAL(nal)
			if err != nil {
				// FFmpeg keeps malformed parameter-set NALs non-fatal unless
				// AV_EF_EXPLODE is set; previously parsed sets stay active.
				continue
			}
			spsList[sps.SPSID] = sps
			acceptedStateOnlyNAL = true
		case NALPPS:
			pps, err := DecodePPS(nal.RBSP, spsList)
			if err != nil {
				// FFmpeg keeps malformed parameter-set NALs non-fatal unless
				// AV_EF_EXPLODE is set; previously parsed sets stay active.
				continue
			}
			ppsList[pps.PPSID] = pps
			acceptedStateOnlyNAL = true
		case NALSEI:
			if st.frameComplete {
				st.resetPicture()
			} else if st.haveSlice {
				continue
			}
			// FFmpeg keeps SEI parse failures non-fatal unless AV_EF_EXPLODE is set.
			if err := sei.Decode(nal.RBSP, spsList); err == nil {
				acceptedStateOnlyNAL = true
			}
		case NALAUD, NALEndSequence, NALEndStream, NALFillerData:
			acceptedStateOnlyNAL = true
		case NALSlice, NALIDRSlice:
			dpbSnapshot := dpb.snapshot()
			returnSliceError := func(err error) ([]*DecodedFrame, error) {
				st.resetPicture()
				dpb.restore(dpbSnapshot)
				return returnFramesWithDecodeError(frames, dpb, flushOutput, err)
			}
			sh, payload, err := parseSliceHeaderWithPayload(nal, ppsList)
			if err != nil {
				return returnSliceError(err)
			}
			if sh.RedundantPicCount != 0 {
				continue
			}
			if sh.SliceTypeNoS != PictureTypeI && sh.SliceTypeNoS != PictureTypeP && sh.SliceTypeNoS != PictureTypeB {
				return returnSliceError(ErrUnsupported)
			}
			if err := validateSimpleFrameReferenceSyntax(sh); err != nil {
				return returnSliceError(fmt.Errorf("validate simple frame reference syntax: %w", err))
			}
			fieldPicture := sh.PictureStructure != PictureFrame
			samePendingFieldFrame := st.fieldPairPending &&
				fieldPicture &&
				sh.FrameNum == st.pendingFieldFrameNum &&
				sh.PictureStructure != st.pendingFieldStructure
			startingComplementaryField := st.haveSlice &&
				!st.frameComplete &&
				sh.FirstMBAddr == 0 &&
				samePendingFieldFrame
			decodingComplementaryField := st.haveSlice && !st.frameComplete && samePendingFieldFrame
			if st.frameComplete || st.haveSlice && sh.FirstMBAddr == 0 && !startingComplementaryField {
				if sh.FirstMBAddr != 0 {
					return returnSliceError(ErrInvalidData)
				}
				st.resetPicture()
			}
			if !st.haveSlice && sh.FirstMBAddr != 0 {
				return returnSliceError(ErrInvalidData)
			}
			if st.frame == nil {
				if err := dpb.handleFrameNumGaps(sh, false); err != nil {
					return returnSliceError(err)
				}
				if sei.PictureTiming.Present != 0 {
					if err := sei.PictureTiming.Process(sh.SPS); err != nil {
						// FFmpeg drops malformed picture-timing SEI without AV_EF_EXPLODE.
						sei.PictureTiming.Present = 0
						sei.PictureTiming.TimecodeCount = 0
					}
				}
				st.frame, st.tables, err = newSimpleDecodedFrame(sh.SPS)
				if err != nil {
					return returnSliceError(err)
				}
				st.frame.SideData = decodedFrameSideDataFromSEI(sei)
				mergePacketSideDataIntoDecodedFrame(&st.frame.SideData, packetSideData)
				if err := dpb.initFramePOC(st.frame, sh, nal.RefIDC); err != nil {
					return returnSliceError(err)
				}
				st.frame.fieldPicture = fieldPicture
				st.frame.mbaff = sh.PictureStructure == PictureFrame && sh.SPS.MBAFF != 0
				applySimpleFrameTimingProps(st.frame, sh.SPS, sei, dpb)
				dpb.applySimpleRecoveryPoint(st.frame, sh, nal.RefIDC, sei)
				consumeFrameSideDataFromSEI(sei)
				if st.frame.BitDepthLuma == 8 {
					st.motionScratch = newH264MotionCompScratchForFrame(st.frame)
				} else {
					st.motionScratchHigh = newH264MotionCompScratchHighForFrame(st.frame)
				}
				st.loopFilterRefFrameIDs = make(map[*DecodedFrame]int8)
			} else if err := st.frame.matchesSPS(sh.SPS); err != nil {
				return returnSliceError(err)
			} else if startingComplementaryField {
				if err := dpb.initFramePOC(st.frame, sh, nal.RefIDC); err != nil {
					return returnSliceError(err)
				}
				applySimpleFrameTimingProps(st.frame, sh.SPS, sei, dpb)
			}

			st.sliceNum++
			if st.sliceNum == ^uint16(0) {
				return returnSliceError(ErrInvalidData)
			}
			for len(st.loopFilterSlices) <= int(st.sliceNum) {
				st.loopFilterSlices = append(st.loopFilterSlices, h264LoopFilterSliceParams{})
			}
			st.loopFilterSlices[st.sliceNum] = h264LoopFilterSliceParamsFromHeader(sh)
			refctx, err := dpb.buildRefContext(sh, st.frame)
			if err != nil {
				return returnSliceError(fmt.Errorf("build simple ref context slice=%d type=%d frame_num=%d refs=%d/%d mods=%d/%d picture=%d: %w",
					st.sliceNum, sh.SliceTypeNoS, sh.FrameNum, sh.RefCount[0], sh.RefCount[1],
					sh.NBRefModifications[0], sh.NBRefModifications[1], sh.PictureStructure, err))
			}
			st.loopFilterSlices[st.sliceNum].Ref2Frame, err = h264LoopFilterRef2Frame(refctx.Entries, st.loopFilterRefFrameIDs)
			if err != nil {
				return returnSliceError(err)
			}
			st.frame.saveRefEntries(refctx.Entries, sh.PictureStructure)
			var result h264FrameSliceDecodeResult
			completeFrameNow := false
			direct := refctx.directMotionContext(st.frame, sh, sei)
			if st.frame.BitDepthLuma == 8 {
				pic := st.frame.picturePlanes()
				result, err = st.tables.decodeFrameSliceData(&payload, &pic, sh, h264FrameSliceDecodeInput{
					SliceNum:      st.sliceNum,
					Refs:          refctx.Refs,
					Direct:        direct,
					PredWeight:    &sh.PredWeightTable,
					MotionScratch: st.motionScratch,
					X264Build:     direct.X264Build,
					X264BuildSet:  true,
				})
				completeFrameNow = result.EndOfFrame && (!fieldPicture || decodingComplementaryField)
				if err == nil && result.EndOfFrame {
					if fieldPicture {
						err = st.tables.filterField(&pic, st.loopFilterSlices, sh.PictureStructure)
					} else if completeFrameNow {
						err = st.tables.filterFrame(&pic, st.loopFilterSlices)
					}
					if err != nil {
						err = fmt.Errorf("filter 8-bit frame slice=%d type=%d: %w", st.sliceNum, sh.SliceTypeNoS, err)
					}
				}
			} else {
				pic := st.frame.picturePlanesHigh()
				result, err = st.tables.decodeFrameSliceDataHigh(&payload, &pic, sh, h264FrameSliceDecodeInputHigh{
					SliceNum:      st.sliceNum,
					Refs:          refctx.RefsHigh,
					Direct:        direct,
					PredWeight:    &sh.PredWeightTable,
					MotionScratch: st.motionScratchHigh,
					X264Build:     direct.X264Build,
					X264BuildSet:  true,
				})
				completeFrameNow = result.EndOfFrame && (!fieldPicture || decodingComplementaryField)
				if err == nil && completeFrameNow {
					err = st.tables.filterFrameHigh(&pic, st.loopFilterSlices)
					if err != nil {
						err = fmt.Errorf("filter high-bit frame slice=%d type=%d: %w", st.sliceNum, sh.SliceTypeNoS, err)
					}
				}
			}
			if err != nil {
				if canDropTerminalDamagedFieldSlice(nals, nalIndex, flushOutput, fieldPicture, decodingComplementaryField) {
					out, drainErr := dpb.drainOutputFrames(true)
					if drainErr != nil {
						return returnSliceError(drainErr)
					}
					frames = append(frames, out...)
					st.resetPicture()
					if len(frames) == 0 {
						return nil, ErrInvalidData
					}
					return frames, nil
				}
				return returnSliceError(fmt.Errorf("decode slice=%d type=%d first_mb=%d frame_num=%d picture=%d refs=%d/%d: %w",
					st.sliceNum, sh.SliceTypeNoS, sh.FirstMBAddr, sh.FrameNum, sh.PictureStructure,
					sh.RefCount[0], sh.RefCount[1], err))
			}
			if result.EndOfFrame && fieldPicture && !decodingComplementaryField {
				st.fieldPairPending = true
				st.pendingFieldStructure = sh.PictureStructure
				st.pendingFieldFrameNum = sh.FrameNum
				if err := dpb.markDecodedFrame(st.frame, sh, nal.RefIDC); err != nil {
					return returnSliceError(err)
				}
			}
			if completeFrameNow {
				st.frameComplete = true
				st.fieldPairPending = false
				st.pendingFieldStructure = 0
				st.pendingFieldFrameNum = 0
				decodedFrames++
				if err := dpb.markDecodedFrame(st.frame, sh, nal.RefIDC); err != nil {
					return returnSliceError(err)
				}
				if err := dpb.holdOutputFrame(st.frame, sh); err != nil {
					return returnSliceError(err)
				}
				out, err := dpb.drainOutputFrames(false)
				if err != nil {
					return returnSliceError(err)
				}
				for i, frame := range out {
					if frame != nil && frame.idrKeyFrame {
						currentIDRSegmentOutputIndex = len(frames) + i
					}
				}
				frames = append(frames, out...)
			}
			st.haveSlice = true
		default:
			continue
		}
	}

	if st.haveSlice && !st.frameComplete {
		if st.hasPendingComplementaryField() && (!flushOutput || decodedFrames != 0) {
			// A terminal first field is not a completed picture. If a later
			// IDR segment already emitted delayed frames, keep the earlier
			// segment output and leave the partial terminal segment unpresented.
			if flushOutput && currentIDRSegmentOutputIndex > 0 {
				return frames[:currentIDRSegmentOutputIndex], nil
			}
			return frames, nil
		}
		return returnFramesWithDecodeError(frames, dpb, flushOutput, ErrInvalidData)
	}
	if flushOutput {
		out, err := dpb.drainOutputFrames(true)
		if err != nil {
			return nil, err
		}
		frames = append(frames, out...)
	}
	if decodedFrames == 0 {
		if !flushOutput && acceptedStateOnlyNAL {
			return frames, nil
		}
		return nil, ErrInvalidData
	}
	return frames, nil
}

func canDropTerminalDamagedFieldSlice(nals []NALUnit, nalIndex int, flushOutput bool, fieldPicture bool, decodingComplementaryField bool) bool {
	// FFmpeg's default error-resilience path does not present a terminal first
	// field without its complementary field. If the final VCL slice for that
	// first field is damaged, drain already-complete delayed frames and drop the
	// partial field rather than promoting it to a decoded picture.
	if !flushOutput || !fieldPicture || decodingComplementaryField || nalIndex < 0 || nalIndex >= len(nals) {
		return false
	}
	for i := nalIndex + 1; i < len(nals); i++ {
		if nals[i].Type == NALSlice || nals[i].Type == NALIDRSlice {
			return false
		}
	}
	return true
}

func returnFramesWithDecodeError(frames []*DecodedFrame, dpb *simpleFrameDPB, flushOutput bool, err error) ([]*DecodedFrame, error) {
	if dpb != nil {
		out, drainErr := dpb.drainOutputFrames(flushOutput)
		frames = append(frames, out...)
		if drainErr != nil {
			if err != nil {
				err = fmt.Errorf("%v; drain output: %w", err, drainErr)
			} else {
				err = drainErr
			}
		}
	}
	if len(frames) != 0 {
		return frames, err
	}
	return nil, err
}

func decodedFrameSideDataFromSEI(sei *H264SEIContext) DecodedFrameSideData {
	if sei == nil {
		return DecodedFrameSideData{}
	}
	return DecodedFrameSideData{
		UserDataUnregistered: cloneByteSlices(sei.Common.Unregistered.Data),
		A53ClosedCaptions:    cloneByteSlice(sei.Common.A53Caption.Data),
		X264Build:            sei.Common.Unregistered.X264Build,
		PictureTiming:        sei.PictureTiming,
		RecoveryPoint:        sei.RecoveryPoint,
		BufferingPeriod:      sei.BufferingPeriod,
		GreenMetadata:        sei.GreenMetadata,
		AFD:                  sei.Common.AFD,
		FramePacking:         sei.Common.FramePacking,
		DisplayOrientation:   sei.Common.DisplayOrientation,
		AlternativeTransfer:  sei.Common.AlternativeTransfer,
		AmbientViewing:       sei.Common.AmbientViewing,
		FilmGrain:            sei.Common.FilmGrain,
		MasteringDisplay:     sei.Common.MasteringDisplay,
		ContentLight:         sei.Common.ContentLight,
		LCEVC:                cloneByteSlice(sei.Common.LCEVC.Data),
	}
}

func mergePacketSideDataIntoDecodedFrame(dst *DecodedFrameSideData, src DecodedFrameSideData) {
	if dst == nil {
		return
	}
	if len(src.A53ClosedCaptions) != 0 {
		dst.A53ClosedCaptions = cloneByteSlice(src.A53ClosedCaptions)
	}
	if src.AFD.Present != 0 {
		dst.AFD = src.AFD
	}
	if len(src.S12MTimecodes) != 0 && dst.PictureTiming.TimecodeCount == 0 {
		dst.S12MTimecodes = cloneUint32Slice(src.S12MTimecodes)
	}
	if src.Stereo3D.Present != 0 {
		dst.Stereo3D = src.Stereo3D
	}
	if src.Spherical.Present != 0 && dst.Spherical.Present == 0 {
		dst.Spherical = src.Spherical
	}
	if src.DisplayMatrix.Present != 0 {
		dst.DisplayMatrix = src.DisplayMatrix
	}
	if src.AmbientViewing.Present != 0 && dst.AmbientViewing.Present == 0 {
		dst.AmbientViewing = src.AmbientViewing
	}
	if src.MasteringMetadata.Present != 0 && dst.MasteringDisplay.Present == 0 && dst.MasteringMetadata.Present == 0 {
		dst.MasteringMetadata = src.MasteringMetadata
	}
	if src.ContentLight.Present != 0 && dst.ContentLight.Present == 0 {
		dst.ContentLight = src.ContentLight
	}
	if len(src.ICCProfile) != 0 && len(dst.ICCProfile) == 0 {
		dst.ICCProfile = cloneByteSlice(src.ICCProfile)
	}
	if len(src.DynamicHDR10Plus) != 0 && len(dst.DynamicHDR10Plus) == 0 {
		dst.DynamicHDR10Plus = cloneByteSlice(src.DynamicHDR10Plus)
	}
	if len(src.LCEVC) != 0 && len(dst.LCEVC) == 0 {
		dst.LCEVC = cloneByteSlice(src.LCEVC)
	}
	if src.ReferenceDisplays.Present != 0 && dst.ReferenceDisplays.Present == 0 {
		dst.ReferenceDisplays = cloneReferenceDisplays(src.ReferenceDisplays)
	}
}

func cloneReferenceDisplays(src AV3DReferenceDisplaysInfo) AV3DReferenceDisplaysInfo {
	if len(src.Displays) > maxInt/16 {
		src.Displays = nil
		return src
	}
	src.Displays = append([]AV3DReferenceDisplay(nil), src.Displays...)
	return src
}

func consumeFrameSideDataFromSEI(sei *H264SEIContext) {
	if sei == nil {
		return
	}
	sei.Common.Unregistered.Data = nil
	sei.Common.A53Caption.Data = nil
	sei.Common.AFD.Present = 0
	sei.Common.LCEVC.Data = nil
	if sei.Common.FilmGrain.Present != 0 && sei.Common.FilmGrain.RepetitionPeriod == 0 {
		sei.Common.FilmGrain.Present = 0
	}
	sei.PictureTiming.Present = 0
	sei.PictureTiming.TimecodeCount = 0
	sei.RecoveryPoint.RecoveryFrameCount = -1
}

func cloneByteSlices(src [][]uint8) [][]uint8 {
	if len(src) == 0 || len(src) > maxInt/32 {
		return nil
	}
	out := make([][]uint8, len(src))
	for i := range src {
		out[i] = cloneByteSlice(src[i])
	}
	return out
}

func cloneByteSlice(src []uint8) []uint8 {
	if len(src) == 0 || len(src) > maxInt/2 {
		return nil
	}
	return append([]uint8(nil), src...)
}

func cloneUint32Slice(src []uint32) []uint32 {
	if len(src) == 0 || len(src) > maxInt/4 {
		return nil
	}
	return append([]uint32(nil), src...)
}

// applySimpleFrameTimingProps mirrors FFmpeg n8.0.1 h264_export_frame_props
// for the simple frame-picture path. Field-coded MBAFF/PAFF decoding remains
// unsupported, but the public frame flags still follow picture-timing SEI.
func applySimpleFrameTimingProps(frame *DecodedFrame, sps *SPS, sei *H264SEIContext, dpb *simpleFrameDPB) {
	if frame == nil || sps == nil {
		return
	}

	interlacedFrame := false
	topFieldFirst := false
	repeatPict := 0
	timingPresent := sei != nil && sei.PictureTiming.Present != 0
	if sps.PicStructPresentFlag != 0 && timingPresent {
		pt := &sei.PictureTiming
		switch pt.PicStruct {
		case h264SEIPicStructFrame:
		case h264SEIPicStructTopField, h264SEIPicStructBottomField:
			interlacedFrame = true
		case h264SEIPicStructTopBottom, h264SEIPicStructBottomTop:
			interlacedFrame = dpb.previousInterlacedFrame()
		case h264SEIPicStructTopBottomTop, h264SEIPicStructBottomTopBottom:
			repeatPict = 1
		case h264SEIPicStructFrameDoubling:
			repeatPict = 2
		case h264SEIPicStructFrameTripling:
			repeatPict = 4
		}

		if (pt.CTType&3) != 0 && pt.PicStruct <= h264SEIPicStructBottomTop {
			interlacedFrame = (pt.CTType & (1 << 1)) != 0
		}
	}
	if dpb != nil {
		dpb.setPreviousInterlacedFrame(interlacedFrame)
	}

	if frame.fieldPOC[0] != frame.fieldPOC[1] {
		topFieldFirst = frame.fieldPOC[0] < frame.fieldPOC[1]
	} else if sps.PicStructPresentFlag != 0 && timingPresent {
		if sei.PictureTiming.PicStruct == h264SEIPicStructTopBottom ||
			sei.PictureTiming.PicStruct == h264SEIPicStructTopBottomTop {
			topFieldFirst = true
		}
	} else if interlacedFrame {
		topFieldFirst = true
	}

	frame.RepeatPict = repeatPict
	frame.InterlacedFrame = interlacedFrame
	frame.TopFieldFirst = topFieldFirst
}

func newSimpleDecodedFrame(sps *SPS) (*DecodedFrame, *macroblockTables, error) {
	if sps == nil {
		return nil, nil, ErrInvalidData
	}
	if sps.FrameMBSOnlyFlag != 0 && sps.MBAFF != 0 {
		return nil, nil, ErrUnsupported
	}
	if sps.BitDepthLuma != sps.BitDepthChroma {
		return nil, nil, ErrUnsupported
	}
	if sps.BitDepthLuma != 8 {
		if err := checkH264DSPHighBitDepth(int(sps.BitDepthLuma)); err != nil {
			return nil, nil, err
		}
	}
	if sps.ChromaFormatIDC > 3 {
		return nil, nil, ErrUnsupported
	}
	mbWidth := int(sps.MBWidth)
	mbHeight := int(sps.MBHeight)
	chromaFormatIDC := int(sps.ChromaFormatIDC)
	if mbWidth <= 0 || mbHeight <= 0 || sps.Width <= 0 || sps.Height <= 0 {
		return nil, nil, ErrInvalidData
	}
	if mbWidth > h264MaxMBWidth || mbHeight > h264MaxMBHeight {
		return nil, nil, ErrInvalidData
	}
	lumaStride, err := checkedMulInt(mbWidth, 16)
	if err != nil {
		return nil, nil, err
	}
	lumaHeight, err := checkedMulInt(mbHeight, 16)
	if err != nil {
		return nil, nil, err
	}
	lumaSamples, err := checkedMulInt(lumaStride, lumaHeight)
	if err != nil {
		return nil, nil, err
	}
	chromaWidth := 0
	chromaSamples := 0
	if chromaFormatIDC != 0 {
		var chromaHeight int
		chromaWidth, chromaHeight, err = h264ChromaFrameSizeChecked(mbWidth, mbHeight, chromaFormatIDC)
		if err != nil {
			return nil, nil, err
		}
		chromaSamples, err = checkedMulInt(chromaWidth, chromaHeight)
		if err != nil {
			return nil, nil, err
		}
	}

	frame := &DecodedFrame{
		LumaStride:                     lumaStride,
		Width:                          int(sps.Width),
		Height:                         int(sps.Height),
		CropLeft:                       int(sps.CropLeft),
		CropTop:                        int(sps.CropTop),
		MBWidth:                        mbWidth,
		MBHeight:                       mbHeight,
		ChromaFormatIDC:                chromaFormatIDC,
		BitDepthLuma:                   int(sps.BitDepthLuma),
		BitDepthChroma:                 int(sps.BitDepthChroma),
		frameMBSOnlyFlag:               sps.FrameMBSOnlyFlag,
		SARNum:                         sps.VUI.SARNum,
		SARDen:                         sps.VUI.SARDen,
		VideoFormat:                    sps.VUI.VideoFormat,
		VideoFullRangeFlag:             sps.VUI.VideoFullRangeFlag,
		ColorPrimaries:                 sps.VUI.ColourPrimaries,
		ColorTransfer:                  sps.VUI.TransferCharacteristics,
		ColorMatrix:                    sps.VUI.MatrixCoeffs,
		ChromaLocation:                 sps.VUI.ChromaLocation,
		ChromaSampleLocTypeTopField:    sps.VUI.ChromaSampleLocTypeTopField,
		ChromaSampleLocTypeBottomField: sps.VUI.ChromaSampleLocTypeBottomField,
		TimingInfoPresentFlag:          sps.TimingInfoPresentFlag,
		NumUnitsInTick:                 sps.NumUnitsInTick,
		TimeScale:                      sps.TimeScale,
		FixedFrameRateFlag:             sps.FixedFrameRateFlag,
	}
	highBitDepth := sps.BitDepthLuma != 8
	if highBitDepth {
		frame.Y16 = make([]uint16, lumaSamples)
	} else {
		frame.Y = make([]uint8, lumaSamples)
	}
	if chromaFormatIDC != 0 {
		frame.ChromaStride = chromaWidth
		if highBitDepth {
			frame.Cb16 = make([]uint16, chromaSamples)
			frame.Cr16 = make([]uint16, chromaSamples)
		} else {
			frame.Cb = make([]uint8, chromaSamples)
			frame.Cr = make([]uint8, chromaSamples)
		}
	}

	tables, err := newMacroblockTables(mbWidth, mbHeight, chromaFormatIDC)
	if err != nil {
		return nil, nil, err
	}
	frame.tables = tables
	if highBitDepth {
		pic := frame.picturePlanesHigh()
		if err := pic.validate(); err != nil {
			return nil, nil, err
		}
	} else {
		pic := frame.picturePlanes()
		if err := pic.validate(); err != nil {
			return nil, nil, err
		}
	}
	return frame, tables, nil
}

func (f *DecodedFrame) picturePlanes() h264PicturePlanes {
	if f == nil {
		return h264PicturePlanes{}
	}
	return h264PicturePlanes{
		Y:                f.Y,
		Cb:               f.Cb,
		Cr:               f.Cr,
		LumaStride:       f.LumaStride,
		ChromaStride:     f.ChromaStride,
		MBWidth:          f.MBWidth,
		MBHeight:         f.MBHeight,
		ChromaFormatIDC:  f.ChromaFormatIDC,
		PictureStructure: PictureFrame,
	}
}

func (f *DecodedFrame) picturePlanesHigh() h264PicturePlanesHigh {
	if f == nil {
		return h264PicturePlanesHigh{}
	}
	return h264PicturePlanesHigh{
		Y:                f.Y16,
		Cb:               f.Cb16,
		Cr:               f.Cr16,
		LumaStride:       f.LumaStride,
		ChromaStride:     f.ChromaStride,
		MBWidth:          f.MBWidth,
		MBHeight:         f.MBHeight,
		ChromaFormatIDC:  f.ChromaFormatIDC,
		PictureStructure: PictureFrame,
	}
}

func (f *DecodedFrame) matchesSPS(sps *SPS) error {
	if f == nil || sps == nil {
		return ErrInvalidData
	}
	if f.MBWidth != int(sps.MBWidth) || f.MBHeight != int(sps.MBHeight) ||
		f.Width != int(sps.Width) || f.Height != int(sps.Height) ||
		f.CropLeft != int(sps.CropLeft) || f.CropTop != int(sps.CropTop) ||
		f.ChromaFormatIDC != int(sps.ChromaFormatIDC) ||
		f.frameMBSOnlyFlag != sps.FrameMBSOnlyFlag ||
		f.BitDepthLuma != int(sps.BitDepthLuma) || f.BitDepthChroma != int(sps.BitDepthChroma) {
		return ErrUnsupported
	}
	return nil
}

func newH264MotionCompScratchForFrame(f *DecodedFrame) *h264MotionCompScratch {
	if f == nil {
		return nil
	}
	lumaStride := f.LumaStride
	chromaStride := f.ChromaStride
	if f.frameMBSOnlyFlag == 0 {
		if lumaStride > maxInt/2 || chromaStride > maxInt/2 {
			return nil
		}
		lumaStride *= 2
		chromaStride *= 2
	}
	yLen, err := checkedMulInt(16, lumaStride)
	if err != nil || len(f.Cb) > maxInt/2 || len(f.Cr) > maxInt/2 {
		return nil
	}
	edge, ok := checkedH264EdgeScratchSize(lumaStride, 16+5, 16+5)
	if !ok {
		return nil
	}
	if f.ChromaFormatIDC != 0 {
		chromaBlockH := 8*f.ChromaFormatIDC + 1
		chromaEdge, ok := checkedH264EdgeScratchSize(chromaStride, 9, chromaBlockH)
		if !ok {
			return nil
		}
		if f.ChromaFormatIDC == 3 {
			chromaEdge, ok = checkedH264EdgeScratchSize(chromaStride, 16+5, 16+5)
			if !ok {
				return nil
			}
		}
		if chromaEdge > edge {
			edge = chromaEdge
		}
	}
	return &h264MotionCompScratch{
		Y:    make([]uint8, yLen),
		Cb:   make([]uint8, len(f.Cb)),
		Cr:   make([]uint8, len(f.Cr)),
		Edge: make([]uint8, edge),
	}
}

func newH264MotionCompScratchHighForFrame(f *DecodedFrame) *h264MotionCompScratchHigh {
	if f == nil {
		return nil
	}
	lumaStride := f.LumaStride
	chromaStride := f.ChromaStride
	if f.frameMBSOnlyFlag == 0 {
		if lumaStride > maxInt/2 || chromaStride > maxInt/2 {
			return nil
		}
		lumaStride *= 2
		chromaStride *= 2
	}
	yLen, err := checkedMulInt(16, lumaStride)
	if err != nil || len(f.Cb16) > maxInt/2 || len(f.Cr16) > maxInt/2 {
		return nil
	}
	edge, ok := checkedH264EdgeScratchSize(lumaStride, 16+5, 16+5)
	if !ok {
		return nil
	}
	if f.ChromaFormatIDC != 0 {
		chromaBlockH := 8*f.ChromaFormatIDC + 1
		chromaEdge, ok := checkedH264EdgeScratchSize(chromaStride, 9, chromaBlockH)
		if !ok {
			return nil
		}
		if f.ChromaFormatIDC == 3 {
			chromaEdge, ok = checkedH264EdgeScratchSize(chromaStride, 16+5, 16+5)
			if !ok {
				return nil
			}
		}
		if chromaEdge > edge {
			edge = chromaEdge
		}
	}
	return &h264MotionCompScratchHigh{
		Y:    make([]uint16, yLen),
		Cb:   make([]uint16, len(f.Cb16)),
		Cr:   make([]uint16, len(f.Cr16)),
		Edge: make([]uint16, edge),
	}
}

func checkedH264EdgeScratchSize(stride int, blockW int, blockH int) (int, bool) {
	if stride <= 0 || blockW <= 0 || blockH <= 0 {
		return 0, false
	}
	edgeStride := h264EdgeStride(stride, blockW)
	rows, err := checkedMulInt(blockH-1, edgeStride)
	if err != nil || rows > maxInt-blockW {
		return 0, false
	}
	return rows + blockW, true
}

func validateSimpleFrameReferenceSyntax(sh *SliceHeader) error {
	if sh == nil {
		return ErrInvalidData
	}
	for i := uint32(0); i < sh.NBMMCO; i++ {
		switch sh.MMCO[i].Opcode {
		case mmcoEnd, mmcoShort2Unused, mmcoLong2Unused, mmcoShort2Long, mmcoSetMaxLong, mmcoReset, mmcoLong:
		default:
			return ErrUnsupported
		}
	}
	if sh.SliceTypeNoS == PictureTypeP || sh.SliceTypeNoS == PictureTypeB {
		if sh.RefCount[0] == 0 {
			return ErrInvalidData
		}
		if sh.RefCount[0] > simpleMaxShortRefs {
			return ErrUnsupported
		}
		if sh.SliceTypeNoS == PictureTypeB {
			if sh.RefCount[1] == 0 {
				return ErrInvalidData
			}
			if sh.RefCount[1] > simpleMaxShortRefs {
				return ErrUnsupported
			}
		}
		listCount := 1
		if sh.SliceTypeNoS == PictureTypeB {
			listCount = 2
		}
		for list := 0; list < listCount; list++ {
			for i := uint32(0); i < sh.NBRefModifications[list]; i++ {
				if i >= maxRefMods {
					return ErrInvalidData
				}
				if sh.RefModifications[list][i].Op > 2 {
					return ErrUnsupported
				}
			}
		}
	}
	return nil
}
