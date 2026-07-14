// SPDX-License-Identifier: LGPL-2.1-or-later
//go:build !purego && arm64

#include "textflag.h"

// h264CABACMVDCommonASM decodes the context-coded MVD prefix and, for values
// below the escape threshold, its bypass-coded sign. CABAC low/range and the
// byte cursor stay in registers across the entire common path.
// func h264CABACMVDCommonASM(c *cabacContext, states *[1024]uint8, ctxBase int, ctx int) (value int32, mvd int)
TEXT ·h264CABACMVDCommonASM(SB), NOSPLIT|NOFRAME, $0-48
	MOVD c+0(FP), R0
	MOVD states+8(FP), R1
	MOVD ctxBase+16(FP), R2
	MOVD ctx+24(FP), R3
	MOVWU 0(R0), R4
	MOVWU 4(R0), R5
	MOVD 16(R0), R6
	MOVD 24(R0), R7
	MOVD 32(R0), R8
	MOVD $·h264CABACTables(SB), R9
	MOVD ZR, R10 // absolute MVD prefix

cabac_mvd_loop:
	ADD R3, R1, R11
	MOVBU (R11), R12
	AND $192, R5, R13
	ADD R13<<1, R12, R13
	ADD $512, R13, R13
	MOVBU (R9)(R13), R13
	SUB R13, R5, R14
	LSLW $17, R14, R15
	SUBW R4, R15, R15
	ASRW $31, R15, R15
	ANDW R14<<17, R15, R16
	SUBW R16, R4, R4
	ADDW R13, R13, R16
	SUBW R5, R16, R16
	ANDW R15, R16, R16
	ADDW R14, R16, R5
	EORW R12, R15, R12
	ADDW $1152, R12, R13
	MOVBU (R9)(R13), R13
	MOVB R13, (R11)
	CLZW R5, R13
	SUBW $23, R13, R13
	LSLW R13, R5, R5
	LSLW R13, R4, R4
	TSTW $65535, R4
	BNE cabac_mvd_bin_ready

	RBITW R4, R13
	CLZW R13, R13
	SUBW $16, R13, R13
	MOVW $-65535, R14
	CMP R7, R6
	BGE cabac_mvd_refill_shift
	MOVBU (R8)(R6), R16
	ADDW R16<<9, R14, R14
	ADD $1, R6, R16
	CMP R7, R16
	BGE cabac_mvd_refill_advance
	MOVBU (R8)(R16), R16
	ADDW R16<<1, R14, R14
cabac_mvd_refill_advance:
	ADD $2, R6, R6
cabac_mvd_refill_shift:
	LSLW R13, R14, R14
	ADDW R14, R4, R4

cabac_mvd_bin_ready:
	TBZ $0, R12, cabac_mvd_prefix_done
	CBNZ R10, cabac_mvd_after_first
	MOVD $1, R10
	ADD $3, R2, R3
	B cabac_mvd_loop

cabac_mvd_after_first:
	CMP $8, R10
	BGE cabac_mvd_escape
	CMP $4, R10
	BGE cabac_mvd_increment
	ADD $1, R3, R3
cabac_mvd_increment:
	ADD $1, R10, R10
	B cabac_mvd_loop

cabac_mvd_prefix_done:
	CBZ R10, cabac_mvd_zero

	// Decode the common bypass-coded sign without returning to Go.
	LSLW $1, R4, R4
	TSTW $65535, R4
	BNE cabac_mvd_sign_ready
	MOVW ZR, R13
	CMP R7, R6
	BGE cabac_mvd_sign_refill_subtract
	MOVBU (R8)(R6), R14
	ADDW R14<<9, R13, R13
	ADD $1, R6, R14
	CMP R7, R14
	BGE cabac_mvd_sign_refill_advance
	MOVBU (R8)(R14), R14
	ADDW R14<<1, R13, R13
cabac_mvd_sign_refill_advance:
	ADD $2, R6, R6
cabac_mvd_sign_refill_subtract:
	SUBW $65535, R13, R13
	ADDW R13, R4, R4

cabac_mvd_sign_ready:
	LSLW $17, R5, R13
	SUBW R13, R4, R14
	ASRW $31, R14, R14
	BIC R14, R13, R13
	SUBW R13, R4, R4
	NEGW R10, R13
	EORW R14, R13, R13
	SUBW R14, R13, R13
	MOVW R4, 0(R0)
	MOVW R5, 4(R0)
	MOVD R6, 16(R0)
	MOVW R13, value+32(FP)
	MOVD R10, mvd+40(FP)
	RET

cabac_mvd_escape:
	MOVD $9, R10
	MOVW R4, 0(R0)
	MOVW R5, 4(R0)
	MOVD R6, 16(R0)
	MOVW ZR, value+32(FP)
	MOVD R10, mvd+40(FP)
	RET

cabac_mvd_zero:
	MOVW R4, 0(R0)
	MOVW R5, 4(R0)
	MOVD R6, 16(R0)
	MOVW ZR, value+32(FP)
	MOVD ZR, mvd+40(FP)
	RET
