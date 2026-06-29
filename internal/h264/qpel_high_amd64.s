// SPDX-License-Identifier: LGPL-2.1-or-later
//go:build !purego && amd64

#include "textflag.h"

// func h264QpelMCHigh00ASM(dst *uint8, src *uint8, dstStride int, srcStride int, size int32, avg int32)
TEXT ·h264QpelMCHigh00ASM(SB), NOSPLIT, $0-40
	MOVQ dst+0(FP), DI
	MOVQ src+8(FP), SI
	MOVQ dstStride+16(FP), DX
	MOVQ srcStride+24(FP), CX
	MOVL size+32(FP), R8
	MOVL avg+36(FP), AX
	TESTL AX, AX
	JNZ   qpel_high00_avg_row

qpel_high00_put_row:
	MOVQ DI, R10
	MOVQ SI, R11
	MOVL size+32(FP), R9
qpel_high00_put_col:
	XORL AX, AX
	MOVW (R11), AX
	MOVW AX, (R10)
	ADDQ $2, R10
	ADDQ $2, R11
	DECL R9
	JNZ  qpel_high00_put_col
	ADDQ DX, DI
	ADDQ CX, SI
	DECL R8
	JNZ  qpel_high00_put_row
	RET

qpel_high00_avg_row:
	MOVQ DI, R10
	MOVQ SI, R11
	MOVL size+32(FP), R9
qpel_high00_avg_col:
	XORL AX, AX
	MOVW (R11), AX
	XORL BX, BX
	MOVW (R10), BX
	ADDL BX, AX
	ADDL $1, AX
	SHRL $1, AX
	MOVW AX, (R10)
	ADDQ $2, R10
	ADDQ $2, R11
	DECL R9
	JNZ  qpel_high00_avg_col
	ADDQ DX, DI
	ADDQ CX, SI
	DECL R8
	JNZ  qpel_high00_avg_row
	RET

// func h264QpelMCHighX0ASM(dst *uint8, src *uint8, dstStride int, srcStride int, size int32, mx int32, max int32, avg int32)
TEXT ·h264QpelMCHighX0ASM(SB), NOSPLIT, $0-48
	MOVQ dst+0(FP), DI
	MOVQ src+8(FP), SI
	MOVQ dstStride+16(FP), DX
	MOVQ srcStride+24(FP), CX
	MOVL size+32(FP), R8
	MOVL mx+36(FP), R14
	MOVL max+40(FP), R13
qpel_highx0_row:
	MOVQ DI, R11
	MOVQ SI, R12
	MOVL size+32(FP), R9
qpel_highx0_col:
	XORL AX, AX
	MOVW (R12), AX
	XORL BX, BX
	MOVW 2(R12), BX
	ADDL BX, AX
	IMULL $20, AX
	MOVL AX, R10
	XORL AX, AX
	MOVW -2(R12), AX
	XORL BX, BX
	MOVW 4(R12), BX
	ADDL BX, AX
	LEAL (AX)(AX*4), AX
	SUBL AX, R10
	XORL AX, AX
	MOVW -4(R12), AX
	XORL BX, BX
	MOVW 6(R12), BX
	ADDL BX, AX
	ADDL AX, R10
	ADDL $16, R10
	SARL $5, R10
	CMPL R10, $0
	JGE  qpel_highx0_nonnegative
	XORL R10, R10
	JMP  qpel_highx0_clip_done
qpel_highx0_nonnegative:
	CMPL R10, R13
	JLE  qpel_highx0_clip_done
	MOVL R13, R10
qpel_highx0_clip_done:
	CMPL R14, $2
	JEQ  qpel_highx0_pred_done
	CMPL R14, $1
	JNE  qpel_highx0_load_next
	XORL AX, AX
	MOVW (R12), AX
	JMP  qpel_highx0_l2
qpel_highx0_load_next:
	XORL AX, AX
	MOVW 2(R12), AX
qpel_highx0_l2:
	ADDL AX, R10
	ADDL $1, R10
	SHRL $1, R10
qpel_highx0_pred_done:
	MOVL avg+44(FP), AX
	TESTL AX, AX
	JZ    qpel_highx0_store
	XORL AX, AX
	MOVW (R11), AX
	ADDL AX, R10
	ADDL $1, R10
	SHRL $1, R10
qpel_highx0_store:
	MOVW R10, (R11)
	ADDQ $2, R11
	ADDQ $2, R12
	DECL R9
	JNZ  qpel_highx0_col
	ADDQ DX, DI
	ADDQ CX, SI
	DECL R8
	JNZ  qpel_highx0_row
	RET

// func h264QpelMCHigh0YASM(dst *uint8, src *uint8, dstStride int, srcStride int, size int32, my int32, max int32, avg int32)
TEXT ·h264QpelMCHigh0YASM(SB), NOSPLIT, $0-48
	MOVQ dst+0(FP), DI
	MOVQ src+8(FP), SI
	MOVQ dstStride+16(FP), DX
	MOVQ srcStride+24(FP), CX
	MOVL size+32(FP), R8
	MOVL my+36(FP), R14
	MOVL max+40(FP), R13
qpel_high0y_row:
	MOVQ DI, R11
	MOVQ SI, R12
	MOVL size+32(FP), R9
qpel_high0y_col:
	XORL AX, AX
	MOVW (R12), AX
	MOVQ R12, R15
	ADDQ CX, R15
	XORL BX, BX
	MOVW (R15), BX
	ADDL BX, AX
	IMULL $20, AX
	MOVL AX, R10
	MOVQ R12, R15
	SUBQ CX, R15
	XORL AX, AX
	MOVW (R15), AX
	MOVQ R12, R15
	ADDQ CX, R15
	ADDQ CX, R15
	XORL BX, BX
	MOVW (R15), BX
	ADDL BX, AX
	LEAL (AX)(AX*4), AX
	SUBL AX, R10
	MOVQ R12, R15
	SUBQ CX, R15
	SUBQ CX, R15
	XORL AX, AX
	MOVW (R15), AX
	MOVQ R12, R15
	ADDQ CX, R15
	ADDQ CX, R15
	ADDQ CX, R15
	XORL BX, BX
	MOVW (R15), BX
	ADDL BX, AX
	ADDL AX, R10
	ADDL $16, R10
	SARL $5, R10
	CMPL R10, $0
	JGE  qpel_high0y_nonnegative
	XORL R10, R10
	JMP  qpel_high0y_clip_done
qpel_high0y_nonnegative:
	CMPL R10, R13
	JLE  qpel_high0y_clip_done
	MOVL R13, R10
qpel_high0y_clip_done:
	CMPL R14, $2
	JEQ  qpel_high0y_pred_done
	CMPL R14, $1
	JNE  qpel_high0y_load_next
	XORL AX, AX
	MOVW (R12), AX
	JMP  qpel_high0y_l2
qpel_high0y_load_next:
	MOVQ R12, R15
	ADDQ CX, R15
	XORL AX, AX
	MOVW (R15), AX
qpel_high0y_l2:
	ADDL AX, R10
	ADDL $1, R10
	SHRL $1, R10
qpel_high0y_pred_done:
	MOVL avg+44(FP), AX
	TESTL AX, AX
	JZ    qpel_high0y_store
	XORL AX, AX
	MOVW (R11), AX
	ADDL AX, R10
	ADDL $1, R10
	SHRL $1, R10
qpel_high0y_store:
	MOVW R10, (R11)
	ADDQ $2, R11
	ADDQ $2, R12
	DECL R9
	JNZ  qpel_high0y_col
	ADDQ DX, DI
	ADDQ CX, SI
	DECL R8
	JNZ  qpel_high0y_row
	RET

// func h264QpelMCHigh22ASM(dst *uint8, src *uint8, dstStride int, srcStride int, size int32, max int32, avg int32)
TEXT ·h264QpelMCHigh22ASM(SB), NOSPLIT, $32-48
	MOVQ dst+0(FP), DI
	MOVQ src+8(FP), SI
	MOVQ dstStride+16(FP), DX
	MOVQ srcStride+24(FP), CX
	MOVL size+32(FP), R8
	MOVL max+36(FP), R14
	MOVL avg+40(FP), R15
qpel_high22_row:
	MOVQ DI, R11
	MOVQ SI, R12
	MOVL size+32(FP), R9
qpel_high22_col:
	MOVQ R12, R13
	SUBQ CX, R13
	SUBQ CX, R13
	XORL AX, AX
	MOVW (R13), AX
	XORL BX, BX
	MOVW 2(R13), BX
	ADDL BX, AX
	IMULL $20, AX
	MOVL AX, R10
	XORL AX, AX
	MOVW -2(R13), AX
	XORL BX, BX
	MOVW 4(R13), BX
	ADDL BX, AX
	LEAL (AX)(AX*4), AX
	SUBL AX, R10
	XORL AX, AX
	MOVW -4(R13), AX
	XORL BX, BX
	MOVW 6(R13), BX
	ADDL BX, AX
	ADDL AX, R10
	MOVL R10, 0(SP)

	MOVQ R12, R13
	SUBQ CX, R13
	XORL AX, AX
	MOVW (R13), AX
	XORL BX, BX
	MOVW 2(R13), BX
	ADDL BX, AX
	IMULL $20, AX
	MOVL AX, R10
	XORL AX, AX
	MOVW -2(R13), AX
	XORL BX, BX
	MOVW 4(R13), BX
	ADDL BX, AX
	LEAL (AX)(AX*4), AX
	SUBL AX, R10
	XORL AX, AX
	MOVW -4(R13), AX
	XORL BX, BX
	MOVW 6(R13), BX
	ADDL BX, AX
	ADDL AX, R10
	MOVL R10, 4(SP)

	MOVQ R12, R13
	XORL AX, AX
	MOVW (R13), AX
	XORL BX, BX
	MOVW 2(R13), BX
	ADDL BX, AX
	IMULL $20, AX
	MOVL AX, R10
	XORL AX, AX
	MOVW -2(R13), AX
	XORL BX, BX
	MOVW 4(R13), BX
	ADDL BX, AX
	LEAL (AX)(AX*4), AX
	SUBL AX, R10
	XORL AX, AX
	MOVW -4(R13), AX
	XORL BX, BX
	MOVW 6(R13), BX
	ADDL BX, AX
	ADDL AX, R10
	MOVL R10, 8(SP)

	MOVQ R12, R13
	ADDQ CX, R13
	XORL AX, AX
	MOVW (R13), AX
	XORL BX, BX
	MOVW 2(R13), BX
	ADDL BX, AX
	IMULL $20, AX
	MOVL AX, R10
	XORL AX, AX
	MOVW -2(R13), AX
	XORL BX, BX
	MOVW 4(R13), BX
	ADDL BX, AX
	LEAL (AX)(AX*4), AX
	SUBL AX, R10
	XORL AX, AX
	MOVW -4(R13), AX
	XORL BX, BX
	MOVW 6(R13), BX
	ADDL BX, AX
	ADDL AX, R10
	MOVL R10, 12(SP)

	MOVQ R12, R13
	ADDQ CX, R13
	ADDQ CX, R13
	XORL AX, AX
	MOVW (R13), AX
	XORL BX, BX
	MOVW 2(R13), BX
	ADDL BX, AX
	IMULL $20, AX
	MOVL AX, R10
	XORL AX, AX
	MOVW -2(R13), AX
	XORL BX, BX
	MOVW 4(R13), BX
	ADDL BX, AX
	LEAL (AX)(AX*4), AX
	SUBL AX, R10
	XORL AX, AX
	MOVW -4(R13), AX
	XORL BX, BX
	MOVW 6(R13), BX
	ADDL BX, AX
	ADDL AX, R10
	MOVL R10, 16(SP)

	MOVQ R12, R13
	ADDQ CX, R13
	ADDQ CX, R13
	ADDQ CX, R13
	XORL AX, AX
	MOVW (R13), AX
	XORL BX, BX
	MOVW 2(R13), BX
	ADDL BX, AX
	IMULL $20, AX
	MOVL AX, R10
	XORL AX, AX
	MOVW -2(R13), AX
	XORL BX, BX
	MOVW 4(R13), BX
	ADDL BX, AX
	LEAL (AX)(AX*4), AX
	SUBL AX, R10
	XORL AX, AX
	MOVW -4(R13), AX
	XORL BX, BX
	MOVW 6(R13), BX
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
	JGE  qpel_high22_nonnegative
	XORL R10, R10
	JMP  qpel_high22_clip_done
qpel_high22_nonnegative:
	CMPL R10, R14
	JLE  qpel_high22_clip_done
	MOVL R14, R10
qpel_high22_clip_done:
	TESTL R15, R15
	JZ    qpel_high22_store
	XORL AX, AX
	MOVW (R11), AX
	ADDL AX, R10
	ADDL $1, R10
	SHRL $1, R10
qpel_high22_store:
	MOVW R10, (R11)
	ADDQ $2, R11
	ADDQ $2, R12
	DECL R9
	JNZ  qpel_high22_col
	ADDQ DX, DI
	ADDQ CX, SI
	DECL R8
	JNZ  qpel_high22_row
	RET
