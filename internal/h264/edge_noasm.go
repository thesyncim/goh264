// SPDX-License-Identifier: LGPL-2.1-or-later
//go:build purego || !arm64

package h264

const h264EmulatedEdgeMCASMEnabled = false

func h264EmulatedEdgeMCRowsASM(dst *uint8, dstStride int, src *uint8, srcStride int, blockW int, blockH int, startX int, startY int, endX int, endY int) {
}
