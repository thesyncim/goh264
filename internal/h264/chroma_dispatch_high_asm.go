// SPDX-License-Identifier: LGPL-2.1-or-later
//go:build !purego && (amd64 || arm64)

package h264

import "unsafe"

func h264ChromaMCStridesHighKernel(dst []uint16, src []uint16, dstStride int, srcStride int, height int32, x int32, y int32, width int32, avg bool) {
	if height > 0 {
		a := (8 - x) * (8 - y)
		b := x * (8 - y)
		c := (8 - x) * y
		d := x * y
		dstStrideBytes := dstStride * 2
		srcStrideBytes := srcStride * 2
		step := 0
		if d == 0 && b+c != 0 {
			step = 2
			if c != 0 {
				step = srcStrideBytes
			}
		}
		avgFlag := int32(0)
		if avg {
			avgFlag = 1
		}
		h264ChromaMCHighASM(
			(*uint8)(unsafe.Pointer(&dst[0])),
			(*uint8)(unsafe.Pointer(&src[0])),
			dstStrideBytes,
			srcStrideBytes,
			height,
			width,
			a,
			b,
			c,
			d,
			step,
			avgFlag,
		)
		return
	}
	h264ChromaMCStridesHighScalar(dst, src, dstStride, srcStride, int(height), int(x), int(y), int(width), avg)
}

//go:noescape
func h264ChromaMCHighASM(dst *uint8, src *uint8, dstStride int, srcStride int, height int32, width int32, a int32, b int32, c int32, d int32, step int, avg int32)
