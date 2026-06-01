// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHigh10DirectSubResidualCAVLCFixtureMacroblockSyntax(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "h264", "high10_direct_sub_residual_cavlc.h264"))
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
			ppsList[pps.PPSID] = pps
		case NALSlice:
			sh, payload, err := parseSliceHeaderWithPayload(nal, &ppsList)
			if err != nil {
				t.Fatal(err)
			}
			if sh.SliceTypeNoS != PictureTypeB {
				continue
			}
			got := decodeHigh10BDeblockFixtureMacroblocksWithDirect8x8(t, sh, &payload, false, 1, false, true)[0]
			if !isHighB8x8DirectSubMacroblock(got.MBType, &got.Inter.SubMBType) {
				t.Fatalf("B macroblock/sub types = %#x/%#x, want direct-sub B8x8", got.MBType, got.Inter.SubMBType)
			}
			if got.CBP != 0x1 || got.CBPTable != 0x1001 {
				t.Fatalf("B direct-sub residual CBP/CBPTable = %#x/%#x, want 0x1/0x1001", got.CBP, got.CBPTable)
			}
			return
		}
	}
	t.Fatal("B slice not found")
}
