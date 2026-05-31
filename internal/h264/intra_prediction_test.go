// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

import "testing"

func TestPredIntraModeUsesMinimumOrDC(t *testing.T) {
	var cache [h264IntraPredModeCacheSize]int8
	base := int(h264Scan8[5])
	cache[base-1] = intraPredHorDown
	cache[base-8] = intraPredDiagDownRight

	got, err := predIntraMode(&cache, 5)
	if err != nil {
		t.Fatal(err)
	}
	if got != intraPredDiagDownRight {
		t.Fatalf("pred mode = %d, want %d", got, intraPredDiagDownRight)
	}

	cache[base-1] = -1
	got, err = predIntraMode(&cache, 5)
	if err != nil {
		t.Fatal(err)
	}
	if got != intraPredDC {
		t.Fatalf("unavailable pred mode = %d, want DC", got)
	}
}

func TestCheckIntra4x4PredModeCacheRewritesAvailableDCModes(t *testing.T) {
	var cache [h264IntraPredModeCacheSize]int8
	for i := range cache {
		cache[i] = intraPredDC
	}

	if err := checkIntra4x4PredModeCache(&cache, 0, 0); err != nil {
		t.Fatal(err)
	}
	if cache[h264Scan8[0]] != intraPredDC128 {
		t.Fatalf("top-left unavailable mode = %d, want dc128", cache[h264Scan8[0]])
	}
	if cache[h264Scan8[1]] != intraPredLeftDC {
		t.Fatalf("top-unavailable mode = %d, want left-dc", cache[h264Scan8[1]])
	}
}

func TestCheckIntra4x4PredModeCacheRejectsUnavailableDirectionalModes(t *testing.T) {
	var cache [h264IntraPredModeCacheSize]int8
	cache[h264Scan8[0]] = intraPredVertical
	if err := checkIntra4x4PredModeCache(&cache, 0, 0xffff); err != ErrInvalidData {
		t.Fatalf("top unavailable err = %v, want ErrInvalidData", err)
	}

	cache = [h264IntraPredModeCacheSize]int8{}
	cache[h264Scan8[0]] = intraPredHorizontal
	if err := checkIntra4x4PredModeCache(&cache, 0xffff, 0); err != ErrInvalidData {
		t.Fatalf("left unavailable err = %v, want ErrInvalidData", err)
	}
}

func TestCheckIntraPredMode(t *testing.T) {
	got, err := checkIntraPredMode(intraPred8x8DC, 0, 0, true)
	if err != nil {
		t.Fatal(err)
	}
	if got != intraPredDC1288x8 {
		t.Fatalf("chroma dc with no top/left = %d, want dc128", got)
	}

	got, err = checkIntraPredMode(intraPred8x8DC, 0, 0x8080, true)
	if err != nil {
		t.Fatal(err)
	}
	if got != intraPred8x8LeftDC {
		t.Fatalf("no-top chroma dc = %d, want left-dc", got)
	}

	if _, err := checkIntraPredMode(intraPred8x8Vertical, 0, 0x8080, false); err != ErrInvalidData {
		t.Fatalf("unavailable vertical err = %v, want ErrInvalidData", err)
	}
	if _, err := checkIntraPredMode(4, 0xffff, 0xffff, false); err != ErrInvalidData {
		t.Fatalf("out-of-range err = %v, want ErrInvalidData", err)
	}
}
