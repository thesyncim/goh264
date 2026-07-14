// SPDX-License-Identifier: LGPL-2.1-or-later
//go:build !purego && arm64

#include "textflag.h"

// h264ChromaMCDualXYASM applies the same fractional chroma interpolation to
// Cb and Cr while retaining the bilinear coefficients across both planes.
// func h264ChromaMCDualXYASM(dstCb *uint8, dstCr *uint8, srcCb *uint8, srcCr *uint8, dstStride int, srcStride int, height int32, width int32, a int32, b int32, c int32, d int32, step int, avg int32)
TEXT ·h264ChromaMCDualXYASM(SB), NOSPLIT, $16-88
	MOVD dstCb+0(FP), R0
	MOVD srcCb+16(FP), R1
	MOVD dstStride+32(FP), R2
	MOVD srcStride+40(FP), R3
	MOVW height+48(FP), R4
	MOVW a+56(FP), R6
	MOVW b+60(FP), R7
	MOVW c+64(FP), R8
	MOVW d+68(FP), R9
	MOVW avg+80(FP), R15
	MOVW width+52(FP), R5
	CMPW $8, R5
	BEQ dual_chroma_xy_cb_width8
	CMPW $4, R5
	BEQ dual_chroma_xy_cb_width4
	BL ·h264ChromaMC2XYNEONInternal(SB)
	B dual_chroma_xy_cr
dual_chroma_xy_cb_width8:
	BL ·h264ChromaMC8XYNEONInternal(SB)
	B dual_chroma_xy_cr
dual_chroma_xy_cb_width4:
	BL ·h264ChromaMC4XYNEONInternal(SB)

dual_chroma_xy_cr:
	MOVD dstCr+8(FP), R0
	MOVD srcCr+24(FP), R1
	MOVD dstStride+32(FP), R2
	MOVD srcStride+40(FP), R3
	MOVW height+48(FP), R4
	MOVW a+56(FP), R6
	MOVW b+60(FP), R7
	MOVW c+64(FP), R8
	MOVW d+68(FP), R9
	MOVW avg+80(FP), R15
	MOVW width+52(FP), R5
	CMPW $8, R5
	BEQ dual_chroma_xy_cr_width8
	CMPW $4, R5
	BEQ dual_chroma_xy_cr_width4
	BL ·h264ChromaMC2XYNEONInternal(SB)
	RET
dual_chroma_xy_cr_width8:
	BL ·h264ChromaMC8XYNEONInternal(SB)
	RET
dual_chroma_xy_cr_width4:
	BL ·h264ChromaMC4XYNEONInternal(SB)
	RET
