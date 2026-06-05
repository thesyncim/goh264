// SPDX-License-Identifier: LGPL-2.1-or-later
//
// Source-shaped port of the bounded MSB-first pieces of FFmpeg n8.0.1
// libavcodec/put_bits.h and libavcodec/golomb.h needed by the H.264 encoder
// path. Acceleration and table shortcuts stay out of this first writer slice.

package h264

import "math/bits"

type BitWriter struct {
	buf    []byte
	bitPos uint32
}

func NewBitWriter(dst []byte) BitWriter {
	return BitWriter{
		buf:    dst,
		bitPos: uint32(len(dst)) * 8,
	}
}

func (bw *BitWriter) BitLen() uint32 {
	if bw == nil {
		return 0
	}
	return bw.bitPos
}

func (bw *BitWriter) Bytes() []byte {
	if bw == nil {
		return nil
	}
	return bw.buf
}

func (bw *BitWriter) ByteAligned() bool {
	return bw == nil || bw.bitPos&7 == 0
}

func (bw *BitWriter) WriteBit(v uint32) {
	if bw.bitPos&7 == 0 {
		bw.buf = append(bw.buf, 0)
	}
	if v&1 != 0 {
		bw.buf[len(bw.buf)-1] |= 1 << uint(7-(bw.bitPos&7))
	}
	bw.bitPos++
}

func (bw *BitWriter) WriteBits(v uint32, n uint32) error {
	if n > 32 {
		return ErrInvalidData
	}
	for i := n; i > 0; i-- {
		bw.WriteBit(v >> (i - 1))
	}
	return nil
}

func (bw *BitWriter) WriteUEGolomb(v uint32) error {
	codeNum := v + 1
	if codeNum == 0 {
		return ErrInvalidData
	}
	width := uint32(bits.Len32(codeNum))
	for i := uint32(0); i < width-1; i++ {
		bw.WriteBit(0)
	}
	return bw.WriteBits(codeNum, width)
}

func (bw *BitWriter) WriteSEGolomb(v int32) error {
	codeNum := int64(v) * 2
	if v <= 0 {
		codeNum = -codeNum
	} else {
		codeNum--
	}
	if codeNum < 0 || codeNum > int64(^uint32(0)-1) {
		return ErrInvalidData
	}
	return bw.WriteUEGolomb(uint32(codeNum))
}

func (bw *BitWriter) WriteRBSPTrailingBits() {
	bw.WriteBit(1)
	for bw.bitPos&7 != 0 {
		bw.WriteBit(0)
	}
}

func AppendEBSP(dst []byte, rbsp []byte) []byte {
	zeros := 0
	for _, b := range rbsp {
		if zeros >= 2 && b <= 0x03 {
			dst = append(dst, 0x03)
			zeros = 0
		}
		dst = append(dst, b)
		if b == 0 {
			zeros++
		} else {
			zeros = 0
		}
	}
	return dst
}

func AppendNAL(dst []byte, refIDC uint8, typ NALUnitType, rbsp []byte) ([]byte, error) {
	if refIDC > 3 || typ > 31 {
		return dst, ErrInvalidData
	}
	dst = append(dst, (refIDC<<5)|uint8(typ))
	dst = AppendEBSP(dst, rbsp)
	return dst, nil
}

func AppendAnnexBNAL(dst []byte, refIDC uint8, typ NALUnitType, rbsp []byte) ([]byte, error) {
	start := len(dst)
	dst = append(dst, 0x00, 0x00, 0x00, 0x01)
	dst, err := AppendNAL(dst, refIDC, typ, rbsp)
	if err != nil {
		return dst[:start], err
	}
	return dst, nil
}

func AppendAVCNAL(dst []byte, nalLengthSize int, refIDC uint8, typ NALUnitType, rbsp []byte) ([]byte, error) {
	if nalLengthSize < 1 || nalLengthSize > 4 {
		return dst, ErrInvalidData
	}
	start := len(dst)
	for i := 0; i < nalLengthSize; i++ {
		dst = append(dst, 0)
	}
	nalStart := len(dst)
	dst, err := AppendNAL(dst, refIDC, typ, rbsp)
	if err != nil {
		return dst[:start], err
	}
	nalLen := len(dst) - nalStart
	if uint64(nalLen) >= uint64(1)<<uint(nalLengthSize*8) {
		return dst[:start], ErrInvalidData
	}
	for i := nalLengthSize - 1; i >= 0; i-- {
		dst[start+i] = byte(nalLen)
		nalLen >>= 8
	}
	return dst, nil
}

func AppendAVCDecoderConfigurationRecord(dst []byte, profileIDC uint8, profileCompatibility uint8, levelIDC uint8, nalLengthSize int, spsNALs [][]byte, ppsNALs [][]byte) ([]byte, error) {
	if nalLengthSize < 1 || nalLengthSize > 4 || len(spsNALs) == 0 || len(spsNALs) > 31 || len(ppsNALs) == 0 || len(ppsNALs) > 255 {
		return dst, ErrInvalidData
	}
	start := len(dst)
	dst = append(dst,
		0x01,
		profileIDC,
		profileCompatibility,
		levelIDC,
		0xfc|byte(nalLengthSize-1),
		0xe0|byte(len(spsNALs)),
	)
	for _, raw := range spsNALs {
		next, err := appendAVCConfigRawNAL(dst, raw, NALSPS)
		if err != nil {
			return dst[:start], err
		}
		dst = next
	}
	dst = append(dst, byte(len(ppsNALs)))
	for _, raw := range ppsNALs {
		next, err := appendAVCConfigRawNAL(dst, raw, NALPPS)
		if err != nil {
			return dst[:start], err
		}
		dst = next
	}
	return dst, nil
}

func appendAVCConfigRawNAL(dst []byte, raw []byte, wantType NALUnitType) ([]byte, error) {
	if len(raw) == 0 || len(raw) > 0xffff || raw[0]&0x80 != 0 || NALUnitType(raw[0]&0x1f) != wantType {
		return dst, ErrInvalidData
	}
	dst = append(dst, byte(len(raw)>>8), byte(len(raw)))
	dst = append(dst, raw...)
	return dst, nil
}
