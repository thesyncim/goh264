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
