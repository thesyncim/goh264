// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

import (
	"bytes"
	"testing"
)

func TestSplitAnnexB(t *testing.T) {
	data := []byte{
		0x00, 0x00, 0x00, 0x01, 0x67, 0xaa, 0x00, 0x00, 0x03, 0x01,
		0x00, 0x00, 0x01, 0x68, 0xbb,
	}

	nals, err := SplitAnnexB(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(nals) != 2 {
		t.Fatalf("got %d NALs, want 2", len(nals))
	}
	if nals[0].Type != NALSPS || nals[1].Type != NALPPS {
		t.Fatalf("types = %v, %v", nals[0].Type, nals[1].Type)
	}
	if !bytes.Equal(nals[0].RBSP, []byte{0xaa, 0x00, 0x00, 0x01}) {
		t.Fatalf("rbsp = %x", nals[0].RBSP)
	}
}

func TestAppendRBSPRejectsUnescapedStartCode(t *testing.T) {
	_, err := AppendRBSP(nil, []byte{0x12, 0x00, 0x00, 0x01, 0x34})
	if err == nil {
		t.Fatal("expected invalid data")
	}
}
