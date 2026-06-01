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

func TestValidateHighFrameSliceMacroblockForReconstructRejectsPResidualGuardBoundaries(t *testing.T) {
	pSlice := &SliceHeader{SliceTypeNoS: PictureTypeP}
	bSlice := &SliceHeader{SliceTypeNoS: PictureTypeB}
	pSkip := MBType16x16 | MBTypeP0L0 | MBTypeP1L0 | MBTypeSkip

	tests := []struct {
		name     string
		sh       *SliceHeader
		mbType   uint32
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateHighFrameSliceMacroblockForReconstruct(tt.sh, tt.mbType, tt.cbp, tt.cbpTable); err != tt.want {
				t.Fatalf("validate err = %v, want %v", err, tt.want)
			}
		})
	}
}
