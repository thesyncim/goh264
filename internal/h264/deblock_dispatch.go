// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

import "unsafe"

const (
	minDeblockCInt = -1 << 31
	maxDeblockCInt = 1<<31 - 1
)

func h264LoopFilterCInts(alpha int, beta int) (int32, int32, error) {
	if alpha < minDeblockCInt || alpha > maxDeblockCInt || beta < minDeblockCInt || beta > maxDeblockCInt {
		return 0, 0, ErrInvalidData
	}
	return int32(alpha), int32(beta), nil
}

func h264LoopFilterLumaDispatch(pix []uint8, offset int, xstride int, ystride int, innerIters int, alpha int, beta int, tc0 *[4]int8) error {
	alpha32, beta32, err := h264LoopFilterCInts(alpha, beta)
	if err != nil {
		return err
	}
	return h264LoopFilterLumaKernel(pix, offset, xstride, ystride, innerIters, alpha32, beta32, tc0)
}

func h264LoopFilterLumaHighDispatch(pix []uint16, offset int, xstride int, ystride int, innerIters int, alpha int, beta int, tc0 *[4]int8, bitDepth int) error {
	alpha32, beta32, err := h264LoopFilterCInts(alpha, beta)
	if err != nil {
		return err
	}
	return h264LoopFilterLumaHighKernel(pix, offset, xstride, ystride, innerIters, alpha32, beta32, tc0, int32(bitDepth))
}

func h264LoopFilterLumaIntraDispatch(pix []uint8, offset int, xstride int, ystride int, innerIters int, alpha int, beta int) error {
	alpha32, beta32, err := h264LoopFilterCInts(alpha, beta)
	if err != nil {
		return err
	}
	return h264LoopFilterLumaIntraKernel(pix, offset, xstride, ystride, innerIters, alpha32, beta32)
}

func h264LoopFilterLumaIntraHighDispatch(pix []uint16, offset int, xstride int, ystride int, innerIters int, alpha int, beta int, bitDepth int) error {
	alpha32, beta32, err := h264LoopFilterCInts(alpha, beta)
	if err != nil {
		return err
	}
	return h264LoopFilterLumaIntraHighKernel(pix, offset, xstride, ystride, innerIters, alpha32, beta32, int32(bitDepth))
}

func h264LoopFilterChromaDispatch(pix []uint8, offset int, xstride int, ystride int, innerIters int, alpha int, beta int, tc0 *[4]int8) error {
	alpha32, beta32, err := h264LoopFilterCInts(alpha, beta)
	if err != nil {
		return err
	}
	return h264LoopFilterChromaKernel(pix, offset, xstride, ystride, innerIters, alpha32, beta32, tc0)
}

func h264LoopFilterChromaHighDispatch(pix []uint16, offset int, xstride int, ystride int, innerIters int, alpha int, beta int, tc0 *[4]int8, bitDepth int) error {
	alpha32, beta32, err := h264LoopFilterCInts(alpha, beta)
	if err != nil {
		return err
	}
	return h264LoopFilterChromaHighKernel(pix, offset, xstride, ystride, innerIters, alpha32, beta32, tc0, int32(bitDepth))
}

func h264LoopFilterChromaIntraDispatch(pix []uint8, offset int, xstride int, ystride int, innerIters int, alpha int, beta int) error {
	alpha32, beta32, err := h264LoopFilterCInts(alpha, beta)
	if err != nil {
		return err
	}
	return h264LoopFilterChromaIntraKernel(pix, offset, xstride, ystride, innerIters, alpha32, beta32)
}

func h264LoopFilterChromaIntraHighDispatch(pix []uint16, offset int, xstride int, ystride int, innerIters int, alpha int, beta int, bitDepth int) error {
	alpha32, beta32, err := h264LoopFilterCInts(alpha, beta)
	if err != nil {
		return err
	}
	return h264LoopFilterChromaIntraHighKernel(pix, offset, xstride, ystride, innerIters, alpha32, beta32, int32(bitDepth))
}

func h264LoopFilterLumaKernel(pix []uint8, offset int, xstride int, ystride int, innerIters int, alpha int32, beta int32, tc0 *[4]int8) error {
	if h264LoopFilterLumaASMEnabled {
		if tc0 == nil {
			return ErrInvalidData
		}
		switch {
		case innerIters == 4:
			if err := checkLoopFilterArgs(pix, offset, xstride, ystride, innerIters, 4, 3, 2); err != nil {
				return err
			}
			switch {
			case ystride == 1:
				h264VLoopFilterLuma8ASM(&pix[offset], xstride, alpha, beta, &tc0[0])
				return nil
			case xstride == 1:
				h264HLoopFilterLuma8ASM(&pix[offset], ystride, alpha, beta, &tc0[0])
				return nil
			}
		case innerIters == 2 && xstride == 1 && h264LoopFilterLumaMBAFF8ASMEnabled:
			if err := checkLoopFilterArgs(pix, offset, xstride, ystride, innerIters, 4, 4, 3); err == nil {
				h264HLoopFilterLumaMBAFF8ASM(&pix[offset], ystride, alpha, beta, &tc0[0])
				return nil
			}
		}
		return h264LoopFilterLuma(pix, offset, xstride, ystride, innerIters, int(alpha), int(beta), tc0)
	}
	return h264LoopFilterLuma(pix, offset, xstride, ystride, innerIters, int(alpha), int(beta), tc0)
}

func h264LoopFilterLumaHighKernel(pix []uint16, offset int, xstride int, ystride int, innerIters int, alpha int32, beta int32, tc0 *[4]int8, bitDepth int32) error {
	return h264LoopFilterLumaHigh(pix, offset, xstride, ystride, innerIters, int(alpha), int(beta), tc0, int(bitDepth))
}

func h264LoopFilterLumaIntraKernel(pix []uint8, offset int, xstride int, ystride int, innerIters int, alpha int32, beta int32) error {
	if h264LoopFilterLumaIntraASMEnabled && innerIters == 4 {
		if err := checkLoopFilterArgs(pix, offset, xstride, ystride, innerIters, 4, 4, 3); err != nil {
			return err
		}
		switch {
		case ystride == 1 && h264LoopFilterLumaIntraV8ASMEnabled:
			h264VLoopFilterLumaIntra8ASM(&pix[offset], xstride, alpha, beta)
			return nil
		case xstride == 1 && h264LoopFilterLumaIntraH8ASMEnabled:
			h264HLoopFilterLumaIntra8ASM(&pix[offset], ystride, alpha, beta)
			return nil
		}
	}
	return h264LoopFilterLumaIntra(pix, offset, xstride, ystride, innerIters, int(alpha), int(beta))
}

func h264LoopFilterLumaIntraHighKernel(pix []uint16, offset int, xstride int, ystride int, innerIters int, alpha int32, beta int32, bitDepth int32) error {
	return h264LoopFilterLumaIntraHigh(pix, offset, xstride, ystride, innerIters, int(alpha), int(beta), int(bitDepth))
}

func h264LoopFilterChromaKernel(pix []uint8, offset int, xstride int, ystride int, innerIters int, alpha int32, beta int32, tc0 *[4]int8) error {
	if h264LoopFilterChromaASMEnabled {
		switch {
		case innerIters == 2 && ystride == 1 && h264LoopFilterChromaV8ASMEnabled:
			if tc0 == nil {
				return ErrInvalidData
			}
			if tc0[0] >= 0 && tc0[1] >= 0 && tc0[2] >= 0 && tc0[3] >= 0 {
				if err := checkLoopFilterArgs(pix, offset, xstride, ystride, innerIters, 4, 2, 1); err != nil {
					return err
				}
				h264VLoopFilterChroma8ASM(&pix[offset], xstride, alpha, beta, &tc0[0])
				return nil
			}
		case innerIters == 2 && xstride == 1 && h264LoopFilterChromaH8ASMEnabled:
			if tc0 == nil {
				return ErrInvalidData
			}
			if tc0[0] >= 0 && tc0[1] >= 0 && tc0[2] >= 0 && tc0[3] >= 0 {
				if err := checkLoopFilterArgs(pix, offset, xstride, ystride, innerIters, 4, 2, 1); err != nil {
					return err
				}
				h264HLoopFilterChroma8ASM(&pix[offset], ystride, alpha, beta, &tc0[0])
				return nil
			}
		case innerIters == 4 && xstride == 1 && h264LoopFilterChroma422H8ASMEnabled:
			if tc0 == nil {
				return ErrInvalidData
			}
			if tc0[0] >= 0 && tc0[1] >= 0 && tc0[2] >= 0 && tc0[3] >= 0 {
				if err := checkLoopFilterArgs(pix, offset, xstride, ystride, innerIters, 4, 2, 1); err != nil {
					return err
				}
				h264HLoopFilterChroma4228ASM(&pix[offset], ystride, alpha, beta, &tc0[0])
				return nil
			}
		}
	}
	return h264LoopFilterChroma(pix, offset, xstride, ystride, innerIters, int(alpha), int(beta), tc0)
}

func h264LoopFilterChromaHighKernel(pix []uint16, offset int, xstride int, ystride int, innerIters int, alpha int32, beta int32, tc0 *[4]int8, bitDepth int32) error {
	if tc0 == nil {
		return ErrInvalidData
	}
	if h264LoopFilterChromaHighASMEnabled && bitDepth == 10 &&
		tc0[0] >= 0 && tc0[1] >= 0 && tc0[2] >= 0 && tc0[3] >= 0 {
		switch {
		case innerIters == 2 && ystride == 1:
			if err := checkLoopFilterHighArgs(pix, offset, xstride, ystride, innerIters, 4, 2, 1, int(bitDepth)); err != nil {
				return err
			}
			h264VLoopFilterChromaHigh10ASM((*uint8)(unsafe.Pointer(&pix[offset])), xstride*2, alpha, beta, &tc0[0])
			return nil
		case innerIters == 2 && xstride == 1:
			if err := checkLoopFilterHighArgs(pix, offset, xstride, ystride, innerIters, 4, 2, 1, int(bitDepth)); err != nil {
				return err
			}
			h264HLoopFilterChromaHigh10ASM((*uint8)(unsafe.Pointer(&pix[offset])), ystride*2, alpha, beta, &tc0[0])
			return nil
		case innerIters == 4 && xstride == 1:
			if err := checkLoopFilterHighArgs(pix, offset, xstride, ystride, innerIters, 4, 2, 1, int(bitDepth)); err != nil {
				return err
			}
			h264HLoopFilterChroma422High10ASM((*uint8)(unsafe.Pointer(&pix[offset])), ystride*2, alpha, beta, &tc0[0])
			return nil
		}
	}
	return h264LoopFilterChromaHigh(pix, offset, xstride, ystride, innerIters, int(alpha), int(beta), tc0, int(bitDepth))
}

func h264LoopFilterChromaIntraKernel(pix []uint8, offset int, xstride int, ystride int, innerIters int, alpha int32, beta int32) error {
	if innerIters == 1 {
		return h264LoopFilterChromaIntra(pix, offset, xstride, ystride, innerIters, int(alpha), int(beta))
	}
	if h264LoopFilterChromaASMEnabled {
		switch {
		case innerIters == 2 && ystride == 1 && h264LoopFilterChromaIntraV8ASMEnabled:
			if err := checkLoopFilterArgs(pix, offset, xstride, ystride, innerIters, 4, 2, 1); err != nil {
				return err
			}
			h264VLoopFilterChromaIntra8ASM(&pix[offset], xstride, alpha, beta)
			return nil
		case innerIters == 2 && xstride == 1 && h264LoopFilterChromaIntraH8ASMEnabled:
			if err := checkLoopFilterArgs(pix, offset, xstride, ystride, innerIters, 4, 2, 1); err != nil {
				return err
			}
			h264HLoopFilterChromaIntra8ASM(&pix[offset], ystride, alpha, beta)
			return nil
		case innerIters == 4 && xstride == 1 && h264LoopFilterChroma422IntraH8ASMEnabled:
			if err := checkLoopFilterArgs(pix, offset, xstride, ystride, innerIters, 4, 2, 1); err != nil {
				return err
			}
			h264HLoopFilterChroma422Intra8ASM(&pix[offset], ystride, alpha, beta)
			return nil
		}
	}
	return h264LoopFilterChromaIntra(pix, offset, xstride, ystride, innerIters, int(alpha), int(beta))
}

func h264LoopFilterChromaIntraHighKernel(pix []uint16, offset int, xstride int, ystride int, innerIters int, alpha int32, beta int32, bitDepth int32) error {
	if innerIters == 1 {
		return h264LoopFilterChromaIntraHigh(pix, offset, xstride, ystride, innerIters, int(alpha), int(beta), int(bitDepth))
	}
	if h264LoopFilterChromaHighASMEnabled && bitDepth == 10 {
		switch {
		case innerIters == 2 && ystride == 1:
			if err := checkLoopFilterHighArgs(pix, offset, xstride, ystride, innerIters, 4, 2, 1, int(bitDepth)); err != nil {
				return err
			}
			h264VLoopFilterChromaIntraHigh10ASM((*uint8)(unsafe.Pointer(&pix[offset])), xstride*2, alpha, beta)
			return nil
		case innerIters == 2 && xstride == 1:
			if err := checkLoopFilterHighArgs(pix, offset, xstride, ystride, innerIters, 4, 2, 1, int(bitDepth)); err != nil {
				return err
			}
			h264HLoopFilterChromaIntraHigh10ASM((*uint8)(unsafe.Pointer(&pix[offset])), ystride*2, alpha, beta)
			return nil
		case innerIters == 4 && xstride == 1:
			if err := checkLoopFilterHighArgs(pix, offset, xstride, ystride, innerIters, 4, 2, 1, int(bitDepth)); err != nil {
				return err
			}
			h264HLoopFilterChroma422IntraHigh10ASM((*uint8)(unsafe.Pointer(&pix[offset])), ystride*2, alpha, beta)
			return nil
		}
	}
	return h264LoopFilterChromaIntraHigh(pix, offset, xstride, ystride, innerIters, int(alpha), int(beta), int(bitDepth))
}
