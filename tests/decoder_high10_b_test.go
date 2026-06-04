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

const high10NonDirectBCAVLCAnnexBHex = `
00000001676e000aa6cedec044000003000400000300083c4894e00000000168ca808cb2000001658884032c431520102fcc02146cc001eb50618003c2f4e2f2194bc35277ff81a11d900d0073811a0ead5e6a41918a25ae77eb3889867421b1520eb6705a82ea70c7e47940a3d5ec0d0e1752eae037285fdf5a87b1c114d58d91eaf5d270c742bccf3d5ea00f98c66c70055affbe581e291d294048cdfef830020002015cc30003608022001be3734a420fc71ca91e09589ea0ad27d8691b7100eca4feeb402878c1a0ccbaf7700198170a34cb3841b0971f0594fac7d688cc49ac70ceb810ffec01cad1e8b46c4d1ffe8000000001419a2982d60840aa36008a6c331800040deb539401e280033c03eb5100820000803a001617a2be9845369ad0002fffc4c200020bec805d01a140084ed6ae18b415c86916df1bc00afffb485ce2e29d4bc35e27bfc1c104000100741ae12a4145f3d1d9405741c104000100b41ec1c1d15739c90767a5c9e2868c288e7e79260cbcd8e26830ea64a4569e6e003e60986e652c2f6bfd27a071cca5e8a5035b540382080141a000234e10403802c15000e5e864d4e438f85a340d58d7a281071381891d2d55bc1649c9364e8005c5189656c2c033c2a80078ce0bd2d0af5aa6b74e08ec36c260d0149cb0882b62318dc4b876a33c7fcd6064637828c5599dc55a00000001019e44e549e1d42e008a03c700448456a27c791fbdf817c269193ae5bbe08000d81cd3c222c80012444105239e252224d70e589d36c84df73f80078c72632d1042aec45fc9c6f52e76bcfb44e0
`

const high10NonDirectBCABACAnnexBHex = `
00000001676e000aa6cedec044000003000400000300083c4894e00000000168ea808cb2000001658884032fb332dda267da50b671b3794ccd4d4fb4fc79d480208e15e3bcd02734f3b6f1741fe6615d15268249b28ea14f42f17a94c111aafddd22d7c668ab3be26e65c65785ecda7647d8e437afa8aab79ec73885f1900234d1bba65b99e6a58732fe01afe3356eb05f6503de4946fff35625d9d3752f595aa2c376aa3e4e9ce86686d3e62ec7c7db6dfea3d0b41b9b00a9aa4fe2406a267d85da5e831c106496b50faccfcec1b391cadf073633759bcf176ade4ef946a3466d0511d684049fcc993e2ab0d75f37696a51ea84cd2e8cc5e794ae54cbefcd0380b494ab9d00000001419a29916b815b76bbeb9cc38fa68d45c65117434a85a686ae6cdd90604872fa9846fc4f6eaeb31f87aec81ae9d4ebcf0b5b1616514300c22224ab43ab4980bcd92cd25dad3cc1983ee2c332c86e244a8a8583e812d2311ba0ea45bc3277ac26a948d4ccbd09b49994d6634c1eb110e83f334fec0f613d098ac101d14b64f946efee32616ba4c0bc6a1b64f290006c42f938f0383d60c34cfd297ece5fa77ce8ff2dff0e8df5c4c96f91af847aa324ffdb7eda0925d9e874b0a34853f5df909307deccaf210066087ef8e4e0d395811d97edf1d20f3405e8eae0b5818ac59068744974de6da663b1948bd0b545705e00000001019e44e6bff09b310553daedc59211cbd9299b0c4a9eef737579987d123bd1c4ecfc341c4a14b3b7c922c91ffa15eaf60d236922235952471b6f023e00c3c8b28e688e2ffe57055152da9d6abfc89fec32e53581
`

const high10NonDirectBFrameRawSize = 768

type high10NonDirectBFixture struct {
	name        string
	hex         string
	cabac       int32
	annexBSize  int
	annexBMD5   string
	frameMD5    []string
	rawVideoMD5 string
}

func TestHigh10NonDirectBFixtureSyntax(t *testing.T) {
	for _, tt := range high10NonDirectBFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := decodeHexFixture(t, tt.hex)
			if len(data) != tt.annexBSize {
				t.Fatalf("annex b size = %d, want %d", len(data), tt.annexBSize)
			}
			sum := md5.Sum(data)
			if got := hex.EncodeToString(sum[:]); got != tt.annexBMD5 {
				t.Fatalf("annex b md5 = %s, want %s", got, tt.annexBMD5)
			}
			assertHigh10NonDirectBFixtureSyntax(t, data, tt.cabac)
		})
	}
}

func TestDecodeAnnexBHigh10NonDirectBFrames(t *testing.T) {
	for _, tt := range high10NonDirectBFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := decodeHexFixture(t, tt.hex)
			assertHigh10NonDirectBFixtureSyntax(t, data, tt.cabac)
			frames, err := NewDecoder().DecodeAnnexBFrames(data)
			if err != nil {
				t.Fatalf("DecodeAnnexBFrames: %v", err)
			}
			assertHigh10NonDirectBFrames(t, frames, tt.frameMD5)
		})
	}
}

func TestDecodeAVCHigh10NonDirectBFrames(t *testing.T) {
	for _, tt := range high10NonDirectBFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := decodeHexFixture(t, tt.hex)
			assertHigh10NonDirectBFixtureSyntax(t, data, tt.cabac)
			for _, nalLengthSize := range []int{2, 3, 4} {
				frames, err := NewDecoder().DecodeAVCFrames(annexBToAVC(t, data, nalLengthSize), nalLengthSize)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: DecodeAVCFrames: %v", nalLengthSize, err)
				}
				assertHigh10NonDirectBFrames(t, frames, tt.frameMD5)
			}
		})
	}
}

func TestDecodeAVCWithConfigurationRecordHigh10NonDirectBFrames(t *testing.T) {
	for _, tt := range high10NonDirectBFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := decodeHexFixture(t, tt.hex)
			assertHigh10NonDirectBFixtureSyntax(t, data, tt.cabac)
			for _, nalLengthSize := range []int{2, 3, 4} {
				config, packet := annexBToAVCConfigAndPacket(t, data, nalLengthSize)
				frames, err := NewDecoder().DecodeAVCFramesWithConfigurationRecord(config, packet)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: DecodeAVCFramesWithConfigurationRecord: %v", nalLengthSize, err)
				}
				assertHigh10NonDirectBFrames(t, frames, tt.frameMD5)
			}
		})
	}
}

func TestDecodeConfiguredAVCHigh10NonDirectBFramesAcrossSamplesFlush(t *testing.T) {
	for _, tt := range high10NonDirectBFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := decodeHexFixture(t, tt.hex)
			assertHigh10NonDirectBFixtureSyntax(t, data, tt.cabac)
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
				var frameCounts []int
				for i, sample := range samples {
					out, err := dec.DecodeConfiguredAVCFrames(sample)
					if err != nil {
						t.Fatalf("nalLengthSize=%d sample[%d]: DecodeConfiguredAVCFrames: %v", nalLengthSize, i, err)
					}
					frameCounts = append(frameCounts, len(out))
					frames = append(frames, out...)
				}
				out, err := dec.FlushDelayedFrames()
				if err != nil {
					t.Fatalf("nalLengthSize=%d flush: %v", nalLengthSize, err)
				}
				frameCounts = append(frameCounts, len(out))
				assertHigh10NonDirectBConfiguredSampleCounts(t, tt.name, nalLengthSize, frameCounts)
				frames = append(frames, out...)
				assertHigh10NonDirectBFrames(t, frames, tt.frameMD5)

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

func assertHigh10NonDirectBConfiguredSampleCounts(t *testing.T, name string, nalLengthSize int, got []int) {
	t.Helper()
	want := []int{0, 1, 1, 1}
	if len(got) != len(want) {
		t.Fatalf("%s nalLengthSize=%d configured sample/flush counts = %v, want %v", name, nalLengthSize, got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%s nalLengthSize=%d configured sample/flush counts = %v, want %v", name, nalLengthSize, got, want)
		}
	}
}

func TestDecodeAutoConfiguredAVCHigh10NonDirectBFramesAcrossSamplesFlush(t *testing.T) {
	for _, tt := range high10NonDirectBFixtures() {
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
			assertHigh10NonDirectBFrames(t, frames, tt.frameMD5)
		})
	}
}

func TestFFmpegRawVideoMD5OracleHigh10NonDirectB(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}

	for _, tt := range high10NonDirectBFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := decodeHexFixture(t, tt.hex)
			assertHigh10NonDirectBFixtureSyntax(t, data, tt.cabac)
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
				line := []byte(fmt.Sprintf("0, %10d, %10d,        1, %8d, %s", i, i, high10NonDirectBFrameRawSize, want))
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
			wantSize := len(tt.frameMD5) * high10NonDirectBFrameRawSize
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

func high10NonDirectBFixtures() []high10NonDirectBFixture {
	return []high10NonDirectBFixture{
		{
			name:       "cavlc",
			hex:        high10NonDirectBCAVLCAnnexBHex,
			cabac:      0,
			annexBSize: 582,
			annexBMD5:  "5a18eb8a8156a259ae2c3c915116fd7f",
			frameMD5: []string{
				"95893f95fdce0f45e7593f4eca8bd834",
				"9e8ad599e09f708487e0614412596665",
				"b7edf8a2678e03b0495ba6a6efebc063",
			},
			rawVideoMD5: "1ccf5f80b965f0e5788e592b2496e432",
		},
		{
			name:       "cabac",
			hex:        high10NonDirectBCABACAnnexBHex,
			cabac:      1,
			annexBSize: 592,
			annexBMD5:  "0067912e1f4bb582a1a6accf6930ab8d",
			frameMD5: []string{
				"b43174bc46328c029e698e5b27960dcd",
				"8b7a30d943aeacb4c000a53bb1dbc212",
				"6c997570b55af8ecd2ad29fbf56386a3",
			},
			rawVideoMD5: "70c7595de7146ac9b0aec7a2cf2d116b",
		},
	}
}

func assertHigh10NonDirectBFrames(t *testing.T, frames []*Frame, want []string) {
	t.Helper()
	assertHigh10FrameMD5Strings(t, frames, want)
	for i, frame := range frames {
		if frame.Width != 16 || frame.Height != 16 {
			t.Fatalf("frame[%d] size = %dx%d, want 16x16", i, frame.Width, frame.Height)
		}
		rawSize, err := frame.RawYUVSize()
		if err != nil {
			t.Fatalf("frame[%d] RawYUVSize: %v", i, err)
		}
		if rawSize != high10NonDirectBFrameRawSize {
			t.Fatalf("frame[%d] raw yuv size = %d, want %d", i, rawSize, high10NonDirectBFrameRawSize)
		}
	}
}

func assertHigh10NonDirectBFixtureSyntax(t *testing.T, data []byte, cabac int32) {
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
	var gotB bool
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
			if pps.CABAC != cabac || pps.Transform8x8Mode != 0 || pps.WeightedPred != 0 ||
				pps.WeightedBipredIDC != 0 || pps.RefCount[0] != 2 || pps.RefCount[1] != 1 {
				t.Fatalf("PPS cabac/8x8/weights/refs = %d/%d/%d/%d/%d/%d, want cabac=%d no-8x8 unweighted refs=2/1",
					pps.CABAC, pps.Transform8x8Mode, pps.WeightedPred, pps.WeightedBipredIDC, pps.RefCount[0], pps.RefCount[1], cabac)
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
				gotB = true
				if sh.ListCount != 2 || sh.RefCount[0] != 1 || sh.RefCount[1] != 1 ||
					sh.PredWeightTable.UseWeight != 0 || sh.PredWeightTable.UseWeightChroma != 0 {
					t.Fatalf("B slice lists/refs/weights = %d/%v/%d/%d, want L0/L1 refs=1/1 unweighted",
						sh.ListCount, sh.RefCount, sh.PredWeightTable.UseWeight, sh.PredWeightTable.UseWeightChroma)
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
	if !gotB {
		t.Fatal("fixture has no B slice")
	}
}
