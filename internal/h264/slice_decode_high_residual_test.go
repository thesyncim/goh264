// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

import "testing"

func TestHighP16x16ResidualHandoffReconstructsExactLuma(t *testing.T) {
	const bitDepth = 10
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
			if reconstruct.MBType != mbType || reconstruct.CBP != cbp || reconstruct.BitDepth != bitDepth || reconstruct.PredWeight != nil || reconstruct.DeblockingFilter {
				t.Fatalf("handoff = type %#x cbp %#x depth %d pwt %v deblock %v",
					reconstruct.MBType, reconstruct.CBP, reconstruct.BitDepth, reconstruct.PredWeight, reconstruct.DeblockingFilter)
			}

			if err := h264HLDecodeFrameMacroblockHigh(dst, reconstruct); err != nil {
				t.Fatalf("reconstruct high P16x16 residual failed: %v", err)
			}

			want := cloneH264HighResidualPicture(ref)
			applyH264HighP16x16LumaResidualExpected(t, want, changed, bitDepth)
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

func TestHighResidualLaneRejectsUnsupportedBoundaries(t *testing.T) {
	t.Run("deblock enabled", func(t *testing.T) {
		m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, 10, 1, 1, true, PictureTypeP)
		sh.RefCount = [2]uint32{1, 0}
		gb := newBitReader([]byte{0xff})

		_, err := m.decodeCAVLCFrameSliceHigh(&gb, dst, sh, h264FrameSliceDecodeInputHigh{SliceNum: 43})
		if err != ErrUnsupported {
			t.Fatalf("deblock high P decode err = %v, want ErrUnsupported", err)
		}
		if gb.bitPos != 0 {
			t.Fatalf("deblock high P consumed %d bits, want 0", gb.bitPos)
		}
	})

	t.Run("b direct macroblock", func(t *testing.T) {
		sh := &SliceHeader{SliceTypeNoS: PictureTypeB}
		mbType := MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2

		if err := validateHighFrameSliceMacroblockForReconstruct(sh, mbType, 0, 0); err != ErrUnsupported {
			t.Fatalf("direct high B validate err = %v, want ErrUnsupported", err)
		}
	})

	t.Run("b partitioned macroblock", func(t *testing.T) {
		sh := &SliceHeader{SliceTypeNoS: PictureTypeB}
		mbType := MBType16x8 | MBTypeP0L0 | MBTypeP1L0 | MBTypeP0L1 | MBTypeP1L1

		if err := validateHighFrameSliceMacroblockForReconstruct(sh, mbType, 0, 0); err != ErrUnsupported {
			t.Fatalf("partitioned high B validate err = %v, want ErrUnsupported", err)
		}
	})

	t.Run("b16x16 bidirectional macroblock", func(t *testing.T) {
		sh := &SliceHeader{SliceTypeNoS: PictureTypeB}
		mbType := MBType16x16 | MBTypeP0L0 | MBTypeP0L1

		if err := validateHighFrameSliceMacroblockForReconstruct(sh, mbType, 1, 1); err != nil {
			t.Fatalf("B16x16 high validate err = %v, want nil", err)
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

		if err := validateHighFrameSliceMacroblockForReconstruct(sh, mbType, 1, 1); err != ErrUnsupported {
			t.Fatalf("partitioned high P validate err = %v, want ErrUnsupported", err)
		}
	})
}

func TestDecodeCAVLCFrameSliceHighRejectsUnsupportedBBeforeWriteback(t *testing.T) {
	for _, tt := range []struct {
		name string
		bits string
	}{
		{name: "skip", bits: "010"},
		{name: "direct", bits: "11"},
		{name: "l1 only", bits: "1011"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, 10, 1, 1, false, PictureTypeB)
			sh.RefCount = [2]uint32{1, 1}
			gb := newBitReader(cavlcBitString(tt.bits))

			_, err := m.decodeCAVLCFrameSliceHigh(&gb, dst, sh, h264FrameSliceDecodeInputHigh{SliceNum: 51})
			if err != ErrUnsupported {
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
		{name: "l1 only", bits: []int{0, 1, 0, 1}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			m, dst, sh := highFrameSliceDecodeFixtureWithMBWidth(t, 10, 1, 1, false, PictureTypeB)
			sh.PPS.CABAC = 1
			sh.RefCount = [2]uint32{1, 1}
			src := &scriptedCABACSource{bits: tt.bits}

			_, err := m.decodeCABACFrameSliceHigh(src, dst, sh, h264FrameSliceDecodeInputHigh{SliceNum: 53})
			if err != ErrUnsupported {
				t.Fatalf("decode high CABAC B err = %v, want ErrUnsupported", err)
			}
			assertHighBRejectUntouched(t, m)
		})
	}
}

func assertHighBRejectUntouched(t *testing.T, m *macroblockTables) {
	t.Helper()
	if m.MacroblockTyp[0] != 0 || m.CBPTable[0] != 0 || m.QScaleTable[0] != 0 || m.SliceTable[0] != ^uint16(0) {
		t.Fatalf("tables type/cbp/q/slice = %#x/%#x/%d/%#x, want untouched",
			m.MacroblockTyp[0], m.CBPTable[0], m.QScaleTable[0], m.SliceTable[0])
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
