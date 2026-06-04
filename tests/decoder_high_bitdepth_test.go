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

const testsrc16High10CAVLCIAnnexBHex = `
00000001676e100aa6cbbd808800000300080000030010200000000168ce0f2c8b0000010605ffff59dc45e9bde6d948b7962cd820d923eeef78323634202d20636f7265203136352072333232322062333536303561202d20482e3236342f4d5045472d342041564320636f
646563202d20436f70796c65667420323030332d32303235202d20687474703a2f2f7777772e766964656f6c616e2e6f72672f783236342e68746d6c202d206f7074696f6e733a2063616261633d30207265663d31206465626c6f636b3d303a303a3020616e616c797365
3d3078333a3078313133206d653d686578207375626d653d37207073793d31207073795f72643d312e30303a302e3030206d697865645f7265663d30206d655f72616e67653d3136206368726f6d615f6d653d31207472656c6c69733d31203878386463743d312063716d
3d3020646561647a6f6e653d32312c313120666173745f70736b69703d31206368726f6d615f71705f6f66667365743d2d3220746872656164733d31206c6f6f6b61686561645f746872656164733d3120736c696365645f746872656164733d30206e723d
3020646563696d6174653d3120696e7465726c616365643d3020626c757261795f636f6d7061743d3020636f6e73747261696e65645f696e7472613d3020626672616d65733d3020776569676874703d30206b6579696e743d31206b6579696e745f6d
696e3d31207363656e656375743d3020696e7472615f726566726573683d302072633d637266206d62747265653d30206372663d32332e302071636f6d703d302e36302071706d696e3d302071706d61783d3831207170737465703d342069
705f726174696f3d312e34302061713d313a312e3030008000000165888432a218ab030217e0214c880004026b50618003c5e33a31a3a05f77ff81a40d900d01ca08b0cad5e6a86cce6e5bd76d22637cc4ca82fabbfc4fb291c84f22c08
ffec16cb43486e123fb6b86906110b49f07abd757181b3284d3d5ea03e4211314054fff7cb079d9f57068abfef83020004e618001518190013e6615663a0422fa93405b3a603213ec353f903b2b7f738004a82119748d
200375c51558d10d09e28ee6d62e9fd846c959808d606feea03a914b82c4d2bbe8
`

const testsrc16High10CABACIAnnexBHex = `
00000001676e100aa6cbbd808800000300080000030010200000000168ee0f2c8b0000010605ffff59dc45e9bde6d948b7962cd820d923eeef78323634202d20636f7265203136352072333232322062333536303561202d20482e3236342f4d5045472d342041564320636f
646563202d20436f70796c65667420323030332d32303235202d20687474703a2f2f7777772e766964656f6c616e2e6f72672f783236342e68746d6c202d206f7074696f6e733a2063616261633d31207265663d31206465626c6f636b3d303a303a3020616e616c797365
3d3078333a3078313133206d653d686578207375626d653d37207073793d31207073795f72643d312e30303a302e3030206d697865645f7265663d30206d655f72616e67653d3136206368726f6d615f6d653d31207472656c6c69733d31203878386463743d312063716d
3d3020646561647a6f6e653d32312c313120666173745f70736b69703d31206368726f6d615f71705f6f66667365743d2d3220746872656164733d31206c6f6f6b61686561645f746872656164733d3120736c696365645f746872656164733d30206e723d
3020646563696d6174653d3120696e7465726c616365643d3020626c757261795f636f6d7061743d3020636f6e73747261696e65645f696e7472613d3020626672616d65733d3020776569676874703d30206b6579696e743d31206b6579696e745f6d
696e3d31207363656e656375743d3020696e7472615f726566726573683d302072633d637266206d62747265653d30206372663d32332e302071636f6d703d302e36302071706d696e3d302071706d61783d3831207170737465703d342069
705f726174696f3d312e34302061713d313a312e3030008000000165888432d4da2982ac9ea8196416cd88b1fe2f490a9de5c1d8c0b2373808b212e49e564abaaf42198713fdb763f080050d4c179d203e05cc847023255aa0359b
6574fda9a04d17eb337b778b2ca784f855cf2a6825b9eb0d3e7b204e5dac7ff83717e7c2440484f18e45d1a6afedc63d23bdc27abe243a5955a9a9ad3c4d97a3c332435c8953ef733211b3855a189cf76fb56e
4db2c2914c68a350879b8251f7ebaeb99c68e2a4efc25611bebd2813f9db93bff574d9d38d
`

const gray16High10CAVLCPSkipAnnexBHex = `
00000001676e000aa6cb4f6022000003000200000300041e244d400000000168ce01ccb22c0000016588843a118a00021031c000a47000298000000001419a2294
`

const gray16High10CABACPSkipAnnexBHex = `
00000001676e000aa6cb4f6022000003000200000300041e244d400000000168ee01ccb22c0000016588843afeee82be0523c4c4d2b7e100000001419a235ffef0
`

const step32x16High10CAVLCP16x16NoResidualAnnexBHex = `
00000001676e000aa6cb45d80880000003008000000301078913500000000168ce01ccb20000016588843a26280004e4b26280007cd58000000001419a22b0101fe0
`

const step32x16High10CABACP16x16NoResidualAnnexBHex = `
00000001676e000aa6cb45d80880000003008000000301078913500000000168ee01ccb20000016588843afef7d4b7ccb2eea3c2b55181f9b5586100000001419a235faa092ccffad67ffc
`

func TestDecodeAnnexBHigh10IntraFrames(t *testing.T) {
	for _, tt := range high10IntraFixtureCases() {
		t.Run(tt.name, func(t *testing.T) {
			frames, err := NewDecoder().DecodeAnnexBFrames(decodeHexFixture(t, tt.hex))
			if err != nil {
				t.Fatal(err)
			}
			assertHigh10FrameMD5Strings(t, frames, []string{tt.md5})
		})
	}
}

func TestDecodeAVCHigh10IntraFrames(t *testing.T) {
	for _, tt := range high10IntraFixtureCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := decodeHexFixture(t, tt.hex)
			for _, nalLengthSize := range []int{2, 3, 4} {
				frames, err := NewDecoder().DecodeAVCFrames(annexBToAVC(t, data, nalLengthSize), nalLengthSize)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: %v", nalLengthSize, err)
				}
				assertHigh10FrameMD5Strings(t, frames, []string{tt.md5})
			}
		})
	}
}

func TestDecodeAVCWithConfigurationRecordHigh10IntraFrames(t *testing.T) {
	for _, tt := range high10IntraFixtureCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := decodeHexFixture(t, tt.hex)
			config, packet := annexBToAVCConfigAndPacket(t, data, 4)
			frames, err := NewDecoder().DecodeAVCFramesWithConfigurationRecord(config, packet)
			if err != nil {
				t.Fatal(err)
			}
			assertHigh10FrameMD5Strings(t, frames, []string{tt.md5})
		})
	}
}

func TestDecodeAnnexBHigh10InterFrames(t *testing.T) {
	for _, tt := range high10InterFixtureCases() {
		t.Run(tt.name, func(t *testing.T) {
			frames, err := NewDecoder().DecodeAnnexBFrames(decodeHexFixture(t, tt.hex))
			if err != nil {
				t.Fatal(err)
			}
			assertHigh10FrameMD5Strings(t, frames, tt.want)
		})
	}
}

func TestDecodeAVCHigh10InterFrames(t *testing.T) {
	for _, tt := range high10InterFixtureCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := decodeHexFixture(t, tt.hex)
			for _, nalLengthSize := range []int{2, 3, 4} {
				frames, err := NewDecoder().DecodeAVCFrames(annexBToAVC(t, data, nalLengthSize), nalLengthSize)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: %v", nalLengthSize, err)
				}
				assertHigh10FrameMD5Strings(t, frames, tt.want)
			}
		})
	}
}

func TestDecodeAVCWithConfigurationRecordHigh10InterFrames(t *testing.T) {
	for _, tt := range high10InterFixtureCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := decodeHexFixture(t, tt.hex)
			for _, nalLengthSize := range []int{2, 3, 4} {
				config, packet := annexBToAVCConfigAndPacket(t, data, nalLengthSize)
				frames, err := NewDecoder().DecodeAVCFramesWithConfigurationRecord(config, packet)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: %v", nalLengthSize, err)
				}
				assertHigh10FrameMD5Strings(t, frames, tt.want)
			}
		})
	}
}

func TestDecodeConfiguredAVCAcrossSamplesHigh10InterFrames(t *testing.T) {
	for _, tt := range high10InterFixtureCases() {
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
				assertHigh10FrameMD5Strings(t, frames, tt.want)
			}
		})
	}
}

func TestFFmpegFrameMD5OracleHigh10Intra(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}

	for _, tt := range high10IntraFixtureCases() {
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
			line := []byte(fmt.Sprintf("0, %10d, %10d,        1,      768, %s", 0, 0, tt.md5))
			if !bytes.Contains(out, line) {
				t.Fatalf("missing %q in framemd5:\n%s", line, out)
			}
		})
	}
}

func TestFFmpegFrameMD5OracleHigh10Inter(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}

	for _, tt := range high10InterFixtureCases() {
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
				line := []byte(fmt.Sprintf("0, %10d, %10d,        1, %8d, %s", i, i, tt.rawSize, hash))
				if !bytes.Contains(out, line) {
					t.Fatalf("frame[%d] missing %q in framemd5:\n%s", i, line, out)
				}
			}
		})
	}
}

func high10IntraFixtureCases() []struct {
	name string
	hex  string
	md5  string
} {
	return []struct {
		name string
		hex  string
		md5  string
	}{
		{
			name: "cavlc",
			hex:  testsrc16High10CAVLCIAnnexBHex,
			md5:  "fd302f00e365b8502c44005ea308c468",
		},
		{
			name: "cabac",
			hex:  testsrc16High10CABACIAnnexBHex,
			md5:  "38ed4870a1ba82aeb0c45b09d67e3e2a",
		},
	}
}

func high10InterFixtureCases() []struct {
	name    string
	hex     string
	rawSize int
	want    []string
} {
	return []struct {
		name    string
		hex     string
		rawSize int
		want    []string
	}{
		{
			name:    "cavlc-pskip",
			hex:     gray16High10CAVLCPSkipAnnexBHex,
			rawSize: 768,
			want: []string{
				"87e217773d3e8b548fdf2002955cfcb9",
				"87e217773d3e8b548fdf2002955cfcb9",
			},
		},
		{
			name:    "cabac-pskip",
			hex:     gray16High10CABACPSkipAnnexBHex,
			rawSize: 768,
			want: []string{
				"87e217773d3e8b548fdf2002955cfcb9",
				"87e217773d3e8b548fdf2002955cfcb9",
			},
		},
		{
			name:    "cavlc-p16x16-no-residual",
			hex:     step32x16High10CAVLCP16x16NoResidualAnnexBHex,
			rawSize: 1536,
			want: []string{
				"e0f04baf1c5940cf72857345ca05bbee",
				"c356cd5790ea90f599ad5c2230869f06",
			},
		},
		{
			name:    "cabac-p16x16-no-residual",
			hex:     step32x16High10CABACP16x16NoResidualAnnexBHex,
			rawSize: 1536,
			want: []string{
				"e0f04baf1c5940cf72857345ca05bbee",
				"c356cd5790ea90f599ad5c2230869f06",
			},
		},
		{
			name:    "cavlc-p16x16-residual",
			hex:     high10ResidualCAVLCP16x16AnnexBHex,
			rawSize: high10ResidualCAVLCFrameRawSize,
			want:    high10ResidualCAVLCFrameMD5,
		},
		{
			name:    "cabac-p16x16-residual",
			hex:     high10CABACP16x16ResidualAnnexBHex,
			rawSize: high10CABACP16x16ResidualRawFrameSize,
			want:    high10CABACP16x16ResidualFrameMD5,
		},
		{
			name:    "cavlc-weighted-p16x16",
			hex:     high10WeightedCAVLCAnnexBHex,
			rawSize: high10WeightedPFrameRawSize,
			want:    high10WeightedPFrameMD5,
		},
		{
			name:    "cabac-weighted-p16x16",
			hex:     high10WeightedCABACAnnexBHex,
			rawSize: high10WeightedPFrameRawSize,
			want:    high10WeightedPFrameMD5,
		},
	}
}

func assertHigh10FrameMD5Strings(t *testing.T, frames []*Frame, want []string) {
	t.Helper()
	if len(frames) != len(want) {
		t.Fatalf("frames = %d, want %d", len(frames), len(want))
	}
	for i, frame := range frames {
		if frame == nil {
			t.Fatalf("frame[%d] is nil", i)
		}
		if frame.BitDepthLuma != 10 || frame.BitDepthChroma != 10 || frame.ChromaFormatIDC != 1 {
			t.Fatalf("frame[%d] format depth/chroma = %d/%d/%d, want 10/10/1", i, frame.BitDepthLuma, frame.BitDepthChroma, frame.ChromaFormatIDC)
		}
		if got, err := frame.RawPixelFormat(); err != nil || got != "yuv420p10le" {
			t.Fatalf("frame[%d] RawPixelFormat = %q/%v, want yuv420p10le/nil", i, got, err)
		}
		if len(frame.Y) != 0 || len(frame.Cb) != 0 || len(frame.Cr) != 0 {
			t.Fatalf("frame[%d] populated 8-bit planes", i)
		}
		if len(frame.Y16) == 0 || len(frame.Cb16) == 0 || len(frame.Cr16) == 0 {
			t.Fatalf("frame[%d] missing high planes", i)
		}
		if _, err := frame.AppendRawYUV(nil); err != ErrUnsupported {
			t.Fatalf("frame[%d] AppendRawYUV error = %v, want ErrUnsupported", i, err)
		}
		raw, err := frame.AppendRawYUVBytesLE(nil)
		if err != nil {
			t.Fatalf("frame[%d] AppendRawYUVBytesLE: %v", i, err)
		}
		rawSize, err := frame.RawYUVSize()
		if err != nil {
			t.Fatalf("frame[%d] RawYUVSize: %v", i, err)
		}
		if len(raw) != rawSize {
			t.Fatalf("frame[%d] raw len = %d, want %d", i, len(raw), rawSize)
		}
		sum := md5.Sum(raw)
		if got := hex.EncodeToString(sum[:]); got != want[i] {
			t.Fatalf("frame[%d] md5 = %s, want %s", i, got, want[i])
		}
	}
}
