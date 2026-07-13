// SPDX-License-Identifier: LGPL-2.1-or-later
//go:build !purego && arm64

package h264

func h264IDCTAdd(dst []uint8, block []int32, stride int) error {
	if err := checkTransformAddArgs(dst, block, 16, stride, 4); err != nil {
		return err
	}
	h264IDCTAddASM(&dst[0], &block[0], stride)
	return nil
}

func h264IDCTDCAdd(dst []uint8, block []int32, stride int) error {
	if err := checkTransformAddArgs(dst, block, 1, stride, 4); err != nil {
		return err
	}
	h264IDCTDCAddASM(&dst[0], &block[0], stride)
	return nil
}

//go:noescape
func h264IDCTAddASM(dst *uint8, block *int32, stride int)

//go:noescape
func h264IDCTDCAddASM(dst *uint8, block *int32, stride int)
