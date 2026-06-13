// SPDX-License-Identifier: LGPL-2.1-or-later

package goh264_test

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/thesyncim/goh264/internal/h264"
)

func TestHigh1214FrameMBAFFExplicitWeightedBFixtureSyntax(t *testing.T) {
	for _, tt := range highFrameMBAFFExplicitWeightedBSkipCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFExplicitWeightedBSkipFixture(tt)
			assertHighFrameMBAFFExplicitWeightedBSkipFixtureSyntax(t, data, tt)
		})
	}
	for _, tt := range highFrameMBAFFExplicitWeightedPartitionedBCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFExplicitWeightedPartitionedBFixture(tt)
			assertHighFrameMBAFFExplicitWeightedPartitionedBFixtureSyntax(t, data, tt)
		})
	}
}

func TestDecodeAnnexBHigh1214FrameMBAFFExplicitWeightedBFrames(t *testing.T) {
	for _, tt := range highFrameMBAFFExplicitWeightedBSkipCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFExplicitWeightedBSkipFixture(tt)
			assertHighFrameMBAFFExplicitWeightedBSkipFixtureSyntax(t, data, tt)

			frames, err := NewDecoder().DecodeAnnexBFrames(data)
			if err != nil {
				t.Fatalf("DecodeAnnexBFrames: %v", err)
			}
			assertHighFrameMBAFFBSkipFrames(t, frames, tt)
		})
	}
	for _, tt := range highFrameMBAFFExplicitWeightedPartitionedBCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFExplicitWeightedPartitionedBFixture(tt)
			assertHighFrameMBAFFExplicitWeightedPartitionedBFixtureSyntax(t, data, tt)

			frames, err := NewDecoder().DecodeAnnexBFrames(data)
			if err != nil {
				t.Fatalf("DecodeAnnexBFrames: %v", err)
			}
			assertHighFrameMBAFFPartitionedBFrames(t, frames, tt)
		})
	}
}

func TestDecodeAVCHigh1214FrameMBAFFExplicitWeightedBFrames(t *testing.T) {
	for _, tt := range highFrameMBAFFExplicitWeightedBSkipCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFExplicitWeightedBSkipFixture(tt)
			assertHighFrameMBAFFExplicitWeightedBSkipFixtureSyntax(t, data, tt)

			for _, nalLengthSize := range []int{2, 3, 4} {
				frames, err := NewDecoder().DecodeAVCFrames(annexBToAVC(t, data, nalLengthSize), nalLengthSize)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: DecodeAVCFrames: %v", nalLengthSize, err)
				}
				assertHighFrameMBAFFBSkipFrames(t, frames, tt)
			}
		})
	}
	for _, tt := range highFrameMBAFFExplicitWeightedPartitionedBCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFExplicitWeightedPartitionedBFixture(tt)
			assertHighFrameMBAFFExplicitWeightedPartitionedBFixtureSyntax(t, data, tt)

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

func TestDecodeAVCWithConfigurationRecordHigh1214FrameMBAFFExplicitWeightedBFrames(t *testing.T) {
	for _, tt := range highFrameMBAFFExplicitWeightedBSkipCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFExplicitWeightedBSkipFixture(tt)
			assertHighFrameMBAFFExplicitWeightedBSkipFixtureSyntax(t, data, tt)

			for _, nalLengthSize := range []int{2, 3, 4} {
				config, packet := annexBToAVCConfigAndPacket(t, data, nalLengthSize)
				frames, err := NewDecoder().DecodeAVCFramesWithConfigurationRecord(config, packet)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: DecodeAVCFramesWithConfigurationRecord: %v", nalLengthSize, err)
				}
				assertHighFrameMBAFFBSkipFrames(t, frames, tt)
			}
		})
	}
	for _, tt := range highFrameMBAFFExplicitWeightedPartitionedBCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFExplicitWeightedPartitionedBFixture(tt)
			assertHighFrameMBAFFExplicitWeightedPartitionedBFixtureSyntax(t, data, tt)

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

func TestDecodeConfiguredAVCHigh1214FrameMBAFFExplicitWeightedBFramesAcrossSamplesFlush(t *testing.T) {
	for _, tt := range highFrameMBAFFExplicitWeightedBSkipCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFExplicitWeightedBSkipFixture(tt)
			assertHighFrameMBAFFExplicitWeightedBSkipFixtureSyntax(t, data, tt)
			assertDecodeConfiguredAVCHighFrameMBAFFExplicitWeightedBFrames(t, data, tt.name, func(frames []*Frame) {
				assertHighFrameMBAFFBSkipFrames(t, frames, tt)
			})
		})
	}
	for _, tt := range highFrameMBAFFExplicitWeightedPartitionedBCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFExplicitWeightedPartitionedBFixture(tt)
			assertHighFrameMBAFFExplicitWeightedPartitionedBFixtureSyntax(t, data, tt)
			assertDecodeConfiguredAVCHighFrameMBAFFExplicitWeightedBFrames(t, data, tt.name, func(frames []*Frame) {
				assertHighFrameMBAFFPartitionedBFrames(t, frames, tt)
			})
		})
	}
}

func TestFFmpegRawVideoFrameMD5OracleHigh1214FrameMBAFFExplicitWeightedB(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}

	for _, tt := range highFrameMBAFFExplicitWeightedBSkipCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFExplicitWeightedBSkipFixture(tt)
			assertHighFrameMBAFFExplicitWeightedBSkipFixtureSyntax(t, data, tt)
			assertFFmpegHighFrameMBAFFBSkipRawVideoOracle(t, data, tt)
		})
	}
	for _, tt := range highFrameMBAFFExplicitWeightedPartitionedBCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFExplicitWeightedPartitionedBFixture(tt)
			assertHighFrameMBAFFExplicitWeightedPartitionedBFixtureSyntax(t, data, tt)
			assertFFmpegHighFrameMBAFFPartitionedBRawVideoOracle(t, data, tt)
		})
	}
}

func highFrameMBAFFExplicitWeightedBSkipCases() []highFrameMBAFFBSkipCase {
	bitstreamMD5 := map[string]string{
		"High12TemporalDirectExplicitWeightedBSkipNoDeblock":     "f387950f3aae904ae8ca650e9dd5b4e4",
		"High12TemporalDirectExplicitWeightedBSkipFrameDeblock":  "14037f8c4c0cf7e9b629365c0b3af10f",
		"High12TemporalDirectExplicitWeightedBSkipSliceBoundary": "3e973254baf0f09cd01d558a2437a23b",
		"High12SpatialDirectExplicitWeightedBSkipNoDeblock":      "8b08ffbbbf501750ddfb441349557537",
		"High12SpatialDirectExplicitWeightedBSkipFrameDeblock":   "b1863596e544d767633bd1170f1dba3a",
		"High12SpatialDirectExplicitWeightedBSkipSliceBoundary":  "59ae2699aa10eb357896d474f2b39d02",
		"High14TemporalDirectExplicitWeightedBSkipNoDeblock":     "ad968a7b505411b38271e0240ed0fd80",
		"High14TemporalDirectExplicitWeightedBSkipFrameDeblock":  "cf07a29ba25548abf3db39c7823b7a48",
		"High14TemporalDirectExplicitWeightedBSkipSliceBoundary": "b1dbf2b57bad30f1a2f0512fb2c45c5c",
		"High14SpatialDirectExplicitWeightedBSkipNoDeblock":      "49037c0a4927adb0208c67a7675a55ca",
		"High14SpatialDirectExplicitWeightedBSkipFrameDeblock":   "5c8266f945c42e93c5c3d4ab1e507f7c",
		"High14SpatialDirectExplicitWeightedBSkipSliceBoundary":  "cc1103b4cb72839a1055efa99cd9ace6",
	}
	var out []highFrameMBAFFBSkipCase
	for _, tt := range highFrameMBAFFBSkipCases() {
		tt.name = strings.Replace(tt.name, "BSkip", "ExplicitWeightedBSkip", 1)
		tt.bitstreamMD5 = highFrameMBAFFExplicitWeightedBBitstreamMD5(bitstreamMD5, tt.name)
		out = append(out, tt)
	}
	return out
}

func highFrameMBAFFExplicitWeightedPartitionedBCases() []highFrameMBAFFPartitionedBCase {
	bitstreamMD5 := map[string]string{
		"High12FieldExplicitWeightedB16x8BiNoDeblock":     "1c8b7b53fe0335fb25c58a9d384802b5",
		"High12FieldExplicitWeightedB16x8BiFrameDeblock":  "047e2503c26a8b4954574462ecb7ee83",
		"High12FieldExplicitWeightedB16x8BiSliceBoundary": "5b00679ffc78efac43bdcf5101077645",
		"High12FieldExplicitWeightedB8x16BiNoDeblock":     "0a87f372dc63754796dd82cea7e557a8",
		"High12FieldExplicitWeightedB8x16BiFrameDeblock":  "1b4aefac46a48f5f4b97d0823c91a58d",
		"High12FieldExplicitWeightedB8x16BiSliceBoundary": "dcc32ff67692ef7686bdb51398c9bfc7",
		"High12FieldExplicitWeightedB8x8BiNoDeblock":      "a7db641a790d5c45d6828c9573a15056",
		"High12FieldExplicitWeightedB8x8BiFrameDeblock":   "d28e693d408deda90a535d741fecdab3",
		"High12FieldExplicitWeightedB8x8BiSliceBoundary":  "20d07dd8f3d9bb1f5773b84800f70bee",
		"High12FrameExplicitWeightedB16x8BiNoDeblock":     "764f764caf9aab8d5a65f393cd63c24b",
		"High12FrameExplicitWeightedB16x8BiFrameDeblock":  "0023ce2bb472236a3ffd779f10c82803",
		"High12FrameExplicitWeightedB16x8BiSliceBoundary": "61a15ef5d94025797df61ea46094e31c",
		"High12FrameExplicitWeightedB8x16BiNoDeblock":     "0c6d0b67d9dbd3afc7f3933c2ecceb4d",
		"High12FrameExplicitWeightedB8x16BiFrameDeblock":  "1d881da523eab46e3a2e4bc58f4a4ddb",
		"High12FrameExplicitWeightedB8x16BiSliceBoundary": "421728880f3c711302f632d9a3407fe6",
		"High12FrameExplicitWeightedB8x8BiNoDeblock":      "10dd97f281c2d26c27c5f6ccafcfa6e6",
		"High12FrameExplicitWeightedB8x8BiFrameDeblock":   "ece2a1bfd96789b1dc22525104c93fd4",
		"High12FrameExplicitWeightedB8x8BiSliceBoundary":  "c7ccae2259d9d87f9969fe6ce48f3b76",
		"High14FieldExplicitWeightedB16x8BiNoDeblock":     "bca3690a1b50bfccd97c3f3b95e35de8",
		"High14FieldExplicitWeightedB16x8BiFrameDeblock":  "aa020e916dc1e7e40f5f5ce98cd570a5",
		"High14FieldExplicitWeightedB16x8BiSliceBoundary": "ee6401f59009a051a911243f18bbc5ef",
		"High14FieldExplicitWeightedB8x16BiNoDeblock":     "3dc2cf4c018ea90ad82b0bdf7c29e512",
		"High14FieldExplicitWeightedB8x16BiFrameDeblock":  "ae9775a7b5f0653f596888b96def0a6a",
		"High14FieldExplicitWeightedB8x16BiSliceBoundary": "46e0e29e79ddb0f7e0830b899690bfff",
		"High14FieldExplicitWeightedB8x8BiNoDeblock":      "f0b4a12ec81f017ea1bac760f3dee719",
		"High14FieldExplicitWeightedB8x8BiFrameDeblock":   "62ae91205ed096b29c523adfe684bf64",
		"High14FieldExplicitWeightedB8x8BiSliceBoundary":  "0f892f1dbfea5352a711b6594d8b77c0",
		"High14FrameExplicitWeightedB16x8BiNoDeblock":     "f0e730cf18ceead87d73d1f14dd94e04",
		"High14FrameExplicitWeightedB16x8BiFrameDeblock":  "a89be44c90156934aed8c09fd40b53b1",
		"High14FrameExplicitWeightedB16x8BiSliceBoundary": "40c95ab88388f588a08b5aa15a36dde1",
		"High14FrameExplicitWeightedB8x16BiNoDeblock":     "c7403d4233ff73772652edfc20544aea",
		"High14FrameExplicitWeightedB8x16BiFrameDeblock":  "ad3085cc2e1d0aaa8541a43a8aff8b87",
		"High14FrameExplicitWeightedB8x16BiSliceBoundary": "20bafc2eb1af20e94363f7578f5d5a1f",
		"High14FrameExplicitWeightedB8x8BiNoDeblock":      "ff72e54d8828f7c5fc73af5e80d2d260",
		"High14FrameExplicitWeightedB8x8BiFrameDeblock":   "4d1be6c6142912bca4d5fed6b7674e15",
		"High14FrameExplicitWeightedB8x8BiSliceBoundary":  "9965eac2feedef0444ee9c28207e2699",
	}
	var out []highFrameMBAFFPartitionedBCase
	for _, tt := range highFrameMBAFFPartitionedBCases() {
		tt.name = strings.Replace(tt.name, "B", "ExplicitWeightedB", 1)
		tt.bitstreamMD5 = highFrameMBAFFExplicitWeightedBBitstreamMD5(bitstreamMD5, tt.name)
		out = append(out, tt)
	}
	return out
}

func highFrameMBAFFExplicitWeightedBBitstreamMD5(bitstreamMD5 map[string]string, name string) string {
	got, ok := bitstreamMD5[name]
	if !ok {
		panic(fmt.Sprintf("missing explicit weighted-B bitstream md5 for %s", name))
	}
	return got
}

func highFrameMBAFFExplicitWeightedBSkipFixture(tt highFrameMBAFFBSkipCase) []byte {
	var data []byte
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALSPS), highFrameMBAFFBInterSPSRBSP(tt.bitDepth)))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALPPS), highFrameMBAFFExplicitWeightedBPPSRBSP(tt.bitDepth)))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALIDRSlice), highFrameMBAFFIntraPCMSliceRBSP(tt.bitDepth)))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALSlice), highFrameMBAFFPSkipNoResidualSliceRBSP()))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(h264.NALSlice), highFrameMBAFFExplicitWeightedBSkipSliceRBSP(2, tt.directSpatial, tt.disableDeblockingFilterIDC)))
	return data
}

func highFrameMBAFFExplicitWeightedPartitionedBFixture(tt highFrameMBAFFPartitionedBCase) []byte {
	var data []byte
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALSPS), highFrameMBAFFBInterSPSRBSP(tt.bitDepth)))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALPPS), highFrameMBAFFExplicitWeightedBPPSRBSP(tt.bitDepth)))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALIDRSlice), highFrameMBAFFIntraPCMSliceRBSP(tt.bitDepth)))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALSlice), highFrameMBAFFPSkipNoResidualSliceRBSP()))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(h264.NALSlice), highFrameMBAFFExplicitWeightedPartitionedBSliceRBSP(tt.payloadBits, tt.disableDeblockingFilterIDC)))
	return data
}

func highFrameMBAFFExplicitWeightedBPPSRBSP(bitDepth int) []byte {
	var b decoderSEIBitBuilder
	b.writeUE(0)
	b.writeUE(0)
	b.writeBit(0)
	b.writeBit(0)
	b.writeUE(0)
	b.writeUE(0)
	b.writeUE(0)
	b.writeBit(0)
	b.writeBits(1, 2)
	b.writeSE(int32(-6 * (bitDepth - 8)))
	b.writeSE(0)
	b.writeSE(0)
	b.writeBit(1)
	b.writeBit(0)
	b.writeBit(0)
	b.writeBit(0)
	b.writeBit(0)
	b.writeSE(0)
	return b.rbsp()
}

func highFrameMBAFFExplicitWeightedBSkipSliceRBSP(frameNum uint32, directSpatial uint32, disableDeblockingFilterIDC uint32) []byte {
	var b decoderSEIBitBuilder
	b.writeUE(0)
	b.writeUE(1)
	b.writeUE(0)
	b.writeBits(frameNum, 4)
	b.writeBit(0)
	b.writeBit(directSpatial)
	b.writeBit(0)
	b.writeBit(0)
	b.writeBit(0)
	writeHighFrameMBAFFExplicitWeightedBPredWeightSyntax(&b)
	b.writeSE(0)
	writeHighCAVLCDeblockSyntax(&b, disableDeblockingFilterIDC)
	b.writeUE(2)
	return b.rbsp()
}

func highFrameMBAFFExplicitWeightedPartitionedBSliceRBSP(payloadBits string, disableDeblockingFilterIDC uint32) []byte {
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
	writeHighFrameMBAFFExplicitWeightedBPredWeightSyntax(&b)
	b.writeSE(0)
	writeHighCAVLCDeblockSyntax(&b, disableDeblockingFilterIDC)
	highIntra16x16WritePayloadBits(&b, payloadBits)
	return b.rbsp()
}

func writeHighFrameMBAFFExplicitWeightedBPredWeightSyntax(b *decoderSEIBitBuilder) {
	b.writeUE(1)
	b.writeUE(1)
	for list := 0; list < 2; list++ {
		lumaWeight, lumaOffset := int32(3), int32(1)
		chroma0Weight, chroma0Offset := int32(3), int32(1)
		chroma1Weight, chroma1Offset := int32(3), int32(-1)
		if list == 1 {
			lumaWeight, lumaOffset = 1, -1
			chroma0Weight, chroma0Offset = 1, -1
			chroma1Weight, chroma1Offset = 1, 1
		}
		b.writeBit(1)
		b.writeSE(lumaWeight)
		b.writeSE(lumaOffset)
		b.writeBit(1)
		b.writeSE(chroma0Weight)
		b.writeSE(chroma0Offset)
		b.writeSE(chroma1Weight)
		b.writeSE(chroma1Offset)
	}
}

func assertDecodeConfiguredAVCHighFrameMBAFFExplicitWeightedBFrames(t *testing.T, data []byte, name string, assert func([]*Frame)) {
	t.Helper()
	for _, nalLengthSize := range []int{2, 3, 4} {
		config, samples := annexBToAVCConfigAndSamples(t, data, nalLengthSize)
		if len(samples) != 3 {
			t.Fatalf("%s nalLengthSize=%d samples = %d, want IDR/P/B", name, nalLengthSize, len(samples))
		}
		dec := NewDecoder()
		if _, err := dec.ConfigureAVCDecoderConfigurationRecord(config); err != nil {
			t.Fatalf("%s nalLengthSize=%d config: %v", name, nalLengthSize, err)
		}
		var frames []*Frame
		for i, sample := range samples {
			out, err := dec.DecodeConfiguredAVCFrames(sample)
			if err != nil {
				t.Fatalf("%s nalLengthSize=%d sample[%d]: DecodeConfiguredAVCFrames: %v", name, nalLengthSize, i, err)
			}
			frames = append(frames, out...)
		}
		out, err := dec.FlushDelayedFrames()
		if err != nil {
			t.Fatalf("%s nalLengthSize=%d flush: %v", name, nalLengthSize, err)
		}
		frames = append(frames, out...)
		assert(frames)
	}
}

func assertHighFrameMBAFFExplicitWeightedBSkipFixtureSyntax(t *testing.T, data []byte, tt highFrameMBAFFBSkipCase) {
	t.Helper()
	sum := md5.Sum(data)
	if got := hex.EncodeToString(sum[:]); got != tt.bitstreamMD5 {
		t.Fatalf("%s frame-MBAFF explicit weighted B-skip bitstream md5 = %s, want %s", tt.name, got, tt.bitstreamMD5)
	}

	bNAL, sps, pps := parseHighFrameMBAFFExplicitWeightedBFixtureSyntax(t, data, tt.name, tt.bitDepth, tt.directSpatial, tt.deblockMode)
	skipRun := readHighFrameMBAFFExplicitWeightedBSkipRun(t, bNAL, sps, pps, tt)
	if skipRun != 2 {
		t.Fatalf("%s B mb_skip_run = %d, want frame-coded pair skip_run 2", tt.name, skipRun)
	}
}

func assertHighFrameMBAFFExplicitWeightedPartitionedBFixtureSyntax(t *testing.T, data []byte, tt highFrameMBAFFPartitionedBCase) {
	t.Helper()
	sum := md5.Sum(data)
	if got := hex.EncodeToString(sum[:]); got != tt.bitstreamMD5 {
		t.Fatalf("%s frame-MBAFF explicit weighted partitioned-B bitstream md5 = %s, want %s", tt.name, got, tt.bitstreamMD5)
	}

	bNAL, sps, pps := parseHighFrameMBAFFExplicitWeightedBFixtureSyntax(t, data, tt.name, tt.bitDepth, 1, tt.deblockMode)
	pair := readHighFrameMBAFFExplicitWeightedCAVLCPartitionedBPair(t, bNAL, sps, pps, tt)
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

func parseHighFrameMBAFFExplicitWeightedBFixtureSyntax(t *testing.T, data []byte, name string, bitDepth int, directSpatial uint32, deblockMode int32) (h264.NALUnit, *h264.SPS, *h264.PPS) {
	t.Helper()
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
				sps.ChromaFormatIDC != 1 || sps.BitDepthLuma != int32(bitDepth) ||
				sps.BitDepthChroma != int32(bitDepth) || sps.RefFrameCount != 2 ||
				sps.FrameMBSOnlyFlag != 0 || sps.MBAFF != 1 || sps.Direct8x8InferenceFlag != 1 {
				t.Fatalf("SPS profile/format = %d %dx%d chroma %d depth %d/%d refs %d frame_mbs_only:%d mbaff:%d direct8x8:%d, want High%d 16x32 4:2:0 frame-MBAFF refs=2 direct8x8",
					sps.ProfileIDC, sps.Width, sps.Height, sps.ChromaFormatIDC,
					sps.BitDepthLuma, sps.BitDepthChroma, sps.RefFrameCount,
					sps.FrameMBSOnlyFlag, sps.MBAFF, sps.Direct8x8InferenceFlag, bitDepth)
			}
			spsList[sps.SPSID] = sps
		case h264.NALPPS:
			pps, err := h264.DecodePPS(nal.RBSP, &spsList)
			if err != nil {
				t.Fatal(err)
			}
			if pps.CABAC != 0 || pps.Transform8x8Mode != 0 || pps.WeightedPred != 0 ||
				pps.WeightedBipredIDC != 1 || pps.RefCount != [2]uint32{1, 1} ||
				pps.DeblockingFilterParametersPresent == 0 {
				t.Fatalf("PPS CABAC/8x8/weights/refs/deblock = %d/%d/%d/%d/%v/%d, want CAVLC/no-8x8/explicit-weighted-B refs=1/1 deblock params",
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
					sh.DirectSpatialMVPred != int32(directSpatial) || sh.DeblockingFilter != deblockMode ||
					sh.PPS.WeightedBipredIDC != 1 {
					t.Fatalf("B ref/lists/refs/direct/deblock/weights = %d/%d/%v/%d/%d/%d, want non-ref explicit-weighted B refs=1/1 direct=%d deblock=%d",
						nal.RefIDC, sh.ListCount, sh.RefCount, sh.DirectSpatialMVPred, sh.DeblockingFilter,
						sh.PPS.WeightedBipredIDC, directSpatial, deblockMode)
				}
				assertHighFrameMBAFFExplicitWeightedBPredWeight(t, sh.PredWeightTable)
				bNAL = nal
			}
			gotTypes = append(gotTypes, sh.SliceTypeNoS)
		default:
			t.Fatalf("unexpected NAL type %d in %s frame-MBAFF explicit weighted-B fixture", nal.Type, name)
		}
	}
	wantTypes := []int32{h264.PictureTypeI, h264.PictureTypeP, h264.PictureTypeB}
	if !highFrameMBAFFBSkipInt32SlicesEqual(gotTypes, wantTypes) {
		t.Fatalf("slice types = %v, want %v", gotTypes, wantTypes)
	}
	if bNAL.RBSP == nil {
		t.Fatal("missing B slice")
	}
	return bNAL, spsList[0], ppsList[0]
}

func assertHighFrameMBAFFExplicitWeightedBPredWeight(t *testing.T, pwt h264.PredWeightTable) {
	t.Helper()
	if pwt.UseWeight != 1 || pwt.UseWeightChroma != 1 ||
		pwt.LumaLog2WeightDenom != 1 || pwt.ChromaLog2WeightDenom != 1 {
		t.Fatalf("pred weights flags/denom = %d/%d/%d/%d, want explicit luma+chroma denom 1/1",
			pwt.UseWeight, pwt.UseWeightChroma, pwt.LumaLog2WeightDenom, pwt.ChromaLog2WeightDenom)
	}
	wantLuma := [2][2]int32{{3, 1}, {1, -1}}
	wantChroma := [2][2][2]int32{
		{{3, 1}, {3, -1}},
		{{1, -1}, {1, 1}},
	}
	for list := 0; list < 2; list++ {
		for _, ref := range []int{0, 16, 17} {
			if got := pwt.LumaWeight[ref][list]; got != wantLuma[list] {
				t.Fatalf("luma weight ref %d list %d = %v, want %v", ref, list, got, wantLuma[list])
			}
			for chroma := 0; chroma < 2; chroma++ {
				if got := pwt.ChromaWeight[ref][list][chroma]; got != wantChroma[list][chroma] {
					t.Fatalf("chroma weight ref %d list %d comp %d = %v, want %v",
						ref, list, chroma, got, wantChroma[list][chroma])
				}
			}
		}
	}
}

func readHighFrameMBAFFExplicitWeightedBSkipRun(t *testing.T, nal h264.NALUnit, sps *h264.SPS, pps *h264.PPS, tt highFrameMBAFFBSkipCase) uint32 {
	t.Helper()
	if sps == nil || pps == nil {
		t.Fatal("missing SPS/PPS for frame-MBAFF explicit weighted B-skip syntax check")
	}
	br := high10ResidualCAVLCBitReader{data: nal.RBSP}
	firstMB := br.readUE(t)
	rawSliceType := br.readUE(t)
	sliceTypeNoS := high10ResidualCAVLCSliceTypeNoS(t, rawSliceType)
	ppsID := br.readUE(t)
	if firstMB != 0 || sliceTypeNoS != h264.PictureTypeB || ppsID != pps.PPSID {
		t.Fatalf("slice header first_mb/type/pps = %d/%d/%d, want first B-skip slice with PPS %d", firstMB, sliceTypeNoS, ppsID, pps.PPSID)
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
	readHighFrameMBAFFExplicitWeightedBPredWeightSyntax(t, &br)
	high10ResidualCAVLCReadRefPicMarking(t, &br, nal)
	if pps.CABAC != 0 {
		br.readUE(t)
	}
	if qpDelta := br.readSE(t); qpDelta != 0 {
		t.Fatalf("slice_qp_delta = %d, want 0", qpDelta)
	}
	readHighFrameMBAFFExplicitWeightedBDeblockSyntax(t, &br, pps, tt.disableDeblockingFilterIDC)
	return br.readUE(t)
}

func readHighFrameMBAFFExplicitWeightedCAVLCPartitionedBPair(t *testing.T, nal h264.NALUnit, sps *h264.SPS, pps *h264.PPS, tt highFrameMBAFFPartitionedBCase) highFrameMBAFFCAVLCPartitionedBPair {
	t.Helper()
	if sps == nil || pps == nil {
		t.Fatal("missing SPS/PPS for frame-MBAFF explicit weighted partitioned B syntax check")
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
	readHighFrameMBAFFExplicitWeightedBPredWeightSyntax(t, &br)
	high10ResidualCAVLCReadRefPicMarking(t, &br, nal)
	if pps.CABAC != 0 {
		br.readUE(t)
	}
	if qpDelta := br.readSE(t); qpDelta != 0 {
		t.Fatalf("slice_qp_delta = %d, want 0", qpDelta)
	}
	readHighFrameMBAFFExplicitWeightedBDeblockSyntax(t, &br, pps, tt.disableDeblockingFilterIDC)

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

func readHighFrameMBAFFExplicitWeightedBPredWeightSyntax(t *testing.T, br *high10ResidualCAVLCBitReader) {
	t.Helper()
	if denom := br.readUE(t); denom != 1 {
		t.Fatalf("luma_log2_weight_denom = %d, want 1", denom)
	}
	if denom := br.readUE(t); denom != 1 {
		t.Fatalf("chroma_log2_weight_denom = %d, want 1", denom)
	}
	for list := 0; list < 2; list++ {
		lumaWeight, lumaOffset := int32(3), int32(1)
		chroma0Weight, chroma0Offset := int32(3), int32(1)
		chroma1Weight, chroma1Offset := int32(3), int32(-1)
		if list == 1 {
			lumaWeight, lumaOffset = 1, -1
			chroma0Weight, chroma0Offset = 1, -1
			chroma1Weight, chroma1Offset = 1, 1
		}
		if flag := br.readBit(t); flag != 1 {
			t.Fatalf("luma_weight_l%d_flag = %d, want 1", list, flag)
		}
		if weight := br.readSE(t); weight != lumaWeight {
			t.Fatalf("luma_weight_l%d[0] = %d, want %d", list, weight, lumaWeight)
		}
		if offset := br.readSE(t); offset != lumaOffset {
			t.Fatalf("luma_offset_l%d[0] = %d, want %d", list, offset, lumaOffset)
		}
		if flag := br.readBit(t); flag != 1 {
			t.Fatalf("chroma_weight_l%d_flag = %d, want 1", list, flag)
		}
		for i, want := range []int32{chroma0Weight, chroma0Offset, chroma1Weight, chroma1Offset} {
			if got := br.readSE(t); got != want {
				t.Fatalf("chroma pred weight list %d value[%d] = %d, want %d", list, i, got, want)
			}
		}
	}
}

func readHighFrameMBAFFExplicitWeightedBDeblockSyntax(t *testing.T, br *high10ResidualCAVLCBitReader, pps *h264.PPS, wantDisableID uint32) {
	t.Helper()
	if pps.DeblockingFilterParametersPresent == 0 {
		return
	}
	disableID := br.readUE(t)
	if disableID != wantDisableID {
		t.Fatalf("disable_deblocking_filter_idc = %d, want %d", disableID, wantDisableID)
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
