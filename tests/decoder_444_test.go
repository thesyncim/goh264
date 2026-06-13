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

const testsrc16CAVLC444AnnexBHex = `
0000000167f4000a91969ec044000003000400000300083c489a800000000168ce0f1121100000010605ffff6ddc45e9bde6d948b7962cd820d923eeef78323634202d20636f7265203136352072333232322062333536303561202d20482e3236342f4d5045472d342041564320636f646563202d20436f
70796c65667420323030332d32303235202d20687474703a2f2f7777772e766964656f6c616e2e6f72672f783236342e68746d6c202d206f7074696f6e733a2063616261633d30207265663d31206465626c6f636b3d303a303a3020616e616c7973653d3078333a3078313333206d653d756d6820737562
6d653d3130207073793d31207073795f72643d312e30303a302e3030206d697865645f7265663d30206d655f72616e67653d3234206368726f6d615f6d653d31207472656c6c69733d32203878386463743d312063716d3d3020646561647a6f6e653d32312c313120666173745f70736b69703d31206368
726f6d615f71705f6f66667365743d3420746872656164733d31206c6f6f6b61686561645f746872656164733d3120736c696365645f746872656164733d30206e723d3020646563696d6174653d3120696e7465726c616365643d3020626c757261795f636f6d7061743d3020636f6e73747261696e6564
5f696e7472613d3020626672616d65733d3020776569676874703d30206b6579696e743d323530206b6579696e745f6d696e3d313236207363656e656375743d3020696e7472615f726566726573683d302072635f6c6f6f6b61686561643d36302072633d637266206d62747265653d31206372663d3233
2e302071636f6d703d302e36302071706d696e3d302071706d61783d3639207170737465703d342069705f726174696f3d312e34302061713d313a312e303000800000016588843aa20e22a802092f0045360c46003eb5061800391978535441d84a825125ffe03c0c1d900ba0361410b862b578c5905731
a7793be9ecd104a393cc2b9fc13b2ddfe107db84b8e0ea72c7050f57b0220c9d2d1d5874f8553feb2206c8143125d11190f57bd96237d0e074458f404a800350c510ac2c05a29ff7cd601a0ee6a2de0e8cb15f803802630001bb6c8335bf3c941a262dc1585506103201526f24b7fe0e2c0481e4414d3890
00c002833c06cc705700d9c06c3cc38fa84044b8920a001c9b270002d6b0943800056d536007488005d929515731b47c92203f10b25fc6f100170022e3d75afdb681e673a338100934de3302473a99fd0ac14985a49100cd228a4c005df910000afc1d000c51138800000001419a20abc50c27001740f1c0
11608ad477f9fbdecfc04c9190eb57ec7be0380115a04f0ef28000402e001e3d14e000603b8046cddef4b9fde38b73f32093235ffc014666408091e7590c75931e3b00084a5e5ecb9d0f8dffe000000001419a41af1450190f9a9411615352b2f7beee02ecd72e3d6e4868f5ab755b1ec16be2c022e1617e
60009fd5f8adf265a4bc35232ebf021fc1e895b0f3bd1577dae7552fbfe4f185e000000001419a612f47c085e4f970c8f70c8f40
`

const testsrc16CABAC444AnnexBHex = `
0000000167f4000a91969ec044000003000400000300083c489a800000000168ee0f1121100000010605ffff6ddc45e9bde6d948b7962cd820d923eeef78323634202d20636f7265203136352072333232322062333536303561202d20482e3236342f4d5045472d342041564320636f646563202d20436f
70796c65667420323030332d32303235202d20687474703a2f2f7777772e766964656f6c616e2e6f72672f783236342e68746d6c202d206f7074696f6e733a2063616261633d31207265663d31206465626c6f636b3d303a303a3020616e616c7973653d3078333a3078313333206d653d756d6820737562
6d653d3130207073793d31207073795f72643d312e30303a302e3030206d697865645f7265663d30206d655f72616e67653d3234206368726f6d615f6d653d31207472656c6c69733d32203878386463743d312063716d3d3020646561647a6f6e653d32312c313120666173745f70736b69703d31206368
726f6d615f71705f6f66667365743d3420746872656164733d31206c6f6f6b61686561645f746872656164733d3120736c696365645f746872656164733d30206e723d3020646563696d6174653d3120696e7465726c616365643d3020626c757261795f636f6d7061743d3020636f6e73747261696e6564
5f696e7472613d3020626672616d65733d3020776569676874703d30206b6579696e743d323530206b6579696e745f6d696e3d313236207363656e656375743d3020696e7472615f726566726573683d302072635f6c6f6f6b61686561643d36302072633d637266206d62747265653d31206372663d3233
2e302071636f6d703d302e36302071706d696e3d302071706d61783d3639207170737465703d342069705f726174696f3d312e34302061713d313a312e303000800000016588843a6ff71043ad62e8dfc3b956a932556b0757e93c226844b1f42b15e226e7275ba2cfc5be88e83bccd10dcb8a54b81c8133
ff68d254424963fda0ef8865d04eff620ca05d49750546da8b9d2f6ffe473f307c7be6f688411ed5b748a8ca48719f8eabec3b974c5177434b423c527ff32b1b8f6a8749feaeb161c121d0db47b7f8000dab2dda76a099efc70a44158e1665432802272ae36b61e12afd74efadabfe5e2ce9707e9b3b4684
43fd5c25c31ce00ea8673fceb3eb16f0d66767cd86607418d78cfdd63535f1927840532fb2c50045eefaba0fec658b7292a80e01edc81b4d2da7b812994ad95e5e7b21df8a19358f71ecfbf710c3e100000001419a22554f65a4b61851e90e447c93df8fede8faacbc8f9115c2423e5c7466b8423ffc1e39
63d72bf9c4b683e4acd4f20e1ce3fd0ac4c4050fb0963f61e5d032258c6c0c3585499fcaf04eb061afddc3c0fc2feee18cf65ee041ff26c5eeec9fb89d0f675c11bf00000001419a42d7d14325e7ef7449db2c9d75fe66b111e4625cfbb589c5653d1d1c6cfcf8187506f1cbebe95d4dd66957aea213683b
05c3ec7d65a2c6c11c9ab6c5c1c00b9cb7f15e2a3922978100000001419a6297b7f944960d1cb124ecfd
`

func TestDecodeAnnexBTestsrc16High444Frames(t *testing.T) {
	for _, tt := range high444FixtureCases() {
		t.Run(tt.name, func(t *testing.T) {
			frames, err := NewDecoder().DecodeAnnexBFrames(decodeHexFixture(t, tt.hex))
			if err != nil {
				t.Fatal(err)
			}
			assertHigh444FrameMD5Strings(t, frames, tt.want)
		})
	}
}

func TestDecodeAVCTestsrc16High444Frames(t *testing.T) {
	for _, tt := range high444FixtureCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := decodeHexFixture(t, tt.hex)
			for _, nalLengthSize := range []int{2, 3, 4} {
				frames, err := NewDecoder().DecodeAVCFrames(annexBToAVC(t, data, nalLengthSize), nalLengthSize)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: %v", nalLengthSize, err)
				}
				assertHigh444FrameMD5Strings(t, frames, tt.want)
			}
		})
	}
}

func TestDecodeAVCCHigh444Frames(t *testing.T) {
	for _, tt := range high444FixtureCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := decodeHexFixture(t, tt.hex)
			config, packet := annexBToAVCConfigAndPacket(t, data, 4)
			frames, err := NewDecoder().DecodeAVCCFrames(config, packet)
			if err != nil {
				t.Fatal(err)
			}
			assertHigh444FrameMD5Strings(t, frames, tt.want)
		})
	}
}

func TestDecodeConfiguredAVCAcrossSamplesHigh444Frames(t *testing.T) {
	for _, tt := range high444FixtureCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := decodeHexFixture(t, tt.hex)
			config, samples := annexBToAVCConfigAndSamples(t, data, 4)
			dec := NewDecoder()
			if _, err := dec.ConfigureAVCC(config); err != nil {
				t.Fatal(err)
			}
			for i, sample := range samples {
				frame, err := dec.DecodeConfiguredAVC(sample)
				if err != nil {
					t.Fatalf("sample[%d]: %v", i, err)
				}
				assertHigh444FrameMD5Strings(t, []*Frame{frame}, tt.want[i:i+1])
			}
		})
	}
}

func TestFFmpegFrameMD5OracleTestsrc16High444(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}

	for _, tt := range high444FixtureCases() {
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
				line := []byte(fmt.Sprintf("0, %10d, %10d,        1,      768, %s", i, i, hash))
				if !bytes.Contains(out, line) {
					t.Fatalf("frame[%d] missing %q in framemd5:\n%s", i, line, out)
				}
			}
		})
	}
}

func high444FixtureCases() []struct {
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
			hex:  testsrc16CAVLC444AnnexBHex,
			want: []string{
				"0ff3893d32b4b1875412d88a6fa4a5b1",
				"008c471027c25eab150c1cc4a30fb9ac",
				"ef107480f4c8b836d91e422e1f3c0b75",
				"6acd1f8bc304066008a32acf64228305",
			},
		},
		{
			name: "cabac",
			hex:  testsrc16CABAC444AnnexBHex,
			want: []string{
				"8539237f1ecaf659fa36c0f76cde8815",
				"6f594f9f9f10d12a399d54882ce6c8e5",
				"5e4250996d28cff7f2e85b95d78995ff",
				"452f232c9a94da5220babd530117a395",
			},
		},
	}
}

func assertHigh444FrameMD5Strings(t *testing.T, frames []*Frame, want []string) {
	t.Helper()
	if len(frames) != len(want) {
		t.Fatalf("frames = %d, want %d", len(frames), len(want))
	}
	for i, frame := range frames {
		if frame.Width != 16 || frame.Height != 16 || frame.ChromaFormatIDC != 3 || frame.BitDepthLuma != 8 || frame.BitDepthChroma != 8 {
			t.Fatalf("frame[%d] metadata = %dx%d chroma %d depth %d/%d", i, frame.Width, frame.Height, frame.ChromaFormatIDC, frame.BitDepthLuma, frame.BitDepthChroma)
		}
		raw, err := frame.AppendRawYUV(nil)
		if err != nil {
			t.Fatalf("frame[%d] raw yuv: %v", i, err)
		}
		if len(raw) != 768 {
			t.Fatalf("frame[%d] raw frame size = %d, want 768", i, len(raw))
		}
		got := md5.Sum(raw)
		if hex.EncodeToString(got[:]) != want[i] {
			t.Fatalf("frame[%d] md5 = %x, want %s", i, got, want[i])
		}
	}
}
