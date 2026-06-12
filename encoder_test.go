// SPDX-License-Identifier: LGPL-2.1-or-later

package goh264

import (
	"bytes"
	"errors"
	"testing"
	"unsafe"

	"github.com/thesyncim/goh264/internal/h264"
)

func TestAppendEncoderP16x16NoResidualMVDsUsesSliceLocalPrediction(t *testing.T) {
	for _, tt := range []struct {
		name              string
		firstMB           int
		macroblockCount   int
		macroblocksPerRow int
		mvs               []encoderP16x16MotionVector
		want              [][2]int32
	}{
		{
			name:              "full two-row frame",
			firstMB:           0,
			macroblockCount:   6,
			macroblocksPerRow: 3,
			want:              [][2]int32{{8, 0}, {}, {}, {}, {}, {}},
		},
		{
			name:              "mid-row slice",
			firstMB:           1,
			macroblockCount:   2,
			macroblocksPerRow: 3,
			want:              [][2]int32{{8, 0}, {}},
		},
		{
			name:              "narrow vertical frame",
			firstMB:           0,
			macroblockCount:   2,
			macroblocksPerRow: 1,
			want:              [][2]int32{{8, 0}, {}},
		},
		{
			name:              "slice crosses from row end",
			firstMB:           2,
			macroblockCount:   4,
			macroblocksPerRow: 3,
			want:              [][2]int32{{8, 0}, {8, 0}, {}, {}},
		},
		{
			name:              "mixed vectors use median prediction",
			firstMB:           0,
			macroblockCount:   6,
			macroblocksPerRow: 3,
			mvs: []encoderP16x16MotionVector{
				{x: 8, y: 0},
				{x: -8, y: 0},
				{x: 0, y: 8},
				{x: 0, y: -8},
				{x: 8, y: 8},
				{x: -8, y: -8},
			},
			want: [][2]int32{{8, 0}, {-16, 0}, {8, 8}, {0, -8}, {8, 8}, {-8, -16}},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			mvs := tt.mvs
			if len(mvs) == 0 {
				mvs = make([]encoderP16x16MotionVector, tt.firstMB+tt.macroblockCount)
				for i := range mvs {
					mvs[i] = encoderP16x16MotionVector{x: 8}
				}
			}
			got := appendEncoderP16x16NoResidualMVDs(nil, mvs, tt.firstMB, tt.macroblockCount, tt.macroblocksPerRow)
			if len(got) != len(tt.want) {
				t.Fatalf("len = %d, want %d", len(got), len(tt.want))
			}
			for i, want := range tt.want {
				if got[i].X != want[0] || got[i].Y != want[1] {
					t.Fatalf("mvd[%d] = {%d, %d}, want {%d, %d}", i, got[i].X, got[i].Y, want[0], want[1])
				}
			}
		})
	}
}

func TestEncoderAccessUnitOutputSizeRejectsOverflow(t *testing.T) {
	nals := []encoderRawNAL{
		{raw: fakeEncoderBytesLen(maxInt - 2)},
		{raw: fakeEncoderBytesLen(1)},
	}
	if _, err := encoderAccessUnitOutputSize(EncoderOutputAnnexB, nals); !errors.Is(err, ErrInvalidData) {
		t.Fatalf("encoderAccessUnitOutputSize overflow error = %v, want ErrInvalidData", err)
	}
}

func TestEncoderAccessUnitHelpersRejectOverflowedNALCount(t *testing.T) {
	nals := fakeEncoderRawNALLen(maxEncoderRawNALListLen + 1)
	if _, _, err := appendEncoderAccessUnit(nil, EncoderOutputAnnexB, nals); !errors.Is(err, ErrInvalidData) {
		t.Fatalf("appendEncoderAccessUnit NAL-count overflow error = %v, want ErrInvalidData", err)
	}
	if _, err := encoderAccessUnitOutputSize(EncoderOutputAnnexB, nals); !errors.Is(err, ErrInvalidData) {
		t.Fatalf("encoderAccessUnitOutputSize NAL-count overflow error = %v, want ErrInvalidData", err)
	}
}

func TestAppendEncoderAccessUnitRejectsOverflowedDestination(t *testing.T) {
	nals := []encoderRawNAL{{raw: []byte{byte(h264.NALSlice)}}}
	for _, format := range []EncoderOutputFormat{EncoderOutputAnnexB, EncoderOutputAVC, EncoderOutputRTP} {
		dst := fakeEncoderBytesLen(maxInt - 4)
		got, units, err := appendEncoderAccessUnit(dst, format, nals)
		if !errors.Is(err, ErrInvalidData) || len(got) != len(dst) || units != nil {
			t.Fatalf("format %v overflow got len=%d units=%v err=%v, want original buffer, nil units, ErrInvalidData", format, len(got), units, err)
		}
	}
}

func TestAppendEncoderAccessUnitRejectsEmptyNALWithoutMutation(t *testing.T) {
	nals := []encoderRawNAL{
		{raw: []byte{byte(h264.NALSPS)}, parameterSet: true},
		{},
	}
	for _, format := range []EncoderOutputFormat{EncoderOutputAnnexB, EncoderOutputAVC, EncoderOutputRTP} {
		dst := []byte{0xaa, 0xbb}
		before := append([]byte(nil), dst...)
		got, units, err := appendEncoderAccessUnit(dst, format, nals)
		if !errors.Is(err, ErrInvalidData) || len(got) != len(dst) || units != nil {
			t.Fatalf("format %v empty NAL got len=%d units=%v err=%v, want original buffer, nil units, ErrInvalidData", format, len(got), units, err)
		}
		if !bytes.Equal(dst, before) {
			t.Fatalf("format %v empty NAL mutated caller buffer: got %x want %x", format, dst, before)
		}
	}
}

func TestAppendEncoderAccessUnitRejectsUnknownFormatWithoutMutation(t *testing.T) {
	nals := []encoderRawNAL{{raw: []byte{byte(h264.NALSlice)}}}
	dst := []byte{0xcc, 0xdd}
	before := append([]byte(nil), dst...)
	got, units, err := appendEncoderAccessUnit(dst, EncoderOutputFormat(99), nals)
	if !errors.Is(err, ErrInvalidData) || len(got) != len(dst) || units != nil {
		t.Fatalf("unknown format got len=%d units=%v err=%v, want original buffer, nil units, ErrInvalidData", len(got), units, err)
	}
	if !bytes.Equal(dst, before) {
		t.Fatalf("unknown format mutated caller buffer: got %x want %x", dst, before)
	}
}

func TestEncoderRTPMode1StoragePlanRejectsOverflow(t *testing.T) {
	nals := []encoderRawNAL{
		{raw: fakeEncoderBytesLen(maxInt - 1)},
	}
	if _, _, err := encoderRTPMode1StoragePlan(nals, 3, false); !errors.Is(err, ErrInvalidData) {
		t.Fatalf("encoderRTPMode1StoragePlan overflow error = %v, want ErrInvalidData", err)
	}
}

func TestEncoderRTPMode1StoragePlanRejectsInvalidPayloadSize(t *testing.T) {
	nals := []encoderRawNAL{{raw: []byte{byte(h264.NALSlice), 0xaa, 0xbb, 0xcc}}}
	if packets, payload, err := encoderRTPMode1StoragePlan(nals, 2, false); !errors.Is(err, ErrInvalidData) || packets != 0 || payload != 0 {
		t.Fatalf("encoderRTPMode1StoragePlan invalid payload packets=%d payload=%d err=%v, want zero plan and ErrInvalidData", packets, payload, err)
	}
}

func TestEncoderRTPStorageHelpersRejectOverflowedNALCount(t *testing.T) {
	nals := fakeEncoderRawNALLen(maxEncoderRawNALListLen + 1)
	if _, err := encoderRawNALPayloadStorageSize(nals); !errors.Is(err, ErrInvalidData) {
		t.Fatalf("encoderRawNALPayloadStorageSize NAL-count overflow error = %v, want ErrInvalidData", err)
	}
	if _, _, err := encoderRTPMode1StoragePlan(nals, 3, false); !errors.Is(err, ErrInvalidData) {
		t.Fatalf("encoderRTPMode1StoragePlan NAL-count overflow error = %v, want ErrInvalidData", err)
	}
	if _, err := packetizeEncoderRTPSingleNAL(nals, 1, 0); !errors.Is(err, ErrInvalidData) {
		t.Fatalf("packetizeEncoderRTPSingleNAL NAL-count overflow error = %v, want ErrInvalidData", err)
	}
}

func TestEncoderRawNALPayloadStorageSizeRejectsEmptyNAL(t *testing.T) {
	nals := []encoderRawNAL{
		{raw: []byte{byte(h264.NALSPS)}, parameterSet: true},
		{},
	}
	if _, err := encoderRawNALPayloadStorageSize(nals); !errors.Is(err, ErrInvalidData) {
		t.Fatalf("encoderRawNALPayloadStorageSize empty NAL error = %v, want ErrInvalidData", err)
	}
}

func TestPacketizeEncoderRTPSingleNALRejectsStorageOverflow(t *testing.T) {
	nals := []encoderRawNAL{
		{raw: fakeEncoderBytesLen(maxInt - 4)},
		{raw: fakeEncoderBytesLen(1)},
	}
	if _, err := packetizeEncoderRTPSingleNAL(nals, maxInt, 0); !errors.Is(err, ErrInvalidData) {
		t.Fatalf("packetizeEncoderRTPSingleNAL storage overflow error = %v, want ErrInvalidData", err)
	}
}

func TestPacketizeEncoderRTPRejectsInvalidPayloadSizeWithoutPackets(t *testing.T) {
	nals := []encoderRawNAL{{raw: []byte{byte(h264.NALSlice)}}}
	if packets, err := packetizeEncoderRTPSingleNAL(nals, 0, 0); !errors.Is(err, ErrInvalidData) || packets != nil {
		t.Fatalf("packetizeEncoderRTPSingleNAL invalid payload packets=%v err=%v, want nil packets and ErrInvalidData", packets, err)
	}
	if packets, err := packetizeEncoderRTPMode1(nals, 2, 0, false); !errors.Is(err, ErrInvalidData) || packets != nil {
		t.Fatalf("packetizeEncoderRTPMode1 invalid payload packets=%v err=%v, want nil packets and ErrInvalidData", packets, err)
	}
}

func TestPacketizeEncoderRTPRejectsEmptyNALWithoutPackets(t *testing.T) {
	nals := []encoderRawNAL{
		{raw: []byte{byte(h264.NALSPS)}, parameterSet: true},
		{},
	}
	if packets, err := packetizeEncoderRTPSingleNAL(nals, 1200, 0); !errors.Is(err, ErrInvalidData) || packets != nil {
		t.Fatalf("packetizeEncoderRTPSingleNAL empty NAL packets=%v err=%v, want nil packets and ErrInvalidData", packets, err)
	}
	for _, stapa := range []bool{false, true} {
		packets, err := packetizeEncoderRTPMode1(nals, 1200, 0, stapa)
		if !errors.Is(err, ErrInvalidData) || packets != nil {
			t.Fatalf("packetizeEncoderRTPMode1 stapa=%v empty NAL packets=%v err=%v, want nil packets and ErrInvalidData", stapa, packets, err)
		}
	}
}

func TestPacketizeEncoderRTPMode1RejectsMalformedSTAPAWithoutPackets(t *testing.T) {
	for _, tt := range []struct {
		name string
		nals []encoderRawNAL
	}{
		{
			name: "empty",
			nals: []encoderRawNAL{
				{raw: []byte{byte(h264.NALSPS)}, parameterSet: true},
				{parameterSet: true},
			},
		},
		{
			name: "oversized",
			nals: []encoderRawNAL{
				{raw: fakeEncoderBytesLen(0x10000), parameterSet: true},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			packets, err := packetizeEncoderRTPMode1(tt.nals, 0x10010, 0, true)
			if !errors.Is(err, ErrInvalidData) || packets != nil {
				t.Fatalf("packetizeEncoderRTPMode1 malformed STAP-A packets=%v err=%v, want nil packets and ErrInvalidData", packets, err)
			}
		})
	}
}

func TestAppendEncoderSTAPARejectsOverflowedDestination(t *testing.T) {
	nals := []encoderRawNAL{
		{raw: []byte{byte(h264.NALSPS)}, parameterSet: true},
		{raw: []byte{byte(h264.NALPPS)}, parameterSet: true},
	}
	dst := fakeEncoderBytesLen(maxInt - 6)
	if got, count, err := appendEncoderSTAPA(dst, nals, 8); !errors.Is(err, ErrInvalidData) || count != 0 || len(got) != len(dst) {
		t.Fatalf("appendEncoderSTAPA overflow got len=%d count=%d err=%v, want original buffer, count 0, ErrInvalidData", len(got), count, err)
	}
}

func TestEncoderSTAPARejectsInvalidPayloadSizeWithoutMutation(t *testing.T) {
	nals := []encoderRawNAL{
		{raw: []byte{byte(h264.NALSPS)}, parameterSet: true},
		{raw: []byte{byte(h264.NALPPS)}, parameterSet: true},
	}
	if size, count, err := encoderSTAPASize(nals, 0); !errors.Is(err, ErrInvalidData) || size != 0 || count != 0 {
		t.Fatalf("encoderSTAPASize invalid payload size=%d count=%d err=%v, want zero plan and ErrInvalidData", size, count, err)
	}
	dst := []byte{0xaa, 0xbb}
	before := append([]byte(nil), dst...)
	got, count, err := appendEncoderSTAPA(dst, nals, 0)
	if !errors.Is(err, ErrInvalidData) || count != 0 || len(got) != len(dst) {
		t.Fatalf("appendEncoderSTAPA invalid payload got len=%d count=%d err=%v, want original buffer, count 0, ErrInvalidData", len(got), count, err)
	}
	if !bytes.Equal(dst, before) {
		t.Fatalf("appendEncoderSTAPA invalid payload mutated caller buffer: got %x want %x", dst, before)
	}
}

func TestEncoderSTAPASizeRejectsMalformedParameterSets(t *testing.T) {
	if _, _, err := encoderSTAPASize([]encoderRawNAL{
		{raw: []byte{byte(h264.NALSPS)}, parameterSet: true},
		{parameterSet: true},
	}, 1200); !errors.Is(err, ErrInvalidData) {
		t.Fatalf("encoderSTAPASize empty parameter-set NAL error = %v, want ErrInvalidData", err)
	}
	if _, _, err := encoderSTAPASize([]encoderRawNAL{
		{raw: fakeEncoderBytesLen(0x10000), parameterSet: true},
	}, 0x10010); !errors.Is(err, ErrInvalidData) {
		t.Fatalf("encoderSTAPASize oversized parameter-set NAL error = %v, want ErrInvalidData", err)
	}
}

func TestAppendEncoderSTAPARejectsMalformedParameterSetWithoutMutation(t *testing.T) {
	for _, tt := range []struct {
		name string
		nals []encoderRawNAL
	}{
		{
			name: "empty",
			nals: []encoderRawNAL{
				{raw: []byte{byte(h264.NALSPS)}, parameterSet: true},
				{parameterSet: true},
			},
		},
		{
			name: "oversized",
			nals: []encoderRawNAL{
				{raw: fakeEncoderBytesLen(0x10000), parameterSet: true},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dst := []byte{0xaa, 0xbb}
			before := append([]byte(nil), dst...)
			got, count, err := appendEncoderSTAPA(dst, tt.nals, 0x10010)
			if !errors.Is(err, ErrInvalidData) || count != 0 || len(got) != len(dst) {
				t.Fatalf("appendEncoderSTAPA malformed got len=%d count=%d err=%v, want original buffer, count 0, ErrInvalidData", len(got), count, err)
			}
			if !bytes.Equal(dst, before) {
				t.Fatalf("appendEncoderSTAPA malformed mutated caller buffer: got %x want %x", dst, before)
			}
		})
	}
}

func TestAppendEncoderSTAPADoesNotMutateWhenNotAggregating(t *testing.T) {
	dst := []byte{0xaa}
	nals := []encoderRawNAL{
		{raw: []byte{byte(h264.NALSPS)}, parameterSet: true},
		{raw: []byte{byte(h264.NALSlice)}},
	}
	got, count, err := appendEncoderSTAPA(dst, nals, 8)
	if err != nil {
		t.Fatalf("appendEncoderSTAPA one parameter set: %v", err)
	}
	if count != 1 || !bytes.Equal(got, dst) {
		t.Fatalf("appendEncoderSTAPA one parameter set got=%x count=%d, want original buffer and count 1", got, count)
	}
}

func TestEncoderReferenceHelpersRejectOverflowedGeometry(t *testing.T) {
	view := encoderFrameView{
		width:  maxInt/2 + 1,
		height: 16,
	}
	ref := encoderReferenceFrame{
		valid:  true,
		width:  view.width,
		height: view.height,
	}
	enc := &Encoder{
		cfg:       EncoderConfig{DeblockMode: EncoderDeblockDisabled},
		reference: ref,
	}
	if enc.referenceMatches(view) {
		t.Fatal("referenceMatches accepted overflowed geometry")
	}
	if got, ok := enc.p16x16NoResidualMotion(view, nil); ok || got != nil {
		t.Fatalf("p16x16NoResidualMotion = %v/%t, want nil/false", got, ok)
	}
	enc.storeReference(view)
	if enc.reference.valid {
		t.Fatal("storeReference kept overflowed geometry valid")
	}
}

func TestEncoderReferencePlaneSizesRejectOverflow(t *testing.T) {
	tests := []struct {
		name string
		view encoderFrameView
	}{
		{name: "luma", view: encoderFrameView{width: maxInt/2 + 1, height: 3}},
		{name: "chroma", view: encoderFrameView{width: maxInt, height: 4}},
		{name: "nonpositive", view: encoderFrameView{width: 0, height: 16}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, _, ok := encoderI420ReferencePlaneSizes(tt.view); ok {
				t.Fatalf("encoderI420ReferencePlaneSizes(%+v) ok, want false", tt.view)
			}
		})
	}
}

func TestEncoderNALBufferRejectsOverflow(t *testing.T) {
	if _, err := encoderNALBuffer(fakeEncoderBytesLen(maxInt)); !errors.Is(err, ErrInvalidData) {
		t.Fatalf("encoderNALBuffer overflow error = %v, want ErrInvalidData", err)
	}
}

func TestEncoderParameterSetsFromH264ClonesHeaderSurfaces(t *testing.T) {
	src := h264.EncoderParameterSets{
		SPS:                           []byte{0x67, 0x42, 0x00},
		PPS:                           []byte{0x68, 0xce},
		AnnexB:                        []byte{0x00, 0x00, 0x01, 0x67, 0x00, 0x00, 0x01, 0x68},
		AVCDecoderConfigurationRecord: []byte{0x01, 0x42, 0x00, 0x1f, 0xff},
	}
	got := encoderParameterSetsFromH264(src)
	if !bytes.Equal(got.SPS, src.SPS) ||
		!bytes.Equal(got.PPS, src.PPS) ||
		!bytes.Equal(got.AnnexB, src.AnnexB) ||
		!bytes.Equal(got.AVCDecoderConfigurationRecord, src.AVCDecoderConfigurationRecord) {
		t.Fatalf("cloned parameter sets = %+v, want source bytes", got)
	}

	src.SPS[0] = 0xff
	src.PPS[0] = 0xff
	src.AnnexB[3] = 0xff
	src.AVCDecoderConfigurationRecord[0] = 0xff
	got.SPS[1] = 0xee
	got.PPS[1] = 0xee
	if got.SPS[0] != 0x67 ||
		got.PPS[0] != 0x68 ||
		got.AnnexB[3] != 0x67 ||
		got.AVCDecoderConfigurationRecord[0] != 0x01 {
		t.Fatalf("parameter-set clone aliases source after mutation: %+v", got)
	}
	if got.AnnexB[4] != 0x00 || got.AVCDecoderConfigurationRecord[1] != 0x42 {
		t.Fatalf("raw parameter-set clone aliases another returned surface: %+v", got)
	}
}

func TestEncoderParameterSetsFromH264RejectsOverflowedHeaderClones(t *testing.T) {
	got := encoderParameterSetsFromH264(h264.EncoderParameterSets{
		SPS:                           fakeEncoderBytesLen(maxInt/2 + 1),
		PPS:                           fakeEncoderBytesLen(maxInt/2 + 1),
		AnnexB:                        fakeEncoderBytesLen(maxInt/2 + 1),
		AVCDecoderConfigurationRecord: fakeEncoderBytesLen(maxInt/2 + 1),
	})
	if got.SPS != nil || got.PPS != nil || got.AnnexB != nil || got.AVCDecoderConfigurationRecord != nil {
		t.Fatalf("overflowed parameter-set clones = sps %d pps %d annexb %d avcc %d, want nils",
			len(got.SPS), len(got.PPS), len(got.AnnexB), len(got.AVCDecoderConfigurationRecord))
	}
}

func TestEncoderSEIFromH264ClonesSurfaces(t *testing.T) {
	src := h264.EncoderSEIMessage{
		NAL:    []byte{0x06, 0x05, 0x01, 0xaa},
		AnnexB: []byte{0x00, 0x00, 0x01, 0x06, 0x05, 0x01, 0xaa},
		AVC:    []byte{0x00, 0x00, 0x00, 0x04, 0x06, 0x05, 0x01, 0xaa},
	}
	got := encoderSEIFromH264(src)
	if !bytes.Equal(got.NAL, src.NAL) ||
		!bytes.Equal(got.AnnexB, src.AnnexB) ||
		!bytes.Equal(got.AVC, src.AVC) {
		t.Fatalf("cloned SEI = %+v, want source bytes", got)
	}

	src.NAL[0] = 0xff
	src.AnnexB[3] = 0xff
	src.AVC[4] = 0xff
	got.NAL[1] = 0xee
	if got.NAL[0] != 0x06 || got.AnnexB[3] != 0x06 || got.AVC[4] != 0x06 {
		t.Fatalf("SEI clone aliases source after mutation: %+v", got)
	}
	if got.AnnexB[4] != 0x05 || got.AVC[5] != 0x05 {
		t.Fatalf("SEI clone aliases another returned surface: %+v", got)
	}
}

func TestEncoderSEIFromH264RejectsOverflowedSurfaceClones(t *testing.T) {
	got := encoderSEIFromH264(h264.EncoderSEIMessage{
		NAL:    fakeEncoderBytesLen(maxInt/2 + 1),
		AnnexB: fakeEncoderBytesLen(maxInt/2 + 1),
		AVC:    fakeEncoderBytesLen(maxInt/2 + 1),
	})
	if got.NAL != nil || got.AnnexB != nil || got.AVC != nil {
		t.Fatalf("overflowed SEI clones = nal %d annexb %d avc %d, want nils",
			len(got.NAL), len(got.AnnexB), len(got.AVC))
	}
}

func TestFrameSideDataFromH264ClonesS12MTimecodes(t *testing.T) {
	src := h264.DecodedFrameSideData{S12MTimecodes: []uint32{0x11223344, 0x55667788}}
	got := frameSideDataFromH264(src, 0, 0)
	if len(got.S12MTimecodes) != 2 || got.S12MTimecodes[0] != src.S12MTimecodes[0] || got.S12MTimecodes[1] != src.S12MTimecodes[1] {
		t.Fatalf("s12m timecodes = %08x, want %08x", got.S12MTimecodes, src.S12MTimecodes)
	}
	src.S12MTimecodes[0] = 0
	if got.S12MTimecodes[0] != 0x11223344 {
		t.Fatalf("s12m timecode aliases source: %08x", got.S12MTimecodes)
	}
}

func TestFrameSideDataFromH264RejectsOverflowedS12MTimecodeClone(t *testing.T) {
	src := h264.DecodedFrameSideData{S12MTimecodes: fakeUint32SliceLen(maxInt/4 + 1)}
	got := frameSideDataFromH264(src, 0, 0)
	if got.S12MTimecodes != nil {
		t.Fatalf("overflowed s12m timecodes = len %d, want nil", len(got.S12MTimecodes))
	}
}

func TestFrameSideDataFromH264ClonesUserDataUnregistered(t *testing.T) {
	src := h264.DecodedFrameSideData{UserDataUnregistered: [][]byte{{0x01, 0x02}, {0x03}}}
	got := frameSideDataFromH264(src, 0, 0)
	if len(got.UserDataUnregistered) != 2 ||
		len(got.UserDataUnregistered[0]) != 2 ||
		got.UserDataUnregistered[0][0] != 0x01 ||
		got.UserDataUnregistered[1][0] != 0x03 {
		t.Fatalf("unregistered user data = %x", got.UserDataUnregistered)
	}
	src.UserDataUnregistered[0][0] = 0xff
	if got.UserDataUnregistered[0][0] != 0x01 {
		t.Fatalf("unregistered user data aliases source: %x", got.UserDataUnregistered)
	}
}

func TestFrameSideDataFromH264RejectsOverflowedUserDataListClone(t *testing.T) {
	src := h264.DecodedFrameSideData{UserDataUnregistered: fakeByteSlicesLen(maxInt/32 + 1)}
	got := frameSideDataFromH264(src, 0, 0)
	if got.UserDataUnregistered != nil {
		t.Fatalf("overflowed unregistered user data list = len %d, want nil", len(got.UserDataUnregistered))
	}
}

func TestFrameSideDataFromH264RejectsOverflowedUserDataPayloadClone(t *testing.T) {
	src := h264.DecodedFrameSideData{UserDataUnregistered: [][]byte{
		fakeEncoderBytesLen(maxInt/2 + 1),
		{0x01, 0x02},
	}}
	got := frameSideDataFromH264(src, 0, 0)
	if len(got.UserDataUnregistered) != 2 {
		t.Fatalf("unregistered user data len = %d, want 2", len(got.UserDataUnregistered))
	}
	if got.UserDataUnregistered[0] != nil {
		t.Fatalf("overflowed unregistered user data payload = len %d, want nil", len(got.UserDataUnregistered[0]))
	}
	if len(got.UserDataUnregistered[1]) != 2 || got.UserDataUnregistered[1][0] != 0x01 {
		t.Fatalf("valid unregistered user data payload = %x, want 0102", got.UserDataUnregistered[1])
	}
}

func TestFrameFromH264ClonesPublicPlanes(t *testing.T) {
	src := &h264.DecodedFrame{
		Width:  2,
		Height: 2,
		Y:      []byte{1, 2, 3, 4},
		Cb:     []byte{5},
		Cr:     []byte{6},
		Y16:    []uint16{7, 8},
		Cb16:   []uint16{9},
		Cr16:   []uint16{10},
	}
	got := frameFromH264(src)
	if got == nil ||
		len(got.Y) != 4 || got.Y[0] != 1 ||
		len(got.Cb) != 1 || got.Cb[0] != 5 ||
		len(got.Cr) != 1 || got.Cr[0] != 6 ||
		len(got.Y16) != 2 || got.Y16[0] != 7 ||
		len(got.Cb16) != 1 || got.Cb16[0] != 9 ||
		len(got.Cr16) != 1 || got.Cr16[0] != 10 {
		t.Fatalf("frame planes = y %v cb %v cr %v y16 %v cb16 %v cr16 %v", got.Y, got.Cb, got.Cr, got.Y16, got.Cb16, got.Cr16)
	}
	src.Y[0], src.Cb[0], src.Cr[0] = 0xff, 0xff, 0xff
	src.Y16[0], src.Cb16[0], src.Cr16[0] = 0xffff, 0xffff, 0xffff
	if got.Y[0] != 1 || got.Cb[0] != 5 || got.Cr[0] != 6 ||
		got.Y16[0] != 7 || got.Cb16[0] != 9 || got.Cr16[0] != 10 {
		t.Fatalf("frame planes alias source = y %v cb %v cr %v y16 %v cb16 %v cr16 %v", got.Y, got.Cb, got.Cr, got.Y16, got.Cb16, got.Cr16)
	}
}

func TestFrameFromH264RejectsOverflowedPublicPlaneClones(t *testing.T) {
	got := frameFromH264(&h264.DecodedFrame{
		Y:   fakeEncoderBytesLen(maxInt/2 + 1),
		Y16: fakeUint16SliceLen(maxInt/2 + 1),
	})
	if got == nil {
		t.Fatal("frameFromH264 returned nil")
	}
	if got.Y != nil || got.Y16 != nil {
		t.Fatalf("overflowed frame planes = y len %d y16 len %d, want nil/nil", len(got.Y), len(got.Y16))
	}
}

func TestFramesFromH264ConvertsFrames(t *testing.T) {
	got := framesFromH264([]*h264.DecodedFrame{{Width: 1}, nil, {Height: 2}})
	if len(got) != 3 {
		t.Fatalf("converted frame count = %d, want 3", len(got))
	}
	if got[0] == nil || got[0].Width != 1 {
		t.Fatalf("first converted frame = %+v, want width 1", got[0])
	}
	if got[1] != nil {
		t.Fatalf("nil source frame converted to %+v, want nil", got[1])
	}
	if got[2] == nil || got[2].Height != 2 {
		t.Fatalf("third converted frame = %+v, want height 2", got[2])
	}
}

func TestFramesFromH264RejectsOverflowedFrameList(t *testing.T) {
	if got := framesFromH264(fakeDecodedFramesLen(maxInt/8 + 1)); got != nil {
		t.Fatalf("overflowed frame list converted to len %d, want nil", len(got))
	}
}

func TestFramesFromH264WithErrorRejectsOverflowedFrameList(t *testing.T) {
	wantErr := errors.New("decode failed after output")
	got, err := framesFromH264WithError(fakeDecodedFramesLen(maxInt/8+1), wantErr)
	if !errors.Is(err, wantErr) {
		t.Fatalf("overflowed frame list error = %v, want %v", err, wantErr)
	}
	if got != nil {
		t.Fatalf("overflowed frame list with error converted to len %d, want nil", len(got))
	}
}

func TestSingleFrameFromFramesClassifiesCardinalityAndPreservesErrors(t *testing.T) {
	wantFrame := &Frame{Width: 16}
	wantErr := errors.New("decode failed after output")

	got, err := singleFrameFromFrames([]*Frame{wantFrame}, wantErr)
	if !errors.Is(err, wantErr) || got != wantFrame {
		t.Fatalf("single frame with error = %+v/%v, want original frame and error", got, err)
	}

	got, err = singleFrameFromFrames(nil, wantErr)
	if !errors.Is(err, wantErr) || got != nil {
		t.Fatalf("zero frames with error = %+v/%v, want nil and original error", got, err)
	}

	got, err = singleFrameFromFrames([]*Frame{{Width: 1}, {Width: 2}}, wantErr)
	if !errors.Is(err, wantErr) || got != nil {
		t.Fatalf("multiple frames with error = %+v/%v, want nil and original error", got, err)
	}

	for _, tt := range []struct {
		name   string
		frames []*Frame
	}{
		{name: "zero", frames: nil},
		{name: "multiple", frames: []*Frame{{Width: 1}, {Width: 2}}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := singleFrameFromFrames(tt.frames, nil)
			if !errors.Is(err, ErrUnsupported) || got != nil {
				t.Fatalf("%s frames without decode error = %+v/%v, want nil ErrUnsupported", tt.name, got, err)
			}
		})
	}
}

func TestPacketFrameSideDataFromPacketClonesBytePayloads(t *testing.T) {
	captions := []byte{0x01, 0x02}
	icc := []byte{0x03}
	hdr10Plus := []byte{0x04}
	lcevc := []byte{0x05}
	got := packetFrameSideDataFromPacket([]PacketSideData{
		{Type: PacketSideDataA53ClosedCaptions, Data: captions},
		{Type: PacketSideDataICCProfile, Data: icc},
		{Type: PacketSideDataDynamicHDR10Plus, Data: hdr10Plus},
		{Type: PacketSideDataLCEVC, Data: lcevc},
	})
	if got.A53ClosedCaptions[0] != 0x01 || got.ICCProfile[0] != 0x03 ||
		got.DynamicHDR10Plus[0] != 0x04 || got.LCEVC[0] != 0x05 {
		t.Fatalf("packet byte side data = captions %x icc %x hdr %x lcevc %x", got.A53ClosedCaptions, got.ICCProfile, got.DynamicHDR10Plus, got.LCEVC)
	}
	captions[0], icc[0], hdr10Plus[0], lcevc[0] = 0xff, 0xff, 0xff, 0xff
	if got.A53ClosedCaptions[0] != 0x01 || got.ICCProfile[0] != 0x03 ||
		got.DynamicHDR10Plus[0] != 0x04 || got.LCEVC[0] != 0x05 {
		t.Fatalf("packet byte side data aliases source = captions %x icc %x hdr %x lcevc %x", got.A53ClosedCaptions, got.ICCProfile, got.DynamicHDR10Plus, got.LCEVC)
	}
}

func TestPacketFrameSideDataFromPacketRejectsOverflowedBytePayloads(t *testing.T) {
	got := packetFrameSideDataFromPacket([]PacketSideData{
		{Type: PacketSideDataA53ClosedCaptions, Data: fakeEncoderBytesLen(maxInt/2 + 1)},
		{Type: PacketSideDataICCProfile, Data: fakeEncoderBytesLen(maxInt/2 + 1)},
		{Type: PacketSideDataDynamicHDR10Plus, Data: fakeEncoderBytesLen(maxInt/2 + 1)},
		{Type: PacketSideDataLCEVC, Data: fakeEncoderBytesLen(maxInt/2 + 1)},
	})
	if got.A53ClosedCaptions != nil || got.ICCProfile != nil || got.DynamicHDR10Plus != nil || got.LCEVC != nil {
		t.Fatalf("overflowed packet byte side data = captions %d icc %d hdr %d lcevc %d",
			len(got.A53ClosedCaptions), len(got.ICCProfile), len(got.DynamicHDR10Plus), len(got.LCEVC))
	}
}

func TestFrameSideDataFromH264RejectsOverflowedBytePayloads(t *testing.T) {
	src := h264.DecodedFrameSideData{
		A53ClosedCaptions: fakeEncoderBytesLen(maxInt/2 + 1),
		ICCProfile:        fakeEncoderBytesLen(maxInt/2 + 1),
		DynamicHDR10Plus:  fakeEncoderBytesLen(maxInt/2 + 1),
		LCEVC:             fakeEncoderBytesLen(maxInt/2 + 1),
	}
	got := frameSideDataFromH264(src, 0, 0)
	if got.A53ClosedCaptions != nil || got.ICCProfile != nil || got.DynamicHDR10Plus != nil || got.LCEVC != nil {
		t.Fatalf("overflowed frame byte side data = captions %d icc %d hdr %d lcevc %d",
			len(got.A53ClosedCaptions), len(got.ICCProfile), len(got.DynamicHDR10Plus), len(got.LCEVC))
	}
}

func TestFrameSideDataFromH264RejectsOverflowedReferenceDisplaysClone(t *testing.T) {
	src := h264.DecodedFrameSideData{
		ReferenceDisplays: h264.AV3DReferenceDisplaysInfo{
			Present:  1,
			Displays: fakeReferenceDisplaysLen(maxInt/16 + 1),
		},
	}
	got := frameSideDataFromH264(src, 0, 0)
	if got.ReferenceDisplays != nil {
		t.Fatalf("overflowed reference displays = %d, want nil", len(got.ReferenceDisplays.Displays))
	}
}

func TestFrameSideDataFromH264ClonesReferenceDisplays(t *testing.T) {
	src := h264.DecodedFrameSideData{
		ReferenceDisplays: h264.AV3DReferenceDisplaysInfo{
			Present:                1,
			PrecRefDisplayWidth:    12,
			RefViewingDistanceFlag: 1,
			PrecRefViewingDist:     9,
			Displays: []h264.AV3DReferenceDisplay{
				{
					LeftViewID:                 3,
					RightViewID:                4,
					ExponentRefDisplayWidth:    5,
					MantissaRefDisplayWidth:    6,
					ExponentRefViewingDistance: 7,
					MantissaRefViewingDistance: 8,
					AdditionalShiftPresentFlag: 1,
					NumSampleShift:             -11,
				},
			},
		},
	}
	got := frameSideDataFromH264(src, 0, 0)
	if got.ReferenceDisplays == nil {
		t.Fatal("reference displays = nil, want converted display")
	}
	if got.ReferenceDisplays.PrecRefDisplayWidth != 12 ||
		!got.ReferenceDisplays.RefViewingDistanceFlag ||
		got.ReferenceDisplays.PrecRefViewingDist != 9 ||
		len(got.ReferenceDisplays.Displays) != 1 {
		t.Fatalf("reference displays header = %+v, want converted header with one display", got.ReferenceDisplays)
	}
	display := got.ReferenceDisplays.Displays[0]
	if display.LeftViewID != 3 ||
		display.RightViewID != 4 ||
		display.ExponentRefDisplayWidth != 5 ||
		display.MantissaRefDisplayWidth != 6 ||
		display.ExponentRefViewingDistance != 7 ||
		display.MantissaRefViewingDistance != 8 ||
		!display.AdditionalShiftPresentFlag ||
		display.NumSampleShift != -11 {
		t.Fatalf("reference display = %+v, want converted source fields", display)
	}

	src.ReferenceDisplays.Displays[0].LeftViewID = 99
	src.ReferenceDisplays.Displays[0].NumSampleShift = 22
	if got.ReferenceDisplays.Displays[0].LeftViewID != 3 || got.ReferenceDisplays.Displays[0].NumSampleShift != -11 {
		t.Fatalf("reference displays alias source after mutation = %+v", got.ReferenceDisplays.Displays[0])
	}
}

func TestCloneEncoderRTPPacketClonesSharedPayloadStorage(t *testing.T) {
	data := []byte{0x80, 0x60, 0, 1, 0, 0, 0, 2, 0, 0, 0, 3, 0x65, 0xaa}
	pkt := EncoderRTPPacket{Data: data, Payload: data[12:]}

	got := cloneEncoderRTPPacket(pkt)
	if len(got.Data) != len(data) || len(got.Payload) != 2 || got.Payload[0] != 0x65 || got.Payload[1] != 0xaa {
		t.Fatalf("cloned shared RTP packet = data %x payload %x", got.Data, got.Payload)
	}
	data[12], data[13] = 0xff, 0xff
	if got.Payload[0] != 0x65 || got.Payload[1] != 0xaa || got.Data[12] != 0x65 || got.Data[13] != 0xaa {
		t.Fatalf("cloned shared RTP packet aliases source = data %x payload %x", got.Data, got.Payload)
	}
	if len(got.Data) > 0 && cap(got.Data) != len(got.Data) {
		t.Fatalf("cloned RTP data cap = %d, want len %d", cap(got.Data), len(got.Data))
	}
	if len(got.Payload) > 0 && cap(got.Payload) != len(got.Payload) {
		t.Fatalf("cloned RTP payload cap = %d, want len %d", cap(got.Payload), len(got.Payload))
	}
}

func TestCloneEncoderRTPPacketClonesSplitPayloadStorage(t *testing.T) {
	data := []byte{0x80, 0x60}
	payload := []byte{0x41, 0xbb}
	pkt := EncoderRTPPacket{Data: data, Payload: payload}

	got := cloneEncoderRTPPacket(pkt)
	if len(got.Data) != 2 || got.Data[0] != 0x80 || len(got.Payload) != 2 || got.Payload[0] != 0x41 {
		t.Fatalf("cloned split RTP packet = data %x payload %x", got.Data, got.Payload)
	}
	data[0], payload[0] = 0xff, 0xff
	if got.Data[0] != 0x80 || got.Payload[0] != 0x41 {
		t.Fatalf("cloned split RTP packet aliases source = data %x payload %x", got.Data, got.Payload)
	}
}

func TestCloneEncoderRTPPacketRejectsOverflowedByteClones(t *testing.T) {
	got := cloneEncoderRTPPacket(EncoderRTPPacket{
		Data:    fakeEncoderBytesLen(maxInt/2 + 1),
		Payload: fakeEncoderBytesLen(maxInt/2 + 1),
	})
	if got.Data != nil || got.Payload != nil {
		t.Fatalf("overflowed split RTP clone = data len %d payload len %d, want nil/nil", len(got.Data), len(got.Payload))
	}

	shared := fakeEncoderBytesLen(maxInt/2 + 1)
	got = cloneEncoderRTPPacket(EncoderRTPPacket{
		Data:    shared,
		Payload: shared[12:],
	})
	if got.Data != nil || got.Payload != nil {
		t.Fatalf("overflowed shared RTP clone = data len %d payload len %d, want nil/nil", len(got.Data), len(got.Payload))
	}
}

func TestEncoderFrameCloneRejectsOverflowedPlanes(t *testing.T) {
	for _, tt := range []struct {
		name  string
		frame EncoderFrame
	}{
		{name: "luma", frame: EncoderFrame{Y: fakeEncoderBytesLen(maxInt/2 + 1)}},
		{name: "cb", frame: EncoderFrame{Cb: fakeEncoderBytesLen(maxInt/2 + 1)}},
		{name: "cr", frame: EncoderFrame{Cr: fakeEncoderBytesLen(maxInt/2 + 1)}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.frame.Clone()
			if !errors.Is(err, ErrInvalidData) || len(got.Y) != 0 || len(got.Cb) != 0 || len(got.Cr) != 0 {
				t.Fatalf("Clone overflow = %+v/%v, want zero ErrInvalidData", got, err)
			}
		})
	}
}

func TestEncoderRTPPacketMetadataClassifiesMalformedPayloads(t *testing.T) {
	for _, tt := range []struct {
		name         string
		payload      []byte
		wantFormat   EncoderRTPPayloadFormat
		wantNALType  uint8
		wantCount    int
		wantStart    bool
		wantEnd      bool
		wantParamSet bool
	}{
		{
			name: "empty-payload",
		},
		{
			name:        "truncated-stapa-length",
			payload:     []byte{24, 0},
			wantFormat:  EncoderRTPPayloadSTAPA,
			wantNALType: 24,
		},
		{
			name:        "zero-size-stapa-nal",
			payload:     []byte{24, 0, 0},
			wantFormat:  EncoderRTPPayloadSTAPA,
			wantNALType: 24,
		},
		{
			name:        "oversize-stapa-nal",
			payload:     []byte{24, 0, 3, 0x67},
			wantFormat:  EncoderRTPPayloadSTAPA,
			wantNALType: 24,
		},
		{
			name:         "valid-parameter-set-stapa",
			payload:      []byte{24, 0, 1, 0x67, 0, 1, 0x68},
			wantFormat:   EncoderRTPPayloadSTAPA,
			wantNALType:  24,
			wantCount:    2,
			wantParamSet: true,
		},
		{
			name:        "valid-mixed-stapa",
			payload:     []byte{24, 0, 1, 0x67, 0, 1, 0x65},
			wantFormat:  EncoderRTPPayloadSTAPA,
			wantNALType: 24,
			wantCount:   2,
		},
		{
			name:       "truncated-fua",
			payload:    []byte{28},
			wantFormat: EncoderRTPPayloadFUA,
			wantCount:  1,
		},
		{
			name:        "fua-start-idr",
			payload:     []byte{28, 0x80 | 5},
			wantFormat:  EncoderRTPPayloadFUA,
			wantNALType: 5,
			wantCount:   1,
			wantStart:   true,
		},
		{
			name:         "single-nal-sps",
			payload:      []byte{0x67},
			wantFormat:   EncoderRTPPayloadSingleNAL,
			wantNALType:  7,
			wantCount:    1,
			wantStart:    true,
			wantEnd:      true,
			wantParamSet: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := encoderRTPPacketMetadataFromPayload(tt.payload)
			if got.PayloadFormat != tt.wantFormat ||
				got.NALUnitType != tt.wantNALType ||
				got.NALUnitCount != tt.wantCount ||
				got.StartOfNAL != tt.wantStart ||
				got.EndOfNAL != tt.wantEnd ||
				got.ParameterSet != tt.wantParamSet {
				t.Fatalf("metadata = %+v, want format=%v nal=%d count=%d start=%v end=%v parameterSet=%v",
					got, tt.wantFormat, tt.wantNALType, tt.wantCount, tt.wantStart, tt.wantEnd, tt.wantParamSet)
			}
		})
	}
}

func fakeEncoderBytesLen(n int) []byte {
	if n <= 0 {
		return nil
	}
	var b byte
	return fakeEncoderSliceLen(&b, n)
}

func fakeEncoderRawNALLen(n int) []encoderRawNAL {
	if n <= 0 {
		return nil
	}
	raw := encoderRawNAL{raw: []byte{1}}
	return fakeEncoderSliceLen(&raw, n)
}

func fakeByteSlicesLen(n int) [][]byte {
	if n <= 0 {
		return nil
	}
	var b []byte
	return fakeEncoderSliceLen(&b, n)
}

func fakeUint32SliceLen(n int) []uint32 {
	if n <= 0 {
		return nil
	}
	var v uint32
	return fakeEncoderSliceLen(&v, n)
}

func fakeUint16SliceLen(n int) []uint16 {
	if n <= 0 {
		return nil
	}
	var v uint16
	return fakeEncoderSliceLen(&v, n)
}

func fakeReferenceDisplaysLen(n int) []h264.AV3DReferenceDisplay {
	if n <= 0 {
		return nil
	}
	var display h264.AV3DReferenceDisplay
	return fakeEncoderSliceLen(&display, n)
}

func fakeDecodedFramesLen(n int) []*h264.DecodedFrame {
	if n <= 0 {
		return nil
	}
	var frame *h264.DecodedFrame
	return fakeEncoderSliceLen(&frame, n)
}

// fakeEncoderSliceLen preserves impossible slice lengths for overflow guards.
func fakeEncoderSliceLen[T any](ptr *T, n int) []T {
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

func TestEncoderBitrateFrameBudgetBytes(t *testing.T) {
	cfg := DefaultEncoderConfig(16, 16)
	cfg.MaxBitrate = 1_000_000
	cfg.FrameRateNum = 30
	cfg.FrameRateDen = 1
	if got := encoderBitrateFrameBudgetBytes(cfg); got != 4167 {
		t.Fatalf("30fps 1Mbps budget = %d, want 4167", got)
	}

	cfg.FrameRateNum = 30000
	cfg.FrameRateDen = 1001
	if got := encoderBitrateFrameBudgetBytes(cfg); got != 4171 {
		t.Fatalf("29.97fps 1Mbps budget = %d, want 4171", got)
	}

	cfg.FrameRateNum = 0
	if got := encoderBitrateFrameBudgetBytes(cfg); got != 0 {
		t.Fatalf("invalid framerate budget = %d, want 0", got)
	}

	cfg.VBVBufferSize = 1_000_000
	if got := encoderVBVBufferBudgetBytes(cfg); got != 125000 {
		t.Fatalf("1Mbit VBV budget = %d, want 125000", got)
	}
	cfg.VBVBufferSize = 65
	if got := encoderVBVBufferBudgetBytes(cfg); got != 9 {
		t.Fatalf("65-bit VBV budget = %d, want 9", got)
	}

	cfg.VBVBufferSize = 0
	if got := encoderVBVBufferBudgetBytes(cfg); got != 0 {
		t.Fatalf("zero VBV budget = %d, want 0", got)
	}
	cfg.VBVBufferSize = -1
	if got := encoderVBVBufferBudgetBytes(cfg); got != 0 {
		t.Fatalf("negative VBV budget = %d, want 0", got)
	}
	cfg.VBVBufferSize = maxInt
	if got := encoderVBVBufferBudgetBytes(cfg); got != maxInt/8+1 {
		t.Fatalf("max-int VBV budget = %d, want %d", got, maxInt/8+1)
	}
}
