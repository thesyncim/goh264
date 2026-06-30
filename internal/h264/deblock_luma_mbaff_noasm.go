// SPDX-License-Identifier: LGPL-2.1-or-later
//go:build purego || !amd64

package h264

const h264LoopFilterLumaMBAFF8ASMEnabled = false

func h264HLoopFilterLumaMBAFF8ASM(pix *uint8, stride int, alpha int32, beta int32, tc0 *int8) {
}
