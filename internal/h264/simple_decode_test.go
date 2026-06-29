// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestApplySimpleFrameTimingPropsFromPictureTiming(t *testing.T) {
	sps := &SPS{PicStructPresentFlag: 1}
	for _, tt := range []struct {
		name          string
		picStruct     int32
		ctType        int32
		fieldPOC      [2]int32
		repeatPict    int
		interlaced    bool
		topFieldFirst bool
	}{
		{
			name:          "top-bottom-uses-initial-prev-interlaced",
			picStruct:     h264SEIPicStructTopBottom,
			interlaced:    true,
			topFieldFirst: true,
		},
		{
			name:          "ct-progressive-overrides-top-bottom-interlace",
			picStruct:     h264SEIPicStructTopBottom,
			ctType:        1,
			topFieldFirst: true,
		},
		{
			name:       "top-field",
			picStruct:  h264SEIPicStructTopField,
			interlaced: true,
		},
		{
			name:          "top-bottom-top-repeat",
			picStruct:     h264SEIPicStructTopBottomTop,
			repeatPict:    1,
			topFieldFirst: true,
		},
		{
			name:       "frame-doubling",
			picStruct:  h264SEIPicStructFrameDoubling,
			repeatPict: 2,
		},
		{
			name:       "frame-tripling",
			picStruct:  h264SEIPicStructFrameTripling,
			repeatPict: 4,
		},
		{
			name:       "field-poc-priority",
			picStruct:  h264SEIPicStructTopBottomTop,
			fieldPOC:   [2]int32{4, 2},
			repeatPict: 1,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			frame := &DecodedFrame{fieldPOC: tt.fieldPOC}
			sei := &H264SEIContext{}
			sei.PictureTiming.Present = 1
			sei.PictureTiming.PicStruct = tt.picStruct
			sei.PictureTiming.CTType = tt.ctType
			var dpb simpleFrameDPB

			applySimpleFrameTimingProps(frame, sps, sei, &dpb)

			if frame.RepeatPict != tt.repeatPict || frame.InterlacedFrame != tt.interlaced ||
				frame.TopFieldFirst != tt.topFieldFirst {
				t.Fatalf("timing = repeat %d interlaced %t top-first %t",
					frame.RepeatPict, frame.InterlacedFrame, frame.TopFieldFirst)
			}
		})
	}
}

func TestSimpleDecoderFlushDelayedFrameRejectsMultipleWithoutDraining(t *testing.T) {
	sps := simpleDPBTestSPS(2)
	earlier := simpleDPBTestFrame(sps, 0)
	earlier.poc = 0
	later := simpleDPBTestFrame(sps, 1)
	later.poc = 2
	dec := &SimpleDecoder{
		dpb: simpleFrameDPB{
			delayed:        []*DecodedFrame{later, earlier},
			frameRecovered: simpleFrameRecoveredSEI,
		},
	}

	frame, err := dec.FlushDelayedFrame()
	if frame != nil {
		t.Fatal("FlushDelayedFrame returned frame, want nil")
	}
	if !errors.Is(err, ErrUnsupported) {
		t.Fatalf("FlushDelayedFrame error = %v, want ErrUnsupported", err)
	}
	if len(dec.dpb.delayed) != 2 || dec.dpb.delayed[0] != later || dec.dpb.delayed[1] != earlier {
		t.Fatalf("delayed frames after failed single flush = %v, want original two-frame queue", dec.dpb.delayed)
	}
	if earlier.recovered != 0 || later.recovered != 0 {
		t.Fatalf("recovered flags after failed single flush = %d/%d, want 0/0", earlier.recovered, later.recovered)
	}

	frames, err := dec.FlushDelayedFrames()
	if err != nil {
		t.Fatalf("FlushDelayedFrames: %v", err)
	}
	if len(frames) != 2 || frames[0] != earlier || frames[1] != later {
		t.Fatalf("FlushDelayedFrames = %v, want earlier then later", frames)
	}
}

func TestCanDropTerminalDamagedFieldSlice(t *testing.T) {
	nals := []NALUnit{
		{Type: NALSPS},
		{Type: NALSlice},
		{Type: NALSEI},
		{Type: NALAUD},
	}
	if !canDropTerminalDamagedFieldSlice(nals, 1, true, true, false) {
		t.Fatal("terminal damaged first-field slice was not droppable")
	}

	for _, tt := range []struct {
		name                       string
		nals                       []NALUnit
		index                      int
		flushOutput                bool
		fieldPicture               bool
		decodingComplementaryField bool
	}{
		{name: "streaming", nals: nals, index: 1, fieldPicture: true},
		{name: "frame-picture", nals: nals, index: 1, flushOutput: true},
		{name: "complementary-field", nals: nals, index: 1, flushOutput: true, fieldPicture: true, decodingComplementaryField: true},
		{name: "later-vcl", nals: []NALUnit{{Type: NALSlice}, {Type: NALSEI}, {Type: NALSlice}}, index: 0, flushOutput: true, fieldPicture: true},
		{name: "bad-index", nals: nals, index: -1, flushOutput: true, fieldPicture: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if canDropTerminalDamagedFieldSlice(tt.nals, tt.index, tt.flushOutput, tt.fieldPicture, tt.decodingComplementaryField) {
				t.Fatalf("canDropTerminalDamagedFieldSlice(%s) = true, want false", tt.name)
			}
		})
	}
}

func TestSimpleDecoderResetsPartialPictureAfterDamagedSlice(t *testing.T) {
	data, err := os.ReadFile(simpleDecodeTestFixturePath(t, "high10_inter_cavlc_idrp.h264"))
	if err != nil {
		t.Fatal(err)
	}
	accessUnits := simpleDecodeTestAccessUnits(t, data)
	if len(accessUnits) < 2 {
		t.Fatalf("access units = %d, want at least 2", len(accessUnits))
	}

	var dec SimpleDecoder
	if frames, err := dec.DecodeNALUnits(accessUnits[0]); err != nil || len(frames) != 1 {
		t.Fatalf("DecodeNALUnits first frames=%d err=%v, want one frame", len(frames), err)
	}
	beforeRefs := append([]uint32(nil), simpleDPBFrameNums(dec.dpb.short)...)
	gapped := simpleDecodeTestSetFirstVCLFrameNum(t, accessUnits[1], &dec.pps, 2)
	damaged := simpleDecodeTestTruncateFirstVCL(t, gapped)
	if frames, err := dec.DecodeNALUnits(damaged); err == nil {
		t.Fatalf("damaged access unit decoded frames=%d, want error", len(frames))
	}
	if dec.st.frame != nil || dec.st.tables != nil || dec.st.motionScratch != nil || dec.st.motionScratchHigh != nil ||
		len(dec.st.loopFilterSlices) != 0 || len(dec.st.loopFilterRefFrameIDs) != 0 || dec.st.haveSlice ||
		dec.st.frameComplete || dec.st.fieldPairPending || dec.st.sliceNum != 0 {
		t.Fatalf("damaged slice left partial picture state: %+v", dec.st)
	}
	if got := simpleDPBFrameNums(dec.dpb.short); !uint32SlicesEqual(got, beforeRefs) {
		t.Fatalf("damaged gapped slice left DPB refs = %v, want %v", got, beforeRefs)
	}
	for _, frame := range dec.dpb.short {
		if frame != nil && frame.invalidGap {
			t.Fatalf("damaged gapped slice left invalid gap ref in %v", simpleDPBFrameNums(dec.dpb.short))
		}
	}

	if frames, err := dec.DecodeNALUnits(accessUnits[1]); err != nil || len(frames) != 1 {
		t.Fatalf("DecodeNALUnits after damaged slice frames=%d err=%v, want one frame", len(frames), err)
	}
}

func simpleDecodeTestAccessUnits(t *testing.T, data []byte) [][]NALUnit {
	t.Helper()
	nals, err := SplitAnnexB(data)
	if err != nil {
		t.Fatal(err)
	}
	var spsList [maxSPSCount]*SPS
	var ppsList [maxPPSCount]*PPS
	var accessUnits [][]NALUnit
	var current []NALUnit
	hasVCL := false
	for _, nal := range nals {
		switch nal.Type {
		case NALSPS:
			sps, err := DecodeSPSFromNAL(nal)
			if err != nil {
				t.Fatal(err)
			}
			spsList[sps.SPSID] = sps
		case NALPPS:
			pps, err := DecodePPS(nal.RBSP, &spsList)
			if err != nil {
				t.Fatal(err)
			}
			ppsList[pps.PPSID] = pps
		case NALSlice, NALIDRSlice:
			sh, err := ParseSliceHeader(nal, &ppsList)
			if err != nil {
				t.Fatal(err)
			}
			if hasVCL && sh.FirstMBAddr == 0 {
				accessUnits = append(accessUnits, current)
				current = nil
				hasVCL = false
			}
			hasVCL = true
		}
		current = append(current, nal)
	}
	if len(current) != 0 {
		accessUnits = append(accessUnits, current)
	}
	return accessUnits
}

func simpleDecodeTestSetFirstVCLFrameNum(t *testing.T, nals []NALUnit, ppsList *[maxPPSCount]*PPS, frameNum uint32) []NALUnit {
	t.Helper()
	out := append([]NALUnit(nil), nals...)
	for i, nal := range out {
		if nal.Type != NALSlice && nal.Type != NALIDRSlice {
			continue
		}
		if nal.Type == NALIDRSlice {
			t.Fatal("first VCL was IDR, want non-IDR slice for frame-num gap test")
		}
		rbsp := append([]byte(nil), nal.RBSP...)
		gb, err := newRBSPBitReader(rbsp)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := gb.readUEGolombLong(); err != nil {
			t.Fatal(err)
		}
		if _, err := gb.readUEGolomb31(); err != nil {
			t.Fatal(err)
		}
		if _, err := gb.readUEGolombLong(); err != nil {
			t.Fatal(err)
		}
		sh, err := ParseSliceHeader(nal, ppsList)
		if err != nil {
			t.Fatal(err)
		}
		if sh.SPS == nil || sh.SPS.Log2MaxFrameNum <= 0 {
			t.Fatalf("invalid log2_max_frame_num: %+v", sh.SPS)
		}
		simpleDecodeTestWriteBits(rbsp, gb.bitPos, uint32(sh.SPS.Log2MaxFrameNum), frameNum)
		nal.RBSP = rbsp
		out[i] = nal
		return out
	}
	t.Fatal("no VCL NAL found")
	return nil
}

func simpleDecodeTestTruncateFirstVCL(t *testing.T, nals []NALUnit) []NALUnit {
	t.Helper()
	var out []NALUnit
	truncated := false
	for _, nal := range nals {
		if !truncated && (nal.Type == NALSlice || nal.Type == NALIDRSlice) {
			if len(nal.RBSP) < 4 {
				t.Fatalf("short VCL RBSP: %x", nal.RBSP)
			}
			annexB, err := AppendAnnexBNAL(nil, nal.RefIDC, nal.Type, nal.RBSP[:len(nal.RBSP)/2])
			if err != nil {
				t.Fatal(err)
			}
			split, err := SplitAnnexB(annexB)
			if err != nil {
				t.Fatal(err)
			}
			if len(split) != 1 {
				t.Fatalf("truncated VCL split into %d NALs, want 1", len(split))
			}
			out = append(out, split[0])
			truncated = true
			continue
		}
		out = append(out, nal)
	}
	if !truncated {
		t.Fatal("no VCL NAL found")
	}
	return out
}

func simpleDecodeTestWriteBits(buf []byte, bitPos uint32, n uint32, v uint32) {
	for i := uint32(0); i < n; i++ {
		byteIndex := (bitPos + i) >> 3
		bitOffset := 7 - ((bitPos + i) & 7)
		mask := byte(1 << bitOffset)
		if (v>>(n-1-i))&1 != 0 {
			buf[byteIndex] |= mask
		} else {
			buf[byteIndex] &^= mask
		}
	}
}

func simpleDecodeTestFixturePath(t *testing.T, name string) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Join(filepath.Dir(file), "..", "..", "testdata", "h264", name)
}
