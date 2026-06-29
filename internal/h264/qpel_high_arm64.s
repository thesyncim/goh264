// SPDX-License-Identifier: LGPL-2.1-or-later
//go:build !purego && arm64

#include "textflag.h"

// func h264QpelMCHigh00ASM(dst *uint8, src *uint8, dstStride int, srcStride int, size int32, avg int32)
TEXT ·h264QpelMCHigh00ASM(SB), NOSPLIT, $0-40
	MOVD dst+0(FP), R0
	MOVD src+8(FP), R1
	MOVD dstStride+16(FP), R2
	MOVD srcStride+24(FP), R3
	MOVW size+32(FP), R4
	MOVW avg+36(FP), R5
	CBNZW R5, qpel_high00_avg_row

qpel_high00_put_row:
	MOVD R0, R10
	MOVD R1, R11
	MOVW size+32(FP), R9
qpel_high00_put_col:
	MOVHU (R11), R12
	MOVH  R12, (R10)
	ADD   $2, R10, R10
	ADD   $2, R11, R11
	SUBW  $1, R9, R9
	CBNZW R9, qpel_high00_put_col
	ADD   R2, R0, R0
	ADD   R3, R1, R1
	SUBW  $1, R4, R4
	CBNZW R4, qpel_high00_put_row
	RET

qpel_high00_avg_row:
	MOVD R0, R10
	MOVD R1, R11
	MOVW size+32(FP), R9
qpel_high00_avg_col:
	MOVHU (R11), R12
	MOVHU (R10), R13
	ADDW  R13, R12, R12
	ADDW  $1, R12, R12
	LSRW  $1, R12, R12
	MOVH  R12, (R10)
	ADD   $2, R10, R10
	ADD   $2, R11, R11
	SUBW  $1, R9, R9
	CBNZW R9, qpel_high00_avg_col
	ADD   R2, R0, R0
	ADD   R3, R1, R1
	SUBW  $1, R4, R4
	CBNZW R4, qpel_high00_avg_row
	RET
