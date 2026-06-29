// SPDX-License-Identifier: LGPL-2.1-or-later
//go:build !purego && amd64

#include "textflag.h"

// func h264VLoopFilterLuma8ASM(pix *uint8, stride int, alpha int32, beta int32, tc0 *int8)
TEXT ·h264VLoopFilterLuma8ASM(SB), NOSPLIT, $16-32
	MOVQ pix+0(FP), R11
	MOVQ stride+8(FP), R12
	MOVQ $1, R13
	MOVL alpha+16(FP), R14
	MOVQ tc0+24(FP), SI
	MOVQ $4, 0(SP)
	JMP  luma8_v_group

luma8_v_group:
	XORL R10, R10
	MOVB (SI), R10
	CMPL R10, $128
	JLT  luma8_v_tc_ready
	SUBL $256, R10
luma8_v_tc_ready:
	CMPL R10, $0
	JGE  luma8_v_active
	LEAQ (R11)(R13*4), R11
	JMP  luma8_v_next_group
luma8_v_active:
	MOVQ $4, R15
luma8_v_inner:
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
	JGE  luma8_v_abs0_done
	NEGL DI
luma8_v_abs0_done:
	CMPL DI, R14
	JGE  luma8_v_sample_done

	MOVL BX, DI
	SUBL AX, DI
	CMPL DI, $0
	JGE  luma8_v_abs1_done
	NEGL DI
luma8_v_abs1_done:
	MOVL beta+20(FP), R8
	CMPL DI, R8
	JGE  luma8_v_sample_done

	MOVL R9, DI
	SUBL DX, DI
	CMPL DI, $0
	JGE  luma8_v_abs2_done
	NEGL DI
luma8_v_abs2_done:
	MOVL beta+20(FP), R8
	CMPL DI, R8
	JGE  luma8_v_sample_done

	MOVL R10, 8(SP)

	MOVL CX, R8
	SUBL AX, R8
	CMPL R8, $0
	JGE  luma8_v_p_abs_done
	NEGL R8
luma8_v_p_abs_done:
	MOVL beta+20(FP), R8
	MOVL CX, R8
	SUBL AX, R8
	CMPL R8, $0
	JGE  luma8_v_p_abs_cmp
	NEGL R8
luma8_v_p_abs_cmp:
	MOVL beta+20(FP), DI
	CMPL R8, DI
	JGE  luma8_v_p_done
	CMPL R10, $0
	JE   luma8_v_p_inc
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
	JGE  luma8_v_p_not_low
	MOVL DI, R8
luma8_v_p_not_low:
	CMPL R8, R10
	JLE  luma8_v_p_clip_done
	MOVL R10, R8
luma8_v_p_clip_done:
	ADDL BX, R8
	MOVQ R11, DI
	SUBQ R12, DI
	SUBQ R12, DI
	MOVB R8, (DI)
luma8_v_p_inc:
	ADDL $1, 8(SP)
luma8_v_p_done:
	MOVQ R11, DI
	ADDQ R12, DI
	ADDQ R12, DI
	XORL R8, R8
	MOVB (DI), R8
	MOVL R8, DI
	SUBL DX, DI
	CMPL DI, $0
	JGE  luma8_v_q_abs_done
	NEGL DI
luma8_v_q_abs_done:
	MOVL beta+20(FP), R8
	CMPL DI, R8
	JGE  luma8_v_delta
	CMPL R10, $0
	JE   luma8_v_q_inc
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
	JGE  luma8_v_q_not_low
	MOVL R8, DI
luma8_v_q_not_low:
	CMPL DI, R10
	JLE  luma8_v_q_clip_done
	MOVL R10, DI
luma8_v_q_clip_done:
	ADDL R9, DI
	MOVQ R11, R8
	ADDQ R12, R8
	MOVB DI, (R8)
luma8_v_q_inc:
	ADDL $1, 8(SP)

luma8_v_delta:
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
	JGE  luma8_v_delta_not_low
	MOVL CX, R8
luma8_v_delta_not_low:
	CMPL R8, DI
	JLE  luma8_v_delta_clip_done
	MOVL DI, R8
luma8_v_delta_clip_done:
	MOVL AX, DI
	ADDL R8, DI
	CMPL DI, $0
	JGE  luma8_v_p0_nonnegative
	XORL DI, DI
	JMP  luma8_v_p0_clip_done
luma8_v_p0_nonnegative:
	CMPL DI, $255
	JLE  luma8_v_p0_clip_done
	MOVL $255, DI
luma8_v_p0_clip_done:
	MOVQ R11, CX
	SUBQ R12, CX
	MOVB DI, (CX)

	MOVL DX, DI
	SUBL R8, DI
	CMPL DI, $0
	JGE  luma8_v_q0_nonnegative
	XORL DI, DI
	JMP  luma8_v_q0_clip_done
luma8_v_q0_nonnegative:
	CMPL DI, $255
	JLE  luma8_v_q0_clip_done
	MOVL $255, DI
luma8_v_q0_clip_done:
	MOVB DI, (R11)

luma8_v_sample_done:
	ADDQ R13, R11
	DECQ R15
	JNZ  luma8_v_inner

luma8_v_next_group:
	INCQ SI
	DECQ 0(SP)
	JNZ  luma8_v_group
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
