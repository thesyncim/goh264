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

	mmcoEnd          uint32 = 0
	mmcoShort2Unused uint32 = 1
	mmcoLong2Unused  uint32 = 2
	mmcoShort2Long   uint32 = 3
	mmcoSetMaxLong   uint32 = 4
	mmcoReset        uint32 = 5
	mmcoLong         uint32 = 6

	maxRefMods   = 32
	maxMMCOCount = 67
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
	RefModifications    [2][maxRefMods]RefModification
	NBRefModifications  [2]uint32
	PredWeightTable     PredWeightTable
	ExplicitRefMarking  int32
	MMCO                [maxMMCOCount]MMCO
	NBMMCO              uint32
	CABACInitIDC        uint32
	QScale              uint32
	SPForSwitchFlag     int32
	SliceQSDelta        int32
	DeblockingFilter    int32
	SliceAlphaC0Offset  int32
	SliceBetaOffset     int32
}

type RefModification struct {
	Op  uint32
	Val uint32
}

type MMCO struct {
	Opcode      uint32
	ShortPicNum uint32
	LongArg     uint32
}

type PredWeightTable struct {
	UseWeight             int32
	UseWeightChroma       int32
	LumaLog2WeightDenom   uint32
	ChromaLog2WeightDenom uint32
	LumaWeightFlag        [2]int32
	ChromaWeightFlag      [2]int32
	LumaWeight            [48][2][2]int32
	ChromaWeight          [48][2][2][2]int32
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
	if sh.SliceTypeNoS != PictureTypeI {
		if err := decodeRefPicListReordering(&gb, sh); err != nil {
			sh.RefCount[0] = 0
			sh.RefCount[1] = 0
			return nil, err
		}
	}

	if (sh.PPS.WeightedPred != 0 && sh.SliceTypeNoS == PictureTypeP) ||
		(sh.PPS.WeightedBipredIDC == 1 && sh.SliceTypeNoS == PictureTypeB) {
		if err := predWeightTable(&gb, sh); err != nil {
			return nil, err
		}
	}

	if nal.RefIDC != 0 {
		if err := decodeRefPicMarking(&gb, sh); err != nil {
			return nil, err
		}
	}

	if sh.SliceTypeNoS != PictureTypeI && sh.PPS.CABAC != 0 {
		cabacInitIDC, err := gb.readUEGolomb31()
		if err != nil {
			return nil, err
		}
		if cabacInitIDC > 2 {
			return nil, ErrInvalidData
		}
		sh.CABACInitIDC = cabacInitIDC
	}

	sliceQPDelta, err := gb.readSEGolombLong()
	if err != nil {
		return nil, err
	}
	qscale := sh.PPS.InitQP + sliceQPDelta
	maxQP := int32(51 + 6*(sh.SPS.BitDepthLuma-8))
	if qscale < 0 || qscale > maxQP {
		return nil, ErrInvalidData
	}
	sh.QScale = uint32(qscale)

	if sh.SliceType == PictureTypeSP {
		spForSwitchFlag, err := gb.readBit()
		if err != nil {
			return nil, err
		}
		sh.SPForSwitchFlag = int32(spForSwitchFlag)
	}
	if sh.SliceType == PictureTypeSP || sh.SliceType == PictureTypeSI {
		sh.SliceQSDelta, err = gb.readSEGolombLong()
		if err != nil {
			return nil, err
		}
	}

	sh.DeblockingFilter = 1
	if sh.PPS.DeblockingFilterParametersPresent != 0 {
		disableIDC, err := gb.readUEGolomb31()
		if err != nil {
			return nil, err
		}
		if disableIDC > 2 {
			return nil, ErrInvalidData
		}
		sh.DeblockingFilter = int32(disableIDC)
		if sh.DeblockingFilter < 2 {
			sh.DeblockingFilter ^= 1
		}
		if sh.DeblockingFilter != 0 {
			alpha, err := gb.readSEGolombLong()
			if err != nil {
				return nil, err
			}
			beta, err := gb.readSEGolombLong()
			if err != nil {
				return nil, err
			}
			if alpha > 6 || alpha < -6 || beta > 6 || beta < -6 {
				return nil, ErrInvalidData
			}
			sh.SliceAlphaC0Offset = alpha * 2
			sh.SliceBetaOffset = beta * 2
		}
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

func decodeRefPicListReordering(gb *bitReader, sh *SliceHeader) error {
	sh.NBRefModifications[0] = 0
	sh.NBRefModifications[1] = 0

	for list := int32(0); list < sh.ListCount; list++ {
		flag, err := gb.readBit()
		if err != nil {
			return err
		}
		if flag == 0 {
			continue
		}

		for index := uint32(0); ; index++ {
			op, err := gb.readUEGolomb31()
			if err != nil {
				return err
			}
			if op == 3 {
				break
			}
			if index >= sh.RefCount[list] || index >= maxRefMods {
				return ErrInvalidData
			}
			if op > 2 {
				return ErrInvalidData
			}
			val, err := gb.readUEGolombLong()
			if err != nil {
				return err
			}
			sh.RefModifications[list][index] = RefModification{Op: op, Val: val}
			sh.NBRefModifications[list]++
		}
	}
	return nil
}

func predWeightTable(gb *bitReader, sh *SliceHeader) error {
	pwt := &sh.PredWeightTable
	pwt.UseWeight = 0
	pwt.UseWeightChroma = 0

	lumaDenom, err := gb.readUEGolomb31()
	if err != nil {
		return err
	}
	pwt.LumaLog2WeightDenom = lumaDenom
	if pwt.LumaLog2WeightDenom > 7 {
		pwt.LumaLog2WeightDenom = 0
	}
	lumaDef := int32(1) << pwt.LumaLog2WeightDenom

	chromaDef := int32(0)
	if sh.SPS.ChromaFormatIDC != 0 {
		chromaDenom, err := gb.readUEGolomb31()
		if err != nil {
			return err
		}
		pwt.ChromaLog2WeightDenom = chromaDenom
		if pwt.ChromaLog2WeightDenom > 7 {
			pwt.ChromaLog2WeightDenom = 0
		}
		chromaDef = int32(1) << pwt.ChromaLog2WeightDenom
	}

	for list := 0; list < 2; list++ {
		pwt.LumaWeightFlag[list] = 0
		pwt.ChromaWeightFlag[list] = 0
		for i := uint32(0); i < sh.RefCount[list]; i++ {
			lumaFlag, err := gb.readBit()
			if err != nil {
				return err
			}
			if lumaFlag != 0 {
				if pwt.LumaWeight[i][list][0], err = gb.readSEGolombLong(); err != nil {
					return err
				}
				if pwt.LumaWeight[i][list][1], err = gb.readSEGolombLong(); err != nil {
					return err
				}
				if !fitsInt8(pwt.LumaWeight[i][list][0]) || !fitsInt8(pwt.LumaWeight[i][list][1]) {
					return ErrInvalidData
				}
				if pwt.LumaWeight[i][list][0] != lumaDef || pwt.LumaWeight[i][list][1] != 0 {
					pwt.UseWeight = 1
					pwt.LumaWeightFlag[list] = 1
				}
			} else {
				pwt.LumaWeight[i][list][0] = lumaDef
				pwt.LumaWeight[i][list][1] = 0
			}

			if sh.SPS.ChromaFormatIDC != 0 {
				chromaFlag, err := gb.readBit()
				if err != nil {
					return err
				}
				if chromaFlag != 0 {
					for j := 0; j < 2; j++ {
						if pwt.ChromaWeight[i][list][j][0], err = gb.readSEGolombLong(); err != nil {
							return err
						}
						if pwt.ChromaWeight[i][list][j][1], err = gb.readSEGolombLong(); err != nil {
							return err
						}
						if !fitsInt8(pwt.ChromaWeight[i][list][j][0]) || !fitsInt8(pwt.ChromaWeight[i][list][j][1]) {
							return ErrInvalidData
						}
						if pwt.ChromaWeight[i][list][j][0] != chromaDef || pwt.ChromaWeight[i][list][j][1] != 0 {
							pwt.UseWeightChroma = 1
							pwt.ChromaWeightFlag[list] = 1
						}
					}
				} else {
					for j := 0; j < 2; j++ {
						pwt.ChromaWeight[i][list][j][0] = chromaDef
						pwt.ChromaWeight[i][list][j][1] = 0
					}
				}
			}

			if sh.PictureStructure == PictureFrame {
				pwt.LumaWeight[16+2*i][list][0] = pwt.LumaWeight[i][list][0]
				pwt.LumaWeight[16+2*i+1][list][0] = pwt.LumaWeight[i][list][0]
				pwt.LumaWeight[16+2*i][list][1] = pwt.LumaWeight[i][list][1]
				pwt.LumaWeight[16+2*i+1][list][1] = pwt.LumaWeight[i][list][1]
				if sh.SPS.ChromaFormatIDC != 0 {
					for j := 0; j < 2; j++ {
						pwt.ChromaWeight[16+2*i][list][j][0] = pwt.ChromaWeight[i][list][j][0]
						pwt.ChromaWeight[16+2*i+1][list][j][0] = pwt.ChromaWeight[i][list][j][0]
						pwt.ChromaWeight[16+2*i][list][j][1] = pwt.ChromaWeight[i][list][j][1]
						pwt.ChromaWeight[16+2*i+1][list][j][1] = pwt.ChromaWeight[i][list][j][1]
					}
				}
			}
		}
		if sh.SliceTypeNoS != PictureTypeB {
			break
		}
	}
	if pwt.UseWeightChroma != 0 {
		pwt.UseWeight = 1
	}
	return nil
}

func decodeRefPicMarking(gb *bitReader, sh *SliceHeader) error {
	nbMMCO := uint32(0)
	if sh.NALType == NALIDRSlice {
		if err := gb.skipBits(1); err != nil {
			return err
		}
		longTermReferenceFlag, err := gb.readBit()
		if err != nil {
			return err
		}
		if longTermReferenceFlag != 0 {
			sh.MMCO[0].Opcode = mmcoLong
			sh.MMCO[0].LongArg = 0
			nbMMCO = 1
		}
		sh.ExplicitRefMarking = 1
		sh.NBMMCO = nbMMCO
		return nil
	}

	explicit, err := gb.readBit()
	if err != nil {
		return err
	}
	sh.ExplicitRefMarking = int32(explicit)
	if explicit == 0 {
		sh.NBMMCO = 0
		return nil
	}

	for i := uint32(0); i < maxMMCOCount; i++ {
		opcode, err := gb.readUEGolomb31()
		if err != nil {
			return err
		}
		sh.MMCO[i].Opcode = opcode
		if opcode == mmcoShort2Unused || opcode == mmcoShort2Long {
			diff, err := gb.readUEGolombLong()
			if err != nil {
				return err
			}
			sh.MMCO[i].ShortPicNum = (sh.CurrPicNum - diff - 1) & (sh.MaxPicNum - 1)
		}
		if opcode == mmcoShort2Long || opcode == mmcoLong2Unused ||
			opcode == mmcoLong || opcode == mmcoSetMaxLong {
			longArg, err := gb.readUEGolomb31()
			if err != nil {
				return err
			}
			if longArg >= 32 ||
				(longArg >= 16 && !(opcode == mmcoSetMaxLong && longArg == 16) &&
					!(opcode == mmcoLong2Unused && sh.PictureStructure != PictureFrame)) {
				sh.NBMMCO = i
				return ErrInvalidData
			}
			sh.MMCO[i].LongArg = longArg
		}
		if opcode > mmcoLong {
			sh.NBMMCO = i
			return ErrInvalidData
		}
		if opcode == mmcoEnd {
			nbMMCO = i
			break
		}
		nbMMCO = i + 1
	}

	sh.NBMMCO = nbMMCO
	return nil
}

func fitsInt8(v int32) bool {
	return v >= -128 && v <= 127
}
