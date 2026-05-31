// SPDX-License-Identifier: LGPL-2.1-or-later
//
// Source-shaped intra prediction mode helpers from FFmpeg n8.0.1
// libavcodec/h264_mvpred.h pred_intra_mode and libavcodec/h264_parse.c
// ff_h264_check_intra4x4_pred_mode / ff_h264_check_intra_pred_mode.

package h264

const (
	intraPredVertical      int8 = 0
	intraPredHorizontal    int8 = 1
	intraPredDC            int8 = 2
	intraPredDiagDownLeft  int8 = 3
	intraPredDiagDownRight int8 = 4
	intraPredVertRight     int8 = 5
	intraPredHorDown       int8 = 6
	intraPredVertLeft      int8 = 7
	intraPredHorUp         int8 = 8
	intraPredLeftDC        int8 = 9
	intraPredTopDC         int8 = 10
	intraPredDC128         int8 = 11

	intraPred8x8DC             = 0
	intraPred8x8Horizontal     = 1
	intraPred8x8Vertical       = 2
	intraPred8x8Plane          = 3
	intraPred8x8LeftDC         = 4
	intraPred8x8TopDC          = 5
	intraPred8x8AlzheimerL0TDC = 7
)

func predIntraMode(cache *[h264IntraPredModeCacheSize]int8, n int) (int8, error) {
	if cache == nil || n < 0 || n >= 16 {
		return 0, ErrInvalidData
	}
	index8 := int(h264Scan8[n])
	left := cache[index8-1]
	top := cache[index8-8]
	minMode := left
	if top < minMode {
		minMode = top
	}
	if minMode < 0 {
		return intraPredDC, nil
	}
	return minMode, nil
}

func predIntra4x4Modes(cache *[h264IntraPredModeCacheSize]int8) ([16]int8, error) {
	var pred [16]int8
	if cache == nil {
		return pred, ErrInvalidData
	}
	for i := range pred {
		mode, err := predIntraMode(cache, i)
		if err != nil {
			return pred, err
		}
		pred[i] = mode
	}
	return pred, nil
}

func checkIntra4x4PredModeCache(cache *[h264IntraPredModeCacheSize]int8, topSamplesAvailable uint16, leftSamplesAvailable uint16) error {
	if cache == nil {
		return ErrInvalidData
	}
	top := [12]int8{
		-1, 0, intraPredLeftDC, -1, -1, -1, -1, -1, 0,
	}
	left := [12]int8{
		0, -1, intraPredTopDC, 0, -1, -1, -1, 0, -1, intraPredDC128,
	}

	if topSamplesAvailable&0x8000 == 0 {
		for i := 0; i < 4; i++ {
			idx := int(h264Scan8[0]) + i
			mode := cache[idx]
			if mode < 0 || int(mode) >= len(top) {
				return ErrInvalidData
			}
			status := top[mode]
			if status < 0 {
				return ErrInvalidData
			}
			if status != 0 {
				cache[idx] = status
			}
		}
	}

	if leftSamplesAvailable&0x8888 != 0x8888 {
		mask := [4]uint16{0x8000, 0x2000, 0x0080, 0x0020}
		for i := 0; i < 4; i++ {
			if leftSamplesAvailable&mask[i] != 0 {
				continue
			}
			idx := int(h264Scan8[0]) + 8*i
			mode := cache[idx]
			if mode < 0 || int(mode) >= len(left) {
				return ErrInvalidData
			}
			status := left[mode]
			if status < 0 {
				return ErrInvalidData
			}
			if status != 0 {
				cache[idx] = status
			}
		}
	}
	return nil
}

func checkIntraPredMode(mode int, topSamplesAvailable uint16, leftSamplesAvailable uint16, isChroma bool) (int, error) {
	top := [4]int{
		intraPred8x8LeftDC, intraPred8x8Horizontal, -1, -1,
	}
	left := [5]int{
		intraPred8x8TopDC, -1, intraPred8x8Vertical, -1, intraPredDC1288x8,
	}

	if mode < 0 || mode > 3 {
		return 0, ErrInvalidData
	}
	if topSamplesAvailable&0x8000 == 0 {
		mode = top[mode]
		if mode < 0 {
			return 0, ErrInvalidData
		}
	}
	if leftSamplesAvailable&0x8080 != 0x8080 {
		if mode < 0 || mode >= len(left) {
			return 0, ErrInvalidData
		}
		mode = left[mode]
		if mode < 0 {
			return 0, ErrInvalidData
		}
		if isChroma && leftSamplesAvailable&0x8080 != 0 {
			mode = intraPred8x8AlzheimerL0TDC +
				boolToInt(leftSamplesAvailable&0x8000 == 0) +
				2*boolToInt(mode == intraPredDC1288x8)
		}
	}
	return mode, nil
}
