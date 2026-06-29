// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

import "testing"

func TestH264QpelPutRepresentativeModes(t *testing.T) {
	const stride = 48
	const offset = 6*stride + 6

	for _, c := range []struct {
		name string
		size int
		mx   int
		my   int
		want [4]uint8
	}{
		{name: "h_lowpass", size: 4, mx: 2, my: 0, want: [4]uint8{101, 128, 117, 144}},
		{name: "hv_lowpass", size: 8, mx: 2, my: 2, want: [4]uint8{25, 99, 141, 36}},
		{name: "diagonal_half", size: 16, mx: 3, my: 3, want: [4]uint8{27, 194, 143, 190}},
		{name: "hv_blend_21", size: 16, mx: 2, my: 1, want: [4]uint8{63, 228, 183, 102}},
		{name: "hv_blend_12", size: 16, mx: 1, my: 2, want: [4]uint8{21, 229, 180, 143}},
		{name: "hv_blend_32", size: 16, mx: 3, my: 2, want: [4]uint8{25, 230, 185, 148}},
		{name: "hv_blend_23", size: 16, mx: 2, my: 3, want: [4]uint8{27, 192, 143, 186}},
	} {
		t.Run(c.name, func(t *testing.T) {
			dst, src := makeQpelUnitFixture(stride, 48)
			if err := h264PutH264QpelMC(dst, offset, src, offset, stride, c.size, c.mx, c.my); err != nil {
				t.Fatal(err)
			}
			got := [4]uint8{
				dst[offset],
				dst[offset+c.size-1],
				dst[offset+(c.size-1)*stride],
				dst[offset+(c.size-1)*stride+c.size-1],
			}
			if got != c.want {
				t.Fatalf("selected samples = %v, want %v", got, c.want)
			}
		})
	}
}

func TestH264QpelAvgRepresentativeMode(t *testing.T) {
	const stride = 48
	const offset = 6*stride + 6
	dst, src := makeQpelUnitFixture(stride, 48)

	if err := h264AvgH264QpelMC(dst, offset, src, offset, stride, 4, 1, 3); err != nil {
		t.Fatal(err)
	}
	got := [4]uint8{dst[offset], dst[offset+3], dst[offset+3*stride], dst[offset+3*stride+3]}
	want := [4]uint8{103, 131, 133, 35}
	if got != want {
		t.Fatalf("selected avg samples = %v, want %v", got, want)
	}
	if dst[offset+4] != 226 {
		t.Fatalf("padding byte changed to %d, want 226", dst[offset+4])
	}
}

func TestH264QpelAvgHVBlendRepresentativeMode(t *testing.T) {
	const stride = 48
	const offset = 6*stride + 6
	dst, src := makeQpelUnitFixture(stride, 48)

	if err := h264AvgH264QpelMC(dst, offset, src, offset, stride, 8, 3, 2); err != nil {
		t.Fatal(err)
	}
	got := [4]uint8{dst[offset], dst[offset+7], dst[offset+7*stride], dst[offset+7*stride+7]}
	want := [4]uint8{104, 59, 91, 77}
	if got != want {
		t.Fatalf("selected hv-blend avg samples = %v, want %v", got, want)
	}
	if dst[offset+8] != 14 {
		t.Fatalf("padding byte changed to %d, want 14", dst[offset+8])
	}
}

func TestH264QpelHighRepresentativeModes(t *testing.T) {
	const stride = 48
	const offset = 6*stride + 6
	dst := make([]uint16, stride*48)
	src := make([]uint16, stride*48)
	for i := range src {
		src[i] = 1023
	}

	if err := h264PutH264QpelMCHigh(dst, offset, src, offset, stride, 4, 2, 0, 10); err != nil {
		t.Fatal(err)
	}
	if dst[offset] != 1023 || dst[offset+3] != 1023 || dst[offset+3*stride+3] != 1023 {
		t.Fatalf("high h-lowpass samples = %d/%d/%d, want 1023", dst[offset], dst[offset+3], dst[offset+3*stride+3])
	}

	for i := range dst {
		dst[i] = 1
	}
	if err := h264AvgH264QpelMCHigh(dst, offset, src, offset, stride, 8, 2, 2, 10); err != nil {
		t.Fatal(err)
	}
	if dst[offset] != 512 || dst[offset+7] != 512 || dst[offset+7*stride+7] != 512 {
		t.Fatalf("high hv avg samples = %d/%d/%d, want 512", dst[offset], dst[offset+7], dst[offset+7*stride+7])
	}
	if dst[offset+8] != 1 {
		t.Fatalf("high padding sample changed to %d, want 1", dst[offset+8])
	}
}

func TestH264QpelValidatesGeometry(t *testing.T) {
	if err := h264PutH264QpelMC(make([]uint8, 16), 0, make([]uint8, 16), 0, 4, 4, 0, 0); err != nil {
		t.Fatalf("mc00 without margins error = %v, want nil", err)
	}
	if err := h264PutH264QpelMC(make([]uint8, 16), 0, make([]uint8, 16), 0, 4, 4, 2, 0); err != ErrInvalidData {
		t.Fatalf("missing horizontal margin error = %v, want ErrInvalidData", err)
	}
	if err := h264PutH264QpelMC(make([]uint8, 16), 0, make([]uint8, 64), 0, 4, 3, 0, 0); err != ErrInvalidData {
		t.Fatalf("invalid size error = %v, want ErrInvalidData", err)
	}
	if err := h264PutH264QpelMC(make([]uint8, 16), 0, make([]uint8, 64), 0, 4, 4, 4, 0); err != ErrInvalidData {
		t.Fatalf("invalid motion fraction error = %v, want ErrInvalidData", err)
	}
	if err := h264PutH264QpelMCHigh(make([]uint16, 16), 0, make([]uint16, 16), 0, 4, 4, 0, 0, 11); err != ErrUnsupported {
		t.Fatalf("unsupported high qpel bit depth err = %v, want ErrUnsupported", err)
	}
	if err := h264PutH264QpelMCHigh(make([]uint16, 16), 0, make([]uint16, 16), 0, 4, 4, 2, 0, 10); err != ErrInvalidData {
		t.Fatalf("missing high horizontal margin error = %v, want ErrInvalidData", err)
	}
	if err := h264QpelMCStrides(make([]uint8, 16), 0, maxInt, make([]uint8, 16), 0, maxInt, 16, 0, 0, false); err != ErrInvalidData {
		t.Fatalf("overflowed qpel geometry error = %v, want ErrInvalidData", err)
	}
	if err := h264QpelMCStridesHigh(make([]uint16, 16), 0, maxInt, make([]uint16, 16), 0, maxInt, 16, 0, 0, false, 10); err != ErrInvalidData {
		t.Fatalf("overflowed high qpel geometry error = %v, want ErrInvalidData", err)
	}
	if err := h264QpelMCStrides(make([]uint8, 16*16), 0, 16, make([]uint8, 16), 0, maxInt, 16, 0, 2, false); err != ErrInvalidData {
		t.Fatalf("overflowed qpel source geometry error = %v, want ErrInvalidData", err)
	}
}

func TestH264QpelRejectsCIntOverflow(t *testing.T) {
	tooLarge := intAboveCInt(t)
	if err := h264QpelMCStrides(make([]uint8, 16), 0, 4, make([]uint8, 16), 0, 4, tooLarge, 0, 0, false); err != ErrInvalidData {
		t.Fatalf("oversized C int size error = %v, want ErrInvalidData", err)
	}
	if err := h264QpelMCStridesHigh(make([]uint16, 16), 0, 4, make([]uint16, 16), 0, 4, 4, 0, 0, false, tooLarge); err != ErrInvalidData {
		t.Fatalf("oversized high C int bit depth error = %v, want ErrInvalidData", err)
	}
}

func BenchmarkH264QpelMC16Put00(b *testing.B) {
	benchmarkH264QpelMCCopy(b, 16, false)
}

func BenchmarkH264QpelMC16Avg00(b *testing.B) {
	benchmarkH264QpelMCCopy(b, 16, true)
}

func BenchmarkH264QpelMC16Put10(b *testing.B) {
	benchmarkH264QpelMC(b, 16, 1, 0, false)
}

func BenchmarkH264QpelMC16Avg10(b *testing.B) {
	benchmarkH264QpelMC(b, 16, 1, 0, true)
}

func BenchmarkH264QpelMC16Put20(b *testing.B) {
	benchmarkH264QpelMC(b, 16, 2, 0, false)
}

func BenchmarkH264QpelMC16Avg20(b *testing.B) {
	benchmarkH264QpelMC(b, 16, 2, 0, true)
}

func BenchmarkH264QpelMC16Put30(b *testing.B) {
	benchmarkH264QpelMC(b, 16, 3, 0, false)
}

func BenchmarkH264QpelMC16Avg30(b *testing.B) {
	benchmarkH264QpelMC(b, 16, 3, 0, true)
}

func BenchmarkH264QpelMC16Put01(b *testing.B) {
	benchmarkH264QpelMC(b, 16, 0, 1, false)
}

func BenchmarkH264QpelMC16Avg01(b *testing.B) {
	benchmarkH264QpelMC(b, 16, 0, 1, true)
}

func BenchmarkH264QpelMC16Put02(b *testing.B) {
	benchmarkH264QpelMC(b, 16, 0, 2, false)
}

func BenchmarkH264QpelMC16Avg02(b *testing.B) {
	benchmarkH264QpelMC(b, 16, 0, 2, true)
}

func BenchmarkH264QpelMC16Put03(b *testing.B) {
	benchmarkH264QpelMC(b, 16, 0, 3, false)
}

func BenchmarkH264QpelMC16Avg03(b *testing.B) {
	benchmarkH264QpelMC(b, 16, 0, 3, true)
}

func BenchmarkH264QpelMC16Put22(b *testing.B) {
	benchmarkH264QpelMC(b, 16, 2, 2, false)
}

func BenchmarkH264QpelMC16Avg22(b *testing.B) {
	benchmarkH264QpelMC(b, 16, 2, 2, true)
}

func BenchmarkH264QpelMC16Put11(b *testing.B) {
	benchmarkH264QpelMC(b, 16, 1, 1, false)
}

func BenchmarkH264QpelMC16Avg11(b *testing.B) {
	benchmarkH264QpelMC(b, 16, 1, 1, true)
}

func BenchmarkH264QpelMC16Put31(b *testing.B) {
	benchmarkH264QpelMC(b, 16, 3, 1, false)
}

func BenchmarkH264QpelMC16Avg31(b *testing.B) {
	benchmarkH264QpelMC(b, 16, 3, 1, true)
}

func BenchmarkH264QpelMC16Put13(b *testing.B) {
	benchmarkH264QpelMC(b, 16, 1, 3, false)
}

func BenchmarkH264QpelMC16Avg13(b *testing.B) {
	benchmarkH264QpelMC(b, 16, 1, 3, true)
}

func BenchmarkH264QpelMC16Put33(b *testing.B) {
	benchmarkH264QpelMC(b, 16, 3, 3, false)
}

func BenchmarkH264QpelMC16Avg33(b *testing.B) {
	benchmarkH264QpelMC(b, 16, 3, 3, true)
}

func BenchmarkH264QpelMC16Put21(b *testing.B) {
	benchmarkH264QpelMC(b, 16, 2, 1, false)
}

func BenchmarkH264QpelMC16Avg21(b *testing.B) {
	benchmarkH264QpelMC(b, 16, 2, 1, true)
}

func BenchmarkH264QpelMC16Put12(b *testing.B) {
	benchmarkH264QpelMC(b, 16, 1, 2, false)
}

func BenchmarkH264QpelMC16Avg12(b *testing.B) {
	benchmarkH264QpelMC(b, 16, 1, 2, true)
}

func BenchmarkH264QpelMC16Put32(b *testing.B) {
	benchmarkH264QpelMC(b, 16, 3, 2, false)
}

func BenchmarkH264QpelMC16Avg32(b *testing.B) {
	benchmarkH264QpelMC(b, 16, 3, 2, true)
}

func BenchmarkH264QpelMC16Put23(b *testing.B) {
	benchmarkH264QpelMC(b, 16, 2, 3, false)
}

func BenchmarkH264QpelMC16Avg23(b *testing.B) {
	benchmarkH264QpelMC(b, 16, 2, 3, true)
}

func BenchmarkH264QpelMC8Put00(b *testing.B) {
	benchmarkH264QpelMCCopy(b, 8, false)
}

func BenchmarkH264QpelMC8Avg00(b *testing.B) {
	benchmarkH264QpelMCCopy(b, 8, true)
}

func BenchmarkH264QpelMC8Put10(b *testing.B) {
	benchmarkH264QpelMC(b, 8, 1, 0, false)
}

func BenchmarkH264QpelMC8Avg10(b *testing.B) {
	benchmarkH264QpelMC(b, 8, 1, 0, true)
}

func BenchmarkH264QpelMC8Put20(b *testing.B) {
	benchmarkH264QpelMC(b, 8, 2, 0, false)
}

func BenchmarkH264QpelMC8Avg20(b *testing.B) {
	benchmarkH264QpelMC(b, 8, 2, 0, true)
}

func BenchmarkH264QpelMC8Put30(b *testing.B) {
	benchmarkH264QpelMC(b, 8, 3, 0, false)
}

func BenchmarkH264QpelMC8Avg30(b *testing.B) {
	benchmarkH264QpelMC(b, 8, 3, 0, true)
}

func BenchmarkH264QpelMC8Put01(b *testing.B) {
	benchmarkH264QpelMC(b, 8, 0, 1, false)
}

func BenchmarkH264QpelMC8Avg01(b *testing.B) {
	benchmarkH264QpelMC(b, 8, 0, 1, true)
}

func BenchmarkH264QpelMC8Put02(b *testing.B) {
	benchmarkH264QpelMC(b, 8, 0, 2, false)
}

func BenchmarkH264QpelMC8Avg02(b *testing.B) {
	benchmarkH264QpelMC(b, 8, 0, 2, true)
}

func BenchmarkH264QpelMC8Put03(b *testing.B) {
	benchmarkH264QpelMC(b, 8, 0, 3, false)
}

func BenchmarkH264QpelMC8Avg03(b *testing.B) {
	benchmarkH264QpelMC(b, 8, 0, 3, true)
}

func BenchmarkH264QpelMC8Put22(b *testing.B) {
	benchmarkH264QpelMC(b, 8, 2, 2, false)
}

func BenchmarkH264QpelMC8Avg22(b *testing.B) {
	benchmarkH264QpelMC(b, 8, 2, 2, true)
}

func BenchmarkH264QpelMC8Put11(b *testing.B) {
	benchmarkH264QpelMC(b, 8, 1, 1, false)
}

func BenchmarkH264QpelMC8Avg11(b *testing.B) {
	benchmarkH264QpelMC(b, 8, 1, 1, true)
}

func BenchmarkH264QpelMC8Put31(b *testing.B) {
	benchmarkH264QpelMC(b, 8, 3, 1, false)
}

func BenchmarkH264QpelMC8Avg31(b *testing.B) {
	benchmarkH264QpelMC(b, 8, 3, 1, true)
}

func BenchmarkH264QpelMC8Put13(b *testing.B) {
	benchmarkH264QpelMC(b, 8, 1, 3, false)
}

func BenchmarkH264QpelMC8Avg13(b *testing.B) {
	benchmarkH264QpelMC(b, 8, 1, 3, true)
}

func BenchmarkH264QpelMC8Put33(b *testing.B) {
	benchmarkH264QpelMC(b, 8, 3, 3, false)
}

func BenchmarkH264QpelMC8Avg33(b *testing.B) {
	benchmarkH264QpelMC(b, 8, 3, 3, true)
}

func BenchmarkH264QpelMC8Put21(b *testing.B) {
	benchmarkH264QpelMC(b, 8, 2, 1, false)
}

func BenchmarkH264QpelMC8Avg21(b *testing.B) {
	benchmarkH264QpelMC(b, 8, 2, 1, true)
}

func BenchmarkH264QpelMC8Put12(b *testing.B) {
	benchmarkH264QpelMC(b, 8, 1, 2, false)
}

func BenchmarkH264QpelMC8Avg12(b *testing.B) {
	benchmarkH264QpelMC(b, 8, 1, 2, true)
}

func BenchmarkH264QpelMC8Put32(b *testing.B) {
	benchmarkH264QpelMC(b, 8, 3, 2, false)
}

func BenchmarkH264QpelMC8Avg32(b *testing.B) {
	benchmarkH264QpelMC(b, 8, 3, 2, true)
}

func BenchmarkH264QpelMC8Put23(b *testing.B) {
	benchmarkH264QpelMC(b, 8, 2, 3, false)
}

func BenchmarkH264QpelMC8Avg23(b *testing.B) {
	benchmarkH264QpelMC(b, 8, 2, 3, true)
}

func BenchmarkH264QpelMC4Put00(b *testing.B) {
	benchmarkH264QpelMCCopy(b, 4, false)
}

func BenchmarkH264QpelMC4Avg00(b *testing.B) {
	benchmarkH264QpelMCCopy(b, 4, true)
}

func BenchmarkH264QpelMC2Put00(b *testing.B) {
	benchmarkH264QpelMCCopy(b, 2, false)
}

func BenchmarkH264QpelMC2Avg00(b *testing.B) {
	benchmarkH264QpelMCCopy(b, 2, true)
}

func benchmarkH264QpelMCCopy(b *testing.B, size int, avg bool) {
	benchmarkH264QpelMC(b, size, 0, 0, avg)
}

func benchmarkH264QpelMC(b *testing.B, size int, mx int, my int, avg bool) {
	const stride = 64
	const rows = 32
	const offset = 6*stride + 6
	dst, src := makeQpelUnitFixture(stride, rows)
	b.ReportAllocs()
	b.SetBytes(int64(size * size))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := h264QpelMCStrides(dst, offset, stride, src, offset, stride, size, mx, my, avg); err != nil {
			b.Fatal(err)
		}
	}
}

func makeQpelUnitFixture(stride int, rows int) ([]uint8, []uint8) {
	dst := make([]uint8, stride*rows)
	src := make([]uint8, stride*rows)
	for i := range dst {
		dst[i] = uint8((20 + i*11) & 255)
		src[i] = uint8((10 + i*9) & 255)
	}
	return dst, src
}
