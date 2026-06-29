// SPDX-License-Identifier: LGPL-2.1-or-later
//go:build !purego && amd64

#include "textflag.h"

// func h264QpelMC16Put00ASM(dst *uint8, src *uint8, dstStride int, srcStride int)
TEXT ·h264QpelMC16Put00ASM(SB), NOSPLIT, $0-32
	MOVQ dst+0(FP), DI
	MOVQ src+8(FP), SI
	MOVQ dstStride+16(FP), DX
	MOVQ srcStride+24(FP), CX
	MOVL $16, R8
qpel16_put00_loop:
	MOVQ (SI), AX
	MOVQ 8(SI), BX
	MOVQ AX, (DI)
	MOVQ BX, 8(DI)
	ADDQ DX, DI
	ADDQ CX, SI
	DECL R8
	JNZ  qpel16_put00_loop
	RET

// func h264QpelMC16Avg00ASM(dst *uint8, src *uint8, dstStride int, srcStride int)
TEXT ·h264QpelMC16Avg00ASM(SB), NOSPLIT, $0-32
	MOVQ dst+0(FP), DI
	MOVQ src+8(FP), SI
	MOVQ dstStride+16(FP), DX
	MOVQ srcStride+24(FP), CX
	MOVL $16, R8
	MOVQ $0xfefefefefefefefe, R9
qpel16_avg00_loop:
	MOVQ (SI), AX
	MOVQ (DI), BX
	MOVQ AX, R10
	ORQ  BX, R10
	XORQ BX, AX
	ANDQ R9, AX
	SHRQ $1, AX
	SUBQ AX, R10
	MOVQ R10, (DI)
	MOVQ 8(SI), AX
	MOVQ 8(DI), BX
	MOVQ AX, R10
	ORQ  BX, R10
	XORQ BX, AX
	ANDQ R9, AX
	SHRQ $1, AX
	SUBQ AX, R10
	MOVQ R10, 8(DI)
	ADDQ DX, DI
	ADDQ CX, SI
	DECL R8
	JNZ  qpel16_avg00_loop
	RET

// func h264QpelMC8Put00ASM(dst *uint8, src *uint8, dstStride int, srcStride int)
TEXT ·h264QpelMC8Put00ASM(SB), NOSPLIT, $0-32
	MOVQ dst+0(FP), DI
	MOVQ src+8(FP), SI
	MOVQ dstStride+16(FP), DX
	MOVQ srcStride+24(FP), CX
	MOVL $8, R8
qpel8_put00_loop:
	MOVQ (SI), AX
	MOVQ AX, (DI)
	ADDQ DX, DI
	ADDQ CX, SI
	DECL R8
	JNZ  qpel8_put00_loop
	RET

// func h264QpelMC8Avg00ASM(dst *uint8, src *uint8, dstStride int, srcStride int)
TEXT ·h264QpelMC8Avg00ASM(SB), NOSPLIT, $0-32
	MOVQ dst+0(FP), DI
	MOVQ src+8(FP), SI
	MOVQ dstStride+16(FP), DX
	MOVQ srcStride+24(FP), CX
	MOVL $8, R8
	MOVQ $0xfefefefefefefefe, R9
qpel8_avg00_loop:
	MOVQ (SI), AX
	MOVQ (DI), BX
	MOVQ AX, R10
	ORQ  BX, R10
	XORQ BX, AX
	ANDQ R9, AX
	SHRQ $1, AX
	SUBQ AX, R10
	MOVQ R10, (DI)
	ADDQ DX, DI
	ADDQ CX, SI
	DECL R8
	JNZ  qpel8_avg00_loop
	RET

// func h264QpelMC16Put10ASM(dst *uint8, src *uint8, dstStride int, srcStride int)
TEXT ·h264QpelMC16Put10ASM(SB), NOSPLIT, $0-32
	MOVQ dst+0(FP), DI
	MOVQ src+8(FP), SI
	MOVQ dstStride+16(FP), DX
	MOVQ srcStride+24(FP), CX
	MOVL $16, R8
qpel16_put10_row:
	MOVQ DI, R11
	MOVQ SI, R12
	MOVL $16, R9
qpel16_put10_col:
	XORL AX, AX
	MOVB (R12), AX
	XORL BX, BX
	MOVB 1(R12), BX
	ADDL BX, AX
	IMULL $20, AX
	MOVL AX, R10
	XORL AX, AX
	MOVB -1(R12), AX
	XORL BX, BX
	MOVB 2(R12), BX
	ADDL BX, AX
	LEAL (AX)(AX*4), AX
	SUBL AX, R10
	XORL AX, AX
	MOVB -2(R12), AX
	XORL BX, BX
	MOVB 3(R12), BX
	ADDL BX, AX
	ADDL AX, R10
	ADDL $16, R10
	SARL $5, R10
	CMPL R10, $0
	JGE  qpel16_put10_nonnegative
	XORL R10, R10
	JMP  qpel16_put10_clip_done
qpel16_put10_nonnegative:
	CMPL R10, $255
	JLE  qpel16_put10_clip_done
	MOVL $255, R10
qpel16_put10_clip_done:
	XORL AX, AX
	MOVB (R12), AX
	ADDL AX, R10
	ADDL $1, R10
	SHRL $1, R10
	MOVB R10, (R11)
	INCQ R11
	INCQ R12
	DECL R9
	JNZ  qpel16_put10_col
	ADDQ DX, DI
	ADDQ CX, SI
	DECL R8
	JNZ  qpel16_put10_row
	RET

// func h264QpelMC16Avg10ASM(dst *uint8, src *uint8, dstStride int, srcStride int)
TEXT ·h264QpelMC16Avg10ASM(SB), NOSPLIT, $0-32
	MOVQ dst+0(FP), DI
	MOVQ src+8(FP), SI
	MOVQ dstStride+16(FP), DX
	MOVQ srcStride+24(FP), CX
	MOVL $16, R8
qpel16_avg10_row:
	MOVQ DI, R11
	MOVQ SI, R12
	MOVL $16, R9
qpel16_avg10_col:
	XORL AX, AX
	MOVB (R12), AX
	XORL BX, BX
	MOVB 1(R12), BX
	ADDL BX, AX
	IMULL $20, AX
	MOVL AX, R10
	XORL AX, AX
	MOVB -1(R12), AX
	XORL BX, BX
	MOVB 2(R12), BX
	ADDL BX, AX
	LEAL (AX)(AX*4), AX
	SUBL AX, R10
	XORL AX, AX
	MOVB -2(R12), AX
	XORL BX, BX
	MOVB 3(R12), BX
	ADDL BX, AX
	ADDL AX, R10
	ADDL $16, R10
	SARL $5, R10
	CMPL R10, $0
	JGE  qpel16_avg10_nonnegative
	XORL R10, R10
	JMP  qpel16_avg10_clip_done
qpel16_avg10_nonnegative:
	CMPL R10, $255
	JLE  qpel16_avg10_clip_done
	MOVL $255, R10
qpel16_avg10_clip_done:
	XORL AX, AX
	MOVB (R12), AX
	ADDL AX, R10
	ADDL $1, R10
	SHRL $1, R10
	XORL AX, AX
	MOVB (R11), AX
	ADDL AX, R10
	ADDL $1, R10
	SHRL $1, R10
	MOVB R10, (R11)
	INCQ R11
	INCQ R12
	DECL R9
	JNZ  qpel16_avg10_col
	ADDQ DX, DI
	ADDQ CX, SI
	DECL R8
	JNZ  qpel16_avg10_row
	RET

// func h264QpelMC8Put10ASM(dst *uint8, src *uint8, dstStride int, srcStride int)
TEXT ·h264QpelMC8Put10ASM(SB), NOSPLIT, $0-32
	MOVQ dst+0(FP), DI
	MOVQ src+8(FP), SI
	MOVQ dstStride+16(FP), DX
	MOVQ srcStride+24(FP), CX
	MOVL $8, R8
qpel8_put10_row:
	MOVQ DI, R11
	MOVQ SI, R12
	MOVL $8, R9
qpel8_put10_col:
	XORL AX, AX
	MOVB (R12), AX
	XORL BX, BX
	MOVB 1(R12), BX
	ADDL BX, AX
	IMULL $20, AX
	MOVL AX, R10
	XORL AX, AX
	MOVB -1(R12), AX
	XORL BX, BX
	MOVB 2(R12), BX
	ADDL BX, AX
	LEAL (AX)(AX*4), AX
	SUBL AX, R10
	XORL AX, AX
	MOVB -2(R12), AX
	XORL BX, BX
	MOVB 3(R12), BX
	ADDL BX, AX
	ADDL AX, R10
	ADDL $16, R10
	SARL $5, R10
	CMPL R10, $0
	JGE  qpel8_put10_nonnegative
	XORL R10, R10
	JMP  qpel8_put10_clip_done
qpel8_put10_nonnegative:
	CMPL R10, $255
	JLE  qpel8_put10_clip_done
	MOVL $255, R10
qpel8_put10_clip_done:
	XORL AX, AX
	MOVB (R12), AX
	ADDL AX, R10
	ADDL $1, R10
	SHRL $1, R10
	MOVB R10, (R11)
	INCQ R11
	INCQ R12
	DECL R9
	JNZ  qpel8_put10_col
	ADDQ DX, DI
	ADDQ CX, SI
	DECL R8
	JNZ  qpel8_put10_row
	RET

// func h264QpelMC8Avg10ASM(dst *uint8, src *uint8, dstStride int, srcStride int)
TEXT ·h264QpelMC8Avg10ASM(SB), NOSPLIT, $0-32
	MOVQ dst+0(FP), DI
	MOVQ src+8(FP), SI
	MOVQ dstStride+16(FP), DX
	MOVQ srcStride+24(FP), CX
	MOVL $8, R8
qpel8_avg10_row:
	MOVQ DI, R11
	MOVQ SI, R12
	MOVL $8, R9
qpel8_avg10_col:
	XORL AX, AX
	MOVB (R12), AX
	XORL BX, BX
	MOVB 1(R12), BX
	ADDL BX, AX
	IMULL $20, AX
	MOVL AX, R10
	XORL AX, AX
	MOVB -1(R12), AX
	XORL BX, BX
	MOVB 2(R12), BX
	ADDL BX, AX
	LEAL (AX)(AX*4), AX
	SUBL AX, R10
	XORL AX, AX
	MOVB -2(R12), AX
	XORL BX, BX
	MOVB 3(R12), BX
	ADDL BX, AX
	ADDL AX, R10
	ADDL $16, R10
	SARL $5, R10
	CMPL R10, $0
	JGE  qpel8_avg10_nonnegative
	XORL R10, R10
	JMP  qpel8_avg10_clip_done
qpel8_avg10_nonnegative:
	CMPL R10, $255
	JLE  qpel8_avg10_clip_done
	MOVL $255, R10
qpel8_avg10_clip_done:
	XORL AX, AX
	MOVB (R12), AX
	ADDL AX, R10
	ADDL $1, R10
	SHRL $1, R10
	XORL AX, AX
	MOVB (R11), AX
	ADDL AX, R10
	ADDL $1, R10
	SHRL $1, R10
	MOVB R10, (R11)
	INCQ R11
	INCQ R12
	DECL R9
	JNZ  qpel8_avg10_col
	ADDQ DX, DI
	ADDQ CX, SI
	DECL R8
	JNZ  qpel8_avg10_row
	RET

// func h264QpelMC16Put20ASM(dst *uint8, src *uint8, dstStride int, srcStride int)
TEXT ·h264QpelMC16Put20ASM(SB), NOSPLIT, $0-32
	MOVQ dst+0(FP), DI
	MOVQ src+8(FP), SI
	MOVQ dstStride+16(FP), DX
	MOVQ srcStride+24(FP), CX
	MOVL $16, R8
qpel16_put20_row:
	MOVQ DI, R11
	MOVQ SI, R12
	MOVL $16, R9
qpel16_put20_col:
	XORL AX, AX
	MOVB (R12), AX
	XORL BX, BX
	MOVB 1(R12), BX
	ADDL BX, AX
	IMULL $20, AX
	MOVL AX, R10
	XORL AX, AX
	MOVB -1(R12), AX
	XORL BX, BX
	MOVB 2(R12), BX
	ADDL BX, AX
	LEAL (AX)(AX*4), AX
	SUBL AX, R10
	XORL AX, AX
	MOVB -2(R12), AX
	XORL BX, BX
	MOVB 3(R12), BX
	ADDL BX, AX
	ADDL AX, R10
	ADDL $16, R10
	SARL $5, R10
	CMPL R10, $0
	JGE  qpel16_put20_nonnegative
	XORL R10, R10
	JMP  qpel16_put20_store
qpel16_put20_nonnegative:
	CMPL R10, $255
	JLE  qpel16_put20_store
	MOVL $255, R10
qpel16_put20_store:
	MOVB R10, (R11)
	INCQ R11
	INCQ R12
	DECL R9
	JNZ  qpel16_put20_col
	ADDQ DX, DI
	ADDQ CX, SI
	DECL R8
	JNZ  qpel16_put20_row
	RET

// func h264QpelMC16Avg20ASM(dst *uint8, src *uint8, dstStride int, srcStride int)
TEXT ·h264QpelMC16Avg20ASM(SB), NOSPLIT, $0-32
	MOVQ dst+0(FP), DI
	MOVQ src+8(FP), SI
	MOVQ dstStride+16(FP), DX
	MOVQ srcStride+24(FP), CX
	MOVL $16, R8
qpel16_avg20_row:
	MOVQ DI, R11
	MOVQ SI, R12
	MOVL $16, R9
qpel16_avg20_col:
	XORL AX, AX
	MOVB (R12), AX
	XORL BX, BX
	MOVB 1(R12), BX
	ADDL BX, AX
	IMULL $20, AX
	MOVL AX, R10
	XORL AX, AX
	MOVB -1(R12), AX
	XORL BX, BX
	MOVB 2(R12), BX
	ADDL BX, AX
	LEAL (AX)(AX*4), AX
	SUBL AX, R10
	XORL AX, AX
	MOVB -2(R12), AX
	XORL BX, BX
	MOVB 3(R12), BX
	ADDL BX, AX
	ADDL AX, R10
	ADDL $16, R10
	SARL $5, R10
	CMPL R10, $0
	JGE  qpel16_avg20_nonnegative
	XORL R10, R10
	JMP  qpel16_avg20_clip_done
qpel16_avg20_nonnegative:
	CMPL R10, $255
	JLE  qpel16_avg20_clip_done
	MOVL $255, R10
qpel16_avg20_clip_done:
	XORL AX, AX
	MOVB (R11), AX
	ADDL AX, R10
	ADDL $1, R10
	SHRL $1, R10
	MOVB R10, (R11)
	INCQ R11
	INCQ R12
	DECL R9
	JNZ  qpel16_avg20_col
	ADDQ DX, DI
	ADDQ CX, SI
	DECL R8
	JNZ  qpel16_avg20_row
	RET

// func h264QpelMC8Put20ASM(dst *uint8, src *uint8, dstStride int, srcStride int)
TEXT ·h264QpelMC8Put20ASM(SB), NOSPLIT, $0-32
	MOVQ dst+0(FP), DI
	MOVQ src+8(FP), SI
	MOVQ dstStride+16(FP), DX
	MOVQ srcStride+24(FP), CX
	MOVL $8, R8
qpel8_put20_row:
	MOVQ DI, R11
	MOVQ SI, R12
	MOVL $8, R9
qpel8_put20_col:
	XORL AX, AX
	MOVB (R12), AX
	XORL BX, BX
	MOVB 1(R12), BX
	ADDL BX, AX
	IMULL $20, AX
	MOVL AX, R10
	XORL AX, AX
	MOVB -1(R12), AX
	XORL BX, BX
	MOVB 2(R12), BX
	ADDL BX, AX
	LEAL (AX)(AX*4), AX
	SUBL AX, R10
	XORL AX, AX
	MOVB -2(R12), AX
	XORL BX, BX
	MOVB 3(R12), BX
	ADDL BX, AX
	ADDL AX, R10
	ADDL $16, R10
	SARL $5, R10
	CMPL R10, $0
	JGE  qpel8_put20_nonnegative
	XORL R10, R10
	JMP  qpel8_put20_store
qpel8_put20_nonnegative:
	CMPL R10, $255
	JLE  qpel8_put20_store
	MOVL $255, R10
qpel8_put20_store:
	MOVB R10, (R11)
	INCQ R11
	INCQ R12
	DECL R9
	JNZ  qpel8_put20_col
	ADDQ DX, DI
	ADDQ CX, SI
	DECL R8
	JNZ  qpel8_put20_row
	RET

// func h264QpelMC8Avg20ASM(dst *uint8, src *uint8, dstStride int, srcStride int)
TEXT ·h264QpelMC8Avg20ASM(SB), NOSPLIT, $0-32
	MOVQ dst+0(FP), DI
	MOVQ src+8(FP), SI
	MOVQ dstStride+16(FP), DX
	MOVQ srcStride+24(FP), CX
	MOVL $8, R8
qpel8_avg20_row:
	MOVQ DI, R11
	MOVQ SI, R12
	MOVL $8, R9
qpel8_avg20_col:
	XORL AX, AX
	MOVB (R12), AX
	XORL BX, BX
	MOVB 1(R12), BX
	ADDL BX, AX
	IMULL $20, AX
	MOVL AX, R10
	XORL AX, AX
	MOVB -1(R12), AX
	XORL BX, BX
	MOVB 2(R12), BX
	ADDL BX, AX
	LEAL (AX)(AX*4), AX
	SUBL AX, R10
	XORL AX, AX
	MOVB -2(R12), AX
	XORL BX, BX
	MOVB 3(R12), BX
	ADDL BX, AX
	ADDL AX, R10
	ADDL $16, R10
	SARL $5, R10
	CMPL R10, $0
	JGE  qpel8_avg20_nonnegative
	XORL R10, R10
	JMP  qpel8_avg20_clip_done
qpel8_avg20_nonnegative:
	CMPL R10, $255
	JLE  qpel8_avg20_clip_done
	MOVL $255, R10
qpel8_avg20_clip_done:
	XORL AX, AX
	MOVB (R11), AX
	ADDL AX, R10
	ADDL $1, R10
	SHRL $1, R10
	MOVB R10, (R11)
	INCQ R11
	INCQ R12
	DECL R9
	JNZ  qpel8_avg20_col
	ADDQ DX, DI
	ADDQ CX, SI
	DECL R8
	JNZ  qpel8_avg20_row
	RET

// func h264QpelMC16Put30ASM(dst *uint8, src *uint8, dstStride int, srcStride int)
TEXT ·h264QpelMC16Put30ASM(SB), NOSPLIT, $0-32
	MOVQ dst+0(FP), DI
	MOVQ src+8(FP), SI
	MOVQ dstStride+16(FP), DX
	MOVQ srcStride+24(FP), CX
	MOVL $16, R8
qpel16_put30_row:
	MOVQ DI, R11
	MOVQ SI, R12
	MOVL $16, R9
qpel16_put30_col:
	XORL AX, AX
	MOVB (R12), AX
	XORL BX, BX
	MOVB 1(R12), BX
	ADDL BX, AX
	IMULL $20, AX
	MOVL AX, R10
	XORL AX, AX
	MOVB -1(R12), AX
	XORL BX, BX
	MOVB 2(R12), BX
	ADDL BX, AX
	LEAL (AX)(AX*4), AX
	SUBL AX, R10
	XORL AX, AX
	MOVB -2(R12), AX
	XORL BX, BX
	MOVB 3(R12), BX
	ADDL BX, AX
	ADDL AX, R10
	ADDL $16, R10
	SARL $5, R10
	CMPL R10, $0
	JGE  qpel16_put30_nonnegative
	XORL R10, R10
	JMP  qpel16_put30_clip_done
qpel16_put30_nonnegative:
	CMPL R10, $255
	JLE  qpel16_put30_clip_done
	MOVL $255, R10
qpel16_put30_clip_done:
	XORL AX, AX
	MOVB 1(R12), AX
	ADDL AX, R10
	ADDL $1, R10
	SHRL $1, R10
	MOVB R10, (R11)
	INCQ R11
	INCQ R12
	DECL R9
	JNZ  qpel16_put30_col
	ADDQ DX, DI
	ADDQ CX, SI
	DECL R8
	JNZ  qpel16_put30_row
	RET

// func h264QpelMC16Avg30ASM(dst *uint8, src *uint8, dstStride int, srcStride int)
TEXT ·h264QpelMC16Avg30ASM(SB), NOSPLIT, $0-32
	MOVQ dst+0(FP), DI
	MOVQ src+8(FP), SI
	MOVQ dstStride+16(FP), DX
	MOVQ srcStride+24(FP), CX
	MOVL $16, R8
qpel16_avg30_row:
	MOVQ DI, R11
	MOVQ SI, R12
	MOVL $16, R9
qpel16_avg30_col:
	XORL AX, AX
	MOVB (R12), AX
	XORL BX, BX
	MOVB 1(R12), BX
	ADDL BX, AX
	IMULL $20, AX
	MOVL AX, R10
	XORL AX, AX
	MOVB -1(R12), AX
	XORL BX, BX
	MOVB 2(R12), BX
	ADDL BX, AX
	LEAL (AX)(AX*4), AX
	SUBL AX, R10
	XORL AX, AX
	MOVB -2(R12), AX
	XORL BX, BX
	MOVB 3(R12), BX
	ADDL BX, AX
	ADDL AX, R10
	ADDL $16, R10
	SARL $5, R10
	CMPL R10, $0
	JGE  qpel16_avg30_nonnegative
	XORL R10, R10
	JMP  qpel16_avg30_clip_done
qpel16_avg30_nonnegative:
	CMPL R10, $255
	JLE  qpel16_avg30_clip_done
	MOVL $255, R10
qpel16_avg30_clip_done:
	XORL AX, AX
	MOVB 1(R12), AX
	ADDL AX, R10
	ADDL $1, R10
	SHRL $1, R10
	XORL AX, AX
	MOVB (R11), AX
	ADDL AX, R10
	ADDL $1, R10
	SHRL $1, R10
	MOVB R10, (R11)
	INCQ R11
	INCQ R12
	DECL R9
	JNZ  qpel16_avg30_col
	ADDQ DX, DI
	ADDQ CX, SI
	DECL R8
	JNZ  qpel16_avg30_row
	RET

// func h264QpelMC8Put30ASM(dst *uint8, src *uint8, dstStride int, srcStride int)
TEXT ·h264QpelMC8Put30ASM(SB), NOSPLIT, $0-32
	MOVQ dst+0(FP), DI
	MOVQ src+8(FP), SI
	MOVQ dstStride+16(FP), DX
	MOVQ srcStride+24(FP), CX
	MOVL $8, R8
qpel8_put30_row:
	MOVQ DI, R11
	MOVQ SI, R12
	MOVL $8, R9
qpel8_put30_col:
	XORL AX, AX
	MOVB (R12), AX
	XORL BX, BX
	MOVB 1(R12), BX
	ADDL BX, AX
	IMULL $20, AX
	MOVL AX, R10
	XORL AX, AX
	MOVB -1(R12), AX
	XORL BX, BX
	MOVB 2(R12), BX
	ADDL BX, AX
	LEAL (AX)(AX*4), AX
	SUBL AX, R10
	XORL AX, AX
	MOVB -2(R12), AX
	XORL BX, BX
	MOVB 3(R12), BX
	ADDL BX, AX
	ADDL AX, R10
	ADDL $16, R10
	SARL $5, R10
	CMPL R10, $0
	JGE  qpel8_put30_nonnegative
	XORL R10, R10
	JMP  qpel8_put30_clip_done
qpel8_put30_nonnegative:
	CMPL R10, $255
	JLE  qpel8_put30_clip_done
	MOVL $255, R10
qpel8_put30_clip_done:
	XORL AX, AX
	MOVB 1(R12), AX
	ADDL AX, R10
	ADDL $1, R10
	SHRL $1, R10
	MOVB R10, (R11)
	INCQ R11
	INCQ R12
	DECL R9
	JNZ  qpel8_put30_col
	ADDQ DX, DI
	ADDQ CX, SI
	DECL R8
	JNZ  qpel8_put30_row
	RET

// func h264QpelMC8Avg30ASM(dst *uint8, src *uint8, dstStride int, srcStride int)
TEXT ·h264QpelMC8Avg30ASM(SB), NOSPLIT, $0-32
	MOVQ dst+0(FP), DI
	MOVQ src+8(FP), SI
	MOVQ dstStride+16(FP), DX
	MOVQ srcStride+24(FP), CX
	MOVL $8, R8
qpel8_avg30_row:
	MOVQ DI, R11
	MOVQ SI, R12
	MOVL $8, R9
qpel8_avg30_col:
	XORL AX, AX
	MOVB (R12), AX
	XORL BX, BX
	MOVB 1(R12), BX
	ADDL BX, AX
	IMULL $20, AX
	MOVL AX, R10
	XORL AX, AX
	MOVB -1(R12), AX
	XORL BX, BX
	MOVB 2(R12), BX
	ADDL BX, AX
	LEAL (AX)(AX*4), AX
	SUBL AX, R10
	XORL AX, AX
	MOVB -2(R12), AX
	XORL BX, BX
	MOVB 3(R12), BX
	ADDL BX, AX
	ADDL AX, R10
	ADDL $16, R10
	SARL $5, R10
	CMPL R10, $0
	JGE  qpel8_avg30_nonnegative
	XORL R10, R10
	JMP  qpel8_avg30_clip_done
qpel8_avg30_nonnegative:
	CMPL R10, $255
	JLE  qpel8_avg30_clip_done
	MOVL $255, R10
qpel8_avg30_clip_done:
	XORL AX, AX
	MOVB 1(R12), AX
	ADDL AX, R10
	ADDL $1, R10
	SHRL $1, R10
	XORL AX, AX
	MOVB (R11), AX
	ADDL AX, R10
	ADDL $1, R10
	SHRL $1, R10
	MOVB R10, (R11)
	INCQ R11
	INCQ R12
	DECL R9
	JNZ  qpel8_avg30_col
	ADDQ DX, DI
	ADDQ CX, SI
	DECL R8
	JNZ  qpel8_avg30_row
	RET

// func h264QpelMCPut0YASM(dst *uint8, src *uint8, dstStride int, srcStride int, size int32, my int32)
// func h264QpelMCPutX0ASM(dst *uint8, src *uint8, dstStride int, srcStride int, size int32, mx int32)
TEXT ·h264QpelMCPutX0ASM(SB), NOSPLIT, $0-40
	MOVQ dst+0(FP), DI
	MOVQ src+8(FP), SI
	MOVQ dstStride+16(FP), DX
	MOVQ srcStride+24(FP), CX
	MOVL size+32(FP), R8
	MOVL mx+36(FP), R15
qpel_putx0_row:
	MOVQ DI, R11
	MOVQ SI, R12
	MOVL size+32(FP), R9
qpel_putx0_col:
	XORL AX, AX
	MOVB (R12), AX
	XORL BX, BX
	MOVB 1(R12), BX
	ADDL BX, AX
	IMULL $20, AX
	MOVL AX, R10
	XORL AX, AX
	MOVB -1(R12), AX
	XORL BX, BX
	MOVB 2(R12), BX
	ADDL BX, AX
	LEAL (AX)(AX*4), AX
	SUBL AX, R10
	XORL AX, AX
	MOVB -2(R12), AX
	XORL BX, BX
	MOVB 3(R12), BX
	ADDL BX, AX
	ADDL AX, R10
	ADDL $16, R10
	SARL $5, R10
	CMPL R10, $0
	JGE  qpel_putx0_nonnegative
	XORL R10, R10
	JMP  qpel_putx0_clip_done
qpel_putx0_nonnegative:
	CMPL R10, $255
	JLE  qpel_putx0_clip_done
	MOVL $255, R10
qpel_putx0_clip_done:
	CMPL R15, $2
	JE   qpel_putx0_store
	XORL AX, AX
	CMPL R15, $1
	JNE  qpel_putx0_load_next
	MOVB (R12), AX
	JMP  qpel_putx0_l2
qpel_putx0_load_next:
	MOVB 1(R12), AX
qpel_putx0_l2:
	ADDL AX, R10
	ADDL $1, R10
	SHRL $1, R10
qpel_putx0_store:
	MOVB R10, (R11)
	INCQ R11
	INCQ R12
	DECL R9
	JNZ  qpel_putx0_col
	ADDQ DX, DI
	ADDQ CX, SI
	DECL R8
	JNZ  qpel_putx0_row
	RET

// func h264QpelMCAvgX0ASM(dst *uint8, src *uint8, dstStride int, srcStride int, size int32, mx int32)
TEXT ·h264QpelMCAvgX0ASM(SB), NOSPLIT, $0-40
	MOVQ dst+0(FP), DI
	MOVQ src+8(FP), SI
	MOVQ dstStride+16(FP), DX
	MOVQ srcStride+24(FP), CX
	MOVL size+32(FP), R8
	MOVL mx+36(FP), R15
qpel_avgx0_row:
	MOVQ DI, R11
	MOVQ SI, R12
	MOVL size+32(FP), R9
qpel_avgx0_col:
	XORL AX, AX
	MOVB (R12), AX
	XORL BX, BX
	MOVB 1(R12), BX
	ADDL BX, AX
	IMULL $20, AX
	MOVL AX, R10
	XORL AX, AX
	MOVB -1(R12), AX
	XORL BX, BX
	MOVB 2(R12), BX
	ADDL BX, AX
	LEAL (AX)(AX*4), AX
	SUBL AX, R10
	XORL AX, AX
	MOVB -2(R12), AX
	XORL BX, BX
	MOVB 3(R12), BX
	ADDL BX, AX
	ADDL AX, R10
	ADDL $16, R10
	SARL $5, R10
	CMPL R10, $0
	JGE  qpel_avgx0_nonnegative
	XORL R10, R10
	JMP  qpel_avgx0_clip_done
qpel_avgx0_nonnegative:
	CMPL R10, $255
	JLE  qpel_avgx0_clip_done
	MOVL $255, R10
qpel_avgx0_clip_done:
	CMPL R15, $2
	JE   qpel_avgx0_pred_done
	XORL AX, AX
	CMPL R15, $1
	JNE  qpel_avgx0_load_next
	MOVB (R12), AX
	JMP  qpel_avgx0_l2
qpel_avgx0_load_next:
	MOVB 1(R12), AX
qpel_avgx0_l2:
	ADDL AX, R10
	ADDL $1, R10
	SHRL $1, R10
qpel_avgx0_pred_done:
	XORL AX, AX
	MOVB (R11), AX
	ADDL AX, R10
	ADDL $1, R10
	SHRL $1, R10
	MOVB R10, (R11)
	INCQ R11
	INCQ R12
	DECL R9
	JNZ  qpel_avgx0_col
	ADDQ DX, DI
	ADDQ CX, SI
	DECL R8
	JNZ  qpel_avgx0_row
	RET

// func h264QpelMCPut0YASM(dst *uint8, src *uint8, dstStride int, srcStride int, size int32, my int32)
TEXT ·h264QpelMCPut0YASM(SB), NOSPLIT, $0-40
	MOVQ dst+0(FP), DI
	MOVQ src+8(FP), SI
	MOVQ dstStride+16(FP), DX
	MOVQ srcStride+24(FP), CX
	MOVL size+32(FP), R8
	MOVL my+36(FP), R15
qpel_put0y_row:
	MOVQ DI, R11
	MOVQ SI, R12
	MOVL size+32(FP), R9
qpel_put0y_col:
	XORL AX, AX
	MOVB (R12), AX
	MOVQ R12, R13
	ADDQ CX, R13
	XORL BX, BX
	MOVB (R13), BX
	ADDL BX, AX
	IMULL $20, AX
	MOVL AX, R10
	MOVQ R12, R13
	SUBQ CX, R13
	XORL AX, AX
	MOVB (R13), AX
	MOVQ R12, R13
	ADDQ CX, R13
	ADDQ CX, R13
	XORL BX, BX
	MOVB (R13), BX
	ADDL BX, AX
	LEAL (AX)(AX*4), AX
	SUBL AX, R10
	MOVQ R12, R13
	SUBQ CX, R13
	SUBQ CX, R13
	XORL AX, AX
	MOVB (R13), AX
	MOVQ R12, R13
	ADDQ CX, R13
	ADDQ CX, R13
	ADDQ CX, R13
	XORL BX, BX
	MOVB (R13), BX
	ADDL BX, AX
	ADDL AX, R10
	ADDL $16, R10
	SARL $5, R10
	CMPL R10, $0
	JGE  qpel_put0y_nonnegative
	XORL R10, R10
	JMP  qpel_put0y_clip_done
qpel_put0y_nonnegative:
	CMPL R10, $255
	JLE  qpel_put0y_clip_done
	MOVL $255, R10
qpel_put0y_clip_done:
	CMPL R15, $2
	JE   qpel_put0y_store
	XORL AX, AX
	CMPL R15, $1
	JNE  qpel_put0y_load_next
	MOVB (R12), AX
	JMP  qpel_put0y_l2
qpel_put0y_load_next:
	MOVQ R12, R13
	ADDQ CX, R13
	MOVB (R13), AX
qpel_put0y_l2:
	ADDL AX, R10
	ADDL $1, R10
	SHRL $1, R10
qpel_put0y_store:
	MOVB R10, (R11)
	INCQ R11
	INCQ R12
	DECL R9
	JNZ  qpel_put0y_col
	ADDQ DX, DI
	ADDQ CX, SI
	DECL R8
	JNZ  qpel_put0y_row
	RET

// func h264QpelMCAvg0YASM(dst *uint8, src *uint8, dstStride int, srcStride int, size int32, my int32)
TEXT ·h264QpelMCAvg0YASM(SB), NOSPLIT, $0-40
	MOVQ dst+0(FP), DI
	MOVQ src+8(FP), SI
	MOVQ dstStride+16(FP), DX
	MOVQ srcStride+24(FP), CX
	MOVL size+32(FP), R8
	MOVL my+36(FP), R15
qpel_avg0y_row:
	MOVQ DI, R11
	MOVQ SI, R12
	MOVL size+32(FP), R9
qpel_avg0y_col:
	XORL AX, AX
	MOVB (R12), AX
	MOVQ R12, R13
	ADDQ CX, R13
	XORL BX, BX
	MOVB (R13), BX
	ADDL BX, AX
	IMULL $20, AX
	MOVL AX, R10
	MOVQ R12, R13
	SUBQ CX, R13
	XORL AX, AX
	MOVB (R13), AX
	MOVQ R12, R13
	ADDQ CX, R13
	ADDQ CX, R13
	XORL BX, BX
	MOVB (R13), BX
	ADDL BX, AX
	LEAL (AX)(AX*4), AX
	SUBL AX, R10
	MOVQ R12, R13
	SUBQ CX, R13
	SUBQ CX, R13
	XORL AX, AX
	MOVB (R13), AX
	MOVQ R12, R13
	ADDQ CX, R13
	ADDQ CX, R13
	ADDQ CX, R13
	XORL BX, BX
	MOVB (R13), BX
	ADDL BX, AX
	ADDL AX, R10
	ADDL $16, R10
	SARL $5, R10
	CMPL R10, $0
	JGE  qpel_avg0y_nonnegative
	XORL R10, R10
	JMP  qpel_avg0y_clip_done
qpel_avg0y_nonnegative:
	CMPL R10, $255
	JLE  qpel_avg0y_clip_done
	MOVL $255, R10
qpel_avg0y_clip_done:
	CMPL R15, $2
	JE   qpel_avg0y_pred_done
	XORL AX, AX
	CMPL R15, $1
	JNE  qpel_avg0y_load_next
	MOVB (R12), AX
	JMP  qpel_avg0y_l2
qpel_avg0y_load_next:
	MOVQ R12, R13
	ADDQ CX, R13
	MOVB (R13), AX
qpel_avg0y_l2:
	ADDL AX, R10
	ADDL $1, R10
	SHRL $1, R10
qpel_avg0y_pred_done:
	XORL AX, AX
	MOVB (R11), AX
	ADDL AX, R10
	ADDL $1, R10
	SHRL $1, R10
	MOVB R10, (R11)
	INCQ R11
	INCQ R12
	DECL R9
	JNZ  qpel_avg0y_col
	ADDQ DX, DI
	ADDQ CX, SI
	DECL R8
	JNZ  qpel_avg0y_row
	RET

// func h264QpelMCPut22ASM(dst *uint8, src *uint8, dstStride int, srcStride int, size int32)
TEXT ·h264QpelMCPut22ASM(SB), NOSPLIT, $32-40
	MOVQ dst+0(FP), DI
	MOVQ src+8(FP), SI
	MOVQ dstStride+16(FP), DX
	MOVQ srcStride+24(FP), CX
	MOVL size+32(FP), R8
qpel_put22_row:
	MOVQ DI, R11
	MOVQ SI, R12
	MOVL size+32(FP), R9
qpel_put22_col:
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
	JGE  qpel_put22_nonnegative
	XORL R10, R10
	JMP  qpel_put22_store
qpel_put22_nonnegative:
	CMPL R10, $255
	JLE  qpel_put22_store
	MOVL $255, R10
qpel_put22_store:
	MOVB R10, (R11)
	INCQ R11
	INCQ R12
	DECL R9
	JNZ  qpel_put22_col
	ADDQ DX, DI
	ADDQ CX, SI
	DECL R8
	JNZ  qpel_put22_row
	RET

// func h264QpelMCAvg22ASM(dst *uint8, src *uint8, dstStride int, srcStride int, size int32)
TEXT ·h264QpelMCAvg22ASM(SB), NOSPLIT, $32-40
	MOVQ dst+0(FP), DI
	MOVQ src+8(FP), SI
	MOVQ dstStride+16(FP), DX
	MOVQ srcStride+24(FP), CX
	MOVL size+32(FP), R8
qpel_avg22_row:
	MOVQ DI, R11
	MOVQ SI, R12
	MOVL size+32(FP), R9
qpel_avg22_col:
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
	JGE  qpel_avg22_nonnegative
	XORL R10, R10
	JMP  qpel_avg22_clip_done
qpel_avg22_nonnegative:
	CMPL R10, $255
	JLE  qpel_avg22_clip_done
	MOVL $255, R10
qpel_avg22_clip_done:
	XORL AX, AX
	MOVB (R11), AX
	ADDL AX, R10
	ADDL $1, R10
	SHRL $1, R10
	MOVB R10, (R11)
	INCQ R11
	INCQ R12
	DECL R9
	JNZ  qpel_avg22_col
	ADDQ DX, DI
	ADDQ CX, SI
	DECL R8
	JNZ  qpel_avg22_row
	RET

// func h264QpelMCPutHVXYASM(dst *uint8, src *uint8, dstStride int, srcStride int, size int32, mx int32, my int32)
TEXT ·h264QpelMCPutHVXYASM(SB), NOSPLIT, $8-48
	MOVQ dst+0(FP), DI
	MOVQ src+8(FP), SI
	MOVQ dstStride+16(FP), DX
	MOVQ srcStride+24(FP), CX
	MOVL size+32(FP), R8
qpel_puthvxy_row:
	MOVQ DI, R11
	MOVQ SI, R12
	MOVL size+32(FP), R9
qpel_puthvxy_col:
	MOVQ R12, R13
	MOVL my+40(FP), R15
	CMPL R15, $3
	JNE  qpel_puthvxy_hptr_ready
	ADDQ CX, R13
qpel_puthvxy_hptr_ready:
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
	JGE  qpel_puthvxy_h_nonnegative
	XORL R10, R10
	JMP  qpel_puthvxy_h_done
qpel_puthvxy_h_nonnegative:
	CMPL R10, $255
	JLE  qpel_puthvxy_h_done
	MOVL $255, R10
qpel_puthvxy_h_done:
	MOVL R10, 0(SP)
	MOVQ R12, R13
	MOVL mx+36(FP), R15
	CMPL R15, $3
	JNE  qpel_puthvxy_vptr_ready
	INCQ R13
qpel_puthvxy_vptr_ready:
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
	JGE  qpel_puthvxy_v_nonnegative
	XORL R10, R10
	JMP  qpel_puthvxy_v_done
qpel_puthvxy_v_nonnegative:
	CMPL R10, $255
	JLE  qpel_puthvxy_v_done
	MOVL $255, R10
qpel_puthvxy_v_done:
	MOVL 0(SP), AX
	ADDL AX, R10
	ADDL $1, R10
	SHRL $1, R10
	MOVB R10, (R11)
	INCQ R11
	INCQ R12
	DECL R9
	JNZ  qpel_puthvxy_col
	ADDQ DX, DI
	ADDQ CX, SI
	DECL R8
	JNZ  qpel_puthvxy_row
	RET

// func h264QpelMCAvgHVXYASM(dst *uint8, src *uint8, dstStride int, srcStride int, size int32, mx int32, my int32)
TEXT ·h264QpelMCAvgHVXYASM(SB), NOSPLIT, $8-48
	MOVQ dst+0(FP), DI
	MOVQ src+8(FP), SI
	MOVQ dstStride+16(FP), DX
	MOVQ srcStride+24(FP), CX
	MOVL size+32(FP), R8
qpel_avghvxy_row:
	MOVQ DI, R11
	MOVQ SI, R12
	MOVL size+32(FP), R9
qpel_avghvxy_col:
	MOVQ R12, R13
	MOVL my+40(FP), R15
	CMPL R15, $3
	JNE  qpel_avghvxy_hptr_ready
	ADDQ CX, R13
qpel_avghvxy_hptr_ready:
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
	JGE  qpel_avghvxy_h_nonnegative
	XORL R10, R10
	JMP  qpel_avghvxy_h_done
qpel_avghvxy_h_nonnegative:
	CMPL R10, $255
	JLE  qpel_avghvxy_h_done
	MOVL $255, R10
qpel_avghvxy_h_done:
	MOVL R10, 0(SP)
	MOVQ R12, R13
	MOVL mx+36(FP), R15
	CMPL R15, $3
	JNE  qpel_avghvxy_vptr_ready
	INCQ R13
qpel_avghvxy_vptr_ready:
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
	JGE  qpel_avghvxy_v_nonnegative
	XORL R10, R10
	JMP  qpel_avghvxy_v_done
qpel_avghvxy_v_nonnegative:
	CMPL R10, $255
	JLE  qpel_avghvxy_v_done
	MOVL $255, R10
qpel_avghvxy_v_done:
	MOVL 0(SP), AX
	ADDL AX, R10
	ADDL $1, R10
	SHRL $1, R10
	XORL AX, AX
	MOVB (R11), AX
	ADDL AX, R10
	ADDL $1, R10
	SHRL $1, R10
	MOVB R10, (R11)
	INCQ R11
	INCQ R12
	DECL R9
	JNZ  qpel_avghvxy_col
	ADDQ DX, DI
	ADDQ CX, SI
	DECL R8
	JNZ  qpel_avghvxy_row
	RET

// func h264QpelMC4Put00ASM(dst *uint8, src *uint8, dstStride int, srcStride int)
TEXT ·h264QpelMC4Put00ASM(SB), NOSPLIT, $0-32
	MOVQ dst+0(FP), DI
	MOVQ src+8(FP), SI
	MOVQ dstStride+16(FP), DX
	MOVQ srcStride+24(FP), CX
	MOVL $4, R8
qpel4_put00_loop:
	MOVL (SI), AX
	MOVL AX, (DI)
	ADDQ DX, DI
	ADDQ CX, SI
	DECL R8
	JNZ  qpel4_put00_loop
	RET

// func h264QpelMC4Avg00ASM(dst *uint8, src *uint8, dstStride int, srcStride int)
TEXT ·h264QpelMC4Avg00ASM(SB), NOSPLIT, $0-32
	MOVQ dst+0(FP), DI
	MOVQ src+8(FP), SI
	MOVQ dstStride+16(FP), DX
	MOVQ srcStride+24(FP), CX
	MOVL $4, R8
	MOVL $0xfefefefe, R9
qpel4_avg00_loop:
	MOVL (SI), AX
	MOVL (DI), BX
	MOVL AX, R10
	ORL  BX, R10
	XORL BX, AX
	ANDL R9, AX
	SHRL $1, AX
	SUBL AX, R10
	MOVL R10, (DI)
	ADDQ DX, DI
	ADDQ CX, SI
	DECL R8
	JNZ  qpel4_avg00_loop
	RET

// func h264QpelMC2Put00ASM(dst *uint8, src *uint8, dstStride int, srcStride int)
TEXT ·h264QpelMC2Put00ASM(SB), NOSPLIT, $0-32
	MOVQ dst+0(FP), DI
	MOVQ src+8(FP), SI
	MOVQ dstStride+16(FP), DX
	MOVQ srcStride+24(FP), CX
	MOVL $2, R8
qpel2_put00_loop:
	MOVW (SI), AX
	MOVW AX, (DI)
	ADDQ DX, DI
	ADDQ CX, SI
	DECL R8
	JNZ  qpel2_put00_loop
	RET

// func h264QpelMC2Avg00ASM(dst *uint8, src *uint8, dstStride int, srcStride int)
TEXT ·h264QpelMC2Avg00ASM(SB), NOSPLIT, $0-32
	MOVQ dst+0(FP), DI
	MOVQ src+8(FP), SI
	MOVQ dstStride+16(FP), DX
	MOVQ srcStride+24(FP), CX
	MOVL $2, R8
	MOVL $0xfefe, R9
qpel2_avg00_loop:
	XORL AX, AX
	XORL BX, BX
	MOVW (SI), AX
	MOVW (DI), BX
	MOVL AX, R10
	ORL  BX, R10
	XORL BX, AX
	ANDL R9, AX
	SHRL $1, AX
	SUBL AX, R10
	MOVW R10, (DI)
	ADDQ DX, DI
	ADDQ CX, SI
	DECL R8
	JNZ  qpel2_avg00_loop
	RET
