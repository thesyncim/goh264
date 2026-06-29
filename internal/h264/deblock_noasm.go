// SPDX-License-Identifier: LGPL-2.1-or-later
//go:build purego || (!amd64 && !arm64)

package h264

const h264LoopFilterLumaASMEnabled = false

func h264VLoopFilterLuma8ASM(pix *uint8, stride int, alpha int32, beta int32, tc0 *int8) {
}

func h264HLoopFilterLuma8ASM(pix *uint8, stride int, alpha int32, beta int32, tc0 *int8) {
}
