// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

import "testing"

func TestBitReaderReadAlignedBytes(t *testing.T) {
	gb := newBitReader([]byte{0b10100000, 0x11, 0x22, 0x33})
	if got, err := gb.readBits(3); err != nil || got != 0b101 {
		t.Fatalf("prefix bits = %b, %v; want 101, nil", got, err)
	}
	got, err := gb.readAlignedBytes(2)
	if err != nil {
		t.Fatal(err)
	}
	if got[0] != 0x11 || got[1] != 0x22 {
		t.Fatalf("aligned bytes = % x, want 11 22", got)
	}
	if gb.bitPos != 24 {
		t.Fatalf("bitPos = %d, want 24", gb.bitPos)
	}
}

func TestBitReaderReadAlignedBytesRejectsShortBuffer(t *testing.T) {
	gb := newBitReader([]byte{0x80})
	if _, err := gb.readBit(); err != nil {
		t.Fatal(err)
	}
	if _, err := gb.readAlignedBytes(1); err != ErrInvalidData {
		t.Fatalf("short aligned read err = %v, want ErrInvalidData", err)
	}
	if gb.bitPos != 1 {
		t.Fatalf("bitPos after failed aligned read = %d, want 1", gb.bitPos)
	}
}
