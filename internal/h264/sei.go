// SPDX-License-Identifier: LGPL-2.1-or-later
//
// Source-shaped H.264 SEI parser subset from FFmpeg n8.0.1
// libavcodec/h264_sei.c, h2645_sei.c, and sei.h.

package h264

const (
	seiTypeBufferingPeriod              = 0
	seiTypePicTiming                    = 1
	seiTypeUserDataRegisteredITUTT35    = 4
	seiTypeUserDataUnregistered         = 5
	seiTypeRecoveryPoint                = 6
	seiTypeGreenMetadata                = 56
	seiTypeFilmGrainCharacteristics     = 19
	seiTypeFramePackingArrangement      = 45
	seiTypeDisplayOrientation           = 47
	seiTypeMasteringDisplayColourVolume = 137
	seiTypeContentLightLevelInfo        = 144
	seiTypeAlternativeTransfer          = 147
	seiTypeAmbientViewingEnvironment    = 148

	h264SEIPicStructFrame           = 0
	h264SEIPicStructTopField        = 1
	h264SEIPicStructBottomField     = 2
	h264SEIPicStructTopBottom       = 3
	h264SEIPicStructBottomTop       = 4
	h264SEIPicStructTopBottomTop    = 5
	h264SEIPicStructBottomTopBottom = 6
	h264SEIPicStructFrameDoubling   = 7
	h264SEIPicStructFrameTripling   = 8

	ituTT35CountryCodeUS       = 0xb5
	ituTT35CountryCodeUK       = 0xb4
	ituTT35ProviderCodeATSC    = 0x0031
	ituTT35ProviderCodeVNOVA   = 0x0050
	ituTT35UserIdentifierDTG1  = 0x44544731
	ituTT35UserIdentifierGA94  = 0x47413934
	a53UserDataTypeCodeCaption = 0x03
)

var seiNumClockTSTable = [9]uint8{1, 1, 1, 2, 2, 3, 3, 2, 3}

type H264SEIContext struct {
	Common          H2645SEI
	PictureTiming   H264SEIPictureTiming
	RecoveryPoint   H264SEIRecoveryPoint
	BufferingPeriod H264SEIBufferingPeriod
	GreenMetadata   H264SEIGreenMetadata
}

type H264SEIPictureTiming struct {
	Payload         [40]uint8
	PayloadSize     int
	Present         int32
	PicStruct       int32
	CTType          int32
	DPBOutputDelay  int32
	CPBRemovalDelay int32
	Timecode        [3]H264SEITimeCode
	TimecodeCount   int32
}

type H264SEITimeCode struct {
	Full      int32
	Frame     int32
	Seconds   int32
	Minutes   int32
	Hours     int32
	DropFrame int32
}

type H264SEIRecoveryPoint struct {
	RecoveryFrameCount int32
}

type H264SEIBufferingPeriod struct {
	Present                int32
	InitialCPBRemovalDelay [32]int32
}

type H264SEIGreenMetadata struct {
	Present                             int32
	GreenMetadataType                   uint8
	PeriodType                          uint8
	NumSeconds                          uint16
	NumPictures                         uint16
	PercentNonZeroMacroblocks           uint8
	PercentIntraCodedMacroblocks        uint8
	PercentSixTapFiltering              uint8
	PercentAlphaPointDeblockingInstance uint8
	XSDMetricType                       uint8
	XSDMetricValue                      uint16
}

type H2645SEI struct {
	A53Caption          H2645SEIA53Caption
	AFD                 H2645SEIAFD
	Unregistered        H2645SEIUnregistered
	FramePacking        H2645SEIFramePacking
	DisplayOrientation  H2645SEIDisplayOrientation
	AlternativeTransfer H2645SEIAlternativeTransfer
	AmbientViewing      H2645SEIAmbientViewingEnvironment
	MasteringDisplay    H2645SEIMasteringDisplay
	ContentLight        H2645SEIContentLight
	FilmGrain           H2645SEIFilmGrainCharacteristics
	LCEVC               H2645SEILCEVC
}

type H2645SEIA53Caption struct {
	Data []uint8
}

type H2645SEIAFD struct {
	Present                 int32
	ActiveFormatDescription uint8
}

type H2645SEIUnregistered struct {
	Data      [][]uint8
	X264Build int32
}

type H2645SEIFramePacking struct {
	Present                     int32
	ArrangementID               uint32
	ArrangementCancelFlag       int32
	ArrangementType             int32
	ArrangementRepetitionPeriod uint32
	ContentInterpretationType   int32
	QuincunxSamplingFlag        int32
	CurrentFrameIsFrame0Flag    int32
}

type H2645SEIDisplayOrientation struct {
	Present               int32
	AnticlockwiseRotation int32
	HFlip                 int32
	VFlip                 int32
}

type AVDisplayMatrix struct {
	Present int32
	Matrix  [9]int32
}

type AVRational struct {
	Num int32
	Den int32
}

type AVStereo3D struct {
	Present                       int32
	Type                          int32
	Flags                         int32
	View                          int32
	PrimaryEye                    int32
	Baseline                      uint32
	HorizontalDisparityAdjustment AVRational
	HorizontalFieldOfView         AVRational
}

type AVSphericalMapping struct {
	Present     int32
	Projection  int32
	Yaw         int32
	Pitch       int32
	Roll        int32
	BoundLeft   uint32
	BoundTop    uint32
	BoundRight  uint32
	BoundBottom uint32
	Padding     uint32
}

type AV3DReferenceDisplaysInfo struct {
	Present                int32
	PrecRefDisplayWidth    uint8
	RefViewingDistanceFlag uint8
	PrecRefViewingDist     uint8
	Displays               []AV3DReferenceDisplay
}

type AV3DReferenceDisplay struct {
	LeftViewID                 uint16
	RightViewID                uint16
	ExponentRefDisplayWidth    uint8
	MantissaRefDisplayWidth    uint8
	ExponentRefViewingDistance uint8
	MantissaRefViewingDistance uint8
	AdditionalShiftPresentFlag uint8
	NumSampleShift             int16
}

type H2645SEIAlternativeTransfer struct {
	Present                          int32
	PreferredTransferCharacteristics int32
}

type H2645SEIAmbientViewingEnvironment struct {
	Present            int32
	AmbientIlluminance uint32
	AmbientLightX      uint16
	AmbientLightY      uint16
}

type H2645SEIFilmGrainCharacteristics struct {
	Present                              int32
	ModelID                              int32
	SeparateColourDescriptionPresentFlag int32
	BitDepthLuma                         int32
	BitDepthChroma                       int32
	FullRange                            int32
	ColorPrimaries                       int32
	TransferCharacteristics              int32
	MatrixCoeffs                         int32
	BlendingModeID                       int32
	Log2ScaleFactor                      int32
	CompModelPresentFlag                 [3]int32
	NumIntensityIntervals                [3]uint16
	NumModelValues                       [3]uint8
	IntensityIntervalLowerBound          [3][256]uint8
	IntensityIntervalUpperBound          [3][256]uint8
	CompModelValue                       [3][256][6]int16
	RepetitionPeriod                     uint32
}

type H2645SEIMasteringDisplay struct {
	Present          int32
	DisplayPrimaries [3][2]uint16
	WhitePoint       [2]uint16
	MaxLuminance     uint32
	MinLuminance     uint32
}

type AVMasteringDisplayMetadata struct {
	Present          int32
	DisplayPrimaries [3][2]uint16
	WhitePoint       [2]uint16
	MaxLuminance     uint32
	MinLuminance     uint32
	HasPrimaries     int32
	HasLuminance     int32
}

type H2645SEIContentLight struct {
	Present                 int32
	MaxContentLightLevel    uint32
	MaxPicAverageLightLevel uint32
}

type H2645SEILCEVC struct {
	Data []uint8
}

func (h *H264SEIContext) Reset() {
	if h == nil {
		return
	}
	h.Common.A53Caption.Data = nil
	h.Common.AFD.Present = 0
	h.Common.Unregistered.Data = nil
	h.Common.Unregistered.X264Build = -1
	h.RecoveryPoint.RecoveryFrameCount = -1
	h.PictureTiming.DPBOutputDelay = 0
	h.PictureTiming.CPBRemovalDelay = -1
	h.PictureTiming.Present = 0
	h.PictureTiming.TimecodeCount = 0
	h.BufferingPeriod.Present = 0
	h.GreenMetadata.Present = 0
	h.Common.FramePacking.Present = 0
	h.Common.DisplayOrientation.Present = 0
	h.Common.MasteringDisplay.Present = 0
	h.Common.ContentLight.Present = 0
	h.Common.AmbientViewing = H2645SEIAmbientViewingEnvironment{}
	h.Common.FilmGrain = H2645SEIFilmGrainCharacteristics{}
	h.Common.LCEVC.Data = nil
}

func DecodeSEI(rbsp []byte, spsList *[maxSPSCount]*SPS) (*H264SEIContext, error) {
	var ctx H264SEIContext
	ctx.Reset()
	err := ctx.Decode(rbsp, spsList)
	return &ctx, err
}

// Decode mirrors FFmpeg ff_h264_sei_decode's message framing and return policy
// for the translated subset.
func (h *H264SEIContext) Decode(rbsp []byte, spsList *[maxSPSCount]*SPS) error {
	if h == nil {
		return ErrInvalidData
	}
	masterErr := error(nil)
	pos := 0
	for len(rbsp)-pos > 2 && peekBE16(rbsp[pos:]) != 0 {
		payloadType := 0
		for {
			if pos >= len(rbsp) {
				return ErrInvalidData
			}
			b := rbsp[pos]
			pos++
			payloadType += int(b)
			if b != 0xff {
				break
			}
		}
		payloadSize := 0
		for {
			if pos >= len(rbsp) {
				return ErrInvalidData
			}
			b := rbsp[pos]
			pos++
			payloadSize += int(b)
			if b != 0xff {
				break
			}
		}
		if payloadSize < 0 || payloadSize > len(rbsp)-pos {
			return ErrInvalidData
		}
		payload := rbsp[pos : pos+payloadSize]
		if err := h.decodeMessage(payloadType, payload, spsList); err != nil {
			if err != errParamSetNotFound {
				return err
			}
			masterErr = err
		}
		pos += payloadSize
	}
	return masterErr
}

func (h *H264SEIContext) decodeMessage(payloadType int, payload []byte, spsList *[maxSPSCount]*SPS) error {
	switch payloadType {
	case seiTypePicTiming:
		return h.PictureTiming.decodePictureTiming(payload)
	case seiTypeRecoveryPoint:
		return h.RecoveryPoint.decodeRecoveryPoint(payload)
	case seiTypeBufferingPeriod:
		return h.BufferingPeriod.decodeBufferingPeriod(payload, spsList)
	case seiTypeGreenMetadata:
		return h.GreenMetadata.decodeGreenMetadata(payload)
	case seiTypeUserDataRegisteredITUTT35:
		return h.Common.decodeRegisteredUserData(payload)
	case seiTypeUserDataUnregistered:
		return h.Common.Unregistered.decodeUnregisteredUserData(payload)
	case seiTypeDisplayOrientation:
		return h.Common.DisplayOrientation.decodeDisplayOrientation(payload)
	case seiTypeFilmGrainCharacteristics:
		return h.Common.FilmGrain.decodeFilmGrainCharacteristics(payload)
	case seiTypeFramePackingArrangement:
		return h.Common.FramePacking.decodeFramePackingArrangement(payload)
	case seiTypeAlternativeTransfer:
		return h.Common.AlternativeTransfer.decodeAlternativeTransfer(payload)
	case seiTypeAmbientViewingEnvironment:
		return h.Common.AmbientViewing.decodeAmbientViewingEnvironment(payload)
	case seiTypeMasteringDisplayColourVolume:
		return h.Common.MasteringDisplay.decodeMasteringDisplay(payload)
	case seiTypeContentLightLevelInfo:
		return h.Common.ContentLight.decodeContentLight(payload)
	default:
		return nil
	}
}

func (h *H264SEIPictureTiming) decodePictureTiming(payload []byte) error {
	if len(payload) > len(h.Payload) {
		return ErrInvalidData
	}
	copy(h.Payload[:], payload)
	for i := len(payload); i < len(h.Payload); i++ {
		h.Payload[i] = 0
	}
	h.PayloadSize = len(payload)
	h.Present = 1
	return nil
}

func (h *H264SEIPictureTiming) Process(sps *SPS) error {
	if h == nil || sps == nil {
		return ErrInvalidData
	}
	gb := newBitReader(h.Payload[:h.PayloadSize])
	if sps.NALHRDParametersPresentFlag != 0 || sps.VCLHRDParametersPresentFlag != 0 {
		cpbRemovalDelay, err := gb.readBits(uint32(sps.CPBRemovalDelayLength))
		if err != nil {
			return err
		}
		dpbOutputDelay, err := gb.readBits(uint32(sps.DPBOutputDelayLength))
		if err != nil {
			return err
		}
		h.CPBRemovalDelay = int32(cpbRemovalDelay)
		h.DPBOutputDelay = int32(dpbOutputDelay)
	}
	if sps.PicStructPresentFlag != 0 {
		picStruct, err := gb.readBits(4)
		if err != nil {
			return err
		}
		if picStruct > h264SEIPicStructFrameTripling {
			return ErrInvalidData
		}
		h.PicStruct = int32(picStruct)
		h.CTType = 0
		h.TimecodeCount = 0
		numClockTS := seiNumClockTSTable[picStruct]
		for i := uint8(0); i < numClockTS; i++ {
			clockTimestampFlag, err := gb.readBit()
			if err != nil {
				return err
			}
			if clockTimestampFlag == 0 {
				continue
			}
			if h.TimecodeCount >= int32(len(h.Timecode)) {
				return ErrInvalidData
			}
			tc := &h.Timecode[h.TimecodeCount]
			h.TimecodeCount++
			ctType, err := gb.readBits(2)
			if err != nil {
				return err
			}
			h.CTType |= 1 << ctType
			if err := gb.skipBits(1); err != nil {
				return err
			}
			countingType, err := gb.readBits(5)
			if err != nil {
				return err
			}
			fullTimestampFlag, err := gb.readBit()
			if err != nil {
				return err
			}
			if err := gb.skipBits(1); err != nil {
				return err
			}
			cntDroppedFlag, err := gb.readBit()
			if err != nil {
				return err
			}
			if cntDroppedFlag != 0 && countingType > 1 && countingType < 7 {
				tc.DropFrame = 1
			}
			frame, err := gb.readBits(8)
			if err != nil {
				return err
			}
			tc.Frame = int32(frame)
			if fullTimestampFlag != 0 {
				tc.Full = 1
				seconds, err := gb.readBits(6)
				if err != nil {
					return err
				}
				minutes, err := gb.readBits(6)
				if err != nil {
					return err
				}
				hours, err := gb.readBits(5)
				if err != nil {
					return err
				}
				tc.Seconds = int32(seconds)
				tc.Minutes = int32(minutes)
				tc.Hours = int32(hours)
			} else {
				tc.Full = 0
				tc.Seconds = 0
				tc.Minutes = 0
				tc.Hours = 0
				secondsFlag, err := gb.readBit()
				if err != nil {
					return err
				}
				if secondsFlag != 0 {
					seconds, err := gb.readBits(6)
					if err != nil {
						return err
					}
					tc.Seconds = int32(seconds)
					minutesFlag, err := gb.readBit()
					if err != nil {
						return err
					}
					if minutesFlag != 0 {
						minutes, err := gb.readBits(6)
						if err != nil {
							return err
						}
						tc.Minutes = int32(minutes)
						hoursFlag, err := gb.readBit()
						if err != nil {
							return err
						}
						if hoursFlag != 0 {
							hours, err := gb.readBits(5)
							if err != nil {
								return err
							}
							tc.Hours = int32(hours)
						}
					}
				}
			}
			if sps.TimeOffsetLength > 0 {
				if err := gb.skipBits(uint32(sps.TimeOffsetLength)); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (h *H264SEIRecoveryPoint) decodeRecoveryPoint(payload []byte) error {
	gb := newBitReader(payload)
	recoveryFrameCount, err := gb.readUEGolombLong()
	if err != nil {
		return err
	}
	if recoveryFrameCount >= 1<<maxLog2MaxFrameNum {
		return ErrInvalidData
	}
	h.RecoveryFrameCount = int32(recoveryFrameCount)
	return gb.skipBits(4)
}

func (h *H264SEIBufferingPeriod) decodeBufferingPeriod(payload []byte, spsList *[maxSPSCount]*SPS) error {
	if spsList == nil {
		return ErrInvalidData
	}
	gb := newBitReader(payload)
	spsID, err := gb.readUEGolomb31()
	if err != nil {
		return err
	}
	if spsID >= maxSPSCount {
		return ErrInvalidData
	}
	sps := spsList[spsID]
	if sps == nil {
		return errParamSetNotFound
	}
	if sps.NALHRDParametersPresentFlag != 0 {
		for i := int32(0); i < sps.CPBCount; i++ {
			delay, err := gb.readBits(uint32(sps.InitialCPBRemovalDelayLength))
			if err != nil {
				return err
			}
			h.InitialCPBRemovalDelay[i] = int32(delay)
			if err := gb.skipBits(uint32(sps.InitialCPBRemovalDelayLength)); err != nil {
				return err
			}
		}
	}
	if sps.VCLHRDParametersPresentFlag != 0 {
		for i := int32(0); i < sps.CPBCount; i++ {
			delay, err := gb.readBits(uint32(sps.InitialCPBRemovalDelayLength))
			if err != nil {
				return err
			}
			h.InitialCPBRemovalDelay[i] = int32(delay)
			if err := gb.skipBits(uint32(sps.InitialCPBRemovalDelayLength)); err != nil {
				return err
			}
		}
	}
	h.Present = 1
	return nil
}

func (h *H264SEIGreenMetadata) decodeGreenMetadata(payload []byte) error {
	if len(payload) < 1 {
		return ErrInvalidData
	}
	h.Present = 1
	h.GreenMetadataType = payload[0]
	pos := 1
	if h.GreenMetadataType == 0 {
		if len(payload)-pos < 5 {
			return ErrInvalidData
		}
		h.PeriodType = payload[pos]
		pos++
		if h.PeriodType == 2 {
			if len(payload)-pos < 2 {
				return ErrInvalidData
			}
			h.NumSeconds = readBE16(payload[pos:])
			pos += 2
		} else if h.PeriodType == 3 {
			if len(payload)-pos < 2 {
				return ErrInvalidData
			}
			h.NumPictures = readBE16(payload[pos:])
			pos += 2
		}
		if len(payload)-pos < 4 {
			return ErrInvalidData
		}
		h.PercentNonZeroMacroblocks = payload[pos]
		h.PercentIntraCodedMacroblocks = payload[pos+1]
		h.PercentSixTapFiltering = payload[pos+2]
		h.PercentAlphaPointDeblockingInstance = payload[pos+3]
	} else if h.GreenMetadataType == 1 {
		if len(payload)-pos < 3 {
			return ErrInvalidData
		}
		h.XSDMetricType = payload[pos]
		h.XSDMetricValue = readBE16(payload[pos+1:])
	}
	return nil
}

func (h *H2645SEI) decodeRegisteredUserData(payload []byte) error {
	if len(payload) < 3 {
		return ErrInvalidData
	}
	pos := 0
	countryCode := payload[pos]
	pos++
	if countryCode == 0xff {
		if len(payload)-pos < 3 {
			return ErrInvalidData
		}
		pos++
	}
	providerCode := readBE16(payload[pos:])
	pos += 2

	if countryCode == ituTT35CountryCodeUS && providerCode == ituTT35ProviderCodeATSC {
		if len(payload)-pos < 4 {
			return ErrInvalidData
		}
		userIdentifier := readBE32(payload[pos:])
		pos += 4
		switch userIdentifier {
		case ituTT35UserIdentifierDTG1:
			return h.AFD.decodeRegisteredUserDataAFD(payload[pos:])
		case ituTT35UserIdentifierGA94:
			return h.A53Caption.decodeRegisteredUserDataClosedCaption(payload[pos:])
		}
	} else if countryCode == ituTT35CountryCodeUK && providerCode == ituTT35ProviderCodeVNOVA {
		if len(payload)-pos < 2 {
			return ErrInvalidData
		}
		pos++
		return h.LCEVC.decodeRegisteredUserDataLCEVC(payload[pos:])
	}
	return nil
}

func (h *H2645SEILCEVC) decodeRegisteredUserDataLCEVC(payload []byte) error {
	h.Data = append(h.Data[:0], payload...)
	return nil
}

func (h *H2645SEIAFD) decodeRegisteredUserDataAFD(payload []byte) error {
	if len(payload) <= 0 {
		return ErrInvalidData
	}
	flag := payload[0]&0x40 != 0
	if flag {
		if len(payload) <= 1 {
			return ErrInvalidData
		}
		h.ActiveFormatDescription = payload[1] & 0x0f
		h.Present = 1
	}
	return nil
}

func (h *H2645SEIA53Caption) decodeRegisteredUserDataClosedCaption(payload []byte) error {
	if len(payload) < 3 {
		return ErrInvalidData
	}
	if payload[0] != a53UserDataTypeCodeCaption {
		return nil
	}
	if payload[1]&0x40 == 0 {
		return nil
	}
	ccCount := int(payload[1] & 0x1f)
	if ccCount == 0 {
		return nil
	}
	ccBytes := ccCount * 3
	if ccBytes >= len(payload)-3 {
		return ErrInvalidData
	}
	maxInt := int(^uint(0) >> 1)
	if len(h.Data) > maxInt-ccBytes {
		return ErrInvalidData
	}
	h.Data = append(h.Data, payload[3:3+ccBytes]...)
	return nil
}

func (h *H2645SEIUnregistered) decodeUnregisteredUserData(payload []byte) error {
	if len(payload) < 16 {
		return ErrInvalidData
	}
	buf := make([]uint8, len(payload)+1)
	copy(buf, payload)
	h.Data = append(h.Data, buf[:len(payload)])
	if build, ok := parseX264Build(payload[16:]); ok {
		h.X264Build = int32(build)
	}
	return nil
}

func (h *H2645SEIDisplayOrientation) decodeDisplayOrientation(payload []byte) error {
	gb := newBitReader(payload)
	cancel, err := gb.readBit()
	if err != nil {
		return err
	}
	h.Present = int32(1 - cancel)
	if h.Present != 0 {
		hflip, err := gb.readBit()
		if err != nil {
			return err
		}
		vflip, err := gb.readBit()
		if err != nil {
			return err
		}
		rotation, err := gb.readBits(16)
		if err != nil {
			return err
		}
		h.HFlip = int32(hflip)
		h.VFlip = int32(vflip)
		h.AnticlockwiseRotation = int32(rotation)
	}
	return nil
}

func (h *H2645SEIFramePacking) decodeFramePackingArrangement(payload []byte) error {
	gb := newBitReader(payload)
	id, err := gb.readUEGolombLong()
	if err != nil {
		return err
	}
	cancel, err := gb.readBit()
	if err != nil {
		return err
	}
	h.ArrangementID = id
	h.ArrangementCancelFlag = int32(cancel)
	h.Present = int32(1 - cancel)
	if h.Present != 0 {
		arrangementType, err := gb.readBits(7)
		if err != nil {
			return err
		}
		quincunx, err := gb.readBit()
		if err != nil {
			return err
		}
		contentInterpretation, err := gb.readBits(6)
		if err != nil {
			return err
		}
		if err := gb.skipBits(3); err != nil {
			return err
		}
		currentFrameIsFrame0, err := gb.readBit()
		if err != nil {
			return err
		}
		if err := gb.skipBits(2); err != nil {
			return err
		}
		if quincunx == 0 && arrangementType != 5 {
			if err := gb.skipBits(16); err != nil {
				return err
			}
		}
		if err := gb.skipBits(8); err != nil {
			return err
		}
		repetition, err := gb.readUEGolombLong()
		if err != nil {
			return err
		}
		h.ArrangementType = int32(arrangementType)
		h.QuincunxSamplingFlag = int32(quincunx)
		h.ContentInterpretationType = int32(contentInterpretation)
		h.CurrentFrameIsFrame0Flag = int32(currentFrameIsFrame0)
		h.ArrangementRepetitionPeriod = repetition
	}
	return gb.skipBits(1)
}

func (h *H2645SEIAlternativeTransfer) decodeAlternativeTransfer(payload []byte) error {
	if len(payload) < 1 {
		return ErrInvalidData
	}
	h.Present = 1
	h.PreferredTransferCharacteristics = int32(payload[0])
	return nil
}

func (h *H2645SEIAmbientViewingEnvironment) decodeAmbientViewingEnvironment(payload []byte) error {
	const maxAmbientLightValue = 50000
	if len(payload) < 8 {
		return ErrInvalidData
	}
	illuminance := readBE32(payload)
	if illuminance == 0 {
		return ErrInvalidData
	}
	lightX := readBE16(payload[4:])
	if lightX > maxAmbientLightValue {
		return ErrInvalidData
	}
	lightY := readBE16(payload[6:])
	if lightY > maxAmbientLightValue {
		return ErrInvalidData
	}
	h.AmbientIlluminance = illuminance
	h.AmbientLightX = lightX
	h.AmbientLightY = lightY
	h.Present = 1
	return nil
}

func (h *H2645SEIFilmGrainCharacteristics) decodeFilmGrainCharacteristics(payload []byte) error {
	gb := newBitReader(payload)
	cancel, err := gb.readBit()
	if err != nil {
		return err
	}
	h.Present = int32(1 - cancel)
	if h.Present == 0 {
		return nil
	}

	*h = H2645SEIFilmGrainCharacteristics{}
	modelID, err := gb.readBits(2)
	if err != nil {
		return err
	}
	separateColour, err := gb.readBit()
	if err != nil {
		return err
	}
	h.ModelID = int32(modelID)
	h.SeparateColourDescriptionPresentFlag = int32(separateColour)
	if separateColour != 0 {
		bitDepthLuma, err := gb.readBits(3)
		if err != nil {
			return err
		}
		bitDepthChroma, err := gb.readBits(3)
		if err != nil {
			return err
		}
		fullRange, err := gb.readBit()
		if err != nil {
			return err
		}
		colorPrimaries, err := gb.readBits(8)
		if err != nil {
			return err
		}
		transferCharacteristics, err := gb.readBits(8)
		if err != nil {
			return err
		}
		matrixCoeffs, err := gb.readBits(8)
		if err != nil {
			return err
		}
		h.BitDepthLuma = int32(bitDepthLuma + 8)
		h.BitDepthChroma = int32(bitDepthChroma + 8)
		h.FullRange = int32(fullRange)
		h.ColorPrimaries = int32(colorPrimaries)
		h.TransferCharacteristics = int32(transferCharacteristics)
		h.MatrixCoeffs = int32(matrixCoeffs)
	}
	blendingMode, err := gb.readBits(2)
	if err != nil {
		return err
	}
	log2Scale, err := gb.readBits(4)
	if err != nil {
		return err
	}
	h.BlendingModeID = int32(blendingMode)
	h.Log2ScaleFactor = int32(log2Scale)
	for c := 0; c < 3; c++ {
		present, err := gb.readBit()
		if err != nil {
			return err
		}
		h.CompModelPresentFlag[c] = int32(present)
	}
	for c := 0; c < 3; c++ {
		if h.CompModelPresentFlag[c] == 0 {
			continue
		}
		intervals, err := gb.readBits(8)
		if err != nil {
			return err
		}
		values, err := gb.readBits(3)
		if err != nil {
			return err
		}
		h.NumIntensityIntervals[c] = uint16(intervals + 1)
		h.NumModelValues[c] = uint8(values + 1)
		if h.NumModelValues[c] > 6 {
			return ErrInvalidData
		}
		for i := 0; i < int(h.NumIntensityIntervals[c]); i++ {
			lower, err := gb.readBits(8)
			if err != nil {
				return err
			}
			upper, err := gb.readBits(8)
			if err != nil {
				return err
			}
			h.IntensityIntervalLowerBound[c][i] = uint8(lower)
			h.IntensityIntervalUpperBound[c][i] = uint8(upper)
			for j := 0; j < int(h.NumModelValues[c]); j++ {
				value, err := gb.readSEGolombLong()
				if err != nil {
					return err
				}
				h.CompModelValue[c][i][j] = int16(value)
			}
		}
	}
	repetitionPeriod, err := gb.readUEGolombLong()
	if err != nil {
		return err
	}
	h.RepetitionPeriod = repetitionPeriod
	h.Present = 1
	return nil
}

func (h *H2645SEIMasteringDisplay) decodeMasteringDisplay(payload []byte) error {
	if len(payload) < 24 {
		return ErrInvalidData
	}
	pos := 0
	for i := 0; i < 3; i++ {
		h.DisplayPrimaries[i][0] = readBE16(payload[pos:])
		h.DisplayPrimaries[i][1] = readBE16(payload[pos+2:])
		pos += 4
	}
	h.WhitePoint[0] = readBE16(payload[pos:])
	h.WhitePoint[1] = readBE16(payload[pos+2:])
	pos += 4
	h.MaxLuminance = readBE32(payload[pos:])
	h.MinLuminance = readBE32(payload[pos+4:])
	h.Present = 2
	return nil
}

func (h *H2645SEIContentLight) decodeContentLight(payload []byte) error {
	if len(payload) < 4 {
		return ErrInvalidData
	}
	h.MaxContentLightLevel = uint32(readBE16(payload))
	h.MaxPicAverageLightLevel = uint32(readBE16(payload[2:]))
	h.Present = 2
	return nil
}

func parseX264Build(data []byte) (int, bool) {
	const prefix = "x264 - core "
	if len(data) < len(prefix) || string(data[:len(prefix)]) != prefix {
		return 0, false
	}
	i := len(prefix)
	build := 0
	digits := 0
	for i < len(data) && data[i] >= '0' && data[i] <= '9' {
		build = build*10 + int(data[i]-'0')
		i++
		digits++
	}
	if digits == 0 {
		return 0, false
	}
	if build > 0 {
		if build == 1 && len(data) >= len(prefix)+4 && string(data[len(prefix):len(prefix)+4]) == "0000" {
			return 67, true
		}
		return build, true
	}
	return 0, false
}

func peekBE16(buf []byte) uint16 {
	if len(buf) < 2 {
		return 0
	}
	return uint16(buf[0])<<8 | uint16(buf[1])
}

func readBE16(buf []byte) uint16 {
	return uint16(buf[0])<<8 | uint16(buf[1])
}

func readBE32(buf []byte) uint32 {
	return uint32(buf[0])<<24 | uint32(buf[1])<<16 | uint32(buf[2])<<8 | uint32(buf[3])
}

var errParamSetNotFound = errSentinel("h264: parameter set not found")

type errSentinel string

func (e errSentinel) Error() string {
	return string(e)
}
