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
		{name: "rtp payload type too large", mutate: func(c *goh264.EncoderConfig) { c.RTPPayloadType = 128 }, want: goh264.ErrInvalidData},
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
