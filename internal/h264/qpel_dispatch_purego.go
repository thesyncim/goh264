// SPDX-License-Identifier: LGPL-2.1-or-later
//go:build purego || (!amd64 && !arm64)

package h264

func h264QpelMCStridesKernel(dst []uint8, dstOffset int, dstStride int, src []uint8, srcOffset int, srcStride int, size int32, mx int32, my int32, avg bool) {
	h264QpelMCStridesScalar(dst, dstOffset, dstStride, src, srcOffset, srcStride, int(size), int(mx), int(my), avg)
}
