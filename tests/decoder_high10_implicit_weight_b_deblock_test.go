// SPDX-License-Identifier: LGPL-2.1-or-later

package goh264_test

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

const high10ImplicitWeightedBDeblockFrameRawSize = 1536

type high10ImplicitWeightedBDeblockFixture struct {
	name        string
	file        string
	cabac       int32
	annexBSize  int
	annexBMD5   string
	rawVideoMD5 string
	frameMD5    []string
}

func TestHigh10ImplicitWeightedBDeblockFixtureSyntax(t *testing.T) {
	for _, tt := range high10ImplicitWeightedBDeblockFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := readHigh10ImplicitWeightedBDeblockFixture(t, tt)
			if len(data) != tt.annexBSize {
				t.Fatalf("annex b size = %d, want %d", len(data), tt.annexBSize)
			}
			sum := md5.Sum(data)
			if got := hex.EncodeToString(sum[:]); got != tt.annexBMD5 {
				t.Fatalf("annex b md5 = %s, want %s", got, tt.annexBMD5)
			}
			assertHigh10ImplicitWeightedBDeblockFixtureSyntax(t, data, tt)
		})
	}
}

func TestDecodeAnnexBHigh10ImplicitWeightedBDeblockFrames(t *testing.T) {
	for _, tt := range high10ImplicitWeightedBDeblockFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := readHigh10ImplicitWeightedBDeblockFixture(t, tt)
			assertHigh10ImplicitWeightedBDeblockFixtureSyntax(t, data, tt)
			frames, err := NewDecoder().DecodeAnnexBFrames(data)
			if err != nil {
				t.Fatalf("DecodeAnnexBFrames: %v", err)
			}
			assertHigh10ImplicitWeightedBDeblockFrames(t, frames, tt)
		})
	}
}

func TestDecodeAVCHigh10ImplicitWeightedBDeblockFrames(t *testing.T) {
	for _, tt := range high10ImplicitWeightedBDeblockFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := readHigh10ImplicitWeightedBDeblockFixture(t, tt)
			assertHigh10ImplicitWeightedBDeblockFixtureSyntax(t, data, tt)
			for _, nalLengthSize := range []int{2, 3, 4} {
				frames, err := NewDecoder().DecodeAVCFrames(annexBToAVC(t, data, nalLengthSize), nalLengthSize)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: DecodeAVCFrames: %v", nalLengthSize, err)
				}
				assertHigh10ImplicitWeightedBDeblockFrames(t, frames, tt)
			}
		})
	}
}

func TestDecodeAVCCHigh10ImplicitWeightedBDeblockFrames(t *testing.T) {
	for _, tt := range high10ImplicitWeightedBDeblockFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := readHigh10ImplicitWeightedBDeblockFixture(t, tt)
			assertHigh10ImplicitWeightedBDeblockFixtureSyntax(t, data, tt)
			for _, nalLengthSize := range []int{2, 3, 4} {
				config, packet := annexBToAVCConfigAndPacket(t, data, nalLengthSize)
				frames, err := NewDecoder().DecodeAVCCFrames(config, packet)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: DecodeAVCCFrames: %v", nalLengthSize, err)
				}
				assertHigh10ImplicitWeightedBDeblockFrames(t, frames, tt)
			}
		})
	}
}

func TestDecodeConfiguredAVCHigh10ImplicitWeightedBDeblockFramesAcrossSamplesFlush(t *testing.T) {
	for _, tt := range high10ImplicitWeightedBDeblockFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := readHigh10ImplicitWeightedBDeblockFixture(t, tt)
			assertHigh10ImplicitWeightedBDeblockFixtureSyntax(t, data, tt)
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
				assertHigh10ImplicitWeightedBDeblockFrames(t, frames, tt)
			}
		})
	}
}

func TestDecodeAutoConfiguredAVCHigh10ImplicitWeightedBDeblockFramesAcrossSamplesFlush(t *testing.T) {
	for _, tt := range high10ImplicitWeightedBDeblockFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := readHigh10ImplicitWeightedBDeblockFixture(t, tt)
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
			assertHigh10ImplicitWeightedBDeblockFrames(t, frames, tt)
		})
	}
}

func TestFFmpegRawVideoMD5OracleHigh10ImplicitWeightedBDeblock(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}
	for _, tt := range high10ImplicitWeightedBDeblockFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := readHigh10ImplicitWeightedBDeblockFixture(t, tt)
			assertHigh10ImplicitWeightedBDeblockFixtureSyntax(t, data, tt)
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
				line := []byte(fmt.Sprintf("0, %10d, %10d,        1, %8d, %s", i, i, high10ImplicitWeightedBDeblockFrameRawSize, want))
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
			if len(raw) != len(tt.frameMD5)*high10ImplicitWeightedBDeblockFrameRawSize {
				t.Fatalf("rawvideo size = %d, want %d", len(raw), len(tt.frameMD5)*high10ImplicitWeightedBDeblockFrameRawSize)
			}
			sum := md5.Sum(raw)
			if got := hex.EncodeToString(sum[:]); got != tt.rawVideoMD5 {
				t.Fatalf("rawvideo md5 = %s, want %s", got, tt.rawVideoMD5)
			}
		})
	}
}

func high10ImplicitWeightedBDeblockFixtures() []high10ImplicitWeightedBDeblockFixture {
	return []high10ImplicitWeightedBDeblockFixture{
		{
			name:        "cavlc",
			file:        "high10_implicit_weight_b_deblock_cavlc.h264",
			cabac:       0,
			annexBSize:  483,
			annexBMD5:   "eb2326a2045e4e62441154e142a230dd",
			rawVideoMD5: "63548f6bb58f542054ccbad346ae62b9",
			frameMD5: []string{
				"08ebef68784433191d84710c0b69b5f4",
				"d62ab5d20886b180268fa13d08ea297b",
				"5f955b1a0f8ba29bacd6acbc1a0dd596",
				"ecce4d7c7a8735d83bd2b03a5e28cfa9",
				"37609e0c4649b287a67e31031faa8449",
			},
		},
		{
			name:        "cabac",
			file:        "high10_implicit_weight_b_deblock_cabac.h264",
			cabac:       1,
			annexBSize:  342,
			annexBMD5:   "f2554bc2baf41b6cf0ce0f0a1542a942",
			rawVideoMD5: "c9d9bcb19a4e12a223a1b4586bf2f8ae",
			frameMD5: []string{
				"1e790344e0614d9ebce915c0a3d09c06",
				"5287d7a383e9e001b9d438449ac43fec",
				"3584c70a06a55c9ca69fb48df3416134",
				"39329cb05d4db2dcb50d1daa2dc66f4a",
				"e572124d2419d619c1ada161a708d7c2",
			},
		},
	}
}

func readHigh10ImplicitWeightedBDeblockFixture(t *testing.T, tt high10ImplicitWeightedBDeblockFixture) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "h264", tt.file))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func assertHigh10ImplicitWeightedBDeblockFrames(t *testing.T, frames []*Frame, tt high10ImplicitWeightedBDeblockFixture) {
	t.Helper()
	if len(frames) != len(tt.frameMD5) {
		t.Fatalf("frames = %d, want %d", len(frames), len(tt.frameMD5))
	}
	assertHigh10FrameMD5Strings(t, frames, tt.frameMD5)
	raw := make([]byte, 0, len(frames)*high10ImplicitWeightedBDeblockFrameRawSize)
	for i, frame := range frames {
		if frame.Width != 32 || frame.Height != 16 {
			t.Fatalf("frame[%d] size = %dx%d, want 32x16", i, frame.Width, frame.Height)
		}
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
		rawSize, err := frame.RawYUVSize()
		if err != nil {
			t.Fatalf("frame[%d] RawYUVSize: %v", i, err)
		}
		if rawSize != high10ImplicitWeightedBDeblockFrameRawSize {
			t.Fatalf("frame[%d] raw yuv size = %d, want %d", i, rawSize, high10ImplicitWeightedBDeblockFrameRawSize)
		}
		raw, err = frame.AppendRawYUVBytesLE(raw)
		if err != nil {
			t.Fatalf("frame[%d] append raw yuv: %v", i, err)
		}
	}
	sum := md5.Sum(raw)
	if got := hex.EncodeToString(sum[:]); got != tt.rawVideoMD5 {
		t.Fatalf("rawvideo md5 = %s, want %s", got, tt.rawVideoMD5)
	}
}

func assertHigh10ImplicitWeightedBDeblockFixtureSyntax(t *testing.T, data []byte, tt high10ImplicitWeightedBDeblockFixture) {
	t.Helper()
	nals, err := h264.SplitAnnexB(data)
	if err != nil {
		t.Fatal(err)
	}

	var spsList [32]*h264.SPS
	var ppsList [256]*h264.PPS
	var gotSlices []int32
	var gotB int
	for _, nal := range nals {
		switch nal.Type {
		case h264.NALSPS:
			sps, err := h264.DecodeSPS(nal.RBSP)
			if err != nil {
				t.Fatal(err)
			}
			if sps.ProfileIDC != 110 || sps.Width != 32 || sps.Height != 16 ||
				sps.ChromaFormatIDC != 1 || sps.BitDepthLuma != 10 || sps.BitDepthChroma != 10 ||
				sps.FrameMBSOnlyFlag != 1 || sps.MBAFF != 0 || sps.Direct8x8InferenceFlag != 1 {
				t.Fatalf("SPS profile/format = %d %dx%d chroma %d depth %d/%d frame_only %d/%d direct8x8 %d, want High10 32x16 yuv420p10le frame-only direct8x8",
					sps.ProfileIDC, sps.Width, sps.Height, sps.ChromaFormatIDC, sps.BitDepthLuma, sps.BitDepthChroma,
					sps.FrameMBSOnlyFlag, sps.MBAFF, sps.Direct8x8InferenceFlag)
			}
			spsList[sps.SPSID] = sps
		case h264.NALPPS:
			pps, err := h264.DecodePPS(nal.RBSP, &spsList)
			if err != nil {
				t.Fatal(err)
			}
			if pps.CABAC != tt.cabac || pps.Transform8x8Mode != 0 || pps.WeightedPred != 0 ||
				pps.WeightedBipredIDC != 2 || pps.RefCount[0] != 1 || pps.RefCount[1] != 1 ||
				pps.DeblockingFilterParametersPresent != 1 {
				t.Fatalf("PPS cabac/8x8/weights/refs/deblock = %d/%d/%d/%d/%d/%d/%d, want cabac=%d no-8x8 implicit-B refs=1/1 deblock params",
					pps.CABAC, pps.Transform8x8Mode, pps.WeightedPred, pps.WeightedBipredIDC,
					pps.RefCount[0], pps.RefCount[1], pps.DeblockingFilterParametersPresent, tt.cabac)
			}
			ppsList[pps.PPSID] = pps
		case h264.NALIDRSlice, h264.NALSlice:
			sh, err := h264.ParseSliceHeader(nal, &ppsList)
			if err != nil {
				t.Fatal(err)
			}
			if sh.PictureStructure != h264.PictureFrame {
				t.Fatalf("slice picture = %d, want frame", sh.PictureStructure)
			}
			switch sh.SliceTypeNoS {
			case h264.PictureTypeI:
				if sh.ListCount != 0 {
					t.Fatalf("I slice lists = %d, want none", sh.ListCount)
				}
			case h264.PictureTypeP:
				if sh.DeblockingFilter != 1 || sh.ListCount != 1 || sh.RefCount[0] != 1 ||
					sh.PredWeightTable.UseWeight != 0 || sh.PredWeightTable.UseWeightChroma != 0 {
					t.Fatalf("P slice deblock/lists/refs/weights = %d/%d/%v/%d/%d, want deblock enabled L0 refs=1 unweighted",
						sh.DeblockingFilter, sh.ListCount, sh.RefCount, sh.PredWeightTable.UseWeight, sh.PredWeightTable.UseWeightChroma)
				}
			case h264.PictureTypeB:
				gotB++
				if sh.DeblockingFilter != 1 || sh.ListCount != 2 || sh.RefCount[0] != 1 || sh.RefCount[1] != 1 ||
					sh.DirectSpatialMVPred != 0 ||
					sh.PredWeightTable.UseWeight != 0 || sh.PredWeightTable.UseWeightChroma != 0 {
					t.Fatalf("B slice deblock/lists/refs/direct/serialized weights = %d/%d/%v/%d/%d/%d, want deblock enabled L0/L1 refs=1/1 temporal implicit weights only",
						sh.DeblockingFilter, sh.ListCount, sh.RefCount, sh.DirectSpatialMVPred,
						sh.PredWeightTable.UseWeight, sh.PredWeightTable.UseWeightChroma)
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
	wantSlices := []int32{h264.PictureTypeI, h264.PictureTypeP, h264.PictureTypeB, h264.PictureTypeB, h264.PictureTypeP}
	if len(gotSlices) != len(wantSlices) {
		t.Fatalf("slice count = %d, want %d", len(gotSlices), len(wantSlices))
	}
	for i, want := range wantSlices {
		if gotSlices[i] != want {
			t.Fatalf("slice[%d] type = %d, want %d", i, gotSlices[i], want)
		}
	}
	if gotB != 2 {
		t.Fatalf("B slices = %d, want 2", gotB)
	}
}
