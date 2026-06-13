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
	high12FrameMBAFFIntraPCMBitstreamMD5 = "cbe015bbf429e401bfc53c2827ca6395"
	high12FrameMBAFFIntraPCMFrameMD5     = "5d168280547309a62ad1066a599c4ba5"
	high12FrameMBAFFIntraPCMRawVideoMD5  = "5d168280547309a62ad1066a599c4ba5"

	high14FrameMBAFFIntraPCMBitstreamMD5 = "7b0f8bb4f259d41f621bc0d5c5f22031"
	high14FrameMBAFFIntraPCMFrameMD5     = "7da709fea95b8767edeb8e5963b37f2a"
	high14FrameMBAFFIntraPCMRawVideoMD5  = "7da709fea95b8767edeb8e5963b37f2a"
)

type highFrameMBAFFIntraPCMCase struct {
	name         string
	bitDepth     int
	bitstreamMD5 string
	frameMD5     string
	rawVideoMD5  string
}

func TestHigh1214FrameMBAFFIntraPCMFixtureSyntax(t *testing.T) {
	for _, tt := range highFrameMBAFFIntraPCMCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFIntraPCMFixture(tt.bitDepth)
			sum := md5.Sum(data)
			if got := hex.EncodeToString(sum[:]); got != tt.bitstreamMD5 {
				t.Fatalf("%s frame-MBAFF IntraPCM bitstream md5 = %s, want %s", tt.name, got, tt.bitstreamMD5)
			}
			assertHighFrameMBAFFIntraPCMFixtureSyntax(t, data, tt)
		})
	}
}

func TestDecodeAnnexBHigh1214FrameMBAFFIntraPCMFrames(t *testing.T) {
	for _, tt := range highFrameMBAFFIntraPCMCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFIntraPCMFixture(tt.bitDepth)
			assertHighFrameMBAFFIntraPCMFixtureSyntax(t, data, tt)

			frames, err := NewDecoder().DecodeAnnexBFrames(data)
			if err != nil {
				t.Fatalf("decode %s frame-MBAFF IntraPCM Annex B: %v", tt.name, err)
			}
			assertHighFrameMBAFFIntraPCMFrames(t, frames, tt)
		})
	}
}

func TestDecodeAVCHigh1214FrameMBAFFIntraPCMFrames(t *testing.T) {
	for _, tt := range highFrameMBAFFIntraPCMCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFIntraPCMFixture(tt.bitDepth)
			assertHighFrameMBAFFIntraPCMFixtureSyntax(t, data, tt)

			for _, nalLengthSize := range []int{2, 3, 4} {
				frames, err := NewDecoder().DecodeAVCFrames(annexBToAVC(t, data, nalLengthSize), nalLengthSize)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: decode %s frame-MBAFF IntraPCM AVC: %v", nalLengthSize, tt.name, err)
				}
				assertHighFrameMBAFFIntraPCMFrames(t, frames, tt)
			}
		})
	}
}

func TestDecodeAVCCHigh1214FrameMBAFFIntraPCMFrames(t *testing.T) {
	for _, tt := range highFrameMBAFFIntraPCMCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFIntraPCMFixture(tt.bitDepth)
			assertHighFrameMBAFFIntraPCMFixtureSyntax(t, data, tt)

			for _, nalLengthSize := range []int{2, 3, 4} {
				config, packet := annexBToAVCConfigAndPacket(t, data, nalLengthSize)
				frames, err := NewDecoder().DecodeAVCCFrames(config, packet)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: decode %s frame-MBAFF IntraPCM configured AVC: %v", nalLengthSize, tt.name, err)
				}
				assertHighFrameMBAFFIntraPCMFrames(t, frames, tt)
			}
		})
	}
}

func TestFFmpegRawVideoMD5OracleHigh1214FrameMBAFFIntraPCM(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}

	for _, tt := range highFrameMBAFFIntraPCMCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFIntraPCMFixture(tt.bitDepth)
			assertHighFrameMBAFFIntraPCMFixtureSyntax(t, data, tt)
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
			line := []byte(fmt.Sprintf("0, %10d, %10d,        1,     1536, %s", 0, 0, tt.frameMD5))
			if !bytes.Contains(framemd5Out, line) {
				t.Fatalf("missing %q in framemd5:\n%s", line, framemd5Out)
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
			if len(raw) != 1536 {
				t.Fatalf("rawvideo size = %d, want 1536", len(raw))
			}
			sum := md5.Sum(raw)
			if got := hex.EncodeToString(sum[:]); got != tt.rawVideoMD5 {
				t.Fatalf("rawvideo md5 = %s, want %s", got, tt.rawVideoMD5)
			}
		})
	}
}

func highFrameMBAFFIntraPCMCases() []highFrameMBAFFIntraPCMCase {
	return []highFrameMBAFFIntraPCMCase{
		{
			name:         "High12",
			bitDepth:     12,
			bitstreamMD5: high12FrameMBAFFIntraPCMBitstreamMD5,
			frameMD5:     high12FrameMBAFFIntraPCMFrameMD5,
			rawVideoMD5:  high12FrameMBAFFIntraPCMRawVideoMD5,
		},
		{
			name:         "High14",
			bitDepth:     14,
			bitstreamMD5: high14FrameMBAFFIntraPCMBitstreamMD5,
			frameMD5:     high14FrameMBAFFIntraPCMFrameMD5,
			rawVideoMD5:  high14FrameMBAFFIntraPCMRawVideoMD5,
		},
	}
}

func highFrameMBAFFIntraPCMFixture(bitDepth int) []byte {
	var data []byte
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALSPS), highFrameMBAFFSPSRBSP(bitDepth)))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALPPS), highIntraPCMPPSRBSP(bitDepth)))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALIDRSlice), highFrameMBAFFIntraPCMSliceRBSP(bitDepth)))
	return data
}

func highFrameMBAFFSPSRBSP(bitDepth int) []byte {
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
	b.writeUE(0)
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

func highFrameMBAFFIntraPCMSliceRBSP(bitDepth int) []byte {
	var b decoderSEIBitBuilder
	b.writeUE(0)
	b.writeUE(2)
	b.writeUE(0)
	b.writeBits(0, 4)
	b.writeBit(0)
	b.writeUE(0)
	b.writeBit(0)
	b.writeBit(0)
	b.writeSE(0)
	b.writeUE(1)

	b.writeBit(1)
	b.writeUE(25)
	highIntraPCMByteAlign(&b)
	rbsp := b.bytes()
	rbsp = append(rbsp, highIntraPCMBytes(bitDepth, 37)...)

	var next decoderSEIBitBuilder
	next.writeUE(25)
	highIntraPCMByteAlign(&next)
	rbsp = append(rbsp, next.bytes()...)
	rbsp = append(rbsp, highIntraPCMBytes(bitDepth, 91)...)
	return append(rbsp, 0x80)
}

func assertHighFrameMBAFFIntraPCMFixtureSyntax(t *testing.T, data []byte, tt highFrameMBAFFIntraPCMCase) {
	t.Helper()
	nals, err := h264.SplitAnnexB(data)
	if err != nil {
		t.Fatal(err)
	}

	var spsList [32]*h264.SPS
	var ppsList [256]*h264.PPS
	var gotVCL []h264.NALUnitType
	for _, nal := range nals {
		switch nal.Type {
		case h264.NALSPS:
			sps, err := h264.DecodeSPS(nal.RBSP)
			if err != nil {
				t.Fatal(err)
			}
			if sps.ProfileIDC != 244 || sps.Width != 16 || sps.Height != 32 ||
				sps.ChromaFormatIDC != 1 || sps.BitDepthLuma != int32(tt.bitDepth) ||
				sps.BitDepthChroma != int32(tt.bitDepth) || sps.FrameMBSOnlyFlag != 0 ||
				sps.MBAFF != 1 {
				t.Fatalf("SPS profile/format = %d %dx%d chroma %d depth %d/%d frame_mbs_only:%d mbaff:%d, want High 4:4:4 Predictive-compatible 16x32 yuv420p%dle frame-MBAFF",
					sps.ProfileIDC, sps.Width, sps.Height, sps.ChromaFormatIDC,
					sps.BitDepthLuma, sps.BitDepthChroma, sps.FrameMBSOnlyFlag, sps.MBAFF,
					tt.bitDepth)
			}
			spsList[sps.SPSID] = sps
		case h264.NALPPS:
			pps, err := h264.DecodePPS(nal.RBSP, &spsList)
			if err != nil {
				t.Fatal(err)
			}
			if pps.CABAC != 0 || pps.DeblockingFilterParametersPresent == 0 {
				t.Fatalf("PPS CABAC/deblock-present = %d/%d, want CAVLC/deblock params", pps.CABAC, pps.DeblockingFilterParametersPresent)
			}
			ppsList[pps.PPSID] = pps
		case h264.NALIDRSlice, h264.NALSlice:
			sh, err := h264.ParseSliceHeader(nal, &ppsList)
			if err != nil {
				t.Fatal(err)
			}
			if sh.PictureStructure != h264.PictureFrame || sh.SliceTypeNoS != h264.PictureTypeI ||
				sh.DeblockingFilter != 0 || sh.QScale != 26 || sh.SPS.MBAFF != 1 {
				t.Fatalf("slice picture/type/deblock/qp/mbaff = %d/%d/%d/%d/%d, want frame/I/disabled/26/1",
					sh.PictureStructure, sh.SliceTypeNoS, sh.DeblockingFilter, sh.QScale, sh.SPS.MBAFF)
			}
			gotVCL = append(gotVCL, nal.Type)
		}
	}
	if len(gotVCL) != 1 || gotVCL[0] != h264.NALIDRSlice {
		t.Fatalf("VCL NALs = %v, want one IDR slice", gotVCL)
	}
}

func assertHighFrameMBAFFIntraPCMFrames(t *testing.T, frames []*Frame, tt highFrameMBAFFIntraPCMCase) {
	t.Helper()
	if len(frames) != 1 {
		t.Fatalf("frames = %d, want 1", len(frames))
	}
	frame := frames[0]
	if frame.Width != 16 || frame.Height != 32 ||
		frame.ChromaFormatIDC != 1 || frame.BitDepthLuma != tt.bitDepth ||
		frame.BitDepthChroma != tt.bitDepth {
		t.Fatalf("frame format = %dx%d chroma %d depth %d/%d, want 16x32 yuv420p%dle",
			frame.Width, frame.Height, frame.ChromaFormatIDC, frame.BitDepthLuma, frame.BitDepthChroma,
			tt.bitDepth)
	}
	if pixFmt, err := frame.RawPixelFormat(); err != nil || pixFmt != fmt.Sprintf("yuv420p%dle", tt.bitDepth) {
		t.Fatalf("RawPixelFormat = %q/%v, want yuv420p%dle/nil", pixFmt, err, tt.bitDepth)
	}
	if size, err := frame.RawYUVSize(); err != nil || size != 1536 {
		t.Fatalf("RawYUVSize = %d/%v, want 1536/nil", size, err)
	}
	raw, err := frame.AppendRawYUVBytesLE(nil)
	if err != nil {
		t.Fatalf("AppendRawYUVBytesLE: %v", err)
	}
	sum := md5.Sum(raw)
	if got := hex.EncodeToString(sum[:]); got != tt.frameMD5 {
		t.Fatalf("frame raw md5 = %s, want %s", got, tt.frameMD5)
	}
	if _, err := frame.AppendRawYUV(nil); err != ErrUnsupported {
		t.Fatalf("AppendRawYUV high%d error = %v, want ErrUnsupported", tt.bitDepth, err)
	}
}
