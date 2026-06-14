// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestBuildEncoderParameterSetsRoundTripsThroughParsers(t *testing.T) {
	cfg := EncoderParameterSetConfig{
		ProfileIDC:                     66,
		ConstraintSetFlags:             0x03,
		LevelIDC:                       31,
		Width:                          638,
		Height:                         478,
		FrameRateNum:                   30000,
		FrameRateDen:                   1001,
		MaxReferenceFrames:             1,
		InitialQP:                      24,
		SARNum:                         1,
		SARDen:                         1,
		FullRange:                      true,
		ColorPrimaries:                 1,
		ColorTransfer:                  1,
		ColorMatrix:                    1,
		ChromaSampleLocTypeTopField:    2,
		ChromaSampleLocTypeBottomField: 2,
	}

	sets, err := BuildEncoderParameterSets(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(sets.SPS) == 0 || sets.SPS[0]&0x1f != byte(NALSPS) {
		t.Fatalf("SPS NAL = %x", sets.SPS)
	}
	if len(sets.PPS) == 0 || sets.PPS[0]&0x1f != byte(NALPPS) {
		t.Fatalf("PPS NAL = %x", sets.PPS)
	}

	nals, err := SplitAnnexB(sets.AnnexB)
	if err != nil {
		t.Fatal(err)
	}
	if len(nals) != 2 || nals[0].Type != NALSPS || nals[1].Type != NALPPS {
		t.Fatalf("Annex B NALs = %+v", nals)
	}
	if !bytes.Equal(nals[0].Raw, sets.SPS) || !bytes.Equal(nals[1].Raw, sets.PPS) {
		t.Fatalf("Annex B raw NALs do not match parameter sets")
	}

	sps, err := DecodeSPS(nals[0].RBSP)
	if err != nil {
		t.Fatal(err)
	}
	if sps.ProfileIDC != 66 || sps.ConstraintSetFlags != 0x03 || sps.LevelIDC != 31 ||
		sps.Width != 638 || sps.Height != 478 || sps.ChromaFormatIDC != 1 ||
		sps.RefFrameCount != 1 || sps.PocType != 2 || sps.Log2MaxFrameNum != 8 {
		t.Fatalf("SPS = %+v", sps)
	}
	if sps.VUI.SARNum != 1 || sps.VUI.SARDen != 1 ||
		sps.VUI.VideoFullRangeFlag != 1 ||
		sps.VUI.ColourPrimaries != 1 ||
		sps.VUI.TransferCharacteristics != 1 ||
		sps.VUI.MatrixCoeffs != 1 ||
		sps.VUI.ChromaSampleLocTypeTopField != 2 ||
		sps.VUI.ChromaSampleLocTypeBottomField != 2 ||
		sps.TimingInfoPresentFlag != 1 ||
		sps.NumUnitsInTick != 1001 ||
		sps.TimeScale != 60000 ||
		sps.FixedFrameRateFlag != 1 ||
		sps.NumReorderFrames != 0 ||
		sps.MaxDecFrameBuffering != 1 {
		t.Fatalf("SPS VUI/restriction = %+v timing=%d/%d fixed=%d reorder=%d dpb=%d",
			sps.VUI, sps.NumUnitsInTick, sps.TimeScale, sps.FixedFrameRateFlag,
			sps.NumReorderFrames, sps.MaxDecFrameBuffering)
	}

	var spsList [maxSPSCount]*SPS
	spsList[sps.SPSID] = sps
	pps, err := DecodePPS(nals[1].RBSP, &spsList)
	if err != nil {
		t.Fatal(err)
	}
	if pps.PPSID != 0 || pps.SPSID != 0 || pps.CABAC != 0 ||
		pps.RefCount[0] != 1 || pps.RefCount[1] != 1 ||
		pps.InitQP != 24 || pps.DeblockingFilterParametersPresent != 1 {
		t.Fatalf("PPS = %+v", pps)
	}

	avcc, err := DecodeAVCDecoderConfigurationRecord(sets.AVCDecoderConfigurationRecord)
	if err != nil {
		t.Fatal(err)
	}
	if got := sets.AVCDecoderConfigurationRecord[2]; got != 0xc0 {
		t.Fatalf("avcC profile_compatibility = %#02x, want 0xc0", got)
	}
	if avcc.NALLengthSize != 4 || avcc.SPS[0] == nil || avcc.PPS[0] == nil {
		t.Fatalf("avcC = %+v", avcc)
	}
}

func TestBuildEncoderParameterSetNALsMatchPackagedSets(t *testing.T) {
	cfg := EncoderParameterSetConfig{
		ProfileIDC:         66,
		ConstraintSetFlags: 0x03,
		LevelIDC:           31,
		Width:              640,
		Height:             480,
		FrameRateNum:       30,
		FrameRateDen:       1,
		MaxReferenceFrames: 1,
		InitialQP:          26,
		NALLengthSize:      4,
	}
	sets, err := BuildEncoderParameterSets(cfg)
	if err != nil {
		t.Fatal(err)
	}
	nals, err := BuildEncoderParameterSetNALs(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(nals.SPS, sets.SPS) || !bytes.Equal(nals.PPS, sets.PPS) {
		t.Fatalf("raw parameter set NALs SPS=%x PPS=%x, want SPS=%x PPS=%x", nals.SPS, nals.PPS, sets.SPS, sets.PPS)
	}
}

func TestBuildEncoderParameterSetsWritesI420Crop(t *testing.T) {
	cfg := EncoderParameterSetConfig{
		ProfileIDC:         66,
		ConstraintSetFlags: 0x03,
		LevelIDC:           31,
		Width:              640,
		Height:             480,
		CropLeft:           2,
		CropRight:          4,
		CropTop:            6,
		CropBottom:         8,
		FrameRateNum:       30,
		FrameRateDen:       1,
		MaxReferenceFrames: 1,
		InitialQP:          26,
	}

	sets, err := BuildEncoderParameterSets(cfg)
	if err != nil {
		t.Fatal(err)
	}
	nals, err := SplitAnnexB(sets.AnnexB)
	if err != nil {
		t.Fatal(err)
	}
	sps, err := DecodeSPS(nals[0].RBSP)
	if err != nil {
		t.Fatal(err)
	}
	if sps.Width != 634 || sps.Height != 466 ||
		sps.CropLeft != 2 || sps.CropRight != 4 ||
		sps.CropTop != 6 || sps.CropBottom != 8 {
		t.Fatalf("SPS crop = width %d height %d left/right/top/bottom %d/%d/%d/%d, want 634x466 2/4/6/8",
			sps.Width, sps.Height, sps.CropLeft, sps.CropRight, sps.CropTop, sps.CropBottom)
	}
}

func TestBuildEncoderParameterSetsRejectsInvalidSyntaxConfig(t *testing.T) {
	cfg := EncoderParameterSetConfig{
		ProfileIDC:         66,
		ConstraintSetFlags: 0x03,
		LevelIDC:           31,
		Width:              16,
		Height:             16,
		FrameRateNum:       30,
		FrameRateDen:       1,
		MaxReferenceFrames: 1,
		InitialQP:          26,
	}
	for _, tt := range []struct {
		name   string
		mutate func(*EncoderParameterSetConfig)
	}{
		{name: "odd width", mutate: func(c *EncoderParameterSetConfig) { c.Width = 15 }},
		{name: "bad sps id", mutate: func(c *EncoderParameterSetConfig) { c.SPSID = maxSPSCount }},
		{name: "bad qp", mutate: func(c *EncoderParameterSetConfig) { c.InitialQP = 52 }},
		{name: "bad sar", mutate: func(c *EncoderParameterSetConfig) { c.SARNum = 1 }},
		{name: "negative crop", mutate: func(c *EncoderParameterSetConfig) { c.CropLeft = -2 }},
		{name: "odd crop", mutate: func(c *EncoderParameterSetConfig) { c.CropTop = 1 }},
		{name: "crop consumes height", mutate: func(c *EncoderParameterSetConfig) { c.CropTop = 8; c.CropBottom = 8 }},
		{name: "bad nal length", mutate: func(c *EncoderParameterSetConfig) { c.NALLengthSize = 5 }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			next := cfg
			tt.mutate(&next)
			if _, err := BuildEncoderParameterSets(next); !errors.Is(err, ErrInvalidData) {
				t.Fatalf("BuildEncoderParameterSets error = %v, want ErrInvalidData", err)
			}
		})
	}
}

func TestEncoderCanonicalSPSPPSNALGoldens(t *testing.T) {
	cfg := EncoderParameterSetConfig{
		ProfileIDC:         66,
		ConstraintSetFlags: 0x03,
		LevelIDC:           31,
		Width:              16,
		Height:             16,
		FrameRateNum:       30,
		FrameRateDen:       1,
		MaxReferenceFrames: 1,
		InitialQP:          26,
	}
	sets, err := BuildEncoderParameterSets(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if want := mustHex(t, "6742c01f95a7a10000030001000003003c8f08042a"); !bytes.Equal(sets.SPS, want) {
		t.Fatalf("canonical SPS = %x, want %x", sets.SPS, want)
	}
	if want := mustHex(t, "68ce3c80"); !bytes.Equal(sets.PPS, want) {
		t.Fatalf("canonical PPS = %x, want %x", sets.PPS, want)
	}
	if want := mustHex(t, "000000016742c01f95a7a10000030001000003003c8f08042a0000000168ce3c80"); !bytes.Equal(sets.AnnexB, want) {
		t.Fatalf("canonical Annex B parameter sets = %x, want %x", sets.AnnexB, want)
	}
	if want := mustHex(t, "0142c01fffe100156742c01f95a7a10000030001000003003c8f08042a01000468ce3c80"); !bytes.Equal(sets.AVCDecoderConfigurationRecord, want) {
		t.Fatalf("canonical avcC = %x, want %x", sets.AVCDecoderConfigurationRecord, want)
	}
}

func TestEncoderCanonicalIDRAndPSkipNALGoldens(t *testing.T) {
	idr, err := BuildEncoderI420IntraPCMIDRSlice(EncoderI420IntraPCMIDRConfig{
		Width:                      16,
		Height:                     16,
		StrideY:                    16,
		StrideCb:                   8,
		StrideCr:                   8,
		Y:                          make([]byte, 16*16),
		Cb:                         make([]byte, 8*8),
		Cr:                         make([]byte, 8*8),
		InitialQP:                  26,
		DisableDeblockingFilterIDC: 0,
		NALLengthSize:              4,
	})
	if err != nil {
		t.Fatal(err)
	}
	idrNALHex := "65b804f0d0" + strings.Repeat("000003", 191) + "000080"
	if want := mustHex(t, "00000001"+idrNALHex); !bytes.Equal(idr.AnnexB, want) {
		t.Fatalf("canonical IDR Annex B = %x, want %x", idr.AnnexB, want)
	}
	if want := mustHex(t, "00000245"+idrNALHex); !bytes.Equal(idr.AVC, want) {
		t.Fatalf("canonical IDR AVC = %x, want %x", idr.AVC, want)
	}

	pskip, err := BuildEncoderI420PSkipSlice(EncoderI420PSkipConfig{
		Width:                      16,
		Height:                     16,
		FrameNum:                   1,
		InitialQP:                  26,
		DisableDeblockingFilterIDC: 0,
		NALLengthSize:              4,
	})
	if err != nil {
		t.Fatal(err)
	}
	if want := mustHex(t, "0000000141e023d4"); !bytes.Equal(pskip.AnnexB, want) {
		t.Fatalf("canonical P-skip Annex B = %x, want %x", pskip.AnnexB, want)
	}
	if want := mustHex(t, "0000000441e023d4"); !bytes.Equal(pskip.AVC, want) {
		t.Fatalf("canonical P-skip AVC = %x, want %x", pskip.AVC, want)
	}
}
