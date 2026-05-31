// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

import "testing"

func TestDecodeCAVLCMBType(t *testing.T) {
	cases := []struct {
		name         string
		bits         string
		sliceType    int32
		sliceTypeNoS int32
		wantType     uint32
		wantPart     uint8
		wantCBP      int
	}{
		{"i intra4x4", "1", PictureTypeI, PictureTypeI, MBTypeIntra4x4, 0, -1},
		{"i intra16x16", "010", PictureTypeI, PictureTypeI, MBTypeIntra16x16, 0, 0},
		{"p inter16x16", "1", PictureTypeP, PictureTypeP, MBType16x16 | MBTypeP0L0, 1, 0},
		{"p intra16x16", "00111", PictureTypeP, PictureTypeP, MBTypeIntra16x16, 0, 0},
		{"b direct", "1", PictureTypeB, PictureTypeB, MBTypeDirect2 | MBTypeL0L1, 1, 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gb := newBitReader(cavlcBitString(tc.bits))
			got, err := decodeCAVLCMBType(&gb, tc.sliceType, tc.sliceTypeNoS)
			if err != nil {
				t.Fatalf("decode mb type failed: %v", err)
			}
			if got.MBType != tc.wantType || got.PartitionCount != tc.wantPart || got.CBP != tc.wantCBP {
				t.Fatalf("mb = type %#x part %d cbp %d, want type %#x part %d cbp %d", got.MBType, got.PartitionCount, got.CBP, tc.wantType, tc.wantPart, tc.wantCBP)
			}
		})
	}
}

func TestDecodeCAVLCCBP(t *testing.T) {
	cases := []struct {
		name         string
		bits         string
		mbType       uint32
		decodeChroma bool
		initialCBP   int
		want         int
	}{
		{"intra chroma", "00100", MBTypeIntra4x4, true, -1, 0},
		{"inter chroma", "0001101", MBType16x16 | MBTypeP0L0, true, 0, 47},
		{"intra gray", "010", MBTypeIntra4x4, false, -1, 0},
		{"inter gray", "010", MBType16x16 | MBTypeP0L0, false, 0, 1},
		{"intra16 returns table cbp", "", MBTypeIntra16x16, true, 32, 32},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gb := newBitReader(cavlcBitString(tc.bits))
			got, err := decodeCAVLCCBP(&gb, tc.mbType, tc.decodeChroma, tc.initialCBP)
			if err != nil {
				t.Fatalf("decode cbp failed: %v", err)
			}
			if got != tc.want {
				t.Fatalf("cbp = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestUpdateCAVLCQScale(t *testing.T) {
	cases := []struct {
		name   string
		qscale int
		dquant int32
		maxQP  int32
		want   int
		err    bool
	}{
		{"same", 26, 0, 51, 26, false},
		{"negative wraps", 0, -1, 51, 51, false},
		{"positive wraps", 51, 1, 51, 0, false},
		{"out of range", 10, 100, 51, 51, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := updateCAVLCQScale(tc.qscale, tc.dquant, tc.maxQP)
			if (err != nil) != tc.err {
				t.Fatalf("err = %v, want err %v", err, tc.err)
			}
			if got != tc.want {
				t.Fatalf("qscale = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestDecodeCAVLCIntra16x16MacroblockNoResidual(t *testing.T) {
	pps := cavlcFlatQMulPPS()
	sps := &SPS{BitDepthLuma: 8, ChromaFormatIDC: 1}
	var ctx cavlcResidualContext
	gb := newBitReader(cavlcBitString("010111"))

	mb, err := ctx.decodeCAVLCIntraMacroblock(&gb, pps, sps, PictureTypeI, PictureTypeI, 26, false, [16]int8{})
	if err != nil {
		t.Fatalf("decode intra mb failed: %v", err)
	}
	if mb.MBType != MBTypeIntra16x16 || mb.CBP != 0 || mb.CBPTable != 0 || mb.QScale != 26 {
		t.Fatalf("mb = type %#x cbp %d cbpTable %d qscale %d", mb.MBType, mb.CBP, mb.CBPTable, mb.QScale)
	}
	if mb.ChromaPredMode != 0 {
		t.Fatalf("chroma pred = %d, want 0", mb.ChromaPredMode)
	}
	if ctx.NonZeroCountCache[h264Scan8[lumaDCBlockIndex]] != 0 {
		t.Fatalf("luma dc nnz = %d, want 0", ctx.NonZeroCountCache[h264Scan8[lumaDCBlockIndex]])
	}
	if gb.bitPos != 6 {
		t.Fatalf("consumed %d bits, want 6", gb.bitPos)
	}
}

func TestDecodeCAVLCIntra4x4MacroblockNoResidual(t *testing.T) {
	pps := cavlcFlatQMulPPS()
	sps := &SPS{BitDepthLuma: 8, ChromaFormatIDC: 1}
	var pred [16]int8
	for i := range pred {
		pred[i] = 2
	}
	var ctx cavlcResidualContext
	fillCAVLCNonZero(&ctx.NonZeroCountCache, int(h264Scan8[0]), 4, 4, 8, 9)
	gb := newBitReader(cavlcBitString("11111111111111111100100"))

	mb, err := ctx.decodeCAVLCIntraMacroblock(&gb, pps, sps, PictureTypeI, PictureTypeI, 20, false, pred)
	if err != nil {
		t.Fatalf("decode intra4x4 mb failed: %v", err)
	}
	if mb.MBType != MBTypeIntra4x4 || mb.CBP != 0 || mb.QScale != 20 {
		t.Fatalf("mb = type %#x cbp %d qscale %d", mb.MBType, mb.CBP, mb.QScale)
	}
	for i, mode := range mb.Intra4x4PredMode {
		if mode != 2 {
			t.Fatalf("pred mode[%d] = %d, want 2", i, mode)
		}
	}
	for _, n := range []int{0, 1, 2, 3} {
		if ctx.NonZeroCountCache[h264Scan8[n]] != 0 {
			t.Fatalf("nnz block%d = %d, want 0", n, ctx.NonZeroCountCache[h264Scan8[n]])
		}
	}
	if gb.bitPos != 23 {
		t.Fatalf("consumed %d bits, want 23", gb.bitPos)
	}
}

func TestDecodeCAVLCIntra4x4ModesWithCacheUsesDecodedNeighbors(t *testing.T) {
	var cache [h264IntraPredModeCacheSize]int8
	for i := range cache {
		cache[i] = 8
	}
	gb := newBitReader(cavlcBitString("0111111111111111111"))
	mb := cavlcMacroblockSyntax{MBType: MBTypeIntra4x4}

	got, err := decodeCAVLCIntra4x4ModesWithCache(&gb, mb, false, &cache)
	if err != nil {
		t.Fatalf("decode intra4x4 modes failed: %v", err)
	}
	if got.Intra4x4PredMode[0] != 7 || got.Intra4x4PredMode[1] != 7 {
		t.Fatalf("modes[0:2] = %d/%d, want 7/7", got.Intra4x4PredMode[0], got.Intra4x4PredMode[1])
	}
	if cache[h264Scan8[1]] != 7 {
		t.Fatalf("cache block1 = %d, want 7", cache[h264Scan8[1]])
	}
}
