// SPDX-License-Identifier: LGPL-2.1-or-later

package goh264_test

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/thesyncim/goh264/internal/h264"
)

const high10ResidualCAVLCP16x16AnnexBHex = `
00000001676e000aa6cb4f6022000003000200000300041e244d400000000168ce0f2c800000016588842ac431520102fcc02146cc001eb50618003c2f4e2f2194bc35277ff81a11d900d0073811a0ead5e6a41918a25ae77eb3889867421b15
20eb6705a82ea70c7e47940a3d5ec0d0e1752eae037285fdf5a87b1c114d58d91eaf5d270c742bccf3d5ea00f98c66c70055affbe581e291d294048cdfef830020002015cc30003608022001be3734a420fc71ca91e09589ea0ad27d8691b710
0eca4feeb402878c1a0ccbaf7700198170a34cb3841b0971f0594fac7d688cc49ac70ceb810ffec01cad1e8b46c4d1ffe800000001419a22bc6c38783e807850474e56a7541e7ef7a85f0152627aafa6708000401c01c791082c8c0012221181
4ae246ac54db61e533933990afdbc001c3814d8eb44598cbd0a3e758bafb7f1437d4
`

const (
	high10ResidualCAVLCFrameRawSize = 768
	high10ResidualCAVLCRawVideoMD5  = "42e8d152117304a86b492cd0d529e90e"
)

var high10ResidualCAVLCFrameMD5 = []string{
	"95893f95fdce0f45e7593f4eca8bd834",
	"22ace8bfddbddf2958ef31f3d56ab09d",
}

func TestHigh10ResidualCAVLCP16x16FixtureSyntax(t *testing.T) {
	assertHigh10ResidualCAVLCP16x16FixtureSyntax(t, decodeHexFixture(t, high10ResidualCAVLCP16x16AnnexBHex))
}

func TestDecodeAnnexBHigh10ResidualCAVLCP16x16Frames(t *testing.T) {
	frames, err := NewDecoder().DecodeAnnexBFrames(decodeHexFixture(t, high10ResidualCAVLCP16x16AnnexBHex))
	if err != nil {
		t.Fatal(err)
	}
	assertHigh10ResidualCAVLCFrameMD5Strings(t, frames)
}

func TestFFmpegRawVideoFrameMD5OracleHigh10ResidualCAVLCP16x16(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}

	data := decodeHexFixture(t, high10ResidualCAVLCP16x16AnnexBHex)
	assertHigh10ResidualCAVLCP16x16FixtureSyntax(t, data)
	path := writeTempH264(t, data)

	cmd := exec.Command("ffmpeg",
		"-v", "error",
		"-f", "h264",
		"-i", path,
		"-an", "-sn", "-dn",
		"-pix_fmt", "yuv420p10le",
		"-f", "framemd5",
		"-",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("ffmpeg framemd5: %v\n%s", err, out)
	}
	for i, want := range high10ResidualCAVLCFrameMD5 {
		line := []byte(fmt.Sprintf("0, %10d, %10d,        1, %8d, %s", i, i, high10ResidualCAVLCFrameRawSize, want))
		if !bytes.Contains(out, line) {
			t.Fatalf("frame[%d] missing %q in framemd5:\n%s", i, line, out)
		}
	}

	rawvideo := exec.Command("ffmpeg",
		"-v", "error",
		"-f", "h264",
		"-i", path,
		"-an", "-sn", "-dn",
		"-pix_fmt", "yuv420p10le",
		"-f", "rawvideo",
		"-",
	)
	raw, err := rawvideo.CombinedOutput()
	if err != nil {
		t.Fatalf("ffmpeg rawvideo: %v\n%s", err, raw)
	}
	if len(raw) != len(high10ResidualCAVLCFrameMD5)*high10ResidualCAVLCFrameRawSize {
		t.Fatalf("rawvideo size = %d, want %d", len(raw), len(high10ResidualCAVLCFrameMD5)*high10ResidualCAVLCFrameRawSize)
	}
	sum := md5.Sum(raw)
	if got := hex.EncodeToString(sum[:]); got != high10ResidualCAVLCRawVideoMD5 {
		t.Fatalf("rawvideo md5 = %s, want %s", got, high10ResidualCAVLCRawVideoMD5)
	}
}

func assertHigh10ResidualCAVLCP16x16FixtureSyntax(t *testing.T, data []byte) {
	t.Helper()
	nals, err := h264.SplitAnnexB(data)
	if err != nil {
		t.Fatal(err)
	}

	var spsList [32]*h264.SPS
	var ppsList [256]*h264.PPS
	var gotVCL []h264.NALUnit
	var gotSlices []int32
	for _, nal := range nals {
		switch nal.Type {
		case h264.NALSPS:
			sps, err := h264.DecodeSPS(nal.RBSP)
			if err != nil {
				t.Fatal(err)
			}
			if sps.ProfileIDC != 110 || sps.Width != 16 || sps.Height != 16 ||
				sps.ChromaFormatIDC != 1 || sps.BitDepthLuma != 10 || sps.BitDepthChroma != 10 ||
				sps.FrameMBSOnlyFlag != 1 || sps.RefFrameCount != 1 {
				t.Fatalf("SPS profile/format = %d %dx%d chroma %d depth %d/%d frame_mbs_only=%d refs=%d, want High10 16x16 yuv420p10le frame-only refs=1",
					sps.ProfileIDC, sps.Width, sps.Height, sps.ChromaFormatIDC, sps.BitDepthLuma, sps.BitDepthChroma, sps.FrameMBSOnlyFlag, sps.RefFrameCount)
			}
			spsList[sps.SPSID] = sps
		case h264.NALPPS:
			pps, err := h264.DecodePPS(nal.RBSP, &spsList)
			if err != nil {
				t.Fatal(err)
			}
			if pps.CABAC != 0 || pps.Transform8x8Mode != 0 || pps.WeightedPred != 0 ||
				pps.WeightedBipredIDC != 0 || pps.RefCount[0] != 1 || pps.RefCount[1] != 1 {
				t.Fatalf("PPS cavlc/8x8/weights/refs = %d/%d/%d/%d/%d/%d, want CAVLC no-8x8 unweighted ref=1",
					pps.CABAC, pps.Transform8x8Mode, pps.WeightedPred, pps.WeightedBipredIDC, pps.RefCount[0], pps.RefCount[1])
			}
			ppsList[pps.PPSID] = pps
		case h264.NALIDRSlice, h264.NALSlice:
			sh, err := h264.ParseSliceHeader(nal, &ppsList)
			if err != nil {
				t.Fatal(err)
			}
			if sh.PictureStructure != h264.PictureFrame || sh.DeblockingFilter != 0 {
				t.Fatalf("slice picture/deblock = %d/%d, want frame/disabled", sh.PictureStructure, sh.DeblockingFilter)
			}
			gotVCL = append(gotVCL, nal)
			gotSlices = append(gotSlices, sh.SliceTypeNoS)
		default:
			t.Fatalf("unexpected NAL type %d in stripped fixture", nal.Type)
		}
	}
	if len(gotVCL) != 2 {
		t.Fatalf("VCL NAL count = %d, want 2", len(gotVCL))
	}
	if gotVCL[0].Type != h264.NALIDRSlice || gotVCL[1].Type != h264.NALSlice {
		t.Fatalf("VCL NALs = %v, want IDR then non-IDR", []h264.NALUnitType{gotVCL[0].Type, gotVCL[1].Type})
	}
	if len(gotSlices) != 2 {
		t.Fatalf("slice count = %d, want 2", len(gotSlices))
	}
	if gotSlices[0] != h264.PictureTypeI || gotSlices[1] != h264.PictureTypeP {
		t.Fatalf("slice types = %v, want I then P", gotSlices)
	}

	pmb := readHigh10ResidualCAVLCFirstPMacroblock(t, gotVCL[1], spsList[0], ppsList[0])
	if pmb.skipRun != 0 || pmb.mbType != 0 || pmb.cbp == 0 {
		t.Fatalf("P macroblock skip/mb_type/cbp = %d/%d/%d (code %d), want non-skip P_L0_16x16 with residual",
			pmb.skipRun, pmb.mbType, pmb.cbp, pmb.cbpCode)
	}
}

func assertHigh10ResidualCAVLCFrameMD5Strings(t *testing.T, frames []*Frame) {
	t.Helper()
	if len(frames) != len(high10ResidualCAVLCFrameMD5) {
		t.Fatalf("frames = %d, want %d", len(frames), len(high10ResidualCAVLCFrameMD5))
	}
	var allRaw []byte
	for i, frame := range frames {
		if frame == nil {
			t.Fatalf("frame[%d] is nil", i)
		}
		if frame.Width != 16 || frame.Height != 16 || frame.ChromaFormatIDC != 1 ||
			frame.BitDepthLuma != 10 || frame.BitDepthChroma != 10 {
			t.Fatalf("frame[%d] metadata = %dx%d chroma %d depth %d/%d, want 16x16 yuv420p10le",
				i, frame.Width, frame.Height, frame.ChromaFormatIDC, frame.BitDepthLuma, frame.BitDepthChroma)
		}
		rawSize, err := frame.RawYUVSize()
		if err != nil {
			t.Fatalf("frame[%d] RawYUVSize: %v", i, err)
		}
		if rawSize != high10ResidualCAVLCFrameRawSize {
			t.Fatalf("frame[%d] RawYUVSize = %d, want %d", i, rawSize, high10ResidualCAVLCFrameRawSize)
		}
		raw, err := frame.AppendRawYUVBytesLE(nil)
		if err != nil {
			t.Fatalf("frame[%d] AppendRawYUVBytesLE: %v", i, err)
		}
		if len(raw) != high10ResidualCAVLCFrameRawSize {
			t.Fatalf("frame[%d] raw len = %d, want %d", i, len(raw), high10ResidualCAVLCFrameRawSize)
		}
		allRaw = append(allRaw, raw...)
		sum := md5.Sum(raw)
		if got := hex.EncodeToString(sum[:]); got != high10ResidualCAVLCFrameMD5[i] {
			t.Fatalf("frame[%d] md5 = %s, want %s", i, got, high10ResidualCAVLCFrameMD5[i])
		}
	}
	if len(allRaw) != len(high10ResidualCAVLCFrameMD5)*high10ResidualCAVLCFrameRawSize {
		t.Fatalf("rawvideo len = %d, want %d", len(allRaw), len(high10ResidualCAVLCFrameMD5)*high10ResidualCAVLCFrameRawSize)
	}
	sum := md5.Sum(allRaw)
	if got := hex.EncodeToString(sum[:]); got != high10ResidualCAVLCRawVideoMD5 {
		t.Fatalf("rawvideo md5 = %s, want %s", got, high10ResidualCAVLCRawVideoMD5)
	}
}

type high10ResidualCAVLCFirstPMacroblock struct {
	skipRun uint32
	mbType  uint32
	cbpCode uint32
	cbp     uint32
}

func readHigh10ResidualCAVLCFirstPMacroblock(t *testing.T, nal h264.NALUnit, sps *h264.SPS, pps *h264.PPS) high10ResidualCAVLCFirstPMacroblock {
	t.Helper()
	if sps == nil || pps == nil {
		t.Fatal("missing SPS/PPS for P macroblock syntax check")
	}
	br := high10ResidualCAVLCBitReader{data: nal.RBSP}
	firstMB := br.readUE(t)
	rawSliceType := br.readUE(t)
	sliceTypeNoS := high10ResidualCAVLCSliceTypeNoS(t, rawSliceType)
	ppsID := br.readUE(t)
	if firstMB != 0 || sliceTypeNoS != h264.PictureTypeP || ppsID != pps.PPSID {
		t.Fatalf("slice header first_mb/type/pps = %d/%d/%d, want first P slice with PPS %d", firstMB, sliceTypeNoS, ppsID, pps.PPSID)
	}
	br.readBits(t, int(sps.Log2MaxFrameNum))
	fieldPic := false
	if sps.FrameMBSOnlyFlag == 0 {
		fieldPic = br.readBit(t) != 0
		if fieldPic {
			br.readBit(t)
		}
	}
	if nal.Type == h264.NALIDRSlice {
		br.readUE(t)
	}
	if sps.PocType == 0 {
		br.readBits(t, int(sps.Log2MaxPocLSB))
		if pps.PicOrderPresent != 0 && !fieldPic {
			br.readSE(t)
		}
	} else if sps.PocType == 1 && sps.DeltaPicOrderAlwaysZeroFlag == 0 {
		br.readSE(t)
		if pps.PicOrderPresent != 0 && !fieldPic {
			br.readSE(t)
		}
	}
	if pps.RedundantPicCntPresent != 0 {
		br.readUE(t)
	}

	refCount0 := pps.RefCount[0]
	if br.readBit(t) != 0 {
		refCount0 = br.readUE(t) + 1
	}
	high10ResidualCAVLCReadRefPicListModifications(t, &br, 1)
	if pps.WeightedPred != 0 {
		t.Fatal("fixture unexpectedly uses weighted P prediction")
	}
	high10ResidualCAVLCReadRefPicMarking(t, &br, nal)
	if pps.CABAC != 0 {
		br.readUE(t)
	}
	br.readSE(t)
	if pps.DeblockingFilterParametersPresent != 0 {
		disableIDC := br.readUE(t)
		if disableIDC != 1 {
			t.Fatalf("disable_deblocking_filter_idc = %d, want 1", disableIDC)
		}
	}

	skipRun := br.readUE(t)
	mbType := br.readUE(t)
	if refCount0 > 1 {
		t.Fatalf("refCount0 = %d, want 1 so ref_idx_l0 is absent", refCount0)
	}
	mvdPairs := 1
	switch mbType {
	case 0:
		mvdPairs = 1
	case 1, 2:
		mvdPairs = 2
	case 3, 4:
		mvdPairs = 0
		for i := 0; i < 4; i++ {
			subMBType := br.readUE(t)
			switch subMBType {
			case 0:
				mvdPairs += 1
			case 1, 2:
				mvdPairs += 2
			case 3:
				mvdPairs += 4
			default:
				t.Fatalf("P sub macroblock type[%d] = %d, want P8x8/P8x4/P4x8/P4x4 syntax", i, subMBType)
			}
		}
	default:
		t.Fatalf("P macroblock type = %d, want P16x16/P16x8/P8x16/P8x8 syntax", mbType)
	}
	for i := 0; i < mvdPairs; i++ {
		br.readSE(t)
		br.readSE(t)
	}
	cbpCode := br.readUE(t)
	if cbpCode >= uint32(len(high10ResidualCAVLCInterCBP)) {
		t.Fatalf("coded_block_pattern code = %d, want < %d", cbpCode, len(high10ResidualCAVLCInterCBP))
	}
	return high10ResidualCAVLCFirstPMacroblock{
		skipRun: skipRun,
		mbType:  mbType,
		cbpCode: cbpCode,
		cbp:     uint32(high10ResidualCAVLCInterCBP[cbpCode]),
	}
}

func high10ResidualCAVLCSliceTypeNoS(t *testing.T, raw uint32) int32 {
	t.Helper()
	if raw > 9 {
		t.Fatalf("slice_type = %d, want <= 9", raw)
	}
	if raw > 4 {
		raw -= 5
	}
	switch raw {
	case 0:
		return h264.PictureTypeP
	case 1:
		return h264.PictureTypeB
	case 2:
		return h264.PictureTypeI
	case 3:
		return h264.PictureTypeSP
	case 4:
		return h264.PictureTypeSI
	default:
		t.Fatalf("slice_type = %d, want known type", raw)
		return 0
	}
}

func high10ResidualCAVLCReadRefPicListModifications(t *testing.T, br *high10ResidualCAVLCBitReader, listCount int) {
	t.Helper()
	for list := 0; list < listCount; list++ {
		if br.readBit(t) == 0 {
			continue
		}
		for {
			op := br.readUE(t)
			if op == 3 {
				break
			}
			if op > 2 {
				t.Fatalf("ref_pic_list_modification op = %d, want <= 2 or terminator", op)
			}
			br.readUE(t)
		}
	}
}

func high10ResidualCAVLCReadRefPicMarking(t *testing.T, br *high10ResidualCAVLCBitReader, nal h264.NALUnit) {
	t.Helper()
	if nal.RefIDC == 0 {
		return
	}
	if nal.Type == h264.NALIDRSlice {
		br.readBit(t)
		br.readBit(t)
		return
	}
	if br.readBit(t) == 0 {
		return
	}
	for {
		op := br.readUE(t)
		if op == 0 {
			break
		}
		switch op {
		case 1, 3:
			br.readUE(t)
		}
		switch op {
		case 2, 3, 4, 6:
			br.readUE(t)
		case 5:
		default:
			t.Fatalf("MMCO opcode = %d, want 0..6", op)
		}
	}
}

type high10ResidualCAVLCBitReader struct {
	data []byte
	bit  int
}

func (br *high10ResidualCAVLCBitReader) readBit(t *testing.T) uint32 {
	t.Helper()
	return br.readBits(t, 1)
}

func (br *high10ResidualCAVLCBitReader) readBits(t *testing.T, n int) uint32 {
	t.Helper()
	if n < 0 || n > 32 {
		t.Fatalf("invalid bit count %d", n)
	}
	var v uint32
	for i := 0; i < n; i++ {
		bytePos := br.bit >> 3
		if bytePos >= len(br.data) {
			t.Fatalf("bitreader overread at bit %d", br.bit)
		}
		v = (v << 1) | uint32((br.data[bytePos]>>(7-uint(br.bit&7)))&1)
		br.bit++
	}
	return v
}

func (br *high10ResidualCAVLCBitReader) readUE(t *testing.T) uint32 {
	t.Helper()
	zeros := 0
	for br.readBit(t) == 0 {
		zeros++
		if zeros > 31 {
			t.Fatal("ue(v) prefix too long")
		}
	}
	if zeros == 0 {
		return 0
	}
	return (uint32(1) << uint(zeros)) - 1 + br.readBits(t, zeros)
}

func (br *high10ResidualCAVLCBitReader) readSE(t *testing.T) int32 {
	t.Helper()
	ue := br.readUE(t)
	if ue&1 == 0 {
		return -int32(ue >> 1)
	}
	return int32((ue + 1) >> 1)
}

var high10ResidualCAVLCInterCBP = [48]uint8{
	0, 16, 1, 2, 4, 8, 32, 3, 5, 10, 12, 15, 47, 7, 11, 13,
	14, 6, 9, 31, 35, 37, 42, 44, 33, 34, 36, 40, 39, 43, 45, 46,
	17, 18, 20, 24, 19, 21, 26, 28, 23, 27, 29, 30, 22, 25, 38, 41,
}
