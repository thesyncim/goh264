// SPDX-License-Identifier: LGPL-2.1-or-later
//go:build !purego && arm64

package h264

import "testing"

func BenchmarkCABACMVD(b *testing.B) {
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
	const componentsPerIteration = 256
	b.Run("Scalar", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			ctx := initialContext
			state := initialState
			decoder := cabacSyntaxDecoder{cabac: &ctx, state: &state}
			for i := range componentsPerIteration {
				decodeCABACMBMVDDecoderScalar(&decoder, 40+7*(i&1), (i*29+3)%160)
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
			for i := range componentsPerIteration {
				decodeCABACMBMVDDecoder(&decoder, 40+7*(i&1), (i*29+3)%160)
			}
		}
	})
}
