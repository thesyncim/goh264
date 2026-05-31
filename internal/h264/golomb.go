// SPDX-License-Identifier: LGPL-2.1-or-later
//
// Source-shaped port of the H.264 Exp-Golomb readers from FFmpeg n8.0.1
// libavcodec/golomb.h. The first pass keeps exact syntax semantics and
// postpones FFmpeg's table/cached-reader acceleration.

package h264

func (gb *bitReader) readUEGolombLong() (uint32, error) {
	var leadingZeros uint32
	for {
		if leadingZeros >= 32 {
			return 0, ErrInvalidData
		}
		bit, err := gb.readBit()
		if err != nil {
			return 0, err
		}
		if bit == 1 {
			break
		}
		leadingZeros++
	}

	if leadingZeros == 0 {
		return 0, nil
	}

	suffix, err := gb.readBits(leadingZeros)
	if err != nil {
		return 0, err
	}
	return (uint32(1) << leadingZeros) - 1 + suffix, nil
}

func (gb *bitReader) readUEGolomb31() (uint32, error) {
	return gb.readUEGolombLong()
}

func (gb *bitReader) readSEGolombLong() (int32, error) {
	codeNum, err := gb.readUEGolombLong()
	if err != nil {
		return 0, err
	}

	value := int32((codeNum + 1) >> 1)
	if codeNum&1 == 0 {
		value = -value
	}
	return value, nil
}
