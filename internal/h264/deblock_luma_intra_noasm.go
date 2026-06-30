// SPDX-License-Identifier: LGPL-2.1-or-later
//go:build purego || !arm64

package h264

const h264LoopFilterLumaIntraASMEnabled = false

func h264VLoopFilterLumaIntra8ASM(pix *uint8, stride int, alpha int32, beta int32) {
}
