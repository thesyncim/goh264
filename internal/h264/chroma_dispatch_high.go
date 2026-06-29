// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

func h264ChromaMCStridesHighKernel(dst []uint16, src []uint16, dstStride int, srcStride int, height int32, x int32, y int32, width int32, avg bool) {
	h264ChromaMCStridesHighScalar(dst, src, dstStride, srcStride, int(height), int(x), int(y), int(width), avg)
}
