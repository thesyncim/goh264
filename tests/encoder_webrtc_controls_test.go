// SPDX-License-Identifier: LGPL-2.1-or-later

package goh264_test

import (
	"bytes"
	"crypto/md5"
	"encoding/binary"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"

	goh264 "github.com/thesyncim/goh264"
	"github.com/thesyncim/goh264/internal/h264"
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
		{name: "negative crop", mutate: func(c *goh264.EncoderConfig) { c.Crop.Left = -2 }, want: goh264.ErrInvalidData},
		{name: "odd I420 crop", mutate: func(c *goh264.EncoderConfig) { c.Crop.Left = 1 }, want: goh264.ErrInvalidData},
		{name: "crop consumes width", mutate: func(c *goh264.EncoderConfig) { c.Crop.Left = c.Width / 2; c.Crop.Right = c.Width / 2 }, want: goh264.ErrInvalidData},
		{name: "slice count beyond macroblocks", mutate: func(c *goh264.EncoderConfig) { c.Width = 16; c.Height = 16; c.SliceCount = 2 }, want: goh264.ErrInvalidData},
		{name: "negative slice byte target", mutate: func(c *goh264.EncoderConfig) { c.SliceMaxBytes = -1 }, want: goh264.ErrInvalidData},
		{name: "deterministic multi worker", mutate: func(c *goh264.EncoderConfig) { c.Workers = 2 }, want: goh264.ErrInvalidData},
		{name: "idr interval beyond gop", mutate: func(c *goh264.EncoderConfig) { c.IDRInterval = c.GOPSize + 1 }, want: goh264.ErrInvalidData},
		{name: "intra refresh not admitted yet", mutate: func(c *goh264.EncoderConfig) { c.IntraRefresh = true }, want: goh264.ErrUnsupported},
		{name: "rtp payload too small", mutate: func(c *goh264.EncoderConfig) { c.RTPMaxPayloadSize = 2 }, want: goh264.ErrInvalidData},
		{name: "rtp payload type too large", mutate: func(c *goh264.EncoderConfig) { c.RTPPayloadType = 128 }, want: goh264.ErrInvalidData},
		{name: "stap-a requires packetization mode 1", mutate: func(c *goh264.EncoderConfig) {
			c.RTPPacketizationMode = goh264.EncoderRTPPacketizationSingleNAL
			c.STAPA = true
		}, want: goh264.ErrUnsupported},
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
		SliceCount:        2,
		SliceMaxBytes:     700,
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
		got.SliceCount != 2 ||
		got.SliceMaxBytes != 700 ||
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

func TestEncoderSPSPPSCadenceModesControlIDRHeaders(t *testing.T) {
	tests := []struct {
		name       string
		mode       goh264.EncoderSPSPPSMode
		beforeIDR  bool
		wantIDRNAL []uint8
	}{
		{
			name:       "in-band keyframes",
			mode:       goh264.EncoderSPSPPSInBandKeyframes,
			beforeIDR:  true,
			wantIDRNAL: []uint8{7, 8, 5},
		},
		{
			name:       "in-band keyframes disabled",
			mode:       goh264.EncoderSPSPPSInBandKeyframes,
			beforeIDR:  false,
			wantIDRNAL: []uint8{5},
		},
		{
			name:       "out-of-band suppresses in-band headers",
			mode:       goh264.EncoderSPSPPSOutOfBand,
			beforeIDR:  true,
			wantIDRNAL: []uint8{5},
		},
		{
			name:       "every IDR overrides boolean",
			mode:       goh264.EncoderSPSPPSEveryIDR,
			beforeIDR:  false,
			wantIDRNAL: []uint8{7, 8, 5},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := goh264.DefaultEncoderConfig(16, 16)
			cfg.OutputFormat = goh264.EncoderOutputAnnexB
			cfg.RTPMaxPayloadSize = 0
			cfg.DeblockMode = goh264.EncoderDeblockDisabled
			cfg.SPSPPSMode = tt.mode
			cfg.SPSPPSBeforeIDR = tt.beforeIDR
			enc, err := goh264.NewEncoder(cfg)
			if err != nil {
				t.Fatalf("NewEncoder: %v", err)
			}

			frame := patternedI420EncoderFrame(16, 16)
			first, err := enc.Encode(frame)
			if err != nil {
				t.Fatalf("Encode first IDR: %v", err)
			}
			if !first.IDR || !first.KeyFrame {
				t.Fatalf("first frame IDR/key = %v/%v, want IDR keyframe", first.IDR, first.KeyFrame)
			}
			assertEncoderNALTypes(t, first.NALUnits, tt.wantIDRNAL)

			frame.PTS += int64(cfg.RTPTimestampIncrement)
			enc.ForceIDR()
			forced, err := enc.Encode(frame)
			if err != nil {
				t.Fatalf("Encode forced IDR: %v", err)
			}
			if !forced.IDR || !forced.KeyFrame {
				t.Fatalf("forced frame IDR/key = %v/%v, want IDR keyframe", forced.IDR, forced.KeyFrame)
			}
			assertEncoderNALTypes(t, forced.NALUnits, tt.wantIDRNAL)
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

func TestEncoderParameterSetsExposeWebRTCCrop(t *testing.T) {
	cfg := goh264.DefaultEncoderConfig(640, 480)
	cfg.Crop = goh264.EncoderCrop{Left: 2, Right: 4, Top: 6, Bottom: 8}

	enc, err := goh264.NewEncoder(cfg)
	if err != nil {
		t.Fatalf("NewEncoder: %v", err)
	}
	headers, err := enc.ParameterSets()
	if err != nil {
		t.Fatalf("ParameterSets: %v", err)
	}

	wantWidth := cfg.Width - cfg.Crop.Left - cfg.Crop.Right
	wantHeight := cfg.Height - cfg.Crop.Top - cfg.Crop.Bottom
	info, err := goh264.NewDecoder().ParseHeadersAnnexB(headers.AnnexB)
	if err != nil {
		t.Fatalf("ParseHeadersAnnexB: %v", err)
	}
	if info.Width != wantWidth || info.Height != wantHeight {
		t.Fatalf("cropped Annex B dimensions = %dx%d, want %dx%d",
			info.Width, info.Height, wantWidth, wantHeight)
	}

	avcc, err := goh264.NewDecoder().ParseAVCDecoderConfigurationRecord(headers.AVCDecoderConfigurationRecord)
	if err != nil {
		t.Fatalf("ParseAVCDecoderConfigurationRecord: %v", err)
	}
	if avcc.StreamInfo.Width != wantWidth || avcc.StreamInfo.Height != wantHeight {
		t.Fatalf("cropped avcC dimensions = %dx%d, want %dx%d",
			avcc.StreamInfo.Width, avcc.StreamInfo.Height, wantWidth, wantHeight)
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

func TestEncoderEncodeAnnexBIDRIntraPCMDecodesThroughLocalAndFFmpeg(t *testing.T) {
	cfg := goh264.DefaultEncoderConfig(18, 18)
	cfg.OutputFormat = goh264.EncoderOutputAnnexB
	cfg.DeblockMode = goh264.EncoderDeblockDisabled
	cfg.RTPMaxPayloadSize = 0
	enc, err := goh264.NewEncoder(cfg)
	if err != nil {
		t.Fatalf("NewEncoder: %v", err)
	}
	frame := patternedI420EncoderFrame(18, 18)
	want := appendI420FrameBytes(nil, frame)

	out, err := enc.Encode(frame)
	if err != nil {
		t.Fatalf("Encode Annex B IDR: %v", err)
	}
	if !out.KeyFrame || !out.IDR || out.PTS != frame.PTS || out.DTS != frame.PTS {
		t.Fatalf("encoded frame metadata key=%v idr=%v pts=%d dts=%d", out.KeyFrame, out.IDR, out.PTS, out.DTS)
	}
	assertEncoderNALTypes(t, out.NALUnits, []uint8{7, 8, 5})
	if len(out.RTPPackets) != 0 {
		t.Fatalf("Annex B output unexpectedly has RTP packets: %d", len(out.RTPPackets))
	}

	decoded, err := goh264.NewDecoder().DecodeAnnexBFrames(out.Data)
	if err != nil {
		t.Fatalf("DecodeAnnexBFrames encoded IDR: %v", err)
	}
	assertDecodedEncoderFrameBytes(t, decoded, want)
	assertFFmpegRawVideoOracle(t, out.Data, want)
	if enc.PendingIDR() {
		t.Fatal("successful IDR encode left PendingIDR set")
	}
}

func TestEncoderEncodeCroppedAnnexBIDRIntraPCMDecodesVisibleFrame(t *testing.T) {
	cfg := goh264.DefaultEncoderConfig(20, 20)
	cfg.OutputFormat = goh264.EncoderOutputAnnexB
	cfg.DeblockMode = goh264.EncoderDeblockDisabled
	cfg.RTPMaxPayloadSize = 0
	cfg.Crop = goh264.EncoderCrop{Left: 2, Right: 2, Top: 2, Bottom: 2}
	enc, err := goh264.NewEncoder(cfg)
	if err != nil {
		t.Fatalf("NewEncoder: %v", err)
	}
	frame := patternedI420EncoderFrame(20, 20)
	want := appendCroppedI420FrameBytes(nil, frame, cfg.Crop)

	out, err := enc.Encode(frame)
	if err != nil {
		t.Fatalf("Encode cropped Annex B IDR: %v", err)
	}
	assertEncoderNALTypes(t, out.NALUnits, []uint8{7, 8, 5})

	decoded, err := goh264.NewDecoder().DecodeAnnexBFrames(out.Data)
	if err != nil {
		t.Fatalf("DecodeAnnexBFrames cropped IDR: %v", err)
	}
	if len(decoded) != 1 ||
		decoded[0].Width != 16 || decoded[0].Height != 16 ||
		decoded[0].CropLeft != 2 || decoded[0].CropTop != 2 {
		t.Fatalf("decoded crop geometry = len %d frame %+v, want 16x16 crop 2,2", len(decoded), decoded[0])
	}
	assertDecodedEncoderFrameBytes(t, decoded, want)
	assertFFmpegRawVideoOracle(t, out.Data, want)
}

func TestEncoderEncodeAVCIDRIntraPCMDecodesThroughConfiguredSurface(t *testing.T) {
	cfg := goh264.DefaultEncoderConfig(16, 16)
	cfg.OutputFormat = goh264.EncoderOutputAVC
	cfg.DeblockMode = goh264.EncoderDeblockDisabled
	cfg.RTPMaxPayloadSize = 0
	enc, err := goh264.NewEncoder(cfg)
	if err != nil {
		t.Fatalf("NewEncoder: %v", err)
	}
	frame := patternedI420EncoderFrame(16, 16)

	out, err := enc.Encode(frame)
	if err != nil {
		t.Fatalf("Encode AVC IDR: %v", err)
	}
	assertEncoderNALTypes(t, out.NALUnits, []uint8{7, 8, 5})
	decoded, err := goh264.NewDecoder().DecodeAVCFrames(out.Data, 4)
	if err != nil {
		t.Fatalf("DecodeAVCFrames encoded IDR: %v", err)
	}
	assertDecodedEncoderFrameBytes(t, decoded, appendI420FrameBytes(nil, frame))
}

func TestEncoderEncodeIdenticalSecondFrameUsesPSkipReference(t *testing.T) {
	cfg := goh264.DefaultEncoderConfig(18, 18)
	cfg.OutputFormat = goh264.EncoderOutputAnnexB
	cfg.DeblockMode = goh264.EncoderDeblockDisabled
	cfg.RTPMaxPayloadSize = 0
	enc, err := goh264.NewEncoder(cfg)
	if err != nil {
		t.Fatalf("NewEncoder: %v", err)
	}
	frame := patternedI420EncoderFrame(18, 18)
	wantFrame := appendI420FrameBytes(nil, frame)

	first, err := enc.Encode(frame)
	if err != nil {
		t.Fatalf("Encode first IDR: %v", err)
	}
	assertEncoderNALTypes(t, first.NALUnits, []uint8{7, 8, 5})

	secondFrame := frame
	secondFrame.PTS += int64(cfg.RTPTimestampIncrement)
	second, err := enc.Encode(secondFrame)
	if err != nil {
		t.Fatalf("Encode second identical P-skip: %v", err)
	}
	if second.KeyFrame || second.IDR || second.PTS != secondFrame.PTS || second.DTS != secondFrame.PTS {
		t.Fatalf("second frame metadata key=%v idr=%v pts=%d dts=%d",
			second.KeyFrame, second.IDR, second.PTS, second.DTS)
	}
	assertEncoderNALTypes(t, second.NALUnits, []uint8{1})

	dec := goh264.NewDecoder()
	decodedFirst, err := dec.DecodeFrames(first.Data)
	if err != nil {
		t.Fatalf("Decode first IDR: %v", err)
	}
	assertDecodedEncoderFrameBytes(t, decodedFirst, wantFrame)
	decodedSecond, err := dec.DecodeFrames(second.Data)
	if err != nil {
		t.Fatalf("Decode second P-skip: %v", err)
	}
	assertDecodedEncoderFrameBytes(t, decodedSecond, wantFrame)

	stream := append(append([]byte(nil), first.Data...), second.Data...)
	wantStream := append(append([]byte(nil), wantFrame...), wantFrame...)
	assertFFmpegRawVideoOracle(t, stream, wantStream)
}

func TestEncoderEncodeChangedSecondFrameUsesPIntraPCM(t *testing.T) {
	cfg := goh264.DefaultEncoderConfig(16, 16)
	cfg.OutputFormat = goh264.EncoderOutputAnnexB
	cfg.DeblockMode = goh264.EncoderDeblockDisabled
	cfg.RTPMaxPayloadSize = 0
	enc, err := goh264.NewEncoder(cfg)
	if err != nil {
		t.Fatalf("NewEncoder: %v", err)
	}
	firstFrame := patternedI420EncoderFrame(16, 16)
	first, err := enc.Encode(firstFrame)
	if err != nil {
		t.Fatalf("Encode first IDR: %v", err)
	}
	assertEncoderNALTypes(t, first.NALUnits, []uint8{7, 8, 5})

	secondFrame := patternedI420EncoderFrame(16, 16)
	secondFrame.Y[0] ^= 0x7f
	secondFrame.PTS += int64(cfg.RTPTimestampIncrement)
	second, err := enc.Encode(secondFrame)
	if err != nil {
		t.Fatalf("Encode changed second frame: %v", err)
	}
	if second.KeyFrame || second.IDR {
		t.Fatalf("changed second frame key=%v idr=%v, want non-IDR P IntraPCM", second.KeyFrame, second.IDR)
	}
	assertEncoderNALTypes(t, second.NALUnits, []uint8{6, 1})

	dec := goh264.NewDecoder()
	decodedFirst, err := dec.DecodeFrames(first.Data)
	if err != nil {
		t.Fatalf("Decode first IDR: %v", err)
	}
	assertDecodedEncoderFrameBytes(t, decodedFirst, appendI420FrameBytes(nil, firstFrame))
	decodedSecond, err := dec.DecodeFrames(second.Data)
	if err != nil {
		t.Fatalf("Decode changed P IntraPCM: %v", err)
	}
	assertDecodedEncoderFrameBytes(t, decodedSecond, appendI420FrameBytes(nil, secondFrame))
	if !decodedSecond[0].KeyFrame ||
		decodedSecond[0].SideData.RecoveryPoint == nil ||
		decodedSecond[0].SideData.RecoveryPoint.RecoveryFrameCount != 0 {
		t.Fatalf("changed P recovery side data key=%v recovery=%+v, want immediate recovery point",
			decodedSecond[0].KeyFrame, decodedSecond[0].SideData.RecoveryPoint)
	}

	stream := append(append([]byte(nil), first.Data...), second.Data...)
	wantStream := appendI420FrameBytes(nil, firstFrame)
	wantStream = appendI420FrameBytes(wantStream, secondFrame)
	assertFFmpegRawVideoOracle(t, stream, wantStream)
}

func TestEncoderEncodeChangedPIntraPCMRecoveryPointSEIForAVCAndRTP(t *testing.T) {
	for _, tt := range []struct {
		name   string
		format goh264.EncoderOutputFormat
	}{
		{name: "avc", format: goh264.EncoderOutputAVC},
		{name: "rtp", format: goh264.EncoderOutputRTP},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cfg := goh264.DefaultEncoderConfig(16, 16)
			cfg.OutputFormat = tt.format
			cfg.DeblockMode = goh264.EncoderDeblockDisabled
			if tt.format != goh264.EncoderOutputRTP {
				cfg.RTPMaxPayloadSize = 0
			}
			enc, err := goh264.NewEncoder(cfg)
			if err != nil {
				t.Fatalf("NewEncoder: %v", err)
			}

			firstFrame := patternedI420EncoderFrame(16, 16)
			first, err := enc.Encode(firstFrame)
			if err != nil {
				t.Fatalf("Encode first IDR: %v", err)
			}
			secondFrame := patternedI420EncoderFrame(16, 16)
			secondFrame.Y[0] ^= 0x31
			secondFrame.PTS += int64(cfg.RTPTimestampIncrement)
			second, err := enc.Encode(secondFrame)
			if err != nil {
				t.Fatalf("Encode changed P IntraPCM: %v", err)
			}
			assertEncoderNALTypes(t, second.NALUnits, []uint8{6, 1})

			dec := goh264.NewDecoder()
			var decodedFirst, decodedSecond []*goh264.Frame
			switch tt.format {
			case goh264.EncoderOutputAVC:
				headers, err := enc.ParameterSets()
				if err != nil {
					t.Fatalf("ParameterSets: %v", err)
				}
				if _, err := dec.ParseAVCDecoderConfigurationRecord(headers.AVCDecoderConfigurationRecord); err != nil {
					t.Fatalf("ParseAVCDecoderConfigurationRecord: %v", err)
				}
				decodedFirst, err = dec.DecodeConfiguredAVCFrames(first.Data)
				if err != nil {
					t.Fatalf("DecodeConfiguredAVCFrames first: %v", err)
				}
				decodedSecond, err = dec.DecodeConfiguredAVCFrames(second.Data)
				if err != nil {
					t.Fatalf("DecodeConfiguredAVCFrames second: %v", err)
				}
			case goh264.EncoderOutputRTP:
				if len(second.RTPPackets) < 2 || second.RTPPackets[0].Payload[0]&0x1f != 6 || second.RTPPackets[0].Marker {
					t.Fatalf("second RTP packets do not lead with non-marker SEI: %+v", second.RTPPackets)
				}
				decodedFirst, err = dec.DecodeFrames(annexBFromEncoderRTPPackets(t, first.RTPPackets))
				if err != nil {
					t.Fatalf("DecodeFrames reassembled first RTP: %v", err)
				}
				decodedSecond, err = dec.DecodeFrames(annexBFromEncoderRTPPackets(t, second.RTPPackets))
				if err != nil {
					t.Fatalf("DecodeFrames reassembled second RTP: %v", err)
				}
			default:
				t.Fatalf("unexpected format %v", tt.format)
			}
			assertDecodedEncoderFrameBytes(t, decodedFirst, appendI420FrameBytes(nil, firstFrame))
			assertDecodedEncoderFrameBytes(t, decodedSecond, appendI420FrameBytes(nil, secondFrame))
			if !decodedSecond[0].KeyFrame ||
				decodedSecond[0].SideData.RecoveryPoint == nil ||
				decodedSecond[0].SideData.RecoveryPoint.RecoveryFrameCount != 0 {
				t.Fatalf("decoded recovery side data key=%v recovery=%+v, want immediate recovery point",
					decodedSecond[0].KeyFrame, decodedSecond[0].SideData.RecoveryPoint)
			}
		})
	}
}

func TestEncoderSliceCountSplitsIDRPSkipAndPIntraPCMAccessUnits(t *testing.T) {
	cfg := goh264.DefaultEncoderConfig(48, 16)
	cfg.OutputFormat = goh264.EncoderOutputAnnexB
	cfg.DeblockMode = goh264.EncoderDeblockDisabled
	cfg.RTPMaxPayloadSize = 0
	cfg.SliceCount = 3
	enc, err := goh264.NewEncoder(cfg)
	if err != nil {
		t.Fatalf("NewEncoder: %v", err)
	}

	firstFrame := patternedI420EncoderFrame(48, 16)
	headers, err := enc.ParameterSets()
	if err != nil {
		t.Fatalf("ParameterSets: %v", err)
	}
	first, err := enc.Encode(firstFrame)
	if err != nil {
		t.Fatalf("Encode first multi-slice IDR: %v", err)
	}
	assertEncoderNALTypes(t, first.NALUnits, []uint8{7, 8, 5, 5, 5})
	assertEncoderVCLFirstMBs(t, first.Data, []uint8{5, 5, 5}, []uint32{0, 1, 2})

	dec := goh264.NewDecoder()
	decodedFirst, err := dec.DecodeFrames(first.Data)
	if err != nil {
		t.Fatalf("Decode first multi-slice IDR: %v", err)
	}
	assertDecodedEncoderFrameBytes(t, decodedFirst, appendI420FrameBytes(nil, firstFrame))

	secondFrame := firstFrame
	secondFrame.PTS += int64(cfg.RTPTimestampIncrement)
	second, err := enc.Encode(secondFrame)
	if err != nil {
		t.Fatalf("Encode multi-slice P-skip: %v", err)
	}
	assertEncoderNALTypes(t, second.NALUnits, []uint8{1, 1, 1})
	assertEncoderVCLFirstMBs(t, append(append([]byte(nil), headers.AnnexB...), second.Data...), []uint8{1, 1, 1}, []uint32{0, 1, 2})
	decodedSecond, err := dec.DecodeFrames(second.Data)
	if err != nil {
		t.Fatalf("Decode multi-slice P-skip: %v", err)
	}
	assertDecodedEncoderFrameBytes(t, decodedSecond, appendI420FrameBytes(nil, secondFrame))

	thirdFrame := patternedI420EncoderFrame(48, 16)
	thirdFrame.Y[0] ^= 0x42
	thirdFrame.PTS = secondFrame.PTS + int64(cfg.RTPTimestampIncrement)
	third, err := enc.Encode(thirdFrame)
	if err != nil {
		t.Fatalf("Encode multi-slice changed P IntraPCM: %v", err)
	}
	assertEncoderNALTypes(t, third.NALUnits, []uint8{6, 1, 1, 1})
	assertEncoderVCLFirstMBs(t, append(append([]byte(nil), headers.AnnexB...), third.Data...), []uint8{1, 1, 1}, []uint32{0, 1, 2})
	decodedThird, err := dec.DecodeFrames(third.Data)
	if err != nil {
		t.Fatalf("Decode multi-slice changed P IntraPCM: %v", err)
	}
	assertDecodedEncoderFrameBytes(t, decodedThird, appendI420FrameBytes(nil, thirdFrame))
	if !decodedThird[0].KeyFrame ||
		decodedThird[0].SideData.RecoveryPoint == nil ||
		decodedThird[0].SideData.RecoveryPoint.RecoveryFrameCount != 0 {
		t.Fatalf("multi-slice changed P recovery side data key=%v recovery=%+v",
			decodedThird[0].KeyFrame, decodedThird[0].SideData.RecoveryPoint)
	}

	stream := append(append([]byte(nil), first.Data...), second.Data...)
	stream = append(stream, third.Data...)
	want := appendI420FrameBytes(nil, firstFrame)
	want = appendI420FrameBytes(want, secondFrame)
	want = appendI420FrameBytes(want, thirdFrame)
	assertFFmpegRawVideoOracle(t, stream, want)
}

func TestEncoderSliceCountFeedsRTPMode1SingleNALPackets(t *testing.T) {
	cfg := goh264.DefaultEncoderConfig(32, 16)
	cfg.DeblockMode = goh264.EncoderDeblockDisabled
	cfg.RTPMaxPayloadSize = 512
	cfg.SliceCount = 2
	enc, err := goh264.NewEncoder(cfg)
	if err != nil {
		t.Fatalf("NewEncoder: %v", err)
	}

	var callbackMetadata []goh264.EncoderRTPPacketMetadata
	enc.SetRTPPacketCallback(func(_ goh264.EncoderRTPPacket, meta goh264.EncoderRTPPacketMetadata) {
		callbackMetadata = append(callbackMetadata, meta)
	})

	frame := patternedI420EncoderFrame(32, 16)
	out, err := enc.Encode(frame)
	if err != nil {
		t.Fatalf("Encode multi-slice RTP IDR: %v", err)
	}
	assertEncoderNALTypes(t, out.NALUnits, []uint8{7, 8, 5, 5})
	assertEncoderVCLFirstMBs(t, out.Data, []uint8{5, 5}, []uint32{0, 1})
	if len(out.RTPPackets) != len(out.NALUnits) {
		t.Fatalf("RTP packets = %d, want one packet per NAL %d", len(out.RTPPackets), len(out.NALUnits))
	}
	if len(callbackMetadata) != len(out.RTPPackets) {
		t.Fatalf("callback metadata = %d, want packet count %d", len(callbackMetadata), len(out.RTPPackets))
	}

	var vclPackets int
	for i, pkt := range out.RTPPackets {
		if len(pkt.Payload) > cfg.RTPMaxPayloadSize {
			t.Fatalf("packet[%d] payload size = %d, max %d", i, len(pkt.Payload), cfg.RTPMaxPayloadSize)
		}
		if pkt.Marker != (i == len(out.RTPPackets)-1) {
			t.Fatalf("packet[%d] marker = %v, want only final marker", i, pkt.Marker)
		}
		meta := callbackMetadata[i]
		if meta.PacketIndex != i || meta.PacketCount != len(out.RTPPackets) {
			t.Fatalf("callback meta[%d] index/count = %d/%d, want %d/%d",
				i, meta.PacketIndex, meta.PacketCount, i, len(out.RTPPackets))
		}
		if meta.NALUnitType == 5 {
			vclPackets++
			if meta.PayloadFormat != goh264.EncoderRTPPayloadSingleNAL ||
				meta.NALUnitCount != 1 ||
				!meta.StartOfNAL || !meta.EndOfNAL ||
				!meta.IDR || !meta.KeyFrame {
				t.Fatalf("VCL callback meta[%d] = %+v, want complete IDR single-NAL packet", i, meta)
			}
		}
	}
	if vclPackets != 2 {
		t.Fatalf("IDR VCL RTP packets = %d, want 2", vclPackets)
	}

	annexB := annexBFromEncoderRTPPackets(t, out.RTPPackets)
	decoded, err := goh264.NewDecoder().DecodeAnnexBFrames(annexB)
	if err != nil {
		t.Fatalf("DecodeAnnexBFrames reassembled multi-slice RTP: %v", err)
	}
	assertDecodedEncoderFrameBytes(t, decoded, appendI420FrameBytes(nil, frame))
}

func TestEncoderEncodeRecoveryPointSEICanBeDisabled(t *testing.T) {
	cfg := goh264.DefaultEncoderConfig(16, 16)
	cfg.OutputFormat = goh264.EncoderOutputAnnexB
	cfg.DeblockMode = goh264.EncoderDeblockDisabled
	cfg.RecoveryPointSEI = false
	cfg.RTPMaxPayloadSize = 0
	enc, err := goh264.NewEncoder(cfg)
	if err != nil {
		t.Fatalf("NewEncoder: %v", err)
	}

	firstFrame := patternedI420EncoderFrame(16, 16)
	first, err := enc.Encode(firstFrame)
	if err != nil {
		t.Fatalf("Encode first IDR: %v", err)
	}
	secondFrame := patternedI420EncoderFrame(16, 16)
	secondFrame.Y[0] ^= 0x55
	secondFrame.PTS += int64(cfg.RTPTimestampIncrement)
	second, err := enc.Encode(secondFrame)
	if err != nil {
		t.Fatalf("Encode changed P IntraPCM: %v", err)
	}
	assertEncoderNALTypes(t, second.NALUnits, []uint8{1})

	dec := goh264.NewDecoder()
	if _, err := dec.DecodeFrames(first.Data); err != nil {
		t.Fatalf("Decode first IDR: %v", err)
	}
	decodedSecond, err := dec.DecodeFrames(second.Data)
	if err != nil {
		t.Fatalf("Decode changed P IntraPCM: %v", err)
	}
	assertDecodedEncoderFrameBytes(t, decodedSecond, appendI420FrameBytes(nil, secondFrame))
	if decodedSecond[0].SideData.RecoveryPoint != nil {
		t.Fatalf("disabled recovery-point SEI still surfaced side data: %+v", decodedSecond[0].SideData.RecoveryPoint)
	}
}

func TestEncoderEncodeForceIDRBypassesPSkipReference(t *testing.T) {
	for _, tt := range []struct {
		name    string
		request func(*goh264.Encoder, *goh264.EncoderFrame)
	}{
		{name: "encoder control", request: func(enc *goh264.Encoder, frame *goh264.EncoderFrame) {
			enc.ForceIDR()
			if !enc.PendingIDR() {
				t.Fatal("ForceIDR did not queue an IDR")
			}
		}},
		{name: "frame flag", request: func(_ *goh264.Encoder, frame *goh264.EncoderFrame) {
			frame.ForceIDR = true
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cfg := goh264.DefaultEncoderConfig(16, 16)
			cfg.OutputFormat = goh264.EncoderOutputAnnexB
			cfg.DeblockMode = goh264.EncoderDeblockDisabled
			cfg.RTPMaxPayloadSize = 0
			enc, err := goh264.NewEncoder(cfg)
			if err != nil {
				t.Fatalf("NewEncoder: %v", err)
			}
			frame := patternedI420EncoderFrame(16, 16)
			if _, err := enc.Encode(frame); err != nil {
				t.Fatalf("Encode first IDR: %v", err)
			}
			frame.PTS += int64(cfg.RTPTimestampIncrement)
			tt.request(enc, &frame)
			out, err := enc.Encode(frame)
			if err != nil {
				t.Fatalf("Encode forced IDR: %v", err)
			}
			if !out.KeyFrame || !out.IDR || enc.PendingIDR() {
				t.Fatalf("forced frame key=%v idr=%v pending=%v, want completed IDR", out.KeyFrame, out.IDR, enc.PendingIDR())
			}
			assertEncoderNALTypes(t, out.NALUnits, []uint8{7, 8, 5})
		})
	}
}

func TestEncoderEncodeRTPMode1FragmentsIDRAccessUnit(t *testing.T) {
	cfg := goh264.DefaultEncoderConfig(16, 16)
	cfg.RTPMaxPayloadSize = 32
	cfg.DeblockMode = goh264.EncoderDeblockDisabled
	enc, err := goh264.NewEncoder(cfg)
	if err != nil {
		t.Fatalf("NewEncoder: %v", err)
	}
	frame := patternedI420EncoderFrame(16, 16)
	frame.PTS = 12345

	out, err := enc.Encode(frame)
	if err != nil {
		t.Fatalf("Encode RTP IDR: %v", err)
	}
	if len(out.RTPPackets) < 3 {
		t.Fatalf("RTP packets = %d, want fragmented access unit", len(out.RTPPackets))
	}
	if out.RTPTime != uint32(frame.PTS) {
		t.Fatalf("RTP time = %d, want frame PTS %d", out.RTPTime, frame.PTS)
	}
	for i, pkt := range out.RTPPackets {
		if len(pkt.Payload) > cfg.RTPMaxPayloadSize {
			t.Fatalf("packet[%d] payload size = %d, max %d", i, len(pkt.Payload), cfg.RTPMaxPayloadSize)
		}
		if pkt.Timestamp != out.RTPTime {
			t.Fatalf("packet[%d] timestamp = %d, want %d", i, pkt.Timestamp, out.RTPTime)
		}
		if pkt.Marker != (i == len(out.RTPPackets)-1) {
			t.Fatalf("packet[%d] marker = %v, want only final marker", i, pkt.Marker)
		}
	}

	annexB := annexBFromEncoderRTPPackets(t, out.RTPPackets)
	decoded, err := goh264.NewDecoder().DecodeAnnexBFrames(annexB)
	if err != nil {
		t.Fatalf("DecodeAnnexBFrames reassembled RTP: %v", err)
	}
	assertDecodedEncoderFrameBytes(t, decoded, appendI420FrameBytes(nil, frame))
}

func TestEncoderEncodeRTPMode1STAPAAggregatesParameterSets(t *testing.T) {
	cfg := goh264.DefaultEncoderConfig(16, 16)
	cfg.STAPA = true
	cfg.RTPMaxPayloadSize = 128
	cfg.DeblockMode = goh264.EncoderDeblockDisabled
	enc, err := goh264.NewEncoder(cfg)
	if err != nil {
		t.Fatalf("NewEncoder: %v", err)
	}
	frame := patternedI420EncoderFrame(16, 16)
	frame.PTS = 67890

	out, err := enc.Encode(frame)
	if err != nil {
		t.Fatalf("Encode RTP IDR with STAP-A: %v", err)
	}
	if len(out.RTPPackets) < 2 {
		t.Fatalf("RTP packets = %d, want STAP-A plus VCL packets", len(out.RTPPackets))
	}
	stap := out.RTPPackets[0]
	if len(stap.Payload) == 0 || stap.Payload[0]&0x1f != 24 {
		t.Fatalf("first RTP payload = %x, want STAP-A type 24", stap.Payload)
	}
	if len(stap.Payload) > cfg.RTPMaxPayloadSize {
		t.Fatalf("STAP-A payload size = %d, max %d", len(stap.Payload), cfg.RTPMaxPayloadSize)
	}
	if stap.Marker {
		t.Fatal("STAP-A parameter-set packet unexpectedly has marker bit")
	}
	assertSTAPANALTypes(t, stap.Payload, []uint8{7, 8})
	for i, pkt := range out.RTPPackets {
		if pkt.Timestamp != out.RTPTime {
			t.Fatalf("packet[%d] timestamp = %d, want %d", i, pkt.Timestamp, out.RTPTime)
		}
		if len(pkt.Payload) > cfg.RTPMaxPayloadSize {
			t.Fatalf("packet[%d] payload size = %d, max %d", i, len(pkt.Payload), cfg.RTPMaxPayloadSize)
		}
		if pkt.Marker != (i == len(out.RTPPackets)-1) {
			t.Fatalf("packet[%d] marker = %v, want only final marker", i, pkt.Marker)
		}
	}

	annexB := annexBFromEncoderRTPPackets(t, out.RTPPackets)
	decoded, err := goh264.NewDecoder().DecodeAnnexBFrames(annexB)
	if err != nil {
		t.Fatalf("DecodeAnnexBFrames reassembled STAP-A RTP: %v", err)
	}
	assertDecodedEncoderFrameBytes(t, decoded, appendI420FrameBytes(nil, frame))
}

func TestEncoderEncodeRTPMode0EmitsSingleNALPackets(t *testing.T) {
	cfg := goh264.DefaultEncoderConfig(16, 16)
	cfg.RTPPacketizationMode = goh264.EncoderRTPPacketizationSingleNAL
	cfg.RTPMaxPayloadSize = 1200
	cfg.DeblockMode = goh264.EncoderDeblockDisabled
	enc, err := goh264.NewEncoder(cfg)
	if err != nil {
		t.Fatalf("NewEncoder mode 0: %v", err)
	}
	frame := patternedI420EncoderFrame(16, 16)
	frame.PTS = 24680

	out, err := enc.Encode(frame)
	if err != nil {
		t.Fatalf("Encode RTP mode 0 IDR: %v", err)
	}
	assertEncoderNALTypes(t, out.NALUnits, []uint8{7, 8, 5})
	if len(out.RTPPackets) != len(out.NALUnits) {
		t.Fatalf("mode 0 RTP packets = %d, want one packet per NAL %d", len(out.RTPPackets), len(out.NALUnits))
	}
	for i, pkt := range out.RTPPackets {
		unit := out.NALUnits[i]
		wantPayload := out.Data[unit.Offset : unit.Offset+unit.Size]
		if !bytes.Equal(pkt.Payload, wantPayload) {
			t.Fatalf("packet[%d] payload does not match raw NAL", i)
		}
		if typ := pkt.Payload[0] & 0x1f; typ == 24 || typ == 28 {
			t.Fatalf("packet[%d] payload type = %d, want single raw NAL", i, typ)
		}
		if pkt.Marker != (i == len(out.RTPPackets)-1) {
			t.Fatalf("packet[%d] marker = %v, want only final marker", i, pkt.Marker)
		}
		if pkt.Timestamp != uint32(frame.PTS) {
			t.Fatalf("packet[%d] timestamp = %d, want %d", i, pkt.Timestamp, frame.PTS)
		}
	}

	annexB := annexBFromEncoderRTPPackets(t, out.RTPPackets)
	decoded, err := goh264.NewDecoder().DecodeAnnexBFrames(annexB)
	if err != nil {
		t.Fatalf("DecodeAnnexBFrames reassembled mode 0 RTP: %v", err)
	}
	assertDecodedEncoderFrameBytes(t, decoded, appendI420FrameBytes(nil, frame))
}

func TestEncoderRTPMode0RejectsOversizeNAL(t *testing.T) {
	cfg := goh264.DefaultEncoderConfig(16, 16)
	cfg.RTPPacketizationMode = goh264.EncoderRTPPacketizationSingleNAL
	cfg.RTPMaxPayloadSize = 64
	cfg.DeblockMode = goh264.EncoderDeblockDisabled
	enc, err := goh264.NewEncoder(cfg)
	if err != nil {
		t.Fatalf("NewEncoder mode 0: %v", err)
	}

	if _, err := enc.Encode(patternedI420EncoderFrame(16, 16)); !errors.Is(err, goh264.ErrInvalidData) {
		t.Fatalf("Encode oversize mode 0 error = %v, want ErrInvalidData", err)
	}
}

func TestEncoderEncodeRTPPacketsCarryWebRTCMetadata(t *testing.T) {
	cfg := goh264.DefaultEncoderConfig(16, 16)
	cfg.RTPPayloadType = 102
	cfg.RTPSSRC = 0xdecafbad
	cfg.RTPMaxPayloadSize = 32
	cfg.DeblockMode = goh264.EncoderDeblockDisabled
	enc, err := goh264.NewEncoder(cfg)
	if err != nil {
		t.Fatalf("NewEncoder: %v", err)
	}

	first, err := enc.Encode(patternedI420EncoderFrame(16, 16))
	if err != nil {
		t.Fatalf("Encode first RTP frame: %v", err)
	}
	assertRTPPacketMetadata(t, first.RTPPackets, cfg.RTPPayloadType, cfg.RTPSSRC, 0)

	secondFrame := patternedI420EncoderFrame(16, 16)
	secondFrame.PTS += int64(cfg.RTPTimestampIncrement)
	second, err := enc.Encode(secondFrame)
	if err != nil {
		t.Fatalf("Encode second RTP frame: %v", err)
	}
	assertRTPPacketMetadata(t, second.RTPPackets, cfg.RTPPayloadType, cfg.RTPSSRC, uint16(len(first.RTPPackets)))
}

func TestEncoderEncodeRTPPacketsCarryFullRTPHeaders(t *testing.T) {
	cfg := goh264.DefaultEncoderConfig(16, 16)
	cfg.RTPPayloadType = 102
	cfg.RTPSSRC = 0xdecafbad
	cfg.RTPMaxPayloadSize = 32
	cfg.DeblockMode = goh264.EncoderDeblockDisabled
	enc, err := goh264.NewEncoder(cfg)
	if err != nil {
		t.Fatalf("NewEncoder: %v", err)
	}

	frame := patternedI420EncoderFrame(16, 16)
	frame.PTS = 0x01020304
	out, err := enc.Encode(frame)
	if err != nil {
		t.Fatalf("Encode RTP frame: %v", err)
	}
	if len(out.RTPPackets) == 0 {
		t.Fatal("RTP packet list is empty")
	}
	for i, pkt := range out.RTPPackets {
		if len(pkt.Data) != 12+len(pkt.Payload) {
			t.Fatalf("packet[%d] full RTP packet length = %d, want header plus payload %d",
				i, len(pkt.Data), 12+len(pkt.Payload))
		}
		if !bytes.Equal(pkt.Data[12:], pkt.Payload) {
			t.Fatalf("packet[%d] RTP payload bytes do not match Data payload", i)
		}
		if pkt.Data[0] != 0x80 {
			t.Fatalf("packet[%d] RTP version/P/X/CC byte = %#x, want 0x80", i, pkt.Data[0])
		}
		if got := pkt.Data[1] & 0x7f; got != cfg.RTPPayloadType {
			t.Fatalf("packet[%d] RTP payload type = %d, want %d", i, got, cfg.RTPPayloadType)
		}
		if got := pkt.Data[1]&0x80 != 0; got != pkt.Marker {
			t.Fatalf("packet[%d] RTP marker header = %v, want packet marker %v", i, got, pkt.Marker)
		}
		if got := binary.BigEndian.Uint16(pkt.Data[2:4]); got != pkt.SequenceNumber {
			t.Fatalf("packet[%d] RTP sequence = %d, want %d", i, got, pkt.SequenceNumber)
		}
		if got := binary.BigEndian.Uint32(pkt.Data[4:8]); got != pkt.Timestamp {
			t.Fatalf("packet[%d] RTP timestamp = %d, want %d", i, got, pkt.Timestamp)
		}
		if got := binary.BigEndian.Uint32(pkt.Data[8:12]); got != pkt.SSRC {
			t.Fatalf("packet[%d] RTP SSRC = %#x, want %#x", i, got, pkt.SSRC)
		}
	}
}

func TestEncoderRTPPacketCallbackReceivesWebRTCMetadata(t *testing.T) {
	cfg := goh264.DefaultEncoderConfig(16, 16)
	cfg.RTPPayloadType = 104
	cfg.RTPSSRC = 0x01020304
	cfg.RTPMaxPayloadSize = 128
	cfg.STAPA = true
	cfg.DeblockMode = goh264.EncoderDeblockDisabled
	enc, err := goh264.NewEncoder(cfg)
	if err != nil {
		t.Fatalf("NewEncoder: %v", err)
	}

	var callbackPackets []goh264.EncoderRTPPacket
	var callbackMetadata []goh264.EncoderRTPPacketMetadata
	enc.SetRTPPacketCallback(func(pkt goh264.EncoderRTPPacket, meta goh264.EncoderRTPPacketMetadata) {
		callbackPackets = append(callbackPackets, pkt)
		callbackMetadata = append(callbackMetadata, meta)
		if len(pkt.Payload) != 0 {
			pkt.Payload[0] ^= 0xff
		}
		if len(pkt.Data) != 0 {
			pkt.Data[0] ^= 0xff
		}
	})

	frame := patternedI420EncoderFrame(16, 16)
	frame.PTS = 0x010203
	out, err := enc.Encode(frame)
	if err != nil {
		t.Fatalf("Encode RTP with callback: %v", err)
	}
	if len(callbackPackets) != len(out.RTPPackets) || len(callbackMetadata) != len(out.RTPPackets) {
		t.Fatalf("callback packets/meta = %d/%d, want RTP packet count %d",
			len(callbackPackets), len(callbackMetadata), len(out.RTPPackets))
	}
	if len(out.RTPPackets) < 3 {
		t.Fatalf("RTP packet count = %d, want STAP-A plus FU-A fragments", len(out.RTPPackets))
	}
	if callbackPackets[0].Payload[0] == out.RTPPackets[0].Payload[0] ||
		callbackPackets[0].Data[0] == out.RTPPackets[0].Data[0] {
		t.Fatal("callback packet aliases returned RTP packet storage")
	}

	var sawSTAPA, sawFUAStart, sawFUAEnd bool
	for i, meta := range callbackMetadata {
		pkt := callbackPackets[i]
		if meta.PacketIndex != i || meta.PacketCount != len(out.RTPPackets) {
			t.Fatalf("callback meta[%d] index/count = %d/%d, want %d/%d",
				i, meta.PacketIndex, meta.PacketCount, i, len(out.RTPPackets))
		}
		if meta.FramePTS != frame.PTS || meta.FrameDTS != frame.PTS ||
			meta.RTPTime != uint32(frame.PTS) || !meta.KeyFrame || !meta.IDR {
			t.Fatalf("callback meta[%d] frame fields = %+v, want IDR frame PTS/RTP metadata", i, meta)
		}
		if pkt.SequenceNumber != out.RTPPackets[i].SequenceNumber ||
			pkt.Timestamp != out.RTPPackets[i].Timestamp ||
			pkt.PayloadType != cfg.RTPPayloadType ||
			pkt.SSRC != cfg.RTPSSRC ||
			pkt.Marker != (i == len(out.RTPPackets)-1) {
			t.Fatalf("callback packet[%d] metadata = %+v, want returned RTP packet fields", i, pkt)
		}
		switch meta.PayloadFormat {
		case goh264.EncoderRTPPayloadSTAPA:
			if meta.NALUnitType != 24 || meta.NALUnitCount != 2 || !meta.ParameterSet ||
				meta.StartOfNAL || meta.EndOfNAL {
				t.Fatalf("STAP-A callback metadata = %+v, want SPS/PPS aggregate", meta)
			}
			sawSTAPA = true
		case goh264.EncoderRTPPayloadFUA:
			if meta.NALUnitType != 5 || meta.NALUnitCount != 1 || meta.ParameterSet {
				t.Fatalf("FU-A callback metadata = %+v, want fragmented IDR NAL", meta)
			}
			sawFUAStart = sawFUAStart || meta.StartOfNAL
			sawFUAEnd = sawFUAEnd || meta.EndOfNAL
		default:
			t.Fatalf("callback payload format = %v, want STAP-A or FU-A for this access unit", meta.PayloadFormat)
		}
	}
	if !sawSTAPA || !sawFUAStart || !sawFUAEnd {
		t.Fatalf("callback saw STAP-A/start/end = %v/%v/%v, want all true",
			sawSTAPA, sawFUAStart, sawFUAEnd)
	}
}

func TestEncoderRTPPacketCallbackCanBeClearedAndSkipsNonRTPOutput(t *testing.T) {
	cfg := goh264.DefaultEncoderConfig(16, 16)
	cfg.OutputFormat = goh264.EncoderOutputAnnexB
	cfg.RTPMaxPayloadSize = 0
	cfg.DeblockMode = goh264.EncoderDeblockDisabled
	enc, err := goh264.NewEncoder(cfg)
	if err != nil {
		t.Fatalf("NewEncoder Annex B: %v", err)
	}
	var calls int
	enc.SetRTPPacketCallback(func(goh264.EncoderRTPPacket, goh264.EncoderRTPPacketMetadata) {
		calls++
	})
	if _, err := enc.Encode(patternedI420EncoderFrame(16, 16)); err != nil {
		t.Fatalf("Encode Annex B with callback: %v", err)
	}
	if calls != 0 {
		t.Fatalf("Annex B callback calls = %d, want 0", calls)
	}

	cfg = goh264.DefaultEncoderConfig(16, 16)
	cfg.DeblockMode = goh264.EncoderDeblockDisabled
	enc, err = goh264.NewEncoder(cfg)
	if err != nil {
		t.Fatalf("NewEncoder RTP: %v", err)
	}
	enc.SetRTPPacketCallback(func(goh264.EncoderRTPPacket, goh264.EncoderRTPPacketMetadata) {
		calls++
	})
	enc.SetRTPPacketCallback(nil)
	if _, err := enc.Encode(patternedI420EncoderFrame(16, 16)); err != nil {
		t.Fatalf("Encode RTP after clearing callback: %v", err)
	}
	if calls != 0 {
		t.Fatalf("cleared callback calls = %d, want 0", calls)
	}
}

func TestEncoderRTPAutoTimestampAdvancesWithoutExplicitPTS(t *testing.T) {
	cfg := goh264.DefaultEncoderConfig(16, 16)
	cfg.DeblockMode = goh264.EncoderDeblockDisabled
	enc, err := goh264.NewEncoder(cfg)
	if err != nil {
		t.Fatalf("NewEncoder: %v", err)
	}

	firstFrame := patternedI420EncoderFrame(16, 16)
	firstFrame.PTS = 0
	first, err := enc.Encode(firstFrame)
	if err != nil {
		t.Fatalf("Encode first zero-PTS RTP frame: %v", err)
	}
	if first.RTPTime != 0 {
		t.Fatalf("first RTP time = %d, want 0", first.RTPTime)
	}
	assertRTPPacketTimestamps(t, first.RTPPackets, first.RTPTime)

	secondFrame := patternedI420EncoderFrame(16, 16)
	secondFrame.PTS = 0
	second, err := enc.Encode(secondFrame)
	if err != nil {
		t.Fatalf("Encode second zero-PTS RTP frame: %v", err)
	}
	if second.RTPTime != cfg.RTPTimestampIncrement {
		t.Fatalf("second RTP time = %d, want default increment %d", second.RTPTime, cfg.RTPTimestampIncrement)
	}
	assertRTPPacketTimestamps(t, second.RTPPackets, second.RTPTime)

	thirdFrame := patternedI420EncoderFrame(16, 16)
	thirdFrame.PTS = 90_000
	thirdFrame.Duration = 1_500
	third, err := enc.Encode(thirdFrame)
	if err != nil {
		t.Fatalf("Encode explicit-PTS RTP frame: %v", err)
	}
	if third.RTPTime != uint32(thirdFrame.PTS) {
		t.Fatalf("third RTP time = %d, want explicit PTS %d", third.RTPTime, thirdFrame.PTS)
	}
	assertRTPPacketTimestamps(t, third.RTPPackets, third.RTPTime)

	fourthFrame := patternedI420EncoderFrame(16, 16)
	fourthFrame.PTS = 0
	fourthFrame.Duration = 1_500
	fourth, err := enc.Encode(fourthFrame)
	if err != nil {
		t.Fatalf("Encode duration-advanced RTP frame: %v", err)
	}
	if fourth.RTPTime != uint32(thirdFrame.PTS+thirdFrame.Duration) {
		t.Fatalf("fourth RTP time = %d, want explicit PTS plus duration %d",
			fourth.RTPTime, thirdFrame.PTS+thirdFrame.Duration)
	}
	assertRTPPacketTimestamps(t, fourth.RTPPackets, fourth.RTPTime)
}

func TestEncoderEncodeIntoValidatesInvalidFrameBeforeBitstream(t *testing.T) {
	enc, err := goh264.NewEncoder(goh264.DefaultEncoderConfig(16, 16))
	if err != nil {
		t.Fatalf("NewEncoder: %v", err)
	}

	frame := validI420EncoderFrame(16, 16)
	if out, err := enc.EncodeInto(make([]byte, 0, 1024), frame); err != nil || !out.IDR {
		t.Fatalf("EncodeInto valid-frame out.IDR=%v error=%v, want successful IDR", out.IDR, err)
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
		"PendingIDR", "RecoveryPointSEI", "SetBitrate", "SetFrameRate", "SetRTPMaxPayloadSize",
		"SetRTPPacketCallback", "Reconfigure",
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

func patternedI420EncoderFrame(width, height int) goh264.EncoderFrame {
	frame := validI420EncoderFrame(width, height)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			frame.Y[y*frame.StrideY+x] = byte((x*11 + y*17 + 3) & 0xff)
		}
	}
	chromaWidth := width / 2
	chromaHeight := height / 2
	for y := 0; y < chromaHeight; y++ {
		for x := 0; x < chromaWidth; x++ {
			frame.Cb[y*frame.StrideCb+x] = byte((x*19 + y*7 + 41) & 0xff)
			frame.Cr[y*frame.StrideCr+x] = byte((x*5 + y*23 + 109) & 0xff)
		}
	}
	frame.PTS = 3000
	return frame
}

func appendI420FrameBytes(dst []byte, frame goh264.EncoderFrame) []byte {
	for y := 0; y < frame.Height; y++ {
		row := frame.Y[y*frame.StrideY : y*frame.StrideY+frame.Width]
		dst = append(dst, row...)
	}
	chromaWidth := frame.Width / 2
	chromaHeight := frame.Height / 2
	for y := 0; y < chromaHeight; y++ {
		row := frame.Cb[y*frame.StrideCb : y*frame.StrideCb+chromaWidth]
		dst = append(dst, row...)
	}
	for y := 0; y < chromaHeight; y++ {
		row := frame.Cr[y*frame.StrideCr : y*frame.StrideCr+chromaWidth]
		dst = append(dst, row...)
	}
	return dst
}

func appendCroppedI420FrameBytes(dst []byte, frame goh264.EncoderFrame, crop goh264.EncoderCrop) []byte {
	width := frame.Width - crop.Left - crop.Right
	height := frame.Height - crop.Top - crop.Bottom
	for y := 0; y < height; y++ {
		row := frame.Y[(crop.Top+y)*frame.StrideY+crop.Left : (crop.Top+y)*frame.StrideY+crop.Left+width]
		dst = append(dst, row...)
	}
	chromaWidth := width / 2
	chromaHeight := height / 2
	chromaLeft := crop.Left / 2
	chromaTop := crop.Top / 2
	for y := 0; y < chromaHeight; y++ {
		row := frame.Cb[(chromaTop+y)*frame.StrideCb+chromaLeft : (chromaTop+y)*frame.StrideCb+chromaLeft+chromaWidth]
		dst = append(dst, row...)
	}
	for y := 0; y < chromaHeight; y++ {
		row := frame.Cr[(chromaTop+y)*frame.StrideCr+chromaLeft : (chromaTop+y)*frame.StrideCr+chromaLeft+chromaWidth]
		dst = append(dst, row...)
	}
	return dst
}

func assertEncoderNALTypes(t *testing.T, nals []goh264.EncoderNALUnit, want []uint8) {
	t.Helper()
	if len(nals) != len(want) {
		t.Fatalf("NAL count = %d, want %d (%+v)", len(nals), len(want), nals)
	}
	for i, typ := range want {
		if nals[i].Type != typ {
			t.Fatalf("NAL[%d] type = %d, want %d (%+v)", i, nals[i].Type, typ, nals)
		}
	}
}

func assertEncoderVCLFirstMBs(t *testing.T, annexB []byte, wantTypes []uint8, wantFirstMBs []uint32) {
	t.Helper()
	nals, err := h264.SplitAnnexB(annexB)
	if err != nil {
		t.Fatalf("SplitAnnexB: %v", err)
	}
	var spsList [32]*h264.SPS
	var ppsList [256]*h264.PPS
	var gotTypes []uint8
	var gotFirstMBs []uint32
	for _, nal := range nals {
		switch nal.Type {
		case h264.NALSPS:
			sps, err := h264.DecodeSPS(nal.RBSP)
			if err != nil {
				t.Fatalf("DecodeSPS: %v", err)
			}
			spsList[sps.SPSID] = sps
		case h264.NALPPS:
			pps, err := h264.DecodePPS(nal.RBSP, &spsList)
			if err != nil {
				t.Fatalf("DecodePPS: %v", err)
			}
			ppsList[pps.PPSID] = pps
		case h264.NALIDRSlice, h264.NALSlice:
			sh, err := h264.ParseSliceHeader(nal, &ppsList)
			if err != nil {
				t.Fatalf("ParseSliceHeader nal=%d: %v", nal.Type, err)
			}
			gotTypes = append(gotTypes, uint8(nal.Type))
			gotFirstMBs = append(gotFirstMBs, sh.FirstMBAddr)
		}
	}
	if !reflect.DeepEqual(gotTypes, wantTypes) || !reflect.DeepEqual(gotFirstMBs, wantFirstMBs) {
		t.Fatalf("VCL types/first MBs = %v/%v, want %v/%v",
			gotTypes, gotFirstMBs, wantTypes, wantFirstMBs)
	}
}

func assertDecodedEncoderFrameBytes(t *testing.T, frames []*goh264.Frame, want []byte) {
	t.Helper()
	if len(frames) != 1 {
		t.Fatalf("decoded frames = %d, want 1", len(frames))
	}
	raw, err := frames[0].AppendRawYUV(nil)
	if err != nil {
		t.Fatalf("AppendRawYUV: %v", err)
	}
	if !bytes.Equal(raw, want) {
		t.Fatalf("decoded raw md5 = %x, want %x", md5.Sum(raw), md5.Sum(want))
	}
}

func assertRTPPacketMetadata(t *testing.T, packets []goh264.EncoderRTPPacket, payloadType uint8, ssrc uint32, firstSeq uint16) {
	t.Helper()
	if len(packets) == 0 {
		t.Fatal("RTP packet list is empty")
	}
	for i, pkt := range packets {
		if pkt.PayloadType != payloadType {
			t.Fatalf("packet[%d] payload type = %d, want %d", i, pkt.PayloadType, payloadType)
		}
		if pkt.SSRC != ssrc {
			t.Fatalf("packet[%d] SSRC = %#x, want %#x", i, pkt.SSRC, ssrc)
		}
		if pkt.SequenceNumber != firstSeq+uint16(i) {
			t.Fatalf("packet[%d] sequence = %d, want %d", i, pkt.SequenceNumber, firstSeq+uint16(i))
		}
	}
}

func assertRTPPacketTimestamps(t *testing.T, packets []goh264.EncoderRTPPacket, want uint32) {
	t.Helper()
	if len(packets) == 0 {
		t.Fatal("RTP packet list is empty")
	}
	for i, pkt := range packets {
		if pkt.Timestamp != want {
			t.Fatalf("packet[%d] timestamp = %d, want %d", i, pkt.Timestamp, want)
		}
		if len(pkt.Data) >= 8 && binary.BigEndian.Uint32(pkt.Data[4:8]) != want {
			t.Fatalf("packet[%d] RTP header timestamp = %d, want %d",
				i, binary.BigEndian.Uint32(pkt.Data[4:8]), want)
		}
	}
}

func assertSTAPANALTypes(t *testing.T, payload []byte, want []uint8) {
	t.Helper()
	if len(payload) == 0 || payload[0]&0x1f != 24 {
		t.Fatalf("payload is not STAP-A: %x", payload)
	}
	var got []uint8
	for pos := 1; pos < len(payload); {
		if pos+2 > len(payload) {
			t.Fatalf("truncated STAP-A length at byte %d: %x", pos, payload)
		}
		size := int(payload[pos])<<8 | int(payload[pos+1])
		pos += 2
		if size == 0 || pos+size > len(payload) {
			t.Fatalf("invalid STAP-A NAL size %d at byte %d of %d", size, pos, len(payload))
		}
		got = append(got, payload[pos]&0x1f)
		pos += size
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("STAP-A NAL types = %v, want %v", got, want)
	}
}

func assertFFmpegRawVideoOracle(t *testing.T, annexB []byte, want []byte) {
	t.Helper()
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Skip("ffmpeg not available")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "encoded.h264")
	if err := os.WriteFile(path, annexB, 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(ffmpeg,
		"-hide_banner", "-loglevel", "error",
		"-i", path,
		"-f", "rawvideo", "-pix_fmt", "yuv420p", "-",
	)
	raw, err := cmd.Output()
	if err != nil {
		t.Fatalf("ffmpeg rawvideo decode: %v", err)
	}
	if !bytes.Equal(raw, want) {
		t.Fatalf("ffmpeg raw md5 = %x, want %x", md5.Sum(raw), md5.Sum(want))
	}
}

func annexBFromEncoderRTPPackets(t *testing.T, packets []goh264.EncoderRTPPacket) []byte {
	t.Helper()
	var out []byte
	var fu []byte
	var inFU bool
	for i, pkt := range packets {
		payload := pkt.Payload
		if len(payload) == 0 {
			t.Fatalf("packet[%d] empty payload", i)
		}
		typ := payload[0] & 0x1f
		if typ == 24 {
			if inFU {
				t.Fatalf("packet[%d] STAP-A while FU-A is open", i)
			}
			for pos := 1; pos < len(payload); {
				if pos+2 > len(payload) {
					t.Fatalf("packet[%d] truncated STAP-A length at byte %d: %x", i, pos, payload)
				}
				size := int(payload[pos])<<8 | int(payload[pos+1])
				pos += 2
				if size == 0 || pos+size > len(payload) {
					t.Fatalf("packet[%d] invalid STAP-A NAL size %d at byte %d of %d", i, size, pos, len(payload))
				}
				out = append(out, 0, 0, 0, 1)
				out = append(out, payload[pos:pos+size]...)
				pos += size
			}
			continue
		}
		if typ != 28 {
			if inFU {
				t.Fatalf("packet[%d] single NAL while FU-A is open", i)
			}
			out = append(out, 0, 0, 0, 1)
			out = append(out, payload...)
			continue
		}
		if len(payload) < 3 {
			t.Fatalf("packet[%d] FU-A payload too small: %x", i, payload)
		}
		start := payload[1]&0x80 != 0
		end := payload[1]&0x40 != 0
		if start {
			if inFU {
				t.Fatalf("packet[%d] starts FU-A while previous is open", i)
			}
			fu = append(fu[:0], (payload[0]&0xe0)|(payload[1]&0x1f))
			inFU = true
		} else if !inFU {
			t.Fatalf("packet[%d] FU-A continuation without start", i)
		}
		fu = append(fu, payload[2:]...)
		if end {
			out = append(out, 0, 0, 0, 1)
			out = append(out, fu...)
			inFU = false
		}
	}
	if inFU {
		t.Fatal("unterminated FU-A sequence")
	}
	return out
}
