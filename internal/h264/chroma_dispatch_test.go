// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

import (
	"fmt"
	"testing"
)

func TestH264ChromaMCDispatchMatchesScalar(t *testing.T) {
	for _, avg := range []bool{false, true} {
		for _, width := range []int{1, 2, 4, 8} {
			for _, xy := range [][2]int{{0, 0}, {3, 0}, {0, 5}, {3, 5}} {
				t.Run(chromaDispatchCaseName(avg, width, xy[0], xy[1]), func(t *testing.T) {
					const stride = 24
					const height = 5
					dstKernel := makeChromaUnitDst(stride, 8)
					dstScalar := append([]uint8(nil), dstKernel...)
					src := makeChromaUnitSrc(stride, 8)

					h264ChromaMCStridesKernel(dstKernel, src, stride, stride, int32(height), int32(xy[0]), int32(xy[1]), int32(width), avg)
					h264ChromaMCStridesScalar(dstScalar, src, stride, stride, height, xy[0], xy[1], width, avg)

					if string(dstKernel) != string(dstScalar) {
						t.Fatalf("kernel output differs from scalar")
					}
				})
			}
		}
	}
}

func TestH264ChromaMCDispatchMatchesScalarSeparateStrides(t *testing.T) {
	for _, avg := range []bool{false, true} {
		for _, width := range []int{1, 2, 4, 8} {
			t.Run(chromaDispatchCaseName(avg, width, 0, 0), func(t *testing.T) {
				const dstStride = 13
				const srcStride = 17
				const height = 6
				dstKernel := makeChromaUnitDst(dstStride, height)
				dstScalar := append([]uint8(nil), dstKernel...)
				src := makeChromaUnitSrc(srcStride, height)

				h264ChromaMCStridesKernel(dstKernel, src, dstStride, srcStride, int32(height), 0, 0, int32(width), avg)
				h264ChromaMCStridesScalar(dstScalar, src, dstStride, srcStride, height, 0, 0, width, avg)

				if string(dstKernel) != string(dstScalar) {
					t.Fatalf("kernel output differs from scalar")
				}
			})
		}
	}
}

func TestH264ChromaMCHighDispatchMatchesScalar(t *testing.T) {
	for _, bitDepth := range []int{9, 10, 12, 14} {
		for _, avg := range []bool{false, true} {
			for _, width := range []int{1, 2, 4, 8} {
				for _, xy := range [][2]int{{0, 0}, {3, 0}, {0, 5}, {3, 5}} {
					t.Run(chromaDispatchCaseName(avg, width, xy[0], xy[1]), func(t *testing.T) {
						const stride = 24
						const height = 5
						dstKernel := makeChromaUnitDstHigh(stride, 8, bitDepth)
						dstScalar := append([]uint16(nil), dstKernel...)
						src := makeChromaUnitSrcHigh(stride, 8, bitDepth)

						h264ChromaMCStridesHighKernel(dstKernel, src, stride, stride, int32(height), int32(xy[0]), int32(xy[1]), int32(width), avg)
						h264ChromaMCStridesHighScalar(dstScalar, src, stride, stride, height, xy[0], xy[1], width, avg)

						if len(dstKernel) != len(dstScalar) {
							t.Fatalf("kernel len = %d, scalar len = %d", len(dstKernel), len(dstScalar))
						}
						for i := range dstKernel {
							if dstKernel[i] != dstScalar[i] {
								t.Fatalf("kernel[%d] = %d, scalar = %d", i, dstKernel[i], dstScalar[i])
							}
						}
					})
				}
			}
		}
	}
}

func chromaDispatchCaseName(avg bool, width int, x int, y int) string {
	op := "put"
	if avg {
		op = "avg"
	}
	return fmt.Sprintf("%s_w%d_x%d_y%d", op, width, x, y)
}

func makeChromaUnitDstHigh(stride int, rows int, bitDepth int) []uint16 {
	dst := make([]uint16, stride*rows)
	mask := (1 << uint(bitDepth)) - 1
	for i := range dst {
		dst[i] = uint16((200 - i*5) & mask)
	}
	return dst
}

func makeChromaUnitSrcHigh(stride int, rows int, bitDepth int) []uint16 {
	src := make([]uint16, stride*rows)
	mask := (1 << uint(bitDepth)) - 1
	for i := range src {
		src[i] = uint16((10 + i*19) & mask)
	}
	return src
}
