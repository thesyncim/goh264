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

func TestSimpleFrameDPBAppendsLongRefsToDefaultPList(t *testing.T) {
	sps := simpleDPBTestSPS(4)
	short := simpleDPBTestFrame(sps, 2)
	long0 := simpleDPBTestFrame(sps, 10)
	long3 := simpleDPBTestFrame(sps, 11)
	dpb := simpleFrameDPB{short: []*DecodedFrame{short}}
	dpb.long[0] = long0
	dpb.long[3] = long3

	list, err := dpb.buildPRefList(simpleDPBTestPHeader(sps, 3, 3))
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 3 || list[0] != short || list[1] != long0 || list[2] != long3 {
		t.Fatalf("default list = %p/%p/%p, want short/long0/long3 %p/%p/%p", list[0], list[1], list[2], short, long0, long3)
	}
}

func TestSimpleFrameDPBReordersLongRefs(t *testing.T) {
	sps := simpleDPBTestSPS(3)
	short := simpleDPBTestFrame(sps, 2)
	long := simpleDPBTestFrame(sps, 10)
	dpb := simpleFrameDPB{short: []*DecodedFrame{short}}
	dpb.long[1] = long
	sh := simpleDPBTestPHeader(sps, 3, 2)
	sh.NBRefModifications[0] = 1
	sh.RefModifications[0][0] = RefModification{Op: 2, Val: 1}

	list, err := dpb.buildPRefList(sh)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 || list[0] != long || list[1] != short {
		t.Fatalf("reordered list = %p/%p, want long/short %p/%p", list[0], list[1], long, short)
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

func TestSimpleFrameDPBSlidingWindowCountsLongRefs(t *testing.T) {
	sps := simpleDPBTestSPS(2)
	long := simpleDPBTestFrame(sps, 10)
	older := simpleDPBTestFrame(sps, 1)
	current := simpleDPBTestFrame(sps, 2)
	dpb := simpleFrameDPB{short: []*DecodedFrame{older}}
	dpb.long[0] = long
	sh := &SliceHeader{
		NALType:          NALSlice,
		SPS:              sps,
		FrameNum:         2,
		PictureStructure: PictureFrame,
	}

	if err := dpb.markDecodedFrame(current, sh, 2); err != nil {
		t.Fatal(err)
	}
	if len(dpb.short) != 1 || dpb.short[0] != current || dpb.long[0] != long {
		t.Fatalf("refs after sliding window = short %v long %v", simpleDPBFrameNums(dpb.short), simpleDPBLongRefs(dpb))
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

func TestSimpleFrameDPBShortToLongMMCO(t *testing.T) {
	sps := simpleDPBTestSPS(3)
	newest := simpleDPBTestFrame(sps, 2)
	older := simpleDPBTestFrame(sps, 1)
	current := simpleDPBTestFrame(sps, 3)
	dpb := simpleFrameDPB{short: []*DecodedFrame{newest, older}}
	sh := &SliceHeader{
		NALType:            NALSlice,
		SPS:                sps,
		FrameNum:           3,
		PictureStructure:   PictureFrame,
		ExplicitRefMarking: 1,
		NBMMCO:             1,
		MMCO: [maxMMCOCount]MMCO{
			{Opcode: mmcoShort2Long, ShortPicNum: 1, LongArg: 2},
		},
	}

	if err := dpb.markDecodedFrame(current, sh, 2); err != nil {
		t.Fatal(err)
	}
	if len(dpb.short) != 2 || dpb.short[0] != current || dpb.short[1] != newest || dpb.long[2] != older {
		t.Fatalf("refs after short2long = short %v long %v", simpleDPBFrameNums(dpb.short), simpleDPBLongRefs(dpb))
	}
}

func TestSimpleFrameDPBLongMMCOAssignsCurrent(t *testing.T) {
	sps := simpleDPBTestSPS(2)
	current := simpleDPBTestFrame(sps, 0)
	var dpb simpleFrameDPB
	sh := &SliceHeader{
		NALType:            NALIDRSlice,
		SPS:                sps,
		FrameNum:           0,
		PictureStructure:   PictureFrame,
		ExplicitRefMarking: 1,
		NBMMCO:             1,
		MMCO: [maxMMCOCount]MMCO{
			{Opcode: mmcoLong, LongArg: 0},
		},
	}

	if err := dpb.markDecodedFrame(current, sh, 3); err != nil {
		t.Fatal(err)
	}
	if len(dpb.short) != 0 || dpb.long[0] != current {
		t.Fatalf("refs after long-current = short %v long %v", simpleDPBFrameNums(dpb.short), simpleDPBLongRefs(dpb))
	}
}

func TestSimpleFrameDPBLongRemovalMMCO(t *testing.T) {
	sps := simpleDPBTestSPS(4)
	long0 := simpleDPBTestFrame(sps, 10)
	long2 := simpleDPBTestFrame(sps, 12)
	long3 := simpleDPBTestFrame(sps, 13)
	current := simpleDPBTestFrame(sps, 4)
	dpb := simpleFrameDPB{}
	dpb.long[0] = long0
	dpb.long[2] = long2
	dpb.long[3] = long3
	sh := &SliceHeader{
		NALType:            NALSlice,
		SPS:                sps,
		FrameNum:           4,
		PictureStructure:   PictureFrame,
		ExplicitRefMarking: 1,
		NBMMCO:             2,
		MMCO: [maxMMCOCount]MMCO{
			{Opcode: mmcoLong2Unused, LongArg: 2},
			{Opcode: mmcoSetMaxLong, LongArg: 3},
		},
	}

	if err := dpb.markDecodedFrame(current, sh, 2); err != nil {
		t.Fatal(err)
	}
	if dpb.long[0] != long0 || dpb.long[2] != nil || dpb.long[3] != nil || len(dpb.short) != 1 || dpb.short[0] != current {
		t.Fatalf("refs after long removals = short %v long %v", simpleDPBFrameNums(dpb.short), simpleDPBLongRefs(dpb))
	}
}

func TestValidateSimpleFrameReferenceSyntaxRejectsUnsupportedMMCO(t *testing.T) {
	sh := simpleDPBTestPHeader(simpleDPBTestSPS(2), 3, 1)
	sh.NBMMCO = 1
	sh.MMCO[0] = MMCO{Opcode: 7}

	if err := validateSimpleFrameReferenceSyntax(sh); err != ErrUnsupported {
		t.Fatalf("err = %v, want ErrUnsupported", err)
	}
}

func TestValidateSimpleFrameReferenceSyntaxAllowsLongRefs(t *testing.T) {
	sh := simpleDPBTestPHeader(simpleDPBTestSPS(2), 3, 1)
	sh.NBRefModifications[0] = 1
	sh.RefModifications[0][0] = RefModification{Op: 2, Val: 0}
	sh.NBMMCO = 5
	sh.MMCO[0] = MMCO{Opcode: mmcoShort2Long, ShortPicNum: 1, LongArg: 0}
	sh.MMCO[1] = MMCO{Opcode: mmcoLong2Unused, LongArg: 0}
	sh.MMCO[2] = MMCO{Opcode: mmcoSetMaxLong, LongArg: 16}
	sh.MMCO[3] = MMCO{Opcode: mmcoLong, LongArg: 0}
	sh.MMCO[4] = MMCO{Opcode: mmcoEnd}

	if err := validateSimpleFrameReferenceSyntax(sh); err != nil {
		t.Fatalf("err = %v, want nil", err)
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

func simpleDPBLongRefs(dpb simpleFrameDPB) map[int]uint32 {
	out := make(map[int]uint32)
	for i, frame := range dpb.long {
		if frame != nil {
			out[i] = frame.frameNum
		}
	}
	return out
}
