// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

import (
	"fmt"
	"testing"
)

func TestValidateHighFrameSliceMacroblockForReconstructAllowsP16x16Residual(t *testing.T) {
	sh := &SliceHeader{SliceTypeNoS: PictureTypeP}

	for _, tt := range []struct {
		name     string
		cbp      int
		cbpTable int
	}{
		{name: "no residual"},
		{name: "luma residual", cbp: 0x01, cbpTable: 0x1001},
		{name: "luma chroma residual", cbp: 0x31, cbpTable: 0xf031},
	} {
		t.Run(tt.name, func(t *testing.T) {
			mbType := MBType16x16 | MBTypeP0L0
			if err := validateHighFrameSliceMacroblockForReconstruct(sh, mbType, tt.cbp, tt.cbpTable); err != nil {
				t.Fatalf("validate P16x16 residual err = %v, want nil", err)
			}
		})
	}
}

func TestValidateHighFrameSliceMacroblockForReconstructAllowsProvedPIntra(t *testing.T) {
	sh := &SliceHeader{SliceTypeNoS: PictureTypeP}

	for _, mbType := range []uint32{
		MBTypeIntra4x4,
		MBTypeIntra4x4 | MBType8x8DCT,
		MBTypeIntra16x16,
	} {
		if err := validateHighFrameSliceMacroblockForReconstruct(sh, mbType, 1, 1); err != nil {
			t.Fatalf("validate high P intra %#x err = %v, want nil", mbType, err)
		}
	}
}

func TestValidateHighFrameSliceMacroblockForReconstructAllowsHigh1214Intra4x4CBP3(t *testing.T) {
	for _, bitDepth := range []int32{12, 14} {
		t.Run(fmt.Sprintf("high%d", bitDepth), func(t *testing.T) {
			sh := &SliceHeader{
				SliceTypeNoS: PictureTypeI,
				SPS:          &SPS{BitDepthLuma: bitDepth},
			}
			for _, cbpTable := range []int{0x03, 0x7003} {
				if err := validateHighFrameSliceMacroblockForReconstruct(sh, MBTypeIntra4x4, 0x03, cbpTable); err != nil {
					t.Fatalf("validate High%d Intra4x4 cbp=0x03 table=%#x err = %v, want nil", bitDepth, cbpTable, err)
				}
			}
		})
	}
}

func TestHigh1214Frame420ScopeAllowsImplicitWeightedBParseAndDecodeWeights(t *testing.T) {
	for _, tt := range []struct {
		name      string
		useWeight int32
	}{
		{name: "parse", useWeight: 0},
		{name: "decode", useWeight: 2},
	} {
		t.Run(tt.name, func(t *testing.T) {
			for _, bitDepth := range []int32{12, 14} {
				for _, cabac := range []int32{0, 1} {
					for _, deblock := range []int32{0, 1, 2} {
						sh := &SliceHeader{
							SliceTypeNoS:     PictureTypeB,
							DeblockingFilter: deblock,
							SPS: &SPS{
								BitDepthLuma:    bitDepth,
								BitDepthChroma:  bitDepth,
								ChromaFormatIDC: 1,
							},
							PPS: &PPS{
								CABAC:             cabac,
								WeightedBipredIDC: 2,
							},
							PredWeightTable: PredWeightTable{
								UseWeight:       tt.useWeight,
								UseWeightChroma: tt.useWeight,
							},
						}
						if !isPublicHighFrameBitDepthScope(sh) {
							t.Fatalf("High%d cabac=%d deblock=%d useWeight=%d not admitted", bitDepth, cabac, deblock, tt.useWeight)
						}
						if err := validateHighFrameSliceDeblockingScope(sh); err != nil {
							t.Fatalf("High%d cabac=%d deblock=%d useWeight=%d deblock scope err = %v", bitDepth, cabac, deblock, tt.useWeight, err)
						}
					}
				}
			}
		})
	}
}

func TestValidateHighFrameSliceMacroblockForReconstructAllowsProvedBIntra(t *testing.T) {
	sh := &SliceHeader{SliceTypeNoS: PictureTypeB}

	for _, mbType := range []uint32{
		MBTypeIntra4x4,
		MBTypeIntra4x4 | MBType8x8DCT,
		MBTypeIntra16x16,
		MBTypeIntra4x4 | MBTypeInterlaced,
	} {
		if err := validateHighFrameSliceBaseMacroblockForDecode(PictureTypeB, mbType&^MBType8x8DCT); err != nil {
			t.Fatalf("validate high B intra base %#x err = %v, want nil", mbType, err)
		}
		if err := validateHighFrameSliceMacroblockForReconstruct(sh, mbType, 1, 1); err != nil {
			t.Fatalf("validate high B intra %#x err = %v, want nil", mbType, err)
		}
	}
}

func TestValidateHighFrameSliceMacroblockForReconstructAllowsFieldBShapes(t *testing.T) {
	sh := &SliceHeader{
		SliceTypeNoS:     PictureTypeB,
		PictureStructure: PictureTopField,
		SPS: &SPS{
			BitDepthLuma:     10,
			BitDepthChroma:   10,
			ChromaFormatIDC:  2,
			FrameMBSOnlyFlag: 0,
			MBAFF:            1,
		},
		PPS: &PPS{WeightedBipredIDC: 2},
	}
	for _, tt := range []struct {
		name     string
		baseType uint32
		mbType   uint32
		cbp      int
		cbpTable int
	}{
		{name: "b16x16 l1", mbType: MBType16x16 | MBTypeP0L1 | MBTypeInterlaced},
		{name: "b16x16 bidirectional", mbType: MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeInterlaced},
		{name: "b16x16 temporal direct", baseType: MBTypeL0L1 | MBTypeDirect2 | MBTypeInterlaced, mbType: MBType16x16 | MBTypeL0L1 | MBTypeDirect2 | MBTypeInterlaced},
		{name: "b16x16 direct skip", baseType: MBTypeL0L1 | MBTypeDirect2 | MBTypeInterlaced, mbType: MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2 | MBTypeSkip | MBTypeInterlaced},
		{name: "b16x8 l0 l1 residual", mbType: MBType16x8 | MBTypeP0L0 | MBTypeP1L1 | MBTypeInterlaced, cbp: 1, cbpTable: 1},
		{name: "b8x16 l1 l1", mbType: MBType8x16 | MBTypeP0L1 | MBTypeP1L1 | MBTypeInterlaced},
	} {
		t.Run(tt.name, func(t *testing.T) {
			baseType := tt.baseType
			if baseType == 0 {
				baseType = tt.mbType &^ MBTypeSkip
			}
			if err := validateHighFrameSliceBaseMacroblockForDecode(PictureTypeB, baseType); err != nil {
				t.Fatalf("validate high field B base err = %v, want nil", err)
			}
			if err := validateHighFrameSliceMacroblockForReconstructWithSubMB(sh, tt.mbType, nil, tt.cbp, tt.cbpTable); err != nil {
				t.Fatalf("validate high field B reconstruct err = %v, want nil", err)
			}
		})
	}
}

func TestValidateHighFrameSliceMacroblockForReconstructAllowsHigh10Chroma422FieldWeightedB(t *testing.T) {
	weights := []struct {
		name             string
		weightedBipredID uint32
		useWeight        int32
		useWeightChroma  int32
	}{
		{name: "explicit-luma", weightedBipredID: 1, useWeight: 1},
		{name: "explicit-chroma", weightedBipredID: 1, useWeightChroma: 1},
		{name: "explicit-luma-chroma", weightedBipredID: 1, useWeight: 1, useWeightChroma: 1},
		{name: "implicit", weightedBipredID: 2, useWeight: 2, useWeightChroma: 2},
	}
	shapes := []struct {
		name     string
		mbType   uint32
		sub      *[4]uint32
		cbp      int
		cbpTable int
	}{
		{name: "b16x16-l0", mbType: MBType16x16 | MBTypeP0L0 | MBTypeInterlaced},
		{name: "b16x16-l1", mbType: MBType16x16 | MBTypeP0L1 | MBTypeInterlaced},
		{name: "b16x16-bi", mbType: MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeInterlaced},
		{name: "b16x8-bi-residual", mbType: MBType16x8 | MBTypeP0L0 | MBTypeP0L1 | MBTypeP1L0 | MBTypeP1L1 | MBTypeInterlaced, cbp: 1, cbpTable: 1},
		{name: "b8x16-l0-l1", mbType: MBType8x16 | MBTypeP0L0 | MBTypeP1L1 | MBTypeInterlaced},
		{name: "b8x8-explicit-sub", mbType: MBType8x8 | MBTypeP0L0 | MBTypeP0L1 | MBTypeP1L0 | MBTypeP1L1 | MBTypeInterlaced, sub: &([4]uint32{
			MBType16x16 | MBTypeP0L0,
			MBType16x16 | MBTypeP0L1,
			MBType16x16 | MBTypeP0L0 | MBTypeP0L1,
			MBType16x16 | MBTypeP0L1,
		})},
	}
	for _, picture := range []int32{PictureTopField, PictureBottomField} {
		for _, deblock := range []int32{0, 1, 2} {
			for _, weight := range weights {
				for _, shape := range shapes {
					t.Run(fmt.Sprintf("picture%d/mode%d/%s/%s", picture, deblock, weight.name, shape.name), func(t *testing.T) {
						sh := &SliceHeader{
							SliceTypeNoS:     PictureTypeB,
							PictureStructure: picture,
							DeblockingFilter: deblock,
							SPS: &SPS{
								BitDepthLuma:     10,
								BitDepthChroma:   10,
								ChromaFormatIDC:  2,
								FrameMBSOnlyFlag: 0,
								MBAFF:            1,
							},
							PPS: &PPS{WeightedBipredIDC: weight.weightedBipredID},
							PredWeightTable: PredWeightTable{
								UseWeight:       weight.useWeight,
								UseWeightChroma: weight.useWeightChroma,
							},
						}
						if err := validateHighFrameSliceMacroblockForReconstructWithSubMB(sh, shape.mbType, shape.sub, shape.cbp, shape.cbpTable); err != nil {
							t.Fatalf("validate high10 422 field weighted B reconstruct err = %v, want nil", err)
						}
					})
				}
			}
		}
	}
}

func TestValidateHighFrameSliceMacroblockForReconstructAllowsHigh10Chroma444FieldWeightedB(t *testing.T) {
	weights := []struct {
		name             string
		weightedBipredID uint32
		useWeight        int32
		useWeightChroma  int32
	}{
		{name: "explicit-luma", weightedBipredID: 1, useWeight: 1},
		{name: "explicit-chroma", weightedBipredID: 1, useWeightChroma: 1},
		{name: "explicit-luma-chroma", weightedBipredID: 1, useWeight: 1, useWeightChroma: 1},
		{name: "implicit", weightedBipredID: 2, useWeight: 2, useWeightChroma: 2},
	}
	shapes := []struct {
		name     string
		mbType   uint32
		sub      *[4]uint32
		cbp      int
		cbpTable int
	}{
		{name: "b16x16-l0", mbType: MBType16x16 | MBTypeP0L0 | MBTypeInterlaced},
		{name: "b16x16-l1", mbType: MBType16x16 | MBTypeP0L1 | MBTypeInterlaced},
		{name: "b16x16-bi", mbType: MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeInterlaced},
		{name: "b16x8-bi-residual", mbType: MBType16x8 | MBTypeP0L0 | MBTypeP0L1 | MBTypeP1L0 | MBTypeP1L1 | MBTypeInterlaced, cbp: 1, cbpTable: 1},
		{name: "b8x16-l0-l1", mbType: MBType8x16 | MBTypeP0L0 | MBTypeP1L1 | MBTypeInterlaced},
		{name: "b8x8-explicit-sub", mbType: MBType8x8 | MBTypeP0L0 | MBTypeP0L1 | MBTypeP1L0 | MBTypeP1L1 | MBTypeInterlaced, sub: &([4]uint32{
			MBType16x16 | MBTypeP0L0,
			MBType16x16 | MBTypeP0L1,
			MBType16x16 | MBTypeP0L0 | MBTypeP0L1,
			MBType16x16 | MBTypeP0L1,
		})},
	}
	for _, picture := range []int32{PictureTopField, PictureBottomField} {
		for _, deblock := range []int32{0, 1, 2} {
			for _, weight := range weights {
				for _, shape := range shapes {
					t.Run(fmt.Sprintf("picture%d/mode%d/%s/%s", picture, deblock, weight.name, shape.name), func(t *testing.T) {
						sh := &SliceHeader{
							SliceTypeNoS:     PictureTypeB,
							PictureStructure: picture,
							DeblockingFilter: deblock,
							SPS: &SPS{
								BitDepthLuma:     10,
								BitDepthChroma:   10,
								ChromaFormatIDC:  3,
								FrameMBSOnlyFlag: 0,
								MBAFF:            1,
							},
							PPS: &PPS{WeightedBipredIDC: weight.weightedBipredID},
							PredWeightTable: PredWeightTable{
								UseWeight:       weight.useWeight,
								UseWeightChroma: weight.useWeightChroma,
							},
						}
						if err := validateHighFrameSliceMacroblockForReconstructWithSubMB(sh, shape.mbType, shape.sub, shape.cbp, shape.cbpTable); err != nil {
							t.Fatalf("validate high10 444 field weighted B reconstruct err = %v, want nil", err)
						}
					})
				}
			}
		}
	}
}

func TestValidateHighFrameSliceMacroblockForReconstructAllowsHigh10Chroma444UnweightedFieldIB(t *testing.T) {
	bSub := [4]uint32{
		MBType16x16 | MBTypeP0L0,
		MBType16x16 | MBTypeP0L1,
		MBType16x16 | MBTypeP0L0 | MBTypeP0L1,
		MBType16x16 | MBTypeP0L1,
	}
	for _, picture := range []int32{PictureTopField, PictureBottomField} {
		for _, cabac := range []int32{0, 1} {
			for _, shape := range []struct {
				name     string
				slice    int32
				deblock  []int32
				mbType   uint32
				sub      *[4]uint32
				cbp      int
				cbpTable int
			}{
				{name: "I/intra4x4", slice: PictureTypeI, deblock: []int32{0, 1, 2}, mbType: MBTypeIntra4x4},
				{name: "I/intra16x16", slice: PictureTypeI, deblock: []int32{0, 1, 2}, mbType: MBTypeIntra16x16},
				{name: "I/intrapcm", slice: PictureTypeI, deblock: []int32{0, 1, 2}, mbType: MBTypeIntraPCM},
				{name: "B/b16x16-l0", slice: PictureTypeB, deblock: []int32{0, 1}, mbType: MBType16x16 | MBTypeP0L0 | MBTypeInterlaced},
				{name: "B/b16x16-bi", slice: PictureTypeB, deblock: []int32{0, 1}, mbType: MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeInterlaced},
				{name: "B/b16x16-direct", slice: PictureTypeB, deblock: []int32{0, 1}, mbType: MBType16x16 | MBTypeL0L1 | MBTypeDirect2 | MBTypeInterlaced},
				{name: "B/direct-skip", slice: PictureTypeB, deblock: []int32{0, 1}, mbType: MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2 | MBTypeSkip | MBTypeInterlaced},
				{name: "B/b16x8-residual", slice: PictureTypeB, deblock: []int32{0, 1}, mbType: MBType16x8 | MBTypeP0L0 | MBTypeP1L1 | MBTypeInterlaced, cbp: 1, cbpTable: 1},
				{name: "B/b8x16", slice: PictureTypeB, deblock: []int32{0, 1}, mbType: MBType8x16 | MBTypeP0L0 | MBTypeP1L1 | MBTypeInterlaced},
				{name: "B/b8x8-sub", slice: PictureTypeB, deblock: []int32{0, 1}, mbType: MBType8x8 | MBTypeP0L0 | MBTypeP0L1 | MBTypeP1L0 | MBTypeP1L1 | MBTypeInterlaced, sub: &bSub, cbp: 4, cbpTable: 4},
			} {
				for _, deblock := range shape.deblock {
					t.Run(fmt.Sprintf("picture%d/cabac%d/%s/mode%d", picture, cabac, shape.name, deblock), func(t *testing.T) {
						sh := &SliceHeader{
							SliceTypeNoS:     shape.slice,
							PictureStructure: picture,
							DeblockingFilter: deblock,
							SPS: &SPS{
								BitDepthLuma:     10,
								BitDepthChroma:   10,
								ChromaFormatIDC:  3,
								FrameMBSOnlyFlag: 0,
								MBAFF:            1,
							},
							PPS: &PPS{CABAC: cabac},
						}
						if err := validateHighFrameSliceMacroblockForReconstructWithSubMB(sh, shape.mbType, shape.sub, shape.cbp, shape.cbpTable); err != nil {
							t.Fatalf("validate high10 444 field unweighted %s reconstruct err = %v, want nil", shape.name, err)
						}
					})
				}
			}
		}
	}
}

func TestValidateHighFrameSliceMacroblockForReconstructAllowsHigh10Chroma422UnweightedFieldIBoundary(t *testing.T) {
	for _, picture := range []int32{PictureTopField, PictureBottomField} {
		for _, cabac := range []int32{0, 1} {
			for _, shape := range []struct {
				name   string
				mbType uint32
			}{
				{name: "intra4x4", mbType: MBTypeIntra4x4},
				{name: "intra16x16", mbType: MBTypeIntra16x16},
				{name: "intrapcm", mbType: MBTypeIntraPCM},
			} {
				t.Run(fmt.Sprintf("picture%d/cabac%d/%s/mode2", picture, cabac, shape.name), func(t *testing.T) {
					sh := &SliceHeader{
						SliceTypeNoS:     PictureTypeI,
						PictureStructure: picture,
						DeblockingFilter: 2,
						SPS: &SPS{
							BitDepthLuma:     10,
							BitDepthChroma:   10,
							ChromaFormatIDC:  2,
							FrameMBSOnlyFlag: 0,
							MBAFF:            1,
						},
						PPS: &PPS{CABAC: cabac},
					}
					if err := validateHighFrameSliceMacroblockForReconstructWithSubMB(sh, shape.mbType, nil, 0, 0); err != nil {
						t.Fatalf("validate high10 422 field unweighted I boundary reconstruct err = %v, want nil", err)
					}
				})
			}
		}
	}
}

func TestValidateHighFrameSliceMacroblockForReconstructAllowsHigh10ChromaFieldUnweightedBBoundary(t *testing.T) {
	bSub := [4]uint32{
		MBType16x16 | MBTypeP0L0,
		MBType16x16 | MBTypeP0L1,
		MBType16x16 | MBTypeP0L0 | MBTypeP0L1,
		MBType16x16 | MBTypeP0L1,
	}
	for _, chroma := range []uint32{2, 3} {
		for _, picture := range []int32{PictureTopField, PictureBottomField} {
			for _, cabac := range []int32{0, 1} {
				for _, shape := range []struct {
					name     string
					mbType   uint32
					sub      *[4]uint32
					cbp      int
					cbpTable int
				}{
					{name: "b16x16-l0", mbType: MBType16x16 | MBTypeP0L0 | MBTypeInterlaced},
					{name: "b16x16-l1", mbType: MBType16x16 | MBTypeP0L1 | MBTypeInterlaced},
					{name: "b16x16-bi", mbType: MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeInterlaced},
					{name: "b16x16-direct", mbType: MBType16x16 | MBTypeL0L1 | MBTypeDirect2 | MBTypeInterlaced},
					{name: "direct-skip", mbType: MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2 | MBTypeSkip | MBTypeInterlaced},
					{name: "b16x8-residual", mbType: MBType16x8 | MBTypeP0L0 | MBTypeP1L1 | MBTypeInterlaced, cbp: 1, cbpTable: 1},
					{name: "b8x16", mbType: MBType8x16 | MBTypeP0L0 | MBTypeP1L1 | MBTypeInterlaced},
					{name: "b8x8-sub", mbType: MBType8x8 | MBTypeP0L0 | MBTypeP0L1 | MBTypeP1L0 | MBTypeP1L1 | MBTypeInterlaced, sub: &bSub, cbp: 4, cbpTable: 4},
				} {
					t.Run(fmt.Sprintf("chroma%d/picture%d/cabac%d/%s/mode2", chroma, picture, cabac, shape.name), func(t *testing.T) {
						sh := &SliceHeader{
							SliceTypeNoS:     PictureTypeB,
							PictureStructure: picture,
							DeblockingFilter: 2,
							SPS: &SPS{
								BitDepthLuma:     10,
								BitDepthChroma:   10,
								ChromaFormatIDC:  chroma,
								FrameMBSOnlyFlag: 0,
								MBAFF:            1,
							},
							PPS: &PPS{CABAC: cabac},
						}
						if err := validateHighFrameSliceMacroblockForReconstructWithSubMB(sh, shape.mbType, shape.sub, shape.cbp, shape.cbpTable); err != nil {
							t.Fatalf("validate high10 chroma field unweighted B boundary reconstruct err = %v, want nil", err)
						}
					})
				}
			}
		}
	}
}

func TestValidateHighFrameSliceMacroblockForReconstructAllowsHigh1214ChromaFieldWeightedB(t *testing.T) {
	weights := []struct {
		name             string
		weightedBipredID uint32
		useWeight        int32
		useWeightChroma  int32
	}{
		{name: "explicit-luma", weightedBipredID: 1, useWeight: 1},
		{name: "explicit-chroma", weightedBipredID: 1, useWeightChroma: 1},
		{name: "explicit-luma-chroma", weightedBipredID: 1, useWeight: 1, useWeightChroma: 1},
		{name: "implicit", weightedBipredID: 2, useWeight: 2, useWeightChroma: 2},
	}
	shapes := []struct {
		name     string
		mbType   uint32
		sub      *[4]uint32
		cbp      int
		cbpTable int
	}{
		{name: "b16x16-l0", mbType: MBType16x16 | MBTypeP0L0 | MBTypeInterlaced},
		{name: "b16x16-l1", mbType: MBType16x16 | MBTypeP0L1 | MBTypeInterlaced},
		{name: "b16x16-bi", mbType: MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeInterlaced},
		{name: "b16x8-bi-residual", mbType: MBType16x8 | MBTypeP0L0 | MBTypeP0L1 | MBTypeP1L0 | MBTypeP1L1 | MBTypeInterlaced, cbp: 1, cbpTable: 1},
		{name: "b8x16-l0-l1", mbType: MBType8x16 | MBTypeP0L0 | MBTypeP1L1 | MBTypeInterlaced},
		{name: "b8x8-explicit-sub", mbType: MBType8x8 | MBTypeP0L0 | MBTypeP0L1 | MBTypeP1L0 | MBTypeP1L1 | MBTypeInterlaced, sub: &([4]uint32{
			MBType16x16 | MBTypeP0L0,
			MBType16x16 | MBTypeP0L1,
			MBType16x16 | MBTypeP0L0 | MBTypeP0L1,
			MBType16x16 | MBTypeP0L1,
		})},
	}
	for _, bitDepth := range []int32{12, 14} {
		for _, chromaFormatIDC := range []uint32{2, 3} {
			for _, picture := range []int32{PictureTopField, PictureBottomField} {
				for _, deblock := range []int32{0, 1, 2} {
					for _, weight := range weights {
						for _, shape := range shapes {
							t.Run(fmt.Sprintf("%s/chroma%d/picture%d/mode%d/%s/%s", bitDepthName(bitDepth), chromaFormatIDC, picture, deblock, weight.name, shape.name), func(t *testing.T) {
								sh := &SliceHeader{
									SliceTypeNoS:     PictureTypeB,
									PictureStructure: picture,
									DeblockingFilter: deblock,
									SPS: &SPS{
										BitDepthLuma:     bitDepth,
										BitDepthChroma:   bitDepth,
										ChromaFormatIDC:  chromaFormatIDC,
										FrameMBSOnlyFlag: 0,
										MBAFF:            1,
									},
									PPS: &PPS{WeightedBipredIDC: weight.weightedBipredID},
									PredWeightTable: PredWeightTable{
										UseWeight:       weight.useWeight,
										UseWeightChroma: weight.useWeightChroma,
									},
								}
								if err := validateHighFrameSliceMacroblockForReconstructWithSubMB(sh, shape.mbType, shape.sub, shape.cbp, shape.cbpTable); err != nil {
									t.Fatalf("validate %s chroma field weighted B reconstruct err = %v, want nil", bitDepthName(bitDepth), err)
								}
							})
						}
					}
				}
			}
		}
	}
}

func TestValidateHighFrameSliceMacroblockForReconstructAllowsBDirectSkip(t *testing.T) {
	sh := &SliceHeader{SliceTypeNoS: PictureTypeB}
	directSub := [4]uint32{
		MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2,
		MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2,
		MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2,
		MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2,
	}
	directSubL0 := [4]uint32{
		MBType16x16 | MBTypeL0 | MBTypeDirect2,
		MBType16x16 | MBTypeL0 | MBTypeDirect2,
		MBType16x16 | MBTypeL0 | MBTypeDirect2,
		MBType16x16 | MBTypeL0 | MBTypeDirect2,
	}
	directSubL1 := [4]uint32{
		MBType16x16 | MBTypeL1 | MBTypeDirect2,
		MBType16x16 | MBTypeL1 | MBTypeDirect2,
		MBType16x16 | MBTypeL1 | MBTypeDirect2,
		MBType16x16 | MBTypeL1 | MBTypeDirect2,
	}
	for _, tt := range []struct {
		name string
		typ  uint32
		sub  *[4]uint32
	}{
		{name: "spatial 16x16", typ: MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2 | MBTypeSkip},
		{name: "spatial 16x16 list0 only", typ: MBType16x16 | MBTypeP0L0 | MBTypeDirect2 | MBTypeSkip, sub: &directSubL0},
		{name: "spatial 16x16 list1 only", typ: MBType16x16 | MBTypeP0L1 | MBTypeDirect2 | MBTypeSkip, sub: &directSubL1},
		{name: "spatial 16x16 list0 full", typ: MBType16x16 | MBTypeL0 | MBTypeDirect2 | MBTypeSkip, sub: &directSubL0},
		{name: "spatial 16x16 list1 full", typ: MBType16x16 | MBTypeL1 | MBTypeDirect2 | MBTypeSkip, sub: &directSubL1},
		{name: "temporal 16x16", typ: MBType16x16 | MBTypeL0L1 | MBTypeDirect2 | MBTypeSkip},
		{name: "temporal 16x8", typ: MBType16x8 | MBTypeL0L1 | MBTypeDirect2 | MBTypeSkip, sub: &directSub},
		{name: "temporal 8x16", typ: MBType8x16 | MBTypeL0L1 | MBTypeDirect2 | MBTypeSkip, sub: &directSub},
		{name: "temporal 16x8 list0 only", typ: MBType16x8 | MBTypeL0 | MBTypeDirect2 | MBTypeSkip, sub: &directSubL0},
		{name: "temporal 8x16 list0 only", typ: MBType8x16 | MBTypeL0 | MBTypeDirect2 | MBTypeSkip, sub: &directSubL0},
		{name: "temporal 16x8 list1 only", typ: MBType16x8 | MBTypeL1 | MBTypeDirect2 | MBTypeSkip, sub: &directSubL1},
		{name: "temporal 8x16 list1 only", typ: MBType8x16 | MBTypeL1 | MBTypeDirect2 | MBTypeSkip, sub: &directSubL1},
		{name: "temporal 8x8 list0 only", typ: MBType8x8 | MBTypeL0 | MBTypeDirect2 | MBTypeSkip, sub: &directSubL0},
		{name: "temporal 8x8 list1 only", typ: MBType8x8 | MBTypeL1 | MBTypeDirect2 | MBTypeSkip, sub: &directSubL1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateHighFrameSliceMacroblockForReconstructWithSubMB(sh, tt.typ, tt.sub, 0, 0); err != nil {
				t.Fatalf("validate high B direct skip %#x err = %v, want nil", tt.typ, err)
			}
		})
	}
}

func TestValidateHighFrameSliceMacroblockForReconstructAllowsB8x8DirectSubNoResidual(t *testing.T) {
	sh := &SliceHeader{SliceTypeNoS: PictureTypeB}
	mbType := MBType8x8 | MBTypeP0L0 | MBTypeP0L1 | MBTypeP1L0 | MBTypeP1L1

	for _, tt := range []struct {
		name string
		sub  [4]uint32
	}{
		{
			name: "direct 8x8 inference",
			sub: [4]uint32{
				MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2,
				MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2,
				MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2,
				MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2,
			},
		},
		{
			name: "direct sub 4x4",
			sub: [4]uint32{
				MBType8x8 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2,
				MBType8x8 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2,
				MBType8x8 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2,
				MBType8x8 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2,
			},
		},
		{
			name: "spatial direct sub 4x4",
			sub: [4]uint32{
				MBType8x8 | MBTypeL0L1 | MBTypeDirect2,
				MBType8x8 | MBTypeL0L1 | MBTypeDirect2,
				MBType8x8 | MBTypeL0L1 | MBTypeDirect2,
				MBType8x8 | MBTypeL0L1 | MBTypeDirect2,
			},
		},
		{
			name: "spatial direct sub 4x4 list0 only",
			sub: [4]uint32{
				MBType8x8 | MBTypeL0 | MBTypeDirect2,
				MBType8x8 | MBTypeL0 | MBTypeDirect2,
				MBType8x8 | MBTypeL0 | MBTypeDirect2,
				MBType8x8 | MBTypeL0 | MBTypeDirect2,
			},
		},
		{
			name: "spatial direct sub 4x4 list1 only",
			sub: [4]uint32{
				MBType8x8 | MBTypeL1 | MBTypeDirect2,
				MBType8x8 | MBTypeL1 | MBTypeDirect2,
				MBType8x8 | MBTypeL1 | MBTypeDirect2,
				MBType8x8 | MBTypeL1 | MBTypeDirect2,
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateHighFrameSliceMacroblockForReconstructWithSubMB(sh, mbType, &tt.sub, 0, 0); err != nil {
				t.Fatalf("validate high B direct-sub err = %v, want nil", err)
			}
		})
	}
}

func TestValidateHighFrameSliceMacroblockForReconstructAllowsNeutralB8x8DirectSubResidual(t *testing.T) {
	sh := &SliceHeader{SliceTypeNoS: PictureTypeB, PPS: &PPS{}}
	mbType := MBType8x8 | MBTypeP0L0 | MBTypeP0L1 | MBTypeP1L0 | MBTypeP1L1
	sub := [4]uint32{
		MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2,
		MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2,
		MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2,
		MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2,
	}

	for _, tt := range []struct {
		name     string
		cbp      int
		cbpTable int
	}{
		{name: "luma", cbp: 0x1, cbpTable: 0x1001},
		{name: "luma-chroma", cbp: 0x31, cbpTable: 0xf031},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateHighFrameSliceMacroblockForReconstructWithSubMB(sh, mbType, &sub, tt.cbp, tt.cbpTable); err != nil {
				t.Fatalf("validate high B direct-sub residual err = %v, want nil", err)
			}
		})
	}
}

func TestValidateHighFrameSliceMacroblockForReconstructAllowsTopLevelB8x8Direct(t *testing.T) {
	sh := &SliceHeader{SliceTypeNoS: PictureTypeB, PPS: &PPS{}}
	mbType := MBType8x8 | MBTypeL0L1 | MBTypeDirect2

	for _, tt := range []struct {
		name     string
		mbType   uint32
		sub      [4]uint32
		cbp      int
		cbpTable int
	}{
		{
			name: "direct 8x8 inference",
			sub: [4]uint32{
				MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2,
				MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2,
				MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2,
				MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2,
			},
		},
		{
			name: "direct sub 4x4 residual",
			sub: [4]uint32{
				MBType8x8 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2,
				MBType8x8 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2,
				MBType8x8 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2,
				MBType8x8 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2,
			},
			cbp:      0x1,
			cbpTable: 0x1001,
		},
		{
			name:   "direct 8x8 inference 8x8 dct residual",
			mbType: MBType8x8 | MBTypeL0L1 | MBTypeDirect2 | MBType8x8DCT,
			sub: [4]uint32{
				MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2,
				MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2,
				MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2,
				MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2,
			},
			cbp:      0x0f,
			cbpTable: 0x0f,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			typ := tt.mbType
			if typ == 0 {
				typ = mbType
			}
			if err := validateHighFrameSliceMacroblockForReconstructWithSubMB(sh, typ, &tt.sub, tt.cbp, tt.cbpTable); err != nil {
				t.Fatalf("validate high top-level B direct 8x8 err = %v, want nil", err)
			}
		})
	}
}

func TestValidateHighFrameSliceMacroblockForReconstructAllowsNeutralBDeblockingDirectSkip(t *testing.T) {
	for _, tt := range []struct {
		name   string
		pps    *PPS
		mbType uint32
	}{
		{name: "cavlc temporal skip", pps: &PPS{}, mbType: MBType16x16 | MBTypeL0L1 | MBTypeDirect2 | MBTypeSkip},
		{name: "cabac temporal skip", pps: &PPS{CABAC: 1}, mbType: MBType16x16 | MBTypeL0L1 | MBTypeDirect2 | MBTypeSkip},
		{name: "cavlc spatial skip", pps: &PPS{}, mbType: MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2 | MBTypeSkip},
		{name: "cabac spatial skip", pps: &PPS{CABAC: 1}, mbType: MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2 | MBTypeSkip},
	} {
		t.Run(tt.name, func(t *testing.T) {
			sh := &SliceHeader{
				SliceTypeNoS:     PictureTypeB,
				DeblockingFilter: 1,
				PPS:              tt.pps,
			}
			if err := validateHighFrameSliceMacroblockForReconstruct(sh, tt.mbType, 0, 0); err != nil {
				t.Fatalf("validate high B deblock direct skip err = %v, want nil", err)
			}
		})
	}
}

func TestValidateHighFrameSliceMacroblockForReconstructAllowsNeutralBDeblockingDirectSub(t *testing.T) {
	mbType := MBType8x8 | MBTypeP0L0 | MBTypeP0L1 | MBTypeP1L0 | MBTypeP1L1
	for _, tt := range []struct {
		name string
		pps  *PPS
		sub  [4]uint32
	}{
		{
			name: "cavlc direct 8x8",
			pps:  &PPS{},
			sub: [4]uint32{
				MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2,
				MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2,
				MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2,
				MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2,
			},
		},
		{
			name: "cabac direct 8x8",
			pps:  &PPS{CABAC: 1},
			sub: [4]uint32{
				MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2,
				MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2,
				MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2,
				MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2,
			},
		},
		{
			name: "cavlc direct sub 4x4",
			pps:  &PPS{},
			sub: [4]uint32{
				MBType8x8 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2,
				MBType8x8 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2,
				MBType8x8 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2,
				MBType8x8 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2,
			},
		},
		{
			name: "cabac spatial direct sub 4x4",
			pps:  &PPS{CABAC: 1},
			sub: [4]uint32{
				MBType8x8 | MBTypeL0L1 | MBTypeDirect2,
				MBType8x8 | MBTypeL0L1 | MBTypeDirect2,
				MBType8x8 | MBTypeL0L1 | MBTypeDirect2,
				MBType8x8 | MBTypeL0L1 | MBTypeDirect2,
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			sh := &SliceHeader{
				SliceTypeNoS:     PictureTypeB,
				DeblockingFilter: 1,
				PPS:              tt.pps,
			}
			if err := validateHighFrameSliceMacroblockForReconstructWithSubMB(sh, mbType, &tt.sub, 0, 0); err != nil {
				t.Fatalf("validate high B deblock direct-sub err = %v, want nil", err)
			}
		})
	}
}

func TestValidateHighFrameSliceMacroblockForReconstructAllowsImplicitWeightedBDeblockingDirectSub(t *testing.T) {
	mbType := MBType8x8 | MBTypeP0L0 | MBTypeP0L1 | MBTypeP1L0 | MBTypeP1L1
	for _, tt := range []struct {
		name string
		pps  *PPS
		sub  [4]uint32
	}{
		{
			name: "cavlc direct 8x8",
			pps:  &PPS{WeightedBipredIDC: 2},
			sub: [4]uint32{
				MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2,
				MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2,
				MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2,
				MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2,
			},
		},
		{
			name: "cabac direct 8x8",
			pps:  &PPS{CABAC: 1, WeightedBipredIDC: 2},
			sub: [4]uint32{
				MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2,
				MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2,
				MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2,
				MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2,
			},
		},
		{
			name: "cavlc direct sub 4x4",
			pps:  &PPS{WeightedBipredIDC: 2},
			sub: [4]uint32{
				MBType8x8 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2,
				MBType8x8 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2,
				MBType8x8 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2,
				MBType8x8 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2,
			},
		},
		{
			name: "cabac spatial direct sub 4x4",
			pps:  &PPS{CABAC: 1, WeightedBipredIDC: 2},
			sub: [4]uint32{
				MBType8x8 | MBTypeL0L1 | MBTypeDirect2,
				MBType8x8 | MBTypeL0L1 | MBTypeDirect2,
				MBType8x8 | MBTypeL0L1 | MBTypeDirect2,
				MBType8x8 | MBTypeL0L1 | MBTypeDirect2,
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			sh := &SliceHeader{
				SliceTypeNoS:     PictureTypeB,
				DeblockingFilter: 1,
				PPS:              tt.pps,
			}
			if err := validateHighFrameSliceMacroblockForReconstructWithSubMB(sh, mbType, &tt.sub, 0, 0); err != nil {
				t.Fatalf("validate high implicit weighted B deblock direct-sub err = %v, want nil", err)
			}
		})
	}
}

func TestValidateHighFrameSliceMacroblockForReconstructAllowsBPartitionedExplicit(t *testing.T) {
	unweighted := &SliceHeader{SliceTypeNoS: PictureTypeB}
	implicitWeighted := &SliceHeader{SliceTypeNoS: PictureTypeB, PPS: &PPS{WeightedBipredIDC: 2}}
	b8x8 := MBType8x8 | MBTypeP0L0 | MBTypeP0L1 | MBTypeP1L0 | MBTypeP1L1
	allL0 := [4]uint32{
		MBType16x16 | MBTypeP0L0,
		MBType16x16 | MBTypeP0L0,
		MBType16x16 | MBTypeP0L0,
		MBType16x16 | MBTypeP0L0,
	}
	mixedSubPartitions := [4]uint32{
		MBType16x16 | MBTypeP0L0,
		MBType16x16 | MBTypeP0L1,
		MBType16x16 | MBTypeP0L0 | MBTypeP0L1,
		MBType8x8 | MBTypeP0L0 | MBTypeP0L1 | MBTypeP1L0 | MBTypeP1L1,
	}
	mixedDirectExplicitSub := [4]uint32{
		MBType16x16 | MBTypeP0L1,
		MBType16x16 | MBTypeP0L0,
		MBType16x16 | MBTypeP0L1,
		MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2,
	}

	tests := []struct {
		name     string
		sh       *SliceHeader
		mbType   uint32
		sub      *[4]uint32
		cbp      int
		cbpTable int
	}{
		{name: "b16x8 l0 l0", sh: unweighted, mbType: MBType16x8 | MBTypeP0L0 | MBTypeP1L0},
		{name: "b16x8 l0 l1 residual", sh: unweighted, mbType: MBType16x8 | MBTypeP0L0 | MBTypeP1L1, cbp: 1, cbpTable: 1},
		{name: "b8x16 l1 l1", sh: unweighted, mbType: MBType8x16 | MBTypeP0L1 | MBTypeP1L1},
		{name: "b8x16 bidirectional residual", sh: unweighted, mbType: MBType8x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeP1L0 | MBTypeP1L1, cbp: 3, cbpTable: 3},
		{name: "b8x8 explicit all l0", sh: unweighted, mbType: b8x8, sub: &allL0},
		{name: "b8x8 explicit mixed subpartitions residual", sh: unweighted, mbType: b8x8, sub: &mixedSubPartitions, cbp: 2, cbpTable: 2},
		{name: "b8x8 mixed direct explicit no residual", sh: unweighted, mbType: b8x8, sub: &mixedDirectExplicitSub},
		{name: "b8x8 mixed direct explicit residual", sh: unweighted, mbType: b8x8, sub: &mixedDirectExplicitSub, cbp: 2, cbpTable: 2},
		{name: "implicit weighted b16x8 l0 l0", sh: implicitWeighted, mbType: MBType16x8 | MBTypeP0L0 | MBTypeP1L0},
		{name: "implicit weighted b8x16 l1 l1", sh: implicitWeighted, mbType: MBType8x16 | MBTypeP0L1 | MBTypeP1L1},
		{name: "implicit weighted b8x8 explicit all l0", sh: implicitWeighted, mbType: b8x8, sub: &allL0},
		{name: "implicit weighted b8x8 explicit mixed subpartitions residual", sh: implicitWeighted, mbType: b8x8, sub: &mixedSubPartitions, cbp: 2, cbpTable: 2},
		{name: "implicit weighted b8x8 mixed direct explicit no residual", sh: implicitWeighted, mbType: b8x8, sub: &mixedDirectExplicitSub},
		{name: "implicit weighted b8x8 mixed direct explicit residual", sh: implicitWeighted, mbType: b8x8, sub: &mixedDirectExplicitSub, cbp: 2, cbpTable: 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateHighFrameSliceBaseMacroblockForDecode(PictureTypeB, tt.mbType); err != nil {
				t.Fatalf("validate high B partitioned base err = %v, want nil", err)
			}
			if err := validateHighFrameSliceMacroblockForReconstructWithSubMB(tt.sh, tt.mbType, tt.sub, tt.cbp, tt.cbpTable); err != nil {
				t.Fatalf("validate high B partitioned reconstruct err = %v, want nil", err)
			}
		})
	}
}

func TestValidateHighFrameSliceMacroblockForReconstructAllowsB16x16Deblocking(t *testing.T) {
	for _, tt := range []struct {
		name   string
		pps    *PPS
		mbType uint32
	}{
		{name: "cavlc l0", pps: &PPS{}, mbType: MBType16x16 | MBTypeP0L0},
		{name: "cabac l0", pps: &PPS{CABAC: 1}, mbType: MBType16x16 | MBTypeP0L0},
		{name: "cavlc non-direct", pps: &PPS{}, mbType: MBType16x16 | MBTypeP0L0 | MBTypeP0L1},
		{name: "cabac non-direct", pps: &PPS{CABAC: 1}, mbType: MBType16x16 | MBTypeP0L0 | MBTypeP0L1},
		{name: "cavlc temporal direct", pps: &PPS{}, mbType: MBType16x16 | MBTypeL0L1 | MBTypeDirect2},
		{name: "cabac temporal direct", pps: &PPS{CABAC: 1}, mbType: MBType16x16 | MBTypeL0L1 | MBTypeDirect2},
		{name: "cavlc spatial direct", pps: &PPS{}, mbType: MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2},
		{name: "cabac spatial direct", pps: &PPS{CABAC: 1}, mbType: MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2},
	} {
		t.Run(tt.name, func(t *testing.T) {
			sh := &SliceHeader{
				SliceTypeNoS:     PictureTypeB,
				DeblockingFilter: 1,
				PPS:              tt.pps,
			}
			if err := validateHighFrameSliceMacroblockForReconstruct(sh, tt.mbType, 0x31, 0xf031); err != nil {
				t.Fatalf("validate high B16x16 deblock err = %v, want nil", err)
			}
		})
	}
}

func TestValidateHighFrameSliceMacroblockForReconstructAllowsImplicitPartitionedBDeblocking(t *testing.T) {
	bExplicitSub := [4]uint32{
		MBType16x16 | MBTypeP0L0,
		MBType16x16 | MBTypeP0L1,
		MBType16x16 | MBTypeP0L0 | MBTypeP0L1,
		MBType16x16 | MBTypeP0L1,
	}
	for _, tt := range []struct {
		name     string
		pps      *PPS
		mbType   uint32
		sub      *[4]uint32
		cbp      int
		cbpTable int
	}{
		{name: "cavlc b16x8", pps: &PPS{WeightedBipredIDC: 2}, mbType: MBType16x8 | MBTypeP0L1 | MBTypeP1L0},
		{name: "cabac b16x8", pps: &PPS{CABAC: 1, WeightedBipredIDC: 2}, mbType: MBType16x8 | MBTypeP0L1 | MBTypeP1L0},
		{name: "cavlc b8x16", pps: &PPS{WeightedBipredIDC: 2}, mbType: MBType8x16 | MBTypeP0L1 | MBTypeP1L0},
		{name: "cabac b8x16", pps: &PPS{CABAC: 1, WeightedBipredIDC: 2}, mbType: MBType8x16 | MBTypeP0L1 | MBTypeP1L0},
		{name: "cavlc b8x8 residual", pps: &PPS{WeightedBipredIDC: 2}, mbType: MBType8x8 | MBTypeP0L0 | MBTypeP0L1 | MBTypeP1L0 | MBTypeP1L1, sub: &bExplicitSub, cbp: 0x5, cbpTable: 0x5005},
		{name: "cabac b8x8 residual", pps: &PPS{CABAC: 1, WeightedBipredIDC: 2}, mbType: MBType8x8 | MBTypeP0L0 | MBTypeP0L1 | MBTypeP1L0 | MBTypeP1L1, sub: &bExplicitSub, cbp: 0x5, cbpTable: 0x5},
	} {
		t.Run(tt.name, func(t *testing.T) {
			sh := &SliceHeader{
				SliceTypeNoS:     PictureTypeB,
				DeblockingFilter: 1,
				PPS:              tt.pps,
			}
			if err := validateHighFrameSliceMacroblockForReconstructWithSubMB(sh, tt.mbType, tt.sub, tt.cbp, tt.cbpTable); err != nil {
				t.Fatalf("validate high implicit partitioned B deblock err = %v, want nil", err)
			}
		})
	}
}

func TestValidateHighFrameSliceMacroblockForReconstructAllowsNeutralPartitionedBDeblocking(t *testing.T) {
	bExplicitSub := [4]uint32{
		MBType16x16 | MBTypeP0L0,
		MBType16x16 | MBTypeP0L1,
		MBType16x16 | MBTypeP0L0 | MBTypeP0L1,
		MBType16x16 | MBTypeP0L1,
	}
	for _, tt := range []struct {
		name     string
		pps      *PPS
		mbType   uint32
		sub      *[4]uint32
		cbp      int
		cbpTable int
	}{
		{name: "cavlc b16x8", pps: &PPS{}, mbType: MBType16x8 | MBTypeP0L1 | MBTypeP1L0},
		{name: "cabac b16x8", pps: &PPS{CABAC: 1}, mbType: MBType16x8 | MBTypeP0L1 | MBTypeP1L0},
		{name: "cavlc b8x16", pps: &PPS{}, mbType: MBType8x16 | MBTypeP0L1 | MBTypeP1L0},
		{name: "cabac b8x16", pps: &PPS{CABAC: 1}, mbType: MBType8x16 | MBTypeP0L1 | MBTypeP1L0},
		{name: "cavlc b8x8 residual", pps: &PPS{}, mbType: MBType8x8 | MBTypeP0L0 | MBTypeP0L1 | MBTypeP1L0 | MBTypeP1L1, sub: &bExplicitSub, cbp: 0x7, cbpTable: 0x7007},
		{name: "cabac b8x8 residual", pps: &PPS{CABAC: 1}, mbType: MBType8x8 | MBTypeP0L0 | MBTypeP0L1 | MBTypeP1L0 | MBTypeP1L1, sub: &bExplicitSub, cbp: 0x7, cbpTable: 0x7},
	} {
		t.Run(tt.name, func(t *testing.T) {
			sh := &SliceHeader{
				SliceTypeNoS:     PictureTypeB,
				DeblockingFilter: 1,
				PPS:              tt.pps,
			}
			if err := validateHighFrameSliceMacroblockForReconstructWithSubMB(sh, tt.mbType, tt.sub, tt.cbp, tt.cbpTable); err != nil {
				t.Fatalf("validate high neutral partitioned B deblock err = %v, want nil", err)
			}
		})
	}
}

func TestValidateHighFrameSliceMacroblockForReconstructAllowsImplicitWeightedPartitionedAndDirectResidual(t *testing.T) {
	bDirectSub := [4]uint32{
		MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2,
		MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2,
		MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2,
		MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2,
	}
	bDirectSubL1 := [4]uint32{
		MBType16x16 | MBTypeL1 | MBTypeDirect2,
		MBType16x16 | MBTypeL1 | MBTypeDirect2,
		MBType16x16 | MBTypeL1 | MBTypeDirect2,
		MBType16x16 | MBTypeL1 | MBTypeDirect2,
	}
	bDirectSubCarrier := MBType8x8 | MBTypeP0L0 | MBTypeP0L1 | MBTypeP1L0 | MBTypeP1L1
	for _, tt := range []struct {
		name   string
		sh     *SliceHeader
		mbType uint32
		sub    *[4]uint32
		cbp    int
	}{
		{
			name:   "implicit b16x8 residual",
			sh:     &SliceHeader{SliceTypeNoS: PictureTypeB, PPS: &PPS{WeightedBipredIDC: 2}},
			mbType: MBType16x8 | MBTypeP0L0 | MBTypeP1L1,
			cbp:    0x1,
		},
		{
			name:   "implicit b8x16 residual",
			sh:     &SliceHeader{SliceTypeNoS: PictureTypeB, PPS: &PPS{WeightedBipredIDC: 2}},
			mbType: MBType8x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeP1L0 | MBTypeP1L1,
			cbp:    0x3,
		},
		{
			name:   "implicit direct sub residual",
			sh:     &SliceHeader{SliceTypeNoS: PictureTypeB, PPS: &PPS{WeightedBipredIDC: 2}},
			mbType: bDirectSubCarrier,
			sub:    &bDirectSub,
			cbp:    0x1,
		},
		{
			name:   "direct 8x16 partition residual",
			sh:     &SliceHeader{SliceTypeNoS: PictureTypeB},
			mbType: MBType8x16 | MBTypeL0L1 | MBTypeDirect2,
			sub:    &bDirectSub,
			cbp:    0x2,
		},
		{
			name:   "direct 8x16 partition 8x8 dct residual",
			sh:     &SliceHeader{SliceTypeNoS: PictureTypeB},
			mbType: MBType8x16 | MBTypeL0L1 | MBTypeDirect2 | MBType8x8DCT,
			sub:    &bDirectSub,
			cbp:    0x0f,
		},
		{
			name:   "direct 16x8 partition list1 residual",
			sh:     &SliceHeader{SliceTypeNoS: PictureTypeB},
			mbType: MBType16x8 | MBTypeL1 | MBTypeDirect2,
			sub:    &bDirectSubL1,
			cbp:    0x1,
		},
		{
			name:   "deblock implicit b16x8 residual",
			sh:     &SliceHeader{SliceTypeNoS: PictureTypeB, DeblockingFilter: 1, PPS: &PPS{WeightedBipredIDC: 2}},
			mbType: MBType16x8 | MBTypeP0L0 | MBTypeP1L0,
			cbp:    0x1,
		},
		{
			name:   "deblock implicit direct sub residual",
			sh:     &SliceHeader{SliceTypeNoS: PictureTypeB, DeblockingFilter: 1, PPS: &PPS{WeightedBipredIDC: 2}},
			mbType: bDirectSubCarrier,
			sub:    &bDirectSub,
			cbp:    0x1,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateHighFrameSliceMacroblockForReconstructWithSubMB(tt.sh, tt.mbType, tt.sub, tt.cbp, tt.cbp); err != nil {
				t.Fatalf("validate err = %v, want nil", err)
			}
		})
	}
}

func TestValidateHighFrameSliceMacroblockForReconstructRejectsPResidualGuardBoundaries(t *testing.T) {
	pSlice := &SliceHeader{SliceTypeNoS: PictureTypeP}
	bSlice := &SliceHeader{SliceTypeNoS: PictureTypeB}
	bImplicitWeightedSlice := &SliceHeader{SliceTypeNoS: PictureTypeB, PPS: &PPS{WeightedBipredIDC: 2}}
	bDeblockSlice := &SliceHeader{SliceTypeNoS: PictureTypeB, DeblockingFilter: 1, PPS: &PPS{}}
	bCABACDeblockSlice := &SliceHeader{SliceTypeNoS: PictureTypeB, DeblockingFilter: 1, PPS: &PPS{CABAC: 1}}
	bImplicitDeblockSlice := &SliceHeader{SliceTypeNoS: PictureTypeB, DeblockingFilter: 1, PPS: &PPS{WeightedBipredIDC: 2}}
	pSkip := MBType16x16 | MBTypeP0L0 | MBTypeP1L0 | MBTypeSkip
	bSkip := MBType16x16 | MBTypeL0L1 | MBTypeDirect2 | MBTypeSkip
	bDirectSubCarrier := MBType8x8 | MBTypeP0L0 | MBTypeP0L1 | MBTypeP1L0 | MBTypeP1L1
	bDirectSub := [4]uint32{
		MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2,
		MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2,
		MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2,
		MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2,
	}
	bExplicitSub := [4]uint32{
		MBType16x16 | MBTypeP0L0,
		MBType16x16 | MBTypeP0L0,
		MBType16x16 | MBTypeP0L0,
		MBType16x16 | MBTypeP0L0,
	}
	tests := []struct {
		name     string
		sh       *SliceHeader
		mbType   uint32
		sub      *[4]uint32
		cbp      int
		cbpTable int
		want     error
	}{
		{name: "nil header", sh: nil, mbType: MBType16x16 | MBTypeP0L0, want: ErrInvalidData},
		{name: "p skip cbp", sh: pSlice, mbType: pSkip, cbp: 1, cbpTable: 1, want: ErrUnsupported},
		{name: "p skip cbp table", sh: pSlice, mbType: pSkip, cbp: 0, cbpTable: 1, want: ErrUnsupported},
		{name: "p16x16 8x8 dct without luma cbp", sh: pSlice, mbType: MBType16x16 | MBTypeP0L0 | MBType8x8DCT, want: ErrUnsupported},
		{name: "direct in p", sh: pSlice, mbType: MBTypeDirect2 | MBTypeL0L1, cbp: 1, cbpTable: 1, want: ErrUnsupported},
		{name: "intra pcm in p", sh: pSlice, mbType: MBTypeIntraPCM, cbp: 1, cbpTable: 1, want: ErrUnsupported},
		{name: "intra16x16 8x8 dct in p", sh: pSlice, mbType: MBTypeIntra16x16 | MBType8x8DCT, cbp: 1, cbpTable: 1, want: ErrUnsupported},
		{name: "intra pcm in b", sh: bSlice, mbType: MBTypeIntraPCM, cbp: 1, cbpTable: 1, want: ErrUnsupported},
		{name: "negative cbp", sh: pSlice, mbType: MBType16x16 | MBTypeP0L0, cbp: -1, want: ErrUnsupported},
		{name: "negative cbp table", sh: pSlice, mbType: MBType16x16 | MBTypeP0L0, cbpTable: -1, want: ErrUnsupported},
		{name: "b direct", sh: bSlice, mbType: MBTypeDirect2 | MBTypeL0L1, cbp: 1, cbpTable: 1, want: ErrUnsupported},
		{name: "b direct list0 residual", sh: bSlice, mbType: MBType16x16 | MBTypeP0L0 | MBTypeDirect2, cbp: 1, cbpTable: 1},
		{name: "b direct list1 residual", sh: bSlice, mbType: MBType16x16 | MBTypeP0L1 | MBTypeDirect2, cbp: 0x0c, cbpTable: 0x0c},
		{name: "b direct partial temporal flags", sh: bSlice, mbType: MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeP1L0 | MBTypeDirect2, want: ErrUnsupported},
		{name: "b direct partition", sh: bSlice, mbType: MBType16x16 | MBType16x8 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2, want: ErrUnsupported},
		{name: "b direct partition missing sub", sh: bSlice, mbType: MBType8x16 | MBTypeL0L1 | MBTypeDirect2, cbp: 2, cbpTable: 2, want: ErrUnsupported},
		{name: "b direct partition 8x8 dct without luma cbp", sh: bSlice, mbType: MBType8x16 | MBTypeL0L1 | MBTypeDirect2 | MBType8x8DCT, sub: &bDirectSub, want: ErrUnsupported},
		{name: "b top-level direct 8x8 dct without luma cbp", sh: bSlice, mbType: MBType8x8 | MBTypeL0L1 | MBTypeDirect2 | MBType8x8DCT, sub: &bDirectSub, want: ErrUnsupported},
		{name: "b direct skip cbp", sh: bSlice, mbType: bSkip, cbp: 1, want: ErrUnsupported},
		{name: "b direct skip cbp table", sh: bSlice, mbType: bSkip, cbpTable: 1, want: ErrUnsupported},
		{name: "b direct skip unresolved", sh: bSlice, mbType: MBTypeDirect2 | MBTypeL0L1 | MBTypeSkip, want: ErrUnsupported},
		{name: "b direct skip partition", sh: bSlice, mbType: MBType8x8 | MBTypeL0L1 | MBTypeDirect2 | MBTypeSkip, want: ErrUnsupported},
		{name: "b16x16 8x8 dct without luma cbp", sh: bSlice, mbType: MBType16x16 | MBTypeP0L1 | MBType8x8DCT, want: ErrUnsupported},
		{name: "b direct 16x16 8x8 dct without luma cbp", sh: bSlice, mbType: MBType16x16 | MBTypeL0L1 | MBTypeDirect2 | MBType8x8DCT, want: ErrUnsupported},
		{name: "b direct sub without sub types", sh: bSlice, mbType: bDirectSubCarrier, want: ErrUnsupported},
		{name: "b direct sub cbp", sh: bSlice, mbType: bDirectSubCarrier, sub: &bDirectSub, cbp: 1},
		{name: "b direct sub cbp table", sh: bSlice, mbType: bDirectSubCarrier, sub: &bDirectSub, cbpTable: 1},
		{name: "b top-level direct 8x8 explicit sub remains guarded", sh: bSlice, mbType: MBType8x8 | MBTypeL0L1 | MBTypeDirect2, sub: &bExplicitSub, want: ErrUnsupported},
		{name: "b explicit 16x8 direct flag remains guarded", sh: bSlice, mbType: MBType16x8 | MBTypeP0L0 | MBTypeP1L0 | MBTypeDirect2, want: ErrUnsupported},
		{name: "b explicit 16x8 skip remains guarded", sh: bSlice, mbType: MBType16x8 | MBTypeP0L0 | MBTypeP1L0 | MBTypeSkip, want: ErrUnsupported},
		{name: "b explicit 16x8 missing partition direction", sh: bSlice, mbType: MBType16x8 | MBTypeP0L0, want: ErrUnsupported},
		{name: "b implicit weighted b16x8 residual", sh: bImplicitWeightedSlice, mbType: MBType16x8 | MBTypeP0L0 | MBTypeP1L1, cbp: 1, cbpTable: 1},
		{name: "b implicit weighted b8x16 residual", sh: bImplicitWeightedSlice, mbType: MBType8x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeP1L0 | MBTypeP1L1, cbp: 3, cbpTable: 3},
		{name: "b implicit weighted direct sub cbp", sh: bImplicitWeightedSlice, mbType: bDirectSubCarrier, sub: &bDirectSub, cbp: 1},
		{name: "b implicit weighted top-level direct 8x8 remains guarded", sh: bImplicitWeightedSlice, mbType: MBType8x8 | MBTypeL0L1 | MBTypeDirect2, sub: &bExplicitSub, want: ErrUnsupported},
		{name: "b deblock skip cbp remains guarded", sh: bDeblockSlice, mbType: bSkip, cbp: 1, want: ErrUnsupported},
		{name: "b deblock skip cbp table remains guarded", sh: bDeblockSlice, mbType: bSkip, cbpTable: 1, want: ErrUnsupported},
		{name: "b deblock implicit weighted direct skip cbp remains guarded", sh: bImplicitDeblockSlice, mbType: bSkip, cbp: 1, want: ErrUnsupported},
		{name: "b deblock direct sub cbp", sh: bDeblockSlice, mbType: bDirectSubCarrier, sub: &bDirectSub, cbp: 1},
		{name: "b deblock direct sub cbp table", sh: bDeblockSlice, mbType: bDirectSubCarrier, sub: &bDirectSub, cbpTable: 1},
		{name: "b deblock partitioned residual", sh: bDeblockSlice, mbType: MBType16x8 | MBTypeP0L0 | MBTypeP1L0, cbp: 1, cbpTable: 1},
		{name: "b deblock cabac partitioned residual", sh: bCABACDeblockSlice, mbType: MBType16x8 | MBTypeP0L0 | MBTypeP1L0, cbp: 1, cbpTable: 1},
		{name: "b deblock implicit weighted partitioned residual", sh: bImplicitDeblockSlice, mbType: MBType16x8 | MBTypeP0L0 | MBTypeP1L0, cbp: 1, cbpTable: 1},
		{name: "b deblock implicit weighted direct sub cbp", sh: bImplicitDeblockSlice, mbType: bDirectSubCarrier, sub: &bDirectSub, cbp: 1},
		{name: "b deblock implicit weighted direct sub cbp table", sh: bImplicitDeblockSlice, mbType: bDirectSubCarrier, sub: &bDirectSub, cbpTable: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateHighFrameSliceMacroblockForReconstructWithSubMB(tt.sh, tt.mbType, tt.sub, tt.cbp, tt.cbpTable); err != tt.want {
				t.Fatalf("validate err = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestValidateHighFrameSliceMacroblockForReconstructAllowsImplicitWeightedB16x16Deblock(t *testing.T) {
	for _, tt := range []struct {
		name     string
		pps      *PPS
		mbType   uint32
		cbp      int
		cbpTable int
	}{
		{name: "cavlc explicit", pps: &PPS{WeightedBipredIDC: 2}, mbType: MBType16x16 | MBTypeP0L0 | MBTypeP0L1, cbp: 0xf, cbpTable: 0xf00f},
		{name: "cabac explicit", pps: &PPS{CABAC: 1, WeightedBipredIDC: 2}, mbType: MBType16x16 | MBTypeP0L0 | MBTypeP0L1, cbp: 0xf, cbpTable: 0xf},
		{name: "cavlc temporal direct", pps: &PPS{WeightedBipredIDC: 2}, mbType: MBType16x16 | MBTypeL0L1 | MBTypeDirect2, cbp: 0x31, cbpTable: 0xf031},
		{name: "cabac temporal direct", pps: &PPS{CABAC: 1, WeightedBipredIDC: 2}, mbType: MBType16x16 | MBTypeL0L1 | MBTypeDirect2, cbp: 0x31, cbpTable: 0x31},
		{name: "cavlc spatial direct skip", pps: &PPS{WeightedBipredIDC: 2}, mbType: MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2 | MBTypeSkip},
		{name: "cabac spatial direct skip", pps: &PPS{CABAC: 1, WeightedBipredIDC: 2}, mbType: MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2 | MBTypeSkip},
	} {
		t.Run(tt.name, func(t *testing.T) {
			sh := &SliceHeader{
				SliceTypeNoS:     PictureTypeB,
				DeblockingFilter: 1,
				PPS:              tt.pps,
			}
			if err := validateHighFrameSliceMacroblockForReconstructWithSubMB(sh, tt.mbType, nil, tt.cbp, tt.cbpTable); err != nil {
				t.Fatalf("validate implicit weighted B16x16 deblock err = %v, want nil", err)
			}
		})
	}
}

func TestValidateHighFrameSliceMacroblockForReconstructAllowsHigh9ChromaImplicitWeightedBSliceBoundaryDeblock(t *testing.T) {
	for _, chromaFormatIDC := range []uint32{2, 3} {
		for _, shape := range []struct {
			name     string
			mbType   uint32
			cbp      int
			cbpTable int
		}{
			{name: "l0", mbType: MBType16x16 | MBTypeP0L0, cbp: 0xf, cbpTable: 0xf},
			{name: "l1", mbType: MBType16x16 | MBTypeP0L1, cbp: 0xf, cbpTable: 0xf},
			{name: "bi", mbType: MBType16x16 | MBTypeP0L0 | MBTypeP0L1, cbp: 0xf, cbpTable: 0xf},
			{name: "temporal-direct", mbType: MBType16x16 | MBTypeL0L1 | MBTypeDirect2, cbp: 0xf, cbpTable: 0xf},
			{name: "spatial-direct-skip", mbType: MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2 | MBTypeSkip},
		} {
			for _, cabac := range []int32{0, 1} {
				t.Run(fmt.Sprintf("%s/%s/cabac%d", chromaFormatName(int(chromaFormatIDC)), shape.name, cabac), func(t *testing.T) {
					sh := &SliceHeader{
						SliceTypeNoS:     PictureTypeB,
						PictureStructure: PictureFrame,
						DeblockingFilter: 2,
						SPS: &SPS{
							BitDepthLuma:    9,
							ChromaFormatIDC: chromaFormatIDC,
						},
						PPS: &PPS{
							CABAC:             cabac,
							WeightedBipredIDC: 2,
						},
						PredWeightTable: PredWeightTable{
							UseWeight:       2,
							UseWeightChroma: 2,
						},
					}
					if err := validateHighFrameSliceMacroblockForReconstructWithSubMB(sh, shape.mbType, nil, shape.cbp, shape.cbpTable); err != nil {
						t.Fatalf("validate high9 chroma implicit weighted-B mode-2 deblock err = %v, want nil", err)
					}
				})
			}
		}
	}
}

func TestValidateHighFrameSliceMacroblockForReconstructAllowsHigh9ChromaExplicitWeightedBSliceBoundaryDeblock(t *testing.T) {
	for _, chromaFormatIDC := range []uint32{2, 3} {
		for _, shape := range []struct {
			name     string
			mbType   uint32
			cbp      int
			cbpTable int
		}{
			{name: "l0", mbType: MBType16x16 | MBTypeP0L0, cbp: 0xf, cbpTable: 0xf},
			{name: "l1", mbType: MBType16x16 | MBTypeP0L1, cbp: 0xf, cbpTable: 0xf},
			{name: "bi", mbType: MBType16x16 | MBTypeP0L0 | MBTypeP0L1, cbp: 0xf, cbpTable: 0xf},
			{name: "temporal-direct", mbType: MBType16x16 | MBTypeL0L1 | MBTypeDirect2, cbp: 0xf, cbpTable: 0xf},
			{name: "spatial-direct-skip", mbType: MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2 | MBTypeSkip},
		} {
			for _, cabac := range []int32{0, 1} {
				for _, weight := range []struct {
					name            string
					useWeight       int32
					useWeightChroma int32
				}{
					{name: "serialized-default"},
					{name: "luma", useWeight: 1},
					{name: "chroma", useWeightChroma: 1},
					{name: "luma-chroma", useWeight: 1, useWeightChroma: 1},
				} {
					t.Run(fmt.Sprintf("%s/%s/cabac%d/%s", chromaFormatName(int(chromaFormatIDC)), shape.name, cabac, weight.name), func(t *testing.T) {
						sh := &SliceHeader{
							SliceTypeNoS:     PictureTypeB,
							PictureStructure: PictureFrame,
							DeblockingFilter: 2,
							SPS: &SPS{
								BitDepthLuma:    9,
								ChromaFormatIDC: chromaFormatIDC,
							},
							PPS: &PPS{
								CABAC:             cabac,
								WeightedBipredIDC: 1,
							},
							PredWeightTable: PredWeightTable{
								UseWeight:       weight.useWeight,
								UseWeightChroma: weight.useWeightChroma,
							},
						}
						if err := validateHighFrameSliceMacroblockForReconstructWithSubMB(sh, shape.mbType, nil, shape.cbp, shape.cbpTable); err != nil {
							t.Fatalf("validate high9 chroma explicit weighted-B mode-2 deblock err = %v, want nil", err)
						}
					})
				}
			}
		}
	}
}

func TestValidateHighFrameSliceMacroblockForReconstructAllowsHigh10ChromaWeightedBSliceBoundaryDeblock(t *testing.T) {
	for _, chromaFormatIDC := range []uint32{2, 3} {
		for _, shape := range []struct {
			name   string
			mbType uint32
		}{
			{name: "l0", mbType: MBType16x16 | MBTypeP0L0},
			{name: "l1", mbType: MBType16x16 | MBTypeP0L1},
			{name: "bi", mbType: MBType16x16 | MBTypeP0L0 | MBTypeP0L1},
		} {
			for _, tt := range []struct {
				name            string
				pps             *PPS
				useWeight       int32
				useWeightChroma int32
				cbpTable        int
			}{
				{name: "cavlc implicit", pps: &PPS{WeightedBipredIDC: 2}, useWeight: 2, useWeightChroma: 2, cbpTable: 0xf00f},
				{name: "cabac implicit", pps: &PPS{CABAC: 1, WeightedBipredIDC: 2}, useWeight: 2, useWeightChroma: 2, cbpTable: 0xf},
				{name: "cavlc explicit", pps: &PPS{WeightedBipredIDC: 1}, useWeight: 1, useWeightChroma: 1, cbpTable: 0xf00f},
				{name: "cabac explicit", pps: &PPS{CABAC: 1, WeightedBipredIDC: 1}, useWeight: 1, useWeightChroma: 1, cbpTable: 0xf},
			} {
				t.Run(fmt.Sprintf("%s/%s/%s", chromaFormatName(int(chromaFormatIDC)), shape.name, tt.name), func(t *testing.T) {
					sh := &SliceHeader{
						SliceTypeNoS:     PictureTypeB,
						PictureStructure: PictureFrame,
						DeblockingFilter: 2,
						SPS: &SPS{
							BitDepthLuma:    10,
							ChromaFormatIDC: chromaFormatIDC,
						},
						PPS: tt.pps,
						PredWeightTable: PredWeightTable{
							UseWeight:       tt.useWeight,
							UseWeightChroma: tt.useWeightChroma,
						},
					}
					if err := validateHighFrameSliceMacroblockForReconstructWithSubMB(sh, shape.mbType, nil, 0xf, tt.cbpTable); err != nil {
						t.Fatalf("validate high10 chroma weighted-B mode-2 deblock err = %v, want nil", err)
					}
				})
			}
		}
	}
}
