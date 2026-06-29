// SPDX-License-Identifier: LGPL-2.1-or-later
//go:build !purego && (amd64 || arm64)

package h264

func h264ChromaMCStridesKernel(dst []uint8, src []uint8, dstStride int, srcStride int, height int32, x int32, y int32, width int32, avg bool) {
	if x == 0 && y == 0 && height > 0 {
		if !avg {
			switch width {
			case 8:
				h264ChromaMC8Put00ASM(&dst[0], &src[0], dstStride, srcStride, height)
				return
			case 4:
				h264ChromaMC4Put00ASM(&dst[0], &src[0], dstStride, srcStride, height)
				return
			case 2:
				h264ChromaMC2Put00ASM(&dst[0], &src[0], dstStride, srcStride, height)
				return
			case 1:
				h264ChromaMC1Put00ASM(&dst[0], &src[0], dstStride, srcStride, height)
				return
			}
		}
		switch width {
		case 8:
			h264ChromaMC8Avg00ASM(&dst[0], &src[0], dstStride, srcStride, height)
			return
		case 4:
			h264ChromaMC4Avg00ASM(&dst[0], &src[0], dstStride, srcStride, height)
			return
		case 2:
			h264ChromaMC2Avg00ASM(&dst[0], &src[0], dstStride, srcStride, height)
			return
		case 1:
			h264ChromaMC1Avg00ASM(&dst[0], &src[0], dstStride, srcStride, height)
			return
		}
	}
	h264ChromaMCStridesScalar(dst, src, dstStride, srcStride, int(height), int(x), int(y), int(width), avg)
}

//go:noescape
func h264ChromaMC8Put00ASM(dst *uint8, src *uint8, dstStride int, srcStride int, height int32)

//go:noescape
func h264ChromaMC8Avg00ASM(dst *uint8, src *uint8, dstStride int, srcStride int, height int32)

//go:noescape
func h264ChromaMC4Put00ASM(dst *uint8, src *uint8, dstStride int, srcStride int, height int32)

//go:noescape
func h264ChromaMC4Avg00ASM(dst *uint8, src *uint8, dstStride int, srcStride int, height int32)

//go:noescape
func h264ChromaMC2Put00ASM(dst *uint8, src *uint8, dstStride int, srcStride int, height int32)

//go:noescape
func h264ChromaMC2Avg00ASM(dst *uint8, src *uint8, dstStride int, srcStride int, height int32)

//go:noescape
func h264ChromaMC1Put00ASM(dst *uint8, src *uint8, dstStride int, srcStride int, height int32)

//go:noescape
func h264ChromaMC1Avg00ASM(dst *uint8, src *uint8, dstStride int, srcStride int, height int32)
