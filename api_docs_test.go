// SPDX-License-Identifier: LGPL-2.1-or-later

package goh264

import (
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestREADMEFocusesOnDecoderOnlyPublicSurface(t *testing.T) {
	readme := readProjectText(t, "README.md")

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

func TestProjectTextAvoidsToolAttribution(t *testing.T) {
	for _, path := range []string{
		"README.md",
		"PATENTS.md",
		"doc.go",
		"docs/production-readiness.md",
		"docs/source-truth.md",
		"docs/translation-ledger.md",
	} {
		text := strings.ToLower(readProjectText(t, path))
		if strings.Contains(text, "co"+"dex") {
			t.Fatalf("%s should not mention implementation tooling", path)
		}
	}
}

func TestPatentNoticeKeepsResponsibilityBoundary(t *testing.T) {
	readme := readProjectText(t, "README.md")
	patents := readProjectText(t, "PATENTS.md")

	for _, snippet := range []string{
		"decoder-only",
		"does not grant patent rights",
		"users and distributors are responsible",
		"PATENTS.md",
	} {
		if !strings.Contains(readme, snippet) {
			t.Fatalf("README missing patent-boundary snippet %q", snippet)
		}
	}
	for _, snippet := range []string{
		"not legal advice",
		"does not grant any patent license",
		"Decoder-only scope is an implementation boundary",
		"You are responsible",
	} {
		if !strings.Contains(patents, snippet) {
			t.Fatalf("PATENTS.md missing responsibility snippet %q", snippet)
		}
	}
}

func readProjectText(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}
