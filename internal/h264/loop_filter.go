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

var h264LoopFilterMaskEdgeTable = [2][8]int{
	{0, 3, 3, 3, 1, 1, 1, 1},
	{0, 3, 1, 1, 3, 3, 3, 3},
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
	PictureStructure     int32
	DeblockingFilter     int32
	SliceAlphaC0Offset   int32
	SliceBetaOffset      int32
	ChromaQPIndexOffset0 int32
	ChromaQPIndexOffset1 int32
	Ref2Frame            [2][]int8
}

type h264LoopFilterContext struct {
	MBXY              int
	TopMBXY           int
	LeftMBXY          int
	LeftMBXYs         [2]int
	TopType           uint32
	LeftType          uint32
	LeftTypes         [2]uint32
	CBP               int
	NonZeroCountCache [h264MotionCacheSize]uint8
	Motion            h264LoopFilterMotionCache
}

// The deblock strength calculation consumes only motion vectors and reference
// ids. Keeping its per-macroblock cache separate avoids zeroing the unrelated
// MVD and direct-prediction arrays carried by the decode-time motion cache.
type h264LoopFilterMotionCache struct {
	MV  [2][h264MotionCacheSize][2]int16
	Ref [2][h264MotionCacheSize]int8
}

func h264LoopFilterSliceParamsFromHeader(sh *SliceHeader) h264LoopFilterSliceParams {
	if sh == nil || sh.PPS == nil {
		return h264LoopFilterSliceParams{}
	}
	return h264LoopFilterSliceParams{
		PPS:                  sh.PPS,
		CABAC:                sh.PPS.CABAC != 0,
		ListCount:            int(sh.ListCount),
		PictureStructure:     sh.PictureStructure,
		DeblockingFilter:     sh.DeblockingFilter,
		SliceAlphaC0Offset:   sh.SliceAlphaC0Offset,
		SliceBetaOffset:      sh.SliceBetaOffset,
		ChromaQPIndexOffset0: sh.PPS.ChromaQPIndexOffset[0],
		ChromaQPIndexOffset1: sh.PPS.ChromaQPIndexOffset[1],
	}
}

func (p *h264LoopFilterSliceParams) validate() error {
	if p.DeblockingFilter < 0 || p.DeblockingFilter > 2 || p.ListCount < 0 || p.ListCount > 2 {
		return ErrInvalidData
	}
	if p.DeblockingFilter == 0 {
		return nil
	}
	if p.PPS == nil || p.PPS.SPS == nil {
		return ErrInvalidData
	}
	if p.PPS.SPS.FrameMBSOnlyFlag == 0 {
		if p.PictureStructure != PictureFrame && p.PictureStructure != PictureTopField && p.PictureStructure != PictureBottomField {
			return ErrUnsupported
		}
	}
	if err := checkH264LoopFilterBitDepth(int(p.PPS.SPS.BitDepthLuma)); err != nil {
		return err
	}
	if p.PPS.SPS.BitDepthLuma != p.PPS.SPS.BitDepthChroma {
		return ErrUnsupported
	}
	return nil
}

func (p *h264LoopFilterSliceParams) fieldPicture() bool {
	return p.PPS != nil && p.PPS.SPS != nil && p.PPS.SPS.FrameMBSOnlyFlag == 0 &&
		p.PictureStructure != PictureFrame
}

func h264LoopFilterFieldMBY(mbY int, p *h264LoopFilterSliceParams) (int, error) {
	if !p.fieldPicture() {
		return mbY, nil
	}
	if mbY < 0 {
		return 0, ErrInvalidData
	}
	if p.PictureStructure == PictureTopField {
		if mbY&1 != 0 {
			return 0, ErrInvalidData
		}
		return mbY >> 1, nil
	}
	if p.PictureStructure == PictureBottomField {
		if mbY&1 == 0 {
			return 0, ErrInvalidData
		}
		return mbY >> 1, nil
	}
	return 0, ErrInvalidData
}

func (p *h264LoopFilterSliceParams) ref2Frame(list int, ref int8) int8 {
	if ref == h264ListNotUsed || list < 0 || list > 1 {
		return ref
	}
	refs := p.Ref2Frame[list]
	if len(refs) == 0 || ref < 0 || int(ref) >= len(refs) {
		return ref
	}
	return refs[ref]
}

func (p *h264LoopFilterSliceParams) ref2FrameForInterlacedLoopFilter(list int, ref int8) int8 {
	if ref < 0 || list < 0 || list > 1 {
		return p.ref2Frame(list, ref)
	}
	refs := p.Ref2Frame[list]
	frameRef := int(ref) >> 1
	if frameRef < 0 || frameRef >= len(refs) || refs[frameRef] < 0 {
		return p.ref2Frame(list, ref)
	}
	structure := int8(PictureTopField)
	if ref&1 != 0 {
		structure = int8(PictureBottomField)
	}
	return (refs[frameRef] &^ 3) | structure
}

func h264LoopFilterRef2Frame(entries [2][]simpleRefEntry, ids map[*DecodedFrame]int8) ([2][]int8, error) {
	return h264LoopFilterRef2FrameInto(entries, ids, [2][]int8{})
}

func h264LoopFilterRef2FrameInto(entries [2][]simpleRefEntry, ids map[*DecodedFrame]int8, dst [2][]int8) ([2][]int8, error) {
	var out [2][]int8
	for list := 0; list < 2; list++ {
		if len(entries[list]) == 0 {
			out[list] = dst[list][:0]
			continue
		}
		if len(entries[list]) > maxInt/32 {
			return [2][]int8{}, ErrInvalidData
		}
		if cap(dst[list]) < len(entries[list]) {
			out[list] = make([]int8, len(entries[list]))
		} else {
			out[list] = dst[list][:len(entries[list])]
		}
		for i, entry := range entries[list] {
			id, err := h264LoopFilterRefFrameID(entry.frame, ids)
			if err != nil {
				return [2][]int8{}, err
			}
			refStructure := entry.pictureStructure
			if refStructure == 0 {
				refStructure = PictureFrame
			}
			if refStructure != PictureTopField && refStructure != PictureBottomField && refStructure != PictureFrame {
				return [2][]int8{}, ErrInvalidData
			}
			ref := int(id)*4 + int(refStructure&PictureFrame)
			if ref > 127 {
				return [2][]int8{}, ErrUnsupported
			}
			out[list][i] = int8(ref)
		}
	}
	return out, nil
}

func h264LoopFilterRefFrameID(frame *DecodedFrame, ids map[*DecodedFrame]int8) (int8, error) {
	if frame == nil || ids == nil {
		return 0, ErrInvalidData
	}
	if id, ok := ids[frame]; ok {
		return id, nil
	}
	if len(ids) >= 127 {
		return 0, ErrUnsupported
	}
	id := int8(len(ids))
	ids[frame] = id
	return id, nil
}

func (m *macroblockTables) loopFilterParamsForMB(params []h264LoopFilterSliceParams, mbXY int, fallback *h264LoopFilterSliceParams) *h264LoopFilterSliceParams {
	if m == nil || mbXY < 0 || mbXY >= len(m.SliceTable) {
		return fallback
	}
	sliceNum := m.SliceTable[mbXY]
	if sliceNum == ^uint16(0) || int(sliceNum) >= len(params) {
		return fallback
	}
	return &params[sliceNum]
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
	if len(params) <= 256 {
		var used [256]bool
		progressive := true
		for mbY := 0; mbY < m.MBHeight; mbY++ {
			for mbX := 0; mbX < m.MBWidth; mbX++ {
				mbXY := mbX + mbY*m.MBStride
				sliceNum := m.SliceTable[mbXY]
				if sliceNum == ^uint16(0) || int(sliceNum) >= len(params) {
					return ErrInvalidData
				}
				used[sliceNum] = true
			}
		}
		for i := range params {
			if !used[i] {
				continue
			}
			p := &params[i]
			if err := p.validate(); err != nil {
				return err
			}
			if p.fieldPicture() || h264LoopFilterFrameMBAFF(p) {
				progressive = false
			}
		}
		if progressive {
			return m.filterFrameProgressiveValidated(dst, params)
		}
	}
	if h264LoopFilterParamsUseFrameMBAFF(params) {
		for mbPairY := 0; mbPairY < m.MBHeight; mbPairY += 2 {
			for mbX := 0; mbX < m.MBWidth; mbX++ {
				for mbY := mbPairY; mbY <= mbPairY+1 && mbY < m.MBHeight; mbY++ {
					if err := m.filterFrameMBAt(dst, params, mbX, mbY); err != nil {
						return err
					}
				}
			}
		}
		return nil
	}
	for mbY := 0; mbY < m.MBHeight; mbY++ {
		for mbX := 0; mbX < m.MBWidth; mbX++ {
			if err := m.filterFrameMBAt(dst, params, mbX, mbY); err != nil {
				return err
			}
		}
	}
	return nil
}

// filterFrameProgressiveValidated is the ordinary frame-picture path after
// frame dimensions, slice references, and slice parameters have been checked.
// It avoids repeating field/MBAFF view setup and destination bounds work for
// every macroblock while preserving the general path for those picture modes.
func (m *macroblockTables) filterFrameProgressiveValidated(dst *h264PicturePlanes, params []h264LoopFilterSliceParams) error {
	var ctx h264LoopFilterContext
	for mbY := 0; mbY < m.MBHeight; mbY++ {
		dstY := mbY * 16 * dst.LumaStride
		dstC := 0
		switch dst.ChromaFormatIDC {
		case 1:
			dstC = mbY * 8 * dst.ChromaStride
		case 2, 3:
			dstC = mbY * 16 * dst.ChromaStride
		}
		for mbX := 0; mbX < m.MBWidth; mbX++ {
			mbXY := mbX + mbY*m.MBStride
			sliceNum := m.SliceTable[mbXY]
			p := &params[sliceNum]
			if p.DeblockingFilter == 0 {
				continue
			}
			ctx = h264LoopFilterContext{}
			if dst.ChromaFormatIDC == 1 && p.CABAC {
				m.fillLoopFilterCachesProgressive420CABACInto(&ctx, mbX, mbY, mbXY, sliceNum, p, params)
			} else if err := m.fillLoopFilterCachesFrameValidatedInto(&ctx, mbXY, sliceNum, p, params); err != nil {
				return err
			}
			y := dstY + mbX*16
			c := dstC
			if dst.ChromaFormatIDC <= 2 {
				c += mbX * 8
			} else {
				c += mbX * 16
			}
			if dst.ChromaFormatIDC == 1 {
				if err := m.filterFrameMacroblockDirProgressive420(dst, y, c, c, p, &ctx, 0); err != nil {
					return err
				}
				if err := m.filterFrameMacroblockDirProgressive420(dst, y, c, c, p, &ctx, 1); err != nil {
					return err
				}
			} else {
				if err := m.filterFrameMacroblockDir(dst, y, c, c, p, &ctx, 0); err != nil {
					return err
				}
				if err := m.filterFrameMacroblockDir(dst, y, c, c, p, &ctx, 1); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// fillLoopFilterCachesProgressive420CABACInto is the ordinary 8-bit CABAC
// cache builder after filterFrame has validated every macroblock, slice, and
// picture parameter and excluded field and MBAFF pictures. Keeping this path
// separate from the general builder removes per-block geometry and range
// checks without changing the public loop-filter contract.
func (m *macroblockTables) fillLoopFilterCachesProgressive420CABACInto(ctx *h264LoopFilterContext, mbX int, mbY int, mbXY int, sliceNum uint16, p *h264LoopFilterSliceParams, params []h264LoopFilterSliceParams) {
	topXY := -1
	leftXY := -1
	if mbY > 0 {
		topXY = mbXY - m.MBStride
	}
	if mbX > 0 {
		leftXY = mbXY - 1
	}

	mbType := m.MacroblockTyp[mbXY]
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

	ctx.MBXY = mbXY
	ctx.TopMBXY = topXY
	ctx.LeftMBXY = leftXY
	ctx.LeftMBXYs = [2]int{leftXY, leftXY}
	ctx.TopType = topType
	ctx.LeftType = leftType
	ctx.LeftTypes = [2]uint32{leftType, leftType}
	ctx.CBP = m.CBPTable[mbXY]
	if isIntra(mbType) {
		return
	}

	base := int(h264Scan8[0])
	for list := 0; list < p.ListCount; list++ {
		if usesList(topType, list) {
			topP := &params[m.SliceTable[topXY]]
			src := int(m.MB2BXY[topXY]) + 3*m.BStride
			copy(ctx.Motion.MV[list][base-8:base-4], m.MotionVal[list][src:src+4])
			refBase := 4*topXY + 2
			ref0 := topP.ref2Frame(list, m.RefIndex[list][refBase])
			ref1 := topP.ref2Frame(list, m.RefIndex[list][refBase+1])
			ctx.Motion.Ref[list][base-8] = ref0
			ctx.Motion.Ref[list][base-7] = ref0
			ctx.Motion.Ref[list][base-6] = ref1
			ctx.Motion.Ref[list][base-5] = ref1
		} else {
			clearMotionRow(&ctx.Motion.MV[list], base-8, 4)
			fillRefRow(&ctx.Motion.Ref[list], base-8, 4, h264ListNotUsed)
		}

		if usesList(leftType, list) {
			leftP := &params[m.SliceTable[leftXY]]
			bXY := int(m.MB2BXY[leftXY]) + 3
			refBase := 4*leftXY + 1
			refTop := leftP.ref2Frame(list, m.RefIndex[list][refBase])
			refBottom := leftP.ref2Frame(list, m.RefIndex[list][refBase+2])
			for row := 0; row < 4; row++ {
				cacheIdx := base - 1 + row*8
				ctx.Motion.MV[list][cacheIdx] = m.MotionVal[list][bXY+row*m.BStride]
				if row < 2 {
					ctx.Motion.Ref[list][cacheIdx] = refTop
				} else {
					ctx.Motion.Ref[list][cacheIdx] = refBottom
				}
			}
		} else {
			for row := 0; row < 4; row++ {
				cacheIdx := base - 1 + row*8
				ctx.Motion.MV[list][cacheIdx] = [2]int16{}
				ctx.Motion.Ref[list][cacheIdx] = h264ListNotUsed
			}
		}

		if !usesList(mbType, list) {
			fillMotionRectangle(&ctx.Motion.MV[list], base, 4, 4, 8, [2]int16{})
			fillRefRectangle(&ctx.Motion.Ref[list], base, 4, 4, 8, h264ListNotUsed)
			continue
		}

		src := 4*mbX + 4*mbY*m.BStride
		for row := 0; row < 4; row++ {
			copy(ctx.Motion.MV[list][base+row*8:base+row*8+4], m.MotionVal[list][src+row*m.BStride:src+row*m.BStride+4])
		}
		refBase := 4 * mbXY
		ref0 := p.ref2Frame(list, m.RefIndex[list][refBase])
		ref1 := p.ref2Frame(list, m.RefIndex[list][refBase+1])
		ref2 := p.ref2Frame(list, m.RefIndex[list][refBase+2])
		ref3 := p.ref2Frame(list, m.RefIndex[list][refBase+3])
		for row := 0; row < 2; row++ {
			rowBase := base + row*8
			ctx.Motion.Ref[list][rowBase] = ref0
			ctx.Motion.Ref[list][rowBase+1] = ref0
			ctx.Motion.Ref[list][rowBase+2] = ref1
			ctx.Motion.Ref[list][rowBase+3] = ref1
		}
		for row := 2; row < 4; row++ {
			rowBase := base + row*8
			ctx.Motion.Ref[list][rowBase] = ref2
			ctx.Motion.Ref[list][rowBase+1] = ref2
			ctx.Motion.Ref[list][rowBase+2] = ref3
			ctx.Motion.Ref[list][rowBase+3] = ref3
		}
	}

	nnz := m.NonZeroCount[mbXY]
	copy(ctx.NonZeroCountCache[4+8:4+8+4], nnz[0:4])
	copy(ctx.NonZeroCountCache[4+16:4+16+4], nnz[4:8])
	copy(ctx.NonZeroCountCache[4+24:4+24+4], nnz[8:12])
	copy(ctx.NonZeroCountCache[4+32:4+32+4], nnz[12:16])
	if topType != 0 {
		copy(ctx.NonZeroCountCache[4:8], m.NonZeroCount[topXY][12:16])
	}
	if leftType != 0 {
		leftNNZ := m.NonZeroCount[leftXY]
		ctx.NonZeroCountCache[3+8] = leftNNZ[3]
		ctx.NonZeroCountCache[3+16] = leftNNZ[7]
		ctx.NonZeroCountCache[3+24] = leftNNZ[11]
		ctx.NonZeroCountCache[3+32] = leftNNZ[15]
	}
}

func (m *macroblockTables) filterFrameMacroblockDirProgressive420(dst *h264PicturePlanes, dstY int, dstCb int, dstCr int, p *h264LoopFilterSliceParams, ctx *h264LoopFilterContext, dir int) error {
	mbType := m.MacroblockTyp[ctx.MBXY]
	mbmXY := ctx.LeftMBXY
	mbmType := ctx.LeftType
	if dir != 0 {
		mbmXY = ctx.TopMBXY
		mbmType = ctx.TopType
	}

	maskEdge := h264LoopFilterMaskEdgeTable[dir][(mbType>>3)&7]
	edges := 4
	if maskEdge == 3 && ctx.CBP&15 == 0 {
		edges = 1
	}
	maskPar0 := mbType & (MBType16x16 | (MBType8x16 >> uint(dir)))
	mvyLimit := h264LoopFilterMVYLimit(mbType)

	if mbmType != 0 {
		bS, err := m.loopFilterBoundaryStrength(ctx, mbType, mbmType, dir, maskPar0, p.ListCount, mvyLimit, false)
		if err != nil {
			return err
		}
		if h264LoopFilterBSSum(bS) != 0 {
			qp := (int(m.QScaleTable[ctx.MBXY]) + int(m.QScaleTable[mbmXY]) + 1) >> 1
			chromaQP := [2]int{
				(int(p.PPS.ChromaQPTable[0][m.QScaleTable[ctx.MBXY]]) + int(p.PPS.ChromaQPTable[0][m.QScaleTable[mbmXY]]) + 1) >> 1,
				(int(p.PPS.ChromaQPTable[1][m.QScaleTable[ctx.MBXY]]) + int(p.PPS.ChromaQPTable[1][m.QScaleTable[mbmXY]]) + 1) >> 1,
			}
			if err := h264ApplyLoopFilterEdgeProgressive420(dst, dstY, dstCb, dstCr, dir, 0, bS, qp, chromaQP, p, true, true, true); err != nil {
				return err
			}
		}
	}

	for edge := 1; edge < edges; edge++ {
		deblockEdge := !is8x8DCT(mbType & (uint32(edge) << 24))
		if !deblockEdge && dir == 0 {
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
		filterChroma := edge&1 == 0
		if err := h264ApplyLoopFilterEdgeProgressive420(dst, dstY, dstCb, dstCr, dir, edge, bS, qp, chromaQP, p, false, deblockEdge, filterChroma); err != nil {
			return err
		}
	}
	return nil
}

func h264ApplyLoopFilterEdgeProgressive420(dst *h264PicturePlanes, dstY int, dstCb int, dstCr int, dir int, edge int, bS [4]int16, qp int, chromaQP [2]int, p *h264LoopFilterSliceParams, intra bool, filterLuma bool, filterChroma bool) error {
	alphaOffset := int(p.SliceAlphaC0Offset)
	betaOffset := int(p.SliceBetaOffset)
	if filterLuma {
		if dir == 0 {
			if err := h264FilterMBEdgeVLuma(dst.Y, dstY+4*edge, dst.LumaStride, bS, qp, alphaOffset, betaOffset, intra); err != nil {
				return err
			}
		} else if err := h264FilterMBEdgeHLuma(dst.Y, dstY+4*edge*dst.LumaStride, dst.LumaStride, bS, qp, alphaOffset, betaOffset, intra); err != nil {
			return err
		}
	}
	if !filterChroma {
		return nil
	}
	if dir == 0 {
		if err := h264FilterMBEdgeVChroma(dst.Cb, dstCb+2*edge, dst.ChromaStride, bS, chromaQP[0], alphaOffset, betaOffset, intra, 1); err != nil {
			return err
		}
		return h264FilterMBEdgeVChroma(dst.Cr, dstCr+2*edge, dst.ChromaStride, bS, chromaQP[1], alphaOffset, betaOffset, intra, 1)
	}
	if err := h264FilterMBEdgeHChroma(dst.Cb, dstCb+2*edge*dst.ChromaStride, dst.ChromaStride, bS, chromaQP[0], alphaOffset, betaOffset, intra); err != nil {
		return err
	}
	return h264FilterMBEdgeHChroma(dst.Cr, dstCr+2*edge*dst.ChromaStride, dst.ChromaStride, bS, chromaQP[1], alphaOffset, betaOffset, intra)
}

func h264LoopFilterParamsUseFrameMBAFF(params []h264LoopFilterSliceParams) bool {
	for i := range params {
		if h264LoopFilterFrameMBAFF(&params[i]) {
			return true
		}
	}
	return false
}

func (m *macroblockTables) filterFrameMBAt(dst *h264PicturePlanes, params []h264LoopFilterSliceParams, mbX int, mbY int) error {
	mbXY := mbX + mbY*m.MBStride
	sliceNum := m.SliceTable[mbXY]
	if sliceNum == ^uint16(0) || int(sliceNum) >= len(params) {
		return ErrInvalidData
	}
	p := &params[sliceNum]
	if err := p.validate(); err != nil {
		return err
	}
	if p.DeblockingFilter == 0 {
		return nil
	}
	var ctx h264LoopFilterContext
	if err := m.fillLoopFilterCachesFrameValidatedInto(&ctx, mbXY, sliceNum, p, params); err != nil {
		return err
	}
	dstView := *dst
	filterMBY := mbY
	var err error
	if p.fieldPicture() {
		filterMBY, err = h264LoopFilterFieldMBY(mbY, p)
		if err != nil {
			return err
		}
		applySimpleFieldRefPlane(&dstView, p.PictureStructure)
	} else if h264LoopFilterFrameMBAFF(p) {
		dstView, filterMBY, err = h264FrameMBAFFLoopFilterView(dst, mbY, m.MacroblockTyp[mbXY])
		if err != nil {
			return err
		}
	}
	return m.filterFrameMacroblock(&dstView, mbX, filterMBY, p, &ctx)
}

func (m *macroblockTables) filterField(dst *h264PicturePlanes, params []h264LoopFilterSliceParams, pictureStructure int32) error {
	if m == nil || dst == nil {
		return ErrInvalidData
	}
	if pictureStructure != PictureTopField && pictureStructure != PictureBottomField {
		return ErrInvalidData
	}
	if err := dst.validate(); err != nil {
		return err
	}
	if m.MBWidth != dst.MBWidth || m.MBHeight != dst.MBHeight || m.ChromaFormatIDC != dst.ChromaFormatIDC {
		return ErrInvalidData
	}
	wantOddRow := pictureStructure == PictureBottomField
	for mbY := 0; mbY < m.MBHeight; mbY++ {
		if (mbY&1 != 0) != wantOddRow {
			continue
		}
		for mbX := 0; mbX < m.MBWidth; mbX++ {
			mbXY := mbX + mbY*m.MBStride
			sliceNum := m.SliceTable[mbXY]
			if sliceNum == ^uint16(0) || int(sliceNum) >= len(params) {
				return ErrInvalidData
			}
			p := &params[sliceNum]
			if p.PictureStructure != pictureStructure {
				return ErrInvalidData
			}
			if err := p.validate(); err != nil {
				return err
			}
			if p.DeblockingFilter == 0 {
				continue
			}
			var ctx h264LoopFilterContext
			if err := m.fillLoopFilterCachesFrameValidatedInto(&ctx, mbXY, sliceNum, p, params); err != nil {
				return err
			}
			dstView := *dst
			filterMBY, err := h264LoopFilterFieldMBY(mbY, p)
			if err != nil {
				return err
			}
			applySimpleFieldRefPlane(&dstView, p.PictureStructure)
			if err := m.filterFrameMacroblock(&dstView, mbX, filterMBY, p, &ctx); err != nil {
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
	if h264LoopFilterParamsUseFrameMBAFF(params) {
		for mbPairY := 0; mbPairY < m.MBHeight; mbPairY += 2 {
			for mbX := 0; mbX < m.MBWidth; mbX++ {
				for mbY := mbPairY; mbY <= mbPairY+1 && mbY < m.MBHeight; mbY++ {
					if err := m.filterFrameHighMBAt(dst, params, mbX, mbY); err != nil {
						return err
					}
				}
			}
		}
		return nil
	}
	for mbY := 0; mbY < m.MBHeight; mbY++ {
		for mbX := 0; mbX < m.MBWidth; mbX++ {
			if err := m.filterFrameHighMBAt(dst, params, mbX, mbY); err != nil {
				return err
			}
		}
	}
	return nil
}

func (m *macroblockTables) filterFrameHighMBAt(dst *h264PicturePlanesHigh, params []h264LoopFilterSliceParams, mbX int, mbY int) error {
	mbXY := mbX + mbY*m.MBStride
	sliceNum := m.SliceTable[mbXY]
	if sliceNum == ^uint16(0) || int(sliceNum) >= len(params) {
		return ErrInvalidData
	}
	p := &params[sliceNum]
	if err := p.validate(); err != nil {
		return err
	}
	if p.DeblockingFilter == 0 {
		return nil
	}
	var ctx h264LoopFilterContext
	if err := m.fillLoopFilterCachesFrameValidatedInto(&ctx, mbXY, sliceNum, p, params); err != nil {
		return err
	}
	dstView := *dst
	filterMBY := mbY
	var err error
	if p.fieldPicture() {
		filterMBY, err = h264LoopFilterFieldMBY(mbY, p)
		if err != nil {
			return err
		}
		applySimpleFieldRefPlaneHigh(&dstView, p.PictureStructure)
	} else if h264LoopFilterFrameMBAFF(p) {
		dstView, filterMBY, err = h264FrameMBAFFLoopFilterViewHigh(dst, mbY, m.MacroblockTyp[mbXY])
		if err != nil {
			return err
		}
	}
	return m.filterFrameMacroblockHigh(&dstView, mbX, filterMBY, *p, &ctx)
}

func (m *macroblockTables) fillLoopFilterCachesFrame(mbXY int, sliceNum uint16, p h264LoopFilterSliceParams, params []h264LoopFilterSliceParams) (h264LoopFilterContext, error) {
	var ctx h264LoopFilterContext
	if err := m.checkCodedMBXY(mbXY); err != nil {
		return ctx, err
	}
	if err := p.validate(); err != nil {
		return ctx, err
	}
	return m.fillLoopFilterCachesFrameValidated(mbXY, sliceNum, &p, params)
}

func (m *macroblockTables) fillLoopFilterCachesFrameValidated(mbXY int, sliceNum uint16, p *h264LoopFilterSliceParams, params []h264LoopFilterSliceParams) (h264LoopFilterContext, error) {
	var ctx h264LoopFilterContext
	err := m.fillLoopFilterCachesFrameValidatedInto(&ctx, mbXY, sliceNum, p, params)
	return ctx, err
}

func (m *macroblockTables) fillLoopFilterCachesFrameValidatedInto(ctx *h264LoopFilterContext, mbXY int, sliceNum uint16, p *h264LoopFilterSliceParams, params []h264LoopFilterSliceParams) error {
	mbX := mbXY % m.MBStride
	mbY := mbXY / m.MBStride
	mbType := m.MacroblockTyp[mbXY]
	mbField := mbType&MBTypeInterlaced != 0
	frameMBAFF := h264LoopFilterFrameMBAFF(p)
	topXY := -1
	leftXY := [2]int{-1, -1}
	topStride := m.MBStride
	topRows := 1
	if p.fieldPicture() || (frameMBAFF && mbField) {
		topStride <<= 1
		topRows = 2
	}
	if mbY >= topRows {
		topXY = mbXY - topStride
	}
	if mbX > 0 {
		leftXY[h264LeftTop] = mbXY - 1
		leftXY[h264LeftBot] = mbXY - 1
	}
	if frameMBAFF {
		leftField := mbX > 0 && m.macroblockTypeIfCoded(mbXY-1)&MBTypeInterlaced != 0
		if mbY&1 != 0 {
			if leftField != mbField {
				leftXY[h264LeftTop] -= m.MBStride
			}
		} else if mbField {
			if topXY >= 0 && m.macroblockTypeIfCoded(topXY)&MBTypeInterlaced == 0 {
				topXY += m.MBStride
			}
			if leftField != mbField {
				leftXY[h264LeftBot] += m.MBStride
			}
		} else if leftField != mbField {
			leftXY[h264LeftBot] += m.MBStride
		}
		if mbX == 0 {
			leftXY = [2]int{-1, -1}
		}
	}

	topType := m.macroblockTypeIfCoded(topXY)
	leftType := [2]uint32{
		m.macroblockTypeIfCoded(leftXY[h264LeftTop]),
		m.macroblockTypeIfCoded(leftXY[h264LeftBot]),
	}
	if p.DeblockingFilter == 2 {
		if !m.sameSlice(topXY, sliceNum) {
			topType = 0
		}
		if !m.sameSlice(leftXY[h264LeftBot], sliceNum) {
			leftType = [2]uint32{}
		}
	}

	ctx.MBXY = mbXY
	ctx.TopMBXY = topXY
	ctx.LeftMBXY = leftXY[h264LeftTop]
	ctx.LeftMBXYs = leftXY
	ctx.TopType = topType
	ctx.LeftType = leftType[h264LeftTop]
	ctx.LeftTypes = leftType
	ctx.CBP = m.CBPTable[mbXY]
	if isIntra(mbType) {
		return nil
	}

	if err := m.fillLoopFilterCachesInterFramePtr(ctx, mbXY, topXY, leftXY[h264LeftTop], mbType, topType, leftType[h264LeftTop], 0, p, params); err != nil {
		return err
	}
	if p.ListCount == 2 {
		if err := m.fillLoopFilterCachesInterFramePtr(ctx, mbXY, topXY, leftXY[h264LeftTop], mbType, topType, leftType[h264LeftTop], 1, p, params); err != nil {
			return err
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
	if leftType[h264LeftTop] != 0 {
		leftNNZ := m.NonZeroCount[leftXY[h264LeftTop]]
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
		if is8x8DCT(leftType[h264LeftTop]) {
			ctx.NonZeroCountCache[3+8*1] = uint8((m.CBPTable[leftXY[h264LeftTop]] & 0x2000) >> 12)
			ctx.NonZeroCountCache[3+8*2] = ctx.NonZeroCountCache[3+8*1]
		}
		if is8x8DCT(leftType[h264LeftBot]) {
			ctx.NonZeroCountCache[3+8*3] = uint8((m.CBPTable[leftXY[h264LeftBot]] & 0x8000) >> 12)
			ctx.NonZeroCountCache[3+8*4] = ctx.NonZeroCountCache[3+8*3]
		}
		if is8x8DCT(mbType) {
			h264SetLoopFilter8x8DCTNNZ(&ctx.NonZeroCountCache, 0, (ctx.CBP&0x1000)>>12)
			h264SetLoopFilter8x8DCTNNZ(&ctx.NonZeroCountCache, 4, (ctx.CBP&0x2000)>>12)
			h264SetLoopFilter8x8DCTNNZ(&ctx.NonZeroCountCache, 8, (ctx.CBP&0x4000)>>12)
			h264SetLoopFilter8x8DCTNNZ(&ctx.NonZeroCountCache, 12, (ctx.CBP&0x8000)>>12)
		}
	}

	return nil
}

func h264LoopFilterFrameMBAFF(p *h264LoopFilterSliceParams) bool {
	return p.PPS != nil && p.PPS.SPS != nil &&
		p.PPS.SPS.MBAFF != 0 && p.PictureStructure == PictureFrame
}

func h264FrameMBAFFLoopFilterView(dst *h264PicturePlanes, mbY int, mbType uint32) (h264PicturePlanes, int, error) {
	if dst == nil || mbY < 0 {
		return h264PicturePlanes{}, 0, ErrInvalidData
	}
	view := *dst
	if mbType&MBTypeInterlaced == 0 {
		return view, mbY, nil
	}
	pictureStructure := PictureTopField
	if mbY&1 != 0 {
		pictureStructure = PictureBottomField
	}
	applySimpleFieldRefPlane(&view, pictureStructure)
	return view, mbY >> 1, nil
}

func h264FrameMBAFFLoopFilterViewHigh(dst *h264PicturePlanesHigh, mbY int, mbType uint32) (h264PicturePlanesHigh, int, error) {
	if dst == nil || mbY < 0 {
		return h264PicturePlanesHigh{}, 0, ErrInvalidData
	}
	view := *dst
	if mbType&MBTypeInterlaced == 0 {
		return view, mbY, nil
	}
	pictureStructure := PictureTopField
	if mbY&1 != 0 {
		pictureStructure = PictureBottomField
	}
	applySimpleFieldRefPlaneHigh(&view, pictureStructure)
	return view, mbY >> 1, nil
}

func h264SetLoopFilter8x8DCTNNZ(cache *[h264MotionCacheSize]uint8, base int, value int) {
	v := uint8(value)
	cache[h264Scan8[base+0]] = v
	cache[h264Scan8[base+1]] = v
	cache[h264Scan8[base+2]] = v
	cache[h264Scan8[base+3]] = v
}

func (m *macroblockTables) fillLoopFilterCachesInterFrame(ctx *h264LoopFilterContext, mbXY int, topXY int, leftXY int, mbType uint32, topType uint32, leftType uint32, list int, p h264LoopFilterSliceParams, params []h264LoopFilterSliceParams) error {
	return m.fillLoopFilterCachesInterFramePtr(ctx, mbXY, topXY, leftXY, mbType, topType, leftType, list, &p, params)
}

func (m *macroblockTables) fillLoopFilterCachesInterFramePtr(ctx *h264LoopFilterContext, mbXY int, topXY int, leftXY int, mbType uint32, topType uint32, leftType uint32, list int, p *h264LoopFilterSliceParams, params []h264LoopFilterSliceParams) error {
	if ctx == nil || list < 0 || list > 1 {
		return ErrInvalidData
	}
	base := int(h264Scan8[0])
	if isInter(mbType) || isDirect(mbType) {
		if usesList(topType, list) {
			if err := m.copyTopMotionForLoopFilter(ctx, topXY, list, base, m.loopFilterParamsForMB(params, topXY, p), mbType); err != nil {
				return err
			}
		} else {
			clearMotionRow(&ctx.Motion.MV[list], base-8, 4)
			fillRefRow(&ctx.Motion.Ref[list], base-8, 4, h264ListNotUsed)
		}

		if mbType&MBTypeInterlaced != leftType&MBTypeInterlaced {
			// FFmpeg leaves the left cache untouched for mixed frame/field MBAFF
			// neighbors; the mixed vertical edge computes strength separately.
		} else if usesList(leftType, list) {
			if err := m.copyLeftMotionForLoopFilter(ctx, leftXY, list, base, m.loopFilterParamsForMB(params, leftXY, p), mbType); err != nil {
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

	if err := m.copyCurrentMotionForLoopFilter(ctx, mbXY, list, base, p, mbType); err != nil {
		return err
	}
	return nil
}

func (m *macroblockTables) copyTopMotionForLoopFilter(ctx *h264LoopFilterContext, topXY int, list int, base int, p *h264LoopFilterSliceParams, currentMBType uint32) error {
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
	ref0 := int8(0)
	ref1 := int8(0)
	if currentMBType&MBTypeInterlaced != 0 && h264LoopFilterFrameMBAFF(p) {
		ref0 = p.ref2FrameForInterlacedLoopFilter(list, m.RefIndex[list][refBase+0])
		ref1 = p.ref2FrameForInterlacedLoopFilter(list, m.RefIndex[list][refBase+1])
	} else {
		ref0 = p.ref2Frame(list, m.RefIndex[list][refBase+0])
		ref1 = p.ref2Frame(list, m.RefIndex[list][refBase+1])
	}
	ctx.Motion.Ref[list][base+0-8] = ref0
	ctx.Motion.Ref[list][base+1-8] = ref0
	ctx.Motion.Ref[list][base+2-8] = ref1
	ctx.Motion.Ref[list][base+3-8] = ref1
	return nil
}

func (m *macroblockTables) copyLeftMotionForLoopFilter(ctx *h264LoopFilterContext, leftXY int, list int, base int, p *h264LoopFilterSliceParams, currentMBType uint32) error {
	if err := m.checkCodedMBXY(leftXY); err != nil {
		return err
	}
	bXY := int(m.MB2BXY[leftXY]) + 3
	refBase := 4*leftXY + 1
	if err := checkRange(len(m.RefIndex[list]), refBase, 3); err != nil {
		return err
	}
	refTop := int8(0)
	refBottom := int8(0)
	if currentMBType&MBTypeInterlaced != 0 && h264LoopFilterFrameMBAFF(p) {
		refTop = p.ref2FrameForInterlacedLoopFilter(list, m.RefIndex[list][refBase])
		refBottom = p.ref2FrameForInterlacedLoopFilter(list, m.RefIndex[list][refBase+2])
	} else {
		refTop = p.ref2Frame(list, m.RefIndex[list][refBase])
		refBottom = p.ref2Frame(list, m.RefIndex[list][refBase+2])
	}
	for row := 0; row < 4; row++ {
		mvIdx := bXY + row*m.BStride
		if err := checkRange(len(m.MotionVal[list]), mvIdx, 1); err != nil {
			return err
		}
		cacheIdx := base - 1 + row*8
		ctx.Motion.MV[list][cacheIdx] = m.MotionVal[list][mvIdx]
		if row < 2 {
			ctx.Motion.Ref[list][cacheIdx] = refTop
		} else {
			ctx.Motion.Ref[list][cacheIdx] = refBottom
		}
	}
	return nil
}

func (m *macroblockTables) copyCurrentMotionForLoopFilter(ctx *h264LoopFilterContext, mbXY int, list int, base int, p *h264LoopFilterSliceParams, currentMBType uint32) error {
	if err := m.checkCodedMBXY(mbXY); err != nil {
		return err
	}
	mbX := mbXY % m.MBStride
	mbY := mbXY / m.MBStride
	src := 4*mbX + 4*mbY*m.BStride
	if err := checkRange(len(m.MotionVal[list]), src, 3*m.BStride+4); err != nil {
		return err
	}
	for row := 0; row < 4; row++ {
		copy(ctx.Motion.MV[list][base+row*8:base+row*8+4], m.MotionVal[list][src+row*m.BStride:src+row*m.BStride+4])
	}

	refBase := 4 * mbXY
	if err := checkRange(len(m.RefIndex[list]), refBase, 4); err != nil {
		return err
	}
	ref0 := int8(0)
	ref1 := int8(0)
	ref2 := int8(0)
	ref3 := int8(0)
	if currentMBType&MBTypeInterlaced != 0 && h264LoopFilterFrameMBAFF(p) {
		ref0 = p.ref2FrameForInterlacedLoopFilter(list, m.RefIndex[list][refBase+0])
		ref1 = p.ref2FrameForInterlacedLoopFilter(list, m.RefIndex[list][refBase+1])
		ref2 = p.ref2FrameForInterlacedLoopFilter(list, m.RefIndex[list][refBase+2])
		ref3 = p.ref2FrameForInterlacedLoopFilter(list, m.RefIndex[list][refBase+3])
	} else {
		ref0 = p.ref2Frame(list, m.RefIndex[list][refBase+0])
		ref1 = p.ref2Frame(list, m.RefIndex[list][refBase+1])
		ref2 = p.ref2Frame(list, m.RefIndex[list][refBase+2])
		ref3 = p.ref2Frame(list, m.RefIndex[list][refBase+3])
	}
	for row := 0; row < 2; row++ {
		ctx.Motion.Ref[list][base+row*8+0] = ref0
		ctx.Motion.Ref[list][base+row*8+1] = ref0
		ctx.Motion.Ref[list][base+row*8+2] = ref1
		ctx.Motion.Ref[list][base+row*8+3] = ref1
	}
	for row := 2; row < 4; row++ {
		ctx.Motion.Ref[list][base+row*8+0] = ref2
		ctx.Motion.Ref[list][base+row*8+1] = ref2
		ctx.Motion.Ref[list][base+row*8+2] = ref3
		ctx.Motion.Ref[list][base+row*8+3] = ref3
	}
	return nil
}

func (m *macroblockTables) filterFrameMacroblock(dst *h264PicturePlanes, mbX int, mbY int, p *h264LoopFilterSliceParams, ctx *h264LoopFilterContext) error {
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

func (m *macroblockTables) filterFrameMacroblockDir(dst *h264PicturePlanes, dstY int, dstCb int, dstCr int, p *h264LoopFilterSliceParams, ctx *h264LoopFilterContext, dir int) error {
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

	maskEdge := h264LoopFilterMaskEdgeTable[dir][(mbType>>3)&7]
	edges := 4
	if maskEdge == 3 && ctx.CBP&15 == 0 {
		edges = 1
	}
	maskPar0 := mbType & (MBType16x16 | (MBType8x16 >> uint(dir)))
	mvyLimit := h264LoopFilterMVYLimit(mbType)
	frameMBAFF := h264LoopFilterFrameMBAFF(p)
	firstVerticalEdgeDone := false

	if dir == 0 && frameMBAFF && mbmType != 0 && mbType&MBTypeInterlaced != mbmType&MBTypeInterlaced {
		if err := m.filterFrameMBAFFMixedVerticalEdge(dst, dstY, dstCb, dstCr, *p, ctx, mbType); err != nil {
			return err
		}
		firstVerticalEdgeDone = true
	}

	if mbmType != 0 && !firstVerticalEdgeDone {
		mbY := ctx.MBXY / m.MBStride
		if frameMBAFF && dir == 1 && mbY&1 == 0 && (mbmType&^mbType)&MBTypeInterlaced != 0 {
			if err := m.filterFrameMBAFFTopHorizontalEdge(dst, dstY, dstCb, dstCr, *p, ctx, mbType); err != nil {
				return err
			}
		} else {
			bS, err := m.loopFilterBoundaryStrength(ctx, mbType, mbmType, dir, maskPar0, p.ListCount, mvyLimit, frameMBAFF)
			if err != nil {
				return err
			}
			if h264LoopFilterBSSum(bS) != 0 {
				qp := (int(m.QScaleTable[ctx.MBXY]) + int(m.QScaleTable[mbmXY]) + 1) >> 1
				chromaQP := [2]int{
					(int(p.PPS.ChromaQPTable[0][m.QScaleTable[ctx.MBXY]]) + int(p.PPS.ChromaQPTable[0][m.QScaleTable[mbmXY]]) + 1) >> 1,
					(int(p.PPS.ChromaQPTable[1][m.QScaleTable[ctx.MBXY]]) + int(p.PPS.ChromaQPTable[1][m.QScaleTable[mbmXY]]) + 1) >> 1,
				}
				if err := h264ApplyLoopFilterEdgePtr(dst, dstY, dstCb, dstCr, dir, 0, bS, qp, chromaQP, p, true, true, true); err != nil {
					return err
				}
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
		if err := h264ApplyLoopFilterEdgePtr(dst, dstY, dstCb, dstCr, dir, edge, bS, qp, chromaQP, p, false, filterLuma, filterChroma); err != nil {
			return err
		}
	}
	return nil
}

func (m *macroblockTables) filterFrameMBAFFTopHorizontalEdge(dst *h264PicturePlanes, dstY int, dstCb int, dstCr int, p h264LoopFilterSliceParams, ctx *h264LoopFilterContext, mbType uint32) error {
	if dst == nil || ctx == nil || p.PPS == nil {
		return ErrInvalidData
	}
	mbnBaseXY := ctx.MBXY - 2*m.MBStride
	if mbnBaseXY < 0 {
		return ErrInvalidData
	}
	sliceNum := m.SliceTable[ctx.MBXY]
	for field := 0; field < 2; field++ {
		mbnXY := mbnBaseXY + field*m.MBStride
		if p.DeblockingFilter == 2 && !m.sameSlice(mbnXY, sliceNum) {
			continue
		}
		bS, err := m.loopFilterMBAFFTopHorizontalStrength(ctx, mbType, mbnXY, p)
		if err != nil {
			return err
		}
		if h264LoopFilterBSSum(bS) == 0 {
			continue
		}
		qp := (int(m.QScaleTable[ctx.MBXY]) + int(m.QScaleTable[mbnXY]) + 1) >> 1
		chromaQP := [2]int{
			(int(p.PPS.ChromaQPTable[0][m.QScaleTable[ctx.MBXY]]) + int(p.PPS.ChromaQPTable[0][m.QScaleTable[mbnXY]]) + 1) >> 1,
			(int(p.PPS.ChromaQPTable[1][m.QScaleTable[ctx.MBXY]]) + int(p.PPS.ChromaQPTable[1][m.QScaleTable[mbnXY]]) + 1) >> 1,
		}
		yOff := dstY + field*dst.LumaStride
		if err := h264FilterMBEdgeHLuma(dst.Y, yOff, 2*dst.LumaStride, bS, qp, int(p.SliceAlphaC0Offset), int(p.SliceBetaOffset), false); err != nil {
			return err
		}
		if dst.ChromaFormatIDC == 0 {
			continue
		}
		cbOff := dstCb + field*dst.ChromaStride
		crOff := dstCr + field*dst.ChromaStride
		switch dst.ChromaFormatIDC {
		case 1, 2:
			if err := h264FilterMBEdgeHChroma(dst.Cb, cbOff, 2*dst.ChromaStride, bS, chromaQP[0], int(p.SliceAlphaC0Offset), int(p.SliceBetaOffset), false); err != nil {
				return err
			}
			if err := h264FilterMBEdgeHChroma(dst.Cr, crOff, 2*dst.ChromaStride, bS, chromaQP[1], int(p.SliceAlphaC0Offset), int(p.SliceBetaOffset), false); err != nil {
				return err
			}
		case 3:
			if err := h264FilterMBEdgeHLuma(dst.Cb, cbOff, 2*dst.ChromaStride, bS, chromaQP[0], int(p.SliceAlphaC0Offset), int(p.SliceBetaOffset), false); err != nil {
				return err
			}
			if err := h264FilterMBEdgeHLuma(dst.Cr, crOff, 2*dst.ChromaStride, bS, chromaQP[1], int(p.SliceAlphaC0Offset), int(p.SliceBetaOffset), false); err != nil {
				return err
			}
		default:
			return ErrUnsupported
		}
	}
	return nil
}

func (m *macroblockTables) loopFilterMBAFFTopHorizontalStrength(ctx *h264LoopFilterContext, mbType uint32, mbnXY int, p h264LoopFilterSliceParams) ([4]int16, error) {
	var bS [4]int16
	if ctx == nil {
		return bS, ErrInvalidData
	}
	if err := m.checkCodedMBXY(mbnXY); err != nil {
		return bS, err
	}
	mbnType := m.MacroblockTyp[mbnXY]
	if isIntra(mbType | mbnType) {
		for i := range bS {
			bS[i] = 3
		}
		return bS, nil
	}
	base := int(h264Scan8[0])
	if !p.CABAC && is8x8DCT(mbnType) {
		for i := 0; i < 4; i++ {
			mask := uint32(0x4000)
			if i >= 2 {
				mask = 0x8000
			}
			if m.CBPTable[mbnXY]&int(mask) != 0 || ctx.NonZeroCountCache[base+i] != 0 {
				bS[i] = 2
			} else {
				bS[i] = 1
			}
		}
		return bS, nil
	}
	mbnNNZ := m.NonZeroCount[mbnXY]
	for i := 0; i < 4; i++ {
		if ctx.NonZeroCountCache[base+i] != 0 || mbnNNZ[12+i] != 0 {
			bS[i] = 2
		} else {
			bS[i] = 1
		}
	}
	return bS, nil
}

func (m *macroblockTables) filterFrameMBAFFMixedVerticalEdge(dst *h264PicturePlanes, dstY int, dstCb int, dstCr int, p h264LoopFilterSliceParams, ctx *h264LoopFilterContext, mbType uint32) error {
	if dst == nil || ctx == nil || p.PPS == nil {
		return ErrInvalidData
	}
	bS, err := m.loopFilterMBAFFMixedVerticalStrength(ctx, mbType, p)
	if err != nil {
		return err
	}
	mbQP := int(m.QScaleTable[ctx.MBXY])
	leftTopQP := int(m.QScaleTable[ctx.LeftMBXYs[h264LeftTop]])
	leftBotQP := int(m.QScaleTable[ctx.LeftMBXYs[h264LeftBot]])
	qp := [2]int{
		(mbQP + leftTopQP + 1) >> 1,
		(mbQP + leftBotQP + 1) >> 1,
	}
	chromaQP := [2][2]int{
		{
			(int(p.PPS.ChromaQPTable[0][m.QScaleTable[ctx.MBXY]]) + int(p.PPS.ChromaQPTable[0][m.QScaleTable[ctx.LeftMBXYs[h264LeftTop]]]) + 1) >> 1,
			(int(p.PPS.ChromaQPTable[0][m.QScaleTable[ctx.MBXY]]) + int(p.PPS.ChromaQPTable[0][m.QScaleTable[ctx.LeftMBXYs[h264LeftBot]]]) + 1) >> 1,
		},
		{
			(int(p.PPS.ChromaQPTable[1][m.QScaleTable[ctx.MBXY]]) + int(p.PPS.ChromaQPTable[1][m.QScaleTable[ctx.LeftMBXYs[h264LeftTop]]]) + 1) >> 1,
			(int(p.PPS.ChromaQPTable[1][m.QScaleTable[ctx.MBXY]]) + int(p.PPS.ChromaQPTable[1][m.QScaleTable[ctx.LeftMBXYs[h264LeftBot]]]) + 1) >> 1,
		},
	}

	mbField := mbType&MBTypeInterlaced != 0
	if mbField {
		if err := h264FilterMBMBAFFEdgeVLuma(dst.Y, dstY, dst.LumaStride, bS, 0, 1, qp[0], int(p.SliceAlphaC0Offset), int(p.SliceBetaOffset), true); err != nil {
			return err
		}
		if err := h264FilterMBMBAFFEdgeVLuma(dst.Y, dstY+8*dst.LumaStride, dst.LumaStride, bS, 4, 1, qp[1], int(p.SliceAlphaC0Offset), int(p.SliceBetaOffset), true); err != nil {
			return err
		}
		if dst.ChromaFormatIDC == 0 {
			return nil
		}
		chromaRows := 8
		if dst.ChromaFormatIDC == 1 {
			chromaRows = 4
		}
		if err := h264ApplyMBAFFMixedVerticalChroma(dst, dstCb, dstCr, dst.ChromaStride, bS, 0, 1, chromaQP[0][0], chromaQP[1][0], p); err != nil {
			return err
		}
		return h264ApplyMBAFFMixedVerticalChroma(dst, dstCb+chromaRows*dst.ChromaStride, dstCr+chromaRows*dst.ChromaStride, dst.ChromaStride, bS, 4, 1, chromaQP[0][1], chromaQP[1][1], p)
	}

	if err := h264FilterMBMBAFFEdgeVLuma(dst.Y, dstY, 2*dst.LumaStride, bS, 0, 2, qp[0], int(p.SliceAlphaC0Offset), int(p.SliceBetaOffset), true); err != nil {
		return err
	}
	if err := h264FilterMBMBAFFEdgeVLuma(dst.Y, dstY+dst.LumaStride, 2*dst.LumaStride, bS, 1, 2, qp[1], int(p.SliceAlphaC0Offset), int(p.SliceBetaOffset), true); err != nil {
		return err
	}
	if dst.ChromaFormatIDC == 0 {
		return nil
	}
	if err := h264ApplyMBAFFMixedVerticalChroma(dst, dstCb, dstCr, 2*dst.ChromaStride, bS, 0, 2, chromaQP[0][0], chromaQP[1][0], p); err != nil {
		return err
	}
	return h264ApplyMBAFFMixedVerticalChroma(dst, dstCb+dst.ChromaStride, dstCr+dst.ChromaStride, 2*dst.ChromaStride, bS, 1, 2, chromaQP[0][1], chromaQP[1][1], p)
}

func (m *macroblockTables) loopFilterMBAFFMixedVerticalStrength(ctx *h264LoopFilterContext, mbType uint32, p h264LoopFilterSliceParams) ([8]int16, error) {
	var bS [8]int16
	if ctx == nil {
		return bS, ErrInvalidData
	}
	if err := m.checkCodedMBXY(ctx.LeftMBXYs[h264LeftTop]); err != nil {
		return bS, err
	}
	if err := m.checkCodedMBXY(ctx.LeftMBXYs[h264LeftBot]); err != nil {
		return bS, err
	}
	if isIntra(mbType) {
		for i := range bS {
			bS[i] = 4
		}
		return bS, nil
	}

	mbY := ctx.MBXY / m.MBStride
	mbField := mbType&MBTypeInterlaced != 0
	offset := h264LoopFilterMBAFFMixedNNZOffset(mbField, mbY&1)
	for i := 0; i < 8; i++ {
		left := i & 1
		if mbField {
			left = i >> 2
		}
		mbnXY := ctx.LeftMBXYs[left]
		mbnType := ctx.LeftTypes[left]
		if isIntra(mbnType) {
			bS[i] = 4
			continue
		}
		neighborNNZ := false
		if !p.CABAC && is8x8DCT(mbnType) {
			maskSelect := mbY & 1
			if mbField && i&2 != 0 {
				maskSelect = 1
			} else if mbField {
				maskSelect = 0
			}
			mask := 2 << 12
			if maskSelect != 0 {
				mask = 8 << 12
			}
			neighborNNZ = m.CBPTable[mbnXY]&mask != 0
		} else {
			neighborNNZ = m.NonZeroCount[mbnXY][offset[i]] != 0
		}
		if ctx.NonZeroCountCache[12+8*(i>>1)] != 0 || neighborNNZ {
			bS[i] = 2
		} else {
			bS[i] = 1
		}
	}
	return bS, nil
}

func h264LoopFilterMBAFFMixedNNZOffset(mbField bool, mbYParity int) [8]int {
	if mbField {
		return [8]int{3, 7, 11, 15, 3, 7, 11, 15}
	}
	if mbYParity != 0 {
		return [8]int{11, 11, 11, 11, 15, 15, 15, 15}
	}
	return [8]int{3, 3, 3, 3, 7, 7, 7, 7}
}

func h264ApplyMBAFFMixedVerticalChroma(dst *h264PicturePlanes, dstCb int, dstCr int, stride int, bS [8]int16, start int, bsi int, chromaQP0 int, chromaQP1 int, p h264LoopFilterSliceParams) error {
	switch dst.ChromaFormatIDC {
	case 1, 2:
		if err := h264FilterMBMBAFFEdgeVChroma(dst.Cb, dstCb, stride, bS, start, bsi, chromaQP0, int(p.SliceAlphaC0Offset), int(p.SliceBetaOffset), dst.ChromaFormatIDC); err != nil {
			return err
		}
		return h264FilterMBMBAFFEdgeVChroma(dst.Cr, dstCr, stride, bS, start, bsi, chromaQP1, int(p.SliceAlphaC0Offset), int(p.SliceBetaOffset), dst.ChromaFormatIDC)
	case 3:
		if err := h264FilterMBMBAFFEdgeVLuma(dst.Cb, dstCb, stride, bS, start, bsi, chromaQP0, int(p.SliceAlphaC0Offset), int(p.SliceBetaOffset), true); err != nil {
			return err
		}
		return h264FilterMBMBAFFEdgeVLuma(dst.Cr, dstCr, stride, bS, start, bsi, chromaQP1, int(p.SliceAlphaC0Offset), int(p.SliceBetaOffset), true)
	default:
		return ErrUnsupported
	}
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

	maskEdge := h264LoopFilterMaskEdgeTable[dir][(mbType>>3)&7]
	edges := 4
	if maskEdge == 3 && ctx.CBP&15 == 0 {
		edges = 1
	}
	maskPar0 := mbType & (MBType16x16 | (MBType8x16 >> uint(dir)))
	mvyLimit := h264LoopFilterMVYLimit(mbType)
	frameMBAFF := h264LoopFilterFrameMBAFF(&p)
	firstVerticalEdgeDone := false

	if dir == 0 && frameMBAFF && mbmType != 0 && mbType&MBTypeInterlaced != mbmType&MBTypeInterlaced {
		if err := m.filterFrameMBAFFMixedVerticalEdgeHigh(dst, dstY, dstCb, dstCr, p, ctx, mbType, bitDepth); err != nil {
			return err
		}
		firstVerticalEdgeDone = true
	}

	if mbmType != 0 && !firstVerticalEdgeDone {
		mbY := ctx.MBXY / m.MBStride
		if frameMBAFF && dir == 1 && mbY&1 == 0 && (mbmType&^mbType)&MBTypeInterlaced != 0 {
			if err := m.filterFrameMBAFFTopHorizontalEdgeHigh(dst, dstY, dstCb, dstCr, p, ctx, mbType, bitDepth); err != nil {
				return err
			}
		} else {
			bS, err := m.loopFilterBoundaryStrength(ctx, mbType, mbmType, dir, maskPar0, p.ListCount, mvyLimit, frameMBAFF)
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

func (m *macroblockTables) filterFrameMBAFFTopHorizontalEdgeHigh(dst *h264PicturePlanesHigh, dstY int, dstCb int, dstCr int, p h264LoopFilterSliceParams, ctx *h264LoopFilterContext, mbType uint32, bitDepth int) error {
	if dst == nil || ctx == nil || p.PPS == nil {
		return ErrInvalidData
	}
	mbnBaseXY := ctx.MBXY - 2*m.MBStride
	if mbnBaseXY < 0 {
		return ErrInvalidData
	}
	sliceNum := m.SliceTable[ctx.MBXY]
	for field := 0; field < 2; field++ {
		mbnXY := mbnBaseXY + field*m.MBStride
		if p.DeblockingFilter == 2 && !m.sameSlice(mbnXY, sliceNum) {
			continue
		}
		bS, err := m.loopFilterMBAFFTopHorizontalStrength(ctx, mbType, mbnXY, p)
		if err != nil {
			return err
		}
		if h264LoopFilterBSSum(bS) == 0 {
			continue
		}
		qp := (int(m.QScaleTable[ctx.MBXY]) + int(m.QScaleTable[mbnXY]) + 1) >> 1
		chromaQP := [2]int{
			(int(p.PPS.ChromaQPTable[0][m.QScaleTable[ctx.MBXY]]) + int(p.PPS.ChromaQPTable[0][m.QScaleTable[mbnXY]]) + 1) >> 1,
			(int(p.PPS.ChromaQPTable[1][m.QScaleTable[ctx.MBXY]]) + int(p.PPS.ChromaQPTable[1][m.QScaleTable[mbnXY]]) + 1) >> 1,
		}
		yOff := dstY + field*dst.LumaStride
		if err := h264FilterMBEdgeHLumaHigh(dst.Y, yOff, 2*dst.LumaStride, bS, qp, int(p.SliceAlphaC0Offset), int(p.SliceBetaOffset), false, bitDepth); err != nil {
			return err
		}
		if dst.ChromaFormatIDC == 0 {
			continue
		}
		cbOff := dstCb + field*dst.ChromaStride
		crOff := dstCr + field*dst.ChromaStride
		switch dst.ChromaFormatIDC {
		case 1, 2:
			if err := h264FilterMBEdgeHChromaHigh(dst.Cb, cbOff, 2*dst.ChromaStride, bS, chromaQP[0], int(p.SliceAlphaC0Offset), int(p.SliceBetaOffset), false, bitDepth); err != nil {
				return err
			}
			if err := h264FilterMBEdgeHChromaHigh(dst.Cr, crOff, 2*dst.ChromaStride, bS, chromaQP[1], int(p.SliceAlphaC0Offset), int(p.SliceBetaOffset), false, bitDepth); err != nil {
				return err
			}
		case 3:
			if err := h264FilterMBEdgeHLumaHigh(dst.Cb, cbOff, 2*dst.ChromaStride, bS, chromaQP[0], int(p.SliceAlphaC0Offset), int(p.SliceBetaOffset), false, bitDepth); err != nil {
				return err
			}
			if err := h264FilterMBEdgeHLumaHigh(dst.Cr, crOff, 2*dst.ChromaStride, bS, chromaQP[1], int(p.SliceAlphaC0Offset), int(p.SliceBetaOffset), false, bitDepth); err != nil {
				return err
			}
		default:
			return ErrUnsupported
		}
	}
	return nil
}

func (m *macroblockTables) filterFrameMBAFFMixedVerticalEdgeHigh(dst *h264PicturePlanesHigh, dstY int, dstCb int, dstCr int, p h264LoopFilterSliceParams, ctx *h264LoopFilterContext, mbType uint32, bitDepth int) error {
	if dst == nil || ctx == nil || p.PPS == nil {
		return ErrInvalidData
	}
	bS, err := m.loopFilterMBAFFMixedVerticalStrength(ctx, mbType, p)
	if err != nil {
		return err
	}
	mbQP := int(m.QScaleTable[ctx.MBXY])
	leftTopQP := int(m.QScaleTable[ctx.LeftMBXYs[h264LeftTop]])
	leftBotQP := int(m.QScaleTable[ctx.LeftMBXYs[h264LeftBot]])
	qp := [2]int{
		(mbQP + leftTopQP + 1) >> 1,
		(mbQP + leftBotQP + 1) >> 1,
	}
	chromaQP := [2][2]int{
		{
			(int(p.PPS.ChromaQPTable[0][m.QScaleTable[ctx.MBXY]]) + int(p.PPS.ChromaQPTable[0][m.QScaleTable[ctx.LeftMBXYs[h264LeftTop]]]) + 1) >> 1,
			(int(p.PPS.ChromaQPTable[0][m.QScaleTable[ctx.MBXY]]) + int(p.PPS.ChromaQPTable[0][m.QScaleTable[ctx.LeftMBXYs[h264LeftBot]]]) + 1) >> 1,
		},
		{
			(int(p.PPS.ChromaQPTable[1][m.QScaleTable[ctx.MBXY]]) + int(p.PPS.ChromaQPTable[1][m.QScaleTable[ctx.LeftMBXYs[h264LeftTop]]]) + 1) >> 1,
			(int(p.PPS.ChromaQPTable[1][m.QScaleTable[ctx.MBXY]]) + int(p.PPS.ChromaQPTable[1][m.QScaleTable[ctx.LeftMBXYs[h264LeftBot]]]) + 1) >> 1,
		},
	}

	mbField := mbType&MBTypeInterlaced != 0
	if mbField {
		if err := h264FilterMBMBAFFEdgeVLumaHigh(dst.Y, dstY, dst.LumaStride, bS, 0, 1, qp[0], int(p.SliceAlphaC0Offset), int(p.SliceBetaOffset), true, bitDepth); err != nil {
			return err
		}
		if err := h264FilterMBMBAFFEdgeVLumaHigh(dst.Y, dstY+8*dst.LumaStride, dst.LumaStride, bS, 4, 1, qp[1], int(p.SliceAlphaC0Offset), int(p.SliceBetaOffset), true, bitDepth); err != nil {
			return err
		}
		if dst.ChromaFormatIDC == 0 {
			return nil
		}
		chromaRows := 8
		if dst.ChromaFormatIDC == 1 {
			chromaRows = 4
		}
		if err := h264ApplyMBAFFMixedVerticalChromaHigh(dst, dstCb, dstCr, dst.ChromaStride, bS, 0, 1, chromaQP[0][0], chromaQP[1][0], p, bitDepth); err != nil {
			return err
		}
		return h264ApplyMBAFFMixedVerticalChromaHigh(dst, dstCb+chromaRows*dst.ChromaStride, dstCr+chromaRows*dst.ChromaStride, dst.ChromaStride, bS, 4, 1, chromaQP[0][1], chromaQP[1][1], p, bitDepth)
	}

	if err := h264FilterMBMBAFFEdgeVLumaHigh(dst.Y, dstY, 2*dst.LumaStride, bS, 0, 2, qp[0], int(p.SliceAlphaC0Offset), int(p.SliceBetaOffset), true, bitDepth); err != nil {
		return err
	}
	if err := h264FilterMBMBAFFEdgeVLumaHigh(dst.Y, dstY+dst.LumaStride, 2*dst.LumaStride, bS, 1, 2, qp[1], int(p.SliceAlphaC0Offset), int(p.SliceBetaOffset), true, bitDepth); err != nil {
		return err
	}
	if dst.ChromaFormatIDC == 0 {
		return nil
	}
	if err := h264ApplyMBAFFMixedVerticalChromaHigh(dst, dstCb, dstCr, 2*dst.ChromaStride, bS, 0, 2, chromaQP[0][0], chromaQP[1][0], p, bitDepth); err != nil {
		return err
	}
	return h264ApplyMBAFFMixedVerticalChromaHigh(dst, dstCb+dst.ChromaStride, dstCr+dst.ChromaStride, 2*dst.ChromaStride, bS, 1, 2, chromaQP[0][1], chromaQP[1][1], p, bitDepth)
}

func h264ApplyMBAFFMixedVerticalChromaHigh(dst *h264PicturePlanesHigh, dstCb int, dstCr int, stride int, bS [8]int16, start int, bsi int, chromaQP0 int, chromaQP1 int, p h264LoopFilterSliceParams, bitDepth int) error {
	switch dst.ChromaFormatIDC {
	case 1, 2:
		if err := h264FilterMBMBAFFEdgeVChromaHigh(dst.Cb, dstCb, stride, bS, start, bsi, chromaQP0, int(p.SliceAlphaC0Offset), int(p.SliceBetaOffset), dst.ChromaFormatIDC, bitDepth); err != nil {
			return err
		}
		return h264FilterMBMBAFFEdgeVChromaHigh(dst.Cr, dstCr, stride, bS, start, bsi, chromaQP1, int(p.SliceAlphaC0Offset), int(p.SliceBetaOffset), dst.ChromaFormatIDC, bitDepth)
	case 3:
		if err := h264FilterMBMBAFFEdgeVLumaHigh(dst.Cb, dstCb, stride, bS, start, bsi, chromaQP0, int(p.SliceAlphaC0Offset), int(p.SliceBetaOffset), true, bitDepth); err != nil {
			return err
		}
		return h264FilterMBMBAFFEdgeVLumaHigh(dst.Cr, dstCr, stride, bS, start, bsi, chromaQP1, int(p.SliceAlphaC0Offset), int(p.SliceBetaOffset), true, bitDepth)
	default:
		return ErrUnsupported
	}
}

func h264LoopFilterMVYLimit(mbType uint32) int {
	if mbType&MBTypeInterlaced != 0 {
		return 2
	}
	return 4
}

func (m *macroblockTables) loopFilterBoundaryStrength(ctx *h264LoopFilterContext, mbType uint32, mbmType uint32, dir int, maskPar0 uint32, listCount int, mvyLimit int, frameMBAFF bool) ([4]int16, error) {
	var bS [4]int16
	if isIntra(mbType | mbmType) {
		v := int16(3)
		if (mbType&MBTypeInterlaced == 0 && mbmType&MBTypeInterlaced == 0) || dir == 0 {
			v = 4
		}
		for i := range bS {
			bS[i] = v
		}
		return bS, nil
	}

	mvDone := false
	if frameMBAFF && dir != 0 && (mbType^mbmType)&MBTypeInterlaced != 0 {
		bS = [4]int16{1, 1, 1, 1}
		mvDone = true
	} else if maskPar0 != 0 && mbmType&(MBType16x16|(MBType8x16>>uint(dir))) != 0 {
		bIdx := int(h264Scan8[0])
		bnIdx := bIdx - 1
		if dir != 0 {
			bnIdx = bIdx - 8
		}
		v, err := h264LoopFilterCheckMV(ctx, bIdx, bnIdx, listCount, mvyLimit)
		if err != nil {
			return bS, err
		}
		bS = [4]int16{v, v, v, v}
		mvDone = true
	}

	bIdx := int(h264Scan8[0])
	bStep := 8
	bnDelta := 1
	if dir != 0 {
		bStep = 1
		bnDelta = 8
	}
	if mvDone {
		for i := 0; i < 4; i++ {
			idx := bIdx + i*bStep
			if ctx.NonZeroCountCache[idx]|ctx.NonZeroCountCache[idx-bnDelta] != 0 {
				bS[i] = 2
			}
		}
		return bS, nil
	}
	for i := 0; i < 4; i++ {
		idx := bIdx + i*bStep
		if ctx.NonZeroCountCache[idx]|ctx.NonZeroCountCache[idx-bnDelta] != 0 {
			bS[i] = 2
		} else {
			v, err := h264LoopFilterCheckMV(ctx, idx, idx-bnDelta, listCount, mvyLimit)
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
		bS = [4]int16{v, v, v, v}
		mvDone = true
	}

	bIdx := int(h264Scan8[0]) + edge
	bStep := 8
	bnDelta := 1
	if dir != 0 {
		bIdx = int(h264Scan8[0]) + edge*8
		bStep = 1
		bnDelta = 8
	}
	if mvDone {
		for i := 0; i < 4; i++ {
			idx := bIdx + i*bStep
			if ctx.NonZeroCountCache[idx]|ctx.NonZeroCountCache[idx-bnDelta] != 0 {
				bS[i] = 2
			}
		}
		return bS, nil
	}
	for i := 0; i < 4; i++ {
		idx := bIdx + i*bStep
		if ctx.NonZeroCountCache[idx]|ctx.NonZeroCountCache[idx-bnDelta] != 0 {
			bS[i] = 2
		} else {
			v, err := h264LoopFilterCheckMV(ctx, idx, idx-bnDelta, listCount, mvyLimit)
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
	return h264ApplyLoopFilterEdgePtr(dst, dstY, dstCb, dstCr, dir, edge, bS, qp, chromaQP, &p, intra, filterLuma, filterChroma)
}

func h264ApplyLoopFilterEdgePtr(dst *h264PicturePlanes, dstY int, dstCb int, dstCr int, dir int, edge int, bS [4]int16, qp int, chromaQP [2]int, p *h264LoopFilterSliceParams, intra bool, filterLuma bool, filterChroma bool) error {
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

func h264FilterMBMBAFFEdgeVLuma(pix []uint8, offset int, stride int, bS [8]int16, start int, bsi int, qp int, alphaOffset int, betaOffset int, intra bool) error {
	alpha, beta, indexA, err := h264LoopFilterThresholds(qp, alphaOffset, betaOffset)
	if err != nil || alpha == 0 || beta == 0 {
		return err
	}
	if start < 0 || bsi <= 0 || start+3*bsi >= len(bS) {
		return ErrInvalidData
	}
	if bS[start] < 4 || !intra {
		tc, err := h264LoopFilterTC8(indexA, bS, start, bsi, 0)
		if err != nil {
			return err
		}
		return h264HLoopFilterLumaMBAFF(pix, offset, stride, alpha, beta, &tc)
	}
	return h264HLoopFilterLumaMBAFFIntra(pix, offset, stride, alpha, beta)
}

func h264FilterMBMBAFFEdgeVLumaHigh(pix []uint16, offset int, stride int, bS [8]int16, start int, bsi int, qp int, alphaOffset int, betaOffset int, intra bool, bitDepth int) error {
	alpha, beta, indexA, err := h264LoopFilterThresholdsForBitDepth(qp, alphaOffset, betaOffset, bitDepth)
	if err != nil || alpha == 0 || beta == 0 {
		return err
	}
	if start < 0 || bsi <= 0 || start+3*bsi >= len(bS) {
		return ErrInvalidData
	}
	if bS[start] < 4 || !intra {
		tc, err := h264LoopFilterTC8(indexA, bS, start, bsi, 0)
		if err != nil {
			return err
		}
		return h264HLoopFilterLumaMBAFFHigh(pix, offset, stride, alpha, beta, &tc, bitDepth)
	}
	return h264HLoopFilterLumaMBAFFIntraHigh(pix, offset, stride, alpha, beta, bitDepth)
}

func h264FilterMBMBAFFEdgeVChroma(pix []uint8, offset int, stride int, bS [8]int16, start int, bsi int, qp int, alphaOffset int, betaOffset int, chromaFormatIDC int) error {
	alpha, beta, indexA, err := h264LoopFilterThresholds(qp, alphaOffset, betaOffset)
	if err != nil || alpha == 0 || beta == 0 {
		return err
	}
	if start < 0 || bsi <= 0 || start+3*bsi >= len(bS) {
		return ErrInvalidData
	}
	if bS[start] < 4 {
		tc, err := h264LoopFilterTC8(indexA, bS, start, bsi, 1)
		if err != nil {
			return err
		}
		if chromaFormatIDC == 2 {
			return h264HLoopFilterChroma422MBAFF(pix, offset, stride, alpha, beta, &tc)
		}
		return h264HLoopFilterChromaMBAFF(pix, offset, stride, alpha, beta, &tc)
	}
	if chromaFormatIDC == 2 {
		return h264HLoopFilterChroma422MBAFFIntra(pix, offset, stride, alpha, beta)
	}
	return h264HLoopFilterChromaMBAFFIntra(pix, offset, stride, alpha, beta)
}

func h264FilterMBMBAFFEdgeVChromaHigh(pix []uint16, offset int, stride int, bS [8]int16, start int, bsi int, qp int, alphaOffset int, betaOffset int, chromaFormatIDC int, bitDepth int) error {
	alpha, beta, indexA, err := h264LoopFilterThresholdsForBitDepth(qp, alphaOffset, betaOffset, bitDepth)
	if err != nil || alpha == 0 || beta == 0 {
		return err
	}
	if start < 0 || bsi <= 0 || start+3*bsi >= len(bS) {
		return ErrInvalidData
	}
	if bS[start] < 4 {
		tc, err := h264LoopFilterTC8(indexA, bS, start, bsi, 1)
		if err != nil {
			return err
		}
		if chromaFormatIDC == 2 {
			return h264HLoopFilterChroma422MBAFFHigh(pix, offset, stride, alpha, beta, &tc, bitDepth)
		}
		return h264HLoopFilterChromaMBAFFHigh(pix, offset, stride, alpha, beta, &tc, bitDepth)
	}
	if chromaFormatIDC == 2 {
		return h264HLoopFilterChroma422MBAFFIntraHigh(pix, offset, stride, alpha, beta, bitDepth)
	}
	return h264HLoopFilterChromaMBAFFIntraHigh(pix, offset, stride, alpha, beta, bitDepth)
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
	if qp < 0 || qp > 51 {
		return 0, 0, 0, ErrInvalidData
	}
	indexA := clipInt(qp+alphaOffset, 0, 51)
	indexB := clipInt(qp+betaOffset, 0, 51)
	return int(h264LoopFilterAlphaTable[indexA]), int(h264LoopFilterBetaTable[indexB]), indexA, nil
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

func h264LoopFilterTC8(indexA int, bS [8]int16, start int, bsi int, plus int8) ([4]int8, error) {
	var tc [4]int8
	if indexA < 0 || indexA >= len(h264LoopFilterTC0Table) || start < 0 || bsi <= 0 || start+3*bsi >= len(bS) {
		return tc, ErrInvalidData
	}
	for i := 0; i < 4; i++ {
		v := bS[start+i*bsi]
		if v < 0 || v > 3 {
			return tc, ErrInvalidData
		}
		tc[i] = h264LoopFilterTC0Table[indexA][v] + plus
	}
	return tc, nil
}
