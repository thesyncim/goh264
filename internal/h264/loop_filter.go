// SPDX-License-Identifier: LGPL-2.1-or-later
//
// Source-shaped simple frame-picture loop-filter integration from
// FFmpeg n8.0.1 libavcodec/h264_loopfilter.c ff_h264_filter_mb,
// filter_mb_dir, filter_mb_edge{v,h}, and fill_filter_caches.

package h264

var h264LoopFilterAlphaTable = [52]uint8{
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	4, 4, 5, 6, 7, 8, 9, 10, 12, 13, 15, 17, 20, 22, 25, 28,
	32, 36, 40, 45, 50, 56, 63, 71, 80, 90, 101, 113, 127, 144, 162, 182,
	203, 226, 255, 255,
}

var h264LoopFilterBetaTable = [52]uint8{
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	2, 2, 2, 3, 3, 3, 3, 4, 4, 4, 6, 6, 7, 7, 8, 8,
	9, 9, 10, 10, 11, 11, 12, 12, 13, 13, 14, 14, 15, 15, 16, 16,
	17, 17, 18, 18,
}

var h264LoopFilterTC0Table = [52][4]int8{
	{-1, 0, 0, 0},
	{-1, 0, 0, 0},
	{-1, 0, 0, 0},
	{-1, 0, 0, 0},
	{-1, 0, 0, 0},
	{-1, 0, 0, 0},
	{-1, 0, 0, 0},
	{-1, 0, 0, 0},
	{-1, 0, 0, 0},
	{-1, 0, 0, 0},
	{-1, 0, 0, 0},
	{-1, 0, 0, 0},
	{-1, 0, 0, 0},
	{-1, 0, 0, 0},
	{-1, 0, 0, 0},
	{-1, 0, 0, 0},
	{-1, 0, 0, 0},
	{-1, 0, 0, 1},
	{-1, 0, 0, 1},
	{-1, 0, 0, 1},
	{-1, 0, 0, 1},
	{-1, 0, 1, 1},
	{-1, 0, 1, 1},
	{-1, 1, 1, 1},
	{-1, 1, 1, 1},
	{-1, 1, 1, 1},
	{-1, 1, 1, 1},
	{-1, 1, 1, 2},
	{-1, 1, 1, 2},
	{-1, 1, 1, 2},
	{-1, 1, 1, 2},
	{-1, 1, 2, 3},
	{-1, 1, 2, 3},
	{-1, 2, 2, 3},
	{-1, 2, 2, 4},
	{-1, 2, 3, 4},
	{-1, 2, 3, 4},
	{-1, 3, 3, 5},
	{-1, 3, 4, 6},
	{-1, 3, 4, 6},
	{-1, 4, 5, 7},
	{-1, 4, 5, 8},
	{-1, 4, 6, 9},
	{-1, 5, 7, 10},
	{-1, 6, 8, 11},
	{-1, 6, 8, 13},
	{-1, 7, 10, 14},
	{-1, 8, 11, 16},
	{-1, 9, 12, 18},
	{-1, 10, 13, 20},
	{-1, 11, 15, 23},
	{-1, 13, 17, 25},
}

type h264LoopFilterSliceParams struct {
	PPS                  *PPS
	CABAC                bool
	ListCount            int
	DeblockingFilter     int32
	SliceAlphaC0Offset   int32
	SliceBetaOffset      int32
	ChromaQPIndexOffset0 int32
	ChromaQPIndexOffset1 int32
}

type h264LoopFilterContext struct {
	MBXY              int
	TopMBXY           int
	LeftMBXY          int
	TopType           uint32
	LeftType          uint32
	CBP               int
	NonZeroCountCache [h264NonZeroCountCacheSize]uint8
	Motion            macroblockMotionCache
}

func h264LoopFilterSliceParamsFromHeader(sh *SliceHeader) h264LoopFilterSliceParams {
	if sh == nil || sh.PPS == nil {
		return h264LoopFilterSliceParams{}
	}
	return h264LoopFilterSliceParams{
		PPS:                  sh.PPS,
		CABAC:                sh.PPS.CABAC != 0,
		ListCount:            int(sh.ListCount),
		DeblockingFilter:     sh.DeblockingFilter,
		SliceAlphaC0Offset:   sh.SliceAlphaC0Offset,
		SliceBetaOffset:      sh.SliceBetaOffset,
		ChromaQPIndexOffset0: sh.PPS.ChromaQPIndexOffset[0],
		ChromaQPIndexOffset1: sh.PPS.ChromaQPIndexOffset[1],
	}
}

func (p h264LoopFilterSliceParams) validate() error {
	if p.DeblockingFilter < 0 || p.DeblockingFilter > 2 || p.ListCount < 0 || p.ListCount > 2 {
		return ErrInvalidData
	}
	if p.DeblockingFilter == 0 {
		return nil
	}
	if p.PPS == nil || p.PPS.SPS == nil {
		return ErrInvalidData
	}
	if p.PPS.SPS.MBAFF != 0 || p.PPS.SPS.FrameMBSOnlyFlag == 0 {
		return ErrUnsupported
	}
	if err := checkH264LoopFilterBitDepth(int(p.PPS.SPS.BitDepthLuma)); err != nil {
		return err
	}
	if p.PPS.SPS.BitDepthLuma != p.PPS.SPS.BitDepthChroma {
		return ErrUnsupported
	}
	return nil
}

func (m *macroblockTables) filterFrame(dst *h264PicturePlanes, params []h264LoopFilterSliceParams) error {
	if m == nil || dst == nil {
		return ErrInvalidData
	}
	if err := dst.validate(); err != nil {
		return err
	}
	if m.MBWidth != dst.MBWidth || m.MBHeight != dst.MBHeight || m.ChromaFormatIDC != dst.ChromaFormatIDC {
		return ErrInvalidData
	}
	for mbY := 0; mbY < m.MBHeight; mbY++ {
		for mbX := 0; mbX < m.MBWidth; mbX++ {
			mbXY := mbX + mbY*m.MBStride
			sliceNum := m.SliceTable[mbXY]
			if sliceNum == ^uint16(0) || int(sliceNum) >= len(params) {
				return ErrInvalidData
			}
			p := params[sliceNum]
			if err := p.validate(); err != nil {
				return err
			}
			if p.DeblockingFilter == 0 {
				continue
			}
			ctx, err := m.fillLoopFilterCachesFrame(mbXY, sliceNum, p)
			if err != nil {
				return err
			}
			if err := m.filterFrameMacroblock(dst, mbX, mbY, p, &ctx); err != nil {
				return err
			}
		}
	}
	return nil
}

func (m *macroblockTables) filterFrameHigh(dst *h264PicturePlanesHigh, params []h264LoopFilterSliceParams) error {
	if m == nil || dst == nil {
		return ErrInvalidData
	}
	if err := dst.validate(); err != nil {
		return err
	}
	if m.MBWidth != dst.MBWidth || m.MBHeight != dst.MBHeight || m.ChromaFormatIDC != dst.ChromaFormatIDC {
		return ErrInvalidData
	}
	for mbY := 0; mbY < m.MBHeight; mbY++ {
		for mbX := 0; mbX < m.MBWidth; mbX++ {
			mbXY := mbX + mbY*m.MBStride
			sliceNum := m.SliceTable[mbXY]
			if sliceNum == ^uint16(0) || int(sliceNum) >= len(params) {
				return ErrInvalidData
			}
			p := params[sliceNum]
			if err := p.validate(); err != nil {
				return err
			}
			if p.DeblockingFilter == 0 {
				continue
			}
			ctx, err := m.fillLoopFilterCachesFrame(mbXY, sliceNum, p)
			if err != nil {
				return err
			}
			if err := m.filterFrameMacroblockHigh(dst, mbX, mbY, p, &ctx); err != nil {
				return err
			}
		}
	}
	return nil
}

func (m *macroblockTables) fillLoopFilterCachesFrame(mbXY int, sliceNum uint16, p h264LoopFilterSliceParams) (h264LoopFilterContext, error) {
	var ctx h264LoopFilterContext
	if err := m.checkCodedMBXY(mbXY); err != nil {
		return ctx, err
	}
	if err := p.validate(); err != nil {
		return ctx, err
	}

	mbX := mbXY % m.MBStride
	mbY := mbXY / m.MBStride
	topXY := -1
	leftXY := -1
	if mbY > 0 {
		topXY = mbXY - m.MBStride
	}
	if mbX > 0 {
		leftXY = mbXY - 1
	}

	topType := m.macroblockTypeIfCoded(topXY)
	leftType := m.macroblockTypeIfCoded(leftXY)
	if p.DeblockingFilter == 2 {
		if !m.sameSlice(topXY, sliceNum) {
			topType = 0
		}
		if !m.sameSlice(leftXY, sliceNum) {
			leftType = 0
		}
	}

	mbType := m.MacroblockTyp[mbXY]
	ctx = h264LoopFilterContext{
		MBXY:     mbXY,
		TopMBXY:  topXY,
		LeftMBXY: leftXY,
		TopType:  topType,
		LeftType: leftType,
		CBP:      m.CBPTable[mbXY],
	}
	if isIntra(mbType) {
		return ctx, nil
	}

	if err := m.fillLoopFilterCachesInterFrame(&ctx, mbXY, topXY, leftXY, mbType, topType, leftType, 0); err != nil {
		return ctx, err
	}
	if p.ListCount == 2 {
		if err := m.fillLoopFilterCachesInterFrame(&ctx, mbXY, topXY, leftXY, mbType, topType, leftType, 1); err != nil {
			return ctx, err
		}
	}

	nnz := m.NonZeroCount[mbXY]
	copy(ctx.NonZeroCountCache[4+8*1:4+8*1+4], nnz[0:4])
	copy(ctx.NonZeroCountCache[4+8*2:4+8*2+4], nnz[4:8])
	copy(ctx.NonZeroCountCache[4+8*3:4+8*3+4], nnz[8:12])
	copy(ctx.NonZeroCountCache[4+8*4:4+8*4+4], nnz[12:16])

	if topType != 0 {
		topNNZ := m.NonZeroCount[topXY]
		copy(ctx.NonZeroCountCache[4+8*0:4+8*0+4], topNNZ[3*4:3*4+4])
	}
	if leftType != 0 {
		leftNNZ := m.NonZeroCount[leftXY]
		ctx.NonZeroCountCache[3+8*1] = leftNNZ[3+0*4]
		ctx.NonZeroCountCache[3+8*2] = leftNNZ[3+1*4]
		ctx.NonZeroCountCache[3+8*3] = leftNNZ[3+2*4]
		ctx.NonZeroCountCache[3+8*4] = leftNNZ[3+3*4]
	}

	if !p.CABAC && p.PPS.Transform8x8Mode != 0 {
		if is8x8DCT(topType) {
			ctx.NonZeroCountCache[4+8*0] = uint8((m.CBPTable[topXY] & 0x4000) >> 12)
			ctx.NonZeroCountCache[5+8*0] = ctx.NonZeroCountCache[4+8*0]
			ctx.NonZeroCountCache[6+8*0] = uint8((m.CBPTable[topXY] & 0x8000) >> 12)
			ctx.NonZeroCountCache[7+8*0] = ctx.NonZeroCountCache[6+8*0]
		}
		if is8x8DCT(leftType) {
			ctx.NonZeroCountCache[3+8*1] = uint8((m.CBPTable[leftXY] & 0x2000) >> 12)
			ctx.NonZeroCountCache[3+8*2] = ctx.NonZeroCountCache[3+8*1]
			ctx.NonZeroCountCache[3+8*3] = uint8((m.CBPTable[leftXY] & 0x8000) >> 12)
			ctx.NonZeroCountCache[3+8*4] = ctx.NonZeroCountCache[3+8*3]
		}
		if is8x8DCT(mbType) {
			h264SetLoopFilter8x8DCTNNZ(&ctx.NonZeroCountCache, 0, (ctx.CBP&0x1000)>>12)
			h264SetLoopFilter8x8DCTNNZ(&ctx.NonZeroCountCache, 4, (ctx.CBP&0x2000)>>12)
			h264SetLoopFilter8x8DCTNNZ(&ctx.NonZeroCountCache, 8, (ctx.CBP&0x4000)>>12)
			h264SetLoopFilter8x8DCTNNZ(&ctx.NonZeroCountCache, 12, (ctx.CBP&0x8000)>>12)
		}
	}

	return ctx, nil
}

func h264SetLoopFilter8x8DCTNNZ(cache *[h264NonZeroCountCacheSize]uint8, base int, value int) {
	v := uint8(value)
	cache[h264Scan8[base+0]] = v
	cache[h264Scan8[base+1]] = v
	cache[h264Scan8[base+2]] = v
	cache[h264Scan8[base+3]] = v
}

func (m *macroblockTables) fillLoopFilterCachesInterFrame(ctx *h264LoopFilterContext, mbXY int, topXY int, leftXY int, mbType uint32, topType uint32, leftType uint32, list int) error {
	if ctx == nil || list < 0 || list > 1 {
		return ErrInvalidData
	}
	base := int(h264Scan8[0])
	if isInter(mbType) || isDirect(mbType) {
		if usesList(topType, list) {
			if err := m.copyTopMotionForLoopFilter(ctx, topXY, list, base); err != nil {
				return err
			}
		} else {
			clearMotionRow(&ctx.Motion.MV[list], base-8, 4)
			fillRefRow(&ctx.Motion.Ref[list], base-8, 4, h264ListNotUsed)
		}

		if usesList(leftType, list) {
			if err := m.copyLeftMotionForLoopFilter(ctx, leftXY, list, base); err != nil {
				return err
			}
		} else {
			for row := 0; row < 4; row++ {
				idx := base - 1 + row*8
				ctx.Motion.MV[list][idx] = [2]int16{}
				ctx.Motion.Ref[list][idx] = h264ListNotUsed
			}
		}
	}

	if !usesList(mbType, list) {
		fillMotionRectangle(&ctx.Motion.MV[list], base, 4, 4, 8, [2]int16{})
		fillRefRectangle(&ctx.Motion.Ref[list], base, 4, 4, 8, h264ListNotUsed)
		return nil
	}

	if err := m.copyCurrentMotionForLoopFilter(ctx, mbXY, list, base); err != nil {
		return err
	}
	return nil
}

func (m *macroblockTables) copyTopMotionForLoopFilter(ctx *h264LoopFilterContext, topXY int, list int, base int) error {
	if err := m.checkCodedMBXY(topXY); err != nil {
		return err
	}
	src := int(m.MB2BXY[topXY]) + 3*m.BStride
	if err := checkRange(len(m.MotionVal[list]), src, 4); err != nil {
		return err
	}
	copy(ctx.Motion.MV[list][base-8:base-8+4], m.MotionVal[list][src:src+4])

	refBase := 4*topXY + 2
	if err := checkRange(len(m.RefIndex[list]), refBase, 2); err != nil {
		return err
	}
	ctx.Motion.Ref[list][base+0-8] = m.RefIndex[list][refBase+0]
	ctx.Motion.Ref[list][base+1-8] = m.RefIndex[list][refBase+0]
	ctx.Motion.Ref[list][base+2-8] = m.RefIndex[list][refBase+1]
	ctx.Motion.Ref[list][base+3-8] = m.RefIndex[list][refBase+1]
	return nil
}

func (m *macroblockTables) copyLeftMotionForLoopFilter(ctx *h264LoopFilterContext, leftXY int, list int, base int) error {
	if err := m.checkCodedMBXY(leftXY); err != nil {
		return err
	}
	bXY := int(m.MB2BXY[leftXY]) + 3
	refBase := 4*leftXY + 1
	if err := checkRange(len(m.RefIndex[list]), refBase, 3); err != nil {
		return err
	}
	for row := 0; row < 4; row++ {
		mvIdx := bXY + row*m.BStride
		if err := checkRange(len(m.MotionVal[list]), mvIdx, 1); err != nil {
			return err
		}
		cacheIdx := base - 1 + row*8
		ctx.Motion.MV[list][cacheIdx] = m.MotionVal[list][mvIdx]
		ctx.Motion.Ref[list][cacheIdx] = m.RefIndex[list][refBase+2*(row>>1)]
	}
	return nil
}

func (m *macroblockTables) copyCurrentMotionForLoopFilter(ctx *h264LoopFilterContext, mbXY int, list int, base int) error {
	if err := m.checkCodedMBXY(mbXY); err != nil {
		return err
	}
	mbX := mbXY % m.MBStride
	mbY := mbXY / m.MBStride
	src := 4*mbX + 4*mbY*m.BStride
	for row := 0; row < 4; row++ {
		if err := checkRange(len(m.MotionVal[list]), src+row*m.BStride, 4); err != nil {
			return err
		}
		copy(ctx.Motion.MV[list][base+row*8:base+row*8+4], m.MotionVal[list][src+row*m.BStride:src+row*m.BStride+4])
	}

	refBase := 4 * mbXY
	if err := checkRange(len(m.RefIndex[list]), refBase, 4); err != nil {
		return err
	}
	for row := 0; row < 2; row++ {
		ctx.Motion.Ref[list][base+row*8+0] = m.RefIndex[list][refBase+0]
		ctx.Motion.Ref[list][base+row*8+1] = m.RefIndex[list][refBase+0]
		ctx.Motion.Ref[list][base+row*8+2] = m.RefIndex[list][refBase+1]
		ctx.Motion.Ref[list][base+row*8+3] = m.RefIndex[list][refBase+1]
	}
	for row := 2; row < 4; row++ {
		ctx.Motion.Ref[list][base+row*8+0] = m.RefIndex[list][refBase+2]
		ctx.Motion.Ref[list][base+row*8+1] = m.RefIndex[list][refBase+2]
		ctx.Motion.Ref[list][base+row*8+2] = m.RefIndex[list][refBase+3]
		ctx.Motion.Ref[list][base+row*8+3] = m.RefIndex[list][refBase+3]
	}
	return nil
}

func (m *macroblockTables) filterFrameMacroblock(dst *h264PicturePlanes, mbX int, mbY int, p h264LoopFilterSliceParams, ctx *h264LoopFilterContext) error {
	if ctx == nil || dst == nil {
		return ErrInvalidData
	}
	dstY, dstCb, dstCr, err := h264MBDestPartOffsets(dst, mbX, mbY, 0, 0)
	if err != nil {
		return err
	}
	if err := m.filterFrameMacroblockDir(dst, dstY, dstCb, dstCr, p, ctx, 0); err != nil {
		return err
	}
	return m.filterFrameMacroblockDir(dst, dstY, dstCb, dstCr, p, ctx, 1)
}

func (m *macroblockTables) filterFrameMacroblockHigh(dst *h264PicturePlanesHigh, mbX int, mbY int, p h264LoopFilterSliceParams, ctx *h264LoopFilterContext) error {
	if ctx == nil || dst == nil {
		return ErrInvalidData
	}
	dstY, dstCb, dstCr, err := h264MBDestPartOffsetsHigh(dst, mbX, mbY, 0, 0)
	if err != nil {
		return err
	}
	if err := m.filterFrameMacroblockDirHigh(dst, dstY, dstCb, dstCr, p, ctx, 0); err != nil {
		return err
	}
	return m.filterFrameMacroblockDirHigh(dst, dstY, dstCb, dstCr, p, ctx, 1)
}

func (m *macroblockTables) filterFrameMacroblockDir(dst *h264PicturePlanes, dstY int, dstCb int, dstCr int, p h264LoopFilterSliceParams, ctx *h264LoopFilterContext, dir int) error {
	if dir < 0 || dir > 1 {
		return ErrInvalidData
	}
	mbType := m.MacroblockTyp[ctx.MBXY]
	mbmXY := ctx.LeftMBXY
	mbmType := ctx.LeftType
	if dir != 0 {
		mbmXY = ctx.TopMBXY
		mbmType = ctx.TopType
	}

	maskEdgeTab := [2][8]int{
		{0, 3, 3, 3, 1, 1, 1, 1},
		{0, 3, 1, 1, 3, 3, 3, 3},
	}
	maskEdge := maskEdgeTab[dir][(mbType>>3)&7]
	edges := 4
	if maskEdge == 3 && ctx.CBP&15 == 0 {
		edges = 1
	}
	maskPar0 := mbType & (MBType16x16 | (MBType8x16 >> uint(dir)))
	mvyLimit := 4

	if mbmType != 0 {
		bS, err := m.loopFilterBoundaryStrength(ctx, mbType, mbmType, dir, maskPar0, p.ListCount, mvyLimit)
		if err != nil {
			return err
		}
		if h264LoopFilterBSSum(bS) != 0 {
			qp := (int(m.QScaleTable[ctx.MBXY]) + int(m.QScaleTable[mbmXY]) + 1) >> 1
			chromaQP := [2]int{
				(int(p.PPS.ChromaQPTable[0][m.QScaleTable[ctx.MBXY]]) + int(p.PPS.ChromaQPTable[0][m.QScaleTable[mbmXY]]) + 1) >> 1,
				(int(p.PPS.ChromaQPTable[1][m.QScaleTable[ctx.MBXY]]) + int(p.PPS.ChromaQPTable[1][m.QScaleTable[mbmXY]]) + 1) >> 1,
			}
			if err := h264ApplyLoopFilterEdge(dst, dstY, dstCb, dstCr, dir, 0, bS, qp, chromaQP, p, true, true, true); err != nil {
				return err
			}
		}
	}

	for edge := 1; edge < edges; edge++ {
		deblockEdge := !is8x8DCT(mbType & (uint32(edge) << 24))
		if !deblockEdge && (dst.ChromaFormatIDC != 2 || dir == 0) {
			continue
		}

		bS, err := m.loopFilterInternalStrength(ctx, mbType, dir, edge, maskEdge, maskPar0, p.ListCount, mvyLimit)
		if err != nil {
			return err
		}
		if h264LoopFilterBSSum(bS) == 0 {
			continue
		}

		qp := int(m.QScaleTable[ctx.MBXY])
		chromaQP := [2]int{
			int(p.PPS.ChromaQPTable[0][m.QScaleTable[ctx.MBXY]]),
			int(p.PPS.ChromaQPTable[1][m.QScaleTable[ctx.MBXY]]),
		}
		filterLuma := deblockEdge
		filterChroma := true
		if dir == 0 && dst.ChromaFormatIDC != 3 && edge&1 != 0 {
			filterChroma = false
		}
		if dir == 1 && dst.ChromaFormatIDC == 1 && edge&1 != 0 {
			filterChroma = false
		}
		if err := h264ApplyLoopFilterEdge(dst, dstY, dstCb, dstCr, dir, edge, bS, qp, chromaQP, p, false, filterLuma, filterChroma); err != nil {
			return err
		}
	}
	return nil
}

func (m *macroblockTables) filterFrameMacroblockDirHigh(dst *h264PicturePlanesHigh, dstY int, dstCb int, dstCr int, p h264LoopFilterSliceParams, ctx *h264LoopFilterContext, dir int) error {
	if dir < 0 || dir > 1 {
		return ErrInvalidData
	}
	if p.PPS == nil || p.PPS.SPS == nil {
		return ErrInvalidData
	}
	bitDepth := int(p.PPS.SPS.BitDepthLuma)
	mbType := m.MacroblockTyp[ctx.MBXY]
	mbmXY := ctx.LeftMBXY
	mbmType := ctx.LeftType
	if dir != 0 {
		mbmXY = ctx.TopMBXY
		mbmType = ctx.TopType
	}

	maskEdgeTab := [2][8]int{
		{0, 3, 3, 3, 1, 1, 1, 1},
		{0, 3, 1, 1, 3, 3, 3, 3},
	}
	maskEdge := maskEdgeTab[dir][(mbType>>3)&7]
	edges := 4
	if maskEdge == 3 && ctx.CBP&15 == 0 {
		edges = 1
	}
	maskPar0 := mbType & (MBType16x16 | (MBType8x16 >> uint(dir)))
	mvyLimit := 4

	if mbmType != 0 {
		bS, err := m.loopFilterBoundaryStrength(ctx, mbType, mbmType, dir, maskPar0, p.ListCount, mvyLimit)
		if err != nil {
			return err
		}
		if h264LoopFilterBSSum(bS) != 0 {
			qp := (int(m.QScaleTable[ctx.MBXY]) + int(m.QScaleTable[mbmXY]) + 1) >> 1
			chromaQP := [2]int{
				(int(p.PPS.ChromaQPTable[0][m.QScaleTable[ctx.MBXY]]) + int(p.PPS.ChromaQPTable[0][m.QScaleTable[mbmXY]]) + 1) >> 1,
				(int(p.PPS.ChromaQPTable[1][m.QScaleTable[ctx.MBXY]]) + int(p.PPS.ChromaQPTable[1][m.QScaleTable[mbmXY]]) + 1) >> 1,
			}
			if err := h264ApplyLoopFilterEdgeHigh(dst, dstY, dstCb, dstCr, dir, 0, bS, qp, chromaQP, p, true, true, true, bitDepth); err != nil {
				return err
			}
		}
	}

	for edge := 1; edge < edges; edge++ {
		deblockEdge := !is8x8DCT(mbType & (uint32(edge) << 24))
		if !deblockEdge && (dst.ChromaFormatIDC != 2 || dir == 0) {
			continue
		}

		bS, err := m.loopFilterInternalStrength(ctx, mbType, dir, edge, maskEdge, maskPar0, p.ListCount, mvyLimit)
		if err != nil {
			return err
		}
		if h264LoopFilterBSSum(bS) == 0 {
			continue
		}

		qp := int(m.QScaleTable[ctx.MBXY])
		chromaQP := [2]int{
			int(p.PPS.ChromaQPTable[0][m.QScaleTable[ctx.MBXY]]),
			int(p.PPS.ChromaQPTable[1][m.QScaleTable[ctx.MBXY]]),
		}
		filterLuma := deblockEdge
		filterChroma := true
		if dir == 0 && dst.ChromaFormatIDC != 3 && edge&1 != 0 {
			filterChroma = false
		}
		if dir == 1 && dst.ChromaFormatIDC == 1 && edge&1 != 0 {
			filterChroma = false
		}
		if err := h264ApplyLoopFilterEdgeHigh(dst, dstY, dstCb, dstCr, dir, edge, bS, qp, chromaQP, p, false, filterLuma, filterChroma, bitDepth); err != nil {
			return err
		}
	}
	return nil
}

func (m *macroblockTables) loopFilterBoundaryStrength(ctx *h264LoopFilterContext, mbType uint32, mbmType uint32, dir int, maskPar0 uint32, listCount int, mvyLimit int) ([4]int16, error) {
	var bS [4]int16
	if isIntra(mbType | mbmType) {
		for i := range bS {
			bS[i] = 4
		}
		return bS, nil
	}

	mvDone := false
	if maskPar0 != 0 && mbmType&(MBType16x16|(MBType8x16>>uint(dir))) != 0 {
		bIdx := int(h264Scan8[0])
		bnIdx := bIdx - 1
		if dir != 0 {
			bnIdx = bIdx - 8
		}
		v, err := h264LoopFilterCheckMV(ctx, bIdx, bnIdx, listCount, mvyLimit)
		if err != nil {
			return bS, err
		}
		for i := range bS {
			bS[i] = v
		}
		mvDone = true
	}

	for i := 0; i < 4; i++ {
		x := 0
		y := i
		if dir != 0 {
			x = i
			y = 0
		}
		bIdx := int(h264Scan8[0]) + x + 8*y
		bnIdx := bIdx - 1
		if dir != 0 {
			bnIdx = bIdx - 8
		}
		if ctx.NonZeroCountCache[bIdx]|ctx.NonZeroCountCache[bnIdx] != 0 {
			bS[i] = 2
		} else if !mvDone {
			v, err := h264LoopFilterCheckMV(ctx, bIdx, bnIdx, listCount, mvyLimit)
			if err != nil {
				return bS, err
			}
			bS[i] = v
		}
	}
	return bS, nil
}

func (m *macroblockTables) loopFilterInternalStrength(ctx *h264LoopFilterContext, mbType uint32, dir int, edge int, maskEdge int, maskPar0 uint32, listCount int, mvyLimit int) ([4]int16, error) {
	var bS [4]int16
	if isIntra(mbType) {
		for i := range bS {
			bS[i] = 3
		}
		return bS, nil
	}

	mvDone := false
	if edge&maskEdge != 0 {
		mvDone = true
	} else if maskPar0 != 0 {
		bIdx := int(h264Scan8[0]) + edge
		if dir != 0 {
			bIdx = int(h264Scan8[0]) + edge*8
		}
		bnIdx := bIdx - 1
		if dir != 0 {
			bnIdx = bIdx - 8
		}
		v, err := h264LoopFilterCheckMV(ctx, bIdx, bnIdx, listCount, mvyLimit)
		if err != nil {
			return bS, err
		}
		for i := range bS {
			bS[i] = v
		}
		mvDone = true
	}

	for i := 0; i < 4; i++ {
		x := edge
		y := i
		if dir != 0 {
			x = i
			y = edge
		}
		bIdx := int(h264Scan8[0]) + x + 8*y
		bnIdx := bIdx - 1
		if dir != 0 {
			bnIdx = bIdx - 8
		}
		if ctx.NonZeroCountCache[bIdx]|ctx.NonZeroCountCache[bnIdx] != 0 {
			bS[i] = 2
		} else if !mvDone {
			v, err := h264LoopFilterCheckMV(ctx, bIdx, bnIdx, listCount, mvyLimit)
			if err != nil {
				return bS, err
			}
			bS[i] = v
		}
	}
	return bS, nil
}

func h264LoopFilterCheckMV(ctx *h264LoopFilterContext, bIdx int, bnIdx int, listCount int, mvyLimit int) (int16, error) {
	if ctx == nil || bIdx < 0 || bIdx >= h264MotionCacheSize || bnIdx < 0 || bnIdx >= h264MotionCacheSize || listCount < 1 || listCount > 2 {
		return 0, ErrInvalidData
	}
	v := ctx.Motion.Ref[0][bIdx] != ctx.Motion.Ref[0][bnIdx]
	if !v && ctx.Motion.Ref[0][bIdx] != h264ListNotUsed {
		v = h264LoopFilterMVDiff(ctx.Motion.MV[0][bIdx], ctx.Motion.MV[0][bnIdx], mvyLimit)
	}
	if listCount == 2 {
		if !v {
			v = ctx.Motion.Ref[1][bIdx] != ctx.Motion.Ref[1][bnIdx] ||
				h264LoopFilterMVDiff(ctx.Motion.MV[1][bIdx], ctx.Motion.MV[1][bnIdx], mvyLimit)
		}
		if v {
			if ctx.Motion.Ref[0][bIdx] != ctx.Motion.Ref[1][bnIdx] || ctx.Motion.Ref[1][bIdx] != ctx.Motion.Ref[0][bnIdx] {
				return 1, nil
			}
			v = h264LoopFilterMVDiff(ctx.Motion.MV[0][bIdx], ctx.Motion.MV[1][bnIdx], mvyLimit) ||
				h264LoopFilterMVDiff(ctx.Motion.MV[1][bIdx], ctx.Motion.MV[0][bnIdx], mvyLimit)
		}
	}
	if v {
		return 1, nil
	}
	return 0, nil
}

func h264LoopFilterMVDiff(a [2]int16, b [2]int16, mvyLimit int) bool {
	return absInt(int(a[0])-int(b[0])) >= 4 || absInt(int(a[1])-int(b[1])) >= mvyLimit
}

func h264LoopFilterBSSum(bS [4]int16) int16 {
	return bS[0] + bS[1] + bS[2] + bS[3]
}

func h264ApplyLoopFilterEdge(dst *h264PicturePlanes, dstY int, dstCb int, dstCr int, dir int, edge int, bS [4]int16, qp int, chromaQP [2]int, p h264LoopFilterSliceParams, intra bool, filterLuma bool, filterChroma bool) error {
	if dst == nil || edge < 0 || edge > 3 {
		return ErrInvalidData
	}
	if filterLuma {
		if dir == 0 {
			if err := h264FilterMBEdgeVLuma(dst.Y, dstY+4*edge, dst.LumaStride, bS, qp, int(p.SliceAlphaC0Offset), int(p.SliceBetaOffset), intra); err != nil {
				return err
			}
		} else if err := h264FilterMBEdgeHLuma(dst.Y, dstY+4*edge*dst.LumaStride, dst.LumaStride, bS, qp, int(p.SliceAlphaC0Offset), int(p.SliceBetaOffset), intra); err != nil {
			return err
		}
	}
	if !filterChroma || dst.ChromaFormatIDC == 0 {
		return nil
	}
	switch dst.ChromaFormatIDC {
	case 1:
		if edge&1 != 0 {
			return nil
		}
		if dir == 0 {
			if err := h264FilterMBEdgeVChroma(dst.Cb, dstCb+2*edge, dst.ChromaStride, bS, chromaQP[0], int(p.SliceAlphaC0Offset), int(p.SliceBetaOffset), intra, dst.ChromaFormatIDC); err != nil {
				return err
			}
			return h264FilterMBEdgeVChroma(dst.Cr, dstCr+2*edge, dst.ChromaStride, bS, chromaQP[1], int(p.SliceAlphaC0Offset), int(p.SliceBetaOffset), intra, dst.ChromaFormatIDC)
		}
		if err := h264FilterMBEdgeHChroma(dst.Cb, dstCb+2*edge*dst.ChromaStride, dst.ChromaStride, bS, chromaQP[0], int(p.SliceAlphaC0Offset), int(p.SliceBetaOffset), intra); err != nil {
			return err
		}
		return h264FilterMBEdgeHChroma(dst.Cr, dstCr+2*edge*dst.ChromaStride, dst.ChromaStride, bS, chromaQP[1], int(p.SliceAlphaC0Offset), int(p.SliceBetaOffset), intra)
	case 2:
		if dir == 0 {
			if edge&1 != 0 {
				return nil
			}
			if err := h264FilterMBEdgeVChroma(dst.Cb, dstCb+2*edge, dst.ChromaStride, bS, chromaQP[0], int(p.SliceAlphaC0Offset), int(p.SliceBetaOffset), intra, dst.ChromaFormatIDC); err != nil {
				return err
			}
			return h264FilterMBEdgeVChroma(dst.Cr, dstCr+2*edge, dst.ChromaStride, bS, chromaQP[1], int(p.SliceAlphaC0Offset), int(p.SliceBetaOffset), intra, dst.ChromaFormatIDC)
		}
		if err := h264FilterMBEdgeHChroma(dst.Cb, dstCb+4*edge*dst.ChromaStride, dst.ChromaStride, bS, chromaQP[0], int(p.SliceAlphaC0Offset), int(p.SliceBetaOffset), intra); err != nil {
			return err
		}
		return h264FilterMBEdgeHChroma(dst.Cr, dstCr+4*edge*dst.ChromaStride, dst.ChromaStride, bS, chromaQP[1], int(p.SliceAlphaC0Offset), int(p.SliceBetaOffset), intra)
	case 3:
		if dir == 0 {
			if err := h264FilterMBEdgeVLuma(dst.Cb, dstCb+4*edge, dst.ChromaStride, bS, chromaQP[0], int(p.SliceAlphaC0Offset), int(p.SliceBetaOffset), intra); err != nil {
				return err
			}
			return h264FilterMBEdgeVLuma(dst.Cr, dstCr+4*edge, dst.ChromaStride, bS, chromaQP[1], int(p.SliceAlphaC0Offset), int(p.SliceBetaOffset), intra)
		}
		if err := h264FilterMBEdgeHLuma(dst.Cb, dstCb+4*edge*dst.ChromaStride, dst.ChromaStride, bS, chromaQP[0], int(p.SliceAlphaC0Offset), int(p.SliceBetaOffset), intra); err != nil {
			return err
		}
		return h264FilterMBEdgeHLuma(dst.Cr, dstCr+4*edge*dst.ChromaStride, dst.ChromaStride, bS, chromaQP[1], int(p.SliceAlphaC0Offset), int(p.SliceBetaOffset), intra)
	default:
		return ErrUnsupported
	}
}

func h264ApplyLoopFilterEdgeHigh(dst *h264PicturePlanesHigh, dstY int, dstCb int, dstCr int, dir int, edge int, bS [4]int16, qp int, chromaQP [2]int, p h264LoopFilterSliceParams, intra bool, filterLuma bool, filterChroma bool, bitDepth int) error {
	if dst == nil || edge < 0 || edge > 3 {
		return ErrInvalidData
	}
	if filterLuma {
		if dir == 0 {
			if err := h264FilterMBEdgeVLumaHigh(dst.Y, dstY+4*edge, dst.LumaStride, bS, qp, int(p.SliceAlphaC0Offset), int(p.SliceBetaOffset), intra, bitDepth); err != nil {
				return err
			}
		} else if err := h264FilterMBEdgeHLumaHigh(dst.Y, dstY+4*edge*dst.LumaStride, dst.LumaStride, bS, qp, int(p.SliceAlphaC0Offset), int(p.SliceBetaOffset), intra, bitDepth); err != nil {
			return err
		}
	}
	if !filterChroma || dst.ChromaFormatIDC == 0 {
		return nil
	}
	switch dst.ChromaFormatIDC {
	case 1:
		if edge&1 != 0 {
			return nil
		}
		if dir == 0 {
			if err := h264FilterMBEdgeVChromaHigh(dst.Cb, dstCb+2*edge, dst.ChromaStride, bS, chromaQP[0], int(p.SliceAlphaC0Offset), int(p.SliceBetaOffset), intra, dst.ChromaFormatIDC, bitDepth); err != nil {
				return err
			}
			return h264FilterMBEdgeVChromaHigh(dst.Cr, dstCr+2*edge, dst.ChromaStride, bS, chromaQP[1], int(p.SliceAlphaC0Offset), int(p.SliceBetaOffset), intra, dst.ChromaFormatIDC, bitDepth)
		}
		if err := h264FilterMBEdgeHChromaHigh(dst.Cb, dstCb+2*edge*dst.ChromaStride, dst.ChromaStride, bS, chromaQP[0], int(p.SliceAlphaC0Offset), int(p.SliceBetaOffset), intra, bitDepth); err != nil {
			return err
		}
		return h264FilterMBEdgeHChromaHigh(dst.Cr, dstCr+2*edge*dst.ChromaStride, dst.ChromaStride, bS, chromaQP[1], int(p.SliceAlphaC0Offset), int(p.SliceBetaOffset), intra, bitDepth)
	case 2:
		if dir == 0 {
			if edge&1 != 0 {
				return nil
			}
			if err := h264FilterMBEdgeVChromaHigh(dst.Cb, dstCb+2*edge, dst.ChromaStride, bS, chromaQP[0], int(p.SliceAlphaC0Offset), int(p.SliceBetaOffset), intra, dst.ChromaFormatIDC, bitDepth); err != nil {
				return err
			}
			return h264FilterMBEdgeVChromaHigh(dst.Cr, dstCr+2*edge, dst.ChromaStride, bS, chromaQP[1], int(p.SliceAlphaC0Offset), int(p.SliceBetaOffset), intra, dst.ChromaFormatIDC, bitDepth)
		}
		if err := h264FilterMBEdgeHChromaHigh(dst.Cb, dstCb+4*edge*dst.ChromaStride, dst.ChromaStride, bS, chromaQP[0], int(p.SliceAlphaC0Offset), int(p.SliceBetaOffset), intra, bitDepth); err != nil {
			return err
		}
		return h264FilterMBEdgeHChromaHigh(dst.Cr, dstCr+4*edge*dst.ChromaStride, dst.ChromaStride, bS, chromaQP[1], int(p.SliceAlphaC0Offset), int(p.SliceBetaOffset), intra, bitDepth)
	case 3:
		if dir == 0 {
			if err := h264FilterMBEdgeVLumaHigh(dst.Cb, dstCb+4*edge, dst.ChromaStride, bS, chromaQP[0], int(p.SliceAlphaC0Offset), int(p.SliceBetaOffset), intra, bitDepth); err != nil {
				return err
			}
			return h264FilterMBEdgeVLumaHigh(dst.Cr, dstCr+4*edge, dst.ChromaStride, bS, chromaQP[1], int(p.SliceAlphaC0Offset), int(p.SliceBetaOffset), intra, bitDepth)
		}
		if err := h264FilterMBEdgeHLumaHigh(dst.Cb, dstCb+4*edge*dst.ChromaStride, dst.ChromaStride, bS, chromaQP[0], int(p.SliceAlphaC0Offset), int(p.SliceBetaOffset), intra, bitDepth); err != nil {
			return err
		}
		return h264FilterMBEdgeHLumaHigh(dst.Cr, dstCr+4*edge*dst.ChromaStride, dst.ChromaStride, bS, chromaQP[1], int(p.SliceAlphaC0Offset), int(p.SliceBetaOffset), intra, bitDepth)
	default:
		return ErrUnsupported
	}
}

func h264FilterMBEdgeVLuma(pix []uint8, offset int, stride int, bS [4]int16, qp int, alphaOffset int, betaOffset int, intra bool) error {
	alpha, beta, indexA, err := h264LoopFilterThresholds(qp, alphaOffset, betaOffset)
	if err != nil || alpha == 0 || beta == 0 {
		return err
	}
	if bS[0] < 4 || !intra {
		tc, err := h264LoopFilterTC(indexA, bS, 0)
		if err != nil {
			return err
		}
		return h264HLoopFilterLuma(pix, offset, stride, alpha, beta, &tc)
	}
	return h264HLoopFilterLumaIntra(pix, offset, stride, alpha, beta)
}

func h264FilterMBEdgeVLumaHigh(pix []uint16, offset int, stride int, bS [4]int16, qp int, alphaOffset int, betaOffset int, intra bool, bitDepth int) error {
	alpha, beta, indexA, err := h264LoopFilterThresholdsForBitDepth(qp, alphaOffset, betaOffset, bitDepth)
	if err != nil || alpha == 0 || beta == 0 {
		return err
	}
	if bS[0] < 4 || !intra {
		tc, err := h264LoopFilterTC(indexA, bS, 0)
		if err != nil {
			return err
		}
		return h264HLoopFilterLumaHigh(pix, offset, stride, alpha, beta, &tc, bitDepth)
	}
	return h264HLoopFilterLumaIntraHigh(pix, offset, stride, alpha, beta, bitDepth)
}

func h264FilterMBEdgeHLuma(pix []uint8, offset int, stride int, bS [4]int16, qp int, alphaOffset int, betaOffset int, intra bool) error {
	alpha, beta, indexA, err := h264LoopFilterThresholds(qp, alphaOffset, betaOffset)
	if err != nil || alpha == 0 || beta == 0 {
		return err
	}
	if bS[0] < 4 || !intra {
		tc, err := h264LoopFilterTC(indexA, bS, 0)
		if err != nil {
			return err
		}
		return h264VLoopFilterLuma(pix, offset, stride, alpha, beta, &tc)
	}
	return h264VLoopFilterLumaIntra(pix, offset, stride, alpha, beta)
}

func h264FilterMBEdgeHLumaHigh(pix []uint16, offset int, stride int, bS [4]int16, qp int, alphaOffset int, betaOffset int, intra bool, bitDepth int) error {
	alpha, beta, indexA, err := h264LoopFilterThresholdsForBitDepth(qp, alphaOffset, betaOffset, bitDepth)
	if err != nil || alpha == 0 || beta == 0 {
		return err
	}
	if bS[0] < 4 || !intra {
		tc, err := h264LoopFilterTC(indexA, bS, 0)
		if err != nil {
			return err
		}
		return h264VLoopFilterLumaHigh(pix, offset, stride, alpha, beta, &tc, bitDepth)
	}
	return h264VLoopFilterLumaIntraHigh(pix, offset, stride, alpha, beta, bitDepth)
}

func h264FilterMBEdgeVChroma(pix []uint8, offset int, stride int, bS [4]int16, qp int, alphaOffset int, betaOffset int, intra bool, chromaFormatIDC int) error {
	alpha, beta, indexA, err := h264LoopFilterThresholds(qp, alphaOffset, betaOffset)
	if err != nil || alpha == 0 || beta == 0 {
		return err
	}
	if bS[0] < 4 || !intra {
		tc, err := h264LoopFilterTC(indexA, bS, 1)
		if err != nil {
			return err
		}
		if chromaFormatIDC == 2 {
			return h264HLoopFilterChroma422(pix, offset, stride, alpha, beta, &tc)
		}
		return h264HLoopFilterChroma(pix, offset, stride, alpha, beta, &tc)
	}
	if chromaFormatIDC == 2 {
		return h264HLoopFilterChroma422Intra(pix, offset, stride, alpha, beta)
	}
	return h264HLoopFilterChromaIntra(pix, offset, stride, alpha, beta)
}

func h264FilterMBEdgeVChromaHigh(pix []uint16, offset int, stride int, bS [4]int16, qp int, alphaOffset int, betaOffset int, intra bool, chromaFormatIDC int, bitDepth int) error {
	alpha, beta, indexA, err := h264LoopFilterThresholdsForBitDepth(qp, alphaOffset, betaOffset, bitDepth)
	if err != nil || alpha == 0 || beta == 0 {
		return err
	}
	if bS[0] < 4 || !intra {
		tc, err := h264LoopFilterTC(indexA, bS, 1)
		if err != nil {
			return err
		}
		if chromaFormatIDC == 2 {
			return h264HLoopFilterChroma422High(pix, offset, stride, alpha, beta, &tc, bitDepth)
		}
		return h264HLoopFilterChromaHigh(pix, offset, stride, alpha, beta, &tc, bitDepth)
	}
	if chromaFormatIDC == 2 {
		return h264HLoopFilterChroma422IntraHigh(pix, offset, stride, alpha, beta, bitDepth)
	}
	return h264HLoopFilterChromaIntraHigh(pix, offset, stride, alpha, beta, bitDepth)
}

func h264FilterMBEdgeHChroma(pix []uint8, offset int, stride int, bS [4]int16, qp int, alphaOffset int, betaOffset int, intra bool) error {
	alpha, beta, indexA, err := h264LoopFilterThresholds(qp, alphaOffset, betaOffset)
	if err != nil || alpha == 0 || beta == 0 {
		return err
	}
	if bS[0] < 4 || !intra {
		tc, err := h264LoopFilterTC(indexA, bS, 1)
		if err != nil {
			return err
		}
		return h264VLoopFilterChroma(pix, offset, stride, alpha, beta, &tc)
	}
	return h264VLoopFilterChromaIntra(pix, offset, stride, alpha, beta)
}

func h264FilterMBEdgeHChromaHigh(pix []uint16, offset int, stride int, bS [4]int16, qp int, alphaOffset int, betaOffset int, intra bool, bitDepth int) error {
	alpha, beta, indexA, err := h264LoopFilterThresholdsForBitDepth(qp, alphaOffset, betaOffset, bitDepth)
	if err != nil || alpha == 0 || beta == 0 {
		return err
	}
	if bS[0] < 4 || !intra {
		tc, err := h264LoopFilterTC(indexA, bS, 1)
		if err != nil {
			return err
		}
		return h264VLoopFilterChromaHigh(pix, offset, stride, alpha, beta, &tc, bitDepth)
	}
	return h264VLoopFilterChromaIntraHigh(pix, offset, stride, alpha, beta, bitDepth)
}

func h264LoopFilterThresholds(qp int, alphaOffset int, betaOffset int) (int, int, int, error) {
	return h264LoopFilterThresholdsForBitDepth(qp, alphaOffset, betaOffset, 8)
}

func h264LoopFilterThresholdsForBitDepth(qp int, alphaOffset int, betaOffset int, bitDepth int) (int, int, int, error) {
	if err := checkH264LoopFilterBitDepth(bitDepth); err != nil {
		return 0, 0, 0, err
	}
	maxQP := h264MaxQPForBitDepth(bitDepth)
	if qp < 0 || qp > maxQP {
		return 0, 0, 0, ErrInvalidData
	}
	qpBDOffset := 6 * (bitDepth - 8)
	indexA := clipInt(qp-qpBDOffset+alphaOffset, 0, 51)
	indexB := clipInt(qp-qpBDOffset+betaOffset, 0, 51)
	return int(h264LoopFilterAlphaTable[indexA]), int(h264LoopFilterBetaTable[indexB]), indexA, nil
}

func checkH264LoopFilterBitDepth(bitDepth int) error {
	switch bitDepth {
	case 8, 9, 10, 12, 14:
		return nil
	default:
		return ErrUnsupported
	}
}

func h264LoopFilterTC(indexA int, bS [4]int16, plus int8) ([4]int8, error) {
	var tc [4]int8
	if indexA < 0 || indexA >= len(h264LoopFilterTC0Table) {
		return tc, ErrInvalidData
	}
	for i := 0; i < 4; i++ {
		if bS[i] < 0 || bS[i] > 3 {
			return tc, ErrInvalidData
		}
		tc[i] = h264LoopFilterTC0Table[indexA][bS[i]] + plus
	}
	return tc, nil
}
