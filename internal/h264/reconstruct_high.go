// SPDX-License-Identifier: LGPL-2.1-or-later
//
// Source-shaped high-bit-depth frame-MB reconstruction helpers from FFmpeg
// n8.0.1 libavcodec/h264_mb_template.c IntraPCM branches.

package h264

type h264PicturePlanesHigh struct {
	Y, Cb, Cr       []uint16
	LumaStride      int
	ChromaStride    int
	MBWidth         int
	MBHeight        int
	ChromaFormatIDC int
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
