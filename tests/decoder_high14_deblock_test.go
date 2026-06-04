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
	high14CAVLCIntraMode1DeblockBitstreamMD5 = "34fc065ec782ea6eb9940dfa2e225223"
	high14CAVLCIntraMode1DeblockFrameMD5     = "efff96b33bda86086ce433d1ca8ae196"
	high14CAVLCIntraMode1DeblockRawVideoMD5  = "efff96b33bda86086ce433d1ca8ae196"

	high14CAVLCP16x16LumaChromaMode1DeblockBitstreamMD5 = "1aad7fb20cd3db4229f45a8c34bdd22c"
	high14CAVLCP16x16LumaChromaMode1DeblockPFrameMD5    = "23270c92d77361b18113194baa743c51"
	high14CAVLCP16x16LumaChromaMode1DeblockRawVideoMD5  = "10a4514f7d5f399f641bf854204ec1c9"
)

type high14CAVLCMode1DeblockCase struct {
	name         string
	data         []byte
	sliceTypes   []int32
	bitstreamMD5 string
	frameMD5     []string
	rawVideoMD5  string
}

func TestHigh14CAVLCMode1DeblockFixtureSyntax(t *testing.T) {
	for _, tt := range high14CAVLCMode1DeblockCases() {
		t.Run(tt.name, func(t *testing.T) {
			assertHigh14CAVLCMode1DeblockFixtureSyntax(t, tt.data, tt.sliceTypes, tt.bitstreamMD5)
		})
	}
}

func TestDecodeAnnexBHigh14CAVLCMode1DeblockFrames(t *testing.T) {
	for _, tt := range high14CAVLCMode1DeblockCases() {
		t.Run(tt.name, func(t *testing.T) {
			assertHigh14CAVLCMode1DeblockFixtureSyntax(t, tt.data, tt.sliceTypes, tt.bitstreamMD5)

			frames, err := NewDecoder().DecodeAnnexBFrames(tt.data)
			if err != nil {
				t.Fatalf("decode High14 CAVLC mode-1 deblock Annex B: %v", err)
			}
			assertHigh14CAVLCMode1DeblockFrames(t, frames, tt.frameMD5, tt.rawVideoMD5)
		})
	}
}

func TestDecodeAVCHigh14CAVLCMode1DeblockFrames(t *testing.T) {
	for _, tt := range high14CAVLCMode1DeblockCases() {
		t.Run(tt.name, func(t *testing.T) {
			for _, nalLengthSize := range []int{2, 3, 4} {
				frames, err := NewDecoder().DecodeAVCFrames(annexBToAVC(t, tt.data, nalLengthSize), nalLengthSize)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: %v", nalLengthSize, err)
				}
				assertHigh14CAVLCMode1DeblockFrames(t, frames, tt.frameMD5, tt.rawVideoMD5)
			}
		})
	}
}

func TestDecodeConfiguredAVCHigh14CAVLCMode1DeblockFrames(t *testing.T) {
	for _, tt := range high14CAVLCMode1DeblockCases() {
		t.Run(tt.name, func(t *testing.T) {
			config, samples := annexBToAVCConfigAndSamples(t, tt.data, 4)
			if len(samples) != len(tt.frameMD5) {
				t.Fatalf("samples = %d, want %d", len(samples), len(tt.frameMD5))
			}

			dec := NewDecoder()
			if _, err := dec.ParseAVCDecoderConfigurationRecord(config); err != nil {
				t.Fatal(err)
			}
			var frames []*Frame
			for i, sample := range samples {
				frame, err := dec.DecodeConfiguredAVC(sample)
				if err != nil {
					t.Fatalf("sample[%d] decode High14 CAVLC mode-1 deblock: %v", i, err)
				}
				frames = append(frames, frame)
			}
			assertHigh14CAVLCMode1DeblockFrames(t, frames, tt.frameMD5, tt.rawVideoMD5)
		})
	}
}

func TestFFmpegRawVideoFrameMD5OracleHigh14CAVLCMode1Deblock(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}

	for _, tt := range high14CAVLCMode1DeblockCases() {
		t.Run(tt.name, func(t *testing.T) {
			assertFFmpegHigh14CAVLCMode1DeblockRawVideoOracle(t, tt.data, tt.frameMD5, tt.rawVideoMD5)
		})
	}
}

func high14CAVLCMode1DeblockCases() []high14CAVLCMode1DeblockCase {
	return []high14CAVLCMode1DeblockCase{
		{
			name:         "intra-luma-chroma",
			data:         high14CAVLCIntraMode1DeblockFixture(),
			sliceTypes:   []int32{h264.PictureTypeI},
			bitstreamMD5: high14CAVLCIntraMode1DeblockBitstreamMD5,
			frameMD5:     []string{high14CAVLCIntraMode1DeblockFrameMD5},
			rawVideoMD5:  high14CAVLCIntraMode1DeblockRawVideoMD5,
		},
		{
			name:         "p16x16-luma-chroma",
			data:         high14CAVLCP16x16LumaChromaMode1DeblockFixture(),
			sliceTypes:   []int32{h264.PictureTypeI, h264.PictureTypeP},
			bitstreamMD5: high14CAVLCP16x16LumaChromaMode1DeblockBitstreamMD5,
			frameMD5: []string{
				high14CAVLCIntraMode1DeblockFrameMD5,
				high14CAVLCP16x16LumaChromaMode1DeblockPFrameMD5,
			},
			rawVideoMD5: high14CAVLCP16x16LumaChromaMode1DeblockRawVideoMD5,
		},
	}
}

func high14CAVLCIntraMode1DeblockFixture() []byte {
	return buildHighIntraAnnexBFixture(14, highIntra16x16ResidualMode1DeblockSliceRBSP(highIntra16x16LumaChromaPayloadBits))
}

func high14CAVLCP16x16LumaChromaMode1DeblockFixture() []byte {
	var data []byte
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALSPS), highInterSPSRBSP(14)))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALPPS), highIntraPCMPPSRBSP(14)))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALIDRSlice), highIntra16x16ResidualMode1DeblockSliceRBSP(highIntra16x16LumaChromaPayloadBits)))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALSlice), highInterP16x16LumaChromaMode1DeblockSliceRBSP()))
	return data
}

func highIntra16x16ResidualMode1DeblockSliceRBSP(payloadBits string) []byte {
	var b decoderSEIBitBuilder
	b.writeUE(0)
	b.writeUE(2)
	b.writeUE(0)
	b.writeBits(0, 4)
	b.writeUE(0)
	b.writeBit(0)
	b.writeBit(0)
	b.writeSE(0)
	writeHighCAVLCMode1DeblockSyntax(&b)
	highIntra16x16WritePayloadBits(&b, payloadBits)
	return b.rbsp()
}

func highInterP16x16LumaChromaMode1DeblockSliceRBSP() []byte {
	var b decoderSEIBitBuilder
	b.writeUE(0)
	b.writeUE(0)
	b.writeUE(0)
	b.writeBits(1, 4)
	b.writeBit(0)
	b.writeBit(0)
	b.writeBit(0)
	b.writeSE(0)
	writeHighCAVLCMode1DeblockSyntax(&b)
	highIntra16x16WritePayloadBits(&b, highInterP16x16LumaChromaResidualPayloadBits)
	return b.rbsp()
}

func writeHighCAVLCMode1DeblockSyntax(b *decoderSEIBitBuilder) {
	b.writeUE(0)
	b.writeSE(0)
	b.writeSE(0)
}

func assertHigh14CAVLCMode1DeblockFixtureSyntax(t *testing.T, data []byte, wantSliceTypes []int32, wantBitstreamMD5 string) {
	t.Helper()
	sum := md5.Sum(data)
	if got := hex.EncodeToString(sum[:]); got != wantBitstreamMD5 {
		t.Fatalf("bitstream md5 = %s, want %s", got, wantBitstreamMD5)
	}
	nals, err := h264.SplitAnnexB(data)
	if err != nil {
		t.Fatal(err)
	}

	var spsList [32]*h264.SPS
	var ppsList [256]*h264.PPS
	var gotSliceTypes []int32
	for _, nal := range nals {
		switch nal.Type {
		case h264.NALSPS:
			sps, err := h264.DecodeSPS(nal.RBSP)
			if err != nil {
				t.Fatal(err)
			}
			if sps.ProfileIDC != 244 || sps.Width != 16 || sps.Height != 16 ||
				sps.ChromaFormatIDC != 1 || sps.BitDepthLuma != 14 || sps.BitDepthChroma != 14 {
				t.Fatalf("SPS profile/format = %d %dx%d chroma %d depth %d/%d, want High14 4:2:0 16x16",
					sps.ProfileIDC, sps.Width, sps.Height, sps.ChromaFormatIDC, sps.BitDepthLuma, sps.BitDepthChroma)
			}
			spsList[sps.SPSID] = sps
		case h264.NALPPS:
			pps, err := h264.DecodePPS(nal.RBSP, &spsList)
			if err != nil {
				t.Fatal(err)
			}
			if pps.CABAC != 0 || pps.Transform8x8Mode != 0 || pps.DeblockingFilterParametersPresent == 0 ||
				pps.RefCount[0] != 1 || pps.RefCount[1] != 1 {
				t.Fatalf("PPS CABAC/8x8/deblock-present/refs = %d/%d/%d/%d/%d, want CAVLC/no-8x8/deblock params/1/1",
					pps.CABAC, pps.Transform8x8Mode, pps.DeblockingFilterParametersPresent, pps.RefCount[0], pps.RefCount[1])
			}
			ppsList[pps.PPSID] = pps
		case h264.NALIDRSlice, h264.NALSlice:
			sh, err := h264.ParseSliceHeader(nal, &ppsList)
			if err != nil {
				t.Fatal(err)
			}
			if sh.PictureStructure != h264.PictureFrame || sh.DeblockingFilter != 1 ||
				sh.SliceAlphaC0Offset != 0 || sh.SliceBetaOffset != 0 || sh.QScale != 26 {
				t.Fatalf("slice picture/deblock/offsets/qp = %d/%d/%d/%d/%d, want frame/mode-1/0/0/26",
					sh.PictureStructure, sh.DeblockingFilter, sh.SliceAlphaC0Offset, sh.SliceBetaOffset, sh.QScale)
			}
			gotSliceTypes = append(gotSliceTypes, sh.SliceTypeNoS)
		default:
			t.Fatalf("unexpected NAL type %d in High14 CAVLC mode-1 deblock fixture", nal.Type)
		}
	}
	if len(gotSliceTypes) != len(wantSliceTypes) {
		t.Fatalf("slice types = %v, want %v", gotSliceTypes, wantSliceTypes)
	}
	for i := range wantSliceTypes {
		if gotSliceTypes[i] != wantSliceTypes[i] {
			t.Fatalf("slice types = %v, want %v", gotSliceTypes, wantSliceTypes)
		}
	}
}

func assertHigh14CAVLCMode1DeblockFrames(t *testing.T, frames []*Frame, wantFrameMD5 []string, wantRawVideoMD5 string) {
	t.Helper()
	if len(frames) != len(wantFrameMD5) {
		t.Fatalf("frames = %d, want %d", len(frames), len(wantFrameMD5))
	}
	var rawVideo []byte
	for i, frame := range frames {
		if frame.Width != 16 || frame.Height != 16 ||
			frame.ChromaFormatIDC != 1 || frame.BitDepthLuma != 14 || frame.BitDepthChroma != 14 {
			t.Fatalf("frame[%d] format = %dx%d chroma %d depth %d/%d, want 16x16 yuv420p14le",
				i, frame.Width, frame.Height, frame.ChromaFormatIDC, frame.BitDepthLuma, frame.BitDepthChroma)
		}
		if pixFmt, err := frame.RawPixelFormat(); err != nil || pixFmt != "yuv420p14le" {
			t.Fatalf("frame[%d] RawPixelFormat = %q/%v, want yuv420p14le/nil", i, pixFmt, err)
		}
		if size, err := frame.RawYUVSize(); err != nil || size != 768 {
			t.Fatalf("frame[%d] RawYUVSize = %d/%v, want 768/nil", i, size, err)
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
	if got := hex.EncodeToString(sum[:]); got != wantRawVideoMD5 {
		t.Fatalf("rawvideo md5 = %s, want %s", got, wantRawVideoMD5)
	}
}

func assertFFmpegHigh14CAVLCMode1DeblockRawVideoOracle(t *testing.T, data []byte, wantFrameMD5 []string, wantRawVideoMD5 string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "high14_mode1_deblock.h264")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	framemd5 := exec.Command("ffmpeg",
		"-v", "error",
		"-xerror",
		"-f", "h264",
		"-i", path,
		"-an", "-sn", "-dn",
		"-pix_fmt", "yuv420p14le",
		"-f", "framemd5",
		"-")
	out, err := framemd5.Output()
	if err != nil {
		t.Fatalf("ffmpeg framemd5: %v", err)
	}
	for i, want := range wantFrameMD5 {
		line := []byte(fmt.Sprintf(",        1,      768, %s", want))
		if !bytes.Contains(out, line) {
			t.Fatalf("ffmpeg framemd5 missing frame[%d] %s:\n%s", i, want, out)
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
		"-")
	raw, err := rawvideo.Output()
	if err != nil {
		t.Fatalf("ffmpeg rawvideo: %v", err)
	}
	if len(raw) != len(wantFrameMD5)*768 {
		t.Fatalf("ffmpeg rawvideo size = %d, want %d", len(raw), len(wantFrameMD5)*768)
	}
	sum := md5.Sum(raw)
	if got := hex.EncodeToString(sum[:]); got != wantRawVideoMD5 {
		t.Fatalf("ffmpeg rawvideo md5 = %s, want %s", got, wantRawVideoMD5)
	}
}
