// SPDX-License-Identifier: LGPL-2.1-or-later

package goh264_test

import (
	"crypto/md5"
	"encoding/hex"
	"os"
	"os/exec"
	"testing"

	"github.com/thesyncim/goh264/internal/h264"
)

type high12ChromaWeightedPCase struct {
	name           string
	chromaFormat   uint32
	chromaWeighted bool
	disableDeblock uint32
	deblockMode    int32
	pixFmt         string
	frameSize      int
	bitstreamMD5   string
	rawVideoMD5    string
	frameMD5       []string
}

func TestHigh12ChromaWeightedPFixtureSyntax(t *testing.T) {
	for _, tt := range high12ChromaWeightedPCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := high12ChromaWeightedPFixture(t, tt)
			assertHigh12ChromaWeightedPFixtureSyntax(t, data, tt)
		})
	}
}

func TestDecodeAnnexBHigh12ChromaWeightedPFrames(t *testing.T) {
	for _, tt := range high12ChromaWeightedPCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := high12ChromaWeightedPFixture(t, tt)
			assertHigh12ChromaWeightedPFixtureSyntax(t, data, tt)
			frames, err := NewDecoder().DecodeAnnexBFrames(data)
			if err != nil {
				t.Fatalf("DecodeAnnexBFrames: %v", err)
			}
			assertHigh12ChromaWeightedPFrames(t, frames, tt)
		})
	}
}

func TestDecodeAVCHigh12ChromaWeightedPFrames(t *testing.T) {
	for _, tt := range high12ChromaWeightedPCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := high12ChromaWeightedPFixture(t, tt)
			assertHigh12ChromaWeightedPFixtureSyntax(t, data, tt)
			for _, nalLengthSize := range []int{2, 3, 4} {
				frames, err := NewDecoder().DecodeAVCFrames(annexBToAVC(t, data, nalLengthSize), nalLengthSize)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: DecodeAVCFrames: %v", nalLengthSize, err)
				}
				assertHigh12ChromaWeightedPFrames(t, frames, tt)
			}
		})
	}
}

func TestDecodeConfiguredAVCHigh12ChromaWeightedPFramesAcrossSamples(t *testing.T) {
	for _, tt := range high12ChromaWeightedPCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := high12ChromaWeightedPFixture(t, tt)
			assertHigh12ChromaWeightedPFixtureSyntax(t, data, tt)
			for _, nalLengthSize := range []int{2, 3, 4} {
				config, samples := annexBToAVCConfigAndSamples(t, data, nalLengthSize)
				if len(samples) != len(tt.frameMD5) {
					t.Fatalf("nalLengthSize=%d samples = %d, want %d", nalLengthSize, len(samples), len(tt.frameMD5))
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
				assertHigh12ChromaWeightedPFrames(t, frames, tt)
			}
		})
	}
}

func TestFFmpegRawVideoFrameMD5OracleHigh12ChromaWeightedP(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}
	for _, tt := range high12ChromaWeightedPCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := high12ChromaWeightedPFixture(t, tt)
			assertHigh12ChromaWeightedPFixtureSyntax(t, data, tt)
			assertFFmpegHigh12ChromaWeightedPRawVideoOracle(t, data, tt)
		})
	}
}

func high12ChromaWeightedPCases() []high12ChromaWeightedPCase {
	return []high12ChromaWeightedPCase{
		{
			name:           "422-luma-chroma-no-deblock",
			chromaFormat:   2,
			chromaWeighted: true,
			disableDeblock: 1,
			deblockMode:    0,
			pixFmt:         "yuv422p12le",
			frameSize:      1024,
			bitstreamMD5:   "a26339d59cc8a6ffd608521197c522f0",
			rawVideoMD5:    "2e33d4f63337571ab10f0f82f6ab0175",
			frameMD5:       []string{"0556a969d4e9ee0393d6007103ccfdae", "e7af38d6a756543d47197a7a2703a5e9"},
		},
		{
			name:           "422-luma-chroma-frame-deblock",
			chromaFormat:   2,
			chromaWeighted: true,
			disableDeblock: 0,
			deblockMode:    1,
			pixFmt:         "yuv422p12le",
			frameSize:      1024,
			bitstreamMD5:   "1d8930290883e572f5a96581eba88276",
			rawVideoMD5:    "2e33d4f63337571ab10f0f82f6ab0175",
			frameMD5:       []string{"0556a969d4e9ee0393d6007103ccfdae", "e7af38d6a756543d47197a7a2703a5e9"},
		},
		{
			name:           "422-luma-chroma-slice-boundary",
			chromaFormat:   2,
			chromaWeighted: true,
			disableDeblock: 2,
			deblockMode:    2,
			pixFmt:         "yuv422p12le",
			frameSize:      1024,
			bitstreamMD5:   "6dd034952f1a85e29773d706cebe5cb2",
			rawVideoMD5:    "2e33d4f63337571ab10f0f82f6ab0175",
			frameMD5:       []string{"0556a969d4e9ee0393d6007103ccfdae", "e7af38d6a756543d47197a7a2703a5e9"},
		},
		{
			name:           "444-luma-chroma-no-deblock",
			chromaFormat:   3,
			chromaWeighted: true,
			disableDeblock: 1,
			deblockMode:    0,
			pixFmt:         "yuv444p12le",
			frameSize:      1536,
			bitstreamMD5:   "8200e38aa1ce591d38e3eae50063d60c",
			rawVideoMD5:    "2ce086c35a06fdeb684611b4ecd88600",
			frameMD5:       []string{"d4fbc620f3b054967df19b16392bdc6e", "ae078b14f7dd6a289bd0162f2570f504"},
		},
		{
			name:           "444-luma-chroma-frame-deblock",
			chromaFormat:   3,
			chromaWeighted: true,
			disableDeblock: 0,
			deblockMode:    1,
			pixFmt:         "yuv444p12le",
			frameSize:      1536,
			bitstreamMD5:   "97be8d4fe258354089c2168425b45279",
			rawVideoMD5:    "2ce086c35a06fdeb684611b4ecd88600",
			frameMD5:       []string{"d4fbc620f3b054967df19b16392bdc6e", "ae078b14f7dd6a289bd0162f2570f504"},
		},
		{
			name:           "444-luma-chroma-slice-boundary",
			chromaFormat:   3,
			chromaWeighted: true,
			disableDeblock: 2,
			deblockMode:    2,
			pixFmt:         "yuv444p12le",
			frameSize:      1536,
			bitstreamMD5:   "159fdb0c040ee81feaed2b581cce5802",
			rawVideoMD5:    "2ce086c35a06fdeb684611b4ecd88600",
			frameMD5:       []string{"d4fbc620f3b054967df19b16392bdc6e", "ae078b14f7dd6a289bd0162f2570f504"},
		},
		{
			name:           "422-luma-only-no-deblock",
			chromaFormat:   2,
			disableDeblock: 1,
			deblockMode:    0,
			pixFmt:         "yuv422p12le",
			frameSize:      1024,
			bitstreamMD5:   "5babb3d126f32f4273a93c178179360d",
			rawVideoMD5:    "a19c71eb203876d857f21cb00d56d6e1",
			frameMD5:       []string{"0556a969d4e9ee0393d6007103ccfdae", "9e89cc7eb714886ad3aa02325f80f166"},
		},
		{
			name:           "422-luma-only-frame-deblock",
			chromaFormat:   2,
			disableDeblock: 0,
			deblockMode:    1,
			pixFmt:         "yuv422p12le",
			frameSize:      1024,
			bitstreamMD5:   "b7478dcd8b2f016ebb11153db8e12c03",
			rawVideoMD5:    "a19c71eb203876d857f21cb00d56d6e1",
			frameMD5:       []string{"0556a969d4e9ee0393d6007103ccfdae", "9e89cc7eb714886ad3aa02325f80f166"},
		},
		{
			name:           "422-luma-only-slice-boundary",
			chromaFormat:   2,
			disableDeblock: 2,
			deblockMode:    2,
			pixFmt:         "yuv422p12le",
			frameSize:      1024,
			bitstreamMD5:   "fd6b2cddd1aa3fcf0f78a5099bee4e00",
			rawVideoMD5:    "a19c71eb203876d857f21cb00d56d6e1",
			frameMD5:       []string{"0556a969d4e9ee0393d6007103ccfdae", "9e89cc7eb714886ad3aa02325f80f166"},
		},
		{
			name:           "444-luma-only-no-deblock",
			chromaFormat:   3,
			disableDeblock: 1,
			deblockMode:    0,
			pixFmt:         "yuv444p12le",
			frameSize:      1536,
			bitstreamMD5:   "ec89d3c7691cc9abe042454c3ddab332",
			rawVideoMD5:    "d20a61b01ea9842311a29d54e7176f6e",
			frameMD5:       []string{"d4fbc620f3b054967df19b16392bdc6e", "29b5196afb78bf2970e35988fcfb7af7"},
		},
		{
			name:           "444-luma-only-frame-deblock",
			chromaFormat:   3,
			disableDeblock: 0,
			deblockMode:    1,
			pixFmt:         "yuv444p12le",
			frameSize:      1536,
			bitstreamMD5:   "b9689a89deb982f41cbfa396dc1b82cf",
			rawVideoMD5:    "d20a61b01ea9842311a29d54e7176f6e",
			frameMD5:       []string{"d4fbc620f3b054967df19b16392bdc6e", "29b5196afb78bf2970e35988fcfb7af7"},
		},
		{
			name:           "444-luma-only-slice-boundary",
			chromaFormat:   3,
			disableDeblock: 2,
			deblockMode:    2,
			pixFmt:         "yuv444p12le",
			frameSize:      1536,
			bitstreamMD5:   "15bba47777d457c040bc8fbd399d1139",
			rawVideoMD5:    "d20a61b01ea9842311a29d54e7176f6e",
			frameMD5:       []string{"d4fbc620f3b054967df19b16392bdc6e", "29b5196afb78bf2970e35988fcfb7af7"},
		},
	}
}

func high12ChromaWeightedPFixture(t *testing.T, tt high12ChromaWeightedPCase) []byte {
	t.Helper()
	var data []byte
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALSPS), high12ChromaWeightedPSPSRBSP(tt.chromaFormat)))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALPPS), high12ChromaWeightedPPPSRBSP()))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALIDRSlice), high12ChromaWeightedPIDRSliceRBSP(tt.chromaFormat, tt.disableDeblock)))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALSlice), high12ChromaWeightedP16x16NoResidualSliceRBSP(tt.disableDeblock, tt.chromaWeighted)))
	sum := md5.Sum(data)
	if got := hex.EncodeToString(sum[:]); got != tt.bitstreamMD5 {
		t.Fatalf("%s bitstream md5 = %s, want %s", tt.name, got, tt.bitstreamMD5)
	}
	return data
}

func high12ChromaWeightedPIDRSliceRBSP(chromaFormat uint32, disableDeblockingFilterIDC uint32) []byte {
	payloadBits := highIntra16x16NoResidualPayloadBits
	if chromaFormat == 3 {
		payloadBits = "001001111"
	}
	return highIntra16x16ResidualDeblockSliceRBSP(payloadBits, disableDeblockingFilterIDC)
}

func high12ChromaWeightedPSPSRBSP(chromaFormat uint32) []byte {
	var b decoderSEIBitBuilder
	b.writeBits(244, 8)
	b.writeBits(0, 8)
	b.writeBits(10, 8)
	b.writeUE(0)
	b.writeUE(chromaFormat)
	if chromaFormat == 3 {
		b.writeBit(0)
	}
	b.writeUE(4)
	b.writeUE(4)
	b.writeBit(0)
	b.writeBit(0)
	b.writeUE(0)
	b.writeUE(2)
	b.writeUE(1)
	b.writeBit(0)
	b.writeUE(0)
	b.writeUE(0)
	b.writeBit(1)
	b.writeBit(1)
	b.writeBit(0)
	b.writeBit(0)
	return b.rbsp()
}

func high12ChromaWeightedPPPSRBSP() []byte {
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
	b.writeSE(-24)
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

func high12ChromaWeightedP16x16NoResidualSliceRBSP(disableDeblockingFilterIDC uint32, chromaWeighted bool) []byte {
	var b decoderSEIBitBuilder
	b.writeUE(0)
	b.writeUE(0)
	b.writeUE(0)
	b.writeBits(1, 4)
	b.writeBit(0)
	b.writeBit(0)
	if chromaWeighted {
		writeHighWeightedPPredWeightSyntax(&b)
	} else {
		writeHighLumaOnlyWeightedPPredWeightSyntax(&b)
	}
	b.writeBit(0)
	b.writeSE(0)
	writeHighCAVLCDeblockSyntax(&b, disableDeblockingFilterIDC)
	highIntra16x16WritePayloadBits(&b, highInterP16x16NoResidualPayloadBits)
	return b.rbsp()
}

func writeHighLumaOnlyWeightedPPredWeightSyntax(b *decoderSEIBitBuilder) {
	b.writeUE(2)
	b.writeUE(1)
	b.writeBit(1)
	b.writeSE(3)
	b.writeSE(-2)
	b.writeBit(0)
}

func assertHigh12ChromaWeightedPFixtureSyntax(t *testing.T, data []byte, tt high12ChromaWeightedPCase) {
	t.Helper()
	nals, err := h264.SplitAnnexB(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(nals) != 4 {
		t.Fatalf("NAL count = %d, want SPS/PPS/IDR/P", len(nals))
	}

	var spsList [32]*h264.SPS
	var ppsList [256]*h264.PPS
	var gotVCL []h264.NALUnitType
	var pSlices int
	var lumaWeightedPSlices int
	var chromaWeightedPSlices int
	for _, nal := range nals {
		switch nal.Type {
		case h264.NALSPS:
			sps, err := h264.DecodeSPS(nal.RBSP)
			if err != nil {
				t.Fatal(err)
			}
			if sps.ProfileIDC != 244 || sps.Width != 16 || sps.Height != 16 ||
				sps.ChromaFormatIDC != tt.chromaFormat || sps.BitDepthLuma != 12 || sps.BitDepthChroma != 12 ||
				sps.FrameMBSOnlyFlag != 1 || sps.MBAFF != 0 || sps.RefFrameCount != 1 {
				t.Fatalf("SPS profile/format = %d %dx%d chroma %d depth %d/%d frame_only/mbaff %d/%d refs %d, want %s",
					sps.ProfileIDC, sps.Width, sps.Height, sps.ChromaFormatIDC, sps.BitDepthLuma, sps.BitDepthChroma,
					sps.FrameMBSOnlyFlag, sps.MBAFF, sps.RefFrameCount, tt.name)
			}
			spsList[sps.SPSID] = sps
		case h264.NALPPS:
			pps, err := h264.DecodePPS(nal.RBSP, &spsList)
			if err != nil {
				t.Fatal(err)
			}
			if pps.CABAC != 0 || pps.Transform8x8Mode != 0 || pps.WeightedPred != 1 ||
				pps.WeightedBipredIDC != 0 || pps.RefCount[0] != 1 || pps.RefCount[1] != 1 ||
				pps.DeblockingFilterParametersPresent == 0 {
				t.Fatalf("PPS cabac/8x8/weights/refs/deblock-present = %d/%d/%d/%d/%d/%d/%d, want CAVLC/no-8x8 weighted P ref=1",
					pps.CABAC, pps.Transform8x8Mode, pps.WeightedPred, pps.WeightedBipredIDC,
					pps.RefCount[0], pps.RefCount[1], pps.DeblockingFilterParametersPresent)
			}
			ppsList[pps.PPSID] = pps
		case h264.NALIDRSlice, h264.NALSlice:
			sh, err := h264.ParseSliceHeader(nal, &ppsList)
			if err != nil {
				t.Fatal(err)
			}
			gotVCL = append(gotVCL, nal.Type)
			if sh.PictureStructure != h264.PictureFrame || sh.DeblockingFilter != tt.deblockMode ||
				sh.SliceAlphaC0Offset != 0 || sh.SliceBetaOffset != 0 || sh.QScale != 26 {
				t.Fatalf("slice picture/deblock/offsets/qp = %d/%d/%d/%d/%d, want frame/mode-%d/0/0/26",
					sh.PictureStructure, sh.DeblockingFilter, sh.SliceAlphaC0Offset, sh.SliceBetaOffset, sh.QScale, tt.deblockMode)
			}
			if sh.SliceTypeNoS == h264.PictureTypeP {
				if sh.ListCount != 1 || sh.RefCount[0] != 1 || sh.PPS == nil || sh.PPS.WeightedPred != 1 {
					t.Fatalf("P slice lists/ref/weighted-p = %d/%d/%v, want one L0 ref with weighted-P PPS",
						sh.ListCount, sh.RefCount[0], sh.PPS)
				}
				pSlices++
				if sh.PredWeightTable.UseWeight != 0 {
					lumaWeightedPSlices++
				}
				if sh.PredWeightTable.UseWeightChroma != 0 {
					chromaWeightedPSlices++
				}
			}
		default:
			t.Fatalf("unexpected NAL type %d in %s", nal.Type, tt.name)
		}
	}
	if len(gotVCL) != 2 || gotVCL[0] != h264.NALIDRSlice || gotVCL[1] != h264.NALSlice {
		t.Fatalf("VCL NALs = %v, want IDR/P", gotVCL)
	}
	if pSlices != 1 {
		t.Fatalf("P slices = %d, want 1", pSlices)
	}
	if lumaWeightedPSlices == 0 {
		t.Fatal("weighted-P fixture has no luma-weighted P slices")
	}
	if tt.chromaWeighted && chromaWeightedPSlices == 0 {
		t.Fatal("weighted-P fixture has no chroma-weighted P slices")
	}
	if !tt.chromaWeighted && chromaWeightedPSlices != 0 {
		t.Fatalf("luma-only weighted-P fixture has %d chroma-weighted P slices, want 0", chromaWeightedPSlices)
	}
}

func assertHigh12ChromaWeightedPFrames(t *testing.T, frames []*Frame, tt high12ChromaWeightedPCase) {
	t.Helper()
	if len(frames) != len(tt.frameMD5) {
		t.Fatalf("frames = %d, want %d", len(frames), len(tt.frameMD5))
	}
	rawVideo := make([]byte, 0, len(frames)*tt.frameSize)
	for i, frame := range frames {
		if frame == nil {
			t.Fatalf("frame[%d] is nil", i)
		}
		if frame.Width != 16 || frame.Height != 16 ||
			frame.ChromaFormatIDC != tt.chromaFormat ||
			frame.BitDepthLuma != 12 || frame.BitDepthChroma != 12 {
			t.Fatalf("frame[%d] metadata = %dx%d chroma %d depth %d/%d, want 16x16 chroma %d High12",
				i, frame.Width, frame.Height, frame.ChromaFormatIDC, frame.BitDepthLuma, frame.BitDepthChroma,
				tt.chromaFormat)
		}
		if got, err := frame.RawPixelFormat(); err != nil || got != tt.pixFmt {
			t.Fatalf("frame[%d] RawPixelFormat = %q/%v, want %s/nil", i, got, err, tt.pixFmt)
		}
		raw, err := frame.AppendRawYUVBytesLE(nil)
		if err != nil {
			t.Fatalf("frame[%d] AppendRawYUVBytesLE: %v", i, err)
		}
		if len(raw) != tt.frameSize {
			t.Fatalf("frame[%d] raw len = %d, want %d", i, len(raw), tt.frameSize)
		}
		sum := md5.Sum(raw)
		if got := hex.EncodeToString(sum[:]); got != tt.frameMD5[i] {
			t.Fatalf("frame[%d] md5 = %s, want %s", i, got, tt.frameMD5[i])
		}
		rawVideo = append(rawVideo, raw...)
	}
	sum := md5.Sum(rawVideo)
	if got := hex.EncodeToString(sum[:]); got != tt.rawVideoMD5 {
		t.Fatalf("rawvideo md5 = %s, want %s", got, tt.rawVideoMD5)
	}
}

func assertFFmpegHigh12ChromaWeightedPRawVideoOracle(t *testing.T, data []byte, tt high12ChromaWeightedPCase) {
	t.Helper()
	path := writeTempH264(t, data)
	frames, err := h264FFmpegFrameMD5s("ffmpeg", path, tt.pixFmt)
	if err != nil {
		t.Fatalf("ffmpeg framemd5: %v", err)
	}
	if len(frames) != len(tt.frameMD5) {
		t.Fatalf("ffmpeg frame md5 count = %d, want %d", len(frames), len(tt.frameMD5))
	}
	for i, want := range tt.frameMD5 {
		if frames[i] != want {
			t.Fatalf("ffmpeg frame[%d] md5 = %s, want %s", i, frames[i], want)
		}
	}
	raw, err := h264FFmpegRawVideoBytes("ffmpeg", path, tt.pixFmt)
	if err != nil {
		t.Fatalf("ffmpeg rawvideo: %v", err)
	}
	if len(raw) != len(tt.frameMD5)*tt.frameSize {
		t.Fatalf("ffmpeg rawvideo size = %d, want %d", len(raw), len(tt.frameMD5)*tt.frameSize)
	}
	sum := md5.Sum(raw)
	if got := hex.EncodeToString(sum[:]); got != tt.rawVideoMD5 {
		t.Fatalf("ffmpeg rawvideo md5 = %s, want %s", got, tt.rawVideoMD5)
	}
}
