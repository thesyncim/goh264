// SPDX-License-Identifier: LGPL-2.1-or-later

package goh264

import "github.com/thesyncim/goh264/internal/h264"

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
	SPSID           uint32
	ProfileIDC      uint8
	Profile         string
	LevelIDC        uint8
	Width           int
	Height          int
	ChromaFormatIDC uint32
	BitDepthLuma    int
	BitDepthChroma  int
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
	X264Build            int
	PictureTiming        *PictureTiming
	RecoveryPoint        *RecoveryPoint
	BufferingPeriod      *BufferingPeriod
	GreenMetadata        *GreenMetadata
	FramePacking         *FramePackingArrangement
	DisplayOrientation   *DisplayOrientation
	AlternativeTransfer  *AlternativeTransfer
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

type FramePackingArrangement struct {
	ArrangementID               uint32
	ArrangementCancelFlag       bool
	ArrangementType             int32
	ArrangementRepetitionPeriod uint32
	ContentInterpretationType   int32
	QuincunxSamplingFlag        bool
	CurrentFrameIsFrame0Flag    bool
}

type DisplayOrientation struct {
	AnticlockwiseRotation int32
	HFlip                 bool
	VFlip                 bool
}

type AlternativeTransfer struct {
	PreferredTransferCharacteristics int32
}

type MasteringDisplay struct {
	DisplayPrimaries [3][2]uint16
	WhitePoint       [2]uint16
	MaxLuminance     uint32
	MinLuminance     uint32
}

type ContentLight struct {
	MaxContentLightLevel    uint16
	MaxPicAverageLightLevel uint16
}

type Frame struct {
	Width           int
	Height          int
	CropLeft        int
	CropTop         int
	ChromaFormatIDC uint32
	BitDepthLuma    int
	BitDepthChroma  int
	YStride         int
	CStride         int
	Y               []byte
	Cb              []byte
	Cr              []byte
	SideData        FrameSideData
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
		Width:           src.Width,
		Height:          src.Height,
		CropLeft:        src.CropLeft,
		CropTop:         src.CropTop,
		ChromaFormatIDC: uint32(src.ChromaFormatIDC),
		BitDepthLuma:    src.BitDepthLuma,
		BitDepthChroma:  src.BitDepthChroma,
		YStride:         src.LumaStride,
		CStride:         src.ChromaStride,
		Y:               src.Y,
		Cb:              src.Cb,
		Cr:              src.Cr,
		SideData:        frameSideDataFromH264(src.SideData),
	}
}

func frameSideDataFromH264(src h264.DecodedFrameSideData) FrameSideData {
	out := FrameSideData{
		UserDataUnregistered: cloneByteSlices(src.UserDataUnregistered),
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
	}
	if src.DisplayOrientation.Present != 0 {
		out.DisplayOrientation = &DisplayOrientation{
			AnticlockwiseRotation: src.DisplayOrientation.AnticlockwiseRotation,
			HFlip:                 src.DisplayOrientation.HFlip != 0,
			VFlip:                 src.DisplayOrientation.VFlip != 0,
		}
	}
	if src.AlternativeTransfer.Present != 0 {
		out.AlternativeTransfer = &AlternativeTransfer{
			PreferredTransferCharacteristics: src.AlternativeTransfer.PreferredTransferCharacteristics,
		}
	}
	if src.MasteringDisplay.Present != 0 {
		out.MasteringDisplay = &MasteringDisplay{
			DisplayPrimaries: src.MasteringDisplay.DisplayPrimaries,
			WhitePoint:       src.MasteringDisplay.WhitePoint,
			MaxLuminance:     src.MasteringDisplay.MaxLuminance,
			MinLuminance:     src.MasteringDisplay.MinLuminance,
		}
	}
	if src.ContentLight.Present != 0 {
		out.ContentLight = &ContentLight{
			MaxContentLightLevel:    src.ContentLight.MaxContentLightLevel,
			MaxPicAverageLightLevel: src.ContentLight.MaxPicAverageLightLevel,
		}
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
		SPSID:           sps.SPSID,
		ProfileIDC:      profileIDC,
		Profile:         profileName(profileIDC, uint8(sps.ConstraintSetFlags)),
		LevelIDC:        uint8(sps.LevelIDC),
		Width:           int(sps.Width),
		Height:          int(sps.Height),
		ChromaFormatIDC: sps.ChromaFormatIDC,
		BitDepthLuma:    int(sps.BitDepthLuma),
		BitDepthChroma:  int(sps.BitDepthChroma),
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
