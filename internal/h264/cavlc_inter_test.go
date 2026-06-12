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

func TestWriteCAVLCRefIndexRoundTripsThroughReader(t *testing.T) {
	tests := []struct {
		refCount uint32
		ref      int32
		bits     uint32
	}{
		{refCount: 1, ref: 0, bits: 0},
		{refCount: 2, ref: 0, bits: 1},
		{refCount: 2, ref: 1, bits: 1},
		{refCount: 3, ref: 0, bits: 1},
		{refCount: 3, ref: 1, bits: 3},
		{refCount: 5, ref: 4, bits: 5},
	}
	for _, tt := range tests {
		var bw BitWriter
		if err := writeCAVLCRefIndex(&bw, tt.refCount, tt.ref); err != nil {
			t.Fatalf("write refCount=%d ref=%d failed: %v", tt.refCount, tt.ref, err)
		}
		if bw.BitLen() != tt.bits {
			t.Fatalf("refCount=%d ref=%d wrote %d bits, want %d", tt.refCount, tt.ref, bw.BitLen(), tt.bits)
		}

		gb := newBitReader(bw.Bytes())
		got, err := readCAVLCRefIndex(&gb, tt.refCount)
		if err != nil {
			t.Fatalf("read refCount=%d ref=%d failed: %v", tt.refCount, tt.ref, err)
		}
		if got != tt.ref {
			t.Fatalf("refCount=%d round trip = %d, want %d", tt.refCount, got, tt.ref)
		}
		if gb.bitPos != bw.BitLen() {
			t.Fatalf("refCount=%d consumed %d bits, want %d", tt.refCount, gb.bitPos, bw.BitLen())
		}
	}
}

func TestWriteCAVLCRefIndexRejectsInvalid(t *testing.T) {
	var bw BitWriter
	tests := []struct {
		name     string
		refCount uint32
		ref      int32
		err      error
	}{
		{name: "nil writer", refCount: 1, ref: 0, err: writeCAVLCRefIndex(nil, 1, 0)},
		{name: "zero refs", refCount: 0, ref: 0, err: writeCAVLCRefIndex(&bw, 0, 0)},
		{name: "negative ref", refCount: 2, ref: -1, err: writeCAVLCRefIndex(&bw, 2, -1)},
		{name: "ref equals count", refCount: 2, ref: 2, err: writeCAVLCRefIndex(&bw, 2, 2)},
	}
	for _, tt := range tests {
		if tt.err != ErrInvalidData {
			t.Fatalf("%s err = %v, want ErrInvalidData", tt.name, tt.err)
		}
	}
}

func TestWriteCAVLCMVDRoundTripsThroughReader(t *testing.T) {
	tests := [][2]int32{
		{},
		{1, -1},
		{-3, 2},
		{127, -128},
	}
	for _, want := range tests {
		var bw BitWriter
		if err := writeCAVLCMVD(&bw, want); err != nil {
			t.Fatalf("write mvd %v failed: %v", want, err)
		}

		gb := newBitReader(bw.Bytes())
		var got [2]int32
		if err := readCAVLCMVD(&gb, &got); err != nil {
			t.Fatalf("read mvd %v failed: %v", want, err)
		}
		if got != want {
			t.Fatalf("mvd round trip = %v, want %v", got, want)
		}
		if gb.bitPos != bw.BitLen() {
			t.Fatalf("mvd consumed %d bits, want %d", gb.bitPos, bw.BitLen())
		}
	}
}

func TestWriteCAVLCMVDRejectsInvalid(t *testing.T) {
	if err := writeCAVLCMVD(nil, [2]int32{}); err != ErrInvalidData {
		t.Fatalf("nil mvd writer err = %v, want ErrInvalidData", err)
	}
	var bw BitWriter
	if err := writeCAVLCMVD(&bw, [2]int32{-2147483648, 0}); err != ErrInvalidData {
		t.Fatalf("invalid mvd x err = %v, want ErrInvalidData", err)
	}
	if err := writeCAVLCMVD(&bw, [2]int32{0, -2147483648}); err != ErrInvalidData {
		t.Fatalf("invalid mvd y err = %v, want ErrInvalidData", err)
	}
}

func TestWriteCAVLCInterPNoResidualMacroblockRoundTripsThroughDecoder(t *testing.T) {
	tests := []struct {
		name     string
		mb       cavlcInterMacroblockSyntax
		refCount [2]uint32
	}{
		{
			name: "p16x16",
			mb: func() cavlcInterMacroblockSyntax {
				mb := cavlcInterMacroblockSyntax{
					cavlcMacroblockSyntax: cavlcMacroblockSyntax{MBType: MBType16x16 | MBTypeP0L0, PartitionCount: 1},
					Ref:                   [2][4]int32{{1}},
				}
				mb.MVD[0][0] = [2]int32{2, -1}
				return mb
			}(),
			refCount: [2]uint32{2, 0},
		},
		{
			name: "p16x8",
			mb: func() cavlcInterMacroblockSyntax {
				mb := cavlcInterMacroblockSyntax{
					cavlcMacroblockSyntax: cavlcMacroblockSyntax{MBType: MBType16x8 | MBTypeP0L0 | MBTypeP1L0, PartitionCount: 2},
					Ref:                   [2][4]int32{{2, 0}},
				}
				mb.MVD[0][0] = [2]int32{1, -2}
				mb.MVD[0][8] = [2]int32{-3, 4}
				return mb
			}(),
			refCount: [2]uint32{3, 0},
		},
		{
			name: "p8x16",
			mb: func() cavlcInterMacroblockSyntax {
				mb := cavlcInterMacroblockSyntax{
					cavlcMacroblockSyntax: cavlcMacroblockSyntax{MBType: MBType8x16 | MBTypeP0L0 | MBTypeP1L0, PartitionCount: 2},
					Ref:                   [2][4]int32{{0, 2}},
				}
				mb.MVD[0][0] = [2]int32{-1, 1}
				mb.MVD[0][4] = [2]int32{3, -4}
				return mb
			}(),
			refCount: [2]uint32{3, 0},
		},
		{
			name: "p8x8 mixed sub partitions",
			mb: func() cavlcInterMacroblockSyntax {
				mb := cavlcInterMacroblockSyntax{
					cavlcMacroblockSyntax: cavlcMacroblockSyntax{MBType: MBType8x8 | MBTypeP0L0 | MBTypeP1L0, PartitionCount: 4},
					Ref:                   [2][4]int32{{0, 1, 2, 0}},
				}
				for i, info := range h264PSubMBTypeInfo {
					mb.SubMBType[i] = info.Type
					mb.SubPartitionCount[i] = info.PartitionCount
				}
				mb.MVD[0][0] = [2]int32{1, 0}
				mb.MVD[0][4] = [2]int32{0, 1}
				mb.MVD[0][6] = [2]int32{0, -1}
				mb.MVD[0][8] = [2]int32{2, 0}
				mb.MVD[0][9] = [2]int32{-2, 0}
				mb.MVD[0][12] = [2]int32{1, 1}
				mb.MVD[0][13] = [2]int32{-1, 1}
				mb.MVD[0][14] = [2]int32{1, -1}
				mb.MVD[0][15] = [2]int32{-1, -1}
				return mb
			}(),
			refCount: [2]uint32{3, 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var bw BitWriter
			if err := writeCAVLCInterPNoResidualMacroblock(&bw, tt.mb, tt.refCount, true); err != nil {
				t.Fatalf("write P no-residual macroblock failed: %v", err)
			}

			pps := cavlcFlatQMulPPS()
			sps := &SPS{BitDepthLuma: 8, ChromaFormatIDC: 1}
			var ctx cavlcResidualContext
			gb := newBitReader(bw.Bytes())
			got, err := ctx.decodeCAVLCInterPMacroblock(&gb, pps, sps, 24, tt.refCount, true)
			if err != nil {
				t.Fatalf("decode written P no-residual macroblock failed: %v", err)
			}
			assertCAVLCInterPMacroblockSyntax(t, got, tt.mb)
			if gb.bitPos != bw.BitLen() {
				t.Fatalf("decoded consumed %d bits, want %d", gb.bitPos, bw.BitLen())
			}
		})
	}
}

func TestWriteCAVLCInterPNoResidualMacroblockRejectsInvalid(t *testing.T) {
	valid := cavlcInterMacroblockSyntax{
		cavlcMacroblockSyntax: cavlcMacroblockSyntax{MBType: MBType16x16 | MBTypeP0L0, PartitionCount: 1},
		Ref:                   [2][4]int32{{0}},
	}
	var bw BitWriter
	if err := writeCAVLCInterPNoResidualMacroblock(nil, valid, [2]uint32{1, 0}, true); err != ErrInvalidData {
		t.Fatalf("nil writer err = %v, want ErrInvalidData", err)
	}
	intra := valid
	intra.MBType = MBTypeIntra4x4
	if err := writeCAVLCInterPNoResidualMacroblock(&bw, intra, [2]uint32{1, 0}, true); err != ErrUnsupported {
		t.Fatalf("intra err = %v, want ErrUnsupported", err)
	}
	residual := valid
	residual.CBP = 1
	if err := writeCAVLCInterPNoResidualMacroblock(&bw, residual, [2]uint32{1, 0}, true); err != ErrUnsupported {
		t.Fatalf("residual err = %v, want ErrUnsupported", err)
	}
	badRef := valid
	badRef.Ref[0][0] = 1
	if err := writeCAVLCInterPNoResidualMacroblock(&bw, badRef, [2]uint32{1, 0}, true); err != ErrInvalidData {
		t.Fatalf("bad ref err = %v, want ErrInvalidData", err)
	}
	badSub := cavlcInterMacroblockSyntax{
		cavlcMacroblockSyntax: cavlcMacroblockSyntax{MBType: MBType8x8 | MBTypeP0L0 | MBTypeP1L0, PartitionCount: 4},
		SubMBType:             [4]uint32{MBTypeIntra4x4},
	}
	if err := writeCAVLCInterPNoResidualMacroblock(&bw, badSub, [2]uint32{1, 0}, true); err != ErrInvalidData {
		t.Fatalf("bad sub err = %v, want ErrInvalidData", err)
	}
}

func TestWriteCAVLCInterPBoundedMacroblockRoundTripsResidualThroughDecoder(t *testing.T) {
	pps := cavlcFlatQMulPPS()
	sps := &SPS{BitDepthLuma: 8, ChromaFormatIDC: 1}
	mb := cavlcInterMacroblockSyntax{
		cavlcMacroblockSyntax: cavlcMacroblockSyntax{
			MBType:         MBType16x16 | MBTypeP0L0,
			PartitionCount: 1,
			CBP:            0x21,
		},
		Ref: [2][4]int32{{1}},
	}
	mb.MVD[0][0] = [2]int32{2, -1}

	chromaACPos := int(h264ZigzagScanCAVLC[1])
	var writer cavlcResidualContext
	writer.MB[0] = 1
	writer.MB[16] = -1
	writer.MB[256] = 1
	writer.MB[512] = -1
	writer.MB[256+chromaACPos] = 1
	writer.MB[512+chromaACPos] = -1
	fillCAVLCNonZero(&writer.NonZeroCountCache, int(h264Scan8[0]), 4, 4, 8, 9)
	fillCAVLCNonZero(&writer.NonZeroCountCache, int(h264Scan8[16]), 4, 4, 8, 9)
	fillCAVLCNonZero(&writer.NonZeroCountCache, int(h264Scan8[32]), 4, 4, 8, 9)

	var bw BitWriter
	cbpTable, err := writeCAVLCInterPBoundedMacroblock(&bw, &writer, pps, sps, mb, [2]uint32{2, 0}, 20, 23)
	if err != nil {
		t.Fatalf("write bounded P macroblock failed: %v", err)
	}
	if cbpTable != 0x1021 {
		t.Fatalf("writer cbpTable = %#x, want 0x1021", cbpTable)
	}

	var decoded cavlcResidualContext
	fillCAVLCNonZero(&decoded.NonZeroCountCache, int(h264Scan8[0]), 4, 4, 8, 9)
	fillCAVLCNonZero(&decoded.NonZeroCountCache, int(h264Scan8[16]), 4, 4, 8, 9)
	fillCAVLCNonZero(&decoded.NonZeroCountCache, int(h264Scan8[32]), 4, 4, 8, 9)
	gb := newBitReader(bw.Bytes())
	got, err := decoded.decodeCAVLCInterPMacroblock(&gb, pps, sps, 20, [2]uint32{2, 0}, false)
	if err != nil {
		t.Fatalf("decode written bounded P macroblock failed: %v", err)
	}
	if got.MBType != mb.MBType || got.PartitionCount != mb.PartitionCount || got.CBP != mb.CBP ||
		got.QScale != 23 || got.ChromaQP != ([2]uint8{23, 23}) || got.CBPTable != cbpTable {
		t.Fatalf("decoded mb = type %#x part %d cbp/q/chroma/cbpTable %#x/%d/%v/%#x, want type %#x part %d cbp/q/chroma/cbpTable %#x/23/[23 23]/%#x",
			got.MBType, got.PartitionCount, got.CBP, got.QScale, got.ChromaQP, got.CBPTable,
			mb.MBType, mb.PartitionCount, mb.CBP, cbpTable)
	}
	if got.Ref[0][0] != 1 || got.MVD[0][0] != ([2]int32{2, -1}) {
		t.Fatalf("decoded motion = ref %d mvd %v, want ref 1 mvd [2 -1]", got.Ref[0][0], got.MVD[0][0])
	}
	if decoded.MB[0] != 1 || decoded.MB[16] != -1 ||
		decoded.MB[256] != 1 || decoded.MB[512] != -1 ||
		decoded.MB[256+chromaACPos] != 1 || decoded.MB[512+chromaACPos] != -1 {
		t.Fatalf("decoded residual coeffs mismatch")
	}
	if decoded.NonZeroCountCache != writer.NonZeroCountCache {
		t.Fatalf("decoded nnz cache differs from writer cache")
	}
	if gb.bitPos != bw.BitLen() {
		t.Fatalf("decoded consumed %d bits, want %d", gb.bitPos, bw.BitLen())
	}
}

func TestWriteCAVLCInterPBoundedMacroblockRejectsInvalid(t *testing.T) {
	pps := cavlcFlatQMulPPS()
	sps := &SPS{BitDepthLuma: 8, ChromaFormatIDC: 1}
	valid := cavlcInterMacroblockSyntax{
		cavlcMacroblockSyntax: cavlcMacroblockSyntax{MBType: MBType16x16 | MBTypeP0L0, PartitionCount: 1, CBP: 1},
		Ref:                   [2][4]int32{{0}},
	}
	var bw BitWriter
	var residual cavlcResidualContext
	for _, tt := range []struct {
		name string
		run  func() error
		want error
	}{
		{name: "nil writer", run: func() error {
			_, err := writeCAVLCInterPBoundedMacroblock(nil, &residual, pps, sps, valid, [2]uint32{1, 0}, 20, 20)
			return err
		}, want: ErrInvalidData},
		{name: "nil residual", run: func() error {
			_, err := writeCAVLCInterPBoundedMacroblock(&bw, nil, pps, sps, valid, [2]uint32{1, 0}, 20, 20)
			return err
		}, want: ErrInvalidData},
		{name: "nil pps", run: func() error {
			_, err := writeCAVLCInterPBoundedMacroblock(&bw, &residual, nil, sps, valid, [2]uint32{1, 0}, 20, 20)
			return err
		}, want: ErrInvalidData},
		{name: "nil sps", run: func() error {
			_, err := writeCAVLCInterPBoundedMacroblock(&bw, &residual, pps, nil, valid, [2]uint32{1, 0}, 20, 20)
			return err
		}, want: ErrInvalidData},
		{name: "intra", run: func() error {
			intra := valid
			intra.MBType = MBTypeIntra4x4
			_, err := writeCAVLCInterPBoundedMacroblock(&bw, &residual, pps, sps, intra, [2]uint32{1, 0}, 20, 20)
			return err
		}, want: ErrUnsupported},
		{name: "bad ref", run: func() error {
			bad := valid
			bad.Ref[0][0] = 1
			_, err := writeCAVLCInterPBoundedMacroblock(&bw, &residual, pps, sps, bad, [2]uint32{1, 0}, 20, 20)
			return err
		}, want: ErrInvalidData},
		{name: "unsupported residual bounds", run: func() error {
			_, err := writeCAVLCInterPBoundedMacroblock(&bw, &residual, pps, &SPS{BitDepthLuma: 8, ChromaFormatIDC: 2}, valid, [2]uint32{1, 0}, 20, 20)
			return err
		}, want: ErrUnsupported},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.run(); err != tt.want {
				t.Fatalf("write bounded P macroblock error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestWriteCAVLCInterBNoResidualMacroblockRoundTripsThroughDecoder(t *testing.T) {
	tests := []struct {
		name     string
		mb       cavlcInterMacroblockSyntax
		want     cavlcInterMacroblockSyntax
		refCount [2]uint32
	}{
		{
			name: "direct",
			mb: cavlcInterMacroblockSyntax{
				cavlcMacroblockSyntax: cavlcMacroblockSyntax{MBType: MBTypeDirect2 | MBTypeL0L1, PartitionCount: 1},
			},
			want: cavlcInterMacroblockSyntax{
				cavlcMacroblockSyntax: cavlcMacroblockSyntax{MBType: MBTypeDirect2 | MBTypeL0L1, PartitionCount: 1},
				Ref:                   [2][4]int32{{-1, -1, -1, -1}, {-1, -1, -1, -1}},
			},
			refCount: [2]uint32{1, 1},
		},
		{
			name: "b16x16 bi",
			mb: func() cavlcInterMacroblockSyntax {
				mb := cavlcInterMacroblockSyntax{
					cavlcMacroblockSyntax: cavlcMacroblockSyntax{MBType: MBType16x16 | MBTypeP0L0 | MBTypeP0L1, PartitionCount: 1},
					Ref:                   [2][4]int32{{1}, {0}},
				}
				mb.MVD[0][0] = [2]int32{2, -1}
				mb.MVD[1][0] = [2]int32{-2, 1}
				return mb
			}(),
			want: func() cavlcInterMacroblockSyntax {
				mb := cavlcInterMacroblockSyntax{
					cavlcMacroblockSyntax: cavlcMacroblockSyntax{MBType: MBType16x16 | MBTypeP0L0 | MBTypeP0L1, PartitionCount: 1},
					Ref:                   [2][4]int32{{1, -1, -1, -1}, {0, -1, -1, -1}},
				}
				mb.MVD[0][0] = [2]int32{2, -1}
				mb.MVD[1][0] = [2]int32{-2, 1}
				return mb
			}(),
			refCount: [2]uint32{2, 1},
		},
		{
			name: "b16x8 crossed lists",
			mb: func() cavlcInterMacroblockSyntax {
				mb := cavlcInterMacroblockSyntax{
					cavlcMacroblockSyntax: cavlcMacroblockSyntax{MBType: MBType16x8 | MBTypeP0L0 | MBTypeP1L1, PartitionCount: 2},
					Ref:                   [2][4]int32{{2}, {0, 1}},
				}
				mb.MVD[0][0] = [2]int32{1, 2}
				mb.MVD[1][8] = [2]int32{-1, -2}
				return mb
			}(),
			want: func() cavlcInterMacroblockSyntax {
				mb := cavlcInterMacroblockSyntax{
					cavlcMacroblockSyntax: cavlcMacroblockSyntax{MBType: MBType16x8 | MBTypeP0L0 | MBTypeP1L1, PartitionCount: 2},
					Ref:                   [2][4]int32{{2, -1, -1, -1}, {-1, 1, -1, -1}},
				}
				mb.MVD[0][0] = [2]int32{1, 2}
				mb.MVD[1][8] = [2]int32{-1, -2}
				return mb
			}(),
			refCount: [2]uint32{3, 2},
		},
		{
			name: "b8x8 mixed direct and explicit",
			mb: func() cavlcInterMacroblockSyntax {
				mb := cavlcInterMacroblockSyntax{
					cavlcMacroblockSyntax: cavlcMacroblockSyntax{MBType: MBType8x8 | MBTypeP0L0 | MBTypeP0L1 | MBTypeP1L0 | MBTypeP1L1, PartitionCount: 4},
					Ref:                   [2][4]int32{{0, 1, 0, 2}, {0, 0, 1, 1}},
					SubMBType:             [4]uint32{MBTypeDirect2, MBType16x16 | MBTypeP0L0, MBType16x16 | MBTypeP0L1, MBType16x16 | MBTypeP0L0 | MBTypeP0L1},
					SubPartitionCount:     [4]uint8{1, 1, 1, 1},
				}
				mb.MVD[0][4] = [2]int32{1, 0}
				mb.MVD[1][8] = [2]int32{0, -1}
				mb.MVD[0][12] = [2]int32{2, 1}
				mb.MVD[1][12] = [2]int32{-2, -1}
				return mb
			}(),
			want: func() cavlcInterMacroblockSyntax {
				mb := cavlcInterMacroblockSyntax{
					cavlcMacroblockSyntax: cavlcMacroblockSyntax{MBType: MBType8x8 | MBTypeP0L0 | MBTypeP0L1 | MBTypeP1L0 | MBTypeP1L1, PartitionCount: 4},
					Ref:                   [2][4]int32{{-1, 1, -1, 2}, {-1, -1, 1, 1}},
					SubMBType:             [4]uint32{MBTypeDirect2, MBType16x16 | MBTypeP0L0, MBType16x16 | MBTypeP0L1, MBType16x16 | MBTypeP0L0 | MBTypeP0L1},
					SubPartitionCount:     [4]uint8{1, 1, 1, 1},
				}
				mb.MVD[0][4] = [2]int32{1, 0}
				mb.MVD[1][8] = [2]int32{0, -1}
				mb.MVD[0][12] = [2]int32{2, 1}
				mb.MVD[1][12] = [2]int32{-2, -1}
				return mb
			}(),
			refCount: [2]uint32{3, 2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var bw BitWriter
			if err := writeCAVLCInterBNoResidualMacroblock(&bw, tt.mb, tt.refCount, true); err != nil {
				t.Fatalf("write B no-residual macroblock failed: %v", err)
			}

			pps := cavlcFlatQMulPPS()
			sps := &SPS{BitDepthLuma: 8, ChromaFormatIDC: 1, Direct8x8InferenceFlag: 1}
			var ctx cavlcResidualContext
			gb := newBitReader(bw.Bytes())
			got, err := ctx.decodeCAVLCInterBMacroblock(&gb, pps, sps, 24, tt.refCount, true)
			if err != nil {
				t.Fatalf("decode written B no-residual macroblock failed: %v", err)
			}
			assertCAVLCInterBMacroblockSyntax(t, got, tt.want)
			if gb.bitPos != bw.BitLen() {
				t.Fatalf("decoded consumed %d bits, want %d", gb.bitPos, bw.BitLen())
			}
		})
	}
}

func TestWriteCAVLCInterBNoResidualMacroblockRejectsInvalid(t *testing.T) {
	valid := cavlcInterMacroblockSyntax{
		cavlcMacroblockSyntax: cavlcMacroblockSyntax{MBType: MBType16x16 | MBTypeP0L0, PartitionCount: 1},
		Ref:                   [2][4]int32{{0}},
	}
	var bw BitWriter
	if err := writeCAVLCInterBNoResidualMacroblock(nil, valid, [2]uint32{1, 1}, true); err != ErrInvalidData {
		t.Fatalf("nil writer err = %v, want ErrInvalidData", err)
	}
	intra := valid
	intra.MBType = MBTypeIntra4x4
	if err := writeCAVLCInterBNoResidualMacroblock(&bw, intra, [2]uint32{1, 1}, true); err != ErrUnsupported {
		t.Fatalf("intra err = %v, want ErrUnsupported", err)
	}
	residual := valid
	residual.CBP = 1
	if err := writeCAVLCInterBNoResidualMacroblock(&bw, residual, [2]uint32{1, 1}, true); err != ErrUnsupported {
		t.Fatalf("residual err = %v, want ErrUnsupported", err)
	}
	badRef := valid
	badRef.Ref[0][0] = 1
	if err := writeCAVLCInterBNoResidualMacroblock(&bw, badRef, [2]uint32{1, 1}, true); err != ErrInvalidData {
		t.Fatalf("bad ref err = %v, want ErrInvalidData", err)
	}
	badSub := cavlcInterMacroblockSyntax{
		cavlcMacroblockSyntax: cavlcMacroblockSyntax{MBType: MBType8x8 | MBTypeP0L0 | MBTypeP0L1 | MBTypeP1L0 | MBTypeP1L1, PartitionCount: 4},
		SubMBType:             [4]uint32{MBTypeIntra4x4},
	}
	if err := writeCAVLCInterBNoResidualMacroblock(&bw, badSub, [2]uint32{1, 1}, true); err != ErrInvalidData {
		t.Fatalf("bad sub err = %v, want ErrInvalidData", err)
	}
}

func assertCAVLCInterPMacroblockSyntax(t *testing.T, got cavlcInterMacroblockSyntax, want cavlcInterMacroblockSyntax) {
	t.Helper()
	if got.MBType != want.MBType || got.PartitionCount != want.PartitionCount || got.CBP != 0 || got.QScale != 24 {
		t.Fatalf("macroblock = type %#x partitions %d cbp/qscale %d/%d, want type %#x partitions %d cbp/qscale 0/24",
			got.MBType, got.PartitionCount, got.CBP, got.QScale, want.MBType, want.PartitionCount)
	}
	if got.SubMBType != want.SubMBType || got.SubPartitionCount != want.SubPartitionCount {
		t.Fatalf("sub macroblocks = %#v/%#v, want %#v/%#v",
			got.SubMBType, got.SubPartitionCount, want.SubMBType, want.SubPartitionCount)
	}
	if got.Ref[0] != want.Ref[0] {
		t.Fatalf("refs = %v, want %v", got.Ref[0], want.Ref[0])
	}
	if got.MVD[0] != want.MVD[0] {
		t.Fatalf("mvd = %v, want %v", got.MVD[0], want.MVD[0])
	}
}

func assertCAVLCInterBMacroblockSyntax(t *testing.T, got cavlcInterMacroblockSyntax, want cavlcInterMacroblockSyntax) {
	t.Helper()
	if got.MBType != want.MBType || got.PartitionCount != want.PartitionCount || got.CBP != 0 || got.QScale != 24 {
		t.Fatalf("macroblock = type %#x partitions %d cbp/qscale %d/%d, want type %#x partitions %d cbp/qscale 0/24",
			got.MBType, got.PartitionCount, got.CBP, got.QScale, want.MBType, want.PartitionCount)
	}
	if got.SubMBType != want.SubMBType || got.SubPartitionCount != want.SubPartitionCount {
		t.Fatalf("sub macroblocks = %#v/%#v, want %#v/%#v",
			got.SubMBType, got.SubPartitionCount, want.SubMBType, want.SubPartitionCount)
	}
	if got.Ref != want.Ref {
		t.Fatalf("refs = %v, want %v", got.Ref, want.Ref)
	}
	if got.MVD != want.MVD {
		t.Fatalf("mvd = %v, want %v", got.MVD, want.MVD)
	}
}

func TestWriteCAVLCPSubMBTypeRoundTripsThroughReader(t *testing.T) {
	for raw, info := range h264PSubMBTypeInfo {
		var bw BitWriter
		if err := writeCAVLCPSubMBType(&bw, info); err != nil {
			t.Fatalf("write P sub mb type %d failed: %v", raw, err)
		}

		gb := newBitReader(bw.Bytes())
		gotRaw, err := gb.readUEGolomb31()
		if err != nil {
			t.Fatalf("read P sub mb type %d failed: %v", raw, err)
		}
		if gotRaw != uint32(raw) {
			t.Fatalf("P sub mb type raw = %d, want %d", gotRaw, raw)
		}
		if gotInfo := h264PSubMBTypeInfo[gotRaw]; gotInfo != info {
			t.Fatalf("P sub mb type info = %#v, want %#v", gotInfo, info)
		}
		if gb.bitPos != bw.BitLen() {
			t.Fatalf("P sub mb type consumed %d bits, want %d", gb.bitPos, bw.BitLen())
		}
	}
}

func TestWriteCAVLCBSubMBTypeRoundTripsThroughReader(t *testing.T) {
	for raw, info := range h264BSubMBTypeInfo {
		var bw BitWriter
		if err := writeCAVLCBSubMBType(&bw, info); err != nil {
			t.Fatalf("write B sub mb type %d failed: %v", raw, err)
		}

		gb := newBitReader(bw.Bytes())
		gotRaw, err := gb.readUEGolomb31()
		if err != nil {
			t.Fatalf("read B sub mb type %d failed: %v", raw, err)
		}
		if gotRaw != uint32(raw) {
			t.Fatalf("B sub mb type raw = %d, want %d", gotRaw, raw)
		}
		if gotRaw >= uint32(len(h264BSubMBTypeInfo)) {
			t.Fatalf("B sub mb type raw = %d outside table", gotRaw)
		}
		if gotInfo := h264BSubMBTypeInfo[gotRaw]; gotInfo != info {
			t.Fatalf("B sub mb type info = %#v, want %#v", gotInfo, info)
		}
		if gb.bitPos != bw.BitLen() {
			t.Fatalf("B sub mb type consumed %d bits, want %d", gb.bitPos, bw.BitLen())
		}
	}
}

func TestWriteCAVLCSubMBTypeRejectsInvalid(t *testing.T) {
	var bw BitWriter
	if err := writeCAVLCPSubMBType(nil, h264PSubMBTypeInfo[0]); err != ErrInvalidData {
		t.Fatalf("nil P sub mb writer err = %v, want ErrInvalidData", err)
	}
	if err := writeCAVLCBSubMBType(nil, h264BSubMBTypeInfo[0]); err != ErrInvalidData {
		t.Fatalf("nil B sub mb writer err = %v, want ErrInvalidData", err)
	}

	invalid := PMBInfo{Type: MBTypeIntra4x4, PartitionCount: 9}
	if err := writeCAVLCPSubMBType(&bw, invalid); err != ErrInvalidData {
		t.Fatalf("invalid P sub mb type err = %v, want ErrInvalidData", err)
	}
	if err := writeCAVLCBSubMBType(&bw, invalid); err != ErrInvalidData {
		t.Fatalf("invalid B sub mb type err = %v, want ErrInvalidData", err)
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
