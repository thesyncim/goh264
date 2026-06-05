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

type high12CABACBCase struct {
	name         string
	sourceFile   string
	idrDeblock   int32
	deblockMode  int32
	direct       int32
	direct8x8    int32
	checkDirect8 bool
	mode2Deblock bool
	width        int
	height       int
	rawFrameSize int
	bitstreamMD5 string
	frameMD5     []string
	rawVideoMD5  string
}

func TestHigh12CABACBFixtureSyntax(t *testing.T) {
	for _, tt := range high12CABACBCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := high12CABACBFixture(t, tt)
			assertHigh12CABACBFixtureSyntax(t, data, tt)
		})
	}
}

func TestDecodeAnnexBHigh12CABACBFrames(t *testing.T) {
	for _, tt := range high12CABACBCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := high12CABACBFixture(t, tt)
			assertHigh12CABACBFixtureSyntax(t, data, tt)

			frames, err := NewDecoder().DecodeAnnexBFrames(data)
			if err != nil {
				t.Fatalf("DecodeAnnexBFrames: %v", err)
			}
			assertHigh12CABACBFrames(t, frames, tt)
		})
	}
}

func TestDecodeAVCHigh12CABACBFrames(t *testing.T) {
	for _, tt := range high12CABACBCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := high12CABACBFixture(t, tt)
			assertHigh12CABACBFixtureSyntax(t, data, tt)

			for _, nalLengthSize := range []int{2, 3, 4} {
				frames, err := NewDecoder().DecodeAVCFrames(annexBToAVC(t, data, nalLengthSize), nalLengthSize)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: DecodeAVCFrames: %v", nalLengthSize, err)
				}
				assertHigh12CABACBFrames(t, frames, tt)
			}
		})
	}
}

func TestDecodeConfiguredAVCHigh12CABACBFramesAcrossSamplesFlush(t *testing.T) {
	for _, tt := range high12CABACBCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := high12CABACBFixture(t, tt)
			assertHigh12CABACBFixtureSyntax(t, data, tt)

			for _, nalLengthSize := range []int{2, 3, 4} {
				config, samples := annexBToAVCConfigAndSamples(t, data, nalLengthSize)
				if len(samples) != len(tt.frameMD5) {
					t.Fatalf("nalLengthSize=%d: samples = %d, want %d", nalLengthSize, len(samples), len(tt.frameMD5))
				}

				dec := NewDecoder()
				if _, err := dec.ParseAVCDecoderConfigurationRecord(config); err != nil {
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
				assertHigh12CABACBConfiguredSampleCounts(t, tt.name, nalLengthSize, frameCounts)
				frames = append(frames, out...)
				assertHigh12CABACBFrames(t, frames, tt)
			}
		})
	}
}

func TestFFmpegRawVideoFrameMD5OracleHigh12CABACB(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}

	for _, tt := range high12CABACBCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := high12CABACBFixture(t, tt)
			assertHigh12CABACBFixtureSyntax(t, data, tt)
			assertFFmpegHigh12CABACBRawVideoOracle(t, data, tt)
		})
	}
}

func high12CABACBCases() []high12CABACBCase {
	return []high12CABACBCase{
		{
			name:         "nondirect-no-deblock",
			sourceFile:   "high10_nondirect_b_cabac.h264",
			width:        16,
			height:       16,
			rawFrameSize: 768,
			bitstreamMD5: "03b734cd1fbef4272835e2a203f2c42c",
			frameMD5: []string{
				"c207163647e7a87cd41197f503d9aede",
				"2dc3f978413b9eefac88acb7bd30647c",
				"8697f5168b16c170e85580aeebce7c67",
			},
			rawVideoMD5: "08c5c19d3b6022910898bdbc22b9be71",
		},
		{
			name:         "nondirect-mode1-deblock",
			sourceFile:   "high10_b_deblock_cabac.h264",
			deblockMode:  1,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			bitstreamMD5: "e1decc6ca42afa7ea3944cd88cb1fb8e",
			frameMD5: []string{
				"c207163647e7a87cd41197f503d9aede",
				"92df8c8e6faca62e23650977978c7c28",
				"fadaa6dd57e157ac14f56c52ddaf0c87",
			},
			rawVideoMD5: "02d1f4b20d4023077dcccfaf12a6efd1",
		},
		{
			name:         "nondirect-mode2-deblock",
			sourceFile:   "high10_b_deblock_cabac.h264",
			deblockMode:  2,
			mode2Deblock: true,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			bitstreamMD5: "9bffc86baa5ac13bea4e917213e442ea",
			frameMD5: []string{
				"c207163647e7a87cd41197f503d9aede",
				"92df8c8e6faca62e23650977978c7c28",
				"fadaa6dd57e157ac14f56c52ddaf0c87",
			},
			rawVideoMD5: "02d1f4b20d4023077dcccfaf12a6efd1",
		},
		{
			name:         "temporal-direct-mode1-deblock",
			sourceFile:   "high10_direct_b_deblock_temporal_cabac.h264",
			idrDeblock:   1,
			deblockMode:  1,
			width:        32,
			height:       16,
			rawFrameSize: 1536,
			bitstreamMD5: "919bc4f3fbdc7b24fda77e49cbe51468",
			frameMD5: []string{
				"9ff9de409c69b282d462098d0f40c362",
				"e75d2316e3dff3d98c57cd60840937d1",
				"b04f667682af8213cb3771ace7bae593",
			},
			rawVideoMD5: "e54e8d3555e1d31a007e5dae98eb693e",
		},
		{
			name:         "temporal-direct-mode2-deblock",
			sourceFile:   "high10_direct_b_deblock_temporal_cabac.h264",
			idrDeblock:   1,
			deblockMode:  2,
			mode2Deblock: true,
			width:        32,
			height:       16,
			rawFrameSize: 1536,
			bitstreamMD5: "11d0c04231016780c62be4887e602c23",
			frameMD5: []string{
				"9ff9de409c69b282d462098d0f40c362",
				"e75d2316e3dff3d98c57cd60840937d1",
				"b04f667682af8213cb3771ace7bae593",
			},
			rawVideoMD5: "e54e8d3555e1d31a007e5dae98eb693e",
		},
		{
			name:         "spatial-direct-mode1-deblock",
			sourceFile:   "high10_direct_b_deblock_spatial_cabac.h264",
			idrDeblock:   1,
			deblockMode:  1,
			direct:       1,
			width:        32,
			height:       16,
			rawFrameSize: 1536,
			bitstreamMD5: "78848c2f46ad15b432c8465d17fae41f",
			frameMD5: []string{
				"9ff9de409c69b282d462098d0f40c362",
				"e75d2316e3dff3d98c57cd60840937d1",
				"b04f667682af8213cb3771ace7bae593",
			},
			rawVideoMD5: "e54e8d3555e1d31a007e5dae98eb693e",
		},
		{
			name:         "spatial-direct-mode2-deblock",
			sourceFile:   "high10_direct_b_deblock_spatial_cabac.h264",
			idrDeblock:   1,
			deblockMode:  2,
			direct:       1,
			mode2Deblock: true,
			width:        32,
			height:       16,
			rawFrameSize: 1536,
			bitstreamMD5: "87cb66dad81bb86641b521ac03dd860d",
			frameMD5: []string{
				"9ff9de409c69b282d462098d0f40c362",
				"e75d2316e3dff3d98c57cd60840937d1",
				"b04f667682af8213cb3771ace7bae593",
			},
			rawVideoMD5: "e54e8d3555e1d31a007e5dae98eb693e",
		},
		{
			name:         "temporal-bskip-mode1-deblock",
			sourceFile:   "high10_bskip_deblock_temporal_cabac.h264",
			deblockMode:  1,
			direct8x8:    1,
			checkDirect8: true,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			bitstreamMD5: "723ff00d7773b76664fe9e37d9009b4a",
			frameMD5: []string{
				"d4753b9733af2865470fb72f96a37071",
				"d4753b9733af2865470fb72f96a37071",
				"d4753b9733af2865470fb72f96a37071",
			},
			rawVideoMD5: "c5844f8a45006553335c482758ad0f49",
		},
		{
			name:         "temporal-bskip-mode2-deblock",
			sourceFile:   "high10_bskip_deblock_temporal_cabac.h264",
			deblockMode:  2,
			direct8x8:    1,
			checkDirect8: true,
			mode2Deblock: true,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			bitstreamMD5: "33c1519b2c5e721a9300a273ccbeee9c",
			frameMD5: []string{
				"d4753b9733af2865470fb72f96a37071",
				"d4753b9733af2865470fb72f96a37071",
				"d4753b9733af2865470fb72f96a37071",
			},
			rawVideoMD5: "c5844f8a45006553335c482758ad0f49",
		},
		{
			name:         "spatial-bskip-mode1-deblock",
			sourceFile:   "high10_bskip_deblock_spatial_cabac.h264",
			deblockMode:  1,
			direct:       1,
			direct8x8:    1,
			checkDirect8: true,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			bitstreamMD5: "ac122445571679702207fce594a1972b",
			frameMD5: []string{
				"d4753b9733af2865470fb72f96a37071",
				"d4753b9733af2865470fb72f96a37071",
				"d4753b9733af2865470fb72f96a37071",
			},
			rawVideoMD5: "c5844f8a45006553335c482758ad0f49",
		},
		{
			name:         "spatial-bskip-mode2-deblock",
			sourceFile:   "high10_bskip_deblock_spatial_cabac.h264",
			deblockMode:  2,
			direct:       1,
			direct8x8:    1,
			checkDirect8: true,
			mode2Deblock: true,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			bitstreamMD5: "eb62cea5f0c30f2281c0f513dc156e4c",
			frameMD5: []string{
				"d4753b9733af2865470fb72f96a37071",
				"d4753b9733af2865470fb72f96a37071",
				"d4753b9733af2865470fb72f96a37071",
			},
			rawVideoMD5: "c5844f8a45006553335c482758ad0f49",
		},
		{
			name:         "direct-sub-b8x8-temporal-mode1-deblock",
			sourceFile:   "high10_cabac_b8x8_temporal_direct_sub_deblock.h264",
			deblockMode:  1,
			direct8x8:    1,
			checkDirect8: true,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			bitstreamMD5: "5b56d76e763e6fbf78d2b4143bcfee43",
			frameMD5: []string{
				"d4753b9733af2865470fb72f96a37071",
				"d4753b9733af2865470fb72f96a37071",
				"d4753b9733af2865470fb72f96a37071",
			},
			rawVideoMD5: "c5844f8a45006553335c482758ad0f49",
		},
		{
			name:         "direct-sub-b8x8-temporal-mode2-deblock",
			sourceFile:   "high10_cabac_b8x8_temporal_direct_sub_deblock.h264",
			deblockMode:  2,
			direct8x8:    1,
			checkDirect8: true,
			mode2Deblock: true,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			bitstreamMD5: "9660b74d1b2dfcfc1d8dc77338d522ac",
			frameMD5: []string{
				"d4753b9733af2865470fb72f96a37071",
				"d4753b9733af2865470fb72f96a37071",
				"d4753b9733af2865470fb72f96a37071",
			},
			rawVideoMD5: "c5844f8a45006553335c482758ad0f49",
		},
		{
			name:         "direct-sub-b8x8-spatial-mode1-deblock",
			sourceFile:   "high10_cabac_b8x8_spatial_direct_sub_deblock.h264",
			deblockMode:  1,
			direct:       1,
			direct8x8:    1,
			checkDirect8: true,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			bitstreamMD5: "fa7d9249f21628d1c69af228fa79c50c",
			frameMD5: []string{
				"d4753b9733af2865470fb72f96a37071",
				"d4753b9733af2865470fb72f96a37071",
				"d4753b9733af2865470fb72f96a37071",
			},
			rawVideoMD5: "c5844f8a45006553335c482758ad0f49",
		},
		{
			name:         "direct-sub-b8x8-spatial-mode2-deblock",
			sourceFile:   "high10_cabac_b8x8_spatial_direct_sub_deblock.h264",
			deblockMode:  2,
			direct:       1,
			direct8x8:    1,
			checkDirect8: true,
			mode2Deblock: true,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			bitstreamMD5: "571a49311ae8a5e15757f07f6f826f7a",
			frameMD5: []string{
				"d4753b9733af2865470fb72f96a37071",
				"d4753b9733af2865470fb72f96a37071",
				"d4753b9733af2865470fb72f96a37071",
			},
			rawVideoMD5: "c5844f8a45006553335c482758ad0f49",
		},
		{
			name:         "direct-sub-b4x4-temporal-mode1-deblock",
			sourceFile:   "high10_cabac_b4x4_temporal_direct_sub_deblock.h264",
			deblockMode:  1,
			checkDirect8: true,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			bitstreamMD5: "86e59d6188806fc1ef62a4da87f84fad",
			frameMD5: []string{
				"d4753b9733af2865470fb72f96a37071",
				"d4753b9733af2865470fb72f96a37071",
				"d4753b9733af2865470fb72f96a37071",
			},
			rawVideoMD5: "c5844f8a45006553335c482758ad0f49",
		},
		{
			name:         "direct-sub-b4x4-temporal-mode2-deblock",
			sourceFile:   "high10_cabac_b4x4_temporal_direct_sub_deblock.h264",
			deblockMode:  2,
			checkDirect8: true,
			mode2Deblock: true,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			bitstreamMD5: "4fa5696a0e384680c00388c5784b2c95",
			frameMD5: []string{
				"d4753b9733af2865470fb72f96a37071",
				"d4753b9733af2865470fb72f96a37071",
				"d4753b9733af2865470fb72f96a37071",
			},
			rawVideoMD5: "c5844f8a45006553335c482758ad0f49",
		},
		{
			name:         "direct-sub-b4x4-spatial-mode1-deblock",
			sourceFile:   "high10_cabac_b4x4_spatial_direct_sub_deblock.h264",
			deblockMode:  1,
			direct:       1,
			checkDirect8: true,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			bitstreamMD5: "0aa8dee699e527573752ef70f4c19908",
			frameMD5: []string{
				"d4753b9733af2865470fb72f96a37071",
				"d4753b9733af2865470fb72f96a37071",
				"d4753b9733af2865470fb72f96a37071",
			},
			rawVideoMD5: "c5844f8a45006553335c482758ad0f49",
		},
		{
			name:         "direct-sub-b4x4-spatial-mode2-deblock",
			sourceFile:   "high10_cabac_b4x4_spatial_direct_sub_deblock.h264",
			deblockMode:  2,
			direct:       1,
			checkDirect8: true,
			mode2Deblock: true,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			bitstreamMD5: "d204027e709a5c7dcc9b2c08304b0a00",
			frameMD5: []string{
				"d4753b9733af2865470fb72f96a37071",
				"d4753b9733af2865470fb72f96a37071",
				"d4753b9733af2865470fb72f96a37071",
			},
			rawVideoMD5: "c5844f8a45006553335c482758ad0f49",
		},
		{
			name:         "partitioned-b8x8-no-deblock",
			sourceFile:   "high10_partitioned_b8x8_cabac.h264",
			width:        16,
			height:       16,
			rawFrameSize: 768,
			bitstreamMD5: "0294381bc155f900af4c33da1d99879f",
			frameMD5: []string{
				"6a406136f799b79e4526c1397e8a9110",
				"ed690926884389d5dfdca65c48c8da4c",
				"f101bc562bd77dd6664d0658f1ca69f7",
			},
			rawVideoMD5: "60c3ca2cd4f90575d3337a86fb6af706",
		},
	}
}

func high12CABACBFixture(t *testing.T, tt high12CABACBCase) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "h264", tt.sourceFile))
	if err != nil {
		t.Fatalf("read %s: %v", tt.sourceFile, err)
	}
	out := highCABACBRewriteAnnexB(t, data, 12, tt.mode2Deblock)
	sum := md5.Sum(out)
	got := hex.EncodeToString(sum[:])
	if got != tt.bitstreamMD5 {
		t.Fatalf("High12 CABAC B generated bitstream md5 = %s, want %s", got, tt.bitstreamMD5)
	}
	return out
}

func assertHigh12CABACBFixtureSyntax(t *testing.T, data []byte, tt high12CABACBCase) {
	t.Helper()
	nals, err := h264.SplitAnnexB(data)
	if err != nil {
		t.Fatal(err)
	}
	var spsList [32]*h264.SPS
	var ppsList [256]*h264.PPS
	var gotSlices []int32
	for i, nal := range nals {
		switch nal.Type {
		case h264.NALSPS:
			sps, err := h264.DecodeSPS(nal.RBSP)
			if err != nil {
				t.Fatal(err)
			}
			if i != 0 || sps.ProfileIDC != 244 || sps.Width != int32(tt.width) || sps.Height != int32(tt.height) ||
				sps.ChromaFormatIDC != 1 || sps.BitDepthLuma != 12 || sps.BitDepthChroma != 12 ||
				sps.FrameMBSOnlyFlag != 1 || sps.MBAFF != 0 || sps.RefFrameCount != 2 {
				t.Fatalf("SPS = nal[%d] profile %d %dx%d chroma %d depth %d/%d frameonly/mbaff %d/%d refs %d, want High12 4:2:0 frame-only refs=2",
					i, sps.ProfileIDC, sps.Width, sps.Height, sps.ChromaFormatIDC, sps.BitDepthLuma, sps.BitDepthChroma,
					sps.FrameMBSOnlyFlag, sps.MBAFF, sps.RefFrameCount)
			}
			if tt.checkDirect8 && int32(sps.Direct8x8InferenceFlag) != tt.direct8x8 {
				t.Fatalf("SPS direct_8x8_inference_flag = %d, want %d", sps.Direct8x8InferenceFlag, tt.direct8x8)
			}
			spsList[sps.SPSID] = sps
		case h264.NALPPS:
			pps, err := h264.DecodePPS(nal.RBSP, &spsList)
			if err != nil {
				t.Fatal(err)
			}
			if i != 1 || pps.CABAC != 1 || pps.Transform8x8Mode != 0 || pps.WeightedPred != 0 ||
				pps.WeightedBipredIDC != 0 || pps.RefCount != [2]uint32{2, 1} {
				t.Fatalf("PPS = nal[%d] cabac/8x8/weights/refs = %d/%d/%d/%d/%v, want CABAC/no-8x8/unweighted refs=2/1",
					i, pps.CABAC, pps.Transform8x8Mode, pps.WeightedPred, pps.WeightedBipredIDC, pps.RefCount)
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
			switch sh.SliceTypeNoS {
			case h264.PictureTypeI:
				if sh.DeblockingFilter != tt.idrDeblock {
					t.Fatalf("I slice deblock = %d, want mode-%d", sh.DeblockingFilter, tt.idrDeblock)
				}
				if sh.ListCount != 0 || sh.RefCount != ([2]uint32{}) {
					t.Fatalf("I slice lists/refs = %d/%v, want none", sh.ListCount, sh.RefCount)
				}
			case h264.PictureTypeP:
				if sh.DeblockingFilter != tt.deblockMode {
					t.Fatalf("P slice deblock = %d, want mode-%d", sh.DeblockingFilter, tt.deblockMode)
				}
				if sh.ListCount != 1 || sh.RefCount[0] != 1 ||
					sh.PredWeightTable.UseWeight != 0 || sh.PredWeightTable.UseWeightChroma != 0 {
					t.Fatalf("P slice lists/refs/weights = %d/%v/%d/%d, want L0 refs=1 unweighted",
						sh.ListCount, sh.RefCount, sh.PredWeightTable.UseWeight, sh.PredWeightTable.UseWeightChroma)
				}
			case h264.PictureTypeB:
				if sh.DeblockingFilter != tt.deblockMode {
					t.Fatalf("B slice deblock = %d, want mode-%d", sh.DeblockingFilter, tt.deblockMode)
				}
				if sh.ListCount != 2 || sh.RefCount[0] != 1 || sh.RefCount[1] != 1 ||
					sh.DirectSpatialMVPred != tt.direct ||
					sh.PredWeightTable.UseWeight != 0 || sh.PredWeightTable.UseWeightChroma != 0 {
					t.Fatalf("B slice lists/refs/direct/weights = %d/%v/%d/%d/%d, want L0/L1 refs=1/1 direct=%d unweighted",
						sh.ListCount, sh.RefCount, sh.DirectSpatialMVPred, sh.PredWeightTable.UseWeight, sh.PredWeightTable.UseWeightChroma, tt.direct)
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
	if len(gotSlices) != 3 || gotSlices[0] != h264.PictureTypeI || gotSlices[1] != h264.PictureTypeP || gotSlices[2] != h264.PictureTypeB {
		t.Fatalf("slice types = %v, want I/P/B", gotSlices)
	}
}

func assertHigh12CABACBConfiguredSampleCounts(t *testing.T, name string, nalLengthSize int, got []int) {
	t.Helper()
	want := []int{0, 1, 1, 1}
	if len(got) != len(want) {
		t.Fatalf("%s nalLengthSize=%d configured sample/flush counts = %v, want %v", name, nalLengthSize, got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%s nalLengthSize=%d configured sample/flush counts = %v, want %v", name, nalLengthSize, got, want)
		}
	}
}

func assertHigh12CABACBFrames(t *testing.T, frames []*Frame, tt high12CABACBCase) {
	t.Helper()
	if len(frames) != len(tt.frameMD5) {
		t.Fatalf("frames = %d, want %d", len(frames), len(tt.frameMD5))
	}
	var rawVideo []byte
	for i, frame := range frames {
		if frame == nil {
			t.Fatalf("frame[%d] is nil", i)
		}
		if frame.Width != tt.width || frame.Height != tt.height ||
			frame.ChromaFormatIDC != 1 || frame.BitDepthLuma != 12 || frame.BitDepthChroma != 12 {
			t.Fatalf("frame[%d] format = %dx%d chroma %d depth %d/%d, want %dx%d yuv420p12le",
				i, frame.Width, frame.Height, frame.ChromaFormatIDC, frame.BitDepthLuma, frame.BitDepthChroma, tt.width, tt.height)
		}
		if pixFmt, err := frame.RawPixelFormat(); err != nil || pixFmt != "yuv420p12le" {
			t.Fatalf("frame[%d] RawPixelFormat = %q/%v, want yuv420p12le/nil", i, pixFmt, err)
		}
		if size, err := frame.RawYUVSize(); err != nil || size != tt.rawFrameSize {
			t.Fatalf("frame[%d] RawYUVSize = %d/%v, want %d/nil", i, size, err, tt.rawFrameSize)
		}
		raw, err := frame.AppendRawYUVBytesLE(nil)
		if err != nil {
			t.Fatalf("frame[%d] AppendRawYUVBytesLE: %v", i, err)
		}
		rawVideo = append(rawVideo, raw...)
		sum := md5.Sum(raw)
		got := hex.EncodeToString(sum[:])
		if got != tt.frameMD5[i] {
			t.Fatalf("frame[%d] raw md5 = %s, want %s", i, got, tt.frameMD5[i])
		}
		if _, err := frame.AppendRawYUV(nil); err != ErrUnsupported {
			t.Fatalf("frame[%d] AppendRawYUV high12 error = %v, want ErrUnsupported", i, err)
		}
	}
	sum := md5.Sum(rawVideo)
	got := hex.EncodeToString(sum[:])
	if got != tt.rawVideoMD5 {
		t.Fatalf("rawvideo md5 = %s, want %s", got, tt.rawVideoMD5)
	}
}

func assertFFmpegHigh12CABACBRawVideoOracle(t *testing.T, data []byte, tt high12CABACBCase) {
	t.Helper()
	path := writeTempH264(t, data)
	framemd5 := exec.Command("ffmpeg",
		"-v", "error",
		"-xerror",
		"-f", "h264",
		"-i", path,
		"-an", "-sn", "-dn",
		"-pix_fmt", "yuv420p12le",
		"-f", "framemd5",
		"-",
	)
	framemd5Out, err := framemd5.Output()
	if err != nil {
		t.Fatalf("ffmpeg framemd5: %v", err)
	}
	for i, want := range tt.frameMD5 {
		line := []byte(fmt.Sprintf("0, %10d, %10d,        1, %8d, %s", i, i, tt.rawFrameSize, want))
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
		"-pix_fmt", "yuv420p12le",
		"-f", "rawvideo",
		"-",
	)
	raw, err := rawvideo.Output()
	if err != nil {
		t.Fatalf("ffmpeg rawvideo: %v", err)
	}
	if len(raw) != len(tt.frameMD5)*tt.rawFrameSize {
		t.Fatalf("rawvideo size = %d, want %d", len(raw), len(tt.frameMD5)*tt.rawFrameSize)
	}
	sum := md5.Sum(raw)
	got := hex.EncodeToString(sum[:])
	if got != tt.rawVideoMD5 {
		t.Fatalf("rawvideo md5 = %s, want %s", got, tt.rawVideoMD5)
	}
}
