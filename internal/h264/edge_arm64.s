// SPDX-License-Identifier: LGPL-2.1-or-later
//go:build !purego && arm64

#include "textflag.h"

// h264EmulatedEdgeMCRowsASM mirrors the copy/extend phase of pinned FFmpeg
// n8.0.1 ff_emulated_edge_mc_8. The Go wrapper has already clamped geometry
// and validated the source and destination spans. src points at the first
// valid source row and column; the output block width is the H.264-fixed 21
// (luma/4:4:4) or 9 (subsampled chroma) pixels.
//
// func h264EmulatedEdgeMCRowsASM(dst *uint8, dstStride int, src *uint8, srcStride int, blockW int, blockH int, startX int, startY int, endX int, endY int)
TEXT ·h264EmulatedEdgeMCRowsASM(SB), NOSPLIT|NOFRAME, $0-80
	MOVD dst+0(FP), R0
	MOVD dstStride+8(FP), R1
	MOVD src+16(FP), R2
	MOVD srcStride+24(FP), R3
	MOVD blockW+32(FP), R4
	MOVD blockH+40(FP), R5
	MOVD startX+48(FP), R6
	MOVD startY+56(FP), R7
	MOVD endX+64(FP), R8
	MOVD endY+72(FP), R9

	MUL  R7, R1, R12
	ADD  R0, R12, R12       // first valid destination row
	MOVD R12, R13           // retained for top-row replication
	SUB  R7, R9, R11        // valid source-row count
	SUB  R6, R8, R10        // central copy width
	CBNZ R6, edge_valid_row
	CMP  R4, R8
	BNE  edge_valid_row
	CMP  $21, R4
	BNE  edge_full_9_row

edge_full_21_row:
	MOVD  0(R2), R20
	MOVD  8(R2), R21
	MOVWU 16(R2), R22
	MOVBU 20(R2), R23
	MOVD  R20, 0(R12)
	MOVD  R21, 8(R12)
	MOVW  R22, 16(R12)
	MOVB  R23, 20(R12)
	ADD   R1, R12, R12
	ADD   R3, R2, R2
	SUB   $1, R11, R11
	CBNZ  R11, edge_full_21_row
	SUB   R1, R12, R17
	B     edge_top_setup

edge_full_9_row:
	MOVD  0(R2), R20
	MOVBU 8(R2), R21
	MOVD  R20, 0(R12)
	MOVB  R21, 8(R12)
	ADD   R1, R12, R12
	ADD   R3, R2, R2
	SUB   $1, R11, R11
	CBNZ  R11, edge_full_9_row
	SUB   R1, R12, R17
	B     edge_top_setup

edge_valid_row:
	ADD  R6, R12, R14
	MOVD R2, R15
	MOVD R10, R16
	CMP  $16, R16
	BLT  edge_valid_copy8
	MOVD 0(R15), R20
	MOVD 8(R15), R21
	MOVD R20, 0(R14)
	MOVD R21, 8(R14)
	ADD  $16, R15, R15
	ADD  $16, R14, R14
	SUB  $16, R16, R16

edge_valid_copy8:
	CMP  $8, R16
	BLT  edge_valid_copy4
	MOVD 0(R15), R20
	MOVD R20, 0(R14)
	ADD  $8, R15, R15
	ADD  $8, R14, R14
	SUB  $8, R16, R16

edge_valid_copy4:
	CMP  $4, R16
	BLT  edge_valid_copy2
	MOVWU 0(R15), R20
	MOVW  R20, 0(R14)
	ADD   $4, R15, R15
	ADD   $4, R14, R14
	SUB   $4, R16, R16

edge_valid_copy2:
	CMP  $2, R16
	BLT  edge_valid_copy1
	MOVHU 0(R15), R20
	MOVH  R20, 0(R14)
	ADD   $2, R15, R15
	ADD   $2, R14, R14
	SUB   $2, R16, R16

edge_valid_copy1:
	CBZ   R16, edge_valid_left
	MOVBU 0(R15), R20
	MOVB  R20, 0(R14)

edge_valid_left:
	CBZ  R6, edge_valid_right
	ADD  R6, R12, R14
	MOVBU 0(R14), R20
	MOVD R12, R15
	MOVD R6, R16
edge_valid_left_loop:
	MOVB R20, 0(R15)
	ADD  $1, R15, R15
	SUB  $1, R16, R16
	CBNZ R16, edge_valid_left_loop

edge_valid_right:
	SUB  R8, R4, R16
	CBZ  R16, edge_valid_done
	ADD  R8, R12, R15
	SUB  $1, R15, R14
	MOVBU 0(R14), R20
edge_valid_right_loop:
	MOVB R20, 0(R15)
	ADD  $1, R15, R15
	SUB  $1, R16, R16
	CBNZ R16, edge_valid_right_loop

edge_valid_done:
	ADD R1, R12, R12
	ADD R3, R2, R2
	SUB $1, R11, R11
	CBNZ R11, edge_valid_row
	SUB R1, R12, R17       // last completed valid row

	// Replicate the first completed row above the valid rectangle.
edge_top_setup:
	MOVD R7, R16
	CBZ  R16, edge_bottom_setup
	MOVD R0, R19
edge_top_loop:
	CMP  $21, R4
	BNE  edge_top_9
	MOVD 0(R13), R20
	MOVD 8(R13), R21
	MOVWU 16(R13), R22
	MOVBU 20(R13), R23
	MOVD R20, 0(R19)
	MOVD R21, 8(R19)
	MOVW R22, 16(R19)
	MOVB R23, 20(R19)
	B edge_top_next
edge_top_9:
	MOVD 0(R13), R20
	MOVBU 8(R13), R21
	MOVD R20, 0(R19)
	MOVB R21, 8(R19)
edge_top_next:
	ADD R1, R19, R19
	SUB $1, R16, R16
	CBNZ R16, edge_top_loop

edge_bottom_setup:
	SUB R9, R5, R16
	CBZ R16, edge_return
	MOVD R12, R19
edge_bottom_loop:
	CMP  $21, R4
	BNE  edge_bottom_9
	MOVD 0(R17), R20
	MOVD 8(R17), R21
	MOVWU 16(R17), R22
	MOVBU 20(R17), R23
	MOVD R20, 0(R19)
	MOVD R21, 8(R19)
	MOVW R22, 16(R19)
	MOVB R23, 20(R19)
	B edge_bottom_next
edge_bottom_9:
	MOVD 0(R17), R20
	MOVBU 8(R17), R21
	MOVD R20, 0(R19)
	MOVB R21, 8(R19)
edge_bottom_next:
	ADD R1, R19, R19
	SUB $1, R16, R16
	CBNZ R16, edge_bottom_loop

edge_return:
	RET
