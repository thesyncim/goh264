// SPDX-License-Identifier: LGPL-2.1-or-later
//
// Source-shaped port of the first H.264 PPS metadata slice from FFmpeg
// n8.0.1 libavcodec/h264_ps.c ff_h264_decode_picture_parameter_set.

package h264

const maxPPSCount = 256

type PPS struct {
	PPSID                             uint32
	SPSID                             uint32
	SPS                               *SPS
	CABAC                             int32
	PicOrderPresent                   int32
	SliceGroupCount                   uint32
	MBSliceGroupMapType               uint32
	RefCount                          [2]uint32
	WeightedPred                      int32
	WeightedBipredIDC                 uint32
	InitQP                            int32
	InitQS                            int32
	ChromaQPIndexOffset               [2]int32
	DeblockingFilterParametersPresent int32
	ConstrainedIntraPred              int32
	RedundantPicCntPresent            int32
	Transform8x8Mode                  int32
	PicScalingMatrixPresentFlag       int32
	PicScalingMatrixPresentMask       uint16
	ScalingMatrix4                    [6][16]uint8
	ScalingMatrix8                    [6][64]uint8
	ChromaQPTable                     [2][qpMaxNum + 1]uint8
	ChromaQPDiff                      int32
	Dequant4Buffer                    [6][qpMaxNum + 1][16]uint32
	Dequant8Buffer                    [6][qpMaxNum + 1][64]uint32
}

func DecodePPS(rbsp []byte, spsList *[maxSPSCount]*SPS) (*PPS, error) {
	gb, err := newRBSPBitReader(rbsp)
	if err != nil {
		return nil, err
	}

	ppsID, err := gb.readUEGolombLong()
	if err != nil {
		return nil, err
	}
	if ppsID >= maxPPSCount {
		return nil, ErrInvalidData
	}

	pps := &PPS{PPSID: ppsID}
	pps.SPSID, err = gb.readUEGolomb31()
	if err != nil {
		return nil, err
	}
	if pps.SPSID >= maxSPSCount || spsList[pps.SPSID] == nil {
		return nil, ErrInvalidData
	}
	pps.SPS = spsList[pps.SPSID]
	pps.ScalingMatrix4 = pps.SPS.ScalingMatrix4
	pps.ScalingMatrix8 = pps.SPS.ScalingMatrix8

	if pps.SPS.BitDepthLuma > 14 {
		return nil, ErrInvalidData
	}
	if pps.SPS.BitDepthLuma == 11 || pps.SPS.BitDepthLuma == 13 {
		return nil, ErrUnsupported
	}

	flag, err := gb.readBit()
	if err != nil {
		return nil, err
	}
	pps.CABAC = int32(flag)

	flag, err = gb.readBit()
	if err != nil {
		return nil, err
	}
	pps.PicOrderPresent = int32(flag)

	sliceGroupCountMinus1, err := gb.readUEGolombLong()
	if err != nil {
		return nil, err
	}
	pps.SliceGroupCount = sliceGroupCountMinus1 + 1
	if pps.SliceGroupCount > 1 {
		mapType, err := gb.readUEGolombLong()
		if err != nil {
			return nil, err
		}
		pps.MBSliceGroupMapType = mapType
		return nil, ErrUnsupported
	}

	for i := 0; i < 2; i++ {
		refCountMinus1, err := gb.readUEGolombLong()
		if err != nil {
			return nil, err
		}
		pps.RefCount[i] = refCountMinus1 + 1
		if pps.RefCount[i]-1 > 31 {
			return nil, ErrInvalidData
		}
	}

	qpBDOffset := int32(6 * (pps.SPS.BitDepthLuma - 8))
	flag, err = gb.readBit()
	if err != nil {
		return nil, err
	}
	pps.WeightedPred = int32(flag)

	weightedBipredIDC, err := gb.readBits(2)
	if err != nil {
		return nil, err
	}
	pps.WeightedBipredIDC = weightedBipredIDC

	initQPDelta, err := gb.readSEGolombLong()
	if err != nil {
		return nil, err
	}
	pps.InitQP = initQPDelta + 26 + qpBDOffset

	initQSDelta, err := gb.readSEGolombLong()
	if err != nil {
		return nil, err
	}
	pps.InitQS = initQSDelta + 26 + qpBDOffset

	pps.ChromaQPIndexOffset[0], err = gb.readSEGolombLong()
	if err != nil {
		return nil, err
	}
	if pps.ChromaQPIndexOffset[0] < -12 || pps.ChromaQPIndexOffset[0] > 12 {
		return nil, ErrInvalidData
	}

	flag, err = gb.readBit()
	if err != nil {
		return nil, err
	}
	pps.DeblockingFilterParametersPresent = int32(flag)

	flag, err = gb.readBit()
	if err != nil {
		return nil, err
	}
	pps.ConstrainedIntraPred = int32(flag)

	flag, err = gb.readBit()
	if err != nil {
		return nil, err
	}
	pps.RedundantPicCntPresent = int32(flag)

	if gb.bitsLeft() > 0 && moreRBSPDataInPPS(pps.SPS) {
		flag, err = gb.readBit()
		if err != nil {
			return nil, err
		}
		pps.Transform8x8Mode = int32(flag)

		flag, err = gb.readBit()
		if err != nil {
			return nil, err
		}
		pps.PicScalingMatrixPresentFlag = int32(flag)
		if flag != 0 {
			if err := decodeScalingMatrices(&gb, pps.SPS, pps.SPS, false, pps.Transform8x8Mode != 0, true, &pps.PicScalingMatrixPresentMask, &pps.ScalingMatrix4, &pps.ScalingMatrix8); err != nil {
				return nil, err
			}
		}

		pps.ChromaQPIndexOffset[1], err = gb.readSEGolombLong()
		if err != nil {
			return nil, err
		}
		if pps.ChromaQPIndexOffset[1] < -12 || pps.ChromaQPIndexOffset[1] > 12 {
			return nil, ErrInvalidData
		}
	} else {
		pps.ChromaQPIndexOffset[1] = pps.ChromaQPIndexOffset[0]
	}

	if pps.ChromaQPIndexOffset[0] != pps.ChromaQPIndexOffset[1] {
		pps.ChromaQPDiff = 1
	}
	buildQPTable(pps, 0, pps.ChromaQPIndexOffset[0], pps.SPS.BitDepthLuma)
	buildQPTable(pps, 1, pps.ChromaQPIndexOffset[1], pps.SPS.BitDepthLuma)
	initDequantTables(pps, pps.SPS)

	return pps, nil
}

func moreRBSPDataInPPS(sps *SPS) bool {
	profileIDC := sps.ProfileIDC
	return !((profileIDC == 66 || profileIDC == 77 || profileIDC == 88) && (sps.ConstraintSetFlags&7) != 0)
}

func buildQPTable(pps *PPS, table int, index int32, depth int32) {
	maxQP := int32(51 + 6*(depth-8))
	for i := int32(0); i <= maxQP; i++ {
		pps.ChromaQPTable[table][i] = h264ChromaQP(depth, uint32(clipInt32(i+index, 0, maxQP)))
	}
}

func initDequantTables(pps *PPS, sps *SPS) {
	initDequant4CoeffTable(pps, sps)
	if pps.Transform8x8Mode != 0 {
		initDequant8CoeffTable(pps, sps)
	}
	if sps.TransformBypass != 0 {
		for i := 0; i < 6; i++ {
			for x := 0; x < 16; x++ {
				pps.Dequant4Buffer[i][0][x] = 1 << 6
			}
		}
		if pps.Transform8x8Mode != 0 {
			for i := 0; i < 6; i++ {
				for x := 0; x < 64; x++ {
					pps.Dequant8Buffer[i][0][x] = 1 << 6
				}
			}
		}
	}
}

func initDequant4CoeffTable(pps *PPS, sps *SPS) {
	maxQP := int(51 + 6*(sps.BitDepthLuma-8))
	for i := 0; i < 6; i++ {
		for q := 0; q <= maxQP; q++ {
			shift := h264QuantDiv6[q] + 2
			idx := h264QuantRem6[q]
			for x := 0; x < 16; x++ {
				dst := (x >> 2) | ((x << 2) & 0x0f)
				initIdx := (x & 1) + ((x >> 2) & 1)
				pps.Dequant4Buffer[i][q][dst] =
					uint32(h264Dequant4CoeffInit[idx][initIdx]) *
						uint32(pps.ScalingMatrix4[i][x]) << shift
			}
		}
	}
}

func initDequant8CoeffTable(pps *PPS, sps *SPS) {
	maxQP := int(51 + 6*(sps.BitDepthLuma-8))
	for i := 0; i < 6; i++ {
		for q := 0; q <= maxQP; q++ {
			shift := h264QuantDiv6[q]
			idx := h264QuantRem6[q]
			for x := 0; x < 64; x++ {
				dst := (x >> 3) | ((x & 7) << 3)
				initIdx := h264Dequant8CoeffInitScan[((x>>1)&12)|(x&3)]
				pps.Dequant8Buffer[i][q][dst] =
					uint32(h264Dequant8CoeffInit[idx][initIdx]) *
						uint32(pps.ScalingMatrix8[i][x]) << shift
			}
		}
	}
}

func clipInt32(v, lo, hi int32) int32 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
