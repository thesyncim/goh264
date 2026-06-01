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
