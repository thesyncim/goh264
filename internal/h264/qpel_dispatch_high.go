// SPDX-License-Identifier: LGPL-2.1-or-later
//go:build purego || (!amd64 && !arm64)

package h264

func h264QpelMCStridesHighKernel(dst []uint16, dstOffset int, dstStride int, src []uint16, srcOffset int, srcStride int, size int32, mx int32, my int32, avg bool, bitDepth int32) {
	h264QpelMCStridesHighScalar(dst, dstOffset, dstStride, src, srcOffset, srcStride, int(size), int(mx), int(my), avg, int(bitDepth))
}
