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

const high10PartitionedBFrameRawSize = 768

type high10PartitionedBFixture struct {
	name        string
	file        string
	cabac       int32
	annexBSize  int
	annexBMD5   string
	frameMD5    []string
	rawVideoMD5 string
}

func TestHigh10PartitionedBFixtureSyntax(t *testing.T) {
	for _, tt := range high10PartitionedBFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := readHigh10PartitionedBFixture(t, tt)
			if len(data) != tt.annexBSize {
				t.Fatalf("annex b size = %d, want %d", len(data), tt.annexBSize)
			}
			sum := md5.Sum(data)
			if got := hex.EncodeToString(sum[:]); got != tt.annexBMD5 {
				t.Fatalf("annex b md5 = %s, want %s", got, tt.annexBMD5)
			}
			assertHigh10PartitionedBFixtureSyntax(t, data, tt)
		})
	}
}

func TestDecodeAnnexBHigh10PartitionedBFrames(t *testing.T) {
	for _, tt := range high10PartitionedBFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := readHigh10PartitionedBFixture(t, tt)
			assertHigh10PartitionedBFixtureSyntax(t, data, tt)
			frames, err := NewDecoder().DecodeAnnexBFrames(data)
			if err != nil {
				t.Fatalf("DecodeAnnexBFrames: %v", err)
			}
			assertHigh10PartitionedBFrames(t, frames, tt)
		})
	}
}

func TestDecodeAVCHigh10PartitionedBFrames(t *testing.T) {
	for _, tt := range high10PartitionedBFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := readHigh10PartitionedBFixture(t, tt)
			assertHigh10PartitionedBFixtureSyntax(t, data, tt)
			for _, nalLengthSize := range []int{2, 3, 4} {
				frames, err := NewDecoder().DecodeAVCFrames(annexBToAVC(t, data, nalLengthSize), nalLengthSize)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: DecodeAVCFrames: %v", nalLengthSize, err)
				}
				assertHigh10PartitionedBFrames(t, frames, tt)
			}
		})
	}
}

func TestDecodeAVCWithConfigurationRecordHigh10PartitionedBFrames(t *testing.T) {
	for _, tt := range high10PartitionedBFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := readHigh10PartitionedBFixture(t, tt)
			assertHigh10PartitionedBFixtureSyntax(t, data, tt)
			for _, nalLengthSize := range []int{2, 3, 4} {
				config, packet := annexBToAVCConfigAndPacket(t, data, nalLengthSize)
				frames, err := NewDecoder().DecodeAVCFramesWithConfigurationRecord(config, packet)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: DecodeAVCFramesWithConfigurationRecord: %v", nalLengthSize, err)
				}
				assertHigh10PartitionedBFrames(t, frames, tt)
			}
		})
	}
}

func TestDecodeConfiguredAVCHigh10PartitionedBFramesAcrossSamplesFlush(t *testing.T) {
	for _, tt := range high10PartitionedBFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := readHigh10PartitionedBFixture(t, tt)
			assertHigh10PartitionedBFixtureSyntax(t, data, tt)
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
				assertHigh10PartitionedBFrames(t, frames, tt)
			}
		})
	}
}

func TestDecodeAutoConfiguredAVCHigh10PartitionedBFramesAcrossSamplesFlush(t *testing.T) {
	for _, tt := range high10PartitionedBFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := readHigh10PartitionedBFixture(t, tt)
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
			assertHigh10PartitionedBFrames(t, frames, tt)
		})
	}
}

func TestFFmpegRawVideoMD5OracleHigh10PartitionedB(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}

	for _, tt := range high10PartitionedBFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join("testdata", "h264", tt.file)
			framemd5 := exec.Command("ffmpeg",
				"-hide_banner", "-v", "error",
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
				line := []byte(fmt.Sprintf("0, %10d, %10d,        1, %8d, %s", i, i, high10PartitionedBFrameRawSize, want))
				if !bytes.Contains(framemd5Out, line) {
					t.Fatalf("frame[%d] missing %q in framemd5:\n%s", i, line, framemd5Out)
				}
			}

			rawCmd := exec.Command("ffmpeg",
				"-hide_banner", "-v", "error",
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
			if len(raw) != len(tt.frameMD5)*high10PartitionedBFrameRawSize {
				t.Fatalf("rawvideo size = %d, want %d", len(raw), len(tt.frameMD5)*high10PartitionedBFrameRawSize)
			}
			sum := md5.Sum(raw)
			if got := hex.EncodeToString(sum[:]); got != tt.rawVideoMD5 {
				t.Fatalf("rawvideo md5 = %s, want %s", got, tt.rawVideoMD5)
			}
		})
	}
}

func high10PartitionedBFixtures() []high10PartitionedBFixture {
	return []high10PartitionedBFixture{
		{
			name:       "cavlc-b16x8",
			file:       "high10_partitioned_b16x8_cavlc.h264",
			annexBSize: 739,
			annexBMD5:  "2798d9490dcb9f4b1495faee8e23c998",
			frameMD5: []string{
				"da42dbbc6702ac820c7162dd19030ea3",
				"6dc0b7afff881b7f69b9176db6c5155e",
				"ae723753e3ae671a34e4f57f325d2cb8",
			},
			rawVideoMD5: "8057ca8e0ee9e2f51fc59b824333e0da",
		},
		{
			name:       "cabac-b16x8",
			file:       "high10_partitioned_b16x8_cabac.h264",
			cabac:      1,
			annexBSize: 741,
			annexBMD5:  "74d9dd3315d2a1b45406508786722c25",
			frameMD5: []string{
				"da42dbbc6702ac820c7162dd19030ea3",
				"6dc0b7afff881b7f69b9176db6c5155e",
				"ae723753e3ae671a34e4f57f325d2cb8",
			},
			rawVideoMD5: "8057ca8e0ee9e2f51fc59b824333e0da",
		},
		{
			name:       "cavlc-b8x16",
			file:       "high10_partitioned_b8x16_cavlc.h264",
			annexBSize: 739,
			annexBMD5:  "8f041ebd2075c5ee3195c6e4ea197d69",
			frameMD5: []string{
				"3de0d9ec87d2b43d34b08554de5509e0",
				"6dc0b7afff881b7f69b9176db6c5155e",
				"360499a4bb17c8730018ce06b58180b7",
			},
			rawVideoMD5: "d927d8a41788f89e93b8d66d54347ec7",
		},
		{
			name:       "cabac-b8x16",
			file:       "high10_partitioned_b8x16_cabac.h264",
			cabac:      1,
			annexBSize: 741,
			annexBMD5:  "0b7b7c3094532f5fff464f7a3819635a",
			frameMD5: []string{
				"3de0d9ec87d2b43d34b08554de5509e0",
				"6dc0b7afff881b7f69b9176db6c5155e",
				"360499a4bb17c8730018ce06b58180b7",
			},
			rawVideoMD5: "d927d8a41788f89e93b8d66d54347ec7",
		},
		{
			name:       "cavlc-b8x8",
			file:       "high10_partitioned_b8x8_cavlc.h264",
			annexBSize: 1406,
			annexBMD5:  "9bd955daf127957bc6684c012a91df6a",
			frameMD5: []string{
				"41ea931c1df0c87907ca7627beeb1dfc",
				"ca7db1692b52de6fd7be03eae5d6b121",
				"e355a7851b20224a769b798c9a63c8b3",
			},
			rawVideoMD5: "017a85619aefcae9c7c98f11f6b829ee",
		},
		{
			name:       "cabac-b8x8",
			file:       "high10_partitioned_b8x8_cabac.h264",
			cabac:      1,
			annexBSize: 1438,
			annexBMD5:  "880484c1f22f9ac1846f5f9cd7652917",
			frameMD5: []string{
				"541565314ead228ebda2b21fc3ee25d6",
				"ccd7e4a2a29432b1db826acd229b78cd",
				"730d70dba915767dc72964eb71a28ae4",
			},
			rawVideoMD5: "63bbee01f26a0382dd58777ccb6c05e3",
		},
	}
}

func readHigh10PartitionedBFixture(t *testing.T, tt high10PartitionedBFixture) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "h264", tt.file))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func assertHigh10PartitionedBFrames(t *testing.T, frames []*Frame, tt high10PartitionedBFixture) {
	t.Helper()
	if len(frames) != len(tt.frameMD5) {
		t.Fatalf("frames = %d, want %d", len(frames), len(tt.frameMD5))
	}
	assertHigh10FrameMD5Strings(t, frames, tt.frameMD5)
	raw := make([]byte, 0, len(frames)*high10PartitionedBFrameRawSize)
	for i, frame := range frames {
		pixFmt, err := frame.RawPixelFormat()
		if err != nil {
			t.Fatalf("frame[%d] RawPixelFormat: %v", i, err)
		}
		bytesPerSample, err := frame.BytesPerSample()
		if err != nil {
			t.Fatalf("frame[%d] BytesPerSample: %v", i, err)
		}
		if pixFmt != "yuv420p10le" || bytesPerSample != 2 {
			t.Fatalf("frame[%d] raw format/sample bytes = %s/%d, want yuv420p10le/2", i, pixFmt, bytesPerSample)
		}
		raw, err = frame.AppendRawYUVBytesLE(raw)
		if err != nil {
			t.Fatalf("frame[%d] append raw yuv: %v", i, err)
		}
		rawSize, err := frame.RawYUVSize()
		if err != nil {
			t.Fatalf("frame[%d] RawYUVSize: %v", i, err)
		}
		if rawSize != high10PartitionedBFrameRawSize {
			t.Fatalf("frame[%d] raw yuv size = %d, want %d", i, rawSize, high10PartitionedBFrameRawSize)
		}
	}
	sum := md5.Sum(raw)
	if got := hex.EncodeToString(sum[:]); got != tt.rawVideoMD5 {
		t.Fatalf("rawvideo md5 = %s, want %s", got, tt.rawVideoMD5)
	}
}

func assertHigh10PartitionedBFixtureSyntax(t *testing.T, data []byte, tt high10PartitionedBFixture) {
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
					sh.PredWeightTable.UseWeight != 0 || sh.PredWeightTable.UseWeightChroma != 0 {
					t.Fatalf("B slice lists/refs/weights = %d/%v/%d/%d, want L0/L1 refs=1/1 unweighted",
						sh.ListCount, sh.RefCount, sh.PredWeightTable.UseWeight, sh.PredWeightTable.UseWeightChroma)
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
