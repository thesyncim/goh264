// SPDX-License-Identifier: LGPL-2.1-or-later
//
// Source-shaped CABAC inter motion-cache fill from FFmpeg n8.0.1
// libavcodec/h264_cabac.c ff_h264_decode_mb_cabac. This layer starts after the
// CABAC ref/MVD syntax has been decoded and preserves FFmpeg's MV/MVD cache
// rectangle writes before h264_mvpred.h write_back_motion persists them.

package h264

func fillCABACInterMotionCache(cache *macroblockMotionCache, mb *cavlcInterMacroblockSyntax, listCount int) error {
	if cache == nil || mb == nil || listCount < 0 || listCount > 2 || isIntra(mb.MBType) {
		return ErrInvalidData
	}
	if isDirect(mb.MBType) {
		return ErrUnsupported
	}
	if mb.PartitionCount == 4 {
		return fillCABACSubInterMotionCache(cache, mb, listCount)
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
			ref := cache.Ref[list][h264Scan8[0]]
			pred, err := predMotion(cache, 0, 4, list, ref)
			if err != nil {
				return err
			}
			fillMVDRectangle(&cache.MVD[list], int(h264Scan8[0]), 4, 4, 8, cabacMVDCachePair(mb.MVD[list][0]))
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
				start := int(h264Scan8[0]) + 16*i
				if !isDir(mb.MBType, i, list) {
					fillMVDRectangle(&cache.MVD[list], start, 4, 2, 8, [2]uint8{})
					fillMotionRectangle(&cache.MV[list], start, 4, 2, 8, [2]int16{})
					continue
				}
				ref := cache.Ref[list][start]
				pred, err := pred16x8Motion(cache, 8*i, list, ref)
				if err != nil {
					return err
				}
				mvd := mb.MVD[list][8*i]
				fillMVDRectangle(&cache.MVD[list], start, 4, 2, 8, cabacMVDCachePair(mvd))
				fillMotionRectangle(&cache.MV[list], start, 4, 2, 8, addMVD(pred, mvd))
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
				start := int(h264Scan8[0]) + 2*i
				if !isDir(mb.MBType, i, list) {
					fillMVDRectangle(&cache.MVD[list], start, 2, 4, 8, [2]uint8{})
					fillMotionRectangle(&cache.MV[list], start, 2, 4, 8, [2]int16{})
					continue
				}
				ref := cache.Ref[list][start]
				pred, err := pred8x16Motion(cache, 4*i, list, ref)
				if err != nil {
					return err
				}
				mvd := mb.MVD[list][4*i]
				fillMVDRectangle(&cache.MVD[list], start, 2, 4, 8, cabacMVDCachePair(mvd))
				fillMotionRectangle(&cache.MV[list], start, 2, 4, 8, addMVD(pred, mvd))
			}
		}
		return nil
	}
	return ErrUnsupported
}

func fillCABACSubInterMotionCache(cache *macroblockMotionCache, mb *cavlcInterMacroblockSyntax, listCount int) error {
	for i := 0; i < 4; i++ {
		if isDirect(mb.SubMBType[i]) {
			return ErrUnsupported
		}
	}

	for list := 0; list < listCount; list++ {
		for i := 0; i < 4; i++ {
			ref := h264ListNotUsed
			if isDir(mb.SubMBType[i], 0, list) {
				ref = int8(mb.Ref[list][i])
			}
			fillRefRectangle(&cache.Ref[list], int(h264Scan8[4*i]), 2, 2, 8, ref)
		}
	}

	for list := 0; list < listCount; list++ {
		for i := 0; i < 4; i++ {
			start := int(h264Scan8[4*i])
			if !isDir(mb.SubMBType[i], 0, list) {
				fillMVDRectangle(&cache.MVD[list], start, 2, 2, 8, [2]uint8{})
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
				pred, err := predMotion(cache, index, blockWidth, list, ref)
				if err != nil {
					return err
				}
				mvd := mb.MVD[list][index]
				writeSubPartitionMVD(cache, list, index, mb.SubMBType[i], cabacMVDCachePair(mvd))
				writeSubPartitionMV(cache, list, index, mb.SubMBType[i], addMVD(pred, mvd))
			}
		}
	}
	return nil
}

func cabacMVDCachePair(delta [2]int32) [2]uint8 {
	return [2]uint8{cabacMVDCacheMagnitude(delta[0]), cabacMVDCacheMagnitude(delta[1])}
}

func cabacMVDCacheMagnitude(delta int32) uint8 {
	magnitude := int64(delta)
	if magnitude < 0 {
		magnitude = -magnitude
	}
	if magnitude > 70 {
		return 70
	}
	return uint8(magnitude)
}

func writeSubPartitionMVD(cache *macroblockMotionCache, list int, index int, subMBType uint32, mvd [2]uint8) {
	dst := int(h264Scan8[index])
	if subMBType&MBType16x16 != 0 {
		cache.MVD[list][dst+1] = mvd
		cache.MVD[list][dst+8] = mvd
		cache.MVD[list][dst+9] = mvd
	} else if subMBType&MBType16x8 != 0 {
		cache.MVD[list][dst+1] = mvd
	} else if subMBType&MBType8x16 != 0 {
		cache.MVD[list][dst+8] = mvd
	}
	cache.MVD[list][dst] = mvd
}

func fillMVDRectangle(cache *[h264MotionCacheSize][2]uint8, start int, width int, height int, stride int, value [2]uint8) {
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			cache[start+y*stride+x] = value
		}
	}
}
