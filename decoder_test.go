// SPDX-License-Identifier: LGPL-2.1-or-later

package goh264

import (
	"bytes"
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
