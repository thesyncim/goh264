// SPDX-License-Identifier: LGPL-2.1-or-later
//go:build !purego && arm64

#include "textflag.h"

// h264WeightPixels16ASM mirrors FFmpeg n8.0.1
// ff_weight_h264_pixels_16_neon for even-height 8-bit weighted prediction.
// func h264WeightPixels16ASM(dst *uint8, stride int, height int32, log2Denom int32, weight int32, offset int32)
TEXT ·h264WeightPixels16ASM(SB), NOSPLIT, $0-32
	MOVD dst+0(FP), R0
	MOVD stride+8(FP), R1
	MOVW height+16(FP), R2
	MOVW log2Denom+20(FP), R3
	MOVW weight+24(FP), R4
	MOVW offset+28(FP), R5

	WORD $0x7100047f // cmp w3, #0x1
	WORD $0x52800026 // mov w6, #0x1
	WORD $0x1ac320a5 // lsl w5, w5, w3
	WORD $0x4e020cb0 // dup.8h v16, w5
	WORD $0xaa0003e5 // mov x5, x0
	WORD $0x540006cd // b.le +0xd8
	WORD $0x4b0300c6 // sub w6, w6, w3
	WORD $0x4e020cd2 // dup.8h v18, w6
	WORD $0x7100009f // cmp w4, #0x0
	WORD $0x5400032b // b.lt +0x64
	WORD $0x4e010c80 // dup.16b v0, w4
	WORD $0x71000842 // subs w2, w2, #0x2
	WORD $0x4cc17014 // ld1.16b {v20}, [x0], x1
	WORD $0x2e34c004 // umull.8h v4, v0, v20
	WORD $0x6e34c006 // umull2.8h v6, v0, v20
	WORD $0x4cc1701c // ld1.16b {v28}, [x0], x1
	WORD $0x2e3cc018 // umull.8h v24, v0, v28
	WORD $0x6e3cc01a // umull2.8h v26, v0, v28
	WORD $0x4e640604 // shadd.8h v4, v16, v4
	WORD $0x4e725484 // srshl.8h v4, v4, v18
	WORD $0x4e660606 // shadd.8h v6, v16, v6
	WORD $0x4e7254c6 // srshl.8h v6, v6, v18
	WORD $0x2e212884 // sqxtun.8b v4, v4
	WORD $0x6e2128c4 // sqxtun2.16b v4, v6
	WORD $0x4e780618 // shadd.8h v24, v16, v24
	WORD $0x4e725718 // srshl.8h v24, v24, v18
	WORD $0x4e7a061a // shadd.8h v26, v16, v26
	WORD $0x4e72575a // srshl.8h v26, v26, v18
	WORD $0x2e212b18 // sqxtun.8b v24, v24
	WORD $0x6e212b58 // sqxtun2.16b v24, v26
	WORD $0x4c8170a4 // st1.16b {v4}, [x5], x1
	WORD $0x4c8170b8 // st1.16b {v24}, [x5], x1
	WORD $0x54fffd61 // b.ne -0x54
	WORD $0xd65f03c0 // ret
	WORD $0x4b0403e4 // neg w4, w4
	WORD $0x4e010c80 // dup.16b v0, w4
	WORD $0x71000842 // subs w2, w2, #0x2
	WORD $0x4cc17014 // ld1.16b {v20}, [x0], x1
	WORD $0x2e34c004 // umull.8h v4, v0, v20
	WORD $0x6e34c006 // umull2.8h v6, v0, v20
	WORD $0x4cc1701c // ld1.16b {v28}, [x0], x1
	WORD $0x2e3cc018 // umull.8h v24, v0, v28
	WORD $0x6e3cc01a // umull2.8h v26, v0, v28
	WORD $0x4e642604 // shsub.8h v4, v16, v4
	WORD $0x4e725484 // srshl.8h v4, v4, v18
	WORD $0x4e662606 // shsub.8h v6, v16, v6
	WORD $0x4e7254c6 // srshl.8h v6, v6, v18
	WORD $0x2e212884 // sqxtun.8b v4, v4
	WORD $0x6e2128c4 // sqxtun2.16b v4, v6
	WORD $0x4e782618 // shsub.8h v24, v16, v24
	WORD $0x4e725718 // srshl.8h v24, v24, v18
	WORD $0x4e7a261a // shsub.8h v26, v16, v26
	WORD $0x4e72575a // srshl.8h v26, v26, v18
	WORD $0x2e212b18 // sqxtun.8b v24, v24
	WORD $0x6e212b58 // sqxtun2.16b v24, v26
	WORD $0x4c8170a4 // st1.16b {v4}, [x5], x1
	WORD $0x4c8170b8 // st1.16b {v24}, [x5], x1
	WORD $0x54fffd61 // b.ne -0x54
	WORD $0xd65f03c0 // ret
	WORD $0x4b0303e6 // neg w6, w3
	WORD $0x4e020cd2 // dup.8h v18, w6
	WORD $0x7100009f // cmp w4, #0x0
	WORD $0x5400032b // b.lt +0x64
	WORD $0x4e010c80 // dup.16b v0, w4
	WORD $0x71000842 // subs w2, w2, #0x2
	WORD $0x4cc17014 // ld1.16b {v20}, [x0], x1
	WORD $0x2e34c004 // umull.8h v4, v0, v20
	WORD $0x6e34c006 // umull2.8h v6, v0, v20
	WORD $0x4cc1701c // ld1.16b {v28}, [x0], x1
	WORD $0x2e3cc018 // umull.8h v24, v0, v28
	WORD $0x6e3cc01a // umull2.8h v26, v0, v28
	WORD $0x4e648604 // add.8h v4, v16, v4
	WORD $0x4e725484 // srshl.8h v4, v4, v18
	WORD $0x4e668606 // add.8h v6, v16, v6
	WORD $0x4e7254c6 // srshl.8h v6, v6, v18
	WORD $0x2e212884 // sqxtun.8b v4, v4
	WORD $0x6e2128c4 // sqxtun2.16b v4, v6
	WORD $0x4e788618 // add.8h v24, v16, v24
	WORD $0x4e725718 // srshl.8h v24, v24, v18
	WORD $0x4e7a861a // add.8h v26, v16, v26
	WORD $0x4e72575a // srshl.8h v26, v26, v18
	WORD $0x2e212b18 // sqxtun.8b v24, v24
	WORD $0x6e212b58 // sqxtun2.16b v24, v26
	WORD $0x4c8170a4 // st1.16b {v4}, [x5], x1
	WORD $0x4c8170b8 // st1.16b {v24}, [x5], x1
	WORD $0x54fffd61 // b.ne -0x54
	WORD $0xd65f03c0 // ret
	WORD $0x4b0403e4 // neg w4, w4
	WORD $0x4e010c80 // dup.16b v0, w4
	WORD $0x71000842 // subs w2, w2, #0x2
	WORD $0x4cc17014 // ld1.16b {v20}, [x0], x1
	WORD $0x2e34c004 // umull.8h v4, v0, v20
	WORD $0x6e34c006 // umull2.8h v6, v0, v20
	WORD $0x4cc1701c // ld1.16b {v28}, [x0], x1
	WORD $0x2e3cc018 // umull.8h v24, v0, v28
	WORD $0x6e3cc01a // umull2.8h v26, v0, v28
	WORD $0x6e648604 // sub.8h v4, v16, v4
	WORD $0x4e725484 // srshl.8h v4, v4, v18
	WORD $0x6e668606 // sub.8h v6, v16, v6
	WORD $0x4e7254c6 // srshl.8h v6, v6, v18
	WORD $0x2e212884 // sqxtun.8b v4, v4
	WORD $0x6e2128c4 // sqxtun2.16b v4, v6
	WORD $0x6e788618 // sub.8h v24, v16, v24
	WORD $0x4e725718 // srshl.8h v24, v24, v18
	WORD $0x6e7a861a // sub.8h v26, v16, v26
	WORD $0x4e72575a // srshl.8h v26, v26, v18
	WORD $0x2e212b18 // sqxtun.8b v24, v24
	WORD $0x6e212b58 // sqxtun2.16b v24, v26
	WORD $0x4c8170a4 // st1.16b {v4}, [x5], x1
	WORD $0x4c8170b8 // st1.16b {v24}, [x5], x1
	WORD $0x54fffd61 // b.ne -0x54
	WORD $0xd65f03c0 // ret
