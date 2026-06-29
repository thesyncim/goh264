// SPDX-License-Identifier: LGPL-2.1-or-later
//go:build !purego && amd64

#include "textflag.h"

// func h264ChromaMC8Put00ASM(dst *uint8, src *uint8, dstStride int, srcStride int, height int32)
TEXT ·h264ChromaMC8Put00ASM(SB), NOSPLIT, $0-40
	MOVQ dst+0(FP), DI
	MOVQ src+8(FP), SI
	MOVQ dstStride+16(FP), DX
	MOVQ srcStride+24(FP), CX
	MOVL height+32(FP), R8
	TESTL R8, R8
	JZ   put_done
put_loop:
	MOVQ (SI), AX
	MOVQ AX, (DI)
	ADDQ DX, DI
	ADDQ CX, SI
	DECL R8
	JNZ  put_loop
put_done:
	RET

// func h264ChromaMC8Avg00ASM(dst *uint8, src *uint8, dstStride int, srcStride int, height int32)
TEXT ·h264ChromaMC8Avg00ASM(SB), NOSPLIT, $0-40
	MOVQ dst+0(FP), DI
	MOVQ src+8(FP), SI
	MOVQ dstStride+16(FP), DX
	MOVQ srcStride+24(FP), CX
	MOVL height+32(FP), R8
	MOVQ $0xfefefefefefefefe, R9
	TESTL R8, R8
	JZ   avg_done
avg_loop:
	MOVQ (SI), AX
	MOVQ (DI), BX
	MOVQ AX, R10
	ORQ  BX, R10
	XORQ BX, AX
	ANDQ R9, AX
	SHRQ $1, AX
	SUBQ AX, R10
	MOVQ R10, (DI)
	ADDQ DX, DI
	ADDQ CX, SI
	DECL R8
	JNZ  avg_loop
avg_done:
	RET
