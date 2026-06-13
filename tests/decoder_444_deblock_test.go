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
)

const testsrc32CAVLC444DeblockAnnexBHex = `
0000000167f4000a9196896c044000000300400000030083c489a80000000168ce0f1121100000010605ffff6ddc45e9bde6d948b7962cd820d923eeef78323634202d20636f7265203136352072333232322062333536303561202d20482e3236342f4d5045472d342041564320636f646563202d20436f
70796c65667420323030332d32303235202d20687474703a2f2f7777772e766964656f6c616e2e6f72672f783236342e68746d6c202d206f7074696f6e733a2063616261633d30207265663d31206465626c6f636b3d313a303a3020616e616c7973653d3078333a3078313333206d653d756d6820737562
6d653d3130207073793d31207073795f72643d312e30303a302e3030206d697865645f7265663d30206d655f72616e67653d3234206368726f6d615f6d653d31207472656c6c69733d32203878386463743d312063716d3d3020646561647a6f6e653d32312c313120666173745f70736b69703d31206368
726f6d615f71705f6f66667365743d3420746872656164733d31206c6f6f6b61686561645f746872656164733d3120736c696365645f746872656164733d30206e723d3020646563696d6174653d3120696e7465726c616365643d3020626c757261795f636f6d7061743d3020636f6e73747261696e6564
5f696e7472613d3020626672616d65733d3020776569676874703d30206b6579696e743d323530206b6579696e745f6d696e3d313236207363656e656375743d3020696e7472615f726566726573683d302072635f6c6f6f6b61686561643d36302072633d637266206d62747265653d31206372663d3233
2e302071636f6d703d302e36302071706d696e3d302071706d61783d3639207170737465703d342069705f726174696f3d312e34302061713d313a312e303000800000016588843fa207158411c4b200229b0c840001003ad459645940004a0bd1494308a6f0beb5ffe0e00080080a641c1060e65e6a4323
289ab84ee19c44f0cec4129385a42bffc77db85f1c1402bdb7943d5ec0c83274b4544d1da2bffdf43d0a1419928512fd6e26830e8524464f57a4000798c66c70055bfbff68078a42a7280c8dfef80380908002712ce692ee0ddbf3cc4bc098e2869500f000f1e7762bfbc79b82500dcf46fd418e0c172801
803833c1e400ddc1f1d64246f2f87604e7a42503cce440dd294e007a5700752005cd43771ecad5b78e91c191050fd237402a037c38cf1dda6dbd809c099c081269facc02afe5c6db2c277ba4201849e270813c88d00018f664800665a5454047f711ca1d040002086c8056988561e706dab8624c3b9c59de
4b5387fff83060c0e020602192801282f45250c229bc2fad7ffac881b2050c497444643d5e04461a28cb716e3483d5ea000d431442b0b0168a7fdf1c02398973cd720ea007f14e5a22194d6e26dc25c28be21d24247abce008e6a1cf31e83aa801e158d45b415c5bfbe3b58f81b400a8f193117a59fcb580
41da9d1830045a9d4062c09879140e69420d628a2646871d5fb800a7c3168fee178fc84daaff144b88030e29a6e9cbecea80397b8251803407800882487147dd4cf1c487ba3c79d50b003e003e3d9908ec806a3489a076064738e7b20e30f01cfc6f2686f4a8e7c10fe8c009d6ea9c9a1990056316b88002
fc21f3964418883f9e0004f7c8824a43c7f50d77e42f8ed21980083cca3ee156a343df35cf1c3c3ff9ca7800081afbfcf0003de1bb3e300010181dc481353ff6abe0df9f4cc3c62113a0f1f69a60ca569400020220df3000000001419a20bfc51070f07d320c42cead47ed6c993f03fe3755a36f5b54c8ed
b0df3cce3df1cd9e1a37d0237b2c8c67e601c008ad01fc364a00038000f1e6e56000c0077e3f8c3a289ad3de973b1a76df5fbf397f0da8f1b1b74d84d464071b00e9a2e01c16600dcda402500898f5200081a83998a82ff1f7c783ead8cd4b12fe03b197140a2e3ce82f648000000001419a40bfc6bf0069
6e326f4c57a32b4143aa2287c6de28246fa0416e38c847e6e4d842012d28c445e2121968c8078f807918fbd41817c41ec4fc54175c8079f1ae203ae3c4c2ec8ada02c001a278a02ecbc1edae2c16c1c45c9800085007535c58499d1fb1d2ea1df4ff38229bd95a41a0f7c8bceaff87bd8f2f41efc0000000
01419a63dbe070f9c2432f63a5c481521f03c78153cebc4ac362b6f3f83897140b05d171605f80
`

const testsrc32CABAC444DeblockAnnexBHex = `
0000000167f4000a9196896c044000000300400000030083c489a80000000168ee0f1121100000010605ffff6ddc45e9bde6d948b7962cd820d923eeef78323634202d20636f7265203136352072333232322062333536303561202d20482e3236342f4d5045472d342041564320636f646563202d20436f
70796c65667420323030332d32303235202d20687474703a2f2f7777772e766964656f6c616e2e6f72672f783236342e68746d6c202d206f7074696f6e733a2063616261633d31207265663d31206465626c6f636b3d313a303a3020616e616c7973653d3078333a3078313333206d653d756d6820737562
6d653d3130207073793d31207073795f72643d312e30303a302e3030206d697865645f7265663d30206d655f72616e67653d3234206368726f6d615f6d653d31207472656c6c69733d32203878386463743d312063716d3d3020646561647a6f6e653d32312c313120666173745f70736b69703d31206368
726f6d615f71705f6f66667365743d3420746872656164733d31206c6f6f6b61686561645f746872656164733d3120736c696365645f746872656164733d30206e723d3020646563696d6174653d3120696e7465726c616365643d3020626c757261795f636f6d7061743d3020636f6e73747261696e6564
5f696e7472613d3020626672616d65733d3020776569676874703d30206b6579696e743d323530206b6579696e745f6d696e3d313236207363656e656375743d3020696e7472615f726566726573683d302072635f6c6f6f6b61686561643d36302072633d637266206d62747265653d31206372663d3233
2e302071636f6d703d302e36302071706d696e3d302071706d61783d3639207170737465703d342069705f726174696f3d312e34302061713d313a312e303000800000016588843f7017047a28ae8ec040afa5a85192ab583abf49e11342258fa158af11376d000087272d2a304ce14a328636f524916670
1ca7a6a43ce26e1845a508678efad2bd5d415eb95910cd9425c96b5aff3be932307f0ecc5cda83fd4fc15b8ae43e68a709ff3edf1ba4464bda3be9317ee62a6902838a0c6f5e14e91d0db47b9b19a6bed72e84f2c172ff0f7872e49e2d2b5f47f324d511234b004a8589b6a796201096db855964a09b3eee
5f33c1e1084f4573fc6967069e2eafdc982066c12a9617246f78ac09d7c448dcf49958f06e70e7f75189efb0f812a9c90bd1f134d5d629b34f99c03b4f72267e00d9c2d278c045c1c62e479e7e171a479df20a4cd26ab2539befcf5caaeed802990d2f7db161163c36153f3be415e313fb0022f9f8d829e2
c5a84a63fa208d18617c996a0ce6d88d0f2e58fc249cefb94f745885efc437647b8d88b4fc9f3a4552302d84d9d9b082b99b7e631a05a9d6728b2a55bdd85383fd62e0aacecc8c6ae4aad9870d94bfca40176bdbb0e6ce554ab73295d362ced221f1abc68cd32fec44907b683751d14e67748ef7ca22c7a9
113d9907d15c7cf9ba771d34729bf46f3cdfd7e644d4bb8358caf3b2ccacddf311e3164970eb8e31e99595b01ef74aa072c0017e3f3dad5a11bfe37c663e63ebedffc9033c6608363f8ae6869e61104f5b902c9f8deef1ef6018ee4cc6c3d634206c39d5d06c9d61bc5c9e1c6e881f5768e7432b24140251
eb0f1554d5e04b0bfffb7ce7d969bb72c80f35e0b2619a173f69147efba72de12e885873a239635c4b267ac27e51fc3a49907c8ae54ff16c3007e100000001419a225f4f55b128ea34be0a60b2a7b9834e474ce4f3c16870fd56514ddddd7ca96e8c48301699bee44aafb5c576048fedbe113b9bece25a9c
d1a4f09d423185fc0d01ff95a87c1a09e4c72734b437604dbfea46570a2a99a2f01f02c58160a150706d6fcadc0b33a54eca7cd3c5010528a3d2a3fdcb32c548f3ffaf1f719f614d576502c1896784dd8bac00000001419a425f561d6c04a07f839a4b3ae69ef1245fa67721d5bc7114ee6d17bf2a55fab8
35548a1f55cd1da3089c8b03af9d08f6b960f382359686d69539dc813ffc59f75e240bba9df86e10c79d6e9b690ccce46a7981ddca6d5b0f1325444e42e7173fc1056dd3d4a7f1c3a91c9366df00000001419a63ff516892ec0ea11cf09dedee7315d4dbb942afc84471e8aa146b9f7946fcdd988d7c07e0
`

func TestDecodeAnnexBTestsrc32High444DeblockFrames(t *testing.T) {
	for _, tt := range high444DeblockFixtureCases() {
		t.Run(tt.name, func(t *testing.T) {
			frames, err := NewDecoder().DecodeAnnexBFrames(decodeHexFixture(t, tt.hex))
			if err != nil {
				t.Fatal(err)
			}
			assertHigh444FrameMD5Format(t, frames, 32, 32, 3072, tt.want)
		})
	}
}

func TestDecodeAVCTestsrc32High444DeblockFrames(t *testing.T) {
	for _, tt := range high444DeblockFixtureCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := decodeHexFixture(t, tt.hex)
			for _, nalLengthSize := range []int{2, 3, 4} {
				frames, err := NewDecoder().DecodeAVCFrames(annexBToAVC(t, data, nalLengthSize), nalLengthSize)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: %v", nalLengthSize, err)
				}
				assertHigh444FrameMD5Format(t, frames, 32, 32, 3072, tt.want)
			}
		})
	}
}

func TestDecodeAVCWithConfigurationRecordHigh444DeblockFrames(t *testing.T) {
	for _, tt := range high444DeblockFixtureCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := decodeHexFixture(t, tt.hex)
			config, packet := annexBToAVCConfigAndPacket(t, data, 4)
			frames, err := NewDecoder().DecodeAVCFramesWithConfigurationRecord(config, packet)
			if err != nil {
				t.Fatal(err)
			}
			assertHigh444FrameMD5Format(t, frames, 32, 32, 3072, tt.want)
		})
	}
}

func TestDecodeConfiguredAVCAcrossSamplesHigh444DeblockFrames(t *testing.T) {
	for _, tt := range high444DeblockFixtureCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := decodeHexFixture(t, tt.hex)
			config, samples := annexBToAVCConfigAndSamples(t, data, 4)
			dec := NewDecoder()
			if _, err := dec.ConfigureAVCDecoderConfigurationRecord(config); err != nil {
				t.Fatal(err)
			}
			for i, sample := range samples {
				frame, err := dec.DecodeConfiguredAVC(sample)
				if err != nil {
					t.Fatalf("sample[%d]: %v", i, err)
				}
				assertHigh444FrameMD5Format(t, []*Frame{frame}, 32, 32, 3072, tt.want[i:i+1])
			}
		})
	}
}

func TestFFmpegFrameMD5OracleTestsrc32High444Deblock(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}

	for _, tt := range high444DeblockFixtureCases() {
		t.Run(tt.name, func(t *testing.T) {
			path := writeTempH264(t, decodeHexFixture(t, tt.hex))
			cmd := exec.Command("ffmpeg",
				"-v", "error",
				"-f", "h264",
				"-i", path,
				"-an", "-sn", "-dn",
				"-f", "framemd5",
				"-",
			)
			out, err := cmd.Output()
			if err != nil {
				t.Fatalf("ffmpeg framemd5: %v", err)
			}
			for i, hash := range tt.want {
				line := []byte(fmt.Sprintf("0, %10d, %10d,        1,     3072, %s", i, i, hash))
				if !bytes.Contains(out, line) {
					t.Fatalf("frame[%d] missing %q in framemd5:\n%s", i, line, out)
				}
			}
		})
	}
}

func high444DeblockFixtureCases() []struct {
	name string
	hex  string
	want []string
} {
	return []struct {
		name string
		hex  string
		want []string
	}{
		{
			name: "cavlc",
			hex:  testsrc32CAVLC444DeblockAnnexBHex,
			want: []string{
				"e6522cb7daa4278fa238f995daea8594",
				"274c8ec306ee4705f93c3cc6bdedc948",
				"d42015040093bf782173b1d8d00a5b74",
				"9d93f36ffaeb8caa764f2b06240ba5d7",
			},
		},
		{
			name: "cabac",
			hex:  testsrc32CABAC444DeblockAnnexBHex,
			want: []string{
				"df7f5b803f967fcd46070b2b182c3805",
				"5bc16fb5ebe5c3021e77c7c82c34127c",
				"5e0f2020cfefc09d993a68c2963ad8ed",
				"f14846abbb44addf3e1ce0e66394b683",
			},
		},
	}
}

func assertHigh444FrameMD5Format(t *testing.T, frames []*Frame, width int, height int, rawSize int, want []string) {
	t.Helper()
	if len(frames) != len(want) {
		t.Fatalf("frames = %d, want %d", len(frames), len(want))
	}
	for i, frame := range frames {
		if frame.Width != width || frame.Height != height || frame.ChromaFormatIDC != 3 || frame.BitDepthLuma != 8 || frame.BitDepthChroma != 8 {
			t.Fatalf("frame[%d] metadata = %dx%d chroma %d depth %d/%d", i, frame.Width, frame.Height, frame.ChromaFormatIDC, frame.BitDepthLuma, frame.BitDepthChroma)
		}
		raw, err := frame.AppendRawYUV(nil)
		if err != nil {
			t.Fatalf("frame[%d] raw yuv: %v", i, err)
		}
		if len(raw) != rawSize {
			t.Fatalf("frame[%d] raw frame size = %d, want %d", i, len(raw), rawSize)
		}
		got := md5.Sum(raw)
		if hex.EncodeToString(got[:]) != want[i] {
			t.Fatalf("frame[%d] md5 = %x, want %s", i, got, want[i])
		}
	}
}
