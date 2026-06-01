// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

import "testing"

func TestSliceMacroblockCursorFrameMappingAndAdvance(t *testing.T) {
	m, err := newMacroblockTables(3, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	sps := &SPS{ChromaFormatIDC: 1, FrameMBSOnlyFlag: 1}
	pps := &PPS{SPS: sps}
	sh := &SliceHeader{
		FirstMBAddr:      4,
		PictureStructure: PictureFrame,
		SPS:              sps,
		PPS:              pps,
	}
	cur, err := newSliceMacroblockCursor(m, sh)
	if err != nil {
		t.Fatal(err)
	}
	if cur.MBX != 1 || cur.MBY != 1 || cur.MBXY != 5 {
		t.Fatalf("cursor = x%d y%d xy%d, want x1 y1 xy5", cur.MBX, cur.MBY, cur.MBXY)
	}
	if !cur.advanceFrameMB() || cur.MBX != 2 || cur.MBY != 1 || cur.MBXY != 6 {
		t.Fatalf("advance once = x%d y%d xy%d", cur.MBX, cur.MBY, cur.MBXY)
	}
	if cur.advanceFrameMB() {
		t.Fatalf("advance past final MB reported more work: x%d y%d xy%d", cur.MBX, cur.MBY, cur.MBXY)
	}

	sh.PictureStructure = PictureTopField
	if _, err := newSliceMacroblockCursor(m, sh); err != ErrUnsupported {
		t.Fatalf("field cursor err = %v, want ErrUnsupported", err)
	}
}

func TestSliceMacroblockCursorFrameMBAFFMappingAndAdvance(t *testing.T) {
	m, err := newMacroblockTables(3, 4, 1)
	if err != nil {
		t.Fatal(err)
	}
	sps := &SPS{ChromaFormatIDC: 1, FrameMBSOnlyFlag: 0, MBAFF: 1}
	pps := &PPS{SPS: sps}
	sh := &SliceHeader{
		FirstMBAddr:      4,
		PictureStructure: PictureFrame,
		SPS:              sps,
		PPS:              pps,
	}
	cur, err := newSliceMacroblockCursor(m, sh)
	if err != nil {
		t.Fatal(err)
	}
	if !cur.FieldOrMBAFF || cur.MBX != 1 || cur.MBY != 2 || cur.MBXY != 9 {
		t.Fatalf("MBAFF cursor = fieldOrMBAFF %v x%d y%d xy%d, want true x1 y2 xy9", cur.FieldOrMBAFF, cur.MBX, cur.MBY, cur.MBXY)
	}
	if !cur.advanceFrameMB() || cur.MBX != 2 || cur.MBY != 2 || cur.MBXY != 10 {
		t.Fatalf("MBAFF advance once = x%d y%d xy%d", cur.MBX, cur.MBY, cur.MBXY)
	}
	if cur.advanceFrameMB() {
		t.Fatalf("MBAFF advance past final pair row reported more work: x%d y%d xy%d", cur.MBX, cur.MBY, cur.MBXY)
	}

	sh.FirstMBAddr = 6
	if _, err := newSliceMacroblockCursor(m, sh); err != ErrInvalidData {
		t.Fatalf("MBAFF first_mb overflow err = %v, want ErrInvalidData", err)
	}
}

func TestFillDecodeNeighborsFrameSliceBoundaries(t *testing.T) {
	m, err := newMacroblockTables(3, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	sliceNum := uint16(7)
	mbXY := 5
	topLeftXY := 0
	topXY := 1
	topRightXY := 2
	leftXY := 4

	m.MacroblockTyp[topLeftXY] = MBTypeIntra4x4
	m.MacroblockTyp[topXY] = MBType16x16 | MBTypeP0L0
	m.MacroblockTyp[topRightXY] = MBType16x16 | MBTypeP0L0
	m.MacroblockTyp[leftXY] = MBType16x16 | MBTypeP0L0
	m.SliceTable[topXY] = sliceNum

	n, err := m.fillDecodeNeighborsFrame(mbXY, sliceNum, MBType16x16|MBTypeP0L0)
	if err != nil {
		t.Fatal(err)
	}
	if n.TopLeftType != 0 || n.TopType != m.MacroblockTyp[topXY] || n.LeftType[0] != 0 || n.LeftType[1] != 0 || n.TopRightType != 0 {
		t.Fatalf("slice-boundary neighbors topLeft/top/left/topRight = %#x/%#x/%#x,%#x/%#x",
			n.TopLeftType, n.TopType, n.LeftType[0], n.LeftType[1], n.TopRightType)
	}
	if n.TopXY != topXY || n.TopLeftXY != topLeftXY || n.TopRightXY != topRightXY || n.LeftXY != ([2]int{leftXY, leftXY}) || n.TopLeftPartition != -1 {
		t.Fatalf("neighbor positions = %+v", n)
	}
}

func TestFillFrameMacroblockDecodeCachesComposesResidualAndMotion(t *testing.T) {
	m, err := newMacroblockTables(3, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	sliceNum := uint16(3)
	mbXY := 5
	topLeftXY := 0
	topXY := 1
	topRightXY := 2
	leftXY := 4
	for _, xy := range []int{topLeftXY, topXY, topRightXY, leftXY} {
		m.SliceTable[xy] = sliceNum
	}
	m.MacroblockTyp[topLeftXY] = MBType16x16 | MBTypeP0L0
	m.MacroblockTyp[topXY] = MBType16x16 | MBTypeP0L0
	m.MacroblockTyp[topRightXY] = MBType16x16 | MBTypeP0L0
	m.MacroblockTyp[leftXY] = MBType16x16 | MBTypeP0L0
	m.CBPTable[topXY] = 0x123
	m.CBPTable[leftXY] = 0x456
	for i := range m.NonZeroCount[topXY] {
		m.NonZeroCount[topXY][i] = uint8(10 + i)
		m.NonZeroCount[leftXY][i] = uint8(80 + i)
	}
	for j := 0; j < 4; j++ {
		m.MotionVal[0][int(m.MB2BXY[topXY])+3*m.BStride+j] = [2]int16{int16(100 + j), int16(200 + j)}
		m.MVDTable[0][int(m.MB2BRXY[topXY])+j] = [2]uint8{uint8(20 + j), uint8(30 + j)}
	}
	m.RefIndex[0][4*topXY+2] = 1
	m.RefIndex[0][4*topXY+3] = 2
	leftBXY := int(m.MB2BXY[leftXY]) + 3
	for i, block := range []int{0, 1, 2, 3} {
		m.MotionVal[0][leftBXY+m.BStride*block] = [2]int16{int16(40 + i), int16(50 + i)}
	}
	m.RefIndex[0][4*leftXY+1] = 3
	m.RefIndex[0][4*leftXY+3] = 4

	var residual cavlcResidualContext
	var motion macroblockMotionCache
	result, err := m.fillFrameMacroblockDecodeCaches(nil, &residual, &motion, frameMacroblockDecodeCacheInput{
		MBXY:         mbXY,
		SliceNum:     sliceNum,
		MBType:       MBType16x8 | MBTypeP0L0 | MBTypeP1L0,
		ListCount:    1,
		SliceTypeNoS: PictureTypeP,
		CABAC:        true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Neighbors.TopType != m.MacroblockTyp[topXY] || result.Neighbors.LeftType[0] != m.MacroblockTyp[leftXY] {
		t.Fatalf("composed neighbor types = %#x/%#x", result.Neighbors.TopType, result.Neighbors.LeftType[0])
	}
	if result.Residual.TopCBP != 0x123 || result.Residual.LeftCBP == 0 {
		t.Fatalf("residual cbp = top %#x left %#x", result.Residual.TopCBP, result.Residual.LeftCBP)
	}
	if residual.NonZeroCountCache[4+8*0] != 22 || residual.NonZeroCountCache[3+8*1] != 83 {
		t.Fatalf("residual cache = top %d left %d", residual.NonZeroCountCache[4+8*0], residual.NonZeroCountCache[3+8*1])
	}
	base := int(h264Scan8[0])
	if motion.MV[0][base-8] != ([2]int16{100, 200}) || motion.Ref[0][base-8] != 1 {
		t.Fatalf("top motion cache = %v ref %d", motion.MV[0][base-8], motion.Ref[0][base-8])
	}
	if motion.MV[0][base-1] != ([2]int16{40, 50}) || motion.Ref[0][base-1] != 3 {
		t.Fatalf("left motion cache = %v ref %d", motion.MV[0][base-1], motion.Ref[0][base-1])
	}
}

func TestFillFrameMacroblockDecodeCachesIntraPrediction(t *testing.T) {
	m, err := newMacroblockTables(2, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	sliceNum := uint16(5)
	mbXY := 4
	topXY := 1
	leftXY := 3
	for _, xy := range []int{0, topXY, leftXY} {
		m.SliceTable[xy] = sliceNum
		m.MacroblockTyp[xy] = MBTypeIntra4x4
	}
	topBase := int(m.MB2BRXY[topXY])
	leftBase := int(m.MB2BRXY[leftXY])
	for i := 0; i < 7; i++ {
		m.Intra4x4Pred[topBase+i] = int8(10 + i)
		m.Intra4x4Pred[leftBase+i] = int8(30 + i)
	}

	var intra [h264IntraPredModeCacheSize]int8
	var residual cavlcResidualContext
	result, err := m.fillFrameMacroblockDecodeCaches(&intra, &residual, nil, frameMacroblockDecodeCacheInput{
		MBXY:     mbXY,
		SliceNum: sliceNum,
		MBType:   MBTypeIntra4x4,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Intra.TopSamplesAvailable != 0xffff || result.Intra.LeftSamplesAvailable != 0xffff {
		t.Fatalf("intra availability = %+v", result.Intra)
	}
	if intra[4+8*0] != 10 || intra[3+8*1] != 36 {
		t.Fatalf("intra cache top/left = %d/%d", intra[4+8*0], intra[3+8*1])
	}
	pred, err := predIntra4x4Modes(&intra)
	if err != nil {
		t.Fatal(err)
	}
	if pred[0] != 10 || pred[5] != intraPredVertical {
		t.Fatalf("pred modes[0,5] = %d/%d", pred[0], pred[5])
	}
}
