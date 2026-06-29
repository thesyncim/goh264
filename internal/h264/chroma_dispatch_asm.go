// SPDX-License-Identifier: LGPL-2.1-or-later
//go:build !purego && (amd64 || arm64)

package h264

func h264ChromaMCStridesKernel(dst []uint8, src []uint8, dstStride int, srcStride int, height int32, x int32, y int32, width int32, avg bool) {
	if height > 0 {
		if x == 0 && y == 0 {
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
		switch width {
		case 8, 4, 2, 1:
			a := (8 - x) * (8 - y)
			b := x * (8 - y)
			c := (8 - x) * y
			d := x * y
			step := 0
			if d == 0 {
				step = 1
				if c != 0 {
					step = srcStride
				}
			}
			avgFlag := int32(0)
			if avg {
				avgFlag = 1
			}
			h264ChromaMCXYASM(&dst[0], &src[0], dstStride, srcStride, height, width, a, b, c, d, step, avgFlag)
			return
		}
	}
	h264ChromaMCStridesScalar(dst, src, dstStride, srcStride, int(height), int(x), int(y), int(width), avg)
}

//go:noescape
func h264ChromaMCXYASM(dst *uint8, src *uint8, dstStride int, srcStride int, height int32, width int32, a int32, b int32, c int32, d int32, step int, avg int32)

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
