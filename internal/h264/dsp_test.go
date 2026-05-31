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

func TestH264WeightedPixelsValidateGeometry(t *testing.T) {
	if err := h264WeightPixels(make([]uint8, 4), 4, 1, 0, 1, 0, 3); err != ErrInvalidData {
		t.Fatalf("invalid width error = %v, want ErrInvalidData", err)
	}
	if err := h264BiweightPixels(make([]uint8, 4), make([]uint8, 3), 4, 1, 0, 1, 1, 0, 4); err != ErrInvalidData {
		t.Fatalf("short src error = %v, want ErrInvalidData", err)
	}
}
