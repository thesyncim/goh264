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

type highFrameMBAFFWeightedResidualPCase struct {
	name                       string
	bitDepth                   int
	fieldFlag                  uint32
	mbType                     uint32
	cbp                        uint32
	payloadBits                string
	residualTailBits           string
	disableDeblockingFilterIDC uint32
	deblockMode                int32
	bitstreamMD5               string
	refFrameMD5                string
	pFrameMD5                  string
	rawVideoMD5                string
}

func TestHigh1214FrameMBAFFWeightedResidualPFixtureSyntax(t *testing.T) {
	for _, tt := range highFrameMBAFFWeightedResidualPCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFWeightedResidualPFixture(tt)
			assertHighFrameMBAFFWeightedResidualPFixtureSyntax(t, data, tt)
		})
	}
}

func TestDecodeAnnexBHigh1214FrameMBAFFWeightedResidualPFrames(t *testing.T) {
	for _, tt := range highFrameMBAFFWeightedResidualPCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFWeightedResidualPFixture(tt)
			assertHighFrameMBAFFWeightedResidualPFixtureSyntax(t, data, tt)

			frames, err := NewDecoder().DecodeAnnexBFrames(data)
			if err != nil {
				t.Fatalf("DecodeAnnexBFrames: %v", err)
			}
			assertHighFrameMBAFFWeightedResidualPFrames(t, frames, tt)
		})
	}
}

func TestDecodeAVCHigh1214FrameMBAFFWeightedResidualPFrames(t *testing.T) {
	for _, tt := range highFrameMBAFFWeightedResidualPCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFWeightedResidualPFixture(tt)
			assertHighFrameMBAFFWeightedResidualPFixtureSyntax(t, data, tt)

			for _, nalLengthSize := range []int{2, 3, 4} {
				frames, err := NewDecoder().DecodeAVCFrames(annexBToAVC(t, data, nalLengthSize), nalLengthSize)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: DecodeAVCFrames: %v", nalLengthSize, err)
				}
				assertHighFrameMBAFFWeightedResidualPFrames(t, frames, tt)
			}
		})
	}
}

func TestDecodeAVCCHigh1214FrameMBAFFWeightedResidualPFrames(t *testing.T) {
	for _, tt := range highFrameMBAFFWeightedResidualPCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFWeightedResidualPFixture(tt)
			assertHighFrameMBAFFWeightedResidualPFixtureSyntax(t, data, tt)

			for _, nalLengthSize := range []int{2, 3, 4} {
				config, packet := annexBToAVCConfigAndPacket(t, data, nalLengthSize)
				frames, err := NewDecoder().DecodeAVCCFrames(config, packet)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: DecodeAVCCFrames: %v", nalLengthSize, err)
				}
				assertHighFrameMBAFFWeightedResidualPFrames(t, frames, tt)
			}
		})
	}
}

func TestDecodeConfiguredAVCHigh1214FrameMBAFFWeightedResidualPFramesAcrossSamples(t *testing.T) {
	for _, tt := range highFrameMBAFFWeightedResidualPCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFWeightedResidualPFixture(tt)
			assertHighFrameMBAFFWeightedResidualPFixtureSyntax(t, data, tt)

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
				assertHighFrameMBAFFWeightedResidualPFrames(t, frames, tt)
			}
		})
	}
}

func TestFFmpegRawVideoMD5OracleHigh1214FrameMBAFFWeightedResidualP(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}

	for _, tt := range highFrameMBAFFWeightedResidualPCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFWeightedResidualPFixture(tt)
			assertHighFrameMBAFFWeightedResidualPFixtureSyntax(t, data, tt)
			assertFFmpegHighFrameMBAFFWeightedPRawVideoOracle(t, data, highFrameMBAFFWeightedResidualPAsWeightedPCase(tt))
		})
	}
}

func highFrameMBAFFWeightedResidualPCases() []highFrameMBAFFWeightedResidualPCase {
	bitstreamMD5 := highFrameMBAFFWeightedResidualPBitstreamMD5()
	var out []highFrameMBAFFWeightedResidualPCase
	for _, bitDepth := range []int{12, 14} {
		for _, coding := range []struct {
			name      string
			fieldFlag uint32
			shapes    []struct {
				name string
				mb   uint32
				luma string
				both string
			}
		}{
			{name: "Field", fieldFlag: 1, shapes: []struct {
				name string
				mb   uint32
				luma string
				both string
			}{
				{name: "P16x16", mb: 0, luma: highFrameMBAFFP16x16LumaResidualPayloadBits, both: highFrameMBAFFP16x16LumaChromaResidualPayloadBits},
				{name: "P16x8", mb: 1, luma: highFrameMBAFFP16x8LumaResidualPayloadBits, both: highFrameMBAFFP16x8LumaChromaResidualPayloadBits},
				{name: "P8x16", mb: 2, luma: highFrameMBAFFP8x16LumaResidualPayloadBits, both: highFrameMBAFFP8x16LumaChromaResidualPayloadBits},
				{name: "P8x8", mb: 3, luma: highFrameMBAFFP8x8LumaResidualPayloadBits, both: highFrameMBAFFP8x8LumaChromaResidualPayloadBits},
			}},
			{name: "Frame", fieldFlag: 0, shapes: []struct {
				name string
				mb   uint32
				luma string
				both string
			}{
				{name: "P16x16", mb: 0, luma: highFrameMBAFFFrameP16x16LumaResidualPayloadBits, both: highFrameMBAFFFrameP16x16LumaChromaResidualPayloadBits},
				{name: "P16x8", mb: 1, luma: highFrameMBAFFFrameP16x8LumaResidualPayloadBits, both: highFrameMBAFFFrameP16x8LumaChromaResidualPayloadBits},
				{name: "P8x16", mb: 2, luma: highFrameMBAFFFrameP8x16LumaResidualPayloadBits, both: highFrameMBAFFFrameP8x16LumaChromaResidualPayloadBits},
				{name: "P8x8", mb: 3, luma: highFrameMBAFFFrameP8x8LumaResidualPayloadBits, both: highFrameMBAFFFrameP8x8LumaChromaResidualPayloadBits},
			}},
		} {
			for _, shape := range coding.shapes {
				for _, residual := range []struct {
					name string
					cbp  uint32
					bits string
					tail string
				}{
					{name: "LumaResidual", cbp: 1, bits: shape.luma, tail: highFrameMBAFFP16x16LumaResidualTailBits},
					{name: "LumaChromaResidual", cbp: 33, bits: shape.both, tail: highFrameMBAFFP16x16LumaChromaResidualTailBits},
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
						name := fmt.Sprintf("High%d%s%sWeighted%s%s", bitDepth, coding.name, shape.name, residual.name, deblock.name)
						hashes := highFrameMBAFFWeightedResidualPFrameHashes(bitDepth, coding.fieldFlag, residual.cbp)
						out = append(out, highFrameMBAFFWeightedResidualPCase{
							name:                       name,
							bitDepth:                   bitDepth,
							fieldFlag:                  coding.fieldFlag,
							mbType:                     shape.mb,
							cbp:                        residual.cbp,
							payloadBits:                residual.bits,
							residualTailBits:           residual.tail,
							disableDeblockingFilterIDC: deblock.disableID,
							deblockMode:                deblock.mode,
							bitstreamMD5:               lookupHighFrameMBAFFWeightedResidualPBitstreamMD5(bitstreamMD5, name),
							refFrameMD5:                hashes.refFrameMD5,
							pFrameMD5:                  hashes.pFrameMD5,
							rawVideoMD5:                hashes.rawVideoMD5,
						})
					}
				}
			}
		}
	}
	if len(out) != 96 {
		panic(fmt.Sprintf("High12/High14 frame-MBAFF weighted residual P cases = %d, want 96", len(out)))
	}
	return out
}

type highFrameMBAFFWeightedResidualPFrameHashSet struct {
	refFrameMD5 string
	pFrameMD5   string
	rawVideoMD5 string
}

func highFrameMBAFFWeightedResidualPFrameHashes(bitDepth int, fieldFlag uint32, cbp uint32) highFrameMBAFFWeightedResidualPFrameHashSet {
	switch {
	case bitDepth == 12 && fieldFlag == 1 && cbp == 1:
		return highFrameMBAFFWeightedResidualPFrameHashSet{
			refFrameMD5: high12FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:   "7d61bd9cf7ca24b27cb824e3f541acc5",
			rawVideoMD5: "4c52f91347e60457fd8607e5a3fe9afa",
		}
	case bitDepth == 12 && fieldFlag == 1 && cbp == 33:
		return highFrameMBAFFWeightedResidualPFrameHashSet{
			refFrameMD5: high12FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:   "f21403c19d09cf491a6484719df5d90a",
			rawVideoMD5: "fd623f314e8020b08b4fe4465a1ee96c",
		}
	case bitDepth == 12 && fieldFlag == 0 && cbp == 1:
		return highFrameMBAFFWeightedResidualPFrameHashSet{
			refFrameMD5: high12FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:   "7d4bbf6c1689dfb4a290cd8c22b53f04",
			rawVideoMD5: "073f76919f9de582d205e30d6bb815bf",
		}
	case bitDepth == 12 && fieldFlag == 0 && cbp == 33:
		return highFrameMBAFFWeightedResidualPFrameHashSet{
			refFrameMD5: high12FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:   "8a8c1321edc65eba35b57e589e3c6a2c",
			rawVideoMD5: "55c83d2f0903d95c95fe0bbc40ad91d1",
		}
	case bitDepth == 14 && fieldFlag == 1 && cbp == 1:
		return highFrameMBAFFWeightedResidualPFrameHashSet{
			refFrameMD5: high14FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:   "ed8802c7dc94e845d92c50c3a79aa307",
			rawVideoMD5: "a175688a6fbeeeff8980b4003eac16f9",
		}
	case bitDepth == 14 && fieldFlag == 1 && cbp == 33:
		return highFrameMBAFFWeightedResidualPFrameHashSet{
			refFrameMD5: high14FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:   "97f893cb4e5d14bf1c877742e3b95c24",
			rawVideoMD5: "7bedff758a51aaf7dc0da3d923dee9c9",
		}
	case bitDepth == 14 && fieldFlag == 0 && cbp == 1:
		return highFrameMBAFFWeightedResidualPFrameHashSet{
			refFrameMD5: high14FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:   "39cadcf250bc60f51d8c99ef9f81209d",
			rawVideoMD5: "cae3cb823993cf3b1731571528e79d2b",
		}
	case bitDepth == 14 && fieldFlag == 0 && cbp == 33:
		return highFrameMBAFFWeightedResidualPFrameHashSet{
			refFrameMD5: high14FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:   "1f2ccf5b2d83b50203baf83a7c2ede88",
			rawVideoMD5: "65a33f6d6d8ebed3e58263c6b945fd3a",
		}
	default:
		panic(fmt.Sprintf("missing frame hashes for bitDepth=%d fieldFlag=%d cbp=%d", bitDepth, fieldFlag, cbp))
	}
}

func lookupHighFrameMBAFFWeightedResidualPBitstreamMD5(bitstreamMD5 map[string]string, name string) string {
	got, ok := bitstreamMD5[name]
	if !ok {
		panic(fmt.Sprintf("missing bitstream md5 for %s", name))
	}
	return got
}

func highFrameMBAFFWeightedResidualPBitstreamMD5() map[string]string {
	return map[string]string{
		"High12FieldP16x16WeightedLumaResidualNoDeblock":           "ee2c101bbc3e65f4ec6035a99ef2833d",
		"High12FieldP16x16WeightedLumaResidualFrameDeblock":        "84c85fb6fd7f601f7073cb176c2add1b",
		"High12FieldP16x16WeightedLumaResidualSliceBoundary":       "380bf4984ac80387163174e834aa6e83",
		"High12FieldP16x16WeightedLumaChromaResidualNoDeblock":     "1b14781e9cdda0babaf058e67b0f1a24",
		"High12FieldP16x16WeightedLumaChromaResidualFrameDeblock":  "064a520694645d924c9bc51044e054b1",
		"High12FieldP16x16WeightedLumaChromaResidualSliceBoundary": "e4b96be2a4ad2691302c7c7376115317",
		"High12FieldP16x8WeightedLumaResidualNoDeblock":            "5c7f9f8c054bd21324cd151c7a5fe913",
		"High12FieldP16x8WeightedLumaResidualFrameDeblock":         "e638cbc57d8571dd78c41b47c85e4020",
		"High12FieldP16x8WeightedLumaResidualSliceBoundary":        "97d0ab77c7da25638f8cd3b536613963",
		"High12FieldP16x8WeightedLumaChromaResidualNoDeblock":      "a9e63b1c5012ca80c9f2ff804c7a0726",
		"High12FieldP16x8WeightedLumaChromaResidualFrameDeblock":   "7a7cc7ec79ba2159f73311ea50cb726f",
		"High12FieldP16x8WeightedLumaChromaResidualSliceBoundary":  "bc895fd0f5990b1a336d415cb7806fd3",
		"High12FieldP8x16WeightedLumaResidualNoDeblock":            "bd8f0f096162bc7d613ebec6f53e1bf3",
		"High12FieldP8x16WeightedLumaResidualFrameDeblock":         "ca2b84cea6d5d397570b859e5cb996bf",
		"High12FieldP8x16WeightedLumaResidualSliceBoundary":        "8f311bf0841840054df6ea15cfb12a99",
		"High12FieldP8x16WeightedLumaChromaResidualNoDeblock":      "60cac23209ad55848a7cba595bdd91d9",
		"High12FieldP8x16WeightedLumaChromaResidualFrameDeblock":   "388a100b0941158668dadbe420918a68",
		"High12FieldP8x16WeightedLumaChromaResidualSliceBoundary":  "73e4cd92fe52f3316eeb5141b1ca5c5f",
		"High12FieldP8x8WeightedLumaResidualNoDeblock":             "2a746aaef9af222ace894344f0082583",
		"High12FieldP8x8WeightedLumaResidualFrameDeblock":          "f5b3ec1998711a0cff2e533772dd9df5",
		"High12FieldP8x8WeightedLumaResidualSliceBoundary":         "a63afcd405187fa9dadbf90dbeeee2c3",
		"High12FieldP8x8WeightedLumaChromaResidualNoDeblock":       "65e4ecac08ff4adcce05529fdd8a6a59",
		"High12FieldP8x8WeightedLumaChromaResidualFrameDeblock":    "bd43977838a3ab75fd7969f9632ca3e3",
		"High12FieldP8x8WeightedLumaChromaResidualSliceBoundary":   "a4c1c2e69d477920457bee6fdd6e6c55",
		"High12FrameP16x16WeightedLumaResidualNoDeblock":           "e81e59c74946b30260f37a22b68cbcf0",
		"High12FrameP16x16WeightedLumaResidualFrameDeblock":        "feff02d4ed73a8fb44d1e27042ef81c1",
		"High12FrameP16x16WeightedLumaResidualSliceBoundary":       "6ad378d404c0e8f1ab991f7e7867d520",
		"High12FrameP16x16WeightedLumaChromaResidualNoDeblock":     "17005479b588a9976acc6927450c50a7",
		"High12FrameP16x16WeightedLumaChromaResidualFrameDeblock":  "5cb1b0580aed93d8838c69c26f87c529",
		"High12FrameP16x16WeightedLumaChromaResidualSliceBoundary": "5dc0355dbe20ea19614d019b279983c9",
		"High12FrameP16x8WeightedLumaResidualNoDeblock":            "0fabbc28a20d08ffd2821b90f2e93c02",
		"High12FrameP16x8WeightedLumaResidualFrameDeblock":         "15767382dda47f1cc9b55d35c5ebf25f",
		"High12FrameP16x8WeightedLumaResidualSliceBoundary":        "ac38c036c36e14e8c3a34255b5c70874",
		"High12FrameP16x8WeightedLumaChromaResidualNoDeblock":      "5a4efba849547746c53a461c05cb6e1c",
		"High12FrameP16x8WeightedLumaChromaResidualFrameDeblock":   "f3de28c4028bc28461364fb520acc630",
		"High12FrameP16x8WeightedLumaChromaResidualSliceBoundary":  "d2777bdb037fca0689090cf4a6448ecb",
		"High12FrameP8x16WeightedLumaResidualNoDeblock":            "d0075b465ba695d7d04b527241b7ad4c",
		"High12FrameP8x16WeightedLumaResidualFrameDeblock":         "fc3b5a78b64eedc5f9598d05b00f4dd4",
		"High12FrameP8x16WeightedLumaResidualSliceBoundary":        "cbf0d86580fd314eefb0a9e026ae8aad",
		"High12FrameP8x16WeightedLumaChromaResidualNoDeblock":      "48e11f9bc21bd02a40cef59fe622a369",
		"High12FrameP8x16WeightedLumaChromaResidualFrameDeblock":   "808f81687bf860696ba3ca8f2d8a1eff",
		"High12FrameP8x16WeightedLumaChromaResidualSliceBoundary":  "ede177fbae12f787c97f28fa7bee7c64",
		"High12FrameP8x8WeightedLumaResidualNoDeblock":             "56ffc6e08c3a89b746c812740d964e2c",
		"High12FrameP8x8WeightedLumaResidualFrameDeblock":          "1ad764d9524665ec68bb6922e25a7a29",
		"High12FrameP8x8WeightedLumaResidualSliceBoundary":         "6ec54c01a927751fce30bd3460f4f03f",
		"High12FrameP8x8WeightedLumaChromaResidualNoDeblock":       "729a8ccb90f242843c8f3d8e6a6bc0bb",
		"High12FrameP8x8WeightedLumaChromaResidualFrameDeblock":    "f435d67b255b2929675b8ecfbc1665ab",
		"High12FrameP8x8WeightedLumaChromaResidualSliceBoundary":   "53bdc311aab4474fdb3ec5567addc39d",
		"High14FieldP16x16WeightedLumaResidualNoDeblock":           "ea9e29c3050426cd14535554bb2bf485",
		"High14FieldP16x16WeightedLumaResidualFrameDeblock":        "ab36ce374dd58ac2cfdf934635315396",
		"High14FieldP16x16WeightedLumaResidualSliceBoundary":       "f95f3fbe1a37dc33d335d0e0861f9e3c",
		"High14FieldP16x16WeightedLumaChromaResidualNoDeblock":     "ce9b7441038b413c1dd242dbd1e0594b",
		"High14FieldP16x16WeightedLumaChromaResidualFrameDeblock":  "2246a26423c1d103704513ae28bfc60d",
		"High14FieldP16x16WeightedLumaChromaResidualSliceBoundary": "bd380ba7b01a650db66faf528b79b674",
		"High14FieldP16x8WeightedLumaResidualNoDeblock":            "ccdf50eaa7144d2c95c2bd7e0afb7e62",
		"High14FieldP16x8WeightedLumaResidualFrameDeblock":         "31d9a66ce0dea18bfb5fb2da5ea68bea",
		"High14FieldP16x8WeightedLumaResidualSliceBoundary":        "cbec7beb952ed7438ba3103131af3b72",
		"High14FieldP16x8WeightedLumaChromaResidualNoDeblock":      "83766aa79aec7314637fc9f820a4f77c",
		"High14FieldP16x8WeightedLumaChromaResidualFrameDeblock":   "23bde0468731070a44ea665f32a3ddfc",
		"High14FieldP16x8WeightedLumaChromaResidualSliceBoundary":  "cc2b713cdb44ddb34cf1dd15aed40e7d",
		"High14FieldP8x16WeightedLumaResidualNoDeblock":            "5ec95207a0d270f68f39172dbb4eafe4",
		"High14FieldP8x16WeightedLumaResidualFrameDeblock":         "ca13d58665aa8b6e6a9670a4d5e32b33",
		"High14FieldP8x16WeightedLumaResidualSliceBoundary":        "911e9adfb40c31635e973a10467a04ec",
		"High14FieldP8x16WeightedLumaChromaResidualNoDeblock":      "864b541026ba53090706994976bbbe0e",
		"High14FieldP8x16WeightedLumaChromaResidualFrameDeblock":   "d5c9360d862a10b2e48141523c3d30b6",
		"High14FieldP8x16WeightedLumaChromaResidualSliceBoundary":  "7ea25289a89b7028a7436c6f90da5f77",
		"High14FieldP8x8WeightedLumaResidualNoDeblock":             "cd491b7ba82b8d8c3243dddb32941db5",
		"High14FieldP8x8WeightedLumaResidualFrameDeblock":          "59a526ae6f94470436938ed605710618",
		"High14FieldP8x8WeightedLumaResidualSliceBoundary":         "d493fee7b55a4c3760411d575e75fa08",
		"High14FieldP8x8WeightedLumaChromaResidualNoDeblock":       "5863b2df95a23ae9f90508321f5e6ad5",
		"High14FieldP8x8WeightedLumaChromaResidualFrameDeblock":    "c807d08f704b3d1ce4e263a88b7ae75e",
		"High14FieldP8x8WeightedLumaChromaResidualSliceBoundary":   "7db1003ef7019f5345e6e55144ac7795",
		"High14FrameP16x16WeightedLumaResidualNoDeblock":           "97d6efddafeb216def1ba9d8c2d27d7b",
		"High14FrameP16x16WeightedLumaResidualFrameDeblock":        "42a18f5352fca3017e68dc95b66438fb",
		"High14FrameP16x16WeightedLumaResidualSliceBoundary":       "01ea7d60de37babb3f8e6ca8a7ba2981",
		"High14FrameP16x16WeightedLumaChromaResidualNoDeblock":     "090cbf18f127552c4384263d33276e73",
		"High14FrameP16x16WeightedLumaChromaResidualFrameDeblock":  "8869d7f9af7026a774de2214cf1eef1c",
		"High14FrameP16x16WeightedLumaChromaResidualSliceBoundary": "caac654b20fc710fd63f22a5ae3dbd71",
		"High14FrameP16x8WeightedLumaResidualNoDeblock":            "6fffb5510436d5b0b75f733ec213dfb1",
		"High14FrameP16x8WeightedLumaResidualFrameDeblock":         "d72fb78b32831c1fa7c0e0736ae18ec9",
		"High14FrameP16x8WeightedLumaResidualSliceBoundary":        "4bdbb30c3f83420ed6897c5821f2b291",
		"High14FrameP16x8WeightedLumaChromaResidualNoDeblock":      "05a0e1f422bbaf4de25efa18f2d5ea5a",
		"High14FrameP16x8WeightedLumaChromaResidualFrameDeblock":   "8db5f14ca83ebb8cd4a53bff278e2d08",
		"High14FrameP16x8WeightedLumaChromaResidualSliceBoundary":  "67c70d738f440e4f8164069359348660",
		"High14FrameP8x16WeightedLumaResidualNoDeblock":            "c4ed70993749b3cfc78eacb7b60d890c",
		"High14FrameP8x16WeightedLumaResidualFrameDeblock":         "f20c4c972ae4ae558f74a8130eb55942",
		"High14FrameP8x16WeightedLumaResidualSliceBoundary":        "fe2c091d9ca9735dd984826ac56b40eb",
		"High14FrameP8x16WeightedLumaChromaResidualNoDeblock":      "8f5b674e05079822eea4cd80999445b3",
		"High14FrameP8x16WeightedLumaChromaResidualFrameDeblock":   "9275ca738035673b6145e829bf84228b",
		"High14FrameP8x16WeightedLumaChromaResidualSliceBoundary":  "8d3125105810f2d0522bb5155a9a2789",
		"High14FrameP8x8WeightedLumaResidualNoDeblock":             "e1aa85c5e1b728e8c988ec74b3a7d729",
		"High14FrameP8x8WeightedLumaResidualFrameDeblock":          "3aa44792f11cf2abb1d7dd27dd2835d7",
		"High14FrameP8x8WeightedLumaResidualSliceBoundary":         "9a012e00a514b078b7256f92d32e8daa",
		"High14FrameP8x8WeightedLumaChromaResidualNoDeblock":       "12dc9d3babae3025576239112778c678",
		"High14FrameP8x8WeightedLumaChromaResidualFrameDeblock":    "68f93842412aa956a6c580ab153fe81e",
		"High14FrameP8x8WeightedLumaChromaResidualSliceBoundary":   "ea106f84e9f416ac112514a3b9b811a0",
	}
}

func highFrameMBAFFWeightedResidualPFixture(tt highFrameMBAFFWeightedResidualPCase) []byte {
	return highFrameMBAFFWeightedPFixture(highFrameMBAFFWeightedResidualPAsWeightedPCase(tt))
}

func highFrameMBAFFWeightedResidualPAsWeightedPCase(tt highFrameMBAFFWeightedResidualPCase) highFrameMBAFFWeightedPCase {
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

func assertHighFrameMBAFFWeightedResidualPFixtureSyntax(t *testing.T, data []byte, tt highFrameMBAFFWeightedResidualPCase) {
	t.Helper()
	sum := md5.Sum(data)
	if got := hex.EncodeToString(sum[:]); got != tt.bitstreamMD5 {
		t.Fatalf("%s frame-MBAFF weighted residual-P bitstream md5 = %s, want %s", tt.name, got, tt.bitstreamMD5)
	}

	nals, spsList, ppsList := parseHighFrameMBAFFWeightedPFixtureSyntax(t, data, highFrameMBAFFWeightedResidualPAsWeightedPCase(tt))
	if tt.mbType == 0 {
		pair := readHighFrameMBAFFWeightedCAVLCP16x16ResidualPair(t, nals[1], spsList[0], ppsList[0], tt)
		if pair.fieldFlag != tt.fieldFlag {
			t.Fatalf("%s frame-MBAFF weighted residual-P pair field flag = %d, want %d", tt.name, pair.fieldFlag, tt.fieldFlag)
		}
		for i, mb := range []highFrameMBAFFCAVLCP16x16Macroblock{pair.top, pair.bottom} {
			if mb.skipRun != 0 || mb.mbType != 0 || mb.refIdxFlag != tt.fieldFlag || mb.cbp != tt.cbp {
				t.Fatalf("%s weighted residual-P macroblock[%d] skip/mb_type/ref_idx_flag/cbp = %d/%d/%d/%d (code %d), want P16x16 cbp %d with field flag %d",
					tt.name, i, mb.skipRun, mb.mbType, mb.refIdxFlag, mb.cbp, mb.cbpCode, tt.cbp, tt.fieldFlag)
			}
		}
		return
	}

	pair := readHighFrameMBAFFWeightedCAVLCPartitionedPResidualPair(t, nals[1], spsList[0], ppsList[0], tt)
	if pair.fieldFlag != tt.fieldFlag {
		t.Fatalf("%s frame-MBAFF weighted residual partitioned-P pair field flag = %d, want %d", tt.name, pair.fieldFlag, tt.fieldFlag)
	}
	wantRefIdxCount := highFrameMBAFFPartitionedPRefIdxCount(t, tt.mbType)
	for i, mb := range []highFrameMBAFFCAVLCPartitionedPMacroblock{pair.top, pair.bottom} {
		if mb.skipRun != 0 || mb.mbType != tt.mbType || mb.cbp != tt.cbp {
			t.Fatalf("%s weighted residual partitioned-P macroblock[%d] skip/mb_type/cbp = %d/%d/%d (code %d), want mb_type %d cbp %d",
				tt.name, i, mb.skipRun, mb.mbType, mb.cbp, mb.cbpCode, tt.mbType, tt.cbp)
		}
		if mb.refIdxCount != wantRefIdxCount {
			t.Fatalf("%s weighted residual partitioned-P macroblock[%d] ref_idx count = %d, want %d", tt.name, i, mb.refIdxCount, wantRefIdxCount)
		}
		for j := 0; j < mb.refIdxCount; j++ {
			if mb.refIdxFlags[j] != tt.fieldFlag {
				t.Fatalf("%s weighted residual partitioned-P macroblock[%d] ref_idx_l0[%d] flag = %d, want %d",
					tt.name, i, j, mb.refIdxFlags[j], tt.fieldFlag)
			}
		}
		if tt.mbType == 3 {
			for j, subType := range mb.subMBType {
				if subType != 0 {
					t.Fatalf("%s weighted residual partitioned-P macroblock[%d] sub_mb_type[%d] = %d, want P_L0_8x8", tt.name, i, j, subType)
				}
			}
		}
	}
}

func readHighFrameMBAFFWeightedCAVLCP16x16ResidualPair(t *testing.T, nal h264.NALUnit, sps *h264.SPS, pps *h264.PPS, tt highFrameMBAFFWeightedResidualPCase) highFrameMBAFFCAVLCP16x16Pair {
	t.Helper()
	br, refCount0 := readHighFrameMBAFFWeightedResidualPHeader(t, nal, sps, pps, tt)
	topSkipRun := br.readUE(t)
	fieldFlag := br.readBit(t)
	refCount0 = h264MBAFFRefCountForSyntax(refCount0, fieldFlag)
	top := readHighFrameMBAFFCAVLCP16x16Macroblock(t, &br, topSkipRun, refCount0, tt.residualTailBits)
	bottomSkipRun := br.readUE(t)
	bottom := readHighFrameMBAFFCAVLCP16x16Macroblock(t, &br, bottomSkipRun, refCount0, tt.residualTailBits)
	return highFrameMBAFFCAVLCP16x16Pair{fieldFlag: fieldFlag, top: top, bottom: bottom}
}

func readHighFrameMBAFFWeightedCAVLCPartitionedPResidualPair(t *testing.T, nal h264.NALUnit, sps *h264.SPS, pps *h264.PPS, tt highFrameMBAFFWeightedResidualPCase) highFrameMBAFFCAVLCPartitionedPPair {
	t.Helper()
	br, refCount0 := readHighFrameMBAFFWeightedResidualPHeader(t, nal, sps, pps, tt)
	topSkipRun := br.readUE(t)
	fieldFlag := br.readBit(t)
	refCount0 = h264MBAFFRefCountForSyntax(refCount0, fieldFlag)
	top := readHighFrameMBAFFCAVLCPartitionedPMacroblock(t, &br, topSkipRun, refCount0, tt.residualTailBits)
	bottomSkipRun := br.readUE(t)
	bottom := readHighFrameMBAFFCAVLCPartitionedPMacroblock(t, &br, bottomSkipRun, refCount0, tt.residualTailBits)
	return highFrameMBAFFCAVLCPartitionedPPair{fieldFlag: fieldFlag, top: top, bottom: bottom}
}

func readHighFrameMBAFFWeightedResidualPHeader(t *testing.T, nal h264.NALUnit, sps *h264.SPS, pps *h264.PPS, tt highFrameMBAFFWeightedResidualPCase) (high10ResidualCAVLCBitReader, uint32) {
	t.Helper()
	if sps == nil || pps == nil {
		t.Fatal("missing SPS/PPS for frame-MBAFF weighted residual-P syntax check")
	}
	br := high10ResidualCAVLCBitReader{data: nal.RBSP}
	firstMB := br.readUE(t)
	rawSliceType := br.readUE(t)
	sliceTypeNoS := high10ResidualCAVLCSliceTypeNoS(t, rawSliceType)
	ppsID := br.readUE(t)
	if firstMB != 0 || sliceTypeNoS != h264.PictureTypeP || ppsID != pps.PPSID {
		t.Fatalf("slice header first_mb/type/pps = %d/%d/%d, want first weighted residual P slice with PPS %d", firstMB, sliceTypeNoS, ppsID, pps.PPSID)
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
	return br, refCount0
}

func assertHighFrameMBAFFWeightedResidualPFrames(t *testing.T, frames []*Frame, tt highFrameMBAFFWeightedResidualPCase) {
	t.Helper()
	assertHighFrameMBAFFWeightedPFrames(t, frames, highFrameMBAFFWeightedResidualPAsWeightedPCase(tt))
}
