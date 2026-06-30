// SPDX-License-Identifier: LGPL-2.1-or-later
//go:build purego || !arm64

package h264

const h264LoopFilterChromaASMEnabled = false

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
