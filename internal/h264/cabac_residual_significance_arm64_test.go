// SPDX-License-Identifier: LGPL-2.1-or-later
//go:build !purego && arm64

package h264

import "testing"

func TestCABACResidualSignificanceFixedARM64MatchesScalar(t *testing.T) {
	buf := make([]byte, 32768)
	for i := range buf {
		buf[i] = byte(i*97 + 31)
	}

	for _, maxCoeff := range []int{16, 15} {
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

		for i := 0; i < 512; i++ {
			sigCtxBase := 105 + i%5
			lastCtxBase := 166 + i%5
			var gotIndex, wantIndex [64]uint8
			var gotCount, gotLast, wantCount, wantLast int
			if maxCoeff == 16 {
				gotCount, gotLast = decodeCABACResidualSignificance4x4Decoder(fast, &gotIndex, sigCtxBase, lastCtxBase)
				wantCount, wantLast = decodeCABACResidualSignificance4x4DecoderScalar(oracle, &wantIndex, sigCtxBase, lastCtxBase)
			} else {
				gotCount, gotLast = decodeCABACResidualSignificanceAC15Decoder(fast, &gotIndex, sigCtxBase, lastCtxBase)
				wantCount, wantLast = decodeCABACResidualSignificanceAC15DecoderScalar(oracle, &wantIndex, sigCtxBase, lastCtxBase)
			}
			if gotCount != wantCount || gotLast != wantLast || gotIndex != wantIndex {
				t.Fatalf("maxCoeff %d step %d result = (%d,%d,%v), want (%d,%d,%v)", maxCoeff, i, gotCount, gotLast, gotIndex, wantCount, wantLast, wantIndex)
			}
			if fastContext.low != oracleContext.low || fastContext.rng != oracleContext.rng || fastContext.bytestream != oracleContext.bytestream || fastState != oracleState {
				t.Fatalf("maxCoeff %d state diverged at step %d", maxCoeff, i)
			}
		}
	}
}

func BenchmarkCABACResidualSignificance4x4(b *testing.B) {
	buf := make([]byte, 8192)
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
	const blocksPerIteration = 256
	b.Run("Scalar", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			ctx := initialContext
			state := initialState
			decoder := cabacSyntaxDecoder{cabac: &ctx, state: &state}
			var index [64]uint8
			for i := range blocksPerIteration {
				decodeCABACResidualSignificance4x4DecoderScalar(&decoder, &index, 105+(i%5), 166+(i%5))
			}
		}
	})
	b.Run("ARM64", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			ctx := initialContext
			state := initialState
			decoder := cabacSyntaxDecoder{cabac: &ctx, state: &state}
			var index [64]uint8
			for i := range blocksPerIteration {
				decodeCABACResidualSignificance4x4Decoder(&decoder, &index, 105+(i%5), 166+(i%5))
			}
		}
	})
}
