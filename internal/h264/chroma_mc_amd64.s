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

// func h264ChromaMC4Put00ASM(dst *uint8, src *uint8, dstStride int, srcStride int, height int32)
TEXT ·h264ChromaMC4Put00ASM(SB), NOSPLIT, $0-40
	MOVQ dst+0(FP), DI
	MOVQ src+8(FP), SI
	MOVQ dstStride+16(FP), DX
	MOVQ srcStride+24(FP), CX
	MOVL height+32(FP), R8
	TESTL R8, R8
	JZ   put4_done
put4_loop:
	MOVL (SI), AX
	MOVL AX, (DI)
	ADDQ DX, DI
	ADDQ CX, SI
	DECL R8
	JNZ  put4_loop
put4_done:
	RET

// func h264ChromaMC4Avg00ASM(dst *uint8, src *uint8, dstStride int, srcStride int, height int32)
TEXT ·h264ChromaMC4Avg00ASM(SB), NOSPLIT, $0-40
	MOVQ dst+0(FP), DI
	MOVQ src+8(FP), SI
	MOVQ dstStride+16(FP), DX
	MOVQ srcStride+24(FP), CX
	MOVL height+32(FP), R8
	MOVL $0xfefefefe, R9
	TESTL R8, R8
	JZ   avg4_done
avg4_loop:
	MOVL (SI), AX
	MOVL (DI), BX
	MOVL AX, R10
	ORL  BX, R10
	XORL BX, AX
	ANDL R9, AX
	SHRL $1, AX
	SUBL AX, R10
	MOVL R10, (DI)
	ADDQ DX, DI
	ADDQ CX, SI
	DECL R8
	JNZ  avg4_loop
avg4_done:
	RET

// func h264ChromaMC2Put00ASM(dst *uint8, src *uint8, dstStride int, srcStride int, height int32)
TEXT ·h264ChromaMC2Put00ASM(SB), NOSPLIT, $0-40
	MOVQ dst+0(FP), DI
	MOVQ src+8(FP), SI
	MOVQ dstStride+16(FP), DX
	MOVQ srcStride+24(FP), CX
	MOVL height+32(FP), R8
	TESTL R8, R8
	JZ   put2_done
put2_loop:
	MOVW (SI), AX
	MOVW AX, (DI)
	ADDQ DX, DI
	ADDQ CX, SI
	DECL R8
	JNZ  put2_loop
put2_done:
	RET

// func h264ChromaMC2Avg00ASM(dst *uint8, src *uint8, dstStride int, srcStride int, height int32)
TEXT ·h264ChromaMC2Avg00ASM(SB), NOSPLIT, $0-40
	MOVQ dst+0(FP), DI
	MOVQ src+8(FP), SI
	MOVQ dstStride+16(FP), DX
	MOVQ srcStride+24(FP), CX
	MOVL height+32(FP), R8
	MOVL $0xfefe, R9
	TESTL R8, R8
	JZ   avg2_done
avg2_loop:
	XORL AX, AX
	XORL BX, BX
	MOVW (SI), AX
	MOVW (DI), BX
	MOVL AX, R10
	ORL  BX, R10
	XORL BX, AX
	ANDL R9, AX
	SHRL $1, AX
	SUBL AX, R10
	MOVW R10, (DI)
	ADDQ DX, DI
	ADDQ CX, SI
	DECL R8
	JNZ  avg2_loop
avg2_done:
	RET

// func h264ChromaMC1Put00ASM(dst *uint8, src *uint8, dstStride int, srcStride int, height int32)
TEXT ·h264ChromaMC1Put00ASM(SB), NOSPLIT, $0-40
	MOVQ dst+0(FP), DI
	MOVQ src+8(FP), SI
	MOVQ dstStride+16(FP), DX
	MOVQ srcStride+24(FP), CX
	MOVL height+32(FP), R8
	TESTL R8, R8
	JZ   put1_done
put1_loop:
	MOVB (SI), AX
	MOVB AX, (DI)
	ADDQ DX, DI
	ADDQ CX, SI
	DECL R8
	JNZ  put1_loop
put1_done:
	RET

// func h264ChromaMC1Avg00ASM(dst *uint8, src *uint8, dstStride int, srcStride int, height int32)
TEXT ·h264ChromaMC1Avg00ASM(SB), NOSPLIT, $0-40
	MOVQ dst+0(FP), DI
	MOVQ src+8(FP), SI
	MOVQ dstStride+16(FP), DX
	MOVQ srcStride+24(FP), CX
	MOVL height+32(FP), R8
	TESTL R8, R8
	JZ   avg1_done
avg1_loop:
	XORL AX, AX
	XORL BX, BX
	MOVB (SI), AX
	MOVB (DI), BX
	ADDL BX, AX
	ADDL $1, AX
	SHRL $1, AX
	MOVB AX, (DI)
	ADDQ DX, DI
	ADDQ CX, SI
	DECL R8
	JNZ  avg1_loop
avg1_done:
	RET
