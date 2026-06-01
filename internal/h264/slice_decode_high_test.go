// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

import "testing"

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

func TestValidateSimpleFrameSliceDecodeHighAllowsHigh10P420NoWeight(t *testing.T) {
	m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, 10, 1, 1, false, PictureTypeP)
	sh.RefCount = [2]uint32{1, 0}

	if err := validateSimpleFrameSliceDecodeInputsHigh(m, dst, sh, 4); err != nil {
		t.Fatalf("high P validation err = %v, want nil", err)
	}
}

func TestValidateSimpleFrameSliceDecodeHighAllowsHigh10B420NoWeight(t *testing.T) {
	m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, 10, 1, 1, false, PictureTypeB)
	sh.RefCount = [2]uint32{1, 1}

	if err := validateSimpleFrameSliceDecodeInputsHigh(m, dst, sh, 4); err != nil {
		t.Fatalf("high B validation err = %v, want nil", err)
	}
}

func TestValidateSimpleFrameSliceDecodeHighRejectsStagedBoundaries(t *testing.T) {
	tests := []struct {
		name     string
		bitDepth int32
		chroma   int32
		format   int
		deblock  bool
		slice    int32
	}{
		{name: "8-bit", bitDepth: 8, chroma: 8, format: 1, slice: PictureTypeI},
		{name: "9-bit", bitDepth: 9, chroma: 9, format: 1, slice: PictureTypeI},
		{name: "12-bit", bitDepth: 12, chroma: 12, format: 1, slice: PictureTypeI},
		{name: "14-bit", bitDepth: 14, chroma: 14, format: 1, slice: PictureTypeI},
		{name: "unequal-depth", bitDepth: 10, chroma: 12, format: 1, slice: PictureTypeI},
		{name: "monochrome", bitDepth: 10, chroma: 10, format: 0, slice: PictureTypeI},
		{name: "422", bitDepth: 10, chroma: 10, format: 2, slice: PictureTypeI},
		{name: "444", bitDepth: 10, chroma: 10, format: 3, slice: PictureTypeI},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, dst, sh := highFrameSliceDecodeFixture(t, tt.bitDepth, tt.format, tt.deblock, tt.slice)
			sh.SPS.BitDepthChroma = tt.chroma

			if err := validateSimpleFrameSliceDecodeInputsHigh(m, dst, sh, 4); err != ErrUnsupported {
				t.Fatalf("high validation err = %v, want ErrUnsupported", err)
			}
		})
	}
}

func TestValidateSimpleFrameSliceDecodeHighAllowsHigh10Deblocking(t *testing.T) {
	for _, sliceType := range []int32{PictureTypeI, PictureTypeP, PictureTypeB} {
		t.Run(pictureTypeName(sliceType), func(t *testing.T) {
			m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, 10, 1, 1, true, sliceType)
			if sliceType == PictureTypeP {
				sh.RefCount = [2]uint32{1, 0}
			}
			if sliceType == PictureTypeB {
				sh.RefCount = [2]uint32{1, 1}
			}

			if err := validateSimpleFrameSliceDecodeInputsHigh(m, dst, sh, 4); err != nil {
				t.Fatalf("high deblock validation err = %v, want nil", err)
			}
		})
	}
}

func TestValidateSimpleFrameSliceDecodeHighRejectsWeightedB(t *testing.T) {
	for _, tt := range []struct {
		name string
		run  func(*SliceHeader)
	}{
		{
			name: "implicit",
			run: func(sh *SliceHeader) {
				sh.PPS.WeightedBipredIDC = 2
				sh.PredWeightTable.UseWeight = 2
				sh.PredWeightTable.UseWeightChroma = 2
			},
		},
		{
			name: "explicit table",
			run: func(sh *SliceHeader) {
				sh.PPS.WeightedBipredIDC = 1
				sh.PredWeightTable.UseWeight = 1
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
	m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, 10, 1, 1, false, PictureTypeP)
	sh.RefCount = [2]uint32{1, 0}
	sh.PPS.WeightedPred = 1
	sh.PredWeightTable.UseWeight = 1

	if err := validateSimpleFrameSliceDecodeInputsHigh(m, dst, sh, 4); err != nil {
		t.Fatalf("high weighted P validation err = %v, want nil", err)
	}
}

func TestValidateSimpleFrameSliceDecodeHighWeightedPStillRejectsStagedBoundaries(t *testing.T) {
	tests := []struct {
		name     string
		bitDepth int32
		chroma   int32
		format   int
		deblock  bool
		slice    int32
	}{
		{name: "9-bit", bitDepth: 9, chroma: 9, format: 1, slice: PictureTypeP},
		{name: "12-bit", bitDepth: 12, chroma: 12, format: 1, slice: PictureTypeP},
		{name: "unequal-depth", bitDepth: 10, chroma: 12, format: 1, slice: PictureTypeP},
		{name: "422", bitDepth: 10, chroma: 10, format: 2, slice: PictureTypeP},
		{name: "444", bitDepth: 10, chroma: 10, format: 3, slice: PictureTypeP},
		{name: "b-slice", bitDepth: 10, chroma: 10, format: 1, slice: PictureTypeB},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, dst, sh := highFrameSliceDecodeFixture(t, tt.bitDepth, tt.format, tt.deblock, tt.slice)
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
	const bitDepth = 10
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
}

func TestDecodeCABACFrameSliceHighReconstructsIntra4x4NoResidual(t *testing.T) {
	const bitDepth = 10
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
}

func TestDecodeCAVLCFrameSliceHighReconstructsPSkip(t *testing.T) {
	const bitDepth = 10
	m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, bitDepth, 1, 1, false, PictureTypeP)
	sh.QScale = 24
	sh.RefCount = [2]uint32{1, 0}
	ref := makeH264SliceDecodePictureHigh(1, 1, 1)
	fillH264MotionCompPlaneHigh(ref.Y, 73, bitDepth)
	fillH264MotionCompPlaneHigh(ref.Cb, 91, bitDepth)
	fillH264MotionCompPlaneHigh(ref.Cr, 119, bitDepth)
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
}

func TestDecodeCAVLCFrameSliceHighReconstructsP16x16NoResidual(t *testing.T) {
	const bitDepth = 10
	m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, bitDepth, 1, 1, false, PictureTypeP)
	sh.QScale = 24
	sh.RefCount = [2]uint32{1, 0}
	ref := makeH264SliceDecodePictureHigh(1, 1, 1)
	fillH264MotionCompPlaneHigh(ref.Y, 37, bitDepth)
	fillH264MotionCompPlaneHigh(ref.Cb, 53, bitDepth)
	fillH264MotionCompPlaneHigh(ref.Cr, 71, bitDepth)
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
}

func TestDecodeCABACFrameSliceHighReconstructsPSkip(t *testing.T) {
	const bitDepth = 10
	m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, bitDepth, 1, 1, false, PictureTypeP)
	sh.PPS.CABAC = 1
	sh.QScale = 24
	sh.RefCount = [2]uint32{1, 0}
	ref := makeH264SliceDecodePictureHigh(1, 1, 1)
	fillH264MotionCompPlaneHigh(ref.Y, 83, bitDepth)
	fillH264MotionCompPlaneHigh(ref.Cb, 107, bitDepth)
	fillH264MotionCompPlaneHigh(ref.Cr, 131, bitDepth)
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
}

func TestDecodeCABACFrameSliceHighReconstructsP16x16NoResidual(t *testing.T) {
	const bitDepth = 10
	m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, bitDepth, 1, 1, false, PictureTypeP)
	sh.PPS.CABAC = 1
	sh.QScale = 24
	sh.RefCount = [2]uint32{1, 0}
	ref := makeH264SliceDecodePictureHigh(1, 1, 1)
	fillH264MotionCompPlaneHigh(ref.Y, 43, bitDepth)
	fillH264MotionCompPlaneHigh(ref.Cb, 61, bitDepth)
	fillH264MotionCompPlaneHigh(ref.Cr, 79, bitDepth)
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
}

func TestDecodeCAVLCFrameSliceHighRejectsUnsupportedBeforeEntropy(t *testing.T) {
	m, dst, sh := highFrameSliceDecodeFixture(t, 10, 1, true, PictureTypeI)
	sh.SPS.ChromaFormatIDC = 2
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

func TestDecodeFrameSliceDataHighRejectsUnsupportedCABACBeforeStartup(t *testing.T) {
	m, dst, sh := highFrameSliceDecodeFixture(t, 10, 1, true, PictureTypeI)
	sh.PPS.CABAC = 1
	sh.SPS.ChromaFormatIDC = 2
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
