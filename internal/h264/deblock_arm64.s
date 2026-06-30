// SPDX-License-Identifier: LGPL-2.1-or-later
//go:build !purego && arm64

#include "textflag.h"

// The deblock leaves below are raw arm64 encodings for the FFmpeg n8.0.1
// NEON kernels ff_h264_v/h_loop_filter_luma_neon and
// ff_h264_v/h_loop_filter_chroma_neon. Go's arm64 assembler does not recognize
// several required signed/saturating NEON mnemonics, so the instruction words
// are emitted directly after the Go ABI arguments are moved into the same
// x0..x4/w2..w3 registers used by the upstream ABI.

// func h264VLoopFilterLuma8ASM(pix *uint8, stride int, alpha int32, beta int32, tc0 *int8)
TEXT ·h264VLoopFilterLuma8ASM(SB), NOSPLIT, $0-32
	MOVD pix+0(FP), R0
	MOVD stride+8(FP), R1
	MOVW alpha+16(FP), R2
	MOVW beta+20(FP), R3
	MOVD tc0+24(FP), R4

	WORD $0x7100005f // cmp w2, #0x0
	WORD $0xb9400086 // ldr w6, [x4]
	WORD $0x7a401860 // ccmp w3, #0x0, #0x0, ne
	WORD $0x4e041cd8 // mov.s v24[0], w6
	WORD $0x0a0640c8 // and w8, w6, w6, lsl #16
	WORD $0x54000060 // b.eq +0x0c
	WORD $0x6a082108 // ands w8, w8, w8, lsl #8
	WORD $0x5400004a // b.ge +0x08
	WORD $0xd65f03c0 // ret
	WORD $0x4cc17000 // ld1.16b {v0}, [x0], x1
	WORD $0x4cc17002 // ld1.16b {v2}, [x0], x1
	WORD $0x4cc17004 // ld1.16b {v4}, [x0], x1
	WORD $0xcb010800 // sub x0, x0, x1, lsl #2
	WORD $0xcb010400 // sub x0, x0, x1, lsl #1
	WORD $0x4cc17014 // ld1.16b {v20}, [x0], x1
	WORD $0x4cc17012 // ld1.16b {v18}, [x0], x1
	WORD $0x4cc17010 // ld1.16b {v16}, [x0], x1
	WORD $0x4e010c56 // dup.16b v22, w2
	WORD $0x2f08a718 // ushll.8h v24, v24, #0
	WORD $0x6e207615 // uabd.16b v21, v16, v0
	WORD $0x2f10a718 // ushll.4s v24, v24, #0
	WORD $0x6e30765c // uabd.16b v28, v18, v16
	WORD $0x6f185718 // sli.8h v24, v24, #8
	WORD $0x6e20745e // uabd.16b v30, v2, v0
	WORD $0x6f305718 // sli.4s v24, v24, #16
	WORD $0x6e3536d5 // cmhi.16b v21, v22, v21
	WORD $0x4e010c76 // dup.16b v22, w3
	WORD $0x4e20ab17 // cmlt.16b v23, v24, #0
	WORD $0x6e3c36dc // cmhi.16b v28, v22, v28
	WORD $0x6e3e36de // cmhi.16b v30, v22, v30
	WORD $0x4e771eb5 // bic.16b v21, v21, v23
	WORD $0x6e307691 // uabd.16b v17, v20, v16
	WORD $0x4e3c1eb5 // and.16b v21, v21, v28
	WORD $0x6e207493 // uabd.16b v19, v4, v0
	WORD $0x4e3e1eb5 // and.16b v21, v21, v30
	WORD $0x0f0c86be // shrn.8b v30, v21, #4
	WORD $0x4e083fc7 // mov.d x7, v30[0]
	WORD $0x6e3136d1 // cmhi.16b v17, v22, v17
	WORD $0x6e3336d3 // cmhi.16b v19, v22, v19
	WORD $0xb4000667 // cbz x7, +0xcc
	WORD $0x4e351e31 // and.16b v17, v17, v21
	WORD $0x4e351e73 // and.16b v19, v19, v21
	WORD $0x4e351f18 // and.16b v24, v24, v21
	WORD $0x6e20161c // urhadd.16b v28, v16, v0
	WORD $0x6e318715 // sub.16b v21, v24, v17
	WORD $0x6e380e57 // uqadd.16b v23, v18, v24
	WORD $0x6e3c0694 // uhadd.16b v20, v20, v28
	WORD $0x6e3386b5 // sub.16b v21, v21, v19
	WORD $0x6e3c049c // uhadd.16b v28, v4, v28
	WORD $0x6e346ef7 // umin.16b v23, v23, v20
	WORD $0x6e382e56 // uqsub.16b v22, v18, v24
	WORD $0x6e380c44 // uqadd.16b v4, v2, v24
	WORD $0x6e3666f7 // umax.16b v23, v23, v22
	WORD $0x6e382c56 // uqsub.16b v22, v2, v24
	WORD $0x6e3c6c9c // umin.16b v28, v4, v28
	WORD $0x2f08a404 // ushll.8h v4, v0, #0
	WORD $0x6e36679c // umax.16b v28, v28, v22
	WORD $0x6f08a414 // ushll2.8h v20, v0, #0
	WORD $0x2e303084 // usubw.8h v4, v4, v16
	WORD $0x6e303294 // usubw2.8h v20, v20, v16
	WORD $0x4f125484 // shl.8h v4, v4, #2
	WORD $0x4f125694 // shl.8h v20, v20, #2
	WORD $0x2e321084 // uaddw.8h v4, v4, v18
	WORD $0x6e321294 // uaddw2.8h v20, v20, v18
	WORD $0x2e223084 // usubw.8h v4, v4, v2
	WORD $0x6e223294 // usubw2.8h v20, v20, v2
	WORD $0x0f0d8c84 // rshrn.8b v4, v4, #3
	WORD $0x4f0d8e84 // rshrn2.16b v4, v20, #3
	WORD $0x6e721ef1 // bsl.16b v17, v23, v18
	WORD $0x6e621f93 // bsl.16b v19, v28, v2
	WORD $0x6e20bab7 // neg.16b v23, v21
	WORD $0x2f08a61c // ushll.8h v28, v16, #0
	WORD $0x4e356c84 // smin.16b v4, v4, v21
	WORD $0x6f08a615 // ushll2.8h v21, v16, #0
	WORD $0x4e376484 // smax.16b v4, v4, v23
	WORD $0x2f08a416 // ushll.8h v22, v0, #0
	WORD $0x6f08a418 // ushll2.8h v24, v0, #0
	WORD $0x0e24139c // saddw.8h v28, v28, v4
	WORD $0x4e2412b5 // saddw2.8h v21, v21, v4
	WORD $0x0e2432d6 // ssubw.8h v22, v22, v4
	WORD $0x4e243318 // ssubw2.8h v24, v24, v4
	WORD $0x2e212b90 // sqxtun.8b v16, v28
	WORD $0x6e212ab0 // sqxtun2.16b v16, v21
	WORD $0x2e212ac0 // sqxtun.8b v0, v22
	WORD $0x6e212b00 // sqxtun2.16b v0, v24
	WORD $0xcb010400 // sub x0, x0, x1, lsl #1
	WORD $0x4c817011 // st1.16b {v17}, [x0], x1
	WORD $0x4c817010 // st1.16b {v16}, [x0], x1
	WORD $0x4c817000 // st1.16b {v0}, [x0], x1
	WORD $0x4c007013 // st1.16b {v19}, [x0]
	WORD $0xd65f03c0 // ret

// func h264HLoopFilterLuma8ASM(pix *uint8, stride int, alpha int32, beta int32, tc0 *int8)
TEXT ·h264HLoopFilterLuma8ASM(SB), NOSPLIT, $0-32
	MOVD pix+0(FP), R0
	MOVD stride+8(FP), R1
	MOVW alpha+16(FP), R2
	MOVW beta+20(FP), R3
	MOVD tc0+24(FP), R4

	WORD $0x7100005f // cmp w2, #0x0
	WORD $0xb9400086 // ldr w6, [x4]
	WORD $0x7a401860 // ccmp w3, #0x0, #0x0, ne
	WORD $0x4e041cd8 // mov.s v24[0], w6
	WORD $0x0a0640c8 // and w8, w6, w6, lsl #16
	WORD $0x54000060 // b.eq +0x0c
	WORD $0x6a082108 // ands w8, w8, w8, lsl #8
	WORD $0x5400004a // b.ge +0x08
	WORD $0xd65f03c0 // ret
	WORD $0xd1001000 // sub x0, x0, #4
	WORD $0x0cc17006 // ld1.8b {v6}, [x0], x1
	WORD $0x0cc17014 // ld1.8b {v20}, [x0], x1
	WORD $0x0cc17012 // ld1.8b {v18}, [x0], x1
	WORD $0x0cc17010 // ld1.8b {v16}, [x0], x1
	WORD $0x0cc17000 // ld1.8b {v0}, [x0], x1
	WORD $0x0cc17002 // ld1.8b {v2}, [x0], x1
	WORD $0x0cc17004 // ld1.8b {v4}, [x0], x1
	WORD $0x0cc1701a // ld1.8b {v26}, [x0], x1
	WORD $0x4dc18406 // ld1.d {v6}[1], [x0], x1
	WORD $0x4dc18414 // ld1.d {v20}[1], [x0], x1
	WORD $0x4dc18412 // ld1.d {v18}[1], [x0], x1
	WORD $0x4dc18410 // ld1.d {v16}[1], [x0], x1
	WORD $0x4dc18400 // ld1.d {v0}[1], [x0], x1
	WORD $0x4dc18402 // ld1.d {v2}[1], [x0], x1
	WORD $0x4dc18404 // ld1.d {v4}[1], [x0], x1
	WORD $0x4dc1841a // ld1.d {v26}[1], [x0], x1
	WORD $0x4e1428d5 // trn1.16b v21, v6, v20
	WORD $0x4e1468d7 // trn2.16b v23, v6, v20
	WORD $0x4e102a54 // trn1.16b v20, v18, v16
	WORD $0x4e106a50 // trn2.16b v16, v18, v16
	WORD $0x4e022806 // trn1.16b v6, v0, v2
	WORD $0x4e026802 // trn2.16b v2, v0, v2
	WORD $0x4e1a2892 // trn1.16b v18, v4, v26
	WORD $0x4e1a689a // trn2.16b v26, v4, v26
	WORD $0x4e5228c0 // trn1.8h v0, v6, v18
	WORD $0x4e5268d2 // trn2.8h v18, v6, v18
	WORD $0x4e5a2844 // trn1.8h v4, v2, v26
	WORD $0x4e5a685a // trn2.8h v26, v2, v26
	WORD $0x4e502ae2 // trn1.8h v2, v23, v16
	WORD $0x4e506af7 // trn2.8h v23, v23, v16
	WORD $0x4e542ab0 // trn1.8h v16, v21, v20
	WORD $0x4e546ab5 // trn2.8h v21, v21, v20
	WORD $0x4e802a06 // trn1.4s v6, v16, v0
	WORD $0x4e806a00 // trn2.4s v0, v16, v0
	WORD $0x4e842854 // trn1.4s v20, v2, v4
	WORD $0x4e846842 // trn2.4s v2, v2, v4
	WORD $0x4e926aa4 // trn2.4s v4, v21, v18
	WORD $0x4e922ab2 // trn1.4s v18, v21, v18
	WORD $0x4e9a2af0 // trn1.4s v16, v23, v26
	WORD $0x4e9a6afa // trn2.4s v26, v23, v26
	WORD $0x4e010c56 // dup.16b v22, w2
	WORD $0x2f08a718 // ushll.8h v24, v24, #0
	WORD $0x6e207615 // uabd.16b v21, v16, v0
	WORD $0x2f10a718 // ushll.4s v24, v24, #0
	WORD $0x6e30765c // uabd.16b v28, v18, v16
	WORD $0x6f185718 // sli.8h v24, v24, #8
	WORD $0x6e20745e // uabd.16b v30, v2, v0
	WORD $0x6f305718 // sli.4s v24, v24, #16
	WORD $0x6e3536d5 // cmhi.16b v21, v22, v21
	WORD $0x4e010c76 // dup.16b v22, w3
	WORD $0x4e20ab17 // cmlt.16b v23, v24, #0
	WORD $0x6e3c36dc // cmhi.16b v28, v22, v28
	WORD $0x6e3e36de // cmhi.16b v30, v22, v30
	WORD $0x4e771eb5 // bic.16b v21, v21, v23
	WORD $0x6e307691 // uabd.16b v17, v20, v16
	WORD $0x4e3c1eb5 // and.16b v21, v21, v28
	WORD $0x6e207493 // uabd.16b v19, v4, v0
	WORD $0x4e3e1eb5 // and.16b v21, v21, v30
	WORD $0x0f0c86be // shrn.8b v30, v21, #4
	WORD $0x4e083fc7 // mov.d x7, v30[0]
	WORD $0x6e3136d1 // cmhi.16b v17, v22, v17
	WORD $0x6e3336d3 // cmhi.16b v19, v22, v19
	WORD $0xb4000907 // cbz x7, +0x120
	WORD $0x4e351e31 // and.16b v17, v17, v21
	WORD $0x4e351e73 // and.16b v19, v19, v21
	WORD $0x4e351f18 // and.16b v24, v24, v21
	WORD $0x6e20161c // urhadd.16b v28, v16, v0
	WORD $0x6e318715 // sub.16b v21, v24, v17
	WORD $0x6e380e57 // uqadd.16b v23, v18, v24
	WORD $0x6e3c0694 // uhadd.16b v20, v20, v28
	WORD $0x6e3386b5 // sub.16b v21, v21, v19
	WORD $0x6e3c049c // uhadd.16b v28, v4, v28
	WORD $0x6e346ef7 // umin.16b v23, v23, v20
	WORD $0x6e382e56 // uqsub.16b v22, v18, v24
	WORD $0x6e380c44 // uqadd.16b v4, v2, v24
	WORD $0x6e3666f7 // umax.16b v23, v23, v22
	WORD $0x6e382c56 // uqsub.16b v22, v2, v24
	WORD $0x6e3c6c9c // umin.16b v28, v4, v28
	WORD $0x2f08a404 // ushll.8h v4, v0, #0
	WORD $0x6e36679c // umax.16b v28, v28, v22
	WORD $0x6f08a414 // ushll2.8h v20, v0, #0
	WORD $0x2e303084 // usubw.8h v4, v4, v16
	WORD $0x6e303294 // usubw2.8h v20, v20, v16
	WORD $0x4f125484 // shl.8h v4, v4, #2
	WORD $0x4f125694 // shl.8h v20, v20, #2
	WORD $0x2e321084 // uaddw.8h v4, v4, v18
	WORD $0x6e321294 // uaddw2.8h v20, v20, v18
	WORD $0x2e223084 // usubw.8h v4, v4, v2
	WORD $0x6e223294 // usubw2.8h v20, v20, v2
	WORD $0x0f0d8c84 // rshrn.8b v4, v4, #3
	WORD $0x4f0d8e84 // rshrn2.16b v4, v20, #3
	WORD $0x6e721ef1 // bsl.16b v17, v23, v18
	WORD $0x6e621f93 // bsl.16b v19, v28, v2
	WORD $0x6e20bab7 // neg.16b v23, v21
	WORD $0x2f08a61c // ushll.8h v28, v16, #0
	WORD $0x4e356c84 // smin.16b v4, v4, v21
	WORD $0x6f08a615 // ushll2.8h v21, v16, #0
	WORD $0x4e376484 // smax.16b v4, v4, v23
	WORD $0x2f08a416 // ushll.8h v22, v0, #0
	WORD $0x6f08a418 // ushll2.8h v24, v0, #0
	WORD $0x0e24139c // saddw.8h v28, v28, v4
	WORD $0x4e2412b5 // saddw2.8h v21, v21, v4
	WORD $0x0e2432d6 // ssubw.8h v22, v22, v4
	WORD $0x4e243318 // ssubw2.8h v24, v24, v4
	WORD $0x2e212b90 // sqxtun.8b v16, v28
	WORD $0x6e212ab0 // sqxtun2.16b v16, v21
	WORD $0x2e212ac0 // sqxtun.8b v0, v22
	WORD $0x6e212b00 // sqxtun2.16b v0, v24
	WORD $0x4e102a35 // trn1.16b v21, v17, v16
	WORD $0x4e106a37 // trn2.16b v23, v17, v16
	WORD $0x4e132819 // trn1.16b v25, v0, v19
	WORD $0x4e13681b // trn2.16b v27, v0, v19
	WORD $0x4e592ab1 // trn1.8h v17, v21, v25
	WORD $0x4e596aa0 // trn2.8h v0, v21, v25
	WORD $0x4e5b2af0 // trn1.8h v16, v23, v27
	WORD $0x4e5b6af3 // trn2.8h v19, v23, v27
	WORD $0xcb011000 // sub x0, x0, x1, lsl #4
	WORD $0x91000800 // add x0, x0, #2
	WORD $0x0d818011 // st1.s {v17}[0], [x0], x1
	WORD $0x0d818010 // st1.s {v16}[0], [x0], x1
	WORD $0x0d818000 // st1.s {v0}[0], [x0], x1
	WORD $0x0d818013 // st1.s {v19}[0], [x0], x1
	WORD $0x0d819011 // st1.s {v17}[1], [x0], x1
	WORD $0x0d819010 // st1.s {v16}[1], [x0], x1
	WORD $0x0d819000 // st1.s {v0}[1], [x0], x1
	WORD $0x0d819013 // st1.s {v19}[1], [x0], x1
	WORD $0x4d818011 // st1.s {v17}[2], [x0], x1
	WORD $0x4d818010 // st1.s {v16}[2], [x0], x1
	WORD $0x4d818000 // st1.s {v0}[2], [x0], x1
	WORD $0x4d818013 // st1.s {v19}[2], [x0], x1
	WORD $0x4d819011 // st1.s {v17}[3], [x0], x1
	WORD $0x4d819010 // st1.s {v16}[3], [x0], x1
	WORD $0x4d819000 // st1.s {v0}[3], [x0], x1
	WORD $0x4d819013 // st1.s {v19}[3], [x0], x1
	WORD $0xd65f03c0 // ret

// func h264VLoopFilterChroma8ASM(pix *uint8, stride int, alpha int32, beta int32, tc0 *int8)
TEXT ·h264VLoopFilterChroma8ASM(SB), NOSPLIT, $0-32
	MOVD pix+0(FP), R0
	MOVD stride+8(FP), R1
	MOVW alpha+16(FP), R2
	MOVW beta+20(FP), R3
	MOVD tc0+24(FP), R4

	WORD $0x7100005f // cmp w2, #0x0
	WORD $0xb9400086 // ldr w6, [x4]
	WORD $0x7a401860 // ccmp w3, #0x0, #0x0, ne
	WORD $0x4e041cd8 // mov.s v24[0], w6
	WORD $0x0a0640c8 // and w8, w6, w6, lsl #16
	WORD $0x54000060 // b.eq +0x0c
	WORD $0x6a082108 // ands w8, w8, w8, lsl #8
	WORD $0x5400004a // b.ge +0x08
	WORD $0xd65f03c0 // ret
	WORD $0xcb010400 // sub x0, x0, x1, lsl #1
	WORD $0x0cc17012 // ld1.8b {v18}, [x0], x1
	WORD $0x0cc17010 // ld1.8b {v16}, [x0], x1
	WORD $0x0cc17000 // ld1.8b {v0}, [x0], x1
	WORD $0x0c407002 // ld1.8b {v2}, [x0]
	WORD $0x0e010c56 // dup.8b v22, w2
	WORD $0x0e010c77 // dup.8b v23, w3
	WORD $0x2f08a718 // ushll.8h v24, v24, #0
	WORD $0x2e20761a // uabd.8b v26, v16, v0
	WORD $0x2e30765c // uabd.8b v28, v18, v16
	WORD $0x2e20745e // uabd.8b v30, v2, v0
	WORD $0x2e3a36da // cmhi.8b v26, v22, v26
	WORD $0x2e3c36fc // cmhi.8b v28, v23, v28
	WORD $0x2e3e36fe // cmhi.8b v30, v23, v30
	WORD $0x2f08a404 // ushll.8h v4, v0, #0
	WORD $0x0e3c1f5a // and.8b v26, v26, v28
	WORD $0x2e303084 // usubw.8h v4, v4, v16
	WORD $0x0e3e1f5a // and.8b v26, v26, v30
	WORD $0x4f125484 // shl.8h v4, v4, #2
	WORD $0x4e083f48 // mov.d x8, v26[0]
	WORD $0x6f185718 // sli.8h v24, v24, #8
	WORD $0x2e321084 // uaddw.8h v4, v4, v18
	WORD $0xb4000208 // cbz x8, +0x40
	WORD $0x2e223084 // usubw.8h v4, v4, v2
	WORD $0x0f0d8c84 // rshrn.8b v4, v4, #3
	WORD $0x0e386c84 // smin.8b v4, v4, v24
	WORD $0x2e20bb19 // neg.8b v25, v24
	WORD $0x0e396484 // smax.8b v4, v4, v25
	WORD $0x2f08a416 // ushll.8h v22, v0, #0
	WORD $0x0e3a1c84 // and.8b v4, v4, v26
	WORD $0x2f08a61c // ushll.8h v28, v16, #0
	WORD $0x0e24139c // saddw.8h v28, v28, v4
	WORD $0x0e2432d6 // ssubw.8h v22, v22, v4
	WORD $0x2e212b90 // sqxtun.8b v16, v28
	WORD $0x2e212ac0 // sqxtun.8b v0, v22
	WORD $0xcb010400 // sub x0, x0, x1, lsl #1
	WORD $0x0c817010 // st1.8b {v16}, [x0], x1
	WORD $0x0c817000 // st1.8b {v0}, [x0], x1
	WORD $0xd65f03c0 // ret

// func h264HLoopFilterChroma8ASM(pix *uint8, stride int, alpha int32, beta int32, tc0 *int8)
TEXT ·h264HLoopFilterChroma8ASM(SB), NOSPLIT, $0-32
	MOVD pix+0(FP), R0
	MOVD stride+8(FP), R1
	MOVW alpha+16(FP), R2
	MOVW beta+20(FP), R3
	MOVD tc0+24(FP), R4

	WORD $0x7100005f // cmp w2, #0x0
	WORD $0xb9400086 // ldr w6, [x4]
	WORD $0x7a401860 // ccmp w3, #0x0, #0x0, ne
	WORD $0x4e041cd8 // mov.s v24[0], w6
	WORD $0x0a0640c8 // and w8, w6, w6, lsl #16
	WORD $0x54000060 // b.eq +0x0c
	WORD $0x6a082108 // ands w8, w8, w8, lsl #8
	WORD $0x5400004a // b.ge +0x08
	WORD $0xd65f03c0 // ret
	WORD $0xd1000800 // sub x0, x0, #2
	WORD $0x0dc18012 // ld1.s {v18}[0], [x0], x1
	WORD $0x0dc18010 // ld1.s {v16}[0], [x0], x1
	WORD $0x0dc18000 // ld1.s {v0}[0], [x0], x1
	WORD $0x0dc18002 // ld1.s {v2}[0], [x0], x1
	WORD $0x0dc19012 // ld1.s {v18}[1], [x0], x1
	WORD $0x0dc19010 // ld1.s {v16}[1], [x0], x1
	WORD $0x0dc19000 // ld1.s {v0}[1], [x0], x1
	WORD $0x0dc19002 // ld1.s {v2}[1], [x0], x1
	WORD $0x0e102a5c // trn1.8b v28, v18, v16
	WORD $0x0e106a5d // trn2.8b v29, v18, v16
	WORD $0x0e02281e // trn1.8b v30, v0, v2
	WORD $0x0e02681f // trn2.8b v31, v0, v2
	WORD $0x0e5e2b92 // trn1.4h v18, v28, v30
	WORD $0x0e5e6b80 // trn2.4h v0, v28, v30
	WORD $0x0e5f2bb0 // trn1.4h v16, v29, v31
	WORD $0x0e5f6ba2 // trn2.4h v2, v29, v31
	WORD $0x0e010c56 // dup.8b v22, w2
	WORD $0x0e010c77 // dup.8b v23, w3
	WORD $0x2f08a718 // ushll.8h v24, v24, #0
	WORD $0x2e20761a // uabd.8b v26, v16, v0
	WORD $0x2e30765c // uabd.8b v28, v18, v16
	WORD $0x2e20745e // uabd.8b v30, v2, v0
	WORD $0x2e3a36da // cmhi.8b v26, v22, v26
	WORD $0x2e3c36fc // cmhi.8b v28, v23, v28
	WORD $0x2e3e36fe // cmhi.8b v30, v23, v30
	WORD $0x2f08a404 // ushll.8h v4, v0, #0
	WORD $0x0e3c1f5a // and.8b v26, v26, v28
	WORD $0x2e303084 // usubw.8h v4, v4, v16
	WORD $0x0e3e1f5a // and.8b v26, v26, v30
	WORD $0x4f125484 // shl.8h v4, v4, #2
	WORD $0x4e083f48 // mov.d x8, v26[0]
	WORD $0x6f185718 // sli.8h v24, v24, #8
	WORD $0x2e321084 // uaddw.8h v4, v4, v18
	WORD $0xb40003c8 // cbz x8, +0x78
	WORD $0x2e223084 // usubw.8h v4, v4, v2
	WORD $0x0f0d8c84 // rshrn.8b v4, v4, #3
	WORD $0x0e386c84 // smin.8b v4, v4, v24
	WORD $0x2e20bb19 // neg.8b v25, v24
	WORD $0x0e396484 // smax.8b v4, v4, v25
	WORD $0x2f08a416 // ushll.8h v22, v0, #0
	WORD $0x0e3a1c84 // and.8b v4, v4, v26
	WORD $0x2f08a61c // ushll.8h v28, v16, #0
	WORD $0x0e24139c // saddw.8h v28, v28, v4
	WORD $0x0e2432d6 // ssubw.8h v22, v22, v4
	WORD $0x2e212b90 // sqxtun.8b v16, v28
	WORD $0x2e212ac0 // sqxtun.8b v0, v22
	WORD $0x0e102a5c // trn1.8b v28, v18, v16
	WORD $0x0e106a5d // trn2.8b v29, v18, v16
	WORD $0x0e02281e // trn1.8b v30, v0, v2
	WORD $0x0e02681f // trn2.8b v31, v0, v2
	WORD $0x0e5e2b92 // trn1.4h v18, v28, v30
	WORD $0x0e5e6b80 // trn2.4h v0, v28, v30
	WORD $0x0e5f2bb0 // trn1.4h v16, v29, v31
	WORD $0x0e5f6ba2 // trn2.4h v2, v29, v31
	WORD $0xcb010c00 // sub x0, x0, x1, lsl #3
	WORD $0x0d818012 // st1.s {v18}[0], [x0], x1
	WORD $0x0d818010 // st1.s {v16}[0], [x0], x1
	WORD $0x0d818000 // st1.s {v0}[0], [x0], x1
	WORD $0x0d818002 // st1.s {v2}[0], [x0], x1
	WORD $0x0d819012 // st1.s {v18}[1], [x0], x1
	WORD $0x0d819010 // st1.s {v16}[1], [x0], x1
	WORD $0x0d819000 // st1.s {v0}[1], [x0], x1
	WORD $0x0d819002 // st1.s {v2}[1], [x0], x1
	WORD $0xd65f03c0 // ret

// func h264VLoopFilterChromaHigh10ASM(pix *uint8, stride int, alpha int32, beta int32, tc0 *int8)
TEXT ·h264VLoopFilterChromaHigh10ASM(SB), NOSPLIT, $0-32
	MOVD pix+0(FP), R0
	MOVD stride+8(FP), R1
	MOVW alpha+16(FP), R2
	MOVW beta+20(FP), R3
	MOVD tc0+24(FP), R4

	WORD $0x7100005f // cmp w2, #0
	WORD $0xb9400086 // ldr w6, [x4]
	WORD $0x7a401860 // ccmp w3, #0, #0, ne
	WORD $0x531e7442 // lsl w2, w2, #2
	WORD $0x4e041cd8 // mov.s v24[0], w6
	WORD $0x531e7463 // lsl w3, w3, #2
	WORD $0x0a0640c8 // and w8, w6, w6, lsl #16
	WORD $0x54000060 // b.eq +0x0c
	WORD $0x6a082108 // ands w8, w8, w8, lsl #8
	WORD $0x5400004a // b.ge +0x08
	WORD $0xd65f03c0 // ret
	WORD $0xaa0003ea // mov x10, x0
	WORD $0xcb010400 // sub x0, x0, x1, lsl #1
	WORD $0x4cc17412 // ld1.8h {v18}, [x0], x1
	WORD $0x4cc17540 // ld1.8h {v0}, [x10], x1
	WORD $0x4cc17410 // ld1.8h {v16}, [x0], x1
	WORD $0x4c407542 // ld1.8h {v2}, [x10]
	WORD $0x4e020c56 // dup.8h v22, w2
	WORD $0x4e020c77 // dup.8h v23, w3
	WORD $0x2f08a718 // ushll.8h v24, v24, #0
	WORD $0x6e60761a // uabd.8h v26, v16, v0
	WORD $0x6e70765c // uabd.8h v28, v18, v16
	WORD $0x6e60745e // uabd.8h v30, v2, v0
	WORD $0x6e7a36da // cmhi.8h v26, v22, v26
	WORD $0x6e7c36fc // cmhi.8h v28, v23, v28
	WORD $0x6e7e36fe // cmhi.8h v30, v23, v30
	WORD $0x4e3c1f5a // and.16b v26, v26, v28
	WORD $0x4ea01c04 // mov.16b v4, v0
	WORD $0x6e708484 // sub.8h v4, v4, v16
	WORD $0x4e3e1f5a // and.16b v26, v26, v30
	WORD $0x4f125484 // shl.8h v4, v4, #2
	WORD $0x4e083f48 // mov.d x8, v26[0]
	WORD $0x4e183f49 // mov.d x9, v26[1]
	WORD $0x6f185718 // sli.8h v24, v24, #8
	WORD $0x2f08a718 // ushll.8h v24, v24, #0
	WORD $0x4e728484 // add.8h v4, v4, v18
	WORD $0xab090108 // adds x8, x8, x9
	WORD $0x4f125718 // shl.8h v24, v24, #2
	WORD $0x54000280 // b.eq +0x50
	WORD $0x4f00847f // movi.8h v31, #3
	WORD $0x6e7f2f18 // uqsub.8h v24, v24, v31
	WORD $0x6e628484 // sub.8h v4, v4, v2
	WORD $0x4f1d2484 // srshr.8h v4, v4, #3
	WORD $0x4e786c84 // smin.8h v4, v4, v24
	WORD $0x6e60bb19 // neg.8h v25, v24
	WORD $0x4e796484 // smax.8h v4, v4, v25
	WORD $0x4e3a1c84 // and.16b v4, v4, v26
	WORD $0x4e648610 // add.8h v16, v16, v4
	WORD $0x6e648400 // sub.8h v0, v0, v4
	WORD $0x6f07a784 // mvni.8h v4, #0xfc, lsl #8
	WORD $0x4f008405 // movi.8h v5, #0
	WORD $0x4e646c00 // smin.8h v0, v0, v4
	WORD $0x4e646e10 // smin.8h v16, v16, v4
	WORD $0x4e656400 // smax.8h v0, v0, v5
	WORD $0x4e656610 // smax.8h v16, v16, v5
	WORD $0xcb010540 // sub x0, x10, x1, lsl #1
	WORD $0x4c817410 // st1.8h {v16}, [x0], x1
	WORD $0x4c817400 // st1.8h {v0}, [x0], x1
	WORD $0xd65f03c0 // ret

// func h264HLoopFilterChromaHigh10ASM(pix *uint8, stride int, alpha int32, beta int32, tc0 *int8)
TEXT ·h264HLoopFilterChromaHigh10ASM(SB), NOSPLIT, $0-32
	MOVD pix+0(FP), R0
	MOVD stride+8(FP), R1
	MOVW alpha+16(FP), R2
	MOVW beta+20(FP), R3
	MOVD tc0+24(FP), R4

	WORD $0x7100005f // cmp w2, #0
	WORD $0xb9400086 // ldr w6, [x4]
	WORD $0x7a401860 // ccmp w3, #0, #0, ne
	WORD $0x531e7442 // lsl w2, w2, #2
	WORD $0x4e041cd8 // mov.s v24[0], w6
	WORD $0x531e7463 // lsl w3, w3, #2
	WORD $0x0a0640c8 // and w8, w6, w6, lsl #16
	WORD $0x54000060 // b.eq +0x0c
	WORD $0x6a082108 // ands w8, w8, w8, lsl #8
	WORD $0x5400004a // b.ge +0x08
	WORD $0xd65f03c0 // ret
	WORD $0xd1001000 // sub x0, x0, #4
	WORD $0x8b01080a // add x10, x0, x1, lsl #2
	WORD $0x0dc18412 // ld1.d {v18}[0], [x0], x1
	WORD $0x4dc18552 // ld1.d {v18}[1], [x10], x1
	WORD $0x0dc18410 // ld1.d {v16}[0], [x0], x1
	WORD $0x4dc18550 // ld1.d {v16}[1], [x10], x1
	WORD $0x0dc18400 // ld1.d {v0}[0], [x0], x1
	WORD $0x4dc18540 // ld1.d {v0}[1], [x10], x1
	WORD $0x0dc18402 // ld1.d {v2}[0], [x0], x1
	WORD $0x4dc18542 // ld1.d {v2}[1], [x10], x1
	WORD $0x4e502a5c // trn1.8h v28, v18, v16
	WORD $0x4e506a5d // trn2.8h v29, v18, v16
	WORD $0x4e42281e // trn1.8h v30, v0, v2
	WORD $0x4e42681f // trn2.8h v31, v0, v2
	WORD $0x4e9e2b92 // trn1.4s v18, v28, v30
	WORD $0x4e9e6b80 // trn2.4s v0, v28, v30
	WORD $0x4e9f2bb0 // trn1.4s v16, v29, v31
	WORD $0x4e9f6ba2 // trn2.4s v2, v29, v31
	WORD $0x4e020c56 // dup.8h v22, w2
	WORD $0x4e020c77 // dup.8h v23, w3
	WORD $0x2f08a718 // ushll.8h v24, v24, #0
	WORD $0x6e60761a // uabd.8h v26, v16, v0
	WORD $0x6e70765c // uabd.8h v28, v18, v16
	WORD $0x6e60745e // uabd.8h v30, v2, v0
	WORD $0x6e7a36da // cmhi.8h v26, v22, v26
	WORD $0x6e7c36fc // cmhi.8h v28, v23, v28
	WORD $0x6e7e36fe // cmhi.8h v30, v23, v30
	WORD $0x4e3c1f5a // and.16b v26, v26, v28
	WORD $0x4ea01c04 // mov.16b v4, v0
	WORD $0x6e708484 // sub.8h v4, v4, v16
	WORD $0x4e3e1f5a // and.16b v26, v26, v30
	WORD $0x4f125484 // shl.8h v4, v4, #2
	WORD $0x4e083f48 // mov.d x8, v26[0]
	WORD $0x4e183f49 // mov.d x9, v26[1]
	WORD $0x6f185718 // sli.8h v24, v24, #8
	WORD $0x2f08a718 // ushll.8h v24, v24, #0
	WORD $0x4e728484 // add.8h v4, v4, v18
	WORD $0xab090108 // adds x8, x8, x9
	WORD $0x4f125718 // shl.8h v24, v24, #2
	WORD $0x54000440 // b.eq +0x88
	WORD $0x4f00847f // movi.8h v31, #3
	WORD $0x6e7f2f18 // uqsub.8h v24, v24, v31
	WORD $0x6e628484 // sub.8h v4, v4, v2
	WORD $0x4f1d2484 // srshr.8h v4, v4, #3
	WORD $0x4e786c84 // smin.8h v4, v4, v24
	WORD $0x6e60bb19 // neg.8h v25, v24
	WORD $0x4e796484 // smax.8h v4, v4, v25
	WORD $0x4e3a1c84 // and.16b v4, v4, v26
	WORD $0x4e648610 // add.8h v16, v16, v4
	WORD $0x6e648400 // sub.8h v0, v0, v4
	WORD $0x6f07a784 // mvni.8h v4, #0xfc, lsl #8
	WORD $0x4f008405 // movi.8h v5, #0
	WORD $0x4e646c00 // smin.8h v0, v0, v4
	WORD $0x4e646e10 // smin.8h v16, v16, v4
	WORD $0x4e656400 // smax.8h v0, v0, v5
	WORD $0x4e656610 // smax.8h v16, v16, v5
	WORD $0x4e502a5c // trn1.8h v28, v18, v16
	WORD $0x4e506a5d // trn2.8h v29, v18, v16
	WORD $0x4e42281e // trn1.8h v30, v0, v2
	WORD $0x4e42681f // trn2.8h v31, v0, v2
	WORD $0x4e9e2b92 // trn1.4s v18, v28, v30
	WORD $0x4e9e6b80 // trn2.4s v0, v28, v30
	WORD $0x4e9f2bb0 // trn1.4s v16, v29, v31
	WORD $0x4e9f6ba2 // trn2.4s v2, v29, v31
	WORD $0xcb010d40 // sub x0, x10, x1, lsl #3
	WORD $0x0d818412 // st1.d {v18}[0], [x0], x1
	WORD $0x0d818410 // st1.d {v16}[0], [x0], x1
	WORD $0x0d818400 // st1.d {v0}[0], [x0], x1
	WORD $0x0d818402 // st1.d {v2}[0], [x0], x1
	WORD $0x4d818412 // st1.d {v18}[1], [x0], x1
	WORD $0x4d818410 // st1.d {v16}[1], [x0], x1
	WORD $0x4d818400 // st1.d {v0}[1], [x0], x1
	WORD $0x4d818402 // st1.d {v2}[1], [x0], x1
	WORD $0xd65f03c0 // ret

// func h264HLoopFilterChroma422High10ASM(pix *uint8, stride int, alpha int32, beta int32, tc0 *int8)
TEXT ·h264HLoopFilterChroma422High10ASM(SB), NOSPLIT, $0-32
	MOVD pix+0(FP), R0
	MOVD stride+8(FP), R1
	MOVW alpha+16(FP), R2
	MOVW beta+20(FP), R3
	MOVD tc0+24(FP), R4

	WORD $0x7100005f // cmp w2, #0
	WORD $0xb9400086 // ldr w6, [x4]
	WORD $0x7a401860 // ccmp w3, #0, #0, ne
	WORD $0x531e7442 // lsl w2, w2, #2
	WORD $0x4e041cd8 // mov.s v24[0], w6
	WORD $0x531e7463 // lsl w3, w3, #2
	WORD $0x0a0640c8 // and w8, w6, w6, lsl #16
	WORD $0x54000060 // b.eq +0x0c
	WORD $0x6a082108 // ands w8, w8, w8, lsl #8
	WORD $0x5400004a // b.ge +0x08
	WORD $0xd65f03c0 // ret
	WORD $0x8b010005 // add x5, x0, x1
	WORD $0xd1001000 // sub x0, x0, #4
	WORD $0x8b010021 // add x1, x1, x1
	WORD $0x8b01080a // add x10, x0, x1, lsl #2
	WORD $0x0dc18412 // ld1.d {v18}[0], [x0], x1
	WORD $0x4dc18552 // ld1.d {v18}[1], [x10], x1
	WORD $0x0dc18410 // ld1.d {v16}[0], [x0], x1
	WORD $0x4dc18550 // ld1.d {v16}[1], [x10], x1
	WORD $0x0dc18400 // ld1.d {v0}[0], [x0], x1
	WORD $0x4dc18540 // ld1.d {v0}[1], [x10], x1
	WORD $0x0dc18402 // ld1.d {v2}[0], [x0], x1
	WORD $0x4dc18542 // ld1.d {v2}[1], [x10], x1
	WORD $0x4e502a5c // trn1.8h v28, v18, v16
	WORD $0x4e506a5d // trn2.8h v29, v18, v16
	WORD $0x4e42281e // trn1.8h v30, v0, v2
	WORD $0x4e42681f // trn2.8h v31, v0, v2
	WORD $0x4e9e2b92 // trn1.4s v18, v28, v30
	WORD $0x4e9e6b80 // trn2.4s v0, v28, v30
	WORD $0x4e9f2bb0 // trn1.4s v16, v29, v31
	WORD $0x4e9f6ba2 // trn2.4s v2, v29, v31
	WORD $0x4e020c56 // dup.8h v22, w2
	WORD $0x4e020c77 // dup.8h v23, w3
	WORD $0x2f08a718 // ushll.8h v24, v24, #0
	WORD $0x6e60761a // uabd.8h v26, v16, v0
	WORD $0x6e70765c // uabd.8h v28, v18, v16
	WORD $0x6e60745e // uabd.8h v30, v2, v0
	WORD $0x6e7a36da // cmhi.8h v26, v22, v26
	WORD $0x6e7c36fc // cmhi.8h v28, v23, v28
	WORD $0x6e7e36fe // cmhi.8h v30, v23, v30
	WORD $0x4e3c1f5a // and.16b v26, v26, v28
	WORD $0x4ea01c04 // mov.16b v4, v0
	WORD $0x6e708484 // sub.8h v4, v4, v16
	WORD $0x4e3e1f5a // and.16b v26, v26, v30
	WORD $0x4f125484 // shl.8h v4, v4, #2
	WORD $0x4e083f48 // mov.d x8, v26[0]
	WORD $0x4e183f49 // mov.d x9, v26[1]
	WORD $0x6f185718 // sli.8h v24, v24, #8
	WORD $0x2f08a718 // ushll.8h v24, v24, #0
	WORD $0x4e728484 // add.8h v4, v4, v18
	WORD $0xab090108 // adds x8, x8, x9
	WORD $0x4f125718 // shl.8h v24, v24, #2
	WORD $0x54000440 // b.eq +0x88
	WORD $0x4f00847f // movi.8h v31, #3
	WORD $0x6e7f2f18 // uqsub.8h v24, v24, v31
	WORD $0x6e628484 // sub.8h v4, v4, v2
	WORD $0x4f1d2484 // srshr.8h v4, v4, #3
	WORD $0x4e786c84 // smin.8h v4, v4, v24
	WORD $0x6e60bb19 // neg.8h v25, v24
	WORD $0x4e796484 // smax.8h v4, v4, v25
	WORD $0x4e3a1c84 // and.16b v4, v4, v26
	WORD $0x4e648610 // add.8h v16, v16, v4
	WORD $0x6e648400 // sub.8h v0, v0, v4
	WORD $0x6f07a784 // mvni.8h v4, #0xfc, lsl #8
	WORD $0x4f008405 // movi.8h v5, #0
	WORD $0x4e646c00 // smin.8h v0, v0, v4
	WORD $0x4e646e10 // smin.8h v16, v16, v4
	WORD $0x4e656400 // smax.8h v0, v0, v5
	WORD $0x4e656610 // smax.8h v16, v16, v5
	WORD $0x4e502a5c // trn1.8h v28, v18, v16
	WORD $0x4e506a5d // trn2.8h v29, v18, v16
	WORD $0x4e42281e // trn1.8h v30, v0, v2
	WORD $0x4e42681f // trn2.8h v31, v0, v2
	WORD $0x4e9e2b92 // trn1.4s v18, v28, v30
	WORD $0x4e9e6b80 // trn2.4s v0, v28, v30
	WORD $0x4e9f2bb0 // trn1.4s v16, v29, v31
	WORD $0x4e9f6ba2 // trn2.4s v2, v29, v31
	WORD $0xcb010d40 // sub x0, x10, x1, lsl #3
	WORD $0x0d818412 // st1.d {v18}[0], [x0], x1
	WORD $0x0d818410 // st1.d {v16}[0], [x0], x1
	WORD $0x0d818400 // st1.d {v0}[0], [x0], x1
	WORD $0x0d818402 // st1.d {v2}[0], [x0], x1
	WORD $0x4d818412 // st1.d {v18}[1], [x0], x1
	WORD $0x4d818410 // st1.d {v16}[1], [x0], x1
	WORD $0x4d818400 // st1.d {v0}[1], [x0], x1
	WORD $0x4d818402 // st1.d {v2}[1], [x0], x1
	WORD $0xd10010a0 // sub x0, x5, #4
	WORD $0x4e041cd8 // mov.s v24[0], w6
	WORD $0x8b01080a // add x10, x0, x1, lsl #2
	WORD $0x0dc18412 // ld1.d {v18}[0], [x0], x1
	WORD $0x4dc18552 // ld1.d {v18}[1], [x10], x1
	WORD $0x0dc18410 // ld1.d {v16}[0], [x0], x1
	WORD $0x4dc18550 // ld1.d {v16}[1], [x10], x1
	WORD $0x0dc18400 // ld1.d {v0}[0], [x0], x1
	WORD $0x4dc18540 // ld1.d {v0}[1], [x10], x1
	WORD $0x0dc18402 // ld1.d {v2}[0], [x0], x1
	WORD $0x4dc18542 // ld1.d {v2}[1], [x10], x1
	WORD $0x4e502a5c // trn1.8h v28, v18, v16
	WORD $0x4e506a5d // trn2.8h v29, v18, v16
	WORD $0x4e42281e // trn1.8h v30, v0, v2
	WORD $0x4e42681f // trn2.8h v31, v0, v2
	WORD $0x4e9e2b92 // trn1.4s v18, v28, v30
	WORD $0x4e9e6b80 // trn2.4s v0, v28, v30
	WORD $0x4e9f2bb0 // trn1.4s v16, v29, v31
	WORD $0x4e9f6ba2 // trn2.4s v2, v29, v31
	WORD $0x4e020c56 // dup.8h v22, w2
	WORD $0x4e020c77 // dup.8h v23, w3
	WORD $0x2f08a718 // ushll.8h v24, v24, #0
	WORD $0x6e60761a // uabd.8h v26, v16, v0
	WORD $0x6e70765c // uabd.8h v28, v18, v16
	WORD $0x6e60745e // uabd.8h v30, v2, v0
	WORD $0x6e7a36da // cmhi.8h v26, v22, v26
	WORD $0x6e7c36fc // cmhi.8h v28, v23, v28
	WORD $0x6e7e36fe // cmhi.8h v30, v23, v30
	WORD $0x4e3c1f5a // and.16b v26, v26, v28
	WORD $0x4ea01c04 // mov.16b v4, v0
	WORD $0x6e708484 // sub.8h v4, v4, v16
	WORD $0x4e3e1f5a // and.16b v26, v26, v30
	WORD $0x4f125484 // shl.8h v4, v4, #2
	WORD $0x4e083f48 // mov.d x8, v26[0]
	WORD $0x4e183f49 // mov.d x9, v26[1]
	WORD $0x6f185718 // sli.8h v24, v24, #8
	WORD $0x2f08a718 // ushll.8h v24, v24, #0
	WORD $0x4e728484 // add.8h v4, v4, v18
	WORD $0xab090108 // adds x8, x8, x9
	WORD $0x4f125718 // shl.8h v24, v24, #2
	WORD $0x54000440 // b.eq +0x88
	WORD $0x4f00847f // movi.8h v31, #3
	WORD $0x6e7f2f18 // uqsub.8h v24, v24, v31
	WORD $0x6e628484 // sub.8h v4, v4, v2
	WORD $0x4f1d2484 // srshr.8h v4, v4, #3
	WORD $0x4e786c84 // smin.8h v4, v4, v24
	WORD $0x6e60bb19 // neg.8h v25, v24
	WORD $0x4e796484 // smax.8h v4, v4, v25
	WORD $0x4e3a1c84 // and.16b v4, v4, v26
	WORD $0x4e648610 // add.8h v16, v16, v4
	WORD $0x6e648400 // sub.8h v0, v0, v4
	WORD $0x6f07a784 // mvni.8h v4, #0xfc, lsl #8
	WORD $0x4f008405 // movi.8h v5, #0
	WORD $0x4e646c00 // smin.8h v0, v0, v4
	WORD $0x4e646e10 // smin.8h v16, v16, v4
	WORD $0x4e656400 // smax.8h v0, v0, v5
	WORD $0x4e656610 // smax.8h v16, v16, v5
	WORD $0x4e502a5c // trn1.8h v28, v18, v16
	WORD $0x4e506a5d // trn2.8h v29, v18, v16
	WORD $0x4e42281e // trn1.8h v30, v0, v2
	WORD $0x4e42681f // trn2.8h v31, v0, v2
	WORD $0x4e9e2b92 // trn1.4s v18, v28, v30
	WORD $0x4e9e6b80 // trn2.4s v0, v28, v30
	WORD $0x4e9f2bb0 // trn1.4s v16, v29, v31
	WORD $0x4e9f6ba2 // trn2.4s v2, v29, v31
	WORD $0xcb010d40 // sub x0, x10, x1, lsl #3
	WORD $0x0d818412 // st1.d {v18}[0], [x0], x1
	WORD $0x0d818410 // st1.d {v16}[0], [x0], x1
	WORD $0x0d818400 // st1.d {v0}[0], [x0], x1
	WORD $0x0d818402 // st1.d {v2}[0], [x0], x1
	WORD $0x4d818412 // st1.d {v18}[1], [x0], x1
	WORD $0x4d818410 // st1.d {v16}[1], [x0], x1
	WORD $0x4d818400 // st1.d {v0}[1], [x0], x1
	WORD $0x4d818402 // st1.d {v2}[1], [x0], x1
	WORD $0xd65f03c0 // ret

// func h264VLoopFilterChromaIntraHigh10ASM(pix *uint8, stride int, alpha int32, beta int32)
TEXT ·h264VLoopFilterChromaIntraHigh10ASM(SB), NOSPLIT, $0-24
	MOVD pix+0(FP), R0
	MOVD stride+8(FP), R1
	MOVW alpha+16(FP), R2
	MOVW beta+20(FP), R3

	WORD $0x2a030044 // orr w4, w2, w3
	WORD $0x35000044 // cbnz w4, +0x08
	WORD $0xd65f03c0 // ret
	WORD $0x531e7442 // lsl w2, w2, #2
	WORD $0x531e7463 // lsl w3, w3, #2
	WORD $0x4e020c5e // dup.8h v30, w2
	WORD $0x4e020c7f // dup.8h v31, w3
	WORD $0xaa0003e9 // mov x9, x0
	WORD $0xcb010400 // sub x0, x0, x1, lsl #1
	WORD $0x4cc17412 // ld1.8h {v18}, [x0], x1
	WORD $0x4cc17531 // ld1.8h {v17}, [x9], x1
	WORD $0x4cc17410 // ld1.8h {v16}, [x0], x1
	WORD $0x4c407533 // ld1.8h {v19}, [x9]
	WORD $0x6e71761a // uabd.8h v26, v16, v17
	WORD $0x6e70765b // uabd.8h v27, v18, v16
	WORD $0x6e71767c // uabd.8h v28, v19, v17
	WORD $0x6e7a37da // cmhi.8h v26, v30, v26
	WORD $0x6e7b37fb // cmhi.8h v27, v31, v27
	WORD $0x6e7c37fc // cmhi.8h v28, v31, v28
	WORD $0x4e3b1f5a // and.16b v26, v26, v27
	WORD $0x4e3c1f5a // and.16b v26, v26, v28
	WORD $0x4e083f42 // mov.d x2, v26[0]
	WORD $0x4e183f43 // mov.d x3, v26[1]
	WORD $0x4f115644 // shl.8h v4, v18, #1
	WORD $0x4f115666 // shl.8h v6, v19, #1
	WORD $0xab030042 // adds x2, x2, x3
	WORD $0x54000180 // b.eq +0x30
	WORD $0x4e738614 // add.8h v20, v16, v19
	WORD $0x4e728636 // add.8h v22, v17, v18
	WORD $0x4e648694 // add.8h v20, v20, v4
	WORD $0x4e6686d6 // add.8h v22, v22, v6
	WORD $0x6f1e2698 // urshr.8h v24, v20, #2
	WORD $0x6f1e26d9 // urshr.8h v25, v22, #2
	WORD $0x6eba1f10 // bit.16b v16, v24, v26
	WORD $0x6eba1f31 // bit.16b v17, v25, v26
	WORD $0xcb010520 // sub x0, x9, x1, lsl #1
	WORD $0x4c817410 // st1.8h {v16}, [x0], x1
	WORD $0x4c817411 // st1.8h {v17}, [x0], x1
	WORD $0xd65f03c0 // ret

// func h264HLoopFilterChromaMBAFFIntraHigh10ASM(pix *uint8, stride int, alpha int32, beta int32)
TEXT ·h264HLoopFilterChromaMBAFFIntraHigh10ASM(SB), NOSPLIT, $0-24
	MOVD pix+0(FP), R0
	MOVD stride+8(FP), R1
	MOVW alpha+16(FP), R2
	MOVW beta+20(FP), R3

	WORD $0x2a030044 // orr w4, w2, w3
	WORD $0x35000044 // cbnz w4, +0x08
	WORD $0xd65f03c0 // ret
	WORD $0x531e7442 // lsl w2, w2, #2
	WORD $0x531e7463 // lsl w3, w3, #2
	WORD $0x4e020c5e // dup.8h v30, w2
	WORD $0x4e020c7f // dup.8h v31, w3
	WORD $0xd1001004 // sub x4, x0, #4
	WORD $0xd1000800 // sub x0, x0, #2
	WORD $0x8b010489 // add x9, x4, x1, lsl #1
	WORD $0x4cc17492 // ld1.8h {v18}, [x4], x1
	WORD $0x4cc17531 // ld1.8h {v17}, [x9], x1
	WORD $0x4cc17490 // ld1.8h {v16}, [x4], x1
	WORD $0x4cc17533 // ld1.8h {v19}, [x9], x1
	WORD $0x4e502a5a // trn1.8h v26, v18, v16
	WORD $0x4e506a5b // trn2.8h v27, v18, v16
	WORD $0x4e532a3c // trn1.8h v28, v17, v19
	WORD $0x4e536a3d // trn2.8h v29, v17, v19
	WORD $0x4e9c2b52 // trn1.4s v18, v26, v28
	WORD $0x4e9c6b51 // trn2.4s v17, v26, v28
	WORD $0x4e9d2b70 // trn1.4s v16, v27, v29
	WORD $0x4e9d6b73 // trn2.4s v19, v27, v29
	WORD $0x6e71761a // uabd.8h v26, v16, v17
	WORD $0x6e70765b // uabd.8h v27, v18, v16
	WORD $0x6e71767c // uabd.8h v28, v19, v17
	WORD $0x6e7a37da // cmhi.8h v26, v30, v26
	WORD $0x6e7b37fb // cmhi.8h v27, v31, v27
	WORD $0x6e7c37fc // cmhi.8h v28, v31, v28
	WORD $0x4e3b1f5a // and.16b v26, v26, v27
	WORD $0x4e3c1f5a // and.16b v26, v26, v28
	WORD $0x4e083f42 // mov.d x2, v26[0]
	WORD $0x4e183f43 // mov.d x3, v26[1]
	WORD $0x4f115644 // shl.8h v4, v18, #1
	WORD $0x4f115666 // shl.8h v6, v19, #1
	WORD $0xab030042 // adds x2, x2, x3
	WORD $0x540001a0 // b.eq +0x34
	WORD $0x4e738614 // add.8h v20, v16, v19
	WORD $0x4e728636 // add.8h v22, v17, v18
	WORD $0x4e648694 // add.8h v20, v20, v4
	WORD $0x4e6686d6 // add.8h v22, v22, v6
	WORD $0x6f1e2698 // urshr.8h v24, v20, #2
	WORD $0x6f1e26d9 // urshr.8h v25, v22, #2
	WORD $0x6eba1f10 // bit.16b v16, v24, v26
	WORD $0x6eba1f31 // bit.16b v17, v25, v26
	WORD $0x0da14010 // st2.h {v16,v17}[0], [x0], x1
	WORD $0x0da14810 // st2.h {v16,v17}[1], [x0], x1
	WORD $0x0da15010 // st2.h {v16,v17}[2], [x0], x1
	WORD $0x0da15810 // st2.h {v16,v17}[3], [x0], x1
	WORD $0xd65f03c0 // ret

// func h264HLoopFilterChromaIntraHigh10ASM(pix *uint8, stride int, alpha int32, beta int32)
TEXT ·h264HLoopFilterChromaIntraHigh10ASM(SB), NOSPLIT, $0-24
	MOVD pix+0(FP), R0
	MOVD stride+8(FP), R1
	MOVW alpha+16(FP), R2
	MOVW beta+20(FP), R3

	WORD $0x2a030044 // orr w4, w2, w3
	WORD $0x35000044 // cbnz w4, +0x08
	WORD $0xd65f03c0 // ret
	WORD $0x531e7442 // lsl w2, w2, #2
	WORD $0x531e7463 // lsl w3, w3, #2
	WORD $0x4e020c5e // dup.8h v30, w2
	WORD $0x4e020c7f // dup.8h v31, w3
	WORD $0xd1001004 // sub x4, x0, #4
	WORD $0xd1000800 // sub x0, x0, #2
	WORD $0x8b010889 // add x9, x4, x1, lsl #2
	WORD $0x0cc17492 // ld1.4h {v18}, [x4], x1
	WORD $0x4dc18532 // ld1.d {v18}[1], [x9], x1
	WORD $0x0cc17490 // ld1.4h {v16}, [x4], x1
	WORD $0x4dc18530 // ld1.d {v16}[1], [x9], x1
	WORD $0x0cc17491 // ld1.4h {v17}, [x4], x1
	WORD $0x4dc18531 // ld1.d {v17}[1], [x9], x1
	WORD $0x0cc17493 // ld1.4h {v19}, [x4], x1
	WORD $0x4dc18533 // ld1.d {v19}[1], [x9], x1
	WORD $0x4e502a5a // trn1.8h v26, v18, v16
	WORD $0x4e506a5b // trn2.8h v27, v18, v16
	WORD $0x4e532a3c // trn1.8h v28, v17, v19
	WORD $0x4e536a3d // trn2.8h v29, v17, v19
	WORD $0x4e9c2b52 // trn1.4s v18, v26, v28
	WORD $0x4e9c6b51 // trn2.4s v17, v26, v28
	WORD $0x4e9d2b70 // trn1.4s v16, v27, v29
	WORD $0x4e9d6b73 // trn2.4s v19, v27, v29
	WORD $0x6e71761a // uabd.8h v26, v16, v17
	WORD $0x6e70765b // uabd.8h v27, v18, v16
	WORD $0x6e71767c // uabd.8h v28, v19, v17
	WORD $0x6e7a37da // cmhi.8h v26, v30, v26
	WORD $0x6e7b37fb // cmhi.8h v27, v31, v27
	WORD $0x6e7c37fc // cmhi.8h v28, v31, v28
	WORD $0x4e3b1f5a // and.16b v26, v26, v27
	WORD $0x4e3c1f5a // and.16b v26, v26, v28
	WORD $0x4e083f42 // mov.d x2, v26[0]
	WORD $0x4e183f43 // mov.d x3, v26[1]
	WORD $0x4f115644 // shl.8h v4, v18, #1
	WORD $0x4f115666 // shl.8h v6, v19, #1
	WORD $0xab030042 // adds x2, x2, x3
	WORD $0x54000220 // b.eq +0x44
	WORD $0x4e738614 // add.8h v20, v16, v19
	WORD $0x4e728636 // add.8h v22, v17, v18
	WORD $0x4e648694 // add.8h v20, v20, v4
	WORD $0x4e6686d6 // add.8h v22, v22, v6
	WORD $0x6f1e2698 // urshr.8h v24, v20, #2
	WORD $0x6f1e26d9 // urshr.8h v25, v22, #2
	WORD $0x6eba1f10 // bit.16b v16, v24, v26
	WORD $0x6eba1f31 // bit.16b v17, v25, v26
	WORD $0x0da14010 // st2.h {v16,v17}[0], [x0], x1
	WORD $0x0da14810 // st2.h {v16,v17}[1], [x0], x1
	WORD $0x0da15010 // st2.h {v16,v17}[2], [x0], x1
	WORD $0x0da15810 // st2.h {v16,v17}[3], [x0], x1
	WORD $0x4da14010 // st2.h {v16,v17}[4], [x0], x1
	WORD $0x4da14810 // st2.h {v16,v17}[5], [x0], x1
	WORD $0x4da15010 // st2.h {v16,v17}[6], [x0], x1
	WORD $0x4da15810 // st2.h {v16,v17}[7], [x0], x1
	WORD $0xd65f03c0 // ret

// func h264HLoopFilterChroma422IntraHigh10ASM(pix *uint8, stride int, alpha int32, beta int32)
TEXT ·h264HLoopFilterChroma422IntraHigh10ASM(SB), NOSPLIT, $0-24
	MOVD pix+0(FP), R0
	MOVD stride+8(FP), R1
	MOVW alpha+16(FP), R2
	MOVW beta+20(FP), R3

	WORD $0x2a030044 // orr w4, w2, w3
	WORD $0x35000044 // cbnz w4, +0x08
	WORD $0xd65f03c0 // ret
	WORD $0x531e7442 // lsl w2, w2, #2
	WORD $0x531e7463 // lsl w3, w3, #2
	WORD $0x4e020c5e // dup.8h v30, w2
	WORD $0x4e020c7f // dup.8h v31, w3
	WORD $0xd1001004 // sub x4, x0, #4
	WORD $0x8b010c05 // add x5, x0, x1, lsl #3
	WORD $0xd1000800 // sub x0, x0, #2
	WORD $0x8b010889 // add x9, x4, x1, lsl #2
	WORD $0x0cc17492 // ld1.4h {v18}, [x4], x1
	WORD $0x4dc18532 // ld1.d {v18}[1], [x9], x1
	WORD $0x0cc17490 // ld1.4h {v16}, [x4], x1
	WORD $0x4dc18530 // ld1.d {v16}[1], [x9], x1
	WORD $0x0cc17491 // ld1.4h {v17}, [x4], x1
	WORD $0x4dc18531 // ld1.d {v17}[1], [x9], x1
	WORD $0x0cc17493 // ld1.4h {v19}, [x4], x1
	WORD $0x4dc18533 // ld1.d {v19}[1], [x9], x1
	WORD $0x4e502a5a // trn1.8h v26, v18, v16
	WORD $0x4e506a5b // trn2.8h v27, v18, v16
	WORD $0x4e532a3c // trn1.8h v28, v17, v19
	WORD $0x4e536a3d // trn2.8h v29, v17, v19
	WORD $0x4e9c2b52 // trn1.4s v18, v26, v28
	WORD $0x4e9c6b51 // trn2.4s v17, v26, v28
	WORD $0x4e9d2b70 // trn1.4s v16, v27, v29
	WORD $0x4e9d6b73 // trn2.4s v19, v27, v29
	WORD $0x6e71761a // uabd.8h v26, v16, v17
	WORD $0x6e70765b // uabd.8h v27, v18, v16
	WORD $0x6e71767c // uabd.8h v28, v19, v17
	WORD $0x6e7a37da // cmhi.8h v26, v30, v26
	WORD $0x6e7b37fb // cmhi.8h v27, v31, v27
	WORD $0x6e7c37fc // cmhi.8h v28, v31, v28
	WORD $0x4e3b1f5a // and.16b v26, v26, v27
	WORD $0x4e3c1f5a // and.16b v26, v26, v28
	WORD $0x4e083f42 // mov.d x2, v26[0]
	WORD $0x4e183f43 // mov.d x3, v26[1]
	WORD $0x4f115644 // shl.8h v4, v18, #1
	WORD $0x4f115666 // shl.8h v6, v19, #1
	WORD $0xab030042 // adds x2, x2, x3
	WORD $0x54000220 // b.eq +0x44
	WORD $0x4e738614 // add.8h v20, v16, v19
	WORD $0x4e728636 // add.8h v22, v17, v18
	WORD $0x4e648694 // add.8h v20, v20, v4
	WORD $0x4e6686d6 // add.8h v22, v22, v6
	WORD $0x6f1e2698 // urshr.8h v24, v20, #2
	WORD $0x6f1e26d9 // urshr.8h v25, v22, #2
	WORD $0x6eba1f10 // bit.16b v16, v24, v26
	WORD $0x6eba1f31 // bit.16b v17, v25, v26
	WORD $0x0da14010 // st2.h {v16,v17}[0], [x0], x1
	WORD $0x0da14810 // st2.h {v16,v17}[1], [x0], x1
	WORD $0x0da15010 // st2.h {v16,v17}[2], [x0], x1
	WORD $0x0da15810 // st2.h {v16,v17}[3], [x0], x1
	WORD $0x4da14010 // st2.h {v16,v17}[4], [x0], x1
	WORD $0x4da14810 // st2.h {v16,v17}[5], [x0], x1
	WORD $0x4da15010 // st2.h {v16,v17}[6], [x0], x1
	WORD $0x4da15810 // st2.h {v16,v17}[7], [x0], x1
	WORD $0xaa0903e4 // mov x4, x9
	WORD $0xd10008a0 // sub x0, x5, #2
	WORD $0x8b010889 // add x9, x4, x1, lsl #2
	WORD $0x0cc17492 // ld1.4h {v18}, [x4], x1
	WORD $0x4dc18532 // ld1.d {v18}[1], [x9], x1
	WORD $0x0cc17490 // ld1.4h {v16}, [x4], x1
	WORD $0x4dc18530 // ld1.d {v16}[1], [x9], x1
	WORD $0x0cc17491 // ld1.4h {v17}, [x4], x1
	WORD $0x4dc18531 // ld1.d {v17}[1], [x9], x1
	WORD $0x0cc17493 // ld1.4h {v19}, [x4], x1
	WORD $0x4dc18533 // ld1.d {v19}[1], [x9], x1
	WORD $0x4e502a5a // trn1.8h v26, v18, v16
	WORD $0x4e506a5b // trn2.8h v27, v18, v16
	WORD $0x4e532a3c // trn1.8h v28, v17, v19
	WORD $0x4e536a3d // trn2.8h v29, v17, v19
	WORD $0x4e9c2b52 // trn1.4s v18, v26, v28
	WORD $0x4e9c6b51 // trn2.4s v17, v26, v28
	WORD $0x4e9d2b70 // trn1.4s v16, v27, v29
	WORD $0x4e9d6b73 // trn2.4s v19, v27, v29
	WORD $0x6e71761a // uabd.8h v26, v16, v17
	WORD $0x6e70765b // uabd.8h v27, v18, v16
	WORD $0x6e71767c // uabd.8h v28, v19, v17
	WORD $0x6e7a37da // cmhi.8h v26, v30, v26
	WORD $0x6e7b37fb // cmhi.8h v27, v31, v27
	WORD $0x6e7c37fc // cmhi.8h v28, v31, v28
	WORD $0x4e3b1f5a // and.16b v26, v26, v27
	WORD $0x4e3c1f5a // and.16b v26, v26, v28
	WORD $0x4e083f42 // mov.d x2, v26[0]
	WORD $0x4e183f43 // mov.d x3, v26[1]
	WORD $0x4f115644 // shl.8h v4, v18, #1
	WORD $0x4f115666 // shl.8h v6, v19, #1
	WORD $0xab030042 // adds x2, x2, x3
	WORD $0x54000220 // b.eq +0x44
	WORD $0x4e738614 // add.8h v20, v16, v19
	WORD $0x4e728636 // add.8h v22, v17, v18
	WORD $0x4e648694 // add.8h v20, v20, v4
	WORD $0x4e6686d6 // add.8h v22, v22, v6
	WORD $0x6f1e2698 // urshr.8h v24, v20, #2
	WORD $0x6f1e26d9 // urshr.8h v25, v22, #2
	WORD $0x6eba1f10 // bit.16b v16, v24, v26
	WORD $0x6eba1f31 // bit.16b v17, v25, v26
	WORD $0x0da14010 // st2.h {v16,v17}[0], [x0], x1
	WORD $0x0da14810 // st2.h {v16,v17}[1], [x0], x1
	WORD $0x0da15010 // st2.h {v16,v17}[2], [x0], x1
	WORD $0x0da15810 // st2.h {v16,v17}[3], [x0], x1
	WORD $0x4da14010 // st2.h {v16,v17}[4], [x0], x1
	WORD $0x4da14810 // st2.h {v16,v17}[5], [x0], x1
	WORD $0x4da15010 // st2.h {v16,v17}[6], [x0], x1
	WORD $0x4da15810 // st2.h {v16,v17}[7], [x0], x1
	WORD $0xd65f03c0 // ret

// func h264VLoopFilterLumaIntra8ASM(pix *uint8, stride int, alpha int32, beta int32)
TEXT ·h264VLoopFilterLumaIntra8ASM(SB), NOSPLIT, $0-24
	MOVD pix+0(FP), R0
	MOVD stride+8(FP), R1
	MOVW alpha+16(FP), R2
	MOVW beta+20(FP), R3

	WORD $0x2a030044 // orr w4, w2, w3
	WORD $0x35000044 // cbnz w4, +0x08
	WORD $0xd65f03c0 // ret
	WORD $0x4e010c5e // dup.16b v30, w2
	WORD $0x4e010c7f // dup.16b v31, w3
	WORD $0x4cc17000 // ld1.16b {v0}, [x0], x1
	WORD $0x4cc17001 // ld1.16b {v1}, [x0], x1
	WORD $0x4cc17002 // ld1.16b {v2}, [x0], x1
	WORD $0x4cc17003 // ld1.16b {v3}, [x0], x1
	WORD $0xcb010c00 // sub x0, x0, x1, lsl #3
	WORD $0x4cc17004 // ld1.16b {v4}, [x0], x1
	WORD $0x4cc17005 // ld1.16b {v5}, [x0], x1
	WORD $0x4cc17006 // ld1.16b {v6}, [x0], x1
	WORD $0x4c407007 // ld1.16b {v7}, [x0]
	WORD $0x6e2074f0 // uabd.16b v16, v7, v0
	WORD $0x6e2774d1 // uabd.16b v17, v6, v7
	WORD $0x6e207432 // uabd.16b v18, v1, v0
	WORD $0x6e3037d3 // cmhi.16b v19, v30, v16
	WORD $0x6e3137f1 // cmhi.16b v17, v31, v17
	WORD $0x6e3237f2 // cmhi.16b v18, v31, v18
	WORD $0x4f00e45d // movi.16b v29, #2
	WORD $0x6f0e07de // ushr.16b v30, v30, #2
	WORD $0x4e3d87de // add.16b v30, v30, v29
	WORD $0x6e3037d0 // cmhi.16b v16, v30, v16
	WORD $0x4e311e73 // and.16b v19, v19, v17
	WORD $0x4e321e73 // and.16b v19, v19, v18
	WORD $0x0f0c8674 // shrn.8b v20, v19, #4
	WORD $0x4e083e84 // mov.d x4, v20[0]
	WORD $0xb4000b04 // cbz x4, +0x160
	WORD $0x2f09a4d4 // ushll.8h v20, v6, #1
	WORD $0x2f09a436 // ushll.8h v22, v1, #1
	WORD $0x6f09a4d5 // ushll2.8h v21, v6, #1
	WORD $0x6f09a437 // ushll2.8h v23, v1, #1
	WORD $0x2e271294 // uaddw.8h v20, v20, v7
	WORD $0x2e2012d6 // uaddw.8h v22, v22, v0
	WORD $0x6e2712b5 // uaddw2.8h v21, v21, v7
	WORD $0x6e2012f7 // uaddw2.8h v23, v23, v0
	WORD $0x2e211294 // uaddw.8h v20, v20, v1
	WORD $0x2e2612d6 // uaddw.8h v22, v22, v6
	WORD $0x6e2112b5 // uaddw2.8h v21, v21, v1
	WORD $0x6e2612f7 // uaddw2.8h v23, v23, v6
	WORD $0x0f0e8e98 // rshrn.8b v24, v20, #2
	WORD $0x0f0e8ed9 // rshrn.8b v25, v22, #2
	WORD $0x4f0e8eb8 // rshrn2.16b v24, v21, #2
	WORD $0x4f0e8ef9 // rshrn2.16b v25, v23, #2
	WORD $0x6e2774b1 // uabd.16b v17, v5, v7
	WORD $0x6e207452 // uabd.16b v18, v2, v0
	WORD $0x6e3137f1 // cmhi.16b v17, v31, v17
	WORD $0x6e3237f2 // cmhi.16b v18, v31, v18
	WORD $0x4e311e11 // and.16b v17, v16, v17
	WORD $0x4e321e12 // and.16b v18, v16, v18
	WORD $0x6e205a3e // mvn.16b v30, v17
	WORD $0x6e205a5f // mvn.16b v31, v18
	WORD $0x4e331fde // and.16b v30, v30, v19
	WORD $0x4e331fff // and.16b v31, v31, v19
	WORD $0x4e311e71 // and.16b v17, v19, v17
	WORD $0x4e321e72 // and.16b v18, v19, v18
	WORD $0x2e2700ba // uaddl.8h v26, v5, v7
	WORD $0x6e2700bb // uaddl2.8h v27, v5, v7
	WORD $0x2e20135a // uaddw.8h v26, v26, v0
	WORD $0x6e20137b // uaddw2.8h v27, v27, v0
	WORD $0x4e7a8694 // add.8h v20, v20, v26
	WORD $0x4e7b86b5 // add.8h v21, v21, v27
	WORD $0x2e201294 // uaddw.8h v20, v20, v0
	WORD $0x6e2012b5 // uaddw2.8h v21, v21, v0
	WORD $0x0f0d8e94 // rshrn.8b v20, v20, #3
	WORD $0x4f0d8eb4 // rshrn2.16b v20, v21, #3
	WORD $0x2e26135a // uaddw.8h v26, v26, v6
	WORD $0x6e26137b // uaddw2.8h v27, v27, v6
	WORD $0x0f0e8f55 // rshrn.8b v21, v26, #2
	WORD $0x4f0e8f75 // rshrn2.16b v21, v27, #2
	WORD $0x2e25009c // uaddl.8h v28, v4, v5
	WORD $0x6e25009d // uaddl2.8h v29, v4, v5
	WORD $0x4f11579c // shl.8h v28, v28, #1
	WORD $0x4f1157bd // shl.8h v29, v29, #1
	WORD $0x4e7a879c // add.8h v28, v28, v26
	WORD $0x4e7b87bd // add.8h v29, v29, v27
	WORD $0x0f0d8f93 // rshrn.8b v19, v28, #3
	WORD $0x4f0d8fb3 // rshrn2.16b v19, v29, #3
	WORD $0x2e20005a // uaddl.8h v26, v2, v0
	WORD $0x6e20005b // uaddl2.8h v27, v2, v0
	WORD $0x2e27135a // uaddw.8h v26, v26, v7
	WORD $0x6e27137b // uaddw2.8h v27, v27, v7
	WORD $0x4e7a86d6 // add.8h v22, v22, v26
	WORD $0x4e7b86f7 // add.8h v23, v23, v27
	WORD $0x2e2712d6 // uaddw.8h v22, v22, v7
	WORD $0x6e2712f7 // uaddw2.8h v23, v23, v7
	WORD $0x0f0d8ed6 // rshrn.8b v22, v22, #3
	WORD $0x4f0d8ef6 // rshrn2.16b v22, v23, #3
	WORD $0x2e21135a // uaddw.8h v26, v26, v1
	WORD $0x6e21137b // uaddw2.8h v27, v27, v1
	WORD $0x0f0e8f57 // rshrn.8b v23, v26, #2
	WORD $0x4f0e8f77 // rshrn2.16b v23, v27, #2
	WORD $0x2e23005c // uaddl.8h v28, v2, v3
	WORD $0x6e23005d // uaddl2.8h v29, v2, v3
	WORD $0x4f11579c // shl.8h v28, v28, #1
	WORD $0x4f1157bd // shl.8h v29, v29, #1
	WORD $0x4e7a879c // add.8h v28, v28, v26
	WORD $0x4e7b87bd // add.8h v29, v29, v27
	WORD $0x0f0d8f9a // rshrn.8b v26, v28, #3
	WORD $0x4f0d8fba // rshrn2.16b v26, v29, #3
	WORD $0x6ebe1f07 // bit.16b v7, v24, v30
	WORD $0x6ebf1f20 // bit.16b v0, v25, v31
	WORD $0x6eb11e87 // bit.16b v7, v20, v17
	WORD $0x6eb11ea6 // bit.16b v6, v21, v17
	WORD $0x6eb11e65 // bit.16b v5, v19, v17
	WORD $0x6eb21ec0 // bit.16b v0, v22, v18
	WORD $0x6eb21ee1 // bit.16b v1, v23, v18
	WORD $0x6eb21f42 // bit.16b v2, v26, v18
	WORD $0xcb010400 // sub x0, x0, x1, lsl #1
	WORD $0x4c817005 // st1.16b {v5}, [x0], x1
	WORD $0x4c817006 // st1.16b {v6}, [x0], x1
	WORD $0x4c817007 // st1.16b {v7}, [x0], x1
	WORD $0x4c817000 // st1.16b {v0}, [x0], x1
	WORD $0x4c817001 // st1.16b {v1}, [x0], x1
	WORD $0x4c007002 // st1.16b {v2}, [x0]
	WORD $0xd65f03c0 // ret

// func h264HLoopFilterLumaIntra8ASM(pix *uint8, stride int, alpha int32, beta int32)
TEXT ·h264HLoopFilterLumaIntra8ASM(SB), NOSPLIT, $0-24
	MOVD pix+0(FP), R0
	MOVD stride+8(FP), R1
	MOVW alpha+16(FP), R2
	MOVW beta+20(FP), R3

	WORD $0x2a030044 // orr w4, w2, w3
	WORD $0x35000044 // cbnz w4, +0x08
	WORD $0xd65f03c0 // ret
	WORD $0x4e010c5e // dup.16b v30, w2
	WORD $0x4e010c7f // dup.16b v31, w3
	WORD $0xd1001000 // sub x0, x0, #4
	WORD $0x0cc17004 // ld1.8b {v4}, [x0], x1
	WORD $0x0cc17005 // ld1.8b {v5}, [x0], x1
	WORD $0x0cc17006 // ld1.8b {v6}, [x0], x1
	WORD $0x0cc17007 // ld1.8b {v7}, [x0], x1
	WORD $0x0cc17000 // ld1.8b {v0}, [x0], x1
	WORD $0x0cc17001 // ld1.8b {v1}, [x0], x1
	WORD $0x0cc17002 // ld1.8b {v2}, [x0], x1
	WORD $0x0cc17003 // ld1.8b {v3}, [x0], x1
	WORD $0x4dc18404 // ld1.d {v4}[1], [x0], x1
	WORD $0x4dc18405 // ld1.d {v5}[1], [x0], x1
	WORD $0x4dc18406 // ld1.d {v6}[1], [x0], x1
	WORD $0x4dc18407 // ld1.d {v7}[1], [x0], x1
	WORD $0x4dc18400 // ld1.d {v0}[1], [x0], x1
	WORD $0x4dc18401 // ld1.d {v1}[1], [x0], x1
	WORD $0x4dc18402 // ld1.d {v2}[1], [x0], x1
	WORD $0x4dc18403 // ld1.d {v3}[1], [x0], x1
	WORD $0x4e052895 // trn1.16b v21, v4, v5
	WORD $0x4e056897 // trn2.16b v23, v4, v5
	WORD $0x4e0728c5 // trn1.16b v5, v6, v7
	WORD $0x4e0768c7 // trn2.16b v7, v6, v7
	WORD $0x4e012804 // trn1.16b v4, v0, v1
	WORD $0x4e016801 // trn2.16b v1, v0, v1
	WORD $0x4e032846 // trn1.16b v6, v2, v3
	WORD $0x4e036843 // trn2.16b v3, v2, v3
	WORD $0x4e462880 // trn1.8h v0, v4, v6
	WORD $0x4e466886 // trn2.8h v6, v4, v6
	WORD $0x4e432822 // trn1.8h v2, v1, v3
	WORD $0x4e436823 // trn2.8h v3, v1, v3
	WORD $0x4e472ae1 // trn1.8h v1, v23, v7
	WORD $0x4e476af7 // trn2.8h v23, v23, v7
	WORD $0x4e452aa7 // trn1.8h v7, v21, v5
	WORD $0x4e456ab5 // trn2.8h v21, v21, v5
	WORD $0x4e8028e4 // trn1.4s v4, v7, v0
	WORD $0x4e8068e0 // trn2.4s v0, v7, v0
	WORD $0x4e822825 // trn1.4s v5, v1, v2
	WORD $0x4e826821 // trn2.4s v1, v1, v2
	WORD $0x4e866aa2 // trn2.4s v2, v21, v6
	WORD $0x4e862aa6 // trn1.4s v6, v21, v6
	WORD $0x4e832ae7 // trn1.4s v7, v23, v3
	WORD $0x4e836ae3 // trn2.4s v3, v23, v3
	WORD $0x6e2074f0 // uabd.16b v16, v7, v0
	WORD $0x6e2774d1 // uabd.16b v17, v6, v7
	WORD $0x6e207432 // uabd.16b v18, v1, v0
	WORD $0x6e3037d3 // cmhi.16b v19, v30, v16
	WORD $0x6e3137f1 // cmhi.16b v17, v31, v17
	WORD $0x6e3237f2 // cmhi.16b v18, v31, v18
	WORD $0x4f00e45d // movi.16b v29, #2
	WORD $0x6f0e07de // ushr.16b v30, v30, #2
	WORD $0x4e3d87de // add.16b v30, v30, v29
	WORD $0x6e3037d0 // cmhi.16b v16, v30, v16
	WORD $0x4e311e73 // and.16b v19, v19, v17
	WORD $0x4e321e73 // and.16b v19, v19, v18
	WORD $0x0f0c8674 // shrn.8b v20, v19, #4
	WORD $0x4e083e84 // mov.d x4, v20[0]
	WORD $0xb4000f44 // cbz x4, +0x1e8
	WORD $0x2f09a4d4 // ushll.8h v20, v6, #1
	WORD $0x2f09a436 // ushll.8h v22, v1, #1
	WORD $0x6f09a4d5 // ushll2.8h v21, v6, #1
	WORD $0x6f09a437 // ushll2.8h v23, v1, #1
	WORD $0x2e271294 // uaddw.8h v20, v20, v7
	WORD $0x2e2012d6 // uaddw.8h v22, v22, v0
	WORD $0x6e2712b5 // uaddw2.8h v21, v21, v7
	WORD $0x6e2012f7 // uaddw2.8h v23, v23, v0
	WORD $0x2e211294 // uaddw.8h v20, v20, v1
	WORD $0x2e2612d6 // uaddw.8h v22, v22, v6
	WORD $0x6e2112b5 // uaddw2.8h v21, v21, v1
	WORD $0x6e2612f7 // uaddw2.8h v23, v23, v6
	WORD $0x0f0e8e98 // rshrn.8b v24, v20, #2
	WORD $0x0f0e8ed9 // rshrn.8b v25, v22, #2
	WORD $0x4f0e8eb8 // rshrn2.16b v24, v21, #2
	WORD $0x4f0e8ef9 // rshrn2.16b v25, v23, #2
	WORD $0x6e2774b1 // uabd.16b v17, v5, v7
	WORD $0x6e207452 // uabd.16b v18, v2, v0
	WORD $0x6e3137f1 // cmhi.16b v17, v31, v17
	WORD $0x6e3237f2 // cmhi.16b v18, v31, v18
	WORD $0x4e311e11 // and.16b v17, v16, v17
	WORD $0x4e321e12 // and.16b v18, v16, v18
	WORD $0x6e205a3e // mvn.16b v30, v17
	WORD $0x6e205a5f // mvn.16b v31, v18
	WORD $0x4e331fde // and.16b v30, v30, v19
	WORD $0x4e331fff // and.16b v31, v31, v19
	WORD $0x4e311e71 // and.16b v17, v19, v17
	WORD $0x4e321e72 // and.16b v18, v19, v18
	WORD $0x2e2700ba // uaddl.8h v26, v5, v7
	WORD $0x6e2700bb // uaddl2.8h v27, v5, v7
	WORD $0x2e20135a // uaddw.8h v26, v26, v0
	WORD $0x6e20137b // uaddw2.8h v27, v27, v0
	WORD $0x4e7a8694 // add.8h v20, v20, v26
	WORD $0x4e7b86b5 // add.8h v21, v21, v27
	WORD $0x2e201294 // uaddw.8h v20, v20, v0
	WORD $0x6e2012b5 // uaddw2.8h v21, v21, v0
	WORD $0x0f0d8e94 // rshrn.8b v20, v20, #3
	WORD $0x4f0d8eb4 // rshrn2.16b v20, v21, #3
	WORD $0x2e26135a // uaddw.8h v26, v26, v6
	WORD $0x6e26137b // uaddw2.8h v27, v27, v6
	WORD $0x0f0e8f55 // rshrn.8b v21, v26, #2
	WORD $0x4f0e8f75 // rshrn2.16b v21, v27, #2
	WORD $0x2e25009c // uaddl.8h v28, v4, v5
	WORD $0x6e25009d // uaddl2.8h v29, v4, v5
	WORD $0x4f11579c // shl.8h v28, v28, #1
	WORD $0x4f1157bd // shl.8h v29, v29, #1
	WORD $0x4e7a879c // add.8h v28, v28, v26
	WORD $0x4e7b87bd // add.8h v29, v29, v27
	WORD $0x0f0d8f93 // rshrn.8b v19, v28, #3
	WORD $0x4f0d8fb3 // rshrn2.16b v19, v29, #3
	WORD $0x2e20005a // uaddl.8h v26, v2, v0
	WORD $0x6e20005b // uaddl2.8h v27, v2, v0
	WORD $0x2e27135a // uaddw.8h v26, v26, v7
	WORD $0x6e27137b // uaddw2.8h v27, v27, v7
	WORD $0x4e7a86d6 // add.8h v22, v22, v26
	WORD $0x4e7b86f7 // add.8h v23, v23, v27
	WORD $0x2e2712d6 // uaddw.8h v22, v22, v7
	WORD $0x6e2712f7 // uaddw2.8h v23, v23, v7
	WORD $0x0f0d8ed6 // rshrn.8b v22, v22, #3
	WORD $0x4f0d8ef6 // rshrn2.16b v22, v23, #3
	WORD $0x2e21135a // uaddw.8h v26, v26, v1
	WORD $0x6e21137b // uaddw2.8h v27, v27, v1
	WORD $0x0f0e8f57 // rshrn.8b v23, v26, #2
	WORD $0x4f0e8f77 // rshrn2.16b v23, v27, #2
	WORD $0x2e23005c // uaddl.8h v28, v2, v3
	WORD $0x6e23005d // uaddl2.8h v29, v2, v3
	WORD $0x4f11579c // shl.8h v28, v28, #1
	WORD $0x4f1157bd // shl.8h v29, v29, #1
	WORD $0x4e7a879c // add.8h v28, v28, v26
	WORD $0x4e7b87bd // add.8h v29, v29, v27
	WORD $0x0f0d8f9a // rshrn.8b v26, v28, #3
	WORD $0x4f0d8fba // rshrn2.16b v26, v29, #3
	WORD $0x6ebe1f07 // bit.16b v7, v24, v30
	WORD $0x6ebf1f20 // bit.16b v0, v25, v31
	WORD $0x6eb11e87 // bit.16b v7, v20, v17
	WORD $0x6eb11ea6 // bit.16b v6, v21, v17
	WORD $0x6eb11e65 // bit.16b v5, v19, v17
	WORD $0x6eb21ec0 // bit.16b v0, v22, v18
	WORD $0x6eb21ee1 // bit.16b v1, v23, v18
	WORD $0x6eb21f42 // bit.16b v2, v26, v18
	WORD $0x4e052895 // trn1.16b v21, v4, v5
	WORD $0x4e056897 // trn2.16b v23, v4, v5
	WORD $0x4e0728c5 // trn1.16b v5, v6, v7
	WORD $0x4e0768c7 // trn2.16b v7, v6, v7
	WORD $0x4e012804 // trn1.16b v4, v0, v1
	WORD $0x4e016801 // trn2.16b v1, v0, v1
	WORD $0x4e032846 // trn1.16b v6, v2, v3
	WORD $0x4e036843 // trn2.16b v3, v2, v3
	WORD $0x4e462880 // trn1.8h v0, v4, v6
	WORD $0x4e466886 // trn2.8h v6, v4, v6
	WORD $0x4e432822 // trn1.8h v2, v1, v3
	WORD $0x4e436823 // trn2.8h v3, v1, v3
	WORD $0x4e472ae1 // trn1.8h v1, v23, v7
	WORD $0x4e476af7 // trn2.8h v23, v23, v7
	WORD $0x4e452aa7 // trn1.8h v7, v21, v5
	WORD $0x4e456ab5 // trn2.8h v21, v21, v5
	WORD $0x4e8028e4 // trn1.4s v4, v7, v0
	WORD $0x4e8068e0 // trn2.4s v0, v7, v0
	WORD $0x4e822825 // trn1.4s v5, v1, v2
	WORD $0x4e826821 // trn2.4s v1, v1, v2
	WORD $0x4e866aa2 // trn2.4s v2, v21, v6
	WORD $0x4e862aa6 // trn1.4s v6, v21, v6
	WORD $0x4e832ae7 // trn1.4s v7, v23, v3
	WORD $0x4e836ae3 // trn2.4s v3, v23, v3
	WORD $0xcb011000 // sub x0, x0, x1, lsl #4
	WORD $0x0c817004 // st1.8b {v4}, [x0], x1
	WORD $0x0c817005 // st1.8b {v5}, [x0], x1
	WORD $0x0c817006 // st1.8b {v6}, [x0], x1
	WORD $0x0c817007 // st1.8b {v7}, [x0], x1
	WORD $0x0c817000 // st1.8b {v0}, [x0], x1
	WORD $0x0c817001 // st1.8b {v1}, [x0], x1
	WORD $0x0c817002 // st1.8b {v2}, [x0], x1
	WORD $0x0c817003 // st1.8b {v3}, [x0], x1
	WORD $0x4d818404 // st1.d {v4}[1], [x0], x1
	WORD $0x4d818405 // st1.d {v5}[1], [x0], x1
	WORD $0x4d818406 // st1.d {v6}[1], [x0], x1
	WORD $0x4d818407 // st1.d {v7}[1], [x0], x1
	WORD $0x4d818400 // st1.d {v0}[1], [x0], x1
	WORD $0x4d818401 // st1.d {v1}[1], [x0], x1
	WORD $0x4d818402 // st1.d {v2}[1], [x0], x1
	WORD $0x4d818403 // st1.d {v3}[1], [x0], x1
	WORD $0xd65f03c0 // ret

// func h264VLoopFilterChromaIntra8ASM(pix *uint8, stride int, alpha int32, beta int32)
TEXT ·h264VLoopFilterChromaIntra8ASM(SB), NOSPLIT, $0-24
	MOVD pix+0(FP), R0
	MOVD stride+8(FP), R1
	MOVW alpha+16(FP), R2
	MOVW beta+20(FP), R3

	WORD $0x2a030044 // orr w4, w2, w3
	WORD $0x35000044 // cbnz w4, +0x08
	WORD $0xd65f03c0 // ret
	WORD $0x4e010c5e // dup.16b v30, w2
	WORD $0x4e010c7f // dup.16b v31, w3
	WORD $0xcb010400 // sub x0, x0, x1, lsl #1
	WORD $0x0cc17012 // ld1.8b {v18}, [x0], x1
	WORD $0x0cc17010 // ld1.8b {v16}, [x0], x1
	WORD $0x0cc17011 // ld1.8b {v17}, [x0], x1
	WORD $0x0c407013 // ld1.8b {v19}, [x0]
	WORD $0x2e31761a // uabd.8b v26, v16, v17
	WORD $0x2e30765b // uabd.8b v27, v18, v16
	WORD $0x2e31767c // uabd.8b v28, v19, v17
	WORD $0x2e3a37da // cmhi.8b v26, v30, v26
	WORD $0x2e3b37fb // cmhi.8b v27, v31, v27
	WORD $0x2e3c37fc // cmhi.8b v28, v31, v28
	WORD $0x0e3b1f5a // and.8b v26, v26, v27
	WORD $0x0e3c1f5a // and.8b v26, v26, v28
	WORD $0x4e083f42 // mov.d x2, v26[0]
	WORD $0x2f09a644 // ushll.8h v4, v18, #1
	WORD $0x2f09a666 // ushll.8h v6, v19, #1
	WORD $0xb4000182 // cbz x2, +0x30
	WORD $0x2e330214 // uaddl.8h v20, v16, v19
	WORD $0x2e320236 // uaddl.8h v22, v17, v18
	WORD $0x4e648694 // add.8h v20, v20, v4
	WORD $0x4e6686d6 // add.8h v22, v22, v6
	WORD $0x2f0e9e98 // uqrshrn.8b v24, v20, #2
	WORD $0x2f0e9ed9 // uqrshrn.8b v25, v22, #2
	WORD $0x2eba1f10 // bit.8b v16, v24, v26
	WORD $0x2eba1f31 // bit.8b v17, v25, v26
	WORD $0xcb010400 // sub x0, x0, x1, lsl #1
	WORD $0x0c817010 // st1.8b {v16}, [x0], x1
	WORD $0x0c817011 // st1.8b {v17}, [x0], x1
	WORD $0xd65f03c0 // ret

// func h264HLoopFilterChromaIntra8ASM(pix *uint8, stride int, alpha int32, beta int32)
TEXT ·h264HLoopFilterChromaIntra8ASM(SB), NOSPLIT, $0-24
	MOVD pix+0(FP), R0
	MOVD stride+8(FP), R1
	MOVW alpha+16(FP), R2
	MOVW beta+20(FP), R3

	WORD $0x2a030044 // orr w4, w2, w3
	WORD $0x35000044 // cbnz w4, +0x08
	WORD $0xd65f03c0 // ret
	WORD $0x4e010c5e // dup.16b v30, w2
	WORD $0x4e010c7f // dup.16b v31, w3
	WORD $0xd1000804 // sub x4, x0, #2
	WORD $0xd1000400 // sub x0, x0, #1
	WORD $0x0cc17092 // ld1.8b {v18}, [x4], x1
	WORD $0x0cc17090 // ld1.8b {v16}, [x4], x1
	WORD $0x0cc17091 // ld1.8b {v17}, [x4], x1
	WORD $0x0cc17093 // ld1.8b {v19}, [x4], x1
	WORD $0x0dc19092 // ld1.s {v18}[1], [x4], x1
	WORD $0x0dc19090 // ld1.s {v16}[1], [x4], x1
	WORD $0x0dc19091 // ld1.s {v17}[1], [x4], x1
	WORD $0x0dc19093 // ld1.s {v19}[1], [x4], x1
	WORD $0x0e102a5a // trn1.8b v26, v18, v16
	WORD $0x0e106a5b // trn2.8b v27, v18, v16
	WORD $0x0e132a3c // trn1.8b v28, v17, v19
	WORD $0x0e136a3d // trn2.8b v29, v17, v19
	WORD $0x0e5c2b52 // trn1.4h v18, v26, v28
	WORD $0x0e5c6b51 // trn2.4h v17, v26, v28
	WORD $0x0e5d2b70 // trn1.4h v16, v27, v29
	WORD $0x0e5d6b73 // trn2.4h v19, v27, v29
	WORD $0x2e31761a // uabd.8b v26, v16, v17
	WORD $0x2e30765b // uabd.8b v27, v18, v16
	WORD $0x2e31767c // uabd.8b v28, v19, v17
	WORD $0x2e3a37da // cmhi.8b v26, v30, v26
	WORD $0x2e3b37fb // cmhi.8b v27, v31, v27
	WORD $0x2e3c37fc // cmhi.8b v28, v31, v28
	WORD $0x0e3b1f5a // and.8b v26, v26, v27
	WORD $0x0e3c1f5a // and.8b v26, v26, v28
	WORD $0x4e083f42 // mov.d x2, v26[0]
	WORD $0x2f09a644 // ushll.8h v4, v18, #1
	WORD $0x2f09a666 // ushll.8h v6, v19, #1
	WORD $0xb4000222 // cbz x2, +0x44
	WORD $0x2e330214 // uaddl.8h v20, v16, v19
	WORD $0x2e320236 // uaddl.8h v22, v17, v18
	WORD $0x4e648694 // add.8h v20, v20, v4
	WORD $0x4e6686d6 // add.8h v22, v22, v6
	WORD $0x2f0e9e98 // uqrshrn.8b v24, v20, #2
	WORD $0x2f0e9ed9 // uqrshrn.8b v25, v22, #2
	WORD $0x2eba1f10 // bit.8b v16, v24, v26
	WORD $0x2eba1f31 // bit.8b v17, v25, v26
	WORD $0x0da10010 // st2.b {v16, v17}[0], [x0], x1
	WORD $0x0da10410 // st2.b {v16, v17}[1], [x0], x1
	WORD $0x0da10810 // st2.b {v16, v17}[2], [x0], x1
	WORD $0x0da10c10 // st2.b {v16, v17}[3], [x0], x1
	WORD $0x0da11010 // st2.b {v16, v17}[4], [x0], x1
	WORD $0x0da11410 // st2.b {v16, v17}[5], [x0], x1
	WORD $0x0da11810 // st2.b {v16, v17}[6], [x0], x1
	WORD $0x0da11c10 // st2.b {v16, v17}[7], [x0], x1
	WORD $0xd65f03c0 // ret

// func h264HLoopFilterChromaMBAFFIntra8ASM(pix *uint8, stride int, alpha int32, beta int32)
TEXT ·h264HLoopFilterChromaMBAFFIntra8ASM(SB), NOSPLIT, $0-24
	MOVD pix+0(FP), R0
	MOVD stride+8(FP), R1
	MOVW alpha+16(FP), R2
	MOVW beta+20(FP), R3

	WORD $0x2a030044 // orr w4, w2, w3
	WORD $0x35000044 // cbnz w4, +0x08
	WORD $0xd65f03c0 // ret
	WORD $0x4e010c5e // dup.16b v30, w2
	WORD $0x4e010c7f // dup.16b v31, w3
	WORD $0xd1000804 // sub x4, x0, #2
	WORD $0xd1000400 // sub x0, x0, #1
	WORD $0x0cc17092 // ld1.8b {v18}, [x4], x1
	WORD $0x0cc17090 // ld1.8b {v16}, [x4], x1
	WORD $0x0cc17091 // ld1.8b {v17}, [x4], x1
	WORD $0x0cc17093 // ld1.8b {v19}, [x4], x1
	WORD $0x0e102a5a // trn1.8b v26, v18, v16
	WORD $0x0e106a5b // trn2.8b v27, v18, v16
	WORD $0x0e132a3c // trn1.8b v28, v17, v19
	WORD $0x0e136a3d // trn2.8b v29, v17, v19
	WORD $0x0e5c2b52 // trn1.4h v18, v26, v28
	WORD $0x0e5c6b51 // trn2.4h v17, v26, v28
	WORD $0x0e5d2b70 // trn1.4h v16, v27, v29
	WORD $0x0e5d6b73 // trn2.4h v19, v27, v29
	WORD $0x2e31761a // uabd.8b v26, v16, v17
	WORD $0x2e30765b // uabd.8b v27, v18, v16
	WORD $0x2e31767c // uabd.8b v28, v19, v17
	WORD $0x2e3a37da // cmhi.8b v26, v30, v26
	WORD $0x2e3b37fb // cmhi.8b v27, v31, v27
	WORD $0x2e3c37fc // cmhi.8b v28, v31, v28
	WORD $0x0e3b1f5a // and.8b v26, v26, v27
	WORD $0x0e3c1f5a // and.8b v26, v26, v28
	WORD $0x4e083f42 // mov.d x2, v26[0]
	WORD $0x2f09a644 // ushll.8h v4, v18, #1
	WORD $0x2f09a666 // ushll.8h v6, v19, #1
	WORD $0xb40001a2 // cbz x2, +0x34
	WORD $0x2e330214 // uaddl.8h v20, v16, v19
	WORD $0x2e320236 // uaddl.8h v22, v17, v18
	WORD $0x4e648694 // add.8h v20, v20, v4
	WORD $0x4e6686d6 // add.8h v22, v22, v6
	WORD $0x2f0e9e98 // uqrshrn.8b v24, v20, #2
	WORD $0x2f0e9ed9 // uqrshrn.8b v25, v22, #2
	WORD $0x2eba1f10 // bit.8b v16, v24, v26
	WORD $0x2eba1f31 // bit.8b v17, v25, v26
	WORD $0x0da10010 // st2.b {v16, v17}[0], [x0], x1
	WORD $0x0da10410 // st2.b {v16, v17}[1], [x0], x1
	WORD $0x0da10810 // st2.b {v16, v17}[2], [x0], x1
	WORD $0x0da10c10 // st2.b {v16, v17}[3], [x0], x1
	WORD $0xd65f03c0 // ret

// func h264HLoopFilterChroma422Intra8ASM(pix *uint8, stride int, alpha int32, beta int32)
TEXT ·h264HLoopFilterChroma422Intra8ASM(SB), NOSPLIT, $0-24
	MOVD pix+0(FP), R0
	MOVD stride+8(FP), R1
	MOVW alpha+16(FP), R2
	MOVW beta+20(FP), R3

	WORD $0x2a030044 // orr w4, w2, w3
	WORD $0x35000044 // cbnz w4, +0x08
	WORD $0xd65f03c0 // ret
	WORD $0x4e010c5e // dup.16b v30, w2
	WORD $0x4e010c7f // dup.16b v31, w3
	WORD $0x8b010c05 // add x5, x0, x1, lsl #3
	WORD $0xd1000804 // sub x4, x0, #2
	WORD $0xd1000400 // sub x0, x0, #1
	WORD $0x0cc17092 // ld1.8b {v18}, [x4], x1
	WORD $0x0cc17090 // ld1.8b {v16}, [x4], x1
	WORD $0x0cc17091 // ld1.8b {v17}, [x4], x1
	WORD $0x0cc17093 // ld1.8b {v19}, [x4], x1
	WORD $0x0dc19092 // ld1.s {v18}[1], [x4], x1
	WORD $0x0dc19090 // ld1.s {v16}[1], [x4], x1
	WORD $0x0dc19091 // ld1.s {v17}[1], [x4], x1
	WORD $0x0dc19093 // ld1.s {v19}[1], [x4], x1
	WORD $0x0e102a5a // trn1.8b v26, v18, v16
	WORD $0x0e106a5b // trn2.8b v27, v18, v16
	WORD $0x0e132a3c // trn1.8b v28, v17, v19
	WORD $0x0e136a3d // trn2.8b v29, v17, v19
	WORD $0x0e5c2b52 // trn1.4h v18, v26, v28
	WORD $0x0e5c6b51 // trn2.4h v17, v26, v28
	WORD $0x0e5d2b70 // trn1.4h v16, v27, v29
	WORD $0x0e5d6b73 // trn2.4h v19, v27, v29
	WORD $0x2e31761a // uabd.8b v26, v16, v17
	WORD $0x2e30765b // uabd.8b v27, v18, v16
	WORD $0x2e31767c // uabd.8b v28, v19, v17
	WORD $0x2e3a37da // cmhi.8b v26, v30, v26
	WORD $0x2e3b37fb // cmhi.8b v27, v31, v27
	WORD $0x2e3c37fc // cmhi.8b v28, v31, v28
	WORD $0x0e3b1f5a // and.8b v26, v26, v27
	WORD $0x0e3c1f5a // and.8b v26, v26, v28
	WORD $0x4e083f42 // mov.d x2, v26[0]
	WORD $0x2f09a644 // ushll.8h v4, v18, #1
	WORD $0x2f09a666 // ushll.8h v6, v19, #1
	WORD $0xb4000222 // cbz x2, +0x44
	WORD $0x2e330214 // uaddl.8h v20, v16, v19
	WORD $0x2e320236 // uaddl.8h v22, v17, v18
	WORD $0x4e648694 // add.8h v20, v20, v4
	WORD $0x4e6686d6 // add.8h v22, v22, v6
	WORD $0x2f0e9e98 // uqrshrn.8b v24, v20, #2
	WORD $0x2f0e9ed9 // uqrshrn.8b v25, v22, #2
	WORD $0x2eba1f10 // bit.8b v16, v24, v26
	WORD $0x2eba1f31 // bit.8b v17, v25, v26
	WORD $0x0da10010 // st2.b {v16, v17}[0], [x0], x1
	WORD $0x0da10410 // st2.b {v16, v17}[1], [x0], x1
	WORD $0x0da10810 // st2.b {v16, v17}[2], [x0], x1
	WORD $0x0da10c10 // st2.b {v16, v17}[3], [x0], x1
	WORD $0x0da11010 // st2.b {v16, v17}[4], [x0], x1
	WORD $0x0da11410 // st2.b {v16, v17}[5], [x0], x1
	WORD $0x0da11810 // st2.b {v16, v17}[6], [x0], x1
	WORD $0x0da11c10 // st2.b {v16, v17}[7], [x0], x1
	WORD $0xd10008a4 // sub x4, x5, #2
	WORD $0xd10004a0 // sub x0, x5, #1
	WORD $0x0cc17092 // ld1.8b {v18}, [x4], x1
	WORD $0x0cc17090 // ld1.8b {v16}, [x4], x1
	WORD $0x0cc17091 // ld1.8b {v17}, [x4], x1
	WORD $0x0cc17093 // ld1.8b {v19}, [x4], x1
	WORD $0x0dc19092 // ld1.s {v18}[1], [x4], x1
	WORD $0x0dc19090 // ld1.s {v16}[1], [x4], x1
	WORD $0x0dc19091 // ld1.s {v17}[1], [x4], x1
	WORD $0x0dc19093 // ld1.s {v19}[1], [x4], x1
	WORD $0x0e102a5a // trn1.8b v26, v18, v16
	WORD $0x0e106a5b // trn2.8b v27, v18, v16
	WORD $0x0e132a3c // trn1.8b v28, v17, v19
	WORD $0x0e136a3d // trn2.8b v29, v17, v19
	WORD $0x0e5c2b52 // trn1.4h v18, v26, v28
	WORD $0x0e5c6b51 // trn2.4h v17, v26, v28
	WORD $0x0e5d2b70 // trn1.4h v16, v27, v29
	WORD $0x0e5d6b73 // trn2.4h v19, v27, v29
	WORD $0x2e31761a // uabd.8b v26, v16, v17
	WORD $0x2e30765b // uabd.8b v27, v18, v16
	WORD $0x2e31767c // uabd.8b v28, v19, v17
	WORD $0x2e3a37da // cmhi.8b v26, v30, v26
	WORD $0x2e3b37fb // cmhi.8b v27, v31, v27
	WORD $0x2e3c37fc // cmhi.8b v28, v31, v28
	WORD $0x0e3b1f5a // and.8b v26, v26, v27
	WORD $0x0e3c1f5a // and.8b v26, v26, v28
	WORD $0x4e083f42 // mov.d x2, v26[0]
	WORD $0x2f09a644 // ushll.8h v4, v18, #1
	WORD $0x2f09a666 // ushll.8h v6, v19, #1
	WORD $0xb4000222 // cbz x2, +0x44
	WORD $0x2e330214 // uaddl.8h v20, v16, v19
	WORD $0x2e320236 // uaddl.8h v22, v17, v18
	WORD $0x4e648694 // add.8h v20, v20, v4
	WORD $0x4e6686d6 // add.8h v22, v22, v6
	WORD $0x2f0e9e98 // uqrshrn.8b v24, v20, #2
	WORD $0x2f0e9ed9 // uqrshrn.8b v25, v22, #2
	WORD $0x2eba1f10 // bit.8b v16, v24, v26
	WORD $0x2eba1f31 // bit.8b v17, v25, v26
	WORD $0x0da10010 // st2.b {v16, v17}[0], [x0], x1
	WORD $0x0da10410 // st2.b {v16, v17}[1], [x0], x1
	WORD $0x0da10810 // st2.b {v16, v17}[2], [x0], x1
	WORD $0x0da10c10 // st2.b {v16, v17}[3], [x0], x1
	WORD $0x0da11010 // st2.b {v16, v17}[4], [x0], x1
	WORD $0x0da11410 // st2.b {v16, v17}[5], [x0], x1
	WORD $0x0da11810 // st2.b {v16, v17}[6], [x0], x1
	WORD $0x0da11c10 // st2.b {v16, v17}[7], [x0], x1
	WORD $0xd65f03c0 // ret

// func h264HLoopFilterChroma4228ASM(pix *uint8, stride int, alpha int32, beta int32, tc0 *int8)
TEXT ·h264HLoopFilterChroma4228ASM(SB), NOSPLIT, $0-32
	MOVD pix+0(FP), R0
	MOVD stride+8(FP), R1
	MOVW alpha+16(FP), R2
	MOVW beta+20(FP), R3
	MOVD tc0+24(FP), R4

	WORD $0x7100005f // cmp w2, #0
	WORD $0xb9400086 // ldr w6, [x4]
	WORD $0x7a401860 // ccmp w3, #0, #0, ne
	WORD $0x4e041cd8 // mov.s v24[0], w6
	WORD $0x0a0640c8 // and w8, w6, w6, lsl #16
	WORD $0x54000060 // b.eq +0x0c
	WORD $0x6a082108 // ands w8, w8, w8, lsl #8
	WORD $0x5400004a // b.ge +0x08
	WORD $0xd65f03c0 // ret
	WORD $0x8b010005 // add x5, x0, x1
	WORD $0xd1000800 // sub x0, x0, #2
	WORD $0x8b010021 // add x1, x1, x1
	WORD $0x0dc18012 // ld1.s {v18}[0], [x0], x1
	WORD $0x0dc18010 // ld1.s {v16}[0], [x0], x1
	WORD $0x0dc18000 // ld1.s {v0}[0], [x0], x1
	WORD $0x0dc18002 // ld1.s {v2}[0], [x0], x1
	WORD $0x0dc19012 // ld1.s {v18}[1], [x0], x1
	WORD $0x0dc19010 // ld1.s {v16}[1], [x0], x1
	WORD $0x0dc19000 // ld1.s {v0}[1], [x0], x1
	WORD $0x0dc19002 // ld1.s {v2}[1], [x0], x1
	WORD $0x0e102a5c // trn1.8b v28, v18, v16
	WORD $0x0e106a5d // trn2.8b v29, v18, v16
	WORD $0x0e02281e // trn1.8b v30, v0, v2
	WORD $0x0e02681f // trn2.8b v31, v0, v2
	WORD $0x0e5e2b92 // trn1.4h v18, v28, v30
	WORD $0x0e5e6b80 // trn2.4h v0, v28, v30
	WORD $0x0e5f2bb0 // trn1.4h v16, v29, v31
	WORD $0x0e5f6ba2 // trn2.4h v2, v29, v31
	WORD $0x0e010c56 // dup.8b v22, w2
	WORD $0x0e010c77 // dup.8b v23, w3
	WORD $0x2f08a718 // ushll.8h v24, v24, #0
	WORD $0x2e20761a // uabd.8b v26, v16, v0
	WORD $0x2e30765c // uabd.8b v28, v18, v16
	WORD $0x2e20745e // uabd.8b v30, v2, v0
	WORD $0x2e3a36da // cmhi.8b v26, v22, v26
	WORD $0x2e3c36fc // cmhi.8b v28, v23, v28
	WORD $0x2e3e36fe // cmhi.8b v30, v23, v30
	WORD $0x2f08a404 // ushll.8h v4, v0, #0
	WORD $0x0e3c1f5a // and.8b v26, v26, v28
	WORD $0x2e303084 // usubw.8h v4, v4, v16
	WORD $0x0e3e1f5a // and.8b v26, v26, v30
	WORD $0x4f125484 // shl.8h v4, v4, #2
	WORD $0x4e083f48 // mov.d x8, v26[0]
	WORD $0x6f185718 // sli.8h v24, v24, #8
	WORD $0x2e321084 // uaddw.8h v4, v4, v18
	WORD $0xb40003c8 // cbz x8, +0x78
	WORD $0x2e223084 // usubw.8h v4, v4, v2
	WORD $0x0f0d8c84 // rshrn.8b v4, v4, #3
	WORD $0x0e386c84 // smin.8b v4, v4, v24
	WORD $0x2e20bb19 // neg.8b v25, v24
	WORD $0x0e396484 // smax.8b v4, v4, v25
	WORD $0x2f08a416 // ushll.8h v22, v0, #0
	WORD $0x0e3a1c84 // and.8b v4, v4, v26
	WORD $0x2f08a61c // ushll.8h v28, v16, #0
	WORD $0x0e24139c // saddw.8h v28, v28, v4
	WORD $0x0e2432d6 // ssubw.8h v22, v22, v4
	WORD $0x2e212b90 // sqxtun.8b v16, v28
	WORD $0x2e212ac0 // sqxtun.8b v0, v22
	WORD $0x0e102a5c // trn1.8b v28, v18, v16
	WORD $0x0e106a5d // trn2.8b v29, v18, v16
	WORD $0x0e02281e // trn1.8b v30, v0, v2
	WORD $0x0e02681f // trn2.8b v31, v0, v2
	WORD $0x0e5e2b92 // trn1.4h v18, v28, v30
	WORD $0x0e5e6b80 // trn2.4h v0, v28, v30
	WORD $0x0e5f2bb0 // trn1.4h v16, v29, v31
	WORD $0x0e5f6ba2 // trn2.4h v2, v29, v31
	WORD $0xcb010c00 // sub x0, x0, x1, lsl #3
	WORD $0x0d818012 // st1.s {v18}[0], [x0], x1
	WORD $0x0d818010 // st1.s {v16}[0], [x0], x1
	WORD $0x0d818000 // st1.s {v0}[0], [x0], x1
	WORD $0x0d818002 // st1.s {v2}[0], [x0], x1
	WORD $0x0d819012 // st1.s {v18}[1], [x0], x1
	WORD $0x0d819010 // st1.s {v16}[1], [x0], x1
	WORD $0x0d819000 // st1.s {v0}[1], [x0], x1
	WORD $0x0d819002 // st1.s {v2}[1], [x0], x1
	WORD $0xd10008a0 // sub x0, x5, #2
	WORD $0x4e041cd8 // mov.s v24[0], w6
	WORD $0x0dc18012 // ld1.s {v18}[0], [x0], x1
	WORD $0x0dc18010 // ld1.s {v16}[0], [x0], x1
	WORD $0x0dc18000 // ld1.s {v0}[0], [x0], x1
	WORD $0x0dc18002 // ld1.s {v2}[0], [x0], x1
	WORD $0x0dc19012 // ld1.s {v18}[1], [x0], x1
	WORD $0x0dc19010 // ld1.s {v16}[1], [x0], x1
	WORD $0x0dc19000 // ld1.s {v0}[1], [x0], x1
	WORD $0x0dc19002 // ld1.s {v2}[1], [x0], x1
	WORD $0x0e102a5c // trn1.8b v28, v18, v16
	WORD $0x0e106a5d // trn2.8b v29, v18, v16
	WORD $0x0e02281e // trn1.8b v30, v0, v2
	WORD $0x0e02681f // trn2.8b v31, v0, v2
	WORD $0x0e5e2b92 // trn1.4h v18, v28, v30
	WORD $0x0e5e6b80 // trn2.4h v0, v28, v30
	WORD $0x0e5f2bb0 // trn1.4h v16, v29, v31
	WORD $0x0e5f6ba2 // trn2.4h v2, v29, v31
	WORD $0x0e010c56 // dup.8b v22, w2
	WORD $0x0e010c77 // dup.8b v23, w3
	WORD $0x2f08a718 // ushll.8h v24, v24, #0
	WORD $0x2e20761a // uabd.8b v26, v16, v0
	WORD $0x2e30765c // uabd.8b v28, v18, v16
	WORD $0x2e20745e // uabd.8b v30, v2, v0
	WORD $0x2e3a36da // cmhi.8b v26, v22, v26
	WORD $0x2e3c36fc // cmhi.8b v28, v23, v28
	WORD $0x2e3e36fe // cmhi.8b v30, v23, v30
	WORD $0x2f08a404 // ushll.8h v4, v0, #0
	WORD $0x0e3c1f5a // and.8b v26, v26, v28
	WORD $0x2e303084 // usubw.8h v4, v4, v16
	WORD $0x0e3e1f5a // and.8b v26, v26, v30
	WORD $0x4f125484 // shl.8h v4, v4, #2
	WORD $0x4e083f48 // mov.d x8, v26[0]
	WORD $0x6f185718 // sli.8h v24, v24, #8
	WORD $0x2e321084 // uaddw.8h v4, v4, v18
	WORD $0xb40003c8 // cbz x8, +0x78
	WORD $0x2e223084 // usubw.8h v4, v4, v2
	WORD $0x0f0d8c84 // rshrn.8b v4, v4, #3
	WORD $0x0e386c84 // smin.8b v4, v4, v24
	WORD $0x2e20bb19 // neg.8b v25, v24
	WORD $0x0e396484 // smax.8b v4, v4, v25
	WORD $0x2f08a416 // ushll.8h v22, v0, #0
	WORD $0x0e3a1c84 // and.8b v4, v4, v26
	WORD $0x2f08a61c // ushll.8h v28, v16, #0
	WORD $0x0e24139c // saddw.8h v28, v28, v4
	WORD $0x0e2432d6 // ssubw.8h v22, v22, v4
	WORD $0x2e212b90 // sqxtun.8b v16, v28
	WORD $0x2e212ac0 // sqxtun.8b v0, v22
	WORD $0x0e102a5c // trn1.8b v28, v18, v16
	WORD $0x0e106a5d // trn2.8b v29, v18, v16
	WORD $0x0e02281e // trn1.8b v30, v0, v2
	WORD $0x0e02681f // trn2.8b v31, v0, v2
	WORD $0x0e5e2b92 // trn1.4h v18, v28, v30
	WORD $0x0e5e6b80 // trn2.4h v0, v28, v30
	WORD $0x0e5f2bb0 // trn1.4h v16, v29, v31
	WORD $0x0e5f6ba2 // trn2.4h v2, v29, v31
	WORD $0xcb010c00 // sub x0, x0, x1, lsl #3
	WORD $0x0d818012 // st1.s {v18}[0], [x0], x1
	WORD $0x0d818010 // st1.s {v16}[0], [x0], x1
	WORD $0x0d818000 // st1.s {v0}[0], [x0], x1
	WORD $0x0d818002 // st1.s {v2}[0], [x0], x1
	WORD $0x0d819012 // st1.s {v18}[1], [x0], x1
	WORD $0x0d819010 // st1.s {v16}[1], [x0], x1
	WORD $0x0d819000 // st1.s {v0}[1], [x0], x1
	WORD $0x0d819002 // st1.s {v2}[1], [x0], x1
	WORD $0xd65f03c0 // ret
