// SPDX-License-Identifier: LGPL-2.1-or-later
//
// Source-shaped MBAFF macroblock neighbor/cache handling from FFmpeg n8.0.1
// libavcodec/h264_mvpred.h fill_decode_neighbors and fill_decode_caches.

package h264

func (m *macroblockTables) fillFrameMacroblockDecodeCachesEntropy(intraCache *[h264IntraPredModeCacheSize]int8, residual *cavlcResidualContext, motion *macroblockMotionCache, in frameMacroblockDecodeCacheInput, frameMBAFF bool) (frameMacroblockDecodeCacheResult, error) {
	var result frameMacroblockDecodeCacheResult
	err := m.fillFrameMacroblockDecodeCachesEntropyInto(&result, intraCache, residual, motion, in, frameMBAFF)
	return result, err
}

func (m *macroblockTables) fillFrameMacroblockDecodeCachesEntropyInto(result *frameMacroblockDecodeCacheResult, intraCache *[h264IntraPredModeCacheSize]int8, residual *cavlcResidualContext, motion *macroblockMotionCache, in frameMacroblockDecodeCacheInput, frameMBAFF bool) error {
	if result == nil {
		return ErrInvalidData
	}
	if !frameMBAFF {
		return m.fillFrameMacroblockDecodeCachesInto(result, intraCache, residual, motion, in)
	}
	*result = frameMacroblockDecodeCacheResult{}
	neighbors, err := m.fillDecodeNeighborsFrameMBAFF(in.MBXY, in.SliceNum, in.MBType)
	if err != nil {
		return err
	}
	result.Neighbors = neighbors

	if !isSkip(in.MBType) {
		if isIntra(in.MBType) {
			result.Intra, err = m.fillIntraPredModeCachesMBAFF(intraCache, neighbors.intraPredNeighbors(in.MBType, in.ConstrainedIntraPred))
			if err != nil {
				return err
			}
		}
		result.Residual, err = m.fillResidualDecodeCaches(residual, neighbors.residualNeighbors(in.MBType, in.CABAC))
		if err != nil {
			return err
		}
	}

	if isInter(in.MBType) || (isDirect(in.MBType) && in.DirectSpatialMVPred) {
		motionNeighbors := neighbors.motionNeighbors(in.MBType, in.ListCount, in.SliceTypeNoS, in.CABAC, in.DirectSpatialMVPred)
		motionNeighbors.FrameMBAFF = frameMBAFF
		if err := m.fillMotionDecodeCaches(motion, motionNeighbors); err != nil {
			return err
		}
		h264MapMBAFFMotionNeighbors(motion, motionNeighbors)
	}
	return nil
}

func (m *macroblockTables) fillDecodeNeighborsFrameEntropy(mbXY int, sliceNum uint16, mbType uint32, fieldPicture bool, frameMBAFF bool) (macroblockDecodeNeighbors, error) {
	var n macroblockDecodeNeighbors
	err := m.fillDecodeNeighborsFrameEntropyInto(&n, mbXY, sliceNum, mbType, fieldPicture, frameMBAFF)
	return n, err
}

func (m *macroblockTables) fillDecodeNeighborsFrameEntropyInto(n *macroblockDecodeNeighbors, mbXY int, sliceNum uint16, mbType uint32, fieldPicture bool, frameMBAFF bool) error {
	if n == nil {
		return ErrInvalidData
	}
	if frameMBAFF {
		neighbors, err := m.fillDecodeNeighborsFrameMBAFF(mbXY, sliceNum, mbType)
		*n = neighbors
		return err
	}
	return m.fillDecodeNeighborsFrameFieldsInto(n, mbXY, sliceNum, mbType, fieldPicture)
}

func (m *macroblockTables) fillDecodeNeighborsFrameMBAFF(mbXY int, sliceNum uint16, mbType uint32) (macroblockDecodeNeighbors, error) {
	var n macroblockDecodeNeighbors
	if err := m.checkCodedMBXY(mbXY); err != nil {
		return n, err
	}
	if sliceNum == ^uint16(0) {
		return n, ErrInvalidData
	}

	mbX := mbXY % m.MBStride
	mbY := mbXY / m.MBStride
	mbField := mbType&MBTypeInterlaced != 0
	topStride := m.MBStride
	if mbField {
		topStride <<= 1
	}

	n = macroblockDecodeNeighbors{
		MBXY:             mbXY,
		MBX:              mbX,
		MBY:              mbY,
		TopLeftXY:        mbXY - topStride - 1,
		TopXY:            mbXY - topStride,
		TopRightXY:       mbXY - topStride + 1,
		LeftXY:           [2]int{mbXY - 1, mbXY - 1},
		TopLeftPartition: -1,
		LeftBlock:        &h264LeftBlockFrame,
	}
	if mbX == 0 {
		n.LeftXY = [2]int{-1, -1}
		n.TopLeftXY = -1
	}
	if mbX+1 >= m.MBWidth {
		n.TopRightXY = -1
	}

	leftMBField := mbX > 0 && m.macroblockTypeIfCoded(mbXY-1)&MBTypeInterlaced != 0
	if (mbY & 1) != 0 {
		if leftMBField != mbField {
			n.LeftXY[0] = mbXY - m.MBStride - 1
			n.LeftXY[1] = n.LeftXY[0]
			if mbField {
				n.LeftXY[1] += m.MBStride
				n.LeftBlock = &h264LeftBlockOptions[3]
			} else {
				n.TopLeftXY += m.MBStride
				n.TopLeftPartition = 0
				n.LeftBlock = &h264LeftBlockOptions[1]
			}
		}
	} else {
		if mbField {
			if n.TopLeftXY >= 0 && m.macroblockTypeIfCoded(n.TopLeftXY)&MBTypeInterlaced == 0 {
				n.TopLeftXY += m.MBStride
			}
			if n.TopRightXY >= 0 && m.macroblockTypeIfCoded(n.TopRightXY)&MBTypeInterlaced == 0 {
				n.TopRightXY += m.MBStride
			}
			if n.TopXY >= 0 && m.macroblockTypeIfCoded(n.TopXY)&MBTypeInterlaced == 0 {
				n.TopXY += m.MBStride
			}
		}
		if leftMBField != mbField {
			if mbField {
				n.LeftXY[1] += m.MBStride
				n.LeftBlock = &h264LeftBlockOptions[3]
			} else {
				n.LeftBlock = &h264LeftBlockOptions[2]
			}
		}
	}

	n.TopLeftType = m.macroblockTypeIfCoded(n.TopLeftXY)
	n.TopType = m.macroblockTypeIfCoded(n.TopXY)
	n.TopRightType = m.macroblockTypeIfCoded(n.TopRightXY)
	n.LeftType[0] = m.macroblockTypeIfCoded(n.LeftXY[0])
	n.LeftType[1] = m.macroblockTypeIfCoded(n.LeftXY[1])

	if !m.sameSlice(n.TopLeftXY, sliceNum) {
		n.TopLeftType = 0
		if !m.sameSlice(n.TopXY, sliceNum) {
			n.TopType = 0
		}
		if !m.sameSlice(n.LeftXY[0], sliceNum) {
			n.LeftType[0] = 0
			n.LeftType[1] = 0
		}
	}
	if !m.sameSlice(n.TopRightXY, sliceNum) {
		n.TopRightType = 0
	}
	return n, nil
}

func (m *macroblockTables) fillIntraPredModeCachesMBAFF(cache *[h264IntraPredModeCacheSize]int8, n intraPredDecodeNeighbors) (intraPredDecodeCacheResult, error) {
	result, err := m.fillIntraPredModeCaches(cache, n)
	if err != nil {
		return result, err
	}
	typeAllowed := func(mbType uint32) bool {
		if n.ConstrainedIntraPred {
			return isIntra(mbType)
		}
		return mbType != 0
	}

	result.TopLeftSamplesAvailable = 0xffff
	result.TopSamplesAvailable = 0xffff
	result.TopRightSamplesAvailable = 0xeeea
	result.LeftSamplesAvailable = 0xffff
	if !typeAllowed(n.TopType) {
		result.TopLeftSamplesAvailable = 0xb3ff
		result.TopSamplesAvailable = 0x33ff
		result.TopRightSamplesAvailable = 0x26ea
	}
	mbField := n.MBType&MBTypeInterlaced != 0
	leftField := n.LeftType[h264LeftTop]&MBTypeInterlaced != 0
	if mbField != leftField {
		if mbField {
			if !typeAllowed(n.LeftType[h264LeftTop]) {
				result.TopLeftSamplesAvailable &= 0xdfff
				result.LeftSamplesAvailable &= 0x5fff
			}
			if !typeAllowed(n.LeftType[h264LeftBot]) {
				result.TopLeftSamplesAvailable &= 0xff5f
				result.LeftSamplesAvailable &= 0xff5f
			}
		} else {
			leftTypeI := m.macroblockTypeIfCoded(n.LeftXY[h264LeftTop] + m.MBStride)
			if !(typeAllowed(leftTypeI) && typeAllowed(n.LeftType[h264LeftTop])) {
				result.TopLeftSamplesAvailable &= 0xdf5f
				result.LeftSamplesAvailable &= 0x5f5f
			}
		}
	} else if !typeAllowed(n.LeftType[h264LeftTop]) {
		result.TopLeftSamplesAvailable &= 0xdf5f
		result.LeftSamplesAvailable &= 0x5f5f
	}
	if !typeAllowed(n.TopLeftType) {
		result.TopLeftSamplesAvailable &= 0x7fff
	}
	if !typeAllowed(n.TopRightType) {
		result.TopRightSamplesAvailable &= 0xfbff
	}
	return result, nil
}

func h264MapMBAFFMotionNeighbors(cache *macroblockMotionCache, n motionDecodeNeighbors) {
	if cache == nil {
		return
	}
	for list := 0; list < n.ListCount && list < 2; list++ {
		if !usesList(n.MBType, list) {
			continue
		}
		h264MapMBAFFMotionNeighbor(cache, list, int(h264Scan8[0])-1-8, n.TopLeftType, n.MBType)
		h264MapMBAFFMotionNeighbor(cache, list, int(h264Scan8[0])+0-8, n.TopType, n.MBType)
		h264MapMBAFFMotionNeighbor(cache, list, int(h264Scan8[0])+1-8, n.TopType, n.MBType)
		h264MapMBAFFMotionNeighbor(cache, list, int(h264Scan8[0])+2-8, n.TopType, n.MBType)
		h264MapMBAFFMotionNeighbor(cache, list, int(h264Scan8[0])+3-8, n.TopType, n.MBType)
		h264MapMBAFFMotionNeighbor(cache, list, int(h264Scan8[0])+4-8, n.TopRightType, n.MBType)
		h264MapMBAFFMotionNeighbor(cache, list, int(h264Scan8[0])-1+0*8, n.LeftType[h264LeftTop], n.MBType)
		h264MapMBAFFMotionNeighbor(cache, list, int(h264Scan8[0])-1+1*8, n.LeftType[h264LeftTop], n.MBType)
		h264MapMBAFFMotionNeighbor(cache, list, int(h264Scan8[0])-1+2*8, n.LeftType[h264LeftBot], n.MBType)
		h264MapMBAFFMotionNeighbor(cache, list, int(h264Scan8[0])-1+3*8, n.LeftType[h264LeftBot], n.MBType)
	}
}

func h264MapMBAFFMotionNeighbor(cache *macroblockMotionCache, list int, idx int, neighborType uint32, mbType uint32) {
	if idx < 0 || idx >= h264MotionCacheSize || list < 0 || list > 1 || cache.Ref[list][idx] < 0 {
		return
	}
	mbField := mbType&MBTypeInterlaced != 0
	neighborField := neighborType&MBTypeInterlaced != 0
	if mbField && !neighborField {
		cache.Ref[list][idx] *= 2
		cache.MV[list][idx][1] /= 2
		cache.MVD[list][idx][1] >>= 1
	} else if !mbField && neighborField {
		cache.Ref[list][idx] >>= 1
		cache.MV[list][idx][1] = int16(int(cache.MV[list][idx][1]) * 2)
		cache.MVD[list][idx][1] <<= 1
	}
}
