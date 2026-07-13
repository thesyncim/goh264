// SPDX-License-Identifier: LGPL-2.1-or-later
//go:build !purego && arm64

#include "textflag.h"

// h264QpelH8AxisNEONInternal applies the horizontal six-tap filter to an
// 8-pixel column group, with optional quarter-pel and destination averaging.
TEXT ·h264QpelH8AxisNEONInternal(SB), NOSPLIT|NOFRAME, $0-0
	WORD $0xaa0003ea
	WORD $0x0cc3a03c
	WORD $0x0cc3a030
	WORD $0x710008c6
	WORD $0x2e1d1382
	WORD $0x2e1d1b83
	WORD $0x2e230042
	WORD $0x2e1d0b84
	WORD $0x2e1d2385
	WORD $0x2e250084
	WORD $0x2e1d2b81
	WORD $0x2e21039c
	WORD $0x2e111200
	WORD $0x6f56005c
	WORD $0x2e111a01
	WORD $0x2e210000
	WORD $0x2e110a01
	WORD $0x6f46409c
	WORD $0x2e112203
	WORD $0x2e230021
	WORD $0x2e112a02
	WORD $0x2e220210
	WORD $0x6f560010
	WORD $0x6f464030
	WORD $0x2f0b8f9c
	WORD $0x2f0b8e10
	WORD $0x7100089f
	WORD $0x540000a0
	WORD $0x0cc37194
	WORD $0x0cc37195
	WORD $0x2e34179c
	WORD $0x2e351610
	WORD $0x340000a5
	WORD $0x0cc27154
	WORD $0x0cc27155
	WORD $0x2e34179c
	WORD $0x2e351610
	WORD $0x0c82701c
	WORD $0x0c827010
	WORD $0x710000df
	WORD $0x54fffb2c
	WORD $0xd65f03c0

// Inputs are R0-R3 dst/src/strides, R4 mx, R5 avg, and R6 size.
TEXT ·h264QpelHAxisNEONInternal(SB), NOSPLIT|NOFRAME, $0-0
	MOVD R30, R14
	MOVD R0, R8
	MOVD R1, R9
	MOVD R6, R15
	WORD $0x52a0028d // mov w13, #0x140000
	WORD $0x728000ad // movk w13, #5
	WORD $0x4e041da6 // mov v6.s[0], w13
	MOVD R9, R12
	CMPW $3, R4
	BNE qpel_haxis_base_ready
	ADD $1, R12, R12
qpel_haxis_base_ready:
	SUB $2, R1, R1
	BL ·h264QpelH8AxisNEONInternal(SB)
	CMPW $8, R15
	BEQ qpel_haxis_done
	ADD $8, R8, R0
	ADD $6, R9, R1
	MOVD R9, R12
	CMPW $3, R4
	BNE qpel_haxis_right_base_ready
	ADD $1, R12, R12
qpel_haxis_right_base_ready:
	ADD $8, R12, R12
	MOVD R15, R6
	BL ·h264QpelH8AxisNEONInternal(SB)
qpel_haxis_done:
	RET R14
