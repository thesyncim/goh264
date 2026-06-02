// SPDX-License-Identifier: LGPL-2.1-or-later

package goh264

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
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

	failures := readH264CorpusManifest(t, defaultH264RealVectorFailureManifest)
	if filter := h264CorpusFilterTokens(); len(filter) != 0 {
		failures = filterH264CorpusEntries(failures, filter)
		if len(failures) == 0 {
			t.Fatalf("%s: no failure entries matched GOH264_CORPUS_FILTER=%q; available known-red filters: %s",
				defaultH264RealVectorFailureManifest, os.Getenv("GOH264_CORPUS_FILTER"), h264CorpusFailureFilterSummary(readH264CorpusManifest(t, defaultH264RealVectorFailureManifest)))
		}
	}

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

func h264GoAnnexBFrameMD5s(entry h264CorpusEntry, data []byte) ([]string, error) {
	frames, err := NewDecoder().DecodeAnnexBFrames(data)
	if err != nil {
		return nil, err
	}
	if len(frames) != entry.FrameCount {
		return nil, fmt.Errorf("frames = %d, want %d", len(frames), entry.FrameCount)
	}
	hashes := make([]string, 0, len(frames))
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
		sum := md5.Sum(raw)
		hashes = append(hashes, hex.EncodeToString(sum[:]))
	}
	return hashes, nil
}
