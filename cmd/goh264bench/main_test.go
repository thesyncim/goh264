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
	if result.NSPerFrame == 0 || result.NSPerRawByte == 0 {
		t.Fatalf("derived rates = ns/frame %v ns/raw-byte %v, want non-zero", result.NSPerFrame, result.NSPerRawByte)
	}
}

func TestAnnotateBenchRatesReportsInputAndRawByteCosts(t *testing.T) {
	result := resultFromSamples("goh264", "in.h264", 2, 2, 1, true, 3, 10, []benchSample{
		{ElapsedMS: 10, TotalFrames: 6, TotalBytes: 20},
		{ElapsedMS: 10, TotalFrames: 6, TotalBytes: 20},
	}, "", "")
	result.InputBytesPerIter = 5
	annotateBenchRates(&result)
	if result.NSPerFrame != 20000000.0/12.0 {
		t.Fatalf("ns/frame = %v, want %v", result.NSPerFrame, 20000000.0/12.0)
	}
	if result.NSPerRawByte != 500000 {
		t.Fatalf("ns/raw-byte = %v, want 500000", result.NSPerRawByte)
	}
	if result.NSPerInputByte != 1000000 {
		t.Fatalf("ns/input-byte = %v, want 1000000", result.NSPerInputByte)
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

func TestReadBenchFailureLedgerAutoValidatesManifestSubset(t *testing.T) {
	dir := t.TempDir()
	row := `{"id":"fate/h264-conformance/frext-hcamff1-hhi","path":"HCAMFF1_HHI.264","url":"https://example.invalid/HCAMFF1_HHI.264","source":"FFmpeg FATE h264-conformance/FRext","format":"annexb","expect":"decode-ok","pix_fmt":"yuv420p","frame_count":10,"frame_size":152064,"bitstream_md5":"0dd0819dd9a276101a25259c0774c02c","rawvideo_md5":"2973f5376378cde879649160d4a46a98","surfaces":["annexb"],"feature_tags":["high","mbaff","field"]}`
	manifestPath := filepath.Join(dir, "manifest.jsonl")
	failurePath := filepath.Join(dir, "failures.jsonl")
	if err := os.WriteFile(manifestPath, []byte(row+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(failurePath, []byte(row+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	entries, err := readBenchCorpusManifest(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	failures, gotPath, err := readBenchFailureLedger(manifestPath, "auto", entries)
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != failurePath {
		t.Fatalf("failure ledger path = %s, want %s", gotPath, failurePath)
	}
	if _, ok := failures["fate/h264-conformance/frext-hcamff1-hhi"]; !ok || len(failures) != 1 {
		t.Fatalf("failures = %+v, want hcamff1 only", failures)
	}
}

func TestBenchManifestReportsKnownRedRowsWithoutBenchmarking(t *testing.T) {
	dir := t.TempDir()
	row := `{"id":"known-red","path":"missing.264","format":"annexb","expect":"decode-ok","pix_fmt":"yuv420p","frame_count":1,"frame_size":16,"bitstream_md5":"00112233445566778899aabbccddeeff","rawvideo_md5":"ffeeddccbbaa99887766554433221100","surfaces":["annexb"],"feature_tags":["unsupported"]}`
	manifestPath := filepath.Join(dir, "manifest.jsonl")
	failurePath := filepath.Join(dir, "failures.jsonl")
	if err := os.WriteFile(manifestPath, []byte(row+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(failurePath, []byte(row+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	report, err := benchManifest(manifestPath, 0, benchOptions{
		iters:         1,
		repeats:       1,
		rawOutput:     true,
		failureLedger: "auto",
	})
	if err != nil {
		t.Fatal(err)
	}
	if report.Metadata.CorpusKnownRed != 1 || report.Metadata.CorpusBench != 0 || report.Metadata.FailureLedger != failurePath {
		t.Fatalf("metadata = %+v, want known-red only with %s", report.Metadata, failurePath)
	}
	if len(report.Results) != 1 || !report.Results[0].Skipped || report.Results[0].ParityStatus != "known-red" {
		t.Fatalf("result = %+v, want visible known-red skipped row", report.Results)
	}
	if report.Results[0].Error == "" || !strings.Contains(report.Results[0].Error, "missing.264") {
		t.Fatalf("known-red error = %q, want missing input detail", report.Results[0].Error)
	}
}

func TestBenchManifestReportsUnsupportedRowsAsSkipped(t *testing.T) {
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "manifest.jsonl")
	row := `{"id":"future","path":"future.264","format":"annexb","expect":"unsupported","pix_fmt":"yuv420p","frame_count":1,"frame_size":16,"bitstream_md5":"00112233445566778899aabbccddeeff","rawvideo_md5":"ffeeddccbbaa99887766554433221100","surfaces":["annexb"],"guard_tags":["future"]}`
	if err := os.WriteFile(manifestPath, []byte(row+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	report, err := benchManifest(manifestPath, 0, benchOptions{
		iters:         1,
		repeats:       1,
		rawOutput:     true,
		failureLedger: "off",
	})
	if err != nil {
		t.Fatal(err)
	}
	if report.Metadata.CorpusSkipped != 1 || report.Metadata.CorpusBench != 0 {
		t.Fatalf("metadata = %+v, want one skipped row", report.Metadata)
	}
	if len(report.Results) != 1 || !report.Results[0].Skipped || report.Results[0].ParityStatus != "unsupported" {
		t.Fatalf("result = %+v, want unsupported skipped row", report.Results)
	}
}

func TestBenchCorpusFilter(t *testing.T) {
	entries := []benchCorpusEntry{
		{
			ID:          "fate/h264-conformance/caba3-sva-b",
			Path:        "CABA3_SVA_B.264",
			Source:      "FFmpeg FATE h264-conformance",
			Expect:      "decode-ok",
			PixFmt:      "yuv420p",
			Surfaces:    []string{"annexb"},
			FeatureTags: []string{"cabac", "main", "temporal-direct", "deblock"},
		},
		{
			ID:          "fate/h264-conformance/cvwp3-toshiba-e",
			Path:        "CVWP3_TOSHIBA_E.264",
			Source:      "FFmpeg FATE h264-conformance",
			Expect:      "decode-ok",
			PixFmt:      "yuv420p",
			Surfaces:    []string{"annexb"},
			FeatureTags: []string{"cabac", "implicit-weight-b", "weighted-bipred"},
		},
	}

	filtered := filterBenchCorpusEntries(entries, benchCorpusFilterTokens("cabac temporal"))
	if len(filtered) != 1 || filtered[0].ID != "fate/h264-conformance/caba3-sva-b" {
		t.Fatalf("filtered entries = %+v, want caba3 only", filtered)
	}

	filtered = filterBenchCorpusEntries(entries, benchCorpusFilterTokens("weighted"))
	if len(filtered) != 1 || filtered[0].ID != "fate/h264-conformance/cvwp3-toshiba-e" {
		t.Fatalf("filtered entries = %+v, want cvwp3 only", filtered)
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

func TestPreflightBenchFFmpegOracleRejectsStrictPixelFormatMismatch(t *testing.T) {
	err := preflightBenchFFmpegOracle("sample.264", benchCorpusEntry{
		ID:     "oracle",
		PixFmt: "yuv420p",
	}, benchOptions{
		ffmpegPixFmt: "yuv444p",
		strictPixFmt: true,
	})
	if err == nil || !strings.Contains(err.Error(), "manifest pixel format") {
		t.Fatalf("preflight err = %v, want strict pixel format mismatch", err)
	}
}
