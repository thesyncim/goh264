// SPDX-License-Identifier: LGPL-2.1-or-later

package goh264_test

import (
	"fmt"
	"sort"
	"strings"
	"testing"
)

func TestH264DecoderTDDContractClassifiesEveryImportedPublicVector(t *testing.T) {
	manifest := readH264CorpusManifest(t, defaultH264RealVectorManifest)
	failures := readH264CorpusManifest(t, defaultH264RealVectorFailureManifest)
	exclusions := h264RealVectorExclusionsByRef(t, readH264RealVectorExclusions(t, defaultH264RealVectorExclusionManifest))
	inventory := h264RealVectorUpstreamInventoryByRef(t, readH264RealVectorUpstreamInventory(t, defaultH264RealVectorUpstreamInventory))
	failureByID := h264CorpusFailureLedgerByID(t, manifest, failures)

	if len(inventory) == 0 {
		t.Fatal("public-vector inventory is empty")
	}
	if len(manifest) == 0 {
		t.Fatal("public-vector manifest is empty")
	}

	manifestByID := make(map[string]h264CorpusEntry, len(manifest))
	manifestByRef := make(map[string]h264CorpusEntry, len(manifest))
	for _, entry := range manifest {
		validateH264CorpusEntry(t, entry)
		if entry.Source == "" || len(entry.FeatureTags) == 0 {
			t.Fatalf("%s: public-vector rows need source and feature_tags", entry.ID)
		}
		if previous, ok := manifestByID[entry.ID]; ok {
			t.Fatalf("%s: duplicate manifest id: previous=%+v current=%+v", entry.ID, previous, entry)
		}
		manifestByID[entry.ID] = entry

		refs := h264RealVectorRefsForEntry(entry)
		if len(refs) == 0 {
			t.Fatalf("%s: public-vector row must map back to an imported ref", entry.ID)
		}
		for _, ref := range refs {
			if previous, ok := manifestByRef[ref]; ok && previous.ID != entry.ID {
				t.Fatalf("%s: duplicate imported ref in manifest: previous=%s current=%s", ref, previous.ID, entry.ID)
			}
			manifestByRef[ref] = entry
		}
	}

	for _, failure := range failures {
		validateH264CorpusKnownFailure(t, failure)
		manifestEntry, ok := manifestByID[failure.ID]
		if !ok {
			t.Fatalf("%s: known-red row is missing from %s", failure.ID, defaultH264RealVectorManifest)
		}
		if refs := h264RealVectorRefsForEntry(manifestEntry); len(refs) == 0 {
			t.Fatalf("%s: known-red row does not map back to an imported ref", failure.ID)
		}
	}

	var missing, outsideInventory, excludedButExecutable []string
	for ref := range inventory {
		if _, ok := manifestByRef[ref]; ok {
			continue
		}
		if _, ok := exclusions[ref]; ok {
			continue
		}
		missing = append(missing, ref)
	}
	for ref, entry := range manifestByRef {
		if _, ok := inventory[ref]; !ok {
			outsideInventory = append(outsideInventory, fmt.Sprintf("%s (%s)", ref, entry.ID))
		}
	}
	for ref, exclusion := range exclusions {
		if entry, ok := manifestByRef[ref]; ok {
			excludedButExecutable = append(excludedButExecutable, fmt.Sprintf("%s (excluded=%s manifest=%s)", ref, exclusion.Reason, entry.ID))
		}
	}

	sort.Strings(missing)
	sort.Strings(outsideInventory)
	sort.Strings(excludedButExecutable)
	if len(missing) != 0 || len(outsideInventory) != 0 || len(excludedButExecutable) != 0 {
		t.Fatalf("public-vector TDD contract drifted\nmissing imported refs:\n%s\nmanifest refs outside inventory:\n%s\nexcluded refs also in manifest:\n%s",
			strings.Join(missing, "\n"),
			strings.Join(outsideInventory, "\n"),
			strings.Join(excludedButExecutable, "\n"))
	}

	t.Logf("public-vector TDD contract: imported=%d manifest=%d green=%d known_red=%d excluded=%d",
		len(inventory), len(manifest), len(manifest)-len(failureByID), len(failureByID), len(exclusions))
}

func h264RealVectorRefsForEntry(entry h264CorpusEntry) []string {
	refs := make(map[string]struct{}, 2)
	if entry.Path != "" {
		refs[h264CleanFATESamplePath(entry.Path)] = struct{}{}
	}
	if suffix := h264FATESuiteURLSuffix(entry.URL); suffix != "" {
		refs[suffix] = struct{}{}
	}
	out := make([]string, 0, len(refs))
	for ref := range refs {
		out = append(out, ref)
	}
	sort.Strings(out)
	return out
}
