// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHigh10PartitionedBFixtureMacroblockSyntax(t *testing.T) {
	for _, tt := range []struct {
		name              string
		file              string
		cabac             bool
		weightedBipredIDC uint32
		wantMask          uint32
	}{
		{name: "cavlc-b16x8", file: "high10_partitioned_b16x8_cavlc.h264", wantMask: MBType16x8},
		{name: "cabac-b16x8", file: "high10_partitioned_b16x8_cabac.h264", cabac: true, wantMask: MBType16x8},
		{name: "cavlc-b8x16", file: "high10_partitioned_b8x16_cavlc.h264", wantMask: MBType8x16},
		{name: "cabac-b8x16", file: "high10_partitioned_b8x16_cabac.h264", cabac: true, wantMask: MBType8x16},
		{name: "cavlc-b8x8", file: "high10_partitioned_b8x8_cavlc.h264", wantMask: MBType8x8},
		{name: "cabac-b8x8", file: "high10_partitioned_b8x8_cabac.h264", cabac: true, wantMask: MBType8x8},
		{name: "implicit-cavlc-b16x8", file: "high10_partitioned_implicit_weight_b16x8_cavlc.h264", weightedBipredIDC: 2, wantMask: MBType16x8},
		{name: "implicit-cabac-b16x8", file: "high10_partitioned_implicit_weight_b16x8_cabac.h264", cabac: true, weightedBipredIDC: 2, wantMask: MBType16x8},
		{name: "implicit-cavlc-b8x16", file: "high10_partitioned_implicit_weight_b8x16_cavlc.h264", weightedBipredIDC: 2, wantMask: MBType8x16},
		{name: "implicit-cabac-b8x16", file: "high10_partitioned_implicit_weight_b8x16_cabac.h264", cabac: true, weightedBipredIDC: 2, wantMask: MBType8x16},
		{name: "implicit-cavlc-b8x8", file: "high10_partitioned_implicit_weight_b8x8_cavlc.h264", weightedBipredIDC: 2, wantMask: MBType8x8},
		{name: "implicit-cabac-b8x8", file: "high10_partitioned_implicit_weight_b8x8_cabac.h264", cabac: true, weightedBipredIDC: 2, wantMask: MBType8x8},
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
					ppsList[pps.PPSID] = pps
				case NALSlice:
					sh, payload, err := parseSliceHeaderWithPayload(nal, &ppsList)
					if err != nil {
						t.Fatal(err)
					}
					if sh.SliceTypeNoS != PictureTypeB {
						continue
					}
					if sh.PPS == nil || sh.PPS.WeightedBipredIDC != tt.weightedBipredIDC {
						got := uint32(0)
						if sh.PPS != nil {
							got = sh.PPS.WeightedBipredIDC
						}
						t.Fatalf("weighted_bipred_idc = %d, want %d", got, tt.weightedBipredIDC)
					}
					got := decodeHigh10PartitionedBFixtureMacroblock(t, sh, &payload, tt.cabac)
					if got.MBType&tt.wantMask == 0 || got.MBType&(MBTypeDirect2|MBTypeSkip) != 0 {
						t.Fatalf("B macroblock type = %#x, want explicit partition mask %#x", got.MBType, tt.wantMask)
					}
					if tt.wantMask == MBType8x8 {
						for i := 0; i < 4; i++ {
							if !isHighBExplicitSubMBType(got.Inter.SubMBType[i]) {
								t.Fatalf("sub[%d] type = %#x, want explicit B sub-MB", i, got.Inter.SubMBType[i])
							}
						}
					}
					return
				}
			}
			t.Fatal("B slice not found")
		})
	}
}

func decodeHigh10PartitionedBFixtureMacroblock(t *testing.T, sh *SliceHeader, payload *bitReader, cabac bool) cavlcFrameMacroblockResult {
	t.Helper()

	m, err := newMacroblockTables(1, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	var work frameMacroblockDecodeWork
	if cabac {
		dec, err := initCABACFrameSliceDecoder(payload, sh)
		if err != nil {
			t.Fatal(err)
		}
		state := cabacFrameSliceState{QScale: int(sh.QScale)}
		got, err := m.decodeCABACFrameSliceMacroblockWithDirectWorkGuard(dec.source(), sh, &state, 0, 71, h264DirectMotionContext{}, &work, true)
		if err != nil {
			t.Fatalf("decode fixture CABAC B macroblock: %v", err)
		}
		return cavlcFrameMacroblockResult{
			MBType: got.MBType,
			Inter:  got.Inter,
		}
	}
	state := newCAVLCFrameSliceState(int(sh.QScale))
	got, err := m.decodeCAVLCFrameSliceMacroblockWithDirectWorkGuard(payload, sh, &state, 0, 71, h264DirectMotionContext{}, &work, true)
	if err != nil {
		t.Fatalf("decode fixture CAVLC B macroblock: %v", err)
	}
	return got
}
