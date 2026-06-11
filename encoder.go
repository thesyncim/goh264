// SPDX-License-Identifier: LGPL-2.1-or-later

package goh264

import (
	"bytes"
	"fmt"
	"time"

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

type EncoderRTPPayloadFormat uint8

const (
	EncoderRTPPayloadSingleNAL EncoderRTPPayloadFormat = iota + 1
	EncoderRTPPayloadSTAPA
	EncoderRTPPayloadFUA
)

type EncoderRTPPacketMetadata struct {
	PacketIndex int
	PacketCount int

	FramePTS int64
	FrameDTS int64
	RTPTime  uint32
	KeyFrame bool
	IDR      bool

	PayloadFormat EncoderRTPPayloadFormat
	NALUnitType   uint8
	NALUnitCount  int
	StartOfNAL    bool
	EndOfNAL      bool
	ParameterSet  bool
}

type EncoderRTPPacketCallback func(packet EncoderRTPPacket, metadata EncoderRTPPacketMetadata)

type EncodedFrame struct {
	Data       []byte
	NALUnits   []EncoderNALUnit
	RTPPackets []EncoderRTPPacket
	KeyFrame   bool
	IDR        bool
	PTS        int64
	DTS        int64
	RTPTime    uint32
	Dropped    bool
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
	TargetBitrate         int
	MaxBitrate            int
	RateControl           EncoderRateControlMode
	VBVBufferSize         *int
	InitialQP             *int
	MinQP                 *int
	MaxQP                 *int
	FrameDrop             EncoderFrameDropMode
	FrameRateNum          int
	FrameRateDen          int
	Width                 int
	Height                int
	DeblockMode           EncoderDeblockMode
	RTPMaxPayloadSize     int
	MaxFrameSize          int
	MaxEncodeTimeUS       int
	SliceCount            int
	SliceMaxBytes         int
	Preset                EncoderPreset
	ForceIDR              bool
	GOPSize               int
	IDRInterval           int
	SPSPPSMode            EncoderSPSPPSMode
	SPSPPSBeforeIDR       *bool
	RecoveryPointSEI      *bool
	OutputFormat          EncoderOutputFormat
	RTPPacketizationMode  *EncoderRTPPacketizationMode
	STAPA                 *bool
	RTPPayloadType        *uint8
	RTPSSRC               *uint32
	RTPTimestampIncrement uint32
}

type Encoder struct {
	cfg                EncoderConfig
	forceIDR           bool
	frameNum           uint32
	idrPicID           uint32
	rtpSequenceNumber  uint16
	nextRTPTime        uint32
	rtpTimeInitialized bool
	reference          encoderReferenceFrame
	p16MVDs            []h264.EncoderMotionVectorDelta
	bitrateCreditBytes int
	bitrateCreditInit  bool
	framesSinceIDR     int
	rtpPacketCallback  EncoderRTPPacketCallback
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
		CropLeft:                       e.cfg.Crop.Left,
		CropRight:                      e.cfg.Crop.Right,
		CropTop:                        e.cfg.Crop.Top,
		CropBottom:                     e.cfg.Crop.Bottom,
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
	lateStart := encoderLateDropStart(e.cfg)
	var nalsBuf [4]encoderRawNAL
	nals := nalsBuf[:0]
	if e.shouldEmitParameterSets(idr) {
		sets, err := e.ParameterSets()
		if err != nil {
			return EncodedFrame{}, err
		}
		nals = append(nals,
			encoderRawNAL{typ: uint8(h264.NALSPS), raw: sets.SPS, keyFrame: true, parameterSet: true},
			encoderRawNAL{typ: uint8(h264.NALPPS), raw: sets.PPS, keyFrame: true, parameterSet: true},
		)
	}

	var sliceRangeBuf [4]encoderSliceRange
	sliceRanges := appendEncoderSliceRanges(sliceRangeBuf[:0], view.width, view.height, e.cfg.SliceCount)
	if idr {
		for _, r := range sliceRanges {
			nal, err := buildEncoderI420IntraPCMIDRNAL(h264.EncoderI420IntraPCMIDRConfig{
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
				FirstMBAddr:                uint32(r.firstMB),
				MacroblockCount:            uint32(r.macroblockCount),
				NALLengthSize:              4,
			})
			if err != nil {
				return EncodedFrame{}, err
			}
			nals = append(nals, encoderRawNAL{typ: uint8(h264.NALIDRSlice), raw: nal, keyFrame: true})
		}
	} else if e.referenceMatches(view) {
		for _, r := range sliceRanges {
			nal, err := buildEncoderI420PSkipNAL(h264.EncoderI420PSkipConfig{
				Width:                      view.width,
				Height:                     view.height,
				FrameNum:                   e.frameNum & 0xff,
				InitialQP:                  e.cfg.InitialQP,
				DisableDeblockingFilterIDC: encoderDeblockingFilterIDC(e.cfg.DeblockMode),
				FirstMBAddr:                uint32(r.firstMB),
				MacroblockCount:            uint32(r.macroblockCount),
				NALLengthSize:              4,
			})
			if err != nil {
				return EncodedFrame{}, err
			}
			nals = append(nals, encoderRawNAL{typ: uint8(h264.NALSlice), raw: nal})
		}
	} else if mvdX, mvdY, ok := e.p16x16NoResidualMotion(view); ok {
		var mvdBuf [64]h264.EncoderMotionVectorDelta
		macroblocksPerRow := view.width >> 4
		for _, r := range sliceRanges {
			mvdsBuf := mvdBuf[:0]
			if r.macroblockCount > cap(mvdsBuf) {
				e.p16MVDs = resizeEncoderP16x16MVDs(e.p16MVDs, r.macroblockCount)
				mvdsBuf = e.p16MVDs[:0]
			}
			mvds := appendEncoderP16x16NoResidualMVDs(mvdsBuf, r.firstMB, r.macroblockCount, macroblocksPerRow, mvdX, mvdY)
			nal, err := buildEncoderI420P16x16NoResidualNAL(h264.EncoderI420P16x16NoResidualConfig{
				Width:                      view.width,
				Height:                     view.height,
				FrameNum:                   e.frameNum & 0xff,
				InitialQP:                  e.cfg.InitialQP,
				DisableDeblockingFilterIDC: encoderDeblockingFilterIDC(e.cfg.DeblockMode),
				FirstMBAddr:                uint32(r.firstMB),
				MacroblockCount:            uint32(r.macroblockCount),
				MVDX:                       mvdX,
				MVDY:                       mvdY,
				MVDs:                       mvds,
				NALLengthSize:              4,
			})
			if err != nil {
				return EncodedFrame{}, err
			}
			nals = append(nals, encoderRawNAL{typ: uint8(h264.NALSlice), raw: nal})
		}
	} else {
		if e.cfg.RecoveryPointSEI {
			sei, err := e.RecoveryPointSEI(0)
			if err != nil {
				return EncodedFrame{}, err
			}
			nals = append(nals, encoderRawNAL{typ: uint8(h264.NALSEI), raw: sei.NAL})
		}
		for _, r := range sliceRanges {
			nal, err := buildEncoderI420IntraPCMPNAL(h264.EncoderI420IntraPCMPConfig{
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
				FirstMBAddr:                uint32(r.firstMB),
				MacroblockCount:            uint32(r.macroblockCount),
				NALLengthSize:              4,
			})
			if err != nil {
				return EncodedFrame{}, err
			}
			nals = append(nals, encoderRawNAL{typ: uint8(h264.NALSlice), raw: nal})
		}
	}

	outputSize, err := encoderAccessUnitOutputSize(e.cfg.OutputFormat, nals)
	if err != nil {
		return EncodedFrame{}, err
	}
	rtpTime := e.encoderRTPTime(frame)
	if miss, err := encoderOutputBudgetMiss(e.cfg, nals, outputSize); err != nil {
		if e.cfg.FrameDrop == EncoderFrameDropToBitrate && miss != encoderOutputBudgetNone {
			e.advanceEncoderRTPTime(frame, rtpTime)
			e.advanceEncoderBitrateBudget(0)
			return EncodedFrame{
				PTS:     frame.PTS,
				DTS:     frame.PTS,
				RTPTime: rtpTime,
				Dropped: true,
			}, nil
		}
		return EncodedFrame{}, err
	}
	if e.encoderBitrateBudgetMiss(outputSize) {
		e.advanceEncoderRTPTime(frame, rtpTime)
		e.advanceEncoderBitrateBudget(0)
		return EncodedFrame{
			PTS:     frame.PTS,
			DTS:     frame.PTS,
			RTPTime: rtpTime,
			Dropped: true,
		}, nil
	}
	data, units, err := appendEncoderAccessUnit(dst, e.cfg.OutputFormat, nals)
	if err != nil {
		return EncodedFrame{}, err
	}
	var packets []EncoderRTPPacket
	if e.cfg.OutputFormat == EncoderOutputRTP {
		switch e.cfg.RTPPacketizationMode {
		case EncoderRTPPacketizationSingleNAL:
			packets, err = packetizeEncoderRTPSingleNAL(nals, e.cfg.RTPMaxPayloadSize, rtpTime)
		case EncoderRTPPacketizationNonInterleaved:
			packets, err = packetizeEncoderRTPMode1(nals, e.cfg.RTPMaxPayloadSize, rtpTime, e.cfg.STAPA)
		default:
			err = encoderInvalid("unknown RTP packetization mode")
		}
		if err != nil {
			return EncodedFrame{}, err
		}
		if encoderLateBudgetMiss(lateStart, e.cfg) {
			e.advanceEncoderRTPTime(frame, rtpTime)
			return EncodedFrame{
				PTS:     frame.PTS,
				DTS:     frame.PTS,
				RTPTime: rtpTime,
				Dropped: true,
			}, nil
		}
		e.stampRTPPackets(packets)
		e.notifyRTPPacketCallback(packets, frame, rtpTime, idr, idr)
	} else if encoderLateBudgetMiss(lateStart, e.cfg) {
		e.advanceEncoderRTPTime(frame, rtpTime)
		return EncodedFrame{
			PTS:     frame.PTS,
			DTS:     frame.PTS,
			RTPTime: rtpTime,
			Dropped: true,
		}, nil
	}

	e.advanceEncoderRTPTime(frame, rtpTime)
	e.advanceEncoderBitrateBudget(outputSize)
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
	e.resetEncoderBitrateBudget()
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
	e.resetEncoderBitrateBudget()
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

func (e *Encoder) SetRTPPacketCallback(callback EncoderRTPPacketCallback) {
	if e != nil {
		e.rtpPacketCallback = callback
	}
}

func (e *Encoder) Reconfigure(update EncoderReconfigure) error {
	if e == nil {
		return encoderInvalid("nil encoder")
	}
	cfg := e.cfg
	oldWidth := cfg.Width
	oldHeight := cfg.Height
	qpRefresh := update.InitialQP != nil || update.MinQP != nil || update.MaxQP != nil
	if update.TargetBitrate != 0 {
		cfg.TargetBitrate = update.TargetBitrate
	}
	if update.MaxBitrate != 0 {
		cfg.MaxBitrate = update.MaxBitrate
	}
	if update.RateControl != 0 {
		cfg.RateControl = update.RateControl
	}
	if update.VBVBufferSize != nil {
		cfg.VBVBufferSize = *update.VBVBufferSize
	}
	if update.InitialQP != nil {
		cfg.InitialQP = *update.InitialQP
	}
	if update.MinQP != nil {
		cfg.MinQP = *update.MinQP
	}
	if update.MaxQP != nil {
		cfg.MaxQP = *update.MaxQP
	}
	if update.FrameDrop != 0 {
		cfg.FrameDrop = update.FrameDrop
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
	if update.DeblockMode != 0 {
		cfg.DeblockMode = update.DeblockMode
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
	if update.SliceCount != 0 {
		cfg.SliceCount = update.SliceCount
	}
	if update.SliceMaxBytes != 0 {
		cfg.SliceMaxBytes = update.SliceMaxBytes
	}
	if update.Preset != 0 {
		cfg.Preset = update.Preset
	}
	if update.GOPSize != 0 {
		cfg.GOPSize = update.GOPSize
	}
	if update.IDRInterval != 0 {
		cfg.IDRInterval = update.IDRInterval
	}
	if update.SPSPPSMode != 0 {
		cfg.SPSPPSMode = update.SPSPPSMode
	}
	if update.SPSPPSBeforeIDR != nil {
		cfg.SPSPPSBeforeIDR = *update.SPSPPSBeforeIDR
	}
	if update.RecoveryPointSEI != nil {
		cfg.RecoveryPointSEI = *update.RecoveryPointSEI
	}
	if update.OutputFormat != 0 {
		cfg.OutputFormat = update.OutputFormat
	}
	if update.RTPPacketizationMode != nil {
		cfg.RTPPacketizationMode = *update.RTPPacketizationMode
	}
	if update.STAPA != nil {
		cfg.STAPA = *update.STAPA
	}
	if update.RTPPayloadType != nil {
		cfg.RTPPayloadType = *update.RTPPayloadType
	}
	if update.RTPSSRC != nil {
		cfg.RTPSSRC = *update.RTPSSRC
	}
	if update.RTPTimestampIncrement != 0 {
		cfg.RTPTimestampIncrement = update.RTPTimestampIncrement
	}
	normalized, err := normalizeEncoderConfigWithExplicitQP(cfg, update.InitialQP != nil, update.MinQP != nil, update.MaxQP != nil)
	if err != nil {
		return err
	}
	bitrateBudgetRefresh := normalized.TargetBitrate != e.cfg.TargetBitrate ||
		normalized.MaxBitrate != e.cfg.MaxBitrate ||
		normalized.VBVBufferSize != e.cfg.VBVBufferSize ||
		normalized.FrameRateNum != e.cfg.FrameRateNum ||
		normalized.FrameRateDen != e.cfg.FrameRateDen ||
		normalized.RateControl != e.cfg.RateControl ||
		normalized.FrameDrop != e.cfg.FrameDrop
	e.cfg = normalized
	if bitrateBudgetRefresh {
		e.resetEncoderBitrateBudget()
	}
	if normalized.Width != oldWidth || normalized.Height != oldHeight {
		e.reference = encoderReferenceFrame{}
		e.framesSinceIDR = 0
		e.forceIDR = true
	}
	if qpRefresh {
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

type encoderSliceRange struct {
	firstMB         int
	macroblockCount int
}

func encoderLateDropStart(cfg EncoderConfig) time.Time {
	if cfg.FrameDrop == EncoderFrameDropLate && cfg.MaxEncodeTimeUS > 0 {
		return time.Now()
	}
	return time.Time{}
}

func encoderLateBudgetMiss(start time.Time, cfg EncoderConfig) bool {
	return !start.IsZero() && time.Since(start) > time.Duration(cfg.MaxEncodeTimeUS)*time.Microsecond
}

func buildEncoderI420IntraPCMIDRNAL(cfg h264.EncoderI420IntraPCMIDRConfig) ([]byte, error) {
	rbsp, err := h264.EncodeI420IntraPCMIDRSliceRBSP(cfg)
	if err != nil {
		return nil, err
	}
	return h264.AppendNAL(nil, 3, h264.NALIDRSlice, rbsp)
}

func buildEncoderI420PSkipNAL(cfg h264.EncoderI420PSkipConfig) ([]byte, error) {
	rbsp, err := h264.EncodeI420PSkipSliceRBSP(cfg)
	if err != nil {
		return nil, err
	}
	return h264.AppendNAL(nil, 2, h264.NALSlice, rbsp)
}

func buildEncoderI420P16x16NoResidualNAL(cfg h264.EncoderI420P16x16NoResidualConfig) ([]byte, error) {
	rbsp, err := h264.EncodeI420P16x16NoResidualSliceRBSP(cfg)
	if err != nil {
		return nil, err
	}
	return h264.AppendNAL(make([]byte, 0, 1+len(rbsp)+len(rbsp)/2), 2, h264.NALSlice, rbsp)
}

func buildEncoderI420IntraPCMPNAL(cfg h264.EncoderI420IntraPCMPConfig) ([]byte, error) {
	rbsp, err := h264.EncodeI420IntraPCMPSliceRBSP(cfg)
	if err != nil {
		return nil, err
	}
	return h264.AppendNAL(nil, 2, h264.NALSlice, rbsp)
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
	return false
}

func (e *Encoder) shouldEmitParameterSets(idr bool) bool {
	if !idr {
		return false
	}
	switch e.cfg.SPSPPSMode {
	case EncoderSPSPPSOutOfBand:
		return false
	case EncoderSPSPPSEveryIDR:
		return true
	default:
		return e.cfg.SPSPPSBeforeIDR
	}
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

func (e *Encoder) p16x16NoResidualMotion(view encoderFrameView) (int32, int32, bool) {
	if view.height < 16 ||
		view.height&15 != 0 ||
		view.width < 16 ||
		view.width&15 != 0 {
		return 0, 0, false
	}
	if e.cfg.DeblockMode != EncoderDeblockDisabled && encoderMacroblockCount(view.width, view.height) != 1 {
		return 0, 0, false
	}
	ref := &e.reference
	if !ref.valid || ref.width != view.width || ref.height != view.height {
		return 0, 0, false
	}
	if len(ref.y) != view.width*view.height {
		return 0, 0, false
	}
	chromaWidth := view.width / 2
	chromaHeight := view.height / 2
	if len(ref.cb) != chromaWidth*chromaHeight || len(ref.cr) != chromaWidth*chromaHeight {
		return 0, 0, false
	}

	primaryCandidates := [...]struct {
		dx int
		dy int
	}{
		{dx: 2, dy: 0},
		{dx: -2, dy: 0},
		{dx: 0, dy: 2},
		{dx: 0, dy: -2},
	}
	for _, candidate := range primaryCandidates {
		if encoderI420MatchesIntegerMotion(ref, view, candidate.dx, candidate.dy) {
			return int32(candidate.dx * 4), int32(candidate.dy * 4), true
		}
	}
	const maxExactMotion = 4
	for radius := 2; radius <= maxExactMotion; radius += 2 {
		for dy := -radius; dy <= radius; dy += 2 {
			for dx := -radius; dx <= radius; dx += 2 {
				if dx == 0 && dy == 0 {
					continue
				}
				if absInt(dx) != radius && absInt(dy) != radius {
					continue
				}
				if (dx == 2 && dy == 0) ||
					(dx == -2 && dy == 0) ||
					(dx == 0 && dy == 2) ||
					(dx == 0 && dy == -2) {
					continue
				}
				if encoderI420MatchesIntegerMotion(ref, view, dx, dy) {
					return int32(dx * 4), int32(dy * 4), true
				}
			}
		}
	}
	return 0, 0, false
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func appendEncoderP16x16NoResidualMVDs(dst []h264.EncoderMotionVectorDelta, firstMB int, macroblockCount int, macroblocksPerRow int, mvdX int32, mvdY int32) []h264.EncoderMotionVectorDelta {
	if cap(dst) < macroblockCount {
		dst = make([]h264.EncoderMotionVectorDelta, macroblockCount)
	} else {
		dst = dst[:macroblockCount]
	}
	if macroblockCount == 0 {
		return dst
	}
	for i := 0; i < macroblockCount; i++ {
		mbAddr := firstMB + i
		if encoderP16x16NoResidualHasMVPredictor(mbAddr, firstMB, macroblocksPerRow) {
			dst[i] = h264.EncoderMotionVectorDelta{}
		} else {
			dst[i] = h264.EncoderMotionVectorDelta{X: mvdX, Y: mvdY}
		}
	}
	return dst
}

func encoderP16x16NoResidualHasMVPredictor(mbAddr int, firstMB int, macroblocksPerRow int) bool {
	if macroblocksPerRow <= 0 || mbAddr <= firstMB {
		return false
	}
	x := mbAddr % macroblocksPerRow
	y := mbAddr / macroblocksPerRow
	if x > 0 && mbAddr-1 >= firstMB {
		return true
	}
	if y == 0 {
		return false
	}
	top := mbAddr - macroblocksPerRow
	if top >= firstMB {
		return true
	}
	if x < macroblocksPerRow-1 && top+1 >= firstMB {
		return true
	}
	return x > 0 && top-1 >= firstMB
}

func encoderI420MatchesIntegerMotion(ref *encoderReferenceFrame, view encoderFrameView, dx int, dy int) bool {
	if !encoderPlaneMatchesIntegerMotion(view.y, view.strideY, view.width, view.height, ref.y, view.width, dx, dy) {
		return false
	}
	chromaWidth := view.width / 2
	chromaHeight := view.height / 2
	chromaDX := dx / 2
	chromaDY := dy / 2
	return encoderPlaneMatchesIntegerMotion(view.cb, view.strideCb, chromaWidth, chromaHeight, ref.cb, chromaWidth, chromaDX, chromaDY) &&
		encoderPlaneMatchesIntegerMotion(view.cr, view.strideCr, chromaWidth, chromaHeight, ref.cr, chromaWidth, chromaDX, chromaDY)
}

func encoderPlaneMatchesIntegerMotion(cur []byte, curStride int, width int, height int, ref []byte, refStride int, dx int, dy int) bool {
	for y := 0; y < height; y++ {
		curRow := cur[y*curStride : y*curStride+width]
		refY := clampEncoderReferenceCoord(y+dy, height)
		for x := 0; x < width; x++ {
			refX := clampEncoderReferenceCoord(x+dx, width)
			if curRow[x] != ref[refY*refStride+refX] {
				return false
			}
		}
	}
	return true
}

func clampEncoderReferenceCoord(v int, limit int) int {
	if v < 0 {
		return 0
	}
	if v >= limit {
		return limit - 1
	}
	return v
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

func resizeEncoderP16x16MVDs(buf []h264.EncoderMotionVectorDelta, size int) []h264.EncoderMotionVectorDelta {
	if cap(buf) < size {
		return make([]h264.EncoderMotionVectorDelta, size)
	}
	return buf[:size]
}

func appendEncoderSliceRanges(dst []encoderSliceRange, width int, height int, sliceCount int) []encoderSliceRange {
	dst = dst[:0]
	total := encoderMacroblockCount(width, height)
	if sliceCount <= 0 {
		sliceCount = 1
	}
	if sliceCount > total {
		sliceCount = total
	}
	base := total / sliceCount
	extra := total % sliceCount
	first := 0
	for i := 0; i < sliceCount; i++ {
		count := base
		if i < extra {
			count++
		}
		dst = append(dst, encoderSliceRange{firstMB: first, macroblockCount: count})
		first += count
	}
	return dst
}

func encoderMacroblockCount(width int, height int) int {
	return ((width + 15) >> 4) * ((height + 15) >> 4)
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

func encoderAccessUnitOutputSize(format EncoderOutputFormat, nals []encoderRawNAL) (int, error) {
	var size int
	for _, nal := range nals {
		if len(nal.raw) == 0 {
			return 0, encoderInvalid("empty encoder NAL")
		}
		switch format {
		case EncoderOutputAVC:
			if uint64(len(nal.raw)) > uint64(^uint32(0)) {
				return 0, encoderInvalid("encoder NAL is too large for AVC output")
			}
			size += 4 + len(nal.raw)
		case EncoderOutputAnnexB, EncoderOutputRTP:
			size += 4 + len(nal.raw)
		default:
			return 0, encoderInvalid("unknown encoder output format")
		}
	}
	return size, nil
}

type encoderOutputBudget uint8

const (
	encoderOutputBudgetNone encoderOutputBudget = iota
	encoderOutputBudgetSlice
	encoderOutputBudgetFrame
)

func validateEncoderOutputBudgets(cfg EncoderConfig, nals []encoderRawNAL, outputSize int) error {
	_, err := encoderOutputBudgetMiss(cfg, nals, outputSize)
	return err
}

func encoderOutputBudgetMiss(cfg EncoderConfig, nals []encoderRawNAL, outputSize int) (encoderOutputBudget, error) {
	if cfg.SliceMaxBytes > 0 {
		for _, nal := range nals {
			if encoderRawNALIsVCL(nal) && len(nal.raw) > cfg.SliceMaxBytes {
				return encoderOutputBudgetSlice, encoderInvalid("encoded slice exceeds slice byte target")
			}
		}
	}
	if cfg.MaxFrameSize > 0 && outputSize > cfg.MaxFrameSize {
		return encoderOutputBudgetFrame, encoderInvalid("encoded access unit exceeds max frame size")
	}
	return encoderOutputBudgetNone, nil
}

func (e *Encoder) encoderBitrateBudgetMiss(outputSize int) bool {
	if e == nil || !encoderBitrateBudgetEnabled(e.cfg) {
		return false
	}
	e.ensureEncoderBitrateBudget()
	return outputSize > e.bitrateCreditBytes
}

func (e *Encoder) advanceEncoderBitrateBudget(outputSize int) {
	if e == nil || !encoderBitrateBudgetEnabled(e.cfg) {
		if e != nil {
			e.resetEncoderBitrateBudget()
		}
		return
	}
	e.ensureEncoderBitrateBudget()
	if outputSize >= e.bitrateCreditBytes {
		e.bitrateCreditBytes = 0
	} else {
		e.bitrateCreditBytes -= outputSize
	}
	e.bitrateCreditBytes += encoderBitrateFrameBudgetBytes(e.cfg)
	if capBytes := encoderVBVBufferBudgetBytes(e.cfg); capBytes > 0 && e.bitrateCreditBytes > capBytes {
		e.bitrateCreditBytes = capBytes
	}
}

func (e *Encoder) ensureEncoderBitrateBudget() {
	if e == nil || e.bitrateCreditInit {
		return
	}
	e.bitrateCreditBytes = encoderVBVBufferBudgetBytes(e.cfg)
	if e.bitrateCreditBytes == 0 {
		e.bitrateCreditBytes = encoderBitrateFrameBudgetBytes(e.cfg)
	}
	e.bitrateCreditInit = true
}

func (e *Encoder) resetEncoderBitrateBudget() {
	if e == nil {
		return
	}
	e.bitrateCreditBytes = 0
	e.bitrateCreditInit = false
}

func encoderBitrateBudgetEnabled(cfg EncoderConfig) bool {
	return cfg.FrameDrop == EncoderFrameDropToBitrate && cfg.RateControl != EncoderRateControlConstantQP
}

func encoderBitrateFrameBudgetBytes(cfg EncoderConfig) int {
	if cfg.MaxBitrate <= 0 || cfg.FrameRateNum <= 0 || cfg.FrameRateDen <= 0 {
		return 0
	}
	bitsNumerator := uint64(cfg.MaxBitrate) * uint64(cfg.FrameRateDen)
	bitsPerFrame := (bitsNumerator + uint64(cfg.FrameRateNum) - 1) / uint64(cfg.FrameRateNum)
	bytesPerFrame := (bitsPerFrame + 7) / 8
	maxInt := uint64(int(^uint(0) >> 1))
	if bytesPerFrame > maxInt {
		return int(maxInt)
	}
	return int(bytesPerFrame)
}

func encoderVBVBufferBudgetBytes(cfg EncoderConfig) int {
	if cfg.VBVBufferSize <= 0 {
		return 0
	}
	bytes := (uint64(cfg.VBVBufferSize) + 7) / 8
	maxInt := uint64(int(^uint(0) >> 1))
	if bytes > maxInt {
		return int(maxInt)
	}
	return int(bytes)
}

func encoderRawNALIsVCL(nal encoderRawNAL) bool {
	return nal.typ == uint8(h264.NALSlice) || nal.typ == uint8(h264.NALIDRSlice)
}

func packetizeEncoderRTPSingleNAL(nals []encoderRawNAL, maxPayloadSize int, timestamp uint32) ([]EncoderRTPPacket, error) {
	if maxPayloadSize < 1 {
		return nil, encoderInvalid("RTP max payload size must fit a NAL header")
	}
	packets := make([]EncoderRTPPacket, 0, len(nals))
	for _, nal := range nals {
		if len(nal.raw) == 0 {
			return nil, encoderInvalid("empty encoder NAL")
		}
		if len(nal.raw) > maxPayloadSize {
			return nil, encoderInvalid("encoder NAL exceeds RTP packetization-mode 0 payload size")
		}
		payload := append([]byte(nil), nal.raw...)
		packets = append(packets, EncoderRTPPacket{Payload: payload, Timestamp: timestamp})
	}
	if len(packets) != 0 {
		packets[len(packets)-1].Marker = true
	}
	return packets, nil
}

func packetizeEncoderRTPMode1(nals []encoderRawNAL, maxPayloadSize int, timestamp uint32, stapa bool) ([]EncoderRTPPacket, error) {
	if maxPayloadSize < 3 {
		return nil, encoderInvalid("RTP max payload size must leave room for FU-A headers")
	}
	packets := make([]EncoderRTPPacket, 0, len(nals))
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

func (e *Encoder) notifyRTPPacketCallback(packets []EncoderRTPPacket, frame EncoderFrame, rtpTime uint32, keyFrame bool, idr bool) {
	callback := e.rtpPacketCallback
	if callback == nil {
		return
	}
	for i, pkt := range packets {
		meta := encoderRTPPacketMetadataFromPayload(pkt.Payload)
		meta.PacketIndex = i
		meta.PacketCount = len(packets)
		meta.FramePTS = frame.PTS
		meta.FrameDTS = frame.PTS
		meta.RTPTime = rtpTime
		meta.KeyFrame = keyFrame
		meta.IDR = idr
		callback(cloneEncoderRTPPacket(pkt), meta)
	}
}

func cloneEncoderRTPPacket(pkt EncoderRTPPacket) EncoderRTPPacket {
	pkt.Data = append([]byte(nil), pkt.Data...)
	pkt.Payload = append([]byte(nil), pkt.Payload...)
	return pkt
}

func encoderRTPPacketMetadataFromPayload(payload []byte) EncoderRTPPacketMetadata {
	if len(payload) == 0 {
		return EncoderRTPPacketMetadata{}
	}
	typ := payload[0] & 0x1f
	switch typ {
	case 24:
		count, parameterSet := encoderRTPSTAPAMetadata(payload)
		return EncoderRTPPacketMetadata{
			PayloadFormat: EncoderRTPPayloadSTAPA,
			NALUnitType:   24,
			NALUnitCount:  count,
			ParameterSet:  parameterSet,
		}
	case 28:
		meta := EncoderRTPPacketMetadata{
			PayloadFormat: EncoderRTPPayloadFUA,
			NALUnitCount:  1,
		}
		if len(payload) >= 2 {
			meta.NALUnitType = payload[1] & 0x1f
			meta.StartOfNAL = payload[1]&0x80 != 0
			meta.EndOfNAL = payload[1]&0x40 != 0
			meta.ParameterSet = encoderRTPNALTypeIsParameterSet(meta.NALUnitType)
		}
		return meta
	default:
		return EncoderRTPPacketMetadata{
			PayloadFormat: EncoderRTPPayloadSingleNAL,
			NALUnitType:   typ,
			NALUnitCount:  1,
			StartOfNAL:    true,
			EndOfNAL:      true,
			ParameterSet:  encoderRTPNALTypeIsParameterSet(typ),
		}
	}
}

func encoderRTPSTAPAMetadata(payload []byte) (int, bool) {
	count := 0
	parameterSet := true
	for pos := 1; pos < len(payload); {
		if pos+2 > len(payload) {
			return count, false
		}
		size := int(payload[pos])<<8 | int(payload[pos+1])
		pos += 2
		if size <= 0 || pos+size > len(payload) {
			return count, false
		}
		if !encoderRTPNALTypeIsParameterSet(payload[pos] & 0x1f) {
			parameterSet = false
		}
		count++
		pos += size
	}
	return count, count != 0 && parameterSet
}

func encoderRTPNALTypeIsParameterSet(typ uint8) bool {
	return typ == uint8(h264.NALSPS) || typ == uint8(h264.NALPPS)
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
	return normalizeEncoderConfigWithExplicitQP(cfg, false, false, false)
}

func normalizeEncoderConfigWithExplicitQP(cfg EncoderConfig, explicitInitialQP, explicitMinQP, explicitMaxQP bool) (EncoderConfig, error) {
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
	if err := validateEncoderCrop(cfg.Crop, cfg.Width, cfg.Height); err != nil {
		return cfg, err
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
	if cfg.InitialQP == 0 && !explicitInitialQP {
		cfg.InitialQP = 26
	}
	if cfg.MinQP == 0 && !explicitMinQP {
		cfg.MinQP = 10
	}
	if cfg.MaxQP == 0 && !explicitMaxQP {
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
	if cfg.SliceCount > encoderMacroblockCount(cfg.Width, cfg.Height) {
		return cfg, encoderInvalid("slice count cannot exceed coded macroblock count")
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
		switch cfg.RTPPacketizationMode {
		case EncoderRTPPacketizationSingleNAL:
			if cfg.RTPMaxPayloadSize < 1 {
				return cfg, encoderInvalid("RTP max payload size must fit a NAL header")
			}
			if cfg.STAPA {
				return cfg, encoderUnsupported("STAP-A aggregation requires RTP packetization-mode 1")
			}
		case EncoderRTPPacketizationNonInterleaved:
			if cfg.RTPMaxPayloadSize < 3 {
				return cfg, encoderInvalid("RTP max payload size must leave room for FU-A headers")
			}
		default:
			return cfg, encoderInvalid("unknown RTP packetization mode")
		}
		if !cfg.DONDisabled {
			return cfg, encoderUnsupported("interleaved DON mode is not part of WebRTC RTP packetization")
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

func validateEncoderCrop(crop EncoderCrop, width int, height int) error {
	if crop.Left < 0 || crop.Right < 0 || crop.Top < 0 || crop.Bottom < 0 {
		return encoderInvalid("crop offsets cannot be negative")
	}
	if crop.Left%2 != 0 || crop.Right%2 != 0 || crop.Top%2 != 0 || crop.Bottom%2 != 0 {
		return encoderInvalid("I420 crop offsets must be even")
	}
	if crop.Left+crop.Right >= width || crop.Top+crop.Bottom >= height {
		return encoderInvalid("crop offsets must leave a visible frame")
	}
	return nil
}

func rtpTimestampIncrement(clock, frameRateNum, frameRateDen int) uint32 {
	if clock <= 0 || frameRateNum <= 0 || frameRateDen <= 0 {
		return 0
	}
	return uint32((clock * frameRateDen) / frameRateNum)
}

func (e *Encoder) encoderRTPTime(frame EncoderFrame) uint32 {
	if frame.PTS != 0 || !e.rtpTimeInitialized {
		return uint32(frame.PTS)
	}
	return e.nextRTPTime
}

func (e *Encoder) advanceEncoderRTPTime(frame EncoderFrame, rtpTime uint32) {
	e.nextRTPTime = rtpTime + encoderFrameRTPDuration(e.cfg, frame)
	e.rtpTimeInitialized = true
}

func encoderFrameRTPDuration(cfg EncoderConfig, frame EncoderFrame) uint32 {
	if frame.Duration > 0 {
		return uint32(frame.Duration)
	}
	if cfg.RTPTimestampIncrement != 0 {
		return cfg.RTPTimestampIncrement
	}
	return rtpTimestampIncrement(cfg.TimeBaseDen, cfg.FrameRateNum, cfg.FrameRateDen)
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
