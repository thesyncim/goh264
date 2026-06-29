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

// func h264QpelMC16Put10ASM(dst *uint8, src *uint8, dstStride int, srcStride int)
TEXT ·h264QpelMC16Put10ASM(SB), NOSPLIT, $0-32
	MOVD dst+0(FP), R0
	MOVD src+8(FP), R1
	MOVD dstStride+16(FP), R2
	MOVD srcStride+24(FP), R3
	MOVW $16, R4
qpel16_put10_row:
	MOVD R0, R10
	MOVD R1, R11
	MOVW $16, R9
qpel16_put10_col:
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
	BGE   qpel16_put10_nonnegative
	MOVW  ZR, R12
	B     qpel16_put10_clip_done
qpel16_put10_nonnegative:
	CMPW  $255, R12
	BLE   qpel16_put10_clip_done
	MOVW  $255, R12
qpel16_put10_clip_done:
	MOVBU (R11), R7
	ADDW  R7, R12, R12
	ADDW  $1, R12, R12
	LSRW  $1, R12, R12
	MOVB  R12, (R10)
	ADD   $1, R10, R10
	ADD   $1, R11, R11
	SUBW  $1, R9, R9
	CBNZW R9, qpel16_put10_col
	ADD   R2, R0, R0
	ADD   R3, R1, R1
	SUBW  $1, R4, R4
	CBNZW R4, qpel16_put10_row
	RET

// func h264QpelMC16Avg10ASM(dst *uint8, src *uint8, dstStride int, srcStride int)
TEXT ·h264QpelMC16Avg10ASM(SB), NOSPLIT, $0-32
	MOVD dst+0(FP), R0
	MOVD src+8(FP), R1
	MOVD dstStride+16(FP), R2
	MOVD srcStride+24(FP), R3
	MOVW $16, R4
qpel16_avg10_row:
	MOVD R0, R10
	MOVD R1, R11
	MOVW $16, R9
qpel16_avg10_col:
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
	BGE   qpel16_avg10_nonnegative
	MOVW  ZR, R12
	B     qpel16_avg10_clip_done
qpel16_avg10_nonnegative:
	CMPW  $255, R12
	BLE   qpel16_avg10_clip_done
	MOVW  $255, R12
qpel16_avg10_clip_done:
	MOVBU (R11), R7
	ADDW  R7, R12, R12
	ADDW  $1, R12, R12
	LSRW  $1, R12, R12
	MOVBU (R10), R7
	ADDW  R7, R12, R12
	ADDW  $1, R12, R12
	LSRW  $1, R12, R12
	MOVB  R12, (R10)
	ADD   $1, R10, R10
	ADD   $1, R11, R11
	SUBW  $1, R9, R9
	CBNZW R9, qpel16_avg10_col
	ADD   R2, R0, R0
	ADD   R3, R1, R1
	SUBW  $1, R4, R4
	CBNZW R4, qpel16_avg10_row
	RET

// func h264QpelMC8Put10ASM(dst *uint8, src *uint8, dstStride int, srcStride int)
TEXT ·h264QpelMC8Put10ASM(SB), NOSPLIT, $0-32
	MOVD dst+0(FP), R0
	MOVD src+8(FP), R1
	MOVD dstStride+16(FP), R2
	MOVD srcStride+24(FP), R3
	MOVW $8, R4
qpel8_put10_row:
	MOVD R0, R10
	MOVD R1, R11
	MOVW $8, R9
qpel8_put10_col:
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
	BGE   qpel8_put10_nonnegative
	MOVW  ZR, R12
	B     qpel8_put10_clip_done
qpel8_put10_nonnegative:
	CMPW  $255, R12
	BLE   qpel8_put10_clip_done
	MOVW  $255, R12
qpel8_put10_clip_done:
	MOVBU (R11), R7
	ADDW  R7, R12, R12
	ADDW  $1, R12, R12
	LSRW  $1, R12, R12
	MOVB  R12, (R10)
	ADD   $1, R10, R10
	ADD   $1, R11, R11
	SUBW  $1, R9, R9
	CBNZW R9, qpel8_put10_col
	ADD   R2, R0, R0
	ADD   R3, R1, R1
	SUBW  $1, R4, R4
	CBNZW R4, qpel8_put10_row
	RET

// func h264QpelMC8Avg10ASM(dst *uint8, src *uint8, dstStride int, srcStride int)
TEXT ·h264QpelMC8Avg10ASM(SB), NOSPLIT, $0-32
	MOVD dst+0(FP), R0
	MOVD src+8(FP), R1
	MOVD dstStride+16(FP), R2
	MOVD srcStride+24(FP), R3
	MOVW $8, R4
qpel8_avg10_row:
	MOVD R0, R10
	MOVD R1, R11
	MOVW $8, R9
qpel8_avg10_col:
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
	BGE   qpel8_avg10_nonnegative
	MOVW  ZR, R12
	B     qpel8_avg10_clip_done
qpel8_avg10_nonnegative:
	CMPW  $255, R12
	BLE   qpel8_avg10_clip_done
	MOVW  $255, R12
qpel8_avg10_clip_done:
	MOVBU (R11), R7
	ADDW  R7, R12, R12
	ADDW  $1, R12, R12
	LSRW  $1, R12, R12
	MOVBU (R10), R7
	ADDW  R7, R12, R12
	ADDW  $1, R12, R12
	LSRW  $1, R12, R12
	MOVB  R12, (R10)
	ADD   $1, R10, R10
	ADD   $1, R11, R11
	SUBW  $1, R9, R9
	CBNZW R9, qpel8_avg10_col
	ADD   R2, R0, R0
	ADD   R3, R1, R1
	SUBW  $1, R4, R4
	CBNZW R4, qpel8_avg10_row
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

// func h264QpelMC16Put30ASM(dst *uint8, src *uint8, dstStride int, srcStride int)
TEXT ·h264QpelMC16Put30ASM(SB), NOSPLIT, $0-32
	MOVD dst+0(FP), R0
	MOVD src+8(FP), R1
	MOVD dstStride+16(FP), R2
	MOVD srcStride+24(FP), R3
	MOVW $16, R4
qpel16_put30_row:
	MOVD R0, R10
	MOVD R1, R11
	MOVW $16, R9
qpel16_put30_col:
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
	BGE   qpel16_put30_nonnegative
	MOVW  ZR, R12
	B     qpel16_put30_clip_done
qpel16_put30_nonnegative:
	CMPW  $255, R12
	BLE   qpel16_put30_clip_done
	MOVW  $255, R12
qpel16_put30_clip_done:
	MOVBU 1(R11), R7
	ADDW  R7, R12, R12
	ADDW  $1, R12, R12
	LSRW  $1, R12, R12
	MOVB  R12, (R10)
	ADD   $1, R10, R10
	ADD   $1, R11, R11
	SUBW  $1, R9, R9
	CBNZW R9, qpel16_put30_col
	ADD   R2, R0, R0
	ADD   R3, R1, R1
	SUBW  $1, R4, R4
	CBNZW R4, qpel16_put30_row
	RET

// func h264QpelMC16Avg30ASM(dst *uint8, src *uint8, dstStride int, srcStride int)
TEXT ·h264QpelMC16Avg30ASM(SB), NOSPLIT, $0-32
	MOVD dst+0(FP), R0
	MOVD src+8(FP), R1
	MOVD dstStride+16(FP), R2
	MOVD srcStride+24(FP), R3
	MOVW $16, R4
qpel16_avg30_row:
	MOVD R0, R10
	MOVD R1, R11
	MOVW $16, R9
qpel16_avg30_col:
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
	BGE   qpel16_avg30_nonnegative
	MOVW  ZR, R12
	B     qpel16_avg30_clip_done
qpel16_avg30_nonnegative:
	CMPW  $255, R12
	BLE   qpel16_avg30_clip_done
	MOVW  $255, R12
qpel16_avg30_clip_done:
	MOVBU 1(R11), R7
	ADDW  R7, R12, R12
	ADDW  $1, R12, R12
	LSRW  $1, R12, R12
	MOVBU (R10), R7
	ADDW  R7, R12, R12
	ADDW  $1, R12, R12
	LSRW  $1, R12, R12
	MOVB  R12, (R10)
	ADD   $1, R10, R10
	ADD   $1, R11, R11
	SUBW  $1, R9, R9
	CBNZW R9, qpel16_avg30_col
	ADD   R2, R0, R0
	ADD   R3, R1, R1
	SUBW  $1, R4, R4
	CBNZW R4, qpel16_avg30_row
	RET

// func h264QpelMC8Put30ASM(dst *uint8, src *uint8, dstStride int, srcStride int)
TEXT ·h264QpelMC8Put30ASM(SB), NOSPLIT, $0-32
	MOVD dst+0(FP), R0
	MOVD src+8(FP), R1
	MOVD dstStride+16(FP), R2
	MOVD srcStride+24(FP), R3
	MOVW $8, R4
qpel8_put30_row:
	MOVD R0, R10
	MOVD R1, R11
	MOVW $8, R9
qpel8_put30_col:
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
	BGE   qpel8_put30_nonnegative
	MOVW  ZR, R12
	B     qpel8_put30_clip_done
qpel8_put30_nonnegative:
	CMPW  $255, R12
	BLE   qpel8_put30_clip_done
	MOVW  $255, R12
qpel8_put30_clip_done:
	MOVBU 1(R11), R7
	ADDW  R7, R12, R12
	ADDW  $1, R12, R12
	LSRW  $1, R12, R12
	MOVB  R12, (R10)
	ADD   $1, R10, R10
	ADD   $1, R11, R11
	SUBW  $1, R9, R9
	CBNZW R9, qpel8_put30_col
	ADD   R2, R0, R0
	ADD   R3, R1, R1
	SUBW  $1, R4, R4
	CBNZW R4, qpel8_put30_row
	RET

// func h264QpelMC8Avg30ASM(dst *uint8, src *uint8, dstStride int, srcStride int)
TEXT ·h264QpelMC8Avg30ASM(SB), NOSPLIT, $0-32
	MOVD dst+0(FP), R0
	MOVD src+8(FP), R1
	MOVD dstStride+16(FP), R2
	MOVD srcStride+24(FP), R3
	MOVW $8, R4
qpel8_avg30_row:
	MOVD R0, R10
	MOVD R1, R11
	MOVW $8, R9
qpel8_avg30_col:
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
	BGE   qpel8_avg30_nonnegative
	MOVW  ZR, R12
	B     qpel8_avg30_clip_done
qpel8_avg30_nonnegative:
	CMPW  $255, R12
	BLE   qpel8_avg30_clip_done
	MOVW  $255, R12
qpel8_avg30_clip_done:
	MOVBU 1(R11), R7
	ADDW  R7, R12, R12
	ADDW  $1, R12, R12
	LSRW  $1, R12, R12
	MOVBU (R10), R7
	ADDW  R7, R12, R12
	ADDW  $1, R12, R12
	LSRW  $1, R12, R12
	MOVB  R12, (R10)
	ADD   $1, R10, R10
	ADD   $1, R11, R11
	SUBW  $1, R9, R9
	CBNZW R9, qpel8_avg30_col
	ADD   R2, R0, R0
	ADD   R3, R1, R1
	SUBW  $1, R4, R4
	CBNZW R4, qpel8_avg30_row
	RET

// func h264QpelMCPut0YASM(dst *uint8, src *uint8, dstStride int, srcStride int, size int32, my int32)
TEXT ·h264QpelMCPut0YASM(SB), NOSPLIT, $0-40
	MOVD dst+0(FP), R0
	MOVD src+8(FP), R1
	MOVD dstStride+16(FP), R2
	MOVD srcStride+24(FP), R3
	MOVW size+32(FP), R4
	MOVW my+36(FP), R14
qpel_put0y_row:
	MOVD R0, R10
	MOVD R1, R11
	MOVW size+32(FP), R9
qpel_put0y_col:
	MOVBU (R11), R5
	ADD   R3, R11, R13
	MOVBU (R13), R6
	ADDW  R6, R5, R5
	LSLW  $4, R5, R12
	ADDW  R5<<2, R12, R12
	SUB   R3, R11, R13
	MOVBU (R13), R5
	ADD   R3, R11, R13
	ADD   R3, R13, R13
	MOVBU (R13), R6
	ADDW  R6, R5, R5
	ADDW  R5<<2, R5, R5
	SUBW  R5, R12, R12
	SUB   R3, R11, R13
	SUB   R3, R13, R13
	MOVBU (R13), R5
	ADD   R3, R11, R13
	ADD   R3, R13, R13
	ADD   R3, R13, R13
	MOVBU (R13), R6
	ADDW  R6, R5, R5
	ADDW  R5, R12, R12
	ADDW  $16, R12, R12
	ASRW  $5, R12, R12
	CMPW  $0, R12
	BGE   qpel_put0y_nonnegative
	MOVW  ZR, R12
	B     qpel_put0y_clip_done
qpel_put0y_nonnegative:
	CMPW  $255, R12
	BLE   qpel_put0y_clip_done
	MOVW  $255, R12
qpel_put0y_clip_done:
	CMPW  $2, R14
	BEQ   qpel_put0y_store
	CMPW  $1, R14
	BNE   qpel_put0y_load_next
	MOVBU (R11), R7
	B     qpel_put0y_l2
qpel_put0y_load_next:
	ADD   R3, R11, R13
	MOVBU (R13), R7
qpel_put0y_l2:
	ADDW  R7, R12, R12
	ADDW  $1, R12, R12
	LSRW  $1, R12, R12
qpel_put0y_store:
	MOVB  R12, (R10)
	ADD   $1, R10, R10
	ADD   $1, R11, R11
	SUBW  $1, R9, R9
	CBNZW R9, qpel_put0y_col
	ADD   R2, R0, R0
	ADD   R3, R1, R1
	SUBW  $1, R4, R4
	CBNZW R4, qpel_put0y_row
	RET

// func h264QpelMCAvg0YASM(dst *uint8, src *uint8, dstStride int, srcStride int, size int32, my int32)
TEXT ·h264QpelMCAvg0YASM(SB), NOSPLIT, $0-40
	MOVD dst+0(FP), R0
	MOVD src+8(FP), R1
	MOVD dstStride+16(FP), R2
	MOVD srcStride+24(FP), R3
	MOVW size+32(FP), R4
	MOVW my+36(FP), R14
qpel_avg0y_row:
	MOVD R0, R10
	MOVD R1, R11
	MOVW size+32(FP), R9
qpel_avg0y_col:
	MOVBU (R11), R5
	ADD   R3, R11, R13
	MOVBU (R13), R6
	ADDW  R6, R5, R5
	LSLW  $4, R5, R12
	ADDW  R5<<2, R12, R12
	SUB   R3, R11, R13
	MOVBU (R13), R5
	ADD   R3, R11, R13
	ADD   R3, R13, R13
	MOVBU (R13), R6
	ADDW  R6, R5, R5
	ADDW  R5<<2, R5, R5
	SUBW  R5, R12, R12
	SUB   R3, R11, R13
	SUB   R3, R13, R13
	MOVBU (R13), R5
	ADD   R3, R11, R13
	ADD   R3, R13, R13
	ADD   R3, R13, R13
	MOVBU (R13), R6
	ADDW  R6, R5, R5
	ADDW  R5, R12, R12
	ADDW  $16, R12, R12
	ASRW  $5, R12, R12
	CMPW  $0, R12
	BGE   qpel_avg0y_nonnegative
	MOVW  ZR, R12
	B     qpel_avg0y_clip_done
qpel_avg0y_nonnegative:
	CMPW  $255, R12
	BLE   qpel_avg0y_clip_done
	MOVW  $255, R12
qpel_avg0y_clip_done:
	CMPW  $2, R14
	BEQ   qpel_avg0y_pred_done
	CMPW  $1, R14
	BNE   qpel_avg0y_load_next
	MOVBU (R11), R7
	B     qpel_avg0y_l2
qpel_avg0y_load_next:
	ADD   R3, R11, R13
	MOVBU (R13), R7
qpel_avg0y_l2:
	ADDW  R7, R12, R12
	ADDW  $1, R12, R12
	LSRW  $1, R12, R12
qpel_avg0y_pred_done:
	MOVBU (R10), R7
	ADDW  R7, R12, R12
	ADDW  $1, R12, R12
	LSRW  $1, R12, R12
	MOVB  R12, (R10)
	ADD   $1, R10, R10
	ADD   $1, R11, R11
	SUBW  $1, R9, R9
	CBNZW R9, qpel_avg0y_col
	ADD   R2, R0, R0
	ADD   R3, R1, R1
	SUBW  $1, R4, R4
	CBNZW R4, qpel_avg0y_row
	RET

// func h264QpelMCPut22ASM(dst *uint8, src *uint8, dstStride int, srcStride int, size int32)
TEXT ·h264QpelMCPut22ASM(SB), NOSPLIT, $32-40
	MOVD dst+0(FP), R0
	MOVD src+8(FP), R1
	MOVD dstStride+16(FP), R2
	MOVD srcStride+24(FP), R3
	MOVW size+32(FP), R4
qpel_put22_row:
	MOVD R0, R10
	MOVD R1, R11
	MOVW size+32(FP), R9
qpel_put22_col:
	SUB   R3, R11, R13
	SUB   R3, R13, R13
	MOVBU (R13), R5
	MOVBU 1(R13), R6
	ADDW  R6, R5, R5
	LSLW  $4, R5, R12
	ADDW  R5<<2, R12, R12
	MOVBU -1(R13), R5
	MOVBU 2(R13), R6
	ADDW  R6, R5, R5
	ADDW  R5<<2, R5, R5
	SUBW  R5, R12, R12
	MOVBU -2(R13), R5
	MOVBU 3(R13), R6
	ADDW  R6, R5, R5
	ADDW  R5, R12, R12
	MOVW  R12, tmp0-32(SP)
	SUB   R3, R11, R13
	MOVBU (R13), R5
	MOVBU 1(R13), R6
	ADDW  R6, R5, R5
	LSLW  $4, R5, R12
	ADDW  R5<<2, R12, R12
	MOVBU -1(R13), R5
	MOVBU 2(R13), R6
	ADDW  R6, R5, R5
	ADDW  R5<<2, R5, R5
	SUBW  R5, R12, R12
	MOVBU -2(R13), R5
	MOVBU 3(R13), R6
	ADDW  R6, R5, R5
	ADDW  R5, R12, R12
	MOVW  R12, tmp1-28(SP)
	MOVD  R11, R13
	MOVBU (R13), R5
	MOVBU 1(R13), R6
	ADDW  R6, R5, R5
	LSLW  $4, R5, R12
	ADDW  R5<<2, R12, R12
	MOVBU -1(R13), R5
	MOVBU 2(R13), R6
	ADDW  R6, R5, R5
	ADDW  R5<<2, R5, R5
	SUBW  R5, R12, R12
	MOVBU -2(R13), R5
	MOVBU 3(R13), R6
	ADDW  R6, R5, R5
	ADDW  R5, R12, R12
	MOVW  R12, tmp2-24(SP)
	ADD   R3, R11, R13
	MOVBU (R13), R5
	MOVBU 1(R13), R6
	ADDW  R6, R5, R5
	LSLW  $4, R5, R12
	ADDW  R5<<2, R12, R12
	MOVBU -1(R13), R5
	MOVBU 2(R13), R6
	ADDW  R6, R5, R5
	ADDW  R5<<2, R5, R5
	SUBW  R5, R12, R12
	MOVBU -2(R13), R5
	MOVBU 3(R13), R6
	ADDW  R6, R5, R5
	ADDW  R5, R12, R12
	MOVW  R12, tmp3-20(SP)
	ADD   R3, R11, R13
	ADD   R3, R13, R13
	MOVBU (R13), R5
	MOVBU 1(R13), R6
	ADDW  R6, R5, R5
	LSLW  $4, R5, R12
	ADDW  R5<<2, R12, R12
	MOVBU -1(R13), R5
	MOVBU 2(R13), R6
	ADDW  R6, R5, R5
	ADDW  R5<<2, R5, R5
	SUBW  R5, R12, R12
	MOVBU -2(R13), R5
	MOVBU 3(R13), R6
	ADDW  R6, R5, R5
	ADDW  R5, R12, R12
	MOVW  R12, tmp4-16(SP)
	ADD   R3, R11, R13
	ADD   R3, R13, R13
	ADD   R3, R13, R13
	MOVBU (R13), R5
	MOVBU 1(R13), R6
	ADDW  R6, R5, R5
	LSLW  $4, R5, R12
	ADDW  R5<<2, R12, R12
	MOVBU -1(R13), R5
	MOVBU 2(R13), R6
	ADDW  R6, R5, R5
	ADDW  R5<<2, R5, R5
	SUBW  R5, R12, R12
	MOVBU -2(R13), R5
	MOVBU 3(R13), R6
	ADDW  R6, R5, R5
	ADDW  R5, R12, R12
	MOVW  R12, tmp5-12(SP)
	MOVW  tmp2-24(SP), R5
	MOVW  tmp3-20(SP), R6
	ADDW  R6, R5, R5
	LSLW  $4, R5, R12
	ADDW  R5<<2, R12, R12
	MOVW  tmp1-28(SP), R5
	MOVW  tmp4-16(SP), R6
	ADDW  R6, R5, R5
	ADDW  R5<<2, R5, R5
	SUBW  R5, R12, R12
	MOVW  tmp0-32(SP), R5
	MOVW  tmp5-12(SP), R6
	ADDW  R6, R5, R5
	ADDW  R5, R12, R12
	ADDW  $512, R12, R12
	ASRW  $10, R12, R12
	CMPW  $0, R12
	BGE   qpel_put22_nonnegative
	MOVW  ZR, R12
	B     qpel_put22_store
qpel_put22_nonnegative:
	CMPW  $255, R12
	BLE   qpel_put22_store
	MOVW  $255, R12
qpel_put22_store:
	MOVB  R12, (R10)
	ADD   $1, R10, R10
	ADD   $1, R11, R11
	SUBW  $1, R9, R9
	CBNZW R9, qpel_put22_col
	ADD   R2, R0, R0
	ADD   R3, R1, R1
	SUBW  $1, R4, R4
	CBNZW R4, qpel_put22_row
	RET

// func h264QpelMCAvg22ASM(dst *uint8, src *uint8, dstStride int, srcStride int, size int32)
TEXT ·h264QpelMCAvg22ASM(SB), NOSPLIT, $32-40
	MOVD dst+0(FP), R0
	MOVD src+8(FP), R1
	MOVD dstStride+16(FP), R2
	MOVD srcStride+24(FP), R3
	MOVW size+32(FP), R4
qpel_avg22_row:
	MOVD R0, R10
	MOVD R1, R11
	MOVW size+32(FP), R9
qpel_avg22_col:
	SUB   R3, R11, R13
	SUB   R3, R13, R13
	MOVBU (R13), R5
	MOVBU 1(R13), R6
	ADDW  R6, R5, R5
	LSLW  $4, R5, R12
	ADDW  R5<<2, R12, R12
	MOVBU -1(R13), R5
	MOVBU 2(R13), R6
	ADDW  R6, R5, R5
	ADDW  R5<<2, R5, R5
	SUBW  R5, R12, R12
	MOVBU -2(R13), R5
	MOVBU 3(R13), R6
	ADDW  R6, R5, R5
	ADDW  R5, R12, R12
	MOVW  R12, tmp0-32(SP)
	SUB   R3, R11, R13
	MOVBU (R13), R5
	MOVBU 1(R13), R6
	ADDW  R6, R5, R5
	LSLW  $4, R5, R12
	ADDW  R5<<2, R12, R12
	MOVBU -1(R13), R5
	MOVBU 2(R13), R6
	ADDW  R6, R5, R5
	ADDW  R5<<2, R5, R5
	SUBW  R5, R12, R12
	MOVBU -2(R13), R5
	MOVBU 3(R13), R6
	ADDW  R6, R5, R5
	ADDW  R5, R12, R12
	MOVW  R12, tmp1-28(SP)
	MOVD  R11, R13
	MOVBU (R13), R5
	MOVBU 1(R13), R6
	ADDW  R6, R5, R5
	LSLW  $4, R5, R12
	ADDW  R5<<2, R12, R12
	MOVBU -1(R13), R5
	MOVBU 2(R13), R6
	ADDW  R6, R5, R5
	ADDW  R5<<2, R5, R5
	SUBW  R5, R12, R12
	MOVBU -2(R13), R5
	MOVBU 3(R13), R6
	ADDW  R6, R5, R5
	ADDW  R5, R12, R12
	MOVW  R12, tmp2-24(SP)
	ADD   R3, R11, R13
	MOVBU (R13), R5
	MOVBU 1(R13), R6
	ADDW  R6, R5, R5
	LSLW  $4, R5, R12
	ADDW  R5<<2, R12, R12
	MOVBU -1(R13), R5
	MOVBU 2(R13), R6
	ADDW  R6, R5, R5
	ADDW  R5<<2, R5, R5
	SUBW  R5, R12, R12
	MOVBU -2(R13), R5
	MOVBU 3(R13), R6
	ADDW  R6, R5, R5
	ADDW  R5, R12, R12
	MOVW  R12, tmp3-20(SP)
	ADD   R3, R11, R13
	ADD   R3, R13, R13
	MOVBU (R13), R5
	MOVBU 1(R13), R6
	ADDW  R6, R5, R5
	LSLW  $4, R5, R12
	ADDW  R5<<2, R12, R12
	MOVBU -1(R13), R5
	MOVBU 2(R13), R6
	ADDW  R6, R5, R5
	ADDW  R5<<2, R5, R5
	SUBW  R5, R12, R12
	MOVBU -2(R13), R5
	MOVBU 3(R13), R6
	ADDW  R6, R5, R5
	ADDW  R5, R12, R12
	MOVW  R12, tmp4-16(SP)
	ADD   R3, R11, R13
	ADD   R3, R13, R13
	ADD   R3, R13, R13
	MOVBU (R13), R5
	MOVBU 1(R13), R6
	ADDW  R6, R5, R5
	LSLW  $4, R5, R12
	ADDW  R5<<2, R12, R12
	MOVBU -1(R13), R5
	MOVBU 2(R13), R6
	ADDW  R6, R5, R5
	ADDW  R5<<2, R5, R5
	SUBW  R5, R12, R12
	MOVBU -2(R13), R5
	MOVBU 3(R13), R6
	ADDW  R6, R5, R5
	ADDW  R5, R12, R12
	MOVW  R12, tmp5-12(SP)
	MOVW  tmp2-24(SP), R5
	MOVW  tmp3-20(SP), R6
	ADDW  R6, R5, R5
	LSLW  $4, R5, R12
	ADDW  R5<<2, R12, R12
	MOVW  tmp1-28(SP), R5
	MOVW  tmp4-16(SP), R6
	ADDW  R6, R5, R5
	ADDW  R5<<2, R5, R5
	SUBW  R5, R12, R12
	MOVW  tmp0-32(SP), R5
	MOVW  tmp5-12(SP), R6
	ADDW  R6, R5, R5
	ADDW  R5, R12, R12
	ADDW  $512, R12, R12
	ASRW  $10, R12, R12
	CMPW  $0, R12
	BGE   qpel_avg22_nonnegative
	MOVW  ZR, R12
	B     qpel_avg22_clip_done
qpel_avg22_nonnegative:
	CMPW  $255, R12
	BLE   qpel_avg22_clip_done
	MOVW  $255, R12
qpel_avg22_clip_done:
	MOVBU (R10), R7
	ADDW  R7, R12, R12
	ADDW  $1, R12, R12
	LSRW  $1, R12, R12
	MOVB  R12, (R10)
	ADD   $1, R10, R10
	ADD   $1, R11, R11
	SUBW  $1, R9, R9
	CBNZW R9, qpel_avg22_col
	ADD   R2, R0, R0
	ADD   R3, R1, R1
	SUBW  $1, R4, R4
	CBNZW R4, qpel_avg22_row
	RET

// func h264QpelMCPutHVXYASM(dst *uint8, src *uint8, dstStride int, srcStride int, size int32, mx int32, my int32)
TEXT ·h264QpelMCPutHVXYASM(SB), NOSPLIT, $16-48
	MOVD dst+0(FP), R0
	MOVD src+8(FP), R1
	MOVD dstStride+16(FP), R2
	MOVD srcStride+24(FP), R3
	MOVW size+32(FP), R4
qpel_puthvxy_row:
	MOVD R0, R10
	MOVD R1, R11
	MOVW size+32(FP), R9
qpel_puthvxy_col:
	MOVD  R11, R13
	MOVW  my+40(FP), R14
	CMPW  $3, R14
	BNE   qpel_puthvxy_hptr_ready
	ADD   R3, R13, R13
qpel_puthvxy_hptr_ready:
	MOVBU (R13), R5
	MOVBU 1(R13), R6
	ADDW  R6, R5, R5
	LSLW  $4, R5, R12
	ADDW  R5<<2, R12, R12
	MOVBU -1(R13), R5
	MOVBU 2(R13), R6
	ADDW  R6, R5, R5
	ADDW  R5<<2, R5, R5
	SUBW  R5, R12, R12
	MOVBU -2(R13), R5
	MOVBU 3(R13), R6
	ADDW  R6, R5, R5
	ADDW  R5, R12, R12
	ADDW  $16, R12, R12
	ASRW  $5, R12, R12
	CMPW  $0, R12
	BGE   qpel_puthvxy_h_nonnegative
	MOVW  ZR, R12
	B     qpel_puthvxy_h_done
qpel_puthvxy_h_nonnegative:
	CMPW  $255, R12
	BLE   qpel_puthvxy_h_done
	MOVW  $255, R12
qpel_puthvxy_h_done:
	MOVW  R12, htmp-16(SP)
	MOVD  R11, R13
	MOVW  mx+36(FP), R14
	CMPW  $3, R14
	BNE   qpel_puthvxy_vptr_ready
	ADD   $1, R13, R13
qpel_puthvxy_vptr_ready:
	MOVBU (R13), R5
	ADD   R3, R13, R14
	MOVBU (R14), R6
	ADDW  R6, R5, R5
	LSLW  $4, R5, R12
	ADDW  R5<<2, R12, R12
	SUB   R3, R13, R14
	MOVBU (R14), R5
	ADD   R3, R13, R14
	ADD   R3, R14, R14
	MOVBU (R14), R6
	ADDW  R6, R5, R5
	ADDW  R5<<2, R5, R5
	SUBW  R5, R12, R12
	SUB   R3, R13, R14
	SUB   R3, R14, R14
	MOVBU (R14), R5
	ADD   R3, R13, R14
	ADD   R3, R14, R14
	ADD   R3, R14, R14
	MOVBU (R14), R6
	ADDW  R6, R5, R5
	ADDW  R5, R12, R12
	ADDW  $16, R12, R12
	ASRW  $5, R12, R12
	CMPW  $0, R12
	BGE   qpel_puthvxy_v_nonnegative
	MOVW  ZR, R12
	B     qpel_puthvxy_v_done
qpel_puthvxy_v_nonnegative:
	CMPW  $255, R12
	BLE   qpel_puthvxy_v_done
	MOVW  $255, R12
qpel_puthvxy_v_done:
	MOVW  htmp-16(SP), R5
	ADDW  R5, R12, R12
	ADDW  $1, R12, R12
	LSRW  $1, R12, R12
	MOVB  R12, (R10)
	ADD   $1, R10, R10
	ADD   $1, R11, R11
	SUBW  $1, R9, R9
	CBNZW R9, qpel_puthvxy_col
	ADD   R2, R0, R0
	ADD   R3, R1, R1
	SUBW  $1, R4, R4
	CBNZW R4, qpel_puthvxy_row
	RET

// func h264QpelMCAvgHVXYASM(dst *uint8, src *uint8, dstStride int, srcStride int, size int32, mx int32, my int32)
TEXT ·h264QpelMCAvgHVXYASM(SB), NOSPLIT, $16-48
	MOVD dst+0(FP), R0
	MOVD src+8(FP), R1
	MOVD dstStride+16(FP), R2
	MOVD srcStride+24(FP), R3
	MOVW size+32(FP), R4
qpel_avghvxy_row:
	MOVD R0, R10
	MOVD R1, R11
	MOVW size+32(FP), R9
qpel_avghvxy_col:
	MOVD  R11, R13
	MOVW  my+40(FP), R14
	CMPW  $3, R14
	BNE   qpel_avghvxy_hptr_ready
	ADD   R3, R13, R13
qpel_avghvxy_hptr_ready:
	MOVBU (R13), R5
	MOVBU 1(R13), R6
	ADDW  R6, R5, R5
	LSLW  $4, R5, R12
	ADDW  R5<<2, R12, R12
	MOVBU -1(R13), R5
	MOVBU 2(R13), R6
	ADDW  R6, R5, R5
	ADDW  R5<<2, R5, R5
	SUBW  R5, R12, R12
	MOVBU -2(R13), R5
	MOVBU 3(R13), R6
	ADDW  R6, R5, R5
	ADDW  R5, R12, R12
	ADDW  $16, R12, R12
	ASRW  $5, R12, R12
	CMPW  $0, R12
	BGE   qpel_avghvxy_h_nonnegative
	MOVW  ZR, R12
	B     qpel_avghvxy_h_done
qpel_avghvxy_h_nonnegative:
	CMPW  $255, R12
	BLE   qpel_avghvxy_h_done
	MOVW  $255, R12
qpel_avghvxy_h_done:
	MOVW  R12, htmp-16(SP)
	MOVD  R11, R13
	MOVW  mx+36(FP), R14
	CMPW  $3, R14
	BNE   qpel_avghvxy_vptr_ready
	ADD   $1, R13, R13
qpel_avghvxy_vptr_ready:
	MOVBU (R13), R5
	ADD   R3, R13, R14
	MOVBU (R14), R6
	ADDW  R6, R5, R5
	LSLW  $4, R5, R12
	ADDW  R5<<2, R12, R12
	SUB   R3, R13, R14
	MOVBU (R14), R5
	ADD   R3, R13, R14
	ADD   R3, R14, R14
	MOVBU (R14), R6
	ADDW  R6, R5, R5
	ADDW  R5<<2, R5, R5
	SUBW  R5, R12, R12
	SUB   R3, R13, R14
	SUB   R3, R14, R14
	MOVBU (R14), R5
	ADD   R3, R13, R14
	ADD   R3, R14, R14
	ADD   R3, R14, R14
	MOVBU (R14), R6
	ADDW  R6, R5, R5
	ADDW  R5, R12, R12
	ADDW  $16, R12, R12
	ASRW  $5, R12, R12
	CMPW  $0, R12
	BGE   qpel_avghvxy_v_nonnegative
	MOVW  ZR, R12
	B     qpel_avghvxy_v_done
qpel_avghvxy_v_nonnegative:
	CMPW  $255, R12
	BLE   qpel_avghvxy_v_done
	MOVW  $255, R12
qpel_avghvxy_v_done:
	MOVW  htmp-16(SP), R5
	ADDW  R5, R12, R12
	ADDW  $1, R12, R12
	LSRW  $1, R12, R12
	MOVBU (R10), R7
	ADDW  R7, R12, R12
	ADDW  $1, R12, R12
	LSRW  $1, R12, R12
	MOVB  R12, (R10)
	ADD   $1, R10, R10
	ADD   $1, R11, R11
	SUBW  $1, R9, R9
	CBNZW R9, qpel_avghvxy_col
	ADD   R2, R0, R0
	ADD   R3, R1, R1
	SUBW  $1, R4, R4
	CBNZW R4, qpel_avghvxy_row
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
