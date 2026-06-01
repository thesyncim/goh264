// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

import (
	"errors"
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
