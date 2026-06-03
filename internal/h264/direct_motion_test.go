// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

import "testing"

func TestPredTemporalDirect16x16MapsColocatedMotion(t *testing.T) {
	m, col, idr := newTemporalDirectTestTables(t, MBType16x16|MBTypeP0L0|MBTypeP1L0)
	bxy := int(col.tables.MB2BXY[0])
	col.tables.RefIndex[0][0] = 0
	col.tables.MotionVal[0][bxy] = [2]int16{4, 2}

	var cache macroblockMotionCache
	var sub [4]uint32
	mbType := MBTypeDirect2 | MBTypeL0L1
	err := m.predDirectMotionFrame(&cache, 0, &mbType, &sub, h264DirectMotionContext{
		RefEntries: [2][]simpleRefEntry{
			{{frame: idr}},
			{{frame: col}},
		},
		CurPOC:             2,
		Direct8x8Inference: true,
	})
	if err != nil {
		t.Fatalf("temporal direct failed: %v", err)
	}
	if mbType&(MBTypeDirect2|MBType16x16|MBTypeL0L1) != (MBTypeDirect2 | MBType16x16 | MBTypeL0L1) {
		t.Fatalf("mbType = %#x", mbType)
	}
	base := int(h264Scan8[0])
	if cache.Ref[0][base] != 0 || cache.Ref[1][base] != 0 {
		t.Fatalf("refs = %d/%d, want 0/0", cache.Ref[0][base], cache.Ref[1][base])
	}
	if cache.MV[0][base] != ([2]int16{2, 1}) || cache.MV[1][base] != ([2]int16{-2, -1}) {
		t.Fatalf("mvs = %v/%v", cache.MV[0][base], cache.MV[1][base])
	}
}

func TestPredTemporalDirectAllowsFieldInterlacedColocatedMotion(t *testing.T) {
	m, col, idr := newTemporalDirectTestTables(t, MBType16x16|MBTypeP0L0|MBTypeP1L0|MBTypeInterlaced)
	bxy := int(col.tables.MB2BXY[0])
	col.tables.RefIndex[0][0] = 0
	col.tables.MotionVal[0][bxy] = [2]int16{4, 2}
	col.refEntries[0] = []simpleRefEntry{{frame: idr, pictureStructure: PictureTopField, poc: 0}}

	var cache macroblockMotionCache
	var sub [4]uint32
	mbType := MBTypeDirect2 | MBTypeL0L1 | MBTypeInterlaced
	err := m.predDirectMotionFrame(&cache, 0, &mbType, &sub, h264DirectMotionContext{
		RefEntries: [2][]simpleRefEntry{
			{{frame: idr, pictureStructure: PictureTopField, poc: 0}},
			{{frame: col, pictureStructure: PictureTopField, poc: 4}},
		},
		CurPOC:             2,
		PictureStructure:   PictureTopField,
		Direct8x8Inference: true,
	})
	if err != nil {
		t.Fatalf("field temporal direct failed: %v", err)
	}
	wantType := MBType16x16 | MBTypeL0L1 | MBTypeDirect2 | MBTypeInterlaced
	if mbType != wantType {
		t.Fatalf("field mbType = %#x, want %#x", mbType, wantType)
	}
	base := int(h264Scan8[0])
	if cache.Ref[0][base] != 0 || cache.Ref[1][base] != 0 {
		t.Fatalf("field refs = %d/%d, want 0/0", cache.Ref[0][base], cache.Ref[1][base])
	}
	if cache.MV[0][base] != ([2]int16{2, 1}) || cache.MV[1][base] != ([2]int16{-2, -1}) {
		t.Fatalf("field mvs = %v/%v", cache.MV[0][base], cache.MV[1][base])
	}
}

func TestPredTemporalDirectFieldPictureMapsExactColocatedFieldRef(t *testing.T) {
	m, col, past := newTemporalDirectTestTables(t, MBType16x16|MBTypeP0L0|MBTypeP1L0|MBTypeInterlaced)
	past.fieldPOC = [2]int32{0, 2}
	past.poc = 0
	col.fieldPOC = [2]int32{4, 6}
	col.poc = 4
	col.refEntries[0] = []simpleRefEntry{
		{frame: past, picID: 4, pictureStructure: PictureTopField, poc: 0},
		{frame: past, picID: 5, pictureStructure: PictureBottomField, poc: 2},
	}
	col.tables.RefIndex[0][0] = 1
	col.tables.MotionVal[0][col.tables.MB2BXY[0]] = [2]int16{4, 0}

	var cache macroblockMotionCache
	var sub [4]uint32
	mbType := MBTypeDirect2 | MBTypeL0L1 | MBTypeInterlaced
	err := m.predDirectMotionFrame(&cache, 0, &mbType, &sub, h264DirectMotionContext{
		RefEntries: [2][]simpleRefEntry{
			{
				{frame: past, picID: 4, pictureStructure: PictureTopField, poc: 0},
				{frame: past, picID: 5, pictureStructure: PictureBottomField, poc: 2},
			},
			{{frame: col, pictureStructure: PictureTopField, poc: 4}},
		},
		CurPOC:             1,
		CurFieldPOC:        [2]int32{1, 3},
		PictureStructure:   PictureTopField,
		Direct8x8Inference: true,
	})
	if err != nil {
		t.Fatalf("field temporal direct exact-ref mapping failed: %v", err)
	}
	base := int(h264Scan8[0])
	if cache.Ref[0][base] != 1 {
		t.Fatalf("field temporal ref = %d, want exact bottom-field list0 ref 1", cache.Ref[0][base])
	}
}

func TestPredTemporalDirectBottomFieldPictureMapsByColmapPictureID(t *testing.T) {
	m, col, past := newTemporalDirectTestTables(t, MBType8x8|MBTypeP0L0|MBTypeP1L0|MBTypeInterlaced)
	next := &DecodedFrame{poc: 4, fieldPOC: [2]int32{4, 6}, frameNum: 1}
	past.fieldPOC = [2]int32{0, 2}
	past.poc = 0
	past.frameNum = 0
	col.fieldPOC = [2]int32{8, 10}
	col.poc = 8
	col.refEntries[0] = []simpleRefEntry{
		{frame: past, picID: 1, pictureStructure: PictureBottomField, poc: 2},
		{frame: next, picID: 2, pictureStructure: PictureTopField, poc: 4},
		{frame: past, picID: 0, pictureStructure: PictureTopField, poc: 0},
	}
	for i8 := 0; i8 < 4; i8++ {
		col.tables.RefIndex[0][i8] = 0
	}
	col.tables.RefIndex[0][0] = 1
	col.tables.MotionVal[0][col.tables.MB2BXY[0]] = [2]int16{4, 0}

	var cache macroblockMotionCache
	var sub [4]uint32
	mbType := MBTypeDirect2 | MBTypeL0L1 | MBTypeInterlaced
	err := m.predDirectMotionFrame(&cache, 0, &mbType, &sub, h264DirectMotionContext{
		RefEntries: [2][]simpleRefEntry{
			{
				{frame: past, picID: 1, pictureStructure: PictureBottomField, poc: 2},
				{frame: past, picID: 0, pictureStructure: PictureTopField, poc: 0},
				{frame: next, picID: 3, pictureStructure: PictureBottomField, poc: 6},
				{frame: next, picID: 2, pictureStructure: PictureTopField, poc: 4},
			},
			{{frame: col, pictureStructure: PictureBottomField, poc: 10}},
		},
		CurPOC:             3,
		CurFieldPOC:        [2]int32{1, 3},
		PictureStructure:   PictureBottomField,
		Direct8x8Inference: true,
	})
	if err != nil {
		t.Fatalf("bottom-field temporal direct colmap mapping failed: %v", err)
	}
	base := int(h264Scan8[0])
	if cache.Ref[0][base] != 3 {
		t.Fatalf("bottom-field temporal ref = %d, want colmap picture-id list0 ref 3", cache.Ref[0][base])
	}
}

func TestPredTemporalDirectFieldPictureFrameRefExpandsToCurrentField(t *testing.T) {
	m, col, _ := newTemporalDirectTestTables(t, MBType8x8|MBTypeP0L0|MBTypeP1L0)
	matching := &DecodedFrame{poc: 8, fieldPOC: [2]int32{8, 10}, frameNum: 1}
	older := &DecodedFrame{poc: 0, fieldPOC: [2]int32{0, 2}, frameNum: 0}
	col.fieldPOC = [2]int32{12, 14}
	col.poc = 12
	col.refEntries[0] = []simpleRefEntry{
		{frame: matching, picID: matching.frameNum, pictureStructure: PictureFrame, poc: matching.poc},
	}
	for i8 := 0; i8 < 4; i8++ {
		col.tables.RefIndex[0][i8] = 0
	}
	col.tables.MotionVal[0][col.tables.MB2BXY[0]] = [2]int16{4, 0}

	var cache macroblockMotionCache
	var sub [4]uint32
	mbType := MBTypeDirect2 | MBTypeL0L1
	err := m.predDirectMotionFrame(&cache, 0, &mbType, &sub, h264DirectMotionContext{
		RefEntries: [2][]simpleRefEntry{
			{
				{frame: matching, picID: 2*matching.frameNum + 1, pictureStructure: PictureTopField, poc: matching.fieldPOC[0]},
				{frame: older, picID: 2*older.frameNum + 1, pictureStructure: PictureTopField, poc: older.fieldPOC[0]},
			},
			{{frame: col, pictureStructure: PictureTopField, poc: col.fieldPOC[0]}},
		},
		CurPOC:             9,
		CurFieldPOC:        [2]int32{9, 11},
		PictureStructure:   PictureTopField,
		Direct8x8Inference: true,
	})
	if err != nil {
		t.Fatalf("field temporal direct frame-ref expansion failed: %v", err)
	}
	base := int(h264Scan8[0])
	if cache.Ref[0][base] != 0 {
		t.Fatalf("field temporal frame ref = %d, want current-field list0 ref 0", cache.Ref[0][base])
	}
}

func TestPredTemporalDirectAllowsFrameCurrentOverInterlacedColocated(t *testing.T) {
	m, err := newMacroblockTables(1, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	colTables, err := newMacroblockTables(1, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	idr := &DecodedFrame{poc: 0}
	col := &DecodedFrame{
		poc:      4,
		fieldPOC: [2]int32{0, 4},
		mbaff:    true,
		tables:   colTables,
		refEntries: [2][]simpleRefEntry{
			{{frame: idr}},
		},
	}
	colTables.MacroblockTyp[0] = MBType16x16 | MBTypeP0L0 | MBTypeP1L0 | MBTypeInterlaced
	colTables.MacroblockTyp[colTables.MBStride] = MBType16x16 | MBTypeP0L0 | MBTypeP1L0 | MBTypeInterlaced
	colTables.RefIndex[0][0] = 0
	colTables.RefIndex[0][1] = 0
	colTables.RefIndex[0][4*colTables.MBStride] = 0
	colTables.RefIndex[0][4*colTables.MBStride+1] = 0
	colTables.MotionVal[0][colTables.MB2BXY[0]] = [2]int16{4, 2}
	colTables.MotionVal[0][int(colTables.MB2BXY[0])+colTables.BStride] = [2]int16{4, 2}
	colTables.MotionVal[0][colTables.MB2BXY[colTables.MBStride]] = [2]int16{4, 2}
	colTables.MotionVal[0][int(colTables.MB2BXY[colTables.MBStride])+colTables.BStride] = [2]int16{4, 2}

	var cache macroblockMotionCache
	var sub [4]uint32
	mbType := MBTypeDirect2 | MBTypeL0L1
	err = m.predDirectMotionFrame(&cache, 0, &mbType, &sub, h264DirectMotionContext{
		RefEntries: [2][]simpleRefEntry{
			{{frame: idr}},
			{{frame: col, pictureStructure: PictureFrame, poc: 4}},
		},
		CurPOC:             2,
		PictureStructure:   PictureFrame,
		Direct8x8Inference: true,
	})
	if err != nil {
		t.Fatalf("frame-over-field temporal direct failed: %v", err)
	}
	if !is16x16(mbType) || !isDirect(mbType) {
		t.Fatalf("mbType = %#x, want direct 16x16 from interlaced colocated single_col branch", mbType)
	}
	base := int(h264Scan8[0])
	if cache.MV[0][base] != ([2]int16{2, 2}) || cache.MV[1][base] != ([2]int16{-2, -2}) {
		t.Fatalf("frame-over-field mvs = %v/%v, want y-shifted temporal scale", cache.MV[0][base], cache.MV[1][base])
	}
}

func TestPredTemporalDirectFrameCurrentOverInterlacedColocatedKeepsPartitionShape(t *testing.T) {
	m, err := newMacroblockTables(1, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	colTables, err := newMacroblockTables(1, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	idr := &DecodedFrame{poc: 0}
	col := &DecodedFrame{
		poc:      4,
		fieldPOC: [2]int32{4, 20},
		mbaff:    true,
		tables:   colTables,
		refEntries: [2][]simpleRefEntry{
			{{frame: idr, pictureStructure: PictureFrame, poc: 0}},
		},
	}
	colTables.MacroblockTyp[0] = MBType16x8 | MBTypeP0L0 | MBTypeP1L0 | MBTypeInterlaced
	colTables.MacroblockTyp[colTables.MBStride] = MBType8x16 | MBTypeP0L0 | MBTypeP1L0 | MBTypeInterlaced
	colTables.RefIndex[0][0] = 0
	colTables.RefIndex[0][1] = 0
	colTables.MotionVal[0][colTables.MB2BXY[0]] = [2]int16{4, 2}
	colTables.MotionVal[0][int(colTables.MB2BXY[0])+3*colTables.BStride] = [2]int16{4, 2}

	var cache macroblockMotionCache
	var sub [4]uint32
	mbType := MBTypeDirect2 | MBTypeL0L1
	err = m.predDirectMotionFrame(&cache, 0, &mbType, &sub, h264DirectMotionContext{
		RefEntries: [2][]simpleRefEntry{
			{{frame: idr, pictureStructure: PictureFrame, poc: 0}},
			{{frame: col, pictureStructure: PictureFrame, poc: 4}},
		},
		CurPOC:             2,
		PictureStructure:   PictureFrame,
		Direct8x8Inference: true,
	})
	if err != nil {
		t.Fatalf("frame-over-field partition temporal direct failed: %v", err)
	}
	if !is16x8(mbType) || is8x8(mbType) || !isDirect(mbType) {
		t.Fatalf("mbType = %#x, want colocated 16x8 shape, not B_8x8", mbType)
	}
}

func TestPredTemporalDirectFrameCurrentOverInterlacedColocatedUsesOldRefColmap(t *testing.T) {
	m, err := newMacroblockTables(1, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	colTables, err := newMacroblockTables(1, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	past0 := &DecodedFrame{poc: -4, fieldPOC: [2]int32{-4, -2}, frameNum: 0}
	past1 := &DecodedFrame{poc: 0, fieldPOC: [2]int32{0, 2}, frameNum: 1}
	past2 := &DecodedFrame{poc: 8, fieldPOC: [2]int32{8, 10}, frameNum: 2}
	col := &DecodedFrame{
		poc:      4,
		fieldPOC: [2]int32{4, 20},
		mbaff:    true,
		tables:   colTables,
		refEntries: [2][]simpleRefEntry{
			{
				{frame: past0, picID: past0.frameNum, pictureStructure: PictureFrame, poc: past0.poc},
				{frame: past1, picID: past1.frameNum, pictureStructure: PictureFrame, poc: past1.poc},
				{frame: past2, picID: past2.frameNum, pictureStructure: PictureFrame, poc: past2.poc},
			},
		},
		fieldRefEntries: [2][2][]simpleRefEntry{
			{
				{
					{frame: past0, picID: 2*past0.frameNum + 1, pictureStructure: PictureTopField, poc: past0.fieldPOC[0]},
					{frame: past2, picID: 2*past2.frameNum + 1, pictureStructure: PictureTopField, poc: past2.fieldPOC[0]},
					{frame: past2, picID: 2*past2.frameNum + 1, pictureStructure: PictureTopField, poc: past2.fieldPOC[0]},
					{frame: past0, picID: 2*past0.frameNum + 1, pictureStructure: PictureTopField, poc: past0.fieldPOC[0]},
				},
				nil,
			},
		},
	}
	colTables.MacroblockTyp[0] = MBType16x8 | MBTypeP0L0 | MBTypeP1L0 | MBTypeInterlaced
	colTables.RefIndex[0][0] = 2
	colTables.RefIndex[0][1] = 3
	colTables.MotionVal[0][colTables.MB2BXY[0]] = [2]int16{4, 2}
	colTables.MotionVal[0][int(colTables.MB2BXY[0])+3*colTables.BStride] = [2]int16{4, 2}

	var cache macroblockMotionCache
	var sub [4]uint32
	mbType := MBTypeDirect2 | MBTypeL0L1
	err = m.predDirectMotionFrame(&cache, 0, &mbType, &sub, h264DirectMotionContext{
		RefEntries: [2][]simpleRefEntry{
			{
				{frame: past0, picID: past0.frameNum, pictureStructure: PictureFrame, poc: past0.poc},
				{frame: past1, picID: past1.frameNum, pictureStructure: PictureFrame, poc: past1.poc},
				{frame: past2, picID: past2.frameNum, pictureStructure: PictureFrame, poc: past2.poc},
			},
			{{frame: col, pictureStructure: PictureFrame, poc: col.poc}},
		},
		CurPOC:             2,
		PictureStructure:   PictureFrame,
		Direct8x8Inference: true,
	})
	if err != nil {
		t.Fatalf("frame-over-field temporal direct field-ref map failed: %v", err)
	}
	base := int(h264Scan8[0])
	if cache.Ref[0][base] != 1 {
		t.Fatalf("frame-over-field left ref = %d, want old-ref colmap frame ref 1", cache.Ref[0][base])
	}
	right := int(h264Scan8[4])
	if cache.Ref[0][right] != 1 {
		t.Fatalf("frame-over-field right ref = %d, want raw field ref 3 to map through old-ref 1", cache.Ref[0][right])
	}
}

func TestTemporalDirectFrameCurrentColFieldColmapCollapsesVirtualFieldRefs(t *testing.T) {
	past0 := &DecodedFrame{poc: -4, fieldPOC: [2]int32{-4, -2}, frameNum: 0}
	past1 := &DecodedFrame{poc: 0, fieldPOC: [2]int32{0, 2}, frameNum: 1}
	col := &DecodedFrame{
		mbaff: true,
		refEntries: [2][]simpleRefEntry{
			{
				{frame: past0, picID: past0.frameNum, pictureStructure: PictureFrame, poc: past0.poc},
				{frame: past1, picID: past1.frameNum, pictureStructure: PictureFrame, poc: past1.poc},
			},
		},
		fieldRefEntries: [2][2][]simpleRefEntry{
			{
				{
					{frame: past0, picID: 2*past0.frameNum + 1, pictureStructure: PictureTopField, poc: past0.fieldPOC[0]},
					{frame: past0, picID: 2*past0.frameNum + 1, pictureStructure: PictureTopField, poc: past0.fieldPOC[0]},
					{frame: past0, picID: 2*past0.frameNum + 1, pictureStructure: PictureTopField, poc: past0.fieldPOC[0]},
					{frame: past0, picID: 2*past0.frameNum + 1, pictureStructure: PictureTopField, poc: past0.fieldPOC[0]},
				},
				nil,
			},
		},
	}
	ctx := h264DirectMotionContext{
		RefEntries: [2][]simpleRefEntry{
			{
				{frame: past0, picID: past0.frameNum, pictureStructure: PictureFrame, poc: past0.poc},
				{frame: past1, picID: past1.frameNum, pictureStructure: PictureFrame, poc: past1.poc},
			},
			{{frame: col, pictureStructure: PictureFrame, poc: col.poc}},
		},
		PictureStructure: PictureFrame,
	}

	for _, rawRef := range []int{2, 3} {
		got, err := temporalDirectMapColFieldRefToFrameList0(ctx, 0, rawRef, 0)
		if err != nil {
			t.Fatalf("raw field ref %d colmap failed: %v", rawRef, err)
		}
		if got != 1 {
			t.Fatalf("raw field ref %d mapped to %d, want old_ref 1 frame ref", rawRef, got)
		}
	}
}

func TestTemporalDirectFrameCurrentColFieldPictureKeepsFieldRefIndex(t *testing.T) {
	past0 := &DecodedFrame{poc: -4, fieldPOC: [2]int32{-4, -2}, frameNum: 0}
	past1 := &DecodedFrame{poc: 0, fieldPOC: [2]int32{0, 2}, frameNum: 1}
	col := &DecodedFrame{
		refEntries: [2][]simpleRefEntry{
			{
				{frame: past0, picID: past0.frameNum, pictureStructure: PictureFrame, poc: past0.poc},
			},
		},
		fieldRefEntries: [2][2][]simpleRefEntry{
			{
				{
					{frame: past0, picID: 2*past0.frameNum + 1, pictureStructure: PictureTopField, poc: past0.fieldPOC[0]},
					{frame: past1, picID: 2*past1.frameNum + 1, pictureStructure: PictureTopField, poc: past1.fieldPOC[0]},
				},
				nil,
			},
		},
	}
	ctx := h264DirectMotionContext{
		RefEntries: [2][]simpleRefEntry{
			{
				{frame: past0, picID: past0.frameNum, pictureStructure: PictureFrame, poc: past0.poc},
				{frame: past1, picID: past1.frameNum, pictureStructure: PictureFrame, poc: past1.poc},
			},
			{{frame: col, pictureStructure: PictureTopField, poc: col.fieldPOC[0]}},
		},
		PictureStructure: PictureFrame,
	}

	got, err := temporalDirectMapColFieldRefToFrameList0(ctx, 0, 1, 0)
	if err != nil {
		t.Fatalf("field-picture colmap failed: %v", err)
	}
	if got != 1 {
		t.Fatalf("field-picture raw ref mapped to %d, want field-ref slot 1", got)
	}
}

func TestTemporalDirectFrameCurrentColFieldPicturePrefersFrameMatchBeforePicID(t *testing.T) {
	colliding := &DecodedFrame{poc: 8, fieldPOC: [2]int32{8, 10}, frameNum: 2}
	matching := &DecodedFrame{poc: 0, fieldPOC: [2]int32{0, 2}, frameNum: 1}
	col := &DecodedFrame{
		fieldRefEntries: [2][2][]simpleRefEntry{
			{
				{
					{frame: colliding, picID: 5, pictureStructure: PictureTopField, poc: colliding.fieldPOC[0]},
					{frame: colliding, picID: 4, pictureStructure: PictureBottomField, poc: colliding.fieldPOC[1]},
					{frame: matching, picID: 3, pictureStructure: PictureTopField, poc: matching.fieldPOC[0]},
					{frame: matching, picID: 2, pictureStructure: PictureBottomField, poc: matching.fieldPOC[1]},
				},
				nil,
			},
		},
	}
	ctx := h264DirectMotionContext{
		RefEntries: [2][]simpleRefEntry{
			{
				{frame: colliding, picID: 2, pictureStructure: PictureFrame, poc: colliding.poc},
				{frame: matching, picID: matching.frameNum, pictureStructure: PictureFrame, poc: matching.poc},
			},
			{{frame: col, pictureStructure: PictureFrame, poc: col.poc}},
		},
		PictureStructure: PictureFrame,
	}

	got, err := temporalDirectMapColFieldRefToFrameList0(ctx, 0, 3, 0)
	if err != nil {
		t.Fatalf("field-picture picID collision colmap failed: %v", err)
	}
	if got != 1 {
		t.Fatalf("field-picture picID collision mapped to %d, want same-frame ref 1", got)
	}
}

func TestDirectColocatedLayoutFrameCurrentUsesColParityAndBottomHalf(t *testing.T) {
	m, err := newMacroblockTables(1, 4, 1)
	if err != nil {
		t.Fatal(err)
	}
	col, err := newMacroblockTables(1, 4, 1)
	if err != nil {
		t.Fatal(err)
	}
	for mbXY := range col.MacroblockTyp {
		col.MacroblockTyp[mbXY] = MBType16x16 | MBTypeP0L0 | MBTypeInterlaced
	}
	colFrame := &DecodedFrame{fieldPOC: [2]int32{20, 4}, tables: col}

	mbXY := 3 * m.MBStride
	layout, err := m.directColocatedLayout(col, mbXY, MBTypeDirect2|MBTypeL0L1, h264DirectMotionContext{
		RefEntries:       [2][]simpleRefEntry{nil, {{frame: colFrame, pictureStructure: PictureFrame}}},
		CurPOC:           0,
		PictureStructure: PictureFrame,
	})
	if err != nil {
		t.Fatalf("layout failed: %v", err)
	}
	if layout.MBXY != mbXY || layout.B8Stride != 0 || !layout.InterlacedMismatch {
		t.Fatalf("layout mbxy/b8/mismatch = %d/%d/%v, want bottom colocated field mismatch", layout.MBXY, layout.B8Stride, layout.InterlacedMismatch)
	}
	if layout.RefBase != 4*mbXY+2 || layout.MVBase != int(col.MB2BXY[mbXY])+2*col.BStride {
		t.Fatalf("layout bases = ref %d mv %d, want bottom-half offsets", layout.RefBase, layout.MVBase)
	}
}

func TestPredTemporalDirectAllowsFieldCurrentOverFrameColocated(t *testing.T) {
	m, err := newMacroblockTables(1, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	colTables, err := newMacroblockTables(1, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	idr := &DecodedFrame{poc: 0, fieldPOC: [2]int32{0, 0}}
	col := &DecodedFrame{
		poc:      4,
		fieldPOC: [2]int32{4, 6},
		tables:   colTables,
		refEntries: [2][]simpleRefEntry{
			{{frame: idr, pictureStructure: PictureFrame, poc: 0}},
		},
	}
	colTables.MacroblockTyp[0] = MBType16x16 | MBTypeP0L0 | MBTypeP1L0
	colTables.MacroblockTyp[colTables.MBStride] = MBType16x16 | MBTypeP0L0 | MBTypeP1L0
	colTables.RefIndex[0][0] = 0
	colTables.MotionVal[0][colTables.MB2BXY[0]] = [2]int16{4, 4}

	var cache macroblockMotionCache
	var sub [4]uint32
	mbType := MBTypeDirect2 | MBTypeL0L1 | MBTypeInterlaced
	err = m.predDirectMotionFrame(&cache, 0, &mbType, &sub, h264DirectMotionContext{
		RefEntries: [2][]simpleRefEntry{
			{{frame: idr, pictureStructure: PictureTopField, poc: 0}},
			{{frame: col, pictureStructure: PictureFrame, poc: 4}},
		},
		CurPOC:             2,
		PictureStructure:   PictureTopField,
		Direct8x8Inference: true,
	})
	if err != nil {
		t.Fatalf("field-over-frame temporal direct failed: %v", err)
	}
	if !is16x8(mbType) || !isDirect(mbType) {
		t.Fatalf("mbType = %#x, want field direct 16x8", mbType)
	}
	base := int(h264Scan8[0])
	if cache.MV[0][base] != ([2]int16{2, 1}) || cache.MV[1][base] != ([2]int16{-2, -1}) {
		t.Fatalf("field-over-frame mvs = %v/%v, want field-scaled temporal direct", cache.MV[0][base], cache.MV[1][base])
	}
}

func TestPredTemporalDirectMapsColocatedRefByPictureID(t *testing.T) {
	m, col, _ := newTemporalDirectTestTables(t, MBType16x16|MBTypeP0L0)
	col.tables.RefIndex[0][0] = 0
	col.tables.MotionVal[0][col.tables.MB2BXY[0]] = [2]int16{4, 2}

	firstCurrentRef := &DecodedFrame{poc: -4, frameNum: 1}
	matchingCurrentRef := &DecodedFrame{poc: 0, frameNum: 7}
	colocatedRef := &DecodedFrame{poc: 0, frameNum: 7}
	col.refEntries[0] = []simpleRefEntry{{frame: colocatedRef, picID: 7}}

	var cache macroblockMotionCache
	var sub [4]uint32
	mbType := MBTypeDirect2 | MBTypeL0L1
	err := m.predDirectMotionFrame(&cache, 0, &mbType, &sub, h264DirectMotionContext{
		RefEntries: [2][]simpleRefEntry{
			{
				{frame: firstCurrentRef, picID: 1},
				{frame: matchingCurrentRef, picID: 7},
			},
			{{frame: col}},
		},
		CurPOC:             2,
		Direct8x8Inference: true,
	})
	if err != nil {
		t.Fatalf("temporal direct picture-id map failed: %v", err)
	}
	base := int(h264Scan8[0])
	if cache.Ref[0][base] != 1 || cache.Ref[1][base] != 0 {
		t.Fatalf("refs = %d/%d, want 1/0", cache.Ref[0][base], cache.Ref[1][base])
	}
	if cache.MV[0][base] != ([2]int16{2, 1}) || cache.MV[1][base] != ([2]int16{-2, -1}) {
		t.Fatalf("mvs = %v/%v, want picture-id mapped scale", cache.MV[0][base], cache.MV[1][base])
	}
}

func TestPredTemporalDirectMapsMBAFFColocatedFieldRef(t *testing.T) {
	m, err := newMacroblockTables(1, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	colTables, err := newMacroblockTables(1, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	colTables.MacroblockTyp[0] = MBType16x16 | MBTypeP0L0 | MBTypeInterlaced
	colTables.RefIndex[0][0] = 3
	colTables.MotionVal[0][colTables.MB2BXY[0]] = [2]int16{4, 2}

	past0 := &DecodedFrame{poc: -4, fieldPOC: [2]int32{-4, -2}, frameNum: 0}
	past1 := &DecodedFrame{poc: 0, fieldPOC: [2]int32{0, 2}, frameNum: 1}
	col := &DecodedFrame{
		poc:      4,
		fieldPOC: [2]int32{4, 6},
		mbaff:    true,
		tables:   colTables,
		refEntries: [2][]simpleRefEntry{
			{
				{frame: past0, picID: past0.frameNum, pictureStructure: PictureFrame, poc: past0.poc},
				{frame: past1, picID: past1.frameNum, pictureStructure: PictureFrame, poc: past1.poc},
			},
		},
	}

	var cache macroblockMotionCache
	var sub [4]uint32
	mbType := MBTypeDirect2 | MBTypeL0L1
	err = m.predDirectMotionFrame(&cache, 0, &mbType, &sub, h264DirectMotionContext{
		RefEntries: [2][]simpleRefEntry{
			{
				{frame: past0, picID: past0.frameNum, pictureStructure: PictureFrame, poc: past0.poc},
				{frame: past1, picID: past1.frameNum, pictureStructure: PictureFrame, poc: past1.poc},
			},
			{{frame: col, pictureStructure: PictureFrame, poc: col.poc}},
		},
		CurPOC:             2,
		PictureStructure:   PictureFrame,
		Direct8x8Inference: true,
	})
	if err != nil {
		t.Fatalf("temporal direct MBAFF field colmap failed: %v", err)
	}
	base := int(h264Scan8[0])
	if cache.Ref[0][base] != 1 || cache.Ref[1][base] != 0 {
		t.Fatalf("refs = %d/%d, want mapped list0 ref 1 and list1 ref 0", cache.Ref[0][base], cache.Ref[1][base])
	}
	if cache.MV[0][base] != ([2]int16{2, 2}) || cache.MV[1][base] != ([2]int16{-2, -2}) {
		t.Fatalf("mvs = %v/%v, want field-ref temporal scale", cache.MV[0][base], cache.MV[1][base])
	}
}

func TestTemporalDirectFrameMBAFFBottomFieldColmapUsesRFieldXOR(t *testing.T) {
	past := &DecodedFrame{poc: 0, fieldPOC: [2]int32{0, 2}, frameNum: 7}
	col := &DecodedFrame{
		refEntries: [2][]simpleRefEntry{
			{{frame: past, picID: past.frameNum, pictureStructure: PictureFrame, poc: past.poc}},
		},
	}
	ctx := h264DirectMotionContext{
		RefEntries: [2][]simpleRefEntry{
			{{frame: past, picID: past.frameNum, pictureStructure: PictureFrame, poc: past.poc}},
			{{frame: col, pictureStructure: PictureFrame, poc: col.poc}},
		},
		PictureStructure: PictureFrame,
	}

	got, err := temporalDirectMapColToList0Field(ctx, 0, 1, true, 1)
	if err != nil {
		t.Fatalf("bottom-field colmap failed: %v", err)
	}
	if got != 1 {
		t.Fatalf("bottom-field colmap ref = %d, want visible top-field ref 1", got)
	}
	got, err = temporalDirectMapColToList0Field(ctx, 0, 0, true, 1)
	if err != nil {
		t.Fatalf("bottom-field colmap bottom ref failed: %v", err)
	}
	if got != 0 {
		t.Fatalf("bottom-field colmap bottom ref = %d, want visible bottom-field ref 0", got)
	}
}

func TestTemporalDirectFrameMBAFFFieldColmapSkipsPicIDCollision(t *testing.T) {
	colliding := &DecodedFrame{poc: 65548, fieldPOC: [2]int32{65548, 65548}, frameNum: 3}
	wrong := &DecodedFrame{poc: 65544, fieldPOC: [2]int32{65544, 65545}, frameNum: 4}
	matching := &DecodedFrame{poc: 65542, fieldPOC: [2]int32{65542, 65543}, frameNum: 1}
	col := &DecodedFrame{
		refEntries: [2][]simpleRefEntry{
			{{frame: matching, picID: 3, pictureStructure: PictureFrame, poc: matching.poc}},
		},
	}
	ctx := h264DirectMotionContext{
		RefEntries: [2][]simpleRefEntry{
			{
				{frame: colliding, picID: 3, pictureStructure: PictureFrame, poc: colliding.poc},
				{frame: wrong, picID: wrong.frameNum, pictureStructure: PictureFrame, poc: wrong.poc},
				{frame: matching, picID: matching.frameNum, pictureStructure: PictureFrame, poc: matching.poc},
			},
			{{frame: col, pictureStructure: PictureFrame, poc: col.poc}},
		},
		PictureStructure: PictureFrame,
	}

	got, err := temporalDirectMapColToList0Field(ctx, 0, 0, true, 0)
	if err != nil {
		t.Fatalf("frame-MBAFF field colmap collision failed: %v", err)
	}
	if got != 4 {
		t.Fatalf("frame-MBAFF field colmap = %d, want expanded same-frame field ref 4", got)
	}
}

func TestPredTemporalDirectFieldPictureUsesMBAFFColocatedFieldRefOffset(t *testing.T) {
	m, err := newMacroblockTables(1, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	colMBs, err := newMacroblockTables(1, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	target := &DecodedFrame{poc: 0, fieldPOC: [2]int32{0, 0}, frameNum: 2}
	past := &DecodedFrame{poc: 6, fieldPOC: [2]int32{6, 8}, frameNum: 1}
	wrong := &DecodedFrame{poc: 8, fieldPOC: [2]int32{8, 10}, frameNum: 4}
	col := &DecodedFrame{
		poc:      10,
		fieldPOC: [2]int32{10, 12},
		frameNum: 3,
		mbaff:    true,
		tables:   colMBs,
		refEntries: [2][]simpleRefEntry{
			{
				{frame: past, picID: past.frameNum, pictureStructure: PictureFrame, poc: past.poc},
				{frame: target, picID: target.frameNum, pictureStructure: PictureFrame, poc: target.poc},
			},
		},
		fieldRefEntries: [2][2][]simpleRefEntry{
			{
				{
					{frame: past, picID: 2*past.frameNum + 1, pictureStructure: PictureTopField, poc: past.fieldPOC[0]},
					{frame: past, picID: 2*past.frameNum + 2, pictureStructure: PictureBottomField, poc: past.fieldPOC[1]},
					{frame: wrong, picID: 2*wrong.frameNum + 1, pictureStructure: PictureTopField, poc: wrong.fieldPOC[0]},
				},
			},
		},
	}
	colMBs.MacroblockTyp[0] = MBType8x8 | MBTypeP0L0 | MBTypeP1L0 | MBTypeInterlaced
	colMBs.MacroblockTyp[colMBs.MBStride] = MBType8x8 | MBTypeP0L0 | MBTypeP1L0 | MBTypeInterlaced
	colMBs.RefIndex[0][3] = 2
	colMBs.MotionVal[0][int(colMBs.MB2BXY[0])+3+3*colMBs.BStride] = [2]int16{11, 7}

	var cache macroblockMotionCache
	sub := [4]uint32{MBTypeDirect2, MBTypeDirect2, MBTypeDirect2, MBTypeDirect2}
	mbType := MBType8x8 | MBTypeL0L1 | MBTypeInterlaced
	err = m.predDirectMotionFrame(&cache, 0, &mbType, &sub, h264DirectMotionContext{
		RefEntries: [2][]simpleRefEntry{
			{
				{frame: past, picID: 2*past.frameNum + 1, pictureStructure: PictureTopField, poc: past.fieldPOC[0]},
				{frame: target, picID: target.frameNum, pictureStructure: PictureFrame, poc: target.poc},
				{frame: target, picID: 2*target.frameNum + 1, pictureStructure: PictureTopField, poc: target.fieldPOC[0]},
				{frame: past, picID: 2*past.frameNum + 2, pictureStructure: PictureBottomField, poc: past.fieldPOC[1]},
				{frame: wrong, picID: 2*wrong.frameNum + 1, pictureStructure: PictureTopField, poc: wrong.fieldPOC[0]},
			},
			{{frame: col, pictureStructure: PictureTopField, poc: col.fieldPOC[0]}},
		},
		CurPOC:             6,
		CurFieldPOC:        [2]int32{6, 8},
		PictureStructure:   PictureTopField,
		Direct8x8Inference: true,
	})
	if err != nil {
		t.Fatalf("field-picture MBAFF colocated direct failed: %v", err)
	}
	base := int(h264Scan8[12])
	if cache.Ref[0][base] != 2 || cache.Ref[1][base] != 0 {
		t.Fatalf("direct refs = %d/%d, want 2/0", cache.Ref[0][base], cache.Ref[1][base])
	}
	if cache.MV[0][base] != ([2]int16{7, 4}) || cache.MV[1][base] != ([2]int16{-4, -3}) {
		t.Fatalf("direct mvs = %v/%v, want FFmpeg-scaled field refs", cache.MV[0][base], cache.MV[1][base])
	}
}

func TestPredTemporalDirectBottomFieldPictureUsesMBAFFColocatedRefOffset(t *testing.T) {
	m, err := newMacroblockTables(1, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	colMBs, err := newMacroblockTables(1, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	past := &DecodedFrame{poc: 6, fieldPOC: [2]int32{6, 8}, frameNum: 1}
	wrong := &DecodedFrame{poc: 12, fieldPOC: [2]int32{12, 14}, frameNum: 4}
	col := &DecodedFrame{
		poc:      10,
		fieldPOC: [2]int32{10, 12},
		frameNum: 3,
		mbaff:    true,
		tables:   colMBs,
		refEntries: [2][]simpleRefEntry{
			{
				{frame: past, picID: past.frameNum, pictureStructure: PictureFrame, poc: past.poc},
			},
		},
	}
	colMBs.MacroblockTyp[0] = MBType8x8 | MBTypeP0L0 | MBTypeP1L0 | MBTypeInterlaced
	colMBs.MacroblockTyp[colMBs.MBStride] = MBType8x8 | MBTypeP0L0 | MBTypeP1L0 | MBTypeInterlaced
	colMBs.RefIndex[0][2] = 1
	colMBs.MotionVal[0][int(colMBs.MB2BXY[0])+3*colMBs.BStride] = [2]int16{0, 0}

	var cache macroblockMotionCache
	sub := [4]uint32{MBTypeDirect2, MBTypeDirect2, MBTypeDirect2, MBTypeDirect2}
	mbType := MBType8x8 | MBTypeL0L1 | MBTypeInterlaced
	err = m.predDirectMotionFrame(&cache, 0, &mbType, &sub, h264DirectMotionContext{
		RefEntries: [2][]simpleRefEntry{
			{
				{frame: wrong, picID: 2*wrong.frameNum + 2, pictureStructure: PictureBottomField, poc: wrong.fieldPOC[1]},
				{frame: wrong, picID: 2*wrong.frameNum + 1, pictureStructure: PictureTopField, poc: wrong.fieldPOC[0]},
				{frame: past, picID: 2*past.frameNum + 2, pictureStructure: PictureBottomField, poc: past.fieldPOC[1]},
				{frame: past, picID: 2*past.frameNum + 1, pictureStructure: PictureTopField, poc: past.fieldPOC[0]},
			},
			{{frame: col, pictureStructure: PictureBottomField, poc: col.fieldPOC[1]}},
		},
		CurPOC:             8,
		CurFieldPOC:        [2]int32{6, 8},
		PictureStructure:   PictureBottomField,
		Direct8x8Inference: true,
	})
	if err != nil {
		t.Fatalf("bottom-field MBAFF colocated direct failed: %v", err)
	}
	base := int(h264Scan8[8])
	if cache.Ref[0][base] != 3 || cache.Ref[1][base] != 0 {
		t.Fatalf("bottom-field direct refs = %d/%d, want 3/0", cache.Ref[0][base], cache.Ref[1][base])
	}
	if cache.MV[0][base] != ([2]int16{0, 0}) || cache.MV[1][base] != ([2]int16{0, 0}) {
		t.Fatalf("bottom-field direct mvs = %v/%v, want zero colocated motion", cache.MV[0][base], cache.MV[1][base])
	}
}

func TestTemporalDirectFrameMBAFFBottomFieldColmapKeepsFieldPictureRefSlot(t *testing.T) {
	past := &DecodedFrame{poc: 0, fieldPOC: [2]int32{0, 2}, frameNum: 7}
	col := &DecodedFrame{
		fieldPicture: true,
		poc:          4,
		fieldPOC:     [2]int32{4, 6},
		fieldRefEntries: [2][2][]simpleRefEntry{
			{
				nil,
				nil,
			},
			{
				{
					{frame: past, picID: 2*past.frameNum + 0, pictureStructure: PictureBottomField, poc: past.fieldPOC[1]},
					{frame: past, picID: 2*past.frameNum + 0, pictureStructure: PictureBottomField, poc: past.fieldPOC[1]},
					{frame: past, picID: 2*past.frameNum + 1, pictureStructure: PictureTopField, poc: past.fieldPOC[0]},
				},
				nil,
			},
		},
	}
	ctx := h264DirectMotionContext{
		RefEntries: [2][]simpleRefEntry{
			{{frame: past, picID: past.frameNum, pictureStructure: PictureFrame, poc: past.poc}},
			{{frame: col, pictureStructure: PictureFrame, poc: col.poc}},
		},
		PictureStructure: PictureFrame,
	}

	got, err := temporalDirectMapColToList0Field(ctx, 0, 2, true, 1)
	if err != nil {
		t.Fatalf("bottom-field field-picture colmap failed: %v", err)
	}
	if got != 1 {
		t.Fatalf("bottom-field field-picture raw ref mapped to %d, want top-field ref 1", got)
	}
}

func TestPredTemporalDirectFrameMBAFFBottomFieldUsesXoredColFieldRef(t *testing.T) {
	m, err := newMacroblockTables(1, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	colTables, err := newMacroblockTables(1, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	mbXY := m.MBStride
	past := &DecodedFrame{poc: 0, fieldPOC: [2]int32{0, 30}, frameNum: 7}
	col := &DecodedFrame{
		poc:      20,
		fieldPOC: [2]int32{10, 12},
		tables:   colTables,
		refEntries: [2][]simpleRefEntry{
			{{frame: past, picID: past.frameNum, pictureStructure: PictureFrame, poc: past.poc}},
		},
	}
	colTables.MacroblockTyp[mbXY] = MBType16x16 | MBTypeP0L0 | MBTypeP1L0 | MBTypeInterlaced
	colTables.RefIndex[0][4*mbXY] = 1
	colTables.MotionVal[0][colTables.MB2BXY[mbXY]] = [2]int16{4, 2}

	var cache macroblockMotionCache
	var sub [4]uint32
	mbType := MBTypeDirect2 | MBTypeL0L1 | MBTypeInterlaced
	err = m.predDirectMotionFrame(&cache, mbXY, &mbType, &sub, h264DirectMotionContext{
		RefEntries: [2][]simpleRefEntry{
			{{frame: past, picID: past.frameNum, pictureStructure: PictureFrame, poc: past.poc}},
			{{frame: col, pictureStructure: PictureFrame, poc: col.poc}},
		},
		CurPOC:             6,
		CurFieldPOC:        [2]int32{4, 6},
		PictureStructure:   PictureFrame,
		Direct8x8Inference: true,
	})
	if err != nil {
		t.Fatalf("frame-MBAFF bottom temporal direct failed: %v", err)
	}
	base := int(h264Scan8[0])
	if cache.Ref[0][base] != 1 || cache.Ref[1][base] != 0 {
		t.Fatalf("refs = %d/%d, want expanded list0 ref 1 and list1 ref 0", cache.Ref[0][base], cache.Ref[1][base])
	}
	if cache.MV[0][base] != ([2]int16{2, 1}) || cache.MV[1][base] != ([2]int16{-2, -1}) {
		t.Fatalf("mvs = %v/%v, want top-field ref scaled from bottom current field", cache.MV[0][base], cache.MV[1][base])
	}
}

func TestTemporalDirectFrameMBAFFFieldDistScaleUsesFieldPOCs(t *testing.T) {
	past := &DecodedFrame{poc: 10, fieldPOC: [2]int32{0, 20}}
	future := &DecodedFrame{poc: 20, fieldPOC: [2]int32{4, 40}}
	ctx := h264DirectMotionContext{
		RefEntries: [2][]simpleRefEntry{
			{{frame: past, pictureStructure: PictureFrame, poc: past.poc}},
			{{frame: future, pictureStructure: PictureFrame, poc: future.poc}},
		},
		CurPOC:           8,
		CurFieldPOC:      [2]int32{2, 30},
		PictureStructure: PictureFrame,
	}
	layout := directColocatedLayout{CurInterlaced: true, CurFieldParity: 0}
	fieldScale, err := temporalDirectDistScaleFactorForLayout(ctx, 0, layout)
	if err != nil {
		t.Fatalf("field dist scale failed: %v", err)
	}
	if fieldScale != 128 {
		t.Fatalf("field dist scale = %d, want 128 from top-field POCs", fieldScale)
	}
	frameScale, err := temporalDirectDistScaleFactor(ctx, 0)
	if err != nil {
		t.Fatalf("frame dist scale failed: %v", err)
	}
	if frameScale == fieldScale {
		t.Fatalf("frame dist scale also = %d; test no longer distinguishes field POCs", frameScale)
	}
}

func TestPredTemporalDirectFrameMBAFFFieldMacroblockUsesFieldScale(t *testing.T) {
	m, err := newMacroblockTables(1, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	colTables, err := newMacroblockTables(1, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	past := &DecodedFrame{poc: 10, fieldPOC: [2]int32{0, 20}}
	col := &DecodedFrame{
		poc:      20,
		fieldPOC: [2]int32{4, 40},
		tables:   colTables,
		refEntries: [2][]simpleRefEntry{
			{{frame: past, pictureStructure: PictureFrame, poc: past.poc}},
		},
	}
	colTables.MacroblockTyp[0] = MBType16x16 | MBTypeP0L0 | MBTypeP1L0
	colTables.MacroblockTyp[colTables.MBStride] = MBType16x16 | MBTypeP0L0 | MBTypeP1L0
	colTables.RefIndex[0][0] = 0
	colTables.MotionVal[0][colTables.MB2BXY[0]] = [2]int16{4, 4}

	var cache macroblockMotionCache
	var sub [4]uint32
	mbType := MBTypeDirect2 | MBTypeL0L1 | MBTypeInterlaced
	err = m.predDirectMotionFrame(&cache, 0, &mbType, &sub, h264DirectMotionContext{
		RefEntries: [2][]simpleRefEntry{
			{{frame: past, pictureStructure: PictureFrame, poc: past.poc}},
			{{frame: col, pictureStructure: PictureFrame, poc: col.poc}},
		},
		CurPOC:             8,
		CurFieldPOC:        [2]int32{2, 30},
		PictureStructure:   PictureFrame,
		Direct8x8Inference: true,
	})
	if err != nil {
		t.Fatalf("frame-MBAFF field temporal direct failed: %v", err)
	}
	if !is16x8(mbType) || !isDirect(mbType) {
		t.Fatalf("mbType = %#x, want interlaced mismatch direct 16x8", mbType)
	}
	base := int(h264Scan8[0])
	if cache.Ref[0][base] != 0 || cache.Ref[1][base] != 0 {
		t.Fatalf("refs = %d/%d, want 0/0", cache.Ref[0][base], cache.Ref[1][base])
	}
	if cache.MV[0][base] != ([2]int16{2, 1}) || cache.MV[1][base] != ([2]int16{-2, -1}) {
		t.Fatalf("mvs = %v/%v, want field-scaled temporal direct", cache.MV[0][base], cache.MV[1][base])
	}
}

func TestPredTemporalDirectColocatedRefMapMissingFallsBackToZero(t *testing.T) {
	m, col, idr := newTemporalDirectTestTables(t, MBType16x16|MBTypeP0L0)
	col.tables.RefIndex[0][0] = 1
	col.tables.MotionVal[0][col.tables.MB2BXY[0]] = [2]int16{4, 2}
	col.refEntries[0] = []simpleRefEntry{
		{frame: idr, picID: idr.frameNum},
		{frame: &DecodedFrame{poc: 8, frameNum: 99}, picID: 99},
	}

	var cache macroblockMotionCache
	var sub [4]uint32
	mbType := MBTypeDirect2 | MBTypeL0L1
	err := m.predDirectMotionFrame(&cache, 0, &mbType, &sub, h264DirectMotionContext{
		RefEntries: [2][]simpleRefEntry{
			{{frame: idr, picID: idr.frameNum}},
			{{frame: col}},
		},
		CurPOC:             2,
		Direct8x8Inference: true,
	})
	if err != nil {
		t.Fatalf("temporal direct missing colmap fallback failed: %v", err)
	}
	base := int(h264Scan8[0])
	if cache.Ref[0][base] != 0 || cache.Ref[1][base] != 0 {
		t.Fatalf("refs = %d/%d, want zero fallback 0/0", cache.Ref[0][base], cache.Ref[1][base])
	}
}

func TestPredTemporalDirect8x8FromColocatedShape(t *testing.T) {
	m, col, idr := newTemporalDirectTestTables(t, MBType8x8|MBTypeP0L0|MBTypeP1L0)
	bxy := int(col.tables.MB2BXY[0])
	for i8, mv := range [4][2]int16{{4, 0}, {0, 4}, {2, 2}, {6, 2}} {
		col.tables.RefIndex[0][i8] = 0
		x8 := i8 & 1
		y8 := i8 >> 1
		col.tables.MotionVal[0][bxy+x8*3+y8*3*col.tables.BStride] = mv
	}

	var cache macroblockMotionCache
	var sub [4]uint32
	mbType := MBTypeDirect2 | MBTypeL0L1
	err := m.predDirectMotionFrame(&cache, 0, &mbType, &sub, h264DirectMotionContext{
		RefEntries: [2][]simpleRefEntry{
			{{frame: idr}},
			{{frame: col}},
		},
		CurPOC:             2,
		Direct8x8Inference: true,
	})
	if err != nil {
		t.Fatalf("temporal 8x8 direct failed: %v", err)
	}
	if !is8x8(mbType) || !isDirect(mbType) {
		t.Fatalf("mbType = %#x, want direct 8x8", mbType)
	}
	wantSub := MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2
	for i8 := 0; i8 < 4; i8++ {
		if sub[i8] != wantSub {
			t.Fatalf("sub[%d] = %#x, want %#x", i8, sub[i8], wantSub)
		}
		base := int(h264Scan8[4*i8])
		if cache.Ref[0][base] != 0 || cache.Ref[1][base] != 0 {
			t.Fatalf("sub[%d] refs = %d/%d", i8, cache.Ref[0][base], cache.Ref[1][base])
		}
	}
	if cache.MV[0][h264Scan8[4]] != ([2]int16{0, 2}) || cache.MV[1][h264Scan8[4]] != ([2]int16{0, -2}) {
		t.Fatalf("sub[1] mvs = %v/%v", cache.MV[0][h264Scan8[4]], cache.MV[1][h264Scan8[4]])
	}
}

func TestPredTemporalDirectWithout8x8InferenceUsesSub4x4Motion(t *testing.T) {
	m, col, idr := newTemporalDirectTestTables(t, MBType8x8|MBTypeP0L0|MBTypeP1L0)
	for i8 := 0; i8 < 4; i8++ {
		col.tables.RefIndex[0][i8] = 0
	}
	for i4, mv := range [4][2]int16{{4, 0}, {0, 6}, {2, 4}, {8, 2}} {
		setColocatedSub4x4MV(t, col.tables, 0, 0, i4, 0, mv)
	}

	var cache macroblockMotionCache
	var sub [4]uint32
	mbType := MBTypeDirect2 | MBTypeL0L1
	err := m.predDirectMotionFrame(&cache, 0, &mbType, &sub, h264DirectMotionContext{
		RefEntries: [2][]simpleRefEntry{
			{{frame: idr}},
			{{frame: col}},
		},
		CurPOC:             2,
		Direct8x8Inference: false,
	})
	if err != nil {
		t.Fatalf("temporal direct without 8x8 inference failed: %v", err)
	}
	if !is8x8(mbType) || !isDirect(mbType) {
		t.Fatalf("mbType = %#x, want direct 8x8 carrier", mbType)
	}
	wantSub := MBType8x8 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2
	for i8 := 0; i8 < 4; i8++ {
		if sub[i8] != wantSub {
			t.Fatalf("sub[%d] = %#x, want B_SUB_4x4 %#x", i8, sub[i8], wantSub)
		}
	}
	wantL0 := [4][2]int16{{2, 0}, {0, 3}, {1, 2}, {4, 1}}
	wantL1 := [4][2]int16{{-2, 0}, {0, -3}, {-1, -2}, {-4, -1}}
	for i4 := 0; i4 < 4; i4++ {
		dst := h264Scan8[i4]
		if cache.MV[0][dst] != wantL0[i4] || cache.MV[1][dst] != wantL1[i4] {
			t.Fatalf("i4=%d mvs = %v/%v, want %v/%v", i4, cache.MV[0][dst], cache.MV[1][dst], wantL0[i4], wantL1[i4])
		}
	}
}

func TestPredSpatialDirect16x16UsesNeighborMedian(t *testing.T) {
	m, col, idr := newTemporalDirectTestTables(t, MBType16x16|MBTypeP0L0|MBTypeP1L0)
	col.tables.RefIndex[0][0] = 0
	col.tables.MotionVal[0][col.tables.MB2BXY[0]] = [2]int16{3, 0}

	var cache macroblockMotionCache
	base := int(h264Scan8[0])
	cache.Ref[0][base-1] = 0
	cache.Ref[0][base-8] = 0
	cache.Ref[0][base+4-8] = 0
	cache.MV[0][base-1] = [2]int16{3, 9}
	cache.MV[0][base-8] = [2]int16{5, 5}
	cache.MV[0][base+4-8] = [2]int16{7, 1}
	cache.Ref[1][base-1] = h264PartNotAvailable
	cache.Ref[1][base-8] = h264PartNotAvailable
	cache.Ref[1][base+4-8] = h264PartNotAvailable
	cache.Ref[1][base-1-8] = h264PartNotAvailable

	var sub [4]uint32
	mbType := MBTypeDirect2 | MBTypeL0L1
	err := m.predDirectMotionFrame(&cache, 0, &mbType, &sub, h264DirectMotionContext{
		RefEntries: [2][]simpleRefEntry{
			{{frame: idr}},
			{{frame: col}},
		},
		DirectSpatialMVPred: true,
		Direct8x8Inference:  true,
		X264Build:           165,
	})
	if err != nil {
		t.Fatalf("spatial direct failed: %v", err)
	}
	if !is16x16(mbType) || !isDirect(mbType) || mbType&MBTypeP0L1 != 0 {
		t.Fatalf("mbType = %#x, want list0-only direct 16x16", mbType)
	}
	if cache.Ref[0][base] != 0 || cache.Ref[1][base] != -1 {
		t.Fatalf("refs = %d/%d, want 0/-1", cache.Ref[0][base], cache.Ref[1][base])
	}
	if cache.MV[0][base] != ([2]int16{5, 5}) || cache.MV[1][base] != ([2]int16{}) {
		t.Fatalf("mvs = %v/%v", cache.MV[0][base], cache.MV[1][base])
	}
}

func TestPredSpatialDirectColZeroClearsZeroRefs(t *testing.T) {
	m, col, idr := newTemporalDirectTestTables(t, MBType16x16|MBTypeP0L0|MBTypeP1L0)
	col.tables.RefIndex[0][0] = 0
	col.tables.MotionVal[0][col.tables.MB2BXY[0]] = [2]int16{1, -1}

	var cache macroblockMotionCache
	base := int(h264Scan8[0])
	for list := 0; list < 2; list++ {
		cache.Ref[list][base-1] = 0
		cache.Ref[list][base-8] = 0
		cache.Ref[list][base+4-8] = 0
		cache.MV[list][base-1] = [2]int16{4, 4}
		cache.MV[list][base-8] = [2]int16{6, 6}
		cache.MV[list][base+4-8] = [2]int16{8, 8}
	}

	var sub [4]uint32
	mbType := MBTypeDirect2 | MBTypeL0L1
	err := m.predDirectMotionFrame(&cache, 0, &mbType, &sub, h264DirectMotionContext{
		RefEntries: [2][]simpleRefEntry{
			{{frame: idr}},
			{{frame: col}},
		},
		DirectSpatialMVPred: true,
		Direct8x8Inference:  true,
		X264Build:           165,
	})
	if err != nil {
		t.Fatalf("spatial col-zero direct failed: %v", err)
	}
	if cache.MV[0][base] != ([2]int16{}) || cache.MV[1][base] != ([2]int16{}) {
		t.Fatalf("col-zero mvs = %v/%v, want zero", cache.MV[0][base], cache.MV[1][base])
	}
}

func TestPredSpatialDirectFrameMBAFFFieldRefsUseFieldRefCount(t *testing.T) {
	m, col, idr := newTemporalDirectTestTables(t, MBTypeIntra4x4|MBTypeInterlaced)
	secondRef := &DecodedFrame{poc: -2}

	var cache macroblockMotionCache
	base := int(h264Scan8[0])
	for list, ref := range [2]int8{2, 1} {
		cache.Ref[list][base-1] = ref
		cache.Ref[list][base-8] = ref
		cache.Ref[list][base+4-8] = ref
		cache.MV[list][base-1] = [2]int16{int16(3 + list), int16(5 + list)}
		cache.MV[list][base-8] = [2]int16{int16(7 + list), int16(9 + list)}
		cache.MV[list][base+4-8] = [2]int16{int16(11 + list), int16(13 + list)}
	}

	sub := [4]uint32{
		MBType16x16 | MBTypeP0L0,
		MBTypeDirect2,
		MBType16x16 | MBTypeP0L0,
		MBTypeDirect2,
	}
	mbType := MBType8x8 | MBTypeL0L1 | MBTypeInterlaced
	err := m.predDirectMotionFrame(&cache, 0, &mbType, &sub, h264DirectMotionContext{
		RefEntries: [2][]simpleRefEntry{
			{{frame: idr}, {frame: secondRef}},
			{{frame: col}},
		},
		PictureStructure:    PictureFrame,
		DirectSpatialMVPred: true,
		Direct8x8Inference:  true,
		X264Build:           165,
	})
	if err != nil {
		t.Fatalf("frame-mbaff field spatial direct failed: %v", err)
	}
	for _, i8 := range []int{1, 3} {
		start := int(h264Scan8[4*i8])
		if cache.Ref[0][start] != 2 || cache.Ref[1][start] != 1 {
			t.Fatalf("direct sub[%d] refs = %d/%d, want field refs 2/1", i8, cache.Ref[0][start], cache.Ref[1][start])
		}
	}
	if sub[1] != (MBType16x16|MBTypeL0L1|MBTypeDirect2) || sub[3] != (MBType16x16|MBTypeL0L1|MBTypeDirect2) {
		t.Fatalf("direct sub types = %#x/%#x, want spatial direct 8x8 sub type", sub[1], sub[3])
	}
}

func TestPredSpatialDirectFieldCurrentOverFrameColocatedKeeps16x8Mismatch(t *testing.T) {
	m, err := newMacroblockTables(1, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	colTables, err := newMacroblockTables(1, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	idr := &DecodedFrame{poc: 0}
	col := &DecodedFrame{poc: 4, tables: colTables}
	colTables.MacroblockTyp[0] = MBTypeIntra16x16
	colTables.MacroblockTyp[colTables.MBStride] = MBType16x16 | MBTypeP0L0

	initialType := MBTypeDirect2 | MBTypeL0L1 | MBTypeInterlaced
	layout, err := m.directColocatedLayout(colTables, 0, initialType, h264DirectMotionContext{
		RefEntries:       [2][]simpleRefEntry{nil, {{frame: col, pictureStructure: PictureFrame}}},
		PictureStructure: PictureTopField,
	})
	if err != nil {
		t.Fatalf("layout failed: %v", err)
	}
	if !layout.InterlacedMismatch {
		t.Fatalf("layout mismatch = false, want field-current over frame colocated")
	}
	for i8 := 0; i8 < 4; i8++ {
		refIndex := directColocatedRefIndex(layout, i8)
		if refIndex < 0 || refIndex >= len(colTables.RefIndex[0]) {
			t.Fatalf("ref index %d out of range", refIndex)
		}
		colTables.RefIndex[0][refIndex] = 0
		mvIndex := layout.MVBase + (i8&1)*3 + (i8>>1)*directColocatedSub8x8RowStride(layout)
		if mvIndex < 0 || mvIndex >= len(colTables.MotionVal[0]) {
			t.Fatalf("mv index %d out of range", mvIndex)
		}
		colTables.MotionVal[0][mvIndex] = [2]int16{4, 4}
	}
	colZeroIndex := layout.MVBase + 0*3 + 1*directColocatedSub8x8RowStride(layout)
	colTables.MotionVal[0][colZeroIndex] = [2]int16{1, 1}

	var cache macroblockMotionCache
	base := int(h264Scan8[0])
	cache.Ref[0][base-1] = 0
	cache.Ref[0][base-8] = 0
	cache.Ref[0][base+4-8] = 0
	cache.MV[0][base-1] = [2]int16{0, 24}
	cache.MV[0][base-8] = [2]int16{0, 24}
	cache.MV[0][base+4-8] = [2]int16{0, 24}
	cache.Ref[1][base-1] = h264PartNotAvailable
	cache.Ref[1][base-8] = h264PartNotAvailable
	cache.Ref[1][base+4-8] = h264PartNotAvailable
	cache.Ref[1][base-8-1] = h264PartNotAvailable

	var sub [4]uint32
	mbType := initialType
	err = m.predDirectMotionFrame(&cache, 0, &mbType, &sub, h264DirectMotionContext{
		RefEntries: [2][]simpleRefEntry{
			{{frame: idr, pictureStructure: PictureTopField, poc: 0}},
			{{frame: col, pictureStructure: PictureFrame, poc: 4}},
		},
		PictureStructure:    PictureTopField,
		DirectSpatialMVPred: true,
		Direct8x8Inference:  true,
		X264Build:           165,
	})
	if err != nil {
		t.Fatalf("spatial direct mismatch failed: %v", err)
	}
	if !is16x8(mbType) || is16x16(mbType) || !isDirect(mbType) {
		t.Fatalf("mbType = %#x, want direct 16x8 retained from mismatch branch", mbType)
	}
	if cache.MV[0][h264Scan8[8]] != ([2]int16{}) {
		t.Fatalf("col-zero half mv = %v, want zero", cache.MV[0][h264Scan8[8]])
	}
	if cache.MV[0][h264Scan8[12]] != ([2]int16{0, 24}) {
		t.Fatalf("nonzero half mv = %v, want neighbor mv", cache.MV[0][h264Scan8[12]])
	}
}

func TestSpatialDirectColZeroUsesFFmpegUnsignedX264BuildCompare(t *testing.T) {
	_, col, _ := newTemporalDirectTestTables(t, MBType16x16|MBTypeP0L0|MBTypeP1L0)
	col.tables.RefIndex[0][0] = -1
	col.tables.RefIndex[1][0] = 0

	layout := directColocatedLayout{
		MBXY:      0,
		MBTypeCol: [2]uint32{col.tables.MacroblockTyp[0], col.tables.MacroblockTyp[0]},
		B8Stride:  2,
		B4Stride:  col.tables.BStride,
		RefBase:   0,
		MVBase:    int(col.tables.MB2BXY[0]),
	}
	list, ok := spatialDirectColZeroList(col.tables, layout, 0, h264DirectMotionContext{
		RefEntries: [2][]simpleRefEntry{
			nil,
			{{frame: col}},
		},
		X264Build: -1,
	})
	if !ok || list != 1 {
		t.Fatalf("col-zero list = %d/%v, want 1/true", list, ok)
	}
}

func TestPredSpatialDirect8x8SameMotionCollapsesTo16x16(t *testing.T) {
	m, col, idr := newTemporalDirectTestTables(t, MBType8x8|MBTypeP0L0|MBTypeP1L0)
	bxy := int(col.tables.MB2BXY[0])
	for i8 := 0; i8 < 4; i8++ {
		col.tables.RefIndex[0][i8] = 0
		x8 := i8 & 1
		y8 := i8 >> 1
		col.tables.MotionVal[0][bxy+x8*3+y8*3*col.tables.BStride] = [2]int16{4, 0}
	}

	var cache macroblockMotionCache
	base := int(h264Scan8[0])
	for list := 0; list < 2; list++ {
		cache.Ref[list][base-1] = 0
		cache.Ref[list][base-8] = 0
		cache.Ref[list][base+4-8] = 0
		cache.MV[list][base-1] = [2]int16{2, -2}
		cache.MV[list][base-8] = [2]int16{6, 2}
		cache.MV[list][base+4-8] = [2]int16{4, 0}
	}

	var sub [4]uint32
	mbType := MBTypeDirect2 | MBTypeL0L1
	err := m.predDirectMotionFrame(&cache, 0, &mbType, &sub, h264DirectMotionContext{
		RefEntries: [2][]simpleRefEntry{
			{{frame: idr}},
			{{frame: col}},
		},
		DirectSpatialMVPred: true,
		Direct8x8Inference:  true,
		X264Build:           165,
	})
	if err != nil {
		t.Fatalf("spatial 8x8 direct failed: %v", err)
	}
	wantType := MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2
	if mbType != wantType {
		t.Fatalf("mbType = %#x, want collapsed %#x", mbType, wantType)
	}
	if cache.MV[0][base] != ([2]int16{4, 0}) || cache.MV[1][base] != ([2]int16{4, 0}) {
		t.Fatalf("collapsed mvs = %v/%v", cache.MV[0][base], cache.MV[1][base])
	}
}

func TestPredSpatialDirectWithout8x8InferenceKeepsPartialSub4x4ColZero(t *testing.T) {
	m, col, idr := newTemporalDirectTestTables(t, MBType8x8|MBTypeP0L0|MBTypeP1L0)
	for i8 := 0; i8 < 4; i8++ {
		col.tables.RefIndex[0][i8] = 0
	}
	for i4, mv := range [4][2]int16{{0, 0}, {1, -1}, {4, 0}, {0, 4}} {
		setColocatedSub4x4MV(t, col.tables, 0, 0, i4, 0, mv)
	}

	var cache macroblockMotionCache
	base := int(h264Scan8[0])
	for list := 0; list < 2; list++ {
		cache.Ref[list][base-1] = 0
		cache.Ref[list][base-8] = 0
		cache.Ref[list][base+4-8] = 0
		cache.MV[list][base-1] = [2]int16{4, 4}
		cache.MV[list][base-8] = [2]int16{4, 4}
		cache.MV[list][base+4-8] = [2]int16{4, 4}
	}

	var sub [4]uint32
	mbType := MBTypeDirect2 | MBTypeL0L1
	err := m.predDirectMotionFrame(&cache, 0, &mbType, &sub, h264DirectMotionContext{
		RefEntries: [2][]simpleRefEntry{
			{{frame: idr}},
			{{frame: col}},
		},
		DirectSpatialMVPred: true,
		Direct8x8Inference:  false,
		X264Build:           165,
	})
	if err != nil {
		t.Fatalf("spatial direct without 8x8 inference failed: %v", err)
	}
	wantSub := MBType8x8 | MBTypeL0L1 | MBTypeDirect2
	if sub[0] != wantSub {
		t.Fatalf("sub[0] = %#x, want partial B_SUB_4x4 %#x", sub[0], wantSub)
	}
	if !is8x8(mbType) || !isDirect(mbType) {
		t.Fatalf("mbType = %#x, want direct 8x8 carrier", mbType)
	}
	for _, i4 := range []int{0, 1} {
		dst := h264Scan8[i4]
		if cache.MV[0][dst] != ([2]int16{}) || cache.MV[1][dst] != ([2]int16{}) {
			t.Fatalf("i4=%d col-zero mvs = %v/%v, want zero", i4, cache.MV[0][dst], cache.MV[1][dst])
		}
	}
	for _, i4 := range []int{2, 3} {
		dst := h264Scan8[i4]
		if cache.MV[0][dst] != ([2]int16{4, 4}) || cache.MV[1][dst] != ([2]int16{4, 4}) {
			t.Fatalf("i4=%d nonzero mvs = %v/%v, want neighbor mv", i4, cache.MV[0][dst], cache.MV[1][dst])
		}
	}
}

func TestWriteBackCAVLCFrameBSkipTemporalDirect(t *testing.T) {
	m, col, idr := newTemporalDirectTestTables(t, MBType16x16|MBTypeP0L0|MBTypeP1L0)
	col.tables.RefIndex[0][0] = 0
	col.tables.MotionVal[0][col.tables.MB2BXY[0]] = [2]int16{4, 2}
	var work frameMacroblockDecodeWork
	sh := &SliceHeader{SliceTypeNoS: PictureTypeB}
	got, err := m.writeBackCAVLCFrameSkipMacroblockWithDirectWork(sh, 22, 0, 7, h264DirectMotionContext{
		RefEntries: [2][]simpleRefEntry{
			{{frame: idr}},
			{{frame: col}},
		},
		CurPOC:             2,
		Direct8x8Inference: true,
	}, &work)
	if err != nil {
		t.Fatalf("write back B-skip failed: %v", err)
	}
	if !got.Skipped || !got.IsInter || !isDirect(got.MBType) || !isSkip(got.MBType) {
		t.Fatalf("result = %+v", got)
	}
	if m.SliceTable[0] != 7 || m.QScaleTable[0] != 22 || !isSkip(m.MacroblockTyp[0]) {
		t.Fatalf("tables slice/q/type = %d/%d/%#x", m.SliceTable[0], m.QScaleTable[0], m.MacroblockTyp[0])
	}
	if m.MotionVal[0][0] != ([2]int16{2, 1}) || m.MotionVal[1][0] != ([2]int16{-2, -1}) {
		t.Fatalf("written mvs = %v/%v", m.MotionVal[0][0], m.MotionVal[1][0])
	}
	if m.RefIndex[0][0] != 0 || m.RefIndex[1][0] != 0 {
		t.Fatalf("written refs = %d/%d", m.RefIndex[0][0], m.RefIndex[1][0])
	}
}

func TestWriteBackCAVLCFrameBSkipSpatialDirectNoNeighbors(t *testing.T) {
	m, col, idr := newTemporalDirectTestTables(t, MBType16x16|MBTypeP0L0|MBTypeP1L0)
	var work frameMacroblockDecodeWork
	sh := &SliceHeader{SliceTypeNoS: PictureTypeB}
	got, err := m.writeBackCAVLCFrameSkipMacroblockWithDirectWork(sh, 22, 0, 7, h264DirectMotionContext{
		RefEntries: [2][]simpleRefEntry{
			{{frame: idr}},
			{{frame: col}},
		},
		DirectSpatialMVPred: true,
		Direct8x8Inference:  true,
		X264Build:           165,
	}, &work)
	if err != nil {
		t.Fatalf("write back spatial B-skip failed: %v", err)
	}
	wantType := MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2 | MBTypeSkip
	if got.MBType != wantType || m.MacroblockTyp[0] != wantType {
		t.Fatalf("spatial bskip type = %#x/%#x, want %#x", got.MBType, m.MacroblockTyp[0], wantType)
	}
	if m.MotionVal[0][0] != ([2]int16{}) || m.MotionVal[1][0] != ([2]int16{}) || m.RefIndex[0][0] != 0 || m.RefIndex[1][0] != 0 {
		t.Fatalf("spatial bskip motion refs/mvs = %d/%d %v/%v", m.RefIndex[0][0], m.RefIndex[1][0], m.MotionVal[0][0], m.MotionVal[1][0])
	}
}

func newTemporalDirectTestTables(t *testing.T, colType uint32) (*macroblockTables, *DecodedFrame, *DecodedFrame) {
	t.Helper()
	m, err := newMacroblockTables(1, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	colTables, err := newMacroblockTables(1, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	idr := &DecodedFrame{poc: 0}
	col := &DecodedFrame{
		poc:    4,
		tables: colTables,
		refEntries: [2][]simpleRefEntry{
			{{frame: idr}},
		},
	}
	colTables.MacroblockTyp[0] = colType
	return m, col, idr
}

func setColocatedSub4x4MV(t *testing.T, m *macroblockTables, mbXY int, i8 int, i4 int, list int, mv [2]int16) {
	t.Helper()
	if m == nil || i8 < 0 || i8 > 3 || i4 < 0 || i4 > 3 || list < 0 || list > 1 {
		t.Fatal("invalid colocated sub4x4 test input")
	}
	x8 := i8 & 1
	y8 := i8 >> 1
	index := int(m.MB2BXY[mbXY]) + x8*2 + (i4 & 1) + (y8*2+(i4>>1))*m.BStride
	if index < 0 || index >= len(m.MotionVal[list]) {
		t.Fatalf("motion index %d out of range", index)
	}
	m.MotionVal[list][index] = mv
}
