// SPDX-License-Identifier: LGPL-2.1-or-later
//go:build !purego && arm64

package h264

import "testing"

func TestH264LoopFilterChromaMBAFFIntraArm64ASMMatchesScalar(t *testing.T) {
	const (
		stride = 32
		rows   = 32
		offset = 12*stride + 12
		alpha  = 20
		beta   = 20
	)
	want := makeLoopFilterUnitFixture(stride, rows)
	got := append([]uint8(nil), want...)
	seedLoopFilterChromaIntra8(want, offset, 1, stride, 1)
	seedLoopFilterChromaIntra8(got, offset, 1, stride, 1)

	if err := h264LoopFilterChromaIntra(want, offset, 1, stride, 1, alpha, beta); err != nil {
		t.Fatalf("scalar: %v", err)
	}
	h264HLoopFilterChromaMBAFFIntra8ASM(&got[offset], stride, alpha, beta)
	assertUint8SlicesEqual(t, got, want)
}
