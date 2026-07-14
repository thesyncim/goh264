// SPDX-License-Identifier: LGPL-2.1-or-later
//go:build purego || !arm64

package h264

func h264ChromaMCDualStridesKernel(dstCb []uint8, dstCr []uint8, srcCb []uint8, srcCr []uint8, dstStride int, srcStride int, height int32, x int32, y int32, width int32, avg bool) {
	h264ChromaMCStridesKernel(dstCb, srcCb, dstStride, srcStride, height, x, y, width, avg)
	h264ChromaMCStridesKernel(dstCr, srcCr, dstStride, srcStride, height, x, y, width, avg)
}
