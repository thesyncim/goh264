// SPDX-License-Identifier: LGPL-2.1-or-later
//go:build !purego && arm64

package h264

import (
	"bytes"
	"testing"
)

func TestH264ChromaMCDualARM64MatchesScalar(t *testing.T) {
	const stride = 32
	for _, width := range []int32{8, 4, 2} {
		for _, height := range []int32{2, 4, 8} {
			for _, xy := range [][2]int32{{3, 0}, {0, 5}, {3, 5}, {7, 7}} {
				for _, avg := range []bool{false, true} {
					srcCb := make([]byte, stride*16)
					srcCr := make([]byte, stride*16)
					dstCb := make([]byte, stride*16)
					dstCr := make([]byte, stride*16)
					for i := range srcCb {
						srcCb[i] = byte(i*37 + int(width)*11)
						srcCr[i] = byte(i*53 + int(height)*7)
						dstCb[i] = byte(i*13 + 9)
						dstCr[i] = byte(i*29 + 3)
					}
					wantCb := append([]byte(nil), dstCb...)
					wantCr := append([]byte(nil), dstCr...)
					h264ChromaMCStridesScalar(wantCb, srcCb, stride, stride, int(height), int(xy[0]), int(xy[1]), int(width), avg)
					h264ChromaMCStridesScalar(wantCr, srcCr, stride, stride, int(height), int(xy[0]), int(xy[1]), int(width), avg)

					h264ChromaMCDualStridesKernel(dstCb, dstCr, srcCb, srcCr, stride, stride, height, xy[0], xy[1], width, avg)
					if !bytes.Equal(dstCb, wantCb) || !bytes.Equal(dstCr, wantCr) {
						t.Fatalf("width=%d height=%d x=%d y=%d avg=%v mismatch", width, height, xy[0], xy[1], avg)
					}
				}
			}
		}
	}
}

func BenchmarkH264ChromaMCDualARM64(b *testing.B) {
	const stride = 32
	srcCb := make([]byte, stride*16)
	srcCr := make([]byte, stride*16)
	dstCb := make([]byte, stride*16)
	dstCr := make([]byte, stride*16)
	for i := range srcCb {
		srcCb[i] = byte(i*37 + 11)
		srcCr[i] = byte(i*53 + 7)
	}
	b.Run("Separate", func(b *testing.B) {
		b.ReportAllocs()
		for range b.N {
			h264ChromaMCStridesKernel(dstCb, srcCb, stride, stride, 8, 3, 5, 8, false)
			h264ChromaMCStridesKernel(dstCr, srcCr, stride, stride, 8, 3, 5, 8, false)
		}
	})
	b.Run("Dual", func(b *testing.B) {
		b.ReportAllocs()
		for range b.N {
			h264ChromaMCDualStridesKernel(dstCb, dstCr, srcCb, srcCr, stride, stride, 8, 3, 5, 8, false)
		}
	})
}
