// SPDX-License-Identifier: LGPL-2.1-or-later
//
// Source-shaped progressive frame-MB subset of FFmpeg n8.0.1
// libavcodec/h264_direct.c pred_spatial_direct_motion,
// pred_temp_direct_motion, and ff_h264_direct_dist_scale_factor. MBAFF/field
// remapping and row progress waits stay unsupported until their surrounding
// decoder state is ported.

package h264

type h264DirectMotionContext struct {
	RefEntries          [2][]simpleRefEntry
	CurPOC              int32
	DirectSpatialMVPred bool
	Direct8x8Inference  bool
	X264Build           int32
}

func (m *macroblockTables) predDirectMotionFrame(cache *macroblockMotionCache, mbXY int, mbType *uint32, subMBType *[4]uint32, ctx h264DirectMotionContext) error {
	if m == nil || cache == nil || mbType == nil || subMBType == nil {
		return ErrInvalidData
	}
	if err := m.checkCodedMBXY(mbXY); err != nil {
		return err
	}
	if len(ctx.RefEntries[0]) == 0 || len(ctx.RefEntries[1]) == 0 || ctx.RefEntries[1][0].frame == nil {
		return ErrUnsupported
	}
	if !ctx.Direct8x8Inference {
		return ErrUnsupported
	}

	col := ctx.RefEntries[1][0].frame
	colTables := col.tables
	if colTables == nil || colTables.MBWidth != m.MBWidth || colTables.MBHeight != m.MBHeight || colTables.BStride != m.BStride {
		return ErrUnsupported
	}
	if err := colTables.checkCodedMBXY(mbXY); err != nil {
		return err
	}
	if err := checkRange(len(colTables.RefIndex[0]), 4*mbXY, 4); err != nil {
		return err
	}

	mbTypeCol := colTables.MacroblockTyp[mbXY]
	if mbTypeCol&MBTypeInterlaced != 0 || *mbType&MBTypeInterlaced != 0 {
		return ErrUnsupported
	}
	if ctx.DirectSpatialMVPred {
		return predSpatialDirectMotionFrame(cache, colTables, mbXY, mbType, subMBType, ctx)
	}

	isB8x8 := is8x8(*mbType)
	directSubType := MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2
	if !isB8x8 && (mbTypeCol&(MBType16x16|MBTypeIntra4x4|MBTypeIntra16x16|MBTypeIntraPCM)) != 0 {
		*mbType |= MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2
		return predTemporalDirect16x16(cache, colTables, mbXY, ctx)
	}
	if !isB8x8 && (mbTypeCol&(MBType16x8|MBType8x16)) != 0 {
		*mbType |= MBTypeL0L1 | MBTypeDirect2 | (mbTypeCol & (MBType16x8 | MBType8x16))
	} else {
		*mbType |= MBType8x8 | MBTypeL0L1
	}
	return predTemporalDirect8x8(cache, colTables, mbXY, *mbType, directSubType, subMBType, ctx, isB8x8)
}

func predSpatialDirectMotionFrame(cache *macroblockMotionCache, col *macroblockTables, mbXY int, mbType *uint32, subMBType *[4]uint32, ctx h264DirectMotionContext) error {
	ref, mv, err := spatialDirectNeighborRefsAndMVs(cache, ctx)
	if err != nil {
		return err
	}
	isB8x8 := is8x8(*mbType)
	directSubType := MBTypeL0L1
	for list := 0; list < 2; list++ {
		if ref[list] >= 0 {
			continue
		}
		mask := MBTypeL0 << uint(2*list)
		mv[list] = [2]int16{}
		ref[list] = -1
		if !isB8x8 {
			*mbType &^= mask
		}
		directSubType &^= mask
	}
	if ref[0] < 0 && ref[1] < 0 {
		ref[0], ref[1] = 0, 0
		if !isB8x8 {
			*mbType |= MBTypeL0L1
		}
		directSubType |= MBTypeL0L1
	}

	base := int(h264Scan8[0])
	if !isB8x8 && mv[0] == ([2]int16{}) && mv[1] == ([2]int16{}) {
		fillRefRectangle(&cache.Ref[0], base, 4, 4, 8, ref[0])
		fillRefRectangle(&cache.Ref[1], base, 4, 4, 8, ref[1])
		fillMotionRectangle(&cache.MV[0], base, 4, 4, 8, [2]int16{})
		fillMotionRectangle(&cache.MV[1], base, 4, 4, 8, [2]int16{})
		*mbType = (*mbType &^ (MBType8x8 | MBType16x8 | MBType8x16 | MBTypeP1L0 | MBTypeP1L1)) | MBType16x16 | MBTypeDirect2
		return nil
	}

	mbTypeCol := col.MacroblockTyp[mbXY]
	directSubType |= MBType16x16 | MBTypeDirect2
	if !isB8x8 && (mbTypeCol&(MBType16x16|MBTypeIntra4x4|MBTypeIntra16x16|MBTypeIntraPCM)) != 0 {
		*mbType |= MBType16x16 | MBTypeDirect2
	} else if !isB8x8 && (mbTypeCol&(MBType16x8|MBType8x16)) != 0 {
		*mbType |= MBTypeDirect2 | (mbTypeCol & (MBType16x8 | MBType8x16))
	} else {
		*mbType |= MBType8x8
	}

	if is16x16(*mbType) {
		return predSpatialDirect16x16(cache, col, mbXY, ref, mv, ctx)
	}
	return predSpatialDirect8x8(cache, col, mbXY, mbType, directSubType, subMBType, ref, mv, ctx, isB8x8)
}

func spatialDirectNeighborRefsAndMVs(cache *macroblockMotionCache, ctx h264DirectMotionContext) ([2]int8, [2][2]int16, error) {
	var ref [2]int8
	var mv [2][2]int16
	if cache == nil {
		return ref, mv, ErrInvalidData
	}
	base := int(h264Scan8[0])
	for list := 0; list < 2; list++ {
		leftRef := cache.Ref[list][base-1]
		topRef := cache.Ref[list][base-8]
		refC := cache.Ref[list][base-8+4]
		c := cache.MV[list][base-8+4]
		if refC == h264PartNotAvailable {
			refC = cache.Ref[list][base-8-1]
			c = cache.MV[list][base-8-1]
		}
		ref[list] = minRefAsUnsigned(leftRef, topRef, refC)
		if ref[list] < 0 {
			continue
		}
		if int(ref[list]) >= len(ctx.RefEntries[list]) || ctx.RefEntries[list][ref[list]].frame == nil {
			return ref, mv, ErrInvalidData
		}
		a := cache.MV[list][base-1]
		b := cache.MV[list][base-8]
		matchCount := boolToInt(leftRef == ref[list]) + boolToInt(topRef == ref[list]) + boolToInt(refC == ref[list])
		if matchCount > 1 {
			mv[list][0] = int16(midPred(int(a[0]), int(b[0]), int(c[0])))
			mv[list][1] = int16(midPred(int(a[1]), int(b[1]), int(c[1])))
		} else if leftRef == ref[list] {
			mv[list] = a
		} else if topRef == ref[list] {
			mv[list] = b
		} else {
			mv[list] = c
		}
	}
	return ref, mv, nil
}

func predSpatialDirect16x16(cache *macroblockMotionCache, col *macroblockTables, mbXY int, ref [2]int8, mv [2][2]int16, ctx h264DirectMotionContext) error {
	base := int(h264Scan8[0])
	fillRefRectangle(&cache.Ref[0], base, 4, 4, 8, ref[0])
	fillRefRectangle(&cache.Ref[1], base, 4, 4, 8, ref[1])
	mv0, mv1 := mv[0], mv[1]
	if spatialDirectColZero(col, mbXY, 0, ctx) {
		mv0, mv1 = [2]int16{}, [2]int16{}
		if ref[0] > 0 {
			mv0 = mv[0]
		}
		if ref[1] > 0 {
			mv1 = mv[1]
		}
	}
	fillMotionRectangle(&cache.MV[0], base, 4, 4, 8, mv0)
	fillMotionRectangle(&cache.MV[1], base, 4, 4, 8, mv1)
	return nil
}

func predSpatialDirect8x8(cache *macroblockMotionCache, col *macroblockTables, mbXY int, mbType *uint32, directSubType uint32, subMBType *[4]uint32, ref [2]int8, mv [2][2]int16, ctx h264DirectMotionContext, wasB8x8 bool) error {
	if mbType == nil || (!is8x8(*mbType) && !is16x8(*mbType) && !is8x16(*mbType)) {
		return ErrInvalidData
	}
	n := 0
	for i8 := 0; i8 < 4; i8++ {
		if wasB8x8 && !isDirect(subMBType[i8]) {
			continue
		}
		subMBType[i8] = directSubType
		base := int(h264Scan8[4*i8])
		fillMotionRectangle(&cache.MV[0], base, 2, 2, 8, mv[0])
		fillMotionRectangle(&cache.MV[1], base, 2, 2, 8, mv[1])
		fillRefRectangle(&cache.Ref[0], base, 2, 2, 8, ref[0])
		fillRefRectangle(&cache.Ref[1], base, 2, 2, 8, ref[1])
		if spatialDirectColZero(col, mbXY, i8, ctx) {
			if ref[0] == 0 {
				fillMotionRectangle(&cache.MV[0], base, 2, 2, 8, [2]int16{})
			}
			if ref[1] == 0 {
				fillMotionRectangle(&cache.MV[1], base, 2, 2, 8, [2]int16{})
			}
			n += 4
		}
	}
	if !wasB8x8 && n&15 == 0 {
		*mbType = (*mbType &^ (MBType8x8 | MBType16x8 | MBType8x16 | MBTypeP1L0 | MBTypeP1L1)) | MBType16x16 | MBTypeDirect2
	}
	return nil
}

func spatialDirectColZero(col *macroblockTables, mbXY int, i8 int, ctx h264DirectMotionContext) bool {
	if col == nil || i8 < 0 || i8 > 3 || isIntra(col.MacroblockTyp[mbXY]) || len(ctx.RefEntries[1]) == 0 || ctx.RefEntries[1][0].long {
		return false
	}
	refIndex := 4*mbXY + i8
	if refIndex < 0 || refIndex >= len(col.RefIndex[0]) || refIndex >= len(col.RefIndex[1]) {
		return false
	}
	x8 := i8 & 1
	y8 := i8 >> 1
	mvIndex := int(col.MB2BXY[mbXY]) + x8*3 + y8*3*col.BStride
	if mvIndex < 0 || mvIndex >= len(col.MotionVal[0]) || mvIndex >= len(col.MotionVal[1]) {
		return false
	}
	if col.RefIndex[0][refIndex] == 0 {
		mv := col.MotionVal[0][mvIndex]
		return absInt(int(mv[0])) <= 1 && absInt(int(mv[1])) <= 1
	}
	if col.RefIndex[0][refIndex] < 0 && col.RefIndex[1][refIndex] == 0 && ctx.X264Build > 33 {
		mv := col.MotionVal[1][mvIndex]
		return absInt(int(mv[0])) <= 1 && absInt(int(mv[1])) <= 1
	}
	return false
}

func minRefAsUnsigned(a int8, b int8, c int8) int8 {
	ua := uint32(int32(a))
	ub := uint32(int32(b))
	uc := uint32(int32(c))
	if ub < ua {
		ua = ub
	}
	if uc < ua {
		ua = uc
	}
	return int8(ua)
}

func predTemporalDirect16x16(cache *macroblockMotionCache, col *macroblockTables, mbXY int, ctx h264DirectMotionContext) error {
	base := int(h264Scan8[0])
	fillRefRectangle(&cache.Ref[1], base, 4, 4, 8, 0)
	if isIntra(col.MacroblockTyp[mbXY]) {
		fillRefRectangle(&cache.Ref[0], base, 4, 4, 8, 0)
		fillMotionRectangle(&cache.MV[0], base, 4, 4, 8, [2]int16{})
		fillMotionRectangle(&cache.MV[1], base, 4, 4, 8, [2]int16{})
		return nil
	}

	ref0, mvCol, err := temporalDirectColocatedRefAndMV(col, mbXY, 0, ctx)
	if err != nil {
		return err
	}
	scale, err := temporalDirectDistScaleFactor(ctx, ref0)
	if err != nil {
		return err
	}
	mv0, mv1 := temporalDirectScaleMV(scale, mvCol)
	fillRefRectangle(&cache.Ref[0], base, 4, 4, 8, ref0)
	fillMotionRectangle(&cache.MV[0], base, 4, 4, 8, mv0)
	fillMotionRectangle(&cache.MV[1], base, 4, 4, 8, mv1)
	return nil
}

func predTemporalDirect8x8(cache *macroblockMotionCache, col *macroblockTables, mbXY int, mbType uint32, directSubType uint32, subMBType *[4]uint32, ctx h264DirectMotionContext, wasB8x8 bool) error {
	if !is8x8(mbType) && !is16x8(mbType) && !is8x16(mbType) {
		return ErrInvalidData
	}
	colIntra := isIntra(col.MacroblockTyp[mbXY])
	for i8 := 0; i8 < 4; i8++ {
		if wasB8x8 && !isDirect(subMBType[i8]) {
			continue
		}
		subMBType[i8] = directSubType
		base := int(h264Scan8[4*i8])
		fillRefRectangle(&cache.Ref[1], base, 2, 2, 8, 0)
		if colIntra {
			fillRefRectangle(&cache.Ref[0], base, 2, 2, 8, 0)
			fillMotionRectangle(&cache.MV[0], base, 2, 2, 8, [2]int16{})
			fillMotionRectangle(&cache.MV[1], base, 2, 2, 8, [2]int16{})
			continue
		}
		ref0, mvCol, err := temporalDirectColocatedRefAndMV(col, mbXY, i8, ctx)
		if err != nil {
			return err
		}
		scale, err := temporalDirectDistScaleFactor(ctx, ref0)
		if err != nil {
			return err
		}
		mv0, mv1 := temporalDirectScaleMV(scale, mvCol)
		fillRefRectangle(&cache.Ref[0], base, 2, 2, 8, ref0)
		fillMotionRectangle(&cache.MV[0], base, 2, 2, 8, mv0)
		fillMotionRectangle(&cache.MV[1], base, 2, 2, 8, mv1)
	}
	return nil
}

func temporalDirectColocatedRefAndMV(col *macroblockTables, mbXY int, i8 int, ctx h264DirectMotionContext) (int8, [2]int16, error) {
	var mv [2]int16
	if col == nil || i8 < 0 || i8 > 3 {
		return 0, mv, ErrInvalidData
	}
	refIndex := 4*mbXY + i8
	if err := checkRange(len(col.RefIndex[0]), refIndex, 1); err != nil {
		return 0, mv, err
	}
	x8 := i8 & 1
	y8 := i8 >> 1
	mvIndex := int(col.MB2BXY[mbXY]) + x8*3 + y8*3*col.BStride
	if err := checkRange(len(col.MotionVal[0]), mvIndex, 1); err != nil {
		return 0, mv, err
	}

	ref := col.RefIndex[0][refIndex]
	list := 0
	if ref < 0 {
		if err := checkRange(len(col.RefIndex[1]), refIndex, 1); err != nil {
			return 0, mv, err
		}
		ref = col.RefIndex[1][refIndex]
		list = 1
	}
	ref0, err := temporalDirectMapColToList0(ctx, list, ref)
	if err != nil {
		return 0, mv, err
	}
	mv = col.MotionVal[list][mvIndex]
	return ref0, mv, nil
}

func temporalDirectMapColToList0(ctx h264DirectMotionContext, list int, ref int8) (int8, error) {
	if list < 0 || list > 1 || ref < 0 {
		return 0, ErrInvalidData
	}
	if int(ref) >= len(ctx.RefEntries[list]) || ctx.RefEntries[list][ref].frame == nil {
		return 0, ErrUnsupported
	}
	target := ctx.RefEntries[list][ref]
	for i, entry := range ctx.RefEntries[0] {
		if entry.frame == target.frame {
			return int8(i), nil
		}
	}
	return 0, ErrUnsupported
}

func temporalDirectDistScaleFactor(ctx h264DirectMotionContext, ref0 int8) (int, error) {
	if ref0 < 0 || int(ref0) >= len(ctx.RefEntries[0]) || len(ctx.RefEntries[1]) == 0 ||
		ctx.RefEntries[0][ref0].frame == nil || ctx.RefEntries[1][0].frame == nil {
		return 0, ErrInvalidData
	}
	list0 := ctx.RefEntries[0][ref0]
	poc0 := list0.frame.poc
	poc1 := ctx.RefEntries[1][0].frame.poc
	td := clipInt(int(int64(poc1)-int64(poc0)), -128, 127)
	if td == 0 || list0.long {
		return 256, nil
	}
	tb := clipInt(int(int64(ctx.CurPOC)-int64(poc0)), -128, 127)
	tx := (16384 + (absInt(td) >> 1)) / td
	return clipInt((tb*tx+32)>>6, -1024, 1023), nil
}

func temporalDirectScaleMV(scale int, mvCol [2]int16) ([2]int16, [2]int16) {
	mx := (scale*int(mvCol[0]) + 128) >> 8
	my := (scale*int(mvCol[1]) + 128) >> 8
	mv0 := [2]int16{int16(mx), int16(my)}
	mv1 := [2]int16{int16(mx - int(mvCol[0])), int16(my - int(mvCol[1]))}
	return mv0, mv1
}
