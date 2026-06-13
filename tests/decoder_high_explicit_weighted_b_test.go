// SPDX-License-Identifier: LGPL-2.1-or-later

package goh264_test

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/thesyncim/goh264/internal/h264"
)

type highExplicitWeightedBExpected struct {
	bitstreamMD5 string
	frameMD5     []string
	rawVideoMD5  string
}

func TestHigh1214ExplicitWeightedBFixtureSyntax(t *testing.T) {
	for _, tt := range high1214ExplicitWeightedBCases(t) {
		t.Run(highWeightedBCaseName(tt), func(t *testing.T) {
			data := highExplicitWeightedBFixture(t, tt)
			assertHighExplicitWeightedBFixtureSyntax(t, data, tt)
		})
	}
}

func TestDecodeAnnexBHigh1214ExplicitWeightedBFrames(t *testing.T) {
	for _, tt := range high1214ExplicitWeightedBCases(t) {
		t.Run(highWeightedBCaseName(tt), func(t *testing.T) {
			data := highExplicitWeightedBFixture(t, tt)
			assertHighExplicitWeightedBFixtureSyntax(t, data, tt)
			frames, err := NewDecoder().DecodeAnnexBFrames(data)
			if err != nil {
				t.Fatalf("DecodeAnnexBFrames: %v", err)
			}
			assertHighWeightedBFrames(t, frames, tt)
		})
	}
}

func TestDecodeAVCHigh1214ExplicitWeightedBFrames(t *testing.T) {
	for _, tt := range high1214ExplicitWeightedBCases(t) {
		t.Run(highWeightedBCaseName(tt), func(t *testing.T) {
			data := highExplicitWeightedBFixture(t, tt)
			assertHighExplicitWeightedBFixtureSyntax(t, data, tt)
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

func TestDecodeConfiguredAVCHigh1214ExplicitWeightedBFramesAcrossSamplesFlush(t *testing.T) {
	for _, tt := range high1214ExplicitWeightedBCases(t) {
		t.Run(highWeightedBCaseName(tt), func(t *testing.T) {
			data := highExplicitWeightedBFixture(t, tt)
			assertHighExplicitWeightedBFixtureSyntax(t, data, tt)
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
				assertHighWeightedBFrames(t, frames, tt)
			}
		})
	}
}

func TestFFmpegRawVideoFrameMD5OracleHigh1214ExplicitWeightedB(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	for _, tt := range high1214ExplicitWeightedBCases(t) {
		t.Run(highWeightedBCaseName(tt), func(t *testing.T) {
			data := highExplicitWeightedBFixture(t, tt)
			assertHighExplicitWeightedBFixtureSyntax(t, data, tt)
			assertFFmpegHighWeightedBRawVideoOracle(t, data, tt)
		})
	}
}

func high1214ExplicitWeightedBCases(t *testing.T) []highWeightedBCase {
	t.Helper()
	expected := high1214ExplicitWeightedBExpected()
	base := high1214ExplicitWeightedBBaseCases()
	out := make([]highWeightedBCase, 0, len(base))
	for _, tt := range base {
		got, ok := expected[highExplicitWeightedBExpectedKey(tt)]
		if !ok {
			t.Fatalf("missing explicit weighted-B expectation for High%d %s", tt.bitDepth, tt.name)
		}
		tt.bitstreamMD5 = got.bitstreamMD5
		tt.frameMD5 = append([]string(nil), got.frameMD5...)
		tt.rawVideoMD5 = got.rawVideoMD5
		out = append(out, tt)
	}
	return out
}

func high1214ExplicitWeightedBBaseCases() []highWeightedBCase {
	base := high1214ImplicitWeightedBCases()
	out := make([]highWeightedBCase, 0, len(base))
	for _, tt := range base {
		tt.name = "explicit-" + tt.name
		tt.bitstreamMD5 = ""
		tt.frameMD5 = nil
		tt.rawVideoMD5 = ""
		out = append(out, tt)
	}
	return out
}

func highExplicitWeightedBExpectedKey(tt highWeightedBCase) string {
	return fmt.Sprintf("high%d-%s", tt.bitDepth, tt.name)
}

func highExplicitWeightedBFixture(t *testing.T, tt highWeightedBCase) []byte {
	t.Helper()
	out := highExplicitWeightedBRewriteSource(t, tt)
	sum := md5.Sum(out)
	if got := hex.EncodeToString(sum[:]); got != tt.bitstreamMD5 {
		t.Fatalf("High%d explicit weighted B generated bitstream md5 = %s, want %s", tt.bitDepth, got, tt.bitstreamMD5)
	}
	return out
}

func highExplicitWeightedBRewriteSource(t *testing.T, tt highWeightedBCase) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "h264", tt.sourceFile))
	if err != nil {
		t.Fatalf("read %s: %v", tt.sourceFile, err)
	}
	return highExplicitWeightedBRewriteAnnexB(t, data, tt.bitDepth, tt.mode2Deblock)
}

func highExplicitWeightedBRewriteAnnexB(t *testing.T, data []byte, bitDepth int, mode2Deblock bool) []byte {
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
				sourceSPS, err := h264.DecodeSPS(rbsp)
				if err != nil {
					t.Fatalf("decode source SPS: %v", err)
				}
				sourceSPSList[sourceSPS.SPSID] = sourceSPS
				raw = highCABACBRewriteSPSRaw(t, raw, bitDepth)
				rewrittenSPS, err := h264.DecodeSPS(high14CABACBEBSPToRBSP(raw[1:]))
				if err != nil {
					t.Fatalf("decode rewritten SPS: %v", err)
				}
				rewrittenSPSList[rewrittenSPS.SPSID] = rewrittenSPS
			case h264.NALPPS:
				sourcePPS, err := h264.DecodePPS(rbsp, &sourceSPSList)
				if err != nil {
					t.Fatalf("decode source PPS: %v", err)
				}
				sourcePPSList[sourcePPS.PPSID] = sourcePPS
				raw = highExplicitWeightedBRewritePPSRaw(t, raw)
				rewrittenPPS, err := h264.DecodePPS(high14CABACBEBSPToRBSP(raw[1:]), &rewrittenSPSList)
				if err != nil {
					t.Fatalf("decode rewritten PPS: %v", err)
				}
				rewrittenPPSList[rewrittenPPS.PPSID] = rewrittenPPS
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
				if sh.SliceTypeNoS == h264.PictureTypeB {
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

func highExplicitWeightedBRewritePPSRaw(t *testing.T, raw []byte) []byte {
	t.Helper()
	if len(raw) < 2 {
		t.Fatal("invalid PPS NAL")
	}
	rbsp := high14CABACBEBSPToRBSP(raw[1:])
	bits := high14CABACBBits(rbsp)
	pos := 0
	high14CABACBReadUEBits(t, bits, &pos)
	high14CABACBReadUEBits(t, bits, &pos)
	high14CABACBReadFixedBits(t, bits, &pos, 1)
	high14CABACBReadFixedBits(t, bits, &pos, 1)
	sliceGroups, _ := high14CABACBReadUEBits(t, bits, &pos)
	if sliceGroups != 0 {
		t.Fatalf("source PPS slice groups = %d, want 0", sliceGroups)
	}
	high14CABACBReadUEBits(t, bits, &pos)
	high14CABACBReadUEBits(t, bits, &pos)
	weightedPredStart := pos
	weightedPred, _ := high14CABACBReadFixedBits(t, bits, &pos, 1)
	weightedBipredStart := pos
	weightedBipred, _ := high14CABACBReadFixedBits(t, bits, &pos, 2)
	if weightedPred != 0 || weightedBipred != 2 {
		t.Fatalf("source PPS weighted pred/bipred = %d/%d, want 0/2", weightedPred, weightedBipred)
	}
	_ = weightedPredStart
	rewritten := bits[:weightedBipredStart] + "01" + bits[weightedBipredStart+2:]
	return append([]byte{raw[0]}, high14CABACBRBSPToEBSP(high14CABACBPackWholeBits(t, rewritten))...)
}

func highExplicitWeightedBRewriteSliceRaw(t *testing.T, raw []byte, sh *h264.SliceHeader) []byte {
	t.Helper()
	if len(raw) < 2 || sh == nil || sh.PPS == nil || sh.SPS == nil || sh.SliceTypeNoS != h264.PictureTypeB {
		t.Fatal("invalid explicit weighted-B slice rewrite input")
	}
	rbsp := high14CABACBEBSPToRBSP(raw[1:])
	bits := high14CABACBBits(rbsp)
	insertAt := highExplicitWeightedBPredWeightInsertPoint(t, bits, sh)
	predWeight := highExplicitWeightedBPredWeightTableBits(t, sh)
	var rewritten string
	if sh.PPS.CABAC != 0 {
		_, _, headerEnd := highCABACBSliceDeblockRange(t, bits, sh)
		origTailStart := (headerEnd + 7) &^ 7
		for _, ch := range bits[headerEnd:origTailStart] {
			if ch != '1' {
				t.Fatalf("CABAC alignment bit = %q, want '1'", ch)
			}
		}
		header := bits[:insertAt] + predWeight + bits[insertAt:headerEnd]
		if mod := len(header) % 8; mod != 0 {
			header += strings.Repeat("1", 8-mod)
		}
		rewritten = header + bits[origTailStart:]
	} else {
		rewritten = bits[:insertAt] + predWeight + bits[insertAt:]
		if mod := len(rewritten) % 8; mod != 0 {
			rewritten += strings.Repeat("0", 8-mod)
		}
	}
	return append([]byte{raw[0]}, high14CABACBRBSPToEBSP(high14CABACBPackWholeBits(t, rewritten))...)
}

func highExplicitWeightedBPredWeightInsertPoint(t *testing.T, bits string, sh *h264.SliceHeader) int {
	t.Helper()
	pos := 0
	high14CABACBReadUEBits(t, bits, &pos)
	sliceTypeCode, _ := high14CABACBReadUEBits(t, bits, &pos)
	if sliceTypeCode > 9 {
		t.Fatalf("slice_type = %d, want <= 9", sliceTypeCode)
	}
	high14CABACBReadUEBits(t, bits, &pos)
	high14CABACBReadFixedBits(t, bits, &pos, int(sh.SPS.Log2MaxFrameNum))
	if sh.SPS.FrameMBSOnlyFlag == 0 {
		fieldPicFlag, _ := high14CABACBReadFixedBits(t, bits, &pos, 1)
		if fieldPicFlag != 0 {
			high14CABACBReadFixedBits(t, bits, &pos, 1)
		}
	}
	if sh.NALType == h264.NALIDRSlice {
		high14CABACBReadUEBits(t, bits, &pos)
	}
	if sh.SPS.PocType == 0 {
		high14CABACBReadFixedBits(t, bits, &pos, int(sh.SPS.Log2MaxPocLSB))
		if sh.PPS.PicOrderPresent == 1 && sh.PictureStructure == h264.PictureFrame {
			high14CABACBReadSEBits(t, bits, &pos)
		}
	}
	if sh.SPS.PocType == 1 && sh.SPS.DeltaPicOrderAlwaysZeroFlag == 0 {
		high14CABACBReadSEBits(t, bits, &pos)
		if sh.PPS.PicOrderPresent == 1 && sh.PictureStructure == h264.PictureFrame {
			high14CABACBReadSEBits(t, bits, &pos)
		}
	}
	if sh.PPS.RedundantPicCntPresent != 0 {
		high14CABACBReadUEBits(t, bits, &pos)
	}
	high14CABACBReadFixedBits(t, bits, &pos, 1)
	override, _ := high14CABACBReadFixedBits(t, bits, &pos, 1)
	if override != 0 {
		high14CABACBReadUEBits(t, bits, &pos)
		high14CABACBReadUEBits(t, bits, &pos)
	}
	highCABACBSkipRefPicListReordering(t, bits, &pos, sh)
	return pos
}

func highExplicitWeightedBPredWeightTableBits(t *testing.T, sh *h264.SliceHeader) string {
	t.Helper()
	if sh == nil || sh.SPS == nil || sh.SPS.ChromaFormatIDC == 0 || sh.ListCount != 2 {
		t.Fatalf("explicit weighted-B source shape = chroma %v lists %v, want chroma B with two lists", sh.SPS, sh.ListCount)
	}
	bits := high14CABACBUEBits(1) + high14CABACBUEBits(1)
	for list := int32(0); list < sh.ListCount; list++ {
		for ref := uint32(0); ref < sh.RefCount[list]; ref++ {
			lumaWeight, lumaOffset := int32(3), int32(1)
			chroma0Weight, chroma0Offset := int32(3), int32(1)
			chroma1Weight, chroma1Offset := int32(3), int32(-1)
			if list == 1 {
				lumaWeight, lumaOffset = 1, -1
				chroma0Weight, chroma0Offset = 1, -1
				chroma1Weight, chroma1Offset = 1, 1
			}
			bits += "1" + high14CABACBSEBits(lumaWeight) + high14CABACBSEBits(lumaOffset)
			bits += "1" +
				high14CABACBSEBits(chroma0Weight) + high14CABACBSEBits(chroma0Offset) +
				high14CABACBSEBits(chroma1Weight) + high14CABACBSEBits(chroma1Offset)
		}
	}
	return bits
}

func high14CABACBSEBits(v int32) string {
	if v <= 0 {
		return high14CABACBUEBits(uint32(-2 * v))
	}
	return high14CABACBUEBits(uint32(2*v - 1))
}

func assertHighExplicitWeightedBFixtureSyntax(t *testing.T, data []byte, tt highWeightedBCase) {
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
				pps.WeightedBipredIDC != 1 || pps.RefCount != wantRefs {
				t.Fatalf("PPS = nal[%d] cabac/8x8/weights/refs = %d/%d/%d/%d/%v, want cabac=%d explicit-B refs=%v",
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
				if sh.ListCount != 2 || sh.RefCount[0] == 0 || sh.RefCount[1] == 0 ||
					sh.PredWeightTable.UseWeight != 1 || sh.PredWeightTable.UseWeightChroma != 1 {
					t.Fatalf("B slice lists/refs/weights = %d/%v/%d/%d, want explicit weights",
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

func high1214ExplicitWeightedBExpected() map[string]highExplicitWeightedBExpected {
	return map[string]highExplicitWeightedBExpected{
		"high12-explicit-cavlc-no-deblock":                             {bitstreamMD5: "036f9ed08680d64eb2c87eb8828c21a9", frameMD5: []string{"d4753b9733af2865470fb72f96a37071", "5ec251763d3925b98335d847284bd934", "5ec251763d3925b98335d847284bd934", "e7ac9c6c5627f3112acde8625c52f8cf", "3e64a4ea01c4befde1217bf6d295a034"}, rawVideoMD5: "a9475845ded793019b769ecd55859af5"},
		"high12-explicit-cabac-no-deblock":                             {bitstreamMD5: "50d226cb59b80a3713e40db672af795c", frameMD5: []string{"d4753b9733af2865470fb72f96a37071", "5ec251763d3925b98335d847284bd934", "5ec251763d3925b98335d847284bd934", "e7ac9c6c5627f3112acde8625c52f8cf", "3e64a4ea01c4befde1217bf6d295a034"}, rawVideoMD5: "a9475845ded793019b769ecd55859af5"},
		"high12-explicit-cavlc-mode1-deblock":                          {bitstreamMD5: "8d217aa45a176c7f6131b8256d7d92a2", frameMD5: []string{"91750ecee8bda0d5c023f6254707914b", "5887d62d49e5cd70b94ba83c3a8dfb99", "de2273391e36f7b39d138c9beeda6b7b", "9f2ec56947f8b92de7fcd2a34ac04ab8", "827d6e4eaf327ddf44be7314cc7902a3"}, rawVideoMD5: "e67ca23a3ab5a8501243d95bb3951c84"},
		"high12-explicit-cavlc-mode2-deblock":                          {bitstreamMD5: "e485acb656944ac2d0492e4ee568029d", frameMD5: []string{"91750ecee8bda0d5c023f6254707914b", "5887d62d49e5cd70b94ba83c3a8dfb99", "de2273391e36f7b39d138c9beeda6b7b", "9f2ec56947f8b92de7fcd2a34ac04ab8", "827d6e4eaf327ddf44be7314cc7902a3"}, rawVideoMD5: "e67ca23a3ab5a8501243d95bb3951c84"},
		"high12-explicit-cabac-mode1-deblock":                          {bitstreamMD5: "9911fb456aa29373774164d506223325", frameMD5: []string{"f94f0f324a3229363360256262647517", "e5777f951cf80d80c15fce3ddac59f61", "9f518dd93980b7ee93ef585e670c9aeb", "43cdcf7832e78060f730127894031732", "a5db27d3c4a732850d8d8bacf2d221ed"}, rawVideoMD5: "954d6cede3c8311fecddeb45c3fcd31a"},
		"high12-explicit-cabac-mode2-deblock":                          {bitstreamMD5: "7a52c588bb6d01f0d1f749974eb01986", frameMD5: []string{"f94f0f324a3229363360256262647517", "e5777f951cf80d80c15fce3ddac59f61", "9f518dd93980b7ee93ef585e670c9aeb", "43cdcf7832e78060f730127894031732", "a5db27d3c4a732850d8d8bacf2d221ed"}, rawVideoMD5: "954d6cede3c8311fecddeb45c3fcd31a"},
		"high12-explicit-cavlc-direct-sub-b8x8-temporal-mode1-deblock": {bitstreamMD5: "5bc01c143865688aaaed645191617ec5", frameMD5: []string{"d4753b9733af2865470fb72f96a37071", "d4753b9733af2865470fb72f96a37071", "d4753b9733af2865470fb72f96a37071"}, rawVideoMD5: "c5844f8a45006553335c482758ad0f49"},
		"high12-explicit-cavlc-direct-sub-b8x8-temporal-mode2-deblock": {bitstreamMD5: "f37b4f49e29c0347b3c5789cc27124ce", frameMD5: []string{"d4753b9733af2865470fb72f96a37071", "d4753b9733af2865470fb72f96a37071", "d4753b9733af2865470fb72f96a37071"}, rawVideoMD5: "c5844f8a45006553335c482758ad0f49"},
		"high12-explicit-cavlc-direct-sub-b8x8-spatial-mode1-deblock":  {bitstreamMD5: "f32191e66e811640ccdecd0c3b59c0ae", frameMD5: []string{"d4753b9733af2865470fb72f96a37071", "d4753b9733af2865470fb72f96a37071", "d4753b9733af2865470fb72f96a37071"}, rawVideoMD5: "c5844f8a45006553335c482758ad0f49"},
		"high12-explicit-cavlc-direct-sub-b8x8-spatial-mode2-deblock":  {bitstreamMD5: "bb8fcb1793c0153d83dfd0a30e29784a", frameMD5: []string{"d4753b9733af2865470fb72f96a37071", "d4753b9733af2865470fb72f96a37071", "d4753b9733af2865470fb72f96a37071"}, rawVideoMD5: "c5844f8a45006553335c482758ad0f49"},
		"high12-explicit-cavlc-direct-sub-b4x4-temporal-mode1-deblock": {bitstreamMD5: "eafb62120f75e357bf7aab186a435787", frameMD5: []string{"d4753b9733af2865470fb72f96a37071", "d4753b9733af2865470fb72f96a37071", "d4753b9733af2865470fb72f96a37071"}, rawVideoMD5: "c5844f8a45006553335c482758ad0f49"},
		"high12-explicit-cavlc-direct-sub-b4x4-temporal-mode2-deblock": {bitstreamMD5: "92d11c662ebb91b9fa5734197a67a33f", frameMD5: []string{"d4753b9733af2865470fb72f96a37071", "d4753b9733af2865470fb72f96a37071", "d4753b9733af2865470fb72f96a37071"}, rawVideoMD5: "c5844f8a45006553335c482758ad0f49"},
		"high12-explicit-cavlc-direct-sub-b4x4-spatial-mode1-deblock":  {bitstreamMD5: "ea30584bcaf617139825c6c05339519f", frameMD5: []string{"d4753b9733af2865470fb72f96a37071", "d4753b9733af2865470fb72f96a37071", "d4753b9733af2865470fb72f96a37071"}, rawVideoMD5: "c5844f8a45006553335c482758ad0f49"},
		"high12-explicit-cavlc-direct-sub-b4x4-spatial-mode2-deblock":  {bitstreamMD5: "d961c83a8c3cf67ad313281a813bcec0", frameMD5: []string{"d4753b9733af2865470fb72f96a37071", "d4753b9733af2865470fb72f96a37071", "d4753b9733af2865470fb72f96a37071"}, rawVideoMD5: "c5844f8a45006553335c482758ad0f49"},
		"high12-explicit-cabac-direct-sub-b8x8-temporal-mode1-deblock": {bitstreamMD5: "46140ca18f8487adec8ddeb3ac46ac68", frameMD5: []string{"d4753b9733af2865470fb72f96a37071", "d4753b9733af2865470fb72f96a37071", "d4753b9733af2865470fb72f96a37071"}, rawVideoMD5: "c5844f8a45006553335c482758ad0f49"},
		"high12-explicit-cabac-direct-sub-b8x8-temporal-mode2-deblock": {bitstreamMD5: "8261fc6c1ec490770d94ce380055ab2b", frameMD5: []string{"d4753b9733af2865470fb72f96a37071", "d4753b9733af2865470fb72f96a37071", "d4753b9733af2865470fb72f96a37071"}, rawVideoMD5: "c5844f8a45006553335c482758ad0f49"},
		"high12-explicit-cabac-direct-sub-b8x8-spatial-mode1-deblock":  {bitstreamMD5: "ffa999c35c302a97cecec87c115144b7", frameMD5: []string{"d4753b9733af2865470fb72f96a37071", "d4753b9733af2865470fb72f96a37071", "d4753b9733af2865470fb72f96a37071"}, rawVideoMD5: "c5844f8a45006553335c482758ad0f49"},
		"high12-explicit-cabac-direct-sub-b8x8-spatial-mode2-deblock":  {bitstreamMD5: "33955443f95d5358141db0df79a89c9e", frameMD5: []string{"d4753b9733af2865470fb72f96a37071", "d4753b9733af2865470fb72f96a37071", "d4753b9733af2865470fb72f96a37071"}, rawVideoMD5: "c5844f8a45006553335c482758ad0f49"},
		"high12-explicit-cabac-direct-sub-b4x4-temporal-mode1-deblock": {bitstreamMD5: "3d229f6440c30154e8390c91b33990e9", frameMD5: []string{"d4753b9733af2865470fb72f96a37071", "d4753b9733af2865470fb72f96a37071", "d4753b9733af2865470fb72f96a37071"}, rawVideoMD5: "c5844f8a45006553335c482758ad0f49"},
		"high12-explicit-cabac-direct-sub-b4x4-temporal-mode2-deblock": {bitstreamMD5: "eba8f4db89bad39b34f35cc4ca2dfd63", frameMD5: []string{"d4753b9733af2865470fb72f96a37071", "d4753b9733af2865470fb72f96a37071", "d4753b9733af2865470fb72f96a37071"}, rawVideoMD5: "c5844f8a45006553335c482758ad0f49"},
		"high12-explicit-cabac-direct-sub-b4x4-spatial-mode1-deblock":  {bitstreamMD5: "6f0d71ef38a1a1a6210ff1a21aeec275", frameMD5: []string{"d4753b9733af2865470fb72f96a37071", "d4753b9733af2865470fb72f96a37071", "d4753b9733af2865470fb72f96a37071"}, rawVideoMD5: "c5844f8a45006553335c482758ad0f49"},
		"high12-explicit-cabac-direct-sub-b4x4-spatial-mode2-deblock":  {bitstreamMD5: "a21bd9c543502e1829148911a9dc837c", frameMD5: []string{"d4753b9733af2865470fb72f96a37071", "d4753b9733af2865470fb72f96a37071", "d4753b9733af2865470fb72f96a37071"}, rawVideoMD5: "c5844f8a45006553335c482758ad0f49"},
		"high12-explicit-cavlc-partitioned-b16x8-no-deblock":           {bitstreamMD5: "a3fbe476e85a1f1227aafd9c0c1be4b8", frameMD5: []string{"ade4ad7e683293ad68a2dc3c9361278b", "27948e12d619427a0496b00c20ee7a1b", "055a705b333032ad172ed257dcb0252b", "b7b0e1d413b0cc5b7a62dcab64640155", "d9a78ae6ec163f39bffd9ef2ff730c84"}, rawVideoMD5: "355de1c1d9ab60863c6e92bd0db163a0"},
		"high12-explicit-cabac-partitioned-b16x8-no-deblock":           {bitstreamMD5: "da4124bfbefd8d481fde4805c24d1f75", frameMD5: []string{"8951fb646c0a7ec597842fc225f6dbc3", "1d637829b93169148daf01710b27aafe", "cbb3b3128a9e9b2e202db86d49051e9f", "6c39e7b5944462bb24e3eb4227f79c3a", "5ac684e18682d64a70e58704a0868497"}, rawVideoMD5: "a1e45d03bbb27ff5b267f4face0cc2d7"},
		"high12-explicit-cavlc-partitioned-b8x16-no-deblock":           {bitstreamMD5: "20246f4e64c62104df5de7f94b22c469", frameMD5: []string{"c5ed38b6d73a131d7eff20f34ff7a2f9", "f55762f52f6becd8bf514b3d64babe90", "e7dd5e43f74f0c387dc1cd3208a344d9", "54c6b7acd225947494942db4d4f39bc8", "9db0994f97fc8679ccb22ff8c7b36ff6"}, rawVideoMD5: "00842221446bb2e7cbd4b65ad2cc73ac"},
		"high12-explicit-cabac-partitioned-b8x16-no-deblock":           {bitstreamMD5: "86ab107366a45fcbde62b3446bfabd6f", frameMD5: []string{"50d103ccff955b9215971dfaf40d8f67", "6e7d6ce74a1fdeef76f032895f016095", "e4aa95295c3f6dbac8b7eb3cb5ec9102", "616864c271c6dfd30102a52557aed7b0", "bbaadd17b1213f8e9507ded718869507"}, rawVideoMD5: "0ce60c6a476c7700c70dcef13f846819"},
		"high12-explicit-cavlc-partitioned-b8x8-no-deblock":            {bitstreamMD5: "068b816c8a3e80ae60db18c2a8f0c894", frameMD5: []string{"9e5e8a1cf791aa907c3e4d4041e8f869", "cbdea0c91147118b1e66c14715a42e53", "4f3d8d8aa5bb70c024420227ea82733f", "ce26dbc242ec57f5d3ec074ddc12a0b7", "fda23ed601ae8f581638dc63fd333629"}, rawVideoMD5: "eff9132f50e4c3ffe4c822feeeeebccc"},
		"high12-explicit-cabac-partitioned-b8x8-no-deblock":            {bitstreamMD5: "3ce29cfd913a540d2709204616c498f6", frameMD5: []string{"9e5e8a1cf791aa907c3e4d4041e8f869", "cbdea0c91147118b1e66c14715a42e53", "19cfe507b2673ca14681781133a90816", "ce26dbc242ec57f5d3ec074ddc12a0b7", "fda23ed601ae8f581638dc63fd333629"}, rawVideoMD5: "2e5b7b401f2ed5ad0397a1406b78be91"},
		"high12-explicit-cavlc-partitioned-b16x8-mode1-deblock":        {bitstreamMD5: "39c146a373bc6a6fc4d1dc70f9ade137", frameMD5: []string{"271fc9eab6e0d7c98af20f9ecffd0491", "20978fa800e66dabd0a7e5932bde8f9f", "f6887492222047961f3de953c1384177", "11bc3a789dd0b701afa7e4e9e5c137c9", "8bbd70c55f2113ce370f0a5c96b0ac09"}, rawVideoMD5: "c7f05fd6d67986e4114433f16016ae7f"},
		"high12-explicit-cavlc-partitioned-b16x8-mode2-deblock":        {bitstreamMD5: "1640d4ddaa69d48c091c4c9e5fbf7fe0", frameMD5: []string{"271fc9eab6e0d7c98af20f9ecffd0491", "20978fa800e66dabd0a7e5932bde8f9f", "f6887492222047961f3de953c1384177", "11bc3a789dd0b701afa7e4e9e5c137c9", "8bbd70c55f2113ce370f0a5c96b0ac09"}, rawVideoMD5: "c7f05fd6d67986e4114433f16016ae7f"},
		"high12-explicit-cabac-partitioned-b16x8-mode1-deblock":        {bitstreamMD5: "abda2e4aa9ea7410506bfb0c632f57c5", frameMD5: []string{"c2bd0dd90f1cf7ed33424c06f47454a5", "3a1058d0e91f19b7abd811eed81b2ded", "0e1534d1976c21401a8e0e5b2c8f056a", "11bc3a789dd0b701afa7e4e9e5c137c9", "8bbd70c55f2113ce370f0a5c96b0ac09"}, rawVideoMD5: "fe007f738cc2198e308027ff0e206a51"},
		"high12-explicit-cabac-partitioned-b16x8-mode2-deblock":        {bitstreamMD5: "394abcbf933dd5f6515ec30e0c4c5453", frameMD5: []string{"c2bd0dd90f1cf7ed33424c06f47454a5", "3a1058d0e91f19b7abd811eed81b2ded", "0e1534d1976c21401a8e0e5b2c8f056a", "11bc3a789dd0b701afa7e4e9e5c137c9", "8bbd70c55f2113ce370f0a5c96b0ac09"}, rawVideoMD5: "fe007f738cc2198e308027ff0e206a51"},
		"high12-explicit-cavlc-partitioned-b8x16-mode1-deblock":        {bitstreamMD5: "f1ce0a69718792a9c19bdf7677f5f341", frameMD5: []string{"ff5aaa5c613fc0046f4224b7b27b68b6", "0e8a8a9322409b59f47df762338b1aac", "378997aef1034e94ffaa22e9bb83270c", "3740befb8f4f876023e6c209d55b56e1", "f68edd1b4866284784aeb99943da8b4e"}, rawVideoMD5: "25f38e924934a0de916daa5f71f31d14"},
		"high12-explicit-cavlc-partitioned-b8x16-mode2-deblock":        {bitstreamMD5: "7f6b70f71bd800e56cb7a097c7119aa9", frameMD5: []string{"ff5aaa5c613fc0046f4224b7b27b68b6", "0e8a8a9322409b59f47df762338b1aac", "378997aef1034e94ffaa22e9bb83270c", "3740befb8f4f876023e6c209d55b56e1", "f68edd1b4866284784aeb99943da8b4e"}, rawVideoMD5: "25f38e924934a0de916daa5f71f31d14"},
		"high12-explicit-cabac-partitioned-b8x16-mode1-deblock":        {bitstreamMD5: "e1970866535210418a31c9aa5477bf3b", frameMD5: []string{"08b7418359830aae9eb5778f08a37a81", "c30c59d97c2530efef8db4ad76e16c27", "05a72553b1204866a1d6a795d6c1fce9", "04d9f462359b99bbdca7de8e9f53e75a", "412ef787cad8ff1f0da2948ff119bc67"}, rawVideoMD5: "2347b0f311184775fdfe510057ddfab5"},
		"high12-explicit-cabac-partitioned-b8x16-mode2-deblock":        {bitstreamMD5: "a826dc1443a094998eb030cfc6266b19", frameMD5: []string{"08b7418359830aae9eb5778f08a37a81", "c30c59d97c2530efef8db4ad76e16c27", "05a72553b1204866a1d6a795d6c1fce9", "04d9f462359b99bbdca7de8e9f53e75a", "412ef787cad8ff1f0da2948ff119bc67"}, rawVideoMD5: "2347b0f311184775fdfe510057ddfab5"},
		"high12-explicit-cavlc-partitioned-b8x8-mode1-deblock":         {bitstreamMD5: "aebeac8a8d867aa8e143c135d3c53841", frameMD5: []string{"69c7144d64f8fcc0994be3f0cfbe4b5d", "51b70124a32a2ea5cf4537de8ad8a4dc", "4cee9c43abca3bc4580d3c07bdc8a445", "a170c75ad560003e135f037b8df24609", "68dddd659051cf9922494b235f4bb5d7"}, rawVideoMD5: "4265e66916cf05f3adb579e50343ff71"},
		"high12-explicit-cavlc-partitioned-b8x8-mode2-deblock":         {bitstreamMD5: "0deac7fd0ac5ee97db276193283837ad", frameMD5: []string{"69c7144d64f8fcc0994be3f0cfbe4b5d", "51b70124a32a2ea5cf4537de8ad8a4dc", "4cee9c43abca3bc4580d3c07bdc8a445", "a170c75ad560003e135f037b8df24609", "68dddd659051cf9922494b235f4bb5d7"}, rawVideoMD5: "4265e66916cf05f3adb579e50343ff71"},
		"high12-explicit-cabac-partitioned-b8x8-mode1-deblock":         {bitstreamMD5: "a5cbbbd83e09a8808f22c6ea39cd6c39", frameMD5: []string{"69c7144d64f8fcc0994be3f0cfbe4b5d", "8b006304d27411d0a2b536341ac5a983", "f1ab5da3842df522f21ba55d3dfc2052", "a170c75ad560003e135f037b8df24609", "d1cab7cbb0d8cfda8d2a80c683d6cf41"}, rawVideoMD5: "4ba8a944286d928102ce90ee3524e364"},
		"high12-explicit-cabac-partitioned-b8x8-mode2-deblock":         {bitstreamMD5: "3d0205505b7f93d4bac1fe88d92d3ae2", frameMD5: []string{"69c7144d64f8fcc0994be3f0cfbe4b5d", "8b006304d27411d0a2b536341ac5a983", "f1ab5da3842df522f21ba55d3dfc2052", "a170c75ad560003e135f037b8df24609", "d1cab7cbb0d8cfda8d2a80c683d6cf41"}, rawVideoMD5: "4ba8a944286d928102ce90ee3524e364"},
		"high14-explicit-cavlc-no-deblock":                             {bitstreamMD5: "b17667ebf188f80591ab00c2aa476460", frameMD5: []string{"6d3514a30f506561e144447d287270ab", "4067c8e01b5d816de6e8518f3bb29414", "4067c8e01b5d816de6e8518f3bb29414", "8a776a45cff9c94f884ccabea68f5e94", "d3641da2bd4bf8a1289cd4e1f6c4f9f7"}, rawVideoMD5: "00e7a4e2e1f5e3cd4e7b10dc9bf92ced"},
		"high14-explicit-cabac-no-deblock":                             {bitstreamMD5: "8de6f2df0d36f99dfd407a5e86471a73", frameMD5: []string{"6d3514a30f506561e144447d287270ab", "4067c8e01b5d816de6e8518f3bb29414", "4067c8e01b5d816de6e8518f3bb29414", "8a776a45cff9c94f884ccabea68f5e94", "d3641da2bd4bf8a1289cd4e1f6c4f9f7"}, rawVideoMD5: "00e7a4e2e1f5e3cd4e7b10dc9bf92ced"},
		"high14-explicit-cavlc-mode1-deblock":                          {bitstreamMD5: "6405e287bae7f53c7e88395f045e8ca7", frameMD5: []string{"3191e00b8ace54980df9c2b2c9e2c5e2", "a39d70e979fee4132bd7c937e6e16b7d", "55c254947bd15f0614db1fe9032e410f", "2eb2f905a6b4739ee22e0bb138b71a6a", "aadd3e1edceb4577255d9720e4372b04"}, rawVideoMD5: "4bb6cbf2bdf63feb8c9f9280d37508b7"},
		"high14-explicit-cavlc-mode2-deblock":                          {bitstreamMD5: "567f01223450570f673f370bb7a99b09", frameMD5: []string{"3191e00b8ace54980df9c2b2c9e2c5e2", "a39d70e979fee4132bd7c937e6e16b7d", "55c254947bd15f0614db1fe9032e410f", "2eb2f905a6b4739ee22e0bb138b71a6a", "aadd3e1edceb4577255d9720e4372b04"}, rawVideoMD5: "4bb6cbf2bdf63feb8c9f9280d37508b7"},
		"high14-explicit-cabac-mode1-deblock":                          {bitstreamMD5: "faa2a72a3b266816069016f16265d6a2", frameMD5: []string{"ba2c72108ade4f4fb88f182fd25e9c15", "26e742d60427187daca631bb702d0af4", "12f9b389b4616770c751a473e6626230", "7b6204ca8ef61175eea1ac577f1022b6", "bd654782846633b1429650d60f451ada"}, rawVideoMD5: "8e621d59255e9e913fff8fe5723c8aaa"},
		"high14-explicit-cabac-mode2-deblock":                          {bitstreamMD5: "9f925a44f81ac5dba1d81d686a14329b", frameMD5: []string{"ba2c72108ade4f4fb88f182fd25e9c15", "26e742d60427187daca631bb702d0af4", "12f9b389b4616770c751a473e6626230", "7b6204ca8ef61175eea1ac577f1022b6", "bd654782846633b1429650d60f451ada"}, rawVideoMD5: "8e621d59255e9e913fff8fe5723c8aaa"},
		"high14-explicit-cavlc-direct-sub-b8x8-temporal-mode1-deblock": {bitstreamMD5: "9915ae3a9e5e8a84ee15e14a29141e20", frameMD5: []string{"6d3514a30f506561e144447d287270ab", "6d3514a30f506561e144447d287270ab", "6d3514a30f506561e144447d287270ab"}, rawVideoMD5: "1f17900b95954131ed58ae42da301444"},
		"high14-explicit-cavlc-direct-sub-b8x8-temporal-mode2-deblock": {bitstreamMD5: "cf39645d25ae8a8abb1e7006b31c9ec1", frameMD5: []string{"6d3514a30f506561e144447d287270ab", "6d3514a30f506561e144447d287270ab", "6d3514a30f506561e144447d287270ab"}, rawVideoMD5: "1f17900b95954131ed58ae42da301444"},
		"high14-explicit-cavlc-direct-sub-b8x8-spatial-mode1-deblock":  {bitstreamMD5: "c9f2856bc66064849b132249a23c856b", frameMD5: []string{"6d3514a30f506561e144447d287270ab", "6d3514a30f506561e144447d287270ab", "6d3514a30f506561e144447d287270ab"}, rawVideoMD5: "1f17900b95954131ed58ae42da301444"},
		"high14-explicit-cavlc-direct-sub-b8x8-spatial-mode2-deblock":  {bitstreamMD5: "d988b4cf691c06fed01bde73d4839bce", frameMD5: []string{"6d3514a30f506561e144447d287270ab", "6d3514a30f506561e144447d287270ab", "6d3514a30f506561e144447d287270ab"}, rawVideoMD5: "1f17900b95954131ed58ae42da301444"},
		"high14-explicit-cavlc-direct-sub-b4x4-temporal-mode1-deblock": {bitstreamMD5: "02edb8580df8909144012dba19775862", frameMD5: []string{"6d3514a30f506561e144447d287270ab", "6d3514a30f506561e144447d287270ab", "6d3514a30f506561e144447d287270ab"}, rawVideoMD5: "1f17900b95954131ed58ae42da301444"},
		"high14-explicit-cavlc-direct-sub-b4x4-temporal-mode2-deblock": {bitstreamMD5: "c1bcf82eb67b3fc203bea567e4e2e608", frameMD5: []string{"6d3514a30f506561e144447d287270ab", "6d3514a30f506561e144447d287270ab", "6d3514a30f506561e144447d287270ab"}, rawVideoMD5: "1f17900b95954131ed58ae42da301444"},
		"high14-explicit-cavlc-direct-sub-b4x4-spatial-mode1-deblock":  {bitstreamMD5: "6c4076d336b1918b9a208b14f81f0f5e", frameMD5: []string{"6d3514a30f506561e144447d287270ab", "6d3514a30f506561e144447d287270ab", "6d3514a30f506561e144447d287270ab"}, rawVideoMD5: "1f17900b95954131ed58ae42da301444"},
		"high14-explicit-cavlc-direct-sub-b4x4-spatial-mode2-deblock":  {bitstreamMD5: "f5ab7f9d1174b1ba9c65ef0408f84db5", frameMD5: []string{"6d3514a30f506561e144447d287270ab", "6d3514a30f506561e144447d287270ab", "6d3514a30f506561e144447d287270ab"}, rawVideoMD5: "1f17900b95954131ed58ae42da301444"},
		"high14-explicit-cabac-direct-sub-b8x8-temporal-mode1-deblock": {bitstreamMD5: "77cf2e4fd3bd1460c7dd8dcebd6faddb", frameMD5: []string{"6d3514a30f506561e144447d287270ab", "6d3514a30f506561e144447d287270ab", "6d3514a30f506561e144447d287270ab"}, rawVideoMD5: "1f17900b95954131ed58ae42da301444"},
		"high14-explicit-cabac-direct-sub-b8x8-temporal-mode2-deblock": {bitstreamMD5: "f3e5822f5403782992ad7fccab249303", frameMD5: []string{"6d3514a30f506561e144447d287270ab", "6d3514a30f506561e144447d287270ab", "6d3514a30f506561e144447d287270ab"}, rawVideoMD5: "1f17900b95954131ed58ae42da301444"},
		"high14-explicit-cabac-direct-sub-b8x8-spatial-mode1-deblock":  {bitstreamMD5: "4e7ae8bc00db3e89cb4094defa834e36", frameMD5: []string{"6d3514a30f506561e144447d287270ab", "6d3514a30f506561e144447d287270ab", "6d3514a30f506561e144447d287270ab"}, rawVideoMD5: "1f17900b95954131ed58ae42da301444"},
		"high14-explicit-cabac-direct-sub-b8x8-spatial-mode2-deblock":  {bitstreamMD5: "60f89c2e5aca0cc8f90fb853446c24d5", frameMD5: []string{"6d3514a30f506561e144447d287270ab", "6d3514a30f506561e144447d287270ab", "6d3514a30f506561e144447d287270ab"}, rawVideoMD5: "1f17900b95954131ed58ae42da301444"},
		"high14-explicit-cabac-direct-sub-b4x4-temporal-mode1-deblock": {bitstreamMD5: "e1e8ce2ff93afa5c6fa321193652e37a", frameMD5: []string{"6d3514a30f506561e144447d287270ab", "6d3514a30f506561e144447d287270ab", "6d3514a30f506561e144447d287270ab"}, rawVideoMD5: "1f17900b95954131ed58ae42da301444"},
		"high14-explicit-cabac-direct-sub-b4x4-temporal-mode2-deblock": {bitstreamMD5: "377b798aad1bc87e2fb71c0d091ac0e3", frameMD5: []string{"6d3514a30f506561e144447d287270ab", "6d3514a30f506561e144447d287270ab", "6d3514a30f506561e144447d287270ab"}, rawVideoMD5: "1f17900b95954131ed58ae42da301444"},
		"high14-explicit-cabac-direct-sub-b4x4-spatial-mode1-deblock":  {bitstreamMD5: "25d60ee10125e950ddb9ae08ed7cd365", frameMD5: []string{"6d3514a30f506561e144447d287270ab", "6d3514a30f506561e144447d287270ab", "6d3514a30f506561e144447d287270ab"}, rawVideoMD5: "1f17900b95954131ed58ae42da301444"},
		"high14-explicit-cabac-direct-sub-b4x4-spatial-mode2-deblock":  {bitstreamMD5: "ea3d464e8cba8d97013d3acbcde064a6", frameMD5: []string{"6d3514a30f506561e144447d287270ab", "6d3514a30f506561e144447d287270ab", "6d3514a30f506561e144447d287270ab"}, rawVideoMD5: "1f17900b95954131ed58ae42da301444"},
		"high14-explicit-cavlc-partitioned-b16x8-no-deblock":           {bitstreamMD5: "c5f5fdc22afb18b680502a9d378ea1c5", frameMD5: []string{"8f6637034ed7562185064dcfa1b65c1e", "bf561ea2b4cf462635de725efcc2d7c5", "b553ac9f0dd2fe50b43d8e587653740e", "439523466dce0a675d9a6b632286170b", "f0a0446289715444f6e00885fb9082b8"}, rawVideoMD5: "6d41bc9571b4d3aba72be28e1f77c300"},
		"high14-explicit-cabac-partitioned-b16x8-no-deblock":           {bitstreamMD5: "37094fc20a400fdd43e7dceff7a866ad", frameMD5: []string{"93596a9bd5980c195830fc88dfc67da8", "0580f1c7cc6eefa85da5d8fc0d5de64e", "028f43cad320fb9b33de4ae405b9b0e7", "d30faa08b72e9a40047090177a418a3a", "c067a2e153fdbf85816bcc02964715b5"}, rawVideoMD5: "22c4e02b8138390a2ecae2e9f7e59f05"},
		"high14-explicit-cavlc-partitioned-b8x16-no-deblock":           {bitstreamMD5: "204969bb98a2861974be8b74f0651be8", frameMD5: []string{"8c25dcf589eb44488598375c1672c2f8", "8574a2a5b8bd99bdd698a41c544e53fc", "d8f36dfb3d550f48a12800f6788f89d3", "988ce2ab88947b2a3d636cda3af7f196", "cad599c236a3944f891198603c878561"}, rawVideoMD5: "685c086dbf0693950de4d3140033c05c"},
		"high14-explicit-cabac-partitioned-b8x16-no-deblock":           {bitstreamMD5: "ca2ca4d106dcf610107fe818903c9a01", frameMD5: []string{"85350066ebc1709573d3e10d0d0a14bc", "1e85ee962dc6438d31d0eceec87d8191", "275ccd517cc236b8d7b66abf6b165fdc", "157e081e0dbfb1a5d6301e2f863b9881", "903373b18bedb3388fd36f7c8a7d3f86"}, rawVideoMD5: "218c4a20de8c858477eb639ac18babf0"},
		"high14-explicit-cavlc-partitioned-b8x8-no-deblock":            {bitstreamMD5: "adc104b73050b022f9bf74bb21ef6204", frameMD5: []string{"0898fde55c5dac131d8c5f745ffe337f", "1efc511305159b981a7a95ab03583712", "0c35870d2235fcbd2b8af2644eef3663", "b33ece0998aa91831e6b66fd618c3d11", "7fffb6247ec88e0a617cff968ebdeeb0"}, rawVideoMD5: "d1ac7453aa069a1807514dc0da32395e"},
		"high14-explicit-cabac-partitioned-b8x8-no-deblock":            {bitstreamMD5: "21604f9a32acd20e187225d21ca0a4d3", frameMD5: []string{"0898fde55c5dac131d8c5f745ffe337f", "1efc511305159b981a7a95ab03583712", "4c90e788079d7b4c4803443fa9de6db9", "b33ece0998aa91831e6b66fd618c3d11", "7fffb6247ec88e0a617cff968ebdeeb0"}, rawVideoMD5: "3f0af47a073f5883aa206e1fd9b0d629"},
		"high14-explicit-cavlc-partitioned-b16x8-mode1-deblock":        {bitstreamMD5: "04218ad40300be9d26be1007987e59da", frameMD5: []string{"76bba608fb2c663b35187dc6fd5ad541", "52752dcd184ca4a579ddbcedd74c6fdb", "e52eaae9c6f6eb75415656f9d8e08caa", "12208fd0262429f37e5942b9e2d1f9d8", "2d4008fca992537dbf2c34325410aedd"}, rawVideoMD5: "6bea2886abdd07db6464e5e03c6dff03"},
		"high14-explicit-cavlc-partitioned-b16x8-mode2-deblock":        {bitstreamMD5: "801bb872e0a442816bef0cf83ccbde86", frameMD5: []string{"76bba608fb2c663b35187dc6fd5ad541", "52752dcd184ca4a579ddbcedd74c6fdb", "e52eaae9c6f6eb75415656f9d8e08caa", "12208fd0262429f37e5942b9e2d1f9d8", "2d4008fca992537dbf2c34325410aedd"}, rawVideoMD5: "6bea2886abdd07db6464e5e03c6dff03"},
		"high14-explicit-cabac-partitioned-b16x8-mode1-deblock":        {bitstreamMD5: "7b28e4c0e3fff14d95a206270b931d4c", frameMD5: []string{"08cfce46517b1a3fb82d43b66d6c3327", "65172513902a80770a85454189bc25bd", "1eba1474d286fb0913852a9169c3e945", "12208fd0262429f37e5942b9e2d1f9d8", "2d4008fca992537dbf2c34325410aedd"}, rawVideoMD5: "9c1d0a1ca0bf739721f06ceba30ce7bd"},
		"high14-explicit-cabac-partitioned-b16x8-mode2-deblock":        {bitstreamMD5: "c1edf080b4e08d323bedee8f5c8fc535", frameMD5: []string{"08cfce46517b1a3fb82d43b66d6c3327", "65172513902a80770a85454189bc25bd", "1eba1474d286fb0913852a9169c3e945", "12208fd0262429f37e5942b9e2d1f9d8", "2d4008fca992537dbf2c34325410aedd"}, rawVideoMD5: "9c1d0a1ca0bf739721f06ceba30ce7bd"},
		"high14-explicit-cavlc-partitioned-b8x16-mode1-deblock":        {bitstreamMD5: "c20be6e99e40ef8cb132edd5bfceef3d", frameMD5: []string{"00e31a71ce397179f111a49ff859faef", "005aa29997439c9e6594f7d9e8121f21", "4deb156ed96698928b6c2bd019f3999a", "735363c6d23f331ee828c237a8ab1946", "d9b2a282d9bdee1682dba11221ad3fd5"}, rawVideoMD5: "6117a975c49955998f7376d374753483"},
		"high14-explicit-cavlc-partitioned-b8x16-mode2-deblock":        {bitstreamMD5: "9cff0bcc01110f2cc482f0a83a2d12f9", frameMD5: []string{"00e31a71ce397179f111a49ff859faef", "005aa29997439c9e6594f7d9e8121f21", "4deb156ed96698928b6c2bd019f3999a", "735363c6d23f331ee828c237a8ab1946", "d9b2a282d9bdee1682dba11221ad3fd5"}, rawVideoMD5: "6117a975c49955998f7376d374753483"},
		"high14-explicit-cabac-partitioned-b8x16-mode1-deblock":        {bitstreamMD5: "084fe69a71eff1cf91e761f7a6f857e7", frameMD5: []string{"f7e31af362f75c2c72082c963526bd62", "64a0c47adeac72d468803f7b718cfae2", "56af104ce910e2b260e543bdd030f55a", "6186d1560513c04ea7cddcca9defa484", "1a92c8cee6d17718c047e7346827c480"}, rawVideoMD5: "ac67b0d74a123f2bb0b587212a2d1b91"},
		"high14-explicit-cabac-partitioned-b8x16-mode2-deblock":        {bitstreamMD5: "e220986706221130507a86545cdf236b", frameMD5: []string{"f7e31af362f75c2c72082c963526bd62", "64a0c47adeac72d468803f7b718cfae2", "56af104ce910e2b260e543bdd030f55a", "6186d1560513c04ea7cddcca9defa484", "1a92c8cee6d17718c047e7346827c480"}, rawVideoMD5: "ac67b0d74a123f2bb0b587212a2d1b91"},
		"high14-explicit-cavlc-partitioned-b8x8-mode1-deblock":         {bitstreamMD5: "bfc44ca43fa722b37d410a1efcc04b16", frameMD5: []string{"6bcadaf1ef408dfb87ed0eef55afc867", "3e533602a0ac279d2b0540dbaa0c4c5e", "bc785f2462e03ea7b175dd24776260db", "7659b11af00b55241e074e254fd9b4d2", "c8b2f1da66efe9ef9af551fcc4a45da5"}, rawVideoMD5: "7197d1fb98488e3a6a5f6f39cc0404e7"},
		"high14-explicit-cavlc-partitioned-b8x8-mode2-deblock":         {bitstreamMD5: "b75cae9f6c5a9c1699857dbe9acb8bb8", frameMD5: []string{"6bcadaf1ef408dfb87ed0eef55afc867", "3e533602a0ac279d2b0540dbaa0c4c5e", "bc785f2462e03ea7b175dd24776260db", "7659b11af00b55241e074e254fd9b4d2", "c8b2f1da66efe9ef9af551fcc4a45da5"}, rawVideoMD5: "7197d1fb98488e3a6a5f6f39cc0404e7"},
		"high14-explicit-cabac-partitioned-b8x8-mode1-deblock":         {bitstreamMD5: "f2a78c33e4aa7c61d885645a83fb600d", frameMD5: []string{"6bcadaf1ef408dfb87ed0eef55afc867", "b3c2cf2e4b8232b43f109ed6dfd1bea6", "832a243bd7b906410a0ec038b9b9e74e", "7659b11af00b55241e074e254fd9b4d2", "068d4c19adf6515d32e3448f379fa2d6"}, rawVideoMD5: "cb14056e94506d6eca267fab09ca5522"},
		"high14-explicit-cabac-partitioned-b8x8-mode2-deblock":         {bitstreamMD5: "619494e09896cb9a8c27fe861cc66e8f", frameMD5: []string{"6bcadaf1ef408dfb87ed0eef55afc867", "b3c2cf2e4b8232b43f109ed6dfd1bea6", "832a243bd7b906410a0ec038b9b9e74e", "7659b11af00b55241e074e254fd9b4d2", "068d4c19adf6515d32e3448f379fa2d6"}, rawVideoMD5: "cb14056e94506d6eca267fab09ca5522"},
	}
}
