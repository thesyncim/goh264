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

type highCAVLCBCase struct {
	bitDepth              int
	name                  string
	sourceFile            string
	mode2Deblock          bool
	width                 int
	height                int
	rawFrameSize          int
	direct8x8             int32
	ppsRefCount           [2]uint32
	wantSlices            []int32
	deblock               []int32
	direct                []int32
	configuredFrameCounts []int
	bitstreamMD5          string
	frameMD5              []string
	rawVideoMD5           string
}

func TestHighCAVLCBFixtureSyntax(t *testing.T) {
	for _, tt := range highCAVLCBCases() {
		t.Run(highCAVLCBCaseName(tt), func(t *testing.T) {
			data := highCAVLCBFixture(t, tt)
			assertHighCAVLCBFixtureSyntax(t, data, tt)
		})
	}
}

func TestDecodeAnnexBHighCAVLCBFrames(t *testing.T) {
	for _, tt := range highCAVLCBCases() {
		t.Run(highCAVLCBCaseName(tt), func(t *testing.T) {
			data := highCAVLCBFixture(t, tt)
			assertHighCAVLCBFixtureSyntax(t, data, tt)

			frames, err := NewDecoder().DecodeAnnexBFrames(data)
			if err != nil {
				t.Fatalf("DecodeAnnexBFrames: %v", err)
			}
			assertHighCAVLCBFrames(t, frames, tt)
		})
	}
}

func TestDecodeAVCHighCAVLCBFrames(t *testing.T) {
	for _, tt := range highCAVLCBCases() {
		t.Run(highCAVLCBCaseName(tt), func(t *testing.T) {
			data := highCAVLCBFixture(t, tt)
			assertHighCAVLCBFixtureSyntax(t, data, tt)

			for _, nalLengthSize := range []int{2, 3, 4} {
				frames, err := NewDecoder().DecodeAVCFrames(annexBToAVC(t, data, nalLengthSize), nalLengthSize)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: DecodeAVCFrames: %v", nalLengthSize, err)
				}
				assertHighCAVLCBFrames(t, frames, tt)
			}
		})
	}
}

func TestDecodeConfiguredAVCHighCAVLCBFramesAcrossSamplesFlush(t *testing.T) {
	for _, tt := range highCAVLCBCases() {
		t.Run(highCAVLCBCaseName(tt), func(t *testing.T) {
			data := highCAVLCBFixture(t, tt)
			assertHighCAVLCBFixtureSyntax(t, data, tt)

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
				assertHighCAVLCBConfiguredSampleCounts(t, tt, nalLengthSize, frameCounts)
				frames = append(frames, out...)
				assertHighCAVLCBFrames(t, frames, tt)
			}
		})
	}
}

func TestFFmpegRawVideoFrameMD5OracleHighCAVLCB(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}

	for _, tt := range highCAVLCBCases() {
		t.Run(highCAVLCBCaseName(tt), func(t *testing.T) {
			data := highCAVLCBFixture(t, tt)
			assertHighCAVLCBFixtureSyntax(t, data, tt)
			assertFFmpegHighCAVLCBRawVideoOracle(t, data, tt)
		})
	}
}

func highCAVLCBFixture(t *testing.T, tt highCAVLCBCase) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "h264", tt.sourceFile))
	if err != nil {
		t.Fatalf("read %s: %v", tt.sourceFile, err)
	}
	out := highCAVLCBRewriteAnnexB(t, data, tt.bitDepth, tt.mode2Deblock)
	sum := md5.Sum(out)
	if got := hex.EncodeToString(sum[:]); got != tt.bitstreamMD5 {
		t.Fatalf("High%d CAVLC B generated bitstream md5 = %s, want %s", tt.bitDepth, got, tt.bitstreamMD5)
	}
	return out
}

func highCAVLCBRewriteAnnexB(t *testing.T, data []byte, bitDepth int, mode2Deblock bool) []byte {
	t.Helper()
	start, prefixLen, ok := high14CABACBFindStartCode(data, 0)
	if !ok {
		t.Fatal("source fixture has no Annex B start code")
	}
	var out []byte
	var spsList [32]*h264.SPS
	var ppsList [256]*h264.PPS
	for ok {
		nalStart := start + prefixLen
		nextStart, nextPrefixLen, nextOK := high14CABACBFindStartCode(data, nalStart)
		nalEnd := len(data)
		if nextOK {
			nalEnd = nextStart
		}
		if nalEnd > nalStart {
			out = append(out, data[start:nalStart]...)
			raw := append([]byte(nil), data[nalStart:nalEnd]...)
			nalType := h264.NALUnitType(raw[0] & 0x1f)
			rbsp := high14CABACBEBSPToRBSP(raw[1:])
			switch nalType {
			case h264.NALSPS:
				sps, err := h264.DecodeSPS(rbsp)
				if err != nil {
					t.Fatalf("decode source SPS: %v", err)
				}
				spsList[sps.SPSID] = sps
				raw = highCABACBRewriteSPSRaw(t, raw, bitDepth)
			case h264.NALPPS:
				pps, err := h264.DecodePPS(rbsp, &spsList)
				if err != nil {
					t.Fatalf("decode source PPS: %v", err)
				}
				ppsList[pps.PPSID] = pps
			case h264.NALSlice, h264.NALIDRSlice:
				nal := h264.NALUnit{RefIDC: raw[0] >> 5 & 0x03, Type: nalType, Raw: raw, RBSP: rbsp}
				sh, err := h264.ParseSliceHeader(nal, &ppsList)
				if err != nil {
					t.Fatalf("parse source slice: %v", err)
				}
				if mode2Deblock && nalType == h264.NALSlice && sh.PPS.CABAC == 0 && sh.DeblockingFilter == 1 {
					raw = highCAVLCBRewriteSliceDeblockMode(t, raw, sh)
					rbsp = high14CABACBEBSPToRBSP(raw[1:])
					nal.RBSP = rbsp
					nal.Raw = raw
					got, err := h264.ParseSliceHeader(nal, &ppsList)
					if err != nil {
						t.Fatalf("parse mode-2 rewritten slice: %v", err)
					}
					if got.DeblockingFilter != 2 {
						t.Fatalf("rewritten slice deblock = %d, want mode-2", got.DeblockingFilter)
					}
				}
			}
			out = append(out, raw...)
		}
		if !nextOK {
			break
		}
		start, prefixLen, ok = nextStart, nextPrefixLen, true
	}
	return out
}

func highCAVLCBRewriteSliceDeblockMode(t *testing.T, raw []byte, sh *h264.SliceHeader) []byte {
	t.Helper()
	if len(raw) < 2 || sh == nil || sh.PPS == nil || sh.SPS == nil {
		t.Fatal("invalid slice rewrite input")
	}
	rbsp := high14CABACBEBSPToRBSP(raw[1:])
	bits := high14CABACBBits(rbsp)
	disableStart, disableEnd, _ := highCABACBSliceDeblockRange(t, bits, sh)
	oldBits := bits[disableStart:disableEnd]
	if oldBits != high14CABACBUEBits(0) {
		t.Fatalf("CAVLC deblock bits = %q, want mode-1", oldBits)
	}
	rewritten := bits[:disableStart] + high14CABACBUEBits(2) + bits[disableEnd:]
	if mod := len(rewritten) % 8; mod != 0 {
		rewritten += strings.Repeat("0", 8-mod)
	}
	rbsp = high14CABACBPackWholeBits(t, rewritten)
	return append([]byte{raw[0]}, high14CABACBRBSPToEBSP(rbsp)...)
}

func highCAVLCBCaseName(tt highCAVLCBCase) string {
	return fmt.Sprintf("high%d-%s", tt.bitDepth, tt.name)
}

func assertHighCAVLCBFixtureSyntax(t *testing.T, data []byte, tt highCAVLCBCase) {
	t.Helper()
	nals, err := h264.SplitAnnexB(data)
	if err != nil {
		t.Fatal(err)
	}
	var spsList [32]*h264.SPS
	var ppsList [256]*h264.PPS
	var gotSlices []int32
	var gotDeblock []int32
	var gotDirect []int32
	for i, nal := range nals {
		switch nal.Type {
		case h264.NALSPS:
			sps, err := h264.DecodeSPS(nal.RBSP)
			if err != nil {
				t.Fatal(err)
			}
			if i != 0 || sps.ProfileIDC != 244 || sps.Width != int32(tt.width) || sps.Height != int32(tt.height) ||
				sps.ChromaFormatIDC != 1 || sps.BitDepthLuma != int32(tt.bitDepth) || sps.BitDepthChroma != int32(tt.bitDepth) ||
				sps.FrameMBSOnlyFlag != 1 || sps.MBAFF != 0 || sps.RefFrameCount != 2 ||
				int32(sps.Direct8x8InferenceFlag) != tt.direct8x8 {
				t.Fatalf("SPS = nal[%d] profile %d %dx%d chroma %d depth %d/%d frameonly/mbaff %d/%d refs %d direct8x8 %d, want High%d 4:2:0 frame-only refs=2 direct8x8=%d",
					i, sps.ProfileIDC, sps.Width, sps.Height, sps.ChromaFormatIDC, sps.BitDepthLuma, sps.BitDepthChroma,
					sps.FrameMBSOnlyFlag, sps.MBAFF, sps.RefFrameCount, sps.Direct8x8InferenceFlag, tt.bitDepth, tt.direct8x8)
			}
			spsList[sps.SPSID] = sps
		case h264.NALPPS:
			pps, err := h264.DecodePPS(nal.RBSP, &spsList)
			if err != nil {
				t.Fatal(err)
			}
			if i != 1 || pps.CABAC != 0 || pps.Transform8x8Mode != 0 || pps.WeightedPred != 0 ||
				pps.WeightedBipredIDC != 0 || pps.RefCount != tt.ppsRefCount {
				t.Fatalf("PPS = nal[%d] cabac/8x8/weights/refs = %d/%d/%d/%d/%v, want CAVLC/no-8x8/unweighted refs=%v",
					i, pps.CABAC, pps.Transform8x8Mode, pps.WeightedPred, pps.WeightedBipredIDC, pps.RefCount, tt.ppsRefCount)
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
				if sh.ListCount != 0 || sh.RefCount != ([2]uint32{}) {
					t.Fatalf("I slice lists/refs = %d/%v, want none", sh.ListCount, sh.RefCount)
				}
			case h264.PictureTypeP:
				if sh.ListCount != 1 || sh.RefCount[0] != 1 ||
					sh.PredWeightTable.UseWeight != 0 || sh.PredWeightTable.UseWeightChroma != 0 {
					t.Fatalf("P slice lists/refs/weights = %d/%v/%d/%d, want L0 refs=1 unweighted",
						sh.ListCount, sh.RefCount, sh.PredWeightTable.UseWeight, sh.PredWeightTable.UseWeightChroma)
				}
			case h264.PictureTypeB:
				if sh.ListCount != 2 || sh.RefCount[0] != 1 || sh.RefCount[1] != 1 ||
					sh.PredWeightTable.UseWeight != 0 || sh.PredWeightTable.UseWeightChroma != 0 {
					t.Fatalf("B slice lists/refs/weights = %d/%v/%d/%d, want L0/L1 refs=1/1 unweighted",
						sh.ListCount, sh.RefCount, sh.PredWeightTable.UseWeight, sh.PredWeightTable.UseWeightChroma)
				}
			default:
				t.Fatalf("unexpected slice type %d", sh.SliceTypeNoS)
			}
			gotSlices = append(gotSlices, sh.SliceTypeNoS)
			gotDeblock = append(gotDeblock, sh.DeblockingFilter)
			gotDirect = append(gotDirect, sh.DirectSpatialMVPred)
		case h264.NALSEI:
		default:
			t.Fatalf("unexpected NAL type %d", nal.Type)
		}
	}
	assertHighCAVLCBInt32Slice(t, "slice types", gotSlices, tt.wantSlices)
	assertHighCAVLCBInt32Slice(t, "deblock modes", gotDeblock, tt.deblock)
	assertHighCAVLCBInt32Slice(t, "direct modes", gotDirect, tt.direct)
}

func assertHighCAVLCBInt32Slice(t *testing.T, name string, got []int32, want []int32) {
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

func assertHighCAVLCBConfiguredSampleCounts(t *testing.T, tt highCAVLCBCase, nalLengthSize int, got []int) {
	t.Helper()
	if len(got) != len(tt.configuredFrameCounts) {
		t.Fatalf("%s nalLengthSize=%d configured sample/flush counts = %v, want %v",
			tt.name, nalLengthSize, got, tt.configuredFrameCounts)
	}
	for i := range tt.configuredFrameCounts {
		if got[i] != tt.configuredFrameCounts[i] {
			t.Fatalf("%s nalLengthSize=%d configured sample/flush counts = %v, want %v",
				tt.name, nalLengthSize, got, tt.configuredFrameCounts)
		}
	}
}

func assertHighCAVLCBFrames(t *testing.T, frames []*Frame, tt highCAVLCBCase) {
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
		if frame.Width != tt.width || frame.Height != tt.height ||
			frame.ChromaFormatIDC != 1 || frame.BitDepthLuma != tt.bitDepth || frame.BitDepthChroma != tt.bitDepth {
			t.Fatalf("frame[%d] format = %dx%d chroma %d depth %d/%d, want %dx%d %s",
				i, frame.Width, frame.Height, frame.ChromaFormatIDC, frame.BitDepthLuma, frame.BitDepthChroma, tt.width, tt.height, wantPixFmt)
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

func assertFFmpegHighCAVLCBRawVideoOracle(t *testing.T, data []byte, tt highCAVLCBCase) {
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

func highCAVLCBCases() []highCAVLCBCase {
	return []highCAVLCBCase{
		{
			bitDepth:              12,
			name:                  "nondirect-mode1-deblock",
			sourceFile:            "high10_b_deblock_cavlc.h264",
			width:                 16,
			height:                16,
			rawFrameSize:          768,
			direct8x8:             1,
			ppsRefCount:           [2]uint32{2, 1},
			wantSlices:            []int32{1, 2, 3},
			deblock:               []int32{0, 1, 1},
			direct:                []int32{0, 0, 0},
			configuredFrameCounts: []int{0, 1, 1, 1},
			bitstreamMD5:          "401f7f2e84b754180511da29f696a361",
			frameMD5: []string{
				"2271a65b978f0140735fd4b887c4eac9",
				"fe6386ebbdc5aeee71012dd68933a44f",
				"17296bd53e49cc810ebf19315bbf50ae",
			},
			rawVideoMD5: "aeacc0c24f5ab957bf6ca6a425d8fa56",
		},
		{
			bitDepth:              12,
			name:                  "nondirect-mode2-deblock",
			sourceFile:            "high10_b_deblock_cavlc.h264",
			mode2Deblock:          true,
			width:                 16,
			height:                16,
			rawFrameSize:          768,
			direct8x8:             1,
			ppsRefCount:           [2]uint32{2, 1},
			wantSlices:            []int32{1, 2, 3},
			deblock:               []int32{0, 2, 2},
			direct:                []int32{0, 0, 0},
			configuredFrameCounts: []int{0, 1, 1, 1},
			bitstreamMD5:          "d12355ad0c72383fbe7e02e48b85805c",
			frameMD5: []string{
				"2271a65b978f0140735fd4b887c4eac9",
				"fe6386ebbdc5aeee71012dd68933a44f",
				"17296bd53e49cc810ebf19315bbf50ae",
			},
			rawVideoMD5: "aeacc0c24f5ab957bf6ca6a425d8fa56",
		},
		{
			bitDepth:              12,
			name:                  "temporal-direct-mode1-deblock",
			sourceFile:            "high10_direct_b_deblock_temporal_cavlc.h264",
			width:                 32,
			height:                16,
			rawFrameSize:          1536,
			direct8x8:             1,
			ppsRefCount:           [2]uint32{2, 1},
			wantSlices:            []int32{1, 2, 3},
			deblock:               []int32{1, 1, 1},
			direct:                []int32{0, 0, 0},
			configuredFrameCounts: []int{0, 1, 1, 1},
			bitstreamMD5:          "15dd8064918fed78680191f9ac033092",
			frameMD5: []string{
				"73eb53ed4cb2a4844c155a97fe84a06e",
				"f3600ce13fee23f7110d2e713961dfc4",
				"579ee4696815eaca069d46f026aab846",
			},
			rawVideoMD5: "577f5505d06994c30ea5d02488eda8c3",
		},
		{
			bitDepth:              12,
			name:                  "temporal-direct-mode2-deblock",
			sourceFile:            "high10_direct_b_deblock_temporal_cavlc.h264",
			mode2Deblock:          true,
			width:                 32,
			height:                16,
			rawFrameSize:          1536,
			direct8x8:             1,
			ppsRefCount:           [2]uint32{2, 1},
			wantSlices:            []int32{1, 2, 3},
			deblock:               []int32{1, 2, 2},
			direct:                []int32{0, 0, 0},
			configuredFrameCounts: []int{0, 1, 1, 1},
			bitstreamMD5:          "e9bcc26f25936f11d7f498102d23bf3b",
			frameMD5: []string{
				"73eb53ed4cb2a4844c155a97fe84a06e",
				"f3600ce13fee23f7110d2e713961dfc4",
				"579ee4696815eaca069d46f026aab846",
			},
			rawVideoMD5: "577f5505d06994c30ea5d02488eda8c3",
		},
		{
			bitDepth:              12,
			name:                  "spatial-direct-mode1-deblock",
			sourceFile:            "high10_direct_b_deblock_spatial_cavlc.h264",
			width:                 32,
			height:                16,
			rawFrameSize:          1536,
			direct8x8:             1,
			ppsRefCount:           [2]uint32{2, 1},
			wantSlices:            []int32{1, 2, 3},
			deblock:               []int32{1, 1, 1},
			direct:                []int32{0, 0, 1},
			configuredFrameCounts: []int{0, 1, 1, 1},
			bitstreamMD5:          "cd8f78ed808fb80ff3a9fb1b589c4a22",
			frameMD5: []string{
				"73eb53ed4cb2a4844c155a97fe84a06e",
				"f3600ce13fee23f7110d2e713961dfc4",
				"579ee4696815eaca069d46f026aab846",
			},
			rawVideoMD5: "577f5505d06994c30ea5d02488eda8c3",
		},
		{
			bitDepth:              12,
			name:                  "spatial-direct-mode2-deblock",
			sourceFile:            "high10_direct_b_deblock_spatial_cavlc.h264",
			mode2Deblock:          true,
			width:                 32,
			height:                16,
			rawFrameSize:          1536,
			direct8x8:             1,
			ppsRefCount:           [2]uint32{2, 1},
			wantSlices:            []int32{1, 2, 3},
			deblock:               []int32{1, 2, 2},
			direct:                []int32{0, 0, 1},
			configuredFrameCounts: []int{0, 1, 1, 1},
			bitstreamMD5:          "0779d169ee8d141f402590b83a252c0b",
			frameMD5: []string{
				"73eb53ed4cb2a4844c155a97fe84a06e",
				"f3600ce13fee23f7110d2e713961dfc4",
				"579ee4696815eaca069d46f026aab846",
			},
			rawVideoMD5: "577f5505d06994c30ea5d02488eda8c3",
		},
		{
			bitDepth:              12,
			name:                  "temporal-bskip-mode1-deblock",
			sourceFile:            "high10_bskip_deblock_temporal_cavlc.h264",
			width:                 16,
			height:                16,
			rawFrameSize:          768,
			direct8x8:             1,
			ppsRefCount:           [2]uint32{2, 1},
			wantSlices:            []int32{1, 2, 3},
			deblock:               []int32{0, 1, 1},
			direct:                []int32{0, 0, 0},
			configuredFrameCounts: []int{0, 1, 1, 1},
			bitstreamMD5:          "85c05d941ab25fb67f8e95c3bb2868d5",
			frameMD5: []string{
				"d4753b9733af2865470fb72f96a37071",
				"d4753b9733af2865470fb72f96a37071",
				"d4753b9733af2865470fb72f96a37071",
			},
			rawVideoMD5: "c5844f8a45006553335c482758ad0f49",
		},
		{
			bitDepth:              12,
			name:                  "temporal-bskip-mode2-deblock",
			sourceFile:            "high10_bskip_deblock_temporal_cavlc.h264",
			mode2Deblock:          true,
			width:                 16,
			height:                16,
			rawFrameSize:          768,
			direct8x8:             1,
			ppsRefCount:           [2]uint32{2, 1},
			wantSlices:            []int32{1, 2, 3},
			deblock:               []int32{0, 2, 2},
			direct:                []int32{0, 0, 0},
			configuredFrameCounts: []int{0, 1, 1, 1},
			bitstreamMD5:          "7437e1d9b2df1084ed3cf0c6accc8fa7",
			frameMD5: []string{
				"d4753b9733af2865470fb72f96a37071",
				"d4753b9733af2865470fb72f96a37071",
				"d4753b9733af2865470fb72f96a37071",
			},
			rawVideoMD5: "c5844f8a45006553335c482758ad0f49",
		},
		{
			bitDepth:              12,
			name:                  "spatial-bskip-mode1-deblock",
			sourceFile:            "high10_bskip_deblock_spatial_cavlc.h264",
			width:                 16,
			height:                16,
			rawFrameSize:          768,
			direct8x8:             1,
			ppsRefCount:           [2]uint32{2, 1},
			wantSlices:            []int32{1, 2, 3},
			deblock:               []int32{0, 1, 1},
			direct:                []int32{0, 0, 1},
			configuredFrameCounts: []int{0, 1, 1, 1},
			bitstreamMD5:          "5b9a275b44704ac2728fdb792840b5fe",
			frameMD5: []string{
				"d4753b9733af2865470fb72f96a37071",
				"d4753b9733af2865470fb72f96a37071",
				"d4753b9733af2865470fb72f96a37071",
			},
			rawVideoMD5: "c5844f8a45006553335c482758ad0f49",
		},
		{
			bitDepth:              12,
			name:                  "spatial-bskip-mode2-deblock",
			sourceFile:            "high10_bskip_deblock_spatial_cavlc.h264",
			mode2Deblock:          true,
			width:                 16,
			height:                16,
			rawFrameSize:          768,
			direct8x8:             1,
			ppsRefCount:           [2]uint32{2, 1},
			wantSlices:            []int32{1, 2, 3},
			deblock:               []int32{0, 2, 2},
			direct:                []int32{0, 0, 1},
			configuredFrameCounts: []int{0, 1, 1, 1},
			bitstreamMD5:          "a36146dacf1b1ba1693178b9f1a10623",
			frameMD5: []string{
				"d4753b9733af2865470fb72f96a37071",
				"d4753b9733af2865470fb72f96a37071",
				"d4753b9733af2865470fb72f96a37071",
			},
			rawVideoMD5: "c5844f8a45006553335c482758ad0f49",
		},
		{
			bitDepth:              12,
			name:                  "direct-sub-b8x8-temporal-mode1-deblock",
			sourceFile:            "high10_cavlc_b8x8_temporal_direct_sub_deblock.h264",
			width:                 16,
			height:                16,
			rawFrameSize:          768,
			direct8x8:             1,
			ppsRefCount:           [2]uint32{2, 1},
			wantSlices:            []int32{1, 2, 3},
			deblock:               []int32{0, 1, 1},
			direct:                []int32{0, 0, 0},
			configuredFrameCounts: []int{0, 1, 1, 1},
			bitstreamMD5:          "ddaa60da4e51d368bfef77630df5e281",
			frameMD5: []string{
				"d4753b9733af2865470fb72f96a37071",
				"d4753b9733af2865470fb72f96a37071",
				"d4753b9733af2865470fb72f96a37071",
			},
			rawVideoMD5: "c5844f8a45006553335c482758ad0f49",
		},
		{
			bitDepth:              12,
			name:                  "direct-sub-b8x8-temporal-mode2-deblock",
			sourceFile:            "high10_cavlc_b8x8_temporal_direct_sub_deblock.h264",
			mode2Deblock:          true,
			width:                 16,
			height:                16,
			rawFrameSize:          768,
			direct8x8:             1,
			ppsRefCount:           [2]uint32{2, 1},
			wantSlices:            []int32{1, 2, 3},
			deblock:               []int32{0, 2, 2},
			direct:                []int32{0, 0, 0},
			configuredFrameCounts: []int{0, 1, 1, 1},
			bitstreamMD5:          "73097b3478a59e4725d72c1c79b8542f",
			frameMD5: []string{
				"d4753b9733af2865470fb72f96a37071",
				"d4753b9733af2865470fb72f96a37071",
				"d4753b9733af2865470fb72f96a37071",
			},
			rawVideoMD5: "c5844f8a45006553335c482758ad0f49",
		},
		{
			bitDepth:              12,
			name:                  "direct-sub-b8x8-spatial-mode1-deblock",
			sourceFile:            "high10_cavlc_b8x8_spatial_direct_sub_deblock.h264",
			width:                 16,
			height:                16,
			rawFrameSize:          768,
			direct8x8:             1,
			ppsRefCount:           [2]uint32{2, 1},
			wantSlices:            []int32{1, 2, 3},
			deblock:               []int32{0, 1, 1},
			direct:                []int32{0, 0, 1},
			configuredFrameCounts: []int{0, 1, 1, 1},
			bitstreamMD5:          "f712786914d94d80d2b88dad6472a789",
			frameMD5: []string{
				"d4753b9733af2865470fb72f96a37071",
				"d4753b9733af2865470fb72f96a37071",
				"d4753b9733af2865470fb72f96a37071",
			},
			rawVideoMD5: "c5844f8a45006553335c482758ad0f49",
		},
		{
			bitDepth:              12,
			name:                  "direct-sub-b8x8-spatial-mode2-deblock",
			sourceFile:            "high10_cavlc_b8x8_spatial_direct_sub_deblock.h264",
			mode2Deblock:          true,
			width:                 16,
			height:                16,
			rawFrameSize:          768,
			direct8x8:             1,
			ppsRefCount:           [2]uint32{2, 1},
			wantSlices:            []int32{1, 2, 3},
			deblock:               []int32{0, 2, 2},
			direct:                []int32{0, 0, 1},
			configuredFrameCounts: []int{0, 1, 1, 1},
			bitstreamMD5:          "c7d002bc59f199c531dc47e1a2f0a9b3",
			frameMD5: []string{
				"d4753b9733af2865470fb72f96a37071",
				"d4753b9733af2865470fb72f96a37071",
				"d4753b9733af2865470fb72f96a37071",
			},
			rawVideoMD5: "c5844f8a45006553335c482758ad0f49",
		},
		{
			bitDepth:              12,
			name:                  "direct-sub-b4x4-temporal-mode1-deblock",
			sourceFile:            "high10_cavlc_b4x4_temporal_direct_sub_deblock.h264",
			width:                 16,
			height:                16,
			rawFrameSize:          768,
			direct8x8:             0,
			ppsRefCount:           [2]uint32{2, 1},
			wantSlices:            []int32{1, 2, 3},
			deblock:               []int32{0, 1, 1},
			direct:                []int32{0, 0, 0},
			configuredFrameCounts: []int{0, 1, 1, 1},
			bitstreamMD5:          "c0b0b020719633be9071cf864a51b231",
			frameMD5: []string{
				"d4753b9733af2865470fb72f96a37071",
				"d4753b9733af2865470fb72f96a37071",
				"d4753b9733af2865470fb72f96a37071",
			},
			rawVideoMD5: "c5844f8a45006553335c482758ad0f49",
		},
		{
			bitDepth:              12,
			name:                  "direct-sub-b4x4-temporal-mode2-deblock",
			sourceFile:            "high10_cavlc_b4x4_temporal_direct_sub_deblock.h264",
			mode2Deblock:          true,
			width:                 16,
			height:                16,
			rawFrameSize:          768,
			direct8x8:             0,
			ppsRefCount:           [2]uint32{2, 1},
			wantSlices:            []int32{1, 2, 3},
			deblock:               []int32{0, 2, 2},
			direct:                []int32{0, 0, 0},
			configuredFrameCounts: []int{0, 1, 1, 1},
			bitstreamMD5:          "bcdae5424d775d9627560546422d33ca",
			frameMD5: []string{
				"d4753b9733af2865470fb72f96a37071",
				"d4753b9733af2865470fb72f96a37071",
				"d4753b9733af2865470fb72f96a37071",
			},
			rawVideoMD5: "c5844f8a45006553335c482758ad0f49",
		},
		{
			bitDepth:              12,
			name:                  "direct-sub-b4x4-spatial-mode1-deblock",
			sourceFile:            "high10_cavlc_b4x4_spatial_direct_sub_deblock.h264",
			width:                 16,
			height:                16,
			rawFrameSize:          768,
			direct8x8:             0,
			ppsRefCount:           [2]uint32{2, 1},
			wantSlices:            []int32{1, 2, 3},
			deblock:               []int32{0, 1, 1},
			direct:                []int32{0, 0, 1},
			configuredFrameCounts: []int{0, 1, 1, 1},
			bitstreamMD5:          "8f01e8318928f05729095536d0c97806",
			frameMD5: []string{
				"d4753b9733af2865470fb72f96a37071",
				"d4753b9733af2865470fb72f96a37071",
				"d4753b9733af2865470fb72f96a37071",
			},
			rawVideoMD5: "c5844f8a45006553335c482758ad0f49",
		},
		{
			bitDepth:              12,
			name:                  "direct-sub-b4x4-spatial-mode2-deblock",
			sourceFile:            "high10_cavlc_b4x4_spatial_direct_sub_deblock.h264",
			mode2Deblock:          true,
			width:                 16,
			height:                16,
			rawFrameSize:          768,
			direct8x8:             0,
			ppsRefCount:           [2]uint32{2, 1},
			wantSlices:            []int32{1, 2, 3},
			deblock:               []int32{0, 2, 2},
			direct:                []int32{0, 0, 1},
			configuredFrameCounts: []int{0, 1, 1, 1},
			bitstreamMD5:          "2d77861f7867269111c59fc5048ef88f",
			frameMD5: []string{
				"d4753b9733af2865470fb72f96a37071",
				"d4753b9733af2865470fb72f96a37071",
				"d4753b9733af2865470fb72f96a37071",
			},
			rawVideoMD5: "c5844f8a45006553335c482758ad0f49",
		},
		{
			bitDepth:              12,
			name:                  "partitioned-b16x8-mode1-deblock",
			sourceFile:            "high10_partitioned_b_deblock_b16x8_cavlc.h264",
			width:                 16,
			height:                16,
			rawFrameSize:          768,
			direct8x8:             1,
			ppsRefCount:           [2]uint32{1, 1},
			wantSlices:            []int32{1, 2, 3, 3, 2},
			deblock:               []int32{1, 1, 1, 1, 1},
			direct:                []int32{0, 0, 0, 0, 0},
			configuredFrameCounts: []int{0, 1, 1, 1, 1, 1},
			bitstreamMD5:          "df6a2ceaf6945033f4c364d54337454a",
			frameMD5: []string{
				"271fc9eab6e0d7c98af20f9ecffd0491",
				"0d3c4890ccc080bfec51093276bc8738",
				"c4944df57076716dcb00e10b2a2dbdbb",
				"11bc3a789dd0b701afa7e4e9e5c137c9",
				"8bbd70c55f2113ce370f0a5c96b0ac09",
			},
			rawVideoMD5: "3f313eb2ae836bae50a4950be5aa2f56",
		},
		{
			bitDepth:              12,
			name:                  "partitioned-b16x8-mode2-deblock",
			sourceFile:            "high10_partitioned_b_deblock_b16x8_cavlc.h264",
			mode2Deblock:          true,
			width:                 16,
			height:                16,
			rawFrameSize:          768,
			direct8x8:             1,
			ppsRefCount:           [2]uint32{1, 1},
			wantSlices:            []int32{1, 2, 3, 3, 2},
			deblock:               []int32{1, 2, 2, 2, 2},
			direct:                []int32{0, 0, 0, 0, 0},
			configuredFrameCounts: []int{0, 1, 1, 1, 1, 1},
			bitstreamMD5:          "6d3200a31715487f64fa681cf04db470",
			frameMD5: []string{
				"271fc9eab6e0d7c98af20f9ecffd0491",
				"0d3c4890ccc080bfec51093276bc8738",
				"c4944df57076716dcb00e10b2a2dbdbb",
				"11bc3a789dd0b701afa7e4e9e5c137c9",
				"8bbd70c55f2113ce370f0a5c96b0ac09",
			},
			rawVideoMD5: "3f313eb2ae836bae50a4950be5aa2f56",
		},
		{
			bitDepth:              12,
			name:                  "partitioned-b8x16-mode1-deblock",
			sourceFile:            "high10_partitioned_b_deblock_b8x16_cavlc.h264",
			width:                 16,
			height:                16,
			rawFrameSize:          768,
			direct8x8:             1,
			ppsRefCount:           [2]uint32{1, 1},
			wantSlices:            []int32{1, 2, 3, 3, 2},
			deblock:               []int32{1, 1, 1, 1, 1},
			direct:                []int32{0, 0, 0, 0, 0},
			configuredFrameCounts: []int{0, 1, 1, 1, 1, 1},
			bitstreamMD5:          "d2d0ef290163ba06348874754ec79578",
			frameMD5: []string{
				"ff5aaa5c613fc0046f4224b7b27b68b6",
				"e03c89cbb3f6fd8a3ccf37f9f11fd17b",
				"d70ec9e8e288c2fbe8626406465bc680",
				"3740befb8f4f876023e6c209d55b56e1",
				"f68edd1b4866284784aeb99943da8b4e",
			},
			rawVideoMD5: "a817b89e01744a52e0f3e0813a9305ee",
		},
		{
			bitDepth:              12,
			name:                  "partitioned-b8x16-mode2-deblock",
			sourceFile:            "high10_partitioned_b_deblock_b8x16_cavlc.h264",
			mode2Deblock:          true,
			width:                 16,
			height:                16,
			rawFrameSize:          768,
			direct8x8:             1,
			ppsRefCount:           [2]uint32{1, 1},
			wantSlices:            []int32{1, 2, 3, 3, 2},
			deblock:               []int32{1, 2, 2, 2, 2},
			direct:                []int32{0, 0, 0, 0, 0},
			configuredFrameCounts: []int{0, 1, 1, 1, 1, 1},
			bitstreamMD5:          "1e48c07200fdb011a149cd480dd039ec",
			frameMD5: []string{
				"ff5aaa5c613fc0046f4224b7b27b68b6",
				"e03c89cbb3f6fd8a3ccf37f9f11fd17b",
				"d70ec9e8e288c2fbe8626406465bc680",
				"3740befb8f4f876023e6c209d55b56e1",
				"f68edd1b4866284784aeb99943da8b4e",
			},
			rawVideoMD5: "a817b89e01744a52e0f3e0813a9305ee",
		},
		{
			bitDepth:              12,
			name:                  "partitioned-b8x8-mode1-deblock",
			sourceFile:            "high10_partitioned_b_deblock_b8x8_cavlc.h264",
			width:                 16,
			height:                16,
			rawFrameSize:          768,
			direct8x8:             1,
			ppsRefCount:           [2]uint32{1, 1},
			wantSlices:            []int32{1, 2, 3, 3, 2},
			deblock:               []int32{1, 1, 1, 1, 1},
			direct:                []int32{0, 0, 0, 0, 0},
			configuredFrameCounts: []int{0, 1, 1, 1, 1, 1},
			bitstreamMD5:          "c15c7e9adbc81c3a67d7084e26b14050",
			frameMD5: []string{
				"69c7144d64f8fcc0994be3f0cfbe4b5d",
				"803a5521723da012f826afe1d0fc9fb5",
				"a6416bf2afb77b2233174057ae89a51d",
				"a170c75ad560003e135f037b8df24609",
				"68dddd659051cf9922494b235f4bb5d7",
			},
			rawVideoMD5: "c80c311bb07b52acb054e3df8e03c244",
		},
		{
			bitDepth:              12,
			name:                  "partitioned-b8x8-mode2-deblock",
			sourceFile:            "high10_partitioned_b_deblock_b8x8_cavlc.h264",
			mode2Deblock:          true,
			width:                 16,
			height:                16,
			rawFrameSize:          768,
			direct8x8:             1,
			ppsRefCount:           [2]uint32{1, 1},
			wantSlices:            []int32{1, 2, 3, 3, 2},
			deblock:               []int32{1, 2, 2, 2, 2},
			direct:                []int32{0, 0, 0, 0, 0},
			configuredFrameCounts: []int{0, 1, 1, 1, 1, 1},
			bitstreamMD5:          "fb8c24030652b70cfff39bbdc473e020",
			frameMD5: []string{
				"69c7144d64f8fcc0994be3f0cfbe4b5d",
				"803a5521723da012f826afe1d0fc9fb5",
				"a6416bf2afb77b2233174057ae89a51d",
				"a170c75ad560003e135f037b8df24609",
				"68dddd659051cf9922494b235f4bb5d7",
			},
			rawVideoMD5: "c80c311bb07b52acb054e3df8e03c244",
		},
		{
			bitDepth:              14,
			name:                  "nondirect-mode1-deblock",
			sourceFile:            "high10_b_deblock_cavlc.h264",
			width:                 16,
			height:                16,
			rawFrameSize:          768,
			direct8x8:             1,
			ppsRefCount:           [2]uint32{2, 1},
			wantSlices:            []int32{1, 2, 3},
			deblock:               []int32{0, 1, 1},
			direct:                []int32{0, 0, 0},
			configuredFrameCounts: []int{0, 1, 1, 1},
			bitstreamMD5:          "8935e9e4900f3227a6a207583bd3d7ce",
			frameMD5: []string{
				"b5c40cef45fdf0a40973472a58c098b9",
				"efb993ac5938d33223e0259fc9a034af",
				"fa5d723fcf39a59ad5330300add86ce9",
			},
			rawVideoMD5: "f24de9bd6af94fad17102001eb2addef",
		},
		{
			bitDepth:              14,
			name:                  "nondirect-mode2-deblock",
			sourceFile:            "high10_b_deblock_cavlc.h264",
			mode2Deblock:          true,
			width:                 16,
			height:                16,
			rawFrameSize:          768,
			direct8x8:             1,
			ppsRefCount:           [2]uint32{2, 1},
			wantSlices:            []int32{1, 2, 3},
			deblock:               []int32{0, 2, 2},
			direct:                []int32{0, 0, 0},
			configuredFrameCounts: []int{0, 1, 1, 1},
			bitstreamMD5:          "b3878b93612bc65720e2081177879ca2",
			frameMD5: []string{
				"b5c40cef45fdf0a40973472a58c098b9",
				"efb993ac5938d33223e0259fc9a034af",
				"fa5d723fcf39a59ad5330300add86ce9",
			},
			rawVideoMD5: "f24de9bd6af94fad17102001eb2addef",
		},
		{
			bitDepth:              14,
			name:                  "temporal-direct-mode1-deblock",
			sourceFile:            "high10_direct_b_deblock_temporal_cavlc.h264",
			width:                 32,
			height:                16,
			rawFrameSize:          1536,
			direct8x8:             1,
			ppsRefCount:           [2]uint32{2, 1},
			wantSlices:            []int32{1, 2, 3},
			deblock:               []int32{1, 1, 1},
			direct:                []int32{0, 0, 0},
			configuredFrameCounts: []int{0, 1, 1, 1},
			bitstreamMD5:          "ec49d9e22dccb7cec848e4cef2e70c74",
			frameMD5: []string{
				"49da495627ee1fa8171085d897fa391b",
				"d8504e1b9ac3d89f271fe308ca4a7352",
				"5487bcebdaf1ad7b5fe17261764e98bd",
			},
			rawVideoMD5: "afbb71db9b9fce02fcfe3bf4137fd250",
		},
		{
			bitDepth:              14,
			name:                  "temporal-direct-mode2-deblock",
			sourceFile:            "high10_direct_b_deblock_temporal_cavlc.h264",
			mode2Deblock:          true,
			width:                 32,
			height:                16,
			rawFrameSize:          1536,
			direct8x8:             1,
			ppsRefCount:           [2]uint32{2, 1},
			wantSlices:            []int32{1, 2, 3},
			deblock:               []int32{1, 2, 2},
			direct:                []int32{0, 0, 0},
			configuredFrameCounts: []int{0, 1, 1, 1},
			bitstreamMD5:          "1a83a6d12dc7173e75ade760f8fd0dd6",
			frameMD5: []string{
				"49da495627ee1fa8171085d897fa391b",
				"d8504e1b9ac3d89f271fe308ca4a7352",
				"5487bcebdaf1ad7b5fe17261764e98bd",
			},
			rawVideoMD5: "afbb71db9b9fce02fcfe3bf4137fd250",
		},
		{
			bitDepth:              14,
			name:                  "spatial-direct-mode1-deblock",
			sourceFile:            "high10_direct_b_deblock_spatial_cavlc.h264",
			width:                 32,
			height:                16,
			rawFrameSize:          1536,
			direct8x8:             1,
			ppsRefCount:           [2]uint32{2, 1},
			wantSlices:            []int32{1, 2, 3},
			deblock:               []int32{1, 1, 1},
			direct:                []int32{0, 0, 1},
			configuredFrameCounts: []int{0, 1, 1, 1},
			bitstreamMD5:          "725250862a6f8083cc5d437fde9b2c62",
			frameMD5: []string{
				"49da495627ee1fa8171085d897fa391b",
				"d8504e1b9ac3d89f271fe308ca4a7352",
				"5487bcebdaf1ad7b5fe17261764e98bd",
			},
			rawVideoMD5: "afbb71db9b9fce02fcfe3bf4137fd250",
		},
		{
			bitDepth:              14,
			name:                  "spatial-direct-mode2-deblock",
			sourceFile:            "high10_direct_b_deblock_spatial_cavlc.h264",
			mode2Deblock:          true,
			width:                 32,
			height:                16,
			rawFrameSize:          1536,
			direct8x8:             1,
			ppsRefCount:           [2]uint32{2, 1},
			wantSlices:            []int32{1, 2, 3},
			deblock:               []int32{1, 2, 2},
			direct:                []int32{0, 0, 1},
			configuredFrameCounts: []int{0, 1, 1, 1},
			bitstreamMD5:          "0aa0059c0ee045950368f6c99c6c737e",
			frameMD5: []string{
				"49da495627ee1fa8171085d897fa391b",
				"d8504e1b9ac3d89f271fe308ca4a7352",
				"5487bcebdaf1ad7b5fe17261764e98bd",
			},
			rawVideoMD5: "afbb71db9b9fce02fcfe3bf4137fd250",
		},
		{
			bitDepth:              14,
			name:                  "temporal-bskip-mode1-deblock",
			sourceFile:            "high10_bskip_deblock_temporal_cavlc.h264",
			width:                 16,
			height:                16,
			rawFrameSize:          768,
			direct8x8:             1,
			ppsRefCount:           [2]uint32{2, 1},
			wantSlices:            []int32{1, 2, 3},
			deblock:               []int32{0, 1, 1},
			direct:                []int32{0, 0, 0},
			configuredFrameCounts: []int{0, 1, 1, 1},
			bitstreamMD5:          "57012f135c68a940f517a161390cdab3",
			frameMD5: []string{
				"6d3514a30f506561e144447d287270ab",
				"6d3514a30f506561e144447d287270ab",
				"6d3514a30f506561e144447d287270ab",
			},
			rawVideoMD5: "1f17900b95954131ed58ae42da301444",
		},
		{
			bitDepth:              14,
			name:                  "temporal-bskip-mode2-deblock",
			sourceFile:            "high10_bskip_deblock_temporal_cavlc.h264",
			mode2Deblock:          true,
			width:                 16,
			height:                16,
			rawFrameSize:          768,
			direct8x8:             1,
			ppsRefCount:           [2]uint32{2, 1},
			wantSlices:            []int32{1, 2, 3},
			deblock:               []int32{0, 2, 2},
			direct:                []int32{0, 0, 0},
			configuredFrameCounts: []int{0, 1, 1, 1},
			bitstreamMD5:          "ac2b66687c73cc1d12994a6fbdb55392",
			frameMD5: []string{
				"6d3514a30f506561e144447d287270ab",
				"6d3514a30f506561e144447d287270ab",
				"6d3514a30f506561e144447d287270ab",
			},
			rawVideoMD5: "1f17900b95954131ed58ae42da301444",
		},
		{
			bitDepth:              14,
			name:                  "spatial-bskip-mode1-deblock",
			sourceFile:            "high10_bskip_deblock_spatial_cavlc.h264",
			width:                 16,
			height:                16,
			rawFrameSize:          768,
			direct8x8:             1,
			ppsRefCount:           [2]uint32{2, 1},
			wantSlices:            []int32{1, 2, 3},
			deblock:               []int32{0, 1, 1},
			direct:                []int32{0, 0, 1},
			configuredFrameCounts: []int{0, 1, 1, 1},
			bitstreamMD5:          "46a3de03a5f3d4ada3022e97210d3d6d",
			frameMD5: []string{
				"6d3514a30f506561e144447d287270ab",
				"6d3514a30f506561e144447d287270ab",
				"6d3514a30f506561e144447d287270ab",
			},
			rawVideoMD5: "1f17900b95954131ed58ae42da301444",
		},
		{
			bitDepth:              14,
			name:                  "spatial-bskip-mode2-deblock",
			sourceFile:            "high10_bskip_deblock_spatial_cavlc.h264",
			mode2Deblock:          true,
			width:                 16,
			height:                16,
			rawFrameSize:          768,
			direct8x8:             1,
			ppsRefCount:           [2]uint32{2, 1},
			wantSlices:            []int32{1, 2, 3},
			deblock:               []int32{0, 2, 2},
			direct:                []int32{0, 0, 1},
			configuredFrameCounts: []int{0, 1, 1, 1},
			bitstreamMD5:          "9f4bbda4e3aa9d1b288cbc080fd74cb6",
			frameMD5: []string{
				"6d3514a30f506561e144447d287270ab",
				"6d3514a30f506561e144447d287270ab",
				"6d3514a30f506561e144447d287270ab",
			},
			rawVideoMD5: "1f17900b95954131ed58ae42da301444",
		},
		{
			bitDepth:              14,
			name:                  "direct-sub-b8x8-temporal-mode1-deblock",
			sourceFile:            "high10_cavlc_b8x8_temporal_direct_sub_deblock.h264",
			width:                 16,
			height:                16,
			rawFrameSize:          768,
			direct8x8:             1,
			ppsRefCount:           [2]uint32{2, 1},
			wantSlices:            []int32{1, 2, 3},
			deblock:               []int32{0, 1, 1},
			direct:                []int32{0, 0, 0},
			configuredFrameCounts: []int{0, 1, 1, 1},
			bitstreamMD5:          "21f514678fdc9d420fb1a267ca2b4de4",
			frameMD5: []string{
				"6d3514a30f506561e144447d287270ab",
				"6d3514a30f506561e144447d287270ab",
				"6d3514a30f506561e144447d287270ab",
			},
			rawVideoMD5: "1f17900b95954131ed58ae42da301444",
		},
		{
			bitDepth:              14,
			name:                  "direct-sub-b8x8-temporal-mode2-deblock",
			sourceFile:            "high10_cavlc_b8x8_temporal_direct_sub_deblock.h264",
			mode2Deblock:          true,
			width:                 16,
			height:                16,
			rawFrameSize:          768,
			direct8x8:             1,
			ppsRefCount:           [2]uint32{2, 1},
			wantSlices:            []int32{1, 2, 3},
			deblock:               []int32{0, 2, 2},
			direct:                []int32{0, 0, 0},
			configuredFrameCounts: []int{0, 1, 1, 1},
			bitstreamMD5:          "935343191b21868ad4b859ef4104431e",
			frameMD5: []string{
				"6d3514a30f506561e144447d287270ab",
				"6d3514a30f506561e144447d287270ab",
				"6d3514a30f506561e144447d287270ab",
			},
			rawVideoMD5: "1f17900b95954131ed58ae42da301444",
		},
		{
			bitDepth:              14,
			name:                  "direct-sub-b8x8-spatial-mode1-deblock",
			sourceFile:            "high10_cavlc_b8x8_spatial_direct_sub_deblock.h264",
			width:                 16,
			height:                16,
			rawFrameSize:          768,
			direct8x8:             1,
			ppsRefCount:           [2]uint32{2, 1},
			wantSlices:            []int32{1, 2, 3},
			deblock:               []int32{0, 1, 1},
			direct:                []int32{0, 0, 1},
			configuredFrameCounts: []int{0, 1, 1, 1},
			bitstreamMD5:          "10f9f09f0f6214ffcbfdcdde0f1c73f3",
			frameMD5: []string{
				"6d3514a30f506561e144447d287270ab",
				"6d3514a30f506561e144447d287270ab",
				"6d3514a30f506561e144447d287270ab",
			},
			rawVideoMD5: "1f17900b95954131ed58ae42da301444",
		},
		{
			bitDepth:              14,
			name:                  "direct-sub-b8x8-spatial-mode2-deblock",
			sourceFile:            "high10_cavlc_b8x8_spatial_direct_sub_deblock.h264",
			mode2Deblock:          true,
			width:                 16,
			height:                16,
			rawFrameSize:          768,
			direct8x8:             1,
			ppsRefCount:           [2]uint32{2, 1},
			wantSlices:            []int32{1, 2, 3},
			deblock:               []int32{0, 2, 2},
			direct:                []int32{0, 0, 1},
			configuredFrameCounts: []int{0, 1, 1, 1},
			bitstreamMD5:          "03bdcb0279f916ed2e92b95c8e304895",
			frameMD5: []string{
				"6d3514a30f506561e144447d287270ab",
				"6d3514a30f506561e144447d287270ab",
				"6d3514a30f506561e144447d287270ab",
			},
			rawVideoMD5: "1f17900b95954131ed58ae42da301444",
		},
		{
			bitDepth:              14,
			name:                  "direct-sub-b4x4-temporal-mode1-deblock",
			sourceFile:            "high10_cavlc_b4x4_temporal_direct_sub_deblock.h264",
			width:                 16,
			height:                16,
			rawFrameSize:          768,
			direct8x8:             0,
			ppsRefCount:           [2]uint32{2, 1},
			wantSlices:            []int32{1, 2, 3},
			deblock:               []int32{0, 1, 1},
			direct:                []int32{0, 0, 0},
			configuredFrameCounts: []int{0, 1, 1, 1},
			bitstreamMD5:          "cbb56f2fe2335de06b894d37a71f62e5",
			frameMD5: []string{
				"6d3514a30f506561e144447d287270ab",
				"6d3514a30f506561e144447d287270ab",
				"6d3514a30f506561e144447d287270ab",
			},
			rawVideoMD5: "1f17900b95954131ed58ae42da301444",
		},
		{
			bitDepth:              14,
			name:                  "direct-sub-b4x4-temporal-mode2-deblock",
			sourceFile:            "high10_cavlc_b4x4_temporal_direct_sub_deblock.h264",
			mode2Deblock:          true,
			width:                 16,
			height:                16,
			rawFrameSize:          768,
			direct8x8:             0,
			ppsRefCount:           [2]uint32{2, 1},
			wantSlices:            []int32{1, 2, 3},
			deblock:               []int32{0, 2, 2},
			direct:                []int32{0, 0, 0},
			configuredFrameCounts: []int{0, 1, 1, 1},
			bitstreamMD5:          "f8137420953fffc4c98c6cf497bf2f6c",
			frameMD5: []string{
				"6d3514a30f506561e144447d287270ab",
				"6d3514a30f506561e144447d287270ab",
				"6d3514a30f506561e144447d287270ab",
			},
			rawVideoMD5: "1f17900b95954131ed58ae42da301444",
		},
		{
			bitDepth:              14,
			name:                  "direct-sub-b4x4-spatial-mode1-deblock",
			sourceFile:            "high10_cavlc_b4x4_spatial_direct_sub_deblock.h264",
			width:                 16,
			height:                16,
			rawFrameSize:          768,
			direct8x8:             0,
			ppsRefCount:           [2]uint32{2, 1},
			wantSlices:            []int32{1, 2, 3},
			deblock:               []int32{0, 1, 1},
			direct:                []int32{0, 0, 1},
			configuredFrameCounts: []int{0, 1, 1, 1},
			bitstreamMD5:          "b9e8fcd953f9e99f58c11eda9ba90cba",
			frameMD5: []string{
				"6d3514a30f506561e144447d287270ab",
				"6d3514a30f506561e144447d287270ab",
				"6d3514a30f506561e144447d287270ab",
			},
			rawVideoMD5: "1f17900b95954131ed58ae42da301444",
		},
		{
			bitDepth:              14,
			name:                  "direct-sub-b4x4-spatial-mode2-deblock",
			sourceFile:            "high10_cavlc_b4x4_spatial_direct_sub_deblock.h264",
			mode2Deblock:          true,
			width:                 16,
			height:                16,
			rawFrameSize:          768,
			direct8x8:             0,
			ppsRefCount:           [2]uint32{2, 1},
			wantSlices:            []int32{1, 2, 3},
			deblock:               []int32{0, 2, 2},
			direct:                []int32{0, 0, 1},
			configuredFrameCounts: []int{0, 1, 1, 1},
			bitstreamMD5:          "e68e626d3457d4335762fdcba7cf591e",
			frameMD5: []string{
				"6d3514a30f506561e144447d287270ab",
				"6d3514a30f506561e144447d287270ab",
				"6d3514a30f506561e144447d287270ab",
			},
			rawVideoMD5: "1f17900b95954131ed58ae42da301444",
		},
		{
			bitDepth:              14,
			name:                  "partitioned-b16x8-mode1-deblock",
			sourceFile:            "high10_partitioned_b_deblock_b16x8_cavlc.h264",
			width:                 16,
			height:                16,
			rawFrameSize:          768,
			direct8x8:             1,
			ppsRefCount:           [2]uint32{1, 1},
			wantSlices:            []int32{1, 2, 3, 3, 2},
			deblock:               []int32{1, 1, 1, 1, 1},
			direct:                []int32{0, 0, 0, 0, 0},
			configuredFrameCounts: []int{0, 1, 1, 1, 1, 1},
			bitstreamMD5:          "f6bc5013d15f4e3f234cb4e3698356c3",
			frameMD5: []string{
				"76bba608fb2c663b35187dc6fd5ad541",
				"d9b8926cacb070d72fb90ce7a4b277f9",
				"bbeace8cfe6901a86b63662f89718fb8",
				"12208fd0262429f37e5942b9e2d1f9d8",
				"2d4008fca992537dbf2c34325410aedd",
			},
			rawVideoMD5: "762ad595a03ede1a1a4e9393adbf2c1b",
		},
		{
			bitDepth:              14,
			name:                  "partitioned-b16x8-mode2-deblock",
			sourceFile:            "high10_partitioned_b_deblock_b16x8_cavlc.h264",
			mode2Deblock:          true,
			width:                 16,
			height:                16,
			rawFrameSize:          768,
			direct8x8:             1,
			ppsRefCount:           [2]uint32{1, 1},
			wantSlices:            []int32{1, 2, 3, 3, 2},
			deblock:               []int32{1, 2, 2, 2, 2},
			direct:                []int32{0, 0, 0, 0, 0},
			configuredFrameCounts: []int{0, 1, 1, 1, 1, 1},
			bitstreamMD5:          "b8d43f93703e43baf53ff1361f9df414",
			frameMD5: []string{
				"76bba608fb2c663b35187dc6fd5ad541",
				"d9b8926cacb070d72fb90ce7a4b277f9",
				"bbeace8cfe6901a86b63662f89718fb8",
				"12208fd0262429f37e5942b9e2d1f9d8",
				"2d4008fca992537dbf2c34325410aedd",
			},
			rawVideoMD5: "762ad595a03ede1a1a4e9393adbf2c1b",
		},
		{
			bitDepth:              14,
			name:                  "partitioned-b8x16-mode1-deblock",
			sourceFile:            "high10_partitioned_b_deblock_b8x16_cavlc.h264",
			width:                 16,
			height:                16,
			rawFrameSize:          768,
			direct8x8:             1,
			ppsRefCount:           [2]uint32{1, 1},
			wantSlices:            []int32{1, 2, 3, 3, 2},
			deblock:               []int32{1, 1, 1, 1, 1},
			direct:                []int32{0, 0, 0, 0, 0},
			configuredFrameCounts: []int{0, 1, 1, 1, 1, 1},
			bitstreamMD5:          "a7709be44c24688a0b4ca1a7b81d49e9",
			frameMD5: []string{
				"00e31a71ce397179f111a49ff859faef",
				"a1ca1454ddcd19006f0d6b84b2971be5",
				"1bd782e7a7386b39616fa9248ec2327d",
				"735363c6d23f331ee828c237a8ab1946",
				"d9b2a282d9bdee1682dba11221ad3fd5",
			},
			rawVideoMD5: "ab46f4b665d41c16b5bcee3937cf807b",
		},
		{
			bitDepth:              14,
			name:                  "partitioned-b8x16-mode2-deblock",
			sourceFile:            "high10_partitioned_b_deblock_b8x16_cavlc.h264",
			mode2Deblock:          true,
			width:                 16,
			height:                16,
			rawFrameSize:          768,
			direct8x8:             1,
			ppsRefCount:           [2]uint32{1, 1},
			wantSlices:            []int32{1, 2, 3, 3, 2},
			deblock:               []int32{1, 2, 2, 2, 2},
			direct:                []int32{0, 0, 0, 0, 0},
			configuredFrameCounts: []int{0, 1, 1, 1, 1, 1},
			bitstreamMD5:          "3e6ae1d9e79d932fd45f731cc8e5a886",
			frameMD5: []string{
				"00e31a71ce397179f111a49ff859faef",
				"a1ca1454ddcd19006f0d6b84b2971be5",
				"1bd782e7a7386b39616fa9248ec2327d",
				"735363c6d23f331ee828c237a8ab1946",
				"d9b2a282d9bdee1682dba11221ad3fd5",
			},
			rawVideoMD5: "ab46f4b665d41c16b5bcee3937cf807b",
		},
		{
			bitDepth:              14,
			name:                  "partitioned-b8x8-mode1-deblock",
			sourceFile:            "high10_partitioned_b_deblock_b8x8_cavlc.h264",
			width:                 16,
			height:                16,
			rawFrameSize:          768,
			direct8x8:             1,
			ppsRefCount:           [2]uint32{1, 1},
			wantSlices:            []int32{1, 2, 3, 3, 2},
			deblock:               []int32{1, 1, 1, 1, 1},
			direct:                []int32{0, 0, 0, 0, 0},
			configuredFrameCounts: []int{0, 1, 1, 1, 1, 1},
			bitstreamMD5:          "aa664bec7db85c932526dcd1bd96201e",
			frameMD5: []string{
				"6bcadaf1ef408dfb87ed0eef55afc867",
				"f586fa4739462f3c5dbe104656c1bfe1",
				"44f20649f43ffd1d26039128351cef1b",
				"7659b11af00b55241e074e254fd9b4d2",
				"c8b2f1da66efe9ef9af551fcc4a45da5",
			},
			rawVideoMD5: "315da38cb68166ba299e5f0657d57d6b",
		},
		{
			bitDepth:              14,
			name:                  "partitioned-b8x8-mode2-deblock",
			sourceFile:            "high10_partitioned_b_deblock_b8x8_cavlc.h264",
			mode2Deblock:          true,
			width:                 16,
			height:                16,
			rawFrameSize:          768,
			direct8x8:             1,
			ppsRefCount:           [2]uint32{1, 1},
			wantSlices:            []int32{1, 2, 3, 3, 2},
			deblock:               []int32{1, 2, 2, 2, 2},
			direct:                []int32{0, 0, 0, 0, 0},
			configuredFrameCounts: []int{0, 1, 1, 1, 1, 1},
			bitstreamMD5:          "eb2d671071d5c66a7d822ff6c301eba2",
			frameMD5: []string{
				"6bcadaf1ef408dfb87ed0eef55afc867",
				"f586fa4739462f3c5dbe104656c1bfe1",
				"44f20649f43ffd1d26039128351cef1b",
				"7659b11af00b55241e074e254fd9b4d2",
				"c8b2f1da66efe9ef9af551fcc4a45da5",
			},
			rawVideoMD5: "315da38cb68166ba299e5f0657d57d6b",
		},
		{
			bitDepth:              12,
			name:                  "nondirect-no-deblock",
			sourceFile:            "high10_nondirect_b_cavlc.h264",
			width:                 16,
			height:                16,
			rawFrameSize:          768,
			direct8x8:             1,
			ppsRefCount:           [2]uint32{2, 1},
			wantSlices:            []int32{1, 2, 3},
			deblock:               []int32{0, 0, 0},
			direct:                []int32{0, 0, 0},
			configuredFrameCounts: []int{0, 1, 1, 1},
			bitstreamMD5:          "0d70d0516ad8158e8dbb5a3d74cd5efa",
			frameMD5: []string{
				"2271a65b978f0140735fd4b887c4eac9",
				"3fd8655393ca937a6c679d378d1bc790",
				"17296bd53e49cc810ebf19315bbf50ae",
			},
			rawVideoMD5: "4ece20c85fc4bb784db6e7beb86b5b84",
		},
		{
			bitDepth:              12,
			name:                  "temporal-direct-no-deblock",
			sourceFile:            "high10_direct_b_temporal_cavlc.h264",
			width:                 32,
			height:                16,
			rawFrameSize:          1536,
			direct8x8:             1,
			ppsRefCount:           [2]uint32{2, 1},
			wantSlices:            []int32{1, 2, 3},
			deblock:               []int32{0, 0, 0},
			direct:                []int32{0, 0, 0},
			configuredFrameCounts: []int{0, 1, 1, 1},
			bitstreamMD5:          "2c890e81014ad5d03a940ecdc2d87b97",
			frameMD5: []string{
				"c4f0a28b666d452d52b9f981f1f4a5f5",
				"1ddc82372683beffb8f6e9d224a8d5db",
				"75b08f9ac9bab88b28b0ca466930d04e",
			},
			rawVideoMD5: "5afd2074afd3e965dd7a61882c4c0713",
		},
		{
			bitDepth:              12,
			name:                  "spatial-direct-no-deblock",
			sourceFile:            "high10_direct_b_spatial_cavlc.h264",
			width:                 32,
			height:                16,
			rawFrameSize:          1536,
			direct8x8:             1,
			ppsRefCount:           [2]uint32{2, 1},
			wantSlices:            []int32{1, 2, 3},
			deblock:               []int32{0, 0, 0},
			direct:                []int32{0, 0, 1},
			configuredFrameCounts: []int{0, 1, 1, 1},
			bitstreamMD5:          "39ececb5ae8b631458e4ee4c4c212d73",
			frameMD5: []string{
				"c4f0a28b666d452d52b9f981f1f4a5f5",
				"1ddc82372683beffb8f6e9d224a8d5db",
				"75b08f9ac9bab88b28b0ca466930d04e",
			},
			rawVideoMD5: "5afd2074afd3e965dd7a61882c4c0713",
		},
		{
			bitDepth:              12,
			name:                  "temporal-bskip-no-deblock",
			sourceFile:            "high10_bskip_temporal_cavlc.h264",
			width:                 16,
			height:                16,
			rawFrameSize:          768,
			direct8x8:             1,
			ppsRefCount:           [2]uint32{2, 1},
			wantSlices:            []int32{1, 2, 3},
			deblock:               []int32{0, 0, 0},
			direct:                []int32{0, 0, 0},
			configuredFrameCounts: []int{0, 1, 1, 1},
			bitstreamMD5:          "4c2bb5398f5731724720bd6d3ac05399",
			frameMD5: []string{
				"941341cdfb37f5687de3a785d311fe7e",
				"941341cdfb37f5687de3a785d311fe7e",
				"941341cdfb37f5687de3a785d311fe7e",
			},
			rawVideoMD5: "52a85525458f5117f2f784bae02a3467",
		},
		{
			bitDepth:              12,
			name:                  "spatial-bskip-no-deblock",
			sourceFile:            "high10_bskip_spatial_cavlc.h264",
			width:                 16,
			height:                16,
			rawFrameSize:          768,
			direct8x8:             1,
			ppsRefCount:           [2]uint32{2, 1},
			wantSlices:            []int32{1, 2, 3},
			deblock:               []int32{0, 0, 0},
			direct:                []int32{0, 0, 1},
			configuredFrameCounts: []int{0, 1, 1, 1},
			bitstreamMD5:          "373e98e4a511f656fc3cbb46e49c8149",
			frameMD5: []string{
				"941341cdfb37f5687de3a785d311fe7e",
				"941341cdfb37f5687de3a785d311fe7e",
				"941341cdfb37f5687de3a785d311fe7e",
			},
			rawVideoMD5: "52a85525458f5117f2f784bae02a3467",
		},
		{
			bitDepth:              12,
			name:                  "direct-sub-b8x8-temporal-no-deblock",
			sourceFile:            "high10_cavlc_b8x8_temporal_direct_sub.h264",
			width:                 16,
			height:                16,
			rawFrameSize:          768,
			direct8x8:             1,
			ppsRefCount:           [2]uint32{2, 1},
			wantSlices:            []int32{1, 2, 3},
			deblock:               []int32{0, 0, 0},
			direct:                []int32{0, 0, 0},
			configuredFrameCounts: []int{0, 1, 1, 1},
			bitstreamMD5:          "23aac00a0cb90d97d2d5b7fa33c7840d",
			frameMD5: []string{
				"941341cdfb37f5687de3a785d311fe7e",
				"941341cdfb37f5687de3a785d311fe7e",
				"941341cdfb37f5687de3a785d311fe7e",
			},
			rawVideoMD5: "52a85525458f5117f2f784bae02a3467",
		},
		{
			bitDepth:              12,
			name:                  "direct-sub-b8x8-spatial-no-deblock",
			sourceFile:            "high10_cavlc_b8x8_spatial_direct_sub.h264",
			width:                 16,
			height:                16,
			rawFrameSize:          768,
			direct8x8:             1,
			ppsRefCount:           [2]uint32{2, 1},
			wantSlices:            []int32{1, 2, 3},
			deblock:               []int32{0, 0, 0},
			direct:                []int32{0, 0, 1},
			configuredFrameCounts: []int{0, 1, 1, 1},
			bitstreamMD5:          "f2f04041fe0ea0519773d20732968f9a",
			frameMD5: []string{
				"941341cdfb37f5687de3a785d311fe7e",
				"941341cdfb37f5687de3a785d311fe7e",
				"941341cdfb37f5687de3a785d311fe7e",
			},
			rawVideoMD5: "52a85525458f5117f2f784bae02a3467",
		},
		{
			bitDepth:              12,
			name:                  "direct-sub-b4x4-temporal-no-deblock",
			sourceFile:            "high10_cavlc_b4x4_temporal_direct_sub.h264",
			width:                 16,
			height:                16,
			rawFrameSize:          768,
			direct8x8:             0,
			ppsRefCount:           [2]uint32{2, 1},
			wantSlices:            []int32{1, 2, 3},
			deblock:               []int32{0, 0, 0},
			direct:                []int32{0, 0, 0},
			configuredFrameCounts: []int{0, 1, 1, 1},
			bitstreamMD5:          "1a93e5084216ce7a149efeeb405dcc40",
			frameMD5: []string{
				"941341cdfb37f5687de3a785d311fe7e",
				"941341cdfb37f5687de3a785d311fe7e",
				"941341cdfb37f5687de3a785d311fe7e",
			},
			rawVideoMD5: "52a85525458f5117f2f784bae02a3467",
		},
		{
			bitDepth:              12,
			name:                  "direct-sub-b4x4-spatial-no-deblock",
			sourceFile:            "high10_cavlc_b4x4_spatial_direct_sub.h264",
			width:                 16,
			height:                16,
			rawFrameSize:          768,
			direct8x8:             0,
			ppsRefCount:           [2]uint32{2, 1},
			wantSlices:            []int32{1, 2, 3},
			deblock:               []int32{0, 0, 0},
			direct:                []int32{0, 0, 1},
			configuredFrameCounts: []int{0, 1, 1, 1},
			bitstreamMD5:          "3f8325b55492dececf0de823a64fa7f6",
			frameMD5: []string{
				"941341cdfb37f5687de3a785d311fe7e",
				"941341cdfb37f5687de3a785d311fe7e",
				"941341cdfb37f5687de3a785d311fe7e",
			},
			rawVideoMD5: "52a85525458f5117f2f784bae02a3467",
		},
		{
			bitDepth:              12,
			name:                  "partitioned-b16x8-no-deblock",
			sourceFile:            "high10_partitioned_b16x8_cavlc.h264",
			width:                 16,
			height:                16,
			rawFrameSize:          768,
			direct8x8:             1,
			ppsRefCount:           [2]uint32{2, 1},
			wantSlices:            []int32{1, 2, 3},
			deblock:               []int32{0, 0, 0},
			direct:                []int32{0, 0, 0},
			configuredFrameCounts: []int{0, 1, 1, 1},
			bitstreamMD5:          "6ba0a84667734e751f88ed928c34d732",
			frameMD5: []string{
				"798a491d538aff9c9646d9f244d97d6e",
				"1127289fae51f3e139849b5208692171",
				"8493262abdb121435ba3f49ee10903c8",
			},
			rawVideoMD5: "0294158839ae2c72e64125bd9a25bab5",
		},
		{
			bitDepth:              12,
			name:                  "partitioned-b8x16-no-deblock",
			sourceFile:            "high10_partitioned_b8x16_cavlc.h264",
			width:                 16,
			height:                16,
			rawFrameSize:          768,
			direct8x8:             1,
			ppsRefCount:           [2]uint32{2, 1},
			wantSlices:            []int32{1, 2, 3},
			deblock:               []int32{0, 0, 0},
			direct:                []int32{0, 0, 0},
			configuredFrameCounts: []int{0, 1, 1, 1},
			bitstreamMD5:          "43d3a653afa721184cefe39e7b4a5031",
			frameMD5: []string{
				"227bc854b7d94794798387033e001792",
				"1127289fae51f3e139849b5208692171",
				"b4939bcf193864db7021fda8a03890c3",
			},
			rawVideoMD5: "b1d72346464edd66198dd739eb3e9608",
		},
		{
			bitDepth:              12,
			name:                  "partitioned-b8x8-no-deblock",
			sourceFile:            "high10_partitioned_b8x8_cavlc.h264",
			width:                 16,
			height:                16,
			rawFrameSize:          768,
			direct8x8:             1,
			ppsRefCount:           [2]uint32{2, 1},
			wantSlices:            []int32{1, 2, 3},
			deblock:               []int32{0, 0, 0},
			direct:                []int32{0, 0, 0},
			configuredFrameCounts: []int{0, 1, 1, 1},
			bitstreamMD5:          "8dc1efb97389eb8be372813a871b3f1c",
			frameMD5: []string{
				"0c4680a345c4ec8f227808ef08085c73",
				"02386c8fea89b8b1b74eea10d2d66052",
				"4353399b1c6d840b79a142ccefb9a0fc",
			},
			rawVideoMD5: "93b68dc71d75fe7e1ff4b03af0d53e3e",
		},
		{
			bitDepth:              14,
			name:                  "nondirect-no-deblock",
			sourceFile:            "high10_nondirect_b_cavlc.h264",
			width:                 16,
			height:                16,
			rawFrameSize:          768,
			direct8x8:             1,
			ppsRefCount:           [2]uint32{2, 1},
			wantSlices:            []int32{1, 2, 3},
			deblock:               []int32{0, 0, 0},
			direct:                []int32{0, 0, 0},
			configuredFrameCounts: []int{0, 1, 1, 1},
			bitstreamMD5:          "c217c34a462372795af00f1f31caee88",
			frameMD5: []string{
				"b5c40cef45fdf0a40973472a58c098b9",
				"e83cb189798b363baecce6e4c1373caf",
				"fa5d723fcf39a59ad5330300add86ce9",
			},
			rawVideoMD5: "0ae7ec305477a2b140ac3ebfa3b1c149",
		},
		{
			bitDepth:              14,
			name:                  "temporal-direct-no-deblock",
			sourceFile:            "high10_direct_b_temporal_cavlc.h264",
			width:                 32,
			height:                16,
			rawFrameSize:          1536,
			direct8x8:             1,
			ppsRefCount:           [2]uint32{2, 1},
			wantSlices:            []int32{1, 2, 3},
			deblock:               []int32{0, 0, 0},
			direct:                []int32{0, 0, 0},
			configuredFrameCounts: []int{0, 1, 1, 1},
			bitstreamMD5:          "4c629ce128438263123b0c7de1a66b9b",
			frameMD5: []string{
				"a1964709908fb5fcd51cf2bc3185dcb1",
				"4abc9e0ec3184201137654e7420edef1",
				"379b353cbaded8f05c480ecebd726389",
			},
			rawVideoMD5: "5a6aa2369d53966c4e4dcc3d713ccee7",
		},
		{
			bitDepth:              14,
			name:                  "spatial-direct-no-deblock",
			sourceFile:            "high10_direct_b_spatial_cavlc.h264",
			width:                 32,
			height:                16,
			rawFrameSize:          1536,
			direct8x8:             1,
			ppsRefCount:           [2]uint32{2, 1},
			wantSlices:            []int32{1, 2, 3},
			deblock:               []int32{0, 0, 0},
			direct:                []int32{0, 0, 1},
			configuredFrameCounts: []int{0, 1, 1, 1},
			bitstreamMD5:          "3026c361a5bb379874fa0b9f78cf9377",
			frameMD5: []string{
				"a1964709908fb5fcd51cf2bc3185dcb1",
				"4abc9e0ec3184201137654e7420edef1",
				"379b353cbaded8f05c480ecebd726389",
			},
			rawVideoMD5: "5a6aa2369d53966c4e4dcc3d713ccee7",
		},
		{
			bitDepth:              14,
			name:                  "temporal-bskip-no-deblock",
			sourceFile:            "high10_bskip_temporal_cavlc.h264",
			width:                 16,
			height:                16,
			rawFrameSize:          768,
			direct8x8:             1,
			ppsRefCount:           [2]uint32{2, 1},
			wantSlices:            []int32{1, 2, 3},
			deblock:               []int32{0, 0, 0},
			direct:                []int32{0, 0, 0},
			configuredFrameCounts: []int{0, 1, 1, 1},
			bitstreamMD5:          "c70957426ab39a87ff1670bff3b2d4bb",
			frameMD5: []string{
				"2163eb71b459070d04147dc124aac7c8",
				"2163eb71b459070d04147dc124aac7c8",
				"2163eb71b459070d04147dc124aac7c8",
			},
			rawVideoMD5: "db1f712add67d8ba54cb8cf10b86419f",
		},
		{
			bitDepth:              14,
			name:                  "spatial-bskip-no-deblock",
			sourceFile:            "high10_bskip_spatial_cavlc.h264",
			width:                 16,
			height:                16,
			rawFrameSize:          768,
			direct8x8:             1,
			ppsRefCount:           [2]uint32{2, 1},
			wantSlices:            []int32{1, 2, 3},
			deblock:               []int32{0, 0, 0},
			direct:                []int32{0, 0, 1},
			configuredFrameCounts: []int{0, 1, 1, 1},
			bitstreamMD5:          "1d481e9297098bebbd0c7175779cb84c",
			frameMD5: []string{
				"2163eb71b459070d04147dc124aac7c8",
				"2163eb71b459070d04147dc124aac7c8",
				"2163eb71b459070d04147dc124aac7c8",
			},
			rawVideoMD5: "db1f712add67d8ba54cb8cf10b86419f",
		},
		{
			bitDepth:              14,
			name:                  "direct-sub-b8x8-temporal-no-deblock",
			sourceFile:            "high10_cavlc_b8x8_temporal_direct_sub.h264",
			width:                 16,
			height:                16,
			rawFrameSize:          768,
			direct8x8:             1,
			ppsRefCount:           [2]uint32{2, 1},
			wantSlices:            []int32{1, 2, 3},
			deblock:               []int32{0, 0, 0},
			direct:                []int32{0, 0, 0},
			configuredFrameCounts: []int{0, 1, 1, 1},
			bitstreamMD5:          "08deb12885d2d1fefb31c1533dcd747b",
			frameMD5: []string{
				"2163eb71b459070d04147dc124aac7c8",
				"2163eb71b459070d04147dc124aac7c8",
				"2163eb71b459070d04147dc124aac7c8",
			},
			rawVideoMD5: "db1f712add67d8ba54cb8cf10b86419f",
		},
		{
			bitDepth:              14,
			name:                  "direct-sub-b8x8-spatial-no-deblock",
			sourceFile:            "high10_cavlc_b8x8_spatial_direct_sub.h264",
			width:                 16,
			height:                16,
			rawFrameSize:          768,
			direct8x8:             1,
			ppsRefCount:           [2]uint32{2, 1},
			wantSlices:            []int32{1, 2, 3},
			deblock:               []int32{0, 0, 0},
			direct:                []int32{0, 0, 1},
			configuredFrameCounts: []int{0, 1, 1, 1},
			bitstreamMD5:          "7087ec0154080f3a0155e85e1be0d046",
			frameMD5: []string{
				"2163eb71b459070d04147dc124aac7c8",
				"2163eb71b459070d04147dc124aac7c8",
				"2163eb71b459070d04147dc124aac7c8",
			},
			rawVideoMD5: "db1f712add67d8ba54cb8cf10b86419f",
		},
		{
			bitDepth:              14,
			name:                  "direct-sub-b4x4-temporal-no-deblock",
			sourceFile:            "high10_cavlc_b4x4_temporal_direct_sub.h264",
			width:                 16,
			height:                16,
			rawFrameSize:          768,
			direct8x8:             0,
			ppsRefCount:           [2]uint32{2, 1},
			wantSlices:            []int32{1, 2, 3},
			deblock:               []int32{0, 0, 0},
			direct:                []int32{0, 0, 0},
			configuredFrameCounts: []int{0, 1, 1, 1},
			bitstreamMD5:          "ad31c3db7f32d7a5f5c0261d09d4020a",
			frameMD5: []string{
				"2163eb71b459070d04147dc124aac7c8",
				"2163eb71b459070d04147dc124aac7c8",
				"2163eb71b459070d04147dc124aac7c8",
			},
			rawVideoMD5: "db1f712add67d8ba54cb8cf10b86419f",
		},
		{
			bitDepth:              14,
			name:                  "direct-sub-b4x4-spatial-no-deblock",
			sourceFile:            "high10_cavlc_b4x4_spatial_direct_sub.h264",
			width:                 16,
			height:                16,
			rawFrameSize:          768,
			direct8x8:             0,
			ppsRefCount:           [2]uint32{2, 1},
			wantSlices:            []int32{1, 2, 3},
			deblock:               []int32{0, 0, 0},
			direct:                []int32{0, 0, 1},
			configuredFrameCounts: []int{0, 1, 1, 1},
			bitstreamMD5:          "7b27029fb78ccea39dcf4739a5ba9c43",
			frameMD5: []string{
				"2163eb71b459070d04147dc124aac7c8",
				"2163eb71b459070d04147dc124aac7c8",
				"2163eb71b459070d04147dc124aac7c8",
			},
			rawVideoMD5: "db1f712add67d8ba54cb8cf10b86419f",
		},
		{
			bitDepth:              14,
			name:                  "partitioned-b16x8-no-deblock",
			sourceFile:            "high10_partitioned_b16x8_cavlc.h264",
			width:                 16,
			height:                16,
			rawFrameSize:          768,
			direct8x8:             1,
			ppsRefCount:           [2]uint32{2, 1},
			wantSlices:            []int32{1, 2, 3},
			deblock:               []int32{0, 0, 0},
			direct:                []int32{0, 0, 0},
			configuredFrameCounts: []int{0, 1, 1, 1},
			bitstreamMD5:          "458f9f4e4cc0c16811a4c1e1226d032d",
			frameMD5: []string{
				"ab9822e28369daf9b7cf3684a15a00d4",
				"75867a3f647b0896d7a8e491431bd2e6",
				"84654175ca7cc28ee8ccd2b93610731b",
			},
			rawVideoMD5: "dd61881fd1190b9d1778ec5897427e62",
		},
		{
			bitDepth:              14,
			name:                  "partitioned-b8x16-no-deblock",
			sourceFile:            "high10_partitioned_b8x16_cavlc.h264",
			width:                 16,
			height:                16,
			rawFrameSize:          768,
			direct8x8:             1,
			ppsRefCount:           [2]uint32{2, 1},
			wantSlices:            []int32{1, 2, 3},
			deblock:               []int32{0, 0, 0},
			direct:                []int32{0, 0, 0},
			configuredFrameCounts: []int{0, 1, 1, 1},
			bitstreamMD5:          "d06604a4177ecad40893f005c81a5a75",
			frameMD5: []string{
				"0602b9991b87ff67a6562312914d5fe6",
				"75867a3f647b0896d7a8e491431bd2e6",
				"ccf4838bb508322a2feb5836297faf46",
			},
			rawVideoMD5: "529819e477450b2324bf63e96370fab3",
		},
		{
			bitDepth:              14,
			name:                  "partitioned-b8x8-no-deblock",
			sourceFile:            "high10_partitioned_b8x8_cavlc.h264",
			width:                 16,
			height:                16,
			rawFrameSize:          768,
			direct8x8:             1,
			ppsRefCount:           [2]uint32{2, 1},
			wantSlices:            []int32{1, 2, 3},
			deblock:               []int32{0, 0, 0},
			direct:                []int32{0, 0, 0},
			configuredFrameCounts: []int{0, 1, 1, 1},
			bitstreamMD5:          "091231b24c14dee0989c7b300427e746",
			frameMD5: []string{
				"1fa06fe34e2b04ff8411571b5fda2be2",
				"325ec631ead813b356c945fbcab086da",
				"b036692ae137b899990b29de7437bffa",
			},
			rawVideoMD5: "271bffd2cc2417d0ebf60bd6ea98c875",
		},
	}
}
