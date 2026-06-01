// SPDX-License-Identifier: LGPL-2.1-or-later
//
// Source-shaped simple-frame integration around FFmpeg n8.0.1
// libavcodec/h264dec.c decode_nal_units/ff_h264_queue_decode_slice and
// libavcodec/h264_slice.c decode_slice. This is intentionally limited to the
// simple frame-picture subset whose macroblock decode/reconstruct path is
// already translated.

package h264

type DecodedFrame struct {
	Y, Cb, Cr                      []uint8
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
	tables                         *macroblockTables
	refEntries                     [2][]simpleRefEntry
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
	DisplayMatrix        AVDisplayMatrix
	DisplayOrientation   H2645SEIDisplayOrientation
	AlternativeTransfer  H2645SEIAlternativeTransfer
	AmbientViewing       H2645SEIAmbientViewingEnvironment
	FilmGrain            H2645SEIFilmGrainCharacteristics
	MasteringMetadata    AVMasteringDisplayMetadata
	MasteringDisplay     H2645SEIMasteringDisplay
	ContentLight         H2645SEIContentLight
}

type SimpleDecoder struct {
	sps [maxSPSCount]*SPS
	pps [maxPPSCount]*PPS
	dpb simpleFrameDPB
	sei H264SEIContext
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
	d.sei.Reset()
	return decodeSimpleNALUnitsWithState(nals, &d.sps, &d.pps, &d.dpb, &d.sei, packetSideData, false)
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
	return d.dpb.drainOutputFrames(true)
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
	return decodeSimpleNALUnitsWithState(nals, &d.sps, &d.pps, &d.dpb, &d.sei, DecodedFrameSideData{}, true)
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
	if spsList == nil || ppsList == nil || dpb == nil || sei == nil {
		return nil, ErrInvalidData
	}
	var frame *DecodedFrame
	var tables *macroblockTables
	var motionScratch *h264MotionCompScratch
	var frames []*DecodedFrame
	var loopFilterSlices []h264LoopFilterSliceParams
	var sliceNum uint16
	haveSlice := false
	frameComplete := false
	decodedFrames := 0

	for _, nal := range nals {
		switch nal.Type {
		case NALSPS:
			sps, err := DecodeSPS(nal.RBSP)
			if err != nil {
				return nil, err
			}
			spsList[sps.SPSID] = sps
		case NALPPS:
			pps, err := DecodePPS(nal.RBSP, spsList)
			if err != nil {
				return nil, err
			}
			ppsList[pps.PPSID] = pps
		case NALSEI:
			if haveSlice {
				continue
			}
			// FFmpeg keeps SEI parse failures non-fatal unless AV_EF_EXPLODE is set.
			_ = sei.Decode(nal.RBSP, spsList)
		case NALSlice, NALIDRSlice:
			sh, payload, err := parseSliceHeaderWithPayload(nal, ppsList)
			if err != nil {
				return nil, err
			}
			if sh.RedundantPicCount != 0 {
				continue
			}
			if sh.SliceTypeNoS != PictureTypeI && sh.SliceTypeNoS != PictureTypeP && sh.SliceTypeNoS != PictureTypeB {
				return nil, ErrUnsupported
			}
			if err := validateSimpleFrameReferenceSyntax(sh); err != nil {
				return nil, err
			}
			if frameComplete || haveSlice && sh.FirstMBAddr == 0 {
				if sh.FirstMBAddr != 0 {
					return nil, ErrInvalidData
				}
				frame = nil
				tables = nil
				motionScratch = nil
				loopFilterSlices = nil
				sliceNum = 0
				haveSlice = false
				frameComplete = false
			}
			if !haveSlice && sh.FirstMBAddr != 0 {
				return nil, ErrInvalidData
			}
			if frame == nil {
				if sei.PictureTiming.Present != 0 {
					if err := sei.PictureTiming.Process(sh.SPS); err != nil {
						// FFmpeg drops malformed picture-timing SEI without AV_EF_EXPLODE.
						sei.PictureTiming.Present = 0
						sei.PictureTiming.TimecodeCount = 0
					}
				}
				frame, tables, err = newSimpleDecodedFrame(sh.SPS)
				if err != nil {
					return nil, err
				}
				frame.SideData = decodedFrameSideDataFromSEI(sei)
				mergePacketSideDataIntoDecodedFrame(&frame.SideData, packetSideData)
				if err := dpb.initFramePOC(frame, sh, nal.RefIDC); err != nil {
					return nil, err
				}
				applySimpleFrameTimingProps(frame, sh.SPS, sei, dpb)
				dpb.applySimpleRecoveryPoint(frame, sh, nal.RefIDC, sei)
				consumeFrameSideDataFromSEI(sei)
				motionScratch = newH264MotionCompScratchForFrame(frame)
			} else if err := frame.matchesSPS(sh.SPS); err != nil {
				return nil, err
			}

			sliceNum++
			if sliceNum == ^uint16(0) {
				return nil, ErrInvalidData
			}
			for len(loopFilterSlices) <= int(sliceNum) {
				loopFilterSlices = append(loopFilterSlices, h264LoopFilterSliceParams{})
			}
			loopFilterSlices[sliceNum] = h264LoopFilterSliceParamsFromHeader(sh)
			pic := frame.picturePlanes()
			refctx, err := dpb.buildRefContext(sh, frame)
			if err != nil {
				return nil, err
			}
			frame.refEntries = cloneSimpleRefEntries2(refctx.Entries)
			result, err := tables.decodeFrameSliceData(&payload, &pic, sh, h264FrameSliceDecodeInput{
				SliceNum:      sliceNum,
				Refs:          refctx.Refs,
				Direct:        refctx.directMotionContext(frame, sh, sei),
				PredWeight:    &sh.PredWeightTable,
				MotionScratch: motionScratch,
			})
			if err != nil {
				return nil, err
			}
			if result.EndOfFrame {
				if err := tables.filterFrame(&pic, loopFilterSlices); err != nil {
					return nil, err
				}
				frameComplete = true
				decodedFrames++
				if err := dpb.markDecodedFrame(frame, sh, nal.RefIDC); err != nil {
					return nil, err
				}
				if err := dpb.holdOutputFrame(frame, sh); err != nil {
					return nil, err
				}
				if !flushOutput {
					out, err := dpb.drainOutputFrames(false)
					if err != nil {
						return nil, err
					}
					frames = append(frames, out...)
				}
			}
			haveSlice = true
		default:
			continue
		}
	}

	if flushOutput {
		out, err := dpb.drainOutputFrames(true)
		if err != nil {
			return nil, err
		}
		frames = append(frames, out...)
	}
	if decodedFrames == 0 || haveSlice && !frameComplete {
		return nil, ErrInvalidData
	}
	return frames, nil
}

func decodedFrameSideDataFromSEI(sei *H264SEIContext) DecodedFrameSideData {
	if sei == nil {
		return DecodedFrameSideData{}
	}
	return DecodedFrameSideData{
		UserDataUnregistered: cloneByteSlices(sei.Common.Unregistered.Data),
		A53ClosedCaptions:    append([]uint8(nil), sei.Common.A53Caption.Data...),
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
	}
}

func mergePacketSideDataIntoDecodedFrame(dst *DecodedFrameSideData, src DecodedFrameSideData) {
	if dst == nil {
		return
	}
	if len(src.A53ClosedCaptions) != 0 {
		dst.A53ClosedCaptions = append([]uint8(nil), src.A53ClosedCaptions...)
	}
	if src.AFD.Present != 0 {
		dst.AFD = src.AFD
	}
	if len(src.S12MTimecodes) != 0 && dst.PictureTiming.TimecodeCount == 0 {
		dst.S12MTimecodes = append([]uint32(nil), src.S12MTimecodes...)
	}
	if src.Stereo3D.Present != 0 {
		dst.Stereo3D = src.Stereo3D
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
}

func consumeFrameSideDataFromSEI(sei *H264SEIContext) {
	if sei == nil {
		return
	}
	sei.Common.Unregistered.Data = nil
	sei.Common.A53Caption.Data = nil
	sei.Common.AFD.Present = 0
	if sei.Common.FilmGrain.Present != 0 && sei.Common.FilmGrain.RepetitionPeriod == 0 {
		sei.Common.FilmGrain.Present = 0
	}
	sei.PictureTiming.Present = 0
	sei.PictureTiming.TimecodeCount = 0
	sei.RecoveryPoint.RecoveryFrameCount = -1
}

func cloneByteSlices(src [][]uint8) [][]uint8 {
	if len(src) == 0 {
		return nil
	}
	out := make([][]uint8, len(src))
	for i := range src {
		out[i] = append([]uint8(nil), src[i]...)
	}
	return out
}

// applySimpleFrameTimingProps mirrors FFmpeg n8.0.1 h264_export_frame_props
// for the simple frame-picture path. Field and MBAFF decoding remain
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
	if sps.BitDepthLuma != 8 || sps.BitDepthChroma != 8 || sps.FrameMBSOnlyFlag == 0 || sps.MBAFF != 0 {
		return nil, nil, ErrUnsupported
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

	frame := &DecodedFrame{
		LumaStride:                     mbWidth * 16,
		Width:                          int(sps.Width),
		Height:                         int(sps.Height),
		CropLeft:                       int(sps.CropLeft),
		CropTop:                        int(sps.CropTop),
		MBWidth:                        mbWidth,
		MBHeight:                       mbHeight,
		ChromaFormatIDC:                chromaFormatIDC,
		BitDepthLuma:                   int(sps.BitDepthLuma),
		BitDepthChroma:                 int(sps.BitDepthChroma),
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
	frame.Y = make([]uint8, frame.LumaStride*mbHeight*16)
	if chromaFormatIDC != 0 {
		chromaWidth, chromaHeight := h264ChromaFrameSize(mbWidth, mbHeight, chromaFormatIDC)
		frame.ChromaStride = chromaWidth
		frame.Cb = make([]uint8, frame.ChromaStride*chromaHeight)
		frame.Cr = make([]uint8, frame.ChromaStride*chromaHeight)
	}

	tables, err := newMacroblockTables(mbWidth, mbHeight, chromaFormatIDC)
	if err != nil {
		return nil, nil, err
	}
	frame.tables = tables
	pic := frame.picturePlanes()
	if err := pic.validate(); err != nil {
		return nil, nil, err
	}
	return frame, tables, nil
}

func (f *DecodedFrame) picturePlanes() h264PicturePlanes {
	if f == nil {
		return h264PicturePlanes{}
	}
	return h264PicturePlanes{
		Y:               f.Y,
		Cb:              f.Cb,
		Cr:              f.Cr,
		LumaStride:      f.LumaStride,
		ChromaStride:    f.ChromaStride,
		MBWidth:         f.MBWidth,
		MBHeight:        f.MBHeight,
		ChromaFormatIDC: f.ChromaFormatIDC,
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
		f.BitDepthLuma != int(sps.BitDepthLuma) || f.BitDepthChroma != int(sps.BitDepthChroma) {
		return ErrUnsupported
	}
	return nil
}

func newH264MotionCompScratchForFrame(f *DecodedFrame) *h264MotionCompScratch {
	if f == nil {
		return nil
	}
	edge := h264EdgeScratchSize(f.LumaStride, 16+5, 16+5)
	if f.ChromaFormatIDC != 0 {
		chromaBlockH := 8*f.ChromaFormatIDC + 1
		chromaEdge := h264EdgeScratchSize(f.ChromaStride, 9, chromaBlockH)
		if f.ChromaFormatIDC == 3 {
			chromaEdge = h264EdgeScratchSize(f.ChromaStride, 16+5, 16+5)
		}
		if chromaEdge > edge {
			edge = chromaEdge
		}
	}
	return &h264MotionCompScratch{
		Y:    make([]uint8, 16*f.LumaStride),
		Cb:   make([]uint8, len(f.Cb)),
		Cr:   make([]uint8, len(f.Cr)),
		Edge: make([]uint8, edge),
	}
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
