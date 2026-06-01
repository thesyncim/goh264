// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

import "testing"

func TestValidateHighFrameSliceMacroblockForReconstructAllowsP16x16Residual(t *testing.T) {
	sh := &SliceHeader{SliceTypeNoS: PictureTypeP}

	for _, tt := range []struct {
		name     string
		cbp      int
		cbpTable int
	}{
		{name: "no residual"},
		{name: "luma residual", cbp: 0x01, cbpTable: 0x1001},
		{name: "luma chroma residual", cbp: 0x31, cbpTable: 0xf031},
	} {
		t.Run(tt.name, func(t *testing.T) {
			mbType := MBType16x16 | MBTypeP0L0
			if err := validateHighFrameSliceMacroblockForReconstruct(sh, mbType, tt.cbp, tt.cbpTable); err != nil {
				t.Fatalf("validate P16x16 residual err = %v, want nil", err)
			}
		})
	}
}

func TestValidateHighFrameSliceMacroblockForReconstructAllowsProvedPIntra(t *testing.T) {
	sh := &SliceHeader{SliceTypeNoS: PictureTypeP}

	for _, mbType := range []uint32{
		MBTypeIntra4x4,
		MBTypeIntra16x16,
	} {
		if err := validateHighFrameSliceMacroblockForReconstruct(sh, mbType, 1, 1); err != nil {
			t.Fatalf("validate high P intra %#x err = %v, want nil", mbType, err)
		}
	}
}

func TestValidateHighFrameSliceMacroblockForReconstructAllowsBDirectSkip(t *testing.T) {
	sh := &SliceHeader{SliceTypeNoS: PictureTypeB}
	for _, mbType := range []uint32{
		MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2 | MBTypeSkip,
		MBType16x16 | MBTypeL0L1 | MBTypeDirect2 | MBTypeSkip,
	} {
		if err := validateHighFrameSliceMacroblockForReconstruct(sh, mbType, 0, 0); err != nil {
			t.Fatalf("validate high B direct skip %#x err = %v, want nil", mbType, err)
		}
	}
}

func TestValidateHighFrameSliceMacroblockForReconstructAllowsB8x8DirectSubNoResidual(t *testing.T) {
	sh := &SliceHeader{SliceTypeNoS: PictureTypeB}
	mbType := MBType8x8 | MBTypeP0L0 | MBTypeP0L1 | MBTypeP1L0 | MBTypeP1L1

	for _, tt := range []struct {
		name string
		sub  [4]uint32
	}{
		{
			name: "direct 8x8 inference",
			sub: [4]uint32{
				MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2,
				MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2,
				MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2,
				MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2,
			},
		},
		{
			name: "direct sub 4x4",
			sub: [4]uint32{
				MBType8x8 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2,
				MBType8x8 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2,
				MBType8x8 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2,
				MBType8x8 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2,
			},
		},
		{
			name: "spatial direct sub 4x4",
			sub: [4]uint32{
				MBType8x8 | MBTypeL0L1 | MBTypeDirect2,
				MBType8x8 | MBTypeL0L1 | MBTypeDirect2,
				MBType8x8 | MBTypeL0L1 | MBTypeDirect2,
				MBType8x8 | MBTypeL0L1 | MBTypeDirect2,
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateHighFrameSliceMacroblockForReconstructWithSubMB(sh, mbType, &tt.sub, 0, 0); err != nil {
				t.Fatalf("validate high B direct-sub err = %v, want nil", err)
			}
		})
	}
}

func TestValidateHighFrameSliceMacroblockForReconstructAllowsBPartitionedExplicit(t *testing.T) {
	unweighted := &SliceHeader{SliceTypeNoS: PictureTypeB}
	implicitWeighted := &SliceHeader{SliceTypeNoS: PictureTypeB, PPS: &PPS{WeightedBipredIDC: 2}}
	b8x8 := MBType8x8 | MBTypeP0L0 | MBTypeP0L1 | MBTypeP1L0 | MBTypeP1L1
	allL0 := [4]uint32{
		MBType16x16 | MBTypeP0L0,
		MBType16x16 | MBTypeP0L0,
		MBType16x16 | MBTypeP0L0,
		MBType16x16 | MBTypeP0L0,
	}
	mixedSubPartitions := [4]uint32{
		MBType16x16 | MBTypeP0L0,
		MBType16x16 | MBTypeP0L1,
		MBType16x16 | MBTypeP0L0 | MBTypeP0L1,
		MBType8x8 | MBTypeP0L0 | MBTypeP0L1 | MBTypeP1L0 | MBTypeP1L1,
	}

	tests := []struct {
		name     string
		sh       *SliceHeader
		mbType   uint32
		sub      *[4]uint32
		cbp      int
		cbpTable int
	}{
		{name: "b16x8 l0 l0", sh: unweighted, mbType: MBType16x8 | MBTypeP0L0 | MBTypeP1L0},
		{name: "b16x8 l0 l1 residual", sh: unweighted, mbType: MBType16x8 | MBTypeP0L0 | MBTypeP1L1, cbp: 1, cbpTable: 1},
		{name: "b8x16 l1 l1", sh: unweighted, mbType: MBType8x16 | MBTypeP0L1 | MBTypeP1L1},
		{name: "b8x16 bidirectional residual", sh: unweighted, mbType: MBType8x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeP1L0 | MBTypeP1L1, cbp: 3, cbpTable: 3},
		{name: "b8x8 explicit all l0", sh: unweighted, mbType: b8x8, sub: &allL0},
		{name: "b8x8 explicit mixed subpartitions residual", sh: unweighted, mbType: b8x8, sub: &mixedSubPartitions, cbp: 2, cbpTable: 2},
		{name: "implicit weighted b16x8 l0 l0", sh: implicitWeighted, mbType: MBType16x8 | MBTypeP0L0 | MBTypeP1L0},
		{name: "implicit weighted b8x16 l1 l1", sh: implicitWeighted, mbType: MBType8x16 | MBTypeP0L1 | MBTypeP1L1},
		{name: "implicit weighted b8x8 explicit all l0", sh: implicitWeighted, mbType: b8x8, sub: &allL0},
		{name: "implicit weighted b8x8 explicit mixed subpartitions residual", sh: implicitWeighted, mbType: b8x8, sub: &mixedSubPartitions, cbp: 2, cbpTable: 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateHighFrameSliceBaseMacroblockForDecode(PictureTypeB, tt.mbType); err != nil {
				t.Fatalf("validate high B partitioned base err = %v, want nil", err)
			}
			if err := validateHighFrameSliceMacroblockForReconstructWithSubMB(tt.sh, tt.mbType, tt.sub, tt.cbp, tt.cbpTable); err != nil {
				t.Fatalf("validate high B partitioned reconstruct err = %v, want nil", err)
			}
		})
	}
}

func TestValidateHighFrameSliceMacroblockForReconstructAllowsB16x16Deblocking(t *testing.T) {
	for _, tt := range []struct {
		name   string
		pps    *PPS
		mbType uint32
	}{
		{name: "cavlc non-direct", pps: &PPS{}, mbType: MBType16x16 | MBTypeP0L0 | MBTypeP0L1},
		{name: "cabac non-direct", pps: &PPS{CABAC: 1}, mbType: MBType16x16 | MBTypeP0L0 | MBTypeP0L1},
		{name: "cavlc temporal direct", pps: &PPS{}, mbType: MBType16x16 | MBTypeL0L1 | MBTypeDirect2},
		{name: "cabac temporal direct", pps: &PPS{CABAC: 1}, mbType: MBType16x16 | MBTypeL0L1 | MBTypeDirect2},
		{name: "cavlc spatial direct", pps: &PPS{}, mbType: MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2},
		{name: "cabac spatial direct", pps: &PPS{CABAC: 1}, mbType: MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2},
	} {
		t.Run(tt.name, func(t *testing.T) {
			sh := &SliceHeader{
				SliceTypeNoS:     PictureTypeB,
				DeblockingFilter: 1,
				PPS:              tt.pps,
			}
			if err := validateHighFrameSliceMacroblockForReconstruct(sh, tt.mbType, 0x31, 0xf031); err != nil {
				t.Fatalf("validate high B16x16 deblock err = %v, want nil", err)
			}
		})
	}
}

func TestValidateHighFrameSliceMacroblockForReconstructAllowsImplicitPartitionedBDeblocking(t *testing.T) {
	bExplicitSub := [4]uint32{
		MBType16x16 | MBTypeP0L0,
		MBType16x16 | MBTypeP0L1,
		MBType16x16 | MBTypeP0L0 | MBTypeP0L1,
		MBType16x16 | MBTypeP0L1,
	}
	for _, tt := range []struct {
		name     string
		pps      *PPS
		mbType   uint32
		sub      *[4]uint32
		cbp      int
		cbpTable int
	}{
		{name: "cavlc b16x8", pps: &PPS{WeightedBipredIDC: 2}, mbType: MBType16x8 | MBTypeP0L1 | MBTypeP1L0},
		{name: "cabac b16x8", pps: &PPS{CABAC: 1, WeightedBipredIDC: 2}, mbType: MBType16x8 | MBTypeP0L1 | MBTypeP1L0},
		{name: "cavlc b8x16", pps: &PPS{WeightedBipredIDC: 2}, mbType: MBType8x16 | MBTypeP0L1 | MBTypeP1L0},
		{name: "cabac b8x16", pps: &PPS{CABAC: 1, WeightedBipredIDC: 2}, mbType: MBType8x16 | MBTypeP0L1 | MBTypeP1L0},
		{name: "cavlc b8x8 residual", pps: &PPS{WeightedBipredIDC: 2}, mbType: MBType8x8 | MBTypeP0L0 | MBTypeP0L1 | MBTypeP1L0 | MBTypeP1L1, sub: &bExplicitSub, cbp: 0x5, cbpTable: 0x5005},
		{name: "cabac b8x8 residual", pps: &PPS{CABAC: 1, WeightedBipredIDC: 2}, mbType: MBType8x8 | MBTypeP0L0 | MBTypeP0L1 | MBTypeP1L0 | MBTypeP1L1, sub: &bExplicitSub, cbp: 0x5, cbpTable: 0x5},
	} {
		t.Run(tt.name, func(t *testing.T) {
			sh := &SliceHeader{
				SliceTypeNoS:     PictureTypeB,
				DeblockingFilter: 1,
				PPS:              tt.pps,
			}
			if err := validateHighFrameSliceMacroblockForReconstructWithSubMB(sh, tt.mbType, tt.sub, tt.cbp, tt.cbpTable); err != nil {
				t.Fatalf("validate high implicit partitioned B deblock err = %v, want nil", err)
			}
		})
	}
}

func TestValidateHighFrameSliceMacroblockForReconstructAllowsNeutralPartitionedBDeblocking(t *testing.T) {
	bExplicitSub := [4]uint32{
		MBType16x16 | MBTypeP0L0,
		MBType16x16 | MBTypeP0L1,
		MBType16x16 | MBTypeP0L0 | MBTypeP0L1,
		MBType16x16 | MBTypeP0L1,
	}
	for _, tt := range []struct {
		name     string
		pps      *PPS
		mbType   uint32
		sub      *[4]uint32
		cbp      int
		cbpTable int
	}{
		{name: "cavlc b16x8", pps: &PPS{}, mbType: MBType16x8 | MBTypeP0L1 | MBTypeP1L0},
		{name: "cabac b16x8", pps: &PPS{CABAC: 1}, mbType: MBType16x8 | MBTypeP0L1 | MBTypeP1L0},
		{name: "cavlc b8x16", pps: &PPS{}, mbType: MBType8x16 | MBTypeP0L1 | MBTypeP1L0},
		{name: "cabac b8x16", pps: &PPS{CABAC: 1}, mbType: MBType8x16 | MBTypeP0L1 | MBTypeP1L0},
		{name: "cavlc b8x8 residual", pps: &PPS{}, mbType: MBType8x8 | MBTypeP0L0 | MBTypeP0L1 | MBTypeP1L0 | MBTypeP1L1, sub: &bExplicitSub, cbp: 0x7, cbpTable: 0x7007},
		{name: "cabac b8x8 residual", pps: &PPS{CABAC: 1}, mbType: MBType8x8 | MBTypeP0L0 | MBTypeP0L1 | MBTypeP1L0 | MBTypeP1L1, sub: &bExplicitSub, cbp: 0x7, cbpTable: 0x7},
	} {
		t.Run(tt.name, func(t *testing.T) {
			sh := &SliceHeader{
				SliceTypeNoS:     PictureTypeB,
				DeblockingFilter: 1,
				PPS:              tt.pps,
			}
			if err := validateHighFrameSliceMacroblockForReconstructWithSubMB(sh, tt.mbType, tt.sub, tt.cbp, tt.cbpTable); err != nil {
				t.Fatalf("validate high neutral partitioned B deblock err = %v, want nil", err)
			}
		})
	}
}

func TestValidateHighFrameSliceMacroblockForReconstructRejectsPResidualGuardBoundaries(t *testing.T) {
	pSlice := &SliceHeader{SliceTypeNoS: PictureTypeP}
	bSlice := &SliceHeader{SliceTypeNoS: PictureTypeB}
	bImplicitWeightedSlice := &SliceHeader{SliceTypeNoS: PictureTypeB, PPS: &PPS{WeightedBipredIDC: 2}}
	bDeblockSlice := &SliceHeader{SliceTypeNoS: PictureTypeB, DeblockingFilter: 1, PPS: &PPS{}}
	bCABACDeblockSlice := &SliceHeader{SliceTypeNoS: PictureTypeB, DeblockingFilter: 1, PPS: &PPS{CABAC: 1}}
	bImplicitDeblockSlice := &SliceHeader{SliceTypeNoS: PictureTypeB, DeblockingFilter: 1, PPS: &PPS{WeightedBipredIDC: 2}}
	pSkip := MBType16x16 | MBTypeP0L0 | MBTypeP1L0 | MBTypeSkip
	bSkip := MBType16x16 | MBTypeL0L1 | MBTypeDirect2 | MBTypeSkip
	bDirectSubCarrier := MBType8x8 | MBTypeP0L0 | MBTypeP0L1 | MBTypeP1L0 | MBTypeP1L1
	bDirectSub := [4]uint32{
		MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2,
		MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2,
		MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2,
		MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2,
	}
	bMixedDirectSub := bDirectSub
	bMixedDirectSub[2] = MBType16x16 | MBTypeP0L0
	bMixedExplicitDirectSub := [4]uint32{
		MBType16x16 | MBTypeP0L0,
		MBType16x16 | MBTypeP0L1,
		MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2,
		MBType16x16 | MBTypeP0L0,
	}
	bExplicitSub := [4]uint32{
		MBType16x16 | MBTypeP0L0,
		MBType16x16 | MBTypeP0L0,
		MBType16x16 | MBTypeP0L0,
		MBType16x16 | MBTypeP0L0,
	}
	tests := []struct {
		name     string
		sh       *SliceHeader
		mbType   uint32
		sub      *[4]uint32
		cbp      int
		cbpTable int
		want     error
	}{
		{name: "nil header", sh: nil, mbType: MBType16x16 | MBTypeP0L0, want: ErrInvalidData},
		{name: "p skip cbp", sh: pSlice, mbType: pSkip, cbp: 1, cbpTable: 1, want: ErrUnsupported},
		{name: "p skip cbp table", sh: pSlice, mbType: pSkip, cbp: 0, cbpTable: 1, want: ErrUnsupported},
		{name: "p16x16 8x8 dct residual", sh: pSlice, mbType: MBType16x16 | MBTypeP0L0 | MBType8x8DCT, cbp: 1, cbpTable: 1, want: ErrUnsupported},
		{name: "direct in p", sh: pSlice, mbType: MBTypeDirect2 | MBTypeL0L1, cbp: 1, cbpTable: 1, want: ErrUnsupported},
		{name: "intra pcm in p", sh: pSlice, mbType: MBTypeIntraPCM, cbp: 1, cbpTable: 1, want: ErrUnsupported},
		{name: "intra 8x8 dct in p", sh: pSlice, mbType: MBTypeIntra4x4 | MBType8x8DCT, cbp: 1, cbpTable: 1, want: ErrUnsupported},
		{name: "intra in b", sh: bSlice, mbType: MBTypeIntra4x4, cbp: 1, cbpTable: 1, want: ErrUnsupported},
		{name: "negative cbp", sh: pSlice, mbType: MBType16x16 | MBTypeP0L0, cbp: -1, want: ErrUnsupported},
		{name: "negative cbp table", sh: pSlice, mbType: MBType16x16 | MBTypeP0L0, cbpTable: -1, want: ErrUnsupported},
		{name: "b direct", sh: bSlice, mbType: MBTypeDirect2 | MBTypeL0L1, cbp: 1, cbpTable: 1, want: ErrUnsupported},
		{name: "b direct missing l1", sh: bSlice, mbType: MBType16x16 | MBTypeP0L0 | MBTypeDirect2, want: ErrUnsupported},
		{name: "b direct partial temporal flags", sh: bSlice, mbType: MBType16x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeP1L0 | MBTypeDirect2, want: ErrUnsupported},
		{name: "b direct partition", sh: bSlice, mbType: MBType16x16 | MBType16x8 | MBTypeP0L0 | MBTypeP0L1 | MBTypeDirect2, want: ErrUnsupported},
		{name: "b direct skip cbp", sh: bSlice, mbType: bSkip, cbp: 1, want: ErrUnsupported},
		{name: "b direct skip cbp table", sh: bSlice, mbType: bSkip, cbpTable: 1, want: ErrUnsupported},
		{name: "b direct skip unresolved", sh: bSlice, mbType: MBTypeDirect2 | MBTypeL0L1 | MBTypeSkip, want: ErrUnsupported},
		{name: "b direct skip partition", sh: bSlice, mbType: MBType8x8 | MBTypeL0L1 | MBTypeDirect2 | MBTypeSkip, want: ErrUnsupported},
		{name: "b direct sub without sub types", sh: bSlice, mbType: bDirectSubCarrier, want: ErrUnsupported},
		{name: "b direct sub cbp", sh: bSlice, mbType: bDirectSubCarrier, sub: &bDirectSub, cbp: 1, want: ErrUnsupported},
		{name: "b direct sub cbp table", sh: bSlice, mbType: bDirectSubCarrier, sub: &bDirectSub, cbpTable: 1, want: ErrUnsupported},
		{name: "b mixed direct sub", sh: bSlice, mbType: bDirectSubCarrier, sub: &bMixedDirectSub, want: ErrUnsupported},
		{name: "b top-level direct 8x8 remains guarded", sh: bSlice, mbType: MBType8x8 | MBTypeL0L1 | MBTypeDirect2, sub: &bDirectSub, want: ErrUnsupported},
		{name: "b explicit direct sub mix remains guarded", sh: bSlice, mbType: bDirectSubCarrier, sub: &bMixedExplicitDirectSub, want: ErrUnsupported},
		{name: "b explicit 16x8 direct flag remains guarded", sh: bSlice, mbType: MBType16x8 | MBTypeP0L0 | MBTypeP1L0 | MBTypeDirect2, want: ErrUnsupported},
		{name: "b explicit 16x8 skip remains guarded", sh: bSlice, mbType: MBType16x8 | MBTypeP0L0 | MBTypeP1L0 | MBTypeSkip, want: ErrUnsupported},
		{name: "b explicit 16x8 missing partition direction", sh: bSlice, mbType: MBType16x8 | MBTypeP0L0, want: ErrUnsupported},
		{name: "b implicit weighted b16x8 residual remains guarded", sh: bImplicitWeightedSlice, mbType: MBType16x8 | MBTypeP0L0 | MBTypeP1L1, cbp: 1, cbpTable: 1, want: ErrUnsupported},
		{name: "b implicit weighted b8x16 residual remains guarded", sh: bImplicitWeightedSlice, mbType: MBType8x16 | MBTypeP0L0 | MBTypeP0L1 | MBTypeP1L0 | MBTypeP1L1, cbp: 3, cbpTable: 3, want: ErrUnsupported},
		{name: "b implicit weighted direct sub cbp remains guarded", sh: bImplicitWeightedSlice, mbType: bDirectSubCarrier, sub: &bDirectSub, cbp: 1, want: ErrUnsupported},
		{name: "b implicit weighted explicit direct sub mix remains guarded", sh: bImplicitWeightedSlice, mbType: bDirectSubCarrier, sub: &bMixedExplicitDirectSub, want: ErrUnsupported},
		{name: "b implicit weighted direct explicit sub mix remains guarded", sh: bImplicitWeightedSlice, mbType: bDirectSubCarrier, sub: &bMixedDirectSub, want: ErrUnsupported},
		{name: "b implicit weighted top-level direct 8x8 remains guarded", sh: bImplicitWeightedSlice, mbType: MBType8x8 | MBTypeL0L1 | MBTypeDirect2, sub: &bExplicitSub, want: ErrUnsupported},
		{name: "b deblock skip remains guarded", sh: bDeblockSlice, mbType: bSkip, want: ErrUnsupported},
		{name: "b deblock partitioned residual remains guarded", sh: bDeblockSlice, mbType: MBType16x8 | MBTypeP0L0 | MBTypeP1L0, cbp: 1, cbpTable: 1, want: ErrUnsupported},
		{name: "b deblock cabac partitioned residual remains guarded", sh: bCABACDeblockSlice, mbType: MBType16x8 | MBTypeP0L0 | MBTypeP1L0, cbp: 1, cbpTable: 1, want: ErrUnsupported},
		{name: "b deblock implicit weighted partitioned residual remains guarded", sh: bImplicitDeblockSlice, mbType: MBType16x8 | MBTypeP0L0 | MBTypeP1L0, cbp: 1, cbpTable: 1, want: ErrUnsupported},
		{name: "b deblock implicit weighted direct sub remains guarded", sh: bImplicitDeblockSlice, mbType: bDirectSubCarrier, sub: &bDirectSub, want: ErrUnsupported},
		{name: "b deblock implicit weighted remains guarded", sh: bImplicitDeblockSlice, mbType: MBType16x16 | MBTypeP0L0 | MBTypeP0L1, want: ErrUnsupported},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateHighFrameSliceMacroblockForReconstructWithSubMB(tt.sh, tt.mbType, tt.sub, tt.cbp, tt.cbpTable); err != tt.want {
				t.Fatalf("validate err = %v, want %v", err, tt.want)
			}
		})
	}
}
