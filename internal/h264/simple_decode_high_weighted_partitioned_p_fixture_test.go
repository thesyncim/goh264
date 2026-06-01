// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestDecodeAnnexBSimpleHigh10WeightedPartitionedPFixturesCarryPartitionedPMacroblocks(t *testing.T) {
	for _, tt := range []struct {
		name  string
		file  string
		cabac int32
	}{
		{name: "cavlc", file: "high10_weighted_partitioned_p_cavlc.h264"},
		{name: "cabac", file: "high10_weighted_partitioned_p_cabac.h264", cabac: 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			data, err := os.ReadFile(highWeightedPartitionedPFixturePath(t, tt.file))
			if err != nil {
				t.Fatal(err)
			}
			assertHigh10WeightedPartitionedPFixtureHeaders(t, data, tt.cabac)

			frames, err := DecodeAnnexBSimpleFrames(data)
			if err != nil {
				t.Fatalf("DecodeAnnexBSimpleFrames: %v", err)
			}
			if len(frames) != 5 {
				t.Fatalf("frames = %d, want 5", len(frames))
			}

			var p16x8, p8x16, p8x8 int
			for i, frame := range frames {
				if frame == nil || frame.tables == nil {
					t.Fatalf("frame[%d] missing tables", i)
				}
				if frame.BitDepthLuma != 10 || frame.BitDepthChroma != 10 ||
					frame.ChromaFormatIDC != 1 || frame.MBWidth != 4 || frame.MBHeight != 4 {
					t.Fatalf("frame[%d] format = depth %d/%d chroma %d mb %dx%d, want High10 4:2:0 4x4 MBs",
						i, frame.BitDepthLuma, frame.BitDepthChroma, frame.ChromaFormatIDC, frame.MBWidth, frame.MBHeight)
				}
				if i == 0 {
					continue
				}
				for y := 0; y < frame.MBHeight; y++ {
					for x := 0; x < frame.MBWidth; x++ {
						mbType := frame.tables.MacroblockTyp[x+y*frame.tables.MBStride]
						if mbType&MBType8x8DCT != 0 {
							t.Fatalf("frame[%d] macroblock %d,%d type = %#x, want no 8x8-DCT in weighted partitioned P fixture", i, x, y, mbType)
						}
						if mbType&MBType16x8 != 0 {
							p16x8++
						}
						if mbType&MBType8x16 != 0 {
							p8x16++
						}
						if mbType&MBType8x8 != 0 {
							p8x8++
						}
					}
				}
			}
			if p16x8 == 0 || p8x16 == 0 || p8x8 == 0 {
				t.Fatalf("partitioned P macroblocks = p16x8:%d p8x16:%d p8x8:%d, want all shapes present", p16x8, p8x16, p8x8)
			}
		})
	}
}

func assertHigh10WeightedPartitionedPFixtureHeaders(t *testing.T, data []byte, cabac int32) {
	t.Helper()

	nals, err := SplitAnnexB(data)
	if err != nil {
		t.Fatal(err)
	}

	var spsList [maxSPSCount]*SPS
	var ppsList [maxPPSCount]*PPS
	var pSlices int
	var chromaWeightedPSlices int
	for _, nal := range nals {
		switch nal.Type {
		case NALSPS:
			sps, err := DecodeSPS(nal.RBSP)
			if err != nil {
				t.Fatal(err)
			}
			if sps.ProfileIDC != 110 || sps.Width != 64 || sps.Height != 64 ||
				sps.ChromaFormatIDC != 1 || sps.BitDepthLuma != 10 || sps.BitDepthChroma != 10 ||
				sps.FrameMBSOnlyFlag != 1 || sps.MBAFF != 0 || sps.RefFrameCount != 1 {
				t.Fatalf("SPS profile/format = %d %dx%d chroma %d depth %d/%d frame_only %d/%d refs %d",
					sps.ProfileIDC, sps.Width, sps.Height, sps.ChromaFormatIDC, sps.BitDepthLuma, sps.BitDepthChroma,
					sps.FrameMBSOnlyFlag, sps.MBAFF, sps.RefFrameCount)
			}
			spsList[sps.SPSID] = sps
		case NALPPS:
			pps, err := DecodePPS(nal.RBSP, &spsList)
			if err != nil {
				t.Fatal(err)
			}
			if pps.CABAC != cabac || pps.Transform8x8Mode != 0 || pps.WeightedPred != 1 ||
				pps.WeightedBipredIDC != 0 || pps.RefCount[0] != 1 || pps.RefCount[1] != 1 {
				t.Fatalf("PPS cabac/8x8/weights/refs = %d/%d/%d/%d/%d/%d, want cabac=%d no-8x8 weighted P ref=1",
					pps.CABAC, pps.Transform8x8Mode, pps.WeightedPred, pps.WeightedBipredIDC, pps.RefCount[0], pps.RefCount[1], cabac)
			}
			ppsList[pps.PPSID] = pps
		case NALSEI:
		case NALIDRSlice, NALSlice:
			sh, err := ParseSliceHeader(nal, &ppsList)
			if err != nil {
				t.Fatal(err)
			}
			if sh.PictureStructure != PictureFrame || sh.DeblockingFilter != 0 {
				t.Fatalf("slice picture/deblock = %d/%d, want frame/disabled", sh.PictureStructure, sh.DeblockingFilter)
			}
			if sh.SliceTypeNoS == PictureTypeP {
				if sh.ListCount != 1 || sh.RefCount[0] != 1 || sh.PredWeightTable.UseWeight == 0 {
					t.Fatalf("P slice lists/ref/weights = %d/%d/%d/%d, want one L0 ref with luma weights",
						sh.ListCount, sh.RefCount[0], sh.PredWeightTable.UseWeight, sh.PredWeightTable.UseWeightChroma)
				}
				pSlices++
				if sh.PredWeightTable.UseWeightChroma != 0 {
					chromaWeightedPSlices++
				}
			}
		default:
			t.Fatalf("unexpected NAL type %d in High10 weighted partitioned P fixture", nal.Type)
		}
	}
	if pSlices != 4 {
		t.Fatalf("P slices = %d, want 4", pSlices)
	}
	if chromaWeightedPSlices == 0 {
		t.Fatal("weighted partitioned P fixture has no chroma-weighted P slices")
	}
}

func highWeightedPartitionedPFixturePath(t *testing.T, name string) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Join(filepath.Dir(file), "..", "..", "testdata", "h264", name)
}
