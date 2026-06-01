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
