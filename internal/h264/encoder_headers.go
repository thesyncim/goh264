// SPDX-License-Identifier: LGPL-2.1-or-later
//
// Source-shaped writer subset for H.264 realtime encoder parameter sets.
// Syntax order follows FFmpeg n8.0.1 libavcodec/cbs_h264_syntax_template.c
// sps()/pps(); baseline field defaults follow hw_base_encode_h264.c.

package h264

type EncoderParameterSetConfig struct {
	ProfileIDC         uint8
	ConstraintSetFlags uint8
	LevelIDC           uint8
	SPSID              uint32
	PPSID              uint32

	Width              int
	Height             int
	CropLeft           int
	CropRight          int
	CropTop            int
	CropBottom         int
	FrameRateNum       int
	FrameRateDen       int
	MaxReferenceFrames uint32
	InitialQP          int

	SARNum                         int32
	SARDen                         int32
	VideoFormat                    int32
	FullRange                      bool
	ColorPrimaries                 int32
	ColorTransfer                  int32
	ColorMatrix                    int32
	ChromaSampleLocTypeTopField    int32
	ChromaSampleLocTypeBottomField int32

	NALLengthSize int
}

type EncoderParameterSets struct {
	SPS                           []byte
	PPS                           []byte
	AnnexB                        []byte
	AVCDecoderConfigurationRecord []byte
}

func BuildEncoderParameterSets(cfg EncoderParameterSetConfig) (EncoderParameterSets, error) {
	if cfg.NALLengthSize == 0 {
		cfg.NALLengthSize = 4
	}
	spsRBSP, err := EncodeBaselineSPSRBSP(cfg)
	if err != nil {
		return EncoderParameterSets{}, err
	}
	ppsRBSP, err := EncodeBaselinePPSRBSP(cfg)
	if err != nil {
		return EncoderParameterSets{}, err
	}
	sps, err := AppendNAL(nil, 3, NALSPS, spsRBSP)
	if err != nil {
		return EncoderParameterSets{}, err
	}
	pps, err := AppendNAL(nil, 3, NALPPS, ppsRBSP)
	if err != nil {
		return EncoderParameterSets{}, err
	}
	annexB, err := AppendAnnexBNAL(nil, 3, NALSPS, spsRBSP)
	if err != nil {
		return EncoderParameterSets{}, err
	}
	annexB, err = AppendAnnexBNAL(annexB, 3, NALPPS, ppsRBSP)
	if err != nil {
		return EncoderParameterSets{}, err
	}
	avcc, err := AppendAVCDecoderConfigurationRecord(nil, cfg.ProfileIDC, encoderConstraintCompatibilityByte(cfg.ConstraintSetFlags), cfg.LevelIDC, cfg.NALLengthSize, [][]byte{sps}, [][]byte{pps})
	if err != nil {
		return EncoderParameterSets{}, err
	}
	return EncoderParameterSets{
		SPS:                           sps,
		PPS:                           pps,
		AnnexB:                        annexB,
		AVCDecoderConfigurationRecord: avcc,
	}, nil
}

func EncodeBaselineSPSRBSP(cfg EncoderParameterSetConfig) ([]byte, error) {
	if err := validateEncoderParameterSetConfig(cfg); err != nil {
		return nil, err
	}
	mbWidth := (cfg.Width + 15) >> 4
	mbHeight := (cfg.Height + 15) >> 4
	cropLeft := cfg.CropLeft
	cropRight := cfg.CropRight + (mbWidth*16 - cfg.Width)
	cropTop := cfg.CropTop
	cropBottom := cfg.CropBottom + (mbHeight*16 - cfg.Height)

	var bw BitWriter
	if err := bw.WriteBits(uint32(cfg.ProfileIDC), 8); err != nil {
		return nil, err
	}
	for i := 0; i < 6; i++ {
		bw.WriteBit(uint32(cfg.ConstraintSetFlags>>uint(i)) & 1)
	}
	if err := bw.WriteBits(0, 2); err != nil {
		return nil, err
	}
	if err := bw.WriteBits(uint32(cfg.LevelIDC), 8); err != nil {
		return nil, err
	}
	if err := bw.WriteUEGolomb(cfg.SPSID); err != nil {
		return nil, err
	}

	if err := bw.WriteUEGolomb(4); err != nil { // log2_max_frame_num_minus4
		return nil, err
	}
	if err := bw.WriteUEGolomb(2); err != nil { // pic_order_cnt_type
		return nil, err
	}
	if err := bw.WriteUEGolomb(cfg.MaxReferenceFrames); err != nil {
		return nil, err
	}
	bw.WriteBit(0) // gaps_in_frame_num_value_allowed_flag
	if err := bw.WriteUEGolomb(uint32(mbWidth - 1)); err != nil {
		return nil, err
	}
	if err := bw.WriteUEGolomb(uint32(mbHeight - 1)); err != nil {
		return nil, err
	}
	bw.WriteBit(1) // frame_mbs_only_flag
	bw.WriteBit(1) // direct_8x8_inference_flag
	if cropLeft != 0 || cropRight != 0 || cropTop != 0 || cropBottom != 0 {
		bw.WriteBit(1)
		for _, v := range []uint32{
			uint32(cropLeft >> 1),
			uint32(cropRight >> 1),
			uint32(cropTop >> 1),
			uint32(cropBottom >> 1),
		} {
			if err := bw.WriteUEGolomb(v); err != nil {
				return nil, err
			}
		}
	} else {
		bw.WriteBit(0)
	}

	bw.WriteBit(1) // vui_parameters_present_flag
	if err := writeEncoderVUI(&bw, cfg); err != nil {
		return nil, err
	}
	bw.WriteRBSPTrailingBits()
	return bw.Bytes(), nil
}

func EncodeBaselinePPSRBSP(cfg EncoderParameterSetConfig) ([]byte, error) {
	if err := validateEncoderParameterSetConfig(cfg); err != nil {
		return nil, err
	}
	var bw BitWriter
	for _, v := range []uint32{cfg.PPSID, cfg.SPSID} {
		if err := bw.WriteUEGolomb(v); err != nil {
			return nil, err
		}
	}
	bw.WriteBit(0)                              // entropy_coding_mode_flag
	bw.WriteBit(0)                              // bottom_field_pic_order_in_frame_present_flag
	if err := bw.WriteUEGolomb(0); err != nil { // num_slice_groups_minus1
		return nil, err
	}
	if err := bw.WriteUEGolomb(0); err != nil { // num_ref_idx_l0_default_active_minus1
		return nil, err
	}
	if err := bw.WriteUEGolomb(0); err != nil { // num_ref_idx_l1_default_active_minus1
		return nil, err
	}
	bw.WriteBit(0) // weighted_pred_flag
	if err := bw.WriteBits(0, 2); err != nil {
		return nil, err
	}
	if err := bw.WriteSEGolomb(int32(cfg.InitialQP - 26)); err != nil {
		return nil, err
	}
	if err := bw.WriteSEGolomb(0); err != nil { // pic_init_qs_minus26
		return nil, err
	}
	if err := bw.WriteSEGolomb(0); err != nil { // chroma_qp_index_offset
		return nil, err
	}
	bw.WriteBit(1) // deblocking_filter_control_present_flag
	bw.WriteBit(0) // constrained_intra_pred_flag
	bw.WriteBit(0) // redundant_pic_cnt_present_flag
	bw.WriteRBSPTrailingBits()
	return bw.Bytes(), nil
}

func writeEncoderVUI(bw *BitWriter, cfg EncoderParameterSetConfig) error {
	if cfg.SARNum != 0 || cfg.SARDen != 0 {
		bw.WriteBit(1)
		if err := bw.WriteBits(255, 8); err != nil {
			return err
		}
		if err := bw.WriteBits(uint32(cfg.SARNum), 16); err != nil {
			return err
		}
		if err := bw.WriteBits(uint32(cfg.SARDen), 16); err != nil {
			return err
		}
	} else {
		bw.WriteBit(0)
	}
	bw.WriteBit(0) // overscan_info_present_flag

	videoSignalPresent := cfg.FullRange || cfg.VideoFormat != 0 || cfg.ColorPrimaries != 0 || cfg.ColorTransfer != 0 || cfg.ColorMatrix != 0
	if videoSignalPresent {
		bw.WriteBit(1)
		videoFormat := uint32(5)
		if cfg.VideoFormat != 0 {
			videoFormat = uint32(cfg.VideoFormat)
		}
		if err := bw.WriteBits(videoFormat, 3); err != nil {
			return err
		}
		if cfg.FullRange {
			bw.WriteBit(1)
		} else {
			bw.WriteBit(0)
		}
		colorDescriptionPresent := cfg.ColorPrimaries != 0 || cfg.ColorTransfer != 0 || cfg.ColorMatrix != 0
		if colorDescriptionPresent {
			bw.WriteBit(1)
			for _, v := range []int32{cfg.ColorPrimaries, cfg.ColorTransfer, cfg.ColorMatrix} {
				if v == 0 {
					v = 2
				}
				if err := bw.WriteBits(uint32(v), 8); err != nil {
					return err
				}
			}
		} else {
			bw.WriteBit(0)
		}
	} else {
		bw.WriteBit(0)
	}

	if cfg.ChromaSampleLocTypeTopField != 0 || cfg.ChromaSampleLocTypeBottomField != 0 {
		bw.WriteBit(1)
		if err := bw.WriteUEGolomb(uint32(cfg.ChromaSampleLocTypeTopField)); err != nil {
			return err
		}
		if err := bw.WriteUEGolomb(uint32(cfg.ChromaSampleLocTypeBottomField)); err != nil {
			return err
		}
	} else {
		bw.WriteBit(0)
	}

	bw.WriteBit(1) // timing_info_present_flag
	if err := bw.WriteBits(uint32(cfg.FrameRateDen), 32); err != nil {
		return err
	}
	if err := bw.WriteBits(uint32(2*cfg.FrameRateNum), 32); err != nil {
		return err
	}
	bw.WriteBit(1) // fixed_frame_rate_flag

	bw.WriteBit(0) // nal_hrd_parameters_present_flag
	bw.WriteBit(0) // vcl_hrd_parameters_present_flag
	bw.WriteBit(0) // pic_struct_present_flag

	bw.WriteBit(1)                              // bitstream_restriction_flag
	bw.WriteBit(1)                              // motion_vectors_over_pic_boundaries_flag
	if err := bw.WriteUEGolomb(0); err != nil { // max_bytes_per_pic_denom
		return err
	}
	if err := bw.WriteUEGolomb(0); err != nil { // max_bits_per_mb_denom
		return err
	}
	if err := bw.WriteUEGolomb(15); err != nil {
		return err
	}
	if err := bw.WriteUEGolomb(15); err != nil {
		return err
	}
	if err := bw.WriteUEGolomb(0); err != nil { // max_num_reorder_frames
		return err
	}
	return bw.WriteUEGolomb(cfg.MaxReferenceFrames)
}

func validateEncoderParameterSetConfig(cfg EncoderParameterSetConfig) error {
	if cfg.ProfileIDC == 0 || cfg.LevelIDC == 0 || cfg.SPSID >= maxSPSCount || cfg.PPSID >= maxPPSCount {
		return ErrInvalidData
	}
	if cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width&1 != 0 || cfg.Height&1 != 0 {
		return ErrInvalidData
	}
	if cfg.CropLeft < 0 || cfg.CropRight < 0 || cfg.CropTop < 0 || cfg.CropBottom < 0 ||
		cfg.CropLeft&1 != 0 || cfg.CropRight&1 != 0 || cfg.CropTop&1 != 0 || cfg.CropBottom&1 != 0 ||
		cfg.CropLeft+cfg.CropRight >= cfg.Width || cfg.CropTop+cfg.CropBottom >= cfg.Height {
		return ErrInvalidData
	}
	if cfg.FrameRateNum <= 0 || cfg.FrameRateDen <= 0 || cfg.FrameRateDen > int(^uint32(0)) || cfg.FrameRateNum > int(^uint32(0)>>1) {
		return ErrInvalidData
	}
	if cfg.MaxReferenceFrames > h264MaxDPBFrames || cfg.InitialQP < 0 || cfg.InitialQP > 51 {
		return ErrInvalidData
	}
	if cfg.SARNum < 0 || cfg.SARDen < 0 || cfg.SARNum > 0xffff || cfg.SARDen > 0xffff || (cfg.SARNum == 0) != (cfg.SARDen == 0) {
		return ErrInvalidData
	}
	if cfg.VideoFormat < 0 || cfg.VideoFormat > 7 || cfg.ColorPrimaries < 0 || cfg.ColorPrimaries > 255 || cfg.ColorTransfer < 0 || cfg.ColorTransfer > 255 || cfg.ColorMatrix < 0 || cfg.ColorMatrix > 255 {
		return ErrInvalidData
	}
	if cfg.ChromaSampleLocTypeTopField < 0 || cfg.ChromaSampleLocTypeTopField > 5 || cfg.ChromaSampleLocTypeBottomField < 0 || cfg.ChromaSampleLocTypeBottomField > 5 {
		return ErrInvalidData
	}
	if cfg.NALLengthSize < 0 || cfg.NALLengthSize > 4 {
		return ErrInvalidData
	}
	return nil
}

func encoderConstraintCompatibilityByte(flags uint8) uint8 {
	var out uint8
	for i := 0; i < 6; i++ {
		if flags&(1<<uint(i)) != 0 {
			out |= 1 << uint(7-i)
		}
	}
	return out
}
