// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

import (
	"errors"
	"strings"
	"testing"
)

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

func TestPredDirectMotionReportsUnsupportedInterlacedColocatedFrame(t *testing.T) {
	m, col, idr := newTemporalDirectTestTables(t, MBType16x16|MBTypeP0L0|MBTypeP1L0|MBTypeInterlaced)

	var cache macroblockMotionCache
	var sub [4]uint32
	mbType := MBTypeDirect2 | MBTypeL0L1
	err := m.predDirectMotionFrame(&cache, 0, &mbType, &sub, h264DirectMotionContext{
		RefEntries: [2][]simpleRefEntry{
			{{frame: idr}},
			{{frame: col, pictureStructure: PictureFrame}},
		},
		PictureStructure:   PictureFrame,
		Direct8x8Inference: true,
	})
	if !errors.Is(err, ErrUnsupported) {
		t.Fatalf("err = %v, want ErrUnsupported", err)
	}
	for _, want := range []string{"direct motion interlaced colocated", "mb_xy=0", "picture=3", "ref1_picture=3"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("err = %q, want detail %q", err, want)
		}
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

func TestSpatialDirectColZeroUsesFFmpegUnsignedX264BuildCompare(t *testing.T) {
	_, col, _ := newTemporalDirectTestTables(t, MBType16x16|MBTypeP0L0|MBTypeP1L0)
	col.tables.RefIndex[0][0] = -1
	col.tables.RefIndex[1][0] = 0

	list, ok := spatialDirectColZeroList(col.tables, 0, 0, h264DirectMotionContext{
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
