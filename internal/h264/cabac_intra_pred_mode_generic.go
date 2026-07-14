// SPDX-License-Identifier: LGPL-2.1-or-later
//go:build purego || !arm64

package h264

func decodeCABACMBIntra4x4PredModeDecoder(src *cabacSyntaxDecoder, predMode int) int {
	return decodeCABACMBIntra4x4PredModeDecoderScalar(src, predMode)
}
