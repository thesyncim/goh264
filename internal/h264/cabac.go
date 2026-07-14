// SPDX-License-Identifier: LGPL-2.1-or-later
//
// Source-shaped port of FFmpeg n8.0.1 libavcodec/cabac.c and
// libavcodec/cabac_functions.h primitives needed by the H.264 CABAC path.

package h264

import "unsafe"

const (
	cabacBits = 16
	cabacMask = (1 << cabacBits) - 1

	h264NormShiftOffset              = 0
	h264LPSRangeOffset               = 512
	h264MLPSStateOffset              = 1024
	h264LastCoeffFlagOffset8x8Offset = 1280
)

type cabacContext struct {
	low             int32
	rng             int32
	bytestreamStart int
	bytestream      int
	bytestreamEnd   int
	buf             []byte
}

func initCABACDecoder(buf []byte) (cabacContext, error) {
	aligned := true
	if len(buf) > 2 {
		aligned = uintptr(unsafe.Pointer(&buf[2]))&1 == 0
	}
	return initCABACDecoderAligned(buf, aligned)
}

func initCABACDecoderAligned(buf []byte, alignedAfterSecondByte bool) (cabacContext, error) {
	c := cabacContext{
		rng:           0x1fe,
		bytestreamEnd: len(buf),
		buf:           buf,
	}
	if len(buf) < 2 {
		return c, ErrInvalidData
	}

	c.low = int32(buf[c.bytestream]) << 18
	c.bytestream++
	c.low += int32(buf[c.bytestream]) << 10
	c.bytestream++
	if alignedAfterSecondByte {
		c.low += 1 << 9
	} else {
		if c.bytestream >= len(buf) {
			return c, ErrInvalidData
		}
		c.low += (int32(buf[c.bytestream]) << 2) + 2
		c.bytestream++
	}

	if (c.rng << (cabacBits + 1)) < c.low {
		return c, ErrInvalidData
	}
	return c, nil
}

func (c *cabacContext) refill() {
	x := int32(0)
	if c.bytestream < c.bytestreamEnd {
		x += int32(c.buf[c.bytestream]) << 9
	}
	if c.bytestream+1 < c.bytestreamEnd {
		x += int32(c.buf[c.bytestream+1]) << 1
	}
	c.low += x
	c.low -= cabacMask
	if c.bytestream < c.bytestreamEnd {
		c.bytestream += cabacBits / 8
	}
}

func (c *cabacContext) refill2() {
	xor := uint32(c.low ^ (c.low - 1))
	i := 7 - int(h264CABACTables[h264NormShiftOffset+int(xor>>(cabacBits-1))])

	x := int32(-cabacMask)
	if c.bytestream < c.bytestreamEnd {
		x += int32(c.buf[c.bytestream]) << 9
	}
	if c.bytestream+1 < c.bytestreamEnd {
		x += int32(c.buf[c.bytestream+1]) << 1
	}
	c.low += x << i
	if c.bytestream < c.bytestreamEnd {
		c.bytestream += cabacBits / 8
	}
}

func (c *cabacContext) getCABAC(state *uint8) int {
	s := int32(*state)
	rangeLPS := int32(h264CABACTables[h264LPSRangeOffset+2*int(c.rng&0xc0)+int(s)])
	bit, lpsMask := int32(0), int32(0)

	c.rng -= rangeLPS
	lpsMask = int32(uint32((c.rng<<(cabacBits+1))-c.low) >> 31)
	lpsMask = -lpsMask

	c.low -= (c.rng << (cabacBits + 1)) & lpsMask
	c.rng += (rangeLPS - c.rng) & lpsMask

	s ^= lpsMask
	*state = h264CABACTables[h264MLPSStateOffset+128+int(s)]
	bit = s & 1

	lpsMask = int32(h264CABACTables[h264NormShiftOffset+int(c.rng)])
	c.rng <<= lpsMask
	c.low <<= lpsMask
	if c.low&cabacMask == 0 {
		c.refill2()
	}
	return int(bit)
}

func (c *cabacContext) getCABACBypass() int {
	c.low += c.low
	if c.low&cabacMask == 0 {
		c.refill()
	}

	rng := c.rng << (cabacBits + 1)
	if c.low < rng {
		return 0
	}
	c.low -= rng
	return 1
}

func (c *cabacContext) getCABACBypassSign(val int32) int32 {
	c.low += c.low
	if c.low&cabacMask == 0 {
		c.refill()
	}

	rng := c.rng << (cabacBits + 1)
	c.low -= rng
	mask := c.low >> 31
	rng &= mask
	c.low += rng
	return (val ^ mask) - mask
}

func (c *cabacContext) getCABACTerminate() int {
	c.rng -= 2
	if c.low < c.rng<<(cabacBits+1) {
		c.renormCABACDecoderOnce()
		return 0
	}
	return c.bytestream - c.bytestreamStart
}

func (c *cabacContext) readIntraPCMBytes(n int) ([]byte, error) {
	if c == nil || n < 0 || c.bytestreamEnd < 0 || c.bytestreamEnd > len(c.buf) {
		return nil, ErrInvalidData
	}

	ptr := c.bytestream
	if c.low&0x1 != 0 {
		ptr--
	}
	if cabacBits == 16 && c.low&0x1ff != 0 {
		ptr--
	}
	if ptr < 0 || ptr > c.bytestreamEnd || n > c.bytestreamEnd-ptr {
		return nil, ErrInvalidData
	}

	pcm := c.buf[ptr : ptr+n]
	next := ptr + n
	nextCABAC, err := initCABACDecoder(c.buf[next:c.bytestreamEnd])
	if err != nil {
		return nil, err
	}
	*c = nextCABAC
	return pcm, nil
}

func (c *cabacContext) renormCABACDecoderOnce() {
	shift := int32(uint32(c.rng-0x100) >> 31)
	c.rng <<= shift
	c.low <<= shift
	if c.low&cabacMask == 0 {
		c.refill()
	}
}

type h264CABACStateTemplateSet struct {
	I  [52][1024]uint8
	PB [3][52][1024]uint8
}

var h264CABACStateTemplates = func() h264CABACStateTemplateSet {
	var templates h264CABACStateTemplateSet
	for qp := int32(0); qp <= 51; qp++ {
		templates.I[qp] = h264CABACStateTemplate(&h264CABACContextInitI, qp)
		for initIDC := range templates.PB {
			templates.PB[initIDC][qp] = h264CABACStateTemplate(&h264CABACContextInitPB[initIDC], qp)
		}
	}
	return templates
}()

func h264CABACStateTemplate(tab *[1024][2]int8, sliceQP int32) [1024]uint8 {
	var states [1024]uint8
	for i := range states {
		pre := 2*(((int32(tab[i][0])*sliceQP)>>4)+int32(tab[i][1])) - 127
		pre ^= pre >> 31
		if pre > 124 {
			pre = 124 + (pre & 1)
		}
		states[i] = uint8(pre)
	}
	return states
}

func initH264CABACStates(sliceTypeNoS int32, cabacInitIDC uint32, qscale int32, bitDepthLuma int32) ([1024]uint8, error) {
	sliceQP := clipInt32(qscale-6*(bitDepthLuma-8), 0, 51)
	if sliceTypeNoS == PictureTypeI {
		return h264CABACStateTemplates.I[sliceQP], nil
	}
	if cabacInitIDC >= 3 {
		return [1024]uint8{}, ErrInvalidData
	}
	return h264CABACStateTemplates.PB[cabacInitIDC][sliceQP], nil
}
