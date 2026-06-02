// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

import "testing"

func TestDecodeCABACChromaResidual420DC(t *testing.T) {
	pps := cavlcFlatQMulPPS()
	var ctx cavlcResidualContext
	src := &scriptedCABACSource{
		bits:  []int{1, 1, 1, 0, 0},
		signs: []int32{1},
	}

	ret, err := ctx.decodeCABACChromaResidual(src, pps, h264ZigzagScanCAVLC[:], MBTypeIntra4x4, 0x10, 1, [2]uint8{0, 0}, 0, 0, false)
	if err != nil {
		t.Fatalf("decode chroma residual failed: %v", err)
	}
	if ret != 0x40 {
		t.Fatalf("ret cbp table bits = %#x, want 0x40", ret)
	}
	if ctx.MB[256] != 1 {
		t.Fatalf("chroma dc coeff = %d, want 1", ctx.MB[256])
	}
	if ctx.NonZeroCountCache[h264Scan8[chromaDCBlockIndex]] != 1 {
		t.Fatalf("chroma dc0 nnz = %d, want 1", ctx.NonZeroCountCache[h264Scan8[chromaDCBlockIndex]])
	}
	if ctx.NonZeroCountCache[h264Scan8[chromaDCBlockIndex+1]] != 0 {
		t.Fatalf("chroma dc1 nnz = %d, want 0", ctx.NonZeroCountCache[h264Scan8[chromaDCBlockIndex+1]])
	}
	wantIndexes(t, src, []int{97, 149, 210, 258, 97})
}

func TestDecodeCABACChromaResidualTypedDCTElemWidth(t *testing.T) {
	for _, tt := range []struct {
		name      string
		narrowDCT bool
		want      int32
	}{
		{name: "8-bit-dctelem", narrowDCT: true, want: dctcoef8(40000)},
		{name: "high-bit-depth-dctelem", narrowDCT: false, want: 40000},
	} {
		t.Run(tt.name, func(t *testing.T) {
			pps := cavlcFlatQMulPPS()
			var ctx cavlcResidualContext
			src := &scriptedCABACSource{
				bits:  []int{1, 1, 1, 0, 0},
				signs: []int32{40000},
			}

			ret, err := ctx.decodeCABACChromaResidualTyped(src, pps, h264ZigzagScanCAVLC[:], MBTypeIntra4x4, 0x10, 1, [2]uint8{0, 0}, 0, 0, false, tt.narrowDCT)
			if err != nil {
				t.Fatalf("decode chroma typed residual failed: %v", err)
			}
			if ret != 0x40 {
				t.Fatalf("ret cbp table bits = %#x, want 0x40", ret)
			}
			if ctx.MB[256] != tt.want {
				t.Fatalf("chroma dc coeff = %d, want %d", ctx.MB[256], tt.want)
			}
			if ctx.NonZeroCountCache[h264Scan8[chromaDCBlockIndex]] != 1 {
				t.Fatalf("chroma dc0 nnz = %d, want 1", ctx.NonZeroCountCache[h264Scan8[chromaDCBlockIndex]])
			}
			wantIndexes(t, src, []int{97, 149, 210, 258, 97})
		})
	}
}

func TestDecodeCABACChromaResidualClearsSkippedChroma(t *testing.T) {
	pps := cavlcFlatQMulPPS()
	var ctx cavlcResidualContext
	for _, n := range []int{16, 17, 18, 19, 32, 33, 34, 35} {
		ctx.NonZeroCountCache[h264Scan8[n]] = 9
	}
	src := &scriptedCABACSource{}

	if ret, err := ctx.decodeCABACChromaResidual(src, pps, h264ZigzagScanCAVLC[:], MBTypeIntra4x4, 0, 1, [2]uint8{0, 0}, 0, 0, false); err != nil {
		t.Fatalf("decode skipped chroma residual failed: %v", err)
	} else if ret != 0 {
		t.Fatalf("ret cbp table bits = %#x, want 0", ret)
	}
	for _, n := range []int{16, 17, 18, 19, 32, 33, 34, 35} {
		if ctx.NonZeroCountCache[h264Scan8[n]] != 0 {
			t.Fatalf("chroma nnz block%d = %d, want 0", n, ctx.NonZeroCountCache[h264Scan8[n]])
		}
	}
	wantIndexes(t, src, nil)
}
