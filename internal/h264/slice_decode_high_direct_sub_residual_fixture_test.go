// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHigh10DirectResidualFixtureMacroblockSyntax(t *testing.T) {
	for _, tt := range []struct {
		name          string
		file          string
		cabac         bool
		wantDirectSub bool
		cbpTable      int
	}{
		{name: "cavlc-direct-sub", file: "high10_direct_sub_residual_cavlc.h264", wantDirectSub: true, cbpTable: 0x1001},
		{name: "cabac-direct-sub", file: "high10_direct_sub_residual_cabac.h264", cabac: true, wantDirectSub: true, cbpTable: 0x1},
		{name: "cabac-b16x16-direct", file: "high10_direct_b_residual_cabac.h264", cabac: true, cbpTable: 0x1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "h264", tt.file))
			if err != nil {
				t.Fatal(err)
			}
			nals, err := SplitAnnexB(data)
			if err != nil {
				t.Fatal(err)
			}

			var spsList [maxSPSCount]*SPS
			var ppsList [maxPPSCount]*PPS
			for _, nal := range nals {
				switch nal.Type {
				case NALSPS:
					sps, err := DecodeSPS(nal.RBSP)
					if err != nil {
						t.Fatal(err)
					}
					spsList[sps.SPSID] = sps
				case NALPPS:
					pps, err := DecodePPS(nal.RBSP, &spsList)
					if err != nil {
						t.Fatal(err)
					}
					if (pps.CABAC != 0) != tt.cabac {
						t.Fatalf("PPS CABAC = %d, want %t", pps.CABAC, tt.cabac)
					}
					ppsList[pps.PPSID] = pps
				case NALSlice:
					sh, payload, err := parseSliceHeaderWithPayload(nal, &ppsList)
					if err != nil {
						t.Fatal(err)
					}
					if sh.SliceTypeNoS != PictureTypeB {
						continue
					}
					got := decodeHigh10BDeblockFixtureMacroblocksWithDirect8x8(t, sh, &payload, tt.cabac, 1, false, true)[0]
					if tt.wantDirectSub {
						if !isHighB8x8DirectSubMacroblock(got.MBType, &got.Inter.SubMBType, got.CBP) {
							t.Fatalf("B macroblock/sub types = %#x/%#x, want direct-sub B8x8", got.MBType, got.Inter.SubMBType)
						}
					} else if !isHighB16x16DirectMacroblock(got.MBType) {
						t.Fatalf("B macroblock type = %#x, want direct B16x16", got.MBType)
					}
					if got.CBP != 0x1 || got.CBPTable != tt.cbpTable {
						t.Fatalf("B direct residual CBP/CBPTable = %#x/%#x, want 0x1/%#x", got.CBP, got.CBPTable, tt.cbpTable)
					}
					return
				}
			}
			t.Fatal("B slice not found")
		})
	}
}
