// SPDX-License-Identifier: LGPL-2.1-or-later

package goh264

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

const high10WeightedCAVLCAnnexBHex = `
00000001676e000aa6cb45d80880000003008000000301078913500000000168cf09c80000016588843a0c6000f904614c59c2e39b2a223bc042989bb51681192d026000f17867421a1503a5ffe1c040632380020010ce9950c88e4afc7e7ffb
55e37c848f85d7ff8710131553863f23ccf5787101300b0f391d2940d18bf7d51a863727cdf57aaae30366509eaf4803599923817efdf24018acbab8265fbe0400068060e7c2800c205000ee662442e08ebcbd0d7a00cb928d4fe40e27ffed38
1f1892c469001d79454476a8c6bd3160afb27a93601240dfffb96c72a5e516e2ffff20be003bc6739c01566bc0afa265f03a3801003990280738056f6ae3250662944b4eb1a8400018276e42c20532700992000113a58010bc653b123e13fe94
6986190b60bf57aa5a0db90e0109f57b007c842113140769493d010c86e407289aac13a61b210c0115eaf555b18fc8715eaf402190e840729555b01c7623eae0cd2b8f050001006600010140300031652817ec40693e4d92bd0110dd824c4ad9
02544a7f7d835199e403d69bf7dbd127cc22eaefdaf705da96024688522e69ec0e3ffde3046c91718966ffe400000001419a208c06320abc6d404331292b513f93ec8517c4a9930262038d67d7e7f08000c851b10600c40024a21a2a90a15f7e
5f5c9fe17cd1a54007f1c834ba8cadf8ed12d2f076cdc503061af3c4743053b65bfa7bd02d4c64d075d383e000000001419a41844150c80a602b00b002605af1b9bcddd82fb927a36a42c4bbb36729fa9a9642061d401563ac60f97066abbb80
d56aa1e1cbff5a3f00000001419a608503f04280fc80fc1ebc13f5279bcde5f5b6
`

const high10WeightedCABACAnnexBHex = `
00000001676e000aa6cb45d80880000003008000000301078913500000000168ef09c80000016588843af9229552af31bfeb266114ddd0aed37125f66c5313a4fb0f9574b7b94b4227fba5a186c6c6f413c11b7355aa94de6c316c18b70dd3d4
f0a08f4735d303754a9019d845cdfb26735259eb44afc123d750c790b00474195cb4621e54e77e4695a379b4ae23efe71d30e4975e52946a5efce353c60da98dfc47889e586324b5c42a9fb6e312c06454f6f6d034819c5f458d39feaad42e3f
93653df5f93a89bce4e8c4ff237c89c7c0ab6c6ba9848676f635a132069f0211b163c04b3f2f28ccbbe041efc75c160d0dc7dc98310a844d2efc72de43551b4732d54a362238453f3ca03fbf6d83ef5d70d9cefe3b282396c2531523ce177e5b
5928c53f25cddef366886537d92585c90b99cbfa172f6b750f98285acc80f15843b7c2cc72be0f10242c059eac879d7bebd1b639287d96fd9b7167e0ef248f79f03539a1f6edda43f15a6a33c48f60bfdd020a06bc28deb99ab542063b1f9eaa
b2e8cccd90b23da98704a47edf6d9f8d66d248244862001c64794a48840b41c7ba21ffafb998c9def9673da8f384bb3b6cd4e7774acbe4ed37e8527f00000001419a208c0632255fc111e5e9e7394d756856de383c062f50db67389906a84b6b
7f0d15983d71227eb6a858961087229a3d84b0b7e37b014fca7b5c9fc96051f5de8eede6e818d0d85b0323e66f3e617b75c09e308de6dc18eb75e4cb5bc000000001419a41844150c80a602b00b002622d7fd7a706f30a41f936b31c061f880b
c341e7ed242405d1753365ac6ae84d2d010d521ef00bb5587080f83f00000001419a608503f04280fc80fc8f5f578b63a5fa0b8959d44760
`

const (
	high10WeightedPFrameRawSize = 1536
	high10WeightedPRawVideoMD5  = "c9f7de8ec190db53525801f41b473de9"
)

var high10WeightedPFrameMD5 = []string{
	"4b1f34db2851def469994d3f52eee679",
	"914bd8170a17a4ff2800d632af8b4e0b",
	"968ca595fffbfded0f4fbc1c0840cdde",
	"36e2a95ad8461d4f280bab116f6087e6",
}

type high10WeightedPFixture struct {
	name  string
	hex   string
	cabac int32
}

func TestHigh10WeightedPFixtureSyntax(t *testing.T) {
	for _, tt := range high10WeightedPFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			assertHigh10WeightedPFixtureSyntax(t, decodeHexFixture(t, tt.hex), tt.cabac)
		})
	}
}

func TestDecodeAnnexBHigh10WeightedPFrames(t *testing.T) {
	for _, tt := range high10WeightedPFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := decodeHexFixture(t, tt.hex)
			assertHigh10WeightedPFixtureSyntax(t, data, tt.cabac)
			frames, err := NewDecoder().DecodeAnnexBFrames(data)
			if err != nil {
				t.Fatalf("DecodeAnnexBFrames: %v", err)
			}
			assertHigh10WeightedPFrames(t, frames)
		})
	}
}

func TestDecodeAVCHigh10WeightedPFrames(t *testing.T) {
	for _, tt := range high10WeightedPFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := decodeHexFixture(t, tt.hex)
			assertHigh10WeightedPFixtureSyntax(t, data, tt.cabac)
			for _, nalLengthSize := range []int{2, 3, 4} {
				frames, err := NewDecoder().DecodeAVCFrames(annexBToAVC(t, data, nalLengthSize), nalLengthSize)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: DecodeAVCFrames: %v", nalLengthSize, err)
				}
				assertHigh10WeightedPFrames(t, frames)
			}
		})
	}
}

func TestDecodeAVCWithConfigurationRecordHigh10WeightedPFrames(t *testing.T) {
	for _, tt := range high10WeightedPFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := decodeHexFixture(t, tt.hex)
			assertHigh10WeightedPFixtureSyntax(t, data, tt.cabac)
			for _, nalLengthSize := range []int{2, 3, 4} {
				config, packet := annexBToAVCConfigAndPacket(t, data, nalLengthSize)
				frames, err := NewDecoder().DecodeAVCFramesWithConfigurationRecord(config, packet)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: DecodeAVCFramesWithConfigurationRecord: %v", nalLengthSize, err)
				}
				assertHigh10WeightedPFrames(t, frames)
			}
		})
	}
}

func TestDecodeConfiguredAVCAcrossSamplesHigh10WeightedPFrames(t *testing.T) {
	for _, tt := range high10WeightedPFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := decodeHexFixture(t, tt.hex)
			assertHigh10WeightedPFixtureSyntax(t, data, tt.cabac)
			for _, nalLengthSize := range []int{2, 3, 4} {
				config, samples := annexBToAVCConfigAndSamples(t, data, nalLengthSize)
				if len(samples) != len(high10WeightedPFrameMD5) {
					t.Fatalf("nalLengthSize=%d: samples = %d, want %d", nalLengthSize, len(samples), len(high10WeightedPFrameMD5))
				}

				dec := NewDecoder()
				if _, err := dec.ParseAVCDecoderConfigurationRecord(config); err != nil {
					t.Fatalf("nalLengthSize=%d: config: %v", nalLengthSize, err)
				}
				var frames []*Frame
				for i, sample := range samples {
					frame, err := dec.DecodeConfiguredAVC(sample)
					if err != nil {
						t.Fatalf("nalLengthSize=%d sample[%d]: DecodeConfiguredAVC: %v", nalLengthSize, i, err)
					}
					frames = append(frames, frame)
				}
				assertHigh10WeightedPFrames(t, frames)
			}
		})
	}
}

func TestFFmpegRawVideoMD5OracleHigh10WeightedP(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}

	for _, tt := range high10WeightedPFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := decodeHexFixture(t, tt.hex)
			assertHigh10WeightedPFixtureSyntax(t, data, tt.cabac)
			path := writeTempH264(t, data)
			framemd5 := exec.Command("ffmpeg",
				"-v", "error",
				"-f", "h264",
				"-i", path,
				"-an", "-sn", "-dn",
				"-pix_fmt", "yuv420p10le",
				"-f", "framemd5",
				"-",
			)
			framemd5Out, err := framemd5.Output()
			if err != nil {
				t.Fatalf("ffmpeg framemd5: %v", err)
			}
			for i, want := range high10WeightedPFrameMD5 {
				line := []byte(fmt.Sprintf("0, %10d, %10d,        1, %8d, %s", i, i, high10WeightedPFrameRawSize, want))
				if !bytes.Contains(framemd5Out, line) {
					t.Fatalf("frame[%d] missing %q in framemd5:\n%s", i, line, framemd5Out)
				}
			}

			cmd := exec.Command("ffmpeg",
				"-v", "error",
				"-f", "h264",
				"-i", path,
				"-an", "-sn", "-dn",
				"-pix_fmt", "yuv420p10le",
				"-f", "rawvideo",
				"-",
			)
			raw, err := cmd.Output()
			if err != nil {
				t.Fatalf("ffmpeg rawvideo: %v", err)
			}
			wantSize := len(high10WeightedPFrameMD5) * high10WeightedPFrameRawSize
			if len(raw) != wantSize {
				t.Fatalf("rawvideo size = %d, want %d", len(raw), wantSize)
			}
			sum := md5.Sum(raw)
			if got := hex.EncodeToString(sum[:]); got != high10WeightedPRawVideoMD5 {
				t.Fatalf("rawvideo md5 = %s, want %s", got, high10WeightedPRawVideoMD5)
			}
			for i, want := range high10WeightedPFrameMD5 {
				frame := raw[i*high10WeightedPFrameRawSize : (i+1)*high10WeightedPFrameRawSize]
				sum := md5.Sum(frame)
				if got := hex.EncodeToString(sum[:]); got != want {
					t.Fatalf("frame[%d] md5 = %s, want %s", i, got, want)
				}
			}
		})
	}
}

func high10WeightedPFixtures() []high10WeightedPFixture {
	return []high10WeightedPFixture{
		{name: "cavlc", hex: high10WeightedCAVLCAnnexBHex, cabac: 0},
		{name: "cabac", hex: high10WeightedCABACAnnexBHex, cabac: 1},
	}
}

func assertHigh10WeightedPFrames(t *testing.T, frames []*Frame) {
	t.Helper()
	assertHigh10FrameMD5Strings(t, frames, high10WeightedPFrameMD5)
	for i, frame := range frames {
		if frame.Width != 32 || frame.Height != 16 {
			t.Fatalf("frame[%d] size = %dx%d, want 32x16", i, frame.Width, frame.Height)
		}
		rawSize, err := frame.RawYUVSize()
		if err != nil {
			t.Fatalf("frame[%d] RawYUVSize: %v", i, err)
		}
		if rawSize != high10WeightedPFrameRawSize {
			t.Fatalf("frame[%d] raw yuv size = %d, want %d", i, rawSize, high10WeightedPFrameRawSize)
		}
	}
}

func assertHigh10WeightedPFixtureSyntax(t *testing.T, data []byte, cabac int32) {
	t.Helper()
	nals, err := h264.SplitAnnexB(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(nals) != 6 {
		t.Fatalf("NAL count = %d, want stripped SPS/PPS/IDR/P/P/P", len(nals))
	}
	wantNALs := []h264.NALUnitType{
		h264.NALSPS,
		h264.NALPPS,
		h264.NALIDRSlice,
		h264.NALSlice,
		h264.NALSlice,
		h264.NALSlice,
	}
	for i, want := range wantNALs {
		if nals[i].Type != want {
			t.Fatalf("NAL[%d] type = %d, want %d", i, nals[i].Type, want)
		}
	}

	var spsList [32]*h264.SPS
	var ppsList [256]*h264.PPS
	var gotSlices []int32
	var pWeights []h264.PredWeightTable
	for _, nal := range nals {
		switch nal.Type {
		case h264.NALSPS:
			sps, err := h264.DecodeSPS(nal.RBSP)
			if err != nil {
				t.Fatal(err)
			}
			if sps.ProfileIDC != 110 || sps.Width != 32 || sps.Height != 16 ||
				sps.ChromaFormatIDC != 1 || sps.BitDepthLuma != 10 || sps.BitDepthChroma != 10 ||
				sps.FrameMBSOnlyFlag != 1 || sps.MBAFF != 0 || sps.RefFrameCount != 1 {
				t.Fatalf("SPS profile/format = %d %dx%d chroma %d depth %d/%d frame_only %d/%d refs %d",
					sps.ProfileIDC, sps.Width, sps.Height, sps.ChromaFormatIDC, sps.BitDepthLuma, sps.BitDepthChroma,
					sps.FrameMBSOnlyFlag, sps.MBAFF, sps.RefFrameCount)
			}
			spsList[sps.SPSID] = sps
		case h264.NALPPS:
			pps, err := h264.DecodePPS(nal.RBSP, &spsList)
			if err != nil {
				t.Fatal(err)
			}
			if pps.CABAC != cabac || pps.Transform8x8Mode != 0 || pps.WeightedPred != 1 ||
				pps.WeightedBipredIDC != 0 || pps.RefCount[0] != 1 || pps.RefCount[1] != 1 {
				t.Fatalf("PPS cabac/8x8/weights/refs = %d/%d/%d/%d/%d/%d, want cabac=%d no-8x8 weighted ref=1",
					pps.CABAC, pps.Transform8x8Mode, pps.WeightedPred, pps.WeightedBipredIDC, pps.RefCount[0], pps.RefCount[1], cabac)
			}
			ppsList[pps.PPSID] = pps
		case h264.NALIDRSlice, h264.NALSlice:
			sh, err := h264.ParseSliceHeader(nal, &ppsList)
			if err != nil {
				t.Fatal(err)
			}
			if sh.PictureStructure != h264.PictureFrame || sh.DeblockingFilter != 0 {
				t.Fatalf("slice picture/deblock = %d/%d, want frame/disabled", sh.PictureStructure, sh.DeblockingFilter)
			}
			if sh.SliceTypeNoS == h264.PictureTypeP {
				if sh.ListCount != 1 || sh.RefCount[0] != 1 {
					t.Fatalf("P slice lists/ref0 = %d/%d, want one L0 ref", sh.ListCount, sh.RefCount[0])
				}
				pWeights = append(pWeights, sh.PredWeightTable)
			}
			gotSlices = append(gotSlices, sh.SliceTypeNoS)
		default:
			t.Fatalf("unexpected NAL type %d in stripped fixture", nal.Type)
		}
	}
	wantSlices := []int32{h264.PictureTypeI, h264.PictureTypeP, h264.PictureTypeP, h264.PictureTypeP}
	if len(gotSlices) != len(wantSlices) {
		t.Fatalf("slice count = %d, want %d", len(gotSlices), len(wantSlices))
	}
	for i, want := range wantSlices {
		if gotSlices[i] != want {
			t.Fatalf("slice[%d] type = %d, want %d", i, gotSlices[i], want)
		}
	}
	assertHigh10WeightedPPredWeight(t, "P1", pWeights[0], 7, 0, [2]int32{99, 1}, 0, [2]int32{1, 0}, [2]int32{1, 0})
	assertHigh10WeightedPPredWeight(t, "P2", pWeights[1], 5, 7, [2]int32{21, 6}, 1, [2]int32{83, 43}, [2]int32{88, 38})
	assertHigh10WeightedPPredWeight(t, "P3", pWeights[2], 7, 1, [2]int32{63, 8}, 1, [2]int32{1, 63}, [2]int32{1, 63})
}

func assertHigh10WeightedPPredWeight(t *testing.T, label string, pwt h264.PredWeightTable, lumaDenom uint32, chromaDenom uint32, luma [2]int32, useChroma int32, cb [2]int32, cr [2]int32) {
	t.Helper()
	if pwt.UseWeight != 1 || pwt.UseWeightChroma != useChroma ||
		pwt.LumaLog2WeightDenom != lumaDenom || pwt.ChromaLog2WeightDenom != chromaDenom {
		t.Fatalf("%s weight flags/denom = use %d/%d denom %d/%d, want 1/%d %d/%d",
			label, pwt.UseWeight, pwt.UseWeightChroma, pwt.LumaLog2WeightDenom, pwt.ChromaLog2WeightDenom,
			useChroma, lumaDenom, chromaDenom)
	}
	if pwt.LumaWeight[0][0] != luma || pwt.ChromaWeight[0][0][0] != cb || pwt.ChromaWeight[0][0][1] != cr {
		t.Fatalf("%s weights = luma %+v chroma %+v, want luma %+v cb %+v cr %+v",
			label, pwt.LumaWeight[0][0], pwt.ChromaWeight[0][0], luma, cb, cr)
	}
}
