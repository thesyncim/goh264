// SPDX-License-Identifier: LGPL-2.1-or-later
//go:build !purego && arm64

#include "textflag.h"

// func h264VLoopFilterLuma8ASM(pix *uint8, stride int, alpha int32, beta int32, tc0 *int8)
TEXT ·h264VLoopFilterLuma8ASM(SB), NOSPLIT, $0-32
	MOVD pix+0(FP), R11
	MOVD stride+8(FP), R12
	MOVD $1, R13
	MOVW alpha+16(FP), R14
	MOVW beta+20(FP), R20
	MOVD tc0+24(FP), R10
	MOVD $4, R19
	B    luma8_v_group

luma8_v_group:
	MOVBU (R10), R16
	CMPW  $128, R16
	BLT   luma8_v_tc_ready
	SUBW  $256, R16, R16
luma8_v_tc_ready:
	CMPW $0, R16
	BGE  luma8_v_active
	ADD  R13, R11, R11
	ADD  R13, R11, R11
	ADD  R13, R11, R11
	ADD  R13, R11, R11
	B    luma8_v_next_group
luma8_v_active:
	MOVD $4, R15
luma8_v_inner:
	SUB   R12, R11, R7
	MOVBU (R7), R0
	SUB   R12, R7, R7
	MOVBU (R7), R1
	SUB   R12, R7, R7
	MOVBU (R7), R2
	MOVBU (R11), R3
	ADD   R12, R11, R7
	MOVBU (R7), R4

	SUBW R3, R0, R6
	CMPW $0, R6
	BGE  luma8_v_abs0_done
	NEGW R6, R6
luma8_v_abs0_done:
	CMPW R14, R6
	BGE  luma8_v_sample_done

	SUBW R0, R1, R6
	CMPW $0, R6
	BGE  luma8_v_abs1_done
	NEGW R6, R6
luma8_v_abs1_done:
	CMPW R20, R6
	BGE  luma8_v_sample_done

	SUBW R3, R4, R6
	CMPW $0, R6
	BGE  luma8_v_abs2_done
	NEGW R6, R6
luma8_v_abs2_done:
	CMPW R20, R6
	BGE  luma8_v_sample_done

	MOVW R16, R17

	SUBW R0, R2, R6
	CMPW $0, R6
	BGE  luma8_v_p_abs_done
	NEGW R6, R6
luma8_v_p_abs_done:
	CMPW R20, R6
	BGE  luma8_v_p_done
	CMPW $0, R16
	BEQ  luma8_v_p_inc
	ADDW R3, R0, R6
	ADDW $1, R6, R6
	ASRW $1, R6, R6
	ADDW R2, R6, R6
	ASRW $1, R6, R6
	SUBW R1, R6, R6
	NEGW R16, R8
	CMPW R8, R6
	BGE  luma8_v_p_not_low
	MOVW R8, R6
luma8_v_p_not_low:
	CMPW R16, R6
	BLE  luma8_v_p_clip_done
	MOVW R16, R6
luma8_v_p_clip_done:
	ADDW R1, R6, R6
	SUB  R12, R11, R7
	SUB  R12, R7, R7
	MOVB R6, (R7)
luma8_v_p_inc:
	ADDW $1, R17, R17
luma8_v_p_done:
	ADD   R12, R11, R7
	ADD   R12, R7, R7
	MOVBU (R7), R5
	SUBW  R3, R5, R6
	CMPW  $0, R6
	BGE   luma8_v_q_abs_done
	NEGW  R6, R6
luma8_v_q_abs_done:
	CMPW R20, R6
	BGE  luma8_v_delta
	CMPW $0, R16
	BEQ  luma8_v_q_inc
	ADDW R3, R0, R6
	ADDW $1, R6, R6
	ASRW $1, R6, R6
	ADDW R5, R6, R6
	ASRW $1, R6, R6
	SUBW R4, R6, R6
	NEGW R16, R8
	CMPW R8, R6
	BGE  luma8_v_q_not_low
	MOVW R8, R6
luma8_v_q_not_low:
	CMPW R16, R6
	BLE  luma8_v_q_clip_done
	MOVW R16, R6
luma8_v_q_clip_done:
	ADDW R4, R6, R6
	ADD  R12, R11, R7
	MOVB R6, (R7)
luma8_v_q_inc:
	ADDW $1, R17, R17

luma8_v_delta:
	SUBW R0, R3, R6
	LSLW $2, R6, R6
	SUBW R4, R1, R8
	ADDW R8, R6, R6
	ADDW $4, R6, R6
	ASRW $3, R6, R6
	NEGW R17, R8
	CMPW R8, R6
	BGE  luma8_v_delta_not_low
	MOVW R8, R6
luma8_v_delta_not_low:
	CMPW R17, R6
	BLE  luma8_v_delta_clip_done
	MOVW R17, R6
luma8_v_delta_clip_done:
	ADDW R6, R0, R8
	CMPW $0, R8
	BGE  luma8_v_p0_nonnegative
	MOVW ZR, R8
	B    luma8_v_p0_clip_done
luma8_v_p0_nonnegative:
	CMPW $255, R8
	BLE  luma8_v_p0_clip_done
	MOVW $255, R8
luma8_v_p0_clip_done:
	SUB  R12, R11, R7
	MOVB R8, (R7)

	SUBW R6, R3, R8
	CMPW $0, R8
	BGE  luma8_v_q0_nonnegative
	MOVW ZR, R8
	B    luma8_v_q0_clip_done
luma8_v_q0_nonnegative:
	CMPW $255, R8
	BLE  luma8_v_q0_clip_done
	MOVW $255, R8
luma8_v_q0_clip_done:
	MOVB R8, (R11)

luma8_v_sample_done:
	ADD  R13, R11, R11
	SUBS $1, R15, R15
	BNE  luma8_v_inner

luma8_v_next_group:
	ADD  $1, R10, R10
	SUBS $1, R19, R19
	BNE  luma8_v_group
	RET

// func h264HLoopFilterLuma8ASM(pix *uint8, stride int, alpha int32, beta int32, tc0 *int8)
TEXT ·h264HLoopFilterLuma8ASM(SB), NOSPLIT, $0-32
	MOVD pix+0(FP), R11
	MOVD $1, R12
	MOVD stride+8(FP), R13
	MOVW alpha+16(FP), R14
	MOVW beta+20(FP), R20
	MOVD tc0+24(FP), R10
	MOVD $4, R19
	B    luma8_h_group

luma8_h_group:
	MOVBU (R10), R16
	CMPW  $128, R16
	BLT   luma8_h_tc_ready
	SUBW  $256, R16, R16
luma8_h_tc_ready:
	CMPW $0, R16
	BGE  luma8_h_active
	ADD  R13, R11, R11
	ADD  R13, R11, R11
	ADD  R13, R11, R11
	ADD  R13, R11, R11
	B    luma8_h_next_group
luma8_h_active:
	MOVD $4, R15
luma8_h_inner:
	SUB   R12, R11, R7
	MOVBU (R7), R0
	SUB   R12, R7, R7
	MOVBU (R7), R1
	SUB   R12, R7, R7
	MOVBU (R7), R2
	MOVBU (R11), R3
	ADD   R12, R11, R7
	MOVBU (R7), R4

	SUBW R3, R0, R6
	CMPW $0, R6
	BGE  luma8_h_abs0_done
	NEGW R6, R6
luma8_h_abs0_done:
	CMPW R14, R6
	BGE  luma8_h_sample_done

	SUBW R0, R1, R6
	CMPW $0, R6
	BGE  luma8_h_abs1_done
	NEGW R6, R6
luma8_h_abs1_done:
	CMPW R20, R6
	BGE  luma8_h_sample_done

	SUBW R3, R4, R6
	CMPW $0, R6
	BGE  luma8_h_abs2_done
	NEGW R6, R6
luma8_h_abs2_done:
	CMPW R20, R6
	BGE  luma8_h_sample_done

	MOVW R16, R17

	SUBW R0, R2, R6
	CMPW $0, R6
	BGE  luma8_h_p_abs_done
	NEGW R6, R6
luma8_h_p_abs_done:
	CMPW R20, R6
	BGE  luma8_h_p_done
	CMPW $0, R16
	BEQ  luma8_h_p_inc
	ADDW R3, R0, R6
	ADDW $1, R6, R6
	ASRW $1, R6, R6
	ADDW R2, R6, R6
	ASRW $1, R6, R6
	SUBW R1, R6, R6
	NEGW R16, R8
	CMPW R8, R6
	BGE  luma8_h_p_not_low
	MOVW R8, R6
luma8_h_p_not_low:
	CMPW R16, R6
	BLE  luma8_h_p_clip_done
	MOVW R16, R6
luma8_h_p_clip_done:
	ADDW R1, R6, R6
	SUB  R12, R11, R7
	SUB  R12, R7, R7
	MOVB R6, (R7)
luma8_h_p_inc:
	ADDW $1, R17, R17
luma8_h_p_done:
	ADD   R12, R11, R7
	ADD   R12, R7, R7
	MOVBU (R7), R5
	SUBW  R3, R5, R6
	CMPW  $0, R6
	BGE   luma8_h_q_abs_done
	NEGW  R6, R6
luma8_h_q_abs_done:
	CMPW R20, R6
	BGE  luma8_h_delta
	CMPW $0, R16
	BEQ  luma8_h_q_inc
	ADDW R3, R0, R6
	ADDW $1, R6, R6
	ASRW $1, R6, R6
	ADDW R5, R6, R6
	ASRW $1, R6, R6
	SUBW R4, R6, R6
	NEGW R16, R8
	CMPW R8, R6
	BGE  luma8_h_q_not_low
	MOVW R8, R6
luma8_h_q_not_low:
	CMPW R16, R6
	BLE  luma8_h_q_clip_done
	MOVW R16, R6
luma8_h_q_clip_done:
	ADDW R4, R6, R6
	ADD  R12, R11, R7
	MOVB R6, (R7)
luma8_h_q_inc:
	ADDW $1, R17, R17

luma8_h_delta:
	SUBW R0, R3, R6
	LSLW $2, R6, R6
	SUBW R4, R1, R8
	ADDW R8, R6, R6
	ADDW $4, R6, R6
	ASRW $3, R6, R6
	NEGW R17, R8
	CMPW R8, R6
	BGE  luma8_h_delta_not_low
	MOVW R8, R6
luma8_h_delta_not_low:
	CMPW R17, R6
	BLE  luma8_h_delta_clip_done
	MOVW R17, R6
luma8_h_delta_clip_done:
	ADDW R6, R0, R8
	CMPW $0, R8
	BGE  luma8_h_p0_nonnegative
	MOVW ZR, R8
	B    luma8_h_p0_clip_done
luma8_h_p0_nonnegative:
	CMPW $255, R8
	BLE  luma8_h_p0_clip_done
	MOVW $255, R8
luma8_h_p0_clip_done:
	SUB  R12, R11, R7
	MOVB R8, (R7)

	SUBW R6, R3, R8
	CMPW $0, R8
	BGE  luma8_h_q0_nonnegative
	MOVW ZR, R8
	B    luma8_h_q0_clip_done
luma8_h_q0_nonnegative:
	CMPW $255, R8
	BLE  luma8_h_q0_clip_done
	MOVW $255, R8
luma8_h_q0_clip_done:
	MOVB R8, (R11)

luma8_h_sample_done:
	ADD  R13, R11, R11
	SUBS $1, R15, R15
	BNE  luma8_h_inner

luma8_h_next_group:
	ADD  $1, R10, R10
	SUBS $1, R19, R19
	BNE  luma8_h_group
	RET
