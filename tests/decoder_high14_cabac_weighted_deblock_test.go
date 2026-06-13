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

const high14CABACWeightedP16x16Mode1DeblockAnnexBHex = `
0000000167f4000aa39cb45d808800000300080000030010789135000000000168ef00d32c80000000016588843ffef7d4b7ccb2eea3c2b55181f9b5586100000001419a23531644ccffaa092ccffad67ffc
`

const high14CABACWeightedP16x16Mode2DeblockAnnexBHex = `
0000000167f4000aa39cb45d808800000300080000030010789135000000000168ef00d32c80000000016588843bfffef7d4b7ccb2eea3c2b55181f9b5586100000001419a23531644ccdfaa092ccffad67ffc
`

const (
	high14CABACWeightedP16x16Mode1DeblockBitstreamMD5 = "b2dff4798f3c5a5c887792c24544fc67"
	high14CABACWeightedP16x16Mode2DeblockBitstreamMD5 = "85f8370cf86e2e0aaea2cbee465c9a84"
	high14CABACWeightedP16x16IDRFrameMD5              = "e4f5c245016013e8a268ca0f41697b10"
	high14CABACWeightedP16x16PFrameMD5                = "00826d07151d2bd52ed10c8a0083959d"
	high14CABACWeightedP16x16DeblockRawVideoMD5       = "0e64af23705ec027979352c8ae263038"
	high14CABACWeightedP16x16DeblockRawFrameSize      = 1536
)

type high14CABACWeightedDeblockCase struct {
	name         string
	hex          string
	deblockMode  int32
	bitstreamMD5 string
}

func TestHigh14CABACWeightedDeblockFixtureSyntax(t *testing.T) {
	for _, tt := range high14CABACWeightedDeblockCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := decodeHexFixture(t, tt.hex)
			assertHigh14CABACWeightedDeblockFixtureSyntax(t, data, tt.deblockMode, tt.bitstreamMD5)
		})
	}
}

func TestDecodeAnnexBHigh14CABACWeightedDeblockFrames(t *testing.T) {
	for _, tt := range high14CABACWeightedDeblockCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := decodeHexFixture(t, tt.hex)
			assertHigh14CABACWeightedDeblockFixtureSyntax(t, data, tt.deblockMode, tt.bitstreamMD5)

			frames, err := NewDecoder().DecodeAnnexBFrames(data)
			if err != nil {
				t.Fatalf("decode High14 CABAC weighted mode-%d deblock Annex B: %v", tt.deblockMode, err)
			}
			assertHigh14CABACWeightedDeblockFrames(t, frames)
		})
	}
}

func TestDecodeAVCHigh14CABACWeightedDeblockFrames(t *testing.T) {
	for _, tt := range high14CABACWeightedDeblockCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := decodeHexFixture(t, tt.hex)
			for _, nalLengthSize := range []int{2, 3, 4} {
				frames, err := NewDecoder().DecodeAVCFrames(annexBToAVC(t, data, nalLengthSize), nalLengthSize)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: %v", nalLengthSize, err)
				}
				assertHigh14CABACWeightedDeblockFrames(t, frames)
			}
		})
	}
}

func TestDecodeConfiguredAVCHigh14CABACWeightedDeblockFrames(t *testing.T) {
	for _, tt := range high14CABACWeightedDeblockCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := decodeHexFixture(t, tt.hex)
			config, samples := annexBToAVCConfigAndSamples(t, data, 4)
			if len(samples) != 2 {
				t.Fatalf("samples = %d, want 2", len(samples))
			}

			dec := NewDecoder()
			if _, err := dec.ConfigureAVCC(config); err != nil {
				t.Fatal(err)
			}
			var frames []*Frame
			for i, sample := range samples {
				frame, err := dec.DecodeConfiguredAVC(sample)
				if err != nil {
					t.Fatalf("sample[%d] decode High14 CABAC weighted mode-%d deblock: %v", i, tt.deblockMode, err)
				}
				frames = append(frames, frame)
			}
			assertHigh14CABACWeightedDeblockFrames(t, frames)
		})
	}
}

func TestFFmpegRawVideoFrameMD5OracleHigh14CABACWeightedDeblock(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}

	for _, tt := range high14CABACWeightedDeblockCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := decodeHexFixture(t, tt.hex)
			assertHigh14CABACWeightedDeblockFixtureSyntax(t, data, tt.deblockMode, tt.bitstreamMD5)
			assertFFmpegHigh14CABACWeightedDeblockRawVideoOracle(t, data)
		})
	}
}

func high14CABACWeightedDeblockCases() []high14CABACWeightedDeblockCase {
	return []high14CABACWeightedDeblockCase{
		{
			name:         "mode1-weighted-p16x16-no-residual",
			hex:          high14CABACWeightedP16x16Mode1DeblockAnnexBHex,
			deblockMode:  1,
			bitstreamMD5: high14CABACWeightedP16x16Mode1DeblockBitstreamMD5,
		},
		{
			name:         "mode2-weighted-p16x16-no-residual",
			hex:          high14CABACWeightedP16x16Mode2DeblockAnnexBHex,
			deblockMode:  2,
			bitstreamMD5: high14CABACWeightedP16x16Mode2DeblockBitstreamMD5,
		},
	}
}

func assertHigh14CABACWeightedDeblockFixtureSyntax(t *testing.T, data []byte, wantDeblockMode int32, wantBitstreamMD5 string) {
	t.Helper()
	sum := md5.Sum(data)
	if got := hex.EncodeToString(sum[:]); got != wantBitstreamMD5 {
		t.Fatalf("High14 CABAC weighted deblock bitstream md5 = %s, want %s", got, wantBitstreamMD5)
	}

	nals, err := h264.SplitAnnexB(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(nals) != 4 {
		t.Fatalf("NAL count = %d, want SPS/PPS/IDR/P", len(nals))
	}

	var spsList [32]*h264.SPS
	var ppsList [256]*h264.PPS
	gotSlices := make([]int32, 0, 2)
	gotQScale := make([]uint32, 0, 2)
	for i, nal := range nals {
		switch nal.Type {
		case h264.NALSPS:
			sps, err := h264.DecodeSPS(nal.RBSP)
			if err != nil {
				t.Fatal(err)
			}
			if i != 0 || sps.ProfileIDC != 244 || sps.Width != 32 || sps.Height != 16 ||
				sps.ChromaFormatIDC != 1 || sps.BitDepthLuma != 14 || sps.BitDepthChroma != 14 {
				t.Fatalf("SPS = nal[%d] profile %d %dx%d chroma %d depth %d/%d, want High14 32x16 yuv420p14le",
					i, sps.ProfileIDC, sps.Width, sps.Height, sps.ChromaFormatIDC, sps.BitDepthLuma, sps.BitDepthChroma)
			}
			spsList[sps.SPSID] = sps
		case h264.NALPPS:
			pps, err := h264.DecodePPS(nal.RBSP, &spsList)
			if err != nil {
				t.Fatal(err)
			}
			if i != 1 || pps.CABAC != 1 || pps.Transform8x8Mode != 0 ||
				pps.DeblockingFilterParametersPresent == 0 || pps.WeightedPred != 1 ||
				pps.WeightedBipredIDC != 0 || pps.RefCount != [2]uint32{1, 1} || pps.InitQP != 10 {
				t.Fatalf("PPS = nal[%d] cabac/8x8/deblock/weight/refs/initQP = %d/%d/%d/%d,%d/%v/%d, want CABAC/no-8x8/deblock/weighted/ref=1/initQP=10",
					i, pps.CABAC, pps.Transform8x8Mode, pps.DeblockingFilterParametersPresent,
					pps.WeightedPred, pps.WeightedBipredIDC, pps.RefCount, pps.InitQP)
			}
			ppsList[pps.PPSID] = pps
		case h264.NALIDRSlice, h264.NALSlice:
			sh, err := h264.ParseSliceHeader(nal, &ppsList)
			if err != nil {
				t.Fatal(err)
			}
			if sh.PictureStructure != h264.PictureFrame || sh.DeblockingFilter != wantDeblockMode ||
				sh.SliceAlphaC0Offset != 0 || sh.SliceBetaOffset != 0 {
				t.Fatalf("slice picture/deblock/offsets = %d/%d/%d/%d, want frame/mode-%d/0/0",
					sh.PictureStructure, sh.DeblockingFilter, sh.SliceAlphaC0Offset, sh.SliceBetaOffset, wantDeblockMode)
			}
			if sh.SliceTypeNoS == h264.PictureTypeP {
				if sh.ListCount != 1 || sh.RefCount[0] != 1 {
					t.Fatalf("P slice lists/ref0 = %d/%d, want one weighted L0 ref", sh.ListCount, sh.RefCount[0])
				}
				assertHigh14WeightedPPredWeight(t, sh.PredWeightTable)
			}
			gotSlices = append(gotSlices, sh.SliceTypeNoS)
			gotQScale = append(gotQScale, sh.QScale)
		default:
			t.Fatalf("unexpected NAL type %d", nal.Type)
		}
	}
	if len(gotSlices) != 2 || gotSlices[0] != h264.PictureTypeI || gotSlices[1] != h264.PictureTypeP {
		t.Fatalf("slice types = %v, want I/P", gotSlices)
	}
	if gotQScale[0] != 7 || gotQScale[1] != 10 {
		t.Fatalf("slice QScale = %v, want 7/10", gotQScale)
	}
}

func assertHigh14CABACWeightedDeblockFrames(t *testing.T, frames []*Frame) {
	t.Helper()
	wantFrameMD5 := []string{
		high14CABACWeightedP16x16IDRFrameMD5,
		high14CABACWeightedP16x16PFrameMD5,
	}
	if len(frames) != len(wantFrameMD5) {
		t.Fatalf("frames = %d, want %d", len(frames), len(wantFrameMD5))
	}
	var rawVideo []byte
	for i, frame := range frames {
		if frame.Width != 32 || frame.Height != 16 ||
			frame.ChromaFormatIDC != 1 || frame.BitDepthLuma != 14 || frame.BitDepthChroma != 14 {
			t.Fatalf("frame[%d] format = %dx%d chroma %d depth %d/%d, want 32x16 yuv420p14le",
				i, frame.Width, frame.Height, frame.ChromaFormatIDC, frame.BitDepthLuma, frame.BitDepthChroma)
		}
		if pixFmt, err := frame.RawPixelFormat(); err != nil || pixFmt != "yuv420p14le" {
			t.Fatalf("frame[%d] RawPixelFormat = %q/%v, want yuv420p14le/nil", i, pixFmt, err)
		}
		if size, err := frame.RawYUVSize(); err != nil || size != high14CABACWeightedP16x16DeblockRawFrameSize {
			t.Fatalf("frame[%d] RawYUVSize = %d/%v, want %d/nil", i, size, err, high14CABACWeightedP16x16DeblockRawFrameSize)
		}
		raw, err := frame.AppendRawYUVBytesLE(nil)
		if err != nil {
			t.Fatalf("frame[%d] AppendRawYUVBytesLE: %v", i, err)
		}
		rawVideo = append(rawVideo, raw...)
		sum := md5.Sum(raw)
		if got := hex.EncodeToString(sum[:]); got != wantFrameMD5[i] {
			t.Fatalf("frame[%d] raw md5 = %s, want %s", i, got, wantFrameMD5[i])
		}
		if _, err := frame.AppendRawYUV(nil); err != ErrUnsupported {
			t.Fatalf("frame[%d] AppendRawYUV high14 error = %v, want ErrUnsupported", i, err)
		}
	}
	sum := md5.Sum(rawVideo)
	if got := hex.EncodeToString(sum[:]); got != high14CABACWeightedP16x16DeblockRawVideoMD5 {
		t.Fatalf("rawvideo md5 = %s, want %s", got, high14CABACWeightedP16x16DeblockRawVideoMD5)
	}
}

func assertFFmpegHigh14CABACWeightedDeblockRawVideoOracle(t *testing.T, data []byte) {
	t.Helper()
	path := writeTempH264(t, data)
	wantFrameMD5 := []string{
		high14CABACWeightedP16x16IDRFrameMD5,
		high14CABACWeightedP16x16PFrameMD5,
	}
	framemd5 := exec.Command("ffmpeg",
		"-v", "error",
		"-xerror",
		"-f", "h264",
		"-i", path,
		"-an", "-sn", "-dn",
		"-pix_fmt", "yuv420p14le",
		"-f", "framemd5",
		"-",
	)
	framemd5Out, err := framemd5.Output()
	if err != nil {
		t.Fatalf("ffmpeg framemd5: %v", err)
	}
	for i, want := range wantFrameMD5 {
		line := []byte(fmt.Sprintf("0, %10d, %10d,        1, %8d, %s", i, i, high14CABACWeightedP16x16DeblockRawFrameSize, want))
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
		"-pix_fmt", "yuv420p14le",
		"-f", "rawvideo",
		"-",
	)
	raw, err := rawvideo.Output()
	if err != nil {
		t.Fatalf("ffmpeg rawvideo: %v", err)
	}
	if len(raw) != len(wantFrameMD5)*high14CABACWeightedP16x16DeblockRawFrameSize {
		t.Fatalf("rawvideo size = %d, want %d", len(raw), len(wantFrameMD5)*high14CABACWeightedP16x16DeblockRawFrameSize)
	}
	sum := md5.Sum(raw)
	if got := hex.EncodeToString(sum[:]); got != high14CABACWeightedP16x16DeblockRawVideoMD5 {
		t.Fatalf("rawvideo md5 = %s, want %s", got, high14CABACWeightedP16x16DeblockRawVideoMD5)
	}
}
