// SPDX-License-Identifier: LGPL-2.1-or-later

package goh264

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestH264RealVectorFrameMD5Diagnostics(t *testing.T) {
	if os.Getenv("GOH264_REAL_VECTOR_FRAMEMD5") != "1" {
		t.Skip("set GOH264_REAL_VECTOR_FRAMEMD5=1 to fail selected raw-MD5 known-red vectors at the first divergent frame")
	}
	ffmpeg := os.Getenv("GOH264_FFMPEG")
	if ffmpeg == "" {
		ffmpeg = "ffmpeg"
	}
	if _, err := exec.LookPath(ffmpeg); err != nil {
		t.Fatalf("ffmpeg oracle not found (%s): %v", ffmpeg, err)
	}

	failures := h264RealVectorFailureEntriesForEnv(t, readH264CorpusManifest(t, defaultH264RealVectorFailureManifest))

	var selected int
	for _, entry := range failures {
		entry := entry
		if entry.KnownFailure == nil || entry.KnownFailure.Class != "raw-md5-mismatch" {
			continue
		}
		selected++
		t.Run(entry.ID, func(t *testing.T) {
			validateH264CorpusEntry(t, entry)
			if !h264CorpusEntryHasSurface(entry, "annexb") {
				t.Fatalf("%s: frame-MD5 diagnostics currently require an annexb surface", entry.ID)
			}
			path := materializeH264CorpusEntry(t, defaultH264RealVectorFailureManifest, entry)
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}
			assertCorpusBitstreamMD5(t, entry, data)

			want, err := h264FFmpegFrameMD5s(ffmpeg, path, entry.PixFmt)
			if err != nil {
				t.Fatalf("%s: ffmpeg frame-MD5 oracle: %v", entry.ID, err)
			}
			got, err := h264GoAnnexBFrameMD5s(entry, data)
			if err != nil {
				t.Fatalf("%s: Go frame-MD5 decode: %v", entry.ID, err)
			}
			if len(got) != len(want) {
				t.Fatalf("%s: frame-MD5 rows = %d, want %d", entry.ID, len(got), len(want))
			}
			if len(want) != entry.FrameCount {
				t.Fatalf("%s: ffmpeg frame-MD5 rows = %d, manifest wants %d", entry.ID, len(want), entry.FrameCount)
			}
			for i := range want {
				if got[i] != want[i] {
					t.Fatalf("%s: first divergent frame %d md5 = %s, want %s", entry.ID, i, got[i], want[i])
				}
			}
			t.Fatalf("%s: frame MD5s matched but failure ledger still records raw MD5 mismatch; remove it from %s",
				entry.ID, defaultH264RealVectorFailureManifest)
		})
	}
	if selected == 0 {
		t.Fatalf("%s: no raw-md5-mismatch known-red rows matched GOH264_CORPUS_FILTER=%q",
			defaultH264RealVectorFailureManifest, os.Getenv("GOH264_CORPUS_FILTER"))
	}
}

func TestH264RealVectorRawDiffDiagnostics(t *testing.T) {
	if os.Getenv("GOH264_REAL_VECTOR_RAWDIFF") != "1" {
		t.Skip("set GOH264_REAL_VECTOR_RAWDIFF=1 to fail selected raw-MD5 known-red vectors at the first divergent raw byte")
	}
	ffmpeg := os.Getenv("GOH264_FFMPEG")
	if ffmpeg == "" {
		ffmpeg = "ffmpeg"
	}
	if _, err := exec.LookPath(ffmpeg); err != nil {
		t.Fatalf("ffmpeg oracle not found (%s): %v", ffmpeg, err)
	}

	failures := h264RealVectorFailureEntriesForEnv(t, readH264CorpusManifest(t, defaultH264RealVectorFailureManifest))

	var selected int
	for _, entry := range failures {
		entry := entry
		if entry.KnownFailure == nil || entry.KnownFailure.Class != "raw-md5-mismatch" {
			continue
		}
		selected++
		t.Run(entry.ID, func(t *testing.T) {
			validateH264CorpusEntry(t, entry)
			if !h264CorpusEntryHasSurface(entry, "annexb") {
				t.Fatalf("%s: raw-diff diagnostics currently require an annexb surface", entry.ID)
			}
			path := materializeH264CorpusEntry(t, defaultH264RealVectorFailureManifest, entry)
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}
			assertCorpusBitstreamMD5(t, entry, data)

			wantRaw, err := h264FFmpegRawVideoBytes(ffmpeg, path, entry.PixFmt)
			if err != nil {
				t.Fatalf("%s: ffmpeg rawvideo oracle: %v", entry.ID, err)
			}
			if len(wantRaw) != entry.FrameCount*entry.FrameSize {
				t.Fatalf("%s: ffmpeg rawvideo size = %d, want %d", entry.ID, len(wantRaw), entry.FrameCount*entry.FrameSize)
			}
			gotFrames, err := h264GoAnnexBRawFrames(entry, data)
			if err != nil {
				t.Fatalf("%s: Go raw frame decode: %v", entry.ID, err)
			}
			if len(gotFrames) != entry.FrameCount {
				t.Fatalf("%s: Go raw frame rows = %d, want %d", entry.ID, len(gotFrames), entry.FrameCount)
			}
			for i, gotFrame := range gotFrames {
				wantFrame := wantRaw[i*entry.FrameSize : (i+1)*entry.FrameSize]
				if len(gotFrame.Raw) != len(wantFrame) {
					t.Fatalf("%s: frame %d raw size = %d, want %d", entry.ID, i, len(gotFrame.Raw), len(wantFrame))
				}
				if bytes.Equal(gotFrame.Raw, wantFrame) {
					continue
				}
				offset := h264FirstDifferentByte(gotFrame.Raw, wantFrame)
				gotSum := md5.Sum(gotFrame.Raw)
				wantSum := md5.Sum(wantFrame)
				t.Fatalf("%s: first divergent raw byte frame %d offset=%d %s go=%d ffmpeg=%d frame_md5=%s want_frame_md5=%s",
					entry.ID, i, offset, h264RawByteLocation(gotFrame, offset),
					gotFrame.Raw[offset], wantFrame[offset],
					hex.EncodeToString(gotSum[:]), hex.EncodeToString(wantSum[:]))
			}
			t.Fatalf("%s: raw frames matched but failure ledger still records raw MD5 mismatch; remove it from %s",
				entry.ID, defaultH264RealVectorFailureManifest)
		})
	}
	if selected == 0 {
		t.Fatalf("%s: no raw-md5-mismatch known-red rows matched GOH264_CORPUS_FILTER=%q",
			defaultH264RealVectorFailureManifest, os.Getenv("GOH264_CORPUS_FILTER"))
	}
}

func h264FFmpegFrameMD5s(ffmpeg string, path string, pixFmt string) ([]string, error) {
	if pixFmt == "" {
		return nil, fmt.Errorf("missing pix_fmt")
	}
	cmd := exec.Command(ffmpeg, "-v", "error", "-f", "h264", "-i", path, "-an", "-sn", "-dn", "-pix_fmt", pixFmt, "-f", "framemd5", "-")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%w: %s", err, bytes.TrimSpace(out))
	}
	var hashes []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Split(line, ",")
		if len(fields) < 6 {
			return nil, fmt.Errorf("malformed framemd5 line %q", line)
		}
		hash := strings.TrimSpace(fields[len(fields)-1])
		if len(hash) != md5.Size*2 {
			return nil, fmt.Errorf("malformed framemd5 hash %q in line %q", hash, line)
		}
		hashes = append(hashes, hash)
	}
	if len(hashes) == 0 {
		return nil, fmt.Errorf("ffmpeg produced no frame-MD5 rows")
	}
	return hashes, nil
}

func h264FFmpegRawVideoBytes(ffmpeg string, path string, pixFmt string) ([]byte, error) {
	if pixFmt == "" {
		return nil, fmt.Errorf("missing pix_fmt")
	}
	cmd := exec.Command(ffmpeg, "-v", "error", "-f", "h264", "-i", path, "-an", "-sn", "-dn", "-pix_fmt", pixFmt, "-f", "rawvideo", "-")
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, fmt.Errorf("%w: %s", err, bytes.TrimSpace(exitErr.Stderr))
		}
		return nil, err
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("ffmpeg produced no rawvideo bytes")
	}
	return out, nil
}

type h264RawFrameBytes struct {
	Raw             []byte
	Width           int
	Height          int
	ChromaFormatIDC uint32
	BytesPerSample  int
}

func h264GoAnnexBFrameMD5s(entry h264CorpusEntry, data []byte) ([]string, error) {
	rawFrames, err := h264GoAnnexBRawFrames(entry, data)
	if err != nil {
		return nil, err
	}
	hashes := make([]string, 0, len(rawFrames))
	for _, frame := range rawFrames {
		sum := md5.Sum(frame.Raw)
		hashes = append(hashes, hex.EncodeToString(sum[:]))
	}
	return hashes, nil
}

func h264GoAnnexBRawFrames(entry h264CorpusEntry, data []byte) ([]h264RawFrameBytes, error) {
	frames, err := NewDecoder().DecodeAnnexBFrames(data)
	if err != nil {
		return nil, err
	}
	if len(frames) != entry.FrameCount {
		return nil, fmt.Errorf("frames = %d, want %d", len(frames), entry.FrameCount)
	}
	rawFrames := make([]h264RawFrameBytes, 0, len(frames))
	for i, frame := range frames {
		pixFmt, err := frame.RawPixelFormat()
		if err != nil {
			return nil, fmt.Errorf("frame[%d] pix_fmt: %w", i, err)
		}
		if pixFmt != entry.PixFmt {
			return nil, fmt.Errorf("frame[%d] pix_fmt = %s, want %s", i, pixFmt, entry.PixFmt)
		}
		raw, err := frame.AppendRawYUVBytesLE(nil)
		if err != nil {
			return nil, fmt.Errorf("frame[%d] raw yuv: %w", i, err)
		}
		if len(raw) != entry.FrameSize {
			return nil, fmt.Errorf("frame[%d] raw size = %d, want %d", i, len(raw), entry.FrameSize)
		}
		bytesPerSample, err := frame.BytesPerSample()
		if err != nil {
			return nil, fmt.Errorf("frame[%d] bytes_per_sample: %w", i, err)
		}
		rawFrames = append(rawFrames, h264RawFrameBytes{
			Raw:             raw,
			Width:           frame.Width,
			Height:          frame.Height,
			ChromaFormatIDC: frame.ChromaFormatIDC,
			BytesPerSample:  bytesPerSample,
		})
	}
	return rawFrames, nil
}

func h264FirstDifferentByte(a []byte, b []byte) int {
	limit := len(a)
	if len(b) < limit {
		limit = len(b)
	}
	for i := 0; i < limit; i++ {
		if a[i] != b[i] {
			return i
		}
	}
	return limit
}

func h264RawByteLocation(frame h264RawFrameBytes, offset int) string {
	if offset < 0 || offset >= len(frame.Raw) || frame.Width <= 0 || frame.Height <= 0 || frame.BytesPerSample <= 0 {
		return "plane=?"
	}
	bytesPerSample := frame.BytesPerSample
	ySamples := frame.Width * frame.Height
	yBytes := ySamples * bytesPerSample
	if offset < yBytes {
		sample := offset / bytesPerSample
		return fmt.Sprintf("plane=Y x=%d y=%d sample_byte=%d", sample%frame.Width, sample/frame.Width, offset%bytesPerSample)
	}
	chromaWidth, chromaHeight, err := frameChromaSize(frame.Width, frame.Height, frame.ChromaFormatIDC)
	if err != nil || chromaWidth <= 0 || chromaHeight <= 0 {
		return "plane=?"
	}
	chromaBytes := chromaWidth * chromaHeight * bytesPerSample
	chromaOffset := offset - yBytes
	if chromaOffset < chromaBytes {
		sample := chromaOffset / bytesPerSample
		return fmt.Sprintf("plane=Cb x=%d y=%d sample_byte=%d", sample%chromaWidth, sample/chromaWidth, chromaOffset%bytesPerSample)
	}
	chromaOffset -= chromaBytes
	if chromaOffset < chromaBytes {
		sample := chromaOffset / bytesPerSample
		return fmt.Sprintf("plane=Cr x=%d y=%d sample_byte=%d", sample%chromaWidth, sample/chromaWidth, chromaOffset%bytesPerSample)
	}
	return "plane=?"
}
