// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

import "testing"

func TestCABACTableLayout(t *testing.T) {
	if len(h264CABACTables) != 512+4*2*64+4*64+63 {
		t.Fatalf("cabac table length = %d", len(h264CABACTables))
	}
	if h264CABACTables[h264NormShiftOffset] != 9 || h264CABACTables[h264NormShiftOffset+128] != 1 {
		t.Fatalf("norm shift spots = %d %d", h264CABACTables[0], h264CABACTables[128])
	}
	if h264CABACTables[h264LPSRangeOffset] != 128 || h264CABACTables[h264LPSRangeOffset+6] != 123 {
		t.Fatalf("lps spots = %d %d", h264CABACTables[h264LPSRangeOffset], h264CABACTables[h264LPSRangeOffset+6])
	}
	if h264CABACTables[h264MLPSStateOffset] != 127 || h264CABACTables[h264MLPSStateOffset+127] != 1 {
		t.Fatalf("mlps spots = %d %d", h264CABACTables[h264MLPSStateOffset], h264CABACTables[h264MLPSStateOffset+127])
	}
	if h264CABACTables[h264LastCoeffFlagOffset8x8Offset] != 0 || h264CABACTables[len(h264CABACTables)-1] != 8 {
		t.Fatalf("last coeff spots = %d %d", h264CABACTables[h264LastCoeffFlagOffset8x8Offset], h264CABACTables[len(h264CABACTables)-1])
	}
}

func TestInitCABACDecoderAlignedBranches(t *testing.T) {
	buf := []byte{0x2a, 0x40, 0x80, 0x11}

	aligned, err := initCABACDecoderAligned(buf, true)
	if err != nil {
		t.Fatalf("aligned init failed: %v", err)
	}
	if aligned.low != 0x2a<<18+0x40<<10+1<<9 || aligned.rng != 0x1fe || aligned.bytestream != 2 {
		t.Fatalf("aligned ctx = low=%#x range=%#x bytestream=%d", aligned.low, aligned.rng, aligned.bytestream)
	}

	unaligned, err := initCABACDecoderAligned(buf, false)
	if err != nil {
		t.Fatalf("unaligned init failed: %v", err)
	}
	if unaligned.low != 0x2a<<18+0x40<<10+0x80<<2+2 || unaligned.rng != 0x1fe || unaligned.bytestream != 3 {
		t.Fatalf("unaligned ctx = low=%#x range=%#x bytestream=%d", unaligned.low, unaligned.rng, unaligned.bytestream)
	}
}

func TestCABACPrimitiveSequence(t *testing.T) {
	c, err := initCABACDecoderAligned([]byte{0x2a, 0x40, 0x80, 0x11, 0x22, 0x33}, false)
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}
	state := uint8(92)

	if got := c.getCABAC(&state); got != 0 || state != 94 || c.low != 0x00a90202 || c.rng != 0x1e8 {
		t.Fatalf("getCABAC #1 got=%d state=%d low=%#x range=%#x", got, state, c.low, c.rng)
	}
	if got := c.getCABAC(&state); got != 0 || state != 96 || c.low != 0x00a90202 || c.rng != 0x1d3 {
		t.Fatalf("getCABAC #2 got=%d state=%d low=%#x range=%#x", got, state, c.low, c.rng)
	}
	if got := c.getCABACBypass(); got != 0 || c.low != 0x01520404 {
		t.Fatalf("bypass got=%d low=%#x", got, c.low)
	}
	if got := c.getCABACBypassSign(-3); got != 3 || c.low != 0x02a40808 {
		t.Fatalf("bypass sign got=%d low=%#x", got, c.low)
	}
}

func TestInitH264CABACStates(t *testing.T) {
	iStates, err := initH264CABACStates(PictureTypeI, 0, 26, 8)
	if err != nil {
		t.Fatalf("init I states failed: %v", err)
	}
	if iStates[0] != 92 || iStates[60] != 44 || iStates[276] != 124 || iStates[1023] != 29 {
		t.Fatalf("I state spots = %d %d %d %d", iStates[0], iStates[60], iStates[276], iStates[1023])
	}

	pbStates, err := initH264CABACStates(PictureTypeP, 2, 31, 10)
	if err != nil {
		t.Fatalf("init PB states failed: %v", err)
	}
	if pbStates[0] != 110 || pbStates[11] != 26 || pbStates[60] != 44 || pbStates[399] != 12 {
		t.Fatalf("PB state spots = %d %d %d %d", pbStates[0], pbStates[11], pbStates[60], pbStates[399])
	}
}
