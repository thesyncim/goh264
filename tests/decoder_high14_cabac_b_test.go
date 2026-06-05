// SPDX-License-Identifier: LGPL-2.1-or-later

package goh264_test

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

type high14CABACBCase struct {
	name         string
	sourceFile   string
	idrDeblock   int32
	deblockMode  int32
	direct       int32
	width        int
	height       int
	rawFrameSize int
	bitstreamMD5 string
	frameMD5     []string
	rawVideoMD5  string
}

func TestHigh14CABACBFixtureSyntax(t *testing.T) {
	for _, tt := range high14CABACBCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := high14CABACBFixture(t, tt)
			assertHigh14CABACBFixtureSyntax(t, data, tt)
		})
	}
}

func TestDecodeAnnexBHigh14CABACBFrames(t *testing.T) {
	for _, tt := range high14CABACBCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := high14CABACBFixture(t, tt)
			assertHigh14CABACBFixtureSyntax(t, data, tt)

			frames, err := NewDecoder().DecodeAnnexBFrames(data)
			if err != nil {
				t.Fatalf("DecodeAnnexBFrames: %v", err)
			}
			assertHigh14CABACBFrames(t, frames, tt)
		})
	}
}

func TestDecodeAVCHigh14CABACBFrames(t *testing.T) {
	for _, tt := range high14CABACBCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := high14CABACBFixture(t, tt)
			assertHigh14CABACBFixtureSyntax(t, data, tt)

			for _, nalLengthSize := range []int{2, 3, 4} {
				frames, err := NewDecoder().DecodeAVCFrames(annexBToAVC(t, data, nalLengthSize), nalLengthSize)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: DecodeAVCFrames: %v", nalLengthSize, err)
				}
				assertHigh14CABACBFrames(t, frames, tt)
			}
		})
	}
}

func TestDecodeConfiguredAVCHigh14CABACBFramesAcrossSamplesFlush(t *testing.T) {
	for _, tt := range high14CABACBCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := high14CABACBFixture(t, tt)
			assertHigh14CABACBFixtureSyntax(t, data, tt)

			for _, nalLengthSize := range []int{2, 3, 4} {
				config, samples := annexBToAVCConfigAndSamples(t, data, nalLengthSize)
				if len(samples) != len(tt.frameMD5) {
					t.Fatalf("nalLengthSize=%d: samples = %d, want %d", nalLengthSize, len(samples), len(tt.frameMD5))
				}

				dec := NewDecoder()
				if _, err := dec.ParseAVCDecoderConfigurationRecord(config); err != nil {
					t.Fatalf("nalLengthSize=%d config: %v", nalLengthSize, err)
				}
				var frames []*Frame
				var frameCounts []int
				for i, sample := range samples {
					out, err := dec.DecodeConfiguredAVCFrames(sample)
					if err != nil {
						t.Fatalf("nalLengthSize=%d sample[%d]: DecodeConfiguredAVCFrames: %v", nalLengthSize, i, err)
					}
					frameCounts = append(frameCounts, len(out))
					frames = append(frames, out...)
				}
				out, err := dec.FlushDelayedFrames()
				if err != nil {
					t.Fatalf("nalLengthSize=%d flush: %v", nalLengthSize, err)
				}
				frameCounts = append(frameCounts, len(out))
				assertHigh14CABACBConfiguredSampleCounts(t, tt.name, nalLengthSize, frameCounts)
				frames = append(frames, out...)
				assertHigh14CABACBFrames(t, frames, tt)
			}
		})
	}
}

func TestFFmpegRawVideoFrameMD5OracleHigh14CABACB(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}

	for _, tt := range high14CABACBCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := high14CABACBFixture(t, tt)
			assertHigh14CABACBFixtureSyntax(t, data, tt)
			assertFFmpegHigh14CABACBRawVideoOracle(t, data, tt)
		})
	}
}

func high14CABACBCases() []high14CABACBCase {
	return []high14CABACBCase{
		{
			name:         "nondirect-no-deblock",
			sourceFile:   "high10_nondirect_b_cabac.h264",
			width:        16,
			height:       16,
			rawFrameSize: 768,
			bitstreamMD5: "583e01815d3e3b74383fc6aab33811f4",
			frameMD5: []string{
				"f0f5acdbc113a3796b55c4eea300919e",
				"a271ed93c0644b0b22a957c015c1cb77",
				"188f53e02250df5d76b13f116a646d82",
			},
			rawVideoMD5: "87f8aa6dd2fd909a5c8f162b17bcd667",
		},
		{
			name:         "nondirect-mode1-deblock",
			sourceFile:   "high10_b_deblock_cabac.h264",
			deblockMode:  1,
			width:        16,
			height:       16,
			rawFrameSize: 768,
			bitstreamMD5: "0c8e177aff2d7e39179da6c10380d503",
			frameMD5: []string{
				"f0f5acdbc113a3796b55c4eea300919e",
				"3b0ba0657277bee91a93e1f3831ce586",
				"0d18bafec419f0364e4742c57e261e7e",
			},
			rawVideoMD5: "79d578dca54599b0254cf34c59ab67a7",
		},
		{
			name:         "temporal-direct-mode1-deblock",
			sourceFile:   "high10_direct_b_deblock_temporal_cabac.h264",
			idrDeblock:   1,
			deblockMode:  1,
			width:        32,
			height:       16,
			rawFrameSize: 1536,
			bitstreamMD5: "9e9e99e24e6d9aeb5dc8e7ad3ba8bf70",
			frameMD5: []string{
				"4135c86c2c49ff515cd47c8fda7bd346",
				"8c071eddc38468b305f03a50124a669b",
				"1e8f8515a4867af7b44578c01fa04914",
			},
			rawVideoMD5: "320b1242759143fe62c86e5bdb48f949",
		},
	}
}

func high14CABACBFixture(t *testing.T, tt high14CABACBCase) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "h264", tt.sourceFile))
	if err != nil {
		t.Fatalf("read %s: %v", tt.sourceFile, err)
	}
	out := high14CABACBRewriteAnnexB(t, data)
	sum := md5.Sum(out)
	if got := hex.EncodeToString(sum[:]); got != tt.bitstreamMD5 {
		t.Fatalf("High14 CABAC B generated bitstream md5 = %s, want %s", got, tt.bitstreamMD5)
	}
	return out
}

func high14CABACBRewriteAnnexB(t *testing.T, data []byte) []byte {
	t.Helper()
	start, prefixLen, ok := high14CABACBFindStartCode(data, 0)
	if !ok {
		t.Fatal("source fixture has no Annex B start code")
	}
	var out []byte
	for ok {
		nalStart := start + prefixLen
		nextStart, nextPrefixLen, nextOK := high14CABACBFindStartCode(data, nalStart)
		nalEnd := len(data)
		if nextOK {
			nalEnd = nextStart
		}
		if nalEnd > nalStart {
			out = append(out, data[start:nalStart]...)
			raw := append([]byte(nil), data[nalStart:nalEnd]...)
			if raw[0]&0x1f == byte(h264.NALSPS) {
				raw = high14CABACBRewriteSPSRaw(t, raw)
			}
			out = append(out, raw...)
		}
		if !nextOK {
			break
		}
		start, prefixLen, ok = nextStart, nextPrefixLen, true
	}
	return out
}

func high14CABACBFindStartCode(data []byte, off int) (int, int, bool) {
	for i := off; i+3 <= len(data); i++ {
		if i+4 <= len(data) && bytes.Equal(data[i:i+4], []byte{0, 0, 0, 1}) {
			return i, 4, true
		}
		if bytes.Equal(data[i:i+3], []byte{0, 0, 1}) {
			return i, 3, true
		}
	}
	return 0, 0, false
}

func high14CABACBRewriteSPSRaw(t *testing.T, raw []byte) []byte {
	t.Helper()
	rbsp := high14CABACBEBSPToRBSP(raw[1:])
	bits := high14CABACBBits(rbsp)
	stop := strings.LastIndexByte(bits, '1')
	if stop < 0 {
		t.Fatal("SPS has no rbsp stop bit")
	}
	syntax := bits[:stop]
	pos := 0
	_, profileBits := high14CABACBReadFixedBits(t, syntax, &pos, 8)
	constraints, constraintsBits := high14CABACBReadFixedBits(t, syntax, &pos, 8)
	level, levelBits := high14CABACBReadFixedBits(t, syntax, &pos, 8)
	_, spsIDBits := high14CABACBReadUEBits(t, syntax, &pos)
	chroma, chromaBits := high14CABACBReadUEBits(t, syntax, &pos)
	bitDepthLumaMinus8, _ := high14CABACBReadUEBits(t, syntax, &pos)
	bitDepthChromaMinus8, _ := high14CABACBReadUEBits(t, syntax, &pos)
	if profileBits != "01101110" || chroma != 1 || bitDepthLumaMinus8 != 2 || bitDepthChromaMinus8 != 2 {
		t.Fatalf("source SPS profile/chroma/depth-minus8 = %s/%d/%d/%d, want High10 4:2:0 2/2",
			profileBits, chroma, bitDepthLumaMinus8, bitDepthChromaMinus8)
	}

	prefix := fmt.Sprintf("%08b%08b%08b", 244, constraints, level)
	payload := prefix + spsIDBits + chromaBits + high14CABACBUEBits(6) + high14CABACBUEBits(6) + syntax[pos:]
	rbsp = high14CABACBPackRBSP(payload)
	_ = constraintsBits
	_ = levelBits
	return append([]byte{raw[0]}, high14CABACBRBSPToEBSP(rbsp)...)
}

func high14CABACBEBSPToRBSP(data []byte) []byte {
	out := make([]byte, 0, len(data))
	zeros := 0
	for _, b := range data {
		if zeros >= 2 && b == 3 {
			zeros = 0
			continue
		}
		out = append(out, b)
		if b == 0 {
			zeros++
		} else {
			zeros = 0
		}
	}
	return out
}

func high14CABACBRBSPToEBSP(data []byte) []byte {
	out := make([]byte, 0, len(data))
	zeros := 0
	for _, b := range data {
		if zeros >= 2 && b <= 3 {
			out = append(out, 3)
			zeros = 0
		}
		out = append(out, b)
		if b == 0 {
			zeros++
		} else {
			zeros = 0
		}
	}
	return out
}

func high14CABACBBits(data []byte) string {
	var b strings.Builder
	b.Grow(len(data) * 8)
	for _, v := range data {
		fmt.Fprintf(&b, "%08b", v)
	}
	return b.String()
}

func high14CABACBReadFixedBits(t *testing.T, bits string, pos *int, n int) (uint32, string) {
	t.Helper()
	if *pos+n > len(bits) {
		t.Fatalf("fixed bits overread pos=%d n=%d len=%d", *pos, n, len(bits))
	}
	raw := bits[*pos : *pos+n]
	*pos += n
	var v uint32
	for _, ch := range raw {
		v <<= 1
		if ch == '1' {
			v |= 1
		}
	}
	return v, raw
}

func high14CABACBReadUEBits(t *testing.T, bits string, pos *int) (uint32, string) {
	t.Helper()
	start := *pos
	zeros := 0
	for *pos+zeros < len(bits) && bits[*pos+zeros] == '0' {
		zeros++
	}
	if *pos+zeros >= len(bits) {
		t.Fatalf("ue overread pos=%d len=%d", *pos, len(bits))
	}
	*pos += zeros + 1
	var suffix uint32
	for i := 0; i < zeros; i++ {
		suffix <<= 1
		if bits[*pos+i] == '1' {
			suffix |= 1
		}
	}
	*pos += zeros
	return (1<<uint(zeros) - 1) + suffix, bits[start:*pos]
}

func high14CABACBUEBits(v uint32) string {
	codeNum := v + 1
	width := 0
	for tmp := codeNum; tmp > 1; tmp >>= 1 {
		width++
	}
	return strings.Repeat("0", width) + fmt.Sprintf("%0*b", width+1, codeNum)
}

func high14CABACBPackRBSP(payload string) []byte {
	bits := payload + "1"
	if mod := len(bits) % 8; mod != 0 {
		bits += strings.Repeat("0", 8-mod)
	}
	out := make([]byte, len(bits)/8)
	for i := range out {
		var v byte
		for _, ch := range bits[i*8 : (i+1)*8] {
			v <<= 1
			if ch == '1' {
				v |= 1
			}
		}
		out[i] = v
	}
	return out
}

func assertHigh14CABACBFixtureSyntax(t *testing.T, data []byte, tt high14CABACBCase) {
	t.Helper()
	nals, err := h264.SplitAnnexB(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(nals) != 5 {
		t.Fatalf("NAL count = %d, want SPS/PPS/IDR/P/B", len(nals))
	}

	var spsList [32]*h264.SPS
	var ppsList [256]*h264.PPS
	var gotSlices []int32
	for i, nal := range nals {
		switch nal.Type {
		case h264.NALSPS:
			sps, err := h264.DecodeSPS(nal.RBSP)
			if err != nil {
				t.Fatal(err)
			}
			if i != 0 || sps.ProfileIDC != 244 || sps.Width != int32(tt.width) || sps.Height != int32(tt.height) ||
				sps.ChromaFormatIDC != 1 || sps.BitDepthLuma != 14 || sps.BitDepthChroma != 14 ||
				sps.FrameMBSOnlyFlag != 1 || sps.MBAFF != 0 || sps.RefFrameCount != 2 {
				t.Fatalf("SPS = nal[%d] profile %d %dx%d chroma %d depth %d/%d frameonly/mbaff %d/%d refs %d, want High14 4:2:0 frame-only refs=2",
					i, sps.ProfileIDC, sps.Width, sps.Height, sps.ChromaFormatIDC, sps.BitDepthLuma, sps.BitDepthChroma,
					sps.FrameMBSOnlyFlag, sps.MBAFF, sps.RefFrameCount)
			}
			spsList[sps.SPSID] = sps
		case h264.NALPPS:
			pps, err := h264.DecodePPS(nal.RBSP, &spsList)
			if err != nil {
				t.Fatal(err)
			}
			if i != 1 || pps.CABAC != 1 || pps.Transform8x8Mode != 0 || pps.WeightedPred != 0 ||
				pps.WeightedBipredIDC != 0 || pps.RefCount != [2]uint32{2, 1} {
				t.Fatalf("PPS = nal[%d] cabac/8x8/weights/refs = %d/%d/%d/%d/%v, want CABAC/no-8x8/unweighted refs=2/1",
					i, pps.CABAC, pps.Transform8x8Mode, pps.WeightedPred, pps.WeightedBipredIDC, pps.RefCount)
			}
			ppsList[pps.PPSID] = pps
		case h264.NALIDRSlice, h264.NALSlice:
			sh, err := h264.ParseSliceHeader(nal, &ppsList)
			if err != nil {
				t.Fatal(err)
			}
			if sh.PictureStructure != h264.PictureFrame {
				t.Fatalf("slice picture = %d, want frame", sh.PictureStructure)
			}
			switch sh.SliceTypeNoS {
			case h264.PictureTypeI:
				if sh.DeblockingFilter != tt.idrDeblock {
					t.Fatalf("I slice deblock = %d, want mode-%d", sh.DeblockingFilter, tt.idrDeblock)
				}
				if sh.ListCount != 0 || sh.RefCount != ([2]uint32{}) {
					t.Fatalf("I slice lists/refs = %d/%v, want none", sh.ListCount, sh.RefCount)
				}
			case h264.PictureTypeP:
				if sh.DeblockingFilter != tt.deblockMode {
					t.Fatalf("P slice deblock = %d, want mode-%d", sh.DeblockingFilter, tt.deblockMode)
				}
				if sh.ListCount != 1 || sh.RefCount[0] != 1 ||
					sh.PredWeightTable.UseWeight != 0 || sh.PredWeightTable.UseWeightChroma != 0 {
					t.Fatalf("P slice lists/refs/weights = %d/%v/%d/%d, want L0 refs=1 unweighted",
						sh.ListCount, sh.RefCount, sh.PredWeightTable.UseWeight, sh.PredWeightTable.UseWeightChroma)
				}
			case h264.PictureTypeB:
				if sh.DeblockingFilter != tt.deblockMode {
					t.Fatalf("B slice deblock = %d, want mode-%d", sh.DeblockingFilter, tt.deblockMode)
				}
				if sh.ListCount != 2 || sh.RefCount[0] != 1 || sh.RefCount[1] != 1 ||
					sh.DirectSpatialMVPred != tt.direct ||
					sh.PredWeightTable.UseWeight != 0 || sh.PredWeightTable.UseWeightChroma != 0 {
					t.Fatalf("B slice lists/refs/direct/weights = %d/%v/%d/%d/%d, want L0/L1 refs=1/1 direct=%d unweighted",
						sh.ListCount, sh.RefCount, sh.DirectSpatialMVPred, sh.PredWeightTable.UseWeight, sh.PredWeightTable.UseWeightChroma, tt.direct)
				}
			default:
				t.Fatalf("unexpected slice type %d", sh.SliceTypeNoS)
			}
			gotSlices = append(gotSlices, sh.SliceTypeNoS)
		default:
			t.Fatalf("unexpected NAL type %d", nal.Type)
		}
	}
	if len(gotSlices) != 3 || gotSlices[0] != h264.PictureTypeI || gotSlices[1] != h264.PictureTypeP || gotSlices[2] != h264.PictureTypeB {
		t.Fatalf("slice types = %v, want I/P/B", gotSlices)
	}
}

func assertHigh14CABACBConfiguredSampleCounts(t *testing.T, name string, nalLengthSize int, got []int) {
	t.Helper()
	want := []int{0, 1, 1, 1}
	if len(got) != len(want) {
		t.Fatalf("%s nalLengthSize=%d configured sample/flush counts = %v, want %v", name, nalLengthSize, got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%s nalLengthSize=%d configured sample/flush counts = %v, want %v", name, nalLengthSize, got, want)
		}
	}
}

func assertHigh14CABACBFrames(t *testing.T, frames []*Frame, tt high14CABACBCase) {
	t.Helper()
	if len(frames) != len(tt.frameMD5) {
		t.Fatalf("frames = %d, want %d", len(frames), len(tt.frameMD5))
	}
	var rawVideo []byte
	for i, frame := range frames {
		if frame == nil {
			t.Fatalf("frame[%d] is nil", i)
		}
		if frame.Width != tt.width || frame.Height != tt.height ||
			frame.ChromaFormatIDC != 1 || frame.BitDepthLuma != 14 || frame.BitDepthChroma != 14 {
			t.Fatalf("frame[%d] format = %dx%d chroma %d depth %d/%d, want %dx%d yuv420p14le",
				i, frame.Width, frame.Height, frame.ChromaFormatIDC, frame.BitDepthLuma, frame.BitDepthChroma, tt.width, tt.height)
		}
		if pixFmt, err := frame.RawPixelFormat(); err != nil || pixFmt != "yuv420p14le" {
			t.Fatalf("frame[%d] RawPixelFormat = %q/%v, want yuv420p14le/nil", i, pixFmt, err)
		}
		if size, err := frame.RawYUVSize(); err != nil || size != tt.rawFrameSize {
			t.Fatalf("frame[%d] RawYUVSize = %d/%v, want %d/nil", i, size, err, tt.rawFrameSize)
		}
		raw, err := frame.AppendRawYUVBytesLE(nil)
		if err != nil {
			t.Fatalf("frame[%d] AppendRawYUVBytesLE: %v", i, err)
		}
		rawVideo = append(rawVideo, raw...)
		sum := md5.Sum(raw)
		if got := hex.EncodeToString(sum[:]); got != tt.frameMD5[i] {
			t.Fatalf("frame[%d] raw md5 = %s, want %s", i, got, tt.frameMD5[i])
		}
		if _, err := frame.AppendRawYUV(nil); err != ErrUnsupported {
			t.Fatalf("frame[%d] AppendRawYUV high14 error = %v, want ErrUnsupported", i, err)
		}
	}
	sum := md5.Sum(rawVideo)
	if got := hex.EncodeToString(sum[:]); got != tt.rawVideoMD5 {
		t.Fatalf("rawvideo md5 = %s, want %s", got, tt.rawVideoMD5)
	}
}

func assertFFmpegHigh14CABACBRawVideoOracle(t *testing.T, data []byte, tt high14CABACBCase) {
	t.Helper()
	path := writeTempH264(t, data)
	framemd5 := exec.Command("ffmpeg",
		"-v", "error",
		"-xerror",
		"-f", "h264",
		"-i", path,
		"-an", "-sn", "-dn",
		"-pix_fmt", "yuv420p14le",
		"-f", "framemd5",
		"-",
	)
	framemd5Out, err := framemd5.Output()
	if err != nil {
		t.Fatalf("ffmpeg framemd5: %v", err)
	}
	for i, want := range tt.frameMD5 {
		line := []byte(fmt.Sprintf("0, %10d, %10d,        1, %8d, %s", i, i, tt.rawFrameSize, want))
		if !bytes.Contains(framemd5Out, line) {
			t.Fatalf("frame[%d] missing %q in framemd5:\n%s", i, line, framemd5Out)
		}
	}

	rawvideo := exec.Command("ffmpeg",
		"-v", "error",
		"-xerror",
		"-f", "h264",
		"-i", path,
		"-an", "-sn", "-dn",
		"-pix_fmt", "yuv420p14le",
		"-f", "rawvideo",
		"-",
	)
	raw, err := rawvideo.Output()
	if err != nil {
		t.Fatalf("ffmpeg rawvideo: %v", err)
	}
	if len(raw) != len(tt.frameMD5)*tt.rawFrameSize {
		t.Fatalf("rawvideo size = %d, want %d", len(raw), len(tt.frameMD5)*tt.rawFrameSize)
	}
	sum := md5.Sum(raw)
	if got := hex.EncodeToString(sum[:]); got != tt.rawVideoMD5 {
		t.Fatalf("rawvideo md5 = %s, want %s", got, tt.rawVideoMD5)
	}
}
