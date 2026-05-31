// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

import "testing"

func TestWriteBackCAVLCIntraMacroblockState(t *testing.T) {
	m, err := newMacroblockTables(3, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if m.SliceTable[0] != ^uint16(0) {
		t.Fatalf("initial slice table = %#x, want %#x", m.SliceTable[0], ^uint16(0))
	}

	var ctx cavlcResidualContext
	for i := range ctx.NonZeroCountCache {
		ctx.NonZeroCountCache[i] = uint8(i)
	}
	mb := cavlcMacroblockSyntax{
		MBType:         MBTypeIntra4x4,
		CBPTable:       0x234,
		QScale:         21,
		ChromaPredMode: 3,
	}
	for i := 0; i < 16; i++ {
		mb.Intra4x4PredMode[i] = int8(i % 9)
	}

	if err := m.writeBackCAVLCIntraMacroblock(1, &mb, &ctx, 4); err != nil {
		t.Fatal(err)
	}
	if m.CBPTable[1] != 0x234 || m.MacroblockTyp[1] != MBTypeIntra4x4 || m.QScaleTable[1] != 21 || m.SliceTable[1] != 4 || m.ChromaPred[1] != 3 {
		t.Fatalf("state cbp/type/q/slice/chroma = %#x/%#x/%d/%d/%d", m.CBPTable[1], m.MacroblockTyp[1], m.QScaleTable[1], m.SliceTable[1], m.ChromaPred[1])
	}
	dst := int(m.MB2BRXY[1])
	wantModes := []int8{
		mb.Intra4x4PredMode[10],
		mb.Intra4x4PredMode[11],
		mb.Intra4x4PredMode[14],
		mb.Intra4x4PredMode[15],
		mb.Intra4x4PredMode[13],
		mb.Intra4x4PredMode[7],
		mb.Intra4x4PredMode[5],
	}
	for i, want := range wantModes {
		if m.Intra4x4Pred[dst+i] != want {
			t.Fatalf("intra writeback mode[%d] = %d, want %d", i, m.Intra4x4Pred[dst+i], want)
		}
	}
	if m.NonZeroCount[1][0] != ctx.NonZeroCountCache[4+8*1] || m.NonZeroCount[1][36] != ctx.NonZeroCountCache[4+8*12] {
		t.Fatalf("nnz writeback spots = %d/%d", m.NonZeroCount[1][0], m.NonZeroCount[1][36])
	}
}

func TestWriteBackCAVLCInterMacroblockState(t *testing.T) {
	m, err := newMacroblockTables(3, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	mbXY := 5
	base := int(h264Scan8[0])
	var motion macroblockMotionCache
	for _, idx := range []int{base - 1, base - 8, base - 8 + 4} {
		motion.Ref[0][idx] = 0
	}
	motion.MV[0][base-1] = [2]int16{1, 11}
	motion.MV[0][base-8] = [2]int16{3, 33}
	motion.MV[0][base-8+4] = [2]int16{2, 22}

	var ctx cavlcResidualContext
	for i := range ctx.NonZeroCountCache {
		ctx.NonZeroCountCache[i] = uint8(100 + i)
	}
	mb := cavlcInterMacroblockSyntax{}
	mb.MBType = MBType16x16 | MBTypeP0L0
	mb.CBPTable = 0x345
	mb.QScale = 19
	mb.Ref[0][0] = 0
	mb.MVD[0][0] = [2]int32{5, 6}

	if err := m.writeBackCAVLCInterMacroblock(mbXY, &mb, &ctx, &motion, 1, PictureTypeP, 9); err != nil {
		t.Fatal(err)
	}
	if m.CBPTable[mbXY] != 0x345 || m.MacroblockTyp[mbXY] != mb.MBType || m.QScaleTable[mbXY] != 19 || m.SliceTable[mbXY] != 9 || m.ChromaPred[mbXY] != 0 {
		t.Fatalf("inter state = %#x/%#x/%d/%d/%d", m.CBPTable[mbXY], m.MacroblockTyp[mbXY], m.QScaleTable[mbXY], m.SliceTable[mbXY], m.ChromaPred[mbXY])
	}
	bXY := int(m.MB2BXY[mbXY])
	if m.MotionVal[0][bXY] != ([2]int16{7, 28}) || m.MotionVal[0][bXY+3+3*m.BStride] != ([2]int16{7, 28}) {
		t.Fatalf("inter motion writeback = %v/%v", m.MotionVal[0][bXY], m.MotionVal[0][bXY+3+3*m.BStride])
	}
	if got := m.RefIndex[0][4*mbXY : 4*mbXY+4]; got[0] != 0 || got[1] != 0 || got[2] != 0 || got[3] != 0 {
		t.Fatalf("inter refs = %v", got)
	}
	if m.NonZeroCount[mbXY][0] != ctx.NonZeroCountCache[4+8*1] || m.NonZeroCount[mbXY][20] != ctx.NonZeroCountCache[4+8*7] {
		t.Fatalf("inter nnz spots = %d/%d", m.NonZeroCount[mbXY][0], m.NonZeroCount[mbXY][20])
	}
}

func TestWriteBackCABACIntraMacroblockState(t *testing.T) {
	m, err := newMacroblockTables(3, 1, 1)
	if err != nil {
		t.Fatal(err)
	}

	var ctx cavlcResidualContext
	for i := range ctx.NonZeroCountCache {
		ctx.NonZeroCountCache[i] = uint8(50 + i)
	}
	var intraCache [h264IntraPredModeCacheSize]int8
	copy(intraCache[4+8*4:4+8*4+4], []int8{1, 2, 3, 4})
	intraCache[7+8*3] = 5
	intraCache[7+8*2] = 6
	intraCache[7+8*1] = 7

	mb := cavlcMacroblockSyntax{
		MBType:         MBTypeIntra4x4 | MBType8x8DCT,
		CBPTable:       0x456,
		QScale:         22,
		ChromaPredMode: 2,
	}
	if err := m.writeBackCABACIntraMacroblock(1, &mb, &ctx, &intraCache, 12); err != nil {
		t.Fatal(err)
	}
	if m.CBPTable[1] != 0x456 || m.MacroblockTyp[1] != mb.MBType || m.QScaleTable[1] != 22 || m.SliceTable[1] != 12 || m.ChromaPred[1] != 2 {
		t.Fatalf("cabac intra state = %#x/%#x/%d/%d/%d", m.CBPTable[1], m.MacroblockTyp[1], m.QScaleTable[1], m.SliceTable[1], m.ChromaPred[1])
	}
	dst := int(m.MB2BRXY[1])
	wantModes := []int8{1, 2, 3, 4, 5, 6, 7}
	for i, want := range wantModes {
		if m.Intra4x4Pred[dst+i] != want {
			t.Fatalf("cabac intra pred[%d] = %d, want %d", i, m.Intra4x4Pred[dst+i], want)
		}
	}
	if m.NonZeroCount[1][0] != ctx.NonZeroCountCache[4+8*1] || m.NonZeroCount[1][36] != ctx.NonZeroCountCache[4+8*12] {
		t.Fatalf("cabac intra nnz spots = %d/%d", m.NonZeroCount[1][0], m.NonZeroCount[1][36])
	}
}

func TestWriteBackCABACInterMacroblockStateWritesMVD(t *testing.T) {
	m, err := newMacroblockTables(3, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	mbXY := 5
	base := int(h264Scan8[0])
	var motion macroblockMotionCache
	for _, idx := range []int{base - 1, base - 8, base - 8 + 4} {
		motion.Ref[0][idx] = 0
	}
	motion.MV[0][base-1] = [2]int16{1, 11}
	motion.MV[0][base-8] = [2]int16{3, 33}
	motion.MV[0][base-8+4] = [2]int16{2, 22}

	var ctx cavlcResidualContext
	for i := range ctx.NonZeroCountCache {
		ctx.NonZeroCountCache[i] = uint8(10 + i)
	}
	mb := cavlcInterMacroblockSyntax{}
	mb.MBType = MBType16x16 | MBTypeP0L0
	mb.CBPTable = 0x567
	mb.QScale = 18
	mb.Ref[0][0] = 0
	mb.MVD[0][0] = [2]int32{5, -6}

	if err := m.writeBackCABACInterMacroblock(mbXY, &mb, &ctx, &motion, 1, PictureTypeP, 13); err != nil {
		t.Fatal(err)
	}
	if m.CBPTable[mbXY] != 0x567 || m.MacroblockTyp[mbXY] != mb.MBType || m.QScaleTable[mbXY] != 18 || m.SliceTable[mbXY] != 13 {
		t.Fatalf("cabac inter state = %#x/%#x/%d/%d", m.CBPTable[mbXY], m.MacroblockTyp[mbXY], m.QScaleTable[mbXY], m.SliceTable[mbXY])
	}
	bXY := int(m.MB2BXY[mbXY])
	if m.MotionVal[0][bXY] != ([2]int16{7, 16}) || m.MotionVal[0][bXY+3+3*m.BStride] != ([2]int16{7, 16}) {
		t.Fatalf("cabac inter motion writeback = %v/%v", m.MotionVal[0][bXY], m.MotionVal[0][bXY+3+3*m.BStride])
	}
	dst := int(m.MB2BRXY[mbXY])
	for i := 0; i < 7; i++ {
		if m.MVDTable[0][dst+i] != ([2]uint8{5, 6}) {
			t.Fatalf("cabac mvd table[%d] = %v, want [5 6]", i, m.MVDTable[0][dst+i])
		}
	}
	if m.NonZeroCount[mbXY][0] != ctx.NonZeroCountCache[4+8*1] || m.NonZeroCount[mbXY][20] != ctx.NonZeroCountCache[4+8*7] {
		t.Fatalf("cabac inter nnz spots = %d/%d", m.NonZeroCount[mbXY][0], m.NonZeroCount[mbXY][20])
	}
}

func TestWriteBackPskipMacroblockState(t *testing.T) {
	m, err := newMacroblockTables(3, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	mbXY := 5
	leftXY := 4
	topXY := 1
	topRightXY := 2
	m.RefIndex[0][4*leftXY+1] = 0
	m.MotionVal[0][int(m.MB2BXY[leftXY])+3] = [2]int16{10, 20}
	m.RefIndex[0][4*topXY+2] = 0
	m.MotionVal[0][int(m.MB2BXY[topXY])+3*m.BStride] = [2]int16{30, 40}
	m.RefIndex[0][4*topRightXY+2] = 0
	m.MotionVal[0][int(m.MB2BXY[topRightXY])+3*m.BStride] = [2]int16{20, 10}
	for i := range m.NonZeroCount[mbXY] {
		m.NonZeroCount[mbXY][i] = 9
	}

	err = m.writeBackPskipMacroblock(mbXY, 24, motionDecodeNeighbors{
		LeftType:     [2]uint32{MBType16x16 | MBTypeP0L0, 0},
		TopType:      MBType16x16 | MBTypeP0L0,
		TopRightType: MBType16x16 | MBTypeP0L0,
		LeftXY:       [2]int{leftXY, 0},
		TopXY:        topXY,
		TopRightXY:   topRightXY,
	}, 11)
	if err != nil {
		t.Fatal(err)
	}
	wantType := MBType16x16 | MBTypeP0L0 | MBTypeP1L0 | MBTypeSkip
	if m.MacroblockTyp[mbXY] != wantType || m.CBPTable[mbXY] != 0 || m.QScaleTable[mbXY] != 24 || m.SliceTable[mbXY] != 11 {
		t.Fatalf("pskip state = %#x/%#x/%d/%d", m.MacroblockTyp[mbXY], m.CBPTable[mbXY], m.QScaleTable[mbXY], m.SliceTable[mbXY])
	}
	bXY := int(m.MB2BXY[mbXY])
	if m.MotionVal[0][bXY] != ([2]int16{20, 20}) || m.MotionVal[0][bXY+3+3*m.BStride] != ([2]int16{20, 20}) {
		t.Fatalf("pskip motion = %v/%v", m.MotionVal[0][bXY], m.MotionVal[0][bXY+3+3*m.BStride])
	}
	if got := m.RefIndex[0][4*mbXY : 4*mbXY+4]; got[0] != 0 || got[1] != 0 || got[2] != 0 || got[3] != 0 {
		t.Fatalf("pskip refs = %v", got)
	}
	for i, v := range m.NonZeroCount[mbXY] {
		if v != 0 {
			t.Fatalf("pskip nnz[%d] = %d, want 0", i, v)
		}
	}
}

func TestWriteBackCABACPskipMacroblockZerosMVDTable(t *testing.T) {
	m, err := newMacroblockTables(3, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	mbXY := 5
	leftXY := 4
	topXY := 1
	topRightXY := 2
	m.RefIndex[0][4*leftXY+1] = 0
	m.MotionVal[0][int(m.MB2BXY[leftXY])+3] = [2]int16{10, 20}
	m.RefIndex[0][4*topXY+2] = 0
	m.MotionVal[0][int(m.MB2BXY[topXY])+3*m.BStride] = [2]int16{30, 40}
	m.RefIndex[0][4*topRightXY+2] = 0
	m.MotionVal[0][int(m.MB2BXY[topRightXY])+3*m.BStride] = [2]int16{20, 10}

	dst := int(m.MB2BRXY[mbXY])
	for i := 0; i < 8; i++ {
		m.MVDTable[0][dst+i] = [2]uint8{9, 9}
	}
	err = m.writeBackCABACPskipMacroblock(mbXY, 24, motionDecodeNeighbors{
		LeftType:     [2]uint32{MBType16x16 | MBTypeP0L0, 0},
		TopType:      MBType16x16 | MBTypeP0L0,
		TopRightType: MBType16x16 | MBTypeP0L0,
		LeftXY:       [2]int{leftXY, 0},
		TopXY:        topXY,
		TopRightXY:   topRightXY,
	}, 14)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 8; i++ {
		if m.MVDTable[0][dst+i] != ([2]uint8{}) {
			t.Fatalf("cabac pskip mvd[%d] = %v, want zero", i, m.MVDTable[0][dst+i])
		}
	}
	if m.SliceTable[mbXY] != 14 || m.QScaleTable[mbXY] != 24 {
		t.Fatalf("cabac pskip state slice/q = %d/%d", m.SliceTable[mbXY], m.QScaleTable[mbXY])
	}
}

func TestWriteBackCAVLCMacroblockStateRejectsInvalidQScale(t *testing.T) {
	m, err := newMacroblockTables(1, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	mb := cavlcMacroblockSyntax{MBType: MBTypeIntra16x16, QScale: qpMaxNum + 1}
	var ctx cavlcResidualContext
	if err := m.writeBackCAVLCIntraMacroblock(0, &mb, &ctx, 0); err != ErrInvalidData {
		t.Fatalf("err = %v, want ErrInvalidData", err)
	}
}
