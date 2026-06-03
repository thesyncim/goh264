// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

import "testing"

func TestH264Pred16x16DCAndPlane(t *testing.T) {
	const stride = 24
	const offset = 4*stride + 4
	pix := makePredictionFixture(stride, 24)

	if err := h264Pred16x16DC(pix, offset, stride); err != nil {
		t.Fatal(err)
	}
	wantDC := uint8(117)
	if pix[offset] != wantDC || pix[offset+15] != wantDC || pix[offset+15*stride+15] != wantDC {
		t.Fatalf("pred16x16 dc samples = %d/%d/%d, want %d",
			pix[offset], pix[offset+15], pix[offset+15*stride+15], wantDC)
	}

	pix = makePredictionFixture(stride, 24)
	if err := h264Pred16x16Plane(pix, offset, stride); err != nil {
		t.Fatal(err)
	}
	if pix[offset] != 78 || pix[offset+15] != 153 || pix[offset+15*stride] != 183 || pix[offset+15*stride+15] != 255 {
		t.Fatalf("pred16x16 plane corners = %d/%d/%d/%d, want 78/153/183/255",
			pix[offset], pix[offset+15], pix[offset+15*stride], pix[offset+15*stride+15])
	}
}

func TestH264PredHighDC128AndValidation(t *testing.T) {
	const stride = 24
	const offset = 4*stride + 4
	pix := make([]uint16, stride*24)

	if err := h264Pred16x16DC128High(pix, offset, stride, 10); err != nil {
		t.Fatal(err)
	}
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			if got := pix[offset+y*stride+x]; got != 512 {
				t.Fatalf("10-bit dc128 sample (%d,%d) = %d, want 512", x, y, got)
			}
		}
	}
	if err := h264Pred16x16DC128High(pix, offset, stride, 8); err != ErrUnsupported {
		t.Fatalf("8-bit high predictor error = %v, want ErrUnsupported", err)
	}
	if err := h264Pred16x16PlaneHigh(make([]uint16, 16*16), 0, 16, 10); err != ErrInvalidData {
		t.Fatalf("missing high plane margins error = %v, want ErrInvalidData", err)
	}
}

func TestH264Pred8x8DCQuadrants(t *testing.T) {
	const stride = 16
	const offset = 4*stride + 4
	pix := makePredictionFixture(stride, 16)

	if err := h264Pred8x8DC(pix, offset, stride); err != nil {
		t.Fatal(err)
	}
	if pix[offset] != 81 || pix[offset+4] != 99 || pix[offset+4*stride] != 112 || pix[offset+4*stride+4] != 105 {
		t.Fatalf("pred8x8 dc quadrants = %d/%d/%d/%d, want 81/99/112/105",
			pix[offset], pix[offset+4], pix[offset+4*stride], pix[offset+4*stride+4])
	}
}

func TestH264Pred8x16Plane(t *testing.T) {
	const stride = 16
	const offset = 4*stride + 4
	pix := makePredictionFixture(stride, 24)

	if err := h264Pred8x16Plane(pix, offset, stride); err != nil {
		t.Fatal(err)
	}
	if pix[offset] != 78 || pix[offset+7] != 113 || pix[offset+15*stride] != 183 || pix[offset+15*stride+7] != 218 {
		t.Fatalf("pred8x16 plane corners = %d/%d/%d/%d, want 78/113/183/218",
			pix[offset], pix[offset+7], pix[offset+15*stride], pix[offset+15*stride+7])
	}
}

func TestH264PredChromaMadCowDispatch(t *testing.T) {
	for _, tc := range []struct {
		name         string
		chromaFormat int
		mode         int
		rows         int
		height       int
		fn           h264PredFunc
	}{
		{"420 l0t", 1, intraPred8x8AlzheimerL0TDC, 16, 8, h264Pred8x8MadCowDCL0T},
		{"420 0lt", 1, intraPred8x8Alzheimer0LTDC, 16, 8, h264Pred8x8MadCowDC0LT},
		{"420 l00", 1, intraPred8x8AlzheimerL00DC, 16, 8, h264Pred8x8MadCowDCL00},
		{"420 0l0", 1, intraPred8x8Alzheimer0L0DC, 16, 8, h264Pred8x8MadCowDC0L0},
		{"422 l0t", 2, intraPred8x8AlzheimerL0TDC, 24, 16, h264Pred8x16MadCowDCL0T},
		{"422 0lt", 2, intraPred8x8Alzheimer0LTDC, 24, 16, h264Pred8x16MadCowDC0LT},
		{"422 l00", 2, intraPred8x8AlzheimerL00DC, 24, 16, h264Pred8x16MadCowDCL00},
		{"422 0l0", 2, intraPred8x8Alzheimer0L0DC, 24, 16, h264Pred8x16MadCowDC0L0},
	} {
		const stride = 16
		const offset = 4*stride + 4
		got := makePredictionFixture(stride, tc.rows)
		want := makePredictionFixture(stride, tc.rows)
		if err := h264PredChromaByMode(got, offset, stride, tc.chromaFormat, tc.mode); err != nil {
			t.Fatalf("%s dispatch: %v", tc.name, err)
		}
		if err := tc.fn(want, offset, stride); err != nil {
			t.Fatalf("%s direct: %v", tc.name, err)
		}
		for y := 0; y < tc.height; y++ {
			for x := 0; x < 8; x++ {
				i := offset + y*stride + x
				if got[i] != want[i] {
					t.Fatalf("%s sample (%d,%d) = %d, want %d", tc.name, x, y, got[i], want[i])
				}
			}
		}
	}
}

func TestH264Pred4x4AngularModes(t *testing.T) {
	const stride = 12
	const offset = 3*stride + 3
	topRight := []uint8{91, 123, 155, 177}

	pix := makePredictionFixture(stride, 12)
	if err := h264Pred4x4DownLeft(pix, offset, stride, topRight); err != nil {
		t.Fatal(err)
	}
	if pix[offset] != 64 || pix[offset+3] != 95 || pix[offset+3*stride] != 95 || pix[offset+3*stride+3] != 172 {
		t.Fatalf("pred4x4 down-left corners = %d/%d/%d/%d, want 64/95/95/172",
			pix[offset], pix[offset+3], pix[offset+3*stride], pix[offset+3*stride+3])
	}

	pix = makePredictionFixture(stride, 12)
	if err := h264Pred4x4VerticalLeft(pix, offset, stride, topRight); err != nil {
		t.Fatal(err)
	}
	if pix[offset] != 62 || pix[offset+3] != 83 || pix[offset+3*stride] != 69 || pix[offset+3*stride+3] != 123 {
		t.Fatalf("pred4x4 vertical-left corners = %d/%d/%d/%d, want 62/83/69/123",
			pix[offset], pix[offset+3], pix[offset+3*stride], pix[offset+3*stride+3])
	}
}

func TestH264Pred8x8LFilteredEdges(t *testing.T) {
	const stride = 28
	const offset = 5*stride + 5

	pix := makePredictionFixture(stride, 18)
	if err := h264Pred8x8LVerticalLeft(pix, offset, stride, true, true); err != nil {
		t.Fatal(err)
	}
	if pix[offset] != 86 || pix[offset+7] != 121 || pix[offset+7*stride] != 103 || pix[offset+7*stride+7] != 138 {
		t.Fatalf("pred8x8l vertical-left corners = %d/%d/%d/%d, want 86/121/103/138",
			pix[offset], pix[offset+7], pix[offset+7*stride], pix[offset+7*stride+7])
	}

	pix = makePredictionFixture(stride, 18)
	if err := h264Pred8x8LDownLeft(pix, offset, stride, false, false); err != nil {
		t.Fatal(err)
	}
	if pix[offset] != 88 || pix[offset+7] != 118 || pix[offset+7*stride] != 118 || pix[offset+7*stride+7] != 118 {
		t.Fatalf("pred8x8l down-left no-topright corners = %d/%d/%d/%d, want 88/118/118/118",
			pix[offset], pix[offset+7], pix[offset+7*stride], pix[offset+7*stride+7])
	}

	pix = makePredictionFixture(stride, 18)
	if err := h264Pred8x8LDC(pix, offset, stride, false, false); err != nil {
		t.Fatal(err)
	}
	if pix[offset] != 105 || pix[offset+7*stride+7] != 105 {
		t.Fatalf("pred8x8l dc samples = %d/%d, want 105/105", pix[offset], pix[offset+7*stride+7])
	}
}

func TestH264Pred8x8LFilterAddWrapsAndClears(t *testing.T) {
	const stride = 28
	const offset = 5*stride + 5

	pix := makePredictionFixture(stride, 18)
	block := makePredictionBlock(64)
	if err := h264Pred8x8LVerticalFilterAdd(pix, offset, block, stride, true, true); err != nil {
		t.Fatal(err)
	}
	if pix[offset] != 74 || pix[offset+7] != 109 || pix[offset+7*stride] != 74 || pix[offset+7*stride+7] != 109 {
		t.Fatalf("pred8x8l vertical filter-add corners = %d/%d/%d/%d, want 74/109/74/109",
			pix[offset], pix[offset+7], pix[offset+7*stride], pix[offset+7*stride+7])
	}
	for i, coeff := range block {
		if coeff != 0 {
			t.Fatalf("vertical filter-add block[%d] = %d, want cleared", i, coeff)
		}
	}

	pix = makePredictionFixture(stride, 18)
	block = makePredictionBlock(64)
	if err := h264Pred8x8LHorizontalFilterAdd(pix, offset, block, stride, false, false); err != nil {
		t.Fatal(err)
	}
	if pix[offset] != 78 || pix[offset+7] != 78 || pix[offset+7*stride] != 123 || pix[offset+7*stride+7] != 123 {
		t.Fatalf("pred8x8l horizontal filter-add corners = %d/%d/%d/%d, want 78/78/123/123",
			pix[offset], pix[offset+7], pix[offset+7*stride], pix[offset+7*stride+7])
	}
	for i, coeff := range block {
		if coeff != 0 {
			t.Fatalf("horizontal filter-add block[%d] = %d, want cleared", i, coeff)
		}
	}
}

func TestH264Pred8x8LAddUsesUnfilteredEdgesAndClears(t *testing.T) {
	const stride = 28
	const offset = 5*stride + 5

	pix := makePredictionFixture(stride, 18)
	block := makePredictionBlock(64)
	wantCol := [8]uint8{}
	v := pix[offset-stride]
	for y := 0; y < 8; y++ {
		v += uint8(dctcoef8Value(block[y*8]))
		wantCol[y] = v
	}
	if err := h264Pred8x8LVerticalAdd(pix, offset, block, stride); err != nil {
		t.Fatal(err)
	}
	for y, want := range wantCol {
		if got := pix[offset+y*stride]; got != want {
			t.Fatalf("vertical add y=%d got=%d want=%d", y, got, want)
		}
	}
	for i, coeff := range block {
		if coeff != 0 {
			t.Fatalf("vertical add block[%d] = %d, want cleared", i, coeff)
		}
	}

	pix = makePredictionFixture(stride, 18)
	block = makePredictionBlock(64)
	wantRow := [8]uint8{}
	v = pix[offset-1]
	for x := 0; x < 8; x++ {
		v += uint8(dctcoef8Value(block[x]))
		wantRow[x] = v
	}
	if err := h264Pred8x8LHorizontalAdd(pix, offset, block, stride); err != nil {
		t.Fatal(err)
	}
	for x, want := range wantRow {
		if got := pix[offset+x]; got != want {
			t.Fatalf("horizontal add x=%d got=%d want=%d", x, got, want)
		}
	}
	for i, coeff := range block {
		if coeff != 0 {
			t.Fatalf("horizontal add block[%d] = %d, want cleared", i, coeff)
		}
	}
}

func TestH264Pred8x8LHighAddUsesUnfilteredEdgesAndClears(t *testing.T) {
	const stride = 12
	const offset = 3*stride + 3
	pix := make([]uint16, stride*12)
	for i := range pix {
		pix[i] = uint16(1000 + i)
	}
	block := makePredictionBlock(64)
	want := [8]uint16{}
	v := pix[offset-1]
	for x := 0; x < 8; x++ {
		v += uint16(uint32(block[x]))
		want[x] = v
	}
	if err := h264Pred8x8LHorizontalAddHigh(pix, offset, block, stride, 10); err != nil {
		t.Fatal(err)
	}
	for x, w := range want {
		if got := pix[offset+x]; got != w {
			t.Fatalf("high horizontal add x=%d got=%d want=%d", x, got, w)
		}
	}
	for i, coeff := range block {
		if coeff != 0 {
			t.Fatalf("high horizontal add block[%d] = %d, want cleared", i, coeff)
		}
	}
}

func TestH264Pred4x4AddWrapsAndClears(t *testing.T) {
	const stride = 8
	const offset = 2*stride + 2
	pix := makePredictionFixture(stride, 8)
	block := []int32{
		10, -2, 300, -300,
		1, 2, 3, 4,
		-5, -6, -7, -8,
		255, 256, -255, -256,
	}

	if err := h264Pred4x4VerticalAdd(pix, offset, block, stride); err != nil {
		t.Fatal(err)
	}
	if pix[offset] != 57 || pix[offset+1] != 50 || pix[offset+2] != 101 || pix[offset+3] != 18 ||
		pix[offset+3*stride] != 52 || pix[offset+3*stride+3] != 14 {
		t.Fatalf("vertical add samples = %d/%d/%d/%d/%d/%d",
			pix[offset], pix[offset+1], pix[offset+2], pix[offset+3],
			pix[offset+3*stride], pix[offset+3*stride+3])
	}
	for i, coeff := range block {
		if coeff != 0 {
			t.Fatalf("block[%d] = %d, want cleared", i, coeff)
		}
	}
}

func TestH264Pred4x4HighAddWrapsAndClears(t *testing.T) {
	const stride = 8
	const offset = 2*stride + 2
	pix := make([]uint16, stride*8)
	pix[offset-stride+0] = 1020
	pix[offset-stride+1] = 1
	pix[offset-stride+2] = 1023
	pix[offset-stride+3] = 5
	block := []int32{
		10, -4, 1, -6,
		0, 0, 0, 0,
		0, 0, 0, 0,
		0, 0, 0, 0,
	}

	if err := h264Pred4x4VerticalAddHigh(pix, offset, block, stride, 10); err != nil {
		t.Fatal(err)
	}
	if pix[offset] != 1030 || pix[offset+1] != 65533 || pix[offset+2] != 1024 || pix[offset+3] != 65535 {
		t.Fatalf("high vertical add row = %d/%d/%d/%d, want 1030/65533/1024/65535",
			pix[offset], pix[offset+1], pix[offset+2], pix[offset+3])
	}
	for i, coeff := range block {
		if coeff != 0 {
			t.Fatalf("high block[%d] = %d, want cleared", i, coeff)
		}
	}
}

func TestH264Pred16x16AddDispatchesBlockOffsets(t *testing.T) {
	const stride = 24
	offsets, err := h264FrameBlockOffsets(stride, stride, 0)
	if err != nil {
		t.Fatal(err)
	}
	base := 4*stride + 4
	for i := range offsets {
		offsets[i] += base
	}
	pix := makePredictionFixture(stride, 24)
	block := make([]int32, 48*16)
	block[0] = 1
	block[15*16+15] = 2

	if err := h264Pred16x16HorizontalAdd(pix, &offsets, block, stride); err != nil {
		t.Fatal(err)
	}
	if pix[offsets[0]] != pix[offsets[0]-1]+1 {
		t.Fatalf("first block did not use left predictor plus residual")
	}
	if block[0] != 0 || block[15*16+15] != 0 {
		t.Fatalf("block coefficients not cleared: %d/%d", block[0], block[15*16+15])
	}
}

func TestH264PredictionValidatesMargins(t *testing.T) {
	if err := h264Pred16x16Plane(make([]uint8, 16*16), 0, 16); err != ErrInvalidData {
		t.Fatalf("missing top/left margin error = %v, want ErrInvalidData", err)
	}
	if err := h264Pred4x4VerticalAdd(make([]uint8, 16), 0, make([]int32, 16), 4); err != ErrInvalidData {
		t.Fatalf("missing top margin add error = %v, want ErrInvalidData", err)
	}
}

func makePredictionFixture(stride int, rows int) []uint8 {
	pix := make([]uint8, stride*rows)
	for y := 0; y < rows; y++ {
		for x := 0; x < stride; x++ {
			pix[y*stride+x] = uint8(30 + (x*5+y*7)%180)
		}
	}
	return pix
}
