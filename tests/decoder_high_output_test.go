// SPDX-License-Identifier: LGPL-2.1-or-later

package goh264_test

import (
	"bytes"
	"errors"
	"testing"
	"unsafe"
)

const maxIntForTest = int(^uint(0) >> 1)

var rawOutputAllocationByteSink []byte
var rawOutputAllocationUint16Sink []uint16

func TestFrameRawPixelFormatAndSize(t *testing.T) {
	tests := []struct {
		name        string
		frame       Frame
		wantFormat  string
		wantBytes   int
		wantBPS     int
		wantSizeErr error
	}{
		{
			name: "yuv420p10le",
			frame: Frame{
				Width: 4, Height: 4, ChromaFormatIDC: 1,
				BitDepthLuma: 10, BitDepthChroma: 10,
			},
			wantFormat: "yuv420p10le",
			wantBytes:  48,
			wantBPS:    2,
		},
		{
			name: "monochrome-yuv420p12le",
			frame: Frame{
				Width: 3, Height: 2, ChromaFormatIDC: 0,
				BitDepthLuma: 12,
			},
			wantFormat: "yuv420p12le",
			wantBytes:  20,
			wantBPS:    2,
		},
		{
			name: "yuv444p",
			frame: Frame{
				Width: 2, Height: 3, ChromaFormatIDC: 3,
				BitDepthLuma: 8, BitDepthChroma: 8,
			},
			wantFormat: "yuv444p",
			wantBytes:  18,
			wantBPS:    1,
		},
		{
			name: "full-range-yuv420p",
			frame: Frame{
				Width: 2, Height: 2, ChromaFormatIDC: 1,
				BitDepthLuma: 8, BitDepthChroma: 8,
				VideoFullRangeFlag: 1,
			},
			wantFormat: "yuvj420p",
			wantBytes:  6,
			wantBPS:    1,
		},
		{
			name: "full-range-yuv422p",
			frame: Frame{
				Width: 2, Height: 2, ChromaFormatIDC: 2,
				BitDepthLuma: 8, BitDepthChroma: 8,
				VideoFullRangeFlag: 1,
			},
			wantFormat: "yuvj422p",
			wantBytes:  8,
			wantBPS:    1,
		},
		{
			name: "full-range-yuv444p",
			frame: Frame{
				Width: 2, Height: 2, ChromaFormatIDC: 3,
				BitDepthLuma: 8, BitDepthChroma: 8,
				VideoFullRangeFlag: 1,
			},
			wantFormat: "yuvj444p",
			wantBytes:  12,
			wantBPS:    1,
		},
		{
			name: "full-range-yuv420p10le",
			frame: Frame{
				Width: 2, Height: 2, ChromaFormatIDC: 1,
				BitDepthLuma: 10, BitDepthChroma: 10,
				VideoFullRangeFlag: 1,
			},
			wantFormat: "yuv420p10le",
			wantBytes:  12,
			wantBPS:    2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotFormat, err := tt.frame.RawPixelFormat()
			if err != nil {
				t.Fatalf("RawPixelFormat error = %v", err)
			}
			if gotFormat != tt.wantFormat {
				t.Fatalf("RawPixelFormat = %q, want %q", gotFormat, tt.wantFormat)
			}
			gotBPS, err := tt.frame.BytesPerSample()
			if err != nil {
				t.Fatalf("BytesPerSample error = %v", err)
			}
			if gotBPS != tt.wantBPS {
				t.Fatalf("BytesPerSample = %d, want %d", gotBPS, tt.wantBPS)
			}
			gotBytes, err := tt.frame.RawYUVSize()
			if err != tt.wantSizeErr {
				t.Fatalf("RawYUVSize error = %v, want %v", err, tt.wantSizeErr)
			}
			if gotBytes != tt.wantBytes {
				t.Fatalf("RawYUVSize = %d, want %d", gotBytes, tt.wantBytes)
			}
		})
	}
}

func TestFrameRawOutputRejectsNilFrame(t *testing.T) {
	var frame *Frame

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("nil frame raw-output helper panicked: %v", r)
		}
	}()

	if got, err := frame.BytesPerSample(); got != 0 || !errors.Is(err, ErrInvalidData) {
		t.Fatalf("BytesPerSample nil frame = (%d, %v), want (0, ErrInvalidData)", got, err)
	}
	if got, err := frame.RawPixelFormat(); got != "" || !errors.Is(err, ErrInvalidData) {
		t.Fatalf("RawPixelFormat nil frame = (%q, %v), want empty format and ErrInvalidData", got, err)
	}
	if got, err := frame.RawYUVSize(); got != 0 || !errors.Is(err, ErrInvalidData) {
		t.Fatalf("RawYUVSize nil frame = (%d, %v), want (0, ErrInvalidData)", got, err)
	}

	byteDst, byteBefore := decoderPrefilledByteBuffer()
	if got, err := frame.AppendRawYUV(byteDst); len(got) != len(byteDst) || !errors.Is(err, ErrInvalidData) {
		t.Fatalf("AppendRawYUV nil frame got len=%d err=%v, want original buffer and ErrInvalidData", len(got), err)
	}
	assertDecoderByteBufferUnchanged(t, byteDst, byteBefore)

	byteLEDst, byteLEBefore := decoderPrefilledByteBuffer()
	if got, err := frame.AppendRawYUVBytesLE(byteLEDst); len(got) != len(byteLEDst) || !errors.Is(err, ErrInvalidData) {
		t.Fatalf("AppendRawYUVBytesLE nil frame got len=%d err=%v, want original buffer and ErrInvalidData", len(got), err)
	}
	assertDecoderByteBufferUnchanged(t, byteLEDst, byteLEBefore)

	uint16Dst, uint16Before := decoderPrefilledUint16Buffer()
	if got, err := frame.AppendRawYUV16(uint16Dst); len(got) != len(uint16Dst) || !errors.Is(err, ErrInvalidData) {
		t.Fatalf("AppendRawYUV16 nil frame got len=%d err=%v, want original buffer and ErrInvalidData", len(got), err)
	}
	assertDecoderUint16BufferUnchanged(t, uint16Dst, uint16Before)
}

func TestFrameRawOutputClassifiesInvalidMetadata(t *testing.T) {
	tests := []struct {
		name  string
		frame Frame
		want  error
	}{
		{
			name: "invalid-chroma-format",
			frame: Frame{
				Width: 2, Height: 2, ChromaFormatIDC: 4,
				BitDepthLuma: 8, BitDepthChroma: 8,
			},
			want: ErrInvalidData,
		},
		{
			name: "nonpositive-luma-depth",
			frame: Frame{
				Width: 2, Height: 2, ChromaFormatIDC: 1,
				BitDepthLuma: 0, BitDepthChroma: 8,
			},
			want: ErrInvalidData,
		},
		{
			name: "unsupported-luma-depth",
			frame: Frame{
				Width: 2, Height: 2, ChromaFormatIDC: 1,
				BitDepthLuma: 11, BitDepthChroma: 11,
			},
			want: ErrUnsupported,
		},
		{
			name: "nonpositive-chroma-depth",
			frame: Frame{
				Width: 2, Height: 2, ChromaFormatIDC: 1,
				BitDepthLuma: 8, BitDepthChroma: 0,
			},
			want: ErrInvalidData,
		},
		{
			name: "mismatched-chroma-depth",
			frame: Frame{
				Width: 2, Height: 2, ChromaFormatIDC: 1,
				BitDepthLuma: 8, BitDepthChroma: 10,
			},
			want: ErrUnsupported,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := tt.frame.BytesPerSample(); !errors.Is(err, tt.want) {
				t.Fatalf("BytesPerSample error = %v, want %v", err, tt.want)
			}
			if _, err := tt.frame.RawPixelFormat(); !errors.Is(err, tt.want) {
				t.Fatalf("RawPixelFormat error = %v, want %v", err, tt.want)
			}
			if _, err := tt.frame.RawYUVSize(); !errors.Is(err, tt.want) {
				t.Fatalf("RawYUVSize error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestFrameRawYUVSizeRejectsNonpositiveDimensions(t *testing.T) {
	tests := []struct {
		name  string
		frame Frame
	}{
		{
			name: "zero-width-8-bit",
			frame: Frame{
				Width: 0, Height: 2, ChromaFormatIDC: 1,
				BitDepthLuma: 8, BitDepthChroma: 8,
			},
		},
		{
			name: "negative-height-8-bit",
			frame: Frame{
				Width: 2, Height: -1, ChromaFormatIDC: 1,
				BitDepthLuma: 8, BitDepthChroma: 8,
			},
		},
		{
			name: "zero-width-high-bit-depth",
			frame: Frame{
				Width: 0, Height: 2, ChromaFormatIDC: 1,
				BitDepthLuma: 10, BitDepthChroma: 10,
			},
		},
		{
			name: "negative-height-high-bit-depth",
			frame: Frame{
				Width: 2, Height: -1, ChromaFormatIDC: 1,
				BitDepthLuma: 10, BitDepthChroma: 10,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.frame.RawYUVSize()
			if got != 0 || !errors.Is(err, ErrInvalidData) {
				t.Fatalf("RawYUVSize = (%d, %v), want (0, ErrInvalidData)", got, err)
			}
		})
	}
}

func TestFrameAppendRawYUV16AndBytesLEPreserveSamplesAndCrop(t *testing.T) {
	frame := Frame{
		Width:           4,
		Height:          4,
		CropLeft:        2,
		CropTop:         1,
		ChromaFormatIDC: 1,
		BitDepthLuma:    10,
		BitDepthChroma:  10,
		YStride:         8,
		CStride:         5,
		Y16:             make([]uint16, 5*8),
		Cb16:            make([]uint16, 2*5),
		Cr16:            make([]uint16, 2*5),
	}
	fillUint16Ramp(frame.Y16, 100)
	fillUint16Ramp(frame.Cb16, 500)
	fillUint16Ramp(frame.Cr16, 800)

	want := make([]uint16, 0, 24)
	for y := 0; y < frame.Height; y++ {
		row := (frame.CropTop+y)*frame.YStride + frame.CropLeft
		want = append(want, frame.Y16[row:row+frame.Width]...)
	}
	for y := 0; y < 2; y++ {
		row := y*frame.CStride + 1
		want = append(want, frame.Cb16[row:row+2]...)
	}
	for y := 0; y < 2; y++ {
		row := y*frame.CStride + 1
		want = append(want, frame.Cr16[row:row+2]...)
	}

	got16, err := frame.AppendRawYUV16(nil)
	if err != nil {
		t.Fatalf("AppendRawYUV16 error = %v", err)
	}
	if !equalUint16Slices(got16, want) {
		t.Fatalf("AppendRawYUV16 = %v, want %v", got16, want)
	}

	gotLE, err := frame.AppendRawYUVBytesLE(nil)
	if err != nil {
		t.Fatalf("AppendRawYUVBytesLE error = %v", err)
	}
	wantLE := rawUint16LE(want)
	if string(gotLE) != string(wantLE) {
		t.Fatalf("AppendRawYUVBytesLE = %v, want %v", gotLE, wantLE)
	}
	if len(gotLE) != 48 {
		t.Fatalf("AppendRawYUVBytesLE len = %d, want 48", len(gotLE))
	}
}

func TestFrameAppendRawYUVBytesLEUses8BitByteSurface(t *testing.T) {
	frame := Frame{
		Width:           2,
		Height:          2,
		ChromaFormatIDC: 1,
		BitDepthLuma:    8,
		BitDepthChroma:  8,
		YStride:         2,
		CStride:         1,
		Y:               []byte{1, 2, 3, 4},
		Cb:              []byte{5},
		Cr:              []byte{6},
	}

	got, err := frame.AppendRawYUVBytesLE(nil)
	if err != nil {
		t.Fatalf("AppendRawYUVBytesLE error = %v", err)
	}
	if string(got) != string([]byte{1, 2, 3, 4, 5, 6}) {
		t.Fatalf("AppendRawYUVBytesLE = %v, want [1 2 3 4 5 6]", got)
	}
}

func TestFrameAppendRawYUVBytesLEExpandsMonochromeToYUV420(t *testing.T) {
	frame := Frame{
		Width:           4,
		Height:          2,
		ChromaFormatIDC: 0,
		BitDepthLuma:    10,
		YStride:         4,
		Y16:             []uint16{1, 2, 3, 4, 5, 6, 7, 8},
	}

	got, err := frame.AppendRawYUVBytesLE(nil)
	if err != nil {
		t.Fatalf("AppendRawYUVBytesLE error = %v", err)
	}
	wantSamples := []uint16{1, 2, 3, 4, 5, 6, 7, 8, 512, 512, 512, 512}
	want := rawUint16LE(wantSamples)
	if string(got) != string(want) {
		t.Fatalf("AppendRawYUVBytesLE = %v, want %v", got, want)
	}
}

func TestFrameAppendRawYUVExpands8BitMonochromeWithoutChromaDepth(t *testing.T) {
	frame := Frame{
		Width:           2,
		Height:          2,
		ChromaFormatIDC: 0,
		BitDepthLuma:    8,
		YStride:         2,
		Y:               []byte{1, 2, 3, 4},
	}

	got, err := frame.AppendRawYUV(nil)
	if err != nil {
		t.Fatalf("AppendRawYUV error = %v", err)
	}
	if string(got) != string([]byte{1, 2, 3, 4, 128, 128}) {
		t.Fatalf("AppendRawYUV = %v, want [1 2 3 4 128 128]", got)
	}
}

func TestFrameAppendRawYUVUsesCallerBufferWithoutAllocation(t *testing.T) {
	frame := Frame{
		Width:           2,
		Height:          2,
		ChromaFormatIDC: 1,
		BitDepthLuma:    8,
		BitDepthChroma:  8,
		YStride:         2,
		CStride:         1,
		Y:               []byte{1, 2, 3, 4},
		Cb:              []byte{5},
		Cr:              []byte{6},
	}
	wantSize, err := frame.RawYUVSize()
	if err != nil {
		t.Fatalf("RawYUVSize: %v", err)
	}
	buf := make([]byte, 0, wantSize)
	out, err := frame.AppendRawYUV(buf)
	if err != nil {
		t.Fatalf("AppendRawYUV: %v", err)
	}
	if len(out) != wantSize || len(buf) != 0 || cap(out) != cap(buf) || &out[0] != &buf[:cap(buf)][0] {
		t.Fatalf("AppendRawYUV caller buffer len/cap/pointer = %d/%d/%t, want len %d cap %d original backing",
			len(out), cap(out), &out[0] == &buf[:cap(buf)][0], wantSize, cap(buf))
	}

	allocs := testing.AllocsPerRun(100, func() {
		out, err := frame.AppendRawYUV(buf[:0])
		if err != nil {
			t.Fatalf("AppendRawYUV: %v", err)
		}
		if len(out) != wantSize {
			t.Fatalf("AppendRawYUV len = %d, want %d", len(out), wantSize)
		}
		rawOutputAllocationByteSink = out
	})
	if allocs != 0 {
		t.Fatalf("AppendRawYUV allocations/run = %.0f, want 0 with caller-owned buffer", allocs)
	}
}

func TestFrameAppendRawYUVBytesLEUsesCallerBufferWithoutAllocation(t *testing.T) {
	tests := []struct {
		name  string
		frame Frame
	}{
		{
			name: "8-bit",
			frame: Frame{
				Width:           2,
				Height:          2,
				ChromaFormatIDC: 1,
				BitDepthLuma:    8,
				BitDepthChroma:  8,
				YStride:         2,
				CStride:         1,
				Y:               []byte{1, 2, 3, 4},
				Cb:              []byte{5},
				Cr:              []byte{6},
			},
		},
		{
			name: "10-bit-cropped",
			frame: Frame{
				Width:           4,
				Height:          4,
				CropLeft:        2,
				CropTop:         1,
				ChromaFormatIDC: 1,
				BitDepthLuma:    10,
				BitDepthChroma:  10,
				YStride:         8,
				CStride:         5,
				Y16:             make([]uint16, 5*8),
				Cb16:            make([]uint16, 2*5),
				Cr16:            make([]uint16, 2*5),
			},
		},
	}
	fillUint16Ramp(tests[1].frame.Y16, 100)
	fillUint16Ramp(tests[1].frame.Cb16, 500)
	fillUint16Ramp(tests[1].frame.Cr16, 800)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			frame := tt.frame
			wantSize, err := frame.RawYUVSize()
			if err != nil {
				t.Fatalf("RawYUVSize: %v", err)
			}
			buf := make([]byte, 0, wantSize)
			out, err := frame.AppendRawYUVBytesLE(buf)
			if err != nil {
				t.Fatalf("AppendRawYUVBytesLE: %v", err)
			}
			if len(out) != wantSize || len(buf) != 0 || cap(out) != cap(buf) || &out[0] != &buf[:cap(buf)][0] {
				t.Fatalf("AppendRawYUVBytesLE caller buffer len/cap/pointer = %d/%d/%t, want len %d cap %d original backing",
					len(out), cap(out), &out[0] == &buf[:cap(buf)][0], wantSize, cap(buf))
			}

			allocs := testing.AllocsPerRun(100, func() {
				out, err := frame.AppendRawYUVBytesLE(buf[:0])
				if err != nil {
					t.Fatalf("AppendRawYUVBytesLE: %v", err)
				}
				if len(out) != wantSize {
					t.Fatalf("AppendRawYUVBytesLE len = %d, want %d", len(out), wantSize)
				}
				rawOutputAllocationByteSink = out
			})
			if allocs != 0 {
				t.Fatalf("AppendRawYUVBytesLE allocations/run = %.0f, want 0 with caller-owned buffer", allocs)
			}
		})
	}
}

func TestFrameAppendRawYUV16UsesCallerBufferWithoutAllocation(t *testing.T) {
	frame := Frame{
		Width:           4,
		Height:          4,
		CropLeft:        2,
		CropTop:         1,
		ChromaFormatIDC: 1,
		BitDepthLuma:    10,
		BitDepthChroma:  10,
		YStride:         8,
		CStride:         5,
		Y16:             make([]uint16, 5*8),
		Cb16:            make([]uint16, 2*5),
		Cr16:            make([]uint16, 2*5),
	}
	fillUint16Ramp(frame.Y16, 100)
	fillUint16Ramp(frame.Cb16, 500)
	fillUint16Ramp(frame.Cr16, 800)
	wantBytes, err := frame.RawYUVSize()
	if err != nil {
		t.Fatalf("RawYUVSize: %v", err)
	}
	wantSamples := wantBytes / 2
	buf := make([]uint16, 0, wantSamples)
	out, err := frame.AppendRawYUV16(buf)
	if err != nil {
		t.Fatalf("AppendRawYUV16: %v", err)
	}
	if len(out) != wantSamples || len(buf) != 0 || cap(out) != cap(buf) || &out[0] != &buf[:cap(buf)][0] {
		t.Fatalf("AppendRawYUV16 caller buffer len/cap/pointer = %d/%d/%t, want len %d cap %d original backing",
			len(out), cap(out), &out[0] == &buf[:cap(buf)][0], wantSamples, cap(buf))
	}

	allocs := testing.AllocsPerRun(100, func() {
		out, err := frame.AppendRawYUV16(buf[:0])
		if err != nil {
			t.Fatalf("AppendRawYUV16: %v", err)
		}
		if len(out) != wantSamples {
			t.Fatalf("AppendRawYUV16 len = %d, want %d", len(out), wantSamples)
		}
		rawOutputAllocationUint16Sink = out
	})
	if allocs != 0 {
		t.Fatalf("AppendRawYUV16 allocations/run = %.0f, want 0 with caller-owned buffer", allocs)
	}
}

func TestFrameHighOutputRejectsWrongSurface(t *testing.T) {
	high := Frame{
		Width: 2, Height: 2, ChromaFormatIDC: 0,
		BitDepthLuma: 10, YStride: 2,
		Y16: []uint16{1, 2, 3, 4},
	}
	if _, err := high.AppendRawYUV(nil); err != ErrUnsupported {
		t.Fatalf("AppendRawYUV high error = %v, want ErrUnsupported", err)
	}
	byteDst, byteBefore := decoderPrefilledByteBuffer()
	if got, err := high.AppendRawYUV(byteDst); err != ErrUnsupported || len(got) != len(byteDst) {
		t.Fatalf("AppendRawYUV high got len=%d err=%v, want original buffer and ErrUnsupported", len(got), err)
	}
	assertDecoderByteBufferUnchanged(t, byteDst, byteBefore)

	eight := Frame{
		Width: 2, Height: 2, ChromaFormatIDC: 0,
		BitDepthLuma: 8, YStride: 2,
		Y: []byte{1, 2, 3, 4},
	}
	if _, err := eight.AppendRawYUV16(nil); err != ErrUnsupported {
		t.Fatalf("AppendRawYUV16 8-bit error = %v, want ErrUnsupported", err)
	}
	uint16Dst, uint16Before := decoderPrefilledUint16Buffer()
	if got, err := eight.AppendRawYUV16(uint16Dst); err != ErrUnsupported || len(got) != len(uint16Dst) {
		t.Fatalf("AppendRawYUV16 8-bit got len=%d err=%v, want original buffer and ErrUnsupported", len(got), err)
	}
	assertDecoderUint16BufferUnchanged(t, uint16Dst, uint16Before)
}

func TestFrameHighOutputRejectsInvalidGeometryAndDepth(t *testing.T) {
	badGeometry := Frame{
		Width: 4, Height: 2, ChromaFormatIDC: 0,
		BitDepthLuma: 12, YStride: 4,
		Y16: []uint16{1, 2, 3},
	}
	if _, err := badGeometry.AppendRawYUVBytesLE(nil); err != ErrInvalidData {
		t.Fatalf("bad geometry error = %v, want ErrInvalidData", err)
	}

	badDepth := Frame{
		Width: 2, Height: 2, ChromaFormatIDC: 0,
		BitDepthLuma: 11, YStride: 2,
		Y16: []uint16{1, 2, 3, 4},
	}
	if _, err := badDepth.RawPixelFormat(); err != ErrUnsupported {
		t.Fatalf("bad depth error = %v, want ErrUnsupported", err)
	}

	badSample := Frame{
		Width: 2, Height: 2, ChromaFormatIDC: 0,
		BitDepthLuma: 10, YStride: 2,
		Y16: []uint16{1, 2, 3, 1024},
	}
	if _, err := badSample.AppendRawYUV16(nil); err != ErrInvalidData {
		t.Fatalf("bad sample AppendRawYUV16 error = %v, want ErrInvalidData", err)
	}
	if _, err := badSample.AppendRawYUVBytesLE(nil); err != ErrInvalidData {
		t.Fatalf("bad sample AppendRawYUVBytesLE error = %v, want ErrInvalidData", err)
	}
}

func TestFrameRawYUVSizeRejectsOverflow(t *testing.T) {
	frame := Frame{
		Width:           maxIntForTest/2 + 1,
		Height:          3,
		ChromaFormatIDC: 0,
		BitDepthLuma:    8,
		BitDepthChroma:  8,
	}
	if _, err := frame.RawYUVSize(); err != ErrInvalidData {
		t.Fatalf("RawYUVSize overflow error = %v, want ErrInvalidData", err)
	}

	for _, chromaFormatIDC := range []uint32{1, 2} {
		frame := Frame{
			Width:           maxIntForTest,
			Height:          1,
			ChromaFormatIDC: chromaFormatIDC,
			BitDepthLuma:    8,
			BitDepthChroma:  8,
		}
		if _, err := frame.RawYUVSize(); err != ErrInvalidData {
			t.Fatalf("RawYUVSize chroma %d overflow error = %v, want ErrInvalidData", chromaFormatIDC, err)
		}
	}
}

func TestFrameAppendRawYUVRejectsOverflowedDestination(t *testing.T) {
	eight := Frame{
		Width:           2,
		Height:          2,
		ChromaFormatIDC: 0,
		BitDepthLuma:    8,
		BitDepthChroma:  8,
		YStride:         2,
		Y:               []byte{1, 2, 3, 4},
	}
	eightDst := fakeDecoderRawBytesLen(maxIntForTest - 5)
	if got, err := eight.AppendRawYUV(eightDst); err != ErrInvalidData || len(got) != len(eightDst) {
		t.Fatalf("AppendRawYUV overflow got len=%d err=%v, want original buffer and ErrInvalidData", len(got), err)
	}
	if got, err := eight.AppendRawYUVBytesLE(eightDst); err != ErrInvalidData || len(got) != len(eightDst) {
		t.Fatalf("AppendRawYUVBytesLE 8-bit overflow got len=%d err=%v, want original buffer and ErrInvalidData", len(got), err)
	}

	high := Frame{
		Width:           2,
		Height:          2,
		ChromaFormatIDC: 0,
		BitDepthLuma:    10,
		BitDepthChroma:  10,
		YStride:         2,
		Y16:             []uint16{1, 2, 3, 4},
	}
	highByteDst := fakeDecoderRawBytesLen(maxIntForTest - 11)
	if got, err := high.AppendRawYUVBytesLE(highByteDst); err != ErrInvalidData || len(got) != len(highByteDst) {
		t.Fatalf("AppendRawYUVBytesLE high overflow got len=%d err=%v, want original buffer and ErrInvalidData", len(got), err)
	}
	highSampleDst := fakeDecoderRawUint16Len(maxIntForTest - 5)
	if got, err := high.AppendRawYUV16(highSampleDst); err != ErrInvalidData || len(got) != len(highSampleDst) {
		t.Fatalf("AppendRawYUV16 overflow got len=%d err=%v, want original buffer and ErrInvalidData", len(got), err)
	}
}

func TestFrameAppendRawYUVRejectsOverflowedPlaneGeometryWithoutPanic(t *testing.T) {
	tests := []struct {
		name string
		call func() error
	}{
		{
			name: "8-bit",
			call: func() error {
				frame := Frame{
					Width:           1,
					Height:          1,
					CropLeft:        maxIntForTest,
					ChromaFormatIDC: 0,
					BitDepthLuma:    8,
					BitDepthChroma:  8,
					YStride:         maxIntForTest,
					Y:               []byte{0},
				}
				_, err := frame.AppendRawYUV(nil)
				return err
			},
		},
		{
			name: "high-bit-depth",
			call: func() error {
				frame := Frame{
					Width:           1,
					Height:          1,
					CropLeft:        maxIntForTest,
					ChromaFormatIDC: 0,
					BitDepthLuma:    10,
					BitDepthChroma:  10,
					YStride:         maxIntForTest,
					Y16:             []uint16{0},
				}
				_, err := frame.AppendRawYUVBytesLE(nil)
				return err
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("%s panicked on overflowed geometry: %v", tt.name, r)
				}
			}()
			if err := tt.call(); err != ErrInvalidData {
				t.Fatalf("%s overflowed geometry error = %v, want ErrInvalidData", tt.name, err)
			}
		})
	}
}

func TestFrameAppendRawYUVErrorPreservesCallerBuffer(t *testing.T) {
	frame := Frame{
		Width:           2,
		Height:          2,
		ChromaFormatIDC: 1,
		BitDepthLuma:    8,
		BitDepthChroma:  8,
		YStride:         2,
		CStride:         1,
		Y:               []byte{1, 2, 3, 4},
		Cb:              nil,
		Cr:              []byte{6},
	}
	dst, before := decoderPrefilledByteBuffer()
	out, err := frame.AppendRawYUV(dst)
	if err != ErrInvalidData {
		t.Fatalf("AppendRawYUV invalid chroma error = %v, want ErrInvalidData", err)
	}
	if len(out) != len(dst) {
		t.Fatalf("AppendRawYUV invalid output len = %d, want original len %d", len(out), len(dst))
	}
	assertDecoderByteBufferUnchanged(t, dst, before)
}

func TestFrameAppendRawYUVHighErrorPreservesCallerBuffer(t *testing.T) {
	tests := []struct {
		name  string
		frame Frame
	}{
		{
			name: "luma",
			frame: Frame{
				Width:           2,
				Height:          2,
				ChromaFormatIDC: 0,
				BitDepthLuma:    10,
				YStride:         2,
				Y16:             []uint16{1, 2, 3, 1024},
			},
		},
		{
			name: "chroma",
			frame: Frame{
				Width:           2,
				Height:          2,
				ChromaFormatIDC: 1,
				BitDepthLuma:    10,
				BitDepthChroma:  10,
				YStride:         2,
				CStride:         1,
				Y16:             []uint16{1, 2, 3, 4},
				Cb16:            []uint16{1024},
				Cr16:            []uint16{512},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Run("uint16", func(t *testing.T) {
				dst, before := decoderPrefilledUint16Buffer()
				out, err := tt.frame.AppendRawYUV16(dst)
				if err != ErrInvalidData {
					t.Fatalf("AppendRawYUV16 bad sample error = %v, want ErrInvalidData", err)
				}
				if len(out) != len(dst) {
					t.Fatalf("AppendRawYUV16 invalid output len = %d, want original len %d", len(out), len(dst))
				}
				assertDecoderUint16BufferUnchanged(t, dst, before)
			})

			t.Run("bytes-le", func(t *testing.T) {
				dst, before := decoderPrefilledByteBuffer()
				out, err := tt.frame.AppendRawYUVBytesLE(dst)
				if err != ErrInvalidData {
					t.Fatalf("AppendRawYUVBytesLE bad sample error = %v, want ErrInvalidData", err)
				}
				if len(out) != len(dst) {
					t.Fatalf("AppendRawYUVBytesLE invalid output len = %d, want original len %d", len(out), len(dst))
				}
				assertDecoderByteBufferUnchanged(t, dst, before)
			})
		})
	}
}

func TestFrameAppendRawYUVMetadataErrorsPreserveCallerBuffer(t *testing.T) {
	tests := []struct {
		name  string
		frame Frame
		want  error
	}{
		{
			name: "invalid-chroma-format",
			frame: Frame{
				Width: 2, Height: 2, ChromaFormatIDC: 4,
				BitDepthLuma: 8, BitDepthChroma: 8,
			},
			want: ErrInvalidData,
		},
		{
			name: "unsupported-luma-depth",
			frame: Frame{
				Width: 2, Height: 2, ChromaFormatIDC: 1,
				BitDepthLuma: 11, BitDepthChroma: 11,
			},
			want: ErrUnsupported,
		},
		{
			name: "mismatched-chroma-depth",
			frame: Frame{
				Width: 2, Height: 2, ChromaFormatIDC: 1,
				BitDepthLuma: 8, BitDepthChroma: 10,
			},
			want: ErrUnsupported,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rawDst, rawBefore := decoderPrefilledByteBuffer()
			rawOut, err := tt.frame.AppendRawYUV(rawDst)
			if !errors.Is(err, tt.want) {
				t.Fatalf("AppendRawYUV error = %v, want %v", err, tt.want)
			}
			if len(rawOut) != len(rawDst) {
				t.Fatalf("AppendRawYUV output len = %d, want original len %d", len(rawOut), len(rawDst))
			}
			assertDecoderByteBufferUnchanged(t, rawDst, rawBefore)

			byteDst, byteBefore := decoderPrefilledByteBuffer()
			byteOut, err := tt.frame.AppendRawYUVBytesLE(byteDst)
			if !errors.Is(err, tt.want) {
				t.Fatalf("AppendRawYUVBytesLE error = %v, want %v", err, tt.want)
			}
			if len(byteOut) != len(byteDst) {
				t.Fatalf("AppendRawYUVBytesLE output len = %d, want original len %d", len(byteOut), len(byteDst))
			}
			assertDecoderByteBufferUnchanged(t, byteDst, byteBefore)

			sampleDst, sampleBefore := decoderPrefilledUint16Buffer()
			sampleOut, err := tt.frame.AppendRawYUV16(sampleDst)
			if !errors.Is(err, tt.want) {
				t.Fatalf("AppendRawYUV16 error = %v, want %v", err, tt.want)
			}
			if len(sampleOut) != len(sampleDst) {
				t.Fatalf("AppendRawYUV16 output len = %d, want original len %d", len(sampleOut), len(sampleDst))
			}
			assertDecoderUint16BufferUnchanged(t, sampleDst, sampleBefore)
		})
	}
}

func TestFrameAppendRawYUVDimensionErrorsPreserveCallerBuffer(t *testing.T) {
	tests := []struct {
		name          string
		frame         Frame
		wantRaw       error
		wantBytesLE   error
		wantSamples16 error
	}{
		{
			name: "zero-width-8-bit",
			frame: Frame{
				Width: 0, Height: 2, ChromaFormatIDC: 1,
				BitDepthLuma: 8, BitDepthChroma: 8,
			},
			wantRaw:       ErrInvalidData,
			wantBytesLE:   ErrInvalidData,
			wantSamples16: ErrUnsupported,
		},
		{
			name: "negative-height-8-bit",
			frame: Frame{
				Width: 2, Height: -1, ChromaFormatIDC: 1,
				BitDepthLuma: 8, BitDepthChroma: 8,
			},
			wantRaw:       ErrInvalidData,
			wantBytesLE:   ErrInvalidData,
			wantSamples16: ErrUnsupported,
		},
		{
			name: "zero-width-high-bit-depth",
			frame: Frame{
				Width: 0, Height: 2, ChromaFormatIDC: 1,
				BitDepthLuma: 10, BitDepthChroma: 10,
			},
			wantRaw:       ErrInvalidData,
			wantBytesLE:   ErrInvalidData,
			wantSamples16: ErrInvalidData,
		},
		{
			name: "negative-height-high-bit-depth",
			frame: Frame{
				Width: 2, Height: -1, ChromaFormatIDC: 1,
				BitDepthLuma: 10, BitDepthChroma: 10,
			},
			wantRaw:       ErrInvalidData,
			wantBytesLE:   ErrInvalidData,
			wantSamples16: ErrInvalidData,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rawDst, rawBefore := decoderPrefilledByteBuffer()
			rawOut, err := tt.frame.AppendRawYUV(rawDst)
			if !errors.Is(err, tt.wantRaw) {
				t.Fatalf("AppendRawYUV error = %v, want %v", err, tt.wantRaw)
			}
			if len(rawOut) != len(rawDst) {
				t.Fatalf("AppendRawYUV output len = %d, want original len %d", len(rawOut), len(rawDst))
			}
			assertDecoderByteBufferUnchanged(t, rawDst, rawBefore)

			byteDst, byteBefore := decoderPrefilledByteBuffer()
			byteOut, err := tt.frame.AppendRawYUVBytesLE(byteDst)
			if !errors.Is(err, tt.wantBytesLE) {
				t.Fatalf("AppendRawYUVBytesLE error = %v, want %v", err, tt.wantBytesLE)
			}
			if len(byteOut) != len(byteDst) {
				t.Fatalf("AppendRawYUVBytesLE output len = %d, want original len %d", len(byteOut), len(byteDst))
			}
			assertDecoderByteBufferUnchanged(t, byteDst, byteBefore)

			sampleDst, sampleBefore := decoderPrefilledUint16Buffer()
			sampleOut, err := tt.frame.AppendRawYUV16(sampleDst)
			if !errors.Is(err, tt.wantSamples16) {
				t.Fatalf("AppendRawYUV16 error = %v, want %v", err, tt.wantSamples16)
			}
			if len(sampleOut) != len(sampleDst) {
				t.Fatalf("AppendRawYUV16 output len = %d, want original len %d", len(sampleOut), len(sampleDst))
			}
			assertDecoderUint16BufferUnchanged(t, sampleDst, sampleBefore)
		})
	}
}

func decoderPrefilledByteBuffer() ([]byte, []byte) {
	backing := bytes.Repeat([]byte{0xcc}, 128)
	prefix := []byte{0xde, 0xad, 0xbe, 0xef}
	copy(backing, prefix)
	return backing[:len(prefix)], append([]byte(nil), backing...)
}

func decoderPrefilledUint16Buffer() ([]uint16, []uint16) {
	backing := make([]uint16, 64)
	for i := range backing {
		backing[i] = 0xcccc
	}
	prefix := []uint16{0xdead, 0xbeef}
	copy(backing, prefix)
	return backing[:len(prefix)], append([]uint16(nil), backing...)
}

func fakeDecoderRawBytesLen(n int) []byte {
	if n <= 0 {
		return nil
	}
	var b byte
	return fakeDecoderRawSliceLen(&b, n)
}

func fakeDecoderRawUint16Len(n int) []uint16 {
	if n <= 0 {
		return nil
	}
	var v uint16
	return fakeDecoderRawSliceLen(&v, n)
}

// fakeDecoderRawSliceLen preserves impossible slice lengths for overflow guards.
func fakeDecoderRawSliceLen[T any](ptr *T, n int) []T {
	h := struct {
		Data unsafe.Pointer
		Len  int
		Cap  int
	}{
		Data: unsafe.Pointer(ptr),
		Len:  n,
		Cap:  n,
	}
	return *(*[]T)(unsafe.Pointer(&h))
}

func assertDecoderByteBufferUnchanged(t *testing.T, dst []byte, before []byte) {
	t.Helper()
	if !bytes.Equal(dst[:cap(dst)], before) {
		t.Fatalf("raw-output helper mutated caller byte buffer on error")
	}
}

func assertDecoderUint16BufferUnchanged(t *testing.T, dst []uint16, before []uint16) {
	t.Helper()
	after := dst[:cap(dst)]
	if len(after) != len(before) {
		t.Fatalf("raw-output helper backing len = %d, want %d", len(after), len(before))
	}
	for i := range after {
		if after[i] != before[i] {
			t.Fatalf("raw-output helper mutated caller uint16 buffer at %d: got %#x want %#x", i, after[i], before[i])
		}
	}
}

func fillUint16Ramp(dst []uint16, start uint16) {
	for i := range dst {
		dst[i] = start + uint16(i)
	}
}

func rawUint16LE(samples []uint16) []byte {
	out := make([]byte, 0, len(samples)*2)
	for _, sample := range samples {
		out = append(out, byte(sample), byte(sample>>8))
	}
	return out
}

func equalUint16Slices(a []uint16, b []uint16) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
