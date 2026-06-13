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

type highFrameMBAFFBSkipCase struct {
	name                       string
	bitDepth                   int
	directSpatial              uint32
	disableDeblockingFilterIDC uint32
	deblockMode                int32
	bitstreamMD5               string
	frameMD5                   []string
	rawVideoMD5                string
}

func TestHigh1214FrameMBAFFBSkipFixtureSyntax(t *testing.T) {
	for _, tt := range highFrameMBAFFBSkipCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFBSkipFixture(tt)
			assertHighFrameMBAFFBSkipFixtureSyntax(t, data, tt)
		})
	}
}

func TestDecodeAnnexBHigh1214FrameMBAFFBSkipFrames(t *testing.T) {
	for _, tt := range highFrameMBAFFBSkipCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFBSkipFixture(tt)
			assertHighFrameMBAFFBSkipFixtureSyntax(t, data, tt)

			frames, err := NewDecoder().DecodeAnnexBFrames(data)
			if err != nil {
				t.Fatalf("DecodeAnnexBFrames: %v", err)
			}
			assertHighFrameMBAFFBSkipFrames(t, frames, tt)
		})
	}
}

func TestDecodeAVCHigh1214FrameMBAFFBSkipFrames(t *testing.T) {
	for _, tt := range highFrameMBAFFBSkipCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFBSkipFixture(tt)
			assertHighFrameMBAFFBSkipFixtureSyntax(t, data, tt)

			for _, nalLengthSize := range []int{2, 3, 4} {
				frames, err := NewDecoder().DecodeAVCFrames(annexBToAVC(t, data, nalLengthSize), nalLengthSize)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: DecodeAVCFrames: %v", nalLengthSize, err)
				}
				assertHighFrameMBAFFBSkipFrames(t, frames, tt)
			}
		})
	}
}

func TestDecodeAVCWithConfigurationRecordHigh1214FrameMBAFFBSkipFrames(t *testing.T) {
	for _, tt := range highFrameMBAFFBSkipCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFBSkipFixture(tt)
			assertHighFrameMBAFFBSkipFixtureSyntax(t, data, tt)

			for _, nalLengthSize := range []int{2, 3, 4} {
				config, packet := annexBToAVCConfigAndPacket(t, data, nalLengthSize)
				frames, err := NewDecoder().DecodeAVCFramesWithConfigurationRecord(config, packet)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: DecodeAVCFramesWithConfigurationRecord: %v", nalLengthSize, err)
				}
				assertHighFrameMBAFFBSkipFrames(t, frames, tt)
			}
		})
	}
}

func TestDecodeConfiguredAVCHigh1214FrameMBAFFBSkipFramesAcrossSamplesFlush(t *testing.T) {
	for _, tt := range highFrameMBAFFBSkipCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFBSkipFixture(tt)
			assertHighFrameMBAFFBSkipFixtureSyntax(t, data, tt)

			for _, nalLengthSize := range []int{2, 3, 4} {
				config, samples := annexBToAVCConfigAndSamples(t, data, nalLengthSize)
				if len(samples) != 3 {
					t.Fatalf("nalLengthSize=%d samples = %d, want IDR/P/B", nalLengthSize, len(samples))
				}
				dec := NewDecoder()
				if _, err := dec.ConfigureAVCDecoderConfigurationRecord(config); err != nil {
					t.Fatalf("nalLengthSize=%d config: %v", nalLengthSize, err)
				}
				var frames []*Frame
				for i, sample := range samples {
					out, err := dec.DecodeConfiguredAVCFrames(sample)
					if err != nil {
						t.Fatalf("nalLengthSize=%d sample[%d]: DecodeConfiguredAVCFrames: %v", nalLengthSize, i, err)
					}
					frames = append(frames, out...)
				}
				out, err := dec.FlushDelayedFrames()
				if err != nil {
					t.Fatalf("nalLengthSize=%d flush: %v", nalLengthSize, err)
				}
				frames = append(frames, out...)
				assertHighFrameMBAFFBSkipFrames(t, frames, tt)
			}
		})
	}
}

func TestFFmpegRawVideoFrameMD5OracleHigh1214FrameMBAFFBSkip(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}

	for _, tt := range highFrameMBAFFBSkipCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFBSkipFixture(tt)
			assertHighFrameMBAFFBSkipFixtureSyntax(t, data, tt)
			assertFFmpegHighFrameMBAFFBSkipRawVideoOracle(t, data, tt)
		})
	}
}

func highFrameMBAFFBSkipCases() []highFrameMBAFFBSkipCase {
	bitstreamMD5 := map[string]string{
		"High12TemporalDirectBSkipNoDeblock":     "d347a5b476f86141719a5e913ac1da12",
		"High12TemporalDirectBSkipFrameDeblock":  "0b0e42dfdd64d6a221b0dbc5b54cac01",
		"High12TemporalDirectBSkipSliceBoundary": "643ed93487e127a9dbb061ce4594676a",
		"High12SpatialDirectBSkipNoDeblock":      "5fea01257f878eeeaa6ccf41222970f9",
		"High12SpatialDirectBSkipFrameDeblock":   "debb6ec2cb822f8132867f48d029ce7c",
		"High12SpatialDirectBSkipSliceBoundary":  "5602c3fb99c560ca19f2229e60988af8",
		"High14TemporalDirectBSkipNoDeblock":     "1d8fa8115df758d58ff5af49520241bd",
		"High14TemporalDirectBSkipFrameDeblock":  "c41c2eef032904faa94a7a4a1aaa2682",
		"High14TemporalDirectBSkipSliceBoundary": "e2f1360028805ef859fb761928550fbd",
		"High14SpatialDirectBSkipNoDeblock":      "37ba96723b671629ccfcdcfa6dd21050",
		"High14SpatialDirectBSkipFrameDeblock":   "d4f476a39c5018745c6af10471c4c5bb",
		"High14SpatialDirectBSkipSliceBoundary":  "b166018195110013d9451d898889e9f0",
	}
	var out []highFrameMBAFFBSkipCase
	for _, bitDepth := range []int{12, 14} {
		frameMD5 := high12FrameMBAFFIntraPCMFrameMD5
		rawVideoMD5 := "94e77e8922a8b65ac84903483c1252ff"
		if bitDepth == 14 {
			frameMD5 = high14FrameMBAFFIntraPCMFrameMD5
			rawVideoMD5 = "389fb07fd25ac40b475b9b13d4e10b13"
		}
		for _, direct := range []struct {
			name string
			flag uint32
		}{
			{name: "TemporalDirect", flag: 0},
			{name: "SpatialDirect", flag: 1},
		} {
			for _, deblock := range []struct {
				name      string
				disableID uint32
				mode      int32
			}{
				{name: "NoDeblock", disableID: 1, mode: 0},
				{name: "FrameDeblock", disableID: 0, mode: 1},
				{name: "SliceBoundary", disableID: 2, mode: 2},
			} {
				name := fmt.Sprintf("High%d%sBSkip%s", bitDepth, direct.name, deblock.name)
				out = append(out, highFrameMBAFFBSkipCase{
					name:                       name,
					bitDepth:                   bitDepth,
					directSpatial:              direct.flag,
					disableDeblockingFilterIDC: deblock.disableID,
					deblockMode:                deblock.mode,
					bitstreamMD5:               highFrameMBAFFBSkipBitstreamMD5(bitstreamMD5, name),
					frameMD5:                   []string{frameMD5, frameMD5, frameMD5},
					rawVideoMD5:                rawVideoMD5,
				})
			}
		}
	}
	return out
}

func highFrameMBAFFBSkipBitstreamMD5(bitstreamMD5 map[string]string, name string) string {
	got, ok := bitstreamMD5[name]
	if !ok {
		panic(fmt.Sprintf("missing bitstream md5 for %s", name))
	}
	return got
}

func highFrameMBAFFBSkipFixture(tt highFrameMBAFFBSkipCase) []byte {
	var data []byte
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALSPS), highFrameMBAFFBInterSPSRBSP(tt.bitDepth)))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALPPS), highIntraPCMPPSRBSP(tt.bitDepth)))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALIDRSlice), highFrameMBAFFIntraPCMSliceRBSP(tt.bitDepth)))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALSlice), highFrameMBAFFPSkipNoResidualSliceRBSP()))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(h264.NALSlice), highFrameMBAFFBSkipSliceRBSP(2, tt.directSpatial, tt.disableDeblockingFilterIDC)))
	return data
}

func highFrameMBAFFBInterSPSRBSP(bitDepth int) []byte {
	var b decoderSEIBitBuilder
	b.writeBits(244, 8)
	b.writeBits(0, 8)
	b.writeBits(10, 8)
	b.writeUE(0)
	b.writeUE(1)
	b.writeUE(uint32(bitDepth - 8))
	b.writeUE(uint32(bitDepth - 8))
	b.writeBit(0)
	b.writeBit(0)
	b.writeUE(0)
	b.writeUE(2)
	b.writeUE(2)
	b.writeBit(0)
	b.writeUE(0)
	b.writeUE(0)
	b.writeBit(0)
	b.writeBit(1)
	b.writeBit(1)
	b.writeBit(0)
	b.writeBit(0)
	return b.rbsp()
}

func highFrameMBAFFBSkipSliceRBSP(frameNum uint32, directSpatial uint32, disableDeblockingFilterIDC uint32) []byte {
	var b decoderSEIBitBuilder
	b.writeUE(0)
	b.writeUE(1)
	b.writeUE(0)
	b.writeBits(frameNum, 4)
	b.writeBit(0)
	b.writeBit(directSpatial)
	b.writeBit(0)
	b.writeBit(0)
	b.writeBit(0)
	b.writeSE(0)
	writeHighCAVLCDeblockSyntax(&b, disableDeblockingFilterIDC)
	b.writeUE(2)
	return b.rbsp()
}

func assertHighFrameMBAFFBSkipFixtureSyntax(t *testing.T, data []byte, tt highFrameMBAFFBSkipCase) {
	t.Helper()
	sum := md5.Sum(data)
	if got := hex.EncodeToString(sum[:]); got != tt.bitstreamMD5 {
		t.Fatalf("%s frame-MBAFF B-skip bitstream md5 = %s, want %s", tt.name, got, tt.bitstreamMD5)
	}

	nals, err := h264.SplitAnnexB(data)
	if err != nil {
		t.Fatal(err)
	}
	var spsList [32]*h264.SPS
	var ppsList [256]*h264.PPS
	var gotTypes []int32
	var bNAL h264.NALUnit
	for i, nal := range nals {
		switch nal.Type {
		case h264.NALSPS:
			sps, err := h264.DecodeSPS(nal.RBSP)
			if err != nil {
				t.Fatal(err)
			}
			if i != 0 || sps.ProfileIDC != 244 || sps.Width != 16 || sps.Height != 32 ||
				sps.ChromaFormatIDC != 1 || sps.BitDepthLuma != int32(tt.bitDepth) ||
				sps.BitDepthChroma != int32(tt.bitDepth) || sps.RefFrameCount != 2 ||
				sps.FrameMBSOnlyFlag != 0 || sps.MBAFF != 1 || sps.Direct8x8InferenceFlag != 1 {
				t.Fatalf("SPS profile/format = %d %dx%d chroma %d depth %d/%d refs %d frame_mbs_only:%d mbaff:%d direct8x8:%d, want High%d 16x32 4:2:0 frame-MBAFF refs=2 direct8x8",
					sps.ProfileIDC, sps.Width, sps.Height, sps.ChromaFormatIDC,
					sps.BitDepthLuma, sps.BitDepthChroma, sps.RefFrameCount,
					sps.FrameMBSOnlyFlag, sps.MBAFF, sps.Direct8x8InferenceFlag, tt.bitDepth)
			}
			spsList[sps.SPSID] = sps
		case h264.NALPPS:
			pps, err := h264.DecodePPS(nal.RBSP, &spsList)
			if err != nil {
				t.Fatal(err)
			}
			if pps.CABAC != 0 || pps.Transform8x8Mode != 0 || pps.WeightedPred != 0 ||
				pps.WeightedBipredIDC != 0 || pps.RefCount != [2]uint32{1, 1} ||
				pps.DeblockingFilterParametersPresent == 0 {
				t.Fatalf("PPS CABAC/8x8/weights/refs/deblock = %d/%d/%d/%d/%v/%d, want CAVLC/no-8x8/unweighted refs=1/1 deblock params",
					pps.CABAC, pps.Transform8x8Mode, pps.WeightedPred,
					pps.WeightedBipredIDC, pps.RefCount, pps.DeblockingFilterParametersPresent)
			}
			ppsList[pps.PPSID] = pps
		case h264.NALIDRSlice, h264.NALSlice:
			sh, err := h264.ParseSliceHeader(nal, &ppsList)
			if err != nil {
				t.Fatal(err)
			}
			if sh.PictureStructure != h264.PictureFrame || sh.QScale != 26 || sh.SPS.MBAFF != 1 {
				t.Fatalf("slice picture/q/mbaff = %d/%d/%d, want frame/26/1", sh.PictureStructure, sh.QScale, sh.SPS.MBAFF)
			}
			if sh.SliceTypeNoS == h264.PictureTypeB {
				if nal.RefIDC != 0 || sh.ListCount != 2 || sh.RefCount != [2]uint32{1, 1} ||
					sh.DirectSpatialMVPred != int32(tt.directSpatial) || sh.DeblockingFilter != tt.deblockMode {
					t.Fatalf("B ref/lists/refs/direct/deblock = %d/%d/%v/%d/%d, want non-ref B refs=1/1 direct=%d deblock=%d",
						nal.RefIDC, sh.ListCount, sh.RefCount, sh.DirectSpatialMVPred, sh.DeblockingFilter,
						tt.directSpatial, tt.deblockMode)
				}
				bNAL = nal
			}
			gotTypes = append(gotTypes, sh.SliceTypeNoS)
		default:
			t.Fatalf("unexpected NAL type %d in %s frame-MBAFF B-skip fixture", nal.Type, tt.name)
		}
	}
	wantTypes := []int32{h264.PictureTypeI, h264.PictureTypeP, h264.PictureTypeB}
	if !highFrameMBAFFBSkipInt32SlicesEqual(gotTypes, wantTypes) {
		t.Fatalf("slice types = %v, want %v", gotTypes, wantTypes)
	}
	if bNAL.RBSP == nil {
		t.Fatal("missing B slice")
	}
	skipRun := readHighFrameMBAFFBSkipRun(t, bNAL, spsList[0], ppsList[0], tt)
	if skipRun != 2 {
		t.Fatalf("%s B mb_skip_run = %d, want frame-coded pair skip_run 2", tt.name, skipRun)
	}
}

func readHighFrameMBAFFBSkipRun(t *testing.T, nal h264.NALUnit, sps *h264.SPS, pps *h264.PPS, tt highFrameMBAFFBSkipCase) uint32 {
	t.Helper()
	if sps == nil || pps == nil {
		t.Fatal("missing SPS/PPS for frame-MBAFF B-skip syntax check")
	}
	br := high10ResidualCAVLCBitReader{data: nal.RBSP}
	firstMB := br.readUE(t)
	rawSliceType := br.readUE(t)
	sliceTypeNoS := high10ResidualCAVLCSliceTypeNoS(t, rawSliceType)
	ppsID := br.readUE(t)
	if firstMB != 0 || sliceTypeNoS != h264.PictureTypeB || ppsID != pps.PPSID {
		t.Fatalf("slice header first_mb/type/pps = %d/%d/%d, want first B-skip slice with PPS %d", firstMB, sliceTypeNoS, ppsID, pps.PPSID)
	}
	if frameNum := br.readBits(t, int(sps.Log2MaxFrameNum)); frameNum != 2 {
		t.Fatalf("B frame_num = %d, want 2", frameNum)
	}
	if fieldPic := br.readBit(t); fieldPic != 0 {
		t.Fatalf("field_pic_flag = %d, want frame picture", fieldPic)
	}
	if direct := br.readBit(t); direct != tt.directSpatial {
		t.Fatalf("direct_spatial_mv_pred_flag = %d, want %d", direct, tt.directSpatial)
	}
	refCount := pps.RefCount
	if br.readBit(t) != 0 {
		refCount[0] = br.readUE(t) + 1
		refCount[1] = br.readUE(t) + 1
	}
	if refCount != [2]uint32{1, 1} {
		t.Fatalf("B ref counts = %v, want 1/1", refCount)
	}
	high10ResidualCAVLCReadRefPicListModifications(t, &br, 2)
	high10ResidualCAVLCReadRefPicMarking(t, &br, nal)
	if pps.CABAC != 0 {
		br.readUE(t)
	}
	if qpDelta := br.readSE(t); qpDelta != 0 {
		t.Fatalf("slice_qp_delta = %d, want 0", qpDelta)
	}
	if pps.DeblockingFilterParametersPresent != 0 {
		disableID := br.readUE(t)
		if disableID != tt.disableDeblockingFilterIDC {
			t.Fatalf("disable_deblocking_filter_idc = %d, want %d", disableID, tt.disableDeblockingFilterIDC)
		}
		if disableID != 1 {
			if alpha := br.readSE(t); alpha != 0 {
				t.Fatalf("slice_alpha_c0_offset_div2 = %d, want 0", alpha)
			}
			if beta := br.readSE(t); beta != 0 {
				t.Fatalf("slice_beta_offset_div2 = %d, want 0", beta)
			}
		}
	}
	return br.readUE(t)
}

func assertHighFrameMBAFFBSkipFrames(t *testing.T, frames []*Frame, tt highFrameMBAFFBSkipCase) {
	t.Helper()
	if len(frames) != len(tt.frameMD5) {
		t.Fatalf("frames = %d, want %d", len(frames), len(tt.frameMD5))
	}
	var rawVideo []byte
	for i, frame := range frames {
		if frame == nil {
			t.Fatalf("frame[%d] is nil", i)
		}
		if frame.Width != 16 || frame.Height != 32 || frame.ChromaFormatIDC != 1 ||
			frame.BitDepthLuma != tt.bitDepth || frame.BitDepthChroma != tt.bitDepth {
			t.Fatalf("frame[%d] format = %dx%d chroma %d depth %d/%d, want 16x32 yuv420p%dle",
				i, frame.Width, frame.Height, frame.ChromaFormatIDC, frame.BitDepthLuma, frame.BitDepthChroma, tt.bitDepth)
		}
		raw, err := frame.AppendRawYUVBytesLE(nil)
		if err != nil {
			t.Fatalf("frame[%d] AppendRawYUVBytesLE: %v", i, err)
		}
		rawVideo = append(rawVideo, raw...)
		sum := md5.Sum(raw)
		if got := hex.EncodeToString(sum[:]); got != tt.frameMD5[i] {
			t.Fatalf("frame[%d] md5 = %s, want %s", i, got, tt.frameMD5[i])
		}
	}
	if len(rawVideo) != 4608 {
		t.Fatalf("rawvideo len = %d, want 4608", len(rawVideo))
	}
	sum := md5.Sum(rawVideo)
	if got := hex.EncodeToString(sum[:]); got != tt.rawVideoMD5 {
		t.Fatalf("rawvideo md5 = %s, want %s", got, tt.rawVideoMD5)
	}
}

func assertFFmpegHighFrameMBAFFBSkipRawVideoOracle(t *testing.T, data []byte, tt highFrameMBAFFBSkipCase) {
	t.Helper()
	path := writeTempH264(t, data)
	pixFmt := fmt.Sprintf("yuv420p%dle", tt.bitDepth)
	framemd5 := exec.Command(
		"ffmpeg",
		"-v", "error",
		"-f", "h264",
		"-i", path,
		"-an", "-sn", "-dn",
		"-pix_fmt", pixFmt,
		"-f", "framemd5",
		"-",
	)
	framemd5Out, err := framemd5.Output()
	if err != nil {
		t.Fatalf("ffmpeg framemd5: %v", err)
	}
	for i, want := range tt.frameMD5 {
		line := []byte(fmt.Sprintf("0, %10d, %10d,        1,     1536, %s", i, i, want))
		if !bytes.Contains(framemd5Out, line) {
			t.Fatalf("missing %q in framemd5:\n%s", line, framemd5Out)
		}
	}

	rawvideo := exec.Command(
		"ffmpeg",
		"-v", "error",
		"-f", "h264",
		"-i", path,
		"-an", "-sn", "-dn",
		"-pix_fmt", pixFmt,
		"-f", "rawvideo",
		"-",
	)
	raw, err := rawvideo.Output()
	if err != nil {
		t.Fatalf("ffmpeg rawvideo: %v", err)
	}
	if len(raw) != 4608 {
		t.Fatalf("rawvideo size = %d, want 4608", len(raw))
	}
	sum := md5.Sum(raw)
	if got := hex.EncodeToString(sum[:]); got != tt.rawVideoMD5 {
		t.Fatalf("rawvideo md5 = %s, want %s", got, tt.rawVideoMD5)
	}
}

func highFrameMBAFFBSkipInt32SlicesEqual(a, b []int32) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
