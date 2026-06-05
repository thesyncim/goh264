// SPDX-License-Identifier: LGPL-2.1-or-later

package goh264_test

import (
	"bytes"
	"errors"
	"reflect"
	"testing"

	goh264 "github.com/thesyncim/goh264"
)

func TestEncoderDefaultRealtimeWebRTCConfig(t *testing.T) {
	cfg := goh264.DefaultEncoderConfig(640, 480)
	enc, err := goh264.NewEncoder(cfg)
	if err != nil {
		t.Fatalf("NewEncoder default: %v", err)
	}

	got := enc.Config()
	if got.Width != 640 || got.Height != 480 {
		t.Fatalf("dimensions = %dx%d, want 640x480", got.Width, got.Height)
	}
	if got.PixelFormat != goh264.EncoderPixelFormatI420 {
		t.Fatalf("pixel format = %v, want I420", got.PixelFormat)
	}
	if got.Profile != goh264.EncoderProfileConstrainedBaseline {
		t.Fatalf("profile = %v, want constrained baseline", got.Profile)
	}
	if got.EntropyMode != goh264.EncoderEntropyCAVLC || got.BFrames != 0 || got.MaxReferenceFrames != 1 {
		t.Fatalf("baseline realtime tools = entropy %v bframes %d refs %d, want CAVLC/0/1",
			got.EntropyMode, got.BFrames, got.MaxReferenceFrames)
	}
	if !got.ZeroLookahead || !got.Deterministic || got.Workers != 1 {
		t.Fatalf("latency controls = zero-lookahead %v deterministic %v workers %d, want true/true/1",
			got.ZeroLookahead, got.Deterministic, got.Workers)
	}
	if got.OutputFormat != goh264.EncoderOutputRTP ||
		got.RTPPacketizationMode != goh264.EncoderRTPPacketizationNonInterleaved ||
		got.RTPMaxPayloadSize != 1200 ||
		!got.DONDisabled ||
		got.RTPPayloadType != 96 {
		t.Fatalf("RTP controls = format %v mode %v payload %d don-disabled %v payload-type %d, want RTP/mode1/1200/true/96",
			got.OutputFormat, got.RTPPacketizationMode, got.RTPMaxPayloadSize, got.DONDisabled, got.RTPPayloadType)
	}
	if got.RTPTimestampIncrement != 3000 {
		t.Fatalf("RTP timestamp increment = %d, want 3000 for 30fps/90kHz", got.RTPTimestampIncrement)
	}
	if !got.RecoveryPointSEI {
		t.Fatal("default WebRTC encoder config should emit recovery-point SEI on recovery pictures")
	}
}

func TestEncoderRealtimeWebRTCRejectsInvalidConfigs(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*goh264.EncoderConfig)
		want   error
	}{
		{name: "zero width", mutate: func(c *goh264.EncoderConfig) { c.Width = 0 }, want: goh264.ErrInvalidData},
		{name: "odd I420 dimensions", mutate: func(c *goh264.EncoderConfig) { c.Width = 641 }, want: goh264.ErrInvalidData},
		{name: "undersized luma stride", mutate: func(c *goh264.EncoderConfig) { c.StrideY = 639 }, want: goh264.ErrInvalidData},
		{name: "unknown pixel format", mutate: func(c *goh264.EncoderConfig) { c.PixelFormat = goh264.EncoderPixelFormat(99) }, want: goh264.ErrUnsupported},
		{name: "main profile not admitted yet", mutate: func(c *goh264.EncoderConfig) { c.Profile = goh264.EncoderProfileMain }, want: goh264.ErrUnsupported},
		{name: "cabac not admitted yet", mutate: func(c *goh264.EncoderConfig) { c.EntropyMode = goh264.EncoderEntropyCABAC }, want: goh264.ErrUnsupported},
		{name: "8x8 transform not admitted yet", mutate: func(c *goh264.EncoderConfig) { c.Transform8x8 = true }, want: goh264.ErrUnsupported},
		{name: "multiple references not admitted yet", mutate: func(c *goh264.EncoderConfig) { c.MaxReferenceFrames = 2 }, want: goh264.ErrUnsupported},
		{name: "b frames disabled", mutate: func(c *goh264.EncoderConfig) { c.BFrames = 1 }, want: goh264.ErrUnsupported},
		{name: "bad bitrate", mutate: func(c *goh264.EncoderConfig) { c.TargetBitrate = 0 }, want: goh264.ErrInvalidData},
		{name: "max bitrate below target", mutate: func(c *goh264.EncoderConfig) { c.MaxBitrate = c.TargetBitrate - 1 }, want: goh264.ErrInvalidData},
		{name: "bad qp range", mutate: func(c *goh264.EncoderConfig) { c.MinQP = 40; c.MaxQP = 20 }, want: goh264.ErrInvalidData},
		{name: "deterministic multi worker", mutate: func(c *goh264.EncoderConfig) { c.Workers = 2 }, want: goh264.ErrInvalidData},
		{name: "idr interval beyond gop", mutate: func(c *goh264.EncoderConfig) { c.IDRInterval = c.GOPSize + 1 }, want: goh264.ErrInvalidData},
		{name: "intra refresh not admitted yet", mutate: func(c *goh264.EncoderConfig) { c.IntraRefresh = true }, want: goh264.ErrUnsupported},
		{name: "rtp payload too small", mutate: func(c *goh264.EncoderConfig) { c.RTPMaxPayloadSize = 2 }, want: goh264.ErrInvalidData},
		{name: "rtp packetization mode 0 not admitted yet", mutate: func(c *goh264.EncoderConfig) { c.RTPPacketizationMode = goh264.EncoderRTPPacketizationSingleNAL }, want: goh264.ErrUnsupported},
		{name: "don enabled not admitted", mutate: func(c *goh264.EncoderConfig) { c.DONDisabled = false }, want: goh264.ErrUnsupported},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := goh264.DefaultEncoderConfig(640, 480)
			tt.mutate(&cfg)
			if err := cfg.Validate(); !errors.Is(err, tt.want) {
				t.Fatalf("Validate error = %v, want %v", err, tt.want)
			}
			if _, err := goh264.NewEncoder(cfg); !errors.Is(err, tt.want) {
				t.Fatalf("NewEncoder error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestEncoderRuntimeControlsValidateAndReconfigure(t *testing.T) {
	enc, err := goh264.NewEncoder(goh264.DefaultEncoderConfig(640, 480))
	if err != nil {
		t.Fatalf("NewEncoder: %v", err)
	}

	if err := enc.SetBitrate(500_000, 700_000); err != nil {
		t.Fatalf("SetBitrate valid: %v", err)
	}
	if got := enc.Config(); got.TargetBitrate != 500_000 || got.MaxBitrate != 700_000 {
		t.Fatalf("bitrate config = %d/%d, want 500000/700000", got.TargetBitrate, got.MaxBitrate)
	}
	if err := enc.SetBitrate(600_000, 0); err != nil {
		t.Fatalf("SetBitrate max default: %v", err)
	}
	if got := enc.Config(); got.TargetBitrate != 600_000 || got.MaxBitrate != 600_000 {
		t.Fatalf("defaulted max bitrate config = %d/%d, want 600000/600000", got.TargetBitrate, got.MaxBitrate)
	}

	before := enc.Config()
	if err := enc.SetBitrate(0, 0); !errors.Is(err, goh264.ErrInvalidData) {
		t.Fatalf("SetBitrate invalid error = %v, want ErrInvalidData", err)
	}
	if got := enc.Config(); got.TargetBitrate != before.TargetBitrate || got.MaxBitrate != before.MaxBitrate {
		t.Fatalf("invalid SetBitrate mutated config = %+v, want %+v", got, before)
	}

	if err := enc.SetFrameRate(60, 1); err != nil {
		t.Fatalf("SetFrameRate valid: %v", err)
	}
	if got := enc.Config(); got.FrameRateNum != 60 || got.FrameRateDen != 1 || got.RTPTimestampIncrement != 1500 {
		t.Fatalf("frame rate config = %d/%d rtp=%d, want 60/1 rtp=1500",
			got.FrameRateNum, got.FrameRateDen, got.RTPTimestampIncrement)
	}

	if err := enc.SetRTPMaxPayloadSize(1000); err != nil {
		t.Fatalf("SetRTPMaxPayloadSize valid: %v", err)
	}
	if got := enc.Config(); got.RTPMaxPayloadSize != 1000 {
		t.Fatalf("rtp max payload = %d, want 1000", got.RTPMaxPayloadSize)
	}

	noParameterSetsBeforeIDR := false
	noRecoveryPointSEI := false
	if err := enc.Reconfigure(goh264.EncoderReconfigure{
		TargetBitrate:     800_000,
		MaxBitrate:        900_000,
		FrameRateNum:      24,
		FrameRateDen:      1,
		Width:             1280,
		Height:            720,
		RTPMaxPayloadSize: 900,
		MaxFrameSize:      80_000,
		MaxEncodeTimeUS:   5_000,
		Preset:            goh264.EncoderPresetBalanced,
		ForceIDR:          true,
		SPSPPSBeforeIDR:   &noParameterSetsBeforeIDR,
		RecoveryPointSEI:  &noRecoveryPointSEI,
	}); err != nil {
		t.Fatalf("Reconfigure valid: %v", err)
	}
	got := enc.Config()
	if got.TargetBitrate != 800_000 || got.MaxBitrate != 900_000 ||
		got.FrameRateNum != 24 || got.FrameRateDen != 1 ||
		got.Width != 1280 || got.Height != 720 ||
		got.RTPMaxPayloadSize != 900 ||
		got.MaxFrameSize != 80_000 ||
		got.MaxEncodeTimeUS != 5_000 ||
		got.Preset != goh264.EncoderPresetBalanced ||
		got.SPSPPSBeforeIDR ||
		got.RecoveryPointSEI {
		t.Fatalf("reconfigured encoder = %+v, want realtime update applied", got)
	}
	if !enc.PendingIDR() {
		t.Fatal("Reconfigure ForceIDR did not queue an IDR")
	}

	before = enc.Config()
	if err := enc.Reconfigure(goh264.EncoderReconfigure{MaxBitrate: 1}); !errors.Is(err, goh264.ErrInvalidData) {
		t.Fatalf("Reconfigure invalid error = %v, want ErrInvalidData", err)
	}
	if got := enc.Config(); got != before {
		t.Fatalf("invalid Reconfigure mutated config = %+v, want %+v", got, before)
	}
}

func TestEncoderKeyframeRequestsQueueIDR(t *testing.T) {
	for _, tt := range []struct {
		name string
		call func(*goh264.Encoder)
	}{
		{name: "force idr", call: func(e *goh264.Encoder) { e.ForceIDR() }},
		{name: "pli", call: func(e *goh264.Encoder) { e.HandlePLI() }},
		{name: "fir", call: func(e *goh264.Encoder) { e.HandleFIR() }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			enc, err := goh264.NewEncoder(goh264.DefaultEncoderConfig(640, 480))
			if err != nil {
				t.Fatalf("NewEncoder: %v", err)
			}
			tt.call(enc)
			if !enc.PendingIDR() {
				t.Fatal("keyframe request did not queue an IDR")
			}
		})
	}
}

func TestEncoderParameterSetsExposeWebRTCHeaders(t *testing.T) {
	cfg := goh264.DefaultEncoderConfig(638, 478)
	cfg.FrameRateNum = 30000
	cfg.FrameRateDen = 1001
	cfg.InitialQP = 24
	cfg.Color.SARNum = 1
	cfg.Color.SARDen = 1
	cfg.Color.FullRange = true
	cfg.Color.ColorPrimaries = 1
	cfg.Color.ColorTransfer = 1
	cfg.Color.ColorMatrix = 1

	enc, err := goh264.NewEncoder(cfg)
	if err != nil {
		t.Fatalf("NewEncoder: %v", err)
	}
	headers, err := enc.ParameterSets()
	if err != nil {
		t.Fatalf("ParameterSets: %v", err)
	}
	if len(headers.SPS) == 0 || headers.SPS[0]&0x1f != 7 {
		t.Fatalf("SPS NAL = %x", headers.SPS)
	}
	if len(headers.PPS) == 0 || headers.PPS[0]&0x1f != 8 {
		t.Fatalf("PPS NAL = %x", headers.PPS)
	}
	if !bytes.Contains(headers.AnnexB, headers.SPS) || !bytes.Contains(headers.AnnexB, headers.PPS) {
		t.Fatalf("Annex B headers do not contain SPS/PPS: %x", headers.AnnexB)
	}

	info, err := goh264.NewDecoder().ParseHeadersAnnexB(headers.AnnexB)
	if err != nil {
		t.Fatalf("ParseHeadersAnnexB: %v", err)
	}
	if info.Profile != "Constrained Baseline" || info.ProfileIDC != 66 || info.LevelIDC != 31 ||
		info.Width != 638 || info.Height != 478 ||
		info.SARNum != 1 || info.SARDen != 1 ||
		info.VideoFullRangeFlag != 1 ||
		info.ColorPrimaries != 1 ||
		info.ColorTransfer != 1 ||
		info.ColorMatrix != 1 ||
		info.TimingInfoPresentFlag != 1 ||
		info.NumUnitsInTick != 1001 ||
		info.TimeScale != 60000 ||
		info.FixedFrameRateFlag != 1 {
		t.Fatalf("Annex B stream info = %+v", info)
	}

	avcc, err := goh264.NewDecoder().ParseAVCDecoderConfigurationRecord(headers.AVCDecoderConfigurationRecord)
	if err != nil {
		t.Fatalf("ParseAVCDecoderConfigurationRecord: %v", err)
	}
	if avcc.NALLengthSize != 4 || avcc.StreamInfo.Width != 638 || avcc.StreamInfo.Height != 478 ||
		avcc.StreamInfo.Profile != "Constrained Baseline" {
		t.Fatalf("avcC = %+v", avcc)
	}
}

func TestEncoderRecoveryPointSEIExposesWebRTCRecoverySignal(t *testing.T) {
	enc, err := goh264.NewEncoder(goh264.DefaultEncoderConfig(16, 16))
	if err != nil {
		t.Fatalf("NewEncoder: %v", err)
	}
	sei, err := enc.RecoveryPointSEI(0)
	if err != nil {
		t.Fatalf("RecoveryPointSEI: %v", err)
	}
	if len(sei.NAL) == 0 || sei.NAL[0]&0x1f != 6 {
		t.Fatalf("SEI NAL = %x, want type 6", sei.NAL)
	}
	if !bytes.Contains(sei.AnnexB, sei.NAL) || !bytes.Contains(sei.AVC, sei.NAL) {
		t.Fatalf("SEI packet surfaces do not contain raw NAL: annexb=%x avc=%x nal=%x", sei.AnnexB, sei.AVC, sei.NAL)
	}
	sei.NAL[0] = 0
	again, err := enc.RecoveryPointSEI(0)
	if err != nil {
		t.Fatalf("RecoveryPointSEI after caller mutation: %v", err)
	}
	if len(again.NAL) == 0 || again.NAL[0]&0x1f != 6 {
		t.Fatalf("SEI NAL aliases caller mutation: %x", again.NAL)
	}
	sei = again

	annexB := insertAnnexBNALBeforeVCL(t, decodeHexFixture(t, black16IPAnnexBHex), sei.NAL, 1)
	frames, err := goh264.NewDecoder().DecodeAnnexBFrames(annexB)
	if err != nil {
		t.Fatalf("DecodeAnnexBFrames: %v", err)
	}
	if len(frames) != 2 || !frames[0].KeyFrame || !frames[1].KeyFrame {
		t.Fatalf("Annex B keyframes = len %d %v", len(frames), frameKeyFlags(frames))
	}
	if frames[1].SideData.RecoveryPoint == nil || frames[1].SideData.RecoveryPoint.RecoveryFrameCount != 0 {
		t.Fatalf("Annex B recovery point = %+v", frames[1].SideData.RecoveryPoint)
	}

	config, samples := annexBToAVCConfigAndSamples(t, decodeHexFixture(t, black16IPAnnexBHex), 4)
	if len(samples) != 2 {
		t.Fatalf("samples = %d, want 2", len(samples))
	}
	samples[1] = append(append([]byte(nil), sei.AVC...), samples[1]...)
	avcDec := goh264.NewDecoder()
	if _, err := avcDec.ParseAVCDecoderConfigurationRecord(config); err != nil {
		t.Fatalf("ParseAVCDecoderConfigurationRecord: %v", err)
	}
	first, err := avcDec.DecodeConfiguredAVC(samples[0])
	if err != nil {
		t.Fatalf("DecodeConfiguredAVC first: %v", err)
	}
	second, err := avcDec.DecodeConfiguredAVC(samples[1])
	if err != nil {
		t.Fatalf("DecodeConfiguredAVC second: %v", err)
	}
	if !first.KeyFrame || !second.KeyFrame ||
		second.SideData.RecoveryPoint == nil ||
		second.SideData.RecoveryPoint.RecoveryFrameCount != 0 {
		t.Fatalf("AVC recovery frames key=%t/%t side=%+v", first.KeyFrame, second.KeyFrame, second.SideData.RecoveryPoint)
	}

	delayed, err := enc.RecoveryPointSEI(4)
	if err != nil {
		t.Fatalf("RecoveryPointSEI nonzero: %v", err)
	}
	samples[1] = append(append([]byte(nil), delayed.AVC...), samples[1][len(sei.AVC):]...)
	avcDec = goh264.NewDecoder()
	if _, err := avcDec.ParseAVCDecoderConfigurationRecord(config); err != nil {
		t.Fatalf("ParseAVCDecoderConfigurationRecord delayed: %v", err)
	}
	if _, err := avcDec.DecodeConfiguredAVC(samples[0]); err != nil {
		t.Fatalf("DecodeConfiguredAVC delayed first: %v", err)
	}
	second, err = avcDec.DecodeConfiguredAVC(samples[1])
	if err != nil {
		t.Fatalf("DecodeConfiguredAVC delayed second: %v", err)
	}
	if second.KeyFrame || second.SideData.RecoveryPoint == nil || second.SideData.RecoveryPoint.RecoveryFrameCount != 4 {
		t.Fatalf("delayed recovery frame key=%t side=%+v, want non-key recovery count 4", second.KeyFrame, second.SideData.RecoveryPoint)
	}
}

func TestEncoderRecoveryPointSEIRejectsInvalidFrameCount(t *testing.T) {
	enc, err := goh264.NewEncoder(goh264.DefaultEncoderConfig(16, 16))
	if err != nil {
		t.Fatalf("NewEncoder: %v", err)
	}
	if _, err := enc.RecoveryPointSEI(1 << 16); !errors.Is(err, goh264.ErrInvalidData) {
		t.Fatalf("RecoveryPointSEI invalid error = %v, want ErrInvalidData", err)
	}
}

func TestEncoderEncodeIntoValidatesFrameBeforeUnsupportedBitstream(t *testing.T) {
	enc, err := goh264.NewEncoder(goh264.DefaultEncoderConfig(16, 16))
	if err != nil {
		t.Fatalf("NewEncoder: %v", err)
	}

	frame := validI420EncoderFrame(16, 16)
	if _, err := enc.EncodeInto(make([]byte, 0, 1024), frame); !errors.Is(err, goh264.ErrUnsupported) {
		t.Fatalf("EncodeInto valid-frame error = %v, want ErrUnsupported until bitstream generation lands", err)
	}

	bad := frame
	bad.Y = nil
	if _, err := enc.EncodeInto(nil, bad); !errors.Is(err, goh264.ErrInvalidData) {
		t.Fatalf("EncodeInto missing luma error = %v, want ErrInvalidData", err)
	}

	bad = frame
	bad.Width = 32
	if _, err := enc.Encode(bad); !errors.Is(err, goh264.ErrInvalidData) {
		t.Fatalf("Encode mismatched dimensions error = %v, want ErrInvalidData", err)
	}
}

func TestEncoderRealtimeWebRTCControlSurfaceCoversRoadmap(t *testing.T) {
	cfgType := reflect.TypeOf(goh264.EncoderConfig{})
	for _, field := range []string{
		"Width", "Height", "StrideY", "StrideCb", "StrideCr", "PixelFormat",
		"Crop", "FrameRateNum", "FrameRateDen", "TimeBaseNum", "TimeBaseDen", "Color",
		"Profile", "LevelIDC", "EntropyMode", "DeblockMode", "Transform8x8",
		"MaxReferenceFrames", "BFrames", "SPSPPSMode",
		"RateControl", "TargetBitrate", "MaxBitrate", "VBVBufferSize", "MaxFrameSize",
		"InitialQP", "MinQP", "MaxQP", "Preset",
		"ZeroLookahead", "FrameDrop", "MaxEncodeTimeUS", "SliceCount", "SliceMaxBytes",
		"Workers", "Deterministic",
		"GOPSize", "IDRInterval", "SPSPPSBeforeIDR", "RecoveryPointSEI", "IntraRefresh",
		"OutputFormat", "RTPMaxPayloadSize", "RTPPacketizationMode", "STAPA", "DONDisabled",
		"RTPPayloadType", "RTPSSRC", "RTPTimestampIncrement",
	} {
		if _, ok := cfgType.FieldByName(field); !ok {
			t.Fatalf("EncoderConfig missing roadmap control field %s", field)
		}
	}

	encType := reflect.TypeOf(&goh264.Encoder{})
	for _, method := range []string{
		"Config", "ParameterSets", "Encode", "EncodeInto", "ForceIDR", "HandlePLI", "HandleFIR",
		"PendingIDR", "RecoveryPointSEI", "SetBitrate", "SetFrameRate", "SetRTPMaxPayloadSize", "Reconfigure",
	} {
		if _, ok := encType.MethodByName(method); !ok {
			t.Fatalf("Encoder missing runtime control method %s", method)
		}
	}
}

func frameKeyFlags(frames []*goh264.Frame) []bool {
	out := make([]bool, len(frames))
	for i, frame := range frames {
		out[i] = frame.KeyFrame
	}
	return out
}

func validI420EncoderFrame(width, height int) goh264.EncoderFrame {
	chromaWidth := width / 2
	chromaHeight := height / 2
	return goh264.EncoderFrame{
		Y:        make([]byte, width*height),
		Cb:       make([]byte, chromaWidth*chromaHeight),
		Cr:       make([]byte, chromaWidth*chromaHeight),
		StrideY:  width,
		StrideCb: chromaWidth,
		StrideCr: chromaWidth,
		Width:    width,
		Height:   height,
		Duration: 3000,
	}
}
