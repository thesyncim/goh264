// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

import "testing"

func TestDecodeSPSRejectsChromaFormatBeyond444LikeFFmpeg(t *testing.T) {
	rbsp := buildHighProfileSPSBoundaryRBSP(t, highProfileSPSBoundaryConfig{
		chromaFormatIDC: 4,
	})
	if _, err := DecodeSPS(rbsp); err != ErrUnsupported {
		t.Fatalf("err = %v, want ErrUnsupported", err)
	}
}

func TestDecodeSPSRejectsSeparateColorPlanesLikeFFmpeg(t *testing.T) {
	rbsp := buildHighProfileSPSBoundaryRBSP(t, highProfileSPSBoundaryConfig{
		chromaFormatIDC:         3,
		separateColorPlanesFlag: 1,
		bitDepthLumaMinus8:      2,
		bitDepthChromaMinus8:    2,
	})
	if _, err := DecodeSPS(rbsp); err != ErrUnsupported {
		t.Fatalf("err = %v, want ErrUnsupported", err)
	}
}

func TestDecodeSPSRejectsMixedBitDepthLikeFFmpeg(t *testing.T) {
	rbsp := buildHighProfileSPSBoundaryRBSP(t, highProfileSPSBoundaryConfig{
		chromaFormatIDC:      1,
		bitDepthLumaMinus8:   2,
		bitDepthChromaMinus8: 4,
	})
	if _, err := DecodeSPS(rbsp); err != ErrUnsupported {
		t.Fatalf("err = %v, want ErrUnsupported", err)
	}
}

func TestDecodeSPSRejectsCropOffsetOverflow(t *testing.T) {
	for _, tt := range []struct {
		name       string
		cropLeft   uint32
		cropRight  uint32
		cropTop    uint32
		cropBottom uint32
	}{
		{name: "horizontal sum wraps", cropLeft: ^uint32(0) - 1, cropRight: 2},
		{name: "vertical sum wraps", cropTop: ^uint32(0) - 1, cropBottom: 2},
	} {
		t.Run(tt.name, func(t *testing.T) {
			rbsp := buildBaselineSPSCropRBSP(t, tt.cropLeft, tt.cropRight, tt.cropTop, tt.cropBottom)
			if _, err := DecodeSPS(rbsp); err != ErrInvalidData {
				t.Fatalf("DecodeSPS crop overflow error = %v, want ErrInvalidData", err)
			}
		})
	}
}

type highProfileSPSBoundaryConfig struct {
	chromaFormatIDC         uint32
	separateColorPlanesFlag uint32
	bitDepthLumaMinus8      uint32
	bitDepthChromaMinus8    uint32
}

func buildHighProfileSPSBoundaryRBSP(t *testing.T, cfg highProfileSPSBoundaryConfig) []byte {
	t.Helper()
	var b spsBitBuilder
	b.writeBits(244, 8) // profile_idc: High 4:4:4 Predictive
	b.writeBits(0, 6)   // constraint_set flags
	b.writeBits(0, 2)   // reserved_zero_2bits
	b.writeBits(30, 8)  // level_idc
	b.writeUE(0)        // seq_parameter_set_id
	b.writeUE(cfg.chromaFormatIDC)
	if cfg.chromaFormatIDC == 3 {
		b.writeBit(cfg.separateColorPlanesFlag)
	}
	if cfg.chromaFormatIDC > 3 {
		return b.rbsp()
	}
	b.writeUE(cfg.bitDepthLumaMinus8)
	b.writeUE(cfg.bitDepthChromaMinus8)
	b.writeBit(0) // qpprime_y_zero_transform_bypass_flag
	b.writeBit(0) // seq_scaling_matrix_present_flag
	b.writeUE(0)  // log2_max_frame_num_minus4
	b.writeUE(0)  // pic_order_cnt_type
	b.writeUE(0)  // log2_max_pic_order_cnt_lsb_minus4
	b.writeUE(1)  // max_num_ref_frames
	b.writeBit(0) // gaps_in_frame_num_value_allowed_flag
	b.writeUE(0)  // pic_width_in_mbs_minus1
	b.writeUE(0)  // pic_height_in_map_units_minus1
	b.writeBit(1) // frame_mbs_only_flag
	b.writeBit(1) // direct_8x8_inference_flag
	b.writeBit(0) // frame_cropping_flag
	b.writeBit(0) // vui_parameters_present_flag
	return b.rbsp()
}

func buildBaselineSPSCropRBSP(t *testing.T, cropLeft, cropRight, cropTop, cropBottom uint32) []byte {
	t.Helper()
	var b spsBitBuilder
	b.writeBits(66, 8) // profile_idc: Baseline
	b.writeBits(0, 6)  // constraint_set flags
	b.writeBits(0, 2)  // reserved_zero_2bits
	b.writeBits(30, 8) // level_idc
	b.writeUE(0)       // seq_parameter_set_id
	b.writeUE(0)       // log2_max_frame_num_minus4
	b.writeUE(0)       // pic_order_cnt_type
	b.writeUE(0)       // log2_max_pic_order_cnt_lsb_minus4
	b.writeUE(1)       // max_num_ref_frames
	b.writeBit(0)      // gaps_in_frame_num_value_allowed_flag
	b.writeUE(0)       // pic_width_in_mbs_minus1
	b.writeUE(0)       // pic_height_in_map_units_minus1
	b.writeBit(1)      // frame_mbs_only_flag
	b.writeBit(1)      // direct_8x8_inference_flag
	b.writeBit(1)      // frame_cropping_flag
	b.writeUE(cropLeft)
	b.writeUE(cropRight)
	b.writeUE(cropTop)
	b.writeUE(cropBottom)
	b.writeBit(0) // vui_parameters_present_flag
	return b.rbsp()
}
