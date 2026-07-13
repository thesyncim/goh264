// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

import (
	"reflect"
	"testing"
)

type scriptedCABACSource struct {
	bits         []int
	bypassBits   []int
	signs        []int32
	terms        []int
	pcm          []byte
	indexes      []int
	pcmReadSizes []int
}

func (s *scriptedCABACSource) get(idx int) int {
	s.indexes = append(s.indexes, idx)
	if len(s.bits) == 0 {
		panic("scripted CABAC bit exhausted")
	}
	bit := s.bits[0]
	s.bits = s.bits[1:]
	return bit
}

func (s *scriptedCABACSource) bypass() int {
	if len(s.bypassBits) == 0 {
		panic("scripted CABAC bypass bit exhausted")
	}
	bit := s.bypassBits[0]
	s.bypassBits = s.bypassBits[1:]
	return bit
}

func (s *scriptedCABACSource) bypassSign(val int32) int32 {
	if len(s.signs) == 0 {
		return val
	}
	sign := s.signs[0]
	s.signs = s.signs[1:]
	return sign
}

func (s *scriptedCABACSource) terminate() int {
	if len(s.terms) == 0 {
		panic("scripted CABAC terminate bit exhausted")
	}
	term := s.terms[0]
	s.terms = s.terms[1:]
	return term
}

func (s *scriptedCABACSource) intraPCMBytes(n int) ([]byte, error) {
	s.pcmReadSizes = append(s.pcmReadSizes, n)
	if n < 0 || len(s.pcm) < n {
		return nil, ErrInvalidData
	}
	pcm := s.pcm[:n]
	s.pcm = s.pcm[n:]
	return pcm, nil
}

func TestDecodeCABACMBType(t *testing.T) {
	t.Run("I intra4x4 neighbor context", func(t *testing.T) {
		src := &scriptedCABACSource{bits: []int{0}}
		mb, err := decodeCABACMBType(src, PictureTypeI, PictureTypeI, MBTypeIntra16x16, MBTypeIntraPCM)
		if err != nil {
			t.Fatalf("decode failed: %v", err)
		}
		if mb.MBType != MBTypeIntra4x4 || mb.CBP != -1 {
			t.Fatalf("mb = type %#x cbp %d", mb.MBType, mb.CBP)
		}
		wantIndexes(t, src, []int{5})
	})

	t.Run("P inter 8x8", func(t *testing.T) {
		src := &scriptedCABACSource{bits: []int{0, 0, 1}}
		mb, err := decodeCABACMBType(src, PictureTypeP, PictureTypeP, 0, 0)
		if err != nil {
			t.Fatalf("decode failed: %v", err)
		}
		if mb.MBType != h264PMBTypeInfo[3].Type || mb.PartitionCount != 4 {
			t.Fatalf("mb = type %#x partitions %d", mb.MBType, mb.PartitionCount)
		}
		wantIndexes(t, src, []int{14, 15, 16})
	})

	t.Run("P intra fallback", func(t *testing.T) {
		src := &scriptedCABACSource{bits: []int{1, 0}}
		mb, err := decodeCABACMBType(src, PictureTypeP, PictureTypeP, 0, 0)
		if err != nil {
			t.Fatalf("decode failed: %v", err)
		}
		if mb.MBType != MBTypeIntra4x4 || mb.CBP != -1 {
			t.Fatalf("mb = type %#x cbp %d", mb.MBType, mb.CBP)
		}
		wantIndexes(t, src, []int{14, 17})
	})

	t.Run("B direct unavailable neighbors", func(t *testing.T) {
		src := &scriptedCABACSource{bits: []int{0}}
		mb, err := decodeCABACMBType(src, PictureTypeB, PictureTypeB, 0, 0)
		if err != nil {
			t.Fatalf("decode failed: %v", err)
		}
		if mb.MBType != h264BMBTypeInfo[0].Type || mb.PartitionCount != 1 {
			t.Fatalf("mb = type %#x partitions %d", mb.MBType, mb.PartitionCount)
		}
		wantIndexes(t, src, []int{27})
	})

	t.Run("B L1 16x16", func(t *testing.T) {
		src := &scriptedCABACSource{bits: []int{1, 0, 1}}
		mb, err := decodeCABACMBType(src, PictureTypeB, PictureTypeB, 0, 0)
		if err != nil {
			t.Fatalf("decode failed: %v", err)
		}
		if mb.MBType != h264BMBTypeInfo[2].Type || mb.PartitionCount != 1 {
			t.Fatalf("mb = type %#x partitions %d", mb.MBType, mb.PartitionCount)
		}
		wantIndexes(t, src, []int{27, 30, 32})
	})
}

func TestDecodeCABACIntraMBType(t *testing.T) {
	src := &scriptedCABACSource{
		bits:  []int{1, 1, 1, 0, 0, 1},
		terms: []int{0},
	}
	raw := decodeCABACIntraMBType(src, 17, false, 0, 0)
	if raw != 18 {
		t.Fatalf("raw intra type = %d, want 18", raw)
	}
	wantIndexes(t, src, []int{17, 18, 19, 19, 20, 20})
}

func TestDecodeCABACMBIntra4x4PredMode(t *testing.T) {
	src := &scriptedCABACSource{bits: []int{0, 1, 0, 1}}
	if got := decodeCABACMBIntra4x4PredMode(src, 5); got != 6 {
		t.Fatalf("pred mode = %d, want 6", got)
	}
	wantIndexes(t, src, []int{68, 69, 69, 69})

	src = &scriptedCABACSource{bits: []int{1}}
	if got := decodeCABACMBIntra4x4PredMode(src, 3); got != 3 {
		t.Fatalf("pred mode = %d, want 3", got)
	}
	wantIndexes(t, src, []int{68})
}

func TestDecodeCABACCBP(t *testing.T) {
	src := &scriptedCABACSource{bits: []int{1, 0, 1, 1}}
	if got := decodeCABACMBCBPLuma(src, 0, 0); got != 13 {
		t.Fatalf("luma cbp = %d, want 13", got)
	}
	wantIndexes(t, src, []int{76, 75, 74, 75})

	src = &scriptedCABACSource{bits: []int{1, 0}}
	if got := decodeCABACMBCBPChroma(src, 32, 16); got != 1 {
		t.Fatalf("chroma cbp = %d, want 1", got)
	}
	wantIndexes(t, src, []int{80, 82})
}

func TestDecodeCABACSubMBTypes(t *testing.T) {
	src := &scriptedCABACSource{bits: []int{1}}
	raw, info := decodeCABACPSubMBType(src)
	if raw != 0 || info != h264PSubMBTypeInfo[0] {
		t.Fatalf("P sub type raw=%d info=%+v", raw, info)
	}
	wantIndexes(t, src, []int{21})

	src = &scriptedCABACSource{bits: []int{1, 0, 1}}
	raw, info = decodeCABACBSubMBType(src)
	if raw != 2 || info != h264BSubMBTypeInfo[2] {
		t.Fatalf("B sub type raw=%d info=%+v", raw, info)
	}
	wantIndexes(t, src, []int{36, 37, 39})
}

func TestDecodeCABACMBRefAndMVD(t *testing.T) {
	src := &scriptedCABACSource{bits: []int{1, 1, 0}}
	if got := decodeCABACMBRef(src, PictureTypeP, 1, 0, 0, 0); got != 2 {
		t.Fatalf("ref = %d, want 2", got)
	}
	wantIndexes(t, src, []int{55, 58, 59})

	src = &scriptedCABACSource{bits: []int{0}}
	mvd, mvda, err := decodeCABACMBMVD(src, 40, 2)
	if err != nil {
		t.Fatalf("mvd zero failed: %v", err)
	}
	if mvd != 0 || mvda != 0 {
		t.Fatalf("mvd=%d mvda=%d, want 0/0", mvd, mvda)
	}
	wantIndexes(t, src, []int{40})

	src = &scriptedCABACSource{
		bits:  []int{1, 1, 1, 0},
		signs: []int32{-3},
	}
	mvd, mvda, err = decodeCABACMBMVD(src, 40, 34)
	if err != nil {
		t.Fatalf("mvd small failed: %v", err)
	}
	if mvd != -3 || mvda != 3 {
		t.Fatalf("mvd=%d mvda=%d, want -3/3", mvd, mvda)
	}
	wantIndexes(t, src, []int{42, 43, 44, 45})
}

func TestCABACSyntaxDecoderPrimitivesMatchContext(t *testing.T) {
	buf := make([]byte, 512)
	for i := range buf {
		buf[i] = byte(i*73 + 19)
	}
	fastContext, err := initCABACDecoder(buf)
	if err != nil {
		t.Fatal(err)
	}
	oracleContext := fastContext
	fastState, err := initH264CABACStates(PictureTypeP, 1, 27, 8)
	if err != nil {
		t.Fatal(err)
	}
	oracleState := fastState
	fast := cabacSyntaxDecoder{cabac: &fastContext, state: &fastState}

	for i := 0; i < 256; i++ {
		idx := (i * 37) % 400
		if got, want := fast.get(idx), oracleContext.getCABAC(&oracleState[idx]); got != want {
			t.Fatalf("get step %d index %d = %d, want %d", i, idx, got, want)
		}
		if i%5 == 0 {
			if got, want := fast.bypass(), oracleContext.getCABACBypass(); got != want {
				t.Fatalf("bypass step %d = %d, want %d", i, got, want)
			}
		}
		if i%7 == 0 {
			val := int32(-1 - i%11)
			if got, want := fast.bypassSign(val), oracleContext.getCABACBypassSign(val); got != want {
				t.Fatalf("bypass sign step %d = %d, want %d", i, got, want)
			}
		}
		if fastContext.low != oracleContext.low || fastContext.rng != oracleContext.rng || fastContext.bytestream != oracleContext.bytestream || fastState != oracleState {
			t.Fatalf("state diverged at step %d", i)
		}
	}
}

func TestDecodeCABACQScaleDiff(t *testing.T) {
	src := &scriptedCABACSource{bits: []int{0}}
	qscale, diff, err := decodeCABACQScaleDiff(src, 26, 0, 51)
	if err != nil {
		t.Fatalf("zero qscale diff failed: %v", err)
	}
	if qscale != 26 || diff != 0 {
		t.Fatalf("qscale=%d diff=%d, want 26/0", qscale, diff)
	}
	wantIndexes(t, src, []int{60})

	src = &scriptedCABACSource{bits: []int{1, 1, 0}}
	qscale, diff, err = decodeCABACQScaleDiff(src, 51, 4, 51)
	if err != nil {
		t.Fatalf("wrapped negative qscale diff failed: %v", err)
	}
	if qscale != 50 || diff != -1 {
		t.Fatalf("qscale=%d diff=%d, want 50/-1", qscale, diff)
	}
	wantIndexes(t, src, []int{61, 62, 63})

	src = &scriptedCABACSource{bits: []int{1, 0}}
	qscale, diff, err = decodeCABACQScaleDiff(src, 51, 0, 51)
	if err != nil {
		t.Fatalf("wrapped positive qscale diff failed: %v", err)
	}
	if qscale != 0 || diff != 1 {
		t.Fatalf("qscale=%d diff=%d, want 0/1", qscale, diff)
	}
	wantIndexes(t, src, []int{60, 62})
}

func TestCABACMVDContext(t *testing.T) {
	cases := []struct {
		amvd int
		want int
	}{
		{0, 40},
		{2, 40},
		{3, 41},
		{32, 41},
		{33, 42},
	}
	for _, tc := range cases {
		if got := cabacMVDContext(40, tc.amvd); got != tc.want {
			t.Fatalf("ctx for amvd %d = %d, want %d", tc.amvd, got, tc.want)
		}
	}
}

func wantIndexes(t *testing.T, src *scriptedCABACSource, want []int) {
	t.Helper()
	if !reflect.DeepEqual(src.indexes, want) {
		t.Fatalf("indexes = %v, want %v", src.indexes, want)
	}
}
