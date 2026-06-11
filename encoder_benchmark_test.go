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
	return benchmarkEncoderConfigSize(format, benchmarkEncoderWidth, benchmarkEncoderHeight)
}

func benchmarkEncoderConfigSize(format EncoderOutputFormat, width int, height int) EncoderConfig {
	cfg := DefaultEncoderConfig(width, height)
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

func benchmarkEncoderExactP16x16ReferenceFrame() EncoderFrame {
	return benchmarkEncoderExactP16x16HorizontalReferenceFrame(benchmarkEncoderWidth, benchmarkEncoderHeight, 2)
}

func benchmarkEncoderOddExactP16x16ConstantChromaReferenceFrame() EncoderFrame {
	frame := benchmarkEncoderExactP16x16HorizontalReferenceFrame(benchmarkEncoderWidth, benchmarkEncoderHeight, 1)
	benchmarkEncoderSetConstantChroma(&frame, 128, 64)
	return frame
}

func benchmarkEncoderExactP16x16HorizontalReferenceFrame(width int, height int, dx int) EncoderFrame {
	frame := benchmarkEncoderI420Frame(width, height)
	// Keep the left edge reversible under clamped +/-dx motion so the benchmark
	// can alternate two frames while staying on the exact P16x16 path.
	for y := 0; y < frame.Height; y++ {
		v := frame.Y[y*frame.StrideY+dx]
		for x := 0; x < dx; x++ {
			frame.Y[y*frame.StrideY+x] = v
		}
	}
	chromaDX := dx / 2
	chromaHeight := frame.Height / 2
	for y := 0; y < chromaHeight; y++ {
		cb := frame.Cb[y*frame.StrideCb+chromaDX]
		cr := frame.Cr[y*frame.StrideCr+chromaDX]
		for x := 0; x < chromaDX; x++ {
			frame.Cb[y*frame.StrideCb+x] = cb
			frame.Cr[y*frame.StrideCr+x] = cr
		}
	}
	return frame
}

func benchmarkEncoderSetConstantChroma(frame *EncoderFrame, cb byte, cr byte) {
	chromaWidth := frame.Width / 2
	chromaHeight := frame.Height / 2
	for y := 0; y < chromaHeight; y++ {
		for x := 0; x < chromaWidth; x++ {
			frame.Cb[y*frame.StrideCb+x] = cb
			frame.Cr[y*frame.StrideCr+x] = cr
		}
	}
}

func benchmarkEncoderIntegerMotionFrame(reference EncoderFrame, dx int, dy int) EncoderFrame {
	chromaWidth := reference.Width / 2
	chromaHeight := reference.Height / 2
	frame := EncoderFrame{
		Y:        make([]byte, reference.Width*reference.Height),
		Cb:       make([]byte, chromaWidth*chromaHeight),
		Cr:       make([]byte, chromaWidth*chromaHeight),
		StrideY:  reference.Width,
		StrideCb: chromaWidth,
		StrideCr: chromaWidth,
		Width:    reference.Width,
		Height:   reference.Height,
		Duration: reference.Duration,
	}
	for y := 0; y < frame.Height; y++ {
		refY := benchmarkEncoderClampCoord(y+dy, frame.Height)
		for x := 0; x < frame.Width; x++ {
			refX := benchmarkEncoderClampCoord(x+dx, frame.Width)
			frame.Y[y*frame.StrideY+x] = reference.Y[refY*reference.StrideY+refX]
		}
	}
	chromaDX := dx / 2
	chromaDY := dy / 2
	for y := 0; y < chromaHeight; y++ {
		refY := benchmarkEncoderClampCoord(y+chromaDY, chromaHeight)
		for x := 0; x < chromaWidth; x++ {
			refX := benchmarkEncoderClampCoord(x+chromaDX, chromaWidth)
			frame.Cb[y*frame.StrideCb+x] = reference.Cb[refY*reference.StrideCb+refX]
			frame.Cr[y*frame.StrideCr+x] = reference.Cr[refY*reference.StrideCr+refX]
		}
	}
	return frame
}

func benchmarkEncoderClampCoord(v int, limit int) int {
	if v < 0 {
		return 0
	}
	if v >= limit {
		return limit - 1
	}
	return v
}

func benchmarkEncoderInputBytes() int {
	return benchmarkEncoderFrameBytes(benchmarkEncoderWidth, benchmarkEncoderHeight)
}

func benchmarkEncoderFrameBytes(width int, height int) int {
	return width * height * 3 / 2
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

func BenchmarkEncodeAnnexBI420ExactP16x16(b *testing.B) {
	cfg := benchmarkEncoderConfig(EncoderOutputAnnexB)
	a := benchmarkEncoderExactP16x16ReferenceFrame()
	shifted := benchmarkEncoderIntegerMotionFrame(a, 2, 0)
	benchmarkEncodeSteadyPFrame(b, cfg, []EncoderFrame{shifted, a}, false)
}

func BenchmarkEncodeAnnexBI420OddExactP16x16ConstantChroma(b *testing.B) {
	cfg := benchmarkEncoderConfig(EncoderOutputAnnexB)
	a := benchmarkEncoderOddExactP16x16ConstantChromaReferenceFrame()
	shifted := benchmarkEncoderIntegerMotionFrame(a, 1, 0)
	benchmarkEncodeSteadyPFrame(b, cfg, []EncoderFrame{shifted, a}, false)
}

func BenchmarkEncodeAnnexBI420ExactP16x16EdgeSearch(b *testing.B) {
	cfg := benchmarkEncoderConfigSize(EncoderOutputAnnexB, 48, 48)
	a := benchmarkEncoderExactP16x16HorizontalReferenceFrame(cfg.Width, cfg.Height, 8)
	shifted := benchmarkEncoderIntegerMotionFrame(a, 8, 0)
	benchmarkEncodeSteadyPFrame(b, cfg, []EncoderFrame{shifted, a}, false)
}

func BenchmarkEncodeAnnexBI420ChangedPIntraPCM(b *testing.B) {
	cfg := benchmarkEncoderConfig(EncoderOutputAnnexB)
	a := benchmarkEncoderI420Frame(benchmarkEncoderWidth, benchmarkEncoderHeight)
	bFrame := benchmarkEncoderI420Frame(benchmarkEncoderWidth, benchmarkEncoderHeight)
	bFrame.Y[0] ^= 0x7f
	benchmarkEncodeSteadyPFrame(b, cfg, []EncoderFrame{bFrame, a}, false)
}

func BenchmarkEncodeAVCI420IDRIntraPCM(b *testing.B) {
	cfg := benchmarkEncoderConfig(EncoderOutputAVC)
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
			b.Fatalf("output idr=%v rtp=%d data=%d, want AVC IDR", out.IDR, len(out.RTPPackets), len(out.Data))
		}
	}
	benchmarkEncodeFrameSink = out
	benchmarkEncodeBytesSink = len(out.Data)
}

func BenchmarkEncodeAVCI420PSkip(b *testing.B) {
	cfg := benchmarkEncoderConfig(EncoderOutputAVC)
	frame := benchmarkEncoderI420Frame(benchmarkEncoderWidth, benchmarkEncoderHeight)
	benchmarkEncodeSteadyPFrame(b, cfg, []EncoderFrame{frame}, false)
}

func BenchmarkEncodeAVCI420ExactP16x16(b *testing.B) {
	cfg := benchmarkEncoderConfig(EncoderOutputAVC)
	a := benchmarkEncoderExactP16x16ReferenceFrame()
	shifted := benchmarkEncoderIntegerMotionFrame(a, 2, 0)
	benchmarkEncodeSteadyPFrame(b, cfg, []EncoderFrame{shifted, a}, false)
}

func BenchmarkEncodeAVCI420ExactP16x16EdgeSearch(b *testing.B) {
	cfg := benchmarkEncoderConfigSize(EncoderOutputAVC, 48, 48)
	a := benchmarkEncoderExactP16x16HorizontalReferenceFrame(cfg.Width, cfg.Height, 8)
	shifted := benchmarkEncoderIntegerMotionFrame(a, 8, 0)
	benchmarkEncodeSteadyPFrame(b, cfg, []EncoderFrame{shifted, a}, false)
}

func BenchmarkEncodeAVCI420ChangedPIntraPCM(b *testing.B) {
	cfg := benchmarkEncoderConfig(EncoderOutputAVC)
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

func BenchmarkEncodeRTPMode0I420IDRIntraPCM(b *testing.B) {
	cfg := benchmarkEncoderConfig(EncoderOutputRTP)
	cfg.RTPPacketizationMode = EncoderRTPPacketizationSingleNAL
	cfg.RTPMaxPayloadSize = 1200
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
		if !out.IDR || len(out.RTPPackets) != 3 || len(out.Data) == 0 {
			b.Fatalf("output idr=%v rtp=%d data=%d, want RTP mode0 IDR", out.IDR, len(out.RTPPackets), len(out.Data))
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

func BenchmarkEncodeRTPI420ExactP16x16(b *testing.B) {
	cfg := benchmarkEncoderConfig(EncoderOutputRTP)
	a := benchmarkEncoderExactP16x16ReferenceFrame()
	shifted := benchmarkEncoderIntegerMotionFrame(a, 2, 0)
	benchmarkEncodeSteadyPFrame(b, cfg, []EncoderFrame{shifted, a}, true)
}

func BenchmarkEncodeRTPI420ExactP16x16EdgeSearch(b *testing.B) {
	cfg := benchmarkEncoderConfigSize(EncoderOutputRTP, 48, 48)
	a := benchmarkEncoderExactP16x16HorizontalReferenceFrame(cfg.Width, cfg.Height, 8)
	shifted := benchmarkEncoderIntegerMotionFrame(a, 8, 0)
	benchmarkEncodeSteadyPFrame(b, cfg, []EncoderFrame{shifted, a}, true)
}

func BenchmarkEncodeRTPI420ChangedPIntraPCM(b *testing.B) {
	cfg := benchmarkEncoderConfig(EncoderOutputRTP)
	a := benchmarkEncoderI420Frame(benchmarkEncoderWidth, benchmarkEncoderHeight)
	bFrame := benchmarkEncoderI420Frame(benchmarkEncoderWidth, benchmarkEncoderHeight)
	bFrame.Y[0] ^= 0x7f
	benchmarkEncodeSteadyPFrame(b, cfg, []EncoderFrame{bFrame, a}, true)
}

func BenchmarkEncodeRTPMode0I420PSkip(b *testing.B) {
	cfg := benchmarkEncoderConfig(EncoderOutputRTP)
	cfg.RTPPacketizationMode = EncoderRTPPacketizationSingleNAL
	cfg.RTPMaxPayloadSize = 1200
	frame := benchmarkEncoderI420Frame(benchmarkEncoderWidth, benchmarkEncoderHeight)
	benchmarkEncodeSteadyPFrame(b, cfg, []EncoderFrame{frame}, true)
}

func BenchmarkEncodeRTPMode0I420ExactP16x16(b *testing.B) {
	cfg := benchmarkEncoderConfig(EncoderOutputRTP)
	cfg.RTPPacketizationMode = EncoderRTPPacketizationSingleNAL
	cfg.RTPMaxPayloadSize = 1200
	a := benchmarkEncoderExactP16x16ReferenceFrame()
	shifted := benchmarkEncoderIntegerMotionFrame(a, 2, 0)
	benchmarkEncodeSteadyPFrame(b, cfg, []EncoderFrame{shifted, a}, true)
}

func BenchmarkEncodeRTPMode0I420ExactP16x16EdgeSearch(b *testing.B) {
	cfg := benchmarkEncoderConfigSize(EncoderOutputRTP, 32, 16)
	cfg.RTPPacketizationMode = EncoderRTPPacketizationSingleNAL
	cfg.RTPMaxPayloadSize = 1200
	a := benchmarkEncoderExactP16x16HorizontalReferenceFrame(cfg.Width, cfg.Height, 8)
	shifted := benchmarkEncoderIntegerMotionFrame(a, 8, 0)
	benchmarkEncodeSteadyPFrame(b, cfg, []EncoderFrame{shifted, a}, true)
}

func BenchmarkEncodeRTPMode0I420ChangedPIntraPCM(b *testing.B) {
	cfg := benchmarkEncoderConfig(EncoderOutputRTP)
	cfg.RTPPacketizationMode = EncoderRTPPacketizationSingleNAL
	cfg.RTPMaxPayloadSize = 1200
	a := benchmarkEncoderI420Frame(benchmarkEncoderWidth, benchmarkEncoderHeight)
	bFrame := benchmarkEncoderI420Frame(benchmarkEncoderWidth, benchmarkEncoderHeight)
	bFrame.Y[0] ^= 0x7f
	benchmarkEncodeSteadyPFrame(b, cfg, []EncoderFrame{bFrame, a}, true)
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
	b.SetBytes(int64(benchmarkEncoderFrameBytes(cfg.Width, cfg.Height)))
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
			b.Fatalf("non-RTP steady P frame returned RTP packets: %d", len(out.RTPPackets))
		}
	}
	benchmarkEncodeFrameSink = out
	benchmarkEncodeBytesSink = len(out.Data)
	benchmarkEncodePacketsSink = len(out.RTPPackets)
}
