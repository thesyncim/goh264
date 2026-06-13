// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

import (
	"encoding/hex"
	"errors"
	"testing"
)

func TestDecodeAVCDecoderConfigurationRecord(t *testing.T) {
	sps := mustHex(t, "6742c01eddec0440000003004000000300a3c58be0")
	pps := mustHex(t, "68ce0fc8")
	config := avcConfigRecord(t, 4, [][]byte{sps}, [][]byte{pps})

	cfg, err := DecodeAVCDecoderConfigurationRecord(config)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.NALLengthSize != 4 || cfg.FirstSPSID != 0 {
		t.Fatalf("config = nal length %d first SPS %d", cfg.NALLengthSize, cfg.FirstSPSID)
	}
	if cfg.SPS[0] == nil || cfg.SPS[0].ProfileIDC != 66 || cfg.SPS[0].Width != 16 || cfg.SPS[0].Height != 16 {
		t.Fatalf("sps = %+v", cfg.SPS[0])
	}
	if cfg.PPS[0] == nil || cfg.PPS[0].SPS != cfg.SPS[0] || cfg.PPS[0].CABAC != 0 {
		t.Fatalf("pps = %+v", cfg.PPS[0])
	}
}

func TestIsAVCDecoderConfigurationRecord(t *testing.T) {
	sps := mustHex(t, "6742c01eddec0440000003004000000300a3c58be0")
	pps := mustHex(t, "68ce0fc8")
	config := avcConfigRecord(t, 4, [][]byte{sps}, [][]byte{pps})
	if !IsAVCDecoderConfigurationRecord(config) {
		t.Fatal("avcC config was not detected")
	}
	for _, tt := range []struct {
		name string
		data []byte
	}{
		{name: "empty", data: nil},
		{name: "no sps", data: avcConfigRecord(t, 4, nil, [][]byte{pps})},
		{name: "no pps", data: avcConfigRecord(t, 4, [][]byte{sps}, nil)},
		{name: "wrong sps type", data: avcConfigRecord(t, 4, [][]byte{pps}, [][]byte{pps})},
		{name: "wrong pps type", data: avcConfigRecord(t, 4, [][]byte{sps}, [][]byte{sps})},
		{name: "bad reserved bits", data: append([]byte(nil), config...)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			data := tt.data
			if tt.name == "bad reserved bits" {
				data[4] &^= 0x80
			}
			if IsAVCDecoderConfigurationRecord(data) {
				t.Fatal("unexpected avcC detection")
			}
		})
	}
}

func TestDecodeAVCDecoderConfigurationRecordRejectsInvalidData(t *testing.T) {
	sps := mustHex(t, "6742c01eddec0440000003004000000300a3c58be0")
	pps := mustHex(t, "68ce0fc8")
	for _, tt := range []struct {
		name string
		data []byte
	}{
		{name: "empty", data: nil},
		{name: "not avcc", data: append([]byte{0}, avcConfigRecord(t, 4, [][]byte{sps}, [][]byte{pps})[1:]...)},
		{name: "short", data: []byte{1, 0x42, 0xc0, 0x1e, 0xff, 0xe1}},
		{name: "bad length-size reserved bits", data: avcConfigRecordWithMutation(t, 4, [][]byte{sps}, [][]byte{pps}, func(data []byte) { data[4] &^= 0x80 })},
		{name: "bad sps-count reserved bits", data: avcConfigRecordWithMutation(t, 4, [][]byte{sps}, [][]byte{pps}, func(data []byte) { data[5] &^= 0x80 })},
		{name: "missing pps", data: avcConfigRecord(t, 4, [][]byte{sps}, nil)},
		{name: "oversized sps", data: []byte{1, 0x42, 0xc0, 0x1e, 0xff, 0xe1, 0xff, 0xff, 0x67}},
		{name: "wrong pps type", data: avcConfigRecord(t, 4, [][]byte{sps}, [][]byte{sps})},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := DecodeAVCDecoderConfigurationRecord(tt.data); err == nil {
				t.Fatal("expected invalid data")
			}
		})
	}
}

func TestDecodeAVCDecoderConfigurationRecordRejectsOverflowedInput(t *testing.T) {
	sps := mustHex(t, "6742c01eddec0440000003004000000300a3c58be0")
	pps := mustHex(t, "68ce0fc8")
	config := avcConfigRecord(t, 4, [][]byte{sps}, [][]byte{pps})
	overflowed := fakeH264SliceLen(&config[0], maxInt/2+1)

	if IsAVCDecoderConfigurationRecord(overflowed) {
		t.Fatal("overflowed avcC config was detected")
	}
	if _, err := DecodeAVCDecoderConfigurationRecord(overflowed); !errors.Is(err, ErrInvalidData) {
		t.Fatalf("overflowed avcC decode error = %v, want ErrInvalidData", err)
	}
}

func avcConfigRecord(t *testing.T, nalLengthSize int, spsNals [][]byte, ppsNals [][]byte) []byte {
	t.Helper()
	if nalLengthSize < 1 || nalLengthSize > 4 {
		t.Fatalf("bad nalLengthSize %d", nalLengthSize)
	}
	if len(spsNals) > 31 || len(ppsNals) > 255 {
		t.Fatalf("too many parameter sets")
	}
	profile, constraints, level := byte(0), byte(0), byte(0)
	if len(spsNals) != 0 && len(spsNals[0]) >= 4 {
		profile, constraints, level = spsNals[0][1], spsNals[0][2], spsNals[0][3]
	}
	out := []byte{1, profile, constraints, level, 0xfc | byte(nalLengthSize-1), 0xe0 | byte(len(spsNals))}
	for _, nal := range spsNals {
		out = appendAVCConfigNAL(t, out, nal)
	}
	out = append(out, byte(len(ppsNals)))
	for _, nal := range ppsNals {
		out = appendAVCConfigNAL(t, out, nal)
	}
	return out
}

func avcConfigRecordWithMutation(t *testing.T, nalLengthSize int, spsNals [][]byte, ppsNals [][]byte, mutate func([]byte)) []byte {
	t.Helper()
	data := avcConfigRecord(t, nalLengthSize, spsNals, ppsNals)
	mutate(data)
	return data
}

func appendAVCConfigNAL(t *testing.T, dst []byte, nal []byte) []byte {
	t.Helper()
	if len(nal) == 0 || len(nal) > 0xffff {
		t.Fatalf("bad NAL length %d", len(nal))
	}
	return append(append(dst, byte(len(nal)>>8), byte(len(nal))), nal...)
}

func mustHex(t *testing.T, s string) []byte {
	t.Helper()
	data, err := hex.DecodeString(s)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
