// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

import (
	"strconv"
	"testing"
)

func TestH264LoopFilterDispatchMatchesScalar(t *testing.T) {
	const (
		stride = 32
		rows   = 32
		offset = 12*stride + 12
		alpha  = 20
		beta   = 20
	)
	tc0 := [4]int8{2, 1, 0, 3}

	cases := []struct {
		name   string
		seed   func([]uint8)
		kernel func([]uint8) error
		scalar func([]uint8) error
	}{
		{
			name:   "LumaVertical",
			seed:   func(p []uint8) { seedLoopFilterLuma8(p, offset, stride, 1, 4) },
			kernel: func(p []uint8) error { return h264LoopFilterLumaKernel(p, offset, stride, 1, 4, alpha, beta, &tc0) },
			scalar: func(p []uint8) error { return h264LoopFilterLuma(p, offset, stride, 1, 4, alpha, beta, &tc0) },
		},
		{
			name:   "LumaHorizontal",
			seed:   func(p []uint8) { seedLoopFilterLuma8(p, offset, 1, stride, 4) },
			kernel: func(p []uint8) error { return h264LoopFilterLumaKernel(p, offset, 1, stride, 4, alpha, beta, &tc0) },
			scalar: func(p []uint8) error { return h264LoopFilterLuma(p, offset, 1, stride, 4, alpha, beta, &tc0) },
		},
		{
			name:   "LumaMBAFF",
			seed:   func(p []uint8) { seedLoopFilterLuma8(p, offset, 1, stride, 2) },
			kernel: func(p []uint8) error { return h264LoopFilterLumaKernel(p, offset, 1, stride, 2, alpha, beta, &tc0) },
			scalar: func(p []uint8) error { return h264LoopFilterLuma(p, offset, 1, stride, 2, alpha, beta, &tc0) },
		},
		{
			name:   "LumaIntraVertical",
			seed:   func(p []uint8) { seedLoopFilterLumaIntra8(p, offset, stride, 1, 4) },
			kernel: func(p []uint8) error { return h264LoopFilterLumaIntraKernel(p, offset, stride, 1, 4, alpha, beta) },
			scalar: func(p []uint8) error { return h264LoopFilterLumaIntra(p, offset, stride, 1, 4, alpha, beta) },
		},
		{
			name:   "LumaIntraHorizontal",
			seed:   func(p []uint8) { seedLoopFilterLumaIntra8(p, offset, 1, stride, 4) },
			kernel: func(p []uint8) error { return h264LoopFilterLumaIntraKernel(p, offset, 1, stride, 4, alpha, beta) },
			scalar: func(p []uint8) error { return h264LoopFilterLumaIntra(p, offset, 1, stride, 4, alpha, beta) },
		},
		{
			name:   "LumaIntraMBAFF",
			seed:   func(p []uint8) { seedLoopFilterLumaIntra8(p, offset, 1, stride, 2) },
			kernel: func(p []uint8) error { return h264LoopFilterLumaIntraKernel(p, offset, 1, stride, 2, alpha, beta) },
			scalar: func(p []uint8) error { return h264LoopFilterLumaIntra(p, offset, 1, stride, 2, alpha, beta) },
		},
		{
			name:   "ChromaVertical",
			seed:   func(p []uint8) { seedLoopFilterChroma8(p, offset, stride, 1, 2) },
			kernel: func(p []uint8) error { return h264LoopFilterChromaKernel(p, offset, stride, 1, 2, alpha, beta, &tc0) },
			scalar: func(p []uint8) error { return h264LoopFilterChroma(p, offset, stride, 1, 2, alpha, beta, &tc0) },
		},
		{
			name:   "ChromaHorizontal",
			seed:   func(p []uint8) { seedLoopFilterChroma8(p, offset, 1, stride, 2) },
			kernel: func(p []uint8) error { return h264LoopFilterChromaKernel(p, offset, 1, stride, 2, alpha, beta, &tc0) },
			scalar: func(p []uint8) error { return h264LoopFilterChroma(p, offset, 1, stride, 2, alpha, beta, &tc0) },
		},
		{
			name:   "ChromaMBAFF",
			seed:   func(p []uint8) { seedLoopFilterChroma8(p, offset, 1, stride, 1) },
			kernel: func(p []uint8) error { return h264LoopFilterChromaKernel(p, offset, 1, stride, 1, alpha, beta, &tc0) },
			scalar: func(p []uint8) error { return h264LoopFilterChroma(p, offset, 1, stride, 1, alpha, beta, &tc0) },
		},
		{
			name:   "Chroma422",
			seed:   func(p []uint8) { seedLoopFilterChroma8(p, offset, 1, stride, 4) },
			kernel: func(p []uint8) error { return h264LoopFilterChromaKernel(p, offset, 1, stride, 4, alpha, beta, &tc0) },
			scalar: func(p []uint8) error { return h264LoopFilterChroma(p, offset, 1, stride, 4, alpha, beta, &tc0) },
		},
		{
			name:   "Chroma422MBAFF",
			seed:   func(p []uint8) { seedLoopFilterChroma8(p, offset, 1, stride, 2) },
			kernel: func(p []uint8) error { return h264LoopFilterChromaKernel(p, offset, 1, stride, 2, alpha, beta, &tc0) },
			scalar: func(p []uint8) error { return h264LoopFilterChroma(p, offset, 1, stride, 2, alpha, beta, &tc0) },
		},
		{
			name:   "ChromaIntraVertical",
			seed:   func(p []uint8) { seedLoopFilterChromaIntra8(p, offset, stride, 1, 2) },
			kernel: func(p []uint8) error { return h264LoopFilterChromaIntraKernel(p, offset, stride, 1, 2, alpha, beta) },
			scalar: func(p []uint8) error { return h264LoopFilterChromaIntra(p, offset, stride, 1, 2, alpha, beta) },
		},
		{
			name:   "ChromaIntraHorizontal",
			seed:   func(p []uint8) { seedLoopFilterChromaIntra8(p, offset, 1, stride, 2) },
			kernel: func(p []uint8) error { return h264LoopFilterChromaIntraKernel(p, offset, 1, stride, 2, alpha, beta) },
			scalar: func(p []uint8) error { return h264LoopFilterChromaIntra(p, offset, 1, stride, 2, alpha, beta) },
		},
		{
			name:   "ChromaIntraMBAFF",
			seed:   func(p []uint8) { seedLoopFilterChromaIntra8(p, offset, 1, stride, 1) },
			kernel: func(p []uint8) error { return h264LoopFilterChromaIntraKernel(p, offset, 1, stride, 1, alpha, beta) },
			scalar: func(p []uint8) error { return h264LoopFilterChromaIntra(p, offset, 1, stride, 1, alpha, beta) },
		},
		{
			name:   "ChromaIntra422",
			seed:   func(p []uint8) { seedLoopFilterChromaIntra8(p, offset, 1, stride, 4) },
			kernel: func(p []uint8) error { return h264LoopFilterChromaIntraKernel(p, offset, 1, stride, 4, alpha, beta) },
			scalar: func(p []uint8) error { return h264LoopFilterChromaIntra(p, offset, 1, stride, 4, alpha, beta) },
		},
		{
			name:   "ChromaIntra422MBAFF",
			seed:   func(p []uint8) { seedLoopFilterChromaIntra8(p, offset, 1, stride, 2) },
			kernel: func(p []uint8) error { return h264LoopFilterChromaIntraKernel(p, offset, 1, stride, 2, alpha, beta) },
			scalar: func(p []uint8) error { return h264LoopFilterChromaIntra(p, offset, 1, stride, 2, alpha, beta) },
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			want := makeLoopFilterUnitFixture(stride, rows)
			got := append([]uint8(nil), want...)
			tc.seed(want)
			tc.seed(got)
			if err := tc.scalar(want); err != nil {
				t.Fatalf("scalar: %v", err)
			}
			if err := tc.kernel(got); err != nil {
				t.Fatalf("kernel: %v", err)
			}
			assertUint8SlicesEqual(t, got, want)
		})
	}
}

func TestH264LoopFilterLumaDispatchSkipsNegativeTC0(t *testing.T) {
	const (
		stride = 32
		rows   = 32
		offset = 12*stride + 12
		alpha  = 20
		beta   = 20
	)
	tc0 := [4]int8{2, -2, 0, 3}

	cases := []struct {
		name    string
		xstride int
		ystride int
	}{
		{name: "Vertical", xstride: stride, ystride: 1},
		{name: "Horizontal", xstride: 1, ystride: stride},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			want := makeLoopFilterUnitFixture(stride, rows)
			got := append([]uint8(nil), want...)
			seedLoopFilterLuma8(want, offset, tc.xstride, tc.ystride, 4)
			seedLoopFilterLuma8(got, offset, tc.xstride, tc.ystride, 4)

			if err := h264LoopFilterLuma(want, offset, tc.xstride, tc.ystride, 4, alpha, beta, &tc0); err != nil {
				t.Fatalf("scalar: %v", err)
			}
			if err := h264LoopFilterLumaKernel(got, offset, tc.xstride, tc.ystride, 4, alpha, beta, &tc0); err != nil {
				t.Fatalf("kernel: %v", err)
			}
			assertUint8SlicesEqual(t, got, want)
		})
	}
}

func TestH264LoopFilterChromaDispatchSkipsNegativeTC0(t *testing.T) {
	const (
		stride = 32
		rows   = 32
		offset = 12*stride + 12
		alpha  = 20
		beta   = 20
	)
	tc0 := [4]int8{2, -2, 0, 3}

	cases := []struct {
		name       string
		xstride    int
		ystride    int
		innerIters int
	}{
		{name: "Vertical", xstride: stride, ystride: 1, innerIters: 2},
		{name: "Horizontal", xstride: 1, ystride: stride, innerIters: 2},
		{name: "Chroma422", xstride: 1, ystride: stride, innerIters: 4},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			want := makeLoopFilterUnitFixture(stride, rows)
			got := append([]uint8(nil), want...)
			seedLoopFilterChroma8(want, offset, tc.xstride, tc.ystride, tc.innerIters)
			seedLoopFilterChroma8(got, offset, tc.xstride, tc.ystride, tc.innerIters)

			if err := h264LoopFilterChroma(want, offset, tc.xstride, tc.ystride, tc.innerIters, alpha, beta, &tc0); err != nil {
				t.Fatalf("scalar: %v", err)
			}
			if err := h264LoopFilterChromaKernel(got, offset, tc.xstride, tc.ystride, tc.innerIters, alpha, beta, &tc0); err != nil {
				t.Fatalf("kernel: %v", err)
			}
			assertUint8SlicesEqual(t, got, want)
		})
	}
}

func TestH264LoopFilterHighDispatchMatchesScalar(t *testing.T) {
	const (
		stride   = 32
		rows     = 32
		offset   = 12*stride + 12
		alpha    = 20
		beta     = 20
		bitDepth = 10
	)
	tc0 := [4]int8{2, 1, 0, 3}

	cases := []struct {
		name   string
		seed   func([]uint16)
		kernel func([]uint16) error
		scalar func([]uint16) error
	}{
		{
			name: "LumaVertical",
			seed: func(p []uint16) { seedLoopFilterLumaHigh(p, offset, stride, 1, 4) },
			kernel: func(p []uint16) error {
				return h264LoopFilterLumaHighKernel(p, offset, stride, 1, 4, alpha, beta, &tc0, bitDepth)
			},
			scalar: func(p []uint16) error {
				return h264LoopFilterLumaHigh(p, offset, stride, 1, 4, alpha, beta, &tc0, bitDepth)
			},
		},
		{
			name: "LumaHorizontal",
			seed: func(p []uint16) { seedLoopFilterLumaHigh(p, offset, 1, stride, 4) },
			kernel: func(p []uint16) error {
				return h264LoopFilterLumaHighKernel(p, offset, 1, stride, 4, alpha, beta, &tc0, bitDepth)
			},
			scalar: func(p []uint16) error {
				return h264LoopFilterLumaHigh(p, offset, 1, stride, 4, alpha, beta, &tc0, bitDepth)
			},
		},
		{
			name: "LumaMBAFF",
			seed: func(p []uint16) { seedLoopFilterLumaHigh(p, offset, 1, stride, 2) },
			kernel: func(p []uint16) error {
				return h264LoopFilterLumaHighKernel(p, offset, 1, stride, 2, alpha, beta, &tc0, bitDepth)
			},
			scalar: func(p []uint16) error {
				return h264LoopFilterLumaHigh(p, offset, 1, stride, 2, alpha, beta, &tc0, bitDepth)
			},
		},
		{
			name: "LumaIntraVertical",
			seed: func(p []uint16) { seedLoopFilterLumaIntraHigh(p, offset, stride, 1, 4) },
			kernel: func(p []uint16) error {
				return h264LoopFilterLumaIntraHighKernel(p, offset, stride, 1, 4, alpha, beta, bitDepth)
			},
			scalar: func(p []uint16) error {
				return h264LoopFilterLumaIntraHigh(p, offset, stride, 1, 4, alpha, beta, bitDepth)
			},
		},
		{
			name: "LumaIntraHorizontal",
			seed: func(p []uint16) { seedLoopFilterLumaIntraHigh(p, offset, 1, stride, 4) },
			kernel: func(p []uint16) error {
				return h264LoopFilterLumaIntraHighKernel(p, offset, 1, stride, 4, alpha, beta, bitDepth)
			},
			scalar: func(p []uint16) error {
				return h264LoopFilterLumaIntraHigh(p, offset, 1, stride, 4, alpha, beta, bitDepth)
			},
		},
		{
			name: "LumaIntraMBAFF",
			seed: func(p []uint16) { seedLoopFilterLumaIntraHigh(p, offset, 1, stride, 2) },
			kernel: func(p []uint16) error {
				return h264LoopFilterLumaIntraHighKernel(p, offset, 1, stride, 2, alpha, beta, bitDepth)
			},
			scalar: func(p []uint16) error {
				return h264LoopFilterLumaIntraHigh(p, offset, 1, stride, 2, alpha, beta, bitDepth)
			},
		},
		{
			name: "ChromaVertical",
			seed: func(p []uint16) { seedLoopFilterChromaHigh(p, offset, stride, 1, 2) },
			kernel: func(p []uint16) error {
				return h264LoopFilterChromaHighKernel(p, offset, stride, 1, 2, alpha, beta, &tc0, bitDepth)
			},
			scalar: func(p []uint16) error {
				return h264LoopFilterChromaHigh(p, offset, stride, 1, 2, alpha, beta, &tc0, bitDepth)
			},
		},
		{
			name: "ChromaHorizontal",
			seed: func(p []uint16) { seedLoopFilterChromaHigh(p, offset, 1, stride, 2) },
			kernel: func(p []uint16) error {
				return h264LoopFilterChromaHighKernel(p, offset, 1, stride, 2, alpha, beta, &tc0, bitDepth)
			},
			scalar: func(p []uint16) error {
				return h264LoopFilterChromaHigh(p, offset, 1, stride, 2, alpha, beta, &tc0, bitDepth)
			},
		},
		{
			name: "ChromaMBAFF",
			seed: func(p []uint16) { seedLoopFilterChromaHigh(p, offset, 1, stride, 1) },
			kernel: func(p []uint16) error {
				return h264LoopFilterChromaHighKernel(p, offset, 1, stride, 1, alpha, beta, &tc0, bitDepth)
			},
			scalar: func(p []uint16) error {
				return h264LoopFilterChromaHigh(p, offset, 1, stride, 1, alpha, beta, &tc0, bitDepth)
			},
		},
		{
			name: "Chroma422",
			seed: func(p []uint16) { seedLoopFilterChromaHigh(p, offset, 1, stride, 4) },
			kernel: func(p []uint16) error {
				return h264LoopFilterChromaHighKernel(p, offset, 1, stride, 4, alpha, beta, &tc0, bitDepth)
			},
			scalar: func(p []uint16) error {
				return h264LoopFilterChromaHigh(p, offset, 1, stride, 4, alpha, beta, &tc0, bitDepth)
			},
		},
		{
			name: "Chroma422MBAFF",
			seed: func(p []uint16) { seedLoopFilterChromaHigh(p, offset, 1, stride, 2) },
			kernel: func(p []uint16) error {
				return h264LoopFilterChromaHighKernel(p, offset, 1, stride, 2, alpha, beta, &tc0, bitDepth)
			},
			scalar: func(p []uint16) error {
				return h264LoopFilterChromaHigh(p, offset, 1, stride, 2, alpha, beta, &tc0, bitDepth)
			},
		},
		{
			name: "ChromaIntraVertical",
			seed: func(p []uint16) { seedLoopFilterChromaIntraHigh(p, offset, stride, 1, 2) },
			kernel: func(p []uint16) error {
				return h264LoopFilterChromaIntraHighKernel(p, offset, stride, 1, 2, alpha, beta, bitDepth)
			},
			scalar: func(p []uint16) error {
				return h264LoopFilterChromaIntraHigh(p, offset, stride, 1, 2, alpha, beta, bitDepth)
			},
		},
		{
			name: "ChromaIntraHorizontal",
			seed: func(p []uint16) { seedLoopFilterChromaIntraHigh(p, offset, 1, stride, 2) },
			kernel: func(p []uint16) error {
				return h264LoopFilterChromaIntraHighKernel(p, offset, 1, stride, 2, alpha, beta, bitDepth)
			},
			scalar: func(p []uint16) error {
				return h264LoopFilterChromaIntraHigh(p, offset, 1, stride, 2, alpha, beta, bitDepth)
			},
		},
		{
			name: "ChromaIntraMBAFF",
			seed: func(p []uint16) { seedLoopFilterChromaIntraHigh(p, offset, 1, stride, 1) },
			kernel: func(p []uint16) error {
				return h264LoopFilterChromaIntraHighKernel(p, offset, 1, stride, 1, alpha, beta, bitDepth)
			},
			scalar: func(p []uint16) error {
				return h264LoopFilterChromaIntraHigh(p, offset, 1, stride, 1, alpha, beta, bitDepth)
			},
		},
		{
			name: "ChromaIntra422",
			seed: func(p []uint16) { seedLoopFilterChromaIntraHigh(p, offset, 1, stride, 4) },
			kernel: func(p []uint16) error {
				return h264LoopFilterChromaIntraHighKernel(p, offset, 1, stride, 4, alpha, beta, bitDepth)
			},
			scalar: func(p []uint16) error {
				return h264LoopFilterChromaIntraHigh(p, offset, 1, stride, 4, alpha, beta, bitDepth)
			},
		},
		{
			name: "ChromaIntra422MBAFF",
			seed: func(p []uint16) { seedLoopFilterChromaIntraHigh(p, offset, 1, stride, 2) },
			kernel: func(p []uint16) error {
				return h264LoopFilterChromaIntraHighKernel(p, offset, 1, stride, 2, alpha, beta, bitDepth)
			},
			scalar: func(p []uint16) error {
				return h264LoopFilterChromaIntraHigh(p, offset, 1, stride, 2, alpha, beta, bitDepth)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			want := makeLoopFilterHighUnitFixture(stride, rows)
			got := append([]uint16(nil), want...)
			tc.seed(want)
			tc.seed(got)
			if err := tc.scalar(want); err != nil {
				t.Fatalf("scalar: %v", err)
			}
			if err := tc.kernel(got); err != nil {
				t.Fatalf("kernel: %v", err)
			}
			assertUint16SlicesEqual(t, got, want)
		})
	}
}

func TestH264LoopFilterDispatchRejectsCIntOverflow(t *testing.T) {
	if strconv.IntSize <= 32 {
		t.Skip("host int cannot represent an out-of-range C int")
	}
	const (
		stride = 16
		offset = 8*stride + 4
	)
	tc0 := [4]int8{2, 2, 2, 2}
	pix := makeLoopFilterUnitFixture(stride, 16)

	if err := h264VLoopFilterLuma(pix, offset, stride, 1<<31, 20, &tc0); err != ErrInvalidData {
		t.Fatalf("alpha overflow err = %v, want ErrInvalidData", err)
	}
	if err := h264VLoopFilterChromaIntra(pix, offset, stride, 20, 1<<31); err != ErrInvalidData {
		t.Fatalf("beta overflow err = %v, want ErrInvalidData", err)
	}
}

func BenchmarkH264LoopFilterDeblock(b *testing.B) {
	const (
		stride   = 32
		rows     = 32
		offset   = 12*stride + 12
		alpha    = 20
		beta     = 20
		bitDepth = 10
	)
	tc0 := [4]int8{2, 1, 0, 3}

	b.Run("LumaVertical", func(b *testing.B) {
		pix := makeLoopFilterUnitFixture(stride, rows)
		seedLoopFilterLuma8(pix, offset, stride, 1, 4)
		b.ReportAllocs()
		b.SetBytes(16 * 16)
		for i := 0; i < b.N; i++ {
			if err := h264LoopFilterLumaKernel(pix, offset, stride, 1, 4, alpha, beta, &tc0); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("LumaHorizontal", func(b *testing.B) {
		pix := makeLoopFilterUnitFixture(stride, rows)
		seedLoopFilterLuma8(pix, offset, 1, stride, 4)
		b.ReportAllocs()
		b.SetBytes(16 * 16)
		for i := 0; i < b.N; i++ {
			if err := h264LoopFilterLumaKernel(pix, offset, 1, stride, 4, alpha, beta, &tc0); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("LumaIntraVertical", func(b *testing.B) {
		pix := makeLoopFilterUnitFixture(stride, rows)
		seedLoopFilterLumaIntra8(pix, offset, stride, 1, 4)
		b.ReportAllocs()
		b.SetBytes(16 * 16)
		for i := 0; i < b.N; i++ {
			if err := h264LoopFilterLumaIntraKernel(pix, offset, stride, 1, 4, alpha, beta); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("LumaIntraHorizontal", func(b *testing.B) {
		pix := makeLoopFilterUnitFixture(stride, rows)
		seedLoopFilterLumaIntra8(pix, offset, 1, stride, 4)
		b.ReportAllocs()
		b.SetBytes(16 * 16)
		for i := 0; i < b.N; i++ {
			if err := h264LoopFilterLumaIntraKernel(pix, offset, 1, stride, 4, alpha, beta); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("ChromaVertical", func(b *testing.B) {
		pix := makeLoopFilterUnitFixture(stride, rows)
		seedLoopFilterChroma8(pix, offset, stride, 1, 2)
		b.ReportAllocs()
		b.SetBytes(8 * 8)
		for i := 0; i < b.N; i++ {
			if err := h264LoopFilterChromaKernel(pix, offset, stride, 1, 2, alpha, beta, &tc0); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("ChromaHorizontal", func(b *testing.B) {
		pix := makeLoopFilterUnitFixture(stride, rows)
		seedLoopFilterChroma8(pix, offset, 1, stride, 2)
		b.ReportAllocs()
		b.SetBytes(8 * 8)
		for i := 0; i < b.N; i++ {
			if err := h264LoopFilterChromaKernel(pix, offset, 1, stride, 2, alpha, beta, &tc0); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("Chroma422Horizontal", func(b *testing.B) {
		pix := makeLoopFilterUnitFixture(stride, rows)
		seedLoopFilterChroma8(pix, offset, 1, stride, 4)
		b.ReportAllocs()
		b.SetBytes(8 * 16)
		for i := 0; i < b.N; i++ {
			if err := h264LoopFilterChromaKernel(pix, offset, 1, stride, 4, alpha, beta, &tc0); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("ChromaIntraVertical", func(b *testing.B) {
		pix := makeLoopFilterUnitFixture(stride, rows)
		seedLoopFilterChromaIntra8(pix, offset, stride, 1, 2)
		b.ReportAllocs()
		b.SetBytes(8 * 8)
		for i := 0; i < b.N; i++ {
			if err := h264LoopFilterChromaIntraKernel(pix, offset, stride, 1, 2, alpha, beta); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("ChromaIntraHorizontal", func(b *testing.B) {
		pix := makeLoopFilterUnitFixture(stride, rows)
		seedLoopFilterChromaIntra8(pix, offset, 1, stride, 2)
		b.ReportAllocs()
		b.SetBytes(8 * 8)
		for i := 0; i < b.N; i++ {
			if err := h264LoopFilterChromaIntraKernel(pix, offset, 1, stride, 2, alpha, beta); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("ChromaIntraMBAFF", func(b *testing.B) {
		pix := makeLoopFilterUnitFixture(stride, rows)
		seedLoopFilterChromaIntra8(pix, offset, 1, stride, 1)
		b.ReportAllocs()
		b.SetBytes(8 * 4)
		for i := 0; i < b.N; i++ {
			if err := h264LoopFilterChromaIntraKernel(pix, offset, 1, stride, 1, alpha, beta); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("ChromaIntra422", func(b *testing.B) {
		pix := makeLoopFilterUnitFixture(stride, rows)
		seedLoopFilterChromaIntra8(pix, offset, 1, stride, 4)
		b.ReportAllocs()
		b.SetBytes(8 * 16)
		for i := 0; i < b.N; i++ {
			if err := h264LoopFilterChromaIntraKernel(pix, offset, 1, stride, 4, alpha, beta); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("ChromaVerticalHigh10", func(b *testing.B) {
		pix := makeLoopFilterHighUnitFixture(stride, rows)
		seedLoopFilterChromaHigh(pix, offset, stride, 1, 2)
		b.ReportAllocs()
		b.SetBytes(8 * 8 * 2)
		for i := 0; i < b.N; i++ {
			if err := h264LoopFilterChromaHighKernel(pix, offset, stride, 1, 2, alpha, beta, &tc0, bitDepth); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("ChromaHorizontalHigh10", func(b *testing.B) {
		pix := makeLoopFilterHighUnitFixture(stride, rows)
		seedLoopFilterChromaHigh(pix, offset, 1, stride, 2)
		b.ReportAllocs()
		b.SetBytes(8 * 8 * 2)
		for i := 0; i < b.N; i++ {
			if err := h264LoopFilterChromaHighKernel(pix, offset, 1, stride, 2, alpha, beta, &tc0, bitDepth); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("Chroma422HorizontalHigh10", func(b *testing.B) {
		pix := makeLoopFilterHighUnitFixture(stride, rows)
		seedLoopFilterChromaHigh(pix, offset, 1, stride, 4)
		b.ReportAllocs()
		b.SetBytes(8 * 16 * 2)
		for i := 0; i < b.N; i++ {
			if err := h264LoopFilterChromaHighKernel(pix, offset, 1, stride, 4, alpha, beta, &tc0, bitDepth); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("ChromaIntraVerticalHigh10", func(b *testing.B) {
		pix := makeLoopFilterHighUnitFixture(stride, rows)
		seedLoopFilterChromaIntraHigh(pix, offset, stride, 1, 2)
		b.ReportAllocs()
		b.SetBytes(8 * 8 * 2)
		for i := 0; i < b.N; i++ {
			if err := h264LoopFilterChromaIntraHighKernel(pix, offset, stride, 1, 2, alpha, beta, bitDepth); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("ChromaIntraHorizontalHigh10", func(b *testing.B) {
		pix := makeLoopFilterHighUnitFixture(stride, rows)
		seedLoopFilterChromaIntraHigh(pix, offset, 1, stride, 2)
		b.ReportAllocs()
		b.SetBytes(8 * 8 * 2)
		for i := 0; i < b.N; i++ {
			if err := h264LoopFilterChromaIntraHighKernel(pix, offset, 1, stride, 2, alpha, beta, bitDepth); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("ChromaIntraMBAFFHigh10", func(b *testing.B) {
		pix := makeLoopFilterHighUnitFixture(stride, rows)
		seedLoopFilterChromaIntraHigh(pix, offset, 1, stride, 1)
		b.ReportAllocs()
		b.SetBytes(8 * 4 * 2)
		for i := 0; i < b.N; i++ {
			if err := h264LoopFilterChromaIntraHighKernel(pix, offset, 1, stride, 1, alpha, beta, bitDepth); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("ChromaIntra422High10", func(b *testing.B) {
		pix := makeLoopFilterHighUnitFixture(stride, rows)
		seedLoopFilterChromaIntraHigh(pix, offset, 1, stride, 4)
		b.ReportAllocs()
		b.SetBytes(8 * 16 * 2)
		for i := 0; i < b.N; i++ {
			if err := h264LoopFilterChromaIntraHighKernel(pix, offset, 1, stride, 4, alpha, beta, bitDepth); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("ChromaIntra422MBAFFHigh10", func(b *testing.B) {
		pix := makeLoopFilterHighUnitFixture(stride, rows)
		seedLoopFilterChromaIntraHigh(pix, offset, 1, stride, 2)
		b.ReportAllocs()
		b.SetBytes(8 * 8 * 2)
		for i := 0; i < b.N; i++ {
			if err := h264LoopFilterChromaIntraHighKernel(pix, offset, 1, stride, 2, alpha, beta, bitDepth); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("LumaVerticalHigh10", func(b *testing.B) {
		pix := makeLoopFilterHighUnitFixture(stride, rows)
		seedLoopFilterLumaHigh(pix, offset, stride, 1, 4)
		b.ReportAllocs()
		b.SetBytes(16 * 16 * 2)
		for i := 0; i < b.N; i++ {
			if err := h264LoopFilterLumaHighKernel(pix, offset, stride, 1, 4, alpha, beta, &tc0, bitDepth); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func seedLoopFilterLuma8(pix []uint8, offset int, xstride int, ystride int, innerIters int) {
	pos := offset
	for i := 0; i < 4; i++ {
		for d := 0; d < innerIters; d++ {
			pix[pos-3*xstride] = 98
			pix[pos-2*xstride] = 100
			pix[pos-1*xstride] = 102
			pix[pos] = 108
			pix[pos+1*xstride] = 110
			pix[pos+2*xstride] = 112
			pos += ystride
		}
	}
}

func seedLoopFilterLumaHigh(pix []uint16, offset int, xstride int, ystride int, innerIters int) {
	pos := offset
	for i := 0; i < 4; i++ {
		for d := 0; d < innerIters; d++ {
			pix[pos-3*xstride] = 98 << 2
			pix[pos-2*xstride] = 100 << 2
			pix[pos-1*xstride] = 102 << 2
			pix[pos] = 108 << 2
			pix[pos+1*xstride] = 110 << 2
			pix[pos+2*xstride] = 112 << 2
			pos += ystride
		}
	}
}

func seedLoopFilterLumaIntra8(pix []uint8, offset int, xstride int, ystride int, innerIters int) {
	pos := offset
	for d := 0; d < 4*innerIters; d++ {
		pix[pos-4*xstride] = 96
		pix[pos-3*xstride] = 98
		pix[pos-2*xstride] = 100
		pix[pos-1*xstride] = 102
		pix[pos] = 108
		pix[pos+1*xstride] = 110
		pix[pos+2*xstride] = 112
		pix[pos+3*xstride] = 114
		pos += ystride
	}
}

func seedLoopFilterLumaIntraHigh(pix []uint16, offset int, xstride int, ystride int, innerIters int) {
	pos := offset
	for d := 0; d < 4*innerIters; d++ {
		pix[pos-4*xstride] = 96 << 2
		pix[pos-3*xstride] = 98 << 2
		pix[pos-2*xstride] = 100 << 2
		pix[pos-1*xstride] = 102 << 2
		pix[pos] = 108 << 2
		pix[pos+1*xstride] = 110 << 2
		pix[pos+2*xstride] = 112 << 2
		pix[pos+3*xstride] = 114 << 2
		pos += ystride
	}
}

func seedLoopFilterChroma8(pix []uint8, offset int, xstride int, ystride int, innerIters int) {
	pos := offset
	for i := 0; i < 4; i++ {
		for d := 0; d < innerIters; d++ {
			pix[pos-2*xstride] = 100
			pix[pos-1*xstride] = 102
			pix[pos] = 108
			pix[pos+1*xstride] = 110
			pos += ystride
		}
	}
}

func seedLoopFilterChromaHigh(pix []uint16, offset int, xstride int, ystride int, innerIters int) {
	pos := offset
	for i := 0; i < 4; i++ {
		for d := 0; d < innerIters; d++ {
			pix[pos-2*xstride] = 100 << 2
			pix[pos-1*xstride] = 102 << 2
			pix[pos] = 108 << 2
			pix[pos+1*xstride] = 110 << 2
			pos += ystride
		}
	}
}

func seedLoopFilterChromaIntra8(pix []uint8, offset int, xstride int, ystride int, innerIters int) {
	pos := offset
	for d := 0; d < 4*innerIters; d++ {
		pix[pos-2*xstride] = 100
		pix[pos-1*xstride] = 102
		pix[pos] = 108
		pix[pos+1*xstride] = 110
		pos += ystride
	}
}

func seedLoopFilterChromaIntraHigh(pix []uint16, offset int, xstride int, ystride int, innerIters int) {
	pos := offset
	for d := 0; d < 4*innerIters; d++ {
		pix[pos-2*xstride] = 100 << 2
		pix[pos-1*xstride] = 102 << 2
		pix[pos] = 108 << 2
		pix[pos+1*xstride] = 110 << 2
		pos += ystride
	}
}

func assertUint8SlicesEqual(t *testing.T, got []uint8, want []uint8) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d", len(got), len(want))
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("pix[%d] = %d, want %d", i, got[i], want[i])
		}
	}
}

func assertUint16SlicesEqual(t *testing.T, got []uint16, want []uint16) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d", len(got), len(want))
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("pix[%d] = %d, want %d", i, got[i], want[i])
		}
	}
}
