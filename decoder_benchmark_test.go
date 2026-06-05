// SPDX-License-Identifier: LGPL-2.1-or-later

package goh264

import (
	"os"
	"path/filepath"
	"testing"
)

const (
	benchmarkHigh10IDRPFrames   = 2
	benchmarkHigh10IDRPRawBytes = 1536
)

var (
	benchmarkDecodeFramesSink int
	benchmarkDecodeBytesSink  int
	benchmarkDecodeRawSink    []byte
)

func benchmarkAnnexBFixture(b *testing.B) []byte {
	b.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "h264", "high10_inter_cavlc_idrp.h264"))
	if err != nil {
		b.Fatal(err)
	}
	return data
}

func BenchmarkDecodeAnnexBHigh10IDRP(b *testing.B) {
	data := benchmarkAnnexBFixture(b)
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()

	var frames []*Frame
	for i := 0; i < b.N; i++ {
		var err error
		frames, err = NewDecoder().DecodeAnnexBFrames(data)
		if err != nil {
			b.Fatal(err)
		}
		if len(frames) != benchmarkHigh10IDRPFrames {
			b.Fatalf("frames = %d, want %d", len(frames), benchmarkHigh10IDRPFrames)
		}
	}
	benchmarkDecodeFramesSink = len(frames)
}

func BenchmarkDecodeAnnexBHigh10IDRPRawYUV(b *testing.B) {
	data := benchmarkAnnexBFixture(b)
	raw := make([]byte, 0, benchmarkHigh10IDRPRawBytes)
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		frames, err := NewDecoder().DecodeAnnexBFrames(data)
		if err != nil {
			b.Fatal(err)
		}
		if len(frames) != benchmarkHigh10IDRPFrames {
			b.Fatalf("frames = %d, want %d", len(frames), benchmarkHigh10IDRPFrames)
		}
		raw = raw[:0]
		for _, frame := range frames {
			raw, err = frame.AppendRawYUVBytesLE(raw)
			if err != nil {
				b.Fatal(err)
			}
		}
		if len(raw) != benchmarkHigh10IDRPRawBytes {
			b.Fatalf("raw bytes = %d, want %d", len(raw), benchmarkHigh10IDRPRawBytes)
		}
	}
	benchmarkDecodeBytesSink = len(raw)
	benchmarkDecodeRawSink = raw
}
