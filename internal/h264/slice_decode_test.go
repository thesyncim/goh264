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

func TestDecodeCAVLCFrameSliceRejectsPendingDeblocking(t *testing.T) {
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

	_, err = m.decodeCAVLCFrameSlice(&gb, dst, sh, h264FrameSliceDecodeInput{SliceNum: 1})
	if err != ErrUnsupported {
		t.Fatalf("err = %v, want ErrUnsupported", err)
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
