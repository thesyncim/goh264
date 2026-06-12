// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

import (
	"errors"
	"testing"
)

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

func TestBitReaderRemainingAlignedBytes(t *testing.T) {
	gb := newBitReader([]byte{0xe0, 0x2a, 0x40, 0x80, 0x11})
	gb.numBits = 35
	if got, err := gb.readBits(3); err != nil || got != 0b111 {
		t.Fatalf("prefix bits = %b, %v; want 111, nil", got, err)
	}
	got, err := gb.remainingAlignedBytes()
	if err != nil {
		t.Fatal(err)
	}
	if gb.bitPos != 8 {
		t.Fatalf("bitPos = %d, want byte-aligned 8", gb.bitPos)
	}
	if len(got) != 4 || got[0] != 0x2a || got[3] != 0x11 {
		t.Fatalf("remaining bytes = % x, want 2a 40 80 11", got)
	}
}

func TestBitReaderRemainingAlignedRawBytesKeepsRBSPTrailingByte(t *testing.T) {
	gb, err := newRBSPBitReader([]byte{0xe0, 0x2a, 0x40, 0x80})
	if err != nil {
		t.Fatal(err)
	}
	if got, err := gb.readBits(3); err != nil || got != 0b111 {
		t.Fatalf("prefix bits = %b, %v; want 111, nil", got, err)
	}
	trimmed, err := gb.remainingAlignedBytes()
	if err != nil {
		t.Fatal(err)
	}
	if len(trimmed) != 2 || trimmed[0] != 0x2a || trimmed[1] != 0x40 {
		t.Fatalf("trimmed remaining bytes = % x, want 2a 40", trimmed)
	}

	gb, err = newRBSPBitReader([]byte{0xe0, 0x2a, 0x40, 0x80})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := gb.readBits(3); err != nil {
		t.Fatal(err)
	}
	raw, err := gb.remainingAlignedRawBytes()
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) != 3 || raw[0] != 0x2a || raw[1] != 0x40 || raw[2] != 0x80 {
		t.Fatalf("raw remaining bytes = % x, want 2a 40 80", raw)
	}
}

func TestBitReaderRejectsOverflowedBitLength(t *testing.T) {
	gb := newBitReader(fakeRBSPBytesLen(maxBitReaderByteLen + 1))
	if got := gb.bitsLeft(); got >= 0 {
		t.Fatalf("overflowed reader bitsLeft = %d, want invalid negative state", got)
	}
	if _, err := gb.readBit(); !errors.Is(err, ErrInvalidData) {
		t.Fatalf("overflowed reader readBit err = %v, want ErrInvalidData", err)
	}
	if _, err := gb.readBits(1); !errors.Is(err, ErrInvalidData) {
		t.Fatalf("overflowed reader readBits err = %v, want ErrInvalidData", err)
	}
	if _, err := gb.showBits(1); !errors.Is(err, ErrInvalidData) {
		t.Fatalf("overflowed reader showBits err = %v, want ErrInvalidData", err)
	}
	if got := gb.showBitsPadded(1); got != 0 {
		t.Fatalf("overflowed reader padded bits = %d, want 0", got)
	}
	if err := gb.skipBits(1); !errors.Is(err, ErrInvalidData) {
		t.Fatalf("overflowed reader skipBits err = %v, want ErrInvalidData", err)
	}
	if _, err := gb.readAlignedBytes(1); !errors.Is(err, ErrInvalidData) {
		t.Fatalf("overflowed reader aligned bytes err = %v, want ErrInvalidData", err)
	}
	if _, err := gb.remainingAlignedBytes(); !errors.Is(err, ErrInvalidData) {
		t.Fatalf("overflowed reader remaining bytes err = %v, want ErrInvalidData", err)
	}
	if _, err := gb.remainingAlignedRawBytes(); !errors.Is(err, ErrInvalidData) {
		t.Fatalf("overflowed reader raw bytes err = %v, want ErrInvalidData", err)
	}
	if _, err := newRBSPBitReader(fakeRBSPBytesLen(maxBitReaderByteLen + 1)); !errors.Is(err, ErrInvalidData) {
		t.Fatalf("overflowed RBSP reader err = %v, want ErrInvalidData", err)
	}
}
