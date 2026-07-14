// SPDX-License-Identifier: LGPL-2.1-or-later
//go:build purego || !arm64

package h264

func decodeCABACMBMVDDecoder(src *cabacSyntaxDecoder, ctxBase int, amvd int) (int32, int, error) {
	return decodeCABACMBMVDDecoderScalar(src, ctxBase, amvd)
}
