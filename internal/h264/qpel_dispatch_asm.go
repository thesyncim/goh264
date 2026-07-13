// SPDX-License-Identifier: LGPL-2.1-or-later
//go:build !purego && (amd64 || arm64)

package h264

//go:noescape
func h264QpelMC16Put00ASM(dst *uint8, src *uint8, dstStride int, srcStride int)

//go:noescape
func h264QpelMC16Avg00ASM(dst *uint8, src *uint8, dstStride int, srcStride int)

//go:noescape
func h264QpelMC8Put00ASM(dst *uint8, src *uint8, dstStride int, srcStride int)

//go:noescape
func h264QpelMC8Avg00ASM(dst *uint8, src *uint8, dstStride int, srcStride int)

//go:noescape
func h264QpelMC16Put10ASM(dst *uint8, src *uint8, dstStride int, srcStride int)

//go:noescape
func h264QpelMC16Avg10ASM(dst *uint8, src *uint8, dstStride int, srcStride int)

//go:noescape
func h264QpelMC8Put10ASM(dst *uint8, src *uint8, dstStride int, srcStride int)

//go:noescape
func h264QpelMC8Avg10ASM(dst *uint8, src *uint8, dstStride int, srcStride int)

//go:noescape
func h264QpelMC16Put20ASM(dst *uint8, src *uint8, dstStride int, srcStride int)

//go:noescape
func h264QpelMC16Avg20ASM(dst *uint8, src *uint8, dstStride int, srcStride int)

//go:noescape
func h264QpelMC8Put20ASM(dst *uint8, src *uint8, dstStride int, srcStride int)

//go:noescape
func h264QpelMC8Avg20ASM(dst *uint8, src *uint8, dstStride int, srcStride int)

//go:noescape
func h264QpelMC16Put30ASM(dst *uint8, src *uint8, dstStride int, srcStride int)

//go:noescape
func h264QpelMC16Avg30ASM(dst *uint8, src *uint8, dstStride int, srcStride int)

//go:noescape
func h264QpelMC8Put30ASM(dst *uint8, src *uint8, dstStride int, srcStride int)

//go:noescape
func h264QpelMC8Avg30ASM(dst *uint8, src *uint8, dstStride int, srcStride int)

//go:noescape
func h264QpelMCPutX0ASM(dst *uint8, src *uint8, dstStride int, srcStride int, size int32, mx int32)

//go:noescape
func h264QpelMCAvgX0ASM(dst *uint8, src *uint8, dstStride int, srcStride int, size int32, mx int32)

//go:noescape
func h264QpelMCPut0YASM(dst *uint8, src *uint8, dstStride int, srcStride int, size int32, my int32)

//go:noescape
func h264QpelMCAvg0YASM(dst *uint8, src *uint8, dstStride int, srcStride int, size int32, my int32)

//go:noescape
func h264QpelMCPut22ASM(dst *uint8, src *uint8, dstStride int, srcStride int, size int32)

//go:noescape
func h264QpelMCAvg22ASM(dst *uint8, src *uint8, dstStride int, srcStride int, size int32)

//go:noescape
func h264QpelMCPutHVXYASM(dst *uint8, src *uint8, dstStride int, srcStride int, size int32, mx int32, my int32)

//go:noescape
func h264QpelMCAvgHVXYASM(dst *uint8, src *uint8, dstStride int, srcStride int, size int32, mx int32, my int32)

//go:noescape
func h264QpelMCPutHVBlendASM(dst *uint8, src *uint8, dstStride int, srcStride int, size int32, mx int32, my int32)

//go:noescape
func h264QpelMCAvgHVBlendASM(dst *uint8, src *uint8, dstStride int, srcStride int, size int32, mx int32, my int32)

//go:noescape
func h264QpelMC4Put00ASM(dst *uint8, src *uint8, dstStride int, srcStride int)

//go:noescape
func h264QpelMC4Avg00ASM(dst *uint8, src *uint8, dstStride int, srcStride int)

//go:noescape
func h264QpelMC2Put00ASM(dst *uint8, src *uint8, dstStride int, srcStride int)

//go:noescape
func h264QpelMC2Avg00ASM(dst *uint8, src *uint8, dstStride int, srcStride int)

func h264QpelMCStridesKernel(dst []uint8, dstOffset int, dstStride int, src []uint8, srcOffset int, srcStride int, size int32, mx int32, my int32, avg bool) {
	if uint32(mx-1) < 3 && uint32(my-1) < 3 {
		switch size {
		case 16, 8, 4, 2:
			dstPtr := &dst[dstOffset]
			srcPtr := &src[srcOffset]
			if mx == 2 && my == 2 {
				if avg {
					h264QpelMCAvg22ASM(dstPtr, srcPtr, dstStride, srcStride, size)
				} else {
					h264QpelMCPut22ASM(dstPtr, srcPtr, dstStride, srcStride, size)
				}
			} else if mx == 2 || my == 2 {
				if avg {
					h264QpelMCAvgHVBlendASM(dstPtr, srcPtr, dstStride, srcStride, size, mx, my)
				} else {
					h264QpelMCPutHVBlendASM(dstPtr, srcPtr, dstStride, srcStride, size, mx, my)
				}
			} else if avg {
				h264QpelMCAvgHVXYASM(dstPtr, srcPtr, dstStride, srcStride, size, mx, my)
			} else {
				h264QpelMCPutHVXYASM(dstPtr, srcPtr, dstStride, srcStride, size, mx, my)
			}
			return
		default:
			h264QpelMCStridesScalar(dst, dstOffset, dstStride, src, srcOffset, srcStride, int(size), int(mx), int(my), avg)
			return
		}
	}
	if mx == 0 && my == 0 {
		dstPtr := &dst[dstOffset]
		srcPtr := &src[srcOffset]
		if avg {
			switch size {
			case 16:
				h264QpelMC16Avg00ASM(dstPtr, srcPtr, dstStride, srcStride)
			case 8:
				h264QpelMC8Avg00ASM(dstPtr, srcPtr, dstStride, srcStride)
			case 4:
				h264QpelMC4Avg00ASM(dstPtr, srcPtr, dstStride, srcStride)
			case 2:
				h264QpelMC2Avg00ASM(dstPtr, srcPtr, dstStride, srcStride)
			default:
				h264QpelMCStridesScalar(dst, dstOffset, dstStride, src, srcOffset, srcStride, int(size), int(mx), int(my), avg)
			}
			return
		}
		switch size {
		case 16:
			h264QpelMC16Put00ASM(dstPtr, srcPtr, dstStride, srcStride)
		case 8:
			h264QpelMC8Put00ASM(dstPtr, srcPtr, dstStride, srcStride)
		case 4:
			h264QpelMC4Put00ASM(dstPtr, srcPtr, dstStride, srcStride)
		case 2:
			h264QpelMC2Put00ASM(dstPtr, srcPtr, dstStride, srcStride)
		default:
			h264QpelMCStridesScalar(dst, dstOffset, dstStride, src, srcOffset, srcStride, int(size), int(mx), int(my), avg)
		}
		return
	}
	if my == 0 && (mx == 1 || mx == 3) {
		dstPtr := &dst[dstOffset]
		srcPtr := &src[srcOffset]
		if mx == 1 {
			if avg {
				switch size {
				case 16:
					h264QpelMC16Avg10ASM(dstPtr, srcPtr, dstStride, srcStride)
				case 8:
					h264QpelMC8Avg10ASM(dstPtr, srcPtr, dstStride, srcStride)
				case 4, 2:
					h264QpelMCAvgX0ASM(dstPtr, srcPtr, dstStride, srcStride, size, mx)
				default:
					h264QpelMCStridesScalar(dst, dstOffset, dstStride, src, srcOffset, srcStride, int(size), int(mx), int(my), avg)
				}
				return
			}
			switch size {
			case 16:
				h264QpelMC16Put10ASM(dstPtr, srcPtr, dstStride, srcStride)
			case 8:
				h264QpelMC8Put10ASM(dstPtr, srcPtr, dstStride, srcStride)
			case 4, 2:
				h264QpelMCPutX0ASM(dstPtr, srcPtr, dstStride, srcStride, size, mx)
			default:
				h264QpelMCStridesScalar(dst, dstOffset, dstStride, src, srcOffset, srcStride, int(size), int(mx), int(my), avg)
			}
			return
		}
		if avg {
			switch size {
			case 16:
				h264QpelMC16Avg30ASM(dstPtr, srcPtr, dstStride, srcStride)
			case 8:
				h264QpelMC8Avg30ASM(dstPtr, srcPtr, dstStride, srcStride)
			case 4, 2:
				h264QpelMCAvgX0ASM(dstPtr, srcPtr, dstStride, srcStride, size, mx)
			default:
				h264QpelMCStridesScalar(dst, dstOffset, dstStride, src, srcOffset, srcStride, int(size), int(mx), int(my), avg)
			}
			return
		}
		switch size {
		case 16:
			h264QpelMC16Put30ASM(dstPtr, srcPtr, dstStride, srcStride)
		case 8:
			h264QpelMC8Put30ASM(dstPtr, srcPtr, dstStride, srcStride)
		case 4, 2:
			h264QpelMCPutX0ASM(dstPtr, srcPtr, dstStride, srcStride, size, mx)
		default:
			h264QpelMCStridesScalar(dst, dstOffset, dstStride, src, srcOffset, srcStride, int(size), int(mx), int(my), avg)
		}
		return
	}
	if mx == 2 && my == 0 {
		dstPtr := &dst[dstOffset]
		srcPtr := &src[srcOffset]
		if avg {
			switch size {
			case 16:
				h264QpelMC16Avg20ASM(dstPtr, srcPtr, dstStride, srcStride)
			case 8:
				h264QpelMC8Avg20ASM(dstPtr, srcPtr, dstStride, srcStride)
			case 4, 2:
				h264QpelMCAvgX0ASM(dstPtr, srcPtr, dstStride, srcStride, size, mx)
			default:
				h264QpelMCStridesScalar(dst, dstOffset, dstStride, src, srcOffset, srcStride, int(size), int(mx), int(my), avg)
			}
			return
		}
		switch size {
		case 16:
			h264QpelMC16Put20ASM(dstPtr, srcPtr, dstStride, srcStride)
		case 8:
			h264QpelMC8Put20ASM(dstPtr, srcPtr, dstStride, srcStride)
		case 4, 2:
			h264QpelMCPutX0ASM(dstPtr, srcPtr, dstStride, srcStride, size, mx)
		default:
			h264QpelMCStridesScalar(dst, dstOffset, dstStride, src, srcOffset, srcStride, int(size), int(mx), int(my), avg)
		}
		return
	}
	if mx == 0 && (my == 1 || my == 2 || my == 3) {
		switch size {
		case 16, 8, 4, 2:
			dstPtr := &dst[dstOffset]
			srcPtr := &src[srcOffset]
			if avg {
				h264QpelMCAvg0YASM(dstPtr, srcPtr, dstStride, srcStride, size, my)
			} else {
				h264QpelMCPut0YASM(dstPtr, srcPtr, dstStride, srcStride, size, my)
			}
			return
		default:
			h264QpelMCStridesScalar(dst, dstOffset, dstStride, src, srcOffset, srcStride, int(size), int(mx), int(my), avg)
			return
		}
	}
	h264QpelMCStridesScalar(dst, dstOffset, dstStride, src, srcOffset, srcStride, int(size), int(mx), int(my), avg)
}
