// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

import (
	"bytes"
	"errors"
	"testing"
	"unsafe"
)

func TestBitWriterExpGolombRoundTripsThroughReader(t *testing.T) {
	var bw BitWriter
	ueValues := []uint32{0, 1, 2, 3, 4, 5, 31, 255, 4095}
	for _, v := range ueValues {
		if err := bw.WriteUEGolomb(v); err != nil {
			t.Fatalf("WriteUEGolomb(%d): %v", v, err)
		}
	}
	seValues := []int32{0, 1, -1, 2, -2, 127, -128, 4096, -4096}
	for _, v := range seValues {
		if err := bw.WriteSEGolomb(v); err != nil {
			t.Fatalf("WriteSEGolomb(%d): %v", v, err)
		}
	}

	gb := newBitReader(bw.Bytes())
	gb.numBits = bw.BitLen()
	for i, want := range ueValues {
		got, err := gb.readUEGolombLong()
		if err != nil {
			t.Fatalf("read ue[%d]: %v", i, err)
		}
		if got != want {
			t.Fatalf("ue[%d] = %d, want %d", i, got, want)
		}
	}
	for i, want := range seValues {
		got, err := gb.readSEGolombLong()
		if err != nil {
			t.Fatalf("read se[%d]: %v", i, err)
		}
		if got != want {
			t.Fatalf("se[%d] = %d, want %d", i, got, want)
		}
	}
	if gb.bitsLeft() != 0 {
		t.Fatalf("bitsLeft = %d, want 0", gb.bitsLeft())
	}
}

func TestBitWriterRBSPTrailingBits(t *testing.T) {
	var bw BitWriter
	if err := bw.WriteBits(0b101, 3); err != nil {
		t.Fatal(err)
	}
	bw.WriteRBSPTrailingBits()
	if !bw.ByteAligned() {
		t.Fatal("writer is not byte-aligned after rbsp_trailing_bits")
	}
	if got, want := bw.Bytes(), []byte{0xb0}; !bytes.Equal(got, want) {
		t.Fatalf("rbsp = %x, want %x", got, want)
	}

	gb, err := newRBSPBitReader(bw.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	got, err := gb.readBits(3)
	if err != nil {
		t.Fatal(err)
	}
	if got != 0b101 || gb.bitsLeft() != 0 {
		t.Fatalf("payload bits = %b left=%d, want 101 left=0", got, gb.bitsLeft())
	}
}

func TestAppendEBSPRoundTripsThroughRBSPExtractor(t *testing.T) {
	rbsp := []byte{0x12, 0x00, 0x00, 0x00, 0x34, 0x00, 0x00, 0x01, 0x56, 0x00, 0x00, 0x02, 0x78, 0x00, 0x00, 0x03, 0x9a, 0x00, 0x00, 0x04}
	wantEBSP := []byte{0x12, 0x00, 0x00, 0x03, 0x00, 0x34, 0x00, 0x00, 0x03, 0x01, 0x56, 0x00, 0x00, 0x03, 0x02, 0x78, 0x00, 0x00, 0x03, 0x03, 0x9a, 0x00, 0x00, 0x04}

	ebsp := AppendEBSP(nil, rbsp)
	if !bytes.Equal(ebsp, wantEBSP) {
		t.Fatalf("ebsp = %x, want %x", ebsp, wantEBSP)
	}
	out, err := AppendRBSP(nil, ebsp)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out, rbsp) {
		t.Fatalf("round-trip rbsp = %x, want %x", out, rbsp)
	}
}

func TestAppendNALAnnexBAndAVCRoundTripThroughParsers(t *testing.T) {
	rbsp := []byte{0x05, 0x00, 0x00, 0x01, 0x80}

	annexB, err := AppendAnnexBNAL(nil, 3, NALSEI, rbsp)
	if err != nil {
		t.Fatal(err)
	}
	nals, err := SplitAnnexB(annexB)
	if err != nil {
		t.Fatal(err)
	}
	if len(nals) != 1 || nals[0].RefIDC != 3 || nals[0].Type != NALSEI || !bytes.Equal(nals[0].RBSP, rbsp) {
		t.Fatalf("annexb nals = %+v", nals)
	}
	if !bytes.Contains(nals[0].Raw, []byte{0x00, 0x00, 0x03, 0x01}) {
		t.Fatalf("raw NAL did not contain emulation-prevention bytes: %x", nals[0].Raw)
	}

	for _, nalLengthSize := range []int{1, 2, 3, 4} {
		avc, err := AppendAVCNAL(nil, nalLengthSize, 3, NALSEI, rbsp)
		if err != nil {
			t.Fatalf("nalLengthSize=%d: %v", nalLengthSize, err)
		}
		nals, err := SplitAVCC(avc, nalLengthSize)
		if err != nil {
			t.Fatalf("nalLengthSize=%d: %v", nalLengthSize, err)
		}
		if len(nals) != 1 || nals[0].RefIDC != 3 || nals[0].Type != NALSEI || !bytes.Equal(nals[0].RBSP, rbsp) {
			t.Fatalf("nalLengthSize=%d nals = %+v", nalLengthSize, nals)
		}
	}
}

func TestAppendAVCDecoderConfigurationRecordRoundTrip(t *testing.T) {
	spsRBSP := testEncoderBaselineSPSRBSP(t)
	ppsRBSP := testEncoderBaselinePPSRBSP(t)
	spsRaw, err := AppendNAL(nil, 3, NALSPS, spsRBSP)
	if err != nil {
		t.Fatal(err)
	}
	ppsRaw, err := AppendNAL(nil, 3, NALPPS, ppsRBSP)
	if err != nil {
		t.Fatal(err)
	}
	avcc, err := AppendAVCDecoderConfigurationRecord(nil, 66, 0xc0, 30, 4, [][]byte{spsRaw}, [][]byte{ppsRaw})
	if err != nil {
		t.Fatal(err)
	}
	if !IsAVCDecoderConfigurationRecord(avcc) {
		t.Fatalf("avcC was not recognized: %x", avcc)
	}
	cfg, err := DecodeAVCDecoderConfigurationRecord(avcc)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.NALLengthSize != 4 || cfg.FirstSPSID != 0 || cfg.SPS[0] == nil || cfg.PPS[0] == nil {
		t.Fatalf("cfg = %+v", cfg)
	}
	if cfg.SPS[0].ProfileIDC != 66 || cfg.SPS[0].LevelIDC != 30 || cfg.SPS[0].Width != 16 || cfg.SPS[0].Height != 16 {
		t.Fatalf("sps = %+v", cfg.SPS[0])
	}
	if cfg.PPS[0].PPSID != 0 || cfg.PPS[0].SPSID != 0 || cfg.PPS[0].RefCount[0] != 1 || cfg.PPS[0].RefCount[1] != 1 {
		t.Fatalf("pps = %+v", cfg.PPS[0])
	}
}

func TestBitWriterRejectsOutOfRangeSyntax(t *testing.T) {
	var bw BitWriter
	if err := bw.WriteBits(0, 33); !errors.Is(err, ErrInvalidData) {
		t.Fatalf("WriteBits err = %v, want ErrInvalidData", err)
	}
	if err := bw.WriteUEGolomb(^uint32(0)); !errors.Is(err, ErrInvalidData) {
		t.Fatalf("WriteUEGolomb err = %v, want ErrInvalidData", err)
	}
	if err := bw.WriteSEGolomb(-2147483648); !errors.Is(err, ErrInvalidData) {
		t.Fatalf("WriteSEGolomb err = %v, want ErrInvalidData", err)
	}
	if _, err := AppendNAL(nil, 4, NALSEI, nil); !errors.Is(err, ErrInvalidData) {
		t.Fatalf("AppendNAL ref err = %v, want ErrInvalidData", err)
	}
	if got, err := AppendAnnexBNAL([]byte{0xaa}, 4, NALSEI, nil); !errors.Is(err, ErrInvalidData) || !bytes.Equal(got, []byte{0xaa}) {
		t.Fatalf("AppendAnnexBNAL rollback got=%x err=%v, want original buffer and ErrInvalidData", got, err)
	}
	if _, err := AppendAVCNAL(nil, 0, 3, NALSEI, nil); !errors.Is(err, ErrInvalidData) {
		t.Fatalf("AppendAVCNAL length err = %v, want ErrInvalidData", err)
	}
	if _, err := AppendAVCDecoderConfigurationRecord(nil, 66, 0, 30, 4, nil, nil); !errors.Is(err, ErrInvalidData) {
		t.Fatalf("AppendAVCDecoderConfigurationRecord err = %v, want ErrInvalidData", err)
	}
}

func TestBitWriterRejectsOverflowedBitPosition(t *testing.T) {
	overflowed := NewBitWriter(fakeRBSPBytesLen(maxBitWriterByteLen + 1))
	if overflowed.ByteAligned() {
		t.Fatal("overflowed initial writer reports byte aligned")
	}
	if got := overflowed.BitLen(); got != ^uint32(0) {
		t.Fatalf("overflowed initial writer bit length = %d, want max uint32", got)
	}
	if err := overflowed.WriteBits(0, 1); !errors.Is(err, ErrInvalidData) {
		t.Fatalf("overflowed initial writer WriteBits err = %v, want ErrInvalidData", err)
	}
	if err := overflowed.WriteAlignedBytes([]byte{0}); !errors.Is(err, ErrInvalidData) {
		t.Fatalf("overflowed initial writer WriteAlignedBytes err = %v, want ErrInvalidData", err)
	}

	full := NewBitWriter(fakeRBSPBytesLen(maxBitWriterByteLen))
	if !full.ByteAligned() {
		t.Fatal("max-sized writer should still be byte aligned")
	}
	if err := full.WriteAlignedBytes([]byte{0}); !errors.Is(err, ErrInvalidData) {
		t.Fatalf("max-sized writer extension err = %v, want ErrInvalidData", err)
	}
	if err := full.WriteBits(0, 8); !errors.Is(err, ErrInvalidData) {
		t.Fatalf("max-sized writer bit extension err = %v, want ErrInvalidData", err)
	}
}

func TestAppendNALRejectsOverflowedEscapedSize(t *testing.T) {
	rbsp := fakeRBSPBytesLen(maxInt)
	prefix := []byte{0xaa}
	if got := AppendEBSP(prefix, rbsp); got != nil {
		t.Fatalf("AppendEBSP overflow got len=%d, want nil", len(got))
	}
	if got, err := AppendNAL(prefix, 3, NALSEI, rbsp); !errors.Is(err, ErrInvalidData) || !bytes.Equal(got, prefix) {
		t.Fatalf("AppendNAL overflow got len=%d err=%v, want original buffer and ErrInvalidData", len(got), err)
	}
	if got, err := AppendAnnexBNAL(prefix, 3, NALSEI, rbsp); !errors.Is(err, ErrInvalidData) || !bytes.Equal(got, prefix) {
		t.Fatalf("AppendAnnexBNAL overflow got len=%d err=%v, want original buffer and ErrInvalidData", len(got), err)
	}
	if _, err := makeNALBuffer(rbsp); !errors.Is(err, ErrInvalidData) {
		t.Fatalf("makeNALBuffer overflow err = %v, want ErrInvalidData", err)
	}
}

func TestAppendAVCPackagingRejectsOverflowedDestination(t *testing.T) {
	dst := fakeRBSPBytesLen(maxInt)
	if got, err := AppendAVCNAL(dst, 4, 3, NALSEI, nil); !errors.Is(err, ErrInvalidData) || len(got) != len(dst) {
		t.Fatalf("AppendAVCNAL overflow got len=%d err=%v, want original buffer and ErrInvalidData", len(got), err)
	}

	spsRaw, err := AppendNAL(nil, 3, NALSPS, testEncoderBaselineSPSRBSP(t))
	if err != nil {
		t.Fatal(err)
	}
	ppsRaw, err := AppendNAL(nil, 3, NALPPS, testEncoderBaselinePPSRBSP(t))
	if err != nil {
		t.Fatal(err)
	}
	if got, err := AppendAVCDecoderConfigurationRecord(dst, 66, 0xc0, 30, 4, [][]byte{spsRaw}, [][]byte{ppsRaw}); !errors.Is(err, ErrInvalidData) || len(got) != len(dst) {
		t.Fatalf("AppendAVCDecoderConfigurationRecord overflow got len=%d err=%v, want original buffer and ErrInvalidData", len(got), err)
	}
	nearFull := fakeRBSPBytesLen(maxInt - 6)
	if _, err := avcDecoderConfigurationRecordCapacity(len(nearFull), [][]byte{spsRaw}, [][]byte{ppsRaw}); !errors.Is(err, ErrInvalidData) {
		t.Fatalf("avcDecoderConfigurationRecordCapacity near-full error = %v, want ErrInvalidData", err)
	}
	if got, err := AppendAVCDecoderConfigurationRecord(nearFull, 66, 0xc0, 30, 4, [][]byte{spsRaw}, [][]byte{ppsRaw}); !errors.Is(err, ErrInvalidData) || len(got) != len(nearFull) {
		t.Fatalf("AppendAVCDecoderConfigurationRecord near-full overflow got len=%d err=%v, want original buffer and ErrInvalidData", len(got), err)
	}
	if got, err := appendAVCConfigRawNAL(dst, spsRaw, NALSPS); !errors.Is(err, ErrInvalidData) || len(got) != len(dst) {
		t.Fatalf("appendAVCConfigRawNAL overflow got len=%d err=%v, want original buffer and ErrInvalidData", len(got), err)
	}
}

func fakeRBSPBytesLen(n int) []byte {
	if n <= 0 {
		return nil
	}
	var b byte
	return fakeH264SliceLen(&b, n)
}

// fakeH264SliceLen preserves impossible slice lengths for overflow guards.
func fakeH264SliceLen[T any](ptr *T, n int) []T {
	h := struct {
		Data unsafe.Pointer
		Len  int
		Cap  int
	}{
		Data: unsafe.Pointer(ptr),
		Len:  n,
		Cap:  n,
	}
	return *(*[]T)(unsafe.Pointer(&h))
}

func testEncoderBaselineSPSRBSP(t *testing.T) []byte {
	t.Helper()
	var bw BitWriter
	if err := bw.WriteBits(66, 8); err != nil {
		t.Fatal(err)
	}
	if err := bw.WriteBits(0xc0, 8); err != nil {
		t.Fatal(err)
	}
	if err := bw.WriteBits(30, 8); err != nil {
		t.Fatal(err)
	}
	for _, v := range []uint32{
		0, // seq_parameter_set_id
		0, // log2_max_frame_num_minus4
		0, // pic_order_cnt_type
		0, // log2_max_pic_order_cnt_lsb_minus4
		1, // max_num_ref_frames
	} {
		if err := bw.WriteUEGolomb(v); err != nil {
			t.Fatal(err)
		}
	}
	bw.WriteBit(0) // gaps_in_frame_num_value_allowed_flag
	bw.WriteUEGolomb(0)
	bw.WriteUEGolomb(0)
	bw.WriteBit(1) // frame_mbs_only_flag
	bw.WriteBit(1) // direct_8x8_inference_flag
	bw.WriteBit(0) // frame_cropping_flag
	bw.WriteBit(0) // vui_parameters_present_flag
	bw.WriteRBSPTrailingBits()
	return bw.Bytes()
}

func testEncoderBaselinePPSRBSP(t *testing.T) []byte {
	t.Helper()
	var bw BitWriter
	for _, v := range []uint32{
		0, // pic_parameter_set_id
		0, // seq_parameter_set_id
	} {
		if err := bw.WriteUEGolomb(v); err != nil {
			t.Fatal(err)
		}
	}
	bw.WriteBit(0) // entropy_coding_mode_flag
	bw.WriteBit(0) // bottom_field_pic_order_in_frame_present_flag
	bw.WriteUEGolomb(0)
	bw.WriteUEGolomb(0)
	bw.WriteUEGolomb(0)
	bw.WriteBit(0) // weighted_pred_flag
	if err := bw.WriteBits(0, 2); err != nil {
		t.Fatal(err)
	}
	if err := bw.WriteSEGolomb(0); err != nil {
		t.Fatal(err)
	}
	if err := bw.WriteSEGolomb(0); err != nil {
		t.Fatal(err)
	}
	if err := bw.WriteSEGolomb(0); err != nil {
		t.Fatal(err)
	}
	bw.WriteBit(1) // deblocking_filter_control_present_flag
	bw.WriteBit(0) // constrained_intra_pred_flag
	bw.WriteBit(0) // redundant_pic_cnt_present_flag
	bw.WriteRBSPTrailingBits()
	return bw.Bytes()
}
