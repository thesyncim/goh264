// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

import "testing"

func TestPredNonZeroCount(t *testing.T) {
	var ctx cavlcResidualContext
	ctx.NonZeroCountCache[h264Scan8[0]-1] = 3
	ctx.NonZeroCountCache[h264Scan8[0]-8] = 4
	if got := ctx.predNonZeroCount(0); got != 4 {
		t.Fatalf("pred nnz = %d, want 4", got)
	}

	ctx.NonZeroCountCache[h264Scan8[1]-1] = 64
	ctx.NonZeroCountCache[h264Scan8[1]-8] = 5
	if got := ctx.predNonZeroCount(1); got != 5 {
		t.Fatalf("unavailable pred nnz = %d, want 5", got)
	}
}

func TestDecodeCAVLCLumaResidualInter4x4(t *testing.T) {
	pps := cavlcFlatQMulPPS()
	var ctx cavlcResidualContext
	gb := newBitReader(cavlcBitString("0101111"))

	ret, err := ctx.decodeLumaResidual(&gb, pps, h264ZigzagScanCAVLC[:], h264ZigzagScan8x8CAVLC[:], MBType16x16|MBTypeP0L0, 1, 0, 0)
	if err != nil {
		t.Fatalf("decode luma residual failed: %v", err)
	}
	if ret != 1 {
		t.Fatalf("ret cbp = %d, want 1", ret)
	}
	if ctx.MB[0] != 1 {
		t.Fatalf("mb[0] = %d, want 1", ctx.MB[0])
	}
	if ctx.NonZeroCountCache[h264Scan8[0]] != 1 {
		t.Fatalf("nnz block0 = %d, want 1", ctx.NonZeroCountCache[h264Scan8[0]])
	}
	for _, n := range []int{1, 2, 3} {
		if ctx.NonZeroCountCache[h264Scan8[n]] != 0 {
			t.Fatalf("nnz block%d = %d, want 0", n, ctx.NonZeroCountCache[h264Scan8[n]])
		}
	}
}

func TestDecodeCAVLCLumaResidualIntra16x16DC(t *testing.T) {
	pps := cavlcFlatQMulPPS()
	var ctx cavlcResidualContext
	gb := newBitReader(cavlcBitString("0101"))

	ret, err := ctx.decodeLumaResidual(&gb, pps, h264ZigzagScanCAVLC[:], h264ZigzagScan8x8CAVLC[:], MBTypeIntra16x16, 0, 0, 0)
	if err != nil {
		t.Fatalf("decode luma residual failed: %v", err)
	}
	if ret != 0 {
		t.Fatalf("ret cbp = %d, want 0", ret)
	}
	if ctx.MBLumaDC[0][0] != 1 {
		t.Fatalf("luma dc[0] = %d, want 1", ctx.MBLumaDC[0][0])
	}
	if ctx.NonZeroCountCache[h264Scan8[lumaDCBlockIndex]] != 1 {
		t.Fatalf("dc nnz = %d, want 1", ctx.NonZeroCountCache[h264Scan8[lumaDCBlockIndex]])
	}
}

func TestDecodeCAVLCLumaResidualClearsSkipped8x8(t *testing.T) {
	pps := cavlcFlatQMulPPS()
	var ctx cavlcResidualContext
	for _, n := range []int{0, 1, 2, 3} {
		ctx.NonZeroCountCache[h264Scan8[n]] = 7
	}
	gb := newBitReader(cavlcBitString("1"))

	ret, err := ctx.decodeLumaResidual(&gb, pps, h264ZigzagScanCAVLC[:], h264ZigzagScan8x8CAVLC[:], MBType16x16|MBTypeP0L0, 0, 0, 0)
	if err != nil {
		t.Fatalf("decode luma residual failed: %v", err)
	}
	if ret != 0 {
		t.Fatalf("ret cbp = %d, want 0", ret)
	}
	for _, n := range []int{0, 1, 2, 3} {
		if ctx.NonZeroCountCache[h264Scan8[n]] != 0 {
			t.Fatalf("nnz block%d = %d, want 0", n, ctx.NonZeroCountCache[h264Scan8[n]])
		}
	}
	if gb.bitPos != 0 {
		t.Fatalf("consumed %d bits, want 0", gb.bitPos)
	}
}

func cavlcFlatQMulPPS() *PPS {
	pps := &PPS{}
	for list := range pps.ChromaQPTable {
		for qp := range pps.ChromaQPTable[list] {
			pps.ChromaQPTable[list][qp] = uint8(qp)
		}
	}
	for cqm := range pps.Dequant4Buffer {
		for qp := range pps.Dequant4Buffer[cqm] {
			for i := range pps.Dequant4Buffer[cqm][qp] {
				pps.Dequant4Buffer[cqm][qp][i] = 64
			}
		}
	}
	for cqm := range pps.Dequant8Buffer {
		for qp := range pps.Dequant8Buffer[cqm] {
			for i := range pps.Dequant8Buffer[cqm][qp] {
				pps.Dequant8Buffer[cqm][qp][i] = 64
			}
		}
	}
	return pps
}
