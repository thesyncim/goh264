// SPDX-License-Identifier: LGPL-2.1-or-later

package goh264

import (
	"errors"
	"testing"
	"unsafe"
)

func TestEncoderEncodeIntoOverflowedP16PlanningDoesNotCommitScratch(t *testing.T) {
	cfg := DefaultEncoderConfig(144, 144)
	cfg.OutputFormat = EncoderOutputAnnexB
	cfg.RTPMaxPayloadSize = 0
	cfg.DeblockMode = EncoderDeblockDisabled
	enc, err := NewEncoder(cfg)
	if err != nil {
		t.Fatalf("NewEncoder: %v", err)
	}

	firstFrame := testPatternedI420EncoderFrame(cfg, 0)
	first, err := enc.Encode(firstFrame)
	if err != nil {
		t.Fatalf("Encode first IDR: %v", err)
	}
	if !first.IDR {
		t.Fatalf("first frame idr=%v, want IDR", first.IDR)
	}
	if len(enc.p16MVs) != 0 || len(enc.p16MVDs) != 0 {
		t.Fatalf("first IDR committed P16 scratch: mvs=%d mvds=%d", len(enc.p16MVs), len(enc.p16MVDs))
	}

	pFrame := testIntegerMotionI420EncoderFrame(cfg, firstFrame, 2, 0, int64(cfg.RTPTimestampIncrement))
	out, err := enc.EncodeInto(testFakeRawBytesLen(int(^uint(0)>>1)-3), pFrame)
	if !errors.Is(err, ErrInvalidData) {
		t.Fatalf("overflowed P16 EncodeInto error = %v, want ErrInvalidData", err)
	}
	if out.Dropped || len(out.Data) != 0 || len(out.NALUnits) != 0 || len(out.RTPPackets) != 0 {
		t.Fatalf("overflowed P16 EncodeInto output = %+v, want empty output", out)
	}
	if len(enc.p16MVs) != 0 || len(enc.p16MVDs) != 0 {
		t.Fatalf("overflowed P16 EncodeInto committed scratch: mvs=%d mvds=%d", len(enc.p16MVs), len(enc.p16MVDs))
	}

	recovered, err := enc.EncodeInto(make([]byte, 0, 4096), pFrame)
	if err != nil {
		t.Fatalf("EncodeInto after overflowed P16 planning: %v", err)
	}
	if recovered.IDR || recovered.Dropped {
		t.Fatalf("post-overflow P16 output idr=%v dropped=%v, want delivered P frame", recovered.IDR, recovered.Dropped)
	}
	if len(enc.p16MVs) < 81 || len(enc.p16MVDs) < 81 {
		t.Fatalf("successful P16 EncodeInto scratch = mvs %d mvds %d, want committed reusable scratch", len(enc.p16MVs), len(enc.p16MVDs))
	}
}

func TestEncoderEncodeIntoDroppedP16PlanningDoesNotCommitScratch(t *testing.T) {
	cfg := DefaultEncoderConfig(144, 144)
	cfg.OutputFormat = EncoderOutputAnnexB
	cfg.RTPMaxPayloadSize = 0
	cfg.DeblockMode = EncoderDeblockDisabled
	cfg.FrameDrop = EncoderFrameDropToBitrate
	cfg.MaxFrameSize = 1 << 20
	enc, err := NewEncoder(cfg)
	if err != nil {
		t.Fatalf("NewEncoder: %v", err)
	}

	firstFrame := testPatternedI420EncoderFrame(cfg, 0)
	first, err := enc.Encode(firstFrame)
	if err != nil {
		t.Fatalf("Encode first IDR: %v", err)
	}
	if !first.IDR {
		t.Fatalf("first frame idr=%v, want IDR", first.IDR)
	}
	if len(enc.p16MVs) != 0 || len(enc.p16MVDs) != 0 {
		t.Fatalf("first IDR committed P16 scratch: mvs=%d mvds=%d", len(enc.p16MVs), len(enc.p16MVDs))
	}
	if err := enc.SetMaxFrameSize(16); err != nil {
		t.Fatalf("SetMaxFrameSize drop budget: %v", err)
	}

	pFrame := testIntegerMotionI420EncoderFrame(cfg, firstFrame, 2, 0, int64(cfg.RTPTimestampIncrement))
	out, err := enc.EncodeInto(make([]byte, 0, 4096), pFrame)
	if err != nil {
		t.Fatalf("dropped P16 EncodeInto error = %v, want nil", err)
	}
	if !out.Dropped || len(out.Data) != 0 || len(out.NALUnits) != 0 || len(out.RTPPackets) != 0 {
		t.Fatalf("dropped P16 EncodeInto output = %+v, want empty dropped output", out)
	}
	if len(enc.p16MVs) != 0 || len(enc.p16MVDs) != 0 {
		t.Fatalf("dropped P16 EncodeInto committed scratch: mvs=%d mvds=%d", len(enc.p16MVs), len(enc.p16MVDs))
	}

	if err := enc.SetMaxFrameSize(1 << 20); err != nil {
		t.Fatalf("SetMaxFrameSize restore budget: %v", err)
	}
	recovered, err := enc.EncodeInto(make([]byte, 0, 4096), pFrame)
	if err != nil {
		t.Fatalf("EncodeInto after dropped P16 planning: %v", err)
	}
	if recovered.IDR || recovered.Dropped {
		t.Fatalf("post-drop P16 output idr=%v dropped=%v, want delivered P frame", recovered.IDR, recovered.Dropped)
	}
	if len(enc.p16MVs) < 81 || len(enc.p16MVDs) < 81 {
		t.Fatalf("successful P16 EncodeInto scratch = mvs %d mvds %d, want committed reusable scratch", len(enc.p16MVs), len(enc.p16MVDs))
	}
}

func TestEncoderP16x16ResidualPlanFromPixelDeltaDerivesLumaDC(t *testing.T) {
	cfg := DefaultEncoderConfig(16, 16)
	cfg.OutputFormat = EncoderOutputAnnexB
	cfg.RTPMaxPayloadSize = 0
	cfg.DeblockMode = EncoderDeblockDisabled
	cfg.RateControl = EncoderRateControlConstantQP
	enc, err := NewEncoder(cfg)
	if err != nil {
		t.Fatalf("NewEncoder: %v", err)
	}

	reference := testPatternedI420EncoderFrame(cfg, 0)
	referenceView, err := enc.validatedFrameView(reference)
	if err != nil {
		t.Fatalf("validated reference view: %v", err)
	}
	enc.storeReference(referenceView)

	target, err := reference.Clone()
	if err != nil {
		t.Fatalf("Clone reference: %v", err)
	}
	target.PTS += int64(cfg.RTPTimestampIncrement)
	qmul, err := enc.p16x16ResidualLumaDCQMul()
	if err != nil {
		t.Fatalf("p16x16ResidualLumaDCQMul: %v", err)
	}
	const wantLevel int32 = 3
	delta := encoderP16x16ResidualPixelDeltaForDCLevel(wantLevel, qmul)
	if delta == 0 {
		t.Fatalf("derived delta for level %d = 0, want visible pixel delta", wantLevel)
	}
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			pos := y*target.StrideY + x
			v := int(target.Y[pos]) + delta
			if v < 0 || v > 255 {
				t.Fatalf("test target luma overflows at %d,%d: %d + %d", x, y, target.Y[pos], delta)
			}
			target.Y[pos] = byte(v)
		}
	}
	targetView, err := enc.validatedFrameView(target)
	if err != nil {
		t.Fatalf("validated target view: %v", err)
	}
	plan, ok, err := enc.p16x16ResidualPlanFromPixelDelta(targetView, []encoderSliceRange{{firstMB: 0, macroblockCount: 1}})
	if err != nil {
		t.Fatalf("p16x16ResidualPlanFromPixelDelta: %v", err)
	}
	if !ok {
		t.Fatal("p16x16ResidualPlanFromPixelDelta did not admit representable luma DC delta")
	}
	if len(plan.lumaCoefficients) != 1 || len(plan.lumaCoefficients[0]) != 1 {
		t.Fatalf("luma coefficients = %+v, want one coefficient for one macroblock", plan.lumaCoefficients)
	}
	if got := plan.lumaCoefficients[0][0]; got.Pos != 0 || got.Value != wantLevel {
		t.Fatalf("luma coefficient = %+v, want pos 0 value %d", got, wantLevel)
	}
}

func TestEncoderP16x16ResidualPlanFromPixelDeltaRejectsUnrepresentableShape(t *testing.T) {
	cfg := DefaultEncoderConfig(16, 16)
	cfg.OutputFormat = EncoderOutputAnnexB
	cfg.RTPMaxPayloadSize = 0
	cfg.DeblockMode = EncoderDeblockDisabled
	cfg.RateControl = EncoderRateControlConstantQP
	enc, err := NewEncoder(cfg)
	if err != nil {
		t.Fatalf("NewEncoder: %v", err)
	}

	reference := testPatternedI420EncoderFrame(cfg, 0)
	referenceView, err := enc.validatedFrameView(reference)
	if err != nil {
		t.Fatalf("validated reference view: %v", err)
	}
	enc.storeReference(referenceView)

	target, err := reference.Clone()
	if err != nil {
		t.Fatalf("Clone reference: %v", err)
	}
	target.PTS += int64(cfg.RTPTimestampIncrement)
	target.Y[8*target.StrideY+8] ^= 0x01
	targetView, err := enc.validatedFrameView(target)
	if err != nil {
		t.Fatalf("validated target view: %v", err)
	}
	if _, ok, err := enc.p16x16ResidualPlanFromPixelDelta(targetView, []encoderSliceRange{{firstMB: 0, macroblockCount: 1}}); err != nil || ok {
		t.Fatalf("p16x16ResidualPlanFromPixelDelta ok=%v err=%v, want clean rejection", ok, err)
	}
}

func testPatternedI420EncoderFrame(cfg EncoderConfig, pts int64) EncoderFrame {
	frame := cfg.I420Frame(
		make([]byte, cfg.StrideY*cfg.Height),
		make([]byte, cfg.StrideCb*(cfg.Height/2)),
		make([]byte, cfg.StrideCr*(cfg.Height/2)),
		pts,
	)
	for y := 0; y < frame.Height; y++ {
		for x := 0; x < frame.Width; x++ {
			frame.Y[y*frame.StrideY+x] = byte((x*5 + y*7 + 13) & 0xff)
		}
	}
	for y := 0; y < frame.Height/2; y++ {
		for x := 0; x < frame.Width/2; x++ {
			frame.Cb[y*frame.StrideCb+x] = byte((x*3 + y*11 + 29) & 0xff)
			frame.Cr[y*frame.StrideCr+x] = byte((x*13 + y*5 + 71) & 0xff)
		}
	}
	return frame
}

func testIntegerMotionI420EncoderFrame(cfg EncoderConfig, reference EncoderFrame, dx int, dy int, pts int64) EncoderFrame {
	frame := cfg.I420Frame(
		make([]byte, cfg.StrideY*cfg.Height),
		make([]byte, cfg.StrideCb*(cfg.Height/2)),
		make([]byte, cfg.StrideCr*(cfg.Height/2)),
		pts,
	)
	for y := 0; y < frame.Height; y++ {
		refY := testClampEncoderCoord(y+dy, frame.Height)
		for x := 0; x < frame.Width; x++ {
			refX := testClampEncoderCoord(x+dx, frame.Width)
			frame.Y[y*frame.StrideY+x] = reference.Y[refY*reference.StrideY+refX]
		}
	}
	chromaDX := dx / 2
	chromaDY := dy / 2
	for y := 0; y < frame.Height/2; y++ {
		refY := testClampEncoderCoord(y+chromaDY, frame.Height/2)
		for x := 0; x < frame.Width/2; x++ {
			refX := testClampEncoderCoord(x+chromaDX, frame.Width/2)
			frame.Cb[y*frame.StrideCb+x] = reference.Cb[refY*reference.StrideCb+refX]
			frame.Cr[y*frame.StrideCr+x] = reference.Cr[refY*reference.StrideCr+refX]
		}
	}
	return frame
}

func testClampEncoderCoord(v int, limit int) int {
	if v < 0 {
		return 0
	}
	if v >= limit {
		return limit - 1
	}
	return v
}

func testFakeRawBytesLen(n int) []byte {
	if n <= 0 {
		return nil
	}
	var b byte
	h := struct {
		Data unsafe.Pointer
		Len  int
		Cap  int
	}{
		Data: unsafe.Pointer(&b),
		Len:  n,
		Cap:  n,
	}
	return *(*[]byte)(unsafe.Pointer(&h))
}
