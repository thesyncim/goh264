// SPDX-License-Identifier: LGPL-2.1-or-later
//go:build !purego && arm64

package h264

import (
	"math/rand"
	"slices"
	"testing"
)

func TestH264IDCTAddASMMatchesScalar(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for stride := 4; stride <= 31; stride++ {
		for iteration := 0; iteration < 256; iteration++ {
			dst := make([]uint8, 3*stride+4)
			block := make([]int32, 16)
			for i := range dst {
				dst[i] = uint8(rng.Uint32())
			}
			for i := range block {
				block[i] = int32(rng.Uint32())
			}

			inputDst := slices.Clone(dst)
			inputBlock := slices.Clone(block)
			wantDst := slices.Clone(dst)
			wantBlock := slices.Clone(block)
			if err := h264IDCTAddScalar(wantDst, wantBlock, stride); err != nil {
				t.Fatal(err)
			}
			if err := h264IDCTAdd(dst, block, stride); err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(dst, wantDst) {
				t.Fatalf("stride %d iteration %d: destination mismatch\ninput dst   %v\ninput block %v\ngot         %v\nwant        %v", stride, iteration, inputDst, inputBlock, dst, wantDst)
			}
			if !slices.Equal(block, wantBlock) {
				t.Fatalf("stride %d iteration %d: coefficient clear mismatch\ngot  %v\nwant %v", stride, iteration, block, wantBlock)
			}
		}
	}
}

func TestH264IDCTDCAddASMMatchesScalar(t *testing.T) {
	rng := rand.New(rand.NewSource(2))
	for stride := 4; stride <= 31; stride++ {
		for iteration := 0; iteration < 256; iteration++ {
			dst := make([]uint8, 3*stride+4)
			block := make([]int32, 16)
			for i := range dst {
				dst[i] = uint8(rng.Uint32())
			}
			block[0] = int32(rng.Uint32())

			wantDst := slices.Clone(dst)
			wantBlock := slices.Clone(block)
			if err := h264IDCTDCAddScalar(wantDst, wantBlock, stride); err != nil {
				t.Fatal(err)
			}
			if err := h264IDCTDCAdd(dst, block, stride); err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(dst, wantDst) {
				t.Fatalf("stride %d iteration %d: destination mismatch\ngot  %v\nwant %v", stride, iteration, dst, wantDst)
			}
			if !slices.Equal(block, wantBlock) {
				t.Fatalf("stride %d iteration %d: coefficient clear mismatch\ngot  %v\nwant %v", stride, iteration, block, wantBlock)
			}
		}
	}
}
