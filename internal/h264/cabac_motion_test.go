// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

import "testing"

func TestFillCABACInterMotionCache16x8WritesMVD(t *testing.T) {
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
	mb.MVD[0][8] = [2]int32{-3, 90}

	if err := fillCABACInterMotionCache(&cache, &mb, 1); err != nil {
		t.Fatal(err)
	}
	if cache.Ref[0][base] != 0 || cache.Ref[0][base+16] != 1 {
		t.Fatalf("16x8 refs = %d/%d", cache.Ref[0][base], cache.Ref[0][base+16])
	}
	if cache.MV[0][base] != ([2]int16{105, 95}) || cache.MV[0][base+3+8] != ([2]int16{105, 95}) {
		t.Fatalf("16x8 top mv = %v/%v", cache.MV[0][base], cache.MV[0][base+3+8])
	}
	if cache.MV[0][base+16] != ([2]int16{47, 150}) || cache.MV[0][base+16+3+8] != ([2]int16{47, 150}) {
		t.Fatalf("16x8 bottom mv = %v/%v", cache.MV[0][base+16], cache.MV[0][base+16+3+8])
	}
	if cache.MVD[0][base] != ([2]uint8{5, 5}) || cache.MVD[0][base+3+8] != ([2]uint8{5, 5}) {
		t.Fatalf("16x8 top mvd = %v/%v", cache.MVD[0][base], cache.MVD[0][base+3+8])
	}
	if cache.MVD[0][base+16] != ([2]uint8{3, 70}) || cache.MVD[0][base+16+3+8] != ([2]uint8{3, 70}) {
		t.Fatalf("16x8 bottom mvd = %v/%v", cache.MVD[0][base+16], cache.MVD[0][base+16+3+8])
	}
}

func TestFillCABACSubInterMotionCacheWritesSubMVDAndClearsUnused(t *testing.T) {
	var cache macroblockMotionCache
	base := int(h264Scan8[0])
	cache.Ref[0][base-1] = 0
	cache.Ref[0][base-8] = 0
	cache.Ref[0][base-8+2] = 0
	cache.MV[0][base-1] = [2]int16{1, 1}
	cache.MV[0][base-8] = [2]int16{3, 3}
	cache.MV[0][base-8+2] = [2]int16{2, 2}
	cache.MV[0][h264Scan8[4]] = [2]int16{99, 99}
	cache.MVD[0][h264Scan8[4]] = [2]uint8{99, 99}

	mb := cavlcInterMacroblockSyntax{}
	mb.MBType = MBType8x8 | MBTypeP0L0 | MBTypeP1L0
	mb.PartitionCount = 4
	mb.SubMBType[0] = MBType16x8 | MBTypeP0L0
	mb.SubPartitionCount[0] = 2
	mb.Ref[0][0] = 0
	mb.MVD[0][0] = [2]int32{4, -5}
	mb.MVD[0][2] = [2]int32{-6, 7}
	for i := 1; i < 4; i++ {
		mb.SubPartitionCount[i] = 1
		mb.Ref[0][i] = -1
	}

	if err := fillCABACInterMotionCache(&cache, &mb, 1); err != nil {
		t.Fatal(err)
	}
	if cache.Ref[0][base] != 0 || cache.Ref[0][base+1] != 0 {
		t.Fatalf("sub refs = %d/%d", cache.Ref[0][base], cache.Ref[0][base+1])
	}
	if cache.MVD[0][base] != ([2]uint8{4, 5}) || cache.MVD[0][base+1] != ([2]uint8{4, 5}) {
		t.Fatalf("first 8x4 mvd = %v/%v", cache.MVD[0][base], cache.MVD[0][base+1])
	}
	second := int(h264Scan8[2])
	if cache.MVD[0][second] != ([2]uint8{6, 7}) || cache.MVD[0][second+1] != ([2]uint8{6, 7}) {
		t.Fatalf("second 8x4 mvd = %v/%v", cache.MVD[0][second], cache.MVD[0][second+1])
	}
	unused := int(h264Scan8[4])
	if cache.Ref[0][unused] != h264ListNotUsed || cache.MV[0][unused] != ([2]int16{}) || cache.MVD[0][unused] != ([2]uint8{}) {
		t.Fatalf("unused sub state ref/mv/mvd = %d/%v/%v", cache.Ref[0][unused], cache.MV[0][unused], cache.MVD[0][unused])
	}
}

func TestFillCABACInterMotionCacheRejectsDirect(t *testing.T) {
	var cache macroblockMotionCache
	mb := cavlcInterMacroblockSyntax{}
	mb.MBType = MBTypeDirect2 | MBTypeL0L1
	if err := fillCABACInterMotionCache(&cache, &mb, 2); err != ErrUnsupported {
		t.Fatalf("err = %v, want ErrUnsupported", err)
	}
}

func TestCABACMVDCacheMagnitude(t *testing.T) {
	cases := []struct {
		delta int32
		want  uint8
	}{
		{0, 0},
		{3, 3},
		{-4, 4},
		{90, 70},
		{-91, 70},
	}
	for _, tc := range cases {
		if got := cabacMVDCacheMagnitude(tc.delta); got != tc.want {
			t.Fatalf("mvd magnitude(%d) = %d, want %d", tc.delta, got, tc.want)
		}
	}
}
