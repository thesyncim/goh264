// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

import "testing"

func TestDecodeCAVLCFrameSliceReconstructsIntraPCMRun(t *testing.T) {
	m, err := newMacroblockTables(2, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	sps := &SPS{BitDepthLuma: 8, ChromaFormatIDC: 1, FrameMBSOnlyFlag: 1}
	pps := cavlcFlatQMulPPS()
	pps.SPS = sps
	sh := &SliceHeader{
		FirstMBAddr:      0,
		SliceType:        PictureTypeI,
		SliceTypeNoS:     PictureTypeI,
		PictureStructure: PictureFrame,
		PPS:              pps,
		SPS:              sps,
		QScale:           20,
		DeblockingFilter: 0,
	}
	dst := makeH264SliceDecodePicture(2, 1, 1)
	pcm0 := h264ReconstructIntraPCM(1, 11)
	pcm1 := h264ReconstructIntraPCM(1, 23)
	gb := newBitReader(append(cavlcIntraPCMBytes(pcm0), cavlcIntraPCMBytes(pcm1)...))

	got, err := m.decodeCAVLCFrameSlice(&gb, dst, sh, h264FrameSliceDecodeInput{SliceNum: 9})
	if err != nil {
		t.Fatalf("decode cavlc slice failed: %v", err)
	}
	if got.Macroblocks != 2 || got.LastMBXY != 1 || !got.EndOfSlice || !got.EndOfFrame {
		t.Fatalf("slice result = %+v, want 2 MBs ending at mb_xy 1 and frame end", got)
	}
	assertH264SliceDecodePCM(t, dst, 0, 0, pcm0)
	assertH264SliceDecodePCM(t, dst, 1, 0, pcm1)
	for _, mbXY := range []int{0, 1} {
		if m.MacroblockTyp[mbXY] != MBTypeIntraPCM || m.CBPTable[mbXY] != 0 || m.QScaleTable[mbXY] != 0 || m.SliceTable[mbXY] != 9 {
			t.Fatalf("tables[%d] type/cbp/q/slice = %#x/%#x/%d/%d", mbXY, m.MacroblockTyp[mbXY], m.CBPTable[mbXY], m.QScaleTable[mbXY], m.SliceTable[mbXY])
		}
	}
	if gb.bitsLeft() != 0 {
		t.Fatalf("bits left = %d, want 0", gb.bitsLeft())
	}
}

func TestDecodeCAVLCFrameSliceReconstructsFieldPictureIntraPCM(t *testing.T) {
	for _, tt := range []struct {
		name       string
		picture    int32
		pcmSeed    int
		wantLastXY func(*macroblockTables) int
	}{
		{name: "top", picture: PictureTopField, pcmSeed: 31, wantLastXY: func(*macroblockTables) int { return 0 }},
		{name: "bottom", picture: PictureBottomField, pcmSeed: 32, wantLastXY: func(m *macroblockTables) int { return m.MBStride }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			m, err := newMacroblockTables(1, 2, 1)
			if err != nil {
				t.Fatal(err)
			}
			sps := &SPS{BitDepthLuma: 8, ChromaFormatIDC: 1, FrameMBSOnlyFlag: 0}
			pps := cavlcFlatQMulPPS()
			pps.SPS = sps
			sh := &SliceHeader{
				FirstMBAddr:      0,
				SliceType:        PictureTypeI,
				SliceTypeNoS:     PictureTypeI,
				PictureStructure: tt.picture,
				PPS:              pps,
				SPS:              sps,
				QScale:           20,
				DeblockingFilter: 0,
			}
			dst := makeH264SliceDecodePicture(1, 2, 1)
			pcm := h264ReconstructIntraPCM(1, tt.pcmSeed)
			gb := newBitReader(cavlcIntraPCMBytes(pcm))

			got, err := m.decodeCAVLCFrameSlice(&gb, dst, sh, h264FrameSliceDecodeInput{SliceNum: 19})
			if err != nil {
				t.Fatalf("decode cavlc field slice failed: %v", err)
			}
			if got.Macroblocks != 1 || got.LastMBXY != tt.wantLastXY(m) || !got.EndOfSlice || !got.EndOfFrame {
				t.Fatalf("slice result = %+v, want one field MB ending at xy %d", got, tt.wantLastXY(m))
			}
			dstView := *dst
			applySimpleFieldRefPlane(&dstView, tt.picture)
			assertH264SliceDecodePCM(t, &dstView, 0, 0, pcm)
		})
	}
}

func TestDecodeFrameSliceDataDispatchesCAVLC(t *testing.T) {
	m, err := newMacroblockTables(1, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	sps := &SPS{BitDepthLuma: 8, ChromaFormatIDC: 1, FrameMBSOnlyFlag: 1}
	pps := cavlcFlatQMulPPS()
	pps.SPS = sps
	sh := &SliceHeader{
		FirstMBAddr:      0,
		SliceType:        PictureTypeI,
		SliceTypeNoS:     PictureTypeI,
		PictureStructure: PictureFrame,
		PPS:              pps,
		SPS:              sps,
		QScale:           20,
		DeblockingFilter: 0,
	}
	dst := makeH264SliceDecodePicture(1, 1, 1)
	pcm := h264ReconstructIntraPCM(1, 17)
	gb := newBitReader(cavlcIntraPCMBytes(pcm))

	got, err := m.decodeFrameSliceData(&gb, dst, sh, h264FrameSliceDecodeInput{SliceNum: 7})
	if err != nil {
		t.Fatalf("decode dispatched cavlc slice failed: %v", err)
	}
	if got.Macroblocks != 1 || !got.EndOfFrame || !got.EndOfSlice {
		t.Fatalf("slice result = %+v, want one-MB frame end", got)
	}
	assertH264SliceDecodePCM(t, dst, 0, 0, pcm)
}

func TestDecodeCABACFrameSliceMBAFFReconstructsFrameCodedPCMPair(t *testing.T) {
	m, err := newMacroblockTables(1, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	sps := &SPS{BitDepthLuma: 8, ChromaFormatIDC: 1, FrameMBSOnlyFlag: 0, MBAFF: 1}
	pps := cavlcFlatQMulPPS()
	pps.SPS = sps
	pps.CABAC = 1
	sh := &SliceHeader{
		FirstMBAddr:      0,
		SliceType:        PictureTypeI,
		SliceTypeNoS:     PictureTypeI,
		PictureStructure: PictureFrame,
		PPS:              pps,
		SPS:              sps,
		QScale:           20,
		DeblockingFilter: 0,
	}
	dst := makeH264SliceDecodePicture(1, 2, 1)
	pcm0 := h264ReconstructIntraPCM(1, 53)
	pcm1 := h264ReconstructIntraPCM(1, 67)
	src := &scriptedCABACSource{
		bits:  []int{0, 1, 1},
		terms: []int{1, 1, 1},
		pcm:   append(append([]byte(nil), pcm0...), pcm1...),
	}

	got, err := m.decodeCABACFrameSlice(src, dst, sh, h264FrameSliceDecodeInput{SliceNum: 12})
	if err != nil {
		t.Fatalf("decode cabac mbaff frame-coded pcm pair failed: %v", err)
	}
	bottomXY := m.MBStride
	if got.Macroblocks != 2 || got.LastMBXY != bottomXY || !got.EndOfSlice || !got.EndOfFrame {
		t.Fatalf("slice result = %+v, want 2 MBs ending at bottom xy %d and frame end", got, bottomXY)
	}
	assertH264SliceDecodePCM(t, dst, 0, 0, pcm0)
	assertH264SliceDecodePCM(t, dst, 0, 1, pcm1)
	for _, mbXY := range []int{0, bottomXY} {
		if m.MacroblockTyp[mbXY] != MBTypeIntraPCM || m.CBPTable[mbXY] != 0xf7ef || m.QScaleTable[mbXY] != 0 || m.SliceTable[mbXY] != 12 {
			t.Fatalf("tables[%d] type/cbp/q/slice = %#x/%#x/%d/%d", mbXY, m.MacroblockTyp[mbXY], m.CBPTable[mbXY], m.QScaleTable[mbXY], m.SliceTable[mbXY])
		}
	}
	if len(src.bits) != 0 || len(src.terms) != 0 || len(src.pcm) != 0 {
		t.Fatalf("script leftovers bits=%d terms=%d pcm=%d, want none", len(src.bits), len(src.terms), len(src.pcm))
	}
}

func TestDecodeCABACFrameSliceReconstructsFieldPictureIntraPCM(t *testing.T) {
	for _, tt := range []struct {
		name       string
		picture    int32
		pcmSeed    int
		wantLastXY func(*macroblockTables) int
	}{
		{name: "top", picture: PictureTopField, pcmSeed: 47, wantLastXY: func(*macroblockTables) int { return 0 }},
		{name: "bottom", picture: PictureBottomField, pcmSeed: 48, wantLastXY: func(m *macroblockTables) int { return m.MBStride }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			m, err := newMacroblockTables(1, 2, 1)
			if err != nil {
				t.Fatal(err)
			}
			sps := &SPS{BitDepthLuma: 8, ChromaFormatIDC: 1, FrameMBSOnlyFlag: 0}
			pps := cavlcFlatQMulPPS()
			pps.SPS = sps
			pps.CABAC = 1
			sh := &SliceHeader{
				FirstMBAddr:      0,
				SliceType:        PictureTypeI,
				SliceTypeNoS:     PictureTypeI,
				PictureStructure: tt.picture,
				PPS:              pps,
				SPS:              sps,
				QScale:           20,
				DeblockingFilter: 0,
			}
			dst := makeH264SliceDecodePicture(1, 2, 1)
			pcm := h264ReconstructIntraPCM(1, tt.pcmSeed)
			src := &scriptedCABACSource{
				bits:  []int{1},
				terms: []int{1, 1},
				pcm:   append([]byte(nil), pcm...),
			}

			got, err := m.decodeCABACFrameSlice(src, dst, sh, h264FrameSliceDecodeInput{SliceNum: 20})
			if err != nil {
				t.Fatalf("decode cabac field slice failed: %v", err)
			}
			if got.Macroblocks != 1 || got.LastMBXY != tt.wantLastXY(m) || !got.EndOfSlice || !got.EndOfFrame {
				t.Fatalf("slice result = %+v, want one field MB ending at xy %d", got, tt.wantLastXY(m))
			}
			dstView := *dst
			applySimpleFieldRefPlane(&dstView, tt.picture)
			assertH264SliceDecodePCM(t, &dstView, 0, 0, pcm)
			if len(src.bits) != 0 || len(src.pcm) != 0 {
				t.Fatalf("script leftovers bits=%d pcm=%d, want none", len(src.bits), len(src.pcm))
			}
		})
	}
}

func TestDecodeFrameSliceDataDispatchesCABACStartup(t *testing.T) {
	m, err := newMacroblockTables(1, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	sps := &SPS{BitDepthLuma: 8, ChromaFormatIDC: 1, FrameMBSOnlyFlag: 1}
	pps := cavlcFlatQMulPPS()
	pps.SPS = sps
	pps.CABAC = 1
	sh := &SliceHeader{
		FirstMBAddr:      0,
		SliceType:        PictureTypeI,
		SliceTypeNoS:     PictureTypeI,
		PictureStructure: PictureFrame,
		PPS:              pps,
		SPS:              sps,
		QScale:           20,
		DeblockingFilter: 0,
	}
	dst := makeH264SliceDecodePicture(1, 1, 1)
	gb := newBitReader([]byte{0xe0, 0x2a})
	if _, err := gb.readBits(3); err != nil {
		t.Fatal(err)
	}

	_, err = m.decodeFrameSliceData(&gb, dst, sh, h264FrameSliceDecodeInput{SliceNum: 7})
	if err != ErrInvalidData {
		t.Fatalf("err = %v, want ErrInvalidData from CABAC startup", err)
	}
	if gb.bitPos != 8 {
		t.Fatalf("bitPos = %d, want CABAC byte realignment", gb.bitPos)
	}
	if m.MacroblockTyp[0] != 0 || m.SliceTable[0] != 0xffff {
		t.Fatalf("tables type/slice = %#x/%#x, want untouched", m.MacroblockTyp[0], m.SliceTable[0])
	}
}

func TestDecodeCABACFrameSliceReconstructsIntraPCMAndEOS(t *testing.T) {
	m, err := newMacroblockTables(1, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	sps := &SPS{BitDepthLuma: 8, ChromaFormatIDC: 1, FrameMBSOnlyFlag: 1}
	pps := cavlcFlatQMulPPS()
	pps.SPS = sps
	pps.CABAC = 1
	sh := &SliceHeader{
		FirstMBAddr:      0,
		SliceType:        PictureTypeI,
		SliceTypeNoS:     PictureTypeI,
		PictureStructure: PictureFrame,
		PPS:              pps,
		SPS:              sps,
		QScale:           23,
		DeblockingFilter: 0,
	}
	dst := makeH264SliceDecodePicture(1, 1, 1)
	pcm := h264ReconstructIntraPCM(1, 37)
	src := &scriptedCABACSource{
		bits:  []int{1},
		terms: []int{1, 1},
		pcm:   append([]byte(nil), pcm...),
	}

	got, err := m.decodeCABACFrameSlice(src, dst, sh, h264FrameSliceDecodeInput{SliceNum: 13})
	if err != nil {
		t.Fatalf("decode cabac slice failed: %v", err)
	}
	if got.Macroblocks != 1 || got.LastMBXY != 0 || !got.EndOfSlice || !got.EndOfFrame {
		t.Fatalf("slice result = %+v, want one-MB frame end", got)
	}
	assertH264SliceDecodePCM(t, dst, 0, 0, pcm)
	if m.MacroblockTyp[0] != MBTypeIntraPCM || m.CBPTable[0] != 0xf7ef || m.QScaleTable[0] != 0 || m.SliceTable[0] != 13 {
		t.Fatalf("tables type/cbp/q/slice = %#x/%#x/%d/%d", m.MacroblockTyp[0], m.CBPTable[0], m.QScaleTable[0], m.SliceTable[0])
	}
	if len(src.pcmReadSizes) != 1 || src.pcmReadSizes[0] != len(pcm) {
		t.Fatalf("pcm read sizes = %v, want [%d]", src.pcmReadSizes, len(pcm))
	}
	wantIndexes(t, src, []int{3})
}

func TestInitCABACFrameSliceDecoderAlignsAndInitializesStates(t *testing.T) {
	sps := &SPS{BitDepthLuma: 8, ChromaFormatIDC: 1, FrameMBSOnlyFlag: 1}
	pps := cavlcFlatQMulPPS()
	pps.SPS = sps
	pps.CABAC = 1
	sh := &SliceHeader{
		SliceType:        PictureTypeP,
		SliceTypeNoS:     PictureTypeP,
		PictureStructure: PictureFrame,
		PPS:              pps,
		SPS:              sps,
		CABACInitIDC:     2,
		QScale:           31,
	}
	gb := newBitReader([]byte{0xe0, 0x2a, 0x40, 0x80, 0x11})
	gb.numBits = 35
	if _, err := gb.readBits(3); err != nil {
		t.Fatal(err)
	}

	got, err := initCABACFrameSliceDecoder(&gb, sh)
	if err != nil {
		t.Fatalf("init cabac frame slice decoder failed: %v", err)
	}
	wantCABAC, err := initCABACDecoder(gb.buf[1:])
	if err != nil {
		t.Fatal(err)
	}
	wantState, err := initH264CABACStates(PictureTypeP, 2, 31, 8)
	if err != nil {
		t.Fatal(err)
	}
	if gb.bitPos != 8 {
		t.Fatalf("bitPos = %d, want aligned 8", gb.bitPos)
	}
	if got.cabac.low != wantCABAC.low || got.cabac.rng != wantCABAC.rng || got.cabac.bytestream != wantCABAC.bytestream || got.cabac.bytestreamEnd != wantCABAC.bytestreamEnd {
		t.Fatalf("cabac ctx = low %#x range %#x byte %d end %d, want low %#x range %#x byte %d end %d", got.cabac.low, got.cabac.rng, got.cabac.bytestream, got.cabac.bytestreamEnd, wantCABAC.low, wantCABAC.rng, wantCABAC.bytestream, wantCABAC.bytestreamEnd)
	}
	if got.state[0] != wantState[0] || got.state[60] != wantState[60] || got.state[399] != wantState[399] {
		t.Fatalf("cabac states = %d/%d/%d, want %d/%d/%d", got.state[0], got.state[60], got.state[399], wantState[0], wantState[60], wantState[399])
	}
}

func TestDecodeCAVLCFrameSliceReconstructsPSkip(t *testing.T) {
	m, err := newMacroblockTables(1, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	sps := &SPS{BitDepthLuma: 8, ChromaFormatIDC: 1, FrameMBSOnlyFlag: 1}
	pps := cavlcFlatQMulPPS()
	pps.SPS = sps
	sh := &SliceHeader{
		FirstMBAddr:      0,
		SliceType:        PictureTypeP,
		SliceTypeNoS:     PictureTypeP,
		PictureStructure: PictureFrame,
		PPS:              pps,
		SPS:              sps,
		QScale:           24,
		RefCount:         [2]uint32{1, 0},
		DeblockingFilter: 0,
	}
	dst := makeH264SliceDecodePicture(1, 1, 1)
	ref := makeH264SliceDecodePicture(1, 1, 1)
	fillH264MotionCompPlane(ref.Y, 73)
	fillH264MotionCompPlane(ref.Cb, 91)
	fillH264MotionCompPlane(ref.Cr, 119)
	gb := newBitReader(cavlcBitString("010"))

	got, err := m.decodeCAVLCFrameSlice(&gb, dst, sh, h264FrameSliceDecodeInput{
		SliceNum:      3,
		Refs:          [2][]*h264PicturePlanes{{ref}},
		MotionScratch: makeH264MotionCompScratch(dst),
	})
	if err != nil {
		t.Fatalf("decode cavlc pskip slice failed: %v", err)
	}
	if got.Macroblocks != 1 || got.LastMBXY != 0 || !got.EndOfSlice || !got.EndOfFrame {
		t.Fatalf("slice result = %+v, want one skipped MB frame end", got)
	}
	assertH264Rows(t, "pskip y", dst.Y, 0, dst.LumaStride, 16, 16, ref.Y, ref.LumaStride)
	assertH264Rows(t, "pskip cb", dst.Cb, 0, dst.ChromaStride, 8, 8, ref.Cb, ref.ChromaStride)
	assertH264Rows(t, "pskip cr", dst.Cr, 0, dst.ChromaStride, 8, 8, ref.Cr, ref.ChromaStride)
	wantType := MBType16x16 | MBTypeP0L0 | MBTypeP1L0 | MBTypeSkip
	if m.MacroblockTyp[0] != wantType || m.QScaleTable[0] != 24 || m.SliceTable[0] != 3 {
		t.Fatalf("tables type/q/slice = %#x/%d/%d", m.MacroblockTyp[0], m.QScaleTable[0], m.SliceTable[0])
	}
}

func TestDecodeCAVLCFrameSliceHighReconstructsPSkipFromRef(t *testing.T) {
	const bitDepth = 10
	m, dst, sh, ref := h264HighPFrameSliceDecodeFixture(t, bitDepth)
	gb := newBitReader(cavlcBitString("010"))

	got, err := m.decodeCAVLCFrameSliceHigh(&gb, dst, sh, h264FrameSliceDecodeInputHigh{
		SliceNum:      21,
		Refs:          [2][]*h264PicturePlanesHigh{{ref}},
		MotionScratch: makeH264MotionCompScratchHigh(dst),
	})
	if err != nil {
		t.Fatalf("decode high cavlc pskip slice failed: %v", err)
	}
	if got.Macroblocks != 1 || got.LastMBXY != 0 || !got.EndOfSlice || !got.EndOfFrame {
		t.Fatalf("slice result = %+v, want one skipped MB frame end", got)
	}
	assertH264SliceDecodeHighRef(t, "cavlc high pskip", dst, ref)
	wantType := MBType16x16 | MBTypeP0L0 | MBTypeP1L0 | MBTypeSkip
	if m.MacroblockTyp[0] != wantType || m.CBPTable[0] != 0 || m.QScaleTable[0] != uint8(sh.QScale) || m.SliceTable[0] != 21 {
		t.Fatalf("tables type/cbp/q/slice = %#x/%#x/%d/%d", m.MacroblockTyp[0], m.CBPTable[0], m.QScaleTable[0], m.SliceTable[0])
	}
	if gb.bitPos != 3 {
		t.Fatalf("consumed %d bits, want 3", gb.bitPos)
	}
}

func TestDecodeCABACFrameSliceHighReconstructsPSkipFromRef(t *testing.T) {
	const bitDepth = 10
	m, dst, sh, ref := h264HighPFrameSliceDecodeFixture(t, bitDepth)
	sh.PPS.CABAC = 1
	src := &scriptedCABACSource{
		bits:  []int{1},
		terms: []int{1},
	}

	got, err := m.decodeCABACFrameSliceHigh(src, dst, sh, h264FrameSliceDecodeInputHigh{
		SliceNum:      22,
		Refs:          [2][]*h264PicturePlanesHigh{{ref}},
		MotionScratch: makeH264MotionCompScratchHigh(dst),
	})
	if err != nil {
		t.Fatalf("decode high cabac pskip slice failed: %v", err)
	}
	if got.Macroblocks != 1 || got.LastMBXY != 0 || !got.EndOfSlice || !got.EndOfFrame {
		t.Fatalf("slice result = %+v, want one skipped MB frame end", got)
	}
	assertH264SliceDecodeHighRef(t, "cabac high pskip", dst, ref)
	wantType := MBType16x16 | MBTypeP0L0 | MBTypeP1L0 | MBTypeSkip
	if m.MacroblockTyp[0] != wantType || m.CBPTable[0] != 0 || m.QScaleTable[0] != uint8(sh.QScale) || m.SliceTable[0] != 22 {
		t.Fatalf("tables type/cbp/q/slice = %#x/%#x/%d/%d", m.MacroblockTyp[0], m.CBPTable[0], m.QScaleTable[0], m.SliceTable[0])
	}
	wantIndexes(t, src, []int{11})
}

func TestDecodeCAVLCFrameSliceHighReconstructsP16x16NoResidualFromRef(t *testing.T) {
	const bitDepth = 10
	m, dst, sh, ref := h264HighPFrameSliceDecodeFixture(t, bitDepth)
	gb := newBitReader(cavlcBitString("11111"))

	got, err := m.decodeCAVLCFrameSliceHigh(&gb, dst, sh, h264FrameSliceDecodeInputHigh{
		SliceNum:      23,
		Refs:          [2][]*h264PicturePlanesHigh{{ref}},
		MotionScratch: makeH264MotionCompScratchHigh(dst),
	})
	if err != nil {
		t.Fatalf("decode high cavlc p16x16 slice failed: %v", err)
	}
	if got.Macroblocks != 1 || got.LastMBXY != 0 || !got.EndOfSlice || !got.EndOfFrame {
		t.Fatalf("slice result = %+v, want one P16x16 MB frame end", got)
	}
	assertH264SliceDecodeHighRef(t, "cavlc high p16x16", dst, ref)
	wantType := MBType16x16 | MBTypeP0L0
	if m.MacroblockTyp[0] != wantType || m.CBPTable[0] != 0 || m.QScaleTable[0] != uint8(sh.QScale) || m.SliceTable[0] != 23 {
		t.Fatalf("tables type/cbp/q/slice = %#x/%#x/%d/%d", m.MacroblockTyp[0], m.CBPTable[0], m.QScaleTable[0], m.SliceTable[0])
	}
	if gb.bitPos != 5 {
		t.Fatalf("consumed %d bits, want 5", gb.bitPos)
	}
}

func TestDecodeCABACFrameSliceHighReconstructsP16x16NoResidualFromRef(t *testing.T) {
	const bitDepth = 10
	m, dst, sh, ref := h264HighPFrameSliceDecodeFixture(t, bitDepth)
	sh.PPS.CABAC = 1
	src := &scriptedCABACSource{
		bits: []int{
			0,
			0, 0, 0,
			0, 0,
			0, 0, 0, 0,
			0,
		},
		terms: []int{1},
	}

	got, err := m.decodeCABACFrameSliceHigh(src, dst, sh, h264FrameSliceDecodeInputHigh{
		SliceNum:      24,
		Refs:          [2][]*h264PicturePlanesHigh{{ref}},
		MotionScratch: makeH264MotionCompScratchHigh(dst),
	})
	if err != nil {
		t.Fatalf("decode high cabac p16x16 slice failed: %v", err)
	}
	if got.Macroblocks != 1 || got.LastMBXY != 0 || !got.EndOfSlice || !got.EndOfFrame {
		t.Fatalf("slice result = %+v, want one P16x16 MB frame end", got)
	}
	assertH264SliceDecodeHighRef(t, "cabac high p16x16", dst, ref)
	wantType := MBType16x16 | MBTypeP0L0
	if m.MacroblockTyp[0] != wantType || m.CBPTable[0] != 0 || m.QScaleTable[0] != uint8(sh.QScale) || m.SliceTable[0] != 24 {
		t.Fatalf("tables type/cbp/q/slice = %#x/%#x/%d/%d", m.MacroblockTyp[0], m.CBPTable[0], m.QScaleTable[0], m.SliceTable[0])
	}
	wantIndexes(t, src, []int{11, 14, 15, 16, 40, 47, 73, 74, 75, 76, 77})
}

func TestDecodeCAVLCFrameSliceHighReconstructsWeightedPSkipFromRef(t *testing.T) {
	for _, bitDepth := range []int32{10, 12, 14} {
		t.Run(bitDepthName(bitDepth), func(t *testing.T) {
			m, dst, sh, ref := h264HighPFrameSliceDecodeFixture(t, bitDepth)
			sh.PPS.WeightedPred = 1
			pwt := highWeightedPPredWeightTable()
			sh.PredWeightTable = pwt
			gb := newBitReader(cavlcBitString("010"))

			got, err := m.decodeCAVLCFrameSliceHigh(&gb, dst, sh, h264FrameSliceDecodeInputHigh{
				SliceNum:      25,
				Refs:          [2][]*h264PicturePlanesHigh{{ref}},
				PredWeight:    &sh.PredWeightTable,
				MotionScratch: makeH264MotionCompScratchHigh(dst),
			})
			if err != nil {
				t.Fatalf("decode high cavlc weighted pskip slice failed: %v", err)
			}
			if got.Macroblocks != 1 || got.LastMBXY != 0 || !got.EndOfSlice || !got.EndOfFrame {
				t.Fatalf("slice result = %+v, want one skipped MB frame end", got)
			}
			want := h264HighWeightedPReference(t, ref, &pwt, int(bitDepth))
			assertH264SliceDecodeHighRef(t, "cavlc high weighted pskip", dst, want)
			if dst.Y[0] == ref.Y[0] {
				t.Fatalf("weighted luma sample unchanged: got %d", dst.Y[0])
			}
			wantType := MBType16x16 | MBTypeP0L0 | MBTypeP1L0 | MBTypeSkip
			if m.MacroblockTyp[0] != wantType || m.CBPTable[0] != 0 || m.QScaleTable[0] != uint8(sh.QScale) || m.SliceTable[0] != 25 {
				t.Fatalf("tables type/cbp/q/slice = %#x/%#x/%d/%d", m.MacroblockTyp[0], m.CBPTable[0], m.QScaleTable[0], m.SliceTable[0])
			}
			if gb.bitPos != 3 {
				t.Fatalf("consumed %d bits, want 3", gb.bitPos)
			}
		})
	}
}

func TestDecodeCABACFrameSliceHighReconstructsWeightedPSkipFromRef(t *testing.T) {
	for _, bitDepth := range []int32{10, 12} {
		t.Run(bitDepthName(bitDepth), func(t *testing.T) {
			m, dst, sh, ref := h264HighPFrameSliceDecodeFixture(t, bitDepth)
			sh.PPS.CABAC = 1
			sh.PPS.WeightedPred = 1
			pwt := highWeightedPPredWeightTable()
			sh.PredWeightTable = pwt
			src := &scriptedCABACSource{
				bits:  []int{1},
				terms: []int{1},
			}

			got, err := m.decodeCABACFrameSliceHigh(src, dst, sh, h264FrameSliceDecodeInputHigh{
				SliceNum:      26,
				Refs:          [2][]*h264PicturePlanesHigh{{ref}},
				PredWeight:    &sh.PredWeightTable,
				MotionScratch: makeH264MotionCompScratchHigh(dst),
			})
			if err != nil {
				t.Fatalf("decode high cabac weighted pskip slice failed: %v", err)
			}
			if got.Macroblocks != 1 || got.LastMBXY != 0 || !got.EndOfSlice || !got.EndOfFrame {
				t.Fatalf("slice result = %+v, want one skipped MB frame end", got)
			}
			want := h264HighWeightedPReference(t, ref, &pwt, int(bitDepth))
			assertH264SliceDecodeHighRef(t, "cabac high weighted pskip", dst, want)
			if dst.Y[0] == ref.Y[0] {
				t.Fatalf("weighted luma sample unchanged: got %d", dst.Y[0])
			}
			wantType := MBType16x16 | MBTypeP0L0 | MBTypeP1L0 | MBTypeSkip
			if m.MacroblockTyp[0] != wantType || m.CBPTable[0] != 0 || m.QScaleTable[0] != uint8(sh.QScale) || m.SliceTable[0] != 26 {
				t.Fatalf("tables type/cbp/q/slice = %#x/%#x/%d/%d", m.MacroblockTyp[0], m.CBPTable[0], m.QScaleTable[0], m.SliceTable[0])
			}
			wantIndexes(t, src, []int{11})
		})
	}
}

func TestDecodeCAVLCFrameSliceHighReconstructsWeightedP16x16FromRef(t *testing.T) {
	for _, bitDepth := range []int32{10, 12, 14} {
		t.Run(bitDepthName(bitDepth), func(t *testing.T) {
			m, dst, sh, ref := h264HighPFrameSliceDecodeFixture(t, bitDepth)
			sh.PPS.WeightedPred = 1
			pwt := highWeightedPPredWeightTable()
			sh.PredWeightTable = pwt
			gb := newBitReader(cavlcBitString("11111"))

			got, err := m.decodeCAVLCFrameSliceHigh(&gb, dst, sh, h264FrameSliceDecodeInputHigh{
				SliceNum:      27,
				Refs:          [2][]*h264PicturePlanesHigh{{ref}},
				PredWeight:    &sh.PredWeightTable,
				MotionScratch: makeH264MotionCompScratchHigh(dst),
			})
			if err != nil {
				t.Fatalf("decode high cavlc weighted p16x16 slice failed: %v", err)
			}
			if got.Macroblocks != 1 || got.LastMBXY != 0 || !got.EndOfSlice || !got.EndOfFrame {
				t.Fatalf("slice result = %+v, want one P16x16 MB frame end", got)
			}
			want := h264HighWeightedPReference(t, ref, &pwt, int(bitDepth))
			assertH264SliceDecodeHighRef(t, "cavlc high weighted p16x16", dst, want)
			if dst.Y[0] == ref.Y[0] {
				t.Fatalf("weighted luma sample unchanged: got %d", dst.Y[0])
			}
			wantType := MBType16x16 | MBTypeP0L0
			if m.MacroblockTyp[0] != wantType || m.CBPTable[0] != 0 || m.QScaleTable[0] != uint8(sh.QScale) || m.SliceTable[0] != 27 {
				t.Fatalf("tables type/cbp/q/slice = %#x/%#x/%d/%d", m.MacroblockTyp[0], m.CBPTable[0], m.QScaleTable[0], m.SliceTable[0])
			}
			if gb.bitPos != 5 {
				t.Fatalf("consumed %d bits, want 5", gb.bitPos)
			}
		})
	}
}

func TestDecodeCABACFrameSliceHighReconstructsWeightedP16x16FromRef(t *testing.T) {
	for _, bitDepth := range []int32{10, 12} {
		t.Run(bitDepthName(bitDepth), func(t *testing.T) {
			m, dst, sh, ref := h264HighPFrameSliceDecodeFixture(t, bitDepth)
			sh.PPS.CABAC = 1
			sh.PPS.WeightedPred = 1
			pwt := highWeightedPPredWeightTable()
			sh.PredWeightTable = pwt
			src := &scriptedCABACSource{
				bits: []int{
					0,
					0, 0, 0,
					0, 0,
					0, 0, 0, 0,
					0,
				},
				terms: []int{1},
			}

			got, err := m.decodeCABACFrameSliceHigh(src, dst, sh, h264FrameSliceDecodeInputHigh{
				SliceNum:      28,
				Refs:          [2][]*h264PicturePlanesHigh{{ref}},
				PredWeight:    &sh.PredWeightTable,
				MotionScratch: makeH264MotionCompScratchHigh(dst),
			})
			if err != nil {
				t.Fatalf("decode high cabac weighted p16x16 slice failed: %v", err)
			}
			if got.Macroblocks != 1 || got.LastMBXY != 0 || !got.EndOfSlice || !got.EndOfFrame {
				t.Fatalf("slice result = %+v, want one P16x16 MB frame end", got)
			}
			want := h264HighWeightedPReference(t, ref, &pwt, int(bitDepth))
			assertH264SliceDecodeHighRef(t, "cabac high weighted p16x16", dst, want)
			if dst.Y[0] == ref.Y[0] {
				t.Fatalf("weighted luma sample unchanged: got %d", dst.Y[0])
			}
			wantType := MBType16x16 | MBTypeP0L0
			if m.MacroblockTyp[0] != wantType || m.CBPTable[0] != 0 || m.QScaleTable[0] != uint8(sh.QScale) || m.SliceTable[0] != 28 {
				t.Fatalf("tables type/cbp/q/slice = %#x/%#x/%d/%d", m.MacroblockTyp[0], m.CBPTable[0], m.QScaleTable[0], m.SliceTable[0])
			}
			wantIndexes(t, src, []int{11, 14, 15, 16, 40, 47, 73, 74, 75, 76, 77})
		})
	}
}

func TestDecodeCAVLCFrameSliceAllowsDeblockingFlag(t *testing.T) {
	m, err := newMacroblockTables(1, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	sps := &SPS{BitDepthLuma: 8, ChromaFormatIDC: 1, FrameMBSOnlyFlag: 1}
	pps := cavlcFlatQMulPPS()
	pps.SPS = sps
	sh := &SliceHeader{
		FirstMBAddr:      0,
		SliceType:        PictureTypeI,
		SliceTypeNoS:     PictureTypeI,
		PictureStructure: PictureFrame,
		PPS:              pps,
		SPS:              sps,
		QScale:           20,
		DeblockingFilter: 1,
	}
	dst := makeH264SliceDecodePicture(1, 1, 1)
	gb := newBitReader(cavlcIntraPCMBytes(h264ReconstructIntraPCM(1, 5)))

	got, err := m.decodeCAVLCFrameSlice(&gb, dst, sh, h264FrameSliceDecodeInput{SliceNum: 1})
	if err != nil {
		t.Fatalf("decode err = %v", err)
	}
	if !got.EndOfFrame || got.Macroblocks != 1 {
		t.Fatalf("slice result = %+v, want one MB frame end", got)
	}
	assertH264SliceDecodePCM(t, dst, 0, 0, h264ReconstructIntraPCM(1, 5))
}

func TestH264FrameMBAFFReconstructViewKeepsFrameCodedMacroblocksInFrameView(t *testing.T) {
	dst := makeH264SliceDecodePicture(1, 4, 1)
	ref := makeH264SliceDecodePicture(1, 4, 1)
	ref.PictureStructure = PictureFrame
	refs := [2][]*h264PicturePlanes{{ref}}
	cur := sliceMacroblockCursor{FrameMBAFF: true, MBY: 2, PixelMBY: 2}
	var refPlanes [2][32]h264PicturePlanes
	var refPtrs [2][32]*h264PicturePlanes

	view, mbY, gotRefs, err := h264FrameMBAFFReconstructView(dst, cur, MBTypeIntra4x4, refs, &refPlanes, &refPtrs)
	if err != nil {
		t.Fatal(err)
	}
	if view.LumaStride != dst.LumaStride || view.ChromaStride != dst.ChromaStride || view.MBHeight != dst.MBHeight || mbY != cur.PixelMBY {
		t.Fatalf("frame-coded view stride/chroma/height/mbY = %d/%d/%d/%d, want %d/%d/%d/%d",
			view.LumaStride, view.ChromaStride, view.MBHeight, mbY, dst.LumaStride, dst.ChromaStride, dst.MBHeight, cur.PixelMBY)
	}
	if len(gotRefs[0]) != 1 || gotRefs[0][0] != ref {
		t.Fatalf("frame-coded refs = %#v, want original ref", gotRefs[0])
	}
}

func TestH264FrameMBAFFReconstructViewMapsBottomFieldDestinationAndRefs(t *testing.T) {
	dst := makeH264SliceDecodePicture(1, 4, 1)
	ref0 := makeH264SliceDecodePicture(1, 4, 1)
	ref1 := makeH264SliceDecodePicture(1, 4, 1)
	ref0.PictureStructure = PictureFrame
	ref1.PictureStructure = PictureFrame
	refs := [2][]*h264PicturePlanes{{ref0, ref1}}
	cur := sliceMacroblockCursor{FrameMBAFF: true, MBY: 1, PixelMBY: 1}
	var refPlanes [2][32]h264PicturePlanes
	var refPtrs [2][32]*h264PicturePlanes

	view, mbY, gotRefs, err := h264FrameMBAFFReconstructView(dst, cur, MBTypeInterlaced|MBType16x16|MBTypeP0L0, refs, &refPlanes, &refPtrs)
	if err != nil {
		t.Fatal(err)
	}
	if view.PictureStructure != PictureBottomField || view.LumaStride != dst.LumaStride*2 || view.ChromaStride != dst.ChromaStride*2 || view.MBHeight != 2 || mbY != 0 {
		t.Fatalf("bottom field view picture/stride/chroma/height/mbY = %d/%d/%d/%d/%d",
			view.PictureStructure, view.LumaStride, view.ChromaStride, view.MBHeight, mbY)
	}
	if &view.Y[0] != &dst.Y[dst.LumaStride] || &view.Cb[0] != &dst.Cb[dst.ChromaStride] || &view.Cr[0] != &dst.Cr[dst.ChromaStride] {
		t.Fatalf("bottom field view does not start on the second frame line")
	}
	if len(gotRefs[0]) != 4 {
		t.Fatalf("field refs len = %d, want 4", len(gotRefs[0]))
	}
	if gotRefs[0][0].PictureStructure != PictureBottomField || &gotRefs[0][0].Y[0] != &ref0.Y[ref0.LumaStride] {
		t.Fatalf("ref0 maps to bottom field of frame 0")
	}
	if gotRefs[0][1].PictureStructure != PictureTopField || &gotRefs[0][1].Y[0] != &ref0.Y[0] {
		t.Fatalf("ref1 maps to top field of frame 0")
	}
	if gotRefs[0][2].PictureStructure != PictureBottomField || &gotRefs[0][2].Y[0] != &ref1.Y[ref1.LumaStride] {
		t.Fatalf("ref2 maps to bottom field of frame 1")
	}
	if gotRefs[0][3].PictureStructure != PictureTopField || &gotRefs[0][3].Y[0] != &ref1.Y[0] {
		t.Fatalf("ref3 maps to top field of frame 1")
	}
}

func makeH264SliceDecodePicture(mbWidth int, mbHeight int, chromaFormatIDC int) *h264PicturePlanes {
	chromaWidth, chromaHeight := h264ChromaFrameSize(mbWidth, mbHeight, chromaFormatIDC)
	p := &h264PicturePlanes{
		Y:               make([]uint8, mbWidth*16*mbHeight*16),
		LumaStride:      mbWidth * 16,
		MBWidth:         mbWidth,
		MBHeight:        mbHeight,
		ChromaFormatIDC: chromaFormatIDC,
	}
	if chromaFormatIDC != 0 {
		p.ChromaStride = chromaWidth
		p.Cb = make([]uint8, chromaWidth*chromaHeight)
		p.Cr = make([]uint8, chromaWidth*chromaHeight)
	}
	return p
}

func cavlcIntraPCMBytes(pcm []byte) []byte {
	out := make([]byte, 0, 2+len(pcm))
	out = append(out, 0x0d, 0x00)
	out = append(out, pcm...)
	return out
}

func assertH264SliceDecodePCM(t *testing.T, dst *h264PicturePlanes, mbX int, mbY int, pcm []byte) {
	t.Helper()
	yOff, cbOff, crOff, err := h264MBDestPartOffsets(dst, mbX, mbY, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	assertH264Rows(t, "slice pcm y", dst.Y, yOff, dst.LumaStride, 16, 16, pcm, 16)
	if dst.ChromaFormatIDC == 0 {
		return
	}
	blockH := 8
	if dst.ChromaFormatIDC == 2 {
		blockH = 16
	}
	assertH264Rows(t, "slice pcm cb", dst.Cb, cbOff, dst.ChromaStride, 8, blockH, pcm[256:], 8)
	assertH264Rows(t, "slice pcm cr", dst.Cr, crOff, dst.ChromaStride, 8, blockH, pcm[256+8*blockH:], 8)
}

func h264HighPFrameSliceDecodeFixture(t *testing.T, bitDepth int32) (*macroblockTables, *h264PicturePlanesHigh, *SliceHeader, *h264PicturePlanesHigh) {
	t.Helper()

	m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, bitDepth, 1, 1, false, PictureTypeP)
	sh.QScale = 24
	sh.RefCount = [2]uint32{1, 0}
	ref := makeH264SliceDecodePictureHigh(1, 1, 1)
	fillH264MotionCompPlaneHigh(ref.Y, 73, int(bitDepth))
	fillH264MotionCompPlaneHigh(ref.Cb, 91, int(bitDepth))
	fillH264MotionCompPlaneHigh(ref.Cr, 119, int(bitDepth))
	return m, dst, sh, ref
}

func assertH264SliceDecodeHighRef(t *testing.T, label string, dst *h264PicturePlanesHigh, ref *h264PicturePlanesHigh) {
	t.Helper()

	assertH264RowsHigh(t, label+" y", dst.Y, 0, dst.LumaStride, 16, 16, ref.Y, ref.LumaStride)
	assertH264RowsHigh(t, label+" cb", dst.Cb, 0, dst.ChromaStride, 8, 8, ref.Cb, ref.ChromaStride)
	assertH264RowsHigh(t, label+" cr", dst.Cr, 0, dst.ChromaStride, 8, 8, ref.Cr, ref.ChromaStride)
}

func highWeightedPPredWeightTable() PredWeightTable {
	pwt := h264MotionCompTestPWT(1)
	pwt.UseWeight = 1
	pwt.UseWeightChroma = 1
	pwt.LumaLog2WeightDenom = 2
	pwt.ChromaLog2WeightDenom = 1
	pwt.LumaWeight[0][0] = [2]int32{3, -2}
	pwt.ChromaWeight[0][0][0] = [2]int32{2, 1}
	pwt.ChromaWeight[0][0][1] = [2]int32{-1, 3}
	return pwt
}

func h264HighWeightedPReference(t *testing.T, ref *h264PicturePlanesHigh, pwt *PredWeightTable, bitDepth int) *h264PicturePlanesHigh {
	t.Helper()

	want := cloneH264HighResidualPicture(ref)
	if err := h264WeightPixelsHigh(want.Y, want.LumaStride, 16, int(pwt.LumaLog2WeightDenom), int(pwt.LumaWeight[0][0][0]), int(pwt.LumaWeight[0][0][1]), 16, bitDepth); err != nil {
		t.Fatal(err)
	}
	if err := h264WeightPixelsHigh(want.Cb, want.ChromaStride, 8, int(pwt.ChromaLog2WeightDenom), int(pwt.ChromaWeight[0][0][0][0]), int(pwt.ChromaWeight[0][0][0][1]), 8, bitDepth); err != nil {
		t.Fatal(err)
	}
	if err := h264WeightPixelsHigh(want.Cr, want.ChromaStride, 8, int(pwt.ChromaLog2WeightDenom), int(pwt.ChromaWeight[0][0][1][0]), int(pwt.ChromaWeight[0][0][1][1]), 8, bitDepth); err != nil {
		t.Fatal(err)
	}
	return want
}
