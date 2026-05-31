// SPDX-License-Identifier: LGPL-2.1-or-later
//
// Source-shaped port of H.264 decoder tables and MB-type flags from FFmpeg
// n8.0.1 libavcodec/mpegutils.h, h264_parse.h, and h264data.c.

package h264

const (
	qpMaxNum = 51 + 6*6

	MBTypeIntra4x4   uint32 = 1 << 0
	MBTypeIntra16x16 uint32 = 1 << 1
	MBTypeIntraPCM   uint32 = 1 << 2
	MBType16x16      uint32 = 1 << 3
	MBType16x8       uint32 = 1 << 4
	MBType8x16       uint32 = 1 << 5
	MBType8x8        uint32 = 1 << 6
	MBTypeInterlaced uint32 = 1 << 7
	MBTypeDirect2    uint32 = 1 << 8
	MBTypeRef0       uint32 = 1 << 9
	MBTypeCBP        uint32 = 1 << 10
	MBTypeQuant      uint32 = 1 << 11
	MBTypeP0L0       uint32 = 1 << 12
	MBTypeP1L0       uint32 = 1 << 13
	MBTypeP0L1       uint32 = 1 << 14
	MBTypeP1L1       uint32 = 1 << 15
	MBTypeL0                = MBTypeP0L0 | MBTypeP1L0
	MBTypeL1                = MBTypeP0L1 | MBTypeP1L1
	MBTypeL0L1              = MBTypeL0 | MBTypeL1
	MBTypeSkip       uint32 = 1 << 17
	MBTypeACPRed     uint32 = 1 << 18
	MBType8x8DCT     uint32 = 0x01000000
)

type IMBInfo struct {
	Type     uint32
	PredMode int8
	CBP      int8
}

type PMBInfo struct {
	Type           uint32
	PartitionCount uint8
}

var h264IMBTypeInfo = [26]IMBInfo{
	{MBTypeIntra4x4, -1, -1},
	{MBTypeIntra16x16, 2, 0},
	{MBTypeIntra16x16, 1, 0},
	{MBTypeIntra16x16, 0, 0},
	{MBTypeIntra16x16, 3, 0},
	{MBTypeIntra16x16, 2, 16},
	{MBTypeIntra16x16, 1, 16},
	{MBTypeIntra16x16, 0, 16},
	{MBTypeIntra16x16, 3, 16},
	{MBTypeIntra16x16, 2, 32},
	{MBTypeIntra16x16, 1, 32},
	{MBTypeIntra16x16, 0, 32},
	{MBTypeIntra16x16, 3, 32},
	{MBTypeIntra16x16, 2, 15},
	{MBTypeIntra16x16, 1, 15},
	{MBTypeIntra16x16, 0, 15},
	{MBTypeIntra16x16, 3, 15},
	{MBTypeIntra16x16, 2, 15 + 16},
	{MBTypeIntra16x16, 1, 15 + 16},
	{MBTypeIntra16x16, 0, 15 + 16},
	{MBTypeIntra16x16, 3, 15 + 16},
	{MBTypeIntra16x16, 2, 15 + 32},
	{MBTypeIntra16x16, 1, 15 + 32},
	{MBTypeIntra16x16, 0, 15 + 32},
	{MBTypeIntra16x16, 3, 15 + 32},
	{MBTypeIntraPCM, -1, -1},
}

var h264PMBTypeInfo = [5]PMBInfo{
	{MBType16x16 | MBTypeP0L0, 1},
	{MBType16x8 | MBTypeP0L0 | MBTypeP1L0, 2},
	{MBType8x16 | MBTypeP0L0 | MBTypeP1L0, 2},
	{MBType8x8 | MBTypeP0L0 | MBTypeP1L0, 4},
	{MBType8x8 | MBTypeP0L0 | MBTypeP1L0 | MBTypeRef0, 4},
}

var h264PSubMBTypeInfo = [4]PMBInfo{
	{MBType16x16 | MBTypeP0L0, 1},
	{MBType16x8 | MBTypeP0L0, 2},
	{MBType8x16 | MBTypeP0L0, 2},
	{MBType8x8 | MBTypeP0L0, 4},
}

var h264BMBTypeInfo = [23]PMBInfo{
	{MBTypeDirect2 | MBTypeL0L1, 1},
	{MBType16x16 | MBTypeP0L0, 1},
	{MBType16x16 | MBTypeP0L1, 1},
	{MBType16x16 | MBTypeP0L0 | MBTypeP0L1, 1},
	{MBType16x8 | MBTypeP0L0 | MBTypeP1L0, 2},
	{MBType8x16 | MBTypeP0L0 | MBTypeP1L0, 2},
	{MBType16x8 | MBTypeP0L1 | MBTypeP1L1, 2},
	{MBType8x16 | MBTypeP0L1 | MBTypeP1L1, 2},
	{MBType16x8 | MBTypeP0L0 | MBTypeP1L1, 2},
	{MBType8x16 | MBTypeP0L0 | MBTypeP1L1, 2},
	{MBType16x8 | MBTypeP0L1 | MBTypeP1L0, 2},
	{MBType8x16 | MBTypeP0L1 | MBTypeP1L0, 2},
	{MBType16x8 | MBTypeP0L0 | MBTypeP1L0 | MBTypeP1L1, 2},
	{MBType8x16 | MBTypeP0L0 | MBTypeP1L0 | MBTypeP1L1, 2},
	{MBType16x8 | MBTypeP0L1 | MBTypeP1L0 | MBTypeP1L1, 2},
	{MBType8x16 | MBTypeP0L1 | MBTypeP1L0 | MBTypeP1L1, 2},
	{MBType16x8 | MBTypeP0L0 | MBTypeP0L1 | MBTypeP1L0, 2},
	{MBType8x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeP1L0, 2},
	{MBType16x8 | MBTypeP0L0 | MBTypeP0L1 | MBTypeP1L1, 2},
	{MBType8x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeP1L1, 2},
	{MBType16x8 | MBTypeP0L0 | MBTypeP0L1 | MBTypeP1L0 | MBTypeP1L1, 2},
	{MBType8x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeP1L0 | MBTypeP1L1, 2},
	{MBType8x8 | MBTypeP0L0 | MBTypeP0L1 | MBTypeP1L0 | MBTypeP1L1, 4},
}

var h264BSubMBTypeInfo = [13]PMBInfo{
	{MBTypeDirect2, 1},
	{MBType16x16 | MBTypeP0L0, 1},
	{MBType16x16 | MBTypeP0L1, 1},
	{MBType16x16 | MBTypeP0L0 | MBTypeP0L1, 1},
	{MBType16x8 | MBTypeP0L0 | MBTypeP1L0, 2},
	{MBType8x16 | MBTypeP0L0 | MBTypeP1L0, 2},
	{MBType16x8 | MBTypeP0L1 | MBTypeP1L1, 2},
	{MBType8x16 | MBTypeP0L1 | MBTypeP1L1, 2},
	{MBType16x8 | MBTypeP0L0 | MBTypeP0L1 | MBTypeP1L0 | MBTypeP1L1, 2},
	{MBType8x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeP1L0 | MBTypeP1L1, 2},
	{MBType8x8 | MBTypeP0L0 | MBTypeP1L0, 4},
	{MBType8x8 | MBTypeP0L1 | MBTypeP1L1, 4},
	{MBType8x8 | MBTypeP0L0 | MBTypeP0L1 | MBTypeP1L0 | MBTypeP1L1, 4},
}

var h264Dequant4CoeffInit = [6][3]uint8{
	{10, 13, 16},
	{11, 14, 18},
	{13, 16, 20},
	{14, 18, 23},
	{16, 20, 25},
	{18, 23, 29},
}

var h264Dequant8CoeffInitScan = [16]uint8{
	0, 3, 4, 3, 3, 1, 5, 1, 4, 5, 2, 5, 3, 1, 5, 1,
}

var h264Dequant8CoeffInit = [6][6]uint8{
	{20, 18, 32, 19, 25, 24},
	{22, 19, 35, 21, 28, 26},
	{26, 23, 42, 24, 33, 31},
	{28, 25, 45, 26, 35, 33},
	{32, 28, 51, 30, 40, 38},
	{36, 32, 58, 34, 46, 43},
}

var h264QuantRem6 = [qpMaxNum + 1]uint8{
	0, 1, 2, 3, 4, 5, 0, 1, 2, 3, 4, 5, 0, 1, 2, 3, 4, 5,
	0, 1, 2, 3, 4, 5, 0, 1, 2, 3, 4, 5, 0, 1, 2, 3, 4, 5,
	0, 1, 2, 3, 4, 5, 0, 1, 2, 3, 4, 5, 0, 1, 2, 3, 4, 5,
	0, 1, 2, 3, 4, 5, 0, 1, 2, 3, 4, 5, 0, 1, 2, 3, 4, 5,
	0, 1, 2, 3, 4, 5, 0, 1, 2, 3, 4, 5, 0, 1, 2, 3,
}

var h264QuantDiv6 = [qpMaxNum + 1]uint8{
	0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 2, 2, 2, 2, 2, 2,
	3, 3, 3, 3, 3, 3, 4, 4, 4, 4, 4, 4, 5, 5, 5, 5, 5, 5,
	6, 6, 6, 6, 6, 6, 7, 7, 7, 7, 7, 7, 8, 8, 8, 8, 8, 8,
	9, 9, 9, 9, 9, 9, 10, 10, 10, 10, 10, 10, 11, 11, 11, 11, 11, 11,
	12, 12, 12, 12, 12, 12, 13, 13, 13, 13, 13, 13, 14, 14, 14, 14,
}

var h264ZigzagScan = [16]uint8{
	0 + 0*4, 1 + 0*4, 0 + 1*4, 0 + 2*4,
	1 + 1*4, 2 + 0*4, 3 + 0*4, 2 + 1*4,
	1 + 2*4, 0 + 3*4, 1 + 3*4, 2 + 2*4,
	3 + 1*4, 3 + 2*4, 2 + 3*4, 3 + 3*4,
}

var h264Scan8 = [16*3 + 3]uint8{
	4 + 1*8, 5 + 1*8, 4 + 2*8, 5 + 2*8,
	6 + 1*8, 7 + 1*8, 6 + 2*8, 7 + 2*8,
	4 + 3*8, 5 + 3*8, 4 + 4*8, 5 + 4*8,
	6 + 3*8, 7 + 3*8, 6 + 4*8, 7 + 4*8,
	4 + 6*8, 5 + 6*8, 4 + 7*8, 5 + 7*8,
	6 + 6*8, 7 + 6*8, 6 + 7*8, 7 + 7*8,
	4 + 8*8, 5 + 8*8, 4 + 9*8, 5 + 9*8,
	6 + 8*8, 7 + 8*8, 6 + 9*8, 7 + 9*8,
	4 + 11*8, 5 + 11*8, 4 + 12*8, 5 + 12*8,
	6 + 11*8, 7 + 11*8, 6 + 12*8, 7 + 12*8,
	4 + 13*8, 5 + 13*8, 4 + 14*8, 5 + 14*8,
	6 + 13*8, 7 + 13*8, 6 + 14*8, 7 + 14*8,
	0 + 0*8, 0 + 5*8, 0 + 10*8,
}

var h264FieldScan = [16]uint8{
	0 + 0*4, 0 + 1*4, 1 + 0*4, 0 + 2*4,
	0 + 3*4, 1 + 1*4, 1 + 2*4, 1 + 3*4,
	2 + 0*4, 2 + 1*4, 2 + 2*4, 2 + 3*4,
	3 + 0*4, 3 + 1*4, 3 + 2*4, 3 + 3*4,
}

var h264ZigzagDirect = [64]uint8{
	0, 1, 8, 16, 9, 2, 3, 10,
	17, 24, 32, 25, 18, 11, 4, 5,
	12, 19, 26, 33, 40, 48, 41, 34,
	27, 20, 13, 6, 7, 14, 21, 28,
	35, 42, 49, 56, 57, 50, 43, 36,
	29, 22, 15, 23, 30, 37, 44, 51,
	58, 59, 52, 45, 38, 31, 39, 46,
	53, 60, 61, 54, 47, 55, 62, 63,
}

var h264ZigzagScan8x8CAVLCRaw = [64]uint8{
	0 + 0*8, 1 + 1*8, 1 + 2*8, 2 + 2*8,
	4 + 1*8, 0 + 5*8, 3 + 3*8, 7 + 0*8,
	3 + 4*8, 1 + 7*8, 5 + 3*8, 6 + 3*8,
	2 + 7*8, 6 + 4*8, 5 + 6*8, 7 + 5*8,
	1 + 0*8, 2 + 0*8, 0 + 3*8, 3 + 1*8,
	3 + 2*8, 0 + 6*8, 4 + 2*8, 6 + 1*8,
	2 + 5*8, 2 + 6*8, 6 + 2*8, 5 + 4*8,
	3 + 7*8, 7 + 3*8, 4 + 7*8, 7 + 6*8,
	0 + 1*8, 3 + 0*8, 0 + 4*8, 4 + 0*8,
	2 + 3*8, 1 + 5*8, 5 + 1*8, 5 + 2*8,
	1 + 6*8, 3 + 5*8, 7 + 1*8, 4 + 5*8,
	4 + 6*8, 7 + 4*8, 5 + 7*8, 6 + 7*8,
	0 + 2*8, 2 + 1*8, 1 + 3*8, 5 + 0*8,
	1 + 4*8, 2 + 4*8, 6 + 0*8, 4 + 3*8,
	0 + 7*8, 4 + 4*8, 7 + 2*8, 3 + 6*8,
	5 + 5*8, 6 + 5*8, 6 + 6*8, 7 + 7*8,
}

var h264ChromaDCScan = [4]uint8{
	(0 + 0*2) * 16, (1 + 0*2) * 16,
	(0 + 1*2) * 16, (1 + 1*2) * 16,
}

var h264Chroma422DCScan = [8]uint8{
	(0 + 0*2) * 16, (0 + 1*2) * 16,
	(1 + 0*2) * 16, (0 + 2*2) * 16,
	(0 + 3*2) * 16, (1 + 1*2) * 16,
	(1 + 2*2) * 16, (1 + 3*2) * 16,
}

var h264ZigzagScanCAVLC = transposeScan4(h264ZigzagScan)
var h264FieldScanCAVLC = transposeScan4(h264FieldScan)
var h264ZigzagScan8x8CAVLC = transposeScan8(h264ZigzagScan8x8CAVLCRaw)

var h264DefaultScaling4 = [2][16]uint8{
	{6, 13, 20, 28, 13, 20, 28, 32, 20, 28, 32, 37, 28, 32, 37, 42},
	{10, 14, 20, 24, 14, 20, 24, 27, 20, 24, 27, 30, 24, 27, 30, 34},
}

var h264DefaultScaling8 = [2][64]uint8{
	{
		6, 10, 13, 16, 18, 23, 25, 27,
		10, 11, 16, 18, 23, 25, 27, 29,
		13, 16, 18, 23, 25, 27, 29, 31,
		16, 18, 23, 25, 27, 29, 31, 33,
		18, 23, 25, 27, 29, 31, 33, 36,
		23, 25, 27, 29, 31, 33, 36, 38,
		25, 27, 29, 31, 33, 36, 38, 40,
		27, 29, 31, 33, 36, 38, 40, 42,
	},
	{
		9, 13, 15, 17, 19, 21, 22, 24,
		13, 13, 17, 19, 21, 22, 24, 25,
		15, 17, 19, 21, 22, 24, 25, 27,
		17, 19, 21, 22, 24, 25, 27, 28,
		19, 21, 22, 24, 25, 27, 28, 30,
		21, 22, 24, 25, 27, 28, 30, 32,
		22, 24, 25, 27, 28, 30, 32, 33,
		24, 25, 27, 28, 30, 32, 33, 35,
	},
}

func h264ChromaQP(depth int32, qp uint32) uint8 {
	if depth < 8 || depth > 14 {
		return 0
	}
	prefix := int((depth - 8) * 6)
	if int(qp) < prefix {
		return uint8(qp)
	}
	base := [52]uint8{
		0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11,
		12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23,
		24, 25, 26, 27, 28, 29, 29, 30, 31, 32, 32, 33,
		34, 34, 35, 35, 36, 36, 37, 37, 37, 38, 38, 38,
		39, 39, 39, 39,
	}
	return base[int(qp)-prefix] + uint8(prefix)
}

func transposeScan4(in [16]uint8) [16]uint8 {
	var out [16]uint8
	for i, v := range in {
		out[i] = (v >> 2) | ((v << 2) & 0x0f)
	}
	return out
}

func transposeScan8(in [64]uint8) [64]uint8 {
	var out [64]uint8
	for i, v := range in {
		out[i] = (v >> 3) | ((v & 7) << 3)
	}
	return out
}
