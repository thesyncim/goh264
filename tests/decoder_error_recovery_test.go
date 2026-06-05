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

func TestDecodePacketFramesAVCRecoversAfterDamagedNewExtradata(t *testing.T) {
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

	damagedConfig := append([]byte(nil), config...)
	damagedConfig = damagedConfig[:len(damagedConfig)-1]
	frames, err = dec.DecodePacketFrames(Packet{
		Data:     samples[1],
		SideData: []PacketSideData{{Type: PacketSideDataNewExtradata, Data: damagedConfig}},
	})
	if err != nil {
		t.Fatalf("decode with damaged avcC side data: %v", err)
	}
	assertFrameMD5Strings(t, frames, []string{"8aaefe0adcea094cfb5161a060bab4e2"})

	frames, err = dec.DecodePacketFrames(Packet{Data: samples[1]})
	if err != nil {
		t.Fatalf("decode after damaged avcC: %v", err)
	}
	assertFrameMD5Strings(t, frames, []string{"8aaefe0adcea094cfb5161a060bab4e2"})
}

func TestDecodePacketFramesAnnexBRecoversAfterDamagedNewExtradata(t *testing.T) {
	data := decodeHexFixture(t, black16IPAnnexBHex)
	extradata, _ := annexBParameterSetsAndPacket(t, data)
	_, samples := annexBToAVCConfigAndSamples(t, data, 4)
	if len(samples) != 2 {
		t.Fatalf("samples = %d, want 2", len(samples))
	}
	first := avcSampleToAnnexB(t, samples[0], 4)
	second := avcSampleToAnnexB(t, samples[1], 4)

	dec := NewDecoder()
	frames, err := dec.DecodePacketFrames(Packet{
		Data:     first,
		SideData: []PacketSideData{{Type: PacketSideDataNewExtradata, Data: extradata}},
	})
	if err != nil {
		t.Fatal(err)
	}
	assertFrameMD5Strings(t, frames, []string{"8aaefe0adcea094cfb5161a060bab4e2"})

	damagedExtradata := truncateFirstParameterSetAnnexB(t, extradata)
	frames, err = dec.DecodePacketFrames(Packet{
		Data:     second,
		SideData: []PacketSideData{{Type: PacketSideDataNewExtradata, Data: damagedExtradata}},
	})
	if err != nil {
		t.Fatalf("decode with damaged Annex B extradata side data: %v", err)
	}
	assertFrameMD5Strings(t, frames, []string{"8aaefe0adcea094cfb5161a060bab4e2"})

	frames, err = dec.DecodePacketFrames(Packet{Data: second})
	if err != nil {
		t.Fatalf("decode after damaged Annex B extradata: %v", err)
	}
	assertFrameMD5Strings(t, frames, []string{"8aaefe0adcea094cfb5161a060bab4e2"})
}

func TestDecodeFramesAnnexBRecoversAfterMalformedInBandParameterSets(t *testing.T) {
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

	packet := malformedInBandParameterSetsAnnexB()
	packet = append(packet, second...)
	frames, err = dec.DecodeFrames(packet)
	if err != nil {
		t.Fatalf("decode after malformed in-band parameter sets: %v", err)
	}
	assertFrameMD5Strings(t, frames, []string{"8aaefe0adcea094cfb5161a060bab4e2"})
}

func TestDecodeConfiguredAVCFramesRecoversAfterMalformedInBandParameterSets(t *testing.T) {
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

	sample := malformedInBandParameterSetsAVC(t, 4)
	sample = append(sample, samples[1]...)
	frames, err = dec.DecodeConfiguredAVCFrames(sample)
	if err != nil {
		t.Fatalf("decode after malformed in-band parameter sets: %v", err)
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

func TestDecodeConfiguredAVCFramesReturnsPriorFramesBeforeDamagedSlice(t *testing.T) {
	data := decodeHexFixture(t, black16IPAnnexBHex)
	config, samples := annexBToAVCConfigAndSamples(t, data, 4)
	if len(samples) != 2 {
		t.Fatalf("samples = %d, want 2", len(samples))
	}

	dec := NewDecoder()
	if _, err := dec.ParseAVCDecoderConfigurationRecord(config); err != nil {
		t.Fatal(err)
	}

	packet := append([]byte(nil), samples[0]...)
	packet = append(packet, truncateFirstVCLAVCPayload(t, samples[1], 4)...)
	frames, err := dec.DecodeConfiguredAVCFrames(packet)
	if err == nil {
		t.Fatal("combined valid+damaged packet returned nil error")
	}
	assertFrameMD5Strings(t, frames, []string{"8aaefe0adcea094cfb5161a060bab4e2"})

	frames, err = dec.DecodeConfiguredAVCFrames(samples[1])
	if err != nil {
		t.Fatalf("decode after combined valid+damaged packet: %v", err)
	}
	assertFrameMD5Strings(t, frames, []string{"8aaefe0adcea094cfb5161a060bab4e2"})
}

func TestDecodeAVCFramesWithConfigurationRecordReturnsPriorFramesBeforeDamagedSlice(t *testing.T) {
	data := decodeHexFixture(t, black16IPAnnexBHex)
	config, samples := annexBToAVCConfigAndSamples(t, data, 4)
	if len(samples) != 2 {
		t.Fatalf("samples = %d, want 2", len(samples))
	}

	packet := append([]byte(nil), samples[0]...)
	packet = append(packet, truncateFirstVCLAVCPayload(t, samples[1], 4)...)
	frames, err := NewDecoder().DecodeAVCFramesWithConfigurationRecord(config, packet)
	if err == nil {
		t.Fatal("configuration-record valid+damaged packet returned nil error")
	}
	assertFrameMD5Strings(t, frames, []string{"8aaefe0adcea094cfb5161a060bab4e2"})
}

func TestDecodePacketFramesAVCReturnsPriorFramesBeforeDamagedSlice(t *testing.T) {
	data := decodeHexFixture(t, black16IPAnnexBHex)
	config, samples := annexBToAVCConfigAndSamples(t, data, 4)
	if len(samples) != 2 {
		t.Fatalf("samples = %d, want 2", len(samples))
	}

	packet := append([]byte(nil), samples[0]...)
	packet = append(packet, truncateFirstVCLAVCPayload(t, samples[1], 4)...)
	frames, err := NewDecoder().DecodePacketFrames(Packet{
		Data:     packet,
		SideData: []PacketSideData{{Type: PacketSideDataNewExtradata, Data: config}},
	})
	if err == nil {
		t.Fatal("packet valid+damaged AVC returned nil error")
	}
	assertFrameMD5Strings(t, frames, []string{"8aaefe0adcea094cfb5161a060bab4e2"})
}

func TestDecodeFramesAnnexBReturnsPriorFramesBeforeDamagedSlice(t *testing.T) {
	data := decodeHexFixture(t, black16IPAnnexBHex)
	config, samples := annexBToAVCConfigAndSamples(t, data, 4)
	if len(samples) != 2 {
		t.Fatalf("samples = %d, want 2", len(samples))
	}

	dec := NewDecoder()
	if frames, err := dec.DecodeFrames(config); err != nil || len(frames) != 0 {
		t.Fatalf("config frames=%d err=%v", len(frames), err)
	}

	packet := avcSampleToAnnexB(t, samples[0], 4)
	packet = append(packet, truncateFirstVCLAnnexBPayload(t, avcSampleToAnnexB(t, samples[1], 4))...)
	frames, err := dec.DecodeFrames(packet)
	if err == nil {
		t.Fatal("combined valid+damaged Annex B packet returned nil error")
	}
	assertFrameMD5Strings(t, frames, []string{"8aaefe0adcea094cfb5161a060bab4e2"})

	second := avcSampleToAnnexB(t, samples[1], 4)
	frames, err = dec.DecodeFrames(second)
	if err != nil {
		t.Fatalf("decode after combined valid+damaged Annex B packet: %v", err)
	}
	assertFrameMD5Strings(t, frames, []string{"8aaefe0adcea094cfb5161a060bab4e2"})
}

func TestDecodeAnnexBFramesReturnsPriorFramesBeforeDamagedSlice(t *testing.T) {
	data := decodeHexFixture(t, black16IPAnnexBHex)
	extradata, _ := annexBParameterSetsAndPacket(t, data)
	_, samples := annexBToAVCConfigAndSamples(t, data, 4)
	if len(samples) != 2 {
		t.Fatalf("samples = %d, want 2", len(samples))
	}

	packet := append([]byte(nil), extradata...)
	packet = append(packet, avcSampleToAnnexB(t, samples[0], 4)...)
	packet = append(packet, truncateFirstVCLAnnexBPayload(t, avcSampleToAnnexB(t, samples[1], 4))...)
	frames, err := NewDecoder().DecodeAnnexBFrames(packet)
	if err == nil {
		t.Fatal("one-shot valid+damaged Annex B packet returned nil error")
	}
	assertFrameMD5Strings(t, frames, []string{"8aaefe0adcea094cfb5161a060bab4e2"})
}

func TestDecodeConfiguredAVCReturnsPriorFrameBeforeDamagedSlice(t *testing.T) {
	data := decodeHexFixture(t, black16IPAnnexBHex)
	config, samples := annexBToAVCConfigAndSamples(t, data, 4)
	if len(samples) != 2 {
		t.Fatalf("samples = %d, want 2", len(samples))
	}

	dec := NewDecoder()
	if _, err := dec.ParseAVCDecoderConfigurationRecord(config); err != nil {
		t.Fatal(err)
	}

	packet := append([]byte(nil), samples[0]...)
	packet = append(packet, truncateFirstVCLAVCPayload(t, samples[1], 4)...)
	frame, err := dec.DecodeConfiguredAVC(packet)
	assertSingleFrameWithDamagedSliceError(t, "configured AVC", frame, err)
}

func TestDecodeAVCWithConfigurationRecordReturnsPriorFrameBeforeDamagedSlice(t *testing.T) {
	data := decodeHexFixture(t, black16IPAnnexBHex)
	config, samples := annexBToAVCConfigAndSamples(t, data, 4)
	if len(samples) != 2 {
		t.Fatalf("samples = %d, want 2", len(samples))
	}

	packet := append([]byte(nil), samples[0]...)
	packet = append(packet, truncateFirstVCLAVCPayload(t, samples[1], 4)...)
	frame, err := NewDecoder().DecodeAVCWithConfigurationRecord(config, packet)
	assertSingleFrameWithDamagedSliceError(t, "configuration-record AVC", frame, err)
}

func TestDecodePacketAVCReturnsPriorFrameBeforeDamagedSlice(t *testing.T) {
	data := decodeHexFixture(t, black16IPAnnexBHex)
	config, samples := annexBToAVCConfigAndSamples(t, data, 4)
	if len(samples) != 2 {
		t.Fatalf("samples = %d, want 2", len(samples))
	}

	packet := append([]byte(nil), samples[0]...)
	packet = append(packet, truncateFirstVCLAVCPayload(t, samples[1], 4)...)
	frame, err := NewDecoder().DecodePacket(Packet{
		Data:     packet,
		SideData: []PacketSideData{{Type: PacketSideDataNewExtradata, Data: config}},
	})
	assertSingleFrameWithDamagedSliceError(t, "packet AVC", frame, err)
}

func TestDecodeAVCReturnsPriorFrameBeforeDamagedSlice(t *testing.T) {
	data := decodeHexFixture(t, black16IPAnnexBHex)
	extradata, _ := annexBParameterSetsAndPacket(t, data)
	_, samples := annexBToAVCConfigAndSamples(t, data, 4)
	if len(samples) != 2 {
		t.Fatalf("samples = %d, want 2", len(samples))
	}

	packet := annexBToAVC(t, extradata, 4)
	packet = append(packet, samples[0]...)
	packet = append(packet, truncateFirstVCLAVCPayload(t, samples[1], 4)...)
	frame, err := NewDecoder().DecodeAVC(packet, 4)
	assertSingleFrameWithDamagedSliceError(t, "one-shot AVC", frame, err)
}

func TestDecodeReturnsPriorFrameBeforeDamagedSlice(t *testing.T) {
	data := decodeHexFixture(t, black16IPAnnexBHex)
	config, samples := annexBToAVCConfigAndSamples(t, data, 4)
	if len(samples) != 2 {
		t.Fatalf("samples = %d, want 2", len(samples))
	}

	dec := NewDecoder()
	if frames, err := dec.DecodeFrames(config); err != nil || len(frames) != 0 {
		t.Fatalf("config frames=%d err=%v", len(frames), err)
	}

	packet := avcSampleToAnnexB(t, samples[0], 4)
	packet = append(packet, truncateFirstVCLAnnexBPayload(t, avcSampleToAnnexB(t, samples[1], 4))...)
	frame, err := dec.Decode(packet)
	assertSingleFrameWithDamagedSliceError(t, "auto-detected Annex B", frame, err)
}

func TestDecodeAnnexBReturnsPriorFrameBeforeDamagedSlice(t *testing.T) {
	data := decodeHexFixture(t, black16IPAnnexBHex)
	extradata, _ := annexBParameterSetsAndPacket(t, data)
	_, samples := annexBToAVCConfigAndSamples(t, data, 4)
	if len(samples) != 2 {
		t.Fatalf("samples = %d, want 2", len(samples))
	}

	packet := append([]byte(nil), extradata...)
	packet = append(packet, avcSampleToAnnexB(t, samples[0], 4)...)
	packet = append(packet, truncateFirstVCLAnnexBPayload(t, avcSampleToAnnexB(t, samples[1], 4))...)
	frame, err := NewDecoder().DecodeAnnexB(packet)
	assertSingleFrameWithDamagedSliceError(t, "one-shot Annex B", frame, err)
}

func assertSingleFrameWithDamagedSliceError(t *testing.T, surface string, frame *Frame, err error) {
	t.Helper()
	if err == nil {
		t.Fatalf("%s valid+damaged packet returned nil error", surface)
	}
	if frame == nil {
		t.Fatalf("%s valid+damaged packet returned nil frame with error %v", surface, err)
	}
	assertFrameMD5Strings(t, []*Frame{frame}, []string{"8aaefe0adcea094cfb5161a060bab4e2"})
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

func truncateFirstParameterSetAnnexB(t *testing.T, data []byte) []byte {
	t.Helper()
	nals, err := h264.SplitAnnexB(data)
	if err != nil {
		t.Fatal(err)
	}
	var out []byte
	truncated := false
	for _, nal := range nals {
		raw := nal.Raw
		if !truncated && (nal.Type == h264.NALSPS || nal.Type == h264.NALPPS) {
			if len(raw) < 2 {
				t.Fatalf("short parameter-set NAL: %x", raw)
			}
			raw = raw[:1]
			truncated = true
		}
		out = appendAnnexBNAL(out, raw)
	}
	if !truncated {
		t.Fatal("no parameter-set NAL found")
	}
	return out
}

func malformedInBandParameterSetsAnnexB() []byte {
	var out []byte
	out = appendAnnexBNAL(out, []byte{0x60 | byte(h264.NALSPS)})
	out = appendAnnexBNAL(out, []byte{0x60 | byte(h264.NALPPS)})
	return out
}

func malformedInBandParameterSetsAVC(t *testing.T, nalLengthSize int) []byte {
	t.Helper()
	var out []byte
	out = appendAVCNALUnit(t, out, []byte{0x60 | byte(h264.NALSPS)}, nalLengthSize)
	out = appendAVCNALUnit(t, out, []byte{0x60 | byte(h264.NALPPS)}, nalLengthSize)
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
