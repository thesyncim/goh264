// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

import "testing"

func BenchmarkH264Pred4x4Simple(b *testing.B) {
	const stride = 32
	const offset = 4*stride + 4
	for _, tc := range []struct {
		name string
		mode int
	}{
		{"Vertical", int(intraPredVertical)},
		{"Horizontal", int(intraPredHorizontal)},
		{"DC", int(intraPredDC)},
		{"LeftDC", int(intraPredLeftDC)},
		{"TopDC", int(intraPredTopDC)},
		{"DC128", int(intraPredDC128)},
	} {
		b.Run(tc.name, func(b *testing.B) {
			pix := make([]uint8, stride*12)
			for i := range pix {
				pix[i] = uint8(i*37 + 11)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				if err := h264Pred4x4ByMode(pix, offset, stride, tc.mode, nil); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
