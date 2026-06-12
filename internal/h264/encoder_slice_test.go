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

func TestEncoderSliceRBSPCapacityChecksOverflow(t *testing.T) {
	if got, err := encoderSliceRBSPCapacity(2, 384); err != nil || got != 800 {
		t.Fatalf("encoderSliceRBSPCapacity(2,384) = %d/%v, want 800/nil", got, err)
	}
	for _, tt := range []struct {
		name               string
		macroblockCount    int
		bytesPerMacroblock int
	}{
		{name: "negative macroblocks", macroblockCount: -1, bytesPerMacroblock: 384},
		{name: "negative byte estimate", macroblockCount: 1, bytesPerMacroblock: -1},
		{name: "payload multiply", macroblockCount: maxInt/384 + 1, bytesPerMacroblock: 384},
		{name: "header add", macroblockCount: maxInt - 31, bytesPerMacroblock: 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := encoderSliceRBSPCapacity(tt.macroblockCount, tt.bytesPerMacroblock); err != ErrInvalidData {
				t.Fatalf("encoderSliceRBSPCapacity(%d,%d) error = %v, want ErrInvalidData",
					tt.macroblockCount, tt.bytesPerMacroblock, err)
			}
		})
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

func TestBuildEncoderI420P16x16NoResidualSliceWritesPerMacroblockMVDs(t *testing.T) {
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
	wantMVDs := []EncoderMotionVectorDelta{
		{X: 4, Y: 0},
		{X: 0, Y: -1},
		{X: -3, Y: 2},
	}
	slice, err := BuildEncoderI420P16x16NoResidualSlice(EncoderI420P16x16NoResidualConfig{
		Width:                      48,
		Height:                     16,
		FrameNum:                   7,
		InitialQP:                  23,
		DisableDeblockingFilterIDC: 1,
		MVDs:                       wantMVDs,
		NALLengthSize:              4,
	})
	if err != nil {
		t.Fatal(err)
	}

	sh, payload := parseEncoderSliceTestHeader(t, sets.AnnexB, slice.AnnexB)
	if sh.FirstMBAddr != 0 || sh.SliceTypeNoS != PictureTypeP || sh.FrameNum != 7 ||
		sh.RefCount[0] != 1 || sh.QScale != 23 {
		t.Fatalf("slice header = %+v", sh)
	}
	assertEncoderP16x16NoResidualPayloadMVDs(t, &payload, wantMVDs)
	if payload.bitsLeft() != 0 {
		t.Fatalf("per-MB P16x16 no-residual payload bitsLeft=%d, want 0", payload.bitsLeft())
	}
}

func TestEncodeI420P16x16ResidualSliceRBSPDecodesCAVLCMacroblock(t *testing.T) {
	pps, sps := encoderResidualSliceTestPPS(20)

	rbsp, err := encodeI420P16x16ResidualSliceRBSP(encoderI420P16x16ResidualConfig{
		Width:                      16,
		Height:                     16,
		FrameNum:                   7,
		InitialQP:                  20,
		NextQP:                     23,
		DisableDeblockingFilterIDC: 1,
		MVDX:                       2,
		MVDY:                       -1,
		Coeff:                      1,
	}, pps, sps)
	if err != nil {
		t.Fatalf("encode residual slice rbsp: %v", err)
	}

	var ppsList [maxPPSCount]*PPS
	ppsList[0] = pps
	sh, payload, err := parseSliceHeaderWithPayload(NALUnit{Type: NALSlice, RefIDC: 2, RBSP: rbsp}, &ppsList)
	if err != nil {
		t.Fatalf("parse residual P slice header: %v", err)
	}
	if sh.FirstMBAddr != 0 || sh.SliceTypeNoS != PictureTypeP || sh.FrameNum != 7 ||
		sh.RefCount[0] != 1 || sh.QScale != 20 || sh.DeblockingFilter != 0 {
		t.Fatalf("slice header = %+v", sh)
	}
	skipRun, err := payload.readUEGolombLong()
	if err != nil {
		t.Fatalf("read generated residual P skip run: %v", err)
	}
	if skipRun != 0 {
		t.Fatalf("skip run = %d, want 0", skipRun)
	}

	var decoded cavlcResidualContext
	got, err := decoded.decodeCAVLCInterPMacroblock(&payload, pps, sps, int(sh.QScale), [2]uint32{1, 0}, false)
	if err != nil {
		t.Fatalf("decode generated residual P macroblock: %v", err)
	}
	if got.MBType != (MBType16x16|MBTypeP0L0) || got.CBP != 1 ||
		got.QScale != 23 || got.ChromaQP != ([2]uint8{23, 23}) || got.CBPTable != 0x1001 {
		t.Fatalf("decoded mb type/cbp/q/chroma/cbpTable = %#x/%#x/%d/%v/%#x",
			got.MBType, got.CBP, got.QScale, got.ChromaQP, got.CBPTable)
	}
	if got.MVD[0][0] != ([2]int32{2, -1}) || decoded.MB[0] != 1 {
		t.Fatalf("decoded motion/residual = %v/%d, want [2 -1]/1", got.MVD[0][0], decoded.MB[0])
	}
	if payload.bitsLeft() != 0 {
		t.Fatalf("residual P payload bitsLeft = %d, want 0", payload.bitsLeft())
	}
}

func TestEncodeI420P16x16ResidualSliceRBSPDecodesPerMacroblockSyntax(t *testing.T) {
	pps, sps := encoderResidualSliceTestPPS(20)
	wantMVDs := []EncoderMotionVectorDelta{
		{X: 2, Y: -1},
		{X: -3, Y: 4},
	}
	wantCoeffs := []int32{1, -2}

	rbsp, err := encodeI420P16x16ResidualSliceRBSP(encoderI420P16x16ResidualConfig{
		Width:                      32,
		Height:                     16,
		FrameNum:                   8,
		InitialQP:                  20,
		NextQP:                     23,
		DisableDeblockingFilterIDC: 1,
		MVDs:                       wantMVDs,
		Coeffs:                     wantCoeffs,
	}, pps, sps)
	if err != nil {
		t.Fatalf("encode residual slice rbsp: %v", err)
	}

	var ppsList [maxPPSCount]*PPS
	ppsList[0] = pps
	sh, payload, err := parseSliceHeaderWithPayload(NALUnit{Type: NALSlice, RefIDC: 2, RBSP: rbsp}, &ppsList)
	if err != nil {
		t.Fatalf("parse residual P slice header: %v", err)
	}
	if sh.FirstMBAddr != 0 || sh.SliceTypeNoS != PictureTypeP || sh.FrameNum != 8 ||
		sh.RefCount[0] != 1 || sh.QScale != 20 || sh.DeblockingFilter != 0 {
		t.Fatalf("slice header = %+v", sh)
	}

	qscale := int(sh.QScale)
	for i := range wantMVDs {
		skipRun, err := payload.readUEGolombLong()
		if err != nil {
			t.Fatalf("read generated residual P skip run[%d]: %v", i, err)
		}
		if skipRun != 0 {
			t.Fatalf("skip run[%d] = %d, want 0", i, skipRun)
		}
		var decoded cavlcResidualContext
		got, err := decoded.decodeCAVLCInterPMacroblock(&payload, pps, sps, qscale, [2]uint32{1, 0}, false)
		if err != nil {
			t.Fatalf("decode generated residual P macroblock[%d]: %v", i, err)
		}
		if got.MBType != (MBType16x16|MBTypeP0L0) || got.CBP != 1 ||
			got.QScale != 23 || got.ChromaQP != ([2]uint8{23, 23}) || got.CBPTable != 0x1001 {
			t.Fatalf("decoded mb[%d] type/cbp/q/chroma/cbpTable = %#x/%#x/%d/%v/%#x",
				i, got.MBType, got.CBP, got.QScale, got.ChromaQP, got.CBPTable)
		}
		if got.MVD[0][0] != ([2]int32{wantMVDs[i].X, wantMVDs[i].Y}) || decoded.MB[0] != wantCoeffs[i] {
			t.Fatalf("decoded mb[%d] motion/residual = %v/%d, want [%d %d]/%d",
				i, got.MVD[0][0], decoded.MB[0], wantMVDs[i].X, wantMVDs[i].Y, wantCoeffs[i])
		}
		qscale = got.QScale
	}
	if payload.bitsLeft() != 0 {
		t.Fatalf("residual P payload bitsLeft = %d, want 0", payload.bitsLeft())
	}
}

func TestEncodeI420P16x16ResidualSliceRBSPDecodesThroughFrameMacroblockPath(t *testing.T) {
	pps, sps := encoderResidualSliceTestPPS(20)
	m, err := newMacroblockTables(2, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	wantMVDs := []EncoderMotionVectorDelta{
		{X: 2, Y: -1},
		{X: -3, Y: 4},
	}
	wantCoeffs := []int32{1, -2}
	rbsp, err := encodeI420P16x16ResidualSliceRBSP(encoderI420P16x16ResidualConfig{
		Width:                      32,
		Height:                     16,
		FrameNum:                   9,
		InitialQP:                  20,
		NextQP:                     23,
		DisableDeblockingFilterIDC: 1,
		MVDs:                       wantMVDs,
		Coeffs:                     wantCoeffs,
	}, pps, sps)
	if err != nil {
		t.Fatalf("encode residual slice rbsp: %v", err)
	}

	var ppsList [maxPPSCount]*PPS
	ppsList[0] = pps
	sh, payload, err := parseSliceHeaderWithPayload(NALUnit{Type: NALSlice, RefIDC: 2, RBSP: rbsp}, &ppsList)
	if err != nil {
		t.Fatalf("parse residual P slice header: %v", err)
	}
	state := newCAVLCFrameSliceState(int(sh.QScale))
	sliceNum := uint16(12)
	wantType := MBType16x16 | MBTypeP0L0
	wantMotion := [][2]int16{
		{2, -1},
		{-1, 3},
	}

	for mbXY := 0; mbXY < 2; mbXY++ {
		got, err := m.decodeCAVLCFrameSliceMacroblock(&payload, sh, &state, mbXY, sliceNum)
		if err != nil {
			t.Fatalf("decode generated residual P frame macroblock[%d]: %v", mbXY, err)
		}
		if got.Skipped || !got.IsInter || got.MBType != wantType || got.CBP != 1 ||
			got.CBPTable != 0x1001 || got.QScale != 23 || got.ChromaQP != ([2]uint8{23, 23}) {
			t.Fatalf("decoded frame mb[%d] skip/inter/type/cbp/cbpTable/q/chroma = %v/%v/%#x/%#x/%#x/%d/%v",
				mbXY, got.Skipped, got.IsInter, got.MBType, got.CBP, got.CBPTable, got.QScale, got.ChromaQP)
		}
		if got.Inter.MVD[0][0] != ([2]int32{wantMVDs[mbXY].X, wantMVDs[mbXY].Y}) {
			t.Fatalf("decoded frame mb[%d] mvd = %v, want [%d %d]",
				mbXY, got.Inter.MVD[0][0], wantMVDs[mbXY].X, wantMVDs[mbXY].Y)
		}
		if state.MBSkipRun != cavlcMBSkipRunUnset || state.QScale != 23 {
			t.Fatalf("state after mb[%d] skip/q = %d/%d, want unset/23", mbXY, state.MBSkipRun, state.QScale)
		}
		if m.MacroblockTyp[mbXY] != wantType || m.CBPTable[mbXY] != 0x1001 ||
			m.QScaleTable[mbXY] != 23 || m.SliceTable[mbXY] != sliceNum {
			t.Fatalf("tables mb[%d] type/cbp/q/slice = %#x/%#x/%d/%d",
				mbXY, m.MacroblockTyp[mbXY], m.CBPTable[mbXY], m.QScaleTable[mbXY], m.SliceTable[mbXY])
		}
		bXY := int(m.MB2BXY[mbXY])
		if m.MotionVal[0][bXY] != wantMotion[mbXY] || m.MotionVal[0][bXY+3+3*m.BStride] != wantMotion[mbXY] {
			t.Fatalf("motion mb[%d] = %v/%v, want %v",
				mbXY, m.MotionVal[0][bXY], m.MotionVal[0][bXY+3+3*m.BStride], wantMotion[mbXY])
		}
		if m.RefIndex[0][4*mbXY] != 0 || m.RefIndex[0][4*mbXY+1] != 0 ||
			m.RefIndex[0][4*mbXY+2] != 0 || m.RefIndex[0][4*mbXY+3] != 0 {
			t.Fatalf("refs mb[%d] = %v, want all 0", mbXY, m.RefIndex[0][4*mbXY:4*mbXY+4])
		}
		if m.NonZeroCount[mbXY][0] != 1 {
			t.Fatalf("nnz mb[%d][0] = %d, want luma residual", mbXY, m.NonZeroCount[mbXY][0])
		}
	}
	if payload.bitsLeft() != 0 {
		t.Fatalf("residual P frame payload bitsLeft = %d, want 0", payload.bitsLeft())
	}
}

func TestEncodeI420P16x16ResidualSliceRBSPDecodesChromaDCThroughFramePath(t *testing.T) {
	pps, sps := encoderResidualSliceTestPPS(20)
	rbsp, err := encodeI420P16x16ResidualSliceRBSP(encoderI420P16x16ResidualConfig{
		Width:                      16,
		Height:                     16,
		FrameNum:                   10,
		InitialQP:                  20,
		NextQP:                     23,
		DisableDeblockingFilterIDC: 1,
		MVDX:                       2,
		MVDY:                       -1,
		Coeff:                      1,
		ChromaDCCoeffCb:            1,
		ChromaDCCoeffCr:            -1,
	}, pps, sps)
	if err != nil {
		t.Fatalf("encode residual chroma DC slice rbsp: %v", err)
	}

	var ppsList [maxPPSCount]*PPS
	ppsList[0] = pps
	sh, payload, err := parseSliceHeaderWithPayload(NALUnit{Type: NALSlice, RefIDC: 2, RBSP: rbsp}, &ppsList)
	if err != nil {
		t.Fatalf("parse residual chroma DC P slice header: %v", err)
	}
	skipRun, err := payload.readUEGolombLong()
	if err != nil {
		t.Fatalf("read generated residual chroma DC P skip run: %v", err)
	}
	if skipRun != 0 {
		t.Fatalf("skip run = %d, want 0", skipRun)
	}
	var decoded cavlcResidualContext
	got, err := decoded.decodeCAVLCInterPMacroblock(&payload, pps, sps, int(sh.QScale), [2]uint32{1, 0}, false)
	if err != nil {
		t.Fatalf("decode generated residual chroma DC P macroblock: %v", err)
	}
	if got.MBType != (MBType16x16|MBTypeP0L0) || got.CBP != 0x11 ||
		got.QScale != 23 || got.ChromaQP != ([2]uint8{23, 23}) || got.CBPTable != 0x1011 {
		t.Fatalf("decoded chroma DC mb type/cbp/q/chroma/cbpTable = %#x/%#x/%d/%v/%#x",
			got.MBType, got.CBP, got.QScale, got.ChromaQP, got.CBPTable)
	}
	if decoded.MB[0] != 1 || decoded.MB[256] != 1 || decoded.MB[512] != -1 {
		t.Fatalf("decoded residual luma/chroma = %d/%d/%d, want 1/1/-1", decoded.MB[0], decoded.MB[256], decoded.MB[512])
	}
	if payload.bitsLeft() != 0 {
		t.Fatalf("residual chroma DC payload bitsLeft = %d, want 0", payload.bitsLeft())
	}

	sh, payload, err = parseSliceHeaderWithPayload(NALUnit{Type: NALSlice, RefIDC: 2, RBSP: rbsp}, &ppsList)
	if err != nil {
		t.Fatalf("reparse residual chroma DC P slice header: %v", err)
	}
	m, err := newMacroblockTables(1, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	state := newCAVLCFrameSliceState(int(sh.QScale))
	frameGot, err := m.decodeCAVLCFrameSliceMacroblock(&payload, sh, &state, 0, 13)
	if err != nil {
		t.Fatalf("decode generated residual chroma DC P frame macroblock: %v", err)
	}
	if frameGot.CBP != 0x11 || frameGot.CBPTable != 0x1011 || frameGot.QScale != 23 ||
		m.CBPTable[0] != 0x1011 || m.QScaleTable[0] != 23 || m.SliceTable[0] != 13 {
		t.Fatalf("frame chroma DC result/table cbp/cbpTable/q = %#x/%#x/%d table %#x/%d/%d",
			frameGot.CBP, frameGot.CBPTable, frameGot.QScale, m.CBPTable[0], m.QScaleTable[0], m.SliceTable[0])
	}
	if m.NonZeroCount[0][0] != 1 {
		t.Fatalf("frame chroma DC luma nnz = %d, want 1", m.NonZeroCount[0][0])
	}
}

func TestEncodeI420P16x16ResidualSliceRBSPDecodesPerMacroblockChromaDC(t *testing.T) {
	pps, sps := encoderResidualSliceTestPPS(20)
	wantMVDs := []EncoderMotionVectorDelta{
		{X: 2, Y: -1},
		{X: -3, Y: 4},
	}
	wantCoeffs := []int32{1, -2}
	wantChromaDC := [][2]int32{
		{1, -1},
		{-1, 1},
	}
	rbsp, err := encodeI420P16x16ResidualSliceRBSP(encoderI420P16x16ResidualConfig{
		Width:                      32,
		Height:                     16,
		FrameNum:                   11,
		InitialQP:                  20,
		NextQP:                     23,
		DisableDeblockingFilterIDC: 1,
		MVDs:                       wantMVDs,
		Coeffs:                     wantCoeffs,
		ChromaDCCoeffs:             wantChromaDC,
	}, pps, sps)
	if err != nil {
		t.Fatalf("encode per-macroblock residual chroma DC slice rbsp: %v", err)
	}

	var ppsList [maxPPSCount]*PPS
	ppsList[0] = pps
	sh, payload, err := parseSliceHeaderWithPayload(NALUnit{Type: NALSlice, RefIDC: 2, RBSP: rbsp}, &ppsList)
	if err != nil {
		t.Fatalf("parse per-macroblock residual chroma DC P slice header: %v", err)
	}
	qscale := int(sh.QScale)
	for i := range wantMVDs {
		skipRun, err := payload.readUEGolombLong()
		if err != nil {
			t.Fatalf("read per-macroblock residual chroma DC skip run[%d]: %v", i, err)
		}
		if skipRun != 0 {
			t.Fatalf("skip run[%d] = %d, want 0", i, skipRun)
		}
		var decoded cavlcResidualContext
		got, err := decoded.decodeCAVLCInterPMacroblock(&payload, pps, sps, qscale, [2]uint32{1, 0}, false)
		if err != nil {
			t.Fatalf("decode per-macroblock residual chroma DC macroblock[%d]: %v", i, err)
		}
		if got.MBType != (MBType16x16|MBTypeP0L0) || got.CBP != 0x11 ||
			got.QScale != 23 || got.ChromaQP != ([2]uint8{23, 23}) || got.CBPTable != 0x1011 {
			t.Fatalf("decoded chroma DC mb[%d] type/cbp/q/chroma/cbpTable = %#x/%#x/%d/%v/%#x",
				i, got.MBType, got.CBP, got.QScale, got.ChromaQP, got.CBPTable)
		}
		if got.MVD[0][0] != ([2]int32{wantMVDs[i].X, wantMVDs[i].Y}) ||
			decoded.MB[0] != wantCoeffs[i] ||
			decoded.MB[256] != wantChromaDC[i][0] ||
			decoded.MB[512] != wantChromaDC[i][1] {
			t.Fatalf("decoded chroma DC mb[%d] motion/residual = %v/%d/%d/%d",
				i, got.MVD[0][0], decoded.MB[0], decoded.MB[256], decoded.MB[512])
		}
		qscale = got.QScale
	}
	if payload.bitsLeft() != 0 {
		t.Fatalf("per-macroblock residual chroma DC payload bitsLeft = %d, want 0", payload.bitsLeft())
	}

	sh, payload, err = parseSliceHeaderWithPayload(NALUnit{Type: NALSlice, RefIDC: 2, RBSP: rbsp}, &ppsList)
	if err != nil {
		t.Fatalf("reparse per-macroblock residual chroma DC P slice header: %v", err)
	}
	m, err := newMacroblockTables(2, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	state := newCAVLCFrameSliceState(int(sh.QScale))
	for mbXY := 0; mbXY < 2; mbXY++ {
		got, err := m.decodeCAVLCFrameSliceMacroblock(&payload, sh, &state, mbXY, 14)
		if err != nil {
			t.Fatalf("decode per-macroblock residual chroma DC frame macroblock[%d]: %v", mbXY, err)
		}
		if got.CBP != 0x11 || got.CBPTable != 0x1011 || got.QScale != 23 ||
			m.CBPTable[mbXY] != 0x1011 || m.QScaleTable[mbXY] != 23 || m.SliceTable[mbXY] != 14 {
			t.Fatalf("frame chroma DC mb[%d] result/table cbp/cbpTable/q = %#x/%#x/%d table %#x/%d/%d",
				mbXY, got.CBP, got.CBPTable, got.QScale, m.CBPTable[mbXY], m.QScaleTable[mbXY], m.SliceTable[mbXY])
		}
	}
	if payload.bitsLeft() != 0 {
		t.Fatalf("per-macroblock frame residual chroma DC payload bitsLeft = %d, want 0", payload.bitsLeft())
	}
}

func TestEncodeI420P16x16ResidualSliceRBSPRejectsInvalid(t *testing.T) {
	pps, sps := encoderResidualSliceTestPPS(20)
	valid := encoderI420P16x16ResidualConfig{
		Width:                      16,
		Height:                     16,
		InitialQP:                  20,
		NextQP:                     20,
		DisableDeblockingFilterIDC: 1,
		Coeff:                      1,
	}
	for _, tt := range []struct {
		name string
		run  func() error
	}{
		{name: "nil pps", run: func() error {
			_, err := encodeI420P16x16ResidualSliceRBSP(valid, nil, sps)
			return err
		}},
		{name: "nil sps", run: func() error {
			_, err := encodeI420P16x16ResidualSliceRBSP(valid, pps, nil)
			return err
		}},
		{name: "bad qp", run: func() error {
			next := valid
			next.NextQP = 52
			_, err := encodeI420P16x16ResidualSliceRBSP(next, pps, sps)
			return err
		}},
		{name: "zero coeff", run: func() error {
			next := valid
			next.Coeff = 0
			_, err := encodeI420P16x16ResidualSliceRBSP(next, pps, sps)
			return err
		}},
		{name: "bad mvd count", run: func() error {
			next := valid
			next.Width = 32
			next.MVDs = []EncoderMotionVectorDelta{{X: 1, Y: 0}}
			_, err := encodeI420P16x16ResidualSliceRBSP(next, pps, sps)
			return err
		}},
		{name: "bad coeff count", run: func() error {
			next := valid
			next.Width = 32
			next.Coeffs = []int32{1}
			_, err := encodeI420P16x16ResidualSliceRBSP(next, pps, sps)
			return err
		}},
		{name: "zero per-macroblock coeff", run: func() error {
			next := valid
			next.Width = 32
			next.Coeff = 0
			next.Coeffs = []int32{1, 0}
			_, err := encodeI420P16x16ResidualSliceRBSP(next, pps, sps)
			return err
		}},
		{name: "partial chroma dc scalar", run: func() error {
			next := valid
			next.ChromaDCCoeffCb = 1
			_, err := encodeI420P16x16ResidualSliceRBSP(next, pps, sps)
			return err
		}},
		{name: "bad chroma dc count", run: func() error {
			next := valid
			next.Width = 32
			next.ChromaDCCoeffs = [][2]int32{{1, -1}}
			_, err := encodeI420P16x16ResidualSliceRBSP(next, pps, sps)
			return err
		}},
		{name: "partial per-macroblock chroma dc", run: func() error {
			next := valid
			next.Width = 32
			next.ChromaDCCoeffs = [][2]int32{{1, -1}, {1, 0}}
			_, err := encodeI420P16x16ResidualSliceRBSP(next, pps, sps)
			return err
		}},
		{name: "bad range", run: func() error {
			next := valid
			next.FirstMBAddr = 1
			_, err := encodeI420P16x16ResidualSliceRBSP(next, pps, sps)
			return err
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.run(); err != ErrInvalidData {
				t.Fatalf("encode residual slice error = %v, want ErrInvalidData", err)
			}
		})
	}
}

func encoderResidualSliceTestPPS(initQP int) (*PPS, *SPS) {
	pps := cavlcFlatQMulPPS()
	sps := &SPS{
		BitDepthLuma:           8,
		ChromaFormatIDC:        1,
		Log2MaxFrameNum:        8,
		FrameMBSOnlyFlag:       1,
		Direct8x8InferenceFlag: 1,
	}
	pps.SPS = sps
	pps.RefCount = [2]uint32{1, 1}
	pps.InitQP = int32(initQP)
	pps.DeblockingFilterParametersPresent = 1
	return pps, sps
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
		{name: "macroblock count overflow", mutate: func(c *EncoderI420PSkipConfig) {
			c.Width = maxInt - 15
			c.Height = 32
		}},
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
		{name: "macroblock count overflow", mutate: func(c *EncoderI420P16x16NoResidualConfig) {
			c.Width = maxInt - 15
			c.Height = 32
		}},
		{name: "bad macroblock count", mutate: func(c *EncoderI420P16x16NoResidualConfig) { c.MacroblockCount = 2 }},
		{name: "bad mvd count", mutate: func(c *EncoderI420P16x16NoResidualConfig) {
			c.MVDs = []EncoderMotionVectorDelta{{}, {}}
		}},
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
		{name: "luma size overflow", mutate: func(c *EncoderI420IntraPCMPConfig) {
			c.Width = maxInt - 15
			c.Height = 32
			c.StrideY = c.Width
			c.StrideCb = c.Width / 2
			c.StrideCr = c.Width / 2
			c.Y = nil
			c.Cb = nil
			c.Cr = nil
		}},
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
		{name: "luma size overflow", mutate: func(c *EncoderI420IntraPCMIDRConfig) {
			c.Width = maxInt - 15
			c.Height = 32
			c.StrideY = c.Width
			c.StrideCb = c.Width / 2
			c.StrideCr = c.Width / 2
			c.Y = nil
			c.Cb = nil
			c.Cr = nil
		}},
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
	wantMVDs := make([]EncoderMotionVectorDelta, macroblockCount)
	for i := range wantMVDs {
		wantMVDs[i] = EncoderMotionVectorDelta{X: wantMVDX, Y: wantMVDY}
	}
	assertEncoderP16x16NoResidualPayloadMVDs(t, payload, wantMVDs)
}

func assertEncoderP16x16NoResidualPayloadMVDs(t *testing.T, payload *bitReader, wantMVDs []EncoderMotionVectorDelta) {
	t.Helper()
	for i, wantMVD := range wantMVDs {
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
		if skipRun != 0 || mbType != 0 || mvdX != wantMVD.X || mvdY != wantMVD.Y || cbp != 0 {
			t.Fatalf("P16x16 macroblock[%d] skip/mb/mvd/cbp = %d/%d/%d,%d/%d, want 0/0/%d,%d/0",
				i, skipRun, mbType, mvdX, mvdY, cbp, wantMVD.X, wantMVD.Y)
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
