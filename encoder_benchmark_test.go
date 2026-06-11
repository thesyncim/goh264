// SPDX-License-Identifier: LGPL-2.1-or-later

package goh264

import "testing"

const (
	benchmarkEncoderWidth  = 16
	benchmarkEncoderHeight = 16
)

var (
	benchmarkEncodeFrameSink   EncodedFrame
	benchmarkEncodeBytesSink   int
	benchmarkEncodePacketsSink int
)

func benchmarkEncoderConfig(format EncoderOutputFormat) EncoderConfig {
	cfg := DefaultEncoderConfig(benchmarkEncoderWidth, benchmarkEncoderHeight)
	cfg.OutputFormat = format
	cfg.DeblockMode = EncoderDeblockDisabled
	cfg.GOPSize = 1 << 30
	cfg.IDRInterval = cfg.GOPSize
	if format != EncoderOutputRTP {
		cfg.RTPMaxPayloadSize = 0
	}
	return cfg
}

func benchmarkEncoderI420Frame(width, height int) EncoderFrame {
	chromaWidth := width / 2
	chromaHeight := height / 2
	frame := EncoderFrame{
		Y:        make([]byte, width*height),
		Cb:       make([]byte, chromaWidth*chromaHeight),
		Cr:       make([]byte, chromaWidth*chromaHeight),
		StrideY:  width,
		StrideCb: chromaWidth,
		StrideCr: chromaWidth,
		Width:    width,
		Height:   height,
		Duration: 3000,
	}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			frame.Y[y*frame.StrideY+x] = byte((x*11 + y*17 + 3) & 0xff)
		}
	}
	for y := 0; y < chromaHeight; y++ {
		for x := 0; x < chromaWidth; x++ {
			frame.Cb[y*frame.StrideCb+x] = byte((x*19 + y*7 + 41) & 0xff)
			frame.Cr[y*frame.StrideCr+x] = byte((x*5 + y*23 + 109) & 0xff)
		}
	}
	return frame
}

func benchmarkEncoderInputBytes() int {
	return benchmarkEncoderWidth * benchmarkEncoderHeight * 3 / 2
}

func BenchmarkEncodeAnnexBI420IDRIntraPCM(b *testing.B) {
	cfg := benchmarkEncoderConfig(EncoderOutputAnnexB)
	frame := benchmarkEncoderI420Frame(benchmarkEncoderWidth, benchmarkEncoderHeight)
	dst := make([]byte, 0, 4096)

	b.ReportAllocs()
	b.SetBytes(int64(benchmarkEncoderInputBytes()))
	b.ResetTimer()

	var out EncodedFrame
	for i := 0; i < b.N; i++ {
		enc, err := NewEncoder(cfg)
		if err != nil {
			b.Fatal(err)
		}
		out, err = enc.EncodeInto(dst[:0], frame)
		if err != nil {
			b.Fatal(err)
		}
		if !out.IDR || len(out.RTPPackets) != 0 || len(out.Data) == 0 {
			b.Fatalf("output idr=%v rtp=%d data=%d, want Annex B IDR", out.IDR, len(out.RTPPackets), len(out.Data))
		}
	}
	benchmarkEncodeFrameSink = out
	benchmarkEncodeBytesSink = len(out.Data)
}

func BenchmarkEncodeAnnexBI420PSkip(b *testing.B) {
	cfg := benchmarkEncoderConfig(EncoderOutputAnnexB)
	frame := benchmarkEncoderI420Frame(benchmarkEncoderWidth, benchmarkEncoderHeight)
	benchmarkEncodeSteadyPFrame(b, cfg, []EncoderFrame{frame}, false)
}

func BenchmarkEncodeAnnexBI420ChangedPIntraPCM(b *testing.B) {
	cfg := benchmarkEncoderConfig(EncoderOutputAnnexB)
	a := benchmarkEncoderI420Frame(benchmarkEncoderWidth, benchmarkEncoderHeight)
	bFrame := benchmarkEncoderI420Frame(benchmarkEncoderWidth, benchmarkEncoderHeight)
	bFrame.Y[0] ^= 0x7f
	benchmarkEncodeSteadyPFrame(b, cfg, []EncoderFrame{bFrame, a}, false)
}

func BenchmarkEncodeRTPI420IDRIntraPCMFUA(b *testing.B) {
	cfg := benchmarkEncoderConfig(EncoderOutputRTP)
	cfg.RTPMaxPayloadSize = 32
	frame := benchmarkEncoderI420Frame(benchmarkEncoderWidth, benchmarkEncoderHeight)
	dst := make([]byte, 0, 4096)

	b.ReportAllocs()
	b.SetBytes(int64(benchmarkEncoderInputBytes()))
	b.ResetTimer()

	var out EncodedFrame
	for i := 0; i < b.N; i++ {
		enc, err := NewEncoder(cfg)
		if err != nil {
			b.Fatal(err)
		}
		out, err = enc.EncodeInto(dst[:0], frame)
		if err != nil {
			b.Fatal(err)
		}
		if !out.IDR || len(out.RTPPackets) == 0 || len(out.Data) == 0 {
			b.Fatalf("output idr=%v rtp=%d data=%d, want RTP IDR", out.IDR, len(out.RTPPackets), len(out.Data))
		}
	}
	benchmarkEncodeFrameSink = out
	benchmarkEncodeBytesSink = len(out.Data)
	benchmarkEncodePacketsSink = len(out.RTPPackets)
}

func BenchmarkEncodeRTPI420PSkip(b *testing.B) {
	cfg := benchmarkEncoderConfig(EncoderOutputRTP)
	frame := benchmarkEncoderI420Frame(benchmarkEncoderWidth, benchmarkEncoderHeight)
	benchmarkEncodeSteadyPFrame(b, cfg, []EncoderFrame{frame}, true)
}

func benchmarkEncodeSteadyPFrame(b *testing.B, cfg EncoderConfig, frames []EncoderFrame, wantRTP bool) {
	b.Helper()
	enc, err := NewEncoder(cfg)
	if err != nil {
		b.Fatal(err)
	}
	if _, err := enc.EncodeInto(make([]byte, 0, 4096), frames[len(frames)-1]); err != nil {
		b.Fatal(err)
	}
	dst := make([]byte, 0, 4096)

	b.ReportAllocs()
	b.SetBytes(int64(benchmarkEncoderInputBytes()))
	b.ResetTimer()

	var out EncodedFrame
	for i := 0; i < b.N; i++ {
		out, err = enc.EncodeInto(dst[:0], frames[i%len(frames)])
		if err != nil {
			b.Fatal(err)
		}
		if out.IDR || len(out.Data) == 0 {
			b.Fatalf("output idr=%v data=%d, want steady P frame", out.IDR, len(out.Data))
		}
		if wantRTP && len(out.RTPPackets) == 0 {
			b.Fatal("RTP steady P frame did not return packets")
		}
		if !wantRTP && len(out.RTPPackets) != 0 {
			b.Fatalf("Annex B steady P frame returned RTP packets: %d", len(out.RTPPackets))
		}
	}
	benchmarkEncodeFrameSink = out
	benchmarkEncodeBytesSink = len(out.Data)
	benchmarkEncodePacketsSink = len(out.RTPPackets)
}
