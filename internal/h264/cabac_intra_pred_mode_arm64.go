// SPDX-License-Identifier: LGPL-2.1-or-later
//go:build !purego && arm64

package h264

func decodeCABACMBIntra4x4PredModeDecoder(src *cabacSyntaxDecoder, predMode int) int {
	return h264CABACIntra4x4PredModeASM(src.cabac, src.state, predMode)
}

//go:noescape
func h264CABACIntra4x4PredModeASM(c *cabacContext, states *[1024]uint8, predMode int) (mode int)
