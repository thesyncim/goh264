// SPDX-License-Identifier: LGPL-2.1-or-later

package goh264_test

import (
	"fmt"
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

func TestDecodeConfiguredAVCFramesDoesNotAliasCallerBuffer(t *testing.T) {
	data := decodeHexFixture(t, black16IPAnnexBHex)
	config, samples := annexBToAVCConfigAndSamples(t, data, 4)
	if len(samples) != 2 {
		t.Fatalf("samples = %d, want 2", len(samples))
	}
	firstSample := append([]byte(nil), samples[0]...)

	dec := NewDecoder()
	if _, err := dec.ParseAVCDecoderConfigurationRecord(config); err != nil {
		t.Fatal(err)
	}
	frames, err := dec.DecodeConfiguredAVCFrames(firstSample)
	if err != nil {
		t.Fatalf("DecodeConfiguredAVCFrames first sample: %v", err)
	}
	assertFrameMD5Strings(t, frames, []string{"8aaefe0adcea094cfb5161a060bab4e2"})

	for i := range firstSample {
		firstSample[i] = 0xff
	}
	assertFrameMD5Strings(t, frames, []string{"8aaefe0adcea094cfb5161a060bab4e2"})

	frames, err = dec.DecodeConfiguredAVCFrames(samples[1])
	if err != nil {
		t.Fatalf("DecodeConfiguredAVCFrames after caller mutation: %v", err)
	}
	assertFrameMD5Strings(t, frames, []string{"8aaefe0adcea094cfb5161a060bab4e2"})
}

func TestDecodeConfiguredAVCDoesNotAliasCallerBuffer(t *testing.T) {
	data := decodeHexFixture(t, black16IPAnnexBHex)
	config, samples := annexBToAVCConfigAndSamples(t, data, 4)
	if len(samples) != 2 {
		t.Fatalf("samples = %d, want 2", len(samples))
	}
	firstSample := append([]byte(nil), samples[0]...)

	dec := NewDecoder()
	if _, err := dec.ParseAVCDecoderConfigurationRecord(config); err != nil {
		t.Fatal(err)
	}
	frame, err := dec.DecodeConfiguredAVC(firstSample)
	if err != nil {
		t.Fatalf("DecodeConfiguredAVC first sample: %v", err)
	}
	assertFrameMD5Strings(t, []*Frame{frame}, []string{"8aaefe0adcea094cfb5161a060bab4e2"})

	for i := range firstSample {
		firstSample[i] = 0xff
	}
	assertFrameMD5Strings(t, []*Frame{frame}, []string{"8aaefe0adcea094cfb5161a060bab4e2"})

	frames, err := dec.DecodeConfiguredAVCFrames(samples[1])
	if err != nil {
		t.Fatalf("DecodeConfiguredAVCFrames after single-frame caller mutation: %v", err)
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

func TestDecodeAVCFramesWithConfigurationRecordDoesNotAliasCallerBuffer(t *testing.T) {
	data := decodeHexFixture(t, black16IPAnnexBHex)
	config, samples := annexBToAVCConfigAndSamples(t, data, 4)
	if len(samples) != 2 {
		t.Fatalf("samples = %d, want 2", len(samples))
	}
	config = append([]byte(nil), config...)
	firstSample := append([]byte(nil), samples[0]...)

	dec := NewDecoder()
	frames, err := dec.DecodeAVCFramesWithConfigurationRecord(config, firstSample)
	if err != nil {
		t.Fatalf("DecodeAVCFramesWithConfigurationRecord first sample: %v", err)
	}
	assertFrameMD5Strings(t, frames, []string{"8aaefe0adcea094cfb5161a060bab4e2"})

	for i := range config {
		config[i] = 0xff
	}
	for i := range firstSample {
		firstSample[i] = 0xff
	}
	assertFrameMD5Strings(t, frames, []string{"8aaefe0adcea094cfb5161a060bab4e2"})

	frames, err = dec.DecodeConfiguredAVCFrames(samples[1])
	if err != nil {
		t.Fatalf("DecodeConfiguredAVCFrames after config-record caller mutation: %v", err)
	}
	assertFrameMD5Strings(t, frames, []string{"8aaefe0adcea094cfb5161a060bab4e2"})
}

func TestDecodeAVCWithConfigurationRecordDoesNotAliasCallerBuffer(t *testing.T) {
	data := decodeHexFixture(t, black16IPAnnexBHex)
	config, samples := annexBToAVCConfigAndSamples(t, data, 4)
	if len(samples) != 2 {
		t.Fatalf("samples = %d, want 2", len(samples))
	}
	config = append([]byte(nil), config...)
	firstSample := append([]byte(nil), samples[0]...)

	dec := NewDecoder()
	frame, err := dec.DecodeAVCWithConfigurationRecord(config, firstSample)
	if err != nil {
		t.Fatalf("DecodeAVCWithConfigurationRecord first sample: %v", err)
	}
	assertFrameMD5Strings(t, []*Frame{frame}, []string{"8aaefe0adcea094cfb5161a060bab4e2"})

	for i := range config {
		config[i] = 0xff
	}
	for i := range firstSample {
		firstSample[i] = 0xff
	}
	assertFrameMD5Strings(t, []*Frame{frame}, []string{"8aaefe0adcea094cfb5161a060bab4e2"})

	frames, err := dec.DecodeConfiguredAVCFrames(samples[1])
	if err != nil {
		t.Fatalf("DecodeConfiguredAVCFrames after single-frame config-record caller mutation: %v", err)
	}
	assertFrameMD5Strings(t, frames, []string{"8aaefe0adcea094cfb5161a060bab4e2"})
}

func TestDecodeAnnexBFramesDoesNotAliasCallerBuffer(t *testing.T) {
	packet := decodeHexFixture(t, black16IPAnnexBHex)
	frames, err := NewDecoder().DecodeAnnexBFrames(packet)
	if err != nil {
		t.Fatalf("DecodeAnnexBFrames: %v", err)
	}
	if len(frames) != 2 {
		t.Fatalf("frames = %d, want 2", len(frames))
	}
	for i := range packet {
		packet[i] = 0xff
	}
	assertFrameMD5Strings(t, frames, []string{
		"8aaefe0adcea094cfb5161a060bab4e2",
		"8aaefe0adcea094cfb5161a060bab4e2",
	})
}

func TestDecodeAnnexBDoesNotAliasCallerBuffer(t *testing.T) {
	packet := decodeHexFixture(t, black16AnnexBHex)
	frame, err := NewDecoder().DecodeAnnexB(packet)
	if err != nil {
		t.Fatalf("DecodeAnnexB: %v", err)
	}
	for i := range packet {
		packet[i] = 0xff
	}
	assertFrameMD5Strings(t, []*Frame{frame}, []string{"8aaefe0adcea094cfb5161a060bab4e2"})
}

func TestDecodeAVCFramesDoesNotAliasCallerBuffer(t *testing.T) {
	data := decodeHexFixture(t, black16IPAnnexBHex)
	packet := annexBToAVC(t, data, 4)
	frames, err := NewDecoder().DecodeAVCFrames(packet, 4)
	if err != nil {
		t.Fatalf("DecodeAVCFrames: %v", err)
	}
	if len(frames) != 2 {
		t.Fatalf("frames = %d, want 2", len(frames))
	}
	for i := range packet {
		packet[i] = 0xff
	}
	assertFrameMD5Strings(t, frames, []string{
		"8aaefe0adcea094cfb5161a060bab4e2",
		"8aaefe0adcea094cfb5161a060bab4e2",
	})
}

func TestDecodeAVCDoesNotAliasCallerBuffer(t *testing.T) {
	data := decodeHexFixture(t, black16AnnexBHex)
	packet := annexBToAVC(t, data, 4)
	frame, err := NewDecoder().DecodeAVC(packet, 4)
	if err != nil {
		t.Fatalf("DecodeAVC: %v", err)
	}
	for i := range packet {
		packet[i] = 0xff
	}
	assertFrameMD5Strings(t, []*Frame{frame}, []string{"8aaefe0adcea094cfb5161a060bab4e2"})
}

func TestDecodeFramesAutoAnnexBDoesNotAliasCallerBuffer(t *testing.T) {
	packet := decodeHexFixture(t, black16IPAnnexBHex)
	frames, err := NewDecoder().DecodeFrames(packet)
	if err != nil {
		t.Fatalf("DecodeFrames Annex B: %v", err)
	}
	if len(frames) != 2 {
		t.Fatalf("frames = %d, want 2", len(frames))
	}
	for i := range packet {
		packet[i] = 0xff
	}
	assertFrameMD5Strings(t, frames, []string{
		"8aaefe0adcea094cfb5161a060bab4e2",
		"8aaefe0adcea094cfb5161a060bab4e2",
	})
}

func TestDecodeFramesAutoAVCDoesNotAliasCallerBuffer(t *testing.T) {
	data := decodeHexFixture(t, black16IPAnnexBHex)
	packet := annexBToAVC(t, data, 4)
	frames, err := NewDecoder().DecodeFrames(packet)
	if err != nil {
		t.Fatalf("DecodeFrames AVC: %v", err)
	}
	if len(frames) != 2 {
		t.Fatalf("frames = %d, want 2", len(frames))
	}
	for i := range packet {
		packet[i] = 0xff
	}
	assertFrameMD5Strings(t, frames, []string{
		"8aaefe0adcea094cfb5161a060bab4e2",
		"8aaefe0adcea094cfb5161a060bab4e2",
	})
}

func TestDecodeAutoAnnexBDoesNotAliasCallerBuffer(t *testing.T) {
	packet := decodeHexFixture(t, black16AnnexBHex)
	frame, err := NewDecoder().Decode(packet)
	if err != nil {
		t.Fatalf("Decode Annex B: %v", err)
	}
	for i := range packet {
		packet[i] = 0xff
	}
	assertFrameMD5Strings(t, []*Frame{frame}, []string{"8aaefe0adcea094cfb5161a060bab4e2"})
}

func TestDecodeAutoAVCDoesNotAliasCallerBuffer(t *testing.T) {
	data := decodeHexFixture(t, black16AnnexBHex)
	packet := annexBToAVC(t, data, 4)
	frame, err := NewDecoder().Decode(packet)
	if err != nil {
		t.Fatalf("Decode AVC: %v", err)
	}
	for i := range packet {
		packet[i] = 0xff
	}
	assertFrameMD5Strings(t, []*Frame{frame}, []string{"8aaefe0adcea094cfb5161a060bab4e2"})
}

func TestDecodePacketFramesDoesNotAliasCallerBuffer(t *testing.T) {
	packet := decodeHexFixture(t, black16IPAnnexBHex)
	frames, err := NewDecoder().DecodePacketFrames(Packet{Data: packet})
	if err != nil {
		t.Fatalf("DecodePacketFrames: %v", err)
	}
	if len(frames) != 2 {
		t.Fatalf("frames = %d, want 2", len(frames))
	}
	for i := range packet {
		packet[i] = 0xff
	}
	assertFrameMD5Strings(t, frames, []string{
		"8aaefe0adcea094cfb5161a060bab4e2",
		"8aaefe0adcea094cfb5161a060bab4e2",
	})
}

func TestDecodePacketDoesNotAliasCallerBuffer(t *testing.T) {
	packet := decodeHexFixture(t, black16AnnexBHex)
	frame, err := NewDecoder().DecodePacket(Packet{Data: packet})
	if err != nil {
		t.Fatalf("DecodePacket: %v", err)
	}
	for i := range packet {
		packet[i] = 0xff
	}
	assertFrameMD5Strings(t, []*Frame{frame}, []string{"8aaefe0adcea094cfb5161a060bab4e2"})
}

func TestParseAVCDecoderConfigurationRecordRejectPreservesStoredConfiguration(t *testing.T) {
	data := decodeHexFixture(t, black16IPAnnexBHex)
	config, samples := annexBToAVCConfigAndSamples(t, data, 4)
	if len(samples) != 2 {
		t.Fatalf("samples = %d, want 2", len(samples))
	}

	dec := NewDecoder()
	if _, err := dec.ParseAVCDecoderConfigurationRecord(config); err != nil {
		t.Fatal(err)
	}

	damagedConfig := append([]byte(nil), config...)
	damagedConfig = damagedConfig[:len(damagedConfig)-1]
	if _, err := dec.ParseAVCDecoderConfigurationRecord(damagedConfig); err == nil {
		t.Fatal("damaged avcC parse returned nil error")
	}

	frames, err := dec.DecodeConfiguredAVCFrames(samples[0])
	if err != nil {
		t.Fatalf("decode after damaged avcC parse: %v", err)
	}
	assertFrameMD5Strings(t, frames, []string{"8aaefe0adcea094cfb5161a060bab4e2"})
}

func TestParseAVCDecoderConfigurationRecordDoesNotAliasCallerBuffer(t *testing.T) {
	data := decodeHexFixture(t, black16IPAnnexBHex)
	config, samples := annexBToAVCConfigAndSamples(t, data, 4)
	if len(samples) != 2 {
		t.Fatalf("samples = %d, want 2", len(samples))
	}
	config = append([]byte(nil), config...)

	dec := NewDecoder()
	if _, err := dec.ParseAVCDecoderConfigurationRecord(config); err != nil {
		t.Fatalf("ParseAVCDecoderConfigurationRecord: %v", err)
	}
	for i := range config {
		config[i] = 0xff
	}

	frames, err := dec.DecodeConfiguredAVCFrames(samples[0])
	if err != nil {
		t.Fatalf("decode after parsed config mutation: %v", err)
	}
	assertFrameMD5Strings(t, frames, []string{"8aaefe0adcea094cfb5161a060bab4e2"})
}

func TestParseHeadersAnnexBDoesNotAliasCallerBuffer(t *testing.T) {
	data := decodeHexFixture(t, black16IPAnnexBHex)
	headers, _ := annexBParameterSetsAndPacket(t, data)
	_, samples := annexBToAVCConfigAndSamples(t, data, 4)
	if len(samples) != 2 {
		t.Fatalf("samples = %d, want 2", len(samples))
	}
	headers = append([]byte(nil), headers...)

	dec := NewDecoder()
	if _, err := dec.ParseHeadersAnnexB(headers); err != nil {
		t.Fatalf("ParseHeadersAnnexB: %v", err)
	}
	for i := range headers {
		headers[i] = 0xff
	}

	frames, err := dec.DecodeFrames(avcSampleToAnnexB(t, samples[0], 4))
	if err != nil {
		t.Fatalf("DecodeFrames after Annex B header mutation: %v", err)
	}
	assertFrameMD5Strings(t, frames, []string{"8aaefe0adcea094cfb5161a060bab4e2"})
}

func TestParseHeadersAVCDoesNotAliasCallerBuffer(t *testing.T) {
	data := decodeHexFixture(t, black16IPAnnexBHex)
	headersAnnexB, _ := annexBParameterSetsAndPacket(t, data)
	for _, nalLengthSize := range []int{2, 3, 4} {
		t.Run(fmt.Sprintf("length%d", nalLengthSize), func(t *testing.T) {
			headers := annexBToAVC(t, headersAnnexB, nalLengthSize)
			_, samples := annexBToAVCConfigAndSamples(t, data, nalLengthSize)
			if len(samples) != 2 {
				t.Fatalf("samples = %d, want 2", len(samples))
			}

			dec := NewDecoder()
			if _, err := dec.ParseHeadersAVC(headers, nalLengthSize); err != nil {
				t.Fatalf("ParseHeadersAVC: %v", err)
			}
			for i := range headers {
				headers[i] = 0xff
			}

			frames, err := dec.DecodeFrames(samples[0])
			if err != nil {
				t.Fatalf("DecodeFrames after AVC header mutation: %v", err)
			}
			assertFrameMD5Strings(t, frames, []string{"8aaefe0adcea094cfb5161a060bab4e2"})
		})
	}
}

func TestParseHeadersAVCConfiguresConfiguredAVCLengthState(t *testing.T) {
	data := decodeHexFixture(t, black16IPAnnexBHex)
	headersAnnexB, _ := annexBParameterSetsAndPacket(t, data)
	for _, nalLengthSize := range []int{2, 3, 4} {
		t.Run(fmt.Sprintf("length%d", nalLengthSize), func(t *testing.T) {
			headers := annexBToAVC(t, headersAnnexB, nalLengthSize)
			_, samples := annexBToAVCConfigAndSamples(t, data, nalLengthSize)
			if len(samples) != 2 {
				t.Fatalf("samples = %d, want 2", len(samples))
			}

			dec := NewDecoder()
			if _, err := dec.ParseHeadersAVC(headers, nalLengthSize); err != nil {
				t.Fatalf("ParseHeadersAVC: %v", err)
			}
			for i := range headers {
				headers[i] = 0xff
			}

			frames, err := dec.DecodeConfiguredAVCFrames(samples[0])
			if err != nil {
				t.Fatalf("DecodeConfiguredAVCFrames after AVC header mutation: %v", err)
			}
			assertFrameMD5Strings(t, frames, []string{"8aaefe0adcea094cfb5161a060bab4e2"})
		})
	}
}

func TestParseHeadersAnnexBPreservesAVCLengthState(t *testing.T) {
	data := decodeHexFixture(t, black16IPAnnexBHex)
	headersAnnexB, _ := annexBParameterSetsAndPacket(t, data)
	headersAVC := annexBToAVC(t, headersAnnexB, 2)
	_, samples := annexBToAVCConfigAndSamples(t, data, 2)
	if len(samples) != 2 {
		t.Fatalf("samples = %d, want 2", len(samples))
	}

	dec := NewDecoder()
	if _, err := dec.ParseHeadersAVC(headersAVC, 2); err != nil {
		t.Fatalf("ParseHeadersAVC: %v", err)
	}
	if _, err := dec.ParseHeadersAnnexB(headersAnnexB); err != nil {
		t.Fatalf("ParseHeadersAnnexB: %v", err)
	}
	for i := range headersAnnexB {
		headersAnnexB[i] = 0xff
	}

	frames, err := dec.DecodeFrames(samples[0])
	if err != nil {
		t.Fatalf("DecodeFrames after Annex B header parse: %v", err)
	}
	assertFrameMD5Strings(t, frames, []string{"8aaefe0adcea094cfb5161a060bab4e2"})
}

func TestParseHeadersRejectPreservesAVCLengthState(t *testing.T) {
	data := decodeHexFixture(t, black16IPAnnexBHex)
	headersAnnexB, _ := annexBParameterSetsAndPacket(t, data)
	headersAVC := annexBToAVC(t, headersAnnexB, 2)
	_, samples := annexBToAVCConfigAndSamples(t, data, 2)
	if len(samples) != 2 {
		t.Fatalf("samples = %d, want 2", len(samples))
	}

	for _, tt := range []struct {
		name  string
		parse func(*Decoder) error
	}{
		{
			name: "annexb",
			parse: func(dec *Decoder) error {
				damagedHeaders := firstParameterSetAnnexB(t, decodeHexFixture(t, testsrc32CAVLCBFramesAnnexBHex), h264.NALSPS)
				damagedHeaders = appendAnnexBNAL(damagedHeaders, []byte{0x60 | byte(h264.NALPPS)})
				_, err := dec.ParseHeadersAnnexB(damagedHeaders)
				return err
			},
		},
		{
			name: "avc",
			parse: func(dec *Decoder) error {
				damagedHeaders := annexBToAVC(t, firstParameterSetAnnexB(t, decodeHexFixture(t, testsrc32CAVLCBFramesAnnexBHex), h264.NALSPS), 4)
				damagedHeaders = appendAVCNALUnit(t, damagedHeaders, []byte{0x60 | byte(h264.NALPPS)}, 4)
				_, err := dec.ParseHeadersAVC(damagedHeaders, 4)
				return err
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dec := NewDecoder()
			if _, err := dec.ParseHeadersAVC(headersAVC, 2); err != nil {
				t.Fatalf("ParseHeadersAVC: %v", err)
			}
			if err := tt.parse(dec); err == nil {
				t.Fatalf("damaged %s headers returned nil error", tt.name)
			}

			frames, err := dec.DecodeFrames(samples[0])
			if err != nil {
				t.Fatalf("DecodeFrames after damaged %s headers: %v", tt.name, err)
			}
			assertFrameMD5Strings(t, frames, []string{"8aaefe0adcea094cfb5161a060bab4e2"})
		})
	}
}

func TestDecodeFramesRejectAVCConfigurationRecordPreservesStoredConfiguration(t *testing.T) {
	data := decodeHexFixture(t, black16IPAnnexBHex)
	config, samples := annexBToAVCConfigAndSamples(t, data, 4)
	if len(samples) != 2 {
		t.Fatalf("samples = %d, want 2", len(samples))
	}

	dec := NewDecoder()
	if frames, err := dec.DecodeFrames(config); err != nil || len(frames) != 0 {
		t.Fatalf("config frames=%d err=%v", len(frames), err)
	}

	damagedConfig := append([]byte(nil), config...)
	damagedConfig = damagedConfig[:len(damagedConfig)-1]
	if out, err := dec.DecodeFrames(damagedConfig); err == nil {
		t.Fatalf("damaged avcC decoded frames=%d, want error", len(out))
	}

	frames, err := dec.DecodeConfiguredAVCFrames(samples[0])
	if err != nil {
		t.Fatalf("decode after damaged avcC DecodeFrames: %v", err)
	}
	assertFrameMD5Strings(t, frames, []string{"8aaefe0adcea094cfb5161a060bab4e2"})
}

func TestDecodeAVCFramesWithConfigurationRecordRejectPreservesStoredConfiguration(t *testing.T) {
	data := decodeHexFixture(t, black16IPAnnexBHex)
	config, samples := annexBToAVCConfigAndSamples(t, data, 4)
	if len(samples) != 2 {
		t.Fatalf("samples = %d, want 2", len(samples))
	}

	dec := NewDecoder()
	if _, err := dec.ParseAVCDecoderConfigurationRecord(config); err != nil {
		t.Fatal(err)
	}

	damagedConfig := append([]byte(nil), config...)
	damagedConfig = damagedConfig[:len(damagedConfig)-1]
	if out, err := dec.DecodeAVCFramesWithConfigurationRecord(damagedConfig, samples[0]); err == nil {
		t.Fatalf("damaged avcC with packet decoded frames=%d, want error", len(out))
	}

	frames, err := dec.DecodeConfiguredAVCFrames(samples[0])
	if err != nil {
		t.Fatalf("decode after damaged avcC DecodeAVCFramesWithConfigurationRecord: %v", err)
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
	for _, nalLengthSize := range []int{2, 3, 4} {
		t.Run(fmt.Sprintf("length%d", nalLengthSize), func(t *testing.T) {
			config, samples := annexBToAVCConfigAndSamples(t, data, nalLengthSize)
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
		})
	}
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

func TestDecodePacketFramesAnnexBNewExtradataRejectPreservesPreviousParameterSets(t *testing.T) {
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

	damagedExtradata := firstParameterSetAnnexB(t, decodeHexFixture(t, testsrc32CAVLCBFramesAnnexBHex), h264.NALSPS)
	damagedExtradata = appendAnnexBNAL(damagedExtradata, []byte{0x60 | byte(h264.NALPPS)})
	frames, err = dec.DecodePacketFrames(Packet{
		Data:     second,
		SideData: []PacketSideData{{Type: PacketSideDataNewExtradata, Data: damagedExtradata}},
	})
	if err != nil {
		t.Fatalf("decode with partially valid damaged Annex B extradata side data: %v", err)
	}
	assertFrameMD5Strings(t, frames, []string{"8aaefe0adcea094cfb5161a060bab4e2"})

	frames, err = dec.DecodePacketFrames(Packet{Data: second})
	if err != nil {
		t.Fatalf("decode after partially valid damaged Annex B extradata: %v", err)
	}
	assertFrameMD5Strings(t, frames, []string{"8aaefe0adcea094cfb5161a060bab4e2"})
}

func TestDecodePacketFramesAnnexBNewExtradataRejectPreservesAVCLengthState(t *testing.T) {
	data := decodeHexFixture(t, black16IPAnnexBHex)
	config, samples := annexBToAVCConfigAndSamples(t, data, 2)
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

	damagedExtradata := firstParameterSetAnnexB(t, decodeHexFixture(t, testsrc32CAVLCBFramesAnnexBHex), h264.NALSPS)
	damagedExtradata = appendAnnexBNAL(damagedExtradata, []byte{0x60 | byte(h264.NALPPS)})
	frames, err = dec.DecodePacketFrames(Packet{
		Data:     samples[1],
		SideData: []PacketSideData{{Type: PacketSideDataNewExtradata, Data: damagedExtradata}},
	})
	if err != nil {
		t.Fatalf("decode length-2 AVC with damaged Annex B extradata side data: %v", err)
	}
	assertFrameMD5Strings(t, frames, []string{"8aaefe0adcea094cfb5161a060bab4e2"})

	frames, err = dec.DecodePacketFrames(Packet{Data: samples[1]})
	if err != nil {
		t.Fatalf("decode length-2 AVC after damaged Annex B extradata: %v", err)
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

func TestParseHeadersAnnexBRejectPreservesStoredConfiguration(t *testing.T) {
	data := decodeHexFixture(t, black16IPAnnexBHex)
	config, samples := annexBToAVCConfigAndSamples(t, data, 4)
	if len(samples) != 2 {
		t.Fatalf("samples = %d, want 2", len(samples))
	}

	dec := NewDecoder()
	if _, err := dec.ParseAVCDecoderConfigurationRecord(config); err != nil {
		t.Fatalf("ParseAVCDecoderConfigurationRecord: %v", err)
	}

	damagedHeaders := firstParameterSetAnnexB(t, decodeHexFixture(t, testsrc32CAVLCBFramesAnnexBHex), h264.NALSPS)
	damagedHeaders = appendAnnexBNAL(damagedHeaders, []byte{0x60 | byte(h264.NALPPS)})
	if _, err := dec.ParseHeadersAnnexB(damagedHeaders); err == nil {
		t.Fatal("ParseHeadersAnnexB partially valid damaged headers returned nil error")
	}

	frames, err := dec.DecodeConfiguredAVCFrames(samples[0])
	if err != nil {
		t.Fatalf("DecodeConfiguredAVCFrames after damaged Annex B header parse: %v", err)
	}
	assertFrameMD5Strings(t, frames, []string{"8aaefe0adcea094cfb5161a060bab4e2"})
}

func TestParseHeadersAVCRejectPreservesStoredConfiguration(t *testing.T) {
	data := decodeHexFixture(t, black16IPAnnexBHex)
	config, samples := annexBToAVCConfigAndSamples(t, data, 4)
	if len(samples) != 2 {
		t.Fatalf("samples = %d, want 2", len(samples))
	}

	dec := NewDecoder()
	if _, err := dec.ParseAVCDecoderConfigurationRecord(config); err != nil {
		t.Fatalf("ParseAVCDecoderConfigurationRecord: %v", err)
	}

	damagedHeaders := annexBToAVC(t, firstParameterSetAnnexB(t, decodeHexFixture(t, testsrc32CAVLCBFramesAnnexBHex), h264.NALSPS), 4)
	damagedHeaders = appendAVCNALUnit(t, damagedHeaders, []byte{0x60 | byte(h264.NALPPS)}, 4)
	if _, err := dec.ParseHeadersAVC(damagedHeaders, 4); err == nil {
		t.Fatal("ParseHeadersAVC partially valid damaged headers returned nil error")
	}

	frames, err := dec.DecodeConfiguredAVCFrames(samples[0])
	if err != nil {
		t.Fatalf("DecodeConfiguredAVCFrames after damaged AVC header parse: %v", err)
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

func firstParameterSetAnnexB(t *testing.T, data []byte, typ h264.NALUnitType) []byte {
	t.Helper()
	nals, err := h264.SplitAnnexB(data)
	if err != nil {
		t.Fatal(err)
	}
	for _, nal := range nals {
		if nal.Type == typ {
			return appendAnnexBNAL(nil, nal.Raw)
		}
	}
	t.Fatalf("no Annex B parameter set type %d found", typ)
	return nil
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
