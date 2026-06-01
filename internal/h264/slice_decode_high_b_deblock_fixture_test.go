// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHigh10BDeblockFixtureMacroblockSyntax(t *testing.T) {
	for _, tt := range []struct {
		name          string
		file          string
		cabac         int32
		directSpatial bool
		wantDirect    bool
	}{
		{name: "cavlc-b16x16", file: "high10_b_deblock_cavlc.h264"},
		{name: "cabac-b16x16", file: "high10_b_deblock_cabac.h264", cabac: 1},
		{name: "cavlc-temporal-direct", file: "high10_direct_b_deblock_temporal_cavlc.h264", wantDirect: true},
		{name: "cabac-temporal-direct", file: "high10_direct_b_deblock_temporal_cabac.h264", cabac: 1, wantDirect: true},
		{name: "cavlc-spatial-direct", file: "high10_direct_b_deblock_spatial_cavlc.h264", directSpatial: true, wantDirect: true},
		{name: "cabac-spatial-direct", file: "high10_direct_b_deblock_spatial_cabac.h264", cabac: 1, directSpatial: true, wantDirect: true},
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
					if sh.DeblockingFilter != 1 || sh.PPS == nil || sh.PPS.CABAC != tt.cabac ||
						isHighBImplicitWeighted(sh) || (sh.DirectSpatialMVPred != 0) != tt.directSpatial {
						t.Fatalf("B slice deblock/cabac/implicit/direct = %d/%v/%t/%d, want cabac=%d deblock-enabled neutral direct=%t",
							sh.DeblockingFilter, sh.PPS, isHighBImplicitWeighted(sh), sh.DirectSpatialMVPred, tt.cabac, tt.directSpatial)
					}

					if tt.wantDirect {
						got := decodeHigh10BDeblockFixtureMacroblocks(t, sh, &payload, tt.cabac != 0, 2, tt.directSpatial)
						var sawExplicit, sawDirect bool
						for _, mb := range got {
							if isHighB16x16ExplicitMacroblock(mb.MBType) {
								sawExplicit = true
							}
							if isHighB16x16DirectMacroblock(mb.MBType) {
								sawDirect = true
							}
							if mb.MBType&(MBTypeSkip|MBType16x8|MBType8x16|MBType8x8|MBTypeIntra4x4|MBTypeIntra16x16|MBTypeIntraPCM) != 0 {
								t.Fatalf("B macroblock type = %#x, want top-level B16x16 explicit/direct only", mb.MBType)
							}
						}
						if !sawExplicit || !sawDirect {
							t.Fatalf("B macroblock types = %#x/%#x, want one explicit B16x16 and one direct B16x16", got[0].MBType, got[1].MBType)
						}
						return
					}

					got := decodeHigh10BDeblockFixtureMacroblocks(t, sh, &payload, tt.cabac != 0, 1, false)[0]
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
		})
	}
}

func TestHigh10ImplicitPartitionedBDeblockFixtureMacroblockSyntax(t *testing.T) {
	for _, tt := range []struct {
		name       string
		file       string
		cabac      int32
		wantShape  uint32
		wantSubMB  bool
		wantCBP    int
		wantCBPSet bool
	}{
		{name: "cavlc-b16x8", file: "high10_partitioned_implicit_weight_b_deblock_b16x8_cavlc.h264", wantShape: MBType16x8},
		{name: "cabac-b16x8", file: "high10_partitioned_implicit_weight_b_deblock_b16x8_cabac.h264", cabac: 1, wantShape: MBType16x8},
		{name: "cavlc-b8x16", file: "high10_partitioned_implicit_weight_b_deblock_b8x16_cavlc.h264", wantShape: MBType8x16},
		{name: "cabac-b8x16", file: "high10_partitioned_implicit_weight_b_deblock_b8x16_cabac.h264", cabac: 1, wantShape: MBType8x16},
		{name: "cavlc-b8x8", file: "high10_partitioned_implicit_weight_b_deblock_b8x8_cavlc.h264", wantShape: MBType8x8, wantSubMB: true, wantCBP: 0x5, wantCBPSet: true},
		{name: "cabac-b8x8", file: "high10_partitioned_implicit_weight_b_deblock_b8x8_cabac.h264", cabac: 1, wantShape: MBType8x8, wantSubMB: true, wantCBP: 0x5, wantCBPSet: true},
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
					if sh.DeblockingFilter != 1 || sh.PPS == nil || sh.PPS.CABAC != tt.cabac || !isHighBImplicitWeighted(sh) {
						t.Fatalf("B slice deblock/cabac/implicit = %d/%v/%t, want cabac=%d deblock-enabled implicit weighted B",
							sh.DeblockingFilter, sh.PPS, isHighBImplicitWeighted(sh), tt.cabac)
					}

					got := decodeHigh10BDeblockFixtureMacroblocks(t, sh, &payload, tt.cabac != 0, 1, false)[0]
					if got.MBType&tt.wantShape == 0 || got.MBType&(MBTypeDirect2|MBTypeSkip|MBTypeIntra4x4|MBTypeIntra16x16|MBTypeIntraPCM) != 0 {
						t.Fatalf("B macroblock type = %#x, want implicit partitioned shape %#x only", got.MBType, tt.wantShape)
					}
					if tt.wantSubMB {
						if !isHighB8x8ExplicitSubMacroblock(got.MBType, &got.Inter.SubMBType) {
							t.Fatalf("B sub macroblock types = %#x, want explicit B8x8 sub partitions", got.Inter.SubMBType)
						}
					} else if !isHighB16x8Or8x16ExplicitMacroblock(got.MBType) {
						t.Fatalf("B macroblock type = %#x, want explicit B16x8/B8x16 partition", got.MBType)
					}
					if tt.wantCBPSet {
						if got.CBP != tt.wantCBP {
							t.Fatalf("B macroblock CBP = %#x, want %#x", got.CBP, tt.wantCBP)
						}
					} else if got.CBP != 0 || got.CBPTable != 0 {
						t.Fatalf("B macroblock CBP/CBPTable = %#x/%#x, want no residual", got.CBP, got.CBPTable)
					}
					return
				}
			}
			t.Fatal("B slice not found")
		})
	}
}

func decodeHigh10BDeblockFixtureMacroblock(t *testing.T, sh *SliceHeader, payload *bitReader, cabac bool) cavlcFrameMacroblockResult {
	t.Helper()
	return decodeHigh10BDeblockFixtureMacroblocks(t, sh, payload, cabac, 1, false)[0]
}

func decodeHigh10BDeblockFixtureMacroblocks(t *testing.T, sh *SliceHeader, payload *bitReader, cabac bool, mbWidth int, directSpatial bool) []cavlcFrameMacroblockResult {
	t.Helper()

	m, err := newMacroblockTables(mbWidth, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	direct := high10BDeblockDirectMotionContext(t, mbWidth, directSpatial)
	got := make([]cavlcFrameMacroblockResult, 0, mbWidth)
	var work frameMacroblockDecodeWork
	if cabac {
		dec, err := initCABACFrameSliceDecoder(payload, sh)
		if err != nil {
			t.Fatal(err)
		}
		state := cabacFrameSliceState{QScale: int(sh.QScale)}
		for mbXY := 0; mbXY < mbWidth; mbXY++ {
			mb, err := m.decodeCABACFrameSliceMacroblockWithDirectWorkGuard(dec.source(), sh, &state, mbXY, 81, direct, &work, true)
			if err != nil {
				t.Fatalf("decode fixture CABAC B macroblock[%d]: %v", mbXY, err)
			}
			got = append(got, cavlcFrameMacroblockResult{
				MBType:   mb.MBType,
				CBP:      mb.CBP,
				CBPTable: mb.CBPTable,
				Inter:    mb.Inter,
			})
		}
		return got
	}
	state := newCAVLCFrameSliceState(int(sh.QScale))
	for mbXY := 0; mbXY < mbWidth; mbXY++ {
		mb, err := m.decodeCAVLCFrameSliceMacroblockWithDirectWorkGuard(payload, sh, &state, mbXY, 81, direct, &work, true)
		if err != nil {
			t.Fatalf("decode fixture CAVLC B macroblock[%d]: %v", mbXY, err)
		}
		got = append(got, mb)
	}
	return got
}

func high10BDeblockDirectMotionContext(t *testing.T, mbWidth int, directSpatial bool) h264DirectMotionContext {
	t.Helper()

	past := makeH264SliceDecodePictureHigh(mbWidth, 1, 1)
	future := makeH264SliceDecodePictureHigh(mbWidth, 1, 1)
	fillH264HighResidualPlane(past.Y, 277)
	fillH264HighResidualPlane(past.Cb, 311)
	fillH264HighResidualPlane(past.Cr, 353)
	fillH264HighResidualPlane(future.Y, 277)
	fillH264HighResidualPlane(future.Cb, 311)
	fillH264HighResidualPlane(future.Cr, 353)

	colTables, err := newMacroblockTables(mbWidth, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	for mbXY := 0; mbXY < mbWidth; mbXY++ {
		colTables.MacroblockTyp[mbXY] = MBType16x16 | MBTypeP0L0 | MBTypeP1L0 | MBTypeSkip
		for i := 0; i < 4; i++ {
			colTables.RefIndex[0][4*mbXY+i] = 0
		}
	}
	pastFrame := decodedFrameFromHighPlanes(past, 0, nil)
	futureFrame := decodedFrameFromHighPlanes(future, 4, colTables)
	futureFrame.refEntries = [2][]simpleRefEntry{{{frame: pastFrame}}}

	return h264DirectMotionContext{
		RefEntries: [2][]simpleRefEntry{
			{{frame: pastFrame}},
			{{frame: futureFrame}},
		},
		CurPOC:              2,
		DirectSpatialMVPred: directSpatial,
		Direct8x8Inference:  true,
		X264Build:           165,
	}
}
