// SPDX-License-Identifier: LGPL-2.1-or-later
//
// Source-shaped frame-MB motion-compensation call-site helpers from FFmpeg
// n8.0.1 libavcodec/h264_mb.c mc_dir_part/mc_part_std and
// libavcodec/h264_mc_template.c hl_motion.

package h264

type h264PicturePlanes struct {
	Y, Cb, Cr       []uint8
	LumaStride      int
	ChromaStride    int
	MBWidth         int
	MBHeight        int
	ChromaFormatIDC int
}

type h264ChromaMCOp func(dst []uint8, src []uint8, stride int, height int, x int, y int) error

func h264HLMotionFrame(dst *h264PicturePlanes, refs [2][]*h264PicturePlanes, cache *macroblockMotionCache, mbType uint32, subMBType [4]uint32, mbX int, mbY int, listCount int) error {
	if dst == nil || cache == nil || mbX < 0 || mbY < 0 || listCount < 0 || listCount > 2 {
		return ErrInvalidData
	}
	if err := dst.validate(); err != nil {
		return err
	}
	if mbX >= dst.MBWidth || mbY >= dst.MBHeight || !isInter(mbType) {
		return ErrInvalidData
	}
	if isDirect(mbType) {
		return ErrUnsupported
	}

	if is16x16(mbType) {
		return h264MCPartFrameStd(dst, refs, cache, mbX, mbY, mbType, 0, 0, true, 16, 0, 0, 0, 16, 8, listCount)
	}
	if is16x8(mbType) {
		if err := h264MCPartFrameStd(dst, refs, cache, mbX, mbY, mbType, 0, 0, false, 8, 8, 0, 0, 8, 8, listCount); err != nil {
			return err
		}
		return h264MCPartFrameStd(dst, refs, cache, mbX, mbY, mbType, 1, 8, false, 8, 8, 0, 4, 8, 8, listCount)
	}
	if is8x16(mbType) {
		delta := 8 * dst.LumaStride
		if err := h264MCPartFrameStd(dst, refs, cache, mbX, mbY, mbType, 0, 0, false, 16, delta, 0, 0, 8, 4, listCount); err != nil {
			return err
		}
		return h264MCPartFrameStd(dst, refs, cache, mbX, mbY, mbType, 1, 4, false, 16, delta, 4, 0, 8, 4, listCount)
	}
	if !is8x8(mbType) {
		return ErrUnsupported
	}

	for i := 0; i < 4; i++ {
		subType := subMBType[i]
		if isDirect(subType) {
			return ErrUnsupported
		}
		n := 4 * i
		xOffset := (i & 1) << 2
		yOffset := (i & 2) << 1

		if isSub8x8(subType) {
			if err := h264MCPartFrameStd(dst, refs, cache, mbX, mbY, subType, 0, n, true, 8, 0, xOffset, yOffset, 8, 4, listCount); err != nil {
				return err
			}
		} else if isSub8x4(subType) {
			if err := h264MCPartFrameStd(dst, refs, cache, mbX, mbY, subType, 0, n, false, 4, 4, xOffset, yOffset, 4, 4, listCount); err != nil {
				return err
			}
			if err := h264MCPartFrameStd(dst, refs, cache, mbX, mbY, subType, 0, n+2, false, 4, 4, xOffset, yOffset+2, 4, 4, listCount); err != nil {
				return err
			}
		} else if isSub4x8(subType) {
			delta := 4 * dst.LumaStride
			if err := h264MCPartFrameStd(dst, refs, cache, mbX, mbY, subType, 0, n, false, 8, delta, xOffset, yOffset, 4, 2, listCount); err != nil {
				return err
			}
			if err := h264MCPartFrameStd(dst, refs, cache, mbX, mbY, subType, 0, n+1, false, 8, delta, xOffset+2, yOffset, 4, 2, listCount); err != nil {
				return err
			}
		} else if isSub4x4(subType) {
			for j := 0; j < 4; j++ {
				subXOffset := xOffset + 2*(j&1)
				subYOffset := yOffset + (j & 2)
				if err := h264MCPartFrameStd(dst, refs, cache, mbX, mbY, subType, 0, n+j, true, 4, 0, subXOffset, subYOffset, 4, 2, listCount); err != nil {
					return err
				}
			}
		} else {
			return ErrUnsupported
		}
	}
	return nil
}

func h264MCPartFrameStd(dst *h264PicturePlanes, refs [2][]*h264PicturePlanes, cache *macroblockMotionCache, mbX int, mbY int, mbType uint32, part int, n int, square bool, height int, delta int, xOffset int, yOffset int, qpelSize int, chromaWidth int, listCount int) error {
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
		if err := h264MCDirPartFrame(dst, ref, cache, n, square, height, delta, 0, dstY, dstCb, dstCr, srcXOffset, srcYOffset, qpelSize, chromaWidth, avg); err != nil {
			return err
		}
		avg = true
	}
	if list1 {
		ref, err := h264MCReference(refs, cache, 1, n)
		if err != nil {
			return err
		}
		if err := h264MCDirPartFrame(dst, ref, cache, n, square, height, delta, 1, dstY, dstCb, dstCr, srcXOffset, srcYOffset, qpelSize, chromaWidth, avg); err != nil {
			return err
		}
	}
	return nil
}

func h264MCDirPartFrame(dst *h264PicturePlanes, ref *h264PicturePlanes, cache *macroblockMotionCache, n int, square bool, height int, delta int, list int, dstY int, dstCb int, dstCr int, srcXOffset int, srcYOffset int, qpelSize int, chromaWidth int, avg bool) error {
	if dst == nil || ref == nil || cache == nil || n < 0 || n >= 16 || list < 0 || list > 1 || height <= 0 || delta < 0 {
		return ErrInvalidData
	}
	if err := h264CheckMotionPlanePair(dst, ref); err != nil {
		return err
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
	if fullMx < -extraWidth ||
		fullMy < -extraHeight ||
		fullMx+16 > ref.MBWidth*16+extraWidth ||
		fullMy+16 > ref.MBHeight*16+extraHeight {
		return ErrUnsupported
	}

	srcY := fullMx + fullMy*ref.LumaStride
	if err := h264CallQpelMC(dst.Y, dstY, ref.Y, srcY, dst.LumaStride, qpelSize, lumaXY, avg); err != nil {
		return err
	}
	if !square {
		if err := h264CallQpelMC(dst.Y, dstY+delta, ref.Y, srcY+delta, dst.LumaStride, qpelSize, lumaXY, avg); err != nil {
			return err
		}
	}

	switch dst.ChromaFormatIDC {
	case 0:
		return nil
	case 3:
		srcC := fullMx + fullMy*ref.ChromaStride
		if err := h264CallQpelMC(dst.Cb, dstCb, ref.Cb, srcC, dst.ChromaStride, qpelSize, lumaXY, avg); err != nil {
			return err
		}
		if !square {
			if err := h264CallQpelMC(dst.Cb, dstCb+delta, ref.Cb, srcC+delta, dst.ChromaStride, qpelSize, lumaXY, avg); err != nil {
				return err
			}
		}
		if err := h264CallQpelMC(dst.Cr, dstCr, ref.Cr, srcC, dst.ChromaStride, qpelSize, lumaXY, avg); err != nil {
			return err
		}
		if !square {
			return h264CallQpelMC(dst.Cr, dstCr+delta, ref.Cr, srcC+delta, dst.ChromaStride, qpelSize, lumaXY, avg)
		}
		return nil
	case 1, 2:
		yShift := 3
		chromaHeight := height
		chromaY := my & 7
		if dst.ChromaFormatIDC == 1 {
			chromaHeight >>= 1
		} else {
			yShift = 2
			chromaY = (my << 1) & 7
		}
		srcC := (mx >> 3) + (my>>yShift)*ref.ChromaStride
		if srcC < 0 || dstCb < 0 || dstCr < 0 || srcC > len(ref.Cb) || srcC > len(ref.Cr) || dstCb > len(dst.Cb) || dstCr > len(dst.Cr) {
			return ErrInvalidData
		}
		op, err := h264ChromaMCOpForWidth(chromaWidth, avg)
		if err != nil {
			return err
		}
		chromaX := mx & 7
		if err := op(dst.Cb[dstCb:], ref.Cb[srcC:], dst.ChromaStride, chromaHeight, chromaX, chromaY); err != nil {
			return err
		}
		return op(dst.Cr[dstCr:], ref.Cr[srcC:], dst.ChromaStride, chromaHeight, chromaX, chromaY)
	default:
		return ErrInvalidData
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
	mx := lumaXY & 3
	my := (lumaXY >> 2) & 3
	if avg {
		return h264AvgH264QpelMC(dst, dstOffset, src, srcOffset, stride, size, mx, my)
	}
	return h264PutH264QpelMC(dst, dstOffset, src, srcOffset, stride, size, mx, my)
}

func h264ChromaMCOpForWidth(width int, avg bool) (h264ChromaMCOp, error) {
	if avg {
		switch width {
		case 1:
			return h264AvgH264ChromaMC1, nil
		case 2:
			return h264AvgH264ChromaMC2, nil
		case 4:
			return h264AvgH264ChromaMC4, nil
		case 8:
			return h264AvgH264ChromaMC8, nil
		}
		return nil, ErrInvalidData
	}
	switch width {
	case 1:
		return h264PutH264ChromaMC1, nil
	case 2:
		return h264PutH264ChromaMC2, nil
	case 4:
		return h264PutH264ChromaMC4, nil
	case 8:
		return h264PutH264ChromaMC8, nil
	}
	return nil, ErrInvalidData
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
