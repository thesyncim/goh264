// SPDX-License-Identifier: LGPL-2.1-or-later

package goh264_test

import (
	"bytes"
	"fmt"
	"testing"

	goh264 "github.com/thesyncim/goh264"
	"github.com/thesyncim/goh264/internal/h264"
)

func TestDecodeConfiguredAVCFramesRecoversAfterDamagedSlicePacket(t *testing.T) {
	data := decodeHexFixture(t, black16IPAnnexBHex)
	config, samples := annexBToAVCConfigAndSamples(t, data, 4)
	if len(samples) != 2 {
		t.Fatalf("samples = %d, want 2", len(samples))
	}

	dec := NewDecoder()
	if _, err := dec.ConfigureAVCC(config); err != nil {
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
	if _, err := dec.ConfigureAVCC(config); err != nil {
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
	if _, err := dec.ConfigureAVCC(config); err != nil {
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

func TestDecodeAVCCRecoversAfterDamagedSlicePacket(t *testing.T) {
	data := decodeHexFixture(t, black16IPAnnexBHex)
	config, samples := annexBToAVCConfigAndSamples(t, data, 4)
	if len(samples) != 2 {
		t.Fatalf("samples = %d, want 2", len(samples))
	}

	dec := NewDecoder()
	frames, err := dec.DecodeAVCCFrames(config, samples[0])
	if err != nil {
		t.Fatal(err)
	}
	assertFrameMD5Strings(t, frames, []string{"8aaefe0adcea094cfb5161a060bab4e2"})

	damaged := truncateFirstVCLAVCPayload(t, samples[1], 4)
	if out, err := dec.DecodeAVCCFrames(config, damaged); err == nil {
		t.Fatalf("damaged packet decoded frames=%d, want error", len(out))
	}

	frames, err = dec.DecodeAVCCFrames(config, samples[1])
	if err != nil {
		t.Fatalf("decode after damaged packet: %v", err)
	}
	assertFrameMD5Strings(t, frames, []string{"8aaefe0adcea094cfb5161a060bab4e2"})
}

func TestDecodeAVCCFramesDoesNotAliasCallerBuffer(t *testing.T) {
	data := decodeHexFixture(t, black16IPAnnexBHex)
	config, samples := annexBToAVCConfigAndSamples(t, data, 4)
	if len(samples) != 2 {
		t.Fatalf("samples = %d, want 2", len(samples))
	}
	config = append([]byte(nil), config...)
	firstSample := append([]byte(nil), samples[0]...)

	dec := NewDecoder()
	frames, err := dec.DecodeAVCCFrames(config, firstSample)
	if err != nil {
		t.Fatalf("DecodeAVCCFrames first sample: %v", err)
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

func TestDecodeAVCCDoesNotAliasCallerBuffer(t *testing.T) {
	data := decodeHexFixture(t, black16IPAnnexBHex)
	config, samples := annexBToAVCConfigAndSamples(t, data, 4)
	if len(samples) != 2 {
		t.Fatalf("samples = %d, want 2", len(samples))
	}
	config = append([]byte(nil), config...)
	firstSample := append([]byte(nil), samples[0]...)

	dec := NewDecoder()
	frame, err := dec.DecodeAVCC(config, firstSample)
	if err != nil {
		t.Fatalf("DecodeAVCC first sample: %v", err)
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

func TestPacketSideDataCloneDeepCopiesPayload(t *testing.T) {
	side := PacketSideData{Type: PacketSideDataA53ClosedCaptions, Data: []byte{1, 2, 3}}
	if err := side.Validate(); err != nil {
		t.Fatalf("PacketSideData.Validate: %v", err)
	}
	clone, err := side.Clone()
	if err != nil {
		t.Fatalf("PacketSideData.Clone: %v", err)
	}
	if clone.Type != side.Type || !bytes.Equal(clone.Data, side.Data) {
		t.Fatalf("PacketSideData.Clone = %+v, want byte-identical copy of %+v", clone, side)
	}
	if &clone.Data[0] == &side.Data[0] {
		t.Fatal("PacketSideData.Clone aliases source payload")
	}
	side.Data[0] ^= 0xff
	if bytes.Equal(clone.Data, side.Data) {
		t.Fatal("mutating source side-data changed clone")
	}
	clone.Data[1] ^= 0xff
	if clone.Data[1] == side.Data[1] {
		t.Fatal("mutating cloned side-data changed source")
	}
}

func TestPacketSideDataAppendDataReturnsCallerOwnedBytes(t *testing.T) {
	backing := []byte{0xca, 0xfe, 1, 2, 3, 0xee}
	side := PacketSideData{Type: PacketSideDataA53ClosedCaptions, Data: backing[2:5]}
	prefix := []byte{0xde, 0xad}
	got, err := side.AppendData(append([]byte(nil), prefix...))
	if err != nil {
		t.Fatalf("PacketSideData.AppendData: %v", err)
	}
	if want := append(prefix, side.Data...); !bytes.Equal(got, want) {
		t.Fatalf("PacketSideData.AppendData = %x, want %x", got, want)
	}
	got[len(prefix)] ^= 0xff
	if backing[2] != 1 || side.Data[0] != 1 {
		t.Fatalf("mutating appended side-data changed source backing/header: backing=%#x side=%#x",
			backing[2], side.Data[0])
	}

	overlapBacking := []byte{0xde, 0xad, 1, 2, 3, 0xee}
	overlapSide := PacketSideData{Type: PacketSideDataA53ClosedCaptions, Data: overlapBacking[2:5]}
	overlap, err := overlapSide.AppendData(overlapBacking[:2])
	if err != nil {
		t.Fatalf("PacketSideData.AppendData overlapping source: %v", err)
	}
	if want := []byte{0xde, 0xad, 1, 2, 3}; !bytes.Equal(overlap, want) {
		t.Fatalf("PacketSideData.AppendData overlapping source = %x, want %x", overlap, want)
	}
	overlapSide.Data[0] ^= 0xff
	if overlap[len(prefix)] != 1 {
		t.Fatalf("overlapping PacketSideData.AppendData output aliases source after mutation: %x", overlap)
	}
}

func TestPacketClonePreservesZeroValue(t *testing.T) {
	var packet Packet
	if err := packet.Validate(); err != nil {
		t.Fatalf("zero Packet.Validate: %v", err)
	}
	if clone, err := packet.Clone(); err != nil || clone.Data != nil || clone.SideData != nil {
		t.Fatalf("zero Packet.Clone = %+v/%v, want zero Packet nil error", clone, err)
	}
}

func TestPacketAppendDataReturnsCallerOwnedBytes(t *testing.T) {
	backing := []byte{0xca, 0xfe, 0x00, 0x00, 0x01, 0x65, 0xee}
	packet := Packet{Data: backing[2:6]}
	prefix := []byte{0xde, 0xad}
	got, err := packet.AppendData(append([]byte(nil), prefix...))
	if err != nil {
		t.Fatalf("Packet.AppendData: %v", err)
	}
	if want := append(prefix, packet.Data...); !bytes.Equal(got, want) {
		t.Fatalf("Packet.AppendData = %x, want %x", got, want)
	}
	got[len(prefix)] ^= 0xff
	if backing[2] != 0 || packet.Data[0] != 0 {
		t.Fatalf("mutating appended packet changed source backing/header: backing=%#x packet=%#x",
			backing[2], packet.Data[0])
	}

	overlapBacking := []byte{0xde, 0xad, 0x00, 0x00, 0x01, 0x65, 0xee}
	overlapPacket := Packet{Data: overlapBacking[2:6]}
	overlap, err := overlapPacket.AppendData(overlapBacking[:2])
	if err != nil {
		t.Fatalf("Packet.AppendData overlapping source: %v", err)
	}
	if want := []byte{0xde, 0xad, 0x00, 0x00, 0x01, 0x65}; !bytes.Equal(overlap, want) {
		t.Fatalf("Packet.AppendData overlapping source = %x, want %x", overlap, want)
	}
	overlapPacket.Data[0] ^= 0xff
	if overlap[len(prefix)] != 0 {
		t.Fatalf("overlapping Packet.AppendData output aliases source after mutation: %x", overlap)
	}
}

func TestPacketAppendSideDataReturnsCallerOwnedValues(t *testing.T) {
	captionsBacking := []byte{0xca, 0xfe, 1, 2, 3, 0xee}
	packet := Packet{SideData: []PacketSideData{
		{Type: PacketSideDataA53ClosedCaptions, Data: captionsBacking[2:5]},
		{Type: PacketSideDataActiveFormat, Data: []byte{0x0a}},
	}}
	prefix := []PacketSideData{{Type: PacketSideDataICCProfile, Data: []byte{0x01}}}
	got, err := packet.AppendSideData(append([]PacketSideData(nil), prefix...))
	if err != nil {
		t.Fatalf("Packet.AppendSideData: %v", err)
	}
	if len(got) != 3 ||
		got[0].Type != prefix[0].Type ||
		got[1].Type != PacketSideDataA53ClosedCaptions ||
		got[2].Type != PacketSideDataActiveFormat ||
		!bytes.Equal(got[1].Data, []byte{1, 2, 3}) ||
		!bytes.Equal(got[2].Data, []byte{0x0a}) {
		t.Fatalf("Packet.AppendSideData = %+v, want prefix plus packet side data", got)
	}
	if &got[1].Data[0] == &packet.SideData[0].Data[0] ||
		&got[2].Data[0] == &packet.SideData[1].Data[0] {
		t.Fatal("Packet.AppendSideData aliases source payload")
	}
	packet.SideData[0].Type = PacketSideDataLCEVC
	packet.SideData[0].Data[0] ^= 0xff
	packet.SideData[1].Data[0] ^= 0xff
	if got[1].Type != PacketSideDataA53ClosedCaptions ||
		!bytes.Equal(got[1].Data, []byte{1, 2, 3}) ||
		!bytes.Equal(got[2].Data, []byte{0x0a}) {
		t.Fatalf("mutating source packet changed appended side data: %+v", got)
	}

	overlapBacking := []PacketSideData{
		{Type: PacketSideDataICCProfile, Data: []byte{0x01}},
		{Type: PacketSideDataA53ClosedCaptions, Data: []byte{0x02, 0x03}},
	}
	overlapPacket := Packet{SideData: overlapBacking[1:]}
	overlap, err := overlapPacket.AppendSideData(overlapBacking[:1])
	if err != nil {
		t.Fatalf("Packet.AppendSideData overlapping source: %v", err)
	}
	if len(overlap) != 2 ||
		overlap[0].Type != PacketSideDataICCProfile ||
		overlap[1].Type != PacketSideDataA53ClosedCaptions ||
		!bytes.Equal(overlap[1].Data, []byte{0x02, 0x03}) {
		t.Fatalf("Packet.AppendSideData overlapping source = %+v", overlap)
	}
	overlapBacking[1].Type = PacketSideDataLCEVC
	overlapBacking[1].Data[0] ^= 0xff
	if overlap[1].Type != PacketSideDataA53ClosedCaptions ||
		!bytes.Equal(overlap[1].Data, []byte{0x02, 0x03}) {
		t.Fatalf("overlapping Packet.AppendSideData output aliases source after mutation: %+v", overlap)
	}
}

func TestPacketCloneDeepCopiesDataAndSideData(t *testing.T) {
	config, samples := annexBToAVCConfigAndSamples(t, decodeHexFixture(t, black16IPAnnexBHex), 4)
	if len(samples) != 2 {
		t.Fatalf("samples = %d, want 2", len(samples))
	}
	packet := Packet{
		Data: samples[0],
		SideData: []PacketSideData{
			{Type: PacketSideDataNewExtradata, Data: config},
			{Type: PacketSideDataA53ClosedCaptions, Data: []byte{1, 2, 3}},
		},
	}
	if err := packet.Validate(); err != nil {
		t.Fatalf("Packet.Validate: %v", err)
	}
	clone, err := packet.Clone()
	if err != nil {
		t.Fatalf("Packet.Clone: %v", err)
	}
	if !bytes.Equal(clone.Data, packet.Data) || len(clone.SideData) != len(packet.SideData) {
		t.Fatalf("Packet.Clone = %+v, want byte-identical copy of %+v", clone, packet)
	}
	if &clone.Data[0] == &packet.Data[0] ||
		&clone.SideData[0].Data[0] == &packet.SideData[0].Data[0] ||
		&clone.SideData[1].Data[0] == &packet.SideData[1].Data[0] {
		t.Fatal("Packet.Clone aliases source storage")
	}
	packet.Data[0] ^= 0xff
	packet.SideData[0].Data[0] ^= 0xff
	packet.SideData[1].Data[0] ^= 0xff
	if bytes.Equal(clone.Data, packet.Data) ||
		bytes.Equal(clone.SideData[0].Data, packet.SideData[0].Data) ||
		bytes.Equal(clone.SideData[1].Data, packet.SideData[1].Data) {
		t.Fatal("mutating source packet changed clone")
	}
	frame, err := NewDecoder().DecodePacket(clone)
	if err != nil {
		t.Fatalf("DecodePacket cloned packet: %v", err)
	}
	assertFrameMD5Strings(t, []*Frame{frame}, []string{"8aaefe0adcea094cfb5161a060bab4e2"})
	clone.Data[1] ^= 0xff
	clone.SideData[0].Data[1] ^= 0xff
	clone.SideData[1].Data[1] ^= 0xff
	if clone.Data[1] == packet.Data[1] ||
		clone.SideData[0].Data[1] == packet.SideData[0].Data[1] ||
		clone.SideData[1].Data[1] == packet.SideData[1].Data[1] {
		t.Fatal("mutating cloned packet changed source")
	}
}

func TestConfigureAVCCRejectPreservesStoredConfiguration(t *testing.T) {
	data := decodeHexFixture(t, black16IPAnnexBHex)
	config, samples := annexBToAVCConfigAndSamples(t, data, 4)
	if len(samples) != 2 {
		t.Fatalf("samples = %d, want 2", len(samples))
	}

	dec := NewDecoder()
	if _, err := dec.ConfigureAVCC(config); err != nil {
		t.Fatal(err)
	}

	damagedConfig := append([]byte(nil), config...)
	damagedConfig = damagedConfig[:len(damagedConfig)-1]
	if _, err := dec.ConfigureAVCC(damagedConfig); err == nil {
		t.Fatal("damaged avcC parse returned nil error")
	}

	frames, err := dec.DecodeConfiguredAVCFrames(samples[0])
	if err != nil {
		t.Fatalf("decode after damaged avcC parse: %v", err)
	}
	assertFrameMD5Strings(t, frames, []string{"8aaefe0adcea094cfb5161a060bab4e2"})
}

func TestConfigureAVCCDoesNotAliasCallerBuffer(t *testing.T) {
	data := decodeHexFixture(t, black16IPAnnexBHex)
	config, samples := annexBToAVCConfigAndSamples(t, data, 4)
	if len(samples) != 2 {
		t.Fatalf("samples = %d, want 2", len(samples))
	}
	config = append([]byte(nil), config...)

	dec := NewDecoder()
	if _, err := dec.ConfigureAVCC(config); err != nil {
		t.Fatalf("ConfigureAVCC: %v", err)
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

func TestDecodeRejectAVCConfigurationRecordPreservesStoredConfiguration(t *testing.T) {
	data := decodeHexFixture(t, black16IPAnnexBHex)
	config, samples := annexBToAVCConfigAndSamples(t, data, 4)
	if len(samples) != 2 {
		t.Fatalf("samples = %d, want 2", len(samples))
	}

	dec := NewDecoder()
	if frame, err := dec.Decode(config); err != ErrUnsupported || frame != nil {
		t.Fatalf("config frame=%+v err=%v, want nil/%v", frame, err, ErrUnsupported)
	}

	damagedConfig := append([]byte(nil), config...)
	damagedConfig = damagedConfig[:len(damagedConfig)-1]
	if frame, err := dec.Decode(damagedConfig); err == nil || frame != nil {
		t.Fatalf("damaged avcC Decode frame=%+v err=%v, want nil/error", frame, err)
	}

	frames, err := dec.DecodeConfiguredAVCFrames(samples[0])
	if err != nil {
		t.Fatalf("decode after damaged avcC Decode: %v", err)
	}
	assertFrameMD5Strings(t, frames, []string{"8aaefe0adcea094cfb5161a060bab4e2"})
}

func TestDecodeAVCCFramesRejectPreservesStoredConfiguration(t *testing.T) {
	data := decodeHexFixture(t, black16IPAnnexBHex)
	config, samples := annexBToAVCConfigAndSamples(t, data, 4)
	if len(samples) != 2 {
		t.Fatalf("samples = %d, want 2", len(samples))
	}

	dec := NewDecoder()
	if _, err := dec.ConfigureAVCC(config); err != nil {
		t.Fatal(err)
	}

	for _, tt := range []struct {
		name string
		call func(config, data []byte) ([]*Frame, error)
	}{
		{
			name: "DecodeAVCCFrames",
			call: dec.DecodeAVCCFrames,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			damagedConfig := append([]byte(nil), config...)
			damagedConfig = damagedConfig[:len(damagedConfig)-1]
			if out, err := tt.call(damagedConfig, samples[0]); err == nil {
				t.Fatalf("damaged avcC with packet decoded frames=%d, want error", len(out))
			}

			frames, err := dec.DecodeConfiguredAVCFrames(samples[0])
			if err != nil {
				t.Fatalf("decode after damaged avcC %s: %v", tt.name, err)
			}
			assertFrameMD5Strings(t, frames, []string{"8aaefe0adcea094cfb5161a060bab4e2"})
		})
	}
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

func TestDecodePacketAVCRecoversAfterDamagedNewExtradata(t *testing.T) {
	data := decodeHexFixture(t, black16IPAnnexBHex)
	for _, nalLengthSize := range []int{2, 3, 4} {
		t.Run(fmt.Sprintf("length%d", nalLengthSize), func(t *testing.T) {
			config, samples := annexBToAVCConfigAndSamples(t, data, nalLengthSize)
			if len(samples) != 2 {
				t.Fatalf("samples = %d, want 2", len(samples))
			}

			dec := NewDecoder()
			frame, err := dec.DecodePacket(Packet{
				Data:     samples[0],
				SideData: []PacketSideData{{Type: PacketSideDataNewExtradata, Data: config}},
			})
			if err != nil {
				t.Fatal(err)
			}
			assertFrameMD5Strings(t, []*Frame{frame}, []string{"8aaefe0adcea094cfb5161a060bab4e2"})

			damagedConfig := append([]byte(nil), config...)
			damagedConfig = damagedConfig[:len(damagedConfig)-1]
			frame, err = dec.DecodePacket(Packet{
				Data:     samples[1],
				SideData: []PacketSideData{{Type: PacketSideDataNewExtradata, Data: damagedConfig}},
			})
			if err != nil {
				t.Fatalf("decode single packet with damaged avcC side data: %v", err)
			}
			assertFrameMD5Strings(t, []*Frame{frame}, []string{"8aaefe0adcea094cfb5161a060bab4e2"})

			frames, err := dec.DecodePacketFrames(Packet{Data: samples[1]})
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

func TestDecodePacketFramesAnnexBNewExtradataClearsAVCLengthState(t *testing.T) {
	data := decodeHexFixture(t, black16IPAnnexBHex)
	extradata, _ := annexBParameterSetsAndPacket(t, data)
	config, samples := annexBToAVCConfigAndSamples(t, data, 2)
	if len(samples) != 2 {
		t.Fatalf("samples = %d, want 2", len(samples))
	}
	firstAnnexB := avcSampleToAnnexB(t, samples[0], 2)

	dec := NewDecoder()
	frames, err := dec.DecodePacketFrames(Packet{
		Data:     samples[0],
		SideData: []PacketSideData{{Type: PacketSideDataNewExtradata, Data: config}},
	})
	if err != nil {
		t.Fatal(err)
	}
	assertFrameMD5Strings(t, frames, []string{"8aaefe0adcea094cfb5161a060bab4e2"})

	frames, err = dec.DecodePacketFrames(Packet{
		Data:     firstAnnexB,
		SideData: []PacketSideData{{Type: PacketSideDataNewExtradata, Data: extradata}},
	})
	if err != nil {
		t.Fatalf("decode Annex B after Annex B extradata side data: %v", err)
	}
	assertFrameMD5Strings(t, frames, []string{"8aaefe0adcea094cfb5161a060bab4e2"})

	if out, err := dec.DecodeConfiguredAVCFrames(samples[1]); err != ErrInvalidData {
		t.Fatalf("DecodeConfiguredAVCFrames after Annex B extradata frames=%d err=%v, want ErrInvalidData", len(out), err)
	}
}

func TestDecodePacketAnnexBNewExtradataClearsAVCLengthState(t *testing.T) {
	data := decodeHexFixture(t, black16IPAnnexBHex)
	extradata, _ := annexBParameterSetsAndPacket(t, data)
	config, samples := annexBToAVCConfigAndSamples(t, data, 2)
	if len(samples) != 2 {
		t.Fatalf("samples = %d, want 2", len(samples))
	}
	firstAnnexB := avcSampleToAnnexB(t, samples[0], 2)

	dec := NewDecoder()
	frame, err := dec.DecodePacket(Packet{
		Data:     samples[0],
		SideData: []PacketSideData{{Type: PacketSideDataNewExtradata, Data: config}},
	})
	if err != nil {
		t.Fatal(err)
	}
	assertFrameMD5Strings(t, []*Frame{frame}, []string{"8aaefe0adcea094cfb5161a060bab4e2"})

	frame, err = dec.DecodePacket(Packet{
		Data:     firstAnnexB,
		SideData: []PacketSideData{{Type: PacketSideDataNewExtradata, Data: extradata}},
	})
	if err != nil {
		t.Fatalf("decode single Annex B after Annex B extradata side data: %v", err)
	}
	assertFrameMD5Strings(t, []*Frame{frame}, []string{"8aaefe0adcea094cfb5161a060bab4e2"})

	if out, err := dec.DecodeConfiguredAVCFrames(samples[1]); err != ErrInvalidData {
		t.Fatalf("DecodeConfiguredAVCFrames after single Annex B extradata frames=%d err=%v, want ErrInvalidData", len(out), err)
	}
}

func TestDecodePacketAnnexBNewExtradataRejectPreservesAVCLengthState(t *testing.T) {
	data := decodeHexFixture(t, black16IPAnnexBHex)
	config, samples := annexBToAVCConfigAndSamples(t, data, 2)
	if len(samples) != 2 {
		t.Fatalf("samples = %d, want 2", len(samples))
	}

	dec := NewDecoder()
	frame, err := dec.DecodePacket(Packet{
		Data:     samples[0],
		SideData: []PacketSideData{{Type: PacketSideDataNewExtradata, Data: config}},
	})
	if err != nil {
		t.Fatal(err)
	}
	assertFrameMD5Strings(t, []*Frame{frame}, []string{"8aaefe0adcea094cfb5161a060bab4e2"})

	damagedExtradata := firstParameterSetAnnexB(t, decodeHexFixture(t, testsrc32CAVLCBFramesAnnexBHex), h264.NALSPS)
	damagedExtradata = appendAnnexBNAL(damagedExtradata, []byte{0x60 | byte(h264.NALPPS)})
	frame, err = dec.DecodePacket(Packet{
		Data:     samples[1],
		SideData: []PacketSideData{{Type: PacketSideDataNewExtradata, Data: damagedExtradata}},
	})
	if err != nil {
		t.Fatalf("decode single length-2 AVC with damaged Annex B extradata side data: %v", err)
	}
	assertFrameMD5Strings(t, []*Frame{frame}, []string{"8aaefe0adcea094cfb5161a060bab4e2"})

	frames, err := dec.DecodePacketFrames(Packet{Data: samples[1]})
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

func TestDecodeFramesAnnexBRejectsPartialInBandParameterSetsTransactionally(t *testing.T) {
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
		t.Fatalf("DecodeFrames first: %v", err)
	}
	assertFrameMD5Strings(t, frames, []string{"8aaefe0adcea094cfb5161a060bab4e2"})
	assertDecoderAVCConfigGeometry(t, dec, 4, 16, 16)

	packet := partiallyValidForeignSPSMalformedPPSAnnexB(t)
	packet = append(packet, second...)
	frames, err = dec.DecodeFrames(packet)
	if err != nil {
		t.Fatalf("decode after partially valid malformed in-band parameter sets: %v", err)
	}
	assertFrameMD5Strings(t, frames, []string{"8aaefe0adcea094cfb5161a060bab4e2"})
	assertDecoderAVCConfigGeometry(t, dec, 4, 16, 16)
}

func TestDecodeFramesAnnexBRejectsPartialInBandHeaderOnlyWithoutPoisoningConfig(t *testing.T) {
	data := decodeHexFixture(t, black16IPAnnexBHex)
	config, samples := annexBToAVCConfigAndSamples(t, data, 4)
	if len(samples) != 2 {
		t.Fatalf("samples = %d, want 2", len(samples))
	}

	dec := NewDecoder()
	if frames, err := dec.DecodeFrames(config); err != nil || len(frames) != 0 {
		t.Fatalf("config frames=%d err=%v", len(frames), err)
	}
	frames, err := dec.DecodeFrames(avcSampleToAnnexB(t, samples[0], 4))
	if err != nil {
		t.Fatalf("DecodeFrames first: %v", err)
	}
	assertFrameMD5Strings(t, frames, []string{"8aaefe0adcea094cfb5161a060bab4e2"})
	assertDecoderAVCConfigGeometry(t, dec, 4, 16, 16)

	out, err := dec.DecodeFrames(partiallyValidForeignSPSMalformedPPSAnnexB(t))
	if err != nil {
		t.Fatalf("partial in-band header-only packet: %v", err)
	}
	if len(out) != 0 {
		t.Fatalf("partial in-band header-only packet returned %d frames, want 0", len(out))
	}
	assertDecoderAVCConfigGeometry(t, dec, 4, 16, 16)

	frames, err = dec.DecodeFrames(avcSampleToAnnexB(t, samples[1], 4))
	if err != nil {
		t.Fatalf("DecodeFrames after partial in-band header-only packet: %v", err)
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
	if _, err := dec.ConfigureAVCC(config); err != nil {
		t.Fatalf("ConfigureAVCC: %v", err)
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
	if _, err := dec.ConfigureAVCC(config); err != nil {
		t.Fatalf("ConfigureAVCC: %v", err)
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
	if _, err := dec.ConfigureAVCC(config); err != nil {
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

func TestDecodeConfiguredAVCFramesRejectsPartialInBandParameterSetsTransactionally(t *testing.T) {
	data := decodeHexFixture(t, black16IPAnnexBHex)
	config, samples := annexBToAVCConfigAndSamples(t, data, 4)
	if len(samples) != 2 {
		t.Fatalf("samples = %d, want 2", len(samples))
	}

	dec := NewDecoder()
	if _, err := dec.ConfigureAVCC(config); err != nil {
		t.Fatalf("ConfigureAVCC: %v", err)
	}
	frames, err := dec.DecodeConfiguredAVCFrames(samples[0])
	if err != nil {
		t.Fatalf("DecodeConfiguredAVCFrames first: %v", err)
	}
	assertFrameMD5Strings(t, frames, []string{"8aaefe0adcea094cfb5161a060bab4e2"})
	assertDecoderAVCConfigGeometry(t, dec, 4, 16, 16)

	sample := partiallyValidForeignSPSMalformedPPSAVC(t, 4)
	sample = append(sample, samples[1]...)
	frames, err = dec.DecodeConfiguredAVCFrames(sample)
	if err != nil {
		t.Fatalf("decode after partially valid malformed in-band parameter sets: %v", err)
	}
	assertFrameMD5Strings(t, frames, []string{"8aaefe0adcea094cfb5161a060bab4e2"})
	assertDecoderAVCConfigGeometry(t, dec, 4, 16, 16)
}

func TestDecodeConfiguredAVCFramesRejectsPartialInBandHeaderOnlyWithoutPoisoningConfig(t *testing.T) {
	data := decodeHexFixture(t, black16IPAnnexBHex)
	config, samples := annexBToAVCConfigAndSamples(t, data, 4)
	if len(samples) != 2 {
		t.Fatalf("samples = %d, want 2", len(samples))
	}

	dec := NewDecoder()
	if _, err := dec.ConfigureAVCC(config); err != nil {
		t.Fatalf("ConfigureAVCC: %v", err)
	}
	frames, err := dec.DecodeConfiguredAVCFrames(samples[0])
	if err != nil {
		t.Fatalf("DecodeConfiguredAVCFrames first: %v", err)
	}
	assertFrameMD5Strings(t, frames, []string{"8aaefe0adcea094cfb5161a060bab4e2"})
	assertDecoderAVCConfigGeometry(t, dec, 4, 16, 16)

	out, err := dec.DecodeConfiguredAVCFrames(partiallyValidForeignSPSMalformedPPSAVC(t, 4))
	if err != nil {
		t.Fatalf("partial in-band header-only packet: %v", err)
	}
	if len(out) != 0 {
		t.Fatalf("partial in-band header-only packet returned %d frames, want 0", len(out))
	}
	assertDecoderAVCConfigGeometry(t, dec, 4, 16, 16)

	frames, err = dec.DecodeConfiguredAVCFrames(samples[1])
	if err != nil {
		t.Fatalf("DecodeConfiguredAVCFrames after partial in-band header-only packet: %v", err)
	}
	assertFrameMD5Strings(t, frames, []string{"8aaefe0adcea094cfb5161a060bab4e2"})
}

func TestDecodeFramesValidInBandParameterSetsBeforeDamagedSliceUpdateConfigAndRecover(t *testing.T) {
	oldData := decodeHexFixture(t, black16IPAnnexBHex)
	oldConfig, oldSamples := annexBToAVCConfigAndSamples(t, oldData, 4)
	if len(oldSamples) != 2 {
		t.Fatalf("old samples = %d, want 2", len(oldSamples))
	}
	newData := decodeHexFixture(t, testsrc32CAVLCBFramesAnnexBHex)
	newParamSets, _ := annexBParameterSetsAndPacket(t, newData)
	_, newSamples := annexBToAVCConfigAndSamples(t, newData, 4)
	if len(newSamples) != 3 {
		t.Fatalf("new samples = %d, want 3", len(newSamples))
	}

	dec := NewDecoder()
	if frames, err := dec.DecodeFrames(oldConfig); err != nil || len(frames) != 0 {
		t.Fatalf("old config frames=%d err=%v", len(frames), err)
	}
	frames, err := dec.DecodeFrames(avcSampleToAnnexB(t, oldSamples[0], 4))
	if err != nil {
		t.Fatalf("DecodeFrames old sample: %v", err)
	}
	assertFrameMD5Strings(t, frames, []string{"8aaefe0adcea094cfb5161a060bab4e2"})
	assertDecoderAVCConfigGeometry(t, dec, 4, 16, 16)

	damaged := append([]byte(nil), newParamSets...)
	damaged = append(damaged, truncateFirstVCLAnnexBPayload(t, avcSampleToAnnexB(t, newSamples[0], 4))...)
	if out, err := dec.DecodeFrames(damaged); err == nil || len(out) != 0 {
		t.Fatalf("valid in-band parameter sets plus damaged slice frames=%d err=%v, want no frames with error", len(out), err)
	}
	assertDecoderAVCConfigGeometry(t, dec, 4, 32, 32)

	var recovered []*Frame
	for i, sample := range newSamples {
		frames, err := dec.DecodeFrames(avcSampleToAnnexB(t, sample, 4))
		if err != nil {
			t.Fatalf("DecodeFrames recovered new sample[%d]: %v", i, err)
		}
		recovered = append(recovered, frames...)
	}
	flushed, err := dec.FlushDelayedFrames()
	if err != nil {
		t.Fatalf("FlushDelayedFrames after recovered new stream: %v", err)
	}
	recovered = append(recovered, flushed...)
	assertFrameMD5Strings(t, recovered, []string{
		"2a9d9acd3e52356ad072de93fdbaca3d",
		"96107676801850afd8aed8546397e3bf",
		"3967b8bfe3a3a8cde4bc22334008eb1f",
	})
}

func TestValidAVCCBeforeDamagedSliceUpdatesConfigAndRecover(t *testing.T) {
	oldData := decodeHexFixture(t, black16IPAnnexBHex)
	oldConfig, oldSamples := annexBToAVCConfigAndSamples(t, oldData, 4)
	if len(oldSamples) != 2 {
		t.Fatalf("old samples = %d, want 2", len(oldSamples))
	}
	newData := decodeHexFixture(t, testsrc32CAVLCBFramesAnnexBHex)
	newConfig, newSamples := annexBToAVCConfigAndSamples(t, newData, 4)
	if len(newSamples) != 3 {
		t.Fatalf("new samples = %d, want 3", len(newSamples))
	}

	for _, tt := range []struct {
		name        string
		damage      func(*Decoder, []byte) ([]*Frame, error)
		decodeValid func(*Decoder, []byte) ([]*Frame, error)
	}{
		{
			name: "DecodeAVCCFrames",
			damage: func(dec *Decoder, damaged []byte) ([]*Frame, error) {
				return dec.DecodeAVCCFrames(newConfig, damaged)
			},
			decodeValid: func(dec *Decoder, sample []byte) ([]*Frame, error) {
				return dec.DecodeConfiguredAVCFrames(sample)
			},
		},
		{
			name: "DecodePacketFrames NEW_EXTRADATA",
			damage: func(dec *Decoder, damaged []byte) ([]*Frame, error) {
				return dec.DecodePacketFrames(Packet{
					Data:     damaged,
					SideData: []PacketSideData{{Type: PacketSideDataNewExtradata, Data: newConfig}},
				})
			},
			decodeValid: func(dec *Decoder, sample []byte) ([]*Frame, error) {
				return dec.DecodePacketFrames(Packet{Data: sample})
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dec := NewDecoder()
			if _, err := dec.ConfigureAVCC(oldConfig); err != nil {
				t.Fatalf("ConfigureAVCC old config: %v", err)
			}
			frames, err := dec.DecodeConfiguredAVCFrames(oldSamples[0])
			if err != nil {
				t.Fatalf("DecodeConfiguredAVCFrames old sample: %v", err)
			}
			assertFrameMD5Strings(t, frames, []string{"8aaefe0adcea094cfb5161a060bab4e2"})
			assertDecoderAVCConfigGeometry(t, dec, 4, 16, 16)

			damaged := truncateFirstVCLAVCPayload(t, newSamples[0], 4)
			if out, err := tt.damage(dec, damaged); err == nil || len(out) != 0 {
				t.Fatalf("%s valid config plus damaged slice frames=%d err=%v, want no frames with error",
					tt.name, len(out), err)
			}
			assertDecoderAVCConfigGeometry(t, dec, 4, 32, 32)

			var recovered []*Frame
			for i, sample := range newSamples {
				frames, err := tt.decodeValid(dec, sample)
				if err != nil {
					t.Fatalf("%s recovered new sample[%d]: %v", tt.name, i, err)
				}
				recovered = append(recovered, frames...)
			}
			flushed, err := dec.FlushDelayedFrames()
			if err != nil {
				t.Fatalf("%s FlushDelayedFrames after recovered new stream: %v", tt.name, err)
			}
			recovered = append(recovered, flushed...)
			assertFrameMD5Strings(t, recovered, []string{
				"2a9d9acd3e52356ad072de93fdbaca3d",
				"96107676801850afd8aed8546397e3bf",
				"3967b8bfe3a3a8cde4bc22334008eb1f",
			})
		})
	}
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

func TestDecodePublicSurfacesRecoverAfterDamagedLaterMultiSlice(t *testing.T) {
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
	first, err := enc.Encode(firstFrame)
	if err != nil {
		t.Fatalf("Encode multi-slice IDR: %v", err)
	}
	secondFrame := firstFrame
	secondFrame.PTS = int64(cfg.RTPTimestampIncrement)
	second, err := enc.Encode(secondFrame)
	if err != nil {
		t.Fatalf("Encode multi-slice P-skip: %v", err)
	}
	if len(first.NALUnits) != 5 || len(second.NALUnits) != 3 {
		t.Fatalf("encoded NAL counts first/second = %d/%d, want 5/3", len(first.NALUnits), len(second.NALUnits))
	}

	wantFirst := appendI420FrameBytes(nil, firstFrame)
	wantSecond := appendI420FrameBytes(nil, secondFrame)
	config, avcSamples := annexBToAVCConfigAndSamples(t, append(append([]byte(nil), first.Data...), second.Data...), 4)
	if len(avcSamples) != 2 {
		t.Fatalf("AVC samples = %d, want 2", len(avcSamples))
	}
	firstAnnexB := append([]byte(nil), first.Data...)
	secondAnnexB := append([]byte(nil), second.Data...)
	damagedSecondAnnexB := truncateVCLAnnexBPayloadAt(t, secondAnnexB, 1)
	damagedSecondAVC := truncateVCLAVCPayloadAt(t, avcSamples[1], 4, 1)

	t.Run("annexb", func(t *testing.T) {
		dec := goh264.NewDecoder()
		frames, err := dec.DecodeFrames(firstAnnexB)
		if err != nil {
			t.Fatalf("DecodeFrames first: %v", err)
		}
		assertDecodedEncoderFrameBytes(t, frames, wantFirst)
		if out, err := dec.DecodeFrames(damagedSecondAnnexB); err == nil || len(out) != 0 {
			t.Fatalf("DecodeFrames damaged later slice frames=%d err=%v, want no frames with error", len(out), err)
		}
		frames, err = dec.DecodeFrames(secondAnnexB)
		if err != nil {
			t.Fatalf("DecodeFrames recovered second: %v", err)
		}
		assertDecodedEncoderFrameBytes(t, frames, wantSecond)
	})

	t.Run("configured-avc", func(t *testing.T) {
		dec := goh264.NewDecoder()
		if _, err := dec.ConfigureAVCC(config); err != nil {
			t.Fatalf("ConfigureAVCC: %v", err)
		}
		frames, err := dec.DecodeConfiguredAVCFrames(avcSamples[0])
		if err != nil {
			t.Fatalf("DecodeConfiguredAVCFrames first: %v", err)
		}
		assertDecodedEncoderFrameBytes(t, frames, wantFirst)
		if out, err := dec.DecodeConfiguredAVCFrames(damagedSecondAVC); err == nil || len(out) != 0 {
			t.Fatalf("DecodeConfiguredAVCFrames damaged later slice frames=%d err=%v, want no frames with error", len(out), err)
		}
		frames, err = dec.DecodeConfiguredAVCFrames(avcSamples[1])
		if err != nil {
			t.Fatalf("DecodeConfiguredAVCFrames recovered second: %v", err)
		}
		assertDecodedEncoderFrameBytes(t, frames, wantSecond)
	})

	t.Run("packet-new-extradata", func(t *testing.T) {
		dec := goh264.NewDecoder()
		frames, err := dec.DecodePacketFrames(goh264.Packet{
			Data:     avcSamples[0],
			SideData: []goh264.PacketSideData{{Type: goh264.PacketSideDataNewExtradata, Data: config}},
		})
		if err != nil {
			t.Fatalf("DecodePacketFrames first: %v", err)
		}
		assertDecodedEncoderFrameBytes(t, frames, wantFirst)
		if out, err := dec.DecodePacketFrames(goh264.Packet{Data: damagedSecondAVC}); err == nil || len(out) != 0 {
			t.Fatalf("DecodePacketFrames damaged later slice frames=%d err=%v, want no frames with error", len(out), err)
		}
		frames, err = dec.DecodePacketFrames(goh264.Packet{Data: avcSamples[1]})
		if err != nil {
			t.Fatalf("DecodePacketFrames recovered second: %v", err)
		}
		assertDecodedEncoderFrameBytes(t, frames, wantSecond)
	})
}

func TestDecodeConfiguredAVCFramesReturnsPriorFramesBeforeDamagedSlice(t *testing.T) {
	data := decodeHexFixture(t, black16IPAnnexBHex)
	config, samples := annexBToAVCConfigAndSamples(t, data, 4)
	if len(samples) != 2 {
		t.Fatalf("samples = %d, want 2", len(samples))
	}

	dec := NewDecoder()
	if _, err := dec.ConfigureAVCC(config); err != nil {
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

func TestDecodeAVCCFramesReturnsPriorFramesBeforeDamagedSlice(t *testing.T) {
	data := decodeHexFixture(t, black16IPAnnexBHex)
	config, samples := annexBToAVCConfigAndSamples(t, data, 4)
	if len(samples) != 2 {
		t.Fatalf("samples = %d, want 2", len(samples))
	}

	packet := append([]byte(nil), samples[0]...)
	packet = append(packet, truncateFirstVCLAVCPayload(t, samples[1], 4)...)
	frames, err := NewDecoder().DecodeAVCCFrames(config, packet)
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
	if _, err := dec.ConfigureAVCC(config); err != nil {
		t.Fatal(err)
	}

	packet := append([]byte(nil), samples[0]...)
	packet = append(packet, truncateFirstVCLAVCPayload(t, samples[1], 4)...)
	frame, err := dec.DecodeConfiguredAVC(packet)
	assertSingleFrameWithDamagedSliceError(t, "configured AVC", frame, err)
}

func TestDecodeAVCCReturnsPriorFrameBeforeDamagedSlice(t *testing.T) {
	data := decodeHexFixture(t, black16IPAnnexBHex)
	config, samples := annexBToAVCConfigAndSamples(t, data, 4)
	if len(samples) != 2 {
		t.Fatalf("samples = %d, want 2", len(samples))
	}

	packet := append([]byte(nil), samples[0]...)
	packet = append(packet, truncateFirstVCLAVCPayload(t, samples[1], 4)...)
	frame, err := NewDecoder().DecodeAVCC(config, packet)
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

func assertDecoderAVCConfigGeometry(t *testing.T, dec *Decoder, nalLengthSize int, width int, height int) {
	t.Helper()
	cfg, err := dec.AVCConfig()
	if err != nil {
		t.Fatalf("AVCConfig: %v", err)
	}
	if cfg.NALLengthSize != nalLengthSize || cfg.StreamInfo.Width != width || cfg.StreamInfo.Height != height {
		t.Fatalf("AVCConfig = length %d %dx%d, want length %d %dx%d",
			cfg.NALLengthSize, cfg.StreamInfo.Width, cfg.StreamInfo.Height, nalLengthSize, width, height)
	}
}

func truncateFirstVCLAVCPayload(t *testing.T, sample []byte, nalLengthSize int) []byte {
	return truncateVCLAVCPayloadAt(t, sample, nalLengthSize, 0)
}

func truncateVCLAVCPayloadAt(t *testing.T, sample []byte, nalLengthSize int, vclIndex int) []byte {
	t.Helper()
	nals, err := h264.SplitAVCC(sample, nalLengthSize)
	if err != nil {
		t.Fatal(err)
	}
	var out []byte
	seenVCL := 0
	truncated := false
	for _, nal := range nals {
		raw := nal.Raw
		if nal.Type == h264.NALSlice || nal.Type == h264.NALIDRSlice {
			if seenVCL == vclIndex {
				if len(raw) < 4 {
					t.Fatalf("short VCL NAL: %x", raw)
				}
				raw = raw[:len(raw)/2]
				truncated = true
			}
			seenVCL++
		}
		out = appendAVCNALUnit(t, out, raw, nalLengthSize)
	}
	if !truncated {
		t.Fatalf("VCL NAL index %d not found", vclIndex)
	}
	return out
}

func truncateFirstVCLAnnexBPayload(t *testing.T, sample []byte) []byte {
	return truncateVCLAnnexBPayloadAt(t, sample, 0)
}

func truncateVCLAnnexBPayloadAt(t *testing.T, sample []byte, vclIndex int) []byte {
	t.Helper()
	nals, err := h264.SplitAnnexB(sample)
	if err != nil {
		t.Fatal(err)
	}
	var out []byte
	seenVCL := 0
	truncated := false
	for _, nal := range nals {
		raw := nal.Raw
		if nal.Type == h264.NALSlice || nal.Type == h264.NALIDRSlice {
			if seenVCL == vclIndex {
				if len(raw) < 4 {
					t.Fatalf("short VCL NAL: %x", raw)
				}
				raw = raw[:len(raw)/2]
				truncated = true
			}
			seenVCL++
		}
		out = appendAnnexBNAL(out, raw)
	}
	if !truncated {
		t.Fatalf("VCL NAL index %d not found", vclIndex)
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

func partiallyValidForeignSPSMalformedPPSAnnexB(t *testing.T) []byte {
	t.Helper()
	out := firstParameterSetAnnexB(t, decodeHexFixture(t, testsrc32CAVLCBFramesAnnexBHex), h264.NALSPS)
	out = appendAnnexBNAL(out, []byte{0x60 | byte(h264.NALPPS)})
	return out
}

func partiallyValidForeignSPSMalformedPPSAVC(t *testing.T, nalLengthSize int) []byte {
	t.Helper()
	out := annexBToAVC(t, firstParameterSetAnnexB(t, decodeHexFixture(t, testsrc32CAVLCBFramesAnnexBHex), h264.NALSPS), nalLengthSize)
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
