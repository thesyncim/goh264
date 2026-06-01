// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

import (
	"math/bits"
	"testing"
)

func TestDecodeSPSVUIHRDBitstreamRestriction(t *testing.T) {
	rbsp := buildSPSRBSP(t, 66, 30, 2, true, func(b *spsBitBuilder) {
		b.writeBit(1)       // aspect_ratio_info_present_flag
		b.writeBits(255, 8) // Extended_SAR
		b.writeBits(4, 16)
		b.writeBits(3, 16)
		b.writeBit(1) // overscan_info_present_flag
		b.writeBit(0)
		b.writeBit(1) // video_signal_type_present_flag
		b.writeBits(5, 3)
		b.writeBit(1)
		b.writeBit(1) // colour_description_present_flag
		b.writeBits(1, 8)
		b.writeBits(1, 8)
		b.writeBits(1, 8)
		b.writeBit(1) // chroma_loc_info_present_flag
		b.writeUE(2)
		b.writeUE(3)

		b.writeBit(1) // timing_info_present_flag
		b.writeBits(1001, 32)
		b.writeBits(60000, 32)
		b.writeBit(1)

		b.writeBit(1) // nal_hrd_parameters_present_flag
		b.writeUE(1)  // cpb_cnt_minus1
		b.writeBits(5, 4)
		b.writeBits(6, 4)
		b.writeUE(99)
		b.writeUE(199)
		b.writeBit(1)
		b.writeUE(299)
		b.writeUE(399)
		b.writeBit(0)
		b.writeBits(23, 5)
		b.writeBits(9, 5)
		b.writeBits(4, 5)
		b.writeBits(12, 5)

		b.writeBit(0) // vcl_hrd_parameters_present_flag
		b.writeBit(1) // low_delay_hrd_flag
		b.writeBit(1) // pic_struct_present_flag
		b.writeBit(1) // bitstream_restriction_flag
		b.writeBit(1) // motion_vectors_over_pic_boundaries_flag
		b.writeUE(0)
		b.writeUE(1)
		b.writeUE(8)
		b.writeUE(9)
		b.writeUE(2)
		b.writeUE(4)
	})

	sps, err := DecodeSPS(rbsp)
	if err != nil {
		t.Fatal(err)
	}
	if sps.VUIParametersPresentFlag != 1 || sps.BitstreamRestrictionFlag != 1 || sps.NumReorderFrames != 2 || sps.MaxDecFrameBuffering != 4 {
		t.Fatalf("restriction = vui %d flag %d reorder %d max %d", sps.VUIParametersPresentFlag, sps.BitstreamRestrictionFlag, sps.NumReorderFrames, sps.MaxDecFrameBuffering)
	}
	if sps.VUI.SARNum != 4 || sps.VUI.SARDen != 3 || sps.VUI.VideoFormat != 5 || sps.VUI.VideoFullRangeFlag != 1 {
		t.Fatalf("vui sar/video = %+v", sps.VUI)
	}
	if sps.VUI.ColourPrimaries != 1 || sps.VUI.TransferCharacteristics != 1 || sps.VUI.MatrixCoeffs != 1 {
		t.Fatalf("vui color = prim %d trc %d matrix %d", sps.VUI.ColourPrimaries, sps.VUI.TransferCharacteristics, sps.VUI.MatrixCoeffs)
	}
	if sps.VUI.ChromaSampleLocTypeTopField != 2 || sps.VUI.ChromaSampleLocTypeBottomField != 3 || sps.VUI.ChromaLocation != 3 {
		t.Fatalf("vui chroma loc = %+v", sps.VUI)
	}
	if sps.TimingInfoPresentFlag != 1 || sps.NumUnitsInTick != 1001 || sps.TimeScale != 60000 || sps.FixedFrameRateFlag != 1 {
		t.Fatalf("timing = present %d tick %d scale %d fixed %d", sps.TimingInfoPresentFlag, sps.NumUnitsInTick, sps.TimeScale, sps.FixedFrameRateFlag)
	}
	if sps.NALHRDParametersPresentFlag != 1 || sps.VCLHRDParametersPresentFlag != 0 || sps.CPBCount != 2 || sps.BitRateScale != 5 {
		t.Fatalf("hrd = nal %d vcl %d cpb %d scale %d", sps.NALHRDParametersPresentFlag, sps.VCLHRDParametersPresentFlag, sps.CPBCount, sps.BitRateScale)
	}
	if sps.BitRateValue[0] != 100 || sps.CPBSizeValue[0] != 200 || sps.BitRateValue[1] != 300 || sps.CPBSizeValue[1] != 400 || sps.CPRFlag != 1 {
		t.Fatalf("hrd values = bitrate %v cpb %v cpr %#x", sps.BitRateValue[:2], sps.CPBSizeValue[:2], sps.CPRFlag)
	}
	if sps.InitialCPBRemovalDelayLength != 24 || sps.CPBRemovalDelayLength != 10 || sps.DPBOutputDelayLength != 5 || sps.TimeOffsetLength != 12 {
		t.Fatalf("hrd lengths = init %d cpb %d dpb %d time %d", sps.InitialCPBRemovalDelayLength, sps.CPBRemovalDelayLength, sps.DPBOutputDelayLength, sps.TimeOffsetLength)
	}
	if sps.PicStructPresentFlag != 1 {
		t.Fatalf("pic_struct_present = %d", sps.PicStructPresentFlag)
	}
}

func TestDecodeSPSDerivesReorderFramesWithoutRestriction(t *testing.T) {
	sps, err := DecodeSPS(buildSPSRBSP(t, 66, 30, 1, false, nil))
	if err != nil {
		t.Fatal(err)
	}
	if sps.BitstreamRestrictionFlag != 0 || sps.NumReorderFrames != h264MaxDPBFrames-1 {
		t.Fatalf("derived reorder = flag %d reorder %d", sps.BitstreamRestrictionFlag, sps.NumReorderFrames)
	}
}

func TestDecodeSPSRejectsInvalidHRDCPBCount(t *testing.T) {
	rbsp := buildSPSRBSP(t, 66, 30, 1, true, func(b *spsBitBuilder) {
		writeMinimalCommonVUI(b)
		b.writeBit(0) // timing_info_present_flag
		b.writeBit(1) // nal_hrd_parameters_present_flag
		b.writeUE(32) // cpb_cnt_minus1 => 33
	})
	if _, err := DecodeSPS(rbsp); err != ErrInvalidData {
		t.Fatalf("err = %v, want ErrInvalidData", err)
	}
}

func TestDecodeSPSRejectsInvalidNumReorderFrames(t *testing.T) {
	rbsp := buildSPSRBSP(t, 66, 30, 1, true, func(b *spsBitBuilder) {
		writeMinimalCommonVUI(b)
		b.writeBit(0) // timing_info_present_flag
		b.writeBit(0) // nal_hrd_parameters_present_flag
		b.writeBit(0) // vcl_hrd_parameters_present_flag
		b.writeBit(0) // pic_struct_present_flag
		b.writeBit(1) // bitstream_restriction_flag
		b.writeBit(1)
		b.writeUE(0)
		b.writeUE(0)
		b.writeUE(0)
		b.writeUE(0)
		b.writeUE(17)
		b.writeUE(17)
	})
	if _, err := DecodeSPS(rbsp); err != ErrInvalidData {
		t.Fatalf("err = %v, want ErrInvalidData", err)
	}
}

func buildSPSRBSP(t *testing.T, profileIDC uint32, levelIDC uint32, refFrames uint32, vui bool, writeVUI func(*spsBitBuilder)) []byte {
	t.Helper()
	var b spsBitBuilder
	b.writeBits(profileIDC, 8)
	b.writeBits(0, 6)
	b.writeBits(0, 2)
	b.writeBits(levelIDC, 8)
	b.writeUE(0) // seq_parameter_set_id
	b.writeUE(0) // log2_max_frame_num_minus4
	b.writeUE(0) // pic_order_cnt_type
	b.writeUE(0) // log2_max_pic_order_cnt_lsb_minus4
	b.writeUE(refFrames)
	b.writeBit(0) // gaps_in_frame_num_value_allowed_flag
	b.writeUE(0)  // pic_width_in_mbs_minus1
	b.writeUE(0)  // pic_height_in_map_units_minus1
	b.writeBit(1) // frame_mbs_only_flag
	b.writeBit(1) // direct_8x8_inference_flag
	b.writeBit(0) // frame_cropping_flag
	if vui {
		b.writeBit(1)
		if writeVUI != nil {
			writeVUI(&b)
		}
	} else {
		b.writeBit(0)
	}
	return b.rbsp()
}

func writeMinimalCommonVUI(b *spsBitBuilder) {
	b.writeBit(0) // aspect_ratio_info_present_flag
	b.writeBit(0) // overscan_info_present_flag
	b.writeBit(0) // video_signal_type_present_flag
	b.writeBit(0) // chroma_loc_info_present_flag
}

type spsBitBuilder struct {
	bits []byte
}

func (b *spsBitBuilder) writeBit(v uint32) {
	if v&1 != 0 {
		b.bits = append(b.bits, 1)
	} else {
		b.bits = append(b.bits, 0)
	}
}

func (b *spsBitBuilder) writeBits(v uint32, n uint32) {
	for i := int(n) - 1; i >= 0; i-- {
		b.writeBit(v >> uint(i))
	}
}

func (b *spsBitBuilder) writeUE(v uint32) {
	codeNum := v + 1
	width := bits.Len32(codeNum)
	for i := 0; i < width-1; i++ {
		b.writeBit(0)
	}
	b.writeBits(codeNum, uint32(width))
}

func (b *spsBitBuilder) rbsp() []byte {
	b.writeBit(1)
	for len(b.bits)&7 != 0 {
		b.writeBit(0)
	}
	out := make([]byte, len(b.bits)/8)
	for i, bit := range b.bits {
		if bit != 0 {
			out[i>>3] |= 1 << uint(7-(i&7))
		}
	}
	return out
}
