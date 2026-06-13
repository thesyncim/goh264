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

type highFrameMBAFFDirectSubBCase struct {
	name                       string
	bitDepth                   int
	directSpatial              uint32
	fieldFlag                  uint32
	payloadBits                string
	disableDeblockingFilterIDC uint32
	deblockMode                int32
	bitstreamMD5               string
	frameMD5                   []string
	rawVideoMD5                string
}

func TestHigh1214FrameMBAFFDirectSubBFixtureSyntax(t *testing.T) {
	for _, tt := range highFrameMBAFFDirectSubBCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFDirectSubBFixture(tt)
			assertHighFrameMBAFFDirectSubBFixtureSyntax(t, data, tt)
		})
	}
}

func TestDecodeAnnexBHigh1214FrameMBAFFDirectSubBFrames(t *testing.T) {
	for _, tt := range highFrameMBAFFDirectSubBCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFDirectSubBFixture(tt)
			assertHighFrameMBAFFDirectSubBFixtureSyntax(t, data, tt)

			frames, err := NewDecoder().DecodeAnnexBFrames(data)
			if err != nil {
				t.Fatalf("DecodeAnnexBFrames: %v", err)
			}
			assertHighFrameMBAFFDirectSubBFrames(t, frames, tt)
		})
	}
}

func TestDecodeAVCHigh1214FrameMBAFFDirectSubBFrames(t *testing.T) {
	for _, tt := range highFrameMBAFFDirectSubBCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFDirectSubBFixture(tt)
			assertHighFrameMBAFFDirectSubBFixtureSyntax(t, data, tt)

			for _, nalLengthSize := range []int{2, 3, 4} {
				frames, err := NewDecoder().DecodeAVCFrames(annexBToAVC(t, data, nalLengthSize), nalLengthSize)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: DecodeAVCFrames: %v", nalLengthSize, err)
				}
				assertHighFrameMBAFFDirectSubBFrames(t, frames, tt)
			}
		})
	}
}

func TestDecodeAVCCHigh1214FrameMBAFFDirectSubBFrames(t *testing.T) {
	for _, tt := range highFrameMBAFFDirectSubBCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFDirectSubBFixture(tt)
			assertHighFrameMBAFFDirectSubBFixtureSyntax(t, data, tt)

			for _, nalLengthSize := range []int{2, 3, 4} {
				config, packet := annexBToAVCConfigAndPacket(t, data, nalLengthSize)
				frames, err := NewDecoder().DecodeAVCCFrames(config, packet)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: DecodeAVCCFrames: %v", nalLengthSize, err)
				}
				assertHighFrameMBAFFDirectSubBFrames(t, frames, tt)
			}
		})
	}
}

func TestDecodeConfiguredAVCHigh1214FrameMBAFFDirectSubBFramesAcrossSamplesFlush(t *testing.T) {
	for _, tt := range highFrameMBAFFDirectSubBCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFDirectSubBFixture(tt)
			assertHighFrameMBAFFDirectSubBFixtureSyntax(t, data, tt)

			for _, nalLengthSize := range []int{2, 3, 4} {
				config, samples := annexBToAVCConfigAndSamples(t, data, nalLengthSize)
				if len(samples) != 3 {
					t.Fatalf("nalLengthSize=%d samples = %d, want IDR/P/B", nalLengthSize, len(samples))
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
				assertHighFrameMBAFFDirectSubBFrames(t, frames, tt)
			}
		})
	}
}

func TestFFmpegRawVideoFrameMD5OracleHigh1214FrameMBAFFDirectSubB(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}

	for _, tt := range highFrameMBAFFDirectSubBCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFDirectSubBFixture(tt)
			assertHighFrameMBAFFDirectSubBFixtureSyntax(t, data, tt)
			assertFFmpegHighFrameMBAFFDirectSubBRawVideoOracle(t, data, tt)
		})
	}
}

func highFrameMBAFFDirectSubBCases() []highFrameMBAFFDirectSubBCase {
	bitstreamMD5 := map[string]string{
		"High12TemporalDirectSubB8x8FieldNoDeblock":     "7602d5da63f913dc8a2555afe378d2ad",
		"High12TemporalDirectSubB8x8FieldFrameDeblock":  "ccd29553f92985385381ad6174b21805",
		"High12TemporalDirectSubB8x8FieldSliceBoundary": "aee4949f26207bfa4ac51d3ddcaffc0d",
		"High12TemporalDirectSubB8x8FrameNoDeblock":     "ea7b8e877bf91e9cd1f811b748e0e9f9",
		"High12TemporalDirectSubB8x8FrameFrameDeblock":  "2009c77f04657eb19df8a783b47579c1",
		"High12TemporalDirectSubB8x8FrameSliceBoundary": "f4f8c48fcc31824689792efe43c98aab",
		"High12SpatialDirectSubB8x8FieldNoDeblock":      "5b7d45669c50be35372b477363e9e9ce",
		"High12SpatialDirectSubB8x8FieldFrameDeblock":   "8905953d2f57cda343318394f4e956c4",
		"High12SpatialDirectSubB8x8FieldSliceBoundary":  "16457cd222b86f5c141e818a67d63ea8",
		"High12SpatialDirectSubB8x8FrameNoDeblock":      "2cf46c8053068973dab9f2a8a3ff10ed",
		"High12SpatialDirectSubB8x8FrameFrameDeblock":   "826706e7b9f3014442d343318a91bd7a",
		"High12SpatialDirectSubB8x8FrameSliceBoundary":  "2dc76103f95cc2c13e65d903f65babe2",
		"High14TemporalDirectSubB8x8FieldNoDeblock":     "b94911df7f35b8e7681c8bfb436bc139",
		"High14TemporalDirectSubB8x8FieldFrameDeblock":  "71304fe812fe2d0119e88ab227866e3a",
		"High14TemporalDirectSubB8x8FieldSliceBoundary": "3d625da6c7122f7d8efdc0e640eae22c",
		"High14TemporalDirectSubB8x8FrameNoDeblock":     "dad74200b5fd44365b3c55aaff504fdf",
		"High14TemporalDirectSubB8x8FrameFrameDeblock":  "e8f3858f3c6a3db746db0270b8f8ddb2",
		"High14TemporalDirectSubB8x8FrameSliceBoundary": "a808d5115e5737246b6f1545e31686a9",
		"High14SpatialDirectSubB8x8FieldNoDeblock":      "1f6f46b9457e9f1bd3c08da4b4ee14b6",
		"High14SpatialDirectSubB8x8FieldFrameDeblock":   "3057368d39d6a4ec807cb01c3f3846ef",
		"High14SpatialDirectSubB8x8FieldSliceBoundary":  "ee6f571d5583a06f206e37d68f78a44b",
		"High14SpatialDirectSubB8x8FrameNoDeblock":      "9dde291767a424aea35d7f28454deafc",
		"High14SpatialDirectSubB8x8FrameFrameDeblock":   "1078530b56fbc6e67fe019f32703da36",
		"High14SpatialDirectSubB8x8FrameSliceBoundary":  "e7bf7271a3103bdd5ad12b54993e93a2",
	}
	var out []highFrameMBAFFDirectSubBCase
	for _, bitDepth := range []int{12, 14} {
		frameMD5 := high12FrameMBAFFIntraPCMFrameMD5
		rawVideoMD5 := "94e77e8922a8b65ac84903483c1252ff"
		if bitDepth == 14 {
			frameMD5 = high14FrameMBAFFIntraPCMFrameMD5
			rawVideoMD5 = "389fb07fd25ac40b475b9b13d4e10b13"
		}
		for _, direct := range []struct {
			name string
			flag uint32
		}{
			{name: "TemporalDirect", flag: 0},
			{name: "SpatialDirect", flag: 1},
		} {
			for _, field := range []struct {
				name string
				flag uint32
			}{
				{name: "Field", flag: 1},
				{name: "Frame", flag: 0},
			} {
				for _, deblock := range []struct {
					name      string
					disableID uint32
					mode      int32
				}{
					{name: "NoDeblock", disableID: 1, mode: 0},
					{name: "FrameDeblock", disableID: 0, mode: 1},
					{name: "SliceBoundary", disableID: 2, mode: 2},
				} {
					name := fmt.Sprintf("High%d%sSubB8x8%s%s", bitDepth, direct.name, field.name, deblock.name)
					out = append(out, highFrameMBAFFDirectSubBCase{
						name:                       name,
						bitDepth:                   bitDepth,
						directSpatial:              direct.flag,
						fieldFlag:                  field.flag,
						payloadBits:                highFrameMBAFFDirectSubBPayloadBits(field.flag),
						disableDeblockingFilterIDC: deblock.disableID,
						deblockMode:                deblock.mode,
						bitstreamMD5:               highFrameMBAFFDirectSubBBitstreamMD5(bitstreamMD5, name),
						frameMD5:                   []string{frameMD5, frameMD5, frameMD5},
						rawVideoMD5:                rawVideoMD5,
					})
				}
			}
		}
	}
	return out
}

func highFrameMBAFFDirectSubBBitstreamMD5(bitstreamMD5 map[string]string, name string) string {
	got, ok := bitstreamMD5[name]
	if !ok {
		panic(fmt.Sprintf("missing direct-sub B bitstream md5 for %s", name))
	}
	return got
}

func highFrameMBAFFDirectSubBPayloadBits(fieldFlag uint32) string {
	mbBits := "000010111" + "1111" + "1"
	pairPrefix := "10"
	if fieldFlag != 0 {
		pairPrefix = "11"
	}
	return pairPrefix + mbBits + "1" + mbBits
}

func highFrameMBAFFDirectSubBFixture(tt highFrameMBAFFDirectSubBCase) []byte {
	var data []byte
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALSPS), highFrameMBAFFBInterSPSRBSP(tt.bitDepth)))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALPPS), highIntraPCMPPSRBSP(tt.bitDepth)))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALIDRSlice), highFrameMBAFFIntraPCMSliceRBSP(tt.bitDepth)))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALSlice), highFrameMBAFFPSkipNoResidualSliceRBSP()))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(h264.NALSlice), highFrameMBAFFDirectSubBSliceRBSP(tt)))
	return data
}

func highFrameMBAFFDirectSubBSliceRBSP(tt highFrameMBAFFDirectSubBCase) []byte {
	var b decoderSEIBitBuilder
	b.writeUE(0)
	b.writeUE(1)
	b.writeUE(0)
	b.writeBits(2, 4)
	b.writeBit(0)
	b.writeBit(tt.directSpatial)
	b.writeBit(0)
	b.writeBit(0)
	b.writeBit(0)
	b.writeSE(0)
	writeHighCAVLCDeblockSyntax(&b, tt.disableDeblockingFilterIDC)
	highIntra16x16WritePayloadBits(&b, tt.payloadBits)
	return b.rbsp()
}

func assertHighFrameMBAFFDirectSubBFixtureSyntax(t *testing.T, data []byte, tt highFrameMBAFFDirectSubBCase) {
	t.Helper()
	sum := md5.Sum(data)
	if got := hex.EncodeToString(sum[:]); got != tt.bitstreamMD5 {
		t.Fatalf("%s frame-MBAFF direct-sub B bitstream md5 = %s, want %s", tt.name, got, tt.bitstreamMD5)
	}

	nals, err := h264.SplitAnnexB(data)
	if err != nil {
		t.Fatal(err)
	}
	var spsList [32]*h264.SPS
	var ppsList [256]*h264.PPS
	var gotTypes []int32
	var bNAL h264.NALUnit
	for i, nal := range nals {
		switch nal.Type {
		case h264.NALSPS:
			sps, err := h264.DecodeSPS(nal.RBSP)
			if err != nil {
				t.Fatal(err)
			}
			if i != 0 || sps.ProfileIDC != 244 || sps.Width != 16 || sps.Height != 32 ||
				sps.ChromaFormatIDC != 1 || sps.BitDepthLuma != int32(tt.bitDepth) ||
				sps.BitDepthChroma != int32(tt.bitDepth) || sps.RefFrameCount != 2 ||
				sps.FrameMBSOnlyFlag != 0 || sps.MBAFF != 1 || sps.Direct8x8InferenceFlag != 1 {
				t.Fatalf("SPS profile/format = %d %dx%d chroma %d depth %d/%d refs %d frame_mbs_only:%d mbaff:%d direct8x8:%d, want High%d 16x32 4:2:0 frame-MBAFF refs=2 direct8x8",
					sps.ProfileIDC, sps.Width, sps.Height, sps.ChromaFormatIDC,
					sps.BitDepthLuma, sps.BitDepthChroma, sps.RefFrameCount,
					sps.FrameMBSOnlyFlag, sps.MBAFF, sps.Direct8x8InferenceFlag, tt.bitDepth)
			}
			spsList[sps.SPSID] = sps
		case h264.NALPPS:
			pps, err := h264.DecodePPS(nal.RBSP, &spsList)
			if err != nil {
				t.Fatal(err)
			}
			if pps.CABAC != 0 || pps.Transform8x8Mode != 0 || pps.WeightedPred != 0 ||
				pps.WeightedBipredIDC != 0 || pps.RefCount != [2]uint32{1, 1} ||
				pps.DeblockingFilterParametersPresent == 0 {
				t.Fatalf("PPS CABAC/8x8/weights/refs/deblock = %d/%d/%d/%d/%v/%d, want CAVLC/no-8x8/unweighted refs=1/1 deblock params",
					pps.CABAC, pps.Transform8x8Mode, pps.WeightedPred,
					pps.WeightedBipredIDC, pps.RefCount, pps.DeblockingFilterParametersPresent)
			}
			ppsList[pps.PPSID] = pps
		case h264.NALIDRSlice, h264.NALSlice:
			sh, err := h264.ParseSliceHeader(nal, &ppsList)
			if err != nil {
				t.Fatal(err)
			}
			if sh.PictureStructure != h264.PictureFrame || sh.QScale != 26 || sh.SPS.MBAFF != 1 {
				t.Fatalf("slice picture/q/mbaff = %d/%d/%d, want frame/26/1", sh.PictureStructure, sh.QScale, sh.SPS.MBAFF)
			}
			if sh.SliceTypeNoS == h264.PictureTypeB {
				if nal.RefIDC != 0 || sh.ListCount != 2 || sh.RefCount != [2]uint32{1, 1} ||
					sh.DirectSpatialMVPred != int32(tt.directSpatial) || sh.DeblockingFilter != tt.deblockMode {
					t.Fatalf("B ref/lists/refs/direct/deblock = %d/%d/%v/%d/%d, want non-ref B refs=1/1 direct=%d deblock=%d",
						nal.RefIDC, sh.ListCount, sh.RefCount, sh.DirectSpatialMVPred, sh.DeblockingFilter,
						tt.directSpatial, tt.deblockMode)
				}
				bNAL = nal
			}
			gotTypes = append(gotTypes, sh.SliceTypeNoS)
		default:
			t.Fatalf("unexpected NAL type %d in %s frame-MBAFF direct-sub B fixture", nal.Type, tt.name)
		}
	}
	wantTypes := []int32{h264.PictureTypeI, h264.PictureTypeP, h264.PictureTypeB}
	if !highFrameMBAFFBSkipInt32SlicesEqual(gotTypes, wantTypes) {
		t.Fatalf("slice types = %v, want %v", gotTypes, wantTypes)
	}
	if bNAL.RBSP == nil {
		t.Fatal("missing B slice")
	}
	pair := readHighFrameMBAFFCAVLCDirectSubBPair(t, bNAL, spsList[0], ppsList[0], tt)
	if pair.fieldFlag != tt.fieldFlag {
		t.Fatalf("%s frame-MBAFF direct-sub B pair field flag = %d, want %d", tt.name, pair.fieldFlag, tt.fieldFlag)
	}
	for i, mb := range []highFrameMBAFFCAVLCDirectSubBMacroblock{pair.top, pair.bottom} {
		if mb.skipRun != 0 || mb.mbTypeCode != 22 || mb.cbp != 0 {
			t.Fatalf("%s direct-sub B macroblock[%d] skip/mb_type/cbp = %d/%d/%d (code %d), want B_8x8 direct-sub no residual",
				tt.name, i, mb.skipRun, mb.mbTypeCode, mb.cbp, mb.cbpCode)
		}
		for j, subType := range mb.subMBType {
			if subType != 0 {
				t.Fatalf("%s direct-sub B macroblock[%d] sub_mb_type[%d] = %d, want B_Direct_8x8 code 0",
					tt.name, i, j, subType)
			}
		}
	}
}

type highFrameMBAFFCAVLCDirectSubBMacroblock struct {
	skipRun    uint32
	mbTypeCode uint32
	subMBType  [4]uint32
	cbpCode    uint32
	cbp        uint32
}

type highFrameMBAFFCAVLCDirectSubBPair struct {
	fieldFlag uint32
	top       highFrameMBAFFCAVLCDirectSubBMacroblock
	bottom    highFrameMBAFFCAVLCDirectSubBMacroblock
}

func readHighFrameMBAFFCAVLCDirectSubBPair(t *testing.T, nal h264.NALUnit, sps *h264.SPS, pps *h264.PPS, tt highFrameMBAFFDirectSubBCase) highFrameMBAFFCAVLCDirectSubBPair {
	t.Helper()
	if sps == nil || pps == nil {
		t.Fatal("missing SPS/PPS for frame-MBAFF direct-sub B syntax check")
	}
	br := high10ResidualCAVLCBitReader{data: nal.RBSP}
	firstMB := br.readUE(t)
	rawSliceType := br.readUE(t)
	sliceTypeNoS := high10ResidualCAVLCSliceTypeNoS(t, rawSliceType)
	ppsID := br.readUE(t)
	if firstMB != 0 || sliceTypeNoS != h264.PictureTypeB || ppsID != pps.PPSID {
		t.Fatalf("slice header first_mb/type/pps = %d/%d/%d, want first B slice with PPS %d", firstMB, sliceTypeNoS, ppsID, pps.PPSID)
	}
	if frameNum := br.readBits(t, int(sps.Log2MaxFrameNum)); frameNum != 2 {
		t.Fatalf("B frame_num = %d, want 2", frameNum)
	}
	if fieldPic := br.readBit(t); fieldPic != 0 {
		t.Fatalf("field_pic_flag = %d, want frame picture", fieldPic)
	}
	if direct := br.readBit(t); direct != tt.directSpatial {
		t.Fatalf("direct_spatial_mv_pred_flag = %d, want %d", direct, tt.directSpatial)
	}
	refCount := pps.RefCount
	if br.readBit(t) != 0 {
		refCount[0] = br.readUE(t) + 1
		refCount[1] = br.readUE(t) + 1
	}
	if refCount != [2]uint32{1, 1} {
		t.Fatalf("B ref counts = %v, want 1/1", refCount)
	}
	high10ResidualCAVLCReadRefPicListModifications(t, &br, 2)
	high10ResidualCAVLCReadRefPicMarking(t, &br, nal)
	if qpDelta := br.readSE(t); qpDelta != 0 {
		t.Fatalf("slice_qp_delta = %d, want 0", qpDelta)
	}
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
	top := readHighFrameMBAFFCAVLCDirectSubBMacroblock(t, &br, topSkipRun, tt)
	bottomSkipRun := br.readUE(t)
	bottom := readHighFrameMBAFFCAVLCDirectSubBMacroblock(t, &br, bottomSkipRun, tt)
	return highFrameMBAFFCAVLCDirectSubBPair{fieldFlag: fieldFlag, top: top, bottom: bottom}
}

func readHighFrameMBAFFCAVLCDirectSubBMacroblock(t *testing.T, br *high10ResidualCAVLCBitReader, skipRun uint32, tt highFrameMBAFFDirectSubBCase) highFrameMBAFFCAVLCDirectSubBMacroblock {
	t.Helper()
	var mb highFrameMBAFFCAVLCDirectSubBMacroblock
	mb.skipRun = skipRun
	mb.mbTypeCode = br.readUE(t)
	if mb.mbTypeCode != 22 {
		t.Fatalf("%s direct-sub B mb_type = %d, want 22", tt.name, mb.mbTypeCode)
	}
	for i := 0; i < 4; i++ {
		mb.subMBType[i] = br.readUE(t)
	}
	cbpCode := br.readUE(t)
	if cbpCode >= uint32(len(high10ResidualCAVLCInterCBP)) {
		t.Fatalf("coded_block_pattern code = %d, want < %d", cbpCode, len(high10ResidualCAVLCInterCBP))
	}
	mb.cbpCode = cbpCode
	mb.cbp = uint32(high10ResidualCAVLCInterCBP[cbpCode])
	return mb
}

func assertHighFrameMBAFFDirectSubBFrames(t *testing.T, frames []*Frame, tt highFrameMBAFFDirectSubBCase) {
	t.Helper()
	if len(frames) != len(tt.frameMD5) {
		t.Fatalf("frames = %d, want %d", len(frames), len(tt.frameMD5))
	}
	var rawVideo []byte
	for i, frame := range frames {
		if frame == nil {
			t.Fatalf("frame[%d] is nil", i)
		}
		if frame.Width != 16 || frame.Height != 32 || frame.ChromaFormatIDC != 1 ||
			frame.BitDepthLuma != tt.bitDepth || frame.BitDepthChroma != tt.bitDepth {
			t.Fatalf("frame[%d] format = %dx%d chroma %d depth %d/%d, want 16x32 yuv420p%dle",
				i, frame.Width, frame.Height, frame.ChromaFormatIDC, frame.BitDepthLuma, frame.BitDepthChroma, tt.bitDepth)
		}
		raw, err := frame.AppendRawYUVBytesLE(nil)
		if err != nil {
			t.Fatalf("frame[%d] AppendRawYUVBytesLE: %v", i, err)
		}
		rawVideo = append(rawVideo, raw...)
		sum := md5.Sum(raw)
		if got := hex.EncodeToString(sum[:]); got != tt.frameMD5[i] {
			t.Fatalf("frame[%d] md5 = %s, want %s", i, got, tt.frameMD5[i])
		}
	}
	if len(rawVideo) != 4608 {
		t.Fatalf("rawvideo len = %d, want 4608", len(rawVideo))
	}
	sum := md5.Sum(rawVideo)
	if got := hex.EncodeToString(sum[:]); got != tt.rawVideoMD5 {
		t.Fatalf("rawvideo md5 = %s, want %s", got, tt.rawVideoMD5)
	}
}

func assertFFmpegHighFrameMBAFFDirectSubBRawVideoOracle(t *testing.T, data []byte, tt highFrameMBAFFDirectSubBCase) {
	t.Helper()
	path := writeTempH264(t, data)
	pixFmt := fmt.Sprintf("yuv420p%dle", tt.bitDepth)
	framemd5 := exec.Command(
		"ffmpeg",
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
		line := []byte(fmt.Sprintf("0, %10d, %10d,        1,     1536, %s", i, i, want))
		if !bytes.Contains(framemd5Out, line) {
			t.Fatalf("missing %q in framemd5:\n%s", line, framemd5Out)
		}
	}

	rawvideo := exec.Command(
		"ffmpeg",
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
	if len(raw) != 4608 {
		t.Fatalf("rawvideo size = %d, want 4608", len(raw))
	}
	sum := md5.Sum(raw)
	if got := hex.EncodeToString(sum[:]); got != tt.rawVideoMD5 {
		t.Fatalf("rawvideo md5 = %s, want %s", got, tt.rawVideoMD5)
	}
}
