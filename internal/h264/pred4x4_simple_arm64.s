// SPDX-License-Identifier: LGPL-2.1-or-later
//go:build !purego && arm64

#include "textflag.h"

// func h264Pred4x4SimpleASM(dst *uint8, stride int, mode int32)
TEXT ·h264Pred4x4SimpleASM(SB), NOSPLIT|NOFRAME, $0-24
	MOVD dst+0(FP), R0
	MOVD stride+8(FP), R1
	MOVW mode+16(FP), R2
	CBZW R2, pred4x4_vertical
	CMPW $1, R2
	BEQ pred4x4_horizontal
	CMPW $2, R2
	BEQ pred4x4_dc
	CMPW $9, R2
	BEQ pred4x4_left_dc
	CMPW $10, R2
	BEQ pred4x4_top_dc
	B pred4x4_dc128

pred4x4_vertical:
	SUB R1, R0, R3
	MOVWU (R3), R4
	MOVW R4, (R0)
	ADD R1, R0, R3
	MOVW R4, (R3)
	ADD R1, R3, R3
	MOVW R4, (R3)
	ADD R1, R3, R3
	MOVW R4, (R3)
	RET

pred4x4_horizontal:
	MOVD R0, R3
	MOVD $0x01010101, R5
	MOVBU -1(R3), R4
	MULW R5, R4, R4
	MOVW R4, (R3)
	ADD R1, R3, R3
	MOVBU -1(R3), R4
	MULW R5, R4, R4
	MOVW R4, (R3)
	ADD R1, R3, R3
	MOVBU -1(R3), R4
	MULW R5, R4, R4
	MOVW R4, (R3)
	ADD R1, R3, R3
	MOVBU -1(R3), R4
	MULW R5, R4, R4
	MOVW R4, (R3)
	RET

pred4x4_dc:
	SUB R1, R0, R3
	MOVBU (R3), R4
	MOVBU 1(R3), R5
	ADD R5, R4, R4
	MOVBU 2(R3), R5
	ADD R5, R4, R4
	MOVBU 3(R3), R5
	ADD R5, R4, R4
	MOVD R0, R3
	MOVBU -1(R3), R5
	ADD R5, R4, R4
	ADD R1, R3, R3
	MOVBU -1(R3), R5
	ADD R5, R4, R4
	ADD R1, R3, R3
	MOVBU -1(R3), R5
	ADD R5, R4, R4
	ADD R1, R3, R3
	MOVBU -1(R3), R5
	ADD R5, R4, R4
	ADD $4, R4, R4
	LSRW $3, R4, R4
	B pred4x4_fill

pred4x4_left_dc:
	MOVD R0, R3
	MOVBU -1(R3), R4
	ADD R1, R3, R3
	MOVBU -1(R3), R5
	ADD R5, R4, R4
	ADD R1, R3, R3
	MOVBU -1(R3), R5
	ADD R5, R4, R4
	ADD R1, R3, R3
	MOVBU -1(R3), R5
	ADD R5, R4, R4
	ADD $2, R4, R4
	LSRW $2, R4, R4
	B pred4x4_fill

pred4x4_top_dc:
	SUB R1, R0, R3
	MOVBU (R3), R4
	MOVBU 1(R3), R5
	ADD R5, R4, R4
	MOVBU 2(R3), R5
	ADD R5, R4, R4
	MOVBU 3(R3), R5
	ADD R5, R4, R4
	ADD $2, R4, R4
	LSRW $2, R4, R4
	B pred4x4_fill

pred4x4_dc128:
	MOVW $0x80808080, R4
	B pred4x4_store_rows

pred4x4_fill:
	MOVD $0x01010101, R5
	MULW R5, R4, R4
pred4x4_store_rows:
	MOVD R0, R3
	MOVW R4, (R3)
	ADD R1, R3, R3
	MOVW R4, (R3)
	ADD R1, R3, R3
	MOVW R4, (R3)
	ADD R1, R3, R3
	MOVW R4, (R3)
	RET
