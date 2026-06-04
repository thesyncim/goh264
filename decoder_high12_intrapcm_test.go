// SPDX-License-Identifier: LGPL-2.1-or-later

package goh264

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

const (
	high12IntraPCMFrameMD5    = "c361aa6cd60683fabf155b7e0baec348"
	high12IntraPCMRawVideoMD5 = "c361aa6cd60683fabf155b7e0baec348"

	high12Intra16x16NoResidualBitstreamMD5 = "39d964d722bf4d6ea2f4f7c6ccd2c296"
	high12Intra16x16NoResidualFrameMD5     = "d4753b9733af2865470fb72f96a37071"
	high12Intra16x16NoResidualRawVideoMD5  = "d4753b9733af2865470fb72f96a37071"

	high14Intra16x16NoResidualBitstreamMD5 = "48233a9be1d9acbbd336c2321ff6570a"
	high14Intra16x16NoResidualFrameMD5     = "6d3514a30f506561e144447d287270ab"
	high14Intra16x16NoResidualRawVideoMD5  = "6d3514a30f506561e144447d287270ab"

	highIntra16x16NoResidualPayloadBits = "00100111"
	highIntra16x16LumaDCPayloadBits     = "00100110101"
	highIntra16x16ChromaDCPayloadBits   = "000100011110101"
	highIntra16x16ChromaACPayloadBits   = "0001100111010101011111111"
	highIntra16x16ChromaDCACPayloadBits = "00011001111010101011111111"
	highIntra16x16LumaChromaPayloadBits = "00001100011010101011111111111111111010101011111111"
	highIntra16x16LumaACPayloadBits     = "0000100001110101111111111111111"
	highIntra16x16LumaDCACPayloadBits   = "0000100001101010101111111111111111"

	high12Intra16x16LumaDCBitstreamMD5 = "db88e5b3785156e19cfde136054cba7a"
	high12Intra16x16LumaDCFrameMD5     = "a759ed5c0b6de7f1fa9b461d4bf176e7"
	high12Intra16x16LumaDCRawVideoMD5  = "a759ed5c0b6de7f1fa9b461d4bf176e7"

	high12Intra16x16ChromaDCBitstreamMD5 = "4168785c49fca52642cdd50135ecce34"
	high12Intra16x16ChromaDCFrameMD5     = "742d60bd66f7503b7c5faa78aabb8625"
	high12Intra16x16ChromaDCRawVideoMD5  = "742d60bd66f7503b7c5faa78aabb8625"

	high12Intra16x16ChromaACBitstreamMD5 = "95e99dcf9eeb52546881fb9111b06f95"
	high12Intra16x16ChromaACFrameMD5     = "be585084fde7ca45efc333e9c63bc5bb"
	high12Intra16x16ChromaACRawVideoMD5  = "be585084fde7ca45efc333e9c63bc5bb"

	high12Intra16x16ChromaDCACBitstreamMD5 = "a29edb105f9d01fe9df33dbc477db316"
	high12Intra16x16ChromaDCACFrameMD5     = "c93d5009915b3181bd70b84b654a2820"
	high12Intra16x16ChromaDCACRawVideoMD5  = "c93d5009915b3181bd70b84b654a2820"

	high12Intra16x16LumaChromaBitstreamMD5 = "2e7d638c5ce2b3532c1f817239ee299e"
	high12Intra16x16LumaChromaFrameMD5     = "9310534e8bd2d5768c9f632f7d530ebd"
	high12Intra16x16LumaChromaRawVideoMD5  = "9310534e8bd2d5768c9f632f7d530ebd"

	high12Intra16x16LumaACBitstreamMD5 = "eb781a441f40c750c0f38a8c688ad0b3"
	high12Intra16x16LumaACFrameMD5     = "03f443e3365aad5c36f629456a498d1f"
	high12Intra16x16LumaACRawVideoMD5  = "03f443e3365aad5c36f629456a498d1f"

	high12Intra16x16LumaDCACBitstreamMD5 = "4289290a8dbceaf42a4c55685b80002f"
	high12Intra16x16LumaDCACFrameMD5     = "6f0c86109525f93b777cf0f2a5f09999"
	high12Intra16x16LumaDCACRawVideoMD5  = "6f0c86109525f93b777cf0f2a5f09999"

	high14Intra16x16LumaDCBitstreamMD5 = "7090e9e0d19c5dc6e3d9a9a11d20209d"
	high14Intra16x16LumaDCFrameMD5     = "5d9e2f990aeb152b36edfbc28f6abec7"
	high14Intra16x16LumaDCRawVideoMD5  = "5d9e2f990aeb152b36edfbc28f6abec7"

	high14Intra16x16ChromaDCBitstreamMD5 = "73c071e839a2d96f8cd52709c5c26851"
	high14Intra16x16ChromaDCFrameMD5     = "f5b3e7d590aaf9e448651b77df72e30f"
	high14Intra16x16ChromaDCRawVideoMD5  = "f5b3e7d590aaf9e448651b77df72e30f"

	high14Intra16x16ChromaACBitstreamMD5 = "6ff81aa5dc7f9b62c6ddca8962e4fb15"
	high14Intra16x16ChromaACFrameMD5     = "033152d65549e9a6641d19b586946895"
	high14Intra16x16ChromaACRawVideoMD5  = "033152d65549e9a6641d19b586946895"

	high14Intra16x16ChromaDCACBitstreamMD5 = "19eaa43e48c052cc2eafc1d817cc00ac"
	high14Intra16x16ChromaDCACFrameMD5     = "ce2a75c854b91ec711b27b9d5637eb07"
	high14Intra16x16ChromaDCACRawVideoMD5  = "ce2a75c854b91ec711b27b9d5637eb07"

	high14Intra16x16LumaChromaBitstreamMD5 = "bb4a22276d013cd7ac065267bba3f539"
	high14Intra16x16LumaChromaFrameMD5     = "efff96b33bda86086ce433d1ca8ae196"
	high14Intra16x16LumaChromaRawVideoMD5  = "efff96b33bda86086ce433d1ca8ae196"

	high14Intra16x16LumaACBitstreamMD5 = "b42bd1d26d007d4536a8b2e6ee67c8e5"
	high14Intra16x16LumaACFrameMD5     = "e94a172beb538c7b9e5c8827f4631369"
	high14Intra16x16LumaACRawVideoMD5  = "e94a172beb538c7b9e5c8827f4631369"

	high14Intra16x16LumaDCACBitstreamMD5 = "0ba6ed1200a99e0256408a1ac0a9e3f9"
	high14Intra16x16LumaDCACFrameMD5     = "d976ce00b5590f5056d94fb917d6e5fd"
	high14Intra16x16LumaDCACRawVideoMD5  = "d976ce00b5590f5056d94fb917d6e5fd"

	high12InterPSkipBitstreamMD5            = "7d2997459703cf739a1404ede5b5218d"
	high12InterP16x16NoResidualBitstreamMD5 = "64d3bcb481cdd7cbe1c6f38177843e04"
	high12InterNoResidualRawVideoMD5        = "d4fbc620f3b054967df19b16392bdc6e"
	high14InterPSkipBitstreamMD5            = "9af4dab76e9f0777543897185a094a0d"
	high14InterP16x16NoResidualBitstreamMD5 = "631b1774ab400cbbbadefac0e58805ec"
	high14InterNoResidualRawVideoMD5        = "522820c0c78e0aee053dfe1aae6269b7"
	highInterPSkipPayloadBits               = "010"
	highInterP16x16NoResidualPayloadBits    = "11111"

	high12InterP16x16LumaResidualBitstreamMD5 = "9717c3ebd61678db0a17e20369931903"
	high12InterP16x16LumaResidualPFrameMD5    = "a6b78a4a17e555decd92d49120f74b5b"
	high12InterP16x16LumaResidualRawVideoMD5  = "6f8bc26e50de7e2c55191803ad3917e8"
	high14InterP16x16LumaResidualBitstreamMD5 = "afa6d613f6e9c81655752d5bbacabba9"
	high14InterP16x16LumaResidualPFrameMD5    = "99f7baedfa2ca103d92a7f74a2e99dea"
	high14InterP16x16LumaResidualRawVideoMD5  = "4ac5d6e045b7f035a57dd64a849f7a1a"
	highInterP16x16LumaResidualPayloadBits    = "111101110101111"

	high12InterP16x16LumaChromaResidualBitstreamMD5 = "ea856fbd13d1298814c737e492bdf9b0"
	high12InterP16x16LumaChromaResidualPFrameMD5    = "bd2cd992f97429434a90b73812b22163"
	high12InterP16x16LumaChromaResidualRawVideoMD5  = "da2c7851de5f2ae340ce0c38b559ef03"
	high14InterP16x16LumaChromaResidualBitstreamMD5 = "46db51094d45f59d2c0e4be824e9911b"
	high14InterP16x16LumaChromaResidualPFrameMD5    = "2761c46974ccb370f321c424d735d7c2"
	high14InterP16x16LumaChromaResidualRawVideoMD5  = "d33933171bf821a0c3f9597144b6083c"
	highInterP16x16LumaChromaResidualPayloadBits    = "111100001100110101111010101011111111"

	high12InterP16x8LumaChromaResidualBitstreamMD5 = "c722cf9cefd3462326ebfda080ad79f4"
	high12InterP16x8LumaChromaResidualPFrameMD5    = "bd2cd992f97429434a90b73812b22163"
	high12InterP16x8LumaChromaResidualRawVideoMD5  = "da2c7851de5f2ae340ce0c38b559ef03"
	high14InterP16x8LumaChromaResidualBitstreamMD5 = "4a91e2ad4e1dc583d0a6e815aa1d7022"
	high14InterP16x8LumaChromaResidualPFrameMD5    = "2761c46974ccb370f321c424d735d7c2"
	high14InterP16x8LumaChromaResidualRawVideoMD5  = "d33933171bf821a0c3f9597144b6083c"
	highInterP16x8LumaChromaResidualPayloadBits    = "1010111100001100110101111010101011111111"

	high12InterP8x16LumaChromaResidualBitstreamMD5 = "53642ce14ee7aa4eec9bb1181fdf0010"
	high12InterP8x16LumaChromaResidualPFrameMD5    = "bd2cd992f97429434a90b73812b22163"
	high12InterP8x16LumaChromaResidualRawVideoMD5  = "da2c7851de5f2ae340ce0c38b559ef03"
	high14InterP8x16LumaChromaResidualBitstreamMD5 = "4be49555368766631ee61918ac80965b"
	high14InterP8x16LumaChromaResidualPFrameMD5    = "2761c46974ccb370f321c424d735d7c2"
	high14InterP8x16LumaChromaResidualRawVideoMD5  = "d33933171bf821a0c3f9597144b6083c"
	highInterP8x16LumaChromaResidualPayloadBits    = "1011111100001100110101111010101011111111"

	high12InterP8x8LumaChromaResidualBitstreamMD5 = "d7f35ee61c3f3707407378498bf32911"
	high12InterP8x8LumaChromaResidualPFrameMD5    = "bd2cd992f97429434a90b73812b22163"
	high12InterP8x8LumaChromaResidualRawVideoMD5  = "da2c7851de5f2ae340ce0c38b559ef03"
	high14InterP8x8LumaChromaResidualBitstreamMD5 = "6aa823e729d0890d4b2992eac254c4d7"
	high14InterP8x8LumaChromaResidualPFrameMD5    = "2761c46974ccb370f321c424d735d7c2"
	high14InterP8x8LumaChromaResidualRawVideoMD5  = "d33933171bf821a0c3f9597144b6083c"
	highInterP8x8LumaChromaResidualPayloadBits    = "10010011111111111100001100110101111010101011111111"
)

func TestHigh12IntraPCMFixtureSyntax(t *testing.T) {
	assertHigh12IntraPCMFixtureSyntax(t, readHigh12IntraPCMFixture(t))
}

func TestDecodeAnnexBHigh12IntraPCMFrame(t *testing.T) {
	data := readHigh12IntraPCMFixture(t)
	assertHigh12IntraPCMFixtureSyntax(t, data)

	frames, err := NewDecoder().DecodeAnnexBFrames(data)
	if err != nil {
		t.Fatalf("decode High12 IntraPCM Annex B: %v", err)
	}
	assertHigh12IntraPCMFrames(t, frames)
}

func TestDecodeAVCHigh12IntraPCMFrame(t *testing.T) {
	data := readHigh12IntraPCMFixture(t)
	assertHigh12IntraPCMFixtureSyntax(t, data)

	for _, nalLengthSize := range []int{2, 3, 4} {
		frames, err := NewDecoder().DecodeAVCFrames(annexBToAVC(t, data, nalLengthSize), nalLengthSize)
		if err != nil {
			t.Fatalf("nalLengthSize=%d: %v", nalLengthSize, err)
		}
		assertHigh12IntraPCMFrames(t, frames)
	}
}

func TestDecodeAVCWithConfigurationRecordHigh12IntraPCMFrame(t *testing.T) {
	data := readHigh12IntraPCMFixture(t)
	assertHigh12IntraPCMFixtureSyntax(t, data)

	for _, nalLengthSize := range []int{2, 3, 4} {
		config, packet := annexBToAVCConfigAndPacket(t, data, nalLengthSize)
		frames, err := NewDecoder().DecodeAVCFramesWithConfigurationRecord(config, packet)
		if err != nil {
			t.Fatalf("nalLengthSize=%d: %v", nalLengthSize, err)
		}
		assertHigh12IntraPCMFrames(t, frames)
	}
}

func TestFFmpegRawVideoMD5OracleHigh12IntraPCM(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}

	path := high12IntraPCMFixturePath(t)
	framemd5 := exec.Command("ffmpeg",
		"-v", "error",
		"-f", "h264",
		"-i", path,
		"-an", "-sn", "-dn",
		"-pix_fmt", "yuv420p12le",
		"-f", "framemd5",
		"-",
	)
	framemd5Out, err := framemd5.Output()
	if err != nil {
		t.Fatalf("ffmpeg framemd5: %v", err)
	}
	line := []byte(fmt.Sprintf("0, %10d, %10d,        1,      768, %s", 0, 0, high12IntraPCMFrameMD5))
	if !bytes.Contains(framemd5Out, line) {
		t.Fatalf("missing %q in framemd5:\n%s", line, framemd5Out)
	}

	rawvideo := exec.Command("ffmpeg",
		"-v", "error",
		"-f", "h264",
		"-i", path,
		"-an", "-sn", "-dn",
		"-pix_fmt", "yuv420p12le",
		"-f", "rawvideo",
		"-",
	)
	raw, err := rawvideo.Output()
	if err != nil {
		t.Fatalf("ffmpeg rawvideo: %v", err)
	}
	if len(raw) != 768 {
		t.Fatalf("rawvideo size = %d, want 768", len(raw))
	}
	sum := md5.Sum(raw)
	if got := hex.EncodeToString(sum[:]); got != high12IntraPCMRawVideoMD5 {
		t.Fatalf("rawvideo md5 = %s, want %s", got, high12IntraPCMRawVideoMD5)
	}
}

func TestHigh12High14Intra16x16NoResidualFixtureSyntax(t *testing.T) {
	for _, tt := range highIntra16x16NoResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highIntra16x16NoResidualFixture(tt.bitDepth)
			sum := md5.Sum(data)
			if got := hex.EncodeToString(sum[:]); got != tt.bitstreamMD5 {
				t.Fatalf("%s bitstream md5 = %s, want %s", tt.name, got, tt.bitstreamMD5)
			}
			assertHighIntra16x16FixtureSyntax(t, data, tt.bitDepth)
		})
	}
}

func TestDecodeAnnexBHigh12High14Intra16x16NoResidualFrame(t *testing.T) {
	for _, tt := range highIntra16x16NoResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highIntra16x16NoResidualFixture(tt.bitDepth)
			assertHighIntra16x16FixtureSyntax(t, data, tt.bitDepth)

			frames, err := NewDecoder().DecodeAnnexBFrames(data)
			if err != nil {
				t.Fatalf("decode %s Annex B: %v", tt.name, err)
			}
			assertHighIntra16x16Frames(t, frames, tt.bitDepth, tt.frameMD5)
		})
	}
}

func TestDecodeAVCHigh12High14Intra16x16NoResidualFrame(t *testing.T) {
	for _, tt := range highIntra16x16NoResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highIntra16x16NoResidualFixture(tt.bitDepth)
			assertHighIntra16x16FixtureSyntax(t, data, tt.bitDepth)

			for _, nalLengthSize := range []int{2, 3, 4} {
				frames, err := NewDecoder().DecodeAVCFrames(annexBToAVC(t, data, nalLengthSize), nalLengthSize)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: %v", nalLengthSize, err)
				}
				assertHighIntra16x16Frames(t, frames, tt.bitDepth, tt.frameMD5)
			}
		})
	}
}

func TestDecodeAVCWithConfigurationRecordHigh12High14Intra16x16NoResidualFrame(t *testing.T) {
	for _, tt := range highIntra16x16NoResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highIntra16x16NoResidualFixture(tt.bitDepth)
			assertHighIntra16x16FixtureSyntax(t, data, tt.bitDepth)

			for _, nalLengthSize := range []int{2, 3, 4} {
				config, packet := annexBToAVCConfigAndPacket(t, data, nalLengthSize)
				frames, err := NewDecoder().DecodeAVCFramesWithConfigurationRecord(config, packet)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: %v", nalLengthSize, err)
				}
				assertHighIntra16x16Frames(t, frames, tt.bitDepth, tt.frameMD5)
			}
		})
	}
}

func TestFFmpegRawVideoMD5OracleHigh12High14Intra16x16NoResidual(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}

	for _, tt := range highIntra16x16NoResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highIntra16x16NoResidualFixture(tt.bitDepth)
			assertHighIntra16x16FixtureSyntax(t, data, tt.bitDepth)
			path := writeTempH264(t, data)
			pixFmt := fmt.Sprintf("yuv420p%dle", tt.bitDepth)

			framemd5 := exec.Command("ffmpeg",
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
			line := []byte(fmt.Sprintf("0, %10d, %10d,        1,      768, %s", 0, 0, tt.frameMD5))
			if !bytes.Contains(framemd5Out, line) {
				t.Fatalf("missing %q in framemd5:\n%s", line, framemd5Out)
			}

			rawvideo := exec.Command("ffmpeg",
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
			if len(raw) != 768 {
				t.Fatalf("rawvideo size = %d, want 768", len(raw))
			}
			sum := md5.Sum(raw)
			if got := hex.EncodeToString(sum[:]); got != tt.rawVideoMD5 {
				t.Fatalf("rawvideo md5 = %s, want %s", got, tt.rawVideoMD5)
			}
		})
	}
}

func TestHigh12High14Intra16x16ResidualFixtureSyntax(t *testing.T) {
	for _, tt := range highIntra16x16ResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highIntra16x16ResidualFixture(tt.bitDepth, tt.payloadBits)
			sum := md5.Sum(data)
			if got := hex.EncodeToString(sum[:]); got != tt.bitstreamMD5 {
				t.Fatalf("%s bitstream md5 = %s, want %s", tt.name, got, tt.bitstreamMD5)
			}
			assertHighIntra16x16FixtureSyntax(t, data, tt.bitDepth)
		})
	}
}

func TestDecodeAnnexBHigh12High14Intra16x16ResidualFrame(t *testing.T) {
	for _, tt := range highIntra16x16ResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highIntra16x16ResidualFixture(tt.bitDepth, tt.payloadBits)
			assertHighIntra16x16FixtureSyntax(t, data, tt.bitDepth)

			frames, err := NewDecoder().DecodeAnnexBFrames(data)
			if err != nil {
				t.Fatalf("decode %s Annex B: %v", tt.name, err)
			}
			assertHighIntra16x16Frames(t, frames, tt.bitDepth, tt.frameMD5)
		})
	}
}

func TestDecodeAVCHigh12High14Intra16x16ResidualFrame(t *testing.T) {
	for _, tt := range highIntra16x16ResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highIntra16x16ResidualFixture(tt.bitDepth, tt.payloadBits)
			assertHighIntra16x16FixtureSyntax(t, data, tt.bitDepth)

			for _, nalLengthSize := range []int{2, 3, 4} {
				frames, err := NewDecoder().DecodeAVCFrames(annexBToAVC(t, data, nalLengthSize), nalLengthSize)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: %v", nalLengthSize, err)
				}
				assertHighIntra16x16Frames(t, frames, tt.bitDepth, tt.frameMD5)
			}
		})
	}
}

func TestDecodeAVCWithConfigurationRecordHigh12High14Intra16x16ResidualFrame(t *testing.T) {
	for _, tt := range highIntra16x16ResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highIntra16x16ResidualFixture(tt.bitDepth, tt.payloadBits)
			assertHighIntra16x16FixtureSyntax(t, data, tt.bitDepth)

			for _, nalLengthSize := range []int{2, 3, 4} {
				config, packet := annexBToAVCConfigAndPacket(t, data, nalLengthSize)
				frames, err := NewDecoder().DecodeAVCFramesWithConfigurationRecord(config, packet)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: %v", nalLengthSize, err)
				}
				assertHighIntra16x16Frames(t, frames, tt.bitDepth, tt.frameMD5)
			}
		})
	}
}

func TestFFmpegRawVideoMD5OracleHigh12High14Intra16x16Residual(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}

	for _, tt := range highIntra16x16ResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highIntra16x16ResidualFixture(tt.bitDepth, tt.payloadBits)
			assertHighIntra16x16FixtureSyntax(t, data, tt.bitDepth)
			path := writeTempH264(t, data)
			pixFmt := fmt.Sprintf("yuv420p%dle", tt.bitDepth)

			framemd5 := exec.Command("ffmpeg",
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
			line := []byte(fmt.Sprintf("0, %10d, %10d,        1,      768, %s", 0, 0, tt.frameMD5))
			if !bytes.Contains(framemd5Out, line) {
				t.Fatalf("missing %q in framemd5:\n%s", line, framemd5Out)
			}

			rawvideo := exec.Command("ffmpeg",
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
			if len(raw) != 768 {
				t.Fatalf("rawvideo size = %d, want 768", len(raw))
			}
			sum := md5.Sum(raw)
			if got := hex.EncodeToString(sum[:]); got != tt.rawVideoMD5 {
				t.Fatalf("rawvideo md5 = %s, want %s", got, tt.rawVideoMD5)
			}
		})
	}
}

func TestHigh12High14InterNoResidualFixtureSyntax(t *testing.T) {
	for _, tt := range highInterNoResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highInterNoResidualFixture(tt.bitDepth, tt.payloadBits)
			sum := md5.Sum(data)
			if got := hex.EncodeToString(sum[:]); got != tt.bitstreamMD5 {
				t.Fatalf("%s bitstream md5 = %s, want %s", tt.name, got, tt.bitstreamMD5)
			}
			assertHighInterNoResidualFixtureSyntax(t, data, tt.bitDepth)
		})
	}
}

func TestDecodeAnnexBHigh12High14InterNoResidualFrames(t *testing.T) {
	for _, tt := range highInterNoResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highInterNoResidualFixture(tt.bitDepth, tt.payloadBits)
			assertHighInterNoResidualFixtureSyntax(t, data, tt.bitDepth)

			frames, err := NewDecoder().DecodeAnnexBFrames(data)
			if err != nil {
				t.Fatalf("decode %s Annex B: %v", tt.name, err)
			}
			assertHighInterNoResidualFrames(t, frames, tt.bitDepth, tt.frameMD5)
		})
	}
}

func TestDecodeAVCHigh12High14InterNoResidualFrames(t *testing.T) {
	for _, tt := range highInterNoResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highInterNoResidualFixture(tt.bitDepth, tt.payloadBits)
			assertHighInterNoResidualFixtureSyntax(t, data, tt.bitDepth)

			for _, nalLengthSize := range []int{2, 3, 4} {
				frames, err := NewDecoder().DecodeAVCFrames(annexBToAVC(t, data, nalLengthSize), nalLengthSize)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: %v", nalLengthSize, err)
				}
				assertHighInterNoResidualFrames(t, frames, tt.bitDepth, tt.frameMD5)
			}
		})
	}
}

func TestDecodeAVCWithConfigurationRecordHigh12High14InterNoResidualFrames(t *testing.T) {
	for _, tt := range highInterNoResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highInterNoResidualFixture(tt.bitDepth, tt.payloadBits)
			assertHighInterNoResidualFixtureSyntax(t, data, tt.bitDepth)

			for _, nalLengthSize := range []int{2, 3, 4} {
				config, packet := annexBToAVCConfigAndPacket(t, data, nalLengthSize)
				frames, err := NewDecoder().DecodeAVCFramesWithConfigurationRecord(config, packet)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: %v", nalLengthSize, err)
				}
				assertHighInterNoResidualFrames(t, frames, tt.bitDepth, tt.frameMD5)
			}
		})
	}
}

func TestFFmpegRawVideoMD5OracleHigh12High14InterNoResidual(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}

	for _, tt := range highInterNoResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highInterNoResidualFixture(tt.bitDepth, tt.payloadBits)
			assertHighInterNoResidualFixtureSyntax(t, data, tt.bitDepth)
			path := writeTempH264(t, data)
			pixFmt := fmt.Sprintf("yuv420p%dle", tt.bitDepth)

			framemd5 := exec.Command("ffmpeg",
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
			for i := 0; i < 2; i++ {
				line := []byte(fmt.Sprintf("0, %10d, %10d,        1,      768, %s", i, i, tt.frameMD5))
				if !bytes.Contains(framemd5Out, line) {
					t.Fatalf("missing %q in framemd5:\n%s", line, framemd5Out)
				}
			}

			rawvideo := exec.Command("ffmpeg",
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
			if len(raw) != 1536 {
				t.Fatalf("rawvideo size = %d, want 1536", len(raw))
			}
			sum := md5.Sum(raw)
			if got := hex.EncodeToString(sum[:]); got != tt.rawVideoMD5 {
				t.Fatalf("rawvideo md5 = %s, want %s", got, tt.rawVideoMD5)
			}
		})
	}
}

func TestHigh12High14InterP16x16ResidualFixtureSyntax(t *testing.T) {
	for _, tt := range highInterP16x16ResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highInterP16x16ResidualFixture(tt.bitDepth)
			sum := md5.Sum(data)
			if got := hex.EncodeToString(sum[:]); got != tt.bitstreamMD5 {
				t.Fatalf("%s bitstream md5 = %s, want %s", tt.name, got, tt.bitstreamMD5)
			}
			assertHighInterP16x16ResidualFixtureSyntax(t, data, tt.bitDepth)
		})
	}
}

func TestDecodeAnnexBHigh12High14InterP16x16ResidualFrames(t *testing.T) {
	for _, tt := range highInterP16x16ResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highInterP16x16ResidualFixture(tt.bitDepth)
			assertHighInterP16x16ResidualFixtureSyntax(t, data, tt.bitDepth)

			frames, err := NewDecoder().DecodeAnnexBFrames(data)
			if err != nil {
				t.Fatalf("decode %s Annex B: %v", tt.name, err)
			}
			assertHighInterP16x16ResidualFrames(t, frames, tt.bitDepth, tt.refFrameMD5, tt.pFrameMD5, tt.rawVideoMD5)
		})
	}
}

func TestDecodeAVCHigh12High14InterP16x16ResidualFrames(t *testing.T) {
	for _, tt := range highInterP16x16ResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highInterP16x16ResidualFixture(tt.bitDepth)
			assertHighInterP16x16ResidualFixtureSyntax(t, data, tt.bitDepth)

			for _, nalLengthSize := range []int{2, 3, 4} {
				frames, err := NewDecoder().DecodeAVCFrames(annexBToAVC(t, data, nalLengthSize), nalLengthSize)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: %v", nalLengthSize, err)
				}
				assertHighInterP16x16ResidualFrames(t, frames, tt.bitDepth, tt.refFrameMD5, tt.pFrameMD5, tt.rawVideoMD5)
			}
		})
	}
}

func TestDecodeAVCWithConfigurationRecordHigh12High14InterP16x16ResidualFrames(t *testing.T) {
	for _, tt := range highInterP16x16ResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highInterP16x16ResidualFixture(tt.bitDepth)
			assertHighInterP16x16ResidualFixtureSyntax(t, data, tt.bitDepth)

			for _, nalLengthSize := range []int{2, 3, 4} {
				config, packet := annexBToAVCConfigAndPacket(t, data, nalLengthSize)
				frames, err := NewDecoder().DecodeAVCFramesWithConfigurationRecord(config, packet)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: %v", nalLengthSize, err)
				}
				assertHighInterP16x16ResidualFrames(t, frames, tt.bitDepth, tt.refFrameMD5, tt.pFrameMD5, tt.rawVideoMD5)
			}
		})
	}
}

func TestFFmpegRawVideoMD5OracleHigh12High14InterP16x16Residual(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}

	for _, tt := range highInterP16x16ResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highInterP16x16ResidualFixture(tt.bitDepth)
			assertHighInterP16x16ResidualFixtureSyntax(t, data, tt.bitDepth)
			path := writeTempH264(t, data)
			pixFmt := fmt.Sprintf("yuv420p%dle", tt.bitDepth)

			framemd5 := exec.Command("ffmpeg",
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
				line := []byte(fmt.Sprintf("0, %10d, %10d,        1,      768, %s", i, i, want))
				if !bytes.Contains(framemd5Out, line) {
					t.Fatalf("missing %q in framemd5:\n%s", line, framemd5Out)
				}
			}

			rawvideo := exec.Command("ffmpeg",
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
			if len(raw) != 1536 {
				t.Fatalf("rawvideo size = %d, want 1536", len(raw))
			}
			sum := md5.Sum(raw)
			if got := hex.EncodeToString(sum[:]); got != tt.rawVideoMD5 {
				t.Fatalf("rawvideo md5 = %s, want %s", got, tt.rawVideoMD5)
			}
		})
	}
}

func TestHigh12High14InterP16x16LumaChromaResidualFixtureSyntax(t *testing.T) {
	for _, tt := range highInterP16x16LumaChromaResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highInterP16x16LumaChromaResidualFixture(tt.bitDepth)
			sum := md5.Sum(data)
			if got := hex.EncodeToString(sum[:]); got != tt.bitstreamMD5 {
				t.Fatalf("%s bitstream md5 = %s, want %s", tt.name, got, tt.bitstreamMD5)
			}
			assertHighInterP16x16LumaChromaResidualFixtureSyntax(t, data, tt.bitDepth)
		})
	}
}

func TestDecodeAnnexBHigh12High14InterP16x16LumaChromaResidualFrames(t *testing.T) {
	for _, tt := range highInterP16x16LumaChromaResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highInterP16x16LumaChromaResidualFixture(tt.bitDepth)
			assertHighInterP16x16LumaChromaResidualFixtureSyntax(t, data, tt.bitDepth)

			frames, err := NewDecoder().DecodeAnnexBFrames(data)
			if err != nil {
				t.Fatalf("decode %s Annex B: %v", tt.name, err)
			}
			assertHighInterP16x16ResidualFrames(t, frames, tt.bitDepth, tt.refFrameMD5, tt.pFrameMD5, tt.rawVideoMD5)
		})
	}
}

func TestDecodeAVCHigh12High14InterP16x16LumaChromaResidualFrames(t *testing.T) {
	for _, tt := range highInterP16x16LumaChromaResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highInterP16x16LumaChromaResidualFixture(tt.bitDepth)
			assertHighInterP16x16LumaChromaResidualFixtureSyntax(t, data, tt.bitDepth)

			for _, nalLengthSize := range []int{2, 3, 4} {
				frames, err := NewDecoder().DecodeAVCFrames(annexBToAVC(t, data, nalLengthSize), nalLengthSize)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: %v", nalLengthSize, err)
				}
				assertHighInterP16x16ResidualFrames(t, frames, tt.bitDepth, tt.refFrameMD5, tt.pFrameMD5, tt.rawVideoMD5)
			}
		})
	}
}

func TestDecodeAVCWithConfigurationRecordHigh12High14InterP16x16LumaChromaResidualFrames(t *testing.T) {
	for _, tt := range highInterP16x16LumaChromaResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highInterP16x16LumaChromaResidualFixture(tt.bitDepth)
			assertHighInterP16x16LumaChromaResidualFixtureSyntax(t, data, tt.bitDepth)

			for _, nalLengthSize := range []int{2, 3, 4} {
				config, packet := annexBToAVCConfigAndPacket(t, data, nalLengthSize)
				frames, err := NewDecoder().DecodeAVCFramesWithConfigurationRecord(config, packet)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: %v", nalLengthSize, err)
				}
				assertHighInterP16x16ResidualFrames(t, frames, tt.bitDepth, tt.refFrameMD5, tt.pFrameMD5, tt.rawVideoMD5)
			}
		})
	}
}

func TestFFmpegRawVideoMD5OracleHigh12High14InterP16x16LumaChromaResidual(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}

	for _, tt := range highInterP16x16LumaChromaResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highInterP16x16LumaChromaResidualFixture(tt.bitDepth)
			assertHighInterP16x16LumaChromaResidualFixtureSyntax(t, data, tt.bitDepth)
			path := writeTempH264(t, data)
			pixFmt := fmt.Sprintf("yuv420p%dle", tt.bitDepth)

			framemd5 := exec.Command("ffmpeg",
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
				line := []byte(fmt.Sprintf("0, %10d, %10d,        1,      768, %s", i, i, want))
				if !bytes.Contains(framemd5Out, line) {
					t.Fatalf("missing %q in framemd5:\n%s", line, framemd5Out)
				}
			}

			rawvideo := exec.Command("ffmpeg",
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
			if len(raw) != 1536 {
				t.Fatalf("rawvideo size = %d, want 1536", len(raw))
			}
			sum := md5.Sum(raw)
			if got := hex.EncodeToString(sum[:]); got != tt.rawVideoMD5 {
				t.Fatalf("rawvideo md5 = %s, want %s", got, tt.rawVideoMD5)
			}
		})
	}
}

func TestHigh12High14InterP16x8LumaChromaResidualFixtureSyntax(t *testing.T) {
	for _, tt := range highInterP16x8LumaChromaResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highInterP16x8LumaChromaResidualFixture(tt.bitDepth)
			sum := md5.Sum(data)
			if got := hex.EncodeToString(sum[:]); got != tt.bitstreamMD5 {
				t.Fatalf("%s bitstream md5 = %s, want %s", tt.name, got, tt.bitstreamMD5)
			}
			assertHighInterP16x8LumaChromaResidualFixtureSyntax(t, data, tt.bitDepth)
		})
	}
}

func TestDecodeAnnexBHigh12High14InterP16x8LumaChromaResidualFrames(t *testing.T) {
	for _, tt := range highInterP16x8LumaChromaResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highInterP16x8LumaChromaResidualFixture(tt.bitDepth)
			assertHighInterP16x8LumaChromaResidualFixtureSyntax(t, data, tt.bitDepth)

			frames, err := NewDecoder().DecodeAnnexBFrames(data)
			if err != nil {
				t.Fatalf("decode %s Annex B: %v", tt.name, err)
			}
			assertHighInterP16x16ResidualFrames(t, frames, tt.bitDepth, tt.refFrameMD5, tt.pFrameMD5, tt.rawVideoMD5)
		})
	}
}

func TestDecodeAVCHigh12High14InterP16x8LumaChromaResidualFrames(t *testing.T) {
	for _, tt := range highInterP16x8LumaChromaResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highInterP16x8LumaChromaResidualFixture(tt.bitDepth)
			assertHighInterP16x8LumaChromaResidualFixtureSyntax(t, data, tt.bitDepth)

			for _, nalLengthSize := range []int{2, 3, 4} {
				frames, err := NewDecoder().DecodeAVCFrames(annexBToAVC(t, data, nalLengthSize), nalLengthSize)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: %v", nalLengthSize, err)
				}
				assertHighInterP16x16ResidualFrames(t, frames, tt.bitDepth, tt.refFrameMD5, tt.pFrameMD5, tt.rawVideoMD5)
			}
		})
	}
}

func TestDecodeAVCWithConfigurationRecordHigh12High14InterP16x8LumaChromaResidualFrames(t *testing.T) {
	for _, tt := range highInterP16x8LumaChromaResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highInterP16x8LumaChromaResidualFixture(tt.bitDepth)
			assertHighInterP16x8LumaChromaResidualFixtureSyntax(t, data, tt.bitDepth)

			for _, nalLengthSize := range []int{2, 3, 4} {
				config, packet := annexBToAVCConfigAndPacket(t, data, nalLengthSize)
				frames, err := NewDecoder().DecodeAVCFramesWithConfigurationRecord(config, packet)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: %v", nalLengthSize, err)
				}
				assertHighInterP16x16ResidualFrames(t, frames, tt.bitDepth, tt.refFrameMD5, tt.pFrameMD5, tt.rawVideoMD5)
			}
		})
	}
}

func TestFFmpegRawVideoMD5OracleHigh12High14InterP16x8LumaChromaResidual(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}

	for _, tt := range highInterP16x8LumaChromaResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highInterP16x8LumaChromaResidualFixture(tt.bitDepth)
			assertHighInterP16x8LumaChromaResidualFixtureSyntax(t, data, tt.bitDepth)
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
				line := []byte(fmt.Sprintf("0, %10d, %10d,        1,      768, %s", i, i, want))
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
			if len(raw) != 1536 {
				t.Fatalf("rawvideo size = %d, want 1536", len(raw))
			}
			sum := md5.Sum(raw)
			if got := hex.EncodeToString(sum[:]); got != tt.rawVideoMD5 {
				t.Fatalf("rawvideo md5 = %s, want %s", got, tt.rawVideoMD5)
			}
		})
	}
}

func TestHigh12High14InterP8x16LumaChromaResidualFixtureSyntax(t *testing.T) {
	for _, tt := range highInterP8x16LumaChromaResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highInterP8x16LumaChromaResidualFixture(tt.bitDepth)
			sum := md5.Sum(data)
			if got := hex.EncodeToString(sum[:]); got != tt.bitstreamMD5 {
				t.Fatalf("%s bitstream md5 = %s, want %s", tt.name, got, tt.bitstreamMD5)
			}
			assertHighInterP8x16LumaChromaResidualFixtureSyntax(t, data, tt.bitDepth)
		})
	}
}

func TestDecodeAnnexBHigh12High14InterP8x16LumaChromaResidualFrames(t *testing.T) {
	for _, tt := range highInterP8x16LumaChromaResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highInterP8x16LumaChromaResidualFixture(tt.bitDepth)
			assertHighInterP8x16LumaChromaResidualFixtureSyntax(t, data, tt.bitDepth)

			frames, err := NewDecoder().DecodeAnnexBFrames(data)
			if err != nil {
				t.Fatalf("decode %s Annex B: %v", tt.name, err)
			}
			assertHighInterP16x16ResidualFrames(t, frames, tt.bitDepth, tt.refFrameMD5, tt.pFrameMD5, tt.rawVideoMD5)
		})
	}
}

func TestDecodeAVCHigh12High14InterP8x16LumaChromaResidualFrames(t *testing.T) {
	for _, tt := range highInterP8x16LumaChromaResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highInterP8x16LumaChromaResidualFixture(tt.bitDepth)
			assertHighInterP8x16LumaChromaResidualFixtureSyntax(t, data, tt.bitDepth)

			for _, nalLengthSize := range []int{2, 3, 4} {
				frames, err := NewDecoder().DecodeAVCFrames(annexBToAVC(t, data, nalLengthSize), nalLengthSize)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: %v", nalLengthSize, err)
				}
				assertHighInterP16x16ResidualFrames(t, frames, tt.bitDepth, tt.refFrameMD5, tt.pFrameMD5, tt.rawVideoMD5)
			}
		})
	}
}

func TestDecodeAVCWithConfigurationRecordHigh12High14InterP8x16LumaChromaResidualFrames(t *testing.T) {
	for _, tt := range highInterP8x16LumaChromaResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highInterP8x16LumaChromaResidualFixture(tt.bitDepth)
			assertHighInterP8x16LumaChromaResidualFixtureSyntax(t, data, tt.bitDepth)

			for _, nalLengthSize := range []int{2, 3, 4} {
				config, packet := annexBToAVCConfigAndPacket(t, data, nalLengthSize)
				frames, err := NewDecoder().DecodeAVCFramesWithConfigurationRecord(config, packet)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: %v", nalLengthSize, err)
				}
				assertHighInterP16x16ResidualFrames(t, frames, tt.bitDepth, tt.refFrameMD5, tt.pFrameMD5, tt.rawVideoMD5)
			}
		})
	}
}

func TestFFmpegRawVideoMD5OracleHigh12High14InterP8x16LumaChromaResidual(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}

	for _, tt := range highInterP8x16LumaChromaResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highInterP8x16LumaChromaResidualFixture(tt.bitDepth)
			assertHighInterP8x16LumaChromaResidualFixtureSyntax(t, data, tt.bitDepth)
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
				line := []byte(fmt.Sprintf("0, %10d, %10d,        1,      768, %s", i, i, want))
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
			if len(raw) != 1536 {
				t.Fatalf("rawvideo size = %d, want 1536", len(raw))
			}
			sum := md5.Sum(raw)
			if got := hex.EncodeToString(sum[:]); got != tt.rawVideoMD5 {
				t.Fatalf("rawvideo md5 = %s, want %s", got, tt.rawVideoMD5)
			}
		})
	}
}

func TestHigh12High14InterP8x8LumaChromaResidualFixtureSyntax(t *testing.T) {
	for _, tt := range highInterP8x8LumaChromaResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highInterP8x8LumaChromaResidualFixture(tt.bitDepth)
			sum := md5.Sum(data)
			if got := hex.EncodeToString(sum[:]); got != tt.bitstreamMD5 {
				t.Fatalf("%s bitstream md5 = %s, want %s", tt.name, got, tt.bitstreamMD5)
			}
			assertHighInterP8x8LumaChromaResidualFixtureSyntax(t, data, tt.bitDepth)
		})
	}
}

func TestDecodeAnnexBHigh12High14InterP8x8LumaChromaResidualFrames(t *testing.T) {
	for _, tt := range highInterP8x8LumaChromaResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highInterP8x8LumaChromaResidualFixture(tt.bitDepth)
			assertHighInterP8x8LumaChromaResidualFixtureSyntax(t, data, tt.bitDepth)

			frames, err := NewDecoder().DecodeAnnexBFrames(data)
			if err != nil {
				t.Fatalf("decode %s Annex B: %v", tt.name, err)
			}
			assertHighInterP16x16ResidualFrames(t, frames, tt.bitDepth, tt.refFrameMD5, tt.pFrameMD5, tt.rawVideoMD5)
		})
	}
}

func TestDecodeAVCHigh12High14InterP8x8LumaChromaResidualFrames(t *testing.T) {
	for _, tt := range highInterP8x8LumaChromaResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highInterP8x8LumaChromaResidualFixture(tt.bitDepth)
			assertHighInterP8x8LumaChromaResidualFixtureSyntax(t, data, tt.bitDepth)

			for _, nalLengthSize := range []int{2, 3, 4} {
				frames, err := NewDecoder().DecodeAVCFrames(annexBToAVC(t, data, nalLengthSize), nalLengthSize)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: %v", nalLengthSize, err)
				}
				assertHighInterP16x16ResidualFrames(t, frames, tt.bitDepth, tt.refFrameMD5, tt.pFrameMD5, tt.rawVideoMD5)
			}
		})
	}
}

func TestDecodeAVCWithConfigurationRecordHigh12High14InterP8x8LumaChromaResidualFrames(t *testing.T) {
	for _, tt := range highInterP8x8LumaChromaResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highInterP8x8LumaChromaResidualFixture(tt.bitDepth)
			assertHighInterP8x8LumaChromaResidualFixtureSyntax(t, data, tt.bitDepth)

			for _, nalLengthSize := range []int{2, 3, 4} {
				config, packet := annexBToAVCConfigAndPacket(t, data, nalLengthSize)
				frames, err := NewDecoder().DecodeAVCFramesWithConfigurationRecord(config, packet)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: %v", nalLengthSize, err)
				}
				assertHighInterP16x16ResidualFrames(t, frames, tt.bitDepth, tt.refFrameMD5, tt.pFrameMD5, tt.rawVideoMD5)
			}
		})
	}
}

func TestFFmpegRawVideoMD5OracleHigh12High14InterP8x8LumaChromaResidual(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}

	for _, tt := range highInterP8x8LumaChromaResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highInterP8x8LumaChromaResidualFixture(tt.bitDepth)
			assertHighInterP8x8LumaChromaResidualFixtureSyntax(t, data, tt.bitDepth)
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
				line := []byte(fmt.Sprintf("0, %10d, %10d,        1,      768, %s", i, i, want))
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
			if len(raw) != 1536 {
				t.Fatalf("rawvideo size = %d, want 1536", len(raw))
			}
			sum := md5.Sum(raw)
			if got := hex.EncodeToString(sum[:]); got != tt.rawVideoMD5 {
				t.Fatalf("rawvideo md5 = %s, want %s", got, tt.rawVideoMD5)
			}
		})
	}
}

func readHigh12IntraPCMFixture(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile(high12IntraPCMFixturePath(t))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func high12IntraPCMFixturePath(t *testing.T) string {
	t.Helper()
	return filepath.Join("testdata", "h264", "high12_intrapcm_cavlc_i.h264")
}

func assertHigh12IntraPCMFrames(t *testing.T, frames []*Frame) {
	t.Helper()
	if len(frames) != 1 {
		t.Fatalf("frames = %d, want 1", len(frames))
	}
	frame := frames[0]
	if frame.Width != 16 || frame.Height != 16 ||
		frame.ChromaFormatIDC != 1 || frame.BitDepthLuma != 12 || frame.BitDepthChroma != 12 {
		t.Fatalf("frame format = %dx%d chroma %d depth %d/%d, want 16x16 yuv420p12le",
			frame.Width, frame.Height, frame.ChromaFormatIDC, frame.BitDepthLuma, frame.BitDepthChroma)
	}
	if pixFmt, err := frame.RawPixelFormat(); err != nil || pixFmt != "yuv420p12le" {
		t.Fatalf("RawPixelFormat = %q/%v, want yuv420p12le/nil", pixFmt, err)
	}
	if size, err := frame.RawYUVSize(); err != nil || size != 768 {
		t.Fatalf("RawYUVSize = %d/%v, want 768/nil", size, err)
	}
	raw, err := frame.AppendRawYUVBytesLE(nil)
	if err != nil {
		t.Fatalf("AppendRawYUVBytesLE: %v", err)
	}
	sum := md5.Sum(raw)
	if got := hex.EncodeToString(sum[:]); got != high12IntraPCMFrameMD5 {
		t.Fatalf("frame raw md5 = %s, want %s", got, high12IntraPCMFrameMD5)
	}
	if _, err := frame.AppendRawYUV(nil); err != ErrUnsupported {
		t.Fatalf("AppendRawYUV high12 error = %v, want ErrUnsupported", err)
	}
}

type highIntra16x16NoResidualCase struct {
	name         string
	bitDepth     int
	bitstreamMD5 string
	frameMD5     string
	rawVideoMD5  string
}

func highIntra16x16NoResidualCases() []highIntra16x16NoResidualCase {
	return []highIntra16x16NoResidualCase{
		{
			name:         "High12",
			bitDepth:     12,
			bitstreamMD5: high12Intra16x16NoResidualBitstreamMD5,
			frameMD5:     high12Intra16x16NoResidualFrameMD5,
			rawVideoMD5:  high12Intra16x16NoResidualRawVideoMD5,
		},
		{
			name:         "High14",
			bitDepth:     14,
			bitstreamMD5: high14Intra16x16NoResidualBitstreamMD5,
			frameMD5:     high14Intra16x16NoResidualFrameMD5,
			rawVideoMD5:  high14Intra16x16NoResidualRawVideoMD5,
		},
	}
}

type highIntra16x16ResidualCase struct {
	name         string
	bitDepth     int
	payloadBits  string
	bitstreamMD5 string
	frameMD5     string
	rawVideoMD5  string
}

type highInterNoResidualCase struct {
	name         string
	bitDepth     int
	payloadBits  string
	bitstreamMD5 string
	frameMD5     string
	rawVideoMD5  string
}

type highInterP16x16ResidualCase struct {
	name         string
	bitDepth     int
	bitstreamMD5 string
	refFrameMD5  string
	pFrameMD5    string
	rawVideoMD5  string
}

func highInterNoResidualCases() []highInterNoResidualCase {
	return []highInterNoResidualCase{
		{
			name:         "High12PSkip",
			bitDepth:     12,
			payloadBits:  highInterPSkipPayloadBits,
			bitstreamMD5: high12InterPSkipBitstreamMD5,
			frameMD5:     high12Intra16x16NoResidualFrameMD5,
			rawVideoMD5:  high12InterNoResidualRawVideoMD5,
		},
		{
			name:         "High12P16x16NoResidual",
			bitDepth:     12,
			payloadBits:  highInterP16x16NoResidualPayloadBits,
			bitstreamMD5: high12InterP16x16NoResidualBitstreamMD5,
			frameMD5:     high12Intra16x16NoResidualFrameMD5,
			rawVideoMD5:  high12InterNoResidualRawVideoMD5,
		},
		{
			name:         "High14PSkip",
			bitDepth:     14,
			payloadBits:  highInterPSkipPayloadBits,
			bitstreamMD5: high14InterPSkipBitstreamMD5,
			frameMD5:     high14Intra16x16NoResidualFrameMD5,
			rawVideoMD5:  high14InterNoResidualRawVideoMD5,
		},
		{
			name:         "High14P16x16NoResidual",
			bitDepth:     14,
			payloadBits:  highInterP16x16NoResidualPayloadBits,
			bitstreamMD5: high14InterP16x16NoResidualBitstreamMD5,
			frameMD5:     high14Intra16x16NoResidualFrameMD5,
			rawVideoMD5:  high14InterNoResidualRawVideoMD5,
		},
	}
}

func highInterP16x16ResidualCases() []highInterP16x16ResidualCase {
	return []highInterP16x16ResidualCase{
		{
			name:         "High12P16x16LumaResidual",
			bitDepth:     12,
			bitstreamMD5: high12InterP16x16LumaResidualBitstreamMD5,
			refFrameMD5:  high12Intra16x16NoResidualFrameMD5,
			pFrameMD5:    high12InterP16x16LumaResidualPFrameMD5,
			rawVideoMD5:  high12InterP16x16LumaResidualRawVideoMD5,
		},
		{
			name:         "High14P16x16LumaResidual",
			bitDepth:     14,
			bitstreamMD5: high14InterP16x16LumaResidualBitstreamMD5,
			refFrameMD5:  high14Intra16x16NoResidualFrameMD5,
			pFrameMD5:    high14InterP16x16LumaResidualPFrameMD5,
			rawVideoMD5:  high14InterP16x16LumaResidualRawVideoMD5,
		},
	}
}

func highInterP16x16LumaChromaResidualCases() []highInterP16x16ResidualCase {
	return []highInterP16x16ResidualCase{
		{
			name:         "High12P16x16LumaChromaResidual",
			bitDepth:     12,
			bitstreamMD5: high12InterP16x16LumaChromaResidualBitstreamMD5,
			refFrameMD5:  high12Intra16x16NoResidualFrameMD5,
			pFrameMD5:    high12InterP16x16LumaChromaResidualPFrameMD5,
			rawVideoMD5:  high12InterP16x16LumaChromaResidualRawVideoMD5,
		},
		{
			name:         "High14P16x16LumaChromaResidual",
			bitDepth:     14,
			bitstreamMD5: high14InterP16x16LumaChromaResidualBitstreamMD5,
			refFrameMD5:  high14Intra16x16NoResidualFrameMD5,
			pFrameMD5:    high14InterP16x16LumaChromaResidualPFrameMD5,
			rawVideoMD5:  high14InterP16x16LumaChromaResidualRawVideoMD5,
		},
	}
}

func highInterP16x8LumaChromaResidualCases() []highInterP16x16ResidualCase {
	return []highInterP16x16ResidualCase{
		{
			name:         "High12P16x8LumaChromaResidual",
			bitDepth:     12,
			bitstreamMD5: high12InterP16x8LumaChromaResidualBitstreamMD5,
			refFrameMD5:  high12Intra16x16NoResidualFrameMD5,
			pFrameMD5:    high12InterP16x8LumaChromaResidualPFrameMD5,
			rawVideoMD5:  high12InterP16x8LumaChromaResidualRawVideoMD5,
		},
		{
			name:         "High14P16x8LumaChromaResidual",
			bitDepth:     14,
			bitstreamMD5: high14InterP16x8LumaChromaResidualBitstreamMD5,
			refFrameMD5:  high14Intra16x16NoResidualFrameMD5,
			pFrameMD5:    high14InterP16x8LumaChromaResidualPFrameMD5,
			rawVideoMD5:  high14InterP16x8LumaChromaResidualRawVideoMD5,
		},
	}
}

func highInterP8x16LumaChromaResidualCases() []highInterP16x16ResidualCase {
	return []highInterP16x16ResidualCase{
		{
			name:         "High12P8x16LumaChromaResidual",
			bitDepth:     12,
			bitstreamMD5: high12InterP8x16LumaChromaResidualBitstreamMD5,
			refFrameMD5:  high12Intra16x16NoResidualFrameMD5,
			pFrameMD5:    high12InterP8x16LumaChromaResidualPFrameMD5,
			rawVideoMD5:  high12InterP8x16LumaChromaResidualRawVideoMD5,
		},
		{
			name:         "High14P8x16LumaChromaResidual",
			bitDepth:     14,
			bitstreamMD5: high14InterP8x16LumaChromaResidualBitstreamMD5,
			refFrameMD5:  high14Intra16x16NoResidualFrameMD5,
			pFrameMD5:    high14InterP8x16LumaChromaResidualPFrameMD5,
			rawVideoMD5:  high14InterP8x16LumaChromaResidualRawVideoMD5,
		},
	}
}

func highInterP8x8LumaChromaResidualCases() []highInterP16x16ResidualCase {
	return []highInterP16x16ResidualCase{
		{
			name:         "High12P8x8LumaChromaResidual",
			bitDepth:     12,
			bitstreamMD5: high12InterP8x8LumaChromaResidualBitstreamMD5,
			refFrameMD5:  high12Intra16x16NoResidualFrameMD5,
			pFrameMD5:    high12InterP8x8LumaChromaResidualPFrameMD5,
			rawVideoMD5:  high12InterP8x8LumaChromaResidualRawVideoMD5,
		},
		{
			name:         "High14P8x8LumaChromaResidual",
			bitDepth:     14,
			bitstreamMD5: high14InterP8x8LumaChromaResidualBitstreamMD5,
			refFrameMD5:  high14Intra16x16NoResidualFrameMD5,
			pFrameMD5:    high14InterP8x8LumaChromaResidualPFrameMD5,
			rawVideoMD5:  high14InterP8x8LumaChromaResidualRawVideoMD5,
		},
	}
}

func highIntra16x16ResidualCases() []highIntra16x16ResidualCase {
	return []highIntra16x16ResidualCase{
		{
			name:         "High12LumaDC",
			bitDepth:     12,
			payloadBits:  highIntra16x16LumaDCPayloadBits,
			bitstreamMD5: high12Intra16x16LumaDCBitstreamMD5,
			frameMD5:     high12Intra16x16LumaDCFrameMD5,
			rawVideoMD5:  high12Intra16x16LumaDCRawVideoMD5,
		},
		{
			name:         "High12ChromaDC",
			bitDepth:     12,
			payloadBits:  highIntra16x16ChromaDCPayloadBits,
			bitstreamMD5: high12Intra16x16ChromaDCBitstreamMD5,
			frameMD5:     high12Intra16x16ChromaDCFrameMD5,
			rawVideoMD5:  high12Intra16x16ChromaDCRawVideoMD5,
		},
		{
			name:         "High12ChromaAC",
			bitDepth:     12,
			payloadBits:  highIntra16x16ChromaACPayloadBits,
			bitstreamMD5: high12Intra16x16ChromaACBitstreamMD5,
			frameMD5:     high12Intra16x16ChromaACFrameMD5,
			rawVideoMD5:  high12Intra16x16ChromaACRawVideoMD5,
		},
		{
			name:         "High12ChromaDCAC",
			bitDepth:     12,
			payloadBits:  highIntra16x16ChromaDCACPayloadBits,
			bitstreamMD5: high12Intra16x16ChromaDCACBitstreamMD5,
			frameMD5:     high12Intra16x16ChromaDCACFrameMD5,
			rawVideoMD5:  high12Intra16x16ChromaDCACRawVideoMD5,
		},
		{
			name:         "High12LumaChroma",
			bitDepth:     12,
			payloadBits:  highIntra16x16LumaChromaPayloadBits,
			bitstreamMD5: high12Intra16x16LumaChromaBitstreamMD5,
			frameMD5:     high12Intra16x16LumaChromaFrameMD5,
			rawVideoMD5:  high12Intra16x16LumaChromaRawVideoMD5,
		},
		{
			name:         "High12LumaAC",
			bitDepth:     12,
			payloadBits:  highIntra16x16LumaACPayloadBits,
			bitstreamMD5: high12Intra16x16LumaACBitstreamMD5,
			frameMD5:     high12Intra16x16LumaACFrameMD5,
			rawVideoMD5:  high12Intra16x16LumaACRawVideoMD5,
		},
		{
			name:         "High12LumaDCAC",
			bitDepth:     12,
			payloadBits:  highIntra16x16LumaDCACPayloadBits,
			bitstreamMD5: high12Intra16x16LumaDCACBitstreamMD5,
			frameMD5:     high12Intra16x16LumaDCACFrameMD5,
			rawVideoMD5:  high12Intra16x16LumaDCACRawVideoMD5,
		},
		{
			name:         "High14LumaDC",
			bitDepth:     14,
			payloadBits:  highIntra16x16LumaDCPayloadBits,
			bitstreamMD5: high14Intra16x16LumaDCBitstreamMD5,
			frameMD5:     high14Intra16x16LumaDCFrameMD5,
			rawVideoMD5:  high14Intra16x16LumaDCRawVideoMD5,
		},
		{
			name:         "High14ChromaDC",
			bitDepth:     14,
			payloadBits:  highIntra16x16ChromaDCPayloadBits,
			bitstreamMD5: high14Intra16x16ChromaDCBitstreamMD5,
			frameMD5:     high14Intra16x16ChromaDCFrameMD5,
			rawVideoMD5:  high14Intra16x16ChromaDCRawVideoMD5,
		},
		{
			name:         "High14ChromaAC",
			bitDepth:     14,
			payloadBits:  highIntra16x16ChromaACPayloadBits,
			bitstreamMD5: high14Intra16x16ChromaACBitstreamMD5,
			frameMD5:     high14Intra16x16ChromaACFrameMD5,
			rawVideoMD5:  high14Intra16x16ChromaACRawVideoMD5,
		},
		{
			name:         "High14ChromaDCAC",
			bitDepth:     14,
			payloadBits:  highIntra16x16ChromaDCACPayloadBits,
			bitstreamMD5: high14Intra16x16ChromaDCACBitstreamMD5,
			frameMD5:     high14Intra16x16ChromaDCACFrameMD5,
			rawVideoMD5:  high14Intra16x16ChromaDCACRawVideoMD5,
		},
		{
			name:         "High14LumaChroma",
			bitDepth:     14,
			payloadBits:  highIntra16x16LumaChromaPayloadBits,
			bitstreamMD5: high14Intra16x16LumaChromaBitstreamMD5,
			frameMD5:     high14Intra16x16LumaChromaFrameMD5,
			rawVideoMD5:  high14Intra16x16LumaChromaRawVideoMD5,
		},
		{
			name:         "High14LumaAC",
			bitDepth:     14,
			payloadBits:  highIntra16x16LumaACPayloadBits,
			bitstreamMD5: high14Intra16x16LumaACBitstreamMD5,
			frameMD5:     high14Intra16x16LumaACFrameMD5,
			rawVideoMD5:  high14Intra16x16LumaACRawVideoMD5,
		},
		{
			name:         "High14LumaDCAC",
			bitDepth:     14,
			payloadBits:  highIntra16x16LumaDCACPayloadBits,
			bitstreamMD5: high14Intra16x16LumaDCACBitstreamMD5,
			frameMD5:     high14Intra16x16LumaDCACFrameMD5,
			rawVideoMD5:  high14Intra16x16LumaDCACRawVideoMD5,
		},
	}
}

func highIntra16x16NoResidualFixture(bitDepth int) []byte {
	return highIntra16x16ResidualFixture(bitDepth, highIntra16x16NoResidualPayloadBits)
}

func highIntra16x16ResidualFixture(bitDepth int, payloadBits string) []byte {
	return buildHighIntraAnnexBFixture(bitDepth, highIntra16x16ResidualSliceRBSP(payloadBits))
}

func highInterNoResidualFixture(bitDepth int, payloadBits string) []byte {
	return highInterFixture(bitDepth, payloadBits)
}

func highInterP16x16ResidualFixture(bitDepth int) []byte {
	return highInterFixture(bitDepth, highInterP16x16LumaResidualPayloadBits)
}

func highInterP16x16LumaChromaResidualFixture(bitDepth int) []byte {
	return highInterFixture(bitDepth, highInterP16x16LumaChromaResidualPayloadBits)
}

func highInterP16x8LumaChromaResidualFixture(bitDepth int) []byte {
	return highInterFixture(bitDepth, highInterP16x8LumaChromaResidualPayloadBits)
}

func highInterP8x16LumaChromaResidualFixture(bitDepth int) []byte {
	return highInterFixture(bitDepth, highInterP8x16LumaChromaResidualPayloadBits)
}

func highInterP8x8LumaChromaResidualFixture(bitDepth int) []byte {
	return highInterFixture(bitDepth, highInterP8x8LumaChromaResidualPayloadBits)
}

func highInterFixture(bitDepth int, payloadBits string) []byte {
	var data []byte
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALSPS), highInterSPSRBSP(bitDepth)))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALPPS), highIntraPCMPPSRBSP(bitDepth)))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALIDRSlice), highIntra16x16ResidualSliceRBSP(highIntra16x16NoResidualPayloadBits)))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALSlice), highInterNoResidualSliceRBSP(payloadBits)))
	return data
}

func assertHighIntra16x16Frames(t *testing.T, frames []*Frame, bitDepth int, wantMD5 string) {
	t.Helper()
	if len(frames) != 1 {
		t.Fatalf("frames = %d, want 1", len(frames))
	}
	frame := frames[0]
	if frame.Width != 16 || frame.Height != 16 ||
		frame.ChromaFormatIDC != 1 || frame.BitDepthLuma != bitDepth || frame.BitDepthChroma != bitDepth {
		t.Fatalf("frame format = %dx%d chroma %d depth %d/%d, want 16x16 yuv420p%dle",
			frame.Width, frame.Height, frame.ChromaFormatIDC, frame.BitDepthLuma, frame.BitDepthChroma, bitDepth)
	}
	pixFmt := fmt.Sprintf("yuv420p%dle", bitDepth)
	if got, err := frame.RawPixelFormat(); err != nil || got != pixFmt {
		t.Fatalf("RawPixelFormat = %q/%v, want %s/nil", got, err, pixFmt)
	}
	if size, err := frame.RawYUVSize(); err != nil || size != 768 {
		t.Fatalf("RawYUVSize = %d/%v, want 768/nil", size, err)
	}
	raw, err := frame.AppendRawYUVBytesLE(nil)
	if err != nil {
		t.Fatalf("AppendRawYUVBytesLE: %v", err)
	}
	sum := md5.Sum(raw)
	if got := hex.EncodeToString(sum[:]); got != wantMD5 {
		t.Fatalf("frame raw md5 = %s, want %s", got, wantMD5)
	}
	if _, err := frame.AppendRawYUV(nil); err != ErrUnsupported {
		t.Fatalf("AppendRawYUV high%d error = %v, want ErrUnsupported", bitDepth, err)
	}
}

func assertHighInterNoResidualFrames(t *testing.T, frames []*Frame, bitDepth int, wantMD5 string) {
	t.Helper()
	if len(frames) != 2 {
		t.Fatalf("frames = %d, want 2", len(frames))
	}
	for i, frame := range frames {
		if frame.Width != 16 || frame.Height != 16 ||
			frame.ChromaFormatIDC != 1 || frame.BitDepthLuma != bitDepth || frame.BitDepthChroma != bitDepth {
			t.Fatalf("frame[%d] format = %dx%d chroma %d depth %d/%d, want 16x16 yuv420p%dle",
				i, frame.Width, frame.Height, frame.ChromaFormatIDC, frame.BitDepthLuma, frame.BitDepthChroma, bitDepth)
		}
		pixFmt := fmt.Sprintf("yuv420p%dle", bitDepth)
		if got, err := frame.RawPixelFormat(); err != nil || got != pixFmt {
			t.Fatalf("frame[%d] RawPixelFormat = %q/%v, want %s/nil", i, got, err, pixFmt)
		}
		if size, err := frame.RawYUVSize(); err != nil || size != 768 {
			t.Fatalf("frame[%d] RawYUVSize = %d/%v, want 768/nil", i, size, err)
		}
		raw, err := frame.AppendRawYUVBytesLE(nil)
		if err != nil {
			t.Fatalf("frame[%d] AppendRawYUVBytesLE: %v", i, err)
		}
		sum := md5.Sum(raw)
		if got := hex.EncodeToString(sum[:]); got != wantMD5 {
			t.Fatalf("frame[%d] raw md5 = %s, want %s", i, got, wantMD5)
		}
		if _, err := frame.AppendRawYUV(nil); err != ErrUnsupported {
			t.Fatalf("frame[%d] AppendRawYUV high%d error = %v, want ErrUnsupported", i, bitDepth, err)
		}
	}
}

func assertHighInterP16x16ResidualFrames(t *testing.T, frames []*Frame, bitDepth int, refFrameMD5 string, pFrameMD5 string, rawVideoMD5 string) {
	t.Helper()
	if len(frames) != 2 {
		t.Fatalf("frames = %d, want 2", len(frames))
	}
	var rawVideo []byte
	for i, frame := range frames {
		if frame.Width != 16 || frame.Height != 16 ||
			frame.ChromaFormatIDC != 1 || frame.BitDepthLuma != bitDepth || frame.BitDepthChroma != bitDepth {
			t.Fatalf("frame[%d] format = %dx%d chroma %d depth %d/%d, want 16x16 yuv420p%dle",
				i, frame.Width, frame.Height, frame.ChromaFormatIDC, frame.BitDepthLuma, frame.BitDepthChroma, bitDepth)
		}
		pixFmt := fmt.Sprintf("yuv420p%dle", bitDepth)
		if got, err := frame.RawPixelFormat(); err != nil || got != pixFmt {
			t.Fatalf("frame[%d] RawPixelFormat = %q/%v, want %s/nil", i, got, err, pixFmt)
		}
		if size, err := frame.RawYUVSize(); err != nil || size != 768 {
			t.Fatalf("frame[%d] RawYUVSize = %d/%v, want 768/nil", i, size, err)
		}
		raw, err := frame.AppendRawYUVBytesLE(nil)
		if err != nil {
			t.Fatalf("frame[%d] AppendRawYUVBytesLE: %v", i, err)
		}
		rawVideo = append(rawVideo, raw...)
		want := refFrameMD5
		if i == 1 {
			want = pFrameMD5
		}
		sum := md5.Sum(raw)
		if got := hex.EncodeToString(sum[:]); got != want {
			t.Fatalf("frame[%d] raw md5 = %s, want %s", i, got, want)
		}
		if _, err := frame.AppendRawYUV(nil); err != ErrUnsupported {
			t.Fatalf("frame[%d] AppendRawYUV high%d error = %v, want ErrUnsupported", i, bitDepth, err)
		}
	}
	if len(rawVideo) != 1536 {
		t.Fatalf("rawvideo len = %d, want 1536", len(rawVideo))
	}
	sum := md5.Sum(rawVideo)
	if got := hex.EncodeToString(sum[:]); got != rawVideoMD5 {
		t.Fatalf("rawvideo md5 = %s, want %s", got, rawVideoMD5)
	}
}

func assertHighIntra16x16FixtureSyntax(t *testing.T, data []byte, bitDepth int) {
	t.Helper()
	nals, err := h264.SplitAnnexB(data)
	if err != nil {
		t.Fatal(err)
	}

	var spsList [32]*h264.SPS
	var ppsList [256]*h264.PPS
	var gotVCL []h264.NALUnitType
	for _, nal := range nals {
		switch nal.Type {
		case h264.NALSPS:
			sps, err := h264.DecodeSPS(nal.RBSP)
			if err != nil {
				t.Fatal(err)
			}
			if sps.ProfileIDC != 244 || sps.Width != 16 || sps.Height != 16 ||
				sps.ChromaFormatIDC != 1 || sps.BitDepthLuma != int32(bitDepth) || sps.BitDepthChroma != int32(bitDepth) {
				t.Fatalf("SPS profile/format = %d %dx%d chroma %d depth %d/%d, want High 4:4:4 Predictive-compatible 16x16 yuv420p%dle",
					sps.ProfileIDC, sps.Width, sps.Height, sps.ChromaFormatIDC, sps.BitDepthLuma, sps.BitDepthChroma, bitDepth)
			}
			spsList[sps.SPSID] = sps
		case h264.NALPPS:
			pps, err := h264.DecodePPS(nal.RBSP, &spsList)
			if err != nil {
				t.Fatal(err)
			}
			if pps.CABAC != 0 || pps.Transform8x8Mode != 0 || pps.DeblockingFilterParametersPresent == 0 {
				t.Fatalf("PPS CABAC/8x8/deblock-present = %d/%d/%d, want CAVLC/no-8x8/deblock params",
					pps.CABAC, pps.Transform8x8Mode, pps.DeblockingFilterParametersPresent)
			}
			ppsList[pps.PPSID] = pps
		case h264.NALIDRSlice, h264.NALSlice:
			sh, err := h264.ParseSliceHeader(nal, &ppsList)
			if err != nil {
				t.Fatal(err)
			}
			if sh.PictureStructure != h264.PictureFrame || sh.SliceTypeNoS != h264.PictureTypeI ||
				sh.DeblockingFilter != 0 || sh.QScale != 26 {
				t.Fatalf("slice picture/type/deblock/qp = %d/%d/%d/%d, want frame/I/disabled/26",
					sh.PictureStructure, sh.SliceTypeNoS, sh.DeblockingFilter, sh.QScale)
			}
			gotVCL = append(gotVCL, nal.Type)
		default:
			t.Fatalf("unexpected NAL type %d in High%d Intra16x16 no-residual fixture", nal.Type, bitDepth)
		}
	}
	if len(gotVCL) != 1 || gotVCL[0] != h264.NALIDRSlice {
		t.Fatalf("VCL NALs = %v, want one IDR slice", gotVCL)
	}
}

func assertHighInterNoResidualFixtureSyntax(t *testing.T, data []byte, bitDepth int) {
	t.Helper()
	nals, err := h264.SplitAnnexB(data)
	if err != nil {
		t.Fatal(err)
	}

	var spsList [32]*h264.SPS
	var ppsList [256]*h264.PPS
	var gotVCL []h264.NALUnitType
	var gotSliceTypes []int32
	for _, nal := range nals {
		switch nal.Type {
		case h264.NALSPS:
			sps, err := h264.DecodeSPS(nal.RBSP)
			if err != nil {
				t.Fatal(err)
			}
			if sps.ProfileIDC != 244 || sps.Width != 16 || sps.Height != 16 ||
				sps.ChromaFormatIDC != 1 || sps.BitDepthLuma != int32(bitDepth) || sps.BitDepthChroma != int32(bitDepth) ||
				sps.RefFrameCount != 1 {
				t.Fatalf("SPS profile/format/refs = %d %dx%d chroma %d depth %d/%d refs %d, want High 4:4:4 Predictive-compatible 16x16 yuv420p%dle refs 1",
					sps.ProfileIDC, sps.Width, sps.Height, sps.ChromaFormatIDC, sps.BitDepthLuma, sps.BitDepthChroma, sps.RefFrameCount, bitDepth)
			}
			spsList[sps.SPSID] = sps
		case h264.NALPPS:
			pps, err := h264.DecodePPS(nal.RBSP, &spsList)
			if err != nil {
				t.Fatal(err)
			}
			if pps.CABAC != 0 || pps.Transform8x8Mode != 0 || pps.DeblockingFilterParametersPresent == 0 ||
				pps.RefCount[0] != 1 || pps.RefCount[1] != 1 {
				t.Fatalf("PPS CABAC/8x8/deblock-present/refs = %d/%d/%d/%d/%d, want CAVLC/no-8x8/deblock params/1/1",
					pps.CABAC, pps.Transform8x8Mode, pps.DeblockingFilterParametersPresent, pps.RefCount[0], pps.RefCount[1])
			}
			ppsList[pps.PPSID] = pps
		case h264.NALIDRSlice, h264.NALSlice:
			sh, err := h264.ParseSliceHeader(nal, &ppsList)
			if err != nil {
				t.Fatal(err)
			}
			if sh.PictureStructure != h264.PictureFrame || sh.DeblockingFilter != 0 || sh.QScale != 26 {
				t.Fatalf("slice picture/deblock/qp = %d/%d/%d, want frame/disabled/26",
					sh.PictureStructure, sh.DeblockingFilter, sh.QScale)
			}
			gotVCL = append(gotVCL, nal.Type)
			gotSliceTypes = append(gotSliceTypes, sh.SliceTypeNoS)
		default:
			t.Fatalf("unexpected NAL type %d in High%d inter no-residual fixture", nal.Type, bitDepth)
		}
	}
	if len(gotVCL) != 2 || gotVCL[0] != h264.NALIDRSlice || gotVCL[1] != h264.NALSlice {
		t.Fatalf("VCL NALs = %v, want IDR slice followed by non-IDR slice", gotVCL)
	}
	if len(gotSliceTypes) != 2 || gotSliceTypes[0] != h264.PictureTypeI || gotSliceTypes[1] != h264.PictureTypeP {
		t.Fatalf("slice types = %v, want I then P", gotSliceTypes)
	}
}

func assertHighInterP16x16ResidualFixtureSyntax(t *testing.T, data []byte, bitDepth int) {
	t.Helper()
	nals, spsList, ppsList, gotSliceTypes := parseHighInterFixtureSyntax(t, data, bitDepth)
	if len(gotSliceTypes) != 2 || gotSliceTypes[0] != h264.PictureTypeI || gotSliceTypes[1] != h264.PictureTypeP {
		t.Fatalf("slice types = %v, want I then P", gotSliceTypes)
	}
	pmb := readHigh10ResidualCAVLCFirstPMacroblock(t, nals[1], spsList[0], ppsList[0])
	if pmb.skipRun != 0 || pmb.mbType != 0 || pmb.cbp != 1 {
		t.Fatalf("P macroblock skip/mb_type/cbp = %d/%d/%d (code %d), want P16x16 luma residual",
			pmb.skipRun, pmb.mbType, pmb.cbp, pmb.cbpCode)
	}
}

func assertHighInterP16x16LumaChromaResidualFixtureSyntax(t *testing.T, data []byte, bitDepth int) {
	t.Helper()
	nals, spsList, ppsList, gotSliceTypes := parseHighInterFixtureSyntax(t, data, bitDepth)
	if len(gotSliceTypes) != 2 || gotSliceTypes[0] != h264.PictureTypeI || gotSliceTypes[1] != h264.PictureTypeP {
		t.Fatalf("slice types = %v, want I then P", gotSliceTypes)
	}
	pmb := readHigh10ResidualCAVLCFirstPMacroblock(t, nals[1], spsList[0], ppsList[0])
	if pmb.skipRun != 0 || pmb.mbType != 0 || pmb.cbp != 33 {
		t.Fatalf("P macroblock skip/mb_type/cbp = %d/%d/%d (code %d), want P16x16 luma+chroma residual",
			pmb.skipRun, pmb.mbType, pmb.cbp, pmb.cbpCode)
	}
}

func assertHighInterP16x8LumaChromaResidualFixtureSyntax(t *testing.T, data []byte, bitDepth int) {
	t.Helper()
	nals, spsList, ppsList, gotSliceTypes := parseHighInterFixtureSyntax(t, data, bitDepth)
	if len(gotSliceTypes) != 2 || gotSliceTypes[0] != h264.PictureTypeI || gotSliceTypes[1] != h264.PictureTypeP {
		t.Fatalf("slice types = %v, want I then P", gotSliceTypes)
	}
	pmb := readHigh10ResidualCAVLCFirstPMacroblock(t, nals[1], spsList[0], ppsList[0])
	if pmb.skipRun != 0 || pmb.mbType != 1 || pmb.cbp != 33 {
		t.Fatalf("P macroblock skip/mb_type/cbp = %d/%d/%d (code %d), want P16x8 luma+chroma residual",
			pmb.skipRun, pmb.mbType, pmb.cbp, pmb.cbpCode)
	}
}

func assertHighInterP8x16LumaChromaResidualFixtureSyntax(t *testing.T, data []byte, bitDepth int) {
	t.Helper()
	nals, spsList, ppsList, gotSliceTypes := parseHighInterFixtureSyntax(t, data, bitDepth)
	if len(gotSliceTypes) != 2 || gotSliceTypes[0] != h264.PictureTypeI || gotSliceTypes[1] != h264.PictureTypeP {
		t.Fatalf("slice types = %v, want I then P", gotSliceTypes)
	}
	pmb := readHigh10ResidualCAVLCFirstPMacroblock(t, nals[1], spsList[0], ppsList[0])
	if pmb.skipRun != 0 || pmb.mbType != 2 || pmb.cbp != 33 {
		t.Fatalf("P macroblock skip/mb_type/cbp = %d/%d/%d (code %d), want P8x16 luma+chroma residual",
			pmb.skipRun, pmb.mbType, pmb.cbp, pmb.cbpCode)
	}
}

func assertHighInterP8x8LumaChromaResidualFixtureSyntax(t *testing.T, data []byte, bitDepth int) {
	t.Helper()
	nals, spsList, ppsList, gotSliceTypes := parseHighInterFixtureSyntax(t, data, bitDepth)
	if len(gotSliceTypes) != 2 || gotSliceTypes[0] != h264.PictureTypeI || gotSliceTypes[1] != h264.PictureTypeP {
		t.Fatalf("slice types = %v, want I then P", gotSliceTypes)
	}
	pmb := readHigh10ResidualCAVLCFirstPMacroblock(t, nals[1], spsList[0], ppsList[0])
	if pmb.skipRun != 0 || pmb.mbType != 3 || pmb.cbp != 33 {
		t.Fatalf("P macroblock skip/mb_type/cbp = %d/%d/%d (code %d), want P8x8 luma+chroma residual",
			pmb.skipRun, pmb.mbType, pmb.cbp, pmb.cbpCode)
	}
}

func parseHighInterFixtureSyntax(t *testing.T, data []byte, bitDepth int) ([]h264.NALUnit, [32]*h264.SPS, [256]*h264.PPS, []int32) {
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
			if sps.ProfileIDC != 244 || sps.Width != 16 || sps.Height != 16 ||
				sps.ChromaFormatIDC != 1 || sps.BitDepthLuma != int32(bitDepth) || sps.BitDepthChroma != int32(bitDepth) ||
				sps.RefFrameCount != 1 {
				t.Fatalf("SPS profile/format/refs = %d %dx%d chroma %d depth %d/%d refs %d, want High 4:4:4 Predictive-compatible 16x16 yuv420p%dle refs 1",
					sps.ProfileIDC, sps.Width, sps.Height, sps.ChromaFormatIDC, sps.BitDepthLuma, sps.BitDepthChroma, sps.RefFrameCount, bitDepth)
			}
			spsList[sps.SPSID] = sps
		case h264.NALPPS:
			pps, err := h264.DecodePPS(nal.RBSP, &spsList)
			if err != nil {
				t.Fatal(err)
			}
			if pps.CABAC != 0 || pps.Transform8x8Mode != 0 || pps.DeblockingFilterParametersPresent == 0 ||
				pps.RefCount[0] != 1 || pps.RefCount[1] != 1 {
				t.Fatalf("PPS CABAC/8x8/deblock-present/refs = %d/%d/%d/%d/%d, want CAVLC/no-8x8/deblock params/1/1",
					pps.CABAC, pps.Transform8x8Mode, pps.DeblockingFilterParametersPresent, pps.RefCount[0], pps.RefCount[1])
			}
			ppsList[pps.PPSID] = pps
		case h264.NALIDRSlice, h264.NALSlice:
			sh, err := h264.ParseSliceHeader(nal, &ppsList)
			if err != nil {
				t.Fatal(err)
			}
			if sh.PictureStructure != h264.PictureFrame || sh.DeblockingFilter != 0 || sh.QScale != 26 {
				t.Fatalf("slice picture/deblock/qp = %d/%d/%d, want frame/disabled/26",
					sh.PictureStructure, sh.DeblockingFilter, sh.QScale)
			}
			gotVCL = append(gotVCL, nal)
			gotSliceTypes = append(gotSliceTypes, sh.SliceTypeNoS)
		default:
			t.Fatalf("unexpected NAL type %d in High%d inter fixture", nal.Type, bitDepth)
		}
	}
	if len(gotVCL) != 2 || gotVCL[0].Type != h264.NALIDRSlice || gotVCL[1].Type != h264.NALSlice {
		gotTypes := make([]h264.NALUnitType, 0, len(gotVCL))
		for _, nal := range gotVCL {
			gotTypes = append(gotTypes, nal.Type)
		}
		t.Fatalf("VCL NALs = %v, want IDR slice followed by non-IDR slice", gotTypes)
	}
	return gotVCL, spsList, ppsList, gotSliceTypes
}

func buildHighIntraPCMAnnexBFixture(bitDepth int, seed int) []byte {
	return buildHighIntraAnnexBFixture(bitDepth, highIntraPCMSliceRBSP(bitDepth, seed))
}

func buildHighIntraAnnexBFixture(bitDepth int, sliceRBSP []byte) []byte {
	var data []byte
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALSPS), highIntraPCMSPSRBSP(bitDepth)))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALPPS), highIntraPCMPPSRBSP(bitDepth)))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALIDRSlice), sliceRBSP))
	return data
}

func highIntraPCMNAL(header byte, rbsp []byte) []byte {
	raw := []byte{header}
	return append(raw, escapeRBSPForNALPayload(rbsp)...)
}

func highIntraPCMSPSRBSP(bitDepth int) []byte {
	return highSPSRBSP(bitDepth, 0)
}

func highInterSPSRBSP(bitDepth int) []byte {
	return highSPSRBSP(bitDepth, 1)
}

func highSPSRBSP(bitDepth int, refFrameCount int) []byte {
	var b decoderSEIBitBuilder
	b.writeBits(244, 8) // High 4:4:4 Predictive profile admits 14-bit 4:2:0.
	b.writeBits(0, 8)
	b.writeBits(10, 8)
	b.writeUE(0)
	b.writeUE(1)
	b.writeUE(uint32(bitDepth - 8))
	b.writeUE(uint32(bitDepth - 8))
	b.writeBit(0)
	b.writeBit(0)
	b.writeUE(0)
	b.writeUE(2)
	b.writeUE(uint32(refFrameCount))
	b.writeBit(0)
	b.writeUE(0)
	b.writeUE(0)
	b.writeBit(1)
	b.writeBit(1)
	b.writeBit(0)
	b.writeBit(0)
	return b.rbsp()
}

func highIntraPCMPPSRBSP(bitDepth int) []byte {
	var b decoderSEIBitBuilder
	b.writeUE(0)
	b.writeUE(0)
	b.writeBit(0)
	b.writeBit(0)
	b.writeUE(0)
	b.writeUE(0)
	b.writeUE(0)
	b.writeBit(0)
	b.writeBits(0, 2)
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

func highIntraPCMSliceRBSP(bitDepth int, seed int) []byte {
	var b decoderSEIBitBuilder
	b.writeUE(0)
	b.writeUE(2)
	b.writeUE(0)
	b.writeBits(0, 4)
	b.writeUE(0)
	b.writeBit(0)
	b.writeBit(0)
	b.writeSE(0)
	b.writeUE(1)
	b.writeUE(25)
	highIntraPCMByteAlign(&b)

	rbsp := b.bytes()
	rbsp = append(rbsp, highIntraPCMBytes(bitDepth, seed)...)
	return append(rbsp, 0x80)
}

func highIntra16x16ResidualSliceRBSP(payloadBits string) []byte {
	var b decoderSEIBitBuilder
	b.writeUE(0)
	b.writeUE(2)
	b.writeUE(0)
	b.writeBits(0, 4)
	b.writeUE(0)
	b.writeBit(0)
	b.writeBit(0)
	b.writeSE(0)
	b.writeUE(1)
	highIntra16x16WritePayloadBits(&b, payloadBits)
	return b.rbsp()
}

func highInterNoResidualSliceRBSP(payloadBits string) []byte {
	var b decoderSEIBitBuilder
	b.writeUE(0)
	b.writeUE(0)
	b.writeUE(0)
	b.writeBits(1, 4)
	b.writeBit(0)
	b.writeBit(0)
	b.writeBit(0)
	b.writeSE(0)
	b.writeUE(1)
	highIntra16x16WritePayloadBits(&b, payloadBits)
	return b.rbsp()
}

func highIntra16x16WritePayloadBits(b *decoderSEIBitBuilder, payloadBits string) {
	for _, bit := range payloadBits {
		switch bit {
		case '0':
			b.writeBit(0)
		case '1':
			b.writeBit(1)
		}
	}
}

func highIntraPCMBytes(bitDepth int, seed int) []byte {
	var b decoderSEIBitBuilder
	maxSample := (1 << uint(bitDepth)) - 1
	for _, plane := range []struct {
		id      int
		samples int
	}{
		{id: 0, samples: 256},
		{id: 1, samples: 64},
		{id: 2, samples: 64},
	} {
		for i := 0; i < plane.samples; i++ {
			sample := (seed + plane.id*997 + i*73 + ((i & 7) << uint(bitDepth/2))) & maxSample
			b.writeBits(uint32(sample), uint32(bitDepth))
		}
	}
	highIntraPCMByteAlign(&b)
	return b.bytes()
}

func highIntraPCMByteAlign(b *decoderSEIBitBuilder) {
	for len(b.bits)&7 != 0 {
		b.writeBit(0)
	}
}

func assertHigh12IntraPCMFixtureSyntax(t *testing.T, data []byte) {
	t.Helper()
	nals, err := h264.SplitAnnexB(data)
	if err != nil {
		t.Fatal(err)
	}

	var spsList [32]*h264.SPS
	var ppsList [256]*h264.PPS
	var gotVCL []h264.NALUnitType
	for _, nal := range nals {
		switch nal.Type {
		case h264.NALSPS:
			sps, err := h264.DecodeSPS(nal.RBSP)
			if err != nil {
				t.Fatal(err)
			}
			if sps.ProfileIDC != 244 || sps.Width != 16 || sps.Height != 16 ||
				sps.ChromaFormatIDC != 1 || sps.BitDepthLuma != 12 || sps.BitDepthChroma != 12 {
				t.Fatalf("SPS profile/format = %d %dx%d chroma %d depth %d/%d, want High 4:4:4 Predictive-compatible 16x16 yuv420p12le",
					sps.ProfileIDC, sps.Width, sps.Height, sps.ChromaFormatIDC, sps.BitDepthLuma, sps.BitDepthChroma)
			}
			spsList[sps.SPSID] = sps
		case h264.NALPPS:
			pps, err := h264.DecodePPS(nal.RBSP, &spsList)
			if err != nil {
				t.Fatal(err)
			}
			if pps.CABAC != 0 || pps.DeblockingFilterParametersPresent == 0 {
				t.Fatalf("PPS CABAC/deblock-present = %d/%d, want CAVLC/deblock params", pps.CABAC, pps.DeblockingFilterParametersPresent)
			}
			ppsList[pps.PPSID] = pps
		case h264.NALIDRSlice, h264.NALSlice:
			sh, err := h264.ParseSliceHeader(nal, &ppsList)
			if err != nil {
				t.Fatal(err)
			}
			if sh.PictureStructure != h264.PictureFrame || sh.SliceTypeNoS != h264.PictureTypeI ||
				sh.DeblockingFilter != 0 || sh.QScale != 26 {
				t.Fatalf("slice picture/type/deblock/qp = %d/%d/%d/%d, want frame/I/disabled/26",
					sh.PictureStructure, sh.SliceTypeNoS, sh.DeblockingFilter, sh.QScale)
			}
			gotVCL = append(gotVCL, nal.Type)
		}
	}
	if len(gotVCL) != 1 || gotVCL[0] != h264.NALIDRSlice {
		t.Fatalf("VCL NALs = %v, want one IDR slice", gotVCL)
	}
}
