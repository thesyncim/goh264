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

	"github.com/thesyncim/goh264/internal/h264"
)

const high10TemporalDirectBCAVLCAnnexBHex = `
00000001676e000aa6cecba1000003000100000300028f1225380000000168ca83cb2000000165888403ac4222a00c0843f0045360c46003eb50282800391978535441d84a825121ffe038080c06320e0202010cbc64883311c26be17fc23842f4e3f29d4be3d623ffc0f5f9d8110f98c1f39a419ffef581995ad0aa9a19721ffeed41a6146791e9740f57adc25c70753963ba7abd200013e33988a783fafeffda00f0aa5f5ac3b55fef81000080f0000805001c150802001c38100007fc6738489638f50612e779a0f275600535581910aaba900b52affca4050b1062b2ebdfc8005fd06b8d84aa672c88471d69571f43c8e047ed8cee004554833fbd860073897e6c8dbcb3fdd89808fb5da9c3a080004115900ba6215879d5ab81993029ce24ef37e5f3fffc30c0e020602192801282f45250c229bc2fad7ffac881b2050c497444643d5e04461a28c9716e3483d5ec0033a120232a82f2567f7ce011cc4b9e6b9075003f8a72d110ca6b713ae02478980ae2a0f11bff9c011cd439e539075680c9d2d1d5387242a0ffe080800080f14000208411000783ec432013e2164140445b6b78084c9bde1c9368046708d7473011694d37ffb006c82b5bea0bb4e6c7efb8e6331ed42831e8b7f0f0cb809e469f40453a2dc0dd0ec5bca03c4cc67008ee329574d29b13488000000001419a2995e1e8a600a590661ccad423cc2fccabf41e76e800a6f49c4c77db1c47b72163d6e486cf5a82000100900140b2610044530001e909187ea49e4163d794bc9c47cb830275e6a4c2f71d800087105ce34c5aa019fbf5e9efeeda0fbf0def533e2b90ff0e18ace000000001019e44e5636a0218c9d2b52ebc985d904f221bd6a12330fff27e318655d4056e755d05b74a4d97ac6dae08000fb817c0c01c300012590c85520e4977eb3cb710e8f7c3afb6b9c985c4a964a0ae0002690798ecd68dfb901c749e377e7775978d52921ec640c5cfc5c531717173f172f1717171714e
`

const high10TemporalDirectBCABACAnnexBHex = `
00000001676e000aa6cecba1000003000100000300028f1225380000000168ea83cb2000000165888403afb7fa75bac06a77adc3b954e694d225df58d9c3105d279306e2b462038ddbd42708d722d1e8cccd81e43a859e8160d233abdeab6f40f47fafc97e3135d796bde887cc2e60bd346a846569bf086031b7bb29ac048e3f1a6371c936aba35f4a7e8d9eecd7d0757f0e78f4eca164074a627e62ea6530b090b34f6a4274da808b3fb22de21862d33ab311e7f97116c167ec8388ce47df43fa9c25af8c31271627dd09856bd1f10a61363a55ea5d97b857c39f56c70029e003429e2d774ee3553498e0afff94f95ae2e4303a72998c9ed1e3142db2059bba9cf13f1d3aea406c5104d6b1abadf6f6744a3cc8837361f78f8cfbd6926d3b79f59dc3fdcaf19dfec96a0058d1cdcbc60722edbbbaeace9a81f4f729405a90ad17e4208ab19aa5eb832d7ddcc3aa53f75b4f1e7cfd112dbd30e6bbd7458ff1b8b786b8198ebcb529b71ed751588c603c9171bbc65a996f5578318851fcd725901d9760ed003fa6b749d180d0900e3fbd2f331f71456c079e7905e87fd058471327f4895f1b2cbeb022ebac827e138032160328213980003f9c0719fe1af8bcd13d03ab4b9df51e6fcd335c45294095fc03dbfcfdfb57bdc02d03497d64d5c259fe087d00000001419a299ad95de616e92b8f6a96ef0f345c610dbdedf4e71fa75382796081478ff89929f81049dbaedb6b6c2be677e3a9af6b68e56989d198a8f4fd97e1ddb93e5fa277b5d8669fffc4807e6228380ba038defc82ef773b41b4f8816ff2b0480b1df7b5e75069d047f941f5c4dc1e03c02209dd5b8000000001019e44e6bffdec599c393e7f7fadb84a5279338aea59ecbad5f01c2581d978f9e7d06949a234c0844b24fbe47dc1c179f327f304ab137a3d2068df0c1d4525b6fdc9e55e991ee4760c65985efe780bdfc8b8e1c056fda4ee9d0658872b9555deb90766b9d243452b05e9e450cbe1be67962420d17434200e008fb5c6911e6ca1
`

const high10SpatialDirectBCAVLCAnnexBHex = `
00000001676e000aa6cecba1000003000100000300028f1225380000000168ca83cb2000000165888403ac4222a00c0843f0045360c46003eb50282800391978535441d84a825121ffe038080c06320e0202010cbc64883311c26be17fc23842f4e3f29d4be3d623ffc0f5f9d8110f98c1f39a419ffef581995ad0aa9a19721ffeed41a6146791e9740f57adc25c70753963ba7abd200013e33988a783fafeffda00f0aa5f5ac3b55fef81000080f0000805001c150802001c38100007fc6738489638f50612e779a0f275600535581910aaba900b52affca4050b1062b2ebdfc8005fd06b8d84aa672c88471d69571f43c8e047ed8cee004554833fbd860073897e6c8dbcb3fdd89808fb5da9c3a080004115900ba6215879d5ab81993029ce24ef37e5f3fffc30c0e020602192801282f45250c229bc2fad7ffac881b2050c497444643d5e04461a28c9716e3483d5ec0033a120232a82f2567f7ce011cc4b9e6b9075003f8a72d110ca6b713ae02478980ae2a0f11bff9c011cd439e539075680c9d2d1d5387242a0ffe080800080f14000208411000783ec432013e2164140445b6b78084c9bde1c9368046708d7473011694d37ffb006c82b5bea0bb4e6c7efb8e6331ed42831e8b7f0f0cb809e469f40453a2dc0dd0ec5bca03c4cc67008ee329574d29b13488000000001419a2995e1e8a600a590661ccad423cc2fccabf41e76e800a6f49c4c77db1c47b72163d6e486cf5a82000100900140b2610044530001e909187ea49e4163d794bc9c47cb830275e6a4c2f71d800087105ce34c5aa019fbf5e9efeeda0fbf0def533e2b90ff0e18ace000000001019e45e5636a0218c9d2b52ebc985d904f221bd6a12330fff27e318655d4056e755d05b74a4d97ac6dae08000fb817c0c01c300012590c85520e4977eb3cb710e8f7c3afb6b9c985c4a964a0ae0002690798ecd68dfb901c749e377e7775978d52921ec640c5cfc5c531717173f172f1717171714e
`

const high10SpatialDirectBCABACAnnexBHex = `
00000001676e000aa6cecba1000003000100000300028f1225380000000168ea83cb2000000165888403afb7fa75bac06a77adc3b954e694d225df58d9c3105d279306e2b462038ddbd42708d722d1e8cccd81e43a859e8160d233abdeab6f40f47fafc97e3135d796bde887cc2e60bd346a846569bf086031b7bb29ac048e3f1a6371c936aba35f4a7e8d9eecd7d0757f0e78f4eca164074a627e62ea6530b090b34f6a4274da808b3fb22de21862d33ab311e7f97116c167ec8388ce47df43fa9c25af8c31271627dd09856bd1f10a61363a55ea5d97b857c39f56c70029e003429e2d774ee3553498e0afff94f95ae2e4303a72998c9ed1e3142db2059bba9cf13f1d3aea406c5104d6b1abadf6f6744a3cc8837361f78f8cfbd6926d3b79f59dc3fdcaf19dfec96a0058d1cdcbc60722edbbbaeace9a81f4f729405a90ad17e4208ab19aa5eb832d7ddcc3aa53f75b4f1e7cfd112dbd30e6bbd7458ff1b8b786b8198ebcb529b71ed751588c603c9171bbc65a996f5578318851fcd725901d9760ed003fa6b749d180d0900e3fbd2f331f71456c079e7905e87fd058471327f4895f1b2cbeb022ebac827e138032160328213980003f9c0719fe1af8bcd13d03ab4b9df51e6fcd335c45294095fc03dbfcfdfb57bdc02d03497d64d5c259fe087d00000001419a299ad95de616e92b8f6a96ef0f345c610dbdedf4e71fa75382796081478ff89929f81049dbaedb6b6c2be677e3a9af6b68e56989d198a8f4fd97e1ddb93e5fa277b5d8669fffc4807e6228380ba038defc82ef773b41b4f8816ff2b0480b1df7b5e75069d047f941f5c4dc1e03c02209dd5b8000000001019e45e6bffdec599c393e7f7fadb84a5279338aea59ecbad5f01c2581d978f9e7d06949a234c0844b24fbe47dc1c179f327f304ab137a3d2068df0c1d4525b6fdc9e55e991ee4760c65985efe780bdfc8b8e1c056fda4ee9d0658872b9555deb90766b9d243452b05e9e450cbe1be67962420d17434200e008fb5c6911e6ca1
`

const high10DirectBFrameRawSize = 1536

type high10DirectBFixture struct {
	name          string
	hex           string
	cabac         int32
	directSpatial int32
	annexBSize    int
	annexBMD5     string
	frameMD5      []string
	rawVideoMD5   string
}

func TestHigh10DirectBFixtureSyntax(t *testing.T) {
	for _, tt := range high10DirectBFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := decodeHexFixture(t, tt.hex)
			if len(data) != tt.annexBSize {
				t.Fatalf("annex b size = %d, want %d", len(data), tt.annexBSize)
			}
			sum := md5.Sum(data)
			if got := hex.EncodeToString(sum[:]); got != tt.annexBMD5 {
				t.Fatalf("annex b md5 = %s, want %s", got, tt.annexBMD5)
			}
			assertHigh10DirectBFixtureSyntax(t, data, tt)
		})
	}
}

func TestDecodeAnnexBHigh10DirectBFrames(t *testing.T) {
	for _, tt := range high10DirectBFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := decodeHexFixture(t, tt.hex)
			assertHigh10DirectBFixtureSyntax(t, data, tt)
			frames, err := NewDecoder().DecodeAnnexBFrames(data)
			if err != nil {
				t.Fatalf("DecodeAnnexBFrames: %v", err)
			}
			assertHigh10DirectBFrames(t, frames, tt.frameMD5)
		})
	}
}

func TestDecodeAVCHigh10DirectBFrames(t *testing.T) {
	for _, tt := range high10DirectBFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := decodeHexFixture(t, tt.hex)
			assertHigh10DirectBFixtureSyntax(t, data, tt)
			for _, nalLengthSize := range []int{2, 3, 4} {
				frames, err := NewDecoder().DecodeAVCFrames(annexBToAVC(t, data, nalLengthSize), nalLengthSize)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: DecodeAVCFrames: %v", nalLengthSize, err)
				}
				assertHigh10DirectBFrames(t, frames, tt.frameMD5)
			}
		})
	}
}

func TestDecodeAVCCHigh10DirectBFrames(t *testing.T) {
	for _, tt := range high10DirectBFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := decodeHexFixture(t, tt.hex)
			assertHigh10DirectBFixtureSyntax(t, data, tt)
			for _, nalLengthSize := range []int{2, 3, 4} {
				config, packet := annexBToAVCConfigAndPacket(t, data, nalLengthSize)
				frames, err := NewDecoder().DecodeAVCCFrames(config, packet)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: DecodeAVCCFrames: %v", nalLengthSize, err)
				}
				assertHigh10DirectBFrames(t, frames, tt.frameMD5)
			}
		})
	}
}

func TestDecodeConfiguredAVCHigh10DirectBFramesAcrossSamplesFlush(t *testing.T) {
	for _, tt := range high10DirectBFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := decodeHexFixture(t, tt.hex)
			assertHigh10DirectBFixtureSyntax(t, data, tt)
			for _, nalLengthSize := range []int{2, 3, 4} {
				config, samples := annexBToAVCConfigAndSamples(t, data, nalLengthSize)
				if len(samples) != len(tt.frameMD5) {
					t.Fatalf("nalLengthSize=%d: samples = %d, want %d", nalLengthSize, len(samples), len(tt.frameMD5))
				}

				dec := NewDecoder()
				if _, err := dec.ConfigureAVCC(config); err != nil {
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
				assertHigh10DirectBFrames(t, frames, tt.frameMD5)

				out, err = dec.FlushDelayedFrames()
				if err != nil {
					t.Fatalf("nalLengthSize=%d second flush: %v", nalLengthSize, err)
				}
				if len(out) != 0 {
					t.Fatalf("nalLengthSize=%d second flush frames = %d, want 0", nalLengthSize, len(out))
				}
			}
		})
	}
}

func TestDecodeAutoConfiguredAVCHigh10DirectBFramesAcrossSamplesFlush(t *testing.T) {
	for _, tt := range high10DirectBFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := decodeHexFixture(t, tt.hex)
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
			assertHigh10DirectBFrames(t, frames, tt.frameMD5)
		})
	}
}

func TestFFmpegRawVideoMD5OracleHigh10DirectB(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}

	for _, tt := range high10DirectBFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := decodeHexFixture(t, tt.hex)
			assertHigh10DirectBFixtureSyntax(t, data, tt)
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
			for i, want := range tt.frameMD5 {
				line := []byte(fmt.Sprintf("0, %10d, %10d,        1, %8d, %s", i, i, high10DirectBFrameRawSize, want))
				if !bytes.Contains(framemd5Out, line) {
					t.Fatalf("frame[%d] missing %q in framemd5:\n%s", i, line, framemd5Out)
				}
			}

			rawCmd := exec.Command("ffmpeg",
				"-v", "error",
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
			wantSize := len(tt.frameMD5) * high10DirectBFrameRawSize
			if len(raw) != wantSize {
				t.Fatalf("rawvideo size = %d, want %d", len(raw), wantSize)
			}
			sum := md5.Sum(raw)
			if got := hex.EncodeToString(sum[:]); got != tt.rawVideoMD5 {
				t.Fatalf("rawvideo md5 = %s, want %s", got, tt.rawVideoMD5)
			}
		})
	}
}

func high10DirectBFixtures() []high10DirectBFixture {
	return []high10DirectBFixture{
		{
			name:          "temporal/cavlc",
			hex:           high10TemporalDirectBCAVLCAnnexBHex,
			cabac:         0,
			directSpatial: 0,
			annexBSize:    720,
			annexBMD5:     "1d30ac7b5a3aebfa9b360e43dd1747c1",
			frameMD5: []string{
				"dde20d70a08020b7171c068825ceab33",
				"6e6d6501898f05aa0f8efd391a783b25",
				"a4524920d19b25b23be978e8479039d0",
			},
			rawVideoMD5: "865b30bbd64725fd8bb720c0576e19d0",
		},
		{
			name:          "temporal/cabac",
			hex:           high10TemporalDirectBCABACAnnexBHex,
			cabac:         1,
			directSpatial: 0,
			annexBSize:    736,
			annexBMD5:     "9ed2b7d4183f1fbdee66af5a3124eac3",
			frameMD5: []string{
				"4737f86fe82079c689aec065ca6bb09f",
				"dc494068394c583d86e4650b4635d8c4",
				"4d9ce06c29c67bf8454164832e1ca92f",
			},
			rawVideoMD5: "779cd7a6b9f8555bf0930465ded641e2",
		},
		{
			name:          "spatial/cavlc",
			hex:           high10SpatialDirectBCAVLCAnnexBHex,
			cabac:         0,
			directSpatial: 1,
			annexBSize:    720,
			annexBMD5:     "d266bc4b06acc6835899d9e18fa6fa47",
			frameMD5: []string{
				"dde20d70a08020b7171c068825ceab33",
				"6e6d6501898f05aa0f8efd391a783b25",
				"a4524920d19b25b23be978e8479039d0",
			},
			rawVideoMD5: "865b30bbd64725fd8bb720c0576e19d0",
		},
		{
			name:          "spatial/cabac",
			hex:           high10SpatialDirectBCABACAnnexBHex,
			cabac:         1,
			directSpatial: 1,
			annexBSize:    736,
			annexBMD5:     "8c12df946dc2a5620753b3e81c000c4c",
			frameMD5: []string{
				"4737f86fe82079c689aec065ca6bb09f",
				"dc494068394c583d86e4650b4635d8c4",
				"4d9ce06c29c67bf8454164832e1ca92f",
			},
			rawVideoMD5: "779cd7a6b9f8555bf0930465ded641e2",
		},
	}
}

func assertHigh10DirectBFrames(t *testing.T, frames []*Frame, want []string) {
	t.Helper()
	assertHigh10FrameMD5Strings(t, frames, want)
	for i, frame := range frames {
		if frame.Width != 32 || frame.Height != 16 {
			t.Fatalf("frame[%d] size = %dx%d, want 32x16", i, frame.Width, frame.Height)
		}
		rawSize, err := frame.RawYUVSize()
		if err != nil {
			t.Fatalf("frame[%d] RawYUVSize: %v", i, err)
		}
		if rawSize != high10DirectBFrameRawSize {
			t.Fatalf("frame[%d] raw yuv size = %d, want %d", i, rawSize, high10DirectBFrameRawSize)
		}
	}
}

func assertHigh10DirectBFixtureSyntax(t *testing.T, data []byte, tt high10DirectBFixture) {
	t.Helper()
	nals, err := h264.SplitAnnexB(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(nals) != 5 {
		t.Fatalf("NAL count = %d, want stripped SPS/PPS/IDR/P/B", len(nals))
	}
	wantNALs := []h264.NALUnitType{
		h264.NALSPS,
		h264.NALPPS,
		h264.NALIDRSlice,
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
	for _, nal := range nals {
		switch nal.Type {
		case h264.NALSPS:
			sps, err := h264.DecodeSPS(nal.RBSP)
			if err != nil {
				t.Fatal(err)
			}
			if sps.ProfileIDC != 110 || sps.Width != 32 || sps.Height != 16 ||
				sps.ChromaFormatIDC != 1 || sps.BitDepthLuma != 10 || sps.BitDepthChroma != 10 ||
				sps.FrameMBSOnlyFlag != 1 || sps.MBAFF != 0 || sps.RefFrameCount != 2 ||
				sps.Direct8x8InferenceFlag == 0 {
				t.Fatalf("SPS profile/format = %d %dx%d chroma %d depth %d/%d frame_only %d/%d refs %d direct8x8 %d, want High10 32x16 yuv420p10le frame-only refs=2 direct8x8",
					sps.ProfileIDC, sps.Width, sps.Height, sps.ChromaFormatIDC, sps.BitDepthLuma, sps.BitDepthChroma,
					sps.FrameMBSOnlyFlag, sps.MBAFF, sps.RefFrameCount, sps.Direct8x8InferenceFlag)
			}
			spsList[sps.SPSID] = sps
		case h264.NALPPS:
			pps, err := h264.DecodePPS(nal.RBSP, &spsList)
			if err != nil {
				t.Fatal(err)
			}
			if pps.CABAC != tt.cabac || pps.Transform8x8Mode != 0 || pps.WeightedPred != 0 ||
				pps.WeightedBipredIDC != 0 || pps.RefCount[0] != 2 || pps.RefCount[1] != 1 {
				t.Fatalf("PPS cabac/8x8/weights/refs = %d/%d/%d/%d/%d/%d, want cabac=%d no-8x8 unweighted refs=2/1",
					pps.CABAC, pps.Transform8x8Mode, pps.WeightedPred, pps.WeightedBipredIDC, pps.RefCount[0], pps.RefCount[1], tt.cabac)
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
			switch sh.SliceTypeNoS {
			case h264.PictureTypeI:
				if sh.ListCount != 0 || sh.RefCount != ([2]uint32{}) {
					t.Fatalf("I slice lists/refs = %d/%v, want none", sh.ListCount, sh.RefCount)
				}
			case h264.PictureTypeP:
				if sh.ListCount != 1 || sh.RefCount[0] != 1 ||
					sh.PredWeightTable.UseWeight != 0 || sh.PredWeightTable.UseWeightChroma != 0 {
					t.Fatalf("P slice lists/refs/weights = %d/%v/%d/%d, want L0 refs=1 unweighted",
						sh.ListCount, sh.RefCount, sh.PredWeightTable.UseWeight, sh.PredWeightTable.UseWeightChroma)
				}
			case h264.PictureTypeB:
				if sh.ListCount != 2 || sh.RefCount[0] != 1 || sh.RefCount[1] != 1 ||
					sh.DirectSpatialMVPred != tt.directSpatial ||
					sh.PredWeightTable.UseWeight != 0 || sh.PredWeightTable.UseWeightChroma != 0 {
					t.Fatalf("B slice lists/refs/direct/weights = %d/%v/%d/%d/%d, want L0/L1 refs=1/1 direct=%d unweighted",
						sh.ListCount, sh.RefCount, sh.DirectSpatialMVPred, sh.PredWeightTable.UseWeight, sh.PredWeightTable.UseWeightChroma, tt.directSpatial)
				}
			default:
				t.Fatalf("unexpected slice type %d", sh.SliceTypeNoS)
			}
			gotSlices = append(gotSlices, sh.SliceTypeNoS)
		default:
			t.Fatalf("unexpected NAL type %d in stripped fixture", nal.Type)
		}
	}
	wantSlices := []int32{h264.PictureTypeI, h264.PictureTypeP, h264.PictureTypeB}
	if len(gotSlices) != len(wantSlices) {
		t.Fatalf("slice count = %d, want %d", len(gotSlices), len(wantSlices))
	}
	for i, want := range wantSlices {
		if gotSlices[i] != want {
			t.Fatalf("slice[%d] type = %d, want %d", i, gotSlices[i], want)
		}
	}
}
