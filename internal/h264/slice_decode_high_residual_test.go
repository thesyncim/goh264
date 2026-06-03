// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

import (
	"errors"
	"testing"
)

func TestHighP16x16ResidualHandoffReconstructsExactLuma(t *testing.T) {
	const cbp = 0x03
	const cbpTable = cbp | (cbp << 12)
	mbType := MBType16x16 | MBTypeP0L0

	tests := []struct {
		name string
		run  func(*SliceHeader, sliceMacroblockCursor, frameMacroblockDecodeWork, h264FrameSliceDecodeInputHigh) h264FrameMBReconstructInputHigh
	}{
		{
			name: "cavlc",
			run: func(sh *SliceHeader, cur sliceMacroblockCursor, work frameMacroblockDecodeWork, in h264FrameSliceDecodeInputHigh) h264FrameMBReconstructInputHigh {
				return h264FrameMBReconstructInputHighFromCAVLC(sh, cur, cavlcFrameMacroblockResult{
					MBType:   mbType,
					CBP:      cbp,
					CBPTable: cbpTable,
					QScale:   int(sh.QScale),
					ChromaQP: [2]uint8{uint8(sh.QScale), uint8(sh.QScale)},
					IsInter:  true,
				}, &work, in)
			},
		},
		{
			name: "cabac",
			run: func(sh *SliceHeader, cur sliceMacroblockCursor, work frameMacroblockDecodeWork, in h264FrameSliceDecodeInputHigh) h264FrameMBReconstructInputHigh {
				return h264FrameMBReconstructInputHighFromCABAC(sh, cur, cabacFrameMacroblockResult{
					MBType:   mbType,
					CBP:      cbp,
					CBPTable: cbpTable,
					QScale:   int(sh.QScale),
					ChromaQP: [2]uint8{uint8(sh.QScale), uint8(sh.QScale)},
					IsInter:  true,
				}, &work, in)
			},
		},
	}

	for _, bitDepth := range []int32{10, 12} {
		intDepth := int(bitDepth)
		for _, tt := range tests {
			t.Run(bitDepthName(bitDepth)+"/"+tt.name, func(t *testing.T) {
				m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, bitDepth, 1, 1, false, PictureTypeP)
				sh.QScale = 24
				sh.RefCount = [2]uint32{1, 0}
				ref := makeH264SliceDecodePictureHigh(1, 1, 1)
				fillH264HighResidualPlane(ref.Y, 400)
				fillH264HighResidualPlane(ref.Cb, 512)
				fillH264HighResidualPlane(ref.Cr, 640)

				cur, err := newSliceMacroblockCursor(m, sh)
				if err != nil {
					t.Fatal(err)
				}
				work, changed := h264HighP16x16LumaResidualWork()
				in := h264FrameSliceDecodeInputHigh{
					Refs:          [2][]*h264PicturePlanesHigh{{ref}},
					MotionScratch: makeH264MotionCompScratchHigh(dst),
				}
				reconstruct := tt.run(sh, cur, work, in)
				if reconstruct.MBType != mbType || reconstruct.CBP != cbp || reconstruct.BitDepth != intDepth || reconstruct.PredWeight != nil || reconstruct.DeblockingFilter {
					t.Fatalf("handoff = type %#x cbp %#x depth %d pwt %v deblock %v",
						reconstruct.MBType, reconstruct.CBP, reconstruct.BitDepth, reconstruct.PredWeight, reconstruct.DeblockingFilter)
				}

				if err := h264HLDecodeFrameMacroblockHigh(dst, reconstruct); err != nil {
					t.Fatalf("reconstruct high P16x16 residual failed: %v", err)
				}

				want := cloneH264HighResidualPicture(ref)
				applyH264HighP16x16LumaResidualExpected(t, want, changed, intDepth)
				assertH264RowsHigh(t, tt.name+" high p16 residual y", dst.Y, 0, dst.LumaStride, 16, 16, want.Y, want.LumaStride)
				assertH264RowsHigh(t, tt.name+" high p16 residual cb", dst.Cb, 0, dst.ChromaStride, 8, 8, want.Cb, want.ChromaStride)
				assertH264RowsHigh(t, tt.name+" high p16 residual cr", dst.Cr, 0, dst.ChromaStride, 8, 8, want.Cr, want.ChromaStride)

				for _, block := range changed {
					if got := reconstruct.Residual.MB[block.index*16]; got != 0 {
						t.Fatalf("%s residual block %d was not cleared: %d", tt.name, block.index, got)
					}
				}
			})
		}
	}
}

func TestHighBDirectSubCABACHandoffReconstructsExactLuma(t *testing.T) {
	const bitDepth = 10
	const cbp = 0x01
	const cbpTable = 0x01

	m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, bitDepth, 1, 1, false, PictureTypeB)
	sh.PPS.CABAC = 1
	sh.QScale = 22
	sh.RefCount = [2]uint32{1, 1}
	sh.SPS.Direct8x8InferenceFlag = 1

	cur, err := newSliceMacroblockCursor(m, sh)
	if err != nil {
		t.Fatal(err)
	}
	refs, direct := highBSkipDirectRefsHigh(t, false)
	direct.Direct8x8Inference = true
	in := h264FrameSliceDecodeInputHigh{
		Refs:          refs,
		Direct:        direct,
		MotionScratch: makeH264MotionCompScratchHigh(dst),
	}

	work, changed := h264HighP16x16LumaResidualWork()
	mbType := MBType8x8 | MBTypeP0L0 | MBTypeP0L1 | MBTypeP1L0 | MBTypeP1L1
	subMBType := [4]uint32{MBTypeDirect2, MBTypeDirect2, MBTypeDirect2, MBTypeDirect2}
	if err := m.predDirectMotionFrame(&work.Motion, 0, &mbType, &subMBType, direct); err != nil {
		t.Fatalf("predict direct-sub motion failed: %v", err)
	}
	if !isHighB8x8DirectSubMacroblock(mbType, &subMBType, cbp) {
		t.Fatalf("direct-sub shape = %#x/%#x, want resolved B8x8 direct-sub", mbType, subMBType)
	}
	if err := validateHighFrameSliceMacroblockForReconstructWithSubMB(sh, mbType, &subMBType, cbp, cbpTable); err != nil {
		t.Fatalf("validate direct-sub residual err = %v, want nil", err)
	}

	reconstruct := h264FrameMBReconstructInputHighFromCABAC(sh, cur, cabacFrameMacroblockResult{
		MBType:   mbType,
		CBP:      cbp,
		CBPTable: cbpTable,
		QScale:   int(sh.QScale),
		ChromaQP: [2]uint8{uint8(sh.QScale), uint8(sh.QScale)},
		Inter: cavlcInterMacroblockSyntax{
			SubMBType: subMBType,
		},
		IsInter: true,
	}, &work, in)
	if reconstruct.MBType != mbType || reconstruct.SubMBType != subMBType || reconstruct.CBP != cbp || reconstruct.BitDepth != bitDepth || reconstruct.PredWeight != nil || reconstruct.DeblockingFilter {
		t.Fatalf("handoff = type %#x sub %#x cbp %#x depth %d pwt %v deblock %v",
			reconstruct.MBType, reconstruct.SubMBType, reconstruct.CBP, reconstruct.BitDepth, reconstruct.PredWeight, reconstruct.DeblockingFilter)
	}

	if err := h264HLDecodeFrameMacroblockHigh(dst, reconstruct); err != nil {
		t.Fatalf("reconstruct high B direct-sub residual failed: %v", err)
	}

	want := cloneH264HighResidualPicture(refs[0][0])
	applyH264HighP16x16LumaResidualExpected(t, want, changed, bitDepth)
	assertH264RowsHigh(t, "cabac high b direct-sub residual y", dst.Y, 0, dst.LumaStride, 16, 16, want.Y, want.LumaStride)
	assertH264RowsHigh(t, "cabac high b direct-sub residual cb", dst.Cb, 0, dst.ChromaStride, 8, 8, want.Cb, want.ChromaStride)
	assertH264RowsHigh(t, "cabac high b direct-sub residual cr", dst.Cr, 0, dst.ChromaStride, 8, 8, want.Cr, want.ChromaStride)

	for _, block := range changed {
		if got := reconstruct.Residual.MB[block.index*16]; got != 0 {
			t.Fatalf("direct-sub residual block %d was not cleared: %d", block.index, got)
		}
	}
}

func TestHighP8x8DCTCABACHandoffReconstructsPublicHigh9Shape(t *testing.T) {
	const bitDepth = 9
	const cbp = 0x1f
	const cbpTable = 0x9f

	m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, bitDepth, 1, 1, false, PictureTypeP)
	sh.PPS.CABAC = 1
	sh.QScale = 22
	sh.RefCount = [2]uint32{1, 0}

	cur, err := newSliceMacroblockCursor(m, sh)
	if err != nil {
		t.Fatal(err)
	}
	ref := makeH264SliceDecodePictureHigh(1, 1, 1)
	fillH264HighResidualPlane(ref.Y, 320)
	fillH264HighResidualPlane(ref.Cb, 448)
	fillH264HighResidualPlane(ref.Cr, 576)
	in := h264FrameSliceDecodeInputHigh{
		Refs:          [2][]*h264PicturePlanesHigh{{ref}},
		MotionScratch: makeH264MotionCompScratchHigh(dst),
	}
	mbType := MBType8x8 | MBTypeP0L0 | MBTypeP1L0 | MBType8x8DCT
	subMBType := [4]uint32{
		MBType16x16 | MBTypeP0L0,
		MBType16x16 | MBTypeP0L0,
		MBType16x16 | MBTypeP0L0,
		MBType16x16 | MBTypeP0L0,
	}
	if err := validateHighFrameSliceMacroblockForReconstructWithSubMB(sh, mbType, &subMBType, cbp, cbpTable); err != nil {
		t.Fatalf("validate high9 P8x8 8x8-DCT residual err = %v, want nil", err)
	}

	var work frameMacroblockDecodeWork
	for _, n := range []int{0, 4, 8, 12} {
		work.Motion.Ref[0][h264Scan8[n]] = 0
		work.Residual.NonZeroCountCache[h264Scan8[n]] = 1
		work.Residual.MB[n*16] = 256
	}
	wantResidual := work.Residual
	reconstruct := h264FrameMBReconstructInputHighFromCABAC(sh, cur, cabacFrameMacroblockResult{
		MBType:   mbType,
		CBP:      cbp,
		CBPTable: cbpTable,
		QScale:   int(sh.QScale),
		ChromaQP: [2]uint8{uint8(sh.QScale), uint8(sh.QScale)},
		Inter: cavlcInterMacroblockSyntax{
			SubMBType: subMBType,
		},
		IsInter: true,
	}, &work, in)
	if reconstruct.MBType != mbType || reconstruct.SubMBType != subMBType || reconstruct.CBP != cbp || reconstruct.BitDepth != bitDepth {
		t.Fatalf("handoff = type %#x sub %#x cbp %#x depth %d",
			reconstruct.MBType, reconstruct.SubMBType, reconstruct.CBP, reconstruct.BitDepth)
	}

	if err := h264HLDecodeFrameMacroblockHigh(dst, reconstruct); err != nil {
		t.Fatalf("reconstruct high9 P8x8 8x8-DCT residual failed: %v", err)
	}

	want := cloneH264HighResidualPicture(ref)
	blockOffset, err := h264FrameBlockOffsets(want.LumaStride, want.ChromaStride, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := h264IDCT8Add4PlaneHigh(want.Y, &blockOffset, wantResidual.MB[:], want.LumaStride, &wantResidual.NonZeroCountCache, 0, bitDepth); err != nil {
		t.Fatalf("build expected high9 P8x8 8x8-DCT residual: %v", err)
	}
	assertH264RowsHigh(t, "cabac high9 p8x8 8x8-DCT residual y", dst.Y, 0, dst.LumaStride, 16, 16, want.Y, want.LumaStride)
	assertH264RowsHigh(t, "cabac high9 p8x8 8x8-DCT residual cb", dst.Cb, 0, dst.ChromaStride, 8, 8, ref.Cb, ref.ChromaStride)
	assertH264RowsHigh(t, "cabac high9 p8x8 8x8-DCT residual cr", dst.Cr, 0, dst.ChromaStride, 8, 8, ref.Cr, ref.ChromaStride)
}

func TestHighPIntra8x8DCTCABACHandoffReconstructsPublicHigh9Shape(t *testing.T) {
	const bitDepth = 9
	const cbp = 0x2e
	const cbpTable = 0x6e

	sps := &SPS{
		BitDepthLuma:     bitDepth,
		BitDepthChroma:   bitDepth,
		ChromaFormatIDC:  1,
		FrameMBSOnlyFlag: 1,
	}
	pps := cavlcFlatQMulPPS()
	pps.SPS = sps
	pps.CABAC = 1
	sh := &SliceHeader{
		SliceType:        PictureTypeP,
		SliceTypeNoS:     PictureTypeP,
		PictureStructure: PictureFrame,
		PPS:              pps,
		SPS:              sps,
		QScale:           22,
		RefCount:         [2]uint32{1, 0},
	}
	mbType := MBTypeIntra4x4 | MBType8x8DCT
	if err := validateHighFrameSliceMacroblockForReconstructWithSubMB(sh, mbType, nil, cbp, cbpTable); err != nil {
		t.Fatalf("validate high9 P intra 8x8-DCT residual err = %v, want nil", err)
	}

	dst := makeH264ReconstructHighPicture(1, 41)
	cur := sliceMacroblockCursor{MBX: 1, MBY: 1, PixelMBY: 1}
	var work frameMacroblockDecodeWork
	work.IntraCache = h264ReconstructIntra8x8PredCache()
	for _, n := range []int{0, 4, 8, 12} {
		work.Residual.NonZeroCountCache[h264Scan8[n]] = 1
		work.Residual.MB[n*16] = int32(224 + n)
	}
	work.Residual.NonZeroCountCache[h264Scan8[chromaDCBlockIndex+0]] = 1
	work.Residual.MB[16*16] = 2
	work.Residual.MB[16*16+16] = -1
	work.Residual.MB[16*16+32] = 1
	work.Residual.MB[16*16+48] = 2
	work.Residual.NonZeroCountCache[h264Scan8[16]] = 1
	work.Residual.MB[16*16+1] = 3
	work.Residual.NonZeroCountCache[h264Scan8[chromaDCBlockIndex+1]] = 1
	work.Residual.MB[32*16] = -2
	work.Residual.MB[32*16+16] = 1
	work.Residual.MB[32*16+32] = -1
	work.Residual.MB[32*16+48] = 2
	work.Residual.NonZeroCountCache[h264Scan8[32]] = 1
	work.Residual.MB[32*16+2] = -3

	yOff := cur.MBY*16*dst.LumaStride + cur.MBX*16
	beforeY := dst.Y[yOff]
	cbOff := cur.MBY*8*dst.ChromaStride + cur.MBX*8
	beforeCb := dst.Cb[cbOff]
	reconstruct := h264FrameMBReconstructInputHighFromCABAC(sh, cur, cabacFrameMacroblockResult{
		MBType:            mbType,
		CBP:               cbp,
		CBPTable:          cbpTable,
		QScale:            int(sh.QScale),
		ChromaQP:          [2]uint8{uint8(sh.QScale), uint8(sh.QScale)},
		ChromaPred:        int32(intraPred8x8DC),
		TopLeftAvailable:  0xffff,
		TopRightAvailable: 0xffff,
		IsIntra:           true,
	}, &work, h264FrameSliceDecodeInputHigh{})
	if reconstruct.MBType != mbType || reconstruct.CBP != cbp || reconstruct.BitDepth != bitDepth || reconstruct.Intra4x4PredCache == nil {
		t.Fatalf("handoff = type %#x cbp %#x depth %d cache %v",
			reconstruct.MBType, reconstruct.CBP, reconstruct.BitDepth, reconstruct.Intra4x4PredCache)
	}

	if err := h264HLDecodeFrameMacroblockHigh(dst, reconstruct); err != nil {
		t.Fatalf("reconstruct high9 P intra 8x8-DCT residual failed: %v", err)
	}
	if dst.Y[yOff] == beforeY {
		t.Fatalf("high9 P intra 8x8-DCT luma top-left was not reconstructed, still %d", beforeY)
	}
	if dst.Cb[cbOff] == beforeCb {
		t.Fatalf("high9 P intra 8x8-DCT chroma top-left was not reconstructed, still %d", beforeCb)
	}
	if reconstruct.Residual.MB[0] != 0 || reconstruct.Residual.MB[4*16] != 0 || reconstruct.Residual.MB[16*16] != 0 {
		t.Fatalf("high9 P intra 8x8-DCT residual blocks were not cleared: %d/%d/%d",
			reconstruct.Residual.MB[0], reconstruct.Residual.MB[4*16], reconstruct.Residual.MB[16*16])
	}
}

func TestHighBExplicitSub8x8DCTHandoffReconstructsExactLuma(t *testing.T) {
	const bitDepth = 9
	const cbp = 0x01
	const cbpTable = 0x01

	m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, bitDepth, 1, 1, false, PictureTypeB)
	sh.PPS.CABAC = 1
	sh.QScale = 22
	sh.RefCount = [2]uint32{1, 1}

	cur, err := newSliceMacroblockCursor(m, sh)
	if err != nil {
		t.Fatal(err)
	}
	refs := highPartitionedBRefsHigh(t)
	in := h264FrameSliceDecodeInputHigh{
		Refs:          refs,
		MotionScratch: makeH264MotionCompScratchHigh(dst),
	}
	mbType := MBType8x8 | MBTypeP0L0 | MBTypeP1L0 | MBType8x8DCT
	subMBType := [4]uint32{
		MBType16x16 | MBTypeP0L0,
		MBType16x16 | MBTypeP0L0,
		MBType16x16 | MBTypeP0L0,
		MBType16x16 | MBTypeP0L0,
	}
	if !isHighB8x8ExplicitSubMacroblock(mbType, &subMBType) {
		t.Fatalf("explicit 8x8-DCT B shape = %#x/%#x, want accepted explicit B8x8 sub partitions", mbType, subMBType)
	}
	if err := validateHighFrameSliceMacroblockForReconstructWithSubMB(sh, mbType, &subMBType, cbp, cbpTable); err != nil {
		t.Fatalf("validate explicit 8x8-DCT B residual err = %v, want nil", err)
	}

	var work frameMacroblockDecodeWork
	for _, n := range []int{0, 4, 8, 12} {
		work.Motion.Ref[0][h264Scan8[n]] = 0
	}
	work.Residual.NonZeroCountCache[h264Scan8[0]] = 1
	work.Residual.MB[0] = 256
	wantResidual := work.Residual
	reconstruct := h264FrameMBReconstructInputHighFromCABAC(sh, cur, cabacFrameMacroblockResult{
		MBType:   mbType,
		CBP:      cbp,
		CBPTable: cbpTable,
		QScale:   int(sh.QScale),
		ChromaQP: [2]uint8{uint8(sh.QScale), uint8(sh.QScale)},
		Inter: cavlcInterMacroblockSyntax{
			SubMBType: subMBType,
		},
		IsInter: true,
	}, &work, in)
	if reconstruct.MBType != mbType || reconstruct.SubMBType != subMBType || reconstruct.CBP != cbp || reconstruct.BitDepth != bitDepth {
		t.Fatalf("handoff = type %#x sub %#x cbp %#x depth %d", reconstruct.MBType, reconstruct.SubMBType, reconstruct.CBP, reconstruct.BitDepth)
	}

	if err := h264HLDecodeFrameMacroblockHigh(dst, reconstruct); err != nil {
		t.Fatalf("reconstruct high B explicit 8x8-DCT residual failed: %v", err)
	}

	want := cloneH264HighResidualPicture(refs[0][0])
	blockOffset, err := h264FrameBlockOffsets(want.LumaStride, want.ChromaStride, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := h264IDCT8Add4PlaneHigh(want.Y, &blockOffset, wantResidual.MB[:], want.LumaStride, &wantResidual.NonZeroCountCache, 0, bitDepth); err != nil {
		t.Fatalf("build expected high B explicit 8x8-DCT residual: %v", err)
	}
	assertH264RowsHigh(t, "cabac high9 b explicit 8x8-DCT residual y", dst.Y, 0, dst.LumaStride, 16, 16, want.Y, want.LumaStride)
	assertH264RowsHigh(t, "cabac high9 b explicit 8x8-DCT residual cb", dst.Cb, 0, dst.ChromaStride, 8, 8, refs[0][0].Cb, refs[0][0].ChromaStride)
	assertH264RowsHigh(t, "cabac high9 b explicit 8x8-DCT residual cr", dst.Cr, 0, dst.ChromaStride, 8, 8, refs[0][0].Cr, refs[0][0].ChromaStride)
}

func TestHighResidualLaneRejectsUnsupportedBoundaries(t *testing.T) {
	t.Run("b direct 16x16 macroblock", func(t *testing.T) {
		sh := &SliceHeader{SliceTypeNoS: PictureTypeB}
		mbType := MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2

		if err := validateHighFrameSliceMacroblockForReconstruct(sh, mbType, 0, 0); err != nil {
			t.Fatalf("direct high B16x16 validate err = %v, want nil", err)
		}
	})

	t.Run("b temporal direct 16x16 macroblock", func(t *testing.T) {
		sh := &SliceHeader{SliceTypeNoS: PictureTypeB}
		mbType := MBType16x16 | MBTypeL0L1 | MBTypeDirect2

		if err := validateHighFrameSliceMacroblockForReconstruct(sh, mbType, 0, 0); err != nil {
			t.Fatalf("temporal direct high B16x16 validate err = %v, want nil", err)
		}
	})

	t.Run("b temporal direct 16x16 8x8 dct residual macroblock", func(t *testing.T) {
		sh := &SliceHeader{SliceTypeNoS: PictureTypeB}
		mbType := MBType16x16 | MBTypeL0L1 | MBTypeDirect2 | MBType8x8DCT

		if err := validateHighFrameSliceMacroblockForReconstruct(sh, mbType, 0x05, 0x05); err != nil {
			t.Fatalf("temporal direct high B16x16 8x8-DCT validate err = %v, want nil", err)
		}
	})

	t.Run("b list1 direct 16x16 residual macroblock", func(t *testing.T) {
		sh := &SliceHeader{SliceTypeNoS: PictureTypeB}
		mbType := MBType16x16 | MBTypeP0L1 | MBTypeDirect2

		if err := validateHighFrameSliceMacroblockForReconstruct(sh, mbType, 0x0c, 0x0c); err != nil {
			t.Fatalf("list1 direct high B16x16 validate err = %v, want nil", err)
		}
	})

	t.Run("b direct unresolved macroblock", func(t *testing.T) {
		sh := &SliceHeader{SliceTypeNoS: PictureTypeB}
		mbType := MBTypeDirect2 | MBTypeL0L1

		if err := validateHighFrameSliceMacroblockForReconstruct(sh, mbType, 0, 0); err != ErrUnsupported {
			t.Fatalf("unresolved direct high B validate err = %v, want ErrUnsupported", err)
		}
	})

	t.Run("b partitioned macroblock", func(t *testing.T) {
		sh := &SliceHeader{SliceTypeNoS: PictureTypeB}
		mbType := MBType16x8 | MBTypeP0L0 | MBTypeP1L0 | MBTypeP0L1 | MBTypeP1L1

		if err := validateHighFrameSliceMacroblockForReconstruct(sh, mbType, 0, 0); err != nil {
			t.Fatalf("partitioned high B validate err = %v, want nil", err)
		}
	})

	t.Run("b16x16 bidirectional macroblock", func(t *testing.T) {
		sh := &SliceHeader{SliceTypeNoS: PictureTypeB}
		mbType := MBType16x16 | MBTypeP0L0 | MBTypeP0L1

		if err := validateHighFrameSliceMacroblockForReconstruct(sh, mbType, 1, 1); err != nil {
			t.Fatalf("B16x16 high validate err = %v, want nil", err)
		}
	})

	t.Run("b16x16 l0 macroblock", func(t *testing.T) {
		sh := &SliceHeader{SliceTypeNoS: PictureTypeB}
		mbType := MBType16x16 | MBTypeP0L0

		if err := validateHighFrameSliceMacroblockForReconstruct(sh, mbType, 0, 0); err != nil {
			t.Fatalf("B16x16 L0 high validate err = %v, want nil", err)
		}
	})

	t.Run("b16x16 l1 8x8 dct residual macroblock", func(t *testing.T) {
		sh := &SliceHeader{SliceTypeNoS: PictureTypeB}
		mbType := MBType16x16 | MBTypeP0L1 | MBType8x8DCT

		if err := validateHighFrameSliceMacroblockForReconstruct(sh, mbType, 0x0b, 0x0b); err != nil {
			t.Fatalf("B16x16 L1 8x8-DCT high validate err = %v, want nil", err)
		}
	})

	t.Run("weighted partitioned p macroblock", func(t *testing.T) {
		sh := &SliceHeader{
			SliceTypeNoS: PictureTypeP,
			PPS:          &PPS{WeightedPred: 1},
			PredWeightTable: PredWeightTable{
				UseWeight: 1,
			},
		}
		mbType := MBType16x8 | MBTypeP0L0 | MBTypeP1L0

		if err := validateHighFrameSliceMacroblockForReconstruct(sh, mbType, 1, 1); err != nil {
			t.Fatalf("weighted partitioned high P validate err = %v, want nil", err)
		}
	})

	t.Run("p intra macroblock", func(t *testing.T) {
		sh := &SliceHeader{SliceTypeNoS: PictureTypeP}
		if err := validateHighFrameSliceMacroblockForReconstruct(sh, MBTypeIntra4x4, 0, 0); err != nil {
			t.Fatalf("intra high P validate err = %v, want nil", err)
		}
	})

	t.Run("b intra macroblock", func(t *testing.T) {
		sh := &SliceHeader{SliceTypeNoS: PictureTypeB}
		if err := validateHighFrameSliceMacroblockForReconstruct(sh, MBTypeIntra4x4, 0, 0); err != nil {
			t.Fatalf("intra high B validate err = %v, want nil", err)
		}
	})
}

func TestDecodeCAVLCFrameSliceHighRejectsUnsupportedBBeforeWriteback(t *testing.T) {
	for _, tt := range []struct {
		name string
		bits string
	}{
		{name: "skip", bits: "010"},
		{name: "direct without refs", bits: "111"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, 10, 1, 1, false, PictureTypeB)
			sh.RefCount = [2]uint32{1, 1}
			gb := newBitReader(cavlcBitString(tt.bits))

			_, err := m.decodeCAVLCFrameSliceHigh(&gb, dst, sh, h264FrameSliceDecodeInputHigh{SliceNum: 51})
			if !errors.Is(err, ErrUnsupported) {
				t.Fatalf("decode high CAVLC B err = %v, want ErrUnsupported", err)
			}
			assertHighBRejectUntouched(t, m)
		})
	}
}

func TestDecodeCABACFrameSliceHighRejectsUnsupportedBBeforeWriteback(t *testing.T) {
	for _, tt := range []struct {
		name string
		bits []int
	}{
		{name: "skip", bits: []int{1}},
		{name: "direct", bits: []int{0, 0}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, 10, 1, 1, false, PictureTypeB)
			sh.PPS.CABAC = 1
			sh.RefCount = [2]uint32{1, 1}
			src := &scriptedCABACSource{bits: tt.bits}

			_, err := m.decodeCABACFrameSliceHigh(src, dst, sh, h264FrameSliceDecodeInputHigh{SliceNum: 53})
			if !errors.Is(err, ErrUnsupported) {
				t.Fatalf("decode high CABAC B err = %v, want ErrUnsupported", err)
			}
			assertHighBRejectUntouched(t, m)
		})
	}
}

func TestDecodeFrameSliceHighReconstructsBSkipFromDirectRefs(t *testing.T) {
	for _, tt := range []struct {
		name            string
		directSpatial   bool
		cabac           bool
		implicitDeblock bool
	}{
		{name: "cavlc-temporal"},
		{name: "cavlc-spatial", directSpatial: true},
		{name: "cabac-temporal", cabac: true},
		{name: "cabac-spatial", directSpatial: true, cabac: true},
		{name: "cavlc-implicit-deblock-temporal", implicitDeblock: true},
		{name: "cabac-implicit-deblock-spatial", directSpatial: true, cabac: true, implicitDeblock: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, 10, 1, 1, false, PictureTypeB)
			sh.QScale = 22
			sh.RefCount = [2]uint32{1, 1}
			if tt.cabac {
				sh.PPS.CABAC = 1
			}
			if tt.implicitDeblock {
				sh.DeblockingFilter = 1
				sh.PPS.WeightedBipredIDC = 2
			}
			refs, direct := highBSkipDirectRefsHigh(t, tt.directSpatial)
			in := h264FrameSliceDecodeInputHigh{
				SliceNum:      61,
				Refs:          refs,
				Direct:        direct,
				MotionScratch: makeH264MotionCompScratchHigh(dst),
			}
			if tt.implicitDeblock {
				in.PredWeight = neutralImplicitBipredWeightTable()
			}

			var got h264FrameSliceDecodeResult
			var err error
			if tt.cabac {
				src := &scriptedCABACSource{bits: []int{1}, terms: []int{1}}
				got, err = m.decodeCABACFrameSliceHigh(src, dst, sh, in)
				if err == nil {
					wantIndexes(t, src, []int{24})
				}
			} else {
				gb := newBitReader(cavlcBitString("010"))
				got, err = m.decodeCAVLCFrameSliceHigh(&gb, dst, sh, in)
				if err == nil && gb.bitPos != 3 {
					t.Fatalf("CAVLC B-skip consumed %d bits, want 3", gb.bitPos)
				}
			}
			if err != nil {
				t.Fatalf("decode high B-skip failed: %v", err)
			}
			if got.Macroblocks != 1 || got.LastMBXY != 0 || !got.EndOfSlice || !got.EndOfFrame {
				t.Fatalf("slice result = %+v, want one skipped B MB frame end", got)
			}
			if !isHighB16x16DirectSkipMacroblock(m.MacroblockTyp[0]) || m.CBPTable[0] != 0 || m.QScaleTable[0] != 22 || m.SliceTable[0] != 61 {
				t.Fatalf("tables type/cbp/q/slice = %#x/%#x/%d/%d", m.MacroblockTyp[0], m.CBPTable[0], m.QScaleTable[0], m.SliceTable[0])
			}
			assertH264RowsHigh(t, tt.name+" high bskip y", dst.Y, 0, dst.LumaStride, 16, 16, refs[0][0].Y, refs[0][0].LumaStride)
			assertH264RowsHigh(t, tt.name+" high bskip cb", dst.Cb, 0, dst.ChromaStride, 8, 8, refs[0][0].Cb, refs[0][0].ChromaStride)
			assertH264RowsHigh(t, tt.name+" high bskip cr", dst.Cr, 0, dst.ChromaStride, 8, 8, refs[0][0].Cr, refs[0][0].ChromaStride)
		})
	}
}

func neutralImplicitBipredWeightTable() *PredWeightTable {
	pwt := &PredWeightTable{UseWeight: 2, UseWeightChroma: 2}
	pwt.ImplicitWeight[0][0][0] = 32
	pwt.ImplicitWeight[0][0][1] = 32
	return pwt
}

func TestDecodeFrameSliceHighReconstructsBDirectSubFromDirectRefs(t *testing.T) {
	for _, tt := range []struct {
		name               string
		cabac              bool
		direct8x8Inference bool
		wantCAVLCBits      int
	}{
		{
			name:               "cavlc-b8x8-temporal",
			direct8x8Inference: true,
			wantCAVLCBits:      15,
		},
		{
			name:               "cavlc-b4x4-temporal",
			direct8x8Inference: false,
			wantCAVLCBits:      15,
		},
		{
			name:               "cabac-b8x8-temporal",
			cabac:              true,
			direct8x8Inference: true,
		},
		{
			name:               "cabac-b4x4-temporal",
			cabac:              true,
			direct8x8Inference: false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, 10, 1, 1, false, PictureTypeB)
			sh.QScale = 22
			sh.RefCount = [2]uint32{1, 1}
			if tt.cabac {
				sh.PPS.CABAC = 1
			}
			if tt.direct8x8Inference {
				sh.SPS.Direct8x8InferenceFlag = 1
			}
			refs, direct := highBSkipDirectRefsHigh(t, false)
			direct.Direct8x8Inference = tt.direct8x8Inference
			in := h264FrameSliceDecodeInputHigh{
				SliceNum:      62,
				Refs:          refs,
				Direct:        direct,
				MotionScratch: makeH264MotionCompScratchHigh(dst),
			}

			var got h264FrameSliceDecodeResult
			var err error
			if tt.cabac {
				src := &scriptedCABACSource{
					bits: []int{
						0,
						1, 1, 1, 1, 1, 1,
						0, 0, 0, 0,
						0, 0, 0, 0,
						0,
					},
					terms: []int{1},
				}
				got, err = m.decodeCABACFrameSliceHigh(src, dst, sh, in)
			} else {
				gb := newBitReader(cavlcBitString("100001011111111"))
				got, err = m.decodeCAVLCFrameSliceHigh(&gb, dst, sh, in)
				if err == nil && gb.bitPos != uint32(tt.wantCAVLCBits) {
					t.Fatalf("CAVLC B direct-sub consumed %d bits, want %d", gb.bitPos, tt.wantCAVLCBits)
				}
			}
			if err != nil {
				t.Fatalf("decode high B direct-sub failed: %v", err)
			}
			if got.Macroblocks != 1 || got.LastMBXY != 0 || !got.EndOfSlice || !got.EndOfFrame {
				t.Fatalf("slice result = %+v, want one B direct-sub MB frame end", got)
			}
			wantMBType := MBType8x8 | MBTypeP0L0 | MBTypeP0L1 | MBTypeP1L0 | MBTypeP1L1
			if m.MacroblockTyp[0] != wantMBType || m.CBPTable[0] != 0 || m.QScaleTable[0] != 22 || m.SliceTable[0] != 62 {
				t.Fatalf("tables type/cbp/q/slice = %#x/%#x/%d/%d", m.MacroblockTyp[0], m.CBPTable[0], m.QScaleTable[0], m.SliceTable[0])
			}
			assertH264RowsHigh(t, tt.name+" high direct-sub y", dst.Y, 0, dst.LumaStride, 16, 16, refs[0][0].Y, refs[0][0].LumaStride)
			assertH264RowsHigh(t, tt.name+" high direct-sub cb", dst.Cb, 0, dst.ChromaStride, 8, 8, refs[0][0].Cb, refs[0][0].ChromaStride)
			assertH264RowsHigh(t, tt.name+" high direct-sub cr", dst.Cr, 0, dst.ChromaStride, 8, 8, refs[0][0].Cr, refs[0][0].ChromaStride)
		})
	}
}

func TestDecodeCABACFrameSliceHighReconstructsBDirectSubResidualFromDirectRefs(t *testing.T) {
	const bitDepth = 10
	const decodedLumaDelta = 5
	const decodedResidualDC = (decodedLumaDelta << 6) - 32
	m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, bitDepth, 1, 1, false, PictureTypeB)
	sh.PPS.CABAC = 1
	sh.QScale = 22
	sh.RefCount = [2]uint32{1, 1}
	sh.SPS.Direct8x8InferenceFlag = 1

	refs, direct := highBSkipDirectRefsHigh(t, false)
	direct.Direct8x8Inference = true
	in := h264FrameSliceDecodeInputHigh{
		SliceNum:      65,
		Refs:          refs,
		Direct:        direct,
		MotionScratch: makeH264MotionCompScratchHigh(dst),
	}
	src := &scriptedCABACSource{
		bits: []int{
			0,
			1, 1, 1, 1, 1, 1,
			0, 0, 0, 0,
			1, 0, 0, 0,
			0,
			0,
			1, 1, 1, 0, 0, 0, 0,
		},
		signs: []int32{int32((decodedResidualDC << 6) - 32)},
		terms: []int{1},
	}

	got, err := m.decodeCABACFrameSliceHigh(src, dst, sh, in)
	if err != nil {
		t.Fatalf("decode high B direct-sub residual failed: %v", err)
	}
	if got.Macroblocks != 1 || got.LastMBXY != 0 || !got.EndOfSlice || !got.EndOfFrame {
		t.Fatalf("slice result = %+v, want one B direct-sub residual MB frame end", got)
	}
	wantMBType := MBType8x8 | MBTypeP0L0 | MBTypeP0L1 | MBTypeP1L0 | MBTypeP1L1
	if m.MacroblockTyp[0] != wantMBType || m.CBPTable[0] != 0x1 || m.QScaleTable[0] != 22 || m.SliceTable[0] != 65 {
		t.Fatalf("tables type/cbp/q/slice = %#x/%#x/%d/%d", m.MacroblockTyp[0], m.CBPTable[0], m.QScaleTable[0], m.SliceTable[0])
	}

	want := cloneH264HighResidualPicture(refs[0][0])
	applyH264HighP16x16LumaResidualExpected(t, want, []h264HighResidualLumaBlock{{index: 0, dc: decodedLumaDelta}}, bitDepth)
	assertH264RowsHigh(t, "cabac high b direct-sub decoded residual y", dst.Y, 0, dst.LumaStride, 16, 16, want.Y, want.LumaStride)
	assertH264RowsHigh(t, "cabac high b direct-sub decoded residual cb", dst.Cb, 0, dst.ChromaStride, 8, 8, refs[0][0].Cb, refs[0][0].ChromaStride)
	assertH264RowsHigh(t, "cabac high b direct-sub decoded residual cr", dst.Cr, 0, dst.ChromaStride, 8, 8, refs[0][0].Cr, refs[0][0].ChromaStride)
}

func TestDecodeCABACFrameSliceHighReconstructsImplicitDeblockBDirectSubResidualFromDirectRefs(t *testing.T) {
	const bitDepth = 10
	const decodedLumaDelta = 5
	const decodedResidualDC = (decodedLumaDelta << 6) - 32
	m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, bitDepth, 1, 1, false, PictureTypeB)
	sh.PPS.CABAC = 1
	sh.PPS.WeightedBipredIDC = 2
	sh.DeblockingFilter = 1
	sh.QScale = 22
	sh.RefCount = [2]uint32{1, 1}
	sh.SPS.Direct8x8InferenceFlag = 1

	refs, direct := highBSkipDirectRefsHigh(t, false)
	direct.Direct8x8Inference = true
	fillH264HighResidualPlane(refs[1][0].Y, 677)
	fillH264HighResidualPlane(refs[1][0].Cb, 711)
	fillH264HighResidualPlane(refs[1][0].Cr, 753)
	pwt := &PredWeightTable{UseWeight: 2, UseWeightChroma: 2}
	pwt.ImplicitWeight[0][0][0] = 21
	pwt.ImplicitWeight[0][0][1] = 21
	in := h264FrameSliceDecodeInputHigh{
		SliceNum:      66,
		Refs:          refs,
		Direct:        direct,
		PredWeight:    pwt,
		MotionScratch: makeH264MotionCompScratchHigh(dst),
	}
	src := &scriptedCABACSource{
		bits: []int{
			0,
			1, 1, 1, 1, 1, 1,
			0, 0, 0, 0,
			1, 0, 0, 0,
			0,
			0,
			1, 1, 1, 0, 0, 0, 0,
		},
		signs: []int32{int32((decodedResidualDC << 6) - 32)},
		terms: []int{1},
	}

	got, err := m.decodeCABACFrameSliceHigh(src, dst, sh, in)
	if err != nil {
		t.Fatalf("decode implicit deblock high B direct-sub residual failed: %v", err)
	}
	if got.Macroblocks != 1 || got.LastMBXY != 0 || !got.EndOfSlice || !got.EndOfFrame {
		t.Fatalf("slice result = %+v, want one implicit deblock B direct-sub residual MB frame end", got)
	}
	wantMBType := MBType8x8 | MBTypeP0L0 | MBTypeP0L1 | MBTypeP1L0 | MBTypeP1L1
	if m.MacroblockTyp[0] != wantMBType || m.CBPTable[0] != 0x1 || m.QScaleTable[0] != 22 || m.SliceTable[0] != 66 {
		t.Fatalf("tables type/cbp/q/slice = %#x/%#x/%d/%d", m.MacroblockTyp[0], m.CBPTable[0], m.QScaleTable[0], m.SliceTable[0])
	}

	want := makeH264SliceDecodePictureHigh(1, 1, 1)
	fillH264HighResidualPlane(want.Y, h264HighImplicitBWeightSample(refs[0][0].Y[0], refs[1][0].Y[0], 21, bitDepth))
	fillH264HighResidualPlane(want.Cb, h264HighImplicitBWeightSample(refs[0][0].Cb[0], refs[1][0].Cb[0], 21, bitDepth))
	fillH264HighResidualPlane(want.Cr, h264HighImplicitBWeightSample(refs[0][0].Cr[0], refs[1][0].Cr[0], 21, bitDepth))
	applyH264HighP16x16LumaResidualExpected(t, want, []h264HighResidualLumaBlock{{index: 0, dc: decodedLumaDelta}}, bitDepth)
	assertH264RowsHigh(t, "cabac implicit deblock high b direct-sub residual y", dst.Y, 0, dst.LumaStride, 16, 16, want.Y, want.LumaStride)
	assertH264RowsHigh(t, "cabac implicit deblock high b direct-sub residual cb", dst.Cb, 0, dst.ChromaStride, 8, 8, want.Cb, want.ChromaStride)
	assertH264RowsHigh(t, "cabac implicit deblock high b direct-sub residual cr", dst.Cr, 0, dst.ChromaStride, 8, 8, want.Cr, want.ChromaStride)
}

func TestDecodeFrameSliceHighReconstructsTopLevelBDirect8x8FromDirectRefs(t *testing.T) {
	for _, tt := range []struct {
		name               string
		cabac              bool
		direct8x8Inference bool
	}{
		{
			name:               "cavlc-direct-8x8-inference",
			direct8x8Inference: true,
		},
		{
			name: "cavlc-direct-sub-4x4",
		},
		{
			name:               "cabac-direct-8x8-inference",
			cabac:              true,
			direct8x8Inference: true,
		},
		{
			name:  "cabac-direct-sub-4x4",
			cabac: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, 10, 1, 1, false, PictureTypeB)
			sh.QScale = 22
			sh.RefCount = [2]uint32{1, 1}
			sh.SPS.Direct8x8InferenceFlag = boolToInt32(tt.direct8x8Inference)
			if tt.cabac {
				sh.PPS.CABAC = 1
			}
			refs, direct := highBSkipDirectRefsHigh(t, false)
			direct.Direct8x8Inference = tt.direct8x8Inference
			direct.RefEntries[1][0].frame.tables.MacroblockTyp[0] = MBType8x8 | MBTypeP0L0
			in := h264FrameSliceDecodeInputHigh{
				SliceNum:      63,
				Refs:          refs,
				Direct:        direct,
				MotionScratch: makeH264MotionCompScratchHigh(dst),
			}

			var got h264FrameSliceDecodeResult
			var err error
			if tt.cabac {
				src := &scriptedCABACSource{
					bits:  []int{0, 0, 0, 0, 0, 0, 0},
					terms: []int{1},
				}
				got, err = m.decodeCABACFrameSliceHigh(src, dst, sh, in)
			} else {
				gb := newBitReader(cavlcBitString("111"))
				got, err = m.decodeCAVLCFrameSliceHigh(&gb, dst, sh, in)
				if err == nil && gb.bitPos != 3 {
					t.Fatalf("CAVLC top-level B direct consumed %d bits, want 3", gb.bitPos)
				}
			}
			if err != nil {
				t.Fatalf("decode high top-level B direct 8x8 failed: %v", err)
			}
			if got.Macroblocks != 1 || got.LastMBXY != 0 || !got.EndOfSlice || !got.EndOfFrame {
				t.Fatalf("slice result = %+v, want one top-level B direct MB frame end", got)
			}
			wantMBType := MBType8x8 | MBTypeL0L1 | MBTypeDirect2
			if m.MacroblockTyp[0] != wantMBType || m.CBPTable[0] != 0 || m.QScaleTable[0] != 22 || m.SliceTable[0] != 63 {
				t.Fatalf("tables type/cbp/q/slice = %#x/%#x/%d/%d", m.MacroblockTyp[0], m.CBPTable[0], m.QScaleTable[0], m.SliceTable[0])
			}
			assertH264RowsHigh(t, tt.name+" high top-level direct y", dst.Y, 0, dst.LumaStride, 16, 16, refs[0][0].Y, refs[0][0].LumaStride)
			assertH264RowsHigh(t, tt.name+" high top-level direct cb", dst.Cb, 0, dst.ChromaStride, 8, 8, refs[0][0].Cb, refs[0][0].ChromaStride)
			assertH264RowsHigh(t, tt.name+" high top-level direct cr", dst.Cr, 0, dst.ChromaStride, 8, 8, refs[0][0].Cr, refs[0][0].ChromaStride)
		})
	}
}

func TestDecodeFrameSliceHighReconstructsPartitionedBExplicit(t *testing.T) {
	b8x8 := MBType8x8 | MBTypeP0L0 | MBTypeP0L1 | MBTypeP1L0 | MBTypeP1L1
	allL0Sub := [4]uint32{
		MBType16x16 | MBTypeP0L0,
		MBType16x16 | MBTypeP0L0,
		MBType16x16 | MBTypeP0L0,
		MBType16x16 | MBTypeP0L0,
	}

	for _, tt := range []struct {
		name          string
		cabac         bool
		cavlcBits     string
		cabacBits     []int
		wantCAVLCBits int
		wantMBType    uint32
		assert        func(t *testing.T, label string, dst *h264PicturePlanesHigh, refs [2][]*h264PicturePlanesHigh)
	}{
		{
			name:          "cavlc-b16x8-l0-l1",
			cavlcBits:     "1000100111111",
			wantCAVLCBits: 13,
			wantMBType:    MBType16x8 | MBTypeP0L0 | MBTypeP1L1,
			assert:        assertHighPartitionedB16x8L0L1,
		},
		{
			name:          "cavlc-b8x16-l0-l1",
			cavlcBits:     "1000101011111",
			wantCAVLCBits: 13,
			wantMBType:    MBType8x16 | MBTypeP0L0 | MBTypeP1L1,
			assert:        assertHighPartitionedB8x16L0L1,
		},
		{
			name:          "cavlc-b8x8-l0",
			cavlcBits:     "1000010111010010010010111111111",
			wantCAVLCBits: 31,
			wantMBType:    b8x8,
			assert:        assertHighPartitionedB8x8L0,
		},
		{
			name:       "cabac-b16x8-l0-l1",
			cabac:      true,
			cabacBits:  []int{0, 1, 1, 0, 1, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			wantMBType: MBType16x8 | MBTypeP0L0 | MBTypeP1L1,
			assert:     assertHighPartitionedB16x8L0L1,
		},
		{
			name:       "cabac-b8x16-l0-l1",
			cabac:      true,
			cabacBits:  []int{0, 1, 1, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			wantMBType: MBType8x16 | MBTypeP0L0 | MBTypeP1L1,
			assert:     assertHighPartitionedB8x16L0L1,
		},
		{
			name:       "cabac-b8x8-l0",
			cabac:      true,
			cabacBits:  []int{0, 1, 1, 1, 1, 1, 1, 1, 0, 0, 1, 0, 0, 1, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			wantMBType: b8x8,
			assert:     assertHighPartitionedB8x8L0,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, 10, 1, 1, false, PictureTypeB)
			sh.QScale = 22
			sh.RefCount = [2]uint32{1, 1}
			if tt.cabac {
				sh.PPS.CABAC = 1
			}
			refs := highPartitionedBRefsHigh(t)
			in := h264FrameSliceDecodeInputHigh{
				SliceNum:      64,
				Refs:          refs,
				MotionScratch: makeH264MotionCompScratchHigh(dst),
			}

			var got h264FrameSliceDecodeResult
			var err error
			if tt.cabac {
				got, err = m.decodeCABACFrameSliceHigh(&scriptedCABACSource{bits: tt.cabacBits, terms: []int{1}}, dst, sh, in)
			} else {
				gb := newBitReader(cavlcBitString(tt.cavlcBits))
				got, err = m.decodeCAVLCFrameSliceHigh(&gb, dst, sh, in)
				if err == nil && gb.bitPos != uint32(tt.wantCAVLCBits) {
					t.Fatalf("CAVLC partitioned B consumed %d bits, want %d", gb.bitPos, tt.wantCAVLCBits)
				}
			}
			if err != nil {
				t.Fatalf("decode high partitioned B failed: %v", err)
			}
			if got.Macroblocks != 1 || got.LastMBXY != 0 || !got.EndOfSlice || !got.EndOfFrame {
				t.Fatalf("slice result = %+v, want one partitioned B MB frame end", got)
			}
			if m.MacroblockTyp[0] != tt.wantMBType || m.CBPTable[0] != 0 || m.QScaleTable[0] != 22 || m.SliceTable[0] != 64 {
				t.Fatalf("tables type/cbp/q/slice = %#x/%#x/%d/%d", m.MacroblockTyp[0], m.CBPTable[0], m.QScaleTable[0], m.SliceTable[0])
			}
			if tt.wantMBType == b8x8 {
				for i := 0; i < 4; i++ {
					if got := m.RefIndex[0][i]; got != 0 {
						t.Fatalf("sub[%d] ref0 = %d, want 0", i, got)
					}
				}
				if err := validateHighFrameSliceMacroblockForReconstructWithSubMB(sh, b8x8, &allL0Sub, 0, 0); err != nil {
					t.Fatalf("validate explicit B8x8 subtypes err = %v, want nil", err)
				}
			}
			tt.assert(t, tt.name, dst, refs)
		})
	}
}

func TestDecodeFrameSliceHighReconstructsPartitionedBSkip(t *testing.T) {
	for _, tt := range []struct {
		name  string
		cabac bool
	}{
		{name: "cavlc"},
		{name: "cabac", cabac: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, 10, 1, 1, false, PictureTypeB)
			sh.RefCount = [2]uint32{1, 1}
			if tt.cabac {
				sh.PPS.CABAC = 1
			}
			refs, direct := highBSkipDirectRefsHigh(t, false)
			direct.RefEntries[1][0].frame.tables.MacroblockTyp[0] = MBType8x8 | MBTypeP0L0
			direct.Direct8x8Inference = false
			in := h264FrameSliceDecodeInputHigh{
				SliceNum:      63,
				Refs:          refs,
				Direct:        direct,
				MotionScratch: makeH264MotionCompScratchHigh(dst),
			}

			var err error
			var got h264FrameSliceDecodeResult
			if tt.cabac {
				got, err = m.decodeCABACFrameSliceHigh(&scriptedCABACSource{bits: []int{1}, terms: []int{1}}, dst, sh, in)
			} else {
				gb := newBitReader(cavlcBitString("010"))
				got, err = m.decodeCAVLCFrameSliceHigh(&gb, dst, sh, in)
			}
			if err != nil {
				t.Fatalf("partitioned B-skip decode err = %v, want nil", err)
			}
			if got.Macroblocks != 1 || m.CBPTable[0] != 0 || m.QScaleTable[0] != uint8(sh.QScale) || m.SliceTable[0] != 63 {
				t.Fatalf("partitioned B-skip result/tables = %+v cbp=%#x q=%d slice=%d", got, m.CBPTable[0], m.QScaleTable[0], m.SliceTable[0])
			}
		})
	}
}

func highPartitionedBRefsHigh(t *testing.T) [2][]*h264PicturePlanesHigh {
	t.Helper()

	past := makeH264SliceDecodePictureHigh(1, 1, 1)
	future := makeH264SliceDecodePictureHigh(1, 1, 1)
	fillH264HighResidualPlane(past.Y, 101)
	fillH264HighResidualPlane(past.Cb, 151)
	fillH264HighResidualPlane(past.Cr, 201)
	fillH264HighResidualPlane(future.Y, 701)
	fillH264HighResidualPlane(future.Cb, 751)
	fillH264HighResidualPlane(future.Cr, 801)
	return [2][]*h264PicturePlanesHigh{{past}, {future}}
}

func assertHighPartitionedB16x8L0L1(t *testing.T, label string, dst *h264PicturePlanesHigh, refs [2][]*h264PicturePlanesHigh) {
	t.Helper()

	assertH264RowsHigh(t, label+" top y", dst.Y, 0, dst.LumaStride, 16, 8, refs[0][0].Y, refs[0][0].LumaStride)
	assertH264RowsHigh(t, label+" bottom y", dst.Y, 8*dst.LumaStride, dst.LumaStride, 16, 8, refs[1][0].Y, refs[1][0].LumaStride)
	assertH264RowsHigh(t, label+" top cb", dst.Cb, 0, dst.ChromaStride, 8, 4, refs[0][0].Cb, refs[0][0].ChromaStride)
	assertH264RowsHigh(t, label+" bottom cb", dst.Cb, 4*dst.ChromaStride, dst.ChromaStride, 8, 4, refs[1][0].Cb, refs[1][0].ChromaStride)
	assertH264RowsHigh(t, label+" top cr", dst.Cr, 0, dst.ChromaStride, 8, 4, refs[0][0].Cr, refs[0][0].ChromaStride)
	assertH264RowsHigh(t, label+" bottom cr", dst.Cr, 4*dst.ChromaStride, dst.ChromaStride, 8, 4, refs[1][0].Cr, refs[1][0].ChromaStride)
}

func assertHighPartitionedB8x16L0L1(t *testing.T, label string, dst *h264PicturePlanesHigh, refs [2][]*h264PicturePlanesHigh) {
	t.Helper()

	assertH264RowsHigh(t, label+" left y", dst.Y, 0, dst.LumaStride, 8, 16, refs[0][0].Y, refs[0][0].LumaStride)
	assertH264RowsHigh(t, label+" right y", dst.Y, 8, dst.LumaStride, 8, 16, refs[1][0].Y, refs[1][0].LumaStride)
	assertH264RowsHigh(t, label+" left cb", dst.Cb, 0, dst.ChromaStride, 4, 8, refs[0][0].Cb, refs[0][0].ChromaStride)
	assertH264RowsHigh(t, label+" right cb", dst.Cb, 4, dst.ChromaStride, 4, 8, refs[1][0].Cb, refs[1][0].ChromaStride)
	assertH264RowsHigh(t, label+" left cr", dst.Cr, 0, dst.ChromaStride, 4, 8, refs[0][0].Cr, refs[0][0].ChromaStride)
	assertH264RowsHigh(t, label+" right cr", dst.Cr, 4, dst.ChromaStride, 4, 8, refs[1][0].Cr, refs[1][0].ChromaStride)
}

func assertHighPartitionedB8x8L0(t *testing.T, label string, dst *h264PicturePlanesHigh, refs [2][]*h264PicturePlanesHigh) {
	t.Helper()

	assertH264RowsHigh(t, label+" y", dst.Y, 0, dst.LumaStride, 16, 16, refs[0][0].Y, refs[0][0].LumaStride)
	assertH264RowsHigh(t, label+" cb", dst.Cb, 0, dst.ChromaStride, 8, 8, refs[0][0].Cb, refs[0][0].ChromaStride)
	assertH264RowsHigh(t, label+" cr", dst.Cr, 0, dst.ChromaStride, 8, 8, refs[0][0].Cr, refs[0][0].ChromaStride)
}

func assertHighBRejectUntouched(t *testing.T, m *macroblockTables) {
	t.Helper()
	if m.MacroblockTyp[0] != 0 || m.CBPTable[0] != 0 || m.QScaleTable[0] != 0 || m.SliceTable[0] != ^uint16(0) {
		t.Fatalf("tables type/cbp/q/slice = %#x/%#x/%d/%#x, want untouched",
			m.MacroblockTyp[0], m.CBPTable[0], m.QScaleTable[0], m.SliceTable[0])
	}
}

func highBSkipDirectRefsHigh(t *testing.T, directSpatial bool) ([2][]*h264PicturePlanesHigh, h264DirectMotionContext) {
	t.Helper()

	past := makeH264SliceDecodePictureHigh(1, 1, 1)
	future := makeH264SliceDecodePictureHigh(1, 1, 1)
	fillH264HighResidualPlane(past.Y, 277)
	fillH264HighResidualPlane(past.Cb, 311)
	fillH264HighResidualPlane(past.Cr, 353)
	fillH264HighResidualPlane(future.Y, 277)
	fillH264HighResidualPlane(future.Cb, 311)
	fillH264HighResidualPlane(future.Cr, 353)

	colTables, err := newMacroblockTables(1, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	colTables.MacroblockTyp[0] = MBType16x16 | MBTypeP0L0 | MBTypeP1L0 | MBTypeSkip
	for i := 0; i < 4; i++ {
		colTables.RefIndex[0][i] = 0
	}
	pastFrame := decodedFrameFromHighPlanes(past, 0, nil)
	futureFrame := decodedFrameFromHighPlanes(future, 4, colTables)
	futureFrame.refEntries = [2][]simpleRefEntry{{{frame: pastFrame}}}

	return [2][]*h264PicturePlanesHigh{{past}, {future}}, h264DirectMotionContext{
		RefEntries: [2][]simpleRefEntry{
			{{frame: pastFrame}},
			{{frame: futureFrame}},
		},
		CurPOC:              2,
		DirectSpatialMVPred: directSpatial,
		Direct8x8Inference:  true,
		X264Build:           165,
	}
}

func decodedFrameFromHighPlanes(p *h264PicturePlanesHigh, poc int32, tables *macroblockTables) *DecodedFrame {
	if p == nil {
		return &DecodedFrame{}
	}
	return &DecodedFrame{
		Y16:             p.Y,
		Cb16:            p.Cb,
		Cr16:            p.Cr,
		LumaStride:      p.LumaStride,
		ChromaStride:    p.ChromaStride,
		Width:           p.MBWidth * 16,
		Height:          p.MBHeight * 16,
		MBWidth:         p.MBWidth,
		MBHeight:        p.MBHeight,
		ChromaFormatIDC: p.ChromaFormatIDC,
		BitDepthLuma:    10,
		BitDepthChroma:  10,
		poc:             poc,
		tables:          tables,
	}
}

type h264HighResidualLumaBlock struct {
	index int
	dc    int
}

func h264HighP16x16LumaResidualWork() (frameMacroblockDecodeWork, []h264HighResidualLumaBlock) {
	blocks := []h264HighResidualLumaBlock{
		{index: 0, dc: 5},
		{index: 5, dc: -2},
	}
	var work frameMacroblockDecodeWork
	work.Motion.Ref[0][h264Scan8[0]] = 0
	work.Motion.MV[0][h264Scan8[0]] = [2]int16{0, 0}
	for _, block := range blocks {
		work.Residual.NonZeroCountCache[h264Scan8[block.index]] = 1
		work.Residual.MB[block.index*16] = int32((block.dc << 6) - 32)
	}
	return work, blocks
}

func applyH264HighP16x16LumaResidualExpected(t *testing.T, pic *h264PicturePlanesHigh, blocks []h264HighResidualLumaBlock, bitDepth int) {
	t.Helper()

	offsets, err := h264FrameBlockOffsets(pic.LumaStride, pic.ChromaStride, 0)
	if err != nil {
		t.Fatal(err)
	}
	max := (1 << uint(bitDepth)) - 1
	for _, block := range blocks {
		offset := offsets[block.index]
		for y := 0; y < 4; y++ {
			for x := 0; x < 4; x++ {
				i := offset + y*pic.LumaStride + x
				v := int(pic.Y[i]) + block.dc
				if v < 0 || v > max {
					t.Fatalf("expected residual sample clips: block=%d sample=%d value=%d", block.index, i, v)
				}
				pic.Y[i] = uint16(v)
			}
		}
	}
}

func h264HighImplicitBWeightSample(past uint16, future uint16, weight0 int, bitDepth int) uint16 {
	weight1 := 64 - weight0
	shift := bitDepth - 8
	offset := int(int32(uint32(0) << uint(shift)))
	scaledOffset := int(int32(uint32((offset+1)|1) << 5))
	return clipUintBitDepth((int(future)*weight1+int(past)*weight0+scaledOffset)>>6, bitDepth)
}

func fillH264HighResidualPlane(p []uint16, v uint16) {
	for i := range p {
		p[i] = v
	}
}

func cloneH264HighResidualPicture(src *h264PicturePlanesHigh) *h264PicturePlanesHigh {
	dst := *src
	dst.Y = append([]uint16(nil), src.Y...)
	dst.Cb = append([]uint16(nil), src.Cb...)
	dst.Cr = append([]uint16(nil), src.Cr...)
	return &dst
}
