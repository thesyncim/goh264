// SPDX-License-Identifier: LGPL-2.1-or-later
//
// Source-shaped simple short-reference DPB/ref-list subset from FFmpeg
// n8.0.1 libavcodec/h264_refs.c h264_initialise_ref_list,
// ff_h264_build_ref_list, and ff_h264_execute_ref_pic_marking. This file is
// intentionally limited to progressive frame-picture short refs for the
// already-translated simple P-slice path.

package h264

const simpleMaxShortRefs = 16

type simpleFrameDPB struct {
	short []*DecodedFrame
}

type simpleRefEntry struct {
	frame *DecodedFrame
	picID uint32
}

func (d *simpleFrameDPB) reset() {
	if d != nil {
		d.short = d.short[:0]
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
		if mod.Op > 1 {
			return nil, ErrUnsupported
		}
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

		ref := d.findShortByFrameNum(pred)
		entry := simpleRefEntry{}
		if ref != nil {
			entry = simpleRefEntry{frame: ref, picID: pred}
		}
		i := int(index)
		if ref != nil {
			for ; i+1 < refCount; i++ {
				if list[i].frame == ref || list[i].picID == pred {
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
	if sh.NALType == NALIDRSlice {
		d.reset()
		if sh.NBMMCO != 0 {
			return ErrUnsupported
		}
	} else if sh.NBMMCO != 0 {
		resetFrameNum, err := d.applyMMCO(sh)
		if err != nil {
			return err
		}
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
	if sh.ExplicitRefMarking == 0 {
		for len(d.short) >= maxRefs {
			d.short = d.short[:len(d.short)-1]
		}
	}

	d.removeShortByFrameNum(frame.frameNum)
	d.short = append(d.short, nil)
	copy(d.short[1:], d.short[:len(d.short)-1])
	d.short[0] = frame
	return nil
}

func (d *simpleFrameDPB) applyMMCO(sh *SliceHeader) (bool, error) {
	resetFrameNum := false
	for i := uint32(0); i < sh.NBMMCO; i++ {
		switch sh.MMCO[i].Opcode {
		case mmcoEnd:
			return resetFrameNum, nil
		case mmcoShort2Unused:
			d.removeShortByFrameNum(sh.MMCO[i].ShortPicNum)
		case mmcoReset:
			d.reset()
			resetFrameNum = true
		default:
			return resetFrameNum, ErrUnsupported
		}
	}
	return resetFrameNum, nil
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

func (d *simpleFrameDPB) removeShortByFrameNum(frameNum uint32) {
	if d == nil {
		return
	}
	for i, frame := range d.short {
		if frame == nil || frame.frameNum != frameNum {
			continue
		}
		copy(d.short[i:], d.short[i+1:])
		d.short[len(d.short)-1] = nil
		d.short = d.short[:len(d.short)-1]
		return
	}
}
