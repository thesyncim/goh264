// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

import (
	"bytes"
	"errors"
	"testing"
)

func TestBuildEncoderRecoveryPointSEIRoundTripsThroughParsers(t *testing.T) {
	for _, tt := range []struct {
		name                  string
		recoveryFrameCount    uint32
		exactMatch            bool
		brokenLink            bool
		changingSliceGroupIDC uint8
	}{
		{name: "idr recovery", exactMatch: true},
		{name: "nonzero recovery", recoveryFrameCount: 4, brokenLink: true, changingSliceGroupIDC: 2},
	} {
		t.Run(tt.name, func(t *testing.T) {
			sei, err := BuildEncoderRecoveryPointSEI(EncoderRecoveryPointSEIConfig{
				RecoveryFrameCount:    tt.recoveryFrameCount,
				ExactMatchFlag:        tt.exactMatch,
				BrokenLinkFlag:        tt.brokenLink,
				ChangingSliceGroupIDC: tt.changingSliceGroupIDC,
				NALLengthSize:         4,
			})
			if err != nil {
				t.Fatal(err)
			}
			if len(sei.NAL) == 0 || sei.NAL[0]>>5 != 0 || NALUnitType(sei.NAL[0]&0x1f) != NALSEI {
				t.Fatalf("SEI NAL header = %x, want ref_idc=0 type=6", sei.NAL)
			}
			assertEncoderRecoveryPointPayload(t, sei.RBSP, tt.recoveryFrameCount, tt.exactMatch, tt.brokenLink, tt.changingSliceGroupIDC)

			ctx, err := DecodeSEI(sei.RBSP, nil)
			if err != nil {
				t.Fatalf("DecodeSEI: %v", err)
			}
			if got := ctx.RecoveryPoint.RecoveryFrameCount; got != int32(tt.recoveryFrameCount) {
				t.Fatalf("recovery_frame_cnt = %d, want %d", got, tt.recoveryFrameCount)
			}

			nals, err := SplitAnnexB(sei.AnnexB)
			if err != nil {
				t.Fatalf("SplitAnnexB: %v", err)
			}
			if len(nals) != 1 || nals[0].Type != NALSEI || !bytes.Equal(nals[0].Raw, sei.NAL) || !bytes.Equal(nals[0].RBSP, sei.RBSP) {
				t.Fatalf("Annex B SEI = %+v", nals)
			}

			nals, err = SplitAVCC(sei.AVC, 4)
			if err != nil {
				t.Fatalf("SplitAVCC: %v", err)
			}
			if len(nals) != 1 || nals[0].Type != NALSEI || !bytes.Equal(nals[0].Raw, sei.NAL) || !bytes.Equal(nals[0].RBSP, sei.RBSP) {
				t.Fatalf("AVC SEI = %+v", nals)
			}
		})
	}
}

func TestBuildEncoderRecoveryPointSEIRejectsInvalidSyntax(t *testing.T) {
	for _, tt := range []struct {
		name   string
		mutate func(*EncoderRecoveryPointSEIConfig)
	}{
		{name: "recovery frame count too large", mutate: func(c *EncoderRecoveryPointSEIConfig) {
			c.RecoveryFrameCount = 1 << maxLog2MaxFrameNum
		}},
		{name: "changing slice group idc too large", mutate: func(c *EncoderRecoveryPointSEIConfig) {
			c.ChangingSliceGroupIDC = 3
		}},
		{name: "bad nal length size", mutate: func(c *EncoderRecoveryPointSEIConfig) {
			c.NALLengthSize = 5
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cfg := EncoderRecoveryPointSEIConfig{
				ExactMatchFlag: true,
				NALLengthSize:  4,
			}
			tt.mutate(&cfg)
			if _, err := BuildEncoderRecoveryPointSEI(cfg); !errors.Is(err, ErrInvalidData) {
				t.Fatalf("BuildEncoderRecoveryPointSEI error = %v, want ErrInvalidData", err)
			}
		})
	}
}

func TestAppendSEIRBSPWritesExtendedHeaders(t *testing.T) {
	payload := bytes.Repeat([]byte{0x12}, 300)
	rbsp := AppendSEIRBSP(nil, 511, payload)
	if len(rbsp) != 3+2+len(payload)+1 {
		t.Fatalf("RBSP size = %d, want extended type and size headers around payload", len(rbsp))
	}
	if got, want := rbsp[:5], []byte{0xff, 0xff, 0x01, 0xff, 0x2d}; !bytes.Equal(got, want) {
		t.Fatalf("extended SEI headers = %x, want %x", got, want)
	}
	if !bytes.Equal(rbsp[5:5+len(payload)], payload) || rbsp[len(rbsp)-1] != 0x80 {
		t.Fatalf("extended SEI payload/trailing bits corrupted")
	}
}

func assertEncoderRecoveryPointPayload(t *testing.T, rbsp []byte, recoveryFrameCount uint32, exactMatch bool, brokenLink bool, changingSliceGroupIDC uint8) {
	t.Helper()
	if len(rbsp) < 3 || rbsp[0] != seiTypeRecoveryPoint {
		t.Fatalf("SEI RBSP header = %x, want recovery-point payload", rbsp)
	}
	payloadSize := int(rbsp[1])
	if len(rbsp) != 2+payloadSize+1 || rbsp[len(rbsp)-1] != 0x80 {
		t.Fatalf("SEI RBSP framing = %x", rbsp)
	}
	gb := newBitReader(rbsp[2 : 2+payloadSize])
	gotRecovery, err := gb.readUEGolombLong()
	if err != nil {
		t.Fatal(err)
	}
	if gotRecovery != recoveryFrameCount {
		t.Fatalf("payload recovery_frame_cnt = %d, want %d", gotRecovery, recoveryFrameCount)
	}
	gotExact, err := gb.readBit()
	if err != nil {
		t.Fatal(err)
	}
	if (gotExact != 0) != exactMatch {
		t.Fatalf("payload exact_match_flag = %d, want %v", gotExact, exactMatch)
	}
	gotBroken, err := gb.readBit()
	if err != nil {
		t.Fatal(err)
	}
	if (gotBroken != 0) != brokenLink {
		t.Fatalf("payload broken_link_flag = %d, want %v", gotBroken, brokenLink)
	}
	gotChanging, err := gb.readBits(2)
	if err != nil {
		t.Fatal(err)
	}
	if gotChanging != uint32(changingSliceGroupIDC) {
		t.Fatalf("payload changing_slice_group_idc = %d, want %d", gotChanging, changingSliceGroupIDC)
	}
}
