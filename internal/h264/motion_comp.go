// SPDX-License-Identifier: LGPL-2.1-or-later
//
// Source-shaped frame-MB motion-compensation call-site helpers from FFmpeg
// n8.0.1 libavcodec/h264_mb.c mc_dir_part/mc_part_std and
// libavcodec/h264_mc_template.c hl_motion.

package h264

type h264PicturePlanes struct {
	Y, Cb, Cr        []uint8
	LumaStride       int
	ChromaStride     int
	MBWidth          int
	MBHeight         int
	ChromaFormatIDC  int
	PictureStructure int32
}

type h264MotionCompScratch struct {
	Y, Cb, Cr []uint8
	Edge      []uint8
}

func h264HLMotionFrame(dst *h264PicturePlanes, refs [2][]*h264PicturePlanes, cache *macroblockMotionCache, mbType uint32, subMBType [4]uint32, mbX int, mbY int, listCount int) error {
	return h264HLMotionFrameCore(dst, refs, cache, mbType, subMBType, mbX, mbY, listCount, nil, nil)
}

func h264HLMotionFrameWithScratch(dst *h264PicturePlanes, refs [2][]*h264PicturePlanes, cache *macroblockMotionCache, mbType uint32, subMBType [4]uint32, mbX int, mbY int, listCount int, scratch *h264MotionCompScratch) error {
	return h264HLMotionFrameCore(dst, refs, cache, mbType, subMBType, mbX, mbY, listCount, nil, scratch)
}

func h264HLMotionFrameWeighted(dst *h264PicturePlanes, refs [2][]*h264PicturePlanes, cache *macroblockMotionCache, mbType uint32, subMBType [4]uint32, mbX int, mbY int, listCount int, pwt *PredWeightTable, scratch *h264MotionCompScratch) error {
	if pwt == nil {
		return ErrInvalidData
	}
	return h264HLMotionFrameCore(dst, refs, cache, mbType, subMBType, mbX, mbY, listCount, pwt, scratch)
}

func h264HLMotionFrameCore(dst *h264PicturePlanes, refs [2][]*h264PicturePlanes, cache *macroblockMotionCache, mbType uint32, subMBType [4]uint32, mbX int, mbY int, listCount int, pwt *PredWeightTable, scratch *h264MotionCompScratch) error {
	if dst == nil || cache == nil || mbX < 0 || mbY < 0 || listCount < 0 || listCount > 2 {
		return ErrInvalidData
	}
	if err := dst.validate(); err != nil {
		return err
	}
	if mbX >= dst.MBWidth || mbY >= dst.MBHeight || !isInter(mbType) {
		return ErrInvalidData
	}

	if is16x16(mbType) {
		return h264MCPartFrame(dst, refs, cache, mbX, mbY, mbType, 0, 0, true, 16, 0, 0, 0, 16, 8, 16, listCount, pwt, scratch)
	}
	if is16x8(mbType) {
		if err := h264MCPartFrame(dst, refs, cache, mbX, mbY, mbType, 0, 0, false, 8, 8, 0, 0, 8, 8, 16, listCount, pwt, scratch); err != nil {
			return err
		}
		return h264MCPartFrame(dst, refs, cache, mbX, mbY, mbType, 1, 8, false, 8, 8, 0, 4, 8, 8, 16, listCount, pwt, scratch)
	}
	if is8x16(mbType) {
		delta := 8 * dst.LumaStride
		if err := h264MCPartFrame(dst, refs, cache, mbX, mbY, mbType, 0, 0, false, 16, delta, 0, 0, 8, 4, 8, listCount, pwt, scratch); err != nil {
			return err
		}
		return h264MCPartFrame(dst, refs, cache, mbX, mbY, mbType, 1, 4, false, 16, delta, 4, 0, 8, 4, 8, listCount, pwt, scratch)
	}
	if !is8x8(mbType) {
		return ErrUnsupported
	}

	for i := 0; i < 4; i++ {
		subType := subMBType[i]
		n := 4 * i
		xOffset := (i & 1) << 2
		yOffset := (i & 2) << 1

		if isSub8x8(subType) {
			if err := h264MCPartFrame(dst, refs, cache, mbX, mbY, subType, 0, n, true, 8, 0, xOffset, yOffset, 8, 4, 8, listCount, pwt, scratch); err != nil {
				return err
			}
		} else if isSub8x4(subType) {
			if err := h264MCPartFrame(dst, refs, cache, mbX, mbY, subType, 0, n, false, 4, 4, xOffset, yOffset, 4, 4, 8, listCount, pwt, scratch); err != nil {
				return err
			}
			if err := h264MCPartFrame(dst, refs, cache, mbX, mbY, subType, 0, n+2, false, 4, 4, xOffset, yOffset+2, 4, 4, 8, listCount, pwt, scratch); err != nil {
				return err
			}
		} else if isSub4x8(subType) {
			delta := 4 * dst.LumaStride
			if err := h264MCPartFrame(dst, refs, cache, mbX, mbY, subType, 0, n, false, 8, delta, xOffset, yOffset, 4, 2, 4, listCount, pwt, scratch); err != nil {
				return err
			}
			if err := h264MCPartFrame(dst, refs, cache, mbX, mbY, subType, 0, n+1, false, 8, delta, xOffset+2, yOffset, 4, 2, 4, listCount, pwt, scratch); err != nil {
				return err
			}
		} else if isSub4x4(subType) {
			for j := 0; j < 4; j++ {
				subXOffset := xOffset + 2*(j&1)
				subYOffset := yOffset + (j & 2)
				if err := h264MCPartFrame(dst, refs, cache, mbX, mbY, subType, 0, n+j, true, 4, 0, subXOffset, subYOffset, 4, 2, 4, listCount, pwt, scratch); err != nil {
					return err
				}
			}
		} else {
			return ErrUnsupported
		}
	}
	return nil
}

func h264MCPartFrame(dst *h264PicturePlanes, refs [2][]*h264PicturePlanes, cache *macroblockMotionCache, mbX int, mbY int, mbType uint32, part int, n int, square bool, height int, delta int, xOffset int, yOffset int, qpelSize int, chromaWidth int, lumaWeightWidth int, listCount int, pwt *PredWeightTable, scratch *h264MotionCompScratch) error {
	list0 := isDir(mbType, part, 0)
	list1 := isDir(mbType, part, 1)
	if h264MCPartUsesWeighted(pwt, cache, n, list0, list1, mbY) {
		return h264MCPartFrameWeighted(dst, refs, cache, mbX, mbY, mbType, part, n, square, height, delta, xOffset, yOffset, qpelSize, chromaWidth, lumaWeightWidth, listCount, pwt, scratch)
	}
	return h264MCPartFrameStd(dst, refs, cache, mbX, mbY, mbType, part, n, square, height, delta, xOffset, yOffset, qpelSize, chromaWidth, listCount, scratch)
}

func h264MCPartUsesWeighted(pwt *PredWeightTable, cache *macroblockMotionCache, n int, list0 bool, list1 bool, mbY int) bool {
	if pwt == nil || cache == nil || n < 0 || n >= 16 {
		return false
	}
	if pwt.UseWeight == 1 {
		return true
	}
	if pwt.UseWeight != 2 || !list0 || !list1 {
		return false
	}
	refn0 := cache.Ref[0][h264Scan8[n]]
	refn1 := cache.Ref[1][h264Scan8[n]]
	if refn0 < 0 || refn1 < 0 || int(refn0) >= len(pwt.ImplicitWeight) || int(refn1) >= len(pwt.ImplicitWeight[0]) {
		return false
	}
	return pwt.ImplicitWeight[refn0][refn1][mbY&1] != 32
}

func h264MCPartFrameStd(dst *h264PicturePlanes, refs [2][]*h264PicturePlanes, cache *macroblockMotionCache, mbX int, mbY int, mbType uint32, part int, n int, square bool, height int, delta int, xOffset int, yOffset int, qpelSize int, chromaWidth int, listCount int, scratch *h264MotionCompScratch) error {
	list0 := isDir(mbType, part, 0)
	list1 := isDir(mbType, part, 1)
	if (!list0 && !list1) || qpelSize <= 0 {
		return nil
	}
	if list0 && listCount < 1 || list1 && listCount < 2 {
		return ErrInvalidData
	}
	dstY, dstCb, dstCr, err := h264MBDestPartOffsets(dst, mbX, mbY, xOffset, yOffset)
	if err != nil {
		return err
	}
	srcXOffset := xOffset + 8*mbX
	srcYOffset := yOffset + 8*mbY
	avg := false

	if list0 {
		ref, err := h264MCReference(refs, cache, 0, n)
		if err != nil {
			return err
		}
		if err := h264MCDirPartFrame(dst, ref, cache, n, square, height, delta, 0, dstY, dstCb, dstCr, srcXOffset, srcYOffset, qpelSize, chromaWidth, avg, scratch); err != nil {
			return err
		}
		avg = true
	}
	if list1 {
		ref, err := h264MCReference(refs, cache, 1, n)
		if err != nil {
			return err
		}
		if err := h264MCDirPartFrame(dst, ref, cache, n, square, height, delta, 1, dstY, dstCb, dstCr, srcXOffset, srcYOffset, qpelSize, chromaWidth, avg, scratch); err != nil {
			return err
		}
	}
	return nil
}

func h264MCPartFrameWeighted(dst *h264PicturePlanes, refs [2][]*h264PicturePlanes, cache *macroblockMotionCache, mbX int, mbY int, mbType uint32, part int, n int, square bool, height int, delta int, xOffset int, yOffset int, qpelSize int, chromaWidth int, lumaWeightWidth int, listCount int, pwt *PredWeightTable, scratch *h264MotionCompScratch) error {
	list0 := isDir(mbType, part, 0)
	list1 := isDir(mbType, part, 1)
	if (!list0 && !list1) || qpelSize <= 0 || lumaWeightWidth <= 0 || pwt == nil {
		return nil
	}
	if list0 && listCount < 1 || list1 && listCount < 2 {
		return ErrInvalidData
	}
	dstY, dstCb, dstCr, err := h264MBDestPartOffsets(dst, mbX, mbY, xOffset, yOffset)
	if err != nil {
		return err
	}
	srcXOffset := xOffset + 8*mbX
	srcYOffset := yOffset + 8*mbY
	chromaHeight, chromaWeightWidth, err := h264ChromaWeightGeometry(dst.ChromaFormatIDC, height, chromaWidth, lumaWeightWidth)
	if err != nil {
		return err
	}

	if list0 && list1 {
		ref0, err := h264MCReference(refs, cache, 0, n)
		if err != nil {
			return err
		}
		ref1, err := h264MCReference(refs, cache, 1, n)
		if err != nil {
			return err
		}
		if scratch == nil || !scratch.valid(dst, height, lumaWeightWidth, chromaHeight, chromaWeightWidth) {
			return ErrInvalidData
		}
		if err := h264MCDirPartFrame(dst, ref0, cache, n, square, height, delta, 0, dstY, dstCb, dstCr, srcXOffset, srcYOffset, qpelSize, chromaWidth, false, scratch); err != nil {
			return err
		}
		if err := h264MCDirPartFramePlanes(scratch.Y, scratch.Cb, scratch.Cr, dst.LumaStride, dst.ChromaStride, dst.ChromaFormatIDC, dst.PictureStructure, ref1, cache, n, square, height, delta, 1, 0, 0, 0, srcXOffset, srcYOffset, qpelSize, chromaWidth, false, scratch); err != nil {
			return err
		}
		refn0 := int(cache.Ref[0][h264Scan8[n]])
		refn1 := int(cache.Ref[1][h264Scan8[n]])
		if refn0 < 0 || refn1 < 0 || refn0 >= len(pwt.LumaWeight) || refn1 >= len(pwt.LumaWeight) {
			return ErrInvalidData
		}
		if pwt.UseWeight == 2 {
			weight0 := int(pwt.ImplicitWeight[refn0][refn1][mbY&1])
			weight1 := 64 - weight0
			if err := h264BiweightPixels(dst.Y[dstY:], scratch.Y, dst.LumaStride, height, 5, weight0, weight1, 0, lumaWeightWidth); err != nil {
				return err
			}
			if dst.ChromaFormatIDC != 0 {
				if err := h264BiweightPixels(dst.Cb[dstCb:], scratch.Cb, dst.ChromaStride, chromaHeight, 5, weight0, weight1, 0, chromaWeightWidth); err != nil {
					return err
				}
				return h264BiweightPixels(dst.Cr[dstCr:], scratch.Cr, dst.ChromaStride, chromaHeight, 5, weight0, weight1, 0, chromaWeightWidth)
			}
			return nil
		}
		if err := h264BiweightPixels(dst.Y[dstY:], scratch.Y, dst.LumaStride, height, int(pwt.LumaLog2WeightDenom), int(pwt.LumaWeight[refn0][0][0]), int(pwt.LumaWeight[refn1][1][0]), int(pwt.LumaWeight[refn0][0][1]+pwt.LumaWeight[refn1][1][1]), lumaWeightWidth); err != nil {
			return err
		}
		if dst.ChromaFormatIDC != 0 {
			if err := h264BiweightPixels(dst.Cb[dstCb:], scratch.Cb, dst.ChromaStride, chromaHeight, int(pwt.ChromaLog2WeightDenom), int(pwt.ChromaWeight[refn0][0][0][0]), int(pwt.ChromaWeight[refn1][1][0][0]), int(pwt.ChromaWeight[refn0][0][0][1]+pwt.ChromaWeight[refn1][1][0][1]), chromaWeightWidth); err != nil {
				return err
			}
			return h264BiweightPixels(dst.Cr[dstCr:], scratch.Cr, dst.ChromaStride, chromaHeight, int(pwt.ChromaLog2WeightDenom), int(pwt.ChromaWeight[refn0][0][1][0]), int(pwt.ChromaWeight[refn1][1][1][0]), int(pwt.ChromaWeight[refn0][0][1][1]+pwt.ChromaWeight[refn1][1][1][1]), chromaWeightWidth)
		}
		return nil
	}

	list := 0
	if list1 {
		list = 1
	}
	ref, err := h264MCReference(refs, cache, list, n)
	if err != nil {
		return err
	}
	refn := int(cache.Ref[list][h264Scan8[n]])
	if refn < 0 || refn >= len(pwt.LumaWeight) {
		return ErrInvalidData
	}
	if err := h264MCDirPartFrame(dst, ref, cache, n, square, height, delta, list, dstY, dstCb, dstCr, srcXOffset, srcYOffset, qpelSize, chromaWidth, false, scratch); err != nil {
		return err
	}
	if err := h264WeightPixels(dst.Y[dstY:], dst.LumaStride, height, int(pwt.LumaLog2WeightDenom), int(pwt.LumaWeight[refn][list][0]), int(pwt.LumaWeight[refn][list][1]), lumaWeightWidth); err != nil {
		return err
	}
	if dst.ChromaFormatIDC != 0 && pwt.UseWeightChroma != 0 {
		if err := h264WeightPixels(dst.Cb[dstCb:], dst.ChromaStride, chromaHeight, int(pwt.ChromaLog2WeightDenom), int(pwt.ChromaWeight[refn][list][0][0]), int(pwt.ChromaWeight[refn][list][0][1]), chromaWeightWidth); err != nil {
			return err
		}
		return h264WeightPixels(dst.Cr[dstCr:], dst.ChromaStride, chromaHeight, int(pwt.ChromaLog2WeightDenom), int(pwt.ChromaWeight[refn][list][1][0]), int(pwt.ChromaWeight[refn][list][1][1]), chromaWeightWidth)
	}
	return nil
}

func h264ChromaWeightGeometry(chromaFormatIDC int, height int, chromaWidth int, lumaWeightWidth int) (int, int, error) {
	switch chromaFormatIDC {
	case 0:
		return 0, 0, nil
	case 1:
		return height >> 1, chromaWidth, nil
	case 2:
		return height, chromaWidth, nil
	case 3:
		return height, lumaWeightWidth, nil
	default:
		return 0, 0, ErrInvalidData
	}
}

func (s *h264MotionCompScratch) valid(dst *h264PicturePlanes, lumaHeight int, lumaWidth int, chromaHeight int, chromaWidth int) bool {
	if s == nil || dst == nil {
		return false
	}
	if !h264PlaneHas(s.Y, dst.LumaStride, lumaHeight, lumaWidth) {
		return false
	}
	if dst.ChromaFormatIDC == 0 {
		return true
	}
	return h264PlaneHas(s.Cb, dst.ChromaStride, chromaHeight, chromaWidth) &&
		h264PlaneHas(s.Cr, dst.ChromaStride, chromaHeight, chromaWidth)
}

func h264PlaneHas(p []uint8, stride int, height int, width int) bool {
	if stride <= 0 || height < 0 || width < 0 {
		return false
	}
	if height == 0 {
		return true
	}
	if width == 0 || stride < width {
		return false
	}
	return len(p) >= (height-1)*stride+width
}

func h264EdgeStride(stride int, blockW int) int {
	if stride < blockW {
		return blockW
	}
	return stride
}

func h264EdgeScratchSize(stride int, blockW int, blockH int) int {
	edgeStride := h264EdgeStride(stride, blockW)
	return (blockH-1)*edgeStride + blockW
}

func h264EdgeScratch(s *h264MotionCompScratch, stride int, blockW int, blockH int) ([]uint8, int, error) {
	if s == nil || stride <= 0 || blockW <= 0 || blockH <= 0 {
		return nil, 0, ErrInvalidData
	}
	edgeStride := h264EdgeStride(stride, blockW)
	needed := h264EdgeScratchSize(stride, blockW, blockH)
	if len(s.Edge) < needed {
		return nil, 0, ErrInvalidData
	}
	return s.Edge, edgeStride, nil
}

func h264EmulatedEdgeMC(buf []uint8, bufOffset int, bufStride int, src []uint8, srcStride int, blockW int, blockH int, srcX int, srcY int, width int, height int) error {
	if bufOffset < 0 || bufStride <= 0 || srcStride <= 0 || blockW <= 0 || blockH <= 0 || width < 0 || height < 0 {
		return ErrInvalidData
	}
	if width == 0 || height == 0 {
		return nil
	}
	if bufStride < blockW || srcStride < width {
		return ErrInvalidData
	}
	bufMax := bufOffset + (blockH-1)*bufStride + blockW - 1
	if bufMax >= len(buf) || len(src) < (height-1)*srcStride+width {
		return ErrInvalidData
	}
	for y := 0; y < blockH; y++ {
		sy := clipInt(srcY+y, 0, height-1)
		dstRow := bufOffset + y*bufStride
		srcRow := sy * srcStride
		for x := 0; x < blockW; x++ {
			sx := clipInt(srcX+x, 0, width-1)
			buf[dstRow+x] = src[srcRow+sx]
		}
	}
	return nil
}

func h264MCDirPartFrame(dst *h264PicturePlanes, ref *h264PicturePlanes, cache *macroblockMotionCache, n int, square bool, height int, delta int, list int, dstY int, dstCb int, dstCr int, srcXOffset int, srcYOffset int, qpelSize int, chromaWidth int, avg bool, scratch *h264MotionCompScratch) error {
	if dst == nil || ref == nil || cache == nil || n < 0 || n >= 16 || list < 0 || list > 1 || height <= 0 || delta < 0 {
		return ErrInvalidData
	}
	if err := h264CheckMotionPlanePair(dst, ref); err != nil {
		return err
	}
	return h264MCDirPartFramePlanes(dst.Y, dst.Cb, dst.Cr, dst.LumaStride, dst.ChromaStride, dst.ChromaFormatIDC, dst.PictureStructure, ref, cache, n, square, height, delta, list, dstY, dstCb, dstCr, srcXOffset, srcYOffset, qpelSize, chromaWidth, avg, scratch)
}

func h264MCDirPartFramePlanes(dstYPlane []uint8, dstCbPlane []uint8, dstCrPlane []uint8, dstLumaStride int, dstChromaStride int, chromaFormatIDC int, dstPictureStructure int32, ref *h264PicturePlanes, cache *macroblockMotionCache, n int, square bool, height int, delta int, list int, dstY int, dstCb int, dstCr int, srcXOffset int, srcYOffset int, qpelSize int, chromaWidth int, avg bool, scratch *h264MotionCompScratch) error {
	if ref == nil || cache == nil || n < 0 || n >= 16 || list < 0 || list > 1 || dstLumaStride <= 0 || dstChromaStride < 0 || chromaFormatIDC < 0 || chromaFormatIDC > 3 || height <= 0 || delta < 0 {
		return ErrInvalidData
	}
	if err := ref.validate(); err != nil {
		return err
	}
	if dstLumaStride != ref.LumaStride || (chromaFormatIDC != 0 && dstChromaStride != ref.ChromaStride) {
		return ErrInvalidData
	}
	mv := cache.MV[list][h264Scan8[n]]
	mx := int(mv[0]) + srcXOffset*8
	my := int(mv[1]) + srcYOffset*8
	lumaXY := (mx & 3) + ((my & 3) << 2)
	fullMx := mx >> 2
	fullMy := my >> 2
	extraWidth := 0
	extraHeight := 0
	if mx&7 != 0 {
		extraWidth -= 3
	}
	if my&7 != 0 {
		extraHeight -= 3
	}
	emu := false
	if fullMx < -extraWidth ||
		fullMy < -extraHeight ||
		fullMx+16 > ref.MBWidth*16+extraWidth ||
		fullMy+16 > ref.MBHeight*16+extraHeight {
		emu = true
	}

	srcY := fullMx + fullMy*ref.LumaStride
	srcYPlane := ref.Y
	srcLumaStride := ref.LumaStride
	if emu {
		edge, edgeStride, err := h264EdgeScratch(scratch, ref.LumaStride, 16+5, 16+5)
		if err != nil {
			return err
		}
		if err := h264EmulatedEdgeMC(edge, 0, edgeStride, ref.Y, ref.LumaStride, 16+5, 16+5, fullMx-2, fullMy-2, ref.MBWidth*16, ref.MBHeight*16); err != nil {
			return err
		}
		srcYPlane = edge
		srcLumaStride = edgeStride
		srcY = 2 + 2*edgeStride
	}
	if err := h264CallQpelMCStrides(dstYPlane, dstY, dstLumaStride, srcYPlane, srcY, srcLumaStride, qpelSize, lumaXY, avg); err != nil {
		return err
	}
	if !square {
		srcDelta := h264RemapDeltaForStride(delta, dstLumaStride, srcLumaStride)
		if err := h264CallQpelMCStrides(dstYPlane, dstY+delta, dstLumaStride, srcYPlane, srcY+srcDelta, srcLumaStride, qpelSize, lumaXY, avg); err != nil {
			return err
		}
	}

	switch chromaFormatIDC {
	case 0:
		return nil
	case 3:
		srcC := fullMx + fullMy*ref.ChromaStride
		srcCbPlane := ref.Cb
		srcChromaStride := ref.ChromaStride
		if emu {
			edge, edgeStride, err := h264EdgeScratch(scratch, ref.ChromaStride, 16+5, 16+5)
			if err != nil {
				return err
			}
			if err := h264EmulatedEdgeMC(edge, 0, edgeStride, ref.Cb, ref.ChromaStride, 16+5, 16+5, fullMx-2, fullMy-2, ref.MBWidth*16, ref.MBHeight*16); err != nil {
				return err
			}
			srcCbPlane = edge
			srcChromaStride = edgeStride
			srcC = 2 + 2*edgeStride
		}
		if err := h264CallQpelMCStrides(dstCbPlane, dstCb, dstChromaStride, srcCbPlane, srcC, srcChromaStride, qpelSize, lumaXY, avg); err != nil {
			return err
		}
		if !square {
			srcDelta := h264RemapDeltaForStride(delta, dstChromaStride, srcChromaStride)
			if err := h264CallQpelMCStrides(dstCbPlane, dstCb+delta, dstChromaStride, srcCbPlane, srcC+srcDelta, srcChromaStride, qpelSize, lumaXY, avg); err != nil {
				return err
			}
		}
		srcCrPlane := ref.Cr
		srcChromaStride = ref.ChromaStride
		if emu {
			edge, edgeStride, err := h264EdgeScratch(scratch, ref.ChromaStride, 16+5, 16+5)
			if err != nil {
				return err
			}
			if err := h264EmulatedEdgeMC(edge, 0, edgeStride, ref.Cr, ref.ChromaStride, 16+5, 16+5, fullMx-2, fullMy-2, ref.MBWidth*16, ref.MBHeight*16); err != nil {
				return err
			}
			srcCrPlane = edge
			srcChromaStride = edgeStride
			srcC = 2 + 2*edgeStride
		}
		if err := h264CallQpelMCStrides(dstCrPlane, dstCr, dstChromaStride, srcCrPlane, srcC, srcChromaStride, qpelSize, lumaXY, avg); err != nil {
			return err
		}
		if !square {
			srcDelta := h264RemapDeltaForStride(delta, dstChromaStride, srcChromaStride)
			return h264CallQpelMCStrides(dstCrPlane, dstCr+delta, dstChromaStride, srcCrPlane, srcC+srcDelta, srcChromaStride, qpelSize, lumaXY, avg)
		}
		return nil
	case 1, 2:
		yShift := 3
		chromaHeight := height
		chromaMy := my
		chromaEmu := emu
		if chromaFormatIDC == 1 {
			if dstParity, ok := h264FieldParity(dstPictureStructure); ok {
				if refParity, ok := h264FieldParity(ref.PictureStructure); ok {
					chromaMy += 2 * (dstParity - refParity)
					chromaFieldY := chromaMy >> 3
					chromaPicH := ref.MBHeight * 8
					if chromaFieldY < 0 || chromaFieldY+8 >= chromaPicH {
						chromaEmu = true
					}
				}
			}
		}
		chromaY := chromaMy & 7
		if chromaFormatIDC == 1 {
			chromaHeight >>= 1
		} else {
			yShift = 2
			chromaY = (chromaMy << 1) & 7
		}
		srcC := (mx >> 3) + (chromaMy>>yShift)*ref.ChromaStride
		chromaX := mx & 7
		if !chromaEmu {
			if srcC < 0 || dstCb < 0 || dstCr < 0 || srcC > len(ref.Cb) || srcC > len(ref.Cr) || dstCb > len(dstCbPlane) || dstCr > len(dstCrPlane) {
				return ErrInvalidData
			}
			if err := h264ChromaMCStrides(dstCbPlane[dstCb:], ref.Cb[srcC:], dstChromaStride, ref.ChromaStride, chromaHeight, chromaX, chromaY, chromaWidth, avg); err != nil {
				return err
			}
			return h264ChromaMCStrides(dstCrPlane[dstCr:], ref.Cr[srcC:], dstChromaStride, ref.ChromaStride, chromaHeight, chromaX, chromaY, chromaWidth, avg)
		}

		{
			blockH := 8*chromaFormatIDC + 1
			picW := ref.MBWidth * 8
			picH := ref.MBHeight * 16
			if chromaFormatIDC == 1 {
				picH >>= 1
			}
			edge, edgeStride, err := h264EdgeScratch(scratch, ref.ChromaStride, 9, blockH)
			if err != nil {
				return err
			}
			if err := h264EmulatedEdgeMC(edge, 0, edgeStride, ref.Cb, ref.ChromaStride, 9, blockH, mx>>3, chromaMy>>yShift, picW, picH); err != nil {
				return err
			}
			if dstCb < 0 || dstCb > len(dstCbPlane) {
				return ErrInvalidData
			}
			if err := h264ChromaMCStrides(dstCbPlane[dstCb:], edge, dstChromaStride, edgeStride, chromaHeight, chromaX, chromaY, chromaWidth, avg); err != nil {
				return err
			}

			edge, edgeStride, err = h264EdgeScratch(scratch, ref.ChromaStride, 9, blockH)
			if err != nil {
				return err
			}
			if err := h264EmulatedEdgeMC(edge, 0, edgeStride, ref.Cr, ref.ChromaStride, 9, blockH, mx>>3, chromaMy>>yShift, picW, picH); err != nil {
				return err
			}
			if dstCr < 0 || dstCr > len(dstCrPlane) {
				return ErrInvalidData
			}
			return h264ChromaMCStrides(dstCrPlane[dstCr:], edge, dstChromaStride, edgeStride, chromaHeight, chromaX, chromaY, chromaWidth, avg)
		}
	default:
		return ErrInvalidData
	}
}

func h264FieldParity(pictureStructure int32) (int, bool) {
	switch pictureStructure {
	case PictureTopField:
		return 0, true
	case PictureBottomField:
		return 1, true
	default:
		return 0, false
	}
}

func h264MBDestPartOffsets(dst *h264PicturePlanes, mbX int, mbY int, xOffset int, yOffset int) (int, int, int, error) {
	if dst == nil || mbX < 0 || mbY < 0 || xOffset < 0 || yOffset < 0 {
		return 0, 0, 0, ErrInvalidData
	}
	dstY := mbY*16*dst.LumaStride + mbX*16 + 2*xOffset + 2*yOffset*dst.LumaStride
	dstCb, dstCr := 0, 0
	switch dst.ChromaFormatIDC {
	case 0:
	case 1:
		dstCb = mbY*8*dst.ChromaStride + mbX*8 + xOffset + yOffset*dst.ChromaStride
		dstCr = dstCb
	case 2:
		dstCb = mbY*16*dst.ChromaStride + mbX*8 + xOffset + 2*yOffset*dst.ChromaStride
		dstCr = dstCb
	case 3:
		dstCb = mbY*16*dst.ChromaStride + mbX*16 + 2*xOffset + 2*yOffset*dst.ChromaStride
		dstCr = dstCb
	default:
		return 0, 0, 0, ErrInvalidData
	}
	return dstY, dstCb, dstCr, nil
}

func h264MCReference(refs [2][]*h264PicturePlanes, cache *macroblockMotionCache, list int, n int) (*h264PicturePlanes, error) {
	if cache == nil || list < 0 || list > 1 || n < 0 || n >= 16 {
		return nil, ErrInvalidData
	}
	refIdx := cache.Ref[list][h264Scan8[n]]
	if refIdx < 0 || int(refIdx) >= len(refs[list]) || refs[list][refIdx] == nil {
		return nil, ErrInvalidData
	}
	return refs[list][refIdx], nil
}

func h264CallQpelMC(dst []uint8, dstOffset int, src []uint8, srcOffset int, stride int, size int, lumaXY int, avg bool) error {
	return h264CallQpelMCStrides(dst, dstOffset, stride, src, srcOffset, stride, size, lumaXY, avg)
}

func h264CallQpelMCStrides(dst []uint8, dstOffset int, dstStride int, src []uint8, srcOffset int, srcStride int, size int, lumaXY int, avg bool) error {
	mx := lumaXY & 3
	my := (lumaXY >> 2) & 3
	if avg {
		return h264QpelMCStrides(dst, dstOffset, dstStride, src, srcOffset, srcStride, size, mx, my, true)
	}
	return h264QpelMCStrides(dst, dstOffset, dstStride, src, srcOffset, srcStride, size, mx, my, false)
}

func h264RemapDeltaForStride(delta int, dstStride int, srcStride int) int {
	if delta > 0 && dstStride > 0 && delta >= dstStride && delta%dstStride == 0 {
		return (delta / dstStride) * srcStride
	}
	return delta
}

func h264CheckMotionPlanePair(dst *h264PicturePlanes, ref *h264PicturePlanes) error {
	if err := dst.validate(); err != nil {
		return err
	}
	if err := ref.validate(); err != nil {
		return err
	}
	if dst.MBWidth != ref.MBWidth ||
		dst.MBHeight != ref.MBHeight ||
		dst.ChromaFormatIDC != ref.ChromaFormatIDC ||
		dst.LumaStride != ref.LumaStride ||
		dst.ChromaStride != ref.ChromaStride {
		return ErrInvalidData
	}
	if dst.ChromaFormatIDC == 3 && dst.ChromaStride != dst.LumaStride {
		return ErrUnsupported
	}
	return nil
}

func (p *h264PicturePlanes) validate() error {
	if p == nil || p.MBWidth <= 0 || p.MBHeight <= 0 || p.LumaStride <= 0 || p.ChromaFormatIDC < 0 || p.ChromaFormatIDC > 3 {
		return ErrInvalidData
	}
	lumaWidth := p.MBWidth * 16
	lumaHeight := p.MBHeight * 16
	if p.LumaStride < lumaWidth || len(p.Y) < (lumaHeight-1)*p.LumaStride+lumaWidth {
		return ErrInvalidData
	}
	if p.ChromaFormatIDC == 0 {
		return nil
	}
	chromaWidth, chromaHeight := h264ChromaFrameSize(p.MBWidth, p.MBHeight, p.ChromaFormatIDC)
	if p.ChromaStride < chromaWidth || len(p.Cb) < (chromaHeight-1)*p.ChromaStride+chromaWidth || len(p.Cr) < (chromaHeight-1)*p.ChromaStride+chromaWidth {
		return ErrInvalidData
	}
	return nil
}

func h264ChromaFrameSize(mbWidth int, mbHeight int, chromaFormatIDC int) (int, int) {
	switch chromaFormatIDC {
	case 1:
		return mbWidth * 8, mbHeight * 8
	case 2:
		return mbWidth * 8, mbHeight * 16
	case 3:
		return mbWidth * 16, mbHeight * 16
	default:
		return 0, 0
	}
}

func isSub8x8(mbType uint32) bool {
	return mbType&MBType16x16 != 0
}

func isSub8x4(mbType uint32) bool {
	return mbType&MBType16x8 != 0
}

func isSub4x8(mbType uint32) bool {
	return mbType&MBType8x16 != 0
}

func isSub4x4(mbType uint32) bool {
	return mbType&MBType8x8 != 0
}
