// SPDX-License-Identifier: LGPL-2.1-or-later
//go:build purego || (!amd64 && !arm64)

package h264

const h264LoopFilterLumaIntraASMEnabled = false
const h264LoopFilterLumaIntraV8ASMEnabled = false
const h264LoopFilterLumaIntraH8ASMEnabled = false

func h264VLoopFilterLumaIntra8ASM(pix *uint8, stride int, alpha int32, beta int32) {
}

func h264HLoopFilterLumaIntra8ASM(pix *uint8, stride int, alpha int32, beta int32) {
}
