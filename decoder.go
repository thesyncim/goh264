// SPDX-License-Identifier: LGPL-2.1-or-later

package goh264

import (
	"encoding/binary"
	"fmt"
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

// PacketSideDataType identifies auxiliary packet metadata accepted by Packet.
type PacketSideDataType uint8

const (
	PacketSideDataNewExtradata              PacketSideDataType = 1
	PacketSideDataDisplayMatrix             PacketSideDataType = 5
	PacketSideDataStereo3D                  PacketSideDataType = 6
	PacketSideDataMasteringDisplayMetadata  PacketSideDataType = 20
	PacketSideDataSpherical                 PacketSideDataType = 21
	PacketSideDataContentLightLevel         PacketSideDataType = 22
	PacketSideDataA53ClosedCaptions         PacketSideDataType = 23
	PacketSideDataActiveFormat              PacketSideDataType = 26
	PacketSideDataICCProfile                PacketSideDataType = 28
	PacketSideDataS12MTimecode              PacketSideDataType = 30
	PacketSideDataDynamicHDR10Plus          PacketSideDataType = 31
	PacketSideDataAmbientViewingEnvironment PacketSideDataType = 35
	PacketSideDataLCEVC                     PacketSideDataType = 37
	PacketSideData3DReferenceDisplays       PacketSideDataType = 38
)

// PacketSideData carries one FFmpeg-compatible packet side-data payload.
//
// Data is borrowed only for the duration of DecodePacket or DecodePacketFrames.
// Decoded FrameSideData byte slices are independent copies and can be mutated
// without affecting the input packet.
type PacketSideData struct {
	Type PacketSideDataType
	Data []byte
}

// Packet is one compressed access unit plus optional packet side data.
//
// Data and SideData payloads are caller-owned input. The decoder reads them
// during the call and does not retain their backing storage after returning.
type Packet struct {
	Data     []byte
	SideData []PacketSideData
}

// FrameSideData contains SEI and packet side-data values attached to a decoded
// frame.
//
// Slice fields are caller-owned output copies. Mutating them does not affect
// decoder state or packet buffers supplied to DecodePacket/DecodePacketFrames.
type FrameSideData struct {
	UserDataUnregistered [][]byte
	A53ClosedCaptions    []byte
	X264Build            int
	S12MTimecodes        []uint32
	PictureTiming        *PictureTiming
	RecoveryPoint        *RecoveryPoint
	BufferingPeriod      *BufferingPeriod
	GreenMetadata        *GreenMetadata
	ActiveFormat         *ActiveFormat
	FramePacking         *FramePackingArrangement
	Stereo3D             *Stereo3D
	Spherical            *SphericalMapping
	DisplayOrientation   *DisplayOrientation
	AlternativeTransfer  *AlternativeTransfer
	AmbientViewing       *AmbientViewingEnvironment
	FilmGrain            *FilmGrainCharacteristics
	MasteringDisplay     *MasteringDisplay
	ContentLight         *ContentLight
	ICCProfile           []byte
	DynamicHDR10Plus     []byte
	LCEVC                []byte
	ReferenceDisplays    *ReferenceDisplaysInfo
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
	Stereo3DTypeUnspecified
)

type Stereo3DView uint8

const (
	Stereo3DViewPacked Stereo3DView = iota
	Stereo3DViewLeft
	Stereo3DViewRight
	Stereo3DViewUnspecified
)

type Stereo3DPrimaryEye uint8

const (
	Stereo3DPrimaryEyeNone Stereo3DPrimaryEye = iota
	Stereo3DPrimaryEyeLeft
	Stereo3DPrimaryEyeRight
)

type Rational struct {
	Num int32
	Den int32
}

type Stereo3D struct {
	Type                          Stereo3DType
	Inverted                      bool
	View                          Stereo3DView
	PrimaryEye                    Stereo3DPrimaryEye
	Baseline                      uint32
	HorizontalDisparityAdjustment Rational
	HorizontalFieldOfView         Rational
	StereoMode                    string
}

type SphericalProjection uint8

const (
	SphericalProjectionEquirectangular SphericalProjection = iota
	SphericalProjectionCubemap
	SphericalProjectionEquirectangularTile
	SphericalProjectionHalfEquirectangular
	SphericalProjectionRectilinear
	SphericalProjectionFisheye
	SphericalProjectionParametricImmersive
)

type SphericalMapping struct {
	Projection  SphericalProjection
	Yaw         int32
	Pitch       int32
	Roll        int32
	BoundLeft   uint32
	BoundTop    uint32
	BoundRight  uint32
	BoundBottom uint32
	Padding     uint32
}

type ReferenceDisplaysInfo struct {
	PrecRefDisplayWidth    uint8
	RefViewingDistanceFlag bool
	PrecRefViewingDist     uint8
	Displays               []ReferenceDisplay
}

type ReferenceDisplay struct {
	LeftViewID                 uint16
	RightViewID                uint16
	ExponentRefDisplayWidth    uint8
	MantissaRefDisplayWidth    uint8
	ExponentRefViewingDistance uint8
	MantissaRefViewingDistance uint8
	AdditionalShiftPresentFlag bool
	NumSampleShift             int16
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
	MaxContentLightLevel    uint32
	MaxPicAverageLightLevel uint32
}

// Frame is one decoded output picture.
//
// For 8-bit pictures, Y, Cb, and Cr contain the decoded planes. For high
// bit-depth pictures, Y16, Cb16, and Cr16 contain the decoded planes. Width and
// Height describe the visible output size; CropLeft and CropTop locate that
// visible rectangle inside the underlying plane buffers.
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
	KeyFrame                       bool
	YStride                        int
	CStride                        int
	Y                              []byte
	Cb                             []byte
	Cr                             []byte
	Y16                            []uint16
	Cb16                           []uint16
	Cr16                           []uint16
	SideData                       FrameSideData
}

// NewDecoder returns a stateful decoder.
func NewDecoder() *Decoder {
	return &Decoder{}
}

// Decode decodes data through DecodeFrames and returns the single output frame.
//
// It returns ErrUnsupported when the packet produces zero or multiple frames
// without another decode error.
func (d *Decoder) Decode(data []byte) (*Frame, error) {
	frames, err := d.DecodeFrames(data)
	return singleFrameFromFrames(frames, err)
}

// DecodeFrames decodes an auto-detected Annex B or configured-AVC packet.
//
// Passing an AVC decoder configuration record updates configured-AVC state and
// returns no frames. Passing empty data flushes delayed frames.
func (d *Decoder) DecodeFrames(data []byte) ([]*Frame, error) {
	return d.decodeFrames(data, h264.DecodedFrameSideData{})
}

func (d *Decoder) decodeFrames(data []byte, packetSideData h264.DecodedFrameSideData) ([]*Frame, error) {
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
	frames, err := d.simple.DecodeNALUnitsWithSideData(nals, packetSideData)
	if err != nil {
		return framesFromH264WithError(frames, err)
	}
	return framesFromH264(frames), nil
}

// DecodePacket decodes pkt through DecodePacketFrames and returns the single
// output frame.
//
// It returns ErrUnsupported when the packet produces zero or multiple frames
// without another decode error.
func (d *Decoder) DecodePacket(pkt Packet) (*Frame, error) {
	frames, err := d.DecodePacketFrames(pkt)
	return singleFrameFromFrames(frames, err)
}

// DecodePacketFrames decodes a compressed packet plus FFmpeg-compatible packet
// side data.
//
// PacketSideDataNewExtradata is parsed before packet data and is non-fatal when
// malformed, leaving the previous configuration in use. Passing an empty packet
// flushes delayed frames and does not attach packet side data.
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
		// h264_decode_frame calls ff_h264_decode_extradata for packet
		// NEW_EXTRADATA and ignores its return value before decoding buf.
		_ = d.decodeNewExtradata(side.Data)
	}
	return d.decodeFrames(pkt.Data, packetFrameSideDataFromPacket(pkt.SideData))
}

// DecodeAnnexB decodes a complete Annex B byte stream and returns the single
// output frame.
func (d *Decoder) DecodeAnnexB(data []byte) (*Frame, error) {
	frames, err := d.DecodeAnnexBFrames(data)
	return singleFrameFromFrames(frames, err)
}

// DecodeAnnexBFrames decodes a complete Annex B byte stream in one call.
//
// For stateful packet streaming and delayed-output flushing, use DecodeFrames.
func (d *Decoder) DecodeAnnexBFrames(data []byte) ([]*Frame, error) {
	if d == nil {
		return nil, ErrInvalidData
	}
	frames, err := h264.DecodeAnnexBSimpleFrames(data)
	if err != nil {
		return framesFromH264WithError(frames, err)
	}
	return framesFromH264(frames), nil
}

// DecodeAVC decodes a complete AVC packet stream with the supplied NAL length
// size and returns the single output frame.
func (d *Decoder) DecodeAVC(data []byte, nalLengthSize int) (*Frame, error) {
	frames, err := d.DecodeAVCFrames(data, nalLengthSize)
	return singleFrameFromFrames(frames, err)
}

// DecodeAVCFrames decodes a complete AVC packet stream with the supplied NAL
// length size in one call.
//
// For stateful AVC streaming, first parse an AVC decoder configuration record
// and then use DecodeConfiguredAVCFrames.
func (d *Decoder) DecodeAVCFrames(data []byte, nalLengthSize int) ([]*Frame, error) {
	if d == nil {
		return nil, ErrInvalidData
	}
	frames, err := h264.DecodeAVCSimpleFrames(data, nalLengthSize)
	if err != nil {
		return framesFromH264WithError(frames, err)
	}
	return framesFromH264(frames), nil
}

// DecodeConfiguredAVC decodes data using the stored AVC configuration and
// returns the single output frame.
func (d *Decoder) DecodeConfiguredAVC(data []byte) (*Frame, error) {
	frames, err := d.DecodeConfiguredAVCFrames(data)
	return singleFrameFromFrames(frames, err)
}

// DecodeConfiguredAVCFrames decodes a stateful AVC packet using the stored AVC
// configuration.
//
// The stored configuration is set by ParseAVCDecoderConfigurationRecord or by a
// prior AVC decoder configuration record passed to DecodeFrames. Passing empty
// data flushes delayed frames.
func (d *Decoder) DecodeConfiguredAVCFrames(data []byte) ([]*Frame, error) {
	if d == nil || d.avcNALLengthSize == 0 {
		return nil, ErrInvalidData
	}
	if len(data) == 0 {
		return d.FlushDelayedFrames()
	}
	frames, err := d.simple.DecodeAVCFrames(data, d.avcNALLengthSize)
	if err != nil {
		return framesFromH264WithError(frames, err)
	}
	return framesFromH264(frames), nil
}

// FlushDelayedFrames drains decoded frames held for delayed output.
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

// DecodeAVCWithConfigurationRecord updates the stored AVC configuration, decodes
// data, and returns the single output frame.
func (d *Decoder) DecodeAVCWithConfigurationRecord(config []byte, data []byte) (*Frame, error) {
	frames, err := d.DecodeAVCFramesWithConfigurationRecord(config, data)
	return singleFrameFromFrames(frames, err)
}

// DecodeAVCFramesWithConfigurationRecord updates the stored AVC configuration
// from config and decodes data with that configuration.
//
// For non-empty data this helper also drains delayed frames before returning.
// Passing empty data updates the configuration and flushes existing delayed
// frames.
func (d *Decoder) DecodeAVCFramesWithConfigurationRecord(config []byte, data []byte) ([]*Frame, error) {
	if d == nil {
		return nil, ErrInvalidData
	}
	cfg, err := h264.DecodeAVCDecoderConfigurationRecord(config)
	if err != nil {
		return nil, err
	}
	d.updateAVCDecoderConfiguration(cfg)
	return d.decodeAVCFramesWithConfig(data, cfg)
}

func (d *Decoder) decodeAVCFramesWithConfig(data []byte, cfg h264.AVCDecoderConfigurationRecord) ([]*Frame, error) {
	if len(data) == 0 {
		return d.FlushDelayedFrames()
	}
	frames, decodeErr := d.simple.DecodeAVCFrames(data, cfg.NALLengthSize)
	flushed, flushErr := d.simple.FlushDelayedFrames()
	if len(flushed) != 0 {
		frames = append(frames, flushed...)
	}
	if decodeErr != nil {
		if flushErr != nil {
			decodeErr = fmt.Errorf("%v; flush delayed: %w", decodeErr, flushErr)
		}
		return framesFromH264WithError(frames, decodeErr)
	}
	if flushErr != nil {
		return framesFromH264WithError(frames, flushErr)
	}
	return framesFromH264(frames), nil
}

func framesFromH264WithError(frames []*h264.DecodedFrame, err error) ([]*Frame, error) {
	if len(frames) != 0 {
		return framesFromH264(frames), err
	}
	return nil, err
}

func singleFrameFromFrames(frames []*Frame, err error) (*Frame, error) {
	if len(frames) == 1 {
		return frames[0], err
	}
	if err != nil {
		return nil, err
	}
	return nil, ErrUnsupported
}

func framesFromH264(frames []*h264.DecodedFrame) []*Frame {
	if len(frames) == 0 || len(frames) > maxInt/8 {
		return nil
	}
	out := make([]*Frame, len(frames))
	for i, frame := range frames {
		out[i] = frameFromH264(frame)
	}
	return out
}

// ParseHeadersAnnexB parses Annex B parameter sets and returns stream metadata.
func (d *Decoder) ParseHeadersAnnexB(data []byte) (StreamInfo, error) {
	nals, err := h264.SplitAnnexB(data)
	if err != nil {
		return StreamInfo{}, err
	}
	return d.parseHeaders(nals)
}

// ParseHeadersAVC parses AVC parameter sets and returns stream metadata.
func (d *Decoder) ParseHeadersAVC(data []byte, nalLengthSize int) (StreamInfo, error) {
	nals, err := h264.SplitAVCC(data, nalLengthSize)
	if err != nil {
		return StreamInfo{}, err
	}
	return d.parseHeaders(nals)
}

// ParseAVCDecoderConfigurationRecord parses an AVC decoder configuration record,
// stores it for configured-AVC decode calls, and returns stream metadata.
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
			sps, err := h264.DecodeSPSFromNAL(nal)
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
			sps, err := h264.DecodeSPSFromNAL(nal)
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

func packetSideDataGet(sideData []PacketSideData, typ PacketSideDataType) (PacketSideData, bool) {
	for _, side := range sideData {
		if side.Type == typ {
			return side, true
		}
	}
	return PacketSideData{}, false
}

func packetFrameSideDataFromPacket(sideData []PacketSideData) h264.DecodedFrameSideData {
	var out h264.DecodedFrameSideData
	if side, ok := packetSideDataGet(sideData, PacketSideDataA53ClosedCaptions); ok {
		out.A53ClosedCaptions = cloneByteSlice(side.Data)
	}
	if side, ok := packetSideDataGet(sideData, PacketSideDataActiveFormat); ok && len(side.Data) > 0 {
		out.AFD = h264.H2645SEIAFD{
			Present:                 1,
			ActiveFormatDescription: side.Data[0],
		}
	}
	if side, ok := packetSideDataGet(sideData, PacketSideDataS12MTimecode); ok {
		out.S12MTimecodes = s12mTimecodesFromPacketSideData(side.Data)
	}
	if side, ok := packetSideDataGet(sideData, PacketSideDataStereo3D); ok {
		if stereo, ok := stereo3DFromPacketSideData(side.Data); ok {
			out.Stereo3D = stereo
		}
	}
	if side, ok := packetSideDataGet(sideData, PacketSideDataSpherical); ok {
		if spherical, ok := sphericalFromPacketSideData(side.Data); ok {
			out.Spherical = spherical
		}
	}
	if side, ok := packetSideDataGet(sideData, PacketSideDataDisplayMatrix); ok {
		if matrix, ok := displayMatrixFromPacketSideData(side.Data); ok {
			out.DisplayMatrix = matrix
		}
	}
	if side, ok := packetSideDataGet(sideData, PacketSideDataAmbientViewingEnvironment); ok {
		if ambient, ok := ambientViewingFromPacketSideData(side.Data); ok {
			out.AmbientViewing = ambient
		}
	}
	if side, ok := packetSideDataGet(sideData, PacketSideDataMasteringDisplayMetadata); ok {
		if mastering, ok := masteringDisplayFromPacketSideData(side.Data); ok {
			out.MasteringMetadata = mastering
		}
	}
	if side, ok := packetSideDataGet(sideData, PacketSideDataContentLightLevel); ok {
		if light, ok := contentLightFromPacketSideData(side.Data); ok {
			out.ContentLight = light
		}
	}
	if side, ok := packetSideDataGet(sideData, PacketSideDataICCProfile); ok && len(side.Data) != 0 {
		out.ICCProfile = cloneByteSlice(side.Data)
	}
	if side, ok := packetSideDataGet(sideData, PacketSideDataDynamicHDR10Plus); ok && len(side.Data) != 0 {
		out.DynamicHDR10Plus = cloneByteSlice(side.Data)
	}
	if side, ok := packetSideDataGet(sideData, PacketSideDataLCEVC); ok && len(side.Data) != 0 {
		out.LCEVC = cloneByteSlice(side.Data)
	}
	if side, ok := packetSideDataGet(sideData, PacketSideData3DReferenceDisplays); ok {
		if displays, ok := referenceDisplaysFromPacketSideData(side.Data); ok {
			out.ReferenceDisplays = displays
		}
	}
	return out
}

func s12mTimecodesFromPacketSideData(data []byte) []uint32 {
	if len(data) < 4 {
		return nil
	}
	count := int(binary.LittleEndian.Uint32(data[:4]))
	if count < 1 || count > 3 || len(data) < 4*(1+count) {
		return nil
	}
	out := make([]uint32, count)
	for i := 0; i < count; i++ {
		off := 4 * (i + 1)
		out[i] = binary.LittleEndian.Uint32(data[off : off+4])
	}
	return out
}

func displayMatrixFromPacketSideData(data []byte) (h264.AVDisplayMatrix, bool) {
	const displayMatrixSideDataSize = 9 * 4
	if len(data) < displayMatrixSideDataSize {
		return h264.AVDisplayMatrix{}, false
	}
	out := h264.AVDisplayMatrix{Present: 1}
	for i := range out.Matrix {
		off := i * 4
		out.Matrix[i] = int32(binary.LittleEndian.Uint32(data[off : off+4]))
	}
	return out, true
}

func stereo3DFromPacketSideData(data []byte) (h264.AVStereo3D, bool) {
	const stereo3DStructSize = 36
	if len(data) < stereo3DStructSize {
		return h264.AVStereo3D{}, false
	}
	return h264.AVStereo3D{
		Present:    1,
		Type:       int32(binary.LittleEndian.Uint32(data[:4])),
		Flags:      int32(binary.LittleEndian.Uint32(data[4:8])),
		View:       int32(binary.LittleEndian.Uint32(data[8:12])),
		PrimaryEye: int32(binary.LittleEndian.Uint32(data[12:16])),
		Baseline:   binary.LittleEndian.Uint32(data[16:20]),
		HorizontalDisparityAdjustment: h264.AVRational{
			Num: int32(binary.LittleEndian.Uint32(data[20:24])),
			Den: int32(binary.LittleEndian.Uint32(data[24:28])),
		},
		HorizontalFieldOfView: h264.AVRational{
			Num: int32(binary.LittleEndian.Uint32(data[28:32])),
			Den: int32(binary.LittleEndian.Uint32(data[32:36])),
		},
	}, true
}

func sphericalFromPacketSideData(data []byte) (h264.AVSphericalMapping, bool) {
	const sphericalStructSize = 36
	if len(data) < sphericalStructSize {
		return h264.AVSphericalMapping{}, false
	}
	return h264.AVSphericalMapping{
		Present:     1,
		Projection:  int32(binary.LittleEndian.Uint32(data[0:4])),
		Yaw:         int32(binary.LittleEndian.Uint32(data[4:8])),
		Pitch:       int32(binary.LittleEndian.Uint32(data[8:12])),
		Roll:        int32(binary.LittleEndian.Uint32(data[12:16])),
		BoundLeft:   binary.LittleEndian.Uint32(data[16:20]),
		BoundTop:    binary.LittleEndian.Uint32(data[20:24]),
		BoundRight:  binary.LittleEndian.Uint32(data[24:28]),
		BoundBottom: binary.LittleEndian.Uint32(data[28:32]),
		Padding:     binary.LittleEndian.Uint32(data[32:36]),
	}, true
}

func ambientViewingFromPacketSideData(data []byte) (h264.H2645SEIAmbientViewingEnvironment, bool) {
	const (
		ambientViewingStructSize = 3 * avRationalSize
		illuminanceDen           = 10000
		chromaDen                = 50000
	)
	if len(data) < ambientViewingStructSize {
		return h264.H2645SEIAmbientViewingEnvironment{}, false
	}
	illuminance, ok := avRationalToScaledUint32(data, 0, illuminanceDen)
	if !ok {
		return h264.H2645SEIAmbientViewingEnvironment{}, false
	}
	x, ok := avRationalToScaledUint32(data, avRationalSize, chromaDen)
	if !ok || x > 0xffff {
		return h264.H2645SEIAmbientViewingEnvironment{}, false
	}
	y, ok := avRationalToScaledUint32(data, 2*avRationalSize, chromaDen)
	if !ok || y > 0xffff {
		return h264.H2645SEIAmbientViewingEnvironment{}, false
	}
	return h264.H2645SEIAmbientViewingEnvironment{
		Present:            1,
		AmbientIlluminance: illuminance,
		AmbientLightX:      uint16(x),
		AmbientLightY:      uint16(y),
	}, true
}

func masteringDisplayFromPacketSideData(data []byte) (h264.AVMasteringDisplayMetadata, bool) {
	const (
		masteringDisplayStructSize = 88
		chromaDen                  = 50000
		lumaDen                    = 10000
		hasPrimariesOffset         = 80
		hasLuminanceOffset         = 84
	)
	if len(data) < masteringDisplayStructSize {
		return h264.AVMasteringDisplayMetadata{}, false
	}
	out := h264.AVMasteringDisplayMetadata{
		Present:      1,
		HasPrimaries: int32(binary.LittleEndian.Uint32(data[hasPrimariesOffset : hasPrimariesOffset+4])),
		HasLuminance: int32(binary.LittleEndian.Uint32(data[hasLuminanceOffset : hasLuminanceOffset+4])),
	}
	pos := 0
	if out.HasPrimaries != 0 {
		for i := range out.DisplayPrimaries {
			for j := range out.DisplayPrimaries[i] {
				v, ok := avRationalToScaledUint32(data, pos, chromaDen)
				if !ok || v > 0xffff {
					return h264.AVMasteringDisplayMetadata{}, false
				}
				out.DisplayPrimaries[i][j] = uint16(v)
				pos += avRationalSize
			}
		}
		for i := range out.WhitePoint {
			v, ok := avRationalToScaledUint32(data, pos, chromaDen)
			if !ok || v > 0xffff {
				return h264.AVMasteringDisplayMetadata{}, false
			}
			out.WhitePoint[i] = uint16(v)
			pos += avRationalSize
		}
	} else {
		pos = 8 * avRationalSize
	}
	if out.HasLuminance != 0 {
		minLuminance, ok := avRationalToScaledUint32(data, pos, lumaDen)
		if !ok {
			return h264.AVMasteringDisplayMetadata{}, false
		}
		maxLuminance, ok := avRationalToScaledUint32(data, pos+avRationalSize, lumaDen)
		if !ok {
			return h264.AVMasteringDisplayMetadata{}, false
		}
		out.MinLuminance = minLuminance
		out.MaxLuminance = maxLuminance
	}
	return out, true
}

func contentLightFromPacketSideData(data []byte) (h264.H2645SEIContentLight, bool) {
	if len(data) < 8 {
		return h264.H2645SEIContentLight{}, false
	}
	return h264.H2645SEIContentLight{
		Present:                 1,
		MaxContentLightLevel:    binary.LittleEndian.Uint32(data[:4]),
		MaxPicAverageLightLevel: binary.LittleEndian.Uint32(data[4:8]),
	}, true
}

func referenceDisplaysFromPacketSideData(data []byte) (h264.AV3DReferenceDisplaysInfo, bool) {
	const (
		referenceDisplaysHeaderSize = 24
		referenceDisplayEntrySize   = 12
		maxReferenceDisplays        = 32
	)
	if len(data) < referenceDisplaysHeaderSize {
		return h264.AV3DReferenceDisplaysInfo{}, false
	}
	count := int(data[3])
	entriesOffset := binary.LittleEndian.Uint64(data[8:16])
	entrySize := binary.LittleEndian.Uint64(data[16:24])
	if count < 1 || count > maxReferenceDisplays ||
		entriesOffset > uint64(len(data)) ||
		entrySize < referenceDisplayEntrySize {
		return h264.AV3DReferenceDisplaysInfo{}, false
	}
	if entrySize > (^uint64(0)-entriesOffset)/uint64(count) {
		return h264.AV3DReferenceDisplaysInfo{}, false
	}
	entriesEnd := entriesOffset + uint64(count)*entrySize
	if entriesEnd > uint64(len(data)) {
		return h264.AV3DReferenceDisplaysInfo{}, false
	}
	out := h264.AV3DReferenceDisplaysInfo{
		Present:                1,
		PrecRefDisplayWidth:    data[0],
		RefViewingDistanceFlag: data[1],
		PrecRefViewingDist:     data[2],
		Displays:               make([]h264.AV3DReferenceDisplay, count),
	}
	for i := 0; i < count; i++ {
		off := int(entriesOffset + uint64(i)*entrySize)
		out.Displays[i] = h264.AV3DReferenceDisplay{
			LeftViewID:                 binary.LittleEndian.Uint16(data[off : off+2]),
			RightViewID:                binary.LittleEndian.Uint16(data[off+2 : off+4]),
			ExponentRefDisplayWidth:    data[off+4],
			MantissaRefDisplayWidth:    data[off+5],
			ExponentRefViewingDistance: data[off+6],
			MantissaRefViewingDistance: data[off+7],
			AdditionalShiftPresentFlag: data[off+8],
			NumSampleShift:             int16(binary.LittleEndian.Uint16(data[off+10 : off+12])),
		}
	}
	return out, true
}

const avRationalSize = 8

func avRationalToScaledUint32(data []byte, off int, scale int64) (uint32, bool) {
	if off < 0 || off+avRationalSize > len(data) || scale <= 0 {
		return 0, false
	}
	num := int64(int32(binary.LittleEndian.Uint32(data[off : off+4])))
	den := int64(int32(binary.LittleEndian.Uint32(data[off+4 : off+8])))
	if num < 0 || den <= 0 {
		return 0, false
	}
	scaled := num * scale
	if scaled%den != 0 {
		return 0, false
	}
	value := scaled / den
	if value < 0 || value > int64(^uint32(0)) {
		return 0, false
	}
	return uint32(value), true
}

// AppendRawYUV appends the visible frame as 8-bit planar YUV.
//
// The output order is Y, Cb, then Cr. The returned slice may share backing
// storage with dst; keep dst unchanged while using the returned bytes.
// High-bit-depth frames return ErrUnsupported.
func (f *Frame) AppendRawYUV(dst []byte) ([]byte, error) {
	if f == nil || f.Width <= 0 || f.Height <= 0 {
		return dst, ErrInvalidData
	}
	if f.BitDepthLuma != 8 || (f.ChromaFormatIDC != 0 && f.BitDepthChroma != 8) {
		return dst, ErrUnsupported
	}
	size, err := f.RawYUVSize()
	if err != nil {
		return dst, err
	}
	if _, err := checkedAddInt(len(dst), size); err != nil {
		return dst, err
	}
	return f.appendRawYUVBytes8(dst)
}

// BytesPerSample returns the raw-output byte width for each visible sample.
func (f *Frame) BytesPerSample() (int, error) {
	depth, err := f.rawBitDepth()
	if err != nil {
		return 0, err
	}
	if depth == 8 {
		return 1, nil
	}
	return 2, nil
}

// RawPixelFormat returns the FFmpeg-style rawvideo pixel format for the frame.
func (f *Frame) RawPixelFormat() (string, error) {
	depth, err := f.rawBitDepth()
	if err != nil {
		return "", err
	}
	base := ""
	if depth == 8 && f.VideoFullRangeFlag == 1 {
		switch f.ChromaFormatIDC {
		case 0, 1:
			return "yuvj420p", nil
		case 2:
			return "yuvj422p", nil
		case 3:
			return "yuvj444p", nil
		default:
			return "", ErrInvalidData
		}
	}
	switch f.ChromaFormatIDC {
	case 0, 1:
		base = "yuv420p"
	case 2:
		base = "yuv422p"
	case 3:
		base = "yuv444p"
	default:
		return "", ErrInvalidData
	}
	return base + rawBitDepthSuffix(depth), nil
}

// RawYUVSize returns the byte count produced by AppendRawYUVBytesLE.
func (f *Frame) RawYUVSize() (int, error) {
	samples, err := f.rawYUVSampleCount()
	if err != nil {
		return 0, err
	}
	bytesPerSample, err := f.BytesPerSample()
	if err != nil {
		return 0, err
	}
	return checkedMulInt(samples, bytesPerSample)
}

// AppendRawYUV16 appends the visible high-bit-depth frame as planar uint16 YUV.
//
// The output order is Y, Cb, then Cr. The returned slice may share backing
// storage with dst; keep dst unchanged while using the returned samples. 8-bit
// frames return ErrUnsupported.
func (f *Frame) AppendRawYUV16(dst []uint16) ([]uint16, error) {
	depth, err := f.rawBitDepth()
	if err != nil {
		return dst, err
	}
	if depth == 8 {
		return dst, ErrUnsupported
	}
	samples, err := f.rawYUVSampleCount()
	if err != nil {
		return dst, err
	}
	if _, err := checkedAddInt(len(dst), samples); err != nil {
		return dst, err
	}
	chromaWidth, chromaHeight, chromaCropLeft, chromaCropTop, err := f.rawYUV16Geometry()
	if err != nil {
		return dst, err
	}
	maxSample := maxRawSample(depth)
	if err := f.validateRawYUV16Samples(chromaWidth, chromaHeight, chromaCropLeft, chromaCropTop, maxSample); err != nil {
		return dst, err
	}
	for y := 0; y < f.Height; y++ {
		row := (f.CropTop+y)*f.YStride + f.CropLeft
		dst, err = appendRawUint16Samples(dst, f.Y16[row:row+f.Width], maxSample)
		if err != nil {
			return dst, err
		}
	}
	if f.ChromaFormatIDC == 0 {
		chromaWidth, chromaHeight, err := frameChromaSize(f.Width, f.Height, 1)
		if err != nil {
			return dst, err
		}
		return appendNeutralRawUint16Samples(dst, chromaWidth*chromaHeight*2, neutralRawChromaSample(depth)), nil
	}
	if chromaWidth == 0 || chromaHeight == 0 {
		return dst, nil
	}
	for y := 0; y < chromaHeight; y++ {
		row := (chromaCropTop+y)*f.CStride + chromaCropLeft
		dst, err = appendRawUint16Samples(dst, f.Cb16[row:row+chromaWidth], maxSample)
		if err != nil {
			return dst, err
		}
	}
	for y := 0; y < chromaHeight; y++ {
		row := (chromaCropTop+y)*f.CStride + chromaCropLeft
		dst, err = appendRawUint16Samples(dst, f.Cr16[row:row+chromaWidth], maxSample)
		if err != nil {
			return dst, err
		}
	}
	return dst, nil
}

// AppendRawYUVBytesLE appends the visible frame as planar YUV bytes.
//
// The output order is Y, Cb, then Cr. 8-bit frames use one byte per sample;
// high-bit-depth frames use little-endian uint16 samples. The returned slice may
// share backing storage with dst; keep dst unchanged while using the returned
// bytes.
func (f *Frame) AppendRawYUVBytesLE(dst []byte) ([]byte, error) {
	depth, err := f.rawBitDepth()
	if err != nil {
		return dst, err
	}
	size, err := f.RawYUVSize()
	if err != nil {
		return dst, err
	}
	if _, err := checkedAddInt(len(dst), size); err != nil {
		return dst, err
	}
	if depth == 8 {
		return f.appendRawYUVBytes8(dst)
	}
	chromaWidth, chromaHeight, chromaCropLeft, chromaCropTop, err := f.rawYUV16Geometry()
	if err != nil {
		return dst, err
	}
	maxSample := maxRawSample(depth)
	if err := f.validateRawYUV16Samples(chromaWidth, chromaHeight, chromaCropLeft, chromaCropTop, maxSample); err != nil {
		return dst, err
	}
	for y := 0; y < f.Height; y++ {
		row := (f.CropTop+y)*f.YStride + f.CropLeft
		dst, err = appendRawUint16LE(dst, f.Y16[row:row+f.Width], maxSample)
		if err != nil {
			return dst, err
		}
	}
	if f.ChromaFormatIDC == 0 {
		chromaWidth, chromaHeight, err := frameChromaSize(f.Width, f.Height, 1)
		if err != nil {
			return dst, err
		}
		return appendNeutralRawUint16LE(dst, chromaWidth*chromaHeight*2, neutralRawChromaSample(depth)), nil
	}
	if chromaWidth == 0 || chromaHeight == 0 {
		return dst, nil
	}
	for y := 0; y < chromaHeight; y++ {
		row := (chromaCropTop+y)*f.CStride + chromaCropLeft
		dst, err = appendRawUint16LE(dst, f.Cb16[row:row+chromaWidth], maxSample)
		if err != nil {
			return dst, err
		}
	}
	for y := 0; y < chromaHeight; y++ {
		row := (chromaCropTop+y)*f.CStride + chromaCropLeft
		dst, err = appendRawUint16LE(dst, f.Cr16[row:row+chromaWidth], maxSample)
		if err != nil {
			return dst, err
		}
	}
	return dst, nil
}

func (f *Frame) appendRawYUVBytes8(dst []byte) ([]byte, error) {
	if f == nil || f.Width <= 0 || f.Height <= 0 {
		return dst, ErrInvalidData
	}
	if !framePlaneHasVisibleRect(len(f.Y), f.YStride, f.CropLeft, f.CropTop, f.Width, f.Height) {
		return dst, ErrInvalidData
	}
	chromaWidth := 0
	chromaHeight := 0
	chromaCropLeft := 0
	chromaCropTop := 0
	if f.ChromaFormatIDC == 0 {
		var err error
		chromaWidth, chromaHeight, err = frameChromaSize(f.Width, f.Height, 1)
		if err != nil {
			return dst, err
		}
	} else {
		var err error
		chromaWidth, chromaHeight, err = frameChromaSize(f.Width, f.Height, f.ChromaFormatIDC)
		if err != nil {
			return dst, err
		}
		if chromaWidth != 0 && chromaHeight != 0 {
			chromaCropLeft, chromaCropTop, err = frameChromaCrop(f.CropLeft, f.CropTop, f.ChromaFormatIDC)
			if err != nil {
				return dst, err
			}
			if !framePlaneHasVisibleRect(len(f.Cb), f.CStride, chromaCropLeft, chromaCropTop, chromaWidth, chromaHeight) ||
				!framePlaneHasVisibleRect(len(f.Cr), f.CStride, chromaCropLeft, chromaCropTop, chromaWidth, chromaHeight) {
				return dst, ErrInvalidData
			}
		}
	}
	for y := 0; y < f.Height; y++ {
		row := (f.CropTop+y)*f.YStride + f.CropLeft
		dst = append(dst, f.Y[row:row+f.Width]...)
	}
	if f.ChromaFormatIDC == 0 {
		return appendNeutralRawBytes(dst, chromaWidth*chromaHeight*2, byte(neutralRawChromaSample(8))), nil
	}
	if chromaWidth == 0 || chromaHeight == 0 {
		return dst, nil
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

func (f *Frame) rawBitDepth() (int, error) {
	if f == nil {
		return 0, ErrInvalidData
	}
	switch f.ChromaFormatIDC {
	case 0, 1, 2, 3:
	default:
		return 0, ErrInvalidData
	}
	switch f.BitDepthLuma {
	case 8, 9, 10, 12, 14:
	default:
		if f.BitDepthLuma <= 0 {
			return 0, ErrInvalidData
		}
		return 0, ErrUnsupported
	}
	if f.ChromaFormatIDC != 0 && f.BitDepthChroma != f.BitDepthLuma {
		if f.BitDepthChroma <= 0 {
			return 0, ErrInvalidData
		}
		return 0, ErrUnsupported
	}
	return f.BitDepthLuma, nil
}

func rawBitDepthSuffix(depth int) string {
	switch depth {
	case 9:
		return "9le"
	case 10:
		return "10le"
	case 12:
		return "12le"
	case 14:
		return "14le"
	default:
		return ""
	}
}

func (f *Frame) rawYUVSampleCount() (int, error) {
	if f == nil || f.Width <= 0 || f.Height <= 0 {
		return 0, ErrInvalidData
	}
	if _, err := f.rawBitDepth(); err != nil {
		return 0, err
	}
	chromaWidth, chromaHeight, err := frameChromaSize(f.Width, f.Height, f.ChromaFormatIDC)
	if err != nil {
		return 0, err
	}
	if f.ChromaFormatIDC == 0 {
		chromaWidth, chromaHeight, err = frameChromaSize(f.Width, f.Height, 1)
		if err != nil {
			return 0, err
		}
	}
	lumaSamples, err := checkedMulInt(f.Width, f.Height)
	if err != nil {
		return 0, err
	}
	chromaSamples, err := checkedMulInt(chromaWidth, chromaHeight)
	if err != nil {
		return 0, err
	}
	chromaSamples, err = checkedMulInt(chromaSamples, 2)
	if err != nil {
		return 0, err
	}
	return checkedAddInt(lumaSamples, chromaSamples)
}

func checkedAddInt(a int, b int) (int, error) {
	if a < 0 || b < 0 {
		return 0, ErrInvalidData
	}
	if a > maxInt-b {
		return 0, ErrInvalidData
	}
	return a + b, nil
}

func checkedMulInt(a int, b int) (int, error) {
	if a < 0 || b < 0 {
		return 0, ErrInvalidData
	}
	if a != 0 && b > maxInt/a {
		return 0, ErrInvalidData
	}
	return a * b, nil
}

const maxInt = int(^uint(0) >> 1)

func (f *Frame) rawYUV16Geometry() (int, int, int, int, error) {
	if f == nil || f.Width <= 0 || f.Height <= 0 {
		return 0, 0, 0, 0, ErrInvalidData
	}
	if !framePlaneHasVisibleRect(len(f.Y16), f.YStride, f.CropLeft, f.CropTop, f.Width, f.Height) {
		return 0, 0, 0, 0, ErrInvalidData
	}
	chromaWidth, chromaHeight, err := frameChromaSize(f.Width, f.Height, f.ChromaFormatIDC)
	if err != nil {
		return 0, 0, 0, 0, err
	}
	if chromaWidth == 0 || chromaHeight == 0 {
		return chromaWidth, chromaHeight, 0, 0, nil
	}
	chromaCropLeft, chromaCropTop, err := frameChromaCrop(f.CropLeft, f.CropTop, f.ChromaFormatIDC)
	if err != nil {
		return 0, 0, 0, 0, err
	}
	if !framePlaneHasVisibleRect(len(f.Cb16), f.CStride, chromaCropLeft, chromaCropTop, chromaWidth, chromaHeight) ||
		!framePlaneHasVisibleRect(len(f.Cr16), f.CStride, chromaCropLeft, chromaCropTop, chromaWidth, chromaHeight) {
		return 0, 0, 0, 0, ErrInvalidData
	}
	return chromaWidth, chromaHeight, chromaCropLeft, chromaCropTop, nil
}

func framePlaneHasVisibleRect(planeLen int, stride int, cropLeft int, cropTop int, width int, height int) bool {
	if planeLen < 0 || stride < 0 || cropLeft < 0 || cropTop < 0 || width <= 0 || height <= 0 {
		return false
	}
	minStride, err := checkedAddInt(width, cropLeft)
	if err != nil || stride < minStride {
		return false
	}
	lastRow, err := checkedAddInt(cropTop, height-1)
	if err != nil {
		return false
	}
	offset, err := checkedMulInt(lastRow, stride)
	if err != nil {
		return false
	}
	offset, err = checkedAddInt(offset, cropLeft)
	if err != nil {
		return false
	}
	end, err := checkedAddInt(offset, width)
	if err != nil {
		return false
	}
	return planeLen >= end
}

func (f *Frame) validateRawYUV16Samples(chromaWidth int, chromaHeight int, chromaCropLeft int, chromaCropTop int, maxSample uint16) error {
	for y := 0; y < f.Height; y++ {
		row := (f.CropTop+y)*f.YStride + f.CropLeft
		if !rawUint16SamplesValid(f.Y16[row:row+f.Width], maxSample) {
			return ErrInvalidData
		}
	}
	if f.ChromaFormatIDC == 0 || chromaWidth == 0 || chromaHeight == 0 {
		return nil
	}
	for y := 0; y < chromaHeight; y++ {
		row := (chromaCropTop+y)*f.CStride + chromaCropLeft
		if !rawUint16SamplesValid(f.Cb16[row:row+chromaWidth], maxSample) ||
			!rawUint16SamplesValid(f.Cr16[row:row+chromaWidth], maxSample) {
			return ErrInvalidData
		}
	}
	return nil
}

func rawUint16SamplesValid(samples []uint16, maxSample uint16) bool {
	for _, sample := range samples {
		if sample > maxSample {
			return false
		}
	}
	return true
}

func maxRawSample(depth int) uint16 {
	return uint16((1 << uint(depth)) - 1)
}

func appendRawUint16Samples(dst []uint16, samples []uint16, maxSample uint16) ([]uint16, error) {
	for _, sample := range samples {
		if sample > maxSample {
			return dst, ErrInvalidData
		}
		dst = append(dst, sample)
	}
	return dst, nil
}

func neutralRawChromaSample(depth int) uint16 {
	return uint16(1 << uint(depth-1))
}

func appendNeutralRawBytes(dst []byte, count int, sample byte) []byte {
	for i := 0; i < count; i++ {
		dst = append(dst, sample)
	}
	return dst
}

func appendNeutralRawUint16Samples(dst []uint16, count int, sample uint16) []uint16 {
	for i := 0; i < count; i++ {
		dst = append(dst, sample)
	}
	return dst
}

func appendNeutralRawUint16LE(dst []byte, count int, sample uint16) []byte {
	for i := 0; i < count; i++ {
		dst = append(dst, byte(sample), byte(sample>>8))
	}
	return dst
}

func appendRawUint16LE(dst []byte, samples []uint16, maxSample uint16) ([]byte, error) {
	for _, sample := range samples {
		if sample > maxSample {
			return dst, ErrInvalidData
		}
		dst = append(dst, byte(sample), byte(sample>>8))
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
		KeyFrame:                       src.KeyFrame,
		YStride:                        src.LumaStride,
		CStride:                        src.ChromaStride,
		Y:                              cloneByteSlice(src.Y),
		Cb:                             cloneByteSlice(src.Cb),
		Cr:                             cloneByteSlice(src.Cr),
		Y16:                            cloneUint16Slice(src.Y16),
		Cb16:                           cloneUint16Slice(src.Cb16),
		Cr16:                           cloneUint16Slice(src.Cr16),
		SideData:                       frameSideDataFromH264(src.SideData, src.TimeScale, src.NumUnitsInTick),
	}
}

func frameSideDataFromH264(src h264.DecodedFrameSideData, timeScale uint32, numUnitsInTick uint32) FrameSideData {
	out := FrameSideData{
		UserDataUnregistered: cloneByteSlices(src.UserDataUnregistered),
		A53ClosedCaptions:    cloneByteSlice(src.A53ClosedCaptions),
		X264Build:            int(src.X264Build),
		S12MTimecodes:        cloneUint32Slice(src.S12MTimecodes),
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
		if src.PictureTiming.TimecodeCount != 0 {
			out.S12MTimecodes = s12mTimecodesFromPictureTiming(src.PictureTiming, timeScale, numUnitsInTick, src.X264Build)
		}
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
	if src.Stereo3D.Present != 0 {
		out.Stereo3D = stereo3DFromPacketSideDataValue(src.Stereo3D)
	}
	if src.Spherical.Present != 0 {
		out.Spherical = sphericalFromPacketSideDataValue(src.Spherical)
	}
	if src.DisplayMatrix.Present != 0 {
		out.DisplayOrientation = displayOrientationFromPacketMatrix(src.DisplayMatrix)
	}
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
	if src.MasteringMetadata.Present != 0 {
		out.MasteringDisplay = masteringDisplayFromPacketMetadata(src.MasteringMetadata)
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
	if len(src.ICCProfile) != 0 {
		out.ICCProfile = cloneByteSlice(src.ICCProfile)
	}
	if len(src.DynamicHDR10Plus) != 0 {
		out.DynamicHDR10Plus = cloneByteSlice(src.DynamicHDR10Plus)
	}
	if len(src.LCEVC) != 0 {
		out.LCEVC = cloneByteSlice(src.LCEVC)
	}
	if src.ReferenceDisplays.Present != 0 {
		out.ReferenceDisplays = referenceDisplaysFromPacketSideDataValue(src.ReferenceDisplays)
	}
	return out
}

func displayOrientationFromPacketMatrix(src h264.AVDisplayMatrix) *DisplayOrientation {
	return &DisplayOrientation{Matrix: src.Matrix}
}

func sphericalFromPacketSideDataValue(src h264.AVSphericalMapping) *SphericalMapping {
	if src.Present == 0 || src.Projection < 0 || src.Projection > int32(SphericalProjectionParametricImmersive) {
		return nil
	}
	return &SphericalMapping{
		Projection:  SphericalProjection(src.Projection),
		Yaw:         src.Yaw,
		Pitch:       src.Pitch,
		Roll:        src.Roll,
		BoundLeft:   src.BoundLeft,
		BoundTop:    src.BoundTop,
		BoundRight:  src.BoundRight,
		BoundBottom: src.BoundBottom,
		Padding:     src.Padding,
	}
}

func masteringDisplayFromPacketMetadata(src h264.AVMasteringDisplayMetadata) *MasteringDisplay {
	return &MasteringDisplay{
		DisplayPrimaries: src.DisplayPrimaries,
		WhitePoint:       src.WhitePoint,
		MaxLuminance:     src.MaxLuminance,
		MinLuminance:     src.MinLuminance,
		HasPrimaries:     src.HasPrimaries != 0,
		HasLuminance:     src.HasLuminance != 0,
	}
}

func referenceDisplaysFromPacketSideDataValue(src h264.AV3DReferenceDisplaysInfo) *ReferenceDisplaysInfo {
	if len(src.Displays) > maxInt/16 {
		return nil
	}
	out := &ReferenceDisplaysInfo{
		PrecRefDisplayWidth:    src.PrecRefDisplayWidth,
		RefViewingDistanceFlag: src.RefViewingDistanceFlag != 0,
		PrecRefViewingDist:     src.PrecRefViewingDist,
		Displays:               make([]ReferenceDisplay, len(src.Displays)),
	}
	for i, display := range src.Displays {
		out.Displays[i] = ReferenceDisplay{
			LeftViewID:                 display.LeftViewID,
			RightViewID:                display.RightViewID,
			ExponentRefDisplayWidth:    display.ExponentRefDisplayWidth,
			MantissaRefDisplayWidth:    display.MantissaRefDisplayWidth,
			ExponentRefViewingDistance: display.ExponentRefViewingDistance,
			MantissaRefViewingDistance: display.MantissaRefViewingDistance,
			AdditionalShiftPresentFlag: display.AdditionalShiftPresentFlag != 0,
			NumSampleShift:             display.NumSampleShift,
		}
	}
	return out
}

func s12mTimecodesFromPictureTiming(src h264.H264SEIPictureTiming, timeScale uint32, numUnitsInTick uint32, x264Build int32) []uint32 {
	count := int(src.TimecodeCount)
	if count <= 0 {
		return nil
	}
	if count > len(src.Timecode) {
		count = len(src.Timecode)
	}
	rateNum, rateDen := h264TimecodeFrameRate(timeScale, numUnitsInTick, x264Build)
	out := make([]uint32, count)
	for i := 0; i < count; i++ {
		tc := src.Timecode[i]
		out[i] = avTimecodeGetSMPTE(rateNum, rateDen, tc.DropFrame != 0, tc.Hours, tc.Minutes, tc.Seconds, tc.Frame)
	}
	return out
}

func h264TimecodeFrameRate(timeScale uint32, numUnitsInTick uint32, x264Build int32) (int64, int64) {
	if timeScale == 0 || numUnitsInTick == 0 {
		return 0, 1
	}
	num := uint64(timeScale)
	den := uint64(numUnitsInTick) * 2
	if x264Build >= 0 && x264Build < 44 {
		den *= 2
	}
	g := gcdUint64(num, den)
	return int64(num / g), int64(den / g)
}

func avTimecodeGetSMPTE(rateNum int64, rateDen int64, drop bool, hh int32, mm int32, ss int32, ff int32) uint32 {
	var tc uint32
	if cmpRational(rateNum, rateDen, 30, 1) > 0 {
		if ff%2 == 1 {
			if cmpRational(rateNum, rateDen, 50, 1) == 0 {
				tc |= 1 << 7
			} else {
				tc |= 1 << 23
			}
		}
		ff /= 2
	}

	hh %= 24
	mm = clipInt32(mm, 0, 59)
	ss = clipInt32(ss, 0, 59)
	ff %= 40

	if drop {
		tc |= 1 << 30
	}
	tc |= uint32(ff/10) << 28
	tc |= uint32(ff%10) << 24
	tc |= uint32(ss/10) << 20
	tc |= uint32(ss%10) << 16
	tc |= uint32(mm/10) << 12
	tc |= uint32(mm%10) << 8
	tc |= uint32(hh/10) << 4
	tc |= uint32(hh % 10)
	return tc
}

func cmpRational(aNum int64, aDen int64, bNum int64, bDen int64) int {
	if aDen == 0 && bDen == 0 {
		return 0
	}
	if aDen == 0 {
		if aNum < 0 {
			return -1
		}
		return 1
	}
	if bDen == 0 {
		if bNum < 0 {
			return 1
		}
		return -1
	}
	lhs := aNum * bDen
	rhs := bNum * aDen
	switch {
	case lhs < rhs:
		return -1
	case lhs > rhs:
		return 1
	default:
		return 0
	}
}

func clipInt32(v int32, min int32, max int32) int32 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func gcdUint64(a uint64, b uint64) uint64 {
	for b != 0 {
		a, b = b, a%b
	}
	if a == 0 {
		return 1
	}
	return a
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

func stereo3DFromPacketSideDataValue(src h264.AVStereo3D) *Stereo3D {
	if src.Present == 0 || src.Type < 0 || src.Type > int32(Stereo3DTypeUnspecified) ||
		src.View < 0 || src.View > int32(Stereo3DViewUnspecified) ||
		src.PrimaryEye < 0 || src.PrimaryEye > int32(Stereo3DPrimaryEyeRight) {
		return nil
	}
	out := &Stereo3D{
		Type:                          Stereo3DType(src.Type),
		Inverted:                      src.Flags&1 != 0,
		View:                          Stereo3DView(src.View),
		PrimaryEye:                    Stereo3DPrimaryEye(src.PrimaryEye),
		Baseline:                      src.Baseline,
		HorizontalDisparityAdjustment: rationalFromH264(src.HorizontalDisparityAdjustment),
		HorizontalFieldOfView:         rationalFromH264(src.HorizontalFieldOfView),
	}
	out.StereoMode = stereo3DMode(out.Type, out.Inverted)
	return out
}

func rationalFromH264(src h264.AVRational) Rational {
	return Rational{Num: src.Num, Den: src.Den}
}

func stereo3DMode(typ Stereo3DType, inverted bool) string {
	switch typ {
	case Stereo3DType2D:
		return "mono"
	case Stereo3DTypeSideBySide, Stereo3DTypeSideBySideQuincunx:
		if inverted {
			return "right_left"
		}
		return "left_right"
	case Stereo3DTypeTopBottom:
		if inverted {
			return "bottom_top"
		}
		return "top_bottom"
	case Stereo3DTypeFrameSequence:
		if inverted {
			return "block_rl"
		}
		return "block_lr"
	case Stereo3DTypeCheckerboard:
		if inverted {
			return "checkerboard_rl"
		}
		return "checkerboard_lr"
	case Stereo3DTypeLines:
		if inverted {
			return "row_interleaved_rl"
		}
		return "row_interleaved_lr"
	case Stereo3DTypeColumns:
		if inverted {
			return "col_interleaved_rl"
		}
		return "col_interleaved_lr"
	default:
		return ""
	}
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
	if len(src) == 0 || len(src) > maxInt/32 {
		return nil
	}
	out := make([][]byte, len(src))
	for i := range src {
		out[i] = cloneByteSlice(src[i])
	}
	return out
}

func cloneUint32Slice(src []uint32) []uint32 {
	if len(src) == 0 || len(src) > maxInt/4 {
		return nil
	}
	return append([]uint32(nil), src...)
}

func cloneByteSlice(src []byte) []byte {
	if len(src) == 0 || len(src) > maxInt/2 {
		return nil
	}
	return append([]byte(nil), src...)
}

func cloneUint16Slice(src []uint16) []uint16 {
	if len(src) == 0 || len(src) > maxInt/2 {
		return nil
	}
	return append([]uint16(nil), src...)
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
