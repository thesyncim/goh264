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

type high10DirectBDeblockFixture struct {
	name          string
	file          string
	cabac         int32
	directSpatial int32
	annexBSize    int
	annexBMD5     string
	frameMD5      []string
	rawVideoMD5   string
}

func TestHigh10DirectBDeblockFixtureSyntax(t *testing.T) {
	for _, tt := range high10DirectBDeblockFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := readHigh10DirectBDeblockFixture(t, tt)
			assertHigh10DirectBDeblockFixtureSyntax(t, data, tt)
		})
	}
}

func TestDecodeAnnexBHigh10DirectBDeblockFrames(t *testing.T) {
	for _, tt := range high10DirectBDeblockFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := readHigh10DirectBDeblockFixture(t, tt)
			assertHigh10DirectBDeblockFixtureSyntax(t, data, tt)
			frames, err := NewDecoder().DecodeAnnexBFrames(data)
			if err != nil {
				t.Fatalf("DecodeAnnexBFrames: %v", err)
			}
			assertHigh10DirectBDeblockFrames(t, frames, tt)
		})
	}
}

func TestDecodeAVCHigh10DirectBDeblockFrames(t *testing.T) {
	for _, tt := range high10DirectBDeblockFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := readHigh10DirectBDeblockFixture(t, tt)
			assertHigh10DirectBDeblockFixtureSyntax(t, data, tt)
			for _, nalLengthSize := range []int{2, 3, 4} {
				frames, err := NewDecoder().DecodeAVCFrames(annexBToAVC(t, data, nalLengthSize), nalLengthSize)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: DecodeAVCFrames: %v", nalLengthSize, err)
				}
				assertHigh10DirectBDeblockFrames(t, frames, tt)
			}
		})
	}
}

func TestDecodeAVCCHigh10DirectBDeblockFrames(t *testing.T) {
	for _, tt := range high10DirectBDeblockFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := readHigh10DirectBDeblockFixture(t, tt)
			assertHigh10DirectBDeblockFixtureSyntax(t, data, tt)
			for _, nalLengthSize := range []int{2, 3, 4} {
				config, packet := annexBToAVCConfigAndPacket(t, data, nalLengthSize)
				frames, err := NewDecoder().DecodeAVCCFrames(config, packet)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: DecodeAVCCFrames: %v", nalLengthSize, err)
				}
				assertHigh10DirectBDeblockFrames(t, frames, tt)
			}
		})
	}
}

func TestDecodeConfiguredAVCHigh10DirectBDeblockFramesAcrossSamplesFlush(t *testing.T) {
	for _, tt := range high10DirectBDeblockFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := readHigh10DirectBDeblockFixture(t, tt)
			assertHigh10DirectBDeblockFixtureSyntax(t, data, tt)
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
						t.Fatalf("nalLengthSize=%d sample[%d]: DecodeConfiguredAVCFrames: %v", nalLengthSize, i, err)
					}
					frames = append(frames, out...)
				}
				out, err := dec.FlushDelayedFrames()
				if err != nil {
					t.Fatalf("nalLengthSize=%d flush: %v", nalLengthSize, err)
				}
				frames = append(frames, out...)
				assertHigh10DirectBDeblockFrames(t, frames, tt)
			}
		})
	}
}

func TestDecodeAutoConfiguredAVCHigh10DirectBDeblockFramesAcrossSamplesFlush(t *testing.T) {
	for _, tt := range high10DirectBDeblockFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := readHigh10DirectBDeblockFixture(t, tt)
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
					t.Fatalf("sample[%d]: DecodeFrames: %v", i, err)
				}
				frames = append(frames, out...)
			}
			out, err = dec.DecodeFrames(nil)
			if err != nil {
				t.Fatalf("flush: %v", err)
			}
			frames = append(frames, out...)
			assertHigh10DirectBDeblockFrames(t, frames, tt)
		})
	}
}

func TestFFmpegRawVideoMD5OracleHigh10DirectBDeblock(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}

	for _, tt := range high10DirectBDeblockFixtures() {
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
				line := []byte(fmt.Sprintf("0, %10d, %10d,        1, %8d, %s", i, i, high10DirectBFrameRawSize, want))
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
			if len(raw) != len(tt.frameMD5)*high10DirectBFrameRawSize {
				t.Fatalf("rawvideo size = %d, want %d", len(raw), len(tt.frameMD5)*high10DirectBFrameRawSize)
			}
			sum := md5.Sum(raw)
			if got := hex.EncodeToString(sum[:]); got != tt.rawVideoMD5 {
				t.Fatalf("rawvideo md5 = %s, want %s", got, tt.rawVideoMD5)
			}
		})
	}
}

func high10DirectBDeblockFixtures() []high10DirectBDeblockFixture {
	return []high10DirectBDeblockFixture{
		{
			name:          "temporal/cavlc",
			file:          "high10_direct_b_deblock_temporal_cavlc.h264",
			directSpatial: 0,
			annexBSize:    789,
			annexBMD5:     "94c4f9b73c8a8b59f756320f20cf7def",
			frameMD5: []string{
				"86945e69a42629edd0fa46f7b8032c1d",
				"46eebce937687169972bc95b770f2953",
				"6185d7575b0476622e2317ad84de9ca8",
			},
			rawVideoMD5: "663118a3e79cd6b41bb20a14867f7015",
		},
		{
			name:          "temporal/cabac",
			file:          "high10_direct_b_deblock_temporal_cabac.h264",
			cabac:         1,
			directSpatial: 0,
			annexBSize:    798,
			annexBMD5:     "59b29d60becffa83b095cd1eafc72757",
			frameMD5: []string{
				"7e34fc5b9647628681a446de7c88c108",
				"b24f513ee6c045f5c1add2a1e89e1af5",
				"50ecc1b26b4ddd9582d37e1703e3a31e",
			},
			rawVideoMD5: "411680af6618b27159866c456c28f6ff",
		},
		{
			name:          "spatial/cavlc",
			file:          "high10_direct_b_deblock_spatial_cavlc.h264",
			directSpatial: 1,
			annexBSize:    789,
			annexBMD5:     "6d64382e77d76c28a17f31208d50a751",
			frameMD5: []string{
				"86945e69a42629edd0fa46f7b8032c1d",
				"46eebce937687169972bc95b770f2953",
				"6185d7575b0476622e2317ad84de9ca8",
			},
			rawVideoMD5: "663118a3e79cd6b41bb20a14867f7015",
		},
		{
			name:          "spatial/cabac",
			file:          "high10_direct_b_deblock_spatial_cabac.h264",
			cabac:         1,
			directSpatial: 1,
			annexBSize:    798,
			annexBMD5:     "a5c947ab318d1ef5a4eac96fb19cbacf",
			frameMD5: []string{
				"7e34fc5b9647628681a446de7c88c108",
				"b24f513ee6c045f5c1add2a1e89e1af5",
				"50ecc1b26b4ddd9582d37e1703e3a31e",
			},
			rawVideoMD5: "411680af6618b27159866c456c28f6ff",
		},
	}
}

func readHigh10DirectBDeblockFixture(t *testing.T, tt high10DirectBDeblockFixture) []byte {
	t.Helper()
	path := filepath.Join("testdata", "h264", tt.file)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if len(data) != tt.annexBSize {
		t.Fatalf("annex b size = %d, want %d", len(data), tt.annexBSize)
	}
	sum := md5.Sum(data)
	if got := hex.EncodeToString(sum[:]); got != tt.annexBMD5 {
		t.Fatalf("annex b md5 = %s, want %s", got, tt.annexBMD5)
	}
	return data
}

func assertHigh10DirectBDeblockFrames(t *testing.T, frames []*Frame, tt high10DirectBDeblockFixture) {
	t.Helper()
	assertHigh10DirectBFrames(t, frames, tt.frameMD5)
	raw := make([]byte, 0, len(frames)*high10DirectBFrameRawSize)
	for i, frame := range frames {
		var err error
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

func assertHigh10DirectBDeblockFixtureSyntax(t *testing.T, data []byte, tt high10DirectBDeblockFixture) {
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
			if sh.PictureStructure != h264.PictureFrame || sh.DeblockingFilter != 1 {
				t.Fatalf("slice picture/deblock = %d/%d, want frame/enabled", sh.PictureStructure, sh.DeblockingFilter)
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
