// SPDX-License-Identifier: LGPL-2.1-or-later
//
// Source-shaped writer subset for H.264 realtime encoder SEI messages.
// Syntax order follows FFmpeg n8.0.1 libavcodec/cbs_h264_syntax_template.c
// sei_recovery_point() and cbs_sei_syntax_template.c sei_message().

package h264

type EncoderRecoveryPointSEIConfig struct {
	RecoveryFrameCount    uint32
	ExactMatchFlag        bool
	BrokenLinkFlag        bool
	ChangingSliceGroupIDC uint8
	NALLengthSize         int
}

type EncoderSEIMessage struct {
	RBSP   []byte
	NAL    []byte
	AnnexB []byte
	AVC    []byte
}

func BuildEncoderRecoveryPointSEI(cfg EncoderRecoveryPointSEIConfig) (EncoderSEIMessage, error) {
	if cfg.NALLengthSize == 0 {
		cfg.NALLengthSize = 4
	}
	payload, err := EncodeRecoveryPointSEIPayload(cfg)
	if err != nil {
		return EncoderSEIMessage{}, err
	}
	rbsp := AppendSEIRBSP(nil, seiTypeRecoveryPoint, payload)
	nal, err := AppendNAL(nil, 0, NALSEI, rbsp)
	if err != nil {
		return EncoderSEIMessage{}, err
	}
	annexB, err := AppendAnnexBNAL(nil, 0, NALSEI, rbsp)
	if err != nil {
		return EncoderSEIMessage{}, err
	}
	avc, err := AppendAVCNAL(nil, cfg.NALLengthSize, 0, NALSEI, rbsp)
	if err != nil {
		return EncoderSEIMessage{}, err
	}
	return EncoderSEIMessage{
		RBSP:   rbsp,
		NAL:    nal,
		AnnexB: annexB,
		AVC:    avc,
	}, nil
}

// BuildEncoderRecoveryPointSEINAL returns only the raw recovery-point SEI NAL.
func BuildEncoderRecoveryPointSEINAL(cfg EncoderRecoveryPointSEIConfig) ([]byte, error) {
	payload, err := EncodeRecoveryPointSEIPayload(cfg)
	if err != nil {
		return nil, err
	}
	rbsp := AppendSEIRBSP(make([]byte, 0, 2+len(payload)+1), seiTypeRecoveryPoint, payload)
	dst, err := makeNALBuffer(rbsp)
	if err != nil {
		return nil, err
	}
	return AppendNAL(dst, 0, NALSEI, rbsp)
}

func EncodeRecoveryPointSEIPayload(cfg EncoderRecoveryPointSEIConfig) ([]byte, error) {
	if cfg.RecoveryFrameCount >= 1<<maxLog2MaxFrameNum || cfg.ChangingSliceGroupIDC > 2 {
		return nil, ErrInvalidData
	}
	if cfg.NALLengthSize < 0 || cfg.NALLengthSize > 4 {
		return nil, ErrInvalidData
	}
	var bw BitWriter
	if err := bw.WriteUEGolomb(cfg.RecoveryFrameCount); err != nil {
		return nil, err
	}
	if cfg.ExactMatchFlag {
		bw.WriteBit(1)
	} else {
		bw.WriteBit(0)
	}
	if cfg.BrokenLinkFlag {
		bw.WriteBit(1)
	} else {
		bw.WriteBit(0)
	}
	if err := bw.WriteBits(uint32(cfg.ChangingSliceGroupIDC), 2); err != nil {
		return nil, err
	}
	return bw.Bytes(), nil
}

func AppendSEIRBSP(dst []byte, payloadType uint32, payload []byte) []byte {
	if uint64(len(payload)) > uint64(^uint32(0)) {
		return nil
	}
	n, err := checkedAddInt(len(dst), encoderSEIHeaderValueSize(payloadType))
	if err != nil {
		return nil
	}
	n, err = checkedAddInt(n, encoderSEIHeaderValueSize(uint32(len(payload))))
	if err != nil {
		return nil
	}
	n, err = checkedAddInt(n, len(payload))
	if err != nil {
		return nil
	}
	if _, err := checkedAddInt(n, 1); err != nil {
		return nil
	}
	dst = appendEncoderSEIHeaderValue(dst, payloadType)
	dst = appendEncoderSEIHeaderValue(dst, uint32(len(payload)))
	dst = append(dst, payload...)
	return append(dst, 0x80)
}

func encoderSEIHeaderValueSize(value uint32) int {
	return int(value/255) + 1
}

func appendEncoderSEIHeaderValue(dst []byte, value uint32) []byte {
	for value >= 255 {
		dst = append(dst, 0xff)
		value -= 255
	}
	return append(dst, uint8(value))
}
