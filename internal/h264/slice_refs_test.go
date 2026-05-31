// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

import "testing"

func TestDecodeRefPicMarkingIDRLongTerm(t *testing.T) {
	gb := bitReaderFromBits(t, "0 1")
	sh := &SliceHeader{NALType: NALIDRSlice}

	if err := decodeRefPicMarking(&gb, sh); err != nil {
		t.Fatal(err)
	}
	if sh.ExplicitRefMarking != 1 || sh.NBMMCO != 1 {
		t.Fatalf("marking = explicit %d nb %d", sh.ExplicitRefMarking, sh.NBMMCO)
	}
	if sh.MMCO[0] != (MMCO{Opcode: mmcoLong, LongArg: 0}) {
		t.Fatalf("mmco[0] = %+v", sh.MMCO[0])
	}
}

func TestDecodeRefPicMarkingLongTermOps(t *testing.T) {
	gb := bitReaderFromBits(t, `
		1
		00100 011 00101
		011 010
		00101 000010001
		00111 1
		1
	`)
	sh := &SliceHeader{
		NALType:          NALSlice,
		CurrPicNum:       8,
		MaxPicNum:        16,
		PictureStructure: PictureFrame,
	}

	if err := decodeRefPicMarking(&gb, sh); err != nil {
		t.Fatal(err)
	}
	if sh.ExplicitRefMarking != 1 || sh.NBMMCO != 4 {
		t.Fatalf("marking = explicit %d nb %d", sh.ExplicitRefMarking, sh.NBMMCO)
	}
	want := []MMCO{
		{Opcode: mmcoShort2Long, ShortPicNum: 5, LongArg: 4},
		{Opcode: mmcoLong2Unused, LongArg: 1},
		{Opcode: mmcoSetMaxLong, LongArg: 16},
		{Opcode: mmcoLong, LongArg: 0},
	}
	for i, w := range want {
		if sh.MMCO[i] != w {
			t.Fatalf("mmco[%d] = %+v, want %+v", i, sh.MMCO[i], w)
		}
	}
}

func TestDecodeRefPicListReorderingLongTerm(t *testing.T) {
	gb := bitReaderFromBits(t, "1 011 00101 00100")
	sh := &SliceHeader{
		SliceTypeNoS: PictureTypeP,
		ListCount:    1,
		RefCount:     [2]uint32{1, 0},
	}

	if err := decodeRefPicListReordering(&gb, sh); err != nil {
		t.Fatal(err)
	}
	if sh.NBRefModifications[0] != 1 {
		t.Fatalf("nb ref mods = %d", sh.NBRefModifications[0])
	}
	if sh.RefModifications[0][0] != (RefModification{Op: 2, Val: 4}) {
		t.Fatalf("ref mod = %+v", sh.RefModifications[0][0])
	}
}
