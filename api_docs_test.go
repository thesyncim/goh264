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
	packageDecoderFunctions := map[string]any{
		"InspectAVCDecoderConfigurationRecord": InspectAVCDecoderConfigurationRecord,
		"InspectAVCC":                          InspectAVCC,
	}
	for _, name := range []string{
		"DecodeFrames",
		"DecodePacketFrames",
		"DecodeAnnexBFrames",
		"DecodeAVCFrames",
		"ConfigureAVCDecoderConfigurationRecord",
		"ConfigureAVCC",
		"InspectAVCDecoderConfigurationRecord",
		"InspectAVCC",
		"DecodeConfiguredAVCFrames",
		"Decode",
		"DecodePacket",
		"DecodeAnnexB",
		"DecodeAVC",
		"DecodeConfiguredAVC",
		"DecodeAVCC",
	} {
		if _, ok := packageDecoderFunctions[name]; ok {
			if reflect.TypeOf(packageDecoderFunctions[name]).Kind() != reflect.Func {
				t.Fatalf("README decoder chooser name %s is not a package function", name)
			}
		} else if _, ok := decoderType.MethodByName(name); !ok {
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
		"OutputFormat",
		"AccessUnitData",
		"NALData",
		"RTPPacketData",
		"RTPPayloadData",
		"AVCC",
		"Limits",
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

func TestREADMEEncoderSampleChecksRuntimeControlErrors(t *testing.T) {
	data, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatalf("read README.md: %v", err)
	}
	readme := string(data)
	for _, forbidden := range []string{
		"err = enc.Set",
		"err = cfg.ValidateFrame",
		"err = enc.ValidateFrame",
		"err = enc.Reset",
		"common quality, budget",
	} {
		if strings.Contains(readme, forbidden) {
			t.Fatalf("README.md encoder sample still contains unchecked or overbroad phrase %q", forbidden)
		}
	}
	for _, required := range []string{
		"must(enc.SetBitrate",
		"must(enc.SetLimits",
		"must(cfg.ValidateFrame",
		"must(enc.ValidateFrame",
		"must(enc.Reset",
		"admitted control, budget",
	} {
		if !strings.Contains(readme, required) {
			t.Fatalf("README.md encoder sample missing checked-control phrase %q", required)
		}
	}
}

func TestREADMEEncoderSampleKeepsRTPHelpersInRTPBranch(t *testing.T) {
	data, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatalf("read README.md: %v", err)
	}
	readme := string(data)
	rtpCase := "case goh264.EncoderOutputRTP:\n\t// Use RTPPackets or the RTP helper methods below.\n\tpacket0, err := out.RTPPacketData(0)"
	if !strings.Contains(readme, rtpCase) {
		t.Fatal("README.md encoder sample should call RTPPacketData only inside the RTP output branch")
	}
	accessUnitCase := "case goh264.EncoderOutputAnnexB, goh264.EncoderOutputAVC:\n\t// Use the access-unit helpers below.\n\taccessUnit, err := out.AccessUnitData()"
	if !strings.Contains(readme, accessUnitCase) {
		t.Fatal("README.md encoder sample should call AccessUnitData inside the Annex B/AVC output branch")
	}
}

func TestREADMEEncoderAdmittedValuesTableDocumentsUnsupportedKnobs(t *testing.T) {
	data, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatalf("read README.md: %v", err)
	}
	readme := string(data)
	for _, phrase := range []string{
		"Accepted encoder setup values today",
		"EncoderProfileConstrainedBaseline",
		"EncoderProfileBaseline",
		"EncoderEntropyCAVLC",
		"Transform8x8=false",
		"MaxReferenceFrames=1",
		"BFrames=0",
		"Main/High profiles",
		"CABAC",
		"multiple refs",
		"B-frames",
		"IntraRefresh=false",
		"enabled intra refresh",
		"packetization-mode 0 with payload size >= 2",
		"packetization-mode 1 with payload size >= 3",
		"STAP-A only in mode 1",
		"DON disabled",
		"payload type 0..127",
	} {
		if !strings.Contains(readme, phrase) {
			t.Fatalf("README.md admitted-values table missing %q", phrase)
		}
	}
}

func TestREADMEDecoderAVCCStatefulSwitchGuidance(t *testing.T) {
	data, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatalf("read README.md: %v", err)
	}
	readme := strings.Join(strings.Fields(string(data)), " ")
	for _, phrase := range []string{
		"without resetting retained references",
		"IDR-bound stream switches",
		"unrelated stream where old references must not be visible",
		"call `Reset` before storing the new avcC",
	} {
		if !strings.Contains(readme, phrase) {
			t.Fatalf("README.md decoder avcC state guidance missing %q", phrase)
		}
	}
}

func TestPublicExamplesUsePreferredDecoderAVCCConfigurationName(t *testing.T) {
	data, err := os.ReadFile("examples_test.go")
	if err != nil {
		t.Fatalf("read examples_test.go: %v", err)
	}
	if strings.Contains(string(data), "dec.ParseAVCC(") {
		t.Fatal("public examples should use ConfigureAVCC for mutating decoder avcC configuration")
	}
}

func requireREADMECodeName(t *testing.T, readme string, name string) {
	t.Helper()
	if !strings.Contains(readme, "`"+name+"`") && !strings.Contains(readme, name) {
		t.Fatalf("README.md missing API chooser name %q", name)
	}
}
