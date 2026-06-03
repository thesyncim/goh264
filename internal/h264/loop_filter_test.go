// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

import "testing"

func TestH264ApplyLoopFilterEdge444UsesLumaChromaPlanes(t *testing.T) {
	const stride = 32
	dst := &h264PicturePlanes{
		Y:               make([]uint8, stride*16),
		Cb:              make([]uint8, stride*16),
		Cr:              make([]uint8, stride*16),
		LumaStride:      stride,
		ChromaStride:    stride,
		MBWidth:         1,
		MBHeight:        1,
		ChromaFormatIDC: 3,
	}
	fill444LoopFilterStep(dst.Cb, stride, 4, 100, 110)
	fill444LoopFilterStep(dst.Cr, stride, 4, 80, 92)
	cbBefore := dst.Cb[3]
	crBefore := dst.Cr[3]

	if err := h264ApplyLoopFilterEdge(dst, 0, 0, 0, 0, 1, [4]int16{3, 3, 3, 3}, 30, [2]int{30, 30}, h264LoopFilterSliceParams{}, false, false, true); err != nil {
		t.Fatal(err)
	}
	if dst.Cb[3] == cbBefore || dst.Cr[3] == crBefore {
		t.Fatalf("4:4:4 chroma planes were not filtered: cb %d->%d cr %d->%d", cbBefore, dst.Cb[3], crBefore, dst.Cr[3])
	}
}

func TestFillLoopFilterCachesFrameCanonicalizesBListRefs(t *testing.T) {
	m, err := newMacroblockTables(1, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	m.MacroblockTyp[0] = MBType8x8 | MBTypeP0L0 | MBTypeP0L1 | MBTypeP1L0 | MBTypeP1L1
	m.SliceTable[0] = 0
	for list := 0; list < 2; list++ {
		for i := 0; i < 4; i++ {
			m.RefIndex[list][i] = 0
		}
	}
	pps := cavlcFlatQMulPPS()
	pps.SPS = &SPS{
		BitDepthLuma:     8,
		BitDepthChroma:   8,
		ChromaFormatIDC:  1,
		FrameMBSOnlyFlag: 1,
	}
	params := []h264LoopFilterSliceParams{{
		PPS:              pps,
		ListCount:        2,
		DeblockingFilter: 1,
		Ref2Frame: [2][]int8{
			{3},
			{5},
		},
	}}

	ctx, err := m.fillLoopFilterCachesFrame(0, 0, params[0], params)
	if err != nil {
		t.Fatal(err)
	}
	base := int(h264Scan8[0])
	if ctx.Motion.Ref[0][base] != 3 || ctx.Motion.Ref[1][base] != 5 {
		t.Fatalf("loop-filter refs = %d/%d, want canonical 3/5", ctx.Motion.Ref[0][base], ctx.Motion.Ref[1][base])
	}
}

func TestH264LoopFilterRef2FramePreservesReferenceStructure(t *testing.T) {
	ref := &DecodedFrame{}
	other := &DecodedFrame{}
	refs, err := h264LoopFilterRef2Frame([2][]simpleRefEntry{
		{
			{frame: ref, pictureStructure: PictureTopField},
			{frame: ref, pictureStructure: PictureBottomField},
			{frame: other, pictureStructure: PictureFrame},
		},
	}, map[*DecodedFrame]int8{})
	if err != nil {
		t.Fatal(err)
	}
	want := []int8{1, 2, 7}
	if len(refs[0]) != len(want) {
		t.Fatalf("ref2frame len = %d, want %d", len(refs[0]), len(want))
	}
	for i := range want {
		if refs[0][i] != want[i] {
			t.Fatalf("ref2frame[%d] = %d, want %d", i, refs[0][i], want[i])
		}
	}
}

func TestFillLoopFilterCachesFrameMBAFFFieldRefsUseExpandedRef2Frame(t *testing.T) {
	m, err := newMacroblockTables(2, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	leftXY := 0
	mbXY := 1
	mbType := MBType16x16 | MBTypeP0L0 | MBTypeInterlaced
	for _, xy := range []int{leftXY, mbXY} {
		m.MacroblockTyp[xy] = mbType
		m.QScaleTable[xy] = 24
		m.SliceTable[xy] = 0
	}
	m.RefIndex[0][4*mbXY+0] = 0
	m.RefIndex[0][4*mbXY+1] = 0
	m.RefIndex[0][4*mbXY+2] = 0
	m.RefIndex[0][4*mbXY+3] = 0
	m.RefIndex[0][4*leftXY+1] = 3
	m.RefIndex[0][4*leftXY+3] = 3

	pps := cavlcFlatQMulPPS()
	pps.SPS = &SPS{
		BitDepthLuma:     8,
		BitDepthChroma:   8,
		ChromaFormatIDC:  1,
		FrameMBSOnlyFlag: 0,
		MBAFF:            1,
	}
	params := []h264LoopFilterSliceParams{{
		PPS:              pps,
		ListCount:        1,
		PictureStructure: PictureFrame,
		DeblockingFilter: 1,
		Ref2Frame: [2][]int8{
			{3, 7},
		},
	}}

	ctx, err := m.fillLoopFilterCachesFrame(mbXY, 0, params[0], params)
	if err != nil {
		t.Fatal(err)
	}
	base := int(h264Scan8[0])
	if got := ctx.Motion.Ref[0][base]; got != 1 {
		t.Fatalf("current expanded ref = %d, want top field id 1", got)
	}
	if got := ctx.Motion.Ref[0][base-1+2*8]; got != 6 {
		t.Fatalf("left expanded ref = %d, want second frame bottom field id 6", got)
	}
	maskPar0 := mbType & (MBType16x16 | MBType8x16)
	bS, err := m.loopFilterBoundaryStrength(&ctx, mbType, mbType, 0, maskPar0, params[0].ListCount, h264LoopFilterMVYLimit(mbType), true)
	if err != nil {
		t.Fatal(err)
	}
	if bS != [4]int16{1, 1, 1, 1} {
		t.Fatalf("MBAFF field-ref boundary bS = %v, want all 1", bS)
	}
}

func TestFillLoopFilterCachesInterFrameMBAFFMixedLeftLeavesMotionCache(t *testing.T) {
	m, err := newMacroblockTables(2, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	leftXY := 0
	mbXY := 1
	leftType := MBType16x16 | MBTypeP0L0
	mbType := MBType16x16 | MBTypeP0L0 | MBTypeInterlaced
	m.MacroblockTyp[leftXY] = leftType
	m.MacroblockTyp[mbXY] = mbType
	for _, xy := range []int{leftXY, mbXY} {
		m.QScaleTable[xy] = 24
		m.SliceTable[xy] = 0
	}
	leftBXY := int(m.MB2BXY[leftXY]) + 3
	for row := 0; row < 4; row++ {
		m.MotionVal[0][leftBXY+row*m.BStride] = [2]int16{int16(50 + row), int16(70 + row)}
	}
	m.RefIndex[0][4*leftXY+1] = 1
	m.RefIndex[0][4*leftXY+3] = 1

	pps := cavlcFlatQMulPPS()
	pps.SPS = &SPS{
		BitDepthLuma:     8,
		BitDepthChroma:   8,
		ChromaFormatIDC:  1,
		FrameMBSOnlyFlag: 0,
		MBAFF:            1,
	}
	p := h264LoopFilterSliceParams{
		PPS:              pps,
		ListCount:        1,
		PictureStructure: PictureFrame,
		DeblockingFilter: 1,
		Ref2Frame: [2][]int8{
			{3, 7},
		},
	}
	base := int(h264Scan8[0])
	ctx := &h264LoopFilterContext{}
	for row := 0; row < 4; row++ {
		idx := base - 1 + row*8
		ctx.Motion.MV[0][idx] = [2]int16{123, 456}
		ctx.Motion.Ref[0][idx] = 77
	}

	if err := m.fillLoopFilterCachesInterFrame(ctx, mbXY, -1, leftXY, mbType, 0, leftType, 0, p, []h264LoopFilterSliceParams{p}); err != nil {
		t.Fatal(err)
	}
	for row := 0; row < 4; row++ {
		idx := base - 1 + row*8
		if ctx.Motion.MV[0][idx] != ([2]int16{123, 456}) || ctx.Motion.Ref[0][idx] != 77 {
			t.Fatalf("mixed left cache row %d = mv %v ref %d, want sentinel untouched", row, ctx.Motion.MV[0][idx], ctx.Motion.Ref[0][idx])
		}
	}

	mbType &^= MBTypeInterlaced
	if err := m.fillLoopFilterCachesInterFrame(ctx, mbXY, -1, leftXY, mbType, 0, leftType, 0, p, []h264LoopFilterSliceParams{p}); err != nil {
		t.Fatal(err)
	}
	for row := 0; row < 4; row++ {
		idx := base - 1 + row*8
		wantMV := [2]int16{int16(50 + row), int16(70 + row)}
		if ctx.Motion.MV[0][idx] != wantMV || ctx.Motion.Ref[0][idx] != 7 {
			t.Fatalf("same left cache row %d = mv %v ref %d, want %v/7", row, ctx.Motion.MV[0][idx], ctx.Motion.Ref[0][idx], wantMV)
		}
	}
}

func TestFillLoopFilterCachesFieldPictureKeepsSameFrameFieldRefsDistinct(t *testing.T) {
	m, err := newMacroblockTables(1, 3, 1)
	if err != nil {
		t.Fatal(err)
	}
	topXY := 0
	curXY := 2 * m.MBStride
	mbType := MBType16x16 | MBTypeP0L0 | MBTypeInterlaced
	for _, mbXY := range []int{topXY, curXY} {
		m.MacroblockTyp[mbXY] = mbType
		m.QScaleTable[mbXY] = 24
		m.SliceTable[mbXY] = 0
	}
	m.RefIndex[0][4*topXY+2] = 1
	m.RefIndex[0][4*topXY+3] = 1

	pps := cavlcFlatQMulPPS()
	pps.SPS = &SPS{
		BitDepthLuma:     8,
		BitDepthChroma:   8,
		ChromaFormatIDC:  1,
		FrameMBSOnlyFlag: 0,
	}
	ref := &DecodedFrame{}
	ref2Frame, err := h264LoopFilterRef2Frame([2][]simpleRefEntry{
		{
			{frame: ref, pictureStructure: PictureTopField},
			{frame: ref, pictureStructure: PictureBottomField},
		},
	}, map[*DecodedFrame]int8{})
	if err != nil {
		t.Fatal(err)
	}
	p := h264LoopFilterSliceParams{
		PPS:              pps,
		ListCount:        1,
		PictureStructure: PictureTopField,
		DeblockingFilter: 1,
		Ref2Frame:        ref2Frame,
	}
	params := []h264LoopFilterSliceParams{p}

	ctx, err := m.fillLoopFilterCachesFrame(curXY, 0, p, params)
	if err != nil {
		t.Fatal(err)
	}
	maskPar0 := mbType & (MBType16x16 | (MBType8x16 >> 1))
	bS, err := m.loopFilterBoundaryStrength(&ctx, mbType, mbType, 1, maskPar0, p.ListCount, h264LoopFilterMVYLimit(mbType), false)
	if err != nil {
		t.Fatal(err)
	}
	if bS != [4]int16{1, 1, 1, 1} {
		t.Fatalf("field ref boundary bS = %v, want all 1 for top/bottom refs from same frame", bS)
	}
}

func TestH264LoopFilterThresholdsHighBitDepthQPBDOffset(t *testing.T) {
	alpha8, beta8, index8, err := h264LoopFilterThresholdsForBitDepth(30, 0, 0, 8)
	if err != nil {
		t.Fatal(err)
	}
	alpha10, beta10, index10, err := h264LoopFilterThresholdsForBitDepth(42, 0, 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	if alpha10 != alpha8 || beta10 != beta8 || index10 != index8 {
		t.Fatalf("High10 qp_bd_offset mapping = alpha %d beta %d index %d, want %d/%d/%d", alpha10, beta10, index10, alpha8, beta8, index8)
	}
	if alpha, beta, index, err := h264LoopFilterThresholdsForBitDepth(11, 0, 0, 10); err != nil || alpha != 0 || beta != 0 || index != 0 {
		t.Fatalf("High10 low qp threshold = alpha %d beta %d index %d err %v, want 0/0/0/nil", alpha, beta, index, err)
	}
	if alpha, beta, index, err := h264LoopFilterThresholdsForBitDepth(63, 0, 0, 10); err != nil || alpha != 255 || beta != 18 || index != 51 {
		t.Fatalf("High10 high qp threshold = alpha %d beta %d index %d err %v, want 255/18/51/nil", alpha, beta, index, err)
	}
}

func TestH264LoopFilterValidateAllows8BitFrameMBAFFDeblock(t *testing.T) {
	pps := cavlcFlatQMulPPS()
	pps.SPS = &SPS{
		BitDepthLuma:     8,
		BitDepthChroma:   8,
		ChromaFormatIDC:  1,
		FrameMBSOnlyFlag: 0,
		MBAFF:            1,
	}
	p := h264LoopFilterSliceParams{
		PPS:              pps,
		ListCount:        1,
		PictureStructure: PictureFrame,
		DeblockingFilter: 1,
	}
	if err := p.validate(); err != nil {
		t.Fatalf("8-bit frame-MBAFF deblock validation err = %v, want nil", err)
	}
}

func TestH264LoopFilterValidateRejectsHighBitDepthMBAFFDeblock(t *testing.T) {
	pps := cavlcFlatQMulPPS()
	pps.SPS = &SPS{
		BitDepthLuma:     10,
		BitDepthChroma:   10,
		ChromaFormatIDC:  1,
		FrameMBSOnlyFlag: 0,
		MBAFF:            1,
	}
	p := h264LoopFilterSliceParams{
		PPS:              pps,
		ListCount:        1,
		PictureStructure: PictureFrame,
		DeblockingFilter: 1,
	}
	if err := p.validate(); err != ErrUnsupported {
		t.Fatalf("High10 frame-MBAFF deblock validation err = %v, want ErrUnsupported", err)
	}
}

func TestH264FrameMBAFFLoopFilterViewMapsFieldCodedRows(t *testing.T) {
	const (
		mbWidth      = 1
		mbHeight     = 2
		lumaStride   = 16
		chromaStride = 8
	)
	dst := &h264PicturePlanes{
		Y:                make([]uint8, lumaStride*32),
		Cb:               make([]uint8, chromaStride*16),
		Cr:               make([]uint8, chromaStride*16),
		LumaStride:       lumaStride,
		ChromaStride:     chromaStride,
		MBWidth:          mbWidth,
		MBHeight:         mbHeight,
		ChromaFormatIDC:  1,
		PictureStructure: PictureFrame,
	}

	top, mbY, err := h264FrameMBAFFLoopFilterView(dst, 0, MBTypeInterlaced|MBTypeIntra4x4)
	if err != nil {
		t.Fatal(err)
	}
	if top.PictureStructure != PictureTopField || top.LumaStride != lumaStride*2 || top.ChromaStride != chromaStride*2 || top.MBHeight != 1 || mbY != 0 {
		t.Fatalf("top MBAFF filter view = picture %d strides %d/%d height %d mbY %d, want top %d/%d/1/0",
			top.PictureStructure, top.LumaStride, top.ChromaStride, top.MBHeight, mbY, lumaStride*2, chromaStride*2)
	}
	if len(top.Y) != len(dst.Y) || len(top.Cb) != len(dst.Cb) || len(top.Cr) != len(dst.Cr) {
		t.Fatalf("top MBAFF filter view shifted planes")
	}

	bottom, mbY, err := h264FrameMBAFFLoopFilterView(dst, 1, MBTypeInterlaced|MBTypeIntra4x4)
	if err != nil {
		t.Fatal(err)
	}
	if bottom.PictureStructure != PictureBottomField || bottom.LumaStride != lumaStride*2 || bottom.ChromaStride != chromaStride*2 || bottom.MBHeight != 1 || mbY != 0 {
		t.Fatalf("bottom MBAFF filter view = picture %d strides %d/%d height %d mbY %d, want bottom %d/%d/1/0",
			bottom.PictureStructure, bottom.LumaStride, bottom.ChromaStride, bottom.MBHeight, mbY, lumaStride*2, chromaStride*2)
	}
	if &bottom.Y[0] != &dst.Y[lumaStride] || &bottom.Cb[0] != &dst.Cb[chromaStride] || &bottom.Cr[0] != &dst.Cr[chromaStride] {
		t.Fatalf("bottom MBAFF filter view did not shift to bottom-field rows")
	}

	frame, mbY, err := h264FrameMBAFFLoopFilterView(dst, 1, MBTypeIntra4x4)
	if err != nil {
		t.Fatal(err)
	}
	if frame.PictureStructure != PictureFrame || frame.LumaStride != lumaStride || frame.ChromaStride != chromaStride || frame.MBHeight != mbHeight || mbY != 1 {
		t.Fatalf("frame-coded MBAFF filter view = picture %d strides %d/%d height %d mbY %d, want frame %d/%d/%d/1",
			frame.PictureStructure, frame.LumaStride, frame.ChromaStride, frame.MBHeight, mbY, lumaStride, chromaStride, mbHeight)
	}
}

func TestFillLoopFilterCachesFrameMBAFFUsesFieldTopNeighbor(t *testing.T) {
	const (
		mbWidth  = 1
		mbHeight = 4
	)
	m, err := newMacroblockTables(mbWidth, mbHeight, 1)
	if err != nil {
		t.Fatal(err)
	}
	for mbY := 0; mbY < mbHeight; mbY++ {
		mbXY := mbY * m.MBStride
		m.MacroblockTyp[mbXY] = MBTypeIntra4x4 | MBTypeInterlaced
		m.QScaleTable[mbXY] = 30
		m.SliceTable[mbXY] = 0
	}
	pps := cavlcFlatQMulPPS()
	pps.SPS = &SPS{
		BitDepthLuma:     8,
		BitDepthChroma:   8,
		ChromaFormatIDC:  1,
		FrameMBSOnlyFlag: 0,
		MBAFF:            1,
	}
	params := []h264LoopFilterSliceParams{{
		PPS:              pps,
		ListCount:        1,
		PictureStructure: PictureFrame,
		DeblockingFilter: 1,
	}}

	ctx, err := m.fillLoopFilterCachesFrame(2*m.MBStride, 0, params[0], params)
	if err != nil {
		t.Fatal(err)
	}
	if ctx.TopMBXY != 0 {
		t.Fatalf("field-coded MBAFF top neighbor = %d, want 0 two rows above", ctx.TopMBXY)
	}
}

func TestFillLoopFilterCachesFrameMBAFFSplitsMixedLeftNeighbors(t *testing.T) {
	const (
		mbWidth  = 2
		mbHeight = 4
	)
	m, err := newMacroblockTables(mbWidth, mbHeight, 1)
	if err != nil {
		t.Fatal(err)
	}
	for mbY := 0; mbY < mbHeight; mbY++ {
		for mbX := 0; mbX < mbWidth; mbX++ {
			mbXY := mbX + mbY*m.MBStride
			m.MacroblockTyp[mbXY] = MBTypeIntra4x4
			m.QScaleTable[mbXY] = 30
			m.SliceTable[mbXY] = 0
		}
	}
	leftTop := 0
	leftBottom := m.MBStride
	current := 1
	m.MacroblockTyp[leftTop] = MBTypeIntra4x4 | MBTypeInterlaced
	m.MacroblockTyp[leftBottom] = MBTypeIntra16x16 | MBTypeInterlaced
	pps := cavlcFlatQMulPPS()
	pps.SPS = &SPS{
		BitDepthLuma:     8,
		BitDepthChroma:   8,
		ChromaFormatIDC:  1,
		FrameMBSOnlyFlag: 0,
		MBAFF:            1,
	}
	params := []h264LoopFilterSliceParams{{
		PPS:              pps,
		ListCount:        1,
		PictureStructure: PictureFrame,
		DeblockingFilter: 1,
	}}

	ctx, err := m.fillLoopFilterCachesFrame(current, 0, params[0], params)
	if err != nil {
		t.Fatal(err)
	}
	if ctx.LeftMBXYs != ([2]int{leftTop, leftBottom}) {
		t.Fatalf("frame-coded MBAFF mixed left xy = %v, want top/bottom %d/%d", ctx.LeftMBXYs, leftTop, leftBottom)
	}
	if ctx.LeftTypes != ([2]uint32{m.MacroblockTyp[leftTop], m.MacroblockTyp[leftBottom]}) {
		t.Fatalf("frame-coded MBAFF mixed left types = %#x/%#x", ctx.LeftTypes[0], ctx.LeftTypes[1])
	}

	fieldCurrent := current + m.MBStride
	m.MacroblockTyp[fieldCurrent] = MBTypeIntra4x4 | MBTypeInterlaced
	m.MacroblockTyp[leftTop] = MBTypeIntra4x4
	m.MacroblockTyp[leftBottom] = MBTypeIntra16x16
	ctx, err = m.fillLoopFilterCachesFrame(fieldCurrent, 0, params[0], params)
	if err != nil {
		t.Fatal(err)
	}
	if ctx.LeftMBXYs != ([2]int{leftTop, leftBottom}) {
		t.Fatalf("field-coded MBAFF mixed left xy = %v, want top/bottom %d/%d", ctx.LeftMBXYs, leftTop, leftBottom)
	}
	if ctx.LeftTypes != ([2]uint32{m.MacroblockTyp[leftTop], m.MacroblockTyp[leftBottom]}) {
		t.Fatalf("field-coded MBAFF mixed left types = %#x/%#x", ctx.LeftTypes[0], ctx.LeftTypes[1])
	}
}

func TestLoopFilterMBAFFMixedVerticalStrengthUsesBothLeftRows(t *testing.T) {
	m, err := newMacroblockTables(2, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	leftTop := 0
	leftBottom := m.MBStride
	current := 1
	m.MacroblockTyp[leftTop] = MBType16x16 | MBTypeP0L0 | MBTypeInterlaced
	m.MacroblockTyp[leftBottom] = MBType16x16 | MBTypeP0L0 | MBTypeInterlaced
	m.MacroblockTyp[current] = MBType16x16 | MBTypeP0L0
	m.NonZeroCount[leftTop][3] = 1
	m.NonZeroCount[leftBottom][7] = 1
	ctx := &h264LoopFilterContext{
		MBXY:      current,
		LeftMBXYs: [2]int{leftTop, leftBottom},
		LeftTypes: [2]uint32{
			m.MacroblockTyp[leftTop],
			m.MacroblockTyp[leftBottom],
		},
	}
	pps := cavlcFlatQMulPPS()

	bS, err := m.loopFilterMBAFFMixedVerticalStrength(ctx, m.MacroblockTyp[current], h264LoopFilterSliceParams{PPS: pps, CABAC: true})
	if err != nil {
		t.Fatal(err)
	}
	want := [8]int16{2, 1, 2, 1, 1, 2, 1, 2}
	if bS != want {
		t.Fatalf("frame-coded mixed MBAFF bS = %v, want %v", bS, want)
	}

	m.MacroblockTyp[current] |= MBTypeInterlaced
	ctx.NonZeroCountCache = [h264NonZeroCountCacheSize]uint8{}
	ctx.NonZeroCountCache[12+8*1] = 1
	bS, err = m.loopFilterMBAFFMixedVerticalStrength(ctx, m.MacroblockTyp[current], h264LoopFilterSliceParams{PPS: pps, CABAC: true})
	if err != nil {
		t.Fatal(err)
	}
	want = [8]int16{2, 1, 2, 2, 1, 2, 1, 1}
	if bS != want {
		t.Fatalf("field-coded mixed MBAFF bS = %v, want %v", bS, want)
	}
}

func TestMacroblockTablesFilterFrameMBAFFFieldViewFiltersFieldRows(t *testing.T) {
	const (
		mbWidth      = 1
		mbHeight     = 2
		lumaStride   = 16
		chromaStride = 8
		qp           = 30
	)
	m, err := newMacroblockTables(mbWidth, mbHeight, 1)
	if err != nil {
		t.Fatal(err)
	}
	dst := &h264PicturePlanes{
		Y:               make([]uint8, lumaStride*32),
		Cb:              make([]uint8, chromaStride*16),
		Cr:              make([]uint8, chromaStride*16),
		LumaStride:      lumaStride,
		ChromaStride:    chromaStride,
		MBWidth:         mbWidth,
		MBHeight:        mbHeight,
		ChromaFormatIDC: 1,
	}
	fillLoopFilterHorizontalStep(dst.Y, lumaStride, 16, 32, 8, 104, 112)
	fillLoopFilterHorizontalStep(dst.Cb, chromaStride, 8, 16, 4, 84, 92)
	fillLoopFilterHorizontalStep(dst.Cr, chromaStride, 8, 16, 4, 64, 72)
	for mbY := 0; mbY < mbHeight; mbY++ {
		mbXY := mbY * m.MBStride
		m.MacroblockTyp[mbXY] = MBTypeIntra16x16 | MBTypeInterlaced
		m.QScaleTable[mbXY] = qp
		m.SliceTable[mbXY] = 0
	}
	pps := cavlcFlatQMulPPS()
	pps.SPS = &SPS{
		BitDepthLuma:     8,
		BitDepthChroma:   8,
		ChromaFormatIDC:  1,
		FrameMBSOnlyFlag: 0,
		MBAFF:            1,
	}
	params := []h264LoopFilterSliceParams{{
		PPS:              pps,
		ListCount:        1,
		PictureStructure: PictureFrame,
		DeblockingFilter: 1,
	}}
	adjacentFrameRowsBefore := [2]uint8{dst.Y[7*lumaStride], dst.Y[8*lumaStride]}
	topFieldRowsBefore := [2]uint8{dst.Y[6*lumaStride], dst.Y[8*lumaStride]}
	bottomFieldRowsBefore := [2]uint8{dst.Y[7*lumaStride], dst.Y[9*lumaStride]}

	if err := m.filterFrame(dst, params); err != nil {
		t.Fatal(err)
	}
	if dst.Y[6*lumaStride] == topFieldRowsBefore[0] || dst.Y[8*lumaStride] == topFieldRowsBefore[1] {
		t.Fatalf("top-field MBAFF horizontal edge did not filter: %v -> [%d %d]",
			topFieldRowsBefore, dst.Y[6*lumaStride], dst.Y[8*lumaStride])
	}
	if dst.Y[7*lumaStride] == bottomFieldRowsBefore[0] || dst.Y[9*lumaStride] == bottomFieldRowsBefore[1] {
		t.Fatalf("bottom-field MBAFF horizontal edge did not filter: %v -> [%d %d]",
			bottomFieldRowsBefore, dst.Y[7*lumaStride], dst.Y[9*lumaStride])
	}
	if dst.Y[7*lumaStride] == adjacentFrameRowsBefore[0] || dst.Y[8*lumaStride] == adjacentFrameRowsBefore[1] {
		t.Fatalf("MBAFF filter unexpectedly left adjacent frame rows untouched: %v -> [%d %d]",
			adjacentFrameRowsBefore, dst.Y[7*lumaStride], dst.Y[8*lumaStride])
	}
}

func TestH264ApplyLoopFilterEdgeHigh420MutatesLumaChroma(t *testing.T) {
	const (
		lumaStride   = 32
		chromaStride = 16
		bitDepth     = 10
	)
	dst := &h264PicturePlanesHigh{
		Y:               make([]uint16, lumaStride*16),
		Cb:              make([]uint16, chromaStride*8),
		Cr:              make([]uint16, chromaStride*8),
		LumaStride:      lumaStride,
		ChromaStride:    chromaStride,
		MBWidth:         2,
		MBHeight:        1,
		ChromaFormatIDC: 1,
	}
	fillHighLoopFilterStep(dst.Y, lumaStride, 16, 16, 8, 400, 408)
	fillHighLoopFilterStep(dst.Cb, chromaStride, 16, 8, 4, 300, 308)
	fillHighLoopFilterStep(dst.Cr, chromaStride, 16, 8, 4, 200, 208)
	yBefore := [2]uint16{dst.Y[7], dst.Y[8]}
	cbBefore := [2]uint16{dst.Cb[3], dst.Cb[4]}
	crBefore := [2]uint16{dst.Cr[3], dst.Cr[4]}

	if err := h264ApplyLoopFilterEdgeHigh(dst, 0, 0, 0, 0, 2, [4]int16{3, 3, 3, 3}, 30, [2]int{30, 30}, h264LoopFilterSliceParams{}, false, true, true, bitDepth); err != nil {
		t.Fatal(err)
	}
	if dst.Y[7] == yBefore[0] || dst.Y[8] == yBefore[1] {
		t.Fatalf("High10 luma edge did not filter: %v -> [%d %d]", yBefore, dst.Y[7], dst.Y[8])
	}
	if dst.Cb[3] == cbBefore[0] || dst.Cb[4] == cbBefore[1] || dst.Cr[3] == crBefore[0] || dst.Cr[4] == crBefore[1] {
		t.Fatalf("High10 chroma edge did not filter: cb %v -> [%d %d] cr %v -> [%d %d]",
			cbBefore, dst.Cb[3], dst.Cb[4], crBefore, dst.Cr[3], dst.Cr[4])
	}
}

func TestH264ApplyLoopFilterEdgeHigh422MutatesLumaChroma(t *testing.T) {
	const (
		lumaStride   = 32
		chromaStride = 16
	)
	for _, bitDepth := range []int{10, 12} {
		t.Run(bitDepthName(int32(bitDepth)), func(t *testing.T) {
			qp := 30 + 6*(bitDepth-8)
			dst := &h264PicturePlanesHigh{
				Y:               make([]uint16, lumaStride*16),
				Cb:              make([]uint16, chromaStride*16),
				Cr:              make([]uint16, chromaStride*16),
				LumaStride:      lumaStride,
				ChromaStride:    chromaStride,
				MBWidth:         1,
				MBHeight:        1,
				ChromaFormatIDC: 2,
			}
			fillHighLoopFilterStep(dst.Y, lumaStride, 16, 16, 8, 400, 408)
			fillHighLoopFilterStep(dst.Cb, chromaStride, 16, 16, 4, 300, 308)
			fillHighLoopFilterStep(dst.Cr, chromaStride, 16, 16, 4, 200, 208)
			yBefore := [2]uint16{dst.Y[7], dst.Y[8]}
			cbBefore := [2]uint16{dst.Cb[3], dst.Cb[4]}
			crBefore := [2]uint16{dst.Cr[3], dst.Cr[4]}

			if err := h264ApplyLoopFilterEdgeHigh(dst, 0, 0, 0, 0, 2, [4]int16{3, 3, 3, 3}, qp, [2]int{qp, qp}, h264LoopFilterSliceParams{}, false, true, true, bitDepth); err != nil {
				t.Fatal(err)
			}
			if dst.Y[7] == yBefore[0] || dst.Y[8] == yBefore[1] {
				t.Fatalf("high 4:2:2 luma edge did not filter: %v -> [%d %d]", yBefore, dst.Y[7], dst.Y[8])
			}
			if dst.Cb[3] == cbBefore[0] || dst.Cb[4] == cbBefore[1] || dst.Cr[3] == crBefore[0] || dst.Cr[4] == crBefore[1] {
				t.Fatalf("high 4:2:2 chroma edge did not filter: cb %v -> [%d %d] cr %v -> [%d %d]",
					cbBefore, dst.Cb[3], dst.Cb[4], crBefore, dst.Cr[3], dst.Cr[4])
			}
		})
	}
}

func TestH264ApplyLoopFilterEdgeHigh444UsesLumaKernelsForChroma(t *testing.T) {
	const stride = 32
	for _, bitDepth := range []int{10, 12} {
		t.Run(bitDepthName(int32(bitDepth)), func(t *testing.T) {
			qp := 30 + 6*(bitDepth-8)
			dst := &h264PicturePlanesHigh{
				Y:               make([]uint16, stride*16),
				Cb:              make([]uint16, stride*16),
				Cr:              make([]uint16, stride*16),
				LumaStride:      stride,
				ChromaStride:    stride,
				MBWidth:         1,
				MBHeight:        1,
				ChromaFormatIDC: 3,
			}
			fillHighLoopFilterStep(dst.Cb, stride, 16, 16, 4, 600, 616)
			fillHighLoopFilterStep(dst.Cr, stride, 16, 16, 4, 500, 516)
			cbBefore := [2]uint16{dst.Cb[3], dst.Cb[4]}
			crBefore := [2]uint16{dst.Cr[3], dst.Cr[4]}

			if err := h264ApplyLoopFilterEdgeHigh(dst, 0, 0, 0, 0, 1, [4]int16{3, 3, 3, 3}, qp, [2]int{qp, qp}, h264LoopFilterSliceParams{}, false, false, true, bitDepth); err != nil {
				t.Fatal(err)
			}
			if dst.Cb[3] == cbBefore[0] || dst.Cb[4] == cbBefore[1] || dst.Cr[3] == crBefore[0] || dst.Cr[4] == crBefore[1] {
				t.Fatalf("high 4:4:4 chroma luma-kernel edge did not filter: cb %v -> [%d %d] cr %v -> [%d %d]",
					cbBefore, dst.Cb[3], dst.Cb[4], crBefore, dst.Cr[3], dst.Cr[4])
			}
		})
	}
}

func TestMacroblockTablesFilterFrameHighDeblocksBoundary(t *testing.T) {
	const (
		mbWidth  = 2
		mbHeight = 1
	)

	for _, bitDepth := range []int{10, 12} {
		t.Run(bitDepthName(int32(bitDepth)), func(t *testing.T) {
			m, err := newMacroblockTables(mbWidth, mbHeight, 1)
			if err != nil {
				t.Fatal(err)
			}
			dst := &h264PicturePlanesHigh{
				Y:               make([]uint16, 32*16),
				Cb:              make([]uint16, 16*8),
				Cr:              make([]uint16, 16*8),
				LumaStride:      32,
				ChromaStride:    16,
				MBWidth:         mbWidth,
				MBHeight:        mbHeight,
				ChromaFormatIDC: 1,
			}
			fillHighLoopFilterStep(dst.Y, dst.LumaStride, 32, 16, 16, 400, 408)
			fillHighLoopFilterStep(dst.Cb, dst.ChromaStride, 16, 8, 8, 300, 308)
			fillHighLoopFilterStep(dst.Cr, dst.ChromaStride, 16, 8, 8, 200, 208)
			for mbXY := 0; mbXY < mbWidth*mbHeight; mbXY++ {
				m.MacroblockTyp[mbXY] = MBTypeIntra16x16
				m.QScaleTable[mbXY] = uint8(30 + 6*(bitDepth-8))
				m.SliceTable[mbXY] = 0
			}
			pps := cavlcFlatQMulPPS()
			pps.SPS = &SPS{
				BitDepthLuma:     int32(bitDepth),
				BitDepthChroma:   int32(bitDepth),
				ChromaFormatIDC:  1,
				FrameMBSOnlyFlag: 1,
			}
			params := []h264LoopFilterSliceParams{{
				PPS:              pps,
				ListCount:        1,
				DeblockingFilter: 1,
			}}
			yBefore := [2]uint16{dst.Y[15], dst.Y[16]}
			cbBefore := [2]uint16{dst.Cb[7], dst.Cb[8]}
			crBefore := [2]uint16{dst.Cr[7], dst.Cr[8]}

			if err := m.filterFrameHigh(dst, params); err != nil {
				t.Fatal(err)
			}
			if dst.Y[15] == yBefore[0] || dst.Y[16] == yBefore[1] {
				t.Fatalf("high frame luma boundary did not filter: %v -> [%d %d]", yBefore, dst.Y[15], dst.Y[16])
			}
			if dst.Cb[7] == cbBefore[0] || dst.Cb[8] == cbBefore[1] || dst.Cr[7] == crBefore[0] || dst.Cr[8] == crBefore[1] {
				t.Fatalf("high frame chroma boundary did not filter: cb %v -> [%d %d] cr %v -> [%d %d]",
					cbBefore, dst.Cb[7], dst.Cb[8], crBefore, dst.Cr[7], dst.Cr[8])
			}
		})
	}
}

func TestMacroblockTablesFilterFrameDeblocksPAFFFramePicture(t *testing.T) {
	const (
		mbWidth      = 2
		mbHeight     = 1
		lumaStride   = 32
		chromaStride = 16
		qp           = 30
	)
	m, err := newMacroblockTables(mbWidth, mbHeight, 1)
	if err != nil {
		t.Fatal(err)
	}
	dst := &h264PicturePlanes{
		Y:               make([]uint8, lumaStride*16),
		Cb:              make([]uint8, chromaStride*8),
		Cr:              make([]uint8, chromaStride*8),
		LumaStride:      lumaStride,
		ChromaStride:    chromaStride,
		MBWidth:         mbWidth,
		MBHeight:        mbHeight,
		ChromaFormatIDC: 1,
	}
	fillLoopFilterStepRows(dst.Y, lumaStride, 16, 16, 104, 112)
	fillLoopFilterStepRows(dst.Cb, chromaStride, 8, 8, 84, 92)
	fillLoopFilterStepRows(dst.Cr, chromaStride, 8, 8, 64, 72)
	for mbXY := 0; mbXY < mbWidth*mbHeight; mbXY++ {
		m.MacroblockTyp[mbXY] = MBTypeIntra16x16
		m.QScaleTable[mbXY] = qp
		m.SliceTable[mbXY] = 0
	}
	pps := cavlcFlatQMulPPS()
	pps.SPS = &SPS{
		BitDepthLuma:     8,
		BitDepthChroma:   8,
		ChromaFormatIDC:  1,
		FrameMBSOnlyFlag: 0,
		MBAFF:            0,
	}
	params := []h264LoopFilterSliceParams{{
		PPS:              pps,
		ListCount:        1,
		PictureStructure: PictureFrame,
		DeblockingFilter: 1,
	}}
	yBefore := [2]uint8{dst.Y[15], dst.Y[16]}
	cbBefore := [2]uint8{dst.Cb[7], dst.Cb[8]}
	crBefore := [2]uint8{dst.Cr[7], dst.Cr[8]}

	if err := m.filterFrame(dst, params); err != nil {
		t.Fatal(err)
	}
	if dst.Y[15] == yBefore[0] || dst.Y[16] == yBefore[1] {
		t.Fatalf("PAFF frame-picture luma boundary did not filter: %v -> [%d %d]", yBefore, dst.Y[15], dst.Y[16])
	}
	if dst.Cb[7] == cbBefore[0] || dst.Cb[8] == cbBefore[1] || dst.Cr[7] == crBefore[0] || dst.Cr[8] == crBefore[1] {
		t.Fatalf("PAFF frame-picture chroma boundary did not filter: cb %v -> [%d %d] cr %v -> [%d %d]",
			cbBefore, dst.Cb[7], dst.Cb[8], crBefore, dst.Cr[7], dst.Cr[8])
	}
}

func TestMacroblockTablesFilterFrameDeblocksPAFFFieldViews(t *testing.T) {
	const (
		mbWidth      = 2
		mbHeight     = 2
		lumaStride   = 32
		chromaStride = 16
		qp           = 30
	)
	m, err := newMacroblockTables(mbWidth, mbHeight, 1)
	if err != nil {
		t.Fatal(err)
	}
	dst := &h264PicturePlanes{
		Y:               make([]uint8, lumaStride*32),
		Cb:              make([]uint8, chromaStride*16),
		Cr:              make([]uint8, chromaStride*16),
		LumaStride:      lumaStride,
		ChromaStride:    chromaStride,
		MBWidth:         mbWidth,
		MBHeight:        mbHeight,
		ChromaFormatIDC: 1,
	}
	fillLoopFilterStepRows(dst.Y, lumaStride, 32, 16, 104, 112)
	fillLoopFilterStepRows(dst.Cb, chromaStride, 16, 8, 84, 92)
	fillLoopFilterStepRows(dst.Cr, chromaStride, 16, 8, 64, 72)
	for mbY := 0; mbY < mbHeight; mbY++ {
		for mbX := 0; mbX < mbWidth; mbX++ {
			mbXY := mbX + mbY*m.MBStride
			m.MacroblockTyp[mbXY] = MBTypeIntra16x16
			m.QScaleTable[mbXY] = qp
			m.SliceTable[mbXY] = uint16(mbY & 1)
		}
	}
	pps := cavlcFlatQMulPPS()
	pps.SPS = &SPS{
		BitDepthLuma:     8,
		BitDepthChroma:   8,
		ChromaFormatIDC:  1,
		FrameMBSOnlyFlag: 0,
	}
	params := []h264LoopFilterSliceParams{
		{PPS: pps, ListCount: 1, PictureStructure: PictureTopField, DeblockingFilter: 1},
		{PPS: pps, ListCount: 1, PictureStructure: PictureBottomField, DeblockingFilter: 1},
	}
	topYBefore := [2]uint8{dst.Y[15], dst.Y[16]}
	bottomYBefore := [2]uint8{dst.Y[lumaStride+15], dst.Y[lumaStride+16]}
	topCbBefore := [2]uint8{dst.Cb[7], dst.Cb[8]}
	bottomCbBefore := [2]uint8{dst.Cb[chromaStride+7], dst.Cb[chromaStride+8]}

	if err := m.filterFrame(dst, params); err != nil {
		t.Fatal(err)
	}
	if dst.Y[15] == topYBefore[0] || dst.Y[16] == topYBefore[1] {
		t.Fatalf("top-field luma boundary did not filter: %v -> [%d %d]", topYBefore, dst.Y[15], dst.Y[16])
	}
	if dst.Y[lumaStride+15] == bottomYBefore[0] || dst.Y[lumaStride+16] == bottomYBefore[1] {
		t.Fatalf("bottom-field luma boundary did not filter: %v -> [%d %d]", bottomYBefore, dst.Y[lumaStride+15], dst.Y[lumaStride+16])
	}
	if dst.Cb[7] == topCbBefore[0] || dst.Cb[8] == topCbBefore[1] {
		t.Fatalf("top-field chroma boundary did not filter: %v -> [%d %d]", topCbBefore, dst.Cb[7], dst.Cb[8])
	}
	if dst.Cb[chromaStride+7] == bottomCbBefore[0] || dst.Cb[chromaStride+8] == bottomCbBefore[1] {
		t.Fatalf("bottom-field chroma boundary did not filter: %v -> [%d %d]",
			bottomCbBefore, dst.Cb[chromaStride+7], dst.Cb[chromaStride+8])
	}
}

func TestMacroblockTablesFilterFieldAllowsComplementaryFieldPending(t *testing.T) {
	const (
		mbWidth      = 2
		mbHeight     = 2
		lumaStride   = 32
		chromaStride = 16
		qp           = 30
	)
	m, err := newMacroblockTables(mbWidth, mbHeight, 1)
	if err != nil {
		t.Fatal(err)
	}
	dst := &h264PicturePlanes{
		Y:               make([]uint8, lumaStride*32),
		Cb:              make([]uint8, chromaStride*16),
		Cr:              make([]uint8, chromaStride*16),
		LumaStride:      lumaStride,
		ChromaStride:    chromaStride,
		MBWidth:         mbWidth,
		MBHeight:        mbHeight,
		ChromaFormatIDC: 1,
	}
	fillLoopFilterStepRows(dst.Y, lumaStride, 32, 16, 104, 112)
	fillLoopFilterStepRows(dst.Cb, chromaStride, 16, 8, 84, 92)
	fillLoopFilterStepRows(dst.Cr, chromaStride, 16, 8, 64, 72)
	for mbX := 0; mbX < mbWidth; mbX++ {
		mbXY := mbX
		m.MacroblockTyp[mbXY] = MBTypeIntra16x16
		m.QScaleTable[mbXY] = qp
		m.SliceTable[mbXY] = 0
	}
	pps := cavlcFlatQMulPPS()
	pps.SPS = &SPS{
		BitDepthLuma:     8,
		BitDepthChroma:   8,
		ChromaFormatIDC:  1,
		FrameMBSOnlyFlag: 0,
	}
	params := []h264LoopFilterSliceParams{{
		PPS:              pps,
		ListCount:        1,
		PictureStructure: PictureTopField,
		DeblockingFilter: 1,
	}}
	topYBefore := [2]uint8{dst.Y[15], dst.Y[16]}
	bottomYBefore := [2]uint8{dst.Y[lumaStride+15], dst.Y[lumaStride+16]}

	if err := m.filterField(dst, params, PictureTopField); err != nil {
		t.Fatal(err)
	}
	if dst.Y[15] == topYBefore[0] || dst.Y[16] == topYBefore[1] {
		t.Fatalf("top field luma boundary did not filter before complementary field: %v -> [%d %d]",
			topYBefore, dst.Y[15], dst.Y[16])
	}
	if dst.Y[lumaStride+15] != bottomYBefore[0] || dst.Y[lumaStride+16] != bottomYBefore[1] {
		t.Fatalf("pending bottom field was modified: %v -> [%d %d]",
			bottomYBefore, dst.Y[lumaStride+15], dst.Y[lumaStride+16])
	}
}

func TestH264LoopFilterMVYLimitMatchesFieldMacroblockShape(t *testing.T) {
	if got := h264LoopFilterMVYLimit(MBType16x16 | MBTypeP0L0); got != 4 {
		t.Fatalf("progressive mvy limit = %d, want 4", got)
	}
	if got := h264LoopFilterMVYLimit(MBType16x16 | MBTypeP0L0 | MBTypeInterlaced); got != 2 {
		t.Fatalf("interlaced mvy limit = %d, want 2", got)
	}
}

func TestLoopFilterBoundaryStrengthFieldIntraHorizontalUsesBS3(t *testing.T) {
	m, err := newMacroblockTables(1, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := &h264LoopFilterContext{}
	bS, err := m.loopFilterBoundaryStrength(ctx, MBTypeIntra4x4|MBTypeInterlaced, MBTypeIntra4x4|MBTypeInterlaced, 1, 0, 1, 2, false)
	if err != nil {
		t.Fatal(err)
	}
	if bS != [4]int16{3, 3, 3, 3} {
		t.Fatalf("field intra horizontal bS = %v, want all 3", bS)
	}
	bS, err = m.loopFilterBoundaryStrength(ctx, MBTypeIntra4x4|MBTypeInterlaced, MBTypeIntra4x4|MBTypeInterlaced, 0, 0, 1, 2, false)
	if err != nil {
		t.Fatal(err)
	}
	if bS != [4]int16{4, 4, 4, 4} {
		t.Fatalf("field intra vertical bS = %v, want all 4", bS)
	}
	bS, err = m.loopFilterBoundaryStrength(ctx, MBTypeIntra4x4, MBTypeIntra4x4, 1, 0, 1, 4, false)
	if err != nil {
		t.Fatal(err)
	}
	if bS != [4]int16{4, 4, 4, 4} {
		t.Fatalf("frame intra horizontal bS = %v, want all 4", bS)
	}
}

func TestLoopFilterBoundaryStrengthFrameMBAFFHorizontalMixedInterlaceUsesBS1(t *testing.T) {
	m, err := newMacroblockTables(1, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := &h264LoopFilterContext{}
	mbType := MBType16x16 | MBTypeP0L0 | MBTypeInterlaced
	topType := MBType16x16 | MBTypeP0L0
	maskPar0 := mbType & (MBType16x16 | (MBType8x16 >> 1))

	bS, err := m.loopFilterBoundaryStrength(ctx, mbType, topType, 1, maskPar0, 1, h264LoopFilterMVYLimit(mbType), true)
	if err != nil {
		t.Fatal(err)
	}
	if bS != [4]int16{1, 1, 1, 1} {
		t.Fatalf("frame-MBAFF mixed horizontal bS = %v, want all 1", bS)
	}

	ctx.NonZeroCountCache[int(h264Scan8[0])+2] = 1
	bS, err = m.loopFilterBoundaryStrength(ctx, mbType, topType, 1, maskPar0, 1, h264LoopFilterMVYLimit(mbType), true)
	if err != nil {
		t.Fatal(err)
	}
	if bS != [4]int16{1, 1, 2, 1} {
		t.Fatalf("frame-MBAFF mixed horizontal nonzero bS = %v, want nonzero slot upgraded", bS)
	}
}

func TestLoopFilterMBAFFTopHorizontalStrengthUsesFieldNeighborRows(t *testing.T) {
	m, err := newMacroblockTables(1, 4, 1)
	if err != nil {
		t.Fatal(err)
	}
	current := 2 * m.MBStride
	topField := 0
	m.MacroblockTyp[current] = MBType16x16 | MBTypeP0L0
	m.MacroblockTyp[topField] = MBType8x8 | MBTypeP0L0 | MBType8x8DCT | MBTypeInterlaced
	m.CBPTable[topField] = 0x4000
	ctx := &h264LoopFilterContext{MBXY: current}
	ctx.NonZeroCountCache[int(h264Scan8[0])+2] = 1

	bS, err := m.loopFilterMBAFFTopHorizontalStrength(ctx, m.MacroblockTyp[current], topField, h264LoopFilterSliceParams{CABAC: false})
	if err != nil {
		t.Fatal(err)
	}
	if bS != [4]int16{2, 2, 2, 1} {
		t.Fatalf("MBAFF top-horizontal CAVLC bS = %v, want cbp/current-nnz driven [2 2 2 1]", bS)
	}
}

func TestMacroblockTablesFilterFrameHigh420SliceBoundaryModeSkipsCrossSliceBoundary(t *testing.T) {
	for _, bitDepth := range []int{10, 12} {
		t.Run(bitDepthName(int32(bitDepth)), func(t *testing.T) {
			dst := high420SliceBoundaryFrame()
			m, params := high420SliceBoundaryTables(t, bitDepth, 2)
			yBoundaryBefore := [2]uint16{dst.Y[15], dst.Y[16]}
			yInternalBefore := [2]uint16{dst.Y[23], dst.Y[24]}
			cbBoundaryBefore := [2]uint16{dst.Cb[7], dst.Cb[8]}
			cbInternalBefore := [2]uint16{dst.Cb[11], dst.Cb[12]}
			crBoundaryBefore := [2]uint16{dst.Cr[7], dst.Cr[8]}
			crInternalBefore := [2]uint16{dst.Cr[11], dst.Cr[12]}

			if err := m.filterFrameHigh(dst, params); err != nil {
				t.Fatal(err)
			}
			if dst.Y[15] != yBoundaryBefore[0] || dst.Y[16] != yBoundaryBefore[1] ||
				dst.Cb[7] != cbBoundaryBefore[0] || dst.Cb[8] != cbBoundaryBefore[1] ||
				dst.Cr[7] != crBoundaryBefore[0] || dst.Cr[8] != crBoundaryBefore[1] {
				t.Fatalf("high 4:2:0 slice-boundary mode filtered cross-slice edge: y %v -> [%d %d] cb %v -> [%d %d] cr %v -> [%d %d]",
					yBoundaryBefore, dst.Y[15], dst.Y[16], cbBoundaryBefore, dst.Cb[7], dst.Cb[8], crBoundaryBefore, dst.Cr[7], dst.Cr[8])
			}
			if dst.Y[23] == yInternalBefore[0] || dst.Y[24] == yInternalBefore[1] ||
				dst.Cb[11] == cbInternalBefore[0] || dst.Cb[12] == cbInternalBefore[1] ||
				dst.Cr[11] == crInternalBefore[0] || dst.Cr[12] == crInternalBefore[1] {
				t.Fatalf("high 4:2:0 slice-boundary mode did not filter same-slice internal edge: y %v -> [%d %d] cb %v -> [%d %d] cr %v -> [%d %d]",
					yInternalBefore, dst.Y[23], dst.Y[24], cbInternalBefore, dst.Cb[11], dst.Cb[12], crInternalBefore, dst.Cr[11], dst.Cr[12])
			}
		})
	}
}

func TestMacroblockTablesFilterFrameHigh422SliceBoundaryModeSkipsCrossSliceBoundary(t *testing.T) {
	for _, bitDepth := range []int{10, 12} {
		t.Run(bitDepthName(int32(bitDepth)), func(t *testing.T) {
			dst := high422SliceBoundaryFrame()
			m, params := high422SliceBoundaryTables(t, bitDepth, 2)
			yBoundaryBefore := [2]uint16{dst.Y[15], dst.Y[16]}
			yInternalBefore := [2]uint16{dst.Y[23], dst.Y[24]}
			cbBoundaryBefore := [2]uint16{dst.Cb[7], dst.Cb[8]}
			cbInternalBefore := [2]uint16{dst.Cb[11], dst.Cb[12]}
			crBoundaryBefore := [2]uint16{dst.Cr[7], dst.Cr[8]}
			crInternalBefore := [2]uint16{dst.Cr[11], dst.Cr[12]}

			if err := m.filterFrameHigh(dst, params); err != nil {
				t.Fatal(err)
			}
			if dst.Y[15] != yBoundaryBefore[0] || dst.Y[16] != yBoundaryBefore[1] ||
				dst.Cb[7] != cbBoundaryBefore[0] || dst.Cb[8] != cbBoundaryBefore[1] ||
				dst.Cr[7] != crBoundaryBefore[0] || dst.Cr[8] != crBoundaryBefore[1] {
				t.Fatalf("high 4:2:2 slice-boundary mode filtered cross-slice edge: y %v -> [%d %d] cb %v -> [%d %d] cr %v -> [%d %d]",
					yBoundaryBefore, dst.Y[15], dst.Y[16], cbBoundaryBefore, dst.Cb[7], dst.Cb[8], crBoundaryBefore, dst.Cr[7], dst.Cr[8])
			}
			if dst.Y[23] == yInternalBefore[0] || dst.Y[24] == yInternalBefore[1] ||
				dst.Cb[11] == cbInternalBefore[0] || dst.Cb[12] == cbInternalBefore[1] ||
				dst.Cr[11] == crInternalBefore[0] || dst.Cr[12] == crInternalBefore[1] {
				t.Fatalf("high 4:2:2 slice-boundary mode did not filter same-slice internal edge: y %v -> [%d %d] cb %v -> [%d %d] cr %v -> [%d %d]",
					yInternalBefore, dst.Y[23], dst.Y[24], cbInternalBefore, dst.Cb[11], dst.Cb[12], crInternalBefore, dst.Cr[11], dst.Cr[12])
			}
		})
	}
}

func TestMacroblockTablesFilterFrameHigh422DCTHorizontalChromaOnlyEdge(t *testing.T) {
	const (
		stride = 16
	)
	for _, bitDepth := range []int{10, 12} {
		t.Run(bitDepthName(int32(bitDepth)), func(t *testing.T) {
			qp := 30 + 6*(bitDepth-8)
			m, err := newMacroblockTables(1, 1, 2)
			if err != nil {
				t.Fatal(err)
			}
			dst := &h264PicturePlanesHigh{
				Y:               make([]uint16, stride*16),
				Cb:              make([]uint16, stride*16),
				Cr:              make([]uint16, stride*16),
				LumaStride:      stride,
				ChromaStride:    stride,
				MBWidth:         1,
				MBHeight:        1,
				ChromaFormatIDC: 2,
			}
			fillHighLoopFilterHorizontalStep(dst.Y, stride, 16, 16, 4, 400, 408)
			fillHighLoopFilterHorizontalStep(dst.Cb, stride, 16, 16, 4, 300, 308)
			fillHighLoopFilterHorizontalStep(dst.Cr, stride, 16, 16, 4, 200, 208)
			m.MacroblockTyp[0] = MBTypeIntra4x4 | MBType8x8DCT
			m.CBPTable[0] = 0xf
			m.QScaleTable[0] = uint8(qp)
			m.SliceTable[0] = 0
			pps := cavlcFlatQMulPPS()
			pps.SPS = &SPS{
				BitDepthLuma:     int32(bitDepth),
				BitDepthChroma:   int32(bitDepth),
				ChromaFormatIDC:  2,
				FrameMBSOnlyFlag: 1,
			}
			params := []h264LoopFilterSliceParams{{
				PPS:              pps,
				ListCount:        1,
				DeblockingFilter: 1,
			}}
			yBefore := [2]uint16{dst.Y[3*stride], dst.Y[4*stride]}
			cbBefore := [2]uint16{dst.Cb[3*stride], dst.Cb[4*stride]}
			crBefore := [2]uint16{dst.Cr[3*stride], dst.Cr[4*stride]}

			if err := m.filterFrameHigh(dst, params); err != nil {
				t.Fatal(err)
			}
			if dst.Y[3*stride] != yBefore[0] || dst.Y[4*stride] != yBefore[1] {
				t.Fatalf("high 4:2:2 8x8-DCT horizontal luma edge filtered: %v -> [%d %d]",
					yBefore, dst.Y[3*stride], dst.Y[4*stride])
			}
			if dst.Cb[3*stride] == cbBefore[0] || dst.Cb[4*stride] == cbBefore[1] ||
				dst.Cr[3*stride] == crBefore[0] || dst.Cr[4*stride] == crBefore[1] {
				t.Fatalf("high 4:2:2 8x8-DCT horizontal chroma-only edge did not filter: cb %v -> [%d %d] cr %v -> [%d %d]",
					cbBefore, dst.Cb[3*stride], dst.Cb[4*stride], crBefore, dst.Cr[3*stride], dst.Cr[4*stride])
			}
		})
	}
}

func fill444LoopFilterStep(pix []uint8, stride int, edge int, left uint8, right uint8) {
	for y := 0; y < 16; y++ {
		row := y * stride
		for x := 0; x < edge; x++ {
			pix[row+x] = left
		}
		for x := edge; x < 16; x++ {
			pix[row+x] = right
		}
	}
}

func fillLoopFilterStepRows(pix []uint8, stride int, height int, edge int, left uint8, right uint8) {
	for y := 0; y < height; y++ {
		row := y * stride
		for x := 0; x < edge; x++ {
			pix[row+x] = left
		}
		for x := edge; x < stride; x++ {
			pix[row+x] = right
		}
	}
}

func fillLoopFilterHorizontalStep(pix []uint8, stride int, width int, height int, edge int, top uint8, bottom uint8) {
	for y := 0; y < edge; y++ {
		row := y * stride
		for x := 0; x < width; x++ {
			pix[row+x] = top
		}
	}
	for y := edge; y < height; y++ {
		row := y * stride
		for x := 0; x < width; x++ {
			pix[row+x] = bottom
		}
	}
}

func high420SliceBoundaryFrame() *h264PicturePlanesHigh {
	const (
		lumaStride   = 32
		chromaStride = 16
	)
	dst := &h264PicturePlanesHigh{
		Y:               make([]uint16, lumaStride*16),
		Cb:              make([]uint16, chromaStride*8),
		Cr:              make([]uint16, chromaStride*8),
		LumaStride:      lumaStride,
		ChromaStride:    chromaStride,
		MBWidth:         2,
		MBHeight:        1,
		ChromaFormatIDC: 1,
	}
	fillHighLoopFilterStep(dst.Y, dst.LumaStride, 32, 16, 16, 400, 408)
	fillHighLoopFilterStep(dst.Cb, dst.ChromaStride, 16, 8, 8, 300, 308)
	fillHighLoopFilterStep(dst.Cr, dst.ChromaStride, 16, 8, 8, 200, 208)
	setHighLoopFilterRightRegion(dst.Y, dst.LumaStride, 16, 24, 416)
	setHighLoopFilterRightRegion(dst.Cb, dst.ChromaStride, 8, 12, 316)
	setHighLoopFilterRightRegion(dst.Cr, dst.ChromaStride, 8, 12, 216)
	return dst
}

func high420SliceBoundaryTables(t *testing.T, bitDepth int, deblockingFilter int) (*macroblockTables, []h264LoopFilterSliceParams) {
	t.Helper()
	m, err := newMacroblockTables(2, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	for mbXY := 0; mbXY < 2; mbXY++ {
		m.MacroblockTyp[mbXY] = MBTypeIntra16x16
		m.CBPTable[mbXY] = 1
		m.QScaleTable[mbXY] = uint8(30 + 6*(bitDepth-8))
		m.SliceTable[mbXY] = uint16(mbXY)
	}
	pps := cavlcFlatQMulPPS()
	pps.SPS = &SPS{
		BitDepthLuma:     int32(bitDepth),
		BitDepthChroma:   int32(bitDepth),
		ChromaFormatIDC:  1,
		FrameMBSOnlyFlag: 1,
	}
	return m, []h264LoopFilterSliceParams{
		{PPS: pps, ListCount: 1, DeblockingFilter: int32(deblockingFilter)},
		{PPS: pps, ListCount: 1, DeblockingFilter: int32(deblockingFilter)},
	}
}

func high422SliceBoundaryFrame() *h264PicturePlanesHigh {
	const (
		lumaStride   = 32
		chromaStride = 16
	)
	dst := &h264PicturePlanesHigh{
		Y:               make([]uint16, lumaStride*16),
		Cb:              make([]uint16, chromaStride*16),
		Cr:              make([]uint16, chromaStride*16),
		LumaStride:      lumaStride,
		ChromaStride:    chromaStride,
		MBWidth:         2,
		MBHeight:        1,
		ChromaFormatIDC: 2,
	}
	fillHighLoopFilterStep(dst.Y, dst.LumaStride, 32, 16, 16, 400, 408)
	fillHighLoopFilterStep(dst.Cb, dst.ChromaStride, 16, 16, 8, 300, 308)
	fillHighLoopFilterStep(dst.Cr, dst.ChromaStride, 16, 16, 8, 200, 208)
	setHighLoopFilterRightRegion(dst.Y, dst.LumaStride, 16, 24, 416)
	setHighLoopFilterRightRegion(dst.Cb, dst.ChromaStride, 16, 12, 316)
	setHighLoopFilterRightRegion(dst.Cr, dst.ChromaStride, 16, 12, 216)
	return dst
}

func high422SliceBoundaryTables(t *testing.T, bitDepth int, deblockingFilter int) (*macroblockTables, []h264LoopFilterSliceParams) {
	t.Helper()
	m, err := newMacroblockTables(2, 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	for mbXY := 0; mbXY < 2; mbXY++ {
		m.MacroblockTyp[mbXY] = MBTypeIntra16x16
		m.CBPTable[mbXY] = 1
		m.QScaleTable[mbXY] = uint8(30 + 6*(bitDepth-8))
		m.SliceTable[mbXY] = uint16(mbXY)
	}
	pps := cavlcFlatQMulPPS()
	pps.SPS = &SPS{
		BitDepthLuma:     int32(bitDepth),
		BitDepthChroma:   int32(bitDepth),
		ChromaFormatIDC:  2,
		FrameMBSOnlyFlag: 1,
	}
	return m, []h264LoopFilterSliceParams{
		{PPS: pps, ListCount: 1, DeblockingFilter: int32(deblockingFilter)},
		{PPS: pps, ListCount: 1, DeblockingFilter: int32(deblockingFilter)},
	}
}

func setHighLoopFilterRightRegion(pix []uint16, stride int, height int, edge int, value uint16) {
	for y := 0; y < height; y++ {
		row := y * stride
		for x := edge; x < stride; x++ {
			pix[row+x] = value
		}
	}
}

func fillHighLoopFilterHorizontalStep(pix []uint16, stride int, width int, height int, edge int, top uint16, bottom uint16) {
	for y := 0; y < edge; y++ {
		row := y * stride
		for x := 0; x < width; x++ {
			pix[row+x] = top
		}
	}
	for y := edge; y < height; y++ {
		row := y * stride
		for x := 0; x < width; x++ {
			pix[row+x] = bottom
		}
	}
}

func fillHighLoopFilterStep(pix []uint16, stride int, width int, height int, edge int, left uint16, right uint16) {
	for y := 0; y < height; y++ {
		row := y * stride
		for x := 0; x < edge; x++ {
			pix[row+x] = left
		}
		for x := edge; x < width; x++ {
			pix[row+x] = right
		}
	}
}
