// SPDX-License-Identifier: LGPL-2.1-or-later
//go:build !purego && arm64

#include "textflag.h"

// func h264QpelMCHigh00ASM(dst *uint8, src *uint8, dstStride int, srcStride int, size int32, avg int32)
TEXT ·h264QpelMCHigh00ASM(SB), NOSPLIT, $0-40
	MOVD dst+0(FP), R0
	MOVD src+8(FP), R1
	MOVD dstStride+16(FP), R2
	MOVD srcStride+24(FP), R3
	MOVW size+32(FP), R4
	MOVW avg+36(FP), R5
	CBNZW R5, qpel_high00_avg_row

qpel_high00_put_row:
	MOVD R0, R10
	MOVD R1, R11
	MOVW size+32(FP), R9
qpel_high00_put_col:
	MOVHU (R11), R12
	MOVH  R12, (R10)
	ADD   $2, R10, R10
	ADD   $2, R11, R11
	SUBW  $1, R9, R9
	CBNZW R9, qpel_high00_put_col
	ADD   R2, R0, R0
	ADD   R3, R1, R1
	SUBW  $1, R4, R4
	CBNZW R4, qpel_high00_put_row
	RET

qpel_high00_avg_row:
	MOVD R0, R10
	MOVD R1, R11
	MOVW size+32(FP), R9
qpel_high00_avg_col:
	MOVHU (R11), R12
	MOVHU (R10), R13
	ADDW  R13, R12, R12
	ADDW  $1, R12, R12
	LSRW  $1, R12, R12
	MOVH  R12, (R10)
	ADD   $2, R10, R10
	ADD   $2, R11, R11
	SUBW  $1, R9, R9
	CBNZW R9, qpel_high00_avg_col
	ADD   R2, R0, R0
	ADD   R3, R1, R1
	SUBW  $1, R4, R4
	CBNZW R4, qpel_high00_avg_row
	RET

// func h264QpelMCHighX0ASM(dst *uint8, src *uint8, dstStride int, srcStride int, size int32, mx int32, max int32, avg int32)
TEXT ·h264QpelMCHighX0ASM(SB), NOSPLIT, $0-48
	MOVD dst+0(FP), R0
	MOVD src+8(FP), R1
	MOVD dstStride+16(FP), R2
	MOVD srcStride+24(FP), R3
	MOVW size+32(FP), R4
	MOVW mx+36(FP), R14
	MOVW max+40(FP), R15
	MOVW avg+44(FP), R17
qpel_highx0_row:
	MOVD R0, R10
	MOVD R1, R11
	MOVW size+32(FP), R9
qpel_highx0_col:
	MOVHU (R11), R5
	MOVHU 2(R11), R6
	ADDW  R6, R5, R5
	LSLW  $4, R5, R12
	ADDW  R5<<2, R12, R12
	MOVHU -2(R11), R5
	MOVHU 4(R11), R6
	ADDW  R6, R5, R5
	ADDW  R5<<2, R5, R5
	SUBW  R5, R12, R12
	MOVHU -4(R11), R5
	MOVHU 6(R11), R6
	ADDW  R6, R5, R5
	ADDW  R5, R12, R12
	ADDW  $16, R12, R12
	ASRW  $5, R12, R12
	CMPW  $0, R12
	BGE   qpel_highx0_nonnegative
	MOVW  ZR, R12
	B     qpel_highx0_clip_done
qpel_highx0_nonnegative:
	CMPW R15, R12
	BLE  qpel_highx0_clip_done
	MOVW R15, R12
qpel_highx0_clip_done:
	CMPW $2, R14
	BEQ  qpel_highx0_pred_done
	CMPW $1, R14
	BNE  qpel_highx0_load_next
	MOVHU (R11), R7
	B     qpel_highx0_l2
qpel_highx0_load_next:
	MOVHU 2(R11), R7
qpel_highx0_l2:
	ADDW R7, R12, R12
	ADDW $1, R12, R12
	LSRW $1, R12, R12
qpel_highx0_pred_done:
	CBZW  R17, qpel_highx0_store
	MOVHU (R10), R7
	ADDW  R7, R12, R12
	ADDW  $1, R12, R12
	LSRW  $1, R12, R12
qpel_highx0_store:
	MOVH  R12, (R10)
	ADD   $2, R10, R10
	ADD   $2, R11, R11
	SUBW  $1, R9, R9
	CBNZW R9, qpel_highx0_col
	ADD   R2, R0, R0
	ADD   R3, R1, R1
	SUBW  $1, R4, R4
	CBNZW R4, qpel_highx0_row
	RET

// func h264QpelMCHigh0YASM(dst *uint8, src *uint8, dstStride int, srcStride int, size int32, my int32, max int32, avg int32)
TEXT ·h264QpelMCHigh0YASM(SB), NOSPLIT, $0-48
	MOVD dst+0(FP), R0
	MOVD src+8(FP), R1
	MOVD dstStride+16(FP), R2
	MOVD srcStride+24(FP), R3
	MOVW size+32(FP), R4
	MOVW my+36(FP), R14
	MOVW max+40(FP), R15
	MOVW avg+44(FP), R17
qpel_high0y_row:
	MOVD R0, R10
	MOVD R1, R11
	MOVW size+32(FP), R9
qpel_high0y_col:
	MOVHU (R11), R5
	ADD   R3, R11, R16
	MOVHU (R16), R6
	ADDW  R6, R5, R5
	LSLW  $4, R5, R12
	ADDW  R5<<2, R12, R12
	SUB   R3, R11, R16
	MOVHU (R16), R5
	ADD   R3, R11, R16
	ADD   R3, R16, R16
	MOVHU (R16), R6
	ADDW  R6, R5, R5
	ADDW  R5<<2, R5, R5
	SUBW  R5, R12, R12
	SUB   R3, R11, R16
	SUB   R3, R16, R16
	MOVHU (R16), R5
	ADD   R3, R11, R16
	ADD   R3, R16, R16
	ADD   R3, R16, R16
	MOVHU (R16), R6
	ADDW  R6, R5, R5
	ADDW  R5, R12, R12
	ADDW  $16, R12, R12
	ASRW  $5, R12, R12
	CMPW  $0, R12
	BGE   qpel_high0y_nonnegative
	MOVW  ZR, R12
	B     qpel_high0y_clip_done
qpel_high0y_nonnegative:
	CMPW R15, R12
	BLE  qpel_high0y_clip_done
	MOVW R15, R12
qpel_high0y_clip_done:
	CMPW $2, R14
	BEQ  qpel_high0y_pred_done
	CMPW $1, R14
	BNE  qpel_high0y_load_next
	MOVHU (R11), R7
	B     qpel_high0y_l2
qpel_high0y_load_next:
	ADD   R3, R11, R16
	MOVHU (R16), R7
qpel_high0y_l2:
	ADDW R7, R12, R12
	ADDW $1, R12, R12
	LSRW $1, R12, R12
qpel_high0y_pred_done:
	CBZW  R17, qpel_high0y_store
	MOVHU (R10), R7
	ADDW  R7, R12, R12
	ADDW  $1, R12, R12
	LSRW  $1, R12, R12
qpel_high0y_store:
	MOVH  R12, (R10)
	ADD   $2, R10, R10
	ADD   $2, R11, R11
	SUBW  $1, R9, R9
	CBNZW R9, qpel_high0y_col
	ADD   R2, R0, R0
	ADD   R3, R1, R1
	SUBW  $1, R4, R4
	CBNZW R4, qpel_high0y_row
	RET

// func h264QpelMCHigh22ASM(dst *uint8, src *uint8, dstStride int, srcStride int, size int32, max int32, avg int32)
TEXT ·h264QpelMCHigh22ASM(SB), NOSPLIT, $32-48
	MOVD dst+0(FP), R0
	MOVD src+8(FP), R1
	MOVD dstStride+16(FP), R2
	MOVD srcStride+24(FP), R3
	MOVW size+32(FP), R4
	MOVW max+36(FP), R15
	MOVW avg+40(FP), R17
qpel_high22_row:
	MOVD R0, R10
	MOVD R1, R11
	MOVW size+32(FP), R9
qpel_high22_col:
	SUB   R3, R11, R13
	SUB   R3, R13, R13
	MOVHU (R13), R5
	MOVHU 2(R13), R6
	ADDW  R6, R5, R5
	LSLW  $4, R5, R12
	ADDW  R5<<2, R12, R12
	MOVHU -2(R13), R5
	MOVHU 4(R13), R6
	ADDW  R6, R5, R5
	ADDW  R5<<2, R5, R5
	SUBW  R5, R12, R12
	MOVHU -4(R13), R5
	MOVHU 6(R13), R6
	ADDW  R6, R5, R5
	ADDW  R5, R12, R12
	MOVW  R12, qpel_high22_tmp0-32(SP)

	SUB   R3, R11, R13
	MOVHU (R13), R5
	MOVHU 2(R13), R6
	ADDW  R6, R5, R5
	LSLW  $4, R5, R12
	ADDW  R5<<2, R12, R12
	MOVHU -2(R13), R5
	MOVHU 4(R13), R6
	ADDW  R6, R5, R5
	ADDW  R5<<2, R5, R5
	SUBW  R5, R12, R12
	MOVHU -4(R13), R5
	MOVHU 6(R13), R6
	ADDW  R6, R5, R5
	ADDW  R5, R12, R12
	MOVW  R12, qpel_high22_tmp1-28(SP)

	MOVD  R11, R13
	MOVHU (R13), R5
	MOVHU 2(R13), R6
	ADDW  R6, R5, R5
	LSLW  $4, R5, R12
	ADDW  R5<<2, R12, R12
	MOVHU -2(R13), R5
	MOVHU 4(R13), R6
	ADDW  R6, R5, R5
	ADDW  R5<<2, R5, R5
	SUBW  R5, R12, R12
	MOVHU -4(R13), R5
	MOVHU 6(R13), R6
	ADDW  R6, R5, R5
	ADDW  R5, R12, R12
	MOVW  R12, qpel_high22_tmp2-24(SP)

	ADD   R3, R11, R13
	MOVHU (R13), R5
	MOVHU 2(R13), R6
	ADDW  R6, R5, R5
	LSLW  $4, R5, R12
	ADDW  R5<<2, R12, R12
	MOVHU -2(R13), R5
	MOVHU 4(R13), R6
	ADDW  R6, R5, R5
	ADDW  R5<<2, R5, R5
	SUBW  R5, R12, R12
	MOVHU -4(R13), R5
	MOVHU 6(R13), R6
	ADDW  R6, R5, R5
	ADDW  R5, R12, R12
	MOVW  R12, qpel_high22_tmp3-20(SP)

	ADD   R3, R11, R13
	ADD   R3, R13, R13
	MOVHU (R13), R5
	MOVHU 2(R13), R6
	ADDW  R6, R5, R5
	LSLW  $4, R5, R12
	ADDW  R5<<2, R12, R12
	MOVHU -2(R13), R5
	MOVHU 4(R13), R6
	ADDW  R6, R5, R5
	ADDW  R5<<2, R5, R5
	SUBW  R5, R12, R12
	MOVHU -4(R13), R5
	MOVHU 6(R13), R6
	ADDW  R6, R5, R5
	ADDW  R5, R12, R12
	MOVW  R12, qpel_high22_tmp4-16(SP)

	ADD   R3, R11, R13
	ADD   R3, R13, R13
	ADD   R3, R13, R13
	MOVHU (R13), R5
	MOVHU 2(R13), R6
	ADDW  R6, R5, R5
	LSLW  $4, R5, R12
	ADDW  R5<<2, R12, R12
	MOVHU -2(R13), R5
	MOVHU 4(R13), R6
	ADDW  R6, R5, R5
	ADDW  R5<<2, R5, R5
	SUBW  R5, R12, R12
	MOVHU -4(R13), R5
	MOVHU 6(R13), R6
	ADDW  R6, R5, R5
	ADDW  R5, R12, R12
	MOVW  R12, qpel_high22_tmp5-12(SP)

	MOVW  qpel_high22_tmp2-24(SP), R5
	MOVW  qpel_high22_tmp3-20(SP), R6
	ADDW  R6, R5, R5
	LSLW  $4, R5, R12
	ADDW  R5<<2, R12, R12
	MOVW  qpel_high22_tmp1-28(SP), R5
	MOVW  qpel_high22_tmp4-16(SP), R6
	ADDW  R6, R5, R5
	ADDW  R5<<2, R5, R5
	SUBW  R5, R12, R12
	MOVW  qpel_high22_tmp0-32(SP), R5
	MOVW  qpel_high22_tmp5-12(SP), R6
	ADDW  R6, R5, R5
	ADDW  R5, R12, R12
	ADDW  $512, R12, R12
	ASRW  $10, R12, R12
	CMPW  $0, R12
	BGE   qpel_high22_nonnegative
	MOVW  ZR, R12
	B     qpel_high22_clip_done
qpel_high22_nonnegative:
	CMPW R15, R12
	BLE  qpel_high22_clip_done
	MOVW R15, R12
qpel_high22_clip_done:
	CBZW  R17, qpel_high22_store
	MOVHU (R10), R7
	ADDW  R7, R12, R12
	ADDW  $1, R12, R12
	LSRW  $1, R12, R12
qpel_high22_store:
	MOVH  R12, (R10)
	ADD   $2, R10, R10
	ADD   $2, R11, R11
	SUBW  $1, R9, R9
	CBNZW R9, qpel_high22_col
	ADD   R2, R0, R0
	ADD   R3, R1, R1
	SUBW  $1, R4, R4
	CBNZW R4, qpel_high22_row
	RET

// func h264QpelMCHighHVXYASM(dst *uint8, src *uint8, dstStride int, srcStride int, size int32, mx int32, my int32, max int32, avg int32)
TEXT ·h264QpelMCHighHVXYASM(SB), NOSPLIT, $16-56
	MOVD dst+0(FP), R0
	MOVD src+8(FP), R1
	MOVD dstStride+16(FP), R2
	MOVD srcStride+24(FP), R3
	MOVW size+32(FP), R4
	MOVW max+44(FP), R15
	MOVW avg+48(FP), R17
qpel_highhvxy_row:
	MOVD R0, R10
	MOVD R1, R11
	MOVW size+32(FP), R9
qpel_highhvxy_col:
	MOVD  R11, R13
	MOVW  my+40(FP), R14
	CMPW  $3, R14
	BNE   qpel_highhvxy_hptr_ready
	ADD   R3, R13, R13
qpel_highhvxy_hptr_ready:
	MOVHU (R13), R5
	MOVHU 2(R13), R6
	ADDW  R6, R5, R5
	LSLW  $4, R5, R12
	ADDW  R5<<2, R12, R12
	MOVHU -2(R13), R5
	MOVHU 4(R13), R6
	ADDW  R6, R5, R5
	ADDW  R5<<2, R5, R5
	SUBW  R5, R12, R12
	MOVHU -4(R13), R5
	MOVHU 6(R13), R6
	ADDW  R6, R5, R5
	ADDW  R5, R12, R12
	ADDW  $16, R12, R12
	ASRW  $5, R12, R12
	CMPW  $0, R12
	BGE   qpel_highhvxy_h_nonnegative
	MOVW  ZR, R12
	B     qpel_highhvxy_h_done
qpel_highhvxy_h_nonnegative:
	CMPW R15, R12
	BLE  qpel_highhvxy_h_done
	MOVW R15, R12
qpel_highhvxy_h_done:
	MOVW  R12, qpel_highhvxy_htmp-16(SP)
	MOVD  R11, R13
	MOVW  mx+36(FP), R14
	CMPW  $3, R14
	BNE   qpel_highhvxy_vptr_ready
	ADD   $2, R13, R13
qpel_highhvxy_vptr_ready:
	MOVHU (R13), R5
	ADD   R3, R13, R16
	MOVHU (R16), R6
	ADDW  R6, R5, R5
	LSLW  $4, R5, R12
	ADDW  R5<<2, R12, R12
	SUB   R3, R13, R16
	MOVHU (R16), R5
	ADD   R3, R13, R16
	ADD   R3, R16, R16
	MOVHU (R16), R6
	ADDW  R6, R5, R5
	ADDW  R5<<2, R5, R5
	SUBW  R5, R12, R12
	SUB   R3, R13, R16
	SUB   R3, R16, R16
	MOVHU (R16), R5
	ADD   R3, R13, R16
	ADD   R3, R16, R16
	ADD   R3, R16, R16
	MOVHU (R16), R6
	ADDW  R6, R5, R5
	ADDW  R5, R12, R12
	ADDW  $16, R12, R12
	ASRW  $5, R12, R12
	CMPW  $0, R12
	BGE   qpel_highhvxy_v_nonnegative
	MOVW  ZR, R12
	B     qpel_highhvxy_v_done
qpel_highhvxy_v_nonnegative:
	CMPW R15, R12
	BLE  qpel_highhvxy_v_done
	MOVW R15, R12
qpel_highhvxy_v_done:
	MOVW  qpel_highhvxy_htmp-16(SP), R5
	ADDW  R5, R12, R12
	ADDW  $1, R12, R12
	LSRW  $1, R12, R12
	CBZW  R17, qpel_highhvxy_store
	MOVHU (R10), R7
	ADDW  R7, R12, R12
	ADDW  $1, R12, R12
	LSRW  $1, R12, R12
qpel_highhvxy_store:
	MOVH  R12, (R10)
	ADD   $2, R10, R10
	ADD   $2, R11, R11
	SUBW  $1, R9, R9
	CBNZW R9, qpel_highhvxy_col
	ADD   R2, R0, R0
	ADD   R3, R1, R1
	SUBW  $1, R4, R4
	CBNZW R4, qpel_highhvxy_row
	RET

// func h264QpelMCHighHVBlendASM(dst *uint8, src *uint8, dstStride int, srcStride int, size int32, mx int32, my int32, max int32, avg int32)
TEXT ·h264QpelMCHighHVBlendASM(SB), NOSPLIT, $32-56
	MOVD dst+0(FP), R0
	MOVD src+8(FP), R1
	MOVD dstStride+16(FP), R2
	MOVD srcStride+24(FP), R3
	MOVW size+32(FP), R4
	MOVW max+44(FP), R15
	MOVW avg+48(FP), R17
qpel_highhvblend_row:
	MOVD R0, R10
	MOVD R1, R11
	MOVW size+32(FP), R9
qpel_highhvblend_col:
	MOVW mx+36(FP), R14
	CMPW $2, R14
	BNE  qpel_highhvblend_vbase
	MOVD R11, R13
	MOVW my+40(FP), R14
	CMPW $3, R14
	BNE  qpel_highhvblend_hbase_ptr_ready
	ADD  R3, R13, R13
qpel_highhvblend_hbase_ptr_ready:
	MOVHU (R13), R5
	MOVHU 2(R13), R6
	ADDW  R6, R5, R5
	LSLW  $4, R5, R12
	ADDW  R5<<2, R12, R12
	MOVHU -2(R13), R5
	MOVHU 4(R13), R6
	ADDW  R6, R5, R5
	ADDW  R5<<2, R5, R5
	SUBW  R5, R12, R12
	MOVHU -4(R13), R5
	MOVHU 6(R13), R6
	ADDW  R6, R5, R5
	ADDW  R5, R12, R12
	ADDW  $16, R12, R12
	ASRW  $5, R12, R12
	CMPW  $0, R12
	BGE   qpel_highhvblend_hbase_nonnegative
	MOVW  ZR, R12
	B     qpel_highhvblend_base_done
qpel_highhvblend_hbase_nonnegative:
	CMPW R15, R12
	BLE  qpel_highhvblend_base_done
	MOVW R15, R12
	B    qpel_highhvblend_base_done
qpel_highhvblend_vbase:
	MOVD R11, R13
	MOVW mx+36(FP), R14
	CMPW $3, R14
	BNE  qpel_highhvblend_vbase_ptr_ready
	ADD  $2, R13, R13
qpel_highhvblend_vbase_ptr_ready:
	MOVHU (R13), R5
	ADD   R3, R13, R16
	MOVHU (R16), R6
	ADDW  R6, R5, R5
	LSLW  $4, R5, R12
	ADDW  R5<<2, R12, R12
	SUB   R3, R13, R16
	MOVHU (R16), R5
	ADD   R3, R13, R16
	ADD   R3, R16, R16
	MOVHU (R16), R6
	ADDW  R6, R5, R5
	ADDW  R5<<2, R5, R5
	SUBW  R5, R12, R12
	SUB   R3, R13, R16
	SUB   R3, R16, R16
	MOVHU (R16), R5
	ADD   R3, R13, R16
	ADD   R3, R16, R16
	ADD   R3, R16, R16
	MOVHU (R16), R6
	ADDW  R6, R5, R5
	ADDW  R5, R12, R12
	ADDW  $16, R12, R12
	ASRW  $5, R12, R12
	CMPW  $0, R12
	BGE   qpel_highhvblend_vbase_nonnegative
	MOVW  ZR, R12
	B     qpel_highhvblend_base_done
qpel_highhvblend_vbase_nonnegative:
	CMPW R15, R12
	BLE  qpel_highhvblend_base_done
	MOVW R15, R12
qpel_highhvblend_base_done:
	MOVW R12, qpel_highhvblend_base-8(SP)

	SUB   R3, R11, R13
	SUB   R3, R13, R13
	MOVHU (R13), R5
	MOVHU 2(R13), R6
	ADDW  R6, R5, R5
	LSLW  $4, R5, R12
	ADDW  R5<<2, R12, R12
	MOVHU -2(R13), R5
	MOVHU 4(R13), R6
	ADDW  R6, R5, R5
	ADDW  R5<<2, R5, R5
	SUBW  R5, R12, R12
	MOVHU -4(R13), R5
	MOVHU 6(R13), R6
	ADDW  R6, R5, R5
	ADDW  R5, R12, R12
	MOVW  R12, qpel_highhvblend_tmp0-32(SP)

	SUB   R3, R11, R13
	MOVHU (R13), R5
	MOVHU 2(R13), R6
	ADDW  R6, R5, R5
	LSLW  $4, R5, R12
	ADDW  R5<<2, R12, R12
	MOVHU -2(R13), R5
	MOVHU 4(R13), R6
	ADDW  R6, R5, R5
	ADDW  R5<<2, R5, R5
	SUBW  R5, R12, R12
	MOVHU -4(R13), R5
	MOVHU 6(R13), R6
	ADDW  R6, R5, R5
	ADDW  R5, R12, R12
	MOVW  R12, qpel_highhvblend_tmp1-28(SP)

	MOVD  R11, R13
	MOVHU (R13), R5
	MOVHU 2(R13), R6
	ADDW  R6, R5, R5
	LSLW  $4, R5, R12
	ADDW  R5<<2, R12, R12
	MOVHU -2(R13), R5
	MOVHU 4(R13), R6
	ADDW  R6, R5, R5
	ADDW  R5<<2, R5, R5
	SUBW  R5, R12, R12
	MOVHU -4(R13), R5
	MOVHU 6(R13), R6
	ADDW  R6, R5, R5
	ADDW  R5, R12, R12
	MOVW  R12, qpel_highhvblend_tmp2-24(SP)

	ADD   R3, R11, R13
	MOVHU (R13), R5
	MOVHU 2(R13), R6
	ADDW  R6, R5, R5
	LSLW  $4, R5, R12
	ADDW  R5<<2, R12, R12
	MOVHU -2(R13), R5
	MOVHU 4(R13), R6
	ADDW  R6, R5, R5
	ADDW  R5<<2, R5, R5
	SUBW  R5, R12, R12
	MOVHU -4(R13), R5
	MOVHU 6(R13), R6
	ADDW  R6, R5, R5
	ADDW  R5, R12, R12
	MOVW  R12, qpel_highhvblend_tmp3-20(SP)

	ADD   R3, R11, R13
	ADD   R3, R13, R13
	MOVHU (R13), R5
	MOVHU 2(R13), R6
	ADDW  R6, R5, R5
	LSLW  $4, R5, R12
	ADDW  R5<<2, R12, R12
	MOVHU -2(R13), R5
	MOVHU 4(R13), R6
	ADDW  R6, R5, R5
	ADDW  R5<<2, R5, R5
	SUBW  R5, R12, R12
	MOVHU -4(R13), R5
	MOVHU 6(R13), R6
	ADDW  R6, R5, R5
	ADDW  R5, R12, R12
	MOVW  R12, qpel_highhvblend_tmp4-16(SP)

	ADD   R3, R11, R13
	ADD   R3, R13, R13
	ADD   R3, R13, R13
	MOVHU (R13), R5
	MOVHU 2(R13), R6
	ADDW  R6, R5, R5
	LSLW  $4, R5, R12
	ADDW  R5<<2, R12, R12
	MOVHU -2(R13), R5
	MOVHU 4(R13), R6
	ADDW  R6, R5, R5
	ADDW  R5<<2, R5, R5
	SUBW  R5, R12, R12
	MOVHU -4(R13), R5
	MOVHU 6(R13), R6
	ADDW  R6, R5, R5
	ADDW  R5, R12, R12
	MOVW  R12, qpel_highhvblend_tmp5-12(SP)

	MOVW  qpel_highhvblend_tmp2-24(SP), R5
	MOVW  qpel_highhvblend_tmp3-20(SP), R6
	ADDW  R6, R5, R5
	LSLW  $4, R5, R12
	ADDW  R5<<2, R12, R12
	MOVW  qpel_highhvblend_tmp1-28(SP), R5
	MOVW  qpel_highhvblend_tmp4-16(SP), R6
	ADDW  R6, R5, R5
	ADDW  R5<<2, R5, R5
	SUBW  R5, R12, R12
	MOVW  qpel_highhvblend_tmp0-32(SP), R5
	MOVW  qpel_highhvblend_tmp5-12(SP), R6
	ADDW  R6, R5, R5
	ADDW  R5, R12, R12
	ADDW  $512, R12, R12
	ASRW  $10, R12, R12
	CMPW  $0, R12
	BGE   qpel_highhvblend_hv_nonnegative
	MOVW  ZR, R12
	B     qpel_highhvblend_hv_done
qpel_highhvblend_hv_nonnegative:
	CMPW R15, R12
	BLE  qpel_highhvblend_hv_done
	MOVW R15, R12
qpel_highhvblend_hv_done:
	MOVW qpel_highhvblend_base-8(SP), R5
	ADDW R5, R12, R12
	ADDW $1, R12, R12
	LSRW $1, R12, R12
	CBZW R17, qpel_highhvblend_store
	MOVHU (R10), R7
	ADDW  R7, R12, R12
	ADDW  $1, R12, R12
	LSRW  $1, R12, R12
qpel_highhvblend_store:
	MOVH  R12, (R10)
	ADD   $2, R10, R10
	ADD   $2, R11, R11
	SUBW  $1, R9, R9
	CBNZW R9, qpel_highhvblend_col
	ADD   R2, R0, R0
	ADD   R3, R1, R1
	SUBW  $1, R4, R4
	CBNZW R4, qpel_highhvblend_row
	RET
