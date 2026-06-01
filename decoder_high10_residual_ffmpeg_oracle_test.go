// SPDX-License-Identifier: LGPL-2.1-or-later

package goh264

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/thesyncim/goh264/internal/h264"
)

var (
	high10ResidualP16x16X264MBLog    = regexp.MustCompile(`mb P\s+I16\.\.4:\s+0\.0%\s+0\.0%\s+0\.0%\s+P16\.\.4:\s+100\.0%.*skip:\s+0\.0%`)
	high10ResidualP16x16X264CodedLog = regexp.MustCompile(`coded y,uvDC,uvAC intra:.*inter:\s+100\.0%\s+100\.0%\s+100\.0%`)
)

func TestFFmpegGeneratedHigh10ResidualP16x16L0Oracle(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg/libx264 residual oracle")
	}

	ffmpeg := requireFFmpegHigh10Libx264(t)
	data := generateHigh10ResidualP16x16AnnexB(t, ffmpeg)
	assertHigh10ResidualP16x16FixtureSyntax(t, data)

	want := ffmpegHigh10ResidualFrameMD5s(t, ffmpeg, data)
	frames, err := NewDecoder().DecodeAnnexBFrames(data)
	if err != nil {
		t.Fatalf("Go decode High10 residual P16x16 Annex B: %v", err)
	}
	got := goHigh10ResidualFrameMD5s(t, frames)
	if len(got) != len(want) {
		t.Fatalf("Go frame md5 count = %d, want %d (%v)", len(got), len(want), want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("frame[%d] md5 = %s, want FFmpeg framemd5 %s\nGo:     %v\nFFmpeg: %v", i, got[i], want[i], got, want)
		}
	}
}

func requireFFmpegHigh10Libx264(t *testing.T) string {
	t.Helper()

	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Skip("ffmpeg not available")
	}
	cmd := exec.Command(ffmpeg, "-hide_banner", "-h", "encoder=libx264")
	out, err := cmd.CombinedOutput()
	if err != nil || !bytes.Contains(out, []byte("Encoder libx264")) {
		t.Skipf("ffmpeg libx264 encoder unavailable: %v", err)
	}
	if !bytes.Contains(out, []byte("yuv420p10le")) {
		t.Skip("ffmpeg libx264 encoder lacks yuv420p10le support")
	}
	return ffmpeg
}

func generateHigh10ResidualP16x16AnnexB(t *testing.T, ffmpeg string) []byte {
	t.Helper()

	path := filepath.Join(t.TempDir(), "high10-residual-p16x16.h264")
	cmd := exec.Command(ffmpeg,
		"-hide_banner",
		"-f", "lavfi",
		"-i", "testsrc2=size=16x16:rate=1:duration=2",
		"-vf", "format=yuv420p10le",
		"-frames:v", "2",
		"-c:v", "libx264",
		"-profile:v", "high10",
		"-preset", "ultrafast",
		"-tune", "fastdecode",
		"-qp", "18",
		"-x264opts", "keyint=2:min-keyint=2:scenecut=0:bframes=0:ref=1:weightp=0:8x8dct=0:partitions=none:no-deblock",
		"-f", "h264",
		"-y", path,
	)
	log, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("ffmpeg/libx264 generate High10 residual fixture: %v\n%s", err, log)
	}
	assertX264High10ResidualP16x16Log(t, string(log))

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Fatal("ffmpeg/libx264 generated empty H.264 fixture")
	}
	return data
}

func assertX264High10ResidualP16x16Log(t *testing.T, log string) {
	t.Helper()
	for _, want := range []string{
		"profile High 10",
		"frame I:1",
		"frame P:1",
	} {
		if !strings.Contains(log, want) {
			t.Fatalf("libx264 log missing %q:\n%s", want, log)
		}
	}
	if !high10ResidualP16x16X264MBLog.MatchString(log) {
		t.Fatalf("libx264 did not report one exact P16x16 non-skip macroblock:\n%s", log)
	}
	if !high10ResidualP16x16X264CodedLog.MatchString(log) {
		t.Fatalf("libx264 did not report coded inter residuals:\n%s", log)
	}
}

func assertHigh10ResidualP16x16FixtureSyntax(t *testing.T, data []byte) {
	t.Helper()

	nals, err := h264.SplitAnnexB(data)
	if err != nil {
		t.Fatal(err)
	}

	var spsList [32]*h264.SPS
	var ppsList [256]*h264.PPS
	var gotVCL []h264.NALUnitType
	var gotSlices []int32
	for _, nal := range nals {
		switch nal.Type {
		case h264.NALSPS:
			sps, err := h264.DecodeSPS(nal.RBSP)
			if err != nil {
				t.Fatal(err)
			}
			if sps.ProfileIDC != 110 || sps.Width != 16 || sps.Height != 16 ||
				sps.ChromaFormatIDC != 1 || sps.BitDepthLuma != 10 || sps.BitDepthChroma != 10 {
				t.Fatalf("SPS profile/format = %d %dx%d chroma %d depth %d/%d, want High10 16x16 yuv420p10le",
					sps.ProfileIDC, sps.Width, sps.Height, sps.ChromaFormatIDC, sps.BitDepthLuma, sps.BitDepthChroma)
			}
			spsList[sps.SPSID] = sps
		case h264.NALPPS:
			pps, err := h264.DecodePPS(nal.RBSP, &spsList)
			if err != nil {
				t.Fatal(err)
			}
			if pps.WeightedPred != 0 || pps.Transform8x8Mode != 0 || pps.RefCount[0] != 1 {
				t.Fatalf("PPS weighted/8x8/ref0 = %d/%d/%d, want 0/0/1", pps.WeightedPred, pps.Transform8x8Mode, pps.RefCount[0])
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
			if sh.SliceTypeNoS == h264.PictureTypeP && (sh.ListCount != 1 || sh.RefCount[0] != 1 || sh.PredWeightTable.UseWeight != 0) {
				t.Fatalf("P slice lists/ref0/weight = %d/%d/%d, want 1/1/0", sh.ListCount, sh.RefCount[0], sh.PredWeightTable.UseWeight)
			}
			gotVCL = append(gotVCL, nal.Type)
			gotSlices = append(gotSlices, sh.SliceTypeNoS)
		}
	}
	if len(gotVCL) != 2 || gotVCL[0] != h264.NALIDRSlice || gotVCL[1] != h264.NALSlice {
		t.Fatalf("VCL NALs = %v, want IDR then non-IDR", gotVCL)
	}
	if gotSlices[0] != h264.PictureTypeI || gotSlices[1] != h264.PictureTypeP {
		t.Fatalf("slice types = %v, want I then P", gotSlices)
	}
}

func ffmpegHigh10ResidualFrameMD5s(t *testing.T, ffmpeg string, data []byte) []string {
	t.Helper()

	path := writeHigh10ResidualTempH264(t, data)
	cmd := exec.Command(ffmpeg,
		"-v", "error",
		"-f", "h264",
		"-i", path,
		"-an", "-sn", "-dn",
		"-pix_fmt", "yuv420p10le",
		"-f", "framemd5",
		"-",
	)
	out, err := cmd.Output()
	if err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			t.Fatalf("ffmpeg framemd5: %v\n%s", err, exit.Stderr)
		}
		t.Fatalf("ffmpeg framemd5: %v", err)
	}
	return parseHigh10ResidualFrameMD5s(t, out)
}

func writeHigh10ResidualTempH264(t *testing.T, data []byte) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "fixture.h264")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func parseHigh10ResidualFrameMD5s(t *testing.T, out []byte) []string {
	t.Helper()

	var md5s []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Split(line, ",")
		if len(fields) != 6 {
			t.Fatalf("unexpected framemd5 line %q in:\n%s", line, out)
		}
		if size := strings.TrimSpace(fields[4]); size != "768" {
			t.Fatalf("framemd5 frame size = %s, want 768 in line %q", size, line)
		}
		hash := strings.TrimSpace(fields[5])
		if len(hash) != 32 {
			t.Fatalf("framemd5 hash = %q, want 32 hex chars in line %q", hash, line)
		}
		md5s = append(md5s, hash)
	}
	if len(md5s) != 2 {
		t.Fatalf("framemd5 frames = %d, want 2:\n%s", len(md5s), out)
	}
	return md5s
}

func goHigh10ResidualFrameMD5s(t *testing.T, frames []*Frame) []string {
	t.Helper()

	if len(frames) != 2 {
		t.Fatalf("Go frames = %d, want 2", len(frames))
	}
	got := make([]string, len(frames))
	for i, frame := range frames {
		if frame == nil {
			t.Fatalf("frame[%d] is nil", i)
		}
		if frame.Width != 16 || frame.Height != 16 || frame.ChromaFormatIDC != 1 ||
			frame.BitDepthLuma != 10 || frame.BitDepthChroma != 10 {
			t.Fatalf("frame[%d] format = %dx%d chroma %d depth %d/%d, want 16x16 High10 4:2:0",
				i, frame.Width, frame.Height, frame.ChromaFormatIDC, frame.BitDepthLuma, frame.BitDepthChroma)
		}
		if pixFmt, err := frame.RawPixelFormat(); err != nil || pixFmt != "yuv420p10le" {
			t.Fatalf("frame[%d] RawPixelFormat = %q/%v, want yuv420p10le/nil", i, pixFmt, err)
		}
		raw, err := frame.AppendRawYUVBytesLE(nil)
		if err != nil {
			t.Fatalf("frame[%d] AppendRawYUVBytesLE: %v", i, err)
		}
		if len(raw) != 768 {
			t.Fatalf("frame[%d] raw size = %d, want 768", i, len(raw))
		}
		sum := md5.Sum(raw)
		got[i] = hex.EncodeToString(sum[:])
	}
	return got
}
