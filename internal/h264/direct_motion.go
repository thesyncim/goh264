// SPDX-License-Identifier: LGPL-2.1-or-later
//
// Source-shaped frame/field subset of FFmpeg n8.0.1
// libavcodec/h264_direct.c pred_spatial_direct_motion,
// pred_temp_direct_motion, and ff_h264_direct_dist_scale_factor. Full MBAFF
// row-progress waits and field colmap parity remain outside this slice.

package h264

import "fmt"

type h264DirectMotionContext struct {
	RefEntries          [2][]simpleRefEntry
	CurPOC              int32
	PictureStructure    int32
	DirectSpatialMVPred bool
	Direct8x8Inference  bool
	X264Build           int32
}

type directColocatedLayout struct {
	MBXY               int
	MBTypeCol          [2]uint32
	B8Stride           int
	B4Stride           int
	RefBase            int
	MVBase             int
	CurInterlaced      bool
	ColInterlaced      bool
	InterlacedMismatch bool
}

func (m *macroblockTables) predDirectMotionFrame(cache *macroblockMotionCache, mbXY int, mbType *uint32, subMBType *[4]uint32, ctx h264DirectMotionContext) error {
	if m == nil || cache == nil || mbType == nil || subMBType == nil {
		return ErrInvalidData
	}
	if err := m.checkCodedMBXY(mbXY); err != nil {
		return err
	}
	if len(ctx.RefEntries[0]) == 0 || len(ctx.RefEntries[1]) == 0 || ctx.RefEntries[1][0].frame == nil {
		return unsupportedDirectMotion("missing refs", mbXY, *mbType, 0, ctx)
	}
	col := ctx.RefEntries[1][0].frame
	colTables := col.tables
	if colTables == nil || colTables.MBWidth != m.MBWidth || colTables.MBHeight != m.MBHeight || colTables.BStride != m.BStride {
		return unsupportedDirectMotion("missing colocated tables", mbXY, *mbType, 0, ctx)
	}
	if err := colTables.checkCodedMBXY(mbXY); err != nil {
		return err
	}
	if err := checkRange(len(colTables.RefIndex[0]), 4*mbXY, 4); err != nil {
		return err
	}

	layout, err := m.directColocatedLayout(colTables, mbXY, *mbType, ctx)
	if err != nil {
		return err
	}
	if ctx.DirectSpatialMVPred {
		return predSpatialDirectMotionFrame(cache, colTables, layout, mbType, subMBType, ctx)
	}

	isB8x8 := is8x8(*mbType)
	directSubType := MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2
	if layout.InterlacedMismatch {
		if !ctx.Direct8x8Inference {
			return unsupportedDirectMotion("interlaced direct without 8x8 inference", mbXY, *mbType, layout.MBTypeCol[0], ctx)
		}
		if !isB8x8 &&
			(layout.MBTypeCol[0]&(MBType16x16|MBTypeIntra4x4|MBTypeIntra16x16|MBTypeIntraPCM)) != 0 &&
			(layout.MBTypeCol[1]&(MBType16x16|MBTypeIntra4x4|MBTypeIntra16x16|MBTypeIntraPCM)) != 0 {
			*mbType |= MBType16x8 | MBTypeL0L1 | MBTypeDirect2
		} else {
			*mbType |= MBType8x8 | MBTypeL0L1
		}
		return predTemporalDirect8x8(cache, colTables, layout, *mbType, directSubType, subMBType, ctx, isB8x8)
	}
	if !isB8x8 && (layout.MBTypeCol[0]&(MBType16x16|MBTypeIntra4x4|MBTypeIntra16x16|MBTypeIntraPCM)) != 0 {
		*mbType |= MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2
		return predTemporalDirect16x16(cache, colTables, layout, ctx)
	}
	if !isB8x8 && (layout.MBTypeCol[0]&(MBType16x8|MBType8x16)) != 0 {
		*mbType |= MBTypeL0L1 | MBTypeDirect2 | (layout.MBTypeCol[0] & (MBType16x8 | MBType8x16))
	} else {
		if !ctx.Direct8x8Inference {
			directSubType = MBType8x8 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2
		}
		*mbType |= MBType8x8 | MBTypeL0L1
	}
	return predTemporalDirect8x8(cache, colTables, layout, *mbType, directSubType, subMBType, ctx, isB8x8)
}

func (m *macroblockTables) directColocatedLayout(col *macroblockTables, mbXY int, mbType uint32, ctx h264DirectMotionContext) (directColocatedLayout, error) {
	var layout directColocatedLayout
	if m == nil || col == nil {
		return layout, ErrInvalidData
	}
	if err := m.checkCodedMBXY(mbXY); err != nil {
		return layout, err
	}
	mbX := mbXY % m.MBStride
	mbY := mbXY / m.MBStride
	layout = directColocatedLayout{
		MBXY:          mbXY,
		B8Stride:      2,
		B4Stride:      col.BStride,
		CurInterlaced: mbType&MBTypeInterlaced != 0,
	}
	if err := col.checkCodedMBXY(layout.MBXY); err != nil {
		return layout, err
	}
	layout.ColInterlaced = col.MacroblockTyp[layout.MBXY]&MBTypeInterlaced != 0

	if layout.ColInterlaced {
		if !layout.CurInterlaced {
			parity := directColocatedParity(ctx)
			layout.MBXY = mbX + ((mbY&^1)+parity)*m.MBStride
			layout.B8Stride = 0
		} else {
			layout.MBXY += m.MBStride * directColocatedFieldOffset(ctx)
		}
		if err := col.checkCodedMBXY(layout.MBXY); err != nil {
			return layout, err
		}
		layout.MBTypeCol[0] = col.MacroblockTyp[layout.MBXY]
		layout.MBTypeCol[1] = layout.MBTypeCol[0]
	} else if layout.CurInterlaced {
		layout.MBXY = mbX + (mbY&^1)*m.MBStride
		if err := col.checkCodedMBXY(layout.MBXY); err != nil {
			return layout, err
		}
		if err := col.checkCodedMBXY(layout.MBXY + m.MBStride); err != nil {
			return layout, err
		}
		layout.MBTypeCol[0] = col.MacroblockTyp[layout.MBXY]
		layout.MBTypeCol[1] = col.MacroblockTyp[layout.MBXY+m.MBStride]
		layout.B8Stride = 2 + 4*m.MBStride
		layout.B4Stride *= 6
		if layout.MBTypeCol[0]&MBTypeInterlaced != layout.MBTypeCol[1]&MBTypeInterlaced {
			layout.MBTypeCol[0] &^= MBTypeInterlaced
			layout.MBTypeCol[1] &^= MBTypeInterlaced
		}
	} else {
		layout.MBTypeCol[0] = col.MacroblockTyp[layout.MBXY]
		layout.MBTypeCol[1] = layout.MBTypeCol[0]
	}

	if err := checkRange(len(col.MB2BXY), layout.MBXY, 1); err != nil {
		return layout, err
	}
	layout.RefBase = 4 * layout.MBXY
	layout.MVBase = int(col.MB2BXY[layout.MBXY])
	if layout.B8Stride == 0 && mbY&1 != 0 {
		layout.RefBase += 2
		layout.MVBase += 2 * layout.B4Stride
	}
	layout.InterlacedMismatch = layout.CurInterlaced != (layout.MBTypeCol[0]&MBTypeInterlaced != 0)
	return layout, nil
}

func directColocatedParity(ctx h264DirectMotionContext) int {
	if ctx.PictureStructure != PictureFrame || len(ctx.RefEntries[1]) == 0 || ctx.RefEntries[1][0].frame == nil {
		return 1
	}
	frame := ctx.RefEntries[1][0].frame
	top := frame.fieldPOC[0]
	bottom := frame.fieldPOC[1]
	if top == 0 && bottom == 0 {
		return 0
	}
	if absInt(int(int64(top)-int64(ctx.CurPOC))) >= absInt(int(int64(bottom)-int64(ctx.CurPOC))) {
		return 1
	}
	return 0
}

func directColocatedFieldOffset(ctx h264DirectMotionContext) int {
	if len(ctx.RefEntries[1]) == 0 {
		return 0
	}
	refPicture := ctx.RefEntries[1][0].pictureStructure
	if ctx.PictureStructure != PictureTopField && ctx.PictureStructure != PictureBottomField {
		return 0
	}
	if refPicture != PictureTopField && refPicture != PictureBottomField {
		return 0
	}
	if ctx.PictureStructure&refPicture != 0 {
		return 0
	}
	return 2*int(refPicture) - 3
}

func unsupportedDirectMotion(reason string, mbXY int, mbType uint32, colType uint32, ctx h264DirectMotionContext) error {
	ref1Picture := int32(0)
	if len(ctx.RefEntries[1]) != 0 {
		ref1Picture = ctx.RefEntries[1][0].pictureStructure
	}
	return fmt.Errorf("direct motion %s mb_xy=%d mb_type=%#x col_type=%#x picture=%d ref1_picture=%d: %w",
		reason, mbXY, mbType, colType, ctx.PictureStructure, ref1Picture, ErrUnsupported)
}

func predSpatialDirectMotionFrame(cache *macroblockMotionCache, col *macroblockTables, layout directColocatedLayout, mbType *uint32, subMBType *[4]uint32, ctx h264DirectMotionContext) error {
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

	directSubType |= MBType16x16 | MBTypeDirect2
	if !isB8x8 && (layout.MBTypeCol[0]&(MBType16x16|MBTypeIntra4x4|MBTypeIntra16x16|MBTypeIntraPCM)) != 0 {
		*mbType |= MBType16x16 | MBTypeDirect2
	} else if !isB8x8 && (layout.MBTypeCol[0]&(MBType16x8|MBType8x16)) != 0 {
		*mbType |= MBTypeDirect2 | (layout.MBTypeCol[0] & (MBType16x8 | MBType8x16))
	} else {
		if !ctx.Direct8x8Inference {
			directSubType = (directSubType &^ MBType16x16) | MBType8x8
		}
		*mbType |= MBType8x8
	}

	if is16x16(*mbType) {
		return predSpatialDirect16x16(cache, col, layout, ref, mv, ctx)
	}
	return predSpatialDirect8x8(cache, col, layout, mbType, directSubType, subMBType, ref, mv, ctx, isB8x8)
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

func predSpatialDirect16x16(cache *macroblockMotionCache, col *macroblockTables, layout directColocatedLayout, ref [2]int8, mv [2][2]int16, ctx h264DirectMotionContext) error {
	base := int(h264Scan8[0])
	fillRefRectangle(&cache.Ref[0], base, 4, 4, 8, ref[0])
	fillRefRectangle(&cache.Ref[1], base, 4, 4, 8, ref[1])
	mv0, mv1 := mv[0], mv[1]
	if spatialDirectColZero(col, layout, 0, ctx) {
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

func predSpatialDirect8x8(cache *macroblockMotionCache, col *macroblockTables, layout directColocatedLayout, mbType *uint32, directSubType uint32, subMBType *[4]uint32, ref [2]int8, mv [2][2]int16, ctx h264DirectMotionContext, wasB8x8 bool) error {
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
		if isSub8x8(directSubType) && spatialDirectColZero(col, layout, i8, ctx) {
			if ref[0] == 0 {
				fillMotionRectangle(&cache.MV[0], base, 2, 2, 8, [2]int16{})
			}
			if ref[1] == 0 {
				fillMotionRectangle(&cache.MV[1], base, 2, 2, 8, [2]int16{})
			}
			n += 4
		} else if isSub4x4(directSubType) {
			list, ok := spatialDirectColZeroList(col, layout, i8, ctx)
			if ok {
				m := 0
				for i4 := 0; i4 < 4; i4++ {
					mvCol, ok := spatialDirectColocatedSub4x4MV(col, layout, i8, i4, list)
					if !ok || absInt(int(mvCol[0])) > 1 || absInt(int(mvCol[1])) > 1 {
						continue
					}
					dst := h264Scan8[4*i8+i4]
					if ref[0] == 0 {
						cache.MV[0][dst] = [2]int16{}
					}
					if ref[1] == 0 {
						cache.MV[1][dst] = [2]int16{}
					}
					m++
				}
				if m&3 == 0 {
					subMBType[i8] = (subMBType[i8] &^ MBType8x8) | MBType16x16
				}
				n += m
			}
		}
	}
	if !wasB8x8 && n&15 == 0 {
		*mbType = (*mbType &^ (MBType8x8 | MBType16x8 | MBType8x16 | MBTypeP1L0 | MBTypeP1L1)) | MBType16x16 | MBTypeDirect2
	}
	return nil
}

func spatialDirectColZero(col *macroblockTables, layout directColocatedLayout, i8 int, ctx h264DirectMotionContext) bool {
	list, ok := spatialDirectColZeroList(col, layout, i8, ctx)
	if !ok {
		return false
	}
	mv, ok := spatialDirectColocatedSub8x8MV(col, layout, i8, list)
	return ok && absInt(int(mv[0])) <= 1 && absInt(int(mv[1])) <= 1
}

func spatialDirectColZeroList(col *macroblockTables, layout directColocatedLayout, i8 int, ctx h264DirectMotionContext) (int, bool) {
	if col == nil || i8 < 0 || i8 > 3 || layout.MBXY < 0 || layout.MBXY >= len(col.MacroblockTyp) ||
		isIntra(layout.MBTypeCol[i8>>1]) || len(ctx.RefEntries[1]) == 0 || ctx.RefEntries[1][0].long {
		return 0, false
	}
	refIndex := directColocatedRefIndex(layout, i8)
	if refIndex < 0 || refIndex >= len(col.RefIndex[0]) || refIndex >= len(col.RefIndex[1]) {
		return 0, false
	}
	if col.RefIndex[0][refIndex] == 0 {
		return 0, true
	}
	if col.RefIndex[0][refIndex] < 0 && col.RefIndex[1][refIndex] == 0 && uint32(ctx.X264Build) > 33 {
		return 1, true
	}
	return 0, false
}

func spatialDirectColocatedSub8x8MV(col *macroblockTables, layout directColocatedLayout, i8 int, list int) ([2]int16, bool) {
	var mv [2]int16
	if col == nil || list < 0 || list > 1 || i8 < 0 || i8 > 3 {
		return mv, false
	}
	x8 := i8 & 1
	y8 := i8 >> 1
	mvIndex := layout.MVBase + x8*3 + y8*directColocatedSub8x8RowStride(layout)
	if mvIndex < 0 || mvIndex >= len(col.MotionVal[list]) {
		return mv, false
	}
	return col.MotionVal[list][mvIndex], true
}

func spatialDirectColocatedSub4x4MV(col *macroblockTables, layout directColocatedLayout, i8 int, i4 int, list int) ([2]int16, bool) {
	var mv [2]int16
	if col == nil || list < 0 || list > 1 || i8 < 0 || i8 > 3 || i4 < 0 || i4 > 3 {
		return mv, false
	}
	x8 := i8 & 1
	y8 := i8 >> 1
	mvIndex := layout.MVBase + x8*2 + (i4 & 1) + (y8*2+(i4>>1))*layout.B4Stride
	if mvIndex < 0 || mvIndex >= len(col.MotionVal[list]) {
		return mv, false
	}
	return col.MotionVal[list][mvIndex], true
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

func predTemporalDirect16x16(cache *macroblockMotionCache, col *macroblockTables, layout directColocatedLayout, ctx h264DirectMotionContext) error {
	base := int(h264Scan8[0])
	fillRefRectangle(&cache.Ref[1], base, 4, 4, 8, 0)
	if isIntra(layout.MBTypeCol[0]) {
		fillRefRectangle(&cache.Ref[0], base, 4, 4, 8, 0)
		fillMotionRectangle(&cache.MV[0], base, 4, 4, 8, [2]int16{})
		fillMotionRectangle(&cache.MV[1], base, 4, 4, 8, [2]int16{})
		return nil
	}

	ref0, list, err := temporalDirectColocatedRefListAt(col, layout.RefBase, ctx)
	if err != nil {
		return err
	}
	mvCol, err := temporalDirectColocatedSub8x8MV(col, layout, 0, list)
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

func predTemporalDirect8x8(cache *macroblockMotionCache, col *macroblockTables, layout directColocatedLayout, mbType uint32, directSubType uint32, subMBType *[4]uint32, ctx h264DirectMotionContext, wasB8x8 bool) error {
	if !is8x8(mbType) && !is16x8(mbType) && !is8x16(mbType) {
		return ErrInvalidData
	}
	for i8 := 0; i8 < 4; i8++ {
		if wasB8x8 && !isDirect(subMBType[i8]) {
			continue
		}
		subMBType[i8] = directSubType
		base := int(h264Scan8[4*i8])
		fillRefRectangle(&cache.Ref[1], base, 2, 2, 8, 0)
		if isIntra(layout.MBTypeCol[i8>>1]) {
			fillRefRectangle(&cache.Ref[0], base, 2, 2, 8, 0)
			fillMotionRectangle(&cache.MV[0], base, 2, 2, 8, [2]int16{})
			fillMotionRectangle(&cache.MV[1], base, 2, 2, 8, [2]int16{})
			continue
		}
		refIndex := directColocatedRefIndex(layout, i8)
		ref0, list, err := temporalDirectColocatedRefListAt(col, refIndex, ctx)
		if err != nil {
			return err
		}
		scale, err := temporalDirectDistScaleFactor(ctx, ref0)
		if err != nil {
			return err
		}
		fillRefRectangle(&cache.Ref[0], base, 2, 2, 8, ref0)
		if layout.InterlacedMismatch {
			mvCol, err := temporalDirectColocatedSub8x8MV(col, layout, i8, list)
			if err != nil {
				return err
			}
			yShift := 0
			if !layout.CurInterlaced {
				yShift = 2
			}
			myCol := int16((int(mvCol[1]) * (1 << yShift)) / 2)
			mv0, _ := temporalDirectScaleMV(scale, [2]int16{mvCol[0], myCol})
			fillMotionRectangle(&cache.MV[0], base, 2, 2, 8, mv0)
			fillMotionRectangle(&cache.MV[1], base, 2, 2, 8, [2]int16{int16(int(mv0[0]) - int(mvCol[0])), int16(int(mv0[1]) - int(myCol))})
			continue
		}
		if isSub8x8(directSubType) {
			mvCol, err := temporalDirectColocatedSub8x8MV(col, layout, i8, list)
			if err != nil {
				return err
			}
			mv0, mv1 := temporalDirectScaleMV(scale, mvCol)
			fillMotionRectangle(&cache.MV[0], base, 2, 2, 8, mv0)
			fillMotionRectangle(&cache.MV[1], base, 2, 2, 8, mv1)
			continue
		}
		for i4 := 0; i4 < 4; i4++ {
			mvCol, err := temporalDirectColocatedSub4x4MV(col, layout, i8, i4, list)
			if err != nil {
				return err
			}
			mv0, mv1 := temporalDirectScaleMV(scale, mvCol)
			dst := h264Scan8[4*i8+i4]
			cache.MV[0][dst] = mv0
			cache.MV[1][dst] = mv1
		}
	}
	return nil
}

func temporalDirectColocatedRefList(col *macroblockTables, mbXY int, i8 int, ctx h264DirectMotionContext) (int8, int, error) {
	if col == nil || i8 < 0 || i8 > 3 {
		return 0, 0, ErrInvalidData
	}
	return temporalDirectColocatedRefListAt(col, 4*mbXY+i8, ctx)
}

func temporalDirectColocatedRefListAt(col *macroblockTables, refIndex int, ctx h264DirectMotionContext) (int8, int, error) {
	if col == nil {
		return 0, 0, ErrInvalidData
	}
	if err := checkRange(len(col.RefIndex[0]), refIndex, 1); err != nil {
		return 0, 0, err
	}

	ref := col.RefIndex[0][refIndex]
	list := 0
	if ref < 0 {
		if err := checkRange(len(col.RefIndex[1]), refIndex, 1); err != nil {
			return 0, 0, err
		}
		ref = col.RefIndex[1][refIndex]
		list = 1
	}
	ref0, err := temporalDirectMapColToList0(ctx, list, ref)
	if err != nil {
		return 0, 0, err
	}
	return ref0, list, nil
}

func temporalDirectColocatedSub8x8MV(col *macroblockTables, layout directColocatedLayout, i8 int, list int) ([2]int16, error) {
	var mv [2]int16
	if col == nil || i8 < 0 || i8 > 3 || list < 0 || list > 1 {
		return mv, ErrInvalidData
	}
	x8 := i8 & 1
	y8 := i8 >> 1
	mvIndex := layout.MVBase + x8*3 + y8*directColocatedSub8x8RowStride(layout)
	if err := checkRange(len(col.MotionVal[list]), mvIndex, 1); err != nil {
		return mv, err
	}
	return col.MotionVal[list][mvIndex], nil
}

func temporalDirectColocatedSub4x4MV(col *macroblockTables, layout directColocatedLayout, i8 int, i4 int, list int) ([2]int16, error) {
	var mv [2]int16
	if col == nil || i8 < 0 || i8 > 3 || i4 < 0 || i4 > 3 || list < 0 || list > 1 {
		return mv, ErrInvalidData
	}
	x8 := i8 & 1
	y8 := i8 >> 1
	mvIndex := layout.MVBase + x8*2 + (i4 & 1) + (y8*2+(i4>>1))*layout.B4Stride
	if err := checkRange(len(col.MotionVal[list]), mvIndex, 1); err != nil {
		return mv, err
	}
	return col.MotionVal[list][mvIndex], nil
}

func directColocatedRefIndex(layout directColocatedLayout, i8 int) int {
	x8 := i8 & 1
	y8 := i8 >> 1
	if layout.InterlacedMismatch {
		return layout.RefBase + x8 + y8*layout.B8Stride
	}
	return layout.RefBase + i8
}

func directColocatedSub8x8RowStride(layout directColocatedLayout) int {
	if layout.InterlacedMismatch {
		return layout.B4Stride
	}
	return 3 * layout.B4Stride
}

func temporalDirectMapColToList0(ctx h264DirectMotionContext, list int, ref int8) (int8, error) {
	if list < 0 || list > 1 || ref < 0 {
		return 0, ErrInvalidData
	}
	if len(ctx.RefEntries[0]) == 0 {
		return 0, ErrInvalidData
	}
	target, ok := temporalDirectColocatedRefEntry(ctx, list, int(ref))
	if !ok {
		return 0, ErrUnsupported
	}
	for i, entry := range ctx.RefEntries[0] {
		if entry.frame != nil && target.frame != nil && entry.frame == target.frame {
			return int8(i), nil
		}
		if temporalDirectSamePictureID(entry, target) {
			return int8(i), nil
		}
	}
	return 0, nil
}

func temporalDirectColocatedRefEntry(ctx h264DirectMotionContext, list int, ref int) (simpleRefEntry, bool) {
	if len(ctx.RefEntries[1]) != 0 && ctx.RefEntries[1][0].frame != nil {
		colEntries := ctx.RefEntries[1][0].frame.refEntries[list]
		if ref < len(colEntries) {
			return colEntries[ref], true
		}
	}
	if ref < len(ctx.RefEntries[list]) {
		return ctx.RefEntries[list][ref], true
	}
	return simpleRefEntry{}, false
}

func temporalDirectSamePictureID(a simpleRefEntry, b simpleRefEntry) bool {
	if a.long != b.long || a.picID != b.picID {
		return false
	}
	return a.frame != nil || b.frame != nil || a.long || a.picID != 0
}

func temporalDirectDistScaleFactor(ctx h264DirectMotionContext, ref0 int8) (int, error) {
	if ref0 < 0 || int(ref0) >= len(ctx.RefEntries[0]) || len(ctx.RefEntries[1]) == 0 ||
		ctx.RefEntries[0][ref0].frame == nil || ctx.RefEntries[1][0].frame == nil {
		return 0, ErrInvalidData
	}
	list0 := ctx.RefEntries[0][ref0]
	poc0 := directRefPOC(list0)
	poc1 := directRefPOC(ctx.RefEntries[1][0])
	td := clipInt(int(int64(poc1)-int64(poc0)), -128, 127)
	if td == 0 || list0.long {
		return 256, nil
	}
	tb := clipInt(int(int64(ctx.CurPOC)-int64(poc0)), -128, 127)
	tx := (16384 + (absInt(td) >> 1)) / td
	return clipInt((tb*tx+32)>>6, -1024, 1023), nil
}

func directRefPOC(entry simpleRefEntry) int32 {
	if entry.pictureStructure == 0 && entry.poc == 0 && entry.frame != nil {
		return entry.frame.poc
	}
	return entry.poc
}

func temporalDirectScaleMV(scale int, mvCol [2]int16) ([2]int16, [2]int16) {
	mx := (scale*int(mvCol[0]) + 128) >> 8
	my := (scale*int(mvCol[1]) + 128) >> 8
	mv0 := [2]int16{int16(mx), int16(my)}
	mv1 := [2]int16{int16(mx - int(mvCol[0])), int16(my - int(mvCol[1]))}
	return mv0, mv1
}
