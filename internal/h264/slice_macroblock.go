// SPDX-License-Identifier: LGPL-2.1-or-later
//
// Source-shaped frame-macroblock slice positioning and neighbor/cache
// orchestration from FFmpeg n8.0.1 libavcodec/h264_slice.c decode_slice and
// libavcodec/h264_mvpred.h fill_decode_neighbors / fill_decode_caches.

package h264

type sliceMacroblockCursor struct {
	MBWidth      int
	MBHeight     int
	MBStride     int
	FieldOrMBAFF bool
	FrameMBAFF   bool
	FieldPicture bool
	MBX          int
	MBY          int
	PixelMBY     int
	MBXY         int
}

type frameMacroblockDecodeWork struct {
	IntraCache [h264IntraPredModeCacheSize]int8
	Residual   cavlcResidualContext
	Motion     macroblockMotionCache
}

// resetForDecode clears neighbor-derived caches while leaving coefficient
// arrays to the residual payload, which knows whether the macroblock can use
// them.
func (w *frameMacroblockDecodeWork) resetForDecode() {
	w.IntraCache = [h264IntraPredModeCacheSize]int8{}
	w.Residual.NonZeroCountCache = [h264NonZeroCountCacheSize]uint8{}
	w.Motion = macroblockMotionCache{}
}

// resetForSkip clears the caches used by skip/direct prediction without
// touching coefficient storage. Skipped macroblocks have CBP zero, so
// reconstruction cannot consume Residual.MB or Residual.MBLumaDC. Residual
// decoding clears coefficient storage before any later macroblock can use it.
func (w *frameMacroblockDecodeWork) resetForSkip() {
	w.resetForDecode()
}

func newSliceMacroblockCursor(m *macroblockTables, sh *SliceHeader) (sliceMacroblockCursor, error) {
	var cur sliceMacroblockCursor
	if m == nil || sh == nil || sh.SPS == nil || sh.PPS == nil {
		return cur, ErrInvalidData
	}
	if sh.PictureStructure != PictureFrame && sh.PictureStructure != PictureTopField && sh.PictureStructure != PictureBottomField {
		return cur, ErrInvalidData
	}
	first := int(sh.FirstMBAddr)
	mbNum := m.MBWidth * m.MBHeight
	frameMBAFF := sh.PictureStructure == PictureFrame && sh.SPS.MBAFF != 0
	fieldPicture := sh.PictureStructure != PictureFrame
	fieldOrMBAFF := frameMBAFF || fieldPicture
	if first < 0 || first >= mbNum || (fieldOrMBAFF && first<<1 >= mbNum) {
		return cur, ErrInvalidData
	}
	cur.MBWidth = m.MBWidth
	cur.MBHeight = m.MBHeight
	cur.MBStride = m.MBStride
	cur.FieldOrMBAFF = fieldOrMBAFF
	cur.FrameMBAFF = frameMBAFF
	cur.FieldPicture = fieldPicture
	cur.MBX = first % m.MBWidth
	fieldRow := first / m.MBWidth
	cur.PixelMBY = fieldRow
	cur.MBY = fieldRow
	if cur.FieldOrMBAFF {
		cur.MBY <<= 1
	}
	if sh.PictureStructure == PictureBottomField {
		cur.MBY++
	}
	if !cur.FieldPicture {
		cur.PixelMBY = cur.MBY
	}
	cur.MBXY = cur.MBX + cur.MBY*m.MBStride
	return cur, nil
}

func (c *sliceMacroblockCursor) advanceFrameMB() bool {
	if c == nil || c.MBWidth <= 0 || c.MBHeight <= 0 || c.MBStride <= 0 {
		return false
	}
	c.MBX++
	if c.MBX >= c.MBWidth {
		c.MBX = 0
		c.MBY++
		if c.FieldOrMBAFF {
			c.MBY++
		}
		if c.FieldPicture {
			c.PixelMBY++
		} else {
			c.PixelMBY = c.MBY
		}
	}
	if c.MBY >= c.MBHeight {
		c.MBXY = c.MBStride * c.MBHeight
		return false
	}
	c.MBXY = c.MBX + c.MBY*c.MBStride
	return true
}

func (c sliceMacroblockCursor) bottomMBAFFFrameMB() (sliceMacroblockCursor, error) {
	if !c.FrameMBAFF || (c.MBY&1) != 0 || c.MBY+1 >= c.MBHeight {
		return sliceMacroblockCursor{}, ErrInvalidData
	}
	c.MBY++
	c.PixelMBY = c.MBY
	c.MBXY += c.MBStride
	return c, nil
}

func (m *macroblockTables) predictFrameMBAFFFieldDecodingFlag(mbXY int, sliceNum uint16) int32 {
	if m == nil || mbXY < 0 || mbXY >= len(m.MacroblockTyp) {
		return 0
	}
	mbType := uint32(0)
	mbX := mbXY % m.MBStride
	leftXY := mbXY - 1
	topXY := mbXY - m.MBStride
	if mbX != 0 && m.sameSlice(leftXY, sliceNum) {
		mbType = m.MacroblockTyp[leftXY]
	} else if m.sameSlice(topXY, sliceNum) {
		mbType = m.MacroblockTyp[topXY]
	}
	if mbType&MBTypeInterlaced != 0 {
		return 1
	}
	return 0
}

type macroblockDecodeNeighbors struct {
	MBXY             int
	MBX              int
	MBY              int
	TopLeftXY        int
	TopXY            int
	TopRightXY       int
	LeftXY           [2]int
	TopLeftType      uint32
	TopType          uint32
	TopRightType     uint32
	LeftType         [2]uint32
	TopLeftPartition int
	LeftBlock        *[32]uint8
}

type frameMacroblockDecodeCacheInput struct {
	MBXY                 int
	SliceNum             uint16
	MBType               uint32
	ListCount            int
	SliceTypeNoS         int32
	CABAC                bool
	FieldPicture         bool
	ConstrainedIntraPred bool
	DirectSpatialMVPred  bool
}

type frameMacroblockDecodeCacheResult struct {
	Neighbors macroblockDecodeNeighbors
	Intra     intraPredDecodeCacheResult
	Residual  residualDecodeCacheResult
}

func (m *macroblockTables) fillDecodeNeighborsFrame(mbXY int, sliceNum uint16, mbType uint32) (macroblockDecodeNeighbors, error) {
	return m.fillDecodeNeighborsFrameFields(mbXY, sliceNum, mbType, false)
}

func (m *macroblockTables) fillDecodeNeighborsFrameFields(mbXY int, sliceNum uint16, mbType uint32, fieldPicture bool) (macroblockDecodeNeighbors, error) {
	var n macroblockDecodeNeighbors
	if err := m.checkCodedMBXY(mbXY); err != nil {
		return n, err
	}
	if sliceNum == ^uint16(0) {
		return n, ErrInvalidData
	}

	mbX := mbXY % m.MBStride
	mbY := mbXY / m.MBStride
	n = macroblockDecodeNeighbors{
		MBXY:             mbXY,
		MBX:              mbX,
		MBY:              mbY,
		TopLeftXY:        -1,
		TopXY:            -1,
		TopRightXY:       -1,
		LeftXY:           [2]int{-1, -1},
		TopLeftPartition: -1,
		LeftBlock:        &h264LeftBlockFrame,
	}

	topStride := m.MBStride
	topRows := 1
	if fieldPicture {
		topStride <<= 1
		topRows = 2
	}
	if mbY >= topRows {
		n.TopXY = mbXY - topStride
		if mbX > 0 {
			n.TopLeftXY = n.TopXY - 1
		}
		if mbX+1 < m.MBWidth {
			n.TopRightXY = n.TopXY + 1
		}
	}
	if mbX > 0 {
		n.LeftXY[0] = mbXY - 1
		n.LeftXY[1] = n.LeftXY[0]
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

func (m *macroblockTables) fillFrameMacroblockDecodeCaches(intraCache *[h264IntraPredModeCacheSize]int8, residual *cavlcResidualContext, motion *macroblockMotionCache, in frameMacroblockDecodeCacheInput) (frameMacroblockDecodeCacheResult, error) {
	var result frameMacroblockDecodeCacheResult
	neighbors, err := m.fillDecodeNeighborsFrameFields(in.MBXY, in.SliceNum, in.MBType, in.FieldPicture)
	if err != nil {
		return result, err
	}
	result.Neighbors = neighbors

	if !isSkip(in.MBType) {
		if isIntra(in.MBType) {
			result.Intra, err = m.fillIntraPredModeCaches(intraCache, neighbors.intraPredNeighbors(in.MBType, in.ConstrainedIntraPred))
			if err != nil {
				return result, err
			}
		}
		result.Residual, err = m.fillResidualDecodeCaches(residual, neighbors.residualNeighbors(in.MBType, in.CABAC))
		if err != nil {
			return result, err
		}
	}

	if isInter(in.MBType) || (isDirect(in.MBType) && in.DirectSpatialMVPred) {
		if err := m.fillMotionDecodeCaches(motion, neighbors.motionNeighbors(in.MBType, in.ListCount, in.SliceTypeNoS, in.CABAC, in.DirectSpatialMVPred)); err != nil {
			return result, err
		}
	}
	return result, nil
}

func (n macroblockDecodeNeighbors) intraPredNeighbors(mbType uint32, constrained bool) intraPredDecodeNeighbors {
	return intraPredDecodeNeighbors{
		MBType:               mbType,
		TopType:              n.TopType,
		TopLeftType:          n.TopLeftType,
		TopRightType:         n.TopRightType,
		LeftType:             n.LeftType,
		TopXY:                n.TopXY,
		LeftXY:               n.LeftXY,
		ConstrainedIntraPred: constrained,
		LeftBlock:            n.LeftBlock,
	}
}

func (n macroblockDecodeNeighbors) residualNeighbors(mbType uint32, cabac bool) residualDecodeNeighbors {
	return residualDecodeNeighbors{
		MBType:    mbType,
		TopType:   n.TopType,
		LeftType:  n.LeftType,
		TopXY:     n.TopXY,
		LeftXY:    n.LeftXY,
		CABAC:     cabac,
		LeftBlock: n.LeftBlock,
	}
}

func (n macroblockDecodeNeighbors) motionNeighbors(mbType uint32, listCount int, sliceTypeNoS int32, cabac bool, directSpatial bool) motionDecodeNeighbors {
	return motionDecodeNeighbors{
		MBType:              mbType,
		TopType:             n.TopType,
		TopLeftType:         n.TopLeftType,
		TopRightType:        n.TopRightType,
		LeftType:            n.LeftType,
		TopXY:               n.TopXY,
		TopLeftXY:           n.TopLeftXY,
		TopRightXY:          n.TopRightXY,
		LeftXY:              n.LeftXY,
		TopLeftPartition:    n.TopLeftPartition,
		ListCount:           listCount,
		SliceTypeNoS:        sliceTypeNoS,
		CABAC:               cabac,
		DirectSpatialMVPred: directSpatial,
		LeftBlock:           n.LeftBlock,
	}
}

func (m *macroblockTables) macroblockTypeIfCoded(mbXY int) uint32 {
	if m.isCodedMBXY(mbXY) {
		return m.MacroblockTyp[mbXY]
	}
	return 0
}

func (m *macroblockTables) sameSlice(mbXY int, sliceNum uint16) bool {
	return m.isCodedMBXY(mbXY) && m.SliceTable[mbXY] == sliceNum
}

func (m *macroblockTables) isCodedMBXY(mbXY int) bool {
	return m != nil && mbXY >= 0 && mbXY < m.MBStride*m.MBHeight && mbXY%m.MBStride < m.MBWidth
}
