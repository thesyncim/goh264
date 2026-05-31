// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

import "testing"

func TestSimpleFrameDPBBuildsDefaultPListNewestFirst(t *testing.T) {
	sps := simpleDPBTestSPS(2)
	newest := simpleDPBTestFrame(sps, 2)
	older := simpleDPBTestFrame(sps, 1)
	dpb := simpleFrameDPB{short: []*DecodedFrame{newest, older}}

	list, err := dpb.buildPRefList(simpleDPBTestPHeader(sps, 3, 2))
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 || list[0] != newest || list[1] != older {
		t.Fatalf("default list = %p/%p, want newest/older %p/%p", list[0], list[1], newest, older)
	}
}

func TestSimpleFrameDPBPadsMissingActiveRefsWithDefault(t *testing.T) {
	sps := simpleDPBTestSPS(2)
	ref := simpleDPBTestFrame(sps, 0)
	dpb := simpleFrameDPB{short: []*DecodedFrame{ref}}

	list, err := dpb.buildPRefList(simpleDPBTestPHeader(sps, 1, 2))
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 || list[0] != ref || list[1] != ref {
		t.Fatalf("padded list = %p/%p, want default/default %p", list[0], list[1], ref)
	}
}

func TestSimpleFrameDPBReordersShortRefs(t *testing.T) {
	sps := simpleDPBTestSPS(2)
	newest := simpleDPBTestFrame(sps, 2)
	older := simpleDPBTestFrame(sps, 1)
	dpb := simpleFrameDPB{short: []*DecodedFrame{newest, older}}
	sh := simpleDPBTestPHeader(sps, 3, 2)
	sh.NBRefModifications[0] = 1
	sh.RefModifications[0][0] = RefModification{Op: 0, Val: 1}

	list, err := dpb.buildPRefList(sh)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 || list[0] != older || list[1] != newest {
		t.Fatalf("reordered list = %p/%p, want older/newest %p/%p", list[0], list[1], older, newest)
	}
}

func TestSimpleFrameDPBSlidingWindowMarking(t *testing.T) {
	sps := simpleDPBTestSPS(2)
	var dpb simpleFrameDPB
	for i := uint32(0); i < 3; i++ {
		frame := simpleDPBTestFrame(sps, i)
		sh := &SliceHeader{
			NALType:          NALSlice,
			SPS:              sps,
			FrameNum:         i,
			PictureStructure: PictureFrame,
		}
		if i == 0 {
			sh.NALType = NALIDRSlice
		}
		if err := dpb.markDecodedFrame(frame, sh, 3); err != nil {
			t.Fatal(err)
		}
	}
	if len(dpb.short) != 2 || dpb.short[0].frameNum != 2 || dpb.short[1].frameNum != 1 {
		t.Fatalf("short refs = %v", simpleDPBFrameNums(dpb.short))
	}
}

func TestSimpleFrameDPBShortMMCO(t *testing.T) {
	sps := simpleDPBTestSPS(2)
	newest := simpleDPBTestFrame(sps, 2)
	older := simpleDPBTestFrame(sps, 1)
	dpb := simpleFrameDPB{short: []*DecodedFrame{newest, older}}
	frame := simpleDPBTestFrame(sps, 3)
	sh := &SliceHeader{
		NALType:            NALSlice,
		SPS:                sps,
		FrameNum:           3,
		PictureStructure:   PictureFrame,
		ExplicitRefMarking: 1,
		NBMMCO:             1,
		MMCO: [maxMMCOCount]MMCO{
			{Opcode: mmcoShort2Unused, ShortPicNum: 1},
		},
	}

	if err := dpb.markDecodedFrame(frame, sh, 2); err != nil {
		t.Fatal(err)
	}
	if len(dpb.short) != 2 || dpb.short[0] != frame || dpb.short[1] != newest {
		t.Fatalf("short refs after mmco = %v", simpleDPBFrameNums(dpb.short))
	}
}

func TestValidateSimpleFrameReferenceSyntaxRejectsUnsupportedMMCO(t *testing.T) {
	sh := simpleDPBTestPHeader(simpleDPBTestSPS(2), 3, 1)
	sh.NBMMCO = 1
	sh.MMCO[0] = MMCO{Opcode: mmcoLong, LongArg: 0}

	if err := validateSimpleFrameReferenceSyntax(sh); err != ErrUnsupported {
		t.Fatalf("err = %v, want ErrUnsupported", err)
	}
}

func simpleDPBTestSPS(refs uint32) *SPS {
	return &SPS{
		RefFrameCount:    refs,
		MBWidth:          1,
		MBHeight:         1,
		Width:            16,
		Height:           16,
		FrameMBSOnlyFlag: 1,
		ChromaFormatIDC:  1,
		BitDepthLuma:     8,
		BitDepthChroma:   8,
	}
}

func simpleDPBTestFrame(sps *SPS, frameNum uint32) *DecodedFrame {
	return &DecodedFrame{
		Y:               make([]uint8, 16*16),
		Cb:              make([]uint8, 8*8),
		Cr:              make([]uint8, 8*8),
		LumaStride:      16,
		ChromaStride:    8,
		Width:           int(sps.Width),
		Height:          int(sps.Height),
		MBWidth:         int(sps.MBWidth),
		MBHeight:        int(sps.MBHeight),
		ChromaFormatIDC: int(sps.ChromaFormatIDC),
		BitDepthLuma:    int(sps.BitDepthLuma),
		BitDepthChroma:  int(sps.BitDepthChroma),
		frameNum:        frameNum,
	}
}

func simpleDPBTestPHeader(sps *SPS, frameNum uint32, refCount uint32) *SliceHeader {
	return &SliceHeader{
		SliceTypeNoS:     PictureTypeP,
		SPS:              sps,
		FrameNum:         frameNum,
		CurrPicNum:       frameNum,
		MaxPicNum:        16,
		PictureStructure: PictureFrame,
		RefCount:         [2]uint32{refCount, 0},
	}
}

func simpleDPBFrameNums(frames []*DecodedFrame) []uint32 {
	out := make([]uint32, 0, len(frames))
	for _, frame := range frames {
		if frame != nil {
			out = append(out, frame.frameNum)
		}
	}
	return out
}
