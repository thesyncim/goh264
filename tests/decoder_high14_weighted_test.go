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

const (
	high14WeightedPNoDeblockBitstreamMD5    = "0e2a86748a45ba31602d74c0efeae13e"
	high14WeightedPMode1DeblockBitstreamMD5 = "86473e2b5a54a60e9885cdeeb0e6f863"
	high14WeightedPMode2DeblockBitstreamMD5 = "f94aa0f87c1d1f62a6f5b6171f50ee2b"
	high14WeightedPIDRFrameMD5              = "6d3514a30f506561e144447d287270ab"
	high14WeightedPPFrameMD5                = "7da479daf8ee941db37effb08c2fcf06"
	high14WeightedPRawVideoMD5              = "d0bef80b89aaa5b5f71ffa8814fcd50f"
	high14WeightedPFrameRawSize             = 768
)

type high14WeightedPCase struct {
	name         string
	data         []byte
	deblockMode  int32
	bitstreamMD5 string
}

func TestHigh14WeightedPFixtureSyntax(t *testing.T) {
	for _, tt := range high14WeightedPCases() {
		t.Run(tt.name, func(t *testing.T) {
			assertHigh14WeightedPFixtureSyntax(t, tt.data, tt.deblockMode, tt.bitstreamMD5)
		})
	}
}

func TestDecodeAnnexBHigh14WeightedPFrames(t *testing.T) {
	for _, tt := range high14WeightedPCases() {
		t.Run(tt.name, func(t *testing.T) {
			assertHigh14WeightedPFixtureSyntax(t, tt.data, tt.deblockMode, tt.bitstreamMD5)

			frames, err := NewDecoder().DecodeAnnexBFrames(tt.data)
			if err != nil {
				t.Fatalf("decode High14 weighted P Annex B: %v", err)
			}
			assertHigh14WeightedPFrames(t, frames)
		})
	}
}

func TestDecodeAVCHigh14WeightedPFrames(t *testing.T) {
	for _, tt := range high14WeightedPCases() {
		t.Run(tt.name, func(t *testing.T) {
			for _, nalLengthSize := range []int{2, 3, 4} {
				frames, err := NewDecoder().DecodeAVCFrames(annexBToAVC(t, tt.data, nalLengthSize), nalLengthSize)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: %v", nalLengthSize, err)
				}
				assertHigh14WeightedPFrames(t, frames)
			}
		})
	}
}

func TestDecodeConfiguredAVCHigh14WeightedPFrames(t *testing.T) {
	for _, tt := range high14WeightedPCases() {
		t.Run(tt.name, func(t *testing.T) {
			config, samples := annexBToAVCConfigAndSamples(t, tt.data, 4)
			if len(samples) != 2 {
				t.Fatalf("samples = %d, want 2", len(samples))
			}

			dec := NewDecoder()
			if _, err := dec.ConfigureAVCDecoderConfigurationRecord(config); err != nil {
				t.Fatal(err)
			}
			var frames []*Frame
			for i, sample := range samples {
				frame, err := dec.DecodeConfiguredAVC(sample)
				if err != nil {
					t.Fatalf("sample[%d] decode High14 weighted P: %v", i, err)
				}
				frames = append(frames, frame)
			}
			assertHigh14WeightedPFrames(t, frames)
		})
	}
}

func TestFFmpegRawVideoFrameMD5OracleHigh14WeightedP(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}

	for _, tt := range high14WeightedPCases() {
		t.Run(tt.name, func(t *testing.T) {
			assertFFmpegHigh14WeightedPRawVideoOracle(t, tt.data)
		})
	}
}

func high14WeightedPCases() []high14WeightedPCase {
	return []high14WeightedPCase{
		{
			name:         "no-deblock",
			data:         high14WeightedPFixture(1),
			deblockMode:  0,
			bitstreamMD5: high14WeightedPNoDeblockBitstreamMD5,
		},
		{
			name:         "mode1-deblock",
			data:         high14WeightedPFixture(0),
			deblockMode:  1,
			bitstreamMD5: high14WeightedPMode1DeblockBitstreamMD5,
		},
		{
			name:         "mode2-slice-boundary-deblock",
			data:         high14WeightedPFixture(2),
			deblockMode:  2,
			bitstreamMD5: high14WeightedPMode2DeblockBitstreamMD5,
		},
	}
}

func high14WeightedPFixture(disableDeblockingFilterIDC uint32) []byte {
	var data []byte
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALSPS), highInterSPSRBSP(14)))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALPPS), highWeightedPPPSRBSP(14)))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALIDRSlice), highIntra16x16ResidualDeblockSliceRBSP(highIntra16x16NoResidualPayloadBits, disableDeblockingFilterIDC)))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALSlice), highWeightedP16x16NoResidualDeblockSliceRBSP(disableDeblockingFilterIDC)))
	return data
}

func highWeightedPPPSRBSP(bitDepth int) []byte {
	var b decoderSEIBitBuilder
	b.writeUE(0)
	b.writeUE(0)
	b.writeBit(0)
	b.writeBit(0)
	b.writeUE(0)
	b.writeUE(0)
	b.writeUE(0)
	b.writeBit(1)
	b.writeBits(0, 2)
	b.writeSE(int32(-6 * (bitDepth - 8)))
	b.writeSE(0)
	b.writeSE(0)
	b.writeBit(1)
	b.writeBit(0)
	b.writeBit(0)
	b.writeBit(0)
	b.writeBit(0)
	b.writeSE(0)
	return b.rbsp()
}

func highWeightedP16x16NoResidualDeblockSliceRBSP(disableDeblockingFilterIDC uint32) []byte {
	var b decoderSEIBitBuilder
	b.writeUE(0)
	b.writeUE(0)
	b.writeUE(0)
	b.writeBits(1, 4)
	b.writeBit(0)
	b.writeBit(0)
	writeHighWeightedPPredWeightSyntax(&b)
	b.writeBit(0)
	b.writeSE(0)
	writeHighCAVLCDeblockSyntax(&b, disableDeblockingFilterIDC)
	highIntra16x16WritePayloadBits(&b, highInterP16x16NoResidualPayloadBits)
	return b.rbsp()
}

func writeHighWeightedPPredWeightSyntax(b *decoderSEIBitBuilder) {
	b.writeUE(2)
	b.writeUE(1)
	b.writeBit(1)
	b.writeSE(3)
	b.writeSE(-2)
	b.writeBit(1)
	b.writeSE(2)
	b.writeSE(1)
	b.writeSE(-1)
	b.writeSE(3)
}

func assertHigh14WeightedPFixtureSyntax(t *testing.T, data []byte, wantDeblockMode int32, wantBitstreamMD5 string) {
	t.Helper()
	sum := md5.Sum(data)
	if got := hex.EncodeToString(sum[:]); got != wantBitstreamMD5 {
		t.Fatalf("bitstream md5 = %s, want %s", got, wantBitstreamMD5)
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
	var gotSlices []int32
	for _, nal := range nals {
		switch nal.Type {
		case h264.NALSPS:
			sps, err := h264.DecodeSPS(nal.RBSP)
			if err != nil {
				t.Fatal(err)
			}
			if sps.ProfileIDC != 244 || sps.Width != 16 || sps.Height != 16 ||
				sps.ChromaFormatIDC != 1 || sps.BitDepthLuma != 14 || sps.BitDepthChroma != 14 || sps.RefFrameCount != 1 {
				t.Fatalf("SPS profile/format = %d %dx%d chroma %d depth %d/%d refs %d, want High14 4:2:0 ref=1",
					sps.ProfileIDC, sps.Width, sps.Height, sps.ChromaFormatIDC, sps.BitDepthLuma, sps.BitDepthChroma, sps.RefFrameCount)
			}
			spsList[sps.SPSID] = sps
		case h264.NALPPS:
			pps, err := h264.DecodePPS(nal.RBSP, &spsList)
			if err != nil {
				t.Fatal(err)
			}
			if pps.CABAC != 0 || pps.Transform8x8Mode != 0 || pps.WeightedPred != 1 ||
				pps.WeightedBipredIDC != 0 || pps.DeblockingFilterParametersPresent == 0 ||
				pps.RefCount[0] != 1 || pps.RefCount[1] != 1 {
				t.Fatalf("PPS CABAC/8x8/weighted/bipred/deblock/refs = %d/%d/%d/%d/%d/%d/%d, want CAVLC/no-8x8/weighted/no-bipred/deblock params/1/1",
					pps.CABAC, pps.Transform8x8Mode, pps.WeightedPred, pps.WeightedBipredIDC,
					pps.DeblockingFilterParametersPresent, pps.RefCount[0], pps.RefCount[1])
			}
			ppsList[pps.PPSID] = pps
		case h264.NALIDRSlice, h264.NALSlice:
			sh, err := h264.ParseSliceHeader(nal, &ppsList)
			if err != nil {
				t.Fatal(err)
			}
			if sh.PictureStructure != h264.PictureFrame || sh.DeblockingFilter != wantDeblockMode ||
				sh.SliceAlphaC0Offset != 0 || sh.SliceBetaOffset != 0 || sh.QScale != 26 {
				t.Fatalf("slice picture/deblock/offsets/qp = %d/%d/%d/%d/%d, want frame/mode-%d/0/0/26",
					sh.PictureStructure, sh.DeblockingFilter, sh.SliceAlphaC0Offset, sh.SliceBetaOffset, sh.QScale, wantDeblockMode)
			}
			if sh.SliceTypeNoS == h264.PictureTypeP {
				if sh.ListCount != 1 || sh.RefCount[0] != 1 {
					t.Fatalf("P slice lists/ref0 = %d/%d, want one L0 ref", sh.ListCount, sh.RefCount[0])
				}
				assertHigh14WeightedPPredWeight(t, sh.PredWeightTable)
			}
			gotSlices = append(gotSlices, sh.SliceTypeNoS)
		default:
			t.Fatalf("unexpected NAL type %d in High14 weighted P fixture", nal.Type)
		}
	}
	wantSlices := []int32{h264.PictureTypeI, h264.PictureTypeP}
	if len(gotSlices) != len(wantSlices) {
		t.Fatalf("slice count = %d, want %d", len(gotSlices), len(wantSlices))
	}
	for i, want := range wantSlices {
		if gotSlices[i] != want {
			t.Fatalf("slice[%d] type = %d, want %d", i, gotSlices[i], want)
		}
	}
}

func assertHigh14WeightedPPredWeight(t *testing.T, pwt h264.PredWeightTable) {
	t.Helper()
	if pwt.UseWeight != 1 || pwt.UseWeightChroma != 1 ||
		pwt.LumaLog2WeightDenom != 2 || pwt.ChromaLog2WeightDenom != 1 {
		t.Fatalf("weight flags/denom = use %d/%d denom %d/%d, want 1/1 2/1",
			pwt.UseWeight, pwt.UseWeightChroma, pwt.LumaLog2WeightDenom, pwt.ChromaLog2WeightDenom)
	}
	if pwt.LumaWeight[0][0] != [2]int32{3, -2} ||
		pwt.ChromaWeight[0][0][0] != [2]int32{2, 1} ||
		pwt.ChromaWeight[0][0][1] != [2]int32{-1, 3} {
		t.Fatalf("weights = luma %+v chroma %+v, want luma [3 -2] cb [2 1] cr [-1 3]",
			pwt.LumaWeight[0][0], pwt.ChromaWeight[0][0])
	}
}

func assertHigh14WeightedPFrames(t *testing.T, frames []*Frame) {
	t.Helper()
	if len(frames) != 2 {
		t.Fatalf("frames = %d, want 2", len(frames))
	}
	var rawVideo []byte
	for i, frame := range frames {
		if frame.Width != 16 || frame.Height != 16 ||
			frame.ChromaFormatIDC != 1 || frame.BitDepthLuma != 14 || frame.BitDepthChroma != 14 {
			t.Fatalf("frame[%d] format = %dx%d chroma %d depth %d/%d, want 16x16 yuv420p14le",
				i, frame.Width, frame.Height, frame.ChromaFormatIDC, frame.BitDepthLuma, frame.BitDepthChroma)
		}
		if got, err := frame.RawPixelFormat(); err != nil || got != "yuv420p14le" {
			t.Fatalf("frame[%d] RawPixelFormat = %q/%v, want yuv420p14le/nil", i, got, err)
		}
		if size, err := frame.RawYUVSize(); err != nil || size != high14WeightedPFrameRawSize {
			t.Fatalf("frame[%d] RawYUVSize = %d/%v, want %d/nil", i, size, err, high14WeightedPFrameRawSize)
		}
		raw, err := frame.AppendRawYUVBytesLE(nil)
		if err != nil {
			t.Fatalf("frame[%d] AppendRawYUVBytesLE: %v", i, err)
		}
		rawVideo = append(rawVideo, raw...)
		want := high14WeightedPIDRFrameMD5
		if i == 1 {
			want = high14WeightedPPFrameMD5
		}
		sum := md5.Sum(raw)
		if got := hex.EncodeToString(sum[:]); got != want {
			t.Fatalf("frame[%d] raw md5 = %s, want %s", i, got, want)
		}
		if _, err := frame.AppendRawYUV(nil); err != ErrUnsupported {
			t.Fatalf("frame[%d] AppendRawYUV high14 error = %v, want ErrUnsupported", i, err)
		}
	}
	if len(rawVideo) != 2*high14WeightedPFrameRawSize {
		t.Fatalf("rawvideo len = %d, want %d", len(rawVideo), 2*high14WeightedPFrameRawSize)
	}
	sum := md5.Sum(rawVideo)
	if got := hex.EncodeToString(sum[:]); got != high14WeightedPRawVideoMD5 {
		t.Fatalf("rawvideo md5 = %s, want %s", got, high14WeightedPRawVideoMD5)
	}
}

func assertFFmpegHigh14WeightedPRawVideoOracle(t *testing.T, data []byte) {
	t.Helper()
	path := writeTempH264(t, data)
	framemd5 := exec.Command("ffmpeg",
		"-v", "error",
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
	wantFrames := []string{high14WeightedPIDRFrameMD5, high14WeightedPPFrameMD5}
	for i, want := range wantFrames {
		line := []byte(fmt.Sprintf("0, %10d, %10d,        1, %8d, %s", i, i, high14WeightedPFrameRawSize, want))
		if !bytes.Contains(framemd5Out, line) {
			t.Fatalf("frame[%d] missing %q in framemd5:\n%s", i, line, framemd5Out)
		}
	}

	rawvideo := exec.Command("ffmpeg",
		"-v", "error",
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
	if len(raw) != 2*high14WeightedPFrameRawSize {
		t.Fatalf("rawvideo size = %d, want %d", len(raw), 2*high14WeightedPFrameRawSize)
	}
	sum := md5.Sum(raw)
	if got := hex.EncodeToString(sum[:]); got != high14WeightedPRawVideoMD5 {
		t.Fatalf("rawvideo md5 = %s, want %s", got, high14WeightedPRawVideoMD5)
	}
}
