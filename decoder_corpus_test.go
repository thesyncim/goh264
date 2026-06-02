// SPDX-License-Identifier: LGPL-2.1-or-later

package goh264

import (
	"bufio"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

const defaultH264CorpusManifest = "testdata/h264/corpus/manifest.jsonl"
const defaultH264RealVectorManifest = "testdata/h264/realvectors/manifest.jsonl"
const defaultH264RealVectorFailureManifest = "testdata/h264/realvectors/failures.jsonl"

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
	URL           string   `json:"url,omitempty"`
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
	FeatureTags   []string `json:"feature_tags,omitempty"`
	Source        string   `json:"source,omitempty"`
}

func TestH264CorpusManifest(t *testing.T) {
	for _, manifest := range h264CorpusManifestPaths() {
		manifest := manifest
		t.Run(filepath.Base(manifest), func(t *testing.T) {
			testH264CorpusManifest(t, manifest)
		})
	}
}

func TestH264RealVectorManifest(t *testing.T) {
	if !h264RealVectorsEnabled() {
		t.Skip("set GOH264_REAL_VECTORS=1 or GOH264_ORACLE=1 to run external public H.264 vectors")
	}
	testH264CorpusManifest(t, defaultH264RealVectorManifest)
}

func TestH264RealVectorFailureLedgerIntegrity(t *testing.T) {
	manifest := readH264CorpusManifest(t, defaultH264RealVectorManifest)
	failures := readH264CorpusManifest(t, defaultH264RealVectorFailureManifest)
	if len(manifest) == 0 {
		t.Fatal("real-vector manifest is empty")
	}
	if len(failures) == 0 {
		t.Fatal("real-vector failure ledger is empty; delete this test only when the public vector lane is fully green")
	}

	byID := make(map[string]h264CorpusEntry, len(manifest))
	for _, entry := range manifest {
		validateH264CorpusEntry(t, entry)
		if entry.URL == "" || entry.Source == "" || len(entry.FeatureTags) == 0 {
			t.Fatalf("%s: real-vector rows need url, source, and feature_tags", entry.ID)
		}
		if previous, ok := byID[entry.ID]; ok {
			t.Fatalf("%s: duplicate real-vector id: previous=%+v current=%+v", entry.ID, previous, entry)
		}
		byID[entry.ID] = entry
	}

	failedIDs := make(map[string]struct{}, len(failures))
	for _, failure := range failures {
		validateH264CorpusEntry(t, failure)
		if failure.Expect != "decode-ok" {
			t.Fatalf("%s: failure ledger rows must stay decode-ok oracle rows, got %q", failure.ID, failure.Expect)
		}
		if _, ok := failedIDs[failure.ID]; ok {
			t.Fatalf("%s: duplicate failure-ledger id", failure.ID)
		}
		failedIDs[failure.ID] = struct{}{}
		manifestEntry, ok := byID[failure.ID]
		if !ok {
			t.Fatalf("%s: failure-ledger row is missing from %s", failure.ID, defaultH264RealVectorManifest)
		}
		if !reflect.DeepEqual(failure, manifestEntry) {
			t.Fatalf("%s: failure-ledger row drifted from real-vector manifest\nfailure=%+v\nmanifest=%+v", failure.ID, failure, manifestEntry)
		}
	}

	var greenCanaries []string
	for _, entry := range manifest {
		if _, failing := failedIDs[entry.ID]; !failing {
			greenCanaries = append(greenCanaries, entry.ID)
		}
	}
	if len(greenCanaries) == 0 {
		t.Fatal("real-vector manifest has no green canary outside failures.jsonl")
	}
}

func TestH264RealVectorFailureFocusedFilters(t *testing.T) {
	manifest := readH264CorpusManifest(t, defaultH264RealVectorManifest)
	failures := readH264CorpusManifest(t, defaultH264RealVectorFailureManifest)
	manifestByID := make(map[string]h264CorpusEntry, len(manifest))
	for _, entry := range manifest {
		manifestByID[entry.ID] = entry
	}

	focusTokens := []string{"mbaff", "paff", "high", "chroma", "b-slice", "direct", "weighted"}
	var applicable int
	for _, token := range focusTokens {
		token := token
		filteredFailures := filterH264CorpusEntries(append([]h264CorpusEntry(nil), failures...), []string{token})
		if len(filteredFailures) == 0 {
			continue
		}
		applicable++
		t.Run(token, func(t *testing.T) {
			filteredManifest := filterH264CorpusEntries(append([]h264CorpusEntry(nil), manifest...), []string{token})
			filteredManifestByID := make(map[string]h264CorpusEntry, len(filteredManifest))
			for _, entry := range filteredManifest {
				filteredManifestByID[entry.ID] = entry
			}
			for _, failure := range filteredFailures {
				manifestEntry, ok := filteredManifestByID[failure.ID]
				if !ok {
					t.Fatalf("%s: known-red row matched filter %q but disappeared from filtered manifest", failure.ID, token)
				}
				if !reflect.DeepEqual(failure, manifestEntry) {
					t.Fatalf("%s: filtered known-red row drifted from real-vector manifest\nfailure=%+v\nmanifest=%+v", failure.ID, failure, manifestEntry)
				}
				if _, ok := manifestByID[failure.ID]; !ok {
					t.Fatalf("%s: known-red row missing from unfiltered manifest", failure.ID)
				}
			}
			t.Logf("%s: known-red ids: %s", token, strings.Join(h264CorpusEntryIDs(filteredFailures), ","))
		})
	}
	if applicable == 0 {
		t.Fatalf("no known-red rows matched focused filters %v; failure ledger tags are %s", focusTokens, h264CorpusFailureFilterSummary(failures))
	}
}

func TestH264RealVectorFailureLedgerFreshness(t *testing.T) {
	if !h264RealVectorsEnabled() && os.Getenv("GOH264_REAL_VECTOR_FAILURES") != "1" {
		t.Skip("set GOH264_REAL_VECTOR_FAILURES=1, GOH264_REAL_VECTORS=1, or GOH264_ORACLE=1 to verify red public vector rows")
	}
	failures := readH264CorpusManifest(t, defaultH264RealVectorFailureManifest)
	if filter := h264CorpusFilterTokens(); len(filter) != 0 {
		failures = filterH264CorpusEntries(failures, filter)
		if len(failures) == 0 {
			t.Fatalf("%s: no failure entries matched GOH264_CORPUS_FILTER=%q; available known-red filters: %s",
				defaultH264RealVectorFailureManifest, os.Getenv("GOH264_CORPUS_FILTER"), h264CorpusFailureFilterSummary(readH264CorpusManifest(t, defaultH264RealVectorFailureManifest)))
		}
	}
	for _, entry := range failures {
		entry := entry
		t.Run(entry.ID, func(t *testing.T) {
			validateH264CorpusEntry(t, entry)
			if entry.Expect != "decode-ok" {
				t.Fatalf("%s: failure ledger rows must stay decode-ok oracle rows, got %q", entry.ID, entry.Expect)
			}
			if !h264CorpusEntryHasSurface(entry, "annexb") {
				t.Fatalf("%s: failure-ledger freshness currently requires an annexb surface", entry.ID)
			}
			path := materializeH264CorpusEntry(t, defaultH264RealVectorFailureManifest, entry)
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}
			assertCorpusBitstreamMD5(t, entry, data)
			matches, detail := h264CorpusAnnexBMatchesOracle(t, entry, data)
			if matches {
				t.Fatalf("%s: failure-ledger row now matches oracle; remove it from %s", entry.ID, defaultH264RealVectorFailureManifest)
			}
			t.Logf("%s: still red: class=%s features=%s surfaces=%s source=%q detail=%s",
				entry.ID,
				h264CorpusOracleFailureClass(detail),
				strings.Join(entry.FeatureTags, ","),
				strings.Join(entry.Surfaces, ","),
				entry.Source,
				detail)
		})
	}
}

func h264RealVectorsEnabled() bool {
	return os.Getenv("GOH264_REAL_VECTORS") == "1" || os.Getenv("GOH264_ORACLE") == "1"
}

func testH264CorpusManifest(t *testing.T, manifest string) {
	entries := readH264CorpusManifest(t, manifest)
	if len(entries) == 0 {
		t.Fatalf("%s: no corpus entries", manifest)
	}
	if filter := h264CorpusFilterTokens(); len(filter) != 0 {
		entries = filterH264CorpusEntries(entries, filter)
		if len(entries) == 0 {
			t.Fatalf("%s: no corpus entries matched GOH264_CORPUS_FILTER=%q", manifest, os.Getenv("GOH264_CORPUS_FILTER"))
		}
	}

	for _, entry := range entries {
		entry := entry
		t.Run(entry.ID, func(t *testing.T) {
			validateH264CorpusEntry(t, entry)
			path := materializeH264CorpusEntry(t, manifest, entry)
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

func h264CorpusEntryHasSurface(entry h264CorpusEntry, want string) bool {
	for _, surface := range entry.Surfaces {
		if surface == want {
			return true
		}
	}
	return false
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

func TestH264CorpusFilter(t *testing.T) {
	entries := []h264CorpusEntry{
		{
			ID:          "fate/h264-conformance/caba3-sva-b",
			Path:        "CABA3_SVA_B.264",
			Source:      "FFmpeg FATE h264-conformance",
			Expect:      "decode-ok",
			PixFmt:      "yuv420p",
			FeatureTags: []string{"cabac", "main", "temporal-direct", "deblock"},
			Surfaces:    []string{"annexb"},
		},
		{
			ID:          "fate/h264-conformance/cvwp3-toshiba-e",
			Path:        "CVWP3_TOSHIBA_E.264",
			Source:      "FFmpeg FATE h264-conformance",
			Expect:      "decode-ok",
			PixFmt:      "yuv420p",
			FeatureTags: []string{"cabac", "implicit-weight-b", "weighted-bipred"},
			Surfaces:    []string{"annexb"},
		},
	}

	filtered := filterH264CorpusEntries(entries, []string{"cabac", "temporal"})
	if len(filtered) != 1 || filtered[0].ID != "fate/h264-conformance/caba3-sva-b" {
		t.Fatalf("filtered entries = %+v, want caba3 only", filtered)
	}

	filtered = filterH264CorpusEntries(entries, []string{"weighted"})
	if len(filtered) != 1 || filtered[0].ID != "fate/h264-conformance/cvwp3-toshiba-e" {
		t.Fatalf("filtered entries = %+v, want cvwp3 only", filtered)
	}
}

func h264CorpusFilterTokens() []string {
	filter := os.Getenv("GOH264_CORPUS_FILTER")
	if filter == "" {
		return nil
	}
	return strings.FieldsFunc(strings.ToLower(filter), func(r rune) bool {
		switch r {
		case ',', ';', ' ', '\t', '\n', '\r':
			return true
		default:
			return false
		}
	})
}

func filterH264CorpusEntries(entries []h264CorpusEntry, tokens []string) []h264CorpusEntry {
	filtered := entries[:0]
	for _, entry := range entries {
		if h264CorpusEntryMatches(entry, tokens) {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

func h264CorpusEntryMatches(entry h264CorpusEntry, tokens []string) bool {
	haystack := strings.ToLower(strings.Join(h264CorpusEntrySearchFields(entry), "\x00"))
	for _, token := range tokens {
		if token != "" && !strings.Contains(haystack, token) {
			return false
		}
	}
	return true
}

func h264CorpusEntrySearchFields(entry h264CorpusEntry) []string {
	fields := []string{
		entry.ID,
		entry.Path,
		entry.URL,
		entry.Format,
		entry.Expect,
		entry.ExpectedError,
		entry.PixFmt,
		entry.Source,
	}
	fields = append(fields, entry.Surfaces...)
	fields = append(fields, entry.GuardTags...)
	fields = append(fields, entry.FeatureTags...)
	return fields
}

func h264CorpusEntryIDs(entries []h264CorpusEntry) []string {
	ids := make([]string, 0, len(entries))
	for _, entry := range entries {
		ids = append(ids, entry.ID)
	}
	return ids
}

func h264CorpusFailureFilterSummary(entries []h264CorpusEntry) string {
	values := make(map[string]struct{})
	for _, entry := range entries {
		fields := []string{entry.ID, entry.Path, entry.PixFmt}
		fields = append(fields, entry.Surfaces...)
		fields = append(fields, entry.GuardTags...)
		fields = append(fields, entry.FeatureTags...)
		for _, value := range fields {
			if value != "" {
				values[strings.ToLower(value)] = struct{}{}
			}
		}
	}
	var sorted []string
	for value := range values {
		sorted = append(sorted, value)
	}
	sort.Strings(sorted)
	return strings.Join(sorted, ",")
}

func TestValidateH264CorpusEntryAllowsURLBackedDecodeOK(t *testing.T) {
	validateH264CorpusEntry(t, h264CorpusEntry{
		ID:           "external",
		URL:          "https://example.invalid/sample.264",
		Format:       "annexb",
		Expect:       "decode-ok",
		PixFmt:       "yuv420p",
		FrameCount:   2,
		FrameSize:    16,
		BitstreamMD5: "00112233445566778899aabbccddeeff",
		RawVideoMD5:  "ffeeddccbbaa99887766554433221100",
		Surfaces:     []string{"annexb"},
		FeatureTags:  []string{"external"},
		Source:       "test",
	})
}

func TestValidateH264CorpusEntryRequiresUnsupportedGuards(t *testing.T) {
	validateH264CorpusEntry(t, h264CorpusEntry{
		ID:            "future",
		Path:          "future.264",
		Format:        "annexb",
		Expect:        "unsupported",
		ExpectedError: "ErrUnsupported",
		Surfaces:      []string{"annexb"},
		GuardTags:     []string{"mbaff"},
	})
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
	if entry.ID == "" || entry.Path == "" && entry.URL == "" {
		t.Fatalf("entry id and path or url must be set: %+v", entry)
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
		if entry.FrameCount <= 0 || entry.FrameSize <= 0 {
			t.Fatalf("%s: frame_count/frame_size must be positive", entry.ID)
		}
		if len(entry.FrameMD5) != 0 && len(entry.FrameMD5) != entry.FrameCount {
			t.Fatalf("%s: frame_md5 count = %d, want 0 or %d", entry.ID, len(entry.FrameMD5), entry.FrameCount)
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

func materializeH264CorpusEntry(t *testing.T, manifest string, entry h264CorpusEntry) string {
	t.Helper()
	baseDir := filepath.Dir(manifest)
	if entry.Path != "" {
		path := entry.Path
		if !filepath.IsAbs(path) {
			path = filepath.Join(baseDir, path)
		}
		if _, err := os.Stat(path); err == nil {
			return path
		}
		if entry.URL == "" {
			return path
		}
	}
	if entry.URL == "" {
		t.Fatalf("%s: no path or url", entry.ID)
	}
	rel := entry.Path
	if rel == "" {
		rel = filepath.Base(entry.URL)
	}
	rel = cleanRelativeH264CorpusPath(t, entry.ID, rel)
	cacheRoot := os.Getenv("GOH264_CORPUS_CACHE")
	if cacheRoot == "" {
		cacheRoot = filepath.Join(baseDir, "cache")
	}
	path := filepath.Join(cacheRoot, rel)
	if _, err := os.Stat(path); err == nil {
		return path
	}
	if os.Getenv("GOH264_CORPUS_FETCH") != "1" {
		t.Fatalf("%s: missing %s; set GOH264_CORPUS_FETCH=1 to download %s", entry.ID, path, entry.URL)
	}
	downloadH264CorpusEntry(t, entry, path)
	return path
}

func cleanRelativeH264CorpusPath(t *testing.T, id string, path string) string {
	t.Helper()
	clean := filepath.Clean(path)
	if clean == "." || filepath.IsAbs(clean) || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || clean == ".." {
		t.Fatalf("%s: unsafe corpus path %q", id, path)
	}
	return clean
}

func downloadH264CorpusEntry(t *testing.T, entry h264CorpusEntry, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("%s: create corpus cache dir: %v", entry.ID, err)
	}
	resp, err := http.Get(entry.URL)
	if err != nil {
		t.Fatalf("%s: download %s: %v", entry.ID, entry.URL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("%s: download %s: status %s", entry.ID, entry.URL, resp.Status)
	}
	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		t.Fatalf("%s: create %s: %v", entry.ID, tmp, err)
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		os.Remove(tmp)
		t.Fatalf("%s: write %s: %v", entry.ID, tmp, err)
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		t.Fatalf("%s: close %s: %v", entry.ID, tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		t.Fatalf("%s: install %s: %v", entry.ID, path, err)
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
		if len(entry.FrameMD5) != 0 {
			if got := hex.EncodeToString(sum[:]); got != entry.FrameMD5[i] {
				t.Fatalf("%s frame[%d] md5 = %s, want %s", entry.ID, i, got, entry.FrameMD5[i])
			}
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

func h264CorpusAnnexBMatchesOracle(t *testing.T, entry h264CorpusEntry, data []byte) (bool, string) {
	t.Helper()
	frames, err := NewDecoder().DecodeAnnexBFrames(data)
	if err != nil {
		return false, fmt.Sprintf("decode error: %v", err)
	}
	if len(frames) != entry.FrameCount {
		return false, fmt.Sprintf("frames = %d, want %d", len(frames), entry.FrameCount)
	}
	rawHash := md5.New()
	var total int
	for i, frame := range frames {
		pixFmt, err := frame.RawPixelFormat()
		if err != nil {
			return false, fmt.Sprintf("frame[%d] pix_fmt: %v", i, err)
		}
		if pixFmt != entry.PixFmt {
			return false, fmt.Sprintf("frame[%d] pix_fmt = %s, want %s", i, pixFmt, entry.PixFmt)
		}
		raw, err := frame.AppendRawYUVBytesLE(nil)
		if err != nil {
			return false, fmt.Sprintf("frame[%d] raw yuv: %v", i, err)
		}
		if len(raw) != entry.FrameSize {
			return false, fmt.Sprintf("frame[%d] raw size = %d, want %d", i, len(raw), entry.FrameSize)
		}
		sum := md5.Sum(raw)
		if len(entry.FrameMD5) != 0 {
			if got := hex.EncodeToString(sum[:]); got != entry.FrameMD5[i] {
				return false, fmt.Sprintf("frame[%d] md5 = %s, want %s", i, got, entry.FrameMD5[i])
			}
		}
		if _, err := rawHash.Write(raw); err != nil {
			return false, fmt.Sprintf("frame[%d] raw hash: %v", i, err)
		}
		total += len(raw)
	}
	if total != entry.FrameCount*entry.FrameSize {
		return false, fmt.Sprintf("raw total = %d, want %d", total, entry.FrameCount*entry.FrameSize)
	}
	if got := hex.EncodeToString(rawHash.Sum(nil)); got != entry.RawVideoMD5 {
		return false, fmt.Sprintf("rawvideo md5 = %s, want %s", got, entry.RawVideoMD5)
	}
	return true, "matched rawvideo oracle"
}

func h264CorpusOracleFailureClass(detail string) string {
	switch {
	case detail == "":
		return ""
	case strings.Contains(detail, "decode error") || strings.Contains(detail, "unsupported"):
		return "decode-error"
	case strings.Contains(detail, "frames ="):
		return "frame-count-mismatch"
	case strings.Contains(detail, "pix_fmt"):
		return "pixel-format-mismatch"
	case strings.Contains(detail, "raw size") || strings.Contains(detail, "raw total"):
		return "raw-size-mismatch"
	case strings.Contains(detail, "rawvideo md5") || strings.Contains(detail, "md5 ="):
		return "raw-md5-mismatch"
	default:
		return "oracle-mismatch"
	}
}

func assertH264CorpusUnsupported(t *testing.T, entry h264CorpusEntry, err error) {
	t.Helper()
	if !errors.Is(err, ErrUnsupported) {
		t.Fatalf("%s: err = %v, want ErrUnsupported for guard tags %v", entry.ID, err, entry.GuardTags)
	}
}
