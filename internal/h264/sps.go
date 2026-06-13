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
	h264ExtendedSAR    = 255
)

type SPS struct {
	SPSID                        uint32
	ProfileIDC                   int32
	LevelIDC                     int32
	ChromaFormatIDC              uint32
	TransformBypass              int32
	Log2MaxFrameNum              int32
	PocType                      uint32
	Log2MaxPocLSB                int32
	DeltaPicOrderAlwaysZeroFlag  int32
	OffsetForNonRefPic           int32
	OffsetForTopToBottomField    int32
	PocCycleLength               uint32
	OffsetForRefFrame            [256]int32
	RefFrameCount                uint32
	GapsInFrameNumAllowedFlag    int32
	MBWidth                      int32
	MBHeight                     int32
	FrameMBSOnlyFlag             int32
	MBAFF                        int32
	Direct8x8InferenceFlag       int32
	Crop                         int32
	CropLeft                     uint32
	CropRight                    uint32
	CropTop                      uint32
	CropBottom                   uint32
	VUIParametersPresentFlag     int32
	BitDepthLuma                 int32
	BitDepthChroma               int32
	ResidualColorTransformFlag   int32
	ConstraintSetFlags           int32
	ScalingMatrixPresent         int32
	ScalingMatrixPresentMask     uint16
	ScalingMatrix4               [6][16]uint8
	ScalingMatrix8               [6][64]uint8
	BitstreamRestrictionFlag     int32
	NumReorderFrames             int32
	MaxDecFrameBuffering         int32
	NALHRDParametersPresentFlag  int32
	VCLHRDParametersPresentFlag  int32
	PicStructPresentFlag         int32
	TimeOffsetLength             int32
	TimingInfoPresentFlag        int32
	NumUnitsInTick               uint32
	TimeScale                    uint32
	FixedFrameRateFlag           int32
	Width                        int32
	Height                       int32
	VUI                          H2645VUI
	CPBCount                     int32
	BitRateScale                 int32
	BitRateValue                 [32]uint32
	CPBSizeValue                 [32]uint32
	CPRFlag                      uint32
	InitialCPBRemovalDelayLength int32
	CPBRemovalDelayLength        int32
	DPBOutputDelayLength         int32
}

type H2645VUI struct {
	SARNum                         int32
	SARDen                         int32
	AspectRatioIDC                 int32
	AspectRatioInfoPresentFlag     int32
	OverscanInfoPresentFlag        int32
	OverscanAppropriateFlag        int32
	VideoSignalTypePresentFlag     int32
	VideoFormat                    int32
	VideoFullRangeFlag             int32
	ColourDescriptionPresentFlag   int32
	ColourPrimaries                int32
	TransferCharacteristics        int32
	MatrixCoeffs                   int32
	ChromaLocInfoPresentFlag       int32
	ChromaSampleLocTypeTopField    int32
	ChromaSampleLocTypeBottomField int32
	ChromaLocation                 int32
}

func DecodeSPS(rbsp []byte) (*SPS, error) {
	return decodeSPS(rbsp, false)
}

func decodeSPS(rbsp []byte, ignoreTruncation bool) (*SPS, error) {
	gb, err := newRBSPBitReader(rbsp)
	if err != nil {
		return nil, err
	}
	sps := &SPS{
		TimeOffsetLength: 24,
		BitDepthLuma:     8,
		BitDepthChroma:   8,
		VUI: H2645VUI{
			SARDen:             1,
			VideoFullRangeFlag: -1,
			MatrixCoeffs:       avColorSpaceUnspecified,
		},
	}
	initFlatScalingMatrices(&sps.ScalingMatrix4, &sps.ScalingMatrix8)

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
			// FFmpeg n8.0.1 request-samples chroma_format_idc > 3 and
			// fails SPS admission.
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
				// FFmpeg n8.0.1 logs separate color planes as unsupported
				// and fails SPS admission.
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
			// FFmpeg n8.0.1 request-samples mixed chroma/luma bit depths
			// and fails SPS admission.
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
			if err := decodeScalingMatrices(&gb, nil, sps, true, true, true, &sps.ScalingMatrixPresentMask, &sps.ScalingMatrix4, &sps.ScalingMatrix8); err != nil {
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
	if mbWidthMinus1 >= h264MaxMBWidth || mbHeightMinus1 >= h264MaxMBHeight {
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
	if vuiPresent != 0 {
		if err := decodeVUIParameters(&gb, sps, ignoreTruncation); err != nil {
			return nil, err
		}
	}
	deriveNumReorderFrames(sps)

	codedWidth := int64(16) * int64(sps.MBWidth)
	codedHeight := int64(16) * int64(sps.MBHeight)
	cropWidth := int64(uint64(sps.CropLeft) + uint64(sps.CropRight))
	cropHeight := int64(uint64(sps.CropTop) + uint64(sps.CropBottom))
	if cropWidth >= codedWidth || cropHeight >= codedHeight {
		return nil, ErrInvalidData
	}
	sps.Width = int32(codedWidth - cropWidth)
	sps.Height = int32(codedHeight - cropHeight)

	return sps, nil
}

// DecodeSPSFromNAL applies FFmpeg's malformed-SPS recovery retries around
// strict RBSP parsing.
func DecodeSPSFromNAL(nal NALUnit) (*SPS, error) {
	if nal.Type != NALSPS {
		return nil, ErrInvalidData
	}
	sps, err := DecodeSPS(nal.RBSP)
	if err == nil {
		return sps, nil
	}
	if len(nal.Raw) <= 1 {
		return nil, err
	}
	// FFmpeg n8.0.1 h264dec.c retries malformed SPS RBSPs with the complete
	// raw NAL payload. Some MOV/MP4 extradata carries unescaped bytes that only
	// parse through this recovery path.
	if rawSPS, rawErr := DecodeSPS(nal.Raw[1:]); rawErr == nil {
		return rawSPS, nil
	}
	if truncatedSPS, truncErr := decodeSPS(nal.RBSP, true); truncErr == nil {
		return truncatedSPS, nil
	}
	return nil, err
}

func isHighProfile(profileIDC int32) bool {
	switch profileIDC {
	case 100, 110, 122, 244, 44, 83, 86, 118, 128, 138, 144:
		return true
	default:
		return false
	}
}

func initFlatScalingMatrices(scaling4 *[6][16]uint8, scaling8 *[6][64]uint8) {
	for i := range scaling4 {
		for j := range scaling4[i] {
			scaling4[i][j] = 16
		}
	}
	for i := range scaling8 {
		for j := range scaling8[i] {
			scaling8[i][j] = 16
		}
	}
}

func decodeScalingMatrices(gb *bitReader, sps *SPS, targetSPS *SPS, isSPS bool, include8x8 bool, presentFlag bool, mask *uint16, scaling4 *[6][16]uint8, scaling8 *[6][64]uint8) error {
	*mask = 0
	if !presentFlag {
		return nil
	}

	chromaFormatIDC := targetSPS.ChromaFormatIDC
	fallbackSPS := !isSPS && sps != nil && sps.ScalingMatrixPresent != 0
	fallback4Intra := h264DefaultScaling4[0][:]
	fallback4Inter := h264DefaultScaling4[1][:]
	fallback8Intra := h264DefaultScaling8[0][:]
	fallback8Inter := h264DefaultScaling8[1][:]
	if fallbackSPS {
		fallback4Intra = sps.ScalingMatrix4[0][:]
		fallback4Inter = sps.ScalingMatrix4[3][:]
		fallback8Intra = sps.ScalingMatrix8[0][:]
		fallback8Inter = sps.ScalingMatrix8[3][:]
	}

	if err := decodeScalingList(gb, scaling4[0][:], h264DefaultScaling4[0][:], fallback4Intra, mask, 0); err != nil {
		return err
	}
	if err := decodeScalingList(gb, scaling4[1][:], h264DefaultScaling4[0][:], scaling4[0][:], mask, 1); err != nil {
		return err
	}
	if err := decodeScalingList(gb, scaling4[2][:], h264DefaultScaling4[0][:], scaling4[1][:], mask, 2); err != nil {
		return err
	}
	if err := decodeScalingList(gb, scaling4[3][:], h264DefaultScaling4[1][:], fallback4Inter, mask, 3); err != nil {
		return err
	}
	if err := decodeScalingList(gb, scaling4[4][:], h264DefaultScaling4[1][:], scaling4[3][:], mask, 4); err != nil {
		return err
	}
	if err := decodeScalingList(gb, scaling4[5][:], h264DefaultScaling4[1][:], scaling4[4][:], mask, 5); err != nil {
		return err
	}

	if isSPS || include8x8 {
		if err := decodeScalingList(gb, scaling8[0][:], h264DefaultScaling8[0][:], fallback8Intra, mask, 6); err != nil {
			return err
		}
		if err := decodeScalingList(gb, scaling8[3][:], h264DefaultScaling8[1][:], fallback8Inter, mask, 7); err != nil {
			return err
		}
		if chromaFormatIDC == 3 {
			if err := decodeScalingList(gb, scaling8[1][:], h264DefaultScaling8[0][:], scaling8[0][:], mask, 8); err != nil {
				return err
			}
			if err := decodeScalingList(gb, scaling8[4][:], h264DefaultScaling8[1][:], scaling8[3][:], mask, 9); err != nil {
				return err
			}
			if err := decodeScalingList(gb, scaling8[2][:], h264DefaultScaling8[0][:], scaling8[1][:], mask, 10); err != nil {
				return err
			}
			if err := decodeScalingList(gb, scaling8[5][:], h264DefaultScaling8[1][:], scaling8[4][:], mask, 11); err != nil {
				return err
			}
		}
	}
	return nil
}

func decodeScalingList(gb *bitReader, factors []uint8, jvtList []uint8, fallbackList []uint8, mask *uint16, pos uint) error {
	present, err := gb.readBit()
	if err != nil {
		return err
	}
	*mask |= uint16(present) << pos
	if present == 0 {
		copy(factors, fallbackList)
		return nil
	}

	var last int32 = 8
	var next int32 = 8
	scan := h264ZigzagDirect[:]
	if len(factors) == 16 {
		scan = h264ZigzagScan[:]
	}
	for i := 0; i < len(factors); i++ {
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
			copy(factors, jvtList)
			break
		}
		if next != 0 {
			last = next
		}
		factors[scan[i]] = uint8(last)
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

	width := uint64(16) * uint64(sps.MBWidth)
	height := uint64(16) * uint64(sps.MBHeight)
	vsub := uint32(0)
	if sps.ChromaFormatIDC == 1 {
		vsub = 1
	}
	hsub := uint32(0)
	if sps.ChromaFormatIDC == 1 || sps.ChromaFormatIDC == 2 {
		hsub = 1
	}
	stepX := uint64(1) << hsub
	frameMBSFactor := uint64(2)
	if sps.FrameMBSOnlyFlag != 0 {
		frameMBSFactor = 1
	}
	stepY := frameMBSFactor << vsub

	cropLeftScaled := uint64(cropLeft) * stepX
	cropRightScaled := uint64(cropRight) * stepX
	cropTopScaled := uint64(cropTop) * stepY
	cropBottomScaled := uint64(cropBottom) * stepY
	cropWidth := cropLeftScaled + cropRightScaled
	cropHeight := cropTopScaled + cropBottomScaled
	if cropWidth >= width || cropHeight >= height {
		return ErrInvalidData
	}

	sps.CropLeft = uint32(cropLeftScaled)
	sps.CropRight = uint32(cropRightScaled)
	sps.CropTop = uint32(cropTopScaled)
	sps.CropBottom = uint32(cropBottomScaled)
	return nil
}

// decodeVUIParameters is a source-shaped port of FFmpeg n8.0.1
// libavcodec/h264_ps.c decode_vui_parameters.
func decodeVUIParameters(gb *bitReader, sps *SPS, ignoreTruncation bool) error {
	if gb == nil || sps == nil {
		return ErrInvalidData
	}
	if err := decodeCommonVUIParameters(gb, &sps.VUI); err != nil {
		if ignoreTruncation && gb.bitsLeft() <= 0 {
			return nil
		}
		return err
	}

	if gb.bitsLeft() <= 0 {
		return nil
	}
	show, err := gb.showBits(1)
	if err != nil {
		return nil
	}
	if show != 0 && gb.bitsLeft() < 10 {
		return nil
	}

	timingInfoPresent, err := gb.readBit()
	if err != nil {
		if ignoreTruncation && gb.bitsLeft() <= 0 {
			return nil
		}
		return err
	}
	sps.TimingInfoPresentFlag = int32(timingInfoPresent)
	if timingInfoPresent != 0 {
		numUnitsInTick, err := gb.readBits(32)
		if err != nil {
			if ignoreTruncation && gb.bitsLeft() <= 0 {
				sps.TimingInfoPresentFlag = 0
				return nil
			}
			return err
		}
		timeScale, err := gb.readBits(32)
		if err != nil {
			if ignoreTruncation && gb.bitsLeft() <= 0 {
				sps.TimingInfoPresentFlag = 0
				return nil
			}
			return err
		}
		if numUnitsInTick == 0 || timeScale == 0 {
			sps.TimingInfoPresentFlag = 0
		} else {
			sps.NumUnitsInTick = numUnitsInTick
			sps.TimeScale = timeScale
		}
		fixedFrameRateFlag, err := gb.readBit()
		if err != nil {
			if ignoreTruncation && gb.bitsLeft() <= 0 {
				return nil
			}
			return err
		}
		sps.FixedFrameRateFlag = int32(fixedFrameRateFlag)
	}

	nalHRDPresent, err := gb.readBit()
	if err != nil {
		if ignoreTruncation && gb.bitsLeft() <= 0 {
			return nil
		}
		return err
	}
	sps.NALHRDParametersPresentFlag = int32(nalHRDPresent)
	if nalHRDPresent != 0 {
		if err := decodeHRDParameters(gb, sps); err != nil {
			return err
		}
	}
	vclHRDPresent, err := gb.readBit()
	if err != nil {
		if ignoreTruncation && gb.bitsLeft() <= 0 {
			return nil
		}
		return err
	}
	sps.VCLHRDParametersPresentFlag = int32(vclHRDPresent)
	if vclHRDPresent != 0 {
		if err := decodeHRDParameters(gb, sps); err != nil {
			return err
		}
	}
	if nalHRDPresent != 0 || vclHRDPresent != 0 {
		if err := gb.skipBits(1); err != nil {
			if ignoreTruncation && gb.bitsLeft() <= 0 {
				return nil
			}
			return err
		}
	}
	picStructPresent, err := gb.readBit()
	if err != nil {
		if ignoreTruncation && gb.bitsLeft() <= 0 {
			return nil
		}
		return err
	}
	sps.PicStructPresentFlag = int32(picStructPresent)
	if gb.bitsLeft() <= 0 {
		return nil
	}

	bitstreamRestriction, err := gb.readBit()
	if err != nil {
		if ignoreTruncation && gb.bitsLeft() <= 0 {
			return nil
		}
		return err
	}
	sps.BitstreamRestrictionFlag = int32(bitstreamRestriction)
	if bitstreamRestriction == 0 {
		return nil
	}
	if err := gb.skipBits(1); err != nil {
		if ignoreTruncation && gb.bitsLeft() <= 0 {
			sps.NumReorderFrames = 0
			sps.BitstreamRestrictionFlag = 0
			return nil
		}
		return err
	}
	if _, err := gb.readUEGolomb31(); err != nil {
		if ignoreTruncation && gb.bitsLeft() <= 0 {
			sps.NumReorderFrames = 0
			sps.BitstreamRestrictionFlag = 0
			return nil
		}
		return err
	}
	if _, err := gb.readUEGolomb31(); err != nil {
		if ignoreTruncation && gb.bitsLeft() <= 0 {
			sps.NumReorderFrames = 0
			sps.BitstreamRestrictionFlag = 0
			return nil
		}
		return err
	}
	if _, err := gb.readUEGolomb31(); err != nil {
		if ignoreTruncation && gb.bitsLeft() <= 0 {
			sps.NumReorderFrames = 0
			sps.BitstreamRestrictionFlag = 0
			return nil
		}
		return err
	}
	if _, err := gb.readUEGolomb31(); err != nil {
		if ignoreTruncation && gb.bitsLeft() <= 0 {
			sps.NumReorderFrames = 0
			sps.BitstreamRestrictionFlag = 0
			return nil
		}
		return err
	}
	numReorderFrames, err := gb.readUEGolomb31()
	if err != nil {
		sps.NumReorderFrames = 0
		sps.BitstreamRestrictionFlag = 0
		if ignoreTruncation && gb.bitsLeft() <= 0 {
			return nil
		}
		return err
	}
	maxDecFrameBuffering, err := gb.readUEGolomb31()
	if err != nil {
		sps.NumReorderFrames = 0
		sps.BitstreamRestrictionFlag = 0
		if ignoreTruncation && gb.bitsLeft() <= 0 {
			return nil
		}
		return err
	}
	sps.NumReorderFrames = int32(numReorderFrames)
	sps.MaxDecFrameBuffering = int32(maxDecFrameBuffering)
	if numReorderFrames > h264MaxDPBFrames {
		sps.NumReorderFrames = h264MaxDPBFrames
		return ErrInvalidData
	}
	return nil
}

// decodeHRDParameters is a source-shaped port of FFmpeg n8.0.1
// libavcodec/h264_ps.c decode_hrd_parameters.
func decodeHRDParameters(gb *bitReader, sps *SPS) error {
	cpbCountMinus1, err := gb.readUEGolomb31()
	if err != nil {
		return err
	}
	cpbCount := cpbCountMinus1 + 1
	if cpbCount > 32 {
		return ErrInvalidData
	}

	sps.CPRFlag = 0
	bitRateScale, err := gb.readBits(4)
	if err != nil {
		return err
	}
	sps.BitRateScale = int32(bitRateScale)
	if err := gb.skipBits(4); err != nil {
		return err
	}
	for i := uint32(0); i < cpbCount; i++ {
		bitRateValueMinus1, err := gb.readUEGolombLong()
		if err != nil {
			return err
		}
		cpbSizeValueMinus1, err := gb.readUEGolombLong()
		if err != nil {
			return err
		}
		cbrFlag, err := gb.readBit()
		if err != nil {
			return err
		}
		sps.BitRateValue[i] = bitRateValueMinus1 + 1
		sps.CPBSizeValue[i] = cpbSizeValueMinus1 + 1
		sps.CPRFlag |= cbrFlag << i
	}
	initialDelayLen, err := gb.readBits(5)
	if err != nil {
		return err
	}
	cpbRemovalLen, err := gb.readBits(5)
	if err != nil {
		return err
	}
	dpbOutputLen, err := gb.readBits(5)
	if err != nil {
		return err
	}
	timeOffsetLen, err := gb.readBits(5)
	if err != nil {
		return err
	}
	sps.InitialCPBRemovalDelayLength = int32(initialDelayLen) + 1
	sps.CPBRemovalDelayLength = int32(cpbRemovalLen) + 1
	sps.DPBOutputDelayLength = int32(dpbOutputLen) + 1
	sps.TimeOffsetLength = int32(timeOffsetLen)
	sps.CPBCount = int32(cpbCount)
	return nil
}

// decodeCommonVUIParameters mirrors FFmpeg n8.0.1
// libavcodec/h2645_vui.c ff_h2645_decode_common_vui_params.
func decodeCommonVUIParameters(gb *bitReader, vui *H2645VUI) error {
	aspectRatioPresent, err := gb.readBit()
	if err != nil {
		return err
	}
	vui.AspectRatioInfoPresentFlag = int32(aspectRatioPresent)
	if aspectRatioPresent != 0 {
		aspectRatioIDC, err := gb.readBits(8)
		if err != nil {
			return err
		}
		vui.AspectRatioIDC = int32(aspectRatioIDC)
		if aspectRatioIDC < uint32(len(h2645PixelAspect)) {
			vui.SARNum = h2645PixelAspect[aspectRatioIDC][0]
			vui.SARDen = h2645PixelAspect[aspectRatioIDC][1]
		} else if aspectRatioIDC == h264ExtendedSAR {
			sarNum, err := gb.readBits(16)
			if err != nil {
				return err
			}
			sarDen, err := gb.readBits(16)
			if err != nil {
				return err
			}
			vui.SARNum = int32(sarNum)
			vui.SARDen = int32(sarDen)
		}
	} else {
		vui.SARNum = 0
		vui.SARDen = 1
	}

	overscanPresent, err := gb.readBit()
	if err != nil {
		return err
	}
	vui.OverscanInfoPresentFlag = int32(overscanPresent)
	if overscanPresent != 0 {
		overscanAppropriate, err := gb.readBit()
		if err != nil {
			return err
		}
		vui.OverscanAppropriateFlag = int32(overscanAppropriate)
	}

	videoSignalPresent, err := gb.readBit()
	if err != nil {
		return err
	}
	vui.VideoSignalTypePresentFlag = int32(videoSignalPresent)
	if videoSignalPresent != 0 {
		videoFormat, err := gb.readBits(3)
		if err != nil {
			return err
		}
		fullRange, err := gb.readBit()
		if err != nil {
			return err
		}
		colourDescriptionPresent, err := gb.readBit()
		if err != nil {
			return err
		}
		vui.VideoFormat = int32(videoFormat)
		vui.VideoFullRangeFlag = int32(fullRange)
		vui.ColourDescriptionPresentFlag = int32(colourDescriptionPresent)
		if colourDescriptionPresent != 0 {
			primaries, err := gb.readBits(8)
			if err != nil {
				return err
			}
			transfer, err := gb.readBits(8)
			if err != nil {
				return err
			}
			matrix, err := gb.readBits(8)
			if err != nil {
				return err
			}
			vui.ColourPrimaries = normalizeColorPrimaries(int32(primaries))
			vui.TransferCharacteristics = normalizeColorTransfer(int32(transfer))
			vui.MatrixCoeffs = normalizeColorSpace(int32(matrix))
		}
	}

	chromaLocPresent, err := gb.readBit()
	if err != nil {
		return err
	}
	vui.ChromaLocInfoPresentFlag = int32(chromaLocPresent)
	if chromaLocPresent != 0 {
		top, err := gb.readUEGolomb31()
		if err != nil {
			return err
		}
		bottom, err := gb.readUEGolomb31()
		if err != nil {
			return err
		}
		vui.ChromaSampleLocTypeTopField = int32(top)
		vui.ChromaSampleLocTypeBottomField = int32(bottom)
		if top <= 5 {
			vui.ChromaLocation = int32(top) + 1
		} else {
			vui.ChromaLocation = avChromaLocUnspecified
		}
	} else {
		vui.ChromaLocation = avChromaLocLeft
	}
	return nil
}

func deriveNumReorderFrames(sps *SPS) {
	if sps == nil || sps.BitstreamRestrictionFlag != 0 || sps.RefFrameCount == 0 {
		return
	}
	sps.NumReorderFrames = h264MaxDPBFrames - 1
	for _, entry := range levelMaxDPBMbs {
		if entry[0] != sps.LevelIDC {
			continue
		}
		maxByLevel := entry[1] / (sps.MBWidth * sps.MBHeight)
		if maxByLevel < sps.NumReorderFrames {
			sps.NumReorderFrames = maxByLevel
		}
		return
	}
}

func normalizeColorPrimaries(v int32) int32 {
	if v == 0 || v == 3 || !validColorPrimaries(v) {
		return avColorPrimariesUnspecified
	}
	return v
}

func normalizeColorTransfer(v int32) int32 {
	if v == 0 || v == 3 || !validColorTransfer(v) {
		return avColorTransferUnspecified
	}
	return v
}

func normalizeColorSpace(v int32) int32 {
	if v == 3 || !validColorSpace(v) {
		return avColorSpaceUnspecified
	}
	return v
}

func validColorPrimaries(v int32) bool {
	switch v {
	case 1, 2, 4, 5, 6, 7, 8, 9, 10, 11, 12, 22:
		return true
	default:
		return false
	}
}

func validColorTransfer(v int32) bool {
	switch v {
	case 1, 2, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18:
		return true
	default:
		return false
	}
}

func validColorSpace(v int32) bool {
	switch v {
	case 0, 1, 2, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17:
		return true
	default:
		return false
	}
}

var h2645PixelAspect = [17][2]int32{
	{0, 1},
	{1, 1},
	{12, 11},
	{10, 11},
	{16, 11},
	{40, 33},
	{24, 11},
	{20, 11},
	{32, 11},
	{80, 33},
	{18, 11},
	{15, 11},
	{64, 33},
	{160, 99},
	{4, 3},
	{3, 2},
	{2, 1},
}

var levelMaxDPBMbs = [][2]int32{
	{10, 396},
	{11, 900},
	{12, 2376},
	{13, 2376},
	{20, 2376},
	{21, 4752},
	{22, 8100},
	{30, 8100},
	{31, 18000},
	{32, 20480},
	{40, 32768},
	{41, 32768},
	{42, 34816},
	{50, 110400},
	{51, 184320},
	{52, 184320},
}

const (
	avColorPrimariesUnspecified = 2
	avColorTransferUnspecified  = 2
	avColorSpaceUnspecified     = 2
	avChromaLocUnspecified      = 0
	avChromaLocLeft             = 1
)
