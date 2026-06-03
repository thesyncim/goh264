// SPDX-License-Identifier: LGPL-2.1-or-later

package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
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

func TestFFmpegBenchLanesExposeFairCPUComparisons(t *testing.T) {
	lanes := ffmpegBenchLanes(benchOptions{runFFmpeg: true, fairCPULanes: true})
	if len(lanes) != 2 {
		t.Fatalf("lanes = %d, want 2", len(lanes))
	}
	if lanes[0].name != "ffmpeg-pure-c" || lanes[0].cpuFlags != "0" || lanes[0].comparisonLane != "pure-c-vs-pure-go" {
		t.Fatalf("pure-C lane = %+v, want explicit cpuflags 0 lane", lanes[0])
	}
	if lanes[1].name != "ffmpeg-native" || lanes[1].cpuFlags != "" || lanes[1].comparisonLane != "native-c+asm-vs-go+asm" {
		t.Fatalf("native lane = %+v, want native C+asm vs Go+asm lane", lanes[1])
	}

	lanes = ffmpegBenchLanes(benchOptions{runFFmpeg: true, ffmpegCPUFlags: "0"})
	if len(lanes) != 1 || lanes[0].name != "ffmpeg-pure-c" || lanes[0].backendKind != "ffmpeg-pure-c" {
		t.Fatalf("single pure-C lane = %+v, want pure-C", lanes)
	}
}

func TestFFmpegArgsIncludesCPUFlagsBeforeInput(t *testing.T) {
	args := ffmpegArgs("in.264", true, "1", "yuv420p", "0")
	got := strings.Join(args, " ")
	want := "-v error -nostdin -cpuflags 0 -threads 1 -i in.264 -an -sn -dn -pix_fmt yuv420p -f rawvideo -"
	if got != want {
		t.Fatalf("args = %q, want %q", got, want)
	}
}

func TestAnnotateFFmpegPeerQuality(t *testing.T) {
	ff := benchResult{RawOutput: true, RawMD5: "abc", BytesPerIter: 10}
	goResult := benchResult{RawOutput: true, RawMD5: "abc", BytesPerIter: 10}
	annotateFFmpegPeerQuality(&ff, goResult)
	if ff.PeerQualityStatus != "rawvideo-md5-match-goh264" || ff.PeerQualityMetric != "rawvideo-md5" || ff.PeerQualityReference != "goh264-rawvideo" {
		t.Fatalf("peer quality = %q/%q/%q, want Go rawvideo match",
			ff.PeerQualityStatus, ff.PeerQualityMetric, ff.PeerQualityReference)
	}
	if ff.ParityStatus != "rawvideo-md5-match-goh264" {
		t.Fatalf("fallback parity = %q, want peer match before an external oracle is attached", ff.ParityStatus)
	}

	ff.RawMD5 = "def"
	ff.ParityStatus = ""
	annotateFFmpegPeerQuality(&ff, goResult)
	if ff.PeerQualityStatus != "rawvideo-md5-mismatch-goh264" || ff.ParityStatus != "rawvideo-md5-mismatch-goh264" || ff.ErrorClass != "raw-md5-mismatch" {
		t.Fatalf("mismatch peer/parity/class = %q/%q/%q, want raw md5 mismatch",
			ff.PeerQualityStatus, ff.ParityStatus, ff.ErrorClass)
	}
}

func TestAnnotateBenchResultQuality(t *testing.T) {
	tests := []struct {
		name       string
		result     benchResult
		want       string
		wantRef    string
		wantMetric string
	}{
		{
			name: "manifest oracle",
			result: benchResult{
				RawOutput:      true,
				RawMD5:         "abc",
				ExpectedRawMD5: "abc",
				ParityStatus:   "rawvideo-md5-ok",
			},
			want:       "rawvideo-md5-ok",
			wantRef:    "manifest-rawvideo-oracle",
			wantMetric: "rawvideo-md5",
		},
		{
			name: "ffmpeg peer quality",
			result: benchResult{
				RawOutput:      true,
				RawMD5:         "abc",
				ParityStatus:   "rawvideo-md5-match-goh264",
				ComparisonLane: "native-c+asm-vs-go+asm",
			},
			want:       "rawvideo-md5-match-goh264",
			wantRef:    "goh264-rawvideo",
			wantMetric: "rawvideo-md5",
		},
		{
			name: "known red ledger",
			result: benchResult{
				RawOutput:      true,
				ExpectedRawMD5: "abc",
				ParityStatus:   "known-red",
			},
			want:       "known-red",
			wantRef:    "failure-ledger",
			wantMetric: "rawvideo-md5",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			annotateBenchResultQuality(&tt.result)
			if tt.result.QualityStatus != tt.want || tt.result.QualityReference != tt.wantRef || tt.result.QualityMetric != tt.wantMetric {
				t.Fatalf("quality = status %q metric %q ref %q, want %q/%q/%q",
					tt.result.QualityStatus, tt.result.QualityMetric, tt.result.QualityReference,
					tt.want, tt.wantMetric, tt.wantRef)
			}
		})
	}
}

func TestReadBenchCorpusManifestAndValidate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.jsonl")
	text := `
# comment
{"id":"ok","path":"sample.h264","format":"annexb","expect":"decode-ok","pix_fmt":"yuv420p","frame_count":2,"frame_size":16,"bitstream_md5":"00112233445566778899aabbccddeeff","rawvideo_md5":"ffeeddccbbaa99887766554433221100","frame_md5":["aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"],"surfaces":["annexb"]}

{"id":"url-ok","url":"https://example.invalid/sample.264","format":"annexb","expect":"decode-ok","pix_fmt":"yuv420p","frame_count":2,"frame_size":16,"bitstream_md5":"00112233445566778899aabbccddeeff","rawvideo_md5":"ffeeddccbbaa99887766554433221100","surfaces":["annexb"],"feature_tags":["external"],"source":"test"}

{"id":"extracted-ok","url":"https://example.invalid/sample.mp4","format":"annexb","expect":"decode-ok","pix_fmt":"yuv420p","frame_count":2,"frame_size":16,"source_md5":"11223344556677889900aabbccddeeff","bitstream_md5":"00112233445566778899aabbccddeeff","rawvideo_md5":"ffeeddccbbaa99887766554433221100","extract":"h264-annexb","surfaces":["annexb"],"feature_tags":["external","extracted-annexb"],"source":"test"}

{"id":"unsupported","path":"later.h264","format":"annexb","expect":"unsupported","guard_tags":["future"]}
`
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}

	entries, err := readBenchCorpusManifest(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 4 {
		t.Fatalf("entries = %d, want 4", len(entries))
	}
	if err := validateBenchCorpusEntry(entries[0]); err != nil {
		t.Fatalf("validate decode-ok: %v", err)
	}
	if err := validateBenchCorpusEntry(entries[1]); err != nil {
		t.Fatalf("validate url decode-ok: %v", err)
	}
	if err := validateBenchCorpusEntry(entries[2]); err != nil {
		t.Fatalf("validate extracted decode-ok: %v", err)
	}
	if err := validateBenchCorpusEntry(entries[3]); err == nil || !strings.Contains(err.Error(), "decode-ok") {
		t.Fatalf("validate unsupported err = %v, want decode-ok rejection", err)
	}
}

func TestReadBenchFailureLedgerAutoValidatesManifestSubset(t *testing.T) {
	dir := t.TempDir()
	row := `{"id":"fate/h264-conformance/frext-hcamff1-hhi","path":"HCAMFF1_HHI.264","url":"https://example.invalid/HCAMFF1_HHI.264","source":"FFmpeg FATE h264-conformance/FRext","format":"annexb","expect":"decode-ok","pix_fmt":"yuv420p","frame_count":10,"frame_size":152064,"bitstream_md5":"0dd0819dd9a276101a25259c0774c02c","rawvideo_md5":"2973f5376378cde879649160d4a46a98","surfaces":["annexb"],"feature_tags":["high","mbaff","field"]}`
	failureRow := `{"id":"fate/h264-conformance/frext-hcamff1-hhi","path":"HCAMFF1_HHI.264","url":"https://example.invalid/HCAMFF1_HHI.264","source":"FFmpeg FATE h264-conformance/FRext","format":"annexb","expect":"decode-ok","pix_fmt":"yuv420p","frame_count":10,"frame_size":152064,"bitstream_md5":"0dd0819dd9a276101a25259c0774c02c","rawvideo_md5":"2973f5376378cde879649160d4a46a98","surfaces":["annexb"],"feature_tags":["high","mbaff","field"],"known_failure":{"class":"decode-error","detail_contains":"unsupported bitstream feature"}}`
	manifestPath := filepath.Join(dir, "manifest.jsonl")
	failurePath := filepath.Join(dir, "failures.jsonl")
	if err := os.WriteFile(manifestPath, []byte(row+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(failurePath, []byte(failureRow+"\n"), 0o644); err != nil {
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

	missingRow := strings.Replace(failureRow, `"id":"fate/h264-conformance/frext-hcamff1-hhi"`, `"id":"fate/h264-conformance/missing-from-manifest"`, 1)
	if err := os.WriteFile(failurePath, []byte(missingRow+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := readBenchFailureLedger(manifestPath, "auto", entries); err == nil || !strings.Contains(err.Error(), "missing from") {
		t.Fatalf("missing ledger row err = %v, want manifest subset rejection", err)
	}

	driftedRow := strings.Replace(failureRow, `"source":"FFmpeg FATE h264-conformance/FRext"`, `"source":"drifted source"`, 1)
	if err := os.WriteFile(failurePath, []byte(driftedRow+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := readBenchFailureLedger(manifestPath, "auto", entries); err == nil || !strings.Contains(err.Error(), "drifted") {
		t.Fatalf("drifted ledger row err = %v, want manifest drift rejection", err)
	}
}

func TestBenchManifestReportsKnownRedRowsWithoutBenchmarking(t *testing.T) {
	dir := t.TempDir()
	row := `{"id":"known-red","path":"missing.264","source":"test public vectors","format":"annexb","expect":"decode-ok","pix_fmt":"yuv420p","frame_count":1,"frame_size":16,"bitstream_md5":"00112233445566778899aabbccddeeff","rawvideo_md5":"ffeeddccbbaa99887766554433221100","surfaces":["annexb"],"feature_tags":["unsupported"]}`
	failureRow := `{"id":"known-red","path":"missing.264","source":"test public vectors","format":"annexb","expect":"decode-ok","pix_fmt":"yuv420p","frame_count":1,"frame_size":16,"bitstream_md5":"00112233445566778899aabbccddeeff","rawvideo_md5":"ffeeddccbbaa99887766554433221100","surfaces":["annexb"],"feature_tags":["unsupported"],"known_failure":{"class":"input-missing","detail_contains":"missing.264"}}`
	manifestPath := filepath.Join(dir, "manifest.jsonl")
	failurePath := filepath.Join(dir, "failures.jsonl")
	if err := os.WriteFile(manifestPath, []byte(row+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(failurePath, []byte(failureRow+"\n"), 0o644); err != nil {
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
	if report.Metadata.CorpusSelected != 1 || report.Metadata.CorpusDecodeOK != 1 || report.Metadata.CorpusGreen != 0 ||
		report.Metadata.CorpusSkipped != 1 || report.Metadata.CorpusNotTimed != 1 {
		t.Fatalf("metadata counts = %+v, want selected/decode-ok/skipped known-red row", report.Metadata)
	}
	if len(report.Results) != 1 || !report.Results[0].Skipped || report.Results[0].ParityStatus != "known-red" {
		t.Fatalf("result = %+v, want visible known-red skipped row", report.Results)
	}
	if report.Results[0].ErrorClass != "input-missing" {
		t.Fatalf("known-red error class = %q, want input-missing", report.Results[0].ErrorClass)
	}
	if got := strings.Join(report.Results[0].FeatureTags, ","); got != "unsupported" {
		t.Fatalf("known-red feature tags = %q, want unsupported", got)
	}
	if got := strings.Join(report.Results[0].Surfaces, ","); got != "annexb" {
		t.Fatalf("known-red surfaces = %q, want annexb", got)
	}
	if report.Results[0].Source != "test public vectors" {
		t.Fatalf("known-red source = %q, want test public vectors", report.Results[0].Source)
	}
	if !strings.HasSuffix(report.Results[0].Input, "missing.264") {
		t.Fatalf("known-red input = %q, want missing.264 path", report.Results[0].Input)
	}
	if report.Results[0].Error == "" || !strings.Contains(report.Results[0].Error, "missing.264") {
		t.Fatalf("known-red error = %q, want missing input detail", report.Results[0].Error)
	}
	if notes := strings.Join(report.Results[0].Notes, "\n"); !strings.Contains(notes, `expected current failure: class=input-missing contains="missing.264"`) {
		t.Fatalf("known-red notes = %q, want expected failure signature", notes)
	}
}

func TestBenchManifestDiagnoseReportsKnownRedRows(t *testing.T) {
	dir := t.TempDir()
	row := `{"id":"known-red","path":"missing.264","source":"test public vectors","format":"annexb","expect":"decode-ok","pix_fmt":"yuv420p","frame_count":1,"frame_size":16,"bitstream_md5":"00112233445566778899aabbccddeeff","rawvideo_md5":"ffeeddccbbaa99887766554433221100","surfaces":["annexb"],"feature_tags":["unsupported"]}`
	failureRow := `{"id":"known-red","path":"missing.264","source":"test public vectors","format":"annexb","expect":"decode-ok","pix_fmt":"yuv420p","frame_count":1,"frame_size":16,"bitstream_md5":"00112233445566778899aabbccddeeff","rawvideo_md5":"ffeeddccbbaa99887766554433221100","surfaces":["annexb"],"feature_tags":["unsupported"],"known_failure":{"class":"input-missing","detail_contains":"missing.264"}}`
	manifestPath := filepath.Join(dir, "manifest.jsonl")
	failurePath := filepath.Join(dir, "failures.jsonl")
	if err := os.WriteFile(manifestPath, []byte(row+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(failurePath, []byte(failureRow+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	report, err := benchManifest(manifestPath, 0, benchOptions{
		iters:         1,
		repeats:       1,
		rawOutput:     true,
		failureLedger: "auto",
		diagnose:      true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if report.Metadata.ComparisonKind != "manifest-goh264-oracle-diagnostic" || report.Metadata.CorpusKnownRed != 1 || report.Metadata.CorpusBench != 0 {
		t.Fatalf("metadata = %+v, want diagnostic known-red without benchmarks", report.Metadata)
	}
	if report.Metadata.CorpusSelected != 1 || report.Metadata.CorpusDecodeOK != 1 || report.Metadata.CorpusGreen != 0 ||
		report.Metadata.CorpusSkipped != 1 || report.Metadata.CorpusNotTimed != 1 {
		t.Fatalf("diagnostic metadata counts = %+v, want visible known-red skip", report.Metadata)
	}
	if len(report.Results) != 1 || !report.Results[0].Skipped || report.Results[0].BaselineKind != "oracle-known-red-diagnostic" || report.Results[0].ParityStatus != "known-red" {
		t.Fatalf("result = %+v, want known-red diagnostic row", report.Results)
	}
	if report.Results[0].ErrorClass != "input-missing" || !strings.Contains(report.Results[0].Error, "missing.264") {
		t.Fatalf("diagnostic error = class %q detail %q, want missing input", report.Results[0].ErrorClass, report.Results[0].Error)
	}
}

func TestKnownRedBenchResultMarksSignatureDrift(t *testing.T) {
	entry := benchCorpusEntry{
		ID:          "known-red",
		Path:        "sample.264",
		Expect:      "decode-ok",
		PixFmt:      "yuv420p",
		FrameCount:  1,
		FrameSize:   16,
		RawVideoMD5: "00112233445566778899aabbccddeeff",
		Surfaces:    []string{"annexb"},
		FeatureTags: []string{"mbaff"},
		KnownFailure: &benchKnownFailure{
			Class:          "decode-error",
			DetailContains: "unsupported bitstream feature",
		},
	}
	result := knownRedBenchResult(entry, "sample.264", []byte{1, 2, 3}, errors.New("decode: h264: invalid data"), "failures.jsonl")
	if result.ParityStatus != "known-red-signature-drift" {
		t.Fatalf("parity status = %q, want known-red-signature-drift", result.ParityStatus)
	}
	if result.ErrorClass != "decode-error" {
		t.Fatalf("error class = %q, want decode-error", result.ErrorClass)
	}
	if notes := strings.Join(result.Notes, "\n"); !strings.Contains(notes, "current failure signature drifted") {
		t.Fatalf("notes = %q, want signature drift note", notes)
	}
}

func TestApplyKnownRedDiagnosticMarksStaleLedgerWithoutGreenwashing(t *testing.T) {
	result := benchResult{
		ParityStatus: "rawvideo-md5-ok",
		ErrorClass:   "",
	}
	failure := benchCorpusEntry{
		ID: "known-red",
		KnownFailure: &benchKnownFailure{
			Class:          "raw-md5-mismatch",
			DetailContains: "old",
		},
	}
	applyKnownRedDiagnostic(&result, failure, "failures.jsonl")
	if !result.Skipped || result.ParityStatus != "rawvideo-md5-ok-failure-ledger-stale" {
		t.Fatalf("known-red stale diagnostic = %+v, want stale ledger status", result)
	}
	if notes := strings.Join(result.Notes, "\n"); !strings.Contains(notes, "passed Go oracle diagnostics") {
		t.Fatalf("notes = %q, want stale ledger note", notes)
	}
}

func TestBenchManifestSkipsStaleKnownRedRowsUntilLedgerUpdates(t *testing.T) {
	dir := t.TempDir()
	entry := writeBenchFixtureEntry(t, dir, "stale-red", "stale.264")
	failure := entry
	failure.KnownFailure = &benchKnownFailure{
		Class:          "raw-md5-mismatch",
		DetailContains: "old-signature",
	}
	manifestPath := filepath.Join(dir, "manifest.jsonl")
	failurePath := filepath.Join(dir, "failures.jsonl")
	writeBenchManifestRows(t, manifestPath, entry)
	writeBenchManifestRows(t, failurePath, failure)

	report, err := benchManifest(manifestPath, 0, benchOptions{
		iters:         1,
		repeats:       1,
		rawOutput:     true,
		failureLedger: "auto",
	})
	if err != nil {
		t.Fatal(err)
	}
	if report.Metadata.CorpusBench != 0 || report.Metadata.CorpusGreen != 0 || report.Metadata.CorpusKnownRed != 0 ||
		report.Metadata.CorpusStaleRed != 1 || report.Metadata.CorpusSkipped != 1 || report.Metadata.CorpusNotTimed != 1 {
		t.Fatalf("metadata = %+v, want stale known-red skipped without timing", report.Metadata)
	}
	if len(report.Results) != 1 || !report.Results[0].Skipped ||
		report.Results[0].BaselineKind != "oracle-known-red-stale" ||
		report.Results[0].ParityStatus != "rawvideo-md5-ok-failure-ledger-stale" {
		t.Fatalf("result = %+v, want stale known-red skip", report.Results)
	}
	if report.Results[0].RawMD5 != entry.RawVideoMD5 {
		t.Fatalf("stale result raw md5 = %q, want oracle %q", report.Results[0].RawMD5, entry.RawVideoMD5)
	}
}

func TestBenchManifestMaxEntriesReportsGreenRowsNotTimed(t *testing.T) {
	dir := t.TempDir()
	entryA := writeBenchFixtureEntry(t, dir, "green-a", "a.264")
	entryB := writeBenchFixtureEntry(t, dir, "green-b", "b.264")
	manifestPath := filepath.Join(dir, "manifest.jsonl")
	writeBenchManifestRows(t, manifestPath, entryA, entryB)

	report, err := benchManifest(manifestPath, 1, benchOptions{
		iters:         1,
		repeats:       1,
		rawOutput:     true,
		failureLedger: "off",
	})
	if err != nil {
		t.Fatal(err)
	}
	if report.Metadata.CorpusSelected != 2 || report.Metadata.CorpusDecodeOK != 2 ||
		report.Metadata.CorpusGreen != 2 || report.Metadata.CorpusBench != 1 ||
		report.Metadata.CorpusSkipped != 1 || report.Metadata.CorpusNotTimed != 1 {
		t.Fatalf("metadata = %+v, want one timed green row and one visible not-timed row", report.Metadata)
	}
	if len(report.Results) != 2 {
		t.Fatalf("results = %d, want timed plus not-timed row", len(report.Results))
	}
	if report.Results[0].Skipped || report.Results[0].ParityStatus != "rawvideo-md5-ok" {
		t.Fatalf("result[0] = %+v, want timed green", report.Results[0])
	}
	if !report.Results[1].Skipped || report.Results[1].BaselineKind != "oracle-green-not-timed" ||
		report.Results[1].ParityStatus != "rawvideo-md5-ok-not-timed" {
		t.Fatalf("result[1] = %+v, want visible green not-timed row", report.Results[1])
	}
}

func TestAnnotateBenchFrameDiagnostics(t *testing.T) {
	result := benchResult{
		FrameDiagnostics: []benchFrameDiagnostic{
			{Index: 0, RawPixelFormat: "yuv420p", Bytes: 16, RawMD5: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
			{Index: 1, RawPixelFormat: "yuv420p", Bytes: 16, RawMD5: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"},
			{Index: 2, RawPixelFormat: "yuv420p", Bytes: 16, RawMD5: "cccccccccccccccccccccccccccccccc"},
		},
	}
	annotateBenchFrameDiagnostics(&result, benchCorpusEntry{
		ID:         "oracle",
		PixFmt:     "yuv420p",
		FrameCount: 4,
		FrameSize:  16,
		FrameMD5: []string{
			"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			"00000000000000000000000000000000",
			"cccccccccccccccccccccccccccccccc",
			"dddddddddddddddddddddddddddddddd",
		},
	})
	if len(result.FrameDiagnostics) != 4 {
		t.Fatalf("frame diagnostics = %d, want 4 including missing expected frame", len(result.FrameDiagnostics))
	}
	if result.FrameDiagnostics[0].ParityStatus != "raw-md5-ok" {
		t.Fatalf("frame[0] parity = %q, want raw-md5-ok", result.FrameDiagnostics[0].ParityStatus)
	}
	if result.FrameDiagnostics[1].ParityStatus != "raw-md5-mismatch" || result.FrameDiagnostics[1].ExpectedRawMD5 == "" {
		t.Fatalf("frame[1] = %+v, want md5 mismatch with expected hash", result.FrameDiagnostics[1])
	}
	if result.FrameDiagnostics[3].ParityStatus != "missing" || result.FrameDiagnostics[3].ExpectedRawMD5 == "" {
		t.Fatalf("frame[3] = %+v, want missing expected frame", result.FrameDiagnostics[3])
	}
}

func writeBenchFixtureEntry(t *testing.T, dir string, id string, name string) benchCorpusEntry {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "h264", "high10_inter_cavlc_idrp.h264"))
	if err != nil {
		t.Fatal(err)
	}
	run, err := decodeGoOnceForFormat(data, true, true)
	if err != nil {
		t.Fatalf("decode fixture: %v", err)
	}
	if run.frames <= 0 || run.bytes <= 0 || run.bytes%int64(run.frames) != 0 {
		t.Fatalf("fixture summary frames/bytes = %d/%d, want stable frame size", run.frames, run.bytes)
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	sum := md5.Sum(data)
	return benchCorpusEntry{
		ID:           id,
		Path:         name,
		Source:       "local benchmark fixture",
		Format:       "annexb",
		Expect:       "decode-ok",
		PixFmt:       run.pixFmt,
		FrameCount:   run.frames,
		FrameSize:    int(run.bytes / int64(run.frames)),
		BitstreamMD5: hex.EncodeToString(sum[:]),
		RawVideoMD5:  run.md5,
		Surfaces:     []string{"annexb"},
		FeatureTags:  []string{"fixture"},
	}
}

func writeBenchManifestRows(t *testing.T, path string, entries ...benchCorpusEntry) {
	t.Helper()
	var text strings.Builder
	for _, entry := range entries {
		row, err := json.Marshal(entry)
		if err != nil {
			t.Fatal(err)
		}
		text.Write(row)
		text.WriteByte('\n')
	}
	if err := os.WriteFile(path, []byte(text.String()), 0o644); err != nil {
		t.Fatal(err)
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
	if got := strings.Join(report.Results[0].Surfaces, ","); got != "annexb" {
		t.Fatalf("skipped surfaces = %q, want annexb", got)
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
		Surfaces:    []string{"annexb"},
		FeatureTags: []string{"cabac", "weighted"},
		Source:      "FFmpeg FATE h264-conformance",
	}
	result := benchResult{
		Name:                 "goh264",
		FramesPerIter:        2,
		BytesPerIter:         32,
		RawPixelFormat:       "yuv420p",
		RawMD5:               "ffeeddccbbaa99887766554433221100",
		PeerQualityStatus:    "rawvideo-md5-match-goh264",
		PeerQualityMetric:    "rawvideo-md5",
		PeerQualityReference: "goh264-rawvideo",
	}
	if err := annotateBenchResultWithOracle(&result, entry); err != nil {
		t.Fatalf("annotate oracle: %v", err)
	}
	if result.EntryID != entry.ID || result.ExpectedBytes != 32 || result.ParityStatus != "rawvideo-md5-ok" {
		t.Fatalf("annotated result = %+v", result)
	}
	if result.PeerQualityStatus != "rawvideo-md5-match-goh264" || result.PeerQualityReference != "goh264-rawvideo" {
		t.Fatalf("peer quality after oracle annotation = %q/%q, want preserved",
			result.PeerQualityStatus, result.PeerQualityReference)
	}
	if result.Source != entry.Source || len(result.FeatureTags) != len(entry.FeatureTags) {
		t.Fatalf("annotated metadata = source %q tags %v, want %q/%v", result.Source, result.FeatureTags, entry.Source, entry.FeatureTags)
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
	}, ffmpegBenchLane{name: "ffmpeg-native", backendKind: "ffmpeg-native-c+asm"})
	if err == nil || !strings.Contains(err.Error(), "manifest pixel format") {
		t.Fatalf("preflight err = %v, want strict pixel format mismatch", err)
	}
}

func TestBenchOracleFailureClass(t *testing.T) {
	tests := map[string]string{
		"missing /tmp/in.264; set GOH264_CORPUS_FETCH=1": "input-missing",
		"decode: unsupported MBAFF":                      "decode-error",
		"frames_per_iter = 2, want 3":                    "frame-count-mismatch",
		"Go raw_pixel_format = yuv420p, want yuv422p":    "pixel-format-mismatch",
		"bytes_per_iter = 10, want 20":                   "raw-size-mismatch",
		"source_md5 = abc, want def":                     "source-md5-mismatch",
		"BITSTREAM_MD5 = abc, want def":                  "bitstream-md5-mismatch",
		"raw_md5 = abc, want def":                        "raw-md5-mismatch",
		"unexpected oracle detail":                       "oracle-mismatch",
	}
	for detail, want := range tests {
		if got := benchOracleFailureClass(detail); got != want {
			t.Fatalf("benchOracleFailureClass(%q) = %q, want %q", detail, got, want)
		}
	}
}
