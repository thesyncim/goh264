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
	ChromaQPDiff                      int32
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
			if err := decodeScalingMatrices(&gb, pps.SPS.ChromaFormatIDC, pps.Transform8x8Mode != 0, &pps.PicScalingMatrixPresentMask); err != nil {
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

	return pps, nil
}

func moreRBSPDataInPPS(sps *SPS) bool {
	profileIDC := sps.ProfileIDC
	return !((profileIDC == 66 || profileIDC == 77 || profileIDC == 88) && (sps.ConstraintSetFlags&7) != 0)
}
