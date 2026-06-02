// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

import (
	"fmt"
	"testing"
)

func TestNewSimpleDecodedFrameHighAllocatesUint16Planes(t *testing.T) {
	for _, bitDepth := range []int32{9, 10, 12, 14} {
		t.Run(fmt.Sprintf("%d-bit", bitDepth), func(t *testing.T) {
			sps := simpleDecodeFrameStorageTestSPS(bitDepth, bitDepth, 1)

			frame, tables, err := newSimpleDecodedFrame(sps)
			if err != nil {
				t.Fatalf("new high frame failed: %v", err)
			}
			if tables == nil || frame.tables != tables {
				t.Fatalf("macroblock tables not attached")
			}
			if len(frame.Y) != 0 || len(frame.Cb) != 0 || len(frame.Cr) != 0 {
				t.Fatalf("8-bit planes allocated for %d-bit frame", bitDepth)
			}
			if len(frame.Y16) != frame.LumaStride*frame.MBHeight*16 {
				t.Fatalf("Y16 len = %d, want %d", len(frame.Y16), frame.LumaStride*frame.MBHeight*16)
			}
			chromaWidth, chromaHeight := h264ChromaFrameSize(frame.MBWidth, frame.MBHeight, frame.ChromaFormatIDC)
			if frame.ChromaStride != chromaWidth || len(frame.Cb16) != chromaWidth*chromaHeight || len(frame.Cr16) != chromaWidth*chromaHeight {
				t.Fatalf("chroma geometry stride/len = %d/%d/%d, want %d/%d/%d",
					frame.ChromaStride, len(frame.Cb16), len(frame.Cr16),
					chromaWidth, chromaWidth*chromaHeight, chromaWidth*chromaHeight)
			}
			if frame.BitDepthLuma != int(bitDepth) || frame.BitDepthChroma != int(bitDepth) {
				t.Fatalf("bit depths = %d/%d, want %d/%d", frame.BitDepthLuma, frame.BitDepthChroma, bitDepth, bitDepth)
			}
		})
	}
}

func TestNewSimpleDecodedFrameKeeps8BitBytePlanes(t *testing.T) {
	sps := simpleDecodeFrameStorageTestSPS(8, 8, 1)

	frame, _, err := newSimpleDecodedFrame(sps)
	if err != nil {
		t.Fatalf("new 8-bit frame failed: %v", err)
	}
	if len(frame.Y16) != 0 || len(frame.Cb16) != 0 || len(frame.Cr16) != 0 {
		t.Fatalf("high planes allocated for 8-bit frame")
	}
	if len(frame.Y) != frame.LumaStride*frame.MBHeight*16 {
		t.Fatalf("Y len = %d, want %d", len(frame.Y), frame.LumaStride*frame.MBHeight*16)
	}
	chromaWidth, chromaHeight := h264ChromaFrameSize(frame.MBWidth, frame.MBHeight, frame.ChromaFormatIDC)
	if frame.ChromaStride != chromaWidth || len(frame.Cb) != chromaWidth*chromaHeight || len(frame.Cr) != chromaWidth*chromaHeight {
		t.Fatalf("8-bit chroma geometry stride/len = %d/%d/%d, want %d/%d/%d",
			frame.ChromaStride, len(frame.Cb), len(frame.Cr),
			chromaWidth, chromaWidth*chromaHeight, chromaWidth*chromaHeight)
	}
}

func TestNewSimpleDecodedFrameRejectsUnsupportedHighBitDepths(t *testing.T) {
	for _, tt := range []struct {
		name        string
		lumaDepth   int32
		chromaDepth int32
	}{
		{name: "11-bit", lumaDepth: 11, chromaDepth: 11},
		{name: "13-bit", lumaDepth: 13, chromaDepth: 13},
		{name: "unequal-high", lumaDepth: 10, chromaDepth: 12},
		{name: "unequal-8-high", lumaDepth: 8, chromaDepth: 10},
	} {
		t.Run(tt.name, func(t *testing.T) {
			sps := simpleDecodeFrameStorageTestSPS(tt.lumaDepth, tt.chromaDepth, 1)

			if _, _, err := newSimpleDecodedFrame(sps); err != ErrUnsupported {
				t.Fatalf("new frame error = %v, want ErrUnsupported", err)
			}
		})
	}
}

func TestNewSimpleDecodedFrameAllowsFrameMBAFFStorage(t *testing.T) {
	sps := simpleDecodeFrameStorageTestSPS(8, 8, 1)
	sps.FrameMBSOnlyFlag = 0
	sps.MBAFF = 1

	frame, tables, err := newSimpleDecodedFrame(sps)
	if err != nil {
		t.Fatalf("new frame-MBAFF frame failed: %v", err)
	}
	if frame.MBWidth != int(sps.MBWidth) || frame.MBHeight != int(sps.MBHeight) || tables.MBHeight != int(sps.MBHeight) {
		t.Fatalf("MBAFF geometry frame=%dx%d tables=%dx%d want %dx%d",
			frame.MBWidth, frame.MBHeight, tables.MBWidth, tables.MBHeight, sps.MBWidth, sps.MBHeight)
	}
	if len(frame.Y) != frame.LumaStride*frame.MBHeight*16 {
		t.Fatalf("MBAFF Y len = %d, want %d", len(frame.Y), frame.LumaStride*frame.MBHeight*16)
	}
}

func TestNewSimpleDecodedFrameAllowsFieldCapableStorage(t *testing.T) {
	sps := simpleDecodeFrameStorageTestSPS(10, 10, 1)
	sps.FrameMBSOnlyFlag = 0
	sps.MBAFF = 0

	frame, tables, err := newSimpleDecodedFrame(sps)
	if err != nil {
		t.Fatalf("new field-capable frame failed: %v", err)
	}
	if frame.MBWidth != int(sps.MBWidth) || frame.MBHeight != int(sps.MBHeight) || tables.MBHeight != int(sps.MBHeight) {
		t.Fatalf("field-capable geometry frame=%dx%d tables=%dx%d want %dx%d",
			frame.MBWidth, frame.MBHeight, tables.MBWidth, tables.MBHeight, sps.MBWidth, sps.MBHeight)
	}
	if len(frame.Y16) != frame.LumaStride*frame.MBHeight*16 {
		t.Fatalf("field-capable Y16 len = %d, want %d", len(frame.Y16), frame.LumaStride*frame.MBHeight*16)
	}
}

func TestNewSimpleDecodedFrameRejectsInvalidMBAFFPictures(t *testing.T) {
	sps := simpleDecodeFrameStorageTestSPS(10, 10, 1)
	sps.FrameMBSOnlyFlag = 1
	sps.MBAFF = 1

	if _, _, err := newSimpleDecodedFrame(sps); err != ErrUnsupported {
		t.Fatalf("new frame error = %v, want ErrUnsupported", err)
	}
}

func TestNewSimpleDecodedFrameHighChromaGeometry(t *testing.T) {
	for _, tt := range []struct {
		name            string
		chromaFormatIDC uint32
		chromaWidth     int
		chromaHeight    int
	}{
		{name: "monochrome", chromaFormatIDC: 0},
		{name: "420", chromaFormatIDC: 1, chromaWidth: 24, chromaHeight: 16},
		{name: "422", chromaFormatIDC: 2, chromaWidth: 24, chromaHeight: 32},
		{name: "444", chromaFormatIDC: 3, chromaWidth: 48, chromaHeight: 32},
	} {
		t.Run(tt.name, func(t *testing.T) {
			sps := simpleDecodeFrameStorageTestSPS(10, 10, tt.chromaFormatIDC)

			frame, _, err := newSimpleDecodedFrame(sps)
			if err != nil {
				t.Fatalf("new high frame failed: %v", err)
			}
			if frame.LumaStride != 48 || len(frame.Y16) != 48*32 {
				t.Fatalf("luma stride/len = %d/%d, want 48/%d", frame.LumaStride, len(frame.Y16), 48*32)
			}
			if frame.ChromaStride != tt.chromaWidth || len(frame.Cb16) != tt.chromaWidth*tt.chromaHeight || len(frame.Cr16) != tt.chromaWidth*tt.chromaHeight {
				t.Fatalf("chroma stride/len = %d/%d/%d, want %d/%d/%d",
					frame.ChromaStride, len(frame.Cb16), len(frame.Cr16),
					tt.chromaWidth, tt.chromaWidth*tt.chromaHeight, tt.chromaWidth*tt.chromaHeight)
			}
		})
	}
}

func TestDecodedFramePicturePlanesHighValidation(t *testing.T) {
	sps := simpleDecodeFrameStorageTestSPS(12, 12, 2)
	frame, _, err := newSimpleDecodedFrame(sps)
	if err != nil {
		t.Fatalf("new high frame failed: %v", err)
	}

	pic := frame.picturePlanesHigh()
	if err := pic.validate(); err != nil {
		t.Fatalf("high picture planes failed validation: %v", err)
	}

	frame.Cr16 = frame.Cr16[:len(frame.Cr16)-1]
	pic = frame.picturePlanesHigh()
	if err := pic.validate(); err != ErrInvalidData {
		t.Fatalf("truncated high picture planes error = %v, want ErrInvalidData", err)
	}
}

func TestNewH264MotionCompScratchHighForFrame(t *testing.T) {
	sps := simpleDecodeFrameStorageTestSPS(10, 10, 3)
	frame, _, err := newSimpleDecodedFrame(sps)
	if err != nil {
		t.Fatalf("new high frame failed: %v", err)
	}
	pic := frame.picturePlanesHigh()

	scratch := newH264MotionCompScratchHighForFrame(frame)
	if scratch == nil {
		t.Fatal("nil high motion scratch")
	}
	if len(scratch.Y) != 16*frame.LumaStride || len(scratch.Cb) != len(frame.Cb16) || len(scratch.Cr) != len(frame.Cr16) {
		t.Fatalf("scratch plane lens = %d/%d/%d, want %d/%d/%d",
			len(scratch.Y), len(scratch.Cb), len(scratch.Cr),
			16*frame.LumaStride, len(frame.Cb16), len(frame.Cr16))
	}
	if len(scratch.Edge) != h264EdgeScratchSize(frame.ChromaStride, 16+5, 16+5) {
		t.Fatalf("scratch edge len = %d, want %d", len(scratch.Edge), h264EdgeScratchSize(frame.ChromaStride, 16+5, 16+5))
	}
	if !scratch.valid(&pic, 16, 16, 16, 16) {
		t.Fatalf("high scratch does not validate against high picture planes")
	}
}

func TestNewH264MotionCompScratchForFieldCapableFrameUsesFieldStrides(t *testing.T) {
	sps := simpleDecodeFrameStorageTestSPS(8, 8, 1)
	sps.FrameMBSOnlyFlag = 0
	frame, _, err := newSimpleDecodedFrame(sps)
	if err != nil {
		t.Fatalf("new field-capable frame failed: %v", err)
	}
	field := frame.picturePlanes()
	applySimpleFieldRefPlane(&field, PictureBottomField)

	scratch := newH264MotionCompScratchForFrame(frame)
	if scratch == nil {
		t.Fatal("nil motion scratch")
	}
	wantEdge := h264EdgeScratchSize(frame.LumaStride*2, 16+5, 16+5)
	if len(scratch.Y) != 16*frame.LumaStride*2 || len(scratch.Edge) != wantEdge {
		t.Fatalf("field-capable scratch Y/edge lens = %d/%d, want %d/%d",
			len(scratch.Y), len(scratch.Edge), 16*frame.LumaStride*2, wantEdge)
	}
	if !scratch.valid(&field, 16, 16, 8, 8) {
		t.Fatalf("scratch does not validate against 8-bit bottom-field picture view")
	}
}

func TestNewH264MotionCompScratchHighForFieldCapableFrameUsesFieldStrides(t *testing.T) {
	sps := simpleDecodeFrameStorageTestSPS(10, 10, 1)
	sps.FrameMBSOnlyFlag = 0
	frame, _, err := newSimpleDecodedFrame(sps)
	if err != nil {
		t.Fatalf("new high field-capable frame failed: %v", err)
	}
	field := frame.picturePlanesHigh()
	applySimpleFieldRefPlaneHigh(&field, PictureBottomField)

	scratch := newH264MotionCompScratchHighForFrame(frame)
	if scratch == nil {
		t.Fatal("nil high motion scratch")
	}
	wantEdge := h264EdgeScratchSize(frame.LumaStride*2, 16+5, 16+5)
	if len(scratch.Y) != 16*frame.LumaStride*2 || len(scratch.Edge) != wantEdge {
		t.Fatalf("high field-capable scratch Y/edge lens = %d/%d, want %d/%d",
			len(scratch.Y), len(scratch.Edge), 16*frame.LumaStride*2, wantEdge)
	}
	if !scratch.valid(&field, 16, 16, 8, 8) {
		t.Fatalf("high scratch does not validate against bottom-field picture view")
	}
}

func simpleDecodeFrameStorageTestSPS(lumaDepth int32, chromaDepth int32, chromaFormatIDC uint32) *SPS {
	const (
		mbWidth  = 3
		mbHeight = 2
	)
	return &SPS{
		MBWidth:          mbWidth,
		MBHeight:         mbHeight,
		Width:            mbWidth * 16,
		Height:           mbHeight * 16,
		FrameMBSOnlyFlag: 1,
		ChromaFormatIDC:  chromaFormatIDC,
		BitDepthLuma:     lumaDepth,
		BitDepthChroma:   chromaDepth,
	}
}
