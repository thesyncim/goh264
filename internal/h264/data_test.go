// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

import "testing"

func TestMBTypeTables(t *testing.T) {
	if h264IMBTypeInfo[0] != (IMBInfo{Type: MBTypeIntra4x4, PredMode: -1, CBP: -1}) {
		t.Fatalf("i mb[0] = %+v", h264IMBTypeInfo[0])
	}
	if h264IMBTypeInfo[25].Type != MBTypeIntraPCM {
		t.Fatalf("i mb[25] type = %#x", h264IMBTypeInfo[25].Type)
	}
	if h264PMBTypeInfo[4].Type != MBType8x8|MBTypeP0L0|MBTypeP1L0|MBTypeRef0 {
		t.Fatalf("p mb[4] type = %#x", h264PMBTypeInfo[4].Type)
	}
	if h264BMBTypeInfo[0].Type != MBTypeDirect2|MBTypeL0L1 {
		t.Fatalf("b mb[0] type = %#x", h264BMBTypeInfo[0].Type)
	}
	if h264BSubMBTypeInfo[12].PartitionCount != 4 {
		t.Fatalf("b sub mb[12] partitions = %d", h264BSubMBTypeInfo[12].PartitionCount)
	}
}

func TestChromaQPTableShape(t *testing.T) {
	cases := []struct {
		depth int32
		qp    uint32
		want  uint8
	}{
		{8, 29, 29},
		{8, 30, 29},
		{8, 39, 35},
		{8, 51, 39},
		{10, 11, 11},
		{10, 12, 12},
		{10, 42, 41},
		{14, 35, 35},
		{14, 36, 36},
		{14, 87, 75},
	}

	for _, tc := range cases {
		if got := h264ChromaQP(tc.depth, tc.qp); got != tc.want {
			t.Fatalf("chroma qp depth=%d qp=%d got=%d want=%d", tc.depth, tc.qp, got, tc.want)
		}
	}
}

func TestDequantTablesFlatScaling(t *testing.T) {
	sps := &SPS{BitDepthLuma: 8, ChromaFormatIDC: 1}
	initFlatScalingMatrices(&sps.ScalingMatrix4, &sps.ScalingMatrix8)
	pps := &PPS{SPS: sps, ScalingMatrix4: sps.ScalingMatrix4, ScalingMatrix8: sps.ScalingMatrix8}

	initDequantTables(pps, sps)

	if got, want := pps.Dequant4Buffer[0][0][0], uint32(640); got != want {
		t.Fatalf("dequant4[0][0][0] = %d, want %d", got, want)
	}
	if got, want := pps.Dequant4Buffer[5][10][15], uint32(3200); got != want {
		t.Fatalf("dequant4[5][10][15] = %d, want %d", got, want)
	}

	pps.Transform8x8Mode = 1
	initDequantTables(pps, sps)
	if got, want := pps.Dequant8Buffer[0][0][0], uint32(320); got != want {
		t.Fatalf("dequant8[0][0][0] = %d, want %d", got, want)
	}
	if got, want := pps.Dequant8Buffer[5][10][63], uint32(896); got != want {
		t.Fatalf("dequant8[5][10][63] = %d, want %d", got, want)
	}
}

func TestScanTables(t *testing.T) {
	if h264Scan8[0] != 12 || h264Scan8[15] != 39 || h264Scan8[48] != 0 || h264Scan8[50] != 80 {
		t.Fatalf("scan8 spots = %d %d %d %d", h264Scan8[0], h264Scan8[15], h264Scan8[48], h264Scan8[50])
	}
	if h264ZigzagScanCAVLC[1] != 4 || h264ZigzagScanCAVLC[2] != 1 || h264ZigzagScanCAVLC[15] != 15 {
		t.Fatalf("zigzag cavlc spots = %d %d %d", h264ZigzagScanCAVLC[1], h264ZigzagScanCAVLC[2], h264ZigzagScanCAVLC[15])
	}
	if h264ZigzagScan8x8CAVLC[1] != 9 || h264ZigzagScan8x8CAVLC[7] != 56 || h264ZigzagScan8x8CAVLC[63] != 63 {
		t.Fatalf("zigzag8x8 cavlc spots = %d %d %d", h264ZigzagScan8x8CAVLC[1], h264ZigzagScan8x8CAVLC[7], h264ZigzagScan8x8CAVLC[63])
	}
	if h264ChromaDCScan != ([4]uint8{0, 16, 32, 48}) {
		t.Fatalf("chroma dc scan = %v", h264ChromaDCScan)
	}
	if h264Chroma422DCScan != ([8]uint8{0, 32, 16, 64, 96, 48, 80, 112}) {
		t.Fatalf("chroma422 dc scan = %v", h264Chroma422DCScan)
	}
}
