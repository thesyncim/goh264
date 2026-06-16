// SPDX-License-Identifier: LGPL-2.1-or-later

package goh264

import (
	"errors"
	"os"
	"testing"

	"github.com/thesyncim/goh264/internal/h264"
)

func TestParseHeadersDoesNotCommitPartialStateOnError(t *testing.T) {
	sps := decoderInternalTestNAL(t, h264.NALSPS)

	dec := NewDecoder()
	_, err := dec.parseHeaders([]h264.NALUnit{
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
	data := decoderInternalTestAnnexB(t)
	dec := NewDecoder()
	for i := 0; i < 3; i++ {
		if _, err := dec.ParseHeadersAnnexB(data); err != nil {
			t.Fatalf("ParseHeadersAnnexB iteration %d: %v", i, err)
		}
		if len(dec.slices) != 0 {
			t.Fatalf("ParseHeadersAnnexB iteration %d retained %d slice headers, want 0", i, len(dec.slices))
		}
	}
}

func decoderInternalTestNAL(t *testing.T, typ h264.NALUnitType) h264.NALUnit {
	t.Helper()
	nals, err := h264.SplitAnnexB(decoderInternalTestAnnexB(t))
	if err != nil {
		t.Fatalf("SplitAnnexB: %v", err)
	}
	for _, nal := range nals {
		if nal.Type == typ {
			return nal
		}
	}
	t.Fatalf("test vector missing NAL type %v", typ)
	return h264.NALUnit{}
}

func decoderInternalTestAnnexB(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile("testdata/h264/high10_inter_cavlc_idrp.h264")
	if err != nil {
		t.Fatalf("read test vector: %v", err)
	}
	return data
}
