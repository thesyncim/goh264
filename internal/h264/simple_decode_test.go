// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

import "testing"

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
