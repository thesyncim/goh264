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
			case 0x00, 0x01, 0x02:
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
