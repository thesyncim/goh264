// SPDX-License-Identifier: LGPL-2.1-or-later
//go:build purego || !arm64

package h264

func h264IDCTAdd(dst []uint8, block []int32, stride int) error {
	return h264IDCTAddScalar(dst, block, stride)
}

func h264IDCTDCAdd(dst []uint8, block []int32, stride int) error {
	return h264IDCTDCAddScalar(dst, block, stride)
}
