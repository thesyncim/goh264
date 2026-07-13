// SPDX-License-Identifier: LGPL-2.1-or-later
//
// Source-shaped port of the CABAC macroblock syntax helpers from FFmpeg n8.0.1
// libavcodec/h264_cabac.c. These helpers stop at syntax decisions and decoded
// deltas; prediction, reference-list side effects, and reconstruction remain in
// the pending h264_mvpred/h264_refs/h264_mb layers.

package h264

import "unsafe"

type cabacSyntaxSource interface {
	get(idx int) int
	bypass() int
	bypassSign(val int32) int32
	terminate() int
}

type cabacIntraPCMSource interface {
	cabacSyntaxSource
	intraPCMBytes(n int) ([]byte, error)
}

type cabacSyntaxDecoder struct {
	cabac *cabacContext
	state *[1024]uint8
}

func (d *cabacSyntaxDecoder) get(idx int) int {
	c := d.cabac
	// Syntax-context indexes are fixed by the H.264 tables and stay within the
	// 1024-byte state array. The primitive differential test covers this seam.
	state := (*uint8)(unsafe.Add(unsafe.Pointer(d.state), uintptr(idx)))
	s := int32(*state)
	rng := c.rng
	low := c.low
	rangeLPS := int32(h264CABACTableUnchecked(h264LPSRangeOffset + 2*int(rng&0xc0) + int(s)))

	rng -= rangeLPS
	lpsMask := -int32(uint32((rng<<(cabacBits+1))-low) >> 31)
	low -= (rng << (cabacBits + 1)) & lpsMask
	rng += (rangeLPS - rng) & lpsMask

	s ^= lpsMask
	*state = h264CABACTableUnchecked(h264MLPSStateOffset + 128 + int(s))
	bit := s & 1

	// The pinned norm-shift table is 0..9. Masking that invariant into the
	// operand lets arm64 use its native register shift directly; an unconstrained
	// Go shift otherwise grows two >=64 guards in this per-bin primitive.
	shift := uint32(h264CABACTableUnchecked(h264NormShiftOffset+int(rng))) & 31
	rng = int32(uint32(rng) << shift)
	low = int32(uint32(low) << shift)
	c.rng = rng
	c.low = low
	if low&cabacMask == 0 {
		c.refill2()
	}
	return int(bit)
}

func h264CABACTableUnchecked(index int) uint8 {
	return *(*uint8)(unsafe.Add(unsafe.Pointer(&h264CABACTables[0]), uintptr(index)))
}

func (d *cabacSyntaxDecoder) bypass() int {
	// See TestCABACSyntaxDecoderPrimitivesMatchContext for differential coverage
	// against the shared CABAC primitive.
	c := d.cabac
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

func (d *cabacSyntaxDecoder) bypassSign(val int32) int32 {
	c := d.cabac
	c.low += c.low
	if c.low&cabacMask == 0 {
		c.refill()
	}

	rng := c.rng << (cabacBits + 1)
	mask := (c.low - rng) >> 31
	c.low -= rng & ^mask
	return (val ^ mask) - mask
}

func (d *cabacSyntaxDecoder) terminate() int {
	return d.cabac.getCABACTerminate()
}

func (d *cabacSyntaxDecoder) intraPCMBytes(n int) ([]byte, error) {
	if d.cabac == nil {
		return nil, ErrInvalidData
	}
	return d.cabac.readIntraPCMBytes(n)
}

func decodeCABACMBType[S cabacSyntaxSource](src S, sliceType int32, sliceTypeNoS int32, leftType uint32, topType uint32) (cavlcMacroblockSyntax, error) {
	var mb cavlcMacroblockSyntax
	var raw int

	if sliceTypeNoS == PictureTypeB {
		ctx := 0
		if !isDirect(leftType - 1) {
			ctx++
		}
		if !isDirect(topType - 1) {
			ctx++
		}

		if src.get(27+ctx) == 0 {
			raw = 0
		} else if src.get(27+3) == 0 {
			raw = 1 + src.get(27+5)
		} else {
			bits := src.get(27+4) << 3
			bits += src.get(27+5) << 2
			bits += src.get(27+5) << 1
			bits += src.get(27 + 5)
			if bits < 8 {
				raw = bits + 3
			} else if bits == 13 {
				raw = decodeCABACIntraMBType(src, 32, false, leftType, topType)
				return cabacIntraMBTypeInfo(raw)
			} else if bits == 14 {
				raw = 11
			} else if bits == 15 {
				raw = 22
			} else {
				bits = (bits << 1) + src.get(27+5)
				raw = bits - 4
			}
		}
		info := h264BMBTypeInfo[raw]
		mb.MBType = info.Type
		mb.PartitionCount = info.PartitionCount
		return mb, nil
	}

	if sliceTypeNoS == PictureTypeP {
		if src.get(14) == 0 {
			if src.get(15) == 0 {
				raw = 3 * src.get(16)
			} else {
				raw = 2 - src.get(17)
			}
			info := h264PMBTypeInfo[raw]
			mb.MBType = info.Type
			mb.PartitionCount = info.PartitionCount
			return mb, nil
		}
		raw = decodeCABACIntraMBType(src, 17, false, leftType, topType)
		return cabacIntraMBTypeInfo(raw)
	}

	if sliceTypeNoS != PictureTypeI {
		return mb, ErrInvalidData
	}
	raw = decodeCABACIntraMBType(src, 3, true, leftType, topType)
	if sliceType == PictureTypeSI && raw != 0 {
		raw--
	}
	return cabacIntraMBTypeInfo(raw)
}

func decodeCABACMBTypeForSource(src cabacSyntaxSource, sliceType int32, sliceTypeNoS int32, leftType uint32, topType uint32) (cavlcMacroblockSyntax, error) {
	if dec, ok := src.(*cabacSyntaxDecoder); ok {
		return decodeCABACMBType(dec, sliceType, sliceTypeNoS, leftType, topType)
	}
	return decodeCABACMBType(src, sliceType, sliceTypeNoS, leftType, topType)
}

func decodeCABACIntraMBType[S cabacSyntaxSource](src S, ctxBase int, intraSlice bool, leftType uint32, topType uint32) int {
	state := ctxBase
	if intraSlice {
		ctx := 0
		if leftType&(MBTypeIntra16x16|MBTypeIntraPCM) != 0 {
			ctx++
		}
		if topType&(MBTypeIntra16x16|MBTypeIntraPCM) != 0 {
			ctx++
		}
		if src.get(state+ctx) == 0 {
			return 0
		}
		state += 2
	} else if src.get(state) == 0 {
		return 0
	}

	if src.terminate() != 0 {
		return 25
	}

	mbType := 1
	mbType += 12 * src.get(state+1)
	if src.get(state+2) != 0 {
		chromaCtx := 2
		if intraSlice {
			chromaCtx++
		}
		mbType += 4 + 4*src.get(state+chromaCtx)
	}
	predCtx0 := 3
	predCtx1 := 3
	if intraSlice {
		predCtx0++
		predCtx1 += 2
	}
	mbType += 2 * src.get(state+predCtx0)
	mbType += src.get(state + predCtx1)
	return mbType
}

func cabacIntraMBTypeInfo(raw int) (cavlcMacroblockSyntax, error) {
	var mb cavlcMacroblockSyntax
	if raw < 0 || raw >= len(h264IMBTypeInfo) {
		return mb, ErrInvalidData
	}
	info := h264IMBTypeInfo[raw]
	mb.MBType = info.Type
	mb.Intra16x16PredMode = info.PredMode
	mb.CBP = int(info.CBP)
	return mb, nil
}

func decodeCABACMBIntra4x4PredMode[S cabacSyntaxSource](src S, predMode int) int {
	if src.get(68) != 0 {
		return predMode
	}

	mode := src.get(69)
	mode += 2 * src.get(69)
	mode += 4 * src.get(69)
	if mode >= predMode {
		mode++
	}
	return mode
}

func decodeCABACMBIntra4x4PredModeForSource(src cabacSyntaxSource, predMode int) int {
	if dec, ok := src.(*cabacSyntaxDecoder); ok {
		return decodeCABACMBIntra4x4PredMode(dec, predMode)
	}
	return decodeCABACMBIntra4x4PredMode(src, predMode)
}

func decodeCABACMBCBPLuma[S cabacSyntaxSource](src S, leftCBP int, topCBP int) int {
	cbp := 0

	ctx := boolToInt(leftCBP&0x02 == 0) + 2*boolToInt(topCBP&0x04 == 0)
	cbp += src.get(73 + ctx)
	ctx = boolToInt(cbp&0x01 == 0) + 2*boolToInt(topCBP&0x08 == 0)
	cbp += src.get(73+ctx) << 1
	ctx = boolToInt(leftCBP&0x08 == 0) + 2*boolToInt(cbp&0x01 == 0)
	cbp += src.get(73+ctx) << 2
	ctx = boolToInt(cbp&0x04 == 0) + 2*boolToInt(cbp&0x02 == 0)
	cbp += src.get(73+ctx) << 3

	return cbp
}

func decodeCABACMBCBPLumaForSource(src cabacSyntaxSource, leftCBP int, topCBP int) int {
	if dec, ok := src.(*cabacSyntaxDecoder); ok {
		return decodeCABACMBCBPLuma(dec, leftCBP, topCBP)
	}
	return decodeCABACMBCBPLuma(src, leftCBP, topCBP)
}

func decodeCABACMBCBPChroma[S cabacSyntaxSource](src S, leftCBP int, topCBP int) int {
	cbpA := (leftCBP >> 4) & 0x03
	cbpB := (topCBP >> 4) & 0x03

	ctx := 0
	if cbpA > 0 {
		ctx++
	}
	if cbpB > 0 {
		ctx += 2
	}
	if src.get(77+ctx) == 0 {
		return 0
	}

	ctx = 4
	if cbpA == 2 {
		ctx++
	}
	if cbpB == 2 {
		ctx += 2
	}
	return 1 + src.get(77+ctx)
}

func decodeCABACMBCBPChromaForSource(src cabacSyntaxSource, leftCBP int, topCBP int) int {
	if dec, ok := src.(*cabacSyntaxDecoder); ok {
		return decodeCABACMBCBPChroma(dec, leftCBP, topCBP)
	}
	return decodeCABACMBCBPChroma(src, leftCBP, topCBP)
}

func decodeCABACPSubMBType[S cabacSyntaxSource](src S) (int, PMBInfo) {
	raw := 3
	if src.get(21) != 0 {
		raw = 0
	} else if src.get(22) == 0 {
		raw = 1
	} else if src.get(23) != 0 {
		raw = 2
	}
	return raw, h264PSubMBTypeInfo[raw]
}

func decodeCABACBSubMBType[S cabacSyntaxSource](src S) (int, PMBInfo) {
	if src.get(36) == 0 {
		return 0, h264BSubMBTypeInfo[0]
	}
	if src.get(37) == 0 {
		raw := 1 + src.get(39)
		return raw, h264BSubMBTypeInfo[raw]
	}

	raw := 3
	if src.get(38) != 0 {
		if src.get(39) != 0 {
			raw = 11 + src.get(39)
			return raw, h264BSubMBTypeInfo[raw]
		}
		raw += 4
	}
	raw += 2 * src.get(39)
	raw += src.get(39)
	return raw, h264BSubMBTypeInfo[raw]
}

func decodeCABACMBRef[S cabacSyntaxSource](src S, sliceTypeNoS int32, refA int32, refB int32, directA uint32, directB uint32) int32 {
	ref := int32(0)
	ctx := 0

	if sliceTypeNoS == PictureTypeB {
		if refA > 0 && directA&(MBTypeDirect2>>1) == 0 {
			ctx++
		}
		if refB > 0 && directB&(MBTypeDirect2>>1) == 0 {
			ctx += 2
		}
	} else {
		if refA > 0 {
			ctx++
		}
		if refB > 0 {
			ctx += 2
		}
	}

	for src.get(54+ctx) != 0 {
		ref++
		ctx = (ctx >> 2) + 4
		if ref >= 32 {
			return -1
		}
	}
	return ref
}

func decodeCABACMBRefForSource(src cabacSyntaxSource, sliceTypeNoS int32, refA int32, refB int32, directA uint32, directB uint32) int32 {
	if dec, ok := src.(*cabacSyntaxDecoder); ok {
		return decodeCABACMBRef(dec, sliceTypeNoS, refA, refB, directA, directB)
	}
	return decodeCABACMBRef(src, sliceTypeNoS, refA, refB, directA, directB)
}

func decodeCABACMBMVD[S cabacSyntaxSource](src S, ctxBase int, amvd int) (int32, int, error) {
	if src.get(cabacMVDContext(ctxBase, amvd)) == 0 {
		return 0, 0, nil
	}

	mvd := 1
	ctxBase += 3
	for mvd < 9 && src.get(ctxBase) != 0 {
		if mvd < 4 {
			ctxBase++
		}
		mvd++
	}

	mvda := mvd
	if mvd >= 9 {
		k := 3
		for src.bypass() != 0 {
			mvd += 1 << k
			k++
			if k > 24 {
				return 0, 0, ErrInvalidData
			}
		}
		for k > 0 {
			k--
			mvd += src.bypass() << k
		}
		mvda = mvd
		if mvda >= 70 {
			mvda = 70
		}
	}
	return src.bypassSign(int32(-mvd)), mvda, nil
}

// decodeCABACMBMVDDecoder keeps range/low in registers across the complete
// production MVD component. Scripted sources continue to use decodeCABACMBMVD.
func decodeCABACMBMVDDecoder(src *cabacSyntaxDecoder, ctxBase int, amvd int) (int32, int, error) {
	c := src.cabac
	states := src.state
	low := c.low
	rng := c.rng
	ctx := cabacMVDContext(ctxBase, amvd)
	mvd := 0
	for {
		state := (*uint8)(unsafe.Add(unsafe.Pointer(states), uintptr(ctx)))
		s := int32(*state)
		rangeLPS := int32(h264CABACTableUnchecked(h264LPSRangeOffset + 2*int(rng&0xc0) + int(s)))
		rng -= rangeLPS
		lpsMask := -int32(uint32((rng<<(cabacBits+1))-low) >> 31)
		low -= (rng << (cabacBits + 1)) & lpsMask
		rng += (rangeLPS - rng) & lpsMask
		s ^= lpsMask
		*state = h264CABACTableUnchecked(h264MLPSStateOffset + 128 + int(s))
		shift := uint32(h264CABACTableUnchecked(h264NormShiftOffset+int(rng))) & 31
		rng = int32(uint32(rng) << shift)
		low = int32(uint32(low) << shift)
		if low&cabacMask == 0 {
			c.low = low
			c.rng = rng
			c.refill2()
			low = c.low
		}
		if s&1 == 0 {
			break
		}
		if mvd == 0 {
			mvd = 1
			ctx = ctxBase + 3
			continue
		}
		if mvd >= 8 {
			mvd = 9
			break
		}
		if mvd < 4 {
			ctx++
		}
		mvd++
	}
	if mvd == 0 {
		c.low = low
		c.rng = rng
		return 0, 0, nil
	}

	mvda := mvd
	if mvd >= 9 {
		c.low = low
		c.rng = rng
		k := 3
		for src.bypass() != 0 {
			mvd += 1 << k
			k++
			if k > 24 {
				return 0, 0, ErrInvalidData
			}
		}
		for k > 0 {
			k--
			mvd += src.bypass() << k
		}
		mvda = mvd
		if mvda >= 70 {
			mvda = 70
		}
		low = c.low
		rng = c.rng
	}

	low += low
	if low&cabacMask == 0 {
		c.low = low
		c.rng = rng
		c.refill()
		low = c.low
	}
	rangeValue := rng << (cabacBits + 1)
	mask := (low - rangeValue) >> 31
	low -= rangeValue &^ mask
	value := (int32(-mvd) ^ mask) - mask
	c.low = low
	c.rng = rng
	return value, mvda, nil
}

func decodeCABACQScaleDiff[S cabacSyntaxSource](src S, qscale int, lastQScaleDiff int, maxQP int) (int, int, error) {
	ctx := 0
	if lastQScaleDiff != 0 {
		ctx = 1
	}
	if src.get(60+ctx) == 0 {
		return qscale, 0, nil
	}

	val := 1
	ctx = 2
	for src.get(60+ctx) != 0 {
		ctx = 3
		val++
		if val > 2*maxQP {
			return qscale, lastQScaleDiff, ErrInvalidData
		}
	}

	diff := 0
	if val&1 != 0 {
		diff = (val + 1) >> 1
	} else {
		diff = -((val + 1) >> 1)
	}
	qscale += diff
	if uint32(qscale) > uint32(maxQP) {
		if qscale < 0 {
			qscale += maxQP + 1
		} else {
			qscale -= maxQP + 1
		}
	}
	return qscale, diff, nil
}

func decodeCABACQScaleDiffForSource(src cabacSyntaxSource, qscale int, lastQScaleDiff int, maxQP int) (int, int, error) {
	if dec, ok := src.(*cabacSyntaxDecoder); ok {
		return decodeCABACQScaleDiff(dec, qscale, lastQScaleDiff, maxQP)
	}
	return decodeCABACQScaleDiff(src, qscale, lastQScaleDiff, maxQP)
}

func cabacMVDContext(ctxBase int, amvd int) int {
	return ctxBase + int(int32(amvd-3)>>31) + int(int32(amvd-33)>>31) + 2
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
