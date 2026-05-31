// SPDX-License-Identifier: LGPL-2.1-or-later
//
// Source-shaped 8-bit H.264 chroma motion compensation helpers from FFmpeg
// n8.0.1 libavcodec/h264chroma_template.c.

package h264

func h264PutH264ChromaMC1(dst []uint8, src []uint8, stride int, height int, x int, y int) error {
	return h264ChromaMC(dst, src, stride, height, x, y, 1, false)
}

func h264PutH264ChromaMC2(dst []uint8, src []uint8, stride int, height int, x int, y int) error {
	return h264ChromaMC(dst, src, stride, height, x, y, 2, false)
}

func h264PutH264ChromaMC4(dst []uint8, src []uint8, stride int, height int, x int, y int) error {
	return h264ChromaMC(dst, src, stride, height, x, y, 4, false)
}

func h264PutH264ChromaMC8(dst []uint8, src []uint8, stride int, height int, x int, y int) error {
	return h264ChromaMC(dst, src, stride, height, x, y, 8, false)
}

func h264AvgH264ChromaMC1(dst []uint8, src []uint8, stride int, height int, x int, y int) error {
	return h264ChromaMC(dst, src, stride, height, x, y, 1, true)
}

func h264AvgH264ChromaMC2(dst []uint8, src []uint8, stride int, height int, x int, y int) error {
	return h264ChromaMC(dst, src, stride, height, x, y, 2, true)
}

func h264AvgH264ChromaMC4(dst []uint8, src []uint8, stride int, height int, x int, y int) error {
	return h264ChromaMC(dst, src, stride, height, x, y, 4, true)
}

func h264AvgH264ChromaMC8(dst []uint8, src []uint8, stride int, height int, x int, y int) error {
	return h264ChromaMC(dst, src, stride, height, x, y, 8, true)
}

func h264ChromaMC(dst []uint8, src []uint8, stride int, height int, x int, y int, width int, avg bool) error {
	if err := checkChromaMCArgs(dst, src, stride, height, x, y, width); err != nil {
		return err
	}
	a := (8 - x) * (8 - y)
	b := x * (8 - y)
	c := (8 - x) * y
	d := x * y

	if d != 0 {
		for i := 0; i < height; i++ {
			row := i * stride
			next := row + stride
			for j := 0; j < width; j++ {
				v := a*int(src[row+j]) + b*int(src[row+j+1]) +
					c*int(src[next+j]) + d*int(src[next+j+1])
				h264ChromaMCStore(dst, row+j, v, avg)
			}
		}
	} else if b+c != 0 {
		e := b + c
		step := 1
		if c != 0 {
			step = stride
		}
		for i := 0; i < height; i++ {
			row := i * stride
			for j := 0; j < width; j++ {
				v := a*int(src[row+j]) + e*int(src[row+step+j])
				h264ChromaMCStore(dst, row+j, v, avg)
			}
		}
	} else {
		for i := 0; i < height; i++ {
			row := i * stride
			for j := 0; j < width; j++ {
				h264ChromaMCStore(dst, row+j, a*int(src[row+j]), avg)
			}
		}
	}
	return nil
}

func h264ChromaMCStore(dst []uint8, offset int, v int, avg bool) {
	pred := uint8((v + 32) >> 6)
	if avg {
		dst[offset] = uint8((int(dst[offset]) + int(pred) + 1) >> 1)
		return
	}
	dst[offset] = pred
}

func checkChromaMCArgs(dst []uint8, src []uint8, stride int, height int, x int, y int, width int) error {
	if stride <= 0 || height < 0 || x < 0 || x >= 8 || y < 0 || y >= 8 {
		return ErrInvalidData
	}
	if width != 1 && width != 2 && width != 4 && width != 8 {
		return ErrInvalidData
	}
	if height == 0 {
		return nil
	}
	dstNeeded := (height-1)*stride + width
	srcNeeded := dstNeeded
	if x != 0 && y != 0 {
		srcNeeded = height*stride + width + 1
	} else if x != 0 {
		srcNeeded = (height-1)*stride + width + 1
	} else if y != 0 {
		srcNeeded = height*stride + width
	}
	if len(dst) < dstNeeded || len(src) < srcNeeded {
		return ErrInvalidData
	}
	return nil
}
