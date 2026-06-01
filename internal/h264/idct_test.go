// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

import "testing"

func TestH264IDCTDCAddClipsAndClears(t *testing.T) {
	dst := []uint8{
		250, 1, 2, 3,
		4, 5, 6, 7,
		8, 9, 10, 11,
		12, 13, 14, 15,
	}
	block := make([]int32, 16)
	block[0] = 512

	if err := h264IDCTDCAdd(dst, block, 4); err != nil {
		t.Fatal(err)
	}
	if dst[0] != 255 || dst[1] != 9 || dst[15] != 23 {
		t.Fatalf("dst endpoints = %d/%d/%d, want 255/9/23", dst[0], dst[1], dst[15])
	}
	if block[0] != 0 {
		t.Fatalf("block[0] = %d, want cleared", block[0])
	}
}

func TestH264IDCTDCAddHighClipsAndClears(t *testing.T) {
	dst := []uint16{
		1020, 1, 2, 3,
		4, 5, 6, 7,
		8, 9, 10, 11,
		12, 13, 14, 15,
	}
	block := make([]int32, 16)
	block[0] = 512

	if err := h264IDCTDCAddHigh(dst, block, 4, 10); err != nil {
		t.Fatal(err)
	}
	if dst[0] != 1023 || dst[1] != 9 || dst[15] != 23 {
		t.Fatalf("high dst endpoints = %d/%d/%d, want 1023/9/23", dst[0], dst[1], dst[15])
	}
	if block[0] != 0 {
		t.Fatalf("high block[0] = %d, want cleared", block[0])
	}
	if err := h264IDCTDCAddHigh(make([]uint16, 16), make([]int32, 16), 4, 11); err != ErrUnsupported {
		t.Fatalf("unsupported bit depth err = %v, want ErrUnsupported", err)
	}
}

func TestH264IDCTAdd16DispatchesDCAndFullBlocks(t *testing.T) {
	offsets, err := h264FrameBlockOffsets(16, 8, 0)
	if err != nil {
		t.Fatal(err)
	}
	dst := make([]uint8, 16*16)
	for i := range dst {
		dst[i] = 100
	}
	var nnzc [h264NonZeroCountCacheSize]uint8
	block := make([]int32, 48*16)

	nnzc[h264Scan8[0]] = 1
	block[0] = 128
	nnzc[h264Scan8[5]] = 2
	block[5*16+0] = 96
	block[5*16+1] = -16

	if err := h264IDCTAdd16(dst, &offsets, block, 16, &nnzc); err != nil {
		t.Fatal(err)
	}
	if dst[0] != 102 || dst[3+3*16] != 102 {
		t.Fatalf("dc block endpoints = %d/%d, want 102", dst[0], dst[3+3*16])
	}
	fullOffset := offsets[5]
	if dst[fullOffset] == 100 || dst[fullOffset+3+3*16] == 100 {
		t.Fatalf("full block did not modify expected pixels at offset %d", fullOffset)
	}
	if block[0] != 0 || block[5*16] != 0 || block[5*16+1] != 0 {
		t.Fatalf("blocks not cleared: %d/%d/%d", block[0], block[5*16], block[5*16+1])
	}
}

func TestH264IDCTAdd16HighDispatchesDCAndFullBlocks(t *testing.T) {
	offsets, err := h264FrameBlockOffsets(16, 8, 0)
	if err != nil {
		t.Fatal(err)
	}
	dst := make([]uint16, 16*16)
	for i := range dst {
		dst[i] = 512
	}
	var nnzc [h264NonZeroCountCacheSize]uint8
	block := make([]int32, 48*16)

	nnzc[h264Scan8[0]] = 1
	block[0] = 128
	nnzc[h264Scan8[5]] = 2
	block[5*16+0] = 96
	block[5*16+1] = -16

	if err := h264IDCTAdd16High(dst, &offsets, block, 16, &nnzc, 10); err != nil {
		t.Fatal(err)
	}
	if dst[0] != 514 || dst[3+3*16] != 514 {
		t.Fatalf("high dc block endpoints = %d/%d, want 514", dst[0], dst[3+3*16])
	}
	fullOffset := offsets[5]
	if dst[fullOffset] == 512 || dst[fullOffset+3+3*16] == 512 {
		t.Fatalf("high full block did not modify expected pixels at offset %d", fullOffset)
	}
	if block[0] != 0 || block[5*16] != 0 || block[5*16+1] != 0 {
		t.Fatalf("high blocks not cleared: %d/%d/%d", block[0], block[5*16], block[5*16+1])
	}
}

func TestH264IDCT8Add4DispatchesDCAndFullBlocks(t *testing.T) {
	offsets, err := h264FrameBlockOffsets(16, 8, 0)
	if err != nil {
		t.Fatal(err)
	}
	dst := make([]uint8, 16*16)
	for i := range dst {
		dst[i] = 100
	}
	var nnzc [h264NonZeroCountCacheSize]uint8
	block := make([]int32, 48*16)

	nnzc[h264Scan8[0]] = 1
	block[0] = 128
	nnzc[h264Scan8[4]] = 2
	block[4*16+0] = 96
	block[4*16+1] = -16

	if err := h264IDCT8Add4(dst, &offsets, block, 16, &nnzc); err != nil {
		t.Fatal(err)
	}
	if dst[0] != 102 || dst[7+7*16] != 102 {
		t.Fatalf("8x8 dc block endpoints = %d/%d, want 102", dst[0], dst[7+7*16])
	}
	if !regionChangedFrom(dst, 100, offsets[4], 16, 8) {
		t.Fatalf("8x8 full block did not modify expected pixels at offset %d", offsets[4])
	}
	if block[0] != 0 || block[4*16] != 0 || block[4*16+1] != 0 {
		t.Fatalf("8x8 blocks not cleared: %d/%d/%d", block[0], block[4*16], block[4*16+1])
	}
}

func TestH264IDCT8Add4HighDispatchesDCAndFullBlocks(t *testing.T) {
	offsets, err := h264FrameBlockOffsets(16, 8, 0)
	if err != nil {
		t.Fatal(err)
	}
	dst := make([]uint16, 16*16)
	for i := range dst {
		dst[i] = 512
	}
	var nnzc [h264NonZeroCountCacheSize]uint8
	block := make([]int32, 48*16)

	nnzc[h264Scan8[0]] = 1
	block[0] = 128
	nnzc[h264Scan8[4]] = 2
	block[4*16+0] = 96
	block[4*16+1] = -16

	if err := h264IDCT8Add4High(dst, &offsets, block, 16, &nnzc, 12); err != nil {
		t.Fatal(err)
	}
	if dst[0] != 514 || dst[7+7*16] != 514 {
		t.Fatalf("high 8x8 dc block endpoints = %d/%d, want 514", dst[0], dst[7+7*16])
	}
	if !regionChangedFromHigh(dst, 512, offsets[4], 16, 8) {
		t.Fatalf("high 8x8 full block did not modify expected pixels at offset %d", offsets[4])
	}
	if block[0] != 0 || block[4*16] != 0 || block[4*16+1] != 0 {
		t.Fatalf("high 8x8 blocks not cleared: %d/%d/%d", block[0], block[4*16], block[4*16+1])
	}
}

func TestH264IDCTAdd16IntraSkipsUntouchedBlocksWithoutDestinationCheck(t *testing.T) {
	offsets, err := h264FrameBlockOffsets(16, 8, 0)
	if err != nil {
		t.Fatal(err)
	}
	var nnzc [h264NonZeroCountCacheSize]uint8
	block := make([]int32, 48*16)

	if err := h264IDCTAdd16Intra(nil, &offsets, block, 16, &nnzc); err != nil {
		t.Fatalf("zero intra blocks should be skipped without touching dst: %v", err)
	}
}

func TestH264IDCTAdd16IntraHighSkipsUntouchedBlocksWithoutDestinationCheck(t *testing.T) {
	offsets, err := h264FrameBlockOffsets(16, 8, 0)
	if err != nil {
		t.Fatal(err)
	}
	var nnzc [h264NonZeroCountCacheSize]uint8
	block := make([]int32, 48*16)

	if err := h264IDCTAdd16IntraHigh(nil, &offsets, block, 16, &nnzc, 10); err != nil {
		t.Fatalf("zero high intra blocks should be skipped without touching dst: %v", err)
	}
}

func TestH264IDCTAdd8DispatchesChroma420(t *testing.T) {
	offsets, err := h264FrameBlockOffsets(16, 8, 0)
	if err != nil {
		t.Fatal(err)
	}
	dest := [2][]uint8{make([]uint8, 8*8), make([]uint8, 8*8)}
	for i := range dest[0] {
		dest[0][i] = 50
		dest[1][i] = 60
	}
	var nnzc [h264NonZeroCountCacheSize]uint8
	block := make([]int32, 48*16)

	nnzc[h264Scan8[16]] = 1
	block[16*16] = 128
	nnzc[h264Scan8[32]] = 2
	block[32*16+0] = 96
	block[32*16+1] = -16

	if err := h264IDCTAdd8(&dest, &offsets, block, 8, &nnzc); err != nil {
		t.Fatal(err)
	}
	if dest[0][0] != 52 || dest[0][3+3*8] != 52 {
		t.Fatalf("chroma420 dc block endpoints = %d/%d, want 52", dest[0][0], dest[0][3+3*8])
	}
	if !regionChangedFrom(dest[1], 60, offsets[32], 8, 4) {
		t.Fatalf("chroma420 full block did not modify expected pixels at offset %d", offsets[32])
	}
	if block[16*16] != 0 || block[32*16] != 0 || block[32*16+1] != 0 {
		t.Fatalf("chroma420 blocks not cleared: %d/%d/%d", block[16*16], block[32*16], block[32*16+1])
	}
}

func TestH264IDCTAdd8HighDispatchesChroma420(t *testing.T) {
	offsets, err := h264FrameBlockOffsets(16, 8, 0)
	if err != nil {
		t.Fatal(err)
	}
	dest := [2][]uint16{make([]uint16, 8*8), make([]uint16, 8*8)}
	for i := range dest[0] {
		dest[0][i] = 300
		dest[1][i] = 400
	}
	var nnzc [h264NonZeroCountCacheSize]uint8
	block := make([]int32, 48*16)

	nnzc[h264Scan8[16]] = 1
	block[16*16] = 128
	nnzc[h264Scan8[32]] = 2
	block[32*16+0] = 96
	block[32*16+1] = -16

	if err := h264IDCTAdd8High(&dest, &offsets, block, 8, &nnzc, 10); err != nil {
		t.Fatal(err)
	}
	if dest[0][0] != 302 || dest[0][3+3*8] != 302 {
		t.Fatalf("high chroma420 dc block endpoints = %d/%d, want 302", dest[0][0], dest[0][3+3*8])
	}
	if !regionChangedFromHigh(dest[1], 400, offsets[32], 8, 4) {
		t.Fatalf("high chroma420 full block did not modify expected pixels at offset %d", offsets[32])
	}
	if block[16*16] != 0 || block[32*16] != 0 || block[32*16+1] != 0 {
		t.Fatalf("high chroma420 blocks not cleared: %d/%d/%d", block[16*16], block[32*16], block[32*16+1])
	}
}

func TestH264IDCTAdd8_422DispatchesLowerRows(t *testing.T) {
	offsets, err := h264FrameBlockOffsets(16, 8, 0)
	if err != nil {
		t.Fatal(err)
	}
	dest := [2][]uint8{make([]uint8, 8*16), make([]uint8, 8*16)}
	for i := range dest[0] {
		dest[0][i] = 40
		dest[1][i] = 70
	}
	var nnzc [h264NonZeroCountCacheSize]uint8
	block := make([]int32, 48*16)

	nnzc[h264Scan8[16]] = 1
	block[16*16] = 128
	nnzc[h264Scan8[24]] = 1
	block[20*16] = 256

	if err := h264IDCTAdd8_422(&dest, &offsets, block, 8, &nnzc); err != nil {
		t.Fatal(err)
	}
	if dest[0][offsets[16]] != 42 || dest[0][offsets[16]+3+3*8] != 42 {
		t.Fatalf("chroma422 upper block endpoints = %d/%d, want 42", dest[0][offsets[16]], dest[0][offsets[16]+3+3*8])
	}
	if dest[0][offsets[24]] != 44 || dest[0][offsets[24]+3+3*8] != 44 {
		t.Fatalf("chroma422 lower block endpoints = %d/%d, want 44", dest[0][offsets[24]], dest[0][offsets[24]+3+3*8])
	}
	if block[16*16] != 0 || block[20*16] != 0 {
		t.Fatalf("chroma422 blocks not cleared: %d/%d", block[16*16], block[20*16])
	}
}

func TestH264IDCTAdd8_422HighDispatchesLowerRows(t *testing.T) {
	offsets, err := h264FrameBlockOffsets(16, 8, 0)
	if err != nil {
		t.Fatal(err)
	}
	dest := [2][]uint16{make([]uint16, 8*16), make([]uint16, 8*16)}
	for i := range dest[0] {
		dest[0][i] = 300
		dest[1][i] = 400
	}
	var nnzc [h264NonZeroCountCacheSize]uint8
	block := make([]int32, 48*16)

	nnzc[h264Scan8[16]] = 1
	block[16*16] = 128
	nnzc[h264Scan8[24]] = 1
	block[20*16] = 256

	if err := h264IDCTAdd8_422High(&dest, &offsets, block, 8, &nnzc, 12); err != nil {
		t.Fatal(err)
	}
	if dest[0][offsets[16]] != 302 || dest[0][offsets[16]+3+3*8] != 302 {
		t.Fatalf("high chroma422 upper block endpoints = %d/%d, want 302", dest[0][offsets[16]], dest[0][offsets[16]+3+3*8])
	}
	if dest[0][offsets[24]] != 304 || dest[0][offsets[24]+3+3*8] != 304 {
		t.Fatalf("high chroma422 lower block endpoints = %d/%d, want 304", dest[0][offsets[24]], dest[0][offsets[24]+3+3*8])
	}
	if block[16*16] != 0 || block[20*16] != 0 {
		t.Fatalf("high chroma422 blocks not cleared: %d/%d", block[16*16], block[20*16])
	}
}

func TestH264LumaDCDequantIDCTScattersDCBlocks(t *testing.T) {
	var input [16]int32
	for i := range input {
		input[i] = int32(i - 7)
	}
	output := make([]int32, 16*16)
	for i := range output {
		output[i] = 12345
	}

	if err := h264LumaDCDequantIDCT(output, &input, 64); err != nil {
		t.Fatal(err)
	}
	for _, idx := range []int{0, 16, 64, 80, 32, 48, 96, 112, 128, 144, 192, 208, 160, 176, 224, 240} {
		if output[idx] == 12345 {
			t.Fatalf("expected scattered dc coefficient at %d to be written", idx)
		}
	}
	if output[1] != 12345 || output[255] != 12345 {
		t.Fatalf("unexpected non-dc writes output[1]/[255] = %d/%d", output[1], output[255])
	}
}

func TestH264LumaDCDequantIDCTHighScattersDCBlocks(t *testing.T) {
	var input [16]int32
	for i := range input {
		input[i] = int32(i*5 - 30)
	}
	output := make([]int32, 16*16)
	for i := range output {
		output[i] = 12345
	}

	if err := h264LumaDCDequantIDCTHigh(output, &input, 96); err != nil {
		t.Fatal(err)
	}
	for _, idx := range []int{0, 16, 64, 80, 32, 48, 96, 112, 128, 144, 192, 208, 160, 176, 224, 240} {
		if output[idx] == 12345 {
			t.Fatalf("expected high scattered dc coefficient at %d to be written", idx)
		}
	}
	if output[1] != 12345 || output[255] != 12345 {
		t.Fatalf("unexpected high non-dc writes output[1]/[255] = %d/%d", output[1], output[255])
	}
}

func TestH264ChromaDCDequantIDCT420(t *testing.T) {
	block := make([]int32, 64)
	block[0] = 1
	block[16] = 2
	block[32] = 3
	block[48] = 4

	if err := h264ChromaDCDequantIDCT(block, 64); err != nil {
		t.Fatal(err)
	}
	if block[0] != 5 || block[16] != -1 || block[32] != -2 || block[48] != 0 {
		t.Fatalf("chroma dc = %d/%d/%d/%d, want 5/-1/-2/0", block[0], block[16], block[32], block[48])
	}
}

func TestH264ChromaDCDequantIDCTHigh420(t *testing.T) {
	block := make([]int32, 64)
	block[0] = 3
	block[16] = -4
	block[32] = 8
	block[48] = -11

	if err := h264ChromaDCDequantIDCTHigh(block, 96); err != nil {
		t.Fatal(err)
	}
	if block[0] != -3 || block[16] != 19 || block[32] != 1 || block[48] != -9 {
		t.Fatalf("high chroma dc = %d/%d/%d/%d, want -3/19/1/-9", block[0], block[16], block[32], block[48])
	}
}

func TestH264FrameBlockOffsets(t *testing.T) {
	offsets, err := h264FrameBlockOffsets(16, 8, 0)
	if err != nil {
		t.Fatal(err)
	}
	if offsets[0] != 0 || offsets[1] != 4 || offsets[2] != 64 || offsets[15] != 204 {
		t.Fatalf("luma offsets = %d/%d/%d/%d", offsets[0], offsets[1], offsets[2], offsets[15])
	}
	if offsets[16] != 0 || offsets[17] != 4 || offsets[18] != 32 || offsets[31] != 108 || offsets[32] != offsets[16] {
		t.Fatalf("chroma offsets = %d/%d/%d/%d mirrored %d", offsets[16], offsets[17], offsets[18], offsets[31], offsets[32])
	}
}

func regionChangedFrom(dst []uint8, baseline uint8, offset int, stride int, size int) bool {
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			if dst[offset+y*stride+x] != baseline {
				return true
			}
		}
	}
	return false
}

func regionChangedFromHigh(dst []uint16, baseline uint16, offset int, stride int, size int) bool {
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			if dst[offset+y*stride+x] != baseline {
				return true
			}
		}
	}
	return false
}
