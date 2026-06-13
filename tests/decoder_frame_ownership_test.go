// SPDX-License-Identifier: LGPL-2.1-or-later

package goh264_test

import "testing"

func TestDecodeConfiguredAVCFramesOwnPublic8BitPlanes(t *testing.T) {
	data := decodeHexFixture(t, black16IPAnnexBHex)
	config, samples := annexBToAVCConfigAndSamples(t, data, 4)
	if len(samples) != 2 {
		t.Fatalf("samples = %d, want 2", len(samples))
	}

	dec := NewDecoder()
	if _, err := dec.ConfigureAVCC(config); err != nil {
		t.Fatal(err)
	}
	first, err := dec.DecodeConfiguredAVC(samples[0])
	if err != nil {
		t.Fatalf("first sample: %v", err)
	}
	fillPublicFrame8(first)

	second, err := dec.DecodeConfiguredAVC(samples[1])
	if err != nil {
		t.Fatalf("second sample: %v", err)
	}
	assertFrameMD5Strings(t, []*Frame{second}, []string{"8aaefe0adcea094cfb5161a060bab4e2"})
}

func TestDecodeConfiguredAVCFrameSurvivesLater8BitDecode(t *testing.T) {
	data := decodeHexFixture(t, black16IPAnnexBHex)
	config, samples := annexBToAVCConfigAndSamples(t, data, 4)
	if len(samples) != 2 {
		t.Fatalf("samples = %d, want 2", len(samples))
	}

	dec := NewDecoder()
	if _, err := dec.ConfigureAVCC(config); err != nil {
		t.Fatal(err)
	}
	first, err := dec.DecodeConfiguredAVC(samples[0])
	if err != nil {
		t.Fatalf("first sample: %v", err)
	}
	assertFrameMD5Strings(t, []*Frame{first}, []string{"8aaefe0adcea094cfb5161a060bab4e2"})

	second, err := dec.DecodeConfiguredAVC(samples[1])
	if err != nil {
		t.Fatalf("second sample: %v", err)
	}
	fillPublicFrame8(second)
	assertFrameMD5Strings(t, []*Frame{first}, []string{"8aaefe0adcea094cfb5161a060bab4e2"})
}

func TestDecodeConfiguredAVCFramesOwnPublicHigh10Planes(t *testing.T) {
	data := decodeHexFixture(t, gray16High10CAVLCPSkipAnnexBHex)
	config, samples := annexBToAVCConfigAndSamples(t, data, 4)
	if len(samples) != 2 {
		t.Fatalf("samples = %d, want 2", len(samples))
	}

	dec := NewDecoder()
	if _, err := dec.ConfigureAVCC(config); err != nil {
		t.Fatal(err)
	}
	first, err := dec.DecodeConfiguredAVC(samples[0])
	if err != nil {
		t.Fatalf("first sample: %v", err)
	}
	fillPublicFrameHigh(first)

	second, err := dec.DecodeConfiguredAVC(samples[1])
	if err != nil {
		t.Fatalf("second sample: %v", err)
	}
	assertHigh10FrameMD5Strings(t, []*Frame{second}, []string{"87e217773d3e8b548fdf2002955cfcb9"})
}

func TestDecodeConfiguredAVCFrameSurvivesLaterHigh10Decode(t *testing.T) {
	data := decodeHexFixture(t, gray16High10CAVLCPSkipAnnexBHex)
	config, samples := annexBToAVCConfigAndSamples(t, data, 4)
	if len(samples) != 2 {
		t.Fatalf("samples = %d, want 2", len(samples))
	}

	dec := NewDecoder()
	if _, err := dec.ConfigureAVCC(config); err != nil {
		t.Fatal(err)
	}
	first, err := dec.DecodeConfiguredAVC(samples[0])
	if err != nil {
		t.Fatalf("first sample: %v", err)
	}
	assertHigh10FrameMD5Strings(t, []*Frame{first}, []string{"87e217773d3e8b548fdf2002955cfcb9"})

	second, err := dec.DecodeConfiguredAVC(samples[1])
	if err != nil {
		t.Fatalf("second sample: %v", err)
	}
	fillPublicFrameHigh(second)
	assertHigh10FrameMD5Strings(t, []*Frame{first}, []string{"87e217773d3e8b548fdf2002955cfcb9"})
}

func fillPublicFrame8(frame *Frame) {
	for i := range frame.Y {
		frame.Y[i] = 0xff
	}
	for i := range frame.Cb {
		frame.Cb[i] = 0
	}
	for i := range frame.Cr {
		frame.Cr[i] = 0xff
	}
}

func fillPublicFrameHigh(frame *Frame) {
	maxLuma := uint16((1 << uint(frame.BitDepthLuma)) - 1)
	maxChroma := uint16((1 << uint(frame.BitDepthChroma)) - 1)
	for i := range frame.Y16 {
		frame.Y16[i] = maxLuma
	}
	for i := range frame.Cb16 {
		frame.Cb16[i] = 0
	}
	for i := range frame.Cr16 {
		frame.Cr16[i] = maxChroma
	}
}
