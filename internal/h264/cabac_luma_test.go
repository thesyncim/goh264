// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

import "testing"

func TestDecodeCABACLumaResidualInter4x4(t *testing.T) {
	pps := cavlcFlatQMulPPS()
	var ctx cavlcResidualContext
	src := &scriptedCABACSource{
		bits:  []int{1, 1, 1, 0, 0, 0, 0},
		signs: []int32{64},
	}

	ret, err := ctx.decodeCABACLumaResidual(src, pps, h264ZigzagScanCAVLC[:], h264ZigzagScan8x8CAVLC[:], MBType16x16|MBTypeP0L0, 1, 0, 0, 0, 0, false, false)
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
	wantIndexes(t, src, []int{93, 134, 195, 248, 94, 95, 93})
}

func TestDecodeCABACLumaResidualIntra16x16DC(t *testing.T) {
	pps := cavlcFlatQMulPPS()
	var ctx cavlcResidualContext
	src := &scriptedCABACSource{
		bits:  []int{1, 1, 1, 0},
		signs: []int32{1},
	}

	ret, err := ctx.decodeCABACLumaResidual(src, pps, h264ZigzagScanCAVLC[:], h264ZigzagScan8x8CAVLC[:], MBTypeIntra16x16, 0, 0, 0, 0, 0, false, false)
	if err != nil {
		t.Fatalf("decode intra16 luma residual failed: %v", err)
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
	for _, n := range []int{0, 1, 2, 3} {
		if ctx.NonZeroCountCache[h264Scan8[n]] != 0 {
			t.Fatalf("ac nnz block%d = %d, want 0", n, ctx.NonZeroCountCache[h264Scan8[n]])
		}
	}
	wantIndexes(t, src, []int{85, 105, 166, 228})
}

func TestDecodeCABACLumaResidualClearsSkipped8x8(t *testing.T) {
	pps := cavlcFlatQMulPPS()
	var ctx cavlcResidualContext
	for _, n := range []int{0, 1, 2, 3} {
		ctx.NonZeroCountCache[h264Scan8[n]] = 7
	}
	src := &scriptedCABACSource{}

	ret, err := ctx.decodeCABACLumaResidual(src, pps, h264ZigzagScanCAVLC[:], h264ZigzagScan8x8CAVLC[:], MBType16x16|MBTypeP0L0, 0, 0, 0, 0, 0, false, false)
	if err != nil {
		t.Fatalf("decode skipped luma residual failed: %v", err)
	}
	if ret != 0 {
		t.Fatalf("ret cbp = %d, want 0", ret)
	}
	for _, n := range []int{0, 1, 2, 3} {
		if ctx.NonZeroCountCache[h264Scan8[n]] != 0 {
			t.Fatalf("nnz block%d = %d, want 0", n, ctx.NonZeroCountCache[h264Scan8[n]])
		}
	}
	wantIndexes(t, src, nil)
}
