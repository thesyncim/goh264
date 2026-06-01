// SPDX-License-Identifier: LGPL-2.1-or-later

package goh264

import (
	"bufio"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const defaultH264CorpusManifest = "testdata/h264/corpus/manifest.jsonl"

func h264CorpusManifestPaths() []string {
	if manifests := os.Getenv("GOH264_CORPUS_MANIFESTS"); manifests != "" {
		var paths []string
		for _, path := range filepath.SplitList(manifests) {
			if path != "" {
				paths = append(paths, path)
			}
		}
		if len(paths) != 0 {
			return paths
		}
	}
	if manifest := os.Getenv("GOH264_CORPUS_MANIFEST"); manifest != "" {
		return []string{manifest}
	}
	return []string{defaultH264CorpusManifest}
}

type h264CorpusEntry struct {
	ID            string   `json:"id"`
	Path          string   `json:"path"`
	Format        string   `json:"format"`
	Expect        string   `json:"expect"`
	ExpectedError string   `json:"expected_error,omitempty"`
	PixFmt        string   `json:"pix_fmt,omitempty"`
	FrameCount    int      `json:"frame_count,omitempty"`
	FrameSize     int      `json:"frame_size,omitempty"`
	BitstreamMD5  string   `json:"bitstream_md5,omitempty"`
	RawVideoMD5   string   `json:"rawvideo_md5,omitempty"`
	FrameMD5      []string `json:"frame_md5,omitempty"`
	Surfaces      []string `json:"surfaces,omitempty"`
	GuardTags     []string `json:"guard_tags,omitempty"`
}

func TestH264CorpusManifest(t *testing.T) {
	for _, manifest := range h264CorpusManifestPaths() {
		manifest := manifest
		t.Run(filepath.Base(manifest), func(t *testing.T) {
			testH264CorpusManifest(t, manifest)
		})
	}
}

func testH264CorpusManifest(t *testing.T, manifest string) {
	entries := readH264CorpusManifest(t, manifest)
	if len(entries) == 0 {
		t.Fatalf("%s: no corpus entries", manifest)
	}

	baseDir := filepath.Dir(manifest)
	for _, entry := range entries {
		entry := entry
		t.Run(entry.ID, func(t *testing.T) {
			validateH264CorpusEntry(t, entry)
			path := entry.Path
			if !filepath.IsAbs(path) {
				path = filepath.Join(baseDir, path)
			}
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}
			assertCorpusBitstreamMD5(t, entry, data)
			for _, surface := range entry.Surfaces {
				surface := surface
				t.Run(surface, func(t *testing.T) {
					frames, err := decodeH264CorpusSurface(t, entry, surface, data)
					if entry.Expect == "unsupported" {
						assertH264CorpusUnsupported(t, entry, err)
						return
					}
					if err != nil {
						t.Fatalf("%s decode: %v", surface, err)
					}
					assertH264CorpusFrames(t, entry, frames)
				})
			}
		})
	}
}

func TestH264CorpusManifestPaths(t *testing.T) {
	t.Setenv("GOH264_CORPUS_MANIFEST", "")
	t.Setenv("GOH264_CORPUS_MANIFESTS", "")
	if got := h264CorpusManifestPaths(); len(got) != 1 || got[0] != defaultH264CorpusManifest {
		t.Fatalf("default manifests = %v, want %s", got, defaultH264CorpusManifest)
	}

	t.Setenv("GOH264_CORPUS_MANIFEST", "one.jsonl")
	if got := h264CorpusManifestPaths(); len(got) != 1 || got[0] != "one.jsonl" {
		t.Fatalf("single manifest = %v, want one.jsonl", got)
	}

	t.Setenv("GOH264_CORPUS_MANIFESTS", strings.Join([]string{"one.jsonl", "two.jsonl"}, string(os.PathListSeparator)))
	if got := h264CorpusManifestPaths(); len(got) != 2 || got[0] != "one.jsonl" || got[1] != "two.jsonl" {
		t.Fatalf("manifest list = %v, want one.jsonl/two.jsonl", got)
	}
}

func readH264CorpusManifest(t *testing.T, path string) []h264CorpusEntry {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open corpus manifest %s: %v", path, err)
	}
	defer f.Close()

	var entries []h264CorpusEntry
	scanner := bufio.NewScanner(f)
	for line := 1; scanner.Scan(); line++ {
		text := strings.TrimSpace(scanner.Text())
		if text == "" || strings.HasPrefix(text, "#") {
			continue
		}
		var entry h264CorpusEntry
		if err := json.Unmarshal([]byte(text), &entry); err != nil {
			t.Fatalf("%s:%d: %v", path, line, err)
		}
		entries = append(entries, entry)
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("read corpus manifest %s: %v", path, err)
	}
	return entries
}

func validateH264CorpusEntry(t *testing.T, entry h264CorpusEntry) {
	t.Helper()
	if entry.ID == "" || entry.Path == "" {
		t.Fatalf("entry id/path must be set: %+v", entry)
	}
	if entry.Format != "annexb" {
		t.Fatalf("%s: format = %q, want annexb", entry.ID, entry.Format)
	}
	if len(entry.Surfaces) == 0 {
		t.Fatalf("%s: surfaces must be non-empty", entry.ID)
	}
	for _, surface := range entry.Surfaces {
		switch surface {
		case "annexb", "avc", "configured-avc", "configured-samples", "auto":
		default:
			t.Fatalf("%s: unknown surface %q", entry.ID, surface)
		}
	}
	switch entry.Expect {
	case "decode-ok":
		if entry.BitstreamMD5 == "" || entry.RawVideoMD5 == "" || entry.PixFmt == "" {
			t.Fatalf("%s: decode-ok entries need bitstream_md5, rawvideo_md5, and pix_fmt", entry.ID)
		}
		if entry.FrameCount <= 0 || entry.FrameSize <= 0 || len(entry.FrameMD5) != entry.FrameCount {
			t.Fatalf("%s: frame_count/frame_size/frame_md5 mismatch", entry.ID)
		}
	case "unsupported":
		if len(entry.GuardTags) == 0 {
			t.Fatalf("%s: unsupported entries must name guard_tags", entry.ID)
		}
		if entry.ExpectedError != "" && entry.ExpectedError != "ErrUnsupported" {
			t.Fatalf("%s: expected_error = %q, want ErrUnsupported", entry.ID, entry.ExpectedError)
		}
	default:
		t.Fatalf("%s: expect = %q, want decode-ok or unsupported", entry.ID, entry.Expect)
	}
}

func decodeH264CorpusSurface(t *testing.T, entry h264CorpusEntry, surface string, data []byte) ([]*Frame, error) {
	t.Helper()
	switch surface {
	case "annexb":
		return NewDecoder().DecodeAnnexBFrames(data)
	case "avc":
		for _, nalLengthSize := range []int{2, 3, 4} {
			frames, err := NewDecoder().DecodeAVCFrames(annexBToAVC(t, data, nalLengthSize), nalLengthSize)
			if err != nil {
				return nil, fmt.Errorf("nal length size %d: %w", nalLengthSize, err)
			}
			if entry.Expect == "decode-ok" {
				assertH264CorpusFrames(t, entry, frames)
			}
			if nalLengthSize == 4 {
				return frames, nil
			}
		}
	case "configured-avc":
		for _, nalLengthSize := range []int{2, 3, 4} {
			config, packet := annexBToAVCConfigAndPacket(t, data, nalLengthSize)
			frames, err := NewDecoder().DecodeAVCFramesWithConfigurationRecord(config, packet)
			if err != nil {
				return nil, fmt.Errorf("nal length size %d: %w", nalLengthSize, err)
			}
			if entry.Expect == "decode-ok" {
				assertH264CorpusFrames(t, entry, frames)
			}
			if nalLengthSize == 4 {
				return frames, nil
			}
		}
	case "configured-samples":
		return decodeH264CorpusConfiguredSamples(t, entry, data, false)
	case "auto":
		return decodeH264CorpusConfiguredSamples(t, entry, data, true)
	}
	return nil, fmt.Errorf("unsupported corpus surface %q", surface)
}

func decodeH264CorpusConfiguredSamples(t *testing.T, entry h264CorpusEntry, data []byte, auto bool) ([]*Frame, error) {
	t.Helper()
	var final []*Frame
	for _, nalLengthSize := range []int{2, 3, 4} {
		config, samples := annexBToAVCConfigAndSamples(t, data, nalLengthSize)
		dec := NewDecoder()
		var frames []*Frame
		if auto {
			out, err := dec.DecodeFrames(config)
			if err != nil {
				return nil, fmt.Errorf("nal length size %d config: %w", nalLengthSize, err)
			}
			if len(out) != 0 {
				return nil, fmt.Errorf("nal length size %d config produced %d frames", nalLengthSize, len(out))
			}
		} else if _, err := dec.ParseAVCDecoderConfigurationRecord(config); err != nil {
			return nil, fmt.Errorf("nal length size %d config: %w", nalLengthSize, err)
		}
		for i, sample := range samples {
			var out []*Frame
			var err error
			if auto {
				out, err = dec.DecodeFrames(sample)
			} else {
				out, err = dec.DecodeConfiguredAVCFrames(sample)
			}
			if err != nil {
				return nil, fmt.Errorf("nal length size %d sample %d: %w", nalLengthSize, i, err)
			}
			frames = append(frames, out...)
		}
		var delayed []*Frame
		var err error
		if auto {
			delayed, err = dec.DecodeFrames(nil)
		} else {
			delayed, err = dec.FlushDelayedFrames()
		}
		if err != nil {
			return nil, fmt.Errorf("nal length size %d flush: %w", nalLengthSize, err)
		}
		frames = append(frames, delayed...)

		if auto {
			delayed, err = dec.DecodeFrames(nil)
		} else {
			delayed, err = dec.FlushDelayedFrames()
		}
		if err != nil {
			return nil, fmt.Errorf("nal length size %d second flush: %w", nalLengthSize, err)
		}
		if len(delayed) != 0 {
			return nil, fmt.Errorf("nal length size %d second flush produced %d frames", nalLengthSize, len(delayed))
		}

		if entry.Expect == "decode-ok" {
			assertH264CorpusFrames(t, entry, frames)
		}
		final = frames
	}
	return final, nil
}

func assertCorpusBitstreamMD5(t *testing.T, entry h264CorpusEntry, data []byte) {
	t.Helper()
	if entry.BitstreamMD5 == "" {
		return
	}
	sum := md5.Sum(data)
	if got := hex.EncodeToString(sum[:]); got != entry.BitstreamMD5 {
		t.Fatalf("%s: bitstream_md5 = %s, want %s", entry.ID, got, entry.BitstreamMD5)
	}
}

func assertH264CorpusFrames(t *testing.T, entry h264CorpusEntry, frames []*Frame) {
	t.Helper()
	if len(frames) != entry.FrameCount {
		t.Fatalf("%s: frames = %d, want %d", entry.ID, len(frames), entry.FrameCount)
	}
	rawHash := md5.New()
	var total int
	for i, frame := range frames {
		pixFmt, err := frame.RawPixelFormat()
		if err != nil {
			t.Fatalf("%s frame[%d] pix_fmt: %v", entry.ID, i, err)
		}
		if pixFmt != entry.PixFmt {
			t.Fatalf("%s frame[%d] pix_fmt = %s, want %s", entry.ID, i, pixFmt, entry.PixFmt)
		}
		raw, err := frame.AppendRawYUVBytesLE(nil)
		if err != nil {
			t.Fatalf("%s frame[%d] raw yuv: %v", entry.ID, i, err)
		}
		if len(raw) != entry.FrameSize {
			t.Fatalf("%s frame[%d] raw size = %d, want %d", entry.ID, i, len(raw), entry.FrameSize)
		}
		sum := md5.Sum(raw)
		if got := hex.EncodeToString(sum[:]); got != entry.FrameMD5[i] {
			t.Fatalf("%s frame[%d] md5 = %s, want %s", entry.ID, i, got, entry.FrameMD5[i])
		}
		if _, err := rawHash.Write(raw); err != nil {
			t.Fatalf("%s frame[%d] raw hash: %v", entry.ID, i, err)
		}
		total += len(raw)
	}
	if total != entry.FrameCount*entry.FrameSize {
		t.Fatalf("%s: raw total = %d, want %d", entry.ID, total, entry.FrameCount*entry.FrameSize)
	}
	if got := hex.EncodeToString(rawHash.Sum(nil)); got != entry.RawVideoMD5 {
		t.Fatalf("%s: rawvideo md5 = %s, want %s", entry.ID, got, entry.RawVideoMD5)
	}
}

func assertH264CorpusUnsupported(t *testing.T, entry h264CorpusEntry, err error) {
	t.Helper()
	if !errors.Is(err, ErrUnsupported) {
		t.Fatalf("%s: err = %v, want ErrUnsupported for guard tags %v", entry.ID, err, entry.GuardTags)
	}
}
