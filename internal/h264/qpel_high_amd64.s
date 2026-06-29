// SPDX-License-Identifier: LGPL-2.1-or-later
//go:build !purego && amd64

#include "textflag.h"

// func h264QpelMCHigh00ASM(dst *uint8, src *uint8, dstStride int, srcStride int, size int32, avg int32)
TEXT ·h264QpelMCHigh00ASM(SB), NOSPLIT, $0-40
	MOVQ dst+0(FP), DI
	MOVQ src+8(FP), SI
	MOVQ dstStride+16(FP), DX
	MOVQ srcStride+24(FP), CX
	MOVL size+32(FP), R8
	MOVL avg+36(FP), AX
	TESTL AX, AX
	JNZ   qpel_high00_avg_row

qpel_high00_put_row:
	MOVQ DI, R10
	MOVQ SI, R11
	MOVL size+32(FP), R9
qpel_high00_put_col:
	XORL AX, AX
	MOVW (R11), AX
	MOVW AX, (R10)
	ADDQ $2, R10
	ADDQ $2, R11
	DECL R9
	JNZ  qpel_high00_put_col
	ADDQ DX, DI
	ADDQ CX, SI
	DECL R8
	JNZ  qpel_high00_put_row
	RET

qpel_high00_avg_row:
	MOVQ DI, R10
	MOVQ SI, R11
	MOVL size+32(FP), R9
qpel_high00_avg_col:
	XORL AX, AX
	MOVW (R11), AX
	XORL BX, BX
	MOVW (R10), BX
	ADDL BX, AX
	ADDL $1, AX
	SHRL $1, AX
	MOVW AX, (R10)
	ADDQ $2, R10
	ADDQ $2, R11
	DECL R9
	JNZ  qpel_high00_avg_col
	ADDQ DX, DI
	ADDQ CX, SI
	DECL R8
	JNZ  qpel_high00_avg_row
	RET
