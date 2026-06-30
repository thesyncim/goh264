// SPDX-License-Identifier: LGPL-2.1-or-later
//go:build purego || !arm64

package h264

const h264LoopFilterChromaMBAFFIntraH8ASMEnabled = false

func h264HLoopFilterChromaMBAFFIntra8ASM(pix *uint8, stride int, alpha int32, beta int32) {
}
