// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

import "testing"

func TestDecodeSPSFromNALRetriesRawPayload(t *testing.T) {
	raw := []byte{
		0x67, 0x4d, 0x00, 0x28, 0x9e, 0x21, 0x02, 0x82,
		0xf4, 0x20, 0x00, 0x00, 0x00, 0x20, 0x00, 0x00,
		0x03, 0x30, 0x80,
	}
	nal, err := parseNAL(raw)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DecodeSPS(nal.RBSP); err == nil {
		t.Fatal("DecodeSPS unexpectedly accepted the unescaped RBSP")
	}
	sps, err := decodeSPSFromNAL(nal)
	if err != nil {
		t.Fatalf("decodeSPSFromNAL: %v", err)
	}
	if sps.SPSID != 0 || sps.ProfileIDC != 77 || sps.LevelIDC != 40 {
		t.Fatalf("SPS id/profile/level = %d/%d/%d, want 0/77/40", sps.SPSID, sps.ProfileIDC, sps.LevelIDC)
	}
}
