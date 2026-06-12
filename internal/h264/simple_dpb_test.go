// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

import (
	"math"
	"reflect"
	"testing"
	"unsafe"
)

func TestApplySimpleFieldRefPlaneBuildsValidHalfHeightViews(t *testing.T) {
	for _, tt := range []struct {
		name    string
		picture int32
	}{
		{name: "top", picture: PictureTopField},
		{name: "bottom", picture: PictureBottomField},
	} {
		t.Run(tt.name, func(t *testing.T) {
			frame := makeH264SliceDecodePicture(45, 30, 1)
			view := *frame

			applySimpleFieldRefPlane(&view, tt.picture)

			if view.MBWidth != 45 || view.MBHeight != 15 {
				t.Fatalf("field view mb geometry = %dx%d, want 45x15", view.MBWidth, view.MBHeight)
			}
			if view.LumaStride != frame.LumaStride*2 || view.ChromaStride != frame.ChromaStride*2 {
				t.Fatalf("field view strides = %d/%d, want %d/%d", view.LumaStride, view.ChromaStride, frame.LumaStride*2, frame.ChromaStride*2)
			}
			if tt.picture == PictureBottomField {
				if len(view.Y) != len(frame.Y)-frame.LumaStride || len(view.Cb) != len(frame.Cb)-frame.ChromaStride || len(view.Cr) != len(frame.Cr)-frame.ChromaStride {
					t.Fatalf("bottom field view lengths = %d/%d/%d, want one source line offset", len(view.Y), len(view.Cb), len(view.Cr))
				}
			}
			if err := view.validate(); err != nil {
				t.Fatalf("field view validate failed: %v", err)
			}
		})
	}
}

func TestApplySimpleFieldRefPlaneHighBuildsValidHalfHeightViews(t *testing.T) {
	for _, tt := range []struct {
		name    string
		picture int32
	}{
		{name: "top", picture: PictureTopField},
		{name: "bottom", picture: PictureBottomField},
	} {
		t.Run(tt.name, func(t *testing.T) {
			frame := makeH264SliceDecodePictureHigh(45, 30, 1)
			view := *frame

			applySimpleFieldRefPlaneHigh(&view, tt.picture)

			if view.MBWidth != 45 || view.MBHeight != 15 {
				t.Fatalf("high field view mb geometry = %dx%d, want 45x15", view.MBWidth, view.MBHeight)
			}
			if view.LumaStride != frame.LumaStride*2 || view.ChromaStride != frame.ChromaStride*2 {
				t.Fatalf("high field view strides = %d/%d, want %d/%d", view.LumaStride, view.ChromaStride, frame.LumaStride*2, frame.ChromaStride*2)
			}
			if tt.picture == PictureBottomField {
				if len(view.Y) != len(frame.Y)-frame.LumaStride || len(view.Cb) != len(frame.Cb)-frame.ChromaStride || len(view.Cr) != len(frame.Cr)-frame.ChromaStride {
					t.Fatalf("high bottom field view lengths = %d/%d/%d, want one source line offset", len(view.Y), len(view.Cb), len(view.Cr))
				}
			}
			if err := view.validate(); err != nil {
				t.Fatalf("high field view validate failed: %v", err)
			}
		})
	}
}

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

func TestSimpleFrameDPBRejectsOverflowedFieldRefEntryCapacity(t *testing.T) {
	sps := simpleDPBTestSPS(2)
	sh := simpleDPBTestPHeader(sps, 1, 1)
	sh.PictureStructure = PictureTopField
	var dpb simpleFrameDPB

	if _, err := dpb.buildDefaultEntriesFromFrames(fakeDecodedFrameSliceLen(maxInt/2+1), sh, false); err != ErrInvalidData {
		t.Fatalf("field ref entries overflow error = %v, want ErrInvalidData", err)
	}
}

func TestSimpleFrameRefEntryAdaptersRejectOverflowedLists(t *testing.T) {
	entries := fakeSimpleRefEntrySliceLen(maxInt/32 + 1)
	if got := cloneSimpleRefEntries(entries); got != nil {
		t.Fatalf("overflowed cloned ref entries = len %d, want nil", len(got))
	}
	if got := simpleFrameEntryPlanesRefs(entries); got != nil {
		t.Fatalf("overflowed low ref planes = len %d, want nil", len(got))
	}
	if got := simpleFrameEntryPlanesRefsHigh(entries); got != nil {
		t.Fatalf("overflowed high ref planes = len %d, want nil", len(got))
	}
	if got := simpleFrameEntryFrames(entries); got != nil {
		t.Fatalf("overflowed ref frames = len %d, want nil", len(got))
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

func TestSimplePOCType0FieldsPreserveComplementaryPOC(t *testing.T) {
	sps := simpleDPBTestSPS(1)
	sps.FrameMBSOnlyFlag = 0
	sps.Log2MaxFrameNum = 4
	sps.PocType = 0
	sps.Log2MaxPocLSB = 4
	var dpb simpleFrameDPB
	dpb.reset()

	frame := simpleDPBTestFrame(sps, 0)
	topHeader := simpleDPBTestPOCHeader(sps, NALIDRSlice, PictureTypeI, 0, 0)
	topHeader.PictureStructure = PictureTopField
	topHeader.CurrPicNum = 1
	topHeader.MaxPicNum = 32
	if err := dpb.initFramePOC(frame, topHeader, 3); err != nil {
		t.Fatal(err)
	}
	if frame.fieldPOC != [2]int32{65536, int32(2147483647)} || frame.poc != 65536 {
		t.Fatalf("top field poc = %v/%d, want top set and bottom left inactive", frame.fieldPOC, frame.poc)
	}
	dpb.finishFramePOC(3)

	bottomHeader := simpleDPBTestPOCHeader(sps, NALSlice, PictureTypeP, 0, 2)
	bottomHeader.PictureStructure = PictureBottomField
	bottomHeader.CurrPicNum = 1
	bottomHeader.MaxPicNum = 32
	if err := dpb.initFramePOC(frame, bottomHeader, 3); err != nil {
		t.Fatal(err)
	}
	if frame.fieldPOC != [2]int32{65536, 65538} || frame.poc != 65536 {
		t.Fatalf("paired field poc = %v/%d, want preserved top and bottom min frame poc", frame.fieldPOC, frame.poc)
	}
}

func TestSimpleFrameDPBBuildsDefaultFieldPListCurrentThenOpposite(t *testing.T) {
	sps := simpleDPBTestSPS(2)
	sps.FrameMBSOnlyFlag = 0
	ref := simpleDPBTestFrame(sps, 2)
	ref.fieldPOC = [2]int32{10, 12}
	ref.poc = 10
	dpb := simpleFrameDPB{short: []*DecodedFrame{ref}}
	dpb.setFrameRefMask(ref, PictureFrame)
	sh := simpleDPBTestPHeader(sps, 3, 2)
	sh.PictureStructure = PictureBottomField
	sh.CurrPicNum = 7
	sh.MaxPicNum = 32

	entries, err := dpb.buildPRefEntries(sh)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("entry count = %d, want 2", len(entries))
	}
	if entries[0].frame != ref || entries[0].pictureStructure != PictureBottomField || entries[0].picID != 5 || entries[0].poc != 12 {
		t.Fatalf("entry0 = %+v, want bottom field pic_id 5 poc 12", entries[0])
	}
	if entries[1].frame != ref || entries[1].pictureStructure != PictureTopField || entries[1].picID != 4 || entries[1].poc != 10 {
		t.Fatalf("entry1 = %+v, want top field pic_id 4 poc 10", entries[1])
	}
}

func TestSimpleFrameDPBReordersFieldShortRefsWithPicNumExtract(t *testing.T) {
	sps := simpleDPBTestSPS(2)
	sps.FrameMBSOnlyFlag = 0
	ref := simpleDPBTestFrame(sps, 2)
	ref.fieldPOC = [2]int32{10, 12}
	ref.poc = 10
	dpb := simpleFrameDPB{short: []*DecodedFrame{ref}}
	dpb.setFrameRefMask(ref, PictureFrame)
	sh := simpleDPBTestPHeader(sps, 3, 2)
	sh.PictureStructure = PictureBottomField
	sh.CurrPicNum = 7
	sh.MaxPicNum = 32
	sh.NBRefModifications[0] = 1
	sh.RefModifications[0][0] = RefModification{Op: 0, Val: 2}

	entries, err := dpb.buildPRefEntries(sh)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 || entries[0].pictureStructure != PictureTopField || entries[0].picID != 4 ||
		entries[1].pictureStructure != PictureBottomField || entries[1].picID != 5 {
		t.Fatalf("reordered field entries = %+v, want top then bottom", entries)
	}
}

func TestSimpleFrameDPBMMCOShortFieldRemovalKeepsOpposite(t *testing.T) {
	sps := simpleDPBTestSPS(2)
	sps.FrameMBSOnlyFlag = 0
	ref := simpleDPBTestFrame(sps, 2)
	ref.fieldPOC = [2]int32{10, 12}
	ref.poc = 10
	current := simpleDPBTestFrame(sps, 3)
	dpb := simpleFrameDPB{short: []*DecodedFrame{ref}}
	dpb.setFrameRefMask(ref, PictureFrame)
	sh := &SliceHeader{
		NALType:          NALSlice,
		SPS:              sps,
		FrameNum:         3,
		PictureStructure: PictureTopField,
		NBMMCO:           1,
		MMCO: [maxMMCOCount]MMCO{
			{Opcode: mmcoShort2Unused, ShortPicNum: 5},
		},
	}

	if _, _, err := dpb.applyMMCO(current, sh); err != nil {
		t.Fatal(err)
	}
	if len(dpb.short) != 1 || dpb.short[0] != ref || dpb.frameRefMask(ref) != PictureBottomField {
		t.Fatalf("refs after top removal = short %v mask %d, want bottom-only ref", simpleDPBFrameNums(dpb.short), dpb.frameRefMask(ref))
	}

	sh.MMCO[0].ShortPicNum = 4
	if _, _, err := dpb.applyMMCO(current, sh); err != nil {
		t.Fatal(err)
	}
	if len(dpb.short) != 0 {
		t.Fatalf("refs after bottom removal = short %v, want no DPB ref", simpleDPBFrameNums(dpb.short))
	}
}

func TestSimpleFrameDPBMarksSecondFieldWithoutDuplicateShortRef(t *testing.T) {
	sps := simpleDPBTestSPS(1)
	sps.FrameMBSOnlyFlag = 0
	frame := simpleDPBTestFrame(sps, 2)
	frame.fieldPOC = [2]int32{10, 12}
	frame.poc = 10
	dpb := simpleFrameDPB{short: []*DecodedFrame{frame}}
	dpb.setFrameRefMask(frame, PictureTopField)
	sh := &SliceHeader{
		NALType:          NALSlice,
		SPS:              sps,
		FrameNum:         2,
		PictureStructure: PictureBottomField,
	}

	if err := dpb.markDecodedFrame(frame, sh, 2); err != nil {
		t.Fatal(err)
	}
	if len(dpb.short) != 1 || dpb.short[0] != frame || dpb.frameRefMask(frame) != PictureFrame {
		t.Fatalf("second-field refs = short %v mask %d, want one complementary ref", simpleDPBFrameNums(dpb.short), dpb.frameRefMask(frame))
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

func TestSimpleRecoveryPointSeedsDerivedReorderDelay(t *testing.T) {
	sps := simpleDPBTestSPS(1)
	sps.Log2MaxFrameNum = 4
	sps.BitstreamRestrictionFlag = 0
	sps.NumReorderFrames = 5
	frame := simpleDPBTestFrame(sps, 3)
	sh := simpleDPBTestPHeader(sps, 3, 1)
	sh.NALType = NALSlice
	sei := &H264SEIContext{}
	sei.Reset()
	sei.RecoveryPoint.RecoveryFrameCount = 0

	var dpb simpleFrameDPB
	dpb.reset()
	dpb.applySimpleRecoveryPoint(frame, sh, 1, sei)
	if dpb.hasBFrames != 5 {
		t.Fatalf("hasBFrames after recovery = %d, want 5", dpb.hasBFrames)
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

func TestSimpleFrameDPBFrameMBAFFKeepsSymmetricImplicitBWeights(t *testing.T) {
	sps := simpleDPBTestSPS(2)
	sps.FrameMBSOnlyFlag = 0
	sps.MBAFF = 1
	pps := &PPS{SPS: sps, WeightedBipredIDC: 2}
	past := simpleDPBTestFrame(sps, 0)
	past.poc = 0
	past.fieldPOC = [2]int32{0, 2}
	future := simpleDPBTestFrame(sps, 1)
	future.poc = 6
	future.fieldPOC = [2]int32{6, 8}
	current := simpleDPBTestFrame(sps, 2)
	current.poc = 3
	current.fieldPOC = [2]int32{2, 4}
	dpb := simpleFrameDPB{short: []*DecodedFrame{future, past}}
	sh := simpleDPBTestBHeader(sps, 2, 1, 1)
	sh.PPS = pps

	if _, err := dpb.buildRefLists(sh, current); err != nil {
		t.Fatal(err)
	}
	if sh.PredWeightTable.UseWeight != 2 || sh.PredWeightTable.UseWeightChroma != 2 {
		t.Fatalf("frame-MBAFF use_weight = %d/%d, want implicit", sh.PredWeightTable.UseWeight, sh.PredWeightTable.UseWeightChroma)
	}
	if sh.PredWeightTable.LumaLog2WeightDenom != 5 || sh.PredWeightTable.ChromaLog2WeightDenom != 5 {
		t.Fatalf("frame-MBAFF denom = %d/%d, want 5/5", sh.PredWeightTable.LumaLog2WeightDenom, sh.PredWeightTable.ChromaLog2WeightDenom)
	}
}

func TestInitImplicitBWeightTableFrameMBAFFFillsExpandedFieldRefs(t *testing.T) {
	sps := simpleDPBTestSPS(2)
	frame0 := simpleDPBTestFrame(sps, 0)
	frame0.fieldPOC = [2]int32{0, 10}
	frame1 := simpleDPBTestFrame(sps, 1)
	frame1.fieldPOC = [2]int32{8, 20}
	current := simpleDPBTestFrame(sps, 2)
	current.fieldPOC = [2]int32{2, 14}
	lists := [2][]simpleRefEntry{
		{{frame: frame0, pictureStructure: PictureFrame, poc: frame0.poc}},
		{{frame: frame1, pictureStructure: PictureFrame, poc: frame1.poc}},
	}
	var pwt PredWeightTable
	pwt.ImplicitWeight[0][0] = [2]int32{7, 9}

	if err := initImplicitBWeightTableFrameMBAFF(&pwt, lists, [2]uint32{1, 1}, current); err != nil {
		t.Fatal(err)
	}
	if got := pwt.ImplicitWeight[0][0]; got != [2]int32{7, 9} {
		t.Fatalf("frame ref 0/0 weights = %v, want preserved", got)
	}
	if got := pwt.ImplicitWeight[16][16]; got != [2]int32{48, -48} {
		t.Fatalf("MBAFF top-field ref weights = %v, want top=48 bottom=-48", got)
	}
	if got := pwt.ImplicitWeight[17][17]; got != [2]int32{116, 39} {
		t.Fatalf("MBAFF bottom-field ref weights = %v, want top=116 bottom=39", got)
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
	recoverSimpleDPBTestFrames(idr, p, b)

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
	recoverSimpleDPBTestFrames(idr, p, b)

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
	recoverSimpleDPBTestFrames(idr, lower)
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

func TestSimpleFrameDPBPrimesReorderDelayAcrossTwoFuturePOCs(t *testing.T) {
	sps := simpleDPBTestSPS(5)
	sps.Log2MaxFrameNum = 4
	sps.PocType = 0
	sps.Log2MaxPocLSB = 4
	var probe simpleFrameDPB
	probe.reset()

	headers := []*SliceHeader{
		simpleDPBTestPOCHeader(sps, NALIDRSlice, PictureTypeI, 0, 0),
		simpleDPBTestPOCHeader(sps, NALSlice, PictureTypeP, 1, 3),
		simpleDPBTestPOCHeader(sps, NALSlice, PictureTypeP, 2, 6),
		simpleDPBTestPOCHeader(sps, NALSlice, PictureTypeB, 3, 1),
		simpleDPBTestPOCHeader(sps, NALSlice, PictureTypeB, 3, 2),
	}
	for i, header := range headers {
		nalRefIDC := uint8(1)
		if header.SliceTypeNoS == PictureTypeB {
			nalRefIDC = 0
		}
		if err := probe.primeOutputReorderDelayFromHeader(header, nalRefIDC); err != nil {
			t.Fatalf("prime header %d: %v", i, err)
		}
	}
	if probe.hasBFrames != 2 {
		t.Fatalf("primed delay = %d, want 2", probe.hasBFrames)
	}

	var dpb simpleFrameDPB
	dpb.reset()
	dpb.hasBFrames = probe.hasBFrames
	frames := []*DecodedFrame{
		simpleDPBTestFrame(sps, 0),
		simpleDPBTestFrame(sps, 1),
		simpleDPBTestFrame(sps, 2),
		simpleDPBTestFrame(sps, 3),
		simpleDPBTestFrame(sps, 4),
	}
	for i, poc := range []int32{0, 3, 6, 1, 2} {
		frames[i].poc = poc
	}
	frames[0].idrKeyFrame = true
	recoverSimpleDPBTestFrames(frames...)

	var out []*DecodedFrame
	for i, frame := range frames {
		if err := dpb.holdOutputFrame(frame, headers[i]); err != nil {
			t.Fatalf("hold frame %d: %v", i, err)
		}
		got, err := dpb.drainOutputFrames(false)
		if err != nil {
			t.Fatalf("drain frame %d: %v", i, err)
		}
		out = append(out, got...)
	}
	got, err := dpb.drainOutputFrames(true)
	if err != nil {
		t.Fatal(err)
	}
	out = append(out, got...)
	want := []*DecodedFrame{frames[0], frames[3], frames[4], frames[1], frames[2]}
	if len(out) != len(want) {
		t.Fatalf("output len = %d, want %d", len(out), len(want))
	}
	for i := range want {
		if out[i] != want[i] {
			t.Fatalf("output[%d] = poc %d, want poc %d", i, out[i].poc, want[i].poc)
		}
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

func TestSimpleFrameDPBMMCOResetClearsReorderHistory(t *testing.T) {
	sps := simpleDPBTestSPS(2)
	current := simpleDPBTestFrame(sps, 3)
	dpb := simpleFrameDPB{}
	dpb.resetLastPOCs()
	dpb.lastPOCs[0] = 10
	dpb.lastPOCs[1] = 8
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
	if !current.mmcoReset {
		t.Fatal("current frame was not marked as an MMCO reset")
	}
	for i, poc := range dpb.lastPOCs {
		if poc != math.MinInt32 {
			t.Fatalf("lastPOCs[%d] = %d, want reset history", i, poc)
		}
	}
}

func TestSimpleFrameDPBOutputsLeadingMMCOResetBeforeLowerPOC(t *testing.T) {
	sps := simpleDPBTestSPS(2)
	reset := simpleDPBTestFrame(sps, 0)
	reset.poc = 50
	reset.mmcoReset = true
	lower := simpleDPBTestFrame(sps, 1)
	lower.poc = 37
	recoverSimpleDPBTestFrames(reset, lower)
	dpb := simpleFrameDPB{
		delayed:    []*DecodedFrame{reset, lower},
		hasBFrames: 1,
	}

	out, err := dpb.drainOutputFrames(false)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0] != reset {
		t.Fatalf("output = %v, want leading MMCO reset before lower POC", out)
	}
	if len(dpb.delayed) != 1 || dpb.delayed[0] != lower {
		t.Fatalf("delayed after reset output = %v, want lower picture retained", dpb.delayed)
	}
}

func TestSimpleFrameDPBStopsOutputScanAtIDRBoundary(t *testing.T) {
	sps := simpleDPBTestSPS(3)
	beforeIDR := simpleDPBTestFrame(sps, 0)
	beforeIDR.poc = 8
	idr := simpleDPBTestFrame(sps, 1)
	idr.poc = 12
	idr.idrKeyFrame = true
	afterIDRLowerPOC := simpleDPBTestFrame(sps, 2)
	afterIDRLowerPOC.poc = 2
	recoverSimpleDPBTestFrames(beforeIDR, idr, afterIDRLowerPOC)
	dpb := simpleFrameDPB{
		delayed:    []*DecodedFrame{beforeIDR, idr, afterIDRLowerPOC},
		hasBFrames: 1,
	}

	out, err := dpb.drainOutputFrames(false)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0] != beforeIDR {
		t.Fatalf("output = %v, want frame before delayed IDR boundary", out)
	}
	if len(dpb.delayed) != 2 || dpb.delayed[0] != idr || dpb.delayed[1] != afterIDRLowerPOC {
		t.Fatalf("delayed after output = %v, want IDR boundary followed by lower POC", dpb.delayed)
	}
}

func TestSimpleFrameDPBDropsUnrecoveredOutputUntilSEIRecovery(t *testing.T) {
	sps := simpleDPBTestSPS(4)
	sps.BitstreamRestrictionFlag = 1
	sps.NumReorderFrames = 3
	var dpb simpleFrameDPB
	dpb.reset()

	leading0 := simpleDPBTestFrame(sps, 60)
	leading0.poc = 10
	leading1 := simpleDPBTestFrame(sps, 59)
	leading1.poc = 12
	leading2 := simpleDPBTestFrame(sps, 60)
	leading2.poc = 14
	recovery := simpleDPBTestFrame(sps, 58)
	recovery.poc = 16
	recovery.recovered = simpleFrameRecoveredSEI
	after := simpleDPBTestFrame(sps, 62)
	after.poc = 18

	frames := []*DecodedFrame{leading0, leading1, leading2, recovery, after}
	headers := []*SliceHeader{
		simpleDPBTestPOCHeader(sps, NALSlice, PictureTypeB, 60, 10),
		simpleDPBTestPOCHeader(sps, NALSlice, PictureTypeB, 59, 12),
		simpleDPBTestPOCHeader(sps, NALSlice, PictureTypeB, 60, 14),
		simpleDPBTestPOCHeader(sps, NALSlice, PictureTypeI, 58, 16),
		simpleDPBTestPOCHeader(sps, NALSlice, PictureTypeP, 62, 18),
	}

	var out []*DecodedFrame
	for i, frame := range frames {
		if err := dpb.holdOutputFrame(frame, headers[i]); err != nil {
			t.Fatalf("hold frame %d: %v", i, err)
		}
		got, err := dpb.drainOutputFrames(false)
		if err != nil {
			t.Fatalf("drain frame %d: %v", i, err)
		}
		out = append(out, got...)
	}
	got, err := dpb.drainOutputFrames(true)
	if err != nil {
		t.Fatal(err)
	}
	out = append(out, got...)

	want := []*DecodedFrame{recovery, after}
	if len(out) != len(want) {
		t.Fatalf("output len = %d, want %d", len(out), len(want))
	}
	for i := range want {
		if out[i] != want[i] {
			t.Fatalf("output[%d] = %p, want %p", i, out[i], want[i])
		}
	}
	if after.recovered&simpleFrameRecoveredSEI == 0 {
		t.Fatalf("post-recovery frame recovered = %#x, want SEI propagation", after.recovered)
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

func TestSimpleFrameDPBFrameNumGapsRefreshShortRefs(t *testing.T) {
	sps := simpleDPBTestSPS(4)
	sps.Log2MaxFrameNum = 8
	dpb := simpleFrameDPB{
		short: []*DecodedFrame{
			simpleDPBTestFrame(sps, 217),
			simpleDPBTestFrame(sps, 67),
			simpleDPBTestFrame(sps, 49),
			simpleDPBTestFrame(sps, 40),
		},
	}
	dpb.poc.prevFrameNum = 217
	sh := &SliceHeader{
		NALType:          NALSlice,
		SPS:              sps,
		FrameNum:         222,
		PictureStructure: PictureFrame,
	}

	if err := dpb.handleFrameNumGaps(sh, false); err != nil {
		t.Fatal(err)
	}
	if got, want := simpleDPBFrameNums(dpb.short), []uint32{221, 220, 219, 218}; !uint32SlicesEqual(got, want) {
		t.Fatalf("gap refs = %v, want %v", got, want)
	}
	for _, frame := range dpb.short {
		if frame == nil || !frame.invalidGap {
			t.Fatalf("gap frame invalid marker = %v for refs %v", frame, simpleDPBFrameNums(dpb.short))
		}
	}

	current := simpleDPBTestFrame(sps, 222)
	if err := dpb.markDecodedFrame(current, sh, 2); err != nil {
		t.Fatal(err)
	}
	if got, want := simpleDPBFrameNums(dpb.short), []uint32{222, 221, 220, 219}; !uint32SlicesEqual(got, want) {
		t.Fatalf("refs after current = %v, want %v", got, want)
	}
}

func TestSimpleFrameDPBFrameNumGapsShortenAcrossWrap(t *testing.T) {
	sps := simpleDPBTestSPS(3)
	sps.Log2MaxFrameNum = 8
	dpb := simpleFrameDPB{
		short: []*DecodedFrame{
			simpleDPBTestFrame(sps, 250),
			simpleDPBTestFrame(sps, 249),
			simpleDPBTestFrame(sps, 248),
		},
	}
	dpb.poc.prevFrameNum = 250
	sh := &SliceHeader{
		NALType:          NALSlice,
		SPS:              sps,
		FrameNum:         0,
		PictureStructure: PictureFrame,
	}

	if err := dpb.handleFrameNumGaps(sh, false); err != nil {
		t.Fatal(err)
	}
	if got, want := simpleDPBFrameNums(dpb.short), []uint32{255, 254, 253}; !uint32SlicesEqual(got, want) {
		t.Fatalf("wrapped gap refs = %v, want %v", got, want)
	}

	current := simpleDPBTestFrame(sps, 0)
	if err := dpb.markDecodedFrame(current, sh, 2); err != nil {
		t.Fatal(err)
	}
	if got, want := simpleDPBFrameNums(dpb.short), []uint32{0, 255, 254}; !uint32SlicesEqual(got, want) {
		t.Fatalf("wrapped refs after current = %v, want %v", got, want)
	}
}

func TestSimpleFrameDPBFrameNumGapsBootstrapMidStream(t *testing.T) {
	sps := simpleDPBTestSPS(2)
	sps.Log2MaxFrameNum = 5
	var dpb simpleFrameDPB
	dpb.reset()
	sh := &SliceHeader{
		NALType:          NALSlice,
		SPS:              sps,
		FrameNum:         27,
		PictureStructure: PictureFrame,
	}

	if err := dpb.handleFrameNumGaps(sh, false); err != nil {
		t.Fatal(err)
	}
	if got, want := simpleDPBFrameNums(dpb.short), []uint32{26, 25}; !uint32SlicesEqual(got, want) {
		t.Fatalf("bootstrap gap refs = %v, want %v", got, want)
	}
	for _, frame := range dpb.short {
		if frame == nil || !frame.invalidGap {
			t.Fatalf("bootstrap gap frame invalid marker = %v for refs %v", frame, simpleDPBFrameNums(dpb.short))
		}
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
		Y:                make([]uint8, 16*16),
		Cb:               make([]uint8, 8*8),
		Cr:               make([]uint8, 8*8),
		LumaStride:       16,
		ChromaStride:     8,
		Width:            int(sps.Width),
		Height:           int(sps.Height),
		MBWidth:          int(sps.MBWidth),
		MBHeight:         int(sps.MBHeight),
		ChromaFormatIDC:  int(sps.ChromaFormatIDC),
		BitDepthLuma:     int(sps.BitDepthLuma),
		BitDepthChroma:   int(sps.BitDepthChroma),
		frameMBSOnlyFlag: sps.FrameMBSOnlyFlag,
		frameNum:         frameNum,
	}
}

func recoverSimpleDPBTestFrames(frames ...*DecodedFrame) {
	for _, frame := range frames {
		frame.recovered |= simpleFrameRecoveredIDR
	}
}

func fakeDecodedFrameSliceLen(n int) []*DecodedFrame {
	if n <= 0 {
		return nil
	}
	var frame *DecodedFrame
	return *(*[]*DecodedFrame)(unsafe.Pointer(&reflect.SliceHeader{
		Data: uintptr(unsafe.Pointer(&frame)),
		Len:  n,
		Cap:  n,
	}))
}

func fakeSimpleRefEntrySliceLen(n int) []simpleRefEntry {
	if n <= 0 {
		return nil
	}
	var entry simpleRefEntry
	return *(*[]simpleRefEntry)(unsafe.Pointer(&reflect.SliceHeader{
		Data: uintptr(unsafe.Pointer(&entry)),
		Len:  n,
		Cap:  n,
	}))
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

func uint32SlicesEqual(a []uint32, b []uint32) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
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
