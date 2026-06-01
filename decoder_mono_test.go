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
)

const testsrc16CAVLCMonoAnnexBHex = `
000000016764000af2d3d80b64000003000400000300083c489a800000000168ce0f2c8b0000010605ffff6edc45e9bde6d948b7962cd820d923eeef78323634202d20636f7265203136352072333536303561202d20482e3236342f4d5045472d342041564320636f646563202d20436f70
796c65667420323030332d32303235202d20687474703a2f2f7777772e766964656f6c616e2e6f72672f783236342e68746d6c202d206f7074696f6e733a2063616261633d30207265663d31206465626c6f636b3d313a303a3020616e616c7973653d3078333a3078313333206d653d756d68207375626d
653d3130207073793d31207073795f72643d312e30303a302e3030206d697865645f7265663d30206d655f72616e67653d3234206368726f6d615f6d653d30207472656c6c69733d32203878386463743d312063716d3d3020646561647a6f6e653d32312c313120666173745f70736b69703d3120636872
6f6d615f71705f6f66667365743d2d3220746872656164733d31206c6f6f6b61686561645f746872656164733d3120736c696365645f746872656164733d30206e723d3020646563696d6174653d3120696e7465726c616365643d3020626c757261795f636f6d7061743d3020636f6e73747261696e6564
5f696e7472613d3020626672616d65733d3020776569676874703d30206b6579696e743d323530206b6579696e745f6d696e3d313236207363656e656375743d3020696e7472615f726566726573683d302072635f6c6f6f6b61686561643d36302072633d637266206d62747265653d31206372663d3233
2e302071636f6d703d302e36302071706d696e3d302071706d61783d3639207170737465703d342069705f726174696f3d312e34302061713d313a312e3030008000000165888413e88444c41410b3c00e923301842000121b6a0c3000710df678bf2ce597e1d5cf7ff80e10e12c80549003205010f8140d
b5781888c12f20e94cbf1abf673c213e4d3d4508b2b881574681e2ffc002b28878dd12dba2c3efd80c4de9935d3c323b396b1087471e0430eae1198f57acc813a050b49444f74f57a80003702085bd66437bf3fdf3d00064073b65bd7a3353505e00000001419a20ffc510703c1d2400d0280e4c241b6a2e
bf9fbde77f00554c2298b5abf63df800000001419a40bfc52a00fc806a12a94a0cc05c23d565efb6373e167335a8e5e8eb611487ad57a8763dc000000001419a63fd0dc23d4a
`

const testsrc16CABACMonoAnnexBHex = `
000000016764000af2d3d80b64000003000400000300083c489a800000000168ee0f2c8b0000010605ffff6edc45e9bde6d948b7962cd820d923eeef78323634202d20636f7265203136352072333536303561202d20482e3236342f4d5045472d342041564320636f646563202d20436f70
796c65667420323030332d32303235202d20687474703a2f2f7777772e766964656f6c616e2e6f72672f783236342e68746d6c202d206f7074696f6e733a2063616261633d31207265663d31206465626c6f636b3d313a303a3020616e616c7973653d3078333a3078313333206d653d756d68207375626d
653d3130207073793d31207073795f72643d312e30303a302e3030206d697865645f7265663d30206d655f72616e67653d3234206368726f6d615f6d653d30207472656c6c69733d32203878386463743d312063716d3d3020646561647a6f6e653d32312c313120666173745f70736b69703d3120636872
6f6d615f71705f6f66667365743d2d3220746872656164733d31206c6f6f6b61686561645f746872656164733d3120736c696365645f746872656164733d30206e723d3020646563696d6174653d3120696e7465726c616365643d3020626c757261795f636f6d7061743d3020636f6e73747261696e6564
5f696e7472613d3020626672616d65733d3020776569676874703d30206b6579696e743d323530206b6579696e745f6d696e3d313236207363656e656375743d3020696e7472615f726566726573683d302072635f6c6f6f6b61686561643d36302072633d637266206d62747265653d31206372663d3233
2e302071636f6d703d302e36302071706d696e3d302071706d61783d3639207170737465703d342069705f726174696f3d312e34302061713d313a312e3030008000000165888413ff76ec11e7ab8d8c3c253fb903ee84ad12ddc03765d8e69b4b2eba7c274b0e722b5611d9d3de550b701402018272396e
e0d5995c51d273d12610e58a40d84a742591009d429a0395c87a7aa21c61ce8c78031dab4521326ba570db8ca2d8e7f872f0c7498ff3133497781c5fdac62ce9f3f2be7927b9f40b588a9107ae782e77fffa7b1d69ff497f7648a4faf100000001419a227f5f488de24fdef2cf587d2d53d7578b681e5814
cf1f0beb16958b26884ec8ffc000000001419a425f4fcfa876e37384c08bf2d190fad7ad629c6825bac0f35dccc5cc8b6c2d0aecc6f2efa58c08513b00000001419a63ffda60751472b2cfe4
`

func TestDecodeAnnexBTestsrc16MonochromeFrames(t *testing.T) {
	for _, tt := range monoFixtureCases() {
		t.Run(tt.name, func(t *testing.T) {
			frames, err := NewDecoder().DecodeAnnexBFrames(decodeHexFixture(t, tt.hex))
			if err != nil {
				t.Fatal(err)
			}
			assertMonoFrameMD5Strings(t, frames, tt.want)
		})
	}
}

func TestDecodeAVCTestsrc16MonochromeFrames(t *testing.T) {
	for _, tt := range monoFixtureCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := decodeHexFixture(t, tt.hex)
			for _, nalLengthSize := range []int{2, 3, 4} {
				frames, err := NewDecoder().DecodeAVCFrames(annexBToAVC(t, data, nalLengthSize), nalLengthSize)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: %v", nalLengthSize, err)
				}
				assertMonoFrameMD5Strings(t, frames, tt.want)
			}
		})
	}
}

func TestDecodeAVCWithConfigurationRecordMonochromeFrames(t *testing.T) {
	for _, tt := range monoFixtureCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := decodeHexFixture(t, tt.hex)
			config, packet := annexBToAVCConfigAndPacket(t, data, 4)
			frames, err := NewDecoder().DecodeAVCFramesWithConfigurationRecord(config, packet)
			if err != nil {
				t.Fatal(err)
			}
			assertMonoFrameMD5Strings(t, frames, tt.want)
		})
	}
}

func TestDecodeConfiguredAVCAcrossSamplesMonochromeFrames(t *testing.T) {
	for _, tt := range monoFixtureCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := decodeHexFixture(t, tt.hex)
			config, samples := annexBToAVCConfigAndSamples(t, data, 4)
			dec := NewDecoder()
			if _, err := dec.ParseAVCDecoderConfigurationRecord(config); err != nil {
				t.Fatal(err)
			}
			for i, sample := range samples {
				frame, err := dec.DecodeConfiguredAVC(sample)
				if err != nil {
					t.Fatalf("sample[%d]: %v", i, err)
				}
				assertMonoFrameMD5Strings(t, []*Frame{frame}, tt.want[i:i+1])
			}
		})
	}
}

func TestFFmpegFrameMD5OracleTestsrc16Monochrome(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}

	for _, tt := range monoFixtureCases() {
		t.Run(tt.name, func(t *testing.T) {
			path := writeTempH264(t, decodeHexFixture(t, tt.hex))
			cmd := exec.Command("ffmpeg",
				"-v", "error",
				"-f", "h264",
				"-i", path,
				"-an", "-sn", "-dn",
				"-pix_fmt", "gray",
				"-f", "framemd5",
				"-",
			)
			out, err := cmd.Output()
			if err != nil {
				t.Fatalf("ffmpeg framemd5: %v", err)
			}
			for i, hash := range tt.want {
				line := []byte(fmt.Sprintf("0, %10d, %10d,        1,      256, %s", i, i, hash))
				if !bytes.Contains(out, line) {
					t.Fatalf("frame[%d] missing %q in framemd5:\n%s", i, line, out)
				}
			}
		})
	}
}

func monoFixtureCases() []struct {
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
			hex:  testsrc16CAVLCMonoAnnexBHex,
			want: []string{
				"7d7c6b5414619f78c6303e94f6c69dba",
				"6ae5ffb09f3156812deccefdf58a6c74",
				"f1dd36e9dbc0f928b6e57afc2022a8f2",
				"504e78844c238b097aa59235df29ec07",
			},
		},
		{
			name: "cabac",
			hex:  testsrc16CABACMonoAnnexBHex,
			want: []string{
				"cf88b0a4244f7df1c3c54613f6290345",
				"d003fa3ed4b3edd4622c36e4c2b5249c",
				"677639d3d5857b18931e727d46e6a4cc",
				"fb50b49ba64db3576559b442d3c4a6ad",
			},
		},
	}
}

func assertMonoFrameMD5Strings(t *testing.T, frames []*Frame, want []string) {
	t.Helper()
	if len(frames) != len(want) {
		t.Fatalf("frames = %d, want %d", len(frames), len(want))
	}
	for i, frame := range frames {
		if frame.Width != 16 || frame.Height != 16 || frame.ChromaFormatIDC != 0 || frame.BitDepthLuma != 8 || frame.BitDepthChroma != 8 {
			t.Fatalf("frame[%d] metadata = %dx%d chroma %d depth %d/%d", i, frame.Width, frame.Height, frame.ChromaFormatIDC, frame.BitDepthLuma, frame.BitDepthChroma)
		}
		raw, err := frame.AppendRawYUV(nil)
		if err != nil {
			t.Fatalf("frame[%d] raw yuv: %v", i, err)
		}
		if len(raw) != 256 {
			t.Fatalf("frame[%d] raw frame size = %d, want 256", i, len(raw))
		}
		got := md5.Sum(raw)
		if hex.EncodeToString(got[:]) != want[i] {
			t.Fatalf("frame[%d] md5 = %x, want %s", i, got, want[i])
		}
	}
}
