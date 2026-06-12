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

func TestCAVLCAllVLCEntriesWriteAndDecodeToTheirSymbols(t *testing.T) {
	assertCAVLCVLCWriteEntries(t, "chroma_dc_coeff_token", cavlcVLC{
		length: cavlcChromaDCCoeffTokenLen[:],
		bits:   cavlcChromaDCCoeffTokenBits[:],
		maxLen: chromaDCCoeffTokenVLCBits,
	})
	assertCAVLCVLCWriteEntries(t, "chroma422_dc_coeff_token", cavlcVLC{
		length: cavlcChroma422DCCoeffTokenLen[:],
		bits:   cavlcChroma422DCCoeffTokenBits[:],
		maxLen: chroma422DCCoeffTokenVLCBits,
	})
	for i := range cavlcCoeffTokenLen {
		assertCAVLCVLCWriteEntries(t, "coeff_token", cavlcVLC{
			length: cavlcCoeffTokenLen[i][:],
			bits:   cavlcCoeffTokenBits[i][:],
			maxLen: coeffTokenVLCBits,
		})
	}
	for i := 0; i < 15; i++ {
		assertCAVLCVLCWriteEntries(t, "total_zeros", cavlcVLC{
			length: cavlcTotalZerosLen[i][:],
			bits:   cavlcTotalZerosBits[i][:],
			maxLen: totalZerosVLCBits,
		})
	}
	for i := range cavlcChromaDCTotalZerosLen {
		assertCAVLCVLCWriteEntries(t, "chroma_dc_total_zeros", cavlcVLC{
			length: cavlcChromaDCTotalZerosLen[i][:],
			bits:   cavlcChromaDCTotalZerosBits[i][:],
			maxLen: chromaDCTotalZerosVLCBits,
		})
	}
	for i := range cavlcChroma422DCTotalZerosLen {
		assertCAVLCVLCWriteEntries(t, "chroma422_dc_total_zeros", cavlcVLC{
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
		assertCAVLCVLCWriteEntries(t, "run", cavlcVLC{
			length: cavlcRunLen[i][:],
			bits:   cavlcRunBits[i][:],
			maxLen: maxLen,
		})
	}
}

func assertCAVLCVLCWriteEntries(t *testing.T, name string, vlc cavlcVLC) {
	t.Helper()
	for symbol, length := range vlc.length {
		if length == 0 {
			continue
		}
		var bw BitWriter
		if err := vlc.write(&bw, symbol); err != nil {
			t.Fatalf("%s symbol %d write failed: %v", name, symbol, err)
		}
		gb := newBitReader(bw.Bytes())
		got, err := vlc.read(&gb)
		if err != nil {
			t.Fatalf("%s symbol %d read written code failed: %v", name, symbol, err)
		}
		if got != symbol {
			t.Fatalf("%s written code symbol = %d, want %d", name, got, symbol)
		}
		if gb.bitPos != uint32(length) {
			t.Fatalf("%s symbol %d consumed %d bits, want %d", name, symbol, gb.bitPos, length)
		}
	}
}

func TestCAVLCWriteResidualSyntaxPrimitivesRoundTrip(t *testing.T) {
	for _, tt := range []struct {
		name         string
		maxCoeff     int
		nC           int
		totalCoeff   int
		trailingOnes int
		totalZeros   int
		runBefore    int
		zerosLeft    int
	}{
		{name: "luma coeff-token total-zeros run", maxCoeff: 16, nC: 0, totalCoeff: 2, trailingOnes: 1, totalZeros: 3, runBefore: 2, zerosLeft: 3},
		{name: "chroma-dc coeff-token total-zeros", maxCoeff: 4, totalCoeff: 1, trailingOnes: 0, totalZeros: 2, runBefore: 1, zerosLeft: 2},
		{name: "chroma422-dc coeff-token total-zeros", maxCoeff: 8, totalCoeff: 3, trailingOnes: 2, totalZeros: 4, runBefore: 0, zerosLeft: 4},
		{name: "full block omits total-zeros bits", maxCoeff: 16, nC: 8, totalCoeff: 16, trailingOnes: 3, totalZeros: 0, runBefore: 0, zerosLeft: 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var bw BitWriter
			if err := writeCAVLCCoeffToken(&bw, tt.totalCoeff, tt.trailingOnes, tt.nC, tt.maxCoeff); err != nil {
				t.Fatalf("write coeff token: %v", err)
			}
			if err := writeCAVLCTotalZeros(&bw, tt.totalZeros, tt.totalCoeff, tt.maxCoeff); err != nil {
				t.Fatalf("write total zeros: %v", err)
			}
			if tt.totalCoeff < tt.maxCoeff {
				if err := writeCAVLCRunBefore(&bw, tt.runBefore, tt.zerosLeft); err != nil {
					t.Fatalf("write run before: %v", err)
				}
			}

			gb := newBitReader(bw.Bytes())
			coeffToken, err := readCAVLCCoeffToken(&gb, tt.nC, tt.maxCoeff)
			if err != nil {
				t.Fatalf("read coeff token: %v", err)
			}
			if got := coeffToken >> 2; got != tt.totalCoeff {
				t.Fatalf("totalCoeff = %d, want %d", got, tt.totalCoeff)
			}
			if got := coeffToken & 3; got != tt.trailingOnes {
				t.Fatalf("trailingOnes = %d, want %d", got, tt.trailingOnes)
			}
			totalZeros, err := readCAVLCTotalZeros(&gb, tt.totalCoeff, tt.maxCoeff)
			if err != nil {
				t.Fatalf("read total zeros: %v", err)
			}
			if totalZeros != tt.totalZeros {
				t.Fatalf("totalZeros = %d, want %d", totalZeros, tt.totalZeros)
			}
			if tt.totalCoeff < tt.maxCoeff {
				runBefore, err := readCAVLCRunBefore(&gb, tt.zerosLeft)
				if err != nil {
					t.Fatalf("read run before: %v", err)
				}
				if runBefore != tt.runBefore {
					t.Fatalf("runBefore = %d, want %d", runBefore, tt.runBefore)
				}
			}
		})
	}
}

func TestCAVLCWriteResidualSyntaxPrimitivesRejectInvalid(t *testing.T) {
	var bw BitWriter
	for _, tt := range []struct {
		name string
		err  error
	}{
		{name: "bad coeff token count", err: writeCAVLCCoeffToken(&bw, 17, 0, 0, 16)},
		{name: "bad trailing ones", err: writeCAVLCCoeffToken(&bw, 1, 2, 0, 16)},
		{name: "bad nC", err: writeCAVLCCoeffToken(&bw, 1, 0, 17, 16)},
		{name: "bad maxCoeff", err: writeCAVLCCoeffToken(&bw, 0, 0, 0, 0)},
		{name: "bad total zeros", err: writeCAVLCTotalZeros(&bw, 15, 2, 16)},
		{name: "bad full-block total zeros", err: writeCAVLCTotalZeros(&bw, 1, 16, 16)},
		{name: "bad run before", err: writeCAVLCRunBefore(&bw, 3, 2)},
		{name: "bad zeros left", err: writeCAVLCRunBefore(&bw, 0, 0)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err != ErrInvalidData {
				t.Fatalf("error = %v, want ErrInvalidData", tt.err)
			}
		})
	}
}

func TestCAVLCWriteResidualTrailingOnesRoundTripsThroughDecoder(t *testing.T) {
	scan := cavlcIdentityScan()
	for _, tt := range []struct {
		name         string
		block        [16]int32
		predictedNnz int
		wantCoeff    int
	}{
		{name: "empty"},
		{
			name:      "single positive",
			block:     [16]int32{0: 1},
			wantCoeff: 1,
		},
		{
			name:         "sparse signs",
			block:        [16]int32{0: -1, 2: 1},
			predictedNnz: 2,
			wantCoeff:    2,
		},
		{
			name:         "three trailing ones",
			block:        [16]int32{1: 1, 2: -1, 5: 1},
			predictedNnz: 4,
			wantCoeff:    3,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var bw BitWriter
			totalCoeff, err := writeCAVLCResidualTrailingOnes(&bw, tt.block[:], 0, scan[:], 16, tt.predictedNnz)
			if err != nil {
				t.Fatalf("write residual: %v", err)
			}
			if totalCoeff != tt.wantCoeff {
				t.Fatalf("totalCoeff = %d, want %d", totalCoeff, tt.wantCoeff)
			}

			gb := newBitReader(bw.Bytes())
			var got [16]int32
			decodedCoeff, err := decodeCAVLCResidual(&gb, got[:], 0, scan[:], nil, 16, tt.predictedNnz)
			if err != nil {
				t.Fatalf("decode written residual: %v", err)
			}
			if decodedCoeff != tt.wantCoeff {
				t.Fatalf("decoded totalCoeff = %d, want %d", decodedCoeff, tt.wantCoeff)
			}
			if got != tt.block {
				t.Fatalf("decoded block = %v, want %v", got, tt.block)
			}
		})
	}
}

func TestCAVLCWriteResidualTrailingOnesRejectsUnsupportedBlocks(t *testing.T) {
	scan := cavlcIdentityScan()
	for _, tt := range []struct {
		name  string
		block [16]int32
	}{
		{name: "level larger than one", block: [16]int32{0: 2}},
		{name: "negative level smaller than minus one", block: [16]int32{0: -2}},
		{name: "too many coefficients", block: [16]int32{0: 1, 1: -1, 2: 1, 3: -1}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var bw BitWriter
			if _, err := writeCAVLCResidualTrailingOnes(&bw, tt.block[:], 0, scan[:], 16, 0); err != ErrInvalidData {
				t.Fatalf("write residual error = %v, want ErrInvalidData", err)
			}
		})
	}

	var bw BitWriter
	block := [16]int32{0: 1}
	if _, err := writeCAVLCResidualTrailingOnes(nil, block[:], 0, scan[:], 16, 0); err != ErrInvalidData {
		t.Fatalf("nil writer error = %v, want ErrInvalidData", err)
	}
	if _, err := writeCAVLCResidualTrailingOnes(&bw, block[:], 0, scan[:4], 16, 0); err != ErrInvalidData {
		t.Fatalf("short scan error = %v, want ErrInvalidData", err)
	}
	if _, err := writeCAVLCResidualTrailingOnes(&bw, block[:], 0, scan[:], 17, 0); err != ErrInvalidData {
		t.Fatalf("bad maxCoeff error = %v, want ErrInvalidData", err)
	}
	if _, err := writeCAVLCResidualTrailingOnes(&bw, block[:], 1, scan[:], 16, 0); err != ErrInvalidData {
		t.Fatalf("block offset overflow error = %v, want ErrInvalidData", err)
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
