// SPDX-License-Identifier: LGPL-2.1-or-later
//
// Source-shaped writer subset for the first H.264 realtime encoder picture
// path. Slice-header syntax order follows FFmpeg n8.0.1
// libavcodec/cbs_h264_syntax_template.c slice_header()/dec_ref_pic_marking().
// The payload is deliberately limited to Baseline CAVLC I_PCM macroblocks so
// the first IDR path is exact and oracle-friendly before quantized coding lands.

package h264

type EncoderI420IntraPCMIDRConfig struct {
	Width    int
	Height   int
	StrideY  int
	StrideCb int
	StrideCr int
	Y        []byte
	Cb       []byte
	Cr       []byte

	FrameNum                   uint32
	IDRPicID                   uint32
	InitialQP                  int
	DisableDeblockingFilterIDC uint32
	FirstMBAddr                uint32
	MacroblockCount            uint32
	NALLengthSize              int
}

type EncoderI420IntraPCMPConfig struct {
	Width    int
	Height   int
	StrideY  int
	StrideCb int
	StrideCr int
	Y        []byte
	Cb       []byte
	Cr       []byte

	FrameNum                   uint32
	InitialQP                  int
	DisableDeblockingFilterIDC uint32
	FirstMBAddr                uint32
	MacroblockCount            uint32
	NALLengthSize              int
}

type EncoderI420PSkipConfig struct {
	Width  int
	Height int

	FrameNum                   uint32
	InitialQP                  int
	DisableDeblockingFilterIDC uint32
	FirstMBAddr                uint32
	MacroblockCount            uint32
	NALLengthSize              int
}

type EncoderIDRSlice struct {
	RBSP   []byte
	NAL    []byte
	AnnexB []byte
	AVC    []byte
}

type EncoderPSkipSlice struct {
	RBSP   []byte
	NAL    []byte
	AnnexB []byte
	AVC    []byte
}

type EncoderPSlice struct {
	RBSP   []byte
	NAL    []byte
	AnnexB []byte
	AVC    []byte
}

func BuildEncoderI420IntraPCMIDRSlice(cfg EncoderI420IntraPCMIDRConfig) (EncoderIDRSlice, error) {
	if cfg.NALLengthSize == 0 {
		cfg.NALLengthSize = 4
	}
	rbsp, err := EncodeI420IntraPCMIDRSliceRBSP(cfg)
	if err != nil {
		return EncoderIDRSlice{}, err
	}
	nal, err := AppendNAL(nil, 3, NALIDRSlice, rbsp)
	if err != nil {
		return EncoderIDRSlice{}, err
	}
	annexB, err := AppendAnnexBNAL(nil, 3, NALIDRSlice, rbsp)
	if err != nil {
		return EncoderIDRSlice{}, err
	}
	avc, err := AppendAVCNAL(nil, cfg.NALLengthSize, 3, NALIDRSlice, rbsp)
	if err != nil {
		return EncoderIDRSlice{}, err
	}
	return EncoderIDRSlice{
		RBSP:   rbsp,
		NAL:    nal,
		AnnexB: annexB,
		AVC:    avc,
	}, nil
}

func BuildEncoderI420IntraPCMPSlice(cfg EncoderI420IntraPCMPConfig) (EncoderPSlice, error) {
	if cfg.NALLengthSize == 0 {
		cfg.NALLengthSize = 4
	}
	rbsp, err := EncodeI420IntraPCMPSliceRBSP(cfg)
	if err != nil {
		return EncoderPSlice{}, err
	}
	nal, err := AppendNAL(nil, 2, NALSlice, rbsp)
	if err != nil {
		return EncoderPSlice{}, err
	}
	annexB, err := AppendAnnexBNAL(nil, 2, NALSlice, rbsp)
	if err != nil {
		return EncoderPSlice{}, err
	}
	avc, err := AppendAVCNAL(nil, cfg.NALLengthSize, 2, NALSlice, rbsp)
	if err != nil {
		return EncoderPSlice{}, err
	}
	return EncoderPSlice{
		RBSP:   rbsp,
		NAL:    nal,
		AnnexB: annexB,
		AVC:    avc,
	}, nil
}

func BuildEncoderI420PSkipSlice(cfg EncoderI420PSkipConfig) (EncoderPSkipSlice, error) {
	if cfg.NALLengthSize == 0 {
		cfg.NALLengthSize = 4
	}
	rbsp, err := EncodeI420PSkipSliceRBSP(cfg)
	if err != nil {
		return EncoderPSkipSlice{}, err
	}
	nal, err := AppendNAL(nil, 2, NALSlice, rbsp)
	if err != nil {
		return EncoderPSkipSlice{}, err
	}
	annexB, err := AppendAnnexBNAL(nil, 2, NALSlice, rbsp)
	if err != nil {
		return EncoderPSkipSlice{}, err
	}
	avc, err := AppendAVCNAL(nil, cfg.NALLengthSize, 2, NALSlice, rbsp)
	if err != nil {
		return EncoderPSkipSlice{}, err
	}
	return EncoderPSkipSlice{
		RBSP:   rbsp,
		NAL:    nal,
		AnnexB: annexB,
		AVC:    avc,
	}, nil
}

func EncodeI420IntraPCMIDRSliceRBSP(cfg EncoderI420IntraPCMIDRConfig) ([]byte, error) {
	if err := validateEncoderI420IntraPCMIDRConfig(cfg); err != nil {
		return nil, err
	}

	var bw BitWriter
	if err := writeEncoderI420IDRSliceHeader(&bw, cfg); err != nil {
		return nil, err
	}

	mbWidth := (cfg.Width + 15) >> 4
	firstMB, macroblockCount := encoderI420SliceRange(cfg.Width, cfg.Height, cfg.FirstMBAddr, cfg.MacroblockCount)
	lastMB := firstMB + macroblockCount
	samples := encoderI420IntraPCMSamples{
		Width:    cfg.Width,
		Height:   cfg.Height,
		StrideY:  cfg.StrideY,
		StrideCb: cfg.StrideCb,
		StrideCr: cfg.StrideCr,
		Y:        cfg.Y,
		Cb:       cfg.Cb,
		Cr:       cfg.Cr,
	}
	for mbAddr := firstMB; mbAddr < lastMB; mbAddr++ {
		mbX := mbAddr % mbWidth
		mbY := mbAddr / mbWidth
		if err := bw.WriteUEGolomb(25); err != nil { // I_PCM
			return nil, err
		}
		bw.WriteZeroAlign()
		if err := writeEncoderI420IntraPCMMacroblock(&bw, samples, mbX, mbY); err != nil {
			return nil, err
		}
	}

	bw.WriteRBSPTrailingBits()
	return bw.Bytes(), nil
}

func EncodeI420IntraPCMPSliceRBSP(cfg EncoderI420IntraPCMPConfig) ([]byte, error) {
	if err := validateEncoderI420IntraPCMPConfig(cfg); err != nil {
		return nil, err
	}

	var bw BitWriter
	if err := writeEncoderI420PSliceHeader(&bw, EncoderI420PSkipConfig{
		Width:                      cfg.Width,
		Height:                     cfg.Height,
		FrameNum:                   cfg.FrameNum,
		InitialQP:                  cfg.InitialQP,
		DisableDeblockingFilterIDC: cfg.DisableDeblockingFilterIDC,
		FirstMBAddr:                cfg.FirstMBAddr,
		MacroblockCount:            cfg.MacroblockCount,
	}); err != nil {
		return nil, err
	}

	mbWidth := (cfg.Width + 15) >> 4
	firstMB, macroblockCount := encoderI420SliceRange(cfg.Width, cfg.Height, cfg.FirstMBAddr, cfg.MacroblockCount)
	lastMB := firstMB + macroblockCount
	samples := encoderI420IntraPCMSamples{
		Width:    cfg.Width,
		Height:   cfg.Height,
		StrideY:  cfg.StrideY,
		StrideCb: cfg.StrideCb,
		StrideCr: cfg.StrideCr,
		Y:        cfg.Y,
		Cb:       cfg.Cb,
		Cr:       cfg.Cr,
	}
	for mbAddr := firstMB; mbAddr < lastMB; mbAddr++ {
		mbX := mbAddr % mbWidth
		mbY := mbAddr / mbWidth
		if err := bw.WriteUEGolomb(0); err != nil { // mb_skip_run
			return nil, err
		}
		if err := bw.WriteUEGolomb(30); err != nil { // P-slice I_PCM = 5 P types + 25 I_PCM index
			return nil, err
		}
		bw.WriteZeroAlign()
		if err := writeEncoderI420IntraPCMMacroblock(&bw, samples, mbX, mbY); err != nil {
			return nil, err
		}
	}

	bw.WriteRBSPTrailingBits()
	return bw.Bytes(), nil
}

func EncodeI420PSkipSliceRBSP(cfg EncoderI420PSkipConfig) ([]byte, error) {
	if err := validateEncoderI420PSkipConfig(cfg); err != nil {
		return nil, err
	}

	var bw BitWriter
	if err := writeEncoderI420PSliceHeader(&bw, cfg); err != nil {
		return nil, err
	}

	_, macroblockCount := encoderI420SliceRange(cfg.Width, cfg.Height, cfg.FirstMBAddr, cfg.MacroblockCount)
	if err := bw.WriteUEGolomb(uint32(macroblockCount)); err != nil {
		return nil, err
	}
	bw.WriteRBSPTrailingBits()
	return bw.Bytes(), nil
}

func writeEncoderI420IDRSliceHeader(bw *BitWriter, cfg EncoderI420IntraPCMIDRConfig) error {
	if err := bw.WriteUEGolomb(cfg.FirstMBAddr); err != nil { // first_mb_in_slice
		return err
	}
	if err := bw.WriteUEGolomb(2); err != nil { // slice_type I
		return err
	}
	if err := bw.WriteUEGolomb(0); err != nil { // pic_parameter_set_id
		return err
	}
	if err := bw.WriteBits(cfg.FrameNum, 8); err != nil {
		return err
	}
	if err := bw.WriteUEGolomb(cfg.IDRPicID); err != nil {
		return err
	}
	bw.WriteBit(0)                              // no_output_of_prior_pics_flag
	bw.WriteBit(0)                              // long_term_reference_flag
	if err := bw.WriteSEGolomb(0); err != nil { // slice_qp_delta
		return err
	}
	if err := bw.WriteUEGolomb(cfg.DisableDeblockingFilterIDC); err != nil {
		return err
	}
	if cfg.DisableDeblockingFilterIDC != 1 {
		if err := bw.WriteSEGolomb(0); err != nil { // slice_alpha_c0_offset_div2
			return err
		}
		if err := bw.WriteSEGolomb(0); err != nil { // slice_beta_offset_div2
			return err
		}
	}
	return nil
}

func writeEncoderI420PSliceHeader(bw *BitWriter, cfg EncoderI420PSkipConfig) error {
	if err := bw.WriteUEGolomb(cfg.FirstMBAddr); err != nil { // first_mb_in_slice
		return err
	}
	if err := bw.WriteUEGolomb(0); err != nil { // slice_type P
		return err
	}
	if err := bw.WriteUEGolomb(0); err != nil { // pic_parameter_set_id
		return err
	}
	if err := bw.WriteBits(cfg.FrameNum, 8); err != nil {
		return err
	}
	bw.WriteBit(0)                              // num_ref_idx_active_override_flag
	bw.WriteBit(0)                              // ref_pic_list_modification_flag_l0
	bw.WriteBit(0)                              // adaptive_ref_pic_marking_mode_flag
	if err := bw.WriteSEGolomb(0); err != nil { // slice_qp_delta
		return err
	}
	if err := bw.WriteUEGolomb(cfg.DisableDeblockingFilterIDC); err != nil {
		return err
	}
	if cfg.DisableDeblockingFilterIDC != 1 {
		if err := bw.WriteSEGolomb(0); err != nil { // slice_alpha_c0_offset_div2
			return err
		}
		if err := bw.WriteSEGolomb(0); err != nil { // slice_beta_offset_div2
			return err
		}
	}
	return nil
}

type encoderI420IntraPCMSamples struct {
	Width    int
	Height   int
	StrideY  int
	StrideCb int
	StrideCr int
	Y        []byte
	Cb       []byte
	Cr       []byte
}

func writeEncoderI420IntraPCMMacroblock(bw *BitWriter, cfg encoderI420IntraPCMSamples, mbX int, mbY int) error {
	var pcm [384]byte
	i := 0
	baseX := mbX << 4
	baseY := mbY << 4
	for y := 0; y < 16; y++ {
		srcY := clampEncoderCoord(baseY+y, cfg.Height)
		for x := 0; x < 16; x++ {
			srcX := clampEncoderCoord(baseX+x, cfg.Width)
			pcm[i] = cfg.Y[srcY*cfg.StrideY+srcX]
			i++
		}
	}

	chromaWidth := cfg.Width >> 1
	chromaHeight := cfg.Height >> 1
	baseCX := mbX << 3
	baseCY := mbY << 3
	for y := 0; y < 8; y++ {
		srcY := clampEncoderCoord(baseCY+y, chromaHeight)
		for x := 0; x < 8; x++ {
			srcX := clampEncoderCoord(baseCX+x, chromaWidth)
			pcm[i] = cfg.Cb[srcY*cfg.StrideCb+srcX]
			i++
		}
	}
	for y := 0; y < 8; y++ {
		srcY := clampEncoderCoord(baseCY+y, chromaHeight)
		for x := 0; x < 8; x++ {
			srcX := clampEncoderCoord(baseCX+x, chromaWidth)
			pcm[i] = cfg.Cr[srcY*cfg.StrideCr+srcX]
			i++
		}
	}
	return bw.WriteAlignedBytes(pcm[:])
}

func validateEncoderI420IntraPCMIDRConfig(cfg EncoderI420IntraPCMIDRConfig) error {
	if err := validateEncoderI420IntraPCMSamples(encoderI420IntraPCMSamples{
		Width:    cfg.Width,
		Height:   cfg.Height,
		StrideY:  cfg.StrideY,
		StrideCb: cfg.StrideCb,
		StrideCr: cfg.StrideCr,
		Y:        cfg.Y,
		Cb:       cfg.Cb,
		Cr:       cfg.Cr,
	}); err != nil {
		return ErrInvalidData
	}
	if cfg.FrameNum >= 1<<8 || cfg.IDRPicID > 65535 ||
		cfg.InitialQP < 0 || cfg.InitialQP > 51 ||
		cfg.DisableDeblockingFilterIDC > 2 ||
		cfg.NALLengthSize < 0 || cfg.NALLengthSize > 4 {
		return ErrInvalidData
	}
	if err := validateEncoderI420SliceRange(cfg.Width, cfg.Height, cfg.FirstMBAddr, cfg.MacroblockCount); err != nil {
		return err
	}
	return nil
}

func validateEncoderI420IntraPCMPConfig(cfg EncoderI420IntraPCMPConfig) error {
	if err := validateEncoderI420IntraPCMSamples(encoderI420IntraPCMSamples{
		Width:    cfg.Width,
		Height:   cfg.Height,
		StrideY:  cfg.StrideY,
		StrideCb: cfg.StrideCb,
		StrideCr: cfg.StrideCr,
		Y:        cfg.Y,
		Cb:       cfg.Cb,
		Cr:       cfg.Cr,
	}); err != nil {
		return ErrInvalidData
	}
	if cfg.FrameNum >= 1<<8 ||
		cfg.InitialQP < 0 || cfg.InitialQP > 51 ||
		cfg.DisableDeblockingFilterIDC > 2 ||
		cfg.NALLengthSize < 0 || cfg.NALLengthSize > 4 {
		return ErrInvalidData
	}
	if err := validateEncoderI420SliceRange(cfg.Width, cfg.Height, cfg.FirstMBAddr, cfg.MacroblockCount); err != nil {
		return err
	}
	return nil
}

func validateEncoderI420IntraPCMSamples(cfg encoderI420IntraPCMSamples) error {
	if cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width&1 != 0 || cfg.Height&1 != 0 {
		return ErrInvalidData
	}
	if cfg.StrideY < cfg.Width || cfg.StrideCb < cfg.Width/2 || cfg.StrideCr < cfg.Width/2 {
		return ErrInvalidData
	}
	if len(cfg.Y) < cfg.StrideY*cfg.Height {
		return ErrInvalidData
	}
	chromaHeight := cfg.Height >> 1
	if len(cfg.Cb) < cfg.StrideCb*chromaHeight || len(cfg.Cr) < cfg.StrideCr*chromaHeight {
		return ErrInvalidData
	}
	return nil
}

func validateEncoderI420PSkipConfig(cfg EncoderI420PSkipConfig) error {
	if cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width&1 != 0 || cfg.Height&1 != 0 {
		return ErrInvalidData
	}
	if cfg.FrameNum >= 1<<8 ||
		cfg.InitialQP < 0 || cfg.InitialQP > 51 ||
		cfg.DisableDeblockingFilterIDC > 2 ||
		cfg.NALLengthSize < 0 || cfg.NALLengthSize > 4 {
		return ErrInvalidData
	}
	if err := validateEncoderI420SliceRange(cfg.Width, cfg.Height, cfg.FirstMBAddr, cfg.MacroblockCount); err != nil {
		return err
	}
	return nil
}

func validateEncoderI420SliceRange(width int, height int, firstMBAddr uint32, macroblockCount uint32) error {
	total := encoderI420MacroblockCount(width, height)
	if total <= 0 {
		return ErrInvalidData
	}
	if uint64(firstMBAddr) >= uint64(total) {
		return ErrInvalidData
	}
	count := macroblockCount
	if count == 0 {
		count = uint32(total) - firstMBAddr
	}
	if count == 0 || uint64(firstMBAddr)+uint64(count) > uint64(total) {
		return ErrInvalidData
	}
	return nil
}

func encoderI420SliceRange(width int, height int, firstMBAddr uint32, macroblockCount uint32) (int, int) {
	total := encoderI420MacroblockCount(width, height)
	first := int(firstMBAddr)
	count := int(macroblockCount)
	if count == 0 {
		count = total - first
	}
	return first, count
}

func encoderI420MacroblockCount(width int, height int) int {
	return ((width + 15) >> 4) * ((height + 15) >> 4)
}

func clampEncoderCoord(v int, limit int) int {
	if v >= limit {
		return limit - 1
	}
	return v
}
