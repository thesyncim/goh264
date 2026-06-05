// SPDX-License-Identifier: LGPL-2.1-or-later

package goh264

import (
	"bytes"
	"fmt"

	"github.com/thesyncim/goh264/internal/h264"
)

type EncoderPixelFormat uint8

const (
	EncoderPixelFormatI420 EncoderPixelFormat = iota + 1
)

type EncoderProfile uint8

const (
	EncoderProfileConstrainedBaseline EncoderProfile = iota + 1
	EncoderProfileBaseline
	EncoderProfileMain
	EncoderProfileHigh
)

type EncoderEntropyMode uint8

const (
	EncoderEntropyCAVLC EncoderEntropyMode = iota + 1
	EncoderEntropyCABAC
)

type EncoderDeblockMode uint8

const (
	EncoderDeblockEnabled EncoderDeblockMode = iota + 1
	EncoderDeblockDisabled
	EncoderDeblockSliceBoundary
)

type EncoderSPSPPSMode uint8

const (
	EncoderSPSPPSInBandKeyframes EncoderSPSPPSMode = iota + 1
	EncoderSPSPPSOutOfBand
	EncoderSPSPPSEveryIDR
)

type EncoderRateControlMode uint8

const (
	EncoderRateControlCBR EncoderRateControlMode = iota + 1
	EncoderRateControlVBR
	EncoderRateControlConstantQP
)

type EncoderPreset uint8

const (
	EncoderPresetRealtime EncoderPreset = iota + 1
	EncoderPresetBalanced
	EncoderPresetQuality
)

type EncoderFrameDropMode uint8

const (
	EncoderFrameDropDisabled EncoderFrameDropMode = iota + 1
	EncoderFrameDropLate
	EncoderFrameDropToBitrate
)

type EncoderOutputFormat uint8

const (
	EncoderOutputAnnexB EncoderOutputFormat = iota + 1
	EncoderOutputAVC
	EncoderOutputRTP
)

type EncoderRTPPacketizationMode uint8

const (
	EncoderRTPPacketizationSingleNAL      EncoderRTPPacketizationMode = 0
	EncoderRTPPacketizationNonInterleaved EncoderRTPPacketizationMode = 1
)

type EncoderCrop struct {
	Left   int
	Right  int
	Top    int
	Bottom int
}

type EncoderColorConfig struct {
	SARNum                         int32
	SARDen                         int32
	VideoFormat                    int32
	FullRange                      bool
	ColorPrimaries                 int32
	ColorTransfer                  int32
	ColorMatrix                    int32
	ChromaSampleLocTypeTopField    int32
	ChromaSampleLocTypeBottomField int32
}

type EncoderConfig struct {
	Width        int
	Height       int
	StrideY      int
	StrideCb     int
	StrideCr     int
	PixelFormat  EncoderPixelFormat
	Crop         EncoderCrop
	FrameRateNum int
	FrameRateDen int
	TimeBaseNum  int
	TimeBaseDen  int
	Color        EncoderColorConfig

	Profile            EncoderProfile
	LevelIDC           uint8
	EntropyMode        EncoderEntropyMode
	DeblockMode        EncoderDeblockMode
	Transform8x8       bool
	MaxReferenceFrames int
	BFrames            int
	SPSPPSMode         EncoderSPSPPSMode

	RateControl   EncoderRateControlMode
	TargetBitrate int
	MaxBitrate    int
	VBVBufferSize int
	MaxFrameSize  int
	InitialQP     int
	MinQP         int
	MaxQP         int
	Preset        EncoderPreset

	ZeroLookahead   bool
	FrameDrop       EncoderFrameDropMode
	MaxEncodeTimeUS int
	SliceCount      int
	SliceMaxBytes   int
	Workers         int
	Deterministic   bool

	GOPSize          int
	IDRInterval      int
	SPSPPSBeforeIDR  bool
	RecoveryPointSEI bool
	IntraRefresh     bool

	OutputFormat          EncoderOutputFormat
	RTPMaxPayloadSize     int
	RTPPacketizationMode  EncoderRTPPacketizationMode
	STAPA                 bool
	DONDisabled           bool
	RTPPayloadType        uint8
	RTPSSRC               uint32
	RTPTimestampIncrement uint32
}

type EncoderFrame struct {
	Y        []byte
	Cb       []byte
	Cr       []byte
	StrideY  int
	StrideCb int
	StrideCr int
	Width    int
	Height   int
	PTS      int64
	Duration int64
	ForceIDR bool
	Color    EncoderColorConfig
}

type EncoderNALUnit struct {
	Type         uint8
	Offset       int
	Size         int
	KeyFrame     bool
	ParameterSet bool
}

type EncoderRTPPacket struct {
	Data           []byte
	Payload        []byte
	PayloadType    uint8
	SequenceNumber uint16
	Timestamp      uint32
	SSRC           uint32
	Marker         bool
}

type EncodedFrame struct {
	Data       []byte
	NALUnits   []EncoderNALUnit
	RTPPackets []EncoderRTPPacket
	KeyFrame   bool
	IDR        bool
	PTS        int64
	DTS        int64
	RTPTime    uint32
}

type EncoderParameterSets struct {
	SPS                           []byte
	PPS                           []byte
	AnnexB                        []byte
	AVCDecoderConfigurationRecord []byte
}

type EncoderSEI struct {
	NAL    []byte
	AnnexB []byte
	AVC    []byte
}

type EncoderReconfigure struct {
	TargetBitrate     int
	MaxBitrate        int
	FrameRateNum      int
	FrameRateDen      int
	Width             int
	Height            int
	RTPMaxPayloadSize int
	MaxFrameSize      int
	MaxEncodeTimeUS   int
	Preset            EncoderPreset
	ForceIDR          bool
	SPSPPSBeforeIDR   *bool
	RecoveryPointSEI  *bool
}

type Encoder struct {
	cfg               EncoderConfig
	forceIDR          bool
	frameNum          uint32
	idrPicID          uint32
	rtpSequenceNumber uint16
	reference         encoderReferenceFrame
	framesSinceIDR    int
}

func DefaultEncoderConfig(width, height int) EncoderConfig {
	return EncoderConfig{
		Width:                 width,
		Height:                height,
		StrideY:               width,
		StrideCb:              (width + 1) / 2,
		StrideCr:              (width + 1) / 2,
		PixelFormat:           EncoderPixelFormatI420,
		FrameRateNum:          30,
		FrameRateDen:          1,
		TimeBaseNum:           1,
		TimeBaseDen:           90000,
		Profile:               EncoderProfileConstrainedBaseline,
		LevelIDC:              31,
		EntropyMode:           EncoderEntropyCAVLC,
		DeblockMode:           EncoderDeblockEnabled,
		MaxReferenceFrames:    1,
		SPSPPSMode:            EncoderSPSPPSInBandKeyframes,
		RateControl:           EncoderRateControlCBR,
		TargetBitrate:         1_000_000,
		MaxBitrate:            1_000_000,
		VBVBufferSize:         1_000_000,
		InitialQP:             26,
		MinQP:                 10,
		MaxQP:                 42,
		Preset:                EncoderPresetRealtime,
		ZeroLookahead:         true,
		FrameDrop:             EncoderFrameDropToBitrate,
		MaxEncodeTimeUS:       10_000,
		SliceCount:            1,
		Workers:               1,
		Deterministic:         true,
		GOPSize:               60,
		IDRInterval:           60,
		SPSPPSBeforeIDR:       true,
		RecoveryPointSEI:      true,
		OutputFormat:          EncoderOutputRTP,
		RTPMaxPayloadSize:     1200,
		RTPPacketizationMode:  EncoderRTPPacketizationNonInterleaved,
		DONDisabled:           true,
		RTPPayloadType:        96,
		RTPTimestampIncrement: 3000,
	}
}

func NewEncoder(cfg EncoderConfig) (*Encoder, error) {
	normalized, err := normalizeEncoderConfig(cfg)
	if err != nil {
		return nil, err
	}
	return &Encoder{cfg: normalized}, nil
}

func (cfg EncoderConfig) Validate() error {
	_, err := normalizeEncoderConfig(cfg)
	return err
}

func (e *Encoder) Config() EncoderConfig {
	if e == nil {
		return EncoderConfig{}
	}
	return e.cfg
}

func (e *Encoder) ParameterSets() (EncoderParameterSets, error) {
	if e == nil {
		return EncoderParameterSets{}, encoderInvalid("nil encoder")
	}
	profileIDC, constraintFlags, err := encoderProfileSyntax(e.cfg.Profile)
	if err != nil {
		return EncoderParameterSets{}, err
	}
	sets, err := h264.BuildEncoderParameterSets(h264.EncoderParameterSetConfig{
		ProfileIDC:                     profileIDC,
		ConstraintSetFlags:             constraintFlags,
		LevelIDC:                       e.cfg.LevelIDC,
		Width:                          e.cfg.Width,
		Height:                         e.cfg.Height,
		FrameRateNum:                   e.cfg.FrameRateNum,
		FrameRateDen:                   e.cfg.FrameRateDen,
		MaxReferenceFrames:             uint32(e.cfg.MaxReferenceFrames),
		InitialQP:                      e.cfg.InitialQP,
		SARNum:                         e.cfg.Color.SARNum,
		SARDen:                         e.cfg.Color.SARDen,
		VideoFormat:                    e.cfg.Color.VideoFormat,
		FullRange:                      e.cfg.Color.FullRange,
		ColorPrimaries:                 e.cfg.Color.ColorPrimaries,
		ColorTransfer:                  e.cfg.Color.ColorTransfer,
		ColorMatrix:                    e.cfg.Color.ColorMatrix,
		ChromaSampleLocTypeTopField:    e.cfg.Color.ChromaSampleLocTypeTopField,
		ChromaSampleLocTypeBottomField: e.cfg.Color.ChromaSampleLocTypeBottomField,
		NALLengthSize:                  4,
	})
	if err != nil {
		return EncoderParameterSets{}, err
	}
	return EncoderParameterSets{
		SPS:                           append([]byte(nil), sets.SPS...),
		PPS:                           append([]byte(nil), sets.PPS...),
		AnnexB:                        append([]byte(nil), sets.AnnexB...),
		AVCDecoderConfigurationRecord: append([]byte(nil), sets.AVCDecoderConfigurationRecord...),
	}, nil
}

func (e *Encoder) RecoveryPointSEI(recoveryFrameCount uint32) (EncoderSEI, error) {
	if e == nil {
		return EncoderSEI{}, encoderInvalid("nil encoder")
	}
	sei, err := h264.BuildEncoderRecoveryPointSEI(h264.EncoderRecoveryPointSEIConfig{
		RecoveryFrameCount:    recoveryFrameCount,
		ExactMatchFlag:        true,
		BrokenLinkFlag:        e.cfg.BFrames > 0,
		ChangingSliceGroupIDC: 0,
		NALLengthSize:         4,
	})
	if err != nil {
		return EncoderSEI{}, err
	}
	return EncoderSEI{
		NAL:    append([]byte(nil), sei.NAL...),
		AnnexB: append([]byte(nil), sei.AnnexB...),
		AVC:    append([]byte(nil), sei.AVC...),
	}, nil
}

func (e *Encoder) Encode(frame EncoderFrame) (EncodedFrame, error) {
	return e.EncodeInto(nil, frame)
}

func (e *Encoder) EncodeInto(dst []byte, frame EncoderFrame) (EncodedFrame, error) {
	if e == nil {
		return EncodedFrame{}, encoderInvalid("nil encoder")
	}
	view, err := e.validatedFrameView(frame)
	if err != nil {
		return EncodedFrame{}, err
	}
	idr := e.shouldEncodeIDR(view, frame)
	var nals []encoderRawNAL
	if idr && e.cfg.SPSPPSBeforeIDR && e.cfg.SPSPPSMode != EncoderSPSPPSOutOfBand {
		sets, err := e.ParameterSets()
		if err != nil {
			return EncodedFrame{}, err
		}
		nals = append(nals,
			encoderRawNAL{typ: uint8(h264.NALSPS), raw: sets.SPS, keyFrame: true, parameterSet: true},
			encoderRawNAL{typ: uint8(h264.NALPPS), raw: sets.PPS, keyFrame: true, parameterSet: true},
		)
	}

	if idr {
		slice, err := h264.BuildEncoderI420IntraPCMIDRSlice(h264.EncoderI420IntraPCMIDRConfig{
			Width:                      view.width,
			Height:                     view.height,
			StrideY:                    view.strideY,
			StrideCb:                   view.strideCb,
			StrideCr:                   view.strideCr,
			Y:                          view.y,
			Cb:                         view.cb,
			Cr:                         view.cr,
			FrameNum:                   e.frameNum & 0xff,
			IDRPicID:                   e.idrPicID & 0xffff,
			InitialQP:                  e.cfg.InitialQP,
			DisableDeblockingFilterIDC: encoderDeblockingFilterIDC(e.cfg.DeblockMode),
			NALLengthSize:              4,
		})
		if err != nil {
			return EncodedFrame{}, err
		}
		nals = append(nals, encoderRawNAL{typ: uint8(h264.NALIDRSlice), raw: slice.NAL, keyFrame: true})
	} else if e.referenceMatches(view) {
		slice, err := h264.BuildEncoderI420PSkipSlice(h264.EncoderI420PSkipConfig{
			Width:                      view.width,
			Height:                     view.height,
			FrameNum:                   e.frameNum & 0xff,
			InitialQP:                  e.cfg.InitialQP,
			DisableDeblockingFilterIDC: encoderDeblockingFilterIDC(e.cfg.DeblockMode),
			NALLengthSize:              4,
		})
		if err != nil {
			return EncodedFrame{}, err
		}
		nals = append(nals, encoderRawNAL{typ: uint8(h264.NALSlice), raw: slice.NAL})
	} else {
		slice, err := h264.BuildEncoderI420IntraPCMPSlice(h264.EncoderI420IntraPCMPConfig{
			Width:                      view.width,
			Height:                     view.height,
			StrideY:                    view.strideY,
			StrideCb:                   view.strideCb,
			StrideCr:                   view.strideCr,
			Y:                          view.y,
			Cb:                         view.cb,
			Cr:                         view.cr,
			FrameNum:                   e.frameNum & 0xff,
			InitialQP:                  e.cfg.InitialQP,
			DisableDeblockingFilterIDC: encoderDeblockingFilterIDC(e.cfg.DeblockMode),
			NALLengthSize:              4,
		})
		if err != nil {
			return EncodedFrame{}, err
		}
		nals = append(nals, encoderRawNAL{typ: uint8(h264.NALSlice), raw: slice.NAL})
	}

	data, units, err := appendEncoderAccessUnit(dst, e.cfg.OutputFormat, nals)
	if err != nil {
		return EncodedFrame{}, err
	}
	rtpTime := uint32(frame.PTS)
	var packets []EncoderRTPPacket
	if e.cfg.OutputFormat == EncoderOutputRTP {
		packets, err = packetizeEncoderRTPMode1(nals, e.cfg.RTPMaxPayloadSize, rtpTime, e.cfg.STAPA)
		if err != nil {
			return EncodedFrame{}, err
		}
		e.stampRTPPackets(packets)
	}

	e.storeReference(view)
	e.forceIDR = false
	e.frameNum = (e.frameNum + 1) & 0xff
	if idr {
		e.idrPicID = (e.idrPicID + 1) & 0xffff
		e.framesSinceIDR = 1
	} else {
		e.framesSinceIDR++
	}
	return EncodedFrame{
		Data:       data,
		NALUnits:   units,
		RTPPackets: packets,
		KeyFrame:   idr,
		IDR:        idr,
		PTS:        frame.PTS,
		DTS:        frame.PTS,
		RTPTime:    rtpTime,
	}, nil
}

func (e *Encoder) ForceIDR() {
	if e != nil {
		e.forceIDR = true
	}
}

func (e *Encoder) HandlePLI() {
	e.ForceIDR()
}

func (e *Encoder) HandleFIR() {
	e.ForceIDR()
}

func (e *Encoder) PendingIDR() bool {
	return e != nil && e.forceIDR
}

func (e *Encoder) SetBitrate(targetBitrate, maxBitrate int) error {
	if e == nil {
		return encoderInvalid("nil encoder")
	}
	cfg := e.cfg
	cfg.TargetBitrate = targetBitrate
	cfg.MaxBitrate = maxBitrate
	normalized, err := normalizeEncoderConfig(cfg)
	if err != nil {
		return err
	}
	e.cfg = normalized
	return nil
}

func (e *Encoder) SetFrameRate(num, den int) error {
	if e == nil {
		return encoderInvalid("nil encoder")
	}
	cfg := e.cfg
	cfg.FrameRateNum = num
	cfg.FrameRateDen = den
	cfg.RTPTimestampIncrement = rtpTimestampIncrement(cfg.TimeBaseDen, num, den)
	normalized, err := normalizeEncoderConfig(cfg)
	if err != nil {
		return err
	}
	e.cfg = normalized
	return nil
}

func (e *Encoder) SetRTPMaxPayloadSize(size int) error {
	if e == nil {
		return encoderInvalid("nil encoder")
	}
	cfg := e.cfg
	cfg.RTPMaxPayloadSize = size
	normalized, err := normalizeEncoderConfig(cfg)
	if err != nil {
		return err
	}
	e.cfg = normalized
	return nil
}

func (e *Encoder) Reconfigure(update EncoderReconfigure) error {
	if e == nil {
		return encoderInvalid("nil encoder")
	}
	cfg := e.cfg
	oldWidth := cfg.Width
	oldHeight := cfg.Height
	if update.TargetBitrate != 0 {
		cfg.TargetBitrate = update.TargetBitrate
	}
	if update.MaxBitrate != 0 {
		cfg.MaxBitrate = update.MaxBitrate
	}
	if update.FrameRateNum != 0 || update.FrameRateDen != 0 {
		cfg.FrameRateNum = update.FrameRateNum
		cfg.FrameRateDen = update.FrameRateDen
		cfg.RTPTimestampIncrement = rtpTimestampIncrement(cfg.TimeBaseDen, cfg.FrameRateNum, cfg.FrameRateDen)
	}
	if update.Width != 0 || update.Height != 0 {
		cfg.Width = update.Width
		cfg.Height = update.Height
		cfg.StrideY = update.Width
		cfg.StrideCb = (update.Width + 1) / 2
		cfg.StrideCr = (update.Width + 1) / 2
	}
	if update.RTPMaxPayloadSize != 0 {
		cfg.RTPMaxPayloadSize = update.RTPMaxPayloadSize
	}
	if update.MaxFrameSize != 0 {
		cfg.MaxFrameSize = update.MaxFrameSize
	}
	if update.MaxEncodeTimeUS != 0 {
		cfg.MaxEncodeTimeUS = update.MaxEncodeTimeUS
	}
	if update.Preset != 0 {
		cfg.Preset = update.Preset
	}
	if update.SPSPPSBeforeIDR != nil {
		cfg.SPSPPSBeforeIDR = *update.SPSPPSBeforeIDR
	}
	if update.RecoveryPointSEI != nil {
		cfg.RecoveryPointSEI = *update.RecoveryPointSEI
	}
	normalized, err := normalizeEncoderConfig(cfg)
	if err != nil {
		return err
	}
	e.cfg = normalized
	if normalized.Width != oldWidth || normalized.Height != oldHeight {
		e.reference = encoderReferenceFrame{}
		e.framesSinceIDR = 0
		e.forceIDR = true
	}
	if update.ForceIDR {
		e.forceIDR = true
	}
	return nil
}

type encoderFrameView struct {
	y        []byte
	cb       []byte
	cr       []byte
	width    int
	height   int
	strideY  int
	strideCb int
	strideCr int
}

type encoderRawNAL struct {
	typ          uint8
	raw          []byte
	keyFrame     bool
	parameterSet bool
}

type encoderReferenceFrame struct {
	valid  bool
	width  int
	height int
	y      []byte
	cb     []byte
	cr     []byte
}

func (e *Encoder) validateFrame(frame EncoderFrame) error {
	_, err := e.validatedFrameView(frame)
	return err
}

func (e *Encoder) validatedFrameView(frame EncoderFrame) (encoderFrameView, error) {
	width := frame.Width
	if width == 0 {
		width = e.cfg.Width
	}
	height := frame.Height
	if height == 0 {
		height = e.cfg.Height
	}
	if width != e.cfg.Width || height != e.cfg.Height {
		return encoderFrameView{}, encoderInvalid("frame dimensions do not match encoder configuration")
	}
	strideY := frame.StrideY
	if strideY == 0 {
		strideY = e.cfg.StrideY
	}
	strideCb := frame.StrideCb
	if strideCb == 0 {
		strideCb = e.cfg.StrideCb
	}
	strideCr := frame.StrideCr
	if strideCr == 0 {
		strideCr = e.cfg.StrideCr
	}
	if strideY < width {
		return encoderFrameView{}, encoderInvalid("frame luma stride is smaller than width")
	}
	chromaWidth := (width + 1) / 2
	chromaHeight := (height + 1) / 2
	if strideCb < chromaWidth || strideCr < chromaWidth {
		return encoderFrameView{}, encoderInvalid("frame chroma stride is smaller than chroma width")
	}
	if len(frame.Y) < strideY*height {
		return encoderFrameView{}, encoderInvalid("frame luma plane is too small")
	}
	if len(frame.Cb) < strideCb*chromaHeight || len(frame.Cr) < strideCr*chromaHeight {
		return encoderFrameView{}, encoderInvalid("frame chroma plane is too small")
	}
	return encoderFrameView{
		y:        frame.Y,
		cb:       frame.Cb,
		cr:       frame.Cr,
		width:    width,
		height:   height,
		strideY:  strideY,
		strideCb: strideCb,
		strideCr: strideCr,
	}, nil
}

func (e *Encoder) shouldEncodeIDR(view encoderFrameView, frame EncoderFrame) bool {
	if e.forceIDR || frame.ForceIDR || !e.reference.valid {
		return true
	}
	if e.cfg.IDRInterval > 0 && e.framesSinceIDR >= e.cfg.IDRInterval {
		return true
	}
	if e.cfg.DeblockMode != EncoderDeblockDisabled {
		return true
	}
	return false
}

func (e *Encoder) referenceMatches(view encoderFrameView) bool {
	ref := &e.reference
	if !ref.valid || ref.width != view.width || ref.height != view.height {
		return false
	}
	if len(ref.y) != view.width*view.height {
		return false
	}
	chromaWidth := view.width / 2
	chromaHeight := view.height / 2
	if len(ref.cb) != chromaWidth*chromaHeight || len(ref.cr) != chromaWidth*chromaHeight {
		return false
	}
	for y := 0; y < view.height; y++ {
		src := view.y[y*view.strideY : y*view.strideY+view.width]
		dst := ref.y[y*view.width : (y+1)*view.width]
		if !bytes.Equal(src, dst) {
			return false
		}
	}
	for y := 0; y < chromaHeight; y++ {
		srcCb := view.cb[y*view.strideCb : y*view.strideCb+chromaWidth]
		srcCr := view.cr[y*view.strideCr : y*view.strideCr+chromaWidth]
		dstCb := ref.cb[y*chromaWidth : (y+1)*chromaWidth]
		dstCr := ref.cr[y*chromaWidth : (y+1)*chromaWidth]
		if !bytes.Equal(srcCb, dstCb) || !bytes.Equal(srcCr, dstCr) {
			return false
		}
	}
	return true
}

func (e *Encoder) storeReference(view encoderFrameView) {
	chromaWidth := view.width / 2
	chromaHeight := view.height / 2
	ref := &e.reference
	ref.width = view.width
	ref.height = view.height
	ref.y = resizeEncoderReferencePlane(ref.y, view.width*view.height)
	ref.cb = resizeEncoderReferencePlane(ref.cb, chromaWidth*chromaHeight)
	ref.cr = resizeEncoderReferencePlane(ref.cr, chromaWidth*chromaHeight)
	for y := 0; y < view.height; y++ {
		copy(ref.y[y*view.width:(y+1)*view.width], view.y[y*view.strideY:y*view.strideY+view.width])
	}
	for y := 0; y < chromaHeight; y++ {
		copy(ref.cb[y*chromaWidth:(y+1)*chromaWidth], view.cb[y*view.strideCb:y*view.strideCb+chromaWidth])
		copy(ref.cr[y*chromaWidth:(y+1)*chromaWidth], view.cr[y*view.strideCr:y*view.strideCr+chromaWidth])
	}
	ref.valid = true
}

func resizeEncoderReferencePlane(buf []byte, size int) []byte {
	if cap(buf) < size {
		return make([]byte, size)
	}
	return buf[:size]
}

func appendEncoderAccessUnit(dst []byte, format EncoderOutputFormat, nals []encoderRawNAL) ([]byte, []EncoderNALUnit, error) {
	units := make([]EncoderNALUnit, 0, len(nals))
	for _, nal := range nals {
		if len(nal.raw) == 0 {
			return dst, nil, encoderInvalid("empty encoder NAL")
		}
		switch format {
		case EncoderOutputAVC:
			if uint64(len(nal.raw)) > uint64(^uint32(0)) {
				return dst, nil, encoderInvalid("encoder NAL is too large for AVC output")
			}
			n := len(nal.raw)
			dst = append(dst, byte(n>>24), byte(n>>16), byte(n>>8), byte(n))
			offset := len(dst)
			dst = append(dst, nal.raw...)
			units = append(units, EncoderNALUnit{
				Type:         nal.typ,
				Offset:       offset,
				Size:         len(nal.raw),
				KeyFrame:     nal.keyFrame,
				ParameterSet: nal.parameterSet,
			})
		case EncoderOutputAnnexB, EncoderOutputRTP:
			dst = append(dst, 0, 0, 0, 1)
			offset := len(dst)
			dst = append(dst, nal.raw...)
			units = append(units, EncoderNALUnit{
				Type:         nal.typ,
				Offset:       offset,
				Size:         len(nal.raw),
				KeyFrame:     nal.keyFrame,
				ParameterSet: nal.parameterSet,
			})
		default:
			return dst, nil, encoderInvalid("unknown encoder output format")
		}
	}
	return dst, units, nil
}

func packetizeEncoderRTPMode1(nals []encoderRawNAL, maxPayloadSize int, timestamp uint32, stapa bool) ([]EncoderRTPPacket, error) {
	if maxPayloadSize < 3 {
		return nil, encoderInvalid("RTP max payload size must leave room for FU-A headers")
	}
	var packets []EncoderRTPPacket
	for i := 0; i < len(nals); {
		if stapa && nals[i].parameterSet {
			payload, count, err := buildEncoderSTAPA(nals[i:], maxPayloadSize)
			if err != nil {
				return nil, err
			}
			if count >= 2 {
				packets = append(packets, EncoderRTPPacket{Payload: payload, Timestamp: timestamp})
				i += count
				continue
			}
		}
		nal := nals[i]
		if len(nal.raw) == 0 {
			return nil, encoderInvalid("empty encoder NAL")
		}
		if len(nal.raw) <= maxPayloadSize {
			payload := append([]byte(nil), nal.raw...)
			packets = append(packets, EncoderRTPPacket{Payload: payload, Timestamp: timestamp})
			i++
			continue
		}
		header := nal.raw[0]
		payload := nal.raw[1:]
		maxFragment := maxPayloadSize - 2
		first := true
		for len(payload) != 0 {
			n := maxFragment
			if n > len(payload) {
				n = len(payload)
			}
			fu := make([]byte, 0, n+2)
			fu = append(fu, (header&0xe0)|28)
			fuHeader := header & 0x1f
			if first {
				fuHeader |= 0x80
			}
			if n == len(payload) {
				fuHeader |= 0x40
			}
			fu = append(fu, fuHeader)
			fu = append(fu, payload[:n]...)
			packets = append(packets, EncoderRTPPacket{Payload: fu, Timestamp: timestamp})
			payload = payload[n:]
			first = false
		}
		i++
	}
	if len(packets) != 0 {
		packets[len(packets)-1].Marker = true
	}
	return packets, nil
}

func (e *Encoder) stampRTPPackets(packets []EncoderRTPPacket) {
	for i := range packets {
		packets[i].PayloadType = e.cfg.RTPPayloadType
		packets[i].SequenceNumber = e.rtpSequenceNumber
		packets[i].SSRC = e.cfg.RTPSSRC
		packets[i].Data = appendEncoderRTPPacket(nil, packets[i])
		e.rtpSequenceNumber++
	}
}

func appendEncoderRTPPacket(dst []byte, pkt EncoderRTPPacket) []byte {
	markerPayloadType := pkt.PayloadType & 0x7f
	if pkt.Marker {
		markerPayloadType |= 0x80
	}
	dst = append(dst,
		0x80,
		markerPayloadType,
		byte(pkt.SequenceNumber>>8), byte(pkt.SequenceNumber),
		byte(pkt.Timestamp>>24), byte(pkt.Timestamp>>16), byte(pkt.Timestamp>>8), byte(pkt.Timestamp),
		byte(pkt.SSRC>>24), byte(pkt.SSRC>>16), byte(pkt.SSRC>>8), byte(pkt.SSRC),
	)
	return append(dst, pkt.Payload...)
}

func buildEncoderSTAPA(nals []encoderRawNAL, maxPayloadSize int) ([]byte, int, error) {
	payload := []byte{24}
	var maxNRI byte
	count := 0
	for _, nal := range nals {
		if !nal.parameterSet {
			break
		}
		if len(nal.raw) == 0 {
			return nil, 0, encoderInvalid("empty encoder NAL")
		}
		if len(nal.raw) > 0xffff {
			return nil, 0, encoderInvalid("encoder NAL is too large for STAP-A")
		}
		need := 2 + len(nal.raw)
		if len(payload)+need > maxPayloadSize {
			break
		}
		if nri := nal.raw[0] & 0x60; nri > maxNRI {
			maxNRI = nri
		}
		payload = append(payload, byte(len(nal.raw)>>8), byte(len(nal.raw)))
		payload = append(payload, nal.raw...)
		count++
	}
	if count < 2 {
		return nil, count, nil
	}
	payload[0] = maxNRI | 24
	return payload, count, nil
}

func encoderDeblockingFilterIDC(mode EncoderDeblockMode) uint32 {
	switch mode {
	case EncoderDeblockDisabled:
		return 1
	case EncoderDeblockSliceBoundary:
		return 2
	default:
		return 0
	}
}

func normalizeEncoderConfig(cfg EncoderConfig) (EncoderConfig, error) {
	if cfg.Width <= 0 || cfg.Height <= 0 {
		return cfg, encoderInvalid("width and height must be positive")
	}
	if cfg.PixelFormat == 0 {
		cfg.PixelFormat = EncoderPixelFormatI420
	}
	if cfg.PixelFormat != EncoderPixelFormatI420 {
		return cfg, encoderUnsupported("only 8-bit I420 input is in the realtime encoder scope today")
	}
	if cfg.Width%2 != 0 || cfg.Height%2 != 0 {
		return cfg, encoderInvalid("I420 width and height must be even")
	}
	if cfg.StrideY == 0 {
		cfg.StrideY = cfg.Width
	}
	if cfg.StrideCb == 0 {
		cfg.StrideCb = cfg.Width / 2
	}
	if cfg.StrideCr == 0 {
		cfg.StrideCr = cfg.Width / 2
	}
	if cfg.StrideY < cfg.Width || cfg.StrideCb < cfg.Width/2 || cfg.StrideCr < cfg.Width/2 {
		return cfg, encoderInvalid("strides must cover the configured planes")
	}
	if cfg.FrameRateNum <= 0 || cfg.FrameRateDen <= 0 {
		return cfg, encoderInvalid("frame rate numerator and denominator must be positive")
	}
	if cfg.TimeBaseNum == 0 {
		cfg.TimeBaseNum = 1
	}
	if cfg.TimeBaseDen == 0 {
		cfg.TimeBaseDen = 90000
	}
	if cfg.TimeBaseNum <= 0 || cfg.TimeBaseDen <= 0 {
		return cfg, encoderInvalid("time base numerator and denominator must be positive")
	}
	if cfg.Profile == 0 {
		cfg.Profile = EncoderProfileConstrainedBaseline
	}
	switch cfg.Profile {
	case EncoderProfileConstrainedBaseline, EncoderProfileBaseline:
	case EncoderProfileMain, EncoderProfileHigh:
		return cfg, encoderUnsupported("Main and High encoder profiles are planned but not admitted yet")
	default:
		return cfg, encoderInvalid("unknown encoder profile")
	}
	if cfg.LevelIDC == 0 {
		cfg.LevelIDC = 31
	}
	if cfg.EntropyMode == 0 {
		cfg.EntropyMode = EncoderEntropyCAVLC
	}
	if cfg.EntropyMode != EncoderEntropyCAVLC {
		return cfg, encoderUnsupported("CABAC is not admitted for the first realtime WebRTC encoder slice")
	}
	if cfg.DeblockMode == 0 {
		cfg.DeblockMode = EncoderDeblockEnabled
	}
	switch cfg.DeblockMode {
	case EncoderDeblockEnabled, EncoderDeblockDisabled, EncoderDeblockSliceBoundary:
	default:
		return cfg, encoderInvalid("unknown deblock mode")
	}
	if cfg.Transform8x8 {
		return cfg, encoderUnsupported("8x8 transform is outside the initial Baseline encoder scope")
	}
	if cfg.MaxReferenceFrames == 0 {
		cfg.MaxReferenceFrames = 1
	}
	if cfg.MaxReferenceFrames != 1 {
		return cfg, encoderUnsupported("the realtime encoder initially admits one reference frame")
	}
	if cfg.BFrames != 0 {
		return cfg, encoderUnsupported("B-frames are disabled for the realtime WebRTC encoder scope")
	}
	if cfg.SPSPPSMode == 0 {
		cfg.SPSPPSMode = EncoderSPSPPSInBandKeyframes
	}
	switch cfg.SPSPPSMode {
	case EncoderSPSPPSInBandKeyframes, EncoderSPSPPSOutOfBand, EncoderSPSPPSEveryIDR:
	default:
		return cfg, encoderInvalid("unknown SPS/PPS emission mode")
	}
	if cfg.RateControl == 0 {
		cfg.RateControl = EncoderRateControlCBR
	}
	switch cfg.RateControl {
	case EncoderRateControlCBR, EncoderRateControlVBR, EncoderRateControlConstantQP:
	default:
		return cfg, encoderInvalid("unknown rate-control mode")
	}
	if cfg.TargetBitrate <= 0 {
		return cfg, encoderInvalid("target bitrate must be positive")
	}
	if cfg.MaxBitrate == 0 {
		cfg.MaxBitrate = cfg.TargetBitrate
	}
	if cfg.MaxBitrate < cfg.TargetBitrate {
		return cfg, encoderInvalid("max bitrate must be greater than or equal to target bitrate")
	}
	if cfg.VBVBufferSize < 0 || cfg.MaxFrameSize < 0 {
		return cfg, encoderInvalid("VBV buffer size and max frame size cannot be negative")
	}
	if cfg.InitialQP == 0 {
		cfg.InitialQP = 26
	}
	if cfg.MinQP == 0 {
		cfg.MinQP = 10
	}
	if cfg.MaxQP == 0 {
		cfg.MaxQP = 42
	}
	if cfg.MinQP < 0 || cfg.MinQP > 51 || cfg.MaxQP < 0 || cfg.MaxQP > 51 || cfg.InitialQP < cfg.MinQP || cfg.InitialQP > cfg.MaxQP {
		return cfg, encoderInvalid("QP range must be within 0..51 and contain the initial QP")
	}
	if cfg.Preset == 0 {
		cfg.Preset = EncoderPresetRealtime
	}
	switch cfg.Preset {
	case EncoderPresetRealtime, EncoderPresetBalanced, EncoderPresetQuality:
	default:
		return cfg, encoderInvalid("unknown encoder preset")
	}
	if cfg.FrameDrop == 0 {
		cfg.FrameDrop = EncoderFrameDropToBitrate
	}
	switch cfg.FrameDrop {
	case EncoderFrameDropDisabled, EncoderFrameDropLate, EncoderFrameDropToBitrate:
	default:
		return cfg, encoderInvalid("unknown frame drop mode")
	}
	if cfg.MaxEncodeTimeUS < 0 || cfg.SliceCount < 0 || cfg.SliceMaxBytes < 0 || cfg.Workers < 0 {
		return cfg, encoderInvalid("latency and worker controls cannot be negative")
	}
	if cfg.SliceCount == 0 {
		cfg.SliceCount = 1
	}
	if cfg.Workers == 0 {
		cfg.Workers = 1
	}
	if cfg.Deterministic && cfg.Workers != 1 {
		return cfg, encoderInvalid("deterministic mode requires one worker")
	}
	if cfg.GOPSize <= 0 {
		cfg.GOPSize = 60
	}
	if cfg.IDRInterval <= 0 {
		cfg.IDRInterval = cfg.GOPSize
	}
	if cfg.IDRInterval > cfg.GOPSize {
		return cfg, encoderInvalid("IDR interval must be less than or equal to GOP size")
	}
	if cfg.IntraRefresh {
		return cfg, encoderUnsupported("intra refresh is planned but not admitted yet")
	}
	if cfg.OutputFormat == 0 {
		cfg.OutputFormat = EncoderOutputRTP
	}
	switch cfg.OutputFormat {
	case EncoderOutputAnnexB, EncoderOutputAVC, EncoderOutputRTP:
	default:
		return cfg, encoderInvalid("unknown encoder output format")
	}
	if cfg.OutputFormat == EncoderOutputRTP {
		if cfg.RTPMaxPayloadSize == 0 {
			cfg.RTPMaxPayloadSize = 1200
		}
		if cfg.RTPMaxPayloadSize < 3 {
			return cfg, encoderInvalid("RTP max payload size must leave room for FU-A headers")
		}
		switch cfg.RTPPacketizationMode {
		case EncoderRTPPacketizationSingleNAL, EncoderRTPPacketizationNonInterleaved:
		default:
			return cfg, encoderInvalid("unknown RTP packetization mode")
		}
		if cfg.RTPPacketizationMode != EncoderRTPPacketizationNonInterleaved {
			return cfg, encoderUnsupported("WebRTC encoder RTP output currently admits packetization-mode 1")
		}
		if !cfg.DONDisabled {
			return cfg, encoderUnsupported("interleaved DON mode is not part of WebRTC packetization-mode 1")
		}
		if cfg.RTPPayloadType == 0 {
			cfg.RTPPayloadType = 96
		}
		if cfg.RTPPayloadType > 127 {
			return cfg, encoderInvalid("RTP payload type must fit in seven bits")
		}
	}
	if cfg.RTPTimestampIncrement == 0 {
		cfg.RTPTimestampIncrement = rtpTimestampIncrement(cfg.TimeBaseDen, cfg.FrameRateNum, cfg.FrameRateDen)
	}
	return cfg, nil
}

func rtpTimestampIncrement(clock, frameRateNum, frameRateDen int) uint32 {
	if clock <= 0 || frameRateNum <= 0 || frameRateDen <= 0 {
		return 0
	}
	return uint32((clock * frameRateDen) / frameRateNum)
}

func encoderProfileSyntax(profile EncoderProfile) (uint8, uint8, error) {
	switch profile {
	case EncoderProfileConstrainedBaseline:
		return 66, 0x03, nil
	case EncoderProfileBaseline:
		return 66, 0x01, nil
	default:
		return 0, 0, encoderUnsupported("profile is not admitted for parameter-set generation")
	}
}

func encoderInvalid(detail string) error {
	return fmt.Errorf("h264: encoder %s: %w", detail, ErrInvalidData)
}

func encoderUnsupported(detail string) error {
	return fmt.Errorf("h264: encoder %s: %w", detail, ErrUnsupported)
}
