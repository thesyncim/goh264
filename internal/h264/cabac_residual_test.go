// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

import "testing"

func TestCABACCBFContext(t *testing.T) {
	var ctx cavlcResidualContext
	ctx.NonZeroCountCache[h264Scan8[0]-1] = 1
	if got, err := ctx.cabacCBFContext(2, 0, 16, false, 0, 0); err != nil || got != 94 {
		t.Fatalf("non-dc cbf ctx = %d err %v, want 94 nil", got, err)
	}

	if got, err := ctx.cabacCBFContext(0, lumaDCBlockIndex+2, 16, true, 0x100<<2, 0x100<<2); err != nil || got != 88 {
		t.Fatalf("luma dc cbf ctx = %d err %v, want 88 nil", got, err)
	}

	if got, err := ctx.cabacCBFContext(3, chromaDCBlockIndex+1, 4, true, 1<<7, 0); err != nil || got != 98 {
		t.Fatalf("chroma dc cbf ctx = %d err %v, want 98 nil", got, err)
	}
}

func TestDecodeCABACResidualDCOneCoeff(t *testing.T) {
	var ctx cavlcResidualContext
	var block [16]int32
	src := &scriptedCABACSource{
		bits:  []int{1, 1, 1, 0},
		signs: []int32{1},
	}

	result, err := ctx.decodeCABACResidualDC(src, block[:], 0, lumaDCBlockIndex, cabacIdentityScan(16), 4, 0, 0, false, false)
	if err != nil {
		t.Fatalf("decode dc residual failed: %v", err)
	}
	if !result.Coded || result.CoeffCount != 1 || result.CBPTableBits != 0x100 {
		t.Fatalf("result = %+v, want coded count=1 cbp=0x100", result)
	}
	if block[0] != 1 {
		t.Fatalf("block[0] = %d, want 1", block[0])
	}
	if ctx.NonZeroCountCache[h264Scan8[lumaDCBlockIndex]] != 1 {
		t.Fatalf("dc nnz = %d, want 1", ctx.NonZeroCountCache[h264Scan8[lumaDCBlockIndex]])
	}
	wantIndexes(t, src, []int{85, 105, 166, 228})
}

func TestDecodeCABACResidualNonDCQuantizedCoeff(t *testing.T) {
	var ctx cavlcResidualContext
	var block [16]int32
	var qmul [16]uint32
	for i := range qmul {
		qmul[i] = 256
	}
	src := &scriptedCABACSource{bits: []int{1, 0, 1, 1, 0}}

	result, err := ctx.decodeCABACResidualNonDC(src, block[:], 2, 0, cabacIdentityScan(16), qmul[:], 4, 0, 0, false, false)
	if err != nil {
		t.Fatalf("decode non-dc residual failed: %v", err)
	}
	if !result.Coded || result.CoeffCount != 1 || result.CBPTableBits != 0 {
		t.Fatalf("result = %+v, want coded count=1 cbp=0", result)
	}
	if block[1] != -4 {
		t.Fatalf("block[1] = %d, want -4", block[1])
	}
	if ctx.NonZeroCountCache[h264Scan8[0]] != 1 {
		t.Fatalf("nnz = %d, want 1", ctx.NonZeroCountCache[h264Scan8[0]])
	}
	wantIndexes(t, src, []int{93, 134, 135, 196, 248})
}

func TestDecodeCABACResidualNonDCTypedDCTElemWidth(t *testing.T) {
	for _, tt := range []struct {
		name      string
		narrowDCT bool
		want      int32
	}{
		{name: "8-bit-dctelem", narrowDCT: true, want: 0},
		{name: "high-bit-depth-dctelem", narrowDCT: false, want: 65536},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var ctx cavlcResidualContext
			var block [16]int32
			var qmul [16]uint32
			for i := range qmul {
				qmul[i] = 1 << 22
			}
			src := &scriptedCABACSource{
				bits:  []int{1, 0, 1, 1, 0},
				signs: []int32{1 << 22},
			}

			result, err := ctx.decodeCABACResidualNonDCTyped(src, block[:], 2, 0, cabacIdentityScan(16), qmul[:], 4, 0, 0, false, false, tt.narrowDCT)
			if err != nil {
				t.Fatalf("decode non-dc typed residual failed: %v", err)
			}
			if !result.Coded || result.CoeffCount != 1 {
				t.Fatalf("result = %+v, want coded count=1", result)
			}
			if block[1] != tt.want {
				t.Fatalf("block[1] = %d, want %d", block[1], tt.want)
			}
			if ctx.NonZeroCountCache[h264Scan8[0]] != 1 {
				t.Fatalf("nnz = %d, want 1", ctx.NonZeroCountCache[h264Scan8[0]])
			}
			wantIndexes(t, src, []int{93, 134, 135, 196, 248})
		})
	}
}

func TestDecodeCABACResidualZeroCBF(t *testing.T) {
	var ctx cavlcResidualContext
	ctx.NonZeroCountCache[h264Scan8[4]] = 9
	var block [16]int32
	var qmul [16]uint32
	src := &scriptedCABACSource{bits: []int{0}}

	result, err := ctx.decodeCABACResidualNonDC(src, block[:], 2, 4, cabacIdentityScan(16), qmul[:], 16, 0, 0, false, false)
	if err != nil {
		t.Fatalf("decode zero cbf failed: %v", err)
	}
	if result.Coded || result.CoeffCount != 0 {
		t.Fatalf("result = %+v, want uncoded", result)
	}
	if ctx.NonZeroCountCache[h264Scan8[4]] != 0 {
		t.Fatalf("nnz = %d, want 0", ctx.NonZeroCountCache[h264Scan8[4]])
	}
	wantIndexes(t, src, []int{93})
}

func TestDecodeCABACResidual8x8SkipsCBFWhenNotChroma444(t *testing.T) {
	var ctx cavlcResidualContext
	var block [64]int32
	var qmul [64]uint32
	for i := range qmul {
		qmul[i] = 64
	}
	src := &scriptedCABACSource{bits: []int{1, 1, 0}}

	result, err := ctx.decodeCABACResidualNonDC(src, block[:], 5, 0, cabacIdentityScan(64), qmul[:], 64, 0, 0, false, false)
	if err != nil {
		t.Fatalf("decode 8x8 residual failed: %v", err)
	}
	if !result.Coded || result.CoeffCount != 1 {
		t.Fatalf("result = %+v, want coded count=1", result)
	}
	if block[0] != -1 {
		t.Fatalf("block[0] = %d, want -1", block[0])
	}
	start := int(h264Scan8[0])
	for _, off := range []int{0, 1, 8, 9} {
		if ctx.NonZeroCountCache[start+off] != 1 {
			t.Fatalf("nnz[%d] = %d, want 1", start+off, ctx.NonZeroCountCache[start+off])
		}
	}
	wantIndexes(t, src, []int{402, 417, 427})
}

func TestDecodeCABACResidualInternalDecoderMatchesGeneric(t *testing.T) {
	buf := make([]byte, 8192)
	for i := range buf {
		buf[i] = byte(i*97 + 31)
	}
	fastContext, err := initCABACDecoder(buf)
	if err != nil {
		t.Fatal(err)
	}
	oracleContext := fastContext
	fastState, err := initH264CABACStates(PictureTypeP, 1, 27, 8)
	if err != nil {
		t.Fatal(err)
	}
	oracleState := fastState
	fast := &cabacSyntaxDecoder{cabac: &fastContext, state: &fastState}
	oracle := &cabacSyntaxDecoder{cabac: &oracleContext, state: &oracleState}
	var fastResidual, oracleResidual cavlcResidualContext
	scan := cabacIdentityScan(16)
	qmul := make([]uint32, 16)
	for i := range qmul {
		qmul[i] = uint32(32 + i*7)
	}

	for i := 0; i < 128; i++ {
		var gotBlock, wantBlock [16]int32
		cat := i % 5
		n := i % 16
		mbField := i&1 != 0
		got, gotErr := decodeCABACResidualInternalDecoder(&fastResidual, fast, gotBlock[:], cat, n, scan, qmul, 16, false, mbField, false, false)
		want, wantErr := decodeCABACResidualInternalSource(&oracleResidual, cabacSyntaxSource(oracle), wantBlock[:], cat, n, scan, qmul, 16, false, mbField, false, false)
		if got != want || gotBlock != wantBlock || fastResidual != oracleResidual || (gotErr != nil) != (wantErr != nil) {
			t.Fatalf("step %d result diverged: got (%+v,%v), want (%+v,%v)", i, got, gotErr, want, wantErr)
		}
		if fastContext.low != oracleContext.low || fastContext.rng != oracleContext.rng || fastContext.bytestream != oracleContext.bytestream || fastState != oracleState {
			t.Fatalf("state diverged at step %d", i)
		}
		if gotErr != nil {
			break
		}
	}
}

func BenchmarkDecodeCABACResidualLevels4x4(b *testing.B) {
	buf := make([]byte, 32768)
	for i := range buf {
		buf[i] = byte(i*97 + 31)
	}
	initialContext, err := initCABACDecoder(buf)
	if err != nil {
		b.Fatal(err)
	}
	initialState, err := initH264CABACStates(PictureTypeP, 1, 27, 8)
	if err != nil {
		b.Fatal(err)
	}
	var index [64]uint8
	index[0], index[1], index[2], index[3] = 0, 3, 7, 12
	scan := cabacIdentityScan(16)
	qmul := make([]uint32, 16)
	for i := range qmul {
		qmul[i] = uint32(32 + i*7)
	}
	var block [16]int32
	const blocksPerIteration = 256
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		ctx := initialContext
		state := initialState
		decoder := cabacSyntaxDecoder{cabac: &ctx, state: &state}
		for range blocksPerIteration {
			if err := decodeCABACResidualLevelsDecoder(&decoder, block[:], scan, qmul, &index, 4, 227, false, false, false); err != nil {
				b.Fatal(err)
			}
		}
	}
}

func cabacIdentityScan(n int) []uint8 {
	scan := make([]uint8, n)
	for i := range scan {
		scan[i] = uint8(i)
	}
	return scan
}
