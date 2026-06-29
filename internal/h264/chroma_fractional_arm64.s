// SPDX-License-Identifier: LGPL-2.1-or-later
//go:build !purego && arm64

#include "textflag.h"

// func h264ChromaMCXYASM(dst *uint8, src *uint8, dstStride int, srcStride int, height int32, width int32, a int32, b int32, c int32, d int32, step int, avg int32)
TEXT ·h264ChromaMCXYASM(SB), NOSPLIT, $0-72
	MOVD dst+0(FP), R0
	MOVD src+8(FP), R1
	MOVD dstStride+16(FP), R2
	MOVD srcStride+24(FP), R3
	MOVW height+32(FP), R4
	MOVW a+40(FP), R6
	MOVW b+44(FP), R7
	MOVW c+48(FP), R8
	MOVW d+52(FP), R9
	MOVD step+56(FP), R14
	MOVW avg+64(FP), R15
	CBZW R4, chroma_xy_done
	CBZW R9, chroma_axis_row

chroma_bilinear_row:
	MOVD R0, R10
	MOVD R1, R11
	MOVW width+36(FP), R5
chroma_bilinear_col:
	MOVBU (R11), R12
	MULW  R6, R12, R12
	MOVBU 1(R11), R13
	MULW  R7, R13, R13
	ADDW  R13, R12, R12
	ADD   R3, R11, R16
	MOVBU (R16), R13
	MULW  R8, R13, R13
	ADDW  R13, R12, R12
	MOVBU 1(R16), R13
	MULW  R9, R13, R13
	ADDW  R13, R12, R12
	ADDW  $32, R12, R12
	ASRW  $6, R12, R12
	CBZW  R15, chroma_bilinear_store
	MOVBU (R10), R13
	ADDW  R13, R12, R12
	ADDW  $1, R12, R12
	LSRW  $1, R12, R12
chroma_bilinear_store:
	MOVB  R12, (R10)
	ADD   $1, R10, R10
	ADD   $1, R11, R11
	SUBW  $1, R5, R5
	CBNZW R5, chroma_bilinear_col
	ADD   R2, R0, R0
	ADD   R3, R1, R1
	SUBW  $1, R4, R4
	CBNZW R4, chroma_bilinear_row
	RET

chroma_axis_row:
	MOVD R0, R10
	MOVD R1, R11
	MOVW width+36(FP), R5
	ADDW R8, R7, R17
chroma_axis_col:
	MOVBU (R11), R12
	MULW  R6, R12, R12
	ADD   R14, R11, R16
	MOVBU (R16), R13
	MULW  R17, R13, R13
	ADDW  R13, R12, R12
	ADDW  $32, R12, R12
	ASRW  $6, R12, R12
	CBZW  R15, chroma_axis_store
	MOVBU (R10), R13
	ADDW  R13, R12, R12
	ADDW  $1, R12, R12
	LSRW  $1, R12, R12
chroma_axis_store:
	MOVB  R12, (R10)
	ADD   $1, R10, R10
	ADD   $1, R11, R11
	SUBW  $1, R5, R5
	CBNZW R5, chroma_axis_col
	ADD   R2, R0, R0
	ADD   R3, R1, R1
	SUBW  $1, R4, R4
	CBNZW R4, chroma_axis_row
chroma_xy_done:
	RET
