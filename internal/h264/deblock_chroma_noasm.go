// SPDX-License-Identifier: LGPL-2.1-or-later
//go:build purego || !arm64

package h264

const h264LoopFilterChromaASMEnabled = false

func h264VLoopFilterChroma8ASM(pix *uint8, stride int, alpha int32, beta int32, tc0 *int8) {
}

func h264HLoopFilterChroma8ASM(pix *uint8, stride int, alpha int32, beta int32, tc0 *int8) {
}
