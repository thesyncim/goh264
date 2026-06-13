// SPDX-License-Identifier: LGPL-2.1-or-later

package goh264_test

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/thesyncim/goh264/internal/h264"
)

type highFrameMBAFFWeightedPCase struct {
	name                       string
	bitDepth                   int
	fieldFlag                  uint32
	payloadBits                string
	disableDeblockingFilterIDC uint32
	deblockMode                int32
	bitstreamMD5               string
	refFrameMD5                string
	pFrameMD5                  string
	rawVideoMD5                string
}

func TestHigh1214FrameMBAFFWeightedPFixtureSyntax(t *testing.T) {
	for _, tt := range highFrameMBAFFWeightedPCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFWeightedPFixture(tt)
			assertHighFrameMBAFFWeightedPFixtureSyntax(t, data, tt)
		})
	}
}

func TestDecodeAnnexBHigh1214FrameMBAFFWeightedPFrames(t *testing.T) {
	for _, tt := range highFrameMBAFFWeightedPCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFWeightedPFixture(tt)
			assertHighFrameMBAFFWeightedPFixtureSyntax(t, data, tt)

			frames, err := NewDecoder().DecodeAnnexBFrames(data)
			if err != nil {
				t.Fatalf("DecodeAnnexBFrames: %v", err)
			}
			assertHighFrameMBAFFWeightedPFrames(t, frames, tt)
		})
	}
}

func TestDecodeAVCHigh1214FrameMBAFFWeightedPFrames(t *testing.T) {
	for _, tt := range highFrameMBAFFWeightedPCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFWeightedPFixture(tt)
			assertHighFrameMBAFFWeightedPFixtureSyntax(t, data, tt)

			for _, nalLengthSize := range []int{2, 3, 4} {
				frames, err := NewDecoder().DecodeAVCFrames(annexBToAVC(t, data, nalLengthSize), nalLengthSize)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: DecodeAVCFrames: %v", nalLengthSize, err)
				}
				assertHighFrameMBAFFWeightedPFrames(t, frames, tt)
			}
		})
	}
}

func TestDecodeAVCCHigh1214FrameMBAFFWeightedPFrames(t *testing.T) {
	for _, tt := range highFrameMBAFFWeightedPCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFWeightedPFixture(tt)
			assertHighFrameMBAFFWeightedPFixtureSyntax(t, data, tt)

			for _, nalLengthSize := range []int{2, 3, 4} {
				config, packet := annexBToAVCConfigAndPacket(t, data, nalLengthSize)
				frames, err := NewDecoder().DecodeAVCCFrames(config, packet)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: DecodeAVCCFrames: %v", nalLengthSize, err)
				}
				assertHighFrameMBAFFWeightedPFrames(t, frames, tt)
			}
		})
	}
}

func TestDecodeConfiguredAVCHigh1214FrameMBAFFWeightedPFramesAcrossSamples(t *testing.T) {
	for _, tt := range highFrameMBAFFWeightedPCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFWeightedPFixture(tt)
			assertHighFrameMBAFFWeightedPFixtureSyntax(t, data, tt)

			for _, nalLengthSize := range []int{2, 3, 4} {
				config, samples := annexBToAVCConfigAndSamples(t, data, nalLengthSize)
				if len(samples) != 2 {
					t.Fatalf("nalLengthSize=%d samples = %d, want IDR/P", nalLengthSize, len(samples))
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
				assertHighFrameMBAFFWeightedPFrames(t, frames, tt)
			}
		})
	}
}

func TestFFmpegRawVideoMD5OracleHigh1214FrameMBAFFWeightedP(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}

	for _, tt := range highFrameMBAFFWeightedPCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFWeightedPFixture(tt)
			assertHighFrameMBAFFWeightedPFixtureSyntax(t, data, tt)
			assertFFmpegHighFrameMBAFFWeightedPRawVideoOracle(t, data, tt)
		})
	}
}

func highFrameMBAFFWeightedPCases() []highFrameMBAFFWeightedPCase {
	return []highFrameMBAFFWeightedPCase{
		{
			name:                       "High12FieldP16x16WeightedNoDeblock",
			bitDepth:                   12,
			fieldFlag:                  1,
			payloadBits:                highFrameMBAFFP16x16NoResidualPayloadBits,
			disableDeblockingFilterIDC: 1,
			deblockMode:                0,
			bitstreamMD5:               "baf5cbc97250900d95427439326fdd8a",
			refFrameMD5:                high12FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:                  "fa513b8ff25f0be0b1bf640c7af240be",
			rawVideoMD5:                "36af2e6a2f931f7073ce4c0a6dd2d357",
		},
		{
			name:                       "High12FieldP16x16WeightedFrameDeblock",
			bitDepth:                   12,
			fieldFlag:                  1,
			payloadBits:                highFrameMBAFFP16x16NoResidualPayloadBits,
			disableDeblockingFilterIDC: 0,
			deblockMode:                1,
			bitstreamMD5:               "0dcc2a1cac2872c5bb1f88aa7f56f903",
			refFrameMD5:                high12FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:                  "fa513b8ff25f0be0b1bf640c7af240be",
			rawVideoMD5:                "36af2e6a2f931f7073ce4c0a6dd2d357",
		},
		{
			name:                       "High12FieldP16x16WeightedSliceBoundary",
			bitDepth:                   12,
			fieldFlag:                  1,
			payloadBits:                highFrameMBAFFP16x16NoResidualPayloadBits,
			disableDeblockingFilterIDC: 2,
			deblockMode:                2,
			bitstreamMD5:               "f0da813d5669deaee5af56d6391c1196",
			refFrameMD5:                high12FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:                  "fa513b8ff25f0be0b1bf640c7af240be",
			rawVideoMD5:                "36af2e6a2f931f7073ce4c0a6dd2d357",
		},
		{
			name:                       "High12FrameP16x16WeightedNoDeblock",
			bitDepth:                   12,
			fieldFlag:                  0,
			payloadBits:                highFrameMBAFFFrameP16x16NoResidualPayloadBits,
			disableDeblockingFilterIDC: 1,
			deblockMode:                0,
			bitstreamMD5:               "b2ce579d7810151529dea39fe13411fc",
			refFrameMD5:                high12FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:                  "fa513b8ff25f0be0b1bf640c7af240be",
			rawVideoMD5:                "36af2e6a2f931f7073ce4c0a6dd2d357",
		},
		{
			name:                       "High12FrameP16x16WeightedFrameDeblock",
			bitDepth:                   12,
			fieldFlag:                  0,
			payloadBits:                highFrameMBAFFFrameP16x16NoResidualPayloadBits,
			disableDeblockingFilterIDC: 0,
			deblockMode:                1,
			bitstreamMD5:               "a2715eb9953582ec5733b8ed2600c386",
			refFrameMD5:                high12FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:                  "fa513b8ff25f0be0b1bf640c7af240be",
			rawVideoMD5:                "36af2e6a2f931f7073ce4c0a6dd2d357",
		},
		{
			name:                       "High12FrameP16x16WeightedSliceBoundary",
			bitDepth:                   12,
			fieldFlag:                  0,
			payloadBits:                highFrameMBAFFFrameP16x16NoResidualPayloadBits,
			disableDeblockingFilterIDC: 2,
			deblockMode:                2,
			bitstreamMD5:               "3015e5f466e1d31dcafea26b1ca116ae",
			refFrameMD5:                high12FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:                  "fa513b8ff25f0be0b1bf640c7af240be",
			rawVideoMD5:                "36af2e6a2f931f7073ce4c0a6dd2d357",
		},
		{
			name:                       "High14FieldP16x16WeightedNoDeblock",
			bitDepth:                   14,
			fieldFlag:                  1,
			payloadBits:                highFrameMBAFFP16x16NoResidualPayloadBits,
			disableDeblockingFilterIDC: 1,
			deblockMode:                0,
			bitstreamMD5:               "511edfee04eacf77b7d35f11ff866465",
			refFrameMD5:                high14FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:                  "a3a69ca9e580d4b6f73a058db6c8c119",
			rawVideoMD5:                "2dba4da6bb3a0d28d5dce98a8622561c",
		},
		{
			name:                       "High14FieldP16x16WeightedFrameDeblock",
			bitDepth:                   14,
			fieldFlag:                  1,
			payloadBits:                highFrameMBAFFP16x16NoResidualPayloadBits,
			disableDeblockingFilterIDC: 0,
			deblockMode:                1,
			bitstreamMD5:               "0e52e27a186f12b4d953f34e3ebd93b7",
			refFrameMD5:                high14FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:                  "a3a69ca9e580d4b6f73a058db6c8c119",
			rawVideoMD5:                "2dba4da6bb3a0d28d5dce98a8622561c",
		},
		{
			name:                       "High14FieldP16x16WeightedSliceBoundary",
			bitDepth:                   14,
			fieldFlag:                  1,
			payloadBits:                highFrameMBAFFP16x16NoResidualPayloadBits,
			disableDeblockingFilterIDC: 2,
			deblockMode:                2,
			bitstreamMD5:               "f0a6c929f38c5905a63510dddf3c6683",
			refFrameMD5:                high14FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:                  "a3a69ca9e580d4b6f73a058db6c8c119",
			rawVideoMD5:                "2dba4da6bb3a0d28d5dce98a8622561c",
		},
		{
			name:                       "High14FrameP16x16WeightedNoDeblock",
			bitDepth:                   14,
			fieldFlag:                  0,
			payloadBits:                highFrameMBAFFFrameP16x16NoResidualPayloadBits,
			disableDeblockingFilterIDC: 1,
			deblockMode:                0,
			bitstreamMD5:               "55c84ef64fa5d6f1212cd08790f2a4d1",
			refFrameMD5:                high14FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:                  "a3a69ca9e580d4b6f73a058db6c8c119",
			rawVideoMD5:                "2dba4da6bb3a0d28d5dce98a8622561c",
		},
		{
			name:                       "High14FrameP16x16WeightedFrameDeblock",
			bitDepth:                   14,
			fieldFlag:                  0,
			payloadBits:                highFrameMBAFFFrameP16x16NoResidualPayloadBits,
			disableDeblockingFilterIDC: 0,
			deblockMode:                1,
			bitstreamMD5:               "e0c78707b1980e9f39852ea21cf66a4d",
			refFrameMD5:                high14FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:                  "a3a69ca9e580d4b6f73a058db6c8c119",
			rawVideoMD5:                "2dba4da6bb3a0d28d5dce98a8622561c",
		},
		{
			name:                       "High14FrameP16x16WeightedSliceBoundary",
			bitDepth:                   14,
			fieldFlag:                  0,
			payloadBits:                highFrameMBAFFFrameP16x16NoResidualPayloadBits,
			disableDeblockingFilterIDC: 2,
			deblockMode:                2,
			bitstreamMD5:               "6914b0b155265eaddb879f9da5ace043",
			refFrameMD5:                high14FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:                  "a3a69ca9e580d4b6f73a058db6c8c119",
			rawVideoMD5:                "2dba4da6bb3a0d28d5dce98a8622561c",
		},
	}
}

func highFrameMBAFFWeightedPFixture(tt highFrameMBAFFWeightedPCase) []byte {
	var data []byte
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALSPS), highFrameMBAFFInterSPSRBSP(tt.bitDepth)))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALPPS), highWeightedPPPSRBSP(tt.bitDepth)))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALIDRSlice), highFrameMBAFFIntraPCMSliceRBSP(tt.bitDepth)))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALSlice), highFrameMBAFFWeightedPSliceRBSP(tt.payloadBits, tt.disableDeblockingFilterIDC)))
	return data
}

func highFrameMBAFFWeightedPSliceRBSP(payloadBits string, disableDeblockingFilterIDC uint32) []byte {
	var b decoderSEIBitBuilder
	b.writeUE(0)
	b.writeUE(0)
	b.writeUE(0)
	b.writeBits(1, 4)
	b.writeBit(0)
	b.writeBit(0)
	b.writeBit(0)
	writeHighWeightedPPredWeightSyntax(&b)
	b.writeBit(0)
	b.writeSE(0)
	writeHighCAVLCDeblockSyntax(&b, disableDeblockingFilterIDC)
	highIntra16x16WritePayloadBits(&b, payloadBits)
	return b.rbsp()
}

func assertHighFrameMBAFFWeightedPFixtureSyntax(t *testing.T, data []byte, tt highFrameMBAFFWeightedPCase) {
	t.Helper()
	sum := md5.Sum(data)
	if got := hex.EncodeToString(sum[:]); got != tt.bitstreamMD5 {
		t.Fatalf("%s frame-MBAFF weighted-P bitstream md5 = %s, want %s", tt.name, got, tt.bitstreamMD5)
	}

	nals, spsList, ppsList := parseHighFrameMBAFFWeightedPFixtureSyntax(t, data, tt)
	pair := readHighFrameMBAFFWeightedCAVLCP16x16Pair(t, nals[1], spsList[0], ppsList[0], tt)
	if pair.fieldFlag != tt.fieldFlag {
		t.Fatalf("%s frame-MBAFF weighted-P pair field flag = %d, want %d", tt.name, pair.fieldFlag, tt.fieldFlag)
	}
	for i, mb := range []highFrameMBAFFCAVLCP16x16Macroblock{pair.top, pair.bottom} {
		if mb.skipRun != 0 || mb.mbType != 0 || mb.refIdxFlag != tt.fieldFlag || mb.cbp != 0 {
			t.Fatalf("%s weighted-P macroblock[%d] skip/mb_type/ref_idx_flag/cbp = %d/%d/%d/%d (code %d), want P16x16 no residual with field flag %d",
				tt.name, i, mb.skipRun, mb.mbType, mb.refIdxFlag, mb.cbp, mb.cbpCode, tt.fieldFlag)
		}
	}
}

func parseHighFrameMBAFFWeightedPFixtureSyntax(t *testing.T, data []byte, tt highFrameMBAFFWeightedPCase) ([]h264.NALUnit, [32]*h264.SPS, [256]*h264.PPS) {
	t.Helper()
	nals, err := h264.SplitAnnexB(data)
	if err != nil {
		t.Fatal(err)
	}
	var spsList [32]*h264.SPS
	var ppsList [256]*h264.PPS
	var gotVCL []h264.NALUnit
	var gotSliceTypes []int32
	for _, nal := range nals {
		switch nal.Type {
		case h264.NALSPS:
			sps, err := h264.DecodeSPS(nal.RBSP)
			if err != nil {
				t.Fatal(err)
			}
			if sps.ProfileIDC != 244 || sps.Width != 16 || sps.Height != 32 ||
				sps.ChromaFormatIDC != 1 || sps.BitDepthLuma != int32(tt.bitDepth) ||
				sps.BitDepthChroma != int32(tt.bitDepth) || sps.RefFrameCount != 1 ||
				sps.FrameMBSOnlyFlag != 0 || sps.MBAFF != 1 {
				t.Fatalf("SPS profile/format = %d %dx%d chroma %d depth %d/%d refs %d frame_mbs_only:%d mbaff:%d, want High 4:4:4 Predictive-compatible 16x32 yuv420p%dle ref frame-MBAFF",
					sps.ProfileIDC, sps.Width, sps.Height, sps.ChromaFormatIDC,
					sps.BitDepthLuma, sps.BitDepthChroma, sps.RefFrameCount,
					sps.FrameMBSOnlyFlag, sps.MBAFF, tt.bitDepth)
			}
			spsList[sps.SPSID] = sps
		case h264.NALPPS:
			pps, err := h264.DecodePPS(nal.RBSP, &spsList)
			if err != nil {
				t.Fatal(err)
			}
			if pps.CABAC != 0 || pps.Transform8x8Mode != 0 || pps.WeightedPred != 1 ||
				pps.WeightedBipredIDC != 0 || pps.DeblockingFilterParametersPresent == 0 ||
				pps.RefCount[0] != 1 || pps.RefCount[1] != 1 {
				t.Fatalf("PPS CABAC/8x8/weighted/bipred/deblock/refs = %d/%d/%d/%d/%d/%d/%d, want CAVLC/no-8x8/weighted/no-bipred/deblock params/1/1",
					pps.CABAC, pps.Transform8x8Mode, pps.WeightedPred, pps.WeightedBipredIDC,
					pps.DeblockingFilterParametersPresent, pps.RefCount[0], pps.RefCount[1])
			}
			ppsList[pps.PPSID] = pps
		case h264.NALIDRSlice, h264.NALSlice:
			sh, err := h264.ParseSliceHeader(nal, &ppsList)
			if err != nil {
				t.Fatal(err)
			}
			wantDeblockMode := tt.deblockMode
			if sh.SliceTypeNoS != h264.PictureTypeP {
				wantDeblockMode = 0
			}
			if sh.PictureStructure != h264.PictureFrame || sh.DeblockingFilter != wantDeblockMode ||
				sh.QScale != 26 || sh.SPS.MBAFF != 1 {
				t.Fatalf("slice picture/deblock/qp/mbaff = %d/%d/%d/%d, want frame/mode-%d/26/1",
					sh.PictureStructure, sh.DeblockingFilter, sh.QScale, sh.SPS.MBAFF, wantDeblockMode)
			}
			if sh.SliceTypeNoS == h264.PictureTypeP {
				if sh.ListCount != 1 || sh.RefCount[0] != 1 {
					t.Fatalf("P slice lists/refs = %d/%v, want one L0 ref", sh.ListCount, sh.RefCount)
				}
				assertHigh14WeightedPPredWeight(t, sh.PredWeightTable)
			}
			gotVCL = append(gotVCL, nal)
			gotSliceTypes = append(gotSliceTypes, sh.SliceTypeNoS)
		default:
			t.Fatalf("unexpected NAL type %d in %s frame-MBAFF weighted-P fixture", nal.Type, tt.name)
		}
	}
	if len(gotVCL) != 2 || gotVCL[0].Type != h264.NALIDRSlice || gotVCL[1].Type != h264.NALSlice {
		gotTypes := make([]h264.NALUnitType, 0, len(gotVCL))
		for _, nal := range gotVCL {
			gotTypes = append(gotTypes, nal.Type)
		}
		t.Fatalf("VCL NALs = %v, want IDR slice followed by non-IDR slice", gotTypes)
	}
	if len(gotSliceTypes) != 2 || gotSliceTypes[0] != h264.PictureTypeI || gotSliceTypes[1] != h264.PictureTypeP {
		t.Fatalf("slice types = %v, want I then P", gotSliceTypes)
	}
	return gotVCL, spsList, ppsList
}

func readHighFrameMBAFFWeightedCAVLCP16x16Pair(t *testing.T, nal h264.NALUnit, sps *h264.SPS, pps *h264.PPS, tt highFrameMBAFFWeightedPCase) highFrameMBAFFCAVLCP16x16Pair {
	t.Helper()
	if sps == nil || pps == nil {
		t.Fatal("missing SPS/PPS for frame-MBAFF weighted-P macroblock syntax check")
	}
	br := high10ResidualCAVLCBitReader{data: nal.RBSP}
	firstMB := br.readUE(t)
	rawSliceType := br.readUE(t)
	sliceTypeNoS := high10ResidualCAVLCSliceTypeNoS(t, rawSliceType)
	ppsID := br.readUE(t)
	if firstMB != 0 || sliceTypeNoS != h264.PictureTypeP || ppsID != pps.PPSID {
		t.Fatalf("slice header first_mb/type/pps = %d/%d/%d, want first weighted P slice with PPS %d", firstMB, sliceTypeNoS, ppsID, pps.PPSID)
	}
	br.readBits(t, int(sps.Log2MaxFrameNum))
	fieldPic := br.readBit(t)
	if fieldPic != 0 {
		t.Fatalf("field_pic_flag = %d, want frame picture", fieldPic)
	}
	if sps.PocType == 0 {
		br.readBits(t, int(sps.Log2MaxPocLSB))
		if pps.PicOrderPresent != 0 {
			br.readSE(t)
		}
	}
	refCount0 := pps.RefCount[0]
	if br.readBit(t) != 0 {
		refCount0 = br.readUE(t) + 1
	}
	high10ResidualCAVLCReadRefPicListModifications(t, &br, 1)
	readHighFrameMBAFFWeightedPPredWeightSyntax(t, &br)
	high10ResidualCAVLCReadRefPicMarking(t, &br, nal)
	if pps.CABAC != 0 {
		br.readUE(t)
	}
	br.readSE(t)
	if pps.DeblockingFilterParametersPresent != 0 {
		disableID := br.readUE(t)
		if disableID != tt.disableDeblockingFilterIDC {
			t.Fatalf("disable_deblocking_filter_idc = %d, want %d", disableID, tt.disableDeblockingFilterIDC)
		}
		if disableID != 1 {
			if alpha := br.readSE(t); alpha != 0 {
				t.Fatalf("slice_alpha_c0_offset_div2 = %d, want 0", alpha)
			}
			if beta := br.readSE(t); beta != 0 {
				t.Fatalf("slice_beta_offset_div2 = %d, want 0", beta)
			}
		}
	}
	topSkipRun := br.readUE(t)
	fieldFlag := br.readBit(t)
	refCount0 = h264MBAFFRefCountForSyntax(refCount0, fieldFlag)
	top := readHighFrameMBAFFCAVLCP16x16Macroblock(t, &br, topSkipRun, refCount0, "")
	bottomSkipRun := br.readUE(t)
	bottom := readHighFrameMBAFFCAVLCP16x16Macroblock(t, &br, bottomSkipRun, refCount0, "")
	return highFrameMBAFFCAVLCP16x16Pair{fieldFlag: fieldFlag, top: top, bottom: bottom}
}

func readHighFrameMBAFFWeightedPPredWeightSyntax(t *testing.T, br *high10ResidualCAVLCBitReader) {
	t.Helper()
	if denom := br.readUE(t); denom != 2 {
		t.Fatalf("luma_log2_weight_denom = %d, want 2", denom)
	}
	if denom := br.readUE(t); denom != 1 {
		t.Fatalf("chroma_log2_weight_denom = %d, want 1", denom)
	}
	if flag := br.readBit(t); flag != 1 {
		t.Fatalf("luma_weight_l0_flag = %d, want 1", flag)
	}
	if weight := br.readSE(t); weight != 3 {
		t.Fatalf("luma_weight_l0[0] = %d, want 3", weight)
	}
	if offset := br.readSE(t); offset != -2 {
		t.Fatalf("luma_offset_l0[0] = %d, want -2", offset)
	}
	if flag := br.readBit(t); flag != 1 {
		t.Fatalf("chroma_weight_l0_flag = %d, want 1", flag)
	}
	for _, want := range []int32{2, 1, -1, 3} {
		if got := br.readSE(t); got != want {
			t.Fatalf("chroma pred weight value = %d, want %d", got, want)
		}
	}
}

func assertHighFrameMBAFFWeightedPFrames(t *testing.T, frames []*Frame, tt highFrameMBAFFWeightedPCase) {
	t.Helper()
	assertHighFrameMBAFFP16x16NoResidualFrames(t, frames, highFrameMBAFFP16x16NoResidualCase{
		name:         tt.name,
		bitDepth:     tt.bitDepth,
		bitstreamMD5: tt.bitstreamMD5,
		refFrameMD5:  tt.refFrameMD5,
		pFrameMD5:    tt.pFrameMD5,
		rawVideoMD5:  tt.rawVideoMD5,
	})
	var rawVideo []byte
	for i, frame := range frames {
		raw, err := frame.AppendRawYUVBytesLE(nil)
		if err != nil {
			t.Fatalf("frame[%d] AppendRawYUVBytesLE: %v", i, err)
		}
		rawVideo = append(rawVideo, raw...)
	}
	if len(rawVideo) != 3072 {
		t.Fatalf("rawvideo len = %d, want 3072", len(rawVideo))
	}
	sum := md5.Sum(rawVideo)
	if got := hex.EncodeToString(sum[:]); got != tt.rawVideoMD5 {
		t.Fatalf("rawvideo md5 = %s, want %s", got, tt.rawVideoMD5)
	}
}

func assertFFmpegHighFrameMBAFFWeightedPRawVideoOracle(t *testing.T, data []byte, tt highFrameMBAFFWeightedPCase) {
	t.Helper()
	path := writeTempH264(t, data)
	pixFmt := fmt.Sprintf("yuv420p%dle", tt.bitDepth)
	framemd5 := exec.Command(
		"ffmpeg",
		"-v", "error",
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
	for i, want := range []string{tt.refFrameMD5, tt.pFrameMD5} {
		line := []byte(fmt.Sprintf("0, %10d, %10d,        1,     1536, %s", i, i, want))
		if !bytes.Contains(framemd5Out, line) {
			t.Fatalf("missing %q in framemd5:\n%s", line, framemd5Out)
		}
	}

	rawvideo := exec.Command(
		"ffmpeg",
		"-v", "error",
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
	if len(raw) != 3072 {
		t.Fatalf("rawvideo size = %d, want 3072", len(raw))
	}
	sum := md5.Sum(raw)
	if got := hex.EncodeToString(sum[:]); got != tt.rawVideoMD5 {
		t.Fatalf("rawvideo md5 = %s, want %s", got, tt.rawVideoMD5)
	}
}
