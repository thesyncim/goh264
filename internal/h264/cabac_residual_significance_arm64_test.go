// SPDX-License-Identifier: LGPL-2.1-or-later
//go:build !purego && arm64

package h264

import "testing"

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
