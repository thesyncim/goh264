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

const high10PartitionedPFrameSize = 12288

type high10PartitionedPFixture struct {
	name         string
	path         string
	cabac        int32
	weighted     bool
	bitstreamMD5 string
	rawVideoMD5  string
	frameMD5     []string
}

func TestHigh10PartitionedPFixtureSyntax(t *testing.T) {
	for _, tt := range high10PartitionedPFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := readHigh10PartitionedPFixture(t, tt)
			assertHigh10PartitionedPFixtureSyntax(t, data, tt)
		})
	}
}

func TestDecodeAnnexBHigh10PartitionedPFrames(t *testing.T) {
	for _, tt := range high10PartitionedPFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := readHigh10PartitionedPFixture(t, tt)
			assertHigh10PartitionedPFixtureSyntax(t, data, tt)
			frames, err := NewDecoder().DecodeAnnexBFrames(data)
			if err != nil {
				t.Fatalf("DecodeAnnexBFrames: %v", err)
			}
			assertHigh10PartitionedPFrames(t, frames, tt)
		})
	}
}

func TestDecodeAVCHigh10PartitionedPFrames(t *testing.T) {
	for _, tt := range high10PartitionedPFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := readHigh10PartitionedPFixture(t, tt)
			assertHigh10PartitionedPFixtureSyntax(t, data, tt)
			for _, nalLengthSize := range []int{2, 3, 4} {
				frames, err := NewDecoder().DecodeAVCFrames(annexBToAVC(t, data, nalLengthSize), nalLengthSize)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: DecodeAVCFrames: %v", nalLengthSize, err)
				}
				assertHigh10PartitionedPFrames(t, frames, tt)
			}
		})
	}
}

func TestDecodeConfiguredAVCHigh10PartitionedPFrames(t *testing.T) {
	for _, tt := range high10PartitionedPFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := readHigh10PartitionedPFixture(t, tt)
			assertHigh10PartitionedPFixtureSyntax(t, data, tt)
			for _, nalLengthSize := range []int{2, 3, 4} {
				config, packet := annexBToAVCConfigAndPacket(t, data, nalLengthSize)
				frames, err := NewDecoder().DecodeAVCFramesWithConfigurationRecord(config, packet)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: DecodeAVCFramesWithConfigurationRecord: %v", nalLengthSize, err)
				}
				assertHigh10PartitionedPFrames(t, frames, tt)
			}
		})
	}
}

func TestDecodeConfiguredSamplesHigh10PartitionedPFlush(t *testing.T) {
	for _, tt := range high10PartitionedPFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := readHigh10PartitionedPFixture(t, tt)
			assertHigh10PartitionedPFixtureSyntax(t, data, tt)
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
				assertHigh10PartitionedPFrames(t, frames, tt)

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

func TestDecodeAutoConfiguredSamplesHigh10PartitionedPFlush(t *testing.T) {
	for _, tt := range high10PartitionedPFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := readHigh10PartitionedPFixture(t, tt)
			assertHigh10PartitionedPFixtureSyntax(t, data, tt)
			for _, nalLengthSize := range []int{2, 3, 4} {
				config, samples := annexBToAVCConfigAndSamples(t, data, nalLengthSize)
				dec := NewDecoder()
				out, err := dec.DecodeFrames(config)
				if err != nil {
					t.Fatalf("nalLengthSize=%d config: %v", nalLengthSize, err)
				}
				if len(out) != 0 {
					t.Fatalf("nalLengthSize=%d config frames = %d, want 0", nalLengthSize, len(out))
				}

				var frames []*Frame
				for i, sample := range samples {
					out, err = dec.DecodeFrames(sample)
					if err != nil {
						t.Fatalf("nalLengthSize=%d sample[%d]: DecodeFrames: %v", nalLengthSize, i, err)
					}
					frames = append(frames, out...)
				}
				out, err = dec.DecodeFrames(nil)
				if err != nil {
					t.Fatalf("nalLengthSize=%d nil flush: %v", nalLengthSize, err)
				}
				frames = append(frames, out...)
				assertHigh10PartitionedPFrames(t, frames, tt)

				out, err = dec.DecodeFrames(nil)
				if err != nil {
					t.Fatalf("nalLengthSize=%d second nil flush: %v", nalLengthSize, err)
				}
				if len(out) != 0 {
					t.Fatalf("nalLengthSize=%d second nil flush frames = %d, want 0", nalLengthSize, len(out))
				}
			}
		})
	}
}

func TestFFmpegRawVideoMD5OracleHigh10PartitionedP(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	for _, tt := range high10PartitionedPFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := readHigh10PartitionedPFixture(t, tt)
			assertHigh10PartitionedPFixtureSyntax(t, data, tt)

			framemd5 := exec.Command("ffmpeg",
				"-hide_banner", "-v", "error",
				"-f", "h264",
				"-i", tt.path,
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
				line := []byte(fmt.Sprintf("0, %10d, %10d,        1, %8d, %s", i, i, high10PartitionedPFrameSize, want))
				if !bytes.Contains(framemd5Out, line) {
					t.Fatalf("frame[%d] missing %q in framemd5:\n%s", i, line, framemd5Out)
				}
			}

			rawvideo := exec.Command("ffmpeg",
				"-hide_banner", "-v", "error",
				"-f", "h264",
				"-i", tt.path,
				"-an", "-sn", "-dn",
				"-pix_fmt", "yuv420p10le",
				"-f", "rawvideo",
				"-",
			)
			raw, err := rawvideo.Output()
			if err != nil {
				t.Fatalf("ffmpeg rawvideo: %v", err)
			}
			if len(raw) != len(tt.frameMD5)*high10PartitionedPFrameSize {
				t.Fatalf("rawvideo size = %d, want %d", len(raw), len(tt.frameMD5)*high10PartitionedPFrameSize)
			}
			sum := md5.Sum(raw)
			if got := hex.EncodeToString(sum[:]); got != tt.rawVideoMD5 {
				t.Fatalf("rawvideo md5 = %s, want %s", got, tt.rawVideoMD5)
			}
		})
	}
}

func high10PartitionedPFixtures() []high10PartitionedPFixture {
	return []high10PartitionedPFixture{
		{
			name:         "cavlc",
			path:         "testdata/h264/high10_partitioned_p_cavlc.h264",
			bitstreamMD5: "1855c563913e2b4372d655417d333cdd",
			rawVideoMD5:  "447dd2695f723fc336ddb1a6c0b710cc",
			frameMD5: []string{
				"1e10f859d4a3be85a0b4057dd7bff92c",
				"a87c5d14c468e549ae461bd63d21e7d6",
				"1e079388524aab8937783f56d36383c6",
				"360bf39f49dbbd060fdbb52e68f1c5ce",
				"d85d56ee1073b087635fcedb5d229025",
			},
		},
		{
			name:         "cabac",
			path:         "testdata/h264/high10_partitioned_p_cabac.h264",
			cabac:        1,
			bitstreamMD5: "4300b297e11dc082735c4f784c46ed62",
			rawVideoMD5:  "d37c9f22040bed0d61923dd6af57147a",
			frameMD5: []string{
				"51f65ff967216cfec2001d4b0ebadf38",
				"c7c7dd164303bafd485a7eeb33f7d653",
				"d65e16f831a92c6de68bfe0140f23c3c",
				"20973f9694657d0f2ff9f3a4b6a4da20",
				"03d3a4917e5158912f0472553cd143a8",
			},
		},
		{
			name:         "weighted-cavlc",
			path:         "testdata/h264/high10_weighted_partitioned_p_cavlc.h264",
			weighted:     true,
			bitstreamMD5: "beef107bee6bf1560ca46706a19deb3d",
			rawVideoMD5:  "de7c3027f1c8967f92c30782d356ab45",
			frameMD5: []string{
				"206e7a7a20a362c37b89ad4538db1ab9",
				"8e52768b6ae70aeef15c9ed9f0144165",
				"5dbe745d8d0fd254db46ae65cd7a7799",
				"340d98bfece7354be2a2114979c124e0",
				"442597be98d297fc3611930e116554d4",
			},
		},
		{
			name:         "weighted-cabac",
			path:         "testdata/h264/high10_weighted_partitioned_p_cabac.h264",
			cabac:        1,
			weighted:     true,
			bitstreamMD5: "ae88c99e3202f9a6d9c045868210a364",
			rawVideoMD5:  "7944a19fe843a899856ff2d24381f79e",
			frameMD5: []string{
				"206e7a7a20a362c37b89ad4538db1ab9",
				"f751694d20b5b4fc7a7cdf05aa01c379",
				"4b5ebed49101b07193ec6279bea3c59c",
				"0d82cbf526a25679ec75fbabea7629fd",
				"df01406ab3b43d769225eb67b8092055",
			},
		},
	}
}

func readHigh10PartitionedPFixture(t *testing.T, tt high10PartitionedPFixture) []byte {
	t.Helper()
	data, err := os.ReadFile(tt.path)
	if err != nil {
		t.Fatalf("read %s: %v", tt.path, err)
	}
	sum := md5.Sum(data)
	if got := hex.EncodeToString(sum[:]); got != tt.bitstreamMD5 {
		t.Fatalf("bitstream md5 = %s, want %s", got, tt.bitstreamMD5)
	}
	return data
}

func assertHigh10PartitionedPFixtureSyntax(t *testing.T, data []byte, tt high10PartitionedPFixture) {
	t.Helper()
	nals, err := h264.SplitAnnexB(data)
	if err != nil {
		t.Fatal(err)
	}

	var spsList [32]*h264.SPS
	var ppsList [256]*h264.PPS
	var gotVCL []h264.NALUnit
	var gotSlices []int32
	var weightedPSlices int
	var weightedChromaPSlices int
	for _, nal := range nals {
		switch nal.Type {
		case h264.NALSPS:
			sps, err := h264.DecodeSPS(nal.RBSP)
			if err != nil {
				t.Fatal(err)
			}
			if sps.ProfileIDC != 110 || sps.Width != 64 || sps.Height != 64 ||
				sps.ChromaFormatIDC != 1 || sps.BitDepthLuma != 10 || sps.BitDepthChroma != 10 ||
				sps.FrameMBSOnlyFlag != 1 || sps.RefFrameCount != 1 {
				t.Fatalf("SPS profile/format = %d %dx%d chroma %d depth %d/%d frame_mbs_only=%d refs=%d, want High10 64x64 yuv420p10le frame-only refs=1",
					sps.ProfileIDC, sps.Width, sps.Height, sps.ChromaFormatIDC, sps.BitDepthLuma, sps.BitDepthChroma, sps.FrameMBSOnlyFlag, sps.RefFrameCount)
			}
			spsList[sps.SPSID] = sps
		case h264.NALPPS:
			pps, err := h264.DecodePPS(nal.RBSP, &spsList)
			if err != nil {
				t.Fatal(err)
			}
			wantWeightedPred := int32(0)
			if tt.weighted {
				wantWeightedPred = 1
			}
			if pps.CABAC != tt.cabac || pps.Transform8x8Mode != 0 || pps.WeightedPred != wantWeightedPred ||
				pps.WeightedBipredIDC != 0 || pps.RefCount[0] != 1 || pps.RefCount[1] != 1 {
				t.Fatalf("PPS cabac/8x8/weights/refs = %d/%d/%d/%d/%d/%d, want %d/no-8x8/weighted=%d/ref=1",
					pps.CABAC, pps.Transform8x8Mode, pps.WeightedPred, pps.WeightedBipredIDC, pps.RefCount[0], pps.RefCount[1],
					tt.cabac, wantWeightedPred)
			}
			ppsList[pps.PPSID] = pps
		case h264.NALSEI:
		case h264.NALIDRSlice, h264.NALSlice:
			sh, err := h264.ParseSliceHeader(nal, &ppsList)
			if err != nil {
				t.Fatal(err)
			}
			if sh.PictureStructure != h264.PictureFrame || sh.DeblockingFilter != 0 {
				t.Fatalf("slice picture/deblock = %d/%d, want frame/disabled", sh.PictureStructure, sh.DeblockingFilter)
			}
			if sh.SliceTypeNoS == h264.PictureTypeP {
				if tt.weighted {
					if sh.PredWeightTable.UseWeight == 0 {
						t.Fatalf("P slice weights = %d/%d, want explicit luma weight", sh.PredWeightTable.UseWeight, sh.PredWeightTable.UseWeightChroma)
					}
					weightedPSlices++
					if sh.PredWeightTable.UseWeightChroma != 0 {
						weightedChromaPSlices++
					}
				} else if sh.PredWeightTable.UseWeight != 0 || sh.PredWeightTable.UseWeightChroma != 0 {
					t.Fatalf("slice weights = %d/%d, want unweighted P", sh.PredWeightTable.UseWeight, sh.PredWeightTable.UseWeightChroma)
				}
			}
			gotVCL = append(gotVCL, nal)
			gotSlices = append(gotSlices, sh.SliceTypeNoS)
		default:
			t.Fatalf("unexpected NAL type %d in High10 partitioned P fixture", nal.Type)
		}
	}
	if len(gotVCL) != len(tt.frameMD5) {
		t.Fatalf("VCL NAL count = %d, want %d", len(gotVCL), len(tt.frameMD5))
	}
	if gotVCL[0].Type != h264.NALIDRSlice {
		t.Fatalf("first VCL NAL = %d, want IDR", gotVCL[0].Type)
	}
	for i := 1; i < len(gotVCL); i++ {
		if gotVCL[i].Type != h264.NALSlice {
			t.Fatalf("VCL NAL[%d] = %d, want non-IDR slice", i, gotVCL[i].Type)
		}
	}
	if gotSlices[0] != h264.PictureTypeI {
		t.Fatalf("slice[0] = %d, want I", gotSlices[0])
	}
	for i := 1; i < len(gotSlices); i++ {
		if gotSlices[i] != h264.PictureTypeP {
			t.Fatalf("slice[%d] = %d, want P", i, gotSlices[i])
		}
	}
	if tt.weighted {
		if weightedPSlices != len(gotSlices)-1 {
			t.Fatalf("weighted P slices = %d, want %d", weightedPSlices, len(gotSlices)-1)
		}
		if weightedChromaPSlices == 0 {
			t.Fatalf("weighted fixture has no chroma-weighted P slices")
		}
	}
}

func assertHigh10PartitionedPFrames(t *testing.T, frames []*Frame, tt high10PartitionedPFixture) {
	t.Helper()
	if len(frames) != len(tt.frameMD5) {
		t.Fatalf("frames = %d, want %d", len(frames), len(tt.frameMD5))
	}
	rawHash := md5.New()
	for i, frame := range frames {
		if frame == nil {
			t.Fatalf("frame[%d] is nil", i)
		}
		if frame.Width != 64 || frame.Height != 64 || frame.ChromaFormatIDC != 1 ||
			frame.BitDepthLuma != 10 || frame.BitDepthChroma != 10 {
			t.Fatalf("frame[%d] format = %dx%d chroma %d depth %d/%d, want High10 64x64 yuv420p10le",
				i, frame.Width, frame.Height, frame.ChromaFormatIDC, frame.BitDepthLuma, frame.BitDepthChroma)
		}
		pixFmt, err := frame.RawPixelFormat()
		if err != nil {
			t.Fatalf("frame[%d] pix_fmt: %v", i, err)
		}
		if pixFmt != "yuv420p10le" {
			t.Fatalf("frame[%d] pix_fmt = %s, want yuv420p10le", i, pixFmt)
		}
		raw, err := frame.AppendRawYUVBytesLE(nil)
		if err != nil {
			t.Fatalf("frame[%d] raw yuv: %v", i, err)
		}
		if len(raw) != high10PartitionedPFrameSize {
			t.Fatalf("frame[%d] raw len = %d, want %d", i, len(raw), high10PartitionedPFrameSize)
		}
		sum := md5.Sum(raw)
		if got := hex.EncodeToString(sum[:]); got != tt.frameMD5[i] {
			t.Fatalf("frame[%d] md5 = %s, want %s", i, got, tt.frameMD5[i])
		}
		if _, err := rawHash.Write(raw); err != nil {
			t.Fatalf("frame[%d] raw hash: %v", i, err)
		}
	}
	if got := hex.EncodeToString(rawHash.Sum(nil)); got != tt.rawVideoMD5 {
		t.Fatalf("rawvideo md5 = %s, want %s", got, tt.rawVideoMD5)
	}
}
