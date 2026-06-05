// SPDX-License-Identifier: LGPL-2.1-or-later

package goh264

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/thesyncim/goh264/internal/h264"
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

func benchmarkAnnexBAccessUnits(b *testing.B, data []byte) [][]byte {
	b.Helper()
	nals, err := h264.SplitAnnexB(data)
	if err != nil {
		b.Fatal(err)
	}

	var spsList [32]*h264.SPS
	var ppsList [256]*h264.PPS
	var units [][]byte
	var unit []byte
	hasVCL := false
	for _, nal := range nals {
		switch nal.Type {
		case h264.NALSPS:
			sps, err := h264.DecodeSPS(nal.RBSP)
			if err != nil {
				b.Fatal(err)
			}
			spsList[sps.SPSID] = sps
		case h264.NALPPS:
			pps, err := h264.DecodePPS(nal.RBSP, &spsList)
			if err != nil {
				b.Fatal(err)
			}
			ppsList[pps.PPSID] = pps
		}

		isVCL := nal.Type == h264.NALSlice || nal.Type == h264.NALIDRSlice
		if isVCL {
			sh, err := h264.ParseSliceHeader(nal, &ppsList)
			if err != nil {
				b.Fatal(err)
			}
			if hasVCL && sh.FirstMBAddr == 0 {
				units = append(units, unit)
				unit = nil
				hasVCL = false
			}
		}
		unit = appendAnnexBUnitNAL(unit, nal.Raw)
		if isVCL {
			hasVCL = true
		}
	}
	if len(unit) != 0 {
		units = append(units, unit)
	}
	if len(units) != benchmarkHigh10IDRPFrames {
		b.Fatalf("access units = %d, want %d", len(units), benchmarkHigh10IDRPFrames)
	}
	return units
}

func appendAnnexBUnitNAL(dst []byte, raw []byte) []byte {
	dst = append(dst, 0x00, 0x00, 0x00, 0x01)
	return append(dst, raw...)
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

func BenchmarkDecodeFramesAnnexBHigh10IDRPAccessUnits(b *testing.B) {
	data := benchmarkAnnexBFixture(b)
	units := benchmarkAnnexBAccessUnits(b, data)
	var inputBytes int64
	for _, unit := range units {
		inputBytes += int64(len(unit))
	}
	b.ReportAllocs()
	b.SetBytes(inputBytes)
	b.ResetTimer()

	var frameCount int
	for i := 0; i < b.N; i++ {
		dec := NewDecoder()
		frameCount = 0
		for _, unit := range units {
			frames, err := dec.DecodeFrames(unit)
			if err != nil {
				b.Fatal(err)
			}
			frameCount += len(frames)
		}
		if frameCount != benchmarkHigh10IDRPFrames {
			b.Fatalf("frames = %d, want %d", frameCount, benchmarkHigh10IDRPFrames)
		}
	}
	benchmarkDecodeFramesSink = frameCount
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
