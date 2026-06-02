// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

import (
	"errors"
	"reflect"
	"testing"
)

func TestDecodeCABACFrameIntra4x4MacroblockWritesState(t *testing.T) {
	m, err := newMacroblockTables(2, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	sps := &SPS{BitDepthLuma: 8, ChromaFormatIDC: 1, FrameMBSOnlyFlag: 1}
	pps := cavlcFlatQMulPPS()
	pps.SPS = sps
	src := &scriptedCABACSource{
		bits: append(append([]int{0}, repeatCABACBits(16, 1)...), []int{
			0,
			0, 0, 0, 0,
			0,
		}...),
	}

	got, err := m.decodeCABACFrameMacroblock(src, cabacFrameMacroblockInput{
		MBXY:         0,
		SliceNum:     2,
		SliceType:    PictureTypeI,
		SliceTypeNoS: PictureTypeI,
		QScale:       20,
		PPS:          pps,
		SPS:          sps,
	})
	if err != nil {
		t.Fatalf("decode cabac frame intra mb failed: %v", err)
	}
	if !got.IsIntra || got.IsInter || got.MBType != MBTypeIntra4x4 || got.CBP != 0 || got.CBPTable != 0 || got.QScale != 20 || got.LastQScaleDiff != 0 {
		t.Fatalf("result = intra %v inter %v type %#x cbp %d cbpTable %d qscale %d diff %d", got.IsIntra, got.IsInter, got.MBType, got.CBP, got.CBPTable, got.QScale, got.LastQScaleDiff)
	}
	if m.MacroblockTyp[0] != MBTypeIntra4x4 || m.CBPTable[0] != 0 || m.QScaleTable[0] != 20 || m.SliceTable[0] != 2 {
		t.Fatalf("tables type/cbp/q/slice = %#x/%#x/%d/%d", m.MacroblockTyp[0], m.CBPTable[0], m.QScaleTable[0], m.SliceTable[0])
	}
	if m.ChromaPred[0] != 0 || got.ChromaPred != intraPredDC1288x8 {
		t.Fatalf("chroma pred table/result = %d/%d, want raw 0/result dc128", m.ChromaPred[0], got.ChromaPred)
	}
	dst := int(m.MB2BRXY[0])
	if m.Intra4x4Pred[dst] != intraPredDC || m.Intra4x4Pred[dst+6] != intraPredDC {
		t.Fatalf("intra pred writeback endpoints = %d/%d, want DC", m.Intra4x4Pred[dst], m.Intra4x4Pred[dst+6])
	}
	for i, v := range m.NonZeroCount[0] {
		if v != 0 {
			t.Fatalf("nnz[%d] = %d, want 0", i, v)
		}
	}
	wantIndexes(t, src, append(append([]int{3}, repeatCABACBits(16, 68)...), []int{64, 73, 74, 75, 76, 77}...))
}

func TestDecodeCABACFrameIntraPCMMacroblockWritesState(t *testing.T) {
	m, err := newMacroblockTables(1, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	sps := &SPS{BitDepthLuma: 8, ChromaFormatIDC: 1, FrameMBSOnlyFlag: 1}
	pps := cavlcFlatQMulPPS()
	pps.SPS = sps
	pcm := h264ReconstructIntraPCM(1, 41)
	src := &scriptedCABACSource{
		bits:  []int{1},
		terms: []int{1},
		pcm:   append([]byte(nil), pcm...),
	}
	sh := &SliceHeader{
		SliceType:        PictureTypeI,
		SliceTypeNoS:     PictureTypeI,
		PictureStructure: PictureFrame,
		PPS:              pps,
		SPS:              sps,
		QScale:           23,
	}
	state := &cabacFrameSliceState{QScale: int(sh.QScale), LastQScaleDiff: -2}

	got, err := m.decodeCABACFrameSliceMacroblock(src, sh, state, 0, 6)
	if err != nil {
		t.Fatalf("decode cabac frame intra pcm failed: %v", err)
	}
	if !got.IsIntra || got.IsInter || got.MBType != MBTypeIntraPCM || got.CBP != 0 || got.CBPTable != 0xf7ef || got.QScale != 0 || got.LastQScaleDiff != 0 {
		t.Fatalf("result intra/inter/type/cbp/cbpTable/q/diff = %v/%v/%#x/%d/%#x/%d/%d", got.IsIntra, got.IsInter, got.MBType, got.CBP, got.CBPTable, got.QScale, got.LastQScaleDiff)
	}
	if state.PrevMBSkipped || state.LastQScaleDiff != 0 {
		t.Fatalf("slice state skipped/diff = %v/%d, want false/0", state.PrevMBSkipped, state.LastQScaleDiff)
	}
	if len(got.IntraPCM) != len(pcm) || len(got.Intra.IntraPCM) != len(pcm) {
		t.Fatalf("pcm lengths result/intra = %d/%d, want %d", len(got.IntraPCM), len(got.Intra.IntraPCM), len(pcm))
	}
	if got.IntraPCM[0] != pcm[0] || got.IntraPCM[len(pcm)-1] != pcm[len(pcm)-1] {
		t.Fatalf("pcm endpoints = %d/%d, want %d/%d", got.IntraPCM[0], got.IntraPCM[len(pcm)-1], pcm[0], pcm[len(pcm)-1])
	}
	if m.MacroblockTyp[0] != MBTypeIntraPCM || m.CBPTable[0] != 0xf7ef || m.QScaleTable[0] != 0 || m.SliceTable[0] != 6 {
		t.Fatalf("tables type/cbp/q/slice = %#x/%#x/%d/%d", m.MacroblockTyp[0], m.CBPTable[0], m.QScaleTable[0], m.SliceTable[0])
	}
	if m.ChromaPred[0] != 0 {
		t.Fatalf("chroma pred = %d, want 0", m.ChromaPred[0])
	}
	for i, v := range m.NonZeroCount[0] {
		if v != 16 {
			t.Fatalf("nnz[%d] = %d, want 16", i, v)
		}
	}
	if !reflect.DeepEqual(src.pcmReadSizes, []int{len(pcm)}) {
		t.Fatalf("pcm read sizes = %v, want [%d]", src.pcmReadSizes, len(pcm))
	}
	wantIndexes(t, src, []int{3})
}

func TestDecodeCABACFieldPictureIntraPCMMacroblockPassesMBAFFGuard(t *testing.T) {
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
			pcm := h264ReconstructIntraPCM(1, 43)
			src := &scriptedCABACSource{
				bits:  []int{1},
				terms: []int{1},
				pcm:   append([]byte(nil), pcm...),
			}
			sh := &SliceHeader{
				SliceType:        PictureTypeI,
				SliceTypeNoS:     PictureTypeI,
				PictureStructure: tt.picture,
				PPS:              pps,
				SPS:              sps,
				QScale:           20,
			}
			state := &cabacFrameSliceState{QScale: int(sh.QScale)}
			mbXY := tt.mbXY(m)

			got, err := m.decodeCABACFrameSliceMacroblock(src, sh, state, mbXY, 6)
			if err != nil {
				t.Fatalf("decode field-picture intra pcm failed: %v", err)
			}
			wantType := MBTypeIntraPCM | MBTypeInterlaced
			if got.MBType != wantType || !got.IsIntra || got.MBFieldDecodingFlag != 0 || state.MBFieldDecodingFlag != 0 {
				t.Fatalf("result type/intra/field = %#x/%v/%d/%d, want intra pcm field-picture without MBAFF flag", got.MBType, got.IsIntra, got.MBFieldDecodingFlag, state.MBFieldDecodingFlag)
			}
			if m.SliceTable[mbXY] != 6 || m.MacroblockTyp[mbXY] != wantType || m.QScaleTable[mbXY] != 0 {
				t.Fatalf("tables slice/type/q = %d/%#x/%d, want 6/intra pcm/0", m.SliceTable[mbXY], m.MacroblockTyp[mbXY], m.QScaleTable[mbXY])
			}
			wantIndexes(t, src, []int{3})
		})
	}
}

func TestDecodeCABACFieldPictureResidualUsesFieldContexts(t *testing.T) {
	m, err := newMacroblockTables(2, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	const sliceNum = uint16(6)
	m.SliceTable[0] = sliceNum
	m.MacroblockTyp[0] = MBTypeIntra4x4 | MBTypeInterlaced

	sps := &SPS{BitDepthLuma: 8, ChromaFormatIDC: 1, FrameMBSOnlyFlag: 0, MBAFF: 1}
	pps := cavlcFlatQMulPPS()
	pps.SPS = sps
	src := &scriptedCABACSource{
		bits: []int{
			1,    // intra16x16
			0, 0, // no luma/chroma CBP
			0, 1, // horizontal intra16x16 pred, valid with left-only samples
			0,       // chroma pred DC
			0,       // qscale diff absent
			1, 1, 1, // luma DC coded, significant, last
			0, // coeff_abs_level_minus1 == 0
		},
		terms: []int{0},
	}

	got, err := m.decodeCABACFrameMacroblock(src, cabacFrameMacroblockInput{
		MBXY:         1,
		SliceNum:     sliceNum,
		SliceType:    PictureTypeI,
		SliceTypeNoS: PictureTypeI,
		QScale:       20,
		FieldPicture: true,
		PPS:          pps,
		SPS:          sps,
	})
	if err != nil {
		t.Fatalf("decode field-picture cabac residual failed: %v", err)
	}
	wantType := MBTypeIntra16x16 | MBTypeInterlaced
	if got.MBType != wantType || !got.IsIntra || got.CBPTable&0x100 == 0 {
		t.Fatalf("result type/intra/cbpTable = %#x/%v/%#x, want field intra16 with luma DC", got.MBType, got.IsIntra, got.CBPTable)
	}
	if m.MacroblockTyp[1] != wantType {
		t.Fatalf("written mb type = %#x, want %#x", m.MacroblockTyp[1], wantType)
	}
	wantIndexes(t, src, []int{3, 6, 7, 9, 10, 64, 60, 87, 277, 338, 228})
}

func TestDecodeCABACFrameMBAFFFrameMacroblockDecodesAfterFieldFlag(t *testing.T) {
	m, err := newMacroblockTables(1, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	sps := &SPS{BitDepthLuma: 8, ChromaFormatIDC: 1, FrameMBSOnlyFlag: 0, MBAFF: 1}
	pps := cavlcFlatQMulPPS()
	pps.SPS = sps
	src := &scriptedCABACSource{
		bits: append(append([]int{0, 0}, repeatCABACBits(16, 1)...), []int{
			0,
			0, 0, 0, 0,
			0,
		}...),
	}
	sh := &SliceHeader{
		SliceType:        PictureTypeI,
		SliceTypeNoS:     PictureTypeI,
		PictureStructure: PictureFrame,
		PPS:              pps,
		SPS:              sps,
		QScale:           20,
	}
	state := &cabacFrameSliceState{QScale: int(sh.QScale)}

	got, err := m.decodeCABACFrameSliceMacroblock(src, sh, state, 0, 3)
	if err != nil {
		t.Fatalf("decode frame-coded mbaff failed: %v", err)
	}
	if got.MBFieldDecodingFlag != 0 || state.MBFieldDecodingFlag != 0 || got.MBType != MBTypeIntra4x4 || !got.IsIntra {
		t.Fatalf("field result/state/type/intra = %d/%d/%#x/%v, want 0/0/intra4x4/true", got.MBFieldDecodingFlag, state.MBFieldDecodingFlag, got.MBType, got.IsIntra)
	}
	if m.SliceTable[0] != 3 || m.MacroblockTyp[0] != MBTypeIntra4x4 || m.QScaleTable[0] != 20 {
		t.Fatalf("tables slice/type/q = %d/%#x/%d, want 3/intra4x4/20", m.SliceTable[0], m.MacroblockTyp[0], m.QScaleTable[0])
	}
	wantIndexes(t, src, append(append([]int{70, 3}, repeatCABACBits(16, 68)...), []int{64, 73, 74, 75, 76, 77}...))
}

func TestDecodeCABACFrameMBAFFFieldMacroblockMarksInterlaced(t *testing.T) {
	m, err := newMacroblockTables(1, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	sps := &SPS{BitDepthLuma: 8, ChromaFormatIDC: 1, FrameMBSOnlyFlag: 0, MBAFF: 1}
	pps := cavlcFlatQMulPPS()
	pps.SPS = sps
	src := &scriptedCABACSource{bits: []int{
		0,
		1,
		0, 0, 0,
		0,
		0, 0,
		0, 0, 0, 0,
		0,
	}}
	sh := &SliceHeader{
		SliceType:        PictureTypeP,
		SliceTypeNoS:     PictureTypeP,
		PictureStructure: PictureFrame,
		PPS:              pps,
		SPS:              sps,
		QScale:           24,
		RefCount:         [2]uint32{1, 0},
	}
	state := &cabacFrameSliceState{QScale: int(sh.QScale)}

	got, err := m.decodeCABACFrameSliceMacroblock(src, sh, state, 0, 3)
	if err != nil {
		t.Fatalf("decode field-coded mbaff failed: %v", err)
	}
	wantType := MBType16x16 | MBTypeP0L0 | MBTypeInterlaced
	if got.MBFieldDecodingFlag != 1 || state.MBFieldDecodingFlag != 1 || got.MBType != wantType || !got.IsInter || got.Inter.Ref[0][0] != 0 {
		t.Fatalf("field result/state/type/inter/ref = %d/%d/%#x/%v/%d, want 1/1/%#x/true/0", got.MBFieldDecodingFlag, state.MBFieldDecodingFlag, got.MBType, got.IsInter, got.Inter.Ref[0][0], wantType)
	}
	if m.SliceTable[0] != 3 || m.MacroblockTyp[0] != wantType || m.QScaleTable[0] != 24 {
		t.Fatalf("tables slice/type/q = %d/%#x/%d, want 3/%#x/24", m.SliceTable[0], m.MacroblockTyp[0], m.QScaleTable[0], wantType)
	}
	wantIndexes(t, src, []int{11, 70, 14, 15, 16, 54, 40, 47, 73, 74, 75, 76, 77})
}

func TestDecodeCABACFrameHighIntraPCMMacroblockReadsBitDepthPayload(t *testing.T) {
	m, err := newMacroblockTables(1, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	sps := &SPS{BitDepthLuma: 12, ChromaFormatIDC: 2, FrameMBSOnlyFlag: 1}
	pps := cavlcFlatQMulPPS()
	pps.SPS = sps
	pcm := h264ReconstructIntraPCMHigh(2, 12, 83)
	src := &scriptedCABACSource{
		bits:  []int{1},
		terms: []int{1},
		pcm:   append([]byte(nil), pcm...),
	}
	sh := &SliceHeader{
		SliceType:        PictureTypeI,
		SliceTypeNoS:     PictureTypeI,
		PictureStructure: PictureFrame,
		PPS:              pps,
		SPS:              sps,
		QScale:           23,
	}
	state := &cabacFrameSliceState{QScale: int(sh.QScale)}

	got, err := m.decodeCABACFrameSliceMacroblock(src, sh, state, 0, 8)
	if err != nil {
		t.Fatalf("decode high cabac frame intra pcm failed: %v", err)
	}
	if len(got.IntraPCM) != len(pcm) || len(src.pcmReadSizes) != 1 || src.pcmReadSizes[0] != len(pcm) {
		t.Fatalf("high cabac pcm lengths/read sizes = %d/%v, want %d", len(got.IntraPCM), src.pcmReadSizes, len(pcm))
	}
	if got.IntraPCM[0] != pcm[0] || got.IntraPCM[len(pcm)-1] != pcm[len(pcm)-1] {
		t.Fatalf("high cabac pcm endpoints = %d/%d, want %d/%d", got.IntraPCM[0], got.IntraPCM[len(pcm)-1], pcm[0], pcm[len(pcm)-1])
	}
}

func TestDecodeCABACResidualPayloadSelectsDCTElemWidth(t *testing.T) {
	for _, tt := range []struct {
		name     string
		bitDepth int32
		want     int32
	}{
		{name: "8-bit-dctelem", bitDepth: 8, want: 0},
		{name: "high-bit-depth-dctelem", bitDepth: 10, want: 65536},
	} {
		t.Run(tt.name, func(t *testing.T) {
			pps := cavlcFlatQMulPPS()
			for i := range pps.Dequant4Buffer[3][20] {
				pps.Dequant4Buffer[3][20][i] = 1 << 22
			}
			sps := &SPS{BitDepthLuma: tt.bitDepth, ChromaFormatIDC: 0}
			var ctx cavlcResidualContext
			src := &scriptedCABACSource{
				bits:  []int{0, 1, 0, 1, 1, 0, 0, 0, 0},
				signs: []int32{1 << 22},
			}

			qscale, _, cbpTable, _, err := ctx.decodeCABACResidualPayload(src, pps, sps, MBType16x16|MBTypeP0L0, 1, 20, 0, residualDecodeCacheResult{})
			if err != nil {
				t.Fatalf("decode cabac residual payload failed: %v", err)
			}
			if qscale != 20 || cbpTable != 1 {
				t.Fatalf("qscale/cbpTable = %d/%#x, want 20/1", qscale, cbpTable)
			}
			pos := int(h264ZigzagScanCAVLC[1])
			if ctx.MB[pos] != tt.want {
				t.Fatalf("luma coeff[%d] = %d, want %d", pos, ctx.MB[pos], tt.want)
			}
		})
	}
}

func TestDecodeCABACFrameP16x16MacroblockAppliesNeighborMotion(t *testing.T) {
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

	src := &scriptedCABACSource{bits: []int{
		0, 0, 0,
		0, 0,
		0, 0, 0, 0,
		0,
	}}
	got, err := m.decodeCABACFrameMacroblock(src, cabacFrameMacroblockInput{
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
		t.Fatalf("decode cabac frame p16x16 mb failed: %v", err)
	}
	wantType := MBType16x16 | MBTypeP0L0
	if !got.IsInter || got.IsIntra || got.MBType != wantType || got.CBP != 0 || got.QScale != 24 {
		t.Fatalf("result = intra %v inter %v type %#x cbp %d qscale %d", got.IsIntra, got.IsInter, got.MBType, got.CBP, got.QScale)
	}
	if m.MacroblockTyp[mbXY] != wantType || m.CBPTable[mbXY] != 0 || m.QScaleTable[mbXY] != 24 || m.SliceTable[mbXY] != sliceNum {
		t.Fatalf("tables type/cbp/q/slice = %#x/%#x/%d/%d", m.MacroblockTyp[mbXY], m.CBPTable[mbXY], m.QScaleTable[mbXY], m.SliceTable[mbXY])
	}
	bXY := int(m.MB2BXY[mbXY])
	if m.MotionVal[0][bXY] != ([2]int16{2, 22}) || m.MotionVal[0][bXY+3+3*m.BStride] != ([2]int16{2, 22}) {
		t.Fatalf("motion writeback = %v/%v, want [2 22]", m.MotionVal[0][bXY], m.MotionVal[0][bXY+3+3*m.BStride])
	}
	dst := int(m.MB2BRXY[mbXY])
	for i := 0; i < 7; i++ {
		if m.MVDTable[0][dst+i] != ([2]uint8{}) {
			t.Fatalf("mvd[%d] = %v, want zero", i, m.MVDTable[0][dst+i])
		}
	}
	wantIndexes(t, src, []int{14, 15, 16, 40, 47, 76, 76, 76, 76, 77})
}

func TestDecodeCABACFrameP16x16MacroblockDecodesRefAndMVD(t *testing.T) {
	m, err := newMacroblockTables(1, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	sps := &SPS{BitDepthLuma: 8, ChromaFormatIDC: 1, FrameMBSOnlyFlag: 1}
	pps := cavlcFlatQMulPPS()
	pps.SPS = sps
	src := &scriptedCABACSource{
		bits: []int{
			0, 0, 0,
			0,
			1, 1, 1, 0,
			0,
			0, 0, 0, 0,
			0,
		},
		signs: []int32{-3},
	}

	got, err := m.decodeCABACFrameMacroblock(src, cabacFrameMacroblockInput{
		MBXY:         0,
		SliceNum:     4,
		SliceType:    PictureTypeP,
		SliceTypeNoS: PictureTypeP,
		QScale:       24,
		RefCount:     [2]uint32{2, 0},
		PPS:          pps,
		SPS:          sps,
	})
	if err != nil {
		t.Fatalf("decode cabac ref/mvd mb failed: %v", err)
	}
	if got.Inter.Ref[0][0] != 0 || got.Inter.MVD[0][0] != ([2]int32{-3, 0}) {
		t.Fatalf("inter ref/mvd = %d/%v, want 0/[-3 0]", got.Inter.Ref[0][0], got.Inter.MVD[0][0])
	}
	bXY := int(m.MB2BXY[0])
	if m.MotionVal[0][bXY] != ([2]int16{-3, 0}) || m.MotionVal[0][bXY+3+3*m.BStride] != ([2]int16{-3, 0}) {
		t.Fatalf("motion writeback = %v/%v, want [-3 0]", m.MotionVal[0][bXY], m.MotionVal[0][bXY+3+3*m.BStride])
	}
	dst := int(m.MB2BRXY[0])
	if m.MVDTable[0][dst] != ([2]uint8{3, 0}) || m.MVDTable[0][dst+6] != ([2]uint8{3, 0}) {
		t.Fatalf("mvd table endpoints = %v/%v, want [3 0]", m.MVDTable[0][dst], m.MVDTable[0][dst+6])
	}
	wantIndexes(t, src, []int{14, 15, 16, 54, 40, 43, 44, 45, 47, 73, 74, 75, 76, 77})
}

func TestDecodeCABACFrameP8x8MacroblockDecodesSubPartitions(t *testing.T) {
	m, err := newMacroblockTables(1, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	sps := &SPS{BitDepthLuma: 8, ChromaFormatIDC: 1, FrameMBSOnlyFlag: 1}
	pps := cavlcFlatQMulPPS()
	pps.SPS = sps
	src := &scriptedCABACSource{bits: []int{
		0, 0, 1,
		1, 1, 1, 1,
		0, 0,
		0, 0,
		0, 0,
		0, 0,
		0, 0, 0, 0,
		0,
	}}

	got, err := m.decodeCABACFrameMacroblock(src, cabacFrameMacroblockInput{
		MBXY:         0,
		SliceNum:     4,
		SliceType:    PictureTypeP,
		SliceTypeNoS: PictureTypeP,
		QScale:       24,
		RefCount:     [2]uint32{1, 0},
		PPS:          pps,
		SPS:          sps,
	})
	if err != nil {
		t.Fatalf("decode cabac p8x8 mb failed: %v", err)
	}
	wantType := MBType8x8 | MBTypeP0L0 | MBTypeP1L0
	if got.MBType != wantType || got.Inter.PartitionCount != 4 {
		t.Fatalf("type/partitions = %#x/%d, want %#x/4", got.MBType, got.Inter.PartitionCount, wantType)
	}
	for i := 0; i < 4; i++ {
		if got.Inter.SubMBType[i] != (MBType16x16|MBTypeP0L0) || got.Inter.SubPartitionCount[i] != 1 || got.Inter.Ref[0][i] != 0 {
			t.Fatalf("sub[%d] type/part/ref = %#x/%d/%d", i, got.Inter.SubMBType[i], got.Inter.SubPartitionCount[i], got.Inter.Ref[0][i])
		}
	}
	if got := m.RefIndex[0][0:4]; got[0] != 0 || got[1] != 0 || got[2] != 0 || got[3] != 0 {
		t.Fatalf("refs = %v, want all 0", got)
	}
	if m.MacroblockTyp[0] != wantType || m.CBPTable[0] != 0 || m.QScaleTable[0] != 24 || m.SliceTable[0] != 4 {
		t.Fatalf("tables type/cbp/q/slice = %#x/%#x/%d/%d", m.MacroblockTyp[0], m.CBPTable[0], m.QScaleTable[0], m.SliceTable[0])
	}
	wantIndexes(t, src, []int{14, 15, 16, 21, 21, 21, 21, 40, 47, 40, 47, 40, 47, 40, 47, 73, 74, 75, 76, 77})
}

func TestDecodeCABACFrameB8x8DirectSubMacroblocks(t *testing.T) {
	m, col, idr := newTemporalDirectTestTables(t, MBType8x8|MBTypeP0L0|MBTypeP1L0)
	bxy := int(col.tables.MB2BXY[0])
	for i8, mv := range [4][2]int16{{4, 0}, {0, 4}, {2, 2}, {6, 2}} {
		col.tables.RefIndex[0][i8] = 0
		x8 := i8 & 1
		y8 := i8 >> 1
		col.tables.MotionVal[0][bxy+x8*3+y8*3*col.tables.BStride] = mv
	}
	sps := &SPS{BitDepthLuma: 8, ChromaFormatIDC: 1, FrameMBSOnlyFlag: 1, Direct8x8InferenceFlag: 1}
	pps := cavlcFlatQMulPPS()
	pps.SPS = sps
	src := &scriptedCABACSource{bits: []int{
		1, 1, 1, 1, 1, 1,
		0, 0, 0, 0,
		0, 0, 0, 0,
		0,
	}}

	got, err := m.decodeCABACFrameMacroblock(src, cabacFrameMacroblockInput{
		MBXY:          0,
		SliceNum:      10,
		SliceType:     PictureTypeB,
		SliceTypeNoS:  PictureTypeB,
		QScale:        18,
		RefCount:      [2]uint32{1, 1},
		DCT8x8Allowed: true,
		Direct: h264DirectMotionContext{
			RefEntries: [2][]simpleRefEntry{
				{{frame: idr}},
				{{frame: col}},
			},
			CurPOC:             2,
			Direct8x8Inference: true,
		},
		PPS: pps,
		SPS: sps,
	})
	if err != nil {
		t.Fatalf("decode cabac b8x8 direct-sub failed: %v", err)
	}
	wantType := MBType8x8 | MBTypeP0L0 | MBTypeP0L1 | MBTypeP1L0 | MBTypeP1L1
	if got.MBType != wantType || m.MacroblockTyp[0] != wantType || got.CBP != 0 || got.QScale != 18 || got.LastQScaleDiff != 0 {
		t.Fatalf("type/cbp/q/diff = %#x/%#x/%d/%d/%d", got.MBType, m.MacroblockTyp[0], got.CBP, got.QScale, got.LastQScaleDiff)
	}
	wantSub := MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2
	for i8 := 0; i8 < 4; i8++ {
		if got.Inter.SubMBType[i8] != wantSub {
			t.Fatalf("sub[%d] = %#x, want %#x", i8, got.Inter.SubMBType[i8], wantSub)
		}
	}
	if m.MotionVal[0][bxy] != ([2]int16{2, 0}) || m.MotionVal[1][bxy] != ([2]int16{-2, 0}) {
		t.Fatalf("direct-sub written mv0/mv1 = %v/%v", m.MotionVal[0][bxy], m.MotionVal[1][bxy])
	}
	if m.MVDTable[0][0] != ([2]uint8{}) || m.MVDTable[1][0] != ([2]uint8{}) {
		t.Fatalf("direct-sub mvd table = %v/%v", m.MVDTable[0][0], m.MVDTable[1][0])
	}
}

func TestDecodeCABACFrameSlicePskipWritesCABACSkipState(t *testing.T) {
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

	dst := int(m.MB2BRXY[mbXY])
	for i := 0; i < 7; i++ {
		m.MVDTable[0][dst+i] = [2]uint8{9, 9}
	}
	sh := &SliceHeader{
		SliceType:        PictureTypeP,
		SliceTypeNoS:     PictureTypeP,
		PictureStructure: PictureFrame,
		PPS:              pps,
		SPS:              sps,
		QScale:           24,
		RefCount:         [2]uint32{1, 0},
	}
	state := &cabacFrameSliceState{QScale: int(sh.QScale), LastQScaleDiff: 3}
	src := &scriptedCABACSource{bits: []int{1}}

	got, err := m.decodeCABACFrameSliceMacroblock(src, sh, state, mbXY, sliceNum)
	if err != nil {
		t.Fatalf("decode cabac pskip failed: %v", err)
	}
	wantType := MBType16x16 | MBTypeP0L0 | MBTypeP1L0 | MBTypeSkip
	if !got.Skipped || !got.IsInter || got.MBType != wantType || got.QScale != 24 || got.LastQScaleDiff != 0 {
		t.Fatalf("result skipped/inter/type/q/diff = %v/%v/%#x/%d/%d", got.Skipped, got.IsInter, got.MBType, got.QScale, got.LastQScaleDiff)
	}
	if !state.PrevMBSkipped || state.LastQScaleDiff != 0 {
		t.Fatalf("state skipped/diff = %v/%d, want true/0", state.PrevMBSkipped, state.LastQScaleDiff)
	}
	if m.MacroblockTyp[mbXY] != wantType || m.CBPTable[mbXY] != 0 || m.QScaleTable[mbXY] != 24 || m.SliceTable[mbXY] != sliceNum {
		t.Fatalf("tables type/cbp/q/slice = %#x/%#x/%d/%d", m.MacroblockTyp[mbXY], m.CBPTable[mbXY], m.QScaleTable[mbXY], m.SliceTable[mbXY])
	}
	bXY := int(m.MB2BXY[mbXY])
	if m.MotionVal[0][bXY] != ([2]int16{20, 20}) || m.MotionVal[0][bXY+3+3*m.BStride] != ([2]int16{20, 20}) {
		t.Fatalf("pskip motion = %v/%v, want [20 20]", m.MotionVal[0][bXY], m.MotionVal[0][bXY+3+3*m.BStride])
	}
	for i := 0; i < 7; i++ {
		if m.MVDTable[0][dst+i] != ([2]uint8{}) {
			t.Fatalf("cabac pskip mvd[%d] = %v, want zero", i, m.MVDTable[0][dst+i])
		}
	}
	wantIndexes(t, src, []int{13})
}

func TestDecodeCABACFrameMBAFFSkipReadsBottomSkipAndFieldFlag(t *testing.T) {
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
	state := &cabacFrameSliceState{QScale: int(sh.QScale), LastQScaleDiff: 3}
	src := &scriptedCABACSource{bits: []int{1, 0, 1}}

	got, err := m.decodeCABACFrameSliceMacroblock(src, sh, state, 0, 5)
	if err != nil {
		t.Fatalf("decode field-coded mbaff skip failed: %v", err)
	}
	if !state.PrevMBSkipped || state.NextMBSkipped || state.LastQScaleDiff != 0 {
		t.Fatalf("skip state prev/next/diff = %v/%v/%d, want true/false/0", state.PrevMBSkipped, state.NextMBSkipped, state.LastQScaleDiff)
	}
	wantType := MBType16x16 | MBTypeP0L0 | MBTypeP1L0 | MBTypeSkip | MBTypeInterlaced
	if !got.Skipped || !got.IsInter || got.MBType != wantType || got.MBFieldDecodingFlag != 1 || state.MBFieldDecodingFlag != 1 || got.LastQScaleDiff != 0 {
		t.Fatalf("field skip result = skipped:%v inter:%v type:%#x field:%d/%d diff:%d, want %#x",
			got.Skipped, got.IsInter, got.MBType, got.MBFieldDecodingFlag, state.MBFieldDecodingFlag, got.LastQScaleDiff, wantType)
	}
	if m.SliceTable[0] != 5 || m.MacroblockTyp[0] != wantType || m.QScaleTable[0] != 24 {
		t.Fatalf("tables slice/type/q = %d/%#x/%d, want 5/%#x/24", m.SliceTable[0], m.MacroblockTyp[0], m.QScaleTable[0], wantType)
	}
	if m.SliceTable[m.MBStride] != ^uint16(0) || m.MacroblockTyp[m.MBStride] != 0 {
		t.Fatalf("bottom tables changed before bottom decode: slice/type = %d/%#x", m.SliceTable[m.MBStride], m.MacroblockTyp[m.MBStride])
	}
	wantIndexes(t, src, []int{11, 11, 70})
}

func TestDecodeCABACFrameMBAFFSkipWritesFrameCodedTopSkip(t *testing.T) {
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
	state := &cabacFrameSliceState{QScale: int(sh.QScale), LastQScaleDiff: 3}
	src := &scriptedCABACSource{bits: []int{1, 0, 0}}

	got, err := m.decodeCABACFrameSliceMacroblock(src, sh, state, 0, 5)
	if err != nil {
		t.Fatalf("decode frame-coded mbaff skip failed: %v", err)
	}
	wantType := MBType16x16 | MBTypeP0L0 | MBTypeP1L0 | MBTypeSkip
	if !got.Skipped || !got.IsInter || got.MBType != wantType || got.MBFieldDecodingFlag != 0 || state.MBFieldDecodingFlag != 0 || got.LastQScaleDiff != 0 {
		t.Fatalf("skip result = skipped:%v inter:%v type:%#x field:%d/%d diff:%d, want frame-coded pskip", got.Skipped, got.IsInter, got.MBType, got.MBFieldDecodingFlag, state.MBFieldDecodingFlag, got.LastQScaleDiff)
	}
	if !state.PrevMBSkipped || state.NextMBSkipped || state.LastQScaleDiff != 0 {
		t.Fatalf("skip state prev/next/diff = %v/%v/%d, want true/false/0", state.PrevMBSkipped, state.NextMBSkipped, state.LastQScaleDiff)
	}
	if m.SliceTable[0] != 5 || m.MacroblockTyp[0] != wantType || m.QScaleTable[0] != 24 {
		t.Fatalf("tables slice/type/q = %d/%#x/%d, want 5/pskip/24", m.SliceTable[0], m.MacroblockTyp[0], m.QScaleTable[0])
	}
	if m.SliceTable[m.MBStride] != ^uint16(0) || m.MacroblockTyp[m.MBStride] != 0 {
		t.Fatalf("bottom tables changed before bottom decode: slice/type = %d/%#x", m.SliceTable[m.MBStride], m.MacroblockTyp[m.MBStride])
	}
	wantIndexes(t, src, []int{11, 11, 70})
}

func TestDecodeCABACFrameMBAFFBottomSkipReusesDecodedNextSkip(t *testing.T) {
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
	state := &cabacFrameSliceState{QScale: int(sh.QScale)}
	src := &scriptedCABACSource{bits: []int{1, 1}}

	top, err := m.decodeCABACFrameSliceMacroblock(src, sh, state, 0, 5)
	if err != nil {
		t.Fatalf("decode top frame-coded mbaff skip failed: %v", err)
	}
	bottom, err := m.decodeCABACFrameSliceMacroblock(src, sh, state, m.MBStride, 5)
	if err != nil {
		t.Fatalf("decode bottom frame-coded mbaff skip failed: %v", err)
	}
	wantType := MBType16x16 | MBTypeP0L0 | MBTypeP1L0 | MBTypeSkip
	if top.MBType != wantType || bottom.MBType != wantType || !top.Skipped || !bottom.Skipped {
		t.Fatalf("top/bottom skip type = %#x/%#x skipped %v/%v, want pskip pair", top.MBType, bottom.MBType, top.Skipped, bottom.Skipped)
	}
	if m.SliceTable[0] != 5 || m.SliceTable[m.MBStride] != 5 || m.MacroblockTyp[0] != wantType || m.MacroblockTyp[m.MBStride] != wantType {
		t.Fatalf("tables top slice/type %d/%#x bottom slice/type %d/%#x, want pskip pair", m.SliceTable[0], m.MacroblockTyp[0], m.SliceTable[m.MBStride], m.MacroblockTyp[m.MBStride])
	}
	wantIndexes(t, src, []int{11, 11})
}

func TestDecodeCABACFrameMBAFFRowStartPredictionSelectsSkipContext(t *testing.T) {
	m, err := newMacroblockTables(2, 4, 1)
	if err != nil {
		t.Fatal(err)
	}
	const sliceNum = uint16(14)
	mbXY := 2 * m.MBStride
	topPairXY := mbXY - 2*m.MBStride
	bottomPairXY := mbXY - m.MBStride

	m.SliceTable[topPairXY] = sliceNum
	m.MacroblockTyp[topPairXY] = MBTypeInterlaced | MBType16x16 | MBTypeP0L0
	m.SliceTable[bottomPairXY] = sliceNum
	m.MacroblockTyp[bottomPairXY] = MBTypeInterlaced | MBType16x16 | MBTypeP0L0 | MBTypeSkip

	field := m.predictFrameMBAFFFieldDecodingFlag(mbXY, sliceNum)
	if field != 1 {
		t.Fatalf("row-start field prediction = %d, want 1", field)
	}

	src := &scriptedCABACSource{bits: []int{0}}
	skip, err := m.decodeCABACMBSkipMBAFF(src, mbXY, 0, 2, PictureTypeP, sliceNum, field)
	if err != nil {
		t.Fatalf("decode predicted-row skip failed: %v", err)
	}
	if skip {
		t.Fatal("skip = true, want false from scripted bit")
	}
	wantIndexes(t, src, []int{12})
}

func TestWriteBackCABACFrameMBAFFBSpatialSkipMapsFieldNeighborMotion(t *testing.T) {
	m, err := newMacroblockTables(2, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	colTables, err := newMacroblockTables(2, 2, 1)
	if err != nil {
		t.Fatal(err)
	}

	const sliceNum = uint16(13)
	mbXY := 1
	leftXY := 0
	leftType := MBTypeInterlaced | MBType16x16 | MBTypeP0L0 | MBTypeP1L0
	m.SliceTable[leftXY] = sliceNum
	m.MacroblockTyp[leftXY] = leftType
	m.RefIndex[0][4*leftXY+1] = 2
	m.MotionVal[0][int(m.MB2BXY[leftXY])+3] = [2]int16{4, 5}

	colTables.MacroblockTyp[mbXY] = MBType16x16 | MBTypeP0L0 | MBTypeP1L0
	past0 := &DecodedFrame{poc: -2}
	past1 := &DecodedFrame{poc: 0}
	col := &DecodedFrame{
		poc:    4,
		tables: colTables,
		refEntries: [2][]simpleRefEntry{
			{{frame: past0}, {frame: past1}},
		},
	}
	sps := &SPS{BitDepthLuma: 8, ChromaFormatIDC: 1, FrameMBSOnlyFlag: 0, MBAFF: 1, Direct8x8InferenceFlag: 1}
	pps := cavlcFlatQMulPPS()
	pps.SPS = sps
	sh := &SliceHeader{
		SliceTypeNoS:     PictureTypeB,
		PictureStructure: PictureFrame,
		PPS:              pps,
		SPS:              sps,
		QScale:           22,
	}
	var work frameMacroblockDecodeWork

	got, err := m.writeBackCABACFrameBSkipMacroblockWithDirectWorkFieldGuard(sh, 22, mbXY, sliceNum, 0, h264DirectMotionContext{
		RefEntries: [2][]simpleRefEntry{
			{{frame: past0}, {frame: past1}},
			{{frame: col}},
		},
		CurPOC:              2,
		PictureStructure:    PictureFrame,
		DirectSpatialMVPred: true,
		Direct8x8Inference:  true,
		X264Build:           165,
	}, &work, false)
	if err != nil {
		t.Fatalf("write back CABAC MBAFF spatial B-skip failed: %v", err)
	}
	if !got.Skipped || !got.IsInter || got.MBFieldDecodingFlag != 0 || got.MBType&MBTypeInterlaced != 0 || !isDirect(got.MBType) || !is16x16(got.MBType) {
		t.Fatalf("result skipped/inter/field/type = %v/%v/%d/%#x, want frame-coded direct 16x16 skip", got.Skipped, got.IsInter, got.MBFieldDecodingFlag, got.MBType)
	}
	if got.Neighbors.LeftXY[h264LeftTop] != leftXY || got.Neighbors.LeftType[h264LeftTop] != leftType {
		t.Fatalf("left neighbor = xy %d type %#x, want %d/%#x", got.Neighbors.LeftXY[h264LeftTop], got.Neighbors.LeftType[h264LeftTop], leftXY, leftType)
	}
	bXY := int(m.MB2BXY[mbXY])
	if m.RefIndex[0][4*mbXY] != 1 || m.MotionVal[0][bXY] != ([2]int16{4, 10}) {
		t.Fatalf("mapped list0 ref/mv = %d/%v, want 1/[4 10]", m.RefIndex[0][4*mbXY], m.MotionVal[0][bXY])
	}
	if work.Motion.Ref[0][int(h264Scan8[0])-1] != 1 || work.Motion.MV[0][int(h264Scan8[0])-1] != ([2]int16{4, 10}) {
		t.Fatalf("mapped neighbor cache ref/mv = %d/%v, want 1/[4 10]", work.Motion.Ref[0][int(h264Scan8[0])-1], work.Motion.MV[0][int(h264Scan8[0])-1])
	}
}

func TestDecodeCABACFrameBDirectUnsupportedBeforeWriteback(t *testing.T) {
	m, err := newMacroblockTables(1, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	sps := &SPS{BitDepthLuma: 8, ChromaFormatIDC: 1, FrameMBSOnlyFlag: 1}
	pps := cavlcFlatQMulPPS()
	pps.SPS = sps
	src := &scriptedCABACSource{bits: []int{0}}

	_, err = m.decodeCABACFrameMacroblock(src, cabacFrameMacroblockInput{
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
	wantIndexes(t, src, []int{27})
}

func TestDecodeCABACFieldDecodingFlagContexts(t *testing.T) {
	m, err := newMacroblockTables(3, 4, 1)
	if err != nil {
		t.Fatal(err)
	}
	const sliceNum = uint16(9)
	mbXY := 1 + 2*m.MBStride
	topPairXY := mbXY - 2*m.MBStride
	m.SliceTable[topPairXY] = sliceNum
	m.MacroblockTyp[topPairXY] = MBTypeIntra4x4 | MBTypeInterlaced

	tests := []struct {
		name        string
		mbX         int
		prevField   bool
		topSame     bool
		topType     uint32
		wantContext int
		wantFlag    int32
	}{
		{name: "none", mbX: 0, topSame: false, wantContext: 70, wantFlag: 0},
		{name: "left", mbX: 1, prevField: true, topSame: false, wantContext: 71, wantFlag: 1},
		{name: "top", mbX: 0, topSame: true, topType: MBTypeIntra4x4 | MBTypeInterlaced, wantContext: 71, wantFlag: 1},
		{name: "left-top", mbX: 1, prevField: true, topSame: true, topType: MBTypeIntra4x4 | MBTypeInterlaced, wantContext: 72, wantFlag: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m.SliceTable[topPairXY] = ^uint16(0)
			m.MacroblockTyp[topPairXY] = 0
			if tt.topSame {
				m.SliceTable[topPairXY] = sliceNum
				m.MacroblockTyp[topPairXY] = tt.topType
			}
			src := &scriptedCABACSource{bits: []int{int(tt.wantFlag)}}

			got, err := m.decodeCABACFieldDecodingFlag(src, mbXY, tt.mbX, sliceNum, tt.prevField)
			if err != nil {
				t.Fatalf("decode field flag failed: %v", err)
			}
			if got != tt.wantFlag {
				t.Fatalf("field flag = %d, want %d", got, tt.wantFlag)
			}
			wantIndexes(t, src, []int{tt.wantContext})
		})
	}
}

func repeatCABACBits(count int, bit int) []int {
	out := make([]int, count)
	for i := range out {
		out[i] = bit
	}
	return out
}
