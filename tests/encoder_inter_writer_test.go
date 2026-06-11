// SPDX-License-Identifier: LGPL-2.1-or-later

package goh264_test

import (
	"testing"

	goh264 "github.com/thesyncim/goh264"
	"github.com/thesyncim/goh264/internal/h264"
)

func TestEncoderI420P16x16NoResidualWriterDecodesThroughLocalAndFFmpeg(t *testing.T) {
	sets, err := h264.BuildEncoderParameterSets(h264.EncoderParameterSetConfig{
		ProfileIDC:         66,
		ConstraintSetFlags: 0x03,
		LevelIDC:           31,
		Width:              16,
		Height:             16,
		FrameRateNum:       30,
		FrameRateDen:       1,
		MaxReferenceFrames: 1,
		InitialQP:          26,
	})
	if err != nil {
		t.Fatalf("BuildEncoderParameterSets: %v", err)
	}
	frame := patternedI420EncoderFrame(16, 16)
	idr, err := h264.BuildEncoderI420IntraPCMIDRSlice(h264.EncoderI420IntraPCMIDRConfig{
		Width:                      16,
		Height:                     16,
		StrideY:                    frame.StrideY,
		StrideCb:                   frame.StrideCb,
		StrideCr:                   frame.StrideCr,
		Y:                          frame.Y,
		Cb:                         frame.Cb,
		Cr:                         frame.Cr,
		FrameNum:                   0,
		IDRPicID:                   0,
		InitialQP:                  26,
		DisableDeblockingFilterIDC: 1,
		NALLengthSize:              4,
	})
	if err != nil {
		t.Fatalf("BuildEncoderI420IntraPCMIDRSlice: %v", err)
	}
	p16, err := h264.BuildEncoderI420P16x16NoResidualSlice(h264.EncoderI420P16x16NoResidualConfig{
		Width:                      16,
		Height:                     16,
		FrameNum:                   1,
		InitialQP:                  26,
		DisableDeblockingFilterIDC: 1,
		NALLengthSize:              4,
	})
	if err != nil {
		t.Fatalf("BuildEncoderI420P16x16NoResidualSlice: %v", err)
	}

	firstAU := append(append([]byte(nil), sets.AnnexB...), idr.AnnexB...)
	dec := goh264.NewDecoder()
	decodedFirst, err := dec.DecodeFrames(firstAU)
	if err != nil {
		t.Fatalf("Decode IDR IntraPCM: %v", err)
	}
	assertDecodedEncoderFrameBytes(t, decodedFirst, appendI420FrameBytes(nil, frame))
	decodedSecond, err := dec.DecodeFrames(p16.AnnexB)
	if err != nil {
		t.Fatalf("Decode P16x16 no-residual: %v", err)
	}
	assertDecodedEncoderFrameBytes(t, decodedSecond, appendI420FrameBytes(nil, frame))

	stream := append(append([]byte(nil), firstAU...), p16.AnnexB...)
	want := appendI420FrameBytes(nil, frame)
	want = appendI420FrameBytes(want, frame)
	assertFFmpegRawVideoOracle(t, stream, want)
}
