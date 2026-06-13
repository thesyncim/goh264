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
	high10PIntraMixedFrameRawSize = 3072
	high10PIntraMixedRawVideoMD5  = "79ab32c577ba4992c3c259bd3a0948ec"
)

var high10PIntraMixedFrameMD5 = []string{
	"d8763101b7caf84ef313361b2c509966",
	"bc57ab466b36d17a8f97bce6d2778fd2",
	"120c1cc35907941cf8adacb9289389a3",
	"270c2deffbbf00fbab9f02a4646f6a70",
	"288f0ebfec67eb2cd02622d6e84e4b78",
	"1dfb9457bc1c92737c21a9eeb0d3c5c0",
	"8588fa7d5b458765c9c23943d089d955",
	"1814743e97f04868d87b1060194606c7",
	"d8105649af4d4c96dc72ec39dd84186d",
	"ba305727a671c37d878ca128112dc50a",
	"8ce69b5e4ceda53f964d9187dd08754f",
	"bc6c9b3bdfb4007e95957c0d1de7bc04",
}

type high10PIntraMixedFixture struct {
	name       string
	file       string
	cabac      int32
	annexBSize int
	annexBMD5  string
}

func TestHigh10PIntraMixedFixtureSyntax(t *testing.T) {
	for _, tt := range high10PIntraMixedFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := readHigh10PIntraMixedFixture(t, tt)
			assertHigh10PIntraMixedFixtureSyntax(t, data, tt)
		})
	}
}

func TestDecodeAnnexBHigh10PIntraMixedFrames(t *testing.T) {
	for _, tt := range high10PIntraMixedFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := readHigh10PIntraMixedFixture(t, tt)
			assertHigh10PIntraMixedFixtureSyntax(t, data, tt)
			frames, err := NewDecoder().DecodeAnnexBFrames(data)
			if err != nil {
				t.Fatalf("DecodeAnnexBFrames: %v", err)
			}
			assertHigh10PIntraMixedFrames(t, frames)
		})
	}
}

func TestDecodeAVCHigh10PIntraMixedFrames(t *testing.T) {
	for _, tt := range high10PIntraMixedFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := readHigh10PIntraMixedFixture(t, tt)
			assertHigh10PIntraMixedFixtureSyntax(t, data, tt)
			for _, nalLengthSize := range []int{2, 3, 4} {
				frames, err := NewDecoder().DecodeAVCFrames(annexBToAVC(t, data, nalLengthSize), nalLengthSize)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: DecodeAVCFrames: %v", nalLengthSize, err)
				}
				assertHigh10PIntraMixedFrames(t, frames)
			}
		})
	}
}

func TestDecodeAVCWithConfigurationRecordHigh10PIntraMixedFrames(t *testing.T) {
	for _, tt := range high10PIntraMixedFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := readHigh10PIntraMixedFixture(t, tt)
			assertHigh10PIntraMixedFixtureSyntax(t, data, tt)
			for _, nalLengthSize := range []int{2, 3, 4} {
				config, packet := annexBToAVCConfigAndPacket(t, data, nalLengthSize)
				frames, err := NewDecoder().DecodeAVCFramesWithConfigurationRecord(config, packet)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: DecodeAVCFramesWithConfigurationRecord: %v", nalLengthSize, err)
				}
				assertHigh10PIntraMixedFrames(t, frames)
			}
		})
	}
}

func TestDecodeConfiguredAVCHigh10PIntraMixedFramesAcrossSamples(t *testing.T) {
	for _, tt := range high10PIntraMixedFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := readHigh10PIntraMixedFixture(t, tt)
			assertHigh10PIntraMixedFixtureSyntax(t, data, tt)
			for _, nalLengthSize := range []int{2, 3, 4} {
				config, samples := annexBToAVCConfigAndSamples(t, data, nalLengthSize)
				if len(samples) != len(high10PIntraMixedFrameMD5) {
					t.Fatalf("nalLengthSize=%d: samples = %d, want %d", nalLengthSize, len(samples), len(high10PIntraMixedFrameMD5))
				}

				dec := NewDecoder()
				if _, err := dec.ConfigureAVCDecoderConfigurationRecord(config); err != nil {
					t.Fatalf("nalLengthSize=%d: config: %v", nalLengthSize, err)
				}
				var frames []*Frame
				for i, sample := range samples {
					out, err := dec.DecodeConfiguredAVCFrames(sample)
					if err != nil {
						t.Fatalf("nalLengthSize=%d sample[%d]: DecodeConfiguredAVCFrames: %v", nalLengthSize, i, err)
					}
					frames = append(frames, out...)
				}
				out, err := dec.FlushDelayedFrames()
				if err != nil {
					t.Fatalf("nalLengthSize=%d flush: %v", nalLengthSize, err)
				}
				frames = append(frames, out...)
				assertHigh10PIntraMixedFrames(t, frames)
			}
		})
	}
}

func TestDecodeAutoConfiguredAVCHigh10PIntraMixedFramesAcrossSamples(t *testing.T) {
	for _, tt := range high10PIntraMixedFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := readHigh10PIntraMixedFixture(t, tt)
			config, samples := annexBToAVCConfigAndSamples(t, data, 4)
			if len(samples) != len(high10PIntraMixedFrameMD5) {
				t.Fatalf("samples = %d, want %d", len(samples), len(high10PIntraMixedFrameMD5))
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
			assertHigh10PIntraMixedFrames(t, frames)
		})
	}
}

func TestFFmpegRawVideoMD5OracleHigh10PIntraMixed(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}

	for _, tt := range high10PIntraMixedFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := readHigh10PIntraMixedFixture(t, tt)
			assertHigh10PIntraMixedFixtureSyntax(t, data, tt)
			path := filepath.Join("testdata", "h264", tt.file)

			framemd5 := exec.Command("ffmpeg",
				"-v", "error",
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
			for i, want := range high10PIntraMixedFrameMD5 {
				line := []byte(fmt.Sprintf("0, %10d, %10d,        1, %8d, %s", i, i, high10PIntraMixedFrameRawSize, want))
				if !bytes.Contains(framemd5Out, line) {
					t.Fatalf("frame[%d] missing %q in framemd5:\n%s", i, line, framemd5Out)
				}
			}

			rawCmd := exec.Command("ffmpeg",
				"-v", "error",
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
			if len(raw) != len(high10PIntraMixedFrameMD5)*high10PIntraMixedFrameRawSize {
				t.Fatalf("rawvideo size = %d, want %d", len(raw), len(high10PIntraMixedFrameMD5)*high10PIntraMixedFrameRawSize)
			}
			sum := md5.Sum(raw)
			if got := hex.EncodeToString(sum[:]); got != high10PIntraMixedRawVideoMD5 {
				t.Fatalf("rawvideo md5 = %s, want %s", got, high10PIntraMixedRawVideoMD5)
			}
		})
	}
}

func high10PIntraMixedFixtures() []high10PIntraMixedFixture {
	return []high10PIntraMixedFixture{
		{
			name:       "cavlc",
			file:       "high10_cavlc_p_intra_mixed.h264",
			cabac:      0,
			annexBSize: 6052,
			annexBMD5:  "2f7cf7da83f2bb10eda8092c9cc3bdc3",
		},
		{
			name:       "cabac",
			file:       "high10_cabac_p_intra_mixed.h264",
			cabac:      1,
			annexBSize: 6215,
			annexBMD5:  "f98517c8acf532fb1005b88e4235bc08",
		},
	}
}

func readHigh10PIntraMixedFixture(t *testing.T, tt high10PIntraMixedFixture) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "h264", tt.file))
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != tt.annexBSize {
		t.Fatalf("annex b size = %d, want %d", len(data), tt.annexBSize)
	}
	sum := md5.Sum(data)
	if got := hex.EncodeToString(sum[:]); got != tt.annexBMD5 {
		t.Fatalf("annex b md5 = %s, want %s", got, tt.annexBMD5)
	}
	return data
}

func assertHigh10PIntraMixedFrames(t *testing.T, frames []*Frame) {
	t.Helper()
	assertHigh10FrameMD5Strings(t, frames, high10PIntraMixedFrameMD5)
	raw := make([]byte, 0, len(frames)*high10PIntraMixedFrameRawSize)
	for i, frame := range frames {
		if frame.Width != 64 || frame.Height != 16 {
			t.Fatalf("frame[%d] size = %dx%d, want 64x16", i, frame.Width, frame.Height)
		}
		rawSize, err := frame.RawYUVSize()
		if err != nil {
			t.Fatalf("frame[%d] RawYUVSize: %v", i, err)
		}
		if rawSize != high10PIntraMixedFrameRawSize {
			t.Fatalf("frame[%d] raw yuv size = %d, want %d", i, rawSize, high10PIntraMixedFrameRawSize)
		}
		raw, err = frame.AppendRawYUVBytesLE(raw)
		if err != nil {
			t.Fatalf("frame[%d] AppendRawYUVBytesLE: %v", i, err)
		}
	}
	sum := md5.Sum(raw)
	if got := hex.EncodeToString(sum[:]); got != high10PIntraMixedRawVideoMD5 {
		t.Fatalf("rawvideo md5 = %s, want %s", got, high10PIntraMixedRawVideoMD5)
	}
}

func assertHigh10PIntraMixedFixtureSyntax(t *testing.T, data []byte, tt high10PIntraMixedFixture) {
	t.Helper()
	nals, err := h264.SplitAnnexB(data)
	if err != nil {
		t.Fatal(err)
	}

	var spsList [32]*h264.SPS
	var ppsList [256]*h264.PPS
	var gotVCL []h264.NALUnitType
	var gotSlices []int32
	var gotP int
	for _, nal := range nals {
		switch nal.Type {
		case h264.NALSPS:
			sps, err := h264.DecodeSPS(nal.RBSP)
			if err != nil {
				t.Fatal(err)
			}
			if sps.ProfileIDC != 110 || sps.Width != 64 || sps.Height != 16 ||
				sps.ChromaFormatIDC != 1 || sps.BitDepthLuma != 10 || sps.BitDepthChroma != 10 ||
				sps.FrameMBSOnlyFlag != 1 || sps.MBAFF != 0 || sps.RefFrameCount != 1 {
				t.Fatalf("SPS profile/format = %d %dx%d chroma %d depth %d/%d frame_only %d/%d refs %d, want High10 64x16 yuv420p10le frame-only refs=1",
					sps.ProfileIDC, sps.Width, sps.Height, sps.ChromaFormatIDC, sps.BitDepthLuma, sps.BitDepthChroma,
					sps.FrameMBSOnlyFlag, sps.MBAFF, sps.RefFrameCount)
			}
			spsList[sps.SPSID] = sps
		case h264.NALPPS:
			pps, err := h264.DecodePPS(nal.RBSP, &spsList)
			if err != nil {
				t.Fatal(err)
			}
			if pps.CABAC != tt.cabac || pps.Transform8x8Mode != 0 || pps.WeightedPred != 0 ||
				pps.WeightedBipredIDC != 0 || pps.RefCount[0] != 1 || pps.RefCount[1] != 1 {
				t.Fatalf("PPS cabac/8x8/weights/refs = %d/%d/%d/%d/%d/%d, want cabac=%d no-8x8 unweighted refs=1/1",
					pps.CABAC, pps.Transform8x8Mode, pps.WeightedPred, pps.WeightedBipredIDC, pps.RefCount[0], pps.RefCount[1], tt.cabac)
			}
			ppsList[pps.PPSID] = pps
		case h264.NALIDRSlice, h264.NALSlice:
			sh, err := h264.ParseSliceHeader(nal, &ppsList)
			if err != nil {
				t.Fatal(err)
			}
			if sh.PictureStructure != h264.PictureFrame || sh.DeblockingFilter != 0 {
				t.Fatalf("slice picture/deblock = %d/%d, want frame/disabled", sh.PictureStructure, sh.DeblockingFilter)
			}
			switch sh.SliceTypeNoS {
			case h264.PictureTypeI:
				if sh.ListCount != 0 {
					t.Fatalf("I slice lists = %d, want none", sh.ListCount)
				}
			case h264.PictureTypeP:
				gotP++
				if sh.ListCount != 1 || sh.RefCount[0] != 1 ||
					sh.PredWeightTable.UseWeight != 0 || sh.PredWeightTable.UseWeightChroma != 0 {
					t.Fatalf("P slice lists/refs/weights = %d/%v/%d/%d, want L0 refs=1 unweighted",
						sh.ListCount, sh.RefCount, sh.PredWeightTable.UseWeight, sh.PredWeightTable.UseWeightChroma)
				}
			default:
				t.Fatalf("unexpected slice type %d", sh.SliceTypeNoS)
			}
			gotVCL = append(gotVCL, nal.Type)
			gotSlices = append(gotSlices, sh.SliceTypeNoS)
		case h264.NALSEI:
		default:
			t.Fatalf("unexpected NAL type %d", nal.Type)
		}
	}
	if len(gotVCL) != len(high10PIntraMixedFrameMD5) || gotVCL[0] != h264.NALIDRSlice {
		t.Fatalf("VCL NALs = len %d first %d, want %d VCLs starting IDR", len(gotVCL), gotVCL[0], len(high10PIntraMixedFrameMD5))
	}
	if len(gotSlices) != len(high10PIntraMixedFrameMD5) || gotSlices[0] != h264.PictureTypeI || gotP != len(high10PIntraMixedFrameMD5)-1 {
		t.Fatalf("slice types = %v, want one I then %d P slices", gotSlices, len(high10PIntraMixedFrameMD5)-1)
	}
}
