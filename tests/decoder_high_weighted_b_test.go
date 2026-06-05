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
	"strings"
	"testing"

	"github.com/thesyncim/goh264/internal/h264"
)

type highWeightedBCase struct {
	bitDepth     int
	name         string
	sourceFile   string
	cabac        int32
	mode2Deblock bool
	width        int
	height       int
	rawFrameSize int
	wantSlices   []int32
	deblock      []int32
	bitstreamMD5 string
	frameMD5     []string
	rawVideoMD5  string
}

func TestHigh1214ImplicitWeightedBFixtureSyntax(t *testing.T) {
	for _, tt := range high1214ImplicitWeightedBCases() {
		t.Run(highWeightedBCaseName(tt), func(t *testing.T) {
			data := highWeightedBFixture(t, tt)
			assertHighWeightedBFixtureSyntax(t, data, tt)
		})
	}
}

func TestDecodeAnnexBHigh1214ImplicitWeightedBFrames(t *testing.T) {
	for _, tt := range high1214ImplicitWeightedBCases() {
		t.Run(highWeightedBCaseName(tt), func(t *testing.T) {
			data := highWeightedBFixture(t, tt)
			assertHighWeightedBFixtureSyntax(t, data, tt)
			frames, err := NewDecoder().DecodeAnnexBFrames(data)
			if err != nil {
				t.Fatalf("DecodeAnnexBFrames: %v", err)
			}
			assertHighWeightedBFrames(t, frames, tt)
		})
	}
}

func TestDecodeAVCHigh1214ImplicitWeightedBFrames(t *testing.T) {
	for _, tt := range high1214ImplicitWeightedBCases() {
		t.Run(highWeightedBCaseName(tt), func(t *testing.T) {
			data := highWeightedBFixture(t, tt)
			assertHighWeightedBFixtureSyntax(t, data, tt)
			for _, nalLengthSize := range []int{2, 3, 4} {
				frames, err := NewDecoder().DecodeAVCFrames(annexBToAVC(t, data, nalLengthSize), nalLengthSize)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: DecodeAVCFrames: %v", nalLengthSize, err)
				}
				assertHighWeightedBFrames(t, frames, tt)
			}
		})
	}
}

func TestDecodeConfiguredAVCHigh1214ImplicitWeightedBFramesAcrossSamplesFlush(t *testing.T) {
	for _, tt := range high1214ImplicitWeightedBCases() {
		t.Run(highWeightedBCaseName(tt), func(t *testing.T) {
			data := highWeightedBFixture(t, tt)
			assertHighWeightedBFixtureSyntax(t, data, tt)
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
				assertHighWeightedBFrames(t, frames, tt)
			}
		})
	}
}

func TestFFmpegRawVideoFrameMD5OracleHigh1214ImplicitWeightedB(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}
	for _, tt := range high1214ImplicitWeightedBCases() {
		t.Run(highWeightedBCaseName(tt), func(t *testing.T) {
			data := highWeightedBFixture(t, tt)
			assertHighWeightedBFixtureSyntax(t, data, tt)
			assertFFmpegHighWeightedBRawVideoOracle(t, data, tt)
		})
	}
}

func highWeightedBCaseName(tt highWeightedBCase) string {
	return fmt.Sprintf("high%d-%s", tt.bitDepth, tt.name)
}

func highWeightedBFixture(t *testing.T, tt highWeightedBCase) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "h264", tt.sourceFile))
	if err != nil {
		t.Fatalf("read %s: %v", tt.sourceFile, err)
	}
	var out []byte
	if tt.cabac != 0 {
		out = highCABACBRewriteAnnexB(t, data, tt.bitDepth, tt.mode2Deblock)
	} else {
		out = highCAVLCBRewriteAnnexB(t, data, tt.bitDepth, tt.mode2Deblock)
	}
	sum := md5.Sum(out)
	if got := hex.EncodeToString(sum[:]); got != tt.bitstreamMD5 {
		t.Fatalf("High%d implicit weighted B generated bitstream md5 = %s, want %s", tt.bitDepth, got, tt.bitstreamMD5)
	}
	return out
}

func highWeightedBPPSRefCount(tt highWeightedBCase) [2]uint32 {
	if strings.Contains(tt.name, "direct-sub") {
		return [2]uint32{2, 1}
	}
	return [2]uint32{1, 1}
}

func assertHighWeightedBFixtureSyntax(t *testing.T, data []byte, tt highWeightedBCase) {
	t.Helper()
	nals, err := h264.SplitAnnexB(data)
	if err != nil {
		t.Fatal(err)
	}
	var spsList [32]*h264.SPS
	var ppsList [256]*h264.PPS
	var gotSlices []int32
	var gotDeblock []int32
	var gotB int
	for i, nal := range nals {
		switch nal.Type {
		case h264.NALSPS:
			sps, err := h264.DecodeSPS(nal.RBSP)
			if err != nil {
				t.Fatal(err)
			}
			if sps.ProfileIDC != 244 || sps.Width != int32(tt.width) || sps.Height != int32(tt.height) ||
				sps.ChromaFormatIDC != 1 || sps.BitDepthLuma != int32(tt.bitDepth) || sps.BitDepthChroma != int32(tt.bitDepth) ||
				sps.FrameMBSOnlyFlag != 1 || sps.MBAFF != 0 {
				t.Fatalf("SPS = nal[%d] profile %d %dx%d chroma %d depth %d/%d frameonly/mbaff %d/%d, want High%d 4:2:0 frame-only",
					i, sps.ProfileIDC, sps.Width, sps.Height, sps.ChromaFormatIDC, sps.BitDepthLuma, sps.BitDepthChroma,
					sps.FrameMBSOnlyFlag, sps.MBAFF, tt.bitDepth)
			}
			spsList[sps.SPSID] = sps
		case h264.NALPPS:
			pps, err := h264.DecodePPS(nal.RBSP, &spsList)
			if err != nil {
				t.Fatal(err)
			}
			wantRefs := highWeightedBPPSRefCount(tt)
			if pps.CABAC != tt.cabac || pps.Transform8x8Mode != 0 || pps.WeightedPred != 0 ||
				pps.WeightedBipredIDC != 2 || pps.RefCount != wantRefs {
				t.Fatalf("PPS = nal[%d] cabac/8x8/weights/refs = %d/%d/%d/%d/%v, want cabac=%d no-8x8 implicit-B refs=%v",
					i, pps.CABAC, pps.Transform8x8Mode, pps.WeightedPred, pps.WeightedBipredIDC, pps.RefCount, tt.cabac, wantRefs)
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
				if sh.ListCount != 0 {
					t.Fatalf("I slice lists = %d, want none", sh.ListCount)
				}
			case h264.PictureTypeP:
				if sh.ListCount != 1 || sh.RefCount[0] != 1 || sh.PredWeightTable.UseWeight != 0 || sh.PredWeightTable.UseWeightChroma != 0 {
					t.Fatalf("P slice lists/refs/weights = %d/%v/%d/%d, want L0 ref=1 unweighted",
						sh.ListCount, sh.RefCount, sh.PredWeightTable.UseWeight, sh.PredWeightTable.UseWeightChroma)
				}
			case h264.PictureTypeB:
				gotB++
				if sh.ListCount != 2 || sh.RefCount[0] != 1 || sh.RefCount[1] != 1 ||
					sh.PredWeightTable.UseWeight != 0 || sh.PredWeightTable.UseWeightChroma != 0 {
					t.Fatalf("B slice lists/refs/serialized weights = %d/%v/%d/%d, want L0/L1 refs=1/1 temporal implicit weights only",
						sh.ListCount, sh.RefCount, sh.PredWeightTable.UseWeight, sh.PredWeightTable.UseWeightChroma)
				}
			default:
				t.Fatalf("unexpected slice type %d", sh.SliceTypeNoS)
			}
			gotSlices = append(gotSlices, sh.SliceTypeNoS)
			gotDeblock = append(gotDeblock, sh.DeblockingFilter)
		case h264.NALSEI:
		default:
			t.Fatalf("unexpected NAL type %d", nal.Type)
		}
	}
	assertHighWeightedBInt32Slice(t, "slice types", gotSlices, tt.wantSlices)
	assertHighWeightedBInt32Slice(t, "deblock modes", gotDeblock, tt.deblock)
	if gotB == 0 {
		t.Fatalf("B slices = 0, want at least one")
	}
}

func assertHighWeightedBFrames(t *testing.T, frames []*Frame, tt highWeightedBCase) {
	t.Helper()
	if len(frames) != len(tt.frameMD5) {
		t.Fatalf("frames = %d, want %d", len(frames), len(tt.frameMD5))
	}
	var rawVideo []byte
	wantPixFmt := fmt.Sprintf("yuv420p%dle", tt.bitDepth)
	for i, frame := range frames {
		if frame == nil {
			t.Fatalf("frame[%d] is nil", i)
		}
		if frame.Width != tt.width || frame.Height != tt.height || frame.ChromaFormatIDC != 1 ||
			frame.BitDepthLuma != tt.bitDepth || frame.BitDepthChroma != tt.bitDepth {
			t.Fatalf("frame[%d] format = %dx%d chroma %d depth %d/%d, want %dx%d %s",
				i, frame.Width, frame.Height, frame.ChromaFormatIDC, frame.BitDepthLuma, frame.BitDepthChroma,
				tt.width, tt.height, wantPixFmt)
		}
		if pixFmt, err := frame.RawPixelFormat(); err != nil || pixFmt != wantPixFmt {
			t.Fatalf("frame[%d] RawPixelFormat = %q/%v, want %s/nil", i, pixFmt, err, wantPixFmt)
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

func assertFFmpegHighWeightedBRawVideoOracle(t *testing.T, data []byte, tt highWeightedBCase) {
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
		"-pix_fmt", pixFmt,
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
	if got := hex.EncodeToString(sum[:]); got != tt.rawVideoMD5 {
		t.Fatalf("rawvideo md5 = %s, want %s", got, tt.rawVideoMD5)
	}
}

func assertHighWeightedBInt32Slice(t *testing.T, name string, got []int32, want []int32) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s = %v, want %v", name, got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%s = %v, want %v", name, got, want)
		}
	}
}

func high1214ImplicitWeightedBCases() []highWeightedBCase {
	return []highWeightedBCase{
		{
			bitDepth:     12,
			name:         "cavlc-no-deblock",
			sourceFile:   "high10_implicit_weight_b_cavlc.h264",
			cabac:        0,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			wantSlices:   []int32{1, 2, 3, 3, 2},
			deblock:      []int32{0, 0, 0, 0, 0},
			bitstreamMD5: "8653dd76a1d017aead1ed464521b5bbc",
			frameMD5:     []string{"d4753b9733af2865470fb72f96a37071", "e4b3a2575cb05c064ac8ffeae18d630f", "c420ccced6bc909f45c382742506f15b", "e7ac9c6c5627f3112acde8625c52f8cf", "3e64a4ea01c4befde1217bf6d295a034"},
			rawVideoMD5:  "3471fa008f324784d435e7db6818c723",
		},
		{
			bitDepth:     12,
			name:         "cabac-no-deblock",
			sourceFile:   "high10_implicit_weight_b_cabac.h264",
			cabac:        1,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			wantSlices:   []int32{1, 2, 3, 3, 2},
			deblock:      []int32{0, 0, 0, 0, 0},
			bitstreamMD5: "d2d32c1573dea22ce97419dd3bb69e4b",
			frameMD5:     []string{"d4753b9733af2865470fb72f96a37071", "e4b3a2575cb05c064ac8ffeae18d630f", "c420ccced6bc909f45c382742506f15b", "e7ac9c6c5627f3112acde8625c52f8cf", "3e64a4ea01c4befde1217bf6d295a034"},
			rawVideoMD5:  "3471fa008f324784d435e7db6818c723",
		},
		{
			bitDepth:     12,
			name:         "cavlc-mode1-deblock",
			sourceFile:   "high10_implicit_weight_b_deblock_cavlc.h264",
			cabac:        0,
			width:        32,
			height:       16,
			rawFrameSize: 1536,
			wantSlices:   []int32{1, 2, 3, 3, 2},
			deblock:      []int32{0, 1, 1, 1, 1},
			bitstreamMD5: "22911c77c400ef473078380a2bbd9d63",
			frameMD5:     []string{"91750ecee8bda0d5c023f6254707914b", "19bbc990cb9ba8783c56917c1ec3eb41", "bb569e904b7d1122f695654f9a4dec6f", "9f2ec56947f8b92de7fcd2a34ac04ab8", "827d6e4eaf327ddf44be7314cc7902a3"},
			rawVideoMD5:  "72c00493bffdcc63292560b2d1fa829f",
		},
		{
			bitDepth:     12,
			name:         "cavlc-mode2-deblock",
			sourceFile:   "high10_implicit_weight_b_deblock_cavlc.h264",
			cabac:        0,
			mode2Deblock: true,
			width:        32,
			height:       16,
			rawFrameSize: 1536,
			wantSlices:   []int32{1, 2, 3, 3, 2},
			deblock:      []int32{0, 2, 2, 2, 2},
			bitstreamMD5: "37ad10c1bd953b13eb43c402b66ec839",
			frameMD5:     []string{"91750ecee8bda0d5c023f6254707914b", "19bbc990cb9ba8783c56917c1ec3eb41", "bb569e904b7d1122f695654f9a4dec6f", "9f2ec56947f8b92de7fcd2a34ac04ab8", "827d6e4eaf327ddf44be7314cc7902a3"},
			rawVideoMD5:  "72c00493bffdcc63292560b2d1fa829f",
		},
		{
			bitDepth:     12,
			name:         "cabac-mode1-deblock",
			sourceFile:   "high10_implicit_weight_b_deblock_cabac.h264",
			cabac:        1,
			width:        32,
			height:       16,
			rawFrameSize: 1536,
			wantSlices:   []int32{1, 2, 3, 3, 2},
			deblock:      []int32{0, 1, 1, 1, 1},
			bitstreamMD5: "eeeb5384ed65c404a7112deef287e0a7",
			frameMD5:     []string{"f94f0f324a3229363360256262647517", "ee568704458b611bfec00a3aee9a3e9d", "1a822aa6478d6e372d59ba4b184a4e6f", "43cdcf7832e78060f730127894031732", "a5db27d3c4a732850d8d8bacf2d221ed"},
			rawVideoMD5:  "8bc78154170330fc7e56f35165696418",
		},
		{
			bitDepth:     12,
			name:         "cabac-mode2-deblock",
			sourceFile:   "high10_implicit_weight_b_deblock_cabac.h264",
			cabac:        1,
			mode2Deblock: true,
			width:        32,
			height:       16,
			rawFrameSize: 1536,
			wantSlices:   []int32{1, 2, 3, 3, 2},
			deblock:      []int32{0, 2, 2, 2, 2},
			bitstreamMD5: "73f47a22f58cf54c3869f20c81c01a98",
			frameMD5:     []string{"f94f0f324a3229363360256262647517", "ee568704458b611bfec00a3aee9a3e9d", "1a822aa6478d6e372d59ba4b184a4e6f", "43cdcf7832e78060f730127894031732", "a5db27d3c4a732850d8d8bacf2d221ed"},
			rawVideoMD5:  "8bc78154170330fc7e56f35165696418",
		},
		{
			bitDepth:     12,
			name:         "cavlc-direct-sub-b8x8-temporal-mode1-deblock",
			sourceFile:   "high10_implicit_weight_cavlc_b8x8_temporal_direct_sub_deblock.h264",
			cabac:        0,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			wantSlices:   []int32{1, 2, 3},
			deblock:      []int32{0, 1, 1},
			bitstreamMD5: "aeeed05bf25c90bfee749c23bb2e91c6",
			frameMD5:     []string{"d4753b9733af2865470fb72f96a37071", "d4753b9733af2865470fb72f96a37071", "d4753b9733af2865470fb72f96a37071"},
			rawVideoMD5:  "c5844f8a45006553335c482758ad0f49",
		},
		{
			bitDepth:     12,
			name:         "cavlc-direct-sub-b8x8-temporal-mode2-deblock",
			sourceFile:   "high10_implicit_weight_cavlc_b8x8_temporal_direct_sub_deblock.h264",
			cabac:        0,
			mode2Deblock: true,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			wantSlices:   []int32{1, 2, 3},
			deblock:      []int32{0, 2, 2},
			bitstreamMD5: "e3aa147732555a8ea7d4f167f4a9dd1f",
			frameMD5:     []string{"d4753b9733af2865470fb72f96a37071", "d4753b9733af2865470fb72f96a37071", "d4753b9733af2865470fb72f96a37071"},
			rawVideoMD5:  "c5844f8a45006553335c482758ad0f49",
		},
		{
			bitDepth:     12,
			name:         "cavlc-direct-sub-b8x8-spatial-mode1-deblock",
			sourceFile:   "high10_implicit_weight_cavlc_b8x8_spatial_direct_sub_deblock.h264",
			cabac:        0,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			wantSlices:   []int32{1, 2, 3},
			deblock:      []int32{0, 1, 1},
			bitstreamMD5: "95cf86f3e3bb28afc234cb8d2ebe20b8",
			frameMD5:     []string{"d4753b9733af2865470fb72f96a37071", "d4753b9733af2865470fb72f96a37071", "d4753b9733af2865470fb72f96a37071"},
			rawVideoMD5:  "c5844f8a45006553335c482758ad0f49",
		},
		{
			bitDepth:     12,
			name:         "cavlc-direct-sub-b8x8-spatial-mode2-deblock",
			sourceFile:   "high10_implicit_weight_cavlc_b8x8_spatial_direct_sub_deblock.h264",
			cabac:        0,
			mode2Deblock: true,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			wantSlices:   []int32{1, 2, 3},
			deblock:      []int32{0, 2, 2},
			bitstreamMD5: "d9e56140af22239c5c5979073a53578f",
			frameMD5:     []string{"d4753b9733af2865470fb72f96a37071", "d4753b9733af2865470fb72f96a37071", "d4753b9733af2865470fb72f96a37071"},
			rawVideoMD5:  "c5844f8a45006553335c482758ad0f49",
		},
		{
			bitDepth:     12,
			name:         "cavlc-direct-sub-b4x4-temporal-mode1-deblock",
			sourceFile:   "high10_implicit_weight_cavlc_b4x4_temporal_direct_sub_deblock.h264",
			cabac:        0,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			wantSlices:   []int32{1, 2, 3},
			deblock:      []int32{0, 1, 1},
			bitstreamMD5: "229c1fccbe0f9a7d2500153f5c79bc9e",
			frameMD5:     []string{"d4753b9733af2865470fb72f96a37071", "d4753b9733af2865470fb72f96a37071", "d4753b9733af2865470fb72f96a37071"},
			rawVideoMD5:  "c5844f8a45006553335c482758ad0f49",
		},
		{
			bitDepth:     12,
			name:         "cavlc-direct-sub-b4x4-temporal-mode2-deblock",
			sourceFile:   "high10_implicit_weight_cavlc_b4x4_temporal_direct_sub_deblock.h264",
			cabac:        0,
			mode2Deblock: true,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			wantSlices:   []int32{1, 2, 3},
			deblock:      []int32{0, 2, 2},
			bitstreamMD5: "928bb5ecbc91b98844421c28c6d01957",
			frameMD5:     []string{"d4753b9733af2865470fb72f96a37071", "d4753b9733af2865470fb72f96a37071", "d4753b9733af2865470fb72f96a37071"},
			rawVideoMD5:  "c5844f8a45006553335c482758ad0f49",
		},
		{
			bitDepth:     12,
			name:         "cavlc-direct-sub-b4x4-spatial-mode1-deblock",
			sourceFile:   "high10_implicit_weight_cavlc_b4x4_spatial_direct_sub_deblock.h264",
			cabac:        0,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			wantSlices:   []int32{1, 2, 3},
			deblock:      []int32{0, 1, 1},
			bitstreamMD5: "71f74c5200210ccac724be678608f5c8",
			frameMD5:     []string{"d4753b9733af2865470fb72f96a37071", "d4753b9733af2865470fb72f96a37071", "d4753b9733af2865470fb72f96a37071"},
			rawVideoMD5:  "c5844f8a45006553335c482758ad0f49",
		},
		{
			bitDepth:     12,
			name:         "cavlc-direct-sub-b4x4-spatial-mode2-deblock",
			sourceFile:   "high10_implicit_weight_cavlc_b4x4_spatial_direct_sub_deblock.h264",
			cabac:        0,
			mode2Deblock: true,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			wantSlices:   []int32{1, 2, 3},
			deblock:      []int32{0, 2, 2},
			bitstreamMD5: "f858edda4db88fa45d0571e48bbce3fb",
			frameMD5:     []string{"d4753b9733af2865470fb72f96a37071", "d4753b9733af2865470fb72f96a37071", "d4753b9733af2865470fb72f96a37071"},
			rawVideoMD5:  "c5844f8a45006553335c482758ad0f49",
		},
		{
			bitDepth:     12,
			name:         "cabac-direct-sub-b8x8-temporal-mode1-deblock",
			sourceFile:   "high10_implicit_weight_cabac_b8x8_temporal_direct_sub_deblock.h264",
			cabac:        1,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			wantSlices:   []int32{1, 2, 3},
			deblock:      []int32{0, 1, 1},
			bitstreamMD5: "90ee7734e10df93c2f612f61afbe47ae",
			frameMD5:     []string{"d4753b9733af2865470fb72f96a37071", "d4753b9733af2865470fb72f96a37071", "d4753b9733af2865470fb72f96a37071"},
			rawVideoMD5:  "c5844f8a45006553335c482758ad0f49",
		},
		{
			bitDepth:     12,
			name:         "cabac-direct-sub-b8x8-temporal-mode2-deblock",
			sourceFile:   "high10_implicit_weight_cabac_b8x8_temporal_direct_sub_deblock.h264",
			cabac:        1,
			mode2Deblock: true,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			wantSlices:   []int32{1, 2, 3},
			deblock:      []int32{0, 2, 2},
			bitstreamMD5: "21482a9bc38c34dd6b16118bb6c08fd4",
			frameMD5:     []string{"d4753b9733af2865470fb72f96a37071", "d4753b9733af2865470fb72f96a37071", "d4753b9733af2865470fb72f96a37071"},
			rawVideoMD5:  "c5844f8a45006553335c482758ad0f49",
		},
		{
			bitDepth:     12,
			name:         "cabac-direct-sub-b8x8-spatial-mode1-deblock",
			sourceFile:   "high10_implicit_weight_cabac_b8x8_spatial_direct_sub_deblock.h264",
			cabac:        1,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			wantSlices:   []int32{1, 2, 3},
			deblock:      []int32{0, 1, 1},
			bitstreamMD5: "63403af1ddcc40ac971c0fec87793b8c",
			frameMD5:     []string{"d4753b9733af2865470fb72f96a37071", "d4753b9733af2865470fb72f96a37071", "d4753b9733af2865470fb72f96a37071"},
			rawVideoMD5:  "c5844f8a45006553335c482758ad0f49",
		},
		{
			bitDepth:     12,
			name:         "cabac-direct-sub-b8x8-spatial-mode2-deblock",
			sourceFile:   "high10_implicit_weight_cabac_b8x8_spatial_direct_sub_deblock.h264",
			cabac:        1,
			mode2Deblock: true,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			wantSlices:   []int32{1, 2, 3},
			deblock:      []int32{0, 2, 2},
			bitstreamMD5: "404d06959d11aad490ddae0d5d18766e",
			frameMD5:     []string{"d4753b9733af2865470fb72f96a37071", "d4753b9733af2865470fb72f96a37071", "d4753b9733af2865470fb72f96a37071"},
			rawVideoMD5:  "c5844f8a45006553335c482758ad0f49",
		},
		{
			bitDepth:     12,
			name:         "cabac-direct-sub-b4x4-temporal-mode1-deblock",
			sourceFile:   "high10_implicit_weight_cabac_b4x4_temporal_direct_sub_deblock.h264",
			cabac:        1,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			wantSlices:   []int32{1, 2, 3},
			deblock:      []int32{0, 1, 1},
			bitstreamMD5: "a10fb351e82c5e4b4f051ae350c3aec6",
			frameMD5:     []string{"d4753b9733af2865470fb72f96a37071", "d4753b9733af2865470fb72f96a37071", "d4753b9733af2865470fb72f96a37071"},
			rawVideoMD5:  "c5844f8a45006553335c482758ad0f49",
		},
		{
			bitDepth:     12,
			name:         "cabac-direct-sub-b4x4-temporal-mode2-deblock",
			sourceFile:   "high10_implicit_weight_cabac_b4x4_temporal_direct_sub_deblock.h264",
			cabac:        1,
			mode2Deblock: true,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			wantSlices:   []int32{1, 2, 3},
			deblock:      []int32{0, 2, 2},
			bitstreamMD5: "2480467c891dc3e1a45ada7150ff326a",
			frameMD5:     []string{"d4753b9733af2865470fb72f96a37071", "d4753b9733af2865470fb72f96a37071", "d4753b9733af2865470fb72f96a37071"},
			rawVideoMD5:  "c5844f8a45006553335c482758ad0f49",
		},
		{
			bitDepth:     12,
			name:         "cabac-direct-sub-b4x4-spatial-mode1-deblock",
			sourceFile:   "high10_implicit_weight_cabac_b4x4_spatial_direct_sub_deblock.h264",
			cabac:        1,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			wantSlices:   []int32{1, 2, 3},
			deblock:      []int32{0, 1, 1},
			bitstreamMD5: "a8582d38ef7cbf69e1996f2b93fb782a",
			frameMD5:     []string{"d4753b9733af2865470fb72f96a37071", "d4753b9733af2865470fb72f96a37071", "d4753b9733af2865470fb72f96a37071"},
			rawVideoMD5:  "c5844f8a45006553335c482758ad0f49",
		},
		{
			bitDepth:     12,
			name:         "cabac-direct-sub-b4x4-spatial-mode2-deblock",
			sourceFile:   "high10_implicit_weight_cabac_b4x4_spatial_direct_sub_deblock.h264",
			cabac:        1,
			mode2Deblock: true,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			wantSlices:   []int32{1, 2, 3},
			deblock:      []int32{0, 2, 2},
			bitstreamMD5: "5f431d9ec8e69b953e5b759a2cbda3c7",
			frameMD5:     []string{"d4753b9733af2865470fb72f96a37071", "d4753b9733af2865470fb72f96a37071", "d4753b9733af2865470fb72f96a37071"},
			rawVideoMD5:  "c5844f8a45006553335c482758ad0f49",
		},
		{
			bitDepth:     12,
			name:         "cavlc-partitioned-b16x8-no-deblock",
			sourceFile:   "high10_partitioned_implicit_weight_b16x8_cavlc.h264",
			cabac:        0,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			wantSlices:   []int32{1, 2, 3, 3, 2},
			deblock:      []int32{0, 0, 0, 0, 0},
			bitstreamMD5: "7943c49798326a4fce5186dcfee7392e",
			frameMD5:     []string{"ade4ad7e683293ad68a2dc3c9361278b", "e372390bbc8443e2acbf1e859a11d8ef", "a4577e8b4131ca5860729044764bceb8", "b7b0e1d413b0cc5b7a62dcab64640155", "d9a78ae6ec163f39bffd9ef2ff730c84"},
			rawVideoMD5:  "4e3f18d154ed3b3bc5aa127b3672361a",
		},
		{
			bitDepth:     12,
			name:         "cabac-partitioned-b16x8-no-deblock",
			sourceFile:   "high10_partitioned_implicit_weight_b16x8_cabac.h264",
			cabac:        1,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			wantSlices:   []int32{1, 2, 3, 3, 2},
			deblock:      []int32{0, 0, 0, 0, 0},
			bitstreamMD5: "aebad4b50d65d20ee9b2b51485e87ed3",
			frameMD5:     []string{"8951fb646c0a7ec597842fc225f6dbc3", "14a709f23644fd78fb2ab274056b75dd", "980c1af1a15079739154a21aede32876", "6c39e7b5944462bb24e3eb4227f79c3a", "5ac684e18682d64a70e58704a0868497"},
			rawVideoMD5:  "4531bde85186b28ac7157071d9f9db94",
		},
		{
			bitDepth:     12,
			name:         "cavlc-partitioned-b8x16-no-deblock",
			sourceFile:   "high10_partitioned_implicit_weight_b8x16_cavlc.h264",
			cabac:        0,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			wantSlices:   []int32{1, 2, 3, 3, 2},
			deblock:      []int32{0, 0, 0, 0, 0},
			bitstreamMD5: "66963966fd8fd6b3486ab74e8cf9c3d2",
			frameMD5:     []string{"c5ed38b6d73a131d7eff20f34ff7a2f9", "9af3d17d256014b47241e6e1ff9c48fc", "31f7c02de1e4e63d25204f2f6f76c92a", "54c6b7acd225947494942db4d4f39bc8", "9db0994f97fc8679ccb22ff8c7b36ff6"},
			rawVideoMD5:  "da3d449c942bff2544f0dc4f10996630",
		},
		{
			bitDepth:     12,
			name:         "cabac-partitioned-b8x16-no-deblock",
			sourceFile:   "high10_partitioned_implicit_weight_b8x16_cabac.h264",
			cabac:        1,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			wantSlices:   []int32{1, 2, 3, 3, 2},
			deblock:      []int32{0, 0, 0, 0, 0},
			bitstreamMD5: "8bcbb7ef520a267c46c606ca1735bde1",
			frameMD5:     []string{"50d103ccff955b9215971dfaf40d8f67", "b27c6f0e24eff4874a919c55cc067a86", "63275c27b10b843e45226e14954cad98", "616864c271c6dfd30102a52557aed7b0", "bbaadd17b1213f8e9507ded718869507"},
			rawVideoMD5:  "f3e83b471dcffb530e0f8419e1baf625",
		},
		{
			bitDepth:     12,
			name:         "cavlc-partitioned-b8x8-no-deblock",
			sourceFile:   "high10_partitioned_implicit_weight_b8x8_cavlc.h264",
			cabac:        0,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			wantSlices:   []int32{1, 2, 3, 3, 2},
			deblock:      []int32{0, 0, 0, 0, 0},
			bitstreamMD5: "e833e3bbf9fd8ee392393d44bbee3db1",
			frameMD5:     []string{"9e5e8a1cf791aa907c3e4d4041e8f869", "db34169c5da80f74769c3504c5faa9af", "4fdc96f3c829c1522e9819b218f487ea", "ce26dbc242ec57f5d3ec074ddc12a0b7", "fda23ed601ae8f581638dc63fd333629"},
			rawVideoMD5:  "1957a2b35322b6fc207af035f549621e",
		},
		{
			bitDepth:     12,
			name:         "cabac-partitioned-b8x8-no-deblock",
			sourceFile:   "high10_partitioned_implicit_weight_b8x8_cabac.h264",
			cabac:        1,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			wantSlices:   []int32{1, 2, 3, 3, 2},
			deblock:      []int32{0, 0, 0, 0, 0},
			bitstreamMD5: "02c0cb7c98be915a63eb361d1cb6989c",
			frameMD5:     []string{"9e5e8a1cf791aa907c3e4d4041e8f869", "db34169c5da80f74769c3504c5faa9af", "9eaafc946d5370813d2f9825509248e1", "ce26dbc242ec57f5d3ec074ddc12a0b7", "fda23ed601ae8f581638dc63fd333629"},
			rawVideoMD5:  "7eea14dfc92a3aad6cd396abba0f36be",
		},
		{
			bitDepth:     12,
			name:         "cavlc-partitioned-b16x8-mode1-deblock",
			sourceFile:   "high10_partitioned_implicit_weight_b_deblock_b16x8_cavlc.h264",
			cabac:        0,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			wantSlices:   []int32{1, 2, 3, 3, 2},
			deblock:      []int32{1, 1, 1, 1, 1},
			bitstreamMD5: "2f2a5e96bf2144e4772050e9f4ee7008",
			frameMD5:     []string{"271fc9eab6e0d7c98af20f9ecffd0491", "0d3c4890ccc080bfec51093276bc8738", "c4944df57076716dcb00e10b2a2dbdbb", "11bc3a789dd0b701afa7e4e9e5c137c9", "8bbd70c55f2113ce370f0a5c96b0ac09"},
			rawVideoMD5:  "3f313eb2ae836bae50a4950be5aa2f56",
		},
		{
			bitDepth:     12,
			name:         "cavlc-partitioned-b16x8-mode2-deblock",
			sourceFile:   "high10_partitioned_implicit_weight_b_deblock_b16x8_cavlc.h264",
			cabac:        0,
			mode2Deblock: true,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			wantSlices:   []int32{1, 2, 3, 3, 2},
			deblock:      []int32{1, 2, 2, 2, 2},
			bitstreamMD5: "c0c2352f7295e055c9e0c9082534e6b1",
			frameMD5:     []string{"271fc9eab6e0d7c98af20f9ecffd0491", "0d3c4890ccc080bfec51093276bc8738", "c4944df57076716dcb00e10b2a2dbdbb", "11bc3a789dd0b701afa7e4e9e5c137c9", "8bbd70c55f2113ce370f0a5c96b0ac09"},
			rawVideoMD5:  "3f313eb2ae836bae50a4950be5aa2f56",
		},
		{
			bitDepth:     12,
			name:         "cabac-partitioned-b16x8-mode1-deblock",
			sourceFile:   "high10_partitioned_implicit_weight_b_deblock_b16x8_cabac.h264",
			cabac:        1,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			wantSlices:   []int32{1, 2, 3, 3, 2},
			deblock:      []int32{1, 1, 1, 1, 1},
			bitstreamMD5: "354ea63c16eb2a13390dd53b7d2e85a3",
			frameMD5:     []string{"c2bd0dd90f1cf7ed33424c06f47454a5", "fdc42261f3a5a54f96d48a53c6e59738", "cf7988d1c26d1d3295dc57c537868bd8", "11bc3a789dd0b701afa7e4e9e5c137c9", "8bbd70c55f2113ce370f0a5c96b0ac09"},
			rawVideoMD5:  "5a304ad9e13e42e0df0cab7a4ded3d60",
		},
		{
			bitDepth:     12,
			name:         "cabac-partitioned-b16x8-mode2-deblock",
			sourceFile:   "high10_partitioned_implicit_weight_b_deblock_b16x8_cabac.h264",
			cabac:        1,
			mode2Deblock: true,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			wantSlices:   []int32{1, 2, 3, 3, 2},
			deblock:      []int32{1, 2, 2, 2, 2},
			bitstreamMD5: "68b6efd62543a0693658b4e87d55592f",
			frameMD5:     []string{"c2bd0dd90f1cf7ed33424c06f47454a5", "fdc42261f3a5a54f96d48a53c6e59738", "cf7988d1c26d1d3295dc57c537868bd8", "11bc3a789dd0b701afa7e4e9e5c137c9", "8bbd70c55f2113ce370f0a5c96b0ac09"},
			rawVideoMD5:  "5a304ad9e13e42e0df0cab7a4ded3d60",
		},
		{
			bitDepth:     12,
			name:         "cavlc-partitioned-b8x16-mode1-deblock",
			sourceFile:   "high10_partitioned_implicit_weight_b_deblock_b8x16_cavlc.h264",
			cabac:        0,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			wantSlices:   []int32{1, 2, 3, 3, 2},
			deblock:      []int32{1, 1, 1, 1, 1},
			bitstreamMD5: "e63acb8fde593ca68cc26e0cd7fa9251",
			frameMD5:     []string{"ff5aaa5c613fc0046f4224b7b27b68b6", "e03c89cbb3f6fd8a3ccf37f9f11fd17b", "d70ec9e8e288c2fbe8626406465bc680", "3740befb8f4f876023e6c209d55b56e1", "f68edd1b4866284784aeb99943da8b4e"},
			rawVideoMD5:  "a817b89e01744a52e0f3e0813a9305ee",
		},
		{
			bitDepth:     12,
			name:         "cavlc-partitioned-b8x16-mode2-deblock",
			sourceFile:   "high10_partitioned_implicit_weight_b_deblock_b8x16_cavlc.h264",
			cabac:        0,
			mode2Deblock: true,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			wantSlices:   []int32{1, 2, 3, 3, 2},
			deblock:      []int32{1, 2, 2, 2, 2},
			bitstreamMD5: "f055ae16639d8ee9b542449a05d9605e",
			frameMD5:     []string{"ff5aaa5c613fc0046f4224b7b27b68b6", "e03c89cbb3f6fd8a3ccf37f9f11fd17b", "d70ec9e8e288c2fbe8626406465bc680", "3740befb8f4f876023e6c209d55b56e1", "f68edd1b4866284784aeb99943da8b4e"},
			rawVideoMD5:  "a817b89e01744a52e0f3e0813a9305ee",
		},
		{
			bitDepth:     12,
			name:         "cabac-partitioned-b8x16-mode1-deblock",
			sourceFile:   "high10_partitioned_implicit_weight_b_deblock_b8x16_cabac.h264",
			cabac:        1,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			wantSlices:   []int32{1, 2, 3, 3, 2},
			deblock:      []int32{1, 1, 1, 1, 1},
			bitstreamMD5: "e78e9012104662489f5b37df31877309",
			frameMD5:     []string{"08b7418359830aae9eb5778f08a37a81", "9262af617ca1cd9b20b958be5a9a6916", "526bc0c11204a20581739278bded1434", "04d9f462359b99bbdca7de8e9f53e75a", "412ef787cad8ff1f0da2948ff119bc67"},
			rawVideoMD5:  "23ed79b1346e28e5acb56f1c18fe3441",
		},
		{
			bitDepth:     12,
			name:         "cabac-partitioned-b8x16-mode2-deblock",
			sourceFile:   "high10_partitioned_implicit_weight_b_deblock_b8x16_cabac.h264",
			cabac:        1,
			mode2Deblock: true,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			wantSlices:   []int32{1, 2, 3, 3, 2},
			deblock:      []int32{1, 2, 2, 2, 2},
			bitstreamMD5: "9fbe3c97b9c5fe36d3daec2ffd1ef866",
			frameMD5:     []string{"08b7418359830aae9eb5778f08a37a81", "9262af617ca1cd9b20b958be5a9a6916", "526bc0c11204a20581739278bded1434", "04d9f462359b99bbdca7de8e9f53e75a", "412ef787cad8ff1f0da2948ff119bc67"},
			rawVideoMD5:  "23ed79b1346e28e5acb56f1c18fe3441",
		},
		{
			bitDepth:     12,
			name:         "cavlc-partitioned-b8x8-mode1-deblock",
			sourceFile:   "high10_partitioned_implicit_weight_b_deblock_b8x8_cavlc.h264",
			cabac:        0,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			wantSlices:   []int32{1, 2, 3, 3, 2},
			deblock:      []int32{1, 1, 1, 1, 1},
			bitstreamMD5: "691cdaa460cfbaac346348246fb6e38f",
			frameMD5:     []string{"69c7144d64f8fcc0994be3f0cfbe4b5d", "762ef5a3828d54fc0910f6b53acd79a5", "3688a24baf0ce2a93b8f9fccc55f8cfe", "a170c75ad560003e135f037b8df24609", "68dddd659051cf9922494b235f4bb5d7"},
			rawVideoMD5:  "717791dcdd61af0ce22de108e5851753",
		},
		{
			bitDepth:     12,
			name:         "cavlc-partitioned-b8x8-mode2-deblock",
			sourceFile:   "high10_partitioned_implicit_weight_b_deblock_b8x8_cavlc.h264",
			cabac:        0,
			mode2Deblock: true,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			wantSlices:   []int32{1, 2, 3, 3, 2},
			deblock:      []int32{1, 2, 2, 2, 2},
			bitstreamMD5: "1fd83a984d876386f0d28d9fd661e738",
			frameMD5:     []string{"69c7144d64f8fcc0994be3f0cfbe4b5d", "762ef5a3828d54fc0910f6b53acd79a5", "3688a24baf0ce2a93b8f9fccc55f8cfe", "a170c75ad560003e135f037b8df24609", "68dddd659051cf9922494b235f4bb5d7"},
			rawVideoMD5:  "717791dcdd61af0ce22de108e5851753",
		},
		{
			bitDepth:     12,
			name:         "cabac-partitioned-b8x8-mode1-deblock",
			sourceFile:   "high10_partitioned_implicit_weight_b_deblock_b8x8_cabac.h264",
			cabac:        1,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			wantSlices:   []int32{1, 2, 3, 3, 2},
			deblock:      []int32{1, 1, 1, 1, 1},
			bitstreamMD5: "ff45cf447a4ceff2a707ac98ffc5500e",
			frameMD5:     []string{"69c7144d64f8fcc0994be3f0cfbe4b5d", "ebe85583f590b8ec2ed0b007f9ada5a2", "bc8cd36867206e07c5dc4bb432262e00", "a170c75ad560003e135f037b8df24609", "d1cab7cbb0d8cfda8d2a80c683d6cf41"},
			rawVideoMD5:  "c848b5284d0faae2ca2f24f1ff7f3087",
		},
		{
			bitDepth:     12,
			name:         "cabac-partitioned-b8x8-mode2-deblock",
			sourceFile:   "high10_partitioned_implicit_weight_b_deblock_b8x8_cabac.h264",
			cabac:        1,
			mode2Deblock: true,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			wantSlices:   []int32{1, 2, 3, 3, 2},
			deblock:      []int32{1, 2, 2, 2, 2},
			bitstreamMD5: "398e296e63225690203e22c77dea53d4",
			frameMD5:     []string{"69c7144d64f8fcc0994be3f0cfbe4b5d", "ebe85583f590b8ec2ed0b007f9ada5a2", "bc8cd36867206e07c5dc4bb432262e00", "a170c75ad560003e135f037b8df24609", "d1cab7cbb0d8cfda8d2a80c683d6cf41"},
			rawVideoMD5:  "c848b5284d0faae2ca2f24f1ff7f3087",
		},
		{
			bitDepth:     14,
			name:         "cavlc-no-deblock",
			sourceFile:   "high10_implicit_weight_b_cavlc.h264",
			cabac:        0,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			wantSlices:   []int32{1, 2, 3, 3, 2},
			deblock:      []int32{0, 0, 0, 0, 0},
			bitstreamMD5: "2bb0961670c1ff47b5827c8787c1f0cb",
			frameMD5:     []string{"6d3514a30f506561e144447d287270ab", "4acd37bae1d468c359df1e6a2d03c623", "38a8c26ad1658bfe6ff32123dcafba67", "8a776a45cff9c94f884ccabea68f5e94", "d3641da2bd4bf8a1289cd4e1f6c4f9f7"},
			rawVideoMD5:  "13f94af6c91ef85d732cf954988555a7",
		},
		{
			bitDepth:     14,
			name:         "cabac-no-deblock",
			sourceFile:   "high10_implicit_weight_b_cabac.h264",
			cabac:        1,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			wantSlices:   []int32{1, 2, 3, 3, 2},
			deblock:      []int32{0, 0, 0, 0, 0},
			bitstreamMD5: "42e34751b4c5a01823985adc28cf2a09",
			frameMD5:     []string{"6d3514a30f506561e144447d287270ab", "4acd37bae1d468c359df1e6a2d03c623", "38a8c26ad1658bfe6ff32123dcafba67", "8a776a45cff9c94f884ccabea68f5e94", "d3641da2bd4bf8a1289cd4e1f6c4f9f7"},
			rawVideoMD5:  "13f94af6c91ef85d732cf954988555a7",
		},
		{
			bitDepth:     14,
			name:         "cavlc-mode1-deblock",
			sourceFile:   "high10_implicit_weight_b_deblock_cavlc.h264",
			cabac:        0,
			width:        32,
			height:       16,
			rawFrameSize: 1536,
			wantSlices:   []int32{1, 2, 3, 3, 2},
			deblock:      []int32{0, 1, 1, 1, 1},
			bitstreamMD5: "11c0fc7871d6ec851c20679622f068b0",
			frameMD5:     []string{"3191e00b8ace54980df9c2b2c9e2c5e2", "f2ced02c400c492feb7ac13939e40966", "b9938ed43cb9869422d577226b7e5f6f", "2eb2f905a6b4739ee22e0bb138b71a6a", "aadd3e1edceb4577255d9720e4372b04"},
			rawVideoMD5:  "cd6e9a35548591b7f50bec659bcc719f",
		},
		{
			bitDepth:     14,
			name:         "cavlc-mode2-deblock",
			sourceFile:   "high10_implicit_weight_b_deblock_cavlc.h264",
			cabac:        0,
			mode2Deblock: true,
			width:        32,
			height:       16,
			rawFrameSize: 1536,
			wantSlices:   []int32{1, 2, 3, 3, 2},
			deblock:      []int32{0, 2, 2, 2, 2},
			bitstreamMD5: "ccebadeabe59128186ac03707db9296d",
			frameMD5:     []string{"3191e00b8ace54980df9c2b2c9e2c5e2", "f2ced02c400c492feb7ac13939e40966", "b9938ed43cb9869422d577226b7e5f6f", "2eb2f905a6b4739ee22e0bb138b71a6a", "aadd3e1edceb4577255d9720e4372b04"},
			rawVideoMD5:  "cd6e9a35548591b7f50bec659bcc719f",
		},
		{
			bitDepth:     14,
			name:         "cabac-mode1-deblock",
			sourceFile:   "high10_implicit_weight_b_deblock_cabac.h264",
			cabac:        1,
			width:        32,
			height:       16,
			rawFrameSize: 1536,
			wantSlices:   []int32{1, 2, 3, 3, 2},
			deblock:      []int32{0, 1, 1, 1, 1},
			bitstreamMD5: "01f05ce2d48dfb2d9471de77c246f639",
			frameMD5:     []string{"ba2c72108ade4f4fb88f182fd25e9c15", "e506cf15911bb465feb4fbdc4defb9d1", "0d305c7e6eeabed863897bcdce23b3d6", "7b6204ca8ef61175eea1ac577f1022b6", "bd654782846633b1429650d60f451ada"},
			rawVideoMD5:  "5c889a75ae35ce0e50c90c6c312a7b6b",
		},
		{
			bitDepth:     14,
			name:         "cabac-mode2-deblock",
			sourceFile:   "high10_implicit_weight_b_deblock_cabac.h264",
			cabac:        1,
			mode2Deblock: true,
			width:        32,
			height:       16,
			rawFrameSize: 1536,
			wantSlices:   []int32{1, 2, 3, 3, 2},
			deblock:      []int32{0, 2, 2, 2, 2},
			bitstreamMD5: "5e9e5d57e6a77349f34634f5a130bbf1",
			frameMD5:     []string{"ba2c72108ade4f4fb88f182fd25e9c15", "e506cf15911bb465feb4fbdc4defb9d1", "0d305c7e6eeabed863897bcdce23b3d6", "7b6204ca8ef61175eea1ac577f1022b6", "bd654782846633b1429650d60f451ada"},
			rawVideoMD5:  "5c889a75ae35ce0e50c90c6c312a7b6b",
		},
		{
			bitDepth:     14,
			name:         "cavlc-direct-sub-b8x8-temporal-mode1-deblock",
			sourceFile:   "high10_implicit_weight_cavlc_b8x8_temporal_direct_sub_deblock.h264",
			cabac:        0,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			wantSlices:   []int32{1, 2, 3},
			deblock:      []int32{0, 1, 1},
			bitstreamMD5: "18f3f15689f2e12fd09c99056a67761b",
			frameMD5:     []string{"6d3514a30f506561e144447d287270ab", "6d3514a30f506561e144447d287270ab", "6d3514a30f506561e144447d287270ab"},
			rawVideoMD5:  "1f17900b95954131ed58ae42da301444",
		},
		{
			bitDepth:     14,
			name:         "cavlc-direct-sub-b8x8-temporal-mode2-deblock",
			sourceFile:   "high10_implicit_weight_cavlc_b8x8_temporal_direct_sub_deblock.h264",
			cabac:        0,
			mode2Deblock: true,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			wantSlices:   []int32{1, 2, 3},
			deblock:      []int32{0, 2, 2},
			bitstreamMD5: "ec28ad58d4b322d0e20a44ae84f92c5a",
			frameMD5:     []string{"6d3514a30f506561e144447d287270ab", "6d3514a30f506561e144447d287270ab", "6d3514a30f506561e144447d287270ab"},
			rawVideoMD5:  "1f17900b95954131ed58ae42da301444",
		},
		{
			bitDepth:     14,
			name:         "cavlc-direct-sub-b8x8-spatial-mode1-deblock",
			sourceFile:   "high10_implicit_weight_cavlc_b8x8_spatial_direct_sub_deblock.h264",
			cabac:        0,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			wantSlices:   []int32{1, 2, 3},
			deblock:      []int32{0, 1, 1},
			bitstreamMD5: "eb4a78a1e0f7cd6eb439572d1af9c01b",
			frameMD5:     []string{"6d3514a30f506561e144447d287270ab", "6d3514a30f506561e144447d287270ab", "6d3514a30f506561e144447d287270ab"},
			rawVideoMD5:  "1f17900b95954131ed58ae42da301444",
		},
		{
			bitDepth:     14,
			name:         "cavlc-direct-sub-b8x8-spatial-mode2-deblock",
			sourceFile:   "high10_implicit_weight_cavlc_b8x8_spatial_direct_sub_deblock.h264",
			cabac:        0,
			mode2Deblock: true,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			wantSlices:   []int32{1, 2, 3},
			deblock:      []int32{0, 2, 2},
			bitstreamMD5: "30bdbdab0190b224ab6ffaf764bc91ae",
			frameMD5:     []string{"6d3514a30f506561e144447d287270ab", "6d3514a30f506561e144447d287270ab", "6d3514a30f506561e144447d287270ab"},
			rawVideoMD5:  "1f17900b95954131ed58ae42da301444",
		},
		{
			bitDepth:     14,
			name:         "cavlc-direct-sub-b4x4-temporal-mode1-deblock",
			sourceFile:   "high10_implicit_weight_cavlc_b4x4_temporal_direct_sub_deblock.h264",
			cabac:        0,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			wantSlices:   []int32{1, 2, 3},
			deblock:      []int32{0, 1, 1},
			bitstreamMD5: "9567fa2056a47873fe650737e6ac0b15",
			frameMD5:     []string{"6d3514a30f506561e144447d287270ab", "6d3514a30f506561e144447d287270ab", "6d3514a30f506561e144447d287270ab"},
			rawVideoMD5:  "1f17900b95954131ed58ae42da301444",
		},
		{
			bitDepth:     14,
			name:         "cavlc-direct-sub-b4x4-temporal-mode2-deblock",
			sourceFile:   "high10_implicit_weight_cavlc_b4x4_temporal_direct_sub_deblock.h264",
			cabac:        0,
			mode2Deblock: true,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			wantSlices:   []int32{1, 2, 3},
			deblock:      []int32{0, 2, 2},
			bitstreamMD5: "6373acf5638d1bce7bc05dd9f3ce65aa",
			frameMD5:     []string{"6d3514a30f506561e144447d287270ab", "6d3514a30f506561e144447d287270ab", "6d3514a30f506561e144447d287270ab"},
			rawVideoMD5:  "1f17900b95954131ed58ae42da301444",
		},
		{
			bitDepth:     14,
			name:         "cavlc-direct-sub-b4x4-spatial-mode1-deblock",
			sourceFile:   "high10_implicit_weight_cavlc_b4x4_spatial_direct_sub_deblock.h264",
			cabac:        0,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			wantSlices:   []int32{1, 2, 3},
			deblock:      []int32{0, 1, 1},
			bitstreamMD5: "f789fe165d572071233c54090726976b",
			frameMD5:     []string{"6d3514a30f506561e144447d287270ab", "6d3514a30f506561e144447d287270ab", "6d3514a30f506561e144447d287270ab"},
			rawVideoMD5:  "1f17900b95954131ed58ae42da301444",
		},
		{
			bitDepth:     14,
			name:         "cavlc-direct-sub-b4x4-spatial-mode2-deblock",
			sourceFile:   "high10_implicit_weight_cavlc_b4x4_spatial_direct_sub_deblock.h264",
			cabac:        0,
			mode2Deblock: true,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			wantSlices:   []int32{1, 2, 3},
			deblock:      []int32{0, 2, 2},
			bitstreamMD5: "9e9584caeae0445d8c5e6364f9651c2e",
			frameMD5:     []string{"6d3514a30f506561e144447d287270ab", "6d3514a30f506561e144447d287270ab", "6d3514a30f506561e144447d287270ab"},
			rawVideoMD5:  "1f17900b95954131ed58ae42da301444",
		},
		{
			bitDepth:     14,
			name:         "cabac-direct-sub-b8x8-temporal-mode1-deblock",
			sourceFile:   "high10_implicit_weight_cabac_b8x8_temporal_direct_sub_deblock.h264",
			cabac:        1,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			wantSlices:   []int32{1, 2, 3},
			deblock:      []int32{0, 1, 1},
			bitstreamMD5: "4cb280ab12b0b7c8e63793ad0e7fd426",
			frameMD5:     []string{"6d3514a30f506561e144447d287270ab", "6d3514a30f506561e144447d287270ab", "6d3514a30f506561e144447d287270ab"},
			rawVideoMD5:  "1f17900b95954131ed58ae42da301444",
		},
		{
			bitDepth:     14,
			name:         "cabac-direct-sub-b8x8-temporal-mode2-deblock",
			sourceFile:   "high10_implicit_weight_cabac_b8x8_temporal_direct_sub_deblock.h264",
			cabac:        1,
			mode2Deblock: true,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			wantSlices:   []int32{1, 2, 3},
			deblock:      []int32{0, 2, 2},
			bitstreamMD5: "5e2ce97cb5f99bab18ade4e536d26c97",
			frameMD5:     []string{"6d3514a30f506561e144447d287270ab", "6d3514a30f506561e144447d287270ab", "6d3514a30f506561e144447d287270ab"},
			rawVideoMD5:  "1f17900b95954131ed58ae42da301444",
		},
		{
			bitDepth:     14,
			name:         "cabac-direct-sub-b8x8-spatial-mode1-deblock",
			sourceFile:   "high10_implicit_weight_cabac_b8x8_spatial_direct_sub_deblock.h264",
			cabac:        1,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			wantSlices:   []int32{1, 2, 3},
			deblock:      []int32{0, 1, 1},
			bitstreamMD5: "2404c6f2ab42288989588f0c7fbdf06d",
			frameMD5:     []string{"6d3514a30f506561e144447d287270ab", "6d3514a30f506561e144447d287270ab", "6d3514a30f506561e144447d287270ab"},
			rawVideoMD5:  "1f17900b95954131ed58ae42da301444",
		},
		{
			bitDepth:     14,
			name:         "cabac-direct-sub-b8x8-spatial-mode2-deblock",
			sourceFile:   "high10_implicit_weight_cabac_b8x8_spatial_direct_sub_deblock.h264",
			cabac:        1,
			mode2Deblock: true,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			wantSlices:   []int32{1, 2, 3},
			deblock:      []int32{0, 2, 2},
			bitstreamMD5: "9179d01cf658ec4c01ed5506727ae398",
			frameMD5:     []string{"6d3514a30f506561e144447d287270ab", "6d3514a30f506561e144447d287270ab", "6d3514a30f506561e144447d287270ab"},
			rawVideoMD5:  "1f17900b95954131ed58ae42da301444",
		},
		{
			bitDepth:     14,
			name:         "cabac-direct-sub-b4x4-temporal-mode1-deblock",
			sourceFile:   "high10_implicit_weight_cabac_b4x4_temporal_direct_sub_deblock.h264",
			cabac:        1,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			wantSlices:   []int32{1, 2, 3},
			deblock:      []int32{0, 1, 1},
			bitstreamMD5: "dbe58f165b53ccd43b0d73e44df823cf",
			frameMD5:     []string{"6d3514a30f506561e144447d287270ab", "6d3514a30f506561e144447d287270ab", "6d3514a30f506561e144447d287270ab"},
			rawVideoMD5:  "1f17900b95954131ed58ae42da301444",
		},
		{
			bitDepth:     14,
			name:         "cabac-direct-sub-b4x4-temporal-mode2-deblock",
			sourceFile:   "high10_implicit_weight_cabac_b4x4_temporal_direct_sub_deblock.h264",
			cabac:        1,
			mode2Deblock: true,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			wantSlices:   []int32{1, 2, 3},
			deblock:      []int32{0, 2, 2},
			bitstreamMD5: "a85f4cba8cbdd0cc93a8b306e0e627fd",
			frameMD5:     []string{"6d3514a30f506561e144447d287270ab", "6d3514a30f506561e144447d287270ab", "6d3514a30f506561e144447d287270ab"},
			rawVideoMD5:  "1f17900b95954131ed58ae42da301444",
		},
		{
			bitDepth:     14,
			name:         "cabac-direct-sub-b4x4-spatial-mode1-deblock",
			sourceFile:   "high10_implicit_weight_cabac_b4x4_spatial_direct_sub_deblock.h264",
			cabac:        1,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			wantSlices:   []int32{1, 2, 3},
			deblock:      []int32{0, 1, 1},
			bitstreamMD5: "5464d054725a4eaffb593da191cf0588",
			frameMD5:     []string{"6d3514a30f506561e144447d287270ab", "6d3514a30f506561e144447d287270ab", "6d3514a30f506561e144447d287270ab"},
			rawVideoMD5:  "1f17900b95954131ed58ae42da301444",
		},
		{
			bitDepth:     14,
			name:         "cabac-direct-sub-b4x4-spatial-mode2-deblock",
			sourceFile:   "high10_implicit_weight_cabac_b4x4_spatial_direct_sub_deblock.h264",
			cabac:        1,
			mode2Deblock: true,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			wantSlices:   []int32{1, 2, 3},
			deblock:      []int32{0, 2, 2},
			bitstreamMD5: "dea7fc62f3e734aacea5f8b135f23c63",
			frameMD5:     []string{"6d3514a30f506561e144447d287270ab", "6d3514a30f506561e144447d287270ab", "6d3514a30f506561e144447d287270ab"},
			rawVideoMD5:  "1f17900b95954131ed58ae42da301444",
		},
		{
			bitDepth:     14,
			name:         "cavlc-partitioned-b16x8-no-deblock",
			sourceFile:   "high10_partitioned_implicit_weight_b16x8_cavlc.h264",
			cabac:        0,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			wantSlices:   []int32{1, 2, 3, 3, 2},
			deblock:      []int32{0, 0, 0, 0, 0},
			bitstreamMD5: "c7105bfb7fc69014e2e4bd6b8e156018",
			frameMD5:     []string{"8f6637034ed7562185064dcfa1b65c1e", "bfaba84ec5568bdfc8efbd35c70c1691", "21588e030169520342bbe17f58dbee9a", "439523466dce0a675d9a6b632286170b", "f0a0446289715444f6e00885fb9082b8"},
			rawVideoMD5:  "f78d40aee01a98b02e5aaaa5934a506f",
		},
		{
			bitDepth:     14,
			name:         "cabac-partitioned-b16x8-no-deblock",
			sourceFile:   "high10_partitioned_implicit_weight_b16x8_cabac.h264",
			cabac:        1,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			wantSlices:   []int32{1, 2, 3, 3, 2},
			deblock:      []int32{0, 0, 0, 0, 0},
			bitstreamMD5: "e1179af1fbb38743a5bc38df0ceea7ec",
			frameMD5:     []string{"93596a9bd5980c195830fc88dfc67da8", "06d5c0ccf137f837d6a2559efd04ea74", "4038b19fc704f102de768562b4069c07", "d30faa08b72e9a40047090177a418a3a", "c067a2e153fdbf85816bcc02964715b5"},
			rawVideoMD5:  "7f365930cb6bad242654317ee9390c54",
		},
		{
			bitDepth:     14,
			name:         "cavlc-partitioned-b8x16-no-deblock",
			sourceFile:   "high10_partitioned_implicit_weight_b8x16_cavlc.h264",
			cabac:        0,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			wantSlices:   []int32{1, 2, 3, 3, 2},
			deblock:      []int32{0, 0, 0, 0, 0},
			bitstreamMD5: "f1707def080c4031860f0d2d76fc45b1",
			frameMD5:     []string{"8c25dcf589eb44488598375c1672c2f8", "11550156ceb8f223272eb0d49cc64a8f", "6046ab0f9f85e4b666539c9ac2c63c8f", "988ce2ab88947b2a3d636cda3af7f196", "cad599c236a3944f891198603c878561"},
			rawVideoMD5:  "474d934a623ea1c063ba5c7df8d0537e",
		},
		{
			bitDepth:     14,
			name:         "cabac-partitioned-b8x16-no-deblock",
			sourceFile:   "high10_partitioned_implicit_weight_b8x16_cabac.h264",
			cabac:        1,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			wantSlices:   []int32{1, 2, 3, 3, 2},
			deblock:      []int32{0, 0, 0, 0, 0},
			bitstreamMD5: "a548557e7bdd1de7b02cf9915e0e35e1",
			frameMD5:     []string{"85350066ebc1709573d3e10d0d0a14bc", "5d13c4a48a1f982e46f7775ddc1d142b", "7e79ea30ef7118c4f19eeec33bd14171", "157e081e0dbfb1a5d6301e2f863b9881", "903373b18bedb3388fd36f7c8a7d3f86"},
			rawVideoMD5:  "c2a9d201746df6b130f80cc530b61e28",
		},
		{
			bitDepth:     14,
			name:         "cavlc-partitioned-b8x8-no-deblock",
			sourceFile:   "high10_partitioned_implicit_weight_b8x8_cavlc.h264",
			cabac:        0,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			wantSlices:   []int32{1, 2, 3, 3, 2},
			deblock:      []int32{0, 0, 0, 0, 0},
			bitstreamMD5: "e40df0ffe2299be5c9dd206c334937a0",
			frameMD5:     []string{"0898fde55c5dac131d8c5f745ffe337f", "dfef68b07ea1dde12db090dac433ee03", "c7488b06581e247bd0231d9e51701834", "b33ece0998aa91831e6b66fd618c3d11", "7fffb6247ec88e0a617cff968ebdeeb0"},
			rawVideoMD5:  "2dd8fa9188c0aac12fdfc34171b87845",
		},
		{
			bitDepth:     14,
			name:         "cabac-partitioned-b8x8-no-deblock",
			sourceFile:   "high10_partitioned_implicit_weight_b8x8_cabac.h264",
			cabac:        1,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			wantSlices:   []int32{1, 2, 3, 3, 2},
			deblock:      []int32{0, 0, 0, 0, 0},
			bitstreamMD5: "301ae702acb653a77b152d20ffec5c0c",
			frameMD5:     []string{"0898fde55c5dac131d8c5f745ffe337f", "dfef68b07ea1dde12db090dac433ee03", "56266144860215b0dca45a2e40db0ad2", "b33ece0998aa91831e6b66fd618c3d11", "7fffb6247ec88e0a617cff968ebdeeb0"},
			rawVideoMD5:  "5636d4911a16361cd1dccf5538b8fd95",
		},
		{
			bitDepth:     14,
			name:         "cavlc-partitioned-b16x8-mode1-deblock",
			sourceFile:   "high10_partitioned_implicit_weight_b_deblock_b16x8_cavlc.h264",
			cabac:        0,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			wantSlices:   []int32{1, 2, 3, 3, 2},
			deblock:      []int32{1, 1, 1, 1, 1},
			bitstreamMD5: "00315f1c855b0f37a6e01fd45c3503b9",
			frameMD5:     []string{"76bba608fb2c663b35187dc6fd5ad541", "d9b8926cacb070d72fb90ce7a4b277f9", "bbeace8cfe6901a86b63662f89718fb8", "12208fd0262429f37e5942b9e2d1f9d8", "2d4008fca992537dbf2c34325410aedd"},
			rawVideoMD5:  "762ad595a03ede1a1a4e9393adbf2c1b",
		},
		{
			bitDepth:     14,
			name:         "cavlc-partitioned-b16x8-mode2-deblock",
			sourceFile:   "high10_partitioned_implicit_weight_b_deblock_b16x8_cavlc.h264",
			cabac:        0,
			mode2Deblock: true,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			wantSlices:   []int32{1, 2, 3, 3, 2},
			deblock:      []int32{1, 2, 2, 2, 2},
			bitstreamMD5: "71f2128ace8892a8612638ec12a4e9da",
			frameMD5:     []string{"76bba608fb2c663b35187dc6fd5ad541", "d9b8926cacb070d72fb90ce7a4b277f9", "bbeace8cfe6901a86b63662f89718fb8", "12208fd0262429f37e5942b9e2d1f9d8", "2d4008fca992537dbf2c34325410aedd"},
			rawVideoMD5:  "762ad595a03ede1a1a4e9393adbf2c1b",
		},
		{
			bitDepth:     14,
			name:         "cabac-partitioned-b16x8-mode1-deblock",
			sourceFile:   "high10_partitioned_implicit_weight_b_deblock_b16x8_cabac.h264",
			cabac:        1,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			wantSlices:   []int32{1, 2, 3, 3, 2},
			deblock:      []int32{1, 1, 1, 1, 1},
			bitstreamMD5: "8affa6c8c4860c066e9a96884846314a",
			frameMD5:     []string{"08cfce46517b1a3fb82d43b66d6c3327", "25b3112989747893b18c7f892ef06151", "470141d109848e52435486bf0c5b6ae4", "12208fd0262429f37e5942b9e2d1f9d8", "2d4008fca992537dbf2c34325410aedd"},
			rawVideoMD5:  "e1efe229db403233497496cb150b5ace",
		},
		{
			bitDepth:     14,
			name:         "cabac-partitioned-b16x8-mode2-deblock",
			sourceFile:   "high10_partitioned_implicit_weight_b_deblock_b16x8_cabac.h264",
			cabac:        1,
			mode2Deblock: true,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			wantSlices:   []int32{1, 2, 3, 3, 2},
			deblock:      []int32{1, 2, 2, 2, 2},
			bitstreamMD5: "371b091fc2d64c7a076c1ba4d9c33460",
			frameMD5:     []string{"08cfce46517b1a3fb82d43b66d6c3327", "25b3112989747893b18c7f892ef06151", "470141d109848e52435486bf0c5b6ae4", "12208fd0262429f37e5942b9e2d1f9d8", "2d4008fca992537dbf2c34325410aedd"},
			rawVideoMD5:  "e1efe229db403233497496cb150b5ace",
		},
		{
			bitDepth:     14,
			name:         "cavlc-partitioned-b8x16-mode1-deblock",
			sourceFile:   "high10_partitioned_implicit_weight_b_deblock_b8x16_cavlc.h264",
			cabac:        0,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			wantSlices:   []int32{1, 2, 3, 3, 2},
			deblock:      []int32{1, 1, 1, 1, 1},
			bitstreamMD5: "9a3e935918ead07024b92008a4563f3f",
			frameMD5:     []string{"00e31a71ce397179f111a49ff859faef", "a1ca1454ddcd19006f0d6b84b2971be5", "1bd782e7a7386b39616fa9248ec2327d", "735363c6d23f331ee828c237a8ab1946", "d9b2a282d9bdee1682dba11221ad3fd5"},
			rawVideoMD5:  "ab46f4b665d41c16b5bcee3937cf807b",
		},
		{
			bitDepth:     14,
			name:         "cavlc-partitioned-b8x16-mode2-deblock",
			sourceFile:   "high10_partitioned_implicit_weight_b_deblock_b8x16_cavlc.h264",
			cabac:        0,
			mode2Deblock: true,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			wantSlices:   []int32{1, 2, 3, 3, 2},
			deblock:      []int32{1, 2, 2, 2, 2},
			bitstreamMD5: "28bb90dc22ada25bd9db755845bf1f1f",
			frameMD5:     []string{"00e31a71ce397179f111a49ff859faef", "a1ca1454ddcd19006f0d6b84b2971be5", "1bd782e7a7386b39616fa9248ec2327d", "735363c6d23f331ee828c237a8ab1946", "d9b2a282d9bdee1682dba11221ad3fd5"},
			rawVideoMD5:  "ab46f4b665d41c16b5bcee3937cf807b",
		},
		{
			bitDepth:     14,
			name:         "cabac-partitioned-b8x16-mode1-deblock",
			sourceFile:   "high10_partitioned_implicit_weight_b_deblock_b8x16_cabac.h264",
			cabac:        1,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			wantSlices:   []int32{1, 2, 3, 3, 2},
			deblock:      []int32{1, 1, 1, 1, 1},
			bitstreamMD5: "4067dc4a8250cd21115ad35dfa010c79",
			frameMD5:     []string{"f7e31af362f75c2c72082c963526bd62", "d896c9943347af8118cfd7e8b8e021bd", "bf92b2cd0f34d54e19c5dbe961cb60dd", "6186d1560513c04ea7cddcca9defa484", "1a92c8cee6d17718c047e7346827c480"},
			rawVideoMD5:  "ada175004bdc3976dff2653f07193259",
		},
		{
			bitDepth:     14,
			name:         "cabac-partitioned-b8x16-mode2-deblock",
			sourceFile:   "high10_partitioned_implicit_weight_b_deblock_b8x16_cabac.h264",
			cabac:        1,
			mode2Deblock: true,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			wantSlices:   []int32{1, 2, 3, 3, 2},
			deblock:      []int32{1, 2, 2, 2, 2},
			bitstreamMD5: "c125d66f01173c22b1a35ad17effa36a",
			frameMD5:     []string{"f7e31af362f75c2c72082c963526bd62", "d896c9943347af8118cfd7e8b8e021bd", "bf92b2cd0f34d54e19c5dbe961cb60dd", "6186d1560513c04ea7cddcca9defa484", "1a92c8cee6d17718c047e7346827c480"},
			rawVideoMD5:  "ada175004bdc3976dff2653f07193259",
		},
		{
			bitDepth:     14,
			name:         "cavlc-partitioned-b8x8-mode1-deblock",
			sourceFile:   "high10_partitioned_implicit_weight_b_deblock_b8x8_cavlc.h264",
			cabac:        0,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			wantSlices:   []int32{1, 2, 3, 3, 2},
			deblock:      []int32{1, 1, 1, 1, 1},
			bitstreamMD5: "029173818ddc1e2eca41ec893cbb08f8",
			frameMD5:     []string{"6bcadaf1ef408dfb87ed0eef55afc867", "5a4c0addc6b2d64f71166bcabb840ee5", "573106279c00735121817d6a01206b55", "7659b11af00b55241e074e254fd9b4d2", "c8b2f1da66efe9ef9af551fcc4a45da5"},
			rawVideoMD5:  "e36255f063b5f84c345297150d338255",
		},
		{
			bitDepth:     14,
			name:         "cavlc-partitioned-b8x8-mode2-deblock",
			sourceFile:   "high10_partitioned_implicit_weight_b_deblock_b8x8_cavlc.h264",
			cabac:        0,
			mode2Deblock: true,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			wantSlices:   []int32{1, 2, 3, 3, 2},
			deblock:      []int32{1, 2, 2, 2, 2},
			bitstreamMD5: "eb9871c90386683bdba5e6238d8ac4dc",
			frameMD5:     []string{"6bcadaf1ef408dfb87ed0eef55afc867", "5a4c0addc6b2d64f71166bcabb840ee5", "573106279c00735121817d6a01206b55", "7659b11af00b55241e074e254fd9b4d2", "c8b2f1da66efe9ef9af551fcc4a45da5"},
			rawVideoMD5:  "e36255f063b5f84c345297150d338255",
		},
		{
			bitDepth:     14,
			name:         "cabac-partitioned-b8x8-mode1-deblock",
			sourceFile:   "high10_partitioned_implicit_weight_b_deblock_b8x8_cabac.h264",
			cabac:        1,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			wantSlices:   []int32{1, 2, 3, 3, 2},
			deblock:      []int32{1, 1, 1, 1, 1},
			bitstreamMD5: "1fdc91718b5eb7c85d4d69365755eb5f",
			frameMD5:     []string{"6bcadaf1ef408dfb87ed0eef55afc867", "85cc821849b33793ba4a35a3d99e6428", "128c03636c12e12064a93cb6fab8350c", "7659b11af00b55241e074e254fd9b4d2", "068d4c19adf6515d32e3448f379fa2d6"},
			rawVideoMD5:  "db18498b0ec34aeb542bd2705fa0a700",
		},
		{
			bitDepth:     14,
			name:         "cabac-partitioned-b8x8-mode2-deblock",
			sourceFile:   "high10_partitioned_implicit_weight_b_deblock_b8x8_cabac.h264",
			cabac:        1,
			mode2Deblock: true,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			wantSlices:   []int32{1, 2, 3, 3, 2},
			deblock:      []int32{1, 2, 2, 2, 2},
			bitstreamMD5: "eb29044ba918856c0da7ae0c134742cd",
			frameMD5:     []string{"6bcadaf1ef408dfb87ed0eef55afc867", "85cc821849b33793ba4a35a3d99e6428", "128c03636c12e12064a93cb6fab8350c", "7659b11af00b55241e074e254fd9b4d2", "068d4c19adf6515d32e3448f379fa2d6"},
			rawVideoMD5:  "db18498b0ec34aeb542bd2705fa0a700",
		},
	}
}
