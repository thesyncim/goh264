// SPDX-License-Identifier: LGPL-2.1-or-later
//
// Source-shaped simple reference DPB/ref-list subset from FFmpeg
// n8.0.1 libavcodec/h264_refs.c h264_initialise_ref_list,
// ff_h264_build_ref_list, and ff_h264_execute_ref_pic_marking. This file is
// intentionally limited to progressive frame-picture refs for the
// already-translated simple P-slice path.

package h264

const simpleMaxShortRefs = 16
const simpleMaxLongRefs = 16

type simpleFrameDPB struct {
	short []*DecodedFrame
	long  [simpleMaxLongRefs]*DecodedFrame
}

type simpleRefEntry struct {
	frame *DecodedFrame
	picID uint32
	long  bool
}

func (d *simpleFrameDPB) reset() {
	if d != nil {
		d.short = d.short[:0]
		for i := range d.long {
			d.long[i] = nil
		}
	}
}

func (d *simpleFrameDPB) buildRefLists(sh *SliceHeader) ([2][]*h264PicturePlanes, error) {
	var refs [2][]*h264PicturePlanes
	if d == nil || sh == nil || sh.SPS == nil {
		return refs, ErrInvalidData
	}
	if sh.SliceTypeNoS == PictureTypeI {
		return refs, nil
	}
	if sh.SliceTypeNoS != PictureTypeP || sh.PictureStructure != PictureFrame {
		return refs, ErrUnsupported
	}

	list, err := d.buildPRefList(sh)
	if err != nil {
		return refs, err
	}
	planes := make([]h264PicturePlanes, len(list))
	refs[0] = make([]*h264PicturePlanes, len(list))
	for i, frame := range list {
		planes[i] = frame.picturePlanes()
		refs[0][i] = &planes[i]
	}
	return refs, nil
}

func (d *simpleFrameDPB) buildPRefList(sh *SliceHeader) ([]*DecodedFrame, error) {
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
	refCount := int(sh.RefCount[0])
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
	for index := uint32(0); index < sh.NBRefModifications[0]; index++ {
		if index >= sh.RefCount[0] || index >= maxRefMods {
			return nil, ErrInvalidData
		}
		mod := sh.RefModifications[0][index]
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

	out := make([]*DecodedFrame, refCount)
	for i := range out {
		if list[i].frame == nil {
			list[i] = defaultRef
		}
		out[i] = list[i].frame
	}
	return out, nil
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
	frame.frameNum = sh.FrameNum
	currentRefAssigned := false
	if sh.NALType == NALIDRSlice {
		d.reset()
		if sh.NBMMCO != 0 {
			resetFrameNum, assigned, err := d.applyMMCO(frame, sh)
			if err != nil {
				return err
			}
			currentRefAssigned = assigned
			if resetFrameNum {
				frame.frameNum = 0
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
			d.reset()
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
