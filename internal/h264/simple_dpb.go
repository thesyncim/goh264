// SPDX-License-Identifier: LGPL-2.1-or-later
//
// Source-shaped simple reference DPB/ref-list subset from FFmpeg
// n8.0.1 libavcodec/h264_refs.c h264_initialise_ref_list,
// ff_h264_build_ref_list, and ff_h264_execute_ref_pic_marking. This file is
// intentionally limited to progressive frame-picture refs for the
// already-translated simple frame-MB path.

package h264

import "math"

const simpleMaxShortRefs = 16
const simpleMaxLongRefs = 16

const (
	simpleFrameRecoveredIDR uint8 = 1 << iota
	simpleFrameRecoveredSEI
)

type simpleFrameDPB struct {
	short              []*DecodedFrame
	long               [simpleMaxLongRefs]*DecodedFrame
	poc                simplePOCContext
	delayed            []*DecodedFrame
	hasBFrames         int
	lastPOCs           [h264MaxDPBFrames]int32
	lastPOCsInit       bool
	nextOutputedPOC    int32
	nextOutputedValid  bool
	prevInterlaced     bool
	prevInterlacedSet  bool
	validRecoveryPoint bool
	recoveryFrame      int32
	frameRecovered     uint8
}

type simpleRefEntry struct {
	frame *DecodedFrame
	picID uint32
	long  bool
}

type simpleFrameRefContext struct {
	Refs    [2][]*h264PicturePlanes
	Entries [2][]simpleRefEntry
}

type simplePOCContext struct {
	pocLSB             int32
	pocMSB             int32
	deltaPOCBottom     int32
	deltaPOC           [2]int32
	frameNum           int32
	prevPOCMSB         int32
	prevPOCLSB         int32
	frameNumOffset     int32
	prevFrameNumOffset int32
	prevFrameNum       int32
}

func (d *simpleFrameDPB) reset() {
	if d != nil {
		d.resetRefs()
		d.poc.reset()
		d.delayed = d.delayed[:0]
		d.hasBFrames = 0
		d.resetLastPOCs()
		d.nextOutputedPOC = 0
		d.nextOutputedValid = false
		d.prevInterlaced = true
		d.prevInterlacedSet = true
		d.validRecoveryPoint = false
		d.recoveryFrame = -1
		d.frameRecovered = 0
	}
}

func (d *simpleFrameDPB) previousInterlacedFrame() bool {
	if d == nil {
		return true
	}
	if !d.prevInterlacedSet {
		d.prevInterlaced = true
		d.prevInterlacedSet = true
	}
	return d.prevInterlaced
}

func (d *simpleFrameDPB) setPreviousInterlacedFrame(v bool) {
	if d == nil {
		return
	}
	d.prevInterlaced = v
	d.prevInterlacedSet = true
}

// applySimpleRecoveryPoint mirrors the frame-picture portion of FFmpeg n8.0.1
// libavcodec/h264_slice.c around recovery_frame tracking and IDR recovery
// marks. The public key flag additionally mirrors h264dec.c output_frame's
// recovery_frame_cnt == 0 promotion.
func (d *simpleFrameDPB) applySimpleRecoveryPoint(frame *DecodedFrame, sh *SliceHeader, nalRefIDC uint8, sei *H264SEIContext) {
	if d == nil || frame == nil || sh == nil || sh.SPS == nil {
		return
	}
	recoveryFrameCount := int32(-1)
	if sei != nil {
		recoveryFrameCount = sei.RecoveryPoint.RecoveryFrameCount
	}
	if recoveryFrameCount >= 0 {
		if int32(sh.FrameNum) != recoveryFrameCount || sh.SliceTypeNoS != PictureTypeI {
			d.validRecoveryPoint = true
		}
		if d.recoveryFrame < 0 ||
			avZeroExtendSimple(d.recoveryFrame-int32(sh.FrameNum), sh.SPS.Log2MaxFrameNum) > uint32(recoveryFrameCount) {
			d.recoveryFrame = int32(avZeroExtendSimple(int32(sh.FrameNum)+recoveryFrameCount, sh.SPS.Log2MaxFrameNum))
			if !d.validRecoveryPoint {
				d.recoveryFrame = int32(sh.FrameNum)
			}
		}
		if recoveryFrameCount == 0 {
			frame.KeyFrame = true
		}
	}

	if sh.NALType == NALIDRSlice {
		frame.KeyFrame = true
		frame.idrKeyFrame = true
		frame.recovered |= simpleFrameRecoveredIDR
		d.frameRecovered |= simpleFrameRecoveredIDR
	}
	if d.recoveryFrame == int32(sh.FrameNum) && nalRefIDC != 0 {
		d.recoveryFrame = -1
		frame.recovered |= simpleFrameRecoveredSEI
	}
	frame.recovered |= d.frameRecovered
}

func avZeroExtendSimple(v int32, bits int32) uint32 {
	if bits <= 0 {
		return 0
	}
	if bits >= 32 {
		return uint32(v)
	}
	return uint32(v) & ((uint32(1) << uint32(bits)) - 1)
}

func (d *simpleFrameDPB) resetRefs() {
	if d != nil {
		d.short = d.short[:0]
		for i := range d.long {
			d.long[i] = nil
		}
	}
}

func (p *simplePOCContext) reset() {
	if p == nil {
		return
	}
	*p = simplePOCContext{
		prevPOCMSB:   1 << 16,
		prevPOCLSB:   -1,
		prevFrameNum: -1,
	}
}

func (d *simpleFrameDPB) initFramePOC(frame *DecodedFrame, sh *SliceHeader, nalRefIDC uint8) error {
	if d == nil || frame == nil || sh == nil || sh.SPS == nil {
		return ErrInvalidData
	}
	if sh.PictureStructure != PictureFrame {
		return ErrUnsupported
	}
	if sh.NALType == NALIDRSlice {
		d.resetRefs()
		d.poc.reset()
		d.resetLastPOCs()
	}
	d.poc.frameNum = int32(sh.FrameNum)
	d.poc.pocLSB = int32(sh.POCLSB)
	d.poc.deltaPOCBottom = sh.DeltaPOCBottom
	d.poc.deltaPOC = sh.DeltaPOC
	fieldPOC, poc, err := d.poc.initPOC(sh.SPS, sh.PictureStructure, nalRefIDC)
	if err != nil {
		return err
	}
	frame.fieldPOC = fieldPOC
	frame.poc = poc
	return nil
}

// initPOC is a source-shaped port of FFmpeg n8.0.1 libavcodec/h264_parse.c
// ff_h264_init_poc for the progressive frame-picture subset.
func (p *simplePOCContext) initPOC(sps *SPS, pictureStructure int32, nalRefIDC uint8) ([2]int32, int32, error) {
	var out [2]int32
	if p == nil || sps == nil {
		return out, 0, ErrInvalidData
	}
	maxFrameNum := int32(1) << uint32(sps.Log2MaxFrameNum)
	fieldPOC := [2]int64{}

	p.frameNumOffset = p.prevFrameNumOffset
	if p.frameNum < p.prevFrameNum {
		p.frameNumOffset += maxFrameNum
	}

	switch sps.PocType {
	case 0:
		maxPOCLSB := int32(1) << uint32(sps.Log2MaxPocLSB)
		if p.prevPOCLSB < 0 {
			p.prevPOCLSB = p.pocLSB
		}
		if p.pocLSB < p.prevPOCLSB && p.prevPOCLSB-p.pocLSB >= maxPOCLSB/2 {
			p.pocMSB = p.prevPOCMSB + maxPOCLSB
		} else if p.pocLSB > p.prevPOCLSB && p.prevPOCLSB-p.pocLSB < -maxPOCLSB/2 {
			p.pocMSB = p.prevPOCMSB - maxPOCLSB
		} else {
			p.pocMSB = p.prevPOCMSB
		}
		fieldPOC[0] = int64(p.pocMSB + p.pocLSB)
		fieldPOC[1] = fieldPOC[0]
		if pictureStructure == PictureFrame {
			fieldPOC[1] += int64(p.deltaPOCBottom)
		}
	case 1:
		absFrameNum := int32(0)
		if sps.PocCycleLength != 0 {
			absFrameNum = p.frameNumOffset + p.frameNum
		}
		if nalRefIDC == 0 && absFrameNum > 0 {
			absFrameNum--
		}
		expectedDeltaPerPOCCycle := int64(0)
		for i := uint32(0); i < sps.PocCycleLength; i++ {
			expectedDeltaPerPOCCycle += int64(sps.OffsetForRefFrame[i])
		}

		expectedPOC := int64(0)
		if absFrameNum > 0 {
			pocCycleCnt := int64(absFrameNum-1) / int64(sps.PocCycleLength)
			frameNumInPOCCycle := uint32((absFrameNum - 1) % int32(sps.PocCycleLength))
			expectedPOC = pocCycleCnt * expectedDeltaPerPOCCycle
			for i := uint32(0); i <= frameNumInPOCCycle; i++ {
				expectedPOC += int64(sps.OffsetForRefFrame[i])
			}
		}
		if nalRefIDC == 0 {
			expectedPOC += int64(sps.OffsetForNonRefPic)
		}
		fieldPOC[0] = expectedPOC + int64(p.deltaPOC[0])
		fieldPOC[1] = fieldPOC[0] + int64(sps.OffsetForTopToBottomField)
		if pictureStructure == PictureFrame {
			fieldPOC[1] += int64(p.deltaPOC[1])
		}
	case 2:
		poc := int64(2 * (p.frameNumOffset + p.frameNum))
		if nalRefIDC == 0 {
			poc--
		}
		fieldPOC[0] = poc
		fieldPOC[1] = poc
	default:
		return out, 0, ErrInvalidData
	}

	if fieldPOC[0] < math.MinInt32 || fieldPOC[0] > math.MaxInt32 ||
		fieldPOC[1] < math.MinInt32 || fieldPOC[1] > math.MaxInt32 {
		return out, 0, ErrInvalidData
	}
	if pictureStructure != PictureBottomField {
		out[0] = int32(fieldPOC[0])
	}
	if pictureStructure != PictureTopField {
		out[1] = int32(fieldPOC[1])
	}
	picPOC := out[0]
	if out[1] < picPOC {
		picPOC = out[1]
	}
	return out, picPOC, nil
}

func (d *simpleFrameDPB) finishFramePOC(nalRefIDC uint8) {
	if d == nil {
		return
	}
	if nalRefIDC != 0 {
		d.poc.prevPOCMSB = d.poc.pocMSB
		d.poc.prevPOCLSB = d.poc.pocLSB
	}
	d.poc.prevFrameNumOffset = d.poc.frameNumOffset
	d.poc.prevFrameNum = d.poc.frameNum
}

func (d *simpleFrameDPB) buildRefLists(sh *SliceHeader, frame *DecodedFrame) ([2][]*h264PicturePlanes, error) {
	ctx, err := d.buildRefContext(sh, frame)
	if err != nil {
		return [2][]*h264PicturePlanes{}, err
	}
	return ctx.Refs, nil
}

func (d *simpleFrameDPB) buildRefContext(sh *SliceHeader, frame *DecodedFrame) (simpleFrameRefContext, error) {
	var ctx simpleFrameRefContext
	var refs [2][]*h264PicturePlanes
	if d == nil || sh == nil || sh.SPS == nil {
		return ctx, ErrInvalidData
	}
	if sh.SliceTypeNoS == PictureTypeI {
		return ctx, nil
	}
	if sh.PictureStructure != PictureFrame {
		return ctx, ErrUnsupported
	}

	switch sh.SliceTypeNoS {
	case PictureTypeP:
		list, err := d.buildPRefEntries(sh)
		if err != nil {
			return ctx, err
		}
		ctx.Entries[0] = cloneSimpleRefEntries(list)
		refs[0] = simpleFrameEntryPlanesRefs(ctx.Entries[0])
	case PictureTypeB:
		if frame == nil {
			return ctx, ErrInvalidData
		}
		lists, err := d.buildBRefEntries(sh, frame.poc)
		if err != nil {
			return ctx, err
		}
		if sh.PPS != nil && sh.PPS.WeightedBipredIDC == 2 {
			if err := initImplicitBWeightTable(&sh.PredWeightTable, lists, sh.RefCount, frame.poc); err != nil {
				return ctx, err
			}
		}
		ctx.Entries[0] = cloneSimpleRefEntries(lists[0])
		ctx.Entries[1] = cloneSimpleRefEntries(lists[1])
		refs[0] = simpleFrameEntryPlanesRefs(ctx.Entries[0])
		refs[1] = simpleFrameEntryPlanesRefs(ctx.Entries[1])
	default:
		return ctx, ErrUnsupported
	}
	ctx.Refs = refs
	return ctx, nil
}

func (c simpleFrameRefContext) directMotionContext(frame *DecodedFrame, sh *SliceHeader, sei *H264SEIContext) h264DirectMotionContext {
	var x264Build int32
	if sei != nil {
		x264Build = sei.Common.Unregistered.X264Build
	}
	var curPOC int32
	if frame != nil {
		curPOC = frame.poc
	}
	direct8x8 := false
	directSpatial := false
	if sh != nil && sh.SPS != nil {
		directSpatial = sh.DirectSpatialMVPred != 0
		direct8x8 = sh.SPS.Direct8x8InferenceFlag != 0
	}
	return h264DirectMotionContext{
		RefEntries:          c.Entries,
		CurPOC:              curPOC,
		DirectSpatialMVPred: directSpatial,
		Direct8x8Inference:  direct8x8,
		X264Build:           x264Build,
	}
}

func cloneSimpleRefEntries2(entries [2][]simpleRefEntry) [2][]simpleRefEntry {
	return [2][]simpleRefEntry{
		cloneSimpleRefEntries(entries[0]),
		cloneSimpleRefEntries(entries[1]),
	}
}

func cloneSimpleRefEntries(entries []simpleRefEntry) []simpleRefEntry {
	if len(entries) == 0 {
		return nil
	}
	out := make([]simpleRefEntry, len(entries))
	copy(out, entries)
	return out
}

func simpleFrameEntryPlanesRefs(list []simpleRefEntry) []*h264PicturePlanes {
	planes := make([]h264PicturePlanes, len(list))
	refs := make([]*h264PicturePlanes, len(list))
	for i, entry := range list {
		planes[i] = entry.frame.picturePlanes()
		refs[i] = &planes[i]
	}
	return refs
}

func simpleFrameEntryFrames(list []simpleRefEntry) []*DecodedFrame {
	frames := make([]*DecodedFrame, len(list))
	for i, entry := range list {
		frames[i] = entry.frame
	}
	return frames
}

func (d *simpleFrameDPB) buildBRefLists(sh *SliceHeader, curPOC int32) ([2][]*DecodedFrame, error) {
	entries, err := d.buildBRefEntries(sh, curPOC)
	if err != nil {
		return [2][]*DecodedFrame{}, err
	}
	return [2][]*DecodedFrame{
		simpleFrameEntryFrames(entries[0]),
		simpleFrameEntryFrames(entries[1]),
	}, nil
}

func (d *simpleFrameDPB) buildBRefEntries(sh *SliceHeader, curPOC int32) ([2][]simpleRefEntry, error) {
	var entries [2][]simpleRefEntry
	if sh.RefCount[0] == 0 || sh.RefCount[1] == 0 {
		return entries, ErrInvalidData
	}
	if sh.RefCount[0] > simpleMaxShortRefs || sh.RefCount[1] > simpleMaxShortRefs {
		return entries, ErrUnsupported
	}

	defaults := [2][]simpleRefEntry{}
	for list := 0; list < 2; list++ {
		var err error
		defaults[list], err = d.buildDefaultBRefList(sh, curPOC, list)
		if err != nil {
			return entries, err
		}
	}
	if simpleRefListsSameFrames(defaults[0], defaults[1]) && len(defaults[1]) > 1 {
		defaults[1][0], defaults[1][1] = defaults[1][1], defaults[1][0]
	}
	for list := 0; list < 2; list++ {
		listEntries, err := d.applyRefModificationsEntries(defaults[list], sh, list)
		if err != nil {
			return entries, err
		}
		entries[list] = listEntries
	}
	return entries, nil
}

func (d *simpleFrameDPB) buildDefaultBRefList(sh *SliceHeader, curPOC int32, list int) ([]simpleRefEntry, error) {
	if list != 0 && list != 1 {
		return nil, ErrInvalidData
	}
	sorted := d.addSortedShortRefs(curPOC, 1^list)
	sorted = append(sorted, d.addSortedShortRefs(curPOC, 0^list)...)
	entries := make([]simpleRefEntry, 0, len(sorted)+d.longCount())
	for _, frame := range sorted {
		if frame == nil {
			continue
		}
		if err := frame.matchesSPS(sh.SPS); err != nil {
			return nil, err
		}
		entries = append(entries, simpleRefEntry{
			frame: frame,
			picID: frame.frameNum,
		})
	}
	for i, frame := range d.long {
		if frame == nil {
			continue
		}
		if err := frame.matchesSPS(sh.SPS); err != nil {
			return nil, err
		}
		entries = append(entries, simpleRefEntry{
			frame: frame,
			picID: uint32(i),
			long:  true,
		})
	}
	return entries, nil
}

func (d *simpleFrameDPB) addSortedShortRefs(curPOC int32, dir int) []*DecodedFrame {
	out := make([]*DecodedFrame, 0, len(d.short))
	limit := curPOC
	for {
		bestPOC := int32(math.MaxInt32)
		if dir != 0 {
			bestPOC = math.MinInt32
		}
		var best *DecodedFrame
		for _, frame := range d.short {
			if frame == nil {
				continue
			}
			if dir != 0 {
				if frame.poc <= limit && frame.poc >= bestPOC {
					bestPOC = frame.poc
					best = frame
				}
			} else if frame.poc > limit && frame.poc < bestPOC {
				bestPOC = frame.poc
				best = frame
			}
		}
		if best == nil {
			break
		}
		out = append(out, best)
		if dir != 0 {
			limit = best.poc - 1
		} else {
			limit = best.poc
		}
	}
	return out
}

func simpleRefListsSameFrames(a []simpleRefEntry, b []simpleRefEntry) bool {
	if len(a) != len(b) || len(a) <= 1 {
		return false
	}
	for i := range a {
		if a[i].frame != b[i].frame {
			return false
		}
	}
	return true
}

func (d *simpleFrameDPB) applyRefModifications(list []simpleRefEntry, sh *SliceHeader, listIndex int) ([]*DecodedFrame, error) {
	entries, err := d.applyRefModificationsEntries(list, sh, listIndex)
	if err != nil {
		return nil, err
	}
	return simpleFrameEntryFrames(entries), nil
}

func (d *simpleFrameDPB) applyRefModificationsEntries(list []simpleRefEntry, sh *SliceHeader, listIndex int) ([]simpleRefEntry, error) {
	if listIndex != 0 && listIndex != 1 {
		return nil, ErrInvalidData
	}
	refCount := int(sh.RefCount[listIndex])
	if refCount == 0 {
		return nil, ErrInvalidData
	}
	if len(list) == 0 {
		return nil, ErrInvalidData
	}
	defaultRef := list[0]
	for len(list) < refCount {
		list = append(list, simpleRefEntry{})
	}

	pred := sh.CurrPicNum
	if sh.MaxPicNum == 0 {
		return nil, ErrInvalidData
	}
	for index := uint32(0); index < sh.NBRefModifications[listIndex]; index++ {
		if index >= sh.RefCount[listIndex] || index >= maxRefMods {
			return nil, ErrInvalidData
		}
		mod := sh.RefModifications[listIndex][index]
		if mod.Op > 2 {
			return nil, ErrUnsupported
		}
		entry := simpleRefEntry{}
		picID := uint32(0)
		isLong := false
		var ref *DecodedFrame
		if mod.Op == 2 {
			picID = mod.Val
			isLong = true
			if picID > 31 {
				return nil, ErrInvalidData
			}
			if picID < simpleMaxLongRefs {
				ref = d.findLongByIndex(picID)
			}
		} else {
			absDiffPicNum := mod.Val + 1
			if absDiffPicNum > sh.MaxPicNum {
				return nil, ErrInvalidData
			}
			if mod.Op == 0 {
				pred -= absDiffPicNum
			} else {
				pred += absDiffPicNum
			}
			pred &= sh.MaxPicNum - 1
			picID = pred
			ref = d.findShortByFrameNum(pred)
		}
		if ref != nil {
			entry = simpleRefEntry{frame: ref, picID: picID, long: isLong}
		}
		i := int(index)
		if ref != nil {
			for ; i+1 < refCount; i++ {
				if list[i].frame != nil && list[i].long == isLong && list[i].picID == picID {
					break
				}
			}
		}
		for ; i > int(index); i-- {
			list[i] = list[i-1]
		}
		list[index] = entry
	}

	out := make([]simpleRefEntry, refCount)
	for i := range out {
		if list[i].frame == nil {
			list[i] = defaultRef
		}
		out[i] = list[i]
	}
	return out, nil
}

func (d *simpleFrameDPB) buildPRefList(sh *SliceHeader) ([]*DecodedFrame, error) {
	entries, err := d.buildPRefEntries(sh)
	if err != nil {
		return nil, err
	}
	return simpleFrameEntryFrames(entries), nil
}

func (d *simpleFrameDPB) buildPRefEntries(sh *SliceHeader) ([]simpleRefEntry, error) {
	if sh.RefCount[0] == 0 {
		return nil, ErrInvalidData
	}
	if sh.RefCount[0] > simpleMaxShortRefs {
		return nil, ErrUnsupported
	}
	list, err := d.buildDefaultPRefList(sh)
	if err != nil {
		return nil, err
	}
	return d.applyRefModificationsEntries(list, sh, 0)
}

// initImplicitBWeightTable is a progressive frame-picture port of FFmpeg
// n8.0.1 libavcodec/h264_slice.c implicit_weight_table(field=-1).
func initImplicitBWeightTable(pwt *PredWeightTable, lists [2][]simpleRefEntry, refCount [2]uint32, curPOC int32) error {
	if pwt == nil {
		return ErrInvalidData
	}
	for i := 0; i < 2; i++ {
		pwt.LumaWeightFlag[i] = 0
		pwt.ChromaWeightFlag[i] = 0
	}

	refCount0 := int(refCount[0])
	refCount1 := int(refCount[1])
	if refCount0 <= 0 || refCount1 <= 0 || refCount0 > len(lists[0]) || refCount1 > len(lists[1]) ||
		refCount0 > len(pwt.ImplicitWeight) || refCount1 > len(pwt.ImplicitWeight[0]) {
		return ErrInvalidData
	}
	if refCount0 == 1 && refCount1 == 1 &&
		int64(lists[0][0].frame.poc)+int64(lists[1][0].frame.poc) == 2*int64(curPOC) {
		pwt.UseWeight = 0
		pwt.UseWeightChroma = 0
		return nil
	}

	pwt.UseWeight = 2
	pwt.UseWeightChroma = 2
	pwt.LumaLog2WeightDenom = 5
	pwt.ChromaLog2WeightDenom = 5

	for ref0 := 0; ref0 < refCount0; ref0++ {
		poc0 := int(lists[0][ref0].frame.poc)
		for ref1 := 0; ref1 < refCount1; ref1++ {
			w := int32(32)
			if !lists[0][ref0].long && !lists[1][ref1].long {
				poc1 := int(lists[1][ref1].frame.poc)
				td := clipInt(poc1-poc0, -128, 127)
				if td != 0 {
					tb := clipInt(int(curPOC)-poc0, -128, 127)
					tx := (16384 + (absInt(td) >> 1)) / td
					distScaleFactor := (tb*tx + 32) >> 8
					if distScaleFactor >= -64 && distScaleFactor <= 128 {
						w = int32(64 - distScaleFactor)
					}
				}
			}
			pwt.ImplicitWeight[ref0][ref1][0] = w
			pwt.ImplicitWeight[ref0][ref1][1] = w
		}
	}
	return nil
}

func (d *simpleFrameDPB) buildDefaultPRefList(sh *SliceHeader) ([]simpleRefEntry, error) {
	list := make([]simpleRefEntry, 0, len(d.short))
	for _, frame := range d.short {
		if frame == nil {
			continue
		}
		if err := frame.matchesSPS(sh.SPS); err != nil {
			return nil, err
		}
		list = append(list, simpleRefEntry{
			frame: frame,
			picID: frame.frameNum,
		})
	}
	for i, frame := range d.long {
		if frame == nil {
			continue
		}
		if err := frame.matchesSPS(sh.SPS); err != nil {
			return nil, err
		}
		list = append(list, simpleRefEntry{
			frame: frame,
			picID: uint32(i),
			long:  true,
		})
	}
	return list, nil
}

func (d *simpleFrameDPB) markDecodedFrame(frame *DecodedFrame, sh *SliceHeader, nalRefIDC uint8) error {
	if d == nil || frame == nil || sh == nil || sh.SPS == nil {
		return ErrInvalidData
	}
	if sh.PictureStructure != PictureFrame {
		return ErrUnsupported
	}
	defer d.finishFramePOC(nalRefIDC)
	frame.frameNum = sh.FrameNum
	currentRefAssigned := false
	if sh.NALType == NALIDRSlice {
		d.resetRefs()
		if sh.NBMMCO != 0 {
			resetFrameNum, assigned, err := d.applyMMCO(frame, sh)
			if err != nil {
				return err
			}
			currentRefAssigned = assigned
			if resetFrameNum {
				frame.frameNum = 0
				frame.mmcoReset = true
			}
		}
	} else if sh.NBMMCO != 0 {
		resetFrameNum, assigned, err := d.applyMMCO(frame, sh)
		if err != nil {
			return err
		}
		currentRefAssigned = assigned
		if resetFrameNum {
			frame.frameNum = 0
			frame.mmcoReset = true
		}
	}
	if nalRefIDC == 0 {
		return nil
	}

	maxRefs := int(sh.SPS.RefFrameCount)
	if maxRefs < 1 {
		maxRefs = 1
	}
	if maxRefs > simpleMaxShortRefs {
		return ErrUnsupported
	}
	if sh.ExplicitRefMarking == 0 && len(d.short) != 0 && d.refCount() >= maxRefs {
		d.removeShortAtIndex(len(d.short) - 1)
	}
	if !currentRefAssigned {
		d.removeShortByFrameNum(frame.frameNum)
		d.short = append(d.short, nil)
		copy(d.short[1:], d.short[:len(d.short)-1])
		d.short[0] = frame
	}
	if d.refCount() > maxRefs {
		if d.longCount() != 0 && len(d.short) == 0 {
			d.removeFirstLong()
		} else if len(d.short) != 0 {
			d.removeShortAtIndex(len(d.short) - 1)
		}
		return ErrInvalidData
	}
	return nil
}

func (d *simpleFrameDPB) holdOutputFrame(frame *DecodedFrame, sh *SliceHeader) error {
	if d == nil || frame == nil || sh == nil || sh.SPS == nil {
		return ErrInvalidData
	}
	d.updateReorderDelay(frame, sh)
	if d.hasBFrames > h264MaxDPBFrames {
		return ErrUnsupported
	}
	if len(d.delayed) > h264MaxDPBFrames {
		return ErrInvalidData
	}
	d.delayed = append(d.delayed, frame)
	return nil
}

// updateReorderDelay mirrors FFmpeg n8.0.1 libavcodec/h264_slice.c
// h264_select_output_frame's dynamic has_b_frames / last_pocs logic.
func (d *simpleFrameDPB) updateReorderDelay(frame *DecodedFrame, sh *SliceHeader) {
	if sh.SPS.BitstreamRestrictionFlag != 0 && sh.SPS.NumReorderFrames > int32(d.hasBFrames) {
		d.hasBFrames = int(sh.SPS.NumReorderFrames)
	}

	outOfOrder := d.notePOCReorder(frame.poc, sh.SliceTypeNoS == PictureTypeB)
	if outOfOrder == h264MaxDPBFrames {
		d.resetLastPOCs()
		d.lastPOCs[0] = frame.poc
		frame.mmcoReset = true
		return
	}
	if d.hasBFrames < outOfOrder && sh.SPS.BitstreamRestrictionFlag == 0 {
		d.hasBFrames = outOfOrder
	}
}

func (d *simpleFrameDPB) notePOCReorder(curPOC int32, isB bool) int {
	d.ensureLastPOCs()
	i := 0
	for ; ; i++ {
		if i == h264MaxDPBFrames || curPOC < d.lastPOCs[i] {
			if i != 0 {
				d.lastPOCs[i-1] = curPOC
			}
			break
		} else if i != 0 {
			d.lastPOCs[i-1] = d.lastPOCs[i]
		}
	}
	outOfOrder := h264MaxDPBFrames - i
	if isB || (d.lastPOCs[h264MaxDPBFrames-2] > math.MinInt32 &&
		int64(d.lastPOCs[h264MaxDPBFrames-1])-int64(d.lastPOCs[h264MaxDPBFrames-2]) > 2) {
		if outOfOrder < 1 {
			outOfOrder = 1
		}
	}
	return outOfOrder
}

func (d *simpleFrameDPB) ensureLastPOCs() {
	if !d.lastPOCsInit {
		d.resetLastPOCs()
	}
}

func (d *simpleFrameDPB) resetLastPOCs() {
	for i := range d.lastPOCs {
		d.lastPOCs[i] = math.MinInt32
	}
	d.lastPOCsInit = true
}

func (d *simpleFrameDPB) drainOutputFrames(flush bool) ([]*DecodedFrame, error) {
	if d == nil {
		return nil, ErrInvalidData
	}
	var out []*DecodedFrame
	for len(d.delayed) != 0 {
		outIdx := d.nextOutputFrameIndex()
		if outIdx < 0 || outIdx >= len(d.delayed) {
			return nil, ErrInvalidData
		}
		frame := d.delayed[outIdx]
		if d.hasBFrames == 0 && len(d.delayed) != 0 && (d.delayed[0].idrKeyFrame || d.delayed[0].mmcoReset) {
			d.nextOutputedValid = false
		}
		outOfOrder := d.nextOutputedValid && frame.poc < d.nextOutputedPOC
		if !flush && !outOfOrder && len(d.delayed) <= d.hasBFrames {
			break
		}
		d.removeDelayedOutputAt(outIdx)
		if !outOfOrder {
			d.nextOutputedPOC = frame.poc
			d.nextOutputedValid = true
		}
		out = append(out, frame)
		if !flush {
			break
		}
	}
	return out, nil
}

func (d *simpleFrameDPB) nextOutputFrameIndex() int {
	outIdx := 0
	for i := 1; i < len(d.delayed) && !d.delayed[i].idrKeyFrame && !d.delayed[i].mmcoReset; i++ {
		if d.delayed[i].poc < d.delayed[outIdx].poc {
			outIdx = i
		}
	}
	return outIdx
}

func (d *simpleFrameDPB) removeDelayedOutputAt(index int) {
	copy(d.delayed[index:], d.delayed[index+1:])
	d.delayed[len(d.delayed)-1] = nil
	d.delayed = d.delayed[:len(d.delayed)-1]
}

func (d *simpleFrameDPB) applyMMCO(frame *DecodedFrame, sh *SliceHeader) (bool, bool, error) {
	resetFrameNum := false
	currentRefAssigned := false
	for i := uint32(0); i < sh.NBMMCO; i++ {
		switch sh.MMCO[i].Opcode {
		case mmcoEnd:
			return resetFrameNum, currentRefAssigned, nil
		case mmcoShort2Unused:
			d.removeShortByFrameNum(sh.MMCO[i].ShortPicNum)
		case mmcoShort2Long:
			if sh.MMCO[i].LongArg >= simpleMaxLongRefs {
				return resetFrameNum, currentRefAssigned, ErrInvalidData
			}
			pic := d.findShortByFrameNum(sh.MMCO[i].ShortPicNum)
			if pic == nil {
				long := d.findLongByIndex(sh.MMCO[i].LongArg)
				if long == nil || long.frameNum != sh.MMCO[i].ShortPicNum {
					return resetFrameNum, currentRefAssigned, ErrInvalidData
				}
				continue
			}
			longIndex := int(sh.MMCO[i].LongArg)
			if d.long[longIndex] != pic {
				d.removeLongByIndex(longIndex)
			}
			d.removeShortByFrameNum(sh.MMCO[i].ShortPicNum)
			d.long[longIndex] = pic
		case mmcoLong2Unused:
			if sh.MMCO[i].LongArg >= simpleMaxLongRefs {
				return resetFrameNum, currentRefAssigned, ErrInvalidData
			}
			d.removeLongByIndex(int(sh.MMCO[i].LongArg))
		case mmcoLong:
			if sh.MMCO[i].LongArg >= simpleMaxLongRefs {
				return resetFrameNum, currentRefAssigned, ErrInvalidData
			}
			longIndex := int(sh.MMCO[i].LongArg)
			d.removeLongRefsForFrame(frame)
			if d.long[longIndex] != frame {
				d.removeLongByIndex(longIndex)
				d.long[longIndex] = frame
			}
			currentRefAssigned = true
		case mmcoSetMaxLong:
			if sh.MMCO[i].LongArg > simpleMaxLongRefs {
				return resetFrameNum, currentRefAssigned, ErrInvalidData
			}
			for j := int(sh.MMCO[i].LongArg); j < simpleMaxLongRefs; j++ {
				d.removeLongByIndex(j)
			}
		case mmcoReset:
			d.resetRefs()
			d.poc.frameNum = 0
			resetFrameNum = true
		default:
			return resetFrameNum, currentRefAssigned, ErrUnsupported
		}
	}
	return resetFrameNum, currentRefAssigned, nil
}

func (d *simpleFrameDPB) findShortByFrameNum(frameNum uint32) *DecodedFrame {
	if d == nil {
		return nil
	}
	for _, frame := range d.short {
		if frame != nil && frame.frameNum == frameNum {
			return frame
		}
	}
	return nil
}

func (d *simpleFrameDPB) findLongByIndex(index uint32) *DecodedFrame {
	if d == nil || index >= simpleMaxLongRefs {
		return nil
	}
	return d.long[index]
}

func (d *simpleFrameDPB) removeShortByFrameNum(frameNum uint32) {
	if d == nil {
		return
	}
	for i, frame := range d.short {
		if frame == nil || frame.frameNum != frameNum {
			continue
		}
		d.removeShortAtIndex(i)
		return
	}
}

func (d *simpleFrameDPB) removeShortAtIndex(index int) {
	if d == nil || index < 0 || index >= len(d.short) {
		return
	}
	copy(d.short[index:], d.short[index+1:])
	d.short[len(d.short)-1] = nil
	d.short = d.short[:len(d.short)-1]
}

func (d *simpleFrameDPB) removeLongByIndex(index int) {
	if d == nil || index < 0 || index >= simpleMaxLongRefs {
		return
	}
	d.long[index] = nil
}

func (d *simpleFrameDPB) removeLongRefsForFrame(frame *DecodedFrame) {
	if d == nil || frame == nil {
		return
	}
	for i, ref := range d.long {
		if ref == frame {
			d.long[i] = nil
		}
	}
}

func (d *simpleFrameDPB) removeFirstLong() {
	if d == nil {
		return
	}
	for i, ref := range d.long {
		if ref != nil {
			d.long[i] = nil
			return
		}
	}
}

func (d *simpleFrameDPB) longCount() int {
	if d == nil {
		return 0
	}
	n := 0
	for _, ref := range d.long {
		if ref != nil {
			n++
		}
	}
	return n
}

func (d *simpleFrameDPB) refCount() int {
	if d == nil {
		return 0
	}
	return len(d.short) + d.longCount()
}
