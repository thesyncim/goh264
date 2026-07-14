// SPDX-License-Identifier: LGPL-2.1-or-later
//go:build !purego && arm64

package h264

import "testing"

func TestCABACIntra4x4PredModeARM64MatchesScalar(t *testing.T) {
	buf := make([]byte, 8192)
	for i := range buf {
		buf[i] = byte(i*73 + 19)
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

	for i := 0; i < 4096; i++ {
		predMode := (i*5 + 3) % 9
		got := decodeCABACMBIntra4x4PredModeDecoder(fast, predMode)
		want := decodeCABACMBIntra4x4PredModeDecoderScalar(oracle, predMode)
		if got != want {
			t.Fatalf("step %d mode = %d, want %d", i, got, want)
		}
		if fastContext.low != oracleContext.low || fastContext.rng != oracleContext.rng || fastContext.bytestream != oracleContext.bytestream || fastState != oracleState {
			t.Fatalf("state diverged at step %d", i)
		}
	}
}

func BenchmarkCABACIntra4x4PredMode(b *testing.B) {
	buf := make([]byte, 8192)
	for i := range buf {
		buf[i] = byte(i*73 + 19)
	}
	initialContext, err := initCABACDecoder(buf)
	if err != nil {
		b.Fatal(err)
	}
	initialState, err := initH264CABACStates(PictureTypeP, 1, 27, 8)
	if err != nil {
		b.Fatal(err)
	}
	const modesPerIteration = 256
	b.Run("Scalar", func(b *testing.B) {
		b.ReportAllocs()
		for range b.N {
			ctx := initialContext
			state := initialState
			decoder := cabacSyntaxDecoder{cabac: &ctx, state: &state}
			for i := range modesPerIteration {
				decodeCABACMBIntra4x4PredModeDecoderScalar(&decoder, (i*5+3)%9)
			}
		}
	})
	b.Run("ARM64", func(b *testing.B) {
		b.ReportAllocs()
		for range b.N {
			ctx := initialContext
			state := initialState
			decoder := cabacSyntaxDecoder{cabac: &ctx, state: &state}
			for i := range modesPerIteration {
				decodeCABACMBIntra4x4PredModeDecoder(&decoder, (i*5+3)%9)
			}
		}
	})
}
