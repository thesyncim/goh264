// SPDX-License-Identifier: LGPL-2.1-or-later
//
// Source-shaped port of the frame-MB motion prediction pieces of FFmpeg n8.0.1
// libavcodec/h264_mvpred.h pred_motion, pred_16x8_motion, pred_8x16_motion,
// pred_pskip_motion, and the inter CAVLC motion-cache fill portions of
// libavcodec/h264_cavlc.c ff_h264_decode_mb_cavlc.

package h264

func midPred(a int, b int, c int) int {
	if a > b {
		if c > b {
			if c > a {
				b = a
			} else {
				b = c
			}
		}
	} else if b > c {
		if c > a {
			b = c
		} else {
			b = a
		}
	}
	return b
}

func fetchDiagonalMV(cache *macroblockMotionCache, i int, list int, partWidth int) ([2]int16, int8, error) {
	return fetchDiagonalMVWithContext(cache, i, list, partWidth, nil)
}

type h264MotionPredContext struct {
	Tables     *macroblockTables
	MBXY       int
	FrameMBAFF bool
	Neighbors  motionDecodeNeighbors
}

func (m *macroblockTables) frameMotionPredContext(mbXY int, frameMBAFF bool, neighbors macroblockDecodeNeighbors, mbType uint32, listCount int, sliceTypeNoS int32, cabac bool, directSpatial bool) *h264MotionPredContext {
	if m == nil || !frameMBAFF {
		return nil
	}
	motionNeighbors := neighbors.motionNeighbors(mbType, listCount, sliceTypeNoS, cabac, directSpatial)
	motionNeighbors.FrameMBAFF = true
	return &h264MotionPredContext{
		Tables:     m,
		MBXY:       mbXY,
		FrameMBAFF: true,
		Neighbors:  motionNeighbors,
	}
}

func fetchDiagonalMVWithContext(cache *macroblockMotionCache, i int, list int, partWidth int, predCtx *h264MotionPredContext) ([2]int16, int8, error) {
	var zero [2]int16
	if cache == nil || list < 0 || list > 1 || (partWidth != 1 && partWidth != 2 && partWidth != 4) {
		return zero, 0, ErrInvalidData
	}
	topRight := i - 8 + partWidth
	if err := checkRange(h264MotionCacheSize, topRight, 1); err != nil {
		return zero, 0, err
	}
	topRightRef := cache.Ref[list][topRight]
	if mv, ref, ok, err := fetchDiagonalMVMBAFF(cache, i, list, topRightRef, predCtx); ok || err != nil {
		return mv, ref, err
	}
	if topRightRef != h264PartNotAvailable {
		return cache.MV[list][topRight], topRightRef, nil
	}

	topLeft := i - 8 - 1
	if err := checkRange(h264MotionCacheSize, topLeft, 1); err != nil {
		return zero, 0, err
	}
	return cache.MV[list][topLeft], cache.Ref[list][topLeft], nil
}

func fetchDiagonalMVMBAFF(cache *macroblockMotionCache, i int, list int, topRightRef int8, predCtx *h264MotionPredContext) ([2]int16, int8, bool, error) {
	var zero [2]int16
	if predCtx == nil || !predCtx.FrameMBAFF || predCtx.Tables == nil || cache == nil {
		return zero, 0, false, nil
	}
	base := int(h264Scan8[0])
	if topRightRef != h264PartNotAvailable || i < base+8 || (i&7) != 4 || cache.Ref[list][base-1] == h264PartNotAvailable {
		return zero, 0, false, nil
	}
	mbField := predCtx.Neighbors.MBType&MBTypeInterlaced != 0
	leftField := predCtx.Neighbors.LeftType[h264LeftTop]&MBTypeInterlaced != 0
	if !mbField && leftField {
		xy := predCtx.Neighbors.LeftXY[h264LeftTop] + predCtx.Tables.MBStride
		y4 := ((predCtx.MBXY / predCtx.Tables.MBStride) & 1) * 2
		y4 += i >> 5
		return predCtx.fetchDiagonalMVMBAFFSource(list, xy, y4, true)
	}
	if mbField && !leftField {
		left := h264LeftTop
		if i >= 36 {
			left = h264LeftBot
		}
		xy := predCtx.Neighbors.LeftXY[left]
		y4 := (i >> 2) & 3
		return predCtx.fetchDiagonalMVMBAFFSource(list, xy, y4, false)
	}
	return zero, 0, false, nil
}

func (predCtx *h264MotionPredContext) fetchDiagonalMVMBAFFSource(list int, xy int, y4 int, frameFromField bool) ([2]int16, int8, bool, error) {
	var zero [2]int16
	m := predCtx.Tables
	if m == nil || list < 0 || list > 1 || y4 < 0 {
		return zero, 0, true, ErrInvalidData
	}
	typeXY := xy + (y4>>2)*m.MBStride
	if err := m.checkMBXY(typeXY); err != nil {
		return zero, 0, true, err
	}
	if !usesList(m.MacroblockTyp[typeXY], list) {
		return zero, h264ListNotUsed, true, nil
	}
	if err := m.checkMBXY(xy); err != nil {
		return zero, 0, true, err
	}
	mvIdx := int(m.MB2BXY[xy]) + 3 + y4*m.BStride
	if err := checkRange(len(m.MotionVal[list]), mvIdx, 1); err != nil {
		return zero, 0, true, err
	}
	refIdx := 4*xy + 1 + (y4 &^ 1)
	if err := checkRange(len(m.RefIndex[list]), refIdx, 1); err != nil {
		return zero, 0, true, err
	}
	mv := m.MotionVal[list][mvIdx]
	ref := m.RefIndex[list][refIdx]
	if frameFromField {
		ref >>= 1
		mv[1] = int16(int(mv[1]) * 2)
	} else {
		ref = int8(int(ref) * 2)
		mv[1] /= 2
	}
	return mv, ref, true, nil
}

func predMotion(cache *macroblockMotionCache, n int, partWidth int, list int, ref int8) ([2]int16, error) {
	return predMotionWithContext(cache, n, partWidth, list, ref, nil)
}

func predMotionWithContext(cache *macroblockMotionCache, n int, partWidth int, list int, ref int8, predCtx *h264MotionPredContext) ([2]int16, error) {
	var pred [2]int16
	if cache == nil || n < 0 || n >= 16 || list < 0 || list > 1 || (partWidth != 1 && partWidth != 2 && partWidth != 4) {
		return pred, ErrInvalidData
	}

	index8 := int(h264Scan8[n])
	topRef := cache.Ref[list][index8-8]
	leftRef := cache.Ref[list][index8-1]
	a := cache.MV[list][index8-1]
	b := cache.MV[list][index8-8]
	c, diagonalRef, err := fetchDiagonalMVWithContext(cache, index8, list, partWidth, predCtx)
	if err != nil {
		return pred, err
	}

	matchCount := boolToInt(diagonalRef == ref) + boolToInt(topRef == ref) + boolToInt(leftRef == ref)
	if matchCount > 1 {
		pred[0] = int16(midPred(int(a[0]), int(b[0]), int(c[0])))
		pred[1] = int16(midPred(int(a[1]), int(b[1]), int(c[1])))
	} else if matchCount == 1 {
		if leftRef == ref {
			pred = a
		} else if topRef == ref {
			pred = b
		} else {
			pred = c
		}
	} else if topRef == h264PartNotAvailable &&
		diagonalRef == h264PartNotAvailable &&
		leftRef != h264PartNotAvailable {
		pred = a
	} else {
		pred[0] = int16(midPred(int(a[0]), int(b[0]), int(c[0])))
		pred[1] = int16(midPred(int(a[1]), int(b[1]), int(c[1])))
	}
	return pred, nil
}

func pred16x8Motion(cache *macroblockMotionCache, n int, list int, ref int8) ([2]int16, error) {
	return pred16x8MotionWithContext(cache, n, list, ref, nil)
}

func pred16x8MotionWithContext(cache *macroblockMotionCache, n int, list int, ref int8, predCtx *h264MotionPredContext) ([2]int16, error) {
	var pred [2]int16
	if cache == nil || list < 0 || list > 1 {
		return pred, ErrInvalidData
	}
	if n == 0 {
		index := int(h264Scan8[0]) - 8
		if cache.Ref[list][index] == ref {
			return cache.MV[list][index], nil
		}
	} else {
		index := int(h264Scan8[8]) - 1
		if cache.Ref[list][index] == ref {
			return cache.MV[list][index], nil
		}
	}
	return predMotionWithContext(cache, n, 4, list, ref, predCtx)
}

func pred8x16Motion(cache *macroblockMotionCache, n int, list int, ref int8) ([2]int16, error) {
	return pred8x16MotionWithContext(cache, n, list, ref, nil)
}

func pred8x16MotionWithContext(cache *macroblockMotionCache, n int, list int, ref int8, predCtx *h264MotionPredContext) ([2]int16, error) {
	var pred [2]int16
	if cache == nil || list < 0 || list > 1 {
		return pred, ErrInvalidData
	}
	if n == 0 {
		index := int(h264Scan8[0]) - 1
		if cache.Ref[list][index] == ref {
			return cache.MV[list][index], nil
		}
	} else {
		c, diagonalRef, err := fetchDiagonalMVWithContext(cache, int(h264Scan8[4]), list, 2, predCtx)
		if err != nil {
			return pred, err
		}
		if diagonalRef == ref {
			return c, nil
		}
	}
	return predMotionWithContext(cache, n, 2, list, ref, predCtx)
}

func (m *macroblockTables) predPSkipMotion(cache *macroblockMotionCache, n motionDecodeNeighbors) error {
	if m == nil || cache == nil {
		return ErrInvalidData
	}
	leftBlock := &h264LeftBlockFrame
	if n.LeftBlock != nil {
		leftBlock = n.LeftBlock
	}
	base := int(h264Scan8[0])
	fillRefRectangle(&cache.Ref[0], base, 4, 4, 8, 0)

	var a, b, c [2]int16
	var leftRef, topRef, diagonalRef int8

	if usesList(n.LeftType[h264LeftTop], 0) {
		if err := m.checkMBXY(n.LeftXY[h264LeftTop]); err != nil {
			return err
		}
		refIdx := 4*n.LeftXY[h264LeftTop] + 1 + int(leftBlock[0]&^1)
		if err := checkRange(len(m.RefIndex[0]), refIdx, 1); err != nil {
			return err
		}
		mvIdx := int(m.MB2BXY[n.LeftXY[h264LeftTop]]) + 3 + m.BStride*int(leftBlock[0])
		if err := checkRange(len(m.MotionVal[0]), mvIdx, 1); err != nil {
			return err
		}
		leftRef = m.RefIndex[0][refIdx]
		a = m.MotionVal[0][mvIdx]
		leftRef, a = h264FixPskipMVForMBAFF(n, n.LeftType[h264LeftTop], leftRef, a)
		if refAndMVZero(leftRef, a) {
			fillMotionRectangle(&cache.MV[0], base, 4, 4, 8, [2]int16{})
			return nil
		}
	} else if n.LeftType[h264LeftTop] != 0 {
		leftRef = h264ListNotUsed
	} else {
		fillMotionRectangle(&cache.MV[0], base, 4, 4, 8, [2]int16{})
		return nil
	}

	if usesList(n.TopType, 0) {
		if err := m.checkMBXY(n.TopXY); err != nil {
			return err
		}
		refIdx := 4*n.TopXY + 2
		if err := checkRange(len(m.RefIndex[0]), refIdx, 1); err != nil {
			return err
		}
		mvIdx := int(m.MB2BXY[n.TopXY]) + 3*m.BStride
		if err := checkRange(len(m.MotionVal[0]), mvIdx, 1); err != nil {
			return err
		}
		topRef = m.RefIndex[0][refIdx]
		b = m.MotionVal[0][mvIdx]
		topRef, b = h264FixPskipMVForMBAFF(n, n.TopType, topRef, b)
		if refAndMVZero(topRef, b) {
			fillMotionRectangle(&cache.MV[0], base, 4, 4, 8, [2]int16{})
			return nil
		}
	} else if n.TopType != 0 {
		topRef = h264ListNotUsed
	} else {
		fillMotionRectangle(&cache.MV[0], base, 4, 4, 8, [2]int16{})
		return nil
	}

	if usesList(n.TopRightType, 0) {
		if err := m.checkMBXY(n.TopRightXY); err != nil {
			return err
		}
		refIdx := 4*n.TopRightXY + 2
		if err := checkRange(len(m.RefIndex[0]), refIdx, 1); err != nil {
			return err
		}
		mvIdx := int(m.MB2BXY[n.TopRightXY]) + 3*m.BStride
		if err := checkRange(len(m.MotionVal[0]), mvIdx, 1); err != nil {
			return err
		}
		diagonalRef = m.RefIndex[0][refIdx]
		c = m.MotionVal[0][mvIdx]
		diagonalRef, c = h264FixPskipMVForMBAFF(n, n.TopRightType, diagonalRef, c)
	} else if n.TopRightType != 0 {
		diagonalRef = h264ListNotUsed
	} else if usesList(n.TopLeftType, 0) {
		if err := m.checkMBXY(n.TopLeftXY); err != nil {
			return err
		}
		refIdx := 4*n.TopLeftXY + 1 + (n.TopLeftPartition & 2)
		if err := checkRange(len(m.RefIndex[0]), refIdx, 1); err != nil {
			return err
		}
		mvIdx := int(m.MB2BXY[n.TopLeftXY]) + 3 + m.BStride + (n.TopLeftPartition & (2 * m.BStride))
		if err := checkRange(len(m.MotionVal[0]), mvIdx, 1); err != nil {
			return err
		}
		diagonalRef = m.RefIndex[0][refIdx]
		c = m.MotionVal[0][mvIdx]
		diagonalRef, c = h264FixPskipMVForMBAFF(n, n.TopLeftType, diagonalRef, c)
	} else if n.TopLeftType != 0 {
		diagonalRef = h264ListNotUsed
	} else {
		diagonalRef = h264PartNotAvailable
	}

	var mv [2]int16
	matchCount := boolToInt(diagonalRef == 0) + boolToInt(topRef == 0) + boolToInt(leftRef == 0)
	if matchCount > 1 {
		mv[0] = int16(midPred(int(a[0]), int(b[0]), int(c[0])))
		mv[1] = int16(midPred(int(a[1]), int(b[1]), int(c[1])))
	} else if matchCount == 1 {
		if leftRef == 0 {
			mv = a
		} else if topRef == 0 {
			mv = b
		} else {
			mv = c
		}
	} else {
		mv[0] = int16(midPred(int(a[0]), int(b[0]), int(c[0])))
		mv[1] = int16(midPred(int(a[1]), int(b[1]), int(c[1])))
	}
	fillMotionRectangle(&cache.MV[0], base, 4, 4, 8, mv)
	return nil
}

func h264FixPskipMVForMBAFF(n motionDecodeNeighbors, neighborType uint32, ref int8, mv [2]int16) (int8, [2]int16) {
	if !n.FrameMBAFF {
		return ref, mv
	}
	mbField := n.MBType&MBTypeInterlaced != 0
	neighborField := neighborType&MBTypeInterlaced != 0
	if mbField && !neighborField {
		ref <<= 1
		mv[1] /= 2
	} else if !mbField && neighborField {
		ref >>= 1
		mv[1] = int16(int(mv[1]) * 2)
	}
	return ref, mv
}

func fillCAVLCInterMotionCache(cache *macroblockMotionCache, mb *cavlcInterMacroblockSyntax, listCount int) error {
	return fillCAVLCInterMotionCacheWithContext(cache, mb, listCount, nil)
}

func fillCAVLCInterMotionCacheWithContext(cache *macroblockMotionCache, mb *cavlcInterMacroblockSyntax, listCount int, predCtx *h264MotionPredContext) error {
	if cache == nil || mb == nil || listCount < 0 || listCount > 2 {
		return ErrInvalidData
	}
	if mb.PartitionCount == 4 {
		return fillCAVLCSubInterMotionCacheWithContext(cache, mb, listCount, predCtx)
	}
	if is16x16(mb.MBType) {
		for list := 0; list < listCount; list++ {
			if !isDir(mb.MBType, 0, list) {
				continue
			}
			ref := int8(mb.Ref[list][0])
			fillRefRectangle(&cache.Ref[list], int(h264Scan8[0]), 4, 4, 8, ref)
		}
		for list := 0; list < listCount; list++ {
			if !isDir(mb.MBType, 0, list) {
				continue
			}
			ref := int8(mb.Ref[list][0])
			pred, err := predMotionWithContext(cache, 0, 4, list, ref, predCtx)
			if err != nil {
				return err
			}
			fillMotionRectangle(&cache.MV[list], int(h264Scan8[0]), 4, 4, 8, addMVD(pred, mb.MVD[list][0]))
		}
		return nil
	}
	if is16x8(mb.MBType) {
		for list := 0; list < listCount; list++ {
			for i := 0; i < 2; i++ {
				ref := h264ListNotUsed
				if isDir(mb.MBType, i, list) {
					ref = int8(mb.Ref[list][i])
				}
				fillRefRectangle(&cache.Ref[list], int(h264Scan8[0])+16*i, 4, 2, 8, ref)
			}
		}
		for list := 0; list < listCount; list++ {
			for i := 0; i < 2; i++ {
				mv := [2]int16{}
				if isDir(mb.MBType, i, list) {
					ref := int8(mb.Ref[list][i])
					pred, err := pred16x8MotionWithContext(cache, 8*i, list, ref, predCtx)
					if err != nil {
						return err
					}
					mv = addMVD(pred, mb.MVD[list][8*i])
				}
				fillMotionRectangle(&cache.MV[list], int(h264Scan8[0])+16*i, 4, 2, 8, mv)
			}
		}
		return nil
	}
	if is8x16(mb.MBType) {
		for list := 0; list < listCount; list++ {
			for i := 0; i < 2; i++ {
				ref := h264ListNotUsed
				if isDir(mb.MBType, i, list) {
					ref = int8(mb.Ref[list][i])
				}
				fillRefRectangle(&cache.Ref[list], int(h264Scan8[0])+2*i, 2, 4, 8, ref)
			}
		}
		for list := 0; list < listCount; list++ {
			for i := 0; i < 2; i++ {
				mv := [2]int16{}
				if isDir(mb.MBType, i, list) {
					ref := int8(mb.Ref[list][i])
					pred, err := pred8x16MotionWithContext(cache, i*4, list, ref, predCtx)
					if err != nil {
						return err
					}
					mv = addMVD(pred, mb.MVD[list][4*i])
				}
				fillMotionRectangle(&cache.MV[list], int(h264Scan8[0])+2*i, 2, 4, 8, mv)
			}
		}
		return nil
	}
	return ErrUnsupported
}

func fillCAVLCSubInterMotionCache(cache *macroblockMotionCache, mb *cavlcInterMacroblockSyntax, listCount int) error {
	return fillCAVLCSubInterMotionCacheWithContext(cache, mb, listCount, nil)
}

func fillCAVLCSubInterMotionCacheWithContext(cache *macroblockMotionCache, mb *cavlcInterMacroblockSyntax, listCount int, predCtx *h264MotionPredContext) error {
	for list := 0; list < listCount; list++ {
		for i := 0; i < 4; i++ {
			start := int(h264Scan8[4*i])
			if isDirect(mb.SubMBType[i]) {
				cache.Ref[list][start] = cache.Ref[list][start+1]
				continue
			}
			ref := h264ListNotUsed
			if isDir(mb.SubMBType[i], 0, list) {
				ref = int8(mb.Ref[list][i])
			}
			fillRefRectangle(&cache.Ref[list], start, 2, 2, 8, ref)
			if !isDir(mb.SubMBType[i], 0, list) {
				fillMotionRectangle(&cache.MV[list], start, 2, 2, 8, [2]int16{})
				continue
			}
			blockWidth := 1
			if mb.SubMBType[i]&(MBType16x16|MBType16x8) != 0 {
				blockWidth = 2
			}
			for j := 0; j < int(mb.SubPartitionCount[i]); j++ {
				index := 4*i + blockWidth*j
				ref := cache.Ref[list][h264Scan8[index]]
				pred, err := predMotionWithContext(cache, index, blockWidth, list, ref, predCtx)
				if err != nil {
					return err
				}
				writeSubPartitionMV(cache, list, index, mb.SubMBType[i], addMVD(pred, mb.MVD[list][index]))
			}
		}
	}
	return nil
}

func writeSubPartitionMV(cache *macroblockMotionCache, list int, index int, subMBType uint32, mv [2]int16) {
	dst := int(h264Scan8[index])
	if subMBType&MBType16x16 != 0 {
		cache.MV[list][dst+1] = mv
		cache.MV[list][dst+8] = mv
		cache.MV[list][dst+9] = mv
	} else if subMBType&MBType16x8 != 0 {
		cache.MV[list][dst+1] = mv
	} else if subMBType&MBType8x16 != 0 {
		cache.MV[list][dst+8] = mv
	}
	cache.MV[list][dst] = mv
}

func addMVD(pred [2]int16, delta [2]int32) [2]int16 {
	return [2]int16{
		int16(int32(pred[0]) + delta[0]),
		int16(int32(pred[1]) + delta[1]),
	}
}

func refAndMVZero(ref int8, mv [2]int16) bool {
	return ref == 0 && mv[0] == 0 && mv[1] == 0
}

func fillMotionRectangle(cache *[h264MotionCacheSize][2]int16, start int, width int, height int, stride int, value [2]int16) {
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			cache[start+y*stride+x] = value
		}
	}
}

func fillRefRectangle(cache *[h264MotionCacheSize]int8, start int, width int, height int, stride int, value int8) {
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			cache[start+y*stride+x] = value
		}
	}
}
