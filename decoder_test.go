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

	"github.com/thesyncim/goh264/internal/h264"
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

const testsrc16WeightedPAnnexBHex = `
00000001674d400ada7b011000000300100000030020f1226a0000000168cf025c800000010605ffff51dc45e9bde6d
948b7962cd820d923eeef78323634202d20636f7265203136352072333232322062333536303561202d20482e3236342f
4d5045472d342041564320636f646563202d20436f70796c65667420323030332d32303235202d20687474703a2f2f77
77772e766964656f6c616e2e6f72672f783236342e68746d6c202d206f7074696f6e733a2063616261633d3020726566
3d31206465626c6f636b3d313a303a3020616e616c7973653d303a30206d653d646961207375626d653d30207073793d
31207073795f72643d312e30303a302e3030206d697865645f7265663d30206d655f72616e67653d3136206368726f6d
615f6d653d31207472656c6c69733d30203878386463743d302063716d3d3020646561647a6f6e653d32312c31312066
6173745f70736b69703d31206368726f6d615f71705f6f66667365743d3020746872656164733d31206c6f6f6b616865
61645f746872656164733d3120736c696365645f746872656164733d30206e723d3020646563696d6174653d3120696e
7465726c616365643d3020626c757261795f636f6d7061743d3020636f6e73747261696e65645f696e7472613d302062
6672616d65733d3020776569676874703d32206b6579696e743d323530206b6579696e745f6d696e3d31323620736365
6e656375743d3020696e7472615f726566726573683d302072633d637266206d62747265653d30206372663d33352e30
2071636f6d703d302e36302071706d696e3d302071706d61783d3639207170737465703d342069705f726174696f3d31
2e34302061713d3000800000016588843f2628000834e000000001419a2605ff1b1706913211e949d2c0fc94c6a10eea
779ac468ef7830b60521d05015482083c5003fc1461b72de99e8d40260f12e4d97c1729400000001419a418d03f06e
07a01da020002021ffc6c39e5aa48e88456e34d9a625c3051b7e68df18f2e93ff63153a27e588266c91d9ed9c769f
b5af5d84d53d7bc443ddc77bc45e121ce35e9a94f076b6c31025d471e6aee67ff53d44c87c17a00000001419a61d4
05f0f926010208ff1bb65e43d01b7e19889bb80c25b606de776a18d2f223e7e65610ce780551c2e9448bf410ccca
43bb93434a0d4dbced8d2ab1a29212608099e1ff0349a3f2
`

const testsrc16CABACAnnexBHex = `
00000001674d400ada7b011000000300100000030020f1226a0000000168ee0f2c800000010605ffff6ddc45e9bde6d
948b7962cd820d923eeef78323634202d20636f7265203136352072333232322062333536303561202d20482e3236342f
4d5045472d342041564320636f646563202d20436f70796c65667420323030332d32303235202d20687474703a2f2f77
77772e766964656f6c616e2e6f72672f783236342e68746d6c202d206f7074696f6e733a2063616261633d31207265
663d31206465626c6f636b3d313a313a3020616e616c7973653d3078313a3078313131206d653d686578207375626d65
3d37207073793d31207073795f72643d312e30303a302e3030206d697865645f7265663d30206d655f72616e67653d31
36206368726f6d615f6d653d31207472656c6c69733d31203878386463743d302063716d3d3020646561647a6f6e653d
32312c313120666173745f70736b69703d31206368726f6d615f71705f6f66667365743d2d3220746872656164733d31
206c6f6f6b61686561645f746872656164733d3120736c696365645f746872656164733d30206e723d3020646563696d
6174653d3120696e7465726c616365643d3020626c757261795f636f6d7061743d3020636f6e73747261696e65645f69
6e7472613d3020626672616d65733d3020776569676874703d30206b6579696e743d323530206b6579696e745f6d696e
3d313236207363656e656375743d3020696e7472615f726566726573683d302072635f6c6f6f6b61686561643d34302072
633d637266206d62747265653d31206372663d32332e302071636f6d703d302e36302071706d696e3d302071706d6178
3d3639207170737465703d342069705f726174696f3d312e34302061713d313a312e303000800000016588843d7fb807
d16f5ebb08170ee5539a5348977d63670c41749e4c1b8ad1880e37487b6885eea19035671c61e1c57f07149b8a2b6
f8dcb03eb4c53c8ab4c9110a806d4366e8932cc6f94b005310c0bd460a3b9e877b335ab50d2e5404c32dd68210b
86a877a1ce0e4a7d7cc4de438550e5346d0d74b97aec55913ed42f40f0c7c70cb1356d044e8b2080e25675311e
7f97116c167ec8388ce47cf3cbba718433d7a03d8cb9202a94eb6c515a994ce3778e8d93e02db8e39a795ef1ce75
7ca62ada6677ed111738994d20a7fba5b9bd1d3635d6106f12295032a37dc5f1797241af0dd3f937f49f5de10000
0001419a227aff60949bc49fbde59eaf44cd5388d782a019dbb7ab4bb730b2cb3ccb07846bf150fcd024c5fdb699
90d1681202fbbfffe420b0b21ce69583d2093c7b1608878605ff96f69e31deb00a791d4ba5bfcd2dffc1947a5f
bfb401e8829ad3a1ec838a47200a3f7240514000000001419a425aff523bf415f7b3ec84fdb633d17afd5ca651
37967d81f22e2c4388ccb3a1e31e9c180f1d0ff3c470ceb1d0ffe7b7537c8f7d031506df4ce7a32da46d2c
5856ea076eba90dafa15a6d1c40fcc414500000001419a63afd90aa719c3475592a4047bed17a9de8f346653382872a0
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

func TestParseHeadersAnnexBWeightedP(t *testing.T) {
	data := decodeHexFixture(t, testsrc16WeightedPAnnexBHex)
	dec := NewDecoder()
	info, err := dec.ParseHeadersAnnexB(data)
	if err != nil {
		t.Fatal(err)
	}
	if info.Profile != "Main" || info.ProfileIDC != 77 {
		t.Fatalf("profile = %q/%d, want Main/77", info.Profile, info.ProfileIDC)
	}
	if dec.pps[0] == nil || dec.pps[0].WeightedPred != 1 {
		t.Fatalf("weighted_pred = %+v", dec.pps[0])
	}
	if len(dec.slices) != 4 {
		t.Fatalf("slices = %d, want 4", len(dec.slices))
	}

	pwt := dec.slices[2].PredWeightTable
	if pwt.UseWeight != 1 || pwt.LumaLog2WeightDenom != 5 || pwt.ChromaLog2WeightDenom != 5 {
		t.Fatalf("slice[2] weight header = use %d denom %d/%d", pwt.UseWeight, pwt.LumaLog2WeightDenom, pwt.ChromaLog2WeightDenom)
	}
	if pwt.LumaWeight[0][0] != [2]int32{63, -13} ||
		pwt.ChromaWeight[0][0][0] != [2]int32{61, -118} ||
		pwt.ChromaWeight[0][0][1] != [2]int32{64, -128} {
		t.Fatalf("slice[2] weights = luma %+v chroma %+v", pwt.LumaWeight[0][0], pwt.ChromaWeight[0][0])
	}

	pwt = dec.slices[3].PredWeightTable
	if pwt.UseWeight != 1 || pwt.LumaLog2WeightDenom != 6 || pwt.ChromaLog2WeightDenom != 1 {
		t.Fatalf("slice[3] weight header = use %d denom %d/%d", pwt.UseWeight, pwt.LumaLog2WeightDenom, pwt.ChromaLog2WeightDenom)
	}
	if pwt.LumaWeight[0][0] != [2]int32{95, -7} ||
		pwt.ChromaWeight[0][0][0] != [2]int32{2, 0} ||
		pwt.ChromaWeight[0][0][1] != [2]int32{3, -64} {
		t.Fatalf("slice[3] weights = luma %+v chroma %+v", pwt.LumaWeight[0][0], pwt.ChromaWeight[0][0])
	}
}

func TestParseHeadersAnnexBCABAC(t *testing.T) {
	data := decodeHexFixture(t, testsrc16CABACAnnexBHex)
	dec := NewDecoder()
	info, err := dec.ParseHeadersAnnexB(data)
	if err != nil {
		t.Fatal(err)
	}
	if info.Profile != "Main" || info.ProfileIDC != 77 || info.LevelIDC != 10 {
		t.Fatalf("profile/level = %q/%d/%d, want Main/77/10", info.Profile, info.ProfileIDC, info.LevelIDC)
	}
	if info.Width != 16 || info.Height != 16 || info.ChromaFormatIDC != 1 {
		t.Fatalf("stream = %dx%d chroma %d", info.Width, info.Height, info.ChromaFormatIDC)
	}
	if dec.pps[0] == nil || dec.pps[0].CABAC != 1 || dec.pps[0].DeblockingFilterParametersPresent != 1 {
		t.Fatalf("pps = %+v", dec.pps[0])
	}
	if len(dec.slices) != 4 {
		t.Fatalf("slices = %d, want 4", len(dec.slices))
	}
	if dec.slices[0].SliceTypeNoS != h264.PictureTypeI || dec.slices[1].SliceTypeNoS != h264.PictureTypeP {
		t.Fatalf("slice types = %d/%d", dec.slices[0].SliceTypeNoS, dec.slices[1].SliceTypeNoS)
	}
}

func TestParseHeadersAVCBlack16(t *testing.T) {
	data := decodeHexFixture(t, black16AnnexBHex)
	annexInfo, err := NewDecoder().ParseHeadersAnnexB(data)
	if err != nil {
		t.Fatal(err)
	}

	dec := NewDecoder()
	info, err := dec.ParseHeadersAVC(annexBToAVC(t, data, 4), 4)
	if err != nil {
		t.Fatal(err)
	}
	if info != annexInfo {
		t.Fatalf("info = %+v, want %+v", info, annexInfo)
	}
	if dec.pps[0] == nil || len(dec.slices) != 1 {
		t.Fatalf("retained parser state: pps=%v slices=%d", dec.pps[0] != nil, len(dec.slices))
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

func TestDecodeAVCBlack16Frame(t *testing.T) {
	data := decodeHexFixture(t, black16AnnexBHex)
	frame, err := NewDecoder().DecodeAVC(annexBToAVC(t, data, 4), 4)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := frame.AppendRawYUV(nil)
	if err != nil {
		t.Fatal(err)
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

func TestDecodeAVCTestsrc16Ref2Frames(t *testing.T) {
	data := decodeHexFixture(t, testsrc16Ref2AnnexBHex)
	want := [][16]byte{
		{0x54, 0xb0, 0x49, 0xd0, 0x5d, 0x99, 0xdc, 0x31, 0xd2, 0x70, 0x40, 0x2e, 0x79, 0x8d, 0x4a, 0xf4},
		{0x68, 0x1e, 0x6d, 0x4e, 0xf3, 0x05, 0x8d, 0x38, 0x80, 0x34, 0x6e, 0x80, 0x39, 0xe9, 0x5b, 0x94},
		{0xef, 0x38, 0xcc, 0x80, 0xfb, 0x47, 0xf6, 0x0e, 0x38, 0xab, 0xc2, 0x50, 0x2a, 0xf7, 0xe5, 0xf9},
		{0x0c, 0xee, 0x44, 0xff, 0x1f, 0x82, 0x79, 0xa9, 0x7b, 0xc3, 0xe5, 0x6e, 0x4f, 0x58, 0xf8, 0x02},
	}

	for _, nalLengthSize := range []int{2, 3, 4} {
		frames, err := NewDecoder().DecodeAVCFrames(annexBToAVC(t, data, nalLengthSize), nalLengthSize)
		if err != nil {
			t.Fatalf("nalLengthSize=%d: %v", nalLengthSize, err)
		}
		if len(frames) != 4 {
			t.Fatalf("nalLengthSize=%d: frames = %d, want 4", nalLengthSize, len(frames))
		}
		for i, frame := range frames {
			raw, err := frame.AppendRawYUV(nil)
			if err != nil {
				t.Fatalf("nalLengthSize=%d frame[%d] raw yuv: %v", nalLengthSize, i, err)
			}
			if got := md5.Sum(raw); got != want[i] {
				t.Fatalf("nalLengthSize=%d frame[%d] md5 = %x, want %x", nalLengthSize, i, got, want[i])
			}
		}

		if _, err := NewDecoder().DecodeAVC(annexBToAVC(t, data, nalLengthSize), nalLengthSize); err != ErrUnsupported {
			t.Fatalf("nalLengthSize=%d: single-frame DecodeAVC err = %v, want ErrUnsupported", nalLengthSize, err)
		}
	}
}

func TestDecodeAVCRejectsInvalidLengthPrefix(t *testing.T) {
	for _, tt := range []struct {
		name          string
		data          []byte
		nalLengthSize int
	}{
		{name: "zero length", data: []byte{0, 0, 0, 0}, nalLengthSize: 4},
		{name: "oversized", data: []byte{0, 0, 0, 2, 0x67}, nalLengthSize: 4},
		{name: "bad length size", data: []byte{1, 0x67}, nalLengthSize: 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := NewDecoder().DecodeAVCFrames(tt.data, tt.nalLengthSize); err == nil {
				t.Fatal("expected invalid data")
			}
		})
	}
}

func TestDecodeAnnexBTestsrc16WeightedPFrames(t *testing.T) {
	data := decodeHexFixture(t, testsrc16WeightedPAnnexBHex)
	frames, err := NewDecoder().DecodeAnnexBFrames(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(frames) != 4 {
		t.Fatalf("frames = %d, want 4", len(frames))
	}
	want := [][16]byte{
		{0x8a, 0xae, 0xfe, 0x0a, 0xdc, 0xea, 0x09, 0x4c, 0xfb, 0x51, 0x61, 0xa0, 0x60, 0xba, 0xb4, 0xe2},
		{0x50, 0xde, 0x7a, 0x95, 0x91, 0x98, 0x0d, 0x98, 0x58, 0x0e, 0x8c, 0xc5, 0xbd, 0xf9, 0x07, 0xcb},
		{0xc6, 0xdf, 0x93, 0x14, 0xa9, 0xf5, 0x4e, 0x22, 0xd4, 0x9d, 0xb2, 0x31, 0x6f, 0x12, 0xeb, 0x99},
		{0x92, 0x44, 0x80, 0x3e, 0x5a, 0x61, 0x5a, 0x34, 0x42, 0x76, 0x08, 0x35, 0x0b, 0xe0, 0xfb, 0xda},
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

func TestDecodeAnnexBTestsrc16CABACFrames(t *testing.T) {
	data := decodeHexFixture(t, testsrc16CABACAnnexBHex)
	frames, err := NewDecoder().DecodeAnnexBFrames(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(frames) != 4 {
		t.Fatalf("frames = %d, want 4", len(frames))
	}
	want := []string{
		"57948a884e4468c79f3291b2693263de",
		"4fb1e27b7087e9f1aa485402993ca525",
		"a7e3e74bb19403d111dd2ffdb4455102",
		"1202e58b9b15f56a341fea8787bcc769",
	}
	for i, frame := range frames {
		raw, err := frame.AppendRawYUV(nil)
		if err != nil {
			t.Fatalf("frame[%d] raw yuv: %v", i, err)
		}
		got := md5.Sum(raw)
		if hex.EncodeToString(got[:]) != want[i] {
			t.Fatalf("frame[%d] md5 = %x, want %s", i, got, want[i])
		}
	}
}

func TestDecodeAVCTestsrc16CABACFrames(t *testing.T) {
	data := decodeHexFixture(t, testsrc16CABACAnnexBHex)
	want := []string{
		"57948a884e4468c79f3291b2693263de",
		"4fb1e27b7087e9f1aa485402993ca525",
		"a7e3e74bb19403d111dd2ffdb4455102",
		"1202e58b9b15f56a341fea8787bcc769",
	}
	for _, nalLengthSize := range []int{2, 3, 4} {
		frames, err := NewDecoder().DecodeAVCFrames(annexBToAVC(t, data, nalLengthSize), nalLengthSize)
		if err != nil {
			t.Fatalf("nalLengthSize=%d: %v", nalLengthSize, err)
		}
		if len(frames) != 4 {
			t.Fatalf("nalLengthSize=%d: frames = %d, want 4", nalLengthSize, len(frames))
		}
		for i, frame := range frames {
			raw, err := frame.AppendRawYUV(nil)
			if err != nil {
				t.Fatalf("nalLengthSize=%d frame[%d] raw yuv: %v", nalLengthSize, i, err)
			}
			got := md5.Sum(raw)
			if hex.EncodeToString(got[:]) != want[i] {
				t.Fatalf("nalLengthSize=%d frame[%d] md5 = %x, want %s", nalLengthSize, i, got, want[i])
			}
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

func TestFFmpegFrameMD5OracleTestsrc16WeightedP(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}

	data := decodeHexFixture(t, testsrc16WeightedPAnnexBHex)
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
		"0,          0,          0,        1,      384, 8aaefe0adcea094cfb5161a060bab4e2",
		"0,          1,          1,        1,      384, 50de7a9591980d98580e8cc5bdf907cb",
		"0,          2,          2,        1,      384, c6df9314a9f54e22d49db2316f12eb99",
		"0,          3,          3,        1,      384, 9244803e5a615a34427608350be0fbda",
	} {
		if !bytes.Contains(out, []byte(line)) {
			t.Fatalf("missing %q in framemd5:\n%s", line, out)
		}
	}
}

func TestFFmpegFrameMD5OracleTestsrc16CABAC(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}

	data := decodeHexFixture(t, testsrc16CABACAnnexBHex)
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
		"0,          0,          0,        1,      384, 57948a884e4468c79f3291b2693263de",
		"0,          1,          1,        1,      384, 4fb1e27b7087e9f1aa485402993ca525",
		"0,          2,          2,        1,      384, a7e3e74bb19403d111dd2ffdb4455102",
		"0,          3,          3,        1,      384, 1202e58b9b15f56a341fea8787bcc769",
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

func annexBToAVC(t *testing.T, data []byte, nalLengthSize int) []byte {
	t.Helper()
	if nalLengthSize < 1 || nalLengthSize > 4 {
		t.Fatalf("invalid nalLengthSize %d", nalLengthSize)
	}

	nals, err := h264.SplitAnnexB(data)
	if err != nil {
		t.Fatal(err)
	}
	maxSize := uint64(1)<<(uint(nalLengthSize)*8) - 1
	var out []byte
	for _, nal := range nals {
		size := len(nal.Raw)
		if uint64(size) > maxSize {
			t.Fatalf("NAL size %d exceeds %d-byte length field", size, nalLengthSize)
		}
		for shift := (nalLengthSize - 1) * 8; shift >= 0; shift -= 8 {
			out = append(out, byte(size>>shift))
		}
		out = append(out, nal.Raw...)
	}
	return out
}

func writeTempH264(t *testing.T, data []byte) string {
	t.Helper()
	path := t.TempDir() + "/fixture.h264"
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
