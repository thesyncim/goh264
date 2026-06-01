// SPDX-License-Identifier: LGPL-2.1-or-later

package goh264

import (
	"math"

	"github.com/thesyncim/goh264/internal/h264"
)

var (
	ErrInvalidData = h264.ErrInvalidData
	ErrUnsupported = h264.ErrUnsupported
)

type Decoder struct {
	sps              [32]*h264.SPS
	pps              [256]*h264.PPS
	slices           []h264.SliceHeader
	avcNALLengthSize int
	simple           h264.SimpleDecoder
}

type StreamInfo struct {
	SPSID                          uint32
	ProfileIDC                     uint8
	Profile                        string
	LevelIDC                       uint8
	Width                          int
	Height                         int
	ChromaFormatIDC                uint32
	BitDepthLuma                   int
	BitDepthChroma                 int
	SARNum                         int32
	SARDen                         int32
	VideoFormat                    int32
	VideoFullRangeFlag             int32
	ColorPrimaries                 int32
	ColorTransfer                  int32
	ColorMatrix                    int32
	ChromaLocation                 int32
	ChromaSampleLocTypeTopField    int32
	ChromaSampleLocTypeBottomField int32
	TimingInfoPresentFlag          int32
	NumUnitsInTick                 uint32
	TimeScale                      uint32
	FixedFrameRateFlag             int32
}

type AVCDecoderConfiguration struct {
	NALLengthSize int
	StreamInfo    StreamInfo
}

type PacketSideDataType uint8

const (
	PacketSideDataNewExtradata PacketSideDataType = 1
)

type PacketSideData struct {
	Type PacketSideDataType
	Data []byte
}

type Packet struct {
	Data     []byte
	SideData []PacketSideData
}

type FrameSideData struct {
	UserDataUnregistered [][]byte
	A53ClosedCaptions    []byte
	X264Build            int
	PictureTiming        *PictureTiming
	RecoveryPoint        *RecoveryPoint
	BufferingPeriod      *BufferingPeriod
	GreenMetadata        *GreenMetadata
	ActiveFormat         *ActiveFormat
	FramePacking         *FramePackingArrangement
	Stereo3D             *Stereo3D
	DisplayOrientation   *DisplayOrientation
	AlternativeTransfer  *AlternativeTransfer
	AmbientViewing       *AmbientViewingEnvironment
	FilmGrain            *FilmGrainCharacteristics
	MasteringDisplay     *MasteringDisplay
	ContentLight         *ContentLight
}

type PictureTiming struct {
	PicStruct       int32
	CTType          int32
	DPBOutputDelay  int32
	CPBRemovalDelay int32
	Timecode        []Timecode
}

type Timecode struct {
	Full      bool
	Frame     int32
	Seconds   int32
	Minutes   int32
	Hours     int32
	DropFrame bool
}

type RecoveryPoint struct {
	RecoveryFrameCount int32
}

type BufferingPeriod struct {
	InitialCPBRemovalDelay [32]int32
}

type GreenMetadata struct {
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

type ActiveFormat struct {
	Description uint8
}

type FramePackingArrangement struct {
	ArrangementID               uint32
	ArrangementCancelFlag       bool
	ArrangementType             int32
	ArrangementRepetitionPeriod uint32
	ContentInterpretationType   int32
	QuincunxSamplingFlag        bool
	CurrentFrameIsFrame0Flag    bool
}

type Stereo3DType uint8

const (
	Stereo3DType2D Stereo3DType = iota
	Stereo3DTypeSideBySide
	Stereo3DTypeTopBottom
	Stereo3DTypeFrameSequence
	Stereo3DTypeCheckerboard
	Stereo3DTypeSideBySideQuincunx
	Stereo3DTypeLines
	Stereo3DTypeColumns
)

type Stereo3DView uint8

const (
	Stereo3DViewPacked Stereo3DView = iota
	Stereo3DViewLeft
	Stereo3DViewRight
	Stereo3DViewUnspecified
)

type Stereo3D struct {
	Type       Stereo3DType
	Inverted   bool
	View       Stereo3DView
	StereoMode string
}

type DisplayOrientation struct {
	AnticlockwiseRotation int32
	HFlip                 bool
	VFlip                 bool
	Matrix                [9]int32
}

type AlternativeTransfer struct {
	PreferredTransferCharacteristics int32
}

type AmbientViewingEnvironment struct {
	AmbientIlluminance uint32
	AmbientLightX      uint16
	AmbientLightY      uint16
}

type FilmGrainCharacteristics struct {
	ModelID                              int32
	SeparateColourDescriptionPresentFlag bool
	BitDepthLuma                         int32
	BitDepthChroma                       int32
	FullRange                            bool
	ColorPrimaries                       int32
	TransferCharacteristics              int32
	MatrixCoeffs                         int32
	BlendingModeID                       int32
	Log2ScaleFactor                      int32
	CompModelPresentFlag                 [3]bool
	NumIntensityIntervals                [3]uint16
	NumModelValues                       [3]uint8
	IntensityIntervalLowerBound          [3][256]uint8
	IntensityIntervalUpperBound          [3][256]uint8
	CompModelValue                       [3][256][6]int16
	RepetitionPeriod                     uint32
}

type MasteringDisplay struct {
	DisplayPrimaries [3][2]uint16
	WhitePoint       [2]uint16
	MaxLuminance     uint32
	MinLuminance     uint32
	HasPrimaries     bool
	HasLuminance     bool
}

type ContentLight struct {
	MaxContentLightLevel    uint16
	MaxPicAverageLightLevel uint16
}

type Frame struct {
	Width                          int
	Height                         int
	CropLeft                       int
	CropTop                        int
	ChromaFormatIDC                uint32
	BitDepthLuma                   int
	BitDepthChroma                 int
	SARNum                         int32
	SARDen                         int32
	VideoFormat                    int32
	VideoFullRangeFlag             int32
	ColorPrimaries                 int32
	ColorTransfer                  int32
	ColorMatrix                    int32
	ChromaLocation                 int32
	ChromaSampleLocTypeTopField    int32
	ChromaSampleLocTypeBottomField int32
	TimingInfoPresentFlag          int32
	NumUnitsInTick                 uint32
	TimeScale                      uint32
	FixedFrameRateFlag             int32
	RepeatPict                     int
	InterlacedFrame                bool
	TopFieldFirst                  bool
	YStride                        int
	CStride                        int
	Y                              []byte
	Cb                             []byte
	Cr                             []byte
	SideData                       FrameSideData
}

func NewDecoder() *Decoder {
	return &Decoder{}
}

func (d *Decoder) Decode(data []byte) (*Frame, error) {
	frames, err := d.DecodeFrames(data)
	if err != nil {
		return nil, err
	}
	if len(frames) != 1 {
		return nil, ErrUnsupported
	}
	return frames[0], nil
}

func (d *Decoder) DecodeFrames(data []byte) ([]*Frame, error) {
	if d == nil {
		return nil, ErrInvalidData
	}
	if len(data) == 0 {
		return d.FlushDelayedFrames()
	}
	if h264.IsAVCDecoderConfigurationRecord(data) {
		cfg, err := h264.DecodeAVCDecoderConfigurationRecord(data)
		if err != nil {
			return nil, err
		}
		d.updateAVCDecoderConfiguration(cfg)
		return nil, nil
	}
	nals, _, err := h264.SplitAutoPacket(data, d.avcNALLengthSize)
	if err != nil {
		return nil, err
	}
	frames, err := d.simple.DecodeNALUnits(nals)
	if err != nil {
		return nil, err
	}
	return framesFromH264(frames), nil
}

func (d *Decoder) DecodePacket(pkt Packet) (*Frame, error) {
	frames, err := d.DecodePacketFrames(pkt)
	if err != nil {
		return nil, err
	}
	if len(frames) != 1 {
		return nil, ErrUnsupported
	}
	return frames[0], nil
}

func (d *Decoder) DecodePacketFrames(pkt Packet) ([]*Frame, error) {
	if d == nil {
		return nil, ErrInvalidData
	}
	if len(pkt.Data) == 0 {
		return d.FlushDelayedFrames()
	}
	for _, side := range pkt.SideData {
		if side.Type != PacketSideDataNewExtradata {
			continue
		}
		if err := d.decodeNewExtradata(side.Data); err != nil {
			return nil, err
		}
	}
	return d.DecodeFrames(pkt.Data)
}

func (d *Decoder) DecodeAnnexB(data []byte) (*Frame, error) {
	frames, err := d.DecodeAnnexBFrames(data)
	if err != nil {
		return nil, err
	}
	if len(frames) != 1 {
		return nil, ErrUnsupported
	}
	return frames[0], nil
}

func (d *Decoder) DecodeAnnexBFrames(data []byte) ([]*Frame, error) {
	if d == nil {
		return nil, ErrInvalidData
	}
	frames, err := h264.DecodeAnnexBSimpleFrames(data)
	if err != nil {
		return nil, err
	}
	out := make([]*Frame, len(frames))
	for i, frame := range frames {
		out[i] = frameFromH264(frame)
	}
	return out, nil
}

func (d *Decoder) DecodeAVC(data []byte, nalLengthSize int) (*Frame, error) {
	frames, err := d.DecodeAVCFrames(data, nalLengthSize)
	if err != nil {
		return nil, err
	}
	if len(frames) != 1 {
		return nil, ErrUnsupported
	}
	return frames[0], nil
}

func (d *Decoder) DecodeAVCFrames(data []byte, nalLengthSize int) ([]*Frame, error) {
	if d == nil {
		return nil, ErrInvalidData
	}
	frames, err := h264.DecodeAVCSimpleFrames(data, nalLengthSize)
	if err != nil {
		return nil, err
	}
	out := make([]*Frame, len(frames))
	for i, frame := range frames {
		out[i] = frameFromH264(frame)
	}
	return out, nil
}

func (d *Decoder) DecodeConfiguredAVC(data []byte) (*Frame, error) {
	frames, err := d.DecodeConfiguredAVCFrames(data)
	if err != nil {
		return nil, err
	}
	if len(frames) != 1 {
		return nil, ErrUnsupported
	}
	return frames[0], nil
}

func (d *Decoder) DecodeConfiguredAVCFrames(data []byte) ([]*Frame, error) {
	if d == nil || d.avcNALLengthSize == 0 {
		return nil, ErrInvalidData
	}
	frames, err := d.simple.DecodeAVCFrames(data, d.avcNALLengthSize)
	if err != nil {
		return nil, err
	}
	return framesFromH264(frames), nil
}

func (d *Decoder) FlushDelayedFrames() ([]*Frame, error) {
	if d == nil {
		return nil, ErrInvalidData
	}
	frames, err := d.simple.FlushDelayedFrames()
	if err != nil {
		return nil, err
	}
	return framesFromH264(frames), nil
}

func (d *Decoder) DecodeAVCWithConfigurationRecord(config []byte, data []byte) (*Frame, error) {
	frames, err := d.DecodeAVCFramesWithConfigurationRecord(config, data)
	if err != nil {
		return nil, err
	}
	if len(frames) != 1 {
		return nil, ErrUnsupported
	}
	return frames[0], nil
}

func (d *Decoder) DecodeAVCFramesWithConfigurationRecord(config []byte, data []byte) ([]*Frame, error) {
	if d == nil {
		return nil, ErrInvalidData
	}
	cfg, err := h264.DecodeAVCDecoderConfigurationRecord(config)
	if err != nil {
		return nil, err
	}
	d.storeAVCDecoderConfiguration(cfg)
	return d.decodeAVCFramesWithConfig(data, cfg)
}

func (d *Decoder) decodeAVCFramesWithConfig(data []byte, cfg h264.AVCDecoderConfigurationRecord) ([]*Frame, error) {
	frames, err := d.simple.DecodeAVCFramesWithConfig(data, cfg)
	if err != nil {
		return nil, err
	}
	return framesFromH264(frames), nil
}

func framesFromH264(frames []*h264.DecodedFrame) []*Frame {
	out := make([]*Frame, len(frames))
	for i, frame := range frames {
		out[i] = frameFromH264(frame)
	}
	return out
}

func (d *Decoder) ParseHeadersAnnexB(data []byte) (StreamInfo, error) {
	nals, err := h264.SplitAnnexB(data)
	if err != nil {
		return StreamInfo{}, err
	}
	return d.parseHeaders(nals)
}

func (d *Decoder) ParseHeadersAVC(data []byte, nalLengthSize int) (StreamInfo, error) {
	nals, err := h264.SplitAVCC(data, nalLengthSize)
	if err != nil {
		return StreamInfo{}, err
	}
	return d.parseHeaders(nals)
}

func (d *Decoder) ParseAVCDecoderConfigurationRecord(data []byte) (AVCDecoderConfiguration, error) {
	if d == nil {
		return AVCDecoderConfiguration{}, ErrInvalidData
	}
	cfg, err := h264.DecodeAVCDecoderConfigurationRecord(data)
	if err != nil {
		return AVCDecoderConfiguration{}, err
	}
	d.storeAVCDecoderConfiguration(cfg)
	sps := cfg.SPS[cfg.FirstSPSID]
	if sps == nil {
		return AVCDecoderConfiguration{}, ErrInvalidData
	}
	return AVCDecoderConfiguration{
		NALLengthSize: cfg.NALLengthSize,
		StreamInfo:    streamInfoFromSPS(sps),
	}, nil
}

func (d *Decoder) parseHeaders(nals []h264.NALUnit) (StreamInfo, error) {
	if d == nil {
		return StreamInfo{}, ErrInvalidData
	}
	var info StreamInfo
	haveSPS := false
	for _, nal := range nals {
		switch nal.Type {
		case h264.NALSPS:
			sps, err := h264.DecodeSPS(nal.RBSP)
			if err != nil {
				return StreamInfo{}, err
			}
			if sps.SPSID < uint32(len(d.sps)) {
				d.sps[sps.SPSID] = sps
			}
			if !haveSPS {
				info = streamInfoFromSPS(sps)
				haveSPS = true
			}
		case h264.NALPPS:
			pps, err := h264.DecodePPS(nal.RBSP, &d.sps)
			if err != nil {
				return StreamInfo{}, err
			}
			if pps.PPSID < uint32(len(d.pps)) {
				d.pps[pps.PPSID] = pps
			}
		case h264.NALSlice, h264.NALIDRSlice:
			slice, err := h264.ParseSliceHeader(nal, &d.pps)
			if err != nil {
				return StreamInfo{}, err
			}
			d.slices = append(d.slices, *slice)
		default:
			continue
		}
	}

	if !haveSPS {
		return StreamInfo{}, ErrInvalidData
	}
	return info, nil
}

func (d *Decoder) storeAVCDecoderConfiguration(cfg h264.AVCDecoderConfigurationRecord) {
	d.sps = cfg.SPS
	d.pps = cfg.PPS
	d.avcNALLengthSize = cfg.NALLengthSize
	_ = d.simple.StoreAVCDecoderConfiguration(cfg)
}

func (d *Decoder) updateAVCDecoderConfiguration(cfg h264.AVCDecoderConfigurationRecord) {
	d.sps = cfg.SPS
	d.pps = cfg.PPS
	d.avcNALLengthSize = cfg.NALLengthSize
	_ = d.simple.UpdateParamSets(d.sps, d.pps)
}

func (d *Decoder) decodeNewExtradata(data []byte) error {
	if d == nil || len(data) == 0 {
		return ErrInvalidData
	}
	if data[0] == 1 {
		cfg, err := h264.DecodeAVCDecoderConfigurationRecord(data)
		if err != nil {
			return err
		}
		d.updateAVCDecoderConfiguration(cfg)
		return nil
	}
	nals, err := h264.SplitAnnexB(data)
	if err != nil {
		return err
	}
	return d.storeAnnexBParameterSets(nals)
}

func (d *Decoder) storeAnnexBParameterSets(nals []h264.NALUnit) error {
	if d == nil {
		return ErrInvalidData
	}
	spsList := d.sps
	ppsList := d.pps
	havePS := false
	for _, nal := range nals {
		switch nal.Type {
		case h264.NALSPS:
			sps, err := h264.DecodeSPS(nal.RBSP)
			if err != nil {
				return err
			}
			if sps.SPSID >= uint32(len(spsList)) {
				return ErrInvalidData
			}
			spsList[sps.SPSID] = sps
			havePS = true
		case h264.NALPPS:
			pps, err := h264.DecodePPS(nal.RBSP, &spsList)
			if err != nil {
				return err
			}
			if pps.PPSID >= uint32(len(ppsList)) {
				return ErrInvalidData
			}
			ppsList[pps.PPSID] = pps
			havePS = true
		default:
			continue
		}
	}
	if !havePS {
		return ErrInvalidData
	}
	d.sps = spsList
	d.pps = ppsList
	d.avcNALLengthSize = 0
	return d.simple.UpdateParamSets(d.sps, d.pps)
}

func (f *Frame) AppendRawYUV(dst []byte) ([]byte, error) {
	if f == nil || f.Width <= 0 || f.Height <= 0 {
		return dst, ErrInvalidData
	}
	if f.BitDepthLuma != 8 || f.BitDepthChroma != 8 {
		return dst, ErrUnsupported
	}
	if f.CropLeft < 0 || f.CropTop < 0 || f.YStride < f.Width+f.CropLeft ||
		len(f.Y) < (f.CropTop+f.Height-1)*f.YStride+f.CropLeft+f.Width {
		return dst, ErrInvalidData
	}
	for y := 0; y < f.Height; y++ {
		row := (f.CropTop+y)*f.YStride + f.CropLeft
		dst = append(dst, f.Y[row:row+f.Width]...)
	}

	chromaWidth, chromaHeight, err := frameChromaSize(f.Width, f.Height, f.ChromaFormatIDC)
	if err != nil {
		return dst, err
	}
	if chromaWidth == 0 || chromaHeight == 0 {
		return dst, nil
	}
	chromaCropLeft, chromaCropTop, err := frameChromaCrop(f.CropLeft, f.CropTop, f.ChromaFormatIDC)
	if err != nil {
		return dst, err
	}
	if f.CStride < chromaWidth+chromaCropLeft ||
		len(f.Cb) < (chromaCropTop+chromaHeight-1)*f.CStride+chromaCropLeft+chromaWidth ||
		len(f.Cr) < (chromaCropTop+chromaHeight-1)*f.CStride+chromaCropLeft+chromaWidth {
		return dst, ErrInvalidData
	}
	for y := 0; y < chromaHeight; y++ {
		row := (chromaCropTop+y)*f.CStride + chromaCropLeft
		dst = append(dst, f.Cb[row:row+chromaWidth]...)
	}
	for y := 0; y < chromaHeight; y++ {
		row := (chromaCropTop+y)*f.CStride + chromaCropLeft
		dst = append(dst, f.Cr[row:row+chromaWidth]...)
	}
	return dst, nil
}

func frameFromH264(src *h264.DecodedFrame) *Frame {
	if src == nil {
		return nil
	}
	return &Frame{
		Width:                          src.Width,
		Height:                         src.Height,
		CropLeft:                       src.CropLeft,
		CropTop:                        src.CropTop,
		ChromaFormatIDC:                uint32(src.ChromaFormatIDC),
		BitDepthLuma:                   src.BitDepthLuma,
		BitDepthChroma:                 src.BitDepthChroma,
		SARNum:                         src.SARNum,
		SARDen:                         src.SARDen,
		VideoFormat:                    src.VideoFormat,
		VideoFullRangeFlag:             src.VideoFullRangeFlag,
		ColorPrimaries:                 src.ColorPrimaries,
		ColorTransfer:                  src.ColorTransfer,
		ColorMatrix:                    src.ColorMatrix,
		ChromaLocation:                 src.ChromaLocation,
		ChromaSampleLocTypeTopField:    src.ChromaSampleLocTypeTopField,
		ChromaSampleLocTypeBottomField: src.ChromaSampleLocTypeBottomField,
		TimingInfoPresentFlag:          src.TimingInfoPresentFlag,
		NumUnitsInTick:                 src.NumUnitsInTick,
		TimeScale:                      src.TimeScale,
		FixedFrameRateFlag:             src.FixedFrameRateFlag,
		RepeatPict:                     src.RepeatPict,
		InterlacedFrame:                src.InterlacedFrame,
		TopFieldFirst:                  src.TopFieldFirst,
		YStride:                        src.LumaStride,
		CStride:                        src.ChromaStride,
		Y:                              src.Y,
		Cb:                             src.Cb,
		Cr:                             src.Cr,
		SideData:                       frameSideDataFromH264(src.SideData),
	}
}

func frameSideDataFromH264(src h264.DecodedFrameSideData) FrameSideData {
	out := FrameSideData{
		UserDataUnregistered: cloneByteSlices(src.UserDataUnregistered),
		A53ClosedCaptions:    append([]byte(nil), src.A53ClosedCaptions...),
		X264Build:            int(src.X264Build),
	}
	if src.PictureTiming.Present != 0 {
		pt := PictureTiming{
			PicStruct:       src.PictureTiming.PicStruct,
			CTType:          src.PictureTiming.CTType,
			DPBOutputDelay:  src.PictureTiming.DPBOutputDelay,
			CPBRemovalDelay: src.PictureTiming.CPBRemovalDelay,
		}
		count := int(src.PictureTiming.TimecodeCount)
		if count > len(src.PictureTiming.Timecode) {
			count = len(src.PictureTiming.Timecode)
		}
		for i := 0; i < count; i++ {
			tc := src.PictureTiming.Timecode[i]
			pt.Timecode = append(pt.Timecode, Timecode{
				Full:      tc.Full != 0,
				Frame:     tc.Frame,
				Seconds:   tc.Seconds,
				Minutes:   tc.Minutes,
				Hours:     tc.Hours,
				DropFrame: tc.DropFrame != 0,
			})
		}
		out.PictureTiming = &pt
	}
	if src.RecoveryPoint.RecoveryFrameCount >= 0 {
		out.RecoveryPoint = &RecoveryPoint{RecoveryFrameCount: src.RecoveryPoint.RecoveryFrameCount}
	}
	if src.BufferingPeriod.Present != 0 {
		out.BufferingPeriod = &BufferingPeriod{InitialCPBRemovalDelay: src.BufferingPeriod.InitialCPBRemovalDelay}
	}
	if src.GreenMetadata.Present != 0 {
		out.GreenMetadata = &GreenMetadata{
			GreenMetadataType:                   src.GreenMetadata.GreenMetadataType,
			PeriodType:                          src.GreenMetadata.PeriodType,
			NumSeconds:                          src.GreenMetadata.NumSeconds,
			NumPictures:                         src.GreenMetadata.NumPictures,
			PercentNonZeroMacroblocks:           src.GreenMetadata.PercentNonZeroMacroblocks,
			PercentIntraCodedMacroblocks:        src.GreenMetadata.PercentIntraCodedMacroblocks,
			PercentSixTapFiltering:              src.GreenMetadata.PercentSixTapFiltering,
			PercentAlphaPointDeblockingInstance: src.GreenMetadata.PercentAlphaPointDeblockingInstance,
			XSDMetricType:                       src.GreenMetadata.XSDMetricType,
			XSDMetricValue:                      src.GreenMetadata.XSDMetricValue,
		}
	}
	if src.AFD.Present != 0 {
		out.ActiveFormat = &ActiveFormat{Description: src.AFD.ActiveFormatDescription}
	}
	if src.FramePacking.Present != 0 {
		out.FramePacking = &FramePackingArrangement{
			ArrangementID:               src.FramePacking.ArrangementID,
			ArrangementCancelFlag:       src.FramePacking.ArrangementCancelFlag != 0,
			ArrangementType:             src.FramePacking.ArrangementType,
			ArrangementRepetitionPeriod: src.FramePacking.ArrangementRepetitionPeriod,
			ContentInterpretationType:   src.FramePacking.ContentInterpretationType,
			QuincunxSamplingFlag:        src.FramePacking.QuincunxSamplingFlag != 0,
			CurrentFrameIsFrame0Flag:    src.FramePacking.CurrentFrameIsFrame0Flag != 0,
		}
		out.Stereo3D = stereo3DFromFramePacking(src.FramePacking)
	}
	out.DisplayOrientation = displayOrientationFromH264(src.DisplayOrientation)
	if src.AlternativeTransfer.Present != 0 {
		out.AlternativeTransfer = &AlternativeTransfer{
			PreferredTransferCharacteristics: src.AlternativeTransfer.PreferredTransferCharacteristics,
		}
	}
	if src.AmbientViewing.Present != 0 {
		out.AmbientViewing = &AmbientViewingEnvironment{
			AmbientIlluminance: src.AmbientViewing.AmbientIlluminance,
			AmbientLightX:      src.AmbientViewing.AmbientLightX,
			AmbientLightY:      src.AmbientViewing.AmbientLightY,
		}
	}
	if src.FilmGrain.Present != 0 {
		out.FilmGrain = filmGrainFromH264(src.FilmGrain)
	}
	if src.MasteringDisplay.Present != 0 {
		out.MasteringDisplay = masteringDisplayFromH264(src.MasteringDisplay)
	}
	if src.ContentLight.Present != 0 {
		out.ContentLight = &ContentLight{
			MaxContentLightLevel:    src.ContentLight.MaxContentLightLevel,
			MaxPicAverageLightLevel: src.ContentLight.MaxPicAverageLightLevel,
		}
	}
	return out
}

func stereo3DFromFramePacking(src h264.H2645SEIFramePacking) *Stereo3D {
	if src.Present == 0 || !validH264FramePackingType(src.ArrangementType) ||
		src.ContentInterpretationType <= 0 || src.ContentInterpretationType >= 3 {
		return nil
	}
	out := &Stereo3D{
		Inverted:   src.ContentInterpretationType == 2,
		View:       Stereo3DViewPacked,
		StereoMode: h264FramePackingStereoMode(src),
	}
	switch src.ArrangementType {
	case 0:
		out.Type = Stereo3DTypeCheckerboard
	case 1:
		out.Type = Stereo3DTypeColumns
	case 2:
		out.Type = Stereo3DTypeLines
	case 3:
		if src.QuincunxSamplingFlag != 0 {
			out.Type = Stereo3DTypeSideBySideQuincunx
		} else {
			out.Type = Stereo3DTypeSideBySide
		}
	case 4:
		out.Type = Stereo3DTypeTopBottom
	case 5:
		out.Type = Stereo3DTypeFrameSequence
		if src.CurrentFrameIsFrame0Flag != 0 {
			out.View = Stereo3DViewLeft
		} else {
			out.View = Stereo3DViewRight
		}
	case 6:
		out.Type = Stereo3DType2D
	}
	return out
}

func validH264FramePackingType(t int32) bool {
	return t >= 0 && t <= 6
}

func h264FramePackingStereoMode(src h264.H2645SEIFramePacking) string {
	if src.ArrangementCancelFlag == 1 {
		return "mono"
	}
	if src.ArrangementCancelFlag != 0 {
		return ""
	}
	rightFirst := src.ContentInterpretationType == 2
	switch src.ArrangementType {
	case 0:
		if rightFirst {
			return "checkerboard_rl"
		}
		return "checkerboard_lr"
	case 1:
		if rightFirst {
			return "col_interleaved_rl"
		}
		return "col_interleaved_lr"
	case 2:
		if rightFirst {
			return "row_interleaved_rl"
		}
		return "row_interleaved_lr"
	case 3:
		if rightFirst {
			return "right_left"
		}
		return "left_right"
	case 4:
		if rightFirst {
			return "bottom_top"
		}
		return "top_bottom"
	case 5:
		if rightFirst {
			return "block_rl"
		}
		return "block_lr"
	default:
		return "mono"
	}
}

func displayOrientationFromH264(src h264.H2645SEIDisplayOrientation) *DisplayOrientation {
	if src.Present == 0 || (src.AnticlockwiseRotation == 0 && src.HFlip == 0 && src.VFlip == 0) {
		return nil
	}
	out := &DisplayOrientation{
		AnticlockwiseRotation: src.AnticlockwiseRotation,
		HFlip:                 src.HFlip != 0,
		VFlip:                 src.VFlip != 0,
	}
	angle := float64(src.AnticlockwiseRotation) * 360 / float64(1<<16)
	angle = -angle
	if src.HFlip != 0 {
		angle = -angle
	}
	if src.VFlip != 0 {
		angle = -angle
	}
	out.Matrix = displayRotationMatrix(angle)
	displayMatrixFlip(&out.Matrix, src.HFlip != 0, src.VFlip != 0)
	return out
}

func displayRotationMatrix(angle float64) [9]int32 {
	radians := -angle * math.Pi / 180.0
	c := math.Cos(radians)
	s := math.Sin(radians)
	return [9]int32{
		int32(c * float64(1<<16)),
		int32(-s * float64(1<<16)),
		0,
		int32(s * float64(1<<16)),
		int32(c * float64(1<<16)),
		0,
		0,
		0,
		1 << 30,
	}
}

func displayMatrixFlip(matrix *[9]int32, hflip bool, vflip bool) {
	if !hflip && !vflip {
		return
	}
	flip := [3]int32{1, 1, 1}
	if hflip {
		flip[0] = -1
	}
	if vflip {
		flip[1] = -1
	}
	for i := range matrix {
		matrix[i] *= flip[i%3]
	}
}

func masteringDisplayFromH264(src h264.H2645SEIMasteringDisplay) *MasteringDisplay {
	const (
		chromaXMin = 5
		chromaXMax = 37000
		chromaYMin = 5
		chromaYMax = 42000
		lumaMin    = 50000
		lumaMax    = 100000000
	)
	mapping := [3]int{2, 0, 1}
	out := &MasteringDisplay{
		WhitePoint:   src.WhitePoint,
		MaxLuminance: src.MaxLuminance,
		MinLuminance: src.MinLuminance,
		HasPrimaries: true,
	}
	for i, j := range mapping {
		out.DisplayPrimaries[i] = src.DisplayPrimaries[j]
		out.HasPrimaries = out.HasPrimaries &&
			out.DisplayPrimaries[i][0] >= chromaXMin && out.DisplayPrimaries[i][0] <= chromaXMax &&
			out.DisplayPrimaries[i][1] >= chromaYMin && out.DisplayPrimaries[i][1] <= chromaYMax
	}
	out.HasPrimaries = out.HasPrimaries &&
		out.WhitePoint[0] >= chromaXMin && out.WhitePoint[0] <= chromaXMax &&
		out.WhitePoint[1] >= chromaYMin && out.WhitePoint[1] <= chromaYMax
	out.HasLuminance = out.MaxLuminance >= lumaMin && out.MaxLuminance <= lumaMax &&
		out.MinLuminance <= lumaMin && out.MinLuminance < out.MaxLuminance
	return out
}

func filmGrainFromH264(src h264.H2645SEIFilmGrainCharacteristics) *FilmGrainCharacteristics {
	out := &FilmGrainCharacteristics{
		ModelID:                              src.ModelID,
		SeparateColourDescriptionPresentFlag: src.SeparateColourDescriptionPresentFlag != 0,
		BitDepthLuma:                         src.BitDepthLuma,
		BitDepthChroma:                       src.BitDepthChroma,
		FullRange:                            src.FullRange != 0,
		ColorPrimaries:                       src.ColorPrimaries,
		TransferCharacteristics:              src.TransferCharacteristics,
		MatrixCoeffs:                         src.MatrixCoeffs,
		BlendingModeID:                       src.BlendingModeID,
		Log2ScaleFactor:                      src.Log2ScaleFactor,
		NumIntensityIntervals:                src.NumIntensityIntervals,
		NumModelValues:                       src.NumModelValues,
		IntensityIntervalLowerBound:          src.IntensityIntervalLowerBound,
		IntensityIntervalUpperBound:          src.IntensityIntervalUpperBound,
		CompModelValue:                       src.CompModelValue,
		RepetitionPeriod:                     src.RepetitionPeriod,
	}
	for i := range src.CompModelPresentFlag {
		out.CompModelPresentFlag[i] = src.CompModelPresentFlag[i] != 0
	}
	return out
}

func cloneByteSlices(src [][]byte) [][]byte {
	if len(src) == 0 {
		return nil
	}
	out := make([][]byte, len(src))
	for i := range src {
		out[i] = append([]byte(nil), src[i]...)
	}
	return out
}

func frameChromaSize(width int, height int, chromaFormatIDC uint32) (int, int, error) {
	switch chromaFormatIDC {
	case 0:
		return 0, 0, nil
	case 1:
		return (width + 1) >> 1, (height + 1) >> 1, nil
	case 2:
		return (width + 1) >> 1, height, nil
	case 3:
		return width, height, nil
	default:
		return 0, 0, ErrInvalidData
	}
}

func frameChromaCrop(cropLeft int, cropTop int, chromaFormatIDC uint32) (int, int, error) {
	if cropLeft < 0 || cropTop < 0 {
		return 0, 0, ErrInvalidData
	}
	switch chromaFormatIDC {
	case 0, 3:
		return cropLeft, cropTop, nil
	case 1:
		return cropLeft >> 1, cropTop >> 1, nil
	case 2:
		return cropLeft >> 1, cropTop, nil
	default:
		return 0, 0, ErrInvalidData
	}
}

func streamInfoFromSPS(sps *h264.SPS) StreamInfo {
	profileIDC := uint8(sps.ProfileIDC)
	return StreamInfo{
		SPSID:                          sps.SPSID,
		ProfileIDC:                     profileIDC,
		Profile:                        profileName(profileIDC, uint8(sps.ConstraintSetFlags)),
		LevelIDC:                       uint8(sps.LevelIDC),
		Width:                          int(sps.Width),
		Height:                         int(sps.Height),
		ChromaFormatIDC:                sps.ChromaFormatIDC,
		BitDepthLuma:                   int(sps.BitDepthLuma),
		BitDepthChroma:                 int(sps.BitDepthChroma),
		SARNum:                         sps.VUI.SARNum,
		SARDen:                         sps.VUI.SARDen,
		VideoFormat:                    sps.VUI.VideoFormat,
		VideoFullRangeFlag:             sps.VUI.VideoFullRangeFlag,
		ColorPrimaries:                 sps.VUI.ColourPrimaries,
		ColorTransfer:                  sps.VUI.TransferCharacteristics,
		ColorMatrix:                    sps.VUI.MatrixCoeffs,
		ChromaLocation:                 sps.VUI.ChromaLocation,
		ChromaSampleLocTypeTopField:    sps.VUI.ChromaSampleLocTypeTopField,
		ChromaSampleLocTypeBottomField: sps.VUI.ChromaSampleLocTypeBottomField,
		TimingInfoPresentFlag:          sps.TimingInfoPresentFlag,
		NumUnitsInTick:                 sps.NumUnitsInTick,
		TimeScale:                      sps.TimeScale,
		FixedFrameRateFlag:             sps.FixedFrameRateFlag,
	}
}

func profileName(profileIDC uint8, constraintSetFlags uint8) string {
	switch profileIDC {
	case 66:
		if constraintSetFlags&0x03 == 0x03 {
			return "Constrained Baseline"
		}
		return "Baseline"
	case 77:
		return "Main"
	case 88:
		return "Extended"
	case 100:
		return "High"
	case 110:
		return "High 10"
	case 122:
		return "High 4:2:2"
	case 244:
		return "High 4:4:4 Predictive"
	default:
		return "Unknown"
	}
}
