// SPDX-License-Identifier: LGPL-2.1-or-later

package goh264

import (
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestREADMEFocusesOnDecoderOnlyPublicSurface(t *testing.T) {
	data, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatalf("read README.md: %v", err)
	}
	readme := string(data)

	decoderType := reflect.TypeOf((*Decoder)(nil))
	packageFunctions := map[string]any{
		"InspectAnnexBHeaders": InspectAnnexBHeaders,
		"InspectAVCHeaders":    InspectAVCHeaders,
		"InspectAVCC":          InspectAVCC,
	}
	for _, name := range []string{
		"DecodeFrames",
		"DecodePacketFrames",
		"DecodeAnnexBFrames",
		"DecodeAVCFrames",
		"DecodeConfiguredAVCFrames",
		"DecodeAVCCFrames",
		"FlushDelayedFrames",
		"InspectAnnexBHeaders",
		"InspectAVCHeaders",
		"InspectAVCC",
		"ConfigureAVCC",
		"Frame",
		"Packet",
		"FrameSideData",
	} {
		if fn, ok := packageFunctions[name]; ok {
			if reflect.TypeOf(fn).Kind() != reflect.Func {
				t.Fatalf("README names non-function package entry %s", name)
			}
		} else if name != "Frame" && name != "Packet" && name != "FrameSideData" {
			if _, ok := decoderType.MethodByName(name); !ok {
				t.Fatalf("README names missing Decoder.%s", name)
			}
		}
		if !strings.Contains(readme, "`"+name+"`") {
			t.Fatalf("README missing public decoder name %s", name)
		}
	}

	for _, stale := range []string{
		"`New" + "Enc" + "oder`",
		"`Enc" + "oder" + "Config`",
		"`Encode`",
		"`Default" + "Enc" + "oder" + "Config`",
		"WebRTC " + "enc" + "oder",
	} {
		if strings.Contains(readme, stale) {
			t.Fatalf("README still references removed write-side surface %q", stale)
		}
	}

	if !strings.Contains(readme, "decoder-only") || !strings.Contains(readme, "PATENTS.md") {
		t.Fatal("README should state decoder-only scope and point to PATENTS.md")
	}
}
