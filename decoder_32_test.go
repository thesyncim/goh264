// SPDX-License-Identifier: LGPL-2.1-or-later

package goh264

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"testing"
)

const testsrc32CAVLC8x8DCTAnnexBHex = `
000000016764000aacb44b6022000003000200000300041e244d400000000168ce0f2c8b0000010605ffff6ddc45e9bde6d948b7962cd820d923eeef78323634202d20636f7265203136352072333232322062333536303561202d20482e3236
342f4d5045472d342041564320636f646563202d20436f70796c65667420323030332d32303235202d20687474703a2f2f7777772e766964656f6c616e2e6f72672f783236342e68746d6c202d206f7074696f6e733a2063616261633d302072
65663d31206465626c6f636b3d313a313a3020616e616c7973653d3078333a3078313133206d653d686578207375626d653d37207073793d31207073795f72643d312e30303a302e3030206d697865645f7265663d30206d655f72616e67653d
3136206368726f6d615f6d653d31207472656c6c69733d31203878386463743d312063716d3d3020646561647a6f6e653d32312c313120666173745f70736b69703d31206368726f6d615f71705f6f66667365743d2d3220746872656164733d
31206c6f6f6b61686561645f746872656164733d3120736c696365645f746872656164733d30206e723d3020646563696d6174653d3120696e7465726c616365643d3020626c757261795f636f6d7061743d3020636f6e73747261696e65645f
696e7472613d3020626672616d65733d3020776569676874703d30206b6579696e743d323530206b6579696e745f6d696e3d313236207363656e656375743d3020696e7472615f726566726573683d302072635f6c6f6f6b61686561643d3430
2072633d637266206d62747265653d31206372663d32332e302071636f6d703d302e36302071706d696e3d302071706d61783d3639207170737465703d342069705f726174696f3d312e34302061713d313a312e303000800000016588843d68
844540181087e008a6c188c007d6a0c30007232f0a6a883b09504a24bffc07010180e641c0404021978c910662384d7c2ff847085e9c7e53a97c7ac47ff841fcec0887cc60f9cd71dffefd80ccad68554d0c8dc846d41a6146791e97f1eaf5b8
4b8e0ea72c774f57a400027c673114f07f5fdffb401e154beb5876affdf02000101c000100a00382a100400387078001ff19ce12258e3d4184b4bcd0793ab0029aa30323155752016a55ff9480ac2a0c565d7fe9000bfa0df1b0954ce59108e3
ad2ae3e87bf811fb633b80115520cfef61801ce24a4d91b7967fba89808fb5da9c3a08000411d900ba62158788ad5c0cc9814e712779bf2f9fffe18607010303994009417a29286114de17d6bffd6440d9028624ba22321eaf02230d1464b8b7
1ae3d5ed0033a1250465505e4bd5ffce011cc4b9e6b9075003f8a72d110ca6b713ae02478980ae2a0f15bff9c011cd439e539075680c9d2d1d5387242a0ffe080800080e14000208411000f05d8864027c42c828088b6d6f0109937bc3926d00
8ce11ae8e6022d29a6fff600d9055b7541769cd8fdf71a6331ef42821e8b7f0f0cb809e469f40453a2dc0dd0e42de501e2663380477194aba694d89a440018e10fb0d1e440c441fb3c000af10000a800080f44000100a000104390204016e110
002d703881005b81e0005ae1000405075c10000401478070020283ae0e000200a3c21f1ffd068a0000081ffff3c00040078800020000030021a1100020b22325c1104c078c97074130400218fb8200017000100860380218fb83800170001008
6200000001419a20f5f0fa24c005d188520f1d5a8e3c193fce462bf498419622cbd3b989f0182d4d98deaf7d7aab1f2ff7c0574b98ab3b9f9820001019000100f01e7c200020830a0007488110c0a8c684ace9b7c3d779aa193d82b0284ed4cd
b2713b0005c0788150816e7d094471f83973462b61dd4044b5d28a05fffb448000000001419a40b5f0c3006ddc628f659316076fa226f798de162237d024de871d99fc40700c381c0c2e0005c9f907251340a9b2c3afeb85ed801ada3ac27501
de76f783ffee970bad2b49d6a204020ec88422a02c00040160ada7544100c3b2f08013014404a320c0614869b958e393cf8b12fee61b4ff0609256837bfcfa84fbe6f4bff5bf0b048bd7654920c652efa004bf730799ebf213db1a9b3e000000
01419a6356f81491fbe164fe0e971205487c3c78153f80d0e52b909f0f6003368a1f9e17830be71ad22f80
`

const testsrc32CABAC8x8DCTAnnexBHex = `
000000016764000aacb44b6022000003000200000300041e244d400000000168ee0f2c8b0000010605ffff6ddc45e9bde6d948b7962cd820d923eeef78323634202d20636f7265203136352072333232322062333536303561202d20482e3236
342f4d5045472d342041564320636f646563202d20436f70796c65667420323030332d32303235202d20687474703a2f2f7777772e766964656f6c616e2e6f72672f783236342e68746d6c202d206f7074696f6e733a2063616261633d312072
65663d31206465626c6f636b3d313a313a3020616e616c7973653d3078333a3078313133206d653d686578207375626d653d37207073793d31207073795f72643d312e30303a302e3030206d697865645f7265663d30206d655f72616e67653d
3136206368726f6d615f6d653d31207472656c6c6973733d31203878386463743d312063716d3d3020646561647a6f6e653d32312c313120666173745f70736b69703d31206368726f6d615f71705f6f66667365743d2d3220746872656164733d
31206c6f6f6b61686561645f746872656164733d3120736c696365645f746872656164733d30206e723d3020646563696d6174653d3120696e7465726c616365643d3020626c757261795f636f6d7061743d3020636f6e73747261696e65645f
696e7472613d3020626672616d65733d3020776569676874703d30206b6579696e743d323530206b6579696e745f6d696e3d313236207363656e656375743d3020696e7472615f726566726573683d302072635f6c6f6f6b61686561643d3430
2072633d637266206d62747265653d31206372663d32332e302071636f6d703d302e36302071706d696e3d302071706d61783d3639207170737465703d342069705f726174696f3d312e34302061713d313a312e303000800000016588843d7f
702371e7171e5205c3b954e694d225df58d9c3105d279306e2b462038ddbd42708d722d1ead30da478e83caf40b06919d5ef55b78dd23fd7e4bf189af59222447e54bb0be46c8a3095dad417062f5e36d3a7a8f30044c6fdb4c40e908f213d1b
b91c6ab586d79e97f1c500680e94c4fcc5d4ca76db21669ed484e9b309167f645bc430c5a637a623cff2e22d82cfd907119c8fbe87f5384b5f18624e2c4fa417056bd1f10bf608f6f679d5d97adc25d097fc78a29d5ddc235fde564c6ac61bd8
1316d18f03d9669e07e7c846633375fca6a22bcf33d803d7e3a75c13a65bfa92f79abadf6f68b9a3cdfaaf361bdd68a1bd6926d3b79f59dc3fdcaf19dfec997296a84f413039176ddde9e12c3f9472b871be6a8b829c4208ab19aa5f783ca8e2
5119b70e505422f5f4f770e1251a78ec1f6d3ec1d19556d958ec21edd163d07ce8ab87df8c2a2c3603496a165502a4851fcc47d901d9760ecd4be9add30478340d781fde97998fb8a2b604333c82f43fe82c2389943a44af8d965f581175d641
4709c0195b0c14a5c77c00e2701c67cbdbda3b5b00192d0032f677f9b7f51cbea801cdff28ddf687ecf7b804cc763d62747e23c9dd389dcb284b869a34d835a2d9e58c1c8162fe32d658364aa2e56861a786fd6140e6d3bad9cb67f29afe1e87
1128a32d710246c520af2c8acbb4d3a08995af9a894807c04e31c328cbbafbbae9a9d17b5acc0100000001419a227aff636b7d7b5fcebb015f5ea1a6f7356ce12e2233ab3af16e86974557ae5672b618ae23d703d489efc415514ee87562834b
a7e3b8a1f9a0498d526300b21a1ee07afbc6ffff9083b870e404c48cd875ff27e188b1dff95d013f4e8f19ac30d01ba00af53795cfe70210e059c214d8e2e604e3c5189984e51cb09d24bccba3a2517b12beed65db7000000001419a425aff56
36e84a646972701c2be9668d4dcfb1f7f90c0968b0b3fca4d86136b10d53c3cfd98fc571f2d1b990e0a7b2d77ffc37d7fd6835ce11d79d0518f096a03fc485334f11f38871dff9db0ef4ac2c38ea4a6e3fffe6614e736ee9f323e3338e14590f
a090a8d2d63bfb205bcbc877b2b4ab88d5b589bb8bffa148cca6da323bd930ee673867cc3e96fbe8bb69231f432f93a294d0d505188fc6e8012b91a6d8626100000001419a63af51682d4e135eb5d1963db7697ff5d4db65c2c7dd1e0b0493f2
8e5c24862dff068fd8c0f05efbee85aa073ee71460
`

func TestDecodeAnnexBTestsrc32High8x8DCTFrames(t *testing.T) {
	for _, tt := range high8x8DCTFixtureCases() {
		t.Run(tt.name, func(t *testing.T) {
			frames, err := NewDecoder().DecodeAnnexBFrames(decodeHexFixture(t, tt.hex))
			if err != nil {
				t.Fatal(err)
			}
			assertHigh8x8DCTFrames(t, frames, tt.want)
		})
	}
}

func TestDecodeAVCTestsrc32High8x8DCTFrames(t *testing.T) {
	for _, tt := range high8x8DCTFixtureCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := decodeHexFixture(t, tt.hex)
			for _, nalLengthSize := range []int{2, 3, 4} {
				frames, err := NewDecoder().DecodeAVCFrames(annexBToAVC(t, data, nalLengthSize), nalLengthSize)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: %v", nalLengthSize, err)
				}
				assertHigh8x8DCTFrames(t, frames, tt.want)
			}
		})
	}
}

func TestDecodeConfiguredAVCTestsrc32High8x8DCTFrames(t *testing.T) {
	for _, tt := range high8x8DCTFixtureCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := decodeHexFixture(t, tt.hex)
			for _, nalLengthSize := range []int{2, 3, 4} {
				config, packet := annexBToAVCConfigAndPacket(t, data, nalLengthSize)
				frames, err := NewDecoder().DecodeAVCFramesWithConfigurationRecord(config, packet)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: %v", nalLengthSize, err)
				}
				assertHigh8x8DCTFrames(t, frames, tt.want)
			}
		})
	}
}

func TestDecodeConfiguredAVCTestsrc32High8x8DCTFramesAcrossSamples(t *testing.T) {
	for _, tt := range high8x8DCTFixtureCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := decodeHexFixture(t, tt.hex)
			for _, nalLengthSize := range []int{2, 3, 4} {
				config, samples := annexBToAVCConfigAndSamples(t, data, nalLengthSize)
				if len(samples) != len(tt.want) {
					t.Fatalf("nalLengthSize=%d: samples = %d, want %d", nalLengthSize, len(samples), len(tt.want))
				}

				dec := NewDecoder()
				if _, err := dec.ParseAVCDecoderConfigurationRecord(config); err != nil {
					t.Fatalf("nalLengthSize=%d: config: %v", nalLengthSize, err)
				}
				var frames []*Frame
				for i, sample := range samples {
					frame, err := dec.DecodeConfiguredAVC(sample)
					if err != nil {
						t.Fatalf("nalLengthSize=%d sample[%d]: %v", nalLengthSize, i, err)
					}
					frames = append(frames, frame)
				}
				assertHigh8x8DCTFrames(t, frames, tt.want)
			}
		})
	}
}

func TestDecodeAutoConfiguredAVCTestsrc32High8x8DCTFramesAcrossSamples(t *testing.T) {
	for _, tt := range high8x8DCTFixtureCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := decodeHexFixture(t, tt.hex)
			config, samples := annexBToAVCConfigAndSamples(t, data, 4)
			if len(samples) != len(tt.want) {
				t.Fatalf("samples = %d, want %d", len(samples), len(tt.want))
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
			assertHigh8x8DCTFrames(t, frames, tt.want)
		})
	}
}

func TestFFmpegFrameMD5OracleTestsrc32High8x8DCT(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}

	for _, tt := range high8x8DCTFixtureCases() {
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
				line := []byte(fmt.Sprintf("0, %10d, %10d,        1, %8d, %s", i, i, 1536, hash))
				if !bytes.Contains(out, line) {
					t.Fatalf("frame[%d] missing %q in framemd5:\n%s", i, line, out)
				}
			}
		})
	}
}

func high8x8DCTFixtureCases() []struct {
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
			hex:  testsrc32CAVLC8x8DCTAnnexBHex,
			want: []string{
				"4d912de8c22019c29a46f3966607408c",
				"11d6e207060405262de9a91bbdd298a9",
				"6bf6d4689852ae04c3c5f7da495e5e48",
				"559d2dfec6c93d5b03fd9f179f8216c4",
			},
		},
		{
			name: "cabac",
			hex:  testsrc32CABAC8x8DCTAnnexBHex,
			want: []string{
				"2f01a945ea8e10134c1c80077e62ca3f",
				"2dcdacc98ced800818b6fe09c2e7fa2b",
				"20e5d5b88002dcf514d3772316464476",
				"8ac7c3f6f20b7e002fdf895532a3fd9b",
			},
		},
	}
}

func assertHigh8x8DCTFrames(t *testing.T, frames []*Frame, want []string) {
	t.Helper()
	for i, frame := range frames {
		if frame.Width != 32 || frame.Height != 32 || frame.ChromaFormatIDC != 1 {
			t.Fatalf("frame[%d] metadata = %dx%d chroma %d", i, frame.Width, frame.Height, frame.ChromaFormatIDC)
		}
	}
	assertFrameMD5Strings(t, frames, want)
}
