// SPDX-License-Identifier: LGPL-2.1-or-later
//go:build !purego && arm64

#include "textflag.h"


// func h264QpelMCPutHVBlendASM(dst *uint8, src *uint8, dstStride int, srcStride int, size int32, mx int32, my int32)
TEXT ·h264QpelMCPutHVBlendASM(SB), NOSPLIT, $32-48
	MOVW size+32(FP), R6
	CMPW $8, R6
	BLT  qpel_puthvblend_scalar
	MOVD dst+0(FP), R0
	MOVD src+8(FP), R1
	MOVD dstStride+16(FP), R2
	MOVD srcStride+24(FP), R3
	MOVW mx+36(FP), R4
	MOVW $0, R5
	MOVW my+40(FP), R7
	BL   ·h264QpelHVBlendNEONInternal(SB)
	RET
qpel_puthvblend_scalar:
	MOVD dst+0(FP), R0
	MOVD src+8(FP), R1
	MOVD dstStride+16(FP), R2
	MOVD srcStride+24(FP), R3
	MOVW size+32(FP), R4
	MOVW $0, R5
	MOVW R5, qpel_puthvblend_avgflag-4(SP)
qpel_puthvblend_row:
	MOVD R0, R10
	MOVD R1, R11
	MOVW size+32(FP), R9
qpel_puthvblend_col:
	MOVW mx+36(FP), R14
	CMPW $2, R14
	BNE  qpel_puthvblend_vbase
	MOVD R11, R13
	MOVW my+40(FP), R14
	CMPW $3, R14
	BNE  qpel_puthvblend_hbase_ptr_ready
	ADD  R3, R13, R13
qpel_puthvblend_hbase_ptr_ready:
	MOVBU (R13), R5
	MOVBU 1(R13), R6
	ADDW  R6, R5, R5
	LSLW  $4, R5, R12
	ADDW  R5<<2, R12, R12
	MOVBU -1(R13), R5
	MOVBU 2(R13), R6
	ADDW  R6, R5, R5
	ADDW  R5<<2, R5, R5
	SUBW  R5, R12, R12
	MOVBU -2(R13), R5
	MOVBU 3(R13), R6
	ADDW  R6, R5, R5
	ADDW  R5, R12, R12
	ADDW  $16, R12, R12
	ASRW  $5, R12, R12
	CMPW  $0, R12
	BGE   qpel_puthvblend_hbase_nonnegative
	MOVW  ZR, R12
	B     qpel_puthvblend_base_done
qpel_puthvblend_hbase_nonnegative:
	CMPW $255, R12
	BLE  qpel_puthvblend_base_done
	MOVW $255, R12
	B    qpel_puthvblend_base_done
qpel_puthvblend_vbase:
	MOVD R11, R13
	MOVW mx+36(FP), R14
	CMPW $3, R14
	BNE  qpel_puthvblend_vbase_ptr_ready
	ADD  $1, R13, R13
qpel_puthvblend_vbase_ptr_ready:
	MOVBU (R13), R5
	ADD   R3, R13, R14
	MOVBU (R14), R6
	ADDW  R6, R5, R5
	LSLW  $4, R5, R12
	ADDW  R5<<2, R12, R12
	SUB   R3, R13, R14
	MOVBU (R14), R5
	ADD   R3, R13, R14
	ADD   R3, R14, R14
	MOVBU (R14), R6
	ADDW  R6, R5, R5
	ADDW  R5<<2, R5, R5
	SUBW  R5, R12, R12
	SUB   R3, R13, R14
	SUB   R3, R14, R14
	MOVBU (R14), R5
	ADD   R3, R13, R14
	ADD   R3, R14, R14
	ADD   R3, R14, R14
	MOVBU (R14), R6
	ADDW  R6, R5, R5
	ADDW  R5, R12, R12
	ADDW  $16, R12, R12
	ASRW  $5, R12, R12
	CMPW  $0, R12
	BGE   qpel_puthvblend_vbase_nonnegative
	MOVW  ZR, R12
	B     qpel_puthvblend_base_done
qpel_puthvblend_vbase_nonnegative:
	CMPW $255, R12
	BLE  qpel_puthvblend_base_done
	MOVW $255, R12
qpel_puthvblend_base_done:
	MOVW R12, qpel_puthvblend_base-8(SP)
	SUB  R3, R11, R13
	SUB  R3, R13, R13
	MOVBU (R13), R5
	MOVBU 1(R13), R6
	ADDW  R6, R5, R5
	LSLW  $4, R5, R12
	ADDW  R5<<2, R12, R12
	MOVBU -1(R13), R5
	MOVBU 2(R13), R6
	ADDW  R6, R5, R5
	ADDW  R5<<2, R5, R5
	SUBW  R5, R12, R12
	MOVBU -2(R13), R5
	MOVBU 3(R13), R6
	ADDW  R6, R5, R5
	ADDW  R5, R12, R12
	MOVW  R12, qpel_puthvblend_tmp0-32(SP)
	SUB   R3, R11, R13
	MOVBU (R13), R5
	MOVBU 1(R13), R6
	ADDW  R6, R5, R5
	LSLW  $4, R5, R12
	ADDW  R5<<2, R12, R12
	MOVBU -1(R13), R5
	MOVBU 2(R13), R6
	ADDW  R6, R5, R5
	ADDW  R5<<2, R5, R5
	SUBW  R5, R12, R12
	MOVBU -2(R13), R5
	MOVBU 3(R13), R6
	ADDW  R6, R5, R5
	ADDW  R5, R12, R12
	MOVW  R12, qpel_puthvblend_tmp1-28(SP)
	MOVD  R11, R13
	MOVBU (R13), R5
	MOVBU 1(R13), R6
	ADDW  R6, R5, R5
	LSLW  $4, R5, R12
	ADDW  R5<<2, R12, R12
	MOVBU -1(R13), R5
	MOVBU 2(R13), R6
	ADDW  R6, R5, R5
	ADDW  R5<<2, R5, R5
	SUBW  R5, R12, R12
	MOVBU -2(R13), R5
	MOVBU 3(R13), R6
	ADDW  R6, R5, R5
	ADDW  R5, R12, R12
	MOVW  R12, qpel_puthvblend_tmp2-24(SP)
	ADD   R3, R11, R13
	MOVBU (R13), R5
	MOVBU 1(R13), R6
	ADDW  R6, R5, R5
	LSLW  $4, R5, R12
	ADDW  R5<<2, R12, R12
	MOVBU -1(R13), R5
	MOVBU 2(R13), R6
	ADDW  R6, R5, R5
	ADDW  R5<<2, R5, R5
	SUBW  R5, R12, R12
	MOVBU -2(R13), R5
	MOVBU 3(R13), R6
	ADDW  R6, R5, R5
	ADDW  R5, R12, R12
	MOVW  R12, qpel_puthvblend_tmp3-20(SP)
	ADD   R3, R11, R13
	ADD   R3, R13, R13
	MOVBU (R13), R5
	MOVBU 1(R13), R6
	ADDW  R6, R5, R5
	LSLW  $4, R5, R12
	ADDW  R5<<2, R12, R12
	MOVBU -1(R13), R5
	MOVBU 2(R13), R6
	ADDW  R6, R5, R5
	ADDW  R5<<2, R5, R5
	SUBW  R5, R12, R12
	MOVBU -2(R13), R5
	MOVBU 3(R13), R6
	ADDW  R6, R5, R5
	ADDW  R5, R12, R12
	MOVW  R12, qpel_puthvblend_tmp4-16(SP)
	ADD   R3, R11, R13
	ADD   R3, R13, R13
	ADD   R3, R13, R13
	MOVBU (R13), R5
	MOVBU 1(R13), R6
	ADDW  R6, R5, R5
	LSLW  $4, R5, R12
	ADDW  R5<<2, R12, R12
	MOVBU -1(R13), R5
	MOVBU 2(R13), R6
	ADDW  R6, R5, R5
	ADDW  R5<<2, R5, R5
	SUBW  R5, R12, R12
	MOVBU -2(R13), R5
	MOVBU 3(R13), R6
	ADDW  R6, R5, R5
	ADDW  R5, R12, R12
	MOVW  R12, qpel_puthvblend_tmp5-12(SP)
	MOVW  qpel_puthvblend_tmp2-24(SP), R5
	MOVW  qpel_puthvblend_tmp3-20(SP), R6
	ADDW  R6, R5, R5
	LSLW  $4, R5, R12
	ADDW  R5<<2, R12, R12
	MOVW  qpel_puthvblend_tmp1-28(SP), R5
	MOVW  qpel_puthvblend_tmp4-16(SP), R6
	ADDW  R6, R5, R5
	ADDW  R5<<2, R5, R5
	SUBW  R5, R12, R12
	MOVW  qpel_puthvblend_tmp0-32(SP), R5
	MOVW  qpel_puthvblend_tmp5-12(SP), R6
	ADDW  R6, R5, R5
	ADDW  R5, R12, R12
	ADDW  $512, R12, R12
	ASRW  $10, R12, R12
	CMPW  $0, R12
	BGE   qpel_puthvblend_hv_nonnegative
	MOVW  ZR, R12
	B     qpel_puthvblend_hv_done
qpel_puthvblend_hv_nonnegative:
	CMPW $255, R12
	BLE  qpel_puthvblend_hv_done
	MOVW $255, R12
qpel_puthvblend_hv_done:
	MOVW qpel_puthvblend_base-8(SP), R5
	ADDW R5, R12, R12
	ADDW $1, R12, R12
	LSRW $1, R12, R12
	MOVW qpel_puthvblend_avgflag-4(SP), R5
	CBZW R5, qpel_puthvblend_store
	MOVBU (R10), R7
	ADDW  R7, R12, R12
	ADDW  $1, R12, R12
	LSRW  $1, R12, R12
qpel_puthvblend_store:
	MOVB  R12, (R10)
	ADD   $1, R10, R10
	ADD   $1, R11, R11
	SUBW  $1, R9, R9
	CBNZW R9, qpel_puthvblend_col
	ADD   R2, R0, R0
	ADD   R3, R1, R1
	SUBW  $1, R4, R4
	CBNZW R4, qpel_puthvblend_row
	RET

// func h264QpelMCAvgHVBlendASM(dst *uint8, src *uint8, dstStride int, srcStride int, size int32, mx int32, my int32)
TEXT ·h264QpelMCAvgHVBlendASM(SB), NOSPLIT, $32-48
	MOVW size+32(FP), R6
	CMPW $8, R6
	BLT  qpel_avghvblend_scalar
	MOVD dst+0(FP), R0
	MOVD src+8(FP), R1
	MOVD dstStride+16(FP), R2
	MOVD srcStride+24(FP), R3
	MOVW mx+36(FP), R4
	MOVW $1, R5
	MOVW my+40(FP), R7
	BL   ·h264QpelHVBlendNEONInternal(SB)
	RET
qpel_avghvblend_scalar:
	MOVD dst+0(FP), R0
	MOVD src+8(FP), R1
	MOVD dstStride+16(FP), R2
	MOVD srcStride+24(FP), R3
	MOVW size+32(FP), R4
	MOVW $1, R5
	MOVW R5, qpel_avghvblend_avgflag-4(SP)
qpel_avghvblend_row:
	MOVD R0, R10
	MOVD R1, R11
	MOVW size+32(FP), R9
qpel_avghvblend_col:
	MOVW mx+36(FP), R14
	CMPW $2, R14
	BNE  qpel_avghvblend_vbase
	MOVD R11, R13
	MOVW my+40(FP), R14
	CMPW $3, R14
	BNE  qpel_avghvblend_hbase_ptr_ready
	ADD  R3, R13, R13
qpel_avghvblend_hbase_ptr_ready:
	MOVBU (R13), R5
	MOVBU 1(R13), R6
	ADDW  R6, R5, R5
	LSLW  $4, R5, R12
	ADDW  R5<<2, R12, R12
	MOVBU -1(R13), R5
	MOVBU 2(R13), R6
	ADDW  R6, R5, R5
	ADDW  R5<<2, R5, R5
	SUBW  R5, R12, R12
	MOVBU -2(R13), R5
	MOVBU 3(R13), R6
	ADDW  R6, R5, R5
	ADDW  R5, R12, R12
	ADDW  $16, R12, R12
	ASRW  $5, R12, R12
	CMPW  $0, R12
	BGE   qpel_avghvblend_hbase_nonnegative
	MOVW  ZR, R12
	B     qpel_avghvblend_base_done
qpel_avghvblend_hbase_nonnegative:
	CMPW $255, R12
	BLE  qpel_avghvblend_base_done
	MOVW $255, R12
	B    qpel_avghvblend_base_done
qpel_avghvblend_vbase:
	MOVD R11, R13
	MOVW mx+36(FP), R14
	CMPW $3, R14
	BNE  qpel_avghvblend_vbase_ptr_ready
	ADD  $1, R13, R13
qpel_avghvblend_vbase_ptr_ready:
	MOVBU (R13), R5
	ADD   R3, R13, R14
	MOVBU (R14), R6
	ADDW  R6, R5, R5
	LSLW  $4, R5, R12
	ADDW  R5<<2, R12, R12
	SUB   R3, R13, R14
	MOVBU (R14), R5
	ADD   R3, R13, R14
	ADD   R3, R14, R14
	MOVBU (R14), R6
	ADDW  R6, R5, R5
	ADDW  R5<<2, R5, R5
	SUBW  R5, R12, R12
	SUB   R3, R13, R14
	SUB   R3, R14, R14
	MOVBU (R14), R5
	ADD   R3, R13, R14
	ADD   R3, R14, R14
	ADD   R3, R14, R14
	MOVBU (R14), R6
	ADDW  R6, R5, R5
	ADDW  R5, R12, R12
	ADDW  $16, R12, R12
	ASRW  $5, R12, R12
	CMPW  $0, R12
	BGE   qpel_avghvblend_vbase_nonnegative
	MOVW  ZR, R12
	B     qpel_avghvblend_base_done
qpel_avghvblend_vbase_nonnegative:
	CMPW $255, R12
	BLE  qpel_avghvblend_base_done
	MOVW $255, R12
qpel_avghvblend_base_done:
	MOVW R12, qpel_avghvblend_base-8(SP)
	SUB  R3, R11, R13
	SUB  R3, R13, R13
	MOVBU (R13), R5
	MOVBU 1(R13), R6
	ADDW  R6, R5, R5
	LSLW  $4, R5, R12
	ADDW  R5<<2, R12, R12
	MOVBU -1(R13), R5
	MOVBU 2(R13), R6
	ADDW  R6, R5, R5
	ADDW  R5<<2, R5, R5
	SUBW  R5, R12, R12
	MOVBU -2(R13), R5
	MOVBU 3(R13), R6
	ADDW  R6, R5, R5
	ADDW  R5, R12, R12
	MOVW  R12, qpel_avghvblend_tmp0-32(SP)
	SUB   R3, R11, R13
	MOVBU (R13), R5
	MOVBU 1(R13), R6
	ADDW  R6, R5, R5
	LSLW  $4, R5, R12
	ADDW  R5<<2, R12, R12
	MOVBU -1(R13), R5
	MOVBU 2(R13), R6
	ADDW  R6, R5, R5
	ADDW  R5<<2, R5, R5
	SUBW  R5, R12, R12
	MOVBU -2(R13), R5
	MOVBU 3(R13), R6
	ADDW  R6, R5, R5
	ADDW  R5, R12, R12
	MOVW  R12, qpel_avghvblend_tmp1-28(SP)
	MOVD  R11, R13
	MOVBU (R13), R5
	MOVBU 1(R13), R6
	ADDW  R6, R5, R5
	LSLW  $4, R5, R12
	ADDW  R5<<2, R12, R12
	MOVBU -1(R13), R5
	MOVBU 2(R13), R6
	ADDW  R6, R5, R5
	ADDW  R5<<2, R5, R5
	SUBW  R5, R12, R12
	MOVBU -2(R13), R5
	MOVBU 3(R13), R6
	ADDW  R6, R5, R5
	ADDW  R5, R12, R12
	MOVW  R12, qpel_avghvblend_tmp2-24(SP)
	ADD   R3, R11, R13
	MOVBU (R13), R5
	MOVBU 1(R13), R6
	ADDW  R6, R5, R5
	LSLW  $4, R5, R12
	ADDW  R5<<2, R12, R12
	MOVBU -1(R13), R5
	MOVBU 2(R13), R6
	ADDW  R6, R5, R5
	ADDW  R5<<2, R5, R5
	SUBW  R5, R12, R12
	MOVBU -2(R13), R5
	MOVBU 3(R13), R6
	ADDW  R6, R5, R5
	ADDW  R5, R12, R12
	MOVW  R12, qpel_avghvblend_tmp3-20(SP)
	ADD   R3, R11, R13
	ADD   R3, R13, R13
	MOVBU (R13), R5
	MOVBU 1(R13), R6
	ADDW  R6, R5, R5
	LSLW  $4, R5, R12
	ADDW  R5<<2, R12, R12
	MOVBU -1(R13), R5
	MOVBU 2(R13), R6
	ADDW  R6, R5, R5
	ADDW  R5<<2, R5, R5
	SUBW  R5, R12, R12
	MOVBU -2(R13), R5
	MOVBU 3(R13), R6
	ADDW  R6, R5, R5
	ADDW  R5, R12, R12
	MOVW  R12, qpel_avghvblend_tmp4-16(SP)
	ADD   R3, R11, R13
	ADD   R3, R13, R13
	ADD   R3, R13, R13
	MOVBU (R13), R5
	MOVBU 1(R13), R6
	ADDW  R6, R5, R5
	LSLW  $4, R5, R12
	ADDW  R5<<2, R12, R12
	MOVBU -1(R13), R5
	MOVBU 2(R13), R6
	ADDW  R6, R5, R5
	ADDW  R5<<2, R5, R5
	SUBW  R5, R12, R12
	MOVBU -2(R13), R5
	MOVBU 3(R13), R6
	ADDW  R6, R5, R5
	ADDW  R5, R12, R12
	MOVW  R12, qpel_avghvblend_tmp5-12(SP)
	MOVW  qpel_avghvblend_tmp2-24(SP), R5
	MOVW  qpel_avghvblend_tmp3-20(SP), R6
	ADDW  R6, R5, R5
	LSLW  $4, R5, R12
	ADDW  R5<<2, R12, R12
	MOVW  qpel_avghvblend_tmp1-28(SP), R5
	MOVW  qpel_avghvblend_tmp4-16(SP), R6
	ADDW  R6, R5, R5
	ADDW  R5<<2, R5, R5
	SUBW  R5, R12, R12
	MOVW  qpel_avghvblend_tmp0-32(SP), R5
	MOVW  qpel_avghvblend_tmp5-12(SP), R6
	ADDW  R6, R5, R5
	ADDW  R5, R12, R12
	ADDW  $512, R12, R12
	ASRW  $10, R12, R12
	CMPW  $0, R12
	BGE   qpel_avghvblend_hv_nonnegative
	MOVW  ZR, R12
	B     qpel_avghvblend_hv_done
qpel_avghvblend_hv_nonnegative:
	CMPW $255, R12
	BLE  qpel_avghvblend_hv_done
	MOVW $255, R12
qpel_avghvblend_hv_done:
	MOVW qpel_avghvblend_base-8(SP), R5
	ADDW R5, R12, R12
	ADDW $1, R12, R12
	LSRW $1, R12, R12
	MOVW qpel_avghvblend_avgflag-4(SP), R5
	CBZW R5, qpel_avghvblend_store
	MOVBU (R10), R7
	ADDW  R7, R12, R12
	ADDW  $1, R12, R12
	LSRW  $1, R12, R12
qpel_avghvblend_store:
	MOVB  R12, (R10)
	ADD   $1, R10, R10
	ADD   $1, R11, R11
	SUBW  $1, R9, R9
	CBNZW R9, qpel_avghvblend_col
	ADD   R2, R0, R0
	ADD   R3, R1, R1
	SUBW  $1, R4, R4
	CBNZW R4, qpel_avghvblend_row
	RET
