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

type high10ChromaWeightedBCase struct {
	name         string
	sourceFile   string
	explicit     bool
	mode2Deblock bool
	cabac        int32
	profileIDC   int32
	chromaFormat uint32
	pixFmt       string
	frameSize    int
	bitstreamMD5 string
	rawVideoMD5  string
	frameMD5     []string
}

func TestHigh10ChromaWeightedBFixtureSyntax(t *testing.T) {
	for _, tt := range high10ChromaWeightedBCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := high10ChromaWeightedBFixture(t, tt)
			assertHigh10ChromaWeightedBFixtureSyntax(t, data, tt)
		})
	}
}

func TestDecodeAnnexBHigh10ChromaWeightedBFrames(t *testing.T) {
	for _, tt := range high10ChromaWeightedBCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := high10ChromaWeightedBFixture(t, tt)
			assertHigh10ChromaWeightedBFixtureSyntax(t, data, tt)
			frames, err := NewDecoder().DecodeAnnexBFrames(data)
			if err != nil {
				t.Fatalf("DecodeAnnexBFrames: %v", err)
			}
			assertHigh10ChromaWeightedBFrames(t, frames, tt)
		})
	}
}

func TestDecodeAVCHigh10ChromaWeightedBFrames(t *testing.T) {
	for _, tt := range high10ChromaWeightedBCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := high10ChromaWeightedBFixture(t, tt)
			assertHigh10ChromaWeightedBFixtureSyntax(t, data, tt)
			for _, nalLengthSize := range []int{2, 3, 4} {
				frames, err := NewDecoder().DecodeAVCFrames(annexBToAVC(t, data, nalLengthSize), nalLengthSize)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: DecodeAVCFrames: %v", nalLengthSize, err)
				}
				assertHigh10ChromaWeightedBFrames(t, frames, tt)
			}
		})
	}
}

func TestDecodeConfiguredAVCHigh10ChromaWeightedBFramesAcrossSamplesFlush(t *testing.T) {
	for _, tt := range high10ChromaWeightedBCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := high10ChromaWeightedBFixture(t, tt)
			assertHigh10ChromaWeightedBFixtureSyntax(t, data, tt)
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
				assertHigh10ChromaWeightedBFrames(t, frames, tt)
			}
		})
	}
}

func TestFFmpegRawVideoFrameMD5OracleHigh10ChromaWeightedB(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}
	for _, tt := range high10ChromaWeightedBCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := high10ChromaWeightedBFixture(t, tt)
			assertHigh10ChromaWeightedBFixtureSyntax(t, data, tt)
			assertFFmpegHigh10ChromaWeightedBRawVideoOracle(t, data, tt)
		})
	}
}

func high10ChromaWeightedBCases() []high10ChromaWeightedBCase {
	base := []high10ChromaWeightedBCase{
		{
			name:         "422-cavlc-implicit",
			sourceFile:   "high10_weighted422_cavlc_b.h264",
			profileIDC:   122,
			chromaFormat: 2,
			pixFmt:       "yuv422p10le",
			frameSize:    16384,
			bitstreamMD5: "be7d0bc8b0640affcccdf0106d267e5e",
			rawVideoMD5:  "89b8ef4ae1a280fbe3ad1fdcbca8eb5d",
			frameMD5: []string{
				"01561e660529dc2f5e4e764c5c49ba29",
				"da46c9c9bb9327208654d3396c364b96",
				"879fe23895727ad1f8718952179b344e",
				"4aa38fd05a5ba4d625f76af8c81fb80d",
				"8487c42726e244a79e0876974bf1447d",
			},
		},
		{
			name:         "422-cavlc-explicit",
			sourceFile:   "high10_weighted422_cavlc_b.h264",
			explicit:     true,
			profileIDC:   122,
			chromaFormat: 2,
			pixFmt:       "yuv422p10le",
			frameSize:    16384,
			bitstreamMD5: "1ad7f931209b3f45a4cdf68cade0f7fd",
			rawVideoMD5:  "da92759bb89b09280627acce8fbb61d2",
			frameMD5: []string{
				"01561e660529dc2f5e4e764c5c49ba29",
				"f642da731f9d3fee32d74ceefa342553",
				"28d649142e316e853452cbe5dec4c3a2",
				"4aa38fd05a5ba4d625f76af8c81fb80d",
				"8487c42726e244a79e0876974bf1447d",
			},
		},
		{
			name:         "422-cabac-implicit",
			sourceFile:   "high10_weighted422_cabac_b.h264",
			cabac:        1,
			profileIDC:   122,
			chromaFormat: 2,
			pixFmt:       "yuv422p10le",
			frameSize:    16384,
			bitstreamMD5: "f25b6210573103d8b9b809d74314ad7f",
			rawVideoMD5:  "627afa02370c2f6fbf399a34d1999fe7",
			frameMD5: []string{
				"fa609c2ed6b07cf665d3fd9fb23415c9",
				"b813b9db771f2cd12088cf8a3f86a9b4",
				"b0f21d828b951cd20c847a72393cfb9b",
				"c92b95bb26e13ec1b0b7896c9b0c87dd",
				"5adace6adfce98e8c0a0d70f2c8e19a6",
			},
		},
		{
			name:         "422-cabac-explicit",
			sourceFile:   "high10_weighted422_cabac_b.h264",
			explicit:     true,
			cabac:        1,
			profileIDC:   122,
			chromaFormat: 2,
			pixFmt:       "yuv422p10le",
			frameSize:    16384,
			bitstreamMD5: "9976385222e65af0f8ff2a2587b304fc",
			rawVideoMD5:  "631528ad0dfe9f38f50a1b971cc822a1",
			frameMD5: []string{
				"fa609c2ed6b07cf665d3fd9fb23415c9",
				"df22553d1b9293ca3beaa0c842b4f1ec",
				"2ca1a1e4e1b53631136744e5cc989713",
				"c92b95bb26e13ec1b0b7896c9b0c87dd",
				"5adace6adfce98e8c0a0d70f2c8e19a6",
			},
		},
		{
			name:         "444-cavlc-implicit",
			sourceFile:   "high10_weighted444_cavlc_b.h264",
			profileIDC:   244,
			chromaFormat: 3,
			pixFmt:       "yuv444p10le",
			frameSize:    24576,
			bitstreamMD5: "5cb1b64c99d2093dbf90e4b924f97cdd",
			rawVideoMD5:  "4586852f9dc3abfbb0f57640202a42e6",
			frameMD5: []string{
				"e9a886d86293c43acc11281936d56fbf",
				"98ce54123338090d85022a6681bcba1f",
				"330de842c5ec40ac65486de4ce65ad15",
				"c61ea3364212cbbe84919f6a19ccfd87",
				"53f5e0ecacccf50b54fd50fcc4141431",
			},
		},
		{
			name:         "444-cavlc-explicit",
			sourceFile:   "high10_weighted444_cavlc_b.h264",
			explicit:     true,
			profileIDC:   244,
			chromaFormat: 3,
			pixFmt:       "yuv444p10le",
			frameSize:    24576,
			bitstreamMD5: "cba2b3ba2bdce28b8d57fce3e636b37f",
			rawVideoMD5:  "b6252358ac36bd34da477b78071e3181",
			frameMD5: []string{
				"e9a886d86293c43acc11281936d56fbf",
				"c3c3ab46a7dc6c243a91b9bcadacd088",
				"5ea178bf422bd771aafba9ac4ffd74aa",
				"c61ea3364212cbbe84919f6a19ccfd87",
				"53f5e0ecacccf50b54fd50fcc4141431",
			},
		},
		{
			name:         "444-cabac-implicit",
			sourceFile:   "high10_weighted444_cabac_b.h264",
			cabac:        1,
			profileIDC:   244,
			chromaFormat: 3,
			pixFmt:       "yuv444p10le",
			frameSize:    24576,
			bitstreamMD5: "e1ef4964e037fe535ca937aeb2f4f6e1",
			rawVideoMD5:  "7d6f7459992f3d47714215a9c4fa31f8",
			frameMD5: []string{
				"f6de0fe7cfb89d883131c7eb197cc56d",
				"eabc814b239eee7bec70b7420fdcd063",
				"a14736925cf1a0cbe437f048f7e904ff",
				"5380ac086622bc42fdb9a2d82db1232a",
				"081f2757b3dd244c15695a2e3099a7db",
			},
		},
		{
			name:         "444-cabac-explicit",
			sourceFile:   "high10_weighted444_cabac_b.h264",
			explicit:     true,
			cabac:        1,
			profileIDC:   244,
			chromaFormat: 3,
			pixFmt:       "yuv444p10le",
			frameSize:    24576,
			bitstreamMD5: "7b289c1866d8e233256272f6c5497bbe",
			rawVideoMD5:  "34181a7513fa612b1f8ac31dc5ad93e4",
			frameMD5: []string{
				"f6de0fe7cfb89d883131c7eb197cc56d",
				"e9c12f844f9fe1651ec2fcf557f2fd1c",
				"1ebd42c125557364f19bb7002ad950d0",
				"5380ac086622bc42fdb9a2d82db1232a",
				"081f2757b3dd244c15695a2e3099a7db",
			},
		},
	}
	return append(base, high10ChromaWeightedBMode2Cases(base)...)
}

func high10ChromaWeightedBMode2Cases(base []high10ChromaWeightedBCase) []high10ChromaWeightedBCase {
	expected := map[string]string{
		"422-cavlc-implicit-mode2": "0d652d362653e0407c613b81367b01e1",
		"422-cavlc-explicit-mode2": "255b9a5ac96b7717de76c2aec91040d2",
		"422-cabac-implicit-mode2": "414c3641b55e39d1c8ac4fea1be12b74",
		"422-cabac-explicit-mode2": "ea9459c0183e49f8de16dbbd58ce98c5",
		"444-cavlc-implicit-mode2": "bf23fea603c1b888c5869694304aeae6",
		"444-cavlc-explicit-mode2": "53054070aa69758862fd8ca88a8faac2",
		"444-cabac-implicit-mode2": "0b62c859094ed823aecb8c688d173d46",
		"444-cabac-explicit-mode2": "6ea84035a33fa09cc9755e2da968bded",
	}
	out := make([]high10ChromaWeightedBCase, 0, len(base))
	for _, tt := range base {
		tt.name += "-mode2"
		tt.mode2Deblock = true
		tt.bitstreamMD5 = expected[tt.name]
		out = append(out, tt)
	}
	return out
}

func high10ChromaWeightedBFixture(t *testing.T, tt high10ChromaWeightedBCase) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "h264", tt.sourceFile))
	if err != nil {
		t.Fatalf("read %s: %v", tt.sourceFile, err)
	}
	if tt.explicit || tt.mode2Deblock {
		data = high10ChromaWeightedBRewriteAnnexB(t, data, tt.explicit, tt.mode2Deblock)
	}
	sum := md5.Sum(data)
	if got := hex.EncodeToString(sum[:]); got != tt.bitstreamMD5 {
		t.Fatalf("%s bitstream md5 = %s, want %s", tt.name, got, tt.bitstreamMD5)
	}
	return data
}

func assertHigh10ChromaWeightedBFrames(t *testing.T, frames []*Frame, tt high10ChromaWeightedBCase) {
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
		if len(frame.Y) != 0 || len(frame.Cb) != 0 || len(frame.Cr) != 0 {
			t.Fatalf("frame[%d] populated 8-bit planes", i)
		}
		if len(frame.Y16) == 0 || len(frame.Cb16) == 0 || len(frame.Cr16) == 0 {
			t.Fatalf("frame[%d] missing high planes", i)
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

func assertHigh10ChromaWeightedBFixtureSyntax(t *testing.T, data []byte, tt high10ChromaWeightedBCase) {
	t.Helper()
	nals, err := h264.SplitAnnexB(data)
	if err != nil {
		t.Fatal(err)
	}
	var spsList [32]*h264.SPS
	var ppsList [256]*h264.PPS
	var gotVCL []h264.NALUnitType
	var gotSlices []int32
	var bSlices int
	for _, nal := range nals {
		switch nal.Type {
		case h264.NALSPS:
			sps, err := h264.DecodeSPS(nal.RBSP)
			if err != nil {
				t.Fatal(err)
			}
			if sps.ProfileIDC != tt.profileIDC || sps.Width != 64 || sps.Height != 64 ||
				sps.ChromaFormatIDC != tt.chromaFormat || sps.BitDepthLuma != 10 || sps.BitDepthChroma != 10 ||
				sps.FrameMBSOnlyFlag != 1 || sps.MBAFF != 0 || sps.RefFrameCount != 2 {
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
			wantWeightedBipred := uint32(2)
			if tt.explicit {
				wantWeightedBipred = 1
			}
			if pps.CABAC != tt.cabac || pps.Transform8x8Mode != 0 || pps.WeightedPred != 0 ||
				pps.WeightedBipredIDC != wantWeightedBipred || pps.RefCount[0] != 2 || pps.RefCount[1] != 1 {
				t.Fatalf("PPS cabac/8x8/weights/refs = %d/%d/%d/%d/%d/%d, want cabac=%d weighted_bipred_idc=%d refs=2/1",
					pps.CABAC, pps.Transform8x8Mode, pps.WeightedPred, pps.WeightedBipredIDC,
					pps.RefCount[0], pps.RefCount[1], tt.cabac, wantWeightedBipred)
			}
			ppsList[pps.PPSID] = pps
		case h264.NALSEI:
		case h264.NALIDRSlice, h264.NALSlice:
			sh, err := h264.ParseSliceHeader(nal, &ppsList)
			if err != nil {
				t.Fatal(err)
			}
			wantDeblock := int32(1)
			if tt.mode2Deblock && nal.Type == h264.NALSlice {
				wantDeblock = 2
			}
			if sh.PictureStructure != h264.PictureFrame || sh.DeblockingFilter != wantDeblock {
				t.Fatalf("slice picture/deblock = %d/%d, want frame/mode%d", sh.PictureStructure, sh.DeblockingFilter, wantDeblock)
			}
			gotVCL = append(gotVCL, nal.Type)
			gotSlices = append(gotSlices, sh.SliceTypeNoS)
			switch sh.SliceTypeNoS {
			case h264.PictureTypeI:
				if sh.ListCount != 0 {
					t.Fatalf("I slice list count = %d, want 0", sh.ListCount)
				}
			case h264.PictureTypeP:
				if sh.ListCount != 1 || sh.RefCount[0] < 1 || sh.PredWeightTable.UseWeight != 0 || sh.PredWeightTable.UseWeightChroma != 0 {
					t.Fatalf("P slice lists/refs/weights = %d/%v/%d/%d, want unweighted P refs",
						sh.ListCount, sh.RefCount, sh.PredWeightTable.UseWeight, sh.PredWeightTable.UseWeightChroma)
				}
			case h264.PictureTypeB:
				bSlices++
				if sh.ListCount != 2 || sh.RefCount[0] != 1 || sh.RefCount[1] != 1 {
					t.Fatalf("B slice lists/refs = %d/%v, want refs=1/1", sh.ListCount, sh.RefCount)
				}
				if tt.explicit {
					if sh.PredWeightTable.UseWeight != 1 || sh.PredWeightTable.UseWeightChroma != 1 {
						t.Fatalf("explicit B weights = %d/%d, want 1/1", sh.PredWeightTable.UseWeight, sh.PredWeightTable.UseWeightChroma)
					}
				} else if sh.PredWeightTable.UseWeight != 0 || sh.PredWeightTable.UseWeightChroma != 0 {
					t.Fatalf("implicit B serialized weights = %d/%d, want none before DPB init",
						sh.PredWeightTable.UseWeight, sh.PredWeightTable.UseWeightChroma)
				}
			default:
				t.Fatalf("slice type = %d, want I/P/B", sh.SliceTypeNoS)
			}
		default:
			t.Fatalf("unexpected NAL type %d in %s", nal.Type, tt.name)
		}
	}
	if len(gotVCL) != len(tt.frameMD5) {
		t.Fatalf("VCL count = %d, want %d", len(gotVCL), len(tt.frameMD5))
	}
	if bSlices == 0 {
		t.Fatal("weighted-B fixture has no B slices")
	}
	if gotVCL[0] != h264.NALIDRSlice {
		t.Fatalf("first VCL = %d, want IDR", gotVCL[0])
	}
	if gotSlices[0] != h264.PictureTypeI {
		t.Fatalf("first slice = %d, want I", gotSlices[0])
	}
}

func assertFFmpegHigh10ChromaWeightedBRawVideoOracle(t *testing.T, data []byte, tt high10ChromaWeightedBCase) {
	t.Helper()
	path := writeTempH264(t, data)
	framemd5 := exec.Command("ffmpeg",
		"-hide_banner", "-v", "error",
		"-f", "h264",
		"-i", path,
		"-an", "-sn", "-dn",
		"-pix_fmt", tt.pixFmt,
		"-f", "framemd5",
		"-",
	)
	framemd5Out, err := framemd5.Output()
	if err != nil {
		t.Fatalf("ffmpeg framemd5: %v", err)
	}
	for i, want := range tt.frameMD5 {
		line := []byte(fmt.Sprintf("0, %10d, %10d,        1, %8d, %s", i, i, tt.frameSize, want))
		if !bytes.Contains(framemd5Out, line) {
			t.Fatalf("frame[%d] missing %q in framemd5:\n%s", i, line, framemd5Out)
		}
	}
	rawCmd := exec.Command("ffmpeg",
		"-hide_banner", "-v", "error",
		"-f", "h264",
		"-i", path,
		"-an", "-sn", "-dn",
		"-pix_fmt", tt.pixFmt,
		"-f", "rawvideo",
		"-",
	)
	raw, err := rawCmd.Output()
	if err != nil {
		t.Fatalf("ffmpeg rawvideo: %v", err)
	}
	if len(raw) != len(tt.frameMD5)*tt.frameSize {
		t.Fatalf("rawvideo size = %d, want %d", len(raw), len(tt.frameMD5)*tt.frameSize)
	}
	sum := md5.Sum(raw)
	if got := hex.EncodeToString(sum[:]); got != tt.rawVideoMD5 {
		t.Fatalf("rawvideo md5 = %s, want %s", got, tt.rawVideoMD5)
	}
}

func high10ChromaWeightedBRewriteAnnexB(t *testing.T, data []byte, explicit bool, mode2Deblock bool) []byte {
	t.Helper()
	start, prefixLen, ok := high14CABACBFindStartCode(data, 0)
	if !ok {
		t.Fatal("source fixture has no Annex B start code")
	}
	var out []byte
	var sourceSPSList [32]*h264.SPS
	var sourcePPSList [256]*h264.PPS
	var rewrittenSPSList [32]*h264.SPS
	var rewrittenPPSList [256]*h264.PPS
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
				sourceSPSList[sps.SPSID] = sps
				rewrittenSPSList[sps.SPSID] = sps
			case h264.NALPPS:
				sourcePPS, err := h264.DecodePPS(rbsp, &sourceSPSList)
				if err != nil {
					t.Fatalf("decode source PPS: %v", err)
				}
				sourcePPSList[sourcePPS.PPSID] = sourcePPS
				if explicit {
					raw = highExplicitWeightedBRewritePPSRaw(t, raw)
					rewrittenPPS, err := h264.DecodePPS(high14CABACBEBSPToRBSP(raw[1:]), &rewrittenSPSList)
					if err != nil {
						t.Fatalf("decode rewritten PPS: %v", err)
					}
					rewrittenPPSList[rewrittenPPS.PPSID] = rewrittenPPS
				}
			case h264.NALSlice, h264.NALIDRSlice:
				nal := h264.NALUnit{RefIDC: raw[0] >> 5 & 0x03, Type: nalType, Raw: raw, RBSP: rbsp}
				sh, err := h264.ParseSliceHeader(nal, &sourcePPSList)
				if err != nil {
					t.Fatalf("parse source slice: %v", err)
				}
				if mode2Deblock && nalType == h264.NALSlice && sh.DeblockingFilter == 1 {
					if sh.PPS.CABAC != 0 {
						raw = highCABACBRewriteSliceDeblockMode(t, raw, sh, 2)
					} else {
						raw = highCAVLCBRewriteSliceDeblockMode(t, raw, sh)
					}
					rbsp = high14CABACBEBSPToRBSP(raw[1:])
					nal.RBSP = rbsp
					nal.Raw = raw
					sh, err = h264.ParseSliceHeader(nal, &sourcePPSList)
					if err != nil {
						t.Fatalf("parse mode-2 rewritten source slice: %v", err)
					}
					if sh.DeblockingFilter != 2 {
						t.Fatalf("mode-2 rewritten slice deblock = %d, want 2", sh.DeblockingFilter)
					}
				}
				if explicit && sh.SliceTypeNoS == h264.PictureTypeB {
					raw = highExplicitWeightedBRewriteSliceRaw(t, raw, sh)
					rbsp = high14CABACBEBSPToRBSP(raw[1:])
					nal.RBSP = rbsp
					nal.Raw = raw
					got, err := h264.ParseSliceHeader(nal, &rewrittenPPSList)
					if err != nil {
						t.Fatalf("parse explicit weighted-B slice: %v", err)
					}
					if got.PPS == nil || got.PPS.WeightedBipredIDC != 1 ||
						got.PredWeightTable.UseWeight != 1 || got.PredWeightTable.UseWeightChroma != 1 {
						t.Fatalf("explicit weighted-B parse weights = pps %v use %d/%d, want weighted_bipred_idc=1 use=1/1",
							got.PPS, got.PredWeightTable.UseWeight, got.PredWeightTable.UseWeightChroma)
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
