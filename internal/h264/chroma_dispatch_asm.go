// SPDX-License-Identifier: LGPL-2.1-or-later
//go:build !purego && (amd64 || arm64)

package h264

func h264ChromaMCStridesKernel(dst []uint8, src []uint8, dstStride int, srcStride int, height int32, x int32, y int32, width int32, avg bool) {
	if width == 8 && x == 0 && y == 0 && height > 0 {
		if avg {
			h264ChromaMC8Avg00ASM(&dst[0], &src[0], dstStride, srcStride, height)
			return
		}
		h264ChromaMC8Put00ASM(&dst[0], &src[0], dstStride, srcStride, height)
		return
	}
	h264ChromaMCStridesScalar(dst, src, dstStride, srcStride, int(height), int(x), int(y), int(width), avg)
}

func h264ChromaMC8Put00ASM(dst *uint8, src *uint8, dstStride int, srcStride int, height int32)

func h264ChromaMC8Avg00ASM(dst *uint8, src *uint8, dstStride int, srcStride int, height int32)
