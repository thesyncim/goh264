// SPDX-License-Identifier: LGPL-2.1-or-later

package goh264

import (
	"errors"
	"testing"
)

func TestDecoderAVCConfigReflectsPublicConfigurationPaths(t *testing.T) {
	encCfg := DefaultEncoderConfig(16, 16)
	headers, err := encCfg.ParameterSets()
	if err != nil {
		t.Fatalf("ParameterSets: %v", err)
	}
	avcc := headers.AVCC()

	if cfg, err := ParseAVCC(avcc); err != nil {
		t.Fatalf("package ParseAVCC: %v", err)
	} else {
		assertDecoderAVCConfig(t, cfg)
	}

	dec := NewDecoder()
	if cfg, err := dec.AVCConfig(); !errors.Is(err, ErrInvalidData) {
		t.Fatalf("fresh AVCConfig = %+v/%v, want ErrInvalidData", cfg, err)
	}

	if cfg, err := dec.ParseAVCC(avcc); err != nil {
		t.Fatalf("decoder ParseAVCC: %v", err)
	} else {
		assertDecoderAVCConfig(t, cfg)
	}
	if cfg, err := dec.AVCConfig(); err != nil {
		t.Fatalf("stored AVCConfig after ParseAVCC: %v", err)
	} else {
		assertDecoderAVCConfig(t, cfg)
	}

	if err := dec.Reset(); err != nil {
		t.Fatalf("Reset after ParseAVCC: %v", err)
	}
	if cfg, err := dec.AVCConfig(); !errors.Is(err, ErrInvalidData) {
		t.Fatalf("reset AVCConfig = %+v/%v, want ErrInvalidData", cfg, err)
	}

	if frames, err := dec.DecodeFrames(avcc); err != nil || len(frames) != 0 {
		t.Fatalf("DecodeFrames avcC = %d frames/%v, want config-only success", len(frames), err)
	}
	if cfg, err := dec.AVCConfig(); err != nil {
		t.Fatalf("stored AVCConfig after DecodeFrames avcC: %v", err)
	} else {
		assertDecoderAVCConfig(t, cfg)
	}

	if err := dec.Reset(); err != nil {
		t.Fatalf("Reset after DecodeFrames avcC: %v", err)
	}
	if frames, err := dec.DecodeAVCCFrames(avcc, nil); err != nil || len(frames) != 0 {
		t.Fatalf("DecodeAVCCFrames config-only = %d frames/%v, want config update and flush success", len(frames), err)
	}
	if cfg, err := dec.AVCConfig(); err != nil {
		t.Fatalf("stored AVCConfig after DecodeAVCCFrames: %v", err)
	} else {
		assertDecoderAVCConfig(t, cfg)
	}
}

func assertDecoderAVCConfig(t *testing.T, cfg AVCConfig) {
	t.Helper()
	if cfg.NALLengthSize != 4 ||
		cfg.StreamInfo.Width != 16 ||
		cfg.StreamInfo.Height != 16 ||
		cfg.StreamInfo.ChromaFormatIDC != 1 ||
		cfg.StreamInfo.BitDepthLuma != 8 ||
		cfg.StreamInfo.BitDepthChroma != 8 {
		t.Fatalf("AVCConfig = %+v, want 4-byte 16x16 4:2:0 8-bit config", cfg)
	}
}
