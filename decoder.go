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
}

func NewDecoder() *Decoder {
	return &Decoder{}
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
	}
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
