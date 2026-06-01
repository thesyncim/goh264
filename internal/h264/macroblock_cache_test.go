// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

import "testing"

func TestMacroblockTablesUseFFmpegStrides(t *testing.T) {
	m, err := newMacroblockTables(3, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	if m.MBStride != 4 || m.BStride != 12 {
		t.Fatalf("strides = mb %d b %d, want 4/12", m.MBStride, m.BStride)
	}
	if len(m.NonZeroCount) != 12 || len(m.Intra4x4Pred) != 64 || len(m.MVDTable[0]) != 64 {
		t.Fatalf("table lengths nnz/intra/mvd = %d/%d/%d", len(m.NonZeroCount), len(m.Intra4x4Pred), len(m.MVDTable[0]))
	}
	if len(m.RefIndex[0]) != 32 || len(m.MotionVal[0]) != 96 || len(m.DirectTable) != 48 {
		t.Fatalf("motion lengths ref/mv/direct = %d/%d/%d", len(m.RefIndex[0]), len(m.MotionVal[0]), len(m.DirectTable))
	}

	mbXY := 2 + 1*m.MBStride
	if m.MB2BXY[mbXY] != 56 || m.MB2BRXY[mbXY] != 48 {
		t.Fatalf("mb2b/mb2br = %d/%d, want 56/48", m.MB2BXY[mbXY], m.MB2BRXY[mbXY])
	}
}

func TestWriteBackNonZeroCount420And422(t *testing.T) {
	var cache [h264NonZeroCountCacheSize]uint8
	for i := range cache {
		cache[i] = uint8(i)
	}

	m420, err := newMacroblockTables(2, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if err := m420.writeBackNonZeroCount(0, &cache); err != nil {
		t.Fatal(err)
	}
	nnz420 := m420.NonZeroCount[0]
	if nnz420[0] != cache[4+8*1] || nnz420[12] != cache[4+8*4] || nnz420[20] != cache[4+8*7] || nnz420[36] != cache[4+8*12] {
		t.Fatalf("420 luma/chroma rows not copied source-shaped: %v", nnz420)
	}
	if nnz420[24] != 0 || nnz420[28] != 0 || nnz420[40] != 0 || nnz420[44] != 0 {
		t.Fatalf("420 copied 422/444-only rows: %v", nnz420[24:48])
	}

	m422, err := newMacroblockTables(2, 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if err := m422.writeBackNonZeroCount(0, &cache); err != nil {
		t.Fatal(err)
	}
	nnz422 := m422.NonZeroCount[0]
	if nnz422[24] != cache[4+8*8] || nnz422[28] != cache[4+8*9] || nnz422[40] != cache[4+8*13] || nnz422[44] != cache[4+8*14] {
		t.Fatalf("422 extra rows = %d/%d/%d/%d", nnz422[24], nnz422[28], nnz422[40], nnz422[44])
	}
}

func TestWriteBackIntraPredMode(t *testing.T) {
	m, err := newMacroblockTables(3, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	var cache [h264IntraPredModeCacheSize]int8
	for i := range cache {
		cache[i] = int8(i)
	}

	mbXY := 1
	if err := m.writeBackIntraPredMode(mbXY, &cache); err != nil {
		t.Fatal(err)
	}
	dst := int(m.MB2BRXY[mbXY])
	want := []int8{36, 37, 38, 39, 31, 23, 15}
	for i, w := range want {
		if m.Intra4x4Pred[dst+i] != w {
			t.Fatalf("intra pred[%d] = %d, want %d", i, m.Intra4x4Pred[dst+i], w)
		}
	}
}

func TestFillIntraPredModeCaches(t *testing.T) {
	m, err := newMacroblockTables(3, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	topXY := 1
	leftXY := 4
	topBase := int(m.MB2BRXY[topXY])
	leftBase := int(m.MB2BRXY[leftXY])
	for i := 0; i < 7; i++ {
		m.Intra4x4Pred[topBase+i] = int8(10 + i)
		m.Intra4x4Pred[leftBase+i] = int8(30 + i)
	}

	var cache [h264IntraPredModeCacheSize]int8
	result, err := m.fillIntraPredModeCaches(&cache, intraPredDecodeNeighbors{
		MBType:       MBTypeIntra4x4,
		TopType:      MBTypeIntra4x4 | MBType8x8DCT,
		TopLeftType:  0,
		TopRightType: 0,
		LeftType:     [2]uint32{MBTypeIntra4x4, 0},
		TopXY:        topXY,
		LeftXY:       [2]int{leftXY, leftXY},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := cache[4+8*0 : 4+8*0+4]; got[0] != 10 || got[1] != 11 || got[2] != 12 || got[3] != 13 {
		t.Fatalf("top intra cache = %v", got)
	}
	if cache[3+8*1] != 36 || cache[3+8*2] != 35 {
		t.Fatalf("left intra cache = %d/%d, want 36/35", cache[3+8*1], cache[3+8*2])
	}
	if cache[3+8*3] != -1 || cache[3+8*4] != -1 {
		t.Fatalf("missing left-bottom defaults = %d/%d, want -1/-1", cache[3+8*3], cache[3+8*4])
	}
	if result.NeighborTransformSize != 1 || result.TopLeftSamplesAvailable != 0x7fff || result.TopRightSamplesAvailable != 0xeaea {
		t.Fatalf("availability = %+v", result)
	}
}

func TestFillResidualDecodeCaches422Neighbors(t *testing.T) {
	m, err := newMacroblockTables(3, 2, 2)
	if err != nil {
		t.Fatal(err)
	}
	topXY := 1
	leftXY := 4
	for i := 0; i < h264MBNonZeroCountSize; i++ {
		m.NonZeroCount[topXY][i] = uint8(10 + i)
		m.NonZeroCount[leftXY][i] = uint8(80 + i)
	}
	m.CBPTable[topXY] = 0x123
	m.CBPTable[leftXY] = 0x456

	var ctx cavlcResidualContext
	result, err := m.fillResidualDecodeCaches(&ctx, residualDecodeNeighbors{
		MBType:   MBType16x16 | MBTypeP0L0,
		TopType:  MBTypeIntra4x4,
		LeftType: [2]uint32{MBType16x16 | MBTypeP0L0, MBType16x16 | MBTypeP0L0},
		TopXY:    topXY,
		LeftXY:   [2]int{leftXY, leftXY},
		CABAC:    true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if ctx.NonZeroCountCache[4+8*0] != 22 || ctx.NonZeroCountCache[4+8*5] != 38 || ctx.NonZeroCountCache[4+8*10] != 54 {
		t.Fatalf("top nnz cache = %d/%d/%d", ctx.NonZeroCountCache[4+8*0], ctx.NonZeroCountCache[4+8*5], ctx.NonZeroCountCache[4+8*10])
	}
	if ctx.NonZeroCountCache[3+8*1] != 83 || ctx.NonZeroCountCache[3+8*2] != 87 {
		t.Fatalf("left luma nnz = %d/%d", ctx.NonZeroCountCache[3+8*1], ctx.NonZeroCountCache[3+8*2])
	}
	if ctx.NonZeroCountCache[3+8*6] != 97 || ctx.NonZeroCountCache[3+8*7] != 101 ||
		ctx.NonZeroCountCache[3+8*11] != 113 || ctx.NonZeroCountCache[3+8*12] != 117 {
		t.Fatalf("left 422 chroma nnz = %d/%d/%d/%d",
			ctx.NonZeroCountCache[3+8*6], ctx.NonZeroCountCache[3+8*7],
			ctx.NonZeroCountCache[3+8*11], ctx.NonZeroCountCache[3+8*12])
	}
	wantLeftCBP := (m.CBPTable[leftXY] & 0x7f0) |
		((m.CBPTable[leftXY] >> (h264LeftBlockFrame[0] &^ 1)) & 2) |
		(((m.CBPTable[leftXY] >> (h264LeftBlockFrame[2] &^ 1)) & 2) << 2)
	if result.TopCBP != 0x123 || result.LeftCBP != wantLeftCBP {
		t.Fatalf("cbp = top %#x left %#x, want %#x/%#x", result.TopCBP, result.LeftCBP, 0x123, wantLeftCBP)
	}
}

func TestFillResidualDecodeCachesCABACUnavailableDefaults(t *testing.T) {
	m, err := newMacroblockTables(1, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	var ctx cavlcResidualContext
	result, err := m.fillResidualDecodeCaches(&ctx, residualDecodeNeighbors{
		MBType: MBType16x16 | MBTypeP0L0,
		CABAC:  true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if ctx.NonZeroCountCache[4+8*0] != 0 || ctx.NonZeroCountCache[3+8*1] != 0 || result.TopCBP != 0x00f || result.LeftCBP != 0x00f {
		t.Fatalf("inter CABAC defaults cache/top/left = %d/%d/%#x/%#x", ctx.NonZeroCountCache[4+8*0], ctx.NonZeroCountCache[3+8*1], result.TopCBP, result.LeftCBP)
	}

	var intraCtx cavlcResidualContext
	result, err = m.fillResidualDecodeCaches(&intraCtx, residualDecodeNeighbors{
		MBType: MBTypeIntra4x4,
		CABAC:  true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if intraCtx.NonZeroCountCache[4+8*0] != 64 || intraCtx.NonZeroCountCache[3+8*1] != 64 || result.TopCBP != 0x7cf || result.LeftCBP != 0x7cf {
		t.Fatalf("intra CABAC defaults cache/top/left = %d/%d/%#x/%#x", intraCtx.NonZeroCountCache[4+8*0], intraCtx.NonZeroCountCache[3+8*1], result.TopCBP, result.LeftCBP)
	}
}

func TestFillMotionDecodeCachesFrameNeighbors(t *testing.T) {
	m, err := newMacroblockTables(3, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	topXY := 1
	leftXY := 4
	topLeftXY := 0

	for j := 0; j < 4; j++ {
		m.MotionVal[0][int(m.MB2BXY[topXY])+3*m.BStride+j] = [2]int16{int16(100 + j), int16(200 + j)}
		m.MVDTable[0][int(m.MB2BRXY[topXY])+j] = [2]uint8{uint8(10 + j), uint8(20 + j)}
	}
	m.RefIndex[0][4*topXY+2] = 7
	m.RefIndex[0][4*topXY+3] = 8

	leftBXY := int(m.MB2BXY[leftXY]) + 3
	leftBRXY := int(m.MB2BRXY[leftXY]) + 6
	for i, block := range []int{0, 1, 2, 3} {
		m.MotionVal[0][leftBXY+m.BStride*block] = [2]int16{int16(30 + i), int16(40 + i)}
		m.MVDTable[0][leftBRXY-block] = [2]uint8{uint8(50 + i), uint8(60 + i)}
	}
	m.RefIndex[0][4*leftXY+1] = 4
	m.RefIndex[0][4*leftXY+3] = 5

	m.MotionVal[0][int(m.MB2BXY[topLeftXY])+3+m.BStride+(-1&(2*m.BStride))] = [2]int16{77, 88}
	m.RefIndex[0][4*topLeftXY+1+(-1&2)] = 6

	var cache macroblockMotionCache
	err = m.fillMotionDecodeCaches(&cache, motionDecodeNeighbors{
		MBType:           MBType16x8 | MBTypeP0L0 | MBTypeP1L0,
		TopType:          MBType16x16 | MBTypeP0L0,
		TopLeftType:      MBType16x16 | MBTypeP0L0,
		TopRightType:     0,
		LeftType:         [2]uint32{MBType16x16 | MBTypeP0L0, MBType16x16 | MBTypeP0L0},
		TopXY:            topXY,
		TopLeftXY:        topLeftXY,
		LeftXY:           [2]int{leftXY, leftXY},
		TopLeftPartition: -1,
		ListCount:        1,
		CABAC:            true,
		SliceTypeNoS:     PictureTypeP,
	})
	if err != nil {
		t.Fatal(err)
	}
	base := int(h264Scan8[0])
	if cache.MV[0][base-8] != ([2]int16{100, 200}) || cache.MV[0][base+3-8] != ([2]int16{103, 203}) {
		t.Fatalf("top mv cache = %v ... %v", cache.MV[0][base-8], cache.MV[0][base+3-8])
	}
	if cache.Ref[0][base+0-8] != 7 || cache.Ref[0][base+2-8] != 8 {
		t.Fatalf("top ref cache = %d/%d", cache.Ref[0][base+0-8], cache.Ref[0][base+2-8])
	}
	if cache.MV[0][base-1] != ([2]int16{30, 40}) || cache.MV[0][base-1+3*8] != ([2]int16{33, 43}) {
		t.Fatalf("left mv cache = %v/%v", cache.MV[0][base-1], cache.MV[0][base-1+3*8])
	}
	if cache.Ref[0][base+4-8] != h264PartNotAvailable || cache.MV[0][base-1-8] != ([2]int16{77, 88}) || cache.Ref[0][base-1-8] != 6 {
		t.Fatalf("topright/topleft cache ref/mv = %d/%v/%d", cache.Ref[0][base+4-8], cache.MV[0][base-1-8], cache.Ref[0][base-1-8])
	}
	if cache.Ref[0][base+2] != h264PartNotAvailable || cache.Ref[0][base+2+2*8] != h264PartNotAvailable {
		t.Fatalf("partition hole refs = %d/%d", cache.Ref[0][base+2], cache.Ref[0][base+2+2*8])
	}
	if cache.MVD[0][base-8] != ([2]uint8{10, 20}) || cache.MVD[0][base-1] != ([2]uint8{50, 60}) || cache.MVD[0][base-1+3*8] != ([2]uint8{53, 63}) {
		t.Fatalf("mvd cache top/left = %v/%v/%v", cache.MVD[0][base-8], cache.MVD[0][base-1], cache.MVD[0][base-1+3*8])
	}
}

func TestFillMotionDecodeCachesBDirectCache(t *testing.T) {
	m, err := newMacroblockTables(3, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	topXY := 1
	leftTopXY := 4
	leftBotXY := 4
	for j := 0; j < 4; j++ {
		m.MotionVal[0][int(m.MB2BXY[topXY])+3*m.BStride+j] = [2]int16{int16(10 + j), int16(20 + j)}
	}
	m.RefIndex[0][4*topXY+2] = 1
	m.RefIndex[0][4*topXY+3] = 2
	leftBXY := int(m.MB2BXY[leftTopXY]) + 3
	for _, block := range []int{0, 1, 2, 3} {
		m.MotionVal[0][leftBXY+m.BStride*block] = [2]int16{int16(30 + block), int16(40 + block)}
	}
	m.RefIndex[0][4*leftTopXY+1] = 3
	m.RefIndex[0][4*leftTopXY+3] = 4
	m.DirectTable[4*topXY+2] = 91
	m.DirectTable[4*topXY+3] = 92
	m.DirectTable[4*leftBotXY+1+int(h264LeftBlockFrame[2]&^1)] = 93

	var cache macroblockMotionCache
	err = m.fillMotionDecodeCaches(&cache, motionDecodeNeighbors{
		MBType:       MBType8x8 | MBTypeP0L0 | MBTypeP1L0,
		TopType:      MBType8x8 | MBTypeP0L0,
		TopLeftType:  MBType16x16 | MBTypeP0L0,
		TopRightType: MBType16x16,
		LeftType: [2]uint32{
			MBTypeDirect2 | MBTypeL0L1,
			MBType8x8 | MBTypeP0L0,
		},
		TopXY:        topXY,
		TopLeftXY:    0,
		TopRightXY:   2,
		LeftXY:       [2]int{leftTopXY, leftBotXY},
		ListCount:    1,
		CABAC:        true,
		SliceTypeNoS: PictureTypeB,
	})
	if err != nil {
		t.Fatal(err)
	}
	base := int(h264Scan8[0])
	if cache.Direct[base] != uint8(MBType16x16>>1) || cache.Direct[base+3+3*8] != uint8(MBType16x16>>1) {
		t.Fatalf("current direct cache fill = %d/%d", cache.Direct[base], cache.Direct[base+3+3*8])
	}
	if cache.Direct[base+0-8] != 91 || cache.Direct[base+2-8] != 92 {
		t.Fatalf("top direct cache = %d/%d", cache.Direct[base+0-8], cache.Direct[base+2-8])
	}
	if cache.Direct[base-1] != uint8(MBTypeDirect2>>1) || cache.Direct[base-1+2*8] != 93 {
		t.Fatalf("left direct cache = %d/%d", cache.Direct[base-1], cache.Direct[base-1+2*8])
	}
}

func TestWriteBackMotionListCopiesMVRefAndMVD(t *testing.T) {
	m, err := newMacroblockTables(3, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	mbXY := 5
	base := int(h264Scan8[0])
	var cache macroblockMotionCache
	for row := 0; row < 4; row++ {
		for col := 0; col < 4; col++ {
			cache.MV[0][base+row*8+col] = [2]int16{int16(row*10 + col), int16(100 + row*10 + col)}
		}
	}
	cache.Ref[0][h264Scan8[0]] = 1
	cache.Ref[0][h264Scan8[4]] = 2
	cache.Ref[0][h264Scan8[8]] = 3
	cache.Ref[0][h264Scan8[12]] = 4
	for i := 0; i < 4; i++ {
		cache.MVD[0][base+8*3+i] = [2]uint8{uint8(10 + i), uint8(20 + i)}
	}
	cache.MVD[0][base+3+8*0] = [2]uint8{31, 41}
	cache.MVD[0][base+3+8*1] = [2]uint8{32, 42}
	cache.MVD[0][base+3+8*2] = [2]uint8{33, 43}

	if err := m.writeBackMotionList(mbXY, MBType16x16|MBTypeP0L0, 0, &cache, true); err != nil {
		t.Fatal(err)
	}
	bXY := int(m.MB2BXY[mbXY])
	if m.MotionVal[0][bXY] != ([2]int16{0, 100}) || m.MotionVal[0][bXY+3*m.BStride+3] != ([2]int16{33, 133}) {
		t.Fatalf("motion writeback = %v/%v", m.MotionVal[0][bXY], m.MotionVal[0][bXY+3*m.BStride+3])
	}
	if got := m.RefIndex[0][4*mbXY : 4*mbXY+4]; got[0] != 1 || got[1] != 2 || got[2] != 3 || got[3] != 4 {
		t.Fatalf("ref writeback = %v", got)
	}
	mvd := int(m.MB2BRXY[mbXY])
	if m.MVDTable[0][mvd+0] != ([2]uint8{10, 20}) || m.MVDTable[0][mvd+3] != ([2]uint8{13, 23}) ||
		m.MVDTable[0][mvd+4] != ([2]uint8{33, 43}) || m.MVDTable[0][mvd+5] != ([2]uint8{32, 42}) ||
		m.MVDTable[0][mvd+6] != ([2]uint8{31, 41}) {
		t.Fatalf("mvd writeback = %v", m.MVDTable[0][mvd:mvd+8])
	}
}

func TestWriteBackMotionFillsUnusedListAndDirectTable(t *testing.T) {
	m, err := newMacroblockTables(3, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	mbXY := 5
	var cache macroblockMotionCache
	sub := [4]uint32{MBType16x8, MBTypeDirect2, MBType16x16, MBType8x8}
	if err := m.writeBackMotion(mbXY, MBType8x8, PictureTypeB, true, &sub, &cache); err != nil {
		t.Fatal(err)
	}
	if got := m.RefIndex[0][4*mbXY : 4*mbXY+4]; got[0] != h264ListNotUsed || got[1] != h264ListNotUsed || got[2] != h264ListNotUsed || got[3] != h264ListNotUsed {
		t.Fatalf("unused list refs = %v", got)
	}
	if m.DirectTable[4*mbXY+0] != uint8(MBType16x8>>1) || m.DirectTable[4*mbXY+1] != uint8(MBTypeDirect2>>1) ||
		m.DirectTable[4*mbXY+2] != uint8(MBType16x16>>1) || m.DirectTable[4*mbXY+3] != uint8(MBType8x8>>1) {
		t.Fatalf("direct table = %v", m.DirectTable[4*mbXY:4*mbXY+4])
	}
}
