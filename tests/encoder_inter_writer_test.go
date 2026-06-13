// SPDX-License-Identifier: LGPL-2.1-or-later

package goh264_test

import (
	"bytes"
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

func TestEncoderI420P16x16ResidualWriterDecodesThroughPublicAndFFmpeg(t *testing.T) {
	const initialQP = 20
	sets, err := h264.BuildEncoderParameterSets(h264.EncoderParameterSetConfig{
		ProfileIDC:         66,
		ConstraintSetFlags: 0x03,
		LevelIDC:           31,
		Width:              16,
		Height:             16,
		FrameRateNum:       30,
		FrameRateDen:       1,
		MaxReferenceFrames: 1,
		InitialQP:          initialQP,
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
		InitialQP:                  initialQP,
		DisableDeblockingFilterIDC: 1,
		NALLengthSize:              4,
	})
	if err != nil {
		t.Fatalf("BuildEncoderI420IntraPCMIDRSlice: %v", err)
	}
	residual, err := h264.BuildEncoderI420P16x16ResidualSlice(h264.EncoderI420P16x16ResidualConfig{
		Width:                      16,
		Height:                     16,
		FrameNum:                   1,
		InitialQP:                  initialQP,
		NextQP:                     initialQP,
		DisableDeblockingFilterIDC: 1,
		Coeff:                      4,
		ChromaDCCoeffCb:            2,
		ChromaDCCoeffCr:            -2,
		ChromaACCoeffCb:            1,
		ChromaACCoeffCr:            -1,
		NALLengthSize:              4,
	})
	if err != nil {
		t.Fatalf("BuildEncoderI420P16x16ResidualSlice: %v", err)
	}

	firstAU := append(append([]byte(nil), sets.AnnexB...), idr.AnnexB...)
	dec := goh264.NewDecoder()
	decodedFirst, err := dec.DecodeFrames(firstAU)
	if err != nil {
		t.Fatalf("Decode IDR IntraPCM: %v", err)
	}
	frameRaw := appendI420FrameBytes(nil, frame)
	assertDecodedEncoderFrameBytes(t, decodedFirst, frameRaw)
	decodedSecond, err := dec.DecodeFrames(residual.AnnexB)
	if err != nil {
		t.Fatalf("Decode P16x16 residual: %v", err)
	}
	if len(decodedSecond) != 1 {
		t.Fatalf("residual decoded frames = %d, want 1", len(decodedSecond))
	}
	residualRaw, err := decodedSecond[0].AppendRawYUV(nil)
	if err != nil {
		t.Fatalf("AppendRawYUV residual: %v", err)
	}
	if len(residualRaw) != len(frameRaw) {
		t.Fatalf("residual raw len = %d, want %d", len(residualRaw), len(frameRaw))
	}
	if bytes.Equal(residualRaw, frameRaw) {
		t.Fatalf("residual frame unexpectedly matched IDR frame")
	}

	stream := append(append([]byte(nil), firstAU...), residual.AnnexB...)
	want := append(append([]byte(nil), frameRaw...), residualRaw...)
	assertFFmpegRawVideoOracle(t, stream, want)
}
