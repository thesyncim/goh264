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
	high10ImplicitWeightedBFrameRawSize = 768
	high10ImplicitWeightedBRawVideoMD5  = "fb94a7906e135740b49588c257f4bc15"
)

var high10ImplicitWeightedBFrameMD5 = []string{
	"857cc91515b2182f4444a4d746b9d721",
	"734370de9ff1562a091bd9da2e7388f4",
	"0278043f7918f89fb326a88e60c9c01b",
	"1b742676a4555b46109892813b9feaa6",
	"aed2dfa63ba343c3f2ef494bff5e3f74",
}

type high10ImplicitWeightedBFixture struct {
	name        string
	file        string
	cabac       int32
	annexBSize  int
	annexBMD5   string
	rawVideoMD5 string
}

func TestHigh10ImplicitWeightedBFixtureSyntax(t *testing.T) {
	for _, tt := range high10ImplicitWeightedBFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := readHigh10ImplicitWeightedBFixture(t, tt)
			if len(data) != tt.annexBSize {
				t.Fatalf("annex b size = %d, want %d", len(data), tt.annexBSize)
			}
			sum := md5.Sum(data)
			if got := hex.EncodeToString(sum[:]); got != tt.annexBMD5 {
				t.Fatalf("annex b md5 = %s, want %s", got, tt.annexBMD5)
			}
			assertHigh10ImplicitWeightedBFixtureSyntax(t, data, tt)
		})
	}
}

func TestDecodeAnnexBHigh10ImplicitWeightedBFrames(t *testing.T) {
	for _, tt := range high10ImplicitWeightedBFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := readHigh10ImplicitWeightedBFixture(t, tt)
			assertHigh10ImplicitWeightedBFixtureSyntax(t, data, tt)
			frames, err := NewDecoder().DecodeAnnexBFrames(data)
			if err != nil {
				t.Fatalf("DecodeAnnexBFrames: %v", err)
			}
			assertHigh10ImplicitWeightedBFrames(t, frames, tt.rawVideoMD5)
		})
	}
}

func TestDecodeAVCHigh10ImplicitWeightedBFrames(t *testing.T) {
	for _, tt := range high10ImplicitWeightedBFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := readHigh10ImplicitWeightedBFixture(t, tt)
			assertHigh10ImplicitWeightedBFixtureSyntax(t, data, tt)
			for _, nalLengthSize := range []int{2, 3, 4} {
				frames, err := NewDecoder().DecodeAVCFrames(annexBToAVC(t, data, nalLengthSize), nalLengthSize)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: DecodeAVCFrames: %v", nalLengthSize, err)
				}
				assertHigh10ImplicitWeightedBFrames(t, frames, tt.rawVideoMD5)
			}
		})
	}
}

func TestDecodeAVCWithConfigurationRecordHigh10ImplicitWeightedBFrames(t *testing.T) {
	for _, tt := range high10ImplicitWeightedBFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := readHigh10ImplicitWeightedBFixture(t, tt)
			assertHigh10ImplicitWeightedBFixtureSyntax(t, data, tt)
			for _, nalLengthSize := range []int{2, 3, 4} {
				config, packet := annexBToAVCConfigAndPacket(t, data, nalLengthSize)
				frames, err := NewDecoder().DecodeAVCFramesWithConfigurationRecord(config, packet)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: DecodeAVCFramesWithConfigurationRecord: %v", nalLengthSize, err)
				}
				assertHigh10ImplicitWeightedBFrames(t, frames, tt.rawVideoMD5)
			}
		})
	}
}

func TestDecodeConfiguredAVCHigh10ImplicitWeightedBFramesAcrossSamplesFlush(t *testing.T) {
	for _, tt := range high10ImplicitWeightedBFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := readHigh10ImplicitWeightedBFixture(t, tt)
			assertHigh10ImplicitWeightedBFixtureSyntax(t, data, tt)
			for _, nalLengthSize := range []int{2, 3, 4} {
				config, samples := annexBToAVCConfigAndSamples(t, data, nalLengthSize)
				if len(samples) != len(high10ImplicitWeightedBFrameMD5) {
					t.Fatalf("nalLengthSize=%d: samples = %d, want %d", nalLengthSize, len(samples), len(high10ImplicitWeightedBFrameMD5))
				}

				dec := NewDecoder()
				if _, err := dec.ParseAVCDecoderConfigurationRecord(config); err != nil {
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
				assertHigh10ImplicitWeightedBFrames(t, frames, tt.rawVideoMD5)
			}
		})
	}
}

func TestDecodeAutoConfiguredAVCHigh10ImplicitWeightedBFramesAcrossSamplesFlush(t *testing.T) {
	for _, tt := range high10ImplicitWeightedBFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := readHigh10ImplicitWeightedBFixture(t, tt)
			config, samples := annexBToAVCConfigAndSamples(t, data, 4)
			if len(samples) != len(high10ImplicitWeightedBFrameMD5) {
				t.Fatalf("samples = %d, want %d", len(samples), len(high10ImplicitWeightedBFrameMD5))
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
			assertHigh10ImplicitWeightedBFrames(t, frames, tt.rawVideoMD5)
		})
	}
}

func TestFFmpegRawVideoMD5OracleHigh10ImplicitWeightedB(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") == "" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}
	for _, tt := range high10ImplicitWeightedBFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := readHigh10ImplicitWeightedBFixture(t, tt)
			assertHigh10ImplicitWeightedBFixtureSyntax(t, data, tt)
			path := filepath.Join("testdata", "h264", tt.file)

			framemd5 := exec.Command("ffmpeg",
				"-hide_banner", "-v", "error",
				"-i", path,
				"-f", "framemd5",
				"-pix_fmt", "yuv420p10le",
				"-")
			framemd5Out, err := framemd5.Output()
			if err != nil {
				t.Fatalf("ffmpeg framemd5: %v", err)
			}
			for i, want := range high10ImplicitWeightedBFrameMD5 {
				line := []byte(fmt.Sprintf("0, %10d, %10d,        1, %8d, %s", i, i, high10ImplicitWeightedBFrameRawSize, want))
				if !bytes.Contains(framemd5Out, line) {
					t.Fatalf("frame[%d] missing %q in framemd5:\n%s", i, line, framemd5Out)
				}
			}

			rawCmd := exec.Command("ffmpeg",
				"-hide_banner", "-v", "error",
				"-i", path,
				"-f", "rawvideo",
				"-pix_fmt", "yuv420p10le",
				"-")
			raw, err := rawCmd.Output()
			if err != nil {
				t.Fatalf("ffmpeg rawvideo: %v", err)
			}
			if len(raw) != len(high10ImplicitWeightedBFrameMD5)*high10ImplicitWeightedBFrameRawSize {
				t.Fatalf("rawvideo size = %d, want %d", len(raw), len(high10ImplicitWeightedBFrameMD5)*high10ImplicitWeightedBFrameRawSize)
			}
			sum := md5.Sum(raw)
			if got := hex.EncodeToString(sum[:]); got != tt.rawVideoMD5 {
				t.Fatalf("rawvideo md5 = %s, want %s", got, tt.rawVideoMD5)
			}
		})
	}
}

func high10ImplicitWeightedBFixtures() []high10ImplicitWeightedBFixture {
	return []high10ImplicitWeightedBFixture{
		{
			name:        "cavlc",
			file:        "high10_implicit_weight_b_cavlc.h264",
			cabac:       0,
			annexBSize:  907,
			annexBMD5:   "41bfa783c0361d76fbc8e0df36a6edca",
			rawVideoMD5: high10ImplicitWeightedBRawVideoMD5,
		},
		{
			name:        "cabac",
			file:        "high10_implicit_weight_b_cabac.h264",
			cabac:       1,
			annexBSize:  845,
			annexBMD5:   "5865569a46cdb4f1692f1a1a589cd16b",
			rawVideoMD5: high10ImplicitWeightedBRawVideoMD5,
		},
	}
}

func readHigh10ImplicitWeightedBFixture(t *testing.T, tt high10ImplicitWeightedBFixture) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "h264", tt.file))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func assertHigh10ImplicitWeightedBFrames(t *testing.T, frames []*Frame, wantRawVideoMD5 string) {
	t.Helper()
	if len(frames) != len(high10ImplicitWeightedBFrameMD5) {
		t.Fatalf("frames = %d, want %d", len(frames), len(high10ImplicitWeightedBFrameMD5))
	}
	assertHigh10FrameMD5Strings(t, frames, high10ImplicitWeightedBFrameMD5)
	raw := make([]byte, 0, len(frames)*high10ImplicitWeightedBFrameRawSize)
	for i, frame := range frames {
		if frame.Width != 16 || frame.Height != 16 {
			t.Fatalf("frame[%d] size = %dx%d, want 16x16", i, frame.Width, frame.Height)
		}
		pixFmt, err := frame.RawPixelFormat()
		if err != nil {
			t.Fatalf("frame[%d] RawPixelFormat: %v", i, err)
		}
		bytesPerSample, err := frame.BytesPerSample()
		if err != nil {
			t.Fatalf("frame[%d] BytesPerSample: %v", i, err)
		}
		if pixFmt != "yuv420p10le" || bytesPerSample != 2 {
			t.Fatalf("frame[%d] raw format/sample bytes = %s/%d, want yuv420p10le/2", i, pixFmt, bytesPerSample)
		}
		rawSize, err := frame.RawYUVSize()
		if err != nil {
			t.Fatalf("frame[%d] RawYUVSize: %v", i, err)
		}
		if rawSize != high10ImplicitWeightedBFrameRawSize {
			t.Fatalf("frame[%d] raw yuv size = %d, want %d", i, rawSize, high10ImplicitWeightedBFrameRawSize)
		}
		raw, err = frame.AppendRawYUVBytesLE(raw)
		if err != nil {
			t.Fatalf("frame[%d] append raw yuv: %v", i, err)
		}
	}
	sum := md5.Sum(raw)
	if got := hex.EncodeToString(sum[:]); got != wantRawVideoMD5 {
		t.Fatalf("rawvideo md5 = %s, want %s", got, wantRawVideoMD5)
	}
}

func assertHigh10ImplicitWeightedBFixtureSyntax(t *testing.T, data []byte, tt high10ImplicitWeightedBFixture) {
	t.Helper()
	nals, err := h264.SplitAnnexB(data)
	if err != nil {
		t.Fatal(err)
	}

	var spsList [32]*h264.SPS
	var ppsList [256]*h264.PPS
	var gotSlices []int32
	var gotB int
	for _, nal := range nals {
		switch nal.Type {
		case h264.NALSPS:
			sps, err := h264.DecodeSPS(nal.RBSP)
			if err != nil {
				t.Fatal(err)
			}
			if sps.ProfileIDC != 110 || sps.Width != 16 || sps.Height != 16 ||
				sps.ChromaFormatIDC != 1 || sps.BitDepthLuma != 10 || sps.BitDepthChroma != 10 ||
				sps.FrameMBSOnlyFlag != 1 || sps.MBAFF != 0 || sps.Direct8x8InferenceFlag != 1 {
				t.Fatalf("SPS profile/format = %d %dx%d chroma %d depth %d/%d frame_only %d/%d direct8x8 %d, want High10 16x16 yuv420p10le frame-only direct8x8",
					sps.ProfileIDC, sps.Width, sps.Height, sps.ChromaFormatIDC, sps.BitDepthLuma, sps.BitDepthChroma,
					sps.FrameMBSOnlyFlag, sps.MBAFF, sps.Direct8x8InferenceFlag)
			}
			spsList[sps.SPSID] = sps
		case h264.NALPPS:
			pps, err := h264.DecodePPS(nal.RBSP, &spsList)
			if err != nil {
				t.Fatal(err)
			}
			if pps.CABAC != tt.cabac || pps.Transform8x8Mode != 0 || pps.WeightedPred != 0 ||
				pps.WeightedBipredIDC != 2 || pps.RefCount[0] != 1 || pps.RefCount[1] != 1 {
				t.Fatalf("PPS cabac/8x8/weights/refs = %d/%d/%d/%d/%d/%d, want cabac=%d no-8x8 implicit-B refs=1/1",
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
				if sh.ListCount != 1 || sh.RefCount[0] != 1 ||
					sh.PredWeightTable.UseWeight != 0 || sh.PredWeightTable.UseWeightChroma != 0 {
					t.Fatalf("P slice lists/refs/weights = %d/%v/%d/%d, want L0 refs=1 unweighted",
						sh.ListCount, sh.RefCount, sh.PredWeightTable.UseWeight, sh.PredWeightTable.UseWeightChroma)
				}
			case h264.PictureTypeB:
				gotB++
				if sh.ListCount != 2 || sh.RefCount[0] != 1 || sh.RefCount[1] != 1 ||
					sh.DirectSpatialMVPred != 0 ||
					sh.PredWeightTable.UseWeight != 0 || sh.PredWeightTable.UseWeightChroma != 0 {
					t.Fatalf("B slice lists/refs/direct/serialized weights = %d/%v/%d/%d/%d, want L0/L1 refs=1/1 temporal implicit weights only",
						sh.ListCount, sh.RefCount, sh.DirectSpatialMVPred, sh.PredWeightTable.UseWeight, sh.PredWeightTable.UseWeightChroma)
				}
			default:
				t.Fatalf("unexpected slice type %d", sh.SliceTypeNoS)
			}
			gotSlices = append(gotSlices, sh.SliceTypeNoS)
		case h264.NALSEI:
		default:
			t.Fatalf("unexpected NAL type %d", nal.Type)
		}
	}
	wantSlices := []int32{h264.PictureTypeI, h264.PictureTypeP, h264.PictureTypeB, h264.PictureTypeB, h264.PictureTypeP}
	if len(gotSlices) != len(wantSlices) {
		t.Fatalf("slice count = %d, want %d", len(gotSlices), len(wantSlices))
	}
	for i, want := range wantSlices {
		if gotSlices[i] != want {
			t.Fatalf("slice[%d] type = %d, want %d", i, gotSlices[i], want)
		}
	}
	if gotB != 2 {
		t.Fatalf("B slices = %d, want 2", gotB)
	}
}
