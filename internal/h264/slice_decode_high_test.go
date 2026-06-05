// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

import (
	"fmt"
	"strings"
	"testing"
)

func TestSimpleFrameSliceDecodeBitDepthGate(t *testing.T) {
	for _, tc := range []struct {
		bitDepth int32
		want     bool
	}{
		{bitDepth: 8, want: true},
		{bitDepth: 9},
		{bitDepth: 10},
		{bitDepth: 12},
		{bitDepth: 14},
	} {
		if got := h264SimpleFrameSliceDecodeSupportsBitDepth(tc.bitDepth); got != tc.want {
			t.Fatalf("supports bit depth %d = %v, want %v", tc.bitDepth, got, tc.want)
		}
	}
}

func TestValidateSimpleFrameSliceDecodeAllows8Bit(t *testing.T) {
	m, dst, sh := simpleFrameSliceDecodeBitDepthFixture(t, 8)

	if err := validateSimpleFrameSliceDecodeInputs(m, dst, sh, 4); err != nil {
		t.Fatalf("8-bit validation err = %v, want nil", err)
	}
}

func TestValidateSimpleFrameSliceDecodeRejectsHighBitDepths(t *testing.T) {
	for _, bitDepth := range []int32{9, 10, 12, 14} {
		t.Run(bitDepthName(bitDepth), func(t *testing.T) {
			m, dst, sh := simpleFrameSliceDecodeBitDepthFixture(t, bitDepth)

			if err := validateSimpleFrameSliceDecodeInputs(m, dst, sh, 4); err != ErrUnsupported {
				t.Fatalf("validation err = %v, want ErrUnsupported", err)
			}
		})
	}
}

func TestDecodeCAVLCFrameSliceRejectsHighBitDepthsAtValidation(t *testing.T) {
	for _, bitDepth := range []int32{9, 10, 12, 14} {
		t.Run(bitDepthName(bitDepth), func(t *testing.T) {
			m, dst, sh := simpleFrameSliceDecodeBitDepthFixture(t, bitDepth)
			gb := newBitReader(nil)

			_, err := m.decodeCAVLCFrameSlice(&gb, dst, sh, h264FrameSliceDecodeInput{SliceNum: 4})
			if err != ErrUnsupported {
				t.Fatalf("decode err = %v, want ErrUnsupported", err)
			}
			if gb.bitPos != 0 {
				t.Fatalf("bit reader consumed %d bits, want 0", gb.bitPos)
			}
			if m.MacroblockTyp[0] != 0 || m.SliceTable[0] != ^uint16(0) {
				t.Fatalf("tables type/slice = %#x/%#x, want untouched", m.MacroblockTyp[0], m.SliceTable[0])
			}
		})
	}
}

func TestDecodeCABACFrameSliceRejectsHighBitDepthsAtValidation(t *testing.T) {
	for _, bitDepth := range []int32{9, 10, 12, 14} {
		t.Run(bitDepthName(bitDepth), func(t *testing.T) {
			m, dst, sh := simpleFrameSliceDecodeBitDepthFixture(t, bitDepth)
			src := &scriptedCABACSource{}

			_, err := m.decodeCABACFrameSlice(src, dst, sh, h264FrameSliceDecodeInput{SliceNum: 4})
			if err != ErrUnsupported {
				t.Fatalf("decode err = %v, want ErrUnsupported", err)
			}
			if len(src.indexes) != 0 || len(src.pcmReadSizes) != 0 || len(src.terms) != 0 {
				t.Fatalf("cabac source was touched: indexes=%v pcmReads=%v terms=%v", src.indexes, src.pcmReadSizes, src.terms)
			}
			if m.MacroblockTyp[0] != 0 || m.SliceTable[0] != ^uint16(0) {
				t.Fatalf("tables type/slice = %#x/%#x, want untouched", m.MacroblockTyp[0], m.SliceTable[0])
			}
		})
	}
}

func TestValidateSimpleFrameSliceDecodeHighAllowsHigh10Intra420(t *testing.T) {
	m, dst, sh := highFrameSliceDecodeFixture(t, 10, 1, false, PictureTypeI)

	if err := validateSimpleFrameSliceDecodeInputsHigh(m, dst, sh, 4); err != nil {
		t.Fatalf("high validation err = %v, want nil", err)
	}
}

func TestValidateSimpleFrameSliceDecodeHighAllowsHigh12Intra420SliceScope(t *testing.T) {
	m, dst, sh := highFrameSliceDecodeFixture(t, 12, 1, false, PictureTypeI)

	if err := validateSimpleFrameSliceDecodeInputsHigh(m, dst, sh, 4); err != nil {
		t.Fatalf("high12 validation err = %v, want nil", err)
	}
}

func TestValidateSimpleFrameSliceDecodeHighAllowsHigh14CAVLCIntra420SliceScope(t *testing.T) {
	m, dst, sh := highFrameSliceDecodeFixture(t, 14, 1, false, PictureTypeI)

	if err := validateSimpleFrameSliceDecodeInputsHigh(m, dst, sh, 4); err != nil {
		t.Fatalf("high14 validation err = %v, want nil", err)
	}
}

func TestValidateSimpleFrameSliceDecodeHighAllowsHigh10High12AndHigh14P420NoWeight(t *testing.T) {
	for _, bitDepth := range []int32{10, 12, 14} {
		t.Run(bitDepthName(bitDepth), func(t *testing.T) {
			m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, bitDepth, 1, 1, false, PictureTypeP)
			sh.RefCount = [2]uint32{1, 0}

			if err := validateSimpleFrameSliceDecodeInputsHigh(m, dst, sh, 4); err != nil {
				t.Fatalf("high P validation err = %v, want nil", err)
			}
		})
	}
}

func TestValidateSimpleFrameSliceDecodeHighAllowsHigh10B420NoWeight(t *testing.T) {
	m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, 10, 1, 1, false, PictureTypeB)
	sh.RefCount = [2]uint32{1, 1}

	if err := validateSimpleFrameSliceDecodeInputsHigh(m, dst, sh, 4); err != nil {
		t.Fatalf("high B validation err = %v, want nil", err)
	}
}

func TestValidateSimpleFrameSliceDecodeHighAllowsFrameMBAFFGeometry(t *testing.T) {
	m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, 8, 1, 1, false, PictureTypeI)
	sh.SPS.FrameMBSOnlyFlag = 0
	sh.SPS.MBAFF = 1
	sh.PictureStructure = PictureFrame

	if err := validateSimpleFrameSliceDecodeInputsHigh(m, dst, sh, 4); err != nil {
		t.Fatalf("frame-MBAFF validation err = %v, want nil", err)
	}

	sh.PictureStructure = PictureTopField
	if err := validateSimpleFrameSliceDecodeInputsHigh(m, dst, sh, 4); err != ErrUnsupported {
		t.Fatalf("8-bit field validation err = %v, want ErrUnsupported", err)
	}
}

func TestValidateSimpleFrameSliceDecodeHighAllowsHigh10Chroma422FieldPictures(t *testing.T) {
	for _, tt := range []struct {
		name             string
		pictureStructure int32
		sliceType        int32
		implicitWeighted bool
	}{
		{name: "top/I", pictureStructure: PictureTopField, sliceType: PictureTypeI},
		{name: "bottom/I", pictureStructure: PictureBottomField, sliceType: PictureTypeI},
		{name: "top/P", pictureStructure: PictureTopField, sliceType: PictureTypeP},
		{name: "bottom/P", pictureStructure: PictureBottomField, sliceType: PictureTypeP},
		{name: "top/B", pictureStructure: PictureTopField, sliceType: PictureTypeB},
		{name: "bottom/B", pictureStructure: PictureBottomField, sliceType: PictureTypeB},
		{name: "top/B-implicit-weight", pictureStructure: PictureTopField, sliceType: PictureTypeB, implicitWeighted: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, 10, 2, 2, true, tt.sliceType)
			sh.SPS.FrameMBSOnlyFlag = 0
			sh.SPS.MBAFF = 1
			sh.PictureStructure = tt.pictureStructure
			if tt.sliceType == PictureTypeP {
				sh.RefCount = [2]uint32{1, 0}
			} else if tt.sliceType == PictureTypeB {
				sh.RefCount = [2]uint32{1, 1}
			}
			if tt.implicitWeighted {
				sh.PPS.WeightedBipredIDC = 2
				sh.PredWeightTable.UseWeight = 2
				sh.PredWeightTable.UseWeightChroma = 2
			}

			if err := validateSimpleFrameSliceDecodeInputsHigh(m, dst, sh, 4); err != nil {
				t.Fatalf("high10 422 field validation err = %v, want nil", err)
			}
		})
	}
}

func TestValidateSimpleFrameSliceDecodeHighAllowsHigh10Chroma422FieldExplicitWeightedB(t *testing.T) {
	for _, tt := range []struct {
		name             string
		pictureStructure int32
		cabac            bool
		deblockMode      int32
		useWeight        int32
		useWeightChroma  int32
	}{
		{name: "top/cavlc/mode0/luma", pictureStructure: PictureTopField, useWeight: 1},
		{name: "top/cavlc/mode0/chroma", pictureStructure: PictureTopField, useWeightChroma: 1},
		{name: "top/cavlc/mode0/luma-chroma", pictureStructure: PictureTopField, useWeight: 1, useWeightChroma: 1},
		{name: "bottom/cavlc/mode0/luma", pictureStructure: PictureBottomField, useWeight: 1},
		{name: "bottom/cavlc/mode0/chroma", pictureStructure: PictureBottomField, useWeightChroma: 1},
		{name: "bottom/cavlc/mode0/luma-chroma", pictureStructure: PictureBottomField, useWeight: 1, useWeightChroma: 1},
		{name: "top/cabac/mode0/luma", pictureStructure: PictureTopField, cabac: true, useWeight: 1},
		{name: "top/cabac/mode0/chroma", pictureStructure: PictureTopField, cabac: true, useWeightChroma: 1},
		{name: "top/cabac/mode0/luma-chroma", pictureStructure: PictureTopField, cabac: true, useWeight: 1, useWeightChroma: 1},
		{name: "bottom/cabac/mode0/luma", pictureStructure: PictureBottomField, cabac: true, useWeight: 1},
		{name: "bottom/cabac/mode0/chroma", pictureStructure: PictureBottomField, cabac: true, useWeightChroma: 1},
		{name: "bottom/cabac/mode0/luma-chroma", pictureStructure: PictureBottomField, cabac: true, useWeight: 1, useWeightChroma: 1},
		{name: "top/cavlc/mode1/luma", pictureStructure: PictureTopField, deblockMode: 1, useWeight: 1},
		{name: "top/cavlc/mode1/chroma", pictureStructure: PictureTopField, deblockMode: 1, useWeightChroma: 1},
		{name: "top/cavlc/mode1/luma-chroma", pictureStructure: PictureTopField, deblockMode: 1, useWeight: 1, useWeightChroma: 1},
		{name: "bottom/cavlc/mode1/luma", pictureStructure: PictureBottomField, deblockMode: 1, useWeight: 1},
		{name: "bottom/cavlc/mode1/chroma", pictureStructure: PictureBottomField, deblockMode: 1, useWeightChroma: 1},
		{name: "bottom/cavlc/mode1/luma-chroma", pictureStructure: PictureBottomField, deblockMode: 1, useWeight: 1, useWeightChroma: 1},
		{name: "top/cabac/mode1/luma", pictureStructure: PictureTopField, cabac: true, deblockMode: 1, useWeight: 1},
		{name: "top/cabac/mode1/chroma", pictureStructure: PictureTopField, cabac: true, deblockMode: 1, useWeightChroma: 1},
		{name: "top/cabac/mode1/luma-chroma", pictureStructure: PictureTopField, cabac: true, deblockMode: 1, useWeight: 1, useWeightChroma: 1},
		{name: "bottom/cabac/mode1/luma", pictureStructure: PictureBottomField, cabac: true, deblockMode: 1, useWeight: 1},
		{name: "bottom/cabac/mode1/chroma", pictureStructure: PictureBottomField, cabac: true, deblockMode: 1, useWeightChroma: 1},
		{name: "bottom/cabac/mode1/luma-chroma", pictureStructure: PictureBottomField, cabac: true, deblockMode: 1, useWeight: 1, useWeightChroma: 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, 10, 2, 2, tt.deblockMode != 0, PictureTypeB)
			sh.SPS.FrameMBSOnlyFlag = 0
			sh.SPS.MBAFF = 1
			sh.PictureStructure = tt.pictureStructure
			sh.RefCount = [2]uint32{1, 1}
			if tt.cabac {
				sh.PPS.CABAC = 1
			}
			sh.PPS.WeightedBipredIDC = 1
			sh.PredWeightTable.UseWeight = tt.useWeight
			sh.PredWeightTable.UseWeightChroma = tt.useWeightChroma

			if err := validateSimpleFrameSliceDecodeInputsHigh(m, dst, sh, 4); err != nil {
				t.Fatalf("high10 422 explicit weighted B field validation err = %v, want nil", err)
			}
			if err := validateSimpleFrameSliceDecodeInputHighRefs(sh, h264FrameSliceDecodeInputHigh{PredWeight: &sh.PredWeightTable}); err != nil {
				t.Fatalf("high10 422 explicit weighted B field ref validation err = %v, want nil", err)
			}
		})
	}
}

func TestValidateSimpleFrameSliceDecodeHighAllowsHigh10Chroma444FieldWeightedB(t *testing.T) {
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
	for _, picture := range []struct {
		name      string
		structure int32
	}{
		{name: "top", structure: PictureTopField},
		{name: "bottom", structure: PictureBottomField},
	} {
		for _, cabac := range []bool{false, true} {
			for _, deblockMode := range []int32{0, 1} {
				for _, weight := range weights {
					entropy := "cavlc"
					if cabac {
						entropy = "cabac"
					}
					name := fmt.Sprintf("%s/%s/mode%d/%s", picture.name, entropy, deblockMode, weight.name)
					t.Run(name, func(t *testing.T) {
						m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, 10, 3, 2, deblockMode != 0, PictureTypeB)
						sh.SPS.FrameMBSOnlyFlag = 0
						sh.SPS.MBAFF = 1
						sh.PictureStructure = picture.structure
						sh.RefCount = [2]uint32{1, 1}
						if cabac {
							sh.PPS.CABAC = 1
						}
						sh.PPS.WeightedBipredIDC = weight.weightedBipredID
						sh.PredWeightTable.UseWeight = weight.useWeight
						sh.PredWeightTable.UseWeightChroma = weight.useWeightChroma

						if err := validateSimpleFrameSliceDecodeInputsHigh(m, dst, sh, 4); err != nil {
							t.Fatalf("high10 444 weighted B field validation err = %v, want nil", err)
						}
						if err := validateSimpleFrameSliceDecodeInputHighRefs(sh, h264FrameSliceDecodeInputHigh{PredWeight: &sh.PredWeightTable}); err != nil {
							t.Fatalf("high10 444 weighted B field ref validation err = %v, want nil", err)
						}
					})
				}
			}
		}
	}
}

func TestValidateSimpleFrameSliceDecodeHighAllowsHigh10ChromaFieldWeightedP(t *testing.T) {
	weights := []struct {
		name  string
		table func(chromaFormatIDC int) PredWeightTable
	}{
		{name: "luma-only", table: func(int) PredWeightTable {
			pwt := highWeightedPPredWeightTable()
			pwt.UseWeightChroma = 0
			return pwt
		}},
		{name: "luma-chroma", table: func(int) PredWeightTable {
			return highWeightedPPredWeightTable()
		}},
		{name: "source-chroma-only", table: highSourceChromaOnlyWeightedPPredWeightTable},
	}
	for _, chromaFormatIDC := range []int{2, 3} {
		for _, picture := range []struct {
			name      string
			structure int32
		}{
			{name: "top", structure: PictureTopField},
			{name: "bottom", structure: PictureBottomField},
		} {
			for _, cabac := range []bool{false, true} {
				for _, deblockMode := range []int32{0, 1} {
					for _, weight := range weights {
						entropy := "cavlc"
						if cabac {
							entropy = "cabac"
						}
						name := fmt.Sprintf("%s/%s/%s/mode%d/%s", chromaFormatName(chromaFormatIDC), picture.name, entropy, deblockMode, weight.name)
						t.Run(name, func(t *testing.T) {
							m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, 10, chromaFormatIDC, 2, deblockMode != 0, PictureTypeP)
							sh.SPS.FrameMBSOnlyFlag = 0
							sh.SPS.MBAFF = 1
							sh.PictureStructure = picture.structure
							sh.RefCount = [2]uint32{1, 0}
							if cabac {
								sh.PPS.CABAC = 1
							}
							sh.PPS.WeightedPred = 1
							sh.PredWeightTable = weight.table(chromaFormatIDC)

							if err := validateSimpleFrameSliceDecodeInputsHigh(m, dst, sh, 4); err != nil {
								t.Fatalf("high10 chroma weighted P field validation err = %v, want nil", err)
							}
						})
					}
				}
			}
		}
	}
}

func TestValidateSimpleFrameSliceDecodeHighRejectsUnprovedHigh10FieldPictures(t *testing.T) {
	for _, tt := range []struct {
		name        string
		chroma      int
		sliceType   int32
		deblockMode int32
		run         func(*SliceHeader)
	}{
		{name: "420/I", chroma: 1, sliceType: PictureTypeI},
		{name: "422/slice-boundary", chroma: 2, sliceType: PictureTypeI, deblockMode: 2},
		{name: "444/unweighted-B", chroma: 3, sliceType: PictureTypeB},
		{name: "444/unweighted-P", chroma: 3, sliceType: PictureTypeP},
		{name: "444/unnormalized-chroma-only-weighted-P", chroma: 3, sliceType: PictureTypeP, run: func(sh *SliceHeader) {
			sh.PPS.WeightedPred = 1
			sh.PredWeightTable = highSourceChromaOnlyWeightedPPredWeightTable(3)
			sh.PredWeightTable.UseWeight = 0
		}},
		{name: "444/I", chroma: 3, sliceType: PictureTypeI},
	} {
		t.Run(tt.name, func(t *testing.T) {
			m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, 10, tt.chroma, 2, true, tt.sliceType)
			sh.SPS.FrameMBSOnlyFlag = 0
			sh.SPS.MBAFF = 1
			sh.PictureStructure = PictureTopField
			if tt.deblockMode != 0 {
				sh.DeblockingFilter = tt.deblockMode
			}
			if tt.sliceType == PictureTypeB {
				sh.RefCount = [2]uint32{1, 1}
			}
			if tt.run != nil {
				tt.run(sh)
			}

			if err := validateSimpleFrameSliceDecodeInputsHigh(m, dst, sh, 4); err != ErrUnsupported {
				t.Fatalf("high10 unproved field validation err = %v, want ErrUnsupported", err)
			}
		})
	}
}

func TestH264FrameMBAFFReconstructViewHighKeepsFrameCodedMacroblocksInFrameView(t *testing.T) {
	dst := makeH264SliceDecodePictureHigh(1, 4, 1)
	ref := makeH264SliceDecodePictureHigh(1, 4, 1)
	refs := [2][]*h264PicturePlanesHigh{{ref}}
	cur := sliceMacroblockCursor{FrameMBAFF: true, MBY: 2, PixelMBY: 2}
	var refPlanes [2][32]h264PicturePlanesHigh
	var refPtrs [2][32]*h264PicturePlanesHigh

	view, mbY, gotRefs, err := h264FrameMBAFFReconstructViewHigh(dst, cur, MBTypeIntra4x4, refs, &refPlanes, &refPtrs)
	if err != nil {
		t.Fatal(err)
	}
	if view.PictureStructure != PictureFrame || view.LumaStride != dst.LumaStride || view.ChromaStride != dst.ChromaStride || view.MBHeight != dst.MBHeight || mbY != cur.PixelMBY {
		t.Fatalf("frame-coded high view picture/stride/chroma/height/mbY = %d/%d/%d/%d/%d, want %d/%d/%d/%d/%d",
			view.PictureStructure, view.LumaStride, view.ChromaStride, view.MBHeight, mbY,
			PictureFrame, dst.LumaStride, dst.ChromaStride, dst.MBHeight, cur.PixelMBY)
	}
	if len(gotRefs[0]) != 1 || gotRefs[0][0] != ref {
		t.Fatalf("frame-coded high refs = %#v, want original ref", gotRefs[0])
	}
}

func TestH264FrameMBAFFReconstructViewHighMapsBottomFieldDestinationAndRefs(t *testing.T) {
	dst := makeH264SliceDecodePictureHigh(1, 4, 1)
	ref0 := makeH264SliceDecodePictureHigh(1, 4, 1)
	ref1 := makeH264SliceDecodePictureHigh(1, 4, 1)
	refs := [2][]*h264PicturePlanesHigh{{ref0, ref1}}
	cur := sliceMacroblockCursor{FrameMBAFF: true, MBY: 1, PixelMBY: 1}
	var refPlanes [2][32]h264PicturePlanesHigh
	var refPtrs [2][32]*h264PicturePlanesHigh

	view, mbY, gotRefs, err := h264FrameMBAFFReconstructViewHigh(dst, cur, MBTypeInterlaced|MBType16x16|MBTypeP0L0, refs, &refPlanes, &refPtrs)
	if err != nil {
		t.Fatal(err)
	}
	if view.PictureStructure != PictureBottomField || view.LumaStride != dst.LumaStride*2 || view.ChromaStride != dst.ChromaStride*2 || view.MBHeight != 2 || mbY != 0 {
		t.Fatalf("bottom high field view picture/stride/chroma/height/mbY = %d/%d/%d/%d/%d",
			view.PictureStructure, view.LumaStride, view.ChromaStride, view.MBHeight, mbY)
	}
	if &view.Y[0] != &dst.Y[dst.LumaStride] || &view.Cb[0] != &dst.Cb[dst.ChromaStride] || &view.Cr[0] != &dst.Cr[dst.ChromaStride] {
		t.Fatalf("bottom high field view does not start on the second frame line")
	}
	if len(gotRefs[0]) != 4 {
		t.Fatalf("high field refs len = %d, want 4", len(gotRefs[0]))
	}
	if gotRefs[0][0].PictureStructure != PictureBottomField || &gotRefs[0][0].Y[0] != &ref0.Y[ref0.LumaStride] {
		t.Fatalf("high ref0 maps to bottom field of frame 0")
	}
	if gotRefs[0][1].PictureStructure != PictureTopField || &gotRefs[0][1].Y[0] != &ref0.Y[0] {
		t.Fatalf("high ref1 maps to top field of frame 0")
	}
	if gotRefs[0][2].PictureStructure != PictureBottomField || &gotRefs[0][2].Y[0] != &ref1.Y[ref1.LumaStride] {
		t.Fatalf("high ref2 maps to bottom field of frame 1")
	}
	if gotRefs[0][3].PictureStructure != PictureTopField || &gotRefs[0][3].Y[0] != &ref1.Y[0] {
		t.Fatalf("high ref3 maps to top field of frame 1")
	}
}

func TestValidateSimpleFrameSliceDecodeHighRejectsStagedBoundaries(t *testing.T) {
	tests := []struct {
		name        string
		bitDepth    int32
		chroma      int32
		format      int
		deblock     bool
		deblockMode int32
		slice       int32
	}{
		{name: "8-bit", bitDepth: 8, chroma: 8, format: 1, slice: PictureTypeI},
		{name: "9-bit-422-slice-boundary-deblock", bitDepth: 9, chroma: 9, format: 2, deblockMode: 2, slice: PictureTypeI},
		{name: "unequal-depth", bitDepth: 10, chroma: 12, format: 1, slice: PictureTypeI},
		{name: "monochrome", bitDepth: 10, chroma: 10, format: 0, slice: PictureTypeI},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, dst, sh := highFrameSliceDecodeFixture(t, tt.bitDepth, tt.format, tt.deblock, tt.slice)
			if tt.deblockMode != 0 {
				sh.DeblockingFilter = tt.deblockMode
			}
			sh.SPS.BitDepthChroma = tt.chroma

			if err := validateSimpleFrameSliceDecodeInputsHigh(m, dst, sh, 4); err != ErrUnsupported {
				t.Fatalf("high validation err = %v, want ErrUnsupported", err)
			}
		})
	}
}

func TestValidateSimpleFrameSliceDecodeHighAllowsHigh14CAVLCDeblocking(t *testing.T) {
	for _, deblockMode := range []int32{1, 2} {
		for _, sliceType := range []int32{PictureTypeI, PictureTypeP} {
			t.Run(fmt.Sprintf("mode%d/%s", deblockMode, pictureTypeName(sliceType)), func(t *testing.T) {
				m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, 14, 1, 1, deblockMode != 0, sliceType)
				sh.DeblockingFilter = deblockMode
				if sliceType == PictureTypeP {
					sh.RefCount = [2]uint32{1, 0}
				}

				if err := validateSimpleFrameSliceDecodeInputsHigh(m, dst, sh, 4); err != nil {
					t.Fatalf("high14 CAVLC mode-%d deblock validation err = %v, want nil", deblockMode, err)
				}
			})
		}
	}
}

func TestValidateSimpleFrameSliceDecodeHighAllowsHigh12High14ExplicitWeightedBMode2Deblock(t *testing.T) {
	for _, bitDepth := range []int32{12, 14} {
		for _, deblockMode := range []int32{2} {
			for _, tt := range []struct {
				name      string
				sliceType int32
				run       func(*SliceHeader)
			}{
				{
					name:      "weighted-CAVLC-B",
					sliceType: PictureTypeB,
					run: func(sh *SliceHeader) {
						sh.PPS.WeightedBipredIDC = 1
						sh.PredWeightTable.UseWeight = 1
						sh.PredWeightTable.UseWeightChroma = 1
						sh.RefCount = [2]uint32{1, 1}
					},
				},
				{
					name:      "weighted-CABAC-B",
					sliceType: PictureTypeB,
					run: func(sh *SliceHeader) {
						sh.PPS.CABAC = 1
						sh.PPS.WeightedBipredIDC = 1
						sh.PredWeightTable.UseWeight = 1
						sh.PredWeightTable.UseWeightChroma = 1
						sh.RefCount = [2]uint32{1, 1}
					},
				},
			} {
				t.Run(fmt.Sprintf("%s/mode%d/%s", bitDepthName(bitDepth), deblockMode, tt.name), func(t *testing.T) {
					m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, bitDepth, 1, 1, true, tt.sliceType)
					sh.DeblockingFilter = deblockMode
					tt.run(sh)

					if err := validateSimpleFrameSliceDecodeInputsHigh(m, dst, sh, 4); err != nil {
						t.Fatalf("high%d weighted B mode-%d deblock validation err = %v, want nil", bitDepth, deblockMode, err)
					}
				})
			}
		}
	}
}

func TestValidateSimpleFrameSliceDecodeHighAllowsHigh12High14CAVLCBNoDeblockAndDeblocking(t *testing.T) {
	for _, bitDepth := range []int32{12, 14} {
		for _, deblockMode := range []int32{0, 1, 2} {
			t.Run(fmt.Sprintf("%s/mode%d/B", bitDepthName(bitDepth), deblockMode), func(t *testing.T) {
				m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, bitDepth, 1, 1, deblockMode != 0, PictureTypeB)
				sh.DeblockingFilter = deblockMode
				sh.RefCount = [2]uint32{1, 1}

				if err := validateSimpleFrameSliceDecodeInputsHigh(m, dst, sh, 4); err != nil {
					t.Fatalf("high%d CAVLC B mode-%d validation err = %v, want nil", bitDepth, deblockMode, err)
				}
			})
		}
	}
}

func TestValidateSimpleFrameSliceDecodeHighAllowsHigh14CABACBNoDeblockAndDeblocking(t *testing.T) {
	for _, deblockMode := range []int32{0, 1, 2} {
		t.Run(fmt.Sprintf("mode%d/B", deblockMode), func(t *testing.T) {
			m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, 14, 1, 1, deblockMode != 0, PictureTypeB)
			sh.PPS.CABAC = 1
			sh.DeblockingFilter = deblockMode
			sh.RefCount = [2]uint32{1, 1}

			if err := validateSimpleFrameSliceDecodeInputsHigh(m, dst, sh, 4); err != nil {
				t.Fatalf("high14 CABAC B mode-%d validation err = %v, want nil", deblockMode, err)
			}
		})
	}
}

func TestValidateSimpleFrameSliceDecodeHighAllowsHigh12CABACBNoDeblockAndDeblocking(t *testing.T) {
	for _, deblockMode := range []int32{0, 1, 2} {
		t.Run(fmt.Sprintf("mode%d/B", deblockMode), func(t *testing.T) {
			m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, 12, 1, 1, deblockMode != 0, PictureTypeB)
			sh.PPS.CABAC = 1
			sh.DeblockingFilter = deblockMode
			sh.RefCount = [2]uint32{1, 1}

			if err := validateSimpleFrameSliceDecodeInputsHigh(m, dst, sh, 4); err != nil {
				t.Fatalf("high12 CABAC B mode-%d validation err = %v, want nil", deblockMode, err)
			}
		})
	}
}

func TestValidateSimpleFrameSliceDecodeHighAllowsHigh14CABACDeblockingIP(t *testing.T) {
	for _, deblockMode := range []int32{1, 2} {
		for _, sliceType := range []int32{PictureTypeI, PictureTypeP} {
			t.Run(fmt.Sprintf("mode%d/%s", deblockMode, pictureTypeName(sliceType)), func(t *testing.T) {
				m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, 14, 1, 1, true, sliceType)
				sh.PPS.CABAC = 1
				sh.DeblockingFilter = deblockMode
				if sliceType == PictureTypeP {
					sh.RefCount = [2]uint32{1, 0}
				}

				if err := validateSimpleFrameSliceDecodeInputsHigh(m, dst, sh, 4); err != nil {
					t.Fatalf("high14 CABAC mode-%d deblock validation err = %v, want nil", deblockMode, err)
				}
			})
		}

		t.Run(fmt.Sprintf("mode%d/weighted-P", deblockMode), func(t *testing.T) {
			m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, 14, 1, 1, true, PictureTypeP)
			sh.PPS.CABAC = 1
			sh.PPS.WeightedPred = 1
			sh.DeblockingFilter = deblockMode
			sh.RefCount = [2]uint32{1, 0}
			sh.PredWeightTable = highWeightedPPredWeightTable()

			if err := validateSimpleFrameSliceDecodeInputsHigh(m, dst, sh, 4); err != nil {
				t.Fatalf("high14 CABAC weighted P mode-%d deblock validation err = %v, want nil", deblockMode, err)
			}
		})
	}
}

func TestValidateSimpleFrameSliceDecodeHighAllowsHigh9Frame420And422(t *testing.T) {
	for _, tt := range []struct {
		name             string
		chromaFormatIDC  int
		sliceType        int32
		implicitWeighted bool
	}{
		{name: "420/I", chromaFormatIDC: 1, sliceType: PictureTypeI},
		{name: "420/P", chromaFormatIDC: 1, sliceType: PictureTypeP},
		{name: "420/B", chromaFormatIDC: 1, sliceType: PictureTypeB},
		{name: "422/I", chromaFormatIDC: 2, sliceType: PictureTypeI},
		{name: "422/P", chromaFormatIDC: 2, sliceType: PictureTypeP},
		{name: "422/B", chromaFormatIDC: 2, sliceType: PictureTypeB},
		{name: "422/B-implicit-weight", chromaFormatIDC: 2, sliceType: PictureTypeB, implicitWeighted: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, 9, tt.chromaFormatIDC, 2, true, tt.sliceType)
			sh.DeblockingFilter = 1
			switch tt.sliceType {
			case PictureTypeP:
				sh.RefCount = [2]uint32{1, 0}
			case PictureTypeB:
				sh.RefCount = [2]uint32{1, 1}
			}
			if tt.implicitWeighted {
				sh.PPS.WeightedBipredIDC = 2
				sh.PredWeightTable.UseWeight = 2
				sh.PredWeightTable.UseWeightChroma = 2
			}

			if err := validateSimpleFrameSliceDecodeInputsHigh(m, dst, sh, 4); err != nil {
				t.Fatalf("high9 validation err = %v, want nil", err)
			}
		})
	}
}

func TestValidateSimpleFrameSliceDecodeHighAllowsHigh14CAVLCWeightedP(t *testing.T) {
	for _, deblockMode := range []int32{0, 1, 2} {
		t.Run(fmt.Sprintf("deblock-mode-%d", deblockMode), func(t *testing.T) {
			m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, 14, 1, 1, deblockMode != 0, PictureTypeP)
			sh.DeblockingFilter = deblockMode
			sh.RefCount = [2]uint32{1, 0}
			sh.PPS.WeightedPred = 1
			sh.PredWeightTable = highWeightedPPredWeightTable()

			if err := validateSimpleFrameSliceDecodeInputsHigh(m, dst, sh, 4); err != nil {
				t.Fatalf("high14 weighted P validation err = %v, want nil", err)
			}
		})
	}
}

func TestValidateSimpleFrameSliceDecodeHighAllowsHigh14CABACNoDeblockIP(t *testing.T) {
	for _, tt := range []struct {
		name string
		run  func(*SliceHeader)
	}{
		{name: "I"},
		{
			name: "P",
			run: func(sh *SliceHeader) {
				sh.RefCount = [2]uint32{1, 0}
			},
		},
		{
			name: "weighted-P",
			run: func(sh *SliceHeader) {
				sh.SliceType = PictureTypeP
				sh.SliceTypeNoS = PictureTypeP
				sh.RefCount = [2]uint32{1, 0}
				sh.PPS.WeightedPred = 1
				sh.PredWeightTable = highWeightedPPredWeightTable()
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			sliceType := PictureTypeI
			if tt.name != "I" {
				sliceType = PictureTypeP
			}
			m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, 14, 1, 1, false, sliceType)
			sh.PPS.CABAC = 1
			if tt.run != nil {
				tt.run(sh)
			}

			if err := validateSimpleFrameSliceDecodeInputsHigh(m, dst, sh, 4); err != nil {
				t.Fatalf("high14 CABAC validation err = %v, want nil", err)
			}
		})
	}
}

func TestValidateSimpleFrameSliceDecodeHighAllowsHigh10AndHigh12Deblocking(t *testing.T) {
	for _, bitDepth := range []int32{10, 12} {
		for _, sliceType := range []int32{PictureTypeI, PictureTypeP} {
			t.Run(bitDepthName(bitDepth)+"/"+pictureTypeName(sliceType), func(t *testing.T) {
				m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, bitDepth, 1, 1, true, sliceType)
				if sliceType == PictureTypeP {
					sh.RefCount = [2]uint32{1, 0}
				}

				if err := validateSimpleFrameSliceDecodeInputsHigh(m, dst, sh, 4); err != nil {
					t.Fatalf("high deblock validation err = %v, want nil", err)
				}
			})
		}
	}
}

func TestValidateSimpleFrameSliceDecodeHighAllowsHigh10BDeblockingAtSliceLevel(t *testing.T) {
	m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, 10, 1, 1, true, PictureTypeB)
	sh.RefCount = [2]uint32{1, 1}

	if err := validateSimpleFrameSliceDecodeInputsHigh(m, dst, sh, 4); err != nil {
		t.Fatalf("high B deblock slice validation err = %v, want nil", err)
	}
}

func TestValidateSimpleFrameSliceDecodeHighAllowsHigh10AndHigh12SliceBoundaryDeblocking(t *testing.T) {
	for _, bitDepth := range []int32{10, 12} {
		for _, cabac := range []int32{0, 1} {
			for _, sliceType := range []int32{PictureTypeI, PictureTypeP} {
				t.Run(fmt.Sprintf("%s/cabac%d/%s", bitDepthName(bitDepth), cabac, pictureTypeName(sliceType)), func(t *testing.T) {
					m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, bitDepth, 1, 2, true, sliceType)
					sh.PPS.CABAC = cabac
					sh.DeblockingFilter = 2
					if sliceType == PictureTypeP {
						sh.RefCount = [2]uint32{1, 0}
					}

					if err := validateSimpleFrameSliceDecodeInputsHigh(m, dst, sh, 4); err != nil {
						t.Fatalf("high slice-boundary deblock validation err = %v, want nil", err)
					}
				})
			}
		}
	}
}

func TestValidateSimpleFrameSliceDecodeHighAllowsHigh10AndHigh12ChromaFrameDeblocking(t *testing.T) {
	for _, bitDepth := range []int32{10, 12} {
		for _, chromaFormatIDC := range []int{2, 3} {
			for _, deblockMode := range []int32{0, 1} {
				for _, sliceType := range []int32{PictureTypeI, PictureTypeP} {
					t.Run(fmt.Sprintf("%s/%s/deblock%d/%s", bitDepthName(bitDepth), chromaFormatName(chromaFormatIDC), deblockMode, pictureTypeName(sliceType)), func(t *testing.T) {
						m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, bitDepth, chromaFormatIDC, 2, deblockMode != 0, sliceType)
						sh.DeblockingFilter = deblockMode
						if sliceType == PictureTypeP {
							sh.RefCount = [2]uint32{1, 0}
						}

						if err := validateSimpleFrameSliceDecodeInputsHigh(m, dst, sh, 4); err != nil {
							t.Fatalf("high chroma frame deblock validation err = %v, want nil", err)
						}
					})
				}
			}
		}
	}
}

func TestValidateSimpleFrameSliceDecodeHighAllowsHigh10ChromaBFrameDeblocking(t *testing.T) {
	for _, chromaFormatIDC := range []int{2, 3} {
		for _, deblockMode := range []int32{0, 1} {
			t.Run(fmt.Sprintf("%s/deblock%d", chromaFormatName(chromaFormatIDC), deblockMode), func(t *testing.T) {
				m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, 10, chromaFormatIDC, 2, deblockMode != 0, PictureTypeB)
				sh.DeblockingFilter = deblockMode
				sh.RefCount = [2]uint32{1, 1}
				sh.PPS.WeightedBipredIDC = 2

				if err := validateSimpleFrameSliceDecodeInputsHigh(m, dst, sh, 4); err != nil {
					t.Fatalf("high10 chroma B frame deblock validation err = %v, want nil", err)
				}
			})
		}
	}
}

func TestValidateSimpleFrameSliceDecodeHighAllowsHigh10ChromaImplicitWeightedBDeblocking(t *testing.T) {
	for _, chromaFormatIDC := range []int{2, 3} {
		for _, deblockMode := range []int32{0, 1} {
			t.Run(fmt.Sprintf("%s/deblock%d", chromaFormatName(chromaFormatIDC), deblockMode), func(t *testing.T) {
				m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, 10, chromaFormatIDC, 2, deblockMode != 0, PictureTypeB)
				sh.DeblockingFilter = deblockMode
				sh.RefCount = [2]uint32{2, 1}
				sh.PPS.WeightedBipredIDC = 2
				sh.PredWeightTable.UseWeight = 2
				sh.PredWeightTable.UseWeightChroma = 2

				if err := validateSimpleFrameSliceDecodeInputsHigh(m, dst, sh, 4); err != nil {
					t.Fatalf("high10 chroma implicit weighted B deblock validation err = %v, want nil", err)
				}
			})
		}
	}
}

func TestValidateSimpleFrameSliceDecodeHighAllowsHigh10ChromaExplicitWeightedBDeblocking(t *testing.T) {
	for _, chromaFormatIDC := range []int{2, 3} {
		for _, deblockMode := range []int32{0, 1} {
			for _, tt := range []struct {
				name            string
				useWeight       int32
				useWeightChroma int32
			}{
				{name: "luma", useWeight: 1},
				{name: "chroma", useWeightChroma: 1},
				{name: "luma-chroma", useWeight: 1, useWeightChroma: 1},
			} {
				t.Run(fmt.Sprintf("%s/deblock%d/%s", chromaFormatName(chromaFormatIDC), deblockMode, tt.name), func(t *testing.T) {
					m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, 10, chromaFormatIDC, 2, deblockMode != 0, PictureTypeB)
					sh.DeblockingFilter = deblockMode
					sh.RefCount = [2]uint32{2, 1}
					sh.PPS.WeightedBipredIDC = 1
					sh.PredWeightTable.UseWeight = tt.useWeight
					sh.PredWeightTable.UseWeightChroma = tt.useWeightChroma

					if err := validateSimpleFrameSliceDecodeInputsHigh(m, dst, sh, 4); err != nil {
						t.Fatalf("high10 chroma explicit weighted B deblock validation err = %v, want nil", err)
					}
				})
			}
		}
	}
}

func TestValidateSimpleFrameSliceDecodeHighAllowsHigh10ChromaWeightedBSliceBoundaryDeblock(t *testing.T) {
	for _, chromaFormatIDC := range []int{2, 3} {
		for _, cabac := range []int32{0, 1} {
			for _, tt := range []struct {
				name            string
				weightedBipred  uint32
				useWeight       int32
				useWeightChroma int32
			}{
				{name: "implicit-serialized", weightedBipred: 2},
				{name: "implicit-initialized", weightedBipred: 2, useWeight: 2, useWeightChroma: 2},
				{name: "explicit-default", weightedBipred: 1},
				{name: "explicit-luma", weightedBipred: 1, useWeight: 1},
				{name: "explicit-chroma", weightedBipred: 1, useWeightChroma: 1},
				{name: "explicit-luma-chroma", weightedBipred: 1, useWeight: 1, useWeightChroma: 1},
			} {
				t.Run(fmt.Sprintf("%s/cabac%d/%s", chromaFormatName(chromaFormatIDC), cabac, tt.name), func(t *testing.T) {
					m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, 10, chromaFormatIDC, 2, true, PictureTypeB)
					sh.DeblockingFilter = 2
					sh.RefCount = [2]uint32{2, 1}
					sh.PPS.CABAC = cabac
					sh.PPS.WeightedBipredIDC = tt.weightedBipred
					sh.PredWeightTable.UseWeight = tt.useWeight
					sh.PredWeightTable.UseWeightChroma = tt.useWeightChroma

					if err := validateSimpleFrameSliceDecodeInputsHigh(m, dst, sh, 4); err != nil {
						t.Fatalf("high10 chroma weighted-B slice-boundary deblock validation err = %v, want nil", err)
					}
				})
			}
		}
	}
}

func TestValidateSimpleFrameSliceDecodeHighAllowsHigh10ChromaWeightedPFrameDeblock(t *testing.T) {
	for _, chromaFormatIDC := range []int{2, 3} {
		for _, deblockMode := range []int32{0, 1} {
			t.Run(fmt.Sprintf("%s/deblock%d/weighted-pps-i", chromaFormatName(chromaFormatIDC), deblockMode), func(t *testing.T) {
				m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, 10, chromaFormatIDC, 2, deblockMode != 0, PictureTypeI)
				sh.DeblockingFilter = deblockMode
				sh.PPS.WeightedPred = 1

				if err := validateSimpleFrameSliceDecodeInputsHigh(m, dst, sh, 4); err != nil {
					t.Fatalf("high chroma weighted-P PPS I validation err = %v, want nil", err)
				}
			})

			t.Run(fmt.Sprintf("%s/deblock%d/weighted-pps-unweighted-p", chromaFormatName(chromaFormatIDC), deblockMode), func(t *testing.T) {
				m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, 10, chromaFormatIDC, 2, deblockMode != 0, PictureTypeP)
				sh.DeblockingFilter = deblockMode
				sh.RefCount = [2]uint32{1, 0}
				sh.PPS.WeightedPred = 1

				if err := validateSimpleFrameSliceDecodeInputsHigh(m, dst, sh, 4); err != nil {
					t.Fatalf("high chroma weighted-P PPS unweighted P validation err = %v, want nil", err)
				}
			})

			t.Run(fmt.Sprintf("%s/deblock%d/weighted-p", chromaFormatName(chromaFormatIDC), deblockMode), func(t *testing.T) {
				m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, 10, chromaFormatIDC, 2, deblockMode != 0, PictureTypeP)
				sh.DeblockingFilter = deblockMode
				sh.RefCount = [2]uint32{1, 0}
				sh.PPS.WeightedPred = 1
				sh.PredWeightTable = highWeightedPPredWeightTable()

				if err := validateSimpleFrameSliceDecodeInputsHigh(m, dst, sh, 4); err != nil {
					t.Fatalf("high chroma weighted P validation err = %v, want nil", err)
				}
			})

			t.Run(fmt.Sprintf("%s/deblock%d/luma-only-weighted-p", chromaFormatName(chromaFormatIDC), deblockMode), func(t *testing.T) {
				m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, 10, chromaFormatIDC, 2, deblockMode != 0, PictureTypeP)
				sh.DeblockingFilter = deblockMode
				sh.RefCount = [2]uint32{1, 0}
				sh.PPS.WeightedPred = 1
				sh.PredWeightTable = highWeightedPPredWeightTable()
				sh.PredWeightTable.UseWeightChroma = 0

				if err := validateSimpleFrameSliceDecodeInputsHigh(m, dst, sh, 4); err != nil {
					t.Fatalf("high chroma luma-only weighted P validation err = %v, want nil", err)
				}
			})
		}
	}
}

func TestValidateSimpleFrameSliceDecodeHighAllowsHigh10ChromaWeightedPSliceBoundaryDeblock(t *testing.T) {
	weights := []struct {
		name  string
		table func(chromaFormatIDC int) PredWeightTable
	}{
		{name: "luma-only", table: func(int) PredWeightTable {
			pwt := highWeightedPPredWeightTable()
			pwt.UseWeightChroma = 0
			return pwt
		}},
		{name: "luma-chroma", table: func(int) PredWeightTable {
			return highWeightedPPredWeightTable()
		}},
		{name: "source-chroma-only", table: highSourceChromaOnlyWeightedPPredWeightTable},
	}
	for _, chromaFormatIDC := range []int{2, 3} {
		for _, cabac := range []int32{0, 1} {
			for _, weight := range weights {
				t.Run(fmt.Sprintf("%s/cabac%d/%s", chromaFormatName(chromaFormatIDC), cabac, weight.name), func(t *testing.T) {
					m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, 10, chromaFormatIDC, 2, true, PictureTypeP)
					sh.DeblockingFilter = 2
					sh.RefCount = [2]uint32{1, 0}
					sh.PPS.CABAC = cabac
					sh.PPS.WeightedPred = 1
					sh.PredWeightTable = weight.table(chromaFormatIDC)

					if err := validateSimpleFrameSliceDecodeInputsHigh(m, dst, sh, 4); err != nil {
						t.Fatalf("high10 chroma weighted P slice-boundary validation err = %v, want nil", err)
					}
				})
			}
		}
	}
}

func TestValidateSimpleFrameSliceDecodeHighAllowsHighChromaWeightedPDeblock(t *testing.T) {
	weights := []struct {
		name  string
		table func(chromaFormatIDC int) PredWeightTable
	}{
		{name: "luma-only", table: func(int) PredWeightTable {
			pwt := highWeightedPPredWeightTable()
			pwt.UseWeightChroma = 0
			return pwt
		}},
		{name: "luma-chroma", table: func(int) PredWeightTable {
			return highWeightedPPredWeightTable()
		}},
	}
	for _, bitDepth := range []int32{12, 14} {
		for _, chromaFormatIDC := range []int{2, 3} {
			for _, deblockMode := range []int32{0, 1, 2} {
				for _, cabac := range []int32{0, 1} {
					t.Run(fmt.Sprintf("%s/%s/deblock%d/cabac%d/weighted-pps-i", bitDepthName(bitDepth), chromaFormatName(chromaFormatIDC), deblockMode, cabac), func(t *testing.T) {
						m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, bitDepth, chromaFormatIDC, 2, deblockMode != 0, PictureTypeI)
						sh.DeblockingFilter = deblockMode
						sh.PPS.CABAC = cabac
						sh.PPS.WeightedPred = 1

						if err := validateSimpleFrameSliceDecodeInputsHigh(m, dst, sh, 4); err != nil {
							t.Fatalf("%s chroma weighted-P PPS I validation err = %v, want nil", bitDepthName(bitDepth), err)
						}
					})

					t.Run(fmt.Sprintf("%s/%s/deblock%d/cabac%d/weighted-pps-unweighted-p", bitDepthName(bitDepth), chromaFormatName(chromaFormatIDC), deblockMode, cabac), func(t *testing.T) {
						m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, bitDepth, chromaFormatIDC, 2, deblockMode != 0, PictureTypeP)
						sh.DeblockingFilter = deblockMode
						sh.RefCount = [2]uint32{1, 0}
						sh.PPS.CABAC = cabac
						sh.PPS.WeightedPred = 1

						if err := validateSimpleFrameSliceDecodeInputsHigh(m, dst, sh, 4); err != nil {
							t.Fatalf("%s chroma weighted-P PPS unweighted P validation err = %v, want nil", bitDepthName(bitDepth), err)
						}
					})

					for _, weight := range weights {
						t.Run(fmt.Sprintf("%s/%s/deblock%d/cabac%d/%s", bitDepthName(bitDepth), chromaFormatName(chromaFormatIDC), deblockMode, cabac, weight.name), func(t *testing.T) {
							m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, bitDepth, chromaFormatIDC, 2, deblockMode != 0, PictureTypeP)
							sh.DeblockingFilter = deblockMode
							sh.RefCount = [2]uint32{1, 0}
							sh.PPS.CABAC = cabac
							sh.PPS.WeightedPred = 1
							sh.PredWeightTable = weight.table(chromaFormatIDC)

							if err := validateSimpleFrameSliceDecodeInputsHigh(m, dst, sh, 4); err != nil {
								t.Fatalf("%s chroma weighted P validation err = %v, want nil", bitDepthName(bitDepth), err)
							}
						})
					}
				}
			}
		}
	}
}

func TestValidateSimpleFrameSliceDecodeHighAllowsHigh14ChromaUnweightedDeblock(t *testing.T) {
	for _, chromaFormatIDC := range []int{2, 3} {
		for _, deblockMode := range []int32{0, 1, 2} {
			for _, cabac := range []int32{0, 1} {
				t.Run(fmt.Sprintf("%s/deblock%d/cabac%d/i", chromaFormatName(chromaFormatIDC), deblockMode, cabac), func(t *testing.T) {
					m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, 14, chromaFormatIDC, 2, deblockMode != 0, PictureTypeI)
					sh.DeblockingFilter = deblockMode
					sh.PPS.CABAC = cabac

					if err := validateSimpleFrameSliceDecodeInputsHigh(m, dst, sh, 4); err != nil {
						t.Fatalf("high14 chroma unweighted I validation err = %v, want nil", err)
					}
				})

				t.Run(fmt.Sprintf("%s/deblock%d/cabac%d/p", chromaFormatName(chromaFormatIDC), deblockMode, cabac), func(t *testing.T) {
					m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, 14, chromaFormatIDC, 2, deblockMode != 0, PictureTypeP)
					sh.DeblockingFilter = deblockMode
					sh.RefCount = [2]uint32{1, 0}
					sh.PPS.CABAC = cabac

					if err := validateSimpleFrameSliceDecodeInputsHigh(m, dst, sh, 4); err != nil {
						t.Fatalf("high14 chroma unweighted P validation err = %v, want nil", err)
					}
				})
			}
		}
	}
}

func TestPredWeightTableCollapsesChromaOnlyPWeightToUseWeight(t *testing.T) {
	gb := bitReaderFromBits(t, "011 010 0 1 00110 011 00100 010")
	sh := &SliceHeader{
		SliceTypeNoS:     PictureTypeP,
		PictureStructure: PictureTopField,
		RefCount:         [2]uint32{1, 0},
		SPS:              &SPS{ChromaFormatIDC: 3},
	}

	if err := predWeightTable(&gb, sh); err != nil {
		t.Fatalf("pred weight table err = %v, want nil", err)
	}
	if gb.bitsLeft() != 0 {
		t.Fatalf("bits left = %d, want 0", gb.bitsLeft())
	}

	pwt := sh.PredWeightTable
	if pwt.UseWeight != 1 || pwt.UseWeightChroma != 1 {
		t.Fatalf("use weight = %d/%d, want source-normalized chroma-only 1/1", pwt.UseWeight, pwt.UseWeightChroma)
	}
	if pwt.LumaWeightFlag[0] != 0 || pwt.ChromaWeightFlag[0] != 1 {
		t.Fatalf("weight flags = luma %d chroma %d, want 0/1", pwt.LumaWeightFlag[0], pwt.ChromaWeightFlag[0])
	}
	if got, want := pwt.LumaWeight[0][0], ([2]int32{4, 0}); got != want {
		t.Fatalf("luma weight = %v, want %v", got, want)
	}
	if got, want := pwt.ChromaWeight[0][0][0], ([2]int32{3, -1}); got != want {
		t.Fatalf("cb weight = %v, want %v", got, want)
	}
	if got, want := pwt.ChromaWeight[0][0][1], ([2]int32{2, 1}); got != want {
		t.Fatalf("cr weight = %v, want %v", got, want)
	}
}

func TestValidateSimpleFrameSliceDecodeHighRejectsUnnormalizedChromaOnlyWeightedPMetadata(t *testing.T) {
	for _, chromaFormatIDC := range []int{2, 3} {
		t.Run(chromaFormatName(chromaFormatIDC), func(t *testing.T) {
			m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, 10, chromaFormatIDC, 2, false, PictureTypeP)
			sh.RefCount = [2]uint32{1, 0}
			sh.PPS.WeightedPred = 1
			sh.PredWeightTable = highSourceChromaOnlyWeightedPPredWeightTable(chromaFormatIDC)
			sh.PredWeightTable.UseWeight = 0

			if err := validateSimpleFrameSliceDecodeInputsHigh(m, dst, sh, 4); err != ErrUnsupported {
				t.Fatalf("unnormalized high chroma weighted P validation err = %v, want ErrUnsupported", err)
			}
		})
	}
}

func TestValidateSimpleFrameSliceDecodeHighAllowsChromaSliceBoundaryDeblocking(t *testing.T) {
	for _, bitDepth := range []int32{10, 12} {
		for _, chromaFormatIDC := range []int{2, 3} {
			for _, cabac := range []int32{0, 1} {
				for _, sliceType := range []int32{PictureTypeI, PictureTypeP} {
					t.Run(fmt.Sprintf("%s/%s/cabac%d/%s", bitDepthName(bitDepth), chromaFormatName(chromaFormatIDC), cabac, pictureTypeName(sliceType)), func(t *testing.T) {
						m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, bitDepth, chromaFormatIDC, 2, true, sliceType)
						sh.PPS.CABAC = cabac
						sh.DeblockingFilter = 2
						if sliceType == PictureTypeP {
							sh.RefCount = [2]uint32{1, 0}
						}

						if err := validateSimpleFrameSliceDecodeInputsHigh(m, dst, sh, 4); err != nil {
							t.Fatalf("%s %s slice-boundary deblock validation err = %v, want nil", bitDepthName(bitDepth), chromaFormatName(chromaFormatIDC), err)
						}
					})
				}
			}
		}
	}
}

func TestValidateSimpleFrameSliceDecodeHighRejectsUnprovedDeblockingModes(t *testing.T) {
	for _, tt := range []struct {
		name     string
		bitDepth int32
		run      func(*SliceHeader)
	}{
		{
			name:     "10-bit/b-slice-boundary-mode",
			bitDepth: 10,
			run: func(sh *SliceHeader) {
				sh.SliceType = PictureTypeB
				sh.SliceTypeNoS = PictureTypeB
				sh.RefCount = [2]uint32{1, 1}
				sh.DeblockingFilter = 2
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, tt.bitDepth, 1, 2, false, PictureTypeI)
			tt.run(sh)

			if err := validateSimpleFrameSliceDecodeInputsHigh(m, dst, sh, 4); err != ErrUnsupported {
				t.Fatalf("high deblock validation err = %v, want ErrUnsupported", err)
			}
		})
	}
}

func TestValidateSimpleFrameSliceDecodeHighAllowsImplicitWeightedB(t *testing.T) {
	m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, 10, 1, 1, false, PictureTypeB)
	sh.RefCount = [2]uint32{1, 1}
	sh.PPS.WeightedBipredIDC = 2

	if err := validateSimpleFrameSliceDecodeInputsHigh(m, dst, sh, 4); err != nil {
		t.Fatalf("serialized implicit weighted high B validation err = %v, want nil", err)
	}

	sh.PredWeightTable.UseWeight = 2
	sh.PredWeightTable.UseWeightChroma = 2
	if err := validateSimpleFrameSliceDecodeInputHighRefs(sh, h264FrameSliceDecodeInputHigh{PredWeight: &sh.PredWeightTable}); err != nil {
		t.Fatalf("initialized implicit weighted high B ref validation err = %v, want nil", err)
	}
}

func TestValidateSimpleFrameSliceDecodeHighAllowsExplicitWeightedB(t *testing.T) {
	for _, tt := range []struct {
		name            string
		useWeight       int32
		useWeightChroma int32
	}{
		{name: "luma", useWeight: 1},
		{name: "luma-chroma", useWeight: 1, useWeightChroma: 1},
		{name: "default-table", useWeight: 0, useWeightChroma: 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, 12, 1, 1, false, PictureTypeB)
			sh.RefCount = [2]uint32{1, 1}
			sh.PPS.WeightedBipredIDC = 1
			sh.PredWeightTable.UseWeight = tt.useWeight
			sh.PredWeightTable.UseWeightChroma = tt.useWeightChroma

			if err := validateSimpleFrameSliceDecodeInputsHigh(m, dst, sh, 4); err != nil {
				t.Fatalf("explicit weighted high B validation err = %v, want nil", err)
			}
			if err := validateSimpleFrameSliceDecodeInputHighRefs(sh, h264FrameSliceDecodeInputHigh{PredWeight: &sh.PredWeightTable}); err != nil {
				t.Fatalf("explicit weighted high B ref validation err = %v, want nil", err)
			}
		})
	}
}

func TestValidateSimpleFrameSliceDecodeHighRejectsUnsupportedWeightedB(t *testing.T) {
	m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, 10, 1, 1, false, PictureTypeB)
	sh.RefCount = [2]uint32{1, 1}
	sh.PPS.WeightedBipredIDC = 2
	sh.PredWeightTable.UseWeight = 2
	sh.PredWeightTable.UseWeightChroma = 0

	if err := validateSimpleFrameSliceDecodeInputsHigh(m, dst, sh, 4); err != ErrUnsupported {
		t.Fatalf("weighted high B validation err = %v, want ErrUnsupported", err)
	}
}

func TestDecodeFrameSliceDataHighRejectsBInputPredWeightBeforeEntropy(t *testing.T) {
	m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, 10, 1, 1, false, PictureTypeB)
	sh.RefCount = [2]uint32{1, 1}
	gb := newBitReader(cavlcBitString("10100"))
	pwt := PredWeightTable{UseWeight: 1}

	_, err := m.decodeFrameSliceDataHigh(&gb, dst, sh, h264FrameSliceDecodeInputHigh{
		SliceNum:   11,
		PredWeight: &pwt,
	})
	if err != ErrUnsupported {
		t.Fatalf("high B input pred weight err = %v, want ErrUnsupported", err)
	}
	if gb.bitPos != 0 {
		t.Fatalf("bit reader consumed %d bits, want 0", gb.bitPos)
	}
}

func TestValidateSimpleFrameSliceDecodeHighAllowsWeightedPMetadata(t *testing.T) {
	for _, bitDepth := range []int32{10, 12} {
		for _, deblockMode := range []int32{0, 1, 2} {
			name := fmt.Sprintf("%s/deblock-mode-%d", bitDepthName(bitDepth), deblockMode)
			t.Run(name, func(t *testing.T) {
				m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, bitDepth, 1, 2, deblockMode != 0, PictureTypeP)
				sh.RefCount = [2]uint32{1, 0}
				sh.DeblockingFilter = deblockMode
				sh.PPS.WeightedPred = 1
				sh.PredWeightTable = highWeightedPPredWeightTable()

				if err := validateSimpleFrameSliceDecodeInputsHigh(m, dst, sh, 4); err != nil {
					t.Fatalf("high weighted P validation err = %v, want nil", err)
				}
			})
		}
	}

	for _, deblockMode := range []int32{0, 1, 2} {
		t.Run(fmt.Sprintf("14-bit/deblock-mode-%d", deblockMode), func(t *testing.T) {
			m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, 14, 1, 2, deblockMode != 0, PictureTypeP)
			sh.DeblockingFilter = deblockMode
			sh.RefCount = [2]uint32{1, 0}
			sh.PPS.WeightedPred = 1
			sh.PredWeightTable = highWeightedPPredWeightTable()

			if err := validateSimpleFrameSliceDecodeInputsHigh(m, dst, sh, 4); err != nil {
				t.Fatalf("high weighted P validation err = %v, want nil", err)
			}
		})
	}
}

func TestValidateSimpleFrameSliceDecodeHighWeightedPStillRejectsStagedBoundaries(t *testing.T) {
	tests := []struct {
		name        string
		bitDepth    int32
		chroma      int32
		format      int
		deblock     bool
		deblockMode int32
		slice       int32
	}{
		{name: "9-bit-422", bitDepth: 9, chroma: 9, format: 2, slice: PictureTypeP},
		{name: "unequal-depth", bitDepth: 10, chroma: 12, format: 1, slice: PictureTypeP},
		{name: "b-slice", bitDepth: 10, chroma: 10, format: 1, slice: PictureTypeB},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, dst, sh := highFrameSliceDecodeFixture(t, tt.bitDepth, tt.format, tt.deblock, tt.slice)
			if tt.deblockMode != 0 {
				sh.DeblockingFilter = tt.deblockMode
			}
			sh.SPS.BitDepthChroma = tt.chroma
			sh.RefCount = [2]uint32{1, 0}
			sh.PPS.WeightedPred = 1
			sh.PredWeightTable.UseWeight = 1
			sh.PredWeightTable.UseWeightChroma = 1

			if err := validateSimpleFrameSliceDecodeInputsHigh(m, dst, sh, 4); err != ErrUnsupported {
				t.Fatalf("weighted high validation err = %v, want ErrUnsupported", err)
			}
		})
	}
}

func TestValidateSimpleFrameSliceDecodeHighAllowsHigh1214ChromaFieldWeightedP(t *testing.T) {
	weights := []struct {
		name  string
		table func(chromaFormatIDC int) PredWeightTable
	}{
		{name: "luma-only", table: func(int) PredWeightTable {
			pwt := highWeightedPPredWeightTable()
			pwt.UseWeightChroma = 0
			return pwt
		}},
		{name: "luma-chroma", table: func(int) PredWeightTable {
			return highWeightedPPredWeightTable()
		}},
		{name: "source-chroma-only", table: highSourceChromaOnlyWeightedPPredWeightTable},
	}
	for _, bitDepth := range []int32{12, 14} {
		for _, chromaFormatIDC := range []int{2, 3} {
			for _, pictureStructure := range []int32{PictureTopField, PictureBottomField} {
				for _, deblockMode := range []int32{0, 1, 2} {
					for _, cabac := range []int32{0, 1} {
						for _, weight := range weights {
							t.Run(fmt.Sprintf("%s/%s/picture%d/deblock%d/cabac%d/%s", bitDepthName(bitDepth), chromaFormatName(chromaFormatIDC), pictureStructure, deblockMode, cabac, weight.name), func(t *testing.T) {
								m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, bitDepth, chromaFormatIDC, 2, deblockMode != 0, PictureTypeP)
								sh.SPS.FrameMBSOnlyFlag = 0
								sh.SPS.MBAFF = 1
								sh.PictureStructure = pictureStructure
								sh.DeblockingFilter = deblockMode
								sh.RefCount = [2]uint32{1, 0}
								sh.PPS.CABAC = cabac
								sh.PPS.WeightedPred = 1
								sh.PredWeightTable = weight.table(chromaFormatIDC)

								if err := validateSimpleFrameSliceDecodeInputsHigh(m, dst, sh, 4); err != nil {
									t.Fatalf("%s chroma field weighted P validation err = %v, want nil", bitDepthName(bitDepth), err)
								}
							})
						}
					}
				}
			}
		}
	}
}

func TestValidateSimpleFrameSliceDecodeHighRejectsHigh1214ChromaFieldWeightedPStagedBoundaries(t *testing.T) {
	for _, bitDepth := range []int32{12, 14} {
		for _, chromaFormatIDC := range []int{2, 3} {
			for _, pictureStructure := range []int32{PictureTopField, PictureBottomField} {
				for _, cabac := range []int32{0, 1} {
					for _, tt := range []struct {
						name string
						run  func(*SliceHeader)
					}{
						{name: "b-slice", run: func(sh *SliceHeader) {
							sh.SliceType = PictureTypeB
							sh.SliceTypeNoS = PictureTypeB
							sh.RefCount = [2]uint32{1, 1}
						}},
						{name: "unequal-depth", run: func(sh *SliceHeader) {
							sh.SPS.BitDepthChroma = bitDepth + 2
						}},
						{name: "frame-mbs-only", run: func(sh *SliceHeader) {
							sh.SPS.FrameMBSOnlyFlag = 1
						}},
					} {
						t.Run(fmt.Sprintf("%s/%s/picture%d/cabac%d/%s", bitDepthName(bitDepth), chromaFormatName(chromaFormatIDC), pictureStructure, cabac, tt.name), func(t *testing.T) {
							m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, bitDepth, chromaFormatIDC, 2, true, PictureTypeP)
							sh.SPS.FrameMBSOnlyFlag = 0
							sh.SPS.MBAFF = 1
							sh.PictureStructure = pictureStructure
							sh.DeblockingFilter = 2
							sh.RefCount = [2]uint32{1, 0}
							sh.PPS.CABAC = cabac
							sh.PPS.WeightedPred = 1
							sh.PredWeightTable = highWeightedPPredWeightTable()
							tt.run(sh)

							if err := validateSimpleFrameSliceDecodeInputsHigh(m, dst, sh, 4); err != ErrUnsupported {
								t.Fatalf("%s chroma field weighted P staged-boundary validation err = %v, want ErrUnsupported", bitDepthName(bitDepth), err)
							}
						})
					}
				}
			}
		}
	}
}

func TestValidateHighFrameSliceReconstructAllowsHigh9IntraResidualScope(t *testing.T) {
	_, _, sh := highFrameSliceDecodeFixture(t, 9, 1, false, PictureTypeI)

	for _, tt := range []struct {
		name     string
		mbType   uint32
		cbp      int
		cbpTable int
	}{
		{name: "intra-pcm", mbType: MBTypeIntraPCM},
		{name: "intra4x4-no-residual", mbType: MBTypeIntra4x4},
		{name: "intra4x4-residual", mbType: MBTypeIntra4x4, cbp: 1, cbpTable: 1},
		{name: "intra16x16-no-residual", mbType: MBTypeIntra16x16},
		{name: "intra16x16-cabac-luma-chroma", mbType: MBTypeIntra16x16, cbp: 0x2f, cbpTable: 0x16f},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateHighFrameSliceMacroblockForReconstructWithSubMB(sh, tt.mbType, nil, tt.cbp, tt.cbpTable); err != nil {
				t.Fatalf("high9 %s reconstruct validation err = %v, want nil", tt.name, err)
			}
		})
	}
}

func TestValidateHighFrameSliceReconstructAllowsHigh12IntraResidualScope(t *testing.T) {
	_, _, sh := highFrameSliceDecodeFixture(t, 12, 1, false, PictureTypeI)

	if err := validateHighFrameSliceMacroblockForReconstructWithSubMB(sh, MBTypeIntraPCM, nil, 0, 0); err != nil {
		t.Fatalf("high12 IntraPCM reconstruct validation err = %v, want nil", err)
	}
	if err := validateHighFrameSliceMacroblockForReconstructWithSubMB(sh, MBTypeIntra4x4, nil, 0, 0); err != nil {
		t.Fatalf("high12 Intra4x4 no-residual reconstruct validation err = %v, want nil", err)
	}
	for _, tt := range []struct {
		name     string
		cbp      int
		cbpTable int
	}{
		{name: "single-luma", cbp: 0x01, cbpTable: 0x01},
		{name: "single-luma-cavlc", cbp: 0x01, cbpTable: 0x1001},
		{name: "partition-luma-chroma-13", cbp: 0x13, cbpTable: 0xd3},
		{name: "partition-luma-chroma-13-cavlc", cbp: 0x13, cbpTable: 0xf013},
		{name: "partition-luma-chroma-15", cbp: 0x15, cbpTable: 0xd5},
		{name: "partition-luma-chroma-15-cavlc", cbp: 0x15, cbpTable: 0x1f015},
		{name: "partition-luma-chroma-17", cbp: 0x17, cbpTable: 0xd7},
		{name: "partition-luma-chroma-17-cavlc", cbp: 0x17, cbpTable: 0x7017},
		{name: "x264-luma-chroma", cbp: 0x2f, cbpTable: 0xef},
		{name: "luma-chroma-cavlc", cbp: 0x2f, cbpTable: 0x7f02f},
		{name: "luma-chroma-cavlc-two-mb", cbp: 0x2f, cbpTable: 0xff02f},
	} {
		t.Run("intra4x4-"+tt.name, func(t *testing.T) {
			if err := validateHighFrameSliceMacroblockForReconstructWithSubMB(sh, MBTypeIntra4x4, nil, tt.cbp, tt.cbpTable); err != nil {
				t.Fatalf("high12 Intra4x4 %s residual reconstruct validation err = %v, want nil", tt.name, err)
			}
		})
	}
	if err := validateHighFrameSliceMacroblockForReconstructWithSubMB(sh, MBTypeIntra16x16, nil, 0, 0); err != nil {
		t.Fatalf("high12 Intra16x16 no-residual reconstruct validation err = %v, want nil", err)
	}
	if err := validateHighFrameSliceMacroblockForReconstructWithSubMB(sh, MBTypeIntra16x16, nil, 0, 0x100); err != nil {
		t.Fatalf("high12 Intra16x16 luma-DC residual reconstruct validation err = %v, want nil", err)
	}
	if err := validateHighFrameSliceMacroblockForReconstructWithSubMB(sh, MBTypeIntra16x16, nil, 0x10, 0x10); err != nil {
		t.Fatalf("high12 Intra16x16 chroma-DC CAVLC residual reconstruct validation err = %v, want nil", err)
	}
	if err := validateHighFrameSliceMacroblockForReconstructWithSubMB(sh, MBTypeIntra16x16, nil, 0x10, 0x50); err != nil {
		t.Fatalf("high12 Intra16x16 chroma-DC CABAC residual reconstruct validation err = %v, want nil", err)
	}
	if err := validateHighFrameSliceMacroblockForReconstructWithSubMB(sh, MBTypeIntra16x16, nil, 0x10, 0x1d0); err != nil {
		t.Fatalf("high12 Intra16x16 partition chroma-DC CABAC residual reconstruct validation err = %v, want nil", err)
	}
	if err := validateHighFrameSliceMacroblockForReconstructWithSubMB(sh, MBTypeIntra16x16, nil, 0x20, 0x20); err != nil {
		t.Fatalf("high12 Intra16x16 chroma-AC residual reconstruct validation err = %v, want nil", err)
	}
	if err := validateHighFrameSliceMacroblockForReconstructWithSubMB(sh, MBTypeIntra16x16, nil, 0x20, 0x60); err != nil {
		t.Fatalf("high12 Intra16x16 chroma-DC/AC CABAC residual reconstruct validation err = %v, want nil", err)
	}
	if err := validateHighFrameSliceMacroblockForReconstructWithSubMB(sh, MBTypeIntra16x16, nil, 0x2f, 0xf02f); err != nil {
		t.Fatalf("high12 Intra16x16 luma/chroma CAVLC residual reconstruct validation err = %v, want nil", err)
	}
	if err := validateHighFrameSliceMacroblockForReconstructWithSubMB(sh, MBTypeIntra16x16, nil, 0x2f, 0x16f); err != nil {
		t.Fatalf("high12 Intra16x16 luma/chroma CABAC residual reconstruct validation err = %v, want nil", err)
	}
	if err := validateHighFrameSliceMacroblockForReconstructWithSubMB(sh, MBTypeIntra16x16, nil, 0x2f, 0xef); err != nil {
		t.Fatalf("high12 Intra16x16 x264 luma/chroma CABAC residual reconstruct validation err = %v, want nil", err)
	}
	if err := validateHighFrameSliceMacroblockForReconstructWithSubMB(sh, MBTypeIntra16x16, nil, 0x0f, 0x0f); err != nil {
		t.Fatalf("high12 Intra16x16 luma-AC CABAC residual reconstruct validation err = %v, want nil", err)
	}
	if err := validateHighFrameSliceMacroblockForReconstructWithSubMB(sh, MBTypeIntra16x16, nil, 0x0f, 0xf00f); err != nil {
		t.Fatalf("high12 Intra16x16 luma-AC CAVLC residual reconstruct validation err = %v, want nil", err)
	}
	if err := validateHighFrameSliceMacroblockForReconstructWithSubMB(sh, MBTypeIntra16x16, nil, 0x0f, 0x10f); err != nil {
		t.Fatalf("high12 Intra16x16 luma-DC/AC CABAC residual reconstruct validation err = %v, want nil", err)
	}
	if err := validateHighFrameSliceMacroblockForReconstructWithSubMB(sh, MBTypeIntra16x16, nil, 1, 1); err != ErrUnsupported {
		t.Fatalf("high12 Intra16x16 residual reconstruct validation err = %v, want ErrUnsupported", err)
	}
	if err := validateHighFrameSliceMacroblockForReconstructWithSubMB(sh, MBTypeIntra16x16, nil, 0, 0x101); err != ErrUnsupported {
		t.Fatalf("high12 Intra16x16 mixed residual reconstruct validation err = %v, want ErrUnsupported", err)
	}
	if err := validateHighFrameSliceMacroblockForReconstructWithSubMB(sh, MBTypeIntra16x16, nil, 0x10, 0x90); err != ErrUnsupported {
		t.Fatalf("high12 Intra16x16 unproved chroma-DC residual reconstruct validation err = %v, want ErrUnsupported", err)
	}
	if err := validateHighFrameSliceMacroblockForReconstructWithSubMB(sh, MBTypeIntra16x16, nil, 0x20, 0xa0); err != ErrUnsupported {
		t.Fatalf("high12 Intra16x16 unproved mixed chroma residual reconstruct validation err = %v, want ErrUnsupported", err)
	}
	if err := validateHighFrameSliceMacroblockForReconstructWithSubMB(sh, MBTypeIntra16x16, nil, 0x2f, 0x1ef); err != ErrUnsupported {
		t.Fatalf("high12 Intra16x16 unproved luma/chroma residual reconstruct validation err = %v, want ErrUnsupported", err)
	}
	if err := validateHighFrameSliceMacroblockForReconstructWithSubMB(sh, MBTypeIntra16x16, nil, 0x0f, 0x110f); err != ErrUnsupported {
		t.Fatalf("high12 Intra16x16 unproved mixed luma residual reconstruct validation err = %v, want ErrUnsupported", err)
	}
}

func TestValidateHighFrameSliceReconstructAllowsHigh14IntraResidualScope(t *testing.T) {
	_, _, sh := highFrameSliceDecodeFixture(t, 14, 1, false, PictureTypeI)

	if err := validateHighFrameSliceMacroblockForReconstructWithSubMB(sh, MBTypeIntraPCM, nil, 0, 0); err != nil {
		t.Fatalf("high14 IntraPCM reconstruct validation err = %v, want nil", err)
	}
	if err := validateHighFrameSliceMacroblockForReconstructWithSubMB(sh, MBTypeIntra4x4, nil, 0, 0); err != nil {
		t.Fatalf("high14 Intra4x4 no-residual reconstruct validation err = %v, want nil", err)
	}
	for _, tt := range []struct {
		name     string
		cbp      int
		cbpTable int
	}{
		{name: "single-luma", cbp: 0x01, cbpTable: 0x01},
		{name: "single-luma-cavlc", cbp: 0x01, cbpTable: 0x1001},
		{name: "partition-luma-chroma-13", cbp: 0x13, cbpTable: 0xd3},
		{name: "partition-luma-chroma-13-cavlc", cbp: 0x13, cbpTable: 0xf013},
		{name: "partition-luma-chroma-15", cbp: 0x15, cbpTable: 0xd5},
		{name: "partition-luma-chroma-15-cavlc", cbp: 0x15, cbpTable: 0x1f015},
		{name: "partition-luma-chroma-17", cbp: 0x17, cbpTable: 0xd7},
		{name: "partition-luma-chroma-17-cavlc", cbp: 0x17, cbpTable: 0x7017},
		{name: "x264-luma-chroma", cbp: 0x2f, cbpTable: 0xef},
		{name: "luma-chroma-cavlc", cbp: 0x2f, cbpTable: 0x7f02f},
		{name: "luma-chroma-cavlc-two-mb", cbp: 0x2f, cbpTable: 0xff02f},
	} {
		t.Run("intra4x4-"+tt.name, func(t *testing.T) {
			if err := validateHighFrameSliceMacroblockForReconstructWithSubMB(sh, MBTypeIntra4x4, nil, tt.cbp, tt.cbpTable); err != nil {
				t.Fatalf("high14 Intra4x4 %s residual reconstruct validation err = %v, want nil", tt.name, err)
			}
		})
	}
	if err := validateHighFrameSliceMacroblockForReconstructWithSubMB(sh, MBTypeIntra16x16, nil, 0, 0); err != nil {
		t.Fatalf("high14 Intra16x16 no-residual reconstruct validation err = %v, want nil", err)
	}
	if err := validateHighFrameSliceMacroblockForReconstructWithSubMB(sh, MBTypeIntra16x16, nil, 0, 0x100); err != nil {
		t.Fatalf("high14 Intra16x16 luma-DC residual reconstruct validation err = %v, want nil", err)
	}
	if err := validateHighFrameSliceMacroblockForReconstructWithSubMB(sh, MBTypeIntra16x16, nil, 0x0f, 0xf00f); err != nil {
		t.Fatalf("high14 Intra16x16 luma-AC/DC residual reconstruct validation err = %v, want nil", err)
	}
	if err := validateHighFrameSliceMacroblockForReconstructWithSubMB(sh, MBTypeIntra16x16, nil, 0x10, 0x10); err != nil {
		t.Fatalf("high14 Intra16x16 chroma-DC residual reconstruct validation err = %v, want nil", err)
	}
	if err := validateHighFrameSliceMacroblockForReconstructWithSubMB(sh, MBTypeIntra16x16, nil, 0x10, 0x1d0); err != nil {
		t.Fatalf("high14 Intra16x16 partition chroma-DC CABAC residual reconstruct validation err = %v, want nil", err)
	}
	if err := validateHighFrameSliceMacroblockForReconstructWithSubMB(sh, MBTypeIntra16x16, nil, 0x20, 0x20); err != nil {
		t.Fatalf("high14 Intra16x16 chroma-AC/DC residual reconstruct validation err = %v, want nil", err)
	}
	if err := validateHighFrameSliceMacroblockForReconstructWithSubMB(sh, MBTypeIntra16x16, nil, 0x2f, 0xf02f); err != nil {
		t.Fatalf("high14 Intra16x16 luma/chroma residual reconstruct validation err = %v, want nil", err)
	}
	for _, tt := range []struct {
		name     string
		mbType   uint32
		cbp      int
		cbpTable int
	}{
		{name: "intra16x16-mixed-luma-dc", mbType: MBTypeIntra16x16, cbp: 0, cbpTable: 0x101},
		{name: "intra16x16-unproved-chroma-dc", mbType: MBTypeIntra16x16, cbp: 0x10, cbpTable: 0x90},
		{name: "intra16x16-unproved-mixed-chroma", mbType: MBTypeIntra16x16, cbp: 0x20, cbpTable: 0xa0},
		{name: "intra16x16-unproved-luma-chroma", mbType: MBTypeIntra16x16, cbp: 0x2f, cbpTable: 0x1ef},
		{name: "intra16x16-unproved-mixed-luma", mbType: MBTypeIntra16x16, cbp: 0x0f, cbpTable: 0x110f},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateHighFrameSliceMacroblockForReconstructWithSubMB(sh, tt.mbType, nil, tt.cbp, tt.cbpTable); err != ErrUnsupported {
				t.Fatalf("high14 %s reconstruct validation err = %v, want ErrUnsupported", tt.name, err)
			}
		})
	}
}

func TestValidateHighFrameSliceReconstructAllowsHigh14CABACIntraResidual(t *testing.T) {
	_, _, sh := highFrameSliceDecodeFixture(t, 14, 1, false, PictureTypeI)
	sh.PPS.CABAC = 1

	for _, tt := range []struct {
		name     string
		mbType   uint32
		cbp      int
		cbpTable int
	}{
		{name: "intra-pcm", mbType: MBTypeIntraPCM},
		{name: "intra4x4-no-residual", mbType: MBTypeIntra4x4},
		{name: "intra4x4-single-luma", mbType: MBTypeIntra4x4, cbp: 0x01, cbpTable: 0x01},
		{name: "intra4x4-partition-luma-chroma-13", mbType: MBTypeIntra4x4, cbp: 0x13, cbpTable: 0xd3},
		{name: "intra4x4-partition-luma-chroma-15", mbType: MBTypeIntra4x4, cbp: 0x15, cbpTable: 0xd5},
		{name: "intra4x4-partition-luma-chroma-17", mbType: MBTypeIntra4x4, cbp: 0x17, cbpTable: 0xd7},
		{name: "intra4x4-luma-chroma-x264", mbType: MBTypeIntra4x4, cbp: 0x2f, cbpTable: 0xef},
		{name: "intra16x16-no-residual", mbType: MBTypeIntra16x16},
		{name: "intra16x16-luma-dc", mbType: MBTypeIntra16x16, cbp: 0, cbpTable: 0x100},
		{name: "intra16x16-chroma-dc", mbType: MBTypeIntra16x16, cbp: 0x10, cbpTable: 0x50},
		{name: "intra16x16-partition-chroma-dc", mbType: MBTypeIntra16x16, cbp: 0x10, cbpTable: 0x1d0},
		{name: "intra16x16-chroma-ac", mbType: MBTypeIntra16x16, cbp: 0x20, cbpTable: 0x60},
		{name: "intra16x16-luma-chroma-x264", mbType: MBTypeIntra16x16, cbp: 0x2f, cbpTable: 0xef},
		{name: "intra16x16-luma-chroma", mbType: MBTypeIntra16x16, cbp: 0x2f, cbpTable: 0x16f},
		{name: "intra16x16-luma-ac", mbType: MBTypeIntra16x16, cbp: 0x0f, cbpTable: 0x0f},
		{name: "intra16x16-luma-dc-ac", mbType: MBTypeIntra16x16, cbp: 0x0f, cbpTable: 0x10f},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateHighFrameSliceMacroblockForReconstructWithSubMB(sh, tt.mbType, nil, tt.cbp, tt.cbpTable); err != nil {
				t.Fatalf("high14 CABAC %s reconstruct validation err = %v, want nil", tt.name, err)
			}
		})
	}

	if err := validateHighFrameSliceMacroblockForReconstructWithSubMB(sh, MBTypeIntra16x16, nil, 0, 0x101); err != ErrUnsupported {
		t.Fatalf("high14 CABAC mixed residual reconstruct validation err = %v, want ErrUnsupported", err)
	}
}

func TestValidateHighFrameSliceReconstructAllowsHigh1214CABACChromaIntraResidual(t *testing.T) {
	for _, tt := range []struct {
		name     string
		bitDepth int32
		chroma   int
		cbp      int
		cbpTable int
	}{
		{name: "high12-422-intra4x4", bitDepth: 12, chroma: 2, cbp: 0x23, cbpTable: 0xe3},
		{name: "high12-444-intra4x4", bitDepth: 12, chroma: 3, cbp: 0x0f, cbpTable: 0x0f},
		{name: "high14-422-intra4x4", bitDepth: 14, chroma: 2, cbp: 0x23, cbpTable: 0xe3},
		{name: "high14-444-intra4x4", bitDepth: 14, chroma: 3, cbp: 0x0f, cbpTable: 0x0f},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, _, sh := highFrameSliceDecodeFixture(t, tt.bitDepth, tt.chroma, false, PictureTypeI)
			sh.PPS.CABAC = 1

			if err := validateHighFrameSliceMacroblockForReconstructWithSubMB(sh, MBTypeIntra4x4, nil, tt.cbp, tt.cbpTable); err != nil {
				t.Fatalf("%s CABAC chroma Intra4x4 reconstruct validation err = %v, want nil", bitDepthName(tt.bitDepth), err)
			}
		})
	}
}

func TestDecodeCAVLCFrameSliceHighReconstructsIntraPCMRun(t *testing.T) {
	const bitDepth = 10
	m, dst, sh := highFrameSliceDecodeFixture(t, bitDepth, 1, false, PictureTypeI)
	pcm0 := h264ReconstructIntraPCMHigh(1, bitDepth, 33)
	pcm1 := h264ReconstructIntraPCMHigh(1, bitDepth, 77)
	gb := newBitReader(append(cavlcIntraPCMBytes(pcm0), cavlcIntraPCMBytes(pcm1)...))

	got, err := m.decodeCAVLCFrameSliceHigh(&gb, dst, sh, h264FrameSliceDecodeInputHigh{SliceNum: 9})
	if err != nil {
		t.Fatalf("decode high cavlc slice failed: %v", err)
	}
	if got.Macroblocks != 2 || got.LastMBXY != 1 || !got.EndOfSlice || !got.EndOfFrame {
		t.Fatalf("slice result = %+v, want 2 MBs ending at mb_xy 1 and frame end", got)
	}
	assertH264SliceDecodePCMHigh(t, dst, 0, 0, h264ReconstructIntraPCMSamples(1, bitDepth, 33))
	assertH264SliceDecodePCMHigh(t, dst, 1, 0, h264ReconstructIntraPCMSamples(1, bitDepth, 77))
	for _, mbXY := range []int{0, 1} {
		if m.MacroblockTyp[mbXY] != MBTypeIntraPCM || m.CBPTable[mbXY] != 0 || m.QScaleTable[mbXY] != 0 || m.SliceTable[mbXY] != 9 {
			t.Fatalf("tables[%d] type/cbp/q/slice = %#x/%#x/%d/%d", mbXY, m.MacroblockTyp[mbXY], m.CBPTable[mbXY], m.QScaleTable[mbXY], m.SliceTable[mbXY])
		}
	}
	if gb.bitsLeft() != 0 {
		t.Fatalf("bits left = %d, want 0", gb.bitsLeft())
	}
}

func TestDecodeCAVLCFrameSliceHighFrameMBAFFReconstructsFieldCodedPCMPair(t *testing.T) {
	const bitDepth = 10
	m, err := newMacroblockTables(1, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	sps := &SPS{BitDepthLuma: bitDepth, BitDepthChroma: bitDepth, ChromaFormatIDC: 1, FrameMBSOnlyFlag: 0, MBAFF: 1}
	pps := cavlcFlatQMulPPS()
	pps.SPS = sps
	sh := &SliceHeader{
		FirstMBAddr:      0,
		SliceType:        PictureTypeI,
		SliceTypeNoS:     PictureTypeI,
		PictureStructure: PictureFrame,
		PPS:              pps,
		SPS:              sps,
		QScale:           20,
		DeblockingFilter: 0,
	}
	dst := makeH264SliceDecodePictureHigh(1, 2, 1)
	pcmTop := h264ReconstructIntraPCMHigh(1, bitDepth, 37)
	pcmBottom := h264ReconstructIntraPCMHigh(1, bitDepth, 91)
	buf := append(cavlcMBAFFIntraPCMBytes(1, pcmTop), cavlcIntraPCMBytes(pcmBottom)...)
	gb := newBitReader(buf)

	got, err := m.decodeCAVLCFrameSliceHigh(&gb, dst, sh, h264FrameSliceDecodeInputHigh{SliceNum: 23})
	if err != nil {
		t.Fatalf("decode high frame-MBAFF cavlc slice failed: %v", err)
	}
	if got.Macroblocks != 2 || got.LastMBXY != m.MBStride || !got.EndOfSlice || !got.EndOfFrame {
		t.Fatalf("slice result = %+v, want MBAFF pair ending at mb_xy %d", got, m.MBStride)
	}
	top := *dst
	applySimpleFieldRefPlaneHigh(&top, PictureTopField)
	assertH264SliceDecodePCMHigh(t, &top, 0, 0, h264ReconstructIntraPCMSamples(1, bitDepth, 37))
	bottom := *dst
	applySimpleFieldRefPlaneHigh(&bottom, PictureBottomField)
	assertH264SliceDecodePCMHigh(t, &bottom, 0, 0, h264ReconstructIntraPCMSamples(1, bitDepth, 91))
	for _, mbXY := range []int{0, m.MBStride} {
		if m.MacroblockTyp[mbXY] != MBTypeIntraPCM|MBTypeInterlaced || m.CBPTable[mbXY] != 0 || m.QScaleTable[mbXY] != 0 || m.SliceTable[mbXY] != 23 {
			t.Fatalf("tables[%d] type/cbp/q/slice = %#x/%#x/%d/%d", mbXY, m.MacroblockTyp[mbXY], m.CBPTable[mbXY], m.QScaleTable[mbXY], m.SliceTable[mbXY])
		}
	}
	if gb.bitsLeft() != 0 {
		t.Fatalf("bits left = %d, want 0", gb.bitsLeft())
	}
}

func cavlcMBAFFIntraPCMBytes(fieldFlag int, pcm []byte) []byte {
	header := []byte{0x06, 0x80}
	if fieldFlag != 0 {
		header = []byte{0x86, 0x80}
	}
	out := append([]byte(nil), header...)
	return append(out, pcm...)
}

func TestDecodeCAVLCFrameSliceHigh14ReconstructsIntraPCM(t *testing.T) {
	const bitDepth = 14
	m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, bitDepth, 1, 1, false, PictureTypeI)
	pcm := h264ReconstructIntraPCMHigh(1, bitDepth, 61)
	gb := newBitReader(cavlcIntraPCMBytes(pcm))

	got, err := m.decodeCAVLCFrameSliceHigh(&gb, dst, sh, h264FrameSliceDecodeInputHigh{SliceNum: 14})
	if err != nil {
		t.Fatalf("decode high14 cavlc slice failed: %v", err)
	}
	if got.Macroblocks != 1 || got.LastMBXY != 0 || !got.EndOfSlice || !got.EndOfFrame {
		t.Fatalf("slice result = %+v, want one-MB frame end", got)
	}
	assertH264SliceDecodePCMHigh(t, dst, 0, 0, h264ReconstructIntraPCMSamples(1, bitDepth, 61))
	if m.MacroblockTyp[0] != MBTypeIntraPCM || m.CBPTable[0] != 0 || m.QScaleTable[0] != 0 || m.SliceTable[0] != 14 {
		t.Fatalf("tables type/cbp/q/slice = %#x/%#x/%d/%d", m.MacroblockTyp[0], m.CBPTable[0], m.QScaleTable[0], m.SliceTable[0])
	}
	if gb.bitsLeft() != 0 {
		t.Fatalf("bits left = %d, want 0", gb.bitsLeft())
	}
}

func TestDecodeFrameSliceDataHighDispatchesCAVLC(t *testing.T) {
	const bitDepth = 10
	m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, bitDepth, 1, 1, false, PictureTypeI)
	pcm := h264ReconstructIntraPCMHigh(1, bitDepth, 45)
	gb := newBitReader(cavlcIntraPCMBytes(pcm))

	got, err := m.decodeFrameSliceDataHigh(&gb, dst, sh, h264FrameSliceDecodeInputHigh{SliceNum: 7})
	if err != nil {
		t.Fatalf("decode high dispatched cavlc slice failed: %v", err)
	}
	if got.Macroblocks != 1 || !got.EndOfFrame || !got.EndOfSlice {
		t.Fatalf("slice result = %+v, want one-MB frame end", got)
	}
	assertH264SliceDecodePCMHigh(t, dst, 0, 0, h264ReconstructIntraPCMSamples(1, bitDepth, 45))
}

func TestDecodeCABACFrameSliceHighReconstructsIntraPCMAndEOS(t *testing.T) {
	for _, bitDepth := range []int32{10, 14} {
		t.Run(bitDepthName(bitDepth), func(t *testing.T) {
			m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, bitDepth, 1, 1, false, PictureTypeI)
			sh.PPS.CABAC = 1
			pcm := h264ReconstructIntraPCMHigh(1, int(bitDepth), 57)
			src := &scriptedCABACSource{
				bits:  []int{1},
				terms: []int{1, 1},
				pcm:   append([]byte(nil), pcm...),
			}

			got, err := m.decodeCABACFrameSliceHigh(src, dst, sh, h264FrameSliceDecodeInputHigh{SliceNum: 13})
			if err != nil {
				t.Fatalf("decode high cabac slice failed: %v", err)
			}
			if got.Macroblocks != 1 || got.LastMBXY != 0 || !got.EndOfSlice || !got.EndOfFrame {
				t.Fatalf("slice result = %+v, want one-MB frame end", got)
			}
			assertH264SliceDecodePCMHigh(t, dst, 0, 0, h264ReconstructIntraPCMSamples(1, int(bitDepth), 57))
			if m.MacroblockTyp[0] != MBTypeIntraPCM || m.CBPTable[0] != 0xf7ef || m.QScaleTable[0] != 0 || m.SliceTable[0] != 13 {
				t.Fatalf("tables type/cbp/q/slice = %#x/%#x/%d/%d", m.MacroblockTyp[0], m.CBPTable[0], m.QScaleTable[0], m.SliceTable[0])
			}
			if len(src.pcmReadSizes) != 1 || src.pcmReadSizes[0] != len(pcm) {
				t.Fatalf("pcm read sizes = %v, want [%d]", src.pcmReadSizes, len(pcm))
			}
			wantIndexes(t, src, []int{3})
		})
	}
}

func TestDecodeCAVLCFrameSliceHighReconstructsIntra4x4NoResidual(t *testing.T) {
	for _, bitDepth := range []int32{9, 10, 12, 14} {
		t.Run(bitDepthName(bitDepth), func(t *testing.T) {
			m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, bitDepth, 1, 1, false, PictureTypeI)
			gb := newBitReader(cavlcBitString("11111111111111111100100"))

			got, err := m.decodeCAVLCFrameSliceHigh(&gb, dst, sh, h264FrameSliceDecodeInputHigh{SliceNum: 17})
			if err != nil {
				t.Fatalf("decode high cavlc intra4x4 slice failed: %v", err)
			}
			if got.Macroblocks != 1 || got.LastMBXY != 0 || !got.EndOfSlice || !got.EndOfFrame {
				t.Fatalf("slice result = %+v, want one-MB frame end", got)
			}
			assertH264ConstantBlockHigh(t, "cavlc high intra4x4 y", dst.Y, 0, dst.LumaStride, 16, 16, 1<<(bitDepth-1))
			assertH264ConstantBlockHigh(t, "cavlc high intra4x4 cb", dst.Cb, 0, dst.ChromaStride, 8, 8, 1<<(bitDepth-1))
			assertH264ConstantBlockHigh(t, "cavlc high intra4x4 cr", dst.Cr, 0, dst.ChromaStride, 8, 8, 1<<(bitDepth-1))
			if m.MacroblockTyp[0] != MBTypeIntra4x4 || m.CBPTable[0] != 0 || m.QScaleTable[0] != 20 || m.SliceTable[0] != 17 {
				t.Fatalf("tables type/cbp/q/slice = %#x/%#x/%d/%d", m.MacroblockTyp[0], m.CBPTable[0], m.QScaleTable[0], m.SliceTable[0])
			}
		})
	}
}

func TestDecodeCAVLCFrameSliceHighReconstructsIntra16x16NoResidual(t *testing.T) {
	for _, bitDepth := range []int32{12, 14} {
		t.Run(bitDepthName(bitDepth), func(t *testing.T) {
			m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, bitDepth, 1, 1, false, PictureTypeI)
			gb := newBitReader(cavlcBitString("00100111"))

			got, err := m.decodeCAVLCFrameSliceHigh(&gb, dst, sh, h264FrameSliceDecodeInputHigh{SliceNum: 21})
			if err != nil {
				t.Fatalf("decode high cavlc intra16x16 slice failed: %v", err)
			}
			if got.Macroblocks != 1 || got.LastMBXY != 0 || !got.EndOfSlice || !got.EndOfFrame {
				t.Fatalf("slice result = %+v, want one-MB frame end", got)
			}
			assertH264ConstantBlockHigh(t, "cavlc high intra16x16 y", dst.Y, 0, dst.LumaStride, 16, 16, 1<<(bitDepth-1))
			assertH264ConstantBlockHigh(t, "cavlc high intra16x16 cb", dst.Cb, 0, dst.ChromaStride, 8, 8, 1<<(bitDepth-1))
			assertH264ConstantBlockHigh(t, "cavlc high intra16x16 cr", dst.Cr, 0, dst.ChromaStride, 8, 8, 1<<(bitDepth-1))
			if m.MacroblockTyp[0] != MBTypeIntra16x16 || m.CBPTable[0] != 0 || m.QScaleTable[0] != 20 || m.SliceTable[0] != 21 {
				t.Fatalf("tables type/cbp/q/slice = %#x/%#x/%d/%d", m.MacroblockTyp[0], m.CBPTable[0], m.QScaleTable[0], m.SliceTable[0])
			}
			if gb.bitPos != 8 {
				t.Fatalf("consumed %d bits, want 8", gb.bitPos)
			}
		})
	}
}

func TestDecodeCAVLCFrameSliceHighReconstructsIntra16x16LumaDCResidual(t *testing.T) {
	for _, bitDepth := range []int32{12, 14} {
		t.Run(bitDepthName(bitDepth), func(t *testing.T) {
			m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, bitDepth, 1, 1, false, PictureTypeI)
			gb := newBitReader(cavlcBitString("00100110101"))

			got, err := m.decodeCAVLCFrameSliceHigh(&gb, dst, sh, h264FrameSliceDecodeInputHigh{SliceNum: 25})
			if err != nil {
				t.Fatalf("decode high cavlc intra16x16 luma-DC residual slice failed: %v", err)
			}
			if got.Macroblocks != 1 || got.LastMBXY != 0 || !got.EndOfSlice || !got.EndOfFrame {
				t.Fatalf("slice result = %+v, want one-MB frame end", got)
			}
			want := h264HighIntra16x16LumaDCResidualExpected(t, int(bitDepth), sh.PPS, int(sh.QScale))
			assertH264RowsHigh(t, "cavlc high intra16x16 luma-DC y", dst.Y, 0, dst.LumaStride, 16, 16, want.Y, want.LumaStride)
			assertH264RowsHigh(t, "cavlc high intra16x16 luma-DC cb", dst.Cb, 0, dst.ChromaStride, 8, 8, want.Cb, want.ChromaStride)
			assertH264RowsHigh(t, "cavlc high intra16x16 luma-DC cr", dst.Cr, 0, dst.ChromaStride, 8, 8, want.Cr, want.ChromaStride)
			if m.MacroblockTyp[0] != MBTypeIntra16x16 || m.CBPTable[0] != 0 || m.QScaleTable[0] != 20 || m.SliceTable[0] != 25 {
				t.Fatalf("tables type/cbp/q/slice = %#x/%#x/%d/%d", m.MacroblockTyp[0], m.CBPTable[0], m.QScaleTable[0], m.SliceTable[0])
			}
			if gb.bitPos != 11 {
				t.Fatalf("consumed %d bits, want 11", gb.bitPos)
			}
		})
	}
}

func TestDecodeCAVLCFrameSliceHighReconstructsIntra16x16ChromaDCResidual(t *testing.T) {
	for _, bitDepth := range []int32{12, 14} {
		t.Run(bitDepthName(bitDepth), func(t *testing.T) {
			m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, bitDepth, 1, 1, false, PictureTypeI)
			gb := newBitReader(cavlcBitString("000100011110101"))

			got, err := m.decodeCAVLCFrameSliceHigh(&gb, dst, sh, h264FrameSliceDecodeInputHigh{SliceNum: 29})
			if err != nil {
				t.Fatalf("decode high cavlc intra16x16 chroma-DC residual slice failed: %v", err)
			}
			if got.Macroblocks != 1 || got.LastMBXY != 0 || !got.EndOfSlice || !got.EndOfFrame {
				t.Fatalf("slice result = %+v, want one-MB frame end", got)
			}
			want := h264HighIntra16x16ChromaDCResidualExpected(t, int(bitDepth), sh.PPS, int(sh.QScale))
			assertH264RowsHigh(t, "cavlc high intra16x16 chroma-DC y", dst.Y, 0, dst.LumaStride, 16, 16, want.Y, want.LumaStride)
			assertH264RowsHigh(t, "cavlc high intra16x16 chroma-DC cb", dst.Cb, 0, dst.ChromaStride, 8, 8, want.Cb, want.ChromaStride)
			assertH264RowsHigh(t, "cavlc high intra16x16 chroma-DC cr", dst.Cr, 0, dst.ChromaStride, 8, 8, want.Cr, want.ChromaStride)
			if m.MacroblockTyp[0] != MBTypeIntra16x16 || m.CBPTable[0] != 0x10 || m.QScaleTable[0] != 20 || m.SliceTable[0] != 29 {
				t.Fatalf("tables type/cbp/q/slice = %#x/%#x/%d/%d", m.MacroblockTyp[0], m.CBPTable[0], m.QScaleTable[0], m.SliceTable[0])
			}
			if gb.bitPos != 15 {
				t.Fatalf("consumed %d bits, want 15", gb.bitPos)
			}
		})
	}
}

func TestDecodeCAVLCFrameSliceHighReconstructsIntra16x16ChromaACResidual(t *testing.T) {
	for _, bitDepth := range []int32{12, 14} {
		t.Run(bitDepthName(bitDepth), func(t *testing.T) {
			m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, bitDepth, 1, 1, false, PictureTypeI)
			gb := newBitReader(cavlcBitString("0001100111010101011111111"))

			got, err := m.decodeCAVLCFrameSliceHigh(&gb, dst, sh, h264FrameSliceDecodeInputHigh{SliceNum: 41})
			if err != nil {
				t.Fatalf("decode high cavlc intra16x16 chroma-AC residual slice failed: %v", err)
			}
			if got.Macroblocks != 1 || got.LastMBXY != 0 || !got.EndOfSlice || !got.EndOfFrame {
				t.Fatalf("slice result = %+v, want one-MB frame end", got)
			}
			want := h264HighIntra16x16ChromaACResidualExpected(t, int(bitDepth), sh.PPS, int(sh.QScale))
			assertH264RowsHigh(t, "cavlc high intra16x16 chroma-AC y", dst.Y, 0, dst.LumaStride, 16, 16, want.Y, want.LumaStride)
			assertH264RowsHigh(t, "cavlc high intra16x16 chroma-AC cb", dst.Cb, 0, dst.ChromaStride, 8, 8, want.Cb, want.ChromaStride)
			assertH264RowsHigh(t, "cavlc high intra16x16 chroma-AC cr", dst.Cr, 0, dst.ChromaStride, 8, 8, want.Cr, want.ChromaStride)
			if m.MacroblockTyp[0] != MBTypeIntra16x16 || m.CBPTable[0] != 0x20 || m.QScaleTable[0] != 20 || m.SliceTable[0] != 41 {
				t.Fatalf("tables type/cbp/q/slice = %#x/%#x/%d/%d", m.MacroblockTyp[0], m.CBPTable[0], m.QScaleTable[0], m.SliceTable[0])
			}
			if gb.bitPos != 25 {
				t.Fatalf("consumed %d bits, want 25", gb.bitPos)
			}
		})
	}
}

func TestDecodeCAVLCFrameSliceHighReconstructsIntra16x16ChromaDCACResidual(t *testing.T) {
	for _, bitDepth := range []int32{12, 14} {
		t.Run(bitDepthName(bitDepth), func(t *testing.T) {
			m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, bitDepth, 1, 1, false, PictureTypeI)
			gb := newBitReader(cavlcBitString("00011001111010101011111111"))

			got, err := m.decodeCAVLCFrameSliceHigh(&gb, dst, sh, h264FrameSliceDecodeInputHigh{SliceNum: 45})
			if err != nil {
				t.Fatalf("decode high cavlc intra16x16 chroma-DC/AC residual slice failed: %v", err)
			}
			if got.Macroblocks != 1 || got.LastMBXY != 0 || !got.EndOfSlice || !got.EndOfFrame {
				t.Fatalf("slice result = %+v, want one-MB frame end", got)
			}
			want := h264HighIntra16x16ChromaDCACResidualExpected(t, int(bitDepth), sh.PPS, int(sh.QScale))
			assertH264RowsHigh(t, "cavlc high intra16x16 chroma-DC/AC y", dst.Y, 0, dst.LumaStride, 16, 16, want.Y, want.LumaStride)
			assertH264RowsHigh(t, "cavlc high intra16x16 chroma-DC/AC cb", dst.Cb, 0, dst.ChromaStride, 8, 8, want.Cb, want.ChromaStride)
			assertH264RowsHigh(t, "cavlc high intra16x16 chroma-DC/AC cr", dst.Cr, 0, dst.ChromaStride, 8, 8, want.Cr, want.ChromaStride)
			if m.MacroblockTyp[0] != MBTypeIntra16x16 || m.CBPTable[0] != 0x20 || m.QScaleTable[0] != 20 || m.SliceTable[0] != 45 {
				t.Fatalf("tables type/cbp/q/slice = %#x/%#x/%d/%d", m.MacroblockTyp[0], m.CBPTable[0], m.QScaleTable[0], m.SliceTable[0])
			}
			if gb.bitPos != 26 {
				t.Fatalf("consumed %d bits, want 26", gb.bitPos)
			}
		})
	}
}

func TestDecodeCAVLCFrameSliceHighReconstructsIntra16x16LumaChromaResidual(t *testing.T) {
	for _, bitDepth := range []int32{12, 14} {
		t.Run(bitDepthName(bitDepth), func(t *testing.T) {
			m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, bitDepth, 1, 1, false, PictureTypeI)
			gb := newBitReader(cavlcBitString("0000110001101010101" + strings.Repeat("1", 15) + "1010101011111111"))

			got, err := m.decodeCAVLCFrameSliceHigh(&gb, dst, sh, h264FrameSliceDecodeInputHigh{SliceNum: 49})
			if err != nil {
				t.Fatalf("decode high cavlc intra16x16 luma/chroma residual slice failed: %v", err)
			}
			if got.Macroblocks != 1 || got.LastMBXY != 0 || !got.EndOfSlice || !got.EndOfFrame {
				t.Fatalf("slice result = %+v, want one-MB frame end", got)
			}
			want := h264HighIntra16x16LumaChromaResidualExpected(t, int(bitDepth), sh.PPS, int(sh.QScale))
			assertH264RowsHigh(t, "cavlc high intra16x16 luma/chroma y", dst.Y, 0, dst.LumaStride, 16, 16, want.Y, want.LumaStride)
			assertH264RowsHigh(t, "cavlc high intra16x16 luma/chroma cb", dst.Cb, 0, dst.ChromaStride, 8, 8, want.Cb, want.ChromaStride)
			assertH264RowsHigh(t, "cavlc high intra16x16 luma/chroma cr", dst.Cr, 0, dst.ChromaStride, 8, 8, want.Cr, want.ChromaStride)
			if m.MacroblockTyp[0] != MBTypeIntra16x16 || m.CBPTable[0] != 0xf02f || m.QScaleTable[0] != 20 || m.SliceTable[0] != 49 {
				t.Fatalf("tables type/cbp/q/slice = %#x/%#x/%d/%d", m.MacroblockTyp[0], m.CBPTable[0], m.QScaleTable[0], m.SliceTable[0])
			}
			if gb.bitPos != 50 {
				t.Fatalf("consumed %d bits, want 50", gb.bitPos)
			}
		})
	}
}

func TestDecodeCAVLCFrameSliceHighReconstructsIntra16x16LumaACResidual(t *testing.T) {
	for _, bitDepth := range []int32{12, 14} {
		t.Run(bitDepthName(bitDepth), func(t *testing.T) {
			m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, bitDepth, 1, 1, false, PictureTypeI)
			gb := newBitReader(cavlcBitString("0000100001110101" + strings.Repeat("1", 15)))

			got, err := m.decodeCAVLCFrameSliceHigh(&gb, dst, sh, h264FrameSliceDecodeInputHigh{SliceNum: 33})
			if err != nil {
				t.Fatalf("decode high cavlc intra16x16 luma-AC residual slice failed: %v", err)
			}
			if got.Macroblocks != 1 || got.LastMBXY != 0 || !got.EndOfSlice || !got.EndOfFrame {
				t.Fatalf("slice result = %+v, want one-MB frame end", got)
			}
			want := h264HighIntra16x16LumaACResidualExpected(t, int(bitDepth))
			assertH264RowsHigh(t, "cavlc high intra16x16 luma-AC y", dst.Y, 0, dst.LumaStride, 16, 16, want.Y, want.LumaStride)
			assertH264RowsHigh(t, "cavlc high intra16x16 luma-AC cb", dst.Cb, 0, dst.ChromaStride, 8, 8, want.Cb, want.ChromaStride)
			assertH264RowsHigh(t, "cavlc high intra16x16 luma-AC cr", dst.Cr, 0, dst.ChromaStride, 8, 8, want.Cr, want.ChromaStride)
			if m.MacroblockTyp[0] != MBTypeIntra16x16 || m.CBPTable[0] != 0xf00f || m.QScaleTable[0] != 20 || m.SliceTable[0] != 33 {
				t.Fatalf("tables type/cbp/q/slice = %#x/%#x/%d/%d", m.MacroblockTyp[0], m.CBPTable[0], m.QScaleTable[0], m.SliceTable[0])
			}
			if gb.bitPos != 31 {
				t.Fatalf("consumed %d bits, want 31", gb.bitPos)
			}
		})
	}
}

func TestDecodeCAVLCFrameSliceHighReconstructsIntra16x16LumaDCACResidual(t *testing.T) {
	for _, bitDepth := range []int32{12, 14} {
		t.Run(bitDepthName(bitDepth), func(t *testing.T) {
			m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, bitDepth, 1, 1, false, PictureTypeI)
			gb := newBitReader(cavlcBitString("0000100001101010101" + strings.Repeat("1", 15)))

			got, err := m.decodeCAVLCFrameSliceHigh(&gb, dst, sh, h264FrameSliceDecodeInputHigh{SliceNum: 37})
			if err != nil {
				t.Fatalf("decode high cavlc intra16x16 luma-DC/AC residual slice failed: %v", err)
			}
			if got.Macroblocks != 1 || got.LastMBXY != 0 || !got.EndOfSlice || !got.EndOfFrame {
				t.Fatalf("slice result = %+v, want one-MB frame end", got)
			}
			want := h264HighIntra16x16LumaDCACResidualExpected(t, int(bitDepth), sh.PPS, int(sh.QScale))
			assertH264RowsHigh(t, "cavlc high intra16x16 luma-DC/AC y", dst.Y, 0, dst.LumaStride, 16, 16, want.Y, want.LumaStride)
			assertH264RowsHigh(t, "cavlc high intra16x16 luma-DC/AC cb", dst.Cb, 0, dst.ChromaStride, 8, 8, want.Cb, want.ChromaStride)
			assertH264RowsHigh(t, "cavlc high intra16x16 luma-DC/AC cr", dst.Cr, 0, dst.ChromaStride, 8, 8, want.Cr, want.ChromaStride)
			if m.MacroblockTyp[0] != MBTypeIntra16x16 || m.CBPTable[0] != 0xf00f || m.QScaleTable[0] != 20 || m.SliceTable[0] != 37 {
				t.Fatalf("tables type/cbp/q/slice = %#x/%#x/%d/%d", m.MacroblockTyp[0], m.CBPTable[0], m.QScaleTable[0], m.SliceTable[0])
			}
			if gb.bitPos != 34 {
				t.Fatalf("consumed %d bits, want 34", gb.bitPos)
			}
		})
	}
}

func TestDecodeCABACFrameSliceHighReconstructsIntra4x4NoResidual(t *testing.T) {
	for _, bitDepth := range []int32{10, 12, 14} {
		t.Run(bitDepthName(bitDepth), func(t *testing.T) {
			m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, bitDepth, 1, 1, false, PictureTypeI)
			sh.PPS.CABAC = 1
			src := &scriptedCABACSource{
				bits: append(append([]int{0}, repeatCABACBits(16, 1)...), []int{
					0,
					0, 0, 0, 0,
					0,
				}...),
				terms: []int{1},
			}

			got, err := m.decodeCABACFrameSliceHigh(src, dst, sh, h264FrameSliceDecodeInputHigh{SliceNum: 19})
			if err != nil {
				t.Fatalf("decode high cabac intra4x4 slice failed: %v", err)
			}
			if got.Macroblocks != 1 || got.LastMBXY != 0 || !got.EndOfSlice || !got.EndOfFrame {
				t.Fatalf("slice result = %+v, want one-MB frame end", got)
			}
			assertH264ConstantBlockHigh(t, "cabac high intra4x4 y", dst.Y, 0, dst.LumaStride, 16, 16, 1<<(bitDepth-1))
			assertH264ConstantBlockHigh(t, "cabac high intra4x4 cb", dst.Cb, 0, dst.ChromaStride, 8, 8, 1<<(bitDepth-1))
			assertH264ConstantBlockHigh(t, "cabac high intra4x4 cr", dst.Cr, 0, dst.ChromaStride, 8, 8, 1<<(bitDepth-1))
			if m.MacroblockTyp[0] != MBTypeIntra4x4 || m.CBPTable[0] != 0 || m.QScaleTable[0] != 20 || m.SliceTable[0] != 19 {
				t.Fatalf("tables type/cbp/q/slice = %#x/%#x/%d/%d", m.MacroblockTyp[0], m.CBPTable[0], m.QScaleTable[0], m.SliceTable[0])
			}
			if len(src.pcmReadSizes) != 0 {
				t.Fatalf("pcm reads = %v, want none", src.pcmReadSizes)
			}
			wantIndexes(t, src, append(append([]int{3}, repeatCABACBits(16, 68)...), []int{64, 73, 74, 75, 76, 77}...))
		})
	}
}

func TestDecodeCABACFrameSliceHighReconstructsIntra16x16NoResidual(t *testing.T) {
	for _, bitDepth := range []int32{12, 14} {
		t.Run(bitDepthName(bitDepth), func(t *testing.T) {
			m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, bitDepth, 1, 1, false, PictureTypeI)
			sh.PPS.CABAC = 1
			src := &scriptedCABACSource{
				bits:  []int{1, 0, 0, 1, 0, 0, 0, 0},
				terms: []int{0, 1},
			}

			got, err := m.decodeCABACFrameSliceHigh(src, dst, sh, h264FrameSliceDecodeInputHigh{SliceNum: 23})
			if err != nil {
				t.Fatalf("decode high cabac intra16x16 slice failed: %v", err)
			}
			if got.Macroblocks != 1 || got.LastMBXY != 0 || !got.EndOfSlice || !got.EndOfFrame {
				t.Fatalf("slice result = %+v, want one-MB frame end", got)
			}
			assertH264ConstantBlockHigh(t, "cabac high intra16x16 y", dst.Y, 0, dst.LumaStride, 16, 16, 1<<(bitDepth-1))
			assertH264ConstantBlockHigh(t, "cabac high intra16x16 cb", dst.Cb, 0, dst.ChromaStride, 8, 8, 1<<(bitDepth-1))
			assertH264ConstantBlockHigh(t, "cabac high intra16x16 cr", dst.Cr, 0, dst.ChromaStride, 8, 8, 1<<(bitDepth-1))
			if m.MacroblockTyp[0] != MBTypeIntra16x16 || m.CBPTable[0] != 0 || m.QScaleTable[0] != 20 || m.SliceTable[0] != 23 {
				t.Fatalf("tables type/cbp/q/slice = %#x/%#x/%d/%d", m.MacroblockTyp[0], m.CBPTable[0], m.QScaleTable[0], m.SliceTable[0])
			}
			if len(src.pcmReadSizes) != 0 {
				t.Fatalf("pcm reads = %v, want none", src.pcmReadSizes)
			}
			wantIndexes(t, src, []int{3, 6, 7, 9, 10, 64, 60, 88})
		})
	}
}

func TestDecodeCABACFrameSliceHigh12ReconstructsIntra16x16LumaDCResidual(t *testing.T) {
	const bitDepth = 12
	m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, bitDepth, 1, 1, false, PictureTypeI)
	sh.PPS.CABAC = 1
	src := &scriptedCABACSource{
		bits:  []int{1, 0, 0, 1, 0, 0, 0, 1, 1, 1, 0},
		signs: []int32{1},
		terms: []int{0, 1},
	}

	got, err := m.decodeCABACFrameSliceHigh(src, dst, sh, h264FrameSliceDecodeInputHigh{SliceNum: 27})
	if err != nil {
		t.Fatalf("decode high cabac intra16x16 luma-DC residual slice failed: %v", err)
	}
	if got.Macroblocks != 1 || got.LastMBXY != 0 || !got.EndOfSlice || !got.EndOfFrame {
		t.Fatalf("slice result = %+v, want one-MB frame end", got)
	}
	want := h264High12Intra16x16LumaDCResidualExpected(t, sh.PPS, int(sh.QScale))
	assertH264RowsHigh(t, "cabac high12 intra16x16 luma-DC y", dst.Y, 0, dst.LumaStride, 16, 16, want.Y, want.LumaStride)
	assertH264RowsHigh(t, "cabac high12 intra16x16 luma-DC cb", dst.Cb, 0, dst.ChromaStride, 8, 8, want.Cb, want.ChromaStride)
	assertH264RowsHigh(t, "cabac high12 intra16x16 luma-DC cr", dst.Cr, 0, dst.ChromaStride, 8, 8, want.Cr, want.ChromaStride)
	if m.MacroblockTyp[0] != MBTypeIntra16x16 || m.CBPTable[0] != 0x100 || m.QScaleTable[0] != 20 || m.SliceTable[0] != 27 {
		t.Fatalf("tables type/cbp/q/slice = %#x/%#x/%d/%d", m.MacroblockTyp[0], m.CBPTable[0], m.QScaleTable[0], m.SliceTable[0])
	}
	if len(src.pcmReadSizes) != 0 {
		t.Fatalf("pcm reads = %v, want none", src.pcmReadSizes)
	}
	wantIndexes(t, src, []int{3, 6, 7, 9, 10, 64, 60, 88, 105, 166, 228})
}

func TestDecodeCABACFrameSliceHigh12ReconstructsIntra16x16ChromaDCResidual(t *testing.T) {
	const bitDepth = 12
	m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, bitDepth, 1, 1, false, PictureTypeI)
	sh.PPS.CABAC = 1
	src := &scriptedCABACSource{
		bits:  []int{1, 0, 1, 0, 1, 0, 0, 0, 0, 1, 1, 1, 0, 0},
		signs: []int32{1},
		terms: []int{0, 1},
	}

	got, err := m.decodeCABACFrameSliceHigh(src, dst, sh, h264FrameSliceDecodeInputHigh{SliceNum: 31})
	if err != nil {
		t.Fatalf("decode high cabac intra16x16 chroma-DC residual slice failed: %v", err)
	}
	if got.Macroblocks != 1 || got.LastMBXY != 0 || !got.EndOfSlice || !got.EndOfFrame {
		t.Fatalf("slice result = %+v, want one-MB frame end", got)
	}
	want := h264High12Intra16x16ChromaDCResidualExpected(t, sh.PPS, int(sh.QScale))
	assertH264RowsHigh(t, "cabac high12 intra16x16 chroma-DC y", dst.Y, 0, dst.LumaStride, 16, 16, want.Y, want.LumaStride)
	assertH264RowsHigh(t, "cabac high12 intra16x16 chroma-DC cb", dst.Cb, 0, dst.ChromaStride, 8, 8, want.Cb, want.ChromaStride)
	assertH264RowsHigh(t, "cabac high12 intra16x16 chroma-DC cr", dst.Cr, 0, dst.ChromaStride, 8, 8, want.Cr, want.ChromaStride)
	if m.MacroblockTyp[0] != MBTypeIntra16x16 || m.CBPTable[0] != 0x50 || m.QScaleTable[0] != 20 || m.SliceTable[0] != 31 {
		t.Fatalf("tables type/cbp/q/slice = %#x/%#x/%d/%d", m.MacroblockTyp[0], m.CBPTable[0], m.QScaleTable[0], m.SliceTable[0])
	}
	if len(src.pcmReadSizes) != 0 {
		t.Fatalf("pcm reads = %v, want none", src.pcmReadSizes)
	}
	wantIndexes(t, src, []int{3, 6, 7, 8, 9, 10, 64, 60, 88, 100, 149, 210, 258, 100})
}

func TestDecodeCABACFrameSliceHigh12ReconstructsIntra16x16ChromaACResidual(t *testing.T) {
	const bitDepth = 12
	m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, bitDepth, 1, 1, false, PictureTypeI)
	sh.PPS.CABAC = 1
	src := &scriptedCABACSource{
		bits: []int{
			1, 0, 1, 1, 1, 0,
			0,
			0,
			0,
			0, 0,
			1, 1, 1, 0,
			0, 0, 0, 0, 0, 0, 0,
		},
		signs: []int32{64},
		terms: []int{0, 1},
	}

	got, err := m.decodeCABACFrameSliceHigh(src, dst, sh, h264FrameSliceDecodeInputHigh{SliceNum: 43})
	if err != nil {
		t.Fatalf("decode high cabac intra16x16 chroma-AC residual slice failed: %v", err)
	}
	if got.Macroblocks != 1 || got.LastMBXY != 0 || !got.EndOfSlice || !got.EndOfFrame {
		t.Fatalf("slice result = %+v, want one-MB frame end", got)
	}
	want := h264High12Intra16x16ChromaACResidualExpected(t, sh.PPS, int(sh.QScale))
	assertH264RowsHigh(t, "cabac high12 intra16x16 chroma-AC y", dst.Y, 0, dst.LumaStride, 16, 16, want.Y, want.LumaStride)
	assertH264RowsHigh(t, "cabac high12 intra16x16 chroma-AC cb", dst.Cb, 0, dst.ChromaStride, 8, 8, want.Cb, want.ChromaStride)
	assertH264RowsHigh(t, "cabac high12 intra16x16 chroma-AC cr", dst.Cr, 0, dst.ChromaStride, 8, 8, want.Cr, want.ChromaStride)
	if m.MacroblockTyp[0] != MBTypeIntra16x16 || m.CBPTable[0] != 0x20 || m.QScaleTable[0] != 20 || m.SliceTable[0] != 43 {
		t.Fatalf("tables type/cbp/q/slice = %#x/%#x/%d/%d", m.MacroblockTyp[0], m.CBPTable[0], m.QScaleTable[0], m.SliceTable[0])
	}
	if len(src.pcmReadSizes) != 0 {
		t.Fatalf("pcm reads = %v, want none", src.pcmReadSizes)
	}
	wantIndexes(t, src, []int{3, 6, 7, 8, 9, 10, 64, 60, 88, 100, 100, 104, 152, 213, 267, 104, 104, 101, 104, 103, 102, 101})
}

func TestDecodeCABACFrameSliceHigh12ReconstructsIntra16x16ChromaDCACResidual(t *testing.T) {
	const bitDepth = 12
	m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, bitDepth, 1, 1, false, PictureTypeI)
	sh.PPS.CABAC = 1
	src := &scriptedCABACSource{
		bits: []int{
			1, 0, 1, 1, 1, 0,
			0,
			0,
			0,
			1, 1, 1, 0,
			0,
			1, 1, 1, 0,
			0, 0, 0, 0, 0, 0, 0,
		},
		signs: []int32{1, 64},
		terms: []int{0, 1},
	}

	got, err := m.decodeCABACFrameSliceHigh(src, dst, sh, h264FrameSliceDecodeInputHigh{SliceNum: 47})
	if err != nil {
		t.Fatalf("decode high cabac intra16x16 chroma-DC/AC residual slice failed: %v", err)
	}
	if got.Macroblocks != 1 || got.LastMBXY != 0 || !got.EndOfSlice || !got.EndOfFrame {
		t.Fatalf("slice result = %+v, want one-MB frame end", got)
	}
	want := h264High12Intra16x16ChromaDCACResidualExpected(t, sh.PPS, int(sh.QScale))
	assertH264RowsHigh(t, "cabac high12 intra16x16 chroma-DC/AC y", dst.Y, 0, dst.LumaStride, 16, 16, want.Y, want.LumaStride)
	assertH264RowsHigh(t, "cabac high12 intra16x16 chroma-DC/AC cb", dst.Cb, 0, dst.ChromaStride, 8, 8, want.Cb, want.ChromaStride)
	assertH264RowsHigh(t, "cabac high12 intra16x16 chroma-DC/AC cr", dst.Cr, 0, dst.ChromaStride, 8, 8, want.Cr, want.ChromaStride)
	if m.MacroblockTyp[0] != MBTypeIntra16x16 || m.CBPTable[0] != 0x60 || m.QScaleTable[0] != 20 || m.SliceTable[0] != 47 {
		t.Fatalf("tables type/cbp/q/slice = %#x/%#x/%d/%d", m.MacroblockTyp[0], m.CBPTable[0], m.QScaleTable[0], m.SliceTable[0])
	}
	if len(src.pcmReadSizes) != 0 {
		t.Fatalf("pcm reads = %v, want none", src.pcmReadSizes)
	}
	wantIndexes(t, src, []int{3, 6, 7, 8, 9, 10, 64, 60, 88, 100, 149, 210, 258, 100, 104, 152, 213, 267, 104, 104, 101, 104, 103, 102, 101})
}

func TestDecodeCABACFrameSliceHigh12ReconstructsIntra16x16LumaChromaResidual(t *testing.T) {
	const bitDepth = 12
	m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, bitDepth, 1, 1, false, PictureTypeI)
	sh.PPS.CABAC = 1
	src := &scriptedCABACSource{
		bits: append(append([]int{
			1, 1, 1, 1, 1, 0,
			0,
			0,
			1, 1, 1, 0,
			1, 1, 1, 0,
		}, repeatCABACBits(15, 0)...), []int{
			1, 1, 1, 0,
			0,
			1, 1, 1, 0,
			0, 0, 0, 0, 0, 0, 0,
		}...),
		signs: []int32{1, 64, 1, 64},
		terms: []int{0, 1},
	}

	got, err := m.decodeCABACFrameSliceHigh(src, dst, sh, h264FrameSliceDecodeInputHigh{SliceNum: 51})
	if err != nil {
		t.Fatalf("decode high cabac intra16x16 luma/chroma residual slice failed: %v", err)
	}
	if got.Macroblocks != 1 || got.LastMBXY != 0 || !got.EndOfSlice || !got.EndOfFrame {
		t.Fatalf("slice result = %+v, want one-MB frame end", got)
	}
	want := h264High12Intra16x16LumaChromaResidualExpected(t, sh.PPS, int(sh.QScale))
	assertH264RowsHigh(t, "cabac high12 intra16x16 luma/chroma y", dst.Y, 0, dst.LumaStride, 16, 16, want.Y, want.LumaStride)
	assertH264RowsHigh(t, "cabac high12 intra16x16 luma/chroma cb", dst.Cb, 0, dst.ChromaStride, 8, 8, want.Cb, want.ChromaStride)
	assertH264RowsHigh(t, "cabac high12 intra16x16 luma/chroma cr", dst.Cr, 0, dst.ChromaStride, 8, 8, want.Cr, want.ChromaStride)
	if m.MacroblockTyp[0] != MBTypeIntra16x16 || m.CBPTable[0] != 0x16f || m.QScaleTable[0] != 20 || m.SliceTable[0] != 51 {
		t.Fatalf("tables type/cbp/q/slice = %#x/%#x/%d/%d", m.MacroblockTyp[0], m.CBPTable[0], m.QScaleTable[0], m.SliceTable[0])
	}
	if len(src.pcmReadSizes) != 0 {
		t.Fatalf("pcm reads = %v, want none", src.pcmReadSizes)
	}
	wantIndexes(t, src, append([]int{3, 6, 7, 8, 9, 10, 64, 60, 88, 105, 166, 228, 92, 120, 181, 238, 92, 92, 89, 91, 91, 89, 89, 90, 89, 90, 89, 89, 89, 89, 89}, []int{100, 149, 210, 258, 100, 104, 152, 213, 267, 104, 104, 101, 104, 103, 102, 101}...))
}

func TestDecodeCABACFrameSliceHigh12ReconstructsIntra16x16LumaACResidual(t *testing.T) {
	const bitDepth = 12
	m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, bitDepth, 1, 1, false, PictureTypeI)
	sh.PPS.CABAC = 1
	src := &scriptedCABACSource{
		bits: append([]int{
			1, 1, 0, 1, 0,
			0,
			0,
			0,
			1, 1, 1, 0,
		}, repeatCABACBits(15, 0)...),
		signs: []int32{64},
		terms: []int{0, 1},
	}

	got, err := m.decodeCABACFrameSliceHigh(src, dst, sh, h264FrameSliceDecodeInputHigh{SliceNum: 35})
	if err != nil {
		t.Fatalf("decode high cabac intra16x16 luma-AC residual slice failed: %v", err)
	}
	if got.Macroblocks != 1 || got.LastMBXY != 0 || !got.EndOfSlice || !got.EndOfFrame {
		t.Fatalf("slice result = %+v, want one-MB frame end", got)
	}
	want := h264High12Intra16x16LumaACResidualExpected(t)
	assertH264RowsHigh(t, "cabac high12 intra16x16 luma-AC y", dst.Y, 0, dst.LumaStride, 16, 16, want.Y, want.LumaStride)
	assertH264RowsHigh(t, "cabac high12 intra16x16 luma-AC cb", dst.Cb, 0, dst.ChromaStride, 8, 8, want.Cb, want.ChromaStride)
	assertH264RowsHigh(t, "cabac high12 intra16x16 luma-AC cr", dst.Cr, 0, dst.ChromaStride, 8, 8, want.Cr, want.ChromaStride)
	if m.MacroblockTyp[0] != MBTypeIntra16x16 || m.CBPTable[0] != 0x0f || m.QScaleTable[0] != 20 || m.SliceTable[0] != 35 {
		t.Fatalf("tables type/cbp/q/slice = %#x/%#x/%d/%d", m.MacroblockTyp[0], m.CBPTable[0], m.QScaleTable[0], m.SliceTable[0])
	}
	if len(src.pcmReadSizes) != 0 {
		t.Fatalf("pcm reads = %v, want none", src.pcmReadSizes)
	}
	wantIndexes(t, src, []int{3, 6, 7, 9, 10, 64, 60, 88, 92, 120, 181, 238, 92, 92, 89, 91, 91, 89, 89, 90, 89, 90, 89, 89, 89, 89, 89})
}

func TestDecodeCABACFrameSliceHigh12ReconstructsIntra16x16LumaDCACResidual(t *testing.T) {
	const bitDepth = 12
	m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, bitDepth, 1, 1, false, PictureTypeI)
	sh.PPS.CABAC = 1
	src := &scriptedCABACSource{
		bits: append([]int{
			1, 1, 0, 1, 0,
			0,
			0,
			1, 1, 1, 0,
			1, 1, 1, 0,
		}, repeatCABACBits(15, 0)...),
		signs: []int32{1, 64},
		terms: []int{0, 1},
	}

	got, err := m.decodeCABACFrameSliceHigh(src, dst, sh, h264FrameSliceDecodeInputHigh{SliceNum: 39})
	if err != nil {
		t.Fatalf("decode high cabac intra16x16 luma-DC/AC residual slice failed: %v", err)
	}
	if got.Macroblocks != 1 || got.LastMBXY != 0 || !got.EndOfSlice || !got.EndOfFrame {
		t.Fatalf("slice result = %+v, want one-MB frame end", got)
	}
	want := h264High12Intra16x16LumaDCACResidualExpected(t, sh.PPS, int(sh.QScale))
	assertH264RowsHigh(t, "cabac high12 intra16x16 luma-DC/AC y", dst.Y, 0, dst.LumaStride, 16, 16, want.Y, want.LumaStride)
	assertH264RowsHigh(t, "cabac high12 intra16x16 luma-DC/AC cb", dst.Cb, 0, dst.ChromaStride, 8, 8, want.Cb, want.ChromaStride)
	assertH264RowsHigh(t, "cabac high12 intra16x16 luma-DC/AC cr", dst.Cr, 0, dst.ChromaStride, 8, 8, want.Cr, want.ChromaStride)
	if m.MacroblockTyp[0] != MBTypeIntra16x16 || m.CBPTable[0] != 0x10f || m.QScaleTable[0] != 20 || m.SliceTable[0] != 39 {
		t.Fatalf("tables type/cbp/q/slice = %#x/%#x/%d/%d", m.MacroblockTyp[0], m.CBPTable[0], m.QScaleTable[0], m.SliceTable[0])
	}
	if len(src.pcmReadSizes) != 0 {
		t.Fatalf("pcm reads = %v, want none", src.pcmReadSizes)
	}
	wantIndexes(t, src, []int{3, 6, 7, 9, 10, 64, 60, 88, 105, 166, 228, 92, 120, 181, 238, 92, 92, 89, 91, 91, 89, 89, 90, 89, 90, 89, 89, 89, 89, 89})
}

func TestDecodeCABACFrameSliceHigh14ReconstructsIntra16x16Residuals(t *testing.T) {
	const bitDepth = 14

	for _, tt := range []struct {
		name        string
		bits        []int
		signs       []int32
		sliceNum    uint16
		wantCBP     int
		want        func(*testing.T, *PPS, int) *h264PicturePlanesHigh
		wantIndexes []int
	}{
		{
			name:     "luma-dc",
			bits:     []int{1, 0, 0, 1, 0, 0, 0, 1, 1, 1, 0},
			signs:    []int32{1},
			sliceNum: 27,
			wantCBP:  0x100,
			want: func(t *testing.T, pps *PPS, qscale int) *h264PicturePlanesHigh {
				return h264HighIntra16x16LumaDCResidualExpected(t, bitDepth, pps, qscale)
			},
			wantIndexes: []int{3, 6, 7, 9, 10, 64, 60, 88, 105, 166, 228},
		},
		{
			name:     "chroma-dc",
			bits:     []int{1, 0, 1, 0, 1, 0, 0, 0, 0, 1, 1, 1, 0, 0},
			signs:    []int32{1},
			sliceNum: 31,
			wantCBP:  0x50,
			want: func(t *testing.T, pps *PPS, qscale int) *h264PicturePlanesHigh {
				return h264HighIntra16x16ChromaDCResidualExpected(t, bitDepth, pps, qscale)
			},
			wantIndexes: []int{3, 6, 7, 8, 9, 10, 64, 60, 88, 100, 149, 210, 258, 100},
		},
		{
			name: "chroma-ac",
			bits: []int{
				1, 0, 1, 1, 1, 0,
				0,
				0,
				0,
				0, 0,
				1, 1, 1, 0,
				0, 0, 0, 0, 0, 0, 0,
			},
			signs:    []int32{64},
			sliceNum: 43,
			wantCBP:  0x20,
			want: func(t *testing.T, pps *PPS, qscale int) *h264PicturePlanesHigh {
				return h264HighIntra16x16ChromaACResidualExpected(t, bitDepth, pps, qscale)
			},
			wantIndexes: []int{3, 6, 7, 8, 9, 10, 64, 60, 88, 100, 100, 104, 152, 213, 267, 104, 104, 101, 104, 103, 102, 101},
		},
		{
			name: "chroma-dc-ac",
			bits: []int{
				1, 0, 1, 1, 1, 0,
				0,
				0,
				0,
				1, 1, 1, 0,
				0,
				1, 1, 1, 0,
				0, 0, 0, 0, 0, 0, 0,
			},
			signs:    []int32{1, 64},
			sliceNum: 47,
			wantCBP:  0x60,
			want: func(t *testing.T, pps *PPS, qscale int) *h264PicturePlanesHigh {
				return h264HighIntra16x16ChromaDCACResidualExpected(t, bitDepth, pps, qscale)
			},
			wantIndexes: []int{3, 6, 7, 8, 9, 10, 64, 60, 88, 100, 149, 210, 258, 100, 104, 152, 213, 267, 104, 104, 101, 104, 103, 102, 101},
		},
		{
			name: "luma-chroma",
			bits: append(append([]int{
				1, 1, 1, 1, 1, 0,
				0,
				0,
				1, 1, 1, 0,
				1, 1, 1, 0,
			}, repeatCABACBits(15, 0)...), []int{
				1, 1, 1, 0,
				0,
				1, 1, 1, 0,
				0, 0, 0, 0, 0, 0, 0,
			}...),
			signs:    []int32{1, 64, 1, 64},
			sliceNum: 51,
			wantCBP:  0x16f,
			want: func(t *testing.T, pps *PPS, qscale int) *h264PicturePlanesHigh {
				return h264HighIntra16x16LumaChromaResidualExpected(t, bitDepth, pps, qscale)
			},
			wantIndexes: append([]int{3, 6, 7, 8, 9, 10, 64, 60, 88, 105, 166, 228, 92, 120, 181, 238, 92, 92, 89, 91, 91, 89, 89, 90, 89, 90, 89, 89, 89, 89, 89}, []int{100, 149, 210, 258, 100, 104, 152, 213, 267, 104, 104, 101, 104, 103, 102, 101}...),
		},
		{
			name: "luma-ac",
			bits: append([]int{
				1, 1, 0, 1, 0,
				0,
				0,
				0,
				1, 1, 1, 0,
			}, repeatCABACBits(15, 0)...),
			signs:    []int32{64},
			sliceNum: 35,
			wantCBP:  0x0f,
			want: func(t *testing.T, _ *PPS, _ int) *h264PicturePlanesHigh {
				return h264HighIntra16x16LumaACResidualExpected(t, bitDepth)
			},
			wantIndexes: []int{3, 6, 7, 9, 10, 64, 60, 88, 92, 120, 181, 238, 92, 92, 89, 91, 91, 89, 89, 90, 89, 90, 89, 89, 89, 89, 89},
		},
		{
			name: "luma-dc-ac",
			bits: append([]int{
				1, 1, 0, 1, 0,
				0,
				0,
				1, 1, 1, 0,
				1, 1, 1, 0,
			}, repeatCABACBits(15, 0)...),
			signs:    []int32{1, 64},
			sliceNum: 39,
			wantCBP:  0x10f,
			want: func(t *testing.T, pps *PPS, qscale int) *h264PicturePlanesHigh {
				return h264HighIntra16x16LumaDCACResidualExpected(t, bitDepth, pps, qscale)
			},
			wantIndexes: []int{3, 6, 7, 9, 10, 64, 60, 88, 105, 166, 228, 92, 120, 181, 238, 92, 92, 89, 91, 91, 89, 89, 90, 89, 90, 89, 89, 89, 89, 89},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, bitDepth, 1, 1, false, PictureTypeI)
			sh.PPS.CABAC = 1
			src := &scriptedCABACSource{
				bits:  append([]int(nil), tt.bits...),
				signs: append([]int32(nil), tt.signs...),
				terms: []int{0, 1},
			}

			got, err := m.decodeCABACFrameSliceHigh(src, dst, sh, h264FrameSliceDecodeInputHigh{SliceNum: tt.sliceNum})
			if err != nil {
				t.Fatalf("decode high14 cabac intra16x16 %s residual slice failed: %v", tt.name, err)
			}
			if got.Macroblocks != 1 || got.LastMBXY != 0 || !got.EndOfSlice || !got.EndOfFrame {
				t.Fatalf("slice result = %+v, want one-MB frame end", got)
			}
			want := tt.want(t, sh.PPS, int(sh.QScale))
			assertH264RowsHigh(t, "cabac high14 intra16x16 "+tt.name+" y", dst.Y, 0, dst.LumaStride, 16, 16, want.Y, want.LumaStride)
			assertH264RowsHigh(t, "cabac high14 intra16x16 "+tt.name+" cb", dst.Cb, 0, dst.ChromaStride, 8, 8, want.Cb, want.ChromaStride)
			assertH264RowsHigh(t, "cabac high14 intra16x16 "+tt.name+" cr", dst.Cr, 0, dst.ChromaStride, 8, 8, want.Cr, want.ChromaStride)
			if m.MacroblockTyp[0] != MBTypeIntra16x16 || m.CBPTable[0] != tt.wantCBP || m.QScaleTable[0] != 20 || m.SliceTable[0] != tt.sliceNum {
				t.Fatalf("tables type/cbp/q/slice = %#x/%#x/%d/%d", m.MacroblockTyp[0], m.CBPTable[0], m.QScaleTable[0], m.SliceTable[0])
			}
			if len(src.pcmReadSizes) != 0 {
				t.Fatalf("pcm reads = %v, want none", src.pcmReadSizes)
			}
			wantIndexes(t, src, tt.wantIndexes)
		})
	}
}

func TestDecodeCAVLCFrameSliceHighReconstructsPIntra4x4NoResidual(t *testing.T) {
	const bitDepth = 10
	m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, bitDepth, 1, 1, false, PictureTypeP)
	sh.RefCount = [2]uint32{1, 0}
	gb := newBitReader(cavlcBitString("1001101111111111111111100100"))

	got, err := m.decodeCAVLCFrameSliceHigh(&gb, dst, sh, h264FrameSliceDecodeInputHigh{SliceNum: 41})
	if err != nil {
		t.Fatalf("decode high cavlc P intra4x4 slice failed: %v", err)
	}
	if got.Macroblocks != 1 || got.LastMBXY != 0 || !got.EndOfSlice || !got.EndOfFrame {
		t.Fatalf("slice result = %+v, want one P intra MB frame end", got)
	}
	assertH264ConstantBlockHigh(t, "cavlc high P intra4x4 y", dst.Y, 0, dst.LumaStride, 16, 16, 1<<(bitDepth-1))
	assertH264ConstantBlockHigh(t, "cavlc high P intra4x4 cb", dst.Cb, 0, dst.ChromaStride, 8, 8, 1<<(bitDepth-1))
	assertH264ConstantBlockHigh(t, "cavlc high P intra4x4 cr", dst.Cr, 0, dst.ChromaStride, 8, 8, 1<<(bitDepth-1))
	if m.MacroblockTyp[0] != MBTypeIntra4x4 || m.CBPTable[0] != 0 || m.QScaleTable[0] != 20 || m.SliceTable[0] != 41 {
		t.Fatalf("tables type/cbp/q/slice = %#x/%#x/%d/%d", m.MacroblockTyp[0], m.CBPTable[0], m.QScaleTable[0], m.SliceTable[0])
	}
}

func TestDecodeCABACFrameSliceHighReconstructsPIntra4x4NoResidual(t *testing.T) {
	const bitDepth = 10
	m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, bitDepth, 1, 1, false, PictureTypeP)
	sh.PPS.CABAC = 1
	sh.RefCount = [2]uint32{1, 0}
	src := &scriptedCABACSource{
		bits: append(append([]int{0, 1, 0}, repeatCABACBits(16, 1)...), []int{
			0,
			0, 0, 0, 0,
			0,
		}...),
		terms: []int{1},
	}

	got, err := m.decodeCABACFrameSliceHigh(src, dst, sh, h264FrameSliceDecodeInputHigh{SliceNum: 43})
	if err != nil {
		t.Fatalf("decode high cabac P intra4x4 slice failed: %v", err)
	}
	if got.Macroblocks != 1 || got.LastMBXY != 0 || !got.EndOfSlice || !got.EndOfFrame {
		t.Fatalf("slice result = %+v, want one P intra MB frame end", got)
	}
	assertH264ConstantBlockHigh(t, "cabac high P intra4x4 y", dst.Y, 0, dst.LumaStride, 16, 16, 1<<(bitDepth-1))
	assertH264ConstantBlockHigh(t, "cabac high P intra4x4 cb", dst.Cb, 0, dst.ChromaStride, 8, 8, 1<<(bitDepth-1))
	assertH264ConstantBlockHigh(t, "cabac high P intra4x4 cr", dst.Cr, 0, dst.ChromaStride, 8, 8, 1<<(bitDepth-1))
	if m.MacroblockTyp[0] != MBTypeIntra4x4 || m.CBPTable[0] != 0 || m.QScaleTable[0] != 20 || m.SliceTable[0] != 43 {
		t.Fatalf("tables type/cbp/q/slice = %#x/%#x/%d/%d", m.MacroblockTyp[0], m.CBPTable[0], m.QScaleTable[0], m.SliceTable[0])
	}
	if len(src.pcmReadSizes) != 0 {
		t.Fatalf("pcm reads = %v, want none", src.pcmReadSizes)
	}
	wantIndexes(t, src, append(append([]int{11, 14, 17}, repeatCABACBits(16, 68)...), []int{64, 73, 74, 75, 76, 77}...))
}

func TestDecodeCAVLCFrameSliceHighReconstructsPSkip(t *testing.T) {
	for _, bitDepth := range []int32{10, 12, 14} {
		t.Run(bitDepthName(bitDepth), func(t *testing.T) {
			m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, bitDepth, 1, 1, false, PictureTypeP)
			sh.QScale = 24
			sh.RefCount = [2]uint32{1, 0}
			ref := makeH264SliceDecodePictureHigh(1, 1, 1)
			fillH264MotionCompPlaneHigh(ref.Y, 73, int(bitDepth))
			fillH264MotionCompPlaneHigh(ref.Cb, 91, int(bitDepth))
			fillH264MotionCompPlaneHigh(ref.Cr, 119, int(bitDepth))
			gb := newBitReader(cavlcBitString("010"))

			got, err := m.decodeCAVLCFrameSliceHigh(&gb, dst, sh, h264FrameSliceDecodeInputHigh{
				SliceNum:      23,
				Refs:          [2][]*h264PicturePlanesHigh{{ref}},
				MotionScratch: makeH264MotionCompScratchHigh(dst),
			})
			if err != nil {
				t.Fatalf("decode high cavlc pskip slice failed: %v", err)
			}
			if got.Macroblocks != 1 || got.LastMBXY != 0 || !got.EndOfSlice || !got.EndOfFrame {
				t.Fatalf("slice result = %+v, want one skipped MB frame end", got)
			}
			assertH264RowsHigh(t, "high cavlc pskip y", dst.Y, 0, dst.LumaStride, 16, 16, ref.Y, ref.LumaStride)
			assertH264RowsHigh(t, "high cavlc pskip cb", dst.Cb, 0, dst.ChromaStride, 8, 8, ref.Cb, ref.ChromaStride)
			assertH264RowsHigh(t, "high cavlc pskip cr", dst.Cr, 0, dst.ChromaStride, 8, 8, ref.Cr, ref.ChromaStride)
			wantType := MBType16x16 | MBTypeP0L0 | MBTypeP1L0 | MBTypeSkip
			if m.MacroblockTyp[0] != wantType || m.CBPTable[0] != 0 || m.QScaleTable[0] != 24 || m.SliceTable[0] != 23 {
				t.Fatalf("tables type/cbp/q/slice = %#x/%#x/%d/%d", m.MacroblockTyp[0], m.CBPTable[0], m.QScaleTable[0], m.SliceTable[0])
			}
		})
	}
}

func TestDecodeCAVLCFrameSliceHighReconstructsP16x16NoResidual(t *testing.T) {
	for _, bitDepth := range []int32{10, 12, 14} {
		t.Run(bitDepthName(bitDepth), func(t *testing.T) {
			m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, bitDepth, 1, 1, false, PictureTypeP)
			sh.QScale = 24
			sh.RefCount = [2]uint32{1, 0}
			ref := makeH264SliceDecodePictureHigh(1, 1, 1)
			fillH264MotionCompPlaneHigh(ref.Y, 37, int(bitDepth))
			fillH264MotionCompPlaneHigh(ref.Cb, 53, int(bitDepth))
			fillH264MotionCompPlaneHigh(ref.Cr, 71, int(bitDepth))
			gb := newBitReader(cavlcBitString("11111"))

			got, err := m.decodeCAVLCFrameSliceHigh(&gb, dst, sh, h264FrameSliceDecodeInputHigh{
				SliceNum:      29,
				Refs:          [2][]*h264PicturePlanesHigh{{ref}},
				MotionScratch: makeH264MotionCompScratchHigh(dst),
			})
			if err != nil {
				t.Fatalf("decode high cavlc p16x16 slice failed: %v", err)
			}
			if got.Macroblocks != 1 || got.LastMBXY != 0 || !got.EndOfSlice || !got.EndOfFrame {
				t.Fatalf("slice result = %+v, want one P16x16 MB frame end", got)
			}
			assertH264RowsHigh(t, "high cavlc p16 y", dst.Y, 0, dst.LumaStride, 16, 16, ref.Y, ref.LumaStride)
			assertH264RowsHigh(t, "high cavlc p16 cb", dst.Cb, 0, dst.ChromaStride, 8, 8, ref.Cb, ref.ChromaStride)
			assertH264RowsHigh(t, "high cavlc p16 cr", dst.Cr, 0, dst.ChromaStride, 8, 8, ref.Cr, ref.ChromaStride)
			wantType := MBType16x16 | MBTypeP0L0
			if m.MacroblockTyp[0] != wantType || m.CBPTable[0] != 0 || m.QScaleTable[0] != 24 || m.SliceTable[0] != 29 {
				t.Fatalf("tables type/cbp/q/slice = %#x/%#x/%d/%d", m.MacroblockTyp[0], m.CBPTable[0], m.QScaleTable[0], m.SliceTable[0])
			}
		})
	}
}

func TestDecodeCABACFrameSliceHighReconstructsPSkip(t *testing.T) {
	for _, bitDepth := range []int32{10, 12, 14} {
		t.Run(bitDepthName(bitDepth), func(t *testing.T) {
			m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, bitDepth, 1, 1, false, PictureTypeP)
			sh.PPS.CABAC = 1
			sh.QScale = 24
			sh.RefCount = [2]uint32{1, 0}
			ref := makeH264SliceDecodePictureHigh(1, 1, 1)
			fillH264MotionCompPlaneHigh(ref.Y, 83, int(bitDepth))
			fillH264MotionCompPlaneHigh(ref.Cb, 107, int(bitDepth))
			fillH264MotionCompPlaneHigh(ref.Cr, 131, int(bitDepth))
			src := &scriptedCABACSource{
				bits:  []int{1},
				terms: []int{1},
			}

			got, err := m.decodeCABACFrameSliceHigh(src, dst, sh, h264FrameSliceDecodeInputHigh{
				SliceNum:      31,
				Refs:          [2][]*h264PicturePlanesHigh{{ref}},
				MotionScratch: makeH264MotionCompScratchHigh(dst),
			})
			if err != nil {
				t.Fatalf("decode high cabac pskip slice failed: %v", err)
			}
			if got.Macroblocks != 1 || got.LastMBXY != 0 || !got.EndOfSlice || !got.EndOfFrame {
				t.Fatalf("slice result = %+v, want one skipped MB frame end", got)
			}
			assertH264RowsHigh(t, "high cabac pskip y", dst.Y, 0, dst.LumaStride, 16, 16, ref.Y, ref.LumaStride)
			assertH264RowsHigh(t, "high cabac pskip cb", dst.Cb, 0, dst.ChromaStride, 8, 8, ref.Cb, ref.ChromaStride)
			assertH264RowsHigh(t, "high cabac pskip cr", dst.Cr, 0, dst.ChromaStride, 8, 8, ref.Cr, ref.ChromaStride)
			wantType := MBType16x16 | MBTypeP0L0 | MBTypeP1L0 | MBTypeSkip
			if m.MacroblockTyp[0] != wantType || m.CBPTable[0] != 0 || m.QScaleTable[0] != 24 || m.SliceTable[0] != 31 {
				t.Fatalf("tables type/cbp/q/slice = %#x/%#x/%d/%d", m.MacroblockTyp[0], m.CBPTable[0], m.QScaleTable[0], m.SliceTable[0])
			}
			wantIndexes(t, src, []int{11})
		})
	}
}

func TestDecodeCABACFrameSliceHighReconstructsP16x16NoResidual(t *testing.T) {
	for _, bitDepth := range []int32{10, 12, 14} {
		t.Run(bitDepthName(bitDepth), func(t *testing.T) {
			m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, bitDepth, 1, 1, false, PictureTypeP)
			sh.PPS.CABAC = 1
			sh.QScale = 24
			sh.RefCount = [2]uint32{1, 0}
			ref := makeH264SliceDecodePictureHigh(1, 1, 1)
			fillH264MotionCompPlaneHigh(ref.Y, 43, int(bitDepth))
			fillH264MotionCompPlaneHigh(ref.Cb, 61, int(bitDepth))
			fillH264MotionCompPlaneHigh(ref.Cr, 79, int(bitDepth))
			src := &scriptedCABACSource{
				bits: []int{
					0,
					0, 0, 0,
					0, 0,
					0, 0, 0, 0,
					0,
				},
				terms: []int{1},
			}

			got, err := m.decodeCABACFrameSliceHigh(src, dst, sh, h264FrameSliceDecodeInputHigh{
				SliceNum:      37,
				Refs:          [2][]*h264PicturePlanesHigh{{ref}},
				MotionScratch: makeH264MotionCompScratchHigh(dst),
			})
			if err != nil {
				t.Fatalf("decode high cabac p16x16 slice failed: %v", err)
			}
			if got.Macroblocks != 1 || got.LastMBXY != 0 || !got.EndOfSlice || !got.EndOfFrame {
				t.Fatalf("slice result = %+v, want one P16x16 MB frame end", got)
			}
			assertH264RowsHigh(t, "high cabac p16 y", dst.Y, 0, dst.LumaStride, 16, 16, ref.Y, ref.LumaStride)
			assertH264RowsHigh(t, "high cabac p16 cb", dst.Cb, 0, dst.ChromaStride, 8, 8, ref.Cb, ref.ChromaStride)
			assertH264RowsHigh(t, "high cabac p16 cr", dst.Cr, 0, dst.ChromaStride, 8, 8, ref.Cr, ref.ChromaStride)
			wantType := MBType16x16 | MBTypeP0L0
			if m.MacroblockTyp[0] != wantType || m.CBPTable[0] != 0 || m.QScaleTable[0] != 24 || m.SliceTable[0] != 37 {
				t.Fatalf("tables type/cbp/q/slice = %#x/%#x/%d/%d", m.MacroblockTyp[0], m.CBPTable[0], m.QScaleTable[0], m.SliceTable[0])
			}
			wantIndexes(t, src, []int{11, 14, 15, 16, 40, 47, 73, 74, 75, 76, 77})
		})
	}
}

func TestDecodeCAVLCFrameSliceHighRejectsUnsupportedBeforeEntropy(t *testing.T) {
	m, dst, sh := highFrameSliceDecodeFixture(t, 10, 2, true, PictureTypeB)
	sh.DeblockingFilter = 2
	sh.RefCount = [2]uint32{1, 1}
	gb := newBitReader(cavlcIntraPCMBytes(h264ReconstructIntraPCMHigh(1, 10, 5)))

	_, err := m.decodeCAVLCFrameSliceHigh(&gb, dst, sh, h264FrameSliceDecodeInputHigh{SliceNum: 2})
	if err != ErrUnsupported {
		t.Fatalf("decode err = %v, want ErrUnsupported", err)
	}
	if gb.bitPos != 0 {
		t.Fatalf("bit reader consumed %d bits, want 0", gb.bitPos)
	}
	if m.MacroblockTyp[0] != 0 || m.SliceTable[0] != ^uint16(0) {
		t.Fatalf("tables type/slice = %#x/%#x, want untouched", m.MacroblockTyp[0], m.SliceTable[0])
	}
}

func TestDecodeFrameSliceDataHighRejectsUnsupportedChromaBSliceBoundaryBeforeStartup(t *testing.T) {
	m, dst, sh := highFrameSliceDecodeFixture(t, 10, 2, true, PictureTypeB)
	sh.PPS.CABAC = 1
	sh.DeblockingFilter = 2
	sh.RefCount = [2]uint32{1, 1}
	gb := newBitReader([]byte{0xe0})
	if _, err := gb.readBits(3); err != nil {
		t.Fatal(err)
	}

	_, err := m.decodeFrameSliceDataHigh(&gb, dst, sh, h264FrameSliceDecodeInputHigh{SliceNum: 2})
	if err != ErrUnsupported {
		t.Fatalf("decode err = %v, want ErrUnsupported", err)
	}
	if gb.bitPos != 3 {
		t.Fatalf("bit reader consumed %d bits, want 3", gb.bitPos)
	}
	if m.MacroblockTyp[0] != 0 || m.SliceTable[0] != ^uint16(0) {
		t.Fatalf("tables type/slice = %#x/%#x, want untouched", m.MacroblockTyp[0], m.SliceTable[0])
	}
}

func simpleFrameSliceDecodeBitDepthFixture(t *testing.T, bitDepth int32) (*macroblockTables, *h264PicturePlanes, *SliceHeader) {
	t.Helper()

	m, err := newMacroblockTables(1, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	sps := &SPS{
		BitDepthLuma:     bitDepth,
		BitDepthChroma:   bitDepth,
		ChromaFormatIDC:  1,
		FrameMBSOnlyFlag: 1,
	}
	pps := cavlcFlatQMulPPS()
	pps.SPS = sps
	sh := &SliceHeader{
		FirstMBAddr:      0,
		SliceType:        PictureTypeI,
		SliceTypeNoS:     PictureTypeI,
		PictureStructure: PictureFrame,
		PPS:              pps,
		SPS:              sps,
		QScale:           20,
		DeblockingFilter: 0,
	}
	return m, makeH264SliceDecodePicture(1, 1, 1), sh
}

func highFrameSliceDecodeFixture(t *testing.T, bitDepth int32, chromaFormatIDC int, deblock bool, sliceType int32) (*macroblockTables, *h264PicturePlanesHigh, *SliceHeader) {
	t.Helper()

	mbWidth := 1
	if chromaFormatIDC == 1 {
		mbWidth = 2
	}
	return highFrameSliceDecodeFixtureWithMBWidth(t, bitDepth, chromaFormatIDC, mbWidth, deblock, sliceType)
}

func highFrameSliceDecodeFixtureWithMBWidth(t *testing.T, bitDepth int32, chromaFormatIDC int, mbWidth int, deblock bool, sliceType int32) (*macroblockTables, *h264PicturePlanesHigh, *SliceHeader) {
	t.Helper()

	m, err := newMacroblockTables(mbWidth, 1, chromaFormatIDC)
	if err != nil {
		t.Fatal(err)
	}
	sps := &SPS{
		BitDepthLuma:     bitDepth,
		BitDepthChroma:   bitDepth,
		ChromaFormatIDC:  uint32(chromaFormatIDC),
		FrameMBSOnlyFlag: 1,
	}
	pps := cavlcFlatQMulPPS()
	pps.SPS = sps
	sh := &SliceHeader{
		FirstMBAddr:      0,
		SliceType:        sliceType,
		SliceTypeNoS:     sliceType,
		PictureStructure: PictureFrame,
		PPS:              pps,
		SPS:              sps,
		QScale:           20,
		DeblockingFilter: 0,
	}
	if deblock {
		sh.DeblockingFilter = 1
	}
	return m, makeH264SliceDecodePictureHigh(mbWidth, 1, chromaFormatIDC), sh
}

func makeH264SliceDecodePictureHigh(mbWidth int, mbHeight int, chromaFormatIDC int) *h264PicturePlanesHigh {
	chromaWidth, chromaHeight := h264ChromaFrameSize(mbWidth, mbHeight, chromaFormatIDC)
	p := &h264PicturePlanesHigh{
		Y:                make([]uint16, mbWidth*16*mbHeight*16),
		LumaStride:       mbWidth * 16,
		MBWidth:          mbWidth,
		MBHeight:         mbHeight,
		ChromaFormatIDC:  chromaFormatIDC,
		PictureStructure: PictureFrame,
	}
	if chromaFormatIDC != 0 {
		p.ChromaStride = chromaWidth
		p.Cb = make([]uint16, chromaWidth*chromaHeight)
		p.Cr = make([]uint16, chromaWidth*chromaHeight)
	}
	return p
}

func h264High12Intra16x16LumaDCResidualExpected(t *testing.T, pps *PPS, qscale int) *h264PicturePlanesHigh {
	t.Helper()
	return h264HighIntra16x16LumaDCResidualExpected(t, 12, pps, qscale)
}

func h264HighIntra16x16LumaDCResidualExpected(t *testing.T, bitDepth int, pps *PPS, qscale int) *h264PicturePlanesHigh {
	t.Helper()
	p := makeH264SliceDecodePictureHigh(1, 1, 1)
	base := uint16(1 << (bitDepth - 1))
	fillH264HighResidualPlane(p.Y, base)
	fillH264HighResidualPlane(p.Cb, base)
	fillH264HighResidualPlane(p.Cr, base)

	var residual cavlcResidualContext
	residual.MBLumaDC[0][0] = 1
	residual.NonZeroCountCache[h264Scan8[lumaDCBlockIndex]] = 1
	if err := h264LumaDCDequantIDCTHigh(residual.MB[:16*16], &residual.MBLumaDC[0], int(pps.Dequant4Buffer[0][qscale][0])); err != nil {
		t.Fatal(err)
	}
	blockOffset, err := h264FrameBlockOffsets(p.LumaStride, p.ChromaStride, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := h264IDCTAdd16IntraPlaneHigh(p.Y, &blockOffset, residual.MB[:], p.LumaStride, &residual.NonZeroCountCache, 0, bitDepth); err != nil {
		t.Fatal(err)
	}
	return p
}

func h264High12Intra16x16ChromaDCResidualExpected(t *testing.T, pps *PPS, qscale int) *h264PicturePlanesHigh {
	t.Helper()
	return h264HighIntra16x16ChromaDCResidualExpected(t, 12, pps, qscale)
}

func h264HighIntra16x16ChromaDCResidualExpected(t *testing.T, bitDepth int, pps *PPS, qscale int) *h264PicturePlanesHigh {
	t.Helper()
	p := makeH264SliceDecodePictureHigh(1, 1, 1)
	base := uint16(1 << (bitDepth - 1))
	fillH264HighResidualPlane(p.Y, base)
	fillH264HighResidualPlane(p.Cb, base)
	fillH264HighResidualPlane(p.Cr, base)

	var residual cavlcResidualContext
	residual.MB[16*16] = 1
	residual.NonZeroCountCache[h264Scan8[chromaDCBlockIndex]] = 1
	blockOffset, err := h264FrameBlockOffsets(p.LumaStride, p.ChromaStride, 0)
	if err != nil {
		t.Fatal(err)
	}
	chromaQP := [2]uint8{pps.ChromaQPTable[0][qscale], pps.ChromaQPTable[1][qscale]}
	if err := h264HLDecodeMBIDCTChromaHigh(p.Cb, p.Cr, 0, 0, p.ChromaStride, &blockOffset, 1, MBTypeIntra16x16, 0x10, chromaQP, pps, &residual, false, intraPredDC1288x8, 0, bitDepth); err != nil {
		t.Fatal(err)
	}
	return p
}

func h264High12Intra16x16ChromaACResidualExpected(t *testing.T, pps *PPS, qscale int) *h264PicturePlanesHigh {
	t.Helper()
	return h264HighIntra16x16ChromaACResidualExpected(t, 12, pps, qscale)
}

func h264HighIntra16x16ChromaACResidualExpected(t *testing.T, bitDepth int, pps *PPS, qscale int) *h264PicturePlanesHigh {
	t.Helper()
	p := makeH264SliceDecodePictureHigh(1, 1, 1)
	base := uint16(1 << (bitDepth - 1))
	fillH264HighResidualPlane(p.Y, base)
	fillH264HighResidualPlane(p.Cb, base)
	fillH264HighResidualPlane(p.Cr, base)

	var residual cavlcResidualContext
	residual.MB[16*16+int(h264ZigzagScanCAVLC[1])] = 1
	residual.NonZeroCountCache[h264Scan8[16]] = 1
	blockOffset, err := h264FrameBlockOffsets(p.LumaStride, p.ChromaStride, 0)
	if err != nil {
		t.Fatal(err)
	}
	chromaQP := [2]uint8{pps.ChromaQPTable[0][qscale], pps.ChromaQPTable[1][qscale]}
	if err := h264HLDecodeMBIDCTChromaHigh(p.Cb, p.Cr, 0, 0, p.ChromaStride, &blockOffset, 1, MBTypeIntra16x16, 0x20, chromaQP, pps, &residual, false, intraPredDC1288x8, 0, bitDepth); err != nil {
		t.Fatal(err)
	}
	return p
}

func h264High12Intra16x16ChromaDCACResidualExpected(t *testing.T, pps *PPS, qscale int) *h264PicturePlanesHigh {
	t.Helper()
	return h264HighIntra16x16ChromaDCACResidualExpected(t, 12, pps, qscale)
}

func h264HighIntra16x16ChromaDCACResidualExpected(t *testing.T, bitDepth int, pps *PPS, qscale int) *h264PicturePlanesHigh {
	t.Helper()
	p := makeH264SliceDecodePictureHigh(1, 1, 1)
	base := uint16(1 << (bitDepth - 1))
	fillH264HighResidualPlane(p.Y, base)
	fillH264HighResidualPlane(p.Cb, base)
	fillH264HighResidualPlane(p.Cr, base)

	var residual cavlcResidualContext
	residual.MB[16*16] = 1
	residual.MB[16*16+int(h264ZigzagScanCAVLC[1])] = 1
	residual.NonZeroCountCache[h264Scan8[chromaDCBlockIndex]] = 1
	residual.NonZeroCountCache[h264Scan8[16]] = 1
	blockOffset, err := h264FrameBlockOffsets(p.LumaStride, p.ChromaStride, 0)
	if err != nil {
		t.Fatal(err)
	}
	chromaQP := [2]uint8{pps.ChromaQPTable[0][qscale], pps.ChromaQPTable[1][qscale]}
	if err := h264HLDecodeMBIDCTChromaHigh(p.Cb, p.Cr, 0, 0, p.ChromaStride, &blockOffset, 1, MBTypeIntra16x16, 0x20, chromaQP, pps, &residual, false, intraPredDC1288x8, 0, bitDepth); err != nil {
		t.Fatal(err)
	}
	return p
}

func h264High12Intra16x16LumaChromaResidualExpected(t *testing.T, pps *PPS, qscale int) *h264PicturePlanesHigh {
	t.Helper()
	return h264HighIntra16x16LumaChromaResidualExpected(t, 12, pps, qscale)
}

func h264HighIntra16x16LumaChromaResidualExpected(t *testing.T, bitDepth int, pps *PPS, qscale int) *h264PicturePlanesHigh {
	t.Helper()
	p := makeH264SliceDecodePictureHigh(1, 1, 1)
	base := uint16(1 << (bitDepth - 1))
	fillH264HighResidualPlane(p.Y, base)
	fillH264HighResidualPlane(p.Cb, base)
	fillH264HighResidualPlane(p.Cr, base)

	var residual cavlcResidualContext
	residual.MBLumaDC[0][0] = 1
	residual.MB[int(h264ZigzagScanCAVLC[1])] = 1
	residual.MB[16*16] = 1
	residual.MB[16*16+int(h264ZigzagScanCAVLC[1])] = 1
	residual.NonZeroCountCache[h264Scan8[lumaDCBlockIndex]] = 1
	residual.NonZeroCountCache[h264Scan8[0]] = 1
	residual.NonZeroCountCache[h264Scan8[chromaDCBlockIndex]] = 1
	residual.NonZeroCountCache[h264Scan8[16]] = 1
	if err := h264LumaDCDequantIDCTHigh(residual.MB[:16*16], &residual.MBLumaDC[0], int(pps.Dequant4Buffer[0][qscale][0])); err != nil {
		t.Fatal(err)
	}
	blockOffset, err := h264FrameBlockOffsets(p.LumaStride, p.ChromaStride, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := h264IDCTAdd16IntraPlaneHigh(p.Y, &blockOffset, residual.MB[:], p.LumaStride, &residual.NonZeroCountCache, 0, bitDepth); err != nil {
		t.Fatal(err)
	}
	chromaQP := [2]uint8{pps.ChromaQPTable[0][qscale], pps.ChromaQPTable[1][qscale]}
	if err := h264HLDecodeMBIDCTChromaHigh(p.Cb, p.Cr, 0, 0, p.ChromaStride, &blockOffset, 1, MBTypeIntra16x16, 0x20, chromaQP, pps, &residual, false, intraPredDC1288x8, 0, bitDepth); err != nil {
		t.Fatal(err)
	}
	return p
}

func h264High12Intra16x16LumaACResidualExpected(t *testing.T) *h264PicturePlanesHigh {
	t.Helper()
	return h264HighIntra16x16LumaACResidualExpected(t, 12)
}

func h264HighIntra16x16LumaACResidualExpected(t *testing.T, bitDepth int) *h264PicturePlanesHigh {
	t.Helper()
	p := makeH264SliceDecodePictureHigh(1, 1, 1)
	base := uint16(1 << (bitDepth - 1))
	fillH264HighResidualPlane(p.Y, base)
	fillH264HighResidualPlane(p.Cb, base)
	fillH264HighResidualPlane(p.Cr, base)

	var residual cavlcResidualContext
	residual.MB[int(h264ZigzagScanCAVLC[1])] = 1
	residual.NonZeroCountCache[h264Scan8[0]] = 1
	blockOffset, err := h264FrameBlockOffsets(p.LumaStride, p.ChromaStride, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := h264IDCTAdd16IntraPlaneHigh(p.Y, &blockOffset, residual.MB[:], p.LumaStride, &residual.NonZeroCountCache, 0, bitDepth); err != nil {
		t.Fatal(err)
	}
	return p
}

func h264High12Intra16x16LumaDCACResidualExpected(t *testing.T, pps *PPS, qscale int) *h264PicturePlanesHigh {
	t.Helper()
	return h264HighIntra16x16LumaDCACResidualExpected(t, 12, pps, qscale)
}

func h264HighIntra16x16LumaDCACResidualExpected(t *testing.T, bitDepth int, pps *PPS, qscale int) *h264PicturePlanesHigh {
	t.Helper()
	p := makeH264SliceDecodePictureHigh(1, 1, 1)
	base := uint16(1 << (bitDepth - 1))
	fillH264HighResidualPlane(p.Y, base)
	fillH264HighResidualPlane(p.Cb, base)
	fillH264HighResidualPlane(p.Cr, base)

	var residual cavlcResidualContext
	residual.MBLumaDC[0][0] = 1
	residual.MB[int(h264ZigzagScanCAVLC[1])] = 1
	residual.NonZeroCountCache[h264Scan8[lumaDCBlockIndex]] = 1
	residual.NonZeroCountCache[h264Scan8[0]] = 1
	if err := h264LumaDCDequantIDCTHigh(residual.MB[:16*16], &residual.MBLumaDC[0], int(pps.Dequant4Buffer[0][qscale][0])); err != nil {
		t.Fatal(err)
	}
	blockOffset, err := h264FrameBlockOffsets(p.LumaStride, p.ChromaStride, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := h264IDCTAdd16IntraPlaneHigh(p.Y, &blockOffset, residual.MB[:], p.LumaStride, &residual.NonZeroCountCache, 0, bitDepth); err != nil {
		t.Fatal(err)
	}
	return p
}

func pictureTypeName(sliceType int32) string {
	switch sliceType {
	case PictureTypeI:
		return "I"
	case PictureTypeP:
		return "P"
	case PictureTypeB:
		return "B"
	default:
		return "unknown"
	}
}

func chromaFormatName(chromaFormatIDC int) string {
	switch chromaFormatIDC {
	case 0:
		return "mono"
	case 1:
		return "420"
	case 2:
		return "422"
	case 3:
		return "444"
	default:
		return "chroma"
	}
}

func assertH264SliceDecodePCMHigh(t *testing.T, dst *h264PicturePlanesHigh, mbX int, mbY int, samples []uint16) {
	t.Helper()
	yOff, cbOff, crOff, err := h264MBDestPartOffsetsHigh(dst, mbX, mbY, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	assertH264RowsHigh(t, "slice high pcm y", dst.Y, yOff, dst.LumaStride, 16, 16, samples, 16)
	if dst.ChromaFormatIDC == 0 {
		return
	}
	blockW, blockH := 8, 8
	if dst.ChromaFormatIDC == 2 {
		blockH = 16
	} else if dst.ChromaFormatIDC == 3 {
		blockW = 16
		blockH = 16
	}
	chromaSamples := blockW * blockH
	assertH264RowsHigh(t, "slice high pcm cb", dst.Cb, cbOff, dst.ChromaStride, blockW, blockH, samples[256:], blockW)
	assertH264RowsHigh(t, "slice high pcm cr", dst.Cr, crOff, dst.ChromaStride, blockW, blockH, samples[256+chromaSamples:], blockW)
}

func bitDepthName(bitDepth int32) string {
	switch bitDepth {
	case 9:
		return "9-bit"
	case 10:
		return "10-bit"
	case 12:
		return "12-bit"
	case 14:
		return "14-bit"
	default:
		return "bit-depth"
	}
}
