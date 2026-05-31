// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

import "testing"

func TestH264ChromaMCPutBranches(t *testing.T) {
	for _, c := range []struct {
		name string
		x    int
		y    int
		want [6]uint8
	}{
		{name: "copy", x: 0, y: 0, want: [6]uint8{10, 37, 118, 145, 26, 53}},
		{name: "horizontal", x: 3, y: 0, want: [6]uint8{13, 40, 121, 148, 29, 56}},
		{name: "vertical", x: 0, y: 5, want: [6]uint8{78, 105, 61, 88, 94, 121}},
		{name: "bilinear", x: 3, y: 5, want: [6]uint8{81, 108, 64, 91, 97, 124}},
	} {
		t.Run(c.name, func(t *testing.T) {
			const stride = 12
			dst := makeChromaUnitDst(stride, 6)
			src := makeChromaUnitSrc(stride, 7)

			if err := h264PutH264ChromaMC4(dst, src, stride, 3, c.x, c.y); err != nil {
				t.Fatal(err)
			}
			got := [6]uint8{dst[0], dst[3], dst[stride], dst[stride+3], dst[2*stride], dst[2*stride+3]}
			if got != c.want {
				t.Fatalf("selected samples = %v, want %v", got, c.want)
			}
			if dst[4] != 0 {
				t.Fatalf("padding byte changed to %d, want 0", dst[4])
			}
		})
	}
}

func TestH264ChromaMCAvgBilinear(t *testing.T) {
	const stride = 12
	dst := makeChromaUnitDst(stride, 6)
	src := makeChromaUnitSrc(stride, 7)

	if err := h264AvgH264ChromaMC2(dst, src, stride, 3, 3, 5); err != nil {
		t.Fatal(err)
	}
	got := [6]uint8{dst[0], dst[1], dst[stride], dst[stride+1], dst[2*stride], dst[2*stride+1]}
	want := [6]uint8{51, 53, 102, 104, 89, 91}
	if got != want {
		t.Fatalf("selected avg samples = %v, want %v", got, want)
	}
	if dst[2] != 10 {
		t.Fatalf("padding byte changed to %d, want 10", dst[2])
	}
}

func TestH264ChromaMCValidatesGeometry(t *testing.T) {
	if err := h264PutH264ChromaMC8(make([]uint8, 8), make([]uint8, 8), 8, 1, 8, 0); err != ErrInvalidData {
		t.Fatalf("invalid x error = %v, want ErrInvalidData", err)
	}
	if err := h264PutH264ChromaMC8(make([]uint8, 8), make([]uint8, 8), 8, 1, 0, -1); err != ErrInvalidData {
		t.Fatalf("invalid y error = %v, want ErrInvalidData", err)
	}
	if err := h264PutH264ChromaMC8(make([]uint8, 8), make([]uint8, 8), 8, 1, 1, 1); err != ErrInvalidData {
		t.Fatalf("short bilinear source error = %v, want ErrInvalidData", err)
	}
	if err := h264ChromaMC(make([]uint8, 8), make([]uint8, 16), 8, 1, 0, 0, 3, false); err != ErrInvalidData {
		t.Fatalf("invalid width error = %v, want ErrInvalidData", err)
	}
}

func makeChromaUnitDst(stride int, rows int) []uint8 {
	dst := make([]uint8, stride*rows)
	for i := range dst {
		dst[i] = uint8((200 - i*5) % 180)
	}
	return dst
}

func makeChromaUnitSrc(stride int, rows int) []uint8 {
	src := make([]uint8, stride*rows)
	for i := range src {
		src[i] = uint8((10 + i*9) % 200)
	}
	return src
}
