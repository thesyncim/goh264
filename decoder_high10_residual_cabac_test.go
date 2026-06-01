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

const high10CABACP16x16ResidualAnnexBHex = `
00000001676e000aa6cb4f6022000003000200000300041e244d400000000168ee014cb20000016588843ae17cdf5e55ff3572742af8df40e838c1f40c25e5ea077dff468bd638c35402fcaa23fe589c
fa232ac88cee3f9447be6f75a29e43869ca834cd7f7660e72a921ae514a09d8f4da01b2bc07c176f22744100000001419a235f9b0bf7fb07fe6c8f8e4f31d997a3d0b34d675860ed941ff25932a1285c
3466
`

var high10CABACP16x16ResidualFrameMD5 = []string{
	"b47c39a842e4395e1ed527f2339c10ee",
	"94edd171434db39321da0bc98328f421",
}

const (
	high10CABACP16x16ResidualRawFrameSize = 768
	high10CABACP16x16ResidualRawVideoMD5  = "f2c1ffc6f537acf9afcb10beecbedb1e"
)

func TestHigh10CABACP16x16ResidualFixtureSyntax(t *testing.T) {
	assertHigh10CABACP16x16ResidualFixtureSyntax(t, decodeHexFixture(t, high10CABACP16x16ResidualAnnexBHex))
}

func TestDecodeAnnexBHigh10CABACP16x16ResidualFrames(t *testing.T) {
	data := decodeHexFixture(t, high10CABACP16x16ResidualAnnexBHex)
	assertHigh10CABACP16x16ResidualFixtureSyntax(t, data)

	frames, err := NewDecoder().DecodeAnnexBFrames(data)
	if err != nil {
		t.Fatalf("decode High10 CABAC P16x16 residual fixture: %v", err)
	}
	assertHigh10CABACP16x16ResidualFrames(t, frames)
}

func TestDecodeConfiguredAVCHigh10CABACP16x16ResidualFrames(t *testing.T) {
	data := decodeHexFixture(t, high10CABACP16x16ResidualAnnexBHex)
	assertHigh10CABACP16x16ResidualFixtureSyntax(t, data)

	config, samples := annexBToAVCConfigAndSamples(t, data, 4)
	if len(samples) != len(high10CABACP16x16ResidualFrameMD5) {
		t.Fatalf("samples = %d, want %d", len(samples), len(high10CABACP16x16ResidualFrameMD5))
	}

	dec := NewDecoder()
	if _, err := dec.ParseAVCDecoderConfigurationRecord(config); err != nil {
		t.Fatal(err)
	}
	var frames []*Frame
	for i, sample := range samples {
		frame, err := dec.DecodeConfiguredAVC(sample)
		if err != nil {
			t.Fatalf("sample[%d] decode High10 CABAC P16x16 residual fixture: %v", i, err)
		}
		frames = append(frames, frame)
	}
	assertHigh10CABACP16x16ResidualFrames(t, frames)
}

func TestFFmpegRawVideoFrameMD5OracleHigh10CABACP16x16Residual(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}

	data := decodeHexFixture(t, high10CABACP16x16ResidualAnnexBHex)
	assertHigh10CABACP16x16ResidualFixtureSyntax(t, data)
	assertFFmpegHigh10CABACP16x16ResidualRawVideoOracle(t, data)
}

func assertHigh10CABACP16x16ResidualFrames(t *testing.T, frames []*Frame) {
	t.Helper()
	assertHigh10FrameMD5Strings(t, frames, high10CABACP16x16ResidualFrameMD5)
	for i, frame := range frames {
		if frame.Width != 16 || frame.Height != 16 {
			t.Fatalf("frame[%d] size = %dx%d, want 16x16", i, frame.Width, frame.Height)
		}
		rawSize, err := frame.RawYUVSize()
		if err != nil {
			t.Fatalf("frame[%d] RawYUVSize: %v", i, err)
		}
		if rawSize != high10CABACP16x16ResidualRawFrameSize {
			t.Fatalf("frame[%d] raw yuv size = %d, want %d", i, rawSize, high10CABACP16x16ResidualRawFrameSize)
		}
		raw, err := frame.AppendRawYUVBytesLE(nil)
		if err != nil {
			t.Fatalf("frame[%d] AppendRawYUVBytesLE: %v", i, err)
		}
		if len(raw) != high10CABACP16x16ResidualRawFrameSize {
			t.Fatalf("frame[%d] raw yuv len = %d, want %d", i, len(raw), high10CABACP16x16ResidualRawFrameSize)
		}
	}
}

func assertFFmpegHigh10CABACP16x16ResidualRawVideoOracle(t *testing.T, data []byte) {
	t.Helper()
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
	for i, want := range high10CABACP16x16ResidualFrameMD5 {
		line := []byte(fmt.Sprintf("0, %10d, %10d,        1, %8d, %s", i, i, high10CABACP16x16ResidualRawFrameSize, want))
		if !bytes.Contains(framemd5Out, line) {
			t.Fatalf("frame[%d] missing %q in framemd5:\n%s", i, line, framemd5Out)
		}
	}

	rawvideo := exec.Command("ffmpeg",
		"-v", "error",
		"-f", "h264",
		"-i", path,
		"-an", "-sn", "-dn",
		"-pix_fmt", "yuv420p10le",
		"-f", "rawvideo",
		"-",
	)
	raw, err := rawvideo.Output()
	if err != nil {
		t.Fatalf("ffmpeg rawvideo: %v", err)
	}
	wantSize := len(high10CABACP16x16ResidualFrameMD5) * high10CABACP16x16ResidualRawFrameSize
	if len(raw) != wantSize {
		t.Fatalf("rawvideo size = %d, want %d", len(raw), wantSize)
	}
	sum := md5.Sum(raw)
	if got := hex.EncodeToString(sum[:]); got != high10CABACP16x16ResidualRawVideoMD5 {
		t.Fatalf("rawvideo md5 = %s, want %s", got, high10CABACP16x16ResidualRawVideoMD5)
	}
	for i, want := range high10CABACP16x16ResidualFrameMD5 {
		sum := md5.Sum(raw[i*high10CABACP16x16ResidualRawFrameSize : (i+1)*high10CABACP16x16ResidualRawFrameSize])
		if got := hex.EncodeToString(sum[:]); got != want {
			t.Fatalf("rawvideo frame[%d] md5 = %s, want %s", i, got, want)
		}
	}
}

func assertHigh10CABACP16x16ResidualFixtureSyntax(t *testing.T, data []byte) {
	t.Helper()
	nals, err := h264.SplitAnnexB(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(nals) != 4 {
		t.Fatalf("NAL count = %d, want 4 stripped SPS/PPS/IDR/P", len(nals))
	}
	wantNALs := []h264.NALUnitType{h264.NALSPS, h264.NALPPS, h264.NALIDRSlice, h264.NALSlice}
	for i, want := range wantNALs {
		if nals[i].Type != want {
			t.Fatalf("NAL[%d] type = %d, want %d", i, nals[i].Type, want)
		}
	}

	var spsList [32]*h264.SPS
	var ppsList [256]*h264.PPS
	var gotVCL []h264.NALUnitType
	var gotSlices []int32
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
			if pps.CABAC != 1 || pps.Transform8x8Mode != 0 {
				t.Fatalf("PPS cabac/8x8dct = %d/%d, want 1/0", pps.CABAC, pps.Transform8x8Mode)
			}
			if pps.RefCount != [2]uint32{1, 1} || pps.WeightedPred != 0 || pps.WeightedBipredIDC != 0 {
				t.Fatalf("PPS refs/weight = %v/%d/%d, want ref=1 and unweighted", pps.RefCount, pps.WeightedPred, pps.WeightedBipredIDC)
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
			if sh.PictureStructure != h264.PictureFrame || sh.DeblockingFilter != 0 {
				t.Fatalf("slice picture/deblock = %d/%d, want frame/disabled", sh.PictureStructure, sh.DeblockingFilter)
			}
			if sh.SliceTypeNoS == h264.PictureTypeP {
				if sh.ListCount != 1 || sh.RefCount[0] != 1 || sh.PredWeightTable.UseWeight != 0 || sh.PredWeightTable.UseWeightChroma != 0 {
					t.Fatalf("P slice lists/ref0/weights = %d/%d/%d/%d, want one L0 ref and unweighted",
						sh.ListCount, sh.RefCount[0], sh.PredWeightTable.UseWeight, sh.PredWeightTable.UseWeightChroma)
				}
			}
			gotVCL = append(gotVCL, nal.Type)
			gotSlices = append(gotSlices, sh.SliceTypeNoS)
		}
	}
	if len(gotVCL) != 2 || gotVCL[0] != h264.NALIDRSlice || gotVCL[1] != h264.NALSlice {
		t.Fatalf("VCL NALs = %v, want IDR then non-IDR", gotVCL)
	}
	if gotSlices[0] != h264.PictureTypeI || gotSlices[1] != h264.PictureTypeP {
		t.Fatalf("slice types = %v, want I then P", gotSlices)
	}
}
