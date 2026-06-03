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

func TestSplitAVCC(t *testing.T) {
	rawNals := [][]byte{
		{0x67, 0xaa, 0x00, 0x00, 0x03, 0x01},
		{0x68, 0xbb},
	}

	for _, nalLengthSize := range []int{1, 2, 3, 4} {
		var data []byte
		for _, raw := range rawNals {
			size := len(raw)
			for shift := (nalLengthSize - 1) * 8; shift >= 0; shift -= 8 {
				data = append(data, byte(size>>shift))
			}
			data = append(data, raw...)
		}

		nals, err := SplitAVCC(data, nalLengthSize)
		if err != nil {
			t.Fatalf("nalLengthSize=%d: %v", nalLengthSize, err)
		}
		if len(nals) != 2 {
			t.Fatalf("nalLengthSize=%d: got %d NALs, want 2", nalLengthSize, len(nals))
		}
		if nals[0].Type != NALSPS || nals[1].Type != NALPPS {
			t.Fatalf("nalLengthSize=%d: types = %v, %v", nalLengthSize, nals[0].Type, nals[1].Type)
		}
		if !bytes.Equal(nals[0].Raw, rawNals[0]) || !bytes.Equal(nals[1].Raw, rawNals[1]) {
			t.Fatalf("nalLengthSize=%d: raw = %x / %x", nalLengthSize, nals[0].Raw, nals[1].Raw)
		}
		if !bytes.Equal(nals[0].RBSP, []byte{0xaa, 0x00, 0x00, 0x01}) {
			t.Fatalf("nalLengthSize=%d: rbsp = %x", nalLengthSize, nals[0].RBSP)
		}
	}
}

func TestSplitAutoPacket(t *testing.T) {
	annexB := []byte{
		0x00, 0x00, 0x00, 0x01, 0x67, 0xaa, 0x00, 0x00, 0x03, 0x01,
		0x00, 0x00, 0x01, 0x68, 0xbb,
	}
	avc := []byte{
		0x00, 0x00, 0x00, 0x06, 0x67, 0xaa, 0x00, 0x00, 0x03, 0x01,
		0x00, 0x00, 0x00, 0x02, 0x68, 0xbb,
	}

	nals, format, err := SplitAutoPacket(annexB, 4)
	if err != nil {
		t.Fatalf("annexb configured length4: %v", err)
	}
	if format != H264PacketFormatAnnexB || len(nals) != 2 || nals[0].Type != NALSPS || nals[1].Type != NALPPS {
		t.Fatalf("annexb format/nals = %d/%v", format, nals)
	}

	nals, format, err = SplitAutoPacket(avc, 4)
	if err != nil {
		t.Fatalf("avc configured length4: %v", err)
	}
	if format != H264PacketFormatAVC || len(nals) != 2 || nals[0].Type != NALSPS || nals[1].Type != NALPPS {
		t.Fatalf("avc format/nals = %d/%v", format, nals)
	}

	nals, format, err = SplitAutoPacket(avc, 0)
	if err != nil {
		t.Fatalf("avc auto length4: %v", err)
	}
	if format != H264PacketFormatAVC || len(nals) != 2 {
		t.Fatalf("auto avc format/nals = %d/%d", format, len(nals))
	}

	avc2 := []byte{0x00, 0x02, 0x67, 0xaa, 0x00, 0x02, 0x68, 0xbb}
	nals, format, err = SplitAutoPacket(avc2, 2)
	if err != nil {
		t.Fatalf("avc configured length2: %v", err)
	}
	if format != H264PacketFormatAVC || len(nals) != 2 {
		t.Fatalf("configured avc2 format/nals = %d/%d", format, len(nals))
	}

	if _, _, err := SplitAutoPacket(avc2, 5); err == nil {
		t.Fatal("expected invalid configured length")
	}
}

func TestSplitAVCCRejectsInvalidSize(t *testing.T) {
	for _, tt := range []struct {
		name          string
		data          []byte
		nalLengthSize int
	}{
		{name: "empty", data: nil, nalLengthSize: 4},
		{name: "zero length", data: []byte{0, 0, 0, 0}, nalLengthSize: 4},
		{name: "missing payload", data: []byte{1}, nalLengthSize: 1},
		{name: "oversized", data: []byte{2, 0x67}, nalLengthSize: 1},
		{name: "bad length size", data: []byte{1, 0x67}, nalLengthSize: 0},
		{name: "bad forbidden bit", data: []byte{1, 0x80}, nalLengthSize: 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := SplitAVCC(tt.data, tt.nalLengthSize); err == nil {
				t.Fatal("expected invalid data")
			}
		})
	}
}

func TestAppendRBSPRejectsUnescapedStartCode(t *testing.T) {
	_, err := AppendRBSP(nil, []byte{0x12, 0x00, 0x00, 0x01, 0x34})
	if err == nil {
		t.Fatal("expected invalid data")
	}
}

func TestSplitAnnexBAllowsTrailingZeroBytes(t *testing.T) {
	data := []byte{
		0x00, 0x00, 0x00, 0x01,
		0x67, 0xaa, 0x80,
		0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x01,
		0x68, 0xbb, 0x80,
	}

	nals, err := SplitAnnexB(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(nals) != 2 {
		t.Fatalf("got %d NALs, want 2", len(nals))
	}
	if !bytes.Equal(nals[0].RBSP, []byte{0xaa, 0x80, 0x00, 0x00, 0x00}) {
		t.Fatalf("rbsp = %x", nals[0].RBSP)
	}
}
