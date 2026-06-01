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

const (
	high12IntraPCMFrameMD5    = "c361aa6cd60683fabf155b7e0baec348"
	high12IntraPCMRawVideoMD5 = "c361aa6cd60683fabf155b7e0baec348"
)

func TestHigh12IntraPCMFixtureSyntax(t *testing.T) {
	assertHigh12IntraPCMFixtureSyntax(t, readHigh12IntraPCMFixture(t))
}

func TestDecodeAnnexBHigh12IntraPCMFrame(t *testing.T) {
	data := readHigh12IntraPCMFixture(t)
	assertHigh12IntraPCMFixtureSyntax(t, data)

	frames, err := NewDecoder().DecodeAnnexBFrames(data)
	if err != nil {
		t.Fatalf("decode High12 IntraPCM Annex B: %v", err)
	}
	assertHigh12IntraPCMFrames(t, frames)
}

func TestDecodeAVCHigh12IntraPCMFrame(t *testing.T) {
	data := readHigh12IntraPCMFixture(t)
	assertHigh12IntraPCMFixtureSyntax(t, data)

	for _, nalLengthSize := range []int{2, 3, 4} {
		frames, err := NewDecoder().DecodeAVCFrames(annexBToAVC(t, data, nalLengthSize), nalLengthSize)
		if err != nil {
			t.Fatalf("nalLengthSize=%d: %v", nalLengthSize, err)
		}
		assertHigh12IntraPCMFrames(t, frames)
	}
}

func TestDecodeAVCWithConfigurationRecordHigh12IntraPCMFrame(t *testing.T) {
	data := readHigh12IntraPCMFixture(t)
	assertHigh12IntraPCMFixtureSyntax(t, data)

	for _, nalLengthSize := range []int{2, 3, 4} {
		config, packet := annexBToAVCConfigAndPacket(t, data, nalLengthSize)
		frames, err := NewDecoder().DecodeAVCFramesWithConfigurationRecord(config, packet)
		if err != nil {
			t.Fatalf("nalLengthSize=%d: %v", nalLengthSize, err)
		}
		assertHigh12IntraPCMFrames(t, frames)
	}
}

func TestFFmpegRawVideoMD5OracleHigh12IntraPCM(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}

	path := high12IntraPCMFixturePath(t)
	framemd5 := exec.Command("ffmpeg",
		"-v", "error",
		"-f", "h264",
		"-i", path,
		"-an", "-sn", "-dn",
		"-pix_fmt", "yuv420p12le",
		"-f", "framemd5",
		"-",
	)
	framemd5Out, err := framemd5.Output()
	if err != nil {
		t.Fatalf("ffmpeg framemd5: %v", err)
	}
	line := []byte(fmt.Sprintf("0, %10d, %10d,        1,      768, %s", 0, 0, high12IntraPCMFrameMD5))
	if !bytes.Contains(framemd5Out, line) {
		t.Fatalf("missing %q in framemd5:\n%s", line, framemd5Out)
	}

	rawvideo := exec.Command("ffmpeg",
		"-v", "error",
		"-f", "h264",
		"-i", path,
		"-an", "-sn", "-dn",
		"-pix_fmt", "yuv420p12le",
		"-f", "rawvideo",
		"-",
	)
	raw, err := rawvideo.Output()
	if err != nil {
		t.Fatalf("ffmpeg rawvideo: %v", err)
	}
	if len(raw) != 768 {
		t.Fatalf("rawvideo size = %d, want 768", len(raw))
	}
	sum := md5.Sum(raw)
	if got := hex.EncodeToString(sum[:]); got != high12IntraPCMRawVideoMD5 {
		t.Fatalf("rawvideo md5 = %s, want %s", got, high12IntraPCMRawVideoMD5)
	}
}

func readHigh12IntraPCMFixture(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile(high12IntraPCMFixturePath(t))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func high12IntraPCMFixturePath(t *testing.T) string {
	t.Helper()
	return filepath.Join("testdata", "h264", "high12_intrapcm_cavlc_i.h264")
}

func assertHigh12IntraPCMFrames(t *testing.T, frames []*Frame) {
	t.Helper()
	if len(frames) != 1 {
		t.Fatalf("frames = %d, want 1", len(frames))
	}
	frame := frames[0]
	if frame.Width != 16 || frame.Height != 16 ||
		frame.ChromaFormatIDC != 1 || frame.BitDepthLuma != 12 || frame.BitDepthChroma != 12 {
		t.Fatalf("frame format = %dx%d chroma %d depth %d/%d, want 16x16 yuv420p12le",
			frame.Width, frame.Height, frame.ChromaFormatIDC, frame.BitDepthLuma, frame.BitDepthChroma)
	}
	if pixFmt, err := frame.RawPixelFormat(); err != nil || pixFmt != "yuv420p12le" {
		t.Fatalf("RawPixelFormat = %q/%v, want yuv420p12le/nil", pixFmt, err)
	}
	if size, err := frame.RawYUVSize(); err != nil || size != 768 {
		t.Fatalf("RawYUVSize = %d/%v, want 768/nil", size, err)
	}
	raw, err := frame.AppendRawYUVBytesLE(nil)
	if err != nil {
		t.Fatalf("AppendRawYUVBytesLE: %v", err)
	}
	sum := md5.Sum(raw)
	if got := hex.EncodeToString(sum[:]); got != high12IntraPCMFrameMD5 {
		t.Fatalf("frame raw md5 = %s, want %s", got, high12IntraPCMFrameMD5)
	}
	if _, err := frame.AppendRawYUV(nil); err != ErrUnsupported {
		t.Fatalf("AppendRawYUV high12 error = %v, want ErrUnsupported", err)
	}
}

func assertHigh12IntraPCMFixtureSyntax(t *testing.T, data []byte) {
	t.Helper()
	nals, err := h264.SplitAnnexB(data)
	if err != nil {
		t.Fatal(err)
	}

	var spsList [32]*h264.SPS
	var ppsList [256]*h264.PPS
	var gotVCL []h264.NALUnitType
	for _, nal := range nals {
		switch nal.Type {
		case h264.NALSPS:
			sps, err := h264.DecodeSPS(nal.RBSP)
			if err != nil {
				t.Fatal(err)
			}
			if sps.ProfileIDC != 244 || sps.Width != 16 || sps.Height != 16 ||
				sps.ChromaFormatIDC != 1 || sps.BitDepthLuma != 12 || sps.BitDepthChroma != 12 {
				t.Fatalf("SPS profile/format = %d %dx%d chroma %d depth %d/%d, want High 4:4:4 Predictive-compatible 16x16 yuv420p12le",
					sps.ProfileIDC, sps.Width, sps.Height, sps.ChromaFormatIDC, sps.BitDepthLuma, sps.BitDepthChroma)
			}
			spsList[sps.SPSID] = sps
		case h264.NALPPS:
			pps, err := h264.DecodePPS(nal.RBSP, &spsList)
			if err != nil {
				t.Fatal(err)
			}
			if pps.CABAC != 0 || pps.DeblockingFilterParametersPresent == 0 {
				t.Fatalf("PPS CABAC/deblock-present = %d/%d, want CAVLC/deblock params", pps.CABAC, pps.DeblockingFilterParametersPresent)
			}
			ppsList[pps.PPSID] = pps
		case h264.NALIDRSlice, h264.NALSlice:
			sh, err := h264.ParseSliceHeader(nal, &ppsList)
			if err != nil {
				t.Fatal(err)
			}
			if sh.PictureStructure != h264.PictureFrame || sh.SliceTypeNoS != h264.PictureTypeI ||
				sh.DeblockingFilter != 0 || sh.QScale != 26 {
				t.Fatalf("slice picture/type/deblock/qp = %d/%d/%d/%d, want frame/I/disabled/26",
					sh.PictureStructure, sh.SliceTypeNoS, sh.DeblockingFilter, sh.QScale)
			}
			gotVCL = append(gotVCL, nal.Type)
		}
	}
	if len(gotVCL) != 1 || gotVCL[0] != h264.NALIDRSlice {
		t.Fatalf("VCL NALs = %v, want one IDR slice", gotVCL)
	}
}
