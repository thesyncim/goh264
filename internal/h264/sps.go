// SPDX-License-Identifier: LGPL-2.1-or-later
//
// Source-shaped port of the first H.264 SPS metadata slice from FFmpeg
// n8.0.1 libavcodec/h264_ps.c ff_h264_decode_seq_parameter_set.

package h264

const (
	maxSPSCount        = 32
	maxLog2MaxFrameNum = 12 + 4
	h264MaxDPBFrames   = 16
	h264MaxMBWidth     = 1055
	h264MaxMBHeight    = 1055
)

type SPS struct {
	SPSID                       uint32
	ProfileIDC                  int32
	LevelIDC                    int32
	ChromaFormatIDC             uint32
	TransformBypass             int32
	Log2MaxFrameNum             int32
	PocType                     uint32
	Log2MaxPocLSB               int32
	DeltaPicOrderAlwaysZeroFlag int32
	OffsetForNonRefPic          int32
	OffsetForTopToBottomField   int32
	PocCycleLength              uint32
	OffsetForRefFrame           [256]int32
	RefFrameCount               uint32
	GapsInFrameNumAllowedFlag   int32
	MBWidth                     int32
	MBHeight                    int32
	FrameMBSOnlyFlag            int32
	MBAFF                       int32
	Direct8x8InferenceFlag      int32
	Crop                        int32
	CropLeft                    uint32
	CropRight                   uint32
	CropTop                     uint32
	CropBottom                  uint32
	VUIParametersPresentFlag    int32
	BitDepthLuma                int32
	BitDepthChroma              int32
	ResidualColorTransformFlag  int32
	ConstraintSetFlags          int32
	ScalingMatrixPresent        int32
	ScalingMatrixPresentMask    uint16
	BitstreamRestrictionFlag    int32
	NumReorderFrames            int32
	MaxDecFrameBuffering        int32
	NALHRDParametersPresentFlag int32
	VCLHRDParametersPresentFlag int32
	PicStructPresentFlag        int32
	TimeOffsetLength            int32
	TimingInfoPresentFlag       int32
	NumUnitsInTick              uint32
	TimeScale                   uint32
	FixedFrameRateFlag          int32
	Width                       int32
	Height                      int32
}

func DecodeSPS(rbsp []byte) (*SPS, error) {
	gb, err := newRBSPBitReader(rbsp)
	if err != nil {
		return nil, err
	}
	sps := &SPS{
		TimeOffsetLength: 24,
		BitDepthLuma:     8,
		BitDepthChroma:   8,
	}

	profileIDC, err := gb.readBits(8)
	if err != nil {
		return nil, err
	}
	sps.ProfileIDC = int32(profileIDC)

	for i := uint32(0); i < 6; i++ {
		flag, err := gb.readBit()
		if err != nil {
			return nil, err
		}
		sps.ConstraintSetFlags |= int32(flag << i)
	}
	if err := gb.skipBits(2); err != nil {
		return nil, err
	}

	levelIDC, err := gb.readBits(8)
	if err != nil {
		return nil, err
	}
	sps.LevelIDC = int32(levelIDC)

	spsID, err := gb.readUEGolomb31()
	if err != nil {
		return nil, err
	}
	if spsID >= maxSPSCount {
		return nil, ErrInvalidData
	}
	sps.SPSID = spsID

	if isHighProfile(sps.ProfileIDC) {
		chromaFormatIDC, err := gb.readUEGolomb31()
		if err != nil {
			return nil, err
		}
		if chromaFormatIDC > 3 {
			return nil, ErrUnsupported
		}
		sps.ChromaFormatIDC = chromaFormatIDC
		if chromaFormatIDC == 3 {
			flag, err := gb.readBit()
			if err != nil {
				return nil, err
			}
			sps.ResidualColorTransformFlag = int32(flag)
			if flag != 0 {
				return nil, ErrUnsupported
			}
		}

		bitDepthLuma, err := gb.readUEGolomb31()
		if err != nil {
			return nil, err
		}
		bitDepthChroma, err := gb.readUEGolomb31()
		if err != nil {
			return nil, err
		}
		sps.BitDepthLuma = int32(bitDepthLuma) + 8
		sps.BitDepthChroma = int32(bitDepthChroma) + 8
		if sps.BitDepthChroma != sps.BitDepthLuma {
			return nil, ErrUnsupported
		}
		if sps.BitDepthLuma < 8 || sps.BitDepthLuma > 14 {
			return nil, ErrInvalidData
		}

		flag, err := gb.readBit()
		if err != nil {
			return nil, err
		}
		sps.TransformBypass = int32(flag)

		present, err := gb.readBit()
		if err != nil {
			return nil, err
		}
		if present != 0 {
			if err := decodeScalingMatrices(&gb, sps.ChromaFormatIDC, true, &sps.ScalingMatrixPresentMask); err != nil {
				return nil, err
			}
			sps.ScalingMatrixPresent = 1
		}
	} else {
		sps.ChromaFormatIDC = 1
	}

	log2MaxFrameNumMinus4, err := gb.readUEGolomb31()
	if err != nil {
		return nil, err
	}
	if log2MaxFrameNumMinus4 > maxLog2MaxFrameNum-4 {
		return nil, ErrInvalidData
	}
	sps.Log2MaxFrameNum = int32(log2MaxFrameNumMinus4) + 4

	pocType, err := gb.readUEGolomb31()
	if err != nil {
		return nil, err
	}
	sps.PocType = pocType
	switch pocType {
	case 0:
		t, err := gb.readUEGolomb31()
		if err != nil {
			return nil, err
		}
		if t > 12 {
			return nil, ErrInvalidData
		}
		sps.Log2MaxPocLSB = int32(t) + 4
	case 1:
		flag, err := gb.readBit()
		if err != nil {
			return nil, err
		}
		sps.DeltaPicOrderAlwaysZeroFlag = int32(flag)
		if sps.OffsetForNonRefPic, err = gb.readSEGolombLong(); err != nil {
			return nil, err
		}
		if sps.OffsetForTopToBottomField, err = gb.readSEGolombLong(); err != nil {
			return nil, err
		}
		sps.PocCycleLength, err = gb.readUEGolombLong()
		if err != nil {
			return nil, err
		}
		if sps.PocCycleLength >= uint32(len(sps.OffsetForRefFrame)) {
			return nil, ErrInvalidData
		}
		for i := uint32(0); i < sps.PocCycleLength; i++ {
			sps.OffsetForRefFrame[i], err = gb.readSEGolombLong()
			if err != nil {
				return nil, err
			}
		}
	case 2:
	default:
		return nil, ErrInvalidData
	}

	sps.RefFrameCount, err = gb.readUEGolomb31()
	if err != nil {
		return nil, err
	}
	if sps.RefFrameCount > h264MaxDPBFrames {
		return nil, ErrInvalidData
	}

	flag, err := gb.readBit()
	if err != nil {
		return nil, err
	}
	sps.GapsInFrameNumAllowedFlag = int32(flag)

	mbWidthMinus1, err := gb.readUEGolombLong()
	if err != nil {
		return nil, err
	}
	mbHeightMinus1, err := gb.readUEGolombLong()
	if err != nil {
		return nil, err
	}
	if mbWidthMinus1+1 > h264MaxMBWidth || mbHeightMinus1+1 > h264MaxMBHeight {
		return nil, ErrInvalidData
	}
	sps.MBWidth = int32(mbWidthMinus1 + 1)
	sps.MBHeight = int32(mbHeightMinus1 + 1)

	frameMBSOnlyFlag, err := gb.readBit()
	if err != nil {
		return nil, err
	}
	sps.FrameMBSOnlyFlag = int32(frameMBSOnlyFlag)
	sps.MBHeight *= 2 - sps.FrameMBSOnlyFlag

	if frameMBSOnlyFlag == 0 {
		mbAFF, err := gb.readBit()
		if err != nil {
			return nil, err
		}
		sps.MBAFF = int32(mbAFF)
	}

	direct8x8, err := gb.readBit()
	if err != nil {
		return nil, err
	}
	sps.Direct8x8InferenceFlag = int32(direct8x8)

	crop, err := gb.readBit()
	if err != nil {
		return nil, err
	}
	sps.Crop = int32(crop)
	if crop != 0 {
		if err := decodeCrop(&gb, sps); err != nil {
			return nil, err
		}
	}

	vuiPresent, err := gb.readBit()
	if err != nil {
		return nil, err
	}
	sps.VUIParametersPresentFlag = int32(vuiPresent)

	sps.Width = 16*sps.MBWidth - int32(sps.CropLeft+sps.CropRight)
	sps.Height = 16*sps.MBHeight - int32(sps.CropTop+sps.CropBottom)
	if sps.Width <= 0 || sps.Height <= 0 {
		return nil, ErrInvalidData
	}

	return sps, nil
}

func isHighProfile(profileIDC int32) bool {
	switch profileIDC {
	case 100, 110, 122, 244, 44, 83, 86, 118, 128, 138, 144:
		return true
	default:
		return false
	}
}

func decodeScalingMatrices(gb *bitReader, chromaFormatIDC uint32, include8x8 bool, mask *uint16) error {
	count := 6
	if include8x8 {
		count = 8
		if chromaFormatIDC == 3 {
			count = 12
		}
	}

	for i := 0; i < count; i++ {
		size := 16
		if i >= 6 {
			size = 64
		}
		if err := decodeScalingList(gb, size, mask, uint(i)); err != nil {
			return err
		}
	}
	return nil
}

func decodeScalingList(gb *bitReader, size int, mask *uint16, pos uint) error {
	present, err := gb.readBit()
	if err != nil {
		return err
	}
	*mask |= uint16(present) << pos
	if present == 0 {
		return nil
	}

	var last int32 = 8
	var next int32 = 8
	for i := 0; i < size; i++ {
		if next != 0 {
			delta, err := gb.readSEGolombLong()
			if err != nil {
				return err
			}
			if delta < -128 || delta > 127 {
				return ErrInvalidData
			}
			next = (last + delta) & 0xff
		}
		if i == 0 && next == 0 {
			break
		}
		if next != 0 {
			last = next
		}
	}
	return nil
}

func decodeCrop(gb *bitReader, sps *SPS) error {
	cropLeft, err := gb.readUEGolombLong()
	if err != nil {
		return err
	}
	cropRight, err := gb.readUEGolombLong()
	if err != nil {
		return err
	}
	cropTop, err := gb.readUEGolombLong()
	if err != nil {
		return err
	}
	cropBottom, err := gb.readUEGolombLong()
	if err != nil {
		return err
	}

	width := int64(16 * sps.MBWidth)
	height := int64(16 * sps.MBHeight)
	vsub := uint32(0)
	if sps.ChromaFormatIDC == 1 {
		vsub = 1
	}
	hsub := uint32(0)
	if sps.ChromaFormatIDC == 1 || sps.ChromaFormatIDC == 2 {
		hsub = 1
	}
	stepX := uint32(1) << hsub
	stepY := uint32(2-uint32(sps.FrameMBSOnlyFlag)) << vsub

	cropWidth := int64(cropLeft+cropRight) * int64(stepX)
	cropHeight := int64(cropTop+cropBottom) * int64(stepY)
	if cropWidth >= width || cropHeight >= height {
		return ErrInvalidData
	}

	sps.CropLeft = cropLeft * stepX
	sps.CropRight = cropRight * stepX
	sps.CropTop = cropTop * stepY
	sps.CropBottom = cropBottom * stepY
	return nil
}
