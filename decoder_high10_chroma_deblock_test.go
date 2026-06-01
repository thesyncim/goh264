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
	"testing"

	"github.com/thesyncim/goh264/internal/h264"
)

type high10ChromaDeblockFixture struct {
	name         string
	file         string
	profileIDC   int32
	chromaFormat uint32
	pixFmt       string
	cabac        int32
	frameSize    int
	bitstreamMD5 string
	rawVideoMD5  string
	frameMD5     []string
}

func TestDecodeAnnexBHigh10ChromaDeblockFrames(t *testing.T) {
	for _, tt := range high10ChromaDeblockFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := readHigh10ChromaDeblockFixture(t, tt)
			assertHigh10ChromaDeblockFixtureSyntax(t, data, tt)

			frames, err := NewDecoder().DecodeAnnexBFrames(data)
			if err != nil {
				t.Fatalf("decode High10 chroma deblock fixture: %v", err)
			}
			assertHigh10ChromaDeblockFrames(t, frames, tt)
		})
	}
}

func TestDecodeAVCHigh10ChromaDeblockFrames(t *testing.T) {
	for _, tt := range high10ChromaDeblockFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := readHigh10ChromaDeblockFixture(t, tt)
			for _, nalLengthSize := range []int{2, 3, 4} {
				frames, err := NewDecoder().DecodeAVCFrames(annexBToAVC(t, data, nalLengthSize), nalLengthSize)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: %v", nalLengthSize, err)
				}
				assertHigh10ChromaDeblockFrames(t, frames, tt)
			}
		})
	}
}

func TestDecodeConfiguredAVCHigh10ChromaDeblockFrames(t *testing.T) {
	for _, tt := range high10ChromaDeblockFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := readHigh10ChromaDeblockFixture(t, tt)
			config, packet := annexBToAVCConfigAndPacket(t, data, 4)
			frames, err := NewDecoder().DecodeAVCFramesWithConfigurationRecord(config, packet)
			if err != nil {
				t.Fatal(err)
			}
			assertHigh10ChromaDeblockFrames(t, frames, tt)
		})
	}
}

func TestDecodeConfiguredAVCSamplesHigh10ChromaDeblockFrames(t *testing.T) {
	for _, tt := range high10ChromaDeblockFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := readHigh10ChromaDeblockFixture(t, tt)
			config, samples := annexBToAVCConfigAndSamples(t, data, 4)
			if len(samples) != len(tt.frameMD5) {
				t.Fatalf("samples = %d, want %d", len(samples), len(tt.frameMD5))
			}
			dec := NewDecoder()
			if _, err := dec.ParseAVCDecoderConfigurationRecord(config); err != nil {
				t.Fatal(err)
			}
			var frames []*Frame
			for i, sample := range samples {
				frame, err := dec.DecodeConfiguredAVC(sample)
				if err != nil {
					t.Fatalf("sample[%d] decode High10 chroma deblock fixture: %v", i, err)
				}
				frames = append(frames, frame)
			}
			assertHigh10ChromaDeblockFrames(t, frames, tt)
		})
	}
}

func TestFFmpegRawVideoFrameMD5OracleHigh10ChromaDeblock(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}

	for _, tt := range high10ChromaDeblockFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := readHigh10ChromaDeblockFixture(t, tt)
			assertHigh10ChromaDeblockFixtureSyntax(t, data, tt)
			assertFFmpegHigh10ChromaDeblockRawVideoOracle(t, data, tt)
		})
	}
}

func high10ChromaDeblockFixtures() []high10ChromaDeblockFixture {
	return []high10ChromaDeblockFixture{
		{
			name:         "422-cavlc",
			file:         "high10_deblock422_cavlc_idrp.h264",
			profileIDC:   122,
			chromaFormat: 2,
			pixFmt:       "yuv422p10le",
			cabac:        0,
			frameSize:    4096,
			bitstreamMD5: "095b3897df89b12b6fba734931771d8b",
			rawVideoMD5:  "710f36ec1dd547e5b584144bb299ee7a",
			frameMD5: []string{
				"754ac4c117c705808e87230f2d39a521",
				"accfc50bf3e08afaf0e073d0849992dc",
			},
		},
		{
			name:         "422-cabac",
			file:         "high10_deblock422_cabac_idrp.h264",
			profileIDC:   122,
			chromaFormat: 2,
			pixFmt:       "yuv422p10le",
			cabac:        1,
			frameSize:    4096,
			bitstreamMD5: "a697f204f63ac7d5d5eab7df23c16755",
			rawVideoMD5:  "1a011c767ac1131c7eb4b07c32f8a1ab",
			frameMD5: []string{
				"77bd0e8f2c734a359d2238bbeffab77b",
				"b5fd410a1bb665f5c10f8268fbfd2d53",
			},
		},
		{
			name:         "444-cavlc",
			file:         "high10_deblock444_cavlc_idrp.h264",
			profileIDC:   244,
			chromaFormat: 3,
			pixFmt:       "yuv444p10le",
			cabac:        0,
			frameSize:    6144,
			bitstreamMD5: "91ac19688e8e9fa26ad3941954b7948f",
			rawVideoMD5:  "6cd1945a6daefd4ab1bc257f6be1d906",
			frameMD5: []string{
				"b456b84535b2b0241a9ad973edaccd25",
				"b0b7fc22ee4cb292a902d4949365c040",
			},
		},
		{
			name:         "444-cabac",
			file:         "high10_deblock444_cabac_idrp.h264",
			profileIDC:   244,
			chromaFormat: 3,
			pixFmt:       "yuv444p10le",
			cabac:        1,
			frameSize:    6144,
			bitstreamMD5: "f3ed8d65e4a600c331770ec9acb4d8f6",
			rawVideoMD5:  "1f70a47728f816c0406fd7aed90bcbb2",
			frameMD5: []string{
				"e0e3b6a956484218ee7c5979780ed9d6",
				"b169bd10fc31bb91aa50a040b1358838",
			},
		},
	}
}

func readHigh10ChromaDeblockFixture(t *testing.T, fixture high10ChromaDeblockFixture) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "h264", fixture.file))
	if err != nil {
		t.Fatal(err)
	}
	sum := md5.Sum(data)
	if got := hex.EncodeToString(sum[:]); got != fixture.bitstreamMD5 {
		t.Fatalf("%s bitstream md5 = %s, want %s", fixture.file, got, fixture.bitstreamMD5)
	}
	return data
}

func assertHigh10ChromaDeblockFrames(t *testing.T, frames []*Frame, fixture high10ChromaDeblockFixture) {
	t.Helper()
	if len(frames) != len(fixture.frameMD5) {
		t.Fatalf("frames = %d, want %d", len(frames), len(fixture.frameMD5))
	}
	rawVideo := make([]byte, 0, len(frames)*fixture.frameSize)
	for i, frame := range frames {
		if frame == nil {
			t.Fatalf("frame[%d] is nil", i)
		}
		if frame.BitDepthLuma != 10 || frame.BitDepthChroma != 10 || frame.ChromaFormatIDC != fixture.chromaFormat {
			t.Fatalf("frame[%d] format depth/chroma = %d/%d/%d, want 10/10/%d",
				i, frame.BitDepthLuma, frame.BitDepthChroma, frame.ChromaFormatIDC, fixture.chromaFormat)
		}
		if got, err := frame.RawPixelFormat(); err != nil || got != fixture.pixFmt {
			t.Fatalf("frame[%d] RawPixelFormat = %q/%v, want %s/nil", i, got, err, fixture.pixFmt)
		}
		if len(frame.Y) != 0 || len(frame.Cb) != 0 || len(frame.Cr) != 0 {
			t.Fatalf("frame[%d] populated 8-bit planes", i)
		}
		if len(frame.Y16) == 0 || len(frame.Cb16) == 0 || len(frame.Cr16) == 0 {
			t.Fatalf("frame[%d] missing high planes", i)
		}
		if _, err := frame.AppendRawYUV(nil); err != ErrUnsupported {
			t.Fatalf("frame[%d] AppendRawYUV error = %v, want ErrUnsupported", i, err)
		}
		raw, err := frame.AppendRawYUVBytesLE(nil)
		if err != nil {
			t.Fatalf("frame[%d] AppendRawYUVBytesLE: %v", i, err)
		}
		if len(raw) != fixture.frameSize {
			t.Fatalf("frame[%d] raw len = %d, want %d", i, len(raw), fixture.frameSize)
		}
		sum := md5.Sum(raw)
		if got := hex.EncodeToString(sum[:]); got != fixture.frameMD5[i] {
			t.Fatalf("frame[%d] md5 = %s, want %s", i, got, fixture.frameMD5[i])
		}
		rawVideo = append(rawVideo, raw...)
	}
	sum := md5.Sum(rawVideo)
	if got := hex.EncodeToString(sum[:]); got != fixture.rawVideoMD5 {
		t.Fatalf("rawvideo md5 = %s, want %s", got, fixture.rawVideoMD5)
	}
}

func assertHigh10ChromaDeblockFixtureSyntax(t *testing.T, data []byte, fixture high10ChromaDeblockFixture) {
	t.Helper()
	nals, err := h264.SplitAnnexB(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(nals) != 4 {
		t.Fatalf("NAL count = %d, want 4 stripped SPS/PPS/IDR/P", len(nals))
	}

	var spsList [32]*h264.SPS
	var ppsList [256]*h264.PPS
	var gotVCL []h264.NALUnitType
	var gotSlices []int32
	for _, nal := range nals {
		switch nal.Type {
		case h264.NALSPS:
			sps, err := h264.DecodeSPS(nal.RBSP)
			if err != nil {
				t.Fatal(err)
			}
			if sps.ProfileIDC != fixture.profileIDC || sps.Width != 32 || sps.Height != 32 ||
				sps.ChromaFormatIDC != fixture.chromaFormat || sps.BitDepthLuma != 10 || sps.BitDepthChroma != 10 {
				t.Fatalf("SPS profile/format = %d %dx%d chroma %d depth %d/%d, want profile %d 32x32 chroma %d 10/10",
					sps.ProfileIDC, sps.Width, sps.Height, sps.ChromaFormatIDC, sps.BitDepthLuma, sps.BitDepthChroma,
					fixture.profileIDC, fixture.chromaFormat)
			}
			if sps.FrameMBSOnlyFlag != 1 || sps.MBAFF != 0 {
				t.Fatalf("SPS frame-only flags = frame_mbs_only:%d mbaff:%d, want 1/0", sps.FrameMBSOnlyFlag, sps.MBAFF)
			}
			spsList[sps.SPSID] = sps
		case h264.NALPPS:
			pps, err := h264.DecodePPS(nal.RBSP, &spsList)
			if err != nil {
				t.Fatal(err)
			}
			if pps.CABAC != fixture.cabac || pps.Transform8x8Mode != 0 {
				t.Fatalf("PPS cabac/8x8dct = %d/%d, want %d/0", pps.CABAC, pps.Transform8x8Mode, fixture.cabac)
			}
			if pps.RefCount != [2]uint32{1, 1} || pps.WeightedPred != 0 || pps.WeightedBipredIDC != 0 {
				t.Fatalf("PPS refs/weight = %v/%d/%d, want ref=1 and unweighted", pps.RefCount, pps.WeightedPred, pps.WeightedBipredIDC)
			}
			ppsList[pps.PPSID] = pps
		case h264.NALIDRSlice, h264.NALSlice:
			sh, err := h264.ParseSliceHeader(nal, &ppsList)
			if err != nil {
				t.Fatal(err)
			}
			if sh.PictureStructure != h264.PictureFrame || sh.DeblockingFilter != 1 {
				t.Fatalf("slice picture/deblock = %d/%d, want frame/enabled", sh.PictureStructure, sh.DeblockingFilter)
			}
			if sh.SliceAlphaC0Offset != 0 || sh.SliceBetaOffset != 0 {
				t.Fatalf("slice deblock offsets = %d/%d, want 0/0", sh.SliceAlphaC0Offset, sh.SliceBetaOffset)
			}
			if sh.SliceTypeNoS == h264.PictureTypeP {
				if sh.ListCount != 1 || sh.RefCount[0] != 1 || sh.PredWeightTable.UseWeight != 0 || sh.PredWeightTable.UseWeightChroma != 0 {
					t.Fatalf("P slice lists/ref0/weights = %d/%d/%d/%d, want one L0 ref and unweighted",
						sh.ListCount, sh.RefCount[0], sh.PredWeightTable.UseWeight, sh.PredWeightTable.UseWeightChroma)
				}
			}
			gotVCL = append(gotVCL, nal.Type)
			gotSlices = append(gotSlices, sh.SliceTypeNoS)
		default:
			t.Fatalf("unexpected NAL type %d", nal.Type)
		}
	}
	if len(gotVCL) != 2 || gotVCL[0] != h264.NALIDRSlice || gotVCL[1] != h264.NALSlice {
		t.Fatalf("VCL NALs = %v, want IDR then non-IDR", gotVCL)
	}
	if gotSlices[0] != h264.PictureTypeI || gotSlices[1] != h264.PictureTypeP {
		t.Fatalf("slice types = %v, want I then P", gotSlices)
	}
}

func assertFFmpegHigh10ChromaDeblockRawVideoOracle(t *testing.T, data []byte, fixture high10ChromaDeblockFixture) {
	t.Helper()
	path := writeTempH264(t, data)
	framemd5 := exec.Command("ffmpeg",
		"-v", "error",
		"-f", "h264",
		"-i", path,
		"-an", "-sn", "-dn",
		"-pix_fmt", fixture.pixFmt,
		"-f", "framemd5",
		"-",
	)
	framemd5Out, err := framemd5.Output()
	if err != nil {
		t.Fatalf("ffmpeg framemd5: %v", err)
	}
	for i, want := range fixture.frameMD5 {
		line := []byte(fmt.Sprintf("0, %10d, %10d,        1, %8d, %s", i, i, fixture.frameSize, want))
		if !bytes.Contains(framemd5Out, line) {
			t.Fatalf("frame[%d] missing %q in framemd5:\n%s", i, line, framemd5Out)
		}
	}

	rawvideo := exec.Command("ffmpeg",
		"-v", "error",
		"-f", "h264",
		"-i", path,
		"-an", "-sn", "-dn",
		"-pix_fmt", fixture.pixFmt,
		"-f", "rawvideo",
		"-",
	)
	raw, err := rawvideo.Output()
	if err != nil {
		t.Fatalf("ffmpeg rawvideo: %v", err)
	}
	if len(raw) != len(fixture.frameMD5)*fixture.frameSize {
		t.Fatalf("rawvideo size = %d, want %d", len(raw), len(fixture.frameMD5)*fixture.frameSize)
	}
	sum := md5.Sum(raw)
	if got := hex.EncodeToString(sum[:]); got != fixture.rawVideoMD5 {
		t.Fatalf("rawvideo md5 = %s, want %s", got, fixture.rawVideoMD5)
	}
}
