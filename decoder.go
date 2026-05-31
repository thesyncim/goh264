// SPDX-License-Identifier: LGPL-2.1-or-later

package goh264

import "github.com/thesyncim/goh264/internal/h264"

var (
	ErrInvalidData = h264.ErrInvalidData
	ErrUnsupported = h264.ErrUnsupported
)

type Decoder struct {
	sps    [32]*h264.SPS
	pps    [256]*h264.PPS
	slices []h264.SliceHeader
}

type StreamInfo struct {
	SPSID           uint32
	ProfileIDC      uint8
	Profile         string
	LevelIDC        uint8
	Width           int
	Height          int
	ChromaFormatIDC uint32
	BitDepthLuma    int
	BitDepthChroma  int
}

func NewDecoder() *Decoder {
	return &Decoder{}
}

func (d *Decoder) ParseHeadersAnnexB(data []byte) (StreamInfo, error) {
	nals, err := h264.SplitAnnexB(data)
	if err != nil {
		return StreamInfo{}, err
	}

	var info StreamInfo
	haveSPS := false
	for _, nal := range nals {
		switch nal.Type {
		case h264.NALSPS:
			sps, err := h264.DecodeSPS(nal.RBSP)
			if err != nil {
				return StreamInfo{}, err
			}
			if sps.SPSID < uint32(len(d.sps)) {
				d.sps[sps.SPSID] = sps
			}
			if !haveSPS {
				info = streamInfoFromSPS(sps)
				haveSPS = true
			}
		case h264.NALPPS:
			pps, err := h264.DecodePPS(nal.RBSP, &d.sps)
			if err != nil {
				return StreamInfo{}, err
			}
			if pps.PPSID < uint32(len(d.pps)) {
				d.pps[pps.PPSID] = pps
			}
		case h264.NALSlice, h264.NALIDRSlice:
			slice, err := h264.ParseSliceHeader(nal, &d.pps)
			if err != nil {
				return StreamInfo{}, err
			}
			d.slices = append(d.slices, *slice)
		default:
			continue
		}
	}

	if !haveSPS {
		return StreamInfo{}, ErrInvalidData
	}
	return info, nil
}

func streamInfoFromSPS(sps *h264.SPS) StreamInfo {
	profileIDC := uint8(sps.ProfileIDC)
	return StreamInfo{
		SPSID:           sps.SPSID,
		ProfileIDC:      profileIDC,
		Profile:         profileName(profileIDC, uint8(sps.ConstraintSetFlags)),
		LevelIDC:        uint8(sps.LevelIDC),
		Width:           int(sps.Width),
		Height:          int(sps.Height),
		ChromaFormatIDC: sps.ChromaFormatIDC,
		BitDepthLuma:    int(sps.BitDepthLuma),
		BitDepthChroma:  int(sps.BitDepthChroma),
	}
}

func profileName(profileIDC uint8, constraintSetFlags uint8) string {
	switch profileIDC {
	case 66:
		if constraintSetFlags&0x03 == 0x03 {
			return "Constrained Baseline"
		}
		return "Baseline"
	case 77:
		return "Main"
	case 88:
		return "Extended"
	case 100:
		return "High"
	case 110:
		return "High 10"
	case 122:
		return "High 4:2:2"
	case 244:
		return "High 4:4:4 Predictive"
	default:
		return "Unknown"
	}
}
