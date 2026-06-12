// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

import (
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
	damaged := simpleDecodeTestTruncateFirstVCL(t, accessUnits[1])
	if frames, err := dec.DecodeNALUnits(damaged); err == nil {
		t.Fatalf("damaged access unit decoded frames=%d, want error", len(frames))
	}
	if dec.st.frame != nil || dec.st.tables != nil || dec.st.motionScratch != nil || dec.st.motionScratchHigh != nil ||
		dec.st.loopFilterSlices != nil || dec.st.loopFilterRefFrameIDs != nil || dec.st.haveSlice ||
		dec.st.frameComplete || dec.st.fieldPairPending || dec.st.sliceNum != 0 {
		t.Fatalf("damaged slice left partial picture state: %+v", dec.st)
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

func simpleDecodeTestFixturePath(t *testing.T, name string) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Join(filepath.Dir(file), "..", "..", "testdata", "h264", name)
}
