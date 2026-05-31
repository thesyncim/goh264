// SPDX-License-Identifier: LGPL-2.1-or-later
//
// Source-shaped port of the bounded MSB-first pieces of FFmpeg n8.0.1
// libavcodec/get_bits.h used by the H.264 decoder path.

package h264

type bitReader struct {
	buf     []byte
	bitPos  uint32
	numBits uint32
}

func newBitReader(buf []byte) bitReader {
	return bitReader{
		buf:     buf,
		numBits: uint32(len(buf)) * 8,
	}
}

func (gb *bitReader) bitsLeft() int32 {
	return int32(gb.numBits) - int32(gb.bitPos)
}

func (gb *bitReader) readBit() (uint32, error) {
	if gb.bitsLeft() < 1 {
		return 0, ErrInvalidData
	}

	byteIndex := gb.bitPos >> 3
	bitOffset := 7 - (gb.bitPos & 7)
	gb.bitPos++

	return uint32((gb.buf[byteIndex] >> bitOffset) & 1), nil
}

func (gb *bitReader) readBits(n uint32) (uint32, error) {
	if n > 32 || int32(n) > gb.bitsLeft() {
		return 0, ErrInvalidData
	}

	var out uint32
	for ; n > 0; n-- {
		bit, err := gb.readBit()
		if err != nil {
			return 0, err
		}
		out = (out << 1) | bit
	}
	return out, nil
}

func (gb *bitReader) skipBits(n uint32) error {
	if int32(n) > gb.bitsLeft() {
		return ErrInvalidData
	}
	gb.bitPos += n
	return nil
}
