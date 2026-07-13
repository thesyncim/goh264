// SPDX-License-Identifier: LGPL-2.1-or-later
//go:build !purego && arm64

#include "textflag.h"

// h264ChromaMC8XYNEONInternal mirrors the pinned FFmpeg width-8 bilinear
// arithmetic while retaining independent destination and source strides.
TEXT ·h264ChromaMC8XYNEONInternal(SB), NOSPLIT|NOFRAME, $0-0
	WORD $0x0e010cc0
	WORD $0x0e010ce1
	WORD $0x0e010d02
	WORD $0x0e010d23
	WORD $0xaa0003ea
	WORD $0x0cc3a024
	WORD $0x2e050885
	WORD $0x0cc3a026
	WORD $0x2e20c090
	WORD $0x2e2180b0
	WORD $0x2e0708c7
	WORD $0x0cc3a024
	WORD $0x2e2280d0
	WORD $0x2e050885
	WORD $0x2e2380f0
	WORD $0x2e20c0d1
	WORD $0x2e2180f1
	WORD $0x2e228091
	WORD $0x2e2380b1
	WORD $0x0f0a8e10
	WORD $0x0f0a8e31
	WORD $0x340000af
	WORD $0x0cc27154
	WORD $0x0cc27155
	WORD $0x2e341610
	WORD $0x2e351631
	WORD $0x0c827010
	WORD $0x0c827011
	WORD $0x71000884
	WORD $0x54fffd4c
	WORD $0xd65f03c0

// func h264ChromaMC8XYNEONASM(dst *uint8, src *uint8, dstStride int, srcStride int, height int32, width int32, a int32, b int32, c int32, d int32, step int, avg int32)
TEXT ·h264ChromaMC8XYNEONASM(SB), NOSPLIT|NOFRAME, $0-72
	MOVD dst+0(FP), R0
	MOVD src+8(FP), R1
	MOVD dstStride+16(FP), R2
	MOVD srcStride+24(FP), R3
	MOVW height+32(FP), R4
	MOVW a+40(FP), R6
	MOVW b+44(FP), R7
	MOVW c+48(FP), R8
	MOVW d+52(FP), R9
	MOVW avg+64(FP), R15
	JMP ·h264ChromaMC8XYNEONInternal(SB)

// Width-4 and width-2 cores retain FFmpeg's NEON bilinear arithmetic, while
// accepting the independent source and destination strides used by goh264.
// R0/R1 are destination/source, R2/R3 are their strides, R4 is height,
// R6-R9 are the four bilinear weights, and R15 selects averaging.
TEXT ·h264ChromaMC4XYNEONInternal(SB), NOSPLIT|NOFRAME, $0-0
	WORD $0x0e010cc0
	WORD $0x0e010ce1
	WORD $0x0e010d02
	WORD $0x0e010d23
	WORD $0x3500022f
	WORD $0xfd400024
	WORD $0x8b030030
	WORD $0xfd400206
	WORD $0x2e040885
	WORD $0x2e0608c7
	WORD $0x2e20c090
	WORD $0x2e2180b0
	WORD $0x2e2280d0
	WORD $0x2e2380f0
	WORD $0x0f0a8e10
	WORD $0xbd000010
	WORD $0x8b020000
	WORD $0x8b030021
	WORD $0x71000484
	WORD $0x54fffe41
	WORD $0xd65f03c0
	WORD $0xfd400024
	WORD $0x8b030030
	WORD $0xfd400206
	WORD $0x2e040885
	WORD $0x2e0608c7
	WORD $0x2e20c090
	WORD $0x2e2180b0
	WORD $0x2e2280d0
	WORD $0x2e2380f0
	WORD $0x0f0a8e10
	WORD $0xbd400014
	WORD $0x2e341610
	WORD $0xbd000010
	WORD $0x8b020000
	WORD $0x8b030021
	WORD $0x71000484
	WORD $0x54fffe01
	WORD $0xd65f03c0

TEXT ·h264ChromaMC2XYNEONInternal(SB), NOSPLIT|NOFRAME, $0-0
	WORD $0x0e010cc0
	WORD $0x0e010ce1
	WORD $0x0e010d02
	WORD $0x0e010d23
	WORD $0x3500022f
	WORD $0xfd400024
	WORD $0x8b030030
	WORD $0xfd400206
	WORD $0x2e040885
	WORD $0x2e0608c7
	WORD $0x2e20c090
	WORD $0x2e2180b0
	WORD $0x2e2280d0
	WORD $0x2e2380f0
	WORD $0x0f0a8e10
	WORD $0x7d000010
	WORD $0x8b020000
	WORD $0x8b030021
	WORD $0x71000484
	WORD $0x54fffe41
	WORD $0xd65f03c0
	WORD $0xfd400024
	WORD $0x8b030030
	WORD $0xfd400206
	WORD $0x2e040885
	WORD $0x2e0608c7
	WORD $0x2e20c090
	WORD $0x2e2180b0
	WORD $0x2e2280d0
	WORD $0x2e2380f0
	WORD $0x0f0a8e10
	WORD $0x7d400014
	WORD $0x2e341610
	WORD $0x7d000010
	WORD $0x8b020000
	WORD $0x8b030021
	WORD $0x71000484
	WORD $0x54fffe01
	WORD $0xd65f03c0

// func h264ChromaMC4XYNEONASM(dst *uint8, src *uint8, dstStride int, srcStride int, height int32, width int32, a int32, b int32, c int32, d int32, step int, avg int32)
TEXT ·h264ChromaMC4XYNEONASM(SB), NOSPLIT|NOFRAME, $0-72
	MOVD dst+0(FP), R0
	MOVD src+8(FP), R1
	MOVD dstStride+16(FP), R2
	MOVD srcStride+24(FP), R3
	MOVW height+32(FP), R4
	MOVW a+40(FP), R6
	MOVW b+44(FP), R7
	MOVW c+48(FP), R8
	MOVW d+52(FP), R9
	MOVW avg+64(FP), R15
	JMP ·h264ChromaMC4XYNEONInternal(SB)

// func h264ChromaMC2XYNEONASM(dst *uint8, src *uint8, dstStride int, srcStride int, height int32, width int32, a int32, b int32, c int32, d int32, step int, avg int32)
TEXT ·h264ChromaMC2XYNEONASM(SB), NOSPLIT|NOFRAME, $0-72
	MOVD dst+0(FP), R0
	MOVD src+8(FP), R1
	MOVD dstStride+16(FP), R2
	MOVD srcStride+24(FP), R3
	MOVW height+32(FP), R4
	MOVW a+40(FP), R6
	MOVW b+44(FP), R7
	MOVW c+48(FP), R8
	MOVW d+52(FP), R9
	MOVW avg+64(FP), R15
	JMP ·h264ChromaMC2XYNEONInternal(SB)
