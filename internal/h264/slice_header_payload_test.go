// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

import "testing"

func TestParseSliceHeaderWithPayloadKeepsMacroblockBitPosition(t *testing.T) {
	sps := &SPS{
		SPSID:            0,
		BitDepthLuma:     8,
		BitDepthChroma:   8,
		ChromaFormatIDC:  1,
		Log2MaxFrameNum:  4,
		PocType:          2,
		FrameMBSOnlyFlag: 1,
	}
	pps := cavlcFlatQMulPPS()
	pps.PPSID = 0
	pps.SPS = sps
	pps.InitQP = 26
	var ppsList [maxPPSCount]*PPS
	ppsList[0] = pps

	rbsp := rbspBytesFromBits("1 011 1 0011 1 1010 1")
	nal := NALUnit{
		RefIDC: 0,
		Type:   NALSlice,
		RBSP:   rbsp,
	}

	sh, payload, err := parseSliceHeaderWithPayload(nal, &ppsList)
	if err != nil {
		t.Fatalf("parse slice header with payload failed: %v", err)
	}
	if sh.FirstMBAddr != 0 || sh.SliceTypeNoS != PictureTypeI || sh.PPSID != 0 || sh.FrameNum != 3 || sh.QScale != 26 {
		t.Fatalf("slice header = first %d type %d pps %d frame %d q %d", sh.FirstMBAddr, sh.SliceTypeNoS, sh.PPSID, sh.FrameNum, sh.QScale)
	}
	if payload.bitsLeft() != 4 {
		t.Fatalf("payload bits left = %d, want 4", payload.bitsLeft())
	}
	got, err := payload.showBits(4)
	if err != nil {
		t.Fatal(err)
	}
	if got != 0b1010 {
		t.Fatalf("payload bits = %04b, want 1010", got)
	}

	shOnly, err := ParseSliceHeader(nal, &ppsList)
	if err != nil {
		t.Fatalf("parse slice header wrapper failed: %v", err)
	}
	if shOnly.FrameNum != sh.FrameNum || shOnly.QScale != sh.QScale {
		t.Fatalf("wrapper header frame/q = %d/%d, want %d/%d", shOnly.FrameNum, shOnly.QScale, sh.FrameNum, sh.QScale)
	}
}

func rbspBytesFromBits(bits string) []byte {
	var clean []byte
	for _, r := range bits {
		switch r {
		case '0', '1':
			clean = append(clean, byte(r))
		case ' ', '_', '\n', '\t':
		default:
			panic("invalid bit string")
		}
	}
	out := make([]byte, (len(clean)+7)/8)
	for i, bit := range clean {
		if bit == '1' {
			out[i>>3] |= 1 << uint(7-(i&7))
		}
	}
	return out
}
