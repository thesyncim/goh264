// SPDX-License-Identifier: LGPL-2.1-or-later
//go:build purego || (!amd64 && !arm64)

package h264

func h264ChromaMCStridesKernel(dst []uint8, src []uint8, dstStride int, srcStride int, height int32, x int32, y int32, width int32, avg bool) {
	h264ChromaMCStridesScalar(dst, src, dstStride, srcStride, int(height), int(x), int(y), int(width), avg)
}
