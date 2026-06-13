// SPDX-License-Identifier: LGPL-2.1-or-later

package goh264_test

import (
	"crypto/md5"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/thesyncim/goh264/internal/h264"
)

type high12CABACChromaDeblockCase struct {
	name         string
	sourceFile   string
	chromaFormat uint32
	deblockMode  int32
	pixFmt       string
	frameSize    int
	bitstreamMD5 string
	rawVideoMD5  string
	frameMD5     []string
}

func TestHigh12CABACChromaDeblockFixtureSyntax(t *testing.T) {
	for _, tt := range high12CABACChromaDeblockCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := high12CABACChromaDeblockFixture(t, tt)
			assertHigh12CABACChromaDeblockFixtureSyntax(t, data, tt)
		})
	}
}

func TestDecodeAnnexBHigh12CABACChromaDeblockFrames(t *testing.T) {
	for _, tt := range high12CABACChromaDeblockCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := high12CABACChromaDeblockFixture(t, tt)
			assertHigh12CABACChromaDeblockFixtureSyntax(t, data, tt)
			frames, err := NewDecoder().DecodeAnnexBFrames(data)
			if err != nil {
				t.Fatalf("DecodeAnnexBFrames: %v", err)
			}
			assertHigh12CABACChromaDeblockFrames(t, frames, tt)
		})
	}
}

func TestDecodeAVCHigh12CABACChromaDeblockFrames(t *testing.T) {
	for _, tt := range high12CABACChromaDeblockCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := high12CABACChromaDeblockFixture(t, tt)
			assertHigh12CABACChromaDeblockFixtureSyntax(t, data, tt)
			for _, nalLengthSize := range []int{2, 3, 4} {
				frames, err := NewDecoder().DecodeAVCFrames(annexBToAVC(t, data, nalLengthSize), nalLengthSize)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: DecodeAVCFrames: %v", nalLengthSize, err)
				}
				assertHigh12CABACChromaDeblockFrames(t, frames, tt)
			}
		})
	}
}

func TestDecodeConfiguredAVCHigh12CABACChromaDeblockFramesAcrossSamples(t *testing.T) {
	for _, tt := range high12CABACChromaDeblockCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := high12CABACChromaDeblockFixture(t, tt)
			assertHigh12CABACChromaDeblockFixtureSyntax(t, data, tt)
			for _, nalLengthSize := range []int{2, 3, 4} {
				config, samples := annexBToAVCConfigAndSamples(t, data, nalLengthSize)
				if len(samples) != len(tt.frameMD5) {
					t.Fatalf("nalLengthSize=%d samples = %d, want %d", nalLengthSize, len(samples), len(tt.frameMD5))
				}
				dec := NewDecoder()
				if _, err := dec.ConfigureAVCC(config); err != nil {
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
				assertHigh12CABACChromaDeblockFrames(t, frames, tt)
			}
		})
	}
}

func TestFFmpegRawVideoFrameMD5OracleHigh12CABACChromaDeblock(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}
	for _, tt := range high12CABACChromaDeblockCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := high12CABACChromaDeblockFixture(t, tt)
			assertHigh12CABACChromaDeblockFixtureSyntax(t, data, tt)
			assertFFmpegHigh12CABACChromaDeblockRawVideoOracle(t, data, tt)
		})
	}
}

func high12CABACChromaDeblockCases() []high12CABACChromaDeblockCase {
	return []high12CABACChromaDeblockCase{
		{
			name:         "422-no-deblock",
			sourceFile:   "high10_deblock422_cabac_idrp.h264",
			chromaFormat: 2,
			deblockMode:  0,
			pixFmt:       "yuv422p12le",
			frameSize:    4096,
			bitstreamMD5: "1fb556e7410bfd4adabbaca1bf37e6dd",
			rawVideoMD5:  "0e90a4b22ad045c5f839275a1f84e6dc",
			frameMD5:     []string{"b6cbea08803e2645f65c899c7f3e3679", "84e42d25c509aa0840ebeca12dc2b2d6"},
		},
		{
			name:         "422-frame-deblock",
			sourceFile:   "high10_deblock422_cabac_idrp.h264",
			chromaFormat: 2,
			deblockMode:  1,
			pixFmt:       "yuv422p12le",
			frameSize:    4096,
			bitstreamMD5: "bda49ec21184f57f97155ff199a76438",
			rawVideoMD5:  "4a0a48e553c84cad161aaaf33937f66b",
			frameMD5:     []string{"bd35194f79ce24f0e9b708d618f658c9", "b3ae2413f9777cc953941054b8d735a2"},
		},
		{
			name:         "422-slice-boundary",
			sourceFile:   "high10_deblock422_cabac_idrp.h264",
			chromaFormat: 2,
			deblockMode:  2,
			pixFmt:       "yuv422p12le",
			frameSize:    4096,
			bitstreamMD5: "c4dc20dbe5e2497828423ac94168727c",
			rawVideoMD5:  "4a0a48e553c84cad161aaaf33937f66b",
			frameMD5:     []string{"bd35194f79ce24f0e9b708d618f658c9", "b3ae2413f9777cc953941054b8d735a2"},
		},
		{
			name:         "444-no-deblock",
			sourceFile:   "high10_deblock444_cabac_idrp.h264",
			chromaFormat: 3,
			deblockMode:  0,
			pixFmt:       "yuv444p12le",
			frameSize:    6144,
			bitstreamMD5: "559410a974bfb2db428785c74167bae1",
			rawVideoMD5:  "c9b52012514af816006b0ac173e76c51",
			frameMD5:     []string{"f87976014827d45c2a5fcc529f415765", "f5553bc10db9061394f7d6955b9f4f6a"},
		},
		{
			name:         "444-frame-deblock",
			sourceFile:   "high10_deblock444_cabac_idrp.h264",
			chromaFormat: 3,
			deblockMode:  1,
			pixFmt:       "yuv444p12le",
			frameSize:    6144,
			bitstreamMD5: "f0adadd34a84edf045e91698796cdbbc",
			rawVideoMD5:  "3ac7d266d53fccf47588a960cf2a1cf6",
			frameMD5:     []string{"599433f8008ddd76d053c6fe6ffffd70", "dba8ae73e9df756388b38da7c4ca76a1"},
		},
		{
			name:         "444-slice-boundary",
			sourceFile:   "high10_deblock444_cabac_idrp.h264",
			chromaFormat: 3,
			deblockMode:  2,
			pixFmt:       "yuv444p12le",
			frameSize:    6144,
			bitstreamMD5: "9f58b504aa02f04cc3f8ef92aded2c11",
			rawVideoMD5:  "3ac7d266d53fccf47588a960cf2a1cf6",
			frameMD5:     []string{"599433f8008ddd76d053c6fe6ffffd70", "dba8ae73e9df756388b38da7c4ca76a1"},
		},
	}
}

func high12CABACChromaDeblockFixture(t *testing.T, tt high12CABACChromaDeblockCase) []byte {
	t.Helper()
	data := high12CABACChromaDeblockRawFixture(t, tt)
	sum := md5.Sum(data)
	if got := hex.EncodeToString(sum[:]); got != tt.bitstreamMD5 {
		t.Fatalf("%s bitstream md5 = %s, want %s", tt.name, got, tt.bitstreamMD5)
	}
	return data
}

func high12CABACChromaDeblockRawFixture(t *testing.T, tt high12CABACChromaDeblockCase) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "h264", tt.sourceFile))
	if err != nil {
		t.Fatalf("read %s: %v", tt.sourceFile, err)
	}
	return high12CABACChromaDeblockRewriteAnnexB(t, data, tt.deblockMode)
}

func high12CABACChromaDeblockRewriteAnnexB(t *testing.T, data []byte, deblockMode int32) []byte {
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
				raw = highCABACChromaWeightedPRewriteSPSRaw(t, raw, 12)
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
				if sh.DeblockingFilter != deblockMode {
					raw = highCABACBRewriteSliceDeblockMode(t, raw, sh, deblockMode)
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

func assertHigh12CABACChromaDeblockFixtureSyntax(t *testing.T, data []byte, tt high12CABACChromaDeblockCase) {
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
	var gotSlices []int32
	for _, nal := range nals {
		switch nal.Type {
		case h264.NALSPS:
			sps, err := h264.DecodeSPS(nal.RBSP)
			if err != nil {
				t.Fatal(err)
			}
			if sps.ProfileIDC != 244 || sps.Width != 32 || sps.Height != 32 ||
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
			if pps.CABAC != 1 || pps.Transform8x8Mode != 0 || pps.WeightedPred != 0 ||
				pps.WeightedBipredIDC != 0 || pps.RefCount[0] != 1 || pps.RefCount[1] != 1 ||
				pps.DeblockingFilterParametersPresent == 0 {
				t.Fatalf("PPS cabac/8x8/weights/refs/deblock-present = %d/%d/%d/%d/%d/%d/%d, want CABAC/no-8x8 unweighted P ref=1",
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
			gotSlices = append(gotSlices, sh.SliceTypeNoS)
			if sh.PictureStructure != h264.PictureFrame || sh.DeblockingFilter != tt.deblockMode ||
				sh.SliceAlphaC0Offset != 0 || sh.SliceBetaOffset != 0 {
				t.Fatalf("slice picture/deblock/offsets = %d/%d/%d/%d, want frame/mode-%d/0/0",
					sh.PictureStructure, sh.DeblockingFilter, sh.SliceAlphaC0Offset, sh.SliceBetaOffset, tt.deblockMode)
			}
			if sh.SliceTypeNoS == h264.PictureTypeP {
				if sh.ListCount != 1 || sh.RefCount[0] != 1 || sh.PPS == nil || sh.PPS.WeightedPred != 0 ||
					sh.PredWeightTable.UseWeight != 0 || sh.PredWeightTable.UseWeightChroma != 0 {
					t.Fatalf("P slice lists/ref/weighted = %d/%d/%v/%d/%d, want one unweighted L0 ref",
						sh.ListCount, sh.RefCount[0], sh.PPS, sh.PredWeightTable.UseWeight, sh.PredWeightTable.UseWeightChroma)
				}
			}
		default:
			t.Fatalf("unexpected NAL type %d in %s", nal.Type, tt.name)
		}
	}
	if len(gotVCL) != 2 || gotVCL[0] != h264.NALIDRSlice || gotVCL[1] != h264.NALSlice {
		t.Fatalf("VCL NALs = %v, want IDR/P", gotVCL)
	}
	if gotSlices[0] != h264.PictureTypeI || gotSlices[1] != h264.PictureTypeP {
		t.Fatalf("slice types = %v, want I/P", gotSlices)
	}
}

func assertHigh12CABACChromaDeblockFrames(t *testing.T, frames []*Frame, tt high12CABACChromaDeblockCase) {
	t.Helper()
	if len(tt.frameMD5) == 0 {
		t.Fatalf("%s missing frame MD5s", tt.name)
	}
	if len(frames) != len(tt.frameMD5) {
		t.Fatalf("frames = %d, want %d", len(frames), len(tt.frameMD5))
	}
	rawVideo := make([]byte, 0, len(frames)*tt.frameSize)
	for i, frame := range frames {
		if frame == nil {
			t.Fatalf("frame[%d] is nil", i)
		}
		if frame.Width != 32 || frame.Height != 32 ||
			frame.ChromaFormatIDC != tt.chromaFormat ||
			frame.BitDepthLuma != 12 || frame.BitDepthChroma != 12 {
			t.Fatalf("frame[%d] metadata = %dx%d chroma %d depth %d/%d, want 32x32 chroma %d High12",
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

func assertFFmpegHigh12CABACChromaDeblockRawVideoOracle(t *testing.T, data []byte, tt high12CABACChromaDeblockCase) {
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
