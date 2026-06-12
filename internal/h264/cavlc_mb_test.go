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

func TestWriteCAVLCMBTypeRoundTripsThroughDecoder(t *testing.T) {
	t.Run("p inter table", func(t *testing.T) {
		for _, info := range h264PMBTypeInfo {
			assertCAVLCMBTypeWriteRoundTrip(t, PictureTypeP, PictureTypeP, cavlcMacroblockSyntax{
				MBType:         info.Type,
				PartitionCount: info.PartitionCount,
			})
		}
	})
	t.Run("b inter table", func(t *testing.T) {
		for _, info := range h264BMBTypeInfo {
			assertCAVLCMBTypeWriteRoundTrip(t, PictureTypeB, PictureTypeB, cavlcMacroblockSyntax{
				MBType:         info.Type,
				PartitionCount: info.PartitionCount,
			})
		}
	})
	t.Run("i table", func(t *testing.T) {
		for _, info := range h264IMBTypeInfo {
			assertCAVLCMBTypeWriteRoundTrip(t, PictureTypeI, PictureTypeI, cavlcMacroblockSyntax{
				MBType:             info.Type,
				Intra16x16PredMode: info.PredMode,
				CBP:                int(info.CBP),
			})
		}
	})
	t.Run("p intra fallback", func(t *testing.T) {
		assertCAVLCMBTypeWriteRoundTrip(t, PictureTypeP, PictureTypeP, cavlcMacroblockSyntax{
			MBType:             MBTypeIntra16x16,
			Intra16x16PredMode: 3,
			CBP:                47,
		})
	})
	t.Run("b intra fallback", func(t *testing.T) {
		assertCAVLCMBTypeWriteRoundTrip(t, PictureTypeB, PictureTypeB, cavlcMacroblockSyntax{
			MBType:             MBTypeIntraPCM,
			Intra16x16PredMode: -1,
			CBP:                -1,
		})
	})
	t.Run("si adjusted intra", func(t *testing.T) {
		assertCAVLCMBTypeWriteRoundTrip(t, PictureTypeSI, PictureTypeI, cavlcMacroblockSyntax{
			MBType:             MBTypeIntra16x16,
			Intra16x16PredMode: 2,
			CBP:                0,
		})
	})
}

func assertCAVLCMBTypeWriteRoundTrip(t *testing.T, sliceType int32, sliceTypeNoS int32, mb cavlcMacroblockSyntax) {
	t.Helper()
	var bw BitWriter
	if err := writeCAVLCMBType(&bw, sliceType, sliceTypeNoS, mb); err != nil {
		t.Fatalf("write mb type %+v failed: %v", mb, err)
	}
	gb := newBitReader(bw.Bytes())
	got, err := decodeCAVLCMBType(&gb, sliceType, sliceTypeNoS)
	if err != nil {
		t.Fatalf("decode written mb type %+v failed: %v", mb, err)
	}
	if got.MBType != mb.MBType || got.PartitionCount != mb.PartitionCount ||
		got.Intra16x16PredMode != mb.Intra16x16PredMode || got.CBP != mb.CBP {
		t.Fatalf("mb = type %#x part %d pred %d cbp %d, want type %#x part %d pred %d cbp %d",
			got.MBType, got.PartitionCount, got.Intra16x16PredMode, got.CBP,
			mb.MBType, mb.PartitionCount, mb.Intra16x16PredMode, mb.CBP)
	}
	if gb.bitPos != bw.BitLen() {
		t.Fatalf("decoded consumed %d bits, want %d", gb.bitPos, bw.BitLen())
	}
}

func TestWriteCAVLCMBTypeRejectsInvalid(t *testing.T) {
	var bw BitWriter
	for _, tt := range []struct {
		name string
		err  error
	}{
		{name: "nil writer", err: writeCAVLCMBType(nil, PictureTypeP, PictureTypeP, cavlcMacroblockSyntax{MBType: MBType16x16 | MBTypeP0L0, PartitionCount: 1})},
		{name: "bad slice", err: writeCAVLCMBType(&bw, PictureTypeSP, PictureTypeSP, cavlcMacroblockSyntax{MBType: MBTypeIntra4x4, CBP: -1})},
		{name: "bad p partition", err: writeCAVLCMBType(&bw, PictureTypeP, PictureTypeP, cavlcMacroblockSyntax{MBType: MBType16x16 | MBTypeP0L0, PartitionCount: 2})},
		{name: "bad intra cbp", err: writeCAVLCMBType(&bw, PictureTypeI, PictureTypeI, cavlcMacroblockSyntax{MBType: MBTypeIntra16x16, Intra16x16PredMode: 0, CBP: 7})},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err != ErrInvalidData {
				t.Fatalf("write mb type error = %v, want ErrInvalidData", tt.err)
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

func TestWriteCAVLCCBPRoundTripsThroughDecoder(t *testing.T) {
	cases := []struct {
		name         string
		mbType       uint32
		decodeChroma bool
		maxCBP       int
	}{
		{name: "intra chroma", mbType: MBTypeIntra4x4, decodeChroma: true, maxCBP: 47},
		{name: "inter chroma", mbType: MBType16x16 | MBTypeP0L0, decodeChroma: true, maxCBP: 47},
		{name: "intra gray", mbType: MBTypeIntra4x4, decodeChroma: false, maxCBP: 15},
		{name: "inter gray", mbType: MBType16x16 | MBTypeP0L0, decodeChroma: false, maxCBP: 15},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for cbp := 0; cbp <= tc.maxCBP; cbp++ {
				var bw BitWriter
				if err := writeCAVLCCBP(&bw, tc.mbType, tc.decodeChroma, cbp); err != nil {
					t.Fatalf("write cbp %d failed: %v", cbp, err)
				}
				gb := newBitReader(bw.Bytes())
				got, err := decodeCAVLCCBP(&gb, tc.mbType, tc.decodeChroma, -1)
				if err != nil {
					t.Fatalf("decode written cbp %d failed: %v", cbp, err)
				}
				if got != cbp {
					t.Fatalf("decoded cbp = %d, want %d", got, cbp)
				}
				if gb.bitPos != bw.BitLen() {
					t.Fatalf("decoded cbp %d consumed %d bits, want %d", cbp, gb.bitPos, bw.BitLen())
				}
			}
		})
	}
}

func TestWriteCAVLCCBPIntra16x16WritesNoBits(t *testing.T) {
	for _, tt := range []struct {
		name         string
		decodeChroma bool
		cbp          int
	}{
		{name: "chroma table value", decodeChroma: true, cbp: 47},
		{name: "gray luma value", decodeChroma: false, cbp: 15},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var bw BitWriter
			if err := writeCAVLCCBP(&bw, MBTypeIntra16x16, tt.decodeChroma, tt.cbp); err != nil {
				t.Fatalf("write intra16 cbp: %v", err)
			}
			if bw.BitLen() != 0 {
				t.Fatalf("intra16 cbp wrote %d bits, want 0", bw.BitLen())
			}
			gb := newBitReader(bw.Bytes())
			got, err := decodeCAVLCCBP(&gb, MBTypeIntra16x16, tt.decodeChroma, tt.cbp)
			if err != nil {
				t.Fatalf("decode intra16 cbp: %v", err)
			}
			if got != tt.cbp {
				t.Fatalf("decoded cbp = %d, want %d", got, tt.cbp)
			}
		})
	}
}

func TestWriteCAVLCCBPRejectsInvalid(t *testing.T) {
	var bw BitWriter
	for _, tt := range []struct {
		name         string
		mbType       uint32
		decodeChroma bool
		cbp          int
		err          error
	}{
		{name: "nil writer", mbType: MBTypeIntra4x4, decodeChroma: true, cbp: 0, err: writeCAVLCCBP(nil, MBTypeIntra4x4, true, 0)},
		{name: "negative cbp", mbType: MBTypeIntra4x4, decodeChroma: true, cbp: -1, err: writeCAVLCCBP(&bw, MBTypeIntra4x4, true, -1)},
		{name: "chroma cbp too large", mbType: MBType16x16 | MBTypeP0L0, decodeChroma: true, cbp: 48, err: writeCAVLCCBP(&bw, MBType16x16|MBTypeP0L0, true, 48)},
		{name: "gray cbp too large", mbType: MBType16x16 | MBTypeP0L0, decodeChroma: false, cbp: 16, err: writeCAVLCCBP(&bw, MBType16x16|MBTypeP0L0, false, 16)},
		{name: "intra16 gray cbp too large", mbType: MBTypeIntra16x16, decodeChroma: false, cbp: 16, err: writeCAVLCCBP(&bw, MBTypeIntra16x16, false, 16)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err != ErrInvalidData {
				t.Fatalf("write cbp error = %v, want ErrInvalidData", tt.err)
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

func TestWriteCAVLCDQuantForQScaleRoundTripsThroughDecoder(t *testing.T) {
	for _, tt := range []struct {
		name       string
		qscale     int
		nextQScale int
		maxQP      int32
		wantDelta  int32
	}{
		{name: "same", qscale: 26, nextQScale: 26, maxQP: 51, wantDelta: 0},
		{name: "positive", qscale: 20, nextQScale: 23, maxQP: 51, wantDelta: 3},
		{name: "negative", qscale: 20, nextQScale: 17, maxQP: 51, wantDelta: -3},
		{name: "positive wraps", qscale: 51, nextQScale: 0, maxQP: 51, wantDelta: 1},
		{name: "negative wraps", qscale: 0, nextQScale: 51, maxQP: 51, wantDelta: -1},
		{name: "high bit depth range", qscale: 60, nextQScale: 63, maxQP: 63, wantDelta: 3},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var bw BitWriter
			if err := writeCAVLCDQuantForQScale(&bw, tt.qscale, tt.nextQScale, tt.maxQP); err != nil {
				t.Fatalf("write dquant: %v", err)
			}
			gb := newBitReader(bw.Bytes())
			delta, err := gb.readSEGolombLong()
			if err != nil {
				t.Fatalf("read written dquant: %v", err)
			}
			if delta != tt.wantDelta {
				t.Fatalf("delta = %d, want %d", delta, tt.wantDelta)
			}
			got, err := updateCAVLCQScale(tt.qscale, delta, tt.maxQP)
			if err != nil {
				t.Fatalf("update qscale: %v", err)
			}
			if got != tt.nextQScale {
				t.Fatalf("qscale = %d, want %d", got, tt.nextQScale)
			}
			if gb.bitPos != bw.BitLen() {
				t.Fatalf("decoded consumed %d bits, want %d", gb.bitPos, bw.BitLen())
			}
		})
	}
}

func TestWriteCAVLCDQuantForQScaleRejectsInvalid(t *testing.T) {
	var bw BitWriter
	for _, tt := range []struct {
		name string
		err  error
	}{
		{name: "nil writer", err: writeCAVLCDQuantForQScale(nil, 0, 0, 51)},
		{name: "negative qscale", err: writeCAVLCDQuantForQScale(&bw, -1, 0, 51)},
		{name: "negative next qscale", err: writeCAVLCDQuantForQScale(&bw, 0, -1, 51)},
		{name: "negative max qp", err: writeCAVLCDQuantForQScale(&bw, 0, 0, -1)},
		{name: "qscale above max", err: writeCAVLCDQuantForQScale(&bw, 52, 0, 51)},
		{name: "next qscale above max", err: writeCAVLCDQuantForQScale(&bw, 0, 52, 51)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err != ErrInvalidData {
				t.Fatalf("write dquant error = %v, want ErrInvalidData", tt.err)
			}
		})
	}
}

func TestWriteCAVLCInterResidualPayloadRoundTripsLumaThroughDecoder(t *testing.T) {
	pps := cavlcFlatQMulPPS()
	sps := &SPS{BitDepthLuma: 8, ChromaFormatIDC: 1}
	mbType := MBType16x16 | MBTypeP0L0
	var writer cavlcResidualContext
	writer.MB[0] = 1
	writer.MB[16] = -1
	fillCAVLCNonZero(&writer.NonZeroCountCache, int(h264Scan8[4]), 4, 4, 8, 9)

	var bw BitWriter
	cbpTable, err := writer.writeCAVLCInterResidualPayload(&bw, pps, sps, mbType, 1, 20, 23)
	if err != nil {
		t.Fatalf("write residual payload failed: %v", err)
	}
	if cbpTable != 0x1001 {
		t.Fatalf("writer cbpTable = %#x, want 0x1001", cbpTable)
	}
	if writer.NonZeroCountCache[h264Scan8[0]] != 1 || writer.NonZeroCountCache[h264Scan8[1]] != 1 {
		t.Fatalf("writer luma nnz = %d/%d, want 1/1", writer.NonZeroCountCache[h264Scan8[0]], writer.NonZeroCountCache[h264Scan8[1]])
	}
	for _, n := range []int{4, 5, 6, 7} {
		if writer.NonZeroCountCache[h264Scan8[n]] != 0 {
			t.Fatalf("writer cleared block%d nnz = %d, want 0", n, writer.NonZeroCountCache[h264Scan8[n]])
		}
	}

	var decoded cavlcResidualContext
	fillCAVLCNonZero(&decoded.NonZeroCountCache, int(h264Scan8[4]), 4, 4, 8, 9)
	gb := newBitReader(bw.Bytes())
	qscale, chromaQP, gotCBPTable, err := decoded.decodeCAVLCResidualPayload(&gb, pps, sps, mbType, 1, 20)
	if err != nil {
		t.Fatalf("decode written residual payload failed: %v", err)
	}
	if qscale != 23 || chromaQP != ([2]uint8{23, 23}) || gotCBPTable != cbpTable {
		t.Fatalf("decoded q/chroma/cbp = %d/%v/%#x, want 23/[23 23]/%#x", qscale, chromaQP, gotCBPTable, cbpTable)
	}
	if decoded.MB[0] != 1 || decoded.MB[16] != -1 {
		t.Fatalf("decoded luma coeffs = %d/%d, want 1/-1", decoded.MB[0], decoded.MB[16])
	}
	if decoded.NonZeroCountCache != writer.NonZeroCountCache {
		t.Fatalf("decoded nnz cache differs from writer cache")
	}
	if gb.bitPos != bw.BitLen() {
		t.Fatalf("decoded consumed %d bits, want %d", gb.bitPos, bw.BitLen())
	}
}

func TestWriteCAVLCInterResidualPayloadRoundTripsChromaDCThroughDecoder(t *testing.T) {
	pps := cavlcFlatQMulPPS()
	sps := &SPS{BitDepthLuma: 8, ChromaFormatIDC: 1}
	mbType := MBType16x16 | MBTypeP0L0
	var writer cavlcResidualContext
	writer.MB[256] = 1
	writer.MB[512] = -1

	var bw BitWriter
	cbpTable, err := writer.writeCAVLCInterResidualPayload(&bw, pps, sps, mbType, 0x10, 18, 18)
	if err != nil {
		t.Fatalf("write chroma residual payload failed: %v", err)
	}
	if cbpTable != 0x10 {
		t.Fatalf("writer cbpTable = %#x, want 0x10", cbpTable)
	}
	if writer.NonZeroCountCache[h264Scan8[chromaDCBlockIndex]] != 1 || writer.NonZeroCountCache[h264Scan8[chromaDCBlockIndex+1]] != 1 {
		t.Fatalf("writer chroma dc nnz = %d/%d, want 1/1",
			writer.NonZeroCountCache[h264Scan8[chromaDCBlockIndex]], writer.NonZeroCountCache[h264Scan8[chromaDCBlockIndex+1]])
	}

	var decoded cavlcResidualContext
	gb := newBitReader(bw.Bytes())
	qscale, chromaQP, gotCBPTable, err := decoded.decodeCAVLCResidualPayload(&gb, pps, sps, mbType, 0x10, 18)
	if err != nil {
		t.Fatalf("decode written chroma residual payload failed: %v", err)
	}
	if qscale != 18 || chromaQP != ([2]uint8{18, 18}) || gotCBPTable != cbpTable {
		t.Fatalf("decoded q/chroma/cbp = %d/%v/%#x, want 18/[18 18]/%#x", qscale, chromaQP, gotCBPTable, cbpTable)
	}
	if decoded.MB[256] != 1 || decoded.MB[512] != -1 {
		t.Fatalf("decoded chroma dc coeffs = %d/%d, want 1/-1", decoded.MB[256], decoded.MB[512])
	}
	if decoded.NonZeroCountCache != writer.NonZeroCountCache {
		t.Fatalf("decoded nnz cache differs from writer cache")
	}
	if gb.bitPos != bw.BitLen() {
		t.Fatalf("decoded consumed %d bits, want %d", gb.bitPos, bw.BitLen())
	}
}

func TestWriteCAVLCInterResidualPayloadRejectsInvalid(t *testing.T) {
	pps := cavlcFlatQMulPPS()
	sps := &SPS{BitDepthLuma: 8, ChromaFormatIDC: 1}
	var ctx cavlcResidualContext
	var bw BitWriter
	for _, tt := range []struct {
		name string
		run  func() error
		want error
	}{
		{name: "nil writer", run: func() error {
			_, err := ctx.writeCAVLCInterResidualPayload(nil, pps, sps, MBType16x16|MBTypeP0L0, 1, 20, 20)
			return err
		}, want: ErrInvalidData},
		{name: "nil pps", run: func() error {
			_, err := ctx.writeCAVLCInterResidualPayload(&bw, nil, sps, MBType16x16|MBTypeP0L0, 1, 20, 20)
			return err
		}, want: ErrInvalidData},
		{name: "nil sps", run: func() error {
			_, err := ctx.writeCAVLCInterResidualPayload(&bw, pps, nil, MBType16x16|MBTypeP0L0, 1, 20, 20)
			return err
		}, want: ErrInvalidData},
		{name: "negative cbp", run: func() error {
			_, err := ctx.writeCAVLCInterResidualPayload(&bw, pps, sps, MBType16x16|MBTypeP0L0, -1, 20, 20)
			return err
		}, want: ErrInvalidData},
		{name: "intra", run: func() error {
			_, err := ctx.writeCAVLCInterResidualPayload(&bw, pps, sps, MBTypeIntra4x4, 1, 20, 20)
			return err
		}, want: ErrUnsupported},
		{name: "8x8 dct", run: func() error {
			_, err := ctx.writeCAVLCInterResidualPayload(&bw, pps, sps, MBType16x16|MBTypeP0L0|MBType8x8DCT, 1, 20, 20)
			return err
		}, want: ErrUnsupported},
		{name: "unsupported chroma", run: func() error {
			_, err := ctx.writeCAVLCInterResidualPayload(&bw, pps, &SPS{BitDepthLuma: 8, ChromaFormatIDC: 2}, MBType16x16|MBTypeP0L0, 1, 20, 20)
			return err
		}, want: ErrUnsupported},
		{name: "non-flat luma qmul", run: func() error {
			nextPPS := cavlcFlatQMulPPS()
			nextPPS.Dequant4Buffer[3][20][0] = 65
			_, err := ctx.writeCAVLCInterResidualPayload(&bw, nextPPS, sps, MBType16x16|MBTypeP0L0, 1, 20, 20)
			return err
		}, want: ErrUnsupported},
		{name: "non-flat chroma qmul", run: func() error {
			nextPPS := cavlcFlatQMulPPS()
			nextPPS.Dequant4Buffer[4][20][0] = 65
			_, err := ctx.writeCAVLCInterResidualPayload(&bw, nextPPS, sps, MBType16x16|MBTypeP0L0, 0x20, 20, 20)
			return err
		}, want: ErrUnsupported},
		{name: "bad qscale", run: func() error {
			_, err := ctx.writeCAVLCInterResidualPayload(&bw, pps, sps, MBType16x16|MBTypeP0L0, 1, 52, 20)
			return err
		}, want: ErrInvalidData},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.run(); err != tt.want {
				t.Fatalf("write residual payload error = %v, want %v", err, tt.want)
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
