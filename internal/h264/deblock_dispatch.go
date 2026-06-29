// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

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
	if h264LoopFilterLumaASMEnabled && innerIters == 4 {
		if tc0 == nil {
			return ErrInvalidData
		}
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
		return h264LoopFilterLuma(pix, offset, xstride, ystride, innerIters, int(alpha), int(beta), tc0)
	}
	return h264LoopFilterLuma(pix, offset, xstride, ystride, innerIters, int(alpha), int(beta), tc0)
}

func h264LoopFilterLumaHighKernel(pix []uint16, offset int, xstride int, ystride int, innerIters int, alpha int32, beta int32, tc0 *[4]int8, bitDepth int32) error {
	return h264LoopFilterLumaHigh(pix, offset, xstride, ystride, innerIters, int(alpha), int(beta), tc0, int(bitDepth))
}

func h264LoopFilterLumaIntraKernel(pix []uint8, offset int, xstride int, ystride int, innerIters int, alpha int32, beta int32) error {
	return h264LoopFilterLumaIntra(pix, offset, xstride, ystride, innerIters, int(alpha), int(beta))
}

func h264LoopFilterLumaIntraHighKernel(pix []uint16, offset int, xstride int, ystride int, innerIters int, alpha int32, beta int32, bitDepth int32) error {
	return h264LoopFilterLumaIntraHigh(pix, offset, xstride, ystride, innerIters, int(alpha), int(beta), int(bitDepth))
}

func h264LoopFilterChromaKernel(pix []uint8, offset int, xstride int, ystride int, innerIters int, alpha int32, beta int32, tc0 *[4]int8) error {
	return h264LoopFilterChroma(pix, offset, xstride, ystride, innerIters, int(alpha), int(beta), tc0)
}

func h264LoopFilterChromaHighKernel(pix []uint16, offset int, xstride int, ystride int, innerIters int, alpha int32, beta int32, tc0 *[4]int8, bitDepth int32) error {
	return h264LoopFilterChromaHigh(pix, offset, xstride, ystride, innerIters, int(alpha), int(beta), tc0, int(bitDepth))
}

func h264LoopFilterChromaIntraKernel(pix []uint8, offset int, xstride int, ystride int, innerIters int, alpha int32, beta int32) error {
	return h264LoopFilterChromaIntra(pix, offset, xstride, ystride, innerIters, int(alpha), int(beta))
}

func h264LoopFilterChromaIntraHighKernel(pix []uint16, offset int, xstride int, ystride int, innerIters int, alpha int32, beta int32, bitDepth int32) error {
	return h264LoopFilterChromaIntraHigh(pix, offset, xstride, ystride, innerIters, int(alpha), int(beta), int(bitDepth))
}
