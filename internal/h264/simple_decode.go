// SPDX-License-Identifier: LGPL-2.1-or-later
//
// Source-shaped simple-frame integration around FFmpeg n8.0.1
// libavcodec/h264dec.c decode_nal_units/ff_h264_queue_decode_slice and
// libavcodec/h264_slice.c decode_slice. This is intentionally limited to the
// simple frame-picture subset whose macroblock decode/reconstruct path is
// already translated.

package h264

type DecodedFrame struct {
	Y, Cb, Cr       []uint8
	LumaStride      int
	ChromaStride    int
	Width           int
	Height          int
	CropLeft        int
	CropTop         int
	MBWidth         int
	MBHeight        int
	ChromaFormatIDC int
	BitDepthLuma    int
	BitDepthChroma  int
	frameNum        uint32
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

func DecodeSimpleNALUnits(nals []NALUnit) ([]*DecodedFrame, error) {
	var spsList [maxSPSCount]*SPS
	var ppsList [maxPPSCount]*PPS
	var frame *DecodedFrame
	var tables *macroblockTables
	var motionScratch *h264MotionCompScratch
	var dpb simpleFrameDPB
	var frames []*DecodedFrame
	var loopFilterSlices []h264LoopFilterSliceParams
	var sliceNum uint16
	haveSlice := false
	frameComplete := false

	for _, nal := range nals {
		switch nal.Type {
		case NALSPS:
			sps, err := DecodeSPS(nal.RBSP)
			if err != nil {
				return nil, err
			}
			spsList[sps.SPSID] = sps
		case NALPPS:
			pps, err := DecodePPS(nal.RBSP, &spsList)
			if err != nil {
				return nil, err
			}
			ppsList[pps.PPSID] = pps
		case NALSlice, NALIDRSlice:
			sh, payload, err := parseSliceHeaderWithPayload(nal, &ppsList)
			if err != nil {
				return nil, err
			}
			if sh.RedundantPicCount != 0 {
				continue
			}
			if sh.SliceTypeNoS != PictureTypeI && sh.SliceTypeNoS != PictureTypeP {
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
				frame, tables, err = newSimpleDecodedFrame(sh.SPS)
				if err != nil {
					return nil, err
				}
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
			refs, err := dpb.buildRefLists(sh)
			if err != nil {
				return nil, err
			}
			result, err := tables.decodeFrameSliceData(&payload, &pic, sh, h264FrameSliceDecodeInput{
				SliceNum:      sliceNum,
				Refs:          refs,
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
				frames = append(frames, frame)
				if err := dpb.markDecodedFrame(frame, sh, nal.RefIDC); err != nil {
					return nil, err
				}
			}
			haveSlice = true
		default:
			continue
		}
	}

	if len(frames) == 0 || haveSlice && !frameComplete {
		return nil, ErrInvalidData
	}
	return frames, nil
}

func newSimpleDecodedFrame(sps *SPS) (*DecodedFrame, *macroblockTables, error) {
	if sps == nil {
		return nil, nil, ErrInvalidData
	}
	if sps.BitDepthLuma != 8 || sps.BitDepthChroma != 8 || sps.FrameMBSOnlyFlag == 0 || sps.MBAFF != 0 || sps.TransformBypass != 0 {
		return nil, nil, ErrUnsupported
	}
	if sps.ChromaFormatIDC > 2 {
		return nil, nil, ErrUnsupported
	}
	mbWidth := int(sps.MBWidth)
	mbHeight := int(sps.MBHeight)
	chromaFormatIDC := int(sps.ChromaFormatIDC)
	if mbWidth <= 0 || mbHeight <= 0 || sps.Width <= 0 || sps.Height <= 0 {
		return nil, nil, ErrInvalidData
	}

	frame := &DecodedFrame{
		LumaStride:      mbWidth * 16,
		Width:           int(sps.Width),
		Height:          int(sps.Height),
		CropLeft:        int(sps.CropLeft),
		CropTop:         int(sps.CropTop),
		MBWidth:         mbWidth,
		MBHeight:        mbHeight,
		ChromaFormatIDC: chromaFormatIDC,
		BitDepthLuma:    int(sps.BitDepthLuma),
		BitDepthChroma:  int(sps.BitDepthChroma),
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
	edge := 20*f.LumaStride + 21
	if f.ChromaFormatIDC != 0 {
		chromaBlockH := 8*f.ChromaFormatIDC + 1
		chromaEdge := (chromaBlockH-1)*f.ChromaStride + 9
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
	if sh.NBRefModifications[1] != 0 {
		return ErrUnsupported
	}
	for i := uint32(0); i < sh.NBMMCO; i++ {
		switch sh.MMCO[i].Opcode {
		case mmcoEnd, mmcoShort2Unused, mmcoLong2Unused, mmcoShort2Long, mmcoSetMaxLong, mmcoReset, mmcoLong:
		default:
			return ErrUnsupported
		}
	}
	if sh.SliceTypeNoS == PictureTypeP {
		if sh.RefCount[0] == 0 {
			return ErrInvalidData
		}
		if sh.RefCount[0] > simpleMaxShortRefs {
			return ErrUnsupported
		}
		for i := uint32(0); i < sh.NBRefModifications[0]; i++ {
			if i >= maxRefMods {
				return ErrInvalidData
			}
			if sh.RefModifications[0][i].Op > 2 {
				return ErrUnsupported
			}
		}
	}
	return nil
}
