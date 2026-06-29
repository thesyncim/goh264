// SPDX-License-Identifier: LGPL-2.1-or-later
//go:build !purego && amd64

#include "textflag.h"


// func h264QpelMCPutHVBlendASM(dst *uint8, src *uint8, dstStride int, srcStride int, size int32, mx int32, my int32)
TEXT ·h264QpelMCPutHVBlendASM(SB), NOSPLIT, $32-48
	MOVQ dst+0(FP), DI
	MOVQ src+8(FP), SI
	MOVQ dstStride+16(FP), DX
	MOVQ srcStride+24(FP), CX
	MOVL size+32(FP), R8
	MOVL $0, 28(SP)
qpel_puthvblend_row:
	MOVQ DI, R11
	MOVQ SI, R12
	MOVL size+32(FP), R9
qpel_puthvblend_col:
	MOVL mx+36(FP), R15
	CMPL R15, $2
	JNE  qpel_puthvblend_vbase
	MOVQ R12, R13
	MOVL my+40(FP), R15
	CMPL R15, $3
	JNE  qpel_puthvblend_hbase_ptr_ready
	ADDQ CX, R13
qpel_puthvblend_hbase_ptr_ready:
	XORL AX, AX
	MOVB (R13), AX
	XORL BX, BX
	MOVB 1(R13), BX
	ADDL BX, AX
	IMULL $20, AX
	MOVL AX, R10
	XORL AX, AX
	MOVB -1(R13), AX
	XORL BX, BX
	MOVB 2(R13), BX
	ADDL BX, AX
	LEAL (AX)(AX*4), AX
	SUBL AX, R10
	XORL AX, AX
	MOVB -2(R13), AX
	XORL BX, BX
	MOVB 3(R13), BX
	ADDL BX, AX
	ADDL AX, R10
	ADDL $16, R10
	SARL $5, R10
	CMPL R10, $0
	JGE  qpel_puthvblend_hbase_nonnegative
	XORL R10, R10
	JMP  qpel_puthvblend_base_done
qpel_puthvblend_hbase_nonnegative:
	CMPL R10, $255
	JLE  qpel_puthvblend_base_done
	MOVL $255, R10
	JMP  qpel_puthvblend_base_done
qpel_puthvblend_vbase:
	MOVQ R12, R13
	MOVL mx+36(FP), R15
	CMPL R15, $3
	JNE  qpel_puthvblend_vbase_ptr_ready
	INCQ R13
qpel_puthvblend_vbase_ptr_ready:
	XORL AX, AX
	MOVB (R13), AX
	MOVQ R13, R15
	ADDQ CX, R15
	XORL BX, BX
	MOVB (R15), BX
	ADDL BX, AX
	IMULL $20, AX
	MOVL AX, R10
	MOVQ R13, R15
	SUBQ CX, R15
	XORL AX, AX
	MOVB (R15), AX
	MOVQ R13, R15
	ADDQ CX, R15
	ADDQ CX, R15
	XORL BX, BX
	MOVB (R15), BX
	ADDL BX, AX
	LEAL (AX)(AX*4), AX
	SUBL AX, R10
	MOVQ R13, R15
	SUBQ CX, R15
	SUBQ CX, R15
	XORL AX, AX
	MOVB (R15), AX
	MOVQ R13, R15
	ADDQ CX, R15
	ADDQ CX, R15
	ADDQ CX, R15
	XORL BX, BX
	MOVB (R15), BX
	ADDL BX, AX
	ADDL AX, R10
	ADDL $16, R10
	SARL $5, R10
	CMPL R10, $0
	JGE  qpel_puthvblend_vbase_nonnegative
	XORL R10, R10
	JMP  qpel_puthvblend_base_done
qpel_puthvblend_vbase_nonnegative:
	CMPL R10, $255
	JLE  qpel_puthvblend_base_done
	MOVL $255, R10
qpel_puthvblend_base_done:
	MOVL R10, 24(SP)
	MOVQ R12, R13
	SUBQ CX, R13
	SUBQ CX, R13
	XORL AX, AX
	MOVB (R13), AX
	XORL BX, BX
	MOVB 1(R13), BX
	ADDL BX, AX
	IMULL $20, AX
	MOVL AX, R10
	XORL AX, AX
	MOVB -1(R13), AX
	XORL BX, BX
	MOVB 2(R13), BX
	ADDL BX, AX
	LEAL (AX)(AX*4), AX
	SUBL AX, R10
	XORL AX, AX
	MOVB -2(R13), AX
	XORL BX, BX
	MOVB 3(R13), BX
	ADDL BX, AX
	ADDL AX, R10
	MOVL R10, 0(SP)
	MOVQ R12, R13
	SUBQ CX, R13
	XORL AX, AX
	MOVB (R13), AX
	XORL BX, BX
	MOVB 1(R13), BX
	ADDL BX, AX
	IMULL $20, AX
	MOVL AX, R10
	XORL AX, AX
	MOVB -1(R13), AX
	XORL BX, BX
	MOVB 2(R13), BX
	ADDL BX, AX
	LEAL (AX)(AX*4), AX
	SUBL AX, R10
	XORL AX, AX
	MOVB -2(R13), AX
	XORL BX, BX
	MOVB 3(R13), BX
	ADDL BX, AX
	ADDL AX, R10
	MOVL R10, 4(SP)
	MOVQ R12, R13
	XORL AX, AX
	MOVB (R13), AX
	XORL BX, BX
	MOVB 1(R13), BX
	ADDL BX, AX
	IMULL $20, AX
	MOVL AX, R10
	XORL AX, AX
	MOVB -1(R13), AX
	XORL BX, BX
	MOVB 2(R13), BX
	ADDL BX, AX
	LEAL (AX)(AX*4), AX
	SUBL AX, R10
	XORL AX, AX
	MOVB -2(R13), AX
	XORL BX, BX
	MOVB 3(R13), BX
	ADDL BX, AX
	ADDL AX, R10
	MOVL R10, 8(SP)
	MOVQ R12, R13
	ADDQ CX, R13
	XORL AX, AX
	MOVB (R13), AX
	XORL BX, BX
	MOVB 1(R13), BX
	ADDL BX, AX
	IMULL $20, AX
	MOVL AX, R10
	XORL AX, AX
	MOVB -1(R13), AX
	XORL BX, BX
	MOVB 2(R13), BX
	ADDL BX, AX
	LEAL (AX)(AX*4), AX
	SUBL AX, R10
	XORL AX, AX
	MOVB -2(R13), AX
	XORL BX, BX
	MOVB 3(R13), BX
	ADDL BX, AX
	ADDL AX, R10
	MOVL R10, 12(SP)
	MOVQ R12, R13
	ADDQ CX, R13
	ADDQ CX, R13
	XORL AX, AX
	MOVB (R13), AX
	XORL BX, BX
	MOVB 1(R13), BX
	ADDL BX, AX
	IMULL $20, AX
	MOVL AX, R10
	XORL AX, AX
	MOVB -1(R13), AX
	XORL BX, BX
	MOVB 2(R13), BX
	ADDL BX, AX
	LEAL (AX)(AX*4), AX
	SUBL AX, R10
	XORL AX, AX
	MOVB -2(R13), AX
	XORL BX, BX
	MOVB 3(R13), BX
	ADDL BX, AX
	ADDL AX, R10
	MOVL R10, 16(SP)
	MOVQ R12, R13
	ADDQ CX, R13
	ADDQ CX, R13
	ADDQ CX, R13
	XORL AX, AX
	MOVB (R13), AX
	XORL BX, BX
	MOVB 1(R13), BX
	ADDL BX, AX
	IMULL $20, AX
	MOVL AX, R10
	XORL AX, AX
	MOVB -1(R13), AX
	XORL BX, BX
	MOVB 2(R13), BX
	ADDL BX, AX
	LEAL (AX)(AX*4), AX
	SUBL AX, R10
	XORL AX, AX
	MOVB -2(R13), AX
	XORL BX, BX
	MOVB 3(R13), BX
	ADDL BX, AX
	ADDL AX, R10
	MOVL R10, 20(SP)
	MOVL 8(SP), AX
	ADDL 12(SP), AX
	IMULL $20, AX
	MOVL AX, R10
	MOVL 4(SP), AX
	ADDL 16(SP), AX
	LEAL (AX)(AX*4), AX
	SUBL AX, R10
	MOVL 0(SP), AX
	ADDL 20(SP), AX
	ADDL AX, R10
	ADDL $512, R10
	SARL $10, R10
	CMPL R10, $0
	JGE  qpel_puthvblend_hv_nonnegative
	XORL R10, R10
	JMP  qpel_puthvblend_hv_done
qpel_puthvblend_hv_nonnegative:
	CMPL R10, $255
	JLE  qpel_puthvblend_hv_done
	MOVL $255, R10
qpel_puthvblend_hv_done:
	MOVL 24(SP), AX
	ADDL AX, R10
	ADDL $1, R10
	SHRL $1, R10
	CMPL 28(SP), $0
	JE   qpel_puthvblend_store
	XORL AX, AX
	MOVB (R11), AX
	ADDL AX, R10
	ADDL $1, R10
	SHRL $1, R10
qpel_puthvblend_store:
	MOVB R10, (R11)
	INCQ R11
	INCQ R12
	DECL R9
	JNZ  qpel_puthvblend_col
	ADDQ DX, DI
	ADDQ CX, SI
	DECL R8
	JNZ  qpel_puthvblend_row
	RET

// func h264QpelMCAvgHVBlendASM(dst *uint8, src *uint8, dstStride int, srcStride int, size int32, mx int32, my int32)
TEXT ·h264QpelMCAvgHVBlendASM(SB), NOSPLIT, $32-48
	MOVQ dst+0(FP), DI
	MOVQ src+8(FP), SI
	MOVQ dstStride+16(FP), DX
	MOVQ srcStride+24(FP), CX
	MOVL size+32(FP), R8
	MOVL $1, 28(SP)
qpel_avghvblend_row:
	MOVQ DI, R11
	MOVQ SI, R12
	MOVL size+32(FP), R9
qpel_avghvblend_col:
	MOVL mx+36(FP), R15
	CMPL R15, $2
	JNE  qpel_avghvblend_vbase
	MOVQ R12, R13
	MOVL my+40(FP), R15
	CMPL R15, $3
	JNE  qpel_avghvblend_hbase_ptr_ready
	ADDQ CX, R13
qpel_avghvblend_hbase_ptr_ready:
	XORL AX, AX
	MOVB (R13), AX
	XORL BX, BX
	MOVB 1(R13), BX
	ADDL BX, AX
	IMULL $20, AX
	MOVL AX, R10
	XORL AX, AX
	MOVB -1(R13), AX
	XORL BX, BX
	MOVB 2(R13), BX
	ADDL BX, AX
	LEAL (AX)(AX*4), AX
	SUBL AX, R10
	XORL AX, AX
	MOVB -2(R13), AX
	XORL BX, BX
	MOVB 3(R13), BX
	ADDL BX, AX
	ADDL AX, R10
	ADDL $16, R10
	SARL $5, R10
	CMPL R10, $0
	JGE  qpel_avghvblend_hbase_nonnegative
	XORL R10, R10
	JMP  qpel_avghvblend_base_done
qpel_avghvblend_hbase_nonnegative:
	CMPL R10, $255
	JLE  qpel_avghvblend_base_done
	MOVL $255, R10
	JMP  qpel_avghvblend_base_done
qpel_avghvblend_vbase:
	MOVQ R12, R13
	MOVL mx+36(FP), R15
	CMPL R15, $3
	JNE  qpel_avghvblend_vbase_ptr_ready
	INCQ R13
qpel_avghvblend_vbase_ptr_ready:
	XORL AX, AX
	MOVB (R13), AX
	MOVQ R13, R15
	ADDQ CX, R15
	XORL BX, BX
	MOVB (R15), BX
	ADDL BX, AX
	IMULL $20, AX
	MOVL AX, R10
	MOVQ R13, R15
	SUBQ CX, R15
	XORL AX, AX
	MOVB (R15), AX
	MOVQ R13, R15
	ADDQ CX, R15
	ADDQ CX, R15
	XORL BX, BX
	MOVB (R15), BX
	ADDL BX, AX
	LEAL (AX)(AX*4), AX
	SUBL AX, R10
	MOVQ R13, R15
	SUBQ CX, R15
	SUBQ CX, R15
	XORL AX, AX
	MOVB (R15), AX
	MOVQ R13, R15
	ADDQ CX, R15
	ADDQ CX, R15
	ADDQ CX, R15
	XORL BX, BX
	MOVB (R15), BX
	ADDL BX, AX
	ADDL AX, R10
	ADDL $16, R10
	SARL $5, R10
	CMPL R10, $0
	JGE  qpel_avghvblend_vbase_nonnegative
	XORL R10, R10
	JMP  qpel_avghvblend_base_done
qpel_avghvblend_vbase_nonnegative:
	CMPL R10, $255
	JLE  qpel_avghvblend_base_done
	MOVL $255, R10
qpel_avghvblend_base_done:
	MOVL R10, 24(SP)
	MOVQ R12, R13
	SUBQ CX, R13
	SUBQ CX, R13
	XORL AX, AX
	MOVB (R13), AX
	XORL BX, BX
	MOVB 1(R13), BX
	ADDL BX, AX
	IMULL $20, AX
	MOVL AX, R10
	XORL AX, AX
	MOVB -1(R13), AX
	XORL BX, BX
	MOVB 2(R13), BX
	ADDL BX, AX
	LEAL (AX)(AX*4), AX
	SUBL AX, R10
	XORL AX, AX
	MOVB -2(R13), AX
	XORL BX, BX
	MOVB 3(R13), BX
	ADDL BX, AX
	ADDL AX, R10
	MOVL R10, 0(SP)
	MOVQ R12, R13
	SUBQ CX, R13
	XORL AX, AX
	MOVB (R13), AX
	XORL BX, BX
	MOVB 1(R13), BX
	ADDL BX, AX
	IMULL $20, AX
	MOVL AX, R10
	XORL AX, AX
	MOVB -1(R13), AX
	XORL BX, BX
	MOVB 2(R13), BX
	ADDL BX, AX
	LEAL (AX)(AX*4), AX
	SUBL AX, R10
	XORL AX, AX
	MOVB -2(R13), AX
	XORL BX, BX
	MOVB 3(R13), BX
	ADDL BX, AX
	ADDL AX, R10
	MOVL R10, 4(SP)
	MOVQ R12, R13
	XORL AX, AX
	MOVB (R13), AX
	XORL BX, BX
	MOVB 1(R13), BX
	ADDL BX, AX
	IMULL $20, AX
	MOVL AX, R10
	XORL AX, AX
	MOVB -1(R13), AX
	XORL BX, BX
	MOVB 2(R13), BX
	ADDL BX, AX
	LEAL (AX)(AX*4), AX
	SUBL AX, R10
	XORL AX, AX
	MOVB -2(R13), AX
	XORL BX, BX
	MOVB 3(R13), BX
	ADDL BX, AX
	ADDL AX, R10
	MOVL R10, 8(SP)
	MOVQ R12, R13
	ADDQ CX, R13
	XORL AX, AX
	MOVB (R13), AX
	XORL BX, BX
	MOVB 1(R13), BX
	ADDL BX, AX
	IMULL $20, AX
	MOVL AX, R10
	XORL AX, AX
	MOVB -1(R13), AX
	XORL BX, BX
	MOVB 2(R13), BX
	ADDL BX, AX
	LEAL (AX)(AX*4), AX
	SUBL AX, R10
	XORL AX, AX
	MOVB -2(R13), AX
	XORL BX, BX
	MOVB 3(R13), BX
	ADDL BX, AX
	ADDL AX, R10
	MOVL R10, 12(SP)
	MOVQ R12, R13
	ADDQ CX, R13
	ADDQ CX, R13
	XORL AX, AX
	MOVB (R13), AX
	XORL BX, BX
	MOVB 1(R13), BX
	ADDL BX, AX
	IMULL $20, AX
	MOVL AX, R10
	XORL AX, AX
	MOVB -1(R13), AX
	XORL BX, BX
	MOVB 2(R13), BX
	ADDL BX, AX
	LEAL (AX)(AX*4), AX
	SUBL AX, R10
	XORL AX, AX
	MOVB -2(R13), AX
	XORL BX, BX
	MOVB 3(R13), BX
	ADDL BX, AX
	ADDL AX, R10
	MOVL R10, 16(SP)
	MOVQ R12, R13
	ADDQ CX, R13
	ADDQ CX, R13
	ADDQ CX, R13
	XORL AX, AX
	MOVB (R13), AX
	XORL BX, BX
	MOVB 1(R13), BX
	ADDL BX, AX
	IMULL $20, AX
	MOVL AX, R10
	XORL AX, AX
	MOVB -1(R13), AX
	XORL BX, BX
	MOVB 2(R13), BX
	ADDL BX, AX
	LEAL (AX)(AX*4), AX
	SUBL AX, R10
	XORL AX, AX
	MOVB -2(R13), AX
	XORL BX, BX
	MOVB 3(R13), BX
	ADDL BX, AX
	ADDL AX, R10
	MOVL R10, 20(SP)
	MOVL 8(SP), AX
	ADDL 12(SP), AX
	IMULL $20, AX
	MOVL AX, R10
	MOVL 4(SP), AX
	ADDL 16(SP), AX
	LEAL (AX)(AX*4), AX
	SUBL AX, R10
	MOVL 0(SP), AX
	ADDL 20(SP), AX
	ADDL AX, R10
	ADDL $512, R10
	SARL $10, R10
	CMPL R10, $0
	JGE  qpel_avghvblend_hv_nonnegative
	XORL R10, R10
	JMP  qpel_avghvblend_hv_done
qpel_avghvblend_hv_nonnegative:
	CMPL R10, $255
	JLE  qpel_avghvblend_hv_done
	MOVL $255, R10
qpel_avghvblend_hv_done:
	MOVL 24(SP), AX
	ADDL AX, R10
	ADDL $1, R10
	SHRL $1, R10
	CMPL 28(SP), $0
	JE   qpel_avghvblend_store
	XORL AX, AX
	MOVB (R11), AX
	ADDL AX, R10
	ADDL $1, R10
	SHRL $1, R10
qpel_avghvblend_store:
	MOVB R10, (R11)
	INCQ R11
	INCQ R12
	DECL R9
	JNZ  qpel_avghvblend_col
	ADDQ DX, DI
	ADDQ CX, SI
	DECL R8
	JNZ  qpel_avghvblend_row
	RET
