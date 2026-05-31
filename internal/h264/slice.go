// SPDX-License-Identifier: LGPL-2.1-or-later
//
// Source-shaped port of the H.264 slice-header front matter from FFmpeg
// n8.0.1 libavcodec/h264_slice.c h264_slice_header_parse and
// libavcodec/h264_parse.c ff_h264_parse_ref_count.

package h264

const (
	PictureTypeI  int32 = 1
	PictureTypeP  int32 = 2
	PictureTypeB  int32 = 3
	PictureTypeSI int32 = 5
	PictureTypeSP int32 = 6

	PictureTopField    int32 = 1
	PictureBottomField int32 = 2
	PictureFrame       int32 = 3
)

var h264GolombToPictureType = [5]int32{
	PictureTypeP,
	PictureTypeB,
	PictureTypeI,
	PictureTypeSP,
	PictureTypeSI,
}

type SliceHeader struct {
	NALType             NALUnitType
	NALRefIDC           uint8
	FirstMBAddr         uint32
	SliceType           int32
	SliceTypeNoS        int32
	SliceTypeFixed      int32
	PPSID               uint32
	PPS                 *PPS
	SPS                 *SPS
	FrameNum            uint32
	PictureStructure    int32
	MBFieldDecodingFlag int32
	CurrPicNum          uint32
	MaxPicNum           uint32
	IDRPicID            uint32
	POCLSB              uint32
	DeltaPOCBottom      int32
	DeltaPOC            [2]int32
	RedundantPicCount   uint32
	DirectSpatialMVPred int32
	ListCount           int32
	RefCount            [2]uint32
}

func ParseSliceHeader(nal NALUnit, ppsList *[maxPPSCount]*PPS) (*SliceHeader, error) {
	if nal.Type != NALSlice && nal.Type != NALIDRSlice {
		return nil, ErrInvalidData
	}

	gb, err := newRBSPBitReader(nal.RBSP)
	if err != nil {
		return nil, err
	}

	sh := &SliceHeader{
		NALType:   nal.Type,
		NALRefIDC: nal.RefIDC,
	}

	sh.FirstMBAddr, err = gb.readUEGolombLong()
	if err != nil {
		return nil, err
	}

	sliceType, err := gb.readUEGolomb31()
	if err != nil {
		return nil, err
	}
	if sliceType > 9 {
		return nil, ErrInvalidData
	}
	if sliceType > 4 {
		sliceType -= 5
		sh.SliceTypeFixed = 1
	}

	sh.SliceType = h264GolombToPictureType[sliceType]
	sh.SliceTypeNoS = sh.SliceType & 3
	if nal.Type == NALIDRSlice && sh.SliceTypeNoS != PictureTypeI {
		return nil, ErrInvalidData
	}

	sh.PPSID, err = gb.readUEGolombLong()
	if err != nil {
		return nil, err
	}
	if sh.PPSID >= maxPPSCount || ppsList[sh.PPSID] == nil {
		return nil, ErrInvalidData
	}
	sh.PPS = ppsList[sh.PPSID]
	sh.SPS = sh.PPS.SPS

	frameNum, err := gb.readBits(uint32(sh.SPS.Log2MaxFrameNum))
	if err != nil {
		return nil, err
	}
	sh.FrameNum = frameNum

	if sh.SPS.FrameMBSOnlyFlag != 0 {
		sh.PictureStructure = PictureFrame
	} else {
		if sh.SPS.Direct8x8InferenceFlag == 0 && sh.SliceType == PictureTypeB {
			return nil, ErrInvalidData
		}
		fieldPicFlag, err := gb.readBit()
		if err != nil {
			return nil, err
		}
		if fieldPicFlag != 0 {
			bottomFieldFlag, err := gb.readBit()
			if err != nil {
				return nil, err
			}
			sh.PictureStructure = PictureTopField + int32(bottomFieldFlag)
		} else {
			sh.PictureStructure = PictureFrame
		}
	}
	if sh.PictureStructure != PictureFrame {
		sh.MBFieldDecodingFlag = 1
	}

	if sh.PictureStructure == PictureFrame {
		sh.CurrPicNum = sh.FrameNum
		sh.MaxPicNum = uint32(1) << uint32(sh.SPS.Log2MaxFrameNum)
	} else {
		sh.CurrPicNum = 2*sh.FrameNum + 1
		sh.MaxPicNum = uint32(1) << uint32(sh.SPS.Log2MaxFrameNum+1)
	}

	if nal.Type == NALIDRSlice {
		idrPicID, err := gb.readUEGolombLong()
		if err != nil {
			return nil, err
		}
		if idrPicID < 65536 {
			sh.IDRPicID = idrPicID
		}
	}

	if sh.SPS.PocType == 0 {
		pocLSB, err := gb.readBits(uint32(sh.SPS.Log2MaxPocLSB))
		if err != nil {
			return nil, err
		}
		sh.POCLSB = pocLSB

		if sh.PPS.PicOrderPresent == 1 && sh.PictureStructure == PictureFrame {
			sh.DeltaPOCBottom, err = gb.readSEGolombLong()
			if err != nil {
				return nil, err
			}
		}
	}

	if sh.SPS.PocType == 1 && sh.SPS.DeltaPicOrderAlwaysZeroFlag == 0 {
		sh.DeltaPOC[0], err = gb.readSEGolombLong()
		if err != nil {
			return nil, err
		}
		if sh.PPS.PicOrderPresent == 1 && sh.PictureStructure == PictureFrame {
			sh.DeltaPOC[1], err = gb.readSEGolombLong()
			if err != nil {
				return nil, err
			}
		}
	}

	if sh.PPS.RedundantPicCntPresent != 0 {
		sh.RedundantPicCount, err = gb.readUEGolombLong()
		if err != nil {
			return nil, err
		}
	}

	if sh.SliceTypeNoS == PictureTypeB {
		directSpatialMVPred, err := gb.readBit()
		if err != nil {
			return nil, err
		}
		sh.DirectSpatialMVPred = int32(directSpatialMVPred)
	}

	if err := parseRefCount(&gb, sh); err != nil {
		return nil, err
	}

	return sh, nil
}

func parseRefCount(gb *bitReader, sh *SliceHeader) error {
	sh.RefCount[0] = sh.PPS.RefCount[0]
	sh.RefCount[1] = sh.PPS.RefCount[1]

	if sh.SliceTypeNoS == PictureTypeI {
		sh.ListCount = 0
		sh.RefCount[0] = 0
		sh.RefCount[1] = 0
		return nil
	}

	max0, max1 := uint32(31), uint32(31)
	if sh.PictureStructure == PictureFrame {
		max0, max1 = 15, 15
	}

	override, err := gb.readBit()
	if err != nil {
		return err
	}
	if override != 0 {
		refCount0, err := gb.readUEGolombLong()
		if err != nil {
			return err
		}
		sh.RefCount[0] = refCount0 + 1
		if sh.SliceTypeNoS == PictureTypeB {
			refCount1, err := gb.readUEGolombLong()
			if err != nil {
				return err
			}
			sh.RefCount[1] = refCount1 + 1
		} else {
			sh.RefCount[1] = 1
		}
	}

	if sh.SliceTypeNoS == PictureTypeB {
		sh.ListCount = 2
	} else {
		sh.ListCount = 1
	}

	if sh.RefCount[0]-1 > max0 || (sh.ListCount == 2 && sh.RefCount[1]-1 > max1) {
		sh.RefCount[0] = 0
		sh.RefCount[1] = 0
		sh.ListCount = 0
		return ErrInvalidData
	}
	if sh.RefCount[1]-1 > max1 {
		sh.RefCount[1] = 0
	}
	return nil
}
