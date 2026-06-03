// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

import "testing"

func TestCAVLCAllVLCEntriesDecodeToTheirSymbols(t *testing.T) {
	assertCAVLCVLCEntries(t, "chroma_dc_coeff_token", cavlcVLC{
		length: cavlcChromaDCCoeffTokenLen[:],
		bits:   cavlcChromaDCCoeffTokenBits[:],
		maxLen: chromaDCCoeffTokenVLCBits,
	})
	assertCAVLCVLCEntries(t, "chroma422_dc_coeff_token", cavlcVLC{
		length: cavlcChroma422DCCoeffTokenLen[:],
		bits:   cavlcChroma422DCCoeffTokenBits[:],
		maxLen: chroma422DCCoeffTokenVLCBits,
	})
	for i := range cavlcCoeffTokenLen {
		assertCAVLCVLCEntries(t, "coeff_token", cavlcVLC{
			length: cavlcCoeffTokenLen[i][:],
			bits:   cavlcCoeffTokenBits[i][:],
			maxLen: coeffTokenVLCBits,
		})
	}
	for i := 0; i < 15; i++ {
		assertCAVLCVLCEntries(t, "total_zeros", cavlcVLC{
			length: cavlcTotalZerosLen[i][:],
			bits:   cavlcTotalZerosBits[i][:],
			maxLen: totalZerosVLCBits,
		})
	}
	for i := range cavlcChromaDCTotalZerosLen {
		assertCAVLCVLCEntries(t, "chroma_dc_total_zeros", cavlcVLC{
			length: cavlcChromaDCTotalZerosLen[i][:],
			bits:   cavlcChromaDCTotalZerosBits[i][:],
			maxLen: chromaDCTotalZerosVLCBits,
		})
	}
	for i := range cavlcChroma422DCTotalZerosLen {
		assertCAVLCVLCEntries(t, "chroma422_dc_total_zeros", cavlcVLC{
			length: cavlcChroma422DCTotalZerosLen[i][:],
			bits:   cavlcChroma422DCTotalZerosBits[i][:],
			maxLen: chroma422DCTotalZerosVLCBits,
		})
	}
	for i := range cavlcRunLen {
		maxLen := uint8(run7VLCBits)
		if i < 6 {
			maxLen = runVLCBits
		}
		assertCAVLCVLCEntries(t, "run", cavlcVLC{
			length: cavlcRunLen[i][:],
			bits:   cavlcRunBits[i][:],
			maxLen: maxLen,
		})
	}
}

func assertCAVLCVLCEntries(t *testing.T, name string, vlc cavlcVLC) {
	t.Helper()
	for symbol, length := range vlc.length {
		if length == 0 {
			continue
		}
		gb := newBitReader(cavlcCodeBytes(vlc.bits[symbol], length))
		got, err := vlc.read(&gb)
		if err != nil {
			t.Fatalf("%s symbol %d read failed: %v", name, symbol, err)
		}
		if got != symbol {
			t.Fatalf("%s code symbol = %d, want %d", name, got, symbol)
		}
		if gb.bitPos != uint32(length) {
			t.Fatalf("%s symbol %d consumed %d bits, want %d", name, symbol, gb.bitPos, length)
		}
	}
}

func TestCAVLCLevelTableMatchesFFmpegSpots(t *testing.T) {
	cases := []struct {
		suffix int
		index  int
		value  int8
		bits   int8
	}{
		{0, 0, 108, 8},
		{0, 1, -4, 8},
		{0, 128, 1, 1},
		{1, 1, 107, 8},
		{1, 255, -1, 2},
		{2, 4, 11, 8},
		{6, 8, 104, 5},
		{6, 255, -32, 7},
	}

	for _, tc := range cases {
		got := cavlcLevelTable[tc.suffix][tc.index]
		if got[0] != tc.value || got[1] != tc.bits {
			t.Fatalf("level[%d][%d] = {%d,%d}, want {%d,%d}", tc.suffix, tc.index, got[0], got[1], tc.value, tc.bits)
		}
	}
}

func TestCAVLCDecodeResidualTrailingOnes(t *testing.T) {
	scan := cavlcIdentityScan()

	t.Run("single trailing one", func(t *testing.T) {
		gb := newBitReader(cavlcBitString("0101"))
		var block [16]int32
		totalCoeff, err := decodeCAVLCResidual(&gb, block[:], 0, scan[:], nil, 16, 0)
		if err != nil {
			t.Fatalf("decode residual failed: %v", err)
		}
		if totalCoeff != 1 {
			t.Fatalf("totalCoeff = %d, want 1", totalCoeff)
		}
		if block[0] != 1 {
			t.Fatalf("block[0] = %d, want 1", block[0])
		}
		if gb.bitPos != 4 {
			t.Fatalf("consumed %d bits, want 4", gb.bitPos)
		}
	})

	t.Run("two trailing ones reversed into scan order", func(t *testing.T) {
		gb := newBitReader(cavlcBitString("00101111"))
		var block [16]int32
		totalCoeff, err := decodeCAVLCResidual(&gb, block[:], 0, scan[:], nil, 16, 0)
		if err != nil {
			t.Fatalf("decode residual failed: %v", err)
		}
		if totalCoeff != 2 {
			t.Fatalf("totalCoeff = %d, want 2", totalCoeff)
		}
		if block[0] != -1 || block[1] != 1 {
			t.Fatalf("block[0:2] = %v, want [-1 1]", block[:2])
		}
	})

	t.Run("run before places zero gap", func(t *testing.T) {
		gb := newBitReader(cavlcBitString("001011100"))
		var block [16]int32
		totalCoeff, err := decodeCAVLCResidual(&gb, block[:], 0, scan[:], nil, 16, 0)
		if err != nil {
			t.Fatalf("decode residual failed: %v", err)
		}
		if totalCoeff != 2 {
			t.Fatalf("totalCoeff = %d, want 2", totalCoeff)
		}
		if block[0] != -1 || block[1] != 0 || block[2] != 1 {
			t.Fatalf("block[0:3] = %v, want [-1 0 1]", block[:3])
		}
	})
}

func TestCAVLCDecodeResidualLevelAndQMul(t *testing.T) {
	scan := cavlcIdentityScan()
	t.Run("positive level", func(t *testing.T) {
		qmul := [16]uint32{128}
		gb := newBitReader(cavlcBitString("00010111"))
		var block [16]int32

		totalCoeff, err := decodeCAVLCResidual(&gb, block[:], 0, scan[:], qmul[:], 16, 0)
		if err != nil {
			t.Fatalf("decode residual failed: %v", err)
		}
		if totalCoeff != 1 {
			t.Fatalf("totalCoeff = %d, want 1", totalCoeff)
		}
		if block[0] != 4 {
			t.Fatalf("block[0] = %d, want 4", block[0])
		}
	})

	t.Run("negative level uses ffmpeg signed wrap point", func(t *testing.T) {
		qmul := [16]uint32{128}
		gb := newBitReader(cavlcBitString("0111"))
		var block [16]int32

		totalCoeff, err := decodeCAVLCResidual(&gb, block[:], 0, scan[:], qmul[:], 16, 0)
		if err != nil {
			t.Fatalf("decode residual failed: %v", err)
		}
		if totalCoeff != 1 {
			t.Fatalf("totalCoeff = %d, want 1", totalCoeff)
		}
		if block[0] != -2 {
			t.Fatalf("block[0] = %d, want -2", block[0])
		}
	})
}

func cavlcIdentityScan() [16]uint8 {
	var scan [16]uint8
	for i := range scan {
		scan[i] = uint8(i)
	}
	return scan
}

func cavlcCodeBytes(bits uint8, length uint8) []byte {
	var out uint32
	for i := int(length) - 1; i >= 0; i-- {
		out = (out << 1) | uint32((bits>>i)&1)
	}
	return cavlcBitsToBytes(out, length)
}

func cavlcBitString(s string) []byte {
	out := make([]byte, (len(s)+7)/8)
	for i := 0; i < len(s); i++ {
		if s[i] == '1' {
			out[i>>3] |= 1 << (7 - uint(i&7))
		}
	}
	return out
}

func cavlcBitsToBytes(bits uint32, length uint8) []byte {
	size := (int(length) + 7) / 8
	out := make([]byte, size)
	for i := 0; i < int(length); i++ {
		bit := (bits >> (int(length) - 1 - i)) & 1
		if bit != 0 {
			out[i/8] |= 1 << (7 - uint(i&7))
		}
	}
	return out
}
