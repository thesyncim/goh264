// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

import (
	"bytes"
	"testing"
)

func TestH264Pred4x4AngularKernelsMatchScalar(t *testing.T) {
	state := uint32(0x6d2b79f5)
	nextByte := func() uint8 {
		state = state*1664525 + 1013904223
		return uint8(state >> 24)
	}
	for iteration := 0; iteration < 4096; iteration++ {
		stride := 8 + iteration%57
		offset := (2+iteration%5)*stride + 2 + iteration%3
		got := make([]uint8, stride*12)
		for i := range got {
			got[i] = nextByte()
		}
		for _, tc := range []struct {
			name   string
			kernel func([]uint8, int, int)
			scalar func([]uint8, int, int)
		}{
			{"DownRight", h264Pred4x4DownRightKernel, h264Pred4x4DownRightScalar},
			{"HorizontalUp", h264Pred4x4HorizontalUpKernel, h264Pred4x4HorizontalUpScalar},
		} {
			want := append([]uint8(nil), got...)
			actual := append([]uint8(nil), got...)
			tc.kernel(actual, offset, stride)
			tc.scalar(want, offset, stride)
			if !bytes.Equal(actual, want) {
				t.Fatalf("%s iteration %d stride %d differs from scalar", tc.name, iteration, stride)
			}
		}
	}
}

func BenchmarkH264Pred4x4Simple(b *testing.B) {
	const stride = 32
	const offset = 4*stride + 4
	for _, tc := range []struct {
		name string
		mode int
	}{
		{"Vertical", int(intraPredVertical)},
		{"Horizontal", int(intraPredHorizontal)},
		{"DC", int(intraPredDC)},
		{"LeftDC", int(intraPredLeftDC)},
		{"TopDC", int(intraPredTopDC)},
		{"DC128", int(intraPredDC128)},
	} {
		b.Run(tc.name, func(b *testing.B) {
			pix := make([]uint8, stride*12)
			for i := range pix {
				pix[i] = uint8(i*37 + 11)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				if err := h264Pred4x4ByMode(pix, offset, stride, tc.mode, nil); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkH264Pred4x4Angular(b *testing.B) {
	const stride = 32
	const offset = 4*stride + 4
	topRight := []uint8{91, 123, 155, 177}
	for _, tc := range []struct {
		name string
		mode int
	}{
		{"DownLeft", int(intraPredDiagDownLeft)},
		{"DownRight", int(intraPredDiagDownRight)},
		{"VerticalRight", int(intraPredVertRight)},
		{"HorizontalDown", int(intraPredHorDown)},
		{"VerticalLeft", int(intraPredVertLeft)},
		{"HorizontalUp", int(intraPredHorUp)},
	} {
		b.Run(tc.name, func(b *testing.B) {
			pix := make([]uint8, stride*12)
			for i := range pix {
				pix[i] = uint8(i*37 + 11)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				if err := h264Pred4x4ByMode(pix, offset, stride, tc.mode, topRight); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
