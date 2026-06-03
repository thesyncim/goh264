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

func TestValidateSimpleFrameSliceDecodeHighAllowsHigh10AndHigh12P420NoWeight(t *testing.T) {
	for _, bitDepth := range []int32{10, 12} {
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
		t.Fatalf("field validation err = %v, want ErrUnsupported", err)
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
		{name: "9-bit", bitDepth: 9, chroma: 9, format: 1, slice: PictureTypeI},
		{name: "12-bit-b-slice-boundary-deblock", bitDepth: 12, chroma: 12, format: 1, deblockMode: 2, slice: PictureTypeB},
		{name: "14-bit-p", bitDepth: 14, chroma: 14, format: 1, slice: PictureTypeP},
		{name: "14-bit-deblock", bitDepth: 14, chroma: 14, format: 1, deblock: true, slice: PictureTypeI},
		{name: "unequal-depth", bitDepth: 10, chroma: 12, format: 1, slice: PictureTypeI},
		{name: "monochrome", bitDepth: 10, chroma: 10, format: 0, slice: PictureTypeI},
		{name: "high10-422-deblock-disabled", bitDepth: 10, chroma: 10, format: 2, slice: PictureTypeI},
		{name: "high10-444-deblock-disabled", bitDepth: 10, chroma: 10, format: 3, slice: PictureTypeI},
		{name: "high12-422-deblock-disabled", bitDepth: 12, chroma: 12, format: 2, slice: PictureTypeI},
		{name: "high12-444-deblock-disabled", bitDepth: 12, chroma: 12, format: 3, slice: PictureTypeI},
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

func TestValidateSimpleFrameSliceDecodeHighRejectsHigh14CABACUntilProved(t *testing.T) {
	m, dst, sh := highFrameSliceDecodeFixture(t, 14, 1, false, PictureTypeI)
	sh.PPS.CABAC = 1

	if err := validateSimpleFrameSliceDecodeInputsHigh(m, dst, sh, 4); err != ErrUnsupported {
		t.Fatalf("high14 CABAC validation err = %v, want ErrUnsupported", err)
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

func TestValidateSimpleFrameSliceDecodeHighAllowsHigh10AndHigh12ChromaDeblocking(t *testing.T) {
	for _, bitDepth := range []int32{10, 12} {
		for _, chromaFormatIDC := range []int{2, 3} {
			for _, sliceType := range []int32{PictureTypeI, PictureTypeP} {
				t.Run(bitDepthName(bitDepth)+"/"+chromaFormatName(chromaFormatIDC)+"/"+pictureTypeName(sliceType), func(t *testing.T) {
					m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, bitDepth, chromaFormatIDC, 2, true, sliceType)
					if sliceType == PictureTypeP {
						sh.RefCount = [2]uint32{1, 0}
					}

					if err := validateSimpleFrameSliceDecodeInputsHigh(m, dst, sh, 4); err != nil {
						t.Fatalf("high chroma deblock validation err = %v, want nil", err)
					}
				})
			}
		}
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
		{
			name:     "12-bit/b-slice-boundary-mode",
			bitDepth: 12,
			run: func(sh *SliceHeader) {
				sh.SliceType = PictureTypeB
				sh.SliceTypeNoS = PictureTypeB
				sh.RefCount = [2]uint32{1, 1}
				sh.DeblockingFilter = 2
			},
		},
		{
			name:     "10-bit/chroma-deblock-disabled",
			bitDepth: 10,
			run: func(sh *SliceHeader) {
				sh.SPS.ChromaFormatIDC = 2
				sh.DeblockingFilter = 0
			},
		},
		{
			name:     "12-bit/chroma-deblock-disabled",
			bitDepth: 12,
			run: func(sh *SliceHeader) {
				sh.SPS.ChromaFormatIDC = 2
				sh.DeblockingFilter = 0
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

func TestValidateSimpleFrameSliceDecodeHighRejectsUnsupportedWeightedB(t *testing.T) {
	for _, tt := range []struct {
		name string
		run  func(*SliceHeader)
	}{
		{
			name: "explicit table",
			run: func(sh *SliceHeader) {
				sh.PPS.WeightedBipredIDC = 1
				sh.PredWeightTable.UseWeight = 1
			},
		},
		{
			name: "mismatched implicit flags",
			run: func(sh *SliceHeader) {
				sh.PPS.WeightedBipredIDC = 2
				sh.PredWeightTable.UseWeight = 2
				sh.PredWeightTable.UseWeightChroma = 0
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, 10, 1, 1, false, PictureTypeB)
			sh.RefCount = [2]uint32{1, 1}
			tt.run(sh)

			if err := validateSimpleFrameSliceDecodeInputsHigh(m, dst, sh, 4); err != ErrUnsupported {
				t.Fatalf("weighted high B validation err = %v, want ErrUnsupported", err)
			}
		})
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
		{name: "9-bit", bitDepth: 9, chroma: 9, format: 1, slice: PictureTypeP},
		{name: "unequal-depth", bitDepth: 10, chroma: 12, format: 1, slice: PictureTypeP},
		{name: "high10-422-deblock-disabled", bitDepth: 10, chroma: 10, format: 2, slice: PictureTypeP},
		{name: "high10-444-deblock-disabled", bitDepth: 10, chroma: 10, format: 3, slice: PictureTypeP},
		{name: "high10-422-weighted-chroma-deblock", bitDepth: 10, chroma: 10, format: 2, deblock: true, slice: PictureTypeP},
		{name: "high10-444-weighted-chroma-deblock", bitDepth: 10, chroma: 10, format: 3, deblock: true, slice: PictureTypeP},
		{name: "high10-422-weighted-chroma-slice-boundary-deblock", bitDepth: 10, chroma: 10, format: 2, deblockMode: 2, slice: PictureTypeP},
		{name: "high10-444-weighted-chroma-slice-boundary-deblock", bitDepth: 10, chroma: 10, format: 3, deblockMode: 2, slice: PictureTypeP},
		{name: "high12-422-weighted-chroma-deblock", bitDepth: 12, chroma: 12, format: 2, deblock: true, slice: PictureTypeP},
		{name: "high12-444-weighted-chroma-deblock", bitDepth: 12, chroma: 12, format: 3, deblock: true, slice: PictureTypeP},
		{name: "high12-422-weighted-chroma-slice-boundary-deblock", bitDepth: 12, chroma: 12, format: 2, deblockMode: 2, slice: PictureTypeP},
		{name: "high12-444-weighted-chroma-slice-boundary-deblock", bitDepth: 12, chroma: 12, format: 3, deblockMode: 2, slice: PictureTypeP},
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

func TestValidateHighFrameSliceReconstructAllowsHigh12IntraResidualScope(t *testing.T) {
	_, _, sh := highFrameSliceDecodeFixture(t, 12, 1, false, PictureTypeI)

	if err := validateHighFrameSliceMacroblockForReconstructWithSubMB(sh, MBTypeIntraPCM, nil, 0, 0); err != nil {
		t.Fatalf("high12 IntraPCM reconstruct validation err = %v, want nil", err)
	}
	if err := validateHighFrameSliceMacroblockForReconstructWithSubMB(sh, MBTypeIntra4x4, nil, 0, 0); err != nil {
		t.Fatalf("high12 Intra4x4 no-residual reconstruct validation err = %v, want nil", err)
	}
	if err := validateHighFrameSliceMacroblockForReconstructWithSubMB(sh, MBTypeIntra4x4, nil, 1, 1); err != ErrUnsupported {
		t.Fatalf("high12 Intra4x4 residual reconstruct validation err = %v, want ErrUnsupported", err)
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

func TestValidateHighFrameSliceReconstructAllowsHigh14CAVLCIntraNoResidual(t *testing.T) {
	_, _, sh := highFrameSliceDecodeFixture(t, 14, 1, false, PictureTypeI)

	if err := validateHighFrameSliceMacroblockForReconstructWithSubMB(sh, MBTypeIntraPCM, nil, 0, 0); err != nil {
		t.Fatalf("high14 IntraPCM reconstruct validation err = %v, want nil", err)
	}
	if err := validateHighFrameSliceMacroblockForReconstructWithSubMB(sh, MBTypeIntra4x4, nil, 0, 0); err != nil {
		t.Fatalf("high14 Intra4x4 no-residual reconstruct validation err = %v, want nil", err)
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
		{name: "intra16x16-cabac-luma-dc", mbType: MBTypeIntra16x16, cbp: 0, cbpTable: 0x101},
		{name: "intra16x16-cabac-luma-ac", mbType: MBTypeIntra16x16, cbp: 0x0f, cbpTable: 0x0f},
		{name: "intra16x16-cabac-chroma-dc", mbType: MBTypeIntra16x16, cbp: 0x10, cbpTable: 0x50},
		{name: "intra16x16-cabac-chroma-ac", mbType: MBTypeIntra16x16, cbp: 0x20, cbpTable: 0x60},
		{name: "intra16x16-cabac-luma-chroma", mbType: MBTypeIntra16x16, cbp: 0x2f, cbpTable: 0x16f},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateHighFrameSliceMacroblockForReconstructWithSubMB(sh, tt.mbType, nil, tt.cbp, tt.cbpTable); err != ErrUnsupported {
				t.Fatalf("high14 %s reconstruct validation err = %v, want ErrUnsupported", tt.name, err)
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
	const bitDepth = 10
	m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, bitDepth, 1, 1, false, PictureTypeI)
	sh.PPS.CABAC = 1
	pcm := h264ReconstructIntraPCMHigh(1, bitDepth, 57)
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
	assertH264SliceDecodePCMHigh(t, dst, 0, 0, h264ReconstructIntraPCMSamples(1, bitDepth, 57))
	if m.MacroblockTyp[0] != MBTypeIntraPCM || m.CBPTable[0] != 0xf7ef || m.QScaleTable[0] != 0 || m.SliceTable[0] != 13 {
		t.Fatalf("tables type/cbp/q/slice = %#x/%#x/%d/%d", m.MacroblockTyp[0], m.CBPTable[0], m.QScaleTable[0], m.SliceTable[0])
	}
	if len(src.pcmReadSizes) != 1 || src.pcmReadSizes[0] != len(pcm) {
		t.Fatalf("pcm read sizes = %v, want [%d]", src.pcmReadSizes, len(pcm))
	}
	wantIndexes(t, src, []int{3})
}

func TestDecodeCAVLCFrameSliceHighReconstructsIntra4x4NoResidual(t *testing.T) {
	for _, bitDepth := range []int32{10, 12, 14} {
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
	for _, bitDepth := range []int32{10, 12} {
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

func TestDecodeCABACFrameSliceHigh12ReconstructsIntra16x16NoResidual(t *testing.T) {
	const bitDepth = 12
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
	assertH264ConstantBlockHigh(t, "cabac high12 intra16x16 y", dst.Y, 0, dst.LumaStride, 16, 16, 1<<(bitDepth-1))
	assertH264ConstantBlockHigh(t, "cabac high12 intra16x16 cb", dst.Cb, 0, dst.ChromaStride, 8, 8, 1<<(bitDepth-1))
	assertH264ConstantBlockHigh(t, "cabac high12 intra16x16 cr", dst.Cr, 0, dst.ChromaStride, 8, 8, 1<<(bitDepth-1))
	if m.MacroblockTyp[0] != MBTypeIntra16x16 || m.CBPTable[0] != 0 || m.QScaleTable[0] != 20 || m.SliceTable[0] != 23 {
		t.Fatalf("tables type/cbp/q/slice = %#x/%#x/%d/%d", m.MacroblockTyp[0], m.CBPTable[0], m.QScaleTable[0], m.SliceTable[0])
	}
	if len(src.pcmReadSizes) != 0 {
		t.Fatalf("pcm reads = %v, want none", src.pcmReadSizes)
	}
	wantIndexes(t, src, []int{3, 6, 7, 9, 10, 64, 60, 88})
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
	for _, bitDepth := range []int32{10, 12} {
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
	for _, bitDepth := range []int32{10, 12} {
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
	for _, bitDepth := range []int32{10, 12} {
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
	for _, bitDepth := range []int32{10, 12} {
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
	m, dst, sh := highFrameSliceDecodeFixture(t, 10, 2, false, PictureTypeI)
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
		Y:               make([]uint16, mbWidth*16*mbHeight*16),
		LumaStride:      mbWidth * 16,
		MBWidth:         mbWidth,
		MBHeight:        mbHeight,
		ChromaFormatIDC: chromaFormatIDC,
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
	if err := h264HLDecodeMBIDCTChromaHigh(p.Cb, p.Cr, p.ChromaStride, &blockOffset, 1, MBTypeIntra16x16, 0x10, chromaQP, pps, &residual, false, intraPredDC1288x8, 0, bitDepth); err != nil {
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
	if err := h264HLDecodeMBIDCTChromaHigh(p.Cb, p.Cr, p.ChromaStride, &blockOffset, 1, MBTypeIntra16x16, 0x20, chromaQP, pps, &residual, false, intraPredDC1288x8, 0, bitDepth); err != nil {
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
	if err := h264HLDecodeMBIDCTChromaHigh(p.Cb, p.Cr, p.ChromaStride, &blockOffset, 1, MBTypeIntra16x16, 0x20, chromaQP, pps, &residual, false, intraPredDC1288x8, 0, bitDepth); err != nil {
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
	if err := h264HLDecodeMBIDCTChromaHigh(p.Cb, p.Cr, p.ChromaStride, &blockOffset, 1, MBTypeIntra16x16, 0x20, chromaQP, pps, &residual, false, intraPredDC1288x8, 0, bitDepth); err != nil {
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
