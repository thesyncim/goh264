// SPDX-License-Identifier: LGPL-2.1-or-later
//
// Source-shaped port of the P-slice inter syntax portions of FFmpeg n8.0.1
// libavcodec/h264_cavlc.c ff_h264_decode_mb_cavlc. Motion prediction and
// reference-list application are intentionally left to the h264_mvpred/h264_refs
// port; this layer preserves the bitstream syntax and decoded deltas.

package h264

type cavlcInterMacroblockSyntax struct {
	cavlcMacroblockSyntax
	SubMBType         [4]uint32
	SubPartitionCount [4]uint8
	Ref               [2][4]int32
	MVD               [2][16][2]int32
}

func (c *cavlcResidualContext) decodeCAVLCInterPMacroblock(gb *bitReader, pps *PPS, sps *SPS, qscale int, refCount [2]uint32, dct8x8Allowed bool) (cavlcInterMacroblockSyntax, error) {
	var mb cavlcInterMacroblockSyntax
	base, err := decodeCAVLCMBType(gb, PictureTypeP, PictureTypeP)
	if err != nil {
		return mb, err
	}
	mb.cavlcMacroblockSyntax = base
	return c.decodeCAVLCInterPMacroblockAfterType(gb, pps, sps, mb, qscale, refCount, dct8x8Allowed)
}

func (c *cavlcResidualContext) decodeCAVLCInterPMacroblockAfterType(gb *bitReader, pps *PPS, sps *SPS, mb cavlcInterMacroblockSyntax, qscale int, refCount [2]uint32, dct8x8Allowed bool) (cavlcInterMacroblockSyntax, error) {
	if gb == nil || pps == nil || sps == nil {
		return mb, ErrInvalidData
	}
	if isIntra(mb.MBType) {
		return mb, ErrUnsupported
	}

	if mb.PartitionCount == 4 {
		for i := 0; i < 4; i++ {
			subType, err := gb.readUEGolomb31()
			if err != nil {
				return mb, err
			}
			if subType >= 4 {
				return mb, ErrInvalidData
			}
			info := h264PSubMBTypeInfo[subType]
			mb.SubPartitionCount[i] = info.PartitionCount
			mb.SubMBType[i] = info.Type
		}

		refTotal := refCount[0]
		if isRef0(mb.MBType) {
			refTotal = 1
		}
		for i := 0; i < 4; i++ {
			if isDir(mb.SubMBType[i], 0, 0) {
				ref, err := readCAVLCRefIndex(gb, refTotal)
				if err != nil {
					return mb, err
				}
				mb.Ref[0][i] = ref
			} else {
				mb.Ref[0][i] = -1
			}
		}

		for i := 0; i < 4; i++ {
			if !isDir(mb.SubMBType[i], 0, 0) {
				continue
			}
			blockWidth := 1
			if mb.SubMBType[i]&(MBType16x16|MBType16x8) != 0 {
				blockWidth = 2
			}
			for j := 0; j < int(mb.SubPartitionCount[i]); j++ {
				index := 4*i + blockWidth*j
				if err := readCAVLCMVD(gb, &mb.MVD[0][index]); err != nil {
					return mb, err
				}
			}
		}
	} else if is16x16(mb.MBType) {
		if isDir(mb.MBType, 0, 0) {
			ref, err := readCAVLCRefIndex(gb, refCount[0])
			if err != nil {
				return mb, err
			}
			mb.Ref[0][0] = ref
			if err := readCAVLCMVD(gb, &mb.MVD[0][0]); err != nil {
				return mb, err
			}
		}
	} else if is16x8(mb.MBType) {
		for i := 0; i < 2; i++ {
			if isDir(mb.MBType, i, 0) {
				ref, err := readCAVLCRefIndex(gb, refCount[0])
				if err != nil {
					return mb, err
				}
				mb.Ref[0][i] = ref
			} else {
				mb.Ref[0][i] = -1
			}
		}
		for i := 0; i < 2; i++ {
			if isDir(mb.MBType, i, 0) {
				if err := readCAVLCMVD(gb, &mb.MVD[0][8*i]); err != nil {
					return mb, err
				}
			}
		}
	} else if is8x16(mb.MBType) {
		for i := 0; i < 2; i++ {
			if isDir(mb.MBType, i, 0) {
				ref, err := readCAVLCRefIndex(gb, refCount[0])
				if err != nil {
					return mb, err
				}
				mb.Ref[0][i] = ref
			} else {
				mb.Ref[0][i] = -1
			}
		}
		for i := 0; i < 2; i++ {
			if isDir(mb.MBType, i, 0) {
				if err := readCAVLCMVD(gb, &mb.MVD[0][4*i]); err != nil {
					return mb, err
				}
			}
		}
	} else {
		return mb, ErrUnsupported
	}

	cbp, err := decodeCAVLCCBP(gb, mb.MBType, sps.ChromaFormatIDC == 1 || sps.ChromaFormatIDC == 2, mb.CBP)
	if err != nil {
		return mb, err
	}
	mb.CBP = cbp
	if mb.PartitionCount == 4 {
		dct8x8Allowed = subMBTypesAllowDCT8x8(dct8x8Allowed, &mb.SubMBType, true)
	}
	if dct8x8Allowed && (mb.CBP&15) != 0 {
		flag, err := gb.readBit()
		if err != nil {
			return mb, err
		}
		if flag != 0 {
			mb.MBType |= MBType8x8DCT
			mb.TransformSize8x8DCT = true
		}
	}

	mb.QScale, mb.ChromaQP, mb.CBPTable, err = c.decodeCAVLCResidualPayload(gb, pps, sps, mb.MBType, mb.CBP, qscale)
	if err != nil {
		return mb, err
	}
	return mb, nil
}

func (c *cavlcResidualContext) decodeCAVLCInterBMacroblock(gb *bitReader, pps *PPS, sps *SPS, qscale int, refCount [2]uint32, dct8x8Allowed bool) (cavlcInterMacroblockSyntax, error) {
	var mb cavlcInterMacroblockSyntax
	base, err := decodeCAVLCMBType(gb, PictureTypeB, PictureTypeB)
	if err != nil {
		return mb, err
	}
	mb.cavlcMacroblockSyntax = base
	return c.decodeCAVLCInterBMacroblockAfterType(gb, pps, sps, mb, qscale, refCount, dct8x8Allowed)
}

func (c *cavlcResidualContext) decodeCAVLCInterBMacroblockAfterType(gb *bitReader, pps *PPS, sps *SPS, mb cavlcInterMacroblockSyntax, qscale int, refCount [2]uint32, dct8x8Allowed bool) (cavlcInterMacroblockSyntax, error) {
	if gb == nil || pps == nil || sps == nil {
		return mb, ErrInvalidData
	}
	for list := 0; list < 2; list++ {
		for i := 0; i < 4; i++ {
			mb.Ref[list][i] = -1
		}
	}
	if isIntra(mb.MBType) {
		return mb, ErrUnsupported
	}

	if isDirect(mb.MBType) {
		// B_Direct carries no ref-index or MVD syntax; h264_direct.c derives
		// the concrete partition shape and motion vectors after the residual
		// syntax has been read by the frame-MB handoff.
	} else if mb.PartitionCount == 4 {
		for i := 0; i < 4; i++ {
			subType, err := gb.readUEGolomb31()
			if err != nil {
				return mb, err
			}
			if subType >= 13 {
				return mb, ErrInvalidData
			}
			info := h264BSubMBTypeInfo[subType]
			if isDirect(info.Type) {
				return mb, ErrUnsupported
			}
			mb.SubPartitionCount[i] = info.PartitionCount
			mb.SubMBType[i] = info.Type
		}

		for list := 0; list < 2; list++ {
			refTotal := refCount[list]
			if isRef0(mb.MBType) {
				refTotal = 1
			}
			for i := 0; i < 4; i++ {
				if isDirect(mb.SubMBType[i]) {
					continue
				}
				if isDir(mb.SubMBType[i], 0, list) {
					ref, err := readCAVLCRefIndex(gb, refTotal)
					if err != nil {
						return mb, err
					}
					mb.Ref[list][i] = ref
				}
			}
		}

		for list := 0; list < 2; list++ {
			for i := 0; i < 4; i++ {
				if isDirect(mb.SubMBType[i]) {
					continue
				}
				if !isDir(mb.SubMBType[i], 0, list) {
					continue
				}
				blockWidth := 1
				if mb.SubMBType[i]&(MBType16x16|MBType16x8) != 0 {
					blockWidth = 2
				}
				for j := 0; j < int(mb.SubPartitionCount[i]); j++ {
					index := 4*i + blockWidth*j
					if err := readCAVLCMVD(gb, &mb.MVD[list][index]); err != nil {
						return mb, err
					}
				}
			}
		}
	} else if is16x16(mb.MBType) || is16x8(mb.MBType) || is8x16(mb.MBType) {
		partitions := 1
		if is16x8(mb.MBType) || is8x16(mb.MBType) {
			partitions = 2
		}
		for list := 0; list < 2; list++ {
			for i := 0; i < partitions; i++ {
				if isDir(mb.MBType, i, list) {
					ref, err := readCAVLCRefIndex(gb, refCount[list])
					if err != nil {
						return mb, err
					}
					mb.Ref[list][i] = ref
				}
			}
		}
		for list := 0; list < 2; list++ {
			for i := 0; i < partitions; i++ {
				if !isDir(mb.MBType, i, list) {
					continue
				}
				index := 0
				if is16x8(mb.MBType) {
					index = 8 * i
				} else if is8x16(mb.MBType) {
					index = 4 * i
				}
				if err := readCAVLCMVD(gb, &mb.MVD[list][index]); err != nil {
					return mb, err
				}
			}
		}
	} else {
		return mb, ErrUnsupported
	}

	cbp, err := decodeCAVLCCBP(gb, mb.MBType, sps.ChromaFormatIDC == 1 || sps.ChromaFormatIDC == 2, mb.CBP)
	if err != nil {
		return mb, err
	}
	mb.CBP = cbp
	if mb.PartitionCount == 4 {
		dct8x8Allowed = subMBTypesAllowDCT8x8(dct8x8Allowed, &mb.SubMBType, sps.Direct8x8InferenceFlag != 0)
	}
	if dct8x8Allowed && (mb.CBP&15) != 0 {
		flag, err := gb.readBit()
		if err != nil {
			return mb, err
		}
		if flag != 0 {
			mb.MBType |= MBType8x8DCT
			mb.TransformSize8x8DCT = true
		}
	}

	mb.QScale, mb.ChromaQP, mb.CBPTable, err = c.decodeCAVLCResidualPayload(gb, pps, sps, mb.MBType, mb.CBP, qscale)
	if err != nil {
		return mb, err
	}
	return mb, nil
}

func readCAVLCRefIndex(gb *bitReader, refCount uint32) (int32, error) {
	if refCount == 0 {
		return 0, ErrInvalidData
	}
	if refCount == 1 {
		return 0, nil
	}
	if refCount == 2 {
		bit, err := gb.readBit()
		if err != nil {
			return 0, err
		}
		return int32(bit ^ 1), nil
	}
	ref, err := gb.readUEGolomb31()
	if err != nil {
		return 0, err
	}
	if ref >= refCount {
		return 0, ErrInvalidData
	}
	return int32(ref), nil
}

func readCAVLCMVD(gb *bitReader, dst *[2]int32) error {
	mx, err := gb.readSEGolombLong()
	if err != nil {
		return err
	}
	my, err := gb.readSEGolombLong()
	if err != nil {
		return err
	}
	dst[0] = mx
	dst[1] = my
	return nil
}

func hasDirectSubMBType(sub *[4]uint32) bool {
	if sub == nil {
		return false
	}
	return isDirect(sub[0] | sub[1] | sub[2] | sub[3])
}

func subMBTypesAllowDCT8x8(allowed bool, sub *[4]uint32, direct8x8Inference bool) bool {
	if !allowed || sub == nil {
		return false
	}
	mask := MBType16x8 | MBType8x16 | MBType8x8
	if !direct8x8Inference {
		mask |= MBTypeDirect2
	}
	for i := 0; i < 4; i++ {
		if sub[i]&mask != 0 {
			return false
		}
	}
	return true
}

func isDir(mbType uint32, part int, list int) bool {
	if list == 0 {
		if part == 0 {
			return mbType&MBTypeP0L0 != 0
		}
		return mbType&MBTypeP1L0 != 0
	}
	if part == 0 {
		return mbType&MBTypeP0L1 != 0
	}
	return mbType&MBTypeP1L1 != 0
}

func isRef0(mbType uint32) bool {
	return mbType&MBTypeRef0 != 0
}

func isDirect(mbType uint32) bool {
	return mbType&MBTypeDirect2 != 0
}

func is16x16(mbType uint32) bool {
	return mbType&MBType16x16 != 0
}

func is16x8(mbType uint32) bool {
	return mbType&MBType16x8 != 0
}

func is8x16(mbType uint32) bool {
	return mbType&MBType8x16 != 0
}
