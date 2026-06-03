// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

import "testing"

func TestReadCAVLCRefIndex(t *testing.T) {
	gb := newBitReader(cavlcBitString(""))
	if got, err := readCAVLCRefIndex(&gb, 1); err != nil || got != 0 {
		t.Fatalf("ref count 1 = %d, %v; want 0 nil", got, err)
	}

	gb = newBitReader(cavlcBitString("0"))
	if got, err := readCAVLCRefIndex(&gb, 2); err != nil || got != 1 {
		t.Fatalf("ref count 2 bit0 = %d, %v; want 1 nil", got, err)
	}

	gb = newBitReader(cavlcBitString("010"))
	if got, err := readCAVLCRefIndex(&gb, 3); err != nil || got != 1 {
		t.Fatalf("ref count 3 ue1 = %d, %v; want 1 nil", got, err)
	}
}

func TestDecodeCAVLCInterP16x16MacroblockNoResidual(t *testing.T) {
	pps := cavlcFlatQMulPPS()
	sps := &SPS{BitDepthLuma: 8, ChromaFormatIDC: 1}
	var ctx cavlcResidualContext
	fillCAVLCNonZero(&ctx.NonZeroCountCache, int(h264Scan8[0]), 4, 4, 8, 7)
	gb := newBitReader(cavlcBitString("1111"))

	mb, err := ctx.decodeCAVLCInterPMacroblock(&gb, pps, sps, 24, [2]uint32{1, 0}, false)
	if err != nil {
		t.Fatalf("decode inter p mb failed: %v", err)
	}
	if mb.MBType != (MBType16x16|MBTypeP0L0) || mb.PartitionCount != 1 || mb.Ref[0][0] != 0 {
		t.Fatalf("mb type %#x partitions %d ref %d", mb.MBType, mb.PartitionCount, mb.Ref[0][0])
	}
	if mb.MVD[0][0] != ([2]int32{}) {
		t.Fatalf("mvd = %v, want zero", mb.MVD[0][0])
	}
	if mb.CBP != 0 || mb.QScale != 24 {
		t.Fatalf("cbp/qscale = %d/%d, want 0/24", mb.CBP, mb.QScale)
	}
	for _, n := range []int{0, 1, 2, 3} {
		if ctx.NonZeroCountCache[h264Scan8[n]] != 0 {
			t.Fatalf("nnz block%d = %d, want 0", n, ctx.NonZeroCountCache[h264Scan8[n]])
		}
	}
	if gb.bitPos != 4 {
		t.Fatalf("consumed %d bits, want 4", gb.bitPos)
	}
}

func TestDecodeCAVLCInterP16x8MacroblockRefsAndMVD(t *testing.T) {
	pps := cavlcFlatQMulPPS()
	sps := &SPS{BitDepthLuma: 8, ChromaFormatIDC: 1}
	var ctx cavlcResidualContext
	gb := newBitReader(cavlcBitString("01001010110111"))

	mb, err := ctx.decodeCAVLCInterPMacroblock(&gb, pps, sps, 12, [2]uint32{2, 0}, false)
	if err != nil {
		t.Fatalf("decode inter p16x8 mb failed: %v", err)
	}
	if mb.MBType != (MBType16x8|MBTypeP0L0|MBTypeP1L0) || mb.Ref[0][0] != 1 || mb.Ref[0][1] != 0 {
		t.Fatalf("type/ref = %#x/%v", mb.MBType, mb.Ref[0])
	}
	if mb.MVD[0][0] != ([2]int32{1, 0}) || mb.MVD[0][8] != ([2]int32{0, -1}) {
		t.Fatalf("mvd0=%v mvd8=%v, want [1 0] [0 -1]", mb.MVD[0][0], mb.MVD[0][8])
	}
	if mb.CBP != 0 || gb.bitPos != 14 {
		t.Fatalf("cbp/consumed = %d/%d, want 0/14", mb.CBP, gb.bitPos)
	}
}

func TestDecodeCAVLCInterP8x8SubMacroblockNoResidual(t *testing.T) {
	pps := cavlcFlatQMulPPS()
	sps := &SPS{BitDepthLuma: 8, ChromaFormatIDC: 1}
	var ctx cavlcResidualContext
	gb := newBitReader(cavlcBitString("001001111111111111"))

	mb, err := ctx.decodeCAVLCInterPMacroblock(&gb, pps, sps, 18, [2]uint32{1, 0}, false)
	if err != nil {
		t.Fatalf("decode inter p8x8 mb failed: %v", err)
	}
	if mb.MBType != (MBType8x8|MBTypeP0L0|MBTypeP1L0) || mb.PartitionCount != 4 {
		t.Fatalf("type/partitions = %#x/%d", mb.MBType, mb.PartitionCount)
	}
	for i := 0; i < 4; i++ {
		if mb.SubMBType[i] != (MBType16x16|MBTypeP0L0) || mb.SubPartitionCount[i] != 1 {
			t.Fatalf("sub[%d] type/partitions = %#x/%d", i, mb.SubMBType[i], mb.SubPartitionCount[i])
		}
		if mb.MVD[0][4*i] != ([2]int32{}) {
			t.Fatalf("sub[%d] mvd = %v, want zero", i, mb.MVD[0][4*i])
		}
	}
	if mb.CBP != 0 || gb.bitPos != 18 {
		t.Fatalf("cbp/consumed = %d/%d, want 0/18", mb.CBP, gb.bitPos)
	}
}

func TestDecodeCAVLCInterBDirectSyntaxSkipsRefsMVD(t *testing.T) {
	pps := cavlcFlatQMulPPS()
	sps := &SPS{BitDepthLuma: 8, ChromaFormatIDC: 1}
	var ctx cavlcResidualContext
	gb := newBitReader(cavlcBitString("11"))

	mb, err := ctx.decodeCAVLCInterBMacroblock(&gb, pps, sps, 18, [2]uint32{1, 1}, false)
	if err != nil {
		t.Fatalf("decode direct B syntax failed: %v", err)
	}
	if mb.MBType != (MBTypeDirect2|MBTypeL0L1) || mb.CBP != 0 || gb.bitPos != 2 {
		t.Fatalf("direct type/cbp/bits = %#x/%d/%d", mb.MBType, mb.CBP, gb.bitPos)
	}
	for list := 0; list < 2; list++ {
		for i := 0; i < 4; i++ {
			if mb.Ref[list][i] != -1 || mb.MVD[list][4*i] != ([2]int32{}) {
				t.Fatalf("direct list %d sub %d ref/mvd = %d/%v", list, i, mb.Ref[list][i], mb.MVD[list][4*i])
			}
		}
	}
}

func TestDecodeCAVLCInterBDirectSkips8x8DCTWhenDirectInferenceDisabled(t *testing.T) {
	pps := cavlcFlatQMulPPS()
	sps := &SPS{BitDepthLuma: 8, ChromaFormatIDC: 0, Direct8x8InferenceFlag: 0}
	var ctx cavlcResidualContext
	gb := newBitReader(cavlcBitString("101010101111"))

	mb, err := ctx.decodeCAVLCInterBMacroblock(&gb, pps, sps, 18, [2]uint32{1, 1}, true)
	if err != nil {
		t.Fatalf("decode direct B residual failed: %v", err)
	}
	if mb.MBType&MBType8x8DCT != 0 || mb.TransformSize8x8DCT {
		t.Fatalf("direct B type %#x transform8x8=%t, want transform flag skipped", mb.MBType, mb.TransformSize8x8DCT)
	}
	if mb.CBP != 1 || mb.QScale != 18 {
		t.Fatalf("cbp/qscale = %d/%d, want 1/18", mb.CBP, mb.QScale)
	}
	if gb.bitPos != 12 {
		t.Fatalf("consumed %d bits, want 12", gb.bitPos)
	}
}

func TestDecodeCAVLCInterB16x16BiMacroblockNoResidual(t *testing.T) {
	pps := cavlcFlatQMulPPS()
	sps := &SPS{BitDepthLuma: 8, ChromaFormatIDC: 1}
	var ctx cavlcResidualContext
	gb := newBitReader(cavlcBitString("001000010110111"))

	mb, err := ctx.decodeCAVLCInterBMacroblock(&gb, pps, sps, 22, [2]uint32{2, 1}, false)
	if err != nil {
		t.Fatalf("decode inter b16x16 mb failed: %v", err)
	}
	if mb.MBType != (MBType16x16|MBTypeP0L0|MBTypeP0L1) || mb.Ref[0][0] != 1 || mb.Ref[1][0] != 0 {
		t.Fatalf("type/ref0/ref1 = %#x/%v/%v", mb.MBType, mb.Ref[0], mb.Ref[1])
	}
	if mb.MVD[0][0] != ([2]int32{1, 0}) || mb.MVD[1][0] != ([2]int32{0, -1}) {
		t.Fatalf("mvd list0=%v list1=%v, want [1 0] [0 -1]", mb.MVD[0][0], mb.MVD[1][0])
	}
	if mb.CBP != 0 || gb.bitPos != 15 {
		t.Fatalf("cbp/consumed = %d/%d, want 0/15", mb.CBP, gb.bitPos)
	}
}

func TestDecodeCAVLCInterB8x8SubMacroblockNoResidual(t *testing.T) {
	pps := cavlcFlatQMulPPS()
	sps := &SPS{BitDepthLuma: 8, ChromaFormatIDC: 1}
	var ctx cavlcResidualContext
	gb := newBitReader(cavlcBitString("000010111010010010010111111111"))

	mb, err := ctx.decodeCAVLCInterBMacroblock(&gb, pps, sps, 16, [2]uint32{1, 1}, false)
	if err != nil {
		t.Fatalf("decode inter b8x8 mb failed: %v", err)
	}
	if mb.MBType != (MBType8x8|MBTypeP0L0|MBTypeP0L1|MBTypeP1L0|MBTypeP1L1) || mb.PartitionCount != 4 {
		t.Fatalf("type/partitions = %#x/%d", mb.MBType, mb.PartitionCount)
	}
	for i := 0; i < 4; i++ {
		if mb.SubMBType[i] != (MBType16x16|MBTypeP0L0) || mb.Ref[0][i] != 0 || mb.Ref[1][i] != -1 {
			t.Fatalf("sub[%d] type/ref0/ref1 = %#x/%d/%d", i, mb.SubMBType[i], mb.Ref[0][i], mb.Ref[1][i])
		}
		if mb.MVD[0][4*i] != ([2]int32{}) {
			t.Fatalf("sub[%d] mvd = %v, want zero", i, mb.MVD[0][4*i])
		}
	}
	if mb.CBP != 0 || gb.bitPos != 30 {
		t.Fatalf("cbp/consumed = %d/%d, want 0/30", mb.CBP, gb.bitPos)
	}
}

func TestDecodeCAVLCInterB8x8DirectSubMacroblockNoResidual(t *testing.T) {
	pps := cavlcFlatQMulPPS()
	sps := &SPS{BitDepthLuma: 8, ChromaFormatIDC: 1, Direct8x8InferenceFlag: 1}
	var ctx cavlcResidualContext
	gb := newBitReader(cavlcBitString("000010111101001100100111111111"))

	mb, err := ctx.decodeCAVLCInterBMacroblock(&gb, pps, sps, 16, [2]uint32{1, 1}, true)
	if err != nil {
		t.Fatalf("decode inter b8x8 direct-sub mb failed: %v", err)
	}
	if mb.MBType != (MBType8x8|MBTypeP0L0|MBTypeP0L1|MBTypeP1L0|MBTypeP1L1) || mb.PartitionCount != 4 {
		t.Fatalf("type/partitions = %#x/%d", mb.MBType, mb.PartitionCount)
	}
	wantSub := [4]uint32{
		MBTypeDirect2,
		MBType16x16 | MBTypeP0L0,
		MBType16x16 | MBTypeP0L1,
		MBType16x16 | MBTypeP0L0 | MBTypeP0L1,
	}
	for i := 0; i < 4; i++ {
		if mb.SubMBType[i] != wantSub[i] {
			t.Fatalf("sub[%d] type = %#x, want %#x", i, mb.SubMBType[i], wantSub[i])
		}
		if isDirect(mb.SubMBType[i]) {
			if mb.Ref[0][i] != -1 || mb.Ref[1][i] != -1 || mb.MVD[0][4*i] != ([2]int32{}) || mb.MVD[1][4*i] != ([2]int32{}) {
				t.Fatalf("direct sub[%d] ref/mvd = %v/%v %v/%v", i, mb.Ref[0][i], mb.Ref[1][i], mb.MVD[0][4*i], mb.MVD[1][4*i])
			}
			continue
		}
		if isDir(mb.SubMBType[i], 0, 0) && mb.Ref[0][i] != 0 {
			t.Fatalf("sub[%d] ref0 = %d, want 0", i, mb.Ref[0][i])
		}
		if isDir(mb.SubMBType[i], 0, 1) && mb.Ref[1][i] != 0 {
			t.Fatalf("sub[%d] ref1 = %d, want 0", i, mb.Ref[1][i])
		}
	}
	if mb.CBP != 0 || gb.bitPos != 30 {
		t.Fatalf("cbp/consumed = %d/%d, want 0/30", mb.CBP, gb.bitPos)
	}
}
