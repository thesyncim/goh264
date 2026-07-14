// SPDX-License-Identifier: LGPL-2.1-or-later
//go:build !purego && arm64

#include "textflag.h"

// h264CABACIntra4x4PredModeASM decodes the previous-mode flag and, when
// needed, all three rem-mode bins while CABAC low/range and the byte cursor
// remain in registers.
// func h264CABACIntra4x4PredModeASM(c *cabacContext, states *[1024]uint8, predMode int) (mode int)
TEXT ·h264CABACIntra4x4PredModeASM(SB), NOSPLIT|NOFRAME, $0-32
	MOVD c+0(FP), R0
	MOVD states+8(FP), R1
	MOVD predMode+16(FP), R2
	MOVD $68, R3
	MOVWU 0(R0), R4
	MOVWU 4(R0), R5
	MOVD 16(R0), R6
	MOVD 24(R0), R7
	MOVD 32(R0), R8
	MOVD $·h264CABACTables(SB), R9
	MOVD ZR, R10
	MOVD ZR, R11

cabac_intra4x4_mode_loop:
	ADD R3, R1, R17
	MOVBU (R17), R12
	AND $192, R5, R13
	ADD R13<<1, R12, R13
	ADD $512, R13, R13
	MOVBU (R9)(R13), R13
	SUB R13, R5, R14
	LSL $17, R14, R15
	SUB R4, R15, R15
	MOVW R15, R15
	LSRW $31, R15, R15
	NEG R15, R15
	AND R14<<17, R15, R16
	SUBW R16, R4, R4
	ADD R13, R13, R16
	SUB R5, R16, R16
	AND R15, R16, R16
	ADD R14, R16, R5
	EOR R12, R15, R12
	ADD $1152, R12, R13
	MOVBU (R9)(R13), R13
	MOVB R13, (R17)
	MOVBU (R9)(R5), R13
	AND $31, R13, R13
	LSLW R13, R5, R5
	LSLW R13, R4, R4
	TSTW $65535, R4
	BNE cabac_intra4x4_mode_bin_ready

	SUBW $1, R4, R13
	EORW R4, R13, R13
	LSRW $15, R13, R13
	MOVBU (R9)(R13), R13
	MOVW $7, R14
	SUBW R13, R14, R13
	MOVW $-65535, R14
	CMP R7, R6
	BGE cabac_intra4x4_mode_refill_shift
	MOVBU (R8)(R6), R16
	ADDW R16<<9, R14, R14
	ADD $1, R6, R16
	CMP R7, R16
	BGE cabac_intra4x4_mode_refill_advance
	MOVBU (R8)(R16), R16
	ADDW R16<<1, R14, R14
cabac_intra4x4_mode_refill_advance:
	ADD $2, R6, R6
cabac_intra4x4_mode_refill_shift:
	LSLW R13, R14, R14
	ADDW R14, R4, R4

cabac_intra4x4_mode_bin_ready:
	CBNZ R10, cabac_intra4x4_mode_explicit
	TBZ $0, R12, cabac_intra4x4_mode_first_zero
	MOVD R2, R11
	B cabac_intra4x4_mode_store

cabac_intra4x4_mode_first_zero:
	MOVD $69, R3
	MOVD $1, R10
	B cabac_intra4x4_mode_loop

cabac_intra4x4_mode_explicit:
	AND $1, R12, R13
	SUB $1, R10, R14
	LSL R14, R13, R13
	ADD R13, R11, R11
	CMP $3, R10
	BEQ cabac_intra4x4_mode_remap
	ADD $1, R10, R10
	B cabac_intra4x4_mode_loop

cabac_intra4x4_mode_remap:
	CMP R2, R11
	BLT cabac_intra4x4_mode_store
	ADD $1, R11, R11

cabac_intra4x4_mode_store:
	MOVW R4, 0(R0)
	MOVW R5, 4(R0)
	MOVD R6, 16(R0)
	MOVD R11, mode+24(FP)
	RET
