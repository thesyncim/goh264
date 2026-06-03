// SPDX-License-Identifier: LGPL-2.1-or-later

package goh264

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/thesyncim/goh264/internal/h264"
)

const (
	high12LumaChromaResidualBitstreamMD5 = "9022341e9cfe2a1dce56fe9efdec8c43"
	high12LumaChromaResidualFrameMD5     = "2d20c856bd29c6daf34dbd29c8fae3ea"
	high12LumaChromaResidualRawVideoMD5  = "2d20c856bd29c6daf34dbd29c8fae3ea"
	high12LumaChromaResidualRawSize      = 4608
)

func TestHigh12LumaChromaResidualFixtureSyntax(t *testing.T) {
	assertHigh12LumaChromaResidualFixtureSyntax(t, readHigh12LumaChromaResidualFixture(t))
}

func TestDecodeAnnexBHigh12LumaChromaResidualFrame(t *testing.T) {
	data := readHigh12LumaChromaResidualFixture(t)
	assertHigh12LumaChromaResidualFixtureSyntax(t, data)

	frames, err := NewDecoder().DecodeAnnexBFrames(data)
	if err != nil {
		t.Fatalf("decode High12 luma/chroma residual Annex B: %v", err)
	}
	assertHigh12LumaChromaResidualFrames(t, frames)
}

func TestDecodeAVCHigh12LumaChromaResidualFrame(t *testing.T) {
	data := readHigh12LumaChromaResidualFixture(t)
	assertHigh12LumaChromaResidualFixtureSyntax(t, data)

	for _, nalLengthSize := range []int{2, 3, 4} {
		frames, err := NewDecoder().DecodeAVCFrames(annexBToAVC(t, data, nalLengthSize), nalLengthSize)
		if err != nil {
			t.Fatalf("nalLengthSize=%d: %v", nalLengthSize, err)
		}
		assertHigh12LumaChromaResidualFrames(t, frames)
	}
}

func TestDecodeAVCWithConfigurationRecordHigh12LumaChromaResidualFrame(t *testing.T) {
	data := readHigh12LumaChromaResidualFixture(t)
	assertHigh12LumaChromaResidualFixtureSyntax(t, data)

	for _, nalLengthSize := range []int{2, 3, 4} {
		config, packet := annexBToAVCConfigAndPacket(t, data, nalLengthSize)
		frames, err := NewDecoder().DecodeAVCFramesWithConfigurationRecord(config, packet)
		if err != nil {
			t.Fatalf("nalLengthSize=%d: %v", nalLengthSize, err)
		}
		assertHigh12LumaChromaResidualFrames(t, frames)
	}
}

func TestFFmpegRawVideoMD5OracleHigh12LumaChromaResidual(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}

	path := high12LumaChromaResidualFixturePath(t)
	framemd5 := exec.Command("ffmpeg",
		"-v", "error",
		"-xerror",
		"-f", "h264",
		"-i", path,
		"-an", "-sn", "-dn",
		"-pix_fmt", "yuv420p12le",
		"-f", "framemd5",
		"-",
	)
	framemd5Out, err := framemd5.Output()
	if err != nil {
		t.Fatalf("ffmpeg framemd5: %v", err)
	}
	line := []byte(fmt.Sprintf("0, %10d, %10d,        1, %8d, %s", 0, 0, high12LumaChromaResidualRawSize, high12LumaChromaResidualFrameMD5))
	if !bytes.Contains(framemd5Out, line) {
		t.Fatalf("missing %q in framemd5:\n%s", line, framemd5Out)
	}

	rawvideo := exec.Command("ffmpeg",
		"-v", "error",
		"-xerror",
		"-f", "h264",
		"-i", path,
		"-an", "-sn", "-dn",
		"-pix_fmt", "yuv420p12le",
		"-f", "rawvideo",
		"-",
	)
	raw, err := rawvideo.Output()
	if err != nil {
		t.Fatalf("ffmpeg rawvideo: %v", err)
	}
	if len(raw) != high12LumaChromaResidualRawSize {
		t.Fatalf("rawvideo size = %d, want %d", len(raw), high12LumaChromaResidualRawSize)
	}
	sum := md5.Sum(raw)
	if got := hex.EncodeToString(sum[:]); got != high12LumaChromaResidualRawVideoMD5 {
		t.Fatalf("rawvideo md5 = %s, want %s", got, high12LumaChromaResidualRawVideoMD5)
	}
}

func readHigh12LumaChromaResidualFixture(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile(high12LumaChromaResidualFixturePath(t))
	if err != nil {
		t.Fatal(err)
	}
	sum := md5.Sum(data)
	if got := hex.EncodeToString(sum[:]); got != high12LumaChromaResidualBitstreamMD5 {
		t.Fatalf("High12 luma/chroma residual bitstream md5 = %s, want %s", got, high12LumaChromaResidualBitstreamMD5)
	}
	return data
}

func high12LumaChromaResidualFixturePath(t *testing.T) string {
	t.Helper()
	return filepath.Join("testdata", "h264", "high12_luma_chroma_residual_cavlc_i.h264")
}

func assertHigh12LumaChromaResidualFrames(t *testing.T, frames []*Frame) {
	t.Helper()
	if len(frames) != 1 {
		t.Fatalf("frames = %d, want 1", len(frames))
	}
	frame := frames[0]
	if frame.Width != 48 || frame.Height != 32 ||
		frame.ChromaFormatIDC != 1 || frame.BitDepthLuma != 12 || frame.BitDepthChroma != 12 {
		t.Fatalf("frame format = %dx%d chroma %d depth %d/%d, want 48x32 yuv420p12le",
			frame.Width, frame.Height, frame.ChromaFormatIDC, frame.BitDepthLuma, frame.BitDepthChroma)
	}
	if pixFmt, err := frame.RawPixelFormat(); err != nil || pixFmt != "yuv420p12le" {
		t.Fatalf("RawPixelFormat = %q/%v, want yuv420p12le/nil", pixFmt, err)
	}
	if size, err := frame.RawYUVSize(); err != nil || size != high12LumaChromaResidualRawSize {
		t.Fatalf("RawYUVSize = %d/%v, want %d/nil", size, err, high12LumaChromaResidualRawSize)
	}
	raw, err := frame.AppendRawYUVBytesLE(nil)
	if err != nil {
		t.Fatalf("AppendRawYUVBytesLE: %v", err)
	}
	sum := md5.Sum(raw)
	if got := hex.EncodeToString(sum[:]); got != high12LumaChromaResidualFrameMD5 {
		t.Fatalf("frame raw md5 = %s, want %s", got, high12LumaChromaResidualFrameMD5)
	}
	if _, err := frame.AppendRawYUV(nil); err != ErrUnsupported {
		t.Fatalf("AppendRawYUV high12 error = %v, want ErrUnsupported", err)
	}
}

func assertHigh12LumaChromaResidualFixtureSyntax(t *testing.T, data []byte) {
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
			if sps.ProfileIDC != 244 || sps.Width != 48 || sps.Height != 32 ||
				sps.ChromaFormatIDC != 1 || sps.BitDepthLuma != 12 || sps.BitDepthChroma != 12 {
				t.Fatalf("SPS profile/format = %d %dx%d chroma %d depth %d/%d, want High 4:4:4 Predictive-compatible 48x32 yuv420p12le",
					sps.ProfileIDC, sps.Width, sps.Height, sps.ChromaFormatIDC, sps.BitDepthLuma, sps.BitDepthChroma)
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
				sh.DeblockingFilter != 0 || sh.QScale != 26 {
				t.Fatalf("slice picture/type/deblock/qp = %d/%d/%d/%d, want frame/I/disabled/26",
					sh.PictureStructure, sh.SliceTypeNoS, sh.DeblockingFilter, sh.QScale)
			}
			assertHigh12LumaChromaResidualMacroblockSyntax(t, nal, sh.SPS, sh.PPS)
			gotVCL = append(gotVCL, nal.Type)
		default:
			t.Fatalf("unexpected NAL type %d", nal.Type)
		}
	}
	if len(gotVCL) != 1 || gotVCL[0] != h264.NALIDRSlice {
		t.Fatalf("VCL NALs = %v, want one IDR slice", gotVCL)
	}
}

func assertHigh12LumaChromaResidualMacroblockSyntax(t *testing.T, nal h264.NALUnit, sps *h264.SPS, pps *h264.PPS) {
	t.Helper()
	br := high10ResidualCAVLCBitReader{data: nal.RBSP}
	if firstMB := br.readUE(t); firstMB != 0 {
		t.Fatalf("first_mb_in_slice = %d, want 0", firstMB)
	}
	if sliceType := high10ResidualCAVLCSliceTypeNoS(t, br.readUE(t)); sliceType != h264.PictureTypeI {
		t.Fatalf("slice_type = %d, want I", sliceType)
	}
	if ppsID := br.readUE(t); ppsID != pps.PPSID {
		t.Fatalf("pic_parameter_set_id = %d, want %d", ppsID, pps.PPSID)
	}
	br.readBits(t, int(sps.Log2MaxFrameNum))
	if nal.Type == h264.NALIDRSlice {
		br.readUE(t)
	}
	if sps.PocType == 0 {
		br.readBits(t, int(sps.Log2MaxPocLSB))
	}
	high10ResidualCAVLCReadRefPicMarking(t, &br, nal)
	if delta := br.readSE(t); delta != 0 {
		t.Fatalf("slice_qp_delta = %d, want 0", delta)
	}
	if disableIDC := br.readUE(t); disableIDC != 1 {
		t.Fatalf("disable_deblocking_filter_idc = %d, want 1", disableIDC)
	}

	assertHigh12LumaChromaResidualBits(t, &br, "mb0 intra4x4 no-residual", "11111111111111111100100")
	assertHigh12LumaChromaResidualBits(t, &br, "mb1 intra16x16 no-residual", "00100111")
	assertHigh12LumaChromaResidualBits(t, &br, "mb2 intra16x16 no-residual", "00100111")
	assertHigh12LumaChromaResidualBits(t, &br, "mb3 luma/chroma", "0000110001101010101"+strings.Repeat("1", 15)+"1010101011111111")
	assertHigh12LumaChromaResidualBits(t, &br, "mb4 intra16x16 no-residual", "00100111")
	assertHigh12LumaChromaResidualBits(t, &br, "mb5 intra16x16 no-residual", "00100111")
	assertHigh12LumaChromaResidualRBSPTrailingBits(t, &br)
}

func assertHigh12LumaChromaResidualBits(t *testing.T, br *high10ResidualCAVLCBitReader, label string, bits string) {
	t.Helper()
	for i, wantByte := range bits {
		var want uint32
		switch wantByte {
		case '0':
		case '1':
			want = 1
		default:
			t.Fatalf("%s bitstring[%d] = %q, want 0/1", label, i, wantByte)
		}
		if got := br.readBit(t); got != want {
			t.Fatalf("%s bit[%d] = %d, want %d", label, i, got, want)
		}
	}
}

func assertHigh12LumaChromaResidualRBSPTrailingBits(t *testing.T, br *high10ResidualCAVLCBitReader) {
	t.Helper()
	if stop := br.readBit(t); stop != 1 {
		t.Fatalf("rbsp_stop_one_bit = %d, want 1", stop)
	}
	padBits := (8 - (br.bit & 7)) & 7
	if padBits != 0 {
		if padding := br.readBits(t, padBits); padding != 0 {
			t.Fatalf("rbsp_alignment_zero_bits = %#x over %d bits, want 0", padding, padBits)
		}
	}
	if br.bit != len(br.data)*8 {
		t.Fatalf("RBSP consumed %d bits, want %d", br.bit, len(br.data)*8)
	}
}
