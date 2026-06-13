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

type high14ChromaWeightedPCase struct {
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

func TestHigh14ChromaWeightedPFixtureSyntax(t *testing.T) {
	for _, tt := range high14ChromaWeightedPCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := high14ChromaWeightedPFixture(t, tt)
			assertHigh14ChromaWeightedPFixtureSyntax(t, data, tt)
		})
	}
}

func TestDecodeAnnexBHigh14ChromaWeightedPFrames(t *testing.T) {
	for _, tt := range high14ChromaWeightedPCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := high14ChromaWeightedPFixture(t, tt)
			assertHigh14ChromaWeightedPFixtureSyntax(t, data, tt)
			frames, err := NewDecoder().DecodeAnnexBFrames(data)
			if err != nil {
				t.Fatalf("DecodeAnnexBFrames: %v", err)
			}
			assertHigh14ChromaWeightedPFrames(t, frames, tt)
		})
	}
}

func TestDecodeAVCHigh14ChromaWeightedPFrames(t *testing.T) {
	for _, tt := range high14ChromaWeightedPCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := high14ChromaWeightedPFixture(t, tt)
			assertHigh14ChromaWeightedPFixtureSyntax(t, data, tt)
			for _, nalLengthSize := range []int{2, 3, 4} {
				frames, err := NewDecoder().DecodeAVCFrames(annexBToAVC(t, data, nalLengthSize), nalLengthSize)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: DecodeAVCFrames: %v", nalLengthSize, err)
				}
				assertHigh14ChromaWeightedPFrames(t, frames, tt)
			}
		})
	}
}

func TestDecodeConfiguredAVCHigh14ChromaWeightedPFramesAcrossSamples(t *testing.T) {
	for _, tt := range high14ChromaWeightedPCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := high14ChromaWeightedPFixture(t, tt)
			assertHigh14ChromaWeightedPFixtureSyntax(t, data, tt)
			for _, nalLengthSize := range []int{2, 3, 4} {
				config, samples := annexBToAVCConfigAndSamples(t, data, nalLengthSize)
				if len(samples) != len(tt.frameMD5) {
					t.Fatalf("nalLengthSize=%d samples = %d, want %d", nalLengthSize, len(samples), len(tt.frameMD5))
				}
				dec := NewDecoder()
				if _, err := dec.ConfigureAVCDecoderConfigurationRecord(config); err != nil {
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
				assertHigh14ChromaWeightedPFrames(t, frames, tt)
			}
		})
	}
}

func TestFFmpegRawVideoFrameMD5OracleHigh14ChromaWeightedP(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}
	for _, tt := range high14ChromaWeightedPCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := high14ChromaWeightedPFixture(t, tt)
			assertHigh14ChromaWeightedPFixtureSyntax(t, data, tt)
			assertFFmpegHigh14ChromaWeightedPRawVideoOracle(t, data, tt)
		})
	}
}

func high14ChromaWeightedPCases() []high14ChromaWeightedPCase {
	return []high14ChromaWeightedPCase{
		{
			name:           "422-luma-chroma-no-deblock",
			chromaFormat:   2,
			chromaWeighted: true,
			disableDeblock: 1,
			deblockMode:    0,
			pixFmt:         "yuv422p14le",
			frameSize:      1024,
			bitstreamMD5:   "1e9c2d0c13ba29993962cf08127797e2",
			rawVideoMD5:    "c9160f9f6efe9151dd177a806f029e5a",
			frameMD5:       []string{"6505470aec88d1552f817a1be891a67c", "47374b0818dfe5c9d97d627249980189"},
		},
		{
			name:           "422-luma-chroma-frame-deblock",
			chromaFormat:   2,
			chromaWeighted: true,
			disableDeblock: 0,
			deblockMode:    1,
			pixFmt:         "yuv422p14le",
			frameSize:      1024,
			bitstreamMD5:   "0a76888d86c1b6146bcde7331d0a354b",
			rawVideoMD5:    "c9160f9f6efe9151dd177a806f029e5a",
			frameMD5:       []string{"6505470aec88d1552f817a1be891a67c", "47374b0818dfe5c9d97d627249980189"},
		},
		{
			name:           "422-luma-chroma-slice-boundary",
			chromaFormat:   2,
			chromaWeighted: true,
			disableDeblock: 2,
			deblockMode:    2,
			pixFmt:         "yuv422p14le",
			frameSize:      1024,
			bitstreamMD5:   "f0acefa5a0edb504b2c648fe75ffb284",
			rawVideoMD5:    "c9160f9f6efe9151dd177a806f029e5a",
			frameMD5:       []string{"6505470aec88d1552f817a1be891a67c", "47374b0818dfe5c9d97d627249980189"},
		},
		{
			name:           "444-luma-chroma-no-deblock",
			chromaFormat:   3,
			chromaWeighted: true,
			disableDeblock: 1,
			deblockMode:    0,
			pixFmt:         "yuv444p14le",
			frameSize:      1536,
			bitstreamMD5:   "e456ba4a55857aea41a5f4df499ff442",
			rawVideoMD5:    "a4eff17bce5b952fee072b581770e6aa",
			frameMD5:       []string{"522820c0c78e0aee053dfe1aae6269b7", "319d5a2b09a986172f3815a8d298818c"},
		},
		{
			name:           "444-luma-chroma-frame-deblock",
			chromaFormat:   3,
			chromaWeighted: true,
			disableDeblock: 0,
			deblockMode:    1,
			pixFmt:         "yuv444p14le",
			frameSize:      1536,
			bitstreamMD5:   "c5a681128fa5c3b722b09fa598f51479",
			rawVideoMD5:    "a4eff17bce5b952fee072b581770e6aa",
			frameMD5:       []string{"522820c0c78e0aee053dfe1aae6269b7", "319d5a2b09a986172f3815a8d298818c"},
		},
		{
			name:           "444-luma-chroma-slice-boundary",
			chromaFormat:   3,
			chromaWeighted: true,
			disableDeblock: 2,
			deblockMode:    2,
			pixFmt:         "yuv444p14le",
			frameSize:      1536,
			bitstreamMD5:   "ede8738249008685dbdae37c94570ca2",
			rawVideoMD5:    "a4eff17bce5b952fee072b581770e6aa",
			frameMD5:       []string{"522820c0c78e0aee053dfe1aae6269b7", "319d5a2b09a986172f3815a8d298818c"},
		},
		{
			name:           "422-luma-only-no-deblock",
			chromaFormat:   2,
			disableDeblock: 1,
			deblockMode:    0,
			pixFmt:         "yuv422p14le",
			frameSize:      1024,
			bitstreamMD5:   "dcb56a19f2fb367a56bef6d24d355697",
			rawVideoMD5:    "7198e8fa4fb907711a070364bf8f3fd8",
			frameMD5:       []string{"6505470aec88d1552f817a1be891a67c", "78a76a300a1dcb00e4892cd297911c29"},
		},
		{
			name:           "422-luma-only-frame-deblock",
			chromaFormat:   2,
			disableDeblock: 0,
			deblockMode:    1,
			pixFmt:         "yuv422p14le",
			frameSize:      1024,
			bitstreamMD5:   "9e7f9167078ab3bbc7440b982b04e633",
			rawVideoMD5:    "7198e8fa4fb907711a070364bf8f3fd8",
			frameMD5:       []string{"6505470aec88d1552f817a1be891a67c", "78a76a300a1dcb00e4892cd297911c29"},
		},
		{
			name:           "422-luma-only-slice-boundary",
			chromaFormat:   2,
			disableDeblock: 2,
			deblockMode:    2,
			pixFmt:         "yuv422p14le",
			frameSize:      1024,
			bitstreamMD5:   "8067e1392ae674c5265bee87c6147db6",
			rawVideoMD5:    "7198e8fa4fb907711a070364bf8f3fd8",
			frameMD5:       []string{"6505470aec88d1552f817a1be891a67c", "78a76a300a1dcb00e4892cd297911c29"},
		},
		{
			name:           "444-luma-only-no-deblock",
			chromaFormat:   3,
			disableDeblock: 1,
			deblockMode:    0,
			pixFmt:         "yuv444p14le",
			frameSize:      1536,
			bitstreamMD5:   "16f55eed1594d71ebb1474f2e34b5cbb",
			rawVideoMD5:    "a29d80bc20167a075ebd465c9f86f62b",
			frameMD5:       []string{"522820c0c78e0aee053dfe1aae6269b7", "0a24668a1b4da831aa34a24c8d286b0a"},
		},
		{
			name:           "444-luma-only-frame-deblock",
			chromaFormat:   3,
			disableDeblock: 0,
			deblockMode:    1,
			pixFmt:         "yuv444p14le",
			frameSize:      1536,
			bitstreamMD5:   "5ffb443feeb80cbb262e3ba71947704a",
			rawVideoMD5:    "a29d80bc20167a075ebd465c9f86f62b",
			frameMD5:       []string{"522820c0c78e0aee053dfe1aae6269b7", "0a24668a1b4da831aa34a24c8d286b0a"},
		},
		{
			name:           "444-luma-only-slice-boundary",
			chromaFormat:   3,
			disableDeblock: 2,
			deblockMode:    2,
			pixFmt:         "yuv444p14le",
			frameSize:      1536,
			bitstreamMD5:   "889861fa425689ad59f762ce63ec8ed9",
			rawVideoMD5:    "a29d80bc20167a075ebd465c9f86f62b",
			frameMD5:       []string{"522820c0c78e0aee053dfe1aae6269b7", "0a24668a1b4da831aa34a24c8d286b0a"},
		},
	}
}

func high14ChromaWeightedPFixture(t *testing.T, tt high14ChromaWeightedPCase) []byte {
	t.Helper()
	var data []byte
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALSPS), high14ChromaWeightedPSPSRBSP(tt.chromaFormat)))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALPPS), high14ChromaWeightedPPPSRBSP()))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALIDRSlice), high14ChromaWeightedPIDRSliceRBSP(tt.chromaFormat, tt.disableDeblock)))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALSlice), high14ChromaWeightedP16x16NoResidualSliceRBSP(tt.disableDeblock, tt.chromaWeighted)))
	sum := md5.Sum(data)
	if got := hex.EncodeToString(sum[:]); got != tt.bitstreamMD5 {
		t.Fatalf("%s bitstream md5 = %s, want %s", tt.name, got, tt.bitstreamMD5)
	}
	return data
}

func high14ChromaWeightedPIDRSliceRBSP(chromaFormat uint32, disableDeblockingFilterIDC uint32) []byte {
	payloadBits := highIntra16x16NoResidualPayloadBits
	if chromaFormat == 3 {
		payloadBits = "001001111"
	}
	return highIntra16x16ResidualDeblockSliceRBSP(payloadBits, disableDeblockingFilterIDC)
}

func high14ChromaWeightedPSPSRBSP(chromaFormat uint32) []byte {
	var b decoderSEIBitBuilder
	b.writeBits(244, 8)
	b.writeBits(0, 8)
	b.writeBits(10, 8)
	b.writeUE(0)
	b.writeUE(chromaFormat)
	if chromaFormat == 3 {
		b.writeBit(0)
	}
	b.writeUE(6)
	b.writeUE(6)
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

func high14ChromaWeightedPPPSRBSP() []byte {
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
	b.writeSE(-36)
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

func high14ChromaWeightedP16x16NoResidualSliceRBSP(disableDeblockingFilterIDC uint32, chromaWeighted bool) []byte {
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
		writeHigh14LumaOnlyWeightedPPredWeightSyntax(&b)
	}
	b.writeBit(0)
	b.writeSE(0)
	writeHighCAVLCDeblockSyntax(&b, disableDeblockingFilterIDC)
	highIntra16x16WritePayloadBits(&b, highInterP16x16NoResidualPayloadBits)
	return b.rbsp()
}

func writeHigh14LumaOnlyWeightedPPredWeightSyntax(b *decoderSEIBitBuilder) {
	b.writeUE(2)
	b.writeUE(1)
	b.writeBit(1)
	b.writeSE(3)
	b.writeSE(-2)
	b.writeBit(0)
}

func assertHigh14ChromaWeightedPFixtureSyntax(t *testing.T, data []byte, tt high14ChromaWeightedPCase) {
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
				sps.ChromaFormatIDC != tt.chromaFormat || sps.BitDepthLuma != 14 || sps.BitDepthChroma != 14 ||
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

func assertHigh14ChromaWeightedPFrames(t *testing.T, frames []*Frame, tt high14ChromaWeightedPCase) {
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
			frame.BitDepthLuma != 14 || frame.BitDepthChroma != 14 {
			t.Fatalf("frame[%d] metadata = %dx%d chroma %d depth %d/%d, want 16x16 chroma %d High14",
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

func assertFFmpegHigh14ChromaWeightedPRawVideoOracle(t *testing.T, data []byte, tt high14ChromaWeightedPCase) {
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
