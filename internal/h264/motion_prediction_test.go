// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

import "testing"

func TestPredMotionBranches(t *testing.T) {
	var cache macroblockMotionCache
	index8 := int(h264Scan8[0])
	left := index8 - 1
	top := index8 - 8
	diag := index8 - 8 + 4

	cache.Ref[0][left] = 2
	cache.Ref[0][top] = 2
	cache.Ref[0][diag] = 2
	cache.MV[0][left] = [2]int16{10, 100}
	cache.MV[0][top] = [2]int16{30, 50}
	cache.MV[0][diag] = [2]int16{20, 80}
	got, err := predMotion(&cache, 0, 4, 0, 2)
	if err != nil {
		t.Fatal(err)
	}
	if got != ([2]int16{20, 80}) {
		t.Fatalf("median pred = %v, want [20 80]", got)
	}

	cache.Ref[0][left] = 1
	cache.Ref[0][top] = 2
	cache.Ref[0][diag] = 3
	cache.MV[0][top] = [2]int16{-7, 44}
	got, err = predMotion(&cache, 0, 4, 0, 2)
	if err != nil {
		t.Fatal(err)
	}
	if got != ([2]int16{-7, 44}) {
		t.Fatalf("single top match = %v, want [-7 44]", got)
	}

	topLeft := index8 - 8 - 1
	cache.Ref[0][left] = 1
	cache.Ref[0][top] = h264PartNotAvailable
	cache.Ref[0][diag] = h264PartNotAvailable
	cache.Ref[0][topLeft] = h264PartNotAvailable
	cache.MV[0][left] = [2]int16{55, -12}
	got, err = predMotion(&cache, 0, 4, 0, 9)
	if err != nil {
		t.Fatal(err)
	}
	if got != ([2]int16{55, -12}) {
		t.Fatalf("unavailable top/diag fallback = %v, want [55 -12]", got)
	}
}

func TestPredPartitionFastPaths(t *testing.T) {
	var cache macroblockMotionCache
	cache.Ref[0][h264Scan8[0]-8] = 5
	cache.MV[0][h264Scan8[0]-8] = [2]int16{1, 2}
	got, err := pred16x8Motion(&cache, 0, 0, 5)
	if err != nil {
		t.Fatal(err)
	}
	if got != ([2]int16{1, 2}) {
		t.Fatalf("16x8 top = %v", got)
	}

	cache.Ref[0][h264Scan8[8]-1] = 6
	cache.MV[0][h264Scan8[8]-1] = [2]int16{3, 4}
	got, err = pred16x8Motion(&cache, 8, 0, 6)
	if err != nil {
		t.Fatal(err)
	}
	if got != ([2]int16{3, 4}) {
		t.Fatalf("16x8 bottom-left = %v", got)
	}

	cache.Ref[0][h264Scan8[0]-1] = 7
	cache.MV[0][h264Scan8[0]-1] = [2]int16{5, 6}
	got, err = pred8x16Motion(&cache, 0, 0, 7)
	if err != nil {
		t.Fatal(err)
	}
	if got != ([2]int16{5, 6}) {
		t.Fatalf("8x16 left = %v", got)
	}

	diag := int(h264Scan8[4]) - 8 + 2
	cache.Ref[0][diag] = 8
	cache.MV[0][diag] = [2]int16{7, 8}
	got, err = pred8x16Motion(&cache, 4, 0, 8)
	if err != nil {
		t.Fatal(err)
	}
	if got != ([2]int16{7, 8}) {
		t.Fatalf("8x16 diagonal = %v", got)
	}
}

func TestPredPSkipMotionMedianAndZero(t *testing.T) {
	m, err := newMacroblockTables(3, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	leftXY := 4
	topXY := 1
	topRightXY := 2
	m.RefIndex[0][4*leftXY+1] = 0
	m.MotionVal[0][int(m.MB2BXY[leftXY])+3] = [2]int16{10, 20}
	m.RefIndex[0][4*topXY+2] = 0
	m.MotionVal[0][int(m.MB2BXY[topXY])+3*m.BStride] = [2]int16{30, 40}
	m.RefIndex[0][4*topRightXY+2] = 0
	m.MotionVal[0][int(m.MB2BXY[topRightXY])+3*m.BStride] = [2]int16{20, 10}

	var cache macroblockMotionCache
	err = m.predPSkipMotion(&cache, motionDecodeNeighbors{
		LeftType:     [2]uint32{MBType16x16 | MBTypeP0L0, 0},
		TopType:      MBType16x16 | MBTypeP0L0,
		TopRightType: MBType16x16 | MBTypeP0L0,
		LeftXY:       [2]int{leftXY, 0},
		TopXY:        topXY,
		TopRightXY:   topRightXY,
	})
	if err != nil {
		t.Fatal(err)
	}
	base := int(h264Scan8[0])
	if cache.Ref[0][base] != 0 || cache.Ref[0][base+3+3*8] != 0 {
		t.Fatalf("pskip refs = %d/%d", cache.Ref[0][base], cache.Ref[0][base+3+3*8])
	}
	if cache.MV[0][base] != ([2]int16{20, 20}) || cache.MV[0][base+3+3*8] != ([2]int16{20, 20}) {
		t.Fatalf("pskip median mv = %v/%v", cache.MV[0][base], cache.MV[0][base+3+3*8])
	}

	for i := range cache.MV[0] {
		cache.MV[0][i] = [2]int16{99, 99}
	}
	if err := m.predPSkipMotion(&cache, motionDecodeNeighbors{}); err != nil {
		t.Fatal(err)
	}
	if cache.MV[0][base] != ([2]int16{}) || cache.MV[0][base+3+3*8] != ([2]int16{}) {
		t.Fatalf("pskip unavailable zero = %v/%v", cache.MV[0][base], cache.MV[0][base+3+3*8])
	}
}

func TestFillCAVLCInterMotionCache16x8(t *testing.T) {
	var cache macroblockMotionCache
	base := int(h264Scan8[0])
	cache.Ref[0][base-8] = 0
	cache.MV[0][base-8] = [2]int16{100, 100}
	cache.Ref[0][int(h264Scan8[8])-1] = 1
	cache.MV[0][int(h264Scan8[8])-1] = [2]int16{50, 60}

	mb := cavlcInterMacroblockSyntax{}
	mb.MBType = MBType16x8 | MBTypeP0L0 | MBTypeP1L0
	mb.Ref[0][0] = 0
	mb.Ref[0][1] = 1
	mb.MVD[0][0] = [2]int32{5, -5}
	mb.MVD[0][8] = [2]int32{-3, 4}

	if err := fillCAVLCInterMotionCache(&cache, &mb, 1); err != nil {
		t.Fatal(err)
	}
	if cache.Ref[0][base] != 0 || cache.Ref[0][base+16] != 1 {
		t.Fatalf("16x8 refs = %d/%d", cache.Ref[0][base], cache.Ref[0][base+16])
	}
	if cache.MV[0][base] != ([2]int16{105, 95}) || cache.MV[0][base+3+8] != ([2]int16{105, 95}) {
		t.Fatalf("16x8 top mv = %v/%v", cache.MV[0][base], cache.MV[0][base+3+8])
	}
	if cache.MV[0][base+16] != ([2]int16{47, 64}) || cache.MV[0][base+16+3+8] != ([2]int16{47, 64}) {
		t.Fatalf("16x8 bottom mv = %v/%v", cache.MV[0][base+16], cache.MV[0][base+16+3+8])
	}
}

func TestFillCAVLCInterMotionCache16x16WrapsToInt16(t *testing.T) {
	var cache macroblockMotionCache
	base := int(h264Scan8[0])
	for _, idx := range []int{base - 1, base - 8, base - 8 + 4} {
		cache.Ref[0][idx] = 2
		cache.MV[0][idx] = [2]int16{32760, -10}
	}

	mb := cavlcInterMacroblockSyntax{}
	mb.MBType = MBType16x16 | MBTypeP0L0
	mb.Ref[0][0] = 2
	mb.MVD[0][0] = [2]int32{20, -32770}

	if err := fillCAVLCInterMotionCache(&cache, &mb, 1); err != nil {
		t.Fatal(err)
	}
	if cache.MV[0][base] != ([2]int16{-32756, 32756}) || cache.MV[0][base+3+3*8] != ([2]int16{-32756, 32756}) {
		t.Fatalf("wrapped 16x16 mv = %v/%v", cache.MV[0][base], cache.MV[0][base+3+3*8])
	}
}

func TestFillCAVLCSubInterMotionCache8x8(t *testing.T) {
	var cache macroblockMotionCache
	base := int(h264Scan8[0])
	cache.Ref[0][base-1] = 0
	cache.Ref[0][base-8] = 0
	cache.Ref[0][base-8+2] = 0
	cache.MV[0][base-1] = [2]int16{1, 1}
	cache.MV[0][base-8] = [2]int16{3, 3}
	cache.MV[0][base-8+2] = [2]int16{2, 2}

	mb := cavlcInterMacroblockSyntax{}
	mb.MBType = MBType8x8 | MBTypeP0L0 | MBTypeP1L0
	mb.PartitionCount = 4
	mb.SubMBType[0] = MBType16x16 | MBTypeP0L0
	mb.SubPartitionCount[0] = 1
	mb.Ref[0][0] = 0
	mb.MVD[0][0] = [2]int32{5, 6}
	for i := 1; i < 4; i++ {
		mb.SubMBType[i] = 0
		mb.SubPartitionCount[i] = 1
		mb.Ref[0][i] = -1
	}

	if err := fillCAVLCInterMotionCache(&cache, &mb, 1); err != nil {
		t.Fatal(err)
	}
	if cache.Ref[0][base] != 0 || cache.Ref[0][base+1] != 0 || cache.Ref[0][base+8] != 0 || cache.Ref[0][base+9] != 0 {
		t.Fatalf("8x8 sub refs = %d/%d/%d/%d", cache.Ref[0][base], cache.Ref[0][base+1], cache.Ref[0][base+8], cache.Ref[0][base+9])
	}
	if cache.MV[0][base] != ([2]int16{7, 8}) || cache.MV[0][base+1] != ([2]int16{7, 8}) ||
		cache.MV[0][base+8] != ([2]int16{7, 8}) || cache.MV[0][base+9] != ([2]int16{7, 8}) {
		t.Fatalf("8x8 sub mv = %v/%v/%v/%v", cache.MV[0][base], cache.MV[0][base+1], cache.MV[0][base+8], cache.MV[0][base+9])
	}
}

func TestInitMotionDecodeCacheSentinels(t *testing.T) {
	var cache macroblockMotionCache
	initMotionDecodeCacheSentinels(&cache)
	for list := 0; list < 2; list++ {
		for _, idx := range []uint8{h264Scan8[5] + 1, h264Scan8[7] + 1, h264Scan8[13] + 1} {
			if cache.Ref[list][idx] != h264PartNotAvailable {
				t.Fatalf("list %d sentinel %d = %d, want PART_NOT_AVAILABLE", list, idx, cache.Ref[list][idx])
			}
		}
	}
}

func TestFillCAVLCSubInterMotionCacheUsesFFmpegDiagonalSentinel(t *testing.T) {
	var cache macroblockMotionCache
	initMotionDecodeCacheSentinels(&cache)

	mb := cavlcInterMacroblockSyntax{}
	mb.MBType = MBType8x8 | MBTypeP0L0 | MBTypeP0L1 | MBTypeP1L0 | MBTypeP1L1
	mb.PartitionCount = 4
	mb.SubMBType = [4]uint32{
		MBType16x16 | MBTypeP0L0 | MBTypeP0L1,
		MBType16x16 | MBTypeP0L0 | MBTypeP0L1,
		MBType16x16 | MBTypeP0L0 | MBTypeP0L1,
		MBType16x16 | MBTypeP0L1,
	}
	mb.SubPartitionCount = [4]uint8{1, 1, 1, 1}
	mb.Ref[0] = [4]int32{0, 0, 0, -1}
	mb.Ref[1] = [4]int32{0, 0, 0, 0}
	mb.MVD[1][0] = [2]int32{0, 40}
	mb.MVD[1][4] = [2]int32{-7, 40}
	mb.MVD[1][8] = [2]int32{0, -22}
	mb.MVD[1][12] = [2]int32{-9, -16}

	if err := fillCAVLCInterMotionCache(&cache, &mb, 2); err != nil {
		t.Fatal(err)
	}
	if got := cache.MV[1][h264Scan8[12]]; got != ([2]int16{-9, 24}) {
		t.Fatalf("sub3 list1 mv = %v, want [-9 24]", got)
	}
}
