// SPDX-License-Identifier: LGPL-2.1-or-later

package goh264

import (
	"errors"
	"reflect"
	"testing"
	"unsafe"
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

func TestEncoderRTPMode1StoragePlanRejectsOverflow(t *testing.T) {
	nals := []encoderRawNAL{
		{raw: fakeEncoderBytesLen(maxInt - 1)},
	}
	if _, _, err := encoderRTPMode1StoragePlan(nals, 3, false); !errors.Is(err, ErrInvalidData) {
		t.Fatalf("encoderRTPMode1StoragePlan overflow error = %v, want ErrInvalidData", err)
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

func fakeEncoderBytesLen(n int) []byte {
	if n <= 0 {
		return nil
	}
	var b byte
	return *(*[]byte)(unsafe.Pointer(&reflect.SliceHeader{
		Data: uintptr(unsafe.Pointer(&b)),
		Len:  n,
		Cap:  n,
	}))
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
}
