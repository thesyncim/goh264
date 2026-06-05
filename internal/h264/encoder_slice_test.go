// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

import "testing"

func TestBuildEncoderI420IntraPCMIDRSliceWritesParseableHeader(t *testing.T) {
	sets, err := BuildEncoderParameterSets(EncoderParameterSetConfig{
		ProfileIDC:         66,
		ConstraintSetFlags: 0x03,
		LevelIDC:           31,
		Width:              18,
		Height:             18,
		FrameRateNum:       30,
		FrameRateDen:       1,
		MaxReferenceFrames: 1,
		InitialQP:          23,
	})
	if err != nil {
		t.Fatal(err)
	}
	frame := encoderSliceTestI420(18, 18)
	slice, err := BuildEncoderI420IntraPCMIDRSlice(EncoderI420IntraPCMIDRConfig{
		Width:                      18,
		Height:                     18,
		StrideY:                    18,
		StrideCb:                   9,
		StrideCr:                   9,
		Y:                          frame.y,
		Cb:                         frame.cb,
		Cr:                         frame.cr,
		FrameNum:                   5,
		IDRPicID:                   4,
		InitialQP:                  23,
		DisableDeblockingFilterIDC: 1,
		NALLengthSize:              4,
	})
	if err != nil {
		t.Fatal(err)
	}

	nals, err := SplitAnnexB(append(append([]byte(nil), sets.AnnexB...), slice.AnnexB...))
	if err != nil {
		t.Fatal(err)
	}
	if len(nals) != 3 || nals[2].Type != NALIDRSlice {
		t.Fatalf("NALs = %+v, want SPS/PPS/IDR", nals)
	}
	sps, err := DecodeSPS(nals[0].RBSP)
	if err != nil {
		t.Fatal(err)
	}
	var spsList [maxSPSCount]*SPS
	spsList[sps.SPSID] = sps
	pps, err := DecodePPS(nals[1].RBSP, &spsList)
	if err != nil {
		t.Fatal(err)
	}
	var ppsList [maxPPSCount]*PPS
	ppsList[pps.PPSID] = pps

	sh, payload, err := parseSliceHeaderWithPayload(nals[2], &ppsList)
	if err != nil {
		t.Fatalf("parse generated IDR slice header: %v", err)
	}
	if sh.FirstMBAddr != 0 || sh.SliceTypeNoS != PictureTypeI || sh.FrameNum != 5 ||
		sh.IDRPicID != 4 || sh.QScale != 23 || sh.DeblockingFilter != 0 ||
		sh.NBMMCO != 0 || sh.ExplicitRefMarking != 1 {
		t.Fatalf("slice header = %+v", sh)
	}
	if payload.bitsLeft() <= 384*8 {
		t.Fatalf("payload bits left = %d, want IntraPCM macroblock payload plus trailing bits", payload.bitsLeft())
	}
}

func TestBuildEncoderI420IntraPCMIDRSliceRejectsInvalidConfig(t *testing.T) {
	frame := encoderSliceTestI420(16, 16)
	cfg := EncoderI420IntraPCMIDRConfig{
		Width:                      16,
		Height:                     16,
		StrideY:                    16,
		StrideCb:                   8,
		StrideCr:                   8,
		Y:                          frame.y,
		Cb:                         frame.cb,
		Cr:                         frame.cr,
		InitialQP:                  26,
		DisableDeblockingFilterIDC: 0,
	}
	for _, tt := range []struct {
		name   string
		mutate func(*EncoderI420IntraPCMIDRConfig)
	}{
		{name: "odd width", mutate: func(c *EncoderI420IntraPCMIDRConfig) { c.Width = 15 }},
		{name: "small luma", mutate: func(c *EncoderI420IntraPCMIDRConfig) { c.Y = c.Y[:len(c.Y)-1] }},
		{name: "bad frame num", mutate: func(c *EncoderI420IntraPCMIDRConfig) { c.FrameNum = 256 }},
		{name: "bad idr pic id", mutate: func(c *EncoderI420IntraPCMIDRConfig) { c.IDRPicID = 65536 }},
		{name: "bad deblock idc", mutate: func(c *EncoderI420IntraPCMIDRConfig) { c.DisableDeblockingFilterIDC = 3 }},
		{name: "bad qp", mutate: func(c *EncoderI420IntraPCMIDRConfig) { c.InitialQP = 52 }},
		{name: "bad nal length", mutate: func(c *EncoderI420IntraPCMIDRConfig) { c.NALLengthSize = 5 }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			next := cfg
			tt.mutate(&next)
			if _, err := BuildEncoderI420IntraPCMIDRSlice(next); err != ErrInvalidData {
				t.Fatalf("BuildEncoderI420IntraPCMIDRSlice error = %v, want ErrInvalidData", err)
			}
		})
	}
}

type encoderSliceTestFrame struct {
	y  []byte
	cb []byte
	cr []byte
}

func encoderSliceTestI420(width, height int) encoderSliceTestFrame {
	chromaWidth := width / 2
	chromaHeight := height / 2
	frame := encoderSliceTestFrame{
		y:  make([]byte, width*height),
		cb: make([]byte, chromaWidth*chromaHeight),
		cr: make([]byte, chromaWidth*chromaHeight),
	}
	for i := range frame.y {
		frame.y[i] = byte(i*3 + 1)
	}
	for i := range frame.cb {
		frame.cb[i] = byte(i*5 + 7)
		frame.cr[i] = byte(i*11 + 13)
	}
	return frame
}
