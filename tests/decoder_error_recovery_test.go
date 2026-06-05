// SPDX-License-Identifier: LGPL-2.1-or-later

package goh264_test

import (
	"testing"

	"github.com/thesyncim/goh264/internal/h264"
)

func TestDecodeConfiguredAVCFramesRecoversAfterDamagedSlicePacket(t *testing.T) {
	data := decodeHexFixture(t, black16IPAnnexBHex)
	config, samples := annexBToAVCConfigAndSamples(t, data, 4)
	if len(samples) != 2 {
		t.Fatalf("samples = %d, want 2", len(samples))
	}

	dec := NewDecoder()
	if _, err := dec.ParseAVCDecoderConfigurationRecord(config); err != nil {
		t.Fatal(err)
	}

	frames, err := dec.DecodeConfiguredAVCFrames(samples[0])
	if err != nil {
		t.Fatal(err)
	}
	assertFrameMD5Strings(t, frames, []string{"8aaefe0adcea094cfb5161a060bab4e2"})

	damaged := truncateFirstVCLAVCPayload(t, samples[1], 4)
	if out, err := dec.DecodeConfiguredAVCFrames(damaged); err == nil {
		t.Fatalf("damaged packet decoded frames=%d, want error", len(out))
	}

	frames, err = dec.DecodeConfiguredAVCFrames(samples[1])
	if err != nil {
		t.Fatalf("decode after damaged packet: %v", err)
	}
	assertFrameMD5Strings(t, frames, []string{"8aaefe0adcea094cfb5161a060bab4e2"})
}

func TestDecodeAVCWithConfigurationRecordRecoversAfterDamagedSlicePacket(t *testing.T) {
	data := decodeHexFixture(t, black16IPAnnexBHex)
	config, samples := annexBToAVCConfigAndSamples(t, data, 4)
	if len(samples) != 2 {
		t.Fatalf("samples = %d, want 2", len(samples))
	}

	dec := NewDecoder()
	frames, err := dec.DecodeAVCFramesWithConfigurationRecord(config, samples[0])
	if err != nil {
		t.Fatal(err)
	}
	assertFrameMD5Strings(t, frames, []string{"8aaefe0adcea094cfb5161a060bab4e2"})

	damaged := truncateFirstVCLAVCPayload(t, samples[1], 4)
	if out, err := dec.DecodeAVCFramesWithConfigurationRecord(config, damaged); err == nil {
		t.Fatalf("damaged packet decoded frames=%d, want error", len(out))
	}

	frames, err = dec.DecodeAVCFramesWithConfigurationRecord(config, samples[1])
	if err != nil {
		t.Fatalf("decode after damaged packet: %v", err)
	}
	assertFrameMD5Strings(t, frames, []string{"8aaefe0adcea094cfb5161a060bab4e2"})
}

func TestDecodePacketFramesAVCRecoversAfterDamagedSlicePacket(t *testing.T) {
	data := decodeHexFixture(t, black16IPAnnexBHex)
	config, samples := annexBToAVCConfigAndSamples(t, data, 4)
	if len(samples) != 2 {
		t.Fatalf("samples = %d, want 2", len(samples))
	}

	dec := NewDecoder()
	frames, err := dec.DecodePacketFrames(Packet{
		Data:     samples[0],
		SideData: []PacketSideData{{Type: PacketSideDataNewExtradata, Data: config}},
	})
	if err != nil {
		t.Fatal(err)
	}
	assertFrameMD5Strings(t, frames, []string{"8aaefe0adcea094cfb5161a060bab4e2"})

	damaged := truncateFirstVCLAVCPayload(t, samples[1], 4)
	if out, err := dec.DecodePacketFrames(Packet{Data: damaged}); err == nil {
		t.Fatalf("damaged packet decoded frames=%d, want error", len(out))
	}

	frames, err = dec.DecodePacketFrames(Packet{Data: samples[1]})
	if err != nil {
		t.Fatalf("decode after damaged packet: %v", err)
	}
	assertFrameMD5Strings(t, frames, []string{"8aaefe0adcea094cfb5161a060bab4e2"})
}

func TestDecodeFramesAnnexBRecoversAfterDamagedSlicePacket(t *testing.T) {
	data := decodeHexFixture(t, black16IPAnnexBHex)
	config, samples := annexBToAVCConfigAndSamples(t, data, 4)
	if len(samples) != 2 {
		t.Fatalf("samples = %d, want 2", len(samples))
	}

	dec := NewDecoder()
	if frames, err := dec.DecodeFrames(config); err != nil || len(frames) != 0 {
		t.Fatalf("config frames=%d err=%v", len(frames), err)
	}

	first := avcSampleToAnnexB(t, samples[0], 4)
	second := avcSampleToAnnexB(t, samples[1], 4)
	frames, err := dec.DecodeFrames(first)
	if err != nil {
		t.Fatal(err)
	}
	assertFrameMD5Strings(t, frames, []string{"8aaefe0adcea094cfb5161a060bab4e2"})

	damaged := truncateFirstVCLAnnexBPayload(t, second)
	if out, err := dec.DecodeFrames(damaged); err == nil {
		t.Fatalf("damaged packet decoded frames=%d, want error", len(out))
	}

	frames, err = dec.DecodeFrames(second)
	if err != nil {
		t.Fatalf("decode after damaged packet: %v", err)
	}
	assertFrameMD5Strings(t, frames, []string{"8aaefe0adcea094cfb5161a060bab4e2"})
}

func truncateFirstVCLAVCPayload(t *testing.T, sample []byte, nalLengthSize int) []byte {
	t.Helper()
	nals, err := h264.SplitAVCC(sample, nalLengthSize)
	if err != nil {
		t.Fatal(err)
	}
	var out []byte
	truncated := false
	for _, nal := range nals {
		raw := nal.Raw
		if !truncated && (nal.Type == h264.NALSlice || nal.Type == h264.NALIDRSlice) {
			if len(raw) < 4 {
				t.Fatalf("short VCL NAL: %x", raw)
			}
			raw = raw[:len(raw)/2]
			truncated = true
		}
		out = appendAVCNALUnit(t, out, raw, nalLengthSize)
	}
	if !truncated {
		t.Fatal("no VCL NAL found")
	}
	return out
}

func truncateFirstVCLAnnexBPayload(t *testing.T, sample []byte) []byte {
	t.Helper()
	nals, err := h264.SplitAnnexB(sample)
	if err != nil {
		t.Fatal(err)
	}
	var out []byte
	truncated := false
	for _, nal := range nals {
		raw := nal.Raw
		if !truncated && (nal.Type == h264.NALSlice || nal.Type == h264.NALIDRSlice) {
			if len(raw) < 4 {
				t.Fatalf("short VCL NAL: %x", raw)
			}
			raw = raw[:len(raw)/2]
			truncated = true
		}
		out = appendAnnexBNAL(out, raw)
	}
	if !truncated {
		t.Fatal("no VCL NAL found")
	}
	return out
}

func avcSampleToAnnexB(t *testing.T, sample []byte, nalLengthSize int) []byte {
	t.Helper()
	nals, err := h264.SplitAVCC(sample, nalLengthSize)
	if err != nil {
		t.Fatal(err)
	}
	var out []byte
	for _, nal := range nals {
		out = appendAnnexBNAL(out, nal.Raw)
	}
	return out
}
