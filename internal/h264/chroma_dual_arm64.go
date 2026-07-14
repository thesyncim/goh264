// SPDX-License-Identifier: LGPL-2.1-or-later
//go:build !purego && arm64

package h264

func h264ChromaMCDualStridesKernel(dstCb []uint8, dstCr []uint8, srcCb []uint8, srcCr []uint8, dstStride int, srcStride int, height int32, x int32, y int32, width int32, avg bool) {
	if height > 0 && (x != 0 || y != 0) && (width == 8 && height >= 2 && height&1 == 0 || width == 4 || width == 2) {
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
		h264ChromaMCDualXYASM(&dstCb[0], &dstCr[0], &srcCb[0], &srcCr[0], dstStride, srcStride, height, width, a, b, c, d, step, avgFlag)
		return
	}
	h264ChromaMCStridesKernel(dstCb, srcCb, dstStride, srcStride, height, x, y, width, avg)
	h264ChromaMCStridesKernel(dstCr, srcCr, dstStride, srcStride, height, x, y, width, avg)
}

//go:noescape
func h264ChromaMCDualXYASM(dstCb *uint8, dstCr *uint8, srcCb *uint8, srcCr *uint8, dstStride int, srcStride int, height int32, width int32, a int32, b int32, c int32, d int32, step int, avg int32)
