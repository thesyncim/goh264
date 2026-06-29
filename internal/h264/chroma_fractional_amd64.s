// SPDX-License-Identifier: LGPL-2.1-or-later
//go:build !purego && amd64

#include "textflag.h"

// func h264ChromaMCXYASM(dst *uint8, src *uint8, dstStride int, srcStride int, height int32, width int32, a int32, b int32, c int32, d int32, step int, avg int32)
TEXT ·h264ChromaMCXYASM(SB), NOSPLIT, $0-72
	MOVQ dst+0(FP), DI
	MOVQ src+8(FP), SI
	MOVQ dstStride+16(FP), DX
	MOVQ srcStride+24(FP), CX
	MOVL height+32(FP), R8
	TESTL R8, R8
	JZ    chroma_xy_done
	MOVL d+52(FP), R15
	TESTL R15, R15
	JZ    chroma_axis_row

chroma_bilinear_row:
	MOVQ DI, R10
	MOVQ SI, R11
	MOVL width+36(FP), R9
chroma_bilinear_col:
	XORL AX, AX
	MOVB (R11), AX
	IMULL a+40(FP), AX
	MOVL AX, R15
	XORL AX, AX
	MOVB 1(R11), AX
	IMULL b+44(FP), AX
	ADDL AX, R15
	MOVQ R11, R12
	ADDQ CX, R12
	XORL AX, AX
	MOVB (R12), AX
	IMULL c+48(FP), AX
	ADDL AX, R15
	XORL AX, AX
	MOVB 1(R12), AX
	IMULL d+52(FP), AX
	ADDL AX, R15
	ADDL $32, R15
	SARL $6, R15
	MOVL avg+64(FP), AX
	TESTL AX, AX
	JZ    chroma_bilinear_store
	XORL AX, AX
	MOVB (R10), AX
	ADDL AX, R15
	ADDL $1, R15
	SHRL $1, R15
chroma_bilinear_store:
	MOVB R15, (R10)
	INCQ R10
	INCQ R11
	DECL R9
	JNZ  chroma_bilinear_col
	ADDQ DX, DI
	ADDQ CX, SI
	DECL R8
	JNZ  chroma_bilinear_row
	RET

chroma_axis_row:
	MOVQ DI, R10
	MOVQ SI, R11
	MOVL width+36(FP), R9
chroma_axis_col:
	XORL AX, AX
	MOVB (R11), AX
	IMULL a+40(FP), AX
	MOVL AX, R15
	MOVQ R11, R12
	ADDQ step+56(FP), R12
	XORL AX, AX
	MOVB (R12), AX
	MOVL b+44(FP), BX
	ADDL c+48(FP), BX
	IMULL BX, AX
	ADDL AX, R15
	ADDL $32, R15
	SARL $6, R15
	MOVL avg+64(FP), AX
	TESTL AX, AX
	JZ    chroma_axis_store
	XORL AX, AX
	MOVB (R10), AX
	ADDL AX, R15
	ADDL $1, R15
	SHRL $1, R15
chroma_axis_store:
	MOVB R15, (R10)
	INCQ R10
	INCQ R11
	DECL R9
	JNZ  chroma_axis_col
	ADDQ DX, DI
	ADDQ CX, SI
	DECL R8
	JNZ  chroma_axis_row
chroma_xy_done:
	RET
