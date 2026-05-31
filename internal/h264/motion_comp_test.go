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

func TestH264HLMotionFrameReturnsUnsupportedForEdgeEmulation(t *testing.T) {
	dst := makeH264MotionCompPicture(1, 5)
	ref := makeH264MotionCompPicture(1, 15)
	refs := [2][]*h264PicturePlanes{{ref}}
	var cache macroblockMotionCache
	cache.Ref[0][h264Scan8[0]] = 0
	cache.MV[0][h264Scan8[0]] = [2]int16{1, 0}

	err := h264HLMotionFrame(dst, refs, &cache, MBType16x16|MBTypeP0L0, [4]uint32{}, 0, 1, 1)
	if !errors.Is(err, ErrUnsupported) {
		t.Fatalf("edge-emulation error = %v, want ErrUnsupported", err)
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
