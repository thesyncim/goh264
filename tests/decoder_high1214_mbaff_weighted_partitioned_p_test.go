// SPDX-License-Identifier: LGPL-2.1-or-later

package goh264_test

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/thesyncim/goh264/internal/h264"
)

type highFrameMBAFFWeightedPartitionedPCase struct {
	name                       string
	bitDepth                   int
	fieldFlag                  uint32
	mbType                     uint32
	payloadBits                string
	disableDeblockingFilterIDC uint32
	deblockMode                int32
	bitstreamMD5               string
	refFrameMD5                string
	pFrameMD5                  string
	rawVideoMD5                string
}

func TestHigh1214FrameMBAFFWeightedPartitionedPFixtureSyntax(t *testing.T) {
	for _, tt := range highFrameMBAFFWeightedPartitionedPCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFWeightedPartitionedPFixture(tt)
			assertHighFrameMBAFFWeightedPartitionedPFixtureSyntax(t, data, tt)
		})
	}
}

func TestDecodeAnnexBHigh1214FrameMBAFFWeightedPartitionedPFrames(t *testing.T) {
	for _, tt := range highFrameMBAFFWeightedPartitionedPCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFWeightedPartitionedPFixture(tt)
			assertHighFrameMBAFFWeightedPartitionedPFixtureSyntax(t, data, tt)

			frames, err := NewDecoder().DecodeAnnexBFrames(data)
			if err != nil {
				t.Fatalf("DecodeAnnexBFrames: %v", err)
			}
			assertHighFrameMBAFFWeightedPartitionedPFrames(t, frames, tt)
		})
	}
}

func TestDecodeAVCHigh1214FrameMBAFFWeightedPartitionedPFrames(t *testing.T) {
	for _, tt := range highFrameMBAFFWeightedPartitionedPCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFWeightedPartitionedPFixture(tt)
			assertHighFrameMBAFFWeightedPartitionedPFixtureSyntax(t, data, tt)

			for _, nalLengthSize := range []int{2, 3, 4} {
				frames, err := NewDecoder().DecodeAVCFrames(annexBToAVC(t, data, nalLengthSize), nalLengthSize)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: DecodeAVCFrames: %v", nalLengthSize, err)
				}
				assertHighFrameMBAFFWeightedPartitionedPFrames(t, frames, tt)
			}
		})
	}
}

func TestDecodeAVCWithConfigurationRecordHigh1214FrameMBAFFWeightedPartitionedPFrames(t *testing.T) {
	for _, tt := range highFrameMBAFFWeightedPartitionedPCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFWeightedPartitionedPFixture(tt)
			assertHighFrameMBAFFWeightedPartitionedPFixtureSyntax(t, data, tt)

			for _, nalLengthSize := range []int{2, 3, 4} {
				config, packet := annexBToAVCConfigAndPacket(t, data, nalLengthSize)
				frames, err := NewDecoder().DecodeAVCFramesWithConfigurationRecord(config, packet)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: DecodeAVCFramesWithConfigurationRecord: %v", nalLengthSize, err)
				}
				assertHighFrameMBAFFWeightedPartitionedPFrames(t, frames, tt)
			}
		})
	}
}

func TestDecodeConfiguredAVCHigh1214FrameMBAFFWeightedPartitionedPFramesAcrossSamples(t *testing.T) {
	for _, tt := range highFrameMBAFFWeightedPartitionedPCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFWeightedPartitionedPFixture(tt)
			assertHighFrameMBAFFWeightedPartitionedPFixtureSyntax(t, data, tt)

			for _, nalLengthSize := range []int{2, 3, 4} {
				config, samples := annexBToAVCConfigAndSamples(t, data, nalLengthSize)
				if len(samples) != 2 {
					t.Fatalf("nalLengthSize=%d samples = %d, want IDR/P", nalLengthSize, len(samples))
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
				assertHighFrameMBAFFWeightedPartitionedPFrames(t, frames, tt)
			}
		})
	}
}

func TestFFmpegRawVideoMD5OracleHigh1214FrameMBAFFWeightedPartitionedP(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}

	for _, tt := range highFrameMBAFFWeightedPartitionedPCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFWeightedPartitionedPFixture(tt)
			assertHighFrameMBAFFWeightedPartitionedPFixtureSyntax(t, data, tt)
			assertFFmpegHighFrameMBAFFWeightedPRawVideoOracle(t, data, highFrameMBAFFWeightedPartitionedPAsWeightedPCase(tt))
		})
	}
}

func highFrameMBAFFWeightedPartitionedPCases() []highFrameMBAFFWeightedPartitionedPCase {
	bitstreamMD5 := highFrameMBAFFWeightedPartitionedPBitstreamMD5()
	var out []highFrameMBAFFWeightedPartitionedPCase
	for _, bitDepth := range []int{12, 14} {
		refFrameMD5 := high12FrameMBAFFIntraPCMFrameMD5
		pFrameMD5 := "fa513b8ff25f0be0b1bf640c7af240be"
		rawVideoMD5 := "36af2e6a2f931f7073ce4c0a6dd2d357"
		if bitDepth == 14 {
			refFrameMD5 = high14FrameMBAFFIntraPCMFrameMD5
			pFrameMD5 = "a3a69ca9e580d4b6f73a058db6c8c119"
			rawVideoMD5 = "2dba4da6bb3a0d28d5dce98a8622561c"
		}
		for _, coding := range []struct {
			name      string
			fieldFlag uint32
			payloads  []struct {
				name   string
				mbType uint32
				bits   string
			}
		}{
			{name: "Field", fieldFlag: 1, payloads: []struct {
				name   string
				mbType uint32
				bits   string
			}{
				{name: "P16x8", mbType: 1, bits: highFrameMBAFFP16x8NoResidualPayloadBits},
				{name: "P8x16", mbType: 2, bits: highFrameMBAFFP8x16NoResidualPayloadBits},
				{name: "P8x8", mbType: 3, bits: highFrameMBAFFP8x8NoResidualPayloadBits},
			}},
			{name: "Frame", fieldFlag: 0, payloads: []struct {
				name   string
				mbType uint32
				bits   string
			}{
				{name: "P16x8", mbType: 1, bits: highFrameMBAFFFrameP16x8NoResidualPayloadBits},
				{name: "P8x16", mbType: 2, bits: highFrameMBAFFFrameP8x16NoResidualPayloadBits},
				{name: "P8x8", mbType: 3, bits: highFrameMBAFFFrameP8x8NoResidualPayloadBits},
			}},
		} {
			for _, payload := range coding.payloads {
				for _, deblock := range []struct {
					name      string
					disableID uint32
					mode      int32
				}{
					{name: "NoDeblock", disableID: 1, mode: 0},
					{name: "FrameDeblock", disableID: 0, mode: 1},
					{name: "SliceBoundary", disableID: 2, mode: 2},
				} {
					name := fmt.Sprintf("High%d%s%sWeighted%s", bitDepth, coding.name, payload.name, deblock.name)
					out = append(out, highFrameMBAFFWeightedPartitionedPCase{
						name:                       name,
						bitDepth:                   bitDepth,
						fieldFlag:                  coding.fieldFlag,
						mbType:                     payload.mbType,
						payloadBits:                payload.bits,
						disableDeblockingFilterIDC: deblock.disableID,
						deblockMode:                deblock.mode,
						bitstreamMD5:               bitstreamMD5[name],
						refFrameMD5:                refFrameMD5,
						pFrameMD5:                  pFrameMD5,
						rawVideoMD5:                rawVideoMD5,
					})
				}
			}
		}
	}
	return out
}

func highFrameMBAFFWeightedPartitionedPBitstreamMD5() map[string]string {
	return map[string]string{
		"High12FieldP16x8WeightedNoDeblock":     "b129b1b5302af1e2f38efc387e2d62ee",
		"High12FieldP16x8WeightedFrameDeblock":  "a58455747c1296f8ca5ea2d9905cd20c",
		"High12FieldP16x8WeightedSliceBoundary": "e942d99268cccc2b3b82d90e50355646",
		"High12FieldP8x16WeightedNoDeblock":     "a0e3226631a4048b88a26defa62ec377",
		"High12FieldP8x16WeightedFrameDeblock":  "5bc1e6da9fa3a65e501ec58c3f6b4323",
		"High12FieldP8x16WeightedSliceBoundary": "2ad62d485058e57412e102f84cb4f3ad",
		"High12FieldP8x8WeightedNoDeblock":      "dfd97cda26560e1e362c7918b4051911",
		"High12FieldP8x8WeightedFrameDeblock":   "2d52e31bf7a45f4b68843d1c719a280c",
		"High12FieldP8x8WeightedSliceBoundary":  "61b4f7135a14e2927288385d86bfc9af",
		"High12FrameP16x8WeightedNoDeblock":     "207b6bf58b9c5d7bc359a55df8c71778",
		"High12FrameP16x8WeightedFrameDeblock":  "70d2f0a4e3b8c1a40523b40f88d7577e",
		"High12FrameP16x8WeightedSliceBoundary": "1093018b326d6b9a5de5cf82a39e1d5c",
		"High12FrameP8x16WeightedNoDeblock":     "8a909b804a3b1129aa8e881e8856462a",
		"High12FrameP8x16WeightedFrameDeblock":  "84b22e7f6271dd3b06d791962ed21130",
		"High12FrameP8x16WeightedSliceBoundary": "7fa8a9047f461f63fe9074162ddd3949",
		"High12FrameP8x8WeightedNoDeblock":      "daf31115b3c334303facdcf29e153e6f",
		"High12FrameP8x8WeightedFrameDeblock":   "a325f1e818686dd8b7d3645393e9a8bf",
		"High12FrameP8x8WeightedSliceBoundary":  "5833a14d1bc734103ee1f9f24f91a962",
		"High14FieldP16x8WeightedNoDeblock":     "91d9a03a98cfde0f11c4c88d678c5060",
		"High14FieldP16x8WeightedFrameDeblock":  "512394b41b8991d2a8457b381c0ba401",
		"High14FieldP16x8WeightedSliceBoundary": "97f1e4b64b520fe4c688d0fb08753c3c",
		"High14FieldP8x16WeightedNoDeblock":     "2482f5758cc025b07dc3c2d15aacf7d1",
		"High14FieldP8x16WeightedFrameDeblock":  "4028f0c144432a0027049ed7d65b5564",
		"High14FieldP8x16WeightedSliceBoundary": "7756669ec222feef9cc47cb1104ecc53",
		"High14FieldP8x8WeightedNoDeblock":      "8fec6b17965805846858612ede5da99e",
		"High14FieldP8x8WeightedFrameDeblock":   "227b4bf953b2454b78e02e2991452518",
		"High14FieldP8x8WeightedSliceBoundary":  "e093768921f9f9325fe25683b826f5e0",
		"High14FrameP16x8WeightedNoDeblock":     "fab1869e31302f2974f77ff785288c38",
		"High14FrameP16x8WeightedFrameDeblock":  "ebb02d6dd2a3cb132c6fe3800c75e837",
		"High14FrameP16x8WeightedSliceBoundary": "2b5c30f26cffc602c98241b3bd340408",
		"High14FrameP8x16WeightedNoDeblock":     "d0f6cfc08d5a1ba55ceed063617a48d0",
		"High14FrameP8x16WeightedFrameDeblock":  "c736059001b6642fac5ca2662c2e3a23",
		"High14FrameP8x16WeightedSliceBoundary": "950e52ac0a5b90dd5e9b69b1f47231f5",
		"High14FrameP8x8WeightedNoDeblock":      "32dd0ac8e67a7796169668287bb3014d",
		"High14FrameP8x8WeightedFrameDeblock":   "076b12c849dd69cb75ccd91108ec2c86",
		"High14FrameP8x8WeightedSliceBoundary":  "cecaf72d8dd8618e3fa505522f2ec262",
	}
}

func highFrameMBAFFWeightedPartitionedPFixture(tt highFrameMBAFFWeightedPartitionedPCase) []byte {
	return highFrameMBAFFWeightedPFixture(highFrameMBAFFWeightedPartitionedPAsWeightedPCase(tt))
}

func highFrameMBAFFWeightedPartitionedPAsWeightedPCase(tt highFrameMBAFFWeightedPartitionedPCase) highFrameMBAFFWeightedPCase {
	return highFrameMBAFFWeightedPCase{
		name:                       tt.name,
		bitDepth:                   tt.bitDepth,
		fieldFlag:                  tt.fieldFlag,
		payloadBits:                tt.payloadBits,
		disableDeblockingFilterIDC: tt.disableDeblockingFilterIDC,
		deblockMode:                tt.deblockMode,
		bitstreamMD5:               tt.bitstreamMD5,
		refFrameMD5:                tt.refFrameMD5,
		pFrameMD5:                  tt.pFrameMD5,
		rawVideoMD5:                tt.rawVideoMD5,
	}
}

func assertHighFrameMBAFFWeightedPartitionedPFixtureSyntax(t *testing.T, data []byte, tt highFrameMBAFFWeightedPartitionedPCase) {
	t.Helper()
	sum := md5.Sum(data)
	if got := hex.EncodeToString(sum[:]); got != tt.bitstreamMD5 {
		t.Fatalf("%s frame-MBAFF weighted partitioned-P bitstream md5 = %s, want %s", tt.name, got, tt.bitstreamMD5)
	}

	nals, spsList, ppsList := parseHighFrameMBAFFWeightedPFixtureSyntax(t, data, highFrameMBAFFWeightedPartitionedPAsWeightedPCase(tt))
	pair := readHighFrameMBAFFWeightedCAVLCPartitionedPPair(t, nals[1], spsList[0], ppsList[0], tt)
	if pair.fieldFlag != tt.fieldFlag {
		t.Fatalf("%s frame-MBAFF weighted partitioned-P pair field flag = %d, want %d", tt.name, pair.fieldFlag, tt.fieldFlag)
	}
	wantRefIdxCount := highFrameMBAFFPartitionedPRefIdxCount(t, tt.mbType)
	for i, mb := range []highFrameMBAFFCAVLCPartitionedPMacroblock{pair.top, pair.bottom} {
		if mb.skipRun != 0 || mb.mbType != tt.mbType || mb.cbp != 0 {
			t.Fatalf("%s weighted partitioned-P macroblock[%d] skip/mb_type/cbp = %d/%d/%d (code %d), want mb_type %d cbp 0",
				tt.name, i, mb.skipRun, mb.mbType, mb.cbp, mb.cbpCode, tt.mbType)
		}
		if mb.refIdxCount != wantRefIdxCount {
			t.Fatalf("%s weighted partitioned-P macroblock[%d] ref_idx count = %d, want %d", tt.name, i, mb.refIdxCount, wantRefIdxCount)
		}
		for j := 0; j < mb.refIdxCount; j++ {
			if mb.refIdxFlags[j] != tt.fieldFlag {
				t.Fatalf("%s weighted partitioned-P macroblock[%d] ref_idx_l0[%d] flag = %d, want %d",
					tt.name, i, j, mb.refIdxFlags[j], tt.fieldFlag)
			}
		}
		if tt.mbType == 3 {
			for j, subType := range mb.subMBType {
				if subType != 0 {
					t.Fatalf("%s weighted partitioned-P macroblock[%d] sub_mb_type[%d] = %d, want P_L0_8x8", tt.name, i, j, subType)
				}
			}
		}
	}
}

func readHighFrameMBAFFWeightedCAVLCPartitionedPPair(t *testing.T, nal h264.NALUnit, sps *h264.SPS, pps *h264.PPS, tt highFrameMBAFFWeightedPartitionedPCase) highFrameMBAFFCAVLCPartitionedPPair {
	t.Helper()
	if sps == nil || pps == nil {
		t.Fatal("missing SPS/PPS for frame-MBAFF weighted partitioned-P syntax check")
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
	if fieldPic := br.readBit(t); fieldPic != 0 {
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
	top := readHighFrameMBAFFCAVLCPartitionedPMacroblock(t, &br, topSkipRun, refCount0, "")
	bottomSkipRun := br.readUE(t)
	bottom := readHighFrameMBAFFCAVLCPartitionedPMacroblock(t, &br, bottomSkipRun, refCount0, "")
	return highFrameMBAFFCAVLCPartitionedPPair{fieldFlag: fieldFlag, top: top, bottom: bottom}
}

func assertHighFrameMBAFFWeightedPartitionedPFrames(t *testing.T, frames []*Frame, tt highFrameMBAFFWeightedPartitionedPCase) {
	t.Helper()
	assertHighFrameMBAFFWeightedPFrames(t, frames, highFrameMBAFFWeightedPartitionedPAsWeightedPCase(tt))
}
