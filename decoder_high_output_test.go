// SPDX-License-Identifier: LGPL-2.1-or-later

package goh264_test

import "testing"

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

func TestFrameHighOutputRejectsWrongSurface(t *testing.T) {
	high := Frame{
		Width: 2, Height: 2, ChromaFormatIDC: 0,
		BitDepthLuma: 10, YStride: 2,
		Y16: []uint16{1, 2, 3, 4},
	}
	if _, err := high.AppendRawYUV(nil); err != ErrUnsupported {
		t.Fatalf("AppendRawYUV high error = %v, want ErrUnsupported", err)
	}

	eight := Frame{
		Width: 2, Height: 2, ChromaFormatIDC: 0,
		BitDepthLuma: 8, YStride: 2,
		Y: []byte{1, 2, 3, 4},
	}
	if _, err := eight.AppendRawYUV16(nil); err != ErrUnsupported {
		t.Fatalf("AppendRawYUV16 8-bit error = %v, want ErrUnsupported", err)
	}
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
