// SPDX-License-Identifier: LGPL-2.1-or-later

package goh264_test

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
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

const defaultH264CorpusManifest = "testdata/h264/corpus/manifest.jsonl"
const defaultH264RealVectorManifest = "testdata/h264/realvectors/manifest.jsonl"
const defaultH264RealVectorFailureManifest = "testdata/h264/realvectors/failures.jsonl"
const defaultH264RealVectorExclusionManifest = "testdata/h264/realvectors/exclusions.jsonl"
const defaultH264RealVectorUpstreamInventory = "testdata/h264/realvectors/upstream-inventory.jsonl"
const h264RealVectorFFmpegFATEInventorySource = "FFmpeg FATE n8.0.1"

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
	ID            string                  `json:"id"`
	Path          string                  `json:"path"`
	URL           string                  `json:"url,omitempty"`
	Format        string                  `json:"format"`
	Expect        string                  `json:"expect"`
	ExpectedError string                  `json:"expected_error,omitempty"`
	PixFmt        string                  `json:"pix_fmt,omitempty"`
	FrameCount    int                     `json:"frame_count,omitempty"`
	FrameSize     int                     `json:"frame_size,omitempty"`
	SourceMD5     string                  `json:"source_md5,omitempty"`
	BitstreamMD5  string                  `json:"bitstream_md5,omitempty"`
	RawVideoMD5   string                  `json:"rawvideo_md5,omitempty"`
	Extract       string                  `json:"extract,omitempty"`
	FrameMD5      []string                `json:"frame_md5,omitempty"`
	FrameGroups   []h264CorpusFrameGroup  `json:"frame_groups,omitempty"`
	Surfaces      []string                `json:"surfaces,omitempty"`
	GuardTags     []string                `json:"guard_tags,omitempty"`
	FeatureTags   []string                `json:"feature_tags,omitempty"`
	Source        string                  `json:"source,omitempty"`
	KnownFailure  *h264CorpusKnownFailure `json:"known_failure,omitempty"`
}

type h264CorpusFrameGroup struct {
	Start     int    `json:"start"`
	Count     int    `json:"count"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	PixFmt    string `json:"pix_fmt"`
	FrameSize int    `json:"frame_size"`
}

type h264CorpusKnownFailure struct {
	Class          string `json:"class"`
	DetailContains string `json:"detail_contains"`
}

type h264RealVectorExclusion struct {
	Ref         string   `json:"ref"`
	Source      string   `json:"source"`
	Reason      string   `json:"reason"`
	FeatureTags []string `json:"feature_tags,omitempty"`
}

type h264RealVectorUpstreamInventoryRef struct {
	Ref       string   `json:"ref"`
	Source    string   `json:"source"`
	Locations []string `json:"locations"`
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

func TestH264RealVectorStrictOracle(t *testing.T) {
	if os.Getenv("GOH264_REAL_VECTOR_STRICT") != "1" {
		t.Skip("set GOH264_REAL_VECTOR_STRICT=1 to run public H.264 vectors as strict decode-ok oracle rows")
	}
	manifest := readH264CorpusManifest(t, defaultH264RealVectorManifest)
	failures := readH264CorpusManifest(t, defaultH264RealVectorFailureManifest)
	failureByID := h264CorpusFailureLedgerByID(t, manifest, failures)
	if filter := h264CorpusFilterTokens(); len(filter) != 0 {
		manifest = filterH264CorpusEntries(manifest, filter)
		if len(manifest) == 0 {
			t.Fatalf("%s: no corpus entries matched GOH264_CORPUS_FILTER=%q", defaultH264RealVectorManifest, os.Getenv("GOH264_CORPUS_FILTER"))
		}
	}

	var strictEntries []h264CorpusEntry
	var knownRedIDs []string
	for _, entry := range manifest {
		if _, knownRed := failureByID[entry.ID]; knownRed {
			knownRedIDs = append(knownRedIDs, entry.ID)
			continue
		}
		strictEntries = append(strictEntries, entry)
	}
	if len(knownRedIDs) != 0 {
		t.Logf("strict oracle excludes known-red ids covered by %s: %s", defaultH264RealVectorFailureManifest, strings.Join(knownRedIDs, ","))
	}
	if len(strictEntries) == 0 {
		t.Skipf("strict oracle selected only known-red rows; run GOH264_REAL_VECTOR_FAILURES=1 or GOH264_REAL_VECTOR_MATRIX=1 to exercise them")
	}
	testH264CorpusEntries(t, defaultH264RealVectorManifest, strictEntries)
}

func TestH264RealVectorKnownRedStrict(t *testing.T) {
	if !h264RealVectorRedOracleEnabled() {
		t.Skip("set GOH264_REAL_VECTOR_RED=1 or GOH264_REAL_VECTOR_STRICT_FAILURES=1 to run known-red public vectors as strict decode-ok oracle rows")
	}
	failures := h264RealVectorFailureEntriesForEnv(t, readH264CorpusManifest(t, defaultH264RealVectorFailureManifest))
	testH264CorpusEntries(t, defaultH264RealVectorFailureManifest, failures)
}

func TestH264RealVectorRedQueue(t *testing.T) {
	if os.Getenv("GOH264_REAL_VECTOR_RED_QUEUE") != "1" {
		t.Skip("set GOH264_REAL_VECTOR_RED_QUEUE=1 to run known-red public vectors as an intentionally failing fix queue")
	}
	failures := h264RealVectorFailureEntriesForEnv(t, readH264CorpusManifest(t, defaultH264RealVectorFailureManifest))
	if len(failures) == 0 {
		t.Skipf("%s has no known-red rows", defaultH264RealVectorFailureManifest)
	}
	t.Logf("public-vector red queue selected=%d ids=%s", len(failures), strings.Join(h264CorpusEntryIDs(failures), ","))
	t.Logf("public-vector red queue lanes: %s", h264CorpusKnownRedLaneSummary(failures))
	testH264CorpusEntries(t, defaultH264RealVectorFailureManifest, failures)
	if !t.Failed() {
		t.Fatalf("public-vector red queue unexpectedly passed; update fixed row(s) in %s and rerun the matrix", defaultH264RealVectorFailureManifest)
	}
}

func TestH264RealVectorKnownRedFilterSelected(t *testing.T) {
	if !h264RealVectorRedOracleEnabled() {
		t.Skip("set GOH264_REAL_VECTOR_RED=1 or GOH264_REAL_VECTOR_STRICT_FAILURES=1 to require that the current filter selects known-red rows")
	}
	failures := readH264CorpusManifest(t, defaultH264RealVectorFailureManifest)
	failures = h264RealVectorFailureEntriesForEnv(t, failures)
	t.Logf("known-red filter selected=%d ids=%s", len(failures), strings.Join(h264CorpusEntryIDs(failures), ","))
}

func TestH264RealVectorFailureLedgerIntegrity(t *testing.T) {
	manifest := readH264CorpusManifest(t, defaultH264RealVectorManifest)
	failures := readH264CorpusManifest(t, defaultH264RealVectorFailureManifest)
	if len(manifest) == 0 {
		t.Fatal("real-vector manifest is empty")
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
		if failure.Expect != "decode-ok" && failure.Expect != "metadata-ok" {
			t.Fatalf("%s: failure ledger rows must stay oracle rows, got %q", failure.ID, failure.Expect)
		}
		validateH264CorpusKnownFailure(t, failure)
		if _, ok := failedIDs[failure.ID]; ok {
			t.Fatalf("%s: duplicate failure-ledger id", failure.ID)
		}
		failedIDs[failure.ID] = struct{}{}
		manifestEntry, ok := byID[failure.ID]
		if !ok {
			t.Fatalf("%s: failure-ledger row is missing from %s", failure.ID, defaultH264RealVectorManifest)
		}
		if !reflect.DeepEqual(h264CorpusEntryWithoutKnownFailure(failure), h264CorpusEntryWithoutKnownFailure(manifestEntry)) {
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
	if len(failures) == 0 {
		t.Skipf("%s has no known-red rows", defaultH264RealVectorFailureManifest)
	}
	manifestByID := make(map[string]h264CorpusEntry, len(manifest))
	for _, entry := range manifest {
		manifestByID[entry.ID] = entry
	}

	focusTokens := []string{"mbaff", "paff", "picaff", "field", "high", "chroma", "b-slice", "direct", "weighted", "partitioned-b", "partitioned-p", "slice-boundary"}
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
				if !reflect.DeepEqual(h264CorpusEntryWithoutKnownFailure(failure), h264CorpusEntryWithoutKnownFailure(manifestEntry)) {
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

func TestH264RealVectorLaneCoverage(t *testing.T) {
	manifest := readH264CorpusManifest(t, defaultH264RealVectorManifest)
	failures := readH264CorpusManifest(t, defaultH264RealVectorFailureManifest)
	for _, lane := range []struct {
		name     string
		tokens   []string
		knownRed bool
	}{
		{name: "implicit weighted B", tokens: []string{"implicit-weight-b"}},
		{name: "partitioned P", tokens: []string{"partitioned-p"}},
		{name: "partitioned B", tokens: []string{"partitioned-b"}},
		{name: "PIC-AFF", tokens: []string{"picaff"}},
		{name: "slice boundary", tokens: []string{"slice-boundary"}},
		{name: "high deblock boundary", tokens: []string{"high", "deblock", "slice-boundary"}},
		{name: "high no-deblock boundary", tokens: []string{"high", "no-deblock", "slice-boundary"}},
		{name: "high10", tokens: []string{"high10"}},
		{name: "cabac chroma", tokens: []string{"cabac", "chroma"}},
	} {
		lane := lane
		t.Run(lane.name, func(t *testing.T) {
			if got := filterH264CorpusEntries(append([]h264CorpusEntry(nil), manifest...), lane.tokens); len(got) == 0 {
				t.Fatalf("real-vector manifest has no rows for tokens %v", lane.tokens)
			}
			if lane.knownRed {
				if got := filterH264CorpusEntries(append([]h264CorpusEntry(nil), failures...), lane.tokens); len(got) == 0 {
					t.Fatalf("failure ledger has no known-red rows for tokens %v", lane.tokens)
				}
			}
		})
	}
}

func TestH264RealVectorLanePublicSurfaceCoverage(t *testing.T) {
	manifest := readH264CorpusManifest(t, defaultH264RealVectorManifest)
	byID := make(map[string]h264CorpusEntry, len(manifest))
	for _, entry := range manifest {
		byID[entry.ID] = entry
	}

	fullConfiguredSurfaces := []string{"annexb", "avc", "configured-avc", "configured-samples", "auto"}
	inBandAVCSurfaces := []string{"annexb", "avc"}
	for _, lane := range []struct {
		name     string
		id       string
		tags     []string
		surfaces []string
	}{
		{
			name:     "implicit weighted B",
			id:       "fate/h264-conformance/cvwp3-toshiba-e",
			tags:     []string{"implicit-weight-b", "weighted-bipred"},
			surfaces: inBandAVCSurfaces,
		},
		{
			name:     "CABAC chroma QP",
			id:       "fate/h264-conformance/cacqp3-sony-d",
			tags:     []string{"cabac", "chroma-qp", "multiple-slices"},
			surfaces: inBandAVCSurfaces,
		},
		{
			name:     "partitioned P",
			id:       "fate/h264-conformance/camp-mot-frm0-full",
			tags:     []string{"cabac", "partitioned-p", "motion"},
			surfaces: fullConfiguredSurfaces,
		},
		{
			name:     "partitioned B",
			id:       "fate/h264-conformance/cvmp-mot-frm-l31-b",
			tags:     []string{"cavlc", "partitioned-b", "b-slice", "motion"},
			surfaces: fullConfiguredSurfaces,
		},
		{
			name:     "PIC-AFF P",
			id:       "fate/h264-conformance/camp-mot-picaff0-full",
			tags:     []string{"cabac", "picaff", "field", "motion"},
			surfaces: fullConfiguredSurfaces,
		},
		{
			name:     "PIC-AFF B",
			id:       "fate/h264-conformance/cvmp-mot-picaff0-full-b",
			tags:     []string{"cavlc", "picaff", "field", "b-slice", "motion"},
			surfaces: fullConfiguredSurfaces,
		},
		{
			name:     "field slice boundary",
			id:       "fate/h264-conformance/slice2-field-aurora4",
			tags:     []string{"field", "multiple-slices", "slice-boundary"},
			surfaces: fullConfiguredSurfaces,
		},
		{
			name:     "high deblock slice boundary",
			id:       "fate/h264-conformance/frext-hpca-fl-brcm-c",
			tags:     []string{"high", "cabac", "deblock", "slice-boundary"},
			surfaces: fullConfiguredSurfaces,
		},
		{
			name:     "high no-deblock slice boundary",
			id:       "fate/h264-conformance/frext-hpca-flnl-brcm-c",
			tags:     []string{"high", "cabac", "no-deblock", "slice-boundary"},
			surfaces: fullConfiguredSurfaces,
		},
		{
			name:     "high10 large NAL",
			id:       "fate/h264-conformance/frext-pph10i1-panasonic-a",
			tags:     []string{"high10", "10-bit", "intra"},
			surfaces: []string{"annexb", "avc4", "configured-avc4"},
		},
		{
			name:     "SPS reinit AVC packet",
			id:       "fate/h264/reinit-small-422-9-to-small-420-9",
			tags:     []string{"reinit", "sps-reinit", "chroma-format-change"},
			surfaces: inBandAVCSurfaces,
		},
	} {
		lane := lane
		t.Run(lane.name, func(t *testing.T) {
			entry, ok := byID[lane.id]
			if !ok {
				t.Fatalf("real-vector manifest missing %s", lane.id)
			}
			for _, tag := range lane.tags {
				if !h264CorpusEntryHasFeatureTag(entry, tag) {
					t.Fatalf("%s missing feature tag %q", lane.id, tag)
				}
			}
			for _, surface := range lane.surfaces {
				if !h264CorpusEntryHasSurface(entry, surface) {
					t.Fatalf("%s missing public decode surface %q", lane.id, surface)
				}
			}
		})
	}
}

func TestH264RealVectorFieldMBAFFCoversPacketizedPublicSurfaces(t *testing.T) {
	manifest := readH264CorpusManifest(t, defaultH264RealVectorManifest)
	const id = "fate/h264-conformance/cavlc-mot-mbaff0-full-b"
	var entry h264CorpusEntry
	found := false
	for _, candidate := range manifest {
		if candidate.ID == id {
			entry = candidate
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("real-vector manifest missing %s", id)
	}
	for _, tag := range []string{"cavlc", "mbaff", "field", "b-slice", "motion"} {
		if !h264CorpusEntryHasFeatureTag(entry, tag) {
			t.Fatalf("%s missing feature tag %q", id, tag)
		}
	}
	for _, surface := range []string{"annexb", "avc4", "configured-avc4"} {
		if !h264CorpusEntryHasSurface(entry, surface) {
			t.Fatalf("%s missing public decode surface %q", id, surface)
		}
	}
}

func TestH264RealVectorUpstreamFATECoverage(t *testing.T) {
	if os.Getenv("GOH264_REAL_VECTOR_UPSTREAM_AUDIT") != "1" {
		t.Skip("set GOH264_REAL_VECTOR_UPSTREAM_AUDIT=1 after scripts/fetch-upstream.sh to audit pinned FFmpeg FATE coverage")
	}
	upstream := os.Getenv("GOH264_UPSTREAM")
	if upstream == "" {
		upstream = filepath.Join(".upstream", "ffmpeg-n8.0.1")
	}
	fateDir := filepath.Join(upstream, "tests", "fate")
	if _, err := os.Stat(filepath.Join(fateDir, "h264.mak")); err != nil {
		t.Fatalf("pinned FFmpeg FATE files missing under %s: %v; run scripts/fetch-upstream.sh", fateDir, err)
	}

	manifest := readH264CorpusManifest(t, defaultH264RealVectorManifest)
	manifestPaths := h264RealVectorManifestFATESamplePaths(manifest)
	upstreamRefs := h264UpstreamFATEH264SampleRefs(t, fateDir)
	inventoryRefs := h264RealVectorUpstreamFATEInventoryByRef(t, readH264RealVectorUpstreamInventory(t, defaultH264RealVectorUpstreamInventory))
	assertH264UpstreamInventoryMatchesGeneratedRefs(t, inventoryRefs, upstreamRefs)
	excludedRefs := h264RealVectorExclusionsByRef(t, readH264RealVectorExclusions(t, defaultH264RealVectorExclusionManifest))

	var missing []string
	var represented, excluded int
	for ref, locations := range upstreamRefs {
		if _, ok := manifestPaths[ref]; ok {
			represented++
			continue
		}
		if exclusion, ok := excludedRefs[ref]; ok {
			excluded++
			t.Logf("excluded upstream H.264-ish ref %s: %s", ref, exclusion.Reason)
			continue
		}
		missing = append(missing, fmt.Sprintf("%s (%s)", ref, strings.Join(locations, ",")))
	}
	sort.Strings(missing)
	if len(missing) != 0 {
		t.Fatalf("real-vector manifest is missing pinned FFmpeg H.264 FATE sample refs:\n%s", strings.Join(missing, "\n"))
	}
	for ref := range excludedRefs {
		if _, ok := upstreamRefs[ref]; !ok {
			t.Fatalf("documented excluded upstream ref %s is no longer produced by the pinned FATE scan", ref)
		}
	}
	t.Logf("upstream H.264 FATE sample refs=%d represented=%d excluded=%d manifest_entries=%d",
		len(upstreamRefs), represented, excluded, len(manifest))
}

func TestH264RealVectorImportedUpstreamInventory(t *testing.T) {
	manifest := readH264CorpusManifest(t, defaultH264RealVectorManifest)
	manifestRefs := h264RealVectorManifestFATESamplePaths(manifest)
	excludedRefs := h264RealVectorExclusionsByRef(t, readH264RealVectorExclusions(t, defaultH264RealVectorExclusionManifest))
	inventoryRefs := h264RealVectorUpstreamInventoryByRef(t, readH264RealVectorUpstreamInventory(t, defaultH264RealVectorUpstreamInventory))

	var missing, extraManifest, staleExclusions []string
	var represented, excluded int
	for ref := range inventoryRefs {
		if _, ok := manifestRefs[ref]; ok {
			represented++
			continue
		}
		if _, ok := excludedRefs[ref]; ok {
			excluded++
			continue
		}
		missing = append(missing, ref)
	}
	for ref := range manifestRefs {
		if _, ok := inventoryRefs[ref]; !ok {
			extraManifest = append(extraManifest, ref)
		}
	}
	for ref := range excludedRefs {
		if _, ok := inventoryRefs[ref]; !ok {
			staleExclusions = append(staleExclusions, ref)
		}
	}
	sort.Strings(missing)
	sort.Strings(extraManifest)
	sort.Strings(staleExclusions)
	if len(missing) != 0 {
		t.Fatalf("real-vector manifest/exclusions are missing imported public refs:\n%s", strings.Join(missing, "\n"))
	}
	if len(extraManifest) != 0 {
		t.Fatalf("real-vector manifest has refs outside imported public inventory:\n%s", strings.Join(extraManifest, "\n"))
	}
	if len(staleExclusions) != 0 {
		t.Fatalf("real-vector exclusions have refs outside imported public inventory:\n%s", strings.Join(staleExclusions, "\n"))
	}
	t.Logf("imported public H.264 inventory refs=%d represented=%d excluded=%d manifest_entries=%d",
		len(inventoryRefs), represented, excluded, len(manifest))
}

func TestH264RealVectorPinnedFATEInventory(t *testing.T) {
	manifest := readH264CorpusManifest(t, defaultH264RealVectorManifest)
	represented := h264RealVectorManifestFATESamplePaths(manifest)
	excluded := h264RealVectorExclusionsByRef(t, readH264RealVectorExclusions(t, defaultH264RealVectorExclusionManifest))
	importedEntries := readH264RealVectorUpstreamInventory(t, defaultH264RealVectorUpstreamInventory)
	imported := h264RealVectorUpstreamInventoryByRef(t, importedEntries)
	importedFATE := h264RealVectorUpstreamFATEInventoryByRef(t, importedEntries)

	const wantManifestRows = 225
	const wantExcludedRefs = 1
	const wantImportedRefs = 226
	const wantImportedFATERefs = 224
	if len(manifest) != wantManifestRows {
		t.Fatalf("real-vector manifest rows = %d, want %d", len(manifest), wantManifestRows)
	}
	if len(excluded) != wantExcludedRefs {
		t.Fatalf("excluded pinned FFmpeg FATE refs = %d, want %d", len(excluded), wantExcludedRefs)
	}
	if len(imported) != wantImportedRefs {
		t.Fatalf("imported public H.264 refs = %d, want %d", len(imported), wantImportedRefs)
	}
	if len(importedFATE) != wantImportedFATERefs {
		t.Fatalf("imported pinned FFmpeg FATE refs = %d, want %d", len(importedFATE), wantImportedFATERefs)
	}
	if _, ok := represented["h264-conformance/FM1_BT_B.h264"]; !ok {
		t.Fatal("malformed H.264 conformance vector FM1_BT_B.h264 must be represented as a negative decoder row")
	}
	if _, ok := excluded["mkv/h264_tta_undecodable.mkv"]; !ok {
		t.Fatal("non-H.264 mkv/h264_tta_undecodable.mkv must stay explicitly excluded")
	}
}

func TestH264RealVectorDocumentationCounts(t *testing.T) {
	manifest := readH264CorpusManifest(t, defaultH264RealVectorManifest)
	failures := readH264CorpusManifest(t, defaultH264RealVectorFailureManifest)
	excluded := h264RealVectorExclusionsByRef(t, readH264RealVectorExclusions(t, defaultH264RealVectorExclusionManifest))
	importedEntries := readH264RealVectorUpstreamInventory(t, defaultH264RealVectorUpstreamInventory)
	imported := h264RealVectorUpstreamInventoryByRef(t, importedEntries)
	importedFATE := h264RealVectorUpstreamFATEInventoryByRef(t, importedEntries)

	green := len(manifest) - len(failures)
	auxiliary := len(imported) - len(importedFATE)

	requireH264DocCountSnippet(t, "README.md", fmt.Sprintf("| Imported public H.264 vector refs | %d |", len(imported)))
	requireH264DocCountSnippet(t, "README.md", fmt.Sprintf("| Pinned FFmpeg FATE refs in imported inventory | %d |", len(importedFATE)))
	requireH264DocCountSnippet(t, "README.md", fmt.Sprintf("| Selected public H.264 vectors | %d |", len(manifest)))
	requireH264DocCountSnippet(t, "README.md", fmt.Sprintf("| Green oracle rows | %d |", green))
	requireH264DocCountSnippet(t, "README.md", fmt.Sprintf("| Known-red rows in `failures.jsonl` | %d |", len(failures)))
	requireH264DocCountSnippet(t, "README.md", fmt.Sprintf("| Explicitly excluded upstream H.264-ish rows | %d |", len(excluded)))
	requireH264DocCountSnippet(t, "README.md", fmt.Sprintf("The selected manifest represents %d imported decoder-facing refs", len(manifest)))

	requireH264DocCountSnippet(t, "docs/source-truth.md",
		fmt.Sprintf("Public vectors: %d imported public refs, %d selected decoder-facing manifest rows, %d green oracle rows, %d known-red",
			len(imported), len(manifest), green, len(failures)))
	requireH264DocCountSnippet(t, "docs/production-readiness.md",
		fmt.Sprintf("currently imports %d public H.264 refs: %d generated from pinned FFmpeg `n8.0.1` FATE makefiles and %d auxiliary",
			len(imported), len(importedFATE), auxiliary))
}

func TestH264RealVectorFailureLedgerFreshness(t *testing.T) {
	if !h264RealVectorsEnabled() && os.Getenv("GOH264_REAL_VECTOR_FAILURES") != "1" {
		t.Skip("set GOH264_REAL_VECTOR_FAILURES=1, GOH264_REAL_VECTORS=1, or GOH264_ORACLE=1 to verify red public vector rows")
	}
	failures := h264RealVectorFailureEntriesForEnv(t, readH264CorpusManifest(t, defaultH264RealVectorFailureManifest))
	for _, entry := range failures {
		entry := entry
		t.Run(entry.ID, func(t *testing.T) {
			validateH264CorpusEntry(t, entry)
			if entry.Expect != "decode-ok" && entry.Expect != "metadata-ok" {
				t.Fatalf("%s: failure ledger rows must stay oracle rows, got %q", entry.ID, entry.Expect)
			}
			validateH264CorpusKnownFailure(t, entry)
			if !h264CorpusEntryHasSurface(entry, "annexb") {
				t.Fatalf("%s: failure-ledger freshness currently requires an annexb surface", entry.ID)
			}
			path := materializeH264CorpusEntry(t, defaultH264RealVectorFailureManifest, entry)
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}
			assertCorpusBitstreamMD5(t, entry, data)
			matches, detail := h264CorpusAnnexBMatchesExpectedOracle(t, entry, data)
			if matches {
				t.Fatalf("%s: failure-ledger row now matches oracle; update %s", entry.ID, defaultH264RealVectorFailureManifest)
			}
			assertH264CorpusKnownFailureStillCurrent(t, entry, detail)
			t.Logf("%s: still red: %s", entry.ID, h264CorpusFailureDetail(entry, detail))
		})
	}
}

func TestH264RealVectorFailureMatrix(t *testing.T) {
	if os.Getenv("GOH264_REAL_VECTOR_MATRIX") != "1" {
		t.Skip("set GOH264_REAL_VECTOR_MATRIX=1 to run the public-vector pass/known-red matrix")
	}
	manifest := readH264CorpusManifest(t, defaultH264RealVectorManifest)
	failures := readH264CorpusManifest(t, defaultH264RealVectorFailureManifest)
	failureByID := h264CorpusFailureLedgerByID(t, manifest, failures)
	if filter := h264CorpusFilterTokens(); len(filter) != 0 {
		manifest = filterH264CorpusEntries(manifest, filter)
		if len(manifest) == 0 {
			t.Fatalf("%s: no manifest entries matched GOH264_CORPUS_FILTER=%q; available filters: %s",
				defaultH264RealVectorManifest, os.Getenv("GOH264_CORPUS_FILTER"), h264CorpusFailureFilterSummary(readH264CorpusManifest(t, defaultH264RealVectorManifest)))
		}
		h264RealVectorFailureEntriesForEnv(t, failures)
	}

	var green, knownRed int
	var redRows []h264CorpusEntry
	for _, entry := range manifest {
		entry := entry
		t.Run(entry.ID, func(t *testing.T) {
			validateH264CorpusEntry(t, entry)
			if entry.Expect != "decode-ok" && entry.Expect != "metadata-ok" && entry.Expect != "decode-error" {
				t.Fatalf("%s: real-vector matrix only supports decode-ok, metadata-ok, and decode-error oracle rows, got %q", entry.ID, entry.Expect)
			}
			path := materializeH264CorpusEntry(t, defaultH264RealVectorManifest, entry)
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}
			assertCorpusBitstreamMD5(t, entry, data)
			matches, detail := h264CorpusAnnexBMatchesExpectedOracle(t, entry, data)
			if failure, ok := failureByID[entry.ID]; ok {
				knownRed++
				redRows = append(redRows, failure)
				if matches {
					t.Fatalf("%s: known-red row now matches oracle; update %s", entry.ID, defaultH264RealVectorFailureManifest)
				}
				assertH264CorpusKnownFailureStillCurrent(t, failure, detail)
				t.Logf("known-red: %s", h264CorpusFailureDetail(failure, detail))
				return
			}
			green++
			if !matches {
				t.Fatalf("%s: unexpected public-vector failure: %s", entry.ID, h264CorpusFailureDetail(entry, detail))
			}
			t.Logf("green: %s", detail)
		})
	}
	t.Logf("public-vector matrix selected=%d green=%d known-red=%d", len(manifest), green, knownRed)
	if len(redRows) != 0 {
		t.Logf("public-vector known-red lanes: %s", h264CorpusKnownRedLaneSummary(redRows))
	}
}

func requireH264DocCountSnippet(t *testing.T, path string, snippet string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	doc := strings.Join(strings.Fields(string(data)), " ")
	want := strings.Join(strings.Fields(snippet), " ")
	if !strings.Contains(doc, want) {
		t.Fatalf("%s missing public-vector count snippet %q", path, snippet)
	}
}

func validateH264CorpusKnownFailure(t *testing.T, entry h264CorpusEntry) {
	t.Helper()
	if entry.KnownFailure == nil {
		t.Fatalf("%s: failure-ledger row must record known_failure", entry.ID)
	}
	if entry.KnownFailure.Class == "" || entry.KnownFailure.DetailContains == "" {
		t.Fatalf("%s: known_failure needs class and detail_contains: %+v", entry.ID, entry.KnownFailure)
	}
	switch entry.KnownFailure.Class {
	case "decode-error", "frame-count-mismatch", "pixel-format-mismatch", "raw-size-mismatch", "source-md5-mismatch", "bitstream-md5-mismatch", "raw-md5-mismatch", "oracle-mismatch", "input-missing":
	default:
		t.Fatalf("%s: unknown known_failure class %q", entry.ID, entry.KnownFailure.Class)
	}
}

func h264CorpusEntryWithoutKnownFailure(entry h264CorpusEntry) h264CorpusEntry {
	entry.KnownFailure = nil
	return entry
}

func h264CorpusFailureLedgerByID(t *testing.T, manifest []h264CorpusEntry, failures []h264CorpusEntry) map[string]h264CorpusEntry {
	t.Helper()
	manifestByID := make(map[string]h264CorpusEntry, len(manifest))
	for _, entry := range manifest {
		manifestByID[entry.ID] = entry
	}
	failureByID := make(map[string]h264CorpusEntry, len(failures))
	for _, failure := range failures {
		validateH264CorpusEntry(t, failure)
		validateH264CorpusKnownFailure(t, failure)
		if _, ok := failureByID[failure.ID]; ok {
			t.Fatalf("%s: duplicate failure-ledger id", failure.ID)
		}
		manifestEntry, ok := manifestByID[failure.ID]
		if !ok {
			t.Fatalf("%s: failure-ledger row is missing from %s", failure.ID, defaultH264RealVectorManifest)
		}
		if !reflect.DeepEqual(h264CorpusEntryWithoutKnownFailure(failure), h264CorpusEntryWithoutKnownFailure(manifestEntry)) {
			t.Fatalf("%s: failure-ledger row drifted from real-vector manifest\nfailure=%+v\nmanifest=%+v", failure.ID, failure, manifestEntry)
		}
		failureByID[failure.ID] = failure
	}
	return failureByID
}

func assertH264CorpusKnownFailureStillCurrent(t *testing.T, entry h264CorpusEntry, detail string) {
	t.Helper()
	validateH264CorpusKnownFailure(t, entry)
	gotClass := h264CorpusOracleFailureClass(detail)
	if gotClass != entry.KnownFailure.Class {
		t.Fatalf("%s: current failure class = %q, want known_failure class %q; detail=%s", entry.ID, gotClass, entry.KnownFailure.Class, detail)
	}
	if !strings.Contains(strings.ToLower(detail), strings.ToLower(entry.KnownFailure.DetailContains)) {
		t.Fatalf("%s: current failure detail %q does not contain known_failure detail %q", entry.ID, detail, entry.KnownFailure.DetailContains)
	}
}

func h264RealVectorsEnabled() bool {
	return os.Getenv("GOH264_REAL_VECTORS") == "1" || os.Getenv("GOH264_ORACLE") == "1"
}

func h264RealVectorRedOracleEnabled() bool {
	return os.Getenv("GOH264_REAL_VECTOR_RED") == "1" || os.Getenv("GOH264_REAL_VECTOR_STRICT_FAILURES") == "1"
}

func h264RealVectorFailureEntriesForEnv(t *testing.T, failures []h264CorpusEntry) []h264CorpusEntry {
	t.Helper()
	if len(failures) == 0 {
		return failures
	}
	if filter := h264CorpusFilterTokens(); len(filter) != 0 {
		filtered := filterH264CorpusEntries(append([]h264CorpusEntry(nil), failures...), filter)
		if len(filtered) == 0 {
			t.Fatalf("%s: no known-red entries matched GOH264_CORPUS_FILTER=%q; available known-red filters: %s",
				defaultH264RealVectorFailureManifest, os.Getenv("GOH264_CORPUS_FILTER"), h264CorpusFailureFilterSummary(failures))
		}
		return filtered
	}
	return failures
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
	testH264CorpusEntries(t, manifest, entries)
}

func testH264CorpusEntries(t *testing.T, manifest string, entries []h264CorpusEntry) {
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
					switch entry.Expect {
					case "decode-error":
						assertH264CorpusExpectedDecodeError(t, entry, err)
						return
					case "unsupported":
						assertH264CorpusUnsupported(t, entry, err)
						return
					}
					if err != nil {
						failH264CorpusOracle(t, entry, fmt.Sprintf("%s decode: %v", surface, err))
					}
					switch entry.Expect {
					case "decode-ok":
						assertH264CorpusFrames(t, entry, frames)
					case "metadata-ok":
						assertH264CorpusFrameMetadata(t, entry, frames)
					}
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

func h264CorpusEntryHasFeatureTag(entry h264CorpusEntry, want string) bool {
	for _, tag := range entry.FeatureTags {
		if tag == want {
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

func TestH264RealVectorRedOracleEnv(t *testing.T) {
	t.Setenv("GOH264_REAL_VECTOR_RED", "")
	t.Setenv("GOH264_REAL_VECTOR_STRICT_FAILURES", "")
	if h264RealVectorRedOracleEnabled() {
		t.Fatal("red oracle disabled env returned enabled")
	}

	t.Setenv("GOH264_REAL_VECTOR_RED", "1")
	if !h264RealVectorRedOracleEnabled() {
		t.Fatal("GOH264_REAL_VECTOR_RED=1 did not enable red oracle")
	}

	t.Setenv("GOH264_REAL_VECTOR_RED", "")
	t.Setenv("GOH264_REAL_VECTOR_STRICT_FAILURES", "1")
	if !h264RealVectorRedOracleEnabled() {
		t.Fatal("GOH264_REAL_VECTOR_STRICT_FAILURES=1 did not enable red oracle")
	}
}

func TestH264RealVectorRedQueueEnv(t *testing.T) {
	t.Setenv("GOH264_REAL_VECTOR_RED_QUEUE", "")
	if os.Getenv("GOH264_REAL_VECTOR_RED_QUEUE") == "1" {
		t.Fatal("red queue env unexpectedly enabled")
	}

	t.Setenv("GOH264_REAL_VECTOR_RED_QUEUE", "1")
	if os.Getenv("GOH264_REAL_VECTOR_RED_QUEUE") != "1" {
		t.Fatal("GOH264_REAL_VECTOR_RED_QUEUE=1 did not enable red queue")
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
		entry.SourceMD5,
		entry.Extract,
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

func h264RealVectorManifestFATESamplePaths(entries []h264CorpusEntry) map[string]struct{} {
	paths := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		if entry.Path != "" {
			paths[h264CleanFATESamplePath(entry.Path)] = struct{}{}
		}
		if suffix := h264FATESuiteURLSuffix(entry.URL); suffix != "" {
			paths[suffix] = struct{}{}
		}
	}
	return paths
}

func h264CleanFATESamplePath(path string) string {
	path = filepath.ToSlash(filepath.Clean(path))
	return strings.TrimPrefix(path, "fate-suite/")
}

func h264FATESuiteURLSuffix(url string) string {
	const prefix = "https://fate-suite.ffmpeg.org/"
	if !strings.HasPrefix(url, prefix) {
		return ""
	}
	return strings.TrimPrefix(url, prefix)
}

func h264UpstreamFATEH264SampleRefs(t *testing.T, fateDir string) map[string][]string {
	t.Helper()
	refs := make(map[string][]string)
	makFiles, err := filepath.Glob(filepath.Join(fateDir, "*.mak"))
	if err != nil {
		t.Fatalf("glob FFmpeg FATE makefiles: %v", err)
	}
	for _, path := range makFiles {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		name := filepath.Base(path)
		for lineNo, line := range strings.Split(string(data), "\n") {
			for _, ref := range h264TargetSampleRefsFromLine(line) {
				if strings.Contains(ref, "$(@:fate-h264-%=%)") {
					for _, reinitRef := range h264UpstreamReinitSampleRefs() {
						h264AddUpstreamFATERef(refs, reinitRef, fmt.Sprintf("%s:%d:reinit", name, lineNo+1))
					}
					continue
				}
				if !h264FATERefLooksH264ish(ref, line) {
					continue
				}
				h264AddUpstreamFATERef(refs, ref, fmt.Sprintf("%s:%d", name, lineNo+1))
			}
		}
	}

	cbsPath := filepath.Join(fateDir, "cbs.mak")
	for _, sample := range h264MakeVariableWords(t, cbsPath, "FATE_CBS_H264_CONFORMANCE_SAMPLES") {
		h264AddUpstreamFATERef(refs, "h264-conformance/"+sample, "cbs.mak:FATE_CBS_H264_CONFORMANCE_SAMPLES")
	}
	for _, sample := range h264MakeVariableWords(t, cbsPath, "FATE_CBS_H264_SAMPLES") {
		h264AddUpstreamFATERef(refs, "h264/"+sample, "cbs.mak:FATE_CBS_H264_SAMPLES")
	}
	h264AddUpstreamFATERef(refs, "h264/interlaced_crop.mp4", "cbs.mak:FATE_CBS_DISCARD_TEST")
	return refs
}

func h264TargetSampleRefsFromLine(line string) []string {
	const marker = "$(TARGET_SAMPLES)/"
	var refs []string
	for {
		idx := strings.Index(line, marker)
		if idx < 0 {
			return refs
		}
		start := idx + len(marker)
		end := start
		for end < len(line) {
			switch line[end] {
			case ' ', '\t', '\r', '\n', '\\', '"', '\'':
				goto found
			}
			end++
		}
	found:
		ref := strings.TrimRight(line[start:end], ")")
		if ref != "" {
			refs = append(refs, ref)
		}
		line = line[end:]
	}
}

func h264FATERefLooksH264ish(ref string, context string) bool {
	haystack := strings.ToLower(ref + " " + context)
	return strings.Contains(haystack, "h264")
}

func h264UpstreamReinitSampleRefs() []string {
	return []string{
		"h264/reinit-large_420_8-to-small_420_8.h264",
		"h264/reinit-small_420_8-to-large_444_10.h264",
		"h264/reinit-small_420_9-to-small_420_8.h264",
		"h264/reinit-small_422_9-to-small_420_9.h264",
	}
}

func h264AddUpstreamFATERef(refs map[string][]string, ref string, location string) {
	ref = h264CleanFATESamplePath(ref)
	for _, existing := range refs[ref] {
		if existing == location {
			return
		}
	}
	refs[ref] = append(refs[ref], location)
}

func h264MakeVariableWords(t *testing.T, path string, name string) []string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	lines := strings.Split(string(data), "\n")
	prefix := name + " ="
	for i, line := range lines {
		if !strings.HasPrefix(line, prefix) {
			continue
		}
		var words []string
		for ; i < len(lines); i++ {
			text := strings.TrimSpace(lines[i])
			if i == 0 || strings.HasPrefix(text, name+" =") {
				parts := strings.SplitN(text, "=", 2)
				text = ""
				if len(parts) == 2 {
					text = strings.TrimSpace(parts[1])
				}
			}
			if text == "" && i != 0 {
				break
			}
			continued := strings.HasSuffix(text, "\\")
			text = strings.TrimSpace(strings.TrimSuffix(text, "\\"))
			if text != "" {
				words = append(words, strings.Fields(text)...)
			}
			if !continued {
				break
			}
		}
		return words
	}
	t.Fatalf("%s: missing make variable %s", path, name)
	return nil
}

func assertH264UpstreamInventoryMatchesGeneratedRefs(t *testing.T, imported map[string]h264RealVectorUpstreamInventoryRef, generated map[string][]string) {
	t.Helper()
	var missing, stale, locationDrift []string
	for ref, locations := range generated {
		entry, ok := imported[ref]
		if !ok {
			missing = append(missing, ref)
			continue
		}
		if !reflect.DeepEqual(entry.Locations, locations) {
			locationDrift = append(locationDrift, fmt.Sprintf("%s imported=%v generated=%v", ref, entry.Locations, locations))
		}
	}
	for ref := range imported {
		if _, ok := generated[ref]; !ok {
			stale = append(stale, ref)
		}
	}
	sort.Strings(missing)
	sort.Strings(stale)
	sort.Strings(locationDrift)
	if len(missing) != 0 || len(stale) != 0 || len(locationDrift) != 0 {
		t.Fatalf("checked-in upstream FATE inventory drifted from pinned FFmpeg scan\nmissing:\n%s\nstale:\n%s\nlocation drift:\n%s",
			strings.Join(missing, "\n"), strings.Join(stale, "\n"), strings.Join(locationDrift, "\n"))
	}
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

func h264CorpusKnownRedLaneSummary(entries []h264CorpusEntry) string {
	classCounts := make(map[string]int)
	featureCounts := make(map[string]int)
	for _, entry := range entries {
		if entry.KnownFailure != nil {
			classCounts[entry.KnownFailure.Class]++
		}
		for _, tag := range entry.FeatureTags {
			featureCounts[tag]++
		}
	}
	return fmt.Sprintf("classes=%s features=%s",
		h264CorpusCountSummary(classCounts),
		h264CorpusCountSummary(featureCounts))
}

func h264CorpusCountSummary(counts map[string]int) string {
	if len(counts) == 0 {
		return "(none)"
	}
	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s:%d", key, counts[key]))
	}
	return strings.Join(parts, ",")
}

func TestH264CorpusOracleFailureClass(t *testing.T) {
	tests := []struct {
		detail string
		want   string
	}{
		{"missing /tmp/in.264; set GOH264_CORPUS_FETCH=1", "input-missing"},
		{"decode error: unsupported MBAFF", "decode-error"},
		{"decode error: temporal direct missing colocated ref entry: h264: unsupported bitstream feature", "decode-error"},
		{"frames = 2, want 3", "frame-count-mismatch"},
		{"frame[0] pix_fmt = yuv420p, want yuv422p", "pixel-format-mismatch"},
		{"frame[0] raw size = 10, want 20", "raw-size-mismatch"},
		{"source_md5 = abc, want def", "source-md5-mismatch"},
		{"bitstream_md5 = abc, want def", "bitstream-md5-mismatch"},
		{"rawvideo md5 = abc, want def", "raw-md5-mismatch"},
		{"unexpected oracle detail", "oracle-mismatch"},
	}
	for _, tt := range tests {
		if got := h264CorpusOracleFailureClass(tt.detail); got != tt.want {
			t.Fatalf("h264CorpusOracleFailureClass(%q) = %q, want %q", tt.detail, got, tt.want)
		}
	}
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

func TestValidateH264CorpusEntryAllowsExtractedAnnexB(t *testing.T) {
	validateH264CorpusEntry(t, h264CorpusEntry{
		ID:           "container",
		URL:          "https://example.invalid/sample.mp4",
		Format:       "annexb",
		Expect:       "decode-ok",
		PixFmt:       "yuv420p",
		FrameCount:   2,
		FrameSize:    16,
		SourceMD5:    "11223344556677889900aabbccddeeff",
		BitstreamMD5: "00112233445566778899aabbccddeeff",
		RawVideoMD5:  "ffeeddccbbaa99887766554433221100",
		Extract:      "h264-annexb",
		Surfaces:     []string{"annexb"},
		FeatureTags:  []string{"container", "extracted-annexb"},
		Source:       "test",
	})
}

func TestValidateH264CorpusEntryAllowsMetadataOracle(t *testing.T) {
	validateH264CorpusEntry(t, h264CorpusEntry{
		ID:           "reinit",
		URL:          "https://example.invalid/reinit.h264",
		Format:       "annexb",
		Expect:       "metadata-ok",
		FrameCount:   4,
		BitstreamMD5: "00112233445566778899aabbccddeeff",
		FrameGroups: []h264CorpusFrameGroup{
			{Start: 0, Count: 2, Width: 352, Height: 288, PixFmt: "yuv420p", FrameSize: 152064},
			{Start: 2, Count: 2, Width: 240, Height: 196, PixFmt: "yuv420p", FrameSize: 70560},
		},
		Surfaces:    []string{"annexb"},
		FeatureTags: []string{"reinit"},
		Source:      "test",
	})
}

func TestValidateH264CorpusEntryAllowsExpectedDecodeError(t *testing.T) {
	validateH264CorpusEntry(t, h264CorpusEntry{
		ID:            "negative",
		URL:           "https://example.invalid/malformed.264",
		Format:        "annexb",
		Expect:        "decode-error",
		ExpectedError: "invalid data",
		BitstreamMD5:  "00112233445566778899aabbccddeeff",
		Surfaces:      []string{"annexb"},
		FeatureTags:   []string{"malformed"},
		Source:        "test",
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

func readH264RealVectorExclusions(t *testing.T, path string) []h264RealVectorExclusion {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open real-vector exclusions %s: %v", path, err)
	}
	defer f.Close()

	var entries []h264RealVectorExclusion
	scanner := bufio.NewScanner(f)
	for line := 1; scanner.Scan(); line++ {
		text := strings.TrimSpace(scanner.Text())
		if text == "" || strings.HasPrefix(text, "#") {
			continue
		}
		var entry h264RealVectorExclusion
		if err := json.Unmarshal([]byte(text), &entry); err != nil {
			t.Fatalf("%s:%d: %v", path, line, err)
		}
		validateH264RealVectorExclusion(t, entry)
		entries = append(entries, entry)
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("read real-vector exclusions %s: %v", path, err)
	}
	return entries
}

func readH264RealVectorUpstreamInventory(t *testing.T, path string) []h264RealVectorUpstreamInventoryRef {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open real-vector upstream inventory %s: %v", path, err)
	}
	defer f.Close()

	var entries []h264RealVectorUpstreamInventoryRef
	scanner := bufio.NewScanner(f)
	for line := 1; scanner.Scan(); line++ {
		text := strings.TrimSpace(scanner.Text())
		if text == "" || strings.HasPrefix(text, "#") {
			continue
		}
		var entry h264RealVectorUpstreamInventoryRef
		if err := json.Unmarshal([]byte(text), &entry); err != nil {
			t.Fatalf("%s:%d: %v", path, line, err)
		}
		validateH264RealVectorUpstreamInventoryRef(t, entry)
		entries = append(entries, entry)
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("read real-vector upstream inventory %s: %v", path, err)
	}
	return entries
}

func validateH264RealVectorExclusion(t *testing.T, entry h264RealVectorExclusion) {
	t.Helper()
	if entry.Ref == "" || entry.Source == "" || entry.Reason == "" {
		t.Fatalf("real-vector exclusion needs ref, source, and reason: %+v", entry)
	}
	if clean := h264CleanFATESamplePath(entry.Ref); clean != entry.Ref {
		t.Fatalf("%s: exclusion ref must be source-normalized, got %q", entry.Ref, clean)
	}
	if len(entry.FeatureTags) == 0 {
		t.Fatalf("%s: real-vector exclusion needs feature_tags", entry.Ref)
	}
}

func validateH264RealVectorUpstreamInventoryRef(t *testing.T, entry h264RealVectorUpstreamInventoryRef) {
	t.Helper()
	if entry.Ref == "" || entry.Source == "" || len(entry.Locations) == 0 {
		t.Fatalf("real-vector upstream inventory row needs ref, source, and locations: %+v", entry)
	}
	if clean := h264CleanFATESamplePath(entry.Ref); clean != entry.Ref {
		t.Fatalf("%s: upstream inventory ref must be source-normalized, got %q", entry.Ref, clean)
	}
	for _, location := range entry.Locations {
		if location == "" {
			t.Fatalf("%s: upstream inventory location must be non-empty: %+v", entry.Ref, entry)
		}
	}
}

func h264RealVectorExclusionsByRef(t *testing.T, entries []h264RealVectorExclusion) map[string]h264RealVectorExclusion {
	t.Helper()
	byRef := make(map[string]h264RealVectorExclusion, len(entries))
	for _, entry := range entries {
		ref := h264CleanFATESamplePath(entry.Ref)
		if previous, ok := byRef[ref]; ok {
			t.Fatalf("%s: duplicate real-vector exclusion: previous=%+v current=%+v", ref, previous, entry)
		}
		byRef[ref] = entry
	}
	return byRef
}

func h264RealVectorUpstreamInventoryByRef(t *testing.T, entries []h264RealVectorUpstreamInventoryRef) map[string]h264RealVectorUpstreamInventoryRef {
	t.Helper()
	byRef := make(map[string]h264RealVectorUpstreamInventoryRef, len(entries))
	var previousRef string
	for _, entry := range entries {
		ref := h264CleanFATESamplePath(entry.Ref)
		if previous, ok := byRef[ref]; ok {
			t.Fatalf("%s: duplicate real-vector upstream inventory ref: previous=%+v current=%+v", ref, previous, entry)
		}
		if previousRef != "" && ref <= previousRef {
			t.Fatalf("%s: real-vector upstream inventory must stay sorted after %s", ref, previousRef)
		}
		previousRef = ref
		byRef[ref] = entry
	}
	return byRef
}

func h264RealVectorUpstreamFATEInventoryByRef(t *testing.T, entries []h264RealVectorUpstreamInventoryRef) map[string]h264RealVectorUpstreamInventoryRef {
	t.Helper()
	var fateEntries []h264RealVectorUpstreamInventoryRef
	for _, entry := range entries {
		if entry.Source == h264RealVectorFFmpegFATEInventorySource {
			fateEntries = append(fateEntries, entry)
		}
	}
	return h264RealVectorUpstreamInventoryByRef(t, fateEntries)
}

func validateH264CorpusEntry(t *testing.T, entry h264CorpusEntry) {
	t.Helper()
	if entry.ID == "" || entry.Path == "" && entry.URL == "" {
		t.Fatalf("entry id and path or url must be set: %+v", entry)
	}
	switch entry.Extract {
	case "", "h264-annexb":
	default:
		t.Fatalf("%s: extract = %q, want h264-annexb or empty", entry.ID, entry.Extract)
	}
	if entry.Extract != "" && entry.SourceMD5 == "" {
		t.Fatalf("%s: extracted entries need source_md5", entry.ID)
	}
	if entry.Format != "annexb" {
		t.Fatalf("%s: format = %q, want annexb", entry.ID, entry.Format)
	}
	if len(entry.Surfaces) == 0 {
		t.Fatalf("%s: surfaces must be non-empty", entry.ID)
	}
	for _, surface := range entry.Surfaces {
		switch surface {
		case "annexb", "avc", "avc4", "configured-avc", "configured-avc4", "configured-samples", "auto":
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
	case "metadata-ok":
		if entry.BitstreamMD5 == "" {
			t.Fatalf("%s: metadata-ok entries need bitstream_md5", entry.ID)
		}
		if entry.FrameCount <= 0 {
			t.Fatalf("%s: frame_count must be positive", entry.ID)
		}
		validateH264CorpusFrameGroups(t, entry)
	case "decode-error":
		if entry.BitstreamMD5 == "" || entry.ExpectedError == "" {
			t.Fatalf("%s: decode-error entries need bitstream_md5 and expected_error", entry.ID)
		}
	case "unsupported":
		if len(entry.GuardTags) == 0 {
			t.Fatalf("%s: unsupported entries must name guard_tags", entry.ID)
		}
		if entry.ExpectedError != "" && entry.ExpectedError != "ErrUnsupported" {
			t.Fatalf("%s: expected_error = %q, want ErrUnsupported", entry.ID, entry.ExpectedError)
		}
	default:
		t.Fatalf("%s: expect = %q, want decode-ok, metadata-ok, decode-error, or unsupported", entry.ID, entry.Expect)
	}
}

func validateH264CorpusFrameGroups(t *testing.T, entry h264CorpusEntry) {
	t.Helper()
	if len(entry.FrameGroups) == 0 {
		t.Fatalf("%s: metadata-ok entries need frame_groups", entry.ID)
	}
	wantStart := 0
	for i, group := range entry.FrameGroups {
		if group.Start != wantStart {
			t.Fatalf("%s: frame_groups[%d].start = %d, want %d", entry.ID, i, group.Start, wantStart)
		}
		if group.Count <= 0 {
			t.Fatalf("%s: frame_groups[%d].count = %d, want positive", entry.ID, i, group.Count)
		}
		if group.Width <= 0 || group.Height <= 0 || group.PixFmt == "" || group.FrameSize <= 0 {
			t.Fatalf("%s: frame_groups[%d] needs positive width/height/frame_size and pix_fmt: %+v", entry.ID, i, group)
		}
		wantStart += group.Count
	}
	if wantStart != entry.FrameCount {
		t.Fatalf("%s: frame_groups cover %d frames, want frame_count %d", entry.ID, wantStart, entry.FrameCount)
	}
}

func materializeH264CorpusEntry(t *testing.T, manifest string, entry h264CorpusEntry) string {
	t.Helper()
	sourcePath := materializeH264CorpusSource(t, manifest, entry)
	if entry.Extract == "" {
		return sourcePath
	}
	sourceData, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatalf("%s: read source %s: %v", entry.ID, sourcePath, err)
	}
	assertCorpusSourceMD5(t, entry, sourceData)
	return extractH264CorpusAnnexB(t, entry, sourcePath)
}

func materializeH264CorpusSource(t *testing.T, manifest string, entry h264CorpusEntry) string {
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

func extractH264CorpusAnnexB(t *testing.T, entry h264CorpusEntry, sourcePath string) string {
	t.Helper()
	if entry.Extract != "h264-annexb" {
		t.Fatalf("%s: unsupported extract mode %q", entry.ID, entry.Extract)
	}
	path := sourcePath + ".h264-annexb"
	if _, err := os.Stat(path); err == nil {
		return path
	}
	if os.Getenv("GOH264_CORPUS_FETCH") != "1" && os.Getenv("GOH264_CORPUS_EXTRACT") != "1" {
		t.Fatalf("%s: missing extracted %s; set GOH264_CORPUS_FETCH=1 or GOH264_CORPUS_EXTRACT=1 to derive it with FFmpeg", entry.ID, path)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("%s: create extract cache dir: %v", entry.ID, err)
	}
	tmp := path + ".tmp"
	os.Remove(tmp)
	if err := runH264CorpusAnnexBExtract(entry, sourcePath, tmp, true); err != nil {
		if retryErr := runH264CorpusAnnexBExtract(entry, sourcePath, tmp, false); retryErr != nil {
			os.Remove(tmp)
			t.Fatalf("%s: extract Annex B from %s: with h264_mp4toannexb: %v; without bitstream filter: %v", entry.ID, sourcePath, err, retryErr)
		}
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		t.Fatalf("%s: install extracted %s: %v", entry.ID, path, err)
	}
	return path
}

func runH264CorpusAnnexBExtract(entry h264CorpusEntry, sourcePath string, outputPath string, withBitstreamFilter bool) error {
	bin := os.Getenv("GOH264_FFMPEG_BIN")
	if bin == "" {
		bin = "ffmpeg"
	}
	args := []string{"-nostdin", "-v", "error", "-y", "-i", sourcePath, "-map", "0:v:0", "-c:v", "copy"}
	if withBitstreamFilter {
		args = append(args, "-bsf:v", "h264_mp4toannexb")
	}
	args = append(args, "-f", "h264", outputPath)
	cmd := exec.Command(bin, args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
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
		return decodeH264CorpusAVCSurface(t, entry, data, []int{2, 3, 4})
	case "avc4":
		return decodeH264CorpusAVCSurface(t, entry, data, []int{4})
	case "configured-avc":
		return decodeH264CorpusConfiguredAVCSurface(t, entry, data, []int{2, 3, 4})
	case "configured-avc4":
		return decodeH264CorpusConfiguredAVCSurface(t, entry, data, []int{4})
	case "configured-samples":
		return decodeH264CorpusConfiguredSamples(t, entry, data, false, []int{2, 3, 4})
	case "auto":
		return decodeH264CorpusConfiguredSamples(t, entry, data, true, []int{2, 3, 4})
	}
	return nil, fmt.Errorf("unsupported corpus surface %q", surface)
}

func decodeH264CorpusAVCSurface(t *testing.T, entry h264CorpusEntry, data []byte, nalLengthSizes []int) ([]*Frame, error) {
	t.Helper()
	var final []*Frame
	for _, nalLengthSize := range nalLengthSizes {
		frames, err := NewDecoder().DecodeAVCFrames(annexBToAVC(t, data, nalLengthSize), nalLengthSize)
		if err != nil {
			return nil, fmt.Errorf("nal length size %d: %w", nalLengthSize, err)
		}
		if entry.Expect == "decode-ok" {
			assertH264CorpusFrames(t, entry, frames)
		}
		final = frames
	}
	return final, nil
}

func decodeH264CorpusConfiguredAVCSurface(t *testing.T, entry h264CorpusEntry, data []byte, nalLengthSizes []int) ([]*Frame, error) {
	t.Helper()
	var final []*Frame
	for _, nalLengthSize := range nalLengthSizes {
		config, packet := annexBToAVCConfigAndPacket(t, data, nalLengthSize)
		frames, err := NewDecoder().DecodeAVCCFrames(config, packet)
		if err != nil {
			return nil, fmt.Errorf("nal length size %d: %w", nalLengthSize, err)
		}
		if entry.Expect == "decode-ok" {
			assertH264CorpusFrames(t, entry, frames)
		}
		final = frames
	}
	return final, nil
}

func decodeH264CorpusConfiguredSamples(t *testing.T, entry h264CorpusEntry, data []byte, auto bool, nalLengthSizes []int) ([]*Frame, error) {
	t.Helper()
	var final []*Frame
	for _, nalLengthSize := range nalLengthSizes {
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
		} else if _, err := dec.ConfigureAVCC(config); err != nil {
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
		failH264CorpusOracle(t, entry, fmt.Sprintf("bitstream_md5 = %s, want %s", got, entry.BitstreamMD5))
	}
}

func assertCorpusSourceMD5(t *testing.T, entry h264CorpusEntry, data []byte) {
	t.Helper()
	if entry.SourceMD5 == "" {
		return
	}
	sum := md5.Sum(data)
	if got := hex.EncodeToString(sum[:]); got != entry.SourceMD5 {
		failH264CorpusOracle(t, entry, fmt.Sprintf("source_md5 = %s, want %s", got, entry.SourceMD5))
	}
}

func assertH264CorpusFrames(t *testing.T, entry h264CorpusEntry, frames []*Frame) {
	t.Helper()
	if len(frames) != entry.FrameCount {
		failH264CorpusOracle(t, entry, fmt.Sprintf("frames = %d, want %d", len(frames), entry.FrameCount))
	}
	rawHash := md5.New()
	var total int
	for i, frame := range frames {
		pixFmt, err := frame.RawPixelFormat()
		if err != nil {
			failH264CorpusOracle(t, entry, fmt.Sprintf("frame[%d] pix_fmt: %v", i, err))
		}
		if pixFmt != entry.PixFmt {
			failH264CorpusOracle(t, entry, fmt.Sprintf("frame[%d] pix_fmt = %s, want %s", i, pixFmt, entry.PixFmt))
		}
		raw, err := frame.AppendRawYUVBytesLE(nil)
		if err != nil {
			failH264CorpusOracle(t, entry, fmt.Sprintf("frame[%d] raw yuv: %v", i, err))
		}
		if len(raw) != entry.FrameSize {
			failH264CorpusOracle(t, entry, fmt.Sprintf("frame[%d] raw size = %d, want %d", i, len(raw), entry.FrameSize))
		}
		sum := md5.Sum(raw)
		if len(entry.FrameMD5) != 0 {
			if got := hex.EncodeToString(sum[:]); got != entry.FrameMD5[i] {
				failH264CorpusOracle(t, entry, fmt.Sprintf("frame[%d] md5 = %s, want %s", i, got, entry.FrameMD5[i]))
			}
		}
		if _, err := rawHash.Write(raw); err != nil {
			failH264CorpusOracle(t, entry, fmt.Sprintf("frame[%d] raw hash: %v", i, err))
		}
		total += len(raw)
	}
	if total != entry.FrameCount*entry.FrameSize {
		failH264CorpusOracle(t, entry, fmt.Sprintf("raw total = %d, want %d", total, entry.FrameCount*entry.FrameSize))
	}
	if got := hex.EncodeToString(rawHash.Sum(nil)); got != entry.RawVideoMD5 {
		failH264CorpusOracle(t, entry, fmt.Sprintf("rawvideo md5 = %s, want %s", got, entry.RawVideoMD5))
	}
}

func assertH264CorpusFrameMetadata(t *testing.T, entry h264CorpusEntry, frames []*Frame) {
	t.Helper()
	matches, detail := h264CorpusFramesMatchMetadata(entry, frames)
	if !matches {
		failH264CorpusOracle(t, entry, detail)
	}
}

func h264CorpusAnnexBMatchesExpectedOracle(t *testing.T, entry h264CorpusEntry, data []byte) (bool, string) {
	t.Helper()
	switch entry.Expect {
	case "decode-ok":
		return h264CorpusAnnexBMatchesOracle(t, entry, data)
	case "metadata-ok":
		return h264CorpusAnnexBMatchesMetadata(t, entry, data)
	case "decode-error":
		return h264CorpusAnnexBMatchesDecodeError(t, entry, data)
	default:
		return false, fmt.Sprintf("expect = %q, want decode-ok, metadata-ok, or decode-error", entry.Expect)
	}
}

func h264CorpusAnnexBMatchesDecodeError(t *testing.T, entry h264CorpusEntry, data []byte) (bool, string) {
	t.Helper()
	frames, err := NewDecoder().DecodeAnnexBFrames(data)
	if err == nil {
		return false, fmt.Sprintf("decode succeeded with %d frames, want error containing %q", len(frames), entry.ExpectedError)
	}
	if !h264CorpusDecodeErrorMatches(entry, err) {
		return false, fmt.Sprintf("decode error: %v, want containing %q", err, entry.ExpectedError)
	}
	return true, fmt.Sprintf("matched expected decode error: %v", err)
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

func h264CorpusAnnexBMatchesMetadata(t *testing.T, entry h264CorpusEntry, data []byte) (bool, string) {
	t.Helper()
	frames, err := NewDecoder().DecodeAnnexBFrames(data)
	if err != nil {
		return false, fmt.Sprintf("decode error: %v", err)
	}
	return h264CorpusFramesMatchMetadata(entry, frames)
}

func h264CorpusFramesMatchMetadata(entry h264CorpusEntry, frames []*Frame) (bool, string) {
	if len(frames) != entry.FrameCount {
		return false, fmt.Sprintf("frames = %d, want %d", len(frames), entry.FrameCount)
	}
	for groupIndex, group := range entry.FrameGroups {
		for offset := 0; offset < group.Count; offset++ {
			frameIndex := group.Start + offset
			if frameIndex < 0 || frameIndex >= len(frames) {
				return false, fmt.Sprintf("frame_groups[%d] index %d outside %d frames", groupIndex, frameIndex, len(frames))
			}
			frame := frames[frameIndex]
			if frame.Width != group.Width || frame.Height != group.Height {
				return false, fmt.Sprintf("frame[%d] size = %dx%d, want %dx%d", frameIndex, frame.Width, frame.Height, group.Width, group.Height)
			}
			pixFmt, err := frame.RawPixelFormat()
			if err != nil {
				return false, fmt.Sprintf("frame[%d] pix_fmt: %v", frameIndex, err)
			}
			if pixFmt != group.PixFmt {
				return false, fmt.Sprintf("frame[%d] pix_fmt = %s, want %s", frameIndex, pixFmt, group.PixFmt)
			}
			raw, err := frame.AppendRawYUVBytesLE(nil)
			if err != nil {
				return false, fmt.Sprintf("frame[%d] raw yuv: %v", frameIndex, err)
			}
			if len(raw) != group.FrameSize {
				return false, fmt.Sprintf("frame[%d] raw size = %d, want %d", frameIndex, len(raw), group.FrameSize)
			}
		}
	}
	return true, fmt.Sprintf("matched frame metadata oracle (%d frames, %d groups)", len(frames), len(entry.FrameGroups))
}

func h264CorpusOracleFailureClass(detail string) string {
	detail = strings.ToLower(detail)
	switch {
	case detail == "":
		return ""
	case strings.Contains(detail, "decode") || strings.Contains(detail, "unsupported"):
		return "decode-error"
	case strings.HasPrefix(detail, "missing ") || strings.Contains(detail, "no such file"):
		return "input-missing"
	case strings.Contains(detail, "frames ="):
		return "frame-count-mismatch"
	case strings.Contains(detail, "pix_fmt"):
		return "pixel-format-mismatch"
	case strings.Contains(detail, "raw size") || strings.Contains(detail, "raw total"):
		return "raw-size-mismatch"
	case strings.Contains(detail, "source_md5"):
		return "source-md5-mismatch"
	case strings.Contains(detail, "bitstream_md5"):
		return "bitstream-md5-mismatch"
	case strings.Contains(detail, "rawvideo md5") || strings.Contains(detail, "md5 ="):
		return "raw-md5-mismatch"
	default:
		return "oracle-mismatch"
	}
}

func failH264CorpusOracle(t *testing.T, entry h264CorpusEntry, detail string) {
	t.Helper()
	t.Fatalf("%s: strict corpus failure: %s", entry.ID, h264CorpusFailureDetail(entry, detail))
}

func h264CorpusFailureDetail(entry h264CorpusEntry, detail string) string {
	return fmt.Sprintf("class=%s features=%s surfaces=%s source=%q detail=%s",
		h264CorpusOracleFailureClass(detail),
		h264CorpusMetadataList(entry.FeatureTags),
		h264CorpusMetadataList(entry.Surfaces),
		entry.Source,
		detail)
}

func h264CorpusMetadataList(values []string) string {
	if len(values) == 0 {
		return "(none)"
	}
	return strings.Join(values, ",")
}

func assertH264CorpusUnsupported(t *testing.T, entry h264CorpusEntry, err error) {
	t.Helper()
	if !errors.Is(err, ErrUnsupported) {
		t.Fatalf("%s: err = %v, want ErrUnsupported for guard tags %v", entry.ID, err, entry.GuardTags)
	}
}

func assertH264CorpusExpectedDecodeError(t *testing.T, entry h264CorpusEntry, err error) {
	t.Helper()
	if err == nil {
		t.Fatalf("%s: decode succeeded, want error containing %q", entry.ID, entry.ExpectedError)
	}
	if !h264CorpusDecodeErrorMatches(entry, err) {
		t.Fatalf("%s: err = %v, want error containing %q", entry.ID, err, entry.ExpectedError)
	}
}

func h264CorpusDecodeErrorMatches(entry h264CorpusEntry, err error) bool {
	if err == nil {
		return false
	}
	want := strings.ToLower(entry.ExpectedError)
	return want == "" || strings.Contains(strings.ToLower(err.Error()), want)
}
