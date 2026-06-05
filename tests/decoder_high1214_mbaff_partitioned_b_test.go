// SPDX-License-Identifier: LGPL-2.1-or-later

package goh264_test

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/thesyncim/goh264/internal/h264"
)

type highFrameMBAFFPartitionedBCase struct {
	name                       string
	bitDepth                   int
	mbTypeCode                 uint32
	subMBTypeCode              uint32
	fieldFlag                  uint32
	fieldRefIdxFlagCount       int
	mvdPairCount               int
	payloadBits                string
	disableDeblockingFilterIDC uint32
	deblockMode                int32
	bitstreamMD5               string
	frameMD5                   []string
	rawVideoMD5                string
}

func TestHigh1214FrameMBAFFPartitionedBFixtureSyntax(t *testing.T) {
	for _, tt := range highFrameMBAFFPartitionedBCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFPartitionedBFixture(tt)
			assertHighFrameMBAFFPartitionedBFixtureSyntax(t, data, tt)
		})
	}
}

func TestDecodeAnnexBHigh1214FrameMBAFFPartitionedBFrames(t *testing.T) {
	for _, tt := range highFrameMBAFFPartitionedBCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFPartitionedBFixture(tt)
			assertHighFrameMBAFFPartitionedBFixtureSyntax(t, data, tt)

			frames, err := NewDecoder().DecodeAnnexBFrames(data)
			if err != nil {
				t.Fatalf("DecodeAnnexBFrames: %v", err)
			}
			assertHighFrameMBAFFPartitionedBFrames(t, frames, tt)
		})
	}
}

func TestDecodeAVCHigh1214FrameMBAFFPartitionedBFrames(t *testing.T) {
	for _, tt := range highFrameMBAFFPartitionedBCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFPartitionedBFixture(tt)
			assertHighFrameMBAFFPartitionedBFixtureSyntax(t, data, tt)

			for _, nalLengthSize := range []int{2, 3, 4} {
				frames, err := NewDecoder().DecodeAVCFrames(annexBToAVC(t, data, nalLengthSize), nalLengthSize)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: DecodeAVCFrames: %v", nalLengthSize, err)
				}
				assertHighFrameMBAFFPartitionedBFrames(t, frames, tt)
			}
		})
	}
}

func TestDecodeAVCWithConfigurationRecordHigh1214FrameMBAFFPartitionedBFrames(t *testing.T) {
	for _, tt := range highFrameMBAFFPartitionedBCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFPartitionedBFixture(tt)
			assertHighFrameMBAFFPartitionedBFixtureSyntax(t, data, tt)

			for _, nalLengthSize := range []int{2, 3, 4} {
				config, packet := annexBToAVCConfigAndPacket(t, data, nalLengthSize)
				frames, err := NewDecoder().DecodeAVCFramesWithConfigurationRecord(config, packet)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: DecodeAVCFramesWithConfigurationRecord: %v", nalLengthSize, err)
				}
				assertHighFrameMBAFFPartitionedBFrames(t, frames, tt)
			}
		})
	}
}

func TestDecodeConfiguredAVCHigh1214FrameMBAFFPartitionedBFramesAcrossSamplesFlush(t *testing.T) {
	for _, tt := range highFrameMBAFFPartitionedBCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFPartitionedBFixture(tt)
			assertHighFrameMBAFFPartitionedBFixtureSyntax(t, data, tt)

			for _, nalLengthSize := range []int{2, 3, 4} {
				config, samples := annexBToAVCConfigAndSamples(t, data, nalLengthSize)
				if len(samples) != 3 {
					t.Fatalf("nalLengthSize=%d samples = %d, want IDR/P/B", nalLengthSize, len(samples))
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
				assertHighFrameMBAFFPartitionedBFrames(t, frames, tt)
			}
		})
	}
}

func TestFFmpegRawVideoFrameMD5OracleHigh1214FrameMBAFFPartitionedB(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}

	for _, tt := range highFrameMBAFFPartitionedBCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFPartitionedBFixture(tt)
			assertHighFrameMBAFFPartitionedBFixtureSyntax(t, data, tt)
			assertFFmpegHighFrameMBAFFPartitionedBRawVideoOracle(t, data, tt)
		})
	}
}

func highFrameMBAFFPartitionedBCases() []highFrameMBAFFPartitionedBCase {
	bitstreamMD5 := map[string]string{
		"High12FieldB16x8BiNoDeblock":     "55620fb7b0b637d476838908a139aa21",
		"High12FieldB16x8BiFrameDeblock":  "eba565940aa78bd79ec1167f916e71a4",
		"High12FieldB16x8BiSliceBoundary": "d6a04ef822b6f4307f32d9b4a6cce83b",
		"High12FieldB8x16BiNoDeblock":     "fa0389b34f2cf313bc039ffc3b78bf3c",
		"High12FieldB8x16BiFrameDeblock":  "ab87b0f1995347f40048171d42663b08",
		"High12FieldB8x16BiSliceBoundary": "a5993fcaf689903562ad49720237735a",
		"High12FieldB8x8BiNoDeblock":      "04cda224cef141c8c705190e745d1f9f",
		"High12FieldB8x8BiFrameDeblock":   "c1f3e44c1dc89d1bc0521149e8ed6cb4",
		"High12FieldB8x8BiSliceBoundary":  "dab9cd963aa7e84a32934031221eabda",
		"High12FrameB16x8BiNoDeblock":     "9034a644494e3756429d862f967d5458",
		"High12FrameB16x8BiFrameDeblock":  "a943aa82d2657a4bf470a345ce2edf8e",
		"High12FrameB16x8BiSliceBoundary": "98d16eea071edfcb51254ce4133fad0b",
		"High12FrameB8x16BiNoDeblock":     "cabfbfc81c94718f7ac5d9b9dfb42dc0",
		"High12FrameB8x16BiFrameDeblock":  "9b41769fabebc13513e0de0aea95aadc",
		"High12FrameB8x16BiSliceBoundary": "1fc9f25dd5cb24f96c0429dfb28e68b1",
		"High12FrameB8x8BiNoDeblock":      "3398a876b3c2224133b972cb3f66fb27",
		"High12FrameB8x8BiFrameDeblock":   "5b242f3b5ea78902534fe206b95e4708",
		"High12FrameB8x8BiSliceBoundary":  "dbfda0e60e72b69c646e8b40790d8f98",
		"High14FieldB16x8BiNoDeblock":     "22d877c91918e80573494c85ff1dabc7",
		"High14FieldB16x8BiFrameDeblock":  "e48e3d1b4dbc806087262420b4f1c9ae",
		"High14FieldB16x8BiSliceBoundary": "3a3cf5f7ebba89e3d386312aec5c6834",
		"High14FieldB8x16BiNoDeblock":     "2066016c030d154cc1605def891c06dd",
		"High14FieldB8x16BiFrameDeblock":  "63b0c3d7010d2250d0f5886415b187a1",
		"High14FieldB8x16BiSliceBoundary": "efcb2d25d17a31e28c7ea60bb042d7e7",
		"High14FieldB8x8BiNoDeblock":      "4c433cf1f46d3d1ead5f8ee5bfa6d3fc",
		"High14FieldB8x8BiFrameDeblock":   "8fa91cab1cfe8619caa138b30b7a3e0a",
		"High14FieldB8x8BiSliceBoundary":  "bbc845af085eab6a8d8f5554a8be5bee",
		"High14FrameB16x8BiNoDeblock":     "d4880c08e60493c85b8d8d88ffb13eb5",
		"High14FrameB16x8BiFrameDeblock":  "c9495a34053b8441a332e2ec755f0925",
		"High14FrameB16x8BiSliceBoundary": "ff2df33d41b0810cc83b0087843e6903",
		"High14FrameB8x16BiNoDeblock":     "63fb4f9c8bd64e200e18a0fa96e95b2d",
		"High14FrameB8x16BiFrameDeblock":  "87a28f3b133b9dbc938303732158c93f",
		"High14FrameB8x16BiSliceBoundary": "36b7171efa27c6df93bd337fe29ebe14",
		"High14FrameB8x8BiNoDeblock":      "f19feb8be029e13035d8eb7e8b90011e",
		"High14FrameB8x8BiFrameDeblock":   "3b3653c5d3aba13adc29862f910ce941",
		"High14FrameB8x8BiSliceBoundary":  "6b2e8a9d905a5414283049513821f1c3",
	}

	shapes := []struct {
		name         string
		mbTypeCode   uint32
		mbTypeBits   string
		subMBTypeBit string
		mvdPairCount int
	}{
		{name: "B16x8Bi", mbTypeCode: 20, mbTypeBits: "000010101", mvdPairCount: 4},
		{name: "B8x16Bi", mbTypeCode: 21, mbTypeBits: "000010110", mvdPairCount: 4},
		{name: "B8x8Bi", mbTypeCode: 22, mbTypeBits: "000010111", subMBTypeBit: "00100", mvdPairCount: 8},
	}
	deblocks := []struct {
		name      string
		disableID uint32
		mode      int32
	}{
		{name: "NoDeblock", disableID: 1, mode: 0},
		{name: "FrameDeblock", disableID: 0, mode: 1},
		{name: "SliceBoundary", disableID: 2, mode: 2},
	}

	var out []highFrameMBAFFPartitionedBCase
	for _, bitDepth := range []int{12, 14} {
		frameMD5 := high12FrameMBAFFIntraPCMFrameMD5
		rawVideoMD5 := "94e77e8922a8b65ac84903483c1252ff"
		if bitDepth == 14 {
			frameMD5 = high14FrameMBAFFIntraPCMFrameMD5
			rawVideoMD5 = "389fb07fd25ac40b475b9b13d4e10b13"
		}
		for _, field := range []struct {
			name string
			flag uint32
		}{
			{name: "Field", flag: 1},
			{name: "Frame", flag: 0},
		} {
			for _, shape := range shapes {
				for _, deblock := range deblocks {
					name := fmt.Sprintf("High%d%s%s%s", bitDepth, field.name, shape.name, deblock.name)
					fieldRefIdxFlagCount := 0
					if field.flag != 0 {
						fieldRefIdxFlagCount = 4
						if shape.mbTypeCode == 22 {
							fieldRefIdxFlagCount = 8
						}
					}
					payloadBits := highFrameMBAFFPartitionedBPayloadBits(
						shape.mbTypeBits,
						shape.subMBTypeBit,
						field.flag,
						fieldRefIdxFlagCount,
						shape.mvdPairCount,
					)
					out = append(out, highFrameMBAFFPartitionedBCase{
						name:                       name,
						bitDepth:                   bitDepth,
						mbTypeCode:                 shape.mbTypeCode,
						subMBTypeCode:              3,
						fieldFlag:                  field.flag,
						fieldRefIdxFlagCount:       fieldRefIdxFlagCount,
						mvdPairCount:               shape.mvdPairCount,
						payloadBits:                payloadBits,
						disableDeblockingFilterIDC: deblock.disableID,
						deblockMode:                deblock.mode,
						bitstreamMD5:               highFrameMBAFFPartitionedBBitstreamMD5(bitstreamMD5, name),
						frameMD5:                   []string{frameMD5, frameMD5, frameMD5},
						rawVideoMD5:                rawVideoMD5,
					})
				}
			}
		}
	}
	return out
}

func highFrameMBAFFPartitionedBBitstreamMD5(bitstreamMD5 map[string]string, name string) string {
	got, ok := bitstreamMD5[name]
	if !ok {
		panic(fmt.Sprintf("missing bitstream md5 for %s", name))
	}
	return got
}

func highFrameMBAFFPartitionedBPayloadBits(mbTypeBits string, subMBTypeBit string, fieldFlag uint32, fieldRefIdxFlagCount int, mvdPairCount int) string {
	mbBits := mbTypeBits
	if subMBTypeBit != "" {
		mbBits += strings.Repeat(subMBTypeBit, 4)
	}
	if fieldFlag != 0 {
		mbBits += strings.Repeat("1", fieldRefIdxFlagCount)
	}
	mbBits += strings.Repeat("1", mvdPairCount*2)
	mbBits += "1"
	pairPrefix := "10"
	if fieldFlag != 0 {
		pairPrefix = "11"
	}
	return pairPrefix + mbBits + "1" + mbBits
}

func highFrameMBAFFPartitionedBFixture(tt highFrameMBAFFPartitionedBCase) []byte {
	var data []byte
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALSPS), highFrameMBAFFBInterSPSRBSP(tt.bitDepth)))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALPPS), highIntraPCMPPSRBSP(tt.bitDepth)))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALIDRSlice), highFrameMBAFFIntraPCMSliceRBSP(tt.bitDepth)))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALSlice), highFrameMBAFFPSkipNoResidualSliceRBSP()))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(h264.NALSlice), highFrameMBAFFPartitionedBSliceRBSP(tt.payloadBits, tt.disableDeblockingFilterIDC)))
	return data
}

func highFrameMBAFFPartitionedBSliceRBSP(payloadBits string, disableDeblockingFilterIDC uint32) []byte {
	var b decoderSEIBitBuilder
	b.writeUE(0)
	b.writeUE(1)
	b.writeUE(0)
	b.writeBits(2, 4)
	b.writeBit(0)
	b.writeBit(1)
	b.writeBit(0)
	b.writeBit(0)
	b.writeBit(0)
	b.writeSE(0)
	writeHighCAVLCDeblockSyntax(&b, disableDeblockingFilterIDC)
	highIntra16x16WritePayloadBits(&b, payloadBits)
	return b.rbsp()
}

func assertHighFrameMBAFFPartitionedBFixtureSyntax(t *testing.T, data []byte, tt highFrameMBAFFPartitionedBCase) {
	t.Helper()
	sum := md5.Sum(data)
	if got := hex.EncodeToString(sum[:]); got != tt.bitstreamMD5 {
		t.Fatalf("%s frame-MBAFF partitioned B bitstream md5 = %s, want %s", tt.name, got, tt.bitstreamMD5)
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
					sh.DirectSpatialMVPred != 1 || sh.DeblockingFilter != tt.deblockMode {
					t.Fatalf("B ref/lists/refs/direct/deblock = %d/%d/%v/%d/%d, want non-ref B refs=1/1 spatial-direct-header deblock=%d",
						nal.RefIDC, sh.ListCount, sh.RefCount, sh.DirectSpatialMVPred, sh.DeblockingFilter, tt.deblockMode)
				}
				bNAL = nal
			}
			gotTypes = append(gotTypes, sh.SliceTypeNoS)
		default:
			t.Fatalf("unexpected NAL type %d in %s frame-MBAFF partitioned B fixture", nal.Type, tt.name)
		}
	}
	wantTypes := []int32{h264.PictureTypeI, h264.PictureTypeP, h264.PictureTypeB}
	if !highFrameMBAFFBSkipInt32SlicesEqual(gotTypes, wantTypes) {
		t.Fatalf("slice types = %v, want %v", gotTypes, wantTypes)
	}
	if bNAL.RBSP == nil {
		t.Fatal("missing B slice")
	}
	pair := readHighFrameMBAFFCAVLCPartitionedBPair(t, bNAL, spsList[0], ppsList[0], tt)
	if pair.fieldFlag != tt.fieldFlag {
		t.Fatalf("%s frame-MBAFF partitioned B pair field flag = %d, want %d", tt.name, pair.fieldFlag, tt.fieldFlag)
	}
	for i, mb := range []highFrameMBAFFCAVLCPartitionedBMacroblock{pair.top, pair.bottom} {
		if mb.skipRun != 0 || mb.mbTypeCode != tt.mbTypeCode || mb.cbp != 0 {
			t.Fatalf("%s partitioned B macroblock[%d] skip/mb_type/cbp = %d/%d/%d (code %d), want mb_type %d no residual",
				tt.name, i, mb.skipRun, mb.mbTypeCode, mb.cbp, mb.cbpCode, tt.mbTypeCode)
		}
		if tt.mbTypeCode == 22 {
			for j, subType := range mb.subMBType {
				if subType != tt.subMBTypeCode {
					t.Fatalf("%s partitioned B macroblock[%d] sub_mb_type[%d] = %d, want B_Bi_8x8 code %d",
						tt.name, i, j, subType, tt.subMBTypeCode)
				}
			}
		}
		for j := 0; j < tt.fieldRefIdxFlagCount; j++ {
			if mb.refIdxFlags[j] != 1 {
				t.Fatalf("%s partitioned B macroblock[%d] ref_idx flag[%d] = %d, want field ref flag 1",
					tt.name, i, j, mb.refIdxFlags[j])
			}
		}
	}
}

type highFrameMBAFFCAVLCPartitionedBMacroblock struct {
	skipRun     uint32
	mbTypeCode  uint32
	subMBType   [4]uint32
	refIdxFlags [8]uint32
	cbpCode     uint32
	cbp         uint32
}

type highFrameMBAFFCAVLCPartitionedBPair struct {
	fieldFlag uint32
	top       highFrameMBAFFCAVLCPartitionedBMacroblock
	bottom    highFrameMBAFFCAVLCPartitionedBMacroblock
}

func readHighFrameMBAFFCAVLCPartitionedBPair(t *testing.T, nal h264.NALUnit, sps *h264.SPS, pps *h264.PPS, tt highFrameMBAFFPartitionedBCase) highFrameMBAFFCAVLCPartitionedBPair {
	t.Helper()
	if sps == nil || pps == nil {
		t.Fatal("missing SPS/PPS for frame-MBAFF partitioned B syntax check")
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
	if direct := br.readBit(t); direct != 1 {
		t.Fatalf("direct_spatial_mv_pred_flag = %d, want 1", direct)
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
	if pps.CABAC != 0 {
		br.readUE(t)
	}
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
	top := readHighFrameMBAFFCAVLCPartitionedBMacroblock(t, &br, topSkipRun, tt)
	bottomSkipRun := br.readUE(t)
	bottom := readHighFrameMBAFFCAVLCPartitionedBMacroblock(t, &br, bottomSkipRun, tt)
	return highFrameMBAFFCAVLCPartitionedBPair{
		fieldFlag: fieldFlag,
		top:       top,
		bottom:    bottom,
	}
}

func readHighFrameMBAFFCAVLCPartitionedBMacroblock(t *testing.T, br *high10ResidualCAVLCBitReader, skipRun uint32, tt highFrameMBAFFPartitionedBCase) highFrameMBAFFCAVLCPartitionedBMacroblock {
	t.Helper()
	var mb highFrameMBAFFCAVLCPartitionedBMacroblock
	mb.skipRun = skipRun
	mb.mbTypeCode = br.readUE(t)
	if mb.mbTypeCode != tt.mbTypeCode {
		t.Fatalf("%s partitioned B mb_type = %d, want %d", tt.name, mb.mbTypeCode, tt.mbTypeCode)
	}
	if mb.mbTypeCode == 22 {
		for i := 0; i < 4; i++ {
			mb.subMBType[i] = br.readUE(t)
		}
	}
	for i := 0; i < tt.fieldRefIdxFlagCount; i++ {
		mb.refIdxFlags[i] = br.readBit(t)
	}
	for i := 0; i < tt.mvdPairCount; i++ {
		if x := br.readSE(t); x != 0 {
			t.Fatalf("%s partitioned B mvd_l%d.x = %d, want 0", tt.name, i, x)
		}
		if y := br.readSE(t); y != 0 {
			t.Fatalf("%s partitioned B mvd_l%d.y = %d, want 0", tt.name, i, y)
		}
	}
	cbpCode := br.readUE(t)
	if cbpCode >= uint32(len(high10ResidualCAVLCInterCBP)) {
		t.Fatalf("coded_block_pattern code = %d, want < %d", cbpCode, len(high10ResidualCAVLCInterCBP))
	}
	mb.cbpCode = cbpCode
	mb.cbp = uint32(high10ResidualCAVLCInterCBP[cbpCode])
	return mb
}

func assertHighFrameMBAFFPartitionedBFrames(t *testing.T, frames []*Frame, tt highFrameMBAFFPartitionedBCase) {
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

func assertFFmpegHighFrameMBAFFPartitionedBRawVideoOracle(t *testing.T, data []byte, tt highFrameMBAFFPartitionedBCase) {
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
