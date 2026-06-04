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

const high10SliceBoundaryDeblockFrameSize = 3072

type high10SliceBoundaryDeblockFixture struct {
	name         string
	file         string
	cabac        int32
	bitstreamMD5 string
	rawVideoMD5  string
	frameMD5     []string
}

func TestHigh10SliceBoundaryDeblockFixtureSyntax(t *testing.T) {
	for _, tt := range high10SliceBoundaryDeblockFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := readHigh10SliceBoundaryDeblockFixture(t, tt)
			assertHigh10SliceBoundaryDeblockFixtureSyntax(t, data, tt)
		})
	}
}

func TestDecodeAnnexBHigh10SliceBoundaryDeblockFrames(t *testing.T) {
	for _, tt := range high10SliceBoundaryDeblockFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := readHigh10SliceBoundaryDeblockFixture(t, tt)
			assertHigh10SliceBoundaryDeblockFixtureSyntax(t, data, tt)

			frames, err := NewDecoder().DecodeAnnexBFrames(data)
			if err != nil {
				t.Fatalf("decode High10 slice-boundary deblock fixture: %v", err)
			}
			assertHigh10SliceBoundaryDeblockFrames(t, frames, tt)
		})
	}
}

func TestDecodeAVCHigh10SliceBoundaryDeblockFrames(t *testing.T) {
	for _, tt := range high10SliceBoundaryDeblockFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := readHigh10SliceBoundaryDeblockFixture(t, tt)
			for _, nalLengthSize := range []int{2, 3, 4} {
				frames, err := NewDecoder().DecodeAVCFrames(annexBToAVC(t, data, nalLengthSize), nalLengthSize)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: %v", nalLengthSize, err)
				}
				assertHigh10SliceBoundaryDeblockFrames(t, frames, tt)
			}
		})
	}
}

func TestDecodeConfiguredAVCHigh10SliceBoundaryDeblockFrames(t *testing.T) {
	for _, tt := range high10SliceBoundaryDeblockFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := readHigh10SliceBoundaryDeblockFixture(t, tt)
			assertHigh10SliceBoundaryDeblockFixtureSyntax(t, data, tt)
			for _, nalLengthSize := range []int{2, 3, 4} {
				config, packet := annexBToAVCConfigAndPacket(t, data, nalLengthSize)
				frames, err := NewDecoder().DecodeAVCFramesWithConfigurationRecord(config, packet)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: DecodeAVCFramesWithConfigurationRecord: %v", nalLengthSize, err)
				}
				assertHigh10SliceBoundaryDeblockFrames(t, frames, tt)
			}
		})
	}
}

func TestDecodeConfiguredAccessUnitSamplesHigh10SliceBoundaryDeblockFrames(t *testing.T) {
	for _, tt := range high10SliceBoundaryDeblockFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := readHigh10SliceBoundaryDeblockFixture(t, tt)
			assertHigh10SliceBoundaryDeblockFixtureSyntax(t, data, tt)
			for _, nalLengthSize := range []int{2, 3, 4} {
				config, samples := annexBToAVCConfigAndAccessUnitSamples(t, data, nalLengthSize)
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
				assertHigh10SliceBoundaryDeblockFrames(t, frames, tt)
			}
		})
	}
}

func TestDecodeAutoConfiguredAccessUnitSamplesHigh10SliceBoundaryDeblockFrames(t *testing.T) {
	for _, tt := range high10SliceBoundaryDeblockFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := readHigh10SliceBoundaryDeblockFixture(t, tt)
			assertHigh10SliceBoundaryDeblockFixtureSyntax(t, data, tt)
			for _, nalLengthSize := range []int{2, 3, 4} {
				config, samples := annexBToAVCConfigAndAccessUnitSamples(t, data, nalLengthSize)
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
				assertHigh10SliceBoundaryDeblockFrames(t, frames, tt)
			}
		})
	}
}

func TestFFmpegRawVideoMD5OracleHigh10SliceBoundaryDeblock(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}
	for _, tt := range high10SliceBoundaryDeblockFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := readHigh10SliceBoundaryDeblockFixture(t, tt)
			assertHigh10SliceBoundaryDeblockFixtureSyntax(t, data, tt)
			assertFFmpegHigh10SliceBoundaryDeblockRawVideoOracle(t, data, tt)
		})
	}
}

func high10SliceBoundaryDeblockFixtures() []high10SliceBoundaryDeblockFixture {
	return []high10SliceBoundaryDeblockFixture{
		{
			name:         "cavlc",
			file:         "high10_slice_boundary_deblock_cavlc.h264",
			cabac:        0,
			bitstreamMD5: "c929a27027d7d3e77041ac3ed79e13a1",
			rawVideoMD5:  "fc65b48f2855bd3a33b1f3cc1a6e9e16",
			frameMD5: []string{
				"07f4ecbe2f86634c4de5b715ce1183c5",
				"2395db9c9fd32c34e3705708c566177e",
			},
		},
		{
			name:         "cabac",
			file:         "high10_slice_boundary_deblock_cabac.h264",
			cabac:        1,
			bitstreamMD5: "713139081e4cb3b74959cb4f5ab8ebae",
			rawVideoMD5:  "37eab6fac969fa7b9e57291bc2cf998e",
			frameMD5: []string{
				"2cdab993b06016b46acbaf9d161f7cc8",
				"9c3bb013624f83c5d2f61ee96f510107",
			},
		},
	}
}

func readHigh10SliceBoundaryDeblockFixture(t *testing.T, tt high10SliceBoundaryDeblockFixture) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "h264", tt.file))
	if err != nil {
		t.Fatal(err)
	}
	sum := md5.Sum(data)
	if got := hex.EncodeToString(sum[:]); got != tt.bitstreamMD5 {
		t.Fatalf("%s bitstream md5 = %s, want %s", tt.name, got, tt.bitstreamMD5)
	}
	return data
}

func assertHigh10SliceBoundaryDeblockFrames(t *testing.T, frames []*Frame, fixture high10SliceBoundaryDeblockFixture) {
	t.Helper()
	if len(frames) != len(fixture.frameMD5) {
		t.Fatalf("frames = %d, want %d", len(frames), len(fixture.frameMD5))
	}
	assertHigh10FrameMD5Strings(t, frames, fixture.frameMD5)
	raw := make([]byte, 0, len(frames)*high10SliceBoundaryDeblockFrameSize)
	for i, frame := range frames {
		rawSize, err := frame.RawYUVSize()
		if err != nil {
			t.Fatalf("frame[%d] RawYUVSize: %v", i, err)
		}
		if rawSize != high10SliceBoundaryDeblockFrameSize {
			t.Fatalf("frame[%d] raw yuv size = %d, want %d", i, rawSize, high10SliceBoundaryDeblockFrameSize)
		}
		raw, err = frame.AppendRawYUVBytesLE(raw)
		if err != nil {
			t.Fatalf("frame[%d] AppendRawYUVBytesLE: %v", i, err)
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
	}
	sum := md5.Sum(raw)
	if got := hex.EncodeToString(sum[:]); got != fixture.rawVideoMD5 {
		t.Fatalf("rawvideo md5 = %s, want %s", got, fixture.rawVideoMD5)
	}
}

func assertHigh10SliceBoundaryDeblockFixtureSyntax(t *testing.T, data []byte, fixture high10SliceBoundaryDeblockFixture) {
	t.Helper()
	nals, err := h264.SplitAnnexB(data)
	if err != nil {
		t.Fatal(err)
	}
	var spsList [32]*h264.SPS
	var ppsList [256]*h264.PPS
	var gotVCL []h264.NALUnit
	for _, nal := range nals {
		switch nal.Type {
		case h264.NALSPS:
			sps, err := h264.DecodeSPS(nal.RBSP)
			if err != nil {
				t.Fatal(err)
			}
			if sps.ProfileIDC != 110 || sps.Width != 32 || sps.Height != 32 ||
				sps.ChromaFormatIDC != 1 || sps.BitDepthLuma != 10 || sps.BitDepthChroma != 10 {
				t.Fatalf("SPS profile/format = %d %dx%d chroma %d depth %d/%d, want High10 32x32 yuv420p10le",
					sps.ProfileIDC, sps.Width, sps.Height, sps.ChromaFormatIDC, sps.BitDepthLuma, sps.BitDepthChroma)
			}
			if sps.FrameMBSOnlyFlag != 1 || sps.MBAFF != 0 {
				t.Fatalf("SPS frame-only flags = frame_mbs_only:%d mbaff:%d, want 1/0", sps.FrameMBSOnlyFlag, sps.MBAFF)
			}
			spsList[sps.SPSID] = sps
		case h264.NALPPS:
			pps, err := h264.DecodePPS(nal.RBSP, &spsList)
			if err != nil {
				t.Fatal(err)
			}
			if pps.CABAC != fixture.cabac || pps.Transform8x8Mode != 0 {
				t.Fatalf("PPS cabac/8x8dct = %d/%d, want %d/0", pps.CABAC, pps.Transform8x8Mode, fixture.cabac)
			}
			if pps.RefCount != [2]uint32{1, 1} || pps.WeightedPred != 0 || pps.WeightedBipredIDC != 0 {
				t.Fatalf("PPS refs/weight = %v/%d/%d, want ref=1 and unweighted", pps.RefCount, pps.WeightedPred, pps.WeightedBipredIDC)
			}
			if pps.DeblockingFilterParametersPresent != 1 {
				t.Fatalf("PPS deblock params present = %d, want 1", pps.DeblockingFilterParametersPresent)
			}
			ppsList[pps.PPSID] = pps
		case h264.NALIDRSlice, h264.NALSlice:
			sh, err := h264.ParseSliceHeader(nal, &ppsList)
			if err != nil {
				t.Fatal(err)
			}
			if sh.PictureStructure != h264.PictureFrame || sh.DeblockingFilter != 2 {
				t.Fatalf("slice picture/deblock = %d/%d, want frame/slice-boundary", sh.PictureStructure, sh.DeblockingFilter)
			}
			if sh.SliceAlphaC0Offset != 0 || sh.SliceBetaOffset != 0 {
				t.Fatalf("slice deblock offsets = %d/%d, want 0/0", sh.SliceAlphaC0Offset, sh.SliceBetaOffset)
			}
			gotVCL = append(gotVCL, nal)
		case h264.NALSEI:
		default:
			t.Fatalf("unexpected NAL type %d", nal.Type)
		}
	}
	if len(gotVCL) != 4 {
		t.Fatalf("VCL NAL count = %d, want 4", len(gotVCL))
	}
	wantTypes := []h264.NALUnitType{h264.NALIDRSlice, h264.NALIDRSlice, h264.NALSlice, h264.NALSlice}
	wantSliceTypes := []int32{h264.PictureTypeI, h264.PictureTypeI, h264.PictureTypeP, h264.PictureTypeP}
	wantFirstMB := []uint32{0, 2, 0, 2}
	for i, nal := range gotVCL {
		sh, err := h264.ParseSliceHeader(nal, &ppsList)
		if err != nil {
			t.Fatal(err)
		}
		if nal.Type != wantTypes[i] || sh.SliceTypeNoS != wantSliceTypes[i] || sh.FirstMBAddr != wantFirstMB[i] {
			t.Fatalf("VCL[%d] type/slice/firstMB = %d/%d/%d, want %d/%d/%d",
				i, nal.Type, sh.SliceTypeNoS, sh.FirstMBAddr, wantTypes[i], wantSliceTypes[i], wantFirstMB[i])
		}
		if sh.SliceTypeNoS == h264.PictureTypeP {
			if sh.ListCount != 1 || sh.RefCount[0] != 1 || sh.PredWeightTable.UseWeight != 0 || sh.PredWeightTable.UseWeightChroma != 0 {
				t.Fatalf("P slice lists/ref0/weights = %d/%d/%d/%d, want one L0 ref and unweighted",
					sh.ListCount, sh.RefCount[0], sh.PredWeightTable.UseWeight, sh.PredWeightTable.UseWeightChroma)
			}
		}
	}
}

func annexBToAVCConfigAndAccessUnitSamples(t *testing.T, data []byte, nalLengthSize int) ([]byte, [][]byte) {
	t.Helper()
	if nalLengthSize < 1 || nalLengthSize > 4 {
		t.Fatalf("invalid nalLengthSize %d", nalLengthSize)
	}
	nals, err := h264.SplitAnnexB(data)
	if err != nil {
		t.Fatal(err)
	}
	var spsList [32]*h264.SPS
	var ppsList [256]*h264.PPS
	var spsNals [][]byte
	var ppsNals [][]byte
	var samples [][]byte
	var sample []byte
	hasVCL := false
	for _, nal := range nals {
		switch nal.Type {
		case h264.NALSPS:
			sps, err := h264.DecodeSPS(nal.RBSP)
			if err != nil {
				t.Fatal(err)
			}
			spsList[sps.SPSID] = sps
			spsNals = append(spsNals, nal.Raw)
		case h264.NALPPS:
			pps, err := h264.DecodePPS(nal.RBSP, &spsList)
			if err != nil {
				t.Fatal(err)
			}
			ppsList[pps.PPSID] = pps
			ppsNals = append(ppsNals, nal.Raw)
		default:
			isVCL := nal.Type == h264.NALSlice || nal.Type == h264.NALIDRSlice
			if isVCL {
				sh, err := h264.ParseSliceHeader(nal, &ppsList)
				if err != nil {
					t.Fatal(err)
				}
				if hasVCL && sh.FirstMBAddr == 0 {
					samples = append(samples, sample)
					sample = nil
					hasVCL = false
				}
			}
			sample = appendAVCNALUnit(t, sample, nal.Raw, nalLengthSize)
			if isVCL {
				hasVCL = true
			}
		}
	}
	if len(sample) != 0 {
		samples = append(samples, sample)
	}
	if len(spsNals) == 0 || len(spsNals) > 31 || len(ppsNals) == 0 || len(ppsNals) > 255 {
		t.Fatalf("parameter set counts: sps=%d pps=%d", len(spsNals), len(ppsNals))
	}
	config := append([]byte{1, spsNals[0][1], spsNals[0][2], spsNals[0][3], byte(0xfc | (nalLengthSize - 1)), byte(0xe0 | len(spsNals))}, nil...)
	for _, raw := range spsNals {
		config = appendAVCConfigNALUnit(t, config, raw)
	}
	config = append(config, byte(len(ppsNals)))
	for _, raw := range ppsNals {
		config = appendAVCConfigNALUnit(t, config, raw)
	}
	return config, samples
}

func assertFFmpegHigh10SliceBoundaryDeblockRawVideoOracle(t *testing.T, data []byte, fixture high10SliceBoundaryDeblockFixture) {
	t.Helper()
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
	for i, want := range fixture.frameMD5 {
		line := []byte(fmt.Sprintf("0, %10d, %10d,        1, %8d, %s", i, i, high10SliceBoundaryDeblockFrameSize, want))
		if !bytes.Contains(framemd5Out, line) {
			t.Fatalf("frame[%d] missing %q in framemd5:\n%s", i, line, framemd5Out)
		}
	}

	rawvideo := exec.Command("ffmpeg",
		"-v", "error",
		"-f", "h264",
		"-i", path,
		"-an", "-sn", "-dn",
		"-pix_fmt", "yuv420p10le",
		"-f", "rawvideo",
		"-",
	)
	raw, err := rawvideo.Output()
	if err != nil {
		t.Fatalf("ffmpeg rawvideo: %v", err)
	}
	if len(raw) != len(fixture.frameMD5)*high10SliceBoundaryDeblockFrameSize {
		t.Fatalf("rawvideo size = %d, want %d", len(raw), len(fixture.frameMD5)*high10SliceBoundaryDeblockFrameSize)
	}
	sum := md5.Sum(raw)
	if got := hex.EncodeToString(sum[:]); got != fixture.rawVideoMD5 {
		t.Fatalf("rawvideo md5 = %s, want %s", got, fixture.rawVideoMD5)
	}
}
