// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

import "testing"

func TestReadUEGolombLong(t *testing.T) {
	gb := bitReaderFromBits(t, "1 010 011 00100 00101 00110")
	want := []uint32{0, 1, 2, 3, 4, 5}
	for i, w := range want {
		got, err := gb.readUEGolombLong()
		if err != nil {
			t.Fatalf("read %d: %v", i, err)
		}
		if got != w {
			t.Fatalf("read %d = %d, want %d", i, got, w)
		}
	}
}

func TestReadSEGolombLong(t *testing.T) {
	gb := bitReaderFromBits(t, "1 010 011 00100 00101")
	want := []int32{0, 1, -1, 2, -2}
	for i, w := range want {
		got, err := gb.readSEGolombLong()
		if err != nil {
			t.Fatalf("read %d: %v", i, err)
		}
		if got != w {
			t.Fatalf("read %d = %d, want %d", i, got, w)
		}
	}
}

func bitReaderFromBits(t *testing.T, bits string) bitReader {
	t.Helper()
	var clean []byte
	for _, r := range bits {
		switch r {
		case '0', '1':
			clean = append(clean, byte(r))
		case ' ', '_', '\n', '\t':
		default:
			t.Fatalf("invalid bit rune %q", r)
		}
	}

	buf := make([]byte, (len(clean)+7)/8)
	for i, bit := range clean {
		if bit == '1' {
			buf[i>>3] |= 1 << uint(7-(i&7))
		}
	}
	gb := newBitReader(buf)
	gb.numBits = uint32(len(clean))
	return gb
}
