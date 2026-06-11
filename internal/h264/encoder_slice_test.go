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

func TestBuildEncoderI420IntraPCMIDRSliceWritesMacroblockRange(t *testing.T) {
	sets, err := BuildEncoderParameterSets(EncoderParameterSetConfig{
		ProfileIDC:         66,
		ConstraintSetFlags: 0x03,
		LevelIDC:           31,
		Width:              48,
		Height:             16,
		FrameRateNum:       30,
		FrameRateDen:       1,
		MaxReferenceFrames: 1,
		InitialQP:          23,
	})
	if err != nil {
		t.Fatal(err)
	}
	frame := encoderSliceTestI420(48, 16)
	slice, err := BuildEncoderI420IntraPCMIDRSlice(EncoderI420IntraPCMIDRConfig{
		Width:                      48,
		Height:                     16,
		StrideY:                    48,
		StrideCb:                   24,
		StrideCr:                   24,
		Y:                          frame.y,
		Cb:                         frame.cb,
		Cr:                         frame.cr,
		FrameNum:                   5,
		IDRPicID:                   4,
		InitialQP:                  23,
		DisableDeblockingFilterIDC: 1,
		FirstMBAddr:                1,
		MacroblockCount:            2,
		NALLengthSize:              4,
	})
	if err != nil {
		t.Fatal(err)
	}

	sh, payload := parseEncoderSliceTestHeader(t, sets.AnnexB, slice.AnnexB)
	if sh.FirstMBAddr != 1 || sh.SliceTypeNoS != PictureTypeI || sh.FrameNum != 5 ||
		sh.IDRPicID != 4 || sh.QScale != 23 {
		t.Fatalf("slice header = %+v", sh)
	}
	if payload.bitsLeft() <= 2*384*8 {
		t.Fatalf("payload bits left = %d, want ranged IntraPCM macroblock payload plus trailing bits", payload.bitsLeft())
	}
}

func TestBuildEncoderI420PSkipSliceWritesParseableHeader(t *testing.T) {
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
	slice, err := BuildEncoderI420PSkipSlice(EncoderI420PSkipConfig{
		Width:                      18,
		Height:                     18,
		FrameNum:                   6,
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
	if len(nals) != 3 || nals[2].Type != NALSlice {
		t.Fatalf("NALs = %+v, want SPS/PPS/P-slice", nals)
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
		t.Fatalf("parse generated P-skip slice header: %v", err)
	}
	if sh.FirstMBAddr != 0 || sh.SliceTypeNoS != PictureTypeP || sh.FrameNum != 6 ||
		sh.RefCount[0] != 1 || sh.NBRefModifications[0] != 0 ||
		sh.NBMMCO != 0 || sh.QScale != 23 || sh.DeblockingFilter != 0 {
		t.Fatalf("slice header = %+v", sh)
	}
	run, err := payload.readUEGolombLong()
	if err != nil {
		t.Fatalf("read generated P-skip payload: %v", err)
	}
	if run != 4 || payload.bitsLeft() != 0 {
		t.Fatalf("P-skip payload run=%d bitsLeft=%d, want run 4 and no payload bits", run, payload.bitsLeft())
	}
}

func TestBuildEncoderI420PSkipSliceWritesMacroblockRange(t *testing.T) {
	sets, err := BuildEncoderParameterSets(EncoderParameterSetConfig{
		ProfileIDC:         66,
		ConstraintSetFlags: 0x03,
		LevelIDC:           31,
		Width:              48,
		Height:             16,
		FrameRateNum:       30,
		FrameRateDen:       1,
		MaxReferenceFrames: 1,
		InitialQP:          23,
	})
	if err != nil {
		t.Fatal(err)
	}
	slice, err := BuildEncoderI420PSkipSlice(EncoderI420PSkipConfig{
		Width:                      48,
		Height:                     16,
		FrameNum:                   6,
		InitialQP:                  23,
		DisableDeblockingFilterIDC: 1,
		FirstMBAddr:                1,
		MacroblockCount:            2,
		NALLengthSize:              4,
	})
	if err != nil {
		t.Fatal(err)
	}

	sh, payload := parseEncoderSliceTestHeader(t, sets.AnnexB, slice.AnnexB)
	if sh.FirstMBAddr != 1 || sh.SliceTypeNoS != PictureTypeP || sh.FrameNum != 6 ||
		sh.RefCount[0] != 1 || sh.QScale != 23 {
		t.Fatalf("slice header = %+v", sh)
	}
	run, err := payload.readUEGolombLong()
	if err != nil {
		t.Fatalf("read generated P-skip payload: %v", err)
	}
	if run != 2 || payload.bitsLeft() != 0 {
		t.Fatalf("P-skip payload run=%d bitsLeft=%d, want run 2 and no payload bits", run, payload.bitsLeft())
	}
}

func TestBuildEncoderI420P16x16NoResidualSliceWritesParseableHeader(t *testing.T) {
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
	slice, err := BuildEncoderI420P16x16NoResidualSlice(EncoderI420P16x16NoResidualConfig{
		Width:                      18,
		Height:                     18,
		FrameNum:                   6,
		InitialQP:                  23,
		DisableDeblockingFilterIDC: 1,
		MVDX:                       1,
		MVDY:                       -1,
		NALLengthSize:              4,
	})
	if err != nil {
		t.Fatal(err)
	}

	nals, err := SplitAnnexB(append(append([]byte(nil), sets.AnnexB...), slice.AnnexB...))
	if err != nil {
		t.Fatal(err)
	}
	if len(nals) != 3 || nals[2].Type != NALSlice {
		t.Fatalf("NALs = %+v, want SPS/PPS/P-slice", nals)
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
		t.Fatalf("parse generated P16x16 no-residual slice header: %v", err)
	}
	if sh.FirstMBAddr != 0 || sh.SliceTypeNoS != PictureTypeP || sh.FrameNum != 6 ||
		sh.RefCount[0] != 1 || sh.NBRefModifications[0] != 0 ||
		sh.NBMMCO != 0 || sh.QScale != 23 || sh.DeblockingFilter != 0 {
		t.Fatalf("slice header = %+v", sh)
	}
	assertEncoderP16x16NoResidualPayload(t, &payload, 4, 1, -1)
	if payload.bitsLeft() != 0 {
		t.Fatalf("P16x16 no-residual payload bitsLeft=%d, want 0", payload.bitsLeft())
	}
}

func TestBuildEncoderI420P16x16NoResidualSliceWritesMacroblockRange(t *testing.T) {
	sets, err := BuildEncoderParameterSets(EncoderParameterSetConfig{
		ProfileIDC:         66,
		ConstraintSetFlags: 0x03,
		LevelIDC:           31,
		Width:              48,
		Height:             16,
		FrameRateNum:       30,
		FrameRateDen:       1,
		MaxReferenceFrames: 1,
		InitialQP:          23,
	})
	if err != nil {
		t.Fatal(err)
	}
	slice, err := BuildEncoderI420P16x16NoResidualSlice(EncoderI420P16x16NoResidualConfig{
		Width:                      48,
		Height:                     16,
		FrameNum:                   6,
		InitialQP:                  23,
		DisableDeblockingFilterIDC: 1,
		FirstMBAddr:                1,
		MacroblockCount:            2,
		MVDX:                       -2,
		MVDY:                       3,
		NALLengthSize:              4,
	})
	if err != nil {
		t.Fatal(err)
	}

	sh, payload := parseEncoderSliceTestHeader(t, sets.AnnexB, slice.AnnexB)
	if sh.FirstMBAddr != 1 || sh.SliceTypeNoS != PictureTypeP || sh.FrameNum != 6 ||
		sh.RefCount[0] != 1 || sh.QScale != 23 {
		t.Fatalf("slice header = %+v", sh)
	}
	assertEncoderP16x16NoResidualPayload(t, &payload, 2, -2, 3)
	if payload.bitsLeft() != 0 {
		t.Fatalf("ranged P16x16 no-residual payload bitsLeft=%d, want 0", payload.bitsLeft())
	}
}

func TestBuildEncoderI420IntraPCMPSliceWritesParseableHeader(t *testing.T) {
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
	slice, err := BuildEncoderI420IntraPCMPSlice(EncoderI420IntraPCMPConfig{
		Width:                      18,
		Height:                     18,
		StrideY:                    18,
		StrideCb:                   9,
		StrideCr:                   9,
		Y:                          frame.y,
		Cb:                         frame.cb,
		Cr:                         frame.cr,
		FrameNum:                   6,
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
	if len(nals) != 3 || nals[2].Type != NALSlice {
		t.Fatalf("NALs = %+v, want SPS/PPS/P-slice", nals)
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
		t.Fatalf("parse generated P IntraPCM slice header: %v", err)
	}
	if sh.FirstMBAddr != 0 || sh.SliceTypeNoS != PictureTypeP || sh.FrameNum != 6 ||
		sh.RefCount[0] != 1 || sh.NBRefModifications[0] != 0 ||
		sh.NBMMCO != 0 || sh.QScale != 23 || sh.DeblockingFilter != 0 {
		t.Fatalf("slice header = %+v", sh)
	}
	skipRun, err := payload.readUEGolombLong()
	if err != nil {
		t.Fatalf("read generated P IntraPCM skip run: %v", err)
	}
	mbType, err := payload.readUEGolombLong()
	if err != nil {
		t.Fatalf("read generated P IntraPCM mb_type: %v", err)
	}
	if skipRun != 0 || mbType != 30 {
		t.Fatalf("first P IntraPCM macroblock skipRun=%d mbType=%d, want 0/30", skipRun, mbType)
	}
	if payload.bitsLeft() <= 384*8 {
		t.Fatalf("payload bits left = %d, want IntraPCM macroblock payload plus trailing bits", payload.bitsLeft())
	}
}

func TestBuildEncoderI420IntraPCMPSliceWritesMacroblockRange(t *testing.T) {
	sets, err := BuildEncoderParameterSets(EncoderParameterSetConfig{
		ProfileIDC:         66,
		ConstraintSetFlags: 0x03,
		LevelIDC:           31,
		Width:              48,
		Height:             16,
		FrameRateNum:       30,
		FrameRateDen:       1,
		MaxReferenceFrames: 1,
		InitialQP:          23,
	})
	if err != nil {
		t.Fatal(err)
	}
	frame := encoderSliceTestI420(48, 16)
	slice, err := BuildEncoderI420IntraPCMPSlice(EncoderI420IntraPCMPConfig{
		Width:                      48,
		Height:                     16,
		StrideY:                    48,
		StrideCb:                   24,
		StrideCr:                   24,
		Y:                          frame.y,
		Cb:                         frame.cb,
		Cr:                         frame.cr,
		FrameNum:                   6,
		InitialQP:                  23,
		DisableDeblockingFilterIDC: 1,
		FirstMBAddr:                1,
		MacroblockCount:            2,
		NALLengthSize:              4,
	})
	if err != nil {
		t.Fatal(err)
	}

	sh, payload := parseEncoderSliceTestHeader(t, sets.AnnexB, slice.AnnexB)
	if sh.FirstMBAddr != 1 || sh.SliceTypeNoS != PictureTypeP || sh.FrameNum != 6 ||
		sh.RefCount[0] != 1 || sh.QScale != 23 {
		t.Fatalf("slice header = %+v", sh)
	}
	skipRun, err := payload.readUEGolombLong()
	if err != nil {
		t.Fatalf("read generated P IntraPCM skip run: %v", err)
	}
	mbType, err := payload.readUEGolombLong()
	if err != nil {
		t.Fatalf("read generated P IntraPCM mb_type: %v", err)
	}
	if skipRun != 0 || mbType != 30 {
		t.Fatalf("first ranged P IntraPCM macroblock skipRun=%d mbType=%d, want 0/30", skipRun, mbType)
	}
	if payload.bitsLeft() <= 2*384*8 {
		t.Fatalf("payload bits left = %d, want ranged IntraPCM macroblock payload plus trailing bits", payload.bitsLeft())
	}
}

func TestBuildEncoderI420PSkipSliceRejectsInvalidConfig(t *testing.T) {
	cfg := EncoderI420PSkipConfig{
		Width:                      16,
		Height:                     16,
		FrameNum:                   1,
		InitialQP:                  26,
		DisableDeblockingFilterIDC: 0,
	}
	for _, tt := range []struct {
		name   string
		mutate func(*EncoderI420PSkipConfig)
	}{
		{name: "odd width", mutate: func(c *EncoderI420PSkipConfig) { c.Width = 15 }},
		{name: "zero height", mutate: func(c *EncoderI420PSkipConfig) { c.Height = 0 }},
		{name: "bad frame num", mutate: func(c *EncoderI420PSkipConfig) { c.FrameNum = 256 }},
		{name: "bad deblock idc", mutate: func(c *EncoderI420PSkipConfig) { c.DisableDeblockingFilterIDC = 3 }},
		{name: "bad qp", mutate: func(c *EncoderI420PSkipConfig) { c.InitialQP = 52 }},
		{name: "bad first mb", mutate: func(c *EncoderI420PSkipConfig) { c.FirstMBAddr = 1 }},
		{name: "bad macroblock count", mutate: func(c *EncoderI420PSkipConfig) { c.MacroblockCount = 2 }},
		{name: "bad nal length", mutate: func(c *EncoderI420PSkipConfig) { c.NALLengthSize = 5 }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			next := cfg
			tt.mutate(&next)
			if _, err := BuildEncoderI420PSkipSlice(next); err != ErrInvalidData {
				t.Fatalf("BuildEncoderI420PSkipSlice error = %v, want ErrInvalidData", err)
			}
		})
	}
}

func TestBuildEncoderI420P16x16NoResidualSliceRejectsInvalidConfig(t *testing.T) {
	cfg := EncoderI420P16x16NoResidualConfig{
		Width:                      16,
		Height:                     16,
		FrameNum:                   1,
		InitialQP:                  26,
		DisableDeblockingFilterIDC: 0,
	}
	for _, tt := range []struct {
		name   string
		mutate func(*EncoderI420P16x16NoResidualConfig)
	}{
		{name: "odd width", mutate: func(c *EncoderI420P16x16NoResidualConfig) { c.Width = 15 }},
		{name: "zero height", mutate: func(c *EncoderI420P16x16NoResidualConfig) { c.Height = 0 }},
		{name: "bad frame num", mutate: func(c *EncoderI420P16x16NoResidualConfig) { c.FrameNum = 256 }},
		{name: "bad deblock idc", mutate: func(c *EncoderI420P16x16NoResidualConfig) { c.DisableDeblockingFilterIDC = 3 }},
		{name: "bad qp", mutate: func(c *EncoderI420P16x16NoResidualConfig) { c.InitialQP = 52 }},
		{name: "bad first mb", mutate: func(c *EncoderI420P16x16NoResidualConfig) { c.FirstMBAddr = 1 }},
		{name: "bad macroblock count", mutate: func(c *EncoderI420P16x16NoResidualConfig) { c.MacroblockCount = 2 }},
		{name: "bad nal length", mutate: func(c *EncoderI420P16x16NoResidualConfig) { c.NALLengthSize = 5 }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			next := cfg
			tt.mutate(&next)
			if _, err := BuildEncoderI420P16x16NoResidualSlice(next); err != ErrInvalidData {
				t.Fatalf("BuildEncoderI420P16x16NoResidualSlice error = %v, want ErrInvalidData", err)
			}
		})
	}
}

func TestBuildEncoderI420IntraPCMPSliceRejectsInvalidConfig(t *testing.T) {
	frame := encoderSliceTestI420(16, 16)
	cfg := EncoderI420IntraPCMPConfig{
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
		mutate func(*EncoderI420IntraPCMPConfig)
	}{
		{name: "odd width", mutate: func(c *EncoderI420IntraPCMPConfig) { c.Width = 15 }},
		{name: "small luma", mutate: func(c *EncoderI420IntraPCMPConfig) { c.Y = c.Y[:len(c.Y)-1] }},
		{name: "bad frame num", mutate: func(c *EncoderI420IntraPCMPConfig) { c.FrameNum = 256 }},
		{name: "bad deblock idc", mutate: func(c *EncoderI420IntraPCMPConfig) { c.DisableDeblockingFilterIDC = 3 }},
		{name: "bad qp", mutate: func(c *EncoderI420IntraPCMPConfig) { c.InitialQP = 52 }},
		{name: "bad first mb", mutate: func(c *EncoderI420IntraPCMPConfig) { c.FirstMBAddr = 1 }},
		{name: "bad macroblock count", mutate: func(c *EncoderI420IntraPCMPConfig) { c.MacroblockCount = 2 }},
		{name: "bad nal length", mutate: func(c *EncoderI420IntraPCMPConfig) { c.NALLengthSize = 5 }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			next := cfg
			tt.mutate(&next)
			if _, err := BuildEncoderI420IntraPCMPSlice(next); err != ErrInvalidData {
				t.Fatalf("BuildEncoderI420IntraPCMPSlice error = %v, want ErrInvalidData", err)
			}
		})
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
		{name: "bad first mb", mutate: func(c *EncoderI420IntraPCMIDRConfig) { c.FirstMBAddr = 1 }},
		{name: "bad macroblock count", mutate: func(c *EncoderI420IntraPCMIDRConfig) { c.MacroblockCount = 2 }},
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

func parseEncoderSliceTestHeader(t *testing.T, annexBHeaders []byte, annexBSlice []byte) (*SliceHeader, bitReader) {
	t.Helper()
	nals, err := SplitAnnexB(append(append([]byte(nil), annexBHeaders...), annexBSlice...))
	if err != nil {
		t.Fatal(err)
	}
	if len(nals) != 3 || (nals[2].Type != NALIDRSlice && nals[2].Type != NALSlice) {
		t.Fatalf("NALs = %+v, want SPS/PPS/VCL", nals)
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
		t.Fatalf("parse generated slice header: %v", err)
	}
	return sh, payload
}

func assertEncoderP16x16NoResidualPayload(t *testing.T, payload *bitReader, macroblockCount int, wantMVDX int32, wantMVDY int32) {
	t.Helper()
	for i := 0; i < macroblockCount; i++ {
		skipRun, err := payload.readUEGolombLong()
		if err != nil {
			t.Fatalf("read generated P16x16 skip run[%d]: %v", i, err)
		}
		mbType, err := payload.readUEGolombLong()
		if err != nil {
			t.Fatalf("read generated P16x16 mb_type[%d]: %v", i, err)
		}
		mvdX, err := payload.readSEGolombLong()
		if err != nil {
			t.Fatalf("read generated P16x16 mvd_l0_x[%d]: %v", i, err)
		}
		mvdY, err := payload.readSEGolombLong()
		if err != nil {
			t.Fatalf("read generated P16x16 mvd_l0_y[%d]: %v", i, err)
		}
		cbp, err := payload.readUEGolombLong()
		if err != nil {
			t.Fatalf("read generated P16x16 cbp[%d]: %v", i, err)
		}
		if skipRun != 0 || mbType != 0 || mvdX != wantMVDX || mvdY != wantMVDY || cbp != 0 {
			t.Fatalf("P16x16 macroblock[%d] skip/mb/mvd/cbp = %d/%d/%d,%d/%d, want 0/0/%d,%d/0",
				i, skipRun, mbType, mvdX, mvdY, cbp, wantMVDX, wantMVDY)
		}
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
