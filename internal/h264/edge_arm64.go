// SPDX-License-Identifier: LGPL-2.1-or-later
//go:build !purego && arm64

package h264

const h264EmulatedEdgeMCASMEnabled = true

// h264EmulatedEdgeMCRowsASM implements the row-copy and horizontal-extension
// phase of FFmpeg's ff_emulated_edge_mc_8 for H.264's fixed 21- and 9-pixel
// scratch blocks.
//
//go:noescape
func h264EmulatedEdgeMCRowsASM(dst *uint8, dstStride int, src *uint8, srcStride int, blockW int, blockH int, startX int, startY int, endX int, endY int)
