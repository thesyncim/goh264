// SPDX-License-Identifier: LGPL-2.1-or-later
//
// Source-shaped port of the table-driven pieces of FFmpeg n8.0.1
// libavcodec/h264_cavlc.c used by H.264 CAVLC residual decoding.

package h264

const (
	levelTabBits = 8

	chromaDCCoeffTokenVLCBits    = 8
	chroma422DCCoeffTokenVLCBits = 13
	coeffTokenVLCBits            = 8
	totalZerosVLCBits            = 9
	chromaDCTotalZerosVLCBits    = 3
	chroma422DCTotalZerosVLCBits = 5
	runVLCBits                   = 3
	run7VLCBits                  = 6
)

const lumaDCBlockIndex = 48
const chromaDCBlockIndex = 49

var cavlcGolombToInterCBPGray = [16]uint8{
	0, 1, 2, 4, 8, 3, 5, 10, 12, 15, 7, 11, 13, 14, 6, 9,
}

var cavlcGolombToIntra4x4CBPGray = [16]uint8{
	15, 0, 7, 11, 13, 14, 3, 5, 10, 12, 1, 2, 4, 8, 6, 9,
}

var cavlcChromaDCCoeffTokenLen = [4 * 5]uint8{
	2, 0, 0, 0,
	6, 1, 0, 0,
	6, 6, 3, 0,
	6, 7, 7, 6,
	6, 8, 8, 7,
}

var cavlcChromaDCCoeffTokenBits = [4 * 5]uint8{
	1, 0, 0, 0,
	7, 1, 0, 0,
	4, 6, 1, 0,
	3, 3, 2, 5,
	2, 3, 2, 0,
}

var cavlcChroma422DCCoeffTokenLen = [4 * 9]uint8{
	1, 0, 0, 0,
	7, 2, 0, 0,
	7, 7, 3, 0,
	9, 7, 7, 5,
	9, 9, 7, 6,
	10, 10, 9, 7,
	11, 11, 10, 7,
	12, 12, 11, 10,
	13, 12, 12, 11,
}

var cavlcChroma422DCCoeffTokenBits = [4 * 9]uint8{
	1, 0, 0, 0,
	15, 1, 0, 0,
	14, 13, 1, 0,
	7, 12, 11, 1,
	6, 5, 10, 1,
	7, 6, 4, 9,
	7, 6, 5, 8,
	7, 6, 5, 4,
	7, 5, 4, 4,
}

var cavlcCoeffTokenLen = [4][4 * 17]uint8{
	{
		1, 0, 0, 0,
		6, 2, 0, 0, 8, 6, 3, 0, 9, 8, 7, 5, 10, 9, 8, 6,
		11, 10, 9, 7, 13, 11, 10, 8, 13, 13, 11, 9, 13, 13, 13, 10,
		14, 14, 13, 11, 14, 14, 14, 13, 15, 15, 14, 14, 15, 15, 15, 14,
		16, 15, 15, 15, 16, 16, 16, 15, 16, 16, 16, 16, 16, 16, 16, 16,
	},
	{
		2, 0, 0, 0,
		6, 2, 0, 0, 6, 5, 3, 0, 7, 6, 6, 4, 8, 6, 6, 4,
		8, 7, 7, 5, 9, 8, 8, 6, 11, 9, 9, 6, 11, 11, 11, 7,
		12, 11, 11, 9, 12, 12, 12, 11, 12, 12, 12, 11, 13, 13, 13, 12,
		13, 13, 13, 13, 13, 14, 13, 13, 14, 14, 14, 13, 14, 14, 14, 14,
	},
	{
		4, 0, 0, 0,
		6, 4, 0, 0, 6, 5, 4, 0, 6, 5, 5, 4, 7, 5, 5, 4,
		7, 5, 5, 4, 7, 6, 6, 4, 7, 6, 6, 4, 8, 7, 7, 5,
		8, 8, 7, 6, 9, 8, 8, 7, 9, 9, 8, 8, 9, 9, 9, 8,
		10, 9, 9, 9, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10,
	},
	{
		6, 0, 0, 0,
		6, 6, 0, 0, 6, 6, 6, 0, 6, 6, 6, 6, 6, 6, 6, 6,
		6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6,
		6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6,
		6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6,
	},
}

var cavlcCoeffTokenBits = [4][4 * 17]uint8{
	{
		1, 0, 0, 0,
		5, 1, 0, 0, 7, 4, 1, 0, 7, 6, 5, 3, 7, 6, 5, 3,
		7, 6, 5, 4, 15, 6, 5, 4, 11, 14, 5, 4, 8, 10, 13, 4,
		15, 14, 9, 4, 11, 10, 13, 12, 15, 14, 9, 12, 11, 10, 13, 8,
		15, 1, 9, 12, 11, 14, 13, 8, 7, 10, 9, 12, 4, 6, 5, 8,
	},
	{
		3, 0, 0, 0,
		11, 2, 0, 0, 7, 7, 3, 0, 7, 10, 9, 5, 7, 6, 5, 4,
		4, 6, 5, 6, 7, 6, 5, 8, 15, 6, 5, 4, 11, 14, 13, 4,
		15, 10, 9, 4, 11, 14, 13, 12, 8, 10, 9, 8, 15, 14, 13, 12,
		11, 10, 9, 12, 7, 11, 6, 8, 9, 8, 10, 1, 7, 6, 5, 4,
	},
	{
		15, 0, 0, 0,
		15, 14, 0, 0, 11, 15, 13, 0, 8, 12, 14, 12, 15, 10, 11, 11,
		11, 8, 9, 10, 9, 14, 13, 9, 8, 10, 9, 8, 15, 14, 13, 13,
		11, 14, 10, 12, 15, 10, 13, 12, 11, 14, 9, 12, 8, 10, 13, 8,
		13, 7, 9, 12, 9, 12, 11, 10, 5, 8, 7, 6, 1, 4, 3, 2,
	},
	{
		3, 0, 0, 0,
		0, 1, 0, 0, 4, 5, 6, 0, 8, 9, 10, 11, 12, 13, 14, 15,
		16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31,
		32, 33, 34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45, 46, 47,
		48, 49, 50, 51, 52, 53, 54, 55, 56, 57, 58, 59, 60, 61, 62, 63,
	},
}

var cavlcTotalZerosLen = [16][16]uint8{
	{1, 3, 3, 4, 4, 5, 5, 6, 6, 7, 7, 8, 8, 9, 9, 9},
	{3, 3, 3, 3, 3, 4, 4, 4, 4, 5, 5, 6, 6, 6, 6},
	{4, 3, 3, 3, 4, 4, 3, 3, 4, 5, 5, 6, 5, 6},
	{5, 3, 4, 4, 3, 3, 3, 4, 3, 4, 5, 5, 5},
	{4, 4, 4, 3, 3, 3, 3, 3, 4, 5, 4, 5},
	{6, 5, 3, 3, 3, 3, 3, 3, 4, 3, 6},
	{6, 5, 3, 3, 3, 2, 3, 4, 3, 6},
	{6, 4, 5, 3, 2, 2, 3, 3, 6},
	{6, 6, 4, 2, 2, 3, 2, 5},
	{5, 5, 3, 2, 2, 2, 4},
	{4, 4, 3, 3, 1, 3},
	{4, 4, 2, 1, 3},
	{3, 3, 1, 2},
	{2, 2, 1},
	{1, 1},
}

var cavlcTotalZerosBits = [16][16]uint8{
	{1, 3, 2, 3, 2, 3, 2, 3, 2, 3, 2, 3, 2, 3, 2, 1},
	{7, 6, 5, 4, 3, 5, 4, 3, 2, 3, 2, 3, 2, 1, 0},
	{5, 7, 6, 5, 4, 3, 4, 3, 2, 3, 2, 1, 1, 0},
	{3, 7, 5, 4, 6, 5, 4, 3, 3, 2, 2, 1, 0},
	{5, 4, 3, 7, 6, 5, 4, 3, 2, 1, 1, 0},
	{1, 1, 7, 6, 5, 4, 3, 2, 1, 1, 0},
	{1, 1, 5, 4, 3, 3, 2, 1, 1, 0},
	{1, 1, 1, 3, 3, 2, 2, 1, 0},
	{1, 0, 1, 3, 2, 1, 1, 1},
	{1, 0, 1, 3, 2, 1, 1},
	{0, 1, 1, 2, 1, 3},
	{0, 1, 1, 1, 1},
	{0, 1, 1, 1},
	{0, 1, 1},
	{0, 1},
}

var cavlcChromaDCTotalZerosLen = [3][4]uint8{
	{1, 2, 3, 3},
	{1, 2, 2, 0},
	{1, 1, 0, 0},
}

var cavlcChromaDCTotalZerosBits = [3][4]uint8{
	{1, 1, 1, 0},
	{1, 1, 0, 0},
	{1, 0, 0, 0},
}

var cavlcChroma422DCTotalZerosLen = [7][8]uint8{
	{1, 3, 3, 4, 4, 4, 5, 5},
	{3, 2, 3, 3, 3, 3, 3},
	{3, 3, 2, 2, 3, 3},
	{3, 2, 2, 2, 3},
	{2, 2, 2, 2},
	{2, 2, 1},
	{1, 1},
}

var cavlcChroma422DCTotalZerosBits = [7][8]uint8{
	{1, 2, 3, 2, 3, 1, 1, 0},
	{0, 1, 1, 4, 5, 6, 7},
	{0, 1, 1, 2, 6, 7},
	{6, 0, 1, 2, 7},
	{0, 1, 2, 3},
	{0, 1, 1},
	{0, 1},
}

var cavlcRunLen = [7][16]uint8{
	{1, 1},
	{1, 2, 2},
	{2, 2, 2, 2},
	{2, 2, 2, 3, 3},
	{2, 2, 3, 3, 3, 3},
	{2, 3, 3, 3, 3, 3, 3},
	{3, 3, 3, 3, 3, 3, 3, 4, 5, 6, 7, 8, 9, 10, 11},
}

var cavlcRunBits = [7][16]uint8{
	{1, 0},
	{1, 1, 0},
	{3, 2, 1, 0},
	{3, 2, 1, 1, 0},
	{3, 2, 3, 2, 1, 0},
	{3, 0, 1, 3, 2, 5, 4},
	{7, 6, 5, 4, 3, 2, 1, 1, 1, 1, 1, 1, 1, 1, 1},
}

var cavlcCoeffTokenTableIndex = [17]uint8{
	0, 0, 1, 1, 2, 2, 2, 2, 3, 3, 3, 3, 3, 3, 3, 3, 3,
}

var cavlcLevelTable = initCAVLCLevelTable()

type cavlcVLC struct {
	length []uint8
	bits   []uint8
	maxLen uint8
}

func (vlc cavlcVLC) write(bw *BitWriter, symbol int) error {
	if bw == nil || symbol < 0 || symbol >= len(vlc.length) {
		return ErrInvalidData
	}
	length := vlc.length[symbol]
	if length == 0 {
		return ErrInvalidData
	}
	return bw.WriteBits(uint32(vlc.bits[symbol]), uint32(length))
}

func (vlc cavlcVLC) read(gb *bitReader) (int, error) {
	var code uint32
	maxLen := vlc.maxLen
	for _, length := range vlc.length {
		if length > maxLen {
			maxLen = length
		}
	}
	for length := uint8(1); length <= maxLen; length++ {
		bit, err := gb.readBit()
		if err != nil {
			return 0, err
		}
		code = (code << 1) | bit
		for symbol, symbolLength := range vlc.length {
			if symbolLength == length && uint32(vlc.bits[symbol]) == code {
				return symbol, nil
			}
		}
	}
	return 0, ErrInvalidData
}

func initCAVLCLevelTable() [7][1 << levelTabBits][2]int8 {
	var table [7][1 << levelTabBits][2]int8
	for suffixLength := 0; suffixLength < 7; suffixLength++ {
		for i := uint32(0); i < 1<<levelTabBits; i++ {
			prefix := levelTabBits - avLog2(2*i)
			if prefix+1+suffixLength <= levelTabBits {
				levelCode := (prefix << suffixLength) +
					int(i>>(avLog2(i)-suffixLength)) - (1 << suffixLength)
				mask := -(levelCode & 1)
				levelCode = (((2 + levelCode) >> 1) ^ mask) - mask
				table[suffixLength][i][0] = int8(levelCode)
				table[suffixLength][i][1] = int8(prefix + 1 + suffixLength)
			} else if prefix+1 <= levelTabBits {
				table[suffixLength][i][0] = int8(prefix + 100)
				table[suffixLength][i][1] = int8(prefix + 1)
			} else {
				table[suffixLength][i][0] = levelTabBits + 100
				table[suffixLength][i][1] = levelTabBits
			}
		}
	}
	return table
}

func coeffTokenVLC(nC int) (cavlcVLC, error) {
	if nC < 0 || nC >= len(cavlcCoeffTokenTableIndex) {
		return cavlcVLC{}, ErrInvalidData
	}
	table := cavlcCoeffTokenTableIndex[nC]
	return cavlcVLC{
		length: cavlcCoeffTokenLen[table][:],
		bits:   cavlcCoeffTokenBits[table][:],
		maxLen: coeffTokenVLCBits,
	}, nil
}

func readCAVLCCoeffToken(gb *bitReader, nC int, maxCoeff int) (int, error) {
	if maxCoeff <= 0 || maxCoeff > 16 {
		return 0, ErrInvalidData
	}
	if maxCoeff <= 8 {
		if maxCoeff == 4 {
			return (cavlcVLC{
				length: cavlcChromaDCCoeffTokenLen[:],
				bits:   cavlcChromaDCCoeffTokenBits[:],
				maxLen: chromaDCCoeffTokenVLCBits,
			}).read(gb)
		}
		return (cavlcVLC{
			length: cavlcChroma422DCCoeffTokenLen[:],
			bits:   cavlcChroma422DCCoeffTokenBits[:],
			maxLen: chroma422DCCoeffTokenVLCBits,
		}).read(gb)
	}

	vlc, err := coeffTokenVLC(nC)
	if err != nil {
		return 0, err
	}
	return vlc.read(gb)
}

func writeCAVLCCoeffToken(bw *BitWriter, totalCoeff int, trailingOnes int, nC int, maxCoeff int) error {
	if maxCoeff <= 0 || maxCoeff > 16 ||
		totalCoeff < 0 || totalCoeff > maxCoeff ||
		trailingOnes < 0 || trailingOnes > 3 || trailingOnes > totalCoeff {
		return ErrInvalidData
	}
	symbol := (totalCoeff << 2) | trailingOnes
	if maxCoeff <= 8 {
		if maxCoeff == 4 {
			return (cavlcVLC{
				length: cavlcChromaDCCoeffTokenLen[:],
				bits:   cavlcChromaDCCoeffTokenBits[:],
				maxLen: chromaDCCoeffTokenVLCBits,
			}).write(bw, symbol)
		}
		return (cavlcVLC{
			length: cavlcChroma422DCCoeffTokenLen[:],
			bits:   cavlcChroma422DCCoeffTokenBits[:],
			maxLen: chroma422DCCoeffTokenVLCBits,
		}).write(bw, symbol)
	}

	vlc, err := coeffTokenVLC(nC)
	if err != nil {
		return err
	}
	return vlc.write(bw, symbol)
}

func readCAVLCTotalZeros(gb *bitReader, totalCoeff int, maxCoeff int) (int, error) {
	if totalCoeff <= 0 || totalCoeff > maxCoeff || maxCoeff > 16 {
		return 0, ErrInvalidData
	}
	if totalCoeff == maxCoeff {
		return 0, nil
	}

	if maxCoeff <= 8 {
		if maxCoeff == 4 {
			return (cavlcVLC{
				length: cavlcChromaDCTotalZerosLen[totalCoeff-1][:],
				bits:   cavlcChromaDCTotalZerosBits[totalCoeff-1][:],
				maxLen: chromaDCTotalZerosVLCBits,
			}).read(gb)
		}
		return (cavlcVLC{
			length: cavlcChroma422DCTotalZerosLen[totalCoeff-1][:],
			bits:   cavlcChroma422DCTotalZerosBits[totalCoeff-1][:],
			maxLen: chroma422DCTotalZerosVLCBits,
		}).read(gb)
	}

	return (cavlcVLC{
		length: cavlcTotalZerosLen[totalCoeff-1][:],
		bits:   cavlcTotalZerosBits[totalCoeff-1][:],
		maxLen: totalZerosVLCBits,
	}).read(gb)
}

func writeCAVLCTotalZeros(bw *BitWriter, totalZeros int, totalCoeff int, maxCoeff int) error {
	if totalCoeff <= 0 || totalCoeff > maxCoeff || maxCoeff > 16 ||
		totalZeros < 0 || totalZeros > maxCoeff-totalCoeff {
		return ErrInvalidData
	}
	if totalCoeff == maxCoeff {
		if totalZeros != 0 {
			return ErrInvalidData
		}
		return nil
	}

	if maxCoeff <= 8 {
		if maxCoeff == 4 {
			return (cavlcVLC{
				length: cavlcChromaDCTotalZerosLen[totalCoeff-1][:],
				bits:   cavlcChromaDCTotalZerosBits[totalCoeff-1][:],
				maxLen: chromaDCTotalZerosVLCBits,
			}).write(bw, totalZeros)
		}
		return (cavlcVLC{
			length: cavlcChroma422DCTotalZerosLen[totalCoeff-1][:],
			bits:   cavlcChroma422DCTotalZerosBits[totalCoeff-1][:],
			maxLen: chroma422DCTotalZerosVLCBits,
		}).write(bw, totalZeros)
	}

	return (cavlcVLC{
		length: cavlcTotalZerosLen[totalCoeff-1][:],
		bits:   cavlcTotalZerosBits[totalCoeff-1][:],
		maxLen: totalZerosVLCBits,
	}).write(bw, totalZeros)
}

func readCAVLCRunBefore(gb *bitReader, zerosLeft int) (int, error) {
	if zerosLeft <= 0 {
		return 0, ErrInvalidData
	}
	row := 6
	maxLen := uint8(run7VLCBits)
	if zerosLeft < 7 {
		row = zerosLeft - 1
		maxLen = runVLCBits
	}
	return (cavlcVLC{
		length: cavlcRunLen[row][:],
		bits:   cavlcRunBits[row][:],
		maxLen: maxLen,
	}).read(gb)
}

func writeCAVLCRunBefore(bw *BitWriter, runBefore int, zerosLeft int) error {
	if zerosLeft <= 0 || runBefore < 0 || runBefore > zerosLeft {
		return ErrInvalidData
	}
	row := 6
	maxLen := uint8(run7VLCBits)
	if zerosLeft < 7 {
		row = zerosLeft - 1
		maxLen = runVLCBits
	}
	return (cavlcVLC{
		length: cavlcRunLen[row][:],
		bits:   cavlcRunBits[row][:],
		maxLen: maxLen,
	}).write(bw, runBefore)
}

func writeCAVLCResidualTrailingOnes(bw *BitWriter, block []int32, n int, scantable []uint8, maxCoeff int, predictedNnz int) (int, error) {
	if bw == nil || maxCoeff <= 0 || maxCoeff > 16 || len(scantable) < maxCoeff {
		return 0, ErrInvalidData
	}
	var scanIndex [3]int
	var level [3]int32
	totalCoeff := 0
	for i := 0; i < maxCoeff; i++ {
		pos := int(scantable[i])
		if pos < 0 || n+pos < 0 || n+pos >= len(block) {
			return 0, ErrInvalidData
		}
		v := block[n+pos]
		if v == 0 {
			continue
		}
		if v != 1 && v != -1 {
			return 0, ErrInvalidData
		}
		if totalCoeff == len(scanIndex) {
			return 0, ErrInvalidData
		}
		scanIndex[totalCoeff] = i
		level[totalCoeff] = v
		totalCoeff++
	}

	if err := writeCAVLCCoeffToken(bw, totalCoeff, totalCoeff, predictedNnz, maxCoeff); err != nil {
		return 0, err
	}
	if totalCoeff == 0 {
		return 0, nil
	}

	for i := totalCoeff - 1; i >= 0; i-- {
		if level[i] < 0 {
			bw.WriteBit(1)
		} else {
			bw.WriteBit(0)
		}
	}

	totalZeros := scanIndex[totalCoeff-1] + 1 - totalCoeff
	if err := writeCAVLCTotalZeros(bw, totalZeros, totalCoeff, maxCoeff); err != nil {
		return 0, err
	}
	zerosLeft := totalZeros
	for i := totalCoeff - 2; i >= 0 && zerosLeft > 0; i-- {
		runBefore := scanIndex[i+1] - scanIndex[i] - 1
		if err := writeCAVLCRunBefore(bw, runBefore, zerosLeft); err != nil {
			return 0, err
		}
		zerosLeft -= runBefore
	}
	return totalCoeff, nil
}

func writeCAVLCResidualSingleLevel(bw *BitWriter, block []int32, n int, scantable []uint8, maxCoeff int, predictedNnz int) (int, error) {
	if bw == nil || maxCoeff <= 0 || maxCoeff > 16 || len(scantable) < maxCoeff {
		return 0, ErrInvalidData
	}
	coeffIndex := -1
	level := int32(0)
	for i := 0; i < maxCoeff; i++ {
		pos := int(scantable[i])
		if pos < 0 || n+pos < 0 || n+pos >= len(block) {
			return 0, ErrInvalidData
		}
		v := block[n+pos]
		if v == 0 {
			continue
		}
		if v == 1 || v == -1 || coeffIndex >= 0 {
			return 0, ErrInvalidData
		}
		coeffIndex = i
		level = v
	}
	if coeffIndex < 0 {
		return writeCAVLCResidualTrailingOnes(bw, block, n, scantable, maxCoeff, predictedNnz)
	}

	if err := writeCAVLCCoeffToken(bw, 1, 0, predictedNnz, maxCoeff); err != nil {
		return 0, err
	}
	if err := writeCAVLCFirstLevel(bw, level); err != nil {
		return 0, err
	}
	totalZeros := coeffIndex
	if err := writeCAVLCTotalZeros(bw, totalZeros, 1, maxCoeff); err != nil {
		return 0, err
	}
	return 1, nil
}

func writeCAVLCResidualTwoLevels(bw *BitWriter, block []int32, n int, scantable []uint8, maxCoeff int, predictedNnz int) (int, error) {
	return writeCAVLCResidualNonTrailingLevels(bw, block, n, scantable, maxCoeff, predictedNnz, 2, 2)
}

func writeCAVLCResidualNonTrailingLevels(bw *BitWriter, block []int32, n int, scantable []uint8, maxCoeff int, predictedNnz int, minCoeff int, maxAdmittedCoeff int) (int, error) {
	if bw == nil || maxCoeff <= 0 || maxCoeff > 16 || len(scantable) < maxCoeff {
		return 0, ErrInvalidData
	}
	if minCoeff <= 0 || maxAdmittedCoeff < minCoeff || maxAdmittedCoeff > 16 {
		return 0, ErrInvalidData
	}
	var scanIndex [16]int
	var level [16]int32
	totalCoeff := 0
	for i := 0; i < maxCoeff; i++ {
		pos := int(scantable[i])
		if pos < 0 || n+pos < 0 || n+pos >= len(block) {
			return 0, ErrInvalidData
		}
		v := block[n+pos]
		if v == 0 {
			continue
		}
		if v == 1 || v == -1 || totalCoeff == maxAdmittedCoeff {
			return 0, ErrInvalidData
		}
		scanIndex[totalCoeff] = i
		level[totalCoeff] = v
		totalCoeff++
	}
	if totalCoeff < minCoeff || totalCoeff > maxAdmittedCoeff {
		return 0, ErrInvalidData
	}

	if err := writeCAVLCCoeffToken(bw, totalCoeff, 0, predictedNnz, maxCoeff); err != nil {
		return 0, err
	}
	firstLevel := level[totalCoeff-1]
	initialSuffixLength := 0
	if totalCoeff > 10 {
		initialSuffixLength = 1
	}
	suffixLength, err := writeCAVLCFirstLevelWithSuffix(bw, firstLevel, initialSuffixLength)
	if err != nil {
		return 0, err
	}
	for i := totalCoeff - 2; i >= 0; i-- {
		if err := writeCAVLCSubsequentLevel(bw, level[i], suffixLength); err != nil {
			return 0, err
		}
		suffixLength = cavlcSuffixLengthAfterSubsequentLevel(level[i], suffixLength)
	}

	totalZeros := scanIndex[totalCoeff-1] + 1 - totalCoeff
	if err := writeCAVLCTotalZeros(bw, totalZeros, totalCoeff, maxCoeff); err != nil {
		return 0, err
	}
	zerosLeft := totalZeros
	for i := totalCoeff - 2; i >= 0 && zerosLeft > 0; i-- {
		runBefore := scanIndex[i+1] - scanIndex[i] - 1
		if err := writeCAVLCRunBefore(bw, runBefore, zerosLeft); err != nil {
			return 0, err
		}
		zerosLeft -= runBefore
	}
	return totalCoeff, nil
}

func writeCAVLCResidualThreeNonTrailingLevels(bw *BitWriter, block []int32, n int, scantable []uint8, maxCoeff int, predictedNnz int) (int, error) {
	return writeCAVLCResidualNonTrailingLevels(bw, block, n, scantable, maxCoeff, predictedNnz, 3, 3)
}

func writeCAVLCResidualFourNonTrailingLevels(bw *BitWriter, block []int32, n int, scantable []uint8, maxCoeff int, predictedNnz int) (int, error) {
	return writeCAVLCResidualNonTrailingLevels(bw, block, n, scantable, maxCoeff, predictedNnz, 4, 4)
}

func writeCAVLCResidualFiveNonTrailingLevels(bw *BitWriter, block []int32, n int, scantable []uint8, maxCoeff int, predictedNnz int) (int, error) {
	return writeCAVLCResidualNonTrailingLevels(bw, block, n, scantable, maxCoeff, predictedNnz, 5, 5)
}

func writeCAVLCResidualSixNonTrailingLevels(bw *BitWriter, block []int32, n int, scantable []uint8, maxCoeff int, predictedNnz int) (int, error) {
	return writeCAVLCResidualNonTrailingLevels(bw, block, n, scantable, maxCoeff, predictedNnz, 6, 6)
}

func writeCAVLCResidualSevenNonTrailingLevels(bw *BitWriter, block []int32, n int, scantable []uint8, maxCoeff int, predictedNnz int) (int, error) {
	return writeCAVLCResidualNonTrailingLevels(bw, block, n, scantable, maxCoeff, predictedNnz, 7, 7)
}

func writeCAVLCResidualEightNonTrailingLevels(bw *BitWriter, block []int32, n int, scantable []uint8, maxCoeff int, predictedNnz int) (int, error) {
	return writeCAVLCResidualNonTrailingLevels(bw, block, n, scantable, maxCoeff, predictedNnz, 8, 8)
}

func writeCAVLCResidualNineNonTrailingLevels(bw *BitWriter, block []int32, n int, scantable []uint8, maxCoeff int, predictedNnz int) (int, error) {
	return writeCAVLCResidualNonTrailingLevels(bw, block, n, scantable, maxCoeff, predictedNnz, 9, 9)
}

func writeCAVLCResidualTenNonTrailingLevels(bw *BitWriter, block []int32, n int, scantable []uint8, maxCoeff int, predictedNnz int) (int, error) {
	return writeCAVLCResidualNonTrailingLevels(bw, block, n, scantable, maxCoeff, predictedNnz, 10, 10)
}

func writeCAVLCResidualElevenNonTrailingLevels(bw *BitWriter, block []int32, n int, scantable []uint8, maxCoeff int, predictedNnz int) (int, error) {
	return writeCAVLCResidualNonTrailingLevels(bw, block, n, scantable, maxCoeff, predictedNnz, 11, 11)
}

func writeCAVLCResidualTwelveNonTrailingLevels(bw *BitWriter, block []int32, n int, scantable []uint8, maxCoeff int, predictedNnz int) (int, error) {
	return writeCAVLCResidualNonTrailingLevels(bw, block, n, scantable, maxCoeff, predictedNnz, 12, 12)
}

func writeCAVLCResidualThirteenNonTrailingLevels(bw *BitWriter, block []int32, n int, scantable []uint8, maxCoeff int, predictedNnz int) (int, error) {
	return writeCAVLCResidualNonTrailingLevels(bw, block, n, scantable, maxCoeff, predictedNnz, 13, 13)
}

func writeCAVLCResidualFourteenNonTrailingLevels(bw *BitWriter, block []int32, n int, scantable []uint8, maxCoeff int, predictedNnz int) (int, error) {
	return writeCAVLCResidualNonTrailingLevels(bw, block, n, scantable, maxCoeff, predictedNnz, 14, 14)
}

func writeCAVLCResidualFifteenNonTrailingLevels(bw *BitWriter, block []int32, n int, scantable []uint8, maxCoeff int, predictedNnz int) (int, error) {
	return writeCAVLCResidualNonTrailingLevels(bw, block, n, scantable, maxCoeff, predictedNnz, 15, 15)
}

func writeCAVLCResidualSingleLevelTrailingOnes(bw *BitWriter, block []int32, n int, scantable []uint8, maxCoeff int, predictedNnz int) (int, error) {
	if bw == nil || maxCoeff <= 0 || maxCoeff > 16 || len(scantable) < maxCoeff {
		return 0, ErrInvalidData
	}
	var scanIndex [3]int
	var level [3]int32
	totalCoeff := 0
	for i := 0; i < maxCoeff; i++ {
		pos := int(scantable[i])
		if pos < 0 || n+pos < 0 || n+pos >= len(block) {
			return 0, ErrInvalidData
		}
		v := block[n+pos]
		if v == 0 {
			continue
		}
		if totalCoeff == len(scanIndex) {
			return 0, ErrInvalidData
		}
		scanIndex[totalCoeff] = i
		level[totalCoeff] = v
		totalCoeff++
	}

	trailingOnes := 0
	for i := totalCoeff - 1; i >= 0 && trailingOnes < 3; i-- {
		if level[i] != 1 && level[i] != -1 {
			break
		}
		trailingOnes++
	}
	if totalCoeff == 0 || trailingOnes == 0 || trailingOnes == totalCoeff || totalCoeff-trailingOnes != 1 {
		return 0, ErrInvalidData
	}
	firstLevel := level[totalCoeff-trailingOnes-1]
	if firstLevel == 1 || firstLevel == -1 {
		return 0, ErrInvalidData
	}

	if err := writeCAVLCCoeffToken(bw, totalCoeff, trailingOnes, predictedNnz, maxCoeff); err != nil {
		return 0, err
	}
	for i := totalCoeff - 1; i >= totalCoeff-trailingOnes; i-- {
		if level[i] < 0 {
			bw.WriteBit(1)
		} else {
			bw.WriteBit(0)
		}
	}
	if err := writeCAVLCFirstLevel(bw, firstLevel); err != nil {
		return 0, err
	}
	totalZeros := scanIndex[totalCoeff-1] + 1 - totalCoeff
	if err := writeCAVLCTotalZeros(bw, totalZeros, totalCoeff, maxCoeff); err != nil {
		return 0, err
	}
	zerosLeft := totalZeros
	for i := totalCoeff - 2; i >= 0 && zerosLeft > 0; i-- {
		runBefore := scanIndex[i+1] - scanIndex[i] - 1
		if err := writeCAVLCRunBefore(bw, runBefore, zerosLeft); err != nil {
			return 0, err
		}
		zerosLeft -= runBefore
	}
	return totalCoeff, nil
}

func writeCAVLCFirstLevel(bw *BitWriter, level int32) error {
	if bw == nil || level == 0 || level == 1 || level == -1 {
		return ErrInvalidData
	}
	_, err := writeCAVLCFirstLevelWithSuffix(bw, level, 0)
	return err
}

func writeCAVLCFirstLevelWithSuffix(bw *BitWriter, level int32, suffixLength int) (int, error) {
	if bw == nil || level == 0 || level == 1 || level == -1 || suffixLength < 0 || suffixLength > 1 {
		return 0, ErrInvalidData
	}
	code, err := cavlcFirstLevelCode(level)
	if err != nil {
		return 0, err
	}
	if suffixLength == 0 {
		if err := writeCAVLCLevelCode(bw, code); err != nil {
			return 0, err
		}
		return cavlcSuffixLengthAfterFirstLevel(level, code), nil
	}
	adjusted := level - 1
	if level < 0 {
		adjusted = level + 1
	}
	if err := writeCAVLCSubsequentLevel(bw, adjusted, suffixLength); err != nil {
		return 0, err
	}
	return cavlcSuffixLengthAfterFirstLevel(level, code), nil
}

func cavlcFirstLevelCode(level int32) (int64, error) {
	if level == 0 || level == 1 || level == -1 {
		return 0, ErrInvalidData
	}
	adjusted := level - 1
	if level < 0 {
		adjusted = level + 1
	}
	code := int64(adjusted)*2 - 2
	if adjusted < 0 {
		code = int64(-adjusted)*2 - 1
	}
	if code < 0 {
		return 0, ErrInvalidData
	}
	return code, nil
}

func cavlcSuffixLengthAfterFirstLevel(level int32, code int64) int {
	if code >= 14 {
		return 2
	}
	suffixLength := 1
	if uint32(level)+3 > 6 {
		suffixLength = 2
	}
	return suffixLength
}

func cavlcSuffixLengthAfterSubsequentLevel(level int32, suffixLength int) int {
	suffixLimit := [7]uint32{0, 3, 6, 12, 24, 48, 1<<31 - 1}
	if suffixLength < 0 || suffixLength >= len(suffixLimit) {
		return suffixLength
	}
	limit := suffixLimit[suffixLength]
	if uint32(level)+limit > 2*limit && suffixLength < 6 {
		return suffixLength + 1
	}
	return suffixLength
}

func writeCAVLCSubsequentLevel(bw *BitWriter, level int32, suffixLength int) error {
	if bw == nil || level == 0 || suffixLength < 1 || suffixLength > 6 {
		return ErrInvalidData
	}
	code := int64(level)*2 - 2
	if level < 0 {
		code = int64(-level)*2 - 1
	}
	prefix := code >> uint(suffixLength)
	if code < 0 || prefix >= 15 {
		return ErrInvalidData
	}
	for i := int64(0); i < prefix; i++ {
		bw.WriteBit(0)
	}
	bw.WriteBit(1)
	suffixMask := int64(1<<uint(suffixLength)) - 1
	suffix := code & suffixMask
	for i := suffixLength - 1; i >= 0; i-- {
		bw.WriteBit(uint32((suffix >> uint(i)) & 1))
	}
	return nil
}

func writeCAVLCLevelCode(bw *BitWriter, code int64) error {
	if bw == nil || code < 0 {
		return ErrInvalidData
	}
	prefix := code
	suffix := int64(0)
	suffixBits := 0
	if code >= 30 {
		prefix = 15
		suffix = code - 30
		suffixBits = 12
		if suffix > 0xfff {
			return ErrInvalidData
		}
	} else if code >= 14 {
		prefix = 14
		suffix = code - 14
		suffixBits = 4
	}
	for i := int64(0); i < prefix; i++ {
		bw.WriteBit(0)
	}
	bw.WriteBit(1)
	for i := suffixBits - 1; i >= 0; i-- {
		bw.WriteBit(uint32((suffix >> uint(i)) & 1))
	}
	return nil
}

func readCAVLCLevels(gb *bitReader, totalCoeff int, trailingOnes int) ([16]int32, error) {
	var level [16]int32
	if totalCoeff <= 0 || totalCoeff > 16 || trailingOnes < 0 || trailingOnes > 3 || trailingOnes > totalCoeff {
		return level, ErrInvalidData
	}

	for i := 0; i < trailingOnes; i++ {
		sign, err := gb.readBit()
		if err != nil {
			return level, err
		}
		if sign == 0 {
			level[i] = 1
		} else {
			level[i] = -1
		}
	}

	if trailingOnes < totalCoeff {
		suffixLength := 0
		if totalCoeff > 10 && trailingOnes < 3 {
			suffixLength = 1
		}

		bitsi := gb.showBitsPadded(levelTabBits)
		levelCode := int32(cavlcLevelTable[suffixLength][bitsi][0])
		if err := gb.skipBits(uint32(cavlcLevelTable[suffixLength][bitsi][1])); err != nil {
			return level, err
		}
		if levelCode >= 100 {
			prefix := levelCode - 100
			if prefix == levelTabBits {
				extraPrefix, err := readCAVLCLevelPrefix(gb)
				if err != nil {
					return level, err
				}
				prefix += int32(extraPrefix)
			}

			switch {
			case prefix < 14:
				if suffixLength != 0 {
					bit, err := gb.readBit()
					if err != nil {
						return level, err
					}
					levelCode = (prefix << 1) + int32(bit)
				} else {
					levelCode = prefix
				}
			case prefix == 14:
				if suffixLength != 0 {
					bit, err := gb.readBit()
					if err != nil {
						return level, err
					}
					levelCode = (prefix << 1) + int32(bit)
				} else {
					bits, err := gb.readBits(4)
					if err != nil {
						return level, err
					}
					levelCode = prefix + int32(bits)
				}
			default:
				levelCode = 30
				if prefix >= 16 {
					if prefix > 25+3 {
						return level, ErrInvalidData
					}
					levelCode += (1 << (prefix - 3)) - 4096
				}
				bits, err := gb.readBits(uint32(prefix - 3))
				if err != nil {
					return level, err
				}
				levelCode += int32(bits)
			}

			if trailingOnes < 3 {
				levelCode += 2
			}
			suffixLength = 2
			mask := -(levelCode & 1)
			level[trailingOnes] = (((2 + levelCode) >> 1) ^ mask) - mask
		} else {
			if trailingOnes < 3 {
				levelCode += (levelCode >> 31) | 1
			}
			suffixLength = 1
			if uint32(levelCode)+3 > 6 {
				suffixLength = 2
			}
			level[trailingOnes] = levelCode
		}

		suffixLimit := [7]uint32{0, 3, 6, 12, 24, 48, 1<<31 - 1}
		for i := trailingOnes + 1; i < totalCoeff; i++ {
			bitsi = gb.showBitsPadded(levelTabBits)
			levelCode = int32(cavlcLevelTable[suffixLength][bitsi][0])
			if err := gb.skipBits(uint32(cavlcLevelTable[suffixLength][bitsi][1])); err != nil {
				return level, err
			}
			if levelCode >= 100 {
				prefix := levelCode - 100
				if prefix == levelTabBits {
					extraPrefix, err := readCAVLCLevelPrefix(gb)
					if err != nil {
						return level, err
					}
					prefix += int32(extraPrefix)
				}
				if prefix < 15 {
					bits, err := gb.readBits(uint32(suffixLength))
					if err != nil {
						return level, err
					}
					levelCode = (prefix << suffixLength) + int32(bits)
				} else {
					levelCode = 15 << suffixLength
					if prefix >= 16 {
						if prefix > 25+3 {
							return level, ErrInvalidData
						}
						levelCode += (1 << (prefix - 3)) - 4096
					}
					bits, err := gb.readBits(uint32(prefix - 3))
					if err != nil {
						return level, err
					}
					levelCode += int32(bits)
				}
				mask := -(levelCode & 1)
				levelCode = (((2 + levelCode) >> 1) ^ mask) - mask
			}
			level[i] = levelCode
			limit := suffixLimit[suffixLength]
			if uint32(levelCode)+limit > 2*limit && suffixLength < 6 {
				suffixLength++
			}
		}
	}

	return level, nil
}

func readCAVLCLevelPrefix(gb *bitReader) (int, error) {
	prefix := 0
	for {
		bit, err := gb.readBit()
		if err != nil {
			return 0, err
		}
		if bit != 0 {
			return prefix, nil
		}
		prefix++
	}
}

func decodeCAVLCResidual(gb *bitReader, block []int32, n int, scantable []uint8, qmul []uint32, maxCoeff int, predictedNnz int) (int, error) {
	coeffToken, err := readCAVLCCoeffToken(gb, predictedNnz, maxCoeff)
	if err != nil {
		return 0, err
	}
	totalCoeff := coeffToken >> 2
	trailingOnes := coeffToken & 3
	if totalCoeff == 0 {
		return 0, nil
	}
	if totalCoeff > maxCoeff {
		return 0, ErrInvalidData
	}

	level, err := readCAVLCLevels(gb, totalCoeff, trailingOnes)
	if err != nil {
		return 0, err
	}

	zerosLeft := 0
	if totalCoeff != maxCoeff {
		zerosLeft, err = readCAVLCTotalZeros(gb, totalCoeff, maxCoeff)
		if err != nil {
			return 0, err
		}
	}

	scanIndex := zerosLeft + totalCoeff - 1
	if scanIndex < 0 || scanIndex >= len(scantable) {
		return 0, ErrInvalidData
	}
	if err := storeCAVLCLevel(block, scantable[scanIndex], level[0], qmul, n); err != nil {
		return 0, err
	}

	i := 1
	for ; i < totalCoeff && zerosLeft > 0; i++ {
		runBefore, err := readCAVLCRunBefore(gb, zerosLeft)
		if err != nil {
			return 0, err
		}
		zerosLeft -= runBefore
		scanIndex -= 1 + runBefore
		if zerosLeft < 0 || scanIndex < 0 || scanIndex >= len(scantable) {
			return 0, ErrInvalidData
		}
		if err := storeCAVLCLevel(block, scantable[scanIndex], level[i], qmul, n); err != nil {
			return 0, err
		}
	}

	for ; i < totalCoeff; i++ {
		scanIndex--
		if scanIndex < 0 || scanIndex >= len(scantable) {
			return 0, ErrInvalidData
		}
		if err := storeCAVLCLevel(block, scantable[scanIndex], level[i], qmul, n); err != nil {
			return 0, err
		}
	}

	if zerosLeft < 0 {
		return 0, ErrInvalidData
	}
	return totalCoeff, nil
}

func storeCAVLCLevel(block []int32, scanPos uint8, level int32, qmul []uint32, n int) error {
	if int(scanPos) >= len(block) {
		return ErrInvalidData
	}
	if qmul == nil || n >= lumaDCBlockIndex {
		block[scanPos] = level
		return nil
	}
	if int(scanPos) >= len(qmul) {
		return ErrInvalidData
	}
	block[scanPos] = int32(uint32(level)*qmul[scanPos]+32) >> 6
	return nil
}
