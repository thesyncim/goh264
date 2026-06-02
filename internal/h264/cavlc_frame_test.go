// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

import (
	"errors"
	"testing"
)

func TestDecodeCAVLCFrameIntra4x4MacroblockWritesState(t *testing.T) {
	m, err := newMacroblockTables(2, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	sps := &SPS{BitDepthLuma: 8, ChromaFormatIDC: 1, FrameMBSOnlyFlag: 1}
	pps := cavlcFlatQMulPPS()
	pps.SPS = sps
	gb := newBitReader(cavlcBitString("11111111111111111100100"))

	got, err := m.decodeCAVLCFrameMacroblock(&gb, cavlcFrameMacroblockInput{
		MBXY:         0,
		SliceNum:     2,
		SliceType:    PictureTypeI,
		SliceTypeNoS: PictureTypeI,
		QScale:       20,
		PPS:          pps,
		SPS:          sps,
	})
	if err != nil {
		t.Fatalf("decode frame intra mb failed: %v", err)
	}
	if !got.IsIntra || got.IsInter || got.MBType != MBTypeIntra4x4 || got.CBP != 0 || got.CBPTable != 0 || got.QScale != 20 {
		t.Fatalf("result = intra %v inter %v type %#x cbp %d cbpTable %d qscale %d", got.IsIntra, got.IsInter, got.MBType, got.CBP, got.CBPTable, got.QScale)
	}
	if got.Neighbors.TopXY != -1 || got.Neighbors.LeftXY != ([2]int{-1, -1}) {
		t.Fatalf("neighbors = %+v, want no top/left", got.Neighbors)
	}
	if m.MacroblockTyp[0] != MBTypeIntra4x4 || m.CBPTable[0] != 0 || m.QScaleTable[0] != 20 || m.SliceTable[0] != 2 {
		t.Fatalf("tables type/cbp/q/slice = %#x/%#x/%d/%d", m.MacroblockTyp[0], m.CBPTable[0], m.QScaleTable[0], m.SliceTable[0])
	}
	if m.ChromaPred[0] != 0 || got.ChromaPred != intraPredDC1288x8 {
		t.Fatalf("chroma pred table/result = %d/%d, want raw 0/result dc128", m.ChromaPred[0], got.ChromaPred)
	}
	for i, v := range m.NonZeroCount[0] {
		if v != 0 {
			t.Fatalf("nnz[%d] = %d, want 0", i, v)
		}
	}
	if gb.bitPos != 23 {
		t.Fatalf("consumed %d bits, want 23", gb.bitPos)
	}
}

func TestDecodeCAVLCFrameIntraPCMMacroblockAlignsAndWritesState(t *testing.T) {
	m, err := newMacroblockTables(2, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	sps := &SPS{BitDepthLuma: 8, ChromaFormatIDC: 1, FrameMBSOnlyFlag: 1}
	pps := cavlcFlatQMulPPS()
	pps.SPS = sps
	pcm := h264ReconstructIntraPCM(1, 33)
	buf := append([]byte{0x0d, 0x00}, pcm...)
	gb := newBitReader(buf)

	got, err := m.decodeCAVLCFrameMacroblock(&gb, cavlcFrameMacroblockInput{
		MBXY:         0,
		SliceNum:     5,
		SliceType:    PictureTypeI,
		SliceTypeNoS: PictureTypeI,
		QScale:       20,
		PPS:          pps,
		SPS:          sps,
	})
	if err != nil {
		t.Fatalf("decode frame intra pcm failed: %v", err)
	}
	if !got.IsIntra || got.IsInter || got.MBType != MBTypeIntraPCM || got.QScale != 0 || len(got.IntraPCM) != len(pcm) {
		t.Fatalf("result intra/inter/type/q/pcm = %v/%v/%#x/%d/%d", got.IsIntra, got.IsInter, got.MBType, got.QScale, len(got.IntraPCM))
	}
	if got.IntraPCM[0] != pcm[0] || got.IntraPCM[len(pcm)-1] != pcm[len(pcm)-1] {
		t.Fatalf("pcm endpoints = %d/%d, want %d/%d", got.IntraPCM[0], got.IntraPCM[len(pcm)-1], pcm[0], pcm[len(pcm)-1])
	}
	if m.MacroblockTyp[0] != MBTypeIntraPCM || m.CBPTable[0] != 0 || m.QScaleTable[0] != 0 || m.SliceTable[0] != 5 {
		t.Fatalf("tables type/cbp/q/slice = %#x/%#x/%d/%d", m.MacroblockTyp[0], m.CBPTable[0], m.QScaleTable[0], m.SliceTable[0])
	}
	for i, v := range m.NonZeroCount[0] {
		if v != 16 {
			t.Fatalf("nnz[%d] = %d, want 16", i, v)
		}
	}
	if gb.bitPos != uint32(16+len(pcm)*8) {
		t.Fatalf("consumed %d bits, want %d", gb.bitPos, 16+len(pcm)*8)
	}
}

func TestDecodeCAVLCFieldPictureIntraPCMMacroblockPassesMBAFFGuard(t *testing.T) {
	for _, tt := range []struct {
		name    string
		picture int32
		mbXY    func(*macroblockTables) int
	}{
		{name: "top", picture: PictureTopField, mbXY: func(*macroblockTables) int { return 0 }},
		{name: "bottom", picture: PictureBottomField, mbXY: func(m *macroblockTables) int { return m.MBStride }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			m, err := newMacroblockTables(1, 2, 1)
			if err != nil {
				t.Fatal(err)
			}
			sps := &SPS{BitDepthLuma: 8, ChromaFormatIDC: 1, FrameMBSOnlyFlag: 0, MBAFF: 1}
			pps := cavlcFlatQMulPPS()
			pps.SPS = sps
			sh := &SliceHeader{
				SliceType:        PictureTypeI,
				SliceTypeNoS:     PictureTypeI,
				PictureStructure: tt.picture,
				PPS:              pps,
				SPS:              sps,
				QScale:           20,
			}
			state := newCAVLCFrameSliceState(int(sh.QScale))
			pcm := h264ReconstructIntraPCM(1, 29)
			buf := append([]byte{0x0d, 0x00}, pcm...)
			gb := newBitReader(buf)
			mbXY := tt.mbXY(m)

			got, err := m.decodeCAVLCFrameSliceMacroblock(&gb, sh, &state, mbXY, 6)
			if err != nil {
				t.Fatalf("decode field-picture intra pcm failed: %v", err)
			}
			wantType := MBTypeIntraPCM | MBTypeInterlaced
			if got.MBType != wantType || !got.IsIntra || got.MBFieldDecodingFlag != 0 || state.MBFieldDecodingFlag != 0 {
				t.Fatalf("result type/intra/field = %#x/%v/%d/%d, want field-picture intra pcm without MBAFF flag", got.MBType, got.IsIntra, got.MBFieldDecodingFlag, state.MBFieldDecodingFlag)
			}
			if m.SliceTable[mbXY] != 6 || m.MacroblockTyp[mbXY] != wantType || m.QScaleTable[mbXY] != 0 {
				t.Fatalf("tables slice/type/q = %d/%#x/%d, want 6/%#x/0", m.SliceTable[mbXY], m.MacroblockTyp[mbXY], m.QScaleTable[mbXY], wantType)
			}
			if gb.bitPos != uint32(16+len(pcm)*8) {
				t.Fatalf("consumed %d bits, want intra pcm without field flag", gb.bitPos)
			}
		})
	}
}

func TestDecodeCAVLCFrameHighIntraPCMMacroblockReadsBitDepthPayload(t *testing.T) {
	m, err := newMacroblockTables(1, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	sps := &SPS{BitDepthLuma: 10, ChromaFormatIDC: 1, FrameMBSOnlyFlag: 1}
	pps := cavlcFlatQMulPPS()
	pps.SPS = sps
	pcm := h264ReconstructIntraPCMHigh(1, 10, 71)
	buf := append([]byte{0x0d, 0x00}, pcm...)
	gb := newBitReader(buf)

	got, err := m.decodeCAVLCFrameMacroblock(&gb, cavlcFrameMacroblockInput{
		MBXY:         0,
		SliceNum:     7,
		SliceType:    PictureTypeI,
		SliceTypeNoS: PictureTypeI,
		QScale:       20,
		PPS:          pps,
		SPS:          sps,
	})
	if err != nil {
		t.Fatalf("decode high frame intra pcm failed: %v", err)
	}
	if len(got.IntraPCM) != len(pcm) || got.IntraPCM[0] != pcm[0] || got.IntraPCM[len(pcm)-1] != pcm[len(pcm)-1] {
		t.Fatalf("high pcm length/endpoints = %d/%d/%d, want %d/%d/%d",
			len(got.IntraPCM), got.IntraPCM[0], got.IntraPCM[len(got.IntraPCM)-1],
			len(pcm), pcm[0], pcm[len(pcm)-1])
	}
	if gb.bitPos != uint32(16+len(pcm)*8) {
		t.Fatalf("consumed high pcm %d bits, want %d", gb.bitPos, 16+len(pcm)*8)
	}
}

func TestDecodeCAVLCFrameP16x16MacroblockAppliesNeighborMotion(t *testing.T) {
	m, err := newMacroblockTables(3, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	sps := &SPS{BitDepthLuma: 8, ChromaFormatIDC: 1, FrameMBSOnlyFlag: 1}
	pps := cavlcFlatQMulPPS()
	pps.SPS = sps
	sliceNum := uint16(7)
	mbXY := 5
	topLeftXY := 0
	topXY := 1
	topRightXY := 2
	leftXY := 4

	for _, xy := range []int{topLeftXY, topXY, topRightXY, leftXY} {
		m.SliceTable[xy] = sliceNum
		m.MacroblockTyp[xy] = MBType16x16 | MBTypeP0L0
	}
	m.RefIndex[0][4*leftXY+1] = 0
	m.MotionVal[0][int(m.MB2BXY[leftXY])+3] = [2]int16{1, 11}
	m.RefIndex[0][4*topXY+2] = 0
	m.MotionVal[0][int(m.MB2BXY[topXY])+3*m.BStride] = [2]int16{3, 33}
	m.RefIndex[0][4*topRightXY+2] = 0
	m.MotionVal[0][int(m.MB2BXY[topRightXY])+3*m.BStride] = [2]int16{2, 22}

	gb := newBitReader(cavlcBitString("1111"))
	got, err := m.decodeCAVLCFrameMacroblock(&gb, cavlcFrameMacroblockInput{
		MBXY:         mbXY,
		SliceNum:     sliceNum,
		SliceType:    PictureTypeP,
		SliceTypeNoS: PictureTypeP,
		QScale:       24,
		RefCount:     [2]uint32{1, 0},
		PPS:          pps,
		SPS:          sps,
	})
	if err != nil {
		t.Fatalf("decode frame p16x16 mb failed: %v", err)
	}
	wantType := MBType16x16 | MBTypeP0L0
	if !got.IsInter || got.IsIntra || got.MBType != wantType || got.CBP != 0 || got.QScale != 24 {
		t.Fatalf("result = intra %v inter %v type %#x cbp %d qscale %d", got.IsIntra, got.IsInter, got.MBType, got.CBP, got.QScale)
	}
	if got.Neighbors.TopType != wantType || got.Neighbors.TopRightType != wantType || got.Neighbors.LeftType[0] != wantType {
		t.Fatalf("neighbor types top/topright/left = %#x/%#x/%#x", got.Neighbors.TopType, got.Neighbors.TopRightType, got.Neighbors.LeftType[0])
	}
	if m.MacroblockTyp[mbXY] != wantType || m.CBPTable[mbXY] != 0 || m.QScaleTable[mbXY] != 24 || m.SliceTable[mbXY] != sliceNum {
		t.Fatalf("tables type/cbp/q/slice = %#x/%#x/%d/%d", m.MacroblockTyp[mbXY], m.CBPTable[mbXY], m.QScaleTable[mbXY], m.SliceTable[mbXY])
	}
	bXY := int(m.MB2BXY[mbXY])
	if m.MotionVal[0][bXY] != ([2]int16{2, 22}) || m.MotionVal[0][bXY+3+3*m.BStride] != ([2]int16{2, 22}) {
		t.Fatalf("motion writeback = %v/%v, want [2 22]", m.MotionVal[0][bXY], m.MotionVal[0][bXY+3+3*m.BStride])
	}
	if got := m.RefIndex[0][4*mbXY : 4*mbXY+4]; got[0] != 0 || got[1] != 0 || got[2] != 0 || got[3] != 0 {
		t.Fatalf("refs = %v, want all 0", got)
	}
	for i, v := range m.NonZeroCount[mbXY] {
		if v != 0 {
			t.Fatalf("nnz[%d] = %d, want 0", i, v)
		}
	}
	if gb.bitPos != 4 {
		t.Fatalf("consumed %d bits, want 4", gb.bitPos)
	}
}

func TestDecodeCAVLCFrameSlicePskipRunWritesSkipState(t *testing.T) {
	m, err := newMacroblockTables(3, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	sps := &SPS{BitDepthLuma: 8, ChromaFormatIDC: 1, FrameMBSOnlyFlag: 1}
	pps := cavlcFlatQMulPPS()
	pps.SPS = sps
	sliceNum := uint16(11)
	mbXY := 5
	leftXY := 4
	topXY := 1
	topRightXY := 2

	for _, xy := range []int{leftXY, topXY, topRightXY} {
		m.SliceTable[xy] = sliceNum
		m.MacroblockTyp[xy] = MBType16x16 | MBTypeP0L0
	}
	m.RefIndex[0][4*leftXY+1] = 0
	m.MotionVal[0][int(m.MB2BXY[leftXY])+3] = [2]int16{10, 20}
	m.RefIndex[0][4*topXY+2] = 0
	m.MotionVal[0][int(m.MB2BXY[topXY])+3*m.BStride] = [2]int16{30, 40}
	m.RefIndex[0][4*topRightXY+2] = 0
	m.MotionVal[0][int(m.MB2BXY[topRightXY])+3*m.BStride] = [2]int16{20, 10}

	sh := &SliceHeader{
		SliceType:        PictureTypeP,
		SliceTypeNoS:     PictureTypeP,
		PictureStructure: PictureFrame,
		PPS:              pps,
		SPS:              sps,
		QScale:           24,
		RefCount:         [2]uint32{1, 0},
	}
	state := newCAVLCFrameSliceState(int(sh.QScale))
	gb := newBitReader(cavlcBitString("010"))

	got, err := m.decodeCAVLCFrameSliceMacroblock(&gb, sh, &state, mbXY, sliceNum)
	if err != nil {
		t.Fatalf("decode pskip run failed: %v", err)
	}
	wantType := MBType16x16 | MBTypeP0L0 | MBTypeP1L0 | MBTypeSkip
	if !got.Skipped || !got.IsInter || got.MBType != wantType || got.QScale != 24 {
		t.Fatalf("result skipped/inter/type/q = %v/%v/%#x/%d", got.Skipped, got.IsInter, got.MBType, got.QScale)
	}
	if state.MBSkipRun != 0 {
		t.Fatalf("skip run state = %d, want 0", state.MBSkipRun)
	}
	if m.MacroblockTyp[mbXY] != wantType || m.CBPTable[mbXY] != 0 || m.QScaleTable[mbXY] != 24 || m.SliceTable[mbXY] != sliceNum {
		t.Fatalf("tables type/cbp/q/slice = %#x/%#x/%d/%d", m.MacroblockTyp[mbXY], m.CBPTable[mbXY], m.QScaleTable[mbXY], m.SliceTable[mbXY])
	}
	bXY := int(m.MB2BXY[mbXY])
	if m.MotionVal[0][bXY] != ([2]int16{20, 20}) || m.MotionVal[0][bXY+3+3*m.BStride] != ([2]int16{20, 20}) {
		t.Fatalf("pskip motion = %v/%v, want [20 20]", m.MotionVal[0][bXY], m.MotionVal[0][bXY+3+3*m.BStride])
	}
	if gb.bitPos != 3 {
		t.Fatalf("consumed %d bits, want 3", gb.bitPos)
	}
}

func TestDecodeCAVLCFrameSliceRunZeroFallsThroughToMacroblock(t *testing.T) {
	m, err := newMacroblockTables(1, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	sps := &SPS{BitDepthLuma: 8, ChromaFormatIDC: 1, FrameMBSOnlyFlag: 1}
	pps := cavlcFlatQMulPPS()
	pps.SPS = sps
	sh := &SliceHeader{
		SliceType:        PictureTypeP,
		SliceTypeNoS:     PictureTypeP,
		PictureStructure: PictureFrame,
		PPS:              pps,
		SPS:              sps,
		QScale:           24,
		RefCount:         [2]uint32{1, 0},
	}
	state := newCAVLCFrameSliceState(int(sh.QScale))
	gb := newBitReader(cavlcBitString("11111"))

	got, err := m.decodeCAVLCFrameSliceMacroblock(&gb, sh, &state, 0, 4)
	if err != nil {
		t.Fatalf("decode run-zero p16x16 failed: %v", err)
	}
	if got.Skipped || !got.IsInter || got.MBType != (MBType16x16|MBTypeP0L0) || got.QScale != 24 {
		t.Fatalf("result skipped/inter/type/q = %v/%v/%#x/%d", got.Skipped, got.IsInter, got.MBType, got.QScale)
	}
	if state.MBSkipRun != cavlcMBSkipRunUnset {
		t.Fatalf("skip run state = %d, want unset", state.MBSkipRun)
	}
	if m.SliceTable[0] != 4 || m.MacroblockTyp[0] != (MBType16x16|MBTypeP0L0) {
		t.Fatalf("state slice/type = %d/%#x", m.SliceTable[0], m.MacroblockTyp[0])
	}
	if gb.bitPos != 5 {
		t.Fatalf("consumed %d bits, want 5", gb.bitPos)
	}
}

func TestDecodeCAVLCFrameMBAFFFrameMacroblockDecodesAfterFieldFlag(t *testing.T) {
	m, err := newMacroblockTables(1, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	sps := &SPS{BitDepthLuma: 8, ChromaFormatIDC: 1, FrameMBSOnlyFlag: 0, MBAFF: 1}
	pps := cavlcFlatQMulPPS()
	pps.SPS = sps
	sh := &SliceHeader{
		SliceType:        PictureTypeI,
		SliceTypeNoS:     PictureTypeI,
		PictureStructure: PictureFrame,
		PPS:              pps,
		SPS:              sps,
		QScale:           20,
	}
	state := newCAVLCFrameSliceState(int(sh.QScale))
	gb := newBitReader(cavlcBitString("0" + "11111111111111111100100"))

	got, err := m.decodeCAVLCFrameSliceMacroblock(&gb, sh, &state, 0, 3)
	if err != nil {
		t.Fatalf("decode frame-coded mbaff failed: %v", err)
	}
	if got.MBFieldDecodingFlag != 0 || state.MBFieldDecodingFlag != 0 || got.MBType != MBTypeIntra4x4 || !got.IsIntra {
		t.Fatalf("field result/state/type/intra = %d/%d/%#x/%v, want 0/0/intra4x4/true", got.MBFieldDecodingFlag, state.MBFieldDecodingFlag, got.MBType, got.IsIntra)
	}
	if gb.bitPos != 24 {
		t.Fatalf("consumed %d bits, want field flag plus intra4x4", gb.bitPos)
	}
	if m.SliceTable[0] != 3 || m.MacroblockTyp[0] != MBTypeIntra4x4 || m.QScaleTable[0] != 20 {
		t.Fatalf("tables slice/type/q = %d/%#x/%d, want 3/intra4x4/20", m.SliceTable[0], m.MacroblockTyp[0], m.QScaleTable[0])
	}
}

func TestDecodeCAVLCFrameMBAFFFieldMacroblockMarksInterlaced(t *testing.T) {
	m, err := newMacroblockTables(1, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	sps := &SPS{BitDepthLuma: 8, ChromaFormatIDC: 1, FrameMBSOnlyFlag: 0, MBAFF: 1}
	pps := cavlcFlatQMulPPS()
	pps.SPS = sps
	sh := &SliceHeader{
		SliceType:        PictureTypeI,
		SliceTypeNoS:     PictureTypeI,
		PictureStructure: PictureFrame,
		PPS:              pps,
		SPS:              sps,
		QScale:           20,
	}
	state := newCAVLCFrameSliceState(int(sh.QScale))
	gb := newBitReader(cavlcBitString("1" + "11111111111111111100100"))

	got, err := m.decodeCAVLCFrameSliceMacroblock(&gb, sh, &state, 0, 3)
	if err != nil {
		t.Fatalf("decode field-coded mbaff failed: %v", err)
	}
	wantType := MBTypeIntra4x4 | MBTypeInterlaced
	if got.MBFieldDecodingFlag != 1 || state.MBFieldDecodingFlag != 1 || got.MBType != wantType || !got.IsIntra {
		t.Fatalf("field result/state/type/intra = %d/%d/%#x/%v, want 1/1/%#x/true", got.MBFieldDecodingFlag, state.MBFieldDecodingFlag, got.MBType, got.IsIntra, wantType)
	}
	if gb.bitPos != 24 {
		t.Fatalf("consumed %d bits, want field flag plus intra4x4", gb.bitPos)
	}
	if m.SliceTable[0] != 3 || m.MacroblockTyp[0] != wantType || m.QScaleTable[0] != 20 {
		t.Fatalf("tables slice/type/q = %d/%#x/%d, want 3/%#x/20", m.SliceTable[0], m.MacroblockTyp[0], m.QScaleTable[0], wantType)
	}
}

func TestDecodeCAVLCFrameMBAFFFieldInterMacroblockDoublesRefCount(t *testing.T) {
	m, err := newMacroblockTables(1, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	sps := &SPS{BitDepthLuma: 8, ChromaFormatIDC: 1, FrameMBSOnlyFlag: 0, MBAFF: 1}
	pps := cavlcFlatQMulPPS()
	pps.SPS = sps
	sh := &SliceHeader{
		SliceType:        PictureTypeP,
		SliceTypeNoS:     PictureTypeP,
		PictureStructure: PictureFrame,
		PPS:              pps,
		SPS:              sps,
		QScale:           24,
		RefCount:         [2]uint32{1, 0},
	}
	state := newCAVLCFrameSliceState(int(sh.QScale))
	gb := newBitReader(cavlcBitString("1111111"))

	got, err := m.decodeCAVLCFrameSliceMacroblock(&gb, sh, &state, 0, 3)
	if err != nil {
		t.Fatalf("decode field-coded mbaff inter failed: %v", err)
	}
	wantType := MBType16x16 | MBTypeP0L0 | MBTypeInterlaced
	if got.MBFieldDecodingFlag != 1 || state.MBFieldDecodingFlag != 1 || got.MBType != wantType || got.Inter.Ref[0][0] != 0 {
		t.Fatalf("field/ref/type = %d/%d/%#x, want field-coded ref0 type %#x", got.MBFieldDecodingFlag, got.Inter.Ref[0][0], got.MBType, wantType)
	}
	if gb.bitPos != 7 {
		t.Fatalf("consumed %d bits, want skip_run+field+mb_type+ref+mvd+cbp", gb.bitPos)
	}
}

func TestDecodeCAVLCFrameMBAFFSkipRunWritesFieldCodedTerminalSkip(t *testing.T) {
	m, err := newMacroblockTables(1, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	sps := &SPS{BitDepthLuma: 8, ChromaFormatIDC: 1, FrameMBSOnlyFlag: 0, MBAFF: 1}
	pps := cavlcFlatQMulPPS()
	pps.SPS = sps
	sh := &SliceHeader{
		SliceType:        PictureTypeP,
		SliceTypeNoS:     PictureTypeP,
		PictureStructure: PictureFrame,
		PPS:              pps,
		SPS:              sps,
		QScale:           24,
		RefCount:         [2]uint32{1, 0},
	}
	state := newCAVLCFrameSliceState(int(sh.QScale))
	gb := newBitReader(cavlcBitString("0101"))

	got, err := m.decodeCAVLCFrameSliceMacroblock(&gb, sh, &state, 0, 5)
	if err != nil {
		t.Fatalf("decode field-coded mbaff skip failed: %v", err)
	}
	if state.MBSkipRun != 0 {
		t.Fatalf("skip run = %d, want post-decrement 0", state.MBSkipRun)
	}
	wantType := MBType16x16 | MBTypeP0L0 | MBTypeP1L0 | MBTypeSkip | MBTypeInterlaced
	if !got.Skipped || !got.IsInter || got.MBFieldDecodingFlag != 1 || state.MBFieldDecodingFlag != 1 || got.MBType != wantType {
		t.Fatalf("field skip result = skipped:%v inter:%v field:%d/%d type:%#x, want %#x", got.Skipped, got.IsInter, got.MBFieldDecodingFlag, state.MBFieldDecodingFlag, got.MBType, wantType)
	}
	if gb.bitPos != 4 {
		t.Fatalf("consumed %d bits, want skip_run plus terminal field flag", gb.bitPos)
	}
	if m.SliceTable[0] != 5 || m.MacroblockTyp[0] != wantType || m.QScaleTable[0] != 24 {
		t.Fatalf("tables slice/type/q = %d/%#x/%d, want 5/%#x/24", m.SliceTable[0], m.MacroblockTyp[0], m.QScaleTable[0], wantType)
	}
}

func TestDecodeCAVLCFrameMBAFFSkipRunWritesFrameCodedTerminalSkip(t *testing.T) {
	m, err := newMacroblockTables(1, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	sps := &SPS{BitDepthLuma: 8, ChromaFormatIDC: 1, FrameMBSOnlyFlag: 0, MBAFF: 1}
	pps := cavlcFlatQMulPPS()
	pps.SPS = sps
	sh := &SliceHeader{
		SliceType:        PictureTypeP,
		SliceTypeNoS:     PictureTypeP,
		PictureStructure: PictureFrame,
		PPS:              pps,
		SPS:              sps,
		QScale:           24,
		RefCount:         [2]uint32{1, 0},
	}
	state := newCAVLCFrameSliceState(int(sh.QScale))
	gb := newBitReader(cavlcBitString("0100"))

	got, err := m.decodeCAVLCFrameSliceMacroblock(&gb, sh, &state, 0, 5)
	if err != nil {
		t.Fatalf("decode frame-coded mbaff skip failed: %v", err)
	}
	wantType := MBType16x16 | MBTypeP0L0 | MBTypeP1L0 | MBTypeSkip
	if !got.Skipped || !got.IsInter || got.MBType != wantType || got.MBFieldDecodingFlag != 0 || state.MBFieldDecodingFlag != 0 {
		t.Fatalf("skip result = skipped:%v inter:%v type:%#x field:%d/%d, want frame-coded pskip", got.Skipped, got.IsInter, got.MBType, got.MBFieldDecodingFlag, state.MBFieldDecodingFlag)
	}
	if state.MBSkipRun != 0 {
		t.Fatalf("skip run = %d, want post-decrement 0", state.MBSkipRun)
	}
	if gb.bitPos != 4 {
		t.Fatalf("consumed %d bits, want skip_run plus terminal field flag", gb.bitPos)
	}
	if m.SliceTable[0] != 5 || m.MacroblockTyp[0] != wantType || m.QScaleTable[0] != 24 {
		t.Fatalf("tables slice/type/q = %d/%#x/%d, want 5/pskip/24", m.SliceTable[0], m.MacroblockTyp[0], m.QScaleTable[0])
	}
}

func TestDecodeCAVLCFrameMBAFFSkipRunDefersFieldFlagAndWritesFrameSkip(t *testing.T) {
	m, err := newMacroblockTables(1, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	sps := &SPS{BitDepthLuma: 8, ChromaFormatIDC: 1, FrameMBSOnlyFlag: 0, MBAFF: 1}
	pps := cavlcFlatQMulPPS()
	pps.SPS = sps
	sh := &SliceHeader{
		SliceType:        PictureTypeP,
		SliceTypeNoS:     PictureTypeP,
		PictureStructure: PictureFrame,
		PPS:              pps,
		SPS:              sps,
		QScale:           24,
		RefCount:         [2]uint32{1, 0},
	}
	state := newCAVLCFrameSliceState(int(sh.QScale))
	gb := newBitReader(cavlcBitString("0111"))

	got, err := m.decodeCAVLCFrameSliceMacroblock(&gb, sh, &state, 0, 5)
	if err != nil {
		t.Fatalf("decode nonterminal frame-coded mbaff skip failed: %v", err)
	}
	if state.MBSkipRun != 1 {
		t.Fatalf("skip run = %d, want post-decrement 1", state.MBSkipRun)
	}
	wantType := MBType16x16 | MBTypeP0L0 | MBTypeP1L0 | MBTypeSkip
	if !got.Skipped || !got.IsInter || got.MBFieldDecodingFlag != 0 || state.MBFieldDecodingFlag != 0 || got.MBType != wantType {
		t.Fatalf("skip result/state/type = %v/%v/%d/%d/%#x, want frame-coded pskip with no field flag yet", got.Skipped, got.IsInter, got.MBFieldDecodingFlag, state.MBFieldDecodingFlag, got.MBType)
	}
	if gb.bitPos != 3 {
		t.Fatalf("consumed %d bits, want skip_run only", gb.bitPos)
	}
	if m.SliceTable[0] != 5 || m.MacroblockTyp[0] != wantType || m.QScaleTable[0] != 24 {
		t.Fatalf("tables slice/type/q = %d/%#x/%d, want 5/pskip/24", m.SliceTable[0], m.MacroblockTyp[0], m.QScaleTable[0])
	}
}

func TestDecodeCAVLCFrameBDirectUnsupportedBeforeWriteback(t *testing.T) {
	m, err := newMacroblockTables(1, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	sps := &SPS{BitDepthLuma: 8, ChromaFormatIDC: 1, FrameMBSOnlyFlag: 1}
	pps := cavlcFlatQMulPPS()
	pps.SPS = sps
	gb := newBitReader(cavlcBitString("11"))

	_, err = m.decodeCAVLCFrameMacroblock(&gb, cavlcFrameMacroblockInput{
		MBXY:                0,
		SliceNum:            3,
		SliceType:           PictureTypeB,
		SliceTypeNoS:        PictureTypeB,
		QScale:              18,
		RefCount:            [2]uint32{1, 1},
		DirectSpatialMVPred: true,
		PPS:                 pps,
		SPS:                 sps,
	})
	if !errors.Is(err, ErrUnsupported) {
		t.Fatalf("err = %v, want ErrUnsupported", err)
	}
	if m.SliceTable[0] != ^uint16(0) || m.MacroblockTyp[0] != 0 {
		t.Fatalf("state changed on unsupported direct: slice/type = %d/%#x", m.SliceTable[0], m.MacroblockTyp[0])
	}
}

func TestWriteBackCAVLCFrameMBAFFBskipMapsSpatialDirectNeighborRefs(t *testing.T) {
	m, err := newMacroblockTables(3, 4, 1)
	if err != nil {
		t.Fatal(err)
	}
	colTables, err := newMacroblockTables(3, 4, 1)
	if err != nil {
		t.Fatal(err)
	}
	sps := &SPS{BitDepthLuma: 8, ChromaFormatIDC: 1, FrameMBSOnlyFlag: 0, MBAFF: 1, Direct8x8InferenceFlag: 1}
	pps := cavlcFlatQMulPPS()
	pps.SPS = sps
	sh := &SliceHeader{
		SliceType:        PictureTypeB,
		SliceTypeNoS:     PictureTypeB,
		PictureStructure: PictureFrame,
		PPS:              pps,
		SPS:              sps,
	}
	sliceNum := uint16(9)
	mbXY := 2*m.MBStride + 1
	neighborType := MBType16x16 | MBTypeL0L1 | MBTypeInterlaced
	for _, xy := range []int{mbXY - 1, mbXY - m.MBStride - 1, mbXY - m.MBStride, mbXY - m.MBStride + 1} {
		m.SliceTable[xy] = sliceNum
		m.MacroblockTyp[xy] = neighborType
		for list := 0; list < 2; list++ {
			for i := 0; i < 4; i++ {
				m.RefIndex[list][4*xy+i] = 2
			}
			bxy := int(m.MB2BXY[xy])
			m.MotionVal[list][bxy+3] = [2]int16{6, 8}
			m.MotionVal[list][bxy+3*m.BStride] = [2]int16{10, 12}
			m.MotionVal[list][bxy+3+3*m.BStride] = [2]int16{14, 16}
		}
	}
	for xy := range colTables.MacroblockTyp {
		colTables.MacroblockTyp[xy] = MBType16x16 | MBTypeP0L0
	}
	idr := &DecodedFrame{poc: 0}
	secondRef := &DecodedFrame{poc: -2}
	col := &DecodedFrame{
		poc:    4,
		tables: colTables,
		refEntries: [2][]simpleRefEntry{
			{{frame: idr}, {frame: secondRef}},
			{{frame: idr}, {frame: secondRef}},
		},
	}
	var work frameMacroblockDecodeWork

	got, err := m.writeBackCAVLCFrameBSkipMacroblockWithDirectWorkFieldGuard(sh, 22, mbXY, sliceNum, 0, h264DirectMotionContext{
		RefEntries: [2][]simpleRefEntry{
			{{frame: idr}, {frame: secondRef}},
			{{frame: col}, {frame: secondRef}},
		},
		PictureStructure:    PictureFrame,
		DirectSpatialMVPred: true,
		Direct8x8Inference:  true,
		X264Build:           165,
	}, &work, false)
	if err != nil {
		t.Fatalf("write back frame-mbaff spatial bskip failed: %v", err)
	}
	if got.MBType&MBTypeInterlaced != 0 || !got.Skipped || !isDirect(got.MBType) {
		t.Fatalf("result type/skipped = %#x/%v, want frame-coded direct skip", got.MBType, got.Skipped)
	}
	if m.RefIndex[0][4*mbXY] != 1 || m.RefIndex[1][4*mbXY] != 1 {
		t.Fatalf("written refs = %d/%d, want MBAFF-mapped frame refs 1/1", m.RefIndex[0][4*mbXY], m.RefIndex[1][4*mbXY])
	}
}
