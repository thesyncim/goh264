// SPDX-License-Identifier: LGPL-2.1-or-later
//go:build !purego && arm64

#include "textflag.h"

// h264CABACResidualSignificanceFixedASM decodes a complete fixed-size
// significance/last scan with range, low, and the byte cursor held in
// registers. The scalar implementation remains the differential oracle.
// func h264CABACResidualSignificanceFixedASM(c *cabacContext, states *[1024]uint8, index *[64]uint8, sigCtxBase int, lastCtxBase int, maxCoeff int) (coeffCount int, last int)
TEXT ·h264CABACResidualSignificanceFixedASM(SB), NOSPLIT|NOFRAME, $0-64
	MOVD c+0(FP), R0
	MOVD states+8(FP), R1
	MOVD index+16(FP), R2
	MOVD sigCtxBase+24(FP), R3
	MOVD lastCtxBase+32(FP), R4
	MOVD maxCoeff+40(FP), R21
	SUB $1, R21, R22
	MOVWU 0(R0), R6
	MOVWU 4(R0), R7
	MOVD 16(R0), R8
	MOVD 24(R0), R9
	MOVD 32(R0), R10
	MOVD $·h264CABACTables(SB), R11
	MOVD ZR, R12 // last
	MOVD ZR, R13 // coefficient count

cabac_sig_loop:
	CMP R22, R12
	BGE cabac_sig_done
	ADD R12, R3, R14
	ADD R14, R1, R14
	MOVBU (R14), R15
	AND $192, R7, R16
	ADD R16<<1, R15, R16
	ADD $512, R16, R16
	MOVBU (R11)(R16), R16
	SUB R16, R7, R17
	LSLW $17, R17, R19
	SUBW R6, R19, R19
	ASRW $31, R19, R19
	ANDW R17<<17, R19, R20
	SUBW R20, R6, R6
	ADDW R16, R16, R20
	SUBW R7, R20, R20
	ANDW R19, R20, R20
	ADDW R17, R20, R7
	EORW R15, R19, R15
	ADDW $1152, R15, R16
	MOVBU (R11)(R16), R16
	MOVB R16, (R14)
	CLZW R7, R16
	SUBW $23, R16, R16
	LSLW R16, R7, R7
	LSLW R16, R6, R6
	TSTW $65535, R6
	BNE cabac_sig_bin_ready

	RBITW R6, R16
	CLZW R16, R16
	SUBW $16, R16, R16
	MOVW $-65535, R17
	CMP R9, R8
	BGE cabac_sig_refill_shift
	MOVBU (R10)(R8), R20
	ADDW R20<<9, R17, R17
	ADD $1, R8, R20
	CMP R9, R20
	BGE cabac_sig_refill_advance
	MOVBU (R10)(R20), R20
	ADDW R20<<1, R17, R17
cabac_sig_refill_advance:
	ADD $2, R8, R8
cabac_sig_refill_shift:
	LSLW R16, R17, R17
	ADDW R17, R6, R6

cabac_sig_bin_ready:
	TBZ $0, R15, cabac_sig_next
	MOVB R12, (R2)(R13)
	ADD $1, R13, R13

	ADD R12, R4, R14
	ADD R14, R1, R14
	MOVBU (R14), R15
	AND $192, R7, R16
	ADD R16<<1, R15, R16
	ADD $512, R16, R16
	MOVBU (R11)(R16), R16
	SUB R16, R7, R17
	LSLW $17, R17, R19
	SUBW R6, R19, R19
	ASRW $31, R19, R19
	ANDW R17<<17, R19, R20
	SUBW R20, R6, R6
	ADDW R16, R16, R20
	SUBW R7, R20, R20
	ANDW R19, R20, R20
	ADDW R17, R20, R7
	EORW R15, R19, R15
	ADDW $1152, R15, R16
	MOVBU (R11)(R16), R16
	MOVB R16, (R14)
	CLZW R7, R16
	SUBW $23, R16, R16
	LSLW R16, R7, R7
	LSLW R16, R6, R6
	TSTW $65535, R6
	BNE cabac_last_bin_ready

	RBITW R6, R16
	CLZW R16, R16
	SUBW $16, R16, R16
	MOVW $-65535, R17
	CMP R9, R8
	BGE cabac_last_refill_shift
	MOVBU (R10)(R8), R20
	ADDW R20<<9, R17, R17
	ADD $1, R8, R20
	CMP R9, R20
	BGE cabac_last_refill_advance
	MOVBU (R10)(R20), R20
	ADDW R20<<1, R17, R17
cabac_last_refill_advance:
	ADD $2, R8, R8
cabac_last_refill_shift:
	LSLW R16, R17, R17
	ADDW R17, R6, R6

cabac_last_bin_ready:
	TBZ $0, R15, cabac_sig_next
	MOVD R21, R12
	B cabac_sig_done

cabac_sig_next:
	ADD $1, R12, R12
	B cabac_sig_loop

cabac_sig_done:
	MOVW R6, 0(R0)
	MOVW R7, 4(R0)
	MOVD R8, 16(R0)
	MOVD R13, coeffCount+48(FP)
	MOVD R12, last+56(FP)
	RET
