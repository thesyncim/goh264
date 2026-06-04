// SPDX-License-Identifier: LGPL-2.1-or-later

package goh264_test

import (
	"testing"

	"github.com/thesyncim/goh264/internal/h264"
)

func TestParseHeadersAnnexBRecoversUnescapedSPSPayload(t *testing.T) {
	data := annexBSPSFixture([]byte{
		0x67, 0x4d, 0x00, 0x28, 0x9e, 0x21, 0x02, 0x82,
		0xf4, 0x20, 0x00, 0x00, 0x00, 0x20, 0x00, 0x00,
		0x03, 0x30, 0x80,
	})
	assertStrictSPSRejects(t, data)

	info, err := NewDecoder().ParseHeadersAnnexB(data)
	if err != nil {
		t.Fatalf("ParseHeadersAnnexB: %v", err)
	}
	if info.SPSID != 0 || info.ProfileIDC != 77 || info.LevelIDC != 40 {
		t.Fatalf("SPS id/profile/level = %d/%d/%d, want 0/77/40", info.SPSID, info.ProfileIDC, info.LevelIDC)
	}
}

func TestParseHeadersAnnexBToleratesTruncatedVUI(t *testing.T) {
	data := annexBSPSFixture([]byte{
		0x67, 0x42, 0x00, 0x1e, 0xab, 0x40, 0xa0, 0xfd,
		0x80, 0x28, 0x30, 0x0d, 0xf8, 0xc7, 0x18, 0x06,
		0xfc, 0x63, 0x8f, 0x68, 0x48, 0x9a,
	})
	assertStrictSPSRejects(t, data)

	info, err := NewDecoder().ParseHeadersAnnexB(data)
	if err != nil {
		t.Fatalf("ParseHeadersAnnexB: %v", err)
	}
	if info.SPSID != 0 || info.ProfileIDC != 66 || info.LevelIDC != 30 {
		t.Fatalf("SPS id/profile/level = %d/%d/%d, want 0/66/30", info.SPSID, info.ProfileIDC, info.LevelIDC)
	}
	if info.Width != 320 || info.Height != 240 {
		t.Fatalf("SPS size = %dx%d, want 320x240", info.Width, info.Height)
	}
}

func annexBSPSFixture(raw []byte) []byte {
	data := []byte{0x00, 0x00, 0x00, 0x01}
	return append(data, raw...)
}

func assertStrictSPSRejects(t *testing.T, data []byte) {
	t.Helper()
	nals, err := h264.SplitAnnexB(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(nals) != 1 {
		t.Fatalf("NAL count = %d, want one SPS", len(nals))
	}
	if nals[0].Type != h264.NALSPS {
		t.Fatalf("NAL type = %v, want SPS", nals[0].Type)
	}
	if _, err := h264.DecodeSPS(nals[0].RBSP); err == nil {
		t.Fatal("DecodeSPS unexpectedly accepted the strict RBSP")
	}
}
