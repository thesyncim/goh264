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

func TestSimplePOCType0FrameOrder(t *testing.T) {
	sps := simpleDPBTestSPS(2)
	sps.Log2MaxFrameNum = 4
	sps.PocType = 0
	sps.Log2MaxPocLSB = 4
	var dpb simpleFrameDPB
	dpb.reset()

	idr := simpleDPBTestFrame(sps, 0)
	idrHeader := simpleDPBTestPOCHeader(sps, NALIDRSlice, PictureTypeI, 0, 0)
	if err := dpb.initFramePOC(idr, idrHeader, 3); err != nil {
		t.Fatal(err)
	}
	if err := dpb.markDecodedFrame(idr, idrHeader, 3); err != nil {
		t.Fatal(err)
	}

	p := simpleDPBTestFrame(sps, 1)
	pHeader := simpleDPBTestPOCHeader(sps, NALSlice, PictureTypeP, 1, 4)
	if err := dpb.initFramePOC(p, pHeader, 2); err != nil {
		t.Fatal(err)
	}
	if err := dpb.markDecodedFrame(p, pHeader, 2); err != nil {
		t.Fatal(err)
	}

	b := simpleDPBTestFrame(sps, 2)
	bHeader := simpleDPBTestPOCHeader(sps, NALSlice, PictureTypeB, 2, 2)
	if err := dpb.initFramePOC(b, bHeader, 0); err != nil {
		t.Fatal(err)
	}
	if err := dpb.markDecodedFrame(b, bHeader, 0); err != nil {
		t.Fatal(err)
	}

	if !(idr.poc < b.poc && b.poc < p.poc) {
		t.Fatalf("poc order idr/b/p = %d/%d/%d, want display order", idr.poc, b.poc, p.poc)
	}
	if len(dpb.short) != 2 || dpb.short[0] != p || dpb.short[1] != idr {
		t.Fatalf("short refs after non-ref B = %v, want P/IDR only", simpleDPBFrameNums(dpb.short))
	}
}

func TestSimpleRecoveryPointMarksImmediateRecoveryKeyFrame(t *testing.T) {
	sps := simpleDPBTestSPS(1)
	sps.Log2MaxFrameNum = 4
	frame := simpleDPBTestFrame(sps, 3)
	sh := simpleDPBTestPHeader(sps, 3, 1)
	sh.NALType = NALSlice
	sei := &H264SEIContext{}
	sei.Reset()
	sei.RecoveryPoint.RecoveryFrameCount = 0

	var dpb simpleFrameDPB
	dpb.reset()
	dpb.applySimpleRecoveryPoint(frame, sh, 1, sei)
	if !frame.KeyFrame || frame.recovered&simpleFrameRecoveredSEI == 0 || dpb.recoveryFrame != -1 {
		t.Fatalf("recovery state = key %v recovered %#x recoveryFrame %d",
			frame.KeyFrame, frame.recovered, dpb.recoveryFrame)
	}
}

func TestSimpleRecoveryPointTracksModuloFrameNum(t *testing.T) {
	sps := simpleDPBTestSPS(1)
	sps.Log2MaxFrameNum = 4
	start := simpleDPBTestFrame(sps, 14)
	startHeader := simpleDPBTestPHeader(sps, 14, 1)
	startHeader.NALType = NALSlice
	sei := &H264SEIContext{}
	sei.Reset()
	sei.RecoveryPoint.RecoveryFrameCount = 3

	var dpb simpleFrameDPB
	dpb.reset()
	dpb.applySimpleRecoveryPoint(start, startHeader, 1, sei)
	if start.KeyFrame || start.recovered&simpleFrameRecoveredSEI != 0 || dpb.recoveryFrame != 1 {
		t.Fatalf("start recovery state = key %v recovered %#x recoveryFrame %d",
			start.KeyFrame, start.recovered, dpb.recoveryFrame)
	}

	next := simpleDPBTestFrame(sps, 1)
	nextHeader := simpleDPBTestPHeader(sps, 1, 1)
	nextHeader.NALType = NALSlice
	emptySEI := &H264SEIContext{}
	emptySEI.Reset()
	dpb.applySimpleRecoveryPoint(next, nextHeader, 1, emptySEI)
	if next.KeyFrame || next.recovered&simpleFrameRecoveredSEI == 0 || dpb.recoveryFrame != -1 {
		t.Fatalf("target recovery state = key %v recovered %#x recoveryFrame %d",
			next.KeyFrame, next.recovered, dpb.recoveryFrame)
	}
}

func TestSimpleFrameDPBBuildsDefaultBListsAroundCurrentPOC(t *testing.T) {
	sps := simpleDPBTestSPS(2)
	past := simpleDPBTestFrame(sps, 0)
	past.poc = 0
	future := simpleDPBTestFrame(sps, 1)
	future.poc = 4
	dpb := simpleFrameDPB{short: []*DecodedFrame{future, past}}

	lists, err := dpb.buildBRefLists(simpleDPBTestBHeader(sps, 2, 1, 1), 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(lists[0]) != 1 || len(lists[1]) != 1 {
		t.Fatalf("B list lengths = %d/%d, want 1/1", len(lists[0]), len(lists[1]))
	}
	if lists[0][0] != past || lists[1][0] != future {
		t.Fatalf("B lists = %p/%p, want list0 past %p list1 future %p", lists[0][0], lists[1][0], past, future)
	}
}

func TestSimpleFrameDPBSwapsIdenticalBLists(t *testing.T) {
	sps := simpleDPBTestSPS(2)
	newer := simpleDPBTestFrame(sps, 1)
	newer.poc = 4
	older := simpleDPBTestFrame(sps, 0)
	older.poc = 0
	dpb := simpleFrameDPB{short: []*DecodedFrame{newer, older}}

	lists, err := dpb.buildBRefLists(simpleDPBTestBHeader(sps, 2, 2, 2), 6)
	if err != nil {
		t.Fatal(err)
	}
	if len(lists[0]) != 2 || len(lists[1]) != 2 {
		t.Fatalf("B list lengths = %d/%d, want 2/2", len(lists[0]), len(lists[1]))
	}
	if lists[0][0] != newer || lists[0][1] != older ||
		lists[1][0] != older || lists[1][1] != newer {
		t.Fatalf("B lists = [%p %p] / [%p %p], want list1 swap", lists[0][0], lists[0][1], lists[1][0], lists[1][1])
	}
}

func TestSimpleFrameDPBInitializesImplicitBWeights(t *testing.T) {
	sps := simpleDPBTestSPS(2)
	pps := &PPS{SPS: sps, WeightedBipredIDC: 2}
	past := simpleDPBTestFrame(sps, 0)
	past.poc = 0
	future := simpleDPBTestFrame(sps, 1)
	future.poc = 6
	current := simpleDPBTestFrame(sps, 2)
	current.poc = 2
	dpb := simpleFrameDPB{short: []*DecodedFrame{future, past}}
	sh := simpleDPBTestBHeader(sps, 2, 1, 1)
	sh.PPS = pps

	refs, err := dpb.buildRefLists(sh, current)
	if err != nil {
		t.Fatal(err)
	}
	if len(refs[0]) != 1 || len(refs[1]) != 1 {
		t.Fatalf("ref lengths = %d/%d, want 1/1", len(refs[0]), len(refs[1]))
	}
	if sh.PredWeightTable.UseWeight != 2 || sh.PredWeightTable.UseWeightChroma != 2 {
		t.Fatalf("use_weight = %d/%d, want implicit", sh.PredWeightTable.UseWeight, sh.PredWeightTable.UseWeightChroma)
	}
	if sh.PredWeightTable.LumaLog2WeightDenom != 5 || sh.PredWeightTable.ChromaLog2WeightDenom != 5 {
		t.Fatalf("denom = %d/%d, want 5/5", sh.PredWeightTable.LumaLog2WeightDenom, sh.PredWeightTable.ChromaLog2WeightDenom)
	}
	if got := sh.PredWeightTable.ImplicitWeight[0][0]; got != [2]int32{43, 43} {
		t.Fatalf("implicit weight = %v, want 43/43", got)
	}
}

func TestSimpleFrameDPBDisablesSymmetricImplicitBWeights(t *testing.T) {
	sps := simpleDPBTestSPS(2)
	pps := &PPS{SPS: sps, WeightedBipredIDC: 2}
	past := simpleDPBTestFrame(sps, 0)
	past.poc = 0
	future := simpleDPBTestFrame(sps, 1)
	future.poc = 6
	current := simpleDPBTestFrame(sps, 2)
	current.poc = 3
	dpb := simpleFrameDPB{short: []*DecodedFrame{future, past}}
	sh := simpleDPBTestBHeader(sps, 2, 1, 1)
	sh.PPS = pps
	sh.PredWeightTable.UseWeight = 2
	sh.PredWeightTable.UseWeightChroma = 2

	if _, err := dpb.buildRefLists(sh, current); err != nil {
		t.Fatal(err)
	}
	if sh.PredWeightTable.UseWeight != 0 || sh.PredWeightTable.UseWeightChroma != 0 {
		t.Fatalf("use_weight = %d/%d, want disabled", sh.PredWeightTable.UseWeight, sh.PredWeightTable.UseWeightChroma)
	}
}

func TestSimpleFrameDPBDelaysBOutputUntilFlush(t *testing.T) {
	sps := simpleDPBTestSPS(2)
	sps.BitstreamRestrictionFlag = 1
	sps.NumReorderFrames = 1
	var dpb simpleFrameDPB
	dpb.reset()
	idr := simpleDPBTestFrame(sps, 0)
	idr.poc = 0
	idr.idrKeyFrame = true
	p := simpleDPBTestFrame(sps, 1)
	p.poc = 4
	b := simpleDPBTestFrame(sps, 2)
	b.poc = 2

	if err := dpb.holdOutputFrame(idr, simpleDPBTestPOCHeader(sps, NALIDRSlice, PictureTypeI, 0, 0)); err != nil {
		t.Fatal(err)
	}
	out, err := dpb.drainOutputFrames(false)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 0 {
		t.Fatalf("output after IDR = %d frames, want delayed", len(out))
	}

	if err := dpb.holdOutputFrame(p, simpleDPBTestPOCHeader(sps, NALSlice, PictureTypeP, 1, 4)); err != nil {
		t.Fatal(err)
	}
	out, err = dpb.drainOutputFrames(false)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0] != idr {
		t.Fatalf("output after P = %v, want IDR %p", out, idr)
	}

	if err := dpb.holdOutputFrame(b, simpleDPBTestPOCHeader(sps, NALSlice, PictureTypeB, 2, 2)); err != nil {
		t.Fatal(err)
	}
	out, err = dpb.drainOutputFrames(false)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0] != b {
		t.Fatalf("output after B = %v, want B %p", out, b)
	}

	out, err = dpb.drainOutputFrames(true)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0] != p {
		t.Fatalf("flush output = %v, want P %p", out, p)
	}
}

func TestSimpleFrameDPBInfersReorderDelayFromPOCGap(t *testing.T) {
	sps := simpleDPBTestSPS(2)
	var dpb simpleFrameDPB
	dpb.reset()
	idr := simpleDPBTestFrame(sps, 0)
	idr.poc = 0
	idr.idrKeyFrame = true
	p := simpleDPBTestFrame(sps, 1)
	p.poc = 4
	b := simpleDPBTestFrame(sps, 2)
	b.poc = 2

	if err := dpb.holdOutputFrame(idr, simpleDPBTestPOCHeader(sps, NALIDRSlice, PictureTypeI, 0, 0)); err != nil {
		t.Fatal(err)
	}
	out, err := dpb.drainOutputFrames(false)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0] != idr {
		t.Fatalf("output after IDR = %v, want IDR %p", out, idr)
	}

	if err := dpb.holdOutputFrame(p, simpleDPBTestPOCHeader(sps, NALSlice, PictureTypeP, 1, 4)); err != nil {
		t.Fatal(err)
	}
	out, err = dpb.drainOutputFrames(false)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 0 {
		t.Fatalf("output after P = %d frames, want inferred delay", len(out))
	}
	if dpb.hasBFrames != 1 {
		t.Fatalf("hasBFrames after POC gap = %d, want 1", dpb.hasBFrames)
	}

	if err := dpb.holdOutputFrame(b, simpleDPBTestPOCHeader(sps, NALSlice, PictureTypeB, 2, 2)); err != nil {
		t.Fatal(err)
	}
	out, err = dpb.drainOutputFrames(false)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0] != b {
		t.Fatalf("output after B = %v, want B %p", out, b)
	}

	out, err = dpb.drainOutputFrames(true)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0] != p {
		t.Fatalf("flush output = %v, want P %p", out, p)
	}
}

func TestSimpleFrameDPBPrimesReorderDelayFromLeadingLowerPOC(t *testing.T) {
	sps := simpleDPBTestSPS(2)
	sps.Log2MaxFrameNum = 4
	sps.PocType = 0
	sps.Log2MaxPocLSB = 4
	var probe simpleFrameDPB
	probe.reset()

	idrHeader := simpleDPBTestPOCHeader(sps, NALIDRSlice, PictureTypeI, 0, 8)
	if err := probe.primeOutputReorderDelayFromHeader(idrHeader, 3); err != nil {
		t.Fatal(err)
	}
	if probe.hasBFrames != 0 {
		t.Fatalf("primed delay after first IDR = %d, want 0", probe.hasBFrames)
	}

	lowerHeader := simpleDPBTestPOCHeader(sps, NALSlice, PictureTypeP, 1, 4)
	if err := probe.primeOutputReorderDelayFromHeader(lowerHeader, 2); err != nil {
		t.Fatal(err)
	}
	if probe.hasBFrames != 1 {
		t.Fatalf("primed delay after lower POC = %d, want 1", probe.hasBFrames)
	}

	var dpb simpleFrameDPB
	dpb.reset()
	dpb.hasBFrames = probe.hasBFrames
	idr := simpleDPBTestFrame(sps, 0)
	idr.poc = 65544
	idr.idrKeyFrame = true
	lower := simpleDPBTestFrame(sps, 1)
	lower.poc = 65540
	if err := dpb.holdOutputFrame(idr, idrHeader); err != nil {
		t.Fatal(err)
	}
	out, err := dpb.drainOutputFrames(false)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 0 {
		t.Fatalf("output after primed IDR = %v, want delayed", out)
	}
	if err := dpb.holdOutputFrame(lower, lowerHeader); err != nil {
		t.Fatal(err)
	}
	out, err = dpb.drainOutputFrames(false)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0] != lower {
		t.Fatalf("output after lower POC = %v, want lower %p", out, lower)
	}
}

func TestSimpleFrameDPBPrimeReorderDelayKeepsContiguousPOCImmediate(t *testing.T) {
	sps := simpleDPBTestSPS(2)
	sps.Log2MaxFrameNum = 4
	sps.PocType = 0
	sps.Log2MaxPocLSB = 4
	var probe simpleFrameDPB
	probe.reset()

	idrHeader := simpleDPBTestPOCHeader(sps, NALIDRSlice, PictureTypeI, 0, 0)
	if err := probe.primeOutputReorderDelayFromHeader(idrHeader, 3); err != nil {
		t.Fatal(err)
	}
	pHeader := simpleDPBTestPOCHeader(sps, NALSlice, PictureTypeP, 1, 2)
	if err := probe.primeOutputReorderDelayFromHeader(pHeader, 2); err != nil {
		t.Fatal(err)
	}
	if probe.hasBFrames != 0 {
		t.Fatalf("primed delay for contiguous POC = %d, want 0", probe.hasBFrames)
	}
}

func TestSimpleFrameDPBMMCOResetPreservesDelayedOutputState(t *testing.T) {
	sps := simpleDPBTestSPS(2)
	old := simpleDPBTestFrame(sps, 1)
	old.poc = 2
	current := simpleDPBTestFrame(sps, 3)
	dpb := simpleFrameDPB{
		short:             []*DecodedFrame{old},
		delayed:           []*DecodedFrame{old},
		hasBFrames:        1,
		nextOutputedPOC:   2,
		nextOutputedValid: true,
	}
	dpb.poc.frameNum = 3
	sh := &SliceHeader{
		NALType:            NALSlice,
		SPS:                sps,
		FrameNum:           3,
		PictureStructure:   PictureFrame,
		ExplicitRefMarking: 1,
		NBMMCO:             1,
		MMCO: [maxMMCOCount]MMCO{
			{Opcode: mmcoReset},
		},
	}

	if err := dpb.markDecodedFrame(current, sh, 2); err != nil {
		t.Fatal(err)
	}
	if len(dpb.delayed) != 1 || dpb.delayed[0] != old || dpb.hasBFrames != 1 || !dpb.nextOutputedValid {
		t.Fatalf("delayed output state changed: len=%d hasB=%d nextValid=%v", len(dpb.delayed), dpb.hasBFrames, dpb.nextOutputedValid)
	}
	if !current.mmcoReset || current.frameNum != 0 || dpb.poc.frameNum != 0 {
		t.Fatalf("current reset state = mmco %v frame %d poc frame %d, want reset to 0", current.mmcoReset, current.frameNum, dpb.poc.frameNum)
	}
	if len(dpb.short) != 1 || dpb.short[0] != current {
		t.Fatalf("short refs after reset = %v, want current only", simpleDPBFrameNums(dpb.short))
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

func TestValidateSimpleFrameReferenceSyntaxAllowsBList1Reorder(t *testing.T) {
	sh := simpleDPBTestBHeader(simpleDPBTestSPS(2), 3, 1, 1)
	sh.NBRefModifications[1] = 1
	sh.RefModifications[1][0] = RefModification{Op: 0, Val: 0}

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

func simpleDPBTestBHeader(sps *SPS, frameNum uint32, refCount0 uint32, refCount1 uint32) *SliceHeader {
	return &SliceHeader{
		SliceTypeNoS:     PictureTypeB,
		SPS:              sps,
		FrameNum:         frameNum,
		CurrPicNum:       frameNum,
		MaxPicNum:        16,
		PictureStructure: PictureFrame,
		RefCount:         [2]uint32{refCount0, refCount1},
		ListCount:        2,
	}
}

func simpleDPBTestPOCHeader(sps *SPS, nalType NALUnitType, sliceType int32, frameNum uint32, pocLSB uint32) *SliceHeader {
	return &SliceHeader{
		NALType:          nalType,
		SliceTypeNoS:     sliceType,
		SPS:              sps,
		FrameNum:         frameNum,
		CurrPicNum:       frameNum,
		MaxPicNum:        16,
		POCLSB:           pocLSB,
		PictureStructure: PictureFrame,
		RefCount:         [2]uint32{1, 1},
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
