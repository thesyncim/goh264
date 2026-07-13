// SPDX-License-Identifier: LGPL-2.1-or-later
//
// Source-shaped simple 8-bit frame-MB reconstruction call-site helpers from
// FFmpeg n8.0.1 libavcodec/h264_mb.c hl_decode_mb_predict_luma,
// hl_decode_mb_idct_luma, and h264_mb_template.c hl_decode_mb.

package h264

type h264FrameMBReconstructInput struct {
	MBType              uint32
	SubMBType           [4]uint32
	MBX                 int
	MBY                 int
	CBP                 int
	QScale              int
	ChromaQP            [2]uint8
	ChromaPredMode      int32
	Intra16x16PredMode  int8
	Intra4x4PredCache   *[h264IntraPredModeCacheSize]int8
	TopLeftAvailable    uint16
	TopRightAvailable   uint16
	ListCount           int
	PPS                 *PPS
	Residual            *cavlcResidualContext
	Motion              *macroblockMotionCache
	Refs                [2][]*h264PicturePlanes
	PredWeight          *PredWeightTable
	MotionWeightMBY     int
	UseMotionWeightMBY  bool
	MotionScratch       *h264MotionCompScratch
	TransformBypass     bool
	DeblockingFilter    bool
	ConstrainedIntra444 bool
	IntraPCM            []byte
	X264Build           int32
	X264BuildSet        bool
}

func h264ProfileIDCFromPPS(pps *PPS) int32 {
	if pps != nil && pps.SPS != nil {
		return pps.SPS.ProfileIDC
	}
	return 0
}

func h264X264BuildUsesUnfiltered8x8LAdd(x264Build int32, x264BuildSet bool) bool {
	if !x264BuildSet {
		x264Build = -1
	}
	return uint32(x264Build) < 151
}

func h264HLDecodeFrameMacroblock(dst *h264PicturePlanes, in h264FrameMBReconstructInput) error {
	return h264HLDecodeFrameMacroblockCore(dst, in, false, nil)
}

// h264HLDecodeFrameMacroblockTrusted is restricted to slice decode, which
// validates the destination picture before constructing per-macroblock views.
func h264HLDecodeFrameMacroblockTrusted(dst *h264PicturePlanes, in h264FrameMBReconstructInput) error {
	return h264HLDecodeFrameMacroblockCore(dst, in, true, nil)
}

func h264HLDecodeFrameMacroblockTrustedWithBlockOffsets(dst *h264PicturePlanes, in h264FrameMBReconstructInput, blockOffset *[48]int) error {
	return h264HLDecodeFrameMacroblockCore(dst, in, true, blockOffset)
}

func h264HLDecodeFrameMacroblockCore(dst *h264PicturePlanes, in h264FrameMBReconstructInput, trustedDst bool, blockOffset *[48]int) error {
	if dst == nil || in.MBX < 0 || in.MBY < 0 || in.QScale < 0 || in.QScale > qpMaxNum {
		return ErrInvalidData
	}
	if in.DeblockingFilter || in.ConstrainedIntra444 {
		return ErrUnsupported
	}
	if !trustedDst {
		if err := dst.validate(); err != nil {
			return err
		}
	}
	if in.MBX >= dst.MBWidth || in.MBY >= dst.MBHeight {
		return ErrInvalidData
	}

	chromaStride := dst.ChromaStride
	if chromaStride == 0 {
		chromaStride = 1
	}
	var localBlockOffset [48]int
	if blockOffset == nil {
		var err error
		localBlockOffset, err = h264FrameBlockOffsets(dst.LumaStride, chromaStride, 0)
		if err != nil {
			return err
		}
		blockOffset = &localBlockOffset
	}
	dstY, dstCb, dstCr, err := h264MBDestPartOffsets(dst, in.MBX, in.MBY, 0, 0)
	if err != nil {
		return err
	}
	if dst.ChromaFormatIDC == 3 {
		return h264HLDecodeFrameMacroblock444(dst, dstY, dstCb, dstCr, blockOffset, in)
	}

	if in.MBType&MBTypeIntraPCM != 0 {
		return h264HLDecodeFrameIntraPCM(dst, dstY, dstCb, dstCr, in.IntraPCM)
	}
	if in.PPS == nil || in.Residual == nil {
		return ErrInvalidData
	}
	if isIntra(in.MBType) {
		if err := h264HLDecodeFrameIntraPredict(dst, dstY, dstCb, dstCr, blockOffset, in); err != nil {
			return err
		}
	} else {
		if in.Motion == nil {
			return ErrInvalidData
		}
		if in.PredWeight != nil {
			weightMBY := in.MBY
			if in.UseMotionWeightMBY {
				weightMBY = in.MotionWeightMBY
			}
			if trustedDst {
				if err := h264HLMotionFrameWeightedWithWeightYTrusted(dst, in.Refs, in.Motion, in.MBType, in.SubMBType, in.MBX, in.MBY, in.ListCount, weightMBY, in.PredWeight, in.MotionScratch); err != nil {
					return err
				}
			} else if err := h264HLMotionFrameWeightedWithWeightY(dst, in.Refs, in.Motion, in.MBType, in.SubMBType, in.MBX, in.MBY, in.ListCount, weightMBY, in.PredWeight, in.MotionScratch); err != nil {
				return err
			}
		} else {
			if trustedDst {
				if err := h264HLMotionFrameWithScratchTrusted(dst, in.Refs, in.Motion, in.MBType, in.SubMBType, in.MBX, in.MBY, in.ListCount, in.MotionScratch); err != nil {
					return err
				}
			} else if err := h264HLMotionFrameWithScratch(dst, in.Refs, in.Motion, in.MBType, in.SubMBType, in.MBX, in.MBY, in.ListCount, in.MotionScratch); err != nil {
				return err
			}
		}
	}

	profileIDC := h264ProfileIDCFromPPS(in.PPS)
	if err := h264HLDecodeMBIDCTLuma(dst.Y, dstY, dst.LumaStride, blockOffset, in.MBType, in.CBP, in.Residual, in.TransformBypass, int(in.Intra16x16PredMode), profileIDC); err != nil {
		return err
	}
	if dst.ChromaFormatIDC != 0 && in.CBP&0x30 != 0 {
		return h264HLDecodeMBIDCTChroma(dst.Cb, dst.Cr, dstCb, dstCr, dst.ChromaStride, blockOffset, dst.ChromaFormatIDC, in.MBType, in.CBP, in.ChromaQP, in.PPS, in.Residual, in.TransformBypass, int(in.ChromaPredMode), profileIDC)
	}
	return nil
}

func h264HLDecodeFrameMacroblock444(dst *h264PicturePlanes, dstY int, dstCb int, dstCr int, blockOffset *[48]int, in h264FrameMBReconstructInput) error {
	if blockOffset == nil {
		return ErrInvalidData
	}
	if in.MBType&MBTypeIntraPCM != 0 {
		return h264HLDecodeFrameIntraPCM(dst, dstY, dstCb, dstCr, in.IntraPCM)
	}
	if in.PPS == nil || in.Residual == nil {
		return ErrInvalidData
	}
	dest := [3][]uint8{dst.Y, dst.Cb, dst.Cr}
	offset := [3]int{dstY, dstCb, dstCr}
	stride := [3]int{dst.LumaStride, dst.ChromaStride, dst.ChromaStride}
	profileIDC := h264ProfileIDCFromPPS(in.PPS)
	if isIntra(in.MBType) {
		for p := 0; p < 3; p++ {
			if err := h264HLDecodeFrameIntraPredictLumaPlane(dest[p], offset[p], stride[p], blockOffset, in, p); err != nil {
				return err
			}
		}
	} else {
		if in.Motion == nil {
			return ErrInvalidData
		}
		if in.PredWeight != nil {
			weightMBY := in.MBY
			if in.UseMotionWeightMBY {
				weightMBY = in.MotionWeightMBY
			}
			if err := h264HLMotionFrameWeightedWithWeightY(dst, in.Refs, in.Motion, in.MBType, in.SubMBType, in.MBX, in.MBY, in.ListCount, weightMBY, in.PredWeight, in.MotionScratch); err != nil {
				return err
			}
		} else if err := h264HLMotionFrameWithScratch(dst, in.Refs, in.Motion, in.MBType, in.SubMBType, in.MBX, in.MBY, in.ListCount, in.MotionScratch); err != nil {
			return err
		}
	}
	for p := 0; p < 3; p++ {
		if err := h264HLDecodeMBIDCTLumaPlane(dest[p], offset[p], stride[p], blockOffset, in.MBType, in.CBP, in.Residual, p, in.TransformBypass, int(in.Intra16x16PredMode), profileIDC); err != nil {
			return err
		}
	}
	return nil
}

func h264HLDecodeFrameIntraPCM(dst *h264PicturePlanes, dstY int, dstCb int, dstCr int, pcm []byte) error {
	if dst == nil || dst.ChromaFormatIDC < 0 || dst.ChromaFormatIDC >= len(h264IntraPCMSampleCount) {
		return ErrUnsupported
	}
	required := h264IntraPCMSampleCount[dst.ChromaFormatIDC]
	if len(pcm) < required {
		return ErrInvalidData
	}
	if err := h264CopyRows(dst.Y, dstY, dst.LumaStride, 16, 16, pcm, 16); err != nil {
		return err
	}
	if dst.ChromaFormatIDC == 0 {
		return nil
	}
	if dst.ChromaFormatIDC == 3 {
		if err := h264CopyRows(dst.Cb, dstCb, dst.ChromaStride, 16, 16, pcm[256:], 16); err != nil {
			return err
		}
		return h264CopyRows(dst.Cr, dstCr, dst.ChromaStride, 16, 16, pcm[512:], 16)
	}
	blockH := 8
	if dst.ChromaFormatIDC == 2 {
		blockH = 16
	}
	srcCb := 256
	srcCr := srcCb + blockH*8
	if err := h264CopyRows(dst.Cb, dstCb, dst.ChromaStride, 8, blockH, pcm[srcCb:], 8); err != nil {
		return err
	}
	return h264CopyRows(dst.Cr, dstCr, dst.ChromaStride, 8, blockH, pcm[srcCr:], 8)
}

func h264CopyRows(dst []uint8, offset int, stride int, width int, height int, src []byte, srcStride int) error {
	if offset < 0 || stride <= 0 || width <= 0 || height <= 0 || srcStride < width {
		return ErrInvalidData
	}
	srcNeed, err := h264PlaneSpanLength(srcStride, height, width)
	if err != nil || len(src) < srcNeed {
		return ErrInvalidData
	}
	dstEnd, err := h264PlaneSpanEnd(offset, stride, height, width)
	if err != nil || len(dst) < dstEnd {
		return ErrInvalidData
	}
	for y := 0; y < height; y++ {
		copy(dst[offset+y*stride:offset+y*stride+width], src[y*srcStride:y*srcStride+width])
	}
	return nil
}

func h264HLDecodeFrameIntraPredict(dst *h264PicturePlanes, dstY int, dstCb int, dstCr int, blockOffset *[48]int, in h264FrameMBReconstructInput) error {
	if dst.ChromaFormatIDC != 0 {
		if err := h264PredChromaByMode(dst.Cb, dstCb, dst.ChromaStride, dst.ChromaFormatIDC, int(in.ChromaPredMode)); err != nil {
			return err
		}
		if err := h264PredChromaByMode(dst.Cr, dstCr, dst.ChromaStride, dst.ChromaFormatIDC, int(in.ChromaPredMode)); err != nil {
			return err
		}
	}
	if isIntra4x4(in.MBType) {
		return h264HLDecodeMBPredictLumaIntra4x4(dst.Y, dstY, dst.LumaStride, blockOffset, in.MBType, in.Intra4x4PredCache, in.TopLeftAvailable, in.TopRightAvailable, in.Residual, in.TransformBypass, h264ProfileIDCFromPPS(in.PPS), in.X264Build, in.X264BuildSet)
	}
	if !isIntra16x16(in.MBType) {
		return ErrUnsupported
	}
	return h264HLDecodeMBPredictLumaIntra16x16(dst.Y, dstY, dst.LumaStride, int(in.Intra16x16PredMode), in.QScale, in.PPS, in.Residual, in.TransformBypass)
}

func h264HLDecodeFrameIntraPredictLumaPlane(dest []uint8, baseOffset int, stride int, blockOffset *[48]int, in h264FrameMBReconstructInput, plane int) error {
	if isIntra4x4(in.MBType) {
		return h264HLDecodeMBPredictLumaIntra4x4Plane(dest, baseOffset, stride, blockOffset, in.MBType, in.Intra4x4PredCache, in.TopLeftAvailable, in.TopRightAvailable, in.Residual, plane, in.TransformBypass, h264ProfileIDCFromPPS(in.PPS), in.X264Build, in.X264BuildSet)
	}
	if !isIntra16x16(in.MBType) {
		return ErrUnsupported
	}
	qscale := in.QScale
	if plane > 0 {
		qscale = int(in.ChromaQP[plane-1])
	}
	return h264HLDecodeMBPredictLumaIntra16x16Plane(dest, baseOffset, stride, int(in.Intra16x16PredMode), qscale, in.PPS, in.Residual, plane, in.TransformBypass)
}

func h264HLDecodeMBPredictLumaIntra16x16(destY []uint8, offset int, stride int, predMode int, qscale int, pps *PPS, residual *cavlcResidualContext, transformBypass bool) error {
	return h264HLDecodeMBPredictLumaIntra16x16Plane(destY, offset, stride, predMode, qscale, pps, residual, 0, transformBypass)
}

func h264HLDecodeMBPredictLumaIntra16x16Plane(destY []uint8, offset int, stride int, predMode int, qscale int, pps *PPS, residual *cavlcResidualContext, plane int, transformBypass bool) error {
	if pps == nil || residual == nil || qscale < 0 || qscale > qpMaxNum {
		return ErrInvalidData
	}
	if plane < 0 || plane > 2 {
		return ErrInvalidData
	}
	if err := h264Pred16x16ByMode(destY, offset, stride, predMode); err != nil {
		return err
	}
	if residual.NonZeroCountCache[h264Scan8[lumaDCBlockIndex+plane]] != 0 {
		block := residual.MB[plane*16*16:]
		if transformBypass {
			h264LumaDCDirect(block[:16*16], &residual.MBLumaDC[plane])
		} else if err := h264LumaDCDequantIDCT(block[:16*16], &residual.MBLumaDC[plane], int(pps.Dequant4Buffer[plane][qscale][0])); err != nil {
			return err
		}
	}
	return nil
}

func h264HLDecodeMBIDCTLuma(destY []uint8, baseY int, stride int, blockOffset *[48]int, mbType uint32, cbp int, residual *cavlcResidualContext, transformBypass bool, intra16x16PredMode int, profileIDC int32) error {
	return h264HLDecodeMBIDCTLumaPlane(destY, baseY, stride, blockOffset, mbType, cbp, residual, 0, transformBypass, intra16x16PredMode, profileIDC)
}

func h264HLDecodeMBIDCTLumaPlane(destY []uint8, baseY int, stride int, blockOffset *[48]int, mbType uint32, cbp int, residual *cavlcResidualContext, plane int, transformBypass bool, intra16x16PredMode int, profileIDC int32) error {
	if residual == nil {
		return ErrInvalidData
	}
	if baseY < 0 || baseY > len(destY) {
		return ErrInvalidData
	}
	if plane < 0 || plane > 2 {
		return ErrInvalidData
	}
	destMB := destY[baseY:]
	if isIntra4x4(mbType) {
		return nil
	}
	if isIntra16x16(mbType) {
		if transformBypass {
			if profileIDC == 244 && intra16x16PredMode == intraPred8x8Vertical {
				return h264Pred16x16VerticalAddAt(destY, baseY, blockOffset, plane*16, residual.MB[plane*16*16:], stride)
			}
			if profileIDC == 244 && intra16x16PredMode == intraPred8x8Horizontal {
				return h264Pred16x16HorizontalAddAt(destY, baseY, blockOffset, plane*16, residual.MB[plane*16*16:], stride)
			}
			return h264AddPixels16BypassPlane(destMB, blockOffset, residual.MB[:], stride, &residual.NonZeroCountCache, plane, true)
		}
		return h264IDCTAdd16IntraPlane(destMB, blockOffset, residual.MB[:], stride, &residual.NonZeroCountCache, plane)
	}
	if cbp&15 == 0 {
		return nil
	}
	if transformBypass {
		if is8x8DCT(mbType) {
			return h264AddPixels8Bypass4Plane(destMB, blockOffset, residual.MB[:], stride, &residual.NonZeroCountCache, plane)
		}
		return h264AddPixels16BypassPlane(destMB, blockOffset, residual.MB[:], stride, &residual.NonZeroCountCache, plane, false)
	}
	if is8x8DCT(mbType) {
		return h264IDCT8Add4Plane(destMB, blockOffset, residual.MB[:], stride, &residual.NonZeroCountCache, plane)
	}
	return h264IDCTAdd16Plane(destMB, blockOffset, residual.MB[:], stride, &residual.NonZeroCountCache, plane)
}

func h264HLDecodeMBPredictLumaIntra4x4(destY []uint8, baseOffset int, stride int, blockOffset *[48]int, mbType uint32, predCache *[h264IntraPredModeCacheSize]int8, topLeftAvailable uint16, topRightAvailable uint16, residual *cavlcResidualContext, transformBypass bool, profileIDC int32, x264Build int32, x264BuildSet bool) error {
	return h264HLDecodeMBPredictLumaIntra4x4Plane(destY, baseOffset, stride, blockOffset, mbType, predCache, topLeftAvailable, topRightAvailable, residual, 0, transformBypass, profileIDC, x264Build, x264BuildSet)
}

func h264HLDecodeMBPredictLumaIntra4x4Plane(destY []uint8, baseOffset int, stride int, blockOffset *[48]int, mbType uint32, predCache *[h264IntraPredModeCacheSize]int8, topLeftAvailable uint16, topRightAvailable uint16, residual *cavlcResidualContext, plane int, transformBypass bool, profileIDC int32, x264Build int32, x264BuildSet bool) error {
	if blockOffset == nil || predCache == nil || residual == nil {
		return ErrInvalidData
	}
	if plane < 0 || plane > 2 {
		return ErrInvalidData
	}
	planeBlock := 16 * plane
	if is8x8DCT(mbType) {
		for i := 0; i < 16; i += 4 {
			index := i + planeBlock
			offset := baseOffset + blockOffset[index]
			dir := int(predCache[h264Scan8[i]])
			hasTopLeft := ((uint32(topLeftAvailable) << uint(i)) & 0x8000) != 0
			hasTopRight := ((uint32(topRightAvailable) << uint(i)) & 0x4000) != 0
			block := residual.MB[index*16 : index*16+64]
			if transformBypass && profileIDC == 244 && (dir == int(intraPredVertical) || dir == int(intraPredHorizontal)) {
				if h264X264BuildUsesUnfiltered8x8LAdd(x264Build, x264BuildSet) {
					if dir == int(intraPredVertical) {
						if err := h264Pred8x8LVerticalAdd(destY, offset, block, stride); err != nil {
							return err
						}
					} else if err := h264Pred8x8LHorizontalAdd(destY, offset, block, stride); err != nil {
						return err
					}
				} else if dir == int(intraPredVertical) {
					if err := h264Pred8x8LVerticalFilterAdd(destY, offset, block, stride, hasTopLeft, hasTopRight); err != nil {
						return err
					}
				} else if err := h264Pred8x8LHorizontalFilterAdd(destY, offset, block, stride, hasTopLeft, hasTopRight); err != nil {
					return err
				}
				continue
			}
			if err := h264Pred8x8LByMode(destY, offset, stride, dir, hasTopLeft, hasTopRight); err != nil {
				return err
			}
			nnz := residual.NonZeroCountCache[h264Scan8[index]]
			if nnz == 0 {
				continue
			}
			if transformBypass {
				if err := h264AddPixels8Clear(destY[offset:], block, stride); err != nil {
					return err
				}
			} else if nnz == 1 && dctcoef8Value(block[0]) != 0 {
				if err := h264IDCT8DCAdd(destY[offset:], block, stride); err != nil {
					return err
				}
			} else if err := h264IDCT8Add(destY[offset:], block, stride); err != nil {
				return err
			}
		}
		return nil
	}

	var unavailableTopRight [4]uint8
	for i := 0; i < 16; i++ {
		index := i + planeBlock
		offset := baseOffset + blockOffset[index]
		dir := int(predCache[h264Scan8[i]])
		var topRight []uint8
		if dir == int(intraPredDiagDownLeft) || dir == int(intraPredVertLeft) {
			hasTopRight := ((uint32(topRightAvailable) << uint(i)) & 0x8000) != 0
			if hasTopRight {
				start := offset + 4 - stride
				if start < 0 || start+4 > len(destY) {
					return ErrInvalidData
				}
				topRight = destY[start : start+4]
			} else {
				index := offset + 3 - stride
				if index < 0 || index >= len(destY) {
					return ErrInvalidData
				}
				for j := range unavailableTopRight {
					unavailableTopRight[j] = destY[index]
				}
				topRight = unavailableTopRight[:]
			}
		}
		nnz := residual.NonZeroCountCache[h264Scan8[index]]
		block := residual.MB[index*16 : index*16+16]
		if transformBypass && profileIDC == 244 && (dir == int(intraPredVertical) || dir == int(intraPredHorizontal)) {
			if dir == int(intraPredVertical) {
				if err := h264Pred4x4VerticalAdd(destY, offset, block, stride); err != nil {
					return err
				}
			} else if err := h264Pred4x4HorizontalAdd(destY, offset, block, stride); err != nil {
				return err
			}
			continue
		}
		if err := h264Pred4x4ByMode(destY, offset, stride, dir, topRight); err != nil {
			return err
		}
		if nnz == 0 {
			continue
		}
		if transformBypass {
			if err := h264AddPixels4Clear(destY[offset:], block, stride); err != nil {
				return err
			}
		} else if nnz == 1 && dctcoef8Value(block[0]) != 0 {
			if err := h264IDCTDCAdd(destY[offset:], block, stride); err != nil {
				return err
			}
		} else if err := h264IDCTAdd(destY[offset:], block, stride); err != nil {
			return err
		}
	}
	return nil
}

func h264IDCTAdd16Plane(dst []uint8, blockOffset *[48]int, block []int32, stride int, nnzc *[h264NonZeroCountCacheSize]uint8, plane int) error {
	if blockOffset == nil || nnzc == nil || plane < 0 || plane > 2 || len(block) < (plane+1)*16*16 {
		return ErrInvalidData
	}
	planeBlock := 16 * plane
	for i := 0; i < 16; i++ {
		index := i + planeBlock
		nnz := nnzc[h264Scan8[index]]
		if nnz == 0 {
			continue
		}
		dstBlock, err := transformBlockDestination(dst, blockOffset[index], stride, 4)
		if err != nil {
			return err
		}
		coef := block[index*16 : index*16+16]
		if nnz == 1 && dctcoef8Value(coef[0]) != 0 {
			if err := h264IDCTDCAdd(dstBlock, coef, stride); err != nil {
				return err
			}
		} else if err := h264IDCTAdd(dstBlock, coef, stride); err != nil {
			return err
		}
	}
	return nil
}

func h264IDCTAdd16IntraPlane(dst []uint8, blockOffset *[48]int, block []int32, stride int, nnzc *[h264NonZeroCountCacheSize]uint8, plane int) error {
	if blockOffset == nil || nnzc == nil || plane < 0 || plane > 2 || len(block) < (plane+1)*16*16 {
		return ErrInvalidData
	}
	planeBlock := 16 * plane
	for i := 0; i < 16; i++ {
		index := i + planeBlock
		coef := block[index*16 : index*16+16]
		if nnzc[h264Scan8[index]] != 0 {
			dstBlock, err := transformBlockDestination(dst, blockOffset[index], stride, 4)
			if err != nil {
				return err
			}
			if err := h264IDCTAdd(dstBlock, coef, stride); err != nil {
				return err
			}
		} else if dctcoef8Value(coef[0]) != 0 {
			dstBlock, err := transformBlockDestination(dst, blockOffset[index], stride, 4)
			if err != nil {
				return err
			}
			if err := h264IDCTDCAdd(dstBlock, coef, stride); err != nil {
				return err
			}
		}
	}
	return nil
}

func h264IDCT8Add4Plane(dst []uint8, blockOffset *[48]int, block []int32, stride int, nnzc *[h264NonZeroCountCacheSize]uint8, plane int) error {
	if blockOffset == nil || nnzc == nil || plane < 0 || plane > 2 || len(block) < (plane+1)*16*16 {
		return ErrInvalidData
	}
	planeBlock := 16 * plane
	for i := 0; i < 16; i += 4 {
		index := i + planeBlock
		nnz := nnzc[h264Scan8[index]]
		if nnz == 0 {
			continue
		}
		dstBlock, err := transformBlockDestination(dst, blockOffset[index], stride, 8)
		if err != nil {
			return err
		}
		coef := block[index*16 : index*16+64]
		if nnz == 1 && dctcoef8Value(coef[0]) != 0 {
			if err := h264IDCT8DCAdd(dstBlock, coef, stride); err != nil {
				return err
			}
		} else if err := h264IDCT8Add(dstBlock, coef, stride); err != nil {
			return err
		}
	}
	return nil
}

func h264AddPixels16BypassPlane(dst []uint8, blockOffset *[48]int, block []int32, stride int, nnzc *[h264NonZeroCountCacheSize]uint8, plane int, includeDCOnly bool) error {
	if blockOffset == nil || nnzc == nil || plane < 0 || plane > 2 || len(block) < (plane+1)*16*16 {
		return ErrInvalidData
	}
	planeBlock := 16 * plane
	for i := 0; i < 16; i++ {
		index := i + planeBlock
		coef := block[index*16 : index*16+16]
		if nnzc[h264Scan8[index]] == 0 && (!includeDCOnly || dctcoef8Value(coef[0]) == 0) {
			continue
		}
		dstBlock, err := transformBlockDestination(dst, blockOffset[index], stride, 4)
		if err != nil {
			return err
		}
		if err := h264AddPixels4Clear(dstBlock, coef, stride); err != nil {
			return err
		}
	}
	return nil
}

func h264AddPixels8Bypass4Plane(dst []uint8, blockOffset *[48]int, block []int32, stride int, nnzc *[h264NonZeroCountCacheSize]uint8, plane int) error {
	if blockOffset == nil || nnzc == nil || plane < 0 || plane > 2 || len(block) < (plane+1)*16*16 {
		return ErrInvalidData
	}
	planeBlock := 16 * plane
	for i := 0; i < 16; i += 4 {
		index := i + planeBlock
		if nnzc[h264Scan8[index]] == 0 {
			continue
		}
		dstBlock, err := transformBlockDestination(dst, blockOffset[index], stride, 8)
		if err != nil {
			return err
		}
		if err := h264AddPixels8Clear(dstBlock, block[index*16:index*16+64], stride); err != nil {
			return err
		}
	}
	return nil
}

func h264LumaDCDirect(output []int32, input *[16]int32) {
	dcMapping := [16]uint8{
		0 * 16, 1 * 16, 4 * 16, 5 * 16,
		2 * 16, 3 * 16, 6 * 16, 7 * 16,
		8 * 16, 9 * 16, 12 * 16, 13 * 16,
		10 * 16, 11 * 16, 14 * 16, 15 * 16,
	}
	if input == nil || len(output) < 16*16 {
		return
	}
	for i := 0; i < 16; i++ {
		output[dcMapping[i]] = input[i]
	}
}

func h264HLDecodeMBIDCTChroma(destCb []uint8, destCr []uint8, baseCb int, baseCr int, stride int, blockOffset *[48]int, chromaFormatIDC int, mbType uint32, cbp int, chromaQP [2]uint8, pps *PPS, residual *cavlcResidualContext, transformBypass bool, chromaPredMode int, profileIDC int32) error {
	if pps == nil || residual == nil {
		return ErrInvalidData
	}
	if baseCb < 0 || baseCb > len(destCb) || baseCr < 0 || baseCr > len(destCr) {
		return ErrInvalidData
	}
	if cbp&0x30 == 0 {
		return nil
	}
	qp0 := int(chromaQP[0])
	qp1 := int(chromaQP[1])
	if qp0 > qpMaxNum || qp1 > qpMaxNum {
		return ErrInvalidData
	}
	if chromaFormatIDC == 2 {
		qp0 += 3
		qp1 += 3
		if qp0 > qpMaxNum || qp1 > qpMaxNum {
			return ErrInvalidData
		}
	}
	cqm0, cqm1 := 4, 5
	if isIntra(mbType) {
		cqm0, cqm1 = 1, 2
	}
	if transformBypass {
		return h264HLDecodeMBAddChromaBypass(destCb, destCr, baseCb, baseCr, stride, blockOffset, chromaFormatIDC, mbType, chromaPredMode, profileIDC, residual)
	}
	if residual.NonZeroCountCache[h264Scan8[chromaDCBlockIndex+0]] != 0 {
		if err := h264ChromaDCDequantIDCTByFormat(residual.MB[16*16:], int(pps.Dequant4Buffer[cqm0][qp0][0]), chromaFormatIDC); err != nil {
			return err
		}
	}
	if residual.NonZeroCountCache[h264Scan8[chromaDCBlockIndex+1]] != 0 {
		if err := h264ChromaDCDequantIDCTByFormat(residual.MB[16*16*2:], int(pps.Dequant4Buffer[cqm1][qp1][0]), chromaFormatIDC); err != nil {
			return err
		}
	}
	dest := [2][]uint8{destCb[baseCb:], destCr[baseCr:]}
	if chromaFormatIDC == 2 {
		return h264IDCTAdd8_422(&dest, blockOffset, residual.MB[:], stride, &residual.NonZeroCountCache)
	}
	if chromaFormatIDC == 1 {
		return h264IDCTAdd8(&dest, blockOffset, residual.MB[:], stride, &residual.NonZeroCountCache)
	}
	return ErrInvalidData
}

func h264HLDecodeMBAddChromaBypass(destCb []uint8, destCr []uint8, baseCb int, baseCr int, stride int, blockOffset *[48]int, chromaFormatIDC int, mbType uint32, chromaPredMode int, profileIDC int32, residual *cavlcResidualContext) error {
	if blockOffset == nil || residual == nil {
		return ErrInvalidData
	}
	if baseCb < 0 || baseCb > len(destCb) || baseCr < 0 || baseCr > len(destCr) {
		return ErrInvalidData
	}
	dest := [2][]uint8{destCb, destCr}
	base := [2]int{baseCb, baseCr}
	if isIntra(mbType) && profileIDC == 244 && (chromaPredMode == intraPred8x8Vertical || chromaPredMode == intraPred8x8Horizontal) {
		for plane := 0; plane < 2; plane++ {
			baseBlock := 16 + plane*16
			block := residual.MB[baseBlock*16:]
			if chromaFormatIDC == 1 {
				if chromaPredMode == intraPred8x8Vertical {
					if err := h264Pred8x8VerticalAddAt(dest[plane], base[plane], blockOffset, baseBlock, block, stride); err != nil {
						return err
					}
				} else if err := h264Pred8x8HorizontalAddAt(dest[plane], base[plane], blockOffset, baseBlock, block, stride); err != nil {
					return err
				}
			} else if chromaFormatIDC == 2 {
				if chromaPredMode == intraPred8x8Vertical {
					if err := h264Pred8x16VerticalAddAt(dest[plane], base[plane], blockOffset, baseBlock, block, stride); err != nil {
						return err
					}
				} else if err := h264Pred8x16HorizontalAddAt(dest[plane], base[plane], blockOffset, baseBlock, block, stride); err != nil {
					return err
				}
			} else {
				return ErrInvalidData
			}
		}
		return nil
	}

	for plane := 0; plane < 2; plane++ {
		j := plane + 1
		for i := j * 16; i < j*16+4; i++ {
			if residual.NonZeroCountCache[h264Scan8[i]] == 0 && dctcoef8Value(residual.MB[i*16]) == 0 {
				continue
			}
			dstBlock, err := transformBlockDestination(dest[plane], base[plane]+blockOffset[i], stride, 4)
			if err != nil {
				return err
			}
			if err := h264AddPixels4Clear(dstBlock, residual.MB[i*16:i*16+16], stride); err != nil {
				return err
			}
		}
		if chromaFormatIDC == 2 {
			for i := j*16 + 4; i < j*16+8; i++ {
				if residual.NonZeroCountCache[h264Scan8[i+4]] == 0 && dctcoef8Value(residual.MB[i*16]) == 0 {
					continue
				}
				dstBlock, err := transformBlockDestination(dest[plane], base[plane]+blockOffset[i+4], stride, 4)
				if err != nil {
					return err
				}
				if err := h264AddPixels4Clear(dstBlock, residual.MB[i*16:i*16+16], stride); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func h264Pred8x8VerticalAddAt(pix []uint8, pixBase int, blockOffset *[48]int, offsetBase int, block []int32, stride int) error {
	if blockOffset == nil || offsetBase < 0 || offsetBase+4 > len(blockOffset) || len(block) < 4*16 {
		return ErrInvalidData
	}
	for i := 0; i < 4; i++ {
		if err := h264Pred4x4VerticalAdd(pix, pixBase+blockOffset[offsetBase+i], block[i*16:i*16+16], stride); err != nil {
			return err
		}
	}
	return nil
}

func h264Pred8x8HorizontalAddAt(pix []uint8, pixBase int, blockOffset *[48]int, offsetBase int, block []int32, stride int) error {
	if blockOffset == nil || offsetBase < 0 || offsetBase+4 > len(blockOffset) || len(block) < 4*16 {
		return ErrInvalidData
	}
	for i := 0; i < 4; i++ {
		if err := h264Pred4x4HorizontalAdd(pix, pixBase+blockOffset[offsetBase+i], block[i*16:i*16+16], stride); err != nil {
			return err
		}
	}
	return nil
}

func h264Pred16x16VerticalAddAt(pix []uint8, pixBase int, blockOffset *[48]int, offsetBase int, block []int32, stride int) error {
	if blockOffset == nil || offsetBase < 0 || offsetBase+16 > len(blockOffset) || len(block) < 16*16 {
		return ErrInvalidData
	}
	for i := 0; i < 16; i++ {
		if err := h264Pred4x4VerticalAdd(pix, pixBase+blockOffset[offsetBase+i], block[i*16:i*16+16], stride); err != nil {
			return err
		}
	}
	return nil
}

func h264Pred16x16HorizontalAddAt(pix []uint8, pixBase int, blockOffset *[48]int, offsetBase int, block []int32, stride int) error {
	if blockOffset == nil || offsetBase < 0 || offsetBase+16 > len(blockOffset) || len(block) < 16*16 {
		return ErrInvalidData
	}
	for i := 0; i < 16; i++ {
		if err := h264Pred4x4HorizontalAdd(pix, pixBase+blockOffset[offsetBase+i], block[i*16:i*16+16], stride); err != nil {
			return err
		}
	}
	return nil
}

func h264Pred8x16VerticalAddAt(pix []uint8, pixBase int, blockOffset *[48]int, offsetBase int, block []int32, stride int) error {
	if blockOffset == nil || offsetBase < 0 || offsetBase+12 > len(blockOffset) || len(block) < 8*16 {
		return ErrInvalidData
	}
	for i := 0; i < 4; i++ {
		if err := h264Pred4x4VerticalAdd(pix, pixBase+blockOffset[offsetBase+i], block[i*16:i*16+16], stride); err != nil {
			return err
		}
	}
	for i := 4; i < 8; i++ {
		if err := h264Pred4x4VerticalAdd(pix, pixBase+blockOffset[offsetBase+i+4], block[i*16:i*16+16], stride); err != nil {
			return err
		}
	}
	return nil
}

func h264Pred8x16HorizontalAddAt(pix []uint8, pixBase int, blockOffset *[48]int, offsetBase int, block []int32, stride int) error {
	if blockOffset == nil || offsetBase < 0 || offsetBase+12 > len(blockOffset) || len(block) < 8*16 {
		return ErrInvalidData
	}
	for i := 0; i < 4; i++ {
		if err := h264Pred4x4HorizontalAdd(pix, pixBase+blockOffset[offsetBase+i], block[i*16:i*16+16], stride); err != nil {
			return err
		}
	}
	for i := 4; i < 8; i++ {
		if err := h264Pred4x4HorizontalAdd(pix, pixBase+blockOffset[offsetBase+i+4], block[i*16:i*16+16], stride); err != nil {
			return err
		}
	}
	return nil
}

func h264ChromaDCDequantIDCTByFormat(block []int32, qmul int, chromaFormatIDC int) error {
	if chromaFormatIDC == 2 {
		return h264Chroma422DCDequantIDCT(block, qmul)
	}
	if chromaFormatIDC == 1 {
		return h264ChromaDCDequantIDCT(block, qmul)
	}
	return ErrInvalidData
}

func h264Pred16x16ByMode(pix []uint8, offset int, stride int, mode int) error {
	switch mode {
	case intraPred8x8DC:
		return h264Pred16x16DC(pix, offset, stride)
	case intraPred8x8Horizontal:
		return h264Pred16x16Horizontal(pix, offset, stride)
	case intraPred8x8Vertical:
		return h264Pred16x16Vertical(pix, offset, stride)
	case intraPred8x8Plane:
		return h264Pred16x16Plane(pix, offset, stride)
	case intraPred8x8LeftDC:
		return h264Pred16x16LeftDC(pix, offset, stride)
	case intraPred8x8TopDC:
		return h264Pred16x16TopDC(pix, offset, stride)
	case intraPredDC1288x8:
		return h264Pred16x16DC128(pix, offset, stride)
	default:
		return ErrInvalidData
	}
}

func h264Pred4x4ByMode(pix []uint8, offset int, stride int, mode int, topRight []uint8) error {
	switch int8(mode) {
	case intraPredVertical:
		return h264Pred4x4Vertical(pix, offset, stride)
	case intraPredHorizontal:
		return h264Pred4x4Horizontal(pix, offset, stride)
	case intraPredDC:
		return h264Pred4x4DC(pix, offset, stride)
	case intraPredDiagDownLeft:
		return h264Pred4x4DownLeft(pix, offset, stride, topRight)
	case intraPredDiagDownRight:
		return h264Pred4x4DownRight(pix, offset, stride)
	case intraPredVertRight:
		return h264Pred4x4VerticalRight(pix, offset, stride)
	case intraPredHorDown:
		return h264Pred4x4HorizontalDown(pix, offset, stride)
	case intraPredVertLeft:
		return h264Pred4x4VerticalLeft(pix, offset, stride, topRight)
	case intraPredHorUp:
		return h264Pred4x4HorizontalUp(pix, offset, stride)
	case intraPredLeftDC:
		return h264Pred4x4LeftDC(pix, offset, stride)
	case intraPredTopDC:
		return h264Pred4x4TopDC(pix, offset, stride)
	case intraPredDC128:
		return h264Pred4x4DC128(pix, offset, stride)
	default:
		return ErrInvalidData
	}
}

func h264Pred8x8LByMode(pix []uint8, offset int, stride int, mode int, hasTopLeft bool, hasTopRight bool) error {
	switch int8(mode) {
	case intraPredVertical:
		return h264Pred8x8LVertical(pix, offset, stride, hasTopLeft, hasTopRight)
	case intraPredHorizontal:
		return h264Pred8x8LHorizontal(pix, offset, stride, hasTopLeft, hasTopRight)
	case intraPredDC:
		return h264Pred8x8LDC(pix, offset, stride, hasTopLeft, hasTopRight)
	case intraPredDiagDownLeft:
		return h264Pred8x8LDownLeft(pix, offset, stride, hasTopLeft, hasTopRight)
	case intraPredDiagDownRight:
		return h264Pred8x8LDownRight(pix, offset, stride, hasTopLeft, hasTopRight)
	case intraPredVertRight:
		return h264Pred8x8LVerticalRight(pix, offset, stride, hasTopLeft, hasTopRight)
	case intraPredHorDown:
		return h264Pred8x8LHorizontalDown(pix, offset, stride, hasTopLeft, hasTopRight)
	case intraPredVertLeft:
		return h264Pred8x8LVerticalLeft(pix, offset, stride, hasTopLeft, hasTopRight)
	case intraPredHorUp:
		return h264Pred8x8LHorizontalUp(pix, offset, stride, hasTopLeft, hasTopRight)
	case intraPredLeftDC:
		return h264Pred8x8LLeftDC(pix, offset, stride, hasTopLeft, hasTopRight)
	case intraPredTopDC:
		return h264Pred8x8LTopDC(pix, offset, stride, hasTopLeft, hasTopRight)
	case intraPredDC128:
		return h264Pred8x8LDC128(pix, offset, stride, hasTopLeft, hasTopRight)
	default:
		return ErrInvalidData
	}
}

func h264PredChromaByMode(pix []uint8, offset int, stride int, chromaFormatIDC int, mode int) error {
	if chromaFormatIDC == 2 {
		return h264Pred8x16ByMode(pix, offset, stride, mode)
	}
	if chromaFormatIDC == 1 {
		return h264Pred8x8ByMode(pix, offset, stride, mode)
	}
	return ErrInvalidData
}

func h264Pred8x8ByMode(pix []uint8, offset int, stride int, mode int) error {
	switch mode {
	case intraPred8x8DC:
		return h264Pred8x8DC(pix, offset, stride)
	case intraPred8x8Horizontal:
		return h264Pred8x8Horizontal(pix, offset, stride)
	case intraPred8x8Vertical:
		return h264Pred8x8Vertical(pix, offset, stride)
	case intraPred8x8Plane:
		return h264Pred8x8Plane(pix, offset, stride)
	case intraPred8x8LeftDC:
		return h264Pred8x8LeftDC(pix, offset, stride)
	case intraPred8x8TopDC:
		return h264Pred8x8TopDC(pix, offset, stride)
	case intraPredDC1288x8:
		return h264Pred8x8DC128(pix, offset, stride)
	case intraPred8x8AlzheimerL0TDC:
		return h264Pred8x8MadCowDCL0T(pix, offset, stride)
	case intraPred8x8Alzheimer0LTDC:
		return h264Pred8x8MadCowDC0LT(pix, offset, stride)
	case intraPred8x8AlzheimerL00DC:
		return h264Pred8x8MadCowDCL00(pix, offset, stride)
	case intraPred8x8Alzheimer0L0DC:
		return h264Pred8x8MadCowDC0L0(pix, offset, stride)
	default:
		return ErrUnsupported
	}
}

func h264Pred8x16ByMode(pix []uint8, offset int, stride int, mode int) error {
	switch mode {
	case intraPred8x8DC:
		return h264Pred8x16DC(pix, offset, stride)
	case intraPred8x8Horizontal:
		return h264Pred8x16Horizontal(pix, offset, stride)
	case intraPred8x8Vertical:
		return h264Pred8x16Vertical(pix, offset, stride)
	case intraPred8x8Plane:
		return h264Pred8x16Plane(pix, offset, stride)
	case intraPred8x8LeftDC:
		return h264Pred8x16LeftDC(pix, offset, stride)
	case intraPred8x8TopDC:
		return h264Pred8x16TopDC(pix, offset, stride)
	case intraPredDC1288x8:
		return h264Pred8x16DC128(pix, offset, stride)
	case intraPred8x8AlzheimerL0TDC:
		return h264Pred8x16MadCowDCL0T(pix, offset, stride)
	case intraPred8x8Alzheimer0LTDC:
		return h264Pred8x16MadCowDC0LT(pix, offset, stride)
	case intraPred8x8AlzheimerL00DC:
		return h264Pred8x16MadCowDCL00(pix, offset, stride)
	case intraPred8x8Alzheimer0L0DC:
		return h264Pred8x16MadCowDC0L0(pix, offset, stride)
	default:
		return ErrUnsupported
	}
}
