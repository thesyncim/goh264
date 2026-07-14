// SPDX-License-Identifier: LGPL-2.1-or-later
//go:build !purego && arm64

package h264

func decodeCABACResidualSignificance4x4Decoder(src *cabacSyntaxDecoder, index *[64]uint8, sigCtxBase int, lastCtxBase int) (int, int) {
	return h264CABACResidualSignificanceFixedASM(src.cabac, src.state, index, sigCtxBase, lastCtxBase, 16)
}

func decodeCABACResidualSignificanceAC15Decoder(src *cabacSyntaxDecoder, index *[64]uint8, sigCtxBase int, lastCtxBase int) (int, int) {
	return h264CABACResidualSignificanceFixedASM(src.cabac, src.state, index, sigCtxBase, lastCtxBase, 15)
}

//go:noescape
func h264CABACResidualSignificanceFixedASM(c *cabacContext, states *[1024]uint8, index *[64]uint8, sigCtxBase int, lastCtxBase int, maxCoeff int) (coeffCount int, last int)
