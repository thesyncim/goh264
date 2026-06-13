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

const high10TemporalBSkipCAVLCAnnexBHex = `
00000001676e000aa6cedec044000003000400000300083c4894e0000000
0168ca8053c80000010605ffff72dc45e9bde6d948b7962cd820d923eeef
78323634202d20636f726520313635207233323232206233353630356120
2d20482e3236342f4d5045472d342041564320636f646563202d20436f70
796c65667420323030332d32303235202d20687474703a2f2f7777772e76
6964656f6c616e2e6f72672f783236342e68746d6c202d206f7074696f6e
733a2063616261633d30207265663d32206465626c6f636b3d303a303a30
20616e616c7973653d3078313a30206d653d686578207375626d653d3220
7073793d31207073795f72643d312e30303a302e3030206d697865645f72
65663d30206d655f72616e67653d3136206368726f6d615f6d653d312074
72656c6c69733d30203878386463743d302063716d3d3020646561647a6f
6e653d32312c313120666173745f70736b69703d31206368726f6d615f71
705f6f66667365743d3020746872656164733d31206c6f6f6b6168656164
5f746872656164733d3120736c696365645f746872656164733d30206e72
3d3020646563696d6174653d3120696e7465726c616365643d3020626c75
7261795f636f6d7061743d3020636f6e73747261696e65645f696e747261
3d3020626672616d65733d3120625f707972616d69643d3020625f616461
70743d3020625f626961733d30206469726563743d322077656967687462
3d30206f70656e5f676f703d3020776569676874703d30206b6579696e74
3d33206b6579696e745f6d696e3d32207363656e656375743d3020696e74
72615f726566726573683d302072633d637170206d62747265653d302071
703d31382069705f726174696f3d312e34302070625f726174696f3d312e
33302061713d30008000000165888403affffc3d140008fc00000001419a
2994a000000001019e44e11280
`

const high10SpatialBSkipCAVLCAnnexBHex = `
00000001676e000aa6cedec044000003000400000300083c4894e0000000
0168ca8053c80000010605ffff72dc45e9bde6d948b7962cd820d923eeef
78323634202d20636f726520313635207233323232206233353630356120
2d20482e3236342f4d5045472d342041564320636f646563202d20436f70
796c65667420323030332d32303235202d20687474703a2f2f7777772e76
6964656f6c616e2e6f72672f783236342e68746d6c202d206f7074696f6e
733a2063616261633d30207265663d32206465626c6f636b3d303a303a30
20616e616c7973653d3078313a30206d653d686578207375626d653d3220
7073793d31207073795f72643d312e30303a302e3030206d697865645f72
65663d30206d655f72616e67653d3136206368726f6d615f6d653d312074
72656c6c69733d30203878386463743d302063716d3d3020646561647a6f
6e653d32312c313120666173745f70736b69703d31206368726f6d615f71
705f6f66667365743d3020746872656164733d31206c6f6f6b6168656164
5f746872656164733d3120736c696365645f746872656164733d30206e72
3d3020646563696d6174653d3120696e7465726c616365643d3020626c75
7261795f636f6d7061743d3020636f6e73747261696e65645f696e747261
3d3020626672616d65733d3120625f707972616d69643d3020625f616461
70743d3020625f626961733d30206469726563743d312077656967687462
3d30206f70656e5f676f703d3020776569676874703d30206b6579696e74
3d33206b6579696e745f6d696e3d32207363656e656375743d3020696e74
72615f726566726573683d302072633d637170206d62747265653d302071
703d31382069705f726174696f3d312e34302070625f726174696f3d312e
33302061713d30008000000165888403affffc3d140008fc00000001419a
2994a000000001019e45e11280
`

const high10TemporalBSkipCABACAnnexBHex = `
00000001676e000aa6cedec044000003000400000300083c4894e0000000
0168ea8053c80000010605ffff72dc45e9bde6d948b7962cd820d923eeef
78323634202d20636f726520313635207233323232206233353630356120
2d20482e3236342f4d5045472d342041564320636f646563202d20436f70
796c65667420323030332d32303235202d20687474703a2f2f7777772e76
6964656f6c616e2e6f72672f783236342e68746d6c202d206f7074696f6e
733a2063616261633d31207265663d32206465626c6f636b3d303a303a30
20616e616c7973653d3078313a30206d653d686578207375626d653d3220
7073793d31207073795f72643d312e30303a302e3030206d697865645f72
65663d30206d655f72616e67653d3136206368726f6d615f6d653d312074
72656c6c69733d30203878386463743d302063716d3d3020646561647a6f
6e653d32312c313120666173745f70736b69703d31206368726f6d615f71
705f6f66667365743d3020746872656164733d31206c6f6f6b6168656164
5f746872656164733d3120736c696365645f746872656164733d30206e72
3d3020646563696d6174653d3120696e7465726c616365643d3020626c75
7261795f636f6d7061743d3020636f6e73747261696e65645f696e747261
3d3020626672616d65733d3120625f707972616d69643d3020625f616461
70743d3020625f626961733d30206469726563743d322077656967687462
3d30206f70656e5f676f703d3020776569676874703d30206b6579696e74
3d33206b6579696e745f6d696e3d32207363656e656375743d3020696e74
72615f726566726573683d302072633d637170206d62747265653d302071
703d31382069705f726174696f3d312e34302070625f726174696f3d312e
33302061713d30008000000165888403affa7fc2553fbbd11fff81000000
01419a299afee000000001019e44e48bb381
`

const high10SpatialBSkipCABACAnnexBHex = `
00000001676e000aa6cedec044000003000400000300083c4894e0000000
0168ea8053c80000010605ffff72dc45e9bde6d948b7962cd820d923eeef
78323634202d20636f726520313635207233323232206233353630356120
2d20482e3236342f4d5045472d342041564320636f646563202d20436f70
796c65667420323030332d32303235202d20687474703a2f2f7777772e76
6964656f6c616e2e6f72672f783236342e68746d6c202d206f7074696f6e
733a2063616261633d31207265663d32206465626c6f636b3d303a303a30
20616e616c7973653d3078313a30206d653d686578207375626d653d3220
7073793d31207073795f72643d312e30303a302e3030206d697865645f72
65663d30206d655f72616e67653d3136206368726f6d615f6d653d312074
72656c6c69733d30203878386463743d302063716d3d3020646561647a6f
6e653d32312c313120666173745f70736b69703d31206368726f6d615f71
705f6f66667365743d3020746872656164733d31206c6f6f6b6168656164
5f746872656164733d3120736c696365645f746872656164733d30206e72
3d3020646563696d6174653d3120696e7465726c616365643d3020626c75
7261795f636f6d7061743d3020636f6e73747261696e65645f696e747261
3d3020626672616d65733d3120625f707972616d69643d3020625f616461
70743d3020625f626961733d30206469726563743d312077656967687462
3d30206f70656e5f676f703d3020776569676874703d30206b6579696e74
3d33206b6579696e745f6d696e3d32207363656e656375743d3020696e74
72615f726566726573683d302072633d637170206d62747265653d302071
703d31382069705f726174696f3d312e34302070625f726174696f3d312e
33302061713d30008000000165888403affa7fc2553fbbd11fff81000000
01419a299afee000000001019e45e48bb381
`

const high10BSkipFrameRawSize = 768

type high10BSkipFixture struct {
	name          string
	path          string
	hex           string
	cabac         int32
	directSpatial int32
	annexBSize    int
	annexBMD5     string
	frameMD5      []string
	rawVideoMD5   string
}

func TestHigh10BSkipFixtureSyntax(t *testing.T) {
	for _, tt := range high10BSkipFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := decodeHexFixture(t, tt.hex)
			if len(data) != tt.annexBSize {
				t.Fatalf("annex b size = %d, want %d", len(data), tt.annexBSize)
			}
			sum := md5.Sum(data)
			if got := hex.EncodeToString(sum[:]); got != tt.annexBMD5 {
				t.Fatalf("annex b md5 = %s, want %s", got, tt.annexBMD5)
			}
			assertHigh10BSkipFixtureSyntax(t, data, tt)
		})
	}
}

func TestHigh10BSkipFileFixturesMatchEmbeddedAnnexB(t *testing.T) {
	for _, tt := range high10BSkipFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			embedded := decodeHexFixture(t, tt.hex)
			disk, err := os.ReadFile(tt.path)
			if err != nil {
				t.Fatalf("read %s: %v", tt.path, err)
			}
			if !bytes.Equal(disk, embedded) {
				diskSum := md5.Sum(disk)
				embeddedSum := md5.Sum(embedded)
				t.Fatalf("%s differs from embedded fixture: file len/md5=%d/%s embedded len/md5=%d/%s",
					tt.path, len(disk), hex.EncodeToString(diskSum[:]), len(embedded), hex.EncodeToString(embeddedSum[:]))
			}
			assertHigh10BSkipFixtureSyntax(t, disk, tt)
		})
	}
}

func TestDecodeAnnexBHigh10BSkipFrames(t *testing.T) {
	for _, tt := range high10BSkipFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := decodeHexFixture(t, tt.hex)
			assertHigh10BSkipFixtureSyntax(t, data, tt)
			frames, err := NewDecoder().DecodeAnnexBFrames(data)
			if err != nil {
				t.Fatalf("DecodeAnnexBFrames: %v", err)
			}
			assertHigh10BSkipFrames(t, frames, tt.frameMD5)
		})
	}
}

func TestDecodeAVCHigh10BSkipFrames(t *testing.T) {
	for _, tt := range high10BSkipFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := decodeHexFixture(t, tt.hex)
			for _, nalLengthSize := range []int{2, 3, 4} {
				frames, err := NewDecoder().DecodeAVCFrames(annexBToAVC(t, data, nalLengthSize), nalLengthSize)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: DecodeAVCFrames: %v", nalLengthSize, err)
				}
				assertHigh10BSkipFrames(t, frames, tt.frameMD5)
			}
		})
	}
}

func TestDecodeAVCCHigh10BSkipFrames(t *testing.T) {
	for _, tt := range high10BSkipFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := decodeHexFixture(t, tt.hex)
			for _, nalLengthSize := range []int{2, 3, 4} {
				config, packet := annexBToAVCConfigAndPacket(t, data, nalLengthSize)
				frames, err := NewDecoder().DecodeAVCCFrames(config, packet)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: DecodeAVCCFrames: %v", nalLengthSize, err)
				}
				assertHigh10BSkipFrames(t, frames, tt.frameMD5)
			}
		})
	}
}

func TestDecodeConfiguredAVCHigh10BSkipFramesAcrossSamplesFlush(t *testing.T) {
	for _, tt := range high10BSkipFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := decodeHexFixture(t, tt.hex)
			for _, nalLengthSize := range []int{2, 3, 4} {
				config, samples := annexBToAVCConfigAndSamples(t, data, nalLengthSize)
				if len(samples) != len(tt.frameMD5) {
					t.Fatalf("nalLengthSize=%d: samples = %d, want %d", nalLengthSize, len(samples), len(tt.frameMD5))
				}
				dec := NewDecoder()
				if _, err := dec.ConfigureAVCC(config); err != nil {
					t.Fatalf("nalLengthSize=%d config: %v", nalLengthSize, err)
				}
				var frames []*Frame
				for i, sample := range samples {
					out, err := dec.DecodeConfiguredAVCFrames(sample)
					if err != nil {
						t.Fatalf("nalLengthSize=%d sample[%d]: %v", nalLengthSize, i, err)
					}
					frames = append(frames, out...)
				}
				out, err := dec.FlushDelayedFrames()
				if err != nil {
					t.Fatalf("nalLengthSize=%d flush: %v", nalLengthSize, err)
				}
				frames = append(frames, out...)
				assertHigh10BSkipFrames(t, frames, tt.frameMD5)
			}
		})
	}
}

func TestDecodeAutoConfiguredAVCHigh10BSkipFramesAcrossSamplesFlush(t *testing.T) {
	for _, tt := range high10BSkipFixtures() {
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
			assertHigh10BSkipFrames(t, frames, tt.frameMD5)
		})
	}
}

func TestFFmpegRawVideoMD5OracleHigh10BSkip(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}

	for _, tt := range high10BSkipFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := decodeHexFixture(t, tt.hex)
			assertHigh10BSkipFixtureSyntax(t, data, tt)
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
				line := []byte(fmt.Sprintf("0, %10d, %10d,        1, %8d, %s", i, i, high10BSkipFrameRawSize, want))
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
			if len(raw) != len(tt.frameMD5)*high10BSkipFrameRawSize {
				t.Fatalf("rawvideo size = %d, want %d", len(raw), len(tt.frameMD5)*high10BSkipFrameRawSize)
			}
			sum := md5.Sum(raw)
			if got := hex.EncodeToString(sum[:]); got != tt.rawVideoMD5 {
				t.Fatalf("rawvideo md5 = %s, want %s", got, tt.rawVideoMD5)
			}
		})
	}
}

func high10BSkipFixtures() []high10BSkipFixture {
	return []high10BSkipFixture{
		{
			name:          "temporal/cavlc",
			path:          "testdata/h264/high10_bskip_temporal_cavlc.h264",
			hex:           high10TemporalBSkipCAVLCAnnexBHex,
			cabac:         0,
			directSpatial: 0,
			annexBSize:    703,
			annexBMD5:     "a3d29c7a7a11a5c9da642487de5a4c37",
			frameMD5: []string{
				"d73be6c1b3e4082e402d67d810323786",
				"d73be6c1b3e4082e402d67d810323786",
				"d73be6c1b3e4082e402d67d810323786",
			},
			rawVideoMD5: "bed8c5ab899fe974cae09585e60b151f",
		},
		{
			name:          "spatial/cavlc",
			path:          "testdata/h264/high10_bskip_spatial_cavlc.h264",
			hex:           high10SpatialBSkipCAVLCAnnexBHex,
			cabac:         0,
			directSpatial: 1,
			annexBSize:    703,
			annexBMD5:     "4ae312697d364153195deec6da9a1973",
			frameMD5: []string{
				"d73be6c1b3e4082e402d67d810323786",
				"d73be6c1b3e4082e402d67d810323786",
				"d73be6c1b3e4082e402d67d810323786",
			},
			rawVideoMD5: "bed8c5ab899fe974cae09585e60b151f",
		},
		{
			name:          "temporal/cabac",
			path:          "testdata/h264/high10_bskip_temporal_cabac.h264",
			hex:           high10TemporalBSkipCABACAnnexBHex,
			cabac:         1,
			directSpatial: 0,
			annexBSize:    708,
			annexBMD5:     "74a9b632842600c57c0e20c03800c772",
			frameMD5: []string{
				"d73be6c1b3e4082e402d67d810323786",
				"d73be6c1b3e4082e402d67d810323786",
				"d73be6c1b3e4082e402d67d810323786",
			},
			rawVideoMD5: "bed8c5ab899fe974cae09585e60b151f",
		},
		{
			name:          "spatial/cabac",
			path:          "testdata/h264/high10_bskip_spatial_cabac.h264",
			hex:           high10SpatialBSkipCABACAnnexBHex,
			cabac:         1,
			directSpatial: 1,
			annexBSize:    708,
			annexBMD5:     "961a79bdc2278420951d4662a1a2c2f3",
			frameMD5: []string{
				"d73be6c1b3e4082e402d67d810323786",
				"d73be6c1b3e4082e402d67d810323786",
				"d73be6c1b3e4082e402d67d810323786",
			},
			rawVideoMD5: "bed8c5ab899fe974cae09585e60b151f",
		},
	}
}

func assertHigh10BSkipFrames(t *testing.T, frames []*Frame, want []string) {
	t.Helper()
	assertHigh10FrameMD5Strings(t, frames, want)
	for i, frame := range frames {
		pixFmt, err := frame.RawPixelFormat()
		if err != nil {
			t.Fatalf("frame[%d] RawPixelFormat: %v", i, err)
		}
		if frame.Width != 16 || frame.Height != 16 || pixFmt != "yuv420p10le" {
			t.Fatalf("frame[%d] geometry/pixfmt = %dx%d %s, want 16x16 yuv420p10le",
				i, frame.Width, frame.Height, pixFmt)
		}
		rawSize, err := frame.RawYUVSize()
		if err != nil {
			t.Fatalf("frame[%d] RawYUVSize: %v", i, err)
		}
		if rawSize != high10BSkipFrameRawSize {
			t.Fatalf("frame[%d] raw yuv size = %d, want %d", i, rawSize, high10BSkipFrameRawSize)
		}
	}
}

func assertHigh10BSkipFixtureSyntax(t *testing.T, data []byte, tt high10BSkipFixture) {
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
				sps.FrameMBSOnlyFlag != 1 || sps.MBAFF != 0 || sps.RefFrameCount != 2 ||
				sps.Direct8x8InferenceFlag == 0 {
				t.Fatalf("SPS profile/format = %d %dx%d chroma %d depth %d/%d frame_only %d/%d refs %d direct8x8 %d, want High10 16x16 yuv420p10le frame-only refs=2 direct8x8",
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
					sh.DirectSpatialMVPred != tt.directSpatial ||
					sh.PredWeightTable.UseWeight != 0 || sh.PredWeightTable.UseWeightChroma != 0 {
					t.Fatalf("B slice lists/refs/direct/weights = %d/%v/%d/%d/%d, want L0/L1 refs=1/1 direct=%d unweighted",
						sh.ListCount, sh.RefCount, sh.DirectSpatialMVPred, sh.PredWeightTable.UseWeight, sh.PredWeightTable.UseWeightChroma, tt.directSpatial)
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
