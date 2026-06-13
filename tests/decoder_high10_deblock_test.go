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

const high10DeblockCAVLCAnnexBHex = `
00000001676e000aa6cb44b6022000000300200000030041e244d40000000168ce09c80000016588843f0c6000fa08c298b385cf36544477808531376a2d02325a04c001e2f0ce84342a074bffc38080c647000400219d32a1911c95f8fcfff6abc6f9091f0bafff0e20262aa70c7e4799eaf0e20260161e
723a5281a317efaa350c6e4f9beaf555c606cca13d5e9006b3324702fdfbe480315975704cbf7c08000d00c1cf85001840a001dccc4885c11d797a1af40197251a9fc81c4fffda703e312588d2003af28a88ed518d7a62c15b64f526c02481bfff72580d4e516e2dee905f001de339ce01d66bc0afa265f0
3a3801003990280738056f6ae3250662944b4eb1a8400018276e42c20532700992000113a58010bc653b123e13fe946986190b60bf57aa5a0db90e0109f57b007c842113140769493d010c86e407289aac13a61b210c0115eaf555b18fc8715eaf402190e840729555b01c7623eae0cd2b8f050001006200
0080a018006794a05e2b1018fdf364af4044376089895b204a89ffdf60d4667a01eb4dfbede893e6117577ed7b82ed4b012344291734f6071ffef1823648b8c4b37ff030e1c000b1400040a6838809878581c404c1c2c7fe38809838581c404c1c2c78820005100200b1400c0b0c81c00c070c810431ae16
03e01c8635c1c07c02ce3c040004045600f4c2c0200ec78380401d8f96020231070101180700803b10700803b18380808c41c04046060000803460000801c4008902404f707008902804f70b11c09008970711c0a008972000000001419a23fc3f5010d89c95a89f6f84967944f18acf398eceeb10bc1000
11052da11908005291a63471537c19f863c11690dde800b63a0c9d0357de0633024090
`

const high10DeblockCABACAnnexBHex = `
00000001676e000aa6cb44b6022000000300200000030041e244d40000000168ee09c80000016588843ff9229552af8afb67b0810efb8656c87ce8f0ee34efd27d87caba5bdca5a113fdd2d0c363637a09e08db9aad54a6f3618b60c5b86e9ea785047a39ae981baa5480cec22e6fd9339a92cf5a257e091
eba863c858023a0cae5a310f2a73bf234ad1bcda5711f7f38e98724baf294a352f7e71a9e306d4c6fe23c44f2c31925ae2154fdb718960322a7b7b681a40ce2fa2c69cff556a171fc9b29efafc9d44de7274627f91be44e3e055b635d4c2433b7b1ad099034f810958d2a1659f97932a5df01f501d7ae5e2
ac8eb15edc44371343e2ecbb18b1b9cb4d2dba2238453f3ca03fbf6d83ef5d70d9cefe3b282396c2531523ce177e5b5928c53f25cddef366886537d92585c90b99cbfa172f6b750f98285acc80f15843b7c2cc72be0f10242c059eac879d7bebd1b639287d96fd9b7167e0ef248f79f03539a1f6edda43f1
5a6a33c48f60bfdd020a06bc28dea86025b7964e9ccb9bd9b6d1c91fb5610bba64eb9a5c1af69ddf884efd6b4f79878caa2bf6d879b62bd8b5c829a1a8beef2559ab6bf36309f5c034a91f327dae6a2784450966c9f8309e9716d743a843693b68fac6363f9b79f0ff69b3d3e3a2dfd060ac3e4aa07ab3d5
ce5b15713612196cdc6a474146f360cdde9df178e2a6370600ea7fd5c562a256c95bab3239c5b156b0086655da5b59cb1468b9d2b1e100000001419a23ffbbea7dedd0c3921585a4fb0ea3d92f197ead1e45f1dfac2aff80e460ed76b9ece4c428e90f4a5218ce77bac50da426941d95c477f27ceb61295e
42983b158c
`

var high10DeblockFrameMD5 = []string{
	"ba8f5dc7f864b5cd854ee7d30e89fde1",
	"108cc5e767fced5c958a56f4e65a2278",
}

const (
	high10DeblockRawFrameSize = 3072
	high10DeblockRawVideoMD5  = "b635135b4e7db55894f75c390cf194c2"
)

type high10DeblockFixture struct {
	name  string
	hex   string
	cabac int32
}

func TestDecodeAnnexBHigh10DeblockFrames(t *testing.T) {
	for _, tt := range high10DeblockFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := decodeHexFixture(t, tt.hex)
			assertHigh10DeblockFixtureSyntax(t, data, tt.cabac)

			frames, err := NewDecoder().DecodeAnnexBFrames(data)
			if err != nil {
				t.Fatalf("decode High10 deblock fixture: %v", err)
			}
			assertHigh10DeblockFrames(t, frames)
		})
	}
}

func TestDecodeAVCHigh10DeblockFrames(t *testing.T) {
	for _, tt := range high10DeblockFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := decodeHexFixture(t, tt.hex)
			for _, nalLengthSize := range []int{2, 3, 4} {
				frames, err := NewDecoder().DecodeAVCFrames(annexBToAVC(t, data, nalLengthSize), nalLengthSize)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: %v", nalLengthSize, err)
				}
				assertHigh10DeblockFrames(t, frames)
			}
		})
	}
}

func TestDecodeConfiguredAVCHigh10DeblockFrames(t *testing.T) {
	for _, tt := range high10DeblockFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := decodeHexFixture(t, tt.hex)
			config, samples := annexBToAVCConfigAndSamples(t, data, 4)
			if len(samples) != len(high10DeblockFrameMD5) {
				t.Fatalf("samples = %d, want %d", len(samples), len(high10DeblockFrameMD5))
			}

			dec := NewDecoder()
			if _, err := dec.ConfigureAVCDecoderConfigurationRecord(config); err != nil {
				t.Fatal(err)
			}
			var frames []*Frame
			for i, sample := range samples {
				frame, err := dec.DecodeConfiguredAVC(sample)
				if err != nil {
					t.Fatalf("sample[%d] decode High10 deblock fixture: %v", i, err)
				}
				frames = append(frames, frame)
			}
			assertHigh10DeblockFrames(t, frames)
		})
	}
}

func TestFFmpegRawVideoFrameMD5OracleHigh10Deblock(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}

	for _, tt := range high10DeblockFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := decodeHexFixture(t, tt.hex)
			assertHigh10DeblockFixtureSyntax(t, data, tt.cabac)
			assertFFmpegHigh10DeblockRawVideoOracle(t, data)
		})
	}
}

func high10DeblockFixtures() []high10DeblockFixture {
	return []high10DeblockFixture{
		{name: "cavlc", hex: high10DeblockCAVLCAnnexBHex, cabac: 0},
		{name: "cabac", hex: high10DeblockCABACAnnexBHex, cabac: 1},
	}
}

func assertHigh10DeblockFrames(t *testing.T, frames []*Frame) {
	t.Helper()
	assertHigh10FrameMD5Strings(t, frames, high10DeblockFrameMD5)
	raw := make([]byte, 0, len(frames)*high10DeblockRawFrameSize)
	for i, frame := range frames {
		rawSize, err := frame.RawYUVSize()
		if err != nil {
			t.Fatalf("frame[%d] RawYUVSize: %v", i, err)
		}
		if rawSize != high10DeblockRawFrameSize {
			t.Fatalf("frame[%d] raw yuv size = %d, want %d", i, rawSize, high10DeblockRawFrameSize)
		}
		raw, err = frame.AppendRawYUVBytesLE(raw)
		if err != nil {
			t.Fatalf("frame[%d] AppendRawYUVBytesLE: %v", i, err)
		}
	}
	sum := md5.Sum(raw)
	if got := hex.EncodeToString(sum[:]); got != high10DeblockRawVideoMD5 {
		t.Fatalf("rawvideo md5 = %s, want %s", got, high10DeblockRawVideoMD5)
	}
}

func assertHigh10DeblockFixtureSyntax(t *testing.T, data []byte, cabac int32) {
	t.Helper()
	nals, err := h264.SplitAnnexB(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(nals) != 4 {
		t.Fatalf("NAL count = %d, want 4 stripped SPS/PPS/IDR/P", len(nals))
	}
	wantNALs := []h264.NALUnitType{h264.NALSPS, h264.NALPPS, h264.NALIDRSlice, h264.NALSlice}
	for i, want := range wantNALs {
		if nals[i].Type != want {
			t.Fatalf("NAL[%d] type = %d, want %d", i, nals[i].Type, want)
		}
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
			if sps.ProfileIDC != 110 || sps.Width != 32 || sps.Height != 32 ||
				sps.ChromaFormatIDC != 1 || sps.BitDepthLuma != 10 || sps.BitDepthChroma != 10 {
				t.Fatalf("SPS profile/format = %d %dx%d chroma %d depth %d/%d, want High10 32x32 yuv420p10le",
					sps.ProfileIDC, sps.Width, sps.Height, sps.ChromaFormatIDC, sps.BitDepthLuma, sps.BitDepthChroma)
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
			if pps.CABAC != cabac || pps.Transform8x8Mode != 0 {
				t.Fatalf("PPS cabac/8x8dct = %d/%d, want %d/0", pps.CABAC, pps.Transform8x8Mode, cabac)
			}
			if pps.RefCount != [2]uint32{1, 1} || pps.WeightedPred != 0 || pps.WeightedBipredIDC != 0 {
				t.Fatalf("PPS refs/weight = %v/%d/%d, want ref=1 and unweighted", pps.RefCount, pps.WeightedPred, pps.WeightedBipredIDC)
			}
			if pps.DeblockingFilterParametersPresent != 1 {
				t.Fatalf("PPS deblock params present = %d, want 1", pps.DeblockingFilterParametersPresent)
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
		}
	}
	if len(gotVCL) != 2 || gotVCL[0] != h264.NALIDRSlice || gotVCL[1] != h264.NALSlice {
		t.Fatalf("VCL NALs = %v, want IDR then non-IDR", gotVCL)
	}
	if gotSlices[0] != h264.PictureTypeI || gotSlices[1] != h264.PictureTypeP {
		t.Fatalf("slice types = %v, want I then P", gotSlices)
	}
}

func assertFFmpegHigh10DeblockRawVideoOracle(t *testing.T, data []byte) {
	t.Helper()
	path := writeTempH264(t, data)
	framemd5 := exec.Command("ffmpeg",
		"-v", "error",
		"-f", "h264",
		"-i", path,
		"-an", "-sn", "-dn",
		"-pix_fmt", "yuv420p10le",
		"-f", "framemd5",
		"-",
	)
	framemd5Out, err := framemd5.Output()
	if err != nil {
		t.Fatalf("ffmpeg framemd5: %v", err)
	}
	for i, want := range high10DeblockFrameMD5 {
		line := []byte(fmt.Sprintf("0, %10d, %10d,        1, %8d, %s", i, i, high10DeblockRawFrameSize, want))
		if !bytes.Contains(framemd5Out, line) {
			t.Fatalf("frame[%d] missing %q in framemd5:\n%s", i, line, framemd5Out)
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
	raw, err := rawvideo.Output()
	if err != nil {
		t.Fatalf("ffmpeg rawvideo: %v", err)
	}
	if len(raw) != len(high10DeblockFrameMD5)*high10DeblockRawFrameSize {
		t.Fatalf("rawvideo size = %d, want %d", len(raw), len(high10DeblockFrameMD5)*high10DeblockRawFrameSize)
	}
	sum := md5.Sum(raw)
	if got := hex.EncodeToString(sum[:]); got != high10DeblockRawVideoMD5 {
		t.Fatalf("rawvideo md5 = %s, want %s", got, high10DeblockRawVideoMD5)
	}
}
