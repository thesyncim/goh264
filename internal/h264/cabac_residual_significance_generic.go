// SPDX-License-Identifier: LGPL-2.1-or-later
//go:build purego || !arm64

package h264

func decodeCABACResidualSignificance4x4Decoder(src *cabacSyntaxDecoder, index *[64]uint8, sigCtxBase int, lastCtxBase int) (int, int) {
	return decodeCABACResidualSignificance4x4DecoderScalar(src, index, sigCtxBase, lastCtxBase)
}

func decodeCABACResidualSignificanceAC15Decoder(src *cabacSyntaxDecoder, index *[64]uint8, sigCtxBase int, lastCtxBase int) (int, int) {
	return decodeCABACResidualSignificanceAC15DecoderScalar(src, index, sigCtxBase, lastCtxBase)
}
