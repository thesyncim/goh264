// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestDecodeAnnexBSimpleHigh10PIntraFixturesCarryIntraPMacroblocks(t *testing.T) {
	for _, tt := range []struct {
		name string
		file string
	}{
		{name: "cavlc", file: "high10_cavlc_p_intra_mixed.h264"},
		{name: "cabac", file: "high10_cabac_p_intra_mixed.h264"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			data, err := os.ReadFile(highPIntraFixturePath(t, tt.file))
			if err != nil {
				t.Fatal(err)
			}
			frames, err := DecodeAnnexBSimpleFrames(data)
			if err != nil {
				t.Fatalf("DecodeAnnexBSimpleFrames: %v", err)
			}
			if len(frames) != 12 {
				t.Fatalf("frames = %d, want 12", len(frames))
			}
			pFramesWithIntra := 0
			for i, frame := range frames {
				if frame == nil || frame.tables == nil {
					t.Fatalf("frame[%d] missing tables", i)
				}
				if frame.BitDepthLuma != 10 || frame.BitDepthChroma != 10 || frame.ChromaFormatIDC != 1 {
					t.Fatalf("frame[%d] format = depth %d/%d chroma %d, want High10 4:2:0",
						i, frame.BitDepthLuma, frame.BitDepthChroma, frame.ChromaFormatIDC)
				}
				hasIntra := false
				for _, mbType := range frame.tables.MacroblockTyp[:frame.MBWidth*frame.MBHeight] {
					if isIntra(mbType) {
						hasIntra = true
						break
					}
				}
				if i == 0 {
					if !hasIntra {
						t.Fatalf("IDR frame has no intra macroblocks")
					}
					continue
				}
				if hasIntra {
					pFramesWithIntra++
				}
			}
			if pFramesWithIntra == 0 {
				t.Fatalf("decoded P frames did not retain any intra macroblocks")
			}
		})
	}
}

func highPIntraFixturePath(t *testing.T, name string) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Join(filepath.Dir(file), "..", "..", "testdata", "h264", name)
}
