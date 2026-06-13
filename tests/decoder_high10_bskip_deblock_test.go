// SPDX-License-Identifier: LGPL-2.1-or-later

package goh264_test

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/thesyncim/goh264/internal/h264"
)

const (
	high10BSkipDeblockFrameRawSize = 768
	high10BSkipDeblockRawVideoMD5  = "078d6e505df703e46ebbbdb155fb47cd"
)

var high10BSkipDeblockFrameMD5 = []string{
	"857cc91515b2182f4444a4d746b9d721",
	"857cc91515b2182f4444a4d746b9d721",
	"857cc91515b2182f4444a4d746b9d721",
}

type high10BSkipDeblockFixture struct {
	name          string
	file          string
	cabac         int32
	directSpatial int32
	annexBSize    int
	annexBMD5     string
}

func TestHigh10BSkipDeblockFixtureSyntax(t *testing.T) {
	for _, tt := range high10BSkipDeblockFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := readHigh10BSkipDeblockFixture(t, tt)
			if len(data) != tt.annexBSize {
				t.Fatalf("annex b size = %d, want %d", len(data), tt.annexBSize)
			}
			sum := md5.Sum(data)
			if got := hex.EncodeToString(sum[:]); got != tt.annexBMD5 {
				t.Fatalf("annex b md5 = %s, want %s", got, tt.annexBMD5)
			}
			assertHigh10BSkipDeblockFixtureSyntax(t, data, tt)
		})
	}
}

func TestDecodeAnnexBHigh10BSkipDeblockFrames(t *testing.T) {
	for _, tt := range high10BSkipDeblockFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := readHigh10BSkipDeblockFixture(t, tt)
			assertHigh10BSkipDeblockFixtureSyntax(t, data, tt)
			frames, err := NewDecoder().DecodeAnnexBFrames(data)
			if err != nil {
				t.Fatalf("DecodeAnnexBFrames: %v", err)
			}
			assertHigh10BSkipDeblockFrames(t, frames)
		})
	}
}

func TestDecodeAVCHigh10BSkipDeblockFrames(t *testing.T) {
	for _, tt := range high10BSkipDeblockFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := readHigh10BSkipDeblockFixture(t, tt)
			assertHigh10BSkipDeblockFixtureSyntax(t, data, tt)
			for _, nalLengthSize := range []int{2, 3, 4} {
				frames, err := NewDecoder().DecodeAVCFrames(annexBToAVC(t, data, nalLengthSize), nalLengthSize)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: DecodeAVCFrames: %v", nalLengthSize, err)
				}
				assertHigh10BSkipDeblockFrames(t, frames)
			}
		})
	}
}

func TestDecodeAVCWithConfigurationRecordHigh10BSkipDeblockFrames(t *testing.T) {
	for _, tt := range high10BSkipDeblockFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := readHigh10BSkipDeblockFixture(t, tt)
			assertHigh10BSkipDeblockFixtureSyntax(t, data, tt)
			for _, nalLengthSize := range []int{2, 3, 4} {
				config, packet := annexBToAVCConfigAndPacket(t, data, nalLengthSize)
				frames, err := NewDecoder().DecodeAVCFramesWithConfigurationRecord(config, packet)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: DecodeAVCFramesWithConfigurationRecord: %v", nalLengthSize, err)
				}
				assertHigh10BSkipDeblockFrames(t, frames)
			}
		})
	}
}

func TestDecodeConfiguredAVCHigh10BSkipDeblockFramesAcrossSamplesFlush(t *testing.T) {
	for _, tt := range high10BSkipDeblockFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := readHigh10BSkipDeblockFixture(t, tt)
			assertHigh10BSkipDeblockFixtureSyntax(t, data, tt)
			for _, nalLengthSize := range []int{2, 3, 4} {
				config, samples := annexBToAVCConfigAndSamples(t, data, nalLengthSize)
				if len(samples) != len(high10BSkipDeblockFrameMD5) {
					t.Fatalf("nalLengthSize=%d: samples = %d, want %d", nalLengthSize, len(samples), len(high10BSkipDeblockFrameMD5))
				}
				dec := NewDecoder()
				if _, err := dec.ConfigureAVCDecoderConfigurationRecord(config); err != nil {
					t.Fatalf("nalLengthSize=%d config: %v", nalLengthSize, err)
				}
				var frames []*Frame
				for i, sample := range samples {
					out, err := dec.DecodeConfiguredAVCFrames(sample)
					if err != nil {
						t.Fatalf("nalLengthSize=%d sample[%d]: %v", nalLengthSize, i, err)
					}
					frames = append(frames, out...)
				}
				out, err := dec.FlushDelayedFrames()
				if err != nil {
					t.Fatalf("nalLengthSize=%d flush: %v", nalLengthSize, err)
				}
				frames = append(frames, out...)
				assertHigh10BSkipDeblockFrames(t, frames)
			}
		})
	}
}

func TestDecodeAutoConfiguredAVCHigh10BSkipDeblockFramesAcrossSamplesFlush(t *testing.T) {
	for _, tt := range high10BSkipDeblockFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := readHigh10BSkipDeblockFixture(t, tt)
			config, samples := annexBToAVCConfigAndSamples(t, data, 4)
			if len(samples) != len(high10BSkipDeblockFrameMD5) {
				t.Fatalf("samples = %d, want %d", len(samples), len(high10BSkipDeblockFrameMD5))
			}

			dec := NewDecoder()
			out, err := dec.DecodeFrames(config)
			if err != nil {
				t.Fatalf("config: %v", err)
			}
			if len(out) != 0 {
				t.Fatalf("config frames = %d, want 0", len(out))
			}

			var frames []*Frame
			for i, sample := range samples {
				out, err = dec.DecodeFrames(sample)
				if err != nil {
					t.Fatalf("sample[%d]: %v", i, err)
				}
				frames = append(frames, out...)
			}
			out, err = dec.DecodeFrames(nil)
			if err != nil {
				t.Fatalf("flush: %v", err)
			}
			frames = append(frames, out...)
			assertHigh10BSkipDeblockFrames(t, frames)
		})
	}
}

func TestFFmpegRawVideoMD5OracleHigh10BSkipDeblock(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}

	for _, tt := range high10BSkipDeblockFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join("testdata", "h264", tt.file)
			framemd5 := exec.Command("ffmpeg",
				"-hide_banner", "-v", "error",
				"-f", "h264",
				"-i", path,
				"-an", "-sn", "-dn",
				"-pix_fmt", "yuv420p10le",
				"-f", "framemd5",
				"-",
			)
			framemd5Out, err := framemd5.Output()
			if err != nil {
				t.Fatalf("ffmpeg framemd5: %v", err)
			}
			for i, want := range high10BSkipDeblockFrameMD5 {
				line := []byte(fmt.Sprintf("0, %10d, %10d,        1, %8d, %s", i, i, high10BSkipDeblockFrameRawSize, want))
				if !bytes.Contains(framemd5Out, line) {
					t.Fatalf("frame[%d] missing %q in framemd5:\n%s", i, line, framemd5Out)
				}
			}

			rawCmd := exec.Command("ffmpeg",
				"-hide_banner", "-v", "error",
				"-f", "h264",
				"-i", path,
				"-an", "-sn", "-dn",
				"-pix_fmt", "yuv420p10le",
				"-f", "rawvideo",
				"-",
			)
			raw, err := rawCmd.Output()
			if err != nil {
				t.Fatalf("ffmpeg rawvideo: %v", err)
			}
			if len(raw) != len(high10BSkipDeblockFrameMD5)*high10BSkipDeblockFrameRawSize {
				t.Fatalf("rawvideo size = %d, want %d", len(raw), len(high10BSkipDeblockFrameMD5)*high10BSkipDeblockFrameRawSize)
			}
			sum := md5.Sum(raw)
			if got := hex.EncodeToString(sum[:]); got != high10BSkipDeblockRawVideoMD5 {
				t.Fatalf("rawvideo md5 = %s, want %s", got, high10BSkipDeblockRawVideoMD5)
			}
		})
	}
}

func high10BSkipDeblockFixtures() []high10BSkipDeblockFixture {
	return []high10BSkipDeblockFixture{
		{
			name:          "temporal/cavlc",
			file:          "high10_bskip_deblock_temporal_cavlc.h264",
			directSpatial: 0,
			annexBSize:    65,
			annexBMD5:     "1942aad653c2e49191d06114a7946dbe",
		},
		{
			name:          "cabac/temporal",
			file:          "high10_bskip_deblock_temporal_cabac.h264",
			cabac:         1,
			directSpatial: 0,
			annexBSize:    71,
			annexBMD5:     "62337639776fbe3b2fb4617e215b3f79",
		},
		{
			name:          "spatial/cavlc",
			file:          "high10_bskip_deblock_spatial_cavlc.h264",
			directSpatial: 1,
			annexBSize:    65,
			annexBMD5:     "d0c6b59b8cf431dc811b079f79a2f2f0",
		},
		{
			name:          "cabac/spatial",
			file:          "high10_bskip_deblock_spatial_cabac.h264",
			cabac:         1,
			directSpatial: 1,
			annexBSize:    71,
			annexBMD5:     "e23a8af196ad69c0bc540d19f56264ea",
		},
	}
}

func readHigh10BSkipDeblockFixture(t *testing.T, tt high10BSkipDeblockFixture) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "h264", tt.file))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func assertHigh10BSkipDeblockFrames(t *testing.T, frames []*Frame) {
	t.Helper()
	assertHigh10FrameMD5Strings(t, frames, high10BSkipDeblockFrameMD5)
	for i, frame := range frames {
		pixFmt, err := frame.RawPixelFormat()
		if err != nil {
			t.Fatalf("frame[%d] RawPixelFormat: %v", i, err)
		}
		bytesPerSample, err := frame.BytesPerSample()
		if err != nil {
			t.Fatalf("frame[%d] BytesPerSample: %v", i, err)
		}
		if frame.Width != 16 || frame.Height != 16 || pixFmt != "yuv420p10le" || bytesPerSample != 2 {
			t.Fatalf("frame[%d] geometry/pixfmt/sample = %dx%d %s/%d, want 16x16 yuv420p10le/2",
				i, frame.Width, frame.Height, pixFmt, bytesPerSample)
		}
		rawSize, err := frame.RawYUVSize()
		if err != nil {
			t.Fatalf("frame[%d] RawYUVSize: %v", i, err)
		}
		if rawSize != high10BSkipDeblockFrameRawSize {
			t.Fatalf("frame[%d] raw yuv size = %d, want %d", i, rawSize, high10BSkipDeblockFrameRawSize)
		}
	}
}

func assertHigh10BSkipDeblockFixtureSyntax(t *testing.T, data []byte, tt high10BSkipDeblockFixture) {
	t.Helper()
	nals, err := h264.SplitAnnexB(data)
	if err != nil {
		t.Fatal(err)
	}

	var spsList [32]*h264.SPS
	var ppsList [256]*h264.PPS
	var gotSlices []int32
	for _, nal := range nals {
		switch nal.Type {
		case h264.NALSPS:
			sps, err := h264.DecodeSPS(nal.RBSP)
			if err != nil {
				t.Fatal(err)
			}
			if sps.ProfileIDC != 110 || sps.Width != 16 || sps.Height != 16 ||
				sps.ChromaFormatIDC != 1 || sps.BitDepthLuma != 10 || sps.BitDepthChroma != 10 ||
				sps.FrameMBSOnlyFlag != 1 || sps.MBAFF != 0 || sps.RefFrameCount != 2 ||
				sps.Direct8x8InferenceFlag == 0 {
				t.Fatalf("SPS profile/format = %d %dx%d chroma %d depth %d/%d frame_only %d/%d refs %d direct8x8 %d, want High10 16x16 yuv420p10le frame-only refs=2 direct8x8",
					sps.ProfileIDC, sps.Width, sps.Height, sps.ChromaFormatIDC, sps.BitDepthLuma, sps.BitDepthChroma,
					sps.FrameMBSOnlyFlag, sps.MBAFF, sps.RefFrameCount, sps.Direct8x8InferenceFlag)
			}
			spsList[sps.SPSID] = sps
		case h264.NALPPS:
			pps, err := h264.DecodePPS(nal.RBSP, &spsList)
			if err != nil {
				t.Fatal(err)
			}
			if pps.CABAC != tt.cabac || pps.Transform8x8Mode != 0 || pps.WeightedPred != 0 ||
				pps.WeightedBipredIDC != 0 || pps.RefCount[0] != 2 || pps.RefCount[1] != 1 ||
				pps.DeblockingFilterParametersPresent != 1 {
				t.Fatalf("PPS cabac/8x8/weights/refs/deblock = %d/%d/%d/%d/%d/%d/%d, want cabac=%d no-8x8 unweighted refs=2/1 deblock params",
					pps.CABAC, pps.Transform8x8Mode, pps.WeightedPred, pps.WeightedBipredIDC,
					pps.RefCount[0], pps.RefCount[1], pps.DeblockingFilterParametersPresent, tt.cabac)
			}
			ppsList[pps.PPSID] = pps
		case h264.NALIDRSlice, h264.NALSlice:
			sh, err := h264.ParseSliceHeader(nal, &ppsList)
			if err != nil {
				t.Fatal(err)
			}
			if sh.PictureStructure != h264.PictureFrame {
				t.Fatalf("slice picture = %d, want frame", sh.PictureStructure)
			}
			switch sh.SliceTypeNoS {
			case h264.PictureTypeI:
				if sh.ListCount != 0 || sh.DeblockingFilter != 0 {
					t.Fatalf("I slice lists/deblock = %d/%d, want none/disabled for flat IDR", sh.ListCount, sh.DeblockingFilter)
				}
			case h264.PictureTypeP:
				if sh.ListCount != 1 || sh.RefCount[0] != 1 || sh.DeblockingFilter != 1 ||
					sh.PredWeightTable.UseWeight != 0 || sh.PredWeightTable.UseWeightChroma != 0 {
					t.Fatalf("P slice lists/refs/deblock/weights = %d/%v/%d/%d/%d, want L0 refs=1 deblock-enabled unweighted",
						sh.ListCount, sh.RefCount, sh.DeblockingFilter, sh.PredWeightTable.UseWeight, sh.PredWeightTable.UseWeightChroma)
				}
			case h264.PictureTypeB:
				if sh.ListCount != 2 || sh.RefCount[0] != 1 || sh.RefCount[1] != 1 || sh.DeblockingFilter != 1 ||
					sh.DirectSpatialMVPred != tt.directSpatial ||
					sh.PredWeightTable.UseWeight != 0 || sh.PredWeightTable.UseWeightChroma != 0 {
					t.Fatalf("B slice lists/refs/deblock/direct/weights = %d/%v/%d/%d/%d/%d, want L0/L1 refs=1/1 deblock-enabled direct=%d unweighted",
						sh.ListCount, sh.RefCount, sh.DeblockingFilter, sh.DirectSpatialMVPred, sh.PredWeightTable.UseWeight, sh.PredWeightTable.UseWeightChroma, tt.directSpatial)
				}
			default:
				t.Fatalf("unexpected slice type %d", sh.SliceTypeNoS)
			}
			gotSlices = append(gotSlices, sh.SliceTypeNoS)
		default:
			t.Fatalf("unexpected NAL type %d in stripped fixture", nal.Type)
		}
	}
	wantSlices := []int32{h264.PictureTypeI, h264.PictureTypeP, h264.PictureTypeB}
	if len(gotSlices) != len(wantSlices) {
		t.Fatalf("slice count = %d, want %d", len(gotSlices), len(wantSlices))
	}
	for i, want := range wantSlices {
		if gotSlices[i] != want {
			t.Fatalf("slice[%d] type = %d, want %d", i, gotSlices[i], want)
		}
	}
}
