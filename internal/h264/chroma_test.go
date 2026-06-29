// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

import "testing"

func TestH264ChromaMCPutBranches(t *testing.T) {
	for _, c := range []struct {
		name string
		x    int
		y    int
		want [6]uint8
	}{
		{name: "copy", x: 0, y: 0, want: [6]uint8{10, 37, 118, 145, 26, 53}},
		{name: "horizontal", x: 3, y: 0, want: [6]uint8{13, 40, 121, 148, 29, 56}},
		{name: "vertical", x: 0, y: 5, want: [6]uint8{78, 105, 61, 88, 94, 121}},
		{name: "bilinear", x: 3, y: 5, want: [6]uint8{81, 108, 64, 91, 97, 124}},
	} {
		t.Run(c.name, func(t *testing.T) {
			const stride = 12
			dst := makeChromaUnitDst(stride, 6)
			src := makeChromaUnitSrc(stride, 7)

			if err := h264PutH264ChromaMC4(dst, src, stride, 3, c.x, c.y); err != nil {
				t.Fatal(err)
			}
			got := [6]uint8{dst[0], dst[3], dst[stride], dst[stride+3], dst[2*stride], dst[2*stride+3]}
			if got != c.want {
				t.Fatalf("selected samples = %v, want %v", got, c.want)
			}
			if dst[4] != 0 {
				t.Fatalf("padding byte changed to %d, want 0", dst[4])
			}
		})
	}
}

func TestH264ChromaMCAvgBilinear(t *testing.T) {
	const stride = 12
	dst := makeChromaUnitDst(stride, 6)
	src := makeChromaUnitSrc(stride, 7)

	if err := h264AvgH264ChromaMC2(dst, src, stride, 3, 3, 5); err != nil {
		t.Fatal(err)
	}
	got := [6]uint8{dst[0], dst[1], dst[stride], dst[stride+1], dst[2*stride], dst[2*stride+1]}
	want := [6]uint8{51, 53, 102, 104, 89, 91}
	if got != want {
		t.Fatalf("selected avg samples = %v, want %v", got, want)
	}
	if dst[2] != 10 {
		t.Fatalf("padding byte changed to %d, want 10", dst[2])
	}
}

func TestH264ChromaMCHighBilinearAndAvg(t *testing.T) {
	const stride = 12
	dst := make([]uint16, stride*6)
	src := make([]uint16, stride*7)
	for i := range dst {
		dst[i] = 1
		src[i] = 1023
	}

	if err := h264PutH264ChromaMC4High(dst, src, stride, 3, 3, 5, 10); err != nil {
		t.Fatal(err)
	}
	if dst[0] != 1023 || dst[3] != 1023 || dst[2*stride+3] != 1023 {
		t.Fatalf("high put samples = %d/%d/%d, want 1023", dst[0], dst[3], dst[2*stride+3])
	}

	for i := range dst {
		dst[i] = 1
	}
	if err := h264AvgH264ChromaMC2High(dst, src, stride, 3, 3, 5, 10); err != nil {
		t.Fatal(err)
	}
	if dst[0] != 512 || dst[1] != 512 || dst[2*stride+1] != 512 {
		t.Fatalf("high avg samples = %d/%d/%d, want 512", dst[0], dst[1], dst[2*stride+1])
	}
	if dst[2] != 1 {
		t.Fatalf("high padding sample changed to %d, want 1", dst[2])
	}
	if err := h264PutH264ChromaMC4High(dst, src, stride, 3, 3, 5, 11); err != ErrUnsupported {
		t.Fatalf("unsupported high chroma bit depth err = %v, want ErrUnsupported", err)
	}
}

func TestH264ChromaMCValidatesGeometry(t *testing.T) {
	if err := h264PutH264ChromaMC8(make([]uint8, 8), make([]uint8, 8), 8, 1, 8, 0); err != ErrInvalidData {
		t.Fatalf("invalid x error = %v, want ErrInvalidData", err)
	}
	if err := h264PutH264ChromaMC8(make([]uint8, 8), make([]uint8, 8), 8, 1, 0, -1); err != ErrInvalidData {
		t.Fatalf("invalid y error = %v, want ErrInvalidData", err)
	}
	if err := h264PutH264ChromaMC8(make([]uint8, 8), make([]uint8, 8), 8, 1, 1, 1); err != ErrInvalidData {
		t.Fatalf("short bilinear source error = %v, want ErrInvalidData", err)
	}
	if err := h264ChromaMC(make([]uint8, 8), make([]uint8, 16), 8, 1, 0, 0, 3, false); err != ErrInvalidData {
		t.Fatalf("invalid width error = %v, want ErrInvalidData", err)
	}
	if err := h264PutH264ChromaMC8High(make([]uint16, 8), make([]uint16, 8), 8, 1, 1, 1, 10); err != ErrInvalidData {
		t.Fatalf("short high bilinear source error = %v, want ErrInvalidData", err)
	}
	if err := h264ChromaMC(make([]uint8, 8), make([]uint8, 8), maxInt, 2, 0, 0, 8, false); err != ErrInvalidData {
		t.Fatalf("overflowed chroma geometry error = %v, want ErrInvalidData", err)
	}
	if err := h264ChromaMCHigh(make([]uint16, 8), make([]uint16, 8), maxInt, 2, 0, 0, 8, false, 10); err != ErrInvalidData {
		t.Fatalf("overflowed high chroma geometry error = %v, want ErrInvalidData", err)
	}
}

func TestH264ChromaMCRejectsCIntOverflow(t *testing.T) {
	tooLarge := intAboveCInt(t)
	if err := h264PutH264ChromaMC8(make([]uint8, 8), make([]uint8, 8), 8, tooLarge, 0, 0); err != ErrInvalidData {
		t.Fatalf("oversized C int height error = %v, want ErrInvalidData", err)
	}
	if err := h264PutH264ChromaMC8High(make([]uint16, 8), make([]uint16, 8), 8, tooLarge, 0, 0, 10); err != ErrInvalidData {
		t.Fatalf("oversized high C int height error = %v, want ErrInvalidData", err)
	}
	if err := h264PutH264ChromaMC8High(make([]uint16, 8), make([]uint16, 8), 8, 1, 0, 0, tooLarge); err != ErrInvalidData {
		t.Fatalf("oversized high C int bit depth error = %v, want ErrInvalidData", err)
	}
}

func intAboveCInt(t *testing.T) int {
	t.Helper()
	if maxInt <= maxCInt {
		t.Skip("native int cannot exceed C int on this platform")
	}
	return int(int64(maxCInt) + 1)
}

func BenchmarkH264ChromaMC8Put00(b *testing.B) {
	benchmarkH264ChromaMCCopy(b, 8, false)
}

func BenchmarkH264ChromaMC8Avg00(b *testing.B) {
	benchmarkH264ChromaMCCopy(b, 8, true)
}

func BenchmarkH264ChromaMC8Put30(b *testing.B) {
	benchmarkH264ChromaMC(b, 8, 3, 0, false)
}

func BenchmarkH264ChromaMC8Avg30(b *testing.B) {
	benchmarkH264ChromaMC(b, 8, 3, 0, true)
}

func BenchmarkH264ChromaMC8Put05(b *testing.B) {
	benchmarkH264ChromaMC(b, 8, 0, 5, false)
}

func BenchmarkH264ChromaMC8Avg05(b *testing.B) {
	benchmarkH264ChromaMC(b, 8, 0, 5, true)
}

func BenchmarkH264ChromaMC8Put35(b *testing.B) {
	benchmarkH264ChromaMC(b, 8, 3, 5, false)
}

func BenchmarkH264ChromaMC8Avg35(b *testing.B) {
	benchmarkH264ChromaMC(b, 8, 3, 5, true)
}

func BenchmarkH264ChromaMC4Put00(b *testing.B) {
	benchmarkH264ChromaMCCopy(b, 4, false)
}

func BenchmarkH264ChromaMC4Avg00(b *testing.B) {
	benchmarkH264ChromaMCCopy(b, 4, true)
}

func BenchmarkH264ChromaMC4Put30(b *testing.B) {
	benchmarkH264ChromaMC(b, 4, 3, 0, false)
}

func BenchmarkH264ChromaMC4Avg30(b *testing.B) {
	benchmarkH264ChromaMC(b, 4, 3, 0, true)
}

func BenchmarkH264ChromaMC4Put05(b *testing.B) {
	benchmarkH264ChromaMC(b, 4, 0, 5, false)
}

func BenchmarkH264ChromaMC4Avg05(b *testing.B) {
	benchmarkH264ChromaMC(b, 4, 0, 5, true)
}

func BenchmarkH264ChromaMC4Put35(b *testing.B) {
	benchmarkH264ChromaMC(b, 4, 3, 5, false)
}

func BenchmarkH264ChromaMC4Avg35(b *testing.B) {
	benchmarkH264ChromaMC(b, 4, 3, 5, true)
}

func BenchmarkH264ChromaMC2Put00(b *testing.B) {
	benchmarkH264ChromaMCCopy(b, 2, false)
}

func BenchmarkH264ChromaMC2Avg00(b *testing.B) {
	benchmarkH264ChromaMCCopy(b, 2, true)
}

func BenchmarkH264ChromaMC2Put30(b *testing.B) {
	benchmarkH264ChromaMC(b, 2, 3, 0, false)
}

func BenchmarkH264ChromaMC2Avg30(b *testing.B) {
	benchmarkH264ChromaMC(b, 2, 3, 0, true)
}

func BenchmarkH264ChromaMC2Put05(b *testing.B) {
	benchmarkH264ChromaMC(b, 2, 0, 5, false)
}

func BenchmarkH264ChromaMC2Avg05(b *testing.B) {
	benchmarkH264ChromaMC(b, 2, 0, 5, true)
}

func BenchmarkH264ChromaMC2Put35(b *testing.B) {
	benchmarkH264ChromaMC(b, 2, 3, 5, false)
}

func BenchmarkH264ChromaMC2Avg35(b *testing.B) {
	benchmarkH264ChromaMC(b, 2, 3, 5, true)
}

func BenchmarkH264ChromaMC1Put00(b *testing.B) {
	benchmarkH264ChromaMCCopy(b, 1, false)
}

func BenchmarkH264ChromaMC1Avg00(b *testing.B) {
	benchmarkH264ChromaMCCopy(b, 1, true)
}

func BenchmarkH264ChromaMC1Put30(b *testing.B) {
	benchmarkH264ChromaMC(b, 1, 3, 0, false)
}

func BenchmarkH264ChromaMC1Avg30(b *testing.B) {
	benchmarkH264ChromaMC(b, 1, 3, 0, true)
}

func BenchmarkH264ChromaMC1Put05(b *testing.B) {
	benchmarkH264ChromaMC(b, 1, 0, 5, false)
}

func BenchmarkH264ChromaMC1Avg05(b *testing.B) {
	benchmarkH264ChromaMC(b, 1, 0, 5, true)
}

func BenchmarkH264ChromaMC1Put35(b *testing.B) {
	benchmarkH264ChromaMC(b, 1, 3, 5, false)
}

func BenchmarkH264ChromaMC1Avg35(b *testing.B) {
	benchmarkH264ChromaMC(b, 1, 3, 5, true)
}

func benchmarkH264ChromaMCCopy(b *testing.B, width int, avg bool) {
	benchmarkH264ChromaMC(b, width, 0, 0, avg)
}

func benchmarkH264ChromaMC(b *testing.B, width int, x int, y int, avg bool) {
	const stride = 64
	const height = 8
	dst := makeChromaUnitDst(stride, height)
	src := makeChromaUnitSrc(stride, height+1)
	b.ReportAllocs()
	b.SetBytes(int64(width * height))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := h264ChromaMCStrides(dst, src, stride, stride, height, x, y, width, avg); err != nil {
			b.Fatal(err)
		}
	}
}

func makeChromaUnitDst(stride int, rows int) []uint8 {
	dst := make([]uint8, stride*rows)
	for i := range dst {
		dst[i] = uint8((200 - i*5) % 180)
	}
	return dst
}

func makeChromaUnitSrc(stride int, rows int) []uint8 {
	src := make([]uint8, stride*rows)
	for i := range src {
		src[i] = uint8((10 + i*9) % 200)
	}
	return src
}
