// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

import "testing"

func TestH264QpelPutRepresentativeModes(t *testing.T) {
	const stride = 48
	const offset = 6*stride + 6

	for _, c := range []struct {
		name string
		size int
		mx   int
		my   int
		want [4]uint8
	}{
		{name: "h_lowpass", size: 4, mx: 2, my: 0, want: [4]uint8{101, 128, 117, 144}},
		{name: "hv_lowpass", size: 8, mx: 2, my: 2, want: [4]uint8{25, 99, 141, 36}},
		{name: "diagonal_half", size: 16, mx: 3, my: 3, want: [4]uint8{27, 194, 143, 190}},
	} {
		t.Run(c.name, func(t *testing.T) {
			dst, src := makeQpelUnitFixture(stride, 48)
			if err := h264PutH264QpelMC(dst, offset, src, offset, stride, c.size, c.mx, c.my); err != nil {
				t.Fatal(err)
			}
			got := [4]uint8{
				dst[offset],
				dst[offset+c.size-1],
				dst[offset+(c.size-1)*stride],
				dst[offset+(c.size-1)*stride+c.size-1],
			}
			if got != c.want {
				t.Fatalf("selected samples = %v, want %v", got, c.want)
			}
		})
	}
}

func TestH264QpelAvgRepresentativeMode(t *testing.T) {
	const stride = 48
	const offset = 6*stride + 6
	dst, src := makeQpelUnitFixture(stride, 48)

	if err := h264AvgH264QpelMC(dst, offset, src, offset, stride, 4, 1, 3); err != nil {
		t.Fatal(err)
	}
	got := [4]uint8{dst[offset], dst[offset+3], dst[offset+3*stride], dst[offset+3*stride+3]}
	want := [4]uint8{103, 131, 133, 35}
	if got != want {
		t.Fatalf("selected avg samples = %v, want %v", got, want)
	}
	if dst[offset+4] != 226 {
		t.Fatalf("padding byte changed to %d, want 226", dst[offset+4])
	}
}

func TestH264QpelHighRepresentativeModes(t *testing.T) {
	const stride = 48
	const offset = 6*stride + 6
	dst := make([]uint16, stride*48)
	src := make([]uint16, stride*48)
	for i := range src {
		src[i] = 1023
	}

	if err := h264PutH264QpelMCHigh(dst, offset, src, offset, stride, 4, 2, 0, 10); err != nil {
		t.Fatal(err)
	}
	if dst[offset] != 1023 || dst[offset+3] != 1023 || dst[offset+3*stride+3] != 1023 {
		t.Fatalf("high h-lowpass samples = %d/%d/%d, want 1023", dst[offset], dst[offset+3], dst[offset+3*stride+3])
	}

	for i := range dst {
		dst[i] = 1
	}
	if err := h264AvgH264QpelMCHigh(dst, offset, src, offset, stride, 8, 2, 2, 10); err != nil {
		t.Fatal(err)
	}
	if dst[offset] != 512 || dst[offset+7] != 512 || dst[offset+7*stride+7] != 512 {
		t.Fatalf("high hv avg samples = %d/%d/%d, want 512", dst[offset], dst[offset+7], dst[offset+7*stride+7])
	}
	if dst[offset+8] != 1 {
		t.Fatalf("high padding sample changed to %d, want 1", dst[offset+8])
	}
}

func TestH264QpelValidatesGeometry(t *testing.T) {
	if err := h264PutH264QpelMC(make([]uint8, 16), 0, make([]uint8, 16), 0, 4, 4, 0, 0); err != nil {
		t.Fatalf("mc00 without margins error = %v, want nil", err)
	}
	if err := h264PutH264QpelMC(make([]uint8, 16), 0, make([]uint8, 16), 0, 4, 4, 2, 0); err != ErrInvalidData {
		t.Fatalf("missing horizontal margin error = %v, want ErrInvalidData", err)
	}
	if err := h264PutH264QpelMC(make([]uint8, 16), 0, make([]uint8, 64), 0, 4, 3, 0, 0); err != ErrInvalidData {
		t.Fatalf("invalid size error = %v, want ErrInvalidData", err)
	}
	if err := h264PutH264QpelMC(make([]uint8, 16), 0, make([]uint8, 64), 0, 4, 4, 4, 0); err != ErrInvalidData {
		t.Fatalf("invalid motion fraction error = %v, want ErrInvalidData", err)
	}
	if err := h264PutH264QpelMCHigh(make([]uint16, 16), 0, make([]uint16, 16), 0, 4, 4, 0, 0, 11); err != ErrUnsupported {
		t.Fatalf("unsupported high qpel bit depth err = %v, want ErrUnsupported", err)
	}
	if err := h264PutH264QpelMCHigh(make([]uint16, 16), 0, make([]uint16, 16), 0, 4, 4, 2, 0, 10); err != ErrInvalidData {
		t.Fatalf("missing high horizontal margin error = %v, want ErrInvalidData", err)
	}
}

func makeQpelUnitFixture(stride int, rows int) ([]uint8, []uint8) {
	dst := make([]uint8, stride*rows)
	src := make([]uint8, stride*rows)
	for i := range dst {
		dst[i] = uint8((20 + i*11) & 255)
		src[i] = uint8((10 + i*9) & 255)
	}
	return dst, src
}
