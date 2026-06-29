// SPDX-License-Identifier: LGPL-2.1-or-later
//go:build !purego && arm64

#include "textflag.h"

// func h264QpelMC16Put00ASM(dst *uint8, src *uint8, dstStride int, srcStride int)
TEXT ·h264QpelMC16Put00ASM(SB), NOSPLIT, $0-32
	MOVD dst+0(FP), R0
	MOVD src+8(FP), R1
	MOVD dstStride+16(FP), R2
	MOVD srcStride+24(FP), R3
	MOVW $16, R4
qpel16_put00_loop:
	MOVD (R1), R5
	MOVD 8(R1), R6
	MOVD R5, (R0)
	MOVD R6, 8(R0)
	ADD  R2, R0, R0
	ADD  R3, R1, R1
	SUBW $1, R4, R4
	CBNZW R4, qpel16_put00_loop
	RET

// func h264QpelMC16Avg00ASM(dst *uint8, src *uint8, dstStride int, srcStride int)
TEXT ·h264QpelMC16Avg00ASM(SB), NOSPLIT, $0-32
	MOVD dst+0(FP), R0
	MOVD src+8(FP), R1
	MOVD dstStride+16(FP), R2
	MOVD srcStride+24(FP), R3
	MOVW $16, R4
	MOVD $0xfefefefefefefefe, R6
qpel16_avg00_loop:
	MOVD (R1), R5
	MOVD (R0), R7
	ORR  R7, R5, R8
	EOR  R7, R5, R5
	AND  R6, R5, R5
	LSR  $1, R5, R5
	SUB  R5, R8, R8
	MOVD R8, (R0)
	MOVD 8(R1), R5
	MOVD 8(R0), R7
	ORR  R7, R5, R8
	EOR  R7, R5, R5
	AND  R6, R5, R5
	LSR  $1, R5, R5
	SUB  R5, R8, R8
	MOVD R8, 8(R0)
	ADD  R2, R0, R0
	ADD  R3, R1, R1
	SUBW $1, R4, R4
	CBNZW R4, qpel16_avg00_loop
	RET

// func h264QpelMC8Put00ASM(dst *uint8, src *uint8, dstStride int, srcStride int)
TEXT ·h264QpelMC8Put00ASM(SB), NOSPLIT, $0-32
	MOVD dst+0(FP), R0
	MOVD src+8(FP), R1
	MOVD dstStride+16(FP), R2
	MOVD srcStride+24(FP), R3
	MOVW $8, R4
qpel8_put00_loop:
	MOVD (R1), R5
	MOVD R5, (R0)
	ADD  R2, R0, R0
	ADD  R3, R1, R1
	SUBW $1, R4, R4
	CBNZW R4, qpel8_put00_loop
	RET

// func h264QpelMC8Avg00ASM(dst *uint8, src *uint8, dstStride int, srcStride int)
TEXT ·h264QpelMC8Avg00ASM(SB), NOSPLIT, $0-32
	MOVD dst+0(FP), R0
	MOVD src+8(FP), R1
	MOVD dstStride+16(FP), R2
	MOVD srcStride+24(FP), R3
	MOVW $8, R4
	MOVD $0xfefefefefefefefe, R6
qpel8_avg00_loop:
	MOVD (R1), R5
	MOVD (R0), R7
	ORR  R7, R5, R8
	EOR  R7, R5, R5
	AND  R6, R5, R5
	LSR  $1, R5, R5
	SUB  R5, R8, R8
	MOVD R8, (R0)
	ADD  R2, R0, R0
	ADD  R3, R1, R1
	SUBW $1, R4, R4
	CBNZW R4, qpel8_avg00_loop
	RET

// func h264QpelMC4Put00ASM(dst *uint8, src *uint8, dstStride int, srcStride int)
TEXT ·h264QpelMC4Put00ASM(SB), NOSPLIT, $0-32
	MOVD dst+0(FP), R0
	MOVD src+8(FP), R1
	MOVD dstStride+16(FP), R2
	MOVD srcStride+24(FP), R3
	MOVW $4, R4
qpel4_put00_loop:
	MOVWU (R1), R5
	MOVW  R5, (R0)
	ADD   R2, R0, R0
	ADD   R3, R1, R1
	SUBW  $1, R4, R4
	CBNZW R4, qpel4_put00_loop
	RET

// func h264QpelMC4Avg00ASM(dst *uint8, src *uint8, dstStride int, srcStride int)
TEXT ·h264QpelMC4Avg00ASM(SB), NOSPLIT, $0-32
	MOVD dst+0(FP), R0
	MOVD src+8(FP), R1
	MOVD dstStride+16(FP), R2
	MOVD srcStride+24(FP), R3
	MOVW $4, R4
	MOVD $0xfefefefe, R6
qpel4_avg00_loop:
	MOVWU (R1), R5
	MOVWU (R0), R7
	ORR   R7, R5, R8
	EOR   R7, R5, R5
	AND   R6, R5, R5
	LSR   $1, R5, R5
	SUB   R5, R8, R8
	MOVW  R8, (R0)
	ADD   R2, R0, R0
	ADD   R3, R1, R1
	SUBW  $1, R4, R4
	CBNZW R4, qpel4_avg00_loop
	RET

// func h264QpelMC2Put00ASM(dst *uint8, src *uint8, dstStride int, srcStride int)
TEXT ·h264QpelMC2Put00ASM(SB), NOSPLIT, $0-32
	MOVD dst+0(FP), R0
	MOVD src+8(FP), R1
	MOVD dstStride+16(FP), R2
	MOVD srcStride+24(FP), R3
	MOVW $2, R4
qpel2_put00_loop:
	MOVHU (R1), R5
	MOVH  R5, (R0)
	ADD   R2, R0, R0
	ADD   R3, R1, R1
	SUBW  $1, R4, R4
	CBNZW R4, qpel2_put00_loop
	RET

// func h264QpelMC2Avg00ASM(dst *uint8, src *uint8, dstStride int, srcStride int)
TEXT ·h264QpelMC2Avg00ASM(SB), NOSPLIT, $0-32
	MOVD dst+0(FP), R0
	MOVD src+8(FP), R1
	MOVD dstStride+16(FP), R2
	MOVD srcStride+24(FP), R3
	MOVW $2, R4
	MOVD $0xfefe, R6
qpel2_avg00_loop:
	MOVHU (R1), R5
	MOVHU (R0), R7
	ORR   R7, R5, R8
	EOR   R7, R5, R5
	AND   R6, R5, R5
	LSR   $1, R5, R5
	SUB   R5, R8, R8
	MOVH  R8, (R0)
	ADD   R2, R0, R0
	ADD   R3, R1, R1
	SUBW  $1, R4, R4
	CBNZW R4, qpel2_avg00_loop
	RET
