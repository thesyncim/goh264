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

func TestEncoderMethodsHandleNilEncoder(t *testing.T) {
	var enc *goh264.Encoder
	if got := enc.Config(); got != (goh264.EncoderConfig{}) {
		t.Fatalf("Config nil encoder = %+v, want zero config", got)
	}
	if enc.PendingIDR() {
		t.Fatal("PendingIDR nil encoder = true, want false")
	}
	noPanic := []struct {
		name string
		call func()
	}{
		{name: "ForceIDR", call: func() { enc.ForceIDR() }},
		{name: "HandlePLI", call: func() { enc.HandlePLI() }},
		{name: "HandleFIR", call: func() { enc.HandleFIR() }},
		{name: "SetRTPPacketCallback", call: func() { enc.SetRTPPacketCallback(nil) }},
	}
	for _, tt := range noPanic {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("%s panicked on nil encoder: %v", tt.name, r)
				}
			}()
			tt.call()
		})
	}

	errorCalls := []struct {
		name string
		call func() error
	}{
		{name: "ParameterSets", call: func() error {
			_, err := enc.ParameterSets()
			return err
		}},
		{name: "RecoveryPointSEI", call: func() error {
			_, err := enc.RecoveryPointSEI(0)
			return err
		}},
		{name: "Encode", call: func() error {
			_, err := enc.Encode(goh264.EncoderFrame{})
			return err
		}},
		{name: "EncodeInto", call: func() error {
			_, err := enc.EncodeInto(nil, goh264.EncoderFrame{})
			return err
		}},
		{name: "SetBitrate", call: func() error {
			return enc.SetBitrate(1, 1)
		}},
		{name: "SetFrameRate", call: func() error {
			return enc.SetFrameRate(1, 1)
		}},
		{name: "SetRTPMaxPayloadSize", call: func() error {
			return enc.SetRTPMaxPayloadSize(1200)
		}},
		{name: "Reconfigure", call: func() error {
			return enc.Reconfigure(goh264.EncoderReconfigure{})
		}},
	}
	for _, tt := range errorCalls {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("%s panicked on nil encoder: %v", tt.name, r)
				}
			}()
			if err := tt.call(); !errors.Is(err, goh264.ErrInvalidData) {
				t.Fatalf("%s nil encoder error = %v, want ErrInvalidData", tt.name, err)
			}
		})
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
		{name: "coded macroblock count overflow", mutate: func(c *goh264.EncoderConfig) {
			c.Width = maxIntForTest - 1
			c.Height = 16
			c.StrideY = c.Width
			c.StrideCb = c.Width / 2
			c.StrideCr = c.Width / 2
		}, want: goh264.ErrInvalidData},
		{name: "derived RTP timestamp overflow", mutate: func(c *goh264.EncoderConfig) {
			c.TimeBaseDen = maxIntForTest
			c.FrameRateNum = 1
			c.FrameRateDen = 2
			c.RTPTimestampIncrement = 0
		}, want: goh264.ErrInvalidData},
		{name: "derived RTP timestamp underflow", mutate: func(c *goh264.EncoderConfig) {
			c.TimeBaseDen = 1
			c.FrameRateNum = 2
			c.FrameRateDen = 1
			c.RTPTimestampIncrement = 0
		}, want: goh264.ErrInvalidData},
		{name: "bitrate frame budget overflow", mutate: func(c *goh264.EncoderConfig) {
			c.TargetBitrate = maxIntForTest
			c.MaxBitrate = maxIntForTest
			c.FrameRateNum = 1
			c.FrameRateDen = 3
			c.RTPTimestampIncrement = 1
		}, want: goh264.ErrInvalidData},
		{name: "configured luma plane size overflow", mutate: func(c *goh264.EncoderConfig) {
			c.Width = maxIntForTest - 15
			c.Height = 32
			c.StrideY = c.Width
			c.StrideCb = c.Width / 2
			c.StrideCr = c.Width / 2
			c.RTPTimestampIncrement = 1
		}, want: goh264.ErrInvalidData},
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
		{name: "horizontal crop overflow", mutate: func(c *goh264.EncoderConfig) {
			c.Crop.Left = maxIntForTest - 1
			c.Crop.Right = maxIntForTest - 1
		}, want: goh264.ErrInvalidData},
		{name: "vertical crop overflow", mutate: func(c *goh264.EncoderConfig) {
			c.Crop.Top = maxIntForTest - 1
			c.Crop.Bottom = maxIntForTest - 1
		}, want: goh264.ErrInvalidData},
		{name: "crop consumes width", mutate: func(c *goh264.EncoderConfig) { c.Crop.Left = c.Width / 2; c.Crop.Right = c.Width / 2 }, want: goh264.ErrInvalidData},
		{name: "partial sar", mutate: func(c *goh264.EncoderConfig) { c.Color.SARNum = 1 }, want: goh264.ErrInvalidData},
		{name: "negative sar", mutate: func(c *goh264.EncoderConfig) { c.Color.SARDen = -1 }, want: goh264.ErrInvalidData},
		{name: "color primaries too large", mutate: func(c *goh264.EncoderConfig) { c.Color.ColorPrimaries = 256 }, want: goh264.ErrInvalidData},
		{name: "negative chroma sample location", mutate: func(c *goh264.EncoderConfig) { c.Color.ChromaSampleLocTypeTopField = -1 }, want: goh264.ErrInvalidData},
		{name: "chroma sample location too large", mutate: func(c *goh264.EncoderConfig) { c.Color.ChromaSampleLocTypeBottomField = 6 }, want: goh264.ErrInvalidData},
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

func TestEncoderReconfigureRejectsBitrateBudgetOverflowWithoutMutation(t *testing.T) {
	enc, err := goh264.NewEncoder(goh264.DefaultEncoderConfig(16, 16))
	if err != nil {
		t.Fatalf("NewEncoder: %v", err)
	}
	before := enc.Config()
	if err := enc.Reconfigure(goh264.EncoderReconfigure{
		TargetBitrate: maxIntForTest,
		MaxBitrate:    maxIntForTest,
		FrameRateNum:  1,
		FrameRateDen:  3,
	}); !errors.Is(err, goh264.ErrInvalidData) {
		t.Fatalf("Reconfigure bitrate-budget overflow error = %v, want ErrInvalidData", err)
	}
	if got := enc.Config(); got != before {
		t.Fatalf("overflow bitrate-budget Reconfigure mutated config = %+v, want %+v", got, before)
	}
}

func TestEncoderSetFrameRateRejectsTimestampOverflowWithoutMutation(t *testing.T) {
	enc, err := goh264.NewEncoder(goh264.DefaultEncoderConfig(16, 16))
	if err != nil {
		t.Fatalf("NewEncoder: %v", err)
	}
	before := enc.Config()
	if err := enc.SetFrameRate(1, maxIntForTest); !errors.Is(err, goh264.ErrInvalidData) {
		t.Fatalf("SetFrameRate overflow error = %v, want ErrInvalidData", err)
	}
	if got := enc.Config(); got != before {
		t.Fatalf("overflow SetFrameRate mutated config = %+v, want %+v", got, before)
	}
}

func TestEncoderSetFrameRateRejectsZeroTimestampIncrementWithoutMutation(t *testing.T) {
	enc, err := goh264.NewEncoder(goh264.DefaultEncoderConfig(16, 16))
	if err != nil {
		t.Fatalf("NewEncoder: %v", err)
	}
	before := enc.Config()
	if err := enc.SetFrameRate(before.TimeBaseDen+1, 1); !errors.Is(err, goh264.ErrInvalidData) {
		t.Fatalf("SetFrameRate zero-increment error = %v, want ErrInvalidData", err)
	}
	if got := enc.Config(); got != before {
		t.Fatalf("zero-increment SetFrameRate mutated config = %+v, want %+v", got, before)
	}
}

func TestEncoderEncodeRejectsFramePlaneSizeOverflowWithoutPanic(t *testing.T) {
	enc, err := goh264.NewEncoder(goh264.DefaultEncoderConfig(16, 16))
	if err != nil {
		t.Fatalf("NewEncoder: %v", err)
	}
	frame := validI420EncoderFrame(16, 16)
	frame.Y = nil
	frame.StrideY = maxIntForTest
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Encode panicked on overflowed frame plane geometry: %v", r)
		}
	}()
	if _, err := enc.Encode(frame); !errors.Is(err, goh264.ErrInvalidData) {
		t.Fatalf("Encode overflowed frame plane geometry error = %v, want ErrInvalidData", err)
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

func TestEncoderInvalidSetterPreservesPendingIDR(t *testing.T) {
	tests := []struct {
		name string
		call func(*goh264.Encoder) error
	}{
		{name: "SetBitrate", call: func(enc *goh264.Encoder) error {
			return enc.SetBitrate(0, 0)
		}},
		{name: "SetFrameRate", call: func(enc *goh264.Encoder) error {
			return enc.SetFrameRate(0, 1)
		}},
		{name: "SetRTPMaxPayloadSize", call: func(enc *goh264.Encoder) error {
			return enc.SetRTPMaxPayloadSize(2)
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enc, err := goh264.NewEncoder(goh264.DefaultEncoderConfig(16, 16))
			if err != nil {
				t.Fatalf("NewEncoder: %v", err)
			}
			first, err := enc.Encode(patternedI420EncoderFrame(16, 16))
			if err != nil {
				t.Fatalf("Encode first IDR: %v", err)
			}
			if !first.IDR || enc.PendingIDR() {
				t.Fatalf("first frame idr=%v pending=%v, want completed IDR", first.IDR, enc.PendingIDR())
			}

			enc.ForceIDR()
			if !enc.PendingIDR() {
				t.Fatalf("%s ForceIDR did not queue IDR", tt.name)
			}
			before := enc.Config()
			if err := tt.call(enc); !errors.Is(err, goh264.ErrInvalidData) {
				t.Fatalf("%s invalid error = %v, want ErrInvalidData", tt.name, err)
			}
			if got := enc.Config(); got != before {
				t.Fatalf("%s invalid call mutated config = %+v, want %+v", tt.name, got, before)
			}
			if !enc.PendingIDR() {
				t.Fatalf("%s invalid call cleared pending IDR", tt.name)
			}

			secondFrame := patternedI420EncoderFrame(16, 16)
			secondFrame.Y[0] ^= 0x11
			second, err := enc.Encode(secondFrame)
			if err != nil {
				t.Fatalf("%s Encode after invalid setter: %v", tt.name, err)
			}
			if !second.IDR || enc.PendingIDR() {
				t.Fatalf("%s post-invalid-setter frame idr=%v pending=%v, want delivered IDR",
					tt.name, second.IDR, enc.PendingIDR())
			}
			assertEncoderNALTypes(t, second.NALUnits, []uint8{7, 8, 5})
		})
	}
}

func TestEncoderValidSetterPreservesPendingIDR(t *testing.T) {
	tests := []struct {
		name string
		call func(*goh264.Encoder) error
	}{
		{name: "SetBitrate", call: func(enc *goh264.Encoder) error {
			return enc.SetBitrate(800_000, 900_000)
		}},
		{name: "SetFrameRate", call: func(enc *goh264.Encoder) error {
			return enc.SetFrameRate(60, 1)
		}},
		{name: "SetRTPMaxPayloadSize", call: func(enc *goh264.Encoder) error {
			return enc.SetRTPMaxPayloadSize(1000)
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enc, err := goh264.NewEncoder(goh264.DefaultEncoderConfig(16, 16))
			if err != nil {
				t.Fatalf("NewEncoder: %v", err)
			}
			first, err := enc.Encode(patternedI420EncoderFrame(16, 16))
			if err != nil {
				t.Fatalf("Encode first IDR: %v", err)
			}
			if !first.IDR || enc.PendingIDR() {
				t.Fatalf("first frame idr=%v pending=%v, want completed IDR", first.IDR, enc.PendingIDR())
			}

			enc.ForceIDR()
			if !enc.PendingIDR() {
				t.Fatalf("%s ForceIDR did not queue IDR", tt.name)
			}
			if err := tt.call(enc); err != nil {
				t.Fatalf("%s valid update: %v", tt.name, err)
			}
			if !enc.PendingIDR() {
				t.Fatalf("%s valid update cleared pending IDR", tt.name)
			}

			secondFrame := patternedI420EncoderFrame(16, 16)
			secondFrame.Y[0] ^= 0x31
			second, err := enc.Encode(secondFrame)
			if err != nil {
				t.Fatalf("%s Encode after valid setter: %v", tt.name, err)
			}
			if !second.IDR || enc.PendingIDR() {
				t.Fatalf("%s post-valid-setter frame idr=%v pending=%v, want delivered IDR",
					tt.name, second.IDR, enc.PendingIDR())
			}
			assertEncoderNALTypes(t, second.NALUnits, []uint8{7, 8, 5})
		})
	}
}

func TestEncoderValidReconfigurePreservesPendingIDR(t *testing.T) {
	spsPPSBeforeIDR := false
	recoveryPointSEI := false
	tests := []struct {
		name             string
		update           goh264.EncoderReconfigure
		wantNALs         []uint8
		wantRTPIncrement uint32
	}{
		{name: "bitrate", update: goh264.EncoderReconfigure{TargetBitrate: 800_000, MaxBitrate: 900_000}, wantNALs: []uint8{7, 8, 5}},
		{name: "frame rate", update: goh264.EncoderReconfigure{FrameRateNum: 60, FrameRateDen: 1}, wantNALs: []uint8{7, 8, 5}},
		{name: "payload size", update: goh264.EncoderReconfigure{RTPMaxPayloadSize: 1000}, wantNALs: []uint8{7, 8, 5}},
		{name: "timestamp increment", update: goh264.EncoderReconfigure{RTPTimestampIncrement: 1234}, wantNALs: []uint8{7, 8, 5}, wantRTPIncrement: 1234},
		{name: "deblock", update: goh264.EncoderReconfigure{DeblockMode: goh264.EncoderDeblockDisabled}, wantNALs: []uint8{7, 8, 5}},
		{name: "sps pps cadence", update: goh264.EncoderReconfigure{
			SPSPPSMode:       goh264.EncoderSPSPPSEveryIDR,
			SPSPPSBeforeIDR:  &spsPPSBeforeIDR,
			RecoveryPointSEI: &recoveryPointSEI,
		}, wantNALs: []uint8{7, 8, 5}},
		{name: "sps pps suppression", update: goh264.EncoderReconfigure{
			SPSPPSBeforeIDR:  &spsPPSBeforeIDR,
			RecoveryPointSEI: &recoveryPointSEI,
		}, wantNALs: []uint8{5}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enc, err := goh264.NewEncoder(goh264.DefaultEncoderConfig(16, 16))
			if err != nil {
				t.Fatalf("NewEncoder: %v", err)
			}
			first, err := enc.Encode(patternedI420EncoderFrame(16, 16))
			if err != nil {
				t.Fatalf("Encode first IDR: %v", err)
			}
			if !first.IDR || enc.PendingIDR() {
				t.Fatalf("first frame idr=%v pending=%v, want completed IDR", first.IDR, enc.PendingIDR())
			}

			enc.ForceIDR()
			if !enc.PendingIDR() {
				t.Fatalf("%s ForceIDR did not queue IDR", tt.name)
			}
			if err := enc.Reconfigure(tt.update); err != nil {
				t.Fatalf("%s valid reconfigure: %v", tt.name, err)
			}
			if !enc.PendingIDR() {
				t.Fatalf("%s valid reconfigure cleared pending IDR", tt.name)
			}
			if tt.wantRTPIncrement != 0 {
				if got := enc.Config().RTPTimestampIncrement; got != tt.wantRTPIncrement {
					t.Fatalf("%s RTP timestamp increment = %d, want %d", tt.name, got, tt.wantRTPIncrement)
				}
			}

			secondFrame := patternedI420EncoderFrame(16, 16)
			secondFrame.Y[0] ^= 0x41
			if tt.wantRTPIncrement != 0 {
				secondFrame.PTS = 0
				secondFrame.Duration = 0
			}
			second, err := enc.Encode(secondFrame)
			if err != nil {
				t.Fatalf("%s Encode after valid reconfigure: %v", tt.name, err)
			}
			if !second.IDR || enc.PendingIDR() {
				t.Fatalf("%s post-valid-reconfigure frame idr=%v pending=%v, want delivered IDR",
					tt.name, second.IDR, enc.PendingIDR())
			}
			assertEncoderNALTypes(t, second.NALUnits, tt.wantNALs)
			if tt.wantRTPIncrement != 0 {
				thirdFrame := secondFrame
				thirdFrame.PTS = 0
				thirdFrame.Duration = 0
				thirdFrame.Y[1] ^= 0x11
				third, err := enc.Encode(thirdFrame)
				if err != nil {
					t.Fatalf("%s Encode after forced IDR: %v", tt.name, err)
				}
				if third.RTPTime != second.RTPTime+tt.wantRTPIncrement {
					t.Fatalf("%s post-reconfigure RTP time = %d, want %d",
						tt.name, third.RTPTime, second.RTPTime+tt.wantRTPIncrement)
				}
			}
		})
	}
}

func TestEncoderValidOutputReconfigurePreservesPendingIDR(t *testing.T) {
	t.Run("avc out-of-band", func(t *testing.T) {
		cfg := goh264.DefaultEncoderConfig(16, 16)
		cfg.DeblockMode = goh264.EncoderDeblockDisabled
		enc, err := goh264.NewEncoder(cfg)
		if err != nil {
			t.Fatalf("NewEncoder: %v", err)
		}
		first, err := enc.Encode(patternedI420EncoderFrame(16, 16))
		if err != nil {
			t.Fatalf("Encode first IDR: %v", err)
		}
		if !first.IDR || enc.PendingIDR() {
			t.Fatalf("first frame idr=%v pending=%v, want completed IDR", first.IDR, enc.PendingIDR())
		}

		enc.ForceIDR()
		if !enc.PendingIDR() {
			t.Fatal("ForceIDR did not queue IDR before AVC reconfigure")
		}
		if err := enc.Reconfigure(goh264.EncoderReconfigure{
			OutputFormat: goh264.EncoderOutputAVC,
			SPSPPSMode:   goh264.EncoderSPSPPSOutOfBand,
		}); err != nil {
			t.Fatalf("Reconfigure AVC output: %v", err)
		}
		if !enc.PendingIDR() {
			t.Fatal("AVC output reconfigure cleared pending IDR")
		}

		frame := patternedI420EncoderFrame(16, 16)
		frame.Y[0] ^= 0x41
		second, err := enc.Encode(frame)
		if err != nil {
			t.Fatalf("Encode after AVC output reconfigure: %v", err)
		}
		if second.Dropped || !second.IDR || enc.PendingIDR() {
			t.Fatalf("AVC output frame dropped=%v idr=%v pending=%v, want delivered IDR",
				second.Dropped, second.IDR, enc.PendingIDR())
		}
		if len(second.RTPPackets) != 0 {
			t.Fatalf("AVC output RTP packets = %d, want 0", len(second.RTPPackets))
		}
		assertEncoderNALTypes(t, second.NALUnits, []uint8{5})
	})

	t.Run("rtp metadata", func(t *testing.T) {
		cfg := goh264.DefaultEncoderConfig(16, 16)
		cfg.DeblockMode = goh264.EncoderDeblockDisabled
		enc, err := goh264.NewEncoder(cfg)
		if err != nil {
			t.Fatalf("NewEncoder: %v", err)
		}
		first, err := enc.Encode(patternedI420EncoderFrame(16, 16))
		if err != nil {
			t.Fatalf("Encode first IDR: %v", err)
		}
		if !first.IDR || enc.PendingIDR() {
			t.Fatalf("first frame idr=%v pending=%v, want completed IDR", first.IDR, enc.PendingIDR())
		}

		payloadType := uint8(113)
		ssrc := uint32(0x10293847)
		enc.ForceIDR()
		if !enc.PendingIDR() {
			t.Fatal("ForceIDR did not queue IDR before RTP metadata reconfigure")
		}
		if err := enc.Reconfigure(goh264.EncoderReconfigure{
			RTPPayloadType: &payloadType,
			RTPSSRC:        &ssrc,
		}); err != nil {
			t.Fatalf("Reconfigure RTP metadata: %v", err)
		}
		if !enc.PendingIDR() {
			t.Fatal("RTP metadata reconfigure cleared pending IDR")
		}

		frame := patternedI420EncoderFrame(16, 16)
		frame.Y[0] ^= 0x37
		second, err := enc.Encode(frame)
		if err != nil {
			t.Fatalf("Encode after RTP metadata reconfigure: %v", err)
		}
		if second.Dropped || !second.IDR || enc.PendingIDR() {
			t.Fatalf("RTP metadata frame dropped=%v idr=%v pending=%v, want delivered IDR",
				second.Dropped, second.IDR, enc.PendingIDR())
		}
		assertEncoderNALTypes(t, second.NALUnits, []uint8{7, 8, 5})
		assertRTPPacketMetadata(t, second.RTPPackets, payloadType, ssrc, uint16(len(first.RTPPackets)))
	})
}

func TestEncoderInvalidReconfigurePreservesPendingIDR(t *testing.T) {
	mode0 := goh264.EncoderRTPPacketizationSingleNAL
	stapa := true
	badPayloadType := uint8(128)
	tests := []struct {
		name    string
		update  goh264.EncoderReconfigure
		wantErr error
	}{
		{name: "bad bitrate", update: goh264.EncoderReconfigure{MaxBitrate: 1}, wantErr: goh264.ErrInvalidData},
		{name: "bad frame rate", update: goh264.EncoderReconfigure{FrameRateNum: 0, FrameRateDen: 1}, wantErr: goh264.ErrInvalidData},
		{name: "bad payload size", update: goh264.EncoderReconfigure{RTPMaxPayloadSize: 2}, wantErr: goh264.ErrInvalidData},
		{name: "bad deblock", update: goh264.EncoderReconfigure{DeblockMode: goh264.EncoderDeblockMode(99)}, wantErr: goh264.ErrInvalidData},
		{name: "bad output format", update: goh264.EncoderReconfigure{OutputFormat: goh264.EncoderOutputFormat(99)}, wantErr: goh264.ErrInvalidData},
		{name: "mode-0 STAP-A", update: goh264.EncoderReconfigure{
			RTPPacketizationMode: &mode0,
			STAPA:                &stapa,
		}, wantErr: goh264.ErrUnsupported},
		{name: "bad RTP payload type", update: goh264.EncoderReconfigure{RTPPayloadType: &badPayloadType}, wantErr: goh264.ErrInvalidData},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enc, err := goh264.NewEncoder(goh264.DefaultEncoderConfig(16, 16))
			if err != nil {
				t.Fatalf("NewEncoder: %v", err)
			}
			first, err := enc.Encode(patternedI420EncoderFrame(16, 16))
			if err != nil {
				t.Fatalf("Encode first IDR: %v", err)
			}
			if !first.IDR || enc.PendingIDR() {
				t.Fatalf("first frame idr=%v pending=%v, want completed IDR", first.IDR, enc.PendingIDR())
			}

			enc.ForceIDR()
			if !enc.PendingIDR() {
				t.Fatalf("%s ForceIDR did not queue IDR", tt.name)
			}
			before := enc.Config()
			if err := enc.Reconfigure(tt.update); !errors.Is(err, tt.wantErr) {
				t.Fatalf("%s invalid reconfigure error = %v, want %v", tt.name, err, tt.wantErr)
			}
			if got := enc.Config(); got != before {
				t.Fatalf("%s invalid reconfigure mutated config = %+v, want %+v", tt.name, got, before)
			}
			if !enc.PendingIDR() {
				t.Fatalf("%s invalid reconfigure cleared pending IDR", tt.name)
			}

			secondFrame := patternedI420EncoderFrame(16, 16)
			secondFrame.Y[0] ^= 0x21
			second, err := enc.Encode(secondFrame)
			if err != nil {
				t.Fatalf("%s Encode after invalid reconfigure: %v", tt.name, err)
			}
			if !second.IDR || enc.PendingIDR() {
				t.Fatalf("%s post-invalid-reconfigure frame idr=%v pending=%v, want delivered IDR",
					tt.name, second.IDR, enc.PendingIDR())
			}
			assertEncoderNALTypes(t, second.NALUnits, []uint8{7, 8, 5})
		})
	}
}

func TestEncoderFrameRateInvalidUpdatesPreserveLiveState(t *testing.T) {
	cfg := goh264.DefaultEncoderConfig(16, 16)
	cfg.DeblockMode = goh264.EncoderDeblockDisabled
	cfg.RTPMaxPayloadSize = 32
	enc, err := goh264.NewEncoder(cfg)
	if err != nil {
		t.Fatalf("NewEncoder: %v", err)
	}

	var callbackCalls int
	enc.SetRTPPacketCallback(func(goh264.EncoderRTPPacket, goh264.EncoderRTPPacketMetadata) {
		callbackCalls++
	})
	frame := patternedI420EncoderFrame(16, 16)
	frame.PTS = 0
	first, err := enc.Encode(frame)
	if err != nil {
		t.Fatalf("Encode first IDR: %v", err)
	}
	if first.Dropped || !first.IDR || first.RTPTime != 0 {
		t.Fatalf("first output dropped/id/time = %v/%v/%d, want IDR time 0",
			first.Dropped, first.IDR, first.RTPTime)
	}
	assertEncoderNALTypes(t, first.NALUnits, []uint8{7, 8, 5})
	firstPacketCount := len(first.RTPPackets)
	if firstPacketCount == 0 || callbackCalls != firstPacketCount {
		t.Fatalf("first RTP packets/callbacks = %d/%d, want nonzero matching count",
			firstPacketCount, callbackCalls)
	}
	before := enc.Config()

	if err := enc.SetFrameRate(0, 1); !errors.Is(err, goh264.ErrInvalidData) {
		t.Fatalf("SetFrameRate invalid error = %v, want ErrInvalidData", err)
	}
	if got := enc.Config(); got != before {
		t.Fatalf("invalid SetFrameRate mutated config = %+v, want %+v", got, before)
	}
	if err := enc.Reconfigure(goh264.EncoderReconfigure{FrameRateDen: 1}); !errors.Is(err, goh264.ErrInvalidData) {
		t.Fatalf("Reconfigure zero frame-rate numerator error = %v, want ErrInvalidData", err)
	}
	if got := enc.Config(); got != before {
		t.Fatalf("invalid frame-rate Reconfigure mutated config = %+v, want %+v", got, before)
	}
	if err := enc.Reconfigure(goh264.EncoderReconfigure{
		FrameRateNum: 0,
		FrameRateDen: 1,
		ForceIDR:     true,
	}); !errors.Is(err, goh264.ErrInvalidData) {
		t.Fatalf("Reconfigure zero frame-rate numerator with ForceIDR error = %v, want ErrInvalidData", err)
	}
	if got := enc.Config(); got != before {
		t.Fatalf("invalid frame-rate ForceIDR Reconfigure mutated config = %+v, want %+v", got, before)
	}
	if enc.PendingIDR() {
		t.Fatal("invalid frame-rate updates queued unexpected IDR")
	}
	if callbackCalls != firstPacketCount {
		t.Fatalf("invalid frame-rate updates invoked callbacks = %d, want still %d",
			callbackCalls, firstPacketCount)
	}

	second, err := enc.Encode(frame)
	if err != nil {
		t.Fatalf("Encode after invalid frame-rate updates: %v", err)
	}
	if second.Dropped || second.IDR || second.RTPTime != before.RTPTimestampIncrement {
		t.Fatalf("post-invalid output dropped/id/time = %v/%v/%d, want P-skip time %d",
			second.Dropped, second.IDR, second.RTPTime, before.RTPTimestampIncrement)
	}
	assertEncoderNALTypes(t, second.NALUnits, []uint8{1})
	assertEncoderVCLFrameNums(t,
		append(append([]byte(nil), first.Data...), second.Data...),
		[]uint8{5, 1},
		[]uint32{0, 1},
	)
	assertRTPPacketMetadata(t, second.RTPPackets, before.RTPPayloadType, before.RTPSSRC, uint16(firstPacketCount))
	if callbackCalls != firstPacketCount+len(second.RTPPackets) {
		t.Fatalf("post-invalid callbacks = %d, want %d",
			callbackCalls, firstPacketCount+len(second.RTPPackets))
	}
}

func TestEncoderReconfigureInvalidLatencyUpdatesPreserveLiveState(t *testing.T) {
	cfg := goh264.DefaultEncoderConfig(16, 16)
	cfg.DeblockMode = goh264.EncoderDeblockDisabled
	cfg.RTPMaxPayloadSize = 32
	enc, err := goh264.NewEncoder(cfg)
	if err != nil {
		t.Fatalf("NewEncoder: %v", err)
	}

	var callbackCalls int
	enc.SetRTPPacketCallback(func(goh264.EncoderRTPPacket, goh264.EncoderRTPPacketMetadata) {
		callbackCalls++
	})
	frame := patternedI420EncoderFrame(16, 16)
	frame.PTS = 0
	first, err := enc.Encode(frame)
	if err != nil {
		t.Fatalf("Encode first IDR: %v", err)
	}
	if first.Dropped || !first.IDR || first.RTPTime != 0 {
		t.Fatalf("first output dropped/id/time = %v/%v/%d, want IDR time 0",
			first.Dropped, first.IDR, first.RTPTime)
	}
	assertEncoderNALTypes(t, first.NALUnits, []uint8{7, 8, 5})
	firstPacketCount := len(first.RTPPackets)
	if firstPacketCount == 0 || callbackCalls != firstPacketCount {
		t.Fatalf("first RTP packets/callbacks = %d/%d, want nonzero matching count",
			firstPacketCount, callbackCalls)
	}
	before := enc.Config()

	for _, tt := range []struct {
		name   string
		update goh264.EncoderReconfigure
	}{
		{name: "negative encode-time budget", update: goh264.EncoderReconfigure{
			MaxEncodeTimeUS: -1,
			ForceIDR:        true,
		}},
		{name: "negative slice count", update: goh264.EncoderReconfigure{
			SliceCount: -1,
			ForceIDR:   true,
		}},
		{name: "negative slice byte target", update: goh264.EncoderReconfigure{
			SliceMaxBytes: -1,
			ForceIDR:      true,
		}},
		{name: "slice count beyond macroblocks", update: goh264.EncoderReconfigure{
			SliceCount: 2,
			ForceIDR:   true,
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if err := enc.Reconfigure(tt.update); !errors.Is(err, goh264.ErrInvalidData) {
				t.Fatalf("Reconfigure invalid latency controls error = %v, want ErrInvalidData", err)
			}
			if got := enc.Config(); got != before {
				t.Fatalf("invalid latency controls mutated config = %+v, want %+v", got, before)
			}
			if enc.PendingIDR() {
				t.Fatal("invalid latency controls queued unexpected IDR")
			}
			if callbackCalls != firstPacketCount {
				t.Fatalf("invalid latency controls invoked callbacks = %d, want still %d",
					callbackCalls, firstPacketCount)
			}
		})
	}

	second, err := enc.Encode(frame)
	if err != nil {
		t.Fatalf("Encode after invalid latency updates: %v", err)
	}
	if second.Dropped || second.IDR || second.RTPTime != before.RTPTimestampIncrement {
		t.Fatalf("post-invalid output dropped/id/time = %v/%v/%d, want P-skip time %d",
			second.Dropped, second.IDR, second.RTPTime, before.RTPTimestampIncrement)
	}
	assertEncoderNALTypes(t, second.NALUnits, []uint8{1})
	assertEncoderVCLFrameNums(t,
		append(append([]byte(nil), first.Data...), second.Data...),
		[]uint8{5, 1},
		[]uint32{0, 1},
	)
	assertRTPPacketMetadata(t, second.RTPPackets, before.RTPPayloadType, before.RTPSSRC, uint16(firstPacketCount))
	if callbackCalls != firstPacketCount+len(second.RTPPackets) {
		t.Fatalf("post-invalid callbacks = %d, want %d",
			callbackCalls, firstPacketCount+len(second.RTPPackets))
	}
}

func TestEncoderReconfigureSwitchesWebRTCPacketizationControls(t *testing.T) {
	cfg := goh264.DefaultEncoderConfig(16, 16)
	cfg.RTPMaxPayloadSize = 32
	cfg.STAPA = true
	cfg.DeblockMode = goh264.EncoderDeblockDisabled
	enc, err := goh264.NewEncoder(cfg)
	if err != nil {
		t.Fatalf("NewEncoder: %v", err)
	}

	firstFrame := patternedI420EncoderFrame(16, 16)
	firstFrame.PTS = 0
	firstFrame.Duration = 0
	first, err := enc.Encode(firstFrame)
	if err != nil {
		t.Fatalf("Encode first mode-1 RTP frame: %v", err)
	}
	var sawSTAPA, sawFUA bool
	for _, pkt := range first.RTPPackets {
		switch pkt.Payload[0] & 0x1f {
		case 24:
			sawSTAPA = true
		case 28:
			sawFUA = true
		}
	}
	if !sawSTAPA || !sawFUA {
		t.Fatalf("first RTP payload forms STAP-A/FU-A = %v/%v, want both before reconfigure", sawSTAPA, sawFUA)
	}

	mode0 := goh264.EncoderRTPPacketizationSingleNAL
	stapa := false
	payloadType := uint8(110)
	ssrc := uint32(0x11223344)
	if err := enc.Reconfigure(goh264.EncoderReconfigure{
		RTPMaxPayloadSize:     1200,
		RTPPacketizationMode:  &mode0,
		STAPA:                 &stapa,
		RTPPayloadType:        &payloadType,
		RTPSSRC:               &ssrc,
		RTPTimestampIncrement: 9000,
		ForceIDR:              true,
	}); err != nil {
		t.Fatalf("Reconfigure RTP mode 0: %v", err)
	}
	got := enc.Config()
	if got.RTPPacketizationMode != goh264.EncoderRTPPacketizationSingleNAL ||
		got.STAPA ||
		got.RTPPayloadType != payloadType ||
		got.RTPSSRC != ssrc ||
		got.RTPTimestampIncrement != 9000 ||
		got.RTPMaxPayloadSize != 1200 {
		t.Fatalf("reconfigured RTP controls = %+v", got)
	}

	secondFrame := firstFrame
	second, err := enc.Encode(secondFrame)
	if err != nil {
		t.Fatalf("Encode forced mode-0 IDR: %v", err)
	}
	if second.RTPTime != cfg.RTPTimestampIncrement {
		t.Fatalf("second RTP time = %d, want prior next timestamp %d", second.RTPTime, cfg.RTPTimestampIncrement)
	}
	assertEncoderNALTypes(t, second.NALUnits, []uint8{7, 8, 5})
	assertRTPPacketMetadata(t, second.RTPPackets, payloadType, ssrc, uint16(len(first.RTPPackets)))
	for i, pkt := range second.RTPPackets {
		if typ := pkt.Payload[0] & 0x1f; typ == 24 || typ == 28 {
			t.Fatalf("second packet[%d] payload type = %d, want mode-0 single raw NAL", i, typ)
		}
		if pkt.Marker != (i == len(second.RTPPackets)-1) {
			t.Fatalf("second packet[%d] marker = %v, want only final marker", i, pkt.Marker)
		}
	}

	thirdFrame := firstFrame
	third, err := enc.Encode(thirdFrame)
	if err != nil {
		t.Fatalf("Encode post-reconfigure P-skip: %v", err)
	}
	if third.RTPTime != second.RTPTime+9000 {
		t.Fatalf("third RTP time = %d, want updated increment from second %d", third.RTPTime, second.RTPTime+9000)
	}
	assertEncoderNALTypes(t, third.NALUnits, []uint8{1})
	assertRTPPacketMetadata(t, third.RTPPackets, payloadType, ssrc, uint16(len(first.RTPPackets)+len(second.RTPPackets)))
}

func TestEncoderSetRTPMaxPayloadSizeRetargetsLivePacketization(t *testing.T) {
	cfg := goh264.DefaultEncoderConfig(16, 16)
	cfg.RTPMaxPayloadSize = 1200
	cfg.DeblockMode = goh264.EncoderDeblockDisabled
	enc, err := goh264.NewEncoder(cfg)
	if err != nil {
		t.Fatalf("NewEncoder: %v", err)
	}

	firstFrame := patternedI420EncoderFrame(16, 16)
	firstFrame.PTS = 30_000
	first, err := enc.Encode(firstFrame)
	if err != nil {
		t.Fatalf("Encode first RTP IDR: %v", err)
	}
	assertEncoderNALTypes(t, first.NALUnits, []uint8{7, 8, 5})
	assertEncoderRTPSingleNALPackets(t, first, cfg.RTPMaxPayloadSize)
	assertRTPPacketMetadata(t, first.RTPPackets, cfg.RTPPayloadType, cfg.RTPSSRC, 0)

	if err := enc.SetRTPMaxPayloadSize(32); err != nil {
		t.Fatalf("SetRTPMaxPayloadSize: %v", err)
	}
	if got := enc.Config(); got.RTPMaxPayloadSize != 32 {
		t.Fatalf("RTP max payload size = %d, want 32", got.RTPMaxPayloadSize)
	}

	secondFrame := patternedI420EncoderFrame(16, 16)
	secondFrame.Y[0] ^= 0x4b
	secondFrame.PTS = firstFrame.PTS + int64(cfg.RTPTimestampIncrement)
	second, err := enc.Encode(secondFrame)
	if err != nil {
		t.Fatalf("Encode small-payload changed P: %v", err)
	}
	if second.IDR || second.RTPTime != uint32(secondFrame.PTS) {
		t.Fatalf("second output idr=%v rtp=%d, want non-IDR RTP time %d",
			second.IDR, second.RTPTime, secondFrame.PTS)
	}
	assertEncoderNALTypes(t, second.NALUnits, []uint8{6, 1})
	assertEncoderRTPPayloadLimit(t, second.RTPPackets, 32)
	assertEncoderRTPHasFUA(t, second.RTPPackets)
	assertRTPPacketTimestamps(t, second.RTPPackets, second.RTPTime)
	assertRTPPacketMetadata(t, second.RTPPackets, cfg.RTPPayloadType, cfg.RTPSSRC, uint16(len(first.RTPPackets)))

	beforeInvalid := enc.Config()
	if err := enc.SetRTPMaxPayloadSize(2); !errors.Is(err, goh264.ErrInvalidData) {
		t.Fatalf("SetRTPMaxPayloadSize invalid error = %v, want ErrInvalidData", err)
	}
	if got := enc.Config(); got != beforeInvalid {
		t.Fatalf("invalid SetRTPMaxPayloadSize mutated config = %+v, want %+v", got, beforeInvalid)
	}

	thirdFrame := secondFrame
	thirdFrame.Y = append([]byte(nil), secondFrame.Y...)
	thirdFrame.Y[1] ^= 0x27
	thirdFrame.PTS = secondFrame.PTS + int64(cfg.RTPTimestampIncrement)
	third, err := enc.Encode(thirdFrame)
	if err != nil {
		t.Fatalf("Encode after invalid payload-size update: %v", err)
	}
	if third.IDR || third.RTPTime != uint32(thirdFrame.PTS) {
		t.Fatalf("third output idr=%v rtp=%d, want non-IDR RTP time %d",
			third.IDR, third.RTPTime, thirdFrame.PTS)
	}
	assertEncoderNALTypes(t, third.NALUnits, []uint8{6, 1})
	assertEncoderRTPPayloadLimit(t, third.RTPPackets, 32)
	assertEncoderRTPHasFUA(t, third.RTPPackets)
	assertRTPPacketTimestamps(t, third.RTPPackets, third.RTPTime)
	assertRTPPacketMetadata(t, third.RTPPackets, cfg.RTPPayloadType, cfg.RTPSSRC, uint16(len(first.RTPPackets)+len(second.RTPPackets)))

	dec := goh264.NewDecoder()
	decodedFirst, err := dec.DecodeFrames(annexBFromEncoderRTPPackets(t, first.RTPPackets))
	if err != nil {
		t.Fatalf("DecodeFrames first RTP IDR: %v", err)
	}
	assertDecodedEncoderFrameBytes(t, decodedFirst, appendI420FrameBytes(nil, firstFrame))
	decodedSecond, err := dec.DecodeFrames(annexBFromEncoderRTPPackets(t, second.RTPPackets))
	if err != nil {
		t.Fatalf("DecodeFrames retargeted RTP changed P: %v", err)
	}
	assertDecodedEncoderFrameBytes(t, decodedSecond, appendI420FrameBytes(nil, secondFrame))
	decodedThird, err := dec.DecodeFrames(annexBFromEncoderRTPPackets(t, third.RTPPackets))
	if err != nil {
		t.Fatalf("DecodeFrames after invalid payload-size update: %v", err)
	}
	assertDecodedEncoderFrameBytes(t, decodedThird, appendI420FrameBytes(nil, thirdFrame))
}

func TestEncoderReconfigureSwitchesOutputFormatForForcedIDR(t *testing.T) {
	cfg := goh264.DefaultEncoderConfig(16, 16)
	cfg.DeblockMode = goh264.EncoderDeblockDisabled
	enc, err := goh264.NewEncoder(cfg)
	if err != nil {
		t.Fatalf("NewEncoder: %v", err)
	}

	firstFrame := patternedI420EncoderFrame(16, 16)
	if _, err := enc.Encode(firstFrame); err != nil {
		t.Fatalf("Encode first RTP frame: %v", err)
	}

	if err := enc.Reconfigure(goh264.EncoderReconfigure{
		OutputFormat: goh264.EncoderOutputAnnexB,
		SPSPPSMode:   goh264.EncoderSPSPPSEveryIDR,
		ForceIDR:     true,
	}); err != nil {
		t.Fatalf("Reconfigure Annex B output: %v", err)
	}
	secondFrame := firstFrame
	secondFrame.PTS += int64(cfg.RTPTimestampIncrement)
	second, err := enc.Encode(secondFrame)
	if err != nil {
		t.Fatalf("Encode forced Annex B IDR: %v", err)
	}
	if len(second.RTPPackets) != 0 {
		t.Fatalf("Annex B output returned RTP packets: %d", len(second.RTPPackets))
	}
	assertEncoderNALTypes(t, second.NALUnits, []uint8{7, 8, 5})
	decoded, err := goh264.NewDecoder().DecodeAnnexBFrames(second.Data)
	if err != nil {
		t.Fatalf("DecodeAnnexBFrames reconfigured IDR: %v", err)
	}
	assertDecodedEncoderFrameBytes(t, decoded, appendI420FrameBytes(nil, secondFrame))
}

func TestEncoderReconfigureSwitchesOutputFormatToAVCForForcedIDR(t *testing.T) {
	cfg := goh264.DefaultEncoderConfig(16, 16)
	cfg.DeblockMode = goh264.EncoderDeblockDisabled
	enc, err := goh264.NewEncoder(cfg)
	if err != nil {
		t.Fatalf("NewEncoder: %v", err)
	}

	var callbackCalls int
	enc.SetRTPPacketCallback(func(goh264.EncoderRTPPacket, goh264.EncoderRTPPacketMetadata) {
		callbackCalls++
	})

	firstFrame := patternedI420EncoderFrame(16, 16)
	first, err := enc.Encode(firstFrame)
	if err != nil {
		t.Fatalf("Encode first RTP IDR: %v", err)
	}
	assertEncoderNALTypes(t, first.NALUnits, []uint8{7, 8, 5})
	firstPacketCount := len(first.RTPPackets)
	if firstPacketCount == 0 || callbackCalls != firstPacketCount {
		t.Fatalf("first RTP packets/callbacks = %d/%d, want nonzero matching count",
			firstPacketCount, callbackCalls)
	}

	if err := enc.Reconfigure(goh264.EncoderReconfigure{
		OutputFormat: goh264.EncoderOutputAVC,
		SPSPPSMode:   goh264.EncoderSPSPPSOutOfBand,
		ForceIDR:     true,
	}); err != nil {
		t.Fatalf("Reconfigure AVC output: %v", err)
	}
	got := enc.Config()
	if got.OutputFormat != goh264.EncoderOutputAVC || got.SPSPPSMode != goh264.EncoderSPSPPSOutOfBand {
		t.Fatalf("reconfigured output controls = format %v spspps %v, want AVC/out-of-band",
			got.OutputFormat, got.SPSPPSMode)
	}

	secondFrame := firstFrame
	secondFrame.PTS = firstFrame.PTS + int64(cfg.RTPTimestampIncrement)
	second, err := enc.Encode(secondFrame)
	if err != nil {
		t.Fatalf("Encode forced AVC IDR: %v", err)
	}
	if !second.IDR || second.RTPTime != first.RTPTime+cfg.RTPTimestampIncrement {
		t.Fatalf("forced AVC IDR/time = %v/%d, want IDR/%d",
			second.IDR, second.RTPTime, first.RTPTime+cfg.RTPTimestampIncrement)
	}
	if len(second.RTPPackets) != 0 || callbackCalls != firstPacketCount {
		t.Fatalf("forced AVC RTP packets/callbacks = %d/%d, want 0/%d",
			len(second.RTPPackets), callbackCalls, firstPacketCount)
	}
	assertEncoderNALTypes(t, second.NALUnits, []uint8{5})

	headers, err := enc.ParameterSets()
	if err != nil {
		t.Fatalf("ParameterSets: %v", err)
	}
	dec := goh264.NewDecoder()
	if _, err := dec.ParseAVCDecoderConfigurationRecord(headers.AVCDecoderConfigurationRecord); err != nil {
		t.Fatalf("ParseAVCDecoderConfigurationRecord: %v", err)
	}
	decodedSecond, err := dec.DecodeConfiguredAVCFrames(second.Data)
	if err != nil {
		t.Fatalf("DecodeConfiguredAVCFrames forced IDR: %v", err)
	}
	assertDecodedEncoderFrameBytes(t, decodedSecond, appendI420FrameBytes(nil, secondFrame))

	thirdFrame := secondFrame
	thirdFrame.PTS = secondFrame.PTS + int64(cfg.RTPTimestampIncrement)
	third, err := enc.Encode(thirdFrame)
	if err != nil {
		t.Fatalf("Encode AVC P-skip after reconfigure: %v", err)
	}
	if third.Dropped || third.IDR || third.RTPTime != second.RTPTime+cfg.RTPTimestampIncrement {
		t.Fatalf("post-reconfigure AVC frame dropped/id/time = %v/%v/%d, want P-skip time %d",
			third.Dropped, third.IDR, third.RTPTime, second.RTPTime+cfg.RTPTimestampIncrement)
	}
	if len(third.RTPPackets) != 0 || callbackCalls != firstPacketCount {
		t.Fatalf("post-reconfigure AVC RTP packets/callbacks = %d/%d, want 0/%d",
			len(third.RTPPackets), callbackCalls, firstPacketCount)
	}
	assertEncoderNALTypes(t, third.NALUnits, []uint8{1})
	decodedThird, err := dec.DecodeConfiguredAVCFrames(third.Data)
	if err != nil {
		t.Fatalf("DecodeConfiguredAVCFrames P-skip: %v", err)
	}
	assertDecodedEncoderFrameBytes(t, decodedThird, appendI420FrameBytes(nil, thirdFrame))
}

func TestEncoderReconfigureSwitchesOutputFormatFromAVCToRTPForForcedIDR(t *testing.T) {
	cfg := goh264.DefaultEncoderConfig(16, 16)
	cfg.OutputFormat = goh264.EncoderOutputAVC
	cfg.SPSPPSMode = goh264.EncoderSPSPPSOutOfBand
	cfg.RTPMaxPayloadSize = 0
	cfg.DeblockMode = goh264.EncoderDeblockDisabled
	enc, err := goh264.NewEncoder(cfg)
	if err != nil {
		t.Fatalf("NewEncoder: %v", err)
	}

	var callbackCalls int
	enc.SetRTPPacketCallback(func(goh264.EncoderRTPPacket, goh264.EncoderRTPPacketMetadata) {
		callbackCalls++
	})

	firstFrame := patternedI420EncoderFrame(16, 16)
	first, err := enc.Encode(firstFrame)
	if err != nil {
		t.Fatalf("Encode first AVC IDR: %v", err)
	}
	if len(first.RTPPackets) != 0 || callbackCalls != 0 {
		t.Fatalf("initial AVC RTP packets/callbacks = %d/%d, want 0/0",
			len(first.RTPPackets), callbackCalls)
	}
	assertEncoderNALTypes(t, first.NALUnits, []uint8{5})

	headers, err := enc.ParameterSets()
	if err != nil {
		t.Fatalf("ParameterSets: %v", err)
	}
	avcDec := goh264.NewDecoder()
	if _, err := avcDec.ParseAVCDecoderConfigurationRecord(headers.AVCDecoderConfigurationRecord); err != nil {
		t.Fatalf("ParseAVCDecoderConfigurationRecord: %v", err)
	}
	decodedFirst, err := avcDec.DecodeConfiguredAVCFrames(first.Data)
	if err != nil {
		t.Fatalf("DecodeConfiguredAVCFrames first IDR: %v", err)
	}
	assertDecodedEncoderFrameBytes(t, decodedFirst, appendI420FrameBytes(nil, firstFrame))

	stapa := true
	payloadType := uint8(112)
	ssrc := uint32(0x55667788)
	if err := enc.Reconfigure(goh264.EncoderReconfigure{
		OutputFormat:      goh264.EncoderOutputRTP,
		SPSPPSMode:        goh264.EncoderSPSPPSEveryIDR,
		RTPMaxPayloadSize: 32,
		STAPA:             &stapa,
		RTPPayloadType:    &payloadType,
		RTPSSRC:           &ssrc,
		ForceIDR:          true,
	}); err != nil {
		t.Fatalf("Reconfigure RTP output: %v", err)
	}
	got := enc.Config()
	if got.OutputFormat != goh264.EncoderOutputRTP ||
		got.SPSPPSMode != goh264.EncoderSPSPPSEveryIDR ||
		got.RTPMaxPayloadSize != 32 ||
		!got.STAPA ||
		got.RTPPayloadType != payloadType ||
		got.RTPSSRC != ssrc {
		t.Fatalf("reconfigured RTP controls = %+v, want RTP/every-IDR/STAP-A/metadata update", got)
	}

	secondFrame := firstFrame
	secondFrame.PTS = firstFrame.PTS + int64(cfg.RTPTimestampIncrement)
	second, err := enc.Encode(secondFrame)
	if err != nil {
		t.Fatalf("Encode forced RTP IDR: %v", err)
	}
	if !second.IDR || second.RTPTime != first.RTPTime+cfg.RTPTimestampIncrement {
		t.Fatalf("forced RTP IDR/time = %v/%d, want IDR/%d",
			second.IDR, second.RTPTime, first.RTPTime+cfg.RTPTimestampIncrement)
	}
	assertEncoderNALTypes(t, second.NALUnits, []uint8{7, 8, 5})
	assertRTPPacketMetadata(t, second.RTPPackets, payloadType, ssrc, 0)
	assertRTPPacketTimestamps(t, second.RTPPackets, second.RTPTime)
	if callbackCalls != len(second.RTPPackets) {
		t.Fatalf("RTP re-entry callbacks = %d, want packet count %d", callbackCalls, len(second.RTPPackets))
	}
	var sawSTAPA, sawFUA bool
	for _, pkt := range second.RTPPackets {
		switch pkt.Payload[0] & 0x1f {
		case 24:
			sawSTAPA = true
		case 28:
			sawFUA = true
		}
	}
	if !sawSTAPA || !sawFUA {
		t.Fatalf("RTP re-entry payload forms STAP-A/FU-A = %v/%v, want both", sawSTAPA, sawFUA)
	}

	rtpDec := goh264.NewDecoder()
	decodedSecond, err := rtpDec.DecodeFrames(annexBFromEncoderRTPPackets(t, second.RTPPackets))
	if err != nil {
		t.Fatalf("Decode RTP re-entry IDR: %v", err)
	}
	assertDecodedEncoderFrameBytes(t, decodedSecond, appendI420FrameBytes(nil, secondFrame))

	thirdFrame := secondFrame
	thirdFrame.PTS = secondFrame.PTS + int64(cfg.RTPTimestampIncrement)
	third, err := enc.Encode(thirdFrame)
	if err != nil {
		t.Fatalf("Encode RTP P-skip after AVC re-entry: %v", err)
	}
	if third.Dropped || third.IDR || third.RTPTime != second.RTPTime+cfg.RTPTimestampIncrement {
		t.Fatalf("post-re-entry RTP frame dropped/id/time = %v/%v/%d, want P-skip time %d",
			third.Dropped, third.IDR, third.RTPTime, second.RTPTime+cfg.RTPTimestampIncrement)
	}
	assertEncoderNALTypes(t, third.NALUnits, []uint8{1})
	assertRTPPacketMetadata(t, third.RTPPackets, payloadType, ssrc, uint16(len(second.RTPPackets)))
	assertRTPPacketTimestamps(t, third.RTPPackets, third.RTPTime)
	if callbackCalls != len(second.RTPPackets)+len(third.RTPPackets) {
		t.Fatalf("post-re-entry RTP callbacks = %d, want %d",
			callbackCalls, len(second.RTPPackets)+len(third.RTPPackets))
	}
	decodedThird, err := rtpDec.DecodeFrames(annexBFromEncoderRTPPackets(t, third.RTPPackets))
	if err != nil {
		t.Fatalf("Decode RTP P-skip after AVC re-entry: %v", err)
	}
	assertDecodedEncoderFrameBytes(t, decodedThird, appendI420FrameBytes(nil, thirdFrame))
}

func TestEncoderReconfigureResolutionResetsReferenceAndQueuesIDR(t *testing.T) {
	cfg := goh264.DefaultEncoderConfig(16, 16)
	cfg.DeblockMode = goh264.EncoderDeblockDisabled
	cfg.RTPMaxPayloadSize = 96
	enc, err := goh264.NewEncoder(cfg)
	if err != nil {
		t.Fatalf("NewEncoder: %v", err)
	}

	firstFrame := patternedI420EncoderFrame(16, 16)
	firstFrame.PTS = 9_000
	first, err := enc.Encode(firstFrame)
	if err != nil {
		t.Fatalf("Encode first RTP IDR: %v", err)
	}
	if !first.IDR {
		t.Fatalf("first frame idr=%v, want IDR", first.IDR)
	}
	assertEncoderNALTypes(t, first.NALUnits, []uint8{7, 8, 5})
	assertRTPPacketMetadata(t, first.RTPPackets, cfg.RTPPayloadType, cfg.RTPSSRC, 0)

	secondFrame := firstFrame
	secondFrame.PTS += int64(cfg.RTPTimestampIncrement)
	second, err := enc.Encode(secondFrame)
	if err != nil {
		t.Fatalf("Encode same-size P-skip: %v", err)
	}
	if second.IDR {
		t.Fatalf("same-size frame idr=%v, want P-skip before resolution reset", second.IDR)
	}
	assertEncoderNALTypes(t, second.NALUnits, []uint8{1})
	assertRTPPacketMetadata(t, second.RTPPackets, cfg.RTPPayloadType, cfg.RTPSSRC, uint16(len(first.RTPPackets)))

	if err := enc.Reconfigure(goh264.EncoderReconfigure{Width: 32, Height: 16}); err != nil {
		t.Fatalf("Reconfigure resolution: %v", err)
	}
	if got := enc.Config(); got.Width != 32 || got.Height != 16 ||
		got.StrideY != 32 || got.StrideCb != 16 || got.StrideCr != 16 {
		t.Fatalf("resolution config = %+v, want 32x16 with matching I420 strides", got)
	}
	if !enc.PendingIDR() {
		t.Fatal("resolution reconfigure did not queue an IDR")
	}

	staleFrame := firstFrame
	staleFrame.PTS = secondFrame.PTS + int64(cfg.RTPTimestampIncrement)
	if _, err := enc.Encode(staleFrame); !errors.Is(err, goh264.ErrInvalidData) {
		t.Fatalf("Encode stale-size frame error = %v, want ErrInvalidData", err)
	}
	if !enc.PendingIDR() {
		t.Fatal("stale-size frame consumed queued resolution-reset IDR")
	}

	resizedFrame := patternedI420EncoderFrame(32, 16)
	resizedFrame.PTS = staleFrame.PTS
	resized, err := enc.Encode(resizedFrame)
	if err != nil {
		t.Fatalf("Encode resized IDR: %v", err)
	}
	if !resized.IDR || enc.PendingIDR() {
		t.Fatalf("resized frame idr=%v pending=%v, want completed IDR", resized.IDR, enc.PendingIDR())
	}
	assertEncoderNALTypes(t, resized.NALUnits, []uint8{7, 8, 5})
	assertRTPPacketMetadata(t, resized.RTPPackets, cfg.RTPPayloadType, cfg.RTPSSRC, uint16(len(first.RTPPackets)+len(second.RTPPackets)))
	assertRTPPacketTimestamps(t, resized.RTPPackets, resized.RTPTime)

	resizedPSkipFrame := resizedFrame
	resizedPSkipFrame.PTS += int64(cfg.RTPTimestampIncrement)
	resizedPSkip, err := enc.Encode(resizedPSkipFrame)
	if err != nil {
		t.Fatalf("Encode resized P-skip: %v", err)
	}
	if resizedPSkip.IDR {
		t.Fatalf("resized follow-up frame idr=%v, want P-skip", resizedPSkip.IDR)
	}
	assertEncoderNALTypes(t, resizedPSkip.NALUnits, []uint8{1})
	assertRTPPacketMetadata(t, resizedPSkip.RTPPackets, cfg.RTPPayloadType, cfg.RTPSSRC, uint16(len(first.RTPPackets)+len(second.RTPPackets)+len(resized.RTPPackets)))
	assertRTPPacketTimestamps(t, resizedPSkip.RTPPackets, resizedPSkip.RTPTime)

	dec := goh264.NewDecoder()
	decodedFirst, err := dec.DecodeFrames(annexBFromEncoderRTPPackets(t, first.RTPPackets))
	if err != nil {
		t.Fatalf("Decode first RTP IDR: %v", err)
	}
	assertDecodedEncoderFrameBytes(t, decodedFirst, appendI420FrameBytes(nil, firstFrame))
	decodedSecond, err := dec.DecodeFrames(annexBFromEncoderRTPPackets(t, second.RTPPackets))
	if err != nil {
		t.Fatalf("Decode same-size P-skip: %v", err)
	}
	assertDecodedEncoderFrameBytes(t, decodedSecond, appendI420FrameBytes(nil, secondFrame))
	decodedResized, err := dec.DecodeFrames(annexBFromEncoderRTPPackets(t, resized.RTPPackets))
	if err != nil {
		t.Fatalf("Decode resized RTP IDR: %v", err)
	}
	assertDecodedEncoderFrameBytes(t, decodedResized, appendI420FrameBytes(nil, resizedFrame))
	decodedResizedPSkip, err := dec.DecodeFrames(annexBFromEncoderRTPPackets(t, resizedPSkip.RTPPackets))
	if err != nil {
		t.Fatalf("Decode resized P-skip: %v", err)
	}
	assertDecodedEncoderFrameBytes(t, decodedResizedPSkip, appendI420FrameBytes(nil, resizedPSkipFrame))
}

func TestEncoderRealtimeControlLoopStressPreservesPacketAndReferenceState(t *testing.T) {
	cfg := goh264.DefaultEncoderConfig(128, 128)
	cfg.DeblockMode = goh264.EncoderDeblockDisabled
	cfg.RTPMaxPayloadSize = 96
	cfg.STAPA = true
	cfg.RTPPayloadType = 97
	cfg.RTPSSRC = 0x10203040
	cfg.RTPTimestampIncrement = 3000
	enc, err := goh264.NewEncoder(cfg)
	if err != nil {
		t.Fatalf("NewEncoder: %v", err)
	}

	var callbackCalls int
	enc.SetRTPPacketCallback(func(goh264.EncoderRTPPacket, goh264.EncoderRTPPacketMetadata) {
		callbackCalls++
	})

	frame := patternedI420EncoderFrame(128, 128)
	frame.PTS = 0
	frame.Duration = 0
	first, err := enc.Encode(frame)
	if err != nil {
		t.Fatalf("Encode initial RTP IDR: %v", err)
	}
	assertEncoderNALTypes(t, first.NALUnits, []uint8{7, 8, 5})
	assertRTPPacketMetadata(t, first.RTPPackets, cfg.RTPPayloadType, cfg.RTPSSRC, 0)
	assertRTPPacketTimestamps(t, first.RTPPackets, 0)
	firstPacketCount := len(first.RTPPackets)
	if callbackCalls != firstPacketCount {
		t.Fatalf("initial callbacks = %d, want packet count %d", callbackCalls, firstPacketCount)
	}
	var sawSTAPA, sawFUA bool
	for _, pkt := range first.RTPPackets {
		switch pkt.Payload[0] & 0x1f {
		case 24:
			sawSTAPA = true
		case 28:
			sawFUA = true
		}
	}
	if !sawSTAPA || !sawFUA {
		t.Fatalf("initial RTP payload forms STAP-A/FU-A = %v/%v, want both", sawSTAPA, sawFUA)
	}

	initialQP := 31
	minQP := 12
	maxQP := 41
	if err := enc.Reconfigure(goh264.EncoderReconfigure{
		OutputFormat:    goh264.EncoderOutputAnnexB,
		SPSPPSMode:      goh264.EncoderSPSPPSEveryIDR,
		RateControl:     goh264.EncoderRateControlVBR,
		InitialQP:       &initialQP,
		MinQP:           &minQP,
		MaxQP:           &maxQP,
		FrameDrop:       goh264.EncoderFrameDropLate,
		MaxEncodeTimeUS: 10_000_000,
		ForceIDR:        true,
	}); err != nil {
		t.Fatalf("Reconfigure Annex B realtime controls: %v", err)
	}
	got := enc.Config()
	if got.OutputFormat != goh264.EncoderOutputAnnexB ||
		got.SPSPPSMode != goh264.EncoderSPSPPSEveryIDR ||
		got.RateControl != goh264.EncoderRateControlVBR ||
		got.InitialQP != initialQP ||
		got.MinQP != minQP ||
		got.MaxQP != maxQP ||
		got.FrameDrop != goh264.EncoderFrameDropLate ||
		got.MaxEncodeTimeUS != 10_000_000 {
		t.Fatalf("Annex B realtime controls = %+v, want reconfigured state", got)
	}

	second, err := enc.Encode(frame)
	if err != nil {
		t.Fatalf("Encode forced Annex B IDR: %v", err)
	}
	if second.RTPTime != first.RTPTime+cfg.RTPTimestampIncrement {
		t.Fatalf("forced Annex B RTP time = %d, want %d", second.RTPTime, first.RTPTime+cfg.RTPTimestampIncrement)
	}
	if len(second.RTPPackets) != 0 || callbackCalls != firstPacketCount {
		t.Fatalf("forced Annex B packets/callbacks = %d/%d, want 0/%d",
			len(second.RTPPackets), callbackCalls, firstPacketCount)
	}
	assertEncoderNALTypes(t, second.NALUnits, []uint8{7, 8, 5})
	assertEncoderVCLQScales(t, second.Data, []uint8{5}, []uint32{uint32(initialQP)})

	annexBDecoder := goh264.NewDecoder()
	decodedSecond, err := annexBDecoder.DecodeFrames(second.Data)
	if err != nil {
		t.Fatalf("Decode forced Annex B IDR: %v", err)
	}
	assertDecodedEncoderFrameBytes(t, decodedSecond, appendI420FrameBytes(nil, frame))

	if err := enc.Reconfigure(goh264.EncoderReconfigure{MaxEncodeTimeUS: 1}); err != nil {
		t.Fatalf("lower MaxEncodeTimeUS: %v", err)
	}
	lateChangedFrame := patternedI420EncoderFrame(128, 128)
	lateChangedFrame.PTS = 0
	lateChangedFrame.Duration = 0
	lateChangedFrame.Y[0] ^= 0x4c
	dropped, err := enc.Encode(lateChangedFrame)
	if err != nil {
		t.Fatalf("Encode late Annex B changed frame: %v", err)
	}
	if !dropped.Dropped || len(dropped.Data) != 0 || len(dropped.NALUnits) != 0 || len(dropped.RTPPackets) != 0 {
		t.Fatalf("late Annex B dropped frame = %+v, want dropped metadata without output", dropped)
	}
	if dropped.RTPTime != second.RTPTime+cfg.RTPTimestampIncrement {
		t.Fatalf("late Annex B RTP time = %d, want %d", dropped.RTPTime, second.RTPTime+cfg.RTPTimestampIncrement)
	}
	if callbackCalls != firstPacketCount {
		t.Fatalf("late Annex B drop invoked callback count %d, want still %d", callbackCalls, firstPacketCount)
	}

	if err := enc.Reconfigure(goh264.EncoderReconfigure{MaxEncodeTimeUS: 10_000_000}); err != nil {
		t.Fatalf("raise MaxEncodeTimeUS: %v", err)
	}
	fourth, err := enc.Encode(frame)
	if err != nil {
		t.Fatalf("Encode Annex B P-skip after late drop: %v", err)
	}
	if fourth.Dropped || fourth.IDR {
		t.Fatalf("post-late-drop Annex B frame dropped=%v idr=%v, want P-skip", fourth.Dropped, fourth.IDR)
	}
	if fourth.RTPTime != dropped.RTPTime+cfg.RTPTimestampIncrement {
		t.Fatalf("post-late-drop Annex B RTP time = %d, want %d", fourth.RTPTime, dropped.RTPTime+cfg.RTPTimestampIncrement)
	}
	if len(fourth.RTPPackets) != 0 || callbackCalls != firstPacketCount {
		t.Fatalf("post-late-drop Annex B packets/callbacks = %d/%d, want 0/%d",
			len(fourth.RTPPackets), callbackCalls, firstPacketCount)
	}
	assertEncoderNALTypes(t, fourth.NALUnits, []uint8{1})
	decodedFourth, err := annexBDecoder.DecodeFrames(fourth.Data)
	if err != nil {
		t.Fatalf("Decode Annex B P-skip after late drop: %v", err)
	}
	assertDecodedEncoderFrameBytes(t, decodedFourth, appendI420FrameBytes(nil, frame))

	stapa := false
	payloadType := uint8(111)
	ssrc := uint32(0xaabbccdd)
	reentryRTPTime := fourth.RTPTime + cfg.RTPTimestampIncrement
	reentryRTPIncrement := uint32(6000)
	if err := enc.Reconfigure(goh264.EncoderReconfigure{
		OutputFormat:          goh264.EncoderOutputRTP,
		STAPA:                 &stapa,
		RTPPayloadType:        &payloadType,
		RTPSSRC:               &ssrc,
		RTPTimestampIncrement: reentryRTPIncrement,
		ForceIDR:              true,
	}); err != nil {
		t.Fatalf("Reconfigure back to RTP realtime controls: %v", err)
	}
	got = enc.Config()
	if got.OutputFormat != goh264.EncoderOutputRTP ||
		got.STAPA ||
		got.RTPPayloadType != payloadType ||
		got.RTPSSRC != ssrc ||
		got.RTPTimestampIncrement != reentryRTPIncrement {
		t.Fatalf("RTP realtime controls = %+v, want reconfigured state", got)
	}

	fifth, err := enc.Encode(lateChangedFrame)
	if err != nil {
		t.Fatalf("Encode RTP IDR after Annex B sequence: %v", err)
	}
	if !fifth.IDR || fifth.RTPTime != reentryRTPTime {
		t.Fatalf("RTP re-entry idr/time = %v/%d, want IDR/%d",
			fifth.IDR, fifth.RTPTime, reentryRTPTime)
	}
	assertEncoderNALTypes(t, fifth.NALUnits, []uint8{7, 8, 5})
	assertRTPPacketMetadata(t, fifth.RTPPackets, payloadType, ssrc, uint16(firstPacketCount))
	assertRTPPacketTimestamps(t, fifth.RTPPackets, fifth.RTPTime)
	sawFUA = false
	for i, pkt := range fifth.RTPPackets {
		switch pkt.Payload[0] & 0x1f {
		case 24:
			t.Fatalf("RTP re-entry packet[%d] used STAP-A after STAPA=false", i)
		case 28:
			sawFUA = true
		}
	}
	if !sawFUA {
		t.Fatal("RTP re-entry did not fragment the large IDR as FU-A")
	}
	if callbackCalls != firstPacketCount+len(fifth.RTPPackets) {
		t.Fatalf("RTP re-entry callbacks = %d, want %d", callbackCalls, firstPacketCount+len(fifth.RTPPackets))
	}

	sixth, err := enc.Encode(lateChangedFrame)
	if err != nil {
		t.Fatalf("Encode RTP P-skip after re-entry: %v", err)
	}
	if sixth.Dropped || sixth.IDR {
		t.Fatalf("post-re-entry RTP frame dropped=%v idr=%v, want P-skip", sixth.Dropped, sixth.IDR)
	}
	if sixth.RTPTime != fifth.RTPTime+reentryRTPIncrement {
		t.Fatalf("post-re-entry RTP time = %d, want %d", sixth.RTPTime, fifth.RTPTime+reentryRTPIncrement)
	}
	assertEncoderNALTypes(t, sixth.NALUnits, []uint8{1})
	assertRTPPacketMetadata(t, sixth.RTPPackets, payloadType, ssrc, uint16(firstPacketCount+len(fifth.RTPPackets)))
	assertRTPPacketTimestamps(t, sixth.RTPPackets, sixth.RTPTime)
	if callbackCalls != firstPacketCount+len(fifth.RTPPackets)+len(sixth.RTPPackets) {
		t.Fatalf("post-re-entry callbacks = %d, want %d",
			callbackCalls, firstPacketCount+len(fifth.RTPPackets)+len(sixth.RTPPackets))
	}

	rtpDecoder := goh264.NewDecoder()
	decodedFifth, err := rtpDecoder.DecodeFrames(annexBFromEncoderRTPPackets(t, fifth.RTPPackets))
	if err != nil {
		t.Fatalf("Decode RTP re-entry IDR: %v", err)
	}
	assertDecodedEncoderFrameBytes(t, decodedFifth, appendI420FrameBytes(nil, lateChangedFrame))
	decodedSixth, err := rtpDecoder.DecodeFrames(annexBFromEncoderRTPPackets(t, sixth.RTPPackets))
	if err != nil {
		t.Fatalf("Decode RTP re-entry P-skip: %v", err)
	}
	assertDecodedEncoderFrameBytes(t, decodedSixth, appendI420FrameBytes(nil, lateChangedFrame))
}

func TestEncoderReconfigureUpdatesRateControlQPDropAndGOPControls(t *testing.T) {
	cfg := goh264.DefaultEncoderConfig(16, 16)
	cfg.OutputFormat = goh264.EncoderOutputAnnexB
	cfg.DeblockMode = goh264.EncoderDeblockDisabled
	cfg.RTPMaxPayloadSize = 0
	enc, err := goh264.NewEncoder(cfg)
	if err != nil {
		t.Fatalf("NewEncoder: %v", err)
	}
	frame := patternedI420EncoderFrame(16, 16)
	first, err := enc.Encode(frame)
	if err != nil {
		t.Fatalf("Encode first IDR: %v", err)
	}
	assertEncoderNALTypes(t, first.NALUnits, []uint8{7, 8, 5})

	vbv := 222_000
	initialQP := 30
	minQP := 12
	maxQP := 40
	if err := enc.Reconfigure(goh264.EncoderReconfigure{
		RateControl:   goh264.EncoderRateControlVBR,
		VBVBufferSize: &vbv,
		InitialQP:     &initialQP,
		MinQP:         &minQP,
		MaxQP:         &maxQP,
		FrameDrop:     goh264.EncoderFrameDropLate,
		GOPSize:       4,
		IDRInterval:   2,
	}); err != nil {
		t.Fatalf("Reconfigure runtime rate controls: %v", err)
	}
	got := enc.Config()
	if got.RateControl != goh264.EncoderRateControlVBR ||
		got.VBVBufferSize != vbv ||
		got.InitialQP != initialQP ||
		got.MinQP != minQP ||
		got.MaxQP != maxQP ||
		got.FrameDrop != goh264.EncoderFrameDropLate ||
		got.GOPSize != 4 ||
		got.IDRInterval != 2 {
		t.Fatalf("reconfigured runtime controls = %+v", got)
	}
	if !enc.PendingIDR() {
		t.Fatal("QP/PPS reconfigure did not queue an IDR refresh")
	}

	secondFrame := frame
	secondFrame.PTS += int64(cfg.RTPTimestampIncrement)
	second, err := enc.Encode(secondFrame)
	if err != nil {
		t.Fatalf("Encode QP refreshed IDR: %v", err)
	}
	if !second.IDR || enc.PendingIDR() {
		t.Fatalf("QP refreshed frame idr=%v pending=%v, want completed IDR", second.IDR, enc.PendingIDR())
	}
	assertEncoderNALTypes(t, second.NALUnits, []uint8{7, 8, 5})
	assertEncoderVCLQScales(t, second.Data, []uint8{5}, []uint32{30})

	thirdFrame := frame
	thirdFrame.PTS = secondFrame.PTS + int64(cfg.RTPTimestampIncrement)
	third, err := enc.Encode(thirdFrame)
	if err != nil {
		t.Fatalf("Encode post-reconfigure P-skip: %v", err)
	}
	assertEncoderNALTypes(t, third.NALUnits, []uint8{1})

	fourthFrame := frame
	fourthFrame.PTS = thirdFrame.PTS + int64(cfg.RTPTimestampIncrement)
	fourth, err := enc.Encode(fourthFrame)
	if err != nil {
		t.Fatalf("Encode IDR interval refresh: %v", err)
	}
	if !fourth.IDR {
		t.Fatalf("fourth frame idr=%v, want IDR after updated interval", fourth.IDR)
	}
	assertEncoderNALTypes(t, fourth.NALUnits, []uint8{7, 8, 5})
}

func TestEncoderReconfigureAcceptsExplicitZeroQP(t *testing.T) {
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

	initialQP := 0
	minQP := 0
	maxQP := 51
	if err := enc.Reconfigure(goh264.EncoderReconfigure{
		InitialQP: &initialQP,
		MinQP:     &minQP,
		MaxQP:     &maxQP,
	}); err != nil {
		t.Fatalf("Reconfigure explicit zero QP: %v", err)
	}
	if got := enc.Config(); got.InitialQP != 0 || got.MinQP != 0 || got.MaxQP != 51 {
		t.Fatalf("explicit zero QP config = %+v, want initial/min/max 0/0/51", got)
	}

	secondFrame := frame
	secondFrame.PTS += int64(cfg.RTPTimestampIncrement)
	second, err := enc.Encode(secondFrame)
	if err != nil {
		t.Fatalf("Encode zero-QP refreshed IDR: %v", err)
	}
	assertEncoderNALTypes(t, second.NALUnits, []uint8{7, 8, 5})
	assertEncoderVCLQScales(t, second.Data, []uint8{5}, []uint32{0})
}

func TestEncoderReconfigureDeblockModeControlsPFrameAdmission(t *testing.T) {
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
	secondFrame := frame
	secondFrame.PTS += int64(cfg.RTPTimestampIncrement)
	second, err := enc.Encode(secondFrame)
	if err != nil {
		t.Fatalf("Encode admitted P-skip: %v", err)
	}
	assertEncoderNALTypes(t, second.NALUnits, []uint8{1})

	if err := enc.Reconfigure(goh264.EncoderReconfigure{
		DeblockMode: goh264.EncoderDeblockEnabled,
	}); err != nil {
		t.Fatalf("Reconfigure deblock enabled: %v", err)
	}
	thirdFrame := frame
	thirdFrame.PTS = secondFrame.PTS + int64(cfg.RTPTimestampIncrement)
	third, err := enc.Encode(thirdFrame)
	if err != nil {
		t.Fatalf("Encode deblock-enabled P-skip: %v", err)
	}
	if third.IDR {
		t.Fatalf("deblock-enabled frame idr=%v, want admitted P-skip", third.IDR)
	}
	assertEncoderNALTypes(t, third.NALUnits, []uint8{1})

	if err := enc.Reconfigure(goh264.EncoderReconfigure{
		DeblockMode: goh264.EncoderDeblockSliceBoundary,
	}); err != nil {
		t.Fatalf("Reconfigure slice-boundary deblock: %v", err)
	}
	fourthFrame := frame
	fourthFrame.PTS = thirdFrame.PTS + int64(cfg.RTPTimestampIncrement)
	fourth, err := enc.Encode(fourthFrame)
	if err != nil {
		t.Fatalf("Encode slice-boundary P-skip: %v", err)
	}
	if fourth.IDR {
		t.Fatalf("slice-boundary frame idr=%v, want admitted P-skip", fourth.IDR)
	}
	assertEncoderNALTypes(t, fourth.NALUnits, []uint8{1})
}

func TestEncoderReconfigureRejectsInvalidRuntimeRateControlsWithoutMutation(t *testing.T) {
	enc, err := goh264.NewEncoder(goh264.DefaultEncoderConfig(16, 16))
	if err != nil {
		t.Fatalf("NewEncoder: %v", err)
	}
	before := enc.Config()

	badVBV := -1
	badInitialQP := 41
	minQP := 12
	maxQP := 40
	tests := []struct {
		name   string
		update goh264.EncoderReconfigure
	}{
		{name: "bad rate-control mode", update: goh264.EncoderReconfigure{RateControl: goh264.EncoderRateControlMode(99), ForceIDR: true}},
		{name: "bad vbv", update: goh264.EncoderReconfigure{VBVBufferSize: &badVBV, ForceIDR: true}},
		{name: "bad qp range", update: goh264.EncoderReconfigure{InitialQP: &badInitialQP, MinQP: &minQP, MaxQP: &maxQP, ForceIDR: true}},
		{name: "bad frame-drop mode", update: goh264.EncoderReconfigure{FrameDrop: goh264.EncoderFrameDropMode(99), ForceIDR: true}},
		{name: "partial width resolution", update: goh264.EncoderReconfigure{Width: 32, ForceIDR: true}},
		{name: "partial height resolution", update: goh264.EncoderReconfigure{Height: 32, ForceIDR: true}},
		{name: "negative width resolution", update: goh264.EncoderReconfigure{Width: -16, Height: 16, ForceIDR: true}},
		{name: "negative height resolution", update: goh264.EncoderReconfigure{Width: 16, Height: -16, ForceIDR: true}},
		{name: "bad gop interval", update: goh264.EncoderReconfigure{GOPSize: 2, IDRInterval: 3, ForceIDR: true}},
		{name: "negative gop size", update: goh264.EncoderReconfigure{GOPSize: -1, ForceIDR: true}},
		{name: "negative idr interval", update: goh264.EncoderReconfigure{IDRInterval: -1, ForceIDR: true}},
		{name: "bad deblock mode", update: goh264.EncoderReconfigure{DeblockMode: goh264.EncoderDeblockMode(99), ForceIDR: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := enc.Reconfigure(tt.update); !errors.Is(err, goh264.ErrInvalidData) {
				t.Fatalf("Reconfigure invalid runtime controls error = %v, want ErrInvalidData", err)
			}
			if got := enc.Config(); got != before {
				t.Fatalf("invalid runtime controls mutated config = %+v, want %+v", got, before)
			}
			if enc.PendingIDR() {
				t.Fatal("invalid runtime controls queued an IDR")
			}
		})
	}
}

func TestEncoderReconfigureRejectsInvalidOutputControlsWithoutMutation(t *testing.T) {
	cfg := goh264.DefaultEncoderConfig(16, 16)
	cfg.OutputFormat = goh264.EncoderOutputAVC
	cfg.SPSPPSMode = goh264.EncoderSPSPPSOutOfBand
	cfg.RTPMaxPayloadSize = 0
	cfg.DeblockMode = goh264.EncoderDeblockDisabled
	enc, err := goh264.NewEncoder(cfg)
	if err != nil {
		t.Fatalf("NewEncoder: %v", err)
	}
	if _, err := enc.Encode(patternedI420EncoderFrame(16, 16)); err != nil {
		t.Fatalf("Encode first AVC frame: %v", err)
	}
	if enc.PendingIDR() {
		t.Fatal("freshly encoded AVC frame left unexpected pending IDR")
	}

	before := enc.Config()
	tests := []struct {
		name   string
		update goh264.EncoderReconfigure
	}{
		{name: "bad sps pps mode", update: goh264.EncoderReconfigure{
			SPSPPSMode: goh264.EncoderSPSPPSMode(99),
			ForceIDR:   true,
		}},
		{name: "bad preset", update: goh264.EncoderReconfigure{
			Preset:   goh264.EncoderPreset(99),
			ForceIDR: true,
		}},
		{name: "bad output format", update: goh264.EncoderReconfigure{
			OutputFormat: goh264.EncoderOutputFormat(99),
			ForceIDR:     true,
		}},
		{name: "bad rtp re-entry payload size", update: goh264.EncoderReconfigure{
			OutputFormat:      goh264.EncoderOutputRTP,
			RTPMaxPayloadSize: 2,
			ForceIDR:          true,
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := enc.Reconfigure(tt.update); !errors.Is(err, goh264.ErrInvalidData) {
				t.Fatalf("Reconfigure invalid output controls error = %v, want ErrInvalidData", err)
			}
			if got := enc.Config(); got != before {
				t.Fatalf("invalid output controls mutated config = %+v, want %+v", got, before)
			}
			if enc.PendingIDR() {
				t.Fatal("invalid output controls queued an IDR")
			}
		})
	}
}

func TestEncoderReconfigureRejectsInvalidWebRTCPacketizationUpdateWithoutMutation(t *testing.T) {
	cfg := goh264.DefaultEncoderConfig(16, 16)
	enc, err := goh264.NewEncoder(cfg)
	if err != nil {
		t.Fatalf("NewEncoder: %v", err)
	}

	before := enc.Config()
	mode0 := goh264.EncoderRTPPacketizationSingleNAL
	stapa := true
	if err := enc.Reconfigure(goh264.EncoderReconfigure{
		RTPPacketizationMode: &mode0,
		STAPA:                &stapa,
		ForceIDR:             true,
	}); !errors.Is(err, goh264.ErrUnsupported) {
		t.Fatalf("Reconfigure mode-0 STAP-A error = %v, want ErrUnsupported", err)
	}
	if got := enc.Config(); got != before {
		t.Fatalf("invalid packetization reconfigure mutated config = %+v, want %+v", got, before)
	}
	if enc.PendingIDR() {
		t.Fatal("invalid packetization reconfigure queued an IDR")
	}

	badPayloadType := uint8(128)
	if err := enc.Reconfigure(goh264.EncoderReconfigure{
		RTPPayloadType: &badPayloadType,
		ForceIDR:       true,
	}); !errors.Is(err, goh264.ErrInvalidData) {
		t.Fatalf("Reconfigure bad payload type error = %v, want ErrInvalidData", err)
	}
	if got := enc.Config(); got != before {
		t.Fatalf("invalid payload type reconfigure mutated config = %+v, want %+v", got, before)
	}
	if enc.PendingIDR() {
		t.Fatal("invalid payload type reconfigure queued an IDR")
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

func TestEncoderReconfigureSPSPPSCadenceControlsLiveIDRHeaders(t *testing.T) {
	cfg := goh264.DefaultEncoderConfig(16, 16)
	cfg.OutputFormat = goh264.EncoderOutputAnnexB
	cfg.DeblockMode = goh264.EncoderDeblockDisabled
	cfg.RTPMaxPayloadSize = 0
	enc, err := goh264.NewEncoder(cfg)
	if err != nil {
		t.Fatalf("NewEncoder: %v", err)
	}

	frame := patternedI420EncoderFrame(16, 16)
	dec := goh264.NewDecoder()
	var stream []byte
	var wantStream []byte

	encodeForcedIDR := func(label string, wantNALs []uint8) goh264.EncodedFrame {
		t.Helper()
		enc.ForceIDR()
		if !enc.PendingIDR() {
			t.Fatalf("%s ForceIDR did not queue IDR", label)
		}
		out, err := enc.Encode(frame)
		if err != nil {
			t.Fatalf("%s Encode: %v", label, err)
		}
		if !out.IDR || enc.PendingIDR() {
			t.Fatalf("%s output idr=%v pending=%v, want completed IDR", label, out.IDR, enc.PendingIDR())
		}
		assertEncoderNALTypes(t, out.NALUnits, wantNALs)
		decoded, err := dec.DecodeFrames(out.Data)
		if err != nil {
			t.Fatalf("%s DecodeFrames: %v", label, err)
		}
		assertDecodedEncoderFrameBytes(t, decoded, appendI420FrameBytes(nil, frame))
		stream = append(stream, out.Data...)
		wantStream = appendI420FrameBytes(wantStream, frame)
		frame.PTS += int64(cfg.RTPTimestampIncrement)
		return out
	}

	// The first frame is an implicit IDR and seeds the decoder with parameter sets.
	first, err := enc.Encode(frame)
	if err != nil {
		t.Fatalf("Encode initial IDR: %v", err)
	}
	if !first.IDR || enc.PendingIDR() {
		t.Fatalf("initial output idr=%v pending=%v, want completed IDR", first.IDR, enc.PendingIDR())
	}
	assertEncoderNALTypes(t, first.NALUnits, []uint8{7, 8, 5})
	decodedFirst, err := dec.DecodeFrames(first.Data)
	if err != nil {
		t.Fatalf("Decode initial IDR: %v", err)
	}
	assertDecodedEncoderFrameBytes(t, decodedFirst, appendI420FrameBytes(nil, frame))
	stream = append(stream, first.Data...)
	wantStream = appendI420FrameBytes(wantStream, frame)
	frame.PTS += int64(cfg.RTPTimestampIncrement)

	if err := enc.Reconfigure(goh264.EncoderReconfigure{
		SPSPPSMode: goh264.EncoderSPSPPSOutOfBand,
	}); err != nil {
		t.Fatalf("Reconfigure out-of-band SPS/PPS: %v", err)
	}
	if enc.PendingIDR() {
		t.Fatal("out-of-band SPS/PPS reconfigure queued unexpected IDR")
	}
	outOfBand := encodeForcedIDR("out-of-band", []uint8{5})

	noBeforeIDR := false
	if err := enc.Reconfigure(goh264.EncoderReconfigure{
		SPSPPSMode:      goh264.EncoderSPSPPSEveryIDR,
		SPSPPSBeforeIDR: &noBeforeIDR,
	}); err != nil {
		t.Fatalf("Reconfigure every-IDR SPS/PPS: %v", err)
	}
	if enc.PendingIDR() {
		t.Fatal("every-IDR SPS/PPS reconfigure queued unexpected IDR")
	}
	everyIDR := encodeForcedIDR("every-IDR", []uint8{7, 8, 5})

	if err := enc.Reconfigure(goh264.EncoderReconfigure{
		SPSPPSMode:      goh264.EncoderSPSPPSInBandKeyframes,
		SPSPPSBeforeIDR: &noBeforeIDR,
	}); err != nil {
		t.Fatalf("Reconfigure in-band suppressed SPS/PPS: %v", err)
	}
	if enc.PendingIDR() {
		t.Fatal("in-band suppressed SPS/PPS reconfigure queued unexpected IDR")
	}
	suppressed := encodeForcedIDR("in-band suppressed", []uint8{5})

	beforeIDR := true
	if err := enc.Reconfigure(goh264.EncoderReconfigure{
		SPSPPSBeforeIDR: &beforeIDR,
	}); err != nil {
		t.Fatalf("Reconfigure in-band restored SPS/PPS: %v", err)
	}
	if got := enc.Config(); got.SPSPPSMode != goh264.EncoderSPSPPSInBandKeyframes || !got.SPSPPSBeforeIDR {
		t.Fatalf("restored SPS/PPS config = %+v, want in-band before IDR", got)
	}
	if enc.PendingIDR() {
		t.Fatal("in-band restored SPS/PPS reconfigure queued unexpected IDR")
	}
	restored := encodeForcedIDR("in-band restored", []uint8{7, 8, 5})

	assertEncoderVCLFrameNums(t, stream,
		[]uint8{5, 5, 5, 5, 5},
		[]uint32{0, 1, 2, 3, 4},
	)
	assertFFmpegRawVideoOracle(t, stream, wantStream)
	if outOfBand.RTPTime != first.RTPTime+cfg.RTPTimestampIncrement ||
		everyIDR.RTPTime != outOfBand.RTPTime+cfg.RTPTimestampIncrement ||
		suppressed.RTPTime != everyIDR.RTPTime+cfg.RTPTimestampIncrement ||
		restored.RTPTime != suppressed.RTPTime+cfg.RTPTimestampIncrement {
		t.Fatalf("forced IDR RTP times = %d/%d/%d/%d/%d, want cadence increment %d",
			first.RTPTime, outOfBand.RTPTime, everyIDR.RTPTime, suppressed.RTPTime, restored.RTPTime,
			cfg.RTPTimestampIncrement)
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

func TestEncoderParameterSetsReturnCallerOwnedSurfaces(t *testing.T) {
	cfg := goh264.DefaultEncoderConfig(638, 478)
	cfg.FrameRateNum = 30000
	cfg.FrameRateDen = 1001
	enc, err := goh264.NewEncoder(cfg)
	if err != nil {
		t.Fatalf("NewEncoder: %v", err)
	}

	headers, err := enc.ParameterSets()
	if err != nil {
		t.Fatalf("ParameterSets: %v", err)
	}
	originalAnnexB := append([]byte(nil), headers.AnnexB...)
	originalAVCC := append([]byte(nil), headers.AVCDecoderConfigurationRecord...)
	originalSPS := append([]byte(nil), headers.SPS...)
	originalPPS := append([]byte(nil), headers.PPS...)

	headers.SPS[0] ^= 0x1f
	headers.PPS[0] ^= 0x1f
	headers.SPS = append(headers.SPS, 0xaa)
	headers.PPS = append(headers.PPS, 0xbb)
	if !bytes.Equal(headers.AnnexB, originalAnnexB) ||
		!bytes.Equal(headers.AVCDecoderConfigurationRecord, originalAVCC) {
		t.Fatalf("raw parameter-set mutation aliased packaged headers:\nAnnexB %x want %x\navcC %x want %x",
			headers.AnnexB, originalAnnexB,
			headers.AVCDecoderConfigurationRecord, originalAVCC)
	}
	headers.AnnexB[0] ^= 0xff
	headers.AVCDecoderConfigurationRecord[0] ^= 0xff
	headers.AnnexB = append(headers.AnnexB, 0xcc)
	headers.AVCDecoderConfigurationRecord = append(headers.AVCDecoderConfigurationRecord, 0xdd)

	again, err := enc.ParameterSets()
	if err != nil {
		t.Fatalf("ParameterSets after caller mutation: %v", err)
	}
	if !bytes.Equal(again.SPS, originalSPS) ||
		!bytes.Equal(again.PPS, originalPPS) ||
		!bytes.Equal(again.AnnexB, originalAnnexB) ||
		!bytes.Equal(again.AVCDecoderConfigurationRecord, originalAVCC) {
		t.Fatalf("ParameterSets aliases caller mutation:\nSPS %x want %x\nPPS %x want %x\nAnnexB %x want %x\navcC %x want %x",
			again.SPS, originalSPS,
			again.PPS, originalPPS,
			again.AnnexB, originalAnnexB,
			again.AVCDecoderConfigurationRecord, originalAVCC)
	}
	if !bytes.Contains(again.AnnexB, again.SPS) || !bytes.Contains(again.AnnexB, again.PPS) {
		t.Fatalf("regenerated Annex B headers do not contain SPS/PPS: %x", again.AnnexB)
	}
	if _, err := goh264.NewDecoder().ParseHeadersAnnexB(again.AnnexB); err != nil {
		t.Fatalf("ParseHeadersAnnexB regenerated headers: %v", err)
	}
	if _, err := goh264.NewDecoder().ParseAVCDecoderConfigurationRecord(again.AVCDecoderConfigurationRecord); err != nil {
		t.Fatalf("ParseAVCDecoderConfigurationRecord regenerated headers: %v", err)
	}
}

func TestEncoderHeaderHelpersPreservePendingIDR(t *testing.T) {
	enc, err := goh264.NewEncoder(goh264.DefaultEncoderConfig(16, 16))
	if err != nil {
		t.Fatalf("NewEncoder: %v", err)
	}
	first, err := enc.Encode(patternedI420EncoderFrame(16, 16))
	if err != nil {
		t.Fatalf("Encode first IDR: %v", err)
	}
	if !first.IDR || enc.PendingIDR() {
		t.Fatalf("first frame idr=%v pending=%v, want completed IDR", first.IDR, enc.PendingIDR())
	}

	enc.ForceIDR()
	if !enc.PendingIDR() {
		t.Fatal("ForceIDR before header helpers did not queue IDR")
	}
	before := enc.Config()
	for i := 0; i < 3; i++ {
		headers, err := enc.ParameterSets()
		if err != nil {
			t.Fatalf("ParameterSets[%d]: %v", i, err)
		}
		if len(headers.SPS) == 0 || len(headers.PPS) == 0 ||
			len(headers.AnnexB) == 0 || len(headers.AVCDecoderConfigurationRecord) == 0 {
			t.Fatalf("ParameterSets[%d] returned empty surfaces: %+v", i, headers)
		}
		sei, err := enc.RecoveryPointSEI(uint32(i))
		if err != nil {
			t.Fatalf("RecoveryPointSEI[%d]: %v", i, err)
		}
		if len(sei.NAL) == 0 || len(sei.AnnexB) == 0 || len(sei.AVC) == 0 {
			t.Fatalf("RecoveryPointSEI[%d] returned empty surfaces: %+v", i, sei)
		}
		if got := enc.Config(); got != before {
			t.Fatalf("header helper[%d] mutated config = %+v, want %+v", i, got, before)
		}
		if !enc.PendingIDR() {
			t.Fatalf("header helper[%d] cleared pending IDR", i)
		}
	}

	secondFrame := patternedI420EncoderFrame(16, 16)
	secondFrame.Y = append([]byte(nil), secondFrame.Y...)
	secondFrame.Y[0] ^= 0x44
	second, err := enc.Encode(secondFrame)
	if err != nil {
		t.Fatalf("Encode after header helpers: %v", err)
	}
	if !second.IDR || enc.PendingIDR() {
		t.Fatalf("post-helper frame idr=%v pending=%v, want delivered IDR",
			second.IDR, enc.PendingIDR())
	}
	assertEncoderNALTypes(t, second.NALUnits, []uint8{7, 8, 5})
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
	originalNAL := append([]byte(nil), sei.NAL...)
	originalAnnexB := append([]byte(nil), sei.AnnexB...)
	originalAVC := append([]byte(nil), sei.AVC...)
	sei.NAL[0] = 0
	sei.NAL = append(sei.NAL, 0xaa)
	if !bytes.Equal(sei.AnnexB, originalAnnexB) || !bytes.Equal(sei.AVC, originalAVC) {
		t.Fatalf("raw SEI mutation aliased packaged surfaces:\nAnnexB %x want %x\nAVC %x want %x",
			sei.AnnexB, originalAnnexB,
			sei.AVC, originalAVC)
	}
	sei.AnnexB[0] ^= 0xff
	sei.AVC[0] ^= 0xff
	sei.AnnexB = append(sei.AnnexB, 0xbb)
	sei.AVC = append(sei.AVC, 0xcc)
	again, err := enc.RecoveryPointSEI(0)
	if err != nil {
		t.Fatalf("RecoveryPointSEI after caller mutation: %v", err)
	}
	if !bytes.Equal(again.NAL, originalNAL) ||
		!bytes.Equal(again.AnnexB, originalAnnexB) ||
		!bytes.Equal(again.AVC, originalAVC) {
		t.Fatalf("SEI aliases caller mutation:\nNAL %x want %x\nAnnexB %x want %x\nAVC %x want %x",
			again.NAL, originalNAL,
			again.AnnexB, originalAnnexB,
			again.AVC, originalAVC)
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
	first, err := enc.Encode(patternedI420EncoderFrame(16, 16))
	if err != nil {
		t.Fatalf("Encode first IDR: %v", err)
	}
	if !first.IDR || enc.PendingIDR() {
		t.Fatalf("first frame idr=%v pending=%v, want completed IDR", first.IDR, enc.PendingIDR())
	}

	enc.ForceIDR()
	if !enc.PendingIDR() {
		t.Fatal("ForceIDR before invalid RecoveryPointSEI did not queue IDR")
	}
	before := enc.Config()
	if _, err := enc.RecoveryPointSEI(1 << 16); !errors.Is(err, goh264.ErrInvalidData) {
		t.Fatalf("RecoveryPointSEI invalid error = %v, want ErrInvalidData", err)
	}
	if got := enc.Config(); got != before {
		t.Fatalf("invalid RecoveryPointSEI mutated config = %+v, want %+v", got, before)
	}
	if !enc.PendingIDR() {
		t.Fatal("invalid RecoveryPointSEI cleared pending IDR")
	}

	secondFrame := patternedI420EncoderFrame(16, 16)
	secondFrame.Y[0] ^= 0x33
	second, err := enc.Encode(secondFrame)
	if err != nil {
		t.Fatalf("Encode after invalid RecoveryPointSEI: %v", err)
	}
	if !second.IDR || enc.PendingIDR() {
		t.Fatalf("post-invalid-RecoveryPointSEI frame idr=%v pending=%v, want delivered IDR",
			second.IDR, enc.PendingIDR())
	}
	assertEncoderNALTypes(t, second.NALUnits, []uint8{7, 8, 5})
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

func TestEncoderEncodeExactP16x16NoResidualMotion(t *testing.T) {
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

	secondFrame := integerMotionI420EncoderFrame(firstFrame, 2, 0)
	secondFrame.PTS = firstFrame.PTS + int64(cfg.RTPTimestampIncrement)
	second, err := enc.Encode(secondFrame)
	if err != nil {
		t.Fatalf("Encode exact P16x16 no-residual motion: %v", err)
	}
	if second.KeyFrame || second.IDR {
		t.Fatalf("exact-motion second frame key=%v idr=%v, want non-IDR P16x16", second.KeyFrame, second.IDR)
	}
	assertEncoderNALTypes(t, second.NALUnits, []uint8{1})

	dec := goh264.NewDecoder()
	decodedFirst, err := dec.DecodeFrames(first.Data)
	if err != nil {
		t.Fatalf("Decode first IDR: %v", err)
	}
	assertDecodedEncoderFrameBytes(t, decodedFirst, appendI420FrameBytes(nil, firstFrame))
	decodedSecond, err := dec.DecodeFrames(second.Data)
	if err != nil {
		t.Fatalf("Decode exact P16x16 no-residual motion: %v", err)
	}
	assertDecodedEncoderFrameBytes(t, decodedSecond, appendI420FrameBytes(nil, secondFrame))
	if decodedSecond[0].KeyFrame || decodedSecond[0].SideData.RecoveryPoint != nil {
		t.Fatalf("decoded exact-motion P frame key=%v recovery=%+v, want predictive non-recovery frame",
			decodedSecond[0].KeyFrame, decodedSecond[0].SideData.RecoveryPoint)
	}

	stream := append(append([]byte(nil), first.Data...), second.Data...)
	wantStream := appendI420FrameBytes(nil, firstFrame)
	wantStream = appendI420FrameBytes(wantStream, secondFrame)
	assertFFmpegRawVideoOracle(t, stream, wantStream)
}

func TestEncoderEncodeExactP16x16NoResidualMotionForAVCAndRTP(t *testing.T) {
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
			assertEncoderNALTypes(t, first.NALUnits, []uint8{7, 8, 5})

			secondFrame := integerMotionI420EncoderFrame(firstFrame, 2, 0)
			secondFrame.PTS = firstFrame.PTS + int64(cfg.RTPTimestampIncrement)
			second, err := enc.Encode(secondFrame)
			if err != nil {
				t.Fatalf("Encode exact P16x16 no-residual motion: %v", err)
			}
			assertEncoderNALTypes(t, second.NALUnits, []uint8{1})

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
				decodedFirst, err = dec.DecodeFrames(annexBFromEncoderRTPPackets(t, first.RTPPackets))
				if err != nil {
					t.Fatalf("DecodeFrames first RTP: %v", err)
				}
				decodedSecond, err = dec.DecodeFrames(annexBFromEncoderRTPPackets(t, second.RTPPackets))
				if err != nil {
					t.Fatalf("DecodeFrames second RTP: %v", err)
				}
			default:
				t.Fatalf("unexpected format %v", tt.format)
			}
			assertDecodedEncoderFrameBytes(t, decodedFirst, appendI420FrameBytes(nil, firstFrame))
			assertDecodedEncoderFrameBytes(t, decodedSecond, appendI420FrameBytes(nil, secondFrame))
			if decodedSecond[0].KeyFrame || decodedSecond[0].SideData.RecoveryPoint != nil {
				t.Fatalf("decoded exact-motion P frame key=%v recovery=%+v, want predictive non-recovery frame",
					decodedSecond[0].KeyFrame, decodedSecond[0].SideData.RecoveryPoint)
			}
		})
	}
}

func TestEncoderEncodeExactP16x16NoResidualMotionWithDeblockControls(t *testing.T) {
	for _, tt := range []struct {
		name        string
		deblock     goh264.EncoderDeblockMode
		wantDeblock int32
	}{
		{name: "enabled", deblock: goh264.EncoderDeblockEnabled, wantDeblock: 1},
		{name: "slice-boundary", deblock: goh264.EncoderDeblockSliceBoundary, wantDeblock: 2},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cfg := goh264.DefaultEncoderConfig(16, 16)
			cfg.OutputFormat = goh264.EncoderOutputAnnexB
			cfg.DeblockMode = tt.deblock
			cfg.RTPMaxPayloadSize = 0
			enc, err := goh264.NewEncoder(cfg)
			if err != nil {
				t.Fatalf("NewEncoder: %v", err)
			}
			headers, err := enc.ParameterSets()
			if err != nil {
				t.Fatalf("ParameterSets: %v", err)
			}
			firstFrame := patternedI420EncoderFrame(16, 16)
			first, err := enc.Encode(firstFrame)
			if err != nil {
				t.Fatalf("Encode first IDR: %v", err)
			}
			assertEncoderNALTypes(t, first.NALUnits, []uint8{7, 8, 5})

			secondFrame := integerMotionI420EncoderFrame(firstFrame, 2, 0)
			secondFrame.PTS = firstFrame.PTS + int64(cfg.RTPTimestampIncrement)
			second, err := enc.Encode(secondFrame)
			if err != nil {
				t.Fatalf("Encode exact P16x16 no-residual motion with deblock %s: %v", tt.name, err)
			}
			if second.KeyFrame || second.IDR {
				t.Fatalf("exact-motion deblock %s second frame key=%v idr=%v, want non-IDR P16x16",
					tt.name, second.KeyFrame, second.IDR)
			}
			assertEncoderNALTypes(t, second.NALUnits, []uint8{1})
			assertEncoderVCLDeblocks(t, append(append([]byte(nil), headers.AnnexB...), second.Data...), []uint8{1}, []int32{tt.wantDeblock})

			dec := goh264.NewDecoder()
			decodedFirst, err := dec.DecodeFrames(first.Data)
			if err != nil {
				t.Fatalf("Decode first IDR: %v", err)
			}
			assertDecodedEncoderFrameBytes(t, decodedFirst, appendI420FrameBytes(nil, firstFrame))
			decodedSecond, err := dec.DecodeFrames(second.Data)
			if err != nil {
				t.Fatalf("Decode exact P16x16 no-residual motion with deblock %s: %v", tt.name, err)
			}
			assertDecodedEncoderFrameBytes(t, decodedSecond, appendI420FrameBytes(nil, secondFrame))
			if decodedSecond[0].KeyFrame || decodedSecond[0].SideData.RecoveryPoint != nil {
				t.Fatalf("decoded exact-motion deblock %s P frame key=%v recovery=%+v, want predictive non-recovery frame",
					tt.name, decodedSecond[0].KeyFrame, decodedSecond[0].SideData.RecoveryPoint)
			}

			stream := append(append([]byte(nil), first.Data...), second.Data...)
			wantStream := appendI420FrameBytes(nil, firstFrame)
			wantStream = appendI420FrameBytes(wantStream, secondFrame)
			assertFFmpegRawVideoOracle(t, stream, wantStream)
		})
	}
}

func TestEncoderEncodeExactP16x16NoResidualMotionWithDeblockControlsForAVCAndRTP(t *testing.T) {
	for _, tt := range []struct {
		name   string
		format goh264.EncoderOutputFormat
	}{
		{name: "avc", format: goh264.EncoderOutputAVC},
		{name: "rtp", format: goh264.EncoderOutputRTP},
	} {
		for _, deblock := range []struct {
			name string
			mode goh264.EncoderDeblockMode
		}{
			{name: "enabled", mode: goh264.EncoderDeblockEnabled},
			{name: "slice-boundary", mode: goh264.EncoderDeblockSliceBoundary},
		} {
			t.Run(tt.name+"/"+deblock.name, func(t *testing.T) {
				cfg := goh264.DefaultEncoderConfig(16, 16)
				cfg.OutputFormat = tt.format
				cfg.DeblockMode = deblock.mode
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
				assertEncoderNALTypes(t, first.NALUnits, []uint8{7, 8, 5})

				secondFrame := integerMotionI420EncoderFrame(firstFrame, 2, 0)
				secondFrame.PTS = firstFrame.PTS + int64(cfg.RTPTimestampIncrement)
				second, err := enc.Encode(secondFrame)
				if err != nil {
					t.Fatalf("Encode exact P16x16 no-residual motion %s/%s: %v", tt.name, deblock.name, err)
				}
				assertEncoderNALTypes(t, second.NALUnits, []uint8{1})

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
						t.Fatalf("DecodeConfiguredAVCFrames first %s: %v", deblock.name, err)
					}
					decodedSecond, err = dec.DecodeConfiguredAVCFrames(second.Data)
					if err != nil {
						t.Fatalf("DecodeConfiguredAVCFrames second %s: %v", deblock.name, err)
					}
				case goh264.EncoderOutputRTP:
					decodedFirst, err = dec.DecodeFrames(annexBFromEncoderRTPPackets(t, first.RTPPackets))
					if err != nil {
						t.Fatalf("DecodeFrames first RTP %s: %v", deblock.name, err)
					}
					decodedSecond, err = dec.DecodeFrames(annexBFromEncoderRTPPackets(t, second.RTPPackets))
					if err != nil {
						t.Fatalf("DecodeFrames second RTP %s: %v", deblock.name, err)
					}
				default:
					t.Fatalf("unexpected format %v", tt.format)
				}
				assertDecodedEncoderFrameBytes(t, decodedFirst, appendI420FrameBytes(nil, firstFrame))
				assertDecodedEncoderFrameBytes(t, decodedSecond, appendI420FrameBytes(nil, secondFrame))
				if decodedSecond[0].KeyFrame || decodedSecond[0].SideData.RecoveryPoint != nil {
					t.Fatalf("decoded exact-motion %s/%s P frame key=%v recovery=%+v, want predictive non-recovery frame",
						tt.name, deblock.name, decodedSecond[0].KeyFrame, decodedSecond[0].SideData.RecoveryPoint)
				}
			})
		}
	}
}

func TestEncoderEncodeMacroblockAlignedExactP16x16NoResidualMotion(t *testing.T) {
	for _, tt := range []struct {
		name            string
		width           int
		height          int
		dx              int
		dy              int
		sliceCount      int
		wantFirstNALs   []uint8
		wantSecondNALs  []uint8
		wantSecondFirst []uint32
	}{
		{
			name:            "single-row-single-slice",
			width:           32,
			height:          16,
			dx:              2,
			sliceCount:      1,
			wantFirstNALs:   []uint8{7, 8, 5},
			wantSecondNALs:  []uint8{1},
			wantSecondFirst: []uint32{0},
		},
		{
			name:            "single-row-two-slices",
			width:           32,
			height:          16,
			dx:              2,
			sliceCount:      2,
			wantFirstNALs:   []uint8{7, 8, 5, 5},
			wantSecondNALs:  []uint8{1, 1},
			wantSecondFirst: []uint32{0, 1},
		},
		{
			name:            "narrow-two-row-single-slice",
			width:           16,
			height:          32,
			dx:              2,
			sliceCount:      1,
			wantFirstNALs:   []uint8{7, 8, 5},
			wantSecondNALs:  []uint8{1},
			wantSecondFirst: []uint32{0},
		},
		{
			name:            "ragged-four-slice-two-row",
			width:           48,
			height:          32,
			dx:              2,
			sliceCount:      4,
			wantFirstNALs:   []uint8{7, 8, 5, 5, 5, 5},
			wantSecondNALs:  []uint8{1, 1, 1, 1},
			wantSecondFirst: []uint32{0, 2, 4, 5},
		},
		{
			name:            "diagonal-two-row-single-slice",
			width:           32,
			height:          32,
			dx:              2,
			dy:              2,
			sliceCount:      1,
			wantFirstNALs:   []uint8{7, 8, 5},
			wantSecondNALs:  []uint8{1},
			wantSecondFirst: []uint32{0},
		},
		{
			name:            "larger-horizontal-single-row",
			width:           32,
			height:          16,
			dx:              4,
			sliceCount:      1,
			wantFirstNALs:   []uint8{7, 8, 5},
			wantSecondNALs:  []uint8{1},
			wantSecondFirst: []uint32{0},
		},
		{
			name:            "edge-horizontal-single-row",
			width:           48,
			height:          16,
			dx:              8,
			sliceCount:      1,
			wantFirstNALs:   []uint8{7, 8, 5},
			wantSecondNALs:  []uint8{1},
			wantSecondFirst: []uint32{0},
		},
		{
			name:            "edge-vertical-two-row",
			width:           16,
			height:          48,
			dy:              8,
			sliceCount:      1,
			wantFirstNALs:   []uint8{7, 8, 5},
			wantSecondNALs:  []uint8{1},
			wantSecondFirst: []uint32{0},
		},
		{
			name:            "edge-diagonal-two-row",
			width:           48,
			height:          48,
			dx:              8,
			dy:              -8,
			sliceCount:      3,
			wantFirstNALs:   []uint8{7, 8, 5, 5, 5},
			wantSecondNALs:  []uint8{1, 1, 1},
			wantSecondFirst: []uint32{0, 3, 6},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cfg := goh264.DefaultEncoderConfig(tt.width, tt.height)
			cfg.OutputFormat = goh264.EncoderOutputAnnexB
			cfg.DeblockMode = goh264.EncoderDeblockDisabled
			cfg.RTPMaxPayloadSize = 0
			cfg.SliceCount = tt.sliceCount
			enc, err := goh264.NewEncoder(cfg)
			if err != nil {
				t.Fatalf("NewEncoder: %v", err)
			}
			headers, err := enc.ParameterSets()
			if err != nil {
				t.Fatalf("ParameterSets: %v", err)
			}
			firstFrame := patternedI420EncoderFrame(tt.width, tt.height)
			first, err := enc.Encode(firstFrame)
			if err != nil {
				t.Fatalf("Encode first IDR: %v", err)
			}
			assertEncoderNALTypes(t, first.NALUnits, tt.wantFirstNALs)

			secondFrame := integerMotionI420EncoderFrame(firstFrame, tt.dx, tt.dy)
			secondFrame.PTS = firstFrame.PTS + int64(cfg.RTPTimestampIncrement)
			second, err := enc.Encode(secondFrame)
			if err != nil {
				t.Fatalf("Encode wide exact P16x16 no-residual motion: %v", err)
			}
			assertEncoderNALTypes(t, second.NALUnits, tt.wantSecondNALs)
			assertEncoderVCLFirstMBs(t, append(append([]byte(nil), headers.AnnexB...), second.Data...), tt.wantSecondNALs, tt.wantSecondFirst)

			dec := goh264.NewDecoder()
			decodedFirst, err := dec.DecodeFrames(first.Data)
			if err != nil {
				t.Fatalf("Decode first IDR: %v", err)
			}
			assertDecodedEncoderFrameBytes(t, decodedFirst, appendI420FrameBytes(nil, firstFrame))
			decodedSecond, err := dec.DecodeFrames(second.Data)
			if err != nil {
				t.Fatalf("Decode wide exact P16x16 no-residual motion: %v", err)
			}
			assertDecodedEncoderFrameBytes(t, decodedSecond, appendI420FrameBytes(nil, secondFrame))

			stream := append(append([]byte(nil), first.Data...), second.Data...)
			wantStream := appendI420FrameBytes(nil, firstFrame)
			wantStream = appendI420FrameBytes(wantStream, secondFrame)
			assertFFmpegRawVideoOracle(t, stream, wantStream)
		})
	}
}

func TestEncoderEncodePerMacroblockExactP16x16NoResidualMotionForAnnexBAVCRTP(t *testing.T) {
	motions := []encoderTestMotion{
		{dx: 2, dy: 0},
		{dx: -2, dy: 0},
		{dx: 0, dy: 2},
		{dx: 0, dy: -2},
	}
	for _, tt := range []struct {
		name   string
		format goh264.EncoderOutputFormat
	}{
		{name: "annexb", format: goh264.EncoderOutputAnnexB},
		{name: "avc", format: goh264.EncoderOutputAVC},
		{name: "rtp", format: goh264.EncoderOutputRTP},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cfg := goh264.DefaultEncoderConfig(32, 32)
			cfg.OutputFormat = tt.format
			cfg.DeblockMode = goh264.EncoderDeblockDisabled
			cfg.SliceCount = 2
			if tt.format != goh264.EncoderOutputRTP {
				cfg.RTPMaxPayloadSize = 0
			}
			enc, err := goh264.NewEncoder(cfg)
			if err != nil {
				t.Fatalf("NewEncoder: %v", err)
			}
			firstFrame := patternedI420EncoderFrame(32, 32)
			first, err := enc.Encode(firstFrame)
			if err != nil {
				t.Fatalf("Encode first IDR: %v", err)
			}
			assertEncoderNALTypes(t, first.NALUnits, []uint8{7, 8, 5, 5})
			headers, err := enc.ParameterSets()
			if err != nil {
				t.Fatalf("ParameterSets: %v", err)
			}

			secondFrame := perMacroblockMotionI420EncoderFrame(firstFrame, motions)
			secondFrame.PTS = firstFrame.PTS + int64(cfg.RTPTimestampIncrement)
			second, err := enc.Encode(secondFrame)
			if err != nil {
				t.Fatalf("Encode per-macroblock exact P16x16 no-residual motion: %v", err)
			}
			assertEncoderNALTypes(t, second.NALUnits, []uint8{1, 1})
			if len(second.Data) >= 128 {
				t.Fatalf("per-macroblock P16x16 output size = %d, want compact no-residual slices", len(second.Data))
			}
			switch tt.format {
			case goh264.EncoderOutputAnnexB:
				assertEncoderVCLFirstMBs(t, append(append([]byte(nil), headers.AnnexB...), second.Data...), []uint8{1, 1}, []uint32{0, 2})
			case goh264.EncoderOutputRTP:
				assertEncoderVCLFirstMBs(t, append(append([]byte(nil), headers.AnnexB...), annexBFromEncoderRTPPackets(t, second.RTPPackets)...), []uint8{1, 1}, []uint32{0, 2})
			}

			dec := goh264.NewDecoder()
			var decodedFirst, decodedSecond []*goh264.Frame
			switch tt.format {
			case goh264.EncoderOutputAnnexB:
				decodedFirst, err = dec.DecodeFrames(first.Data)
				if err != nil {
					t.Fatalf("DecodeFrames first Annex B: %v", err)
				}
				decodedSecond, err = dec.DecodeFrames(second.Data)
				if err != nil {
					t.Fatalf("DecodeFrames second Annex B: %v", err)
				}
			case goh264.EncoderOutputAVC:
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
				decodedFirst, err = dec.DecodeFrames(annexBFromEncoderRTPPackets(t, first.RTPPackets))
				if err != nil {
					t.Fatalf("DecodeFrames first RTP: %v", err)
				}
				decodedSecond, err = dec.DecodeFrames(annexBFromEncoderRTPPackets(t, second.RTPPackets))
				if err != nil {
					t.Fatalf("DecodeFrames second RTP: %v", err)
				}
			default:
				t.Fatalf("unexpected format %v", tt.format)
			}
			assertDecodedEncoderFrameBytes(t, decodedFirst, appendI420FrameBytes(nil, firstFrame))
			assertDecodedEncoderFrameBytes(t, decodedSecond, appendI420FrameBytes(nil, secondFrame))
			if decodedSecond[0].KeyFrame || decodedSecond[0].SideData.RecoveryPoint != nil {
				t.Fatalf("decoded per-macroblock P frame key=%v recovery=%+v, want predictive non-recovery frame",
					decodedSecond[0].KeyFrame, decodedSecond[0].SideData.RecoveryPoint)
			}
			if tt.format == goh264.EncoderOutputAnnexB {
				stream := append(append([]byte(nil), first.Data...), second.Data...)
				wantStream := appendI420FrameBytes(nil, firstFrame)
				wantStream = appendI420FrameBytes(wantStream, secondFrame)
				assertFFmpegRawVideoOracle(t, stream, wantStream)
			}
		})
	}
}

func TestEncoderEncodeOddPixelExactP16x16NoResidualMotionWithConstantChroma(t *testing.T) {
	for _, tt := range []struct {
		name string
		dx   int
		dy   int
	}{
		{name: "horizontal", dx: 1},
		{name: "vertical", dy: -1},
		{name: "diagonal", dx: 1, dy: 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cfg := goh264.DefaultEncoderConfig(32, 32)
			cfg.OutputFormat = goh264.EncoderOutputAnnexB
			cfg.DeblockMode = goh264.EncoderDeblockDisabled
			cfg.RTPMaxPayloadSize = 0
			enc, err := goh264.NewEncoder(cfg)
			if err != nil {
				t.Fatalf("NewEncoder: %v", err)
			}

			firstFrame := patternedI420EncoderFrame(32, 32)
			setConstantI420Chroma(&firstFrame, 128, 64)
			first, err := enc.Encode(firstFrame)
			if err != nil {
				t.Fatalf("Encode first IDR: %v", err)
			}
			assertEncoderNALTypes(t, first.NALUnits, []uint8{7, 8, 5})

			secondFrame := integerMotionI420EncoderFrame(firstFrame, tt.dx, tt.dy)
			secondFrame.PTS = firstFrame.PTS + int64(cfg.RTPTimestampIncrement)
			second, err := enc.Encode(secondFrame)
			if err != nil {
				t.Fatalf("Encode odd-pixel exact P16x16 no-residual motion: %v", err)
			}
			if second.KeyFrame || second.IDR {
				t.Fatalf("odd-pixel exact-motion second frame key=%v idr=%v, want non-IDR P16x16", second.KeyFrame, second.IDR)
			}
			assertEncoderNALTypes(t, second.NALUnits, []uint8{1})

			dec := goh264.NewDecoder()
			decodedFirst, err := dec.DecodeFrames(first.Data)
			if err != nil {
				t.Fatalf("Decode first IDR: %v", err)
			}
			assertDecodedEncoderFrameBytes(t, decodedFirst, appendI420FrameBytes(nil, firstFrame))
			decodedSecond, err := dec.DecodeFrames(second.Data)
			if err != nil {
				t.Fatalf("Decode odd-pixel exact P16x16 no-residual motion: %v", err)
			}
			assertDecodedEncoderFrameBytes(t, decodedSecond, appendI420FrameBytes(nil, secondFrame))

			stream := append(append([]byte(nil), first.Data...), second.Data...)
			wantStream := appendI420FrameBytes(nil, firstFrame)
			wantStream = appendI420FrameBytes(wantStream, secondFrame)
			assertFFmpegRawVideoOracle(t, stream, wantStream)
		})
	}
}

func TestEncoderEncodeOddPixelExactP16x16NoResidualMotionForAVCAndRTP(t *testing.T) {
	for _, tt := range []struct {
		name   string
		format goh264.EncoderOutputFormat
	}{
		{name: "avc", format: goh264.EncoderOutputAVC},
		{name: "rtp", format: goh264.EncoderOutputRTP},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cfg := goh264.DefaultEncoderConfig(32, 32)
			cfg.OutputFormat = tt.format
			cfg.DeblockMode = goh264.EncoderDeblockDisabled
			if tt.format != goh264.EncoderOutputRTP {
				cfg.RTPMaxPayloadSize = 0
			}
			enc, err := goh264.NewEncoder(cfg)
			if err != nil {
				t.Fatalf("NewEncoder: %v", err)
			}

			firstFrame := patternedI420EncoderFrame(32, 32)
			setConstantI420Chroma(&firstFrame, 128, 64)
			first, err := enc.Encode(firstFrame)
			if err != nil {
				t.Fatalf("Encode first IDR: %v", err)
			}
			assertEncoderNALTypes(t, first.NALUnits, []uint8{7, 8, 5})

			secondFrame := integerMotionI420EncoderFrame(firstFrame, 1, 1)
			secondFrame.PTS = firstFrame.PTS + int64(cfg.RTPTimestampIncrement)
			second, err := enc.Encode(secondFrame)
			if err != nil {
				t.Fatalf("Encode odd-pixel exact P16x16 no-residual motion: %v", err)
			}
			if second.KeyFrame || second.IDR {
				t.Fatalf("odd-pixel exact-motion second frame key=%v idr=%v, want non-IDR P16x16", second.KeyFrame, second.IDR)
			}
			assertEncoderNALTypes(t, second.NALUnits, []uint8{1})

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
				decodedFirst, err = dec.DecodeFrames(annexBFromEncoderRTPPackets(t, first.RTPPackets))
				if err != nil {
					t.Fatalf("DecodeFrames first RTP: %v", err)
				}
				decodedSecond, err = dec.DecodeFrames(annexBFromEncoderRTPPackets(t, second.RTPPackets))
				if err != nil {
					t.Fatalf("DecodeFrames second RTP: %v", err)
				}
			default:
				t.Fatalf("unexpected format %v", tt.format)
			}
			assertDecodedEncoderFrameBytes(t, decodedFirst, appendI420FrameBytes(nil, firstFrame))
			assertDecodedEncoderFrameBytes(t, decodedSecond, appendI420FrameBytes(nil, secondFrame))
			if decodedSecond[0].KeyFrame || decodedSecond[0].SideData.RecoveryPoint != nil {
				t.Fatalf("decoded odd-pixel exact-motion P frame key=%v recovery=%+v, want predictive non-recovery frame",
					decodedSecond[0].KeyFrame, decodedSecond[0].SideData.RecoveryPoint)
			}
		})
	}
}

func TestEncoderEncodeOddPixelExactP16x16RequiresConstantChroma(t *testing.T) {
	for _, tt := range []struct {
		name   string
		format goh264.EncoderOutputFormat
		ffmpeg bool
	}{
		{name: "annexb", format: goh264.EncoderOutputAnnexB, ffmpeg: true},
		{name: "avc", format: goh264.EncoderOutputAVC},
		{name: "rtp", format: goh264.EncoderOutputRTP},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cfg := goh264.DefaultEncoderConfig(32, 32)
			cfg.OutputFormat = tt.format
			cfg.DeblockMode = goh264.EncoderDeblockDisabled
			if tt.format != goh264.EncoderOutputRTP {
				cfg.RTPMaxPayloadSize = 0
			}
			enc, err := goh264.NewEncoder(cfg)
			if err != nil {
				t.Fatalf("NewEncoder: %v", err)
			}

			firstFrame := patternedI420EncoderFrame(32, 32)
			first, err := enc.Encode(firstFrame)
			if err != nil {
				t.Fatalf("Encode first IDR: %v", err)
			}
			assertEncoderNALTypes(t, first.NALUnits, []uint8{7, 8, 5})

			secondFrame := integerMotionI420EncoderFrame(firstFrame, 1, 0)
			secondFrame.PTS = firstFrame.PTS + int64(cfg.RTPTimestampIncrement)
			second, err := enc.Encode(secondFrame)
			if err != nil {
				t.Fatalf("Encode odd-pixel patterned-chroma fallback: %v", err)
			}
			if second.KeyFrame || second.IDR {
				t.Fatalf("odd-pixel patterned-chroma fallback key=%v idr=%v, want non-IDR P IntraPCM", second.KeyFrame, second.IDR)
			}
			assertEncoderNALTypes(t, second.NALUnits, []uint8{6, 1})

			dec := goh264.NewDecoder()
			var decodedFirst, decodedSecond []*goh264.Frame
			switch tt.format {
			case goh264.EncoderOutputAnnexB:
				decodedFirst, err = dec.DecodeFrames(first.Data)
				if err != nil {
					t.Fatalf("Decode first IDR: %v", err)
				}
				decodedSecond, err = dec.DecodeFrames(second.Data)
				if err != nil {
					t.Fatalf("Decode odd-pixel patterned-chroma fallback: %v", err)
				}
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
				decodedFirst, err = dec.DecodeFrames(annexBFromEncoderRTPPackets(t, first.RTPPackets))
				if err != nil {
					t.Fatalf("DecodeFrames first RTP: %v", err)
				}
				decodedSecond, err = dec.DecodeFrames(annexBFromEncoderRTPPackets(t, second.RTPPackets))
				if err != nil {
					t.Fatalf("DecodeFrames second RTP: %v", err)
				}
			default:
				t.Fatalf("unexpected format %v", tt.format)
			}
			assertDecodedEncoderFrameBytes(t, decodedFirst, appendI420FrameBytes(nil, firstFrame))
			assertDecodedEncoderFrameBytes(t, decodedSecond, appendI420FrameBytes(nil, secondFrame))
			if !decodedSecond[0].KeyFrame ||
				decodedSecond[0].SideData.RecoveryPoint == nil ||
				decodedSecond[0].SideData.RecoveryPoint.RecoveryFrameCount != 0 {
				t.Fatalf("decoded odd-pixel fallback key=%v recovery=%+v, want immediate recovery point",
					decodedSecond[0].KeyFrame, decodedSecond[0].SideData.RecoveryPoint)
			}

			if tt.ffmpeg {
				stream := append(append([]byte(nil), first.Data...), second.Data...)
				wantStream := appendI420FrameBytes(nil, firstFrame)
				wantStream = appendI420FrameBytes(wantStream, secondFrame)
				assertFFmpegRawVideoOracle(t, stream, wantStream)
			}
		})
	}
}

func TestEncoderEncodeWideExactP16x16RequiresDeblockDisabled(t *testing.T) {
	cfg := goh264.DefaultEncoderConfig(32, 16)
	cfg.OutputFormat = goh264.EncoderOutputAnnexB
	cfg.RTPMaxPayloadSize = 0
	enc, err := goh264.NewEncoder(cfg)
	if err != nil {
		t.Fatalf("NewEncoder: %v", err)
	}
	if got := enc.Config().DeblockMode; got != goh264.EncoderDeblockEnabled {
		t.Fatalf("default deblock mode = %v, want enabled", got)
	}

	firstFrame := patternedI420EncoderFrame(32, 16)
	first, err := enc.Encode(firstFrame)
	if err != nil {
		t.Fatalf("Encode first IDR: %v", err)
	}
	assertEncoderNALTypes(t, first.NALUnits, []uint8{7, 8, 5})

	secondFrame := integerMotionI420EncoderFrame(firstFrame, 2, 0)
	secondFrame.PTS = firstFrame.PTS + int64(cfg.RTPTimestampIncrement)
	second, err := enc.Encode(secondFrame)
	if err != nil {
		t.Fatalf("Encode wide exact-motion deblock-enabled fallback: %v", err)
	}
	if second.KeyFrame || second.IDR {
		t.Fatalf("wide exact-motion fallback key=%v idr=%v, want non-IDR P IntraPCM", second.KeyFrame, second.IDR)
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
		t.Fatalf("Decode wide exact-motion deblock-enabled fallback: %v", err)
	}
	assertDecodedEncoderFrameBytes(t, decodedSecond, appendI420FrameBytes(nil, secondFrame))
	if !decodedSecond[0].KeyFrame ||
		decodedSecond[0].SideData.RecoveryPoint == nil ||
		decodedSecond[0].SideData.RecoveryPoint.RecoveryFrameCount != 0 {
		t.Fatalf("wide exact-motion fallback recovery side data key=%v recovery=%+v, want immediate recovery point",
			decodedSecond[0].KeyFrame, decodedSecond[0].SideData.RecoveryPoint)
	}

	stream := append(append([]byte(nil), first.Data...), second.Data...)
	wantStream := appendI420FrameBytes(nil, firstFrame)
	wantStream = appendI420FrameBytes(wantStream, secondFrame)
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

func TestEncoderEncodeChangedSecondFrameUsesPIntraPCMWithDefaultDeblock(t *testing.T) {
	cfg := goh264.DefaultEncoderConfig(16, 16)
	cfg.OutputFormat = goh264.EncoderOutputAnnexB
	cfg.RTPMaxPayloadSize = 0
	enc, err := goh264.NewEncoder(cfg)
	if err != nil {
		t.Fatalf("NewEncoder: %v", err)
	}
	if got := enc.Config().DeblockMode; got != goh264.EncoderDeblockEnabled {
		t.Fatalf("default deblock mode = %v, want enabled", got)
	}

	firstFrame := patternedI420EncoderFrame(16, 16)
	first, err := enc.Encode(firstFrame)
	if err != nil {
		t.Fatalf("Encode first IDR: %v", err)
	}
	assertEncoderNALTypes(t, first.NALUnits, []uint8{7, 8, 5})

	secondFrame := patternedI420EncoderFrame(16, 16)
	secondFrame.Y[0] ^= 0x6a
	secondFrame.PTS += int64(cfg.RTPTimestampIncrement)
	second, err := enc.Encode(secondFrame)
	if err != nil {
		t.Fatalf("Encode changed deblock-enabled P IntraPCM: %v", err)
	}
	if second.KeyFrame || second.IDR {
		t.Fatalf("changed deblock-enabled frame key=%v idr=%v, want non-IDR P IntraPCM", second.KeyFrame, second.IDR)
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
		t.Fatalf("Decode changed deblock-enabled P IntraPCM: %v", err)
	}
	assertDecodedEncoderFrameBytes(t, decodedSecond, appendI420FrameBytes(nil, secondFrame))
	if !decodedSecond[0].KeyFrame ||
		decodedSecond[0].SideData.RecoveryPoint == nil ||
		decodedSecond[0].SideData.RecoveryPoint.RecoveryFrameCount != 0 {
		t.Fatalf("changed deblock-enabled P recovery side data key=%v recovery=%+v, want immediate recovery point",
			decodedSecond[0].KeyFrame, decodedSecond[0].SideData.RecoveryPoint)
	}

	stream := append(append([]byte(nil), first.Data...), second.Data...)
	wantStream := appendI420FrameBytes(nil, firstFrame)
	wantStream = appendI420FrameBytes(wantStream, secondFrame)
	assertFFmpegRawVideoOracle(t, stream, wantStream)
}

func TestEncoderEncodeChangedSecondFrameUsesPIntraPCMWithSliceBoundaryDeblock(t *testing.T) {
	cfg := goh264.DefaultEncoderConfig(16, 16)
	cfg.OutputFormat = goh264.EncoderOutputAnnexB
	cfg.DeblockMode = goh264.EncoderDeblockSliceBoundary
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
	secondFrame.Y[0] ^= 0x23
	secondFrame.PTS += int64(cfg.RTPTimestampIncrement)
	second, err := enc.Encode(secondFrame)
	if err != nil {
		t.Fatalf("Encode changed slice-boundary P IntraPCM: %v", err)
	}
	if second.KeyFrame || second.IDR {
		t.Fatalf("changed slice-boundary frame key=%v idr=%v, want non-IDR P IntraPCM", second.KeyFrame, second.IDR)
	}
	assertEncoderNALTypes(t, second.NALUnits, []uint8{6, 1})

	dec := goh264.NewDecoder()
	if _, err := dec.DecodeFrames(first.Data); err != nil {
		t.Fatalf("Decode first IDR: %v", err)
	}
	decodedSecond, err := dec.DecodeFrames(second.Data)
	if err != nil {
		t.Fatalf("Decode changed slice-boundary P IntraPCM: %v", err)
	}
	assertDecodedEncoderFrameBytes(t, decodedSecond, appendI420FrameBytes(nil, secondFrame))

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

	var callbackPackets []goh264.EncoderRTPPacket
	var callbackMetadata []goh264.EncoderRTPPacketMetadata
	enc.SetRTPPacketCallback(func(pkt goh264.EncoderRTPPacket, meta goh264.EncoderRTPPacketMetadata) {
		callbackPackets = append(callbackPackets, pkt)
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
	if len(callbackPackets) != len(out.RTPPackets) || len(callbackMetadata) != len(out.RTPPackets) {
		t.Fatalf("callback packets/meta = %d/%d, want packet count %d",
			len(callbackPackets), len(callbackMetadata), len(out.RTPPackets))
	}
	assertEncoderRTPSingleNALCallbackMetadata(t, callbackPackets, callbackMetadata, out, frame, cfg, true, true)

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

func TestEncoderMaxFrameSizeRejectsOversizeAccessUnitWithoutAdvancingState(t *testing.T) {
	cfg := goh264.DefaultEncoderConfig(16, 16)
	cfg.OutputFormat = goh264.EncoderOutputAnnexB
	cfg.DeblockMode = goh264.EncoderDeblockDisabled
	cfg.FrameDrop = goh264.EncoderFrameDropDisabled
	cfg.RTPMaxPayloadSize = 0
	cfg.MaxFrameSize = 16
	enc, err := goh264.NewEncoder(cfg)
	if err != nil {
		t.Fatalf("NewEncoder: %v", err)
	}
	frame := patternedI420EncoderFrame(16, 16)

	if _, err := enc.Encode(frame); !errors.Is(err, goh264.ErrInvalidData) {
		t.Fatalf("oversize MaxFrameSize encode error = %v, want ErrInvalidData", err)
	}
	if err := enc.Reconfigure(goh264.EncoderReconfigure{MaxFrameSize: 4096}); err != nil {
		t.Fatalf("raise MaxFrameSize: %v", err)
	}
	first, err := enc.Encode(frame)
	if err != nil {
		t.Fatalf("Encode after rejected MaxFrameSize frame: %v", err)
	}
	assertEncoderNALTypes(t, first.NALUnits, []uint8{7, 8, 5})
	assertEncoderVCLFrameNums(t, first.Data, []uint8{5}, []uint32{0})

	secondFrame := frame
	secondFrame.PTS += int64(cfg.RTPTimestampIncrement)
	second, err := enc.Encode(secondFrame)
	if err != nil {
		t.Fatalf("Encode second frame after MaxFrameSize recovery: %v", err)
	}
	assertEncoderNALTypes(t, second.NALUnits, []uint8{1})
	assertEncoderVCLFrameNums(t, append(append([]byte(nil), first.Data...), second.Data...), []uint8{5, 1}, []uint32{0, 1})
}

func TestEncoderSliceMaxBytesRejectsOversizeSliceWithoutAdvancingRTPState(t *testing.T) {
	cfg := goh264.DefaultEncoderConfig(16, 16)
	cfg.DeblockMode = goh264.EncoderDeblockDisabled
	cfg.FrameDrop = goh264.EncoderFrameDropDisabled
	cfg.SliceMaxBytes = 1
	enc, err := goh264.NewEncoder(cfg)
	if err != nil {
		t.Fatalf("NewEncoder: %v", err)
	}
	frame := patternedI420EncoderFrame(16, 16)

	if _, err := enc.Encode(frame); !errors.Is(err, goh264.ErrInvalidData) {
		t.Fatalf("oversize SliceMaxBytes encode error = %v, want ErrInvalidData", err)
	}
	if err := enc.Reconfigure(goh264.EncoderReconfigure{SliceMaxBytes: 4096}); err != nil {
		t.Fatalf("raise SliceMaxBytes: %v", err)
	}
	out, err := enc.Encode(frame)
	if err != nil {
		t.Fatalf("Encode after rejected SliceMaxBytes frame: %v", err)
	}
	assertEncoderNALTypes(t, out.NALUnits, []uint8{7, 8, 5})
	assertEncoderVCLFrameNums(t, out.Data, []uint8{5}, []uint32{0})
	assertRTPPacketMetadata(t, out.RTPPackets, cfg.RTPPayloadType, cfg.RTPSSRC, 0)
}

func TestEncoderFrameDropToBitrateDropsOversizeFrameWithoutAdvancingReferenceOrPacketState(t *testing.T) {
	cfg := goh264.DefaultEncoderConfig(16, 16)
	cfg.DeblockMode = goh264.EncoderDeblockDisabled
	cfg.MaxFrameSize = 4096
	enc, err := goh264.NewEncoder(cfg)
	if err != nil {
		t.Fatalf("NewEncoder: %v", err)
	}

	var callbackCalls int
	enc.SetRTPPacketCallback(func(goh264.EncoderRTPPacket, goh264.EncoderRTPPacketMetadata) {
		callbackCalls++
	})
	firstFrame := patternedI420EncoderFrame(16, 16)
	firstFrame.PTS = 0
	first, err := enc.Encode(firstFrame)
	if err != nil {
		t.Fatalf("Encode first IDR: %v", err)
	}
	assertEncoderNALTypes(t, first.NALUnits, []uint8{7, 8, 5})
	firstPacketCount := len(first.RTPPackets)
	if firstPacketCount == 0 || callbackCalls != firstPacketCount {
		t.Fatalf("first RTP packets/callbacks = %d/%d, want nonzero matching count", firstPacketCount, callbackCalls)
	}

	if err := enc.Reconfigure(goh264.EncoderReconfigure{MaxFrameSize: 16}); err != nil {
		t.Fatalf("lower MaxFrameSize: %v", err)
	}
	droppedFrame := patternedI420EncoderFrame(16, 16)
	droppedFrame.PTS = 0
	droppedFrame.Y[0] ^= 0x40
	dropped, err := enc.Encode(droppedFrame)
	if err != nil {
		t.Fatalf("Encode dropped bitrate frame: %v", err)
	}
	if !dropped.Dropped || len(dropped.Data) != 0 || len(dropped.NALUnits) != 0 || len(dropped.RTPPackets) != 0 {
		t.Fatalf("dropped frame = %+v, want dropped metadata without output", dropped)
	}
	if dropped.RTPTime != first.RTPTime+cfg.RTPTimestampIncrement {
		t.Fatalf("dropped RTP time = %d, want %d", dropped.RTPTime, first.RTPTime+cfg.RTPTimestampIncrement)
	}
	if callbackCalls != firstPacketCount {
		t.Fatalf("dropped frame invoked callback count %d, want still %d", callbackCalls, firstPacketCount)
	}

	if err := enc.Reconfigure(goh264.EncoderReconfigure{MaxFrameSize: 4096}); err != nil {
		t.Fatalf("raise MaxFrameSize: %v", err)
	}
	thirdFrame := firstFrame
	thirdFrame.PTS = 0
	third, err := enc.Encode(thirdFrame)
	if err != nil {
		t.Fatalf("Encode after dropped bitrate frame: %v", err)
	}
	if third.Dropped || third.IDR {
		t.Fatalf("post-drop frame dropped=%v idr=%v, want transmitted P-skip", third.Dropped, third.IDR)
	}
	if third.RTPTime != dropped.RTPTime+cfg.RTPTimestampIncrement {
		t.Fatalf("post-drop RTP time = %d, want %d", third.RTPTime, dropped.RTPTime+cfg.RTPTimestampIncrement)
	}
	assertEncoderNALTypes(t, third.NALUnits, []uint8{1})
	assertEncoderVCLFrameNums(t, append(append([]byte(nil), first.Data...), third.Data...), []uint8{5, 1}, []uint32{0, 1})
	assertRTPPacketMetadata(t, third.RTPPackets, cfg.RTPPayloadType, cfg.RTPSSRC, uint16(firstPacketCount))
}

func TestEncoderEncodeIntoFrameDropToBitrateReturnsEmptyOutputAndPreservesState(t *testing.T) {
	cfg := goh264.DefaultEncoderConfig(16, 16)
	cfg.DeblockMode = goh264.EncoderDeblockDisabled
	cfg.RTPMaxPayloadSize = 32
	cfg.MaxFrameSize = 4096
	enc, err := goh264.NewEncoder(cfg)
	if err != nil {
		t.Fatalf("NewEncoder: %v", err)
	}

	var callbackCalls int
	enc.SetRTPPacketCallback(func(goh264.EncoderRTPPacket, goh264.EncoderRTPPacketMetadata) {
		callbackCalls++
	})
	dst := make([]byte, 0, 4096)
	firstFrame := patternedI420EncoderFrame(16, 16)
	firstFrame.PTS = 0
	first, err := enc.EncodeInto(dst[:0], firstFrame)
	if err != nil {
		t.Fatalf("EncodeInto first IDR: %v", err)
	}
	if first.Dropped || !first.IDR || first.RTPTime != 0 {
		t.Fatalf("first EncodeInto output dropped/id/time = %v/%v/%d, want IDR time 0",
			first.Dropped, first.IDR, first.RTPTime)
	}
	if cap(first.Data) != cap(dst) {
		t.Fatalf("first EncodeInto data cap = %d, want caller cap %d", cap(first.Data), cap(dst))
	}
	assertEncoderNALTypes(t, first.NALUnits, []uint8{7, 8, 5})
	assertRTPPacketMetadata(t, first.RTPPackets, cfg.RTPPayloadType, cfg.RTPSSRC, 0)
	firstPacketCount := len(first.RTPPackets)
	if firstPacketCount == 0 || callbackCalls != firstPacketCount {
		t.Fatalf("first RTP packets/callbacks = %d/%d, want nonzero matching count",
			firstPacketCount, callbackCalls)
	}
	firstData := append([]byte(nil), first.Data...)

	if err := enc.Reconfigure(goh264.EncoderReconfigure{MaxFrameSize: 16}); err != nil {
		t.Fatalf("lower MaxFrameSize: %v", err)
	}
	droppedFrame := patternedI420EncoderFrame(16, 16)
	droppedFrame.PTS = 0
	droppedFrame.Y[0] ^= 0x40
	dropped, err := enc.EncodeInto(dst[:0], droppedFrame)
	if err != nil {
		t.Fatalf("EncodeInto dropped bitrate frame: %v", err)
	}
	if !dropped.Dropped || len(dropped.Data) != 0 || len(dropped.NALUnits) != 0 || len(dropped.RTPPackets) != 0 {
		t.Fatalf("EncodeInto dropped frame = %+v, want dropped metadata without returned output", dropped)
	}
	if dropped.RTPTime != first.RTPTime+cfg.RTPTimestampIncrement {
		t.Fatalf("dropped RTP time = %d, want %d", dropped.RTPTime, first.RTPTime+cfg.RTPTimestampIncrement)
	}
	if callbackCalls != firstPacketCount {
		t.Fatalf("EncodeInto dropped frame invoked callbacks = %d, want still %d", callbackCalls, firstPacketCount)
	}

	if err := enc.Reconfigure(goh264.EncoderReconfigure{MaxFrameSize: 4096}); err != nil {
		t.Fatalf("raise MaxFrameSize: %v", err)
	}
	recoveredFrame := firstFrame
	recoveredFrame.PTS = 0
	recovered, err := enc.EncodeInto(dst[:0], recoveredFrame)
	if err != nil {
		t.Fatalf("EncodeInto after dropped bitrate frame: %v", err)
	}
	if recovered.Dropped || recovered.IDR {
		t.Fatalf("post-drop EncodeInto output dropped=%v idr=%v, want transmitted P-skip",
			recovered.Dropped, recovered.IDR)
	}
	if recovered.RTPTime != dropped.RTPTime+cfg.RTPTimestampIncrement {
		t.Fatalf("post-drop RTP time = %d, want %d", recovered.RTPTime, dropped.RTPTime+cfg.RTPTimestampIncrement)
	}
	if cap(recovered.Data) != cap(dst) {
		t.Fatalf("post-drop EncodeInto data cap = %d, want caller cap %d", cap(recovered.Data), cap(dst))
	}
	assertEncoderNALTypes(t, recovered.NALUnits, []uint8{1})
	assertEncoderVCLFrameNums(t, append(append([]byte(nil), firstData...), recovered.Data...), []uint8{5, 1}, []uint32{0, 1})
	assertRTPPacketMetadata(t, recovered.RTPPackets, cfg.RTPPayloadType, cfg.RTPSSRC, uint16(firstPacketCount))
	if callbackCalls != firstPacketCount+len(recovered.RTPPackets) {
		t.Fatalf("post-drop callbacks = %d, want %d", callbackCalls, firstPacketCount+len(recovered.RTPPackets))
	}

	callbacksAfterRecovered := callbackCalls
	if err := enc.Reconfigure(goh264.EncoderReconfigure{MaxFrameSize: 16}); err != nil {
		t.Fatalf("lower MaxFrameSize for forced IDR drop: %v", err)
	}
	enc.ForceIDR()
	if !enc.PendingIDR() {
		t.Fatal("ForceIDR before bitrate drop did not queue IDR")
	}
	forcedDropFrame := firstFrame
	forcedDropFrame.PTS = 0
	forcedDropped, err := enc.EncodeInto(dst[:0], forcedDropFrame)
	if err != nil {
		t.Fatalf("EncodeInto forced IDR bitrate drop: %v", err)
	}
	if !forcedDropped.Dropped || len(forcedDropped.Data) != 0 || len(forcedDropped.NALUnits) != 0 || len(forcedDropped.RTPPackets) != 0 {
		t.Fatalf("forced IDR bitrate drop output = %+v, want dropped metadata without output", forcedDropped)
	}
	if forcedDropped.RTPTime != recovered.RTPTime+cfg.RTPTimestampIncrement {
		t.Fatalf("forced IDR bitrate drop RTP time = %d, want %d", forcedDropped.RTPTime, recovered.RTPTime+cfg.RTPTimestampIncrement)
	}
	if !enc.PendingIDR() {
		t.Fatal("bitrate-dropped forced IDR consumed pending IDR")
	}
	if callbackCalls != callbacksAfterRecovered {
		t.Fatalf("forced IDR bitrate drop callbacks = %d, want still %d", callbackCalls, callbacksAfterRecovered)
	}

	if err := enc.Reconfigure(goh264.EncoderReconfigure{MaxFrameSize: 4096}); err != nil {
		t.Fatalf("raise MaxFrameSize for forced IDR: %v", err)
	}
	forced, err := enc.EncodeInto(dst[:0], forcedDropFrame)
	if err != nil {
		t.Fatalf("EncodeInto after forced IDR bitrate drop: %v", err)
	}
	if forced.Dropped || !forced.IDR || enc.PendingIDR() {
		t.Fatalf("post-bitrate-drop forced output dropped=%v idr=%v pending=%v, want transmitted IDR",
			forced.Dropped, forced.IDR, enc.PendingIDR())
	}
	if forced.RTPTime != forcedDropped.RTPTime+cfg.RTPTimestampIncrement {
		t.Fatalf("post-bitrate-drop forced RTP time = %d, want %d", forced.RTPTime, forcedDropped.RTPTime+cfg.RTPTimestampIncrement)
	}
	assertEncoderNALTypes(t, forced.NALUnits, []uint8{7, 8, 5})
	assertRTPPacketMetadata(t, forced.RTPPackets, cfg.RTPPayloadType, cfg.RTPSSRC, uint16(callbacksAfterRecovered))
	if callbackCalls != callbacksAfterRecovered+len(forced.RTPPackets) {
		t.Fatalf("post-bitrate-drop forced callbacks = %d, want %d", callbackCalls, callbacksAfterRecovered+len(forced.RTPPackets))
	}
}

func TestEncoderFrameDropToBitrateDropsOversizeSliceWithoutAdvancingFrameState(t *testing.T) {
	cfg := goh264.DefaultEncoderConfig(16, 16)
	cfg.DeblockMode = goh264.EncoderDeblockDisabled
	cfg.SliceMaxBytes = 1
	enc, err := goh264.NewEncoder(cfg)
	if err != nil {
		t.Fatalf("NewEncoder: %v", err)
	}
	frame := patternedI420EncoderFrame(16, 16)
	frame.PTS = 0

	dropped, err := enc.Encode(frame)
	if err != nil {
		t.Fatalf("Encode dropped slice-budget frame: %v", err)
	}
	if !dropped.Dropped || len(dropped.Data) != 0 || len(dropped.NALUnits) != 0 || len(dropped.RTPPackets) != 0 {
		t.Fatalf("slice-budget dropped frame = %+v, want dropped metadata without output", dropped)
	}
	if dropped.RTPTime != 0 {
		t.Fatalf("first dropped RTP time = %d, want 0", dropped.RTPTime)
	}

	if err := enc.Reconfigure(goh264.EncoderReconfigure{SliceMaxBytes: 4096}); err != nil {
		t.Fatalf("raise SliceMaxBytes: %v", err)
	}
	out, err := enc.Encode(frame)
	if err != nil {
		t.Fatalf("Encode after dropped slice-budget frame: %v", err)
	}
	if out.Dropped || !out.IDR {
		t.Fatalf("post-slice-drop frame dropped=%v idr=%v, want first transmitted IDR", out.Dropped, out.IDR)
	}
	if out.RTPTime != cfg.RTPTimestampIncrement {
		t.Fatalf("post-slice-drop RTP time = %d, want %d", out.RTPTime, cfg.RTPTimestampIncrement)
	}
	assertEncoderNALTypes(t, out.NALUnits, []uint8{7, 8, 5})
	assertEncoderVCLFrameNums(t, out.Data, []uint8{5}, []uint32{0})
	assertRTPPacketMetadata(t, out.RTPPackets, cfg.RTPPayloadType, cfg.RTPSSRC, 0)
}

func TestEncoderFrameDropToBitrateDropsChangedOversizeSliceWithoutAdvancingReferenceOrPacketState(t *testing.T) {
	cfg := goh264.DefaultEncoderConfig(16, 16)
	cfg.DeblockMode = goh264.EncoderDeblockDisabled
	cfg.SliceMaxBytes = 4096
	enc, err := goh264.NewEncoder(cfg)
	if err != nil {
		t.Fatalf("NewEncoder: %v", err)
	}

	var callbackCalls int
	enc.SetRTPPacketCallback(func(goh264.EncoderRTPPacket, goh264.EncoderRTPPacketMetadata) {
		callbackCalls++
	})
	firstFrame := patternedI420EncoderFrame(16, 16)
	firstFrame.PTS = 0
	first, err := enc.Encode(firstFrame)
	if err != nil {
		t.Fatalf("Encode first IDR: %v", err)
	}
	assertEncoderNALTypes(t, first.NALUnits, []uint8{7, 8, 5})
	firstPacketCount := len(first.RTPPackets)
	if firstPacketCount == 0 || callbackCalls != firstPacketCount {
		t.Fatalf("first RTP packets/callbacks = %d/%d, want nonzero matching count", firstPacketCount, callbackCalls)
	}

	if err := enc.Reconfigure(goh264.EncoderReconfigure{SliceMaxBytes: 1}); err != nil {
		t.Fatalf("lower SliceMaxBytes: %v", err)
	}
	droppedFrame := patternedI420EncoderFrame(16, 16)
	droppedFrame.PTS = 0
	droppedFrame.Y[0] ^= 0x40
	dropped, err := enc.Encode(droppedFrame)
	if err != nil {
		t.Fatalf("Encode dropped changed slice-budget frame: %v", err)
	}
	if !dropped.Dropped || len(dropped.Data) != 0 || len(dropped.NALUnits) != 0 || len(dropped.RTPPackets) != 0 {
		t.Fatalf("dropped changed slice-budget frame = %+v, want dropped metadata without output", dropped)
	}
	if dropped.RTPTime != first.RTPTime+cfg.RTPTimestampIncrement {
		t.Fatalf("dropped changed slice-budget RTP time = %d, want %d", dropped.RTPTime, first.RTPTime+cfg.RTPTimestampIncrement)
	}
	if callbackCalls != firstPacketCount {
		t.Fatalf("dropped changed slice-budget frame invoked callback count %d, want still %d", callbackCalls, firstPacketCount)
	}

	if err := enc.Reconfigure(goh264.EncoderReconfigure{SliceMaxBytes: 4096}); err != nil {
		t.Fatalf("raise SliceMaxBytes: %v", err)
	}
	recoveredFrame := firstFrame
	recoveredFrame.PTS = 0
	recovered, err := enc.Encode(recoveredFrame)
	if err != nil {
		t.Fatalf("Encode after dropped changed slice-budget frame: %v", err)
	}
	if recovered.Dropped || recovered.IDR {
		t.Fatalf("post-slice-drop frame dropped=%v idr=%v, want transmitted P-skip", recovered.Dropped, recovered.IDR)
	}
	if recovered.RTPTime != dropped.RTPTime+cfg.RTPTimestampIncrement {
		t.Fatalf("post-slice-drop RTP time = %d, want %d", recovered.RTPTime, dropped.RTPTime+cfg.RTPTimestampIncrement)
	}
	assertEncoderNALTypes(t, recovered.NALUnits, []uint8{1})
	assertEncoderVCLFrameNums(t, append(append([]byte(nil), first.Data...), recovered.Data...), []uint8{5, 1}, []uint32{0, 1})
	assertRTPPacketMetadata(t, recovered.RTPPackets, cfg.RTPPayloadType, cfg.RTPSSRC, uint16(firstPacketCount))
	if callbackCalls != firstPacketCount+len(recovered.RTPPackets) {
		t.Fatalf("post-slice-drop callbacks = %d, want %d", callbackCalls, firstPacketCount+len(recovered.RTPPackets))
	}

	recoveredPacketCount := len(recovered.RTPPackets)
	if err := enc.Reconfigure(goh264.EncoderReconfigure{SliceMaxBytes: 1}); err != nil {
		t.Fatalf("lower SliceMaxBytes before forced IDR drop: %v", err)
	}
	enc.ForceIDR()
	if !enc.PendingIDR() {
		t.Fatal("ForceIDR before slice-budget drop did not queue IDR")
	}
	forcedDropFrame := firstFrame
	forcedDropFrame.PTS = 0
	forcedDropFrame.Y[0] ^= 0x20
	forcedDrop, err := enc.Encode(forcedDropFrame)
	if err != nil {
		t.Fatalf("Encode forced IDR slice-budget drop: %v", err)
	}
	if !forcedDrop.Dropped || len(forcedDrop.Data) != 0 || len(forcedDrop.NALUnits) != 0 || len(forcedDrop.RTPPackets) != 0 {
		t.Fatalf("forced IDR slice-budget drop = %+v, want dropped metadata without output", forcedDrop)
	}
	if forcedDrop.RTPTime != recovered.RTPTime+cfg.RTPTimestampIncrement {
		t.Fatalf("forced IDR slice-budget drop RTP time = %d, want %d", forcedDrop.RTPTime, recovered.RTPTime+cfg.RTPTimestampIncrement)
	}
	if !enc.PendingIDR() {
		t.Fatal("forced IDR slice-budget drop cleared pending IDR before an IDR was transmitted")
	}
	if callbackCalls != firstPacketCount+recoveredPacketCount {
		t.Fatalf("forced IDR slice-budget drop callbacks = %d, want still %d", callbackCalls, firstPacketCount+recoveredPacketCount)
	}

	if err := enc.Reconfigure(goh264.EncoderReconfigure{SliceMaxBytes: 4096}); err != nil {
		t.Fatalf("raise SliceMaxBytes before forced IDR recovery: %v", err)
	}
	forcedRecoverFrame := firstFrame
	forcedRecoverFrame.PTS = 0
	forcedRecover, err := enc.Encode(forcedRecoverFrame)
	if err != nil {
		t.Fatalf("Encode after forced IDR slice-budget drop: %v", err)
	}
	if forcedRecover.Dropped || !forcedRecover.IDR || enc.PendingIDR() {
		t.Fatalf("post-forced-slice-drop frame dropped=%v idr=%v pending=%v, want transmitted IDR and cleared pending state",
			forcedRecover.Dropped, forcedRecover.IDR, enc.PendingIDR())
	}
	if forcedRecover.RTPTime != forcedDrop.RTPTime+cfg.RTPTimestampIncrement {
		t.Fatalf("post-forced-slice-drop RTP time = %d, want %d", forcedRecover.RTPTime, forcedDrop.RTPTime+cfg.RTPTimestampIncrement)
	}
	assertEncoderNALTypes(t, forcedRecover.NALUnits, []uint8{7, 8, 5})
	assertRTPPacketMetadata(t, forcedRecover.RTPPackets, cfg.RTPPayloadType, cfg.RTPSSRC, uint16(firstPacketCount+recoveredPacketCount))
	if callbackCalls != firstPacketCount+recoveredPacketCount+len(forcedRecover.RTPPackets) {
		t.Fatalf("post-forced-slice-drop callbacks = %d, want %d",
			callbackCalls, firstPacketCount+recoveredPacketCount+len(forcedRecover.RTPPackets))
	}
}

func TestEncoderFrameDropToBitrateDropsMaxBitrateBudgetWithoutAdvancingState(t *testing.T) {
	cfg := goh264.DefaultEncoderConfig(16, 16)
	cfg.DeblockMode = goh264.EncoderDeblockDisabled
	cfg.TargetBitrate = 1_000
	cfg.MaxBitrate = 1_000
	cfg.VBVBufferSize = 64
	enc, err := goh264.NewEncoder(cfg)
	if err != nil {
		t.Fatalf("NewEncoder: %v", err)
	}

	var callbackCalls int
	enc.SetRTPPacketCallback(func(goh264.EncoderRTPPacket, goh264.EncoderRTPPacketMetadata) {
		callbackCalls++
	})
	frame := patternedI420EncoderFrame(16, 16)
	frame.PTS = 0

	dropped, err := enc.Encode(frame)
	if err != nil {
		t.Fatalf("Encode bitrate-budget frame: %v", err)
	}
	if !dropped.Dropped || len(dropped.Data) != 0 || len(dropped.NALUnits) != 0 || len(dropped.RTPPackets) != 0 {
		t.Fatalf("bitrate-budget dropped frame = %+v, want dropped metadata without output", dropped)
	}
	if dropped.RTPTime != 0 {
		t.Fatalf("first bitrate-budget dropped RTP time = %d, want 0", dropped.RTPTime)
	}
	if callbackCalls != 0 {
		t.Fatalf("bitrate-budget dropped frame invoked callback count %d, want 0", callbackCalls)
	}

	vbv := 1_000_000
	if err := enc.Reconfigure(goh264.EncoderReconfigure{MaxBitrate: 1_000_000, VBVBufferSize: &vbv}); err != nil {
		t.Fatalf("raise MaxBitrate/VBV: %v", err)
	}
	out, err := enc.Encode(frame)
	if err != nil {
		t.Fatalf("Encode after bitrate-budget drop: %v", err)
	}
	if out.Dropped || !out.IDR {
		t.Fatalf("post-bitrate-drop frame dropped=%v idr=%v, want first transmitted IDR", out.Dropped, out.IDR)
	}
	if out.RTPTime != cfg.RTPTimestampIncrement {
		t.Fatalf("post-bitrate-drop RTP time = %d, want %d", out.RTPTime, cfg.RTPTimestampIncrement)
	}
	assertEncoderNALTypes(t, out.NALUnits, []uint8{7, 8, 5})
	assertEncoderVCLFrameNums(t, out.Data, []uint8{5}, []uint32{0})
	if callbackCalls != len(out.RTPPackets) {
		t.Fatalf("post-bitrate-drop callback count = %d, want %d", callbackCalls, len(out.RTPPackets))
	}
	assertRTPPacketMetadata(t, out.RTPPackets, cfg.RTPPayloadType, cfg.RTPSSRC, 0)
}

func TestEncoderFrameDropToBitrateConsumesAndRefillsMaxBitrateCredit(t *testing.T) {
	frame := patternedI420EncoderFrame(16, 16)
	frame.PTS = 0
	changedFrame := patternedI420EncoderFrame(16, 16)
	changedFrame.PTS = 0
	changedFrame.Y[0] ^= 0x5a

	probeCfg := goh264.DefaultEncoderConfig(16, 16)
	probeCfg.DeblockMode = goh264.EncoderDeblockDisabled
	probeCfg.FrameDrop = goh264.EncoderFrameDropDisabled
	probe, err := goh264.NewEncoder(probeCfg)
	if err != nil {
		t.Fatalf("NewEncoder probe: %v", err)
	}
	probeIDR, err := probe.Encode(frame)
	if err != nil {
		t.Fatalf("probe IDR: %v", err)
	}
	probePSkip, err := probe.Encode(frame)
	if err != nil {
		t.Fatalf("probe P-skip: %v", err)
	}
	probeChanged, err := probe.Encode(changedFrame)
	if err != nil {
		t.Fatalf("probe changed P: %v", err)
	}
	idrBytes := len(probeIDR.Data)
	pskipBytes := len(probePSkip.Data)
	changedBytes := len(probeChanged.Data)
	if idrBytes == 0 || pskipBytes == 0 || changedBytes <= pskipBytes {
		t.Fatalf("probe sizes IDR/P-skip/changed = %d/%d/%d, want changed > p-skip > 0",
			idrBytes, pskipBytes, changedBytes)
	}

	cfg := goh264.DefaultEncoderConfig(16, 16)
	cfg.DeblockMode = goh264.EncoderDeblockDisabled
	cfg.TargetBitrate = pskipBytes * 8 * cfg.FrameRateNum / cfg.FrameRateDen
	cfg.MaxBitrate = cfg.TargetBitrate
	cfg.VBVBufferSize = idrBytes * 8
	enc, err := goh264.NewEncoder(cfg)
	if err != nil {
		t.Fatalf("NewEncoder: %v", err)
	}

	var callbackCalls int
	enc.SetRTPPacketCallback(func(goh264.EncoderRTPPacket, goh264.EncoderRTPPacketMetadata) {
		callbackCalls++
	})

	first, err := enc.Encode(frame)
	if err != nil {
		t.Fatalf("Encode initial budgeted IDR: %v", err)
	}
	if first.Dropped || !first.IDR || len(first.Data) != idrBytes {
		t.Fatalf("initial budgeted output dropped=%v idr=%v data=%d, want transmitted IDR size %d",
			first.Dropped, first.IDR, len(first.Data), idrBytes)
	}
	firstPacketCount := len(first.RTPPackets)
	if callbackCalls != firstPacketCount {
		t.Fatalf("initial callbacks = %d, want %d", callbackCalls, firstPacketCount)
	}

	second, err := enc.Encode(frame)
	if err != nil {
		t.Fatalf("Encode budgeted P-skip: %v", err)
	}
	if second.Dropped || second.IDR || len(second.Data) != pskipBytes {
		t.Fatalf("budgeted P-skip dropped=%v idr=%v data=%d, want transmitted P-skip size %d",
			second.Dropped, second.IDR, len(second.Data), pskipBytes)
	}
	if second.RTPTime != first.RTPTime+cfg.RTPTimestampIncrement {
		t.Fatalf("second RTP time = %d, want %d", second.RTPTime, first.RTPTime+cfg.RTPTimestampIncrement)
	}
	secondPacketStart := firstPacketCount
	assertRTPPacketMetadata(t, second.RTPPackets, cfg.RTPPayloadType, cfg.RTPSSRC, uint16(secondPacketStart))

	dropped, err := enc.Encode(changedFrame)
	if err != nil {
		t.Fatalf("Encode changed frame over bitrate credit: %v", err)
	}
	if !dropped.Dropped || len(dropped.Data) != 0 || len(dropped.NALUnits) != 0 || len(dropped.RTPPackets) != 0 {
		t.Fatalf("changed bitrate-credit frame = %+v, want dropped metadata without output", dropped)
	}
	if dropped.RTPTime != second.RTPTime+cfg.RTPTimestampIncrement {
		t.Fatalf("dropped RTP time = %d, want %d", dropped.RTPTime, second.RTPTime+cfg.RTPTimestampIncrement)
	}
	callbackAfterSecond := firstPacketCount + len(second.RTPPackets)
	if callbackCalls != callbackAfterSecond {
		t.Fatalf("dropped frame callbacks = %d, want still %d", callbackCalls, callbackAfterSecond)
	}

	recovered, err := enc.Encode(frame)
	if err != nil {
		t.Fatalf("Encode after bitrate-credit drop: %v", err)
	}
	if recovered.Dropped || recovered.IDR || len(recovered.Data) != pskipBytes {
		t.Fatalf("post-drop budgeted output dropped=%v idr=%v data=%d, want transmitted P-skip size %d",
			recovered.Dropped, recovered.IDR, len(recovered.Data), pskipBytes)
	}
	if recovered.RTPTime != dropped.RTPTime+cfg.RTPTimestampIncrement {
		t.Fatalf("recovered RTP time = %d, want %d", recovered.RTPTime, dropped.RTPTime+cfg.RTPTimestampIncrement)
	}
	assertEncoderNALTypes(t, recovered.NALUnits, []uint8{1})
	assertEncoderVCLFrameNums(t,
		append(append(append([]byte(nil), first.Data...), second.Data...), recovered.Data...),
		[]uint8{5, 1, 1},
		[]uint32{0, 1, 2},
	)
	assertRTPPacketMetadata(t, recovered.RTPPackets, cfg.RTPPayloadType, cfg.RTPSSRC, uint16(callbackAfterSecond))
	if callbackCalls != callbackAfterSecond+len(recovered.RTPPackets) {
		t.Fatalf("post-drop callbacks = %d, want %d", callbackCalls, callbackAfterSecond+len(recovered.RTPPackets))
	}
}

func TestEncoderReconfigureLowerBitrateBudgetResetsCreditBeforeNextFrame(t *testing.T) {
	frame := patternedI420EncoderFrame(16, 16)
	frame.PTS = 0

	probeCfg := goh264.DefaultEncoderConfig(16, 16)
	probeCfg.DeblockMode = goh264.EncoderDeblockDisabled
	probeCfg.FrameDrop = goh264.EncoderFrameDropDisabled
	probe, err := goh264.NewEncoder(probeCfg)
	if err != nil {
		t.Fatalf("NewEncoder probe: %v", err)
	}
	if _, err := probe.Encode(frame); err != nil {
		t.Fatalf("probe IDR: %v", err)
	}
	probePSkip, err := probe.Encode(frame)
	if err != nil {
		t.Fatalf("probe P-skip: %v", err)
	}
	pskipBytes := len(probePSkip.Data)
	if pskipBytes < 2 {
		t.Fatalf("probe P-skip size = %d, want at least 2 bytes", pskipBytes)
	}

	cfg := goh264.DefaultEncoderConfig(16, 16)
	cfg.DeblockMode = goh264.EncoderDeblockDisabled
	cfg.TargetBitrate = 1_000_000
	cfg.MaxBitrate = 1_000_000
	cfg.VBVBufferSize = 1_000_000
	enc, err := goh264.NewEncoder(cfg)
	if err != nil {
		t.Fatalf("NewEncoder: %v", err)
	}

	var callbackCalls int
	enc.SetRTPPacketCallback(func(goh264.EncoderRTPPacket, goh264.EncoderRTPPacketMetadata) {
		callbackCalls++
	})

	first, err := enc.Encode(frame)
	if err != nil {
		t.Fatalf("Encode high-budget IDR: %v", err)
	}
	if first.Dropped || !first.IDR {
		t.Fatalf("high-budget IDR dropped=%v idr=%v, want transmitted IDR", first.Dropped, first.IDR)
	}
	firstPacketCount := len(first.RTPPackets)
	if callbackCalls != firstPacketCount {
		t.Fatalf("high-budget IDR callbacks = %d, want %d", callbackCalls, firstPacketCount)
	}

	second, err := enc.Encode(frame)
	if err != nil {
		t.Fatalf("Encode high-budget P-skip: %v", err)
	}
	if second.Dropped || second.IDR || len(second.Data) != pskipBytes {
		t.Fatalf("high-budget P-skip dropped=%v idr=%v data=%d, want transmitted P-skip size %d",
			second.Dropped, second.IDR, len(second.Data), pskipBytes)
	}
	secondPacketStart := firstPacketCount
	assertRTPPacketMetadata(t, second.RTPPackets, cfg.RTPPayloadType, cfg.RTPSSRC, uint16(secondPacketStart))
	callbackAfterSecond := firstPacketCount + len(second.RTPPackets)
	if callbackCalls != callbackAfterSecond {
		t.Fatalf("high-budget P-skip callbacks = %d, want %d", callbackCalls, callbackAfterSecond)
	}

	lowCreditBytes := pskipBytes - 1
	lowBudgetBits := lowCreditBytes * 8
	lowBitrate := lowBudgetBits * cfg.FrameRateNum / cfg.FrameRateDen
	if err := enc.Reconfigure(goh264.EncoderReconfigure{
		TargetBitrate: lowBitrate,
		MaxBitrate:    lowBitrate,
		VBVBufferSize: &lowBudgetBits,
	}); err != nil {
		t.Fatalf("lower bitrate/VBV budget: %v", err)
	}
	dropped, err := enc.Encode(frame)
	if err != nil {
		t.Fatalf("Encode after lowered bitrate budget: %v", err)
	}
	if !dropped.Dropped || len(dropped.Data) != 0 || len(dropped.NALUnits) != 0 || len(dropped.RTPPackets) != 0 {
		t.Fatalf("lowered-budget frame = %+v, want dropped metadata without output", dropped)
	}
	if dropped.RTPTime != second.RTPTime+cfg.RTPTimestampIncrement {
		t.Fatalf("lowered-budget RTP time = %d, want %d", dropped.RTPTime, second.RTPTime+cfg.RTPTimestampIncrement)
	}
	if callbackCalls != callbackAfterSecond {
		t.Fatalf("lowered-budget callbacks = %d, want still %d", callbackCalls, callbackAfterSecond)
	}

	vbv := 1_000_000
	if err := enc.Reconfigure(goh264.EncoderReconfigure{
		TargetBitrate: 1_000_000,
		MaxBitrate:    1_000_000,
		VBVBufferSize: &vbv,
	}); err != nil {
		t.Fatalf("raise bitrate/VBV budget: %v", err)
	}
	recovered, err := enc.Encode(frame)
	if err != nil {
		t.Fatalf("Encode after lowered-budget drop: %v", err)
	}
	if recovered.Dropped || recovered.IDR || len(recovered.Data) != pskipBytes {
		t.Fatalf("post-lowered-budget output dropped=%v idr=%v data=%d, want transmitted P-skip size %d",
			recovered.Dropped, recovered.IDR, len(recovered.Data), pskipBytes)
	}
	if recovered.RTPTime != dropped.RTPTime+cfg.RTPTimestampIncrement {
		t.Fatalf("post-lowered-budget RTP time = %d, want %d",
			recovered.RTPTime, dropped.RTPTime+cfg.RTPTimestampIncrement)
	}
	assertEncoderNALTypes(t, recovered.NALUnits, []uint8{1})
	assertEncoderVCLFrameNums(t,
		append(append(append([]byte(nil), first.Data...), second.Data...), recovered.Data...),
		[]uint8{5, 1, 1},
		[]uint32{0, 1, 2},
	)
	assertRTPPacketMetadata(t, recovered.RTPPackets, cfg.RTPPayloadType, cfg.RTPSSRC, uint16(callbackAfterSecond))
	if callbackCalls != callbackAfterSecond+len(recovered.RTPPackets) {
		t.Fatalf("post-lowered-budget callbacks = %d, want %d",
			callbackCalls, callbackAfterSecond+len(recovered.RTPPackets))
	}
}

func TestEncoderSetBitrateResetsFrameBudgetCreditBeforeNextFrame(t *testing.T) {
	frame := patternedI420EncoderFrame(16, 16)
	frame.PTS = 0

	probeCfg := goh264.DefaultEncoderConfig(16, 16)
	probeCfg.DeblockMode = goh264.EncoderDeblockDisabled
	probeCfg.FrameDrop = goh264.EncoderFrameDropDisabled
	probe, err := goh264.NewEncoder(probeCfg)
	if err != nil {
		t.Fatalf("NewEncoder probe: %v", err)
	}
	probeIDR, err := probe.Encode(frame)
	if err != nil {
		t.Fatalf("probe IDR: %v", err)
	}
	probePSkip, err := probe.Encode(frame)
	if err != nil {
		t.Fatalf("probe P-skip: %v", err)
	}
	idrBytes := len(probeIDR.Data)
	pskipBytes := len(probePSkip.Data)
	if idrBytes == 0 || pskipBytes < 2 {
		t.Fatalf("probe sizes IDR/P-skip = %d/%d, want IDR > 0 and P-skip >= 2 bytes",
			idrBytes, pskipBytes)
	}

	cfg := goh264.DefaultEncoderConfig(16, 16)
	cfg.DeblockMode = goh264.EncoderDeblockDisabled
	cfg.VBVBufferSize = 0
	highBitrate := idrBytes * 8 * cfg.FrameRateNum / cfg.FrameRateDen
	cfg.TargetBitrate = highBitrate
	cfg.MaxBitrate = highBitrate
	enc, err := goh264.NewEncoder(cfg)
	if err != nil {
		t.Fatalf("NewEncoder: %v", err)
	}

	var callbackCalls int
	enc.SetRTPPacketCallback(func(goh264.EncoderRTPPacket, goh264.EncoderRTPPacketMetadata) {
		callbackCalls++
	})

	first, err := enc.Encode(frame)
	if err != nil {
		t.Fatalf("Encode high-frame-budget IDR: %v", err)
	}
	if first.Dropped || !first.IDR || len(first.Data) != idrBytes {
		t.Fatalf("high-frame-budget IDR dropped=%v idr=%v data=%d, want transmitted IDR size %d",
			first.Dropped, first.IDR, len(first.Data), idrBytes)
	}
	firstPacketCount := len(first.RTPPackets)
	if callbackCalls != firstPacketCount {
		t.Fatalf("high-frame-budget IDR callbacks = %d, want %d", callbackCalls, firstPacketCount)
	}

	second, err := enc.Encode(frame)
	if err != nil {
		t.Fatalf("Encode high-frame-budget P-skip: %v", err)
	}
	if second.Dropped || second.IDR || len(second.Data) != pskipBytes {
		t.Fatalf("high-frame-budget P-skip dropped=%v idr=%v data=%d, want transmitted P-skip size %d",
			second.Dropped, second.IDR, len(second.Data), pskipBytes)
	}
	callbackAfterSecond := firstPacketCount + len(second.RTPPackets)
	if callbackCalls != callbackAfterSecond {
		t.Fatalf("high-frame-budget P-skip callbacks = %d, want %d", callbackCalls, callbackAfterSecond)
	}
	assertRTPPacketMetadata(t, second.RTPPackets, cfg.RTPPayloadType, cfg.RTPSSRC, uint16(firstPacketCount))

	lowFrameBudgetBytes := pskipBytes - 1
	lowBitrate := lowFrameBudgetBytes * 8 * cfg.FrameRateNum / cfg.FrameRateDen
	if err := enc.SetBitrate(lowBitrate, lowBitrate); err != nil {
		t.Fatalf("SetBitrate lowered frame budget: %v", err)
	}
	if got := enc.Config(); got.TargetBitrate != lowBitrate || got.MaxBitrate != lowBitrate {
		t.Fatalf("lowered bitrate config = %d/%d, want %d/%d",
			got.TargetBitrate, got.MaxBitrate, lowBitrate, lowBitrate)
	}
	dropped, err := enc.Encode(frame)
	if err != nil {
		t.Fatalf("Encode after lowered SetBitrate: %v", err)
	}
	if !dropped.Dropped || len(dropped.Data) != 0 || len(dropped.NALUnits) != 0 || len(dropped.RTPPackets) != 0 {
		t.Fatalf("lowered-SetBitrate frame = %+v, want dropped metadata without output", dropped)
	}
	if dropped.RTPTime != second.RTPTime+cfg.RTPTimestampIncrement {
		t.Fatalf("lowered-SetBitrate RTP time = %d, want %d",
			dropped.RTPTime, second.RTPTime+cfg.RTPTimestampIncrement)
	}
	if callbackCalls != callbackAfterSecond {
		t.Fatalf("lowered-SetBitrate callbacks = %d, want still %d", callbackCalls, callbackAfterSecond)
	}

	if err := enc.SetBitrate(highBitrate, highBitrate); err != nil {
		t.Fatalf("SetBitrate raised frame budget: %v", err)
	}
	recovered, err := enc.Encode(frame)
	if err != nil {
		t.Fatalf("Encode after raised SetBitrate: %v", err)
	}
	if recovered.Dropped || recovered.IDR || len(recovered.Data) != pskipBytes {
		t.Fatalf("post-SetBitrate output dropped=%v idr=%v data=%d, want transmitted P-skip size %d",
			recovered.Dropped, recovered.IDR, len(recovered.Data), pskipBytes)
	}
	if recovered.RTPTime != dropped.RTPTime+cfg.RTPTimestampIncrement {
		t.Fatalf("post-SetBitrate RTP time = %d, want %d",
			recovered.RTPTime, dropped.RTPTime+cfg.RTPTimestampIncrement)
	}
	assertEncoderNALTypes(t, recovered.NALUnits, []uint8{1})
	assertEncoderVCLFrameNums(t,
		append(append(append([]byte(nil), first.Data...), second.Data...), recovered.Data...),
		[]uint8{5, 1, 1},
		[]uint32{0, 1, 2},
	)
	assertRTPPacketMetadata(t, recovered.RTPPackets, cfg.RTPPayloadType, cfg.RTPSSRC, uint16(callbackAfterSecond))
	if callbackCalls != callbackAfterSecond+len(recovered.RTPPackets) {
		t.Fatalf("post-SetBitrate callbacks = %d, want %d",
			callbackCalls, callbackAfterSecond+len(recovered.RTPPackets))
	}
}

func TestEncoderSetFrameRateResetsFrameBudgetAndRTPIncrement(t *testing.T) {
	frame := patternedI420EncoderFrame(16, 16)
	frame.PTS = 0
	frame.Duration = 0

	probeCfg := goh264.DefaultEncoderConfig(16, 16)
	probeCfg.DeblockMode = goh264.EncoderDeblockDisabled
	probeCfg.FrameDrop = goh264.EncoderFrameDropDisabled
	probe, err := goh264.NewEncoder(probeCfg)
	if err != nil {
		t.Fatalf("NewEncoder probe: %v", err)
	}
	probeIDR, err := probe.Encode(frame)
	if err != nil {
		t.Fatalf("probe IDR: %v", err)
	}
	probePSkip, err := probe.Encode(frame)
	if err != nil {
		t.Fatalf("probe P-skip: %v", err)
	}
	idrBytes := len(probeIDR.Data)
	pskipBytes := len(probePSkip.Data)
	if idrBytes <= pskipBytes || pskipBytes < 2 {
		t.Fatalf("probe sizes IDR/P-skip = %d/%d, want IDR > P-skip >= 2 bytes",
			idrBytes, pskipBytes)
	}

	cfg := goh264.DefaultEncoderConfig(16, 16)
	cfg.DeblockMode = goh264.EncoderDeblockDisabled
	cfg.FrameRateNum = 1
	cfg.FrameRateDen = 1
	cfg.RTPTimestampIncrement = 0
	cfg.VBVBufferSize = 0
	cfg.TargetBitrate = idrBytes * 8
	cfg.MaxBitrate = cfg.TargetBitrate
	enc, err := goh264.NewEncoder(cfg)
	if err != nil {
		t.Fatalf("NewEncoder: %v", err)
	}
	initialCfg := enc.Config()
	if initialCfg.RTPTimestampIncrement != 90_000 {
		t.Fatalf("initial RTP timestamp increment = %d, want 90000", initialCfg.RTPTimestampIncrement)
	}

	var callbackCalls int
	enc.SetRTPPacketCallback(func(goh264.EncoderRTPPacket, goh264.EncoderRTPPacketMetadata) {
		callbackCalls++
	})

	first, err := enc.Encode(frame)
	if err != nil {
		t.Fatalf("Encode 1fps IDR: %v", err)
	}
	if first.Dropped || !first.IDR || len(first.Data) != idrBytes {
		t.Fatalf("1fps IDR dropped=%v idr=%v data=%d, want transmitted IDR size %d",
			first.Dropped, first.IDR, len(first.Data), idrBytes)
	}
	firstPacketCount := len(first.RTPPackets)
	if callbackCalls != firstPacketCount {
		t.Fatalf("1fps IDR callbacks = %d, want %d", callbackCalls, firstPacketCount)
	}

	second, err := enc.Encode(frame)
	if err != nil {
		t.Fatalf("Encode 1fps P-skip: %v", err)
	}
	if second.Dropped || second.IDR || len(second.Data) != pskipBytes {
		t.Fatalf("1fps P-skip dropped=%v idr=%v data=%d, want transmitted P-skip size %d",
			second.Dropped, second.IDR, len(second.Data), pskipBytes)
	}
	if second.RTPTime != first.RTPTime+initialCfg.RTPTimestampIncrement {
		t.Fatalf("1fps P-skip RTP time = %d, want %d",
			second.RTPTime, first.RTPTime+initialCfg.RTPTimestampIncrement)
	}
	callbackAfterSecond := firstPacketCount + len(second.RTPPackets)
	if callbackCalls != callbackAfterSecond {
		t.Fatalf("1fps P-skip callbacks = %d, want %d", callbackCalls, callbackAfterSecond)
	}
	assertRTPPacketMetadata(t, second.RTPPackets, initialCfg.RTPPayloadType, initialCfg.RTPSSRC, uint16(firstPacketCount))

	maxFastFrameBits := (pskipBytes - 1) * 8
	fastFrameRateNum := (idrBytes*8 + maxFastFrameBits - 1) / maxFastFrameBits
	if fastFrameRateNum <= 1 || fastFrameRateNum > 90_000 {
		t.Fatalf("derived fast frame rate = %d for IDR/P-skip sizes %d/%d, want 2..90000",
			fastFrameRateNum, idrBytes, pskipBytes)
	}
	if err := enc.SetFrameRate(fastFrameRateNum, 1); err != nil {
		t.Fatalf("SetFrameRate fast budget: %v", err)
	}
	fastCfg := enc.Config()
	fastIncrement := uint32(90_000 / fastFrameRateNum)
	if fastCfg.FrameRateNum != fastFrameRateNum || fastCfg.FrameRateDen != 1 ||
		fastCfg.RTPTimestampIncrement != fastIncrement {
		t.Fatalf("fast frame-rate config = %d/%d rtp=%d, want %d/1 rtp=%d",
			fastCfg.FrameRateNum, fastCfg.FrameRateDen, fastCfg.RTPTimestampIncrement,
			fastFrameRateNum, fastIncrement)
	}
	dropped, err := enc.Encode(frame)
	if err != nil {
		t.Fatalf("Encode after fast SetFrameRate: %v", err)
	}
	if !dropped.Dropped || len(dropped.Data) != 0 || len(dropped.NALUnits) != 0 || len(dropped.RTPPackets) != 0 {
		t.Fatalf("fast-frame-rate output = %+v, want dropped metadata without output", dropped)
	}
	if dropped.RTPTime != second.RTPTime+initialCfg.RTPTimestampIncrement {
		t.Fatalf("fast-frame-rate dropped RTP time = %d, want old next timestamp %d",
			dropped.RTPTime, second.RTPTime+initialCfg.RTPTimestampIncrement)
	}
	if callbackCalls != callbackAfterSecond {
		t.Fatalf("fast-frame-rate dropped callbacks = %d, want still %d", callbackCalls, callbackAfterSecond)
	}

	if err := enc.SetFrameRate(1, 1); err != nil {
		t.Fatalf("SetFrameRate restored budget: %v", err)
	}
	restoredCfg := enc.Config()
	if restoredCfg.RTPTimestampIncrement != initialCfg.RTPTimestampIncrement {
		t.Fatalf("restored RTP timestamp increment = %d, want %d",
			restoredCfg.RTPTimestampIncrement, initialCfg.RTPTimestampIncrement)
	}
	recovered, err := enc.Encode(frame)
	if err != nil {
		t.Fatalf("Encode after restored SetFrameRate: %v", err)
	}
	if recovered.Dropped || recovered.IDR || len(recovered.Data) != pskipBytes {
		t.Fatalf("post-frame-rate output dropped=%v idr=%v data=%d, want transmitted P-skip size %d",
			recovered.Dropped, recovered.IDR, len(recovered.Data), pskipBytes)
	}
	if recovered.RTPTime != dropped.RTPTime+fastIncrement {
		t.Fatalf("post-frame-rate RTP time = %d, want fast increment from dropped frame %d",
			recovered.RTPTime, dropped.RTPTime+fastIncrement)
	}
	assertEncoderNALTypes(t, recovered.NALUnits, []uint8{1})
	assertEncoderVCLFrameNums(t,
		append(append(append([]byte(nil), first.Data...), second.Data...), recovered.Data...),
		[]uint8{5, 1, 1},
		[]uint32{0, 1, 2},
	)
	assertRTPPacketMetadata(t, recovered.RTPPackets, restoredCfg.RTPPayloadType, restoredCfg.RTPSSRC, uint16(callbackAfterSecond))
	callbackAfterRecovered := callbackAfterSecond + len(recovered.RTPPackets)
	if callbackCalls != callbackAfterRecovered {
		t.Fatalf("post-frame-rate callbacks = %d, want %d", callbackCalls, callbackAfterRecovered)
	}

	final, err := enc.Encode(frame)
	if err != nil {
		t.Fatalf("Encode after restored frame-rate recovery: %v", err)
	}
	if final.Dropped || final.IDR || len(final.Data) != pskipBytes {
		t.Fatalf("restored-frame-rate output dropped=%v idr=%v data=%d, want transmitted P-skip size %d",
			final.Dropped, final.IDR, len(final.Data), pskipBytes)
	}
	if final.RTPTime != recovered.RTPTime+restoredCfg.RTPTimestampIncrement {
		t.Fatalf("restored-frame-rate RTP time = %d, want %d",
			final.RTPTime, recovered.RTPTime+restoredCfg.RTPTimestampIncrement)
	}
	assertEncoderNALTypes(t, final.NALUnits, []uint8{1})
	assertEncoderVCLFrameNums(t,
		append(append(append(append([]byte(nil), first.Data...), second.Data...), recovered.Data...), final.Data...),
		[]uint8{5, 1, 1, 1},
		[]uint32{0, 1, 2, 3},
	)
	assertRTPPacketMetadata(t, final.RTPPackets, restoredCfg.RTPPayloadType, restoredCfg.RTPSSRC, uint16(callbackAfterRecovered))
	if callbackCalls != callbackAfterRecovered+len(final.RTPPackets) {
		t.Fatalf("restored-frame-rate callbacks = %d, want %d",
			callbackCalls, callbackAfterRecovered+len(final.RTPPackets))
	}
}

func TestEncoderFrameDropDisabledDoesNotApplyDerivedBitrateBudget(t *testing.T) {
	cfg := goh264.DefaultEncoderConfig(16, 16)
	cfg.DeblockMode = goh264.EncoderDeblockDisabled
	cfg.TargetBitrate = 1_000
	cfg.MaxBitrate = 1_000
	cfg.VBVBufferSize = 64
	cfg.FrameDrop = goh264.EncoderFrameDropDisabled
	enc, err := goh264.NewEncoder(cfg)
	if err != nil {
		t.Fatalf("NewEncoder: %v", err)
	}

	out, err := enc.Encode(patternedI420EncoderFrame(16, 16))
	if err != nil {
		t.Fatalf("Encode with disabled frame drop and low MaxBitrate: %v", err)
	}
	if out.Dropped || !out.IDR || len(out.Data) == 0 {
		t.Fatalf("disabled-drop output dropped=%v idr=%v data=%d, want transmitted IDR", out.Dropped, out.IDR, len(out.Data))
	}
}

func TestEncoderReconfigureFrameDropModeTogglesDerivedBitrateBudget(t *testing.T) {
	frame := patternedI420EncoderFrame(16, 16)
	frame.PTS = 0

	probeCfg := goh264.DefaultEncoderConfig(16, 16)
	probeCfg.DeblockMode = goh264.EncoderDeblockDisabled
	probeCfg.FrameDrop = goh264.EncoderFrameDropDisabled
	probe, err := goh264.NewEncoder(probeCfg)
	if err != nil {
		t.Fatalf("NewEncoder probe: %v", err)
	}
	probeIDR, err := probe.Encode(frame)
	if err != nil {
		t.Fatalf("probe IDR: %v", err)
	}
	probePSkip, err := probe.Encode(frame)
	if err != nil {
		t.Fatalf("probe P-skip: %v", err)
	}
	idrBytes := len(probeIDR.Data)
	pskipBytes := len(probePSkip.Data)
	if idrBytes == 0 || pskipBytes < 2 {
		t.Fatalf("probe sizes IDR/P-skip = %d/%d, want IDR > 0 and P-skip >= 2 bytes",
			idrBytes, pskipBytes)
	}

	cfg := goh264.DefaultEncoderConfig(16, 16)
	cfg.DeblockMode = goh264.EncoderDeblockDisabled
	cfg.FrameDrop = goh264.EncoderFrameDropDisabled
	lowBudgetBits := (pskipBytes - 1) * 8
	lowBitrate := lowBudgetBits * cfg.FrameRateNum / cfg.FrameRateDen
	cfg.TargetBitrate = lowBitrate
	cfg.MaxBitrate = lowBitrate
	cfg.VBVBufferSize = lowBudgetBits
	enc, err := goh264.NewEncoder(cfg)
	if err != nil {
		t.Fatalf("NewEncoder: %v", err)
	}

	var callbackCalls int
	enc.SetRTPPacketCallback(func(goh264.EncoderRTPPacket, goh264.EncoderRTPPacketMetadata) {
		callbackCalls++
	})

	first, err := enc.Encode(frame)
	if err != nil {
		t.Fatalf("Encode disabled-drop low-budget IDR: %v", err)
	}
	if first.Dropped || !first.IDR || len(first.Data) != idrBytes {
		t.Fatalf("disabled-drop low-budget output dropped=%v idr=%v data=%d, want transmitted IDR size %d",
			first.Dropped, first.IDR, len(first.Data), idrBytes)
	}
	assertEncoderNALTypes(t, first.NALUnits, []uint8{7, 8, 5})
	firstPacketCount := len(first.RTPPackets)
	if callbackCalls != firstPacketCount {
		t.Fatalf("disabled-drop callbacks = %d, want %d", callbackCalls, firstPacketCount)
	}

	if err := enc.Reconfigure(goh264.EncoderReconfigure{
		FrameDrop: goh264.EncoderFrameDropToBitrate,
	}); err != nil {
		t.Fatalf("Reconfigure frame drop to bitrate: %v", err)
	}
	if got := enc.Config(); got.FrameDrop != goh264.EncoderFrameDropToBitrate {
		t.Fatalf("frame drop mode = %v, want ToBitrate", got.FrameDrop)
	}
	if enc.PendingIDR() {
		t.Fatal("frame-drop-only reconfigure queued IDR")
	}
	dropped, err := enc.Encode(frame)
	if err != nil {
		t.Fatalf("Encode after frame-drop ToBitrate reconfigure: %v", err)
	}
	if !dropped.Dropped || len(dropped.Data) != 0 || len(dropped.NALUnits) != 0 || len(dropped.RTPPackets) != 0 {
		t.Fatalf("ToBitrate low-budget output = %+v, want dropped metadata without output", dropped)
	}
	if dropped.RTPTime != first.RTPTime+cfg.RTPTimestampIncrement {
		t.Fatalf("ToBitrate dropped RTP time = %d, want %d",
			dropped.RTPTime, first.RTPTime+cfg.RTPTimestampIncrement)
	}
	if callbackCalls != firstPacketCount {
		t.Fatalf("ToBitrate dropped callbacks = %d, want still %d", callbackCalls, firstPacketCount)
	}

	if err := enc.Reconfigure(goh264.EncoderReconfigure{
		FrameDrop: goh264.EncoderFrameDropDisabled,
	}); err != nil {
		t.Fatalf("Reconfigure frame drop disabled: %v", err)
	}
	if got := enc.Config(); got.FrameDrop != goh264.EncoderFrameDropDisabled {
		t.Fatalf("frame drop mode = %v, want Disabled", got.FrameDrop)
	}
	recovered, err := enc.Encode(frame)
	if err != nil {
		t.Fatalf("Encode after frame-drop disabled reconfigure: %v", err)
	}
	if recovered.Dropped || recovered.IDR || len(recovered.Data) != pskipBytes {
		t.Fatalf("disabled-drop recovered output dropped=%v idr=%v data=%d, want transmitted P-skip size %d",
			recovered.Dropped, recovered.IDR, len(recovered.Data), pskipBytes)
	}
	if recovered.RTPTime != dropped.RTPTime+cfg.RTPTimestampIncrement {
		t.Fatalf("disabled-drop recovered RTP time = %d, want %d",
			recovered.RTPTime, dropped.RTPTime+cfg.RTPTimestampIncrement)
	}
	assertEncoderNALTypes(t, recovered.NALUnits, []uint8{1})
	assertEncoderVCLFrameNums(t,
		append(append([]byte(nil), first.Data...), recovered.Data...),
		[]uint8{5, 1},
		[]uint32{0, 1},
	)
	assertRTPPacketMetadata(t, recovered.RTPPackets, cfg.RTPPayloadType, cfg.RTPSSRC, uint16(firstPacketCount))
	if callbackCalls != firstPacketCount+len(recovered.RTPPackets) {
		t.Fatalf("disabled-drop recovered callbacks = %d, want %d",
			callbackCalls, firstPacketCount+len(recovered.RTPPackets))
	}
}

func TestEncoderReconfigureExplicitZeroVBVDisablesCapAndResetsBudget(t *testing.T) {
	frame := patternedI420EncoderFrame(16, 16)
	frame.PTS = 0

	probeCfg := goh264.DefaultEncoderConfig(16, 16)
	probeCfg.DeblockMode = goh264.EncoderDeblockDisabled
	probeCfg.FrameDrop = goh264.EncoderFrameDropDisabled
	probe, err := goh264.NewEncoder(probeCfg)
	if err != nil {
		t.Fatalf("NewEncoder probe: %v", err)
	}
	probeIDR, err := probe.Encode(frame)
	if err != nil {
		t.Fatalf("probe IDR: %v", err)
	}
	probePSkip, err := probe.Encode(frame)
	if err != nil {
		t.Fatalf("probe P-skip: %v", err)
	}
	idrBytes := len(probeIDR.Data)
	pskipBytes := len(probePSkip.Data)
	if idrBytes == 0 || pskipBytes < 2 {
		t.Fatalf("probe sizes IDR/P-skip = %d/%d, want IDR > 0 and P-skip >= 2 bytes",
			idrBytes, pskipBytes)
	}

	cfg := goh264.DefaultEncoderConfig(16, 16)
	cfg.DeblockMode = goh264.EncoderDeblockDisabled
	cfg.FrameDrop = goh264.EncoderFrameDropDisabled
	cfg.TargetBitrate = pskipBytes * 8 * cfg.FrameRateNum / cfg.FrameRateDen
	cfg.MaxBitrate = cfg.TargetBitrate
	cfg.VBVBufferSize = (pskipBytes - 1) * 8
	enc, err := goh264.NewEncoder(cfg)
	if err != nil {
		t.Fatalf("NewEncoder: %v", err)
	}

	var callbackCalls int
	enc.SetRTPPacketCallback(func(goh264.EncoderRTPPacket, goh264.EncoderRTPPacketMetadata) {
		callbackCalls++
	})

	first, err := enc.Encode(frame)
	if err != nil {
		t.Fatalf("Encode disabled-drop IDR: %v", err)
	}
	if first.Dropped || !first.IDR || len(first.Data) != idrBytes {
		t.Fatalf("disabled-drop IDR dropped=%v idr=%v data=%d, want transmitted IDR size %d",
			first.Dropped, first.IDR, len(first.Data), idrBytes)
	}
	firstPacketCount := len(first.RTPPackets)
	if callbackCalls != firstPacketCount {
		t.Fatalf("disabled-drop callbacks = %d, want %d", callbackCalls, firstPacketCount)
	}

	if err := enc.Reconfigure(goh264.EncoderReconfigure{
		FrameDrop: goh264.EncoderFrameDropToBitrate,
	}); err != nil {
		t.Fatalf("Reconfigure frame drop to bitrate: %v", err)
	}
	capped, err := enc.Encode(frame)
	if err != nil {
		t.Fatalf("Encode capped P-skip: %v", err)
	}
	if !capped.Dropped || len(capped.Data) != 0 || len(capped.NALUnits) != 0 || len(capped.RTPPackets) != 0 {
		t.Fatalf("capped P-skip output = %+v, want dropped metadata without output", capped)
	}
	if callbackCalls != firstPacketCount {
		t.Fatalf("capped P-skip callbacks = %d, want still %d", callbackCalls, firstPacketCount)
	}

	zeroVBV := 0
	if err := enc.Reconfigure(goh264.EncoderReconfigure{
		VBVBufferSize: &zeroVBV,
	}); err != nil {
		t.Fatalf("Reconfigure explicit zero VBV: %v", err)
	}
	if got := enc.Config(); got.VBVBufferSize != 0 || got.FrameDrop != goh264.EncoderFrameDropToBitrate {
		t.Fatalf("post-zero-VBV config = %+v, want VBVBufferSize=0 and ToBitrate", got)
	}
	recovered, err := enc.Encode(frame)
	if err != nil {
		t.Fatalf("Encode after zero VBV: %v", err)
	}
	if recovered.Dropped || recovered.IDR || len(recovered.Data) != pskipBytes {
		t.Fatalf("zero-VBV recovered output dropped=%v idr=%v data=%d, want transmitted P-skip size %d",
			recovered.Dropped, recovered.IDR, len(recovered.Data), pskipBytes)
	}
	if recovered.RTPTime != capped.RTPTime+cfg.RTPTimestampIncrement {
		t.Fatalf("zero-VBV recovered RTP time = %d, want %d",
			recovered.RTPTime, capped.RTPTime+cfg.RTPTimestampIncrement)
	}
	assertEncoderNALTypes(t, recovered.NALUnits, []uint8{1})
	assertEncoderVCLFrameNums(t,
		append(append([]byte(nil), first.Data...), recovered.Data...),
		[]uint8{5, 1},
		[]uint32{0, 1},
	)
	assertRTPPacketMetadata(t, recovered.RTPPackets, cfg.RTPPayloadType, cfg.RTPSSRC, uint16(firstPacketCount))
	if callbackCalls != firstPacketCount+len(recovered.RTPPackets) {
		t.Fatalf("zero-VBV recovered callbacks = %d, want %d",
			callbackCalls, firstPacketCount+len(recovered.RTPPackets))
	}
}

func TestEncoderFrameDropLateDoesNotApplyDerivedBitrateBudgetAcrossReconfigure(t *testing.T) {
	frame := patternedI420EncoderFrame(16, 16)
	frame.PTS = 0

	probeCfg := goh264.DefaultEncoderConfig(16, 16)
	probeCfg.DeblockMode = goh264.EncoderDeblockDisabled
	probeCfg.FrameDrop = goh264.EncoderFrameDropDisabled
	probe, err := goh264.NewEncoder(probeCfg)
	if err != nil {
		t.Fatalf("NewEncoder probe: %v", err)
	}
	probeIDR, err := probe.Encode(frame)
	if err != nil {
		t.Fatalf("probe IDR: %v", err)
	}
	probePSkip, err := probe.Encode(frame)
	if err != nil {
		t.Fatalf("probe P-skip: %v", err)
	}
	idrBytes := len(probeIDR.Data)
	pskipBytes := len(probePSkip.Data)
	if idrBytes == 0 || pskipBytes < 2 {
		t.Fatalf("probe sizes IDR/P-skip = %d/%d, want IDR > 0 and P-skip >= 2 bytes",
			idrBytes, pskipBytes)
	}

	cfg := goh264.DefaultEncoderConfig(16, 16)
	cfg.DeblockMode = goh264.EncoderDeblockDisabled
	cfg.FrameDrop = goh264.EncoderFrameDropLate
	cfg.MaxEncodeTimeUS = 10_000_000
	lowBudgetBits := (pskipBytes - 1) * 8
	lowBitrate := lowBudgetBits * cfg.FrameRateNum / cfg.FrameRateDen
	cfg.TargetBitrate = lowBitrate
	cfg.MaxBitrate = lowBitrate
	cfg.VBVBufferSize = lowBudgetBits
	enc, err := goh264.NewEncoder(cfg)
	if err != nil {
		t.Fatalf("NewEncoder: %v", err)
	}

	var callbackCalls int
	enc.SetRTPPacketCallback(func(goh264.EncoderRTPPacket, goh264.EncoderRTPPacketMetadata) {
		callbackCalls++
	})

	first, err := enc.Encode(frame)
	if err != nil {
		t.Fatalf("Encode late-drop low-budget IDR: %v", err)
	}
	if first.Dropped || !first.IDR || len(first.Data) != idrBytes {
		t.Fatalf("late-drop low-budget output dropped=%v idr=%v data=%d, want transmitted IDR size %d",
			first.Dropped, first.IDR, len(first.Data), idrBytes)
	}
	assertEncoderNALTypes(t, first.NALUnits, []uint8{7, 8, 5})
	firstPacketCount := len(first.RTPPackets)
	if callbackCalls != firstPacketCount {
		t.Fatalf("late-drop callbacks = %d, want %d", callbackCalls, firstPacketCount)
	}

	if err := enc.Reconfigure(goh264.EncoderReconfigure{
		FrameDrop: goh264.EncoderFrameDropToBitrate,
	}); err != nil {
		t.Fatalf("Reconfigure frame drop to bitrate: %v", err)
	}
	if got := enc.Config(); got.FrameDrop != goh264.EncoderFrameDropToBitrate {
		t.Fatalf("frame drop mode = %v, want ToBitrate", got.FrameDrop)
	}
	dropped, err := enc.Encode(frame)
	if err != nil {
		t.Fatalf("Encode after frame-drop ToBitrate reconfigure: %v", err)
	}
	if !dropped.Dropped || len(dropped.Data) != 0 || len(dropped.NALUnits) != 0 || len(dropped.RTPPackets) != 0 {
		t.Fatalf("ToBitrate low-budget output = %+v, want dropped metadata without output", dropped)
	}
	if dropped.RTPTime != first.RTPTime+cfg.RTPTimestampIncrement {
		t.Fatalf("ToBitrate dropped RTP time = %d, want %d",
			dropped.RTPTime, first.RTPTime+cfg.RTPTimestampIncrement)
	}
	if callbackCalls != firstPacketCount {
		t.Fatalf("ToBitrate dropped callbacks = %d, want still %d", callbackCalls, firstPacketCount)
	}

	if err := enc.Reconfigure(goh264.EncoderReconfigure{
		FrameDrop:       goh264.EncoderFrameDropLate,
		MaxEncodeTimeUS: 10_000_000,
	}); err != nil {
		t.Fatalf("Reconfigure frame drop late: %v", err)
	}
	if got := enc.Config(); got.FrameDrop != goh264.EncoderFrameDropLate || got.MaxEncodeTimeUS != 10_000_000 {
		t.Fatalf("late frame drop config = mode %v max-time %d, want Late/10000000",
			got.FrameDrop, got.MaxEncodeTimeUS)
	}
	recovered, err := enc.Encode(frame)
	if err != nil {
		t.Fatalf("Encode after frame-drop late reconfigure: %v", err)
	}
	if recovered.Dropped || recovered.IDR || len(recovered.Data) != pskipBytes {
		t.Fatalf("late-drop recovered output dropped=%v idr=%v data=%d, want transmitted P-skip size %d",
			recovered.Dropped, recovered.IDR, len(recovered.Data), pskipBytes)
	}
	if recovered.RTPTime != dropped.RTPTime+cfg.RTPTimestampIncrement {
		t.Fatalf("late-drop recovered RTP time = %d, want %d",
			recovered.RTPTime, dropped.RTPTime+cfg.RTPTimestampIncrement)
	}
	assertEncoderNALTypes(t, recovered.NALUnits, []uint8{1})
	assertEncoderVCLFrameNums(t,
		append(append([]byte(nil), first.Data...), recovered.Data...),
		[]uint8{5, 1},
		[]uint32{0, 1},
	)
	assertRTPPacketMetadata(t, recovered.RTPPackets, cfg.RTPPayloadType, cfg.RTPSSRC, uint16(firstPacketCount))
	if callbackCalls != firstPacketCount+len(recovered.RTPPackets) {
		t.Fatalf("late-drop recovered callbacks = %d, want %d",
			callbackCalls, firstPacketCount+len(recovered.RTPPackets))
	}
}

func TestEncoderConstantQPDoesNotApplyDerivedBitrateBudgetAcrossReconfigure(t *testing.T) {
	cfg := goh264.DefaultEncoderConfig(16, 16)
	cfg.DeblockMode = goh264.EncoderDeblockDisabled
	cfg.RateControl = goh264.EncoderRateControlConstantQP
	cfg.TargetBitrate = 1_000
	cfg.MaxBitrate = 1_000
	cfg.VBVBufferSize = 56
	enc, err := goh264.NewEncoder(cfg)
	if err != nil {
		t.Fatalf("NewEncoder: %v", err)
	}

	var callbackCalls int
	enc.SetRTPPacketCallback(func(goh264.EncoderRTPPacket, goh264.EncoderRTPPacketMetadata) {
		callbackCalls++
	})

	frame := patternedI420EncoderFrame(16, 16)
	frame.PTS = int64(cfg.RTPTimestampIncrement)
	first, err := enc.Encode(frame)
	if err != nil {
		t.Fatalf("Encode ConstantQP low-budget IDR: %v", err)
	}
	if first.Dropped || !first.IDR || len(first.Data) == 0 || len(first.RTPPackets) == 0 {
		t.Fatalf("ConstantQP low-budget output dropped=%v idr=%v data=%d rtp=%d, want transmitted IDR",
			first.Dropped, first.IDR, len(first.Data), len(first.RTPPackets))
	}
	assertEncoderNALTypes(t, first.NALUnits, []uint8{7, 8, 5})
	firstPacketCount := len(first.RTPPackets)
	if callbackCalls != firstPacketCount {
		t.Fatalf("ConstantQP callbacks = %d, want %d", callbackCalls, firstPacketCount)
	}

	if err := enc.Reconfigure(goh264.EncoderReconfigure{
		RateControl: goh264.EncoderRateControlCBR,
	}); err != nil {
		t.Fatalf("Reconfigure ConstantQP to CBR: %v", err)
	}
	if enc.PendingIDR() {
		t.Fatal("rate-control-only reconfigure queued IDR")
	}
	cbrFrame := frame
	cbrFrame.PTS = frame.PTS + int64(cfg.RTPTimestampIncrement)
	cbrDropped, err := enc.Encode(cbrFrame)
	if err != nil {
		t.Fatalf("Encode low-budget CBR P-skip: %v", err)
	}
	if !cbrDropped.Dropped || len(cbrDropped.Data) != 0 || len(cbrDropped.NALUnits) != 0 || len(cbrDropped.RTPPackets) != 0 {
		t.Fatalf("low-budget CBR output = %+v, want dropped metadata without output", cbrDropped)
	}
	if cbrDropped.RTPTime != first.RTPTime+cfg.RTPTimestampIncrement {
		t.Fatalf("low-budget CBR RTP time = %d, want %d", cbrDropped.RTPTime, first.RTPTime+cfg.RTPTimestampIncrement)
	}
	if callbackCalls != firstPacketCount {
		t.Fatalf("low-budget CBR callbacks = %d, want still %d", callbackCalls, firstPacketCount)
	}

	if err := enc.Reconfigure(goh264.EncoderReconfigure{
		RateControl: goh264.EncoderRateControlConstantQP,
	}); err != nil {
		t.Fatalf("Reconfigure CBR to ConstantQP: %v", err)
	}
	recoveredFrame := frame
	recoveredFrame.PTS = cbrFrame.PTS + int64(cfg.RTPTimestampIncrement)
	recovered, err := enc.Encode(recoveredFrame)
	if err != nil {
		t.Fatalf("Encode ConstantQP after CBR drop: %v", err)
	}
	if recovered.Dropped || recovered.IDR || len(recovered.Data) == 0 || len(recovered.RTPPackets) == 0 {
		t.Fatalf("post-CBR ConstantQP output dropped=%v idr=%v data=%d rtp=%d, want transmitted P-skip",
			recovered.Dropped, recovered.IDR, len(recovered.Data), len(recovered.RTPPackets))
	}
	if recovered.RTPTime != cbrDropped.RTPTime+cfg.RTPTimestampIncrement {
		t.Fatalf("post-CBR ConstantQP RTP time = %d, want %d",
			recovered.RTPTime, cbrDropped.RTPTime+cfg.RTPTimestampIncrement)
	}
	assertEncoderNALTypes(t, recovered.NALUnits, []uint8{1})
	assertEncoderVCLFrameNums(t,
		append(append([]byte(nil), first.Data...), recovered.Data...),
		[]uint8{5, 1},
		[]uint32{0, 1},
	)
	assertRTPPacketMetadata(t, recovered.RTPPackets, cfg.RTPPayloadType, cfg.RTPSSRC, uint16(firstPacketCount))
	if callbackCalls != firstPacketCount+len(recovered.RTPPackets) {
		t.Fatalf("post-CBR ConstantQP callbacks = %d, want %d",
			callbackCalls, firstPacketCount+len(recovered.RTPPackets))
	}
}

func TestEncoderFrameDropLateDropsOverBudgetFrameWithoutAdvancingReferenceOrPacketState(t *testing.T) {
	cfg := goh264.DefaultEncoderConfig(128, 128)
	cfg.DeblockMode = goh264.EncoderDeblockDisabled
	cfg.FrameDrop = goh264.EncoderFrameDropLate
	cfg.MaxEncodeTimeUS = 1
	enc, err := goh264.NewEncoder(cfg)
	if err != nil {
		t.Fatalf("NewEncoder: %v", err)
	}

	var callbackCalls int
	enc.SetRTPPacketCallback(func(goh264.EncoderRTPPacket, goh264.EncoderRTPPacketMetadata) {
		callbackCalls++
	})
	frame := patternedI420EncoderFrame(128, 128)
	frame.PTS = 0
	dropped, err := enc.Encode(frame)
	if err != nil {
		t.Fatalf("Encode late-drop frame: %v", err)
	}
	if !dropped.Dropped || len(dropped.Data) != 0 || len(dropped.NALUnits) != 0 || len(dropped.RTPPackets) != 0 {
		t.Fatalf("late dropped frame = %+v, want dropped metadata without output", dropped)
	}
	if dropped.RTPTime != 0 {
		t.Fatalf("late dropped RTP time = %d, want 0", dropped.RTPTime)
	}
	if callbackCalls != 0 {
		t.Fatalf("late dropped frame invoked callback count %d, want 0", callbackCalls)
	}

	if err := enc.Reconfigure(goh264.EncoderReconfigure{MaxEncodeTimeUS: 10_000_000}); err != nil {
		t.Fatalf("raise MaxEncodeTimeUS: %v", err)
	}
	out, err := enc.Encode(frame)
	if err != nil {
		t.Fatalf("Encode after late dropped frame: %v", err)
	}
	if out.Dropped || !out.IDR {
		t.Fatalf("post-late-drop frame dropped=%v idr=%v, want first transmitted IDR", out.Dropped, out.IDR)
	}
	if out.RTPTime != cfg.RTPTimestampIncrement {
		t.Fatalf("post-late-drop RTP time = %d, want %d", out.RTPTime, cfg.RTPTimestampIncrement)
	}
	assertEncoderNALTypes(t, out.NALUnits, []uint8{7, 8, 5})
	assertEncoderVCLFrameNums(t, out.Data, []uint8{5}, []uint32{0})
	assertRTPPacketMetadata(t, out.RTPPackets, cfg.RTPPayloadType, cfg.RTPSSRC, 0)
	if callbackCalls != len(out.RTPPackets) {
		t.Fatalf("post-late-drop callbacks = %d, want transmitted packet count %d", callbackCalls, len(out.RTPPackets))
	}

	firstPacketCount := len(out.RTPPackets)
	if err := enc.Reconfigure(goh264.EncoderReconfigure{MaxEncodeTimeUS: 1}); err != nil {
		t.Fatalf("lower MaxEncodeTimeUS: %v", err)
	}
	lateChangedFrame := patternedI420EncoderFrame(128, 128)
	lateChangedFrame.Y[0] ^= 0x4c
	lateChangedFrame.PTS = 0
	dropped, err = enc.Encode(lateChangedFrame)
	if err != nil {
		t.Fatalf("Encode late-drop changed frame: %v", err)
	}
	if !dropped.Dropped || len(dropped.Data) != 0 || len(dropped.NALUnits) != 0 || len(dropped.RTPPackets) != 0 {
		t.Fatalf("late dropped changed frame = %+v, want dropped metadata without output", dropped)
	}
	if dropped.RTPTime != out.RTPTime+cfg.RTPTimestampIncrement {
		t.Fatalf("late dropped changed RTP time = %d, want %d", dropped.RTPTime, out.RTPTime+cfg.RTPTimestampIncrement)
	}
	if callbackCalls != firstPacketCount {
		t.Fatalf("late dropped changed frame invoked callback count %d, want still %d", callbackCalls, firstPacketCount)
	}

	if err := enc.Reconfigure(goh264.EncoderReconfigure{MaxEncodeTimeUS: 10_000_000}); err != nil {
		t.Fatalf("raise MaxEncodeTimeUS for P-skip: %v", err)
	}
	pskipFrame := frame
	pskipFrame.PTS = 0
	pskip, err := enc.Encode(pskipFrame)
	if err != nil {
		t.Fatalf("Encode after late dropped changed frame: %v", err)
	}
	if pskip.Dropped || pskip.IDR {
		t.Fatalf("post-late-drop matching frame dropped=%v idr=%v, want transmitted P-skip", pskip.Dropped, pskip.IDR)
	}
	if pskip.RTPTime != dropped.RTPTime+cfg.RTPTimestampIncrement {
		t.Fatalf("post-late-drop matching RTP time = %d, want %d", pskip.RTPTime, dropped.RTPTime+cfg.RTPTimestampIncrement)
	}
	assertEncoderNALTypes(t, pskip.NALUnits, []uint8{1})
	assertEncoderVCLFrameNums(t, append(append([]byte(nil), out.Data...), pskip.Data...), []uint8{5, 1}, []uint32{0, 1})
	assertRTPPacketMetadata(t, pskip.RTPPackets, cfg.RTPPayloadType, cfg.RTPSSRC, uint16(firstPacketCount))
	if callbackCalls != firstPacketCount+len(pskip.RTPPackets) {
		t.Fatalf("post-late-drop matching callbacks = %d, want %d", callbackCalls, firstPacketCount+len(pskip.RTPPackets))
	}

	callbacksAfterPSkip := callbackCalls
	if err := enc.Reconfigure(goh264.EncoderReconfigure{MaxEncodeTimeUS: 1}); err != nil {
		t.Fatalf("lower MaxEncodeTimeUS for forced IDR: %v", err)
	}
	enc.ForceIDR()
	if !enc.PendingIDR() {
		t.Fatal("ForceIDR before late drop did not queue IDR")
	}
	forcedDropped, err := enc.Encode(frame)
	if err != nil {
		t.Fatalf("Encode late-drop forced IDR: %v", err)
	}
	if !forcedDropped.Dropped || len(forcedDropped.Data) != 0 || len(forcedDropped.NALUnits) != 0 || len(forcedDropped.RTPPackets) != 0 {
		t.Fatalf("late dropped forced IDR = %+v, want dropped metadata without output", forcedDropped)
	}
	if forcedDropped.RTPTime != pskip.RTPTime+cfg.RTPTimestampIncrement {
		t.Fatalf("late dropped forced IDR RTP time = %d, want %d", forcedDropped.RTPTime, pskip.RTPTime+cfg.RTPTimestampIncrement)
	}
	if !enc.PendingIDR() {
		t.Fatal("late dropped forced IDR consumed pending IDR")
	}
	if callbackCalls != callbacksAfterPSkip {
		t.Fatalf("late dropped forced IDR callbacks = %d, want still %d", callbackCalls, callbacksAfterPSkip)
	}

	if err := enc.Reconfigure(goh264.EncoderReconfigure{MaxEncodeTimeUS: 10_000_000}); err != nil {
		t.Fatalf("raise MaxEncodeTimeUS for forced IDR: %v", err)
	}
	forced, err := enc.Encode(frame)
	if err != nil {
		t.Fatalf("Encode after late dropped forced IDR: %v", err)
	}
	if forced.Dropped || !forced.IDR || enc.PendingIDR() {
		t.Fatalf("post-late-drop forced output dropped=%v idr=%v pending=%v, want transmitted IDR",
			forced.Dropped, forced.IDR, enc.PendingIDR())
	}
	if forced.RTPTime != forcedDropped.RTPTime+cfg.RTPTimestampIncrement {
		t.Fatalf("post-late-drop forced RTP time = %d, want %d", forced.RTPTime, forcedDropped.RTPTime+cfg.RTPTimestampIncrement)
	}
	assertEncoderNALTypes(t, forced.NALUnits, []uint8{7, 8, 5})
	assertRTPPacketMetadata(t, forced.RTPPackets, cfg.RTPPayloadType, cfg.RTPSSRC, uint16(callbacksAfterPSkip))
	if callbackCalls != callbacksAfterPSkip+len(forced.RTPPackets) {
		t.Fatalf("post-late-drop forced callbacks = %d, want %d", callbackCalls, callbacksAfterPSkip+len(forced.RTPPackets))
	}
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

func TestEncoderReconfigureRecoveryPointSEITogglesChangedPFrames(t *testing.T) {
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

	enabledFrame := patternedI420EncoderFrame(16, 16)
	enabledFrame.Y[0] ^= 0x21
	enabledFrame.PTS = firstFrame.PTS + int64(cfg.RTPTimestampIncrement)
	enabled, err := enc.Encode(enabledFrame)
	if err != nil {
		t.Fatalf("Encode recovery-enabled P IntraPCM: %v", err)
	}
	assertEncoderNALTypes(t, enabled.NALUnits, []uint8{6, 1})
	if enc.PendingIDR() {
		t.Fatal("recovery-enabled P frame queued unexpected IDR")
	}

	disableRecovery := false
	if err := enc.Reconfigure(goh264.EncoderReconfigure{RecoveryPointSEI: &disableRecovery}); err != nil {
		t.Fatalf("Reconfigure RecoveryPointSEI off: %v", err)
	}
	if got := enc.Config(); got.RecoveryPointSEI {
		t.Fatalf("RecoveryPointSEI config = %v, want false", got.RecoveryPointSEI)
	}
	if enc.PendingIDR() {
		t.Fatal("RecoveryPointSEI disable queued unexpected IDR")
	}

	disabledFrame := patternedI420EncoderFrame(16, 16)
	disabledFrame.Y[1] ^= 0x42
	disabledFrame.PTS = enabledFrame.PTS + int64(cfg.RTPTimestampIncrement)
	disabled, err := enc.Encode(disabledFrame)
	if err != nil {
		t.Fatalf("Encode recovery-disabled P IntraPCM: %v", err)
	}
	assertEncoderNALTypes(t, disabled.NALUnits, []uint8{1})

	enableRecovery := true
	if err := enc.Reconfigure(goh264.EncoderReconfigure{RecoveryPointSEI: &enableRecovery}); err != nil {
		t.Fatalf("Reconfigure RecoveryPointSEI on: %v", err)
	}
	if got := enc.Config(); !got.RecoveryPointSEI {
		t.Fatalf("RecoveryPointSEI config = %v, want true", got.RecoveryPointSEI)
	}
	if enc.PendingIDR() {
		t.Fatal("RecoveryPointSEI enable queued unexpected IDR")
	}

	reenabledFrame := patternedI420EncoderFrame(16, 16)
	reenabledFrame.Y[2] ^= 0x63
	reenabledFrame.PTS = disabledFrame.PTS + int64(cfg.RTPTimestampIncrement)
	reenabled, err := enc.Encode(reenabledFrame)
	if err != nil {
		t.Fatalf("Encode recovery-reenabled P IntraPCM: %v", err)
	}
	assertEncoderNALTypes(t, reenabled.NALUnits, []uint8{6, 1})

	stream := append(append([]byte(nil), first.Data...), enabled.Data...)
	stream = append(stream, disabled.Data...)
	stream = append(stream, reenabled.Data...)
	assertEncoderVCLFrameNums(t, stream,
		[]uint8{5, 1, 1, 1},
		[]uint32{0, 1, 2, 3},
	)

	dec := goh264.NewDecoder()
	decodedFirst, err := dec.DecodeFrames(first.Data)
	if err != nil {
		t.Fatalf("Decode first IDR: %v", err)
	}
	assertDecodedEncoderFrameBytes(t, decodedFirst, appendI420FrameBytes(nil, firstFrame))

	decodedEnabled, err := dec.DecodeFrames(enabled.Data)
	if err != nil {
		t.Fatalf("Decode recovery-enabled P IntraPCM: %v", err)
	}
	assertDecodedEncoderFrameBytes(t, decodedEnabled, appendI420FrameBytes(nil, enabledFrame))
	if !decodedEnabled[0].KeyFrame ||
		decodedEnabled[0].SideData.RecoveryPoint == nil ||
		decodedEnabled[0].SideData.RecoveryPoint.RecoveryFrameCount != 0 {
		t.Fatalf("enabled recovery side data key=%v recovery=%+v, want immediate recovery point",
			decodedEnabled[0].KeyFrame, decodedEnabled[0].SideData.RecoveryPoint)
	}

	decodedDisabled, err := dec.DecodeFrames(disabled.Data)
	if err != nil {
		t.Fatalf("Decode recovery-disabled P IntraPCM: %v", err)
	}
	assertDecodedEncoderFrameBytes(t, decodedDisabled, appendI420FrameBytes(nil, disabledFrame))
	if decodedDisabled[0].KeyFrame || decodedDisabled[0].SideData.RecoveryPoint != nil {
		t.Fatalf("disabled recovery side data key=%v recovery=%+v, want no recovery point",
			decodedDisabled[0].KeyFrame, decodedDisabled[0].SideData.RecoveryPoint)
	}

	decodedReenabled, err := dec.DecodeFrames(reenabled.Data)
	if err != nil {
		t.Fatalf("Decode recovery-reenabled P IntraPCM: %v", err)
	}
	assertDecodedEncoderFrameBytes(t, decodedReenabled, appendI420FrameBytes(nil, reenabledFrame))
	if !decodedReenabled[0].KeyFrame ||
		decodedReenabled[0].SideData.RecoveryPoint == nil ||
		decodedReenabled[0].SideData.RecoveryPoint.RecoveryFrameCount != 0 {
		t.Fatalf("reenabled recovery side data key=%v recovery=%+v, want immediate recovery point",
			decodedReenabled[0].KeyFrame, decodedReenabled[0].SideData.RecoveryPoint)
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
	var callbackPackets []goh264.EncoderRTPPacket
	var callbackMetadata []goh264.EncoderRTPPacketMetadata
	enc.SetRTPPacketCallback(func(pkt goh264.EncoderRTPPacket, meta goh264.EncoderRTPPacketMetadata) {
		callbackPackets = append(callbackPackets, pkt)
		callbackMetadata = append(callbackMetadata, meta)
	})
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
	if len(callbackPackets) != len(out.RTPPackets) || len(callbackMetadata) != len(out.RTPPackets) {
		t.Fatalf("STAP-A callbacks packets/meta = %d/%d, want RTP packet count %d",
			len(callbackPackets), len(callbackMetadata), len(out.RTPPackets))
	}
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
		callbackPkt := callbackPackets[i]
		meta := callbackMetadata[i]
		if callbackPkt.PayloadType != pkt.PayloadType ||
			callbackPkt.SequenceNumber != pkt.SequenceNumber ||
			callbackPkt.Timestamp != pkt.Timestamp ||
			callbackPkt.SSRC != pkt.SSRC ||
			callbackPkt.Marker != pkt.Marker ||
			!bytes.Equal(callbackPkt.Payload, pkt.Payload) ||
			!bytes.Equal(callbackPkt.Data, pkt.Data) {
			t.Fatalf("STAP-A callback packet[%d] = %+v, want returned RTP packet fields", i, callbackPkt)
		}
		assertEncoderRTPCallbackPacketDoesNotAliasReturned(t, callbackPkt, pkt, i)
		if meta.PacketIndex != i || meta.PacketCount != len(out.RTPPackets) ||
			meta.FramePTS != frame.PTS || meta.FrameDTS != frame.PTS ||
			meta.RTPTime != out.RTPTime || !meta.KeyFrame || !meta.IDR {
			t.Fatalf("STAP-A callback meta[%d] frame fields = %+v, want IDR timing metadata", i, meta)
		}
		if i == 0 {
			if meta.PayloadFormat != goh264.EncoderRTPPayloadSTAPA ||
				meta.NALUnitType != 24 ||
				meta.NALUnitCount != 2 ||
				!meta.ParameterSet ||
				meta.StartOfNAL || meta.EndOfNAL {
				t.Fatalf("STAP-A callback meta[0] = %+v, want SPS/PPS aggregate metadata", meta)
			}
			continue
		}
		if meta.PayloadFormat != goh264.EncoderRTPPayloadFUA &&
			meta.PayloadFormat != goh264.EncoderRTPPayloadSingleNAL {
			t.Fatalf("STAP-A VCL callback meta[%d] payload format = %v, want FU-A or single-NAL", i, meta.PayloadFormat)
		}
		if meta.NALUnitType != 5 || meta.NALUnitCount != 1 || meta.ParameterSet {
			t.Fatalf("STAP-A VCL callback meta[%d] = %+v, want IDR VCL metadata", i, meta)
		}
	}

	annexB := annexBFromEncoderRTPPackets(t, out.RTPPackets)
	decoded, err := goh264.NewDecoder().DecodeAnnexBFrames(annexB)
	if err != nil {
		t.Fatalf("DecodeAnnexBFrames reassembled STAP-A RTP: %v", err)
	}
	assertDecodedEncoderFrameBytes(t, decoded, appendI420FrameBytes(nil, frame))
}

func TestEncoderEncodeRTPMode1STAPADoesNotAggregateChangedPRecoverySEI(t *testing.T) {
	cfg := goh264.DefaultEncoderConfig(16, 16)
	cfg.STAPA = true
	cfg.RTPMaxPayloadSize = 1200
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
	})

	firstFrame := patternedI420EncoderFrame(16, 16)
	firstFrame.PTS = 10101
	first, err := enc.Encode(firstFrame)
	if err != nil {
		t.Fatalf("Encode first STAP-A IDR: %v", err)
	}
	if len(first.RTPPackets) < 2 || first.RTPPackets[0].Payload[0]&0x1f != 24 {
		t.Fatalf("first STAP-A IDR packets = %d first payload %x, want SPS/PPS aggregate then VCL",
			len(first.RTPPackets), first.RTPPackets[0].Payload)
	}
	assertSTAPANALTypes(t, first.RTPPackets[0].Payload, []uint8{7, 8})
	callbackPackets = callbackPackets[:0]
	callbackMetadata = callbackMetadata[:0]

	secondFrame := patternedI420EncoderFrame(16, 16)
	secondFrame.Y[0] ^= 0x57
	secondFrame.PTS = firstFrame.PTS + int64(cfg.RTPTimestampIncrement)
	second, err := enc.Encode(secondFrame)
	if err != nil {
		t.Fatalf("Encode changed STAP-A P frame: %v", err)
	}
	if second.IDR || second.KeyFrame {
		t.Fatalf("changed STAP-A P frame key=%v idr=%v, want non-IDR output", second.KeyFrame, second.IDR)
	}
	assertEncoderNALTypes(t, second.NALUnits, []uint8{6, 1})
	if len(second.RTPPackets) != 2 {
		t.Fatalf("changed STAP-A P RTP packet count = %d, want SEI and P slice packets", len(second.RTPPackets))
	}
	for i, pkt := range second.RTPPackets {
		if len(pkt.Payload) == 0 {
			t.Fatalf("changed STAP-A P packet[%d] has empty payload", i)
		}
		if got := pkt.Payload[0] & 0x1f; got == 24 {
			t.Fatalf("changed STAP-A P packet[%d] unexpectedly used STAP-A payload: %x", i, pkt.Payload)
		}
		if len(pkt.Payload) > cfg.RTPMaxPayloadSize {
			t.Fatalf("changed STAP-A P packet[%d] payload size = %d, max %d", i, len(pkt.Payload), cfg.RTPMaxPayloadSize)
		}
		if pkt.Timestamp != second.RTPTime {
			t.Fatalf("changed STAP-A P packet[%d] timestamp = %d, want %d", i, pkt.Timestamp, second.RTPTime)
		}
		if pkt.Marker != (i == len(second.RTPPackets)-1) {
			t.Fatalf("changed STAP-A P packet[%d] marker = %v, want only final marker", i, pkt.Marker)
		}
	}
	if got := second.RTPPackets[0].Payload[0] & 0x1f; got != 6 {
		t.Fatalf("changed STAP-A P packet[0] NAL type = %d, want recovery SEI", got)
	}
	if got := second.RTPPackets[1].Payload[0] & 0x1f; got != 1 {
		t.Fatalf("changed STAP-A P packet[1] NAL type = %d, want P slice", got)
	}
	if len(callbackPackets) != len(second.RTPPackets) || len(callbackMetadata) != len(second.RTPPackets) {
		t.Fatalf("changed STAP-A P callbacks packets/meta = %d/%d, want RTP packet count %d",
			len(callbackPackets), len(callbackMetadata), len(second.RTPPackets))
	}
	for i, meta := range callbackMetadata {
		pkt := callbackPackets[i]
		wantType := []uint8{6, 1}[i]
		if meta.PacketIndex != i || meta.PacketCount != len(second.RTPPackets) {
			t.Fatalf("changed STAP-A P callback meta[%d] index/count = %d/%d, want %d/%d",
				i, meta.PacketIndex, meta.PacketCount, i, len(second.RTPPackets))
		}
		if meta.FramePTS != secondFrame.PTS || meta.FrameDTS != secondFrame.PTS ||
			meta.RTPTime != second.RTPTime || meta.KeyFrame || meta.IDR {
			t.Fatalf("changed STAP-A P callback meta[%d] frame fields = %+v, want non-IDR P-frame timing", i, meta)
		}
		if pkt.SequenceNumber != second.RTPPackets[i].SequenceNumber ||
			pkt.Timestamp != second.RTPPackets[i].Timestamp ||
			pkt.PayloadType != cfg.RTPPayloadType ||
			pkt.SSRC != cfg.RTPSSRC ||
			pkt.Marker != (i == len(second.RTPPackets)-1) ||
			!bytes.Equal(pkt.Payload, second.RTPPackets[i].Payload) ||
			!bytes.Equal(pkt.Data, second.RTPPackets[i].Data) {
			t.Fatalf("changed STAP-A P callback packet[%d] metadata = %+v, want returned RTP packet fields", i, pkt)
		}
		assertEncoderRTPCallbackPacketDoesNotAliasReturned(t, pkt, second.RTPPackets[i], i)
		if meta.PayloadFormat != goh264.EncoderRTPPayloadSingleNAL ||
			meta.NALUnitType != wantType ||
			meta.NALUnitCount != 1 ||
			!meta.StartOfNAL || !meta.EndOfNAL ||
			meta.ParameterSet {
			t.Fatalf("changed STAP-A P callback meta[%d] = %+v, want single-NAL type %d", i, meta, wantType)
		}
	}

	dec := goh264.NewDecoder()
	decodedFirst, err := dec.DecodeFrames(annexBFromEncoderRTPPackets(t, first.RTPPackets))
	if err != nil {
		t.Fatalf("Decode first STAP-A IDR: %v", err)
	}
	assertDecodedEncoderFrameBytes(t, decodedFirst, appendI420FrameBytes(nil, firstFrame))
	decodedSecond, err := dec.DecodeFrames(annexBFromEncoderRTPPackets(t, second.RTPPackets))
	if err != nil {
		t.Fatalf("Decode changed STAP-A P frame: %v", err)
	}
	assertDecodedEncoderFrameBytes(t, decodedSecond, appendI420FrameBytes(nil, secondFrame))
	if !decodedSecond[0].KeyFrame ||
		decodedSecond[0].SideData.RecoveryPoint == nil ||
		decodedSecond[0].SideData.RecoveryPoint.RecoveryFrameCount != 0 {
		t.Fatalf("changed STAP-A P recovery key=%v recovery=%+v, want immediate recovery point",
			decodedSecond[0].KeyFrame, decodedSecond[0].SideData.RecoveryPoint)
	}
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
	assertEncoderRTPMode0RawNALPackets(t, out, cfg.RTPMaxPayloadSize)

	annexB := annexBFromEncoderRTPPackets(t, out.RTPPackets)
	decoded, err := goh264.NewDecoder().DecodeAnnexBFrames(annexB)
	if err != nil {
		t.Fatalf("DecodeAnnexBFrames reassembled mode 0 RTP: %v", err)
	}
	assertDecodedEncoderFrameBytes(t, decoded, appendI420FrameBytes(nil, frame))
}

func TestEncoderEncodeRTPMode0EmitsPFrameSingleNALPackets(t *testing.T) {
	for _, tt := range []struct {
		name         string
		prepareFirst func(*goh264.EncoderFrame)
		nextFrame    func(goh264.EncoderFrame) goh264.EncoderFrame
		wantNALs     []uint8
		wantRecovery bool
	}{
		{
			name: "p-skip",
			nextFrame: func(first goh264.EncoderFrame) goh264.EncoderFrame {
				return first
			},
			wantNALs: []uint8{1},
		},
		{
			name: "exact-p16x16",
			nextFrame: func(first goh264.EncoderFrame) goh264.EncoderFrame {
				return integerMotionI420EncoderFrame(first, 2, 0)
			},
			wantNALs: []uint8{1},
		},
		{
			name: "odd-exact-p16x16-constant-chroma",
			prepareFirst: func(first *goh264.EncoderFrame) {
				setConstantI420Chroma(first, 128, 64)
			},
			nextFrame: func(first goh264.EncoderFrame) goh264.EncoderFrame {
				return integerMotionI420EncoderFrame(first, 1, 0)
			},
			wantNALs: []uint8{1},
		},
		{
			name: "odd-exact-p16x16-patterned-chroma-fallback",
			nextFrame: func(first goh264.EncoderFrame) goh264.EncoderFrame {
				return integerMotionI420EncoderFrame(first, 1, 0)
			},
			wantNALs:     []uint8{6, 1},
			wantRecovery: true,
		},
		{
			name: "changed-p-intrapcm",
			nextFrame: func(first goh264.EncoderFrame) goh264.EncoderFrame {
				second := patternedI420EncoderFrame(first.Width, first.Height)
				second.Y[0] ^= 0x39
				return second
			},
			wantNALs:     []uint8{6, 1},
			wantRecovery: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cfg := goh264.DefaultEncoderConfig(16, 16)
			cfg.RTPPacketizationMode = goh264.EncoderRTPPacketizationSingleNAL
			cfg.RTPMaxPayloadSize = 1200
			cfg.DeblockMode = goh264.EncoderDeblockDisabled
			enc, err := goh264.NewEncoder(cfg)
			if err != nil {
				t.Fatalf("NewEncoder mode 0: %v", err)
			}
			firstFrame := patternedI420EncoderFrame(16, 16)
			if tt.prepareFirst != nil {
				tt.prepareFirst(&firstFrame)
			}
			firstFrame.PTS = 9000
			first, err := enc.Encode(firstFrame)
			if err != nil {
				t.Fatalf("Encode first mode 0 IDR: %v", err)
			}
			assertEncoderNALTypes(t, first.NALUnits, []uint8{7, 8, 5})
			assertEncoderRTPMode0RawNALPackets(t, first, cfg.RTPMaxPayloadSize)

			secondFrame := tt.nextFrame(firstFrame)
			secondFrame.PTS = firstFrame.PTS + int64(cfg.RTPTimestampIncrement)
			second, err := enc.Encode(secondFrame)
			if err != nil {
				t.Fatalf("Encode mode 0 %s: %v", tt.name, err)
			}
			if second.IDR || second.KeyFrame {
				t.Fatalf("mode 0 %s key=%v idr=%v, want non-IDR P frame", tt.name, second.KeyFrame, second.IDR)
			}
			assertEncoderNALTypes(t, second.NALUnits, tt.wantNALs)
			assertEncoderRTPMode0RawNALPackets(t, second, cfg.RTPMaxPayloadSize)

			dec := goh264.NewDecoder()
			decodedFirst, err := dec.DecodeFrames(annexBFromEncoderRTPPackets(t, first.RTPPackets))
			if err != nil {
				t.Fatalf("DecodeFrames first mode 0 IDR: %v", err)
			}
			assertDecodedEncoderFrameBytes(t, decodedFirst, appendI420FrameBytes(nil, firstFrame))
			decodedSecond, err := dec.DecodeFrames(annexBFromEncoderRTPPackets(t, second.RTPPackets))
			if err != nil {
				t.Fatalf("DecodeFrames mode 0 %s: %v", tt.name, err)
			}
			assertDecodedEncoderFrameBytes(t, decodedSecond, appendI420FrameBytes(nil, secondFrame))
			recovery := decodedSecond[0].SideData.RecoveryPoint
			if tt.wantRecovery {
				if !decodedSecond[0].KeyFrame || recovery == nil || recovery.RecoveryFrameCount != 0 {
					t.Fatalf("mode 0 %s recovery key=%v recovery=%+v, want immediate recovery point",
						tt.name, decodedSecond[0].KeyFrame, recovery)
				}
			} else if decodedSecond[0].KeyFrame || recovery != nil {
				t.Fatalf("mode 0 %s decoded key=%v recovery=%+v, want predictive non-recovery frame",
					tt.name, decodedSecond[0].KeyFrame, recovery)
			}
		})
	}
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

func TestEncoderEncodeIntoRTPMode0RejectPreservesCallerBuffer(t *testing.T) {
	cfg := goh264.DefaultEncoderConfig(16, 16)
	cfg.RTPPacketizationMode = goh264.EncoderRTPPacketizationSingleNAL
	cfg.RTPMaxPayloadSize = 64
	cfg.DeblockMode = goh264.EncoderDeblockDisabled
	enc, err := goh264.NewEncoder(cfg)
	if err != nil {
		t.Fatalf("NewEncoder mode 0: %v", err)
	}

	dst, backingBefore := encoderPrefilledCallerBuffer()
	out, err := enc.EncodeInto(dst, patternedI420EncoderFrame(16, 16))
	if !errors.Is(err, goh264.ErrInvalidData) {
		t.Fatalf("EncodeInto oversize mode 0 error = %v, want ErrInvalidData", err)
	}
	if len(out.Data) != 0 || len(out.NALUnits) != 0 || len(out.RTPPackets) != 0 ||
		out.KeyFrame || out.IDR || out.PTS != 0 || out.DTS != 0 || out.RTPTime != 0 || out.Dropped {
		t.Fatalf("EncodeInto oversize mode 0 output = %+v, want empty frame", out)
	}
	assertEncoderCallerBufferUnchanged(t, dst, backingBefore)
}

func TestEncoderRTPMode0OversizeRejectPreservesLiveState(t *testing.T) {
	for _, tt := range []struct {
		name     string
		forceIDR bool
		mutate   func(goh264.EncoderFrame) goh264.EncoderFrame
		wantIDR  bool
		wantNALs []uint8
	}{
		{
			name:     "queued-idr",
			forceIDR: true,
			mutate: func(frame goh264.EncoderFrame) goh264.EncoderFrame {
				return frame
			},
			wantIDR:  true,
			wantNALs: []uint8{7, 8, 5},
		},
		{
			name: "p-intrapcm",
			mutate: func(frame goh264.EncoderFrame) goh264.EncoderFrame {
				frame = cloneI420EncoderFrame(frame)
				frame.Y[0] ^= 0x44
				return frame
			},
			wantNALs: []uint8{6, 1},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cfg := goh264.DefaultEncoderConfig(16, 16)
			cfg.RTPPacketizationMode = goh264.EncoderRTPPacketizationSingleNAL
			cfg.RTPMaxPayloadSize = 1200
			cfg.DeblockMode = goh264.EncoderDeblockDisabled
			enc, err := goh264.NewEncoder(cfg)
			if err != nil {
				t.Fatalf("NewEncoder mode 0: %v", err)
			}
			var callbackCalls int
			enc.SetRTPPacketCallback(func(goh264.EncoderRTPPacket, goh264.EncoderRTPPacketMetadata) {
				callbackCalls++
			})

			firstFrame := patternedI420EncoderFrame(16, 16)
			firstFrame.PTS = 0
			first, err := enc.EncodeInto(make([]byte, 0, 4096), firstFrame)
			if err != nil {
				t.Fatalf("Encode first mode-0 IDR: %v", err)
			}
			assertEncoderNALTypes(t, first.NALUnits, []uint8{7, 8, 5})
			firstPacketCount := len(first.RTPPackets)
			if firstPacketCount == 0 || callbackCalls != firstPacketCount {
				t.Fatalf("first mode-0 packets/callbacks = %d/%d, want nonzero matching count",
					firstPacketCount, callbackCalls)
			}

			if err := enc.SetRTPMaxPayloadSize(64); err != nil {
				t.Fatalf("lower RTP payload size: %v", err)
			}
			if tt.forceIDR {
				enc.ForceIDR()
				if !enc.PendingIDR() {
					t.Fatal("ForceIDR did not queue IDR before mode-0 oversize rejection")
				}
			}
			nextFrame := tt.mutate(firstFrame)
			nextFrame.PTS = int64(cfg.RTPTimestampIncrement)
			dst, backingBefore := encoderPrefilledCallerBuffer()
			rejected, err := enc.EncodeInto(dst, nextFrame)
			if !errors.Is(err, goh264.ErrInvalidData) {
				t.Fatalf("EncodeInto mode-0 oversize %s error = %v, want ErrInvalidData", tt.name, err)
			}
			if rejected.Dropped || len(rejected.Data) != 0 || len(rejected.NALUnits) != 0 || len(rejected.RTPPackets) != 0 {
				t.Fatalf("mode-0 oversize %s output = %+v, want empty output", tt.name, rejected)
			}
			assertEncoderCallerBufferUnchanged(t, dst, backingBefore)
			if enc.PendingIDR() != tt.forceIDR {
				t.Fatalf("mode-0 oversize %s pending IDR = %v, want %v", tt.name, enc.PendingIDR(), tt.forceIDR)
			}
			if callbackCalls != firstPacketCount {
				t.Fatalf("mode-0 oversize %s callbacks = %d, want still %d",
					tt.name, callbackCalls, firstPacketCount)
			}

			if err := enc.SetRTPMaxPayloadSize(1200); err != nil {
				t.Fatalf("restore RTP payload size: %v", err)
			}
			recovered, err := enc.EncodeInto(make([]byte, 0, 4096), nextFrame)
			if err != nil {
				t.Fatalf("EncodeInto after mode-0 oversize %s: %v", tt.name, err)
			}
			if recovered.Dropped || recovered.IDR != tt.wantIDR || enc.PendingIDR() {
				t.Fatalf("post-mode-0-oversize %s output dropped=%v idr=%v pending=%v, want idr=%v",
					tt.name, recovered.Dropped, recovered.IDR, enc.PendingIDR(), tt.wantIDR)
			}
			if recovered.RTPTime != uint32(nextFrame.PTS) {
				t.Fatalf("post-mode-0-oversize %s RTP time = %d, want %d",
					tt.name, recovered.RTPTime, nextFrame.PTS)
			}
			assertEncoderNALTypes(t, recovered.NALUnits, tt.wantNALs)
			assertEncoderRTPMode0RawNALPackets(t, recovered, 1200)
			assertRTPPacketMetadata(t, recovered.RTPPackets, cfg.RTPPayloadType, cfg.RTPSSRC, uint16(firstPacketCount))
			if callbackCalls != firstPacketCount+len(recovered.RTPPackets) {
				t.Fatalf("post-mode-0-oversize %s callbacks = %d, want %d",
					tt.name, callbackCalls, firstPacketCount+len(recovered.RTPPackets))
			}
		})
	}
}

func TestEncoderRTPMode1STAPAFallbackAtSmallPayloadPreservesLiveState(t *testing.T) {
	cfg := goh264.DefaultEncoderConfig(16, 16)
	cfg.RTPPacketizationMode = goh264.EncoderRTPPacketizationNonInterleaved
	cfg.RTPMaxPayloadSize = 128
	cfg.STAPA = true
	cfg.DeblockMode = goh264.EncoderDeblockDisabled
	enc, err := goh264.NewEncoder(cfg)
	if err != nil {
		t.Fatalf("NewEncoder STAP-A: %v", err)
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
	frame.PTS = 0
	first, err := enc.EncodeInto(make([]byte, 0, 4096), frame)
	if err != nil {
		t.Fatalf("Encode first STAP-A IDR: %v", err)
	}
	if !first.IDR || enc.PendingIDR() {
		t.Fatalf("first STAP-A frame idr=%v pending=%v, want completed IDR", first.IDR, enc.PendingIDR())
	}
	if len(first.RTPPackets) < 2 || len(callbackPackets) != len(first.RTPPackets) || len(callbackMetadata) != len(first.RTPPackets) {
		t.Fatalf("first STAP-A packets/callbacks/meta = %d/%d/%d, want multiple matching packets",
			len(first.RTPPackets), len(callbackPackets), len(callbackMetadata))
	}
	if first.RTPPackets[0].Payload[0]&0x1f != 24 {
		t.Fatalf("first STAP-A payload type = %d, want STAP-A", first.RTPPackets[0].Payload[0]&0x1f)
	}
	firstPacketCount := len(first.RTPPackets)

	if err := enc.SetRTPMaxPayloadSize(3); err != nil {
		t.Fatalf("lower RTP payload size for STAP-A fallback: %v", err)
	}
	enc.ForceIDR()
	if !enc.PendingIDR() {
		t.Fatal("ForceIDR did not queue IDR before STAP-A fallback")
	}
	nextFrame := frame
	nextFrame.PTS = int64(cfg.RTPTimestampIncrement)
	fallback, err := enc.EncodeInto(make([]byte, 0, 4096), nextFrame)
	if err != nil {
		t.Fatalf("EncodeInto STAP-A small-payload fallback: %v", err)
	}
	if fallback.Dropped || !fallback.IDR || enc.PendingIDR() {
		t.Fatalf("STAP-A fallback output dropped=%v idr=%v pending=%v, want delivered IDR",
			fallback.Dropped, fallback.IDR, enc.PendingIDR())
	}
	if fallback.RTPTime != uint32(nextFrame.PTS) {
		t.Fatalf("STAP-A fallback RTP time = %d, want %d", fallback.RTPTime, nextFrame.PTS)
	}
	assertEncoderNALTypes(t, fallback.NALUnits, []uint8{7, 8, 5})
	if len(fallback.RTPPackets) <= len(first.RTPPackets) {
		t.Fatalf("STAP-A fallback packet count = %d, want more than aggregated count %d", len(fallback.RTPPackets), len(first.RTPPackets))
	}
	for i, pkt := range fallback.RTPPackets {
		if len(pkt.Payload) > 3 {
			t.Fatalf("STAP-A fallback packet[%d] payload size = %d, want <= 3", i, len(pkt.Payload))
		}
		if len(pkt.Payload) != 0 && pkt.Payload[0]&0x1f == 24 {
			t.Fatalf("STAP-A fallback packet[%d] unexpectedly used STAP-A payload: %x", i, pkt.Payload)
		}
	}
	assertRTPPacketMetadata(t, fallback.RTPPackets, cfg.RTPPayloadType, cfg.RTPSSRC, uint16(firstPacketCount))
	if len(callbackPackets) != firstPacketCount+len(fallback.RTPPackets) ||
		len(callbackMetadata) != firstPacketCount+len(fallback.RTPPackets) {
		t.Fatalf("STAP-A fallback callbacks/meta = %d/%d, want %d",
			len(callbackPackets), len(callbackMetadata), firstPacketCount+len(fallback.RTPPackets))
	}
	fallbackCallbackPackets := callbackPackets[firstPacketCount:]
	fallbackMetadata := callbackMetadata[firstPacketCount:]
	for i, meta := range fallbackMetadata {
		pkt := fallbackCallbackPackets[i]
		if pkt.PayloadType != fallback.RTPPackets[i].PayloadType ||
			pkt.SequenceNumber != fallback.RTPPackets[i].SequenceNumber ||
			pkt.Timestamp != fallback.RTPPackets[i].Timestamp ||
			pkt.SSRC != fallback.RTPPackets[i].SSRC ||
			pkt.Marker != fallback.RTPPackets[i].Marker {
			t.Fatalf("STAP-A fallback callback packet[%d] metadata = %+v, want returned RTP packet fields", i, pkt)
		}
		if meta.PacketIndex != i || meta.PacketCount != len(fallback.RTPPackets) ||
			meta.FramePTS != nextFrame.PTS || meta.FrameDTS != nextFrame.PTS ||
			meta.RTPTime != fallback.RTPTime || !meta.KeyFrame || !meta.IDR {
			t.Fatalf("STAP-A fallback callback meta[%d] = %+v, want IDR packet timing/index fields", i, meta)
		}
		if meta.PayloadFormat == goh264.EncoderRTPPayloadSTAPA {
			t.Fatalf("STAP-A fallback callback meta[%d] reported STAP-A: %+v", i, meta)
		}
		if meta.PayloadFormat != goh264.EncoderRTPPayloadSingleNAL &&
			meta.PayloadFormat != goh264.EncoderRTPPayloadFUA {
			t.Fatalf("STAP-A fallback callback meta[%d] payload format = %v, want single-NAL or FU-A", i, meta.PayloadFormat)
		}
		if meta.PayloadFormat == goh264.EncoderRTPPayloadFUA &&
			meta.NALUnitType != 7 &&
			meta.NALUnitType != 8 &&
			meta.NALUnitType != 5 {
			t.Fatalf("STAP-A fallback FU-A meta[%d] NAL type = %d, want SPS/PPS/IDR", i, meta.NALUnitType)
		}
	}
	dec := goh264.NewDecoder()
	decodedFirst, err := dec.DecodeFrames(annexBFromEncoderRTPPackets(t, first.RTPPackets))
	if err != nil {
		t.Fatalf("Decode first STAP-A IDR: %v", err)
	}
	assertDecodedEncoderFrameBytes(t, decodedFirst, appendI420FrameBytes(nil, frame))
	decodedFallback, err := dec.DecodeFrames(annexBFromEncoderRTPPackets(t, fallback.RTPPackets))
	if err != nil {
		t.Fatalf("Decode STAP-A fallback IDR: %v", err)
	}
	assertDecodedEncoderFrameBytes(t, decodedFallback, appendI420FrameBytes(nil, nextFrame))

	if err := enc.SetRTPMaxPayloadSize(128); err != nil {
		t.Fatalf("restore RTP payload size after STAP-A fallback: %v", err)
	}
	pFrame := nextFrame
	pFrame.PTS += int64(cfg.RTPTimestampIncrement)
	recovered, err := enc.EncodeInto(make([]byte, 0, 4096), pFrame)
	if err != nil {
		t.Fatalf("EncodeInto after STAP-A fallback: %v", err)
	}
	if recovered.Dropped || recovered.IDR || enc.PendingIDR() {
		t.Fatalf("post-STAP-A-fallback output dropped=%v idr=%v pending=%v, want delivered P-skip",
			recovered.Dropped, recovered.IDR, enc.PendingIDR())
	}
	if recovered.RTPTime != uint32(pFrame.PTS) {
		t.Fatalf("post-STAP-A-fallback RTP time = %d, want %d", recovered.RTPTime, pFrame.PTS)
	}
	assertEncoderNALTypes(t, recovered.NALUnits, []uint8{1})
	assertRTPPacketMetadata(t, recovered.RTPPackets, cfg.RTPPayloadType, cfg.RTPSSRC, uint16(firstPacketCount+len(fallback.RTPPackets)))
	if len(callbackPackets) != firstPacketCount+len(fallback.RTPPackets)+len(recovered.RTPPackets) ||
		len(callbackMetadata) != firstPacketCount+len(fallback.RTPPackets)+len(recovered.RTPPackets) {
		t.Fatalf("post-STAP-A-fallback callbacks/meta = %d/%d, want %d",
			len(callbackPackets), len(callbackMetadata), firstPacketCount+len(fallback.RTPPackets)+len(recovered.RTPPackets))
	}
	recoveredCallbackPackets := callbackPackets[firstPacketCount+len(fallback.RTPPackets):]
	recoveredMetadata := callbackMetadata[firstPacketCount+len(fallback.RTPPackets):]
	for i, meta := range recoveredMetadata {
		pkt := recoveredCallbackPackets[i]
		if pkt.PayloadType != recovered.RTPPackets[i].PayloadType ||
			pkt.SequenceNumber != recovered.RTPPackets[i].SequenceNumber ||
			pkt.Timestamp != recovered.RTPPackets[i].Timestamp ||
			pkt.SSRC != recovered.RTPPackets[i].SSRC ||
			pkt.Marker != recovered.RTPPackets[i].Marker {
			t.Fatalf("post-STAP-A-fallback callback packet[%d] metadata = %+v, want returned RTP packet fields", i, pkt)
		}
		if meta.PacketIndex != i || meta.PacketCount != len(recovered.RTPPackets) ||
			meta.FramePTS != pFrame.PTS || meta.FrameDTS != pFrame.PTS ||
			meta.RTPTime != recovered.RTPTime || meta.KeyFrame || meta.IDR {
			t.Fatalf("post-STAP-A-fallback callback meta[%d] = %+v, want non-IDR P-skip timing/index fields", i, meta)
		}
		if meta.PayloadFormat != goh264.EncoderRTPPayloadSingleNAL ||
			meta.NALUnitType != 1 ||
			meta.NALUnitCount != 1 ||
			!meta.StartOfNAL || !meta.EndOfNAL ||
			meta.ParameterSet {
			t.Fatalf("post-STAP-A-fallback callback meta[%d] = %+v, want single-NAL P-slice", i, meta)
		}
	}
}

func TestEncoderEncodeIntoLateDropPreservesCallerBuffer(t *testing.T) {
	for _, tt := range []struct {
		name         string
		outputFormat goh264.EncoderOutputFormat
	}{
		{name: "annexb", outputFormat: goh264.EncoderOutputAnnexB},
		{name: "avc", outputFormat: goh264.EncoderOutputAVC},
		{name: "rtp", outputFormat: goh264.EncoderOutputRTP},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cfg := goh264.DefaultEncoderConfig(16, 16)
			cfg.OutputFormat = tt.outputFormat
			cfg.DeblockMode = goh264.EncoderDeblockDisabled
			cfg.FrameDrop = goh264.EncoderFrameDropLate
			cfg.MaxEncodeTimeUS = 1
			if tt.outputFormat != goh264.EncoderOutputRTP {
				cfg.RTPMaxPayloadSize = 0
			}
			enc, err := goh264.NewEncoder(cfg)
			if err != nil {
				t.Fatalf("NewEncoder: %v", err)
			}

			dst, backingBefore := encoderPrefilledCallerBuffer()
			out, err := enc.EncodeInto(dst, patternedI420EncoderFrame(16, 16))
			if err != nil {
				t.Fatalf("EncodeInto late drop: %v", err)
			}
			if !out.Dropped || len(out.Data) != 0 || len(out.NALUnits) != 0 || len(out.RTPPackets) != 0 {
				t.Fatalf("late-drop output = %+v, want dropped metadata without output", out)
			}
			assertEncoderCallerBufferUnchanged(t, dst, backingBefore)
		})
	}
}

func TestEncoderEncodeIntoBitrateDropPreservesCallerBuffer(t *testing.T) {
	for _, tt := range []struct {
		name         string
		outputFormat goh264.EncoderOutputFormat
		mutate       func(*goh264.EncoderConfig)
	}{
		{name: "annexb max-frame-size", outputFormat: goh264.EncoderOutputAnnexB, mutate: func(cfg *goh264.EncoderConfig) {
			cfg.MaxFrameSize = 16
		}},
		{name: "avc max-frame-size", outputFormat: goh264.EncoderOutputAVC, mutate: func(cfg *goh264.EncoderConfig) {
			cfg.MaxFrameSize = 16
		}},
		{name: "rtp max-frame-size", outputFormat: goh264.EncoderOutputRTP, mutate: func(cfg *goh264.EncoderConfig) {
			cfg.MaxFrameSize = 16
		}},
		{name: "annexb slice-max-bytes", outputFormat: goh264.EncoderOutputAnnexB, mutate: func(cfg *goh264.EncoderConfig) {
			cfg.SliceMaxBytes = 1
		}},
		{name: "avc slice-max-bytes", outputFormat: goh264.EncoderOutputAVC, mutate: func(cfg *goh264.EncoderConfig) {
			cfg.SliceMaxBytes = 1
		}},
		{name: "rtp slice-max-bytes", outputFormat: goh264.EncoderOutputRTP, mutate: func(cfg *goh264.EncoderConfig) {
			cfg.SliceMaxBytes = 1
		}},
		{name: "annexb max-bitrate", outputFormat: goh264.EncoderOutputAnnexB, mutate: func(cfg *goh264.EncoderConfig) {
			cfg.TargetBitrate = 1_000
			cfg.MaxBitrate = 1_000
			cfg.VBVBufferSize = 64
		}},
		{name: "avc max-bitrate", outputFormat: goh264.EncoderOutputAVC, mutate: func(cfg *goh264.EncoderConfig) {
			cfg.TargetBitrate = 1_000
			cfg.MaxBitrate = 1_000
			cfg.VBVBufferSize = 64
		}},
		{name: "rtp max-bitrate", outputFormat: goh264.EncoderOutputRTP, mutate: func(cfg *goh264.EncoderConfig) {
			cfg.TargetBitrate = 1_000
			cfg.MaxBitrate = 1_000
			cfg.VBVBufferSize = 64
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cfg := goh264.DefaultEncoderConfig(16, 16)
			cfg.OutputFormat = tt.outputFormat
			cfg.DeblockMode = goh264.EncoderDeblockDisabled
			cfg.FrameDrop = goh264.EncoderFrameDropToBitrate
			if tt.outputFormat != goh264.EncoderOutputRTP {
				cfg.RTPMaxPayloadSize = 0
			}
			tt.mutate(&cfg)
			enc, err := goh264.NewEncoder(cfg)
			if err != nil {
				t.Fatalf("NewEncoder: %v", err)
			}

			dst, backingBefore := encoderPrefilledCallerBuffer()
			out, err := enc.EncodeInto(dst, patternedI420EncoderFrame(16, 16))
			if err != nil {
				t.Fatalf("EncodeInto bitrate drop: %v", err)
			}
			if !out.Dropped || len(out.Data) != 0 || len(out.NALUnits) != 0 || len(out.RTPPackets) != 0 {
				t.Fatalf("bitrate-drop output = %+v, want dropped metadata without output", out)
			}
			assertEncoderCallerBufferUnchanged(t, dst, backingBefore)
		})
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

func TestEncoderEncodedFrameNALUnitsIndexOutputData(t *testing.T) {
	for _, tt := range []struct {
		name         string
		outputFormat goh264.EncoderOutputFormat
	}{
		{name: "annexb", outputFormat: goh264.EncoderOutputAnnexB},
		{name: "avc", outputFormat: goh264.EncoderOutputAVC},
		{name: "rtp", outputFormat: goh264.EncoderOutputRTP},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cfg := goh264.DefaultEncoderConfig(16, 16)
			cfg.OutputFormat = tt.outputFormat
			cfg.DeblockMode = goh264.EncoderDeblockDisabled
			if tt.outputFormat != goh264.EncoderOutputRTP {
				cfg.RTPMaxPayloadSize = 0
			}
			enc, err := goh264.NewEncoder(cfg)
			if err != nil {
				t.Fatalf("NewEncoder: %v", err)
			}
			out, err := enc.Encode(patternedI420EncoderFrame(16, 16))
			if err != nil {
				t.Fatalf("Encode: %v", err)
			}
			assertEncodedFrameNALUnitIndexes(t, out, cfg.OutputFormat)
		})
	}
}

func TestEncoderEncodeIntoAppendsAndIndexesAfterPrefix(t *testing.T) {
	for _, tt := range []struct {
		name         string
		outputFormat goh264.EncoderOutputFormat
	}{
		{name: "annexb", outputFormat: goh264.EncoderOutputAnnexB},
		{name: "avc", outputFormat: goh264.EncoderOutputAVC},
		{name: "rtp", outputFormat: goh264.EncoderOutputRTP},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cfg := goh264.DefaultEncoderConfig(16, 16)
			cfg.OutputFormat = tt.outputFormat
			cfg.DeblockMode = goh264.EncoderDeblockDisabled
			if tt.outputFormat != goh264.EncoderOutputRTP {
				cfg.RTPMaxPayloadSize = 0
			}
			enc, err := goh264.NewEncoder(cfg)
			if err != nil {
				t.Fatalf("NewEncoder: %v", err)
			}

			prefix := []byte{0xde, 0xad, 0xbe, 0xef, 0x55}
			dst := append([]byte(nil), prefix...)
			out, err := enc.EncodeInto(dst, patternedI420EncoderFrame(16, 16))
			if err != nil {
				t.Fatalf("EncodeInto with prefix: %v", err)
			}
			if !bytes.HasPrefix(out.Data, prefix) {
				t.Fatalf("EncodeInto output does not preserve caller prefix: got %x want prefix %x", out.Data, prefix)
			}
			if len(out.Data) == len(prefix) {
				t.Fatalf("EncodeInto output length = prefix length %d, want appended access unit", len(prefix))
			}
			assertEncodedFrameNALUnitIndexesFrom(t, out, cfg.OutputFormat, len(prefix))
		})
	}
}

func TestEncoderDoesNotRetainInputFramePlanes(t *testing.T) {
	for _, tt := range []struct {
		name         string
		outputFormat goh264.EncoderOutputFormat
	}{
		{name: "annexb", outputFormat: goh264.EncoderOutputAnnexB},
		{name: "avc", outputFormat: goh264.EncoderOutputAVC},
		{name: "rtp", outputFormat: goh264.EncoderOutputRTP},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cfg := goh264.DefaultEncoderConfig(16, 16)
			cfg.OutputFormat = tt.outputFormat
			cfg.GOPSize = 10000
			cfg.IDRInterval = 10000
			cfg.DeblockMode = goh264.EncoderDeblockDisabled
			if tt.outputFormat != goh264.EncoderOutputRTP {
				cfg.RTPMaxPayloadSize = 0
			}

			firstFrame := patternedI420EncoderFrame(16, 16)
			firstFrame.PTS = 0
			secondFrame := cloneI420EncoderFrame(firstFrame)

			enc, err := goh264.NewEncoder(cfg)
			if err != nil {
				t.Fatalf("NewEncoder: %v", err)
			}
			first, err := enc.Encode(firstFrame)
			if err != nil {
				t.Fatalf("Encode first frame: %v", err)
			}
			if !first.IDR {
				t.Fatalf("first frame IDR = false, want true")
			}

			mutateI420EncoderFramePlanes(&firstFrame)

			second, err := enc.Encode(secondFrame)
			if err != nil {
				t.Fatalf("Encode second frame after input mutation: %v", err)
			}
			if second.IDR {
				t.Fatalf("second frame IDR = true, want inter frame")
			}
			assertEncodedFrameNALUnitIndexes(t, second, cfg.OutputFormat)

			controlEnc, err := goh264.NewEncoder(cfg)
			if err != nil {
				t.Fatalf("NewEncoder control: %v", err)
			}
			controlFirst := cloneI420EncoderFrame(secondFrame)
			if _, err := controlEnc.Encode(controlFirst); err != nil {
				t.Fatalf("Encode control first frame: %v", err)
			}
			controlSecond, err := controlEnc.Encode(secondFrame)
			if err != nil {
				t.Fatalf("Encode control second frame: %v", err)
			}

			if !bytes.Equal(second.Data, controlSecond.Data) {
				t.Fatalf("second encoded bytes changed after caller mutated the first input frame")
			}
			if !reflect.DeepEqual(second.NALUnits, controlSecond.NALUnits) {
				t.Fatalf("second NAL metadata changed after caller mutated the first input frame")
			}
			if !reflect.DeepEqual(second.RTPPackets, controlSecond.RTPPackets) {
				t.Fatalf("second RTP packets changed after caller mutated the first input frame")
			}
		})
	}
}

func TestEncoderEncodeResultsSurviveLaterEncode(t *testing.T) {
	for _, tt := range []struct {
		name         string
		outputFormat goh264.EncoderOutputFormat
	}{
		{name: "annexb", outputFormat: goh264.EncoderOutputAnnexB},
		{name: "avc", outputFormat: goh264.EncoderOutputAVC},
		{name: "rtp", outputFormat: goh264.EncoderOutputRTP},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cfg := goh264.DefaultEncoderConfig(16, 16)
			cfg.OutputFormat = tt.outputFormat
			cfg.GOPSize = 10000
			cfg.IDRInterval = 10000
			cfg.DeblockMode = goh264.EncoderDeblockDisabled
			if tt.outputFormat != goh264.EncoderOutputRTP {
				cfg.RTPMaxPayloadSize = 0
			}
			enc, err := goh264.NewEncoder(cfg)
			if err != nil {
				t.Fatalf("NewEncoder: %v", err)
			}

			first, err := enc.Encode(patternedI420EncoderFrame(16, 16))
			if err != nil {
				t.Fatalf("Encode first frame: %v", err)
			}
			firstData := append([]byte(nil), first.Data...)
			firstNALUnits := append([]goh264.EncoderNALUnit(nil), first.NALUnits...)
			firstRTPPackets := cloneEncoderRTPPackets(first.RTPPackets)

			secondFrame := patternedI420EncoderFrame(16, 16)
			secondFrame.PTS += int64(cfg.RTPTimestampIncrement)
			if _, err := enc.Encode(secondFrame); err != nil {
				t.Fatalf("Encode second frame: %v", err)
			}

			if !bytes.Equal(first.Data, firstData) {
				t.Fatalf("first EncodedFrame.Data changed after later encode")
			}
			if !reflect.DeepEqual(first.NALUnits, firstNALUnits) {
				t.Fatalf("first EncodedFrame.NALUnits changed after later encode")
			}
			if !reflect.DeepEqual(first.RTPPackets, firstRTPPackets) {
				t.Fatalf("first EncodedFrame.RTPPackets changed after later encode")
			}
			assertEncodedFrameNALUnitIndexes(t, first, cfg.OutputFormat)
		})
	}
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

func TestEncoderEncodeRTPPacketSlicesAppendDoesNotAliasNextPacket(t *testing.T) {
	for _, tt := range []struct {
		name           string
		maxPayloadSize int
		stapa          bool
	}{
		{name: "fua", maxPayloadSize: 32},
		{name: "stap-a", maxPayloadSize: 128, stapa: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cfg := goh264.DefaultEncoderConfig(16, 16)
			cfg.RTPMaxPayloadSize = tt.maxPayloadSize
			cfg.STAPA = tt.stapa
			cfg.DeblockMode = goh264.EncoderDeblockDisabled
			enc, err := goh264.NewEncoder(cfg)
			if err != nil {
				t.Fatalf("NewEncoder: %v", err)
			}

			out, err := enc.Encode(patternedI420EncoderFrame(16, 16))
			if err != nil {
				t.Fatalf("Encode RTP frame: %v", err)
			}
			if len(out.RTPPackets) < 2 {
				t.Fatalf("RTP packet count = %d, want at least two packets for append isolation", len(out.RTPPackets))
			}
			for i, pkt := range out.RTPPackets {
				if cap(pkt.Payload) != len(pkt.Payload) {
					t.Fatalf("packet[%d] Payload cap = %d, want clipped length %d", i, cap(pkt.Payload), len(pkt.Payload))
				}
				if cap(pkt.Data) != len(pkt.Data) {
					t.Fatalf("packet[%d] Data cap = %d, want clipped length %d", i, cap(pkt.Data), len(pkt.Data))
				}
				if len(pkt.Data) != 12+len(pkt.Payload) || !bytes.Equal(pkt.Data[12:], pkt.Payload) {
					t.Fatalf("packet[%d] Data/Payload lengths or bytes do not match", i)
				}
				orig := pkt.Payload[0]
				pkt.Payload[0] ^= 0xff
				if pkt.Data[12] != pkt.Payload[0] {
					t.Fatalf("packet[%d] Payload is not backed by Data payload bytes", i)
				}
				pkt.Payload[0] = orig
			}

			nextPayloadBefore := append([]byte(nil), out.RTPPackets[1].Payload...)
			nextBefore := append([]byte(nil), out.RTPPackets[1].Data...)
			grownPayload := append(out.RTPPackets[0].Payload, 0x55)
			grownPayload[len(out.RTPPackets[0].Payload)] ^= 0xff
			grown := append(out.RTPPackets[0].Data, 0xaa)
			grown[len(out.RTPPackets[0].Data)] ^= 0xff
			if !bytes.Equal(out.RTPPackets[1].Payload, nextPayloadBefore) {
				t.Fatal("appending to packet[0] Payload mutated packet[1] Payload")
			}
			if !bytes.Equal(out.RTPPackets[1].Data, nextBefore) {
				t.Fatal("appending to packet[0] Data mutated packet[1] Data")
			}
		})
	}
}

func TestEncoderRTPPacketsDoNotAliasEncodedFrameData(t *testing.T) {
	for _, tt := range []struct {
		name              string
		packetizationMode goh264.EncoderRTPPacketizationMode
		maxPayloadSize    int
		stapa             bool
	}{
		{name: "mode0", packetizationMode: goh264.EncoderRTPPacketizationSingleNAL, maxPayloadSize: 1200},
		{name: "mode1-fua", packetizationMode: goh264.EncoderRTPPacketizationNonInterleaved, maxPayloadSize: 32},
		{name: "mode1-stapa-fua", packetizationMode: goh264.EncoderRTPPacketizationNonInterleaved, maxPayloadSize: 128, stapa: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cfg := goh264.DefaultEncoderConfig(16, 16)
			cfg.RTPPacketizationMode = tt.packetizationMode
			cfg.RTPMaxPayloadSize = tt.maxPayloadSize
			cfg.STAPA = tt.stapa
			cfg.DeblockMode = goh264.EncoderDeblockDisabled
			enc, err := goh264.NewEncoder(cfg)
			if err != nil {
				t.Fatalf("NewEncoder: %v", err)
			}

			out, err := enc.Encode(patternedI420EncoderFrame(16, 16))
			if err != nil {
				t.Fatalf("Encode RTP frame: %v", err)
			}
			if len(out.Data) == 0 || len(out.RTPPackets) == 0 {
				t.Fatalf("RTP output data/packets = %d/%d, want nonzero", len(out.Data), len(out.RTPPackets))
			}

			packetPayloads := make([][]byte, len(out.RTPPackets))
			packetData := make([][]byte, len(out.RTPPackets))
			for i, pkt := range out.RTPPackets {
				packetPayloads[i] = append([]byte(nil), pkt.Payload...)
				packetData[i] = append([]byte(nil), pkt.Data...)
			}
			frameData := append([]byte(nil), out.Data...)

			for i := range out.Data {
				out.Data[i] ^= 0xff
			}
			for i, pkt := range out.RTPPackets {
				if !bytes.Equal(pkt.Payload, packetPayloads[i]) {
					t.Fatalf("mutating EncodedFrame.Data changed RTP packet[%d] Payload", i)
				}
				if !bytes.Equal(pkt.Data, packetData[i]) {
					t.Fatalf("mutating EncodedFrame.Data changed RTP packet[%d] Data", i)
				}
			}

			for _, pkt := range out.RTPPackets {
				if len(pkt.Payload) != 0 {
					pkt.Payload[0] ^= 0xff
				}
				if len(pkt.Data) != 0 {
					pkt.Data[0] ^= 0xff
				}
			}
			for i, got := range out.Data {
				want := frameData[i] ^ 0xff
				if got != want {
					t.Fatalf("mutating RTP packets changed EncodedFrame.Data byte %d: got %#x want %#x", i, got, want)
				}
			}
		})
	}
}

func TestEncoderEncodeIntoRTPPacketsDoNotAliasCallerBuffer(t *testing.T) {
	for _, tt := range []struct {
		name              string
		packetizationMode goh264.EncoderRTPPacketizationMode
		maxPayloadSize    int
		stapa             bool
	}{
		{name: "mode0", packetizationMode: goh264.EncoderRTPPacketizationSingleNAL, maxPayloadSize: 1200},
		{name: "mode1-fua", packetizationMode: goh264.EncoderRTPPacketizationNonInterleaved, maxPayloadSize: 32},
		{name: "mode1-stap-a", packetizationMode: goh264.EncoderRTPPacketizationNonInterleaved, maxPayloadSize: 128, stapa: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cfg := goh264.DefaultEncoderConfig(16, 16)
			cfg.RTPPacketizationMode = tt.packetizationMode
			cfg.RTPMaxPayloadSize = tt.maxPayloadSize
			cfg.STAPA = tt.stapa
			cfg.DeblockMode = goh264.EncoderDeblockDisabled
			enc, err := goh264.NewEncoder(cfg)
			if err != nil {
				t.Fatalf("NewEncoder: %v", err)
			}

			dst := make([]byte, 0, 4096)
			out, err := enc.EncodeInto(dst[:0], patternedI420EncoderFrame(16, 16))
			if err != nil {
				t.Fatalf("EncodeInto RTP frame: %v", err)
			}
			if len(out.Data) == 0 || len(out.RTPPackets) == 0 {
				t.Fatalf("RTP output data/packets = %d/%d, want nonzero", len(out.Data), len(out.RTPPackets))
			}
			if cap(out.Data) != cap(dst) {
				t.Fatalf("EncodeInto Data cap = %d, want caller buffer cap %d", cap(out.Data), cap(dst))
			}

			packetPayloads := make([][]byte, len(out.RTPPackets))
			packetData := make([][]byte, len(out.RTPPackets))
			for i, pkt := range out.RTPPackets {
				packetPayloads[i] = append([]byte(nil), pkt.Payload...)
				packetData[i] = append([]byte(nil), pkt.Data...)
			}

			for i := range out.Data {
				out.Data[i] ^= 0xff
			}
			for i, pkt := range out.RTPPackets {
				if !bytes.Equal(pkt.Payload, packetPayloads[i]) {
					t.Fatalf("mutating caller-backed EncodedFrame.Data changed RTP packet[%d] Payload", i)
				}
				if !bytes.Equal(pkt.Data, packetData[i]) {
					t.Fatalf("mutating caller-backed EncodedFrame.Data changed RTP packet[%d] Data", i)
				}
			}
		})
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

	var sawSTAPA, sawFUAStart, sawFUAEnd bool
	for i, meta := range callbackMetadata {
		pkt := callbackPackets[i]
		assertEncoderRTPCallbackPacketDoesNotAliasReturned(t, pkt, out.RTPPackets[i], i)
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

func TestEncoderRTPPacketCallbackReceivesMode0IDRSingleNALMetadata(t *testing.T) {
	cfg := goh264.DefaultEncoderConfig(16, 16)
	cfg.RTPPacketizationMode = goh264.EncoderRTPPacketizationSingleNAL
	cfg.RTPMaxPayloadSize = 1200
	cfg.RTPPayloadType = 106
	cfg.RTPSSRC = 0x55667788
	cfg.DeblockMode = goh264.EncoderDeblockDisabled
	enc, err := goh264.NewEncoder(cfg)
	if err != nil {
		t.Fatalf("NewEncoder mode 0: %v", err)
	}

	var callbackPackets []goh264.EncoderRTPPacket
	var callbackMetadata []goh264.EncoderRTPPacketMetadata
	enc.SetRTPPacketCallback(func(pkt goh264.EncoderRTPPacket, meta goh264.EncoderRTPPacketMetadata) {
		callbackPackets = append(callbackPackets, pkt)
		callbackMetadata = append(callbackMetadata, meta)
	})

	frame := patternedI420EncoderFrame(16, 16)
	frame.PTS = 0x222000
	out, err := enc.Encode(frame)
	if err != nil {
		t.Fatalf("Encode mode 0 IDR with callback: %v", err)
	}
	assertEncoderNALTypes(t, out.NALUnits, []uint8{7, 8, 5})
	assertEncoderRTPMode0RawNALPackets(t, out, cfg.RTPMaxPayloadSize)
	if len(callbackPackets) != len(out.RTPPackets) || len(callbackMetadata) != len(out.RTPPackets) {
		t.Fatalf("callback packets/meta = %d/%d, want RTP packet count %d",
			len(callbackPackets), len(callbackMetadata), len(out.RTPPackets))
	}
	for i, meta := range callbackMetadata {
		pkt := callbackPackets[i]
		wantType := out.NALUnits[i].Type
		if meta.PacketIndex != i || meta.PacketCount != len(out.RTPPackets) {
			t.Fatalf("callback meta[%d] index/count = %d/%d, want %d/%d",
				i, meta.PacketIndex, meta.PacketCount, i, len(out.RTPPackets))
		}
		if meta.FramePTS != frame.PTS || meta.FrameDTS != frame.PTS ||
			meta.RTPTime != out.RTPTime || !meta.KeyFrame || !meta.IDR {
			t.Fatalf("callback meta[%d] frame fields = %+v, want IDR timing metadata", i, meta)
		}
		if pkt.SequenceNumber != out.RTPPackets[i].SequenceNumber ||
			pkt.Timestamp != out.RTPPackets[i].Timestamp ||
			pkt.PayloadType != cfg.RTPPayloadType ||
			pkt.SSRC != cfg.RTPSSRC ||
			pkt.Marker != (i == len(out.RTPPackets)-1) ||
			!bytes.Equal(pkt.Payload, out.RTPPackets[i].Payload) ||
			!bytes.Equal(pkt.Data, out.RTPPackets[i].Data) {
			t.Fatalf("callback packet[%d] metadata = %+v, want returned RTP packet fields", i, pkt)
		}
		assertEncoderRTPCallbackPacketDoesNotAliasReturned(t, pkt, out.RTPPackets[i], i)
		if meta.PayloadFormat != goh264.EncoderRTPPayloadSingleNAL ||
			meta.NALUnitType != wantType ||
			meta.NALUnitCount != 1 ||
			!meta.StartOfNAL || !meta.EndOfNAL ||
			meta.ParameterSet != (wantType == 7 || wantType == 8) {
			t.Fatalf("callback meta[%d] = %+v, want complete mode-0 IDR single-NAL type %d",
				i, meta, wantType)
		}
	}
}

func TestEncoderRTPPacketCallbackReceivesMode1SingleNALMetadata(t *testing.T) {
	for _, tt := range []struct {
		name         string
		prepareFirst func(*goh264.EncoderFrame)
		nextFrame    func(goh264.EncoderFrame) goh264.EncoderFrame
		wantNALs     []uint8
	}{
		{
			name: "idr",
			nextFrame: func(first goh264.EncoderFrame) goh264.EncoderFrame {
				return first
			},
			wantNALs: []uint8{7, 8, 5},
		},
		{
			name: "p-skip",
			nextFrame: func(first goh264.EncoderFrame) goh264.EncoderFrame {
				return first
			},
			wantNALs: []uint8{1},
		},
		{
			name: "exact-p16x16",
			nextFrame: func(first goh264.EncoderFrame) goh264.EncoderFrame {
				return integerMotionI420EncoderFrame(first, 2, 0)
			},
			wantNALs: []uint8{1},
		},
		{
			name: "odd-exact-p16x16-constant-chroma",
			prepareFirst: func(first *goh264.EncoderFrame) {
				setConstantI420Chroma(first, 128, 64)
			},
			nextFrame: func(first goh264.EncoderFrame) goh264.EncoderFrame {
				return integerMotionI420EncoderFrame(first, 1, 0)
			},
			wantNALs: []uint8{1},
		},
		{
			name: "odd-exact-p16x16-patterned-chroma-fallback",
			nextFrame: func(first goh264.EncoderFrame) goh264.EncoderFrame {
				return integerMotionI420EncoderFrame(first, 1, 0)
			},
			wantNALs: []uint8{6, 1},
		},
		{
			name: "changed-p-intrapcm",
			nextFrame: func(first goh264.EncoderFrame) goh264.EncoderFrame {
				second := patternedI420EncoderFrame(first.Width, first.Height)
				second.Y[0] ^= 0x51
				return second
			},
			wantNALs: []uint8{6, 1},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cfg := goh264.DefaultEncoderConfig(16, 16)
			cfg.RTPMaxPayloadSize = 1200
			cfg.RTPPayloadType = 107
			cfg.RTPSSRC = 0x99aabbcc
			cfg.DeblockMode = goh264.EncoderDeblockDisabled
			enc, err := goh264.NewEncoder(cfg)
			if err != nil {
				t.Fatalf("NewEncoder mode 1: %v", err)
			}

			var callbackPackets []goh264.EncoderRTPPacket
			var callbackMetadata []goh264.EncoderRTPPacketMetadata
			enc.SetRTPPacketCallback(func(pkt goh264.EncoderRTPPacket, meta goh264.EncoderRTPPacketMetadata) {
				callbackPackets = append(callbackPackets, pkt)
				callbackMetadata = append(callbackMetadata, meta)
			})

			firstFrame := patternedI420EncoderFrame(16, 16)
			if tt.prepareFirst != nil {
				tt.prepareFirst(&firstFrame)
			}
			firstFrame.PTS = 24_000
			first, err := enc.Encode(firstFrame)
			if err != nil {
				t.Fatalf("Encode first mode 1 IDR: %v", err)
			}
			assertEncoderNALTypes(t, first.NALUnits, []uint8{7, 8, 5})
			assertEncoderRTPSingleNALPackets(t, first, cfg.RTPMaxPayloadSize)
			if tt.name == "idr" {
				assertEncoderRTPSingleNALCallbackMetadata(t, callbackPackets, callbackMetadata, first, firstFrame, cfg, true, true)
				return
			}
			callbackPackets = callbackPackets[:0]
			callbackMetadata = callbackMetadata[:0]

			secondFrame := tt.nextFrame(firstFrame)
			secondFrame.PTS = firstFrame.PTS + int64(cfg.RTPTimestampIncrement)
			second, err := enc.Encode(secondFrame)
			if err != nil {
				t.Fatalf("Encode mode 1 %s: %v", tt.name, err)
			}
			assertEncoderNALTypes(t, second.NALUnits, tt.wantNALs)
			assertEncoderRTPSingleNALPackets(t, second, cfg.RTPMaxPayloadSize)
			assertEncoderRTPSingleNALCallbackMetadata(t, callbackPackets, callbackMetadata, second, secondFrame, cfg, false, false)
		})
	}
}

func TestEncoderRTPPacketCallbackReceivesPFrameSingleNALMetadata(t *testing.T) {
	for _, tt := range []struct {
		name         string
		prepareFirst func(*goh264.EncoderFrame)
		nextFrame    func(goh264.EncoderFrame) goh264.EncoderFrame
		wantNALs     []uint8
	}{
		{
			name: "p-skip",
			nextFrame: func(first goh264.EncoderFrame) goh264.EncoderFrame {
				return first
			},
			wantNALs: []uint8{1},
		},
		{
			name: "exact-p16x16",
			nextFrame: func(first goh264.EncoderFrame) goh264.EncoderFrame {
				return integerMotionI420EncoderFrame(first, 2, 0)
			},
			wantNALs: []uint8{1},
		},
		{
			name: "odd-exact-p16x16-constant-chroma",
			prepareFirst: func(first *goh264.EncoderFrame) {
				setConstantI420Chroma(first, 128, 64)
			},
			nextFrame: func(first goh264.EncoderFrame) goh264.EncoderFrame {
				return integerMotionI420EncoderFrame(first, 1, 0)
			},
			wantNALs: []uint8{1},
		},
		{
			name: "odd-exact-p16x16-patterned-chroma-fallback",
			nextFrame: func(first goh264.EncoderFrame) goh264.EncoderFrame {
				return integerMotionI420EncoderFrame(first, 1, 0)
			},
			wantNALs: []uint8{6, 1},
		},
		{
			name: "changed-p-intrapcm",
			nextFrame: func(first goh264.EncoderFrame) goh264.EncoderFrame {
				second := patternedI420EncoderFrame(first.Width, first.Height)
				second.Y[0] ^= 0x2d
				return second
			},
			wantNALs: []uint8{6, 1},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cfg := goh264.DefaultEncoderConfig(16, 16)
			cfg.RTPPacketizationMode = goh264.EncoderRTPPacketizationSingleNAL
			cfg.RTPMaxPayloadSize = 1200
			cfg.RTPPayloadType = 105
			cfg.RTPSSRC = 0x10203040
			cfg.DeblockMode = goh264.EncoderDeblockDisabled
			enc, err := goh264.NewEncoder(cfg)
			if err != nil {
				t.Fatalf("NewEncoder mode 0: %v", err)
			}

			var callbackPackets []goh264.EncoderRTPPacket
			var callbackMetadata []goh264.EncoderRTPPacketMetadata
			enc.SetRTPPacketCallback(func(pkt goh264.EncoderRTPPacket, meta goh264.EncoderRTPPacketMetadata) {
				callbackPackets = append(callbackPackets, pkt)
				callbackMetadata = append(callbackMetadata, meta)
			})

			firstFrame := patternedI420EncoderFrame(16, 16)
			if tt.prepareFirst != nil {
				tt.prepareFirst(&firstFrame)
			}
			firstFrame.PTS = 12_000
			first, err := enc.Encode(firstFrame)
			if err != nil {
				t.Fatalf("Encode first mode 0 IDR: %v", err)
			}
			assertEncoderNALTypes(t, first.NALUnits, []uint8{7, 8, 5})
			callbackPackets = callbackPackets[:0]
			callbackMetadata = callbackMetadata[:0]

			secondFrame := tt.nextFrame(firstFrame)
			secondFrame.PTS = firstFrame.PTS + int64(cfg.RTPTimestampIncrement)
			second, err := enc.Encode(secondFrame)
			if err != nil {
				t.Fatalf("Encode mode 0 %s: %v", tt.name, err)
			}
			assertEncoderNALTypes(t, second.NALUnits, tt.wantNALs)
			if len(callbackPackets) != len(second.RTPPackets) || len(callbackMetadata) != len(second.RTPPackets) {
				t.Fatalf("callback packets/meta = %d/%d, want RTP packet count %d",
					len(callbackPackets), len(callbackMetadata), len(second.RTPPackets))
			}
			for i, meta := range callbackMetadata {
				pkt := callbackPackets[i]
				if meta.PacketIndex != i || meta.PacketCount != len(second.RTPPackets) {
					t.Fatalf("callback meta[%d] index/count = %d/%d, want %d/%d",
						i, meta.PacketIndex, meta.PacketCount, i, len(second.RTPPackets))
				}
				if meta.FramePTS != secondFrame.PTS || meta.FrameDTS != secondFrame.PTS ||
					meta.RTPTime != second.RTPTime || meta.KeyFrame || meta.IDR {
					t.Fatalf("callback meta[%d] frame fields = %+v, want non-IDR P-frame timing metadata", i, meta)
				}
				if pkt.SequenceNumber != second.RTPPackets[i].SequenceNumber ||
					pkt.Timestamp != second.RTPPackets[i].Timestamp ||
					pkt.PayloadType != cfg.RTPPayloadType ||
					pkt.SSRC != cfg.RTPSSRC ||
					pkt.Marker != (i == len(second.RTPPackets)-1) ||
					!bytes.Equal(pkt.Payload, second.RTPPackets[i].Payload) ||
					!bytes.Equal(pkt.Data, second.RTPPackets[i].Data) {
					t.Fatalf("callback packet[%d] metadata = %+v, want returned RTP packet fields", i, pkt)
				}
				assertEncoderRTPCallbackPacketDoesNotAliasReturned(t, pkt, second.RTPPackets[i], i)
				if meta.PayloadFormat != goh264.EncoderRTPPayloadSingleNAL ||
					meta.NALUnitType != tt.wantNALs[i] ||
					meta.NALUnitCount != 1 ||
					!meta.StartOfNAL || !meta.EndOfNAL ||
					meta.ParameterSet {
					t.Fatalf("callback meta[%d] = %+v, want complete P-frame single-NAL type %d",
						i, meta, tt.wantNALs[i])
				}
			}
		})
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
	first, err := enc.Encode(patternedI420EncoderFrame(16, 16))
	if err != nil {
		t.Fatalf("Encode RTP before clearing callback: %v", err)
	}
	firstPacketCount := len(first.RTPPackets)
	if firstPacketCount == 0 || calls != firstPacketCount {
		t.Fatalf("initial RTP packets/callbacks = %d/%d, want nonzero matching count", firstPacketCount, calls)
	}
	enc.ForceIDR()
	if !enc.PendingIDR() {
		t.Fatal("ForceIDR before clearing callback did not queue IDR")
	}
	enc.SetRTPPacketCallback(nil)
	if !enc.PendingIDR() {
		t.Fatal("clearing RTP callback cleared pending IDR")
	}
	secondFrame := patternedI420EncoderFrame(16, 16)
	secondFrame.Y[0] ^= 0x22
	second, err := enc.Encode(secondFrame)
	if err != nil {
		t.Fatalf("Encode RTP after clearing callback: %v", err)
	}
	if second.Dropped || !second.IDR || enc.PendingIDR() {
		t.Fatalf("post-clear frame dropped=%v idr=%v pending=%v, want delivered IDR",
			second.Dropped, second.IDR, enc.PendingIDR())
	}
	assertRTPPacketMetadata(t, second.RTPPackets, cfg.RTPPayloadType, cfg.RTPSSRC, uint16(firstPacketCount))
	if calls != firstPacketCount {
		t.Fatalf("cleared callback calls = %d, want still %d", calls, firstPacketCount)
	}

	var firstCallbackCalls, replacementCallbackCalls int
	enc.SetRTPPacketCallback(func(goh264.EncoderRTPPacket, goh264.EncoderRTPPacketMetadata) {
		firstCallbackCalls++
	})
	thirdFrame := secondFrame
	thirdFrame.PTS += int64(cfg.RTPTimestampIncrement)
	third, err := enc.Encode(thirdFrame)
	if err != nil {
		t.Fatalf("Encode RTP before replacing callback: %v", err)
	}
	thirdPacketCount := len(third.RTPPackets)
	if thirdPacketCount == 0 || firstCallbackCalls != thirdPacketCount {
		t.Fatalf("pre-replacement RTP packets/callbacks = %d/%d, want nonzero matching count",
			thirdPacketCount, firstCallbackCalls)
	}
	enc.ForceIDR()
	if !enc.PendingIDR() {
		t.Fatal("ForceIDR before replacing callback did not queue IDR")
	}
	enc.SetRTPPacketCallback(func(goh264.EncoderRTPPacket, goh264.EncoderRTPPacketMetadata) {
		replacementCallbackCalls++
	})
	if !enc.PendingIDR() {
		t.Fatal("replacing RTP callback cleared pending IDR")
	}
	fourthFrame := thirdFrame
	fourthFrame.PTS += int64(cfg.RTPTimestampIncrement)
	fourthFrame.Y[0] ^= 0x44
	fourth, err := enc.Encode(fourthFrame)
	if err != nil {
		t.Fatalf("Encode RTP after replacing callback: %v", err)
	}
	if fourth.Dropped || !fourth.IDR || enc.PendingIDR() {
		t.Fatalf("post-replace frame dropped=%v idr=%v pending=%v, want delivered IDR",
			fourth.Dropped, fourth.IDR, enc.PendingIDR())
	}
	assertRTPPacketMetadata(t, fourth.RTPPackets, cfg.RTPPayloadType, cfg.RTPSSRC, uint16(firstPacketCount+len(second.RTPPackets)+thirdPacketCount))
	if firstCallbackCalls != thirdPacketCount {
		t.Fatalf("replaced callback calls = %d, want still %d", firstCallbackCalls, thirdPacketCount)
	}
	if replacementCallbackCalls != len(fourth.RTPPackets) {
		t.Fatalf("replacement callback calls = %d, want forced-IDR packet count %d", replacementCallbackCalls, len(fourth.RTPPackets))
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
	cfg := goh264.DefaultEncoderConfig(16, 16)
	cfg.DeblockMode = goh264.EncoderDeblockDisabled
	cfg.RTPMaxPayloadSize = 32
	enc, err := goh264.NewEncoder(cfg)
	if err != nil {
		t.Fatalf("NewEncoder: %v", err)
	}
	var callbackCalls int
	enc.SetRTPPacketCallback(func(goh264.EncoderRTPPacket, goh264.EncoderRTPPacketMetadata) {
		callbackCalls++
	})

	frame := patternedI420EncoderFrame(16, 16)
	frame.PTS = 0
	first, err := enc.EncodeInto(make([]byte, 0, 4096), frame)
	if err != nil {
		t.Fatalf("EncodeInto valid frame: %v", err)
	}
	if !first.IDR || first.RTPTime != 0 {
		t.Fatalf("first valid output IDR/time = %v/%d, want IDR/0", first.IDR, first.RTPTime)
	}
	assertEncoderNALTypes(t, first.NALUnits, []uint8{7, 8, 5})
	assertRTPPacketMetadata(t, first.RTPPackets, cfg.RTPPayloadType, cfg.RTPSSRC, 0)
	firstPacketCount := len(first.RTPPackets)
	if firstPacketCount == 0 || callbackCalls != firstPacketCount {
		t.Fatalf("first RTP packets/callbacks = %d/%d, want nonzero matching count",
			firstPacketCount, callbackCalls)
	}

	bad := frame
	bad.PTS = int64(cfg.RTPTimestampIncrement)
	bad.Y = nil
	if out, err := enc.EncodeInto(make([]byte, 0, 4096), bad); !errors.Is(err, goh264.ErrInvalidData) {
		t.Fatalf("EncodeInto missing luma error = %v, want ErrInvalidData", err)
	} else if out.Dropped || len(out.Data) != 0 || len(out.NALUnits) != 0 || len(out.RTPPackets) != 0 {
		t.Fatalf("invalid missing-luma output = %+v, want empty output", out)
	}

	bad = frame
	bad.PTS = int64(cfg.RTPTimestampIncrement)
	bad.Width = 32
	if out, err := enc.Encode(bad); !errors.Is(err, goh264.ErrInvalidData) {
		t.Fatalf("Encode mismatched dimensions error = %v, want ErrInvalidData", err)
	} else if out.Dropped || len(out.Data) != 0 || len(out.NALUnits) != 0 || len(out.RTPPackets) != 0 {
		t.Fatalf("invalid dimension output = %+v, want empty output", out)
	}
	bad = frame
	bad.PTS = int64(^uint32(0)) + 1
	if out, err := enc.Encode(bad); !errors.Is(err, goh264.ErrInvalidData) {
		t.Fatalf("Encode overflowed PTS error = %v, want ErrInvalidData", err)
	} else if out.Dropped || len(out.Data) != 0 || len(out.NALUnits) != 0 || len(out.RTPPackets) != 0 {
		t.Fatalf("invalid PTS output = %+v, want empty output", out)
	}
	bad.PTS = -1
	if out, err := enc.Encode(bad); !errors.Is(err, goh264.ErrInvalidData) {
		t.Fatalf("Encode negative PTS error = %v, want ErrInvalidData", err)
	} else if out.Dropped || len(out.Data) != 0 || len(out.NALUnits) != 0 || len(out.RTPPackets) != 0 {
		t.Fatalf("negative PTS output = %+v, want empty output", out)
	}
	bad = frame
	bad.PTS = int64(cfg.RTPTimestampIncrement)
	bad.Duration = int64(^uint32(0)) + 1
	if out, err := enc.Encode(bad); !errors.Is(err, goh264.ErrInvalidData) {
		t.Fatalf("Encode overflowed duration error = %v, want ErrInvalidData", err)
	} else if out.Dropped || len(out.Data) != 0 || len(out.NALUnits) != 0 || len(out.RTPPackets) != 0 {
		t.Fatalf("invalid duration output = %+v, want empty output", out)
	}
	bad.Duration = -1
	if out, err := enc.Encode(bad); !errors.Is(err, goh264.ErrInvalidData) {
		t.Fatalf("Encode negative duration error = %v, want ErrInvalidData", err)
	} else if out.Dropped || len(out.Data) != 0 || len(out.NALUnits) != 0 || len(out.RTPPackets) != 0 {
		t.Fatalf("negative duration output = %+v, want empty output", out)
	}
	bad = frame
	bad.PTS = int64(cfg.RTPTimestampIncrement)
	bad.ForceIDR = true
	bad.Color.SARNum = 1
	invalidForceIDRDst, invalidForceIDRBefore := encoderPrefilledCallerBuffer()
	if out, err := enc.EncodeInto(invalidForceIDRDst, bad); !errors.Is(err, goh264.ErrInvalidData) {
		t.Fatalf("EncodeInto invalid ForceIDR frame color error = %v, want ErrInvalidData", err)
	} else if out.Dropped || len(out.Data) != 0 || len(out.NALUnits) != 0 || len(out.RTPPackets) != 0 {
		t.Fatalf("invalid ForceIDR frame color output = %+v, want empty output", out)
	}
	assertEncoderCallerBufferUnchanged(t, invalidForceIDRDst, invalidForceIDRBefore)
	if callbackCalls != firstPacketCount {
		t.Fatalf("invalid frames invoked callbacks = %d, want still %d", callbackCalls, firstPacketCount)
	}

	secondFrame := frame
	secondFrame.PTS = int64(cfg.RTPTimestampIncrement)
	second, err := enc.EncodeInto(make([]byte, 0, 4096), secondFrame)
	if err != nil {
		t.Fatalf("EncodeInto after invalid frames: %v", err)
	}
	if second.Dropped || second.IDR || second.RTPTime != uint32(secondFrame.PTS) {
		t.Fatalf("post-invalid output dropped/id/time = %v/%v/%d, want P-skip time %d",
			second.Dropped, second.IDR, second.RTPTime, secondFrame.PTS)
	}
	assertEncoderNALTypes(t, second.NALUnits, []uint8{1})
	assertEncoderVCLFrameNums(t, append(append([]byte(nil), first.Data...), second.Data...), []uint8{5, 1}, []uint32{0, 1})
	assertRTPPacketMetadata(t, second.RTPPackets, cfg.RTPPayloadType, cfg.RTPSSRC, uint16(firstPacketCount))
	if callbackCalls != firstPacketCount+len(second.RTPPackets) {
		t.Fatalf("post-invalid callbacks = %d, want %d", callbackCalls, firstPacketCount+len(second.RTPPackets))
	}
}

func TestEncoderEncodeIntoInvalidFramePreservesPendingIDR(t *testing.T) {
	cfg := goh264.DefaultEncoderConfig(16, 16)
	cfg.DeblockMode = goh264.EncoderDeblockDisabled
	cfg.RTPMaxPayloadSize = 32
	enc, err := goh264.NewEncoder(cfg)
	if err != nil {
		t.Fatalf("NewEncoder: %v", err)
	}
	var callbackCalls int
	enc.SetRTPPacketCallback(func(goh264.EncoderRTPPacket, goh264.EncoderRTPPacketMetadata) {
		callbackCalls++
	})

	frame := patternedI420EncoderFrame(16, 16)
	first, err := enc.EncodeInto(make([]byte, 0, 4096), frame)
	if err != nil {
		t.Fatalf("EncodeInto first IDR: %v", err)
	}
	if !first.IDR || enc.PendingIDR() {
		t.Fatalf("first frame idr=%v pending=%v, want completed IDR", first.IDR, enc.PendingIDR())
	}
	firstPacketCount := len(first.RTPPackets)
	if firstPacketCount == 0 || callbackCalls != firstPacketCount {
		t.Fatalf("first RTP packets/callbacks = %d/%d, want nonzero matching count",
			firstPacketCount, callbackCalls)
	}

	tests := []struct {
		name   string
		mutate func(*goh264.EncoderFrame)
	}{
		{name: "missing luma", mutate: func(f *goh264.EncoderFrame) { f.Y = nil }},
		{name: "mismatched width", mutate: func(f *goh264.EncoderFrame) { f.Width = 32 }},
		{name: "invalid frame color", mutate: func(f *goh264.EncoderFrame) { f.Color.SARNum = 1 }},
		{name: "negative pts", mutate: func(f *goh264.EncoderFrame) { f.PTS = -1 }},
		{name: "overflow duration", mutate: func(f *goh264.EncoderFrame) { f.Duration = int64(^uint32(0)) + 1 }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enc.ForceIDR()
			if !enc.PendingIDR() {
				t.Fatalf("%s ForceIDR did not queue IDR", tt.name)
			}
			beforeCfg := enc.Config()
			bad := frame
			bad.PTS = int64(cfg.RTPTimestampIncrement)
			tt.mutate(&bad)

			dst, beforeDst := encoderPrefilledCallerBuffer()
			out, err := enc.EncodeInto(dst, bad)
			if !errors.Is(err, goh264.ErrInvalidData) {
				t.Fatalf("%s EncodeInto error = %v, want ErrInvalidData", tt.name, err)
			}
			if out.Dropped || len(out.Data) != 0 || len(out.NALUnits) != 0 || len(out.RTPPackets) != 0 {
				t.Fatalf("%s invalid output = %+v, want empty output", tt.name, out)
			}
			assertEncoderCallerBufferUnchanged(t, dst, beforeDst)
			if got := enc.Config(); got != beforeCfg {
				t.Fatalf("%s invalid EncodeInto mutated config = %+v, want %+v", tt.name, got, beforeCfg)
			}
			if !enc.PendingIDR() {
				t.Fatalf("%s invalid EncodeInto cleared pending IDR", tt.name)
			}
			if callbackCalls != firstPacketCount {
				t.Fatalf("%s invalid EncodeInto callbacks = %d, want still %d",
					tt.name, callbackCalls, firstPacketCount)
			}

			next := frame
			next.PTS = int64(cfg.RTPTimestampIncrement)
			next.Y = append([]byte(nil), frame.Y...)
			next.Y[0] ^= 0x55
			second, err := enc.EncodeInto(make([]byte, 0, 4096), next)
			if err != nil {
				t.Fatalf("%s EncodeInto after invalid frame: %v", tt.name, err)
			}
			if second.Dropped || !second.IDR || enc.PendingIDR() {
				t.Fatalf("%s post-invalid output dropped=%v idr=%v pending=%v, want delivered IDR",
					tt.name, second.Dropped, second.IDR, enc.PendingIDR())
			}
			assertEncoderNALTypes(t, second.NALUnits, []uint8{7, 8, 5})
			firstPacketCount = callbackCalls
		})
	}
}

func TestEncoderEncodeIntoOverflowedDestinationPreservesPendingIDRAndLiveState(t *testing.T) {
	for _, tt := range []struct {
		name   string
		format goh264.EncoderOutputFormat
	}{
		{name: "annexb", format: goh264.EncoderOutputAnnexB},
		{name: "avc", format: goh264.EncoderOutputAVC},
		{name: "rtp", format: goh264.EncoderOutputRTP},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cfg := goh264.DefaultEncoderConfig(16, 16)
			cfg.DeblockMode = goh264.EncoderDeblockDisabled
			cfg.OutputFormat = tt.format
			if cfg.OutputFormat == goh264.EncoderOutputRTP {
				cfg.RTPMaxPayloadSize = 32
			} else {
				cfg.RTPMaxPayloadSize = 0
			}
			enc, err := goh264.NewEncoder(cfg)
			if err != nil {
				t.Fatalf("NewEncoder: %v", err)
			}
			var callbackCalls int
			enc.SetRTPPacketCallback(func(goh264.EncoderRTPPacket, goh264.EncoderRTPPacketMetadata) {
				callbackCalls++
			})

			frame := patternedI420EncoderFrame(16, 16)
			first, err := enc.EncodeInto(make([]byte, 0, 4096), frame)
			if err != nil {
				t.Fatalf("EncodeInto first IDR: %v", err)
			}
			if !first.IDR || enc.PendingIDR() {
				t.Fatalf("first frame idr=%v pending=%v, want completed IDR", first.IDR, enc.PendingIDR())
			}
			firstPacketCount := len(first.RTPPackets)
			if cfg.OutputFormat == goh264.EncoderOutputRTP {
				if firstPacketCount == 0 || callbackCalls != firstPacketCount {
					t.Fatalf("first RTP packets/callbacks = %d/%d, want nonzero matching count",
						firstPacketCount, callbackCalls)
				}
			} else if firstPacketCount != 0 || callbackCalls != 0 {
				t.Fatalf("non-RTP first packets/callbacks = %d/%d, want none", firstPacketCount, callbackCalls)
			}
			beforeCfg := enc.Config()

			enc.ForceIDR()
			if !enc.PendingIDR() {
				t.Fatal("ForceIDR did not queue IDR before overflowed EncodeInto")
			}
			overflowDst := fakeDecoderRawBytesLen(maxIntForTest - 3)
			forcedFrame := frame
			forcedFrame.PTS = int64(cfg.RTPTimestampIncrement)
			out, err := enc.EncodeInto(overflowDst, forcedFrame)
			if !errors.Is(err, goh264.ErrInvalidData) {
				t.Fatalf("overflowed EncodeInto error = %v, want ErrInvalidData", err)
			}
			if out.Dropped || len(out.Data) != 0 || len(out.NALUnits) != 0 || len(out.RTPPackets) != 0 {
				t.Fatalf("overflowed EncodeInto output = %+v, want empty output", out)
			}
			if got := enc.Config(); got != beforeCfg {
				t.Fatalf("overflowed EncodeInto mutated config = %+v, want %+v", got, beforeCfg)
			}
			if !enc.PendingIDR() {
				t.Fatal("overflowed EncodeInto consumed pending IDR")
			}
			if callbackCalls != firstPacketCount {
				t.Fatalf("overflowed EncodeInto callbacks = %d, want still %d", callbackCalls, firstPacketCount)
			}

			recoveredFrame := frame
			recoveredFrame.PTS = int64(cfg.RTPTimestampIncrement)
			recoveredFrame.Y = append([]byte(nil), frame.Y...)
			recoveredFrame.Y[0] ^= 0x55
			recovered, err := enc.EncodeInto(make([]byte, 0, 4096), recoveredFrame)
			if err != nil {
				t.Fatalf("EncodeInto after overflowed destination: %v", err)
			}
			if recovered.Dropped || !recovered.IDR || enc.PendingIDR() {
				t.Fatalf("post-overflow output dropped=%v idr=%v pending=%v, want delivered IDR",
					recovered.Dropped, recovered.IDR, enc.PendingIDR())
			}
			if recovered.RTPTime != uint32(recoveredFrame.PTS) {
				t.Fatalf("post-overflow RTP time = %d, want %d", recovered.RTPTime, recoveredFrame.PTS)
			}
			assertEncoderNALTypes(t, recovered.NALUnits, []uint8{7, 8, 5})
			if cfg.OutputFormat == goh264.EncoderOutputRTP {
				assertRTPPacketMetadata(t, recovered.RTPPackets, cfg.RTPPayloadType, cfg.RTPSSRC, uint16(firstPacketCount))
			} else if len(recovered.RTPPackets) != 0 {
				t.Fatalf("non-RTP recovered packets = %d, want none", len(recovered.RTPPackets))
			}
			if callbackCalls != firstPacketCount+len(recovered.RTPPackets) {
				t.Fatalf("post-overflow callbacks = %d, want %d",
					callbackCalls, firstPacketCount+len(recovered.RTPPackets))
			}
		})
	}
}

func TestEncoderEncodeIntoPFrameOverflowedDestinationPreservesLiveState(t *testing.T) {
	for _, tt := range []struct {
		name   string
		format goh264.EncoderOutputFormat
	}{
		{name: "annexb", format: goh264.EncoderOutputAnnexB},
		{name: "avc", format: goh264.EncoderOutputAVC},
		{name: "rtp", format: goh264.EncoderOutputRTP},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cfg := goh264.DefaultEncoderConfig(16, 16)
			cfg.DeblockMode = goh264.EncoderDeblockDisabled
			cfg.OutputFormat = tt.format
			if cfg.OutputFormat == goh264.EncoderOutputRTP {
				cfg.RTPMaxPayloadSize = 32
			} else {
				cfg.RTPMaxPayloadSize = 0
			}
			enc, err := goh264.NewEncoder(cfg)
			if err != nil {
				t.Fatalf("NewEncoder: %v", err)
			}
			var callbackCalls int
			enc.SetRTPPacketCallback(func(goh264.EncoderRTPPacket, goh264.EncoderRTPPacketMetadata) {
				callbackCalls++
			})

			frame := patternedI420EncoderFrame(16, 16)
			first, err := enc.EncodeInto(make([]byte, 0, 4096), frame)
			if err != nil {
				t.Fatalf("EncodeInto first IDR: %v", err)
			}
			if !first.IDR || enc.PendingIDR() {
				t.Fatalf("first frame idr=%v pending=%v, want completed IDR", first.IDR, enc.PendingIDR())
			}
			firstPacketCount := len(first.RTPPackets)
			if cfg.OutputFormat == goh264.EncoderOutputRTP {
				if firstPacketCount == 0 || callbackCalls != firstPacketCount {
					t.Fatalf("first RTP packets/callbacks = %d/%d, want nonzero matching count",
						firstPacketCount, callbackCalls)
				}
			} else if firstPacketCount != 0 || callbackCalls != 0 {
				t.Fatalf("non-RTP first packets/callbacks = %d/%d, want none", firstPacketCount, callbackCalls)
			}
			beforeCfg := enc.Config()

			pFrame := frame
			pFrame.PTS = int64(cfg.RTPTimestampIncrement)
			out, err := enc.EncodeInto(fakeDecoderRawBytesLen(maxIntForTest-3), pFrame)
			if !errors.Is(err, goh264.ErrInvalidData) {
				t.Fatalf("overflowed P-frame EncodeInto error = %v, want ErrInvalidData", err)
			}
			if out.Dropped || len(out.Data) != 0 || len(out.NALUnits) != 0 || len(out.RTPPackets) != 0 {
				t.Fatalf("overflowed P-frame EncodeInto output = %+v, want empty output", out)
			}
			if got := enc.Config(); got != beforeCfg {
				t.Fatalf("overflowed P-frame EncodeInto mutated config = %+v, want %+v", got, beforeCfg)
			}
			if enc.PendingIDR() {
				t.Fatal("overflowed P-frame EncodeInto queued unexpected IDR")
			}
			if callbackCalls != firstPacketCount {
				t.Fatalf("overflowed P-frame callbacks = %d, want still %d", callbackCalls, firstPacketCount)
			}

			recovered, err := enc.EncodeInto(make([]byte, 0, 4096), pFrame)
			if err != nil {
				t.Fatalf("EncodeInto after overflowed P-frame destination: %v", err)
			}
			if recovered.Dropped || recovered.IDR || enc.PendingIDR() {
				t.Fatalf("post-overflow P-frame output dropped=%v idr=%v pending=%v, want delivered P-skip",
					recovered.Dropped, recovered.IDR, enc.PendingIDR())
			}
			if recovered.RTPTime != uint32(pFrame.PTS) {
				t.Fatalf("post-overflow P-frame RTP time = %d, want %d", recovered.RTPTime, pFrame.PTS)
			}
			assertEncoderNALTypes(t, recovered.NALUnits, []uint8{1})
			assertEncodedFrameNALUnitIndexes(t, recovered, cfg.OutputFormat)
			if cfg.OutputFormat != goh264.EncoderOutputAVC {
				assertEncoderVCLFrameNums(t, append(append([]byte(nil), first.Data...), recovered.Data...), []uint8{5, 1}, []uint32{0, 1})
			}
			if cfg.OutputFormat == goh264.EncoderOutputRTP {
				assertRTPPacketMetadata(t, recovered.RTPPackets, cfg.RTPPayloadType, cfg.RTPSSRC, uint16(firstPacketCount))
			} else if len(recovered.RTPPackets) != 0 {
				t.Fatalf("non-RTP recovered P-frame packets = %d, want none", len(recovered.RTPPackets))
			}
			if callbackCalls != firstPacketCount+len(recovered.RTPPackets) {
				t.Fatalf("post-overflow P-frame callbacks = %d, want %d",
					callbackCalls, firstPacketCount+len(recovered.RTPPackets))
			}
		})
	}
}

func assertEncoderEncodeIntoOddPatternedChromaFallbackAllocationCanary(t *testing.T, cfg goh264.EncoderConfig, label string, wantRTPPackets int, maxAllocs float64) {
	t.Helper()
	cfg.DeblockMode = goh264.EncoderDeblockDisabled
	cfg.GOPSize = 10000
	cfg.IDRInterval = 10000
	if cfg.OutputFormat != goh264.EncoderOutputRTP {
		cfg.RTPMaxPayloadSize = 0
	}
	a := patternedI420EncoderFrame(16, 16)
	b := integerMotionI420EncoderFrame(a, 1, 0)
	encs := primedI420EncoderPool(t, cfg, a, 128)
	dst := make([]byte, 0, 4096)
	var call int
	allocs := testing.AllocsPerRun(100, func() {
		if call >= len(encs) {
			t.Fatalf("encoder pool exhausted after %d calls", call)
		}
		out, err := encs[call].EncodeInto(dst[:0], b)
		call++
		if err != nil {
			t.Fatalf("EncodeInto %s odd patterned-chroma fallback: %v", label, err)
		}
		if out.IDR || len(out.RTPPackets) != wantRTPPackets || len(out.Data) == 0 ||
			len(out.NALUnits) != 2 || out.NALUnits[0].Type != 6 || out.NALUnits[1].Type != 1 {
			t.Fatalf("%s odd patterned-chroma fallback output idr=%v rtp=%d data=%d nals=%+v",
				label, out.IDR, len(out.RTPPackets), len(out.Data), out.NALUnits)
		}
		if cap(out.Data) != cap(dst) {
			t.Fatalf("EncodeInto did not reuse caller output capacity: got cap %d want %d", cap(out.Data), cap(dst))
		}
	})
	t.Logf("%s odd patterned-chroma fallback EncodeInto allocations/run = %.0f", label, allocs)
	if allocs > maxAllocs {
		t.Fatalf("%s odd patterned-chroma fallback EncodeInto allocations/run = %.0f, want <= %.0f", label, allocs, maxAllocs)
	}
}

func assertEncoderEncodeIntoPerMacroblockExactP16x16AllocationCanary(t *testing.T, cfg goh264.EncoderConfig, label string, wantRTPPackets int, maxAllocs float64) {
	t.Helper()
	cfg.DeblockMode = goh264.EncoderDeblockDisabled
	cfg.SliceCount = 2
	cfg.GOPSize = 10000
	cfg.IDRInterval = 10000
	if cfg.OutputFormat != goh264.EncoderOutputRTP {
		cfg.RTPMaxPayloadSize = 0
	}
	a := patternedI420EncoderFrame(32, 32)
	b := perMacroblockMotionI420EncoderFrame(a, []encoderTestMotion{
		{dx: 2, dy: 0},
		{dx: -2, dy: 0},
		{dx: 0, dy: 2},
		{dx: 0, dy: -2},
	})
	encs := primedI420EncoderPool(t, cfg, a, 128)
	dst := make([]byte, 0, 4096)
	var call int
	allocs := testing.AllocsPerRun(100, func() {
		if call >= len(encs) {
			t.Fatalf("encoder pool exhausted after %d calls", call)
		}
		out, err := encs[call].EncodeInto(dst[:0], b)
		call++
		if err != nil {
			t.Fatalf("EncodeInto %s per-macroblock exact P16x16: %v", label, err)
		}
		if out.IDR || len(out.RTPPackets) != wantRTPPackets || len(out.Data) == 0 ||
			len(out.NALUnits) != 2 || out.NALUnits[0].Type != 1 || out.NALUnits[1].Type != 1 {
			t.Fatalf("%s per-macroblock exact P16x16 output idr=%v rtp=%d data=%d nals=%+v",
				label, out.IDR, len(out.RTPPackets), len(out.Data), out.NALUnits)
		}
		if cap(out.Data) != cap(dst) {
			t.Fatalf("EncodeInto did not reuse caller output capacity: got cap %d want %d", cap(out.Data), cap(dst))
		}
	})
	t.Logf("%s per-macroblock exact P16x16 EncodeInto allocations/run = %.0f", label, allocs)
	if allocs > maxAllocs {
		t.Fatalf("%s per-macroblock exact P16x16 EncodeInto allocations/run = %.0f, want <= %.0f", label, allocs, maxAllocs)
	}
}

func TestEncoderEncodeIntoAllocationCanary(t *testing.T) {
	t.Run("annexb forced idr", func(t *testing.T) {
		cfg := goh264.DefaultEncoderConfig(16, 16)
		cfg.OutputFormat = goh264.EncoderOutputAnnexB
		cfg.DeblockMode = goh264.EncoderDeblockDisabled
		cfg.RTPMaxPayloadSize = 0
		cfg.GOPSize = 10000
		cfg.IDRInterval = 10000
		enc, err := goh264.NewEncoder(cfg)
		if err != nil {
			t.Fatalf("NewEncoder: %v", err)
		}
		frame := patternedI420EncoderFrame(16, 16)
		if _, err := enc.EncodeInto(make([]byte, 0, 4096), frame); err != nil {
			t.Fatalf("prime IDR: %v", err)
		}
		dst := make([]byte, 0, 4096)
		allocs := testing.AllocsPerRun(100, func() {
			enc.ForceIDR()
			out, err := enc.EncodeInto(dst[:0], frame)
			if err != nil {
				t.Fatalf("EncodeInto forced IDR: %v", err)
			}
			if !out.IDR || len(out.RTPPackets) != 0 || len(out.Data) == 0 {
				t.Fatalf("forced IDR output idr=%v rtp=%d data=%d", out.IDR, len(out.RTPPackets), len(out.Data))
			}
			if cap(out.Data) != cap(dst) {
				t.Fatalf("EncodeInto did not reuse caller output capacity: got cap %d want %d", cap(out.Data), cap(dst))
			}
		})
		t.Logf("annexb forced IDR EncodeInto allocations/run = %.0f", allocs)
		if allocs > 8 {
			t.Fatalf("annexb forced IDR EncodeInto allocations/run = %.0f, want <= 8", allocs)
		}
	})

	t.Run("annexb steady p-skip", func(t *testing.T) {
		cfg := goh264.DefaultEncoderConfig(16, 16)
		cfg.OutputFormat = goh264.EncoderOutputAnnexB
		cfg.DeblockMode = goh264.EncoderDeblockDisabled
		cfg.RTPMaxPayloadSize = 0
		cfg.GOPSize = 10000
		cfg.IDRInterval = 10000
		enc, err := goh264.NewEncoder(cfg)
		if err != nil {
			t.Fatalf("NewEncoder: %v", err)
		}
		frame := patternedI420EncoderFrame(16, 16)
		if _, err := enc.EncodeInto(make([]byte, 0, 4096), frame); err != nil {
			t.Fatalf("prime IDR: %v", err)
		}
		dst := make([]byte, 0, 4096)
		allocs := testing.AllocsPerRun(100, func() {
			out, err := enc.EncodeInto(dst[:0], frame)
			if err != nil {
				t.Fatalf("EncodeInto P-skip: %v", err)
			}
			if out.IDR || len(out.RTPPackets) != 0 || len(out.Data) == 0 {
				t.Fatalf("steady P-skip output idr=%v rtp=%d data=%d", out.IDR, len(out.RTPPackets), len(out.Data))
			}
			if cap(out.Data) != cap(dst) {
				t.Fatalf("EncodeInto did not reuse caller output capacity: got cap %d want %d", cap(out.Data), cap(dst))
			}
		})
		t.Logf("annexb steady p-skip EncodeInto allocations/run = %.0f", allocs)
		if allocs > 4 {
			t.Fatalf("annexb steady P-skip EncodeInto allocations/run = %.0f, want <= 4", allocs)
		}
	})

	t.Run("annexb exact p16x16", func(t *testing.T) {
		cfg := goh264.DefaultEncoderConfig(16, 16)
		cfg.OutputFormat = goh264.EncoderOutputAnnexB
		cfg.DeblockMode = goh264.EncoderDeblockDisabled
		cfg.RTPMaxPayloadSize = 0
		cfg.GOPSize = 10000
		cfg.IDRInterval = 10000
		a := patternedI420EncoderFrame(16, 16)
		b := integerMotionI420EncoderFrame(a, 2, 0)
		encs := primedI420EncoderPool(t, cfg, a, 128)
		dst := make([]byte, 0, 4096)
		var call int
		allocs := testing.AllocsPerRun(100, func() {
			if call >= len(encs) {
				t.Fatalf("encoder pool exhausted after %d calls", call)
			}
			out, err := encs[call].EncodeInto(dst[:0], b)
			call++
			if err != nil {
				t.Fatalf("EncodeInto exact P16x16: %v", err)
			}
			if out.IDR || len(out.RTPPackets) != 0 || len(out.Data) == 0 ||
				len(out.NALUnits) != 1 || out.NALUnits[0].Type != 1 {
				t.Fatalf("exact P16x16 output idr=%v rtp=%d data=%d nals=%+v",
					out.IDR, len(out.RTPPackets), len(out.Data), out.NALUnits)
			}
			if cap(out.Data) != cap(dst) {
				t.Fatalf("EncodeInto did not reuse caller output capacity: got cap %d want %d", cap(out.Data), cap(dst))
			}
		})
		t.Logf("annexb exact P16x16 EncodeInto allocations/run = %.0f", allocs)
		if allocs > 4 {
			t.Fatalf("annexb exact P16x16 EncodeInto allocations/run = %.0f, want <= 4", allocs)
		}
	})

	t.Run("annexb odd exact p16x16 constant chroma", func(t *testing.T) {
		cfg := goh264.DefaultEncoderConfig(16, 16)
		cfg.OutputFormat = goh264.EncoderOutputAnnexB
		cfg.DeblockMode = goh264.EncoderDeblockDisabled
		cfg.RTPMaxPayloadSize = 0
		cfg.GOPSize = 10000
		cfg.IDRInterval = 10000
		a := patternedI420EncoderFrame(16, 16)
		setConstantI420Chroma(&a, 128, 64)
		b := integerMotionI420EncoderFrame(a, 1, 0)
		encs := primedI420EncoderPool(t, cfg, a, 128)
		dst := make([]byte, 0, 4096)
		var call int
		allocs := testing.AllocsPerRun(100, func() {
			if call >= len(encs) {
				t.Fatalf("encoder pool exhausted after %d calls", call)
			}
			out, err := encs[call].EncodeInto(dst[:0], b)
			call++
			if err != nil {
				t.Fatalf("EncodeInto odd exact P16x16: %v", err)
			}
			if out.IDR || len(out.RTPPackets) != 0 || len(out.Data) == 0 ||
				len(out.NALUnits) != 1 || out.NALUnits[0].Type != 1 {
				t.Fatalf("odd exact P16x16 output idr=%v rtp=%d data=%d nals=%+v",
					out.IDR, len(out.RTPPackets), len(out.Data), out.NALUnits)
			}
			if cap(out.Data) != cap(dst) {
				t.Fatalf("EncodeInto did not reuse caller output capacity: got cap %d want %d", cap(out.Data), cap(dst))
			}
		})
		t.Logf("annexb odd exact P16x16 constant-chroma EncodeInto allocations/run = %.0f", allocs)
		if allocs > 4 {
			t.Fatalf("annexb odd exact P16x16 constant-chroma EncodeInto allocations/run = %.0f, want <= 4", allocs)
		}
	})

	t.Run("annexb odd patterned-chroma fallback", func(t *testing.T) {
		cfg := goh264.DefaultEncoderConfig(16, 16)
		cfg.OutputFormat = goh264.EncoderOutputAnnexB
		assertEncoderEncodeIntoOddPatternedChromaFallbackAllocationCanary(t, cfg, "annexb", 0, 6)
	})

	t.Run("annexb exact p16x16 deblock controls", func(t *testing.T) {
		for _, tt := range []struct {
			name    string
			deblock goh264.EncoderDeblockMode
		}{
			{name: "enabled", deblock: goh264.EncoderDeblockEnabled},
			{name: "slice-boundary", deblock: goh264.EncoderDeblockSliceBoundary},
		} {
			t.Run(tt.name, func(t *testing.T) {
				cfg := goh264.DefaultEncoderConfig(16, 16)
				cfg.OutputFormat = goh264.EncoderOutputAnnexB
				cfg.DeblockMode = tt.deblock
				cfg.RTPMaxPayloadSize = 0
				cfg.GOPSize = 10000
				cfg.IDRInterval = 10000
				a := patternedI420EncoderFrame(16, 16)
				b := integerMotionI420EncoderFrame(a, 2, 0)
				encs := primedI420EncoderPool(t, cfg, a, 128)
				dst := make([]byte, 0, 4096)
				var call int
				allocs := testing.AllocsPerRun(100, func() {
					if call >= len(encs) {
						t.Fatalf("encoder pool exhausted after %d calls", call)
					}
					out, err := encs[call].EncodeInto(dst[:0], b)
					call++
					if err != nil {
						t.Fatalf("EncodeInto exact P16x16 deblock %s: %v", tt.name, err)
					}
					if out.IDR || len(out.RTPPackets) != 0 || len(out.Data) == 0 ||
						len(out.NALUnits) != 1 || out.NALUnits[0].Type != 1 {
						t.Fatalf("exact P16x16 deblock %s output idr=%v rtp=%d data=%d nals=%+v",
							tt.name, out.IDR, len(out.RTPPackets), len(out.Data), out.NALUnits)
					}
					if cap(out.Data) != cap(dst) {
						t.Fatalf("EncodeInto did not reuse caller output capacity: got cap %d want %d", cap(out.Data), cap(dst))
					}
				})
				t.Logf("annexb exact P16x16 deblock %s EncodeInto allocations/run = %.0f", tt.name, allocs)
				if allocs > 4 {
					t.Fatalf("annexb exact P16x16 deblock %s EncodeInto allocations/run = %.0f, want <= 4", tt.name, allocs)
				}
			})
		}
	})

	t.Run("annexb exact p16x16 macroblock-aligned", func(t *testing.T) {
		cfg := goh264.DefaultEncoderConfig(160, 128)
		cfg.OutputFormat = goh264.EncoderOutputAnnexB
		cfg.DeblockMode = goh264.EncoderDeblockDisabled
		cfg.RTPMaxPayloadSize = 0
		cfg.GOPSize = 10000
		cfg.IDRInterval = 10000
		a := patternedI420EncoderFrame(160, 128)
		b := integerMotionI420EncoderFrame(a, 2, 0)
		c := integerMotionI420EncoderFrame(b, 2, 0)
		encs := make([]*goh264.Encoder, 128)
		for i := range encs {
			enc, err := goh264.NewEncoder(cfg)
			if err != nil {
				t.Fatalf("NewEncoder[%d]: %v", i, err)
			}
			if _, err := enc.EncodeInto(make([]byte, 0, 65536), a); err != nil {
				t.Fatalf("prime IDR[%d]: %v", i, err)
			}
			if _, err := enc.EncodeInto(make([]byte, 0, 65536), b); err != nil {
				t.Fatalf("prime exact P16x16[%d]: %v", i, err)
			}
			encs[i] = enc
		}
		dst := make([]byte, 0, 65536)
		var call int
		allocs := testing.AllocsPerRun(100, func() {
			if call >= len(encs) {
				t.Fatalf("encoder pool exhausted after %d calls", call)
			}
			out, err := encs[call].EncodeInto(dst[:0], c)
			call++
			if err != nil {
				t.Fatalf("EncodeInto macroblock-aligned exact P16x16: %v", err)
			}
			if out.IDR || len(out.RTPPackets) != 0 || len(out.Data) == 0 ||
				len(out.NALUnits) != 1 || out.NALUnits[0].Type != 1 {
				t.Fatalf("macroblock-aligned exact P16x16 output idr=%v rtp=%d data=%d nals=%+v",
					out.IDR, len(out.RTPPackets), len(out.Data), out.NALUnits)
			}
			if cap(out.Data) != cap(dst) {
				t.Fatalf("EncodeInto did not reuse caller output capacity: got cap %d want %d", cap(out.Data), cap(dst))
			}
		})
		t.Logf("annexb macroblock-aligned exact P16x16 EncodeInto allocations/run = %.0f", allocs)
		if allocs > 4 {
			t.Fatalf("annexb macroblock-aligned exact P16x16 EncodeInto allocations/run = %.0f, want <= 4", allocs)
		}
	})

	t.Run("annexb per-macroblock exact p16x16", func(t *testing.T) {
		cfg := goh264.DefaultEncoderConfig(32, 32)
		cfg.OutputFormat = goh264.EncoderOutputAnnexB
		assertEncoderEncodeIntoPerMacroblockExactP16x16AllocationCanary(t, cfg, "annexb", 0, 5)
	})

	t.Run("annexb exact p16x16 edge search", func(t *testing.T) {
		cfg := goh264.DefaultEncoderConfig(48, 48)
		cfg.OutputFormat = goh264.EncoderOutputAnnexB
		cfg.DeblockMode = goh264.EncoderDeblockDisabled
		cfg.RTPMaxPayloadSize = 0
		cfg.GOPSize = 10000
		cfg.IDRInterval = 10000
		a := patternedI420EncoderFrame(48, 48)
		b := integerMotionI420EncoderFrame(a, 8, -8)
		encs := primedI420EncoderPool(t, cfg, a, 128)
		dst := make([]byte, 0, 65536)
		var call int
		allocs := testing.AllocsPerRun(100, func() {
			if call >= len(encs) {
				t.Fatalf("encoder pool exhausted after %d calls", call)
			}
			out, err := encs[call].EncodeInto(dst[:0], b)
			call++
			if err != nil {
				t.Fatalf("EncodeInto 8-pixel edge exact P16x16: %v", err)
			}
			if out.IDR || len(out.RTPPackets) != 0 || len(out.Data) == 0 ||
				len(out.NALUnits) != 1 || out.NALUnits[0].Type != 1 {
				t.Fatalf("8-pixel edge exact P16x16 output idr=%v rtp=%d data=%d nals=%+v",
					out.IDR, len(out.RTPPackets), len(out.Data), out.NALUnits)
			}
			if cap(out.Data) != cap(dst) {
				t.Fatalf("EncodeInto did not reuse caller output capacity: got cap %d want %d", cap(out.Data), cap(dst))
			}
		})
		t.Logf("annexb 8-pixel edge exact P16x16 EncodeInto allocations/run = %.0f", allocs)
		if allocs > 4 {
			t.Fatalf("annexb 8-pixel edge exact P16x16 EncodeInto allocations/run = %.0f, want <= 4", allocs)
		}
	})

	t.Run("annexb changed p-intrapcm", func(t *testing.T) {
		cfg := goh264.DefaultEncoderConfig(16, 16)
		cfg.OutputFormat = goh264.EncoderOutputAnnexB
		cfg.DeblockMode = goh264.EncoderDeblockDisabled
		cfg.RTPMaxPayloadSize = 0
		cfg.GOPSize = 10000
		cfg.IDRInterval = 10000
		enc, err := goh264.NewEncoder(cfg)
		if err != nil {
			t.Fatalf("NewEncoder: %v", err)
		}
		a := patternedI420EncoderFrame(16, 16)
		b := patternedI420EncoderFrame(16, 16)
		b.Y[0] ^= 0x7f
		if _, err := enc.EncodeInto(make([]byte, 0, 4096), a); err != nil {
			t.Fatalf("prime IDR: %v", err)
		}
		dst := make([]byte, 0, 4096)
		allocs := testing.AllocsPerRun(100, func() {
			out, err := enc.EncodeInto(dst[:0], b)
			if err != nil {
				t.Fatalf("EncodeInto changed P: %v", err)
			}
			if out.IDR || len(out.RTPPackets) != 0 || len(out.Data) == 0 {
				t.Fatalf("changed P output idr=%v rtp=%d data=%d", out.IDR, len(out.RTPPackets), len(out.Data))
			}
			out, err = enc.EncodeInto(dst[:0], a)
			if err != nil {
				t.Fatalf("EncodeInto changed P reset: %v", err)
			}
			if out.IDR || len(out.RTPPackets) != 0 || len(out.Data) == 0 {
				t.Fatalf("changed P reset output idr=%v rtp=%d data=%d", out.IDR, len(out.RTPPackets), len(out.Data))
			}
		})
		t.Logf("annexb changed P IntraPCM EncodeInto allocations/run = %.0f", allocs)
		if allocs > 12 {
			t.Fatalf("annexb changed P IntraPCM EncodeInto allocations/run = %.0f, want <= 12", allocs)
		}
	})

	t.Run("avc forced idr", func(t *testing.T) {
		cfg := goh264.DefaultEncoderConfig(16, 16)
		cfg.OutputFormat = goh264.EncoderOutputAVC
		cfg.DeblockMode = goh264.EncoderDeblockDisabled
		cfg.RTPMaxPayloadSize = 0
		cfg.GOPSize = 10000
		cfg.IDRInterval = 10000
		enc, err := goh264.NewEncoder(cfg)
		if err != nil {
			t.Fatalf("NewEncoder: %v", err)
		}
		frame := patternedI420EncoderFrame(16, 16)
		if _, err := enc.EncodeInto(make([]byte, 0, 4096), frame); err != nil {
			t.Fatalf("prime IDR: %v", err)
		}
		dst := make([]byte, 0, 4096)
		allocs := testing.AllocsPerRun(100, func() {
			enc.ForceIDR()
			out, err := enc.EncodeInto(dst[:0], frame)
			if err != nil {
				t.Fatalf("EncodeInto AVC forced IDR: %v", err)
			}
			if !out.IDR || len(out.RTPPackets) != 0 || len(out.Data) == 0 ||
				len(out.NALUnits) != 3 || out.NALUnits[0].Type != 7 || out.NALUnits[1].Type != 8 || out.NALUnits[2].Type != 5 {
				t.Fatalf("forced AVC IDR output idr=%v rtp=%d data=%d nals=%+v",
					out.IDR, len(out.RTPPackets), len(out.Data), out.NALUnits)
			}
			if cap(out.Data) != cap(dst) {
				t.Fatalf("EncodeInto did not reuse caller output capacity: got cap %d want %d", cap(out.Data), cap(dst))
			}
		})
		t.Logf("avc forced IDR EncodeInto allocations/run = %.0f", allocs)
		if allocs > 8 {
			t.Fatalf("avc forced IDR EncodeInto allocations/run = %.0f, want <= 8", allocs)
		}
	})

	t.Run("avc steady p-skip", func(t *testing.T) {
		cfg := goh264.DefaultEncoderConfig(16, 16)
		cfg.OutputFormat = goh264.EncoderOutputAVC
		cfg.DeblockMode = goh264.EncoderDeblockDisabled
		cfg.RTPMaxPayloadSize = 0
		cfg.GOPSize = 10000
		cfg.IDRInterval = 10000
		enc, err := goh264.NewEncoder(cfg)
		if err != nil {
			t.Fatalf("NewEncoder: %v", err)
		}
		frame := patternedI420EncoderFrame(16, 16)
		if _, err := enc.EncodeInto(make([]byte, 0, 4096), frame); err != nil {
			t.Fatalf("prime IDR: %v", err)
		}
		dst := make([]byte, 0, 4096)
		allocs := testing.AllocsPerRun(100, func() {
			out, err := enc.EncodeInto(dst[:0], frame)
			if err != nil {
				t.Fatalf("EncodeInto AVC P-skip: %v", err)
			}
			if out.IDR || len(out.RTPPackets) != 0 || len(out.Data) == 0 ||
				len(out.NALUnits) != 1 || out.NALUnits[0].Type != 1 {
				t.Fatalf("steady AVC P-skip output idr=%v rtp=%d data=%d nals=%+v",
					out.IDR, len(out.RTPPackets), len(out.Data), out.NALUnits)
			}
			if cap(out.Data) != cap(dst) {
				t.Fatalf("EncodeInto did not reuse caller output capacity: got cap %d want %d", cap(out.Data), cap(dst))
			}
		})
		t.Logf("avc steady p-skip EncodeInto allocations/run = %.0f", allocs)
		if allocs > 4 {
			t.Fatalf("avc steady P-skip EncodeInto allocations/run = %.0f, want <= 4", allocs)
		}
	})

	t.Run("avc exact p16x16", func(t *testing.T) {
		cfg := goh264.DefaultEncoderConfig(16, 16)
		cfg.OutputFormat = goh264.EncoderOutputAVC
		cfg.DeblockMode = goh264.EncoderDeblockDisabled
		cfg.RTPMaxPayloadSize = 0
		cfg.GOPSize = 10000
		cfg.IDRInterval = 10000
		a := patternedI420EncoderFrame(16, 16)
		b := integerMotionI420EncoderFrame(a, 2, 0)
		encs := primedI420EncoderPool(t, cfg, a, 128)
		dst := make([]byte, 0, 4096)
		var call int
		allocs := testing.AllocsPerRun(100, func() {
			if call >= len(encs) {
				t.Fatalf("encoder pool exhausted after %d calls", call)
			}
			out, err := encs[call].EncodeInto(dst[:0], b)
			call++
			if err != nil {
				t.Fatalf("EncodeInto AVC exact P16x16: %v", err)
			}
			if out.IDR || len(out.RTPPackets) != 0 || len(out.Data) == 0 ||
				len(out.NALUnits) != 1 || out.NALUnits[0].Type != 1 {
				t.Fatalf("exact AVC P16x16 output idr=%v rtp=%d data=%d nals=%+v",
					out.IDR, len(out.RTPPackets), len(out.Data), out.NALUnits)
			}
			if cap(out.Data) != cap(dst) {
				t.Fatalf("EncodeInto did not reuse caller output capacity: got cap %d want %d", cap(out.Data), cap(dst))
			}
		})
		t.Logf("avc exact P16x16 EncodeInto allocations/run = %.0f", allocs)
		if allocs > 4 {
			t.Fatalf("avc exact P16x16 EncodeInto allocations/run = %.0f, want <= 4", allocs)
		}
	})

	t.Run("avc odd exact p16x16 constant chroma", func(t *testing.T) {
		cfg := goh264.DefaultEncoderConfig(16, 16)
		cfg.OutputFormat = goh264.EncoderOutputAVC
		cfg.DeblockMode = goh264.EncoderDeblockDisabled
		cfg.RTPMaxPayloadSize = 0
		cfg.GOPSize = 10000
		cfg.IDRInterval = 10000
		a := patternedI420EncoderFrame(16, 16)
		setConstantI420Chroma(&a, 128, 64)
		b := integerMotionI420EncoderFrame(a, 1, 0)
		encs := primedI420EncoderPool(t, cfg, a, 128)
		dst := make([]byte, 0, 4096)
		var call int
		allocs := testing.AllocsPerRun(100, func() {
			if call >= len(encs) {
				t.Fatalf("encoder pool exhausted after %d calls", call)
			}
			out, err := encs[call].EncodeInto(dst[:0], b)
			call++
			if err != nil {
				t.Fatalf("EncodeInto AVC odd exact P16x16: %v", err)
			}
			if out.IDR || len(out.RTPPackets) != 0 || len(out.Data) == 0 ||
				len(out.NALUnits) != 1 || out.NALUnits[0].Type != 1 {
				t.Fatalf("odd exact AVC P16x16 output idr=%v rtp=%d data=%d nals=%+v",
					out.IDR, len(out.RTPPackets), len(out.Data), out.NALUnits)
			}
			if cap(out.Data) != cap(dst) {
				t.Fatalf("EncodeInto did not reuse caller output capacity: got cap %d want %d", cap(out.Data), cap(dst))
			}
		})
		t.Logf("avc odd exact P16x16 constant-chroma EncodeInto allocations/run = %.0f", allocs)
		if allocs > 4 {
			t.Fatalf("avc odd exact P16x16 constant-chroma EncodeInto allocations/run = %.0f, want <= 4", allocs)
		}
	})

	t.Run("avc odd patterned-chroma fallback", func(t *testing.T) {
		cfg := goh264.DefaultEncoderConfig(16, 16)
		cfg.OutputFormat = goh264.EncoderOutputAVC
		assertEncoderEncodeIntoOddPatternedChromaFallbackAllocationCanary(t, cfg, "avc", 0, 6)
	})

	t.Run("avc exact p16x16 edge search", func(t *testing.T) {
		cfg := goh264.DefaultEncoderConfig(48, 48)
		cfg.OutputFormat = goh264.EncoderOutputAVC
		cfg.DeblockMode = goh264.EncoderDeblockDisabled
		cfg.RTPMaxPayloadSize = 0
		cfg.GOPSize = 10000
		cfg.IDRInterval = 10000
		a := patternedI420EncoderFrame(48, 48)
		b := integerMotionI420EncoderFrame(a, 8, -8)
		encs := primedI420EncoderPool(t, cfg, a, 128)
		dst := make([]byte, 0, 65536)
		var call int
		allocs := testing.AllocsPerRun(100, func() {
			if call >= len(encs) {
				t.Fatalf("encoder pool exhausted after %d calls", call)
			}
			out, err := encs[call].EncodeInto(dst[:0], b)
			call++
			if err != nil {
				t.Fatalf("EncodeInto AVC 8-pixel edge exact P16x16: %v", err)
			}
			if out.IDR || len(out.RTPPackets) != 0 || len(out.Data) == 0 ||
				len(out.NALUnits) != 1 || out.NALUnits[0].Type != 1 {
				t.Fatalf("8-pixel edge AVC exact P16x16 output idr=%v rtp=%d data=%d nals=%+v",
					out.IDR, len(out.RTPPackets), len(out.Data), out.NALUnits)
			}
			if cap(out.Data) != cap(dst) {
				t.Fatalf("EncodeInto did not reuse caller output capacity: got cap %d want %d", cap(out.Data), cap(dst))
			}
		})
		t.Logf("avc 8-pixel edge exact P16x16 EncodeInto allocations/run = %.0f", allocs)
		if allocs > 4 {
			t.Fatalf("avc 8-pixel edge exact P16x16 EncodeInto allocations/run = %.0f, want <= 4", allocs)
		}
	})

	t.Run("avc per-macroblock exact p16x16", func(t *testing.T) {
		cfg := goh264.DefaultEncoderConfig(32, 32)
		cfg.OutputFormat = goh264.EncoderOutputAVC
		assertEncoderEncodeIntoPerMacroblockExactP16x16AllocationCanary(t, cfg, "avc", 0, 5)
	})

	t.Run("avc changed p-intrapcm", func(t *testing.T) {
		cfg := goh264.DefaultEncoderConfig(16, 16)
		cfg.OutputFormat = goh264.EncoderOutputAVC
		cfg.DeblockMode = goh264.EncoderDeblockDisabled
		cfg.RTPMaxPayloadSize = 0
		cfg.GOPSize = 10000
		cfg.IDRInterval = 10000
		enc, err := goh264.NewEncoder(cfg)
		if err != nil {
			t.Fatalf("NewEncoder: %v", err)
		}
		a := patternedI420EncoderFrame(16, 16)
		b := patternedI420EncoderFrame(16, 16)
		b.Y[0] ^= 0x7f
		if _, err := enc.EncodeInto(make([]byte, 0, 4096), a); err != nil {
			t.Fatalf("prime IDR: %v", err)
		}
		dst := make([]byte, 0, 4096)
		allocs := testing.AllocsPerRun(100, func() {
			out, err := enc.EncodeInto(dst[:0], b)
			if err != nil {
				t.Fatalf("EncodeInto AVC changed P: %v", err)
			}
			if out.IDR || len(out.RTPPackets) != 0 || len(out.Data) == 0 ||
				len(out.NALUnits) != 2 || out.NALUnits[0].Type != 6 || out.NALUnits[1].Type != 1 {
				t.Fatalf("changed AVC P output idr=%v rtp=%d data=%d nals=%+v",
					out.IDR, len(out.RTPPackets), len(out.Data), out.NALUnits)
			}
			out, err = enc.EncodeInto(dst[:0], a)
			if err != nil {
				t.Fatalf("EncodeInto AVC changed P reset: %v", err)
			}
			if out.IDR || len(out.RTPPackets) != 0 || len(out.Data) == 0 ||
				len(out.NALUnits) != 2 || out.NALUnits[0].Type != 6 || out.NALUnits[1].Type != 1 {
				t.Fatalf("changed AVC P reset output idr=%v rtp=%d data=%d nals=%+v",
					out.IDR, len(out.RTPPackets), len(out.Data), out.NALUnits)
			}
		})
		t.Logf("avc changed P IntraPCM EncodeInto allocations/run = %.0f", allocs)
		if allocs > 12 {
			t.Fatalf("avc changed P IntraPCM EncodeInto allocations/run = %.0f, want <= 12", allocs)
		}
	})

	t.Run("rtp forced idr", func(t *testing.T) {
		cfg := goh264.DefaultEncoderConfig(16, 16)
		cfg.DeblockMode = goh264.EncoderDeblockDisabled
		cfg.RTPMaxPayloadSize = 32
		cfg.GOPSize = 10000
		cfg.IDRInterval = 10000
		enc, err := goh264.NewEncoder(cfg)
		if err != nil {
			t.Fatalf("NewEncoder: %v", err)
		}
		frame := patternedI420EncoderFrame(16, 16)
		if _, err := enc.EncodeInto(make([]byte, 0, 4096), frame); err != nil {
			t.Fatalf("prime IDR: %v", err)
		}
		dst := make([]byte, 0, 4096)
		allocs := testing.AllocsPerRun(100, func() {
			enc.ForceIDR()
			out, err := enc.EncodeInto(dst[:0], frame)
			if err != nil {
				t.Fatalf("EncodeInto RTP forced IDR: %v", err)
			}
			if !out.IDR || len(out.RTPPackets) == 0 || len(out.Data) == 0 {
				t.Fatalf("forced RTP IDR output idr=%v rtp=%d data=%d", out.IDR, len(out.RTPPackets), len(out.Data))
			}
			if cap(out.Data) != cap(dst) {
				t.Fatalf("EncodeInto did not reuse caller output capacity: got cap %d want %d", cap(out.Data), cap(dst))
			}
		})
		t.Logf("rtp forced IDR EncodeInto allocations/run = %.0f", allocs)
		if allocs > 10 {
			t.Fatalf("rtp forced IDR EncodeInto allocations/run = %.0f, want <= 10", allocs)
		}
	})

	t.Run("rtp stapa forced idr", func(t *testing.T) {
		cfg := goh264.DefaultEncoderConfig(16, 16)
		cfg.DeblockMode = goh264.EncoderDeblockDisabled
		cfg.STAPA = true
		cfg.RTPMaxPayloadSize = 128
		cfg.GOPSize = 10000
		cfg.IDRInterval = 10000
		enc, err := goh264.NewEncoder(cfg)
		if err != nil {
			t.Fatalf("NewEncoder: %v", err)
		}
		frame := patternedI420EncoderFrame(16, 16)
		if _, err := enc.EncodeInto(make([]byte, 0, 4096), frame); err != nil {
			t.Fatalf("prime IDR: %v", err)
		}
		dst := make([]byte, 0, 4096)
		allocs := testing.AllocsPerRun(100, func() {
			enc.ForceIDR()
			out, err := enc.EncodeInto(dst[:0], frame)
			if err != nil {
				t.Fatalf("EncodeInto RTP STAP-A forced IDR: %v", err)
			}
			if !out.IDR || len(out.RTPPackets) < 2 || len(out.Data) == 0 ||
				len(out.NALUnits) != 3 || out.NALUnits[0].Type != 7 || out.NALUnits[1].Type != 8 || out.NALUnits[2].Type != 5 {
				t.Fatalf("forced RTP STAP-A IDR output idr=%v rtp=%d data=%d nals=%+v",
					out.IDR, len(out.RTPPackets), len(out.Data), out.NALUnits)
			}
			if len(out.RTPPackets[0].Payload) == 0 || out.RTPPackets[0].Payload[0]&0x1f != 24 {
				t.Fatalf("forced RTP STAP-A IDR first payload = %x, want STAP-A", out.RTPPackets[0].Payload)
			}
			if cap(out.Data) != cap(dst) {
				t.Fatalf("EncodeInto did not reuse caller output capacity: got cap %d want %d", cap(out.Data), cap(dst))
			}
		})
		t.Logf("rtp STAP-A forced IDR EncodeInto allocations/run = %.0f", allocs)
		if allocs > 10 {
			t.Fatalf("rtp STAP-A forced IDR EncodeInto allocations/run = %.0f, want <= 10", allocs)
		}
	})

	t.Run("rtp mode0 forced idr", func(t *testing.T) {
		cfg := goh264.DefaultEncoderConfig(16, 16)
		cfg.DeblockMode = goh264.EncoderDeblockDisabled
		cfg.RTPPacketizationMode = goh264.EncoderRTPPacketizationSingleNAL
		cfg.RTPMaxPayloadSize = 1200
		cfg.GOPSize = 10000
		cfg.IDRInterval = 10000
		enc, err := goh264.NewEncoder(cfg)
		if err != nil {
			t.Fatalf("NewEncoder: %v", err)
		}
		frame := patternedI420EncoderFrame(16, 16)
		if _, err := enc.EncodeInto(make([]byte, 0, 4096), frame); err != nil {
			t.Fatalf("prime IDR: %v", err)
		}
		dst := make([]byte, 0, 4096)
		allocs := testing.AllocsPerRun(100, func() {
			enc.ForceIDR()
			out, err := enc.EncodeInto(dst[:0], frame)
			if err != nil {
				t.Fatalf("EncodeInto RTP mode0 forced IDR: %v", err)
			}
			if !out.IDR || len(out.RTPPackets) != 3 || len(out.Data) == 0 {
				t.Fatalf("forced RTP mode0 IDR output idr=%v rtp=%d data=%d", out.IDR, len(out.RTPPackets), len(out.Data))
			}
			assertEncoderNALTypes(t, out.NALUnits, []uint8{7, 8, 5})
			if cap(out.Data) != cap(dst) {
				t.Fatalf("EncodeInto did not reuse caller output capacity: got cap %d want %d", cap(out.Data), cap(dst))
			}
		})
		t.Logf("rtp mode0 forced IDR EncodeInto allocations/run = %.0f", allocs)
		if allocs > 10 {
			t.Fatalf("rtp mode0 forced IDR EncodeInto allocations/run = %.0f, want <= 10", allocs)
		}
	})

	t.Run("rtp exact p16x16", func(t *testing.T) {
		cfg := goh264.DefaultEncoderConfig(16, 16)
		cfg.DeblockMode = goh264.EncoderDeblockDisabled
		cfg.GOPSize = 10000
		cfg.IDRInterval = 10000
		a := patternedI420EncoderFrame(16, 16)
		b := integerMotionI420EncoderFrame(a, 2, 0)
		encs := primedI420EncoderPool(t, cfg, a, 128)
		dst := make([]byte, 0, 4096)
		var call int
		allocs := testing.AllocsPerRun(100, func() {
			if call >= len(encs) {
				t.Fatalf("encoder pool exhausted after %d calls", call)
			}
			out, err := encs[call].EncodeInto(dst[:0], b)
			call++
			if err != nil {
				t.Fatalf("EncodeInto RTP exact P16x16: %v", err)
			}
			if out.IDR || len(out.RTPPackets) == 0 || len(out.Data) == 0 ||
				len(out.NALUnits) != 1 || out.NALUnits[0].Type != 1 {
				t.Fatalf("exact RTP P16x16 output idr=%v rtp=%d data=%d nals=%+v",
					out.IDR, len(out.RTPPackets), len(out.Data), out.NALUnits)
			}
			if cap(out.Data) != cap(dst) {
				t.Fatalf("EncodeInto did not reuse caller output capacity: got cap %d want %d", cap(out.Data), cap(dst))
			}
		})
		t.Logf("rtp exact P16x16 EncodeInto allocations/run = %.0f", allocs)
		if allocs > 5 {
			t.Fatalf("rtp exact P16x16 EncodeInto allocations/run = %.0f, want <= 5", allocs)
		}
	})

	t.Run("rtp odd exact p16x16 constant chroma", func(t *testing.T) {
		cfg := goh264.DefaultEncoderConfig(16, 16)
		cfg.DeblockMode = goh264.EncoderDeblockDisabled
		cfg.GOPSize = 10000
		cfg.IDRInterval = 10000
		a := patternedI420EncoderFrame(16, 16)
		setConstantI420Chroma(&a, 128, 64)
		b := integerMotionI420EncoderFrame(a, 1, 0)
		encs := primedI420EncoderPool(t, cfg, a, 128)
		dst := make([]byte, 0, 4096)
		var call int
		allocs := testing.AllocsPerRun(100, func() {
			if call >= len(encs) {
				t.Fatalf("encoder pool exhausted after %d calls", call)
			}
			out, err := encs[call].EncodeInto(dst[:0], b)
			call++
			if err != nil {
				t.Fatalf("EncodeInto RTP odd exact P16x16: %v", err)
			}
			if out.IDR || len(out.RTPPackets) == 0 || len(out.Data) == 0 ||
				len(out.NALUnits) != 1 || out.NALUnits[0].Type != 1 {
				t.Fatalf("odd exact RTP P16x16 output idr=%v rtp=%d data=%d nals=%+v",
					out.IDR, len(out.RTPPackets), len(out.Data), out.NALUnits)
			}
			if cap(out.Data) != cap(dst) {
				t.Fatalf("EncodeInto did not reuse caller output capacity: got cap %d want %d", cap(out.Data), cap(dst))
			}
		})
		t.Logf("rtp odd exact P16x16 constant-chroma EncodeInto allocations/run = %.0f", allocs)
		if allocs > 5 {
			t.Fatalf("rtp odd exact P16x16 constant-chroma EncodeInto allocations/run = %.0f, want <= 5", allocs)
		}
	})

	t.Run("rtp odd patterned-chroma fallback", func(t *testing.T) {
		cfg := goh264.DefaultEncoderConfig(16, 16)
		assertEncoderEncodeIntoOddPatternedChromaFallbackAllocationCanary(t, cfg, "rtp", 2, 8)
	})

	t.Run("rtp exact p16x16 edge search", func(t *testing.T) {
		cfg := goh264.DefaultEncoderConfig(48, 48)
		cfg.DeblockMode = goh264.EncoderDeblockDisabled
		cfg.GOPSize = 10000
		cfg.IDRInterval = 10000
		a := patternedI420EncoderFrame(48, 48)
		b := integerMotionI420EncoderFrame(a, 8, -8)
		encs := primedI420EncoderPool(t, cfg, a, 128)
		dst := make([]byte, 0, 65536)
		var call int
		allocs := testing.AllocsPerRun(100, func() {
			if call >= len(encs) {
				t.Fatalf("encoder pool exhausted after %d calls", call)
			}
			out, err := encs[call].EncodeInto(dst[:0], b)
			call++
			if err != nil {
				t.Fatalf("EncodeInto RTP 8-pixel edge exact P16x16: %v", err)
			}
			if out.IDR || len(out.RTPPackets) == 0 || len(out.Data) == 0 ||
				len(out.NALUnits) != 1 || out.NALUnits[0].Type != 1 {
				t.Fatalf("8-pixel edge RTP exact P16x16 output idr=%v rtp=%d data=%d nals=%+v",
					out.IDR, len(out.RTPPackets), len(out.Data), out.NALUnits)
			}
			if cap(out.Data) != cap(dst) {
				t.Fatalf("EncodeInto did not reuse caller output capacity: got cap %d want %d", cap(out.Data), cap(dst))
			}
		})
		t.Logf("rtp 8-pixel edge exact P16x16 EncodeInto allocations/run = %.0f", allocs)
		if allocs > 5 {
			t.Fatalf("rtp 8-pixel edge exact P16x16 EncodeInto allocations/run = %.0f, want <= 5", allocs)
		}
	})

	t.Run("rtp per-macroblock exact p16x16", func(t *testing.T) {
		cfg := goh264.DefaultEncoderConfig(32, 32)
		assertEncoderEncodeIntoPerMacroblockExactP16x16AllocationCanary(t, cfg, "rtp", 2, 7)
	})

	t.Run("rtp steady p-skip", func(t *testing.T) {
		cfg := goh264.DefaultEncoderConfig(16, 16)
		cfg.DeblockMode = goh264.EncoderDeblockDisabled
		cfg.GOPSize = 10000
		cfg.IDRInterval = 10000
		enc, err := goh264.NewEncoder(cfg)
		if err != nil {
			t.Fatalf("NewEncoder: %v", err)
		}
		frame := patternedI420EncoderFrame(16, 16)
		if _, err := enc.EncodeInto(make([]byte, 0, 4096), frame); err != nil {
			t.Fatalf("prime IDR: %v", err)
		}
		dst := make([]byte, 0, 4096)
		allocs := testing.AllocsPerRun(100, func() {
			out, err := enc.EncodeInto(dst[:0], frame)
			if err != nil {
				t.Fatalf("EncodeInto RTP P-skip: %v", err)
			}
			if out.IDR || len(out.RTPPackets) == 0 || len(out.Data) == 0 {
				t.Fatalf("steady RTP P-skip output idr=%v rtp=%d data=%d", out.IDR, len(out.RTPPackets), len(out.Data))
			}
			if cap(out.Data) != cap(dst) {
				t.Fatalf("EncodeInto did not reuse caller output capacity: got cap %d want %d", cap(out.Data), cap(dst))
			}
		})
		t.Logf("rtp steady p-skip EncodeInto allocations/run = %.0f", allocs)
		if allocs > 5 {
			t.Fatalf("rtp steady P-skip EncodeInto allocations/run = %.0f, want <= 5", allocs)
		}
	})

	t.Run("rtp changed p-intrapcm", func(t *testing.T) {
		cfg := goh264.DefaultEncoderConfig(16, 16)
		cfg.DeblockMode = goh264.EncoderDeblockDisabled
		cfg.GOPSize = 10000
		cfg.IDRInterval = 10000
		enc, err := goh264.NewEncoder(cfg)
		if err != nil {
			t.Fatalf("NewEncoder: %v", err)
		}
		a := patternedI420EncoderFrame(16, 16)
		b := patternedI420EncoderFrame(16, 16)
		b.Y[0] ^= 0x7f
		if _, err := enc.EncodeInto(make([]byte, 0, 4096), a); err != nil {
			t.Fatalf("prime IDR: %v", err)
		}
		dst := make([]byte, 0, 4096)
		allocs := testing.AllocsPerRun(100, func() {
			out, err := enc.EncodeInto(dst[:0], b)
			if err != nil {
				t.Fatalf("EncodeInto RTP changed P: %v", err)
			}
			if out.IDR || len(out.RTPPackets) != 2 || len(out.Data) == 0 {
				t.Fatalf("changed RTP P output idr=%v rtp=%d data=%d", out.IDR, len(out.RTPPackets), len(out.Data))
			}
			assertEncoderNALTypes(t, out.NALUnits, []uint8{6, 1})
			out, err = enc.EncodeInto(dst[:0], a)
			if err != nil {
				t.Fatalf("EncodeInto RTP changed P reset: %v", err)
			}
			if out.IDR || len(out.RTPPackets) != 2 || len(out.Data) == 0 {
				t.Fatalf("changed RTP P reset output idr=%v rtp=%d data=%d", out.IDR, len(out.RTPPackets), len(out.Data))
			}
			assertEncoderNALTypes(t, out.NALUnits, []uint8{6, 1})
			if cap(out.Data) != cap(dst) {
				t.Fatalf("EncodeInto did not reuse caller output capacity: got cap %d want %d", cap(out.Data), cap(dst))
			}
		})
		t.Logf("rtp changed P IntraPCM EncodeInto allocations/run = %.0f", allocs)
		if allocs > 16 {
			t.Fatalf("rtp changed P IntraPCM EncodeInto allocations/run = %.0f, want <= 16", allocs)
		}
	})

	t.Run("rtp mode0 steady p-skip", func(t *testing.T) {
		cfg := goh264.DefaultEncoderConfig(16, 16)
		cfg.DeblockMode = goh264.EncoderDeblockDisabled
		cfg.RTPPacketizationMode = goh264.EncoderRTPPacketizationSingleNAL
		cfg.RTPMaxPayloadSize = 1200
		cfg.GOPSize = 10000
		cfg.IDRInterval = 10000
		enc, err := goh264.NewEncoder(cfg)
		if err != nil {
			t.Fatalf("NewEncoder: %v", err)
		}
		frame := patternedI420EncoderFrame(16, 16)
		if _, err := enc.EncodeInto(make([]byte, 0, 4096), frame); err != nil {
			t.Fatalf("prime IDR: %v", err)
		}
		dst := make([]byte, 0, 4096)
		allocs := testing.AllocsPerRun(100, func() {
			out, err := enc.EncodeInto(dst[:0], frame)
			if err != nil {
				t.Fatalf("EncodeInto RTP mode0 P-skip: %v", err)
			}
			if out.IDR || len(out.RTPPackets) != 1 || len(out.Data) == 0 ||
				len(out.NALUnits) != 1 || out.NALUnits[0].Type != 1 {
				t.Fatalf("steady RTP mode0 P-skip output idr=%v rtp=%d data=%d nals=%+v",
					out.IDR, len(out.RTPPackets), len(out.Data), out.NALUnits)
			}
			if cap(out.Data) != cap(dst) {
				t.Fatalf("EncodeInto did not reuse caller output capacity: got cap %d want %d", cap(out.Data), cap(dst))
			}
		})
		t.Logf("rtp mode0 steady p-skip EncodeInto allocations/run = %.0f", allocs)
		if allocs > 5 {
			t.Fatalf("rtp mode0 steady P-skip EncodeInto allocations/run = %.0f, want <= 5", allocs)
		}
	})

	t.Run("rtp mode0 exact p16x16", func(t *testing.T) {
		cfg := goh264.DefaultEncoderConfig(16, 16)
		cfg.DeblockMode = goh264.EncoderDeblockDisabled
		cfg.RTPPacketizationMode = goh264.EncoderRTPPacketizationSingleNAL
		cfg.RTPMaxPayloadSize = 1200
		cfg.GOPSize = 10000
		cfg.IDRInterval = 10000
		a := patternedI420EncoderFrame(16, 16)
		b := integerMotionI420EncoderFrame(a, 2, 0)
		encs := primedI420EncoderPool(t, cfg, a, 128)
		dst := make([]byte, 0, 4096)
		var call int
		allocs := testing.AllocsPerRun(100, func() {
			if call >= len(encs) {
				t.Fatalf("encoder pool exhausted after %d calls", call)
			}
			out, err := encs[call].EncodeInto(dst[:0], b)
			call++
			if err != nil {
				t.Fatalf("EncodeInto RTP mode0 exact P16x16: %v", err)
			}
			if out.IDR || len(out.RTPPackets) != 1 || len(out.Data) == 0 ||
				len(out.NALUnits) != 1 || out.NALUnits[0].Type != 1 {
				t.Fatalf("exact RTP mode0 P16x16 output idr=%v rtp=%d data=%d nals=%+v",
					out.IDR, len(out.RTPPackets), len(out.Data), out.NALUnits)
			}
			if cap(out.Data) != cap(dst) {
				t.Fatalf("EncodeInto did not reuse caller output capacity: got cap %d want %d", cap(out.Data), cap(dst))
			}
		})
		t.Logf("rtp mode0 exact P16x16 EncodeInto allocations/run = %.0f", allocs)
		if allocs > 5 {
			t.Fatalf("rtp mode0 exact P16x16 EncodeInto allocations/run = %.0f, want <= 5", allocs)
		}
	})

	t.Run("rtp mode0 odd exact p16x16 constant chroma", func(t *testing.T) {
		cfg := goh264.DefaultEncoderConfig(16, 16)
		cfg.DeblockMode = goh264.EncoderDeblockDisabled
		cfg.RTPPacketizationMode = goh264.EncoderRTPPacketizationSingleNAL
		cfg.RTPMaxPayloadSize = 1200
		cfg.GOPSize = 10000
		cfg.IDRInterval = 10000
		a := patternedI420EncoderFrame(16, 16)
		setConstantI420Chroma(&a, 128, 64)
		b := integerMotionI420EncoderFrame(a, 1, 0)
		encs := primedI420EncoderPool(t, cfg, a, 128)
		dst := make([]byte, 0, 4096)
		var call int
		allocs := testing.AllocsPerRun(100, func() {
			if call >= len(encs) {
				t.Fatalf("encoder pool exhausted after %d calls", call)
			}
			out, err := encs[call].EncodeInto(dst[:0], b)
			call++
			if err != nil {
				t.Fatalf("EncodeInto RTP mode0 odd exact P16x16: %v", err)
			}
			if out.IDR || len(out.RTPPackets) != 1 || len(out.Data) == 0 ||
				len(out.NALUnits) != 1 || out.NALUnits[0].Type != 1 {
				t.Fatalf("odd exact RTP mode0 P16x16 output idr=%v rtp=%d data=%d nals=%+v",
					out.IDR, len(out.RTPPackets), len(out.Data), out.NALUnits)
			}
			if cap(out.Data) != cap(dst) {
				t.Fatalf("EncodeInto did not reuse caller output capacity: got cap %d want %d", cap(out.Data), cap(dst))
			}
		})
		t.Logf("rtp mode0 odd exact P16x16 constant-chroma EncodeInto allocations/run = %.0f", allocs)
		if allocs > 5 {
			t.Fatalf("rtp mode0 odd exact P16x16 constant-chroma EncodeInto allocations/run = %.0f, want <= 5", allocs)
		}
	})

	t.Run("rtp mode0 odd patterned-chroma fallback", func(t *testing.T) {
		cfg := goh264.DefaultEncoderConfig(16, 16)
		cfg.RTPPacketizationMode = goh264.EncoderRTPPacketizationSingleNAL
		cfg.RTPMaxPayloadSize = 1200
		assertEncoderEncodeIntoOddPatternedChromaFallbackAllocationCanary(t, cfg, "rtp mode0", 2, 8)
	})

	t.Run("rtp mode0 exact p16x16 edge search", func(t *testing.T) {
		cfg := goh264.DefaultEncoderConfig(32, 16)
		cfg.DeblockMode = goh264.EncoderDeblockDisabled
		cfg.RTPPacketizationMode = goh264.EncoderRTPPacketizationSingleNAL
		cfg.RTPMaxPayloadSize = 1200
		cfg.GOPSize = 10000
		cfg.IDRInterval = 10000
		a := patternedI420EncoderFrame(32, 16)
		b := integerMotionI420EncoderFrame(a, 8, -8)
		encs := primedI420EncoderPool(t, cfg, a, 128)
		dst := make([]byte, 0, 65536)
		var call int
		allocs := testing.AllocsPerRun(100, func() {
			if call >= len(encs) {
				t.Fatalf("encoder pool exhausted after %d calls", call)
			}
			out, err := encs[call].EncodeInto(dst[:0], b)
			call++
			if err != nil {
				t.Fatalf("EncodeInto RTP mode0 8-pixel edge exact P16x16: %v", err)
			}
			if out.IDR || len(out.RTPPackets) != 1 || len(out.Data) == 0 ||
				len(out.NALUnits) != 1 || out.NALUnits[0].Type != 1 {
				t.Fatalf("8-pixel edge RTP mode0 exact P16x16 output idr=%v rtp=%d data=%d nals=%+v",
					out.IDR, len(out.RTPPackets), len(out.Data), out.NALUnits)
			}
			if cap(out.Data) != cap(dst) {
				t.Fatalf("EncodeInto did not reuse caller output capacity: got cap %d want %d", cap(out.Data), cap(dst))
			}
		})
		t.Logf("rtp mode0 8-pixel edge exact P16x16 EncodeInto allocations/run = %.0f", allocs)
		if allocs > 5 {
			t.Fatalf("rtp mode0 8-pixel edge exact P16x16 EncodeInto allocations/run = %.0f, want <= 5", allocs)
		}
	})

	t.Run("rtp mode0 per-macroblock exact p16x16", func(t *testing.T) {
		cfg := goh264.DefaultEncoderConfig(32, 32)
		cfg.RTPPacketizationMode = goh264.EncoderRTPPacketizationSingleNAL
		cfg.RTPMaxPayloadSize = 1200
		assertEncoderEncodeIntoPerMacroblockExactP16x16AllocationCanary(t, cfg, "rtp mode0", 2, 7)
	})

	t.Run("rtp mode0 changed p-intrapcm", func(t *testing.T) {
		cfg := goh264.DefaultEncoderConfig(16, 16)
		cfg.DeblockMode = goh264.EncoderDeblockDisabled
		cfg.RTPPacketizationMode = goh264.EncoderRTPPacketizationSingleNAL
		cfg.RTPMaxPayloadSize = 1200
		cfg.GOPSize = 10000
		cfg.IDRInterval = 10000
		enc, err := goh264.NewEncoder(cfg)
		if err != nil {
			t.Fatalf("NewEncoder: %v", err)
		}
		a := patternedI420EncoderFrame(16, 16)
		b := patternedI420EncoderFrame(16, 16)
		b.Y[0] ^= 0x7f
		if _, err := enc.EncodeInto(make([]byte, 0, 4096), a); err != nil {
			t.Fatalf("prime IDR: %v", err)
		}
		dst := make([]byte, 0, 4096)
		allocs := testing.AllocsPerRun(100, func() {
			out, err := enc.EncodeInto(dst[:0], b)
			if err != nil {
				t.Fatalf("EncodeInto RTP mode0 changed P: %v", err)
			}
			if out.IDR || len(out.RTPPackets) != 2 || len(out.Data) == 0 {
				t.Fatalf("changed RTP mode0 P output idr=%v rtp=%d data=%d", out.IDR, len(out.RTPPackets), len(out.Data))
			}
			assertEncoderNALTypes(t, out.NALUnits, []uint8{6, 1})
			out, err = enc.EncodeInto(dst[:0], a)
			if err != nil {
				t.Fatalf("EncodeInto RTP mode0 changed P reset: %v", err)
			}
			if out.IDR || len(out.RTPPackets) != 2 || len(out.Data) == 0 {
				t.Fatalf("changed RTP mode0 P reset output idr=%v rtp=%d data=%d", out.IDR, len(out.RTPPackets), len(out.Data))
			}
			assertEncoderNALTypes(t, out.NALUnits, []uint8{6, 1})
			if cap(out.Data) != cap(dst) {
				t.Fatalf("EncodeInto did not reuse caller output capacity: got cap %d want %d", cap(out.Data), cap(dst))
			}
		})
		t.Logf("rtp mode0 changed P IntraPCM EncodeInto allocations/run = %.0f", allocs)
		if allocs > 16 {
			t.Fatalf("rtp mode0 changed P IntraPCM EncodeInto allocations/run = %.0f, want <= 16", allocs)
		}
	})

	for _, tt := range []struct {
		name         string
		outputFormat goh264.EncoderOutputFormat
		budget       string
		maxAllocs    float64
	}{
		{name: "annexb max-frame-size drop", outputFormat: goh264.EncoderOutputAnnexB, budget: "frame", maxAllocs: 7},
		{name: "annexb slice-max-bytes drop", outputFormat: goh264.EncoderOutputAnnexB, budget: "slice", maxAllocs: 7},
		{name: "avc max-frame-size drop", outputFormat: goh264.EncoderOutputAVC, budget: "frame", maxAllocs: 7},
		{name: "avc slice-max-bytes drop", outputFormat: goh264.EncoderOutputAVC, budget: "slice", maxAllocs: 7},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cfg := goh264.DefaultEncoderConfig(16, 16)
			cfg.OutputFormat = tt.outputFormat
			cfg.DeblockMode = goh264.EncoderDeblockDisabled
			cfg.RTPMaxPayloadSize = 0
			cfg.MaxFrameSize = 4096
			cfg.SliceMaxBytes = 4096
			cfg.GOPSize = 10000
			cfg.IDRInterval = 10000
			enc, err := goh264.NewEncoder(cfg)
			if err != nil {
				t.Fatalf("NewEncoder: %v", err)
			}
			a := patternedI420EncoderFrame(16, 16)
			a.PTS = 0
			first, err := enc.EncodeInto(make([]byte, 0, 4096), a)
			if err != nil {
				t.Fatalf("prime IDR: %v", err)
			}
			if !first.IDR || len(first.Data) == 0 || len(first.RTPPackets) != 0 {
				t.Fatalf("prime output idr=%v data=%d rtp=%d, want non-RTP IDR",
					first.IDR, len(first.Data), len(first.RTPPackets))
			}
			switch tt.budget {
			case "frame":
				if err := enc.Reconfigure(goh264.EncoderReconfigure{MaxFrameSize: 16}); err != nil {
					t.Fatalf("lower MaxFrameSize: %v", err)
				}
			case "slice":
				if err := enc.Reconfigure(goh264.EncoderReconfigure{SliceMaxBytes: 1}); err != nil {
					t.Fatalf("lower SliceMaxBytes: %v", err)
				}
			default:
				t.Fatalf("unknown budget %q", tt.budget)
			}
			b := patternedI420EncoderFrame(16, 16)
			b.PTS = 0
			b.Y[0] ^= 0x40
			dst := make([]byte, 0, 4096)
			allocs := testing.AllocsPerRun(100, func() {
				out, err := enc.EncodeInto(dst[:0], b)
				if err != nil {
					t.Fatalf("EncodeInto %s: %v", tt.name, err)
				}
				if !out.Dropped || len(out.Data) != 0 || len(out.NALUnits) != 0 || len(out.RTPPackets) != 0 {
					t.Fatalf("%s output = %+v, want empty dropped metadata", tt.name, out)
				}
			})
			t.Logf("%s EncodeInto allocations/run = %.0f", tt.name, allocs)
			if allocs > tt.maxAllocs {
				t.Fatalf("%s EncodeInto allocations/run = %.0f, want <= %.0f", tt.name, allocs, tt.maxAllocs)
			}
			if err := enc.Reconfigure(goh264.EncoderReconfigure{MaxFrameSize: 4096, SliceMaxBytes: 4096}); err != nil {
				t.Fatalf("restore budgets: %v", err)
			}
			recovered, err := enc.EncodeInto(dst[:0], a)
			if err != nil {
				t.Fatalf("EncodeInto after %s: %v", tt.name, err)
			}
			if recovered.Dropped || recovered.IDR || len(recovered.Data) == 0 || len(recovered.RTPPackets) != 0 {
				t.Fatalf("post-%s output dropped=%v idr=%v data=%d rtp=%d, want non-RTP P-skip recovery",
					tt.name, recovered.Dropped, recovered.IDR, len(recovered.Data), len(recovered.RTPPackets))
			}
		})
	}

	for _, tt := range []struct {
		name         string
		outputFormat goh264.EncoderOutputFormat
	}{
		{name: "annexb late drop", outputFormat: goh264.EncoderOutputAnnexB},
		{name: "avc late drop", outputFormat: goh264.EncoderOutputAVC},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cfg := goh264.DefaultEncoderConfig(128, 128)
			cfg.OutputFormat = tt.outputFormat
			cfg.DeblockMode = goh264.EncoderDeblockDisabled
			cfg.RTPMaxPayloadSize = 0
			cfg.FrameDrop = goh264.EncoderFrameDropLate
			cfg.MaxEncodeTimeUS = 10_000_000
			cfg.GOPSize = 10000
			cfg.IDRInterval = 10000
			enc, err := goh264.NewEncoder(cfg)
			if err != nil {
				t.Fatalf("NewEncoder: %v", err)
			}
			a := patternedI420EncoderFrame(128, 128)
			a.PTS = 0
			first, err := enc.EncodeInto(make([]byte, 0, 65536), a)
			if err != nil {
				t.Fatalf("prime IDR: %v", err)
			}
			if !first.IDR || len(first.Data) == 0 || len(first.RTPPackets) != 0 {
				t.Fatalf("prime output idr=%v data=%d rtp=%d, want non-RTP IDR",
					first.IDR, len(first.Data), len(first.RTPPackets))
			}
			if err := enc.Reconfigure(goh264.EncoderReconfigure{MaxEncodeTimeUS: 1}); err != nil {
				t.Fatalf("lower MaxEncodeTimeUS: %v", err)
			}
			b := patternedI420EncoderFrame(128, 128)
			b.PTS = 0
			b.Y[0] ^= 0x40
			dst := make([]byte, 0, 65536)
			allocs := testing.AllocsPerRun(100, func() {
				out, err := enc.EncodeInto(dst[:0], b)
				if err != nil {
					t.Fatalf("EncodeInto %s: %v", tt.name, err)
				}
				if !out.Dropped || len(out.Data) != 0 || len(out.NALUnits) != 0 || len(out.RTPPackets) != 0 {
					t.Fatalf("%s output = %+v, want empty dropped metadata", tt.name, out)
				}
			})
			t.Logf("%s EncodeInto allocations/run = %.0f", tt.name, allocs)
			if allocs > 8 {
				t.Fatalf("%s EncodeInto allocations/run = %.0f, want <= 8", tt.name, allocs)
			}
			if err := enc.Reconfigure(goh264.EncoderReconfigure{MaxEncodeTimeUS: 10_000_000}); err != nil {
				t.Fatalf("restore MaxEncodeTimeUS: %v", err)
			}
			recovered, err := enc.EncodeInto(dst[:0], a)
			if err != nil {
				t.Fatalf("EncodeInto after %s: %v", tt.name, err)
			}
			if recovered.Dropped || recovered.IDR || len(recovered.Data) == 0 || len(recovered.RTPPackets) != 0 {
				t.Fatalf("post-%s output dropped=%v idr=%v data=%d rtp=%d, want non-RTP P-skip recovery",
					tt.name, recovered.Dropped, recovered.IDR, len(recovered.Data), len(recovered.RTPPackets))
			}
		})
	}

	t.Run("rtp max-frame-size drop", func(t *testing.T) {
		cfg := goh264.DefaultEncoderConfig(16, 16)
		cfg.DeblockMode = goh264.EncoderDeblockDisabled
		cfg.RTPMaxPayloadSize = 32
		cfg.MaxFrameSize = 4096
		cfg.GOPSize = 10000
		cfg.IDRInterval = 10000
		enc, err := goh264.NewEncoder(cfg)
		if err != nil {
			t.Fatalf("NewEncoder: %v", err)
		}
		var callbackCalls int
		enc.SetRTPPacketCallback(func(goh264.EncoderRTPPacket, goh264.EncoderRTPPacketMetadata) {
			callbackCalls++
		})
		a := patternedI420EncoderFrame(16, 16)
		a.PTS = 0
		first, err := enc.EncodeInto(make([]byte, 0, 4096), a)
		if err != nil {
			t.Fatalf("prime IDR: %v", err)
		}
		firstPacketCount := len(first.RTPPackets)
		if !first.IDR || firstPacketCount == 0 || callbackCalls != firstPacketCount {
			t.Fatalf("prime output idr=%v packets/callbacks=%d/%d, want RTP IDR callbacks",
				first.IDR, firstPacketCount, callbackCalls)
		}
		if err := enc.Reconfigure(goh264.EncoderReconfigure{MaxFrameSize: 16}); err != nil {
			t.Fatalf("lower MaxFrameSize: %v", err)
		}
		b := patternedI420EncoderFrame(16, 16)
		b.PTS = 0
		b.Y[0] ^= 0x40
		dst := make([]byte, 0, 4096)
		allocs := testing.AllocsPerRun(100, func() {
			out, err := enc.EncodeInto(dst[:0], b)
			if err != nil {
				t.Fatalf("EncodeInto RTP max-frame-size drop: %v", err)
			}
			if !out.Dropped || len(out.Data) != 0 || len(out.NALUnits) != 0 || len(out.RTPPackets) != 0 {
				t.Fatalf("RTP max-frame-size drop output = %+v, want empty dropped metadata", out)
			}
			if callbackCalls != firstPacketCount {
				t.Fatalf("RTP max-frame-size drop invoked callbacks = %d, want still %d", callbackCalls, firstPacketCount)
			}
		})
		t.Logf("rtp max-frame-size drop EncodeInto allocations/run = %.0f", allocs)
		if allocs > 7 {
			t.Fatalf("rtp max-frame-size drop EncodeInto allocations/run = %.0f, want <= 7", allocs)
		}
	})

	t.Run("rtp slice-max-bytes drop", func(t *testing.T) {
		cfg := goh264.DefaultEncoderConfig(16, 16)
		cfg.DeblockMode = goh264.EncoderDeblockDisabled
		cfg.RTPMaxPayloadSize = 32
		cfg.SliceMaxBytes = 4096
		cfg.GOPSize = 10000
		cfg.IDRInterval = 10000
		enc, err := goh264.NewEncoder(cfg)
		if err != nil {
			t.Fatalf("NewEncoder: %v", err)
		}
		var callbackCalls int
		enc.SetRTPPacketCallback(func(goh264.EncoderRTPPacket, goh264.EncoderRTPPacketMetadata) {
			callbackCalls++
		})
		a := patternedI420EncoderFrame(16, 16)
		a.PTS = 0
		first, err := enc.EncodeInto(make([]byte, 0, 4096), a)
		if err != nil {
			t.Fatalf("prime IDR: %v", err)
		}
		firstPacketCount := len(first.RTPPackets)
		if !first.IDR || firstPacketCount == 0 || callbackCalls != firstPacketCount {
			t.Fatalf("prime output idr=%v packets/callbacks=%d/%d, want RTP IDR callbacks",
				first.IDR, firstPacketCount, callbackCalls)
		}
		if err := enc.Reconfigure(goh264.EncoderReconfigure{SliceMaxBytes: 1}); err != nil {
			t.Fatalf("lower SliceMaxBytes: %v", err)
		}
		b := patternedI420EncoderFrame(16, 16)
		b.PTS = 0
		b.Y[0] ^= 0x40
		dst := make([]byte, 0, 4096)
		allocs := testing.AllocsPerRun(100, func() {
			out, err := enc.EncodeInto(dst[:0], b)
			if err != nil {
				t.Fatalf("EncodeInto RTP slice-max-bytes drop: %v", err)
			}
			if !out.Dropped || len(out.Data) != 0 || len(out.NALUnits) != 0 || len(out.RTPPackets) != 0 {
				t.Fatalf("RTP slice-max-bytes drop output = %+v, want empty dropped metadata", out)
			}
			if callbackCalls != firstPacketCount {
				t.Fatalf("RTP slice-max-bytes drop invoked callbacks = %d, want still %d", callbackCalls, firstPacketCount)
			}
		})
		t.Logf("rtp slice-max-bytes drop EncodeInto allocations/run = %.0f", allocs)
		if allocs > 7 {
			t.Fatalf("rtp slice-max-bytes drop EncodeInto allocations/run = %.0f, want <= 7", allocs)
		}
	})

	t.Run("rtp late drop", func(t *testing.T) {
		cfg := goh264.DefaultEncoderConfig(128, 128)
		cfg.DeblockMode = goh264.EncoderDeblockDisabled
		cfg.FrameDrop = goh264.EncoderFrameDropLate
		cfg.MaxEncodeTimeUS = 10_000_000
		cfg.GOPSize = 10000
		cfg.IDRInterval = 10000
		enc, err := goh264.NewEncoder(cfg)
		if err != nil {
			t.Fatalf("NewEncoder: %v", err)
		}
		var callbackCalls int
		enc.SetRTPPacketCallback(func(goh264.EncoderRTPPacket, goh264.EncoderRTPPacketMetadata) {
			callbackCalls++
		})
		a := patternedI420EncoderFrame(128, 128)
		a.PTS = 0
		first, err := enc.EncodeInto(make([]byte, 0, 65536), a)
		if err != nil {
			t.Fatalf("prime IDR: %v", err)
		}
		firstPacketCount := len(first.RTPPackets)
		if !first.IDR || firstPacketCount == 0 || callbackCalls != firstPacketCount {
			t.Fatalf("prime output idr=%v packets/callbacks=%d/%d, want RTP IDR callbacks",
				first.IDR, firstPacketCount, callbackCalls)
		}
		if err := enc.Reconfigure(goh264.EncoderReconfigure{MaxEncodeTimeUS: 1}); err != nil {
			t.Fatalf("lower MaxEncodeTimeUS: %v", err)
		}
		b := patternedI420EncoderFrame(128, 128)
		b.PTS = 0
		b.Y[0] ^= 0x40
		dst := make([]byte, 0, 65536)
		allocs := testing.AllocsPerRun(100, func() {
			out, err := enc.EncodeInto(dst[:0], b)
			if err != nil {
				t.Fatalf("EncodeInto RTP late drop: %v", err)
			}
			if !out.Dropped || len(out.Data) != 0 || len(out.NALUnits) != 0 || len(out.RTPPackets) != 0 {
				t.Fatalf("RTP late drop output = %+v, want empty dropped metadata", out)
			}
			if callbackCalls != firstPacketCount {
				t.Fatalf("RTP late drop invoked callbacks = %d, want still %d", callbackCalls, firstPacketCount)
			}
		})
		t.Logf("rtp late drop EncodeInto allocations/run = %.0f", allocs)
		if allocs > 8 {
			t.Fatalf("rtp late drop EncodeInto allocations/run = %.0f, want <= 8", allocs)
		}
	})
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

	reconfigType := reflect.TypeOf(goh264.EncoderReconfigure{})
	for _, field := range []string{
		"TargetBitrate", "MaxBitrate", "FrameRateNum", "FrameRateDen", "Width", "Height",
		"RTPMaxPayloadSize", "MaxFrameSize", "MaxEncodeTimeUS", "SliceCount", "SliceMaxBytes",
		"Preset", "ForceIDR", "SPSPPSMode", "SPSPPSBeforeIDR", "RecoveryPointSEI",
		"OutputFormat", "RTPPacketizationMode", "STAPA", "RTPPayloadType", "RTPSSRC",
		"RTPTimestampIncrement", "RateControl", "VBVBufferSize", "InitialQP", "MinQP",
		"MaxQP", "FrameDrop", "GOPSize", "IDRInterval", "DeblockMode",
	} {
		if _, ok := reconfigType.FieldByName(field); !ok {
			t.Fatalf("EncoderReconfigure missing roadmap control field %s", field)
		}
	}
}

func TestEncoderRealtimeWebRTCResultSurfaceCoversRoadmap(t *testing.T) {
	for _, tt := range []struct {
		name   string
		typ    reflect.Type
		fields []string
	}{
		{
			name: "EncoderCrop",
			typ:  reflect.TypeOf(goh264.EncoderCrop{}),
			fields: []string{
				"Left", "Right", "Top", "Bottom",
			},
		},
		{
			name: "EncoderColorConfig",
			typ:  reflect.TypeOf(goh264.EncoderColorConfig{}),
			fields: []string{
				"SARNum", "SARDen", "VideoFormat", "FullRange", "ColorPrimaries",
				"ColorTransfer", "ColorMatrix", "ChromaSampleLocTypeTopField",
				"ChromaSampleLocTypeBottomField",
			},
		},
		{
			name: "EncoderFrame",
			typ:  reflect.TypeOf(goh264.EncoderFrame{}),
			fields: []string{
				"Y", "Cb", "Cr", "StrideY", "StrideCb", "StrideCr",
				"Width", "Height", "PTS", "Duration", "ForceIDR", "Color",
			},
		},
		{
			name: "EncoderNALUnit",
			typ:  reflect.TypeOf(goh264.EncoderNALUnit{}),
			fields: []string{
				"Type", "Offset", "Size", "KeyFrame", "ParameterSet",
			},
		},
		{
			name: "EncoderRTPPacket",
			typ:  reflect.TypeOf(goh264.EncoderRTPPacket{}),
			fields: []string{
				"Data", "Payload", "PayloadType", "SequenceNumber", "Timestamp", "SSRC", "Marker",
			},
		},
		{
			name: "EncoderRTPPacketMetadata",
			typ:  reflect.TypeOf(goh264.EncoderRTPPacketMetadata{}),
			fields: []string{
				"PacketIndex", "PacketCount", "FramePTS", "FrameDTS", "RTPTime", "KeyFrame", "IDR",
				"PayloadFormat", "NALUnitType", "NALUnitCount", "StartOfNAL", "EndOfNAL", "ParameterSet",
			},
		},
		{
			name: "EncodedFrame",
			typ:  reflect.TypeOf(goh264.EncodedFrame{}),
			fields: []string{
				"Data", "NALUnits", "RTPPackets", "KeyFrame", "IDR", "PTS", "DTS", "RTPTime", "Dropped",
			},
		},
		{
			name: "EncoderParameterSets",
			typ:  reflect.TypeOf(goh264.EncoderParameterSets{}),
			fields: []string{
				"SPS", "PPS", "AnnexB", "AVCDecoderConfigurationRecord",
			},
		},
		{
			name: "EncoderSEI",
			typ:  reflect.TypeOf(goh264.EncoderSEI{}),
			fields: []string{
				"NAL", "AnnexB", "AVC",
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			for _, field := range tt.fields {
				if _, ok := tt.typ.FieldByName(field); !ok {
					t.Fatalf("%s missing public field %s", tt.name, field)
				}
			}
		})
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

func cloneI420EncoderFrame(frame goh264.EncoderFrame) goh264.EncoderFrame {
	clone := frame
	clone.Y = append([]byte(nil), frame.Y...)
	clone.Cb = append([]byte(nil), frame.Cb...)
	clone.Cr = append([]byte(nil), frame.Cr...)
	return clone
}

func mutateI420EncoderFramePlanes(frame *goh264.EncoderFrame) {
	for i := range frame.Y {
		frame.Y[i] ^= 0xff
	}
	for i := range frame.Cb {
		frame.Cb[i] ^= 0x7f
	}
	for i := range frame.Cr {
		frame.Cr[i] ^= 0x3f
	}
}

func cloneEncoderRTPPackets(packets []goh264.EncoderRTPPacket) []goh264.EncoderRTPPacket {
	clones := append([]goh264.EncoderRTPPacket(nil), packets...)
	for i := range clones {
		clones[i].Data = append([]byte(nil), packets[i].Data...)
		clones[i].Payload = append([]byte(nil), packets[i].Payload...)
	}
	return clones
}

func encoderPrefilledCallerBuffer() ([]byte, []byte) {
	backing := bytes.Repeat([]byte{0xcc}, 4096)
	prefix := []byte{0xde, 0xad, 0xbe, 0xef, 0x55}
	copy(backing, prefix)
	return backing[:len(prefix)], append([]byte(nil), backing...)
}

func assertEncoderCallerBufferUnchanged(t *testing.T, dst []byte, before []byte) {
	t.Helper()
	after := dst[:cap(dst)]
	if !bytes.Equal(after, before) {
		t.Fatalf("EncodeInto mutated caller buffer on non-output path")
	}
}

func setConstantI420Chroma(frame *goh264.EncoderFrame, cb byte, cr byte) {
	chromaWidth := frame.Width / 2
	chromaHeight := frame.Height / 2
	for y := 0; y < chromaHeight; y++ {
		for x := 0; x < chromaWidth; x++ {
			frame.Cb[y*frame.StrideCb+x] = cb
			frame.Cr[y*frame.StrideCr+x] = cr
		}
	}
}

func integerMotionI420EncoderFrame(reference goh264.EncoderFrame, dx int, dy int) goh264.EncoderFrame {
	frame := validI420EncoderFrame(reference.Width, reference.Height)
	frame.PTS = reference.PTS
	frame.Duration = reference.Duration
	for y := 0; y < frame.Height; y++ {
		refY := clampEncoderTestCoord(y+dy, frame.Height)
		for x := 0; x < frame.Width; x++ {
			refX := clampEncoderTestCoord(x+dx, frame.Width)
			frame.Y[y*frame.StrideY+x] = reference.Y[refY*reference.StrideY+refX]
		}
	}
	chromaWidth := frame.Width / 2
	chromaHeight := frame.Height / 2
	chromaDX := dx / 2
	chromaDY := dy / 2
	for y := 0; y < chromaHeight; y++ {
		refY := clampEncoderTestCoord(y+chromaDY, chromaHeight)
		for x := 0; x < chromaWidth; x++ {
			refX := clampEncoderTestCoord(x+chromaDX, chromaWidth)
			frame.Cb[y*frame.StrideCb+x] = reference.Cb[refY*reference.StrideCb+refX]
			frame.Cr[y*frame.StrideCr+x] = reference.Cr[refY*reference.StrideCr+refX]
		}
	}
	return frame
}

type encoderTestMotion struct {
	dx int
	dy int
}

func perMacroblockMotionI420EncoderFrame(reference goh264.EncoderFrame, motions []encoderTestMotion) goh264.EncoderFrame {
	frame := validI420EncoderFrame(reference.Width, reference.Height)
	frame.PTS = reference.PTS
	frame.Duration = reference.Duration
	mbWidth := reference.Width / 16
	mbHeight := reference.Height / 16
	if len(motions) != mbWidth*mbHeight {
		panic("per-macroblock motion count does not match frame")
	}
	for mbY := 0; mbY < mbHeight; mbY++ {
		for mbX := 0; mbX < mbWidth; mbX++ {
			motion := motions[mbY*mbWidth+mbX]
			left := mbX * 16
			top := mbY * 16
			for y := 0; y < 16; y++ {
				refY := clampEncoderTestCoord(top+y+motion.dy, frame.Height)
				for x := 0; x < 16; x++ {
					refX := clampEncoderTestCoord(left+x+motion.dx, frame.Width)
					frame.Y[(top+y)*frame.StrideY+left+x] = reference.Y[refY*reference.StrideY+refX]
				}
			}
			chromaLeft := mbX * 8
			chromaTop := mbY * 8
			chromaWidth := frame.Width / 2
			chromaHeight := frame.Height / 2
			chromaDX := motion.dx / 2
			chromaDY := motion.dy / 2
			for y := 0; y < 8; y++ {
				refY := clampEncoderTestCoord(chromaTop+y+chromaDY, chromaHeight)
				for x := 0; x < 8; x++ {
					refX := clampEncoderTestCoord(chromaLeft+x+chromaDX, chromaWidth)
					frame.Cb[(chromaTop+y)*frame.StrideCb+chromaLeft+x] = reference.Cb[refY*reference.StrideCb+refX]
					frame.Cr[(chromaTop+y)*frame.StrideCr+chromaLeft+x] = reference.Cr[refY*reference.StrideCr+refX]
				}
			}
		}
	}
	return frame
}

func primedI420EncoderPool(t *testing.T, cfg goh264.EncoderConfig, frame goh264.EncoderFrame, count int) []*goh264.Encoder {
	t.Helper()
	encs := make([]*goh264.Encoder, count)
	for i := range encs {
		enc, err := goh264.NewEncoder(cfg)
		if err != nil {
			t.Fatalf("NewEncoder[%d]: %v", i, err)
		}
		if _, err := enc.EncodeInto(make([]byte, 0, 4096), frame); err != nil {
			t.Fatalf("prime IDR[%d]: %v", i, err)
		}
		encs[i] = enc
	}
	return encs
}

func clampEncoderTestCoord(v int, limit int) int {
	if v < 0 {
		return 0
	}
	if v >= limit {
		return limit - 1
	}
	return v
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

func assertEncodedFrameNALUnitIndexes(t *testing.T, out goh264.EncodedFrame, format goh264.EncoderOutputFormat) {
	t.Helper()
	assertEncodedFrameNALUnitIndexesFrom(t, out, format, 0)
}

func assertEncodedFrameNALUnitIndexesFrom(t *testing.T, out goh264.EncodedFrame, format goh264.EncoderOutputFormat, dataStart int) {
	t.Helper()
	if out.Dropped {
		t.Fatal("cannot validate NAL indexes for dropped frame")
	}
	if dataStart < 0 || dataStart > len(out.Data) {
		t.Fatalf("encoded frame data start = %d outside data length %d", dataStart, len(out.Data))
	}
	accessUnit := out.Data[dataStart:]
	if len(accessUnit) == 0 || len(out.NALUnits) == 0 {
		t.Fatalf("encoded frame data/nals = %d/%d, want populated access unit", len(out.Data), len(out.NALUnits))
	}
	var parsed []h264.NALUnit
	var err error
	switch format {
	case goh264.EncoderOutputAnnexB, goh264.EncoderOutputRTP:
		parsed, err = h264.SplitAnnexB(accessUnit)
	case goh264.EncoderOutputAVC:
		parsed, err = h264.SplitAVCC(accessUnit, 4)
	default:
		t.Fatalf("unknown output format %v", format)
	}
	if err != nil {
		t.Fatalf("split encoded frame data: %v", err)
	}
	if len(parsed) != len(out.NALUnits) {
		t.Fatalf("parsed NAL count = %d, public NAL count = %d", len(parsed), len(out.NALUnits))
	}
	for i, unit := range out.NALUnits {
		if unit.Offset < dataStart || unit.Size <= 0 || unit.Offset+unit.Size > len(out.Data) {
			t.Fatalf("NAL[%d] offset/size = %d/%d outside access unit range [%d,%d)", i, unit.Offset, unit.Size, dataStart, len(out.Data))
		}
		raw := out.Data[unit.Offset : unit.Offset+unit.Size]
		if !bytes.Equal(raw, parsed[i].Raw) {
			t.Fatalf("NAL[%d] raw bytes do not match indexed EncodedFrame.Data", i)
		}
		if unit.Type != uint8(parsed[i].Type) || unit.Type != raw[0]&0x1f {
			t.Fatalf("NAL[%d] type = %d parsed=%d raw=%d", i, unit.Type, parsed[i].Type, raw[0]&0x1f)
		}
		wantParameterSet := unit.Type == 7 || unit.Type == 8
		if unit.ParameterSet != wantParameterSet {
			t.Fatalf("NAL[%d] parameterSet = %v, want %v", i, unit.ParameterSet, wantParameterSet)
		}
		wantKeyFrame := unit.Type == 5 || unit.ParameterSet
		if unit.KeyFrame != wantKeyFrame {
			t.Fatalf("NAL[%d] keyFrame = %v, want %v", i, unit.KeyFrame, wantKeyFrame)
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

func assertEncoderVCLFrameNums(t *testing.T, annexB []byte, wantTypes []uint8, wantFrameNums []uint32) {
	t.Helper()
	nals, err := h264.SplitAnnexB(annexB)
	if err != nil {
		t.Fatalf("SplitAnnexB: %v", err)
	}
	var spsList [32]*h264.SPS
	var ppsList [256]*h264.PPS
	var gotTypes []uint8
	var gotFrameNums []uint32
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
			gotFrameNums = append(gotFrameNums, sh.FrameNum)
		}
	}
	if !reflect.DeepEqual(gotTypes, wantTypes) || !reflect.DeepEqual(gotFrameNums, wantFrameNums) {
		t.Fatalf("VCL types/frame nums = %v/%v, want %v/%v",
			gotTypes, gotFrameNums, wantTypes, wantFrameNums)
	}
}

func assertEncoderVCLQScales(t *testing.T, annexB []byte, wantTypes []uint8, wantQScales []uint32) {
	t.Helper()
	nals, err := h264.SplitAnnexB(annexB)
	if err != nil {
		t.Fatalf("SplitAnnexB: %v", err)
	}
	var spsList [32]*h264.SPS
	var ppsList [256]*h264.PPS
	var gotTypes []uint8
	var gotQScales []uint32
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
			gotQScales = append(gotQScales, sh.QScale)
		}
	}
	if !reflect.DeepEqual(gotTypes, wantTypes) || !reflect.DeepEqual(gotQScales, wantQScales) {
		t.Fatalf("VCL types/QScales = %v/%v, want %v/%v",
			gotTypes, gotQScales, wantTypes, wantQScales)
	}
}

func assertEncoderVCLDeblocks(t *testing.T, annexB []byte, wantTypes []uint8, wantDeblocks []int32) {
	t.Helper()
	nals, err := h264.SplitAnnexB(annexB)
	if err != nil {
		t.Fatalf("SplitAnnexB: %v", err)
	}
	var spsList [32]*h264.SPS
	var ppsList [256]*h264.PPS
	var gotTypes []uint8
	var gotDeblocks []int32
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
			gotDeblocks = append(gotDeblocks, sh.DeblockingFilter)
		}
	}
	if !reflect.DeepEqual(gotTypes, wantTypes) || !reflect.DeepEqual(gotDeblocks, wantDeblocks) {
		t.Fatalf("VCL types/deblocks = %v/%v, want %v/%v",
			gotTypes, gotDeblocks, wantTypes, wantDeblocks)
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
		if len(pkt.Data) != 12+len(pkt.Payload) {
			t.Fatalf("packet[%d] full RTP packet length = %d, want header plus payload %d",
				i, len(pkt.Data), 12+len(pkt.Payload))
		}
		if !bytes.Equal(pkt.Data[12:], pkt.Payload) {
			t.Fatalf("packet[%d] RTP payload bytes do not match Data payload", i)
		}
		if cap(pkt.Data) != len(pkt.Data) {
			t.Fatalf("packet[%d] Data cap = %d, want clipped length %d", i, cap(pkt.Data), len(pkt.Data))
		}
		if cap(pkt.Payload) != len(pkt.Payload) {
			t.Fatalf("packet[%d] Payload cap = %d, want clipped length %d", i, cap(pkt.Payload), len(pkt.Payload))
		}
		if pkt.Data[0] != 0x80 {
			t.Fatalf("packet[%d] RTP version/P/X/CC byte = %#x, want 0x80", i, pkt.Data[0])
		}
		if pkt.PayloadType != payloadType {
			t.Fatalf("packet[%d] payload type = %d, want %d", i, pkt.PayloadType, payloadType)
		}
		if got := pkt.Data[1] & 0x7f; got != payloadType {
			t.Fatalf("packet[%d] RTP header payload type = %d, want %d", i, got, payloadType)
		}
		if got := pkt.Data[1]&0x80 != 0; got != pkt.Marker {
			t.Fatalf("packet[%d] RTP marker header = %v, want packet marker %v", i, got, pkt.Marker)
		}
		if pkt.SSRC != ssrc {
			t.Fatalf("packet[%d] SSRC = %#x, want %#x", i, pkt.SSRC, ssrc)
		}
		if got := binary.BigEndian.Uint32(pkt.Data[8:12]); got != ssrc {
			t.Fatalf("packet[%d] RTP header SSRC = %#x, want %#x", i, got, ssrc)
		}
		if pkt.SequenceNumber != firstSeq+uint16(i) {
			t.Fatalf("packet[%d] sequence = %d, want %d", i, pkt.SequenceNumber, firstSeq+uint16(i))
		}
		if got := binary.BigEndian.Uint16(pkt.Data[2:4]); got != pkt.SequenceNumber {
			t.Fatalf("packet[%d] RTP header sequence = %d, want %d", i, got, pkt.SequenceNumber)
		}
		if got := binary.BigEndian.Uint32(pkt.Data[4:8]); got != pkt.Timestamp {
			t.Fatalf("packet[%d] RTP header timestamp = %d, want %d", i, got, pkt.Timestamp)
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

func assertEncoderRTPPayloadLimit(t *testing.T, packets []goh264.EncoderRTPPacket, maxPayloadSize int) {
	t.Helper()
	if len(packets) == 0 {
		t.Fatal("RTP packet list is empty")
	}
	for i, pkt := range packets {
		if len(pkt.Payload) > maxPayloadSize {
			t.Fatalf("packet[%d] payload size = %d, max %d", i, len(pkt.Payload), maxPayloadSize)
		}
		if pkt.Marker != (i == len(packets)-1) {
			t.Fatalf("packet[%d] marker = %v, want only final marker", i, pkt.Marker)
		}
	}
}

func assertEncoderRTPHasFUA(t *testing.T, packets []goh264.EncoderRTPPacket) {
	t.Helper()
	var sawStart, sawEnd bool
	for i, pkt := range packets {
		if len(pkt.Payload) < 2 || pkt.Payload[0]&0x1f != 28 {
			continue
		}
		if pkt.Payload[1]&0x80 != 0 {
			sawStart = true
		}
		if pkt.Payload[1]&0x40 != 0 {
			sawEnd = true
		}
		if nalType := pkt.Payload[1] & 0x1f; nalType != 1 && nalType != 5 {
			t.Fatalf("packet[%d] FU-A NAL type = %d, want VCL type 1 or 5", i, nalType)
		}
	}
	if !sawStart || !sawEnd {
		t.Fatalf("FU-A start/end = %v/%v, want both true", sawStart, sawEnd)
	}
}

func assertEncoderRTPMode0RawNALPackets(t *testing.T, out goh264.EncodedFrame, maxPayloadSize int) {
	t.Helper()
	assertEncoderRTPSingleNALPackets(t, out, maxPayloadSize)
}

func assertEncoderRTPSingleNALPackets(t *testing.T, out goh264.EncodedFrame, maxPayloadSize int) {
	t.Helper()
	if len(out.RTPPackets) != len(out.NALUnits) {
		t.Fatalf("single-NAL RTP packets = %d, want one packet per NAL %d", len(out.RTPPackets), len(out.NALUnits))
	}
	for i, pkt := range out.RTPPackets {
		unit := out.NALUnits[i]
		wantPayload := out.Data[unit.Offset : unit.Offset+unit.Size]
		if !bytes.Equal(pkt.Payload, wantPayload) {
			t.Fatalf("packet[%d] payload does not match raw NAL", i)
		}
		if maxPayloadSize > 0 && len(pkt.Payload) > maxPayloadSize {
			t.Fatalf("packet[%d] payload size = %d, max %d", i, len(pkt.Payload), maxPayloadSize)
		}
		if typ := pkt.Payload[0] & 0x1f; typ == 24 || typ == 28 {
			t.Fatalf("packet[%d] payload type = %d, want single raw NAL", i, typ)
		}
		if pkt.Marker != (i == len(out.RTPPackets)-1) {
			t.Fatalf("packet[%d] marker = %v, want only final marker", i, pkt.Marker)
		}
		if pkt.Timestamp != out.RTPTime {
			t.Fatalf("packet[%d] timestamp = %d, want %d", i, pkt.Timestamp, out.RTPTime)
		}
	}
}

func assertEncoderRTPSingleNALCallbackMetadata(t *testing.T, callbackPackets []goh264.EncoderRTPPacket, callbackMetadata []goh264.EncoderRTPPacketMetadata, out goh264.EncodedFrame, frame goh264.EncoderFrame, cfg goh264.EncoderConfig, keyFrame bool, idr bool) {
	t.Helper()
	if len(callbackPackets) != len(out.RTPPackets) || len(callbackMetadata) != len(out.RTPPackets) {
		t.Fatalf("callback packets/meta = %d/%d, want RTP packet count %d",
			len(callbackPackets), len(callbackMetadata), len(out.RTPPackets))
	}
	for i, meta := range callbackMetadata {
		pkt := callbackPackets[i]
		wantType := out.NALUnits[i].Type
		if meta.PacketIndex != i || meta.PacketCount != len(out.RTPPackets) {
			t.Fatalf("callback meta[%d] index/count = %d/%d, want %d/%d",
				i, meta.PacketIndex, meta.PacketCount, i, len(out.RTPPackets))
		}
		if meta.FramePTS != frame.PTS || meta.FrameDTS != frame.PTS ||
			meta.RTPTime != out.RTPTime || meta.KeyFrame != keyFrame || meta.IDR != idr {
			t.Fatalf("callback meta[%d] frame fields = %+v, want key=%v idr=%v timing metadata",
				i, meta, keyFrame, idr)
		}
		if pkt.SequenceNumber != out.RTPPackets[i].SequenceNumber ||
			pkt.Timestamp != out.RTPPackets[i].Timestamp ||
			pkt.PayloadType != cfg.RTPPayloadType ||
			pkt.SSRC != cfg.RTPSSRC ||
			pkt.Marker != (i == len(out.RTPPackets)-1) ||
			!bytes.Equal(pkt.Payload, out.RTPPackets[i].Payload) ||
			!bytes.Equal(pkt.Data, out.RTPPackets[i].Data) {
			t.Fatalf("callback packet[%d] metadata = %+v, want returned RTP packet fields", i, pkt)
		}
		assertEncoderRTPCallbackPacketDoesNotAliasReturned(t, pkt, out.RTPPackets[i], i)
		if meta.PayloadFormat != goh264.EncoderRTPPayloadSingleNAL ||
			meta.NALUnitType != wantType ||
			meta.NALUnitCount != 1 ||
			!meta.StartOfNAL || !meta.EndOfNAL ||
			meta.ParameterSet != (wantType == 7 || wantType == 8) {
			t.Fatalf("callback meta[%d] = %+v, want complete single-NAL type %d",
				i, meta, wantType)
		}
	}
}

func assertEncoderRTPCallbackPacketDoesNotAliasReturned(t *testing.T, callbackPkt goh264.EncoderRTPPacket, returnedPkt goh264.EncoderRTPPacket, index int) {
	t.Helper()
	if len(callbackPkt.Data) != 12+len(callbackPkt.Payload) {
		t.Fatalf("callback packet[%d] Data length = %d, want RTP header plus payload %d",
			index, len(callbackPkt.Data), 12+len(callbackPkt.Payload))
	}
	if !bytes.Equal(callbackPkt.Data[12:], callbackPkt.Payload) {
		t.Fatalf("callback packet[%d] Payload bytes are not backed by Data payload bytes", index)
	}
	if cap(callbackPkt.Data) != len(callbackPkt.Data) {
		t.Fatalf("callback packet[%d] Data cap = %d, want clipped length %d", index, cap(callbackPkt.Data), len(callbackPkt.Data))
	}
	if cap(callbackPkt.Payload) != len(callbackPkt.Payload) {
		t.Fatalf("callback packet[%d] Payload cap = %d, want clipped length %d", index, cap(callbackPkt.Payload), len(callbackPkt.Payload))
	}
	if len(callbackPkt.Payload) != 0 {
		callbackPkt.Payload[0] ^= 0xff
		if callbackPkt.Data[12] != callbackPkt.Payload[0] {
			t.Fatalf("callback packet[%d] Payload is not a view over Data payload bytes", index)
		}
		callbackPkt.Payload[0] ^= 0xff
	}
	if len(callbackPkt.Payload) != 0 {
		before := append([]byte(nil), returnedPkt.Payload...)
		callbackPkt.Payload[0] ^= 0xff
		if !bytes.Equal(returnedPkt.Payload, before) {
			t.Fatalf("callback packet[%d] Payload aliases returned RTP packet storage", index)
		}
	}
	if len(callbackPkt.Data) != 0 {
		before := append([]byte(nil), returnedPkt.Data...)
		callbackPkt.Data[0] ^= 0xff
		if !bytes.Equal(returnedPkt.Data, before) {
			t.Fatalf("callback packet[%d] Data aliases returned RTP packet storage", index)
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
