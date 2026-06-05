// SPDX-License-Identifier: LGPL-2.1-or-later
//
// Source-shaped port of the H.264 NAL constants from FFmpeg n8.0.1
// libavcodec/h264.h and the H.264 RBSP extraction path from
// libavcodec/h2645_parse.c.

package h264

type NALUnitType uint8

const (
	NALUnspecified     NALUnitType = 0
	NALSlice           NALUnitType = 1
	NALDPA             NALUnitType = 2
	NALDPB             NALUnitType = 3
	NALDPC             NALUnitType = 4
	NALIDRSlice        NALUnitType = 5
	NALSEI             NALUnitType = 6
	NALSPS             NALUnitType = 7
	NALPPS             NALUnitType = 8
	NALAUD             NALUnitType = 9
	NALEndSequence     NALUnitType = 10
	NALEndStream       NALUnitType = 11
	NALFillerData      NALUnitType = 12
	NALSPSext          NALUnitType = 13
	NALPrefix          NALUnitType = 14
	NALSubSPS          NALUnitType = 15
	NALDPS             NALUnitType = 16
	NALReserved17      NALUnitType = 17
	NALReserved18      NALUnitType = 18
	NALAuxiliarySlice  NALUnitType = 19
	NALExtenSlice      NALUnitType = 20
	NALDepthExtenSlice NALUnitType = 21
	NALReserved22      NALUnitType = 22
	NALReserved23      NALUnitType = 23
	NALUnspecified24   NALUnitType = 24
	NALUnspecified25   NALUnitType = 25
	NALUnspecified26   NALUnitType = 26
	NALUnspecified27   NALUnitType = 27
	NALUnspecified28   NALUnitType = 28
	NALUnspecified29   NALUnitType = 29
	NALUnspecified30   NALUnitType = 30
	NALUnspecified31   NALUnitType = 31
)

type NALUnit struct {
	RefIDC uint8
	Type   NALUnitType
	Raw    []byte
	RBSP   []byte
}

type AVCDecoderConfigurationRecord struct {
	NALLengthSize int
	FirstSPSID    uint32
	SPS           [maxSPSCount]*SPS
	PPS           [maxPPSCount]*PPS
}

type H264PacketFormat uint8

const (
	H264PacketFormatAnnexB H264PacketFormat = iota
	H264PacketFormatAVC
)

func SplitAnnexB(data []byte) ([]NALUnit, error) {
	var out []NALUnit

	start, prefixLen, ok := findStartCode(data, 0)
	if !ok {
		return nil, ErrInvalidData
	}

	for ok {
		nalStart := start + prefixLen
		nextStart, nextPrefixLen, nextOK := findStartCode(data, nalStart)
		nalEnd := len(data)
		if nextOK {
			nalEnd = nextStart
		}

		if nalEnd > nalStart {
			nal, err := parseNAL(data[nalStart:nalEnd])
			if err != nil {
				return nil, err
			}
			out = append(out, nal)
		}

		if !nextOK {
			break
		}
		start, prefixLen, ok = nextStart, nextPrefixLen, true
	}

	if len(out) == 0 {
		return nil, ErrInvalidData
	}
	return out, nil
}

func DecodeAVCDecoderConfigurationRecord(data []byte) (AVCDecoderConfigurationRecord, error) {
	var cfg AVCDecoderConfigurationRecord
	if len(data) < 7 || data[0] != 1 {
		return cfg, ErrInvalidData
	}

	cfg.NALLengthSize = int(data[4]&0x03) + 1
	pos := 6
	spsCount := int(data[5] & 0x1f)
	haveSPS := false
	for i := 0; i < spsCount; i++ {
		raw, err := readAVCConfigNAL(data, &pos)
		if err != nil {
			return cfg, err
		}
		nal, err := parseAVCConfigNAL(raw, NALSPS)
		if err != nil {
			return cfg, err
		}
		sps, err := DecodeSPSFromNAL(nal)
		if err != nil {
			return cfg, err
		}
		cfg.SPS[sps.SPSID] = sps
		if !haveSPS {
			cfg.FirstSPSID = sps.SPSID
			haveSPS = true
		}
	}

	if pos >= len(data) {
		return cfg, ErrInvalidData
	}
	ppsCount := int(data[pos])
	pos++
	havePPS := false
	for i := 0; i < ppsCount; i++ {
		raw, err := readAVCConfigNAL(data, &pos)
		if err != nil {
			return cfg, err
		}
		nal, err := parseAVCConfigNAL(raw, NALPPS)
		if err != nil {
			return cfg, err
		}
		pps, err := DecodePPS(nal.RBSP, &cfg.SPS)
		if err != nil {
			return cfg, err
		}
		cfg.PPS[pps.PPSID] = pps
		havePPS = true
	}
	if !haveSPS || !havePPS {
		return cfg, ErrInvalidData
	}
	return cfg, nil
}

// IsAVCDecoderConfigurationRecord mirrors the FFmpeg n8.0.1
// libavcodec/h264dec.c is_avcc_extradata record walk. The public Go facade
// accepts standalone avcC records with non-zero profile-compatibility flags,
// while FFmpeg's frame-path wrapper only probes those packets after extra gates.
func IsAVCDecoderConfigurationRecord(data []byte) bool {
	if len(data) < 9 || data[0] != 1 || data[4]&0xfc != 0xfc {
		return false
	}
	pos := 6
	spsCount := int(data[5] & 0x1f)
	if spsCount == 0 {
		return false
	}
	for i := 0; i < spsCount; i++ {
		raw, ok := peekAVCConfigNAL(data, &pos)
		if !ok || raw[0]&0x9f != uint8(NALSPS) {
			return false
		}
	}
	if pos >= len(data) {
		return false
	}
	ppsCount := int(data[pos])
	pos++
	if ppsCount == 0 {
		return false
	}
	for i := 0; i < ppsCount; i++ {
		raw, ok := peekAVCConfigNAL(data, &pos)
		if !ok || raw[0]&0x9f != uint8(NALPPS) {
			return false
		}
	}
	return true
}

func peekAVCConfigNAL(data []byte, pos *int) ([]byte, bool) {
	if pos == nil || *pos < 0 || *pos+2 > len(data) {
		return nil, false
	}
	nalSize := int(data[*pos])<<8 | int(data[*pos+1])
	*pos += 2
	if nalSize <= 0 || nalSize > len(data)-*pos {
		return nil, false
	}
	raw := data[*pos : *pos+nalSize]
	*pos += nalSize
	return raw, true
}

func readAVCConfigNAL(data []byte, pos *int) ([]byte, error) {
	if pos == nil || *pos < 0 || *pos+2 > len(data) {
		return nil, ErrInvalidData
	}
	nalSize := int(data[*pos])<<8 | int(data[*pos+1])
	*pos += 2
	if nalSize <= 0 || nalSize > len(data)-*pos {
		return nil, ErrInvalidData
	}
	raw := data[*pos : *pos+nalSize]
	*pos += nalSize
	return raw, nil
}

func parseAVCConfigNAL(raw []byte, wantType NALUnitType) (NALUnit, error) {
	if len(raw) == 0 || raw[0]&0x80 != 0 || NALUnitType(raw[0]&0x1f) != wantType {
		return NALUnit{}, ErrInvalidData
	}
	nal, err := parseNAL(raw)
	if err == nil {
		return nal, nil
	}
	return NALUnit{
		RefIDC: (raw[0] >> 5) & 3,
		Type:   wantType,
		Raw:    raw,
		RBSP:   raw[1:],
	}, nil
}

func SplitAVCC(data []byte, nalLengthSize int) ([]NALUnit, error) {
	if nalLengthSize < 1 || nalLengthSize > 4 {
		return nil, ErrInvalidData
	}

	var out []NALUnit
	for pos := 0; pos < len(data); {
		if pos >= len(data)-nalLengthSize {
			return nil, ErrInvalidData
		}

		nalSize := 0
		for i := 0; i < nalLengthSize; i++ {
			nalSize = (nalSize << 8) | int(data[pos])
			pos++
		}
		if nalSize <= 0 || nalSize > len(data)-pos {
			return nil, ErrInvalidData
		}

		nal, err := parseNAL(data[pos : pos+nalSize])
		if err != nil {
			return nil, err
		}
		out = append(out, nal)
		pos += nalSize
	}

	if len(out) == 0 {
		return nil, ErrInvalidData
	}
	return out, nil
}

// SplitAutoPacket ports FFmpeg's nal_length_size==4 is_avc sniffing branch
// from libavcodec/h264dec.c decode_nal_units, then falls back to the explicit
// configured packet format or Annex B/4-byte AVC detection for unconfigured
// public decode calls.
func SplitAutoPacket(data []byte, configuredNALLengthSize int) ([]NALUnit, H264PacketFormat, error) {
	if len(data) == 0 {
		return nil, 0, ErrInvalidData
	}
	if configuredNALLengthSize < 0 || configuredNALLengthSize > 4 {
		return nil, 0, ErrInvalidData
	}

	if configuredNALLengthSize == 4 {
		if len(data) > 8 && be32(data, 0) == 1 && be32(data, 5) > uint32(len(data)) {
			nals, err := SplitAnnexB(data)
			return nals, H264PacketFormatAnnexB, err
		}
		if len(data) > 3 && be32(data, 0) > 1 && be32(data, 0) <= uint32(len(data)) {
			nals, err := SplitAVCC(data, 4)
			return nals, H264PacketFormatAVC, err
		}
	}
	if configuredNALLengthSize >= 1 {
		nals, err := SplitAVCC(data, configuredNALLengthSize)
		if err != nil && looksAnnexB(data) {
			nals, err = SplitAnnexB(data)
			return nals, H264PacketFormatAnnexB, err
		}
		return nals, H264PacketFormatAVC, err
	}
	if looksAnnexB(data) {
		nals, err := SplitAnnexB(data)
		return nals, H264PacketFormatAnnexB, err
	}
	if len(data) > 3 && be32(data, 0) > 1 && be32(data, 0) <= uint32(len(data)) {
		nals, err := SplitAVCC(data, 4)
		return nals, H264PacketFormatAVC, err
	}
	return nil, 0, ErrInvalidData
}

func looksAnnexB(data []byte) bool {
	_, _, ok := findStartCode(data, 0)
	return ok
}

func be32(data []byte, off int) uint32 {
	return uint32(data[off])<<24 | uint32(data[off+1])<<16 | uint32(data[off+2])<<8 | uint32(data[off+3])
}

func parseNAL(raw []byte) (NALUnit, error) {
	if len(raw) == 0 {
		return NALUnit{}, ErrInvalidData
	}

	header := raw[0]
	if header&0x80 != 0 {
		return NALUnit{}, ErrInvalidData
	}

	rbsp, err := AppendRBSP(nil, raw[1:])
	if err != nil {
		return NALUnit{}, err
	}

	return NALUnit{
		RefIDC: (header >> 5) & 3,
		Type:   NALUnitType(header & 0x1f),
		Raw:    raw,
		RBSP:   rbsp,
	}, nil
}

func findStartCode(data []byte, from int) (start int, prefixLen int, ok bool) {
	for i := from; i+3 <= len(data); i++ {
		if data[i] != 0 || data[i+1] != 0 {
			continue
		}
		if data[i+2] == 1 {
			return i, 3, true
		}
		if i+4 <= len(data) && data[i+2] == 0 && data[i+3] == 1 {
			return i, 4, true
		}
	}
	return 0, 0, false
}

func AppendRBSP(dst []byte, ebsp []byte) ([]byte, error) {
	zeros := 0
	for _, b := range ebsp {
		if zeros >= 2 {
			switch b {
			case 0x03:
				zeros = 0
				continue
			// FFmpeg h2645_parse.c keeps 00 00 00 trailing padding bytes
			// and lets get_bit_length discard them later.
			case 0x01, 0x02:
				return nil, ErrInvalidData
			}
		}

		dst = append(dst, b)
		if b == 0 {
			zeros++
		} else {
			zeros = 0
		}
	}
	return dst, nil
}
