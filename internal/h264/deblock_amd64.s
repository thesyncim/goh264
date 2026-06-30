// SPDX-License-Identifier: LGPL-2.1-or-later
//go:build !purego && amd64

#include "textflag.h"

DATA ·h264DeblockPB1+0(SB)/8, $0x0101010101010101
DATA ·h264DeblockPB1+8(SB)/8, $0x0101010101010101
GLOBL ·h264DeblockPB1(SB), RODATA, $16

DATA ·h264DeblockPB3+0(SB)/8, $0x0303030303030303
DATA ·h264DeblockPB3+8(SB)/8, $0x0303030303030303
GLOBL ·h264DeblockPB3(SB), RODATA, $16

DATA ·h264DeblockPBA1+0(SB)/8, $0xa1a1a1a1a1a1a1a1
DATA ·h264DeblockPBA1+8(SB)/8, $0xa1a1a1a1a1a1a1a1
GLOBL ·h264DeblockPBA1(SB), RODATA, $16

// func h264VLoopFilterLuma8ASM(pix *uint8, stride int, alpha int32, beta int32, tc0 *int8)
TEXT ·h264VLoopFilterLuma8ASM(SB), NOSPLIT, $0-32
	MOVQ pix+0(FP), DI
	MOVQ stride+8(FP), SI
	MOVL alpha+16(FP), AX
	MOVL beta+20(FP), BX
	MOVQ tc0+24(FP), CX

	TESTL AX, AX
	JLE   luma8_v_ret
	TESTL BX, BX
	JLE   luma8_v_ret
	DECL AX
	DECL BX
	MOVQ SI, DX
	LEAQ (SI)(SI*2), R8
	MOVQ DI, R9
	SUBQ R8, R9

	MOVOU (R9)(SI*1), X0   // p1
	MOVOU (R9)(SI*2), X1   // p0
	MOVOU (DI), X2         // q0
	MOVOU (DI)(SI*1), X3   // q1

	MOVD AX, X4
	PSHUFLW $0, X4, X4
	PUNPCKLQDQ X4, X4
	PACKUSWB X4, X4        // alpha - 1
	MOVD BX, X5
	PSHUFLW $0, X5, X5
	PUNPCKLQDQ X5, X5
	PACKUSWB X5, X5        // beta - 1

	MOVOU X2, X6
	MOVOU X1, X7
	PSUBUSB X1, X6
	PSUBUSB X2, X7
	POR X6, X7
	PSUBUSB X4, X7

	MOVOU X1, X6
	MOVOU X0, X4
	PSUBUSB X0, X6
	PSUBUSB X1, X4
	POR X6, X4
	PSUBUSB X5, X4
	POR X4, X7

	MOVOU X2, X6
	MOVOU X3, X4
	PSUBUSB X3, X6
	PSUBUSB X2, X4
	POR X6, X4
	PSUBUSB X5, X4
	POR X4, X7
	PXOR X6, X6
	PCMPEQB X6, X7        // base mask

	MOVD (CX), X8
	PUNPCKLBW X8, X8
	PUNPCKLBW X8, X8
	MOVOU X8, X6
	PCMPEQB X9, X9
	PCMPGTB X9, X6        // tc0 >= 0
	PAND X7, X6           // base mask with inactive tc0 lanes cleared
	MOVOU X6, X9
	PAND X9, X8           // masked tc0

	MOVOU (R9), X3        // p2
	MOVOU X3, X7
	MOVOU X1, X6
	PSUBUSB X1, X7
	PSUBUSB X3, X6
	PSUBUSB X5, X7
	PSUBUSB X5, X6
	PCMPEQB X7, X6
	PAND X9, X6           // |p2-p0| < beta and base mask
	MOVOU X8, X7
	PSUBB X6, X7          // tc + p-side increment
	PAND X8, X6           // p1 clip tc

	MOVOU X1, X4
	PAVGB X2, X4
	PAVGB X4, X3
	MOVOU (R9), X10
	PXOR X10, X4
	PAND ·h264DeblockPB1(SB), X4
	PSUBUSB X4, X3
	MOVOU X0, X10
	PSUBUSB X6, X10
	PADDUSB X0, X6
	PMAXUB X10, X3
	PMINUB X6, X3
	MOVOU X3, (R9)(SI*1)

	MOVOU (DI)(SI*2), X4  // q2
	MOVOU X4, X3
	MOVOU X2, X6
	PSUBUSB X2, X3
	PSUBUSB X4, X6
	PSUBUSB X5, X3
	PSUBUSB X5, X6
	PCMPEQB X3, X6
	PAND X9, X6           // |q2-q0| < beta and base mask
	PAND X6, X8           // q1 clip tc
	PSUBB X6, X7          // final tc
	MOVOU (DI)(SI*1), X3  // q1

	MOVOU X1, X6
	PAVGB X2, X6
	PAVGB X6, X4
	MOVOU (DI)(SI*2), X10
	PXOR X10, X6
	PAND ·h264DeblockPB1(SB), X6
	PSUBUSB X6, X4
	MOVOU X3, X10
	PSUBUSB X8, X10
	PADDUSB X3, X8
	PMAXUB X10, X4
	PMINUB X8, X4
	MOVOU X4, (DI)(SI*1)

	PCMPEQB X4, X4
	MOVOU X1, X5
	PXOR X2, X5
	PXOR X4, X3
	PAND ·h264DeblockPB1(SB), X5
	PAVGB X0, X3
	PXOR X1, X4
	PAVGB ·h264DeblockPB3(SB), X3
	PAVGB X2, X4
	PAVGB X5, X3
	MOVOU ·h264DeblockPBA1(SB), X6
	PADDUSB X4, X3
	PSUBUSB X3, X6
	PSUBUSB ·h264DeblockPBA1(SB), X3
	PMINUB X7, X6
	PMINUB X7, X3
	PSUBUSB X6, X1
	PSUBUSB X3, X2
	PADDUSB X3, X1
	PADDUSB X6, X2

	MOVOU X1, (R9)(SI*2)
	MOVOU X2, (DI)
luma8_v_ret:
	RET

// func h264HLoopFilterLuma8ASM(pix *uint8, stride int, alpha int32, beta int32, tc0 *int8)
TEXT ·h264HLoopFilterLuma8ASM(SB), NOSPLIT, $16-32
	MOVQ pix+0(FP), R11
	MOVQ $1, R12
	MOVQ stride+8(FP), R13
	MOVL alpha+16(FP), R14
	MOVQ tc0+24(FP), SI
	MOVQ $4, 0(SP)
	JMP  luma8_h_group

luma8_h_group:
	XORL R10, R10
	MOVB (SI), R10
	CMPL R10, $128
	JLT  luma8_h_tc_ready
	SUBL $256, R10
luma8_h_tc_ready:
	CMPL R10, $0
	JGE  luma8_h_active
	LEAQ (R11)(R13*4), R11
	JMP  luma8_h_next_group
luma8_h_active:
	MOVQ $4, R15
luma8_h_inner:
	MOVQ R11, DI
	SUBQ R12, DI
	XORL AX, AX
	MOVB (DI), AX
	MOVQ DI, CX
	SUBQ R12, CX
	XORL BX, BX
	MOVB (CX), BX
	MOVQ CX, DI
	SUBQ R12, DI
	XORL CX, CX
	MOVB (DI), CX
	XORL DX, DX
	MOVB (R11), DX
	MOVQ R11, DI
	ADDQ R12, DI
	XORL R9, R9
	MOVB (DI), R9

	MOVL AX, DI
	SUBL DX, DI
	CMPL DI, $0
	JGE  luma8_h_abs0_done
	NEGL DI
luma8_h_abs0_done:
	CMPL DI, R14
	JGE  luma8_h_sample_done

	MOVL BX, DI
	SUBL AX, DI
	CMPL DI, $0
	JGE  luma8_h_abs1_done
	NEGL DI
luma8_h_abs1_done:
	MOVL beta+20(FP), R8
	CMPL DI, R8
	JGE  luma8_h_sample_done

	MOVL R9, DI
	SUBL DX, DI
	CMPL DI, $0
	JGE  luma8_h_abs2_done
	NEGL DI
luma8_h_abs2_done:
	MOVL beta+20(FP), R8
	CMPL DI, R8
	JGE  luma8_h_sample_done

	MOVL R10, 8(SP)

	MOVL CX, R8
	SUBL AX, R8
	CMPL R8, $0
	JGE  luma8_h_p_abs_done
	NEGL R8
luma8_h_p_abs_done:
	MOVL beta+20(FP), R8
	MOVL CX, R8
	SUBL AX, R8
	CMPL R8, $0
	JGE  luma8_h_p_abs_cmp
	NEGL R8
luma8_h_p_abs_cmp:
	MOVL beta+20(FP), DI
	CMPL R8, DI
	JGE  luma8_h_p_done
	CMPL R10, $0
	JE   luma8_h_p_inc
	MOVL AX, R8
	ADDL DX, R8
	ADDL $1, R8
	SARL $1, R8
	ADDL CX, R8
	SARL $1, R8
	SUBL BX, R8
	MOVL R10, DI
	NEGL DI
	CMPL R8, DI
	JGE  luma8_h_p_not_low
	MOVL DI, R8
luma8_h_p_not_low:
	CMPL R8, R10
	JLE  luma8_h_p_clip_done
	MOVL R10, R8
luma8_h_p_clip_done:
	ADDL BX, R8
	MOVQ R11, DI
	SUBQ R12, DI
	SUBQ R12, DI
	MOVB R8, (DI)
luma8_h_p_inc:
	ADDL $1, 8(SP)
luma8_h_p_done:
	MOVQ R11, DI
	ADDQ R12, DI
	ADDQ R12, DI
	XORL R8, R8
	MOVB (DI), R8
	MOVL R8, DI
	SUBL DX, DI
	CMPL DI, $0
	JGE  luma8_h_q_abs_done
	NEGL DI
luma8_h_q_abs_done:
	MOVL beta+20(FP), R8
	CMPL DI, R8
	JGE  luma8_h_delta
	CMPL R10, $0
	JE   luma8_h_q_inc
	MOVQ R11, DI
	ADDQ R12, DI
	ADDQ R12, DI
	XORL R8, R8
	MOVB (DI), R8
	MOVL AX, DI
	ADDL DX, DI
	ADDL $1, DI
	SARL $1, DI
	ADDL R8, DI
	SARL $1, DI
	SUBL R9, DI
	MOVL R10, R8
	NEGL R8
	CMPL DI, R8
	JGE  luma8_h_q_not_low
	MOVL R8, DI
luma8_h_q_not_low:
	CMPL DI, R10
	JLE  luma8_h_q_clip_done
	MOVL R10, DI
luma8_h_q_clip_done:
	ADDL R9, DI
	MOVQ R11, R8
	ADDQ R12, R8
	MOVB DI, (R8)
luma8_h_q_inc:
	ADDL $1, 8(SP)

luma8_h_delta:
	MOVL DX, R8
	SUBL AX, R8
	SHLL $2, R8
	MOVL BX, DI
	SUBL R9, DI
	ADDL DI, R8
	ADDL $4, R8
	SARL $3, R8
	MOVL 8(SP), DI
	MOVL DI, CX
	NEGL CX
	CMPL R8, CX
	JGE  luma8_h_delta_not_low
	MOVL CX, R8
luma8_h_delta_not_low:
	CMPL R8, DI
	JLE  luma8_h_delta_clip_done
	MOVL DI, R8
luma8_h_delta_clip_done:
	MOVL AX, DI
	ADDL R8, DI
	CMPL DI, $0
	JGE  luma8_h_p0_nonnegative
	XORL DI, DI
	JMP  luma8_h_p0_clip_done
luma8_h_p0_nonnegative:
	CMPL DI, $255
	JLE  luma8_h_p0_clip_done
	MOVL $255, DI
luma8_h_p0_clip_done:
	MOVQ R11, CX
	SUBQ R12, CX
	MOVB DI, (CX)

	MOVL DX, DI
	SUBL R8, DI
	CMPL DI, $0
	JGE  luma8_h_q0_nonnegative
	XORL DI, DI
	JMP  luma8_h_q0_clip_done
luma8_h_q0_nonnegative:
	CMPL DI, $255
	JLE  luma8_h_q0_clip_done
	MOVL $255, DI
luma8_h_q0_clip_done:
	MOVB DI, (R11)

luma8_h_sample_done:
	ADDQ R13, R11
	DECQ R15
	JNZ  luma8_h_inner

luma8_h_next_group:
	INCQ SI
	DECQ 0(SP)
	JNZ  luma8_h_group
	RET
