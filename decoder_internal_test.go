// SPDX-License-Identifier: LGPL-2.1-or-later

package goh264

import (
	"errors"
	"testing"

	"github.com/thesyncim/goh264/internal/h264"
)

func TestParseHeadersDoesNotCommitPartialStateOnError(t *testing.T) {
	enc, err := NewEncoder(DefaultEncoderConfig(16, 16))
	if err != nil {
		t.Fatalf("NewEncoder: %v", err)
	}
	sets, err := enc.ParameterSets()
	if err != nil {
		t.Fatalf("ParameterSets: %v", err)
	}
	nals, err := h264.SplitAnnexB(sets.AnnexB)
	if err != nil {
		t.Fatalf("SplitAnnexB parameter sets: %v", err)
	}
	var sps h264.NALUnit
	for _, nal := range nals {
		if nal.Type == h264.NALSPS {
			sps = nal
			break
		}
	}
	if sps.Type != h264.NALSPS {
		t.Fatal("generated parameter sets did not include SPS")
	}

	dec := NewDecoder()
	_, err = dec.parseHeaders([]h264.NALUnit{
		sps,
		{Type: h264.NALPPS, Raw: []byte{byte(h264.NALPPS)}, RBSP: nil},
	})
	if !errors.Is(err, ErrInvalidData) {
		t.Fatalf("parseHeaders partial SPS plus malformed PPS error = %v, want ErrInvalidData", err)
	}
	for i, got := range dec.sps {
		if got != nil {
			t.Fatalf("parseHeaders committed SPS[%d] after malformed PPS", i)
		}
	}
	for i, got := range dec.pps {
		if got != nil {
			t.Fatalf("parseHeaders committed PPS[%d] after malformed PPS", i)
		}
	}
	if len(dec.slices) != 0 {
		t.Fatalf("parseHeaders committed %d slices after malformed PPS, want 0", len(dec.slices))
	}
}

func TestParseHeadersDoesNotRetainSliceHeaders(t *testing.T) {
	enc, err := NewEncoder(DefaultEncoderConfig(16, 16))
	if err != nil {
		t.Fatalf("NewEncoder: %v", err)
	}
	frame := EncoderFrame{
		Width:    16,
		Height:   16,
		StrideY:  16,
		StrideCb: 8,
		StrideCr: 8,
		Y:        make([]byte, 16*16),
		Cb:       make([]byte, 8*8),
		Cr:       make([]byte, 8*8),
	}
	out, err := enc.Encode(frame)
	if err != nil {
		t.Fatalf("Encode IDR: %v", err)
	}
	dec := NewDecoder()
	for i := 0; i < 3; i++ {
		if _, err := dec.ParseHeadersAnnexB(out.Data); err != nil {
			t.Fatalf("ParseHeadersAnnexB iteration %d: %v", i, err)
		}
		if len(dec.slices) != 0 {
			t.Fatalf("ParseHeadersAnnexB iteration %d retained %d slice headers, want 0", i, len(dec.slices))
		}
	}
}
