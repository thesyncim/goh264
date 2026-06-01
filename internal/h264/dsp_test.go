// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

import "testing"

func TestH264AddPixels4ClearWrapsAndClears(t *testing.T) {
	dst := []uint8{
		250, 1, 2, 3,
		4, 5, 6, 7,
		8, 9, 10, 11,
		12, 13, 14, 15,
	}
	block := []int32{
		10, -2, 300, -300,
		1, 2, 3, 4,
		-5, -6, -7, -8,
		255, 256, -255, -256,
	}

	if err := h264AddPixels4Clear(dst, block, 4); err != nil {
		t.Fatal(err)
	}
	want := []uint8{
		4, 255, 46, 215,
		5, 7, 9, 11,
		3, 3, 3, 3,
		11, 13, 15, 15,
	}
	for i := range want {
		if dst[i] != want[i] {
			t.Fatalf("dst[%d] = %d, want %d", i, dst[i], want[i])
		}
	}
	for i, coeff := range block {
		if coeff != 0 {
			t.Fatalf("block[%d] = %d, want cleared", i, coeff)
		}
	}
}

func TestH264AddPixels8ClearUsesStrideAndClears(t *testing.T) {
	dst := make([]uint8, 10*8)
	block := make([]int32, 64)
	for i := range dst {
		dst[i] = uint8(40 + i)
	}
	for i := range block {
		block[i] = int32(i - 32)
	}

	if err := h264AddPixels8Clear(dst, block, 10); err != nil {
		t.Fatal(err)
	}
	if dst[0] != 8 || dst[7] != 22 || dst[10] != 26 || dst[77] != 148 {
		t.Fatalf("selected dst = %d/%d/%d/%d, want 8/22/26/148", dst[0], dst[7], dst[10], dst[77])
	}
	if dst[8] != 48 || dst[9] != 49 {
		t.Fatalf("padding bytes changed: %d/%d", dst[8], dst[9])
	}
	for i, coeff := range block {
		if coeff != 0 {
			t.Fatalf("block[%d] = %d, want cleared", i, coeff)
		}
	}
}

func TestH264AddPixelsHighWrapsAndClears(t *testing.T) {
	dst := []uint16{
		65530, 1, 2, 3,
		4, 5, 6, 7,
		8, 9, 10, 11,
		12, 13, 14, 15,
	}
	block := []int32{
		10, -2, 70000, -70000,
		1, 2, 3, 4,
		-5, -6, -7, -8,
		32767, 32768, -32767, -32768,
	}

	if err := h264AddPixels4ClearHigh(dst, block, 4); err != nil {
		t.Fatal(err)
	}
	want := []uint16{
		4, 65535, 4466, 61075,
		5, 7, 9, 11,
		3, 3, 3, 3,
		32779, 32781, 32783, 32783,
	}
	for i := range want {
		if dst[i] != want[i] {
			t.Fatalf("dst[%d] = %d, want %d", i, dst[i], want[i])
		}
	}
	for i, coeff := range block {
		if coeff != 0 {
			t.Fatalf("block[%d] = %d, want cleared", i, coeff)
		}
	}
}

func TestH264WeightPixelsClipsAndRounds(t *testing.T) {
	dst := []uint8{10, 20, 250, 255}

	if err := h264WeightPixels(dst, 4, 1, 1, 2, 1, 4); err != nil {
		t.Fatal(err)
	}
	want := []uint8{11, 21, 251, 255}
	for i := range want {
		if dst[i] != want[i] {
			t.Fatalf("dst[%d] = %d, want %d", i, dst[i], want[i])
		}
	}

	if err := h264WeightPixels(dst, 4, 1, 0, -3, 0, 4); err != nil {
		t.Fatal(err)
	}
	for i, value := range dst {
		if value != 0 {
			t.Fatalf("negative weight dst[%d] = %d, want clipped zero", i, value)
		}
	}
}

func TestH264WeightPixelsHighClipsAndRounds(t *testing.T) {
	dst := []uint16{10, 1000, 1023, 512}

	if err := h264WeightPixelsHigh(dst, 4, 1, 1, 2, 1, 4, 10); err != nil {
		t.Fatal(err)
	}
	want := []uint16{14, 1004, 1023, 516}
	for i := range want {
		if dst[i] != want[i] {
			t.Fatalf("dst[%d] = %d, want %d", i, dst[i], want[i])
		}
	}

	if err := h264WeightPixelsHigh(dst, 4, 1, 0, -3, 0, 4, 10); err != nil {
		t.Fatal(err)
	}
	for i, value := range dst {
		if value != 0 {
			t.Fatalf("negative high-bit-depth weight dst[%d] = %d, want clipped zero", i, value)
		}
	}
}

func TestH264BiweightPixelsClipsAndRounds(t *testing.T) {
	dst := []uint8{10, 20, 250, 255}
	src := []uint8{20, 40, 10, 255}

	if err := h264BiweightPixels(dst, src, 4, 1, 1, 2, 2, 0, 4); err != nil {
		t.Fatal(err)
	}
	want := []uint8{15, 30, 130, 255}
	for i := range want {
		if dst[i] != want[i] {
			t.Fatalf("dst[%d] = %d, want %d", i, dst[i], want[i])
		}
	}
}

func TestH264BiweightPixelsHighClipsAndRounds(t *testing.T) {
	dst := []uint16{10, 20, 1000, 1023}
	src := []uint16{20, 40, 10, 1023}

	if err := h264BiweightPixelsHigh(dst, src, 4, 1, 1, 2, 2, 0, 4, 10); err != nil {
		t.Fatal(err)
	}
	want := []uint16{15, 30, 505, 1023}
	for i := range want {
		if dst[i] != want[i] {
			t.Fatalf("dst[%d] = %d, want %d", i, dst[i], want[i])
		}
	}
}

func TestH264LoopFilterLumaMutatesBoundary(t *testing.T) {
	const stride = 16
	pix := makeLoopFilterUnitFixture(stride, 16)
	offset := 8*stride + 4
	pix[offset-3*stride] = 98
	pix[offset-2*stride] = 100
	pix[offset-1*stride] = 102
	pix[offset] = 108
	pix[offset+stride] = 110
	pix[offset+2*stride] = 112
	tc0 := [4]int8{2, -1, 0, 1}

	if err := h264VLoopFilterLuma(pix, offset, stride, 20, 20, &tc0); err != nil {
		t.Fatal(err)
	}
	if pix[offset-2*stride] != 101 || pix[offset-1*stride] != 104 || pix[offset] != 106 || pix[offset+stride] != 108 {
		t.Fatalf("filtered luma edge = %d/%d/%d/%d, want 101/104/106/108",
			pix[offset-2*stride], pix[offset-1*stride], pix[offset], pix[offset+stride])
	}
}

func TestH264LoopFilterLumaHighMutatesBoundary(t *testing.T) {
	const stride = 16
	pix := makeLoopFilterHighUnitFixture(stride, 16)
	offset := 8*stride + 4
	pix[offset-3*stride] = 98 << 2
	pix[offset-2*stride] = 100 << 2
	pix[offset-1*stride] = 102 << 2
	pix[offset] = 108 << 2
	pix[offset+stride] = 110 << 2
	pix[offset+2*stride] = 112 << 2
	tc0 := [4]int8{2, -1, 0, 1}

	if err := h264VLoopFilterLumaHigh(pix, offset, stride, 20, 20, &tc0, 10); err != nil {
		t.Fatal(err)
	}
	if pix[offset-2*stride] != 406 || pix[offset-1*stride] != 415 || pix[offset] != 425 || pix[offset+stride] != 434 {
		t.Fatalf("filtered high luma edge = %d/%d/%d/%d, want 406/415/425/434",
			pix[offset-2*stride], pix[offset-1*stride], pix[offset], pix[offset+stride])
	}
}

func TestH264LoopFilterLumaIntraStrongPath(t *testing.T) {
	const stride = 16
	pix := makeLoopFilterUnitFixture(stride, 16)
	offset := 8*stride + 4
	pix[offset-4*stride] = 96
	pix[offset-3*stride] = 98
	pix[offset-2*stride] = 100
	pix[offset-1*stride] = 102
	pix[offset] = 108
	pix[offset+stride] = 110
	pix[offset+2*stride] = 112
	pix[offset+3*stride] = 114

	if err := h264VLoopFilterLumaIntra(pix, offset, stride, 200, 20); err != nil {
		t.Fatal(err)
	}
	if pix[offset-3*stride] != 100 || pix[offset-2*stride] != 102 || pix[offset-1*stride] != 104 ||
		pix[offset] != 107 || pix[offset+stride] != 108 || pix[offset+2*stride] != 111 {
		t.Fatalf("filtered intra luma edge = %d/%d/%d/%d/%d/%d, want 100/102/104/107/108/111",
			pix[offset-3*stride], pix[offset-2*stride], pix[offset-1*stride],
			pix[offset], pix[offset+stride], pix[offset+2*stride])
	}
}

func TestH264LoopFilterChromaMutatesBoundary(t *testing.T) {
	const stride = 16
	pix := makeLoopFilterUnitFixture(stride, 16)
	offset := 8*stride + 4
	pix[offset-2*stride] = 100
	pix[offset-1*stride] = 102
	pix[offset] = 108
	pix[offset+stride] = 110
	tc0 := [4]int8{2, -1, 0, 1}

	if err := h264VLoopFilterChroma(pix, offset, stride, 20, 20, &tc0); err != nil {
		t.Fatal(err)
	}
	if pix[offset-1*stride] != 104 || pix[offset] != 106 {
		t.Fatalf("filtered chroma edge = %d/%d, want 104/106", pix[offset-1*stride], pix[offset])
	}
}

func TestH264WeightedPixelsValidateGeometry(t *testing.T) {
	if err := h264WeightPixels(make([]uint8, 4), 4, 1, 0, 1, 0, 3); err != ErrInvalidData {
		t.Fatalf("invalid width error = %v, want ErrInvalidData", err)
	}
	if err := h264BiweightPixels(make([]uint8, 4), make([]uint8, 3), 4, 1, 0, 1, 1, 0, 4); err != ErrInvalidData {
		t.Fatalf("short src error = %v, want ErrInvalidData", err)
	}
	if err := h264VLoopFilterLuma(make([]uint8, 64), 0, 8, 20, 20, &[4]int8{}); err != ErrInvalidData {
		t.Fatalf("short loop-filter margin error = %v, want ErrInvalidData", err)
	}
	if err := h264VLoopFilterChroma(make([]uint8, 64), 32, 8, 20, 20, nil); err != ErrInvalidData {
		t.Fatalf("nil tc0 error = %v, want ErrInvalidData", err)
	}
	if err := h264WeightPixelsHigh(make([]uint16, 4), 4, 1, 0, 1, 0, 4, 11); err != ErrUnsupported {
		t.Fatalf("unsupported high bit depth error = %v, want ErrUnsupported", err)
	}
	if err := h264BiweightPixelsHigh(make([]uint16, 4), make([]uint16, 3), 4, 1, 0, 1, 1, 0, 4, 10); err != ErrInvalidData {
		t.Fatalf("short high src error = %v, want ErrInvalidData", err)
	}
	if err := h264VLoopFilterLumaHigh(make([]uint16, 64), 0, 8, 20, 20, &[4]int8{}, 10); err != ErrInvalidData {
		t.Fatalf("short high loop-filter margin error = %v, want ErrInvalidData", err)
	}
}

func makeLoopFilterUnitFixture(stride int, rows int) []uint8 {
	pix := make([]uint8, stride*rows)
	for i := range pix {
		pix[i] = uint8(30 + (i*7)%180)
	}
	return pix
}

func makeLoopFilterHighUnitFixture(stride int, rows int) []uint16 {
	pix := make([]uint16, stride*rows)
	for i := range pix {
		pix[i] = uint16(120 + (i*7)%720)
	}
	return pix
}
