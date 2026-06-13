// SPDX-License-Identifier: LGPL-2.1-or-later

package goh264_test

import (
	"crypto/md5"
	"encoding/hex"
	"os"
	"os/exec"
	"testing"

	"github.com/thesyncim/goh264/internal/h264"
)

type high12ChromaDeblockCase struct {
	name           string
	chromaFormat   uint32
	disableDeblock uint32
	deblockMode    int32
	pixFmt         string
	frameSize      int
	bitstreamMD5   string
	rawVideoMD5    string
	frameMD5       []string
}

func TestHigh12ChromaDeblockFixtureSyntax(t *testing.T) {
	for _, tt := range high12ChromaDeblockCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := high12ChromaDeblockFixture(t, tt)
			assertHigh12ChromaDeblockFixtureSyntax(t, data, tt)
		})
	}
}

func TestDecodeAnnexBHigh12ChromaDeblockFrames(t *testing.T) {
	for _, tt := range high12ChromaDeblockCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := high12ChromaDeblockFixture(t, tt)
			assertHigh12ChromaDeblockFixtureSyntax(t, data, tt)
			frames, err := NewDecoder().DecodeAnnexBFrames(data)
			if err != nil {
				t.Fatalf("DecodeAnnexBFrames: %v", err)
			}
			assertHigh12ChromaDeblockFrames(t, frames, tt)
		})
	}
}

func TestDecodeAVCHigh12ChromaDeblockFrames(t *testing.T) {
	for _, tt := range high12ChromaDeblockCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := high12ChromaDeblockFixture(t, tt)
			assertHigh12ChromaDeblockFixtureSyntax(t, data, tt)
			for _, nalLengthSize := range []int{2, 3, 4} {
				frames, err := NewDecoder().DecodeAVCFrames(annexBToAVC(t, data, nalLengthSize), nalLengthSize)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: DecodeAVCFrames: %v", nalLengthSize, err)
				}
				assertHigh12ChromaDeblockFrames(t, frames, tt)
			}
		})
	}
}

func TestDecodeConfiguredAVCHigh12ChromaDeblockFramesAcrossSamples(t *testing.T) {
	for _, tt := range high12ChromaDeblockCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := high12ChromaDeblockFixture(t, tt)
			assertHigh12ChromaDeblockFixtureSyntax(t, data, tt)
			for _, nalLengthSize := range []int{2, 3, 4} {
				config, samples := annexBToAVCConfigAndSamples(t, data, nalLengthSize)
				if len(samples) != len(tt.frameMD5) {
					t.Fatalf("nalLengthSize=%d samples = %d, want %d", nalLengthSize, len(samples), len(tt.frameMD5))
				}
				dec := NewDecoder()
				if _, err := dec.ConfigureAVCC(config); err != nil {
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
				assertHigh12ChromaDeblockFrames(t, frames, tt)
			}
		})
	}
}

func TestFFmpegRawVideoFrameMD5OracleHigh12ChromaDeblock(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}
	for _, tt := range high12ChromaDeblockCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := high12ChromaDeblockFixture(t, tt)
			assertHigh12ChromaDeblockFixtureSyntax(t, data, tt)
			assertFFmpegHigh12ChromaDeblockRawVideoOracle(t, data, tt)
		})
	}
}

func high12ChromaDeblockCases() []high12ChromaDeblockCase {
	return []high12ChromaDeblockCase{
		{
			name:           "422-no-deblock",
			chromaFormat:   2,
			disableDeblock: 1,
			deblockMode:    0,
			pixFmt:         "yuv422p12le",
			frameSize:      1024,
			bitstreamMD5:   "c9a49deb102fc9ec3e161024fb6880c6",
			rawVideoMD5:    "a547bc3fb7b06910bf3973122a426661",
			frameMD5:       []string{"0556a969d4e9ee0393d6007103ccfdae", "0556a969d4e9ee0393d6007103ccfdae"},
		},
		{
			name:           "422-frame-deblock",
			chromaFormat:   2,
			disableDeblock: 0,
			deblockMode:    1,
			pixFmt:         "yuv422p12le",
			frameSize:      1024,
			bitstreamMD5:   "ad552666c3d5f07d985dc54fbb3974dd",
			rawVideoMD5:    "a547bc3fb7b06910bf3973122a426661",
			frameMD5:       []string{"0556a969d4e9ee0393d6007103ccfdae", "0556a969d4e9ee0393d6007103ccfdae"},
		},
		{
			name:           "422-slice-boundary",
			chromaFormat:   2,
			disableDeblock: 2,
			deblockMode:    2,
			pixFmt:         "yuv422p12le",
			frameSize:      1024,
			bitstreamMD5:   "746fa9f0ee8d8c6ab32ef9f0bb320126",
			rawVideoMD5:    "a547bc3fb7b06910bf3973122a426661",
			frameMD5:       []string{"0556a969d4e9ee0393d6007103ccfdae", "0556a969d4e9ee0393d6007103ccfdae"},
		},
		{
			name:           "444-no-deblock",
			chromaFormat:   3,
			disableDeblock: 1,
			deblockMode:    0,
			pixFmt:         "yuv444p12le",
			frameSize:      1536,
			bitstreamMD5:   "c1b71fa7ec70c646748eb11a6fdafe42",
			rawVideoMD5:    "d1829fcdc48805bafcbef3f908d5be45",
			frameMD5:       []string{"d4fbc620f3b054967df19b16392bdc6e", "d4fbc620f3b054967df19b16392bdc6e"},
		},
		{
			name:           "444-frame-deblock",
			chromaFormat:   3,
			disableDeblock: 0,
			deblockMode:    1,
			pixFmt:         "yuv444p12le",
			frameSize:      1536,
			bitstreamMD5:   "9e05b37a68f8e20d2ec8ead0f9e909ef",
			rawVideoMD5:    "d1829fcdc48805bafcbef3f908d5be45",
			frameMD5:       []string{"d4fbc620f3b054967df19b16392bdc6e", "d4fbc620f3b054967df19b16392bdc6e"},
		},
		{
			name:           "444-slice-boundary",
			chromaFormat:   3,
			disableDeblock: 2,
			deblockMode:    2,
			pixFmt:         "yuv444p12le",
			frameSize:      1536,
			bitstreamMD5:   "5d3fc779dc15b59832168e2c238ecb46",
			rawVideoMD5:    "d1829fcdc48805bafcbef3f908d5be45",
			frameMD5:       []string{"d4fbc620f3b054967df19b16392bdc6e", "d4fbc620f3b054967df19b16392bdc6e"},
		},
	}
}

func high12ChromaDeblockFixture(t *testing.T, tt high12ChromaDeblockCase) []byte {
	t.Helper()
	var data []byte
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALSPS), high12ChromaDeblockSPSRBSP(tt.chromaFormat)))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALPPS), high12ChromaDeblockPPSRBSP()))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALIDRSlice), high14ChromaDeblockIDRSliceRBSP(tt.chromaFormat, tt.disableDeblock)))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALSlice), high14ChromaDeblock16x16NoResidualSliceRBSP(tt.disableDeblock)))
	sum := md5.Sum(data)
	if got := hex.EncodeToString(sum[:]); got != tt.bitstreamMD5 {
		t.Fatalf("%s bitstream md5 = %s, want %s", tt.name, got, tt.bitstreamMD5)
	}
	return data
}

func high12ChromaDeblockSPSRBSP(chromaFormat uint32) []byte {
	var b decoderSEIBitBuilder
	b.writeBits(244, 8)
	b.writeBits(0, 8)
	b.writeBits(10, 8)
	b.writeUE(0)
	b.writeUE(chromaFormat)
	if chromaFormat == 3 {
		b.writeBit(0)
	}
	b.writeUE(4)
	b.writeUE(4)
	b.writeBit(0)
	b.writeBit(0)
	b.writeUE(0)
	b.writeUE(2)
	b.writeUE(1)
	b.writeBit(0)
	b.writeUE(0)
	b.writeUE(0)
	b.writeBit(1)
	b.writeBit(1)
	b.writeBit(0)
	b.writeBit(0)
	return b.rbsp()
}

func high12ChromaDeblockPPSRBSP() []byte {
	var b decoderSEIBitBuilder
	b.writeUE(0)
	b.writeUE(0)
	b.writeBit(0)
	b.writeBit(0)
	b.writeUE(0)
	b.writeUE(0)
	b.writeUE(0)
	b.writeBit(0)
	b.writeBits(0, 2)
	b.writeSE(-24)
	b.writeSE(0)
	b.writeSE(0)
	b.writeBit(1)
	b.writeBit(0)
	b.writeBit(0)
	b.writeBit(0)
	b.writeBit(0)
	b.writeSE(0)
	return b.rbsp()
}

func assertHigh12ChromaDeblockFixtureSyntax(t *testing.T, data []byte, tt high12ChromaDeblockCase) {
	t.Helper()
	nals, err := h264.SplitAnnexB(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(nals) != 4 {
		t.Fatalf("NAL count = %d, want SPS/PPS/IDR/P", len(nals))
	}

	var spsList [32]*h264.SPS
	var ppsList [256]*h264.PPS
	var gotVCL []h264.NALUnitType
	var pSlices int
	for _, nal := range nals {
		switch nal.Type {
		case h264.NALSPS:
			sps, err := h264.DecodeSPS(nal.RBSP)
			if err != nil {
				t.Fatal(err)
			}
			if sps.ProfileIDC != 244 || sps.Width != 16 || sps.Height != 16 ||
				sps.ChromaFormatIDC != tt.chromaFormat || sps.BitDepthLuma != 12 || sps.BitDepthChroma != 12 ||
				sps.FrameMBSOnlyFlag != 1 || sps.MBAFF != 0 || sps.RefFrameCount != 1 {
				t.Fatalf("SPS profile/format = %d %dx%d chroma %d depth %d/%d frame_only/mbaff %d/%d refs %d, want %s",
					sps.ProfileIDC, sps.Width, sps.Height, sps.ChromaFormatIDC, sps.BitDepthLuma, sps.BitDepthChroma,
					sps.FrameMBSOnlyFlag, sps.MBAFF, sps.RefFrameCount, tt.name)
			}
			spsList[sps.SPSID] = sps
		case h264.NALPPS:
			pps, err := h264.DecodePPS(nal.RBSP, &spsList)
			if err != nil {
				t.Fatal(err)
			}
			if pps.CABAC != 0 || pps.Transform8x8Mode != 0 || pps.WeightedPred != 0 ||
				pps.WeightedBipredIDC != 0 || pps.RefCount[0] != 1 || pps.RefCount[1] != 1 ||
				pps.DeblockingFilterParametersPresent == 0 {
				t.Fatalf("PPS cabac/8x8/weights/refs/deblock-present = %d/%d/%d/%d/%d/%d/%d, want CAVLC/no-8x8 unweighted P ref=1",
					pps.CABAC, pps.Transform8x8Mode, pps.WeightedPred, pps.WeightedBipredIDC,
					pps.RefCount[0], pps.RefCount[1], pps.DeblockingFilterParametersPresent)
			}
			ppsList[pps.PPSID] = pps
		case h264.NALIDRSlice, h264.NALSlice:
			sh, err := h264.ParseSliceHeader(nal, &ppsList)
			if err != nil {
				t.Fatal(err)
			}
			gotVCL = append(gotVCL, nal.Type)
			if sh.PictureStructure != h264.PictureFrame || sh.DeblockingFilter != tt.deblockMode ||
				sh.SliceAlphaC0Offset != 0 || sh.SliceBetaOffset != 0 || sh.QScale != 26 {
				t.Fatalf("slice picture/deblock/offsets/qp = %d/%d/%d/%d/%d, want frame/mode-%d/0/0/26",
					sh.PictureStructure, sh.DeblockingFilter, sh.SliceAlphaC0Offset, sh.SliceBetaOffset, sh.QScale, tt.deblockMode)
			}
			if sh.SliceTypeNoS == h264.PictureTypeP {
				if sh.ListCount != 1 || sh.RefCount[0] != 1 || sh.PPS == nil || sh.PPS.WeightedPred != 0 {
					t.Fatalf("P slice lists/ref/weighted-p = %d/%d/%v, want one L0 ref with unweighted PPS",
						sh.ListCount, sh.RefCount[0], sh.PPS)
				}
				pSlices++
				if sh.PredWeightTable.UseWeight != 0 || sh.PredWeightTable.UseWeightChroma != 0 {
					t.Fatalf("P slice weight flags = %d/%d, want no pred-weight table", sh.PredWeightTable.UseWeight, sh.PredWeightTable.UseWeightChroma)
				}
			}
		default:
			t.Fatalf("unexpected NAL type %d in %s", nal.Type, tt.name)
		}
	}
	if len(gotVCL) != 2 || gotVCL[0] != h264.NALIDRSlice || gotVCL[1] != h264.NALSlice {
		t.Fatalf("VCL NALs = %v, want IDR/P", gotVCL)
	}
	if pSlices != 1 {
		t.Fatalf("P slices = %d, want 1", pSlices)
	}
}

func assertHigh12ChromaDeblockFrames(t *testing.T, frames []*Frame, tt high12ChromaDeblockCase) {
	t.Helper()
	if len(frames) != len(tt.frameMD5) {
		t.Fatalf("frames = %d, want %d", len(frames), len(tt.frameMD5))
	}
	rawVideo := make([]byte, 0, len(frames)*tt.frameSize)
	for i, frame := range frames {
		if frame == nil {
			t.Fatalf("frame[%d] is nil", i)
		}
		if frame.Width != 16 || frame.Height != 16 ||
			frame.ChromaFormatIDC != tt.chromaFormat ||
			frame.BitDepthLuma != 12 || frame.BitDepthChroma != 12 {
			t.Fatalf("frame[%d] metadata = %dx%d chroma %d depth %d/%d, want 16x16 chroma %d High12",
				i, frame.Width, frame.Height, frame.ChromaFormatIDC, frame.BitDepthLuma, frame.BitDepthChroma,
				tt.chromaFormat)
		}
		if got, err := frame.RawPixelFormat(); err != nil || got != tt.pixFmt {
			t.Fatalf("frame[%d] RawPixelFormat = %q/%v, want %s/nil", i, got, err, tt.pixFmt)
		}
		raw, err := frame.AppendRawYUVBytesLE(nil)
		if err != nil {
			t.Fatalf("frame[%d] AppendRawYUVBytesLE: %v", i, err)
		}
		if len(raw) != tt.frameSize {
			t.Fatalf("frame[%d] raw len = %d, want %d", i, len(raw), tt.frameSize)
		}
		sum := md5.Sum(raw)
		if got := hex.EncodeToString(sum[:]); got != tt.frameMD5[i] {
			t.Fatalf("frame[%d] md5 = %s, want %s", i, got, tt.frameMD5[i])
		}
		rawVideo = append(rawVideo, raw...)
	}
	sum := md5.Sum(rawVideo)
	if got := hex.EncodeToString(sum[:]); got != tt.rawVideoMD5 {
		t.Fatalf("rawvideo md5 = %s, want %s", got, tt.rawVideoMD5)
	}
}

func assertFFmpegHigh12ChromaDeblockRawVideoOracle(t *testing.T, data []byte, tt high12ChromaDeblockCase) {
	t.Helper()
	path := writeTempH264(t, data)
	frames, err := h264FFmpegFrameMD5s("ffmpeg", path, tt.pixFmt)
	if err != nil {
		t.Fatalf("ffmpeg framemd5: %v", err)
	}
	if len(frames) != len(tt.frameMD5) {
		t.Fatalf("ffmpeg frame md5 count = %d, want %d", len(frames), len(tt.frameMD5))
	}
	for i, want := range tt.frameMD5 {
		if frames[i] != want {
			t.Fatalf("ffmpeg frame[%d] md5 = %s, want %s", i, frames[i], want)
		}
	}
	raw, err := h264FFmpegRawVideoBytes("ffmpeg", path, tt.pixFmt)
	if err != nil {
		t.Fatalf("ffmpeg rawvideo: %v", err)
	}
	if len(raw) != len(tt.frameMD5)*tt.frameSize {
		t.Fatalf("ffmpeg rawvideo size = %d, want %d", len(raw), len(tt.frameMD5)*tt.frameSize)
	}
	sum := md5.Sum(raw)
	if got := hex.EncodeToString(sum[:]); got != tt.rawVideoMD5 {
		t.Fatalf("ffmpeg rawvideo md5 = %s, want %s", got, tt.rawVideoMD5)
	}
}
