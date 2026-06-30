// SPDX-License-Identifier: LGPL-2.1-or-later
//go:build purego || (!amd64 && !arm64)

package h264

const h264LoopFilterChromaASMEnabled = false
const h264LoopFilterChromaV8ASMEnabled = false
const h264LoopFilterChromaH8ASMEnabled = false
const h264LoopFilterChroma422H8ASMEnabled = false
const h264LoopFilterChromaIntraV8ASMEnabled = false
const h264LoopFilterChromaIntraH8ASMEnabled = false
const h264LoopFilterChroma422IntraH8ASMEnabled = false

func h264VLoopFilterChroma8ASM(pix *uint8, stride int, alpha int32, beta int32, tc0 *int8) {
}

func h264HLoopFilterChroma8ASM(pix *uint8, stride int, alpha int32, beta int32, tc0 *int8) {
}

func h264HLoopFilterChroma4228ASM(pix *uint8, stride int, alpha int32, beta int32, tc0 *int8) {
}

func h264VLoopFilterChromaIntra8ASM(pix *uint8, stride int, alpha int32, beta int32) {
}

func h264HLoopFilterChromaIntra8ASM(pix *uint8, stride int, alpha int32, beta int32) {
}

func h264HLoopFilterChroma422Intra8ASM(pix *uint8, stride int, alpha int32, beta int32) {
}
