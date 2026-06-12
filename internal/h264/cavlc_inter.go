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

type cavlcInterBDirectHook func(*cavlcInterMacroblockSyntax) error

func writeCAVLCPSubMBType(bw *BitWriter, info PMBInfo) error {
	if bw == nil {
		return ErrInvalidData
	}
	for raw, tableInfo := range h264PSubMBTypeInfo {
		if tableInfo == info {
			return bw.WriteUEGolomb(uint32(raw))
		}
	}
	return ErrInvalidData
}

func writeCAVLCBSubMBType(bw *BitWriter, info PMBInfo) error {
	if bw == nil {
		return ErrInvalidData
	}
	for raw, tableInfo := range h264BSubMBTypeInfo {
		if tableInfo == info {
			return bw.WriteUEGolomb(uint32(raw))
		}
	}
	return ErrInvalidData
}

func writeCAVLCRefIndex(bw *BitWriter, refCount uint32, ref int32) error {
	if bw == nil || refCount == 0 || ref < 0 || uint32(ref) >= refCount {
		return ErrInvalidData
	}
	if refCount == 1 {
		return nil
	}
	if refCount == 2 {
		bw.WriteBit(uint32(ref) ^ 1)
		return nil
	}
	return bw.WriteUEGolomb(uint32(ref))
}

func writeCAVLCMVD(bw *BitWriter, mvd [2]int32) error {
	if bw == nil {
		return ErrInvalidData
	}
	if err := bw.WriteSEGolomb(mvd[0]); err != nil {
		return err
	}
	return bw.WriteSEGolomb(mvd[1])
}

func writeCAVLCInterPNoResidualMacroblock(bw *BitWriter, mb cavlcInterMacroblockSyntax, refCount [2]uint32, decodeChroma bool) error {
	if bw == nil {
		return ErrInvalidData
	}
	if isIntra(mb.MBType) || mb.CBP != 0 {
		return ErrUnsupported
	}
	if err := writeCAVLCInterPMacroblockMotion(bw, mb, refCount); err != nil {
		return err
	}

	return writeCAVLCCBP(bw, mb.MBType, decodeChroma, 0)
}

func writeCAVLCInterPBoundedMacroblock(bw *BitWriter, residual *cavlcResidualContext, pps *PPS, sps *SPS, mb cavlcInterMacroblockSyntax, refCount [2]uint32, qscale int, nextQScale int) (int, error) {
	if bw == nil || residual == nil || pps == nil || sps == nil {
		return 0, ErrInvalidData
	}
	if isIntra(mb.MBType) {
		return 0, ErrUnsupported
	}
	if mb.CBP == 0 {
		if err := writeCAVLCInterPNoResidualMacroblock(bw, mb, refCount, sps.ChromaFormatIDC == 1 || sps.ChromaFormatIDC == 2); err != nil {
			return 0, err
		}
		cbpTable, err := residual.writeCAVLCInterResidualPayload(bw, pps, sps, mb.MBType, 0, qscale, nextQScale)
		return cbpTable, err
	}
	if err := writeCAVLCInterPMacroblockMotion(bw, mb, refCount); err != nil {
		return 0, err
	}
	if err := writeCAVLCCBP(bw, mb.MBType, sps.ChromaFormatIDC == 1 || sps.ChromaFormatIDC == 2, mb.CBP); err != nil {
		return 0, err
	}
	return residual.writeCAVLCInterResidualPayload(bw, pps, sps, mb.MBType, mb.CBP, qscale, nextQScale)
}

func writeCAVLCInterPMacroblockMotion(bw *BitWriter, mb cavlcInterMacroblockSyntax, refCount [2]uint32) error {
	if bw == nil {
		return ErrInvalidData
	}
	if err := writeCAVLCMBType(bw, PictureTypeP, PictureTypeP, mb.cavlcMacroblockSyntax); err != nil {
		return err
	}

	if mb.PartitionCount == 4 {
		for i := 0; i < 4; i++ {
			info := PMBInfo{Type: mb.SubMBType[i], PartitionCount: mb.SubPartitionCount[i]}
			if err := writeCAVLCPSubMBType(bw, info); err != nil {
				return err
			}
		}

		refTotal := refCount[0]
		if isRef0(mb.MBType) {
			refTotal = 1
		}
		for i := 0; i < 4; i++ {
			if isDir(mb.SubMBType[i], 0, 0) {
				if err := writeCAVLCRefIndex(bw, refTotal, mb.Ref[0][i]); err != nil {
					return err
				}
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
				if err := writeCAVLCMVD(bw, mb.MVD[0][index]); err != nil {
					return err
				}
			}
		}
	} else if is16x16(mb.MBType) {
		if isDir(mb.MBType, 0, 0) {
			if err := writeCAVLCRefIndex(bw, refCount[0], mb.Ref[0][0]); err != nil {
				return err
			}
			if err := writeCAVLCMVD(bw, mb.MVD[0][0]); err != nil {
				return err
			}
		}
	} else if is16x8(mb.MBType) {
		for i := 0; i < 2; i++ {
			if isDir(mb.MBType, i, 0) {
				if err := writeCAVLCRefIndex(bw, refCount[0], mb.Ref[0][i]); err != nil {
					return err
				}
			}
		}
		for i := 0; i < 2; i++ {
			if isDir(mb.MBType, i, 0) {
				if err := writeCAVLCMVD(bw, mb.MVD[0][8*i]); err != nil {
					return err
				}
			}
		}
	} else if is8x16(mb.MBType) {
		for i := 0; i < 2; i++ {
			if isDir(mb.MBType, i, 0) {
				if err := writeCAVLCRefIndex(bw, refCount[0], mb.Ref[0][i]); err != nil {
					return err
				}
			}
		}
		for i := 0; i < 2; i++ {
			if isDir(mb.MBType, i, 0) {
				if err := writeCAVLCMVD(bw, mb.MVD[0][4*i]); err != nil {
					return err
				}
			}
		}
	} else {
		return ErrUnsupported
	}
	return nil
}

func writeCAVLCInterBNoResidualMacroblock(bw *BitWriter, mb cavlcInterMacroblockSyntax, refCount [2]uint32, decodeChroma bool) error {
	if bw == nil {
		return ErrInvalidData
	}
	if isIntra(mb.MBType) || mb.CBP != 0 {
		return ErrUnsupported
	}
	if err := writeCAVLCInterBMacroblockMotion(bw, mb, refCount); err != nil {
		return err
	}

	return writeCAVLCCBP(bw, mb.MBType, decodeChroma, 0)
}

func writeCAVLCInterBBoundedMacroblock(bw *BitWriter, residual *cavlcResidualContext, pps *PPS, sps *SPS, mb cavlcInterMacroblockSyntax, refCount [2]uint32, qscale int, nextQScale int) (int, error) {
	if bw == nil || residual == nil || pps == nil || sps == nil {
		return 0, ErrInvalidData
	}
	if isIntra(mb.MBType) {
		return 0, ErrUnsupported
	}
	if mb.CBP == 0 {
		if err := writeCAVLCInterBNoResidualMacroblock(bw, mb, refCount, sps.ChromaFormatIDC == 1 || sps.ChromaFormatIDC == 2); err != nil {
			return 0, err
		}
		cbpTable, err := residual.writeCAVLCInterResidualPayload(bw, pps, sps, mb.MBType, 0, qscale, nextQScale)
		return cbpTable, err
	}
	if err := writeCAVLCInterBMacroblockMotion(bw, mb, refCount); err != nil {
		return 0, err
	}
	if err := writeCAVLCCBP(bw, mb.MBType, sps.ChromaFormatIDC == 1 || sps.ChromaFormatIDC == 2, mb.CBP); err != nil {
		return 0, err
	}
	return residual.writeCAVLCInterResidualPayload(bw, pps, sps, mb.MBType, mb.CBP, qscale, nextQScale)
}

func writeCAVLCInterBMacroblockMotion(bw *BitWriter, mb cavlcInterMacroblockSyntax, refCount [2]uint32) error {
	if bw == nil {
		return ErrInvalidData
	}
	if err := writeCAVLCMBType(bw, PictureTypeB, PictureTypeB, mb.cavlcMacroblockSyntax); err != nil {
		return err
	}

	if isDirect(mb.MBType) {
		// B_Direct carries no ref-index or MVD syntax.
	} else if mb.PartitionCount == 4 {
		for i := 0; i < 4; i++ {
			info := PMBInfo{Type: mb.SubMBType[i], PartitionCount: mb.SubPartitionCount[i]}
			if err := writeCAVLCBSubMBType(bw, info); err != nil {
				return err
			}
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
					if err := writeCAVLCRefIndex(bw, refTotal, mb.Ref[list][i]); err != nil {
						return err
					}
				}
			}
		}

		for list := 0; list < 2; list++ {
			for i := 0; i < 4; i++ {
				if isDirect(mb.SubMBType[i]) || !isDir(mb.SubMBType[i], 0, list) {
					continue
				}
				blockWidth := 1
				if mb.SubMBType[i]&(MBType16x16|MBType16x8) != 0 {
					blockWidth = 2
				}
				for j := 0; j < int(mb.SubPartitionCount[i]); j++ {
					index := 4*i + blockWidth*j
					if err := writeCAVLCMVD(bw, mb.MVD[list][index]); err != nil {
						return err
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
					if err := writeCAVLCRefIndex(bw, refCount[list], mb.Ref[list][i]); err != nil {
						return err
					}
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
				if err := writeCAVLCMVD(bw, mb.MVD[list][index]); err != nil {
					return err
				}
			}
		}
	} else {
		return ErrUnsupported
	}
	return nil
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
	return c.decodeCAVLCInterBMacroblockAfterTypeWithDirectHook(gb, pps, sps, mb, qscale, refCount, dct8x8Allowed, nil)
}

func (c *cavlcResidualContext) decodeCAVLCInterBMacroblockAfterTypeWithDirectHook(gb *bitReader, pps *PPS, sps *SPS, mb cavlcInterMacroblockSyntax, qscale int, refCount [2]uint32, dct8x8Allowed bool, directHook cavlcInterBDirectHook) (cavlcInterMacroblockSyntax, error) {
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

	if directHook != nil && (isDirect(mb.MBType) || mb.PartitionCount == 4 && hasDirectSubMBType(&mb.SubMBType)) {
		if err := directHook(&mb); err != nil {
			return mb, err
		}
	}

	cbp, err := decodeCAVLCCBP(gb, mb.MBType, sps.ChromaFormatIDC == 1 || sps.ChromaFormatIDC == 2, mb.CBP)
	if err != nil {
		return mb, err
	}
	mb.CBP = cbp
	if isDirect(mb.MBType) {
		dct8x8Allowed = dct8x8Allowed && sps.Direct8x8InferenceFlag != 0
	}
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
