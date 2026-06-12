// SPDX-License-Identifier: LGPL-2.1-or-later

package goh264

import (
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestREADMECodecAPIChooserNamesPublicEntryPoints(t *testing.T) {
	data, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatalf("read README.md: %v", err)
	}
	readme := string(data)

	decoderType := reflect.TypeOf((*Decoder)(nil))
	for _, name := range []string{
		"DecodeFrames",
		"DecodePacketFrames",
		"DecodeAnnexBFrames",
		"DecodeAVCFrames",
		"ParseAVCC",
		"DecodeConfiguredAVCFrames",
		"Decode",
		"DecodePacket",
		"DecodeAnnexB",
		"DecodeAVC",
		"DecodeConfiguredAVC",
		"DecodeAVCC",
	} {
		if _, ok := decoderType.MethodByName(name); !ok {
			t.Fatalf("README decoder chooser names missing Decoder.%s", name)
		}
		requireREADMECodeName(t, readme, name)
	}

	for _, tt := range []struct {
		typeName string
		typ      reflect.Type
		method   string
	}{
		{typeName: "Packet", typ: reflect.TypeOf(Packet{}), method: "Clone"},
		{typeName: "PacketSideData", typ: reflect.TypeOf(PacketSideData{}), method: "Clone"},
		{typeName: "Frame", typ: reflect.TypeOf((*Frame)(nil)), method: "Clone"},
		{typeName: "FrameSideData", typ: reflect.TypeOf(FrameSideData{}), method: "Clone"},
	} {
		if _, ok := tt.typ.MethodByName(tt.method); !ok {
			t.Fatalf("README decoder ownership names missing %s.%s", tt.typeName, tt.method)
		}
		requireREADMECodeName(t, readme, tt.typeName)
	}

	encoderType := reflect.TypeOf((*Encoder)(nil))
	for _, name := range []string{
		"ValidateFrame",
		"Encode",
		"EncodeInto",
		"HandlePLI",
		"Reset",
		"ParameterSets",
		"RecoveryPointSEI",
		"SetBitrate",
		"SetRateControl",
		"SetVBVBufferSize",
		"SetFrameDropMode",
		"SetQP",
		"SetFrameRate",
		"SetRTPTimestampIncrement",
		"SetGOP",
		"SetResolution",
		"SetDeblockMode",
		"SetRTPMaxPayloadSize",
		"SetMaxFrameSize",
		"SetPreset",
		"SetSliceCount",
		"SetSliceMaxBytes",
		"SetMaxEncodeTimeUS",
		"SetSPSPPSMode",
		"SetSPSPPSBeforeIDR",
		"SetIntraRefresh",
		"SetRecoveryPointSEI",
		"SetRTPPacketizationMode",
		"SetRTPMetadata",
		"SetOutputFormat",
		"SetRTPPacketCallback",
		"Reconfigure",
		"I420Frame",
	} {
		if _, ok := encoderType.MethodByName(name); !ok {
			t.Fatalf("README encoder chooser names missing Encoder.%s", name)
		}
		requireREADMECodeName(t, readme, name)
	}

	encoderConfigType := reflect.TypeOf(EncoderConfig{})
	for _, name := range []string{
		"Normalize",
		"Validate",
		"ValidateFrame",
		"ParameterSets",
		"RecoveryPointSEIMessage",
		"I420Frame",
	} {
		if _, ok := encoderConfigType.MethodByName(name); !ok {
			t.Fatalf("README encoder chooser names missing EncoderConfig.%s", name)
		}
		requireREADMECodeName(t, readme, name)
	}

	for _, name := range []string{
		"DefaultEncoderConfig",
		"Clone",
		"Append",
		"AccessUnitData",
		"NALData",
		"RTPPacketData",
		"RTPPayloadData",
		"AVCC",
		"MaxFrameSizeLimit",
		"SliceMaxBytesLimit",
		"MaxEncodeTimeUSLimit",
	} {
		requireREADMECodeName(t, readme, name)
	}
}

func TestREADMEQualityStatusDoesNotTreatExamplesAsParityEvidence(t *testing.T) {
	data, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatalf("read README.md: %v", err)
	}
	readme := string(data)
	for _, phrase := range []string{
		"not production-ready as a complete codec package yet",
		"not quality-parity with a production H.264 encoder",
		"Examples",
		"API smoke tests only",
		"oracle-backed bitstream parity",
		"release readiness",
	} {
		if !strings.Contains(readme, phrase) {
			t.Fatalf("README.md missing quality/parity status phrase %q", phrase)
		}
	}
}

func requireREADMECodeName(t *testing.T, readme string, name string) {
	t.Helper()
	if !strings.Contains(readme, "`"+name+"`") && !strings.Contains(readme, name) {
		t.Fatalf("README.md missing API chooser name %q", name)
	}
}
