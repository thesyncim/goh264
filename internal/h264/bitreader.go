// SPDX-License-Identifier: LGPL-2.1-or-later
//
// Source-shaped port of the bounded MSB-first pieces of FFmpeg n8.0.1
// libavcodec/get_bits.h used by the H.264 decoder path.

package h264

import "math/bits"

type bitReader struct {
	buf     []byte
	bitPos  uint32
	numBits uint32
	invalid bool
}

func newBitReader(buf []byte) bitReader {
	if len(buf) > maxBitReaderByteLen {
		return bitReader{buf: buf, numBits: uint32(maxBitReaderByteLen) * 8, invalid: true}
	}
	return bitReader{
		buf:     buf,
		numBits: uint32(len(buf)) * 8,
	}
}

func newRBSPBitReader(buf []byte) (bitReader, error) {
	numBits, err := rbspBitLength(buf)
	if err != nil {
		return bitReader{}, err
	}
	return bitReader{
		buf:     buf,
		numBits: numBits,
	}, nil
}

func rbspBitLength(buf []byte) (uint32, error) {
	if len(buf) > maxBitReaderByteLen {
		return 0, ErrInvalidData
	}
	size := len(buf)
	for size > 0 && buf[size-1] == 0 {
		size--
	}
	if size == 0 {
		return 0, ErrInvalidData
	}

	trailingPadding := bits.TrailingZeros8(buf[size-1]) + 1
	numBits := size*8 - trailingPadding
	if numBits < 0 {
		return 0, ErrInvalidData
	}
	return uint32(numBits), nil
}

func (gb *bitReader) bitsLeft() int32 {
	if gb == nil || gb.invalid {
		return -1
	}
	return int32(gb.numBits) - int32(gb.bitPos)
}

func (gb *bitReader) readBit() (uint32, error) {
	if gb == nil || gb.invalid {
		return 0, ErrInvalidData
	}
	if gb.bitsLeft() < 1 {
		return 0, ErrInvalidData
	}

	byteIndex := gb.bitPos >> 3
	bitOffset := 7 - (gb.bitPos & 7)
	gb.bitPos++

	return uint32((gb.buf[byteIndex] >> bitOffset) & 1), nil
}

func (gb *bitReader) readBits(n uint32) (uint32, error) {
	if gb == nil || gb.invalid {
		return 0, ErrInvalidData
	}
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

func (gb *bitReader) showBits(n uint32) (uint32, error) {
	if gb == nil || gb.invalid {
		return 0, ErrInvalidData
	}
	if n > 32 || int32(n) > gb.bitsLeft() {
		return 0, ErrInvalidData
	}

	bitPos := gb.bitPos
	out, err := gb.readBits(n)
	gb.bitPos = bitPos
	return out, err
}

func (gb *bitReader) showBitsPadded(n uint32) uint32 {
	if gb == nil || gb.invalid {
		return 0
	}
	if n > 32 {
		n = 32
	}

	available := n
	if left := gb.bitsLeft(); left < int32(available) {
		if left <= 0 {
			return 0
		}
		available = uint32(left)
	}

	bitPos := gb.bitPos
	var out uint32
	for i := uint32(0); i < available; i++ {
		byteIndex := bitPos >> 3
		bitOffset := 7 - (bitPos & 7)
		out = (out << 1) | uint32((gb.buf[byteIndex]>>bitOffset)&1)
		bitPos++
	}
	return out << (n - available)
}

func (gb *bitReader) skipBits(n uint32) error {
	if gb == nil || gb.invalid {
		return ErrInvalidData
	}
	if int32(n) > gb.bitsLeft() {
		return ErrInvalidData
	}
	gb.bitPos += n
	return nil
}

func (gb *bitReader) alignToByte() error {
	if gb == nil || gb.invalid {
		return ErrInvalidData
	}
	aligned := (gb.bitPos + 7) &^ 7
	if aligned > gb.numBits {
		return ErrInvalidData
	}
	gb.bitPos = aligned
	return nil
}

func (gb *bitReader) readAlignedBytes(n int) ([]byte, error) {
	if gb == nil || gb.invalid {
		return nil, ErrInvalidData
	}
	if n < 0 {
		return nil, ErrInvalidData
	}
	aligned := (gb.bitPos + 7) &^ 7
	if aligned > gb.numBits {
		return nil, ErrInvalidData
	}
	if uint64(n)*8 > uint64(gb.numBits-aligned) {
		return nil, ErrInvalidData
	}
	start := int(aligned >> 3)
	gb.bitPos = aligned + uint32(n)*8
	return gb.buf[start : start+n], nil
}

func (gb *bitReader) remainingAlignedBytes() ([]byte, error) {
	if gb == nil {
		return nil, ErrInvalidData
	}
	if err := gb.alignToByte(); err != nil {
		return nil, err
	}
	left := gb.bitsLeft()
	if left < 0 {
		return nil, ErrInvalidData
	}
	n := int((left + 7) >> 3)
	start := int(gb.bitPos >> 3)
	if start < 0 || n < 0 || start+n > len(gb.buf) {
		return nil, ErrInvalidData
	}
	return gb.buf[start : start+n], nil
}

// CABAC is initialized from FFmpeg's raw aligned byte tail, including
// rbsp_trailing_bits; non-CABAC syntax readers stay bounded by numBits.
func (gb *bitReader) remainingAlignedRawBytes() ([]byte, error) {
	if gb == nil || gb.invalid {
		return nil, ErrInvalidData
	}
	aligned := (gb.bitPos + 7) &^ 7
	if uint64(aligned) > uint64(len(gb.buf))*8 {
		return nil, ErrInvalidData
	}
	gb.bitPos = aligned
	start := int(gb.bitPos >> 3)
	if start < 0 || start > len(gb.buf) {
		return nil, ErrInvalidData
	}
	return gb.buf[start:], nil
}

const maxBitReaderByteLen = int((^uint32(0) >> 1) / 8)
