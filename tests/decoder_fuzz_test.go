// SPDX-License-Identifier: LGPL-2.1-or-later

package goh264_test

import (
	"encoding/hex"
	"strings"
	"testing"

	"github.com/thesyncim/goh264/internal/h264"
)

func TestDecodeMalformedPublicSurfacesNoPanic(t *testing.T) {
	validAnnexB := decodeHexFixture(t, black16AnnexBHex)
	validAVC := annexBToAVC(t, validAnnexB, 4)
	validConfig, validSamples := annexBToAVCConfigAndSamples(t, validAnnexB, 4)
	bFrameAnnexB := decodeHexFixture(t, testsrc32CAVLCBFramesAnnexBHex)
	bFrameConfig, bFrameSamples := annexBToAVCConfigAndSamples(t, bFrameAnnexB, 4)

	cases := []struct {
		name string
		data []byte
		aux  []byte
	}{
		{name: "no start code", data: []byte{0x12, 0x34, 0x56, 0x78}},
		{name: "annexb start only", data: []byte{0x00, 0x00, 0x01}},
		{name: "annexb forbidden nal bit", data: []byte{0x00, 0x00, 0x01, 0x80}},
		{name: "annexb empty sps", data: []byte{0x00, 0x00, 0x01, 0x67}},
		{name: "avc oversized length", data: []byte{0x00, 0x00, 0x00, 0x05, 0x65}, aux: validConfig},
		{name: "truncated avc", data: validAVC[:len(validAVC)-1], aux: validConfig},
		{name: "truncated configured sample", data: validSamples[0][:len(validSamples[0])/2], aux: validConfig},
		{name: "truncated config", data: validSamples[0], aux: validConfig[:len(validConfig)-1]},
		{name: "b-frame reordered stream", data: bFrameAnnexB},
		{name: "b-frame truncated middle sample", data: bFrameSamples[1][:len(bFrameSamples[1])/2], aux: bFrameConfig},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assertDecodePublicSurfacesNoPanic(t, tt.data, tt.aux)
		})
	}
}

func FuzzDecodePublicSurfacesNoPanic(f *testing.F) {
	validAnnexB := decodeHexFixtureForFuzz(f, black16AnnexBHex)
	validAVC := annexBToAVCForFuzz(f, validAnnexB, 4)
	validConfig, validPacket := annexBToAVCConfigAndPacketForFuzz(f, validAnnexB, 4)
	bFrameAnnexB := decodeHexFixtureForFuzz(f, testsrc32CAVLCBFramesAnnexBHex)
	bFrameConfig, bFramePacket := annexBToAVCConfigAndPacketForFuzz(f, bFrameAnnexB, 4)

	f.Add([]byte{}, []byte{})
	f.Add([]byte{0x00, 0x00, 0x01}, []byte{})
	f.Add([]byte{0x00, 0x00, 0x01, 0x80}, []byte{})
	f.Add(validAnnexB, []byte{})
	f.Add(validAVC, validConfig)
	f.Add(validPacket, validConfig)
	f.Add(validPacket[:len(validPacket)/2], validConfig)
	f.Add(validPacket, validConfig[:len(validConfig)-1])
	f.Add(bFrameAnnexB, []byte{})
	f.Add(bFramePacket, bFrameConfig)
	f.Add(bFramePacket[:len(bFramePacket)/2], bFrameConfig)

	f.Fuzz(func(t *testing.T, data []byte, aux []byte) {
		if len(data) > 4096 || len(aux) > 4096 {
			t.Skip("bounded public-surface fuzz smoke")
		}
		assertDecodePublicSurfacesNoPanic(t, data, aux)
	})
}

func assertDecodePublicSurfacesNoPanic(t testing.TB, data []byte, aux []byte) {
	t.Helper()
	assertNoPanic(t, "DecodeFrames", func() {
		_, _ = NewDecoder().DecodeFrames(data)
	})
	assertNoPanic(t, "DecodeAnnexBFrames", func() {
		_, _ = NewDecoder().DecodeAnnexBFrames(data)
	})
	assertNoPanic(t, "ParseHeadersAnnexB", func() {
		_, _ = NewDecoder().ParseHeadersAnnexB(data)
	})
	for _, nalLengthSize := range []int{1, 2, 3, 4} {
		assertNoPanic(t, "DecodeAVCFrames", func() {
			_, _ = NewDecoder().DecodeAVCFrames(data, nalLengthSize)
		})
	}
	assertNoPanic(t, "DecodeAVCCFrames", func() {
		_, _ = NewDecoder().DecodeAVCCFrames(aux, data)
	})
	assertNoPanic(t, "DecodeConfiguredAVCFrames", func() {
		dec := NewDecoder()
		if _, err := dec.ConfigureAVCC(aux); err == nil {
			_, _ = dec.DecodeConfiguredAVCFrames(data)
			_, _ = dec.FlushDelayedFrames()
		}
	})
	assertNoPanic(t, "DecodePacketFrames", func() {
		_, _ = NewDecoder().DecodePacketFrames(Packet{
			Data:     data,
			SideData: decoderFuzzPacketSideData(aux),
		})
	})
}

func assertNoPanic(t testing.TB, name string, fn func()) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("%s panicked: %v", name, r)
		}
	}()
	fn()
}

func decoderFuzzPacketSideData(data []byte) []PacketSideData {
	out := []PacketSideData{
		{Type: PacketSideDataDisplayMatrix, Data: data},
		{Type: PacketSideDataStereo3D, Data: data},
		{Type: PacketSideDataMasteringDisplayMetadata, Data: data},
		{Type: PacketSideDataSpherical, Data: data},
		{Type: PacketSideDataContentLightLevel, Data: data},
		{Type: PacketSideDataA53ClosedCaptions, Data: data},
		{Type: PacketSideDataActiveFormat, Data: data},
		{Type: PacketSideDataICCProfile, Data: data},
		{Type: PacketSideDataS12MTimecode, Data: data},
		{Type: PacketSideDataDynamicHDR10Plus, Data: data},
		{Type: PacketSideDataAmbientViewingEnvironment, Data: data},
		{Type: PacketSideDataLCEVC, Data: data},
		{Type: PacketSideData3DReferenceDisplays, Data: data},
	}
	if len(data) != 0 {
		out = append([]PacketSideData{{Type: PacketSideDataNewExtradata, Data: data}}, out...)
	}
	return out
}

func decodeHexFixtureForFuzz(f *testing.F, s string) []byte {
	f.Helper()
	clean := strings.NewReplacer("\n", "", "\t", "", " ", "").Replace(s)
	data, err := hex.DecodeString(clean)
	if err != nil {
		f.Fatal(err)
	}
	return data
}

func annexBToAVCForFuzz(f *testing.F, data []byte, nalLengthSize int) []byte {
	f.Helper()
	nals, err := h264.SplitAnnexB(data)
	if err != nil {
		f.Fatal(err)
	}
	var out []byte
	for _, nal := range nals {
		out = appendAVCNALUnitForFuzz(f, out, nal.Raw, nalLengthSize)
	}
	return out
}

func annexBToAVCConfigAndPacketForFuzz(f *testing.F, data []byte, nalLengthSize int) ([]byte, []byte) {
	f.Helper()
	nals, err := h264.SplitAnnexB(data)
	if err != nil {
		f.Fatal(err)
	}
	var spsNals [][]byte
	var ppsNals [][]byte
	var packet []byte
	for _, nal := range nals {
		switch nal.Type {
		case h264.NALSPS:
			spsNals = append(spsNals, nal.Raw)
		case h264.NALPPS:
			ppsNals = append(ppsNals, nal.Raw)
		default:
			packet = appendAVCNALUnitForFuzz(f, packet, nal.Raw, nalLengthSize)
		}
	}
	if len(spsNals) == 0 || len(ppsNals) == 0 || len(packet) == 0 {
		f.Fatalf("fuzz config split produced sps=%d pps=%d packet=%d", len(spsNals), len(ppsNals), len(packet))
	}
	if len(spsNals[0]) < 4 {
		f.Fatalf("short SPS NAL: %x", spsNals[0])
	}

	config := []byte{
		1,
		spsNals[0][1],
		spsNals[0][2],
		spsNals[0][3],
		0xfc | byte(nalLengthSize-1),
		0xe0 | byte(len(spsNals)),
	}
	for _, raw := range spsNals {
		config = appendAVCConfigNALUnitForFuzz(f, config, raw)
	}
	config = append(config, byte(len(ppsNals)))
	for _, raw := range ppsNals {
		config = appendAVCConfigNALUnitForFuzz(f, config, raw)
	}
	return config, packet
}

func appendAVCNALUnitForFuzz(f *testing.F, dst []byte, raw []byte, nalLengthSize int) []byte {
	f.Helper()
	if nalLengthSize < 1 || nalLengthSize > 4 {
		f.Fatalf("invalid nalLengthSize %d", nalLengthSize)
	}
	maxSize := uint64(1)<<(uint(nalLengthSize)*8) - 1
	size := len(raw)
	if size == 0 || uint64(size) > maxSize {
		f.Fatalf("NAL size %d exceeds %d-byte length field", size, nalLengthSize)
	}
	for shift := (nalLengthSize - 1) * 8; shift >= 0; shift -= 8 {
		dst = append(dst, byte(size>>shift))
	}
	return append(dst, raw...)
}

func appendAVCConfigNALUnitForFuzz(f *testing.F, dst []byte, raw []byte) []byte {
	f.Helper()
	if len(raw) == 0 || len(raw) > 0xffff {
		f.Fatalf("bad config NAL size %d", len(raw))
	}
	dst = append(dst, byte(len(raw)>>8), byte(len(raw)))
	return append(dst, raw...)
}
