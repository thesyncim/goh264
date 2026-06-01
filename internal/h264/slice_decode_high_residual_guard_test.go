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

func TestValidateHighFrameSliceMacroblockForReconstructRejectsPResidualGuardBoundaries(t *testing.T) {
	pSlice := &SliceHeader{SliceTypeNoS: PictureTypeP}
	bSlice := &SliceHeader{SliceTypeNoS: PictureTypeB}
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
		{name: "p16x8 no residual", sh: pSlice, mbType: MBType16x8 | MBTypeP0L0 | MBTypeP1L0, want: ErrUnsupported},
		{name: "p16x8 residual", sh: pSlice, mbType: MBType16x8 | MBTypeP0L0 | MBTypeP1L0, cbp: 1, cbpTable: 1, want: ErrUnsupported},
		{name: "p8x16 no residual", sh: pSlice, mbType: MBType8x16 | MBTypeP0L0 | MBTypeP1L0, want: ErrUnsupported},
		{name: "p8x16 residual", sh: pSlice, mbType: MBType8x16 | MBTypeP0L0 | MBTypeP1L0, cbp: 2, cbpTable: 2, want: ErrUnsupported},
		{name: "p8x8 no residual", sh: pSlice, mbType: MBType8x8 | MBTypeP0L0 | MBTypeP1L0, want: ErrUnsupported},
		{name: "p8x8 residual", sh: pSlice, mbType: MBType8x8 | MBTypeP0L0 | MBTypeP1L0, cbp: 4, cbpTable: 4, want: ErrUnsupported},
		{name: "p8x8 ref0 residual", sh: pSlice, mbType: MBType8x8 | MBTypeP0L0 | MBTypeP1L0 | MBTypeRef0, cbp: 8, cbpTable: 8, want: ErrUnsupported},
		{name: "p16x16 8x8 dct residual", sh: pSlice, mbType: MBType16x16 | MBTypeP0L0 | MBType8x8DCT, cbp: 1, cbpTable: 1, want: ErrUnsupported},
		{name: "direct in p", sh: pSlice, mbType: MBTypeDirect2 | MBTypeL0L1, cbp: 1, cbpTable: 1, want: ErrUnsupported},
		{name: "intra in p", sh: pSlice, mbType: MBTypeIntra4x4, cbp: 1, cbpTable: 1, want: ErrUnsupported},
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateHighFrameSliceMacroblockForReconstructWithSubMB(tt.sh, tt.mbType, tt.sub, tt.cbp, tt.cbpTable); err != tt.want {
				t.Fatalf("validate err = %v, want %v", err, tt.want)
			}
		})
	}
}
