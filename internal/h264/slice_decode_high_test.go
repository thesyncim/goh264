// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

import "testing"

func TestSimpleFrameSliceDecodeBitDepthGate(t *testing.T) {
	for _, tc := range []struct {
		bitDepth int32
		want     bool
	}{
		{bitDepth: 8, want: true},
		{bitDepth: 9},
		{bitDepth: 10},
		{bitDepth: 12},
		{bitDepth: 14},
	} {
		if got := h264SimpleFrameSliceDecodeSupportsBitDepth(tc.bitDepth); got != tc.want {
			t.Fatalf("supports bit depth %d = %v, want %v", tc.bitDepth, got, tc.want)
		}
	}
}

func TestValidateSimpleFrameSliceDecodeAllows8Bit(t *testing.T) {
	m, dst, sh := simpleFrameSliceDecodeBitDepthFixture(t, 8)

	if err := validateSimpleFrameSliceDecodeInputs(m, dst, sh, 4); err != nil {
		t.Fatalf("8-bit validation err = %v, want nil", err)
	}
}

func TestValidateSimpleFrameSliceDecodeRejectsHighBitDepths(t *testing.T) {
	for _, bitDepth := range []int32{9, 10, 12, 14} {
		t.Run(bitDepthName(bitDepth), func(t *testing.T) {
			m, dst, sh := simpleFrameSliceDecodeBitDepthFixture(t, bitDepth)

			if err := validateSimpleFrameSliceDecodeInputs(m, dst, sh, 4); err != ErrUnsupported {
				t.Fatalf("validation err = %v, want ErrUnsupported", err)
			}
		})
	}
}

func TestDecodeCAVLCFrameSliceRejectsHighBitDepthsAtValidation(t *testing.T) {
	for _, bitDepth := range []int32{9, 10, 12, 14} {
		t.Run(bitDepthName(bitDepth), func(t *testing.T) {
			m, dst, sh := simpleFrameSliceDecodeBitDepthFixture(t, bitDepth)
			gb := newBitReader(nil)

			_, err := m.decodeCAVLCFrameSlice(&gb, dst, sh, h264FrameSliceDecodeInput{SliceNum: 4})
			if err != ErrUnsupported {
				t.Fatalf("decode err = %v, want ErrUnsupported", err)
			}
			if gb.bitPos != 0 {
				t.Fatalf("bit reader consumed %d bits, want 0", gb.bitPos)
			}
			if m.MacroblockTyp[0] != 0 || m.SliceTable[0] != ^uint16(0) {
				t.Fatalf("tables type/slice = %#x/%#x, want untouched", m.MacroblockTyp[0], m.SliceTable[0])
			}
		})
	}
}

func TestDecodeCABACFrameSliceRejectsHighBitDepthsAtValidation(t *testing.T) {
	for _, bitDepth := range []int32{9, 10, 12, 14} {
		t.Run(bitDepthName(bitDepth), func(t *testing.T) {
			m, dst, sh := simpleFrameSliceDecodeBitDepthFixture(t, bitDepth)
			src := &scriptedCABACSource{}

			_, err := m.decodeCABACFrameSlice(src, dst, sh, h264FrameSliceDecodeInput{SliceNum: 4})
			if err != ErrUnsupported {
				t.Fatalf("decode err = %v, want ErrUnsupported", err)
			}
			if len(src.indexes) != 0 || len(src.pcmReadSizes) != 0 || len(src.terms) != 0 {
				t.Fatalf("cabac source was touched: indexes=%v pcmReads=%v terms=%v", src.indexes, src.pcmReadSizes, src.terms)
			}
			if m.MacroblockTyp[0] != 0 || m.SliceTable[0] != ^uint16(0) {
				t.Fatalf("tables type/slice = %#x/%#x, want untouched", m.MacroblockTyp[0], m.SliceTable[0])
			}
		})
	}
}

func simpleFrameSliceDecodeBitDepthFixture(t *testing.T, bitDepth int32) (*macroblockTables, *h264PicturePlanes, *SliceHeader) {
	t.Helper()

	m, err := newMacroblockTables(1, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	sps := &SPS{
		BitDepthLuma:     bitDepth,
		BitDepthChroma:   bitDepth,
		ChromaFormatIDC:  1,
		FrameMBSOnlyFlag: 1,
	}
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
	return m, makeH264SliceDecodePicture(1, 1, 1), sh
}

func bitDepthName(bitDepth int32) string {
	switch bitDepth {
	case 9:
		return "9-bit"
	case 10:
		return "10-bit"
	case 12:
		return "12-bit"
	case 14:
		return "14-bit"
	default:
		return "bit-depth"
	}
}
