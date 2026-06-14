// SPDX-License-Identifier: LGPL-2.1-or-later

package goh264

import (
	"os"
	"path/filepath"
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
		"InspectAnnexBHeaders": InspectAnnexBHeaders,
		"InspectAVCHeaders":    InspectAVCHeaders,
		"InspectAVCC":          InspectAVCC,
	}
	for _, name := range []string{
		"DecodeFrames",
		"DecodePacketFrames",
		"DecodeAnnexBFrames",
		"DecodeAVCFrames",
		"InspectAnnexBHeaders",
		"InspectAVCHeaders",
		"ConfigureAVCC",
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
		{typeName: "Packet", typ: reflect.TypeOf(Packet{}), method: "Validate"},
		{typeName: "Packet", typ: reflect.TypeOf(Packet{}), method: "AppendData"},
		{typeName: "Packet", typ: reflect.TypeOf(Packet{}), method: "AppendSideData"},
		{typeName: "PacketSideData", typ: reflect.TypeOf(PacketSideData{}), method: "Clone"},
		{typeName: "PacketSideData", typ: reflect.TypeOf(PacketSideData{}), method: "Validate"},
		{typeName: "PacketSideData", typ: reflect.TypeOf(PacketSideData{}), method: "AppendData"},
		{typeName: "Frame", typ: reflect.TypeOf((*Frame)(nil)), method: "Validate"},
		{typeName: "Frame", typ: reflect.TypeOf((*Frame)(nil)), method: "Clone"},
		{typeName: "FrameSideData", typ: reflect.TypeOf(FrameSideData{}), method: "Clone"},
		{typeName: "FrameSideData", typ: reflect.TypeOf(FrameSideData{}), method: "Validate"},
		{typeName: "FrameSideData", typ: reflect.TypeOf(FrameSideData{}), method: "AppendUserDataUnregistered"},
		{typeName: "FrameSideData", typ: reflect.TypeOf(FrameSideData{}), method: "AppendA53ClosedCaptions"},
		{typeName: "FrameSideData", typ: reflect.TypeOf(FrameSideData{}), method: "AppendICCProfile"},
		{typeName: "FrameSideData", typ: reflect.TypeOf(FrameSideData{}), method: "AppendDynamicHDR10Plus"},
		{typeName: "FrameSideData", typ: reflect.TypeOf(FrameSideData{}), method: "AppendLCEVC"},
		{typeName: "FrameSideData", typ: reflect.TypeOf(FrameSideData{}), method: "AppendS12MTimecodes"},
		{typeName: "PictureTiming", typ: reflect.TypeOf((*PictureTiming)(nil)), method: "Validate"},
		{typeName: "PictureTiming", typ: reflect.TypeOf((*PictureTiming)(nil)), method: "Clone"},
		{typeName: "PictureTiming", typ: reflect.TypeOf((*PictureTiming)(nil)), method: "AppendTimecodes"},
		{typeName: "ReferenceDisplaysInfo", typ: reflect.TypeOf((*ReferenceDisplaysInfo)(nil)), method: "Validate"},
		{typeName: "ReferenceDisplaysInfo", typ: reflect.TypeOf((*ReferenceDisplaysInfo)(nil)), method: "Clone"},
		{typeName: "ReferenceDisplaysInfo", typ: reflect.TypeOf((*ReferenceDisplaysInfo)(nil)), method: "AppendDisplays"},
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
		"DefaultRTPEncoderConfig",
		"DefaultAnnexBEncoderConfig",
		"DefaultAVCEncoderConfig",
		"DefaultRealtimeEncoderConfig",
		"DefaultEncoderConfig",
		"AppendData",
		"AppendUserDataUnregistered",
		"AppendA53ClosedCaptions",
		"AppendICCProfile",
		"AppendDynamicHDR10Plus",
		"AppendLCEVC",
		"AppendS12MTimecodes",
		"AppendTimecodes",
		"AppendDisplays",
		"Clone",
		"Append",
		"SPSData",
		"PPSData",
		"AnnexBData",
		"AVCCData",
		"AVCData",
		"AppendSPS",
		"AppendPPS",
		"AppendAnnexB",
		"AppendAVCC",
		"AppendNAL",
		"AppendAVC",
		"AppendNALData",
		"AppendAccessUnitData",
		"AppendRTPPacketData",
		"AppendRTPPayloadData",
		"PacketData",
		"AppendPacketData",
		"PayloadData",
		"AppendPayloadData",
		"OutputFormat",
		"AccessUnitData",
		"AccessUnitRange",
		"AccessUnitFormat",
		"NALData",
		"RTPPacketData",
		"RTPPayloadData",
		"Limits",
		"MaxFrameSizeLimit",
		"SliceMaxBytesLimit",
		"MaxEncodeTimeUSLimit",
	} {
		requireREADMECodeName(t, readme, name)
	}

	for _, tt := range []struct {
		typeName string
		typ      reflect.Type
		method   string
	}{
		{typeName: "EncoderParameterSets", typ: reflect.TypeOf(EncoderParameterSets{}), method: "Validate"},
		{typeName: "EncoderParameterSets", typ: reflect.TypeOf(EncoderParameterSets{}), method: "Clone"},
		{typeName: "EncoderParameterSets", typ: reflect.TypeOf(EncoderParameterSets{}), method: "SPSData"},
		{typeName: "EncoderParameterSets", typ: reflect.TypeOf(EncoderParameterSets{}), method: "PPSData"},
		{typeName: "EncoderParameterSets", typ: reflect.TypeOf(EncoderParameterSets{}), method: "AnnexBData"},
		{typeName: "EncoderParameterSets", typ: reflect.TypeOf(EncoderParameterSets{}), method: "AVCCData"},
		{typeName: "EncoderParameterSets", typ: reflect.TypeOf(EncoderParameterSets{}), method: "AppendSPS"},
		{typeName: "EncoderParameterSets", typ: reflect.TypeOf(EncoderParameterSets{}), method: "AppendPPS"},
		{typeName: "EncoderParameterSets", typ: reflect.TypeOf(EncoderParameterSets{}), method: "AppendAnnexB"},
		{typeName: "EncoderParameterSets", typ: reflect.TypeOf(EncoderParameterSets{}), method: "AppendAVCC"},
		{typeName: "EncoderSEI", typ: reflect.TypeOf(EncoderSEI{}), method: "Validate"},
		{typeName: "EncoderSEI", typ: reflect.TypeOf(EncoderSEI{}), method: "Clone"},
		{typeName: "EncoderSEI", typ: reflect.TypeOf(EncoderSEI{}), method: "NALData"},
		{typeName: "EncoderSEI", typ: reflect.TypeOf(EncoderSEI{}), method: "AnnexBData"},
		{typeName: "EncoderSEI", typ: reflect.TypeOf(EncoderSEI{}), method: "AVCData"},
		{typeName: "EncoderSEI", typ: reflect.TypeOf(EncoderSEI{}), method: "AppendNAL"},
		{typeName: "EncoderSEI", typ: reflect.TypeOf(EncoderSEI{}), method: "AppendAnnexB"},
		{typeName: "EncoderSEI", typ: reflect.TypeOf(EncoderSEI{}), method: "AppendAVC"},
		{typeName: "EncoderFrame", typ: reflect.TypeOf(EncoderFrame{}), method: "Validate"},
		{typeName: "EncoderFrame", typ: reflect.TypeOf(EncoderFrame{}), method: "Clone"},
		{typeName: "EncoderRTPPacket", typ: reflect.TypeOf(EncoderRTPPacket{}), method: "PacketData"},
		{typeName: "EncoderRTPPacket", typ: reflect.TypeOf(EncoderRTPPacket{}), method: "AppendPacketData"},
		{typeName: "EncoderRTPPacket", typ: reflect.TypeOf(EncoderRTPPacket{}), method: "PayloadData"},
		{typeName: "EncoderRTPPacket", typ: reflect.TypeOf(EncoderRTPPacket{}), method: "AppendPayloadData"},
		{typeName: "EncoderRTPPacket", typ: reflect.TypeOf(EncoderRTPPacket{}), method: "Validate"},
		{typeName: "EncoderRTPPacket", typ: reflect.TypeOf(EncoderRTPPacket{}), method: "Clone"},
		{typeName: "EncodedFrame", typ: reflect.TypeOf(EncodedFrame{}), method: "NALData"},
		{typeName: "EncodedFrame", typ: reflect.TypeOf(EncodedFrame{}), method: "AppendNALData"},
		{typeName: "EncodedFrame", typ: reflect.TypeOf(EncodedFrame{}), method: "AccessUnitData"},
		{typeName: "EncodedFrame", typ: reflect.TypeOf(EncodedFrame{}), method: "AccessUnitRange"},
		{typeName: "EncodedFrame", typ: reflect.TypeOf(EncodedFrame{}), method: "AccessUnitFormat"},
		{typeName: "EncodedFrame", typ: reflect.TypeOf(EncodedFrame{}), method: "AppendAccessUnitData"},
		{typeName: "EncodedFrame", typ: reflect.TypeOf(EncodedFrame{}), method: "RTPPacketData"},
		{typeName: "EncodedFrame", typ: reflect.TypeOf(EncodedFrame{}), method: "AppendRTPPacketData"},
		{typeName: "EncodedFrame", typ: reflect.TypeOf(EncodedFrame{}), method: "RTPPayloadData"},
		{typeName: "EncodedFrame", typ: reflect.TypeOf(EncodedFrame{}), method: "AppendRTPPayloadData"},
		{typeName: "EncodedFrame", typ: reflect.TypeOf(EncodedFrame{}), method: "Validate"},
		{typeName: "EncodedFrame", typ: reflect.TypeOf(EncodedFrame{}), method: "Clone"},
	} {
		if _, ok := tt.typ.MethodByName(tt.method); !ok {
			t.Fatalf("README encoder ownership names missing %s.%s", tt.typeName, tt.method)
		}
		requireREADMECodeName(t, readme, tt.typeName)
	}
}

func TestREADMEStructureKeepsDecoderPacketSurfaceTyped(t *testing.T) {
	data, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatalf("read README.md: %v", err)
	}
	readme := string(data)

	for _, phrase := range []string{
		"## API At A Glance",
		"| Decode stateful Annex B packets, stored configured-AVC packets, or avcC records | `dec.DecodeFrames(data)` |",
		"| Decode packet bytes plus packet side data such as `NEW_EXTRADATA` | `dec.DecodePacketFrames(Packet{Data: data, SideData: sideData})` |",
		"| Encode guarded realtime I420 to Annex B, AVC, or RTP | start from `DefaultRTPEncoderConfig`, `DefaultAnnexBEncoderConfig`, or `DefaultAVCEncoderConfig`; call `Normalize`, `NewEncoder`, then `Encode` or `EncodeInto` |",
		"Use the detailed Decoder API and Encoder API sections below for state,\nownership, error, and admission rules.",
	} {
		if !strings.Contains(readme, phrase) {
			t.Fatalf("README.md API-at-a-glance section missing phrase %q", phrase)
		}
	}

	for _, phrase := range []string{
		"frames, err := dec.DecodeFrames(packetData)",
		"frames, err = dec.DecodePacketFrames(goh264.Packet{",
		"Data:     packetData,",
		"SideData: sideData,",
		"}) // packet side data and NEW_EXTRADATA",
	} {
		if !strings.Contains(readme, phrase) {
			t.Fatalf("README.md decoder recommended path missing typed packet phrase %q", phrase)
		}
	}
	for _, forbidden := range []string{
		"DecodePacketFrames(packet) //",
		"DecodePacketFrames(packetData)",
	} {
		if strings.Contains(readme, forbidden) {
			t.Fatalf("README.md decoder recommended path should not blur Packet and []byte surfaces with %q", forbidden)
		}
	}

	seen := map[string]int{}
	for i, line := range strings.Split(readme, "\n") {
		if !strings.HasPrefix(line, "## ") {
			continue
		}
		heading := strings.TrimSpace(line)
		if firstLine, ok := seen[heading]; ok {
			t.Fatalf("README.md duplicate level-two heading %q at lines %d and %d", heading, firstLine, i+1)
		}
		seen[heading] = i + 1
	}
	requiredOrder := []string{
		"## API At A Glance",
		"## Quality And Parity Evidence",
		"## Decoder API",
		"## State And Ownership Boundaries",
		"## Encoder API",
		"## Trust And Verification",
		"## Performance",
	}
	last := -1
	for _, heading := range requiredOrder {
		idx := strings.Index(readme, heading+"\n")
		if idx < 0 {
			t.Fatalf("README.md missing heading %q", heading)
		}
		if idx <= last {
			t.Fatalf("README.md heading %q appears out of order", heading)
		}
		last = idx
	}
}

func TestDecoderOwnershipAPIReturnsErrors(t *testing.T) {
	errorType := reflect.TypeOf((*error)(nil)).Elem()
	oldCloneHelperName := "Clone" + "Checked"
	for _, tt := range []struct {
		typeName string
		typ      reflect.Type
	}{
		{typeName: "Packet", typ: reflect.TypeOf(Packet{})},
		{typeName: "PacketSideData", typ: reflect.TypeOf(PacketSideData{})},
		{typeName: "Frame", typ: reflect.TypeOf((*Frame)(nil))},
		{typeName: "FrameSideData", typ: reflect.TypeOf(FrameSideData{})},
		{typeName: "PictureTiming", typ: reflect.TypeOf((*PictureTiming)(nil))},
		{typeName: "ReferenceDisplaysInfo", typ: reflect.TypeOf((*ReferenceDisplaysInfo)(nil))},
	} {
		method, ok := tt.typ.MethodByName("Clone")
		if !ok {
			t.Fatalf("%s missing Clone", tt.typeName)
		}
		if method.Type.NumOut() == 0 || !method.Type.Out(method.Type.NumOut()-1).Implements(errorType) {
			t.Fatalf("%s.Clone should return an error", tt.typeName)
		}
		if _, ok := tt.typ.MethodByName(oldCloneHelperName); ok {
			t.Fatalf("%s.%s is not a canonical decoder ownership helper", tt.typeName, oldCloneHelperName)
		}
	}
	for _, tt := range []struct {
		typeName string
		typ      reflect.Type
		method   string
	}{
		{typeName: "Packet", typ: reflect.TypeOf(Packet{}), method: "AppendData"},
		{typeName: "Packet", typ: reflect.TypeOf(Packet{}), method: "AppendSideData"},
		{typeName: "PacketSideData", typ: reflect.TypeOf(PacketSideData{}), method: "AppendData"},
		{typeName: "FrameSideData", typ: reflect.TypeOf(FrameSideData{}), method: "AppendUserDataUnregistered"},
		{typeName: "FrameSideData", typ: reflect.TypeOf(FrameSideData{}), method: "AppendA53ClosedCaptions"},
		{typeName: "FrameSideData", typ: reflect.TypeOf(FrameSideData{}), method: "AppendICCProfile"},
		{typeName: "FrameSideData", typ: reflect.TypeOf(FrameSideData{}), method: "AppendDynamicHDR10Plus"},
		{typeName: "FrameSideData", typ: reflect.TypeOf(FrameSideData{}), method: "AppendLCEVC"},
		{typeName: "FrameSideData", typ: reflect.TypeOf(FrameSideData{}), method: "AppendS12MTimecodes"},
		{typeName: "PictureTiming", typ: reflect.TypeOf((*PictureTiming)(nil)), method: "Validate"},
		{typeName: "PictureTiming", typ: reflect.TypeOf((*PictureTiming)(nil)), method: "AppendTimecodes"},
		{typeName: "ReferenceDisplaysInfo", typ: reflect.TypeOf((*ReferenceDisplaysInfo)(nil)), method: "Validate"},
		{typeName: "ReferenceDisplaysInfo", typ: reflect.TypeOf((*ReferenceDisplaysInfo)(nil)), method: "AppendDisplays"},
	} {
		method, ok := tt.typ.MethodByName(tt.method)
		if !ok {
			t.Fatalf("%s missing %s", tt.typeName, tt.method)
		}
		if method.Type.NumOut() == 0 || !method.Type.Out(method.Type.NumOut()-1).Implements(errorType) {
			t.Fatalf("%s.%s should return an error", tt.typeName, tt.method)
		}
	}
}

func TestREADMEEncoderConfigValidationRowsSeparateValidateAndNormalize(t *testing.T) {
	data, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatalf("read README.md: %v", err)
	}
	readme := string(data)

	for _, phrase := range []string{
		"| Validate and see the exact setup before construction |",
		"`EncoderConfig.Normalize` or `Validate`",
	} {
		if strings.Contains(readme, phrase) {
			t.Fatalf("README.md couples EncoderConfig.Validate and Normalize in chooser row: %q", phrase)
		}
	}
	for _, phrase := range []string{
		"| Validate setup before construction | `EncoderConfig.Validate` |",
		"| View exact setup before construction | `EncoderConfig.Normalize` |",
		"`EncoderConfig.Validate` reports whether setup can be accepted without\n  returning normalized values",
		"Use `EncoderConfig.Normalize` when the caller\n  needs the exact setup",
	} {
		if !strings.Contains(readme, phrase) {
			t.Fatalf("README.md missing separated EncoderConfig validation phrase %q", phrase)
		}
	}
}

func TestEncoderHelperAPIReturnsErrors(t *testing.T) {
	errorType := reflect.TypeOf((*error)(nil)).Elem()
	for _, tt := range []struct {
		typeName string
		typ      reflect.Type
		method   string
	}{
		{typeName: "EncoderParameterSets", typ: reflect.TypeOf(EncoderParameterSets{}), method: "Validate"},
		{typeName: "EncoderParameterSets", typ: reflect.TypeOf(EncoderParameterSets{}), method: "SPSData"},
		{typeName: "EncoderParameterSets", typ: reflect.TypeOf(EncoderParameterSets{}), method: "PPSData"},
		{typeName: "EncoderParameterSets", typ: reflect.TypeOf(EncoderParameterSets{}), method: "AnnexBData"},
		{typeName: "EncoderParameterSets", typ: reflect.TypeOf(EncoderParameterSets{}), method: "AVCCData"},
		{typeName: "EncoderParameterSets", typ: reflect.TypeOf(EncoderParameterSets{}), method: "AppendSPS"},
		{typeName: "EncoderParameterSets", typ: reflect.TypeOf(EncoderParameterSets{}), method: "AppendPPS"},
		{typeName: "EncoderParameterSets", typ: reflect.TypeOf(EncoderParameterSets{}), method: "AppendAnnexB"},
		{typeName: "EncoderParameterSets", typ: reflect.TypeOf(EncoderParameterSets{}), method: "AppendAVCC"},
		{typeName: "EncoderParameterSets", typ: reflect.TypeOf(EncoderParameterSets{}), method: "Clone"},
		{typeName: "EncoderSEI", typ: reflect.TypeOf(EncoderSEI{}), method: "Validate"},
		{typeName: "EncoderSEI", typ: reflect.TypeOf(EncoderSEI{}), method: "NALData"},
		{typeName: "EncoderSEI", typ: reflect.TypeOf(EncoderSEI{}), method: "AnnexBData"},
		{typeName: "EncoderSEI", typ: reflect.TypeOf(EncoderSEI{}), method: "AVCData"},
		{typeName: "EncoderSEI", typ: reflect.TypeOf(EncoderSEI{}), method: "AppendNAL"},
		{typeName: "EncoderSEI", typ: reflect.TypeOf(EncoderSEI{}), method: "AppendAnnexB"},
		{typeName: "EncoderSEI", typ: reflect.TypeOf(EncoderSEI{}), method: "AppendAVC"},
		{typeName: "EncoderSEI", typ: reflect.TypeOf(EncoderSEI{}), method: "Clone"},
		{typeName: "EncoderFrame", typ: reflect.TypeOf(EncoderFrame{}), method: "Validate"},
		{typeName: "EncoderFrame", typ: reflect.TypeOf(EncoderFrame{}), method: "Clone"},
		{typeName: "EncoderRTPPacket", typ: reflect.TypeOf(EncoderRTPPacket{}), method: "PacketData"},
		{typeName: "EncoderRTPPacket", typ: reflect.TypeOf(EncoderRTPPacket{}), method: "AppendPacketData"},
		{typeName: "EncoderRTPPacket", typ: reflect.TypeOf(EncoderRTPPacket{}), method: "PayloadData"},
		{typeName: "EncoderRTPPacket", typ: reflect.TypeOf(EncoderRTPPacket{}), method: "AppendPayloadData"},
		{typeName: "EncoderRTPPacket", typ: reflect.TypeOf(EncoderRTPPacket{}), method: "Validate"},
		{typeName: "EncoderRTPPacket", typ: reflect.TypeOf(EncoderRTPPacket{}), method: "Clone"},
		{typeName: "EncodedFrame", typ: reflect.TypeOf(EncodedFrame{}), method: "NALData"},
		{typeName: "EncodedFrame", typ: reflect.TypeOf(EncodedFrame{}), method: "AppendNALData"},
		{typeName: "EncodedFrame", typ: reflect.TypeOf(EncodedFrame{}), method: "AccessUnitData"},
		{typeName: "EncodedFrame", typ: reflect.TypeOf(EncodedFrame{}), method: "AccessUnitRange"},
		{typeName: "EncodedFrame", typ: reflect.TypeOf(EncodedFrame{}), method: "AccessUnitFormat"},
		{typeName: "EncodedFrame", typ: reflect.TypeOf(EncodedFrame{}), method: "AppendAccessUnitData"},
		{typeName: "EncodedFrame", typ: reflect.TypeOf(EncodedFrame{}), method: "RTPPacketData"},
		{typeName: "EncodedFrame", typ: reflect.TypeOf(EncodedFrame{}), method: "AppendRTPPacketData"},
		{typeName: "EncodedFrame", typ: reflect.TypeOf(EncodedFrame{}), method: "RTPPayloadData"},
		{typeName: "EncodedFrame", typ: reflect.TypeOf(EncodedFrame{}), method: "AppendRTPPayloadData"},
		{typeName: "EncodedFrame", typ: reflect.TypeOf(EncodedFrame{}), method: "Validate"},
		{typeName: "EncodedFrame", typ: reflect.TypeOf(EncodedFrame{}), method: "Clone"},
	} {
		method, ok := tt.typ.MethodByName(tt.method)
		if !ok {
			t.Fatalf("%s missing %s", tt.typeName, tt.method)
		}
		if method.Type.NumOut() == 0 || !method.Type.Out(method.Type.NumOut()-1).Implements(errorType) {
			t.Fatalf("%s.%s should return an error", tt.typeName, tt.method)
		}
	}

	for _, tt := range []struct {
		typeName string
		typ      reflect.Type
		baseName string
	}{
		{typeName: "EncoderParameterSets", typ: reflect.TypeOf(EncoderParameterSets{}), baseName: "Validate"},
		{typeName: "EncoderParameterSets", typ: reflect.TypeOf(EncoderParameterSets{}), baseName: "SPSData"},
		{typeName: "EncoderParameterSets", typ: reflect.TypeOf(EncoderParameterSets{}), baseName: "PPSData"},
		{typeName: "EncoderParameterSets", typ: reflect.TypeOf(EncoderParameterSets{}), baseName: "AnnexBData"},
		{typeName: "EncoderParameterSets", typ: reflect.TypeOf(EncoderParameterSets{}), baseName: "AVCCData"},
		{typeName: "EncoderParameterSets", typ: reflect.TypeOf(EncoderParameterSets{}), baseName: "AppendSPS"},
		{typeName: "EncoderParameterSets", typ: reflect.TypeOf(EncoderParameterSets{}), baseName: "AppendPPS"},
		{typeName: "EncoderParameterSets", typ: reflect.TypeOf(EncoderParameterSets{}), baseName: "AppendAnnexB"},
		{typeName: "EncoderParameterSets", typ: reflect.TypeOf(EncoderParameterSets{}), baseName: "AppendAVCC"},
		{typeName: "EncoderParameterSets", typ: reflect.TypeOf(EncoderParameterSets{}), baseName: "Clone"},
		{typeName: "EncoderSEI", typ: reflect.TypeOf(EncoderSEI{}), baseName: "Validate"},
		{typeName: "EncoderSEI", typ: reflect.TypeOf(EncoderSEI{}), baseName: "NALData"},
		{typeName: "EncoderSEI", typ: reflect.TypeOf(EncoderSEI{}), baseName: "AnnexBData"},
		{typeName: "EncoderSEI", typ: reflect.TypeOf(EncoderSEI{}), baseName: "AVCData"},
		{typeName: "EncoderSEI", typ: reflect.TypeOf(EncoderSEI{}), baseName: "AppendNAL"},
		{typeName: "EncoderSEI", typ: reflect.TypeOf(EncoderSEI{}), baseName: "AppendAnnexB"},
		{typeName: "EncoderSEI", typ: reflect.TypeOf(EncoderSEI{}), baseName: "AppendAVC"},
		{typeName: "EncoderSEI", typ: reflect.TypeOf(EncoderSEI{}), baseName: "Clone"},
		{typeName: "EncoderFrame", typ: reflect.TypeOf(EncoderFrame{}), baseName: "Validate"},
		{typeName: "EncoderFrame", typ: reflect.TypeOf(EncoderFrame{}), baseName: "Clone"},
		{typeName: "EncoderRTPPacket", typ: reflect.TypeOf(EncoderRTPPacket{}), baseName: "PacketData"},
		{typeName: "EncoderRTPPacket", typ: reflect.TypeOf(EncoderRTPPacket{}), baseName: "AppendPacketData"},
		{typeName: "EncoderRTPPacket", typ: reflect.TypeOf(EncoderRTPPacket{}), baseName: "PayloadData"},
		{typeName: "EncoderRTPPacket", typ: reflect.TypeOf(EncoderRTPPacket{}), baseName: "AppendPayloadData"},
		{typeName: "EncoderRTPPacket", typ: reflect.TypeOf(EncoderRTPPacket{}), baseName: "Validate"},
		{typeName: "EncoderRTPPacket", typ: reflect.TypeOf(EncoderRTPPacket{}), baseName: "Clone"},
		{typeName: "EncodedFrame", typ: reflect.TypeOf(EncodedFrame{}), baseName: "NALData"},
		{typeName: "EncodedFrame", typ: reflect.TypeOf(EncodedFrame{}), baseName: "AppendNALData"},
		{typeName: "EncodedFrame", typ: reflect.TypeOf(EncodedFrame{}), baseName: "AccessUnitData"},
		{typeName: "EncodedFrame", typ: reflect.TypeOf(EncodedFrame{}), baseName: "AccessUnitRange"},
		{typeName: "EncodedFrame", typ: reflect.TypeOf(EncodedFrame{}), baseName: "AccessUnitFormat"},
		{typeName: "EncodedFrame", typ: reflect.TypeOf(EncodedFrame{}), baseName: "AppendAccessUnitData"},
		{typeName: "EncodedFrame", typ: reflect.TypeOf(EncodedFrame{}), baseName: "RTPPacketData"},
		{typeName: "EncodedFrame", typ: reflect.TypeOf(EncodedFrame{}), baseName: "AppendRTPPacketData"},
		{typeName: "EncodedFrame", typ: reflect.TypeOf(EncodedFrame{}), baseName: "RTPPayloadData"},
		{typeName: "EncodedFrame", typ: reflect.TypeOf(EncodedFrame{}), baseName: "AppendRTPPayloadData"},
		{typeName: "EncodedFrame", typ: reflect.TypeOf(EncodedFrame{}), baseName: "Validate"},
		{typeName: "EncodedFrame", typ: reflect.TypeOf(EncodedFrame{}), baseName: "Clone"},
	} {
		methodName := tt.baseName + "Checked"
		if _, ok := tt.typ.MethodByName(methodName); ok {
			t.Fatalf("%s.%s is not a canonical encoder helper name", tt.typeName, methodName)
		}
	}
}

func TestREADMEQualityEvidenceDoesNotTreatExamplesAsParityEvidence(t *testing.T) {
	data, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatalf("read README.md: %v", err)
	}
	readme := string(data)
	readmeLower := strings.ToLower(readme)
	for _, phrase := range []string{
		"Quality And Parity Evidence",
		"Guarded realtime subset",
		"Outside current contract / evidence targets",
		"Examples",
		"API smoke tests only",
		"broader/full bitstream parity beyond admitted oracle rows",
		"acceptance",
	} {
		if !strings.Contains(readme, phrase) {
			t.Fatalf("README.md missing quality/parity status phrase %q", phrase)
		}
	}
	for _, forbidden := range []string{
		statusPhrase("pre", "-re", "lease"),
		statusPhrase("pre", "-production"),
		statusPhrase("not production", "-ready"),
		statusPhrase("non", "-production"),
		statusPhrase("non", "-re", "lease"),
		statusPhrase("re", "lease tag"),
		statusPhrase("re", "lease readiness"),
		statusPhrase("re", "lease artifacts"),
		statusPhrase("published ", "version"),
		statusPhrase("no ", "tag"),
		statusPhrase("no ", "tags"),
		statusPhrase("de", "pre", "cate"),
		statusPhrase("de", "pre", "cated"),
		statusPhrase("de", "pre", "cation"),
	} {
		if strings.Contains(readmeLower, forbidden) {
			t.Fatalf("README.md should not use shipping-status phrase %q", forbidden)
		}
	}
}

func TestDocsAndScriptsAvoidShippingStatusWording(t *testing.T) {
	var files []string
	for _, root := range []string{"README.md", "docs", "scripts"} {
		info, err := os.Stat(root)
		if err != nil {
			t.Fatalf("stat %s: %v", root, err)
		}
		if !info.IsDir() {
			files = append(files, root)
			continue
		}
		err = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				return nil
			}
			if strings.HasSuffix(path, ".md") || strings.HasSuffix(path, ".sh") {
				files = append(files, path)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", root, err)
		}
	}
	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		text := strings.ToLower(string(data))
		for _, forbidden := range []string{
			statusPhrase("pre", "-re", "lease"),
			statusPhrase("pre ", "re", "lease"),
			statusPhrase("pre", "-production"),
			statusPhrase("pre ", "production"),
			statusPhrase("not production", "-ready"),
			statusPhrase("non", "-production"),
			statusPhrase("non", "-re", "lease"),
			statusPhrase("non ", "re", "lease"),
			statusPhrase("re", "lease", "-evidence"),
			statusPhrase("re", "lease evidence"),
			statusPhrase("re", "lease", "_evidence"),
			statusPhrase("re", "lease", "-alloc"),
			statusPhrase("re", "lease alloc"),
			statusPhrase("re", "lease", "_alloc"),
			statusPhrase("re", "lease gate"),
			statusPhrase("re", "lease runner"),
			statusPhrase("re", "lease checklist"),
			statusPhrase("re", "lease canary"),
			statusPhrase("re", "lease docs"),
			statusPhrase("re", "lease path"),
			statusPhrase("re", "lease readiness"),
			statusPhrase("re", "lease artifacts"),
			statusPhrase("re", "lease tag"),
			statusPhrase("re", "leased version"),
			statusPhrase("published ", "version"),
			statusPhrase("rem", "oved ", "from the ", "ledger"),
			statusPhrase("rem", "ove ", "from the ", "ledger"),
			statusPhrase("encoder api (experimental)"),
			statusPhrase("experimental admitted subset"),
			statusPhrase("rejected/not-yet-admitted"),
			statusPhrase("not-yet-admitted"),
			statusPhrase("not admitted yet"),
			statusPhrase("unsupported future tools"),
			statusPhrase("no ", "tag"),
			statusPhrase("no ", "tags"),
			statusPhrase("goh264", "_re", "lease"),
			statusPhrase("goh264_full", "_re", "lease"),
			statusPhrase("goh264_encoder", "_re", "lease"),
			statusPhrase("de", "pre", "cate"),
			statusPhrase("de", "pre", "cated"),
			statusPhrase("de", "pre", "cation"),
		} {
			if strings.Contains(text, forbidden) {
				t.Fatalf("%s should not use shipping-status wording %q", path, forbidden)
			}
		}
	}
}

func statusPhrase(parts ...string) string {
	return strings.Join(parts, "")
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
		"must(enc.SetOutputFormat(goh264.EncoderOutputAVC)) // queues an IDR boundary\nenc.SetRTPPacketCallback",
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
		"unsupported tools return ErrUnsupported",
		"`SetRTPPacketCallback` per-packet metadata callbacks for RTP output",
		"Zero scalar fields in `EncoderReconfigure` mean unchanged",
		"pointer fields, grouped `Limits`, or dedicated setters",
		"`EncoderConfig.ExplicitQP=true`",
		"`SetQP` and pointer QP fields",
		"`FrameRateNum`/`FrameRateDen` and `Width`/`Height` must be supplied",
		"explicit timestamp increment controls subsequent automatic RTP cadence",
		"When `Limits` is non-nil, it is applied after the individual budget",
	} {
		if !strings.Contains(readme, required) {
			t.Fatalf("README.md encoder sample missing checked-control phrase %q", required)
		}
	}
}

func TestEncoderQualityEvidenceNamesAPISurfaceGate(t *testing.T) {
	readmeData, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatalf("read README.md: %v", err)
	}
	scriptData, err := os.ReadFile("scripts/h264-encoder-quality-evidence.sh")
	if err != nil {
		t.Fatalf("read encoder quality evidence script: %v", err)
	}
	readme := string(readmeData)
	script := string(scriptData)
	for _, phrase := range []string{
		"API-surface",
		"output-ownership",
		"bitstream-oracles",
		"residual-boundary",
	} {
		if !strings.Contains(readme, phrase) {
			t.Fatalf("README.md encoder quality evidence text missing %q", phrase)
		}
	}
	for _, phrase := range []string{
		"run_go_test_gate()",
		"run_exact_go_test_gate()",
		"go test \"$pkg\" -list \"$pattern\"",
		"go test \"$pkg\" -run '^$' -list \"$pattern\"",
		"status: fail (no matching tests)",
		"status: fail (missing focused test",
		"status: fail (focused test skipped)",
		"run_go_test_gate encoder-contract ./tests",
		"run_exact_go_test_gate encoder-api-surfaces ./tests",
		"run_exact_go_test_gate encoder-output-ownership ./tests",
		"run_exact_go_test_gate encoder-bitstream-oracles ./tests",
		"run_go_test_gate encoder-residual-boundary ./tests",
		"run_go_test_gate encoder-allocation-canary ./tests",
		"run_go_test_gate encoder-writers ./internal/h264",
		"GOH264_ENCODER_REQUIRE_FFMPEG=1",
		"GOH264_FFMPEG_BIN",
		"ffmpeg-oracle: fail",
		"ffmpeg-oracle: pass",
		"encoder_api_doc_tests",
		"run_exact_go_test_gate encoder-api-docs .",
		"encoder-api-surfaces",
		"encoder_api_surface_tests",
		"encoder-output-ownership",
		"encoder_output_ownership_tests",
		"encoder_bitstream_oracle_tests",
		"encoder-bitstream-oracles",
		"TestEncoderEncodeAnnexBIDRIntraPCMDecodesThroughLocalAndFFmpeg",
		"TestEncoderEncodeCroppedAnnexBIDRIntraPCMDecodesVisibleFrame",
		"TestEncoderEncodeAVCIDRIntraPCMDecodesThroughConfiguredSurface",
		"TestEncoderEncodeIdenticalSecondFrameUsesPSkipReference",
		"TestEncoderEncodeExactP16x16NoResidualMotion",
		"TestEncoderEncodeExactP16x16NoResidualMotionForAVCAndRTP",
		"TestEncoderEncodeExactP16x16NoResidualMotionWithDeblockControls",
		"TestEncoderEncodeMacroblockAlignedExactP16x16NoResidualMotion",
		"TestEncoderEncodePerMacroblockExactP16x16NoResidualMotionForAnnexBAVCRTP",
		"TestEncoderEncodePerMacroblockExactP16x16FallsBackWithDeblockControls",
		"TestEncoderEncodeOddPixelExactP16x16NoResidualMotionWithConstantChroma",
		"TestEncoderEncodeOddPixelExactP16x16NoResidualMotionForAVCAndRTP",
		"TestEncoderEncodeChangedSecondFrameUsesPIntraPCM",
		"TestEncoderEncodeChangedSecondFrameUsesPIntraPCMWithDefaultDeblock",
		"TestEncoderEncodeChangedSecondFrameUsesPIntraPCMWithSliceBoundaryDeblock",
		"TestEncoderEncodeChangedPIntraPCMRecoveryPointSEIForAVCAndRTP",
		"TestEncoderResidualShapedPDeltaUsesResidualPAcrossPublicOutputs",
		"TestEncoderSliceCountSplitsIDRPSkipAndPIntraPCMAccessUnits",
		"TestEncoderSliceCountFeedsRTPMode1SingleNALPackets",
		"TestEncoderEncodeForceIDRBypassesPSkipReference",
		"TestEncoderEncodeRTPMode1FragmentsIDRAccessUnit",
		"TestEncoderEncodeRTPMode1STAPAAggregatesParameterSets",
		"TestEncoderEncodeRTPMode1STAPADoesNotAggregateChangedPRecoverySEI",
		"TestEncoderEncodeRTPMode0EmitsSingleNALPackets",
		"TestEncoderEncodeRTPMode0EmitsPFrameSingleNALPackets",
		"TestEncoderRTPMode1STAPAFallbackAtSmallPayloadPreservesLiveState",
		"TestEncoderEncodeIntoRTPPacketsDoNotAliasAccessUnitData",
		"TestEncoderConfigExplicitQPZeroConstructsAndEncodes",
		"TestEncoderRTPPayloadTypeZeroSelectsDynamicDefault",
		"TestEncoderRTPPayloadTypeZeroNormalizesForNonRTPOutputs",
		"TestEncoderRTPPacketCallbackReconfigureAppliesAfterCurrentResult",
		"TestEncoderRealtimeWebRTCRejectsInvalidConfigs",
		"TestEncoderRTPAutoTimestampAdvancesWithTimestampModeAuto",
		"TestEncoderTimestampAutoOutputTimingMatchesChosenRTPTime",
		"TestEncoderTimestampAutoDroppedOutputTimingMatchesChosenRTPTime",
		"TestEncoderTimestampAutoDoesNotValidateIgnoredPTS",
		"TestEncoderRTPExplicitZeroPTSAfterNonZeroPTSIsHonored",
		"TestEncoderReconfigureExplicitTimestampIncrementWinsWithFrameRate",
		"TestEncoderEncodeIntoValidatesInvalidFrameBeforeBitstream",
		"TestEncoderRealtimeWebRTCControlSurfaceCoversRoadmap",
		"TestEncoderSetQPZeroSurvivesNoopReconfigureAndEncodes",
		"TestEncoderReconfigureLimitPointersDisableBudgets",
		"TestEncoderReconfigureZeroScalarFieldsAreNoOps",
		"TestEncoderReconfigureLimitsGroupUpdatesBudgetsAtomically",
		"TestEncoderZeroValueExplicitSettersRejectWithoutMutation",
		"TestEncoderNonRTPConfigsRejectInvalidRTPControls",
		"TestEncoderInvalidRTPControlsRejectForNonRTPOutputsWithoutMutation",
		"TestEncoderReconfigureOutputFormatQueuesIDRBoundary",
		"TestEncoderManualNonRTPConfigDefaultsDONDisabledForRTPReentry",
		"TestEncoderInvalidRTPSettersPreservePacketState",
		"TestEncoderFrameColorDoesNotOverrideConfigHeaders",
		"TestEncoderValidSetterPreservesPendingIDR",
		"TestEncoderInvalidSetterPreservesPendingIDR",
		"TestEncoderValidReconfigurePreservesPendingIDR",
		"TestEncoderValidOutputReconfigurePreservesPendingIDR",
		"TestEncoderInvalidReconfigurePreservesPendingIDR",
		"TestEncoderInvalidReconfigureWithForceIDRDoesNotQueueIDR",
		"TestEncoderFrameRateInvalidUpdatesPreserveLiveState",
		"TestEncoderSetIntraRefreshEnableIsUnsupportedAndPreservesState",
		"TestEncoderSetIntraRefreshDisablePreservesLiveReference",
		"TestEncoderSetLimitsUpdatesBudgetsAtomically",
		"TestEncodedFrameAccessUnitRangeAndFormat",
		"TestEncodedFrameNALDataRejectsInvalidIndexesAndMetadata",
		"TestEncodedFrameRTPDataRejectsInvalidIndexesAndMetadata",
		"TestEncodedFrameRTPPacketDataRejectsMalformedPayload",
		"TestEncodedFrameAppendNALAndAccessUnitDataReturnCallerOwnedBytes",
		"TestEncodedFrameAppendRTPDataReturnsCallerOwnedBytes",
		"TestEncoderAppendHelpersIsolateOverlappingSource",
		"TestEncoderParameterSetsDataHelpersReturnClippedBytes",
		"TestEncoderSEIDataHelpersReturnClippedBytes",
		"TestEncoderRTPPacketDataRejectsMalformedPayload",
		"TestEncoderRTPPacketDataHelpersReturnClippedCallerOwnedBytes",
		"TestEncodedFrameCloneRejectsInvalidMetadata",
		"TestEncodedFrameValidateRejectsFrameLevelMetadataMismatches",
		"TestEncodedFrameValidateRejectsRTPListMetadataMismatches",
		"TestEncoderChromaOnlyResidualPUsesResidualAcrossPublicOutputs",
		"TestEncoderCombinedResidualPUsesResidualAcrossPublicOutputs",
		"TestEncoderMultiMacroblockLumaDCResidualPUsesResidualAcrossPublicOutputs",
		"TestEncoderMultiSliceLumaDCResidualPUsesResidualAcrossPublicOutputs",
		"TestEncoderSixMacroblockRowCrossingLumaDCResidualPUsesResidualAcrossPublicOutputs",
		"TestEncoderHelperClonesRejectOverflowedPublicStorage",
		"TestEncoderAppendHelpersRejectOverflowedPublicStorage",
		"TestEncodedFrameOutputHelpersRejectOverflowedPublicStorage",
		"encoder-writers",
		"TestCAVLCWriteResidual",
		"TestWriteCAVLCInterPBoundedMacroblock",
		"TestEncodeI420P16x16ResidualSliceRBSP",
		"TestEncoderFrameCloneDeepCopiesInputPlanes",
		"TestEncoderFrameValidateAndCloneRejectOverflowedPlanes",
		"TestEncoderParameterSetsReturnCallerOwnedSurfaces",
		"TestEncoderEncodeIntoRTPMode0RejectPreservesCallerBuffer",
		"TestEncoderDoesNotRetainInputFramePlanes",
		"TestEncoderEncodeResultsSurviveLaterEncode",
		"TestEncodedFrameCloneDeepCopiesResultStorage",
		"TestEncoderRTPPacketsDoNotAliasEncodedFrameData",
		"TestEncoderRTPPacketCallbackPacketsSurviveLaterEncode",
		"encoder-api-docs",
		"TestREADMECodecAPIChooserNamesPublicEntryPoints",
		"TestEncoderHelperAPIReturnsErrors",
		"TestREADMEEncoderRTPDataSurfaceDocumentsPacketBytes",
	} {
		if !strings.Contains(script, phrase) {
			t.Fatalf("encoder quality evidence script missing API-surface gate phrase %q", phrase)
		}
	}
	for _, forbidden := range []string{
		"run_gate encoder-contract go test",
		"run_gate encoder-api-surfaces go test",
		"run_gate encoder-output-ownership go test",
		"run_gate encoder-bitstream-oracles go test",
		"run_gate encoder-residual-boundary go test",
		"run_gate encoder-allocation-canary go test",
		"run_gate encoder-writers go test",
		"run_go_test_gate encoder-api-surfaces ./tests",
		"run_go_test_gate encoder-output-ownership ./tests",
		"run_go_test_gate encoder-bitstream-oracles ./tests",
	} {
		if strings.Contains(script, forbidden) {
			t.Fatalf("encoder quality evidence script should preflight focused gate %q", forbidden)
		}
	}
}

func TestDecoderQualityEvidenceNamesAPISurfaceAndRefGates(t *testing.T) {
	readmeData, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatalf("read README.md: %v", err)
	}
	scriptData, err := os.ReadFile("scripts/h264-decoder-quality-evidence.sh")
	if err != nil {
		t.Fatalf("read decoder quality evidence script: %v", err)
	}
	readme := string(readmeData)
	script := string(scriptData)
	for _, phrase := range []string{
		"decoder API-surface",
		"decoder output-ownership gates",
		"ref-modification gates",
		"delayed-output rollback gates",
		"native/FFmpeg oracle smoke gates",
	} {
		if !strings.Contains(readme, phrase) {
			t.Fatalf("README.md decoder quality evidence text missing %q", phrase)
		}
	}
	for _, phrase := range []string{
		"run_go_test_gate()",
		"run_exact_go_test_gate()",
		"run_exact_env_go_test_gate()",
		"run_exact_oracle_go_test_gate()",
		"require_oracle_command()",
		"require_oracle_file()",
		"go test \"$pkg\" -list \"$pattern\"",
		"go test \"$pkg\" -run '^$' -list \"$pattern\"",
		"status: fail (no matching tests)",
		"status: fail (missing focused test",
		"status: fail (focused test skipped)",
		"status: fail (oracle test skipped)",
		"run_exact_go_test_gate decoder-api-surfaces ./tests",
		"run_exact_go_test_gate decoder-output-ownership ./tests",
		"run_exact_go_test_gate decoder-ref-modifications ./internal/h264",
		"run_exact_go_test_gate decoder-delayed-output ./internal/h264",
		"run_exact_go_test_gate decoder-public-delayed-output ./tests",
		"run_exact_oracle_go_test_gate decoder-ffmpeg-oracle-smoke ./tests",
		"run_exact_oracle_go_test_gate decoder-native-oracle-smoke ./internal/h264",
		"run_exact_env_go_test_gate real-vector-failure-ledger ./tests",
		"run_exact_env_go_test_gate real-vector-matrix ./tests",
		"require_oracle_command ffmpeg",
		"require_oracle_command ffprobe",
		"require_oracle_command cc",
		"require_oracle_file .upstream/ffmpeg-n8.0.1/libavcodec/cabac.c",
		"require_oracle_file .upstream/ffmpeg-n8.0.1/libavcodec/h264idct_template.c",
		"decoder-api-surfaces",
		"decoder-output-ownership",
		"decoder_output_ownership_tests",
		"decoder-ref-modifications",
		"decoder-delayed-output",
		"decoder-public-delayed-output",
		"decoder_public_delayed_output_tests",
		"decoder-ffmpeg-oracle-smoke",
		"decoder-native-oracle-smoke",
		"TestParseHeadersAnnexBBlack16",
		"TestParseHeadersAVCBlack16",
		"TestPackageAVCCParsersDoNotMutateDecoderState",
		"TestDecodeAVCOneByteLengthSizePublicSurfaces",
		"TestFrameCloneRejectsOverflowedPublicStorage",
		"TestDecoderCloneHelpersRejectOverflowedPublicStorage",
		"TestDecodePacketFramesIgnoresOverflowedSideDataListWithoutDroppingPacket",
		"TestDecodePacketRejectsOverflowedPacketDataBeforeNewExtradata",
		"TestDecodePacketFramesIgnoresOverflowedPacketSideDataPayloadsWithoutDroppingPacket",
		"TestDecodePacketFramesOverflowedPacketSideDataPayloadSuppressesLaterDuplicate",
		"TestDecodeFramesSEIOnlyPacketAppliesToNextFrame",
		"TestDecodePacketFramesSEIOnlyPacketAppliesToNextFrame",
		"TestDecodeAVCCFramesIncompatibleConfigurationDoesNotUseStalePFrameReference",
		"TestDecodePacketFramesNewExtradataIncompatibleConfigurationDoesNotUseStalePFrameReference",
		"TestDecodePacketFramesAnnexBNewExtradataIncompatibleConfigurationDoesNotUseStalePFrameReference",
		"TestDecodePacketFramesDuplicateNewExtradataFirstEntryWins",
		"TestDecodePacketFramesMalformedDuplicateNewExtradataSuppressesLaterEntries",
		"TestDecodePacketFramesEmptyDuplicateNewExtradataSuppressesLaterEntries",
		"TestDecodePacketFramesOverflowedDuplicateNewExtradataSuppressesLaterEntries",
		"TestDecodeFramesInBandIncompatibleParameterSetsDoNotUseStalePFrameReference",
		"TestDecodePacketFramesInBandIncompatibleParameterSetsDoNotUseStalePFrameReference",
		"TestDecodeConfiguredAVCFramesInBandIncompatibleParameterSetsDoNotUseStalePFrameReference",
		"TestParseHeadersAnnexBIncompatibleHeadersDoNotUseStalePFrameReference",
		"TestParseHeadersAVCIncompatibleHeadersDoNotUseStalePFrameReference",
		"TestDecodeFramesAnnexBRejectsPartialInBandParameterSetsTransactionally",
		"TestDecodeFramesAnnexBRejectsPartialInBandHeaderOnlyWithoutPoisoningConfig",
		"TestDecodeConfiguredAVCFramesRejectsPartialInBandParameterSetsTransactionally",
		"TestDecodeConfiguredAVCFramesRejectsPartialInBandHeaderOnlyWithoutPoisoningConfig",
		"TestDecodeFramesValidInBandParameterSetsBeforeDamagedSliceUpdateConfigAndRecover",
		"TestValidAVCCBeforeDamagedSliceUpdatesConfigAndRecover",
		"TestDecodeAVCCFramesMultiSPSConfigurationUsesPacketActiveSPSForDPBReset",
		"TestDecodeFramesStandaloneMultiSPSConfigurationResetsForNonFirstActiveSPS",
		"TestDecodePacketFramesMultiSPSNewExtradataUsesPacketActiveSPSForDPBReset",
		"TestDecodePacketFramesAnnexBMultiSPSNewExtradataUsesPacketActiveSPSForDPBReset",
		"TestDecoderAVCConfigUsesAVCCFirstSPSForMultiSPSConfiguration",
		"TestDecoderAVCConfigUsesPacketActiveSPSForMultiSPSConfiguration",
		"TestSimpleFrameDPBRejectsMissingShortRefModificationTarget",
		"TestSimpleFrameDPBRejectsMissingLongRefModificationTarget",
		"TestSimpleDecoderFlushDelayedFrameRejectsMultipleWithoutDraining",
		"TestDecodeAVCCFramesEmptyPacketIncompatibleConfigPreservesDelayedFlush",
		"TestDecodeAVCCEmptyPacketIncompatibleConfigPreservesSingleDelayedFlush",
		"TestDecodePacketFramesRepeatedNewExtradataPreservesDelayedBFrames",
		"TestDecodeAVCCFramesBFramesFlushesReorderedPrefixBeforeDamagedSlice",
		"TestParseHeadersRejectPreservesDelayedConfiguredAVCFlush",
		"TestDecodeConfiguredAVCFramesDoesNotAliasCallerBuffer",
		"TestDecodePacketFramesDoesNotAliasCallerBuffer",
		"TestConfigureAVCCDoesNotAliasCallerBuffer",
		"TestPacketCloneDeepCopiesDataAndSideData",
		"TestPacketAppendSideDataReturnsCallerOwnedValues",
		"TestPacketAppendDataReturnsCallerOwnedBytes",
		"TestPacketSideDataAppendDataReturnsCallerOwnedBytes",
		"TestFrameSideDataNestedHelpersDeepCopyAndValidate",
		"TestFrameSideDataAppendHelpersReturnCallerOwnedBytes",
		"TestFrameSideDataAppendTypedHelpersReturnCallerOwnedValues",
		"TestFrameCloneDeepCopiesPlanesAndSideData",
		"TestDecodeFrameSideDataByteSlicesAreCallerOwned",
		"TestFrameAppendRawYUVIsolatesOverlappingSource",
		"TestFrameAppendRawYUVErrorPreservesCallerBuffer",
		"TestS12MTimecodePackingMatchesNativeFFmpegOracle",
		"TestFFprobeOracleBlack16",
		"TestFFmpegFrameMD5OracleBlack16",
		"TestCABACPrimitiveSequenceUpstreamOracle",
		"TestH264IDCTUpstreamOracle",
	} {
		if !strings.Contains(script, phrase) {
			t.Fatalf("decoder quality evidence script missing focused gate phrase %q", phrase)
		}
	}
	for _, forbidden := range []string{
		"run_gate decoder-api-surfaces go test",
		"run_gate decoder-output-ownership go test",
		"run_gate decoder-ref-modifications go test",
		"run_gate decoder-delayed-output go test",
		"run_go_test_gate decoder-api-surfaces ./tests",
		"run_go_test_gate decoder-output-ownership ./tests",
		"run_go_test_gate decoder-ref-modifications ./internal/h264",
		"run_go_test_gate decoder-delayed-output ./internal/h264",
		"run_go_test_gate decoder-public-delayed-output ./tests",
		"run_env_gate real-vector-failure-ledger",
		"run_env_gate real-vector-matrix",
	} {
		if strings.Contains(script, forbidden) {
			t.Fatalf("decoder quality evidence script should preflight focused gate %q", forbidden)
		}
	}
}

func TestQualityEvidenceHelpersRejectEmptyAndSkippedRuns(t *testing.T) {
	for _, tt := range []struct {
		path    string
		phrases []string
	}{
		{
			path: "scripts/h264-real-vector-strict.sh",
			phrases: []string{
				"go test ./tests -run '^$' -list \"$pattern\"",
				"TestH264RealVectorStrictOracle",
				"status: fail (missing focused test",
				"status: fail (focused test skipped)",
			},
		},
		{
			path: "scripts/h264-real-vector-upstream-audit.sh",
			phrases: []string{
				"go test ./tests -run '^$' -list \"$pattern\"",
				"TestH264RealVectorImportedUpstreamInventory",
				"TestH264RealVectorPinnedFATEInventory",
				"TestH264RealVectorDocumentationCounts",
				"TestH264RealVectorUpstreamFATECoverage",
				"status: fail (missing focused test",
				"status: fail (focused test skipped)",
			},
		},
		{
			path: "scripts/h264-decoder-fuzz-smoke.sh",
			phrases: []string{
				"go test ./tests -run '^$' -list \"$PATTERN\"",
				"grep -Eq '^Fuzz' \"$list_log\"",
				"status: fail (no matching fuzz targets)",
				"testing: warning: no fuzz tests to fuzz|^--- SKIP: ",
				"status: fail (fuzz target did not run)",
			},
		},
		{
			path: "scripts/h264-benchstat-canary.sh",
			phrases: []string{
				"go test -run '^$' -list \"$PATTERN\" .",
				"grep -Eq '^Benchmark' \"$list_log\"",
				"status: fail (no matching benchmarks)",
				"grep -Eq '^Benchmark' \"$log\"",
				"status: fail (benchmark did not run)",
			},
		},
	} {
		t.Run(tt.path, func(t *testing.T) {
			data, err := os.ReadFile(tt.path)
			if err != nil {
				t.Fatalf("read %s: %v", tt.path, err)
			}
			script := string(data)
			for _, phrase := range tt.phrases {
				if !strings.Contains(script, phrase) {
					t.Fatalf("%s missing empty-run guard phrase %q", tt.path, phrase)
				}
			}
		})
	}
}

func TestQualityEvidenceRunGoTestGateRejectsSkippedFocusedRuns(t *testing.T) {
	for _, path := range []string{
		"scripts/h264-decoder-quality-evidence.sh",
		"scripts/h264-encoder-quality-evidence.sh",
	} {
		t.Run(path, func(t *testing.T) {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}
			body := qualityEvidenceFunctionBody(t, string(data), "run_go_test_gate", "run_exact_go_test_gate")
			for _, phrase := range []string{
				"local log=\"$out_dir/$name.log\"",
				"go test \"$pkg\" -run \"$pattern\" \"$@\" 2>&1 | tee \"$log\"",
				"grep -Eq '^--- SKIP: ' \"$log\"",
				"status: fail (focused test skipped)",
			} {
				if !strings.Contains(body, phrase) {
					t.Fatalf("%s run_go_test_gate missing skipped-run guard phrase %q", path, phrase)
				}
			}
			if forbidden := "run_gate \"$name\" go test \"$pkg\" -run \"$pattern\" \"$@\""; strings.Contains(body, forbidden) {
				t.Fatalf("%s run_go_test_gate should not delegate focused run through run_gate", path)
			}
		})
	}
}

func TestQualityEvidenceDirtyDiagnosticModeIsNotCleanPass(t *testing.T) {
	for _, path := range []string{
		"scripts/h264-quality-evidence.sh",
		"scripts/h264-decoder-quality-evidence.sh",
		"scripts/h264-encoder-quality-evidence.sh",
	} {
		t.Run(path, func(t *testing.T) {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}
			script := string(data)
			for _, phrase := range []string{
				"worktree-clean: pass",
				"worktree-clean: allowed-dirty",
				"git status --short: empty",
			} {
				if !strings.Contains(script, phrase) {
					t.Fatalf("%s missing dirty-diagnostic phrase %q", path, phrase)
				}
			}
			if strings.Contains(script, "fi\nprintf '\\nworktree-clean: pass\\n'") {
				t.Fatalf("%s reports clean pass unconditionally after dirty override branch", path)
			}
		})
	}
}

func qualityEvidenceFunctionBody(t *testing.T, script, name, nextName string) string {
	t.Helper()
	startNeedle := name + "() {"
	start := strings.Index(script, startNeedle)
	if start < 0 {
		t.Fatalf("script missing function %s", name)
	}
	endNeedle := "\n" + nextName + "() {"
	end := strings.Index(script[start:], endNeedle)
	if end < 0 {
		t.Fatalf("script missing function %s after %s", nextName, name)
	}
	return script[start : start+end]
}

func TestDecoderEvidenceDocsNameNewExtradataDuplicateSemantics(t *testing.T) {
	for _, path := range []string{"docs/production-readiness.md", "docs/source-truth.md"} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		text := strings.Join(strings.Fields(string(data)), " ")
		lowerText := strings.ToLower(text)
		for _, phrase := range []string{
			"first-entry duplicate packet side-data semantics",
			"empty, malformed, or overflowed first entries",
		} {
			if !strings.Contains(lowerText, phrase) {
				t.Fatalf("%s missing decoder duplicate side-data evidence phrase %q", path, phrase)
			}
		}
		if !strings.Contains(text, "`NEW_EXTRADATA`") {
			t.Fatalf("%s missing decoder duplicate side-data evidence phrase %q", path, "`NEW_EXTRADATA`")
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
		"Caller-constructed `EncodedFrame` values must set `OutputFormat`",
		"keep RTP\n  packet storage matched to that format",
		"`EncodedFrame.Validate` checks public result shape",
		"frame-level\n  keyframe/IDR metadata",
		"RTP packet-list metadata",
		"payload byte parity against\n  the access-unit NAL list",
		"FU-A fragment start/continuation/end\n  consistency",
		"access-unit/RTP helper\n  methods, `Validate`, or `Clone`",
		"RTP results that lack RTP packets",
		"`EncodedFrame.Data` is retained only as an Annex B access-unit view",
		"`AccessUnitData`",
		"`NALData`",
		"`RTPPackets`",
		"`RTPPacketData`",
		"`RTPPayloadData`",
		"`PacketData`",
		"`PayloadData`",
		"`AppendPacketData`",
		"`AppendPayloadData`",
		"Packet-level helpers",
		"encoder-emitted 12-byte RTP header shape",
		"exported packet metadata",
		"RTP payload view",
		"admitted single-NAL, STAP-A,\n  and FU-A payload syntax",
		"STAP-B, MTAP, FU-B,\n  nested STAP-A units, and FU-A fragments whose reconstructed NAL type is\n  another packetization unit are rejected",
		"`PacketData`, payload helpers",
		"packet clones require",
		"`Payload`",
		"`Data[12:]`",
	} {
		if !strings.Contains(readme, phrase) {
			t.Fatalf("README.md missing RTP data-surface phrase %q", phrase)
		}
	}
}

func TestREADMEEncoderSamplesSeparateAccessUnitAndRTPSurfaces(t *testing.T) {
	data, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatalf("read README.md: %v", err)
	}
	readme := string(data)
	if strings.Contains(readme, "switch out.OutputFormat") {
		t.Fatal("README.md encoder sample should not mix mutually exclusive output formats in one runtime switch")
	}
	for _, phrase := range []string{
		"must(enc.SetOutputFormat(goh264.EncoderOutputAVC)) // queues an IDR boundary",
		"accessUnit, err := out.AccessUnitData()",
		"nal0, err := out.NALData(0)",
		"For RTP output, set the RTP output format before encoding",
		"must(enc.SetOutputFormat(goh264.EncoderOutputRTP))",
		"packet0, err := out.RTPPacketData(0)",
		"payload0, err := out.RTPPayloadData(0)",
	} {
		if !strings.Contains(readme, phrase) {
			t.Fatalf("README.md encoder output samples missing separated surface phrase %q", phrase)
		}
	}
	avcIndex := strings.Index(readme, "must(enc.SetOutputFormat(goh264.EncoderOutputAVC))")
	rtpIndex := strings.Index(readme, "must(enc.SetOutputFormat(goh264.EncoderOutputRTP))")
	if avcIndex < 0 || rtpIndex < 0 || rtpIndex < avcIndex {
		t.Fatalf("README.md RTP output sample should follow the AVC/access-unit sample, indexes avc=%d rtp=%d", avcIndex, rtpIndex)
	}
}

func TestREADMEEncoderAdmittedValuesTableDocumentsUnsupportedKnobs(t *testing.T) {
	data, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatalf("read README.md: %v", err)
	}
	readme := string(data)
	for _, phrase := range []string{
		"Accepted encoder setup values",
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
		"zero scalar QP fields normally select derived defaults",
		"`EncoderConfig.ExplicitQP=true` when QP 0 is an intentional setup value",
		"`SetQP` and pointer QP fields in `EncoderReconfigure` treat zero as an explicit",
		"VBR mode; invalid bitrate ordering",
		"EncoderPresetRealtime",
		"Balanced/Quality presets; only `EncoderPresetRealtime` drives current mode selection",
		"`TimeBaseNum=1`",
		"`TimeBaseDen>0`",
		"`RTPTimestampIncrement>0`, or zero to derive cadence from `TimeBaseDen` and frame rate",
		"zero-derived setup and `SetFrameRate`, automatic timestamps carry fractional\nframe-rate remainders forward",
		"Non-1 time-base numerator",
		"Workers>1` only with `Deterministic=false`",
		"no parallel throughput guarantee",
		"IntraRefresh=false",
		"enabled intra refresh",
		"packetization-mode 0 with payload size >= 2",
		"packetization-mode 1 with payload size >= 3",
		"STAP-A only in mode 1",
		"DON disabled",
		"payload type 1..127, with zero selecting the dynamic default 96",
		"`RTPPayloadType` zero selects the dynamic default 96 during config\nnormalization, `SetRTPMetadata`, and pointer-based `EncoderReconfigure`",
		"use\n1..127 to emit a specific payload type",
		"Annex B and AVC configs normalize `DONDisabled=true`",
		"`SetOutputFormat(EncoderOutputRTP)` uses admitted RTP defaults",
		"direct RTP\nconfigs with `DONDisabled=false` return `ErrUnsupported`",
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
		"supported RTP, Annex B, or AVC template",
		"`DefaultRTPEncoderConfig`, `DefaultAnnexBEncoderConfig`, or `DefaultAVCEncoderConfig`; `DefaultRealtimeEncoderConfig` and `DefaultEncoderConfig` return the RTP template",
		"Read the exact live setup after accepted setters",
		"`Encoder.Config`",
		"Encoder.Config` returns the exact normalized live configuration",
		"`EncoderConfig` owns encoded crop/color metadata",
		"`EncoderFrame.Color` is",
		"validated input metadata",
		"does not rewrite SPS/VUI per frame",
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
		"prior references are not used across the incompatible boundary",
		"IDR-bound stream switches",
		"unrelated stream where the decoder cannot infer the boundary from avcC",
		"call `Reset` before storing the new avcC",
		"PacketSideDataNewExtradata",
		"uses the same stateful update rule",
		"Empty `DecodePacket` or `DecodePacketFrames` calls are flush-only and do not apply `NEW_EXTRADATA` or any other packet side data.",
		"Duplicate packet side data follows first-entry semantics",
		"packet `NEW_EXTRADATA` configuration updates",
		"scalar active-format and S12M timecode values",
		"structured layouts",
		"A53 captions, ICC profile, HDR10+, and LCEVC byte payloads",
	} {
		if !strings.Contains(readme, phrase) {
			t.Fatalf("README.md decoder avcC state guidance missing %q", phrase)
		}
	}
}

func TestREADMEDecoderRawOutputDocumentsOverlapIsolation(t *testing.T) {
	data, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatalf("read README.md: %v", err)
	}
	readme := strings.Join(strings.Fields(string(data)), " ")
	for _, phrase := range []string{
		"`RawYUVBytesLE` returns a caller-owned rawvideo byte buffer",
		"`RawYUV16` returns a caller-owned uint16 sample buffer",
		"`Frame.Validate` checks decoded-frame plane and side-data storage for caller-owned preflight",
		"Raw-output append helpers isolate output when the caller destination overlaps frame plane storage",
	} {
		if !strings.Contains(readme, phrase) {
			t.Fatalf("README.md raw-output ownership docs missing %q", phrase)
		}
	}
}

func TestREADMEStateAndOwnershipDocumentsDecoderEncoderBoundaries(t *testing.T) {
	data, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatalf("read README.md: %v", err)
	}
	readme := string(data)
	for _, phrase := range []string{
		"## State And Ownership Boundaries",
		"`Decoder.DecodeFrames` / `DecodePacketFrames`",
		"Retain decoder references and delayed output across stream packets",
		"Empty-input delayed-output calls through\nsingle-frame helpers consume delayed output only when exactly one frame is\navailable",
		"queued delayed output remains available to `FlushDelayedFrames`",
		"`Decoder.ConfigureAVCC`",
		"resets decoder picture state for the configured-AVC boundary",
		"`Decoder.DecodeAVCCFrames` / packet `NEW_EXTRADATA`",
		"Compatible parameter-set updates retain references",
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
			t.Fatalf("README.md state/ownership table missing %q", phrase)
		}
	}
}

func TestPublicCommentsDocumentStateAndOwnershipBoundaries(t *testing.T) {
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
		"ErrInvalidData reports malformed input or invalid public API arguments",
		"ErrUnsupported reports valid inputs or controls outside the supported",
		"decoder or encoder contract",
		"Errors can wrap this sentinel with additional detail; use errors.Is to",
		"test for it",
		"InspectAnnexBHeaders parses Annex B parameter sets and returns stream\n// metadata without changing decoder state",
		"InspectAVCHeaders parses length-prefixed AVC parameter sets and returns\n// stream metadata without changing decoder state",
		"ParseHeadersAVC parses AVC parameter sets, stores SPS/PPS state and the AVC\n// NAL length size for later DecodeConfiguredAVCFrames calls",
		"ConfigureAVCC parses an avcC record, stores it for configured-AVC decode\n// calls, resets decoder picture state",
		"InspectAVCC parses avcC metadata without changing decoder state",
		"DecodeAVCCFrames updates the stored AVC configuration from an avcC record,\n// decodes data with that configuration, and drains delayed frames",
		"FlushDelayedFrame drains delayed output only when exactly one delayed frame is\n// available and returns that frame",
		"without another decode error; in that case delayed output remains available\n// to FlushDelayedFrames",
	} {
		if !strings.Contains(decoder, phrase) {
			t.Fatalf("decoder public comments missing state/ownership phrase %q", phrase)
		}
	}
	for _, phrase := range []string{
		"Start from DefaultRTPEncoderConfig, DefaultAnnexBEncoderConfig, or\n// DefaultAVCEncoderConfig",
		"DefaultRealtimeEncoderConfig returns a realtime/WebRTC-oriented 8-bit I420",
		"DefaultRTPEncoderConfig returns the realtime/WebRTC RTP template",
		"DefaultAnnexBEncoderConfig returns the realtime 8-bit I420 template with\n// Annex B access-unit output selected",
		"DefaultAVCEncoderConfig returns the realtime 8-bit I420 template with AVC\n// length-prefixed access-unit output selected",
		"DefaultEncoderConfig returns the same realtime template",
		"NewEncoder applies derived defaults and rejects invalid or unsupported\n// controls",
		"Validate reports whether cfg is accepted without returning the\n// normalized values; Normalize returns the exact validated setup",
		"Config returns a copy of the current normalized encoder configuration",
		"Reset clears encoder coding state while preserving configuration and RTP callback",
		"After Reset, the next successfully encoded frame starts a fresh sequence",
	} {
		if !strings.Contains(encoder, phrase) {
			t.Fatalf("encoder public comments missing state/ownership phrase %q", phrase)
		}
	}
}

func TestPublicExamplesUsePreferredDecoderAVCCConfigurationName(t *testing.T) {
	data, err := os.ReadFile("examples_test.go")
	if err != nil {
		t.Fatalf("read examples_test.go: %v", err)
	}
	for _, nonCanonical := range []string{
		statusPhrase("Parse", "AVCC"),
		statusPhrase("Parse", "AVCDecoderConfigurationRecord"),
		statusPhrase("Inspect", "AVCDecoderConfigurationRecord"),
		statusPhrase("Configure", "AVCDecoderConfigurationRecord"),
		statusPhrase("Decode", "AVCWithConfigurationRecord"),
		statusPhrase("Decode", "AVCFramesWithConfigurationRecord"),
	} {
		if strings.Contains(string(data), nonCanonical) {
			t.Fatalf("public examples should use canonical decoder avcC API names instead of %s", nonCanonical)
		}
	}
}

func TestREADMEDecoderAVCCNameMap(t *testing.T) {
	data, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatalf("read README.md: %v", err)
	}
	readme := string(data)
	for _, phrase := range []string{
		"avcC name map",
		"Stateless avcC metadata inspection",
		"`InspectAVCC`",
		"Store avcC for configured-AVC streaming",
		"`ConfigureAVCC`",
		"Decode with already stored avcC",
		"`DecodeConfiguredAVCFrames`",
		"Update avcC, decode one packet, then drain delayed output",
		"`DecodeAVCCFrames`",
		"| Need | Helper | Single-frame helper |",
		"Single-frame helper",
	} {
		if !strings.Contains(readme, phrase) {
			t.Fatalf("README.md decoder avcC name-map table missing %q", phrase)
		}
	}
}

func requireREADMECodeName(t *testing.T, readme string, name string) {
	t.Helper()
	if !strings.Contains(readme, "`"+name+"`") && !strings.Contains(readme, name) {
		t.Fatalf("README.md missing API chooser name %q", name)
	}
}
