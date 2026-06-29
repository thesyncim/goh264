// SPDX-License-Identifier: LGPL-2.1-or-later
//go:build !purego && (amd64 || arm64)

package h264

func DecoderBackendKind() string {
	return "go-partial-asm"
}

func DecoderBackendNote() string {
	return "Go decoder build has assembly dispatch enabled for implemented kernels; unported kernels still use scalar fallback."
}
