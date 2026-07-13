// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

import (
	"bytes"
	"fmt"
	"testing"
)

func TestH264WeightPixels16DispatchMatchesScalar(t *testing.T) {
	weights := []int{-4096, -129, -128, -127, -91, -5, -1, 0, 1, 3, 73, 126, 127, 128, 4096}
	offsets := []int{-4096, -129, -128, -65, -64, -17, -1, 0, 1, 29, 126, 127, 128, 4096}
	for _, stride := range []int{16, 23, 32} {
		for _, height := range []int{2, 4, 8, 16} {
			for log2Denom := 0; log2Denom <= 7; log2Denom++ {
				for _, weight := range weights {
					for _, offset := range offsets {
						name := fmt.Sprintf("stride=%d/height=%d/denom=%d/weight=%d/offset=%d", stride, height, log2Denom, weight, offset)
						t.Run(name, func(t *testing.T) {
							got := makeWeightPixelsFixture(stride, height)
							want := append([]byte(nil), got...)
							h264WeightPixelsScalar(want, stride, height, log2Denom, weight, offset, 16)
							if err := h264WeightPixels(got, stride, height, log2Denom, weight, offset, 16); err != nil {
								t.Fatalf("h264WeightPixels: %v", err)
							}
							if !bytes.Equal(got, want) {
								for i := range got {
									if got[i] != want[i] {
										t.Fatalf("byte %d = %d, want %d", i, got[i], want[i])
									}
								}
								t.Fatal("weighted output differs")
							}
						})
					}
				}
			}
		}
	}
}

func BenchmarkH264WeightPixels16(b *testing.B) {
	for _, height := range []int{8, 16} {
		b.Run(fmt.Sprintf("Dispatch/Height%d", height), func(b *testing.B) {
			dst := makeWeightPixelsFixture(32, height)
			b.ReportAllocs()
			b.SetBytes(int64(16 * height))
			for b.Loop() {
				if err := h264WeightPixels(dst, 32, height, 3, -5, 11, 16); err != nil {
					b.Fatal(err)
				}
			}
		})
		b.Run(fmt.Sprintf("Scalar/Height%d", height), func(b *testing.B) {
			dst := makeWeightPixelsFixture(32, height)
			b.ReportAllocs()
			b.SetBytes(int64(16 * height))
			for b.Loop() {
				h264WeightPixelsScalar(dst, 32, height, 3, -5, 11, 16)
			}
		})
	}
}

func makeWeightPixelsFixture(stride int, height int) []byte {
	dst := make([]byte, stride*height)
	for i := range dst {
		dst[i] = byte((i*37 + i/7*19 + 11) & 0xff)
	}
	return dst
}
