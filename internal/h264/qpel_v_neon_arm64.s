// SPDX-License-Identifier: LGPL-2.1-or-later
//go:build !purego && arm64

#include "textflag.h"

// The width-8 core mirrors the pinned FFmpeg n8.0.1 vertical qpel NEON
// arithmetic. R0/R1 are destination and source-minus-two-rows, R2/R3 are
// independent destination/source strides, R4 is my, R5 is the avg flag, and
// R12 is the quarter-pel base source.
TEXT ·h264QpelV8NEONInternal(SB), NOSPLIT|NOFRAME, $0-0
	WORD $0x0cc37030
	WORD $0x0cc37031
	WORD $0x0cc37032
	WORD $0x0cc37033
	WORD $0x0cc37034
	WORD $0x0cc37035
	WORD $0x0cc37036
	WORD $0x0cc37037
	WORD $0x0cc37038
	WORD $0x0cc37039
	WORD $0x0cc3703a
	WORD $0x0cc3703b
	WORD $0x0c40703c
	WORD $0x2e330242
	WORD $0x2e340260
	WORD $0x2e340224
	WORD $0x2e350241
	WORD $0x2e350210
	WORD $0x2e360231
	WORD $0x6f560050
	WORD $0x6f464090
	WORD $0x6f560011
	WORD $0x6f464031
	WORD $0x2f0b8e10
	WORD $0x2f0b8e31
	WORD $0x2e350282
	WORD $0x2e3602a0
	WORD $0x2e360264
	WORD $0x2e370281
	WORD $0x2e370252
	WORD $0x2e380273
	WORD $0x6f560052
	WORD $0x6f464092
	WORD $0x6f560013
	WORD $0x6f464033
	WORD $0x2f0b8e52
	WORD $0x2f0b8e73
	WORD $0x2e3702c2
	WORD $0x2e3802e0
	WORD $0x2e3802a4
	WORD $0x2e3902c1
	WORD $0x2e390294
	WORD $0x2e3a02b5
	WORD $0x6f560054
	WORD $0x6f464094
	WORD $0x6f560015
	WORD $0x6f464035
	WORD $0x2f0b8e94
	WORD $0x2f0b8eb5
	WORD $0x2e390302
	WORD $0x2e3a0320
	WORD $0x2e3a02e4
	WORD $0x2e3b0301
	WORD $0x2e3b02d6
	WORD $0x2e3c02f7
	WORD $0x6f560056
	WORD $0x6f464096
	WORD $0x6f560017
	WORD $0x6f464037
	WORD $0x2f0b8ed6
	WORD $0x2f0b8ef7
	WORD $0x7100089f
	WORD $0x54000220
	WORD $0x0cc77198
	WORD $0x0cc77199
	WORD $0x0cc7719a
	WORD $0x0cc7719b
	WORD $0x0cc7719c
	WORD $0x2e301710
	WORD $0x2e311731
	WORD $0x0cc7719d
	WORD $0x2e321752
	WORD $0x2e331773
	WORD $0x0cc7719e
	WORD $0x2e341794
	WORD $0x2e3517b5
	WORD $0x0cc7719f
	WORD $0x2e3617d6
	WORD $0x2e3717f7
	WORD $0x34000245
	WORD $0x0cc27018
	WORD $0x0cc27019
	WORD $0x0cc2701a
	WORD $0x2e381610
	WORD $0x0cc2701b
	WORD $0x2e391631
	WORD $0x0cc2701c
	WORD $0x2e3a1652
	WORD $0x0cc2701d
	WORD $0x2e3b1673
	WORD $0x0cc2701e
	WORD $0x2e3c1694
	WORD $0x0cc2701f
	WORD $0x2e3d16b5
	WORD $0x2e3e16d6
	WORD $0x2e3f16f7
	WORD $0xcb020c00
	WORD $0x0c827010
	WORD $0x0c827011
	WORD $0x0c827012
	WORD $0x0c827013
	WORD $0x0c827014
	WORD $0x0c827015
	WORD $0x0c827016
	WORD $0x0c827017
	WORD $0xd65f03c0

// func h264QpelMCPut0YNEONASM(dst *uint8, src *uint8, dstStride int, srcStride int, size int32, my int32)
TEXT ·h264QpelMCPut0YNEONASM(SB), NOSPLIT|NOFRAME, $0-40
	MOVD dst+0(FP), R0
	MOVD src+8(FP), R1
	MOVD dstStride+16(FP), R2
	MOVD srcStride+24(FP), R3
	MOVW size+32(FP), R6
	MOVW my+36(FP), R4
	MOVW $0, R5
	JMP   ·h264QpelMC0YNEONInternal(SB)

// func h264QpelMCAvg0YNEONASM(dst *uint8, src *uint8, dstStride int, srcStride int, size int32, my int32)
TEXT ·h264QpelMCAvg0YNEONASM(SB), NOSPLIT|NOFRAME, $0-40
	MOVD dst+0(FP), R0
	MOVD src+8(FP), R1
	MOVD dstStride+16(FP), R2
	MOVD srcStride+24(FP), R3
	MOVW size+32(FP), R6
	MOVW my+36(FP), R4
	MOVW $1, R5
	JMP   ·h264QpelMC0YNEONInternal(SB)

TEXT ·h264QpelMC0YNEONInternal(SB), NOSPLIT|NOFRAME, $0-0
qpel_0y_neon_entry:
	MOVD R30, R14
	WORD $0x52a0028d // mov w13, #0x140000
	WORD $0x728000ad // movk w13, #5
	WORD $0x4e041da6 // mov v6.s[0], w13
	MOVD R1, R12
	MOVD R3, R7
	CMPW $3, R4
	BNE  qpel_0y_neon_base_ready
	ADD  R3, R12, R12
qpel_0y_neon_base_ready:
	SUB R3, R1, R1
	SUB R3, R1, R1
	CMPW $8, R6
	BEQ  qpel_0y_neon_width8

	BL  ·h264QpelV8NEONInternal(SB)
	SUB R3<<2, R1, R1
	BL  ·h264QpelV8NEONInternal(SB)

	SUB R2<<4, R0, R0
	ADD $8, R0, R0
	SUB R3<<4, R1, R1
	SUB R3<<2, R1, R1
	ADD $8, R1, R1
	SUB R3<<4, R12, R12
	ADD $8, R12, R12

	BL  ·h264QpelV8NEONInternal(SB)
	SUB R3<<2, R1, R1
	BL  ·h264QpelV8NEONInternal(SB)
	RET R14

qpel_0y_neon_width8:
	BL  ·h264QpelV8NEONInternal(SB)
	RET R14

// R0 is a packed temporary destination, R1 is source-minus-two-columns,
// R2/R3 are source and temporary strides, and R12 is the row count.
TEXT ·h264QpelH8NEONInternal(SB), NOSPLIT|NOFRAME, $0-0
	WORD $0x0cc2a03c
	WORD $0x0cc2a030
	WORD $0xf100098c
	WORD $0x2e1d1382
	WORD $0x2e1d1b83
	WORD $0x2e230042
	WORD $0x2e1d0b84
	WORD $0x2e1d2385
	WORD $0x2e250084
	WORD $0x2e1d2b81
	WORD $0x2e21039c
	WORD $0x2e111200
	WORD $0x6f56005c
	WORD $0x2e111a01
	WORD $0x2e210000
	WORD $0x2e110a01
	WORD $0x6f46409c
	WORD $0x2e112203
	WORD $0x2e230021
	WORD $0x2e112a02
	WORD $0x2e220210
	WORD $0x6f560010
	WORD $0x6f464030
	WORD $0x2f0b8f9c
	WORD $0x2f0b8e10
	WORD $0x0c83701c
	WORD $0x0c837010
	WORD $0x54fffca1
	WORD $0xd65f03c0

// h264QpelHVXYNEONInternal combines horizontal and vertical half-pel filters
// exactly as FFmpeg's mc11/mc31/mc13/mc33 leaves. Inputs are in registers:
// R0-R3 dst/src/strides, R4 mx, R5 avg, R6 size, and R7 my.
TEXT ·h264QpelHVXYNEONInternal(SB), NOSPLIT|NOFRAME, $0-0
	MOVD R30, R14
	MOVD R0, R8
	MOVD R1, R9
	MOVD R2, R15
	WORD $0x52a0028d // mov w13, #0x140000
	WORD $0x728000ad // movk w13, #5
	WORD $0x4e041da6 // mov v6.s[0], w13
	MOVD R3, R13

	MOVD R9, R10
	CMPW $3, R7
	BNE  qpel_hvxy_hsrc_ready
	ADD  R13, R10, R10
qpel_hvxy_hsrc_ready:
	MOVD RSP, R11
	SUB  $256, RSP, RSP
	MOVD RSP, R0
	MOVD R10, R1
	SUB  $2, R1, R1
	MOVD R13, R2
	MOVD R6, R3
	MOVD R6, R12
	BL   ·h264QpelH8NEONInternal(SB)
	CMPW $8, R6
	BEQ  qpel_hvxy_horizontal_done
	ADD  $8, RSP, R0
	MOVD R10, R1
	ADD  $6, R1, R1
	MOVD R6, R12
	BL   ·h264QpelH8NEONInternal(SB)
qpel_hvxy_horizontal_done:

	MOVD R8, R0
	MOVD R9, R1
	CMPW $3, R4
	BNE  qpel_hvxy_vsrc_ready
	ADD  $1, R1, R1
qpel_hvxy_vsrc_ready:
	SUB  R13, R1, R1
	SUB  R13, R1, R1
	MOVD R15, R2
	MOVD R13, R3
	MOVD RSP, R12
	MOVD R6, R7
	MOVW $1, R4
	CMPW $8, R6
	BEQ  qpel_hvxy_width8

	BL  ·h264QpelV8NEONInternal(SB)
	SUB R3<<2, R1, R1
	BL  ·h264QpelV8NEONInternal(SB)

	SUB R2<<4, R0, R0
	ADD $8, R0, R0
	SUB R3<<4, R1, R1
	SUB R3<<2, R1, R1
	ADD $8, R1, R1
	SUB R7<<4, R12, R12
	ADD $8, R12, R12

	BL  ·h264QpelV8NEONInternal(SB)
	SUB R3<<2, R1, R1
	BL  ·h264QpelV8NEONInternal(SB)
	MOVD R11, RSP
	RET R14

qpel_hvxy_width8:
	BL   ·h264QpelV8NEONInternal(SB)
	MOVD R11, RSP
	RET  R14

// R0/R1 are destination and source-minus-two in both axes, R2/R3 are
// independent strides, R4 selects an additional prediction in R12/R7, and R5
// selects averaging with the existing destination.
TEXT ·h264QpelHV8NEONInternal(SB), NOSPLIT|NOFRAME, $0-0
	MOVD R30, R17
	MOVD R12, R10
	BL   ·h264QpelHV8TopNEONInternal(SB)
	MOVD R10, R12
	CBZW R4, qpel_hv8_base_done
	WORD $0x0cc77198
	WORD $0x0cc77199
	WORD $0x0cc7719a
	WORD $0x0cc7719b
	WORD $0x0cc7719c
	WORD $0x2e301710
	WORD $0x2e311731
	WORD $0x0cc7719d
	WORD $0x2e321752
	WORD $0x2e331773
	WORD $0x0cc7719e
	WORD $0x2e341794
	WORD $0x2e3517b5
	WORD $0x0cc7719f
	WORD $0x2e3617d6
	WORD $0x2e3717f7
qpel_hv8_base_done:
	CBZW R5, qpel_hv8_avg_done
	WORD $0x0cc27018
	WORD $0x0cc27019
	WORD $0x0cc2701a
	WORD $0x2e381610
	WORD $0x0cc2701b
	WORD $0x2e391631
	WORD $0x0cc2701c
	WORD $0x2e3a1652
	WORD $0x0cc2701d
	WORD $0x2e3b1673
	WORD $0x0cc2701e
	WORD $0x2e3c1694
	WORD $0x0cc2701f
	WORD $0x2e3d16b5
	WORD $0x2e3e16d6
	WORD $0x2e3f16f7
	WORD $0xcb020c00
qpel_hv8_avg_done:
	WORD $0x0c827010
	WORD $0x0c827011
	WORD $0x0c827012
	WORD $0x0c827013
	WORD $0x0c827014
	WORD $0x0c827015
	WORD $0x0c827016
	WORD $0x0c827017
	RET  R17

// Inputs are R0-R3 dst/src/strides, R5 avg, and R6 size.
TEXT ·h264Qpel22NEONInternal(SB), NOSPLIT|NOFRAME, $0-0
	MOVD R30, R14
	MOVW $0, R4
	SUB  R3, R1, R1
	SUB  R3, R1, R1
	SUB  $2, R1, R1
	CMPW $8, R6
	BEQ  qpel_22_width8

	BL  ·h264QpelHV8NEONInternal(SB)
	SUB R3<<2, R1, R1
	BL  ·h264QpelHV8NEONInternal(SB)

	SUB R2<<4, R0, R0
	ADD $8, R0, R0
	SUB R3<<4, R1, R1
	SUB R3<<2, R1, R1
	ADD $8, R1, R1

	BL  ·h264QpelHV8NEONInternal(SB)
	SUB R3<<2, R1, R1
	BL  ·h264QpelHV8NEONInternal(SB)
	RET R14

qpel_22_width8:
	BL  ·h264QpelHV8NEONInternal(SB)
	RET R14

// Inputs are R0-R3 dst/src/strides, R4 mx, R5 avg, R6 size, and R7 my.
// The temporary first holds the horizontal or vertical half-pel prediction;
// h264QpelHV8NEONInternal then rounds it with the two-axis half-pel result.
TEXT ·h264QpelHVBlendNEONInternal(SB), NOSPLIT|NOFRAME, $0-0
	MOVD R30, R14
	MOVD R0, R8
	MOVD R1, R9
	MOVD R2, R15
	MOVD R5, R10
	WORD $0x52a0028d // mov w13, #0x140000
	WORD $0x728000ad // movk w13, #5
	WORD $0x4e041da6 // mov v6.s[0], w13
	MOVD R3, R13
	MOVD RSP, R11
	SUB  $256, RSP, RSP

	CMPW $2, R4
	BNE  qpel_hvblend_vertical_base
	MOVD RSP, R0
	MOVD R9, R1
	CMPW $3, R7
	BNE  qpel_hvblend_hsrc_ready
	ADD  R13, R1, R1
qpel_hvblend_hsrc_ready:
	SUB  $2, R1, R1
	MOVD R13, R2
	MOVD R6, R3
	MOVD R6, R12
	BL   ·h264QpelH8NEONInternal(SB)
	CMPW $8, R6
	BEQ  qpel_hvblend_base_done
	ADD  $8, RSP, R0
	MOVD R9, R1
	CMPW $3, R7
	BNE  qpel_hvblend_hsrc_right_ready
	ADD  R13, R1, R1
qpel_hvblend_hsrc_right_ready:
	ADD  $6, R1, R1
	MOVD R6, R12
	BL   ·h264QpelH8NEONInternal(SB)
	B    qpel_hvblend_base_done

qpel_hvblend_vertical_base:
	MOVD RSP, R0
	MOVD R9, R1
	CMPW $3, R4
	BNE  qpel_hvblend_vsrc_ready
	ADD  $1, R1, R1
qpel_hvblend_vsrc_ready:
	SUB  R13, R1, R1
	SUB  R13, R1, R1
	MOVD R6, R2
	MOVD R13, R3
	MOVW $2, R4
	MOVW $0, R5
	MOVD R13, R7
	CMPW $8, R6
	BEQ  qpel_hvblend_vbase_width8
	BL   ·h264QpelV8NEONInternal(SB)
	SUB  R3<<2, R1, R1
	BL   ·h264QpelV8NEONInternal(SB)
	SUB  R2<<4, R0, R0
	ADD  $8, R0, R0
	SUB  R3<<4, R1, R1
	SUB  R3<<2, R1, R1
	ADD  $8, R1, R1
	BL   ·h264QpelV8NEONInternal(SB)
	SUB  R3<<2, R1, R1
	BL   ·h264QpelV8NEONInternal(SB)
	B    qpel_hvblend_base_done
qpel_hvblend_vbase_width8:
	BL   ·h264QpelV8NEONInternal(SB)

qpel_hvblend_base_done:
	MOVD R8, R0
	MOVD R9, R1
	SUB  R13, R1, R1
	SUB  R13, R1, R1
	SUB  $2, R1, R1
	MOVD R15, R2
	MOVD R13, R3
	MOVW $1, R4
	MOVD R10, R5
	MOVD R6, R7
	MOVD RSP, R12
	CMPW $8, R6
	BEQ  qpel_hvblend_width8

	BL  ·h264QpelHV8NEONInternal(SB)
	SUB R3<<2, R1, R1
	BL  ·h264QpelHV8NEONInternal(SB)

	SUB R2<<4, R0, R0
	ADD $8, R0, R0
	SUB R3<<4, R1, R1
	SUB R3<<2, R1, R1
	ADD $8, R1, R1
	SUB R7<<4, R12, R12
	ADD $8, R12, R12

	BL  ·h264QpelHV8NEONInternal(SB)
	SUB R3<<2, R1, R1
	BL  ·h264QpelHV8NEONInternal(SB)
	MOVD R11, RSP
	RET R14

qpel_hvblend_width8:
	BL   ·h264QpelHV8NEONInternal(SB)
	MOVD R11, RSP
	RET  R14
