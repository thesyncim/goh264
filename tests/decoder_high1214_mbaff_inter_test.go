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

const (
	high12FrameMBAFFP16x16NoResidualBitstreamMD5 = "1bdb8c58f3a8a6a7f2d9802921c74361"
	high12FrameMBAFFP16x16NoResidualPFrameMD5    = "5d168280547309a62ad1066a599c4ba5"
	high12FrameMBAFFP16x16NoResidualRawVideoMD5  = "5d1af51e10e3ea2a87b530ca462543c2"

	high14FrameMBAFFP16x16NoResidualBitstreamMD5 = "fb218b8d8ed9f2c46171ceeb150b46c0"
	high14FrameMBAFFP16x16NoResidualPFrameMD5    = "7da709fea95b8767edeb8e5963b37f2a"
	high14FrameMBAFFP16x16NoResidualRawVideoMD5  = "237561940e2f07d30cac465fcea640bb"

	highFrameMBAFFP16x16NoResidualPayloadBits = "1111111111111"
)

type highFrameMBAFFP16x16NoResidualCase struct {
	name         string
	bitDepth     int
	bitstreamMD5 string
	refFrameMD5  string
	pFrameMD5    string
	rawVideoMD5  string
}

func TestHigh1214FrameMBAFFP16x16NoResidualFixtureSyntax(t *testing.T) {
	for _, tt := range highFrameMBAFFP16x16NoResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFP16x16NoResidualFixture(tt.bitDepth)
			sum := md5.Sum(data)
			if got := hex.EncodeToString(sum[:]); got != tt.bitstreamMD5 {
				t.Fatalf("%s frame-MBAFF P16x16 bitstream md5 = %s, want %s", tt.name, got, tt.bitstreamMD5)
			}
			assertHighFrameMBAFFP16x16NoResidualFixtureSyntax(t, data, tt)
		})
	}
}

func TestDecodeAnnexBHigh1214FrameMBAFFP16x16NoResidualFrames(t *testing.T) {
	for _, tt := range highFrameMBAFFP16x16NoResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFP16x16NoResidualFixture(tt.bitDepth)
			assertHighFrameMBAFFP16x16NoResidualFixtureSyntax(t, data, tt)

			frames, err := NewDecoder().DecodeAnnexBFrames(data)
			if err != nil {
				t.Fatalf("decode %s frame-MBAFF P16x16 Annex B: %v", tt.name, err)
			}
			assertHighFrameMBAFFP16x16NoResidualFrames(t, frames, tt)
		})
	}
}

func TestDecodeAVCHigh1214FrameMBAFFP16x16NoResidualFrames(t *testing.T) {
	for _, tt := range highFrameMBAFFP16x16NoResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFP16x16NoResidualFixture(tt.bitDepth)
			assertHighFrameMBAFFP16x16NoResidualFixtureSyntax(t, data, tt)

			for _, nalLengthSize := range []int{2, 3, 4} {
				frames, err := NewDecoder().DecodeAVCFrames(annexBToAVC(t, data, nalLengthSize), nalLengthSize)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: decode %s frame-MBAFF P16x16 AVC: %v", nalLengthSize, tt.name, err)
				}
				assertHighFrameMBAFFP16x16NoResidualFrames(t, frames, tt)
			}
		})
	}
}

func TestDecodeAVCWithConfigurationRecordHigh1214FrameMBAFFP16x16NoResidualFrames(t *testing.T) {
	for _, tt := range highFrameMBAFFP16x16NoResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFP16x16NoResidualFixture(tt.bitDepth)
			assertHighFrameMBAFFP16x16NoResidualFixtureSyntax(t, data, tt)

			for _, nalLengthSize := range []int{2, 3, 4} {
				config, packet := annexBToAVCConfigAndPacket(t, data, nalLengthSize)
				frames, err := NewDecoder().DecodeAVCFramesWithConfigurationRecord(config, packet)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: decode %s frame-MBAFF P16x16 configured AVC: %v", nalLengthSize, tt.name, err)
				}
				assertHighFrameMBAFFP16x16NoResidualFrames(t, frames, tt)
			}
		})
	}
}

func TestFFmpegRawVideoMD5OracleHigh1214FrameMBAFFP16x16NoResidual(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}

	for _, tt := range highFrameMBAFFP16x16NoResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFP16x16NoResidualFixture(tt.bitDepth)
			assertHighFrameMBAFFP16x16NoResidualFixtureSyntax(t, data, tt)
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
			for i, want := range []string{tt.refFrameMD5, tt.pFrameMD5} {
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
			if len(raw) != 3072 {
				t.Fatalf("rawvideo size = %d, want 3072", len(raw))
			}
			sum := md5.Sum(raw)
			if got := hex.EncodeToString(sum[:]); got != tt.rawVideoMD5 {
				t.Fatalf("rawvideo md5 = %s, want %s", got, tt.rawVideoMD5)
			}
		})
	}
}

func highFrameMBAFFP16x16NoResidualCases() []highFrameMBAFFP16x16NoResidualCase {
	return []highFrameMBAFFP16x16NoResidualCase{
		{
			name:         "High12",
			bitDepth:     12,
			bitstreamMD5: high12FrameMBAFFP16x16NoResidualBitstreamMD5,
			refFrameMD5:  high12FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:    high12FrameMBAFFP16x16NoResidualPFrameMD5,
			rawVideoMD5:  high12FrameMBAFFP16x16NoResidualRawVideoMD5,
		},
		{
			name:         "High14",
			bitDepth:     14,
			bitstreamMD5: high14FrameMBAFFP16x16NoResidualBitstreamMD5,
			refFrameMD5:  high14FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:    high14FrameMBAFFP16x16NoResidualPFrameMD5,
			rawVideoMD5:  high14FrameMBAFFP16x16NoResidualRawVideoMD5,
		},
	}
}

func highFrameMBAFFP16x16NoResidualFixture(bitDepth int) []byte {
	var data []byte
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALSPS), highFrameMBAFFInterSPSRBSP(bitDepth)))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALPPS), highIntraPCMPPSRBSP(bitDepth)))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALIDRSlice), highFrameMBAFFIntraPCMSliceRBSP(bitDepth)))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALSlice), highFrameMBAFFP16x16NoResidualSliceRBSP()))
	return data
}

func highFrameMBAFFInterSPSRBSP(bitDepth int) []byte {
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
	b.writeUE(1)
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

func highFrameMBAFFP16x16NoResidualSliceRBSP() []byte {
	var b decoderSEIBitBuilder
	b.writeUE(0)
	b.writeUE(0)
	b.writeUE(0)
	b.writeBits(1, 4)
	b.writeBit(0)
	b.writeBit(0)
	b.writeBit(0)
	b.writeBit(0)
	b.writeSE(0)
	b.writeUE(1)
	highIntra16x16WritePayloadBits(&b, highFrameMBAFFP16x16NoResidualPayloadBits)
	return b.rbsp()
}

func assertHighFrameMBAFFP16x16NoResidualFixtureSyntax(t *testing.T, data []byte, tt highFrameMBAFFP16x16NoResidualCase) {
	t.Helper()
	nals, err := h264.SplitAnnexB(data)
	if err != nil {
		t.Fatal(err)
	}

	var spsList [32]*h264.SPS
	var ppsList [256]*h264.PPS
	var gotVCL []h264.NALUnitType
	var gotSliceTypes []int32
	for _, nal := range nals {
		switch nal.Type {
		case h264.NALSPS:
			sps, err := h264.DecodeSPS(nal.RBSP)
			if err != nil {
				t.Fatal(err)
			}
			if sps.ProfileIDC != 244 || sps.Width != 16 || sps.Height != 32 ||
				sps.ChromaFormatIDC != 1 || sps.BitDepthLuma != int32(tt.bitDepth) ||
				sps.BitDepthChroma != int32(tt.bitDepth) || sps.RefFrameCount != 1 ||
				sps.FrameMBSOnlyFlag != 0 || sps.MBAFF != 1 {
				t.Fatalf("SPS profile/format = %d %dx%d chroma %d depth %d/%d refs %d frame_mbs_only:%d mbaff:%d, want High 4:4:4 Predictive-compatible 16x32 yuv420p%dle ref frame-MBAFF",
					sps.ProfileIDC, sps.Width, sps.Height, sps.ChromaFormatIDC,
					sps.BitDepthLuma, sps.BitDepthChroma, sps.RefFrameCount,
					sps.FrameMBSOnlyFlag, sps.MBAFF, tt.bitDepth)
			}
			spsList[sps.SPSID] = sps
		case h264.NALPPS:
			pps, err := h264.DecodePPS(nal.RBSP, &spsList)
			if err != nil {
				t.Fatal(err)
			}
			if pps.CABAC != 0 || pps.Transform8x8Mode != 0 || pps.DeblockingFilterParametersPresent == 0 ||
				pps.RefCount[0] != 1 || pps.RefCount[1] != 1 {
				t.Fatalf("PPS CABAC/8x8/deblock-present/refs = %d/%d/%d/%d/%d, want CAVLC/no-8x8/deblock params/1/1",
					pps.CABAC, pps.Transform8x8Mode, pps.DeblockingFilterParametersPresent, pps.RefCount[0], pps.RefCount[1])
			}
			ppsList[pps.PPSID] = pps
		case h264.NALIDRSlice, h264.NALSlice:
			sh, err := h264.ParseSliceHeader(nal, &ppsList)
			if err != nil {
				t.Fatal(err)
			}
			if sh.PictureStructure != h264.PictureFrame || sh.DeblockingFilter != 0 ||
				sh.QScale != 26 || sh.SPS.MBAFF != 1 {
				t.Fatalf("slice picture/deblock/qp/mbaff = %d/%d/%d/%d, want frame/disabled/26/1",
					sh.PictureStructure, sh.DeblockingFilter, sh.QScale, sh.SPS.MBAFF)
			}
			if sh.SliceTypeNoS == h264.PictureTypeP && (sh.ListCount != 1 || sh.RefCount[0] != 1) {
				t.Fatalf("P slice lists/refs = %d/%v, want one L0 ref", sh.ListCount, sh.RefCount)
			}
			gotVCL = append(gotVCL, nal.Type)
			gotSliceTypes = append(gotSliceTypes, sh.SliceTypeNoS)
		default:
			t.Fatalf("unexpected NAL type %d in %s frame-MBAFF P16x16 fixture", nal.Type, tt.name)
		}
	}
	if len(gotVCL) != 2 || gotVCL[0] != h264.NALIDRSlice || gotVCL[1] != h264.NALSlice {
		t.Fatalf("VCL NALs = %v, want IDR slice followed by non-IDR slice", gotVCL)
	}
	if len(gotSliceTypes) != 2 || gotSliceTypes[0] != h264.PictureTypeI || gotSliceTypes[1] != h264.PictureTypeP {
		t.Fatalf("slice types = %v, want I then P", gotSliceTypes)
	}
}

func assertHighFrameMBAFFP16x16NoResidualFrames(t *testing.T, frames []*Frame, tt highFrameMBAFFP16x16NoResidualCase) {
	t.Helper()
	if len(frames) != 2 {
		t.Fatalf("frames = %d, want 2", len(frames))
	}
	for i, frame := range frames {
		if frame.Width != 16 || frame.Height != 32 ||
			frame.ChromaFormatIDC != 1 || frame.BitDepthLuma != tt.bitDepth ||
			frame.BitDepthChroma != tt.bitDepth {
			t.Fatalf("frame[%d] format = %dx%d chroma %d depth %d/%d, want 16x32 yuv420p%dle",
				i, frame.Width, frame.Height, frame.ChromaFormatIDC, frame.BitDepthLuma, frame.BitDepthChroma,
				tt.bitDepth)
		}
		if pixFmt, err := frame.RawPixelFormat(); err != nil || pixFmt != fmt.Sprintf("yuv420p%dle", tt.bitDepth) {
			t.Fatalf("frame[%d] RawPixelFormat = %q/%v, want yuv420p%dle/nil", i, pixFmt, err, tt.bitDepth)
		}
		if size, err := frame.RawYUVSize(); err != nil || size != 1536 {
			t.Fatalf("frame[%d] RawYUVSize = %d/%v, want 1536/nil", i, size, err)
		}
		raw, err := frame.AppendRawYUVBytesLE(nil)
		if err != nil {
			t.Fatalf("frame[%d] AppendRawYUVBytesLE: %v", i, err)
		}
		want := tt.refFrameMD5
		if i == 1 {
			want = tt.pFrameMD5
		}
		sum := md5.Sum(raw)
		if got := hex.EncodeToString(sum[:]); got != want {
			t.Fatalf("frame[%d] raw md5 = %s, want %s", i, got, want)
		}
		if _, err := frame.AppendRawYUV(nil); err != ErrUnsupported {
			t.Fatalf("frame[%d] AppendRawYUV high%d error = %v, want ErrUnsupported", i, tt.bitDepth, err)
		}
	}
}
