// SPDX-License-Identifier: LGPL-2.1-or-later
//go:build !purego && arm64

package h264

// decodeCABACMBMVDDecoder keeps the complete common prefix, including the sign
// bin, in one assembly call. Only the rare escape suffix remains here so its
// invalid-data limit stays explicit and identical to the scalar oracle.
func decodeCABACMBMVDDecoder(src *cabacSyntaxDecoder, ctxBase int, amvd int) (int32, int, error) {
	value, mvd := h264CABACMVDCommonASM(src.cabac, src.state, ctxBase, cabacMVDContext(ctxBase, amvd))
	if mvd < 9 {
		return value, mvd, nil
	}

	k := 3
	for src.bypass() != 0 {
		mvd += 1 << k
		k++
		if k > 24 {
			return 0, 0, ErrInvalidData
		}
	}
	for k > 0 {
		k--
		mvd += src.bypass() << k
	}
	mvda := mvd
	if mvda >= 70 {
		mvda = 70
	}
	return src.bypassSign(int32(-mvd)), mvda, nil
}

//go:noescape
func h264CABACMVDCommonASM(c *cabacContext, states *[1024]uint8, ctxBase int, ctx int) (value int32, mvd int)
