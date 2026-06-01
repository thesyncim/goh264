// SPDX-License-Identifier: LGPL-2.1-or-later

package goh264

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

const (
	high10BDeblockCAVLCPath        = "testdata/h264/high10_b_deblock_cavlc.h264"
	high10BDeblockCAVLCAnnexBSize  = 582
	high10BDeblockCAVLCAnnexBMD5   = "b8c45671afd9b919b7f391e09f9eced0"
	high10BDeblockFrameRawSize     = 768
	high10BDeblockCAVLCRawVideoMD5 = "35a2a24c460551f2c43e759dde953583"
)

var high10BDeblockCAVLCFrameMD5 = []string{
	"95893f95fdce0f45e7593f4eca8bd834",
	"6be70b93adcb7bb8f78d667776b774dc",
	"b7edf8a2678e03b0495ba6a6efebc063",
}

func TestHigh10BDeblockCAVLCFixtureSyntax(t *testing.T) {
	data := readHigh10BDeblockCAVLCFixture(t)
	assertHigh10BDeblockCAVLCFixtureSyntax(t, data)
}

func TestDecodeAnnexBHigh10BDeblockCAVLCFrames(t *testing.T) {
	data := readHigh10BDeblockCAVLCFixture(t)
	assertHigh10BDeblockCAVLCFixtureSyntax(t, data)

	frames, err := NewDecoder().DecodeAnnexBFrames(data)
	if err != nil {
		t.Fatalf("DecodeAnnexBFrames: %v", err)
	}
	assertHigh10BDeblockCAVLCFrames(t, frames)
}

func TestDecodeAVCHigh10BDeblockCAVLCFrames(t *testing.T) {
	data := readHigh10BDeblockCAVLCFixture(t)
	assertHigh10BDeblockCAVLCFixtureSyntax(t, data)

	for _, nalLengthSize := range []int{2, 3, 4} {
		frames, err := NewDecoder().DecodeAVCFrames(annexBToAVC(t, data, nalLengthSize), nalLengthSize)
		if err != nil {
			t.Fatalf("nalLengthSize=%d: DecodeAVCFrames: %v", nalLengthSize, err)
		}
		assertHigh10BDeblockCAVLCFrames(t, frames)
	}
}

func TestDecodeAVCWithConfigurationRecordHigh10BDeblockCAVLCFrames(t *testing.T) {
	data := readHigh10BDeblockCAVLCFixture(t)
	assertHigh10BDeblockCAVLCFixtureSyntax(t, data)

	for _, nalLengthSize := range []int{2, 3, 4} {
		config, packet := annexBToAVCConfigAndPacket(t, data, nalLengthSize)
		frames, err := NewDecoder().DecodeAVCFramesWithConfigurationRecord(config, packet)
		if err != nil {
			t.Fatalf("nalLengthSize=%d: DecodeAVCFramesWithConfigurationRecord: %v", nalLengthSize, err)
		}
		assertHigh10BDeblockCAVLCFrames(t, frames)
	}
}

func TestDecodeConfiguredAVCHigh10BDeblockCAVLCFramesAcrossSamplesFlush(t *testing.T) {
	data := readHigh10BDeblockCAVLCFixture(t)
	assertHigh10BDeblockCAVLCFixtureSyntax(t, data)

	for _, nalLengthSize := range []int{2, 3, 4} {
		config, samples := annexBToAVCConfigAndSamples(t, data, nalLengthSize)
		if len(samples) != len(high10BDeblockCAVLCFrameMD5) {
			t.Fatalf("nalLengthSize=%d: samples = %d, want %d", nalLengthSize, len(samples), len(high10BDeblockCAVLCFrameMD5))
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
		assertHigh10BDeblockCAVLCFrames(t, frames)
	}
}

func TestDecodeAutoConfiguredAVCHigh10BDeblockCAVLCFramesAcrossSamplesFlush(t *testing.T) {
	data := readHigh10BDeblockCAVLCFixture(t)
	config, samples := annexBToAVCConfigAndSamples(t, data, 4)
	if len(samples) != len(high10BDeblockCAVLCFrameMD5) {
		t.Fatalf("samples = %d, want %d", len(samples), len(high10BDeblockCAVLCFrameMD5))
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
			t.Fatalf("sample[%d]: DecodeFrames: %v", i, err)
		}
		frames = append(frames, out...)
	}
	out, err = dec.DecodeFrames(nil)
	if err != nil {
		t.Fatalf("flush: %v", err)
	}
	frames = append(frames, out...)
	assertHigh10BDeblockCAVLCFrames(t, frames)
}

func TestFFmpegRawVideoMD5OracleHigh10BDeblockCAVLC(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}

	data := readHigh10BDeblockCAVLCFixture(t)
	assertHigh10BDeblockCAVLCFixtureSyntax(t, data)
	path := writeTempH264(t, data)
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
	for i, want := range high10BDeblockCAVLCFrameMD5 {
		line := []byte(fmt.Sprintf("0, %10d, %10d,        1, %8d, %s", i, i, high10BDeblockFrameRawSize, want))
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
	if len(raw) != len(high10BDeblockCAVLCFrameMD5)*high10BDeblockFrameRawSize {
		t.Fatalf("rawvideo size = %d, want %d", len(raw), len(high10BDeblockCAVLCFrameMD5)*high10BDeblockFrameRawSize)
	}
	sum := md5.Sum(raw)
	if got := hex.EncodeToString(sum[:]); got != high10BDeblockCAVLCRawVideoMD5 {
		t.Fatalf("rawvideo md5 = %s, want %s", got, high10BDeblockCAVLCRawVideoMD5)
	}
}

func readHigh10BDeblockCAVLCFixture(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile(high10BDeblockCAVLCPath)
	if err != nil {
		t.Fatalf("read %s: %v", high10BDeblockCAVLCPath, err)
	}
	if len(data) != high10BDeblockCAVLCAnnexBSize {
		t.Fatalf("annex b size = %d, want %d", len(data), high10BDeblockCAVLCAnnexBSize)
	}
	sum := md5.Sum(data)
	if got := hex.EncodeToString(sum[:]); got != high10BDeblockCAVLCAnnexBMD5 {
		t.Fatalf("annex b md5 = %s, want %s", got, high10BDeblockCAVLCAnnexBMD5)
	}
	return data
}

func assertHigh10BDeblockCAVLCFrames(t *testing.T, frames []*Frame) {
	t.Helper()
	assertHigh10FrameMD5Strings(t, frames, high10BDeblockCAVLCFrameMD5)
	raw := make([]byte, 0, len(frames)*high10BDeblockFrameRawSize)
	for i, frame := range frames {
		rawSize, err := frame.RawYUVSize()
		if err != nil {
			t.Fatalf("frame[%d] RawYUVSize: %v", i, err)
		}
		if rawSize != high10BDeblockFrameRawSize {
			t.Fatalf("frame[%d] raw yuv size = %d, want %d", i, rawSize, high10BDeblockFrameRawSize)
		}
		raw, err = frame.AppendRawYUVBytesLE(raw)
		if err != nil {
			t.Fatalf("frame[%d] AppendRawYUVBytesLE: %v", i, err)
		}
	}
	sum := md5.Sum(raw)
	if got := hex.EncodeToString(sum[:]); got != high10BDeblockCAVLCRawVideoMD5 {
		t.Fatalf("rawvideo md5 = %s, want %s", got, high10BDeblockCAVLCRawVideoMD5)
	}
}

func assertHigh10BDeblockCAVLCFixtureSyntax(t *testing.T, data []byte) {
	t.Helper()
	nals, err := h264.SplitAnnexB(data)
	if err != nil {
		t.Fatal(err)
	}

	var spsList [32]*h264.SPS
	var ppsList [256]*h264.PPS
	var gotNALs []h264.NALUnitType
	var gotSlices []int32
	var gotDeblock []int32
	for _, nal := range nals {
		switch nal.Type {
		case h264.NALSPS:
			sps, err := h264.DecodeSPS(nal.RBSP)
			if err != nil {
				t.Fatal(err)
			}
			if sps.ProfileIDC != 110 || sps.Width != 16 || sps.Height != 16 ||
				sps.ChromaFormatIDC != 1 || sps.BitDepthLuma != 10 || sps.BitDepthChroma != 10 {
				t.Fatalf("SPS profile/format = %d %dx%d chroma %d depth %d/%d, want High10 16x16 yuv420p10le",
					sps.ProfileIDC, sps.Width, sps.Height, sps.ChromaFormatIDC, sps.BitDepthLuma, sps.BitDepthChroma)
			}
			if sps.FrameMBSOnlyFlag != 1 || sps.MBAFF != 0 {
				t.Fatalf("SPS frame-only flags = frame_mbs_only:%d mbaff:%d, want 1/0", sps.FrameMBSOnlyFlag, sps.MBAFF)
			}
			spsList[sps.SPSID] = sps
		case h264.NALPPS:
			pps, err := h264.DecodePPS(nal.RBSP, &spsList)
			if err != nil {
				t.Fatal(err)
			}
			if pps.CABAC != 0 || pps.Transform8x8Mode != 0 {
				t.Fatalf("PPS cabac/8x8dct = %d/%d, want 0/0", pps.CABAC, pps.Transform8x8Mode)
			}
			if pps.RefCount != [2]uint32{2, 1} || pps.WeightedPred != 0 || pps.WeightedBipredIDC != 0 {
				t.Fatalf("PPS refs/weight = %v/%d/%d, want ref=2/1 and unweighted", pps.RefCount, pps.WeightedPred, pps.WeightedBipredIDC)
			}
			if pps.DeblockingFilterParametersPresent != 1 {
				t.Fatalf("PPS deblock params present = %d, want 1", pps.DeblockingFilterParametersPresent)
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
			if sh.SliceAlphaC0Offset != 0 || sh.SliceBetaOffset != 0 {
				t.Fatalf("slice deblock offsets = %d/%d, want 0/0", sh.SliceAlphaC0Offset, sh.SliceBetaOffset)
			}
			switch sh.SliceTypeNoS {
			case h264.PictureTypeI:
				if sh.DeblockingFilter != 0 {
					t.Fatalf("IDR deblock = %d, want disabled", sh.DeblockingFilter)
				}
				if sh.ListCount != 0 {
					t.Fatalf("I slice lists = %d, want none", sh.ListCount)
				}
			case h264.PictureTypeP:
				if sh.ListCount != 1 || sh.RefCount[0] != 1 || sh.DeblockingFilter != 1 ||
					sh.PredWeightTable.UseWeight != 0 || sh.PredWeightTable.UseWeightChroma != 0 {
					t.Fatalf("P slice lists/ref/deblock/weights = %d/%d/%d/%d/%d, want one L0 ref, deblock enabled, unweighted",
						sh.ListCount, sh.RefCount[0], sh.DeblockingFilter, sh.PredWeightTable.UseWeight, sh.PredWeightTable.UseWeightChroma)
				}
			case h264.PictureTypeB:
				if sh.ListCount != 2 || sh.RefCount != [2]uint32{1, 1} || sh.DeblockingFilter != 1 ||
					sh.DirectSpatialMVPred != 0 || sh.PredWeightTable.UseWeight != 0 || sh.PredWeightTable.UseWeightChroma != 0 {
					t.Fatalf("B slice lists/refs/deblock/direct/weights = %d/%v/%d/%d/%d/%d, want L0/L1 refs=1/1, temporal flag, deblock enabled, unweighted",
						sh.ListCount, sh.RefCount, sh.DeblockingFilter, sh.DirectSpatialMVPred, sh.PredWeightTable.UseWeight, sh.PredWeightTable.UseWeightChroma)
				}
			default:
				t.Fatalf("unexpected slice type %d", sh.SliceTypeNoS)
			}
			gotNALs = append(gotNALs, nal.Type)
			gotSlices = append(gotSlices, sh.SliceTypeNoS)
			gotDeblock = append(gotDeblock, sh.DeblockingFilter)
		}
	}

	wantNALs := []h264.NALUnitType{h264.NALIDRSlice, h264.NALSlice, h264.NALSlice}
	wantSlices := []int32{h264.PictureTypeI, h264.PictureTypeP, h264.PictureTypeB}
	wantDeblock := []int32{0, 1, 1}
	if fmt.Sprint(gotNALs) != fmt.Sprint(wantNALs) ||
		fmt.Sprint(gotSlices) != fmt.Sprint(wantSlices) ||
		fmt.Sprint(gotDeblock) != fmt.Sprint(wantDeblock) {
		t.Fatalf("VCL = nals %v slices %v deblock %v, want %v/%v/%v",
			gotNALs, gotSlices, gotDeblock, wantNALs, wantSlices, wantDeblock)
	}
}
