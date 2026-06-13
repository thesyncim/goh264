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
	high10DirectSubDeblockFrameRawSize = 768
	high10DirectSubDeblockRawVideoMD5  = "078d6e505df703e46ebbbdb155fb47cd"
)

var high10DirectSubDeblockFrameMD5 = []string{
	"857cc91515b2182f4444a4d746b9d721",
	"857cc91515b2182f4444a4d746b9d721",
	"857cc91515b2182f4444a4d746b9d721",
}

type high10DirectSubDeblockFixture struct {
	name              string
	file              string
	cabac             int32
	weightedBipredIDC uint32
	directSpatial     int32
	direct8x8         int32
	annexBSize        int
	annexBMD5         string
}

func TestHigh10DirectSubDeblockFixtureSyntax(t *testing.T) {
	for _, tt := range high10DirectSubDeblockFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := readHigh10DirectSubDeblockFixture(t, tt)
			if len(data) != tt.annexBSize {
				t.Fatalf("annex b size = %d, want %d", len(data), tt.annexBSize)
			}
			sum := md5.Sum(data)
			if got := hex.EncodeToString(sum[:]); got != tt.annexBMD5 {
				t.Fatalf("annex b md5 = %s, want %s", got, tt.annexBMD5)
			}
			assertHigh10DirectSubDeblockFixtureSyntax(t, data, tt)
		})
	}
}

func TestDecodeAnnexBHigh10DirectSubDeblockFrames(t *testing.T) {
	for _, tt := range high10DirectSubDeblockFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := readHigh10DirectSubDeblockFixture(t, tt)
			assertHigh10DirectSubDeblockFixtureSyntax(t, data, tt)
			frames, err := NewDecoder().DecodeAnnexBFrames(data)
			if err != nil {
				t.Fatalf("DecodeAnnexBFrames: %v", err)
			}
			assertHigh10DirectSubDeblockFrames(t, frames)
		})
	}
}

func TestDecodeAVCHigh10DirectSubDeblockFrames(t *testing.T) {
	for _, tt := range high10DirectSubDeblockFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := readHigh10DirectSubDeblockFixture(t, tt)
			assertHigh10DirectSubDeblockFixtureSyntax(t, data, tt)
			for _, nalLengthSize := range []int{2, 3, 4} {
				frames, err := NewDecoder().DecodeAVCFrames(annexBToAVC(t, data, nalLengthSize), nalLengthSize)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: DecodeAVCFrames: %v", nalLengthSize, err)
				}
				assertHigh10DirectSubDeblockFrames(t, frames)
			}
		})
	}
}

func TestDecodeAVCCHigh10DirectSubDeblockFrames(t *testing.T) {
	for _, tt := range high10DirectSubDeblockFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := readHigh10DirectSubDeblockFixture(t, tt)
			assertHigh10DirectSubDeblockFixtureSyntax(t, data, tt)
			for _, nalLengthSize := range []int{2, 3, 4} {
				config, packet := annexBToAVCConfigAndPacket(t, data, nalLengthSize)
				frames, err := NewDecoder().DecodeAVCCFrames(config, packet)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: DecodeAVCCFrames: %v", nalLengthSize, err)
				}
				assertHigh10DirectSubDeblockFrames(t, frames)
			}
		})
	}
}

func TestDecodeConfiguredAVCHigh10DirectSubDeblockFramesAcrossSamplesFlush(t *testing.T) {
	for _, tt := range high10DirectSubDeblockFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := readHigh10DirectSubDeblockFixture(t, tt)
			assertHigh10DirectSubDeblockFixtureSyntax(t, data, tt)
			for _, nalLengthSize := range []int{2, 3, 4} {
				config, samples := annexBToAVCConfigAndSamples(t, data, nalLengthSize)
				if len(samples) != len(high10DirectSubDeblockFrameMD5) {
					t.Fatalf("nalLengthSize=%d: samples = %d, want %d", nalLengthSize, len(samples), len(high10DirectSubDeblockFrameMD5))
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
				assertHigh10DirectSubDeblockFrames(t, frames)
			}
		})
	}
}

func TestDecodeAutoConfiguredAVCHigh10DirectSubDeblockFramesAcrossSamplesFlush(t *testing.T) {
	for _, tt := range high10DirectSubDeblockFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := readHigh10DirectSubDeblockFixture(t, tt)
			config, samples := annexBToAVCConfigAndSamples(t, data, 4)
			if len(samples) != len(high10DirectSubDeblockFrameMD5) {
				t.Fatalf("samples = %d, want %d", len(samples), len(high10DirectSubDeblockFrameMD5))
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
			assertHigh10DirectSubDeblockFrames(t, frames)
		})
	}
}

func TestFFmpegRawVideoMD5OracleHigh10DirectSubDeblock(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}

	for _, tt := range high10DirectSubDeblockFixtures() {
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
			for i, want := range high10DirectSubDeblockFrameMD5 {
				line := []byte(fmt.Sprintf("0, %10d, %10d,        1, %8d, %s", i, i, high10DirectSubDeblockFrameRawSize, want))
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
			if len(raw) != len(high10DirectSubDeblockFrameMD5)*high10DirectSubDeblockFrameRawSize {
				t.Fatalf("rawvideo size = %d, want %d", len(raw), len(high10DirectSubDeblockFrameMD5)*high10DirectSubDeblockFrameRawSize)
			}
			sum := md5.Sum(raw)
			if got := hex.EncodeToString(sum[:]); got != high10DirectSubDeblockRawVideoMD5 {
				t.Fatalf("rawvideo md5 = %s, want %s", got, high10DirectSubDeblockRawVideoMD5)
			}
		})
	}
}

func high10DirectSubDeblockFixtures() []high10DirectSubDeblockFixture {
	return []high10DirectSubDeblockFixture{
		{
			name:          "cavlc-b8x8-temporal",
			file:          "high10_cavlc_b8x8_temporal_direct_sub_deblock.h264",
			directSpatial: 0,
			direct8x8:     1,
			annexBSize:    66,
			annexBMD5:     "78b3cec237d32febec29cb4d9398e623",
		},
		{
			name:          "cabac-b8x8-temporal",
			file:          "high10_cabac_b8x8_temporal_direct_sub_deblock.h264",
			cabac:         1,
			directSpatial: 0,
			direct8x8:     1,
			annexBSize:    74,
			annexBMD5:     "f76df753492ebb25ce7c8b96acb39c54",
		},
		{
			name:          "cavlc-b8x8-spatial",
			file:          "high10_cavlc_b8x8_spatial_direct_sub_deblock.h264",
			directSpatial: 1,
			direct8x8:     1,
			annexBSize:    66,
			annexBMD5:     "af3ee3eaae4ce7fc0ef2c2168fb7ef1d",
		},
		{
			name:          "cabac-b8x8-spatial",
			file:          "high10_cabac_b8x8_spatial_direct_sub_deblock.h264",
			cabac:         1,
			directSpatial: 1,
			direct8x8:     1,
			annexBSize:    74,
			annexBMD5:     "dfa92d7238c7d42b2afc12ae05deb6f9",
		},
		{
			name:          "cavlc-b4x4-temporal",
			file:          "high10_cavlc_b4x4_temporal_direct_sub_deblock.h264",
			directSpatial: 0,
			direct8x8:     0,
			annexBSize:    66,
			annexBMD5:     "8eb21297fd5ebf52ba68284179010143",
		},
		{
			name:          "cabac-b4x4-temporal",
			file:          "high10_cabac_b4x4_temporal_direct_sub_deblock.h264",
			cabac:         1,
			directSpatial: 0,
			direct8x8:     0,
			annexBSize:    74,
			annexBMD5:     "353bdd174a91f11782bc0c5a938e5ae9",
		},
		{
			name:          "cavlc-b4x4-spatial",
			file:          "high10_cavlc_b4x4_spatial_direct_sub_deblock.h264",
			directSpatial: 1,
			direct8x8:     0,
			annexBSize:    66,
			annexBMD5:     "c8ca695aa97e157646c53d47fc38154d",
		},
		{
			name:          "cabac-b4x4-spatial",
			file:          "high10_cabac_b4x4_spatial_direct_sub_deblock.h264",
			cabac:         1,
			directSpatial: 1,
			direct8x8:     0,
			annexBSize:    74,
			annexBMD5:     "600c53594971dfac2c41f630785a6790",
		},
		{
			name:              "implicit-cavlc-b8x8-temporal",
			file:              "high10_implicit_weight_cavlc_b8x8_temporal_direct_sub_deblock.h264",
			weightedBipredIDC: 2,
			directSpatial:     0,
			direct8x8:         1,
			annexBSize:        66,
			annexBMD5:         "c2a8a3772b13c14edc65bdfbfec7f163",
		},
		{
			name:              "implicit-cabac-b8x8-temporal",
			file:              "high10_implicit_weight_cabac_b8x8_temporal_direct_sub_deblock.h264",
			cabac:             1,
			weightedBipredIDC: 2,
			directSpatial:     0,
			direct8x8:         1,
			annexBSize:        74,
			annexBMD5:         "8533d0791ce508eac1c937e467be4cdc",
		},
		{
			name:              "implicit-cavlc-b8x8-spatial",
			file:              "high10_implicit_weight_cavlc_b8x8_spatial_direct_sub_deblock.h264",
			weightedBipredIDC: 2,
			directSpatial:     1,
			direct8x8:         1,
			annexBSize:        66,
			annexBMD5:         "fbb6747c4e45b8df212b12e5b460a144",
		},
		{
			name:              "implicit-cabac-b8x8-spatial",
			file:              "high10_implicit_weight_cabac_b8x8_spatial_direct_sub_deblock.h264",
			cabac:             1,
			weightedBipredIDC: 2,
			directSpatial:     1,
			direct8x8:         1,
			annexBSize:        74,
			annexBMD5:         "b3370e5a49ad956ac44aa1401e16fa32",
		},
		{
			name:              "implicit-cavlc-b4x4-temporal",
			file:              "high10_implicit_weight_cavlc_b4x4_temporal_direct_sub_deblock.h264",
			weightedBipredIDC: 2,
			directSpatial:     0,
			direct8x8:         0,
			annexBSize:        66,
			annexBMD5:         "573b75bd19d44bbcf61137a03a9235ad",
		},
		{
			name:              "implicit-cabac-b4x4-temporal",
			file:              "high10_implicit_weight_cabac_b4x4_temporal_direct_sub_deblock.h264",
			cabac:             1,
			weightedBipredIDC: 2,
			directSpatial:     0,
			direct8x8:         0,
			annexBSize:        74,
			annexBMD5:         "64fd307051fa62ce382cbf5dacf893ae",
		},
		{
			name:              "implicit-cavlc-b4x4-spatial",
			file:              "high10_implicit_weight_cavlc_b4x4_spatial_direct_sub_deblock.h264",
			weightedBipredIDC: 2,
			directSpatial:     1,
			direct8x8:         0,
			annexBSize:        66,
			annexBMD5:         "b091749f36abfad1e06c302cad8b5f90",
		},
		{
			name:              "implicit-cabac-b4x4-spatial",
			file:              "high10_implicit_weight_cabac_b4x4_spatial_direct_sub_deblock.h264",
			cabac:             1,
			weightedBipredIDC: 2,
			directSpatial:     1,
			direct8x8:         0,
			annexBSize:        74,
			annexBMD5:         "8cd327d0c9088a8a08fb5be9d3e0a766",
		},
	}
}

func readHigh10DirectSubDeblockFixture(t *testing.T, tt high10DirectSubDeblockFixture) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "h264", tt.file))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func assertHigh10DirectSubDeblockFrames(t *testing.T, frames []*Frame) {
	t.Helper()
	assertHigh10FrameMD5Strings(t, frames, high10DirectSubDeblockFrameMD5)
	raw := make([]byte, 0, len(frames)*high10DirectSubDeblockFrameRawSize)
	for i, frame := range frames {
		pixFmt, err := frame.RawPixelFormat()
		if err != nil {
			t.Fatalf("frame[%d] RawPixelFormat: %v", i, err)
		}
		bytesPerSample, err := frame.BytesPerSample()
		if err != nil {
			t.Fatalf("frame[%d] BytesPerSample: %v", i, err)
		}
		if frame.Width != 16 || frame.Height != 16 || pixFmt != "yuv420p10le" || bytesPerSample != 2 {
			t.Fatalf("frame[%d] geometry/pixfmt/sample = %dx%d %s/%d, want 16x16 yuv420p10le/2",
				i, frame.Width, frame.Height, pixFmt, bytesPerSample)
		}
		raw, err = frame.AppendRawYUVBytesLE(raw)
		if err != nil {
			t.Fatalf("frame[%d] append raw yuv: %v", i, err)
		}
		rawSize, err := frame.RawYUVSize()
		if err != nil {
			t.Fatalf("frame[%d] RawYUVSize: %v", i, err)
		}
		if rawSize != high10DirectSubDeblockFrameRawSize {
			t.Fatalf("frame[%d] raw yuv size = %d, want %d", i, rawSize, high10DirectSubDeblockFrameRawSize)
		}
	}
	sum := md5.Sum(raw)
	if got := hex.EncodeToString(sum[:]); got != high10DirectSubDeblockRawVideoMD5 {
		t.Fatalf("rawvideo md5 = %s, want %s", got, high10DirectSubDeblockRawVideoMD5)
	}
}

func assertHigh10DirectSubDeblockFixtureSyntax(t *testing.T, data []byte, tt high10DirectSubDeblockFixture) {
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
				pps.WeightedBipredIDC != tt.weightedBipredIDC || pps.RefCount[0] != 2 || pps.RefCount[1] != 1 ||
				pps.DeblockingFilterParametersPresent != 1 {
				t.Fatalf("PPS cabac/8x8/weights/refs/deblock = %d/%d/%d/%d/%d/%d/%d, want cabac=%d no-8x8 weighted_bipred_idc=%d refs=2/1 deblock params",
					pps.CABAC, pps.Transform8x8Mode, pps.WeightedPred, pps.WeightedBipredIDC,
					pps.RefCount[0], pps.RefCount[1], pps.DeblockingFilterParametersPresent, tt.cabac, tt.weightedBipredIDC)
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
				if sh.ListCount != 0 || sh.DeblockingFilter != 0 {
					t.Fatalf("I slice lists/deblock = %d/%d, want none/disabled for flat IDR", sh.ListCount, sh.DeblockingFilter)
				}
			case h264.PictureTypeP:
				if sh.ListCount != 1 || sh.RefCount[0] != 1 || sh.DeblockingFilter != 1 ||
					sh.PredWeightTable.UseWeight != 0 || sh.PredWeightTable.UseWeightChroma != 0 {
					t.Fatalf("P slice lists/refs/deblock/weights = %d/%v/%d/%d/%d, want L0 refs=1 deblock-enabled unweighted",
						sh.ListCount, sh.RefCount, sh.DeblockingFilter, sh.PredWeightTable.UseWeight, sh.PredWeightTable.UseWeightChroma)
				}
			case h264.PictureTypeB:
				if sh.ListCount != 2 || sh.RefCount[0] != 1 || sh.RefCount[1] != 1 || sh.DeblockingFilter != 1 ||
					sh.PPS == nil || sh.PPS.WeightedBipredIDC != tt.weightedBipredIDC ||
					sh.DirectSpatialMVPred != tt.directSpatial ||
					sh.PredWeightTable.UseWeight != 0 || sh.PredWeightTable.UseWeightChroma != 0 {
					t.Fatalf("B slice lists/refs/deblock/implicit/direct/weights = %d/%v/%d/%v/%d/%d/%d, want L0/L1 refs=1/1 deblock-enabled weighted_bipred_idc=%d direct=%d no serialized weights",
						sh.ListCount, sh.RefCount, sh.DeblockingFilter, sh.PPS, sh.DirectSpatialMVPred, sh.PredWeightTable.UseWeight, sh.PredWeightTable.UseWeightChroma, tt.weightedBipredIDC, tt.directSpatial)
				}
				assertHigh10DirectSubDeblockPayload(t, nal.Raw, tt)
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

func assertHigh10DirectSubDeblockPayload(t *testing.T, raw []byte, tt high10DirectSubDeblockFixture) {
	t.Helper()
	if tt.cabac != 0 {
		assertHigh10DirectSubDeblockCABACPayload(t, raw, tt)
		return
	}
	assertHigh10DirectSubDeblockCAVLCPayload(t, raw)
}

func assertHigh10DirectSubDeblockCAVLCPayload(t *testing.T, raw []byte) {
	t.Helper()
	if len(raw) != 7 {
		t.Fatalf("B direct-sub deblock NAL size = %d, want 7", len(raw))
	}
	want := "100001011111111"
	for i, bit := range want {
		if got := h264FixtureBit(raw, 37+i); got != int(bit-'0') {
			t.Fatalf("B direct-sub deblock payload bit[%d] = %d, want %c", i, got, bit)
		}
	}
}

func assertHigh10DirectSubDeblockCABACPayload(t *testing.T, raw []byte, tt high10DirectSubDeblockFixture) {
	t.Helper()
	if len(raw) != 10 {
		t.Fatalf("B direct-sub deblock CABAC NAL size = %d, want 10", len(raw))
	}
	wantPrefix := []byte{0x01, 0x9e, byte(0x44 + tt.directSpatial), 0xe4, 0x9f}
	if !bytes.Equal(raw[:len(wantPrefix)], wantPrefix) {
		t.Fatalf("B direct-sub deblock CABAC header/alignment = %x, want %x", raw[:len(wantPrefix)], wantPrefix)
	}
	wantBody := []byte{0xbe, 0x27, 0xfe, 0xed, 0x80}
	if !bytes.Equal(raw[len(wantPrefix):], wantBody) {
		t.Fatalf("B direct-sub deblock CABAC body = %x, want %x", raw[len(wantPrefix):], wantBody)
	}
}
