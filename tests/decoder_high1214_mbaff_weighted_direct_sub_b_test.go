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

type highFrameMBAFFWeightedDirectSubBSuite struct {
	cases        func() []highFrameMBAFFDirectSubBCase
	fixture      func(highFrameMBAFFDirectSubBCase) []byte
	assertSyntax func(*testing.T, []byte, highFrameMBAFFDirectSubBCase)
}

func TestHigh1214FrameMBAFFWeightedDirectSubBFixtureSyntax(t *testing.T) {
	for _, suite := range highFrameMBAFFWeightedDirectSubBSuites() {
		for _, tt := range suite.cases() {
			t.Run(tt.name, func(t *testing.T) {
				data := suite.fixture(tt)
				suite.assertSyntax(t, data, tt)
			})
		}
	}
}

func TestDecodeAnnexBHigh1214FrameMBAFFWeightedDirectSubBFrames(t *testing.T) {
	for _, suite := range highFrameMBAFFWeightedDirectSubBSuites() {
		for _, tt := range suite.cases() {
			t.Run(tt.name, func(t *testing.T) {
				data := suite.fixture(tt)
				suite.assertSyntax(t, data, tt)

				frames, err := NewDecoder().DecodeAnnexBFrames(data)
				if err != nil {
					t.Fatalf("DecodeAnnexBFrames: %v", err)
				}
				assertHighFrameMBAFFDirectSubBFrames(t, frames, tt)
			})
		}
	}
}

func TestDecodeAVCHigh1214FrameMBAFFWeightedDirectSubBFrames(t *testing.T) {
	for _, suite := range highFrameMBAFFWeightedDirectSubBSuites() {
		for _, tt := range suite.cases() {
			t.Run(tt.name, func(t *testing.T) {
				data := suite.fixture(tt)
				suite.assertSyntax(t, data, tt)

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
}

func TestDecodeAVCCHigh1214FrameMBAFFWeightedDirectSubBFrames(t *testing.T) {
	for _, suite := range highFrameMBAFFWeightedDirectSubBSuites() {
		for _, tt := range suite.cases() {
			t.Run(tt.name, func(t *testing.T) {
				data := suite.fixture(tt)
				suite.assertSyntax(t, data, tt)

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
}

func TestDecodeConfiguredAVCHigh1214FrameMBAFFWeightedDirectSubBFramesAcrossSamplesFlush(t *testing.T) {
	for _, suite := range highFrameMBAFFWeightedDirectSubBSuites() {
		for _, tt := range suite.cases() {
			t.Run(tt.name, func(t *testing.T) {
				data := suite.fixture(tt)
				suite.assertSyntax(t, data, tt)
				assertDecodeConfiguredAVCHighFrameMBAFFWeightedDirectSubBFrames(t, data, tt)
			})
		}
	}
}

func TestFFmpegRawVideoFrameMD5OracleHigh1214FrameMBAFFWeightedDirectSubB(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}

	for _, suite := range highFrameMBAFFWeightedDirectSubBSuites() {
		for _, tt := range suite.cases() {
			t.Run(tt.name, func(t *testing.T) {
				data := suite.fixture(tt)
				suite.assertSyntax(t, data, tt)
				assertFFmpegHighFrameMBAFFDirectSubBRawVideoOracle(t, data, tt)
			})
		}
	}
}

func highFrameMBAFFWeightedDirectSubBSuites() []highFrameMBAFFWeightedDirectSubBSuite {
	return []highFrameMBAFFWeightedDirectSubBSuite{
		{
			cases:        highFrameMBAFFImplicitWeightedDirectSubBCases,
			fixture:      highFrameMBAFFImplicitWeightedDirectSubBFixture,
			assertSyntax: assertHighFrameMBAFFImplicitWeightedDirectSubBFixtureSyntax,
		},
		{
			cases:        highFrameMBAFFExplicitWeightedDirectSubBCases,
			fixture:      highFrameMBAFFExplicitWeightedDirectSubBFixture,
			assertSyntax: assertHighFrameMBAFFExplicitWeightedDirectSubBFixtureSyntax,
		},
	}
}

func highFrameMBAFFImplicitWeightedDirectSubBCases() []highFrameMBAFFDirectSubBCase {
	bitstreamMD5 := map[string]string{
		"High12TemporalDirectImplicitWeightedSubB8x8FieldNoDeblock":     "b0574ec3f7c39ccbc0d4c7d2c8b9923f",
		"High12TemporalDirectImplicitWeightedSubB8x8FieldFrameDeblock":  "e07c0a6fb3d0998f9a3b9dd3ce400c5d",
		"High12TemporalDirectImplicitWeightedSubB8x8FieldSliceBoundary": "0c4b66a1809bdcb4905a5eeda1df2e28",
		"High12TemporalDirectImplicitWeightedSubB8x8FrameNoDeblock":     "28adb88495919c281a4a737bc96f475a",
		"High12TemporalDirectImplicitWeightedSubB8x8FrameFrameDeblock":  "aa6372845168a9c1a27d8faf792af8cc",
		"High12TemporalDirectImplicitWeightedSubB8x8FrameSliceBoundary": "5b2e9737f2e8d4cec8a356c302b143d0",
		"High12SpatialDirectImplicitWeightedSubB8x8FieldNoDeblock":      "adf18a08bf179123e515f2472804aa80",
		"High12SpatialDirectImplicitWeightedSubB8x8FieldFrameDeblock":   "2809e5e45bcf395c9029411cb96077f4",
		"High12SpatialDirectImplicitWeightedSubB8x8FieldSliceBoundary":  "558d4e3695769f19cc7500e3ca57dfd9",
		"High12SpatialDirectImplicitWeightedSubB8x8FrameNoDeblock":      "dace8148a9215719cad47e1e31393463",
		"High12SpatialDirectImplicitWeightedSubB8x8FrameFrameDeblock":   "9a5dc4eeb135f7c01041fba754dea12f",
		"High12SpatialDirectImplicitWeightedSubB8x8FrameSliceBoundary":  "8b44ab038c7b6fa64cc1170ef9ce4e1b",
		"High14TemporalDirectImplicitWeightedSubB8x8FieldNoDeblock":     "254833be5107477c4b4609864a1aa224",
		"High14TemporalDirectImplicitWeightedSubB8x8FieldFrameDeblock":  "a68f493bc128c824b143fcd43a08e0ab",
		"High14TemporalDirectImplicitWeightedSubB8x8FieldSliceBoundary": "f44439be8f6a88acfd1ba97bd36df318",
		"High14TemporalDirectImplicitWeightedSubB8x8FrameNoDeblock":     "d60f4f5f9f458014b0bfb1d8c56775df",
		"High14TemporalDirectImplicitWeightedSubB8x8FrameFrameDeblock":  "5dd367253eb8a8860eeb9274381a3d61",
		"High14TemporalDirectImplicitWeightedSubB8x8FrameSliceBoundary": "cb020baea94e2141a61a0d4a465f27e0",
		"High14SpatialDirectImplicitWeightedSubB8x8FieldNoDeblock":      "edf639754bc3edff44fdd1a4fd7284bb",
		"High14SpatialDirectImplicitWeightedSubB8x8FieldFrameDeblock":   "bc9fa343f42abae22661572b4c9af172",
		"High14SpatialDirectImplicitWeightedSubB8x8FieldSliceBoundary":  "6d6fd46c2f02005aa0d59622c619c710",
		"High14SpatialDirectImplicitWeightedSubB8x8FrameNoDeblock":      "63bfcd0c9c4db7dc3a72b9fe5feae8bf",
		"High14SpatialDirectImplicitWeightedSubB8x8FrameFrameDeblock":   "6390b4fbddacbe04f7cf5d53d3aaa005",
		"High14SpatialDirectImplicitWeightedSubB8x8FrameSliceBoundary":  "cf04c93e925a245cede924d81f9be30d",
	}
	return highFrameMBAFFWeightedDirectSubBCases("ImplicitWeighted", bitstreamMD5)
}

func highFrameMBAFFExplicitWeightedDirectSubBCases() []highFrameMBAFFDirectSubBCase {
	bitstreamMD5 := map[string]string{
		"High12TemporalDirectExplicitWeightedSubB8x8FieldNoDeblock":     "6b341355c5017a2799a59442707bec4f",
		"High12TemporalDirectExplicitWeightedSubB8x8FieldFrameDeblock":  "1f6fb5c061a4c230ef976bea5d8297e6",
		"High12TemporalDirectExplicitWeightedSubB8x8FieldSliceBoundary": "093683722c95789328807bd24885f2b9",
		"High12TemporalDirectExplicitWeightedSubB8x8FrameNoDeblock":     "3342da1ac13b362693f6315ef8b812f2",
		"High12TemporalDirectExplicitWeightedSubB8x8FrameFrameDeblock":  "1f5ac3ce4bf4b15cb0f72747b69f09c2",
		"High12TemporalDirectExplicitWeightedSubB8x8FrameSliceBoundary": "3ad8c0e028e12940befb761fb6fc756c",
		"High12SpatialDirectExplicitWeightedSubB8x8FieldNoDeblock":      "72ef0b28deb411c905c12537b6189022",
		"High12SpatialDirectExplicitWeightedSubB8x8FieldFrameDeblock":   "b2ae48b6585b16f656816de35920c4dd",
		"High12SpatialDirectExplicitWeightedSubB8x8FieldSliceBoundary":  "f8da420b9c3807ba75fea73cfeeaae4b",
		"High12SpatialDirectExplicitWeightedSubB8x8FrameNoDeblock":      "f5c1c5aa11c9b444181801be4bb1217c",
		"High12SpatialDirectExplicitWeightedSubB8x8FrameFrameDeblock":   "ea398257621aad947b17f3d461f24861",
		"High12SpatialDirectExplicitWeightedSubB8x8FrameSliceBoundary":  "c8272f4da4c66aaf9b0f4955cd1d9242",
		"High14TemporalDirectExplicitWeightedSubB8x8FieldNoDeblock":     "f11cfde50213a0404b5215d21b0f5065",
		"High14TemporalDirectExplicitWeightedSubB8x8FieldFrameDeblock":  "10371a5dacf10c0045fea99edacd34b8",
		"High14TemporalDirectExplicitWeightedSubB8x8FieldSliceBoundary": "d18c939209be02151178de744ab8c266",
		"High14TemporalDirectExplicitWeightedSubB8x8FrameNoDeblock":     "07f5b15cf5d7f1ee5d0ecdd55009b2fd",
		"High14TemporalDirectExplicitWeightedSubB8x8FrameFrameDeblock":  "fd5cc329a9ad963a7745794ec4443ef7",
		"High14TemporalDirectExplicitWeightedSubB8x8FrameSliceBoundary": "306706772ddfc0a1a3eb1dfa73847489",
		"High14SpatialDirectExplicitWeightedSubB8x8FieldNoDeblock":      "c9a27c8bf05caa802efb630a06c13f16",
		"High14SpatialDirectExplicitWeightedSubB8x8FieldFrameDeblock":   "e9620431654f6382a0586834388e3d43",
		"High14SpatialDirectExplicitWeightedSubB8x8FieldSliceBoundary":  "90fff4f75e54f9ae7f3dd16a261894a0",
		"High14SpatialDirectExplicitWeightedSubB8x8FrameNoDeblock":      "b33a56f63169c4bf6505c3601cb7bc54",
		"High14SpatialDirectExplicitWeightedSubB8x8FrameFrameDeblock":   "9b8bbb08ae5c714666a804becdb8ef9a",
		"High14SpatialDirectExplicitWeightedSubB8x8FrameSliceBoundary":  "d9fe7033a7ded7f9ba52bb27c61b388d",
	}
	return highFrameMBAFFWeightedDirectSubBCases("ExplicitWeighted", bitstreamMD5)
}

func highFrameMBAFFWeightedDirectSubBCases(weighted string, bitstreamMD5 map[string]string) []highFrameMBAFFDirectSubBCase {
	var out []highFrameMBAFFDirectSubBCase
	for _, tt := range highFrameMBAFFDirectSubBCases() {
		tt.name = strings.Replace(tt.name, "SubB8x8", weighted+"SubB8x8", 1)
		tt.bitstreamMD5 = highFrameMBAFFWeightedDirectSubBBitstreamMD5(bitstreamMD5, tt.name)
		out = append(out, tt)
	}
	return out
}

func highFrameMBAFFWeightedDirectSubBBitstreamMD5(bitstreamMD5 map[string]string, name string) string {
	got, ok := bitstreamMD5[name]
	if !ok {
		panic(fmt.Sprintf("missing weighted direct-sub B bitstream md5 for %s", name))
	}
	return got
}

func highFrameMBAFFImplicitWeightedDirectSubBFixture(tt highFrameMBAFFDirectSubBCase) []byte {
	var data []byte
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALSPS), highFrameMBAFFBInterSPSRBSP(tt.bitDepth)))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALPPS), highFrameMBAFFImplicitWeightedBPPSRBSP(tt.bitDepth)))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALIDRSlice), highFrameMBAFFIntraPCMSliceRBSP(tt.bitDepth)))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALSlice), highFrameMBAFFPSkipNoResidualSliceRBSP()))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(h264.NALSlice), highFrameMBAFFDirectSubBSliceRBSP(tt)))
	return data
}

func highFrameMBAFFExplicitWeightedDirectSubBFixture(tt highFrameMBAFFDirectSubBCase) []byte {
	var data []byte
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALSPS), highFrameMBAFFBInterSPSRBSP(tt.bitDepth)))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALPPS), highFrameMBAFFExplicitWeightedBPPSRBSP(tt.bitDepth)))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALIDRSlice), highFrameMBAFFIntraPCMSliceRBSP(tt.bitDepth)))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALSlice), highFrameMBAFFPSkipNoResidualSliceRBSP()))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(h264.NALSlice), highFrameMBAFFExplicitWeightedDirectSubBSliceRBSP(tt)))
	return data
}

func highFrameMBAFFExplicitWeightedDirectSubBSliceRBSP(tt highFrameMBAFFDirectSubBCase) []byte {
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
	writeHighFrameMBAFFExplicitWeightedBPredWeightSyntax(&b)
	b.writeSE(0)
	writeHighCAVLCDeblockSyntax(&b, tt.disableDeblockingFilterIDC)
	highIntra16x16WritePayloadBits(&b, tt.payloadBits)
	return b.rbsp()
}

func assertDecodeConfiguredAVCHighFrameMBAFFWeightedDirectSubBFrames(t *testing.T, data []byte, tt highFrameMBAFFDirectSubBCase) {
	t.Helper()
	for _, nalLengthSize := range []int{2, 3, 4} {
		config, samples := annexBToAVCConfigAndSamples(t, data, nalLengthSize)
		if len(samples) != 3 {
			t.Fatalf("%s nalLengthSize=%d samples = %d, want IDR/P/B", tt.name, nalLengthSize, len(samples))
		}
		dec := NewDecoder()
		if _, err := dec.ConfigureAVCC(config); err != nil {
			t.Fatalf("%s nalLengthSize=%d config: %v", tt.name, nalLengthSize, err)
		}
		var frames []*Frame
		for i, sample := range samples {
			out, err := dec.DecodeConfiguredAVCFrames(sample)
			if err != nil {
				t.Fatalf("%s nalLengthSize=%d sample[%d]: DecodeConfiguredAVCFrames: %v", tt.name, nalLengthSize, i, err)
			}
			frames = append(frames, out...)
		}
		out, err := dec.FlushDelayedFrames()
		if err != nil {
			t.Fatalf("%s nalLengthSize=%d flush: %v", tt.name, nalLengthSize, err)
		}
		frames = append(frames, out...)
		assertHighFrameMBAFFDirectSubBFrames(t, frames, tt)
	}
}

func assertHighFrameMBAFFImplicitWeightedDirectSubBFixtureSyntax(t *testing.T, data []byte, tt highFrameMBAFFDirectSubBCase) {
	t.Helper()
	sum := md5.Sum(data)
	if got := hex.EncodeToString(sum[:]); got != tt.bitstreamMD5 {
		t.Fatalf("%s frame-MBAFF implicit weighted direct-sub B bitstream md5 = %s, want %s", tt.name, got, tt.bitstreamMD5)
	}

	bNAL, sps, pps := parseHighFrameMBAFFImplicitWeightedDirectSubBFixtureSyntax(t, data, tt)
	pair := readHighFrameMBAFFCAVLCDirectSubBPair(t, bNAL, sps, pps, tt)
	assertHighFrameMBAFFDirectSubBPair(t, pair, tt)
}

func assertHighFrameMBAFFExplicitWeightedDirectSubBFixtureSyntax(t *testing.T, data []byte, tt highFrameMBAFFDirectSubBCase) {
	t.Helper()
	sum := md5.Sum(data)
	if got := hex.EncodeToString(sum[:]); got != tt.bitstreamMD5 {
		t.Fatalf("%s frame-MBAFF explicit weighted direct-sub B bitstream md5 = %s, want %s", tt.name, got, tt.bitstreamMD5)
	}

	bNAL, sps, pps := parseHighFrameMBAFFExplicitWeightedBFixtureSyntax(t, data, tt.name, tt.bitDepth, tt.directSpatial, tt.deblockMode)
	pair := readHighFrameMBAFFExplicitWeightedCAVLCDirectSubBPair(t, bNAL, sps, pps, tt)
	assertHighFrameMBAFFDirectSubBPair(t, pair, tt)
}

func parseHighFrameMBAFFImplicitWeightedDirectSubBFixtureSyntax(t *testing.T, data []byte, tt highFrameMBAFFDirectSubBCase) (h264.NALUnit, *h264.SPS, *h264.PPS) {
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
			t.Fatalf("unexpected NAL type %d in %s frame-MBAFF implicit weighted direct-sub B fixture", nal.Type, tt.name)
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

func readHighFrameMBAFFExplicitWeightedCAVLCDirectSubBPair(t *testing.T, nal h264.NALUnit, sps *h264.SPS, pps *h264.PPS, tt highFrameMBAFFDirectSubBCase) highFrameMBAFFCAVLCDirectSubBPair {
	t.Helper()
	if sps == nil || pps == nil {
		t.Fatal("missing SPS/PPS for frame-MBAFF explicit weighted direct-sub B syntax check")
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
	readHighFrameMBAFFExplicitWeightedBPredWeightSyntax(t, &br)
	high10ResidualCAVLCReadRefPicMarking(t, &br, nal)
	if qpDelta := br.readSE(t); qpDelta != 0 {
		t.Fatalf("slice_qp_delta = %d, want 0", qpDelta)
	}
	readHighFrameMBAFFExplicitWeightedBDeblockSyntax(t, &br, pps, tt.disableDeblockingFilterIDC)

	topSkipRun := br.readUE(t)
	fieldFlag := br.readBit(t)
	top := readHighFrameMBAFFCAVLCDirectSubBMacroblock(t, &br, topSkipRun, tt)
	bottomSkipRun := br.readUE(t)
	bottom := readHighFrameMBAFFCAVLCDirectSubBMacroblock(t, &br, bottomSkipRun, tt)
	return highFrameMBAFFCAVLCDirectSubBPair{fieldFlag: fieldFlag, top: top, bottom: bottom}
}

func assertHighFrameMBAFFDirectSubBPair(t *testing.T, pair highFrameMBAFFCAVLCDirectSubBPair, tt highFrameMBAFFDirectSubBCase) {
	t.Helper()
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
