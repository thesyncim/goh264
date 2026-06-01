// SPDX-License-Identifier: LGPL-2.1-or-later

package goh264

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

const high10InterOracleCAVLCIDRPAnnexBHex = `
00000001676e000aa6cb4f6022000003000200000300041e244d400000000168ce0f2c8b0000010605ffff69dc45e9bde6d948b7962cd820d923eeef78323634202d20636f7265203136352072333232322062333536303561202d20482e3236
342f4d5045472d342041564320636f646563202d20436f70796c65667420323030332d32303235202d20687474703a2f2f7777772e766964656f6c616e2e6f72672f783236342e68746d6c202d206f7074696f6e733a2063616261633d302072
65663d31206465626c6f636b3d303a303a3020616e616c7973653d3078333a3078313333206d653d756d68207375626d653d3130207073793d31207073795f72643d312e30303a302e3030206d697865645f7265663d30206d655f72616e6765
3d3234206368726f6d615f6d653d31207472656c6c69733d32203878386463743d312063716d3d3020646561647a6f6e653d32312c313120666173745f70736b69703d31206368726f6d615f71705f6f66667365743d2d322074687265616473
3d31206c6f6f6b61686561645f746872656164733d3120736c696365645f746872656164733d30206e723d3020646563696d6174653d3120696e7465726c616365643d3020626c757261795f636f6d7061743d3020636f6e73747261696e6564
5f696e7472613d3020626672616d65733d3020776569676874703d30206b6579696e743d32206b6579696e745f6d696e3d32207363656e656375743d3020696e7472615f726566726573683d302072635f6c6f6f6b61686561643d322072633d
637266206d62747265653d31206372663d32332e302071636f6d703d302e36302071706d696e3d302071706d61783d3831207170737465703d342069705f726174696f3d312e34302061713d313a312e303000800000016588846ad83a061b06
8001d916ff0b0d4f048bfef98b0661e2e1e50cf23e08861913fa51af527c6f03b510c56d9268be0d1bb5c04e3b1498e7ab961055affbfc62fc852170b5e82b4782c0036ec0b5fc1451356fe0998d98a438825764188076e07e7d860ba76dae59
e62ca50025fc180008020004b0233c000a82389d7ca2c4931aa2c4637b4c18c0d5b9881edf6e4b0087db9f1c04044aa5af78359000f0f01c172d64a65296bbeb8038070ba527ac1eb7afd474c8c0004019425fc051ba4d27fed06e00887e3342
a58cd8eb4000000001419a212f0c0d8009541c98df0194015af61e9d8b7d8267856f3e50156014f920bdce4be1004c22000471c006c689af3728e80d4957cc7cc326004a6fba190c2a7c13132e52f3889f40
`

const high10InterOracleCABACIDRPAnnexBHex = `
00000001676e000aa6cb4f6022000003000200000300041e244d400000000168ee0f2c8b0000010605ffff69dc45e9bde6d948b7962cd820d923eeef78323634202d20636f7265203136352072333232322062333536303561202d20482e3236
342f4d5045472d342041564320636f646563202d20436f70796c65667420323030332d32303235202d20687474703a2f2f7777772e766964656f6c616e2e6f72672f783236342e68746d6c202d206f7074696f6e733a2063616261633d312072
65663d31206465626c6f636b3d303a303a3020616e616c7973653d3078333a3078313333206d653d756d68207375626d653d3130207073793d31207073795f72643d312e30303a302e3030206d697865645f7265663d30206d655f72616e6765
3d3234206368726f6d615f6d653d31207472656c6c69733d32203878386463743d312063716d3d3020646561647a6f6e653d32312c313120666173745f70736b69703d31206368726f6d615f71705f6f66667365743d2d322074687265616473
3d31206c6f6f6b61686561645f746872656164733d3120736c696365645f746872656164733d30206e723d3020646563696d6174653d3120696e7465726c616365643d3020626c757261795f636f6d7061743d3020636f6e73747261696e6564
5f696e7472613d3020626672616d65733d3020776569676874703d30206b6579696e743d32206b6579696e745f6d696e3d32207363656e656375743d3020696e7472615f726566726573683d302072635f6c6f6f6b61686561643d322072633d
637266206d62747265653d31206372663d32332e302071636f6d703d302e36302071706d696e3d302071706d61783d3831207170737465703d342069705f726174696f3d312e34302061713d313a312e303000800000016588846b52d394454d
86176d42565743bf5683bb0c4543d7c50782b4688e59d6cc7aa9fa6ab551e45316e8446e64bbc90c5e6773035a336e339ae19981ddcbadec2df4be08044bafd8b4bcfb12340abdb5da98d4719f27181a9129b3eddc62c5d89fef7ab14c00dcbd
c4b5013a1fce80422e77a731c9649dbcc3b704e0fdf25cbb4a5511ae84444d24b042a701d5b5856799815ca6f5c15109d4e8d9085c516d88c485837d7f5542e234eda00909e95f9daff981f60f7f6f8710b1ef94d8ef862cce80a7f2a1c7f9ab
fb3186cd6d112f925dfb11a8d260c081b78cbfaec3a6a700000001419a2297b92dcb7c4992454ea495c57c14ccaa4c4e9c3de5fc8fcf269563838eac5b13749fd579dda9547fc373fea01b2250401071c276db03691ab3a4e225b9f1badb47c3
fa76847038eafa7e7e6d7263a6584357fe
`

type high10InterOracleFixture struct {
	name        string
	hex         string
	frameMD5    []string
	rawVideoMD5 string
}

func TestHigh10InterIDRPFixtureSyntax(t *testing.T) {
	for _, tt := range high10InterOracleFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			assertHigh10InterIDRPFixtureSyntax(t, decodeHexFixture(t, tt.hex))
		})
	}
}

func TestFFmpegRawVideoFrameMD5OracleHigh10InterIDRP(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}

	for _, tt := range high10InterOracleFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			data := decodeHexFixture(t, tt.hex)
			assertHigh10InterIDRPFixtureSyntax(t, data)
			assertFFmpegHigh10InterRawVideoOracle(t, data, tt.frameMD5, tt.rawVideoMD5)
		})
	}
}

func high10InterOracleFixtures() []high10InterOracleFixture {
	return []high10InterOracleFixture{
		{
			name: "cavlc",
			hex:  high10InterOracleCAVLCIDRPAnnexBHex,
			frameMD5: []string{
				"2d6de0d2739c1d35ff0a20dae4b160b9",
				"b7f5564e9801239f767c533585417968",
			},
			rawVideoMD5: "3158a3c99065e8812397ee21c0908120",
		},
		{
			name: "cabac",
			hex:  high10InterOracleCABACIDRPAnnexBHex,
			frameMD5: []string{
				"019d55eda90e3d86fe5a07fa46c6e6ea",
				"e3c6507eef730da41b826bbcaa6deaef",
			},
			rawVideoMD5: "becbaf3c879970d451fe3554ebb12f7d",
		},
	}
}

func assertFFmpegHigh10InterRawVideoOracle(t *testing.T, data []byte, wantFrameMD5 []string, wantRawVideoMD5 string) {
	t.Helper()
	if len(wantFrameMD5) != 2 {
		t.Fatalf("fixture frame md5 count = %d, want 2", len(wantFrameMD5))
	}

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
	for i, want := range wantFrameMD5 {
		line := []byte(fmt.Sprintf("0, %10d, %10d,        1,      768, %s", i, i, want))
		if !bytes.Contains(framemd5Out, line) {
			t.Fatalf("missing %q in framemd5:\n%s", line, framemd5Out)
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
	if len(raw) != 2*768 {
		t.Fatalf("rawvideo size = %d, want %d", len(raw), 2*768)
	}
	sum := md5.Sum(raw)
	if got := hex.EncodeToString(sum[:]); got != wantRawVideoMD5 {
		t.Fatalf("rawvideo md5 = %s, want %s", got, wantRawVideoMD5)
	}
	for i, want := range wantFrameMD5 {
		sum := md5.Sum(raw[i*768 : (i+1)*768])
		if got := hex.EncodeToString(sum[:]); got != want {
			t.Fatalf("rawvideo frame[%d] md5 = %s, want %s", i, got, want)
		}
	}
}

func assertHigh10InterIDRPFixtureSyntax(t *testing.T, data []byte) {
	t.Helper()
	nals, err := h264.SplitAnnexB(data)
	if err != nil {
		t.Fatal(err)
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
			if sps.ProfileIDC != 110 || sps.Width != 16 || sps.Height != 16 ||
				sps.ChromaFormatIDC != 1 || sps.BitDepthLuma != 10 || sps.BitDepthChroma != 10 {
				t.Fatalf("SPS profile/format = %d %dx%d chroma %d depth %d/%d, want High10 16x16 yuv420p10le",
					sps.ProfileIDC, sps.Width, sps.Height, sps.ChromaFormatIDC, sps.BitDepthLuma, sps.BitDepthChroma)
			}
			spsList[sps.SPSID] = sps
		case h264.NALPPS:
			pps, err := h264.DecodePPS(nal.RBSP, &spsList)
			if err != nil {
				t.Fatal(err)
			}
			ppsList[pps.PPSID] = pps
		case h264.NALIDRSlice, h264.NALSlice:
			sh, err := h264.ParseSliceHeader(nal, &ppsList)
			if err != nil {
				t.Fatal(err)
			}
			if sh.PictureStructure != h264.PictureFrame || sh.DeblockingFilter != 0 {
				t.Fatalf("slice picture/deblock = %d/%d, want frame/disabled", sh.PictureStructure, sh.DeblockingFilter)
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
