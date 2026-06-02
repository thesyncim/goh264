// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

import "testing"

type simpleDPBHighTestFrameData struct {
	frame     *DecodedFrame
	y, cb, cr []uint16
}

func TestSimpleFrameDPBHighPRefsExposeUint16Planes(t *testing.T) {
	sps := simpleDPBHighTestSPS(2, 1)
	newest := simpleDPBHighTestFrame(sps, 2, 31)
	older := simpleDPBHighTestFrame(sps, 1, 73)
	current := simpleDPBHighTestFrame(sps, 3, 109)
	dpb := simpleFrameDPB{short: []*DecodedFrame{newest.frame, older.frame}}

	ctx, err := dpb.buildRefContext(simpleDPBTestPHeader(sps, 3, 2), current.frame)
	if err != nil {
		t.Fatal(err)
	}
	if len(ctx.Refs[0]) != 0 || len(ctx.RefsHigh[0]) != 2 {
		t.Fatalf("ref lengths byte/high = %d/%d, want 0/2", len(ctx.Refs[0]), len(ctx.RefsHigh[0]))
	}
	requireSimpleDPBHighRef(t, "newest", ctx.RefsHigh[0][0], newest)
	requireSimpleDPBHighRef(t, "older", ctx.RefsHigh[0][1], older)
}

func TestSimpleFrameDPBHighPRefsPreserveLongShortOrdering(t *testing.T) {
	sps := simpleDPBHighTestSPS(4, 2)
	short := simpleDPBHighTestFrame(sps, 2, 11)
	long0 := simpleDPBHighTestFrame(sps, 10, 151)
	long3 := simpleDPBHighTestFrame(sps, 11, 211)
	current := simpleDPBHighTestFrame(sps, 3, 307)
	dpb := simpleFrameDPB{short: []*DecodedFrame{short.frame}}
	dpb.long[0] = long0.frame
	dpb.long[3] = long3.frame

	refs, err := dpb.buildRefListsHigh(simpleDPBTestPHeader(sps, 3, 3), current.frame)
	if err != nil {
		t.Fatal(err)
	}
	if len(refs[0]) != 3 {
		t.Fatalf("default P high refs = %d, want 3", len(refs[0]))
	}
	requireSimpleDPBHighRef(t, "short", refs[0][0], short)
	requireSimpleDPBHighRef(t, "long0", refs[0][1], long0)
	requireSimpleDPBHighRef(t, "long3", refs[0][2], long3)

	long1 := simpleDPBHighTestFrame(sps, 12, 401)
	dpb.long[1] = long1.frame
	sh := simpleDPBTestPHeader(sps, 3, 2)
	sh.NBRefModifications[0] = 1
	sh.RefModifications[0][0] = RefModification{Op: 2, Val: 1}
	refs, err = dpb.buildRefListsHigh(sh, current.frame)
	if err != nil {
		t.Fatal(err)
	}
	if len(refs[0]) != 2 {
		t.Fatalf("reordered P high refs = %d, want 2", len(refs[0]))
	}
	requireSimpleDPBHighRef(t, "long1", refs[0][0], long1)
	requireSimpleDPBHighRef(t, "short", refs[0][1], short)
}

func TestSimpleFrameDPBHighBRefsPreservePOCOrdering(t *testing.T) {
	sps := simpleDPBHighTestSPS(2, 3)
	past := simpleDPBHighTestFrame(sps, 0, 17)
	past.frame.poc = 0
	future := simpleDPBHighTestFrame(sps, 1, 149)
	future.frame.poc = 4
	current := simpleDPBHighTestFrame(sps, 2, 283)
	current.frame.poc = 2
	dpb := simpleFrameDPB{short: []*DecodedFrame{future.frame, past.frame}}

	refs, err := dpb.buildRefListsHigh(simpleDPBTestBHeader(sps, 2, 1, 1), current.frame)
	if err != nil {
		t.Fatal(err)
	}
	if len(refs[0]) != 1 || len(refs[1]) != 1 {
		t.Fatalf("B high refs lengths = %d/%d, want 1/1", len(refs[0]), len(refs[1]))
	}
	requireSimpleDPBHighRef(t, "list0 past", refs[0][0], past)
	requireSimpleDPBHighRef(t, "list1 future", refs[1][0], future)
}

func TestSimpleFrameDPB8BitRefContextKeepsByteRefs(t *testing.T) {
	sps := simpleDPBTestSPS(1)
	ref := simpleDPBTestFrame(sps, 0)
	current := simpleDPBTestFrame(sps, 1)
	dpb := simpleFrameDPB{short: []*DecodedFrame{ref}}

	ctx, err := dpb.buildRefContext(simpleDPBTestPHeader(sps, 1, 1), current)
	if err != nil {
		t.Fatal(err)
	}
	if len(ctx.Refs[0]) != 1 || len(ctx.RefsHigh[0]) != 0 {
		t.Fatalf("8-bit ref lengths byte/high = %d/%d, want 1/0", len(ctx.Refs[0]), len(ctx.RefsHigh[0]))
	}
	if len(ctx.Refs[0][0].Y) != len(ref.Y) || len(ref.Y) == 0 || &ctx.Refs[0][0].Y[0] != &ref.Y[0] {
		t.Fatalf("8-bit ref did not preserve byte luma backing")
	}
}

func requireSimpleDPBHighRef(t *testing.T, label string, got *h264PicturePlanesHigh, want simpleDPBHighTestFrameData) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s ref is nil", label)
	}
	if err := got.validate(); err != nil {
		t.Fatalf("%s ref validation failed: %v", label, err)
	}
	if got.LumaStride != want.frame.LumaStride || got.ChromaStride != want.frame.ChromaStride ||
		got.MBWidth != want.frame.MBWidth || got.MBHeight != want.frame.MBHeight ||
		got.ChromaFormatIDC != want.frame.ChromaFormatIDC {
		t.Fatalf("%s metadata = stride %d/%d mb %dx%d chroma %d, want stride %d/%d mb %dx%d chroma %d",
			label, got.LumaStride, got.ChromaStride, got.MBWidth, got.MBHeight, got.ChromaFormatIDC,
			want.frame.LumaStride, want.frame.ChromaStride, want.frame.MBWidth, want.frame.MBHeight, want.frame.ChromaFormatIDC)
	}
	if !sameSimpleDPBHighBacking(got.Y, want.y) {
		t.Fatalf("%s luma backing mismatch", label)
	}
	if got.ChromaFormatIDC == 0 {
		if len(got.Cb) != 0 || len(got.Cr) != 0 {
			t.Fatalf("%s monochrome chroma lengths = %d/%d, want 0/0", label, len(got.Cb), len(got.Cr))
		}
		return
	}
	if !sameSimpleDPBHighBacking(got.Cb, want.cb) || !sameSimpleDPBHighBacking(got.Cr, want.cr) {
		t.Fatalf("%s chroma backing mismatch", label)
	}
	got.Y[5] ^= 1
	if want.y[5] != got.Y[5] {
		t.Fatalf("%s luma view is not backed by the source uint16 plane", label)
	}
	got.Y[5] ^= 1
}

func simpleDPBHighTestSPS(refs uint32, chromaFormatIDC int) *SPS {
	sps := simpleDPBTestSPS(refs)
	sps.ChromaFormatIDC = uint32(chromaFormatIDC)
	sps.BitDepthLuma = 10
	sps.BitDepthChroma = 10
	return sps
}

func simpleDPBHighTestFrame(sps *SPS, frameNum uint32, seed int) simpleDPBHighTestFrameData {
	lumaStride := int(sps.MBWidth)*16 + 4
	lumaHeight := int(sps.MBHeight) * 16
	y := make([]uint16, lumaStride*lumaHeight)
	fillSimpleDPBHighPlane(y, seed)

	frame := &DecodedFrame{
		Y16:              y,
		LumaStride:       lumaStride,
		Width:            int(sps.Width),
		Height:           int(sps.Height),
		MBWidth:          int(sps.MBWidth),
		MBHeight:         int(sps.MBHeight),
		ChromaFormatIDC:  int(sps.ChromaFormatIDC),
		BitDepthLuma:     int(sps.BitDepthLuma),
		BitDepthChroma:   int(sps.BitDepthChroma),
		frameMBSOnlyFlag: sps.FrameMBSOnlyFlag,
		frameNum:         frameNum,
	}

	out := simpleDPBHighTestFrameData{frame: frame, y: y}
	if sps.ChromaFormatIDC != 0 {
		chromaWidth, chromaHeight := h264ChromaFrameSize(int(sps.MBWidth), int(sps.MBHeight), int(sps.ChromaFormatIDC))
		frame.ChromaStride = chromaWidth + 3
		out.cb = make([]uint16, frame.ChromaStride*chromaHeight)
		out.cr = make([]uint16, frame.ChromaStride*chromaHeight)
		fillSimpleDPBHighPlane(out.cb, seed+503)
		fillSimpleDPBHighPlane(out.cr, seed+907)
		frame.Cb16 = out.cb
		frame.Cr16 = out.cr
	}
	return out
}

func fillSimpleDPBHighPlane(p []uint16, seed int) {
	for i := range p {
		p[i] = uint16((seed + 17*i + (i >> 3)) & 0x03ff)
	}
}

func sameSimpleDPBHighBacking(a []uint16, b []uint16) bool {
	return len(a) == len(b) && (len(a) == 0 || &a[0] == &b[0])
}
