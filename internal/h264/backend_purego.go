// SPDX-License-Identifier: LGPL-2.1-or-later
//go:build purego || (!amd64 && !arm64)

package h264

func DecoderBackendKind() string {
	return "go-pure"
}

func DecoderBackendNote() string {
	return "Go decoder backend is pure Go in this build; native C+asm comparison lanes remain quality-valid but are not Go+asm performance claims."
}
