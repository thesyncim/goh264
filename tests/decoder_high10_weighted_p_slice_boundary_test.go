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

type high10WeightedPSliceBoundaryCase struct {
	name           string
	sourceFile     string
	cabac          int32
	profileIDC     int32
	chromaFormat   uint32
	chromaWeighted bool
	pixFmt         string
	frameSize      int
	bitstreamMD5   string
	rawVideoMD5    string
	frameMD5       []string
}

func TestHigh10WeightedPSliceBoundaryFixtureSyntax(t *testing.T) {
	for _, tt := range high10WeightedPSliceBoundaryCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := high10WeightedPSliceBoundaryFixture(t, tt)
			assertHigh10WeightedPSliceBoundaryFixtureSyntax(t, data, tt)
		})
	}
}

func TestDecodeAnnexBHigh10WeightedPSliceBoundaryFrames(t *testing.T) {
	for _, tt := range high10WeightedPSliceBoundaryCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := high10WeightedPSliceBoundaryFixture(t, tt)
			assertHigh10WeightedPSliceBoundaryFixtureSyntax(t, data, tt)
			frames, err := NewDecoder().DecodeAnnexBFrames(data)
			if err != nil {
				t.Fatalf("DecodeAnnexBFrames: %v", err)
			}
			assertHigh10WeightedPSliceBoundaryFrames(t, frames, tt)
		})
	}
}

func TestDecodeAVCHigh10WeightedPSliceBoundaryFrames(t *testing.T) {
	for _, tt := range high10WeightedPSliceBoundaryCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := high10WeightedPSliceBoundaryFixture(t, tt)
			assertHigh10WeightedPSliceBoundaryFixtureSyntax(t, data, tt)
			for _, nalLengthSize := range []int{2, 3, 4} {
				frames, err := NewDecoder().DecodeAVCFrames(annexBToAVC(t, data, nalLengthSize), nalLengthSize)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: DecodeAVCFrames: %v", nalLengthSize, err)
				}
				assertHigh10WeightedPSliceBoundaryFrames(t, frames, tt)
			}
		})
	}
}

func TestDecodeConfiguredAVCHigh10WeightedPSliceBoundaryFramesAcrossSamples(t *testing.T) {
	for _, tt := range high10WeightedPSliceBoundaryCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := high10WeightedPSliceBoundaryFixture(t, tt)
			assertHigh10WeightedPSliceBoundaryFixtureSyntax(t, data, tt)
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
				assertHigh10WeightedPSliceBoundaryFrames(t, frames, tt)
			}
		})
	}
}

func TestFFmpegRawVideoFrameMD5OracleHigh10WeightedPSliceBoundary(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}
	for _, tt := range high10WeightedPSliceBoundaryCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := high10WeightedPSliceBoundaryFixture(t, tt)
			assertHigh10WeightedPSliceBoundaryFixtureSyntax(t, data, tt)
			assertFFmpegHigh10WeightedPSliceBoundaryRawVideoOracle(t, data, tt)
		})
	}
}

func high10WeightedPSliceBoundaryCases() []high10WeightedPSliceBoundaryCase {
	return []high10WeightedPSliceBoundaryCase{
		{
			name:           "422-cavlc-luma-chroma",
			sourceFile:     "high10_weighted422_deblock_cavlc_p.h264",
			profileIDC:     122,
			chromaFormat:   2,
			chromaWeighted: true,
			pixFmt:         "yuv422p10le",
			frameSize:      16384,
			bitstreamMD5:   "41d31124869322fa6d4fd8314c3eb78e",
			rawVideoMD5:    "7b009b0fa2eca2856bc01aec68a35dc6",
			frameMD5: []string{
				"c858ffaec930ea7da4b8f7da55bb42bd",
				"76164f4854b1a0485179e0300457bfbc",
				"7abb6df0c5d287684a8e325125cd2974",
				"36029cd0cd49b02e970169fae8823aaa",
				"9066d25ca7de0c647164635822a45a62",
			},
		},
		{
			name:           "422-cabac-luma-chroma",
			sourceFile:     "high10_weighted422_deblock_cabac_p.h264",
			cabac:          1,
			profileIDC:     122,
			chromaFormat:   2,
			chromaWeighted: true,
			pixFmt:         "yuv422p10le",
			frameSize:      16384,
			bitstreamMD5:   "ee77415e2bd6741e2d8c01e6a30083ab",
			rawVideoMD5:    "40fa56f24c86d687f00adc006a97ebc2",
			frameMD5: []string{
				"c858ffaec930ea7da4b8f7da55bb42bd",
				"51fe63f4fe31e2b4067354b5bb256601",
				"485a61705bd7336f204fac1ca6f90e20",
				"95214215beeaec0d6abed5fefde77679",
				"ded75c9e1fbf5b7456d55fd0c8773f6a",
			},
		},
		{
			name:           "444-cavlc-luma-chroma",
			sourceFile:     "high10_weighted444_deblock_cavlc_p.h264",
			profileIDC:     244,
			chromaFormat:   3,
			chromaWeighted: true,
			pixFmt:         "yuv444p10le",
			frameSize:      24576,
			bitstreamMD5:   "b565974cd16f4dcf0045ff371e055488",
			rawVideoMD5:    "8e810e33da6fc18f326bfe09e1fd5e2e",
			frameMD5: []string{
				"c6c0575b0e38de3102e528e789e43386",
				"b286e6f8b8c649442c54cd21ca1f4792",
				"58154c37640272131a0b1cf2f4f65bf8",
				"23e38f72c3bd9f772b33f777b7ba9075",
				"e7f730c794408f5f9733338b3e15717b",
			},
		},
		{
			name:           "444-cabac-luma-chroma",
			sourceFile:     "high10_weighted444_deblock_cabac_p.h264",
			cabac:          1,
			profileIDC:     244,
			chromaFormat:   3,
			chromaWeighted: true,
			pixFmt:         "yuv444p10le",
			frameSize:      24576,
			bitstreamMD5:   "00ba3487ac9ea1dfbdb546a4f284c548",
			rawVideoMD5:    "a1df43c43024eb39d59dfd486818382a",
			frameMD5: []string{
				"c6c0575b0e38de3102e528e789e43386",
				"a1bec05b56e629f518f07f498efe00f7",
				"6940a67aec526924e8402c45a6dc672b",
				"52dcee9302fe0f7cea463a82e32c75b5",
				"5da3a776e5cdec5e959be82b15375b82",
			},
		},
		{
			name:         "422-cavlc-luma-only",
			sourceFile:   "high10_luma_weighted422_deblock_cavlc_p.h264",
			profileIDC:   122,
			chromaFormat: 2,
			pixFmt:       "yuv422p10le",
			frameSize:    16384,
			bitstreamMD5: "0e5be39cd68d87965ba65d21eff450f6",
			rawVideoMD5:  "0b8c6eaf2a2e806acec5dfaa7eb1c17e",
			frameMD5: []string{
				"c858ffaec930ea7da4b8f7da55bb42bd",
				"2083a6a6d0cfc39e18a39766f323daef",
				"b8901fd7d626d1641c67ff4ebfa657af",
				"8d1d92538d182f0ac5f1f969d784e764",
				"079f3af3195bef303561dd716475cb31",
			},
		},
		{
			name:         "422-cabac-luma-only",
			sourceFile:   "high10_luma_weighted422_deblock_cabac_p.h264",
			cabac:        1,
			profileIDC:   122,
			chromaFormat: 2,
			pixFmt:       "yuv422p10le",
			frameSize:    16384,
			bitstreamMD5: "ef6a081c87ba0572a34c0c62da9339d3",
			rawVideoMD5:  "806ebc73cd6360b0cec49c5ae6d03396",
			frameMD5: []string{
				"c858ffaec930ea7da4b8f7da55bb42bd",
				"5028c4d55e2e222b89596ec36649f528",
				"1967249847946f1d421a90ba492923f4",
				"3364324cbafcf29a75a12d999d97cf0f",
				"03fe6b4154a999bdeb89c79bcb624783",
			},
		},
		{
			name:         "444-cavlc-luma-only",
			sourceFile:   "high10_luma_weighted444_deblock_cavlc_p.h264",
			profileIDC:   244,
			chromaFormat: 3,
			pixFmt:       "yuv444p10le",
			frameSize:    24576,
			bitstreamMD5: "201cb2dd0df8f8b897ccfa6c346ba290",
			rawVideoMD5:  "896d007e7f72cbaef4f0839ecf408131",
			frameMD5: []string{
				"c6c0575b0e38de3102e528e789e43386",
				"27fc5ad921d677769388c321ec3528dc",
				"196a0e12eea81dc599087032e094c72d",
				"bac9b0b4878cc01da78b7aaddb1cde51",
				"1ecc501042f48a1c41075846ea4d6574",
			},
		},
		{
			name:         "444-cabac-luma-only",
			sourceFile:   "high10_luma_weighted444_deblock_cabac_p.h264",
			cabac:        1,
			profileIDC:   244,
			chromaFormat: 3,
			pixFmt:       "yuv444p10le",
			frameSize:    24576,
			bitstreamMD5: "b78bb1911809f87177e100e11f82cc0c",
			rawVideoMD5:  "5af3f8ebd5f79fe35a66f12449b52f1f",
			frameMD5: []string{
				"c6c0575b0e38de3102e528e789e43386",
				"3dc1a1c02902c12f81315a5b94c838d1",
				"304d6c864cc46fb31accd7c682ebcbad",
				"67615cc5171fa673745f6544090df8c9",
				"d5d444a87a9cf5aacd32b788d7fd9892",
			},
		},
	}
}

func high10WeightedPSliceBoundaryFixture(t *testing.T, tt high10WeightedPSliceBoundaryCase) []byte {
	t.Helper()
	data := high10WeightedPSliceBoundaryRewriteAnnexB(t, readHigh10WeightedPSliceBoundarySource(t, tt))
	sum := md5.Sum(data)
	if got := hex.EncodeToString(sum[:]); got != tt.bitstreamMD5 {
		t.Fatalf("%s bitstream md5 = %s, want %s", tt.name, got, tt.bitstreamMD5)
	}
	return data
}

func readHigh10WeightedPSliceBoundarySource(t *testing.T, tt high10WeightedPSliceBoundaryCase) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "h264", tt.sourceFile))
	if err != nil {
		t.Fatalf("read %s: %v", tt.sourceFile, err)
	}
	return data
}

func high10WeightedPSliceBoundaryRewriteAnnexB(t *testing.T, data []byte) []byte {
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
					t.Fatalf("decode SPS: %v", err)
				}
				spsList[sps.SPSID] = sps
			case h264.NALPPS:
				pps, err := h264.DecodePPS(rbsp, &spsList)
				if err != nil {
					t.Fatalf("decode PPS: %v", err)
				}
				ppsList[pps.PPSID] = pps
			case h264.NALSlice, h264.NALIDRSlice:
				nal := h264.NALUnit{RefIDC: raw[0] >> 5 & 0x03, Type: nalType, Raw: raw, RBSP: rbsp}
				sh, err := h264.ParseSliceHeader(nal, &ppsList)
				if err != nil {
					t.Fatalf("parse source slice: %v", err)
				}
				if nalType == h264.NALSlice && sh.SliceTypeNoS == h264.PictureTypeP && sh.DeblockingFilter == 1 {
					if sh.PPS.CABAC != 0 {
						raw = highCABACBRewriteSliceDeblockMode(t, raw, sh, 2)
					} else {
						raw = highCAVLCBRewriteSliceDeblockMode(t, raw, sh)
					}
					rbsp = high14CABACBEBSPToRBSP(raw[1:])
					nal.RBSP = rbsp
					nal.Raw = raw
					sh, err = h264.ParseSliceHeader(nal, &ppsList)
					if err != nil {
						t.Fatalf("parse rewritten slice: %v", err)
					}
					if sh.DeblockingFilter != 2 {
						t.Fatalf("rewritten slice deblock = %d, want mode-2", sh.DeblockingFilter)
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

func assertHigh10WeightedPSliceBoundaryFixtureSyntax(t *testing.T, data []byte, tt high10WeightedPSliceBoundaryCase) {
	t.Helper()
	nals, err := h264.SplitAnnexB(data)
	if err != nil {
		t.Fatal(err)
	}
	var spsList [32]*h264.SPS
	var ppsList [256]*h264.PPS
	var pSlices int
	var lumaWeightedPSlices int
	var chromaWeightedPSlices int
	var gotVCL []h264.NALUnitType
	for _, nal := range nals {
		switch nal.Type {
		case h264.NALSPS:
			sps, err := h264.DecodeSPS(nal.RBSP)
			if err != nil {
				t.Fatal(err)
			}
			if sps.ProfileIDC != tt.profileIDC || sps.Width != 64 || sps.Height != 64 ||
				sps.ChromaFormatIDC != tt.chromaFormat || sps.BitDepthLuma != 10 || sps.BitDepthChroma != 10 ||
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
			if pps.CABAC != tt.cabac || pps.Transform8x8Mode != 0 || pps.WeightedPred != 1 ||
				pps.WeightedBipredIDC != 0 || pps.RefCount[0] != 1 || pps.RefCount[1] != 1 ||
				pps.DeblockingFilterParametersPresent == 0 {
				t.Fatalf("PPS cabac/8x8/weights/refs/deblock-present = %d/%d/%d/%d/%d/%d/%d, want %s",
					pps.CABAC, pps.Transform8x8Mode, pps.WeightedPred, pps.WeightedBipredIDC,
					pps.RefCount[0], pps.RefCount[1], pps.DeblockingFilterParametersPresent, tt.name)
			}
			ppsList[pps.PPSID] = pps
		case h264.NALSEI:
		case h264.NALIDRSlice, h264.NALSlice:
			sh, err := h264.ParseSliceHeader(nal, &ppsList)
			if err != nil {
				t.Fatal(err)
			}
			gotVCL = append(gotVCL, nal.Type)
			wantDeblock := int32(1)
			if nal.Type == h264.NALSlice && sh.SliceTypeNoS == h264.PictureTypeP {
				wantDeblock = 2
			}
			if sh.PictureStructure != h264.PictureFrame || sh.DeblockingFilter != wantDeblock {
				t.Fatalf("slice picture/deblock = %d/%d, want frame/mode%d", sh.PictureStructure, sh.DeblockingFilter, wantDeblock)
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
	if len(gotVCL) != 5 || gotVCL[0] != h264.NALIDRSlice {
		t.Fatalf("VCL NALs = %v, want IDR plus four P slices", gotVCL)
	}
	if pSlices != 4 {
		t.Fatalf("P slices = %d, want 4", pSlices)
	}
	if lumaWeightedPSlices == 0 {
		t.Fatal("weighted-P slice-boundary fixture has no luma-weighted P slices")
	}
	if tt.chromaWeighted && chromaWeightedPSlices == 0 {
		t.Fatal("weighted-P slice-boundary fixture has no chroma-weighted P slices")
	}
	if !tt.chromaWeighted && chromaWeightedPSlices != 0 {
		t.Fatalf("luma-only weighted-P slice-boundary fixture has %d chroma-weighted P slices, want 0", chromaWeightedPSlices)
	}
}

func assertHigh10WeightedPSliceBoundaryFrames(t *testing.T, frames []*Frame, tt high10WeightedPSliceBoundaryCase) {
	t.Helper()
	if len(frames) != len(tt.frameMD5) {
		t.Fatalf("frames = %d, want %d", len(frames), len(tt.frameMD5))
	}
	rawVideo := make([]byte, 0, len(frames)*tt.frameSize)
	for i, frame := range frames {
		if frame == nil {
			t.Fatalf("frame[%d] is nil", i)
		}
		if frame.Width != 64 || frame.Height != 64 ||
			frame.ChromaFormatIDC != tt.chromaFormat ||
			frame.BitDepthLuma != 10 || frame.BitDepthChroma != 10 {
			t.Fatalf("frame[%d] metadata = %dx%d chroma %d depth %d/%d, want 64x64 chroma %d High10",
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

func assertFFmpegHigh10WeightedPSliceBoundaryRawVideoOracle(t *testing.T, data []byte, tt high10WeightedPSliceBoundaryCase) {
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
