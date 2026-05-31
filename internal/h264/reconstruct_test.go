// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

import (
	"errors"
	"testing"
)

func TestH264HLDecodeFrameMacroblockIntra16x16Reconstructs420(t *testing.T) {
	dst := makeH264MotionCompPicture(1, 17)
	residual := h264ReconstructResidual420()
	mbX, mbY := 1, 1
	yOff := mbY*16*dst.LumaStride + mbX*16
	before := dst.Y[yOff]

	if err := h264HLDecodeFrameMacroblock(dst, h264FrameMBReconstructInput{
		MBType:             MBTypeIntra16x16,
		MBX:                mbX,
		MBY:                mbY,
		CBP:                0x31,
		QScale:             20,
		ChromaQP:           [2]uint8{20, 21},
		ChromaPredMode:     int32(intraPred8x8Horizontal),
		Intra16x16PredMode: int8(intraPred8x8Vertical),
		PPS:                cavlcFlatQMulPPS(),
		Residual:           &residual,
	}); err != nil {
		t.Fatal(err)
	}

	if dst.Y[yOff] == before {
		t.Fatalf("luma top-left was not reconstructed, still %d", before)
	}
	if residual.MB[0] != 0 || residual.MB[16*16] != 0 || residual.MB[32*16] != 0 {
		t.Fatalf("residual blocks were not cleared after IDCT: %d/%d/%d", residual.MB[0], residual.MB[16*16], residual.MB[32*16])
	}
}

func TestH264HLDecodeFrameMacroblockInterP16x16MotionThenResidual(t *testing.T) {
	dst := makeH264MotionCompPicture(1, 9)
	ref := makeH264MotionCompPicture(1, 77)
	refs := [2][]*h264PicturePlanes{{ref}}
	var cache macroblockMotionCache
	cache.Ref[0][h264Scan8[0]] = 0
	cache.MV[0][h264Scan8[0]] = [2]int16{0, 0}
	residual := h264ReconstructResidualInter420()

	const mbX = 1
	const mbY = 1
	yOff := mbY*16*dst.LumaStride + mbX*16
	refSample := ref.Y[yOff]
	if err := h264HLDecodeFrameMacroblock(dst, h264FrameMBReconstructInput{
		MBType:        MBType16x16 | MBTypeP0L0,
		MBX:           mbX,
		MBY:           mbY,
		CBP:           0x21,
		QScale:        18,
		ChromaQP:      [2]uint8{18, 18},
		ListCount:     1,
		PPS:           cavlcFlatQMulPPS(),
		Residual:      &residual,
		Motion:        &cache,
		Refs:          refs,
		MotionScratch: makeH264MotionCompScratch(dst),
	}); err != nil {
		t.Fatal(err)
	}
	if dst.Y[yOff] == refSample {
		t.Fatalf("inter luma sample stayed at pure motion-comp value %d", refSample)
	}
}

func TestH264HLDecodeFrameMacroblockRejectsUnsupportedIntra4x4(t *testing.T) {
	dst := makeH264MotionCompPicture(1, 17)
	residual := h264ReconstructResidual420()
	err := h264HLDecodeFrameMacroblock(dst, h264FrameMBReconstructInput{
		MBType:   MBTypeIntra4x4,
		MBX:      1,
		MBY:      1,
		PPS:      cavlcFlatQMulPPS(),
		Residual: &residual,
	})
	if !errors.Is(err, ErrUnsupported) {
		t.Fatalf("err = %v, want ErrUnsupported", err)
	}
}

func h264ReconstructResidual420() cavlcResidualContext {
	var c cavlcResidualContext
	c.NonZeroCountCache[h264Scan8[lumaDCBlockIndex]] = 1
	c.MBLumaDC[0][0] = 3
	c.MBLumaDC[0][5] = -2
	c.MBLumaDC[0][10] = 4
	c.MBLumaDC[0][15] = 1

	c.NonZeroCountCache[h264Scan8[0]] = 2
	c.MB[0] = 10
	c.MB[1] = -4
	c.NonZeroCountCache[h264Scan8[5]] = 1
	c.MB[5*16] = 12

	c.NonZeroCountCache[h264Scan8[chromaDCBlockIndex+0]] = 1
	c.MB[16*16+0] = 2
	c.MB[16*16+16] = -1
	c.MB[16*16+32] = 3
	c.MB[16*16+48] = 1
	c.NonZeroCountCache[h264Scan8[16]] = 2
	c.MB[16*16+1] = 5

	c.NonZeroCountCache[h264Scan8[chromaDCBlockIndex+1]] = 1
	c.MB[32*16+0] = -2
	c.MB[32*16+16] = 4
	c.MB[32*16+32] = 1
	c.MB[32*16+48] = -3
	c.NonZeroCountCache[h264Scan8[32]] = 2
	c.MB[32*16+2] = -6
	return c
}

func h264ReconstructResidual422() cavlcResidualContext {
	c := h264ReconstructResidual420()
	c.NonZeroCountCache[h264Scan8[20]] = 1
	c.MB[20*16] = 9
	c.NonZeroCountCache[h264Scan8[36]] = 1
	c.MB[36*16] = -7
	c.MB[16*16+64] = 5
	c.MB[16*16+80] = -4
	c.MB[16*16+96] = 2
	c.MB[16*16+112] = 1
	c.MB[32*16+64] = -5
	c.MB[32*16+80] = 3
	c.MB[32*16+96] = 2
	c.MB[32*16+112] = -1
	return c
}

func h264ReconstructResidualInter420() cavlcResidualContext {
	var c cavlcResidualContext
	c.NonZeroCountCache[h264Scan8[0]] = 2
	c.MB[0] = 128
	c.MB[1] = -16
	c.NonZeroCountCache[h264Scan8[chromaDCBlockIndex+0]] = 1
	c.MB[16*16+0] = 3
	c.MB[16*16+16] = 1
	c.MB[16*16+32] = -2
	c.MB[16*16+48] = 4
	return c
}
