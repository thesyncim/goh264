// SPDX-License-Identifier: LGPL-2.1-or-later
//
// Source-shaped writer subset for the first H.264 realtime encoder picture
// path. Slice-header syntax order follows FFmpeg n8.0.1
// libavcodec/cbs_h264_syntax_template.c slice_header()/dec_ref_pic_marking().
// The payload remains deliberately bounded: exact I_PCM IDR/P recovery,
// no-residual P16x16 prediction, and explicitly configured CAVLC residual
// slices whose syntax is covered by local and FFmpeg oracles.

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

type EncoderMotionVectorDelta struct {
	X int32
	Y int32
}

type EncoderI420P16x16NoResidualConfig struct {
	Width  int
	Height int

	FrameNum                   uint32
	InitialQP                  int
	DisableDeblockingFilterIDC uint32
	FirstMBAddr                uint32
	MacroblockCount            uint32
	MVDX                       int32
	MVDY                       int32
	MVDs                       []EncoderMotionVectorDelta
	NALLengthSize              int
}

type EncoderResidualCoefficient struct {
	Pos   int
	Value int32
}

type EncoderChromaResidualCoefficients struct {
	Cb []EncoderResidualCoefficient
	Cr []EncoderResidualCoefficient
}

type EncoderI420P16x16ResidualConfig struct {
	Width  int
	Height int

	FrameNum                   uint32
	InitialQP                  int
	NextQP                     int
	NextQPs                    []int
	DisableDeblockingFilterIDC uint32
	FirstMBAddr                uint32
	MacroblockCount            uint32
	MVDX                       int32
	MVDY                       int32
	MVDs                       []EncoderMotionVectorDelta
	Coeff                      int32
	Coeffs                     []int32
	CoeffPos                   int
	CoeffPositions             []int
	LumaCoefficients           [][]EncoderResidualCoefficient
	ChromaDCCoeffCb            int32
	ChromaDCCoeffCr            int32
	ChromaDCCoeffs             [][2]int32
	ChromaDCCoefficients       []EncoderChromaResidualCoefficients
	ChromaDCCoeffPos           int
	ChromaDCCoeffPositions     []int
	ChromaACCoeffCb            int32
	ChromaACCoeffCr            int32
	ChromaACCoeffs             [][2]int32
	ChromaACCoefficients       []EncoderChromaResidualCoefficients
	ChromaACCoeffPos           int
	ChromaACCoeffPositions     []int
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

func BuildEncoderI420P16x16NoResidualSlice(cfg EncoderI420P16x16NoResidualConfig) (EncoderPSlice, error) {
	if cfg.NALLengthSize == 0 {
		cfg.NALLengthSize = 4
	}
	rbsp, err := EncodeI420P16x16NoResidualSliceRBSP(cfg)
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

func BuildEncoderI420P16x16ResidualSlice(cfg EncoderI420P16x16ResidualConfig) (EncoderPSlice, error) {
	if cfg.NALLengthSize == 0 {
		cfg.NALLengthSize = 4
	}
	rbsp, err := EncodeI420P16x16ResidualSliceRBSP(cfg)
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

	firstMB, macroblockCount := encoderI420SliceRange(cfg.Width, cfg.Height, cfg.FirstMBAddr, cfg.MacroblockCount)
	rbspCap, err := encoderSliceRBSPCapacity(macroblockCount, 384)
	if err != nil {
		return nil, err
	}
	bw := NewBitWriter(make([]byte, 0, rbspCap))
	if err := writeEncoderI420IDRSliceHeader(&bw, cfg); err != nil {
		return nil, err
	}

	mbWidth := (cfg.Width + 15) >> 4
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

	firstMB, macroblockCount := encoderI420SliceRange(cfg.Width, cfg.Height, cfg.FirstMBAddr, cfg.MacroblockCount)
	rbspCap, err := encoderSliceRBSPCapacity(macroblockCount, 384)
	if err != nil {
		return nil, err
	}
	bw := NewBitWriter(make([]byte, 0, rbspCap))
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

func EncodeI420P16x16NoResidualSliceRBSP(cfg EncoderI420P16x16NoResidualConfig) ([]byte, error) {
	if err := validateEncoderI420P16x16NoResidualConfig(cfg); err != nil {
		return nil, err
	}

	_, macroblockCount := encoderI420SliceRange(cfg.Width, cfg.Height, cfg.FirstMBAddr, cfg.MacroblockCount)
	rbspCap, err := encoderSliceRBSPCapacity(macroblockCount, 8)
	if err != nil {
		return nil, err
	}
	bw := NewBitWriter(make([]byte, 0, rbspCap))
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

	for i := 0; i < macroblockCount; i++ {
		mvdX, mvdY := cfg.MVDX, cfg.MVDY
		if len(cfg.MVDs) > 0 {
			mvdX, mvdY = cfg.MVDs[i].X, cfg.MVDs[i].Y
		}
		if err := bw.WriteUEGolomb(0); err != nil { // mb_skip_run
			return nil, err
		}
		mb := cavlcInterMacroblockSyntax{
			cavlcMacroblockSyntax: cavlcMacroblockSyntax{
				MBType:         MBType16x16 | MBTypeP0L0,
				PartitionCount: 1,
			},
			Ref: [2][4]int32{{0}},
		}
		mb.MVD[0][0] = [2]int32{mvdX, mvdY}
		if err := writeCAVLCInterPNoResidualMacroblock(&bw, mb, [2]uint32{1, 0}, true); err != nil {
			return nil, err
		}
	}
	bw.WriteRBSPTrailingBits()
	return bw.Bytes(), nil
}

func EncodeI420P16x16ResidualSliceRBSP(cfg EncoderI420P16x16ResidualConfig) ([]byte, error) {
	pps, sps, err := encoderI420P16x16ResidualParameterSets(cfg)
	if err != nil {
		return nil, err
	}
	return encodeI420P16x16ResidualSliceRBSP(cfg, pps, sps)
}

func encodeI420P16x16ResidualSliceRBSP(cfg EncoderI420P16x16ResidualConfig, pps *PPS, sps *SPS) ([]byte, error) {
	if err := validateEncoderI420P16x16ResidualConfig(cfg); err != nil {
		return nil, err
	}
	if pps == nil || sps == nil {
		return nil, ErrInvalidData
	}

	firstMBAddr, macroblockCount := encoderI420SliceRange(cfg.Width, cfg.Height, cfg.FirstMBAddr, cfg.MacroblockCount)
	rbspCap, err := encoderSliceRBSPCapacity(macroblockCount, 16)
	if err != nil {
		return nil, err
	}
	mbWidth := (cfg.Width + 15) >> 4
	mbHeight := (cfg.Height + 15) >> 4
	contextTables, err := newMacroblockTables(mbWidth, mbHeight, 1)
	if err != nil {
		return nil, err
	}
	bw := NewBitWriter(make([]byte, 0, rbspCap))
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

	qscale := cfg.InitialQP
	const residualSliceContextNum = uint16(1)
	for i := 0; i < macroblockCount; i++ {
		mvdX, mvdY := cfg.MVDX, cfg.MVDY
		if len(cfg.MVDs) > 0 {
			mvdX, mvdY = cfg.MVDs[i].X, cfg.MVDs[i].Y
		}
		coeff := cfg.Coeff
		if len(cfg.Coeffs) > 0 {
			coeff = cfg.Coeffs[i]
		}
		coeffPos := cfg.CoeffPos
		if len(cfg.CoeffPositions) > 0 {
			coeffPos = cfg.CoeffPositions[i]
		}
		lumaCoefficients := cfg.LumaCoefficients
		nextQP := cfg.NextQP
		if len(cfg.NextQPs) > 0 {
			nextQP = cfg.NextQPs[i]
		}
		chromaDCCb, chromaDCCr := cfg.ChromaDCCoeffCb, cfg.ChromaDCCoeffCr
		if len(cfg.ChromaDCCoeffs) > 0 {
			chromaDCCb, chromaDCCr = cfg.ChromaDCCoeffs[i][0], cfg.ChromaDCCoeffs[i][1]
		}
		chromaDCCoefficients := cfg.ChromaDCCoefficients
		chromaDCPos := int(h264ChromaDCScan[cfg.ChromaDCCoeffPos])
		if len(cfg.ChromaDCCoeffPositions) > 0 {
			chromaDCPos = int(h264ChromaDCScan[cfg.ChromaDCCoeffPositions[i]])
		}
		chromaACCb, chromaACCr := cfg.ChromaACCoeffCb, cfg.ChromaACCoeffCr
		if len(cfg.ChromaACCoeffs) > 0 {
			chromaACCb, chromaACCr = cfg.ChromaACCoeffs[i][0], cfg.ChromaACCoeffs[i][1]
		}
		chromaACCoefficients := cfg.ChromaACCoefficients
		chromaACPos := int(h264ZigzagScanCAVLC[1])
		if cfg.ChromaACCoeffPos != 0 {
			chromaACPos = cfg.ChromaACCoeffPos
		}
		if len(cfg.ChromaACCoeffPositions) > 0 {
			chromaACPos = cfg.ChromaACCoeffPositions[i]
		}
		cbp := 1
		if len(lumaCoefficients) > 0 {
			cbp = encoderLumaCBPFromCoefficients(lumaCoefficients[i])
		}
		if len(chromaACCoefficients) > 0 || chromaACCb != 0 || chromaACCr != 0 {
			cbp |= 0x20
		} else if len(chromaDCCoefficients) > 0 || chromaDCCb != 0 || chromaDCCr != 0 {
			cbp |= 0x10
		}
		if err := bw.WriteUEGolomb(0); err != nil { // mb_skip_run
			return nil, err
		}
		mb := cavlcInterMacroblockSyntax{
			cavlcMacroblockSyntax: cavlcMacroblockSyntax{
				MBType:         MBType16x16 | MBTypeP0L0,
				PartitionCount: 1,
				CBP:            cbp,
			},
			Ref: [2][4]int32{{0}},
		}
		mb.MVD[0][0] = [2]int32{mvdX, mvdY}
		var residual cavlcResidualContext
		mbAddr := firstMBAddr + i
		mbXY := mbAddr%mbWidth + (mbAddr/mbWidth)*contextTables.MBStride
		neighbors, err := contextTables.fillDecodeNeighborsFrame(mbXY, residualSliceContextNum, mb.MBType)
		if err != nil {
			return nil, err
		}
		if _, err := contextTables.fillResidualDecodeCaches(&residual, neighbors.residualNeighbors(mb.MBType, false)); err != nil {
			return nil, err
		}
		if len(lumaCoefficients) > 0 {
			for _, luma := range lumaCoefficients[i] {
				residual.MB[luma.Pos] = luma.Value
			}
		} else {
			residual.MB[coeffPos] = coeff
		}
		if len(chromaDCCoefficients) > 0 {
			for _, coeff := range chromaDCCoefficients[i].Cb {
				residual.MB[256+int(h264ChromaDCScan[coeff.Pos])] = coeff.Value
			}
			for _, coeff := range chromaDCCoefficients[i].Cr {
				residual.MB[512+int(h264ChromaDCScan[coeff.Pos])] = coeff.Value
			}
		} else {
			residual.MB[256+chromaDCPos] = chromaDCCb
			residual.MB[512+chromaDCPos] = chromaDCCr
		}
		if len(chromaACCoefficients) > 0 {
			for _, coeff := range chromaACCoefficients[i].Cb {
				residual.MB[256+coeff.Pos] = coeff.Value
			}
			for _, coeff := range chromaACCoefficients[i].Cr {
				residual.MB[512+coeff.Pos] = coeff.Value
			}
		} else {
			residual.MB[256+chromaACPos] = chromaACCb
			residual.MB[512+chromaACPos] = chromaACCr
		}
		if _, err := writeCAVLCInterPBoundedMacroblock(&bw, &residual, pps, sps, mb, [2]uint32{1, 0}, qscale, nextQP); err != nil {
			return nil, err
		}
		if err := contextTables.writeBackMacroblockTables(mbXY, mb.MBType, mb.CBPTable, mb.QScale, residualSliceContextNum); err != nil {
			return nil, err
		}
		if err := contextTables.writeBackNonZeroCount(mbXY, &residual.NonZeroCountCache); err != nil {
			return nil, err
		}
		qscale = nextQP
	}
	bw.WriteRBSPTrailingBits()
	return bw.Bytes(), nil
}

func encoderI420P16x16ResidualParameterSets(cfg EncoderI420P16x16ResidualConfig) (*PPS, *SPS, error) {
	spsRBSP, ppsRBSP, err := encodeBaselineParameterSetRBSPs(EncoderParameterSetConfig{
		ProfileIDC:         66,
		ConstraintSetFlags: 0x03,
		LevelIDC:           31,
		Width:              cfg.Width,
		Height:             cfg.Height,
		FrameRateNum:       30,
		FrameRateDen:       1,
		MaxReferenceFrames: 1,
		InitialQP:          cfg.InitialQP,
	})
	if err != nil {
		return nil, nil, err
	}
	sps, err := DecodeSPS(spsRBSP)
	if err != nil {
		return nil, nil, err
	}
	var spsList [maxSPSCount]*SPS
	spsList[sps.SPSID] = sps
	pps, err := DecodePPS(ppsRBSP, &spsList)
	if err != nil {
		return nil, nil, err
	}
	return pps, sps, nil
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
	lumaSamples, err := checkedMulInt(cfg.StrideY, cfg.Height)
	if err != nil {
		return ErrInvalidData
	}
	chromaHeight := cfg.Height >> 1
	cbSamples, err := checkedMulInt(cfg.StrideCb, chromaHeight)
	if err != nil {
		return ErrInvalidData
	}
	crSamples, err := checkedMulInt(cfg.StrideCr, chromaHeight)
	if err != nil {
		return ErrInvalidData
	}
	if len(cfg.Y) < lumaSamples {
		return ErrInvalidData
	}
	if len(cfg.Cb) < cbSamples || len(cfg.Cr) < crSamples {
		return ErrInvalidData
	}
	return nil
}

func validateEncoderI420P16x16NoResidualConfig(cfg EncoderI420P16x16NoResidualConfig) error {
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
	_, macroblockCount := encoderI420SliceRange(cfg.Width, cfg.Height, cfg.FirstMBAddr, cfg.MacroblockCount)
	if len(cfg.MVDs) > 0 && len(cfg.MVDs) != macroblockCount {
		return ErrInvalidData
	}
	return nil
}

func validateEncoderI420P16x16ResidualConfig(cfg EncoderI420P16x16ResidualConfig) error {
	if cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width&1 != 0 || cfg.Height&1 != 0 {
		return ErrInvalidData
	}
	if cfg.FrameNum >= 1<<8 ||
		cfg.InitialQP < 0 || cfg.InitialQP > 51 ||
		cfg.NextQP < 0 || cfg.NextQP > 51 ||
		cfg.DisableDeblockingFilterIDC > 2 ||
		cfg.NALLengthSize < 0 || cfg.NALLengthSize > 4 {
		return ErrInvalidData
	}
	if err := validateEncoderI420SliceRange(cfg.Width, cfg.Height, cfg.FirstMBAddr, cfg.MacroblockCount); err != nil {
		return err
	}
	_, macroblockCount := encoderI420SliceRange(cfg.Width, cfg.Height, cfg.FirstMBAddr, cfg.MacroblockCount)
	if len(cfg.MVDs) > 0 && len(cfg.MVDs) != macroblockCount {
		return ErrInvalidData
	}
	if len(cfg.NextQPs) > 0 && len(cfg.NextQPs) != macroblockCount {
		return ErrInvalidData
	}
	if len(cfg.CoeffPositions) > 0 && len(cfg.CoeffPositions) != macroblockCount {
		return ErrInvalidData
	}
	if len(cfg.LumaCoefficients) > 0 && len(cfg.LumaCoefficients) != macroblockCount {
		return ErrInvalidData
	}
	for _, qp := range cfg.NextQPs {
		if qp < 0 || qp > 51 {
			return ErrInvalidData
		}
	}
	if len(cfg.ChromaDCCoeffs) > 0 && len(cfg.ChromaDCCoeffs) != macroblockCount {
		return ErrInvalidData
	}
	if len(cfg.ChromaDCCoefficients) > 0 && len(cfg.ChromaDCCoefficients) != macroblockCount {
		return ErrInvalidData
	}
	if len(cfg.ChromaDCCoeffPositions) > 0 && len(cfg.ChromaDCCoeffPositions) != macroblockCount {
		return ErrInvalidData
	}
	if len(cfg.ChromaACCoeffs) > 0 && len(cfg.ChromaACCoeffs) != macroblockCount {
		return ErrInvalidData
	}
	if len(cfg.ChromaACCoefficients) > 0 && len(cfg.ChromaACCoefficients) != macroblockCount {
		return ErrInvalidData
	}
	if len(cfg.ChromaACCoeffPositions) > 0 && len(cfg.ChromaACCoeffPositions) != macroblockCount {
		return ErrInvalidData
	}
	hasChromaDC := len(cfg.ChromaDCCoefficients) > 0 || len(cfg.ChromaDCCoeffs) > 0 ||
		cfg.ChromaDCCoeffCb != 0 || cfg.ChromaDCCoeffCr != 0
	hasChromaAC := len(cfg.ChromaACCoefficients) > 0 || len(cfg.ChromaACCoeffs) > 0 ||
		cfg.ChromaACCoeffCb != 0 || cfg.ChromaACCoeffCr != 0
	if len(cfg.LumaCoefficients) > 0 {
		if len(cfg.Coeffs) > 0 || len(cfg.CoeffPositions) > 0 || cfg.Coeff != 0 || cfg.CoeffPos != 0 {
			return ErrInvalidData
		}
		hasLumaResidual := false
		for _, coeffs := range cfg.LumaCoefficients {
			if len(coeffs) > 0 {
				hasLumaResidual = true
			}
			if !validEncoderLumaResidualCoefficients(coeffs) {
				return ErrInvalidData
			}
		}
		if !hasLumaResidual && !hasChromaDC && !hasChromaAC {
			return ErrInvalidData
		}
	} else if len(cfg.CoeffPositions) > 0 {
		for _, pos := range cfg.CoeffPositions {
			if pos < 0 || pos >= 16 {
				return ErrInvalidData
			}
		}
	} else if cfg.CoeffPos < 0 || cfg.CoeffPos >= 16 {
		return ErrInvalidData
	}
	if len(cfg.LumaCoefficients) == 0 {
		if len(cfg.Coeffs) > 0 {
			if len(cfg.Coeffs) != macroblockCount {
				return ErrInvalidData
			}
			for _, coeff := range cfg.Coeffs {
				if coeff == 0 {
					return ErrInvalidData
				}
			}
		} else if cfg.Coeff == 0 {
			return ErrInvalidData
		}
	}
	if len(cfg.ChromaDCCoefficients) > 0 {
		if len(cfg.ChromaDCCoeffs) > 0 || len(cfg.ChromaDCCoeffPositions) > 0 ||
			cfg.ChromaDCCoeffCb != 0 || cfg.ChromaDCCoeffCr != 0 || cfg.ChromaDCCoeffPos != 0 {
			return ErrInvalidData
		}
		for _, coeffs := range cfg.ChromaDCCoefficients {
			if !validEncoderResidualCoefficients(coeffs.Cb, 0, len(h264ChromaDCScan)) ||
				!validEncoderResidualCoefficients(coeffs.Cr, 0, len(h264ChromaDCScan)) {
				return ErrInvalidData
			}
		}
	} else if len(cfg.ChromaDCCoeffs) > 0 {
		for _, coeff := range cfg.ChromaDCCoeffs {
			if coeff[0] == 0 || coeff[1] == 0 {
				return ErrInvalidData
			}
		}
	} else if (cfg.ChromaDCCoeffCb == 0) != (cfg.ChromaDCCoeffCr == 0) {
		return ErrInvalidData
	}
	hasScalarChromaDC := len(cfg.ChromaDCCoeffs) > 0 || cfg.ChromaDCCoeffCb != 0 || cfg.ChromaDCCoeffCr != 0
	if !hasScalarChromaDC && (len(cfg.ChromaDCCoeffPositions) > 0 || cfg.ChromaDCCoeffPos != 0) {
		return ErrInvalidData
	}
	if len(cfg.ChromaDCCoeffPositions) > 0 {
		for _, pos := range cfg.ChromaDCCoeffPositions {
			if pos < 0 || pos >= len(h264ChromaDCScan) {
				return ErrInvalidData
			}
		}
	} else if cfg.ChromaDCCoeffPos < 0 || cfg.ChromaDCCoeffPos >= len(h264ChromaDCScan) {
		return ErrInvalidData
	}
	if len(cfg.ChromaACCoefficients) > 0 {
		if len(cfg.ChromaACCoeffs) > 0 || len(cfg.ChromaACCoeffPositions) > 0 ||
			cfg.ChromaACCoeffCb != 0 || cfg.ChromaACCoeffCr != 0 || cfg.ChromaACCoeffPos != 0 {
			return ErrInvalidData
		}
		for _, coeffs := range cfg.ChromaACCoefficients {
			if !validEncoderResidualCoefficients(coeffs.Cb, 1, 16) ||
				!validEncoderResidualCoefficients(coeffs.Cr, 1, 16) {
				return ErrInvalidData
			}
		}
	} else if len(cfg.ChromaACCoeffs) > 0 {
		for _, coeff := range cfg.ChromaACCoeffs {
			if coeff[0] == 0 || coeff[1] == 0 {
				return ErrInvalidData
			}
		}
	} else if (cfg.ChromaACCoeffCb == 0) != (cfg.ChromaACCoeffCr == 0) {
		return ErrInvalidData
	}
	hasScalarChromaAC := len(cfg.ChromaACCoeffs) > 0 || cfg.ChromaACCoeffCb != 0 || cfg.ChromaACCoeffCr != 0
	if !hasScalarChromaAC && (len(cfg.ChromaACCoeffPositions) > 0 || cfg.ChromaACCoeffPos != 0) {
		return ErrInvalidData
	}
	if len(cfg.ChromaACCoeffPositions) > 0 {
		for _, pos := range cfg.ChromaACCoeffPositions {
			if pos <= 0 || pos >= 16 {
				return ErrInvalidData
			}
		}
	} else if cfg.ChromaACCoeffPos < 0 || cfg.ChromaACCoeffPos >= 16 {
		return ErrInvalidData
	}
	return nil
}

func validEncoderLumaResidualCoefficients(coeffs []EncoderResidualCoefficient) bool {
	if len(coeffs) == 0 {
		return true
	}
	return validEncoderResidualCoefficients(coeffs, 0, 256)
}

func validEncoderResidualCoefficients(coeffs []EncoderResidualCoefficient, minPos int, maxPos int) bool {
	if len(coeffs) == 0 || len(coeffs) > maxPos-minPos || minPos < 0 || maxPos > 256 || minPos >= maxPos {
		return false
	}
	var seen [256]bool
	for _, coeff := range coeffs {
		if coeff.Pos < minPos || coeff.Pos >= maxPos || coeff.Value == 0 || seen[coeff.Pos] {
			return false
		}
		seen[coeff.Pos] = true
	}
	return true
}

func encoderLumaCBPFromCoefficients(coeffs []EncoderResidualCoefficient) int {
	cbp := 0
	for _, coeff := range coeffs {
		cbp |= 1 << (coeff.Pos / 64)
	}
	return cbp
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
	total, err := encoderI420MacroblockCountChecked(width, height)
	if err != nil || total <= 0 {
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
	total, _ := encoderI420MacroblockCountChecked(width, height)
	first := int(firstMBAddr)
	count := int(macroblockCount)
	if count == 0 {
		count = total - first
	}
	return first, count
}

func encoderI420MacroblockCount(width int, height int) int {
	total, err := encoderI420MacroblockCountChecked(width, height)
	if err != nil {
		return 0
	}
	return total
}

func encoderI420MacroblockCountChecked(width int, height int) (int, error) {
	mbWidthInput, err := checkedAddInt(width, 15)
	if err != nil {
		return 0, ErrInvalidData
	}
	mbHeightInput, err := checkedAddInt(height, 15)
	if err != nil {
		return 0, ErrInvalidData
	}
	total, err := checkedMulInt(mbWidthInput>>4, mbHeightInput>>4)
	if err != nil {
		return 0, ErrInvalidData
	}
	if uint64(total) > uint64(^uint32(0)) {
		return 0, ErrInvalidData
	}
	return total, nil
}

func encoderSliceRBSPCapacity(macroblockCount int, bytesPerMacroblock int) (int, error) {
	if macroblockCount < 0 || bytesPerMacroblock < 0 {
		return 0, ErrInvalidData
	}
	payload, err := checkedMulInt(macroblockCount, bytesPerMacroblock)
	if err != nil {
		return 0, ErrInvalidData
	}
	capacity, err := checkedAddInt(32, payload)
	if err != nil {
		return 0, ErrInvalidData
	}
	return capacity, nil
}

const maxInt = int(^uint(0) >> 1)

func checkedAddInt(a int, b int) (int, error) {
	if b > 0 && a > maxInt-b {
		return 0, ErrInvalidData
	}
	if b < 0 && a < -maxInt-1-b {
		return 0, ErrInvalidData
	}
	return a + b, nil
}

func checkedMulInt(a int, b int) (int, error) {
	if a == 0 || b == 0 {
		return 0, nil
	}
	if a < 0 || b < 0 {
		return 0, ErrInvalidData
	}
	if a > maxInt/b {
		return 0, ErrInvalidData
	}
	return a * b, nil
}

func clampEncoderCoord(v int, limit int) int {
	if v >= limit {
		return limit - 1
	}
	return v
}
