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
		"Config",
		"ValidateFrame",
		"Encode",
		"EncodeInto",
		"ForceIDR",
		"HandlePLI",
		"HandleFIR",
		"PendingIDR",
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
		"DefaultRealtimeEncoderConfig",
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
		"production readiness",
	} {
		if !strings.Contains(readme, phrase) {
			t.Fatalf("README.md missing quality/parity status phrase %q", phrase)
		}
	}
	for _, forbidden := range []string{
		"pre" + "-release",
		"pre" + "-production",
		"release " + "tag",
		"release " + "readiness",
		"de" + "precate",
		"de" + "precated",
		"de" + "precation",
	} {
		if strings.Contains(readme, forbidden) {
			t.Fatalf("README.md should not use shipping-lifecycle status phrase %q", forbidden)
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
		"liveCfg := enc.Config()",
		"must(liveCfg.ValidateFrame",
		"must(enc.ValidateFrame",
		"must(enc.Reset",
		"admitted control, budget",
		"`SetDeblockMode`, `SetRTPMaxPayloadSize`, `SetPreset`",
		"Zero scalar fields in `EncoderReconfigure` mean unchanged",
		"pointer fields, grouped `Limits`, or dedicated setters",
		"`FrameRateNum`/`FrameRateDen` and `Width`/`Height` must be supplied",
		"When `Limits` is non-nil, it is applied after the individual budget",
	} {
		if !strings.Contains(readme, required) {
			t.Fatalf("README.md encoder sample missing checked-control phrase %q", required)
		}
	}
}

func TestEncoderReleaseEvidenceNamesAPISurfaceGate(t *testing.T) {
	readmeData, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatalf("read README.md: %v", err)
	}
	scriptData, err := os.ReadFile("scripts/h264-encoder-release-evidence.sh")
	if err != nil {
		t.Fatalf("read encoder release evidence script: %v", err)
	}
	readme := string(readmeData)
	script := string(scriptData)
	for _, phrase := range []string{
		"API-surface",
		"residual-boundary",
	} {
		if !strings.Contains(readme, phrase) {
			t.Fatalf("README.md encoder release evidence text missing %q", phrase)
		}
	}
	for _, phrase := range []string{
		"encoder-api-surfaces",
		"TestEncoderEncodeIntoRTPPacketsDoNotAliasAccessUnitData",
		"TestEncoderReconfigureZeroScalarFieldsAreNoOps",
		"TestEncoderZeroValueExplicitSettersRejectWithoutMutation",
		"TestEncoderNonRTPConfigsRejectInvalidRTPControls",
		"TestEncoderInvalidRTPControlsRejectForNonRTPOutputsWithoutMutation",
		"TestEncoderReconfigureOutputFormatQueuesIDRBoundary",
		"TestEncodedFrameNALDataRejectsInvalidIndexesAndMetadata",
		"TestEncodedFrameRTPDataRejectsInvalidIndexesAndMetadata",
		"encoder-writers",
		"TestCAVLCWriteResidual",
		"TestWriteCAVLCInterPBoundedMacroblock",
		"TestEncodeI420P16x16ResidualSliceRBSP",
	} {
		if !strings.Contains(script, phrase) {
			t.Fatalf("encoder release evidence script missing API-surface gate phrase %q", phrase)
		}
	}
}

func TestDecoderReleaseEvidenceNamesAPISurfaceAndRefGates(t *testing.T) {
	readmeData, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatalf("read README.md: %v", err)
	}
	scriptData, err := os.ReadFile("scripts/h264-decoder-release-evidence.sh")
	if err != nil {
		t.Fatalf("read decoder release evidence script: %v", err)
	}
	readme := string(readmeData)
	script := string(scriptData)
	for _, phrase := range []string{
		"decoder API-surface",
		"ref-modification gates",
	} {
		if !strings.Contains(readme, phrase) {
			t.Fatalf("README.md decoder release evidence text missing %q", phrase)
		}
	}
	for _, phrase := range []string{
		"decoder-api-surfaces",
		"decoder-ref-modifications",
		"TestDecodeAVCCFramesIncompatibleConfigurationDoesNotUseStalePFrameReference",
		"TestDecodePacketFramesNewExtradataIncompatibleConfigurationDoesNotUseStalePFrameReference",
		"TestDecodePacketFramesAnnexBNewExtradataIncompatibleConfigurationDoesNotUseStalePFrameReference",
		"TestParseHeadersAnnexBIncompatibleHeadersDoNotUseStalePFrameReference",
		"TestParseHeadersAVCIncompatibleHeadersDoNotUseStalePFrameReference",
		"TestDecodeAVCCFramesMultiSPSConfigurationUsesPacketActiveSPSForDPBReset",
		"TestDecodeFramesStandaloneMultiSPSConfigurationResetsForNonFirstActiveSPS",
		"TestDecodePacketFramesMultiSPSNewExtradataUsesPacketActiveSPSForDPBReset",
		"TestDecodePacketFramesAnnexBMultiSPSNewExtradataUsesPacketActiveSPSForDPBReset",
		"TestDecoderAVCConfigUsesAVCCFirstSPSForMultiSPSConfiguration",
		"TestDecoderAVCConfigUsesPacketActiveSPSForMultiSPSConfiguration",
		"TestSimpleFrameDPBRejectsMissingShortRefModificationTarget",
		"TestSimpleFrameDPBRejectsMissingLongRefModificationTarget",
	} {
		if !strings.Contains(script, phrase) {
			t.Fatalf("decoder release evidence script missing focused gate phrase %q", phrase)
		}
	}
}

func TestREADMEEncoderRTPDataSurfaceDocumentsPacketBytes(t *testing.T) {
	data, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatalf("read README.md: %v", err)
	}
	readme := string(data)
	for _, phrase := range []string{
		"For RTP output",
		"`EncodedFrame.Data` remains an Annex B access-unit view",
		"`AccessUnitData`",
		"`NALData`",
		"`RTPPackets`",
		"`RTPPacketData`",
		"`RTPPayloadData`",
	} {
		if !strings.Contains(readme, phrase) {
			t.Fatalf("README.md missing RTP data-surface phrase %q", phrase)
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
		"CBR or ConstantQP",
		"VBR until it drives quality decisions",
		"EncoderPresetRealtime",
		"Balanced/Quality presets until they drive mode decisions",
		"Workers>1` only with `Deterministic=false`",
		"no parallel throughput guarantee yet",
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

func TestREADMEEncoderDocumentsRealtimeDefaultAndLiveConfig(t *testing.T) {
	data, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatalf("read README.md: %v", err)
	}
	readme := string(data)
	for _, phrase := range []string{
		"supported realtime/WebRTC baseline",
		"`DefaultRealtimeEncoderConfig`; `DefaultEncoderConfig` is a compatibility alias",
		"Read the exact live setup after accepted setters",
		"`Encoder.Config`",
		"Encoder.Config` returns the exact normalized live configuration",
	} {
		if !strings.Contains(readme, phrase) {
			t.Fatalf("README.md realtime default/live config docs missing %q", phrase)
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
		"Compatible in-stream avcC updates retain references",
		"incompatible active SPS changes reset picture state",
		"old references are not visible to the new stream",
		"IDR-bound stream switches",
		"unrelated stream where the decoder cannot infer the boundary from avcC",
		"call `Reset` before storing the new avcC",
		"PacketSideDataNewExtradata",
		"uses the same stateful update rule",
	} {
		if !strings.Contains(readme, phrase) {
			t.Fatalf("README.md decoder avcC state guidance missing %q", phrase)
		}
	}
}

func TestREADMEStateLifecycleDocumentsDecoderEncoderBoundaries(t *testing.T) {
	data, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatalf("read README.md: %v", err)
	}
	readme := string(data)
	for _, phrase := range []string{
		"## State Lifecycle",
		"`Decoder.DecodeFrames` / `DecodePacketFrames`",
		"Retain decoder references and delayed output across stream packets",
		"`Decoder.ConfigureAVCC`",
		"resets decoder picture state for a new configured-AVC stream",
		"`Decoder.DecodeAVCCFrames` / packet `NEW_EXTRADATA`",
		"Compatible avcC or Annex B parameter-set updates retain references",
		"incompatible active SPS changes reset picture state before decoding",
		"`Decoder.Reset`",
		"Clears stored SPS/PPS, avcC length-size, references, delayed output, and parsed slice state",
		"`Encoder.Reset`",
		"preserving configuration and RTP callback",
		"`Encoder.SetQP` / `SetResolution` / `SetOutputFormat`",
		"queue an IDR boundary",
		"Invalid encoder setters or `Reconfigure` updates",
		"Leave configuration, queued IDR state, RTP sequence/callback state, frame number, timestamp, and references unchanged",
	} {
		if !strings.Contains(readme, phrase) {
			t.Fatalf("README.md lifecycle table missing %q", phrase)
		}
	}
}

func TestPublicCommentsDocumentStateLifecycleBoundaries(t *testing.T) {
	decoderData, err := os.ReadFile("decoder.go")
	if err != nil {
		t.Fatalf("read decoder.go: %v", err)
	}
	encoderData, err := os.ReadFile("encoder.go")
	if err != nil {
		t.Fatalf("read encoder.go: %v", err)
	}
	decoder := string(decoderData)
	encoder := string(encoderData)
	for _, phrase := range []string{
		"ParseHeadersAVC parses AVC parameter sets, stores SPS/PPS state and the AVC\n// NAL length size for later DecodeConfiguredAVCFrames calls",
		"Storing a configuration resets decoder picture state for a new",
		"ParseAVCC parses an avcC record, stores it for configured-AVC decode calls,\n// resets decoder picture state",
		"ConfigureAVCC parses an avcC record, stores it for configured-AVC decode\n// calls, resets decoder picture state",
		"ConfigureAVCC is the preferred short avcC API",
		"InspectAVCC is the preferred short stateless avcC name",
	} {
		if !strings.Contains(decoder, phrase) {
			t.Fatalf("decoder public comments missing lifecycle phrase %q", phrase)
		}
	}
	for _, phrase := range []string{
		"DefaultRealtimeEncoderConfig returns a realtime/WebRTC-oriented 8-bit I420",
		"DefaultEncoderConfig remains as a compatibility alias",
		"Config returns a copy of the current normalized encoder configuration",
		"Reset clears encoder coding state while preserving configuration and RTP callback",
		"After Reset, the next successfully encoded frame starts a fresh sequence",
	} {
		if !strings.Contains(encoder, phrase) {
			t.Fatalf("encoder public comments missing lifecycle phrase %q", phrase)
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

func TestREADMEDecoderAVCCPreferredNamesTable(t *testing.T) {
	data, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatalf("read README.md: %v", err)
	}
	readme := string(data)
	for _, phrase := range []string{
		"Preferred avcC names",
		"Stateless avcC metadata inspection",
		"`InspectAVCC`",
		"Store avcC for configured-AVC streaming",
		"`ConfigureAVCC`",
		"Decode with already stored avcC",
		"`DecodeConfiguredAVCFrames`",
		"Update avcC, decode one packet, then drain delayed output",
		"`DecodeAVCCFrames`",
		"Equivalent or compatibility names",
		"Single-frame helper",
		"decoder `ParseAVCC`",
		"package `ParseAVCC`",
	} {
		if !strings.Contains(readme, phrase) {
			t.Fatalf("README.md decoder avcC preferred-name table missing %q", phrase)
		}
	}
}

func requireREADMECodeName(t *testing.T, readme string, name string) {
	t.Helper()
	if !strings.Contains(readme, "`"+name+"`") && !strings.Contains(readme, name) {
		t.Fatalf("README.md missing API chooser name %q", name)
	}
}
