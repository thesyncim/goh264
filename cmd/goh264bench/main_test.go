// SPDX-License-Identifier: LGPL-2.1-or-later

package main

import (
	"crypto/md5"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSampleStats(t *testing.T) {
	stats := sampleStats([]benchSample{
		{ElapsedMS: 4},
		{ElapsedMS: 2},
		{ElapsedMS: 8},
		{ElapsedMS: 6},
	})
	if stats.mean != 5 {
		t.Fatalf("mean = %v, want 5", stats.mean)
	}
	if stats.median != 5 {
		t.Fatalf("median = %v, want 5", stats.median)
	}
	if stats.min != 2 || stats.max != 8 {
		t.Fatalf("min/max = %v/%v, want 2/8", stats.min, stats.max)
	}
	if stats.stddev == 0 || stats.cv == 0 {
		t.Fatalf("stddev/cv = %v/%v, want non-zero", stats.stddev, stats.cv)
	}
}

func TestResultFromSamplesAggregatesRepeats(t *testing.T) {
	result := resultFromSamples("goh264", "in.h264", 2, 2, 1, true, 3, 12, []benchSample{
		{ElapsedMS: 10, TotalFrames: 6, TotalBytes: 24, AllocBytes: 100, Allocs: 4, RawMD5: "abc"},
		{ElapsedMS: 20, TotalFrames: 6, TotalBytes: 24, AllocBytes: 200, Allocs: 6, RawMD5: "abc"},
	}, "abc", "")
	if result.TotalFrames != 12 || result.TotalBytes != 48 {
		t.Fatalf("totals = %d/%d, want 12/48", result.TotalFrames, result.TotalBytes)
	}
	if result.AllocBytes != 300 || result.Allocs != 10 {
		t.Fatalf("allocs = %d/%d, want 300/10", result.AllocBytes, result.Allocs)
	}
	if result.MeanElapsedMS != 15 || result.MedianElapsedMS != 15 || result.ElapsedMS != 30 {
		t.Fatalf("elapsed summary = total %v mean %v median %v, want 30/15/15",
			result.ElapsedMS, result.MeanElapsedMS, result.MedianElapsedMS)
	}
}

func TestReadBenchCorpusManifestAndValidate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.jsonl")
	text := `
# comment
{"id":"ok","path":"sample.h264","format":"annexb","expect":"decode-ok","pix_fmt":"yuv420p","frame_count":2,"frame_size":16,"bitstream_md5":"00112233445566778899aabbccddeeff","rawvideo_md5":"ffeeddccbbaa99887766554433221100","frame_md5":["aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"],"surfaces":["annexb"]}

{"id":"url-ok","url":"https://example.invalid/sample.264","format":"annexb","expect":"decode-ok","pix_fmt":"yuv420p","frame_count":2,"frame_size":16,"bitstream_md5":"00112233445566778899aabbccddeeff","rawvideo_md5":"ffeeddccbbaa99887766554433221100","surfaces":["annexb"],"feature_tags":["external"],"source":"test"}

{"id":"unsupported","path":"later.h264","format":"annexb","expect":"unsupported","guard_tags":["future"]}
`
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}

	entries, err := readBenchCorpusManifest(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Fatalf("entries = %d, want 3", len(entries))
	}
	if err := validateBenchCorpusEntry(entries[0]); err != nil {
		t.Fatalf("validate decode-ok: %v", err)
	}
	if err := validateBenchCorpusEntry(entries[1]); err != nil {
		t.Fatalf("validate url decode-ok: %v", err)
	}
	if err := validateBenchCorpusEntry(entries[2]); err == nil || !strings.Contains(err.Error(), "decode-ok") {
		t.Fatalf("validate unsupported err = %v, want decode-ok rejection", err)
	}
}

func TestValidateBenchBitstreamMD5(t *testing.T) {
	data := []byte("h264")
	sum := md5.Sum(data)
	entry := benchCorpusEntry{ID: "sample", BitstreamMD5: hex.EncodeToString(sum[:])}
	if err := validateBenchBitstreamMD5(entry, data); err != nil {
		t.Fatalf("validate bitstream md5: %v", err)
	}
	entry.BitstreamMD5 = "00000000000000000000000000000000"
	if err := validateBenchBitstreamMD5(entry, data); err == nil {
		t.Fatal("validate bitstream md5 mismatch err = nil, want error")
	}
}

func TestAnnotateBenchResultWithOracle(t *testing.T) {
	entry := benchCorpusEntry{
		ID:          "oracle",
		PixFmt:      "yuv420p",
		FrameCount:  2,
		FrameSize:   16,
		RawVideoMD5: "ffeeddccbbaa99887766554433221100",
	}
	result := benchResult{
		Name:           "goh264",
		FramesPerIter:  2,
		BytesPerIter:   32,
		RawPixelFormat: "yuv420p",
		RawMD5:         "ffeeddccbbaa99887766554433221100",
	}
	if err := annotateBenchResultWithOracle(&result, entry); err != nil {
		t.Fatalf("annotate oracle: %v", err)
	}
	if result.EntryID != entry.ID || result.ExpectedBytes != 32 || result.ParityStatus != "rawvideo-md5-ok" {
		t.Fatalf("annotated result = %+v", result)
	}

	bad := result
	bad.RawMD5 = "00000000000000000000000000000000"
	if err := annotateBenchResultWithOracle(&bad, entry); err == nil {
		t.Fatal("annotate raw md5 mismatch err = nil, want error")
	}
}
