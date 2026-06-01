// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

import "testing"

func TestH264ApplyLoopFilterEdge444UsesLumaChromaPlanes(t *testing.T) {
	const stride = 32
	dst := &h264PicturePlanes{
		Y:               make([]uint8, stride*16),
		Cb:              make([]uint8, stride*16),
		Cr:              make([]uint8, stride*16),
		LumaStride:      stride,
		ChromaStride:    stride,
		MBWidth:         1,
		MBHeight:        1,
		ChromaFormatIDC: 3,
	}
	fill444LoopFilterStep(dst.Cb, stride, 4, 100, 110)
	fill444LoopFilterStep(dst.Cr, stride, 4, 80, 92)
	cbBefore := dst.Cb[3]
	crBefore := dst.Cr[3]

	if err := h264ApplyLoopFilterEdge(dst, 0, 0, 0, 0, 1, [4]int16{3, 3, 3, 3}, 30, [2]int{30, 30}, h264LoopFilterSliceParams{}, false, false, true); err != nil {
		t.Fatal(err)
	}
	if dst.Cb[3] == cbBefore || dst.Cr[3] == crBefore {
		t.Fatalf("4:4:4 chroma planes were not filtered: cb %d->%d cr %d->%d", cbBefore, dst.Cb[3], crBefore, dst.Cr[3])
	}
}

func TestH264LoopFilterThresholdsHighBitDepthQPBDOffset(t *testing.T) {
	alpha8, beta8, index8, err := h264LoopFilterThresholdsForBitDepth(30, 0, 0, 8)
	if err != nil {
		t.Fatal(err)
	}
	alpha10, beta10, index10, err := h264LoopFilterThresholdsForBitDepth(42, 0, 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	if alpha10 != alpha8 || beta10 != beta8 || index10 != index8 {
		t.Fatalf("High10 qp_bd_offset mapping = alpha %d beta %d index %d, want %d/%d/%d", alpha10, beta10, index10, alpha8, beta8, index8)
	}
	if alpha, beta, index, err := h264LoopFilterThresholdsForBitDepth(11, 0, 0, 10); err != nil || alpha != 0 || beta != 0 || index != 0 {
		t.Fatalf("High10 low qp threshold = alpha %d beta %d index %d err %v, want 0/0/0/nil", alpha, beta, index, err)
	}
	if alpha, beta, index, err := h264LoopFilterThresholdsForBitDepth(63, 0, 0, 10); err != nil || alpha != 255 || beta != 18 || index != 51 {
		t.Fatalf("High10 high qp threshold = alpha %d beta %d index %d err %v, want 255/18/51/nil", alpha, beta, index, err)
	}
}

func TestH264ApplyLoopFilterEdgeHigh420MutatesLumaChroma(t *testing.T) {
	const (
		lumaStride   = 32
		chromaStride = 16
		bitDepth     = 10
	)
	dst := &h264PicturePlanesHigh{
		Y:               make([]uint16, lumaStride*16),
		Cb:              make([]uint16, chromaStride*8),
		Cr:              make([]uint16, chromaStride*8),
		LumaStride:      lumaStride,
		ChromaStride:    chromaStride,
		MBWidth:         2,
		MBHeight:        1,
		ChromaFormatIDC: 1,
	}
	fillHighLoopFilterStep(dst.Y, lumaStride, 16, 16, 8, 400, 408)
	fillHighLoopFilterStep(dst.Cb, chromaStride, 16, 8, 4, 300, 308)
	fillHighLoopFilterStep(dst.Cr, chromaStride, 16, 8, 4, 200, 208)
	yBefore := [2]uint16{dst.Y[7], dst.Y[8]}
	cbBefore := [2]uint16{dst.Cb[3], dst.Cb[4]}
	crBefore := [2]uint16{dst.Cr[3], dst.Cr[4]}

	if err := h264ApplyLoopFilterEdgeHigh(dst, 0, 0, 0, 0, 2, [4]int16{3, 3, 3, 3}, 30, [2]int{30, 30}, h264LoopFilterSliceParams{}, false, true, true, bitDepth); err != nil {
		t.Fatal(err)
	}
	if dst.Y[7] == yBefore[0] || dst.Y[8] == yBefore[1] {
		t.Fatalf("High10 luma edge did not filter: %v -> [%d %d]", yBefore, dst.Y[7], dst.Y[8])
	}
	if dst.Cb[3] == cbBefore[0] || dst.Cb[4] == cbBefore[1] || dst.Cr[3] == crBefore[0] || dst.Cr[4] == crBefore[1] {
		t.Fatalf("High10 chroma edge did not filter: cb %v -> [%d %d] cr %v -> [%d %d]",
			cbBefore, dst.Cb[3], dst.Cb[4], crBefore, dst.Cr[3], dst.Cr[4])
	}
}

func TestMacroblockTablesFilterFrameHighDeblocksBoundary(t *testing.T) {
	const (
		mbWidth  = 2
		mbHeight = 1
		bitDepth = 10
	)
	m, err := newMacroblockTables(mbWidth, mbHeight, 1)
	if err != nil {
		t.Fatal(err)
	}
	dst := &h264PicturePlanesHigh{
		Y:               make([]uint16, 32*16),
		Cb:              make([]uint16, 16*8),
		Cr:              make([]uint16, 16*8),
		LumaStride:      32,
		ChromaStride:    16,
		MBWidth:         mbWidth,
		MBHeight:        mbHeight,
		ChromaFormatIDC: 1,
	}
	fillHighLoopFilterStep(dst.Y, dst.LumaStride, 32, 16, 16, 400, 408)
	fillHighLoopFilterStep(dst.Cb, dst.ChromaStride, 16, 8, 8, 300, 308)
	fillHighLoopFilterStep(dst.Cr, dst.ChromaStride, 16, 8, 8, 200, 208)
	for mbXY := 0; mbXY < mbWidth*mbHeight; mbXY++ {
		m.MacroblockTyp[mbXY] = MBTypeIntra16x16
		m.QScaleTable[mbXY] = 30
		m.SliceTable[mbXY] = 0
	}
	pps := cavlcFlatQMulPPS()
	pps.SPS = &SPS{
		BitDepthLuma:     bitDepth,
		BitDepthChroma:   bitDepth,
		ChromaFormatIDC:  1,
		FrameMBSOnlyFlag: 1,
	}
	params := []h264LoopFilterSliceParams{{
		PPS:              pps,
		ListCount:        1,
		DeblockingFilter: 1,
	}}
	yBefore := [2]uint16{dst.Y[15], dst.Y[16]}
	cbBefore := [2]uint16{dst.Cb[7], dst.Cb[8]}
	crBefore := [2]uint16{dst.Cr[7], dst.Cr[8]}

	if err := m.filterFrameHigh(dst, params); err != nil {
		t.Fatal(err)
	}
	if dst.Y[15] == yBefore[0] || dst.Y[16] == yBefore[1] {
		t.Fatalf("High10 frame luma boundary did not filter: %v -> [%d %d]", yBefore, dst.Y[15], dst.Y[16])
	}
	if dst.Cb[7] == cbBefore[0] || dst.Cb[8] == cbBefore[1] || dst.Cr[7] == crBefore[0] || dst.Cr[8] == crBefore[1] {
		t.Fatalf("High10 frame chroma boundary did not filter: cb %v -> [%d %d] cr %v -> [%d %d]",
			cbBefore, dst.Cb[7], dst.Cb[8], crBefore, dst.Cr[7], dst.Cr[8])
	}
}

func fill444LoopFilterStep(pix []uint8, stride int, edge int, left uint8, right uint8) {
	for y := 0; y < 16; y++ {
		row := y * stride
		for x := 0; x < edge; x++ {
			pix[row+x] = left
		}
		for x := edge; x < 16; x++ {
			pix[row+x] = right
		}
	}
}

func fillHighLoopFilterStep(pix []uint16, stride int, width int, height int, edge int, left uint16, right uint16) {
	for y := 0; y < height; y++ {
		row := y * stride
		for x := 0; x < edge; x++ {
			pix[row+x] = left
		}
		for x := edge; x < width; x++ {
			pix[row+x] = right
		}
	}
}
