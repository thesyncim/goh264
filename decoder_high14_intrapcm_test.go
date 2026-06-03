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
	high14IntraPCMBitstreamMD5 = "7aa11e6969b38cf018c6f8118ec77e64"
	high14IntraPCMFrameMD5     = "bfcda7ad1ae9016f683f1ba787d06e94"
	high14IntraPCMRawVideoMD5  = "bfcda7ad1ae9016f683f1ba787d06e94"
)

func TestHigh14IntraPCMFixtureSyntax(t *testing.T) {
	assertHigh14IntraPCMFixtureSyntax(t, readHigh14IntraPCMFixture(t))
}

func TestDecodeAnnexBHigh14IntraPCMFrame(t *testing.T) {
	data := readHigh14IntraPCMFixture(t)
	assertHigh14IntraPCMFixtureSyntax(t, data)

	frames, err := NewDecoder().DecodeAnnexBFrames(data)
	if err != nil {
		t.Fatalf("decode High14 IntraPCM Annex B: %v", err)
	}
	assertHigh14IntraPCMFrames(t, frames)
}

func TestDecodeAVCHigh14IntraPCMFrame(t *testing.T) {
	data := readHigh14IntraPCMFixture(t)
	assertHigh14IntraPCMFixtureSyntax(t, data)

	for _, nalLengthSize := range []int{2, 3, 4} {
		frames, err := NewDecoder().DecodeAVCFrames(annexBToAVC(t, data, nalLengthSize), nalLengthSize)
		if err != nil {
			t.Fatalf("nalLengthSize=%d: %v", nalLengthSize, err)
		}
		assertHigh14IntraPCMFrames(t, frames)
	}
}

func TestDecodeAVCWithConfigurationRecordHigh14IntraPCMFrame(t *testing.T) {
	data := readHigh14IntraPCMFixture(t)
	assertHigh14IntraPCMFixtureSyntax(t, data)

	for _, nalLengthSize := range []int{2, 3, 4} {
		config, packet := annexBToAVCConfigAndPacket(t, data, nalLengthSize)
		frames, err := NewDecoder().DecodeAVCFramesWithConfigurationRecord(config, packet)
		if err != nil {
			t.Fatalf("nalLengthSize=%d: %v", nalLengthSize, err)
		}
		assertHigh14IntraPCMFrames(t, frames)
	}
}

func TestFFmpegRawVideoMD5OracleHigh14IntraPCM(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}

	path := high14IntraPCMFixturePath(t)
	framemd5 := exec.Command("ffmpeg",
		"-v", "error",
		"-xerror",
		"-f", "h264",
		"-i", path,
		"-an", "-sn", "-dn",
		"-pix_fmt", "yuv420p14le",
		"-f", "framemd5",
		"-",
	)
	framemd5Out, err := framemd5.Output()
	if err != nil {
		t.Fatalf("ffmpeg framemd5: %v", err)
	}
	line := []byte(fmt.Sprintf("0, %10d, %10d,        1,      768, %s", 0, 0, high14IntraPCMFrameMD5))
	if !bytes.Contains(framemd5Out, line) {
		t.Fatalf("missing %q in framemd5:\n%s", line, framemd5Out)
	}

	rawvideo := exec.Command("ffmpeg",
		"-v", "error",
		"-xerror",
		"-f", "h264",
		"-i", path,
		"-an", "-sn", "-dn",
		"-pix_fmt", "yuv420p14le",
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
	if got := hex.EncodeToString(sum[:]); got != high14IntraPCMRawVideoMD5 {
		t.Fatalf("rawvideo md5 = %s, want %s", got, high14IntraPCMRawVideoMD5)
	}
}

func readHigh14IntraPCMFixture(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile(high14IntraPCMFixturePath(t))
	if err != nil {
		t.Fatal(err)
	}
	sum := md5.Sum(data)
	if got := hex.EncodeToString(sum[:]); got != high14IntraPCMBitstreamMD5 {
		t.Fatalf("High14 IntraPCM bitstream md5 = %s, want %s", got, high14IntraPCMBitstreamMD5)
	}
	return data
}

func high14IntraPCMFixturePath(t *testing.T) string {
	t.Helper()
	return filepath.Join("testdata", "h264", "high14_intrapcm_cavlc_i.h264")
}

func assertHigh14IntraPCMFrames(t *testing.T, frames []*Frame) {
	t.Helper()
	if len(frames) != 1 {
		t.Fatalf("frames = %d, want 1", len(frames))
	}
	frame := frames[0]
	if frame.Width != 16 || frame.Height != 16 ||
		frame.ChromaFormatIDC != 1 || frame.BitDepthLuma != 14 || frame.BitDepthChroma != 14 {
		t.Fatalf("frame format = %dx%d chroma %d depth %d/%d, want 16x16 yuv420p14le",
			frame.Width, frame.Height, frame.ChromaFormatIDC, frame.BitDepthLuma, frame.BitDepthChroma)
	}
	if pixFmt, err := frame.RawPixelFormat(); err != nil || pixFmt != "yuv420p14le" {
		t.Fatalf("RawPixelFormat = %q/%v, want yuv420p14le/nil", pixFmt, err)
	}
	if size, err := frame.RawYUVSize(); err != nil || size != 768 {
		t.Fatalf("RawYUVSize = %d/%v, want 768/nil", size, err)
	}
	raw, err := frame.AppendRawYUVBytesLE(nil)
	if err != nil {
		t.Fatalf("AppendRawYUVBytesLE: %v", err)
	}
	sum := md5.Sum(raw)
	if got := hex.EncodeToString(sum[:]); got != high14IntraPCMFrameMD5 {
		t.Fatalf("frame raw md5 = %s, want %s", got, high14IntraPCMFrameMD5)
	}
	if _, err := frame.AppendRawYUV(nil); err != ErrUnsupported {
		t.Fatalf("AppendRawYUV high14 error = %v, want ErrUnsupported", err)
	}
}

func assertHigh14IntraPCMFixtureSyntax(t *testing.T, data []byte) {
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
				sps.ChromaFormatIDC != 1 || sps.BitDepthLuma != 14 || sps.BitDepthChroma != 14 {
				t.Fatalf("SPS profile/format = %d %dx%d chroma %d depth %d/%d, want High 4:4:4 Predictive-compatible 16x16 yuv420p14le",
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
