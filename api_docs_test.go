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

	encoderType := reflect.TypeOf((*Encoder)(nil))
	for _, name := range []string{
		"ValidateFrame",
		"Encode",
		"EncodeInto",
		"SetBitrate",
		"SetQP",
		"SetOutputFormat",
		"SetRTPMetadata",
		"Reconfigure",
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
	} {
		requireREADMECodeName(t, readme, name)
	}
}

func requireREADMECodeName(t *testing.T, readme string, name string) {
	t.Helper()
	if !strings.Contains(readme, "`"+name+"`") && !strings.Contains(readme, name) {
		t.Fatalf("README.md missing API chooser name %q", name)
	}
}
