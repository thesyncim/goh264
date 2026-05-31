// SPDX-License-Identifier: LGPL-2.1-or-later

package goh264

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"
)

const black16AnnexBHex = `
000000016742c01eddec0440000003004000000300a3c58be00000000168ce0fc80000010605ffff4ddc45e9bde6d948
b7962cd820d923eeef78323634202d20636f7265203136352072333232322062333536303561202d20482e3236342f4d
5045472d342041564320636f646563202d20436f70796c65667420323030332d32303235202d20687474703a2f2f7777
772e766964656f6c616e2e6f72672f783236342e68746d6c202d206f7074696f6e733a2063616261633d30207265663d
31206465626c6f636b3d303a303a3020616e616c7973653d303a30206d653d646961207375626d653d30207073793d31
207073795f72643d312e30303a302e3030206d697865645f7265663d30206d655f72616e67653d3136206368726f6d61
5f6d653d31207472656c6c69733d30203878386463743d302063716d3d3020646561647a6f6e653d32312c3131206661
73745f70736b69703d31206368726f6d615f71705f6f66667365743d3020746872656164733d31206c6f6f6b61686561
645f746872656164733d3120736c696365645f746872656164733d30206e723d3020646563696d6174653d3120696e74
65726c616365643d3020626c757261795f636f6d7061743d3020636f6e73747261696e65645f696e7472613d30206266
72616d65733d3020776569676874703d30206b6579696e743d31206b6579696e745f6d696e3d31207363656e65637574
3d3020696e7472615f726566726573683d302072633d637266206d62747265653d30206372663d32332e302071636f6d
703d302e36302071706d696e3d302071706d61783d3639207170737465703d342069705f726174696f3d312e34302061
713d3000800000016588843a2628000902e0
`

const black16IPAnnexBHex = `
000000016742c00ada7b011000000300100000030028f1226a0000000168ce0fc80000010605ffff51dc45e9bde6d948
b7962cd820d923eeef78323634202d20636f7265203136352072333232322062333536303561202d20482e3236342f4d
5045472d342041564320636f646563202d20436f70796c65667420323030332d32303235202d20687474703a2f2f7777
772e766964656f6c616e2e6f72672f783236342e68746d6c202d206f7074696f6e733a2063616261633d30207265663d
31206465626c6f636b3d303a303a3020616e616c7973653d303a30206d653d646961207375626d653d30207073793d31
207073795f72643d312e30303a302e3030206d697865645f7265663d30206d655f72616e67653d3136206368726f6d61
5f6d653d31207472656c6c69733d30203878386463743d302063716d3d3020646561647a6f6e653d32312c3131206661
73745f70736b69703d31206368726f6d615f71705f6f66667365743d3020746872656164733d31206c6f6f6b61686561
645f746872656164733d3120736c696365645f746872656164733d30206e723d3020646563696d6174653d3120696e74
65726c616365643d3020626c757261795f636f6d7061743d3020636f6e73747261696e65645f696e7472613d30206266
72616d65733d3020776569676874703d30206b6579696e743d323530206b6579696e745f6d696e3d313236207363656e
656375743d3020696e7472615f726566726573683d302072633d637266206d62747265653d30206372663d32332e3020
71636f6d703d302e36302071706d696e3d302071706d61783d3639207170737465703d342069705f726174696f3d312e
34302061713d3000800000016588843a2628000902e000000001419a2014a5
`

const testsrc16DeblockAnnexBHex = `
000000016742c00ada7b011000000300100000030020f1226a0000000168ce025c800000010605ffff51dc45e9bde6d
948b7962cd820d923eeef78323634202d20636f7265203136352072333232322062333536303561202d20482e3236342f
4d5045472d342041564320636f646563202d20436f70796c65667420323030332d32303235202d20687474703a2f2f77
77772e766964656f6c616e2e6f72672f783236342e68746d6c202d206f7074696f6e733a2063616261633d3020726566
3d31206465626c6f636b3d313a303a3020616e616c7973653d303a30206d653d646961207375626d653d30207073793d
31207073795f72643d312e30303a302e3030206d697865645f7265663d30206d655f72616e67653d3136206368726f6d
615f6d653d31207472656c6c69733d30203878386463743d302063716d3d3020646561647a6f6e653d32312c31312066
6173745f70736b69703d31206368726f6d615f71705f6f66667365743d3020746872656164733d31206c6f6f6b616865
61645f746872656164733d3120736c696365645f746872656164733d30206e723d3020646563696d6174653d3120696e
7465726c616365643d3020626c757261795f636f6d7061743d3020636f6e73747261696e65645f696e7472613d302062
6672616d65733d3020776569676874703d30206b6579696e743d323530206b6579696e745f6d696e3d31323620736365
6e656375743d3020696e7472615f726566726573683d302072633d637266206d62747265653d30206372663d33352e30
2071636f6d703d302e36302071706d696e3d302071706d61783d3639207170737465703d342069705f726174696f3d31
2e34302061713d3000800000016588843f0c60007225021e249d0097af0e71e4c9e58006113914e1feff4601d9812e50
32094cb78f77fb322e4a719f7f8f0b0d232c59e3c2c05b35cc287f7d27562cbcf55e794d262e7a41d254c0fdfbe40
cd398287f7d03800518602c1a00ce52384c793469d02c0f3718d1ffbc385c429623483ddd01dcbcc4b22cfa31ec48
cffbf186dc3836bc80
`

const testsrc16IPDeblockAnnexBHex = testsrc16DeblockAnnexBHex + `
00000001419a20ffc2d4c031602e32a4bf0483732e1009dca2840048ca30d8a77106dff4b1b7e00a89d0b18ec4c
0c3c0
`

const testsrc16Ref2AnnexBHex = `
000000016742c00adb7b011000000300100000030020f1226e0000000168ca8097200000010605ffff51dc45e9bde6d
948b7962cd820d923eeef78323634202d20636f7265203136352072333232322062333536303561202d20482e3236342f
4d5045472d342041564320636f646563202d20436f70796c65667420323030332d32303235202d20687474703a2f2f77
77772e766964656f6c616e2e6f72672f783236342e68746d6c202d206f7074696f6e733a2063616261633d3020726566
3d32206465626c6f636b3d313a303a3020616e616c7973653d303a30206d653d646961207375626d653d30207073793d
31207073795f72643d312e30303a302e3030206d697865645f7265663d30206d655f72616e67653d3136206368726f6d
615f6d653d31207472656c6c69733d30203878386463743d302063716d3d3020646561647a6f6e653d32312c31312066
6173745f70736b69703d31206368726f6d615f71705f6f66667365743d3020746872656164733d31206c6f6f6b616865
61645f746872656164733d3120736c696365645f746872656164733d30206e723d3020646563696d6174653d3120696e
7465726c616365643d3020626c757261795f636f6d7061743d3020636f6e73747261696e65645f696e7472613d302062
6672616d65733d3020776569676874703d30206b6579696e743d323530206b6579696e745f6d696e3d31323620736365
6e656375743d3020696e7472615f726566726573683d302072633d637266206d62747265653d30206372663d33352e30
2071636f6d703d302e36302071706d696e3d302071706d61783d3639207170737465703d342069705f726174696f3d31
2e34302061713d3000800000016588843f0c60007225021e249d0097af0e71e4c9e58006113914e1feff4601d9812e50
32094cb78f77fb322e4a719f7f8f0b0d232c59e3c2c05b35cc287f7d27562cbcf55e794d262e7a41d254c0fdfbe40
cd398287f7d03800518602c1a00ce52384c793469d02c0f3718d1ffbc385c429623483ddd01dcbcc4b22cfa31ec48
cffbf186dc3836bc8000000001419a387fe16a6018b01719525f8241b9970804ee5142002465186c53b8836ffa58dbf0
0544e858c7626061e000000001419a405ff85ac80c74843f4b11eb2726fef48216d9208aab55f07ed7d7814744b3fd
038000000001419a607ff8d87b097f8662a8390581e6916a4d14be58430389468a086d081fe0
`

func TestParseHeadersAnnexBBlack16(t *testing.T) {
	data := decodeHexFixture(t, black16AnnexBHex)
	dec := NewDecoder()
	info, err := dec.ParseHeadersAnnexB(data)
	if err != nil {
		t.Fatal(err)
	}

	if info.Profile != "Constrained Baseline" {
		t.Fatalf("profile = %q", info.Profile)
	}
	if info.ProfileIDC != 66 || info.LevelIDC != 30 {
		t.Fatalf("profile/level = %d/%d", info.ProfileIDC, info.LevelIDC)
	}
	if info.Width != 16 || info.Height != 16 {
		t.Fatalf("size = %dx%d", info.Width, info.Height)
	}
	if info.ChromaFormatIDC != 1 || info.BitDepthLuma != 8 || info.BitDepthChroma != 8 {
		t.Fatalf("format = chroma %d depth %d/%d", info.ChromaFormatIDC, info.BitDepthLuma, info.BitDepthChroma)
	}
	if dec.pps[0] == nil {
		t.Fatal("PPS 0 was not retained")
	}
	if dec.pps[0].CABAC != 0 || dec.pps[0].SliceGroupCount != 1 || dec.pps[0].RefCount != [2]uint32{1, 1} {
		t.Fatalf("pps = %+v", dec.pps[0])
	}
	if dec.pps[0].ChromaQPTable[0][30] != 29 || dec.pps[0].Dequant4Buffer[0][0][0] != 640 {
		t.Fatalf("pps tables not initialized: chromaQP=%d dequant=%d", dec.pps[0].ChromaQPTable[0][30], dec.pps[0].Dequant4Buffer[0][0][0])
	}
	if len(dec.slices) != 1 {
		t.Fatalf("slices = %d", len(dec.slices))
	}
	if dec.slices[0].SliceType != 1 || dec.slices[0].PPSID != 0 || dec.slices[0].PictureStructure != 3 {
		t.Fatalf("slice = %+v", dec.slices[0])
	}
	if dec.slices[0].ChromaQP != [2]uint8{dec.pps[0].ChromaQPTable[0][dec.slices[0].QScale], dec.pps[0].ChromaQPTable[1][dec.slices[0].QScale]} {
		t.Fatalf("slice chroma qp = %+v", dec.slices[0].ChromaQP)
	}
}

func TestDecodeAnnexBBlack16Frame(t *testing.T) {
	data := decodeHexFixture(t, black16AnnexBHex)
	frame, err := NewDecoder().DecodeAnnexB(data)
	if err != nil {
		t.Fatal(err)
	}
	if frame.Width != 16 || frame.Height != 16 || frame.ChromaFormatIDC != 1 || frame.BitDepthLuma != 8 || frame.BitDepthChroma != 8 {
		t.Fatalf("frame metadata = %dx%d chroma %d depth %d/%d", frame.Width, frame.Height, frame.ChromaFormatIDC, frame.BitDepthLuma, frame.BitDepthChroma)
	}
	raw, err := frame.AppendRawYUV(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) != 384 {
		t.Fatalf("raw frame size = %d, want 384", len(raw))
	}
	if got := md5.Sum(raw); got != [16]byte{0x8a, 0xae, 0xfe, 0x0a, 0xdc, 0xea, 0x09, 0x4c, 0xfb, 0x51, 0x61, 0xa0, 0x60, 0xba, 0xb4, 0xe2} {
		t.Fatalf("frame md5 = %x, want 8aaefe0adcea094cfb5161a060bab4e2", got)
	}
}

func TestDecodeAnnexBBlack16IPFrames(t *testing.T) {
	data := decodeHexFixture(t, black16IPAnnexBHex)
	dec := NewDecoder()
	frames, err := dec.DecodeAnnexBFrames(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(frames) != 2 {
		t.Fatalf("frames = %d, want 2", len(frames))
	}
	for i, frame := range frames {
		if frame.Width != 16 || frame.Height != 16 || frame.ChromaFormatIDC != 1 {
			t.Fatalf("frame[%d] metadata = %dx%d chroma %d", i, frame.Width, frame.Height, frame.ChromaFormatIDC)
		}
		raw, err := frame.AppendRawYUV(nil)
		if err != nil {
			t.Fatalf("frame[%d] raw yuv: %v", i, err)
		}
		if got := md5.Sum(raw); got != [16]byte{0x8a, 0xae, 0xfe, 0x0a, 0xdc, 0xea, 0x09, 0x4c, 0xfb, 0x51, 0x61, 0xa0, 0x60, 0xba, 0xb4, 0xe2} {
			t.Fatalf("frame[%d] md5 = %x, want 8aaefe0adcea094cfb5161a060bab4e2", i, got)
		}
	}
	if _, err := dec.DecodeAnnexB(data); err != ErrUnsupported {
		t.Fatalf("single-frame DecodeAnnexB err = %v, want ErrUnsupported for multi-frame packet", err)
	}
}

func TestDecodeAnnexBTestsrc16DeblockFrame(t *testing.T) {
	data := decodeHexFixture(t, testsrc16DeblockAnnexBHex)
	frame, err := NewDecoder().DecodeAnnexB(data)
	if err != nil {
		t.Fatal(err)
	}
	if frame.Width != 16 || frame.Height != 16 || frame.ChromaFormatIDC != 1 {
		t.Fatalf("frame metadata = %dx%d chroma %d", frame.Width, frame.Height, frame.ChromaFormatIDC)
	}
	raw, err := frame.AppendRawYUV(nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := md5.Sum(raw); got != [16]byte{0x54, 0xb0, 0x49, 0xd0, 0x5d, 0x99, 0xdc, 0x31, 0xd2, 0x70, 0x40, 0x2e, 0x79, 0x8d, 0x4a, 0xf4} {
		t.Fatalf("frame md5 = %x, want 54b049d05d99dc31d270402e798d4af4", got)
	}
}

func TestDecodeAnnexBTestsrc16IPDeblockFrames(t *testing.T) {
	data := decodeHexFixture(t, testsrc16IPDeblockAnnexBHex)
	frames, err := NewDecoder().DecodeAnnexBFrames(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(frames) != 2 {
		t.Fatalf("frames = %d, want 2", len(frames))
	}
	want := [][16]byte{
		{0x54, 0xb0, 0x49, 0xd0, 0x5d, 0x99, 0xdc, 0x31, 0xd2, 0x70, 0x40, 0x2e, 0x79, 0x8d, 0x4a, 0xf4},
		{0x68, 0x1e, 0x6d, 0x4e, 0xf3, 0x05, 0x8d, 0x38, 0x80, 0x34, 0x6e, 0x80, 0x39, 0xe9, 0x5b, 0x94},
	}
	for i, frame := range frames {
		raw, err := frame.AppendRawYUV(nil)
		if err != nil {
			t.Fatalf("frame[%d] raw yuv: %v", i, err)
		}
		if got := md5.Sum(raw); got != want[i] {
			t.Fatalf("frame[%d] md5 = %x, want %x", i, got, want[i])
		}
	}
}

func TestDecodeAnnexBTestsrc16Ref2Frames(t *testing.T) {
	data := decodeHexFixture(t, testsrc16Ref2AnnexBHex)
	frames, err := NewDecoder().DecodeAnnexBFrames(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(frames) != 4 {
		t.Fatalf("frames = %d, want 4", len(frames))
	}
	want := [][16]byte{
		{0x54, 0xb0, 0x49, 0xd0, 0x5d, 0x99, 0xdc, 0x31, 0xd2, 0x70, 0x40, 0x2e, 0x79, 0x8d, 0x4a, 0xf4},
		{0x68, 0x1e, 0x6d, 0x4e, 0xf3, 0x05, 0x8d, 0x38, 0x80, 0x34, 0x6e, 0x80, 0x39, 0xe9, 0x5b, 0x94},
		{0xef, 0x38, 0xcc, 0x80, 0xfb, 0x47, 0xf6, 0x0e, 0x38, 0xab, 0xc2, 0x50, 0x2a, 0xf7, 0xe5, 0xf9},
		{0x0c, 0xee, 0x44, 0xff, 0x1f, 0x82, 0x79, 0xa9, 0x7b, 0xc3, 0xe5, 0x6e, 0x4f, 0x58, 0xf8, 0x02},
	}
	for i, frame := range frames {
		raw, err := frame.AppendRawYUV(nil)
		if err != nil {
			t.Fatalf("frame[%d] raw yuv: %v", i, err)
		}
		if got := md5.Sum(raw); got != want[i] {
			t.Fatalf("frame[%d] md5 = %x, want %x", i, got, want[i])
		}
	}
}

func TestFFprobeOracleBlack16(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffprobe oracle")
	}
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe not available")
	}

	data := decodeHexFixture(t, black16AnnexBHex)
	path := writeTempH264(t, data)

	cmd := exec.Command("ffprobe",
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=codec_name,profile,width,height,level,pix_fmt",
		"-of", "json",
		path,
	)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("ffprobe: %v", err)
	}

	var probe struct {
		Streams []struct {
			CodecName string `json:"codec_name"`
			Profile   string `json:"profile"`
			Width     int    `json:"width"`
			Height    int    `json:"height"`
			Level     int    `json:"level"`
			PixFmt    string `json:"pix_fmt"`
		} `json:"streams"`
	}
	if err := json.Unmarshal(out, &probe); err != nil {
		t.Fatal(err)
	}
	if len(probe.Streams) != 1 {
		t.Fatalf("ffprobe streams = %d", len(probe.Streams))
	}

	info, err := NewDecoder().ParseHeadersAnnexB(data)
	if err != nil {
		t.Fatal(err)
	}
	stream := probe.Streams[0]
	if stream.CodecName != "h264" || stream.PixFmt != "yuv420p" {
		t.Fatalf("unexpected oracle stream: %+v", stream)
	}
	if stream.Profile != info.Profile || stream.Width != info.Width || stream.Height != info.Height || stream.Level != int(info.LevelIDC) {
		t.Fatalf("oracle %+v, go %+v", stream, info)
	}
}

func TestFFmpegFrameMD5OracleBlack16(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}

	data := decodeHexFixture(t, black16AnnexBHex)
	path := writeTempH264(t, data)

	cmd := exec.Command("ffmpeg",
		"-v", "error",
		"-f", "h264",
		"-i", path,
		"-an", "-sn", "-dn",
		"-f", "framemd5",
		"-",
	)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("ffmpeg framemd5: %v", err)
	}
	if !bytes.Contains(out, []byte("0,          0,          0,        1,      384, 8aaefe0adcea094cfb5161a060bab4e2")) {
		t.Fatalf("unexpected framemd5:\n%s", out)
	}
}

func TestFFmpegFrameMD5OracleBlack16IP(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}

	data := decodeHexFixture(t, black16IPAnnexBHex)
	path := writeTempH264(t, data)

	cmd := exec.Command("ffmpeg",
		"-v", "error",
		"-f", "h264",
		"-i", path,
		"-an", "-sn", "-dn",
		"-f", "framemd5",
		"-",
	)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("ffmpeg framemd5: %v", err)
	}
	if !bytes.Contains(out, []byte("0,          0,          0,        1,      384, 8aaefe0adcea094cfb5161a060bab4e2")) ||
		!bytes.Contains(out, []byte("0,          1,          1,        1,      384, 8aaefe0adcea094cfb5161a060bab4e2")) {
		t.Fatalf("unexpected framemd5:\n%s", out)
	}
}

func TestFFmpegFrameMD5OracleTestsrc16Deblock(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}

	data := decodeHexFixture(t, testsrc16DeblockAnnexBHex)
	path := writeTempH264(t, data)

	cmd := exec.Command("ffmpeg",
		"-v", "error",
		"-f", "h264",
		"-i", path,
		"-an", "-sn", "-dn",
		"-f", "framemd5",
		"-",
	)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("ffmpeg framemd5: %v", err)
	}
	if !bytes.Contains(out, []byte("0,          0,          0,        1,      384, 54b049d05d99dc31d270402e798d4af4")) {
		t.Fatalf("unexpected framemd5:\n%s", out)
	}
}

func TestFFmpegFrameMD5OracleTestsrc16IPDeblock(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}

	data := decodeHexFixture(t, testsrc16IPDeblockAnnexBHex)
	path := writeTempH264(t, data)

	cmd := exec.Command("ffmpeg",
		"-v", "error",
		"-f", "h264",
		"-i", path,
		"-an", "-sn", "-dn",
		"-f", "framemd5",
		"-",
	)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("ffmpeg framemd5: %v", err)
	}
	if !bytes.Contains(out, []byte("0,          0,          0,        1,      384, 54b049d05d99dc31d270402e798d4af4")) ||
		!bytes.Contains(out, []byte("0,          1,          1,        1,      384, 681e6d4ef3058d3880346e8039e95b94")) {
		t.Fatalf("unexpected framemd5:\n%s", out)
	}
}

func TestFFmpegFrameMD5OracleTestsrc16Ref2(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}

	data := decodeHexFixture(t, testsrc16Ref2AnnexBHex)
	path := writeTempH264(t, data)

	cmd := exec.Command("ffmpeg",
		"-v", "error",
		"-f", "h264",
		"-i", path,
		"-an", "-sn", "-dn",
		"-f", "framemd5",
		"-",
	)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("ffmpeg framemd5: %v", err)
	}
	for _, line := range []string{
		"0,          0,          0,        1,      384, 54b049d05d99dc31d270402e798d4af4",
		"0,          1,          1,        1,      384, 681e6d4ef3058d3880346e8039e95b94",
		"0,          2,          2,        1,      384, ef38cc80fb47f60e38abc2502af7e5f9",
		"0,          3,          3,        1,      384, 0cee44ff1f8279a97bc3e56e4f58f802",
	} {
		if !bytes.Contains(out, []byte(line)) {
			t.Fatalf("missing %q in framemd5:\n%s", line, out)
		}
	}
}

func decodeHexFixture(t *testing.T, s string) []byte {
	t.Helper()
	clean := strings.NewReplacer("\n", "", "\t", "", " ", "").Replace(s)
	data, err := hex.DecodeString(clean)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func writeTempH264(t *testing.T, data []byte) string {
	t.Helper()
	path := t.TempDir() + "/fixture.h264"
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
