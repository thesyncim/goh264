// SPDX-License-Identifier: LGPL-2.1-or-later
//
// Source-shaped port of the frame-MB macroblock cache and write-back pieces of
// FFmpeg n8.0.1 libavcodec/h264_mvpred.h fill_decode_caches,
// write_back_non_zero_count, write_back_intra_pred_mode, write_back_motion_list,
// and write_back_motion.

package h264

const (
	h264IntraPredModeCacheSize = 5 * 8
	h264MBNonZeroCountSize     = 48
	h264MotionCacheSize        = 5 * 8

	h264LeftTop = 0
	h264LeftBot = 1

	h264ListNotUsed      int8 = -1
	h264PartNotAvailable int8 = -2
)

var h264LeftBlockOptions = [4][32]uint8{
	{
		0, 1, 2, 3, 7, 10, 8, 11,
		3 + 0*4, 3 + 1*4, 3 + 2*4, 3 + 3*4,
		1 + 4*4, 1 + 8*4, 1 + 5*4, 1 + 9*4,
	},
	{
		2, 2, 3, 3, 8, 11, 8, 11,
		3 + 2*4, 3 + 2*4, 3 + 3*4, 3 + 3*4,
		1 + 5*4, 1 + 9*4, 1 + 5*4, 1 + 9*4,
	},
	{
		0, 0, 1, 1, 7, 10, 7, 10,
		3 + 0*4, 3 + 0*4, 3 + 1*4, 3 + 1*4,
		1 + 4*4, 1 + 8*4, 1 + 4*4, 1 + 8*4,
	},
	{
		0, 2, 0, 2, 7, 10, 7, 10,
		3 + 0*4, 3 + 2*4, 3 + 0*4, 3 + 2*4,
		1 + 4*4, 1 + 8*4, 1 + 4*4, 1 + 8*4,
	},
}

var h264LeftBlockFrame = h264LeftBlockOptions[0]

type macroblockTables struct {
	MBWidth         int
	MBHeight        int
	MBStride        int
	BStride         int
	ChromaFormatIDC int
	ChromaYShift    int
	MB2BXY          []uint32
	MB2BRXY         []uint32
	NonZeroCount    [][h264MBNonZeroCountSize]uint8
	CBPTable        []int
	QScaleTable     []uint8
	ChromaPred      []int8
	Intra4x4Pred    []int8
	MacroblockTyp   []uint32
	SliceTable      []uint16
	RefIndex        [2][]int8
	MotionVal       [2][][2]int16
	MVDTable        [2][][2]uint8
	DirectTable     []uint8
	ListCounts      []uint8
}

type macroblockMotionCache struct {
	MV     [2][h264MotionCacheSize][2]int16
	Ref    [2][h264MotionCacheSize]int8
	MVD    [2][h264MotionCacheSize][2]uint8
	Direct [h264MotionCacheSize]uint8
}

func h264MBAFFFieldRefCount(refCount [2]uint32) [2]uint32 {
	for list := 0; list < 2; list++ {
		refCount[list] <<= 1
	}
	return refCount
}

type intraPredDecodeNeighbors struct {
	MBType               uint32
	TopType              uint32
	TopLeftType          uint32
	TopRightType         uint32
	LeftType             [2]uint32
	TopXY                int
	LeftXY               [2]int
	ConstrainedIntraPred bool
	LeftBlock            *[32]uint8
}

type intraPredDecodeCacheResult struct {
	TopLeftSamplesAvailable  uint16
	TopSamplesAvailable      uint16
	TopRightSamplesAvailable uint16
	LeftSamplesAvailable     uint16
	NeighborTransformSize    int
}

type residualDecodeNeighbors struct {
	MBType    uint32
	TopType   uint32
	LeftType  [2]uint32
	TopXY     int
	LeftXY    [2]int
	CABAC     bool
	LeftBlock *[32]uint8
}

type residualDecodeCacheResult struct {
	TopCBP  int
	LeftCBP int
}

type motionDecodeNeighbors struct {
	MBType              uint32
	TopType             uint32
	TopLeftType         uint32
	TopRightType        uint32
	LeftType            [2]uint32
	TopXY               int
	TopLeftXY           int
	TopRightXY          int
	LeftXY              [2]int
	TopLeftPartition    int
	ListCount           int
	SliceTypeNoS        int32
	CABAC               bool
	DirectSpatialMVPred bool
	FrameMBAFF          bool
	LeftBlock           *[32]uint8
}

func newMacroblockTables(mbWidth int, mbHeight int, chromaFormatIDC int) (*macroblockTables, error) {
	if mbWidth <= 0 || mbHeight <= 0 || chromaFormatIDC < 0 || chromaFormatIDC > 3 {
		return nil, ErrInvalidData
	}
	mbStride, err := checkedAddInt(mbWidth, 1)
	if err != nil {
		return nil, err
	}
	mbHeightPlus1, err := checkedAddInt(mbHeight, 1)
	if err != nil {
		return nil, err
	}
	bigMBNum, err := checkedMulInt(mbStride, mbHeightPlus1)
	if err != nil {
		return nil, err
	}
	mbArraySize, err := checkedMulInt(mbStride, mbHeight)
	if err != nil {
		return nil, err
	}
	rowMBNum, err := checkedMulInt(2, mbStride)
	if err != nil {
		return nil, err
	}
	bStride, err := checkedMulInt(mbWidth, 4)
	if err != nil {
		return nil, err
	}
	intraPredSize, err := checkedMulInt(rowMBNum, 8)
	if err != nil {
		return nil, err
	}
	directSize, err := checkedMulInt(bigMBNum, 4)
	if err != nil {
		return nil, err
	}
	refIndexSize, err := checkedMulInt(4, mbArraySize)
	if err != nil {
		return nil, err
	}
	motionValRows, err := checkedMulInt(bStride, mbHeight)
	if err != nil {
		return nil, err
	}
	motionValSize, err := checkedMulInt(motionValRows, 4)
	if err != nil {
		return nil, err
	}
	chromaYShift := 0
	if chromaFormatIDC <= 1 {
		chromaYShift = 1
	}
	m := &macroblockTables{
		MBWidth:         mbWidth,
		MBHeight:        mbHeight,
		MBStride:        mbStride,
		BStride:         bStride,
		ChromaFormatIDC: chromaFormatIDC,
		ChromaYShift:    chromaYShift,
		MB2BXY:          make([]uint32, bigMBNum),
		MB2BRXY:         make([]uint32, bigMBNum),
		NonZeroCount:    make([][h264MBNonZeroCountSize]uint8, bigMBNum),
		CBPTable:        make([]int, bigMBNum),
		QScaleTable:     make([]uint8, bigMBNum),
		ChromaPred:      make([]int8, bigMBNum),
		Intra4x4Pred:    make([]int8, intraPredSize),
		MacroblockTyp:   make([]uint32, bigMBNum),
		SliceTable:      make([]uint16, bigMBNum),
		DirectTable:     make([]uint8, directSize),
		ListCounts:      make([]uint8, bigMBNum),
	}
	for i := range m.SliceTable {
		m.SliceTable[i] = ^uint16(0)
	}
	for list := 0; list < 2; list++ {
		m.RefIndex[list] = make([]int8, refIndexSize)
		m.MotionVal[list] = make([][2]int16, motionValSize)
		m.MVDTable[list] = make([][2]uint8, intraPredSize)
	}
	for y := 0; y < mbHeight; y++ {
		for x := 0; x < mbWidth; x++ {
			mbXY := x + y*m.MBStride
			m.MB2BXY[mbXY] = uint32(4*x + 4*y*m.BStride)
			m.MB2BRXY[mbXY] = uint32(8 * (mbXY % (2 * m.MBStride)))
		}
	}
	return m, nil
}

func (m *macroblockTables) resetForDecode() {
	if m == nil {
		return
	}
	clear(m.NonZeroCount)
	clear(m.CBPTable)
	clear(m.QScaleTable)
	clear(m.ChromaPred)
	clear(m.Intra4x4Pred)
	clear(m.MacroblockTyp)
	clear(m.DirectTable)
	clear(m.ListCounts)
	for i := range m.SliceTable {
		m.SliceTable[i] = ^uint16(0)
	}
	for list := 0; list < 2; list++ {
		clear(m.RefIndex[list])
		clear(m.MotionVal[list])
		clear(m.MVDTable[list])
	}
}

func (m *macroblockTables) checkMBXY(mbXY int) error {
	if m == nil || mbXY < 0 || mbXY >= len(m.NonZeroCount) {
		return ErrInvalidData
	}
	return nil
}

func (m *macroblockTables) checkCodedMBXY(mbXY int) error {
	if err := m.checkMBXY(mbXY); err != nil {
		return err
	}
	if mbXY >= m.MBStride*m.MBHeight || mbXY%m.MBStride >= m.MBWidth {
		return ErrInvalidData
	}
	return nil
}

func checkRange(length int, start int, count int) error {
	if start < 0 || count < 0 || start > length-count {
		return ErrInvalidData
	}
	return nil
}

func (m *macroblockTables) writeBackNonZeroCount(mbXY int, cache *[h264NonZeroCountCacheSize]uint8) error {
	if cache == nil {
		return ErrInvalidData
	}
	if err := m.checkCodedMBXY(mbXY); err != nil {
		return err
	}
	nnz := &m.NonZeroCount[mbXY]
	copy(nnz[0:4], cache[4+8*1:4+8*1+4])
	copy(nnz[4:8], cache[4+8*2:4+8*2+4])
	copy(nnz[8:12], cache[4+8*3:4+8*3+4])
	copy(nnz[12:16], cache[4+8*4:4+8*4+4])
	copy(nnz[16:20], cache[4+8*6:4+8*6+4])
	copy(nnz[20:24], cache[4+8*7:4+8*7+4])
	copy(nnz[32:36], cache[4+8*11:4+8*11+4])
	copy(nnz[36:40], cache[4+8*12:4+8*12+4])

	if m.ChromaYShift == 0 {
		copy(nnz[24:28], cache[4+8*8:4+8*8+4])
		copy(nnz[28:32], cache[4+8*9:4+8*9+4])
		copy(nnz[40:44], cache[4+8*13:4+8*13+4])
		copy(nnz[44:48], cache[4+8*14:4+8*14+4])
	}
	return nil
}

func (m *macroblockTables) writeBackIntraPredMode(mbXY int, cache *[h264IntraPredModeCacheSize]int8) error {
	if cache == nil {
		return ErrInvalidData
	}
	if err := m.checkCodedMBXY(mbXY); err != nil {
		return err
	}
	dst := int(m.MB2BRXY[mbXY])
	if dst+7 > len(m.Intra4x4Pred) {
		return ErrInvalidData
	}
	copy(m.Intra4x4Pred[dst:dst+4], cache[4+8*4:4+8*4+4])
	m.Intra4x4Pred[dst+4] = cache[7+8*3]
	m.Intra4x4Pred[dst+5] = cache[7+8*2]
	m.Intra4x4Pred[dst+6] = cache[7+8*1]
	return nil
}

func (m *macroblockTables) fillIntraPredModeCaches(cache *[h264IntraPredModeCacheSize]int8, n intraPredDecodeNeighbors) (intraPredDecodeCacheResult, error) {
	var result intraPredDecodeCacheResult
	if cache == nil || m == nil || !isIntra(n.MBType) {
		return result, ErrInvalidData
	}
	leftBlock := &h264LeftBlockFrame
	if n.LeftBlock != nil {
		leftBlock = n.LeftBlock
	}
	typeAllowed := func(mbType uint32) bool {
		if n.ConstrainedIntraPred {
			return isIntra(mbType)
		}
		return mbType != 0
	}
	defaultMode := func(mbType uint32) int8 {
		if typeAllowed(mbType) {
			return 2
		}
		return -1
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
	if !typeAllowed(n.LeftType[h264LeftTop]) {
		result.TopLeftSamplesAvailable &= 0xdf5f
		result.LeftSamplesAvailable &= 0x5f5f
	}
	if !typeAllowed(n.TopLeftType) {
		result.TopLeftSamplesAvailable &= 0x7fff
	}
	if !typeAllowed(n.TopRightType) {
		result.TopRightSamplesAvailable &= 0xfbff
	}
	result.NeighborTransformSize = boolToInt(is8x8DCT(n.TopType)) + boolToInt(is8x8DCT(n.LeftType[h264LeftTop]))

	if !isIntra4x4(n.MBType) {
		return result, nil
	}

	if isIntra4x4(n.TopType) {
		if err := m.checkMBXY(n.TopXY); err != nil {
			return result, err
		}
		src := int(m.MB2BRXY[n.TopXY])
		if err := checkRange(len(m.Intra4x4Pred), src, 4); err != nil {
			return result, err
		}
		copy(cache[4+8*0:4+8*0+4], m.Intra4x4Pred[src:src+4])
	} else {
		mode := defaultMode(n.TopType)
		cache[4+8*0] = mode
		cache[5+8*0] = mode
		cache[6+8*0] = mode
		cache[7+8*0] = mode
	}

	for i := 0; i < 2; i++ {
		if isIntra4x4(n.LeftType[i]) {
			if err := m.checkMBXY(n.LeftXY[i]); err != nil {
				return result, err
			}
			src := int(m.MB2BRXY[n.LeftXY[i]])
			if err := checkRange(len(m.Intra4x4Pred), src, 7); err != nil {
				return result, err
			}
			cache[3+8*1+2*8*i] = m.Intra4x4Pred[src+6-int(leftBlock[0+2*i])]
			cache[3+8*2+2*8*i] = m.Intra4x4Pred[src+6-int(leftBlock[1+2*i])]
		} else {
			mode := defaultMode(n.LeftType[i])
			cache[3+8*1+2*8*i] = mode
			cache[3+8*2+2*8*i] = mode
		}
	}
	return result, nil
}

func (m *macroblockTables) fillResidualDecodeCaches(c *cavlcResidualContext, n residualDecodeNeighbors) (residualDecodeCacheResult, error) {
	var result residualDecodeCacheResult
	if c == nil || m == nil {
		return result, ErrInvalidData
	}
	leftBlock := &h264LeftBlockFrame
	if n.LeftBlock != nil {
		leftBlock = n.LeftBlock
	}
	empty := uint8(64)
	if n.CABAC && !isIntra(n.MBType) {
		empty = 0
	}

	if n.TopType != 0 {
		if err := m.checkMBXY(n.TopXY); err != nil {
			return result, err
		}
		nnz := m.NonZeroCount[n.TopXY]
		copy(c.NonZeroCountCache[4+8*0:4+8*0+4], nnz[4*3:4*3+4])
		if m.ChromaYShift == 0 {
			copy(c.NonZeroCountCache[4+8*5:4+8*5+4], nnz[4*7:4*7+4])
			copy(c.NonZeroCountCache[4+8*10:4+8*10+4], nnz[4*11:4*11+4])
		} else {
			copy(c.NonZeroCountCache[4+8*5:4+8*5+4], nnz[4*5:4*5+4])
			copy(c.NonZeroCountCache[4+8*10:4+8*10+4], nnz[4*9:4*9+4])
		}
	} else {
		fillCAVLCNonZero(&c.NonZeroCountCache, 4+8*0, 4, 1, 8, empty)
		fillCAVLCNonZero(&c.NonZeroCountCache, 4+8*5, 4, 1, 8, empty)
		fillCAVLCNonZero(&c.NonZeroCountCache, 4+8*10, 4, 1, 8, empty)
	}

	for i := 0; i < 2; i++ {
		if n.LeftType[i] != 0 {
			if err := m.checkMBXY(n.LeftXY[i]); err != nil {
				return result, err
			}
			nnz := m.NonZeroCount[n.LeftXY[i]]
			c.NonZeroCountCache[3+8*1+2*8*i] = nnz[leftBlock[8+0+2*i]]
			c.NonZeroCountCache[3+8*2+2*8*i] = nnz[leftBlock[8+1+2*i]]
			if m.ChromaFormatIDC == 3 {
				c.NonZeroCountCache[3+8*6+2*8*i] = nnz[leftBlock[8+0+2*i]+4*4]
				c.NonZeroCountCache[3+8*7+2*8*i] = nnz[leftBlock[8+1+2*i]+4*4]
				c.NonZeroCountCache[3+8*11+2*8*i] = nnz[leftBlock[8+0+2*i]+8*4]
				c.NonZeroCountCache[3+8*12+2*8*i] = nnz[leftBlock[8+1+2*i]+8*4]
			} else if m.ChromaFormatIDC == 2 {
				c.NonZeroCountCache[3+8*6+2*8*i] = nnz[int(leftBlock[8+0+2*i])-2+4*4]
				c.NonZeroCountCache[3+8*7+2*8*i] = nnz[int(leftBlock[8+1+2*i])-2+4*4]
				c.NonZeroCountCache[3+8*11+2*8*i] = nnz[int(leftBlock[8+0+2*i])-2+8*4]
				c.NonZeroCountCache[3+8*12+2*8*i] = nnz[int(leftBlock[8+1+2*i])-2+8*4]
			} else {
				c.NonZeroCountCache[3+8*6+8*i] = nnz[leftBlock[8+4+2*i]]
				c.NonZeroCountCache[3+8*11+8*i] = nnz[leftBlock[8+5+2*i]]
			}
		} else {
			c.NonZeroCountCache[3+8*1+2*8*i] = empty
			c.NonZeroCountCache[3+8*2+2*8*i] = empty
			c.NonZeroCountCache[3+8*6+2*8*i] = empty
			c.NonZeroCountCache[3+8*7+2*8*i] = empty
			c.NonZeroCountCache[3+8*11+2*8*i] = empty
			c.NonZeroCountCache[3+8*12+2*8*i] = empty
		}
	}

	if n.CABAC {
		if n.TopType != 0 {
			result.TopCBP = m.CBPTable[n.TopXY]
		} else if isIntra(n.MBType) {
			result.TopCBP = 0x7cf
		} else {
			result.TopCBP = 0x00f
		}
		if n.LeftType[h264LeftTop] != 0 {
			if err := m.checkMBXY(n.LeftXY[h264LeftTop]); err != nil {
				return result, err
			}
			if err := m.checkMBXY(n.LeftXY[h264LeftBot]); err != nil {
				return result, err
			}
			result.LeftCBP = (m.CBPTable[n.LeftXY[h264LeftTop]] & 0x7f0) |
				((m.CBPTable[n.LeftXY[h264LeftTop]] >> (leftBlock[0] &^ 1)) & 2) |
				(((m.CBPTable[n.LeftXY[h264LeftBot]] >> (leftBlock[2] &^ 1)) & 2) << 2)
		} else if isIntra(n.MBType) {
			result.LeftCBP = 0x7cf
		} else {
			result.LeftCBP = 0x00f
		}
	}
	return result, nil
}

func (m *macroblockTables) fillMotionDecodeCaches(cache *macroblockMotionCache, n motionDecodeNeighbors) error {
	if cache == nil || m == nil || n.ListCount < 0 || n.ListCount > 2 {
		return ErrInvalidData
	}
	initMotionDecodeCacheSentinels(cache)
	if !(isInter(n.MBType) || (isDirect(n.MBType) && n.DirectSpatialMVPred)) {
		return nil
	}
	leftBlock := &h264LeftBlockFrame
	if n.LeftBlock != nil {
		leftBlock = n.LeftBlock
	}
	base := int(h264Scan8[0])
	for list := 0; list < n.ListCount; list++ {
		if !usesList(n.MBType, list) {
			continue
		}
		if usesList(n.TopType, list) {
			if err := m.copyTopMotion(cache, n.TopXY, list, base); err != nil {
				return err
			}
		} else {
			clearMotionRow(&cache.MV[list], base-8, 4)
			ref := h264PartNotAvailable
			if n.TopType != 0 {
				ref = h264ListNotUsed
			}
			fillRefRow(&cache.Ref[list], base-8, 4, ref)
		}

		if n.MBType&(MBType16x8|MBType8x8) != 0 {
			for i := 0; i < 2; i++ {
				cacheIdx := base - 1 + i*2*8
				if usesList(n.LeftType[i], list) {
					if err := m.copyLeftMotionPair(cache, n.LeftXY[i], int(leftBlock[0+2*i]), int(leftBlock[1+2*i]), cacheIdx, list); err != nil {
						return err
					}
				} else {
					cache.MV[list][cacheIdx] = [2]int16{}
					cache.MV[list][cacheIdx+8] = [2]int16{}
					ref := h264PartNotAvailable
					if n.LeftType[i] != 0 {
						ref = h264ListNotUsed
					}
					cache.Ref[list][cacheIdx] = ref
					cache.Ref[list][cacheIdx+8] = ref
				}
			}
		} else if usesList(n.LeftType[h264LeftTop], list) {
			if err := m.copyLeftMotionSingle(cache, n.LeftXY[h264LeftTop], int(leftBlock[0]), base-1, list); err != nil {
				return err
			}
		} else {
			cache.MV[list][base-1] = [2]int16{}
			if n.LeftType[h264LeftTop] != 0 {
				cache.Ref[list][base-1] = h264ListNotUsed
			} else {
				cache.Ref[list][base-1] = h264PartNotAvailable
			}
		}

		if usesList(n.TopRightType, list) {
			if err := m.copyTopRightMotion(cache, n.TopRightXY, base+4-8, list); err != nil {
				return err
			}
		} else {
			cache.MV[list][base+4-8] = [2]int16{}
			if n.TopRightType != 0 {
				cache.Ref[list][base+4-8] = h264ListNotUsed
			} else {
				cache.Ref[list][base+4-8] = h264PartNotAvailable
			}
		}
		if cache.Ref[list][base+2-8] < 0 || cache.Ref[list][base+4-8] < 0 {
			if usesList(n.TopLeftType, list) {
				if err := m.copyTopLeftMotion(cache, n.TopLeftXY, n.TopLeftPartition, base-1-8, list); err != nil {
					return err
				}
			} else {
				cache.MV[list][base-1-8] = [2]int16{}
				if n.TopLeftType != 0 {
					cache.Ref[list][base-1-8] = h264ListNotUsed
				} else {
					cache.Ref[list][base-1-8] = h264PartNotAvailable
				}
			}
		}

		if isSkip(n.MBType) || isDirect(n.MBType) {
			continue
		}
		cache.Ref[list][base+2+8*0] = h264PartNotAvailable
		cache.Ref[list][base+2+8*2] = h264PartNotAvailable
		cache.MV[list][base+2+8*0] = [2]int16{}
		cache.MV[list][base+2+8*2] = [2]int16{}

		if n.CABAC {
			if err := m.fillMVDNeighbors(cache, n, leftBlock, list, base); err != nil {
				return err
			}
			if n.SliceTypeNoS == PictureTypeB {
				fillDirectRectangle(&cache.Direct, base, 4, 4, 8, uint8(MBType16x16>>1))
				if err := m.fillDirectNeighbors(cache, n, leftBlock, base); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func initMotionDecodeCacheSentinels(cache *macroblockMotionCache) {
	if cache == nil {
		return
	}
	for list := 0; list < 2; list++ {
		cache.Ref[list][h264Scan8[5]+1] = h264PartNotAvailable
		cache.Ref[list][h264Scan8[7]+1] = h264PartNotAvailable
		cache.Ref[list][h264Scan8[13]+1] = h264PartNotAvailable
	}
}

func (m *macroblockTables) copyTopMotion(cache *macroblockMotionCache, topXY int, list int, base int) error {
	if err := m.checkMBXY(topXY); err != nil {
		return err
	}
	src := int(m.MB2BXY[topXY]) + 3*m.BStride
	if err := checkRange(len(m.MotionVal[list]), src, 4); err != nil {
		return err
	}
	copy(cache.MV[list][base-8:base-8+4], m.MotionVal[list][src:src+4])
	refBase := 4 * topXY
	if err := checkRange(len(m.RefIndex[list]), refBase+2, 2); err != nil {
		return err
	}
	cache.Ref[list][base+0-8] = m.RefIndex[list][refBase+2]
	cache.Ref[list][base+1-8] = m.RefIndex[list][refBase+2]
	cache.Ref[list][base+2-8] = m.RefIndex[list][refBase+3]
	cache.Ref[list][base+3-8] = m.RefIndex[list][refBase+3]
	return nil
}

func (m *macroblockTables) copyLeftMotionPair(cache *macroblockMotionCache, leftXY int, firstBlock int, secondBlock int, cacheIdx int, list int) error {
	if err := m.checkMBXY(leftXY); err != nil {
		return err
	}
	bXY := int(m.MB2BXY[leftXY]) + 3
	firstMV := bXY + m.BStride*firstBlock
	secondMV := bXY + m.BStride*secondBlock
	if err := checkRange(len(m.MotionVal[list]), firstMV, 1); err != nil {
		return err
	}
	if err := checkRange(len(m.MotionVal[list]), secondMV, 1); err != nil {
		return err
	}
	cache.MV[list][cacheIdx] = m.MotionVal[list][firstMV]
	cache.MV[list][cacheIdx+8] = m.MotionVal[list][secondMV]
	b8XY := 4*leftXY + 1
	firstRef := b8XY + (firstBlock &^ 1)
	secondRef := b8XY + (secondBlock &^ 1)
	if err := checkRange(len(m.RefIndex[list]), firstRef, 1); err != nil {
		return err
	}
	if err := checkRange(len(m.RefIndex[list]), secondRef, 1); err != nil {
		return err
	}
	cache.Ref[list][cacheIdx] = m.RefIndex[list][firstRef]
	cache.Ref[list][cacheIdx+8] = m.RefIndex[list][secondRef]
	return nil
}

func (m *macroblockTables) copyLeftMotionSingle(cache *macroblockMotionCache, leftXY int, leftBlock int, cacheIdx int, list int) error {
	if err := m.checkMBXY(leftXY); err != nil {
		return err
	}
	bXY := int(m.MB2BXY[leftXY]) + 3
	mvIdx := bXY + m.BStride*leftBlock
	if err := checkRange(len(m.MotionVal[list]), mvIdx, 1); err != nil {
		return err
	}
	cache.MV[list][cacheIdx] = m.MotionVal[list][mvIdx]
	refIdx := 4*leftXY + 1 + (leftBlock &^ 1)
	if err := checkRange(len(m.RefIndex[list]), refIdx, 1); err != nil {
		return err
	}
	cache.Ref[list][cacheIdx] = m.RefIndex[list][refIdx]
	return nil
}

func (m *macroblockTables) copyTopRightMotion(cache *macroblockMotionCache, topRightXY int, cacheIdx int, list int) error {
	if err := m.checkMBXY(topRightXY); err != nil {
		return err
	}
	mvIdx := int(m.MB2BXY[topRightXY]) + 3*m.BStride
	if err := checkRange(len(m.MotionVal[list]), mvIdx, 1); err != nil {
		return err
	}
	cache.MV[list][cacheIdx] = m.MotionVal[list][mvIdx]
	refIdx := 4*topRightXY + 2
	if err := checkRange(len(m.RefIndex[list]), refIdx, 1); err != nil {
		return err
	}
	cache.Ref[list][cacheIdx] = m.RefIndex[list][refIdx]
	return nil
}

func (m *macroblockTables) copyTopLeftMotion(cache *macroblockMotionCache, topLeftXY int, topLeftPartition int, cacheIdx int, list int) error {
	if err := m.checkMBXY(topLeftXY); err != nil {
		return err
	}
	mvIdx := int(m.MB2BXY[topLeftXY]) + 3 + m.BStride + (topLeftPartition & (2 * m.BStride))
	if err := checkRange(len(m.MotionVal[list]), mvIdx, 1); err != nil {
		return err
	}
	cache.MV[list][cacheIdx] = m.MotionVal[list][mvIdx]
	refIdx := 4*topLeftXY + 1 + (topLeftPartition & 2)
	if err := checkRange(len(m.RefIndex[list]), refIdx, 1); err != nil {
		return err
	}
	cache.Ref[list][cacheIdx] = m.RefIndex[list][refIdx]
	return nil
}

func (m *macroblockTables) fillMVDNeighbors(cache *macroblockMotionCache, n motionDecodeNeighbors, leftBlock *[32]uint8, list int, base int) error {
	if usesList(n.TopType, list) {
		if err := m.checkMBXY(n.TopXY); err != nil {
			return err
		}
		src := int(m.MB2BRXY[n.TopXY])
		if err := checkRange(len(m.MVDTable[list]), src, 4); err != nil {
			return err
		}
		copy(cache.MVD[list][base-8:base-8+4], m.MVDTable[list][src:src+4])
	} else {
		clearMVDRow(&cache.MVD[list], base-8, 4)
	}
	if usesList(n.LeftType[h264LeftTop], list) {
		if err := m.checkMBXY(n.LeftXY[h264LeftTop]); err != nil {
			return err
		}
		src := int(m.MB2BRXY[n.LeftXY[h264LeftTop]]) + 6
		for row := 0; row < 2; row++ {
			idx := src - int(leftBlock[row])
			if err := checkRange(len(m.MVDTable[list]), idx, 1); err != nil {
				return err
			}
			cache.MVD[list][base-1+row*8] = m.MVDTable[list][idx]
		}
	} else {
		cache.MVD[list][base-1+0*8] = [2]uint8{}
		cache.MVD[list][base-1+1*8] = [2]uint8{}
	}
	if usesList(n.LeftType[h264LeftBot], list) {
		if err := m.checkMBXY(n.LeftXY[h264LeftBot]); err != nil {
			return err
		}
		src := int(m.MB2BRXY[n.LeftXY[h264LeftBot]]) + 6
		for row := 0; row < 2; row++ {
			idx := src - int(leftBlock[2+row])
			if err := checkRange(len(m.MVDTable[list]), idx, 1); err != nil {
				return err
			}
			cache.MVD[list][base-1+(2+row)*8] = m.MVDTable[list][idx]
		}
	} else {
		cache.MVD[list][base-1+2*8] = [2]uint8{}
		cache.MVD[list][base-1+3*8] = [2]uint8{}
	}
	cache.MVD[list][base+2+8*0] = [2]uint8{}
	cache.MVD[list][base+2+8*2] = [2]uint8{}
	return nil
}

func (m *macroblockTables) fillDirectNeighbors(cache *macroblockMotionCache, n motionDecodeNeighbors, leftBlock *[32]uint8, base int) error {
	if isDirect(n.TopType) {
		fillDirectRectangle(&cache.Direct, base-8, 4, 1, 8, uint8(MBTypeDirect2>>1))
	} else if is8x8(n.TopType) {
		b8XY := 4 * n.TopXY
		if err := checkRange(len(m.DirectTable), b8XY+2, 2); err != nil {
			return err
		}
		cache.Direct[base+0-8] = m.DirectTable[b8XY+2]
		cache.Direct[base+2-8] = m.DirectTable[b8XY+3]
	} else {
		fillDirectRectangle(&cache.Direct, base-8, 4, 1, 8, uint8(MBType16x16>>1))
	}

	if isDirect(n.LeftType[h264LeftTop]) {
		cache.Direct[base-1+0*8] = uint8(MBTypeDirect2 >> 1)
	} else if is8x8(n.LeftType[h264LeftTop]) {
		idx := 4*n.LeftXY[h264LeftTop] + 1 + int(leftBlock[0]&^1)
		if err := checkRange(len(m.DirectTable), idx, 1); err != nil {
			return err
		}
		cache.Direct[base-1+0*8] = m.DirectTable[idx]
	} else {
		cache.Direct[base-1+0*8] = uint8(MBType16x16 >> 1)
	}
	if isDirect(n.LeftType[h264LeftBot]) {
		cache.Direct[base-1+2*8] = uint8(MBTypeDirect2 >> 1)
	} else if is8x8(n.LeftType[h264LeftBot]) {
		idx := 4*n.LeftXY[h264LeftBot] + 1 + int(leftBlock[2]&^1)
		if err := checkRange(len(m.DirectTable), idx, 1); err != nil {
			return err
		}
		cache.Direct[base-1+2*8] = m.DirectTable[idx]
	} else {
		cache.Direct[base-1+2*8] = uint8(MBType16x16 >> 1)
	}
	return nil
}

func (m *macroblockTables) writeBackMotionList(mbXY int, mbType uint32, list int, cache *macroblockMotionCache, cabac bool) error {
	if cache == nil || list < 0 || list > 1 {
		return ErrInvalidData
	}
	if err := m.checkCodedMBXY(mbXY); err != nil {
		return err
	}
	mbX := mbXY % m.MBStride
	mbY := mbXY / m.MBStride
	bXY := 4*mbX + 4*mbY*m.BStride
	b8XY := 4 * mbXY
	if err := checkRange(len(m.MotionVal[list]), bXY, 3*m.BStride+4); err != nil {
		return err
	}
	base := int(h264Scan8[0])
	for row := 0; row < 4; row++ {
		dst := bXY + row*m.BStride
		copy(m.MotionVal[list][dst:dst+4], cache.MV[list][base+row*8:base+row*8+4])
	}
	if cabac {
		dst := int(m.MB2BRXY[mbXY])
		if err := checkRange(len(m.MVDTable[list]), dst, 8); err != nil {
			return err
		}
		if isSkip(mbType) {
			for i := 0; i < 8; i++ {
				m.MVDTable[list][dst+i] = [2]uint8{}
			}
		} else {
			copy(m.MVDTable[list][dst:dst+4], cache.MVD[list][base+8*3:base+8*3+4])
			m.MVDTable[list][dst+6] = cache.MVD[list][base+3+8*0]
			m.MVDTable[list][dst+5] = cache.MVD[list][base+3+8*1]
			m.MVDTable[list][dst+4] = cache.MVD[list][base+3+8*2]
		}
	}
	if err := checkRange(len(m.RefIndex[list]), b8XY, 4); err != nil {
		return err
	}
	m.RefIndex[list][b8XY+0] = cache.Ref[list][h264Scan8[0]]
	m.RefIndex[list][b8XY+1] = cache.Ref[list][h264Scan8[4]]
	m.RefIndex[list][b8XY+2] = cache.Ref[list][h264Scan8[8]]
	m.RefIndex[list][b8XY+3] = cache.Ref[list][h264Scan8[12]]
	return nil
}

func (m *macroblockTables) writeBackMotion(mbXY int, mbType uint32, sliceTypeNoS int32, cabac bool, subMBType *[4]uint32, cache *macroblockMotionCache) error {
	if cache == nil {
		return ErrInvalidData
	}
	if err := m.checkCodedMBXY(mbXY); err != nil {
		return err
	}
	if usesList(mbType, 0) {
		if err := m.writeBackMotionList(mbXY, mbType, 0, cache, cabac); err != nil {
			return err
		}
	} else {
		b8XY := 4 * mbXY
		if err := checkRange(len(m.RefIndex[0]), b8XY, 4); err != nil {
			return err
		}
		m.RefIndex[0][b8XY+0] = h264ListNotUsed
		m.RefIndex[0][b8XY+1] = h264ListNotUsed
		m.RefIndex[0][b8XY+2] = h264ListNotUsed
		m.RefIndex[0][b8XY+3] = h264ListNotUsed
	}
	if usesList(mbType, 1) {
		if err := m.writeBackMotionList(mbXY, mbType, 1, cache, cabac); err != nil {
			return err
		}
	}
	if sliceTypeNoS == PictureTypeB && cabac && is8x8(mbType) {
		if subMBType == nil {
			return ErrInvalidData
		}
		base := 4 * mbXY
		if err := checkRange(len(m.DirectTable), base+1, 3); err != nil {
			return err
		}
		// FFmpeg's write_back_motion updates only sub partitions 1..3.
		// Slot 0 is left untouched; normally it stays the zero/default
		// neighbor sentinel.
		m.DirectTable[base+1] = uint8(subMBType[1] >> 1)
		m.DirectTable[base+2] = uint8(subMBType[2] >> 1)
		m.DirectTable[base+3] = uint8(subMBType[3] >> 1)
	}
	return nil
}

func clearMotionRow(cache *[h264MotionCacheSize][2]int16, start int, count int) {
	for i := 0; i < count; i++ {
		cache[start+i] = [2]int16{}
	}
}

func clearMVDRow(cache *[h264MotionCacheSize][2]uint8, start int, count int) {
	for i := 0; i < count; i++ {
		cache[start+i] = [2]uint8{}
	}
}

func fillRefRow(cache *[h264MotionCacheSize]int8, start int, count int, value int8) {
	for i := 0; i < count; i++ {
		cache[start+i] = value
	}
}

func fillDirectRectangle(cache *[h264MotionCacheSize]uint8, start int, width int, height int, stride int, value uint8) {
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			cache[start+y*stride+x] = value
		}
	}
}

func usesList(mbType uint32, list int) bool {
	if list < 0 || list > 1 {
		return false
	}
	return mbType&((MBTypeP0L0|MBTypeP1L0)<<uint(2*list)) != 0
}

func isInter(mbType uint32) bool {
	return mbType&(MBType16x16|MBType16x8|MBType8x16|MBType8x8) != 0
}

func is8x8(mbType uint32) bool {
	return mbType&MBType8x8 != 0
}

func isSkip(mbType uint32) bool {
	return mbType&MBTypeSkip != 0
}
