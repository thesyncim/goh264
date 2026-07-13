// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

import (
	"errors"
	"fmt"
	"testing"
)

const (
	h264MotionCompTestMBWidth      = 4
	h264MotionCompTestMBHeight     = 4
	h264MotionCompTestLumaStride   = 80
	h264MotionCompTestChromaStride = 48
)

func TestH264HLMotionFrameBipredUsesAvgForSecondList(t *testing.T) {
	dst := makeH264MotionCompPicture(1, 3)
	ref0 := makeH264MotionCompPicture(1, 41)
	ref1 := makeH264MotionCompPicture(1, 83)
	refs := [2][]*h264PicturePlanes{{ref0}, {ref1}}
	var cache macroblockMotionCache
	cache.Ref[0][h264Scan8[0]] = 0
	cache.Ref[1][h264Scan8[0]] = 0

	const mbX = 1
	const mbY = 1
	mbType := MBType16x16 | MBTypeP0L0 | MBTypeP0L1
	if err := h264HLMotionFrame(dst, refs, &cache, mbType, [4]uint32{}, mbX, mbY, 2); err != nil {
		t.Fatal(err)
	}

	yOff := mbY*16*dst.LumaStride + mbX*16
	wantY := uint8((int(ref0.Y[yOff]) + int(ref1.Y[yOff]) + 1) >> 1)
	if dst.Y[yOff] != wantY {
		t.Fatalf("bipred luma sample = %d, want %d", dst.Y[yOff], wantY)
	}
	cOff := mbY*8*dst.ChromaStride + mbX*8
	wantCb := uint8((int(ref0.Cb[cOff]) + int(ref1.Cb[cOff]) + 1) >> 1)
	wantCr := uint8((int(ref0.Cr[cOff]) + int(ref1.Cr[cOff]) + 1) >> 1)
	if dst.Cb[cOff] != wantCb || dst.Cr[cOff] != wantCr {
		t.Fatalf("bipred chroma samples = %d/%d, want %d/%d", dst.Cb[cOff], dst.Cr[cOff], wantCb, wantCr)
	}
}

func TestH264EmulatedEdgeMCRejectsOverflowedGeometry(t *testing.T) {
	if err := h264EmulatedEdgeMC(nil, maxInt-1, maxInt, []uint8{0, 1}, 1, 1, 2, 0, 0, 1, 2); err != ErrInvalidData {
		t.Fatalf("overflowed edge buffer geometry error = %v, want ErrInvalidData", err)
	}
	if err := h264EmulatedEdgeMC(make([]uint8, 2), 0, 1, nil, maxInt, 1, 2, 0, 0, 1, 2); err != ErrInvalidData {
		t.Fatalf("overflowed edge source geometry error = %v, want ErrInvalidData", err)
	}
}

func TestH264EmulatedEdgeMCMatchesClampedPixelOracle(t *testing.T) {
	const (
		width     = 32
		height    = 27
		srcStride = 40
		bufStride = 37
		bufOffset = 5
	)
	src := make([]uint8, srcStride*height)
	fillH264MotionCompPlane(src, 29)
	xs := []int{-40, -21, -20, -3, -2, -1, 0, 1, 15, 29, 30, 31, 32, 33, 50}
	ys := []int{-40, -21, -20, -3, -2, -1, 0, 1, 13, 24, 25, 26, 27, 28, 45}
	for _, shape := range []struct {
		name           string
		blockW, blockH int
	}{
		{name: "Luma21x21", blockW: 21, blockH: 21},
		{name: "Chroma9x9", blockW: 9, blockH: 9},
		{name: "Chroma422_9x17", blockW: 9, blockH: 17},
	} {
		for _, srcY := range ys {
			for _, srcX := range xs {
				name := fmt.Sprintf("%s/x%d/y%d", shape.name, srcX, srcY)
				t.Run(name, func(t *testing.T) {
					buf := make([]uint8, bufOffset+(shape.blockH-1)*bufStride+shape.blockW+7)
					for i := range buf {
						buf[i] = 0xcd
					}
					if err := h264EmulatedEdgeMC(buf, bufOffset, bufStride, src, srcStride, shape.blockW, shape.blockH, srcX, srcY, width, height); err != nil {
						t.Fatal(err)
					}
					for y := 0; y < shape.blockH; y++ {
						wantY := min(max(srcY+y, 0), height-1)
						for x := 0; x < shape.blockW; x++ {
							wantX := min(max(srcX+x, 0), width-1)
							got := buf[bufOffset+y*bufStride+x]
							want := src[wantY*srcStride+wantX]
							if got != want {
								t.Fatalf("pixel (%d,%d) = %d, want %d", x, y, got, want)
							}
						}
					}
				})
			}
		}
	}
}

func TestH264EmulatedEdgeMCHighRejectsOverflowedGeometry(t *testing.T) {
	if err := h264EmulatedEdgeMCHigh(nil, maxInt-1, maxInt, []uint16{0, 1}, 1, 1, 2, 0, 0, 1, 2); err != ErrInvalidData {
		t.Fatalf("overflowed high edge buffer geometry error = %v, want ErrInvalidData", err)
	}
	if err := h264EmulatedEdgeMCHigh(make([]uint16, 2), 0, 1, nil, maxInt, 1, 2, 0, 0, 1, 2); err != ErrInvalidData {
		t.Fatalf("overflowed high edge source geometry error = %v, want ErrInvalidData", err)
	}
}

func TestH264EdgeScratchRejectsOverflowedGeometry(t *testing.T) {
	if _, _, err := h264EdgeScratch(&h264MotionCompScratch{Edge: make([]uint8, 1)}, maxInt/8+1, 9, 9); err != ErrInvalidData {
		t.Fatalf("overflowed edge scratch error = %v, want ErrInvalidData", err)
	}
}

func TestH264EdgeScratchHighRejectsOverflowedGeometry(t *testing.T) {
	if _, _, err := h264EdgeScratchHigh(&h264MotionCompScratchHigh{Edge: make([]uint16, 1)}, maxInt/8+1, 9, 9); err != ErrInvalidData {
		t.Fatalf("overflowed high edge scratch error = %v, want ErrInvalidData", err)
	}
}

func BenchmarkH264EmulatedEdgeMC(b *testing.B) {
	for _, c := range []struct {
		name string
		srcX int
		srcY int
	}{
		{name: "Inside21x21", srcX: 16, srcY: 16},
		{name: "TopLeft21x21", srcX: -2, srcY: -2},
		{name: "BottomRight21x21", srcX: 45, srcY: 45},
	} {
		b.Run(c.name, func(b *testing.B) {
			benchmarkH264EmulatedEdgeMC(b, c.srcX, c.srcY)
		})
	}
}

func BenchmarkH264EmulatedEdgeMCHigh10(b *testing.B) {
	for _, c := range []struct {
		name string
		srcX int
		srcY int
	}{
		{name: "Inside21x21", srcX: 16, srcY: 16},
		{name: "TopLeft21x21", srcX: -2, srcY: -2},
		{name: "BottomRight21x21", srcX: 45, srcY: 45},
	} {
		b.Run(c.name, func(b *testing.B) {
			benchmarkH264EmulatedEdgeMCHigh(b, c.srcX, c.srcY)
		})
	}
}

func TestH264HLMotionFrameList1OnlyUsesPut(t *testing.T) {
	dst := makeH264MotionCompPicture(1, 17)
	ref1 := makeH264MotionCompPicture(1, 99)
	refs := [2][]*h264PicturePlanes{nil, {ref1}}
	var cache macroblockMotionCache
	cache.Ref[1][h264Scan8[0]] = 0

	const mbX = 1
	const mbY = 1
	mbType := MBType16x16 | MBTypeP0L1
	if err := h264HLMotionFrame(dst, refs, &cache, mbType, [4]uint32{}, mbX, mbY, 2); err != nil {
		t.Fatal(err)
	}

	yOff := mbY*16*dst.LumaStride + mbX*16
	if dst.Y[yOff] != ref1.Y[yOff] {
		t.Fatalf("list1-only luma sample = %d, want put %d", dst.Y[yOff], ref1.Y[yOff])
	}
}

func TestH264HLMotionFrameWeightedExplicitList0(t *testing.T) {
	dst := makeH264MotionCompPicture(1, 23)
	want := makeH264MotionCompPicture(1, 23)
	ref0 := makeH264MotionCompPicture(1, 91)
	refs := [2][]*h264PicturePlanes{{ref0}}
	var cache macroblockMotionCache
	cache.Ref[0][h264Scan8[0]] = 0

	pwt := h264MotionCompTestPWT(1)
	pwt.UseWeight = 1
	pwt.UseWeightChroma = 1
	pwt.LumaLog2WeightDenom = 2
	pwt.ChromaLog2WeightDenom = 1
	pwt.LumaWeight[0][0] = [2]int32{3, -2}
	pwt.ChromaWeight[0][0][0] = [2]int32{2, 1}
	pwt.ChromaWeight[0][0][1] = [2]int32{-1, 3}

	const mbX = 1
	const mbY = 1
	mbType := MBType16x16 | MBTypeP0L0
	if err := h264HLMotionFrameWeighted(dst, refs, &cache, mbType, [4]uint32{}, mbX, mbY, 1, &pwt, nil); err != nil {
		t.Fatal(err)
	}
	if err := h264HLMotionFrame(want, refs, &cache, mbType, [4]uint32{}, mbX, mbY, 1); err != nil {
		t.Fatal(err)
	}
	yOff := mbY*16*want.LumaStride + mbX*16
	if err := h264WeightPixels(want.Y[yOff:], want.LumaStride, 16, 2, 3, -2, 16); err != nil {
		t.Fatal(err)
	}
	cOff := mbY*8*want.ChromaStride + mbX*8
	if err := h264WeightPixels(want.Cb[cOff:], want.ChromaStride, 8, 1, 2, 1, 8); err != nil {
		t.Fatal(err)
	}
	if err := h264WeightPixels(want.Cr[cOff:], want.ChromaStride, 8, 1, -1, 3, 8); err != nil {
		t.Fatal(err)
	}

	requireH264BlockEqual(t, dst.Y, want.Y, dst.LumaStride, yOff, yOff, 16, 16)
	requireH264BlockEqual(t, dst.Cb, want.Cb, dst.ChromaStride, cOff, cOff, 8, 8)
	requireH264BlockEqual(t, dst.Cr, want.Cr, dst.ChromaStride, cOff, cOff, 8, 8)
}

func TestH264HLMotionFrameHigh9ExplicitWeightedChromaPList0(t *testing.T) {
	const bitDepth = 9
	weights := []struct {
		name  string
		table func(chromaFormatIDC int) PredWeightTable
	}{
		{name: "luma-only", table: func(chromaFormatIDC int) PredWeightTable {
			pwt := h264MotionCompTestPWT(chromaFormatIDC)
			pwt.UseWeight = 1
			pwt.LumaLog2WeightDenom = 2
			pwt.LumaWeight[0][0] = [2]int32{3, -2}
			return pwt
		}},
		{name: "luma-chroma", table: func(chromaFormatIDC int) PredWeightTable {
			pwt := h264MotionCompTestPWT(chromaFormatIDC)
			pwt.UseWeight = 1
			pwt.UseWeightChroma = 1
			pwt.LumaLog2WeightDenom = 2
			pwt.ChromaLog2WeightDenom = 1
			pwt.LumaWeight[0][0] = [2]int32{3, -2}
			pwt.ChromaWeight[0][0][0] = [2]int32{2, 1}
			pwt.ChromaWeight[0][0][1] = [2]int32{-1, 3}
			return pwt
		}},
		{name: "source-chroma-only", table: highSourceChromaOnlyWeightedPPredWeightTable},
	}
	for _, chromaFormatIDC := range []int{2, 3} {
		for _, weight := range weights {
			t.Run(fmt.Sprintf("%s/%s", chromaFormatName(chromaFormatIDC), weight.name), func(t *testing.T) {
				dst := makeH264MotionCompPictureHigh(chromaFormatIDC, bitDepth, 31)
				want := makeH264MotionCompPictureHigh(chromaFormatIDC, bitDepth, 31)
				ref0 := makeH264MotionCompPictureHigh(chromaFormatIDC, bitDepth, 103)
				refs := [2][]*h264PicturePlanesHigh{{ref0}}
				var cache macroblockMotionCache
				cache.Ref[0][h264Scan8[0]] = 0

				pwt := weight.table(chromaFormatIDC)
				const mbX = 1
				const mbY = 1
				mbType := MBType16x16 | MBTypeP0L0
				if err := h264HLMotionFrameWeightedHigh(dst, refs, &cache, mbType, [4]uint32{}, mbX, mbY, 1, &pwt, nil, bitDepth); err != nil {
					t.Fatal(err)
				}
				if err := h264HLMotionFrameHigh(want, refs, &cache, mbType, [4]uint32{}, mbX, mbY, 1, bitDepth); err != nil {
					t.Fatal(err)
				}
				yOff, cbOff, crOff, err := h264MBDestPartOffsetsHigh(want, mbX, mbY, 0, 0)
				if err != nil {
					t.Fatal(err)
				}
				if pwt.UseWeight != 0 {
					if err := h264WeightPixelsHigh(want.Y[yOff:], want.LumaStride, 16, int(pwt.LumaLog2WeightDenom), int(pwt.LumaWeight[0][0][0]), int(pwt.LumaWeight[0][0][1]), 16, bitDepth); err != nil {
						t.Fatal(err)
					}
				}
				chromaHeight, chromaWidth, err := h264ChromaWeightGeometry(want.ChromaFormatIDC, 16, 8, 16)
				if err != nil {
					t.Fatal(err)
				}
				if pwt.UseWeightChroma != 0 {
					if err := h264WeightPixelsHigh(want.Cb[cbOff:], want.ChromaStride, chromaHeight, int(pwt.ChromaLog2WeightDenom), int(pwt.ChromaWeight[0][0][0][0]), int(pwt.ChromaWeight[0][0][0][1]), chromaWidth, bitDepth); err != nil {
						t.Fatal(err)
					}
					if err := h264WeightPixelsHigh(want.Cr[crOff:], want.ChromaStride, chromaHeight, int(pwt.ChromaLog2WeightDenom), int(pwt.ChromaWeight[0][0][1][0]), int(pwt.ChromaWeight[0][0][1][1]), chromaWidth, bitDepth); err != nil {
						t.Fatal(err)
					}
				}

				requireH264BlockEqualHigh(t, dst.Y, want.Y, dst.LumaStride, yOff, yOff, 16, 16)
				requireH264BlockEqualHigh(t, dst.Cb, want.Cb, dst.ChromaStride, cbOff, cbOff, chromaWidth, chromaHeight)
				requireH264BlockEqualHigh(t, dst.Cr, want.Cr, dst.ChromaStride, crOff, crOff, chromaWidth, chromaHeight)
			})
		}
	}
}

func TestH264HLMotionFrameHigh10ExplicitWeightedField422List0(t *testing.T) {
	const bitDepth = 10
	dst := makeH264MotionCompPictureHigh(2, bitDepth, 25)
	want := makeH264MotionCompPictureHigh(2, bitDepth, 25)
	ref0 := makeH264MotionCompPictureHigh(2, bitDepth, 93)
	applySimpleFieldRefPlaneHigh(dst, PictureTopField)
	applySimpleFieldRefPlaneHigh(want, PictureTopField)
	applySimpleFieldRefPlaneHigh(ref0, PictureTopField)
	refs := [2][]*h264PicturePlanesHigh{{ref0}}
	var cache macroblockMotionCache
	cache.Ref[0][h264Scan8[0]] = 0

	pwt := h264MotionCompTestPWT(2)
	pwt.UseWeight = 1
	pwt.UseWeightChroma = 1
	pwt.LumaLog2WeightDenom = 2
	pwt.ChromaLog2WeightDenom = 1
	pwt.LumaWeight[0][0] = [2]int32{3, -2}
	pwt.ChromaWeight[0][0][0] = [2]int32{2, 1}
	pwt.ChromaWeight[0][0][1] = [2]int32{-1, 3}

	const mbX = 1
	const mbY = 1
	mbType := MBType16x16 | MBTypeP0L0 | MBTypeInterlaced
	if err := h264HLMotionFrameWeightedHigh(dst, refs, &cache, mbType, [4]uint32{}, mbX, mbY, 1, &pwt, nil, bitDepth); err != nil {
		t.Fatal(err)
	}
	if err := h264HLMotionFrameHigh(want, refs, &cache, mbType, [4]uint32{}, mbX, mbY, 1, bitDepth); err != nil {
		t.Fatal(err)
	}
	yOff := mbY*16*want.LumaStride + mbX*16
	if err := h264WeightPixelsHigh(want.Y[yOff:], want.LumaStride, 16, 2, 3, -2, 16, bitDepth); err != nil {
		t.Fatal(err)
	}
	_, cbOff, crOff, err := h264MBDestPartOffsetsHigh(want, mbX, mbY, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	chromaHeight, chromaWidth, err := h264ChromaWeightGeometry(want.ChromaFormatIDC, 16, 8, 16)
	if err != nil {
		t.Fatal(err)
	}
	if err := h264WeightPixelsHigh(want.Cb[cbOff:], want.ChromaStride, chromaHeight, 1, 2, 1, chromaWidth, bitDepth); err != nil {
		t.Fatal(err)
	}
	if err := h264WeightPixelsHigh(want.Cr[crOff:], want.ChromaStride, chromaHeight, 1, -1, 3, chromaWidth, bitDepth); err != nil {
		t.Fatal(err)
	}

	requireH264BlockEqualHigh(t, dst.Y, want.Y, dst.LumaStride, yOff, yOff, 16, 16)
	requireH264BlockEqualHigh(t, dst.Cb, want.Cb, dst.ChromaStride, cbOff, cbOff, chromaWidth, chromaHeight)
	requireH264BlockEqualHigh(t, dst.Cr, want.Cr, dst.ChromaStride, crOff, crOff, chromaWidth, chromaHeight)
}

func TestH264HLMotionFrameHigh10ExplicitWeightedField444List0(t *testing.T) {
	const bitDepth = 10
	dst := makeH264MotionCompPictureHigh(3, bitDepth, 29)
	want := makeH264MotionCompPictureHigh(3, bitDepth, 29)
	ref0 := makeH264MotionCompPictureHigh(3, bitDepth, 97)
	applySimpleFieldRefPlaneHigh(dst, PictureTopField)
	applySimpleFieldRefPlaneHigh(want, PictureTopField)
	applySimpleFieldRefPlaneHigh(ref0, PictureTopField)
	refs := [2][]*h264PicturePlanesHigh{{ref0}}
	var cache macroblockMotionCache
	cache.Ref[0][h264Scan8[0]] = 0

	pwt := h264MotionCompTestPWT(3)
	pwt.UseWeight = 1
	pwt.UseWeightChroma = 1
	pwt.LumaLog2WeightDenom = 2
	pwt.ChromaLog2WeightDenom = 1
	pwt.LumaWeight[0][0] = [2]int32{3, -2}
	pwt.ChromaWeight[0][0][0] = [2]int32{2, 1}
	pwt.ChromaWeight[0][0][1] = [2]int32{-1, 3}

	const mbX = 1
	const mbY = 1
	mbType := MBType16x16 | MBTypeP0L0 | MBTypeInterlaced
	if err := h264HLMotionFrameWeightedHigh(dst, refs, &cache, mbType, [4]uint32{}, mbX, mbY, 1, &pwt, nil, bitDepth); err != nil {
		t.Fatal(err)
	}
	if err := h264HLMotionFrameHigh(want, refs, &cache, mbType, [4]uint32{}, mbX, mbY, 1, bitDepth); err != nil {
		t.Fatal(err)
	}
	yOff := mbY*16*want.LumaStride + mbX*16
	if err := h264WeightPixelsHigh(want.Y[yOff:], want.LumaStride, 16, 2, 3, -2, 16, bitDepth); err != nil {
		t.Fatal(err)
	}
	_, cbOff, crOff, err := h264MBDestPartOffsetsHigh(want, mbX, mbY, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	chromaHeight, chromaWidth, err := h264ChromaWeightGeometry(want.ChromaFormatIDC, 16, 8, 16)
	if err != nil {
		t.Fatal(err)
	}
	if err := h264WeightPixelsHigh(want.Cb[cbOff:], want.ChromaStride, chromaHeight, 1, 2, 1, chromaWidth, bitDepth); err != nil {
		t.Fatal(err)
	}
	if err := h264WeightPixelsHigh(want.Cr[crOff:], want.ChromaStride, chromaHeight, 1, -1, 3, chromaWidth, bitDepth); err != nil {
		t.Fatal(err)
	}

	requireH264BlockEqualHigh(t, dst.Y, want.Y, dst.LumaStride, yOff, yOff, 16, 16)
	requireH264BlockEqualHigh(t, dst.Cb, want.Cb, dst.ChromaStride, cbOff, cbOff, chromaWidth, chromaHeight)
	requireH264BlockEqualHigh(t, dst.Cr, want.Cr, dst.ChromaStride, crOff, crOff, chromaWidth, chromaHeight)
}

func TestH264HLMotionFrameHigh10ExplicitLumaOnlyWeightedField444List0(t *testing.T) {
	const bitDepth = 10
	dst := makeH264MotionCompPictureHigh(3, bitDepth, 33)
	want := makeH264MotionCompPictureHigh(3, bitDepth, 33)
	ref0 := makeH264MotionCompPictureHigh(3, bitDepth, 107)
	applySimpleFieldRefPlaneHigh(dst, PictureBottomField)
	applySimpleFieldRefPlaneHigh(want, PictureBottomField)
	applySimpleFieldRefPlaneHigh(ref0, PictureBottomField)
	refs := [2][]*h264PicturePlanesHigh{{ref0}}
	var cache macroblockMotionCache
	cache.Ref[0][h264Scan8[0]] = 0

	pwt := h264MotionCompTestPWT(3)
	pwt.UseWeight = 1
	pwt.LumaLog2WeightDenom = 2
	pwt.LumaWeight[0][0] = [2]int32{3, -2}

	const mbX = 1
	const mbY = 1
	mbType := MBType16x16 | MBTypeP0L0 | MBTypeInterlaced
	if err := h264HLMotionFrameWeightedHigh(dst, refs, &cache, mbType, [4]uint32{}, mbX, mbY, 1, &pwt, nil, bitDepth); err != nil {
		t.Fatal(err)
	}
	if err := h264HLMotionFrameHigh(want, refs, &cache, mbType, [4]uint32{}, mbX, mbY, 1, bitDepth); err != nil {
		t.Fatal(err)
	}
	yOff := mbY*16*want.LumaStride + mbX*16
	if err := h264WeightPixelsHigh(want.Y[yOff:], want.LumaStride, 16, 2, 3, -2, 16, bitDepth); err != nil {
		t.Fatal(err)
	}
	_, cbOff, crOff, err := h264MBDestPartOffsetsHigh(want, mbX, mbY, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	chromaHeight, chromaWidth, err := h264ChromaWeightGeometry(want.ChromaFormatIDC, 16, 8, 16)
	if err != nil {
		t.Fatal(err)
	}

	requireH264BlockEqualHigh(t, dst.Y, want.Y, dst.LumaStride, yOff, yOff, 16, 16)
	requireH264BlockEqualHigh(t, dst.Cb, want.Cb, dst.ChromaStride, cbOff, cbOff, chromaWidth, chromaHeight)
	requireH264BlockEqualHigh(t, dst.Cr, want.Cr, dst.ChromaStride, crOff, crOff, chromaWidth, chromaHeight)
}

func TestH264HLMotionFrameHigh10ExplicitSourceChromaOnlyWeightedField444List0(t *testing.T) {
	const bitDepth = 10
	dst := makeH264MotionCompPictureHigh(3, bitDepth, 37)
	want := makeH264MotionCompPictureHigh(3, bitDepth, 37)
	ref0 := makeH264MotionCompPictureHigh(3, bitDepth, 113)
	applySimpleFieldRefPlaneHigh(dst, PictureTopField)
	applySimpleFieldRefPlaneHigh(want, PictureTopField)
	applySimpleFieldRefPlaneHigh(ref0, PictureTopField)
	refs := [2][]*h264PicturePlanesHigh{{ref0}}
	var cache macroblockMotionCache
	cache.Ref[0][h264Scan8[0]] = 0

	pwt := highSourceChromaOnlyWeightedPPredWeightTable(3)

	const mbX = 1
	const mbY = 1
	mbType := MBType16x16 | MBTypeP0L0 | MBTypeInterlaced
	if err := h264HLMotionFrameWeightedHigh(dst, refs, &cache, mbType, [4]uint32{}, mbX, mbY, 1, &pwt, nil, bitDepth); err != nil {
		t.Fatal(err)
	}
	if err := h264HLMotionFrameHigh(want, refs, &cache, mbType, [4]uint32{}, mbX, mbY, 1, bitDepth); err != nil {
		t.Fatal(err)
	}
	yOff, cbOff, crOff, err := h264MBDestPartOffsetsHigh(want, mbX, mbY, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	chromaHeight, chromaWidth, err := h264ChromaWeightGeometry(want.ChromaFormatIDC, 16, 8, 16)
	if err != nil {
		t.Fatal(err)
	}
	if err := h264WeightPixelsHigh(want.Cb[cbOff:], want.ChromaStride, chromaHeight, 1, 3, -1, chromaWidth, bitDepth); err != nil {
		t.Fatal(err)
	}
	if err := h264WeightPixelsHigh(want.Cr[crOff:], want.ChromaStride, chromaHeight, 1, 2, 1, chromaWidth, bitDepth); err != nil {
		t.Fatal(err)
	}

	requireH264BlockEqualHigh(t, dst.Y, want.Y, dst.LumaStride, yOff, yOff, 16, 16)
	requireH264BlockEqualHigh(t, dst.Cb, want.Cb, dst.ChromaStride, cbOff, cbOff, chromaWidth, chromaHeight)
	requireH264BlockEqualHigh(t, dst.Cr, want.Cr, dst.ChromaStride, crOff, crOff, chromaWidth, chromaHeight)
}

func TestH264HLMotionFrameHigh1214ExplicitWeightedFieldPList0(t *testing.T) {
	weights := []struct {
		name  string
		table func(chromaFormatIDC int) PredWeightTable
	}{
		{name: "luma-only", table: func(int) PredWeightTable {
			pwt := highWeightedPPredWeightTable()
			pwt.UseWeightChroma = 0
			return pwt
		}},
		{name: "luma-chroma", table: func(int) PredWeightTable {
			return highWeightedPPredWeightTable()
		}},
		{name: "source-chroma-only", table: highSourceChromaOnlyWeightedPPredWeightTable},
	}
	for _, bitDepth := range []int{12, 14} {
		for _, chromaFormatIDC := range []int{2, 3} {
			for _, picture := range []int32{PictureTopField, PictureBottomField} {
				for _, weight := range weights {
					t.Run(fmt.Sprintf("%s/%s/picture%d/%s", bitDepthName(int32(bitDepth)), chromaFormatName(chromaFormatIDC), picture, weight.name), func(t *testing.T) {
						dst := makeH264MotionCompPictureHigh(chromaFormatIDC, bitDepth, 43)
						want := makeH264MotionCompPictureHigh(chromaFormatIDC, bitDepth, 43)
						ref0 := makeH264MotionCompPictureHigh(chromaFormatIDC, bitDepth, 119)
						applySimpleFieldRefPlaneHigh(dst, picture)
						applySimpleFieldRefPlaneHigh(want, picture)
						applySimpleFieldRefPlaneHigh(ref0, picture)
						refs := [2][]*h264PicturePlanesHigh{{ref0}}
						var cache macroblockMotionCache
						cache.Ref[0][h264Scan8[0]] = 0

						pwt := weight.table(chromaFormatIDC)
						const mbX = 1
						const mbY = 1
						mbType := MBType16x16 | MBTypeP0L0 | MBTypeInterlaced
						if err := h264HLMotionFrameWeightedHigh(dst, refs, &cache, mbType, [4]uint32{}, mbX, mbY, 1, &pwt, nil, bitDepth); err != nil {
							t.Fatal(err)
						}
						if err := h264HLMotionFrameHigh(want, refs, &cache, mbType, [4]uint32{}, mbX, mbY, 1, bitDepth); err != nil {
							t.Fatal(err)
						}
						yOff, cbOff, crOff, err := h264MBDestPartOffsetsHigh(want, mbX, mbY, 0, 0)
						if err != nil {
							t.Fatal(err)
						}
						if pwt.UseWeight != 0 {
							if err := h264WeightPixelsHigh(want.Y[yOff:], want.LumaStride, 16, int(pwt.LumaLog2WeightDenom), int(pwt.LumaWeight[0][0][0]), int(pwt.LumaWeight[0][0][1]), 16, bitDepth); err != nil {
								t.Fatal(err)
							}
						}
						chromaHeight, chromaWidth, err := h264ChromaWeightGeometry(want.ChromaFormatIDC, 16, 8, 16)
						if err != nil {
							t.Fatal(err)
						}
						if pwt.UseWeightChroma != 0 {
							if err := h264WeightPixelsHigh(want.Cb[cbOff:], want.ChromaStride, chromaHeight, int(pwt.ChromaLog2WeightDenom), int(pwt.ChromaWeight[0][0][0][0]), int(pwt.ChromaWeight[0][0][0][1]), chromaWidth, bitDepth); err != nil {
								t.Fatal(err)
							}
							if err := h264WeightPixelsHigh(want.Cr[crOff:], want.ChromaStride, chromaHeight, int(pwt.ChromaLog2WeightDenom), int(pwt.ChromaWeight[0][0][1][0]), int(pwt.ChromaWeight[0][0][1][1]), chromaWidth, bitDepth); err != nil {
								t.Fatal(err)
							}
						}

						requireH264BlockEqualHigh(t, dst.Y, want.Y, dst.LumaStride, yOff, yOff, 16, 16)
						requireH264BlockEqualHigh(t, dst.Cb, want.Cb, dst.ChromaStride, cbOff, cbOff, chromaWidth, chromaHeight)
						requireH264BlockEqualHigh(t, dst.Cr, want.Cr, dst.ChromaStride, crOff, crOff, chromaWidth, chromaHeight)
					})
				}
			}
		}
	}
}

func TestH264HLMotionFrameHigh10ImplicitWeightedField444BipredUsesParityWeight(t *testing.T) {
	const bitDepth = 10
	dst := makeH264MotionCompPictureHigh(3, bitDepth, 31)
	want := makeH264MotionCompPictureHigh(3, bitDepth, 31)
	tmp := makeH264MotionCompPictureHigh(3, bitDepth, 31)
	ref0 := makeH264MotionCompPictureHigh(3, bitDepth, 101)
	ref1 := makeH264MotionCompPictureHigh(3, bitDepth, 149)
	applySimpleFieldRefPlaneHigh(dst, PictureTopField)
	applySimpleFieldRefPlaneHigh(want, PictureTopField)
	applySimpleFieldRefPlaneHigh(tmp, PictureTopField)
	applySimpleFieldRefPlaneHigh(ref0, PictureTopField)
	applySimpleFieldRefPlaneHigh(ref1, PictureTopField)
	refs := [2][]*h264PicturePlanesHigh{{ref0}, {ref1}}
	cache := makeH264MotionCompBipredCache(0, 0)

	pwt := h264MotionCompTestPWT(3)
	pwt.UseWeight = 2
	pwt.UseWeightChroma = 2

	const mbX = 1
	const mbY = 1
	const weight0 = 21
	mbType := MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeInterlaced
	weightRef0, weightRef1 := h264ImplicitWeightIndexes(mbType, 0, 0, mbY)
	pwt.ImplicitWeight[weightRef0][weightRef1][mbY&1] = weight0

	if err := h264HLMotionFrameWeightedHigh(dst, refs, &cache, mbType, [4]uint32{}, mbX, mbY, 2, &pwt, makeH264MotionCompScratchHigh(dst), bitDepth); err != nil {
		t.Fatal(err)
	}
	if err := h264HLMotionFrameHigh(want, refs, &cache, MBType16x16|MBTypeP0L0|MBTypeInterlaced, [4]uint32{}, mbX, mbY, 2, bitDepth); err != nil {
		t.Fatal(err)
	}
	if err := h264HLMotionFrameHigh(tmp, refs, &cache, MBType16x16|MBTypeP0L1|MBTypeInterlaced, [4]uint32{}, mbX, mbY, 2, bitDepth); err != nil {
		t.Fatal(err)
	}
	yOff, cbOff, crOff, err := h264MBDestPartOffsetsHigh(want, mbX, mbY, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	weight1 := 64 - weight0
	if err := h264BiweightPixelsHigh(want.Y[yOff:], tmp.Y[yOff:], want.LumaStride, 16, 5, weight0, weight1, 0, 16, bitDepth); err != nil {
		t.Fatal(err)
	}
	chromaHeight, chromaWidth, err := h264ChromaWeightGeometry(want.ChromaFormatIDC, 16, 8, 16)
	if err != nil {
		t.Fatal(err)
	}
	if err := h264BiweightPixelsHigh(want.Cb[cbOff:], tmp.Cb[cbOff:], want.ChromaStride, chromaHeight, 5, weight0, weight1, 0, chromaWidth, bitDepth); err != nil {
		t.Fatal(err)
	}
	if err := h264BiweightPixelsHigh(want.Cr[crOff:], tmp.Cr[crOff:], want.ChromaStride, chromaHeight, 5, weight0, weight1, 0, chromaWidth, bitDepth); err != nil {
		t.Fatal(err)
	}

	requireH264BlockEqualHigh(t, dst.Y, want.Y, dst.LumaStride, yOff, yOff, 16, 16)
	requireH264BlockEqualHigh(t, dst.Cb, want.Cb, dst.ChromaStride, cbOff, cbOff, chromaWidth, chromaHeight)
	requireH264BlockEqualHigh(t, dst.Cr, want.Cr, dst.ChromaStride, crOff, crOff, chromaWidth, chromaHeight)
}

func TestH264HLMotionFrameHigh1214WeightedFieldBipredUsesParityWeight(t *testing.T) {
	for _, bitDepth := range []int{12, 14} {
		for _, chromaFormatIDC := range []int{2, 3} {
			for _, picture := range []int32{PictureTopField, PictureBottomField} {
				for _, tt := range []struct {
					name            string
					useWeight       int32
					useWeightChroma int32
				}{
					{name: "explicit", useWeight: 1, useWeightChroma: 1},
					{name: "implicit", useWeight: 2, useWeightChroma: 2},
				} {
					t.Run(fmt.Sprintf("%s/%s/picture%d/%s", bitDepthName(int32(bitDepth)), chromaFormatName(chromaFormatIDC), picture, tt.name), func(t *testing.T) {
						dst := makeH264MotionCompPictureHigh(chromaFormatIDC, bitDepth, 41)
						want := makeH264MotionCompPictureHigh(chromaFormatIDC, bitDepth, 41)
						tmp := makeH264MotionCompPictureHigh(chromaFormatIDC, bitDepth, 41)
						ref0 := makeH264MotionCompPictureHigh(chromaFormatIDC, bitDepth, 137)
						ref1 := makeH264MotionCompPictureHigh(chromaFormatIDC, bitDepth, 181)
						applySimpleFieldRefPlaneHigh(dst, picture)
						applySimpleFieldRefPlaneHigh(want, picture)
						applySimpleFieldRefPlaneHigh(tmp, picture)
						applySimpleFieldRefPlaneHigh(ref0, picture)
						applySimpleFieldRefPlaneHigh(ref1, picture)
						refs := [2][]*h264PicturePlanesHigh{{ref0}, {ref1}}
						cache := makeH264MotionCompBipredCache(0, 0)

						pwt := h264MotionCompTestPWT(chromaFormatIDC)
						pwt.UseWeight = tt.useWeight
						pwt.UseWeightChroma = tt.useWeightChroma
						if tt.useWeight == 1 {
							pwt.LumaLog2WeightDenom = 1
							pwt.ChromaLog2WeightDenom = 1
							pwt.LumaWeight[0][0] = [2]int32{3, -1}
							pwt.LumaWeight[0][1] = [2]int32{1, 2}
							pwt.ChromaWeight[0][0][0] = [2]int32{3, 1}
							pwt.ChromaWeight[0][1][0] = [2]int32{1, -1}
							pwt.ChromaWeight[0][0][1] = [2]int32{1, 2}
							pwt.ChromaWeight[0][1][1] = [2]int32{3, 0}
						}

						const mbX = 1
						const mbY = 1
						const implicitWeight0 = 21
						mbType := MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeInterlaced
						if tt.useWeight == 2 {
							weightRef0, weightRef1 := h264ImplicitWeightIndexes(mbType, 0, 0, mbY)
							pwt.ImplicitWeight[weightRef0][weightRef1][mbY&1] = implicitWeight0
						}

						if err := h264HLMotionFrameWeightedHigh(dst, refs, &cache, mbType, [4]uint32{}, mbX, mbY, 2, &pwt, makeH264MotionCompScratchHigh(dst), bitDepth); err != nil {
							t.Fatal(err)
						}
						if err := h264HLMotionFrameHigh(want, refs, &cache, MBType16x16|MBTypeP0L0|MBTypeInterlaced, [4]uint32{}, mbX, mbY, 2, bitDepth); err != nil {
							t.Fatal(err)
						}
						if err := h264HLMotionFrameHigh(tmp, refs, &cache, MBType16x16|MBTypeP0L1|MBTypeInterlaced, [4]uint32{}, mbX, mbY, 2, bitDepth); err != nil {
							t.Fatal(err)
						}

						yOff, cbOff, crOff, err := h264MBDestPartOffsetsHigh(want, mbX, mbY, 0, 0)
						if err != nil {
							t.Fatal(err)
						}
						chromaHeight, chromaWidth, err := h264ChromaWeightGeometry(want.ChromaFormatIDC, 16, 8, 16)
						if err != nil {
							t.Fatal(err)
						}

						if tt.useWeight == 2 {
							weight1 := 64 - implicitWeight0
							if err := h264BiweightPixelsHigh(want.Y[yOff:], tmp.Y[yOff:], want.LumaStride, 16, 5, implicitWeight0, weight1, 0, 16, bitDepth); err != nil {
								t.Fatal(err)
							}
							if err := h264BiweightPixelsHigh(want.Cb[cbOff:], tmp.Cb[cbOff:], want.ChromaStride, chromaHeight, 5, implicitWeight0, weight1, 0, chromaWidth, bitDepth); err != nil {
								t.Fatal(err)
							}
							if err := h264BiweightPixelsHigh(want.Cr[crOff:], tmp.Cr[crOff:], want.ChromaStride, chromaHeight, 5, implicitWeight0, weight1, 0, chromaWidth, bitDepth); err != nil {
								t.Fatal(err)
							}
						} else {
							if err := h264BiweightPixelsHigh(want.Y[yOff:], tmp.Y[yOff:], want.LumaStride, 16, int(pwt.LumaLog2WeightDenom), int(pwt.LumaWeight[0][0][0]), int(pwt.LumaWeight[0][1][0]), int(pwt.LumaWeight[0][0][1]+pwt.LumaWeight[0][1][1]), 16, bitDepth); err != nil {
								t.Fatal(err)
							}
							if err := h264BiweightPixelsHigh(want.Cb[cbOff:], tmp.Cb[cbOff:], want.ChromaStride, chromaHeight, int(pwt.ChromaLog2WeightDenom), int(pwt.ChromaWeight[0][0][0][0]), int(pwt.ChromaWeight[0][1][0][0]), int(pwt.ChromaWeight[0][0][0][1]+pwt.ChromaWeight[0][1][0][1]), chromaWidth, bitDepth); err != nil {
								t.Fatal(err)
							}
							if err := h264BiweightPixelsHigh(want.Cr[crOff:], tmp.Cr[crOff:], want.ChromaStride, chromaHeight, int(pwt.ChromaLog2WeightDenom), int(pwt.ChromaWeight[0][0][1][0]), int(pwt.ChromaWeight[0][1][1][0]), int(pwt.ChromaWeight[0][0][1][1]+pwt.ChromaWeight[0][1][1][1]), chromaWidth, bitDepth); err != nil {
								t.Fatal(err)
							}
						}

						requireH264BlockEqualHigh(t, dst.Y, want.Y, dst.LumaStride, yOff, yOff, 16, 16)
						requireH264BlockEqualHigh(t, dst.Cb, want.Cb, dst.ChromaStride, cbOff, cbOff, chromaWidth, chromaHeight)
						requireH264BlockEqualHigh(t, dst.Cr, want.Cr, dst.ChromaStride, crOff, crOff, chromaWidth, chromaHeight)
					})
				}
			}
		}
	}
}

func TestH264HLMotionFrameHigh9ImplicitWeightedChromaBipredUsesParityWeight(t *testing.T) {
	const bitDepth = 9
	for _, chromaFormatIDC := range []int{2, 3} {
		t.Run(chromaFormatName(chromaFormatIDC), func(t *testing.T) {
			dst := makeH264MotionCompPictureHigh(chromaFormatIDC, bitDepth, 53)
			want := makeH264MotionCompPictureHigh(chromaFormatIDC, bitDepth, 53)
			tmp := makeH264MotionCompPictureHigh(chromaFormatIDC, bitDepth, 53)
			ref0 := makeH264MotionCompPictureHigh(chromaFormatIDC, bitDepth, 163)
			ref1 := makeH264MotionCompPictureHigh(chromaFormatIDC, bitDepth, 197)
			refs := [2][]*h264PicturePlanesHigh{{ref0}, {ref1}}
			cache := makeH264MotionCompBipredCache(0, 0)

			pwt := h264MotionCompTestPWT(chromaFormatIDC)
			pwt.UseWeight = 2
			pwt.UseWeightChroma = 2

			const mbX = 1
			const mbY = 1
			const weight0 = 21
			mbType := MBType16x16 | MBTypeP0L0 | MBTypeP0L1
			weightRef0, weightRef1 := h264ImplicitWeightIndexes(mbType, 0, 0, mbY)
			pwt.ImplicitWeight[weightRef0][weightRef1][mbY&1] = weight0

			if err := h264HLMotionFrameWeightedHigh(dst, refs, &cache, mbType, [4]uint32{}, mbX, mbY, 2, &pwt, makeH264MotionCompScratchHigh(dst), bitDepth); err != nil {
				t.Fatal(err)
			}
			if err := h264HLMotionFrameHigh(want, refs, &cache, MBType16x16|MBTypeP0L0, [4]uint32{}, mbX, mbY, 2, bitDepth); err != nil {
				t.Fatal(err)
			}
			if err := h264HLMotionFrameHigh(tmp, refs, &cache, MBType16x16|MBTypeP0L1, [4]uint32{}, mbX, mbY, 2, bitDepth); err != nil {
				t.Fatal(err)
			}

			yOff, cbOff, crOff, err := h264MBDestPartOffsetsHigh(want, mbX, mbY, 0, 0)
			if err != nil {
				t.Fatal(err)
			}
			chromaHeight, chromaWidth, err := h264ChromaWeightGeometry(want.ChromaFormatIDC, 16, 8, 16)
			if err != nil {
				t.Fatal(err)
			}
			weight1 := 64 - weight0
			if err := h264BiweightPixelsHigh(want.Y[yOff:], tmp.Y[yOff:], want.LumaStride, 16, 5, weight0, weight1, 0, 16, bitDepth); err != nil {
				t.Fatal(err)
			}
			if err := h264BiweightPixelsHigh(want.Cb[cbOff:], tmp.Cb[cbOff:], want.ChromaStride, chromaHeight, 5, weight0, weight1, 0, chromaWidth, bitDepth); err != nil {
				t.Fatal(err)
			}
			if err := h264BiweightPixelsHigh(want.Cr[crOff:], tmp.Cr[crOff:], want.ChromaStride, chromaHeight, 5, weight0, weight1, 0, chromaWidth, bitDepth); err != nil {
				t.Fatal(err)
			}

			requireH264BlockEqualHigh(t, dst.Y, want.Y, dst.LumaStride, yOff, yOff, 16, 16)
			requireH264BlockEqualHigh(t, dst.Cb, want.Cb, dst.ChromaStride, cbOff, cbOff, chromaWidth, chromaHeight)
			requireH264BlockEqualHigh(t, dst.Cr, want.Cr, dst.ChromaStride, crOff, crOff, chromaWidth, chromaHeight)
		})
	}
}

func TestH264HLMotionFrameHigh9ExplicitWeightedChromaBipredUsesParityWeight(t *testing.T) {
	const bitDepth = 9
	for _, chromaFormatIDC := range []int{2, 3} {
		t.Run(chromaFormatName(chromaFormatIDC), func(t *testing.T) {
			dst := makeH264MotionCompPictureHigh(chromaFormatIDC, bitDepth, 53)
			want := makeH264MotionCompPictureHigh(chromaFormatIDC, bitDepth, 53)
			tmp := makeH264MotionCompPictureHigh(chromaFormatIDC, bitDepth, 53)
			ref0 := makeH264MotionCompPictureHigh(chromaFormatIDC, bitDepth, 163)
			ref1 := makeH264MotionCompPictureHigh(chromaFormatIDC, bitDepth, 197)
			refs := [2][]*h264PicturePlanesHigh{{ref0}, {ref1}}
			cache := makeH264MotionCompBipredCache(0, 0)

			pwt := h264MotionCompTestPWT(chromaFormatIDC)
			pwt.UseWeight = 1
			pwt.UseWeightChroma = 1
			pwt.LumaLog2WeightDenom = 1
			pwt.ChromaLog2WeightDenom = 1
			pwt.LumaWeight[0][0] = [2]int32{3, -1}
			pwt.LumaWeight[0][1] = [2]int32{1, 2}
			pwt.ChromaWeight[0][0][0] = [2]int32{3, 1}
			pwt.ChromaWeight[0][1][0] = [2]int32{1, -1}
			pwt.ChromaWeight[0][0][1] = [2]int32{1, 2}
			pwt.ChromaWeight[0][1][1] = [2]int32{3, 0}

			const mbX = 1
			const mbY = 1
			mbType := MBType16x16 | MBTypeP0L0 | MBTypeP0L1
			if err := h264HLMotionFrameWeightedHigh(dst, refs, &cache, mbType, [4]uint32{}, mbX, mbY, 2, &pwt, makeH264MotionCompScratchHigh(dst), bitDepth); err != nil {
				t.Fatal(err)
			}
			if err := h264HLMotionFrameHigh(want, refs, &cache, MBType16x16|MBTypeP0L0, [4]uint32{}, mbX, mbY, 2, bitDepth); err != nil {
				t.Fatal(err)
			}
			if err := h264HLMotionFrameHigh(tmp, refs, &cache, MBType16x16|MBTypeP0L1, [4]uint32{}, mbX, mbY, 2, bitDepth); err != nil {
				t.Fatal(err)
			}

			yOff, cbOff, crOff, err := h264MBDestPartOffsetsHigh(want, mbX, mbY, 0, 0)
			if err != nil {
				t.Fatal(err)
			}
			chromaHeight, chromaWidth, err := h264ChromaWeightGeometry(want.ChromaFormatIDC, 16, 8, 16)
			if err != nil {
				t.Fatal(err)
			}
			if err := h264BiweightPixelsHigh(want.Y[yOff:], tmp.Y[yOff:], want.LumaStride, 16, int(pwt.LumaLog2WeightDenom), int(pwt.LumaWeight[0][0][0]), int(pwt.LumaWeight[0][1][0]), int(pwt.LumaWeight[0][0][1]+pwt.LumaWeight[0][1][1]), 16, bitDepth); err != nil {
				t.Fatal(err)
			}
			if err := h264BiweightPixelsHigh(want.Cb[cbOff:], tmp.Cb[cbOff:], want.ChromaStride, chromaHeight, int(pwt.ChromaLog2WeightDenom), int(pwt.ChromaWeight[0][0][0][0]), int(pwt.ChromaWeight[0][1][0][0]), int(pwt.ChromaWeight[0][0][0][1]+pwt.ChromaWeight[0][1][0][1]), chromaWidth, bitDepth); err != nil {
				t.Fatal(err)
			}
			if err := h264BiweightPixelsHigh(want.Cr[crOff:], tmp.Cr[crOff:], want.ChromaStride, chromaHeight, int(pwt.ChromaLog2WeightDenom), int(pwt.ChromaWeight[0][0][1][0]), int(pwt.ChromaWeight[0][1][1][0]), int(pwt.ChromaWeight[0][0][1][1]+pwt.ChromaWeight[0][1][1][1]), chromaWidth, bitDepth); err != nil {
				t.Fatal(err)
			}

			requireH264BlockEqualHigh(t, dst.Y, want.Y, dst.LumaStride, yOff, yOff, 16, 16)
			requireH264BlockEqualHigh(t, dst.Cb, want.Cb, dst.ChromaStride, cbOff, cbOff, chromaWidth, chromaHeight)
			requireH264BlockEqualHigh(t, dst.Cr, want.Cr, dst.ChromaStride, crOff, crOff, chromaWidth, chromaHeight)
		})
	}
}

func TestH264HLMotionFrameWeightedImplicitRequiresScratch(t *testing.T) {
	dst := makeH264MotionCompPicture(1, 31)
	ref0 := makeH264MotionCompPicture(1, 61)
	ref1 := makeH264MotionCompPicture(1, 97)
	refs := [2][]*h264PicturePlanes{{ref0}, {ref1}}
	var cache macroblockMotionCache
	cache.Ref[0][h264Scan8[0]] = 0
	cache.Ref[1][h264Scan8[0]] = 0

	pwt := h264MotionCompTestPWT(1)
	pwt.UseWeight = 2
	pwt.ImplicitWeight[0][0][1] = 21

	err := h264HLMotionFrameWeighted(dst, refs, &cache, MBType16x16|MBTypeP0L0|MBTypeP0L1, [4]uint32{}, 1, 1, 2, &pwt, nil)
	if !errors.Is(err, ErrInvalidData) {
		t.Fatalf("missing scratch error = %v, want ErrInvalidData", err)
	}
}

func TestH264HLMotionFrameImplicitWeightedPartitionedBWeight32FallsBackToStd(t *testing.T) {
	dst := makeH264MotionCompPicture(1, 29)
	want := makeH264MotionCompPicture(1, 29)
	ref0 := makeH264MotionCompPicture(1, 67)
	ref1 := makeH264MotionCompPicture(1, 113)
	refs := [2][]*h264PicturePlanes{{ref0}, {ref1}}
	cache := makeH264MotionCompBipredCache(0, 0)

	pwt := h264MotionCompTestPWT(1)
	pwt.UseWeight = 2
	pwt.UseWeightChroma = 2

	const mbX = 1
	const mbY = 1
	mbType := MBType16x8 | MBTypeP0L0 | MBTypeP0L1 | MBTypeP1L0 | MBTypeP1L1
	if err := h264HLMotionFrameWeighted(dst, refs, &cache, mbType, [4]uint32{}, mbX, mbY, 2, &pwt, nil); err != nil {
		t.Fatal(err)
	}
	if err := h264HLMotionFrame(want, refs, &cache, mbType, [4]uint32{}, mbX, mbY, 2); err != nil {
		t.Fatal(err)
	}

	requireH264MotionCompMBEqual(t, dst, want, mbX, mbY)
}

func TestH264MCPartUsesWeightedKeepsMBAFFFieldWeightsOffset(t *testing.T) {
	cache := makeH264MotionCompBipredCache(0, 0)
	pwt := h264MotionCompTestPWT(1)
	pwt.UseWeight = 2
	pwt.UseWeightChroma = 2
	pwt.ImplicitWeight[0][0] = [2]int32{32, 32}
	pwt.ImplicitWeight[16][16] = [2]int32{21, 32}
	pwt.ImplicitWeight[17][17] = [2]int32{32, 21}

	frameMBType := MBType16x16 | MBTypeP0L0 | MBTypeP0L1
	if h264MCPartUsesWeighted(&pwt, &cache, frameMBType, 0, true, true, 0) {
		t.Fatalf("frame-coded implicit weight used MBAFF field offset")
	}

	fieldMBType := frameMBType | MBTypeInterlaced
	if !h264MCPartUsesWeighted(&pwt, &cache, fieldMBType, 0, true, true, 0) {
		t.Fatalf("top field-coded MBAFF implicit weight did not use offset")
	}
	if !h264MCPartUsesWeighted(&pwt, &cache, fieldMBType, 0, true, true, 1) {
		t.Fatalf("bottom field-coded MBAFF implicit weight did not xor original macroblock parity")
	}
}

func TestH264HLMotionFrameImplicitWeightedUsesOriginalMBAFFParity(t *testing.T) {
	dst := makeH264MotionCompPicture(1, 33)
	want := makeH264MotionCompPicture(1, 33)
	tmp := makeH264MotionCompPicture(1, 33)
	ref0 := makeH264MotionCompPicture(1, 71)
	ref1 := makeH264MotionCompPicture(1, 109)
	refs := [2][]*h264PicturePlanes{{ref0}, {ref1}}
	cache := makeH264MotionCompBipredCache(0, 0)

	pwt := h264MotionCompTestPWT(1)
	pwt.UseWeight = 2
	pwt.UseWeightChroma = 2
	pwt.ImplicitWeight[0][0] = [2]int32{32, 21}

	const mbX = 1
	const viewMBY = 0
	const originalMBY = 1
	const weight0 = 21
	mbType := MBType16x8 | MBTypeP0L0 | MBTypeP0L1 | MBTypeP1L0 | MBTypeP1L1
	if err := h264HLMotionFrameWeightedWithWeightY(dst, refs, &cache, mbType, [4]uint32{}, mbX, viewMBY, 2, originalMBY, &pwt, makeH264MotionCompScratch(dst)); err != nil {
		t.Fatal(err)
	}
	if err := h264HLMotionFrame(want, refs, &cache, MBType16x8|MBTypeP0L0|MBTypeP1L0, [4]uint32{}, mbX, viewMBY, 2); err != nil {
		t.Fatal(err)
	}
	if err := h264HLMotionFrame(tmp, refs, &cache, MBType16x8|MBTypeP0L1|MBTypeP1L1, [4]uint32{}, mbX, viewMBY, 2); err != nil {
		t.Fatal(err)
	}
	applyH264ImplicitBiweight16x8(t, want, tmp, mbX, viewMBY, weight0)

	requireH264MotionCompMBEqual(t, dst, want, mbX, viewMBY)
}

func TestH264HLMotionFrameImplicitWeightedMBAFF8x8SubpartsUseParentFieldFlag(t *testing.T) {
	dst := makeH264MotionCompPicture(1, 35)
	want := makeH264MotionCompPicture(1, 35)
	tmp := makeH264MotionCompPicture(1, 35)
	unused := makeH264MotionCompPicture(1, 43)
	ref0 := makeH264MotionCompPicture(1, 79)
	ref1 := makeH264MotionCompPicture(1, 117)
	refs := [2][]*h264PicturePlanes{{unused, ref0}, {ref1}}
	cache := makeH264MotionCompBipredCache(1, 0)

	pwt := h264MotionCompTestPWT(1)
	pwt.UseWeight = 2
	pwt.UseWeightChroma = 2
	pwt.ImplicitWeight[1][0] = [2]int32{32, 32}
	pwt.ImplicitWeight[17][16] = [2]int32{21, 32}

	const mbX = 1
	const viewMBY = 0
	const originalMBY = 0
	const weight0 = 21
	mbType := MBType8x8 | MBTypeL0L1 | MBTypeInterlaced
	sub := [4]uint32{
		MBType16x16 | MBTypeP0L0 | MBTypeP0L1,
		MBType16x16 | MBTypeP0L0 | MBTypeP0L1,
		MBType16x16 | MBTypeP0L0 | MBTypeP0L1,
		MBType16x16 | MBTypeP0L0 | MBTypeP0L1,
	}
	if err := h264HLMotionFrameWeightedWithWeightY(dst, refs, &cache, mbType, sub, mbX, viewMBY, 2, originalMBY, &pwt, makeH264MotionCompScratch(dst)); err != nil {
		t.Fatal(err)
	}

	subL0 := [4]uint32{
		MBType16x16 | MBTypeP0L0,
		MBType16x16 | MBTypeP0L0,
		MBType16x16 | MBTypeP0L0,
		MBType16x16 | MBTypeP0L0,
	}
	subL1 := [4]uint32{
		MBType16x16 | MBTypeP0L1,
		MBType16x16 | MBTypeP0L1,
		MBType16x16 | MBTypeP0L1,
		MBType16x16 | MBTypeP0L1,
	}
	if err := h264HLMotionFrame(want, refs, &cache, MBType8x8|MBTypeP0L0, subL0, mbX, viewMBY, 2); err != nil {
		t.Fatal(err)
	}
	if err := h264HLMotionFrame(tmp, refs, &cache, MBType8x8|MBTypeP0L1, subL1, mbX, viewMBY, 2); err != nil {
		t.Fatal(err)
	}
	applyH264ImplicitBiweight8x8Subparts(t, want, tmp, mbX, viewMBY, weight0)

	requireH264MotionCompMBEqual(t, dst, want, mbX, viewMBY)
}

func TestH264HLMotionFrameHigh10ImplicitWeightedPartitionedB16x8UsesParityWeight(t *testing.T) {
	const bitDepth = 10
	dst := makeH264MotionCompPictureHigh(1, bitDepth, 37)
	want := makeH264MotionCompPictureHigh(1, bitDepth, 37)
	tmp := makeH264MotionCompPictureHigh(1, bitDepth, 37)
	ref0 := makeH264MotionCompPictureHigh(1, bitDepth, 73)
	ref1 := makeH264MotionCompPictureHigh(1, bitDepth, 131)
	refs := [2][]*h264PicturePlanesHigh{{ref0}, {ref1}}
	cache := makeH264MotionCompBipredCache(0, 0)

	pwt := h264MotionCompTestPWT(1)
	pwt.UseWeight = 2
	pwt.UseWeightChroma = 2
	pwt.ImplicitWeight[0][0] = [2]int32{32, 21}

	const mbX = 1
	const mbY = 1
	const weight0 = 21
	mbType := MBType16x8 | MBTypeP0L0 | MBTypeP0L1 | MBTypeP1L0 | MBTypeP1L1
	if err := h264HLMotionFrameWeightedHigh(dst, refs, &cache, mbType, [4]uint32{}, mbX, mbY, 2, &pwt, makeH264MotionCompScratchHigh(dst), bitDepth); err != nil {
		t.Fatal(err)
	}
	if err := h264HLMotionFrameHigh(want, refs, &cache, MBType16x8|MBTypeP0L0|MBTypeP1L0, [4]uint32{}, mbX, mbY, 2, bitDepth); err != nil {
		t.Fatal(err)
	}
	if err := h264HLMotionFrameHigh(tmp, refs, &cache, MBType16x8|MBTypeP0L1|MBTypeP1L1, [4]uint32{}, mbX, mbY, 2, bitDepth); err != nil {
		t.Fatal(err)
	}
	applyH264ImplicitBiweight16x8High(t, want, tmp, mbX, mbY, weight0, bitDepth)

	requireH264MotionCompMBEqualHigh(t, dst, want, mbX, mbY)
}

func TestH264HLMotionFrameHigh10ImplicitWeightedMBAFF8x8SubpartsUseParentFieldFlag(t *testing.T) {
	const bitDepth = 10
	dst := makeH264MotionCompPictureHigh(1, bitDepth, 39)
	want := makeH264MotionCompPictureHigh(1, bitDepth, 39)
	tmp := makeH264MotionCompPictureHigh(1, bitDepth, 39)
	unused := makeH264MotionCompPictureHigh(1, bitDepth, 47)
	ref0 := makeH264MotionCompPictureHigh(1, bitDepth, 83)
	ref1 := makeH264MotionCompPictureHigh(1, bitDepth, 127)
	refs := [2][]*h264PicturePlanesHigh{{unused, ref0}, {ref1}}
	cache := makeH264MotionCompBipredCache(1, 0)

	pwt := h264MotionCompTestPWT(1)
	pwt.UseWeight = 2
	pwt.UseWeightChroma = 2
	pwt.ImplicitWeight[1][0] = [2]int32{32, 32}
	pwt.ImplicitWeight[17][16] = [2]int32{21, 32}

	const mbX = 1
	const viewMBY = 0
	const originalMBY = 0
	const weight0 = 21
	mbType := MBType8x8 | MBTypeL0L1 | MBTypeInterlaced
	sub := [4]uint32{
		MBType16x16 | MBTypeP0L0 | MBTypeP0L1,
		MBType16x16 | MBTypeP0L0 | MBTypeP0L1,
		MBType16x16 | MBTypeP0L0 | MBTypeP0L1,
		MBType16x16 | MBTypeP0L0 | MBTypeP0L1,
	}
	if err := h264HLMotionFrameWeightedHighWithWeightY(dst, refs, &cache, mbType, sub, mbX, viewMBY, 2, originalMBY, &pwt, makeH264MotionCompScratchHigh(dst), bitDepth); err != nil {
		t.Fatal(err)
	}

	subL0 := [4]uint32{
		MBType16x16 | MBTypeP0L0,
		MBType16x16 | MBTypeP0L0,
		MBType16x16 | MBTypeP0L0,
		MBType16x16 | MBTypeP0L0,
	}
	subL1 := [4]uint32{
		MBType16x16 | MBTypeP0L1,
		MBType16x16 | MBTypeP0L1,
		MBType16x16 | MBTypeP0L1,
		MBType16x16 | MBTypeP0L1,
	}
	if err := h264HLMotionFrameHigh(want, refs, &cache, MBType8x8|MBTypeP0L0, subL0, mbX, viewMBY, 2, bitDepth); err != nil {
		t.Fatal(err)
	}
	if err := h264HLMotionFrameHigh(tmp, refs, &cache, MBType8x8|MBTypeP0L1, subL1, mbX, viewMBY, 2, bitDepth); err != nil {
		t.Fatal(err)
	}
	applyH264ImplicitBiweight8x8SubpartsHigh(t, want, tmp, mbX, viewMBY, weight0, bitDepth)

	requireH264MotionCompMBEqualHigh(t, dst, want, mbX, viewMBY)
}

func TestH264HLMotionFrameSubPartitionsCopyFullMB(t *testing.T) {
	dst := makeH264MotionCompPicture(2, 7)
	ref := makeH264MotionCompPicture(2, 123)
	refs := [2][]*h264PicturePlanes{{ref}}
	var cache macroblockMotionCache
	for n := 0; n < 16; n++ {
		cache.Ref[0][h264Scan8[n]] = 0
	}

	const mbX = 1
	const mbY = 1
	mbType := MBType8x8 | MBTypeP0L0 | MBTypeP1L0
	subMBType := [4]uint32{
		MBType16x16 | MBTypeP0L0,
		MBType16x8 | MBTypeP0L0,
		MBType8x16 | MBTypeP0L0,
		MBType8x8 | MBTypeP0L0,
	}
	if err := h264HLMotionFrame(dst, refs, &cache, mbType, subMBType, mbX, mbY, 1); err != nil {
		t.Fatal(err)
	}

	yOff := mbY*16*dst.LumaStride + mbX*16
	requireH264BlockEqual(t, dst.Y, ref.Y, dst.LumaStride, yOff, yOff, 16, 16)
	cOff := mbY*16*dst.ChromaStride + mbX*8
	requireH264BlockEqual(t, dst.Cb, ref.Cb, dst.ChromaStride, cOff, cOff, 8, 16)
	requireH264BlockEqual(t, dst.Cr, ref.Cr, dst.ChromaStride, cOff, cOff, 8, 16)
}

func TestH264HLMotionFrameEdgeEmulationClipsTopLeft(t *testing.T) {
	dst := makeH264MotionCompPicture(1, 5)
	ref := makeH264MotionCompPicture(1, 15)
	refs := [2][]*h264PicturePlanes{{ref}}
	var cache macroblockMotionCache
	cache.Ref[0][h264Scan8[0]] = 0
	cache.MV[0][h264Scan8[0]] = [2]int16{1, 0}

	if err := h264HLMotionFrameWithScratch(dst, refs, &cache, MBType16x16|MBTypeP0L0, [4]uint32{}, 0, 1, 1, makeH264MotionCompScratch(dst)); err != nil {
		t.Fatal(err)
	}
	yOff := 16 * dst.LumaStride
	if dst.Y[yOff] != 66 {
		t.Fatalf("edge-emulated luma sample = %d, want 66", dst.Y[yOff])
	}
	cOff := 8 * dst.ChromaStride
	if dst.Cb[cOff] != 86 || dst.Cr[cOff] != 128 {
		t.Fatalf("edge-emulated chroma samples = %d/%d, want 86/128", dst.Cb[cOff], dst.Cr[cOff])
	}
}

func TestH264HLMotionFrameEdgeEmulationRequiresScratch(t *testing.T) {
	dst := makeH264MotionCompPicture(1, 5)
	ref := makeH264MotionCompPicture(1, 15)
	refs := [2][]*h264PicturePlanes{{ref}}
	var cache macroblockMotionCache
	cache.Ref[0][h264Scan8[0]] = 0
	cache.MV[0][h264Scan8[0]] = [2]int16{1, 0}

	err := h264HLMotionFrame(dst, refs, &cache, MBType16x16|MBTypeP0L0, [4]uint32{}, 0, 1, 1)
	if !errors.Is(err, ErrInvalidData) {
		t.Fatalf("missing edge scratch error = %v, want ErrInvalidData", err)
	}
}

func TestH264HLMotionFrameEdgeEmulationAllowsWideBlockOnTightStride(t *testing.T) {
	dst := makeH264MotionCompTightPicture(1, 5)
	ref := makeH264MotionCompTightPicture(1, 15)
	refs := [2][]*h264PicturePlanes{{ref}}
	var cache macroblockMotionCache
	cache.Ref[0][h264Scan8[0]] = 0
	cache.MV[0][h264Scan8[0]] = [2]int16{1, 0}

	if err := h264HLMotionFrameWithScratch(dst, refs, &cache, MBType16x16|MBTypeP0L0, [4]uint32{}, 0, 0, 1, makeH264MotionCompScratch(dst)); err != nil {
		t.Fatal(err)
	}
	if dst.Y[0] != 18 {
		t.Fatalf("tight-stride edge luma sample = %d, want 18", dst.Y[0])
	}
	if dst.Cb[0] != 46 || dst.Cr[0] != 88 {
		t.Fatalf("tight-stride edge chroma samples = %d/%d, want 46/88", dst.Cb[0], dst.Cr[0])
	}
}

func makeH264MotionCompBipredCache(ref0 int8, ref1 int8) macroblockMotionCache {
	var cache macroblockMotionCache
	for n := 0; n < 16; n++ {
		cache.Ref[0][h264Scan8[n]] = ref0
		cache.Ref[1][h264Scan8[n]] = ref1
	}
	return cache
}

func makeH264MotionCompPicture(chromaFormatIDC int, seed int) *h264PicturePlanes {
	chromaStride := h264MotionCompTestChromaStride
	if chromaFormatIDC == 3 {
		chromaStride = h264MotionCompTestLumaStride
	}
	p := &h264PicturePlanes{
		Y:               make([]uint8, h264MotionCompTestLumaStride*h264MotionCompTestMBHeight*16),
		LumaStride:      h264MotionCompTestLumaStride,
		ChromaStride:    chromaStride,
		MBWidth:         h264MotionCompTestMBWidth,
		MBHeight:        h264MotionCompTestMBHeight,
		ChromaFormatIDC: chromaFormatIDC,
	}
	fillH264MotionCompPlane(p.Y, seed)
	if chromaFormatIDC != 0 {
		_, chromaHeight := h264ChromaFrameSize(p.MBWidth, p.MBHeight, chromaFormatIDC)
		p.Cb = make([]uint8, chromaStride*chromaHeight)
		p.Cr = make([]uint8, chromaStride*chromaHeight)
		fillH264MotionCompPlane(p.Cb, seed+29)
		fillH264MotionCompPlane(p.Cr, seed+71)
	}
	return p
}

func makeH264MotionCompPictureHigh(chromaFormatIDC int, bitDepth int, seed int) *h264PicturePlanesHigh {
	chromaStride := h264MotionCompTestChromaStride
	if chromaFormatIDC == 3 {
		chromaStride = h264MotionCompTestLumaStride
	}
	p := &h264PicturePlanesHigh{
		Y:               make([]uint16, h264MotionCompTestLumaStride*h264MotionCompTestMBHeight*16),
		LumaStride:      h264MotionCompTestLumaStride,
		ChromaStride:    chromaStride,
		MBWidth:         h264MotionCompTestMBWidth,
		MBHeight:        h264MotionCompTestMBHeight,
		ChromaFormatIDC: chromaFormatIDC,
	}
	fillH264MotionCompPlaneHigh(p.Y, seed, bitDepth)
	if chromaFormatIDC != 0 {
		_, chromaHeight := h264ChromaFrameSize(p.MBWidth, p.MBHeight, chromaFormatIDC)
		p.Cb = make([]uint16, chromaStride*chromaHeight)
		p.Cr = make([]uint16, chromaStride*chromaHeight)
		fillH264MotionCompPlaneHigh(p.Cb, seed+29, bitDepth)
		fillH264MotionCompPlaneHigh(p.Cr, seed+71, bitDepth)
	}
	return p
}

func makeH264MotionCompTightPicture(chromaFormatIDC int, seed int) *h264PicturePlanes {
	chromaWidth, chromaHeight := h264ChromaFrameSize(1, 1, chromaFormatIDC)
	p := &h264PicturePlanes{
		Y:               make([]uint8, 16*16),
		LumaStride:      16,
		ChromaStride:    chromaWidth,
		MBWidth:         1,
		MBHeight:        1,
		ChromaFormatIDC: chromaFormatIDC,
	}
	fillH264MotionCompPlane(p.Y, seed)
	if chromaFormatIDC != 0 {
		p.Cb = make([]uint8, chromaWidth*chromaHeight)
		p.Cr = make([]uint8, chromaWidth*chromaHeight)
		fillH264MotionCompPlane(p.Cb, seed+29)
		fillH264MotionCompPlane(p.Cr, seed+71)
	}
	return p
}

func makeH264MotionCompScratch(p *h264PicturePlanes) *h264MotionCompScratch {
	s := &h264MotionCompScratch{
		Y:    make([]uint8, p.LumaStride*16),
		Edge: make([]uint8, h264MotionCompScratchEdgeSize(p)),
	}
	if p.ChromaFormatIDC != 0 {
		_, chromaHeight := h264ChromaFrameSize(1, 1, p.ChromaFormatIDC)
		s.Cb = make([]uint8, p.ChromaStride*chromaHeight)
		s.Cr = make([]uint8, p.ChromaStride*chromaHeight)
	}
	return s
}

func makeH264MotionCompScratchHigh(p *h264PicturePlanesHigh) *h264MotionCompScratchHigh {
	s := &h264MotionCompScratchHigh{
		Y:    make([]uint16, p.LumaStride*16),
		Edge: make([]uint16, h264MotionCompScratchEdgeSizeHigh(p)),
	}
	if p.ChromaFormatIDC != 0 {
		_, chromaHeight := h264ChromaFrameSize(1, 1, p.ChromaFormatIDC)
		s.Cb = make([]uint16, p.ChromaStride*chromaHeight)
		s.Cr = make([]uint16, p.ChromaStride*chromaHeight)
	}
	return s
}

func h264MotionCompScratchEdgeSize(p *h264PicturePlanes) int {
	luma := h264EdgeScratchSize(p.LumaStride, 16+5, 16+5)
	chroma := 0
	if p.ChromaFormatIDC == 1 || p.ChromaFormatIDC == 2 {
		chroma = h264EdgeScratchSize(p.ChromaStride, 9, 8*p.ChromaFormatIDC+1)
	} else if p.ChromaFormatIDC == 3 {
		chroma = h264EdgeScratchSize(p.ChromaStride, 16+5, 16+5)
	}
	if chroma > luma {
		return chroma
	}
	return luma
}

func h264MotionCompScratchEdgeSizeHigh(p *h264PicturePlanesHigh) int {
	luma := h264EdgeScratchSize(p.LumaStride, 16+5, 16+5)
	chroma := 0
	if p.ChromaFormatIDC == 1 || p.ChromaFormatIDC == 2 {
		chroma = h264EdgeScratchSize(p.ChromaStride, 9, 8*p.ChromaFormatIDC+1)
	} else if p.ChromaFormatIDC == 3 {
		chroma = h264EdgeScratchSize(p.ChromaStride, 16+5, 16+5)
	}
	if chroma > luma {
		return chroma
	}
	return luma
}

func h264MotionCompTestPWT(chromaFormatIDC int) PredWeightTable {
	pwt := PredWeightTable{}
	pwt.LumaLog2WeightDenom = 0
	pwt.ChromaLog2WeightDenom = 0
	for ref := 0; ref < 48; ref++ {
		for list := 0; list < 2; list++ {
			pwt.LumaWeight[ref][list] = [2]int32{1, 0}
			for c := 0; c < 2; c++ {
				pwt.ChromaWeight[ref][list][c] = [2]int32{1, 0}
			}
		}
		for ref1 := 0; ref1 < 48; ref1++ {
			pwt.ImplicitWeight[ref][ref1] = [2]int32{32, 32}
		}
	}
	if chromaFormatIDC == 0 {
		pwt.ChromaLog2WeightDenom = 0
	}
	return pwt
}

func fillH264MotionCompPlane(p []uint8, seed int) {
	for i := range p {
		p[i] = uint8((seed + i*13 + (i>>4)*7) & 255)
	}
}

func fillH264MotionCompPlaneHigh(p []uint16, seed int, bitDepth int) {
	mask := (1 << bitDepth) - 1
	for i := range p {
		p[i] = uint16((seed + i*13 + (i>>4)*7) & mask)
	}
}

func benchmarkH264EmulatedEdgeMC(b *testing.B, srcX int, srcY int) {
	const (
		srcStride = 80
		bufStride = 80
		width     = 64
		height    = 64
		blockW    = 21
		blockH    = 21
	)
	src := make([]uint8, srcStride*height)
	buf := make([]uint8, h264EdgeScratchSize(bufStride, blockW, blockH))
	fillH264MotionCompPlane(src, 17)
	b.ReportAllocs()
	b.SetBytes(blockW * blockH)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := h264EmulatedEdgeMC(buf, 0, bufStride, src, srcStride, blockW, blockH, srcX, srcY, width, height); err != nil {
			b.Fatal(err)
		}
	}
}

func benchmarkH264EmulatedEdgeMCHigh(b *testing.B, srcX int, srcY int) {
	const (
		bitDepth  = 10
		srcStride = 80
		bufStride = 80
		width     = 64
		height    = 64
		blockW    = 21
		blockH    = 21
	)
	src := make([]uint16, srcStride*height)
	buf := make([]uint16, h264EdgeScratchSize(bufStride, blockW, blockH))
	fillH264MotionCompPlaneHigh(src, 17, bitDepth)
	b.ReportAllocs()
	b.SetBytes(blockW * blockH * 2)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := h264EmulatedEdgeMCHigh(buf, 0, bufStride, src, srcStride, blockW, blockH, srcX, srcY, width, height); err != nil {
			b.Fatal(err)
		}
	}
}

func requireH264BlockEqual(t *testing.T, got []uint8, want []uint8, stride int, gotOff int, wantOff int, width int, height int) {
	t.Helper()
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			g := got[gotOff+y*stride+x]
			w := want[wantOff+y*stride+x]
			if g != w {
				t.Fatalf("block[%d,%d] = %d, want %d", x, y, g, w)
			}
		}
	}
}

func requireH264BlockEqualHigh(t *testing.T, got []uint16, want []uint16, stride int, gotOff int, wantOff int, width int, height int) {
	t.Helper()
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			g := got[gotOff+y*stride+x]
			w := want[wantOff+y*stride+x]
			if g != w {
				t.Fatalf("block[%d,%d] = %d, want %d", x, y, g, w)
			}
		}
	}
}

func requireH264MotionCompMBEqual(t *testing.T, got *h264PicturePlanes, want *h264PicturePlanes, mbX int, mbY int) {
	t.Helper()
	yOff := mbY*16*got.LumaStride + mbX*16
	requireH264BlockEqual(t, got.Y, want.Y, got.LumaStride, yOff, yOff, 16, 16)
	if got.ChromaFormatIDC == 0 {
		return
	}
	chromaWidth, chromaHeight := h264ChromaFrameSize(1, 1, got.ChromaFormatIDC)
	_, cbOff, crOff, err := h264MBDestPartOffsets(got, mbX, mbY, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	requireH264BlockEqual(t, got.Cb, want.Cb, got.ChromaStride, cbOff, cbOff, chromaWidth, chromaHeight)
	requireH264BlockEqual(t, got.Cr, want.Cr, got.ChromaStride, crOff, crOff, chromaWidth, chromaHeight)
}

func requireH264MotionCompMBEqualHigh(t *testing.T, got *h264PicturePlanesHigh, want *h264PicturePlanesHigh, mbX int, mbY int) {
	t.Helper()
	yOff := mbY*16*got.LumaStride + mbX*16
	requireH264BlockEqualHigh(t, got.Y, want.Y, got.LumaStride, yOff, yOff, 16, 16)
	if got.ChromaFormatIDC == 0 {
		return
	}
	chromaWidth, chromaHeight := h264ChromaFrameSize(1, 1, got.ChromaFormatIDC)
	_, cbOff, crOff, err := h264MBDestPartOffsetsHigh(got, mbX, mbY, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	requireH264BlockEqualHigh(t, got.Cb, want.Cb, got.ChromaStride, cbOff, cbOff, chromaWidth, chromaHeight)
	requireH264BlockEqualHigh(t, got.Cr, want.Cr, got.ChromaStride, crOff, crOff, chromaWidth, chromaHeight)
}

func applyH264ImplicitBiweight16x8(t *testing.T, dst *h264PicturePlanes, src *h264PicturePlanes, mbX int, mbY int, weight0 int) {
	t.Helper()
	const log2Denom = 5
	weight1 := 64 - weight0
	for _, yOffset := range []int{0, 4} {
		dstY, dstCb, dstCr, err := h264MBDestPartOffsets(dst, mbX, mbY, 0, yOffset)
		if err != nil {
			t.Fatal(err)
		}
		srcY, srcCb, srcCr, err := h264MBDestPartOffsets(src, mbX, mbY, 0, yOffset)
		if err != nil {
			t.Fatal(err)
		}
		if err := h264BiweightPixels(dst.Y[dstY:], src.Y[srcY:], dst.LumaStride, 8, log2Denom, weight0, weight1, 0, 16); err != nil {
			t.Fatal(err)
		}
		if dst.ChromaFormatIDC == 0 {
			continue
		}
		chromaHeight, chromaWidth, err := h264ChromaWeightGeometry(dst.ChromaFormatIDC, 8, 8, 16)
		if err != nil {
			t.Fatal(err)
		}
		if err := h264BiweightPixels(dst.Cb[dstCb:], src.Cb[srcCb:], dst.ChromaStride, chromaHeight, log2Denom, weight0, weight1, 0, chromaWidth); err != nil {
			t.Fatal(err)
		}
		if err := h264BiweightPixels(dst.Cr[dstCr:], src.Cr[srcCr:], dst.ChromaStride, chromaHeight, log2Denom, weight0, weight1, 0, chromaWidth); err != nil {
			t.Fatal(err)
		}
	}
}

func applyH264ImplicitBiweight16x8High(t *testing.T, dst *h264PicturePlanesHigh, src *h264PicturePlanesHigh, mbX int, mbY int, weight0 int, bitDepth int) {
	t.Helper()
	const log2Denom = 5
	weight1 := 64 - weight0
	for _, yOffset := range []int{0, 4} {
		dstY, dstCb, dstCr, err := h264MBDestPartOffsetsHigh(dst, mbX, mbY, 0, yOffset)
		if err != nil {
			t.Fatal(err)
		}
		srcY, srcCb, srcCr, err := h264MBDestPartOffsetsHigh(src, mbX, mbY, 0, yOffset)
		if err != nil {
			t.Fatal(err)
		}
		if err := h264BiweightPixelsHigh(dst.Y[dstY:], src.Y[srcY:], dst.LumaStride, 8, log2Denom, weight0, weight1, 0, 16, bitDepth); err != nil {
			t.Fatal(err)
		}
		if dst.ChromaFormatIDC == 0 {
			continue
		}
		chromaHeight, chromaWidth, err := h264ChromaWeightGeometry(dst.ChromaFormatIDC, 8, 8, 16)
		if err != nil {
			t.Fatal(err)
		}
		if err := h264BiweightPixelsHigh(dst.Cb[dstCb:], src.Cb[srcCb:], dst.ChromaStride, chromaHeight, log2Denom, weight0, weight1, 0, chromaWidth, bitDepth); err != nil {
			t.Fatal(err)
		}
		if err := h264BiweightPixelsHigh(dst.Cr[dstCr:], src.Cr[srcCr:], dst.ChromaStride, chromaHeight, log2Denom, weight0, weight1, 0, chromaWidth, bitDepth); err != nil {
			t.Fatal(err)
		}
	}
}

func applyH264ImplicitBiweight8x8Subparts(t *testing.T, dst *h264PicturePlanes, src *h264PicturePlanes, mbX int, mbY int, weight0 int) {
	t.Helper()
	const log2Denom = 5
	weight1 := 64 - weight0
	for i := 0; i < 4; i++ {
		xOffset := (i & 1) << 2
		yOffset := (i & 2) << 1
		dstY, dstCb, dstCr, err := h264MBDestPartOffsets(dst, mbX, mbY, xOffset, yOffset)
		if err != nil {
			t.Fatal(err)
		}
		srcY, srcCb, srcCr, err := h264MBDestPartOffsets(src, mbX, mbY, xOffset, yOffset)
		if err != nil {
			t.Fatal(err)
		}
		if err := h264BiweightPixels(dst.Y[dstY:], src.Y[srcY:], dst.LumaStride, 8, log2Denom, weight0, weight1, 0, 8); err != nil {
			t.Fatal(err)
		}
		if dst.ChromaFormatIDC == 0 {
			continue
		}
		chromaHeight, chromaWidth, err := h264ChromaWeightGeometry(dst.ChromaFormatIDC, 8, 4, 8)
		if err != nil {
			t.Fatal(err)
		}
		if err := h264BiweightPixels(dst.Cb[dstCb:], src.Cb[srcCb:], dst.ChromaStride, chromaHeight, log2Denom, weight0, weight1, 0, chromaWidth); err != nil {
			t.Fatal(err)
		}
		if err := h264BiweightPixels(dst.Cr[dstCr:], src.Cr[srcCr:], dst.ChromaStride, chromaHeight, log2Denom, weight0, weight1, 0, chromaWidth); err != nil {
			t.Fatal(err)
		}
	}
}

func applyH264ImplicitBiweight8x8SubpartsHigh(t *testing.T, dst *h264PicturePlanesHigh, src *h264PicturePlanesHigh, mbX int, mbY int, weight0 int, bitDepth int) {
	t.Helper()
	const log2Denom = 5
	weight1 := 64 - weight0
	for i := 0; i < 4; i++ {
		xOffset := (i & 1) << 2
		yOffset := (i & 2) << 1
		dstY, dstCb, dstCr, err := h264MBDestPartOffsetsHigh(dst, mbX, mbY, xOffset, yOffset)
		if err != nil {
			t.Fatal(err)
		}
		srcY, srcCb, srcCr, err := h264MBDestPartOffsetsHigh(src, mbX, mbY, xOffset, yOffset)
		if err != nil {
			t.Fatal(err)
		}
		if err := h264BiweightPixelsHigh(dst.Y[dstY:], src.Y[srcY:], dst.LumaStride, 8, log2Denom, weight0, weight1, 0, 8, bitDepth); err != nil {
			t.Fatal(err)
		}
		if dst.ChromaFormatIDC == 0 {
			continue
		}
		chromaHeight, chromaWidth, err := h264ChromaWeightGeometry(dst.ChromaFormatIDC, 8, 4, 8)
		if err != nil {
			t.Fatal(err)
		}
		if err := h264BiweightPixelsHigh(dst.Cb[dstCb:], src.Cb[srcCb:], dst.ChromaStride, chromaHeight, log2Denom, weight0, weight1, 0, chromaWidth, bitDepth); err != nil {
			t.Fatal(err)
		}
		if err := h264BiweightPixelsHigh(dst.Cr[dstCr:], src.Cr[srcCr:], dst.ChromaStride, chromaHeight, log2Denom, weight0, weight1, 0, chromaWidth, bitDepth); err != nil {
			t.Fatal(err)
		}
	}
}
