// SPDX-License-Identifier: LGPL-2.1-or-later

package goh264_test

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/thesyncim/goh264/internal/h264"
)

const high10PartitionedBFrameRawSize = 768

type high10PartitionedBFixture struct {
	name              string
	file              string
	cabac             int32
	deblockingFilter  int32
	weightedBipredIDC uint32
	refCount          [2]uint32
	wantSlices        []int32
	annexBSize        int
	annexBMD5         string
	frameMD5          []string
	rawVideoMD5       string
}

func TestHigh10PartitionedBFixtureSyntax(t *testing.T) {
	for _, tt := range high10PartitionedBFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := readHigh10PartitionedBFixture(t, tt)
			if len(data) != tt.annexBSize {
				t.Fatalf("annex b size = %d, want %d", len(data), tt.annexBSize)
			}
			sum := md5.Sum(data)
			if got := hex.EncodeToString(sum[:]); got != tt.annexBMD5 {
				t.Fatalf("annex b md5 = %s, want %s", got, tt.annexBMD5)
			}
			assertHigh10PartitionedBFixtureSyntax(t, data, tt)
		})
	}
}

func TestDecodeAnnexBHigh10PartitionedBFrames(t *testing.T) {
	for _, tt := range high10PartitionedBFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := readHigh10PartitionedBFixture(t, tt)
			assertHigh10PartitionedBFixtureSyntax(t, data, tt)
			frames, err := NewDecoder().DecodeAnnexBFrames(data)
			if err != nil {
				t.Fatalf("DecodeAnnexBFrames: %v", err)
			}
			assertHigh10PartitionedBFrames(t, frames, tt)
		})
	}
}

func TestDecodeAVCHigh10PartitionedBFrames(t *testing.T) {
	for _, tt := range high10PartitionedBFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := readHigh10PartitionedBFixture(t, tt)
			assertHigh10PartitionedBFixtureSyntax(t, data, tt)
			for _, nalLengthSize := range []int{2, 3, 4} {
				frames, err := NewDecoder().DecodeAVCFrames(annexBToAVC(t, data, nalLengthSize), nalLengthSize)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: DecodeAVCFrames: %v", nalLengthSize, err)
				}
				assertHigh10PartitionedBFrames(t, frames, tt)
			}
		})
	}
}

func TestDecodeAVCWithConfigurationRecordHigh10PartitionedBFrames(t *testing.T) {
	for _, tt := range high10PartitionedBFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := readHigh10PartitionedBFixture(t, tt)
			assertHigh10PartitionedBFixtureSyntax(t, data, tt)
			for _, nalLengthSize := range []int{2, 3, 4} {
				config, packet := annexBToAVCConfigAndPacket(t, data, nalLengthSize)
				frames, err := NewDecoder().DecodeAVCFramesWithConfigurationRecord(config, packet)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: DecodeAVCFramesWithConfigurationRecord: %v", nalLengthSize, err)
				}
				assertHigh10PartitionedBFrames(t, frames, tt)
			}
		})
	}
}

func TestDecodeConfiguredAVCHigh10PartitionedBFramesAcrossSamplesFlush(t *testing.T) {
	for _, tt := range high10PartitionedBFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := readHigh10PartitionedBFixture(t, tt)
			assertHigh10PartitionedBFixtureSyntax(t, data, tt)
			for _, nalLengthSize := range []int{2, 3, 4} {
				config, samples := annexBToAVCConfigAndSamples(t, data, nalLengthSize)
				if len(samples) != len(tt.frameMD5) {
					t.Fatalf("nalLengthSize=%d: samples = %d, want %d", nalLengthSize, len(samples), len(tt.frameMD5))
				}

				dec := NewDecoder()
				if _, err := dec.ParseAVCDecoderConfigurationRecord(config); err != nil {
					t.Fatalf("nalLengthSize=%d: config: %v", nalLengthSize, err)
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
				assertHigh10PartitionedBFrames(t, frames, tt)
			}
		})
	}
}

func TestDecodeAutoConfiguredAVCHigh10PartitionedBFramesAcrossSamplesFlush(t *testing.T) {
	for _, tt := range high10PartitionedBFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := readHigh10PartitionedBFixture(t, tt)
			config, samples := annexBToAVCConfigAndSamples(t, data, 4)
			if len(samples) != len(tt.frameMD5) {
				t.Fatalf("samples = %d, want %d", len(samples), len(tt.frameMD5))
			}

			dec := NewDecoder()
			out, err := dec.DecodeFrames(config)
			if err != nil {
				t.Fatalf("config: %v", err)
			}
			if len(out) != 0 {
				t.Fatalf("config frames = %d, want 0", len(out))
			}

			var frames []*Frame
			for i, sample := range samples {
				out, err = dec.DecodeFrames(sample)
				if err != nil {
					t.Fatalf("sample[%d]: %v", i, err)
				}
				frames = append(frames, out...)
			}
			out, err = dec.DecodeFrames(nil)
			if err != nil {
				t.Fatalf("flush: %v", err)
			}
			frames = append(frames, out...)
			assertHigh10PartitionedBFrames(t, frames, tt)
		})
	}
}

func TestFFmpegRawVideoMD5OracleHigh10PartitionedB(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}

	for _, tt := range high10PartitionedBFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join("testdata", "h264", tt.file)
			framemd5 := exec.Command("ffmpeg",
				"-hide_banner", "-v", "error",
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
			for i, want := range tt.frameMD5 {
				line := []byte(fmt.Sprintf("0, %10d, %10d,        1, %8d, %s", i, i, high10PartitionedBFrameRawSize, want))
				if !bytes.Contains(framemd5Out, line) {
					t.Fatalf("frame[%d] missing %q in framemd5:\n%s", i, line, framemd5Out)
				}
			}

			rawCmd := exec.Command("ffmpeg",
				"-hide_banner", "-v", "error",
				"-f", "h264",
				"-i", path,
				"-an", "-sn", "-dn",
				"-pix_fmt", "yuv420p10le",
				"-f", "rawvideo",
				"-",
			)
			raw, err := rawCmd.Output()
			if err != nil {
				t.Fatalf("ffmpeg rawvideo: %v", err)
			}
			if len(raw) != len(tt.frameMD5)*high10PartitionedBFrameRawSize {
				t.Fatalf("rawvideo size = %d, want %d", len(raw), len(tt.frameMD5)*high10PartitionedBFrameRawSize)
			}
			sum := md5.Sum(raw)
			if got := hex.EncodeToString(sum[:]); got != tt.rawVideoMD5 {
				t.Fatalf("rawvideo md5 = %s, want %s", got, tt.rawVideoMD5)
			}
		})
	}
}

func high10PartitionedBFixtures() []high10PartitionedBFixture {
	return []high10PartitionedBFixture{
		{
			name:       "cavlc-b16x8",
			file:       "high10_partitioned_b16x8_cavlc.h264",
			annexBSize: 739,
			annexBMD5:  "2798d9490dcb9f4b1495faee8e23c998",
			frameMD5: []string{
				"da42dbbc6702ac820c7162dd19030ea3",
				"6dc0b7afff881b7f69b9176db6c5155e",
				"ae723753e3ae671a34e4f57f325d2cb8",
			},
			rawVideoMD5: "8057ca8e0ee9e2f51fc59b824333e0da",
		},
		{
			name:       "cabac-b16x8",
			file:       "high10_partitioned_b16x8_cabac.h264",
			cabac:      1,
			annexBSize: 741,
			annexBMD5:  "74d9dd3315d2a1b45406508786722c25",
			frameMD5: []string{
				"da42dbbc6702ac820c7162dd19030ea3",
				"6dc0b7afff881b7f69b9176db6c5155e",
				"ae723753e3ae671a34e4f57f325d2cb8",
			},
			rawVideoMD5: "8057ca8e0ee9e2f51fc59b824333e0da",
		},
		{
			name:       "cavlc-b8x16",
			file:       "high10_partitioned_b8x16_cavlc.h264",
			annexBSize: 739,
			annexBMD5:  "8f041ebd2075c5ee3195c6e4ea197d69",
			frameMD5: []string{
				"3de0d9ec87d2b43d34b08554de5509e0",
				"6dc0b7afff881b7f69b9176db6c5155e",
				"360499a4bb17c8730018ce06b58180b7",
			},
			rawVideoMD5: "d927d8a41788f89e93b8d66d54347ec7",
		},
		{
			name:       "cabac-b8x16",
			file:       "high10_partitioned_b8x16_cabac.h264",
			cabac:      1,
			annexBSize: 741,
			annexBMD5:  "0b7b7c3094532f5fff464f7a3819635a",
			frameMD5: []string{
				"3de0d9ec87d2b43d34b08554de5509e0",
				"6dc0b7afff881b7f69b9176db6c5155e",
				"360499a4bb17c8730018ce06b58180b7",
			},
			rawVideoMD5: "d927d8a41788f89e93b8d66d54347ec7",
		},
		{
			name:       "cavlc-b8x8",
			file:       "high10_partitioned_b8x8_cavlc.h264",
			annexBSize: 1406,
			annexBMD5:  "9bd955daf127957bc6684c012a91df6a",
			frameMD5: []string{
				"41ea931c1df0c87907ca7627beeb1dfc",
				"ca7db1692b52de6fd7be03eae5d6b121",
				"e355a7851b20224a769b798c9a63c8b3",
			},
			rawVideoMD5: "017a85619aefcae9c7c98f11f6b829ee",
		},
		{
			name:       "cabac-b8x8",
			file:       "high10_partitioned_b8x8_cabac.h264",
			cabac:      1,
			annexBSize: 1438,
			annexBMD5:  "880484c1f22f9ac1846f5f9cd7652917",
			frameMD5: []string{
				"541565314ead228ebda2b21fc3ee25d6",
				"ccd7e4a2a29432b1db826acd229b78cd",
				"730d70dba915767dc72964eb71a28ae4",
			},
			rawVideoMD5: "63bbee01f26a0382dd58777ccb6c05e3",
		},
		{
			name:              "implicit-cavlc-b16x8",
			file:              "high10_partitioned_implicit_weight_b16x8_cavlc.h264",
			weightedBipredIDC: 2,
			refCount:          [2]uint32{1, 1},
			wantSlices:        []int32{h264.PictureTypeI, h264.PictureTypeP, h264.PictureTypeB, h264.PictureTypeB, h264.PictureTypeP},
			annexBSize:        828,
			annexBMD5:         "f7a8b5d2e8e06a91f9e2b3a011fb2c9f",
			frameMD5: []string{
				"271857125d16f1e579ab8775ff8824e4",
				"4a991c090da8499717e6d8baefc4d99f",
				"45971f69242128a678232982b08bc214",
				"6ce9090764f79f041a6d8e7c8721a071",
				"1eb7b379666ef9c75e9a94137d6234ba",
			},
			rawVideoMD5: "b85b69946077d6e700034f18e03afa02",
		},
		{
			name:              "implicit-cabac-b16x8",
			file:              "high10_partitioned_implicit_weight_b16x8_cabac.h264",
			cabac:             1,
			weightedBipredIDC: 2,
			refCount:          [2]uint32{1, 1},
			wantSlices:        []int32{h264.PictureTypeI, h264.PictureTypeP, h264.PictureTypeB, h264.PictureTypeB, h264.PictureTypeP},
			annexBSize:        829,
			annexBMD5:         "aa7076b8e6ffe06af2af84cdf381cb52",
			frameMD5: []string{
				"0f82b9127b0e6bb9dc711a9458d44b52",
				"b7a8173d25add7d9cd50f026223ae634",
				"0a57a13b2e24dbb4874639eca6a5944f",
				"39cf654fa801077ea9ad0d8ba3325d11",
				"0482be2ceecc0f9610cac239bfe667fe",
			},
			rawVideoMD5: "5954cb46ad68184de947dbb604748924",
		},
		{
			name:              "implicit-cavlc-b8x16",
			file:              "high10_partitioned_implicit_weight_b8x16_cavlc.h264",
			weightedBipredIDC: 2,
			refCount:          [2]uint32{1, 1},
			wantSlices:        []int32{h264.PictureTypeI, h264.PictureTypeP, h264.PictureTypeB, h264.PictureTypeB, h264.PictureTypeP},
			annexBSize:        828,
			annexBMD5:         "34cdb3fd5c7a9e3346acd2187d918c03",
			frameMD5: []string{
				"29d5d2aaed62b66c44d057ba080ad9c2",
				"c974f45970b7fbc4ef7655d273b474b2",
				"ed427f5e7039649fb8b4ffe2205a494a",
				"77808f4a4031f282e052e4af18e2bdc2",
				"d6cffa2c14b584600417d507cb8ebdde",
			},
			rawVideoMD5: "0b5de5fe0388cb1f75b2a462f8b9252a",
		},
		{
			name:              "implicit-cabac-b8x16",
			file:              "high10_partitioned_implicit_weight_b8x16_cabac.h264",
			cabac:             1,
			weightedBipredIDC: 2,
			refCount:          [2]uint32{1, 1},
			wantSlices:        []int32{h264.PictureTypeI, h264.PictureTypeP, h264.PictureTypeB, h264.PictureTypeB, h264.PictureTypeP},
			annexBSize:        825,
			annexBMD5:         "161bcc46653e699e834eff53c0e4df9d",
			frameMD5: []string{
				"966450d2d5db913c02fe419ff4a1071f",
				"366dcdfb255eed2b6573e22f90164815",
				"caaafb37d7c69c4ce4348b7891b2d006",
				"7d4897420172ede22d049bb9d610fc15",
				"5281cb998560a53391d609f50b4b4041",
			},
			rawVideoMD5: "8d8aca4b4693bee11d56c99cf139007f",
		},
		{
			name:              "implicit-cavlc-b8x8",
			file:              "high10_partitioned_implicit_weight_b8x8_cavlc.h264",
			weightedBipredIDC: 2,
			refCount:          [2]uint32{1, 1},
			wantSlices:        []int32{h264.PictureTypeI, h264.PictureTypeP, h264.PictureTypeB, h264.PictureTypeB, h264.PictureTypeP},
			annexBSize:        1001,
			annexBMD5:         "cf2cc71caf7d42bfac77844b6e3c80cf",
			frameMD5: []string{
				"8cbdcb50cd5f5d9131e77984a2bab067",
				"c7346aa3da7874825c8ca3b5d2b95047",
				"98ce911bc3f1729b9539bb139315df53",
				"6b1aa1eb773a8526faded165280930e4",
				"8525deb01886a349e26bd9c6c2ad35d7",
			},
			rawVideoMD5: "d9feb695639d1c22e395c150e8f7f99f",
		},
		{
			name:              "implicit-cabac-b8x8",
			file:              "high10_partitioned_implicit_weight_b8x8_cabac.h264",
			cabac:             1,
			weightedBipredIDC: 2,
			refCount:          [2]uint32{1, 1},
			wantSlices:        []int32{h264.PictureTypeI, h264.PictureTypeP, h264.PictureTypeB, h264.PictureTypeB, h264.PictureTypeP},
			annexBSize:        898,
			annexBMD5:         "558e36221572460fdd1d77b44aaa691a",
			frameMD5: []string{
				"8cbdcb50cd5f5d9131e77984a2bab067",
				"c7346aa3da7874825c8ca3b5d2b95047",
				"5a2212faff41302837ae9faa389f054d",
				"6b1aa1eb773a8526faded165280930e4",
				"8525deb01886a349e26bd9c6c2ad35d7",
			},
			rawVideoMD5: "2306e0d4cd6e403f86776208ccd87c3f",
		},
		{
			name:             "deblock-cavlc-b16x8",
			file:             "high10_partitioned_b_deblock_b16x8_cavlc.h264",
			deblockingFilter: 1,
			refCount:         [2]uint32{1, 1},
			wantSlices:       []int32{h264.PictureTypeI, h264.PictureTypeP, h264.PictureTypeB, h264.PictureTypeB, h264.PictureTypeP},
			annexBSize:       206,
			annexBMD5:        "01095a38261fef19552e4929824ccdcb",
			frameMD5: []string{
				"9570db7f92854146eadcc3957be6d270",
				"0de994fbc787cc9e354041ecce3fdd0e",
				"d6171b07f049f9bfa97568481d5c8d9d",
				"65d16f5c3629cec7d94f270affb515e2",
				"d005979c10c72526ff99388e06193f1c",
			},
			rawVideoMD5: "ce1499b723a2463f097dcbaa82ef88f9",
		},
		{
			name:             "deblock-cabac-b16x8",
			file:             "high10_partitioned_b_deblock_b16x8_cabac.h264",
			cabac:            1,
			deblockingFilter: 1,
			refCount:         [2]uint32{1, 1},
			wantSlices:       []int32{h264.PictureTypeI, h264.PictureTypeP, h264.PictureTypeB, h264.PictureTypeB, h264.PictureTypeP},
			annexBSize:       195,
			annexBMD5:        "5ec8535fb027a28c46fda27e6d4a7b2c",
			frameMD5: []string{
				"02f9ac80c4a2cd773011cad7ccd6ecba",
				"7674c59097f3640cdae50743d968c7be",
				"8e24af2c2559a21c423b6d733399d6a5",
				"65d16f5c3629cec7d94f270affb515e2",
				"d005979c10c72526ff99388e06193f1c",
			},
			rawVideoMD5: "d5dc1d436914f2815eb00bcba1b1ac14",
		},
		{
			name:             "deblock-cavlc-b8x16",
			file:             "high10_partitioned_b_deblock_b8x16_cavlc.h264",
			deblockingFilter: 1,
			refCount:         [2]uint32{1, 1},
			wantSlices:       []int32{h264.PictureTypeI, h264.PictureTypeP, h264.PictureTypeB, h264.PictureTypeB, h264.PictureTypeP},
			annexBSize:       216,
			annexBMD5:        "d5b7fe0071a82c931d1ce7d3104d2407",
			frameMD5: []string{
				"eb4a60270b6233d28dc8418d50ca6b4d",
				"a6dd816bb125d86385e307ccff3e9adc",
				"0bcc4d628ee4ebd33fba195469aefe8d",
				"8973252a29ca2ad29d03a51c03cd36f9",
				"02fc60d34c12e58ae9a576515bace1ac",
			},
			rawVideoMD5: "0ed4ad2f961f74d5a860b2aadef5f667",
		},
		{
			name:             "deblock-cabac-b8x16",
			file:             "high10_partitioned_b_deblock_b8x16_cabac.h264",
			cabac:            1,
			deblockingFilter: 1,
			refCount:         [2]uint32{1, 1},
			wantSlices:       []int32{h264.PictureTypeI, h264.PictureTypeP, h264.PictureTypeB, h264.PictureTypeB, h264.PictureTypeP},
			annexBSize:       197,
			annexBMD5:        "c5d86626f5556bba6be7d3015b2593e4",
			frameMD5: []string{
				"e7d203bec8af43abb7ccf00c20daf3d0",
				"314125962988a819636c8d261aaa86f9",
				"a2a76f9401e1a2e0841c901e8f7f44de",
				"e6f4672ae8b8b2a9532503fee24f7fad",
				"40f1ee0aa76da7ca8f9b7fe0bf9e052d",
			},
			rawVideoMD5: "93441085702db4f988978350cab69119",
		},
		{
			name:             "deblock-cavlc-b8x8",
			file:             "high10_partitioned_b_deblock_b8x8_cavlc.h264",
			deblockingFilter: 1,
			refCount:         [2]uint32{1, 1},
			wantSlices:       []int32{h264.PictureTypeI, h264.PictureTypeP, h264.PictureTypeB, h264.PictureTypeB, h264.PictureTypeP},
			annexBSize:       296,
			annexBMD5:        "9df4e39e473b9722f8ce9c9b5933f0df",
			frameMD5: []string{
				"92ee8f26a66cbf9ec8ea654a22762e94",
				"78e7edf20b24e794425a9889858c3c5f",
				"2e12808266439e7e12637a9765a5b50f",
				"5567bcfcdb8fcad2ff456e922e538235",
				"7ba69fd6ddc4b1987a431b2dce1a6694",
			},
			rawVideoMD5: "8fd167253ff0894b0855dc624561411e",
		},
		{
			name:             "deblock-cabac-b8x8",
			file:             "high10_partitioned_b_deblock_b8x8_cabac.h264",
			cabac:            1,
			deblockingFilter: 1,
			refCount:         [2]uint32{1, 1},
			wantSlices:       []int32{h264.PictureTypeI, h264.PictureTypeP, h264.PictureTypeB, h264.PictureTypeB, h264.PictureTypeP},
			annexBSize:       240,
			annexBMD5:        "25b01cda4d0c46349ee6bb892bda9d62",
			frameMD5: []string{
				"92ee8f26a66cbf9ec8ea654a22762e94",
				"a75a2bdb1fdab73cbb3cca27bb571fc8",
				"2e12808266439e7e12637a9765a5b50f",
				"5567bcfcdb8fcad2ff456e922e538235",
				"6d444e43c2bc194f7c5876575181a40e",
			},
			rawVideoMD5: "73130d29fe042428a34c9d1d9f02c3e4",
		},
		{
			name:              "implicit-deblock-cavlc-b16x8",
			file:              "high10_partitioned_implicit_weight_b_deblock_b16x8_cavlc.h264",
			deblockingFilter:  1,
			weightedBipredIDC: 2,
			refCount:          [2]uint32{1, 1},
			wantSlices:        []int32{h264.PictureTypeI, h264.PictureTypeP, h264.PictureTypeB, h264.PictureTypeB, h264.PictureTypeP},
			annexBSize:        206,
			annexBMD5:         "bff70b73784de5b54ebc89d825be781d",
			frameMD5: []string{
				"9570db7f92854146eadcc3957be6d270",
				"0de994fbc787cc9e354041ecce3fdd0e",
				"d6171b07f049f9bfa97568481d5c8d9d",
				"65d16f5c3629cec7d94f270affb515e2",
				"d005979c10c72526ff99388e06193f1c",
			},
			rawVideoMD5: "ce1499b723a2463f097dcbaa82ef88f9",
		},
		{
			name:              "implicit-deblock-cabac-b16x8",
			file:              "high10_partitioned_implicit_weight_b_deblock_b16x8_cabac.h264",
			cabac:             1,
			deblockingFilter:  1,
			weightedBipredIDC: 2,
			refCount:          [2]uint32{1, 1},
			wantSlices:        []int32{h264.PictureTypeI, h264.PictureTypeP, h264.PictureTypeB, h264.PictureTypeB, h264.PictureTypeP},
			annexBSize:        195,
			annexBMD5:         "5bc300f4a7660f99611569d24be3c67a",
			frameMD5: []string{
				"02f9ac80c4a2cd773011cad7ccd6ecba",
				"7674c59097f3640cdae50743d968c7be",
				"8e24af2c2559a21c423b6d733399d6a5",
				"65d16f5c3629cec7d94f270affb515e2",
				"d005979c10c72526ff99388e06193f1c",
			},
			rawVideoMD5: "d5dc1d436914f2815eb00bcba1b1ac14",
		},
		{
			name:              "implicit-deblock-cavlc-b8x16",
			file:              "high10_partitioned_implicit_weight_b_deblock_b8x16_cavlc.h264",
			deblockingFilter:  1,
			weightedBipredIDC: 2,
			refCount:          [2]uint32{1, 1},
			wantSlices:        []int32{h264.PictureTypeI, h264.PictureTypeP, h264.PictureTypeB, h264.PictureTypeB, h264.PictureTypeP},
			annexBSize:        216,
			annexBMD5:         "2ff41684aee3f31b0f62d00391f7d45a",
			frameMD5: []string{
				"eb4a60270b6233d28dc8418d50ca6b4d",
				"a6dd816bb125d86385e307ccff3e9adc",
				"0bcc4d628ee4ebd33fba195469aefe8d",
				"8973252a29ca2ad29d03a51c03cd36f9",
				"02fc60d34c12e58ae9a576515bace1ac",
			},
			rawVideoMD5: "0ed4ad2f961f74d5a860b2aadef5f667",
		},
		{
			name:              "implicit-deblock-cabac-b8x16",
			file:              "high10_partitioned_implicit_weight_b_deblock_b8x16_cabac.h264",
			cabac:             1,
			deblockingFilter:  1,
			weightedBipredIDC: 2,
			refCount:          [2]uint32{1, 1},
			wantSlices:        []int32{h264.PictureTypeI, h264.PictureTypeP, h264.PictureTypeB, h264.PictureTypeB, h264.PictureTypeP},
			annexBSize:        197,
			annexBMD5:         "8bbc78473ea2d8a97bd7485f66dc0f52",
			frameMD5: []string{
				"e7d203bec8af43abb7ccf00c20daf3d0",
				"314125962988a819636c8d261aaa86f9",
				"a2a76f9401e1a2e0841c901e8f7f44de",
				"e6f4672ae8b8b2a9532503fee24f7fad",
				"40f1ee0aa76da7ca8f9b7fe0bf9e052d",
			},
			rawVideoMD5: "93441085702db4f988978350cab69119",
		},
		{
			name:              "implicit-deblock-cavlc-b8x8",
			file:              "high10_partitioned_implicit_weight_b_deblock_b8x8_cavlc.h264",
			deblockingFilter:  1,
			weightedBipredIDC: 2,
			refCount:          [2]uint32{1, 1},
			wantSlices:        []int32{h264.PictureTypeI, h264.PictureTypeP, h264.PictureTypeB, h264.PictureTypeB, h264.PictureTypeP},
			annexBSize:        292,
			annexBMD5:         "e7fd4fa5d7e352cdd60ebbcbf5499026",
			frameMD5: []string{
				"92ee8f26a66cbf9ec8ea654a22762e94",
				"6e654a6170477e0cccabe38b52a449cc",
				"726c63d619559ea32017c49a0e8a9a9f",
				"5567bcfcdb8fcad2ff456e922e538235",
				"7ba69fd6ddc4b1987a431b2dce1a6694",
			},
			rawVideoMD5: "758eb51ab3fa142adaaafb4ca7871eff",
		},
		{
			name:              "implicit-deblock-cabac-b8x8",
			file:              "high10_partitioned_implicit_weight_b_deblock_b8x8_cabac.h264",
			cabac:             1,
			deblockingFilter:  1,
			weightedBipredIDC: 2,
			refCount:          [2]uint32{1, 1},
			wantSlices:        []int32{h264.PictureTypeI, h264.PictureTypeP, h264.PictureTypeB, h264.PictureTypeB, h264.PictureTypeP},
			annexBSize:        236,
			annexBMD5:         "4d9d0a6130711cb19e276dfe690ccc19",
			frameMD5: []string{
				"92ee8f26a66cbf9ec8ea654a22762e94",
				"ae4bb3e8d65bf5f9220a8b746cdd13a9",
				"d46fb13bdf0c45d39a38bf8e5de846f1",
				"5567bcfcdb8fcad2ff456e922e538235",
				"6d444e43c2bc194f7c5876575181a40e",
			},
			rawVideoMD5: "88e5b24a139b6bc30cf9d879b1f34c56",
		},
	}
}

func readHigh10PartitionedBFixture(t *testing.T, tt high10PartitionedBFixture) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "h264", tt.file))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func assertHigh10PartitionedBFrames(t *testing.T, frames []*Frame, tt high10PartitionedBFixture) {
	t.Helper()
	if len(frames) != len(tt.frameMD5) {
		t.Fatalf("frames = %d, want %d", len(frames), len(tt.frameMD5))
	}
	assertHigh10FrameMD5Strings(t, frames, tt.frameMD5)
	raw := make([]byte, 0, len(frames)*high10PartitionedBFrameRawSize)
	for i, frame := range frames {
		pixFmt, err := frame.RawPixelFormat()
		if err != nil {
			t.Fatalf("frame[%d] RawPixelFormat: %v", i, err)
		}
		bytesPerSample, err := frame.BytesPerSample()
		if err != nil {
			t.Fatalf("frame[%d] BytesPerSample: %v", i, err)
		}
		if pixFmt != "yuv420p10le" || bytesPerSample != 2 {
			t.Fatalf("frame[%d] raw format/sample bytes = %s/%d, want yuv420p10le/2", i, pixFmt, bytesPerSample)
		}
		raw, err = frame.AppendRawYUVBytesLE(raw)
		if err != nil {
			t.Fatalf("frame[%d] append raw yuv: %v", i, err)
		}
		rawSize, err := frame.RawYUVSize()
		if err != nil {
			t.Fatalf("frame[%d] RawYUVSize: %v", i, err)
		}
		if rawSize != high10PartitionedBFrameRawSize {
			t.Fatalf("frame[%d] raw yuv size = %d, want %d", i, rawSize, high10PartitionedBFrameRawSize)
		}
	}
	sum := md5.Sum(raw)
	if got := hex.EncodeToString(sum[:]); got != tt.rawVideoMD5 {
		t.Fatalf("rawvideo md5 = %s, want %s", got, tt.rawVideoMD5)
	}
}

func assertHigh10PartitionedBFixtureSyntax(t *testing.T, data []byte, tt high10PartitionedBFixture) {
	t.Helper()
	nals, err := h264.SplitAnnexB(data)
	if err != nil {
		t.Fatal(err)
	}

	var spsList [32]*h264.SPS
	var ppsList [256]*h264.PPS
	var gotSlices []int32
	for _, nal := range nals {
		switch nal.Type {
		case h264.NALSPS:
			sps, err := h264.DecodeSPS(nal.RBSP)
			if err != nil {
				t.Fatal(err)
			}
			if sps.ProfileIDC != 110 || sps.Width != 16 || sps.Height != 16 ||
				sps.ChromaFormatIDC != 1 || sps.BitDepthLuma != 10 || sps.BitDepthChroma != 10 ||
				sps.FrameMBSOnlyFlag != 1 || sps.MBAFF != 0 || sps.RefFrameCount != 2 {
				t.Fatalf("SPS profile/format = %d %dx%d chroma %d depth %d/%d frame_only %d/%d refs %d, want High10 16x16 yuv420p10le frame-only refs=2",
					sps.ProfileIDC, sps.Width, sps.Height, sps.ChromaFormatIDC, sps.BitDepthLuma, sps.BitDepthChroma,
					sps.FrameMBSOnlyFlag, sps.MBAFF, sps.RefFrameCount)
			}
			spsList[sps.SPSID] = sps
		case h264.NALPPS:
			pps, err := h264.DecodePPS(nal.RBSP, &spsList)
			if err != nil {
				t.Fatal(err)
			}
			wantRefCount := tt.ppsRefCount()
			if pps.CABAC != tt.cabac || pps.Transform8x8Mode != 0 || pps.WeightedPred != 0 ||
				pps.WeightedBipredIDC != tt.weightedBipredIDC ||
				pps.RefCount[0] != wantRefCount[0] || pps.RefCount[1] != wantRefCount[1] {
				t.Fatalf("PPS cabac/8x8/weights/refs = %d/%d/%d/%d/%d/%d, want cabac=%d no-8x8 weighted_bipred_idc=%d refs=%d/%d",
					pps.CABAC, pps.Transform8x8Mode, pps.WeightedPred, pps.WeightedBipredIDC, pps.RefCount[0], pps.RefCount[1],
					tt.cabac, tt.weightedBipredIDC, wantRefCount[0], wantRefCount[1])
			}
			ppsList[pps.PPSID] = pps
		case h264.NALIDRSlice, h264.NALSlice:
			sh, err := h264.ParseSliceHeader(nal, &ppsList)
			if err != nil {
				t.Fatal(err)
			}
			if sh.PictureStructure != h264.PictureFrame || sh.DeblockingFilter != tt.deblockingFilterValue() {
				t.Fatalf("slice picture/deblock = %d/%d, want frame/%d", sh.PictureStructure, sh.DeblockingFilter, tt.deblockingFilterValue())
			}
			switch sh.SliceTypeNoS {
			case h264.PictureTypeI:
				if sh.ListCount != 0 {
					t.Fatalf("I slice lists = %d, want none", sh.ListCount)
				}
			case h264.PictureTypeP:
				if sh.ListCount != 1 || sh.RefCount[0] != 1 ||
					sh.PredWeightTable.UseWeight != 0 || sh.PredWeightTable.UseWeightChroma != 0 {
					t.Fatalf("P slice lists/refs/weights = %d/%v/%d/%d, want L0 refs=1 unweighted",
						sh.ListCount, sh.RefCount, sh.PredWeightTable.UseWeight, sh.PredWeightTable.UseWeightChroma)
				}
			case h264.PictureTypeB:
				if sh.ListCount != 2 || sh.RefCount[0] != 1 || sh.RefCount[1] != 1 ||
					sh.PredWeightTable.UseWeight != 0 || sh.PredWeightTable.UseWeightChroma != 0 {
					t.Fatalf("B slice lists/refs/weights = %d/%v/%d/%d, want L0/L1 refs=1/1 unweighted",
						sh.ListCount, sh.RefCount, sh.PredWeightTable.UseWeight, sh.PredWeightTable.UseWeightChroma)
				}
				if tt.weightedBipredIDC == 2 && sh.DirectSpatialMVPred != 0 {
					t.Fatalf("B slice direct_spatial_mv_pred_flag = %d, want temporal implicit-weight fixture", sh.DirectSpatialMVPred)
				}
			default:
				t.Fatalf("unexpected slice type %d", sh.SliceTypeNoS)
			}
			gotSlices = append(gotSlices, sh.SliceTypeNoS)
		case h264.NALSEI:
		default:
			t.Fatalf("unexpected NAL type %d", nal.Type)
		}
	}
	wantSlices := tt.sliceTypes()
	if len(gotSlices) != len(wantSlices) {
		t.Fatalf("slice count = %d, want %d", len(gotSlices), len(wantSlices))
	}
	for i, want := range wantSlices {
		if gotSlices[i] != want {
			t.Fatalf("slice[%d] type = %d, want %d", i, gotSlices[i], want)
		}
	}
}

func (tt high10PartitionedBFixture) ppsRefCount() [2]uint32 {
	if tt.refCount != [2]uint32{} {
		return tt.refCount
	}
	return [2]uint32{2, 1}
}

func (tt high10PartitionedBFixture) deblockingFilterValue() int32 {
	if tt.deblockingFilter != 0 {
		return tt.deblockingFilter
	}
	return 0
}

func (tt high10PartitionedBFixture) sliceTypes() []int32 {
	if len(tt.wantSlices) != 0 {
		return tt.wantSlices
	}
	return []int32{h264.PictureTypeI, h264.PictureTypeP, h264.PictureTypeB}
}
