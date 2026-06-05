// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

import (
	"fmt"
	"testing"
)

func TestValidateHighFrameSliceMacroblockForReconstructAllowsPartitionedP(t *testing.T) {
	pSlice := &SliceHeader{SliceTypeNoS: PictureTypeP}
	cabacP := &SliceHeader{SliceTypeNoS: PictureTypeP, PPS: &PPS{CABAC: 1}}
	weightedP := &SliceHeader{
		SliceTypeNoS: PictureTypeP,
		PPS:          &PPS{WeightedPred: 1},
		PredWeightTable: PredWeightTable{
			UseWeight:       1,
			UseWeightChroma: 1,
		},
	}
	p8x8Sub := [4]uint32{
		MBType16x16 | MBTypeP0L0,
		MBType16x8 | MBTypeP0L0,
		MBType8x16 | MBTypeP0L0,
		MBType8x8 | MBTypeP0L0,
	}
	p8x8DCTSub := [4]uint32{
		MBType16x16 | MBTypeP0L0,
		MBType16x16 | MBTypeP0L0,
		MBType16x16 | MBTypeP0L0,
		MBType16x16 | MBTypeP0L0,
	}

	for _, tt := range []struct {
		name     string
		sh       *SliceHeader
		mbType   uint32
		sub      *[4]uint32
		cbp      int
		cbpTable int
	}{
		{name: "p16x16 8x8 dct residual", sh: pSlice, mbType: MBType16x16 | MBTypeP0L0 | MBType8x8DCT, cbp: 1, cbpTable: 1},
		{name: "p16x8 no residual", sh: pSlice, mbType: MBType16x8 | MBTypeP0L0 | MBTypeP1L0},
		{name: "p16x8 residual", sh: pSlice, mbType: MBType16x8 | MBTypeP0L0 | MBTypeP1L0, cbp: 1, cbpTable: 1},
		{name: "p16x8 8x8 dct residual", sh: pSlice, mbType: MBType16x8 | MBTypeP0L0 | MBTypeP1L0 | MBType8x8DCT, cbp: 1, cbpTable: 1},
		{name: "p8x16 residual", sh: pSlice, mbType: MBType8x16 | MBTypeP0L0 | MBTypeP1L0, cbp: 2, cbpTable: 2},
		{name: "p8x8 sub partitions", sh: pSlice, mbType: MBType8x8 | MBTypeP0L0 | MBTypeP1L0, sub: &p8x8Sub},
		{name: "p8x8 ref0 residual", sh: pSlice, mbType: MBType8x8 | MBTypeP0L0 | MBTypeP1L0 | MBTypeRef0, sub: &p8x8Sub, cbp: 4, cbpTable: 4},
		{name: "p8x8 8x8 dct residual", sh: pSlice, mbType: MBType8x8 | MBTypeP0L0 | MBTypeP1L0 | MBType8x8DCT, sub: &p8x8DCTSub, cbp: 0x1f, cbpTable: 0x9f},
		{name: "cabac p16x8 residual", sh: cabacP, mbType: MBType16x8 | MBTypeP0L0 | MBTypeP1L0, cbp: 1, cbpTable: 1},
		{name: "cabac p8x8 residual", sh: cabacP, mbType: MBType8x8 | MBTypeP0L0 | MBTypeP1L0, sub: &p8x8Sub, cbp: 4, cbpTable: 4},
		{name: "cabac p8x8 8x8 dct residual", sh: cabacP, mbType: MBType8x8 | MBTypeP0L0 | MBTypeP1L0 | MBType8x8DCT, sub: &p8x8DCTSub, cbp: 0x1f, cbpTable: 0x9f},
		{name: "weighted p16x8 residual", sh: weightedP, mbType: MBType16x8 | MBTypeP0L0 | MBTypeP1L0, cbp: 1, cbpTable: 1},
		{name: "weighted p8x8 residual", sh: weightedP, mbType: MBType8x8 | MBTypeP0L0 | MBTypeP1L0, sub: &p8x8Sub, cbp: 4, cbpTable: 4},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateHighFrameSliceMacroblockForReconstructWithSubMB(tt.sh, tt.mbType, tt.sub, tt.cbp, tt.cbpTable); err != nil {
				t.Fatalf("validate high P partitioned err = %v, want nil", err)
			}
		})
	}
}

func TestValidateHighFrameSliceMacroblockForReconstructAllowsFieldPShapes(t *testing.T) {
	fieldP := &SliceHeader{
		SliceTypeNoS:     PictureTypeP,
		PictureStructure: PictureBottomField,
		SPS: &SPS{
			BitDepthLuma:     10,
			BitDepthChroma:   10,
			ChromaFormatIDC:  2,
			FrameMBSOnlyFlag: 0,
			MBAFF:            1,
		},
		PPS: &PPS{CABAC: 1},
	}
	for _, tt := range []struct {
		name     string
		mbType   uint32
		cbp      int
		cbpTable int
	}{
		{name: "pskip", mbType: MBType16x16 | MBTypeP0L0 | MBTypeP1L0 | MBTypeSkip | MBTypeInterlaced},
		{name: "p16x16 residual", mbType: MBType16x16 | MBTypeP0L0 | MBTypeInterlaced, cbp: 1, cbpTable: 1},
		{name: "p16x8 dct residual", mbType: MBType16x8 | MBTypeP0L0 | MBTypeP1L0 | MBTypeInterlaced | MBType8x8DCT, cbp: 0x2f, cbpTable: 0xef},
		{name: "p8x16 residual", mbType: MBType8x16 | MBTypeP0L0 | MBTypeP1L0 | MBTypeInterlaced, cbp: 2, cbpTable: 2},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateHighFrameSliceMacroblockForReconstructWithSubMB(fieldP, tt.mbType, nil, tt.cbp, tt.cbpTable); err != nil {
				t.Fatalf("validate high field P shape err = %v, want nil", err)
			}
		})
	}
}

func TestValidateHighFrameSliceMacroblockForReconstructAllowsHighChromaFieldWeightedP(t *testing.T) {
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
	sub := [4]uint32{
		MBType16x16 | MBTypeP0L0,
		MBType16x8 | MBTypeP0L0,
		MBType8x16 | MBTypeP0L0,
		MBType8x8 | MBTypeP0L0,
	}
	shapes := []struct {
		name     string
		mbType   uint32
		sub      *[4]uint32
		cbp      int
		cbpTable int
	}{
		{name: "pskip", mbType: MBType16x16 | MBTypeP0L0 | MBTypeP1L0 | MBTypeSkip | MBTypeInterlaced},
		{name: "p16x16 residual", mbType: MBType16x16 | MBTypeP0L0 | MBTypeInterlaced, cbp: 1, cbpTable: 1},
		{name: "p16x8 residual", mbType: MBType16x8 | MBTypeP0L0 | MBTypeP1L0 | MBTypeInterlaced, cbp: 1, cbpTable: 1},
		{name: "p8x16 residual", mbType: MBType8x16 | MBTypeP0L0 | MBTypeP1L0 | MBTypeInterlaced, cbp: 2, cbpTable: 2},
		{name: "p8x8 explicit sub", mbType: MBType8x8 | MBTypeP0L0 | MBTypeP1L0 | MBTypeInterlaced, sub: &sub, cbp: 4, cbpTable: 4},
	}
	for _, bitDepth := range []int32{10, 12, 14} {
		for _, chromaFormatIDC := range []uint32{2, 3} {
			for _, picture := range []int32{PictureTopField, PictureBottomField} {
				for _, deblock := range []int32{0, 1} {
					for _, weight := range weights {
						for _, shape := range shapes {
							t.Run(fmt.Sprintf("%s/chroma%d/picture%d/mode%d/%s/%s", bitDepthName(bitDepth), chromaFormatIDC, picture, deblock, weight.name, shape.name), func(t *testing.T) {
								sh := &SliceHeader{
									SliceTypeNoS:     PictureTypeP,
									PictureStructure: picture,
									DeblockingFilter: deblock,
									SPS: &SPS{
										BitDepthLuma:     bitDepth,
										BitDepthChroma:   bitDepth,
										ChromaFormatIDC:  chromaFormatIDC,
										FrameMBSOnlyFlag: 0,
										MBAFF:            1,
									},
									PPS:             &PPS{WeightedPred: 1},
									PredWeightTable: weight.table(int(chromaFormatIDC)),
								}
								if err := validateHighFrameSliceMacroblockForReconstructWithSubMB(sh, shape.mbType, shape.sub, shape.cbp, shape.cbpTable); err != nil {
									t.Fatalf("validate %s chroma field weighted P reconstruct err = %v, want nil", bitDepthName(bitDepth), err)
								}
							})
						}
					}
				}
			}
		}
	}
}

func TestValidateHighFrameSliceMacroblockForReconstructRejectsUnsupportedPartitionedP(t *testing.T) {
	pSlice := &SliceHeader{SliceTypeNoS: PictureTypeP}
	p8x8Sub := [4]uint32{
		MBType16x16 | MBTypeP0L0,
		MBType16x16 | MBTypeP0L0,
		MBType16x16 | MBTypeP0L0,
		MBType16x16 | MBTypeP0L0,
	}
	invalidSub := p8x8Sub
	invalidSub[1] = MBType16x16 | MBTypeP0L1
	smallSub := p8x8Sub
	smallSub[1] = MBType16x8 | MBTypeP0L0

	for _, tt := range []struct {
		name     string
		sh       *SliceHeader
		mbType   uint32
		sub      *[4]uint32
		cbp      int
		cbpTable int
	}{
		{name: "p8x8 without sub types", sh: pSlice, mbType: MBType8x8 | MBTypeP0L0 | MBTypeP1L0},
		{name: "p8x8 invalid sub type", sh: pSlice, mbType: MBType8x8 | MBTypeP0L0 | MBTypeP1L0, sub: &invalidSub},
		{name: "p16x8 8x8 dct without luma cbp", sh: pSlice, mbType: MBType16x8 | MBTypeP0L0 | MBTypeP1L0 | MBType8x8DCT},
		{name: "p8x8 8x8 dct without luma cbp", sh: pSlice, mbType: MBType8x8 | MBTypeP0L0 | MBTypeP1L0 | MBType8x8DCT, sub: &p8x8Sub},
		{name: "p8x8 8x8 dct sub partition too small", sh: pSlice, mbType: MBType8x8 | MBTypeP0L0 | MBTypeP1L0 | MBType8x8DCT, sub: &smallSub, cbp: 1, cbpTable: 1},
		{name: "p16x16 8x8 dct without luma cbp", sh: pSlice, mbType: MBType16x16 | MBTypeP0L0 | MBType8x8DCT},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateHighFrameSliceMacroblockForReconstructWithSubMB(tt.sh, tt.mbType, tt.sub, tt.cbp, tt.cbpTable); err != ErrUnsupported {
				t.Fatalf("validate high P unsupported partition err = %v, want ErrUnsupported", err)
			}
		})
	}
}

func TestDecodeFrameSliceHighReconstructsPartitionedPNoResidual(t *testing.T) {
	for _, bitDepth := range []int32{10, 12} {
		for _, tt := range []struct {
			name      string
			cavlcBits string
			cabacBits []int
			wantType  uint32
		}{
			{
				name:      "p16x8",
				cavlcBits: "101011111",
				cabacBits: []int{0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				wantType:  MBType16x8 | MBTypeP0L0 | MBTypeP1L0,
			},
			{
				name:      "p8x16",
				cavlcBits: "101111111",
				cabacBits: []int{0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				wantType:  MBType8x16 | MBTypeP0L0 | MBTypeP1L0,
			},
			{
				name:      "p8x8",
				cavlcBits: "1001001111111111111",
				cabacBits: []int{0, 0, 0, 1, 1, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				wantType:  MBType8x8 | MBTypeP0L0 | MBTypeP1L0,
			},
		} {
			t.Run(bitDepthName(bitDepth)+"/cavlc-"+tt.name, func(t *testing.T) {
				m, dst, sh, ref := highPartitionedPFrameSliceDecodeFixture(t, bitDepth, false)
				gb := newBitReader(cavlcBitString(tt.cavlcBits))

				got, err := m.decodeCAVLCFrameSliceHigh(&gb, dst, sh, h264FrameSliceDecodeInputHigh{
					SliceNum:      71,
					Refs:          [2][]*h264PicturePlanesHigh{{ref}},
					MotionScratch: makeH264MotionCompScratchHigh(dst),
				})
				assertHighPartitionedPSliceResult(t, got, err, m, dst, ref, tt.wantType, 71)
			})
			t.Run(bitDepthName(bitDepth)+"/cabac-"+tt.name, func(t *testing.T) {
				m, dst, sh, ref := highPartitionedPFrameSliceDecodeFixture(t, bitDepth, true)
				src := &scriptedCABACSource{bits: tt.cabacBits, terms: []int{1}}

				got, err := m.decodeCABACFrameSliceHigh(src, dst, sh, h264FrameSliceDecodeInputHigh{
					SliceNum:      73,
					Refs:          [2][]*h264PicturePlanesHigh{{ref}},
					MotionScratch: makeH264MotionCompScratchHigh(dst),
				})
				assertHighPartitionedPSliceResult(t, got, err, m, dst, ref, tt.wantType, 73)
			})
		}
	}
}

func TestDecodeFrameSliceHigh12ReconstructsWeightedPartitionedPNoResidual(t *testing.T) {
	const bitDepth = 12

	for _, tt := range []struct {
		name      string
		cavlcBits string
		cabacBits []int
		wantType  uint32
	}{
		{
			name:      "p16x8",
			cavlcBits: "101011111",
			cabacBits: []int{0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			wantType:  MBType16x8 | MBTypeP0L0 | MBTypeP1L0,
		},
		{
			name:      "p8x16",
			cavlcBits: "101111111",
			cabacBits: []int{0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			wantType:  MBType8x16 | MBTypeP0L0 | MBTypeP1L0,
		},
		{
			name:      "p8x8",
			cavlcBits: "1001001111111111111",
			cabacBits: []int{0, 0, 0, 1, 1, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			wantType:  MBType8x8 | MBTypeP0L0 | MBTypeP1L0,
		},
	} {
		t.Run("cavlc-"+tt.name, func(t *testing.T) {
			m, dst, sh, ref := highPartitionedPFrameSliceDecodeFixture(t, bitDepth, false)
			pwt := highWeightedPPredWeightTable()
			sh.PPS.WeightedPred = 1
			sh.PredWeightTable = pwt
			gb := newBitReader(cavlcBitString(tt.cavlcBits))

			got, err := m.decodeCAVLCFrameSliceHigh(&gb, dst, sh, h264FrameSliceDecodeInputHigh{
				SliceNum:      75,
				Refs:          [2][]*h264PicturePlanesHigh{{ref}},
				PredWeight:    &sh.PredWeightTable,
				MotionScratch: makeH264MotionCompScratchHigh(dst),
			})
			want := h264HighWeightedPReference(t, ref, &pwt, bitDepth)
			assertHighPartitionedPSliceResult(t, got, err, m, dst, want, tt.wantType, 75)
		})
		t.Run("cabac-"+tt.name, func(t *testing.T) {
			m, dst, sh, ref := highPartitionedPFrameSliceDecodeFixture(t, bitDepth, true)
			pwt := highWeightedPPredWeightTable()
			sh.PPS.WeightedPred = 1
			sh.PredWeightTable = pwt
			src := &scriptedCABACSource{bits: tt.cabacBits, terms: []int{1}}

			got, err := m.decodeCABACFrameSliceHigh(src, dst, sh, h264FrameSliceDecodeInputHigh{
				SliceNum:      77,
				Refs:          [2][]*h264PicturePlanesHigh{{ref}},
				PredWeight:    &sh.PredWeightTable,
				MotionScratch: makeH264MotionCompScratchHigh(dst),
			})
			want := h264HighWeightedPReference(t, ref, &pwt, bitDepth)
			assertHighPartitionedPSliceResult(t, got, err, m, dst, want, tt.wantType, 77)
		})
	}
}

func highPartitionedPFrameSliceDecodeFixture(t *testing.T, bitDepth int32, cabac bool) (*macroblockTables, *h264PicturePlanesHigh, *SliceHeader, *h264PicturePlanesHigh) {
	t.Helper()
	m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, bitDepth, 1, 1, false, PictureTypeP)
	sh.QScale = 24
	sh.RefCount = [2]uint32{1, 0}
	if cabac {
		sh.PPS.CABAC = 1
	}
	ref := makeH264SliceDecodePictureHigh(1, 1, 1)
	fillH264MotionCompPlaneHigh(ref.Y, 137, int(bitDepth))
	fillH264MotionCompPlaneHigh(ref.Cb, 211, int(bitDepth))
	fillH264MotionCompPlaneHigh(ref.Cr, 293, int(bitDepth))
	return m, dst, sh, ref
}

func assertHighPartitionedPSliceResult(t *testing.T, got h264FrameSliceDecodeResult, err error, m *macroblockTables, dst *h264PicturePlanesHigh, want *h264PicturePlanesHigh, wantType uint32, wantSlice uint16) {
	t.Helper()
	if err != nil {
		t.Fatalf("decode high partitioned P failed: %v", err)
	}
	if got.Macroblocks != 1 || got.LastMBXY != 0 || !got.EndOfSlice || !got.EndOfFrame {
		t.Fatalf("slice result = %+v, want one partitioned P MB frame end", got)
	}
	if m.MacroblockTyp[0] != wantType || m.CBPTable[0] != 0 || m.QScaleTable[0] != 24 || m.SliceTable[0] != wantSlice {
		t.Fatalf("tables type/cbp/q/slice = %#x/%#x/%d/%d, want %#x/0/24/%d",
			m.MacroblockTyp[0], m.CBPTable[0], m.QScaleTable[0], m.SliceTable[0], wantType, wantSlice)
	}
	assertH264RowsHigh(t, "high partitioned p y", dst.Y, 0, dst.LumaStride, 16, 16, want.Y, want.LumaStride)
	assertH264RowsHigh(t, "high partitioned p cb", dst.Cb, 0, dst.ChromaStride, 8, 8, want.Cb, want.ChromaStride)
	assertH264RowsHigh(t, "high partitioned p cr", dst.Cr, 0, dst.ChromaStride, 8, 8, want.Cr, want.ChromaStride)
}
