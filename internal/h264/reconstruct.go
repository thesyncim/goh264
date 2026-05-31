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
	MotionScratch       *h264MotionCompScratch
	TransformBypass     bool
	DeblockingFilter    bool
	ConstrainedIntra444 bool
	IntraPCM            []byte
}

func h264HLDecodeFrameMacroblock(dst *h264PicturePlanes, in h264FrameMBReconstructInput) error {
	if dst == nil || in.MBX < 0 || in.MBY < 0 || in.QScale < 0 || in.QScale > qpMaxNum {
		return ErrInvalidData
	}
	if in.TransformBypass || in.DeblockingFilter || in.ConstrainedIntra444 {
		return ErrUnsupported
	}
	if err := dst.validate(); err != nil {
		return err
	}
	if in.MBX >= dst.MBWidth || in.MBY >= dst.MBHeight {
		return ErrInvalidData
	}
	if dst.ChromaFormatIDC == 3 {
		return ErrUnsupported
	}

	chromaStride := dst.ChromaStride
	if chromaStride == 0 {
		chromaStride = 1
	}
	blockOffset, err := h264FrameBlockOffsets(dst.LumaStride, chromaStride, 0)
	if err != nil {
		return err
	}
	dstY, dstCb, dstCr, err := h264MBDestPartOffsets(dst, in.MBX, in.MBY, 0, 0)
	if err != nil {
		return err
	}

	if in.MBType&MBTypeIntraPCM != 0 {
		return h264HLDecodeFrameIntraPCM(dst, dstY, dstCb, dstCr, in.IntraPCM)
	}
	if in.PPS == nil || in.Residual == nil {
		return ErrInvalidData
	}
	if isIntra(in.MBType) {
		if err := h264HLDecodeFrameIntraPredict(dst, dstY, dstCb, dstCr, &blockOffset, in); err != nil {
			return err
		}
	} else {
		if in.Motion == nil {
			return ErrInvalidData
		}
		if in.PredWeight != nil {
			if err := h264HLMotionFrameWeighted(dst, in.Refs, in.Motion, in.MBType, in.SubMBType, in.MBX, in.MBY, in.ListCount, in.PredWeight, in.MotionScratch); err != nil {
				return err
			}
		} else if err := h264HLMotionFrameWithScratch(dst, in.Refs, in.Motion, in.MBType, in.SubMBType, in.MBX, in.MBY, in.ListCount, in.MotionScratch); err != nil {
			return err
		}
	}

	if err := h264HLDecodeMBIDCTLuma(dst.Y[dstY:], dst.LumaStride, &blockOffset, in.MBType, in.CBP, in.Residual); err != nil {
		return err
	}
	if dst.ChromaFormatIDC != 0 && in.CBP&0x30 != 0 {
		return h264HLDecodeMBIDCTChroma(dst.Cb[dstCb:], dst.Cr[dstCr:], dst.ChromaStride, &blockOffset, dst.ChromaFormatIDC, in.MBType, in.CBP, in.ChromaQP, in.PPS, in.Residual)
	}
	return nil
}

func h264HLDecodeFrameIntraPCM(dst *h264PicturePlanes, dstY int, dstCb int, dstCr int, pcm []byte) error {
	if dst == nil || dst.ChromaFormatIDC < 0 || dst.ChromaFormatIDC >= len(h264IntraPCMSampleCount) || dst.ChromaFormatIDC == 3 {
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
	if offset < 0 || stride <= 0 || width <= 0 || height <= 0 || srcStride < width || len(src) < (height-1)*srcStride+width {
		return ErrInvalidData
	}
	if len(dst) < offset+(height-1)*stride+width {
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
		return h264HLDecodeMBPredictLumaIntra4x4(dst.Y, dstY, dst.LumaStride, blockOffset, in.MBType, in.Intra4x4PredCache, in.TopLeftAvailable, in.TopRightAvailable, in.Residual)
	}
	if !isIntra16x16(in.MBType) {
		return ErrUnsupported
	}
	return h264HLDecodeMBPredictLumaIntra16x16(dst.Y, dstY, dst.LumaStride, int(in.Intra16x16PredMode), in.QScale, in.PPS, in.Residual)
}

func h264HLDecodeMBPredictLumaIntra16x16(destY []uint8, offset int, stride int, predMode int, qscale int, pps *PPS, residual *cavlcResidualContext) error {
	if pps == nil || residual == nil || qscale < 0 || qscale > qpMaxNum {
		return ErrInvalidData
	}
	if err := h264Pred16x16ByMode(destY, offset, stride, predMode); err != nil {
		return err
	}
	if residual.NonZeroCountCache[h264Scan8[lumaDCBlockIndex]] != 0 {
		if err := h264LumaDCDequantIDCT(residual.MB[:16*16], &residual.MBLumaDC[0], int(pps.Dequant4Buffer[0][qscale][0])); err != nil {
			return err
		}
	}
	return nil
}

func h264HLDecodeMBIDCTLuma(destY []uint8, stride int, blockOffset *[48]int, mbType uint32, cbp int, residual *cavlcResidualContext) error {
	if residual == nil {
		return ErrInvalidData
	}
	if isIntra4x4(mbType) {
		return nil
	}
	if isIntra16x16(mbType) {
		return h264IDCTAdd16Intra(destY, blockOffset, residual.MB[:], stride, &residual.NonZeroCountCache)
	}
	if cbp&15 == 0 {
		return nil
	}
	if is8x8DCT(mbType) {
		return h264IDCT8Add4(destY, blockOffset, residual.MB[:], stride, &residual.NonZeroCountCache)
	}
	return h264IDCTAdd16(destY, blockOffset, residual.MB[:], stride, &residual.NonZeroCountCache)
}

func h264HLDecodeMBPredictLumaIntra4x4(destY []uint8, baseOffset int, stride int, blockOffset *[48]int, mbType uint32, predCache *[h264IntraPredModeCacheSize]int8, topLeftAvailable uint16, topRightAvailable uint16, residual *cavlcResidualContext) error {
	if blockOffset == nil || predCache == nil || residual == nil {
		return ErrInvalidData
	}
	if is8x8DCT(mbType) {
		for i := 0; i < 16; i += 4 {
			offset := baseOffset + blockOffset[i]
			dir := int(predCache[h264Scan8[i]])
			hasTopLeft := ((uint32(topLeftAvailable) << uint(i)) & 0x8000) != 0
			hasTopRight := ((uint32(topRightAvailable) << uint(i)) & 0x4000) != 0
			if err := h264Pred8x8LByMode(destY, offset, stride, dir, hasTopLeft, hasTopRight); err != nil {
				return err
			}
			nnz := residual.NonZeroCountCache[h264Scan8[i]]
			if nnz == 0 {
				continue
			}
			block := residual.MB[i*16 : i*16+64]
			if nnz == 1 && dctcoef8Value(block[0]) != 0 {
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
		offset := baseOffset + blockOffset[i]
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
		if err := h264Pred4x4ByMode(destY, offset, stride, dir, topRight); err != nil {
			return err
		}
		nnz := residual.NonZeroCountCache[h264Scan8[i]]
		if nnz == 0 {
			continue
		}
		block := residual.MB[i*16 : i*16+16]
		if nnz == 1 && dctcoef8Value(block[0]) != 0 {
			if err := h264IDCTDCAdd(destY[offset:], block, stride); err != nil {
				return err
			}
		} else if err := h264IDCTAdd(destY[offset:], block, stride); err != nil {
			return err
		}
	}
	return nil
}

func h264HLDecodeMBIDCTChroma(destCb []uint8, destCr []uint8, stride int, blockOffset *[48]int, chromaFormatIDC int, mbType uint32, cbp int, chromaQP [2]uint8, pps *PPS, residual *cavlcResidualContext) error {
	if pps == nil || residual == nil {
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
	dest := [2][]uint8{destCb, destCr}
	if chromaFormatIDC == 2 {
		return h264IDCTAdd8_422(&dest, blockOffset, residual.MB[:], stride, &residual.NonZeroCountCache)
	}
	if chromaFormatIDC == 1 {
		return h264IDCTAdd8(&dest, blockOffset, residual.MB[:], stride, &residual.NonZeroCountCache)
	}
	return ErrInvalidData
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
	default:
		return ErrUnsupported
	}
}
