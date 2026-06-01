// SPDX-License-Identifier: LGPL-2.1-or-later

package main

import "testing"

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
