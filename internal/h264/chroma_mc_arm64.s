// SPDX-License-Identifier: LGPL-2.1-or-later
//go:build !purego && arm64

#include "textflag.h"

// func h264ChromaMC8Put00ASM(dst *uint8, src *uint8, dstStride int, srcStride int, height int32)
TEXT ·h264ChromaMC8Put00ASM(SB), NOSPLIT, $0-40
	MOVD dst+0(FP), R0
	MOVD src+8(FP), R1
	MOVD dstStride+16(FP), R2
	MOVD srcStride+24(FP), R3
	MOVW height+32(FP), R4
	CBZW R4, put_done
put_loop:
	MOVD (R1), R5
	MOVD R5, (R0)
	ADD  R2, R0, R0
	ADD  R3, R1, R1
	SUBW $1, R4, R4
	CBNZW R4, put_loop
put_done:
	RET

// func h264ChromaMC8Avg00ASM(dst *uint8, src *uint8, dstStride int, srcStride int, height int32)
TEXT ·h264ChromaMC8Avg00ASM(SB), NOSPLIT, $0-40
	MOVD dst+0(FP), R0
	MOVD src+8(FP), R1
	MOVD dstStride+16(FP), R2
	MOVD srcStride+24(FP), R3
	MOVW height+32(FP), R4
	MOVD $0xfefefefefefefefe, R6
	CBZW R4, avg_done
avg_loop:
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
	CBNZW R4, avg_loop
avg_done:
	RET

// func h264ChromaMC4Put00ASM(dst *uint8, src *uint8, dstStride int, srcStride int, height int32)
TEXT ·h264ChromaMC4Put00ASM(SB), NOSPLIT, $0-40
	MOVD dst+0(FP), R0
	MOVD src+8(FP), R1
	MOVD dstStride+16(FP), R2
	MOVD srcStride+24(FP), R3
	MOVW height+32(FP), R4
	CBZW R4, put4_done
put4_loop:
	MOVWU (R1), R5
	MOVW  R5, (R0)
	ADD   R2, R0, R0
	ADD   R3, R1, R1
	SUBW  $1, R4, R4
	CBNZW R4, put4_loop
put4_done:
	RET

// func h264ChromaMC4Avg00ASM(dst *uint8, src *uint8, dstStride int, srcStride int, height int32)
TEXT ·h264ChromaMC4Avg00ASM(SB), NOSPLIT, $0-40
	MOVD dst+0(FP), R0
	MOVD src+8(FP), R1
	MOVD dstStride+16(FP), R2
	MOVD srcStride+24(FP), R3
	MOVW height+32(FP), R4
	MOVD $0xfefefefe, R6
	CBZW R4, avg4_done
avg4_loop:
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
	CBNZW R4, avg4_loop
avg4_done:
	RET

// func h264ChromaMC2Put00ASM(dst *uint8, src *uint8, dstStride int, srcStride int, height int32)
TEXT ·h264ChromaMC2Put00ASM(SB), NOSPLIT, $0-40
	MOVD dst+0(FP), R0
	MOVD src+8(FP), R1
	MOVD dstStride+16(FP), R2
	MOVD srcStride+24(FP), R3
	MOVW height+32(FP), R4
	CBZW R4, put2_done
put2_loop:
	MOVHU (R1), R5
	MOVH  R5, (R0)
	ADD   R2, R0, R0
	ADD   R3, R1, R1
	SUBW  $1, R4, R4
	CBNZW R4, put2_loop
put2_done:
	RET

// func h264ChromaMC2Avg00ASM(dst *uint8, src *uint8, dstStride int, srcStride int, height int32)
TEXT ·h264ChromaMC2Avg00ASM(SB), NOSPLIT, $0-40
	MOVD dst+0(FP), R0
	MOVD src+8(FP), R1
	MOVD dstStride+16(FP), R2
	MOVD srcStride+24(FP), R3
	MOVW height+32(FP), R4
	MOVD $0xfefe, R6
	CBZW R4, avg2_done
avg2_loop:
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
	CBNZW R4, avg2_loop
avg2_done:
	RET

// func h264ChromaMC1Put00ASM(dst *uint8, src *uint8, dstStride int, srcStride int, height int32)
TEXT ·h264ChromaMC1Put00ASM(SB), NOSPLIT, $0-40
	MOVD dst+0(FP), R0
	MOVD src+8(FP), R1
	MOVD dstStride+16(FP), R2
	MOVD srcStride+24(FP), R3
	MOVW height+32(FP), R4
	CBZW R4, put1_done
put1_loop:
	MOVBU (R1), R5
	MOVB  R5, (R0)
	ADD   R2, R0, R0
	ADD   R3, R1, R1
	SUBW  $1, R4, R4
	CBNZW R4, put1_loop
put1_done:
	RET

// func h264ChromaMC1Avg00ASM(dst *uint8, src *uint8, dstStride int, srcStride int, height int32)
TEXT ·h264ChromaMC1Avg00ASM(SB), NOSPLIT, $0-40
	MOVD dst+0(FP), R0
	MOVD src+8(FP), R1
	MOVD dstStride+16(FP), R2
	MOVD srcStride+24(FP), R3
	MOVW height+32(FP), R4
	CBZW R4, avg1_done
avg1_loop:
	MOVBU (R1), R5
	MOVBU (R0), R7
	ADD   R7, R5, R5
	ADD   $1, R5, R5
	LSR   $1, R5, R5
	MOVB  R5, (R0)
	ADD   R2, R0, R0
	ADD   R3, R1, R1
	SUBW  $1, R4, R4
	CBNZW R4, avg1_loop
avg1_done:
	RET
