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
