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

func TestHigh1214FrameMBAFFImplicitWeightedBFixtureSyntax(t *testing.T) {
	for _, tt := range highFrameMBAFFImplicitWeightedBSkipCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFImplicitWeightedBSkipFixture(tt)
			assertHighFrameMBAFFImplicitWeightedBSkipFixtureSyntax(t, data, tt)
		})
	}
	for _, tt := range highFrameMBAFFImplicitWeightedPartitionedBCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFImplicitWeightedPartitionedBFixture(tt)
			assertHighFrameMBAFFImplicitWeightedPartitionedBFixtureSyntax(t, data, tt)
		})
	}
}

func TestDecodeAnnexBHigh1214FrameMBAFFImplicitWeightedBFrames(t *testing.T) {
	for _, tt := range highFrameMBAFFImplicitWeightedBSkipCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFImplicitWeightedBSkipFixture(tt)
			assertHighFrameMBAFFImplicitWeightedBSkipFixtureSyntax(t, data, tt)

			frames, err := NewDecoder().DecodeAnnexBFrames(data)
			if err != nil {
				t.Fatalf("DecodeAnnexBFrames: %v", err)
			}
			assertHighFrameMBAFFBSkipFrames(t, frames, tt)
		})
	}
	for _, tt := range highFrameMBAFFImplicitWeightedPartitionedBCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFImplicitWeightedPartitionedBFixture(tt)
			assertHighFrameMBAFFImplicitWeightedPartitionedBFixtureSyntax(t, data, tt)

			frames, err := NewDecoder().DecodeAnnexBFrames(data)
			if err != nil {
				t.Fatalf("DecodeAnnexBFrames: %v", err)
			}
			assertHighFrameMBAFFPartitionedBFrames(t, frames, tt)
		})
	}
}

func TestDecodeAVCHigh1214FrameMBAFFImplicitWeightedBFrames(t *testing.T) {
	for _, tt := range highFrameMBAFFImplicitWeightedBSkipCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFImplicitWeightedBSkipFixture(tt)
			assertHighFrameMBAFFImplicitWeightedBSkipFixtureSyntax(t, data, tt)

			for _, nalLengthSize := range []int{2, 3, 4} {
				frames, err := NewDecoder().DecodeAVCFrames(annexBToAVC(t, data, nalLengthSize), nalLengthSize)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: DecodeAVCFrames: %v", nalLengthSize, err)
				}
				assertHighFrameMBAFFBSkipFrames(t, frames, tt)
			}
		})
	}
	for _, tt := range highFrameMBAFFImplicitWeightedPartitionedBCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFImplicitWeightedPartitionedBFixture(tt)
			assertHighFrameMBAFFImplicitWeightedPartitionedBFixtureSyntax(t, data, tt)

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

func TestDecodeAVCWithConfigurationRecordHigh1214FrameMBAFFImplicitWeightedBFrames(t *testing.T) {
	for _, tt := range highFrameMBAFFImplicitWeightedBSkipCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFImplicitWeightedBSkipFixture(tt)
			assertHighFrameMBAFFImplicitWeightedBSkipFixtureSyntax(t, data, tt)

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
	for _, tt := range highFrameMBAFFImplicitWeightedPartitionedBCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFImplicitWeightedPartitionedBFixture(tt)
			assertHighFrameMBAFFImplicitWeightedPartitionedBFixtureSyntax(t, data, tt)

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

func TestDecodeConfiguredAVCHigh1214FrameMBAFFImplicitWeightedBFramesAcrossSamplesFlush(t *testing.T) {
	for _, tt := range highFrameMBAFFImplicitWeightedBSkipCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFImplicitWeightedBSkipFixture(tt)
			assertHighFrameMBAFFImplicitWeightedBSkipFixtureSyntax(t, data, tt)
			assertDecodeConfiguredAVCHighFrameMBAFFImplicitWeightedBFrames(t, data, tt.name, func(frames []*Frame) {
				assertHighFrameMBAFFBSkipFrames(t, frames, tt)
			})
		})
	}
	for _, tt := range highFrameMBAFFImplicitWeightedPartitionedBCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFImplicitWeightedPartitionedBFixture(tt)
			assertHighFrameMBAFFImplicitWeightedPartitionedBFixtureSyntax(t, data, tt)
			assertDecodeConfiguredAVCHighFrameMBAFFImplicitWeightedBFrames(t, data, tt.name, func(frames []*Frame) {
				assertHighFrameMBAFFPartitionedBFrames(t, frames, tt)
			})
		})
	}
}

func TestFFmpegRawVideoFrameMD5OracleHigh1214FrameMBAFFImplicitWeightedB(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}

	for _, tt := range highFrameMBAFFImplicitWeightedBSkipCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFImplicitWeightedBSkipFixture(tt)
			assertHighFrameMBAFFImplicitWeightedBSkipFixtureSyntax(t, data, tt)
			assertFFmpegHighFrameMBAFFBSkipRawVideoOracle(t, data, tt)
		})
	}
	for _, tt := range highFrameMBAFFImplicitWeightedPartitionedBCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFImplicitWeightedPartitionedBFixture(tt)
			assertHighFrameMBAFFImplicitWeightedPartitionedBFixtureSyntax(t, data, tt)
			assertFFmpegHighFrameMBAFFPartitionedBRawVideoOracle(t, data, tt)
		})
	}
}

func highFrameMBAFFImplicitWeightedBSkipCases() []highFrameMBAFFBSkipCase {
	bitstreamMD5 := map[string]string{
		"High12TemporalDirectImplicitWeightedBSkipNoDeblock":     "adbcb3149c7252442f4003eceefb4b34",
		"High12TemporalDirectImplicitWeightedBSkipFrameDeblock":  "fc7df318519feef056833ec02f198269",
		"High12TemporalDirectImplicitWeightedBSkipSliceBoundary": "d68dd02d1f0c1c2b907fe570b3181eb7",
		"High12SpatialDirectImplicitWeightedBSkipNoDeblock":      "f7540bf8b6dad827bce05337d95ac410",
		"High12SpatialDirectImplicitWeightedBSkipFrameDeblock":   "ef59b242d26996180a4f1604140a2ecf",
		"High12SpatialDirectImplicitWeightedBSkipSliceBoundary":  "00c87de85c67e620229cc6b3104b7af0",
		"High14TemporalDirectImplicitWeightedBSkipNoDeblock":     "d7953322e9663e2218c0600bf8305ae2",
		"High14TemporalDirectImplicitWeightedBSkipFrameDeblock":  "80210d66161211886ea6e8ac0505604f",
		"High14TemporalDirectImplicitWeightedBSkipSliceBoundary": "0de7a55b588f8dc8e62a266a13e1b181",
		"High14SpatialDirectImplicitWeightedBSkipNoDeblock":      "62c91f5046e46571d34764b98d169d08",
		"High14SpatialDirectImplicitWeightedBSkipFrameDeblock":   "c5b7da7522d2f3597e6a37ef23a5d3a7",
		"High14SpatialDirectImplicitWeightedBSkipSliceBoundary":  "0b0bfde27c00eda7aff3dea86786d715",
	}
	var out []highFrameMBAFFBSkipCase
	for _, tt := range highFrameMBAFFBSkipCases() {
		tt.name = strings.Replace(tt.name, "BSkip", "ImplicitWeightedBSkip", 1)
		tt.bitstreamMD5 = highFrameMBAFFImplicitWeightedBBitstreamMD5(bitstreamMD5, tt.name)
		out = append(out, tt)
	}
	return out
}

func highFrameMBAFFImplicitWeightedPartitionedBCases() []highFrameMBAFFPartitionedBCase {
	bitstreamMD5 := map[string]string{
		"High12FieldImplicitWeightedB16x8BiNoDeblock":     "c785d0ef7e0d1307c0db5eacc98dd019",
		"High12FieldImplicitWeightedB16x8BiFrameDeblock":  "93e46ad8d579f591a9d9294a2d510fe3",
		"High12FieldImplicitWeightedB16x8BiSliceBoundary": "b4e1b9d564ba1d2b0eb048c9ef34fffb",
		"High12FieldImplicitWeightedB8x16BiNoDeblock":     "d08580f5209289dce18a795a9dd6acae",
		"High12FieldImplicitWeightedB8x16BiFrameDeblock":  "aec220aecd2f9c672d1fadefb238ce62",
		"High12FieldImplicitWeightedB8x16BiSliceBoundary": "298c7eee878dc1c3bda4a0e7205fec5c",
		"High12FieldImplicitWeightedB8x8BiNoDeblock":      "b7ef81a95895d35740c0d99849f97442",
		"High12FieldImplicitWeightedB8x8BiFrameDeblock":   "9a45bf7ff719b3275f23ea7347f07874",
		"High12FieldImplicitWeightedB8x8BiSliceBoundary":  "9bc8e9ce2d3aff6f669431063acdb7d5",
		"High12FrameImplicitWeightedB16x8BiNoDeblock":     "29dd151747fa969fbe834b0db2fd5249",
		"High12FrameImplicitWeightedB16x8BiFrameDeblock":  "a429c47145cc2eec852d16e9ed920795",
		"High12FrameImplicitWeightedB16x8BiSliceBoundary": "4658337497b07145ca56bd9930d78c2c",
		"High12FrameImplicitWeightedB8x16BiNoDeblock":     "4a1500cf57f4cf865b58553a5fe3737b",
		"High12FrameImplicitWeightedB8x16BiFrameDeblock":  "1c09ac81f3df0b68c60a88aca66f1cff",
		"High12FrameImplicitWeightedB8x16BiSliceBoundary": "83f7821b3772a6b2c0410e8e9c843ebc",
		"High12FrameImplicitWeightedB8x8BiNoDeblock":      "92623aeca1964878d80d768abed2f54e",
		"High12FrameImplicitWeightedB8x8BiFrameDeblock":   "c0891821857dbd73e6f24c0253c91c1a",
		"High12FrameImplicitWeightedB8x8BiSliceBoundary":  "d5274e3c68b2f938094ef544576f4f50",
		"High14FieldImplicitWeightedB16x8BiNoDeblock":     "f90a388c6114a4a0210ad4fa742c79c2",
		"High14FieldImplicitWeightedB16x8BiFrameDeblock":  "7a14afeb230396528c86ab959e8194e8",
		"High14FieldImplicitWeightedB16x8BiSliceBoundary": "ba3c508a03df057347057400d4f8d9b1",
		"High14FieldImplicitWeightedB8x16BiNoDeblock":     "06a5f0855ad319a13426a046984baa40",
		"High14FieldImplicitWeightedB8x16BiFrameDeblock":  "1ec1e62b631dbfa280df2aba4aeff147",
		"High14FieldImplicitWeightedB8x16BiSliceBoundary": "f6900487834b6073d9f563457c0d490b",
		"High14FieldImplicitWeightedB8x8BiNoDeblock":      "08ddd37a5d94f10a5a45cbb5b94f5800",
		"High14FieldImplicitWeightedB8x8BiFrameDeblock":   "930ddff8dab70baee918c98cba081ce2",
		"High14FieldImplicitWeightedB8x8BiSliceBoundary":  "711e20030d57e8c7dd324af53642f8e0",
		"High14FrameImplicitWeightedB16x8BiNoDeblock":     "53462e5a263f61025933a9ecb6c4c17f",
		"High14FrameImplicitWeightedB16x8BiFrameDeblock":  "6bc3a2937a256472fe94734e15769f0f",
		"High14FrameImplicitWeightedB16x8BiSliceBoundary": "5264fadd5c0d796ff851b00b9d3497c2",
		"High14FrameImplicitWeightedB8x16BiNoDeblock":     "6ff4177415b4a6e7c76a8029a145e755",
		"High14FrameImplicitWeightedB8x16BiFrameDeblock":  "1af6417462139e2a3b5b27120692e8fd",
		"High14FrameImplicitWeightedB8x16BiSliceBoundary": "c2bf9757bea362c47c1fe3b19e32f205",
		"High14FrameImplicitWeightedB8x8BiNoDeblock":      "b207a25bce2cd9812f6e5cc11857cea9",
		"High14FrameImplicitWeightedB8x8BiFrameDeblock":   "f4a64abc41ea47d355ebea4c587bdadb",
		"High14FrameImplicitWeightedB8x8BiSliceBoundary":  "216466182b7891e38a4d782e1aa9aa38",
	}
	var out []highFrameMBAFFPartitionedBCase
	for _, tt := range highFrameMBAFFPartitionedBCases() {
		tt.name = strings.Replace(tt.name, "B", "ImplicitWeightedB", 1)
		tt.bitstreamMD5 = highFrameMBAFFImplicitWeightedBBitstreamMD5(bitstreamMD5, tt.name)
		out = append(out, tt)
	}
	return out
}

func highFrameMBAFFImplicitWeightedBBitstreamMD5(bitstreamMD5 map[string]string, name string) string {
	got, ok := bitstreamMD5[name]
	if !ok {
		panic(fmt.Sprintf("missing implicit weighted-B bitstream md5 for %s", name))
	}
	return got
}

func highFrameMBAFFImplicitWeightedBSkipFixture(tt highFrameMBAFFBSkipCase) []byte {
	var data []byte
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALSPS), highFrameMBAFFBInterSPSRBSP(tt.bitDepth)))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALPPS), highFrameMBAFFImplicitWeightedBPPSRBSP(tt.bitDepth)))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALIDRSlice), highFrameMBAFFIntraPCMSliceRBSP(tt.bitDepth)))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALSlice), highFrameMBAFFPSkipNoResidualSliceRBSP()))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(h264.NALSlice), highFrameMBAFFBSkipSliceRBSP(2, tt.directSpatial, tt.disableDeblockingFilterIDC)))
	return data
}

func highFrameMBAFFImplicitWeightedPartitionedBFixture(tt highFrameMBAFFPartitionedBCase) []byte {
	var data []byte
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALSPS), highFrameMBAFFBInterSPSRBSP(tt.bitDepth)))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALPPS), highFrameMBAFFImplicitWeightedBPPSRBSP(tt.bitDepth)))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALIDRSlice), highFrameMBAFFIntraPCMSliceRBSP(tt.bitDepth)))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALSlice), highFrameMBAFFPSkipNoResidualSliceRBSP()))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(h264.NALSlice), highFrameMBAFFPartitionedBSliceRBSP(tt.payloadBits, tt.disableDeblockingFilterIDC)))
	return data
}

func highFrameMBAFFImplicitWeightedBPPSRBSP(bitDepth int) []byte {
	var b decoderSEIBitBuilder
	b.writeUE(0)
	b.writeUE(0)
	b.writeBit(0)
	b.writeBit(0)
	b.writeUE(0)
	b.writeUE(0)
	b.writeUE(0)
	b.writeBit(0)
	b.writeBits(2, 2)
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

func assertDecodeConfiguredAVCHighFrameMBAFFImplicitWeightedBFrames(t *testing.T, data []byte, name string, assert func([]*Frame)) {
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

func assertHighFrameMBAFFImplicitWeightedBSkipFixtureSyntax(t *testing.T, data []byte, tt highFrameMBAFFBSkipCase) {
	t.Helper()
	sum := md5.Sum(data)
	if got := hex.EncodeToString(sum[:]); got != tt.bitstreamMD5 {
		t.Fatalf("%s frame-MBAFF implicit weighted B-skip bitstream md5 = %s, want %s", tt.name, got, tt.bitstreamMD5)
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
				pps.WeightedBipredIDC != 2 || pps.RefCount != [2]uint32{1, 1} ||
				pps.DeblockingFilterParametersPresent == 0 {
				t.Fatalf("PPS CABAC/8x8/weights/refs/deblock = %d/%d/%d/%d/%v/%d, want CAVLC/no-8x8/implicit-weighted-B refs=1/1 deblock params",
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
					sh.DirectSpatialMVPred != int32(tt.directSpatial) || sh.DeblockingFilter != tt.deblockMode ||
					sh.PPS.WeightedBipredIDC != 2 || sh.PredWeightTable.UseWeight != 0 ||
					sh.PredWeightTable.UseWeightChroma != 0 {
					t.Fatalf("B ref/lists/refs/direct/deblock/weights = %d/%d/%v/%d/%d/%d/%d/%d, want non-ref implicit-weighted B refs=1/1 direct=%d deblock=%d no serialized weights",
						nal.RefIDC, sh.ListCount, sh.RefCount, sh.DirectSpatialMVPred, sh.DeblockingFilter,
						sh.PPS.WeightedBipredIDC, sh.PredWeightTable.UseWeight, sh.PredWeightTable.UseWeightChroma,
						tt.directSpatial, tt.deblockMode)
				}
				bNAL = nal
			}
			gotTypes = append(gotTypes, sh.SliceTypeNoS)
		default:
			t.Fatalf("unexpected NAL type %d in %s frame-MBAFF implicit weighted B-skip fixture", nal.Type, tt.name)
		}
	}
	wantTypes := []int32{h264.PictureTypeI, h264.PictureTypeP, h264.PictureTypeB}
	if !highFrameMBAFFBSkipInt32SlicesEqual(gotTypes, wantTypes) {
		t.Fatalf("slice types = %v, want %v", gotTypes, wantTypes)
	}
	if bNAL.RBSP == nil {
		t.Fatal("missing B slice")
	}
	skipRun := readHighFrameMBAFFBSkipRun(t, bNAL, spsList[0], ppsList[0], tt)
	if skipRun != 2 {
		t.Fatalf("%s B mb_skip_run = %d, want frame-coded pair skip_run 2", tt.name, skipRun)
	}
}

func assertHighFrameMBAFFImplicitWeightedPartitionedBFixtureSyntax(t *testing.T, data []byte, tt highFrameMBAFFPartitionedBCase) {
	t.Helper()
	sum := md5.Sum(data)
	if got := hex.EncodeToString(sum[:]); got != tt.bitstreamMD5 {
		t.Fatalf("%s frame-MBAFF implicit weighted partitioned-B bitstream md5 = %s, want %s", tt.name, got, tt.bitstreamMD5)
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
				pps.WeightedBipredIDC != 2 || pps.RefCount != [2]uint32{1, 1} ||
				pps.DeblockingFilterParametersPresent == 0 {
				t.Fatalf("PPS CABAC/8x8/weights/refs/deblock = %d/%d/%d/%d/%v/%d, want CAVLC/no-8x8/implicit-weighted-B refs=1/1 deblock params",
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
					sh.DirectSpatialMVPred != 1 || sh.DeblockingFilter != tt.deblockMode ||
					sh.PPS.WeightedBipredIDC != 2 || sh.PredWeightTable.UseWeight != 0 ||
					sh.PredWeightTable.UseWeightChroma != 0 {
					t.Fatalf("B ref/lists/refs/direct/deblock/weights = %d/%d/%v/%d/%d/%d/%d/%d, want non-ref implicit-weighted B refs=1/1 spatial-direct-header deblock=%d no serialized weights",
						nal.RefIDC, sh.ListCount, sh.RefCount, sh.DirectSpatialMVPred, sh.DeblockingFilter,
						sh.PPS.WeightedBipredIDC, sh.PredWeightTable.UseWeight, sh.PredWeightTable.UseWeightChroma, tt.deblockMode)
				}
				bNAL = nal
			}
			gotTypes = append(gotTypes, sh.SliceTypeNoS)
		default:
			t.Fatalf("unexpected NAL type %d in %s frame-MBAFF implicit weighted partitioned-B fixture", nal.Type, tt.name)
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
