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

func TestCAVLCWriteResidualSingleLevelRoundTripsThroughDecoder(t *testing.T) {
	scan := cavlcIdentityScan()
	for _, tt := range []struct {
		name         string
		block        [16]int32
		predictedNnz int
	}{
		{name: "empty"},
		{name: "positive level two", block: [16]int32{0: 2}},
		{name: "negative level two", block: [16]int32{0: -2}},
		{name: "sparse positive level", block: [16]int32{4: 4}, predictedNnz: 2},
		{name: "sparse negative level", block: [16]int32{7: -4}, predictedNnz: 4},
		{name: "prefix fourteen positive level", block: [16]int32{0: 9}},
		{name: "prefix fourteen negative level", block: [16]int32{0: -9}},
		{name: "prefix fifteen positive level", block: [16]int32{0: 17}, predictedNnz: 3},
		{name: "prefix fifteen negative level", block: [16]int32{0: -17}, predictedNnz: 7},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var bw BitWriter
			totalCoeff, err := writeCAVLCResidualSingleLevel(&bw, tt.block[:], 0, scan[:], 16, tt.predictedNnz)
			if err != nil {
				t.Fatalf("write residual: %v", err)
			}
			wantCoeff := 1
			if tt.block == ([16]int32{}) {
				wantCoeff = 0
			}
			if totalCoeff != wantCoeff {
				t.Fatalf("totalCoeff = %d, want %d", totalCoeff, wantCoeff)
			}

			gb := newBitReader(bw.Bytes())
			var got [16]int32
			decodedCoeff, err := decodeCAVLCResidual(&gb, got[:], 0, scan[:], nil, 16, tt.predictedNnz)
			if err != nil {
				t.Fatalf("decode written residual: %v", err)
			}
			if decodedCoeff != wantCoeff {
				t.Fatalf("decoded totalCoeff = %d, want %d", decodedCoeff, wantCoeff)
			}
			if got != tt.block {
				t.Fatalf("decoded block = %v, want %v", got, tt.block)
			}
		})
	}
}

func TestCAVLCWriteResidualSingleLevelRejectsUnsupportedBlocks(t *testing.T) {
	scan := cavlcIdentityScan()
	for _, tt := range []struct {
		name  string
		block [16]int32
	}{
		{name: "trailing-one positive belongs to trailing-ones writer", block: [16]int32{0: 1}},
		{name: "trailing-one negative belongs to trailing-ones writer", block: [16]int32{0: -1}},
		{name: "multiple nonzero coefficients", block: [16]int32{0: 2, 1: -2}},
		{name: "mixed trailing one and level", block: [16]int32{0: 2, 2: 1}},
		{name: "positive level beyond bounded prefix fifteen", block: [16]int32{0: 3000}},
		{name: "negative level beyond bounded prefix fifteen", block: [16]int32{0: -3000}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var bw BitWriter
			if _, err := writeCAVLCResidualSingleLevel(&bw, tt.block[:], 0, scan[:], 16, 0); err != ErrInvalidData {
				t.Fatalf("write residual error = %v, want ErrInvalidData", err)
			}
		})
	}

	var bw BitWriter
	block := [16]int32{0: 2}
	if _, err := writeCAVLCResidualSingleLevel(nil, block[:], 0, scan[:], 16, 0); err != ErrInvalidData {
		t.Fatalf("nil writer error = %v, want ErrInvalidData", err)
	}
	if _, err := writeCAVLCResidualSingleLevel(&bw, block[:], 0, scan[:4], 16, 0); err != ErrInvalidData {
		t.Fatalf("short scan error = %v, want ErrInvalidData", err)
	}
	if _, err := writeCAVLCResidualSingleLevel(&bw, block[:], 0, scan[:], 17, 0); err != ErrInvalidData {
		t.Fatalf("bad maxCoeff error = %v, want ErrInvalidData", err)
	}
	if _, err := writeCAVLCResidualSingleLevel(&bw, block[:], 1, scan[:], 16, 0); err != ErrInvalidData {
		t.Fatalf("block offset overflow error = %v, want ErrInvalidData", err)
	}
}

func TestCAVLCWriteResidualTwoLevelsRoundTripsThroughDecoder(t *testing.T) {
	scan := cavlcIdentityScan()
	for _, tt := range []struct {
		name         string
		block        [16]int32
		predictedNnz int
	}{
		{name: "adjacent positive levels", block: [16]int32{0: 2, 1: 3}},
		{name: "adjacent negative levels", block: [16]int32{0: -2, 1: -3}},
		{name: "sparse mixed levels", block: [16]int32{0: -3, 4: 4}, predictedNnz: 2},
		{name: "first decoded level raises suffix length", block: [16]int32{1: 5, 3: -9}, predictedNnz: 4},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var bw BitWriter
			totalCoeff, err := writeCAVLCResidualTwoLevels(&bw, tt.block[:], 0, scan[:], 16, tt.predictedNnz)
			if err != nil {
				t.Fatalf("write residual: %v", err)
			}
			if totalCoeff != 2 {
				t.Fatalf("totalCoeff = %d, want 2", totalCoeff)
			}

			gb := newBitReader(bw.Bytes())
			var got [16]int32
			decodedCoeff, err := decodeCAVLCResidual(&gb, got[:], 0, scan[:], nil, 16, tt.predictedNnz)
			if err != nil {
				t.Fatalf("decode written residual: %v", err)
			}
			if decodedCoeff != 2 {
				t.Fatalf("decoded totalCoeff = %d, want 2", decodedCoeff)
			}
			if got != tt.block {
				t.Fatalf("decoded block = %v, want %v", got, tt.block)
			}
		})
	}
}

func TestCAVLCWriteResidualTwoLevelsRejectsUnsupportedBlocks(t *testing.T) {
	scan := cavlcIdentityScan()
	for _, tt := range []struct {
		name  string
		block [16]int32
	}{
		{name: "empty", block: [16]int32{}},
		{name: "only single level", block: [16]int32{0: 2}},
		{name: "trailing-one positive", block: [16]int32{0: 2, 1: 1}},
		{name: "trailing-one negative", block: [16]int32{0: -1, 1: -2}},
		{name: "three levels", block: [16]int32{0: 2, 1: -3, 2: 4}},
		{name: "subsequent level beyond bounded prefix", block: [16]int32{0: 3000, 1: 2}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var bw BitWriter
			if _, err := writeCAVLCResidualTwoLevels(&bw, tt.block[:], 0, scan[:], 16, 0); err != ErrInvalidData {
				t.Fatalf("write residual error = %v, want ErrInvalidData", err)
			}
		})
	}

	var bw BitWriter
	block := [16]int32{0: 2, 1: 3}
	if _, err := writeCAVLCResidualTwoLevels(nil, block[:], 0, scan[:], 16, 0); err != ErrInvalidData {
		t.Fatalf("nil writer error = %v, want ErrInvalidData", err)
	}
	if _, err := writeCAVLCResidualTwoLevels(&bw, block[:], 0, scan[:4], 16, 0); err != ErrInvalidData {
		t.Fatalf("short scan error = %v, want ErrInvalidData", err)
	}
	if _, err := writeCAVLCResidualTwoLevels(&bw, block[:], 0, scan[:], 17, 0); err != ErrInvalidData {
		t.Fatalf("bad maxCoeff error = %v, want ErrInvalidData", err)
	}
	if _, err := writeCAVLCResidualTwoLevels(&bw, block[:], 1, scan[:], 16, 0); err != ErrInvalidData {
		t.Fatalf("block offset overflow error = %v, want ErrInvalidData", err)
	}
}

func TestCAVLCWriteResidualSingleLevelTrailingOnesRoundTripsThroughDecoder(t *testing.T) {
	scan := cavlcIdentityScan()
	for _, tt := range []struct {
		name         string
		block        [16]int32
		predictedNnz int
		wantCoeff    int
	}{
		{
			name:      "adjacent positive trailing one",
			block:     [16]int32{0: 2, 1: 1},
			wantCoeff: 2,
		},
		{
			name:         "sparse negative level positive trailing one",
			block:        [16]int32{0: -2, 3: 1},
			predictedNnz: 2,
			wantCoeff:    2,
		},
		{
			name:         "two sparse trailing signs",
			block:        [16]int32{0: 3, 2: -1, 5: 1},
			predictedNnz: 4,
			wantCoeff:    3,
		},
		{
			name:         "negative first level and two trailing signs",
			block:        [16]int32{1: -4, 4: 1, 7: -1},
			predictedNnz: 8,
			wantCoeff:    3,
		},
		{
			name:      "prefix fourteen first level",
			block:     [16]int32{0: 9, 1: 1},
			wantCoeff: 2,
		},
		{
			name:         "prefix fifteen first level",
			block:        [16]int32{0: -17, 3: -1},
			predictedNnz: 2,
			wantCoeff:    2,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var bw BitWriter
			totalCoeff, err := writeCAVLCResidualSingleLevelTrailingOnes(&bw, tt.block[:], 0, scan[:], 16, tt.predictedNnz)
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

func TestCAVLCWriteResidualSingleLevelTrailingOnesRejectsUnsupportedBlocks(t *testing.T) {
	scan := cavlcIdentityScan()
	for _, tt := range []struct {
		name  string
		block [16]int32
	}{
		{name: "empty", block: [16]int32{}},
		{name: "only trailing ones", block: [16]int32{0: 1, 1: -1}},
		{name: "only single level", block: [16]int32{0: 2}},
		{name: "trailing one before first level", block: [16]int32{0: 1, 2: 2}},
		{name: "two non-trailing levels", block: [16]int32{0: 2, 1: 3, 2: 1}},
		{name: "too many coefficients", block: [16]int32{0: 2, 1: 1, 2: -1, 3: 1}},
		{name: "first level beyond bounded prefix fifteen", block: [16]int32{0: 3000, 1: 1}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var bw BitWriter
			if _, err := writeCAVLCResidualSingleLevelTrailingOnes(&bw, tt.block[:], 0, scan[:], 16, 0); err != ErrInvalidData {
				t.Fatalf("write residual error = %v, want ErrInvalidData", err)
			}
		})
	}

	var bw BitWriter
	block := [16]int32{0: 2, 1: 1}
	if _, err := writeCAVLCResidualSingleLevelTrailingOnes(nil, block[:], 0, scan[:], 16, 0); err != ErrInvalidData {
		t.Fatalf("nil writer error = %v, want ErrInvalidData", err)
	}
	if _, err := writeCAVLCResidualSingleLevelTrailingOnes(&bw, block[:], 0, scan[:4], 16, 0); err != ErrInvalidData {
		t.Fatalf("short scan error = %v, want ErrInvalidData", err)
	}
	if _, err := writeCAVLCResidualSingleLevelTrailingOnes(&bw, block[:], 0, scan[:], 17, 0); err != ErrInvalidData {
		t.Fatalf("bad maxCoeff error = %v, want ErrInvalidData", err)
	}
	if _, err := writeCAVLCResidualSingleLevelTrailingOnes(&bw, block[:], 1, scan[:], 16, 0); err != ErrInvalidData {
		t.Fatalf("block offset overflow error = %v, want ErrInvalidData", err)
	}
}

func TestCAVLCWriteResidualBoundedChromaContextsRoundTrip(t *testing.T) {
	scan := cavlcIdentityScan()
	for _, tt := range []struct {
		name      string
		maxCoeff  int
		block     [16]int32
		write     func(*BitWriter, []int32, int, []uint8, int, int) (int, error)
		wantCoeff int
	}{
		{
			name:      "chroma-dc trailing ones",
			maxCoeff:  4,
			block:     [16]int32{0: 1, 2: -1},
			write:     writeCAVLCResidualTrailingOnes,
			wantCoeff: 2,
		},
		{
			name:      "chroma422-dc trailing ones",
			maxCoeff:  8,
			block:     [16]int32{1: -1, 6: 1},
			write:     writeCAVLCResidualTrailingOnes,
			wantCoeff: 2,
		},
		{
			name:      "chroma-dc single level",
			maxCoeff:  4,
			block:     [16]int32{3: -3},
			write:     writeCAVLCResidualSingleLevel,
			wantCoeff: 1,
		},
		{
			name:      "chroma422-dc single level",
			maxCoeff:  8,
			block:     [16]int32{6: 4},
			write:     writeCAVLCResidualSingleLevel,
			wantCoeff: 1,
		},
		{
			name:      "chroma-dc single level trailing one",
			maxCoeff:  4,
			block:     [16]int32{0: 2, 3: -1},
			write:     writeCAVLCResidualSingleLevelTrailingOnes,
			wantCoeff: 2,
		},
		{
			name:      "chroma422-dc single level two trailing signs",
			maxCoeff:  8,
			block:     [16]int32{1: -2, 3: 1, 7: -1},
			write:     writeCAVLCResidualSingleLevelTrailingOnes,
			wantCoeff: 3,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var bw BitWriter
			totalCoeff, err := tt.write(&bw, tt.block[:], 0, scan[:], tt.maxCoeff, 0)
			if err != nil {
				t.Fatalf("write residual: %v", err)
			}
			if totalCoeff != tt.wantCoeff {
				t.Fatalf("totalCoeff = %d, want %d", totalCoeff, tt.wantCoeff)
			}

			gb := newBitReader(bw.Bytes())
			var got [16]int32
			decodedCoeff, err := decodeCAVLCResidual(&gb, got[:], 0, scan[:], nil, tt.maxCoeff, 0)
			if err != nil {
				t.Fatalf("decode written residual: %v", err)
			}
			if decodedCoeff != tt.wantCoeff {
				t.Fatalf("decoded totalCoeff = %d, want %d", decodedCoeff, tt.wantCoeff)
			}
			for i := 0; i < tt.maxCoeff; i++ {
				if got[i] != tt.block[i] {
					t.Fatalf("decoded block[%d] = %d, want %d; block=%v", i, got[i], tt.block[i], got)
				}
			}
		})
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
