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

func TestHigh10ImplicitWeightedB16x16DeblockFixtureMacroblockSyntax(t *testing.T) {
	for _, tt := range []struct {
		name     string
		file     string
		cabac    int32
		cbpTable int
	}{
		{name: "cavlc", file: "high10_implicit_weight_b_deblock_cavlc.h264", cbpTable: 0xf00f},
		{name: "cabac", file: "high10_implicit_weight_b_deblock_cabac.h264", cabac: 1, cbpTable: 0xf},
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
			var gotB int
			for _, nal := range nals {
				switch nal.Type {
				case NALSPS:
					sps, err := DecodeSPS(nal.RBSP)
					if err != nil {
						t.Fatal(err)
					}
					if sps.Width != 32 || sps.Height != 16 || sps.ProfileIDC != 110 ||
						sps.BitDepthLuma != 10 || sps.BitDepthChroma != 10 {
						t.Fatalf("SPS profile/size/depth = %d %dx%d %d/%d, want High10 32x16 10-bit",
							sps.ProfileIDC, sps.Width, sps.Height, sps.BitDepthLuma, sps.BitDepthChroma)
					}
					spsList[sps.SPSID] = sps
				case NALPPS:
					pps, err := DecodePPS(nal.RBSP, &spsList)
					if err != nil {
						t.Fatal(err)
					}
					if pps.CABAC != tt.cabac || pps.WeightedBipredIDC != 2 ||
						pps.WeightedPred != 0 || pps.RefCount[0] != 1 || pps.RefCount[1] != 1 ||
						pps.Transform8x8Mode != 0 || pps.DeblockingFilterParametersPresent != 1 {
						t.Fatalf("PPS cabac/weights/refs/8x8/deblock = %d/%d/%d/%d/%d/%d/%d, want cabac=%d implicit-B refs=1/1 deblock params",
							pps.CABAC, pps.WeightedBipredIDC, pps.WeightedPred, pps.RefCount[0], pps.RefCount[1],
							pps.Transform8x8Mode, pps.DeblockingFilterParametersPresent, tt.cabac)
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
					gotB++
					if sh.DeblockingFilter != 1 || sh.PPS == nil || sh.PPS.CABAC != tt.cabac ||
						!isHighBImplicitWeighted(sh) || sh.DirectSpatialMVPred != 0 ||
						sh.ListCount != 2 || sh.RefCount[0] != 1 || sh.RefCount[1] != 1 ||
						sh.PredWeightTable.UseWeight != 0 || sh.PredWeightTable.UseWeightChroma != 0 {
						t.Fatalf("B slice deblock/cabac/implicit/direct/lists/refs/weights = %d/%v/%t/%d/%d/%v/%d/%d, want deblock implicit temporal refs=1/1 no serialized weights",
							sh.DeblockingFilter, sh.PPS, isHighBImplicitWeighted(sh), sh.DirectSpatialMVPred, sh.ListCount,
							sh.RefCount, sh.PredWeightTable.UseWeight, sh.PredWeightTable.UseWeightChroma)
					}
					got := decodeHigh10BDeblockFixtureMacroblocks(t, sh, &payload, tt.cabac != 0, 2, false)
					for i, mb := range got {
						wantType := MBType16x16 | MBTypeP0L0 | MBTypeP0L1
						if mb.MBType != wantType || mb.MBType&(MBTypeDirect2|MBTypeSkip|MBType16x8|MBType8x16|MBType8x8) != 0 {
							t.Fatalf("B macroblock[%d] type = %#x, want implicit weighted B16x16 bidirectional", i, mb.MBType)
						}
						if mb.CBP != 0xf || mb.CBPTable != tt.cbpTable {
							t.Fatalf("B macroblock[%d] CBP/CBPTable = %#x/%#x, want 0xf/%#x", i, mb.CBP, mb.CBPTable, tt.cbpTable)
						}
					}
				}
			}
			if gotB != 2 {
				t.Fatalf("B slices = %d, want 2", gotB)
			}
		})
	}
}

func TestHigh10PartitionedBDeblockFixtureMacroblockSyntax(t *testing.T) {
	for _, tt := range []struct {
		name              string
		file              string
		cabac             int32
		weightedBipredIDC uint32
		wantShape         uint32
		wantSubMB         bool
		wantCBP           int
		wantCBPSet        bool
	}{
		{name: "neutral-cavlc-b16x8", file: "high10_partitioned_b_deblock_b16x8_cavlc.h264", wantShape: MBType16x8},
		{name: "neutral-cabac-b16x8", file: "high10_partitioned_b_deblock_b16x8_cabac.h264", cabac: 1, wantShape: MBType16x8},
		{name: "neutral-cavlc-b8x16", file: "high10_partitioned_b_deblock_b8x16_cavlc.h264", wantShape: MBType8x16},
		{name: "neutral-cabac-b8x16", file: "high10_partitioned_b_deblock_b8x16_cabac.h264", cabac: 1, wantShape: MBType8x16},
		{name: "neutral-cavlc-b8x8", file: "high10_partitioned_b_deblock_b8x8_cavlc.h264", wantShape: MBType8x8, wantSubMB: true, wantCBP: 0x7, wantCBPSet: true},
		{name: "neutral-cabac-b8x8", file: "high10_partitioned_b_deblock_b8x8_cabac.h264", cabac: 1, wantShape: MBType8x8, wantSubMB: true, wantCBP: 0x7, wantCBPSet: true},
		{name: "implicit-cavlc-b16x8", file: "high10_partitioned_implicit_weight_b_deblock_b16x8_cavlc.h264", weightedBipredIDC: 2, wantShape: MBType16x8},
		{name: "implicit-cabac-b16x8", file: "high10_partitioned_implicit_weight_b_deblock_b16x8_cabac.h264", cabac: 1, weightedBipredIDC: 2, wantShape: MBType16x8},
		{name: "implicit-cavlc-b8x16", file: "high10_partitioned_implicit_weight_b_deblock_b8x16_cavlc.h264", weightedBipredIDC: 2, wantShape: MBType8x16},
		{name: "implicit-cabac-b8x16", file: "high10_partitioned_implicit_weight_b_deblock_b8x16_cabac.h264", cabac: 1, weightedBipredIDC: 2, wantShape: MBType8x16},
		{name: "implicit-cavlc-b8x8", file: "high10_partitioned_implicit_weight_b_deblock_b8x8_cavlc.h264", weightedBipredIDC: 2, wantShape: MBType8x8, wantSubMB: true, wantCBP: 0x5, wantCBPSet: true},
		{name: "implicit-cabac-b8x8", file: "high10_partitioned_implicit_weight_b_deblock_b8x8_cabac.h264", cabac: 1, weightedBipredIDC: 2, wantShape: MBType8x8, wantSubMB: true, wantCBP: 0x5, wantCBPSet: true},
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
						sh.PPS.WeightedBipredIDC != tt.weightedBipredIDC {
						t.Fatalf("B slice deblock/cabac/weighted = %d/%v/%d, want cabac=%d weighted_bipred_idc=%d",
							sh.DeblockingFilter, sh.PPS, sh.PPS.WeightedBipredIDC, tt.cabac, tt.weightedBipredIDC)
					}

					got := decodeHigh10BDeblockFixtureMacroblocks(t, sh, &payload, tt.cabac != 0, 1, false)[0]
					if got.MBType&tt.wantShape == 0 || got.MBType&(MBTypeDirect2|MBTypeSkip|MBTypeIntra4x4|MBTypeIntra16x16|MBTypeIntraPCM) != 0 {
						t.Fatalf("B macroblock type = %#x, want partitioned shape %#x only", got.MBType, tt.wantShape)
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

func TestHigh10BSkipAndDirectSubDeblockFixtureMacroblockSyntax(t *testing.T) {
	for _, tt := range []struct {
		name              string
		file              string
		cabac             int32
		weightedBipredIDC uint32
		directSpatial     bool
		direct8x8         bool
		wantDirectSub     bool
	}{
		{name: "bskip temporal cavlc", file: "high10_bskip_deblock_temporal_cavlc.h264", direct8x8: true},
		{name: "bskip temporal cabac", file: "high10_bskip_deblock_temporal_cabac.h264", cabac: 1, direct8x8: true},
		{name: "bskip spatial cavlc", file: "high10_bskip_deblock_spatial_cavlc.h264", directSpatial: true, direct8x8: true},
		{name: "bskip spatial cabac", file: "high10_bskip_deblock_spatial_cabac.h264", cabac: 1, directSpatial: true, direct8x8: true},
		{name: "direct-sub b8x8 temporal cavlc", file: "high10_cavlc_b8x8_temporal_direct_sub_deblock.h264", direct8x8: true, wantDirectSub: true},
		{name: "direct-sub b8x8 temporal cabac", file: "high10_cabac_b8x8_temporal_direct_sub_deblock.h264", cabac: 1, direct8x8: true, wantDirectSub: true},
		{name: "direct-sub b8x8 spatial cavlc", file: "high10_cavlc_b8x8_spatial_direct_sub_deblock.h264", directSpatial: true, direct8x8: true, wantDirectSub: true},
		{name: "direct-sub b8x8 spatial cabac", file: "high10_cabac_b8x8_spatial_direct_sub_deblock.h264", cabac: 1, directSpatial: true, direct8x8: true, wantDirectSub: true},
		{name: "direct-sub b4x4 temporal cavlc", file: "high10_cavlc_b4x4_temporal_direct_sub_deblock.h264", wantDirectSub: true},
		{name: "direct-sub b4x4 temporal cabac", file: "high10_cabac_b4x4_temporal_direct_sub_deblock.h264", cabac: 1, wantDirectSub: true},
		{name: "direct-sub b4x4 spatial cavlc", file: "high10_cavlc_b4x4_spatial_direct_sub_deblock.h264", directSpatial: true, wantDirectSub: true},
		{name: "direct-sub b4x4 spatial cabac", file: "high10_cabac_b4x4_spatial_direct_sub_deblock.h264", cabac: 1, directSpatial: true, wantDirectSub: true},
		{name: "implicit direct-sub b8x8 temporal cavlc", file: "high10_implicit_weight_cavlc_b8x8_temporal_direct_sub_deblock.h264", weightedBipredIDC: 2, direct8x8: true, wantDirectSub: true},
		{name: "implicit direct-sub b8x8 temporal cabac", file: "high10_implicit_weight_cabac_b8x8_temporal_direct_sub_deblock.h264", cabac: 1, weightedBipredIDC: 2, direct8x8: true, wantDirectSub: true},
		{name: "implicit direct-sub b8x8 spatial cavlc", file: "high10_implicit_weight_cavlc_b8x8_spatial_direct_sub_deblock.h264", weightedBipredIDC: 2, directSpatial: true, direct8x8: true, wantDirectSub: true},
		{name: "implicit direct-sub b8x8 spatial cabac", file: "high10_implicit_weight_cabac_b8x8_spatial_direct_sub_deblock.h264", cabac: 1, weightedBipredIDC: 2, directSpatial: true, direct8x8: true, wantDirectSub: true},
		{name: "implicit direct-sub b4x4 temporal cavlc", file: "high10_implicit_weight_cavlc_b4x4_temporal_direct_sub_deblock.h264", weightedBipredIDC: 2, wantDirectSub: true},
		{name: "implicit direct-sub b4x4 temporal cabac", file: "high10_implicit_weight_cabac_b4x4_temporal_direct_sub_deblock.h264", cabac: 1, weightedBipredIDC: 2, wantDirectSub: true},
		{name: "implicit direct-sub b4x4 spatial cavlc", file: "high10_implicit_weight_cavlc_b4x4_spatial_direct_sub_deblock.h264", weightedBipredIDC: 2, directSpatial: true, wantDirectSub: true},
		{name: "implicit direct-sub b4x4 spatial cabac", file: "high10_implicit_weight_cabac_b4x4_spatial_direct_sub_deblock.h264", cabac: 1, weightedBipredIDC: 2, directSpatial: true, wantDirectSub: true},
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
					if sps.Direct8x8InferenceFlag != boolToInt32(tt.direct8x8) {
						t.Fatalf("direct_8x8_inference_flag = %d, want %t", sps.Direct8x8InferenceFlag, tt.direct8x8)
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
					wantImplicit := tt.weightedBipredIDC == 2
					if sh.DeblockingFilter != 1 || sh.PPS == nil || sh.PPS.CABAC != tt.cabac ||
						sh.PPS.WeightedBipredIDC != tt.weightedBipredIDC ||
						isHighBImplicitWeighted(sh) != wantImplicit || (sh.DirectSpatialMVPred != 0) != tt.directSpatial ||
						sh.PredWeightTable.UseWeight != 0 || sh.PredWeightTable.UseWeightChroma != 0 {
						t.Fatalf("B slice deblock/cabac/weighted/implicit/direct/weights = %d/%v/%d/%t/%d/%d/%d, want cabac=%d weighted_bipred_idc=%d implicit=%t direct=%t no serialized weights",
							sh.DeblockingFilter, sh.PPS, sh.PPS.WeightedBipredIDC, isHighBImplicitWeighted(sh), sh.DirectSpatialMVPred,
							sh.PredWeightTable.UseWeight, sh.PredWeightTable.UseWeightChroma,
							tt.cabac, tt.weightedBipredIDC, wantImplicit, tt.directSpatial)
					}

					got := decodeHigh10BDeblockFixtureMacroblocksWithDirect8x8(t, sh, &payload, tt.cabac != 0, 1, tt.directSpatial, tt.direct8x8)[0]
					if got.CBP != 0 || got.CBPTable != 0 {
						t.Fatalf("B macroblock CBP/CBPTable = %#x/%#x, want no residual", got.CBP, got.CBPTable)
					}
					if tt.wantDirectSub {
						if got.MBType&(MBTypeSkip|MBType16x8|MBType8x16|MBTypeIntra4x4|MBTypeIntra16x16|MBTypeIntraPCM) != 0 ||
							!isHighB8x8DirectSubMacroblock(got.MBType, &got.Inter.SubMBType) {
							t.Fatalf("B macroblock/sub types = %#x/%#x, want direct-sub B8x8/B_SUB_4x4", got.MBType, got.Inter.SubMBType)
						}
					} else if !isHighB16x16DirectSkipMacroblock(got.MBType) {
						t.Fatalf("B macroblock type = %#x, want direct skip", got.MBType)
					}
					return
				}
			}
			t.Fatal("B slice not found")
		})
	}
}

func TestHigh10BResidualDeblockFixtureFiltersInternalHighEdges(t *testing.T) {
	for _, tt := range []struct {
		name  string
		file  string
		cabac bool
	}{
		{name: "cavlc", file: "high10_b_deblock_cavlc.h264"},
		{name: "cabac", file: "high10_b_deblock_cabac.h264", cabac: true},
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
					if sh.DeblockingFilter != 1 || sh.SPS.BitDepthLuma != 10 || sh.SPS.ChromaFormatIDC != 1 {
						t.Fatalf("B slice deblock/depth/chroma = %d/%d/%d, want High10 4:2:0 deblock enabled",
							sh.DeblockingFilter, sh.SPS.BitDepthLuma, sh.SPS.ChromaFormatIDC)
					}

					m, got := decodeHigh10BDeblockFixtureTablesWithDirect8x8(t, sh, &payload, tt.cabac, 1, false, true)
					mb := got[0]
					wantType := MBType16x16 | MBTypeP0L0 | MBTypeP0L1
					if mb.MBType != wantType || mb.CBP == 0 || mb.CBPTable == 0 || m.CBPTable[0] == 0 || !hasHigh10FixtureNonZeroCount(m.NonZeroCount[0]) {
						t.Fatalf("B fixture table type/cbp/cbpTable/table/nnz = %#x/%#x/%#x/%#x/%v, want residual B16x16 writeback",
							mb.MBType, mb.CBP, mb.CBPTable, m.CBPTable[0], m.NonZeroCount[0])
					}

					dst := makeH264SliceDecodePictureHigh(1, 1, 1)
					fillHighLoopFilterStep(dst.Y, dst.LumaStride, 16, 16, 8, 400, 404)
					fillHighLoopFilterStep(dst.Cb, dst.ChromaStride, 8, 8, 4, 300, 304)
					fillHighLoopFilterStep(dst.Cr, dst.ChromaStride, 8, 8, 4, 200, 204)
					yBefore := [2]uint16{dst.Y[7], dst.Y[8]}
					cbBefore := [2]uint16{dst.Cb[3], dst.Cb[4]}
					crBefore := [2]uint16{dst.Cr[3], dst.Cr[4]}
					params := make([]h264LoopFilterSliceParams, int(m.SliceTable[0])+1)
					params[m.SliceTable[0]] = h264LoopFilterSliceParamsFromHeader(sh)

					if err := m.filterFrameHigh(dst, params); err != nil {
						t.Fatalf("filter high B deblock fixture: %v", err)
					}
					if dst.Y[7] == yBefore[0] || dst.Y[8] == yBefore[1] {
						t.Fatalf("High10 B residual deblock fixture left luma internal edge untouched: %v -> [%d %d]",
							yBefore, dst.Y[7], dst.Y[8])
					}
					if dst.Cb[3] == cbBefore[0] || dst.Cb[4] == cbBefore[1] ||
						dst.Cr[3] == crBefore[0] || dst.Cr[4] == crBefore[1] {
						t.Fatalf("High10 B residual deblock fixture left chroma internal edge untouched: cb %v -> [%d %d] cr %v -> [%d %d]",
							cbBefore, dst.Cb[3], dst.Cb[4], crBefore, dst.Cr[3], dst.Cr[4])
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
	return decodeHigh10BDeblockFixtureMacroblocksWithDirect8x8(t, sh, payload, cabac, mbWidth, directSpatial, true)
}

func decodeHigh10BDeblockFixtureMacroblocksWithDirect8x8(t *testing.T, sh *SliceHeader, payload *bitReader, cabac bool, mbWidth int, directSpatial bool, direct8x8 bool) []cavlcFrameMacroblockResult {
	t.Helper()
	_, got := decodeHigh10BDeblockFixtureTablesWithDirect8x8(t, sh, payload, cabac, mbWidth, directSpatial, direct8x8)
	return got
}

func decodeHigh10BDeblockFixtureTablesWithDirect8x8(t *testing.T, sh *SliceHeader, payload *bitReader, cabac bool, mbWidth int, directSpatial bool, direct8x8 bool) (*macroblockTables, []cavlcFrameMacroblockResult) {
	t.Helper()

	m, err := newMacroblockTables(mbWidth, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	direct := high10BDeblockDirectMotionContext(t, mbWidth, directSpatial, direct8x8)
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
		return m, got
	}
	state := newCAVLCFrameSliceState(int(sh.QScale))
	for mbXY := 0; mbXY < mbWidth; mbXY++ {
		mb, err := m.decodeCAVLCFrameSliceMacroblockWithDirectWorkGuard(payload, sh, &state, mbXY, 81, direct, &work, true)
		if err != nil {
			t.Fatalf("decode fixture CAVLC B macroblock[%d]: %v", mbXY, err)
		}
		got = append(got, mb)
	}
	return m, got
}

func high10BDeblockDirectMotionContext(t *testing.T, mbWidth int, directSpatial bool, direct8x8 bool) h264DirectMotionContext {
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
		Direct8x8Inference:  direct8x8,
		X264Build:           165,
	}
}

func hasHigh10FixtureNonZeroCount(nnz [h264MBNonZeroCountSize]uint8) bool {
	for _, v := range nnz {
		if v != 0 {
			return true
		}
	}
	return false
}

func boolToInt32(v bool) int32 {
	if v {
		return 1
	}
	return 0
}
