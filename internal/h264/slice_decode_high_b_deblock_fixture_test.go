// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHigh10BDeblockCAVLCFixtureMacroblockSyntax(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "h264", "high10_b_deblock_cavlc.h264"))
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
			if sh.DeblockingFilter != 1 || sh.PPS == nil || sh.PPS.CABAC != 0 || isHighBImplicitWeighted(sh) {
				t.Fatalf("B slice deblock/cabac/implicit = %d/%v/%t, want CAVLC deblock-enabled neutral B",
					sh.DeblockingFilter, sh.PPS, isHighBImplicitWeighted(sh))
			}

			got := decodeHigh10BDeblockCAVLCFixtureMacroblock(t, sh, &payload)
			wantType := MBType16x16 | MBTypeP0L0 | MBTypeP0L1
			if got.MBType != wantType || got.MBType&(MBTypeDirect2|MBTypeSkip|MBType16x8|MBType8x16|MBType8x8) != 0 {
				t.Fatalf("B macroblock type = %#x, want non-direct B16x16 bidirectional", got.MBType)
			}
			if got.CBP == 0 || got.CBPTable == 0 {
				t.Fatalf("B macroblock CBP/CBPTable = %#x/%#x, want residual-bearing deblock proof", got.CBP, got.CBPTable)
			}
			return
		}
	}
	t.Fatal("B slice not found")
}

func decodeHigh10BDeblockCAVLCFixtureMacroblock(t *testing.T, sh *SliceHeader, payload *bitReader) cavlcFrameMacroblockResult {
	t.Helper()

	m, err := newMacroblockTables(1, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	var work frameMacroblockDecodeWork
	state := newCAVLCFrameSliceState(int(sh.QScale))
	got, err := m.decodeCAVLCFrameSliceMacroblockWithDirectWorkGuard(payload, sh, &state, 0, 81, h264DirectMotionContext{}, &work, true)
	if err != nil {
		t.Fatalf("decode fixture CAVLC B macroblock: %v", err)
	}
	return got
}
