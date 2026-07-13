// SPDX-License-Identifier: LGPL-2.1-or-later
//go:build !purego && arm64

#include "textflag.h"

// h264IDCTAddASM is the 8-bit 4x4 inverse transform used by FFmpeg's
// ff_h264_idct_add_neon, adapted for int32 coefficient storage.
// func h264IDCTAddASM(dst *uint8, block *int32, stride int)
TEXT ·h264IDCTAddASM(SB), NOSPLIT|NOFRAME, $0-24
	MOVD dst+0(FP), R0
	MOVD block+8(FP), R1
	MOVD stride+16(FP), R2

	WORD $0x4c402820 // ld1.4s {v0, v1, v2, v3}, [x1]
	WORD $0x0e612800 // xtn.4h v0, v0
	WORD $0x0e612821 // xtn.4h v1, v1
	WORD $0x0e612842 // xtn.4h v2, v2
	WORD $0x0e612863 // xtn.4h v3, v3
	WORD $0xb9400023 // ldr w3, [x1]
	WORD $0x11008063 // add w3, w3, #32
	WORD $0x4e021c60 // ins v0.h[0], w3
	WORD $0x4f00e41e // movi.16b v30, #0
	WORD $0xad00783e // stp q30, q30, [x1]
	WORD $0xad01783e // stp q30, q30, [x1, #32]

	WORD $0x0e628404 // add.4h v4, v0, v2
	WORD $0x0f1f0430 // sshr.4h v16, v1, #1
	WORD $0x0f1f0471 // sshr.4h v17, v3, #1
	WORD $0x2e628405 // sub.4h v5, v0, v2
	WORD $0x2e638606 // sub.4h v6, v16, v3
	WORD $0x0e718427 // add.4h v7, v1, v17
	WORD $0x0e678480 // add.4h v0, v4, v7
	WORD $0x0e6684a1 // add.4h v1, v5, v6
	WORD $0x2e6684a2 // sub.4h v2, v5, v6
	WORD $0x2e678483 // sub.4h v3, v4, v7

	WORD $0x0e412804 // trn1.4h v4, v0, v1
	WORD $0x0e416805 // trn2.4h v5, v0, v1
	WORD $0x0e432846 // trn1.4h v6, v2, v3
	WORD $0x0e436847 // trn2.4h v7, v2, v3
	WORD $0x0e862880 // trn1.2s v0, v4, v6
	WORD $0x0e866882 // trn2.2s v2, v4, v6
	WORD $0x0e8728a1 // trn1.2s v1, v5, v7
	WORD $0x0e8768a3 // trn2.2s v3, v5, v7

	WORD $0x0f10a400 // sxtl.4s v0, v0
	WORD $0x0f10a421 // sxtl.4s v1, v1
	WORD $0x0f10a442 // sxtl.4s v2, v2
	WORD $0x0f10a463 // sxtl.4s v3, v3
	WORD $0x4ea28404 // add.4s v4, v0, v2
	WORD $0x4f3f0470 // sshr.4s v16, v3, #1
	WORD $0x4f3f0431 // sshr.4s v17, v1, #1
	WORD $0x6ea28405 // sub.4s v5, v0, v2
	WORD $0x4ea18606 // add.4s v6, v16, v1
	WORD $0x6ea38627 // sub.4s v7, v17, v3
	WORD $0x4ea68494 // add.4s v20, v4, v6
	WORD $0x4ea784b5 // add.4s v21, v5, v7
	WORD $0x6ea784b6 // sub.4s v22, v5, v7
	WORD $0x6ea68497 // sub.4s v23, v4, v6
	WORD $0x4f3a0694 // sshr.4s v20, v20, #6
	WORD $0x4f3a06b5 // sshr.4s v21, v21, #6
	WORD $0x4f3a06d6 // sshr.4s v22, v22, #6
	WORD $0x4f3a06f7 // sshr.4s v23, v23, #6

	WORD $0x0dc28012 // ld1.s {v18}[0], [x0], x2
	WORD $0x0dc29012 // ld1.s {v18}[1], [x0], x2
	WORD $0x0dc29013 // ld1.s {v19}[1], [x0], x2
	WORD $0x0dc28013 // ld1.s {v19}[0], [x0], x2
	WORD $0xcb020800 // sub x0, x0, x2, lsl #2
	WORD $0x2f08a658 // uxtl.8h v24, v18
	WORD $0x2f10a719 // uxtl.4s v25, v24
	WORD $0x6f10a71a // uxtl2.4s v26, v24
	WORD $0x2f08a678 // uxtl.8h v24, v19
	WORD $0x2f10a71b // uxtl.4s v27, v24
	WORD $0x6f10a71c // uxtl2.4s v28, v24
	WORD $0x4eb98694 // add.4s v20, v20, v25
	WORD $0x4eba86b5 // add.4s v21, v21, v26
	WORD $0x4ebc86d6 // add.4s v22, v22, v28
	WORD $0x4ebb86f7 // add.4s v23, v23, v27
	WORD $0x2e612a80 // sqxtun.4h v0, v20
	WORD $0x6e612aa0 // sqxtun2.8h v0, v21
	WORD $0x2e612ac1 // sqxtun.4h v1, v22
	WORD $0x6e612ae1 // sqxtun2.8h v1, v23
	WORD $0x2e212800 // sqxtun.8b v0, v0
	WORD $0x2e212821 // sqxtun.8b v1, v1
	WORD $0x0d828000 // st1.s {v0}[0], [x0], x2
	WORD $0x0d829000 // st1.s {v0}[1], [x0], x2
	WORD $0x0d828001 // st1.s {v1}[0], [x0], x2
	WORD $0x0d829001 // st1.s {v1}[1], [x0], x2
	RET

// h264IDCTDCAddASM adds one signed DC coefficient to a 4x4 byte block.
// func h264IDCTDCAddASM(dst *uint8, block *int32, stride int)
TEXT ·h264IDCTDCAddASM(SB), NOSPLIT|NOFRAME, $0-24
	MOVD dst+0(FP), R0
	MOVD block+8(FP), R1
	MOVD stride+16(FP), R2

	MOVH (R1), R3
	ADD  $32, R3, R3
	ASR  $6, R3, R3
	MOVW ZR, (R1)

	WORD $0x0dc28012 // ld1.s {v18}[0], [x0], x2
	WORD $0x0dc29012 // ld1.s {v18}[1], [x0], x2
	WORD $0x0dc29013 // ld1.s {v19}[1], [x0], x2
	WORD $0x0dc28013 // ld1.s {v19}[0], [x0], x2
	WORD $0xcb020800 // sub x0, x0, x2, lsl #2
	WORD $0x4e020c74 // dup.8h v20, w3
	WORD $0x2f08a658 // uxtl.8h v24, v18
	WORD $0x2f08a679 // uxtl.8h v25, v19
	WORD $0x4e748718 // add.8h v24, v24, v20
	WORD $0x4e748739 // add.8h v25, v25, v20
	WORD $0x2e212b00 // sqxtun.8b v0, v24
	WORD $0x2e212b21 // sqxtun.8b v1, v25
	WORD $0x0d828000 // st1.s {v0}[0], [x0], x2
	WORD $0x0d829000 // st1.s {v0}[1], [x0], x2
	WORD $0x0d829001 // st1.s {v1}[1], [x0], x2
	WORD $0x0d828001 // st1.s {v1}[0], [x0], x2
	RET
