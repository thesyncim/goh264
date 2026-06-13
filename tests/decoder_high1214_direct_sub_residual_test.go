// SPDX-License-Identifier: LGPL-2.1-or-later

package goh264_test

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/thesyncim/goh264/internal/h264"
)

type high1214DirectSubResidualCase struct {
	bitDepth     int
	entropy      string
	bitstreamMD5 string
	frameMD5     []string
	rawVideoMD5  string
}

func TestHigh1214DirectSubResidualFixtureSyntax(t *testing.T) {
	for _, tt := range high1214DirectSubResidualCases() {
		t.Run(high1214DirectSubResidualCaseName(tt), func(t *testing.T) {
			data := high1214DirectSubResidualFixture(t, tt)
			assertHigh1214DirectSubResidualFixtureSyntax(t, data, tt)
		})
	}
}

func TestDecodeAnnexBHigh1214DirectSubResidualFrames(t *testing.T) {
	for _, tt := range high1214DirectSubResidualCases() {
		t.Run(high1214DirectSubResidualCaseName(tt), func(t *testing.T) {
			data := high1214DirectSubResidualFixture(t, tt)
			assertHigh1214DirectSubResidualFixtureSyntax(t, data, tt)

			frames, err := NewDecoder().DecodeAnnexBFrames(data)
			if err != nil {
				t.Fatalf("DecodeAnnexBFrames: %v", err)
			}
			assertHigh1214DirectSubResidualFrames(t, frames, tt)
		})
	}
}

func TestDecodeAVCHigh1214DirectSubResidualFrames(t *testing.T) {
	for _, tt := range high1214DirectSubResidualCases() {
		t.Run(high1214DirectSubResidualCaseName(tt), func(t *testing.T) {
			data := high1214DirectSubResidualFixture(t, tt)
			assertHigh1214DirectSubResidualFixtureSyntax(t, data, tt)

			for _, nalLengthSize := range []int{2, 3, 4} {
				frames, err := NewDecoder().DecodeAVCFrames(annexBToAVC(t, data, nalLengthSize), nalLengthSize)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: DecodeAVCFrames: %v", nalLengthSize, err)
				}
				assertHigh1214DirectSubResidualFrames(t, frames, tt)
			}
		})
	}
}

func TestDecodeAVCConfigurationRecordHigh1214DirectSubResidualFrames(t *testing.T) {
	for _, tt := range high1214DirectSubResidualCases() {
		t.Run(high1214DirectSubResidualCaseName(tt), func(t *testing.T) {
			data := high1214DirectSubResidualFixture(t, tt)
			assertHigh1214DirectSubResidualFixtureSyntax(t, data, tt)

			for _, nalLengthSize := range []int{2, 3, 4} {
				config, packet := annexBToAVCConfigAndPacket(t, data, nalLengthSize)
				frames, err := NewDecoder().DecodeAVCFramesWithConfigurationRecord(config, packet)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: DecodeAVCFramesWithConfigurationRecord: %v", nalLengthSize, err)
				}
				assertHigh1214DirectSubResidualFrames(t, frames, tt)
			}
		})
	}
}

func TestDecodeConfiguredAVCHigh1214DirectSubResidualFramesAcrossSamplesFlush(t *testing.T) {
	for _, tt := range high1214DirectSubResidualCases() {
		t.Run(high1214DirectSubResidualCaseName(tt), func(t *testing.T) {
			data := high1214DirectSubResidualFixture(t, tt)
			assertHigh1214DirectSubResidualFixtureSyntax(t, data, tt)

			for _, nalLengthSize := range []int{2, 3, 4} {
				config, samples := annexBToAVCConfigAndSamples(t, data, nalLengthSize)
				if len(samples) != len(tt.frameMD5) {
					t.Fatalf("nalLengthSize=%d: samples = %d, want %d", nalLengthSize, len(samples), len(tt.frameMD5))
				}

				dec := NewDecoder()
				if _, err := dec.ConfigureAVCDecoderConfigurationRecord(config); err != nil {
					t.Fatalf("nalLengthSize=%d config: %v", nalLengthSize, err)
				}
				var frames []*Frame
				var frameCounts []int
				for i, sample := range samples {
					out, err := dec.DecodeConfiguredAVCFrames(sample)
					if err != nil {
						t.Fatalf("nalLengthSize=%d sample[%d]: DecodeConfiguredAVCFrames: %v", nalLengthSize, i, err)
					}
					frameCounts = append(frameCounts, len(out))
					frames = append(frames, out...)
				}
				out, err := dec.FlushDelayedFrames()
				if err != nil {
					t.Fatalf("nalLengthSize=%d flush: %v", nalLengthSize, err)
				}
				frameCounts = append(frameCounts, len(out))
				assertHigh1214DirectSubResidualConfiguredSampleCounts(t, tt, nalLengthSize, frameCounts)
				frames = append(frames, out...)
				assertHigh1214DirectSubResidualFrames(t, frames, tt)
			}
		})
	}
}

func TestDecodeAutoHigh1214DirectSubResidualFramesAcrossSamplesFlush(t *testing.T) {
	for _, tt := range high1214DirectSubResidualCases() {
		t.Run(high1214DirectSubResidualCaseName(tt), func(t *testing.T) {
			data := high1214DirectSubResidualFixture(t, tt)
			assertHigh1214DirectSubResidualFixtureSyntax(t, data, tt)

			for _, nalLengthSize := range []int{2, 3, 4} {
				config, samples := annexBToAVCConfigAndSamples(t, data, nalLengthSize)
				if len(samples) != len(tt.frameMD5) {
					t.Fatalf("nalLengthSize=%d: samples = %d, want %d", nalLengthSize, len(samples), len(tt.frameMD5))
				}

				dec := NewDecoder()
				out, err := dec.DecodeFrames(config)
				if err != nil {
					t.Fatalf("nalLengthSize=%d config: %v", nalLengthSize, err)
				}
				if len(out) != 0 {
					t.Fatalf("nalLengthSize=%d config produced %d frames", nalLengthSize, len(out))
				}
				var frames []*Frame
				var frameCounts []int
				for i, sample := range samples {
					out, err := dec.DecodeFrames(sample)
					if err != nil {
						t.Fatalf("nalLengthSize=%d sample[%d]: DecodeFrames: %v", nalLengthSize, i, err)
					}
					frameCounts = append(frameCounts, len(out))
					frames = append(frames, out...)
				}
				out, err = dec.DecodeFrames(nil)
				if err != nil {
					t.Fatalf("nalLengthSize=%d flush: %v", nalLengthSize, err)
				}
				frameCounts = append(frameCounts, len(out))
				assertHigh1214DirectSubResidualConfiguredSampleCounts(t, tt, nalLengthSize, frameCounts)
				frames = append(frames, out...)

				out, err = dec.DecodeFrames(nil)
				if err != nil {
					t.Fatalf("nalLengthSize=%d second flush: %v", nalLengthSize, err)
				}
				if len(out) != 0 {
					t.Fatalf("nalLengthSize=%d second flush produced %d frames", nalLengthSize, len(out))
				}
				assertHigh1214DirectSubResidualFrames(t, frames, tt)
			}
		})
	}
}

func TestFFmpegRawVideoFrameMD5OracleHigh1214DirectSubResidual(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}

	for _, tt := range high1214DirectSubResidualCases() {
		t.Run(high1214DirectSubResidualCaseName(tt), func(t *testing.T) {
			data := high1214DirectSubResidualFixture(t, tt)
			assertHigh1214DirectSubResidualFixtureSyntax(t, data, tt)
			assertFFmpegHigh1214DirectSubResidualRawVideoOracle(t, data, tt)
		})
	}
}

func high1214DirectSubResidualCases() []high1214DirectSubResidualCase {
	return []high1214DirectSubResidualCase{
		{
			bitDepth:     12,
			entropy:      "cavlc",
			bitstreamMD5: "4cedf9f993c60d42654feda919063dbb",
			frameMD5: []string{
				"941341cdfb37f5687de3a785d311fe7e",
				"6163f46d717ef6750a7243042f470b7f",
				"941341cdfb37f5687de3a785d311fe7e",
			},
			rawVideoMD5: "8c9f16ddef20fa797dadbffea09dbcc0",
		},
		{
			bitDepth:     14,
			entropy:      "cavlc",
			bitstreamMD5: "f81088389c2b5088b82fb349cfcfa640",
			frameMD5: []string{
				"2163eb71b459070d04147dc124aac7c8",
				"95d31e69959b9947cb4165c06812c66b",
				"2163eb71b459070d04147dc124aac7c8",
			},
			rawVideoMD5: "dc70104d653095507fc2aa2bcb2f2c08",
		},
		{
			bitDepth:     12,
			entropy:      "cabac",
			bitstreamMD5: "5872c8413bce732783a7168572790551",
			frameMD5: []string{
				"941341cdfb37f5687de3a785d311fe7e",
				"6163f46d717ef6750a7243042f470b7f",
				"941341cdfb37f5687de3a785d311fe7e",
			},
			rawVideoMD5: "8c9f16ddef20fa797dadbffea09dbcc0",
		},
		{
			bitDepth:     14,
			entropy:      "cabac",
			bitstreamMD5: "f8a863c76d46d57dfffa7868f06a3dc1",
			frameMD5: []string{
				"2163eb71b459070d04147dc124aac7c8",
				"95d31e69959b9947cb4165c06812c66b",
				"2163eb71b459070d04147dc124aac7c8",
			},
			rawVideoMD5: "dc70104d653095507fc2aa2bcb2f2c08",
		},
	}
}

func high1214DirectSubResidualFixture(t *testing.T, tt high1214DirectSubResidualCase) []byte {
	t.Helper()
	var out []byte
	switch tt.entropy {
	case "cavlc":
		out = highCAVLCBRewriteAnnexB(t, readHigh10DirectSubResidualCAVLCFixture(t), tt.bitDepth, false)
	case "cabac":
		out = highCABACBRewriteAnnexB(t, readHigh10DirectSubResidualCABACFixture(t), tt.bitDepth, false)
	default:
		t.Fatalf("unsupported entropy %q", tt.entropy)
	}
	sum := md5.Sum(out)
	if got := hex.EncodeToString(sum[:]); got != tt.bitstreamMD5 {
		t.Fatalf("%s generated bitstream md5 = %s, want %s", high1214DirectSubResidualCaseName(tt), got, tt.bitstreamMD5)
	}
	return out
}

func high1214DirectSubResidualCaseName(tt high1214DirectSubResidualCase) string {
	return fmt.Sprintf("high%d-%s", tt.bitDepth, tt.entropy)
}

func assertHigh1214DirectSubResidualFixtureSyntax(t *testing.T, data []byte, tt high1214DirectSubResidualCase) {
	t.Helper()
	nals, err := h264.SplitAnnexB(data)
	if err != nil {
		t.Fatal(err)
	}

	wantCABAC := int32(0)
	if tt.entropy == "cabac" {
		wantCABAC = 1
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
			if sps.ProfileIDC != 244 || sps.Width != 16 || sps.Height != 16 ||
				sps.ChromaFormatIDC != 1 || sps.BitDepthLuma != int32(tt.bitDepth) || sps.BitDepthChroma != int32(tt.bitDepth) ||
				sps.FrameMBSOnlyFlag != 1 || sps.MBAFF != 0 || sps.RefFrameCount != 2 ||
				sps.Direct8x8InferenceFlag != 1 {
				t.Fatalf("SPS profile/format = %d %dx%d chroma %d depth %d/%d frame_only %d/%d refs %d direct8x8 %d, want High%d 16x16 yuv420p%dle frame-only refs=2 direct8x8",
					sps.ProfileIDC, sps.Width, sps.Height, sps.ChromaFormatIDC, sps.BitDepthLuma, sps.BitDepthChroma,
					sps.FrameMBSOnlyFlag, sps.MBAFF, sps.RefFrameCount, sps.Direct8x8InferenceFlag, tt.bitDepth, tt.bitDepth)
			}
			spsList[sps.SPSID] = sps
		case h264.NALPPS:
			pps, err := h264.DecodePPS(nal.RBSP, &spsList)
			if err != nil {
				t.Fatal(err)
			}
			if pps.CABAC != wantCABAC || pps.Transform8x8Mode != 0 || pps.WeightedPred != 0 ||
				pps.WeightedBipredIDC != 0 || pps.RefCount[0] != 2 || pps.RefCount[1] != 1 {
				t.Fatalf("PPS cabac/8x8/weights/refs = %d/%d/%d/%d/%d/%d, want %s no-8x8 unweighted refs=2/1",
					pps.CABAC, pps.Transform8x8Mode, pps.WeightedPred, pps.WeightedBipredIDC, pps.RefCount[0], pps.RefCount[1], tt.entropy)
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
			if sh.SliceTypeNoS == h264.PictureTypeB {
				if sh.ListCount != 2 || sh.RefCount[0] != 1 || sh.RefCount[1] != 1 ||
					sh.DirectSpatialMVPred != 0 ||
					sh.PredWeightTable.UseWeight != 0 || sh.PredWeightTable.UseWeightChroma != 0 {
					t.Fatalf("B slice lists/refs/direct/weights = %d/%v/%d/%d/%d, want L0/L1 refs=1/1 temporal unweighted",
						sh.ListCount, sh.RefCount, sh.DirectSpatialMVPred, sh.PredWeightTable.UseWeight, sh.PredWeightTable.UseWeightChroma)
				}
				if tt.entropy == "cabac" {
					assertHigh10DirectSubResidualCABACPayload(t, nal.Raw)
				} else {
					assertHigh10DirectSubResidualCAVLCPayload(t, nal.Raw)
				}
			}
			gotSlices = append(gotSlices, sh.SliceTypeNoS)
		case h264.NALSEI:
		default:
			t.Fatalf("unexpected NAL type %d", nal.Type)
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

func assertHigh1214DirectSubResidualConfiguredSampleCounts(t *testing.T, tt high1214DirectSubResidualCase, nalLengthSize int, got []int) {
	t.Helper()
	want := []int{0, 1, 1, 1}
	if len(got) != len(want) {
		t.Fatalf("%s nalLengthSize=%d configured sample/flush counts = %v, want %v",
			high1214DirectSubResidualCaseName(tt), nalLengthSize, got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%s nalLengthSize=%d configured sample/flush counts = %v, want %v",
				high1214DirectSubResidualCaseName(tt), nalLengthSize, got, want)
		}
	}
}

func assertHigh1214DirectSubResidualFrames(t *testing.T, frames []*Frame, tt high1214DirectSubResidualCase) {
	t.Helper()
	if len(frames) != len(tt.frameMD5) {
		t.Fatalf("frames = %d, want %d", len(frames), len(tt.frameMD5))
	}
	wantPixFmt := fmt.Sprintf("yuv420p%dle", tt.bitDepth)
	var rawVideo []byte
	for i, frame := range frames {
		if frame == nil {
			t.Fatalf("frame[%d] is nil", i)
		}
		if frame.Width != 16 || frame.Height != 16 ||
			frame.ChromaFormatIDC != 1 || frame.BitDepthLuma != tt.bitDepth || frame.BitDepthChroma != tt.bitDepth {
			t.Fatalf("frame[%d] format = %dx%d chroma %d depth %d/%d, want 16x16 %s",
				i, frame.Width, frame.Height, frame.ChromaFormatIDC, frame.BitDepthLuma, frame.BitDepthChroma, wantPixFmt)
		}
		if pixFmt, err := frame.RawPixelFormat(); err != nil || pixFmt != wantPixFmt {
			t.Fatalf("frame[%d] RawPixelFormat = %q/%v, want %s/nil", i, pixFmt, err, wantPixFmt)
		}
		if size, err := frame.RawYUVSize(); err != nil || size != high10DirectSubResidualFrameRawSize {
			t.Fatalf("frame[%d] RawYUVSize = %d/%v, want %d/nil", i, size, err, high10DirectSubResidualFrameRawSize)
		}
		raw, err := frame.AppendRawYUVBytesLE(nil)
		if err != nil {
			t.Fatalf("frame[%d] AppendRawYUVBytesLE: %v", i, err)
		}
		rawVideo = append(rawVideo, raw...)
		sum := md5.Sum(raw)
		if got := hex.EncodeToString(sum[:]); got != tt.frameMD5[i] {
			t.Fatalf("frame[%d] raw md5 = %s, want %s", i, got, tt.frameMD5[i])
		}
		if _, err := frame.AppendRawYUV(nil); err != ErrUnsupported {
			t.Fatalf("frame[%d] AppendRawYUV high%d error = %v, want ErrUnsupported", i, tt.bitDepth, err)
		}
	}
	sum := md5.Sum(rawVideo)
	if got := hex.EncodeToString(sum[:]); got != tt.rawVideoMD5 {
		t.Fatalf("rawvideo md5 = %s, want %s", got, tt.rawVideoMD5)
	}
}

func assertFFmpegHigh1214DirectSubResidualRawVideoOracle(t *testing.T, data []byte, tt high1214DirectSubResidualCase) {
	t.Helper()
	path := writeTempH264(t, data)
	pixFmt := fmt.Sprintf("yuv420p%dle", tt.bitDepth)
	framemd5 := exec.Command("ffmpeg",
		"-v", "error",
		"-xerror",
		"-f", "h264",
		"-i", path,
		"-an", "-sn", "-dn",
		"-pix_fmt", pixFmt,
		"-f", "framemd5",
		"-",
	)
	framemd5Out, err := framemd5.Output()
	if err != nil {
		t.Fatalf("ffmpeg framemd5: %v", err)
	}
	for i, want := range tt.frameMD5 {
		line := []byte(fmt.Sprintf("0, %10d, %10d,        1, %8d, %s", i, i, high10DirectSubResidualFrameRawSize, want))
		if !bytes.Contains(framemd5Out, line) {
			t.Fatalf("frame[%d] missing %q in framemd5:\n%s", i, line, framemd5Out)
		}
	}

	rawvideo := exec.Command("ffmpeg",
		"-v", "error",
		"-xerror",
		"-f", "h264",
		"-i", path,
		"-an", "-sn", "-dn",
		"-pix_fmt", pixFmt,
		"-f", "rawvideo",
		"-",
	)
	raw, err := rawvideo.Output()
	if err != nil {
		t.Fatalf("ffmpeg rawvideo: %v", err)
	}
	wantSize := len(tt.frameMD5) * high10DirectSubResidualFrameRawSize
	if len(raw) != wantSize {
		t.Fatalf("rawvideo size = %d, want %d", len(raw), wantSize)
	}
	sum := md5.Sum(raw)
	if got := hex.EncodeToString(sum[:]); got != tt.rawVideoMD5 {
		t.Fatalf("rawvideo md5 = %s, want %s", got, tt.rawVideoMD5)
	}
}
