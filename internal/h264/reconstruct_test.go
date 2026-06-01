// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

import "testing"

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

func TestH264HLDecodeFrameMacroblockIntraPCMReconstructs420(t *testing.T) {
	dst := makeH264MotionCompPicture(1, 17)
	pcm := h264ReconstructIntraPCM(1, 33)
	mbX, mbY := 1, 1

	if err := h264HLDecodeFrameMacroblock(dst, h264FrameMBReconstructInput{
		MBType:   MBTypeIntraPCM,
		MBX:      mbX,
		MBY:      mbY,
		IntraPCM: pcm,
	}); err != nil {
		t.Fatal(err)
	}

	yOff, cbOff, crOff, err := h264MBDestPartOffsets(dst, mbX, mbY, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	assertH264Rows(t, "pcm y", dst.Y, yOff, dst.LumaStride, 16, 16, pcm, 16)
	assertH264Rows(t, "pcm cb", dst.Cb, cbOff, dst.ChromaStride, 8, 8, pcm[256:], 8)
	assertH264Rows(t, "pcm cr", dst.Cr, crOff, dst.ChromaStride, 8, 8, pcm[256+8*8:], 8)
}

func TestH264HLDecodeFrameMacroblockIntraPCMReconstructs422(t *testing.T) {
	dst := makeH264MotionCompPicture(2, 21)
	pcm := h264ReconstructIntraPCM(2, 49)
	mbX, mbY := 1, 1

	if err := h264HLDecodeFrameMacroblock(dst, h264FrameMBReconstructInput{
		MBType:   MBTypeIntraPCM,
		MBX:      mbX,
		MBY:      mbY,
		IntraPCM: pcm,
	}); err != nil {
		t.Fatal(err)
	}

	yOff, cbOff, crOff, err := h264MBDestPartOffsets(dst, mbX, mbY, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	assertH264Rows(t, "pcm y", dst.Y, yOff, dst.LumaStride, 16, 16, pcm, 16)
	assertH264Rows(t, "pcm cb", dst.Cb, cbOff, dst.ChromaStride, 8, 16, pcm[256:], 8)
	assertH264Rows(t, "pcm cr", dst.Cr, crOff, dst.ChromaStride, 8, 16, pcm[256+16*8:], 8)
}

func TestH264HLDecodeFrameIntraPCMHighReconstructs420(t *testing.T) {
	const bitDepth = 10
	dst := makeH264ReconstructHighPicture(1, 17)
	pcm := h264ReconstructIntraPCMHigh(1, bitDepth, 33)
	samples := h264ReconstructIntraPCMSamples(1, bitDepth, 33)
	mbX, mbY := 1, 1
	yOff, cbOff, crOff, err := h264MBDestPartOffsetsHigh(dst, mbX, mbY, 0, 0)
	if err != nil {
		t.Fatal(err)
	}

	if err := h264HLDecodeFrameIntraPCMHigh(dst, yOff, cbOff, crOff, pcm, bitDepth); err != nil {
		t.Fatal(err)
	}

	assertH264RowsHigh(t, "high pcm y", dst.Y, yOff, dst.LumaStride, 16, 16, samples, 16)
	assertH264RowsHigh(t, "high pcm cb", dst.Cb, cbOff, dst.ChromaStride, 8, 8, samples[256:], 8)
	assertH264RowsHigh(t, "high pcm cr", dst.Cr, crOff, dst.ChromaStride, 8, 8, samples[256+8*8:], 8)
}

func TestH264HLDecodeFrameIntraPCMHighReconstructs444(t *testing.T) {
	const bitDepth = 14
	dst := makeH264ReconstructHighPicture(3, 23)
	pcm := h264ReconstructIntraPCMHigh(3, bitDepth, 57)
	samples := h264ReconstructIntraPCMSamples(3, bitDepth, 57)
	mbX, mbY := 1, 1
	yOff, cbOff, crOff, err := h264MBDestPartOffsetsHigh(dst, mbX, mbY, 0, 0)
	if err != nil {
		t.Fatal(err)
	}

	if err := h264HLDecodeFrameIntraPCMHigh(dst, yOff, cbOff, crOff, pcm, bitDepth); err != nil {
		t.Fatal(err)
	}

	assertH264RowsHigh(t, "high pcm444 y", dst.Y, yOff, dst.LumaStride, 16, 16, samples, 16)
	assertH264RowsHigh(t, "high pcm444 cb", dst.Cb, cbOff, dst.ChromaStride, 16, 16, samples[256:], 16)
	assertH264RowsHigh(t, "high pcm444 cr", dst.Cr, crOff, dst.ChromaStride, 16, 16, samples[512:], 16)
	if err := h264HLDecodeFrameIntraPCMHigh(dst, yOff, cbOff, crOff, pcm[:len(pcm)-1], bitDepth); err != ErrInvalidData {
		t.Fatalf("truncated high pcm error = %v, want ErrInvalidData", err)
	}
}

func TestH264HLDecodeFrameMacroblockIntra16x16Reconstructs444PaddedChromaStride(t *testing.T) {
	dst := makeH264MotionCompPicture(3, 23)
	_, chromaHeight := h264ChromaFrameSize(dst.MBWidth, dst.MBHeight, dst.ChromaFormatIDC)
	dst.ChromaStride = dst.LumaStride + 16
	dst.Cb = make([]uint8, dst.ChromaStride*chromaHeight)
	dst.Cr = make([]uint8, dst.ChromaStride*chromaHeight)
	fillH264MotionCompPlane(dst.Cb, 52)
	fillH264MotionCompPlane(dst.Cr, 94)
	var residual cavlcResidualContext
	mbX, mbY := 1, 1

	if err := h264HLDecodeFrameMacroblock(dst, h264FrameMBReconstructInput{
		MBType:             MBTypeIntra16x16,
		MBX:                mbX,
		MBY:                mbY,
		QScale:             20,
		ChromaQP:           [2]uint8{20, 21},
		Intra16x16PredMode: int8(intraPredDC1288x8),
		PPS:                cavlcFlatQMulPPS(),
		Residual:           &residual,
	}); err != nil {
		t.Fatal(err)
	}

	_, cbOff, crOff, err := h264MBDestPartOffsets(dst, mbX, mbY, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	assertH264ConstantBlock(t, "444 cb", dst.Cb, cbOff, dst.ChromaStride, 16, 16, 128)
	assertH264ConstantBlock(t, "444 cr", dst.Cr, crOff, dst.ChromaStride, 16, 16, 128)
}

func TestH264HLDecodeFrameMacroblockIntra4x4Reconstructs420(t *testing.T) {
	dst := makeH264MotionCompPicture(1, 17)
	residual := h264ReconstructResidualIntra4x4()
	predCache := h264ReconstructIntra4x4PredCache()
	mbX, mbY := 1, 1
	yOff := mbY*16*dst.LumaStride + mbX*16
	before := dst.Y[yOff]

	if err := h264HLDecodeFrameMacroblock(dst, h264FrameMBReconstructInput{
		MBType:            MBTypeIntra4x4,
		MBX:               mbX,
		MBY:               mbY,
		CBP:               0x31,
		QScale:            20,
		ChromaQP:          [2]uint8{20, 21},
		ChromaPredMode:    int32(intraPred8x8DC),
		Intra4x4PredCache: &predCache,
		TopLeftAvailable:  0xffff,
		TopRightAvailable: 0xffff,
		PPS:               cavlcFlatQMulPPS(),
		Residual:          &residual,
	}); err != nil {
		t.Fatal(err)
	}

	if dst.Y[yOff] == before {
		t.Fatalf("intra4x4 luma top-left was not reconstructed, still %d", before)
	}
	if residual.MB[0] != 0 || residual.MB[5*16] != 0 || residual.MB[16*16] != 0 {
		t.Fatalf("intra4x4 residual blocks were not cleared after reconstruction: %d/%d/%d", residual.MB[0], residual.MB[5*16], residual.MB[16*16])
	}
}

func TestH264HLDecodeFrameMacroblockIntra8x8Reconstructs422(t *testing.T) {
	dst := makeH264MotionCompPicture(2, 31)
	residual := h264ReconstructResidualIntra8x8()
	predCache := h264ReconstructIntra8x8PredCache()
	mbX, mbY := 1, 1
	yOff := mbY*16*dst.LumaStride + mbX*16
	before := dst.Y[yOff]

	if err := h264HLDecodeFrameMacroblock(dst, h264FrameMBReconstructInput{
		MBType:            MBTypeIntra4x4 | MBType8x8DCT,
		MBX:               mbX,
		MBY:               mbY,
		CBP:               0x33,
		QScale:            22,
		ChromaQP:          [2]uint8{22, 23},
		ChromaPredMode:    int32(intraPred8x8Plane),
		Intra4x4PredCache: &predCache,
		TopLeftAvailable:  0xffff,
		TopRightAvailable: 0xffff,
		PPS:               cavlcFlatQMulPPS(),
		Residual:          &residual,
	}); err != nil {
		t.Fatal(err)
	}

	if dst.Y[yOff] == before {
		t.Fatalf("intra8x8 luma top-left was not reconstructed, still %d", before)
	}
	if residual.MB[0] != 0 || residual.MB[4*16] != 0 || residual.MB[16*16] != 0 {
		t.Fatalf("intra8x8 residual blocks were not cleared after reconstruction: %d/%d/%d", residual.MB[0], residual.MB[4*16], residual.MB[16*16])
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

func h264ReconstructIntraPCM(chromaFormatIDC int, seed int) []byte {
	n := h264IntraPCMSampleCount[chromaFormatIDC]
	pcm := make([]byte, n)
	for i := range pcm {
		pcm[i] = uint8((seed + 17*i + (i >> 3) + 3*(i>>6)) & 255)
	}
	return pcm
}

func h264ReconstructIntraPCMSamples(chromaFormatIDC int, bitDepth int, seed int) []uint16 {
	n := h264IntraPCMSampleCount[chromaFormatIDC]
	max := (1 << uint(bitDepth)) - 1
	samples := make([]uint16, n)
	for i := range samples {
		samples[i] = uint16((seed + 17*i + (i >> 3) + 3*(i>>6)) & max)
	}
	return samples
}

func h264ReconstructIntraPCMHigh(chromaFormatIDC int, bitDepth int, seed int) []byte {
	return h264PackIntraPCMHigh(h264ReconstructIntraPCMSamples(chromaFormatIDC, bitDepth, seed), bitDepth)
}

func h264PackIntraPCMHigh(samples []uint16, bitDepth int) []byte {
	out := make([]byte, (len(samples)*bitDepth+7)>>3)
	bitPos := 0
	for _, sample := range samples {
		for bit := bitDepth - 1; bit >= 0; bit-- {
			if (sample>>uint(bit))&1 != 0 {
				out[bitPos>>3] |= 1 << uint(7-(bitPos&7))
			}
			bitPos++
		}
	}
	return out
}

func makeH264ReconstructHighPicture(chromaFormatIDC int, seed int) *h264PicturePlanesHigh {
	const mbWidth = 4
	const mbHeight = 4
	p := &h264PicturePlanesHigh{
		Y:               make([]uint16, 80*mbHeight*16),
		LumaStride:      80,
		MBWidth:         mbWidth,
		MBHeight:        mbHeight,
		ChromaFormatIDC: chromaFormatIDC,
	}
	fillH264ReconstructHighPlane(p.Y, seed)
	if chromaFormatIDC != 0 {
		chromaWidth, chromaHeight := h264ChromaFrameSize(mbWidth, mbHeight, chromaFormatIDC)
		p.ChromaStride = 48
		if p.ChromaStride < chromaWidth {
			p.ChromaStride = 80
		}
		p.Cb = make([]uint16, p.ChromaStride*chromaHeight)
		p.Cr = make([]uint16, p.ChromaStride*chromaHeight)
		fillH264ReconstructHighPlane(p.Cb, seed+29)
		fillH264ReconstructHighPlane(p.Cr, seed+71)
	}
	return p
}

func fillH264ReconstructHighPlane(p []uint16, seed int) {
	for i := range p {
		p[i] = uint16((seed + i*13 + (i>>4)*7) & 0x3fff)
	}
}

func assertH264Rows(t *testing.T, label string, dst []uint8, offset int, stride int, width int, height int, src []byte, srcStride int) {
	t.Helper()
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			got := dst[offset+y*stride+x]
			want := src[y*srcStride+x]
			if got != want {
				t.Fatalf("%s[%d,%d] = %d, want %d", label, x, y, got, want)
			}
		}
	}
}

func assertH264RowsHigh(t *testing.T, label string, dst []uint16, offset int, stride int, width int, height int, src []uint16, srcStride int) {
	t.Helper()
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			got := dst[offset+y*stride+x]
			want := src[y*srcStride+x]
			if got != want {
				t.Fatalf("%s[%d,%d] = %d, want %d", label, x, y, got, want)
			}
		}
	}
}

func assertH264ConstantBlock(t *testing.T, label string, dst []uint8, offset int, stride int, width int, height int, want uint8) {
	t.Helper()
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if got := dst[offset+y*stride+x]; got != want {
				t.Fatalf("%s[%d,%d] = %d, want %d", label, x, y, got, want)
			}
		}
	}
}

func h264ReconstructResidualIntra4x4() cavlcResidualContext {
	c := h264ReconstructResidual420()
	c.NonZeroCountCache[h264Scan8[lumaDCBlockIndex]] = 0
	for i := range c.MBLumaDC[0] {
		c.MBLumaDC[0][i] = 0
	}
	c.NonZeroCountCache[h264Scan8[0]] = 2
	c.MB[0] = 9
	c.MB[1] = -3
	c.NonZeroCountCache[h264Scan8[5]] = 1
	c.MB[5*16] = 11
	c.NonZeroCountCache[h264Scan8[10]] = 2
	c.MB[10*16] = -8
	c.MB[10*16+3] = 5
	c.NonZeroCountCache[h264Scan8[15]] = 1
	c.MB[15*16] = 6
	return c
}

func h264ReconstructResidualIntra8x8() cavlcResidualContext {
	c := h264ReconstructResidual422()
	c.NonZeroCountCache[h264Scan8[lumaDCBlockIndex]] = 0
	for i := range c.MBLumaDC[0] {
		c.MBLumaDC[0][i] = 0
	}
	for _, i := range []int{0, 4, 8, 12} {
		c.NonZeroCountCache[h264Scan8[i]] = 1
		c.MB[i*16] = int32(8 + i)
		c.MB[i*16+7] = int32(i - 5)
	}
	c.NonZeroCountCache[h264Scan8[4]] = 2
	c.MB[4*16+1] = -4
	c.NonZeroCountCache[h264Scan8[8]] = 2
	c.MB[8*16+9] = 3
	return c
}

func h264ReconstructIntra4x4PredCache() [h264IntraPredModeCacheSize]int8 {
	var cache [h264IntraPredModeCacheSize]int8
	modes := [16]int8{
		intraPredVertical, intraPredHorizontal, intraPredDC, intraPredDiagDownLeft,
		intraPredDiagDownRight, intraPredVertRight, intraPredHorDown, intraPredVertLeft,
		intraPredHorUp, intraPredLeftDC, intraPredTopDC, intraPredDC128,
		intraPredVertical, intraPredHorizontal, intraPredDC, intraPredDiagDownLeft,
	}
	for i, mode := range modes {
		cache[h264Scan8[i]] = mode
	}
	return cache
}

func h264ReconstructIntra8x8PredCache() [h264IntraPredModeCacheSize]int8 {
	var cache [h264IntraPredModeCacheSize]int8
	modes := map[int]int8{
		0:  intraPredVertical,
		4:  intraPredDiagDownLeft,
		8:  intraPredVertRight,
		12: intraPredHorDown,
	}
	for i, mode := range modes {
		cache[h264Scan8[i]] = mode
	}
	return cache
}
