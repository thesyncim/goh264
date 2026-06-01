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

const testsrc16CAVLCLosslessAnnexBHex = `
0000000167f4000aaeb4f6022000000300200000030041e244d40000000168ce01af200000010605ffff0adc45e9bde6d948b7962cd820d923eeef78323634202d20636f7265203136352072333232322062333536303561202d20482e323634
2f4d5045472d342041564320636f646563202d20436f70796c65667420323030332d32303235202d20687474703a2f2f7777772e766964656f6c616e2e6f72672f783236342e68746d6c202d206f7074696f6e733a2063616261633d30207265
663d31206465626c6f636b3d303a303a3020616e616c7973653d3078313a3078313331206d653d756d68207375626d653d39207073793d30206d697865645f7265663d30206d655f72616e67653d3234206368726f6d615f6d653d3120747265
6c6c69733d30203878386463743d302063716d3d3020646561647a6f6e653d32312c313120666173745f70736b69703d30206368726f6d615f71705f6f66667365743d3020746872656164733d31206c6f6f6b61686561645f74687265616473
3d3120736c696365645f746872656164733d30206e723d3020646563696d6174653d3120696e7465726c616365643d3020626c757261795f636f6d7061743d3020636f6e73747261696e65645f696e7472613d3020626672616d65733d302077
6569676874703d30206b6579696e743d323530206b6579696e745f6d696e3d313236207363656e656375743d3020696e7472615f726566726573683d302072633d637170206d62747265653d302071703d300080000001658884ac40046010f0
e00080002146000210e000209606c08dadadadadadadadadadadac30c0e000218e000215400020dc04e0380023804846583c000f00c380787d78804036a61200021ae000217600020fdfffc09f90f0001097000107a000103e000581b701e808
44c1e0002104000200c000205600058471e08000210400020cc0002096004c08c13c64000086b000085d000083fa290000212c00020f600020bc0881600021ae000217600020fc000200c0371f388800010b6000107b1c000b0373bde3e7d040
00202c000201e00020cc00301020002144161a00070001045a002030e1c001858b01c1e361e0f078e0000412ca1080aa0116c88a832eddbb0bdc000082e800080181e05c35a29744abdcab452d2e8f0000209650801200ac8212a9568154abdb
6954001c0004110004208208005208068ab02a2a2ae000040c800040180f1759c1f4eb00fcf51e3c7c7c000082156000100800010081603eec408c40ec2f0bddb80001032000100603c9004383e08081dc80408fe88100000001419a22bc6c00
582118c6318c6318c13800021dc00021a4000212c000203c16130984c8f3800446318c631f819800119ce3fdf8200046739d4b316d93f070001023000100700b0993ffc78004673bbe2008000864808000600004158001e00021ca00022ba000
21aa000202a012845cc4219659600020880002004c600f0001072007000108f0048322966208004049658100563800000001419a42bc6c07a481fe0f0001022000100600a09224489101feffb0fa492480fff9ee000081100008030050491224
48a3c600023bde7fbf1000119cf7bde06c5e65244fcb0001023000100706f80e0002046000200e0161327e5800461e838004673bb50058000822800081480123efcc5fafc6c804846640000822000601e2e047015d0058094000400c30a0fa45
f8d90000480a25990070020b811c000c09b000000001419a62bc2be07000100200c14fbf0380023bdf6a2800080e80c5080f14000403c0628001501f
`

const testsrc16CABACLosslessAnnexBHex = `
0000000167f4000aaeb4f6022000000300200000030041e244d40000000168ee01af200000010605ffff0adc45e9bde6d948b7962cd820d923eeef78323634202d20636f7265203136352072333232322062333536303561202d20482e323634
2f4d5045472d342041564320636f646563202d20436f70796c65667420323030332d32303235202d20687474703a2f2f7777772e766964656f6c616e2e6f72672f783236342e68746d6c202d206f7074696f6e733a2063616261633d31207265
663d31206465626c6f636b3d303a303a3020616e616c7973653d3078313a3078313331206d653d756d68207375626d653d39207073793d30206d697865645f7265663d30206d655f72616e67653d3234206368726f6d615f6d653d3120747265
6c6c69733d30203878386463743d302063716d3d3020646561647a6f6e653d32312c313120666173745f70736b69703d30206368726f6d615f71705f6f66667365743d3020746872656164733d31206c6f6f6b61686561645f74687265616473
3d3120736c696365645f746872656164733d30206e723d3020646563696d6174653d3120696e7465726c616365643d3020626c757261795f636f6d7061743d3020636f6e73747261696e65645f696e7472613d3020626672616d65733d302077
6569676874703d30206b6579696e743d323530206b6579696e745f6d696e3d313236207363656e656375743d3020696e7472615f726566726573683d302072633d637170206d62747265653d302071703d300080000001658884afe563c08096
d4aa2c2fc75f6bfb258590b8a816a99827cfa3c181c99bfc9696eb48a0c6fffffffecd12109c3f488afbbf295bb9b1016f87aafb5dbe788a2e9a12074efc24e370c9cd3b5d85561c6cac97a09bcbf36a90c97cbe3283595833d49c55667f781e
6b924adf485c7e2ca9ede4278f229ad67cd556d77bd55bda793e62b5d32f3d260f64bd5a8e4cbdcf3775d243f4368df1d37d03f81f3e52cbfe0cd312a8ea237b227461ebc8ad113e8339977fd338ace9ab11c26bb5a8d6b5c0fafe2a6ada1fe2
e8b9a57cc5ca7d23b93568f8f44e7862b07b546dbeb5135d1e4353163c5b77a8d3d23c2965d05bd4064bbfa7f0ecd2fbc3ec0b23c20db30a8bec09503efb536844e1321762998cca8ad2ea756496eb1a79217df37d4e74ff9a541c477bc3f5d1
62e18034d9e03e944b1df408115a3cb69b7346c1b3b81138687db3ce0803a65b5997c353eabb76497f918300000001419a235fab5e290affcff4837b0854cb669abd5b7ffa861a9024e13bd9c72380483a71685001c62074853aefbb3f8a9b63
b405c579654e0c7179e5bf7740f909de94c9f333f93fc2e12d8548420f6f19e81ed7ea2c0e06ef221697ffff9fb5ff029814e80caa3b7a4457a36675f4baf9d5594b4a4722e8ac42593a3494fce6bd01e85ef3309c37d585b08be00000000141
9a435faae4f2fcf48fce97c165b1b2a55f20bb5dc7e08d80c11bd325b0a4edd10bb014db2d926f19a4b97b622f90315b141bc49eddbcb3f874b911e7d1c7802491593842de8055cde85394c07282fe37d7c67a503b0a06f7347fffe7c5ff0289
037eac257dab1f405bd05df2777da17cb0c1f0459fe7828b1f3ef2a432de4f3145006fe680b6d649056a165a1f00000001419a635fac15c1de63fc0bf6190fe11e3ed73d8e495d6c3fff9e1823643a5f8a2d34d87f
`

func TestDecodeAnnexBTestsrc16LosslessFrames(t *testing.T) {
	for _, tt := range losslessFixtureCases() {
		t.Run(tt.name, func(t *testing.T) {
			frames, err := NewDecoder().DecodeAnnexBFrames(decodeHexFixture(t, tt.hex))
			if err != nil {
				t.Fatal(err)
			}
			assertLosslessFrameMD5Format(t, frames, tt.want)
		})
	}
}

func TestDecodeAVCTestsrc16LosslessFrames(t *testing.T) {
	for _, tt := range losslessFixtureCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := decodeHexFixture(t, tt.hex)
			for _, nalLengthSize := range []int{2, 3, 4} {
				frames, err := NewDecoder().DecodeAVCFrames(annexBToAVC(t, data, nalLengthSize), nalLengthSize)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: %v", nalLengthSize, err)
				}
				assertLosslessFrameMD5Format(t, frames, tt.want)
			}
		})
	}
}

func TestDecodeAVCWithConfigurationRecordLosslessFrames(t *testing.T) {
	for _, tt := range losslessFixtureCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := decodeHexFixture(t, tt.hex)
			config, packet := annexBToAVCConfigAndPacket(t, data, 4)
			frames, err := NewDecoder().DecodeAVCFramesWithConfigurationRecord(config, packet)
			if err != nil {
				t.Fatal(err)
			}
			assertLosslessFrameMD5Format(t, frames, tt.want)
		})
	}
}

func TestDecodeConfiguredAVCAcrossSamplesLosslessFrames(t *testing.T) {
	for _, tt := range losslessFixtureCases() {
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
				assertLosslessFrameMD5Format(t, []*Frame{frame}, tt.want[i:i+1])
			}
		})
	}
}

func TestFFmpegFrameMD5OracleTestsrc16Lossless(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}

	for _, tt := range losslessFixtureCases() {
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
				line := []byte(fmt.Sprintf("0, %10d, %10d,        1,      384, %s", i, i, hash))
				if !bytes.Contains(out, line) {
					t.Fatalf("frame[%d] missing %q in framemd5:\n%s", i, line, out)
				}
			}
		})
	}
}

func losslessFixtureCases() []struct {
	name string
	hex  string
	want []string
} {
	want := []string{
		"69fcf25f35e829e5a3d96cbaaf22bbb6",
		"8563271dc08ef4ed388ebc1f7016834c",
		"1a054a3901101da0f6b6c58d8e71bbdb",
		"a0addb72f5ea0957ef8a05b782f0e9ff",
	}
	return []struct {
		name string
		hex  string
		want []string
	}{
		{name: "cavlc", hex: testsrc16CAVLCLosslessAnnexBHex, want: want},
		{name: "cabac", hex: testsrc16CABACLosslessAnnexBHex, want: want},
	}
}

func assertLosslessFrameMD5Format(t *testing.T, frames []*Frame, want []string) {
	t.Helper()
	if len(frames) != len(want) {
		t.Fatalf("frames = %d, want %d", len(frames), len(want))
	}
	for i, frame := range frames {
		if frame.Width != 16 || frame.Height != 16 || frame.ChromaFormatIDC != 1 || frame.BitDepthLuma != 8 || frame.BitDepthChroma != 8 {
			t.Fatalf("frame[%d] metadata = %dx%d chroma %d depth %d/%d", i, frame.Width, frame.Height, frame.ChromaFormatIDC, frame.BitDepthLuma, frame.BitDepthChroma)
		}
		raw, err := frame.AppendRawYUV(nil)
		if err != nil {
			t.Fatalf("frame[%d] raw yuv: %v", i, err)
		}
		if len(raw) != 384 {
			t.Fatalf("frame[%d] raw frame size = %d, want 384", i, len(raw))
		}
		got := md5.Sum(raw)
		if hex.EncodeToString(got[:]) != want[i] {
			t.Fatalf("frame[%d] md5 = %x, want %s", i, got, want[i])
		}
	}
}
