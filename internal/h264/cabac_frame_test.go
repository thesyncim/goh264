// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

import (
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
	if err != ErrUnsupported {
		t.Fatalf("err = %v, want ErrUnsupported", err)
	}
	if m.SliceTable[0] != ^uint16(0) || m.MacroblockTyp[0] != 0 {
		t.Fatalf("state changed on unsupported direct: slice/type = %d/%#x", m.SliceTable[0], m.MacroblockTyp[0])
	}
	wantIndexes(t, src, []int{27})
}

func repeatCABACBits(count int, bit int) []int {
	out := make([]int, count)
	for i := range out {
		out[i] = bit
	}
	return out
}
