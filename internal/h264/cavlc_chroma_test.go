// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

import "testing"

func TestDecodeCAVLCChromaResidualDConly420(t *testing.T) {
	pps := cavlcFlatQMulPPS()
	var ctx cavlcResidualContext
	gb := newBitReader(cavlcBitString("10101"))

	err := ctx.decodeChromaResidual(&gb, pps, h264ZigzagScanCAVLC[:], MBType16x16|MBTypeP0L0, 0x10, 1, [2]uint8{})
	if err != nil {
		t.Fatalf("decode chroma residual failed: %v", err)
	}
	if ctx.MB[256] != 1 {
		t.Fatalf("chroma dc mb[256] = %d, want 1", ctx.MB[256])
	}
	if ctx.NonZeroCountCache[h264Scan8[chromaDCBlockIndex]] != 1 {
		t.Fatalf("chroma dc nnz = %d, want 1", ctx.NonZeroCountCache[h264Scan8[chromaDCBlockIndex]])
	}
	if ctx.NonZeroCountCache[h264Scan8[chromaDCBlockIndex+1]] != 0 {
		t.Fatalf("second chroma dc nnz = %d, want 0", ctx.NonZeroCountCache[h264Scan8[chromaDCBlockIndex+1]])
	}
}

func TestDecodeCAVLCChromaResidualDConly422(t *testing.T) {
	pps := cavlcFlatQMulPPS()
	var ctx cavlcResidualContext
	gb := newBitReader(cavlcBitString("01011"))

	err := ctx.decodeChromaResidual(&gb, pps, h264ZigzagScanCAVLC[:], MBType16x16|MBTypeP0L0, 0x10, 2, [2]uint8{})
	if err != nil {
		t.Fatalf("decode chroma residual failed: %v", err)
	}
	if ctx.MB[256] != 1 {
		t.Fatalf("chroma422 dc mb[256] = %d, want 1", ctx.MB[256])
	}
	if ctx.NonZeroCountCache[h264Scan8[chromaDCBlockIndex]] != 1 {
		t.Fatalf("chroma422 dc nnz = %d, want 1", ctx.NonZeroCountCache[h264Scan8[chromaDCBlockIndex]])
	}
}

func TestDecodeCAVLCChromaResidualAC420(t *testing.T) {
	pps := cavlcFlatQMulPPS()
	var ctx cavlcResidualContext
	gb := newBitReader(cavlcBitString("010101011111111"))

	err := ctx.decodeChromaResidual(&gb, pps, h264ZigzagScanCAVLC[:], MBType16x16|MBTypeP0L0, 0x20, 1, [2]uint8{})
	if err != nil {
		t.Fatalf("decode chroma residual failed: %v", err)
	}
	acPos := 16*16 + int(h264ZigzagScanCAVLC[1])
	if ctx.MB[acPos] != 1 {
		t.Fatalf("first chroma ac = %d, want 1", ctx.MB[acPos])
	}
	if ctx.NonZeroCountCache[h264Scan8[16]] != 1 {
		t.Fatalf("first chroma ac nnz = %d, want 1", ctx.NonZeroCountCache[h264Scan8[16]])
	}
	for _, n := range []int{17, 18, 19, 32, 33, 34, 35} {
		if ctx.NonZeroCountCache[h264Scan8[n]] != 0 {
			t.Fatalf("chroma ac nnz block%d = %d, want 0", n, ctx.NonZeroCountCache[h264Scan8[n]])
		}
	}
}

func TestDecodeCAVLCChromaResidualClearsAC(t *testing.T) {
	pps := cavlcFlatQMulPPS()
	var ctx cavlcResidualContext
	fillCAVLCNonZero(&ctx.NonZeroCountCache, int(h264Scan8[16]), 4, 4, 8, 9)
	fillCAVLCNonZero(&ctx.NonZeroCountCache, int(h264Scan8[32]), 4, 4, 8, 9)
	gb := newBitReader(cavlcBitString("1"))

	err := ctx.decodeChromaResidual(&gb, pps, h264ZigzagScanCAVLC[:], MBType16x16|MBTypeP0L0, 0, 1, [2]uint8{})
	if err != nil {
		t.Fatalf("decode chroma residual failed: %v", err)
	}
	for _, n := range []int{16, 17, 18, 19, 32, 33, 34, 35} {
		if ctx.NonZeroCountCache[h264Scan8[n]] != 0 {
			t.Fatalf("chroma nnz block%d = %d, want 0", n, ctx.NonZeroCountCache[h264Scan8[n]])
		}
	}
	if gb.bitPos != 0 {
		t.Fatalf("consumed %d bits, want 0", gb.bitPos)
	}
}
