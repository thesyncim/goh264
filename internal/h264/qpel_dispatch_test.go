// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

import "testing"

func TestH264QpelMCDispatchMatchesScalar(t *testing.T) {
	const stride = 48
	const offset = 6*stride + 6

	for _, avg := range []bool{false, true} {
		for _, size := range []int{2, 4, 8, 16} {
			for my := 0; my < 4; my++ {
				for mx := 0; mx < 4; mx++ {
					t.Run(qpelDispatchCaseName(avg, size, mx, my), func(t *testing.T) {
						dstKernel, src := makeQpelUnitFixture(stride, 48)
						dstScalar := append([]uint8(nil), dstKernel...)

						h264QpelMCStridesKernel(dstKernel, offset, stride, src, offset, stride, int32(size), int32(mx), int32(my), avg)
						h264QpelMCStridesScalar(dstScalar, offset, stride, src, offset, stride, size, mx, my, avg)

						if string(dstKernel) != string(dstScalar) {
							t.Fatalf("kernel output differs from scalar")
						}
					})
				}
			}
		}
	}
}

func TestH264QpelMCHighDispatchMatchesScalar(t *testing.T) {
	const stride = 48
	const offset = 6*stride + 6

	for _, bitDepth := range []int{9, 10, 12, 14} {
		for _, avg := range []bool{false, true} {
			for _, size := range []int{2, 4, 8, 16} {
				for my := 0; my < 4; my++ {
					for mx := 0; mx < 4; mx++ {
						t.Run(qpelDispatchCaseName(avg, size, mx, my), func(t *testing.T) {
							dstKernel, src := makeQpelUnitFixtureHigh(stride, 48, bitDepth)
							dstScalar := append([]uint16(nil), dstKernel...)

							h264QpelMCStridesHighKernel(dstKernel, offset, stride, src, offset, stride, int32(size), int32(mx), int32(my), avg, int32(bitDepth))
							h264QpelMCStridesHighScalar(dstScalar, offset, stride, src, offset, stride, size, mx, my, avg, bitDepth)

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
}

func qpelDispatchCaseName(avg bool, size int, mx int, my int) string {
	op := "put"
	if avg {
		op = "avg"
	}
	return op + "_qpel" + itoaSmall(size) + "_" + h264QpelOracleSuffixes[my*4+mx]
}

func itoaSmall(v int) string {
	switch v {
	case 2:
		return "2"
	case 4:
		return "4"
	case 8:
		return "8"
	case 16:
		return "16"
	default:
		return "x"
	}
}

func makeQpelUnitFixtureHigh(stride int, rows int, bitDepth int) ([]uint16, []uint16) {
	dst := make([]uint16, stride*rows)
	src := make([]uint16, stride*rows)
	mask := (1 << uint(bitDepth)) - 1
	for i := range dst {
		dst[i] = uint16((20 + i*37) & mask)
		src[i] = uint16((10 + i*29) & mask)
	}
	return dst, src
}
