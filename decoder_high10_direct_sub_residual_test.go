// SPDX-License-Identifier: LGPL-2.1-or-later

package goh264

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
	high10DirectSubResidualCAVLCFile       = "high10_direct_sub_residual_cavlc.h264"
	high10DirectSubResidualCAVLCAnnexBSize = 706
	high10DirectSubResidualCAVLCAnnexBMD5  = "052417fe695d2d64889b9828a2e25f26"
	high10DirectSubResidualFrameRawSize    = 768
	high10DirectSubResidualRawVideoMD5     = "e639fe5788451a5d74a77c98c214bfd3"
)

var high10DirectSubResidualFrameMD5 = []string{
	"d73be6c1b3e4082e402d67d810323786",
	"ce9b035164407d7c9532c31a0c08c9b1",
	"d73be6c1b3e4082e402d67d810323786",
}

func TestHigh10DirectSubResidualCAVLCFixtureSyntax(t *testing.T) {
	data := readHigh10DirectSubResidualCAVLCFixture(t)
	if len(data) != high10DirectSubResidualCAVLCAnnexBSize {
		t.Fatalf("annex b size = %d, want %d", len(data), high10DirectSubResidualCAVLCAnnexBSize)
	}
	sum := md5.Sum(data)
	if got := hex.EncodeToString(sum[:]); got != high10DirectSubResidualCAVLCAnnexBMD5 {
		t.Fatalf("annex b md5 = %s, want %s", got, high10DirectSubResidualCAVLCAnnexBMD5)
	}
	assertHigh10DirectSubResidualCAVLCFixtureSyntax(t, data)
}

func TestDecodeAnnexBHigh10DirectSubResidualCAVLCFrames(t *testing.T) {
	data := readHigh10DirectSubResidualCAVLCFixture(t)
	assertHigh10DirectSubResidualCAVLCFixtureSyntax(t, data)
	frames, err := NewDecoder().DecodeAnnexBFrames(data)
	if err != nil {
		t.Fatalf("DecodeAnnexBFrames: %v", err)
	}
	assertHigh10DirectSubResidualFrames(t, frames)
}

func TestDecodeAVCHigh10DirectSubResidualCAVLCFrames(t *testing.T) {
	data := readHigh10DirectSubResidualCAVLCFixture(t)
	assertHigh10DirectSubResidualCAVLCFixtureSyntax(t, data)
	for _, nalLengthSize := range []int{2, 3, 4} {
		frames, err := NewDecoder().DecodeAVCFrames(annexBToAVC(t, data, nalLengthSize), nalLengthSize)
		if err != nil {
			t.Fatalf("nalLengthSize=%d: DecodeAVCFrames: %v", nalLengthSize, err)
		}
		assertHigh10DirectSubResidualFrames(t, frames)
	}
}

func TestDecodeConfiguredAVCHigh10DirectSubResidualCAVLCFramesAcrossSamplesFlush(t *testing.T) {
	data := readHigh10DirectSubResidualCAVLCFixture(t)
	assertHigh10DirectSubResidualCAVLCFixtureSyntax(t, data)
	for _, nalLengthSize := range []int{2, 3, 4} {
		config, samples := annexBToAVCConfigAndSamples(t, data, nalLengthSize)
		if len(samples) != len(high10DirectSubResidualFrameMD5) {
			t.Fatalf("nalLengthSize=%d: samples = %d, want %d", nalLengthSize, len(samples), len(high10DirectSubResidualFrameMD5))
		}

		dec := NewDecoder()
		if _, err := dec.ParseAVCDecoderConfigurationRecord(config); err != nil {
			t.Fatalf("nalLengthSize=%d config: %v", nalLengthSize, err)
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
		assertHigh10DirectSubResidualFrames(t, frames)
	}
}

func TestFFmpegRawVideoMD5OracleHigh10DirectSubResidualCAVLC(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}
	data := readHigh10DirectSubResidualCAVLCFixture(t)
	assertHigh10DirectSubResidualCAVLCFixtureSyntax(t, data)
	path := filepath.Join("testdata", "h264", high10DirectSubResidualCAVLCFile)

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
	for i, want := range high10DirectSubResidualFrameMD5 {
		line := []byte(fmt.Sprintf("0, %10d, %10d,        1, %8d, %s", i, i, high10DirectSubResidualFrameRawSize, want))
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
	if len(raw) != len(high10DirectSubResidualFrameMD5)*high10DirectSubResidualFrameRawSize {
		t.Fatalf("rawvideo size = %d, want %d", len(raw), len(high10DirectSubResidualFrameMD5)*high10DirectSubResidualFrameRawSize)
	}
	sum := md5.Sum(raw)
	if got := hex.EncodeToString(sum[:]); got != high10DirectSubResidualRawVideoMD5 {
		t.Fatalf("rawvideo md5 = %s, want %s", got, high10DirectSubResidualRawVideoMD5)
	}
}

func readHigh10DirectSubResidualCAVLCFixture(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "h264", high10DirectSubResidualCAVLCFile))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func assertHigh10DirectSubResidualFrames(t *testing.T, frames []*Frame) {
	t.Helper()
	if len(frames) != len(high10DirectSubResidualFrameMD5) {
		t.Fatalf("frames = %d, want %d", len(frames), len(high10DirectSubResidualFrameMD5))
	}
	assertHigh10FrameMD5Strings(t, frames, high10DirectSubResidualFrameMD5)
	var raw []byte
	for _, frame := range frames {
		var err error
		raw, err = frame.AppendRawYUVBytesLE(raw)
		if err != nil {
			t.Fatal(err)
		}
	}
	sum := md5.Sum(raw)
	if got := hex.EncodeToString(sum[:]); got != high10DirectSubResidualRawVideoMD5 {
		t.Fatalf("rawvideo md5 = %s, want %s", got, high10DirectSubResidualRawVideoMD5)
	}
}

func assertHigh10DirectSubResidualCAVLCFixtureSyntax(t *testing.T, data []byte) {
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
				sps.Direct8x8InferenceFlag != 1 {
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
			if pps.CABAC != 0 || pps.Transform8x8Mode != 0 || pps.WeightedPred != 0 ||
				pps.WeightedBipredIDC != 0 || pps.RefCount[0] != 2 || pps.RefCount[1] != 1 {
				t.Fatalf("PPS cabac/8x8/weights/refs = %d/%d/%d/%d/%d/%d, want CAVLC no-8x8 unweighted refs=2/1",
					pps.CABAC, pps.Transform8x8Mode, pps.WeightedPred, pps.WeightedBipredIDC, pps.RefCount[0], pps.RefCount[1])
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
				assertHigh10DirectSubResidualCAVLCPayload(t, nal.Raw)
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

func assertHigh10DirectSubResidualCAVLCPayload(t *testing.T, raw []byte) {
	t.Helper()
	if len(raw) != 9 {
		t.Fatalf("B direct-sub residual NAL size = %d, want 9", len(raw))
	}
	want := "10000101111111011100010111111"
	for i, bit := range want {
		if got := h264FixtureBit(raw, 37+i); got != int(bit-'0') {
			t.Fatalf("B direct-sub residual payload bit[%d] = %d, want %c", i, got, bit)
		}
	}
}
