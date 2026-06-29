// SPDX-License-Identifier: LGPL-2.1-or-later
//go:build !purego && (amd64 || arm64)

package h264

import "unsafe"

func h264QpelMCStridesHighKernel(dst []uint16, dstOffset int, dstStride int, src []uint16, srcOffset int, srcStride int, size int32, mx int32, my int32, avg bool, bitDepth int32) {
	avgFlag := int32(0)
	if avg {
		avgFlag = 1
	}
	if mx == 0 && my == 0 {
		h264QpelMCHigh00ASM(
			(*uint8)(unsafe.Pointer(&dst[dstOffset])),
			(*uint8)(unsafe.Pointer(&src[srcOffset])),
			dstStride*2,
			srcStride*2,
			size,
			avgFlag,
		)
		return
	}
	max := int32((1 << uint(bitDepth)) - 1)
	if my == 0 {
		h264QpelMCHighX0ASM(
			(*uint8)(unsafe.Pointer(&dst[dstOffset])),
			(*uint8)(unsafe.Pointer(&src[srcOffset])),
			dstStride*2,
			srcStride*2,
			size,
			mx,
			max,
			avgFlag,
		)
		return
	}
	if mx == 0 {
		h264QpelMCHigh0YASM(
			(*uint8)(unsafe.Pointer(&dst[dstOffset])),
			(*uint8)(unsafe.Pointer(&src[srcOffset])),
			dstStride*2,
			srcStride*2,
			size,
			my,
			max,
			avgFlag,
		)
		return
	}
	if mx == 2 && my == 2 {
		h264QpelMCHigh22ASM(
			(*uint8)(unsafe.Pointer(&dst[dstOffset])),
			(*uint8)(unsafe.Pointer(&src[srcOffset])),
			dstStride*2,
			srcStride*2,
			size,
			max,
			avgFlag,
		)
		return
	}
	h264QpelMCStridesHighScalar(dst, dstOffset, dstStride, src, srcOffset, srcStride, int(size), int(mx), int(my), avg, int(bitDepth))
}

//go:noescape
func h264QpelMCHigh00ASM(dst *uint8, src *uint8, dstStride int, srcStride int, size int32, avg int32)

//go:noescape
func h264QpelMCHighX0ASM(dst *uint8, src *uint8, dstStride int, srcStride int, size int32, mx int32, max int32, avg int32)

//go:noescape
func h264QpelMCHigh0YASM(dst *uint8, src *uint8, dstStride int, srcStride int, size int32, my int32, max int32, avg int32)

//go:noescape
func h264QpelMCHigh22ASM(dst *uint8, src *uint8, dstStride int, srcStride int, size int32, max int32, avg int32)
