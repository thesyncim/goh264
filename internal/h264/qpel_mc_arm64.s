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

// func h264QpelMC16Put20ASM(dst *uint8, src *uint8, dstStride int, srcStride int)
TEXT ·h264QpelMC16Put20ASM(SB), NOSPLIT, $0-32
	MOVD dst+0(FP), R0
	MOVD src+8(FP), R1
	MOVD dstStride+16(FP), R2
	MOVD srcStride+24(FP), R3
	MOVW $16, R4
qpel16_put20_row:
	MOVD R0, R10
	MOVD R1, R11
	MOVW $16, R9
qpel16_put20_col:
	MOVBU (R11), R5
	MOVBU 1(R11), R6
	ADDW  R6, R5, R5
	LSLW  $4, R5, R12
	ADDW  R5<<2, R12, R12
	MOVBU -1(R11), R5
	MOVBU 2(R11), R6
	ADDW  R6, R5, R5
	ADDW  R5<<2, R5, R5
	SUBW  R5, R12, R12
	MOVBU -2(R11), R5
	MOVBU 3(R11), R6
	ADDW  R6, R5, R5
	ADDW  R5, R12, R12
	ADDW  $16, R12, R12
	ASRW  $5, R12, R12
	CMPW  $0, R12
	BGE   qpel16_put20_nonnegative
	MOVW  ZR, R12
	B     qpel16_put20_store
qpel16_put20_nonnegative:
	CMPW  $255, R12
	BLE   qpel16_put20_store
	MOVW  $255, R12
qpel16_put20_store:
	MOVB  R12, (R10)
	ADD   $1, R10, R10
	ADD   $1, R11, R11
	SUBW  $1, R9, R9
	CBNZW R9, qpel16_put20_col
	ADD   R2, R0, R0
	ADD   R3, R1, R1
	SUBW  $1, R4, R4
	CBNZW R4, qpel16_put20_row
	RET

// func h264QpelMC16Avg20ASM(dst *uint8, src *uint8, dstStride int, srcStride int)
TEXT ·h264QpelMC16Avg20ASM(SB), NOSPLIT, $0-32
	MOVD dst+0(FP), R0
	MOVD src+8(FP), R1
	MOVD dstStride+16(FP), R2
	MOVD srcStride+24(FP), R3
	MOVW $16, R4
qpel16_avg20_row:
	MOVD R0, R10
	MOVD R1, R11
	MOVW $16, R9
qpel16_avg20_col:
	MOVBU (R11), R5
	MOVBU 1(R11), R6
	ADDW  R6, R5, R5
	LSLW  $4, R5, R12
	ADDW  R5<<2, R12, R12
	MOVBU -1(R11), R5
	MOVBU 2(R11), R6
	ADDW  R6, R5, R5
	ADDW  R5<<2, R5, R5
	SUBW  R5, R12, R12
	MOVBU -2(R11), R5
	MOVBU 3(R11), R6
	ADDW  R6, R5, R5
	ADDW  R5, R12, R12
	ADDW  $16, R12, R12
	ASRW  $5, R12, R12
	CMPW  $0, R12
	BGE   qpel16_avg20_nonnegative
	MOVW  ZR, R12
	B     qpel16_avg20_clip_done
qpel16_avg20_nonnegative:
	CMPW  $255, R12
	BLE   qpel16_avg20_clip_done
	MOVW  $255, R12
qpel16_avg20_clip_done:
	MOVBU (R10), R7
	ADDW  R7, R12, R12
	ADDW  $1, R12, R12
	LSRW  $1, R12, R12
	MOVB  R12, (R10)
	ADD   $1, R10, R10
	ADD   $1, R11, R11
	SUBW  $1, R9, R9
	CBNZW R9, qpel16_avg20_col
	ADD   R2, R0, R0
	ADD   R3, R1, R1
	SUBW  $1, R4, R4
	CBNZW R4, qpel16_avg20_row
	RET

// func h264QpelMC8Put20ASM(dst *uint8, src *uint8, dstStride int, srcStride int)
TEXT ·h264QpelMC8Put20ASM(SB), NOSPLIT, $0-32
	MOVD dst+0(FP), R0
	MOVD src+8(FP), R1
	MOVD dstStride+16(FP), R2
	MOVD srcStride+24(FP), R3
	MOVW $8, R4
qpel8_put20_row:
	MOVD R0, R10
	MOVD R1, R11
	MOVW $8, R9
qpel8_put20_col:
	MOVBU (R11), R5
	MOVBU 1(R11), R6
	ADDW  R6, R5, R5
	LSLW  $4, R5, R12
	ADDW  R5<<2, R12, R12
	MOVBU -1(R11), R5
	MOVBU 2(R11), R6
	ADDW  R6, R5, R5
	ADDW  R5<<2, R5, R5
	SUBW  R5, R12, R12
	MOVBU -2(R11), R5
	MOVBU 3(R11), R6
	ADDW  R6, R5, R5
	ADDW  R5, R12, R12
	ADDW  $16, R12, R12
	ASRW  $5, R12, R12
	CMPW  $0, R12
	BGE   qpel8_put20_nonnegative
	MOVW  ZR, R12
	B     qpel8_put20_store
qpel8_put20_nonnegative:
	CMPW  $255, R12
	BLE   qpel8_put20_store
	MOVW  $255, R12
qpel8_put20_store:
	MOVB  R12, (R10)
	ADD   $1, R10, R10
	ADD   $1, R11, R11
	SUBW  $1, R9, R9
	CBNZW R9, qpel8_put20_col
	ADD   R2, R0, R0
	ADD   R3, R1, R1
	SUBW  $1, R4, R4
	CBNZW R4, qpel8_put20_row
	RET

// func h264QpelMC8Avg20ASM(dst *uint8, src *uint8, dstStride int, srcStride int)
TEXT ·h264QpelMC8Avg20ASM(SB), NOSPLIT, $0-32
	MOVD dst+0(FP), R0
	MOVD src+8(FP), R1
	MOVD dstStride+16(FP), R2
	MOVD srcStride+24(FP), R3
	MOVW $8, R4
qpel8_avg20_row:
	MOVD R0, R10
	MOVD R1, R11
	MOVW $8, R9
qpel8_avg20_col:
	MOVBU (R11), R5
	MOVBU 1(R11), R6
	ADDW  R6, R5, R5
	LSLW  $4, R5, R12
	ADDW  R5<<2, R12, R12
	MOVBU -1(R11), R5
	MOVBU 2(R11), R6
	ADDW  R6, R5, R5
	ADDW  R5<<2, R5, R5
	SUBW  R5, R12, R12
	MOVBU -2(R11), R5
	MOVBU 3(R11), R6
	ADDW  R6, R5, R5
	ADDW  R5, R12, R12
	ADDW  $16, R12, R12
	ASRW  $5, R12, R12
	CMPW  $0, R12
	BGE   qpel8_avg20_nonnegative
	MOVW  ZR, R12
	B     qpel8_avg20_clip_done
qpel8_avg20_nonnegative:
	CMPW  $255, R12
	BLE   qpel8_avg20_clip_done
	MOVW  $255, R12
qpel8_avg20_clip_done:
	MOVBU (R10), R7
	ADDW  R7, R12, R12
	ADDW  $1, R12, R12
	LSRW  $1, R12, R12
	MOVB  R12, (R10)
	ADD   $1, R10, R10
	ADD   $1, R11, R11
	SUBW  $1, R9, R9
	CBNZW R9, qpel8_avg20_col
	ADD   R2, R0, R0
	ADD   R3, R1, R1
	SUBW  $1, R4, R4
	CBNZW R4, qpel8_avg20_row
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
