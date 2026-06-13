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

const (
	high10DirectSubFrameRawSize = 768
	high10DirectSubRawVideoMD5  = "bed8c5ab899fe974cae09585e60b151f"
)

var high10DirectSubFrameMD5 = []string{
	"d73be6c1b3e4082e402d67d810323786",
	"d73be6c1b3e4082e402d67d810323786",
	"d73be6c1b3e4082e402d67d810323786",
}

type high10DirectSubFixture struct {
	name          string
	file          string
	cabac         int32
	directSpatial int32
	direct8x8     int32
	annexBSize    int
	annexBMD5     string
}

func TestHigh10DirectSubFixtureSyntax(t *testing.T) {
	for _, tt := range high10DirectSubFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := readHigh10DirectSubFixture(t, tt)
			if len(data) != tt.annexBSize {
				t.Fatalf("annex b size = %d, want %d", len(data), tt.annexBSize)
			}
			sum := md5.Sum(data)
			if got := hex.EncodeToString(sum[:]); got != tt.annexBMD5 {
				t.Fatalf("annex b md5 = %s, want %s", got, tt.annexBMD5)
			}
			assertHigh10DirectSubFixtureSyntax(t, data, tt)
		})
	}
}

func TestDecodeAnnexBHigh10DirectSubFrames(t *testing.T) {
	for _, tt := range high10DirectSubFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := readHigh10DirectSubFixture(t, tt)
			assertHigh10DirectSubFixtureSyntax(t, data, tt)
			frames, err := NewDecoder().DecodeAnnexBFrames(data)
			if err != nil {
				t.Fatalf("DecodeAnnexBFrames: %v", err)
			}
			assertHigh10DirectSubFrames(t, frames)
		})
	}
}

func TestDecodeAVCHigh10DirectSubFrames(t *testing.T) {
	for _, tt := range high10DirectSubFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := readHigh10DirectSubFixture(t, tt)
			assertHigh10DirectSubFixtureSyntax(t, data, tt)
			for _, nalLengthSize := range []int{2, 3, 4} {
				frames, err := NewDecoder().DecodeAVCFrames(annexBToAVC(t, data, nalLengthSize), nalLengthSize)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: DecodeAVCFrames: %v", nalLengthSize, err)
				}
				assertHigh10DirectSubFrames(t, frames)
			}
		})
	}
}

func TestDecodeAVCCHigh10DirectSubFrames(t *testing.T) {
	for _, tt := range high10DirectSubFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := readHigh10DirectSubFixture(t, tt)
			assertHigh10DirectSubFixtureSyntax(t, data, tt)
			for _, nalLengthSize := range []int{2, 3, 4} {
				config, packet := annexBToAVCConfigAndPacket(t, data, nalLengthSize)
				frames, err := NewDecoder().DecodeAVCCFrames(config, packet)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: DecodeAVCCFrames: %v", nalLengthSize, err)
				}
				assertHigh10DirectSubFrames(t, frames)
			}
		})
	}
}

func TestDecodeConfiguredAVCHigh10DirectSubFramesAcrossSamplesFlush(t *testing.T) {
	for _, tt := range high10DirectSubFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := readHigh10DirectSubFixture(t, tt)
			assertHigh10DirectSubFixtureSyntax(t, data, tt)
			for _, nalLengthSize := range []int{2, 3, 4} {
				config, samples := annexBToAVCConfigAndSamples(t, data, nalLengthSize)
				if len(samples) != len(high10DirectSubFrameMD5) {
					t.Fatalf("nalLengthSize=%d: samples = %d, want %d", nalLengthSize, len(samples), len(high10DirectSubFrameMD5))
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
				assertHigh10DirectSubFrames(t, frames)

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

func TestFFmpegRawVideoMD5OracleHigh10DirectSub(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") == "" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}
	for _, tt := range high10DirectSubFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := readHigh10DirectSubFixture(t, tt)
			assertHigh10DirectSubFixtureSyntax(t, data, tt)

			framemd5 := exec.Command("ffmpeg",
				"-hide_banner", "-v", "error",
				"-i", filepath.Join("testdata", "h264", tt.file),
				"-f", "framemd5",
				"-pix_fmt", "yuv420p10le",
				"-")
			framemd5Out, err := framemd5.Output()
			if err != nil {
				t.Fatalf("ffmpeg framemd5: %v", err)
			}
			for i, want := range high10DirectSubFrameMD5 {
				line := []byte(fmt.Sprintf("0, %10d, %10d,        1, %8d, %s", i, i, high10DirectSubFrameRawSize, want))
				if !bytes.Contains(framemd5Out, line) {
					t.Fatalf("frame[%d] missing %q in framemd5:\n%s", i, line, framemd5Out)
				}
			}
			rawCmd := exec.Command("ffmpeg",
				"-hide_banner", "-v", "error",
				"-i", filepath.Join("testdata", "h264", tt.file),
				"-f", "rawvideo",
				"-pix_fmt", "yuv420p10le",
				"-")
			raw, err := rawCmd.Output()
			if err != nil {
				t.Fatalf("ffmpeg rawvideo: %v", err)
			}
			if len(raw) != len(high10DirectSubFrameMD5)*high10DirectSubFrameRawSize {
				t.Fatalf("rawvideo size = %d, want %d", len(raw), len(high10DirectSubFrameMD5)*high10DirectSubFrameRawSize)
			}
			sum := md5.Sum(raw)
			if got := hex.EncodeToString(sum[:]); got != high10DirectSubRawVideoMD5 {
				t.Fatalf("rawvideo md5 = %s, want %s", got, high10DirectSubRawVideoMD5)
			}
		})
	}
}

func high10DirectSubFixtures() []high10DirectSubFixture {
	return []high10DirectSubFixture{
		{
			name:          "cavlc-b8x8-temporal",
			file:          "high10_cavlc_b8x8_temporal_direct_sub.h264",
			directSpatial: 0,
			direct8x8:     1,
			annexBSize:    704,
			annexBMD5:     "737b17dbc09f1d038fabccad1308afd4",
		},
		{
			name:          "cabac-b8x8-temporal",
			file:          "high10_cabac_b8x8_temporal_direct_sub.h264",
			cabac:         1,
			directSpatial: 0,
			direct8x8:     1,
			annexBSize:    711,
			annexBMD5:     "ac402e6f18e176ba51da9899b3285e66",
		},
		{
			name:          "cavlc-b8x8-spatial",
			file:          "high10_cavlc_b8x8_spatial_direct_sub.h264",
			directSpatial: 1,
			direct8x8:     1,
			annexBSize:    704,
			annexBMD5:     "87dc52d6a6ca8d0309c3bf064ab36eeb",
		},
		{
			name:          "cabac-b8x8-spatial",
			file:          "high10_cabac_b8x8_spatial_direct_sub.h264",
			cabac:         1,
			directSpatial: 1,
			direct8x8:     1,
			annexBSize:    711,
			annexBMD5:     "70b723a521824437321dca37b6b4f335",
		},
		{
			name:          "cavlc-b4x4-temporal",
			file:          "high10_cavlc_b4x4_temporal_direct_sub.h264",
			directSpatial: 0,
			direct8x8:     0,
			annexBSize:    704,
			annexBMD5:     "c1abf23eeb9ccb84465e8b701886c9e8",
		},
		{
			name:          "cabac-b4x4-temporal",
			file:          "high10_cabac_b4x4_temporal_direct_sub.h264",
			cabac:         1,
			directSpatial: 0,
			direct8x8:     0,
			annexBSize:    711,
			annexBMD5:     "56fbd77d91f0ce2e22d485e77c98a491",
		},
		{
			name:          "cavlc-b4x4-spatial",
			file:          "high10_cavlc_b4x4_spatial_direct_sub.h264",
			directSpatial: 1,
			direct8x8:     0,
			annexBSize:    704,
			annexBMD5:     "33bafb77f946ce2d9fe1168e8f9de609",
		},
		{
			name:          "cabac-b4x4-spatial",
			file:          "high10_cabac_b4x4_spatial_direct_sub.h264",
			cabac:         1,
			directSpatial: 1,
			direct8x8:     0,
			annexBSize:    711,
			annexBMD5:     "3f567a5a22de5ae171658d71264a83f5",
		},
	}
}

func readHigh10DirectSubFixture(t *testing.T, tt high10DirectSubFixture) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "h264", tt.file))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func assertHigh10DirectSubFrames(t *testing.T, frames []*Frame) {
	t.Helper()
	if len(frames) != len(high10DirectSubFrameMD5) {
		t.Fatalf("frames = %d, want %d", len(frames), len(high10DirectSubFrameMD5))
	}
	assertHigh10FrameMD5Strings(t, frames, high10DirectSubFrameMD5)
	raw := make([]byte, 0, len(frames)*high10DirectSubFrameRawSize)
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
			t.Fatalf("frame[%d] raw format/sample bytes = %s/%d, want yuv420p10le/2",
				i, pixFmt, bytesPerSample)
		}
		raw, err = frame.AppendRawYUVBytesLE(raw)
		if err != nil {
			t.Fatalf("frame[%d] append raw yuv: %v", i, err)
		}
		rawSize, err := frame.RawYUVSize()
		if err != nil {
			t.Fatalf("frame[%d] RawYUVSize: %v", i, err)
		}
		if rawSize != high10DirectSubFrameRawSize {
			t.Fatalf("frame[%d] raw yuv size = %d, want %d", i, rawSize, high10DirectSubFrameRawSize)
		}
	}
	sum := md5.Sum(raw)
	if got := hex.EncodeToString(sum[:]); got != high10DirectSubRawVideoMD5 {
		t.Fatalf("rawvideo md5 = %s, want %s", got, high10DirectSubRawVideoMD5)
	}
}

func assertHigh10DirectSubFixtureSyntax(t *testing.T, data []byte, tt high10DirectSubFixture) {
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
				int32(sps.Direct8x8InferenceFlag) != tt.direct8x8 {
				t.Fatalf("SPS profile/format = %d %dx%d chroma %d depth %d/%d frame_only %d/%d refs %d direct8x8 %d, want High10 16x16 yuv420p10le frame-only refs=2 direct8x8=%d",
					sps.ProfileIDC, sps.Width, sps.Height, sps.ChromaFormatIDC, sps.BitDepthLuma, sps.BitDepthChroma,
					sps.FrameMBSOnlyFlag, sps.MBAFF, sps.RefFrameCount, sps.Direct8x8InferenceFlag, tt.direct8x8)
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
				assertHigh10DirectSubPayload(t, nal.Raw, tt)
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

func assertHigh10DirectSubPayload(t *testing.T, raw []byte, tt high10DirectSubFixture) {
	t.Helper()
	if tt.cabac != 0 {
		assertHigh10DirectSubCABACPayload(t, raw, tt)
		return
	}
	assertHigh10DirectSubCAVLCPayload(t, raw)
}

func assertHigh10DirectSubCAVLCPayload(t *testing.T, raw []byte) {
	t.Helper()
	if len(raw) != 7 {
		t.Fatalf("B direct-sub NAL size = %d, want 7", len(raw))
	}
	want := "100001011111111"
	for i, bit := range want {
		if got := h264FixtureBit(raw, 37+i); got != int(bit-'0') {
			t.Fatalf("B direct-sub payload bit[%d] = %d, want %c", i, got, bit)
		}
	}
}

func assertHigh10DirectSubCABACPayload(t *testing.T, raw []byte, tt high10DirectSubFixture) {
	t.Helper()
	if len(raw) != 10 {
		t.Fatalf("B direct-sub CABAC NAL size = %d, want 10", len(raw))
	}
	wantPrefix := []byte{0x01, 0x9e, byte(0x44 + tt.directSpatial), 0xe4, 0x8b}
	if !bytes.Equal(raw[:len(wantPrefix)], wantPrefix) {
		t.Fatalf("B direct-sub CABAC header/alignment = %x, want %x", raw[:len(wantPrefix)], wantPrefix)
	}
	wantBody := []byte{0xbe, 0x27, 0xfe, 0xed, 0x80}
	if !bytes.Equal(raw[len(wantPrefix):], wantBody) {
		t.Fatalf("B direct-sub CABAC body = %x, want %x", raw[len(wantPrefix):], wantBody)
	}
}

func h264FixtureBit(data []byte, pos int) int {
	return int((data[pos/8] >> uint(7-pos%8)) & 1)
}
