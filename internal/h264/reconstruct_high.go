// SPDX-License-Identifier: LGPL-2.1-or-later
//
// Source-shaped high-bit-depth frame-MB reconstruction helpers from FFmpeg
// n8.0.1 libavcodec/h264_mb.c hl_decode_mb_predict_luma,
// hl_decode_mb_idct_luma, and h264_mb_template.c hl_decode_mb.

package h264

type h264PicturePlanesHigh struct {
	Y, Cb, Cr       []uint16
	LumaStride      int
	ChromaStride    int
	MBWidth         int
	MBHeight        int
	ChromaFormatIDC int
}

type h264FrameMBReconstructInputHigh struct {
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
	Refs                [2][]*h264PicturePlanesHigh
	PredWeight          *PredWeightTable
	MotionWeightMBY     int
	UseMotionWeightMBY  bool
	MotionScratch       *h264MotionCompScratchHigh
	TransformBypass     bool
	DeblockingFilter    bool
	ConstrainedIntra444 bool
	BitDepth            int
	IntraPCM            []byte
}

func h264MaxQPForBitDepth(bitDepth int) int {
	return 51 + 6*(bitDepth-8)
}

func h264HLDecodeFrameMacroblockHigh(dst *h264PicturePlanesHigh, in h264FrameMBReconstructInputHigh) error {
	if err := checkH264DSPHighBitDepth(in.BitDepth); err != nil {
		return err
	}
	if dst == nil || in.MBX < 0 || in.MBY < 0 || in.QScale < 0 || in.QScale > h264MaxQPForBitDepth(in.BitDepth) {
		return ErrInvalidData
	}
	if in.ConstrainedIntra444 {
		return ErrUnsupported
	}
	if err := dst.validate(); err != nil {
		return err
	}
	if in.MBX >= dst.MBWidth || in.MBY >= dst.MBHeight {
		return ErrInvalidData
	}

	chromaStride := dst.ChromaStride
	if chromaStride == 0 {
		chromaStride = 1
	}
	blockOffset, err := h264FrameBlockOffsets(dst.LumaStride, chromaStride, 0)
	if err != nil {
		return err
	}
	dstY, dstCb, dstCr, err := h264MBDestPartOffsetsHigh(dst, in.MBX, in.MBY, 0, 0)
	if err != nil {
		return err
	}
	if dst.ChromaFormatIDC == 3 {
		return h264HLDecodeFrameMacroblock444High(dst, dstY, dstCb, dstCr, &blockOffset, in)
	}

	if in.MBType&MBTypeIntraPCM != 0 {
		return h264HLDecodeFrameIntraPCMHigh(dst, dstY, dstCb, dstCr, in.IntraPCM, in.BitDepth)
	}
	if in.PPS == nil || in.Residual == nil {
		return ErrInvalidData
	}
	if isIntra(in.MBType) {
		if err := h264HLDecodeFrameIntraPredictHigh(dst, dstY, dstCb, dstCr, &blockOffset, in); err != nil {
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
			if err := h264HLMotionFrameWeightedHighWithWeightY(dst, in.Refs, in.Motion, in.MBType, in.SubMBType, in.MBX, in.MBY, in.ListCount, weightMBY, in.PredWeight, in.MotionScratch, in.BitDepth); err != nil {
				return err
			}
		} else if err := h264HLMotionFrameWithScratchHigh(dst, in.Refs, in.Motion, in.MBType, in.SubMBType, in.MBX, in.MBY, in.ListCount, in.MotionScratch, in.BitDepth); err != nil {
			return err
		}
	}

	profileIDC := h264ProfileIDCFromPPS(in.PPS)
	if err := h264HLDecodeMBIDCTLumaHigh(dst.Y[dstY:], dst.LumaStride, &blockOffset, in.MBType, in.CBP, in.Residual, in.TransformBypass, int(in.Intra16x16PredMode), profileIDC, in.BitDepth); err != nil {
		return err
	}
	if dst.ChromaFormatIDC != 0 && in.CBP&0x30 != 0 {
		return h264HLDecodeMBIDCTChromaHigh(dst.Cb[dstCb:], dst.Cr[dstCr:], dst.ChromaStride, &blockOffset, dst.ChromaFormatIDC, in.MBType, in.CBP, in.ChromaQP, in.PPS, in.Residual, in.TransformBypass, int(in.ChromaPredMode), profileIDC, in.BitDepth)
	}
	return nil
}

func h264HLDecodeFrameMacroblock444High(dst *h264PicturePlanesHigh, dstY int, dstCb int, dstCr int, blockOffset *[48]int, in h264FrameMBReconstructInputHigh) error {
	if blockOffset == nil {
		return ErrInvalidData
	}
	if in.MBType&MBTypeIntraPCM != 0 {
		return h264HLDecodeFrameIntraPCMHigh(dst, dstY, dstCb, dstCr, in.IntraPCM, in.BitDepth)
	}
	if in.PPS == nil || in.Residual == nil {
		return ErrInvalidData
	}
	dest := [3][]uint16{dst.Y, dst.Cb, dst.Cr}
	offset := [3]int{dstY, dstCb, dstCr}
	stride := [3]int{dst.LumaStride, dst.ChromaStride, dst.ChromaStride}
	profileIDC := h264ProfileIDCFromPPS(in.PPS)
	if isIntra(in.MBType) {
		for p := 0; p < 3; p++ {
			if err := h264HLDecodeFrameIntraPredictLumaPlaneHigh(dest[p], offset[p], stride[p], blockOffset, in, p); err != nil {
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
			if err := h264HLMotionFrameWeightedHighWithWeightY(dst, in.Refs, in.Motion, in.MBType, in.SubMBType, in.MBX, in.MBY, in.ListCount, weightMBY, in.PredWeight, in.MotionScratch, in.BitDepth); err != nil {
				return err
			}
		} else if err := h264HLMotionFrameWithScratchHigh(dst, in.Refs, in.Motion, in.MBType, in.SubMBType, in.MBX, in.MBY, in.ListCount, in.MotionScratch, in.BitDepth); err != nil {
			return err
		}
	}
	for p := 0; p < 3; p++ {
		if err := h264HLDecodeMBIDCTLumaPlaneHigh(dest[p][offset[p]:], stride[p], blockOffset, in.MBType, in.CBP, in.Residual, p, in.TransformBypass, int(in.Intra16x16PredMode), profileIDC, in.BitDepth); err != nil {
			return err
		}
	}
	return nil
}

func h264IntraPCMBitCount(chromaFormatIDC int, bitDepth int) (int, error) {
	if chromaFormatIDC < 0 || chromaFormatIDC >= len(h264IntraPCMSampleCount) {
		return 0, ErrInvalidData
	}
	switch bitDepth {
	case 8, 9, 10, 12, 14:
	default:
		return 0, ErrUnsupported
	}
	return h264IntraPCMSampleCount[chromaFormatIDC] * bitDepth, nil
}

func h264IntraPCMByteCount(chromaFormatIDC int, bitDepth int) (int, error) {
	bits, err := h264IntraPCMBitCount(chromaFormatIDC, bitDepth)
	if err != nil {
		return 0, err
	}
	return (bits + 7) >> 3, nil
}

func h264HLDecodeFrameIntraPCMHigh(dst *h264PicturePlanesHigh, dstY int, dstCb int, dstCr int, pcm []byte, bitDepth int) error {
	if dst == nil {
		return ErrInvalidData
	}
	if err := checkH264DSPHighBitDepth(bitDepth); err != nil {
		return err
	}
	if err := dst.validate(); err != nil {
		return err
	}
	bitCount, err := h264IntraPCMBitCount(dst.ChromaFormatIDC, bitDepth)
	if err != nil {
		return err
	}
	byteCount, err := h264IntraPCMByteCount(dst.ChromaFormatIDC, bitDepth)
	if err != nil {
		return err
	}
	if len(pcm) < byteCount {
		return ErrInvalidData
	}
	gb := newBitReader(pcm[:byteCount])
	gb.numBits = uint32(bitCount)

	if err := h264ReadIntraPCMPlaneHigh(&gb, dst.Y, dstY, dst.LumaStride, 16, 16, bitDepth); err != nil {
		return err
	}
	if dst.ChromaFormatIDC == 0 {
		return nil
	}
	chromaWidth, chromaHeight := 8, 8
	if dst.ChromaFormatIDC == 2 {
		chromaHeight = 16
	} else if dst.ChromaFormatIDC == 3 {
		chromaWidth = 16
		chromaHeight = 16
	}
	if err := h264ReadIntraPCMPlaneHigh(&gb, dst.Cb, dstCb, dst.ChromaStride, chromaWidth, chromaHeight, bitDepth); err != nil {
		return err
	}
	return h264ReadIntraPCMPlaneHigh(&gb, dst.Cr, dstCr, dst.ChromaStride, chromaWidth, chromaHeight, bitDepth)
}

func h264ReadIntraPCMPlaneHigh(gb *bitReader, dst []uint16, offset int, stride int, width int, height int, bitDepth int) error {
	if gb == nil || offset < 0 || stride <= 0 || width <= 0 || height <= 0 {
		return ErrInvalidData
	}
	if len(dst) < offset+(height-1)*stride+width {
		return ErrInvalidData
	}
	for y := 0; y < height; y++ {
		row := offset + y*stride
		for x := 0; x < width; x++ {
			v, err := gb.readBits(uint32(bitDepth))
			if err != nil {
				return err
			}
			dst[row+x] = uint16(v)
		}
	}
	return nil
}

func h264HLDecodeFrameIntraPredictHigh(dst *h264PicturePlanesHigh, dstY int, dstCb int, dstCr int, blockOffset *[48]int, in h264FrameMBReconstructInputHigh) error {
	if dst.ChromaFormatIDC != 0 {
		if err := h264PredChromaByModeHigh(dst.Cb, dstCb, dst.ChromaStride, dst.ChromaFormatIDC, int(in.ChromaPredMode), in.BitDepth); err != nil {
			return err
		}
		if err := h264PredChromaByModeHigh(dst.Cr, dstCr, dst.ChromaStride, dst.ChromaFormatIDC, int(in.ChromaPredMode), in.BitDepth); err != nil {
			return err
		}
	}
	if isIntra4x4(in.MBType) {
		return h264HLDecodeMBPredictLumaIntra4x4High(dst.Y, dstY, dst.LumaStride, blockOffset, in.MBType, in.Intra4x4PredCache, in.TopLeftAvailable, in.TopRightAvailable, in.Residual, in.TransformBypass, h264ProfileIDCFromPPS(in.PPS), in.BitDepth)
	}
	if !isIntra16x16(in.MBType) {
		return ErrUnsupported
	}
	return h264HLDecodeMBPredictLumaIntra16x16High(dst.Y, dstY, dst.LumaStride, int(in.Intra16x16PredMode), in.QScale, in.PPS, in.Residual, in.TransformBypass, in.BitDepth)
}

func h264HLDecodeFrameIntraPredictLumaPlaneHigh(dest []uint16, baseOffset int, stride int, blockOffset *[48]int, in h264FrameMBReconstructInputHigh, plane int) error {
	if isIntra4x4(in.MBType) {
		return h264HLDecodeMBPredictLumaIntra4x4PlaneHigh(dest, baseOffset, stride, blockOffset, in.MBType, in.Intra4x4PredCache, in.TopLeftAvailable, in.TopRightAvailable, in.Residual, plane, in.TransformBypass, h264ProfileIDCFromPPS(in.PPS), in.BitDepth)
	}
	if !isIntra16x16(in.MBType) {
		return ErrUnsupported
	}
	qscale := in.QScale
	if plane > 0 {
		qscale = int(in.ChromaQP[plane-1])
	}
	return h264HLDecodeMBPredictLumaIntra16x16PlaneHigh(dest, baseOffset, stride, int(in.Intra16x16PredMode), qscale, in.PPS, in.Residual, plane, in.TransformBypass, in.BitDepth)
}

func h264HLDecodeMBPredictLumaIntra16x16High(destY []uint16, offset int, stride int, predMode int, qscale int, pps *PPS, residual *cavlcResidualContext, transformBypass bool, bitDepth int) error {
	return h264HLDecodeMBPredictLumaIntra16x16PlaneHigh(destY, offset, stride, predMode, qscale, pps, residual, 0, transformBypass, bitDepth)
}

func h264HLDecodeMBPredictLumaIntra16x16PlaneHigh(destY []uint16, offset int, stride int, predMode int, qscale int, pps *PPS, residual *cavlcResidualContext, plane int, transformBypass bool, bitDepth int) error {
	if pps == nil || residual == nil || qscale < 0 || qscale > h264MaxQPForBitDepth(bitDepth) {
		return ErrInvalidData
	}
	if plane < 0 || plane > 2 {
		return ErrInvalidData
	}
	if err := h264Pred16x16ByModeHigh(destY, offset, stride, predMode, bitDepth); err != nil {
		return err
	}
	if residual.NonZeroCountCache[h264Scan8[lumaDCBlockIndex+plane]] != 0 {
		block := residual.MB[plane*16*16:]
		if transformBypass {
			h264LumaDCDirect(block[:16*16], &residual.MBLumaDC[plane])
		} else if err := h264LumaDCDequantIDCTHigh(block[:16*16], &residual.MBLumaDC[plane], int(pps.Dequant4Buffer[plane][qscale][0])); err != nil {
			return err
		}
	}
	return nil
}

func h264HLDecodeMBIDCTLumaHigh(destY []uint16, stride int, blockOffset *[48]int, mbType uint32, cbp int, residual *cavlcResidualContext, transformBypass bool, intra16x16PredMode int, profileIDC int32, bitDepth int) error {
	return h264HLDecodeMBIDCTLumaPlaneHigh(destY, stride, blockOffset, mbType, cbp, residual, 0, transformBypass, intra16x16PredMode, profileIDC, bitDepth)
}

func h264HLDecodeMBIDCTLumaPlaneHigh(destY []uint16, stride int, blockOffset *[48]int, mbType uint32, cbp int, residual *cavlcResidualContext, plane int, transformBypass bool, intra16x16PredMode int, profileIDC int32, bitDepth int) error {
	if residual == nil {
		return ErrInvalidData
	}
	if plane < 0 || plane > 2 {
		return ErrInvalidData
	}
	if isIntra4x4(mbType) {
		return nil
	}
	if isIntra16x16(mbType) {
		if transformBypass {
			if profileIDC == 244 && intra16x16PredMode == intraPred8x8Vertical {
				return h264Pred16x16VerticalAddAtHigh(destY, blockOffset, plane*16, residual.MB[plane*16*16:], stride, bitDepth)
			}
			if profileIDC == 244 && intra16x16PredMode == intraPred8x8Horizontal {
				return h264Pred16x16HorizontalAddAtHigh(destY, blockOffset, plane*16, residual.MB[plane*16*16:], stride, bitDepth)
			}
			return h264AddPixels16BypassPlaneHigh(destY, blockOffset, residual.MB[:], stride, &residual.NonZeroCountCache, plane, true, bitDepth)
		}
		return h264IDCTAdd16IntraPlaneHigh(destY, blockOffset, residual.MB[:], stride, &residual.NonZeroCountCache, plane, bitDepth)
	}
	if cbp&15 == 0 {
		return nil
	}
	if transformBypass {
		if is8x8DCT(mbType) {
			return h264AddPixels8Bypass4PlaneHigh(destY, blockOffset, residual.MB[:], stride, &residual.NonZeroCountCache, plane, bitDepth)
		}
		return h264AddPixels16BypassPlaneHigh(destY, blockOffset, residual.MB[:], stride, &residual.NonZeroCountCache, plane, false, bitDepth)
	}
	if is8x8DCT(mbType) {
		return h264IDCT8Add4PlaneHigh(destY, blockOffset, residual.MB[:], stride, &residual.NonZeroCountCache, plane, bitDepth)
	}
	return h264IDCTAdd16PlaneHigh(destY, blockOffset, residual.MB[:], stride, &residual.NonZeroCountCache, plane, bitDepth)
}

func h264HLDecodeMBPredictLumaIntra4x4High(destY []uint16, baseOffset int, stride int, blockOffset *[48]int, mbType uint32, predCache *[h264IntraPredModeCacheSize]int8, topLeftAvailable uint16, topRightAvailable uint16, residual *cavlcResidualContext, transformBypass bool, profileIDC int32, bitDepth int) error {
	return h264HLDecodeMBPredictLumaIntra4x4PlaneHigh(destY, baseOffset, stride, blockOffset, mbType, predCache, topLeftAvailable, topRightAvailable, residual, 0, transformBypass, profileIDC, bitDepth)
}

func h264HLDecodeMBPredictLumaIntra4x4PlaneHigh(destY []uint16, baseOffset int, stride int, blockOffset *[48]int, mbType uint32, predCache *[h264IntraPredModeCacheSize]int8, topLeftAvailable uint16, topRightAvailable uint16, residual *cavlcResidualContext, plane int, transformBypass bool, profileIDC int32, bitDepth int) error {
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
				if dir == int(intraPredVertical) {
					if err := h264Pred8x8LVerticalFilterAddHigh(destY, offset, block, stride, hasTopLeft, hasTopRight, bitDepth); err != nil {
						return err
					}
				} else if err := h264Pred8x8LHorizontalFilterAddHigh(destY, offset, block, stride, hasTopLeft, hasTopRight, bitDepth); err != nil {
					return err
				}
				continue
			}
			if err := h264Pred8x8LByModeHigh(destY, offset, stride, dir, hasTopLeft, hasTopRight, bitDepth); err != nil {
				return err
			}
			nnz := residual.NonZeroCountCache[h264Scan8[index]]
			if nnz == 0 {
				continue
			}
			if transformBypass {
				if err := h264AddPixels8ClearHigh(destY[offset:], block, stride); err != nil {
					return err
				}
			} else if nnz == 1 && block[0] != 0 {
				if err := h264IDCT8DCAddHigh(destY[offset:], block, stride, bitDepth); err != nil {
					return err
				}
			} else if err := h264IDCT8AddHigh(destY[offset:], block, stride, bitDepth); err != nil {
				return err
			}
		}
		return nil
	}

	var unavailableTopRight [4]uint16
	for i := 0; i < 16; i++ {
		index := i + planeBlock
		offset := baseOffset + blockOffset[index]
		dir := int(predCache[h264Scan8[i]])
		var topRight []uint16
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
				if err := h264Pred4x4VerticalAddHigh(destY, offset, block, stride, bitDepth); err != nil {
					return err
				}
			} else if err := h264Pred4x4HorizontalAddHigh(destY, offset, block, stride, bitDepth); err != nil {
				return err
			}
			continue
		}
		if err := h264Pred4x4ByModeHigh(destY, offset, stride, dir, topRight, bitDepth); err != nil {
			return err
		}
		if nnz == 0 {
			continue
		}
		if transformBypass {
			if err := h264AddPixels4ClearHigh(destY[offset:], block, stride); err != nil {
				return err
			}
		} else if nnz == 1 && block[0] != 0 {
			if err := h264IDCTDCAddHigh(destY[offset:], block, stride, bitDepth); err != nil {
				return err
			}
		} else if err := h264IDCTAddHigh(destY[offset:], block, stride, bitDepth); err != nil {
			return err
		}
	}
	return nil
}

func h264IDCTAdd16PlaneHigh(dst []uint16, blockOffset *[48]int, block []int32, stride int, nnzc *[h264NonZeroCountCacheSize]uint8, plane int, bitDepth int) error {
	if err := checkH264DSPHighBitDepth(bitDepth); err != nil {
		return err
	}
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
		dstBlock, err := transformBlockDestinationHigh(dst, blockOffset[index], stride, 4)
		if err != nil {
			return err
		}
		coef := block[index*16 : index*16+16]
		if nnz == 1 && coef[0] != 0 {
			if err := h264IDCTDCAddHigh(dstBlock, coef, stride, bitDepth); err != nil {
				return err
			}
		} else if err := h264IDCTAddHigh(dstBlock, coef, stride, bitDepth); err != nil {
			return err
		}
	}
	return nil
}

func h264IDCTAdd16IntraPlaneHigh(dst []uint16, blockOffset *[48]int, block []int32, stride int, nnzc *[h264NonZeroCountCacheSize]uint8, plane int, bitDepth int) error {
	if err := checkH264DSPHighBitDepth(bitDepth); err != nil {
		return err
	}
	if blockOffset == nil || nnzc == nil || plane < 0 || plane > 2 || len(block) < (plane+1)*16*16 {
		return ErrInvalidData
	}
	planeBlock := 16 * plane
	for i := 0; i < 16; i++ {
		index := i + planeBlock
		coef := block[index*16 : index*16+16]
		if nnzc[h264Scan8[index]] != 0 {
			dstBlock, err := transformBlockDestinationHigh(dst, blockOffset[index], stride, 4)
			if err != nil {
				return err
			}
			if err := h264IDCTAddHigh(dstBlock, coef, stride, bitDepth); err != nil {
				return err
			}
		} else if coef[0] != 0 {
			dstBlock, err := transformBlockDestinationHigh(dst, blockOffset[index], stride, 4)
			if err != nil {
				return err
			}
			if err := h264IDCTDCAddHigh(dstBlock, coef, stride, bitDepth); err != nil {
				return err
			}
		}
	}
	return nil
}

func h264IDCT8Add4PlaneHigh(dst []uint16, blockOffset *[48]int, block []int32, stride int, nnzc *[h264NonZeroCountCacheSize]uint8, plane int, bitDepth int) error {
	if err := checkH264DSPHighBitDepth(bitDepth); err != nil {
		return err
	}
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
		dstBlock, err := transformBlockDestinationHigh(dst, blockOffset[index], stride, 8)
		if err != nil {
			return err
		}
		coef := block[index*16 : index*16+64]
		if nnz == 1 && coef[0] != 0 {
			if err := h264IDCT8DCAddHigh(dstBlock, coef, stride, bitDepth); err != nil {
				return err
			}
		} else if err := h264IDCT8AddHigh(dstBlock, coef, stride, bitDepth); err != nil {
			return err
		}
	}
	return nil
}

func h264AddPixels16BypassPlaneHigh(dst []uint16, blockOffset *[48]int, block []int32, stride int, nnzc *[h264NonZeroCountCacheSize]uint8, plane int, includeDCOnly bool, bitDepth int) error {
	if err := checkH264DSPHighBitDepth(bitDepth); err != nil {
		return err
	}
	if blockOffset == nil || nnzc == nil || plane < 0 || plane > 2 || len(block) < (plane+1)*16*16 {
		return ErrInvalidData
	}
	planeBlock := 16 * plane
	for i := 0; i < 16; i++ {
		index := i + planeBlock
		coef := block[index*16 : index*16+16]
		if nnzc[h264Scan8[index]] == 0 && (!includeDCOnly || coef[0] == 0) {
			continue
		}
		dstBlock, err := transformBlockDestinationHigh(dst, blockOffset[index], stride, 4)
		if err != nil {
			return err
		}
		if err := h264AddPixels4ClearHigh(dstBlock, coef, stride); err != nil {
			return err
		}
	}
	return nil
}

func h264AddPixels8Bypass4PlaneHigh(dst []uint16, blockOffset *[48]int, block []int32, stride int, nnzc *[h264NonZeroCountCacheSize]uint8, plane int, bitDepth int) error {
	if err := checkH264DSPHighBitDepth(bitDepth); err != nil {
		return err
	}
	if blockOffset == nil || nnzc == nil || plane < 0 || plane > 2 || len(block) < (plane+1)*16*16 {
		return ErrInvalidData
	}
	planeBlock := 16 * plane
	for i := 0; i < 16; i += 4 {
		index := i + planeBlock
		if nnzc[h264Scan8[index]] == 0 {
			continue
		}
		dstBlock, err := transformBlockDestinationHigh(dst, blockOffset[index], stride, 8)
		if err != nil {
			return err
		}
		if err := h264AddPixels8ClearHigh(dstBlock, block[index*16:index*16+64], stride); err != nil {
			return err
		}
	}
	return nil
}

func h264HLDecodeMBIDCTChromaHigh(destCb []uint16, destCr []uint16, stride int, blockOffset *[48]int, chromaFormatIDC int, mbType uint32, cbp int, chromaQP [2]uint8, pps *PPS, residual *cavlcResidualContext, transformBypass bool, chromaPredMode int, profileIDC int32, bitDepth int) error {
	if pps == nil || residual == nil {
		return ErrInvalidData
	}
	if cbp&0x30 == 0 {
		return nil
	}
	maxQP := h264MaxQPForBitDepth(bitDepth)
	qp0 := int(chromaQP[0])
	qp1 := int(chromaQP[1])
	if qp0 > maxQP || qp1 > maxQP {
		return ErrInvalidData
	}
	if chromaFormatIDC == 2 {
		qp0 += 3
		qp1 += 3
		if qp0 > maxQP || qp1 > maxQP {
			return ErrInvalidData
		}
	}
	cqm0, cqm1 := 4, 5
	if isIntra(mbType) {
		cqm0, cqm1 = 1, 2
	}
	if transformBypass {
		return h264HLDecodeMBAddChromaBypassHigh(destCb, destCr, stride, blockOffset, chromaFormatIDC, mbType, chromaPredMode, profileIDC, residual, bitDepth)
	}
	if residual.NonZeroCountCache[h264Scan8[chromaDCBlockIndex+0]] != 0 {
		if err := h264ChromaDCDequantIDCTByFormatHigh(residual.MB[16*16:], int(pps.Dequant4Buffer[cqm0][qp0][0]), chromaFormatIDC); err != nil {
			return err
		}
	}
	if residual.NonZeroCountCache[h264Scan8[chromaDCBlockIndex+1]] != 0 {
		if err := h264ChromaDCDequantIDCTByFormatHigh(residual.MB[16*16*2:], int(pps.Dequant4Buffer[cqm1][qp1][0]), chromaFormatIDC); err != nil {
			return err
		}
	}
	dest := [2][]uint16{destCb, destCr}
	if chromaFormatIDC == 2 {
		return h264IDCTAdd8_422High(&dest, blockOffset, residual.MB[:], stride, &residual.NonZeroCountCache, bitDepth)
	}
	if chromaFormatIDC == 1 {
		return h264IDCTAdd8High(&dest, blockOffset, residual.MB[:], stride, &residual.NonZeroCountCache, bitDepth)
	}
	return ErrInvalidData
}

func h264HLDecodeMBAddChromaBypassHigh(destCb []uint16, destCr []uint16, stride int, blockOffset *[48]int, chromaFormatIDC int, mbType uint32, chromaPredMode int, profileIDC int32, residual *cavlcResidualContext, bitDepth int) error {
	if blockOffset == nil || residual == nil {
		return ErrInvalidData
	}
	dest := [2][]uint16{destCb, destCr}
	if isIntra(mbType) && profileIDC == 244 && (chromaPredMode == intraPred8x8Vertical || chromaPredMode == intraPred8x8Horizontal) {
		for plane := 0; plane < 2; plane++ {
			baseBlock := 16 + plane*16
			block := residual.MB[baseBlock*16:]
			if chromaFormatIDC == 1 {
				if chromaPredMode == intraPred8x8Vertical {
					if err := h264Pred8x8VerticalAddAtHigh(dest[plane], blockOffset, baseBlock, block, stride, bitDepth); err != nil {
						return err
					}
				} else if err := h264Pred8x8HorizontalAddAtHigh(dest[plane], blockOffset, baseBlock, block, stride, bitDepth); err != nil {
					return err
				}
			} else if chromaFormatIDC == 2 {
				if chromaPredMode == intraPred8x8Vertical {
					if err := h264Pred8x16VerticalAddAtHigh(dest[plane], blockOffset, baseBlock, block, stride, bitDepth); err != nil {
						return err
					}
				} else if err := h264Pred8x16HorizontalAddAtHigh(dest[plane], blockOffset, baseBlock, block, stride, bitDepth); err != nil {
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
			if residual.NonZeroCountCache[h264Scan8[i]] == 0 && residual.MB[i*16] == 0 {
				continue
			}
			dstBlock, err := transformBlockDestinationHigh(dest[plane], blockOffset[i], stride, 4)
			if err != nil {
				return err
			}
			if err := h264AddPixels4ClearHigh(dstBlock, residual.MB[i*16:i*16+16], stride); err != nil {
				return err
			}
		}
		if chromaFormatIDC == 2 {
			for i := j*16 + 4; i < j*16+8; i++ {
				if residual.NonZeroCountCache[h264Scan8[i+4]] == 0 && residual.MB[i*16] == 0 {
					continue
				}
				dstBlock, err := transformBlockDestinationHigh(dest[plane], blockOffset[i+4], stride, 4)
				if err != nil {
					return err
				}
				if err := h264AddPixels4ClearHigh(dstBlock, residual.MB[i*16:i*16+16], stride); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func h264Pred8x8VerticalAddAtHigh(pix []uint16, blockOffset *[48]int, offsetBase int, block []int32, stride int, bitDepth int) error {
	if err := checkH264DSPHighBitDepth(bitDepth); err != nil {
		return err
	}
	if blockOffset == nil || offsetBase < 0 || offsetBase+4 > len(blockOffset) || len(block) < 4*16 {
		return ErrInvalidData
	}
	for i := 0; i < 4; i++ {
		if err := h264Pred4x4VerticalAddHigh(pix, blockOffset[offsetBase+i], block[i*16:i*16+16], stride, bitDepth); err != nil {
			return err
		}
	}
	return nil
}

func h264Pred8x8HorizontalAddAtHigh(pix []uint16, blockOffset *[48]int, offsetBase int, block []int32, stride int, bitDepth int) error {
	if err := checkH264DSPHighBitDepth(bitDepth); err != nil {
		return err
	}
	if blockOffset == nil || offsetBase < 0 || offsetBase+4 > len(blockOffset) || len(block) < 4*16 {
		return ErrInvalidData
	}
	for i := 0; i < 4; i++ {
		if err := h264Pred4x4HorizontalAddHigh(pix, blockOffset[offsetBase+i], block[i*16:i*16+16], stride, bitDepth); err != nil {
			return err
		}
	}
	return nil
}

func h264Pred16x16VerticalAddAtHigh(pix []uint16, blockOffset *[48]int, offsetBase int, block []int32, stride int, bitDepth int) error {
	if err := checkH264DSPHighBitDepth(bitDepth); err != nil {
		return err
	}
	if blockOffset == nil || offsetBase < 0 || offsetBase+16 > len(blockOffset) || len(block) < 16*16 {
		return ErrInvalidData
	}
	for i := 0; i < 16; i++ {
		if err := h264Pred4x4VerticalAddHigh(pix, blockOffset[offsetBase+i], block[i*16:i*16+16], stride, bitDepth); err != nil {
			return err
		}
	}
	return nil
}

func h264Pred16x16HorizontalAddAtHigh(pix []uint16, blockOffset *[48]int, offsetBase int, block []int32, stride int, bitDepth int) error {
	if err := checkH264DSPHighBitDepth(bitDepth); err != nil {
		return err
	}
	if blockOffset == nil || offsetBase < 0 || offsetBase+16 > len(blockOffset) || len(block) < 16*16 {
		return ErrInvalidData
	}
	for i := 0; i < 16; i++ {
		if err := h264Pred4x4HorizontalAddHigh(pix, blockOffset[offsetBase+i], block[i*16:i*16+16], stride, bitDepth); err != nil {
			return err
		}
	}
	return nil
}

func h264Pred8x16VerticalAddAtHigh(pix []uint16, blockOffset *[48]int, offsetBase int, block []int32, stride int, bitDepth int) error {
	if err := checkH264DSPHighBitDepth(bitDepth); err != nil {
		return err
	}
	if blockOffset == nil || offsetBase < 0 || offsetBase+12 > len(blockOffset) || len(block) < 8*16 {
		return ErrInvalidData
	}
	for i := 0; i < 4; i++ {
		if err := h264Pred4x4VerticalAddHigh(pix, blockOffset[offsetBase+i], block[i*16:i*16+16], stride, bitDepth); err != nil {
			return err
		}
	}
	for i := 4; i < 8; i++ {
		if err := h264Pred4x4VerticalAddHigh(pix, blockOffset[offsetBase+i+4], block[i*16:i*16+16], stride, bitDepth); err != nil {
			return err
		}
	}
	return nil
}

func h264Pred8x16HorizontalAddAtHigh(pix []uint16, blockOffset *[48]int, offsetBase int, block []int32, stride int, bitDepth int) error {
	if err := checkH264DSPHighBitDepth(bitDepth); err != nil {
		return err
	}
	if blockOffset == nil || offsetBase < 0 || offsetBase+12 > len(blockOffset) || len(block) < 8*16 {
		return ErrInvalidData
	}
	for i := 0; i < 4; i++ {
		if err := h264Pred4x4HorizontalAddHigh(pix, blockOffset[offsetBase+i], block[i*16:i*16+16], stride, bitDepth); err != nil {
			return err
		}
	}
	for i := 4; i < 8; i++ {
		if err := h264Pred4x4HorizontalAddHigh(pix, blockOffset[offsetBase+i+4], block[i*16:i*16+16], stride, bitDepth); err != nil {
			return err
		}
	}
	return nil
}

func h264ChromaDCDequantIDCTByFormatHigh(block []int32, qmul int, chromaFormatIDC int) error {
	if chromaFormatIDC == 2 {
		return h264Chroma422DCDequantIDCTHigh(block, qmul)
	}
	if chromaFormatIDC == 1 {
		return h264ChromaDCDequantIDCTHigh(block, qmul)
	}
	return ErrInvalidData
}

func h264Pred16x16ByModeHigh(pix []uint16, offset int, stride int, mode int, bitDepth int) error {
	switch mode {
	case intraPred8x8DC:
		return h264Pred16x16DCHigh(pix, offset, stride, bitDepth)
	case intraPred8x8Horizontal:
		return h264Pred16x16HorizontalHigh(pix, offset, stride, bitDepth)
	case intraPred8x8Vertical:
		return h264Pred16x16VerticalHigh(pix, offset, stride, bitDepth)
	case intraPred8x8Plane:
		return h264Pred16x16PlaneHigh(pix, offset, stride, bitDepth)
	case intraPred8x8LeftDC:
		return h264Pred16x16LeftDCHigh(pix, offset, stride, bitDepth)
	case intraPred8x8TopDC:
		return h264Pred16x16TopDCHigh(pix, offset, stride, bitDepth)
	case intraPredDC1288x8:
		return h264Pred16x16DC128High(pix, offset, stride, bitDepth)
	default:
		return ErrInvalidData
	}
}

func h264Pred4x4ByModeHigh(pix []uint16, offset int, stride int, mode int, topRight []uint16, bitDepth int) error {
	switch int8(mode) {
	case intraPredVertical:
		return h264Pred4x4VerticalHigh(pix, offset, stride, bitDepth)
	case intraPredHorizontal:
		return h264Pred4x4HorizontalHigh(pix, offset, stride, bitDepth)
	case intraPredDC:
		return h264Pred4x4DCHigh(pix, offset, stride, bitDepth)
	case intraPredDiagDownLeft:
		return h264Pred4x4DownLeftHigh(pix, offset, stride, topRight, bitDepth)
	case intraPredDiagDownRight:
		return h264Pred4x4DownRightHigh(pix, offset, stride, bitDepth)
	case intraPredVertRight:
		return h264Pred4x4VerticalRightHigh(pix, offset, stride, bitDepth)
	case intraPredHorDown:
		return h264Pred4x4HorizontalDownHigh(pix, offset, stride, bitDepth)
	case intraPredVertLeft:
		return h264Pred4x4VerticalLeftHigh(pix, offset, stride, topRight, bitDepth)
	case intraPredHorUp:
		return h264Pred4x4HorizontalUpHigh(pix, offset, stride, bitDepth)
	case intraPredLeftDC:
		return h264Pred4x4LeftDCHigh(pix, offset, stride, bitDepth)
	case intraPredTopDC:
		return h264Pred4x4TopDCHigh(pix, offset, stride, bitDepth)
	case intraPredDC128:
		return h264Pred4x4DC128High(pix, offset, stride, bitDepth)
	default:
		return ErrInvalidData
	}
}

func h264Pred8x8LByModeHigh(pix []uint16, offset int, stride int, mode int, hasTopLeft bool, hasTopRight bool, bitDepth int) error {
	switch int8(mode) {
	case intraPredVertical:
		return h264Pred8x8LVerticalHigh(pix, offset, stride, hasTopLeft, hasTopRight, bitDepth)
	case intraPredHorizontal:
		return h264Pred8x8LHorizontalHigh(pix, offset, stride, hasTopLeft, hasTopRight, bitDepth)
	case intraPredDC:
		return h264Pred8x8LDCHigh(pix, offset, stride, hasTopLeft, hasTopRight, bitDepth)
	case intraPredDiagDownLeft:
		return h264Pred8x8LDownLeftHigh(pix, offset, stride, hasTopLeft, hasTopRight, bitDepth)
	case intraPredDiagDownRight:
		return h264Pred8x8LDownRightHigh(pix, offset, stride, hasTopLeft, hasTopRight, bitDepth)
	case intraPredVertRight:
		return h264Pred8x8LVerticalRightHigh(pix, offset, stride, hasTopLeft, hasTopRight, bitDepth)
	case intraPredHorDown:
		return h264Pred8x8LHorizontalDownHigh(pix, offset, stride, hasTopLeft, hasTopRight, bitDepth)
	case intraPredVertLeft:
		return h264Pred8x8LVerticalLeftHigh(pix, offset, stride, hasTopLeft, hasTopRight, bitDepth)
	case intraPredHorUp:
		return h264Pred8x8LHorizontalUpHigh(pix, offset, stride, hasTopLeft, hasTopRight, bitDepth)
	case intraPredLeftDC:
		return h264Pred8x8LLeftDCHigh(pix, offset, stride, hasTopLeft, hasTopRight, bitDepth)
	case intraPredTopDC:
		return h264Pred8x8LTopDCHigh(pix, offset, stride, hasTopLeft, hasTopRight, bitDepth)
	case intraPredDC128:
		return h264Pred8x8LDC128High(pix, offset, stride, hasTopLeft, hasTopRight, bitDepth)
	default:
		return ErrInvalidData
	}
}

func h264PredChromaByModeHigh(pix []uint16, offset int, stride int, chromaFormatIDC int, mode int, bitDepth int) error {
	if chromaFormatIDC == 2 {
		return h264Pred8x16ByModeHigh(pix, offset, stride, mode, bitDepth)
	}
	if chromaFormatIDC == 1 {
		return h264Pred8x8ByModeHigh(pix, offset, stride, mode, bitDepth)
	}
	return ErrInvalidData
}

func h264Pred8x8ByModeHigh(pix []uint16, offset int, stride int, mode int, bitDepth int) error {
	switch mode {
	case intraPred8x8DC:
		return h264Pred8x8DCHigh(pix, offset, stride, bitDepth)
	case intraPred8x8Horizontal:
		return h264Pred8x8HorizontalHigh(pix, offset, stride, bitDepth)
	case intraPred8x8Vertical:
		return h264Pred8x8VerticalHigh(pix, offset, stride, bitDepth)
	case intraPred8x8Plane:
		return h264Pred8x8PlaneHigh(pix, offset, stride, bitDepth)
	case intraPred8x8LeftDC:
		return h264Pred8x8LeftDCHigh(pix, offset, stride, bitDepth)
	case intraPred8x8TopDC:
		return h264Pred8x8TopDCHigh(pix, offset, stride, bitDepth)
	case intraPredDC1288x8:
		return h264Pred8x8DC128High(pix, offset, stride, bitDepth)
	case intraPred8x8AlzheimerL0TDC:
		return h264Pred8x8MadCowDCL0THigh(pix, offset, stride, bitDepth)
	case intraPred8x8Alzheimer0LTDC:
		return h264Pred8x8MadCowDC0LTHigh(pix, offset, stride, bitDepth)
	case intraPred8x8AlzheimerL00DC:
		return h264Pred8x8MadCowDCL00High(pix, offset, stride, bitDepth)
	case intraPred8x8Alzheimer0L0DC:
		return h264Pred8x8MadCowDC0L0High(pix, offset, stride, bitDepth)
	default:
		return ErrUnsupported
	}
}

func h264Pred8x16ByModeHigh(pix []uint16, offset int, stride int, mode int, bitDepth int) error {
	switch mode {
	case intraPred8x8DC:
		return h264Pred8x16DCHigh(pix, offset, stride, bitDepth)
	case intraPred8x8Horizontal:
		return h264Pred8x16HorizontalHigh(pix, offset, stride, bitDepth)
	case intraPred8x8Vertical:
		return h264Pred8x16VerticalHigh(pix, offset, stride, bitDepth)
	case intraPred8x8Plane:
		return h264Pred8x16PlaneHigh(pix, offset, stride, bitDepth)
	case intraPred8x8LeftDC:
		return h264Pred8x16LeftDCHigh(pix, offset, stride, bitDepth)
	case intraPred8x8TopDC:
		return h264Pred8x16TopDCHigh(pix, offset, stride, bitDepth)
	case intraPredDC1288x8:
		return h264Pred8x16DC128High(pix, offset, stride, bitDepth)
	case intraPred8x8AlzheimerL0TDC:
		return h264Pred8x16MadCowDCL0THigh(pix, offset, stride, bitDepth)
	case intraPred8x8Alzheimer0LTDC:
		return h264Pred8x16MadCowDC0LTHigh(pix, offset, stride, bitDepth)
	case intraPred8x8AlzheimerL00DC:
		return h264Pred8x16MadCowDCL00High(pix, offset, stride, bitDepth)
	case intraPred8x8Alzheimer0L0DC:
		return h264Pred8x16MadCowDC0L0High(pix, offset, stride, bitDepth)
	default:
		return ErrUnsupported
	}
}

func h264MBDestPartOffsetsHigh(dst *h264PicturePlanesHigh, mbX int, mbY int, xOffset int, yOffset int) (int, int, int, error) {
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

func (p *h264PicturePlanesHigh) validate() error {
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
