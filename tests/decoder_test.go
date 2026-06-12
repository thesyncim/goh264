// SPDX-License-Identifier: LGPL-2.1-or-later

package goh264_test

import (
	"bytes"
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/bits"
	"os"
	"os/exec"
	"path/filepath"
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

const testsrc16CAVLC422AnnexBHex = `
00000001677a000abcb4f6022000000300200000030041e244d40000000168ce0f2c800000010605ffff6ddc45e9bde
6d948b7962cd820d923eeef78323634202d20636f7265203136352072333232322062333536303561202d20482e323634
2f4d5045472d342041564320636f646563202d20436f70796c65667420323030332d32303235202d20687474703a2f
2f7777772e766964656f6c616e2e6f72672f783236342e68746d6c202d206f7074696f6e733a2063616261633d3020
7265663d31206465626c6f636b3d313a313a3020616e616c7973653d3078313a3078313131206d653d68657820737562
6d653d37207073793d31207073795f72643d312e30303a302e3030206d697865645f7265663d30206d655f72616e67
653d3136206368726f6d615f6d653d31207472656c6c69733d31203878386463743d302063716d3d3020646561647a
6f6e653d32312c313120666173745f70736b69703d31206368726f6d615f71705f6f66667365743d2d322074687265
6164733d31206c6f6f6b61686561645f746872656164733d3120736c696365645f746872656164733d30206e723d30
20646563696d6174653d3120696e7465726c616365643d3020626c757261795f636f6d7061743d3020636f6e737472
61696e65645f696e7472613d3020626672616d65733d3020776569676874703d30206b6579696e743d323530206b65
79696e745f6d696e3d313236207363656e656375743d3020696e7472615f726566726573683d302072635f6c6f6f6b
61686561643d34302072633d637266206d62747265653d31206372663d32332e302071636f6d703d302e3630207170
6d696e3d302071706d61783d3639207170737465703d342069705f726174696f3d312e34302061713d313a312e3030
00800000016588843d710c5581810bf0045360c46003eb5061800391978535441d84a825125ffe03a081d900ba036141
4b862b578c5905731a7793be9ecf088b4727985635102515841fa4c110b9823ea97814ffec0883274b47561d3e114f
fac881b2050c497444663d5eb70970a2f887496cf57a800350c510ac2c05a29ff7cd601a0ee6a2de0e8cb17f800e8
18080e1ec8117f003800448d241c3e10002c13b08f0cf6785a31814e1b029a4610421fe601987042250b03ac89eb
252490001c6609afd202b5300024433a530002217f32ecf4bbc000404868bbcae400b5a2301367840001a03810272
4b56780fbf4c802687d5251831ae244c2b33a3b60002010345e85724328d0000808c2b573d000012a1cc1d0848b
656a0002006c0b9174aa800000001419a20f5f1b0e0783a4801a05018984036d45d7f3f7bc938e30042d308a42
d6afd8f7c00e0007c00e00d2264225a460071c0954c31000b00233c2448e67cf760acda600060016d1603d3785
981fbf8f0bf4257dd870058000805a00a14d74219eaaa661420080d130104c69a5ed26a4494cff498a4c1f000000
001419a40b5f1b803f203e25529c58171a6565ef7699f00e1d9ad4d97adc22290b5aaf533b1ec01a401a0a5bf85
ee500bd030cacf78dc090697a31a558038da14b8f0b612b16b1f3c01e04f1733bae0f840900ee52679177a61d5
f558d62185362000000001419a635f0cf82fb841ff70b2c01166b958f0fe00bb6bacf5b5beff80
`

const testsrc16CABAC422AnnexBHex = `
00000001677a000abcb4f6022000000300200000030041e244d40000000168ee0f2c800000010605ffff6ddc45e9bde
6d948b7962cd820d923eeef78323634202d20636f7265203136352072333232322062333536303561202d20482e323634
2f4d5045472d342041564320636f646563202d20436f70796c65667420323030332d32303235202d20687474703a2f
2f7777772e766964656f6c616e2e6f72672f783236342e68746d6c202d206f7074696f6e733a2063616261633d3120
7265663d31206465626c6f636b3d313a313a3020616e616c7973653d3078313a3078313131206d653d68657820737562
6d653d37207073793d31207073795f72643d312e30303a302e3030206d697865645f7265663d30206d655f72616e67
653d3136206368726f6d615f6d653d31207472656c6c69733d31203878386463743d302063716d3d3020646561647a
6f6e653d32312c313120666173745f70736b69703d31206368726f6d615f71705f6f66667365743d2d322074687265
6164733d31206c6f6f6b61686561645f746872656164733d3120736c696365645f746872656164733d30206e723d30
20646563696d6174653d3120696e7465726c616365643d3020626c757261795f636f6d7061743d3020636f6e737472
61696e65645f696e7472613d3020626672616d65733d3020776569676874703d30206b6579696e743d323530206b65
79696e745f6d696e3d313236207363656e656375743d3020696e7472615f726566726573683d302072635f6c6f6f6b
61686561643d34302072633d637266206d62747265653d31206372663d32332e302071636f6d703d302e3630207170
6d696e3d302071706d61783d3639207170737465703d342069705f726174696f3d312e34302061713d313a312e3030
00800000016588843d7fb807d16f5ebb08170ee5539a5348977d63670c41749e4c1b8ad1880e37487b6885eea190
35671c61e1c57f07149b8a2b6f8dcb03eb4c53c8ab4c9110a806d4366e8932cc6f94b005310c0bd460a3b9e87
7b335ab50d2e5404c32dd68210b86a877a1ce0e4a7d7cc4de438550e5346d0d74b97aec55913ed42f40f0c7c
70cb13301bf34c4dabe48de7ce8f7189f945a6e8609dfb626228a083a5f889d1e0bffca5fc07f0b80c0682c0bc
506a746b4a9f77f7b8dbcce85a2364e18ce3c967e9095dd1371407c06e9031fead250899c71c86d55eef84c
9142dc34f87ef41ed25a9d2339d0a28e9cfbe69eb9efc4ce821f0ab48449146d6d9b7f2332255d7e1d92d12
b29fb86f1736c9da3678100000001419a227aff60949bc49fbde59eaf44cd5388d782a019dbb7ab4bb730b2cb3
ccb07846bf00bc37cc9d04e0c9cd1b317df12412cc858788eb3463f5fa65ff0e755245cd1b6232f83edafffdec
71bf76d11778b0a3ee3eb9ee9895eb1e13de8a8632b995bc14f8407d4ecfb9f008a093c6d68d6f966ea259140
cd157dc8734886eb927ffe00000001419a425aff523bf415f7b3ec84fdb633d17afd5ca65137967d81f22e2c4388
ccb3a1e31e9c180f08fe1835891d369c18261e698f3edb857dae08f7e17cd8b746b6546637ad5cec5509c9d6
9cead974d0da4ab018cb5b777c853538ac73129a9aa0d7c900000001419a63afd90aa719c34755988b947c0a60b
16ce16dad84c715edf368d1cad2
`

func TestParseHeadersRejectsNilDecoder(t *testing.T) {
	var dec *Decoder
	if _, err := dec.ParseHeadersAnnexB([]byte{0, 0, 1, 0x67}); err != ErrInvalidData {
		t.Fatalf("ParseHeadersAnnexB nil decoder error = %v, want ErrInvalidData", err)
	}
	if _, err := dec.ParseHeadersAVC([]byte{0, 0, 0, 1, 0x67}, 4); err != ErrInvalidData {
		t.Fatalf("ParseHeadersAVC nil decoder error = %v, want ErrInvalidData", err)
	}
}

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
	if info.SARDen != 1 || info.VideoFullRangeFlag != -1 || info.ColorMatrix != 2 {
		t.Fatalf("vui defaults = sar %d:%d range %d matrix %d", info.SARNum, info.SARDen, info.VideoFullRangeFlag, info.ColorMatrix)
	}
}

func TestParseHeadersAnnexBExposesVUIMetadata(t *testing.T) {
	data := appendAnnexBNAL(nil, decoderSPSNALWithRichVUI(t))
	info, err := NewDecoder().ParseHeadersAnnexB(data)
	if err != nil {
		t.Fatal(err)
	}
	if info.Width != 16 || info.Height != 16 || info.ProfileIDC != 66 || info.LevelIDC != 30 {
		t.Fatalf("basic stream info = %+v", info)
	}
	if info.SARNum != 4 || info.SARDen != 3 || info.VideoFormat != 5 || info.VideoFullRangeFlag != 1 {
		t.Fatalf("vui sar/video = %+v", info)
	}
	if info.ColorPrimaries != 1 || info.ColorTransfer != 1 || info.ColorMatrix != 1 {
		t.Fatalf("vui color = prim %d trc %d matrix %d", info.ColorPrimaries, info.ColorTransfer, info.ColorMatrix)
	}
	if info.ChromaSampleLocTypeTopField != 2 || info.ChromaSampleLocTypeBottomField != 3 || info.ChromaLocation != 3 {
		t.Fatalf("vui chroma location = top %d bottom %d loc %d", info.ChromaSampleLocTypeTopField, info.ChromaSampleLocTypeBottomField, info.ChromaLocation)
	}
	if info.TimingInfoPresentFlag != 1 || info.NumUnitsInTick != 1001 || info.TimeScale != 60000 || info.FixedFrameRateFlag != 1 {
		t.Fatalf("vui timing = present %d tick %d scale %d fixed %d", info.TimingInfoPresentFlag, info.NumUnitsInTick, info.TimeScale, info.FixedFrameRateFlag)
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
}

func TestParseHeadersAnnexBHigh422(t *testing.T) {
	for _, tt := range []struct {
		name string
		hex  string
	}{
		{name: "cavlc", hex: testsrc16CAVLC422AnnexBHex},
		{name: "cabac", hex: testsrc16CABAC422AnnexBHex},
	} {
		t.Run(tt.name, func(t *testing.T) {
			data := decodeHexFixture(t, tt.hex)
			dec := NewDecoder()
			info, err := dec.ParseHeadersAnnexB(data)
			if err != nil {
				t.Fatal(err)
			}
			if info.Profile != "High 4:2:2" || info.ProfileIDC != 122 || info.LevelIDC != 10 {
				t.Fatalf("profile/level = %q/%d/%d, want High 4:2:2/122/10", info.Profile, info.ProfileIDC, info.LevelIDC)
			}
			if info.Width != 16 || info.Height != 16 || info.ChromaFormatIDC != 2 || info.BitDepthLuma != 8 || info.BitDepthChroma != 8 {
				t.Fatalf("stream info = %+v", info)
			}
		})
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
	if frame.SARDen != 1 || frame.VideoFullRangeFlag != -1 || frame.ColorMatrix != 2 {
		t.Fatalf("frame vui defaults = sar %d:%d range %d matrix %d", frame.SARNum, frame.SARDen, frame.VideoFullRangeFlag, frame.ColorMatrix)
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

func TestDecodeAutoBlack16AnnexBAndAVC(t *testing.T) {
	data := decodeHexFixture(t, black16AnnexBHex)
	for _, tt := range []struct {
		name string
		data []byte
	}{
		{name: "annexb", data: data},
		{name: "avc4", data: annexBToAVC(t, data, 4)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			frame, err := NewDecoder().Decode(tt.data)
			if err != nil {
				t.Fatal(err)
			}
			assertFrameMD5Strings(t, []*Frame{frame}, []string{"8aaefe0adcea094cfb5161a060bab4e2"})
		})
	}
}

func TestDecodeAutoAVCConfigurationPacket(t *testing.T) {
	data := decodeHexFixture(t, black16AnnexBHex)
	config, packet := annexBToAVCConfigAndPacket(t, data, 4)
	dec := NewDecoder()
	frames, err := dec.DecodeFrames(config)
	if err != nil {
		t.Fatal(err)
	}
	if len(frames) != 0 {
		t.Fatalf("config frames = %d, want 0", len(frames))
	}
	frames, err = dec.DecodeFrames(packet)
	if err != nil {
		t.Fatal(err)
	}
	assertFrameMD5Strings(t, frames, []string{"8aaefe0adcea094cfb5161a060bab4e2"})
}

func TestDecodeAutoConfiguredLength4SwitchesToAnnexB(t *testing.T) {
	data := decodeHexFixture(t, black16AnnexBHex)
	config, _ := annexBToAVCConfigAndPacket(t, data, 4)
	dec := NewDecoder()
	if frames, err := dec.DecodeFrames(config); err != nil || len(frames) != 0 {
		t.Fatalf("config decode frames=%d err=%v", len(frames), err)
	}
	frames, err := dec.DecodeFrames(data)
	if err != nil {
		t.Fatal(err)
	}
	assertFrameMD5Strings(t, frames, []string{"8aaefe0adcea094cfb5161a060bab4e2"})
}

func TestDecodePacketFramesNewExtradataAVC(t *testing.T) {
	data := decodeHexFixture(t, black16AnnexBHex)
	config, packet := annexBToAVCConfigAndPacket(t, data, 4)
	dec := NewDecoder()
	frames, err := dec.DecodePacketFrames(Packet{
		Data: packet,
		SideData: []PacketSideData{
			{Type: PacketSideDataType(99), Data: []byte("ignored")},
			{Type: PacketSideDataNewExtradata, Data: config},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	assertFrameMD5Strings(t, frames, []string{"8aaefe0adcea094cfb5161a060bab4e2"})
}

func TestDecodePacketFramesRepeatedNewExtradataDoesNotResetDPB(t *testing.T) {
	data := decodeHexFixture(t, black16IPAnnexBHex)
	config, samples := annexBToAVCConfigAndSamples(t, data, 4)
	dec := NewDecoder()
	var frames []*Frame
	for i, sample := range samples {
		out, err := dec.DecodePacketFrames(Packet{
			Data:     sample,
			SideData: []PacketSideData{{Type: PacketSideDataNewExtradata, Data: config}},
		})
		if err != nil {
			t.Fatalf("sample[%d]: %v", i, err)
		}
		frames = append(frames, out...)
	}
	assertFrameMD5Strings(t, frames, []string{
		"8aaefe0adcea094cfb5161a060bab4e2",
		"8aaefe0adcea094cfb5161a060bab4e2",
	})
}

func TestDecodePacketFramesNewExtradataAnnexB(t *testing.T) {
	data := decodeHexFixture(t, black16AnnexBHex)
	extradata, packet := annexBParameterSetsAndPacket(t, data)
	dec := NewDecoder()
	frame, err := dec.DecodePacket(Packet{
		Data:     packet,
		SideData: []PacketSideData{{Type: PacketSideDataNewExtradata, Data: extradata}},
	})
	if err != nil {
		t.Fatal(err)
	}
	assertFrameMD5Strings(t, []*Frame{frame}, []string{"8aaefe0adcea094cfb5161a060bab4e2"})
}

func TestDecodePacketFramesPacketSideDataMapsToFrame(t *testing.T) {
	captions := []byte{0x01, 0x02, 0x03}
	frame, err := NewDecoder().DecodePacket(Packet{
		Data: decodeHexFixture(t, black16AnnexBHex),
		SideData: []PacketSideData{
			{Type: PacketSideDataA53ClosedCaptions, Data: captions},
			{Type: PacketSideDataA53ClosedCaptions, Data: []byte{0xff}},
			{Type: PacketSideDataActiveFormat, Data: []byte{0x0a}},
			{Type: PacketSideDataS12MTimecode, Data: []byte{
				0x02, 0x00, 0x00, 0x00,
				0x44, 0x33, 0x22, 0x11,
				0x88, 0x77, 0x66, 0x55,
				0x00, 0x00, 0x00, 0x00,
			}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	assertFrameMD5Strings(t, []*Frame{frame}, []string{"8aaefe0adcea094cfb5161a060bab4e2"})
	captions[0] = 0xee
	if got, want := frame.SideData.A53ClosedCaptions, []byte{0x01, 0x02, 0x03}; !bytes.Equal(got, want) {
		t.Fatalf("packet a53 captions = %x, want %x", got, want)
	}
	if frame.SideData.ActiveFormat == nil || frame.SideData.ActiveFormat.Description != 0x0a {
		t.Fatalf("packet active format = %+v", frame.SideData.ActiveFormat)
	}
	if got, want := frame.SideData.S12MTimecodes, []uint32{0x11223344, 0x55667788}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("packet s12m timecodes = %08x, want %08x", got, want)
	}
}

func TestDecodePacketFramesGlobalPacketSideDataMapsToFrame(t *testing.T) {
	primaries := [3][2]uint16{{30000, 35000}, {10000, 20000}, {15000, 25000}}
	white := [2]uint16{15635, 16450}
	matrix := [9]int32{65536, 0, 0, 0, -65536, 0, 123, 456, 1 << 30}
	iccProfile := []byte{0x00, 0x00, 0x02, 0x10, 'a', 'c', 's', 'p'}
	dynamicHDR10Plus := []byte{0x4c, 0x01, 0x02, 0x03, 0x80}
	lcevc := []byte{0x7e, 0x01, 0x00, 0x03, 0x02, 0x7f}
	frame, err := NewDecoder().DecodePacket(Packet{
		Data: decodeHexFixture(t, black16AnnexBHex),
		SideData: []PacketSideData{
			{Type: PacketSideDataDisplayMatrix, Data: decoderPacketDisplayMatrixSideData(matrix)},
			{Type: PacketSideDataStereo3D, Data: decoderPacketStereo3DSideData(
				int32(Stereo3DTypeTopBottom), 1, int32(Stereo3DViewLeft), int32(Stereo3DPrimaryEyeRight), 65000,
				Rational{Num: -1, Den: 2}, Rational{Num: 90, Den: 1},
			)},
			{Type: PacketSideDataSpherical, Data: decoderPacketSphericalSideData(
				int32(SphericalProjectionEquirectangularTile), 1<<16, -(2 << 16), 3<<16,
				1000, 2000, 3000, 4000, 12,
			)},
			{Type: PacketSideDataAmbientViewingEnvironment, Data: decoderPacketAmbientViewingSideData(12345, 25000, 16667)},
			{Type: PacketSideDataMasteringDisplayMetadata, Data: decoderPacketMasteringDisplaySideData(primaries, white, 10000000, 100, true, true)},
			{Type: PacketSideDataContentLightLevel, Data: decoderPacketContentLightSideData(4000, 300)},
			{Type: PacketSideDataICCProfile, Data: iccProfile},
			{Type: PacketSideDataDynamicHDR10Plus, Data: dynamicHDR10Plus},
			{Type: PacketSideDataLCEVC, Data: lcevc},
			{Type: PacketSideData3DReferenceDisplays, Data: decoderPacketReferenceDisplaysSideData(
				12, true, 9,
				[]ReferenceDisplay{{
					LeftViewID:                 3,
					RightViewID:                4,
					ExponentRefDisplayWidth:    2,
					MantissaRefDisplayWidth:    33,
					ExponentRefViewingDistance: 5,
					MantissaRefViewingDistance: 44,
					AdditionalShiftPresentFlag: true,
					NumSampleShift:             -7,
				}},
			)},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	assertFrameMD5Strings(t, []*Frame{frame}, []string{"8aaefe0adcea094cfb5161a060bab4e2"})
	side := frame.SideData
	if side.DisplayOrientation == nil || side.DisplayOrientation.Matrix != matrix {
		t.Fatalf("packet display matrix = %+v", side.DisplayOrientation)
	}
	if side.Stereo3D == nil ||
		side.Stereo3D.Type != Stereo3DTypeTopBottom ||
		!side.Stereo3D.Inverted ||
		side.Stereo3D.View != Stereo3DViewLeft ||
		side.Stereo3D.PrimaryEye != Stereo3DPrimaryEyeRight ||
		side.Stereo3D.Baseline != 65000 ||
		side.Stereo3D.HorizontalDisparityAdjustment != (Rational{Num: -1, Den: 2}) ||
		side.Stereo3D.HorizontalFieldOfView != (Rational{Num: 90, Den: 1}) ||
		side.Stereo3D.StereoMode != "bottom_top" {
		t.Fatalf("packet stereo3d = %+v", side.Stereo3D)
	}
	if side.Spherical == nil ||
		side.Spherical.Projection != SphericalProjectionEquirectangularTile ||
		side.Spherical.Yaw != 1<<16 ||
		side.Spherical.Pitch != -(2<<16) ||
		side.Spherical.Roll != 3<<16 ||
		side.Spherical.BoundLeft != 1000 ||
		side.Spherical.BoundTop != 2000 ||
		side.Spherical.BoundRight != 3000 ||
		side.Spherical.BoundBottom != 4000 ||
		side.Spherical.Padding != 12 {
		t.Fatalf("packet spherical = %+v", side.Spherical)
	}
	if side.AmbientViewing == nil || side.AmbientViewing.AmbientIlluminance != 12345 ||
		side.AmbientViewing.AmbientLightX != 25000 || side.AmbientViewing.AmbientLightY != 16667 {
		t.Fatalf("packet ambient viewing = %+v", side.AmbientViewing)
	}
	if side.MasteringDisplay == nil ||
		side.MasteringDisplay.DisplayPrimaries != primaries ||
		side.MasteringDisplay.WhitePoint != white ||
		side.MasteringDisplay.MaxLuminance != 10000000 ||
		side.MasteringDisplay.MinLuminance != 100 ||
		!side.MasteringDisplay.HasPrimaries || !side.MasteringDisplay.HasLuminance {
		t.Fatalf("packet mastering display = %+v", side.MasteringDisplay)
	}
	if side.ContentLight == nil || side.ContentLight.MaxContentLightLevel != 4000 ||
		side.ContentLight.MaxPicAverageLightLevel != 300 {
		t.Fatalf("packet content light = %+v", side.ContentLight)
	}
	iccProfile[0] = 0xff
	if got, want := side.ICCProfile, []byte{0x00, 0x00, 0x02, 0x10, 'a', 'c', 's', 'p'}; !bytes.Equal(got, want) {
		t.Fatalf("packet icc profile = %x, want %x", got, want)
	}
	dynamicHDR10Plus[0] = 0xff
	if got, want := side.DynamicHDR10Plus, []byte{0x4c, 0x01, 0x02, 0x03, 0x80}; !bytes.Equal(got, want) {
		t.Fatalf("packet dynamic hdr10+ = %x, want %x", got, want)
	}
	lcevc[0] = 0xff
	if got, want := side.LCEVC, []byte{0x7e, 0x01, 0x00, 0x03, 0x02, 0x7f}; !bytes.Equal(got, want) {
		t.Fatalf("packet lcevc = %x, want %x", got, want)
	}
	if side.ReferenceDisplays == nil ||
		side.ReferenceDisplays.PrecRefDisplayWidth != 12 ||
		!side.ReferenceDisplays.RefViewingDistanceFlag ||
		side.ReferenceDisplays.PrecRefViewingDist != 9 ||
		len(side.ReferenceDisplays.Displays) != 1 {
		t.Fatalf("packet reference displays = %+v", side.ReferenceDisplays)
	}
	if display := side.ReferenceDisplays.Displays[0]; display.LeftViewID != 3 ||
		display.RightViewID != 4 ||
		display.ExponentRefDisplayWidth != 2 ||
		display.MantissaRefDisplayWidth != 33 ||
		display.ExponentRefViewingDistance != 5 ||
		display.MantissaRefViewingDistance != 44 ||
		!display.AdditionalShiftPresentFlag ||
		display.NumSampleShift != -7 {
		t.Fatalf("packet reference display[0] = %+v", display)
	}
}

func TestDecodePacketFramesPacketDisplayAndStereoWinPublicFirstSideData(t *testing.T) {
	matrix := [9]int32{0, 65536, 0, -65536, 0, 0, 0, 0, 1 << 30}
	data := prependAnnexBNAL(decodeHexFixture(t, black16AnnexBHex), decoderTestSEINAL(
		decoderSEITestMessage{typ: decoderSEITypeDisplayOrientation, payload: decoderSEIDisplayOrientationPayload()},
		decoderSEITestMessage{typ: decoderSEITypeFramePackingArrangement, payload: decoderSEIFramePackingPayload()},
	))
	frame, err := NewDecoder().DecodePacket(Packet{
		Data: data,
		SideData: []PacketSideData{
			{Type: PacketSideDataDisplayMatrix, Data: decoderPacketDisplayMatrixSideData(matrix)},
			{Type: PacketSideDataStereo3D, Data: decoderPacketStereo3DSideData(
				int32(Stereo3DTypeColumns), 0, int32(Stereo3DViewRight), int32(Stereo3DPrimaryEyeLeft), 32000,
				Rational{Num: 0, Den: 1}, Rational{Num: 100, Den: 1},
			)},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	assertFrameMD5Strings(t, []*Frame{frame}, []string{"8aaefe0adcea094cfb5161a060bab4e2"})
	if frame.SideData.DisplayOrientation == nil || frame.SideData.DisplayOrientation.Matrix != matrix {
		t.Fatalf("display matrix = %+v", frame.SideData.DisplayOrientation)
	}
	if frame.SideData.Stereo3D == nil ||
		frame.SideData.Stereo3D.Type != Stereo3DTypeColumns ||
		frame.SideData.Stereo3D.View != Stereo3DViewRight ||
		frame.SideData.Stereo3D.PrimaryEye != Stereo3DPrimaryEyeLeft ||
		frame.SideData.Stereo3D.StereoMode != "col_interleaved_lr" {
		t.Fatalf("stereo3d = %+v", frame.SideData.Stereo3D)
	}
}

func TestPacketGlobalSideDataRejectsNonExactRationals(t *testing.T) {
	ambient := decoderPacketAmbientViewingSideData(12345, 25000, 16667)
	binary.LittleEndian.PutUint32(ambient[4:8], 7)
	mastering := decoderPacketMasteringDisplaySideData(
		[3][2]uint16{{30000, 35000}, {10000, 20000}, {15000, 25000}},
		[2]uint16{15635, 16450},
		10000000,
		100,
		true,
		true,
	)
	binary.LittleEndian.PutUint32(mastering[4:8], 7)

	frame, err := NewDecoder().DecodePacket(Packet{
		Data: decodeHexFixture(t, black16AnnexBHex),
		SideData: []PacketSideData{
			{Type: PacketSideDataAmbientViewingEnvironment, Data: ambient},
			{Type: PacketSideDataMasteringDisplayMetadata, Data: mastering},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if frame.SideData.AmbientViewing != nil {
		t.Fatalf("ambient side data with non-exact rational was accepted: %+v", frame.SideData.AmbientViewing)
	}
	if frame.SideData.MasteringDisplay != nil {
		t.Fatalf("mastering side data with non-exact rational was accepted: %+v", frame.SideData.MasteringDisplay)
	}
}

func TestPacketReferenceDisplaysRejectsInvalidNativeLayout(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "offset beyond payload",
			data: []byte{
				12, 1, 9, 1,
				0, 0, 0, 0,
				0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
				12, 0, 0, 0, 0, 0, 0, 0,
			},
		},
		{
			name: "entry extent overflow",
			data: func() []byte {
				out := make([]byte, 36)
				out[0], out[1], out[2], out[3] = 12, 1, 9, 2
				binary.LittleEndian.PutUint64(out[8:16], 24)
				binary.LittleEndian.PutUint64(out[16:24], ^uint64(0)/2+1)
				return out
			}(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			frame, err := NewDecoder().DecodePacket(Packet{
				Data: decodeHexFixture(t, black16AnnexBHex),
				SideData: []PacketSideData{{
					Type: PacketSideData3DReferenceDisplays,
					Data: tt.data,
				}},
			})
			if err != nil {
				t.Fatal(err)
			}
			if frame.SideData.ReferenceDisplays != nil {
				t.Fatalf("reference displays with invalid native layout accepted: %+v", frame.SideData.ReferenceDisplays)
			}
		})
	}
}

func TestDecodeFrameS12MTimecodePackingMatchesFFmpegBranches(t *testing.T) {
	for _, tt := range []struct {
		name                      string
		numUnitsInTick, timeScale uint32
		drop                      bool
		frame                     uint32
		want                      uint32
	}{
		{
			name:           "ntsc-drop-under-30fps",
			numUnitsInTick: 1001,
			timeScale:      60000,
			drop:           true,
			frame:          12,
			want:           0x52345607,
		},
		{
			name:           "50fps-odd-frame-uses-field-mark-bit",
			numUnitsInTick: 1,
			timeScale:      100,
			frame:          13,
			want:           0x06345687,
		},
		{
			name:           "60fps-odd-frame-uses-frame-mark-bit",
			numUnitsInTick: 1,
			timeScale:      120,
			frame:          13,
			want:           0x06b45607,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := decodePictureTimingS12M(t, tt.numUnitsInTick, tt.timeScale, tt.drop, tt.frame)
			if got != tt.want {
				t.Fatalf("smpte = %#08x, want %#08x", got, tt.want)
			}
		})
	}
}

func TestDecodePacketFramesPacketSideDataMergesWithSEIInFFmpegOrder(t *testing.T) {
	base := replaceAnnexBSPS(t, decodeHexFixture(t, black16AnnexBHex), decoderSPSNALWithPicStructVUI())
	data := prependAnnexBNAL(base, decoderTestSEINAL(
		decoderSEITestMessage{typ: decoderSEITypeUserDataRegisteredITUTT35, payload: decoderSEIRegisteredAFDPayload(0x0e)},
		decoderSEITestMessage{typ: decoderSEITypeUserDataRegisteredITUTT35, payload: decoderSEIRegisteredA53Payload([]byte{0x04, 0x05, 0x06})},
		decoderSEITestMessage{typ: decoderSEITypePicTiming, payload: decoderSEIPictureTimingTimecodePayload()},
	))
	frame, err := NewDecoder().DecodePacket(Packet{
		Data: data,
		SideData: []PacketSideData{
			{Type: PacketSideDataA53ClosedCaptions, Data: []byte{0xaa}},
			{Type: PacketSideDataActiveFormat, Data: []byte{0x01}},
			{Type: PacketSideDataS12MTimecode, Data: []byte{
				0x01, 0x00, 0x00, 0x00,
				0xef, 0xbe, 0xad, 0xde,
				0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00,
			}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	assertFrameMD5Strings(t, []*Frame{frame}, []string{"8aaefe0adcea094cfb5161a060bab4e2"})
	if got, want := frame.SideData.A53ClosedCaptions, []byte{0xaa}; !bytes.Equal(got, want) {
		t.Fatalf("frame a53 captions = %x, want %x", got, want)
	}
	if frame.SideData.ActiveFormat == nil || frame.SideData.ActiveFormat.Description != 0x01 {
		t.Fatalf("frame active format = %+v", frame.SideData.ActiveFormat)
	}
	if got, want := frame.SideData.S12MTimecodes, []uint32{0x40345607}; len(got) != len(want) || got[0] != want[0] {
		t.Fatalf("frame s12m timecodes = %08x, want %08x", got, want)
	}
}

func TestDecodePacketFramesGlobalPacketSideDataDoesNotReplaceCodedSEI(t *testing.T) {
	data := prependAnnexBNAL(decodeHexFixture(t, black16AnnexBHex), decoderTestSEINAL(
		decoderSEITestMessage{typ: decoderSEITypeUserDataRegisteredITUTT35, payload: decoderSEIRegisteredLCEVCPayload([]byte{0x7e, 0x10, 0x00, 0x03, 0x11})},
		decoderSEITestMessage{typ: decoderSEITypeAmbientViewingEnvironment, payload: decoderSEIAmbientViewingPayload()},
		decoderSEITestMessage{typ: decoderSEITypeMasteringDisplayColourVolume, payload: decoderSEIMasteringDisplayPayload()},
		decoderSEITestMessage{typ: decoderSEITypeContentLightLevelInfo, payload: []byte{0x03, 0xe8, 0x00, 0xfa}},
	))
	frame, err := NewDecoder().DecodePacket(Packet{
		Data: data,
		SideData: []PacketSideData{
			{Type: PacketSideDataAmbientViewingEnvironment, Data: decoderPacketAmbientViewingSideData(1, 2, 3)},
			{Type: PacketSideDataMasteringDisplayMetadata, Data: decoderPacketMasteringDisplaySideData(
				[3][2]uint16{{10, 20}, {30, 40}, {50, 60}}, [2]uint16{70, 80}, 900000, 90, true, true,
			)},
			{Type: PacketSideDataContentLightLevel, Data: decoderPacketContentLightSideData(9, 8)},
			{Type: PacketSideDataLCEVC, Data: []byte{0x01, 0x02, 0x03}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	assertFrameMD5Strings(t, []*Frame{frame}, []string{"8aaefe0adcea094cfb5161a060bab4e2"})
	side := frame.SideData
	if side.AmbientViewing == nil || side.AmbientViewing.AmbientIlluminance != 12345 ||
		side.AmbientViewing.AmbientLightX != 25000 || side.AmbientViewing.AmbientLightY != 16667 {
		t.Fatalf("coded ambient viewing = %+v", side.AmbientViewing)
	}
	if side.MasteringDisplay == nil ||
		side.MasteringDisplay.DisplayPrimaries != [3][2]uint16{{30000, 35000}, {10000, 20000}, {15000, 25000}} ||
		side.MasteringDisplay.WhitePoint != [2]uint16{15635, 16450} ||
		side.MasteringDisplay.MaxLuminance != 10000000 ||
		side.MasteringDisplay.MinLuminance != 100 {
		t.Fatalf("coded mastering display = %+v", side.MasteringDisplay)
	}
	if side.ContentLight == nil || side.ContentLight.MaxContentLightLevel != 1000 ||
		side.ContentLight.MaxPicAverageLightLevel != 250 {
		t.Fatalf("coded content light = %+v", side.ContentLight)
	}
	if got, want := side.LCEVC, []byte{0x7e, 0x10, 0x00, 0x03, 0x11}; !bytes.Equal(got, want) {
		t.Fatalf("coded lcevc = %x, want %x", got, want)
	}
}

func TestDecodeFrameSideDataFromLeadingSEI(t *testing.T) {
	data := prependAnnexBNAL(decodeHexFixture(t, black16AnnexBHex), decoderTestSEINAL(
		decoderSEITestMessage{typ: decoderSEITypeUserDataRegisteredITUTT35, payload: decoderSEIRegisteredAFDPayload(0x0e)},
		decoderSEITestMessage{typ: decoderSEITypeUserDataRegisteredITUTT35, payload: decoderSEIRegisteredA53Payload([]byte{0x04, 0x05, 0x06})},
		decoderSEITestMessage{typ: decoderSEITypeUserDataRegisteredITUTT35, payload: decoderSEIRegisteredLCEVCPayload([]byte{0x7e, 0x00, 0x00, 0x03, 0x01})},
		decoderSEITestMessage{typ: decoderSEITypeRecoveryPoint, payload: decoderSEIRecoveryPointPayload()},
		decoderSEITestMessage{typ: decoderSEITypeGreenMetadata, payload: []byte{0, 2, 0x01, 0x23, 1, 2, 3, 4}},
		decoderSEITestMessage{typ: decoderSEITypeDisplayOrientation, payload: decoderSEIDisplayOrientationPayload()},
		decoderSEITestMessage{typ: decoderSEITypeFramePackingArrangement, payload: decoderSEIFramePackingPayload()},
		decoderSEITestMessage{typ: decoderSEITypeAlternativeTransfer, payload: []byte{16}},
		decoderSEITestMessage{typ: decoderSEITypeAmbientViewingEnvironment, payload: decoderSEIAmbientViewingPayload()},
		decoderSEITestMessage{typ: decoderSEITypeFilmGrainCharacteristics, payload: decoderSEIFilmGrainPayload()},
		decoderSEITestMessage{typ: decoderSEITypeMasteringDisplayColourVolume, payload: decoderSEIMasteringDisplayPayload()},
		decoderSEITestMessage{typ: decoderSEITypeContentLightLevelInfo, payload: []byte{0x03, 0xe8, 0x00, 0xfa}},
	))
	frame, err := NewDecoder().Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	assertFrameMD5Strings(t, []*Frame{frame}, []string{"8aaefe0adcea094cfb5161a060bab4e2"})
	side := frame.SideData
	if side.X264Build != 165 || len(side.UserDataUnregistered) != 1 {
		t.Fatalf("unregistered side data = build %d count %d", side.X264Build, len(side.UserDataUnregistered))
	}
	if side.RecoveryPoint == nil || side.RecoveryPoint.RecoveryFrameCount != 4 {
		t.Fatalf("recovery point = %+v", side.RecoveryPoint)
	}
	if side.ActiveFormat == nil || side.ActiveFormat.Description != 0x0e {
		t.Fatalf("active format = %+v", side.ActiveFormat)
	}
	if got, want := side.A53ClosedCaptions, []byte{0x04, 0x05, 0x06}; !bytes.Equal(got, want) {
		t.Fatalf("a53 captions = %x, want %x", got, want)
	}
	if got, want := side.LCEVC, []byte{0x7e, 0x00, 0x00, 0x03, 0x01}; !bytes.Equal(got, want) {
		t.Fatalf("lcevc = %x, want %x", got, want)
	}
	if side.GreenMetadata == nil || side.GreenMetadata.NumSeconds != 0x0123 ||
		side.GreenMetadata.PercentIntraCodedMacroblocks != 2 {
		t.Fatalf("green metadata = %+v", side.GreenMetadata)
	}
	if side.DisplayOrientation == nil || !side.DisplayOrientation.HFlip ||
		side.DisplayOrientation.VFlip || side.DisplayOrientation.AnticlockwiseRotation != 0x4000 ||
		side.DisplayOrientation.Matrix != [9]int32{0, 65536, 0, 65536, 0, 0, 0, 0, 1 << 30} {
		t.Fatalf("display orientation = %+v", side.DisplayOrientation)
	}
	if side.FramePacking == nil || side.FramePacking.ArrangementID != 2 ||
		side.FramePacking.ArrangementType != 3 || !side.FramePacking.CurrentFrameIsFrame0Flag {
		t.Fatalf("frame packing = %+v", side.FramePacking)
	}
	if side.Stereo3D == nil || side.Stereo3D.Type != Stereo3DTypeSideBySide ||
		!side.Stereo3D.Inverted || side.Stereo3D.View != Stereo3DViewPacked ||
		side.Stereo3D.StereoMode != "right_left" {
		t.Fatalf("stereo 3d = %+v", side.Stereo3D)
	}
	if side.AlternativeTransfer == nil || side.AlternativeTransfer.PreferredTransferCharacteristics != 16 {
		t.Fatalf("alternative transfer = %+v", side.AlternativeTransfer)
	}
	if side.AmbientViewing == nil || side.AmbientViewing.AmbientIlluminance != 12345 ||
		side.AmbientViewing.AmbientLightX != 25000 || side.AmbientViewing.AmbientLightY != 16667 {
		t.Fatalf("ambient viewing = %+v", side.AmbientViewing)
	}
	fg := side.FilmGrain
	if fg == nil || fg.ModelID != 1 || !fg.SeparateColourDescriptionPresentFlag ||
		fg.BitDepthLuma != 10 || fg.BitDepthChroma != 8 || !fg.FullRange ||
		fg.ColorPrimaries != 9 || fg.TransferCharacteristics != 16 || fg.MatrixCoeffs != 9 ||
		fg.BlendingModeID != 1 || fg.Log2ScaleFactor != 7 || fg.RepetitionPeriod != 4 {
		t.Fatalf("film grain header = %+v", fg)
	}
	if fg.CompModelPresentFlag != [3]bool{true, true, false} ||
		fg.NumIntensityIntervals != [3]uint16{1, 2, 0} ||
		fg.NumModelValues != [3]uint8{2, 1, 0} {
		t.Fatalf("film grain component counts = present %+v intervals %+v values %+v",
			fg.CompModelPresentFlag, fg.NumIntensityIntervals, fg.NumModelValues)
	}
	if fg.IntensityIntervalLowerBound[0][0] != 10 || fg.IntensityIntervalUpperBound[0][0] != 20 ||
		fg.CompModelValue[0][0][0] != 3 || fg.CompModelValue[0][0][1] != -2 ||
		fg.IntensityIntervalLowerBound[1][1] != 41 || fg.IntensityIntervalUpperBound[1][1] != 60 ||
		fg.CompModelValue[1][1][0] != 5 {
		t.Fatalf("film grain component data = %+v %+v %+v", fg.IntensityIntervalLowerBound, fg.IntensityIntervalUpperBound, fg.CompModelValue)
	}
	if side.MasteringDisplay == nil ||
		side.MasteringDisplay.DisplayPrimaries != [3][2]uint16{{30000, 35000}, {10000, 20000}, {15000, 25000}} ||
		side.MasteringDisplay.WhitePoint != [2]uint16{15635, 16450} ||
		side.MasteringDisplay.MaxLuminance != 10000000 ||
		side.MasteringDisplay.MinLuminance != 100 ||
		!side.MasteringDisplay.HasPrimaries || !side.MasteringDisplay.HasLuminance {
		t.Fatalf("mastering display = %+v", side.MasteringDisplay)
	}
	if side.ContentLight == nil || side.ContentLight.MaxContentLightLevel != 1000 ||
		side.ContentLight.MaxPicAverageLightLevel != 250 {
		t.Fatalf("content light = %+v", side.ContentLight)
	}
}

func TestDecodeFrameSideDataByteSlicesAreCallerOwned(t *testing.T) {
	data := prependAnnexBNAL(decodeHexFixture(t, black16AnnexBHex), decoderTestSEINAL(
		decoderSEITestMessage{typ: decoderSEITypeUserDataRegisteredITUTT35, payload: decoderSEIRegisteredA53Payload([]byte{0x04, 0x05, 0x06})},
		decoderSEITestMessage{typ: decoderSEITypeUserDataRegisteredITUTT35, payload: decoderSEIRegisteredLCEVCPayload([]byte{0x7e, 0x00, 0x00, 0x03, 0x01})},
	))

	frame, err := NewDecoder().Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	side := frame.SideData
	if len(side.UserDataUnregistered) != 1 {
		t.Fatalf("unregistered side data count = %d, want 1", len(side.UserDataUnregistered))
	}
	wantUnregistered := append([]byte(nil), side.UserDataUnregistered[0]...)
	wantA53 := append([]byte(nil), side.A53ClosedCaptions...)
	wantLCEVC := append([]byte(nil), side.LCEVC...)
	if len(wantUnregistered) == 0 || len(wantA53) == 0 || len(wantLCEVC) == 0 {
		t.Fatalf("side data = unregistered %x a53 %x lcevc %x", wantUnregistered, wantA53, wantLCEVC)
	}

	for i := range side.UserDataUnregistered[0] {
		side.UserDataUnregistered[0][i] ^= 0xff
	}
	for i := range side.A53ClosedCaptions {
		side.A53ClosedCaptions[i] ^= 0xff
	}
	for i := range side.LCEVC {
		side.LCEVC[i] ^= 0xff
	}

	frame, err = NewDecoder().Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	side = frame.SideData
	if len(side.UserDataUnregistered) != 1 || !bytes.Equal(side.UserDataUnregistered[0], wantUnregistered) {
		t.Fatalf("unregistered after caller mutation = %x, want %x", side.UserDataUnregistered, wantUnregistered)
	}
	if !bytes.Equal(side.A53ClosedCaptions, wantA53) {
		t.Fatalf("a53 after caller mutation = %x, want %x", side.A53ClosedCaptions, wantA53)
	}
	if !bytes.Equal(side.LCEVC, wantLCEVC) {
		t.Fatalf("lcevc after caller mutation = %x, want %x", side.LCEVC, wantLCEVC)
	}
}

func TestDecodeFrameSideDataSkipsNoopDisplayMatrixAndInvalidStereo3D(t *testing.T) {
	data := prependAnnexBNAL(decodeHexFixture(t, black16AnnexBHex), decoderTestSEINAL(
		decoderSEITestMessage{typ: decoderSEITypeDisplayOrientation, payload: decoderSEIDisplayOrientationPayloadWith(0, false, false)},
		decoderSEITestMessage{typ: decoderSEITypeFramePackingArrangement, payload: decoderSEIFramePackingPayloadWith(7, 1, false, false)},
	))
	frame, err := NewDecoder().Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	assertFrameMD5Strings(t, []*Frame{frame}, []string{"8aaefe0adcea094cfb5161a060bab4e2"})
	if frame.SideData.DisplayOrientation != nil {
		t.Fatalf("display orientation = %+v, want nil for no-op transform", frame.SideData.DisplayOrientation)
	}
	if frame.SideData.Stereo3D != nil {
		t.Fatalf("stereo 3d = %+v, want nil for invalid H.264 frame packing type", frame.SideData.Stereo3D)
	}
	if frame.SideData.FramePacking == nil || frame.SideData.FramePacking.ArrangementType != 7 {
		t.Fatalf("raw frame packing = %+v", frame.SideData.FramePacking)
	}
}

func TestDecodeFrameOneShotSEISideDataIsNotRepeated(t *testing.T) {
	base := replaceAnnexBSPS(t, decodeHexFixture(t, black16IPAnnexBHex), decoderSPSNALWithPicStructVUI())
	data := prependAnnexBNAL(base, decoderTestSEINAL(
		decoderSEITestMessage{typ: decoderSEITypeUserDataRegisteredITUTT35, payload: decoderSEIRegisteredAFDPayload(0x0d)},
		decoderSEITestMessage{typ: decoderSEITypeUserDataRegisteredITUTT35, payload: decoderSEIRegisteredA53Payload([]byte{0x01, 0x02, 0x03})},
		decoderSEITestMessage{typ: decoderSEITypeUserDataRegisteredITUTT35, payload: decoderSEIRegisteredLCEVCPayload([]byte{0x7e, 0x01, 0x00, 0x03, 0x02})},
		decoderSEITestMessage{typ: decoderSEITypePicTiming, payload: decoderSEIPictureTimingTimecodePayload()},
		decoderSEITestMessage{typ: decoderSEITypeFilmGrainCharacteristics, payload: decoderSEIFilmGrainPayloadWithRepetition(0)},
	))
	frames, err := NewDecoder().DecodeAnnexBFrames(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(frames) != 2 {
		t.Fatalf("frames = %d, want 2", len(frames))
	}
	assertFrameMD5Strings(t, frames, []string{
		"8aaefe0adcea094cfb5161a060bab4e2",
		"8aaefe0adcea094cfb5161a060bab4e2",
	})

	first := frames[0].SideData
	if first.ActiveFormat == nil || first.ActiveFormat.Description != 0x0d {
		t.Fatalf("first active format = %+v", first.ActiveFormat)
	}
	if got, want := first.A53ClosedCaptions, []byte{0x01, 0x02, 0x03}; !bytes.Equal(got, want) {
		t.Fatalf("first a53 captions = %x, want %x", got, want)
	}
	if got, want := first.LCEVC, []byte{0x7e, 0x01, 0x00, 0x03, 0x02}; !bytes.Equal(got, want) {
		t.Fatalf("first lcevc = %x, want %x", got, want)
	}
	if len(first.UserDataUnregistered) != 1 || first.X264Build != 165 {
		t.Fatalf("first unregistered = build %d count %d", first.X264Build, len(first.UserDataUnregistered))
	}
	if first.FilmGrain == nil || first.FilmGrain.RepetitionPeriod != 0 {
		t.Fatalf("first film grain = %+v", first.FilmGrain)
	}
	if got, want := first.S12MTimecodes, []uint32{0x40345607}; len(got) != len(want) || got[0] != want[0] {
		t.Fatalf("first s12m timecodes = %08x, want %08x", got, want)
	}

	second := frames[1].SideData
	if second.ActiveFormat != nil || len(second.A53ClosedCaptions) != 0 ||
		len(second.LCEVC) != 0 || len(second.UserDataUnregistered) != 0 ||
		len(second.S12MTimecodes) != 0 || second.FilmGrain != nil {
		t.Fatalf("second repeated one-shot side data = %+v", second)
	}
}

func TestDecodeFrameKeyFrameFlags(t *testing.T) {
	frames, err := NewDecoder().DecodeAnnexBFrames(decodeHexFixture(t, black16IPAnnexBHex))
	if err != nil {
		t.Fatal(err)
	}
	if len(frames) != 2 {
		t.Fatalf("frames = %d, want 2", len(frames))
	}
	if !frames[0].KeyFrame || frames[1].KeyFrame {
		t.Fatalf("key frames = %t/%t, want true/false", frames[0].KeyFrame, frames[1].KeyFrame)
	}
}

func TestDecodeFrameRecoveryPointZeroMarksKeyFrame(t *testing.T) {
	frames := decodeConfiguredIPWithRecoveryPoint(t, 0)
	if !frames[0].KeyFrame || !frames[1].KeyFrame {
		t.Fatalf("key frames = %t/%t, want true/true", frames[0].KeyFrame, frames[1].KeyFrame)
	}
	if frames[1].SideData.RecoveryPoint == nil || frames[1].SideData.RecoveryPoint.RecoveryFrameCount != 0 {
		t.Fatalf("second recovery point = %+v", frames[1].SideData.RecoveryPoint)
	}
}

func TestDecodeFrameRecoveryPointNonZeroDoesNotMarkImmediateKeyFrame(t *testing.T) {
	frames := decodeConfiguredIPWithRecoveryPoint(t, 4)
	if !frames[0].KeyFrame || frames[1].KeyFrame {
		t.Fatalf("key frames = %t/%t, want true/false", frames[0].KeyFrame, frames[1].KeyFrame)
	}
	if frames[1].SideData.RecoveryPoint == nil || frames[1].SideData.RecoveryPoint.RecoveryFrameCount != 4 {
		t.Fatalf("second recovery point = %+v", frames[1].SideData.RecoveryPoint)
	}
}

func TestDecodeFrameTimingFromPictureTimingSEI(t *testing.T) {
	base := replaceAnnexBSPS(t, decodeHexFixture(t, black16AnnexBHex), decoderSPSNALWithPicStructVUI())
	for _, tt := range []struct {
		name          string
		payload       []byte
		picStruct     int32
		repeatPict    int
		interlaced    bool
		topFieldFirst bool
	}{
		{
			name:          "top-bottom-uses-initial-prev-interlaced",
			payload:       decoderSEIPictureTimingPayload(decoderSEIPicStructTopBottom),
			picStruct:     decoderSEIPicStructTopBottom,
			interlaced:    true,
			topFieldFirst: true,
		},
		{
			name:       "top-field",
			payload:    decoderSEIPictureTimingPayload(decoderSEIPicStructTopField),
			picStruct:  decoderSEIPicStructTopField,
			interlaced: true,
		},
		{
			name:          "top-bottom-top-repeat",
			payload:       decoderSEIPictureTimingPayload(decoderSEIPicStructTopBottomTop),
			picStruct:     decoderSEIPicStructTopBottomTop,
			repeatPict:    1,
			topFieldFirst: true,
		},
		{
			name:       "frame-doubling",
			payload:    decoderSEIPictureTimingPayload(decoderSEIPicStructFrameDoubling),
			picStruct:  decoderSEIPicStructFrameDoubling,
			repeatPict: 2,
		},
		{
			name:       "frame-tripling",
			payload:    decoderSEIPictureTimingPayload(decoderSEIPicStructFrameTripling),
			picStruct:  decoderSEIPicStructFrameTripling,
			repeatPict: 4,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			data := prependAnnexBNAL(base, decoderTestSEINAL(decoderSEITestMessage{
				typ:     decoderSEITypePicTiming,
				payload: tt.payload,
			}))
			frame, err := NewDecoder().Decode(data)
			if err != nil {
				t.Fatal(err)
			}
			assertFrameMD5Strings(t, []*Frame{frame}, []string{"8aaefe0adcea094cfb5161a060bab4e2"})
			if frame.RepeatPict != tt.repeatPict || frame.InterlacedFrame != tt.interlaced ||
				frame.TopFieldFirst != tt.topFieldFirst {
				t.Fatalf("frame timing = repeat %d interlaced %t top-first %t",
					frame.RepeatPict, frame.InterlacedFrame, frame.TopFieldFirst)
			}
			if frame.SideData.PictureTiming == nil {
				t.Fatalf("missing picture timing side data")
			}
			if frame.SideData.PictureTiming.PicStruct != tt.picStruct ||
				frame.SideData.PictureTiming.CTType != 0 ||
				len(frame.SideData.PictureTiming.Timecode) != 0 {
				t.Fatalf("picture timing = %+v", frame.SideData.PictureTiming)
			}
		})
	}
}

func TestDecodeFrameS12MTimecodeFromPictureTimingSEI(t *testing.T) {
	base := replaceAnnexBSPS(t, decodeHexFixture(t, black16AnnexBHex), decoderSPSNALWithPicStructVUI())
	data := prependAnnexBNAL(base, decoderTestSEINAL(decoderSEITestMessage{
		typ:     decoderSEITypePicTiming,
		payload: decoderSEIPictureTimingTimecodePayload(),
	}))
	frame, err := NewDecoder().Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	assertFrameMD5Strings(t, []*Frame{frame}, []string{"8aaefe0adcea094cfb5161a060bab4e2"})
	if got, want := frame.SideData.S12MTimecodes, []uint32{0x40345607}; len(got) != len(want) || got[0] != want[0] {
		t.Fatalf("s12m timecodes = %08x, want %08x", got, want)
	}
	pt := frame.SideData.PictureTiming
	if pt == nil || len(pt.Timecode) != 1 {
		t.Fatalf("picture timing = %+v", pt)
	}
	tc := pt.Timecode[0]
	if pt.PicStruct != decoderSEIPicStructFrame || pt.CTType != 1<<2 ||
		!tc.Full || !tc.DropFrame || tc.Frame != 0 || tc.Seconds != 34 ||
		tc.Minutes != 56 || tc.Hours != 7 {
		t.Fatalf("picture timing timecode = %+v %+v", pt, tc)
	}
}

func decodePictureTimingS12M(t *testing.T, numUnitsInTick uint32, timeScale uint32, drop bool, frameNum uint32) uint32 {
	t.Helper()

	base := replaceAnnexBSPS(t, decodeHexFixture(t, black16AnnexBHex), decoderSPSNALWithPicStructVUITiming(numUnitsInTick, timeScale))
	data := prependAnnexBNAL(base, decoderTestSEINAL(decoderSEITestMessage{
		typ:     decoderSEITypePicTiming,
		payload: decoderSEIPictureTimingTimecodePayloadWithFrame(drop, frameNum),
	}))
	frame, err := NewDecoder().Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(frame.SideData.S12MTimecodes) != 1 {
		t.Fatalf("s12m timecodes = %08x, want one value", frame.SideData.S12MTimecodes)
	}
	return frame.SideData.S12MTimecodes[0]
}

func TestS12MTimecodePackingMatchesNativeFFmpegOracle(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native FFmpeg timecode oracle")
	}
	cc := os.Getenv("CC")
	if cc == "" {
		cc = "cc"
	}
	if _, err := exec.LookPath(cc); err != nil {
		t.Skip("C compiler not available")
	}
	dir := t.TempDir()
	src := filepath.Join(dir, "timecode_oracle.c")
	bin := filepath.Join(dir, "timecode_oracle")
	if err := os.WriteFile(src, []byte(timecodeOracleC), 0o600); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command(cc, "-std=c99", "-Wall", "-Wextra", src, "-o", bin).CombinedOutput(); err != nil {
		t.Fatalf("compile timecode oracle: %v\n%s", err, out)
	}
	out, err := exec.Command(bin).Output()
	if err != nil {
		t.Fatalf("run timecode oracle: %v", err)
	}
	got := strings.TrimSpace(string(out))
	want := strings.Join([]string{
		fmt.Sprintf("%08x", decodePictureTimingS12M(t, 1001, 60000, true, 12)),
		fmt.Sprintf("%08x", decodePictureTimingS12M(t, 1, 100, false, 13)),
		fmt.Sprintf("%08x", decodePictureTimingS12M(t, 1, 120, false, 13)),
	}, "\n")
	if got != want {
		t.Fatalf("oracle mismatch\nC:\n%s\nGo:\n%s", got, want)
	}
}

func TestPacketGlobalSideDataLayoutMatchesNativeFFmpegOracle(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native FFmpeg packet side-data oracle")
	}
	cc := os.Getenv("CC")
	if cc == "" {
		cc = "cc"
	}
	if _, err := exec.LookPath(cc); err != nil {
		t.Skip("C compiler not available")
	}
	root := decoderRepoRoot(t)
	upstream := filepath.Join(root, ".upstream", "ffmpeg-n8.0.1")
	if _, err := os.Stat(filepath.Join(upstream, "libavcodec", "packet.h")); err != nil {
		t.Skipf("pinned upstream cache not available: %v", err)
	}
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "libavutil"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "libavutil", "avconfig.h"), []byte(strings.Join([]string{
		"#define AV_HAVE_BIGENDIAN 0",
		"#define AV_HAVE_FAST_UNALIGNED 1",
		"",
	}, "\n")), 0o600); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(dir, "packet_side_data_oracle.c")
	bin := filepath.Join(dir, "packet_side_data_oracle")
	if err := os.WriteFile(src, []byte(packetGlobalSideDataOracleC), 0o600); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command(cc, "-std=c99", "-Wall", "-Wextra", "-I"+dir, "-I"+upstream, src, "-o", bin).CombinedOutput(); err != nil {
		t.Fatalf("compile packet side-data oracle: %v\n%s", err, out)
	}
	out, err := exec.Command(bin).Output()
	if err != nil {
		t.Fatalf("run packet side-data oracle: %v", err)
	}
	got := strings.TrimSpace(string(out))
	want := strings.Join([]string{
		fmt.Sprintf("%d %d %d %d %d %d %d %d %d %d", PacketSideDataDisplayMatrix, PacketSideDataStereo3D, PacketSideDataMasteringDisplayMetadata, PacketSideDataSpherical, PacketSideDataContentLightLevel, PacketSideDataICCProfile, PacketSideDataDynamicHDR10Plus, PacketSideDataAmbientViewingEnvironment, PacketSideDataLCEVC, PacketSideData3DReferenceDisplays),
		"8 8 0 4 24 88 0 48 64 72 80 84",
		"36 36 0 4 8 12 16 20 28",
		"0 1 2 3 4 5 6 7 8 0 1 2 3 1",
		"36 0 4 8 12 16 20 24 28 32",
		"0 1 2 3 4 5 6",
		"24 0 1 2 3 8 16",
		"12 0 2 4 5 6 7 8 10",
	}, "\n")
	if got != want {
		t.Fatalf("packet side-data oracle mismatch\nC:\n%s\nGo:\n%s", got, want)
	}
}

func decoderRepoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(wd, ".upstream", "ffmpeg-n8.0.1")); err == nil {
			return wd
		}
		parent := filepath.Dir(wd)
		if parent == wd {
			t.Skip("repo root with pinned upstream cache not found")
		}
		wd = parent
	}
}

const packetGlobalSideDataOracleC = `
#include <stddef.h>
#include <stdio.h>

#include "libavcodec/packet.h"
#include "libavutil/ambient_viewing_environment.h"
#include "libavutil/display.h"
#include "libavutil/mastering_display_metadata.h"
#include "libavutil/spherical.h"
#include "libavutil/stereo3d.h"
#include "libavutil/tdrdi.h"

int main(void)
{
    printf("%d %d %d %d %d %d %d %d %d %d\n",
           AV_PKT_DATA_DISPLAYMATRIX,
           AV_PKT_DATA_STEREO3D,
           AV_PKT_DATA_MASTERING_DISPLAY_METADATA,
           AV_PKT_DATA_SPHERICAL,
           AV_PKT_DATA_CONTENT_LIGHT_LEVEL,
           AV_PKT_DATA_ICC_PROFILE,
           AV_PKT_DATA_DYNAMIC_HDR10_PLUS,
           AV_PKT_DATA_AMBIENT_VIEWING_ENVIRONMENT,
           AV_PKT_DATA_LCEVC,
           AV_PKT_DATA_3D_REFERENCE_DISPLAYS);
    printf("%zu %zu %zu %zu %zu %zu %zu %zu %zu %zu %zu %zu\n",
           sizeof(AVRational),
           sizeof(AVContentLightMetadata),
           offsetof(AVContentLightMetadata, MaxCLL),
           offsetof(AVContentLightMetadata, MaxFALL),
           sizeof(AVAmbientViewingEnvironment),
           sizeof(AVMasteringDisplayMetadata),
           offsetof(AVMasteringDisplayMetadata, display_primaries),
           offsetof(AVMasteringDisplayMetadata, white_point),
           offsetof(AVMasteringDisplayMetadata, min_luminance),
           offsetof(AVMasteringDisplayMetadata, max_luminance),
           offsetof(AVMasteringDisplayMetadata, has_primaries),
           offsetof(AVMasteringDisplayMetadata, has_luminance));
    printf("%zu %zu %zu %zu %zu %zu %zu %zu %zu\n",
           sizeof(int32_t[9]),
           sizeof(AVStereo3D),
           offsetof(AVStereo3D, type),
           offsetof(AVStereo3D, flags),
           offsetof(AVStereo3D, view),
           offsetof(AVStereo3D, primary_eye),
           offsetof(AVStereo3D, baseline),
           offsetof(AVStereo3D, horizontal_disparity_adjustment),
           offsetof(AVStereo3D, horizontal_field_of_view));
    printf("%d %d %d %d %d %d %d %d %d %d %d %d %d %d\n",
           AV_STEREO3D_2D,
           AV_STEREO3D_SIDEBYSIDE,
           AV_STEREO3D_TOPBOTTOM,
           AV_STEREO3D_FRAMESEQUENCE,
           AV_STEREO3D_CHECKERBOARD,
           AV_STEREO3D_SIDEBYSIDE_QUINCUNX,
           AV_STEREO3D_LINES,
           AV_STEREO3D_COLUMNS,
           AV_STEREO3D_UNSPEC,
           AV_STEREO3D_VIEW_PACKED,
           AV_STEREO3D_VIEW_LEFT,
           AV_STEREO3D_VIEW_RIGHT,
           AV_STEREO3D_VIEW_UNSPEC,
           AV_STEREO3D_FLAG_INVERT);
    printf("%zu %zu %zu %zu %zu %zu %zu %zu %zu %zu\n",
           sizeof(AVSphericalMapping),
           offsetof(AVSphericalMapping, projection),
           offsetof(AVSphericalMapping, yaw),
           offsetof(AVSphericalMapping, pitch),
           offsetof(AVSphericalMapping, roll),
           offsetof(AVSphericalMapping, bound_left),
           offsetof(AVSphericalMapping, bound_top),
           offsetof(AVSphericalMapping, bound_right),
           offsetof(AVSphericalMapping, bound_bottom),
           offsetof(AVSphericalMapping, padding));
    printf("%d %d %d %d %d %d %d\n",
           AV_SPHERICAL_EQUIRECTANGULAR,
           AV_SPHERICAL_CUBEMAP,
           AV_SPHERICAL_EQUIRECTANGULAR_TILE,
           AV_SPHERICAL_HALF_EQUIRECTANGULAR,
           AV_SPHERICAL_RECTILINEAR,
           AV_SPHERICAL_FISHEYE,
           AV_SPHERICAL_PARAMETRIC_IMMERSIVE);
    printf("%zu %zu %zu %zu %zu %zu %zu\n",
           sizeof(AV3DReferenceDisplaysInfo),
           offsetof(AV3DReferenceDisplaysInfo, prec_ref_display_width),
           offsetof(AV3DReferenceDisplaysInfo, ref_viewing_distance_flag),
           offsetof(AV3DReferenceDisplaysInfo, prec_ref_viewing_dist),
           offsetof(AV3DReferenceDisplaysInfo, num_ref_displays),
           offsetof(AV3DReferenceDisplaysInfo, entries_offset),
           offsetof(AV3DReferenceDisplaysInfo, entry_size));
    printf("%zu %zu %zu %zu %zu %zu %zu %zu %zu\n",
           sizeof(AV3DReferenceDisplay),
           offsetof(AV3DReferenceDisplay, left_view_id),
           offsetof(AV3DReferenceDisplay, right_view_id),
           offsetof(AV3DReferenceDisplay, exponent_ref_display_width),
           offsetof(AV3DReferenceDisplay, mantissa_ref_display_width),
           offsetof(AV3DReferenceDisplay, exponent_ref_viewing_distance),
           offsetof(AV3DReferenceDisplay, mantissa_ref_viewing_distance),
           offsetof(AV3DReferenceDisplay, additional_shift_present_flag),
           offsetof(AV3DReferenceDisplay, num_sample_shift));
    return 0;
}
`

const timecodeOracleC = `
#include <stdint.h>
#include <stdio.h>

typedef struct AVRational {
    int num;
    int den;
} AVRational;

static int av_cmp_q(AVRational a, AVRational b)
{
    int64_t lhs = (int64_t)a.num * b.den;
    int64_t rhs = (int64_t)b.num * a.den;
    if (lhs < rhs)
        return -1;
    if (lhs > rhs)
        return 1;
    return 0;
}

static int av_clip(int a, int amin, int amax)
{
    if (a < amin)
        return amin;
    if (a > amax)
        return amax;
    return a;
}

static uint32_t av_timecode_get_smpte(AVRational rate, int drop, int hh, int mm, int ss, int ff)
{
    uint32_t tc = 0;

    if (av_cmp_q(rate, (AVRational) {30, 1}) == 1) {
        if (ff % 2 == 1) {
            if (av_cmp_q(rate, (AVRational) {50, 1}) == 0)
                tc |= (1 << 7);
            else
                tc |= (1 << 23);
        }
        ff /= 2;
    }

    hh = hh % 24;
    mm = av_clip(mm, 0, 59);
    ss = av_clip(ss, 0, 59);
    ff = ff % 40;

    tc |= drop << 30;
    tc |= (ff / 10) << 28;
    tc |= (ff % 10) << 24;
    tc |= (ss / 10) << 20;
    tc |= (ss % 10) << 16;
    tc |= (mm / 10) << 12;
    tc |= (mm % 10) << 8;
    tc |= (hh / 10) << 4;
    tc |= (hh % 10);

    return tc;
}

int main(void)
{
    printf("%08x\n", av_timecode_get_smpte((AVRational) {30000, 1001}, 1, 7, 56, 34, 12));
    printf("%08x\n", av_timecode_get_smpte((AVRational) {50, 1}, 0, 7, 56, 34, 13));
    printf("%08x\n", av_timecode_get_smpte((AVRational) {60, 1}, 0, 7, 56, 34, 13));
    return 0;
}
`

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

func TestDecodeAnnexBTestsrc16High422Frames(t *testing.T) {
	for _, tt := range []struct {
		name string
		hex  string
		want []string
	}{
		{
			name: "cavlc",
			hex:  testsrc16CAVLC422AnnexBHex,
			want: []string{
				"b37a1f7943ce6c7d9646786f348f4ce9",
				"e705648238ec1a68ce2fc83f8d1b7293",
				"13cfed6389834373ccb5b6bb61f6cf9d",
				"f0b4d1caf4e666cc4767cfe273de480e",
			},
		},
		{
			name: "cabac",
			hex:  testsrc16CABAC422AnnexBHex,
			want: []string{
				"e06b0f34fe689940304653e5c3840a53",
				"424fb373278235a5d2b0808968cb0e58",
				"b6e4d159f8c0b0bb452de55824214ac6",
				"892dfdee5dbf37558f99a6fe0c278abb",
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			frames, err := NewDecoder().DecodeAnnexBFrames(decodeHexFixture(t, tt.hex))
			if err != nil {
				t.Fatal(err)
			}
			assertHigh422FrameMD5Strings(t, frames, tt.want)
		})
	}
}

func TestDecodeAVCTestsrc16High422Frames(t *testing.T) {
	for _, tt := range []struct {
		name string
		hex  string
		want []string
	}{
		{
			name: "cavlc",
			hex:  testsrc16CAVLC422AnnexBHex,
			want: []string{
				"b37a1f7943ce6c7d9646786f348f4ce9",
				"e705648238ec1a68ce2fc83f8d1b7293",
				"13cfed6389834373ccb5b6bb61f6cf9d",
				"f0b4d1caf4e666cc4767cfe273de480e",
			},
		},
		{
			name: "cabac",
			hex:  testsrc16CABAC422AnnexBHex,
			want: []string{
				"e06b0f34fe689940304653e5c3840a53",
				"424fb373278235a5d2b0808968cb0e58",
				"b6e4d159f8c0b0bb452de55824214ac6",
				"892dfdee5dbf37558f99a6fe0c278abb",
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			data := decodeHexFixture(t, tt.hex)
			for _, nalLengthSize := range []int{2, 3, 4} {
				frames, err := NewDecoder().DecodeAVCFrames(annexBToAVC(t, data, nalLengthSize), nalLengthSize)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: %v", nalLengthSize, err)
				}
				assertHigh422FrameMD5Strings(t, frames, tt.want)
			}
		})
	}
}

func TestParseAVCDecoderConfigurationRecordCABAC(t *testing.T) {
	data := decodeHexFixture(t, testsrc16CABACAnnexBHex)
	config, packet := annexBToAVCConfigAndPacket(t, data, 3)

	dec := NewDecoder()
	cfg, err := dec.ParseAVCDecoderConfigurationRecord(config)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.NALLengthSize != 3 {
		t.Fatalf("nal length size = %d, want 3", cfg.NALLengthSize)
	}
	if cfg.StreamInfo.Profile != "Main" || cfg.StreamInfo.ProfileIDC != 77 || cfg.StreamInfo.LevelIDC != 10 {
		t.Fatalf("stream info = %+v", cfg.StreamInfo)
	}
	frames, err := dec.DecodeConfiguredAVCFrames(packet)
	if err != nil {
		t.Fatal(err)
	}
	assertFrameMD5Strings(t, frames, []string{
		"57948a884e4468c79f3291b2693263de",
		"4fb1e27b7087e9f1aa485402993ca525",
		"a7e3e74bb19403d111dd2ffdb4455102",
		"1202e58b9b15f56a341fea8787bcc769",
	})
}

func TestDecodeConfiguredAVCFramesRequiresConfiguration(t *testing.T) {
	if _, err := NewDecoder().DecodeConfiguredAVCFrames([]byte{0, 0, 1, 0x65}); err != ErrInvalidData {
		t.Fatalf("err = %v, want ErrInvalidData", err)
	}
}

func TestDecodeAVCWithConfigurationRecordRef2Frames(t *testing.T) {
	data := decodeHexFixture(t, testsrc16Ref2AnnexBHex)
	want := []string{
		"54b049d05d99dc31d270402e798d4af4",
		"681e6d4ef3058d3880346e8039e95b94",
		"ef38cc80fb47f60e38abc2502af7e5f9",
		"0cee44ff1f8279a97bc3e56e4f58f802",
	}
	for _, nalLengthSize := range []int{2, 3, 4} {
		config, packet := annexBToAVCConfigAndPacket(t, data, nalLengthSize)
		frames, err := NewDecoder().DecodeAVCFramesWithConfigurationRecord(config, packet)
		if err != nil {
			t.Fatalf("nalLengthSize=%d: %v", nalLengthSize, err)
		}
		assertFrameMD5Strings(t, frames, want)
	}
}

func TestDecodeAVCWithConfigurationRecordCABACFrames(t *testing.T) {
	data := decodeHexFixture(t, testsrc16CABACAnnexBHex)
	want := []string{
		"57948a884e4468c79f3291b2693263de",
		"4fb1e27b7087e9f1aa485402993ca525",
		"a7e3e74bb19403d111dd2ffdb4455102",
		"1202e58b9b15f56a341fea8787bcc769",
	}
	for _, nalLengthSize := range []int{2, 3, 4} {
		config, packet := annexBToAVCConfigAndPacket(t, data, nalLengthSize)
		frames, err := NewDecoder().DecodeAVCFramesWithConfigurationRecord(config, packet)
		if err != nil {
			t.Fatalf("nalLengthSize=%d: %v", nalLengthSize, err)
		}
		assertFrameMD5Strings(t, frames, want)
	}
}

func TestDecodeAVCWithConfigurationRecordHigh422Frames(t *testing.T) {
	for _, tt := range []struct {
		name string
		hex  string
		want []string
	}{
		{
			name: "cavlc",
			hex:  testsrc16CAVLC422AnnexBHex,
			want: []string{
				"b37a1f7943ce6c7d9646786f348f4ce9",
				"e705648238ec1a68ce2fc83f8d1b7293",
				"13cfed6389834373ccb5b6bb61f6cf9d",
				"f0b4d1caf4e666cc4767cfe273de480e",
			},
		},
		{
			name: "cabac",
			hex:  testsrc16CABAC422AnnexBHex,
			want: []string{
				"e06b0f34fe689940304653e5c3840a53",
				"424fb373278235a5d2b0808968cb0e58",
				"b6e4d159f8c0b0bb452de55824214ac6",
				"892dfdee5dbf37558f99a6fe0c278abb",
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			data := decodeHexFixture(t, tt.hex)
			for _, nalLengthSize := range []int{2, 3, 4} {
				config, packet := annexBToAVCConfigAndPacket(t, data, nalLengthSize)
				frames, err := NewDecoder().DecodeAVCFramesWithConfigurationRecord(config, packet)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: %v", nalLengthSize, err)
				}
				assertHigh422FrameMD5Strings(t, frames, tt.want)
			}
		})
	}
}

func TestDecodeConfiguredAVCAcrossSamplesRef2Frames(t *testing.T) {
	data := decodeHexFixture(t, testsrc16Ref2AnnexBHex)
	want := []string{
		"54b049d05d99dc31d270402e798d4af4",
		"681e6d4ef3058d3880346e8039e95b94",
		"ef38cc80fb47f60e38abc2502af7e5f9",
		"0cee44ff1f8279a97bc3e56e4f58f802",
	}
	config, samples := annexBToAVCConfigAndSamples(t, data, 4)
	if len(samples) != len(want) {
		t.Fatalf("samples = %d, want %d", len(samples), len(want))
	}

	dec := NewDecoder()
	if _, err := dec.ParseAVCDecoderConfigurationRecord(config); err != nil {
		t.Fatal(err)
	}
	for i, sample := range samples {
		frame, err := dec.DecodeConfiguredAVC(sample)
		if err != nil {
			t.Fatalf("sample[%d]: %v", i, err)
		}
		assertFrameMD5Strings(t, []*Frame{frame}, want[i:i+1])
	}
}

func TestDecodeConfiguredAVCAcrossSamplesCABACFrames(t *testing.T) {
	data := decodeHexFixture(t, testsrc16CABACAnnexBHex)
	want := []string{
		"57948a884e4468c79f3291b2693263de",
		"4fb1e27b7087e9f1aa485402993ca525",
		"a7e3e74bb19403d111dd2ffdb4455102",
		"1202e58b9b15f56a341fea8787bcc769",
	}
	config, samples := annexBToAVCConfigAndSamples(t, data, 3)
	if len(samples) != len(want) {
		t.Fatalf("samples = %d, want %d", len(samples), len(want))
	}

	dec := NewDecoder()
	if _, err := dec.ParseAVCDecoderConfigurationRecord(config); err != nil {
		t.Fatal(err)
	}
	for i, sample := range samples {
		frame, err := dec.DecodeConfiguredAVC(sample)
		if err != nil {
			t.Fatalf("sample[%d]: %v", i, err)
		}
		assertFrameMD5Strings(t, []*Frame{frame}, want[i:i+1])
	}
}

func TestDecodeConfiguredAVCAcrossSamplesHigh422Frames(t *testing.T) {
	for _, tt := range []struct {
		name string
		hex  string
		want []string
	}{
		{
			name: "cavlc",
			hex:  testsrc16CAVLC422AnnexBHex,
			want: []string{
				"b37a1f7943ce6c7d9646786f348f4ce9",
				"e705648238ec1a68ce2fc83f8d1b7293",
				"13cfed6389834373ccb5b6bb61f6cf9d",
				"f0b4d1caf4e666cc4767cfe273de480e",
			},
		},
		{
			name: "cabac",
			hex:  testsrc16CABAC422AnnexBHex,
			want: []string{
				"e06b0f34fe689940304653e5c3840a53",
				"424fb373278235a5d2b0808968cb0e58",
				"b6e4d159f8c0b0bb452de55824214ac6",
				"892dfdee5dbf37558f99a6fe0c278abb",
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			config, samples := annexBToAVCConfigAndSamples(t, decodeHexFixture(t, tt.hex), 3)
			if len(samples) != len(tt.want) {
				t.Fatalf("samples = %d, want %d", len(samples), len(tt.want))
			}

			dec := NewDecoder()
			if _, err := dec.ParseAVCDecoderConfigurationRecord(config); err != nil {
				t.Fatal(err)
			}
			for i, sample := range samples {
				frame, err := dec.DecodeConfiguredAVC(sample)
				if err != nil {
					t.Fatalf("sample[%d]: %v", i, err)
				}
				assertHigh422FrameMD5Strings(t, []*Frame{frame}, tt.want[i:i+1])
			}
		})
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
		"-show_entries", "stream=codec_name,profile,width,height,level,pix_fmt,sample_aspect_ratio,r_frame_rate",
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
			SAR       string `json:"sample_aspect_ratio"`
			FrameRate string `json:"r_frame_rate"`
		} `json:"streams"`
	}
	if err := json.Unmarshal(out, &probe); err != nil {
		t.Fatal(err)
	}
	if len(probe.Streams) != 1 {
		t.Fatalf("ffprobe streams = %d", len(probe.Streams))
	}

	dec := NewDecoder()
	info, err := dec.ParseHeadersAnnexB(data)
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
	if stream.SAR != ratioColonString(info.SARNum, info.SARDen) {
		t.Fatalf("oracle SAR %s, go %d:%d", stream.SAR, info.SARNum, info.SARDen)
	}
	if info.TimingInfoPresentFlag != 0 {
		if stream.FrameRate != ratioSlashString(int64(info.TimeScale), int64(info.NumUnitsInTick)) {
			t.Fatalf("oracle r_frame_rate %s, go timing %d/%d", stream.FrameRate, info.TimeScale, info.NumUnitsInTick)
		}
	}
}

func TestFFprobeOracleRecoveryPointKeyFrame(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffprobe oracle")
	}
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe not available")
	}

	data := insertAnnexBNALBeforeVCL(t, decodeHexFixture(t, black16IPAnnexBHex), decoderTestSEINAL(
		decoderSEITestMessage{typ: decoderSEITypeRecoveryPoint, payload: decoderSEIRecoveryPointPayloadWith(0)},
	), 1)
	path := writeTempH264(t, data)

	cmd := exec.Command("ffprobe",
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "frame=key_frame",
		"-of", "json",
		path,
	)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("ffprobe: %v", err)
	}
	var probe struct {
		Frames []struct {
			KeyFrame int `json:"key_frame"`
		} `json:"frames"`
	}
	if err := json.Unmarshal(out, &probe); err != nil {
		t.Fatal(err)
	}

	frames := decodeConfiguredIPWithRecoveryPoint(t, 0)
	if len(probe.Frames) != len(frames) {
		t.Fatalf("oracle frames = %d, go = %d", len(probe.Frames), len(frames))
	}
	for i := range frames {
		want := probe.Frames[i].KeyFrame != 0
		if frames[i].KeyFrame != want {
			t.Fatalf("frame[%d] key = %t, oracle %t", i, frames[i].KeyFrame, want)
		}
	}
}

func TestFFprobeOracleLCEVCSideData(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffprobe oracle")
	}
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe not available")
	}

	wantLCEVC := []byte{0x7e, 0x00, 0x00, 0x03, 0x01}
	data := prependAnnexBNAL(decodeHexFixture(t, black16AnnexBHex), decoderTestSEINAL(
		decoderSEITestMessage{typ: decoderSEITypeUserDataRegisteredITUTT35, payload: decoderSEIRegisteredLCEVCPayload(wantLCEVC)},
	))
	path := writeTempH264(t, data)

	cmd := exec.Command("ffprobe",
		"-v", "error",
		"-select_streams", "v:0",
		"-show_frames",
		"-of", "json",
		path,
	)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("ffprobe: %v", err)
	}
	var probe struct {
		Frames []struct {
			SideDataList []struct {
				SideDataType string `json:"side_data_type"`
			} `json:"side_data_list"`
		} `json:"frames"`
	}
	if err := json.Unmarshal(out, &probe); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, frame := range probe.Frames {
		for _, side := range frame.SideDataList {
			if strings.Contains(side.SideDataType, "LCEVC") {
				found = true
			}
		}
	}
	if !found {
		t.Fatalf("ffprobe LCEVC side data not found in %s", out)
	}

	frame, err := NewDecoder().Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	if got := frame.SideData.LCEVC; !bytes.Equal(got, wantLCEVC) {
		t.Fatalf("go lcevc = %x, want %x", got, wantLCEVC)
	}
}

func TestFFprobeOracleHigh422(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffprobe oracle")
	}
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe not available")
	}

	for _, tt := range []struct {
		name string
		hex  string
	}{
		{name: "cavlc", hex: testsrc16CAVLC422AnnexBHex},
		{name: "cabac", hex: testsrc16CABAC422AnnexBHex},
	} {
		t.Run(tt.name, func(t *testing.T) {
			data := decodeHexFixture(t, tt.hex)
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
			if stream.CodecName != "h264" || stream.PixFmt != "yuv422p" {
				t.Fatalf("unexpected oracle stream: %+v", stream)
			}
			if stream.Profile != info.Profile || stream.Width != info.Width || stream.Height != info.Height || stream.Level != int(info.LevelIDC) {
				t.Fatalf("oracle %+v, go %+v", stream, info)
			}
		})
	}
}

func ratioColonString(num int32, den int32) string {
	if den == 0 {
		den = 1
	}
	return fmt.Sprintf("%d:%d", num, den)
}

func ratioSlashString(num int64, den int64) string {
	if den == 0 {
		den = 1
	}
	g := gcdInt64(num, den)
	return fmt.Sprintf("%d/%d", num/g, den/g)
}

func gcdInt64(a int64, b int64) int64 {
	if a < 0 {
		a = -a
	}
	if b < 0 {
		b = -b
	}
	for b != 0 {
		a, b = b, a%b
	}
	if a == 0 {
		return 1
	}
	return a
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

func TestFFmpegFrameMD5OracleTestsrc16High422(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}

	for _, tt := range []struct {
		name string
		hex  string
		want []string
	}{
		{
			name: "cavlc",
			hex:  testsrc16CAVLC422AnnexBHex,
			want: []string{
				"0,          0,          0,        1,      512, b37a1f7943ce6c7d9646786f348f4ce9",
				"0,          1,          1,        1,      512, e705648238ec1a68ce2fc83f8d1b7293",
				"0,          2,          2,        1,      512, 13cfed6389834373ccb5b6bb61f6cf9d",
				"0,          3,          3,        1,      512, f0b4d1caf4e666cc4767cfe273de480e",
			},
		},
		{
			name: "cabac",
			hex:  testsrc16CABAC422AnnexBHex,
			want: []string{
				"0,          0,          0,        1,      512, e06b0f34fe689940304653e5c3840a53",
				"0,          1,          1,        1,      512, 424fb373278235a5d2b0808968cb0e58",
				"0,          2,          2,        1,      512, b6e4d159f8c0b0bb452de55824214ac6",
				"0,          3,          3,        1,      512, 892dfdee5dbf37558f99a6fe0c278abb",
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			data := decodeHexFixture(t, tt.hex)
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
			for _, line := range tt.want {
				if !bytes.Contains(out, []byte(line)) {
					t.Fatalf("missing %q in framemd5:\n%s", line, out)
				}
			}
		})
	}
}

const (
	decoderSEITypePicTiming                    = 1
	decoderSEITypeUserDataRegisteredITUTT35    = 4
	decoderSEITypeRecoveryPoint                = 6
	decoderSEITypeFilmGrainCharacteristics     = 19
	decoderSEITypeGreenMetadata                = 56
	decoderSEITypeFramePackingArrangement      = 45
	decoderSEITypeDisplayOrientation           = 47
	decoderSEITypeMasteringDisplayColourVolume = 137
	decoderSEITypeContentLightLevelInfo        = 144
	decoderSEITypeAlternativeTransfer          = 147
	decoderSEITypeAmbientViewingEnvironment    = 148

	decoderSEIPicStructFrame         = 0
	decoderSEIPicStructTopField      = 1
	decoderSEIPicStructTopBottom     = 3
	decoderSEIPicStructTopBottomTop  = 5
	decoderSEIPicStructFrameDoubling = 7
	decoderSEIPicStructFrameTripling = 8
)

func decoderSPSNALWithRichVUI(t *testing.T) []byte {
	t.Helper()
	return decoderSPSNALWithRichVUITiming(t, 1001, 60000)
}

func decoderSPSNALWithRichVUITiming(t *testing.T, numUnitsInTick uint32, timeScale uint32) []byte {
	t.Helper()
	var b decoderSEIBitBuilder
	b.writeBits(66, 8)   // profile_idc
	b.writeBits(0xc0, 8) // constraint flags plus reserved bits
	b.writeBits(30, 8)   // level_idc
	b.writeUE(0)         // seq_parameter_set_id
	b.writeUE(0)         // log2_max_frame_num_minus4
	b.writeUE(2)         // pic_order_cnt_type
	b.writeUE(0)         // max_num_ref_frames
	b.writeBit(0)        // gaps_in_frame_num_value_allowed_flag
	b.writeUE(0)         // pic_width_in_mbs_minus1
	b.writeUE(0)         // pic_height_in_map_units_minus1
	b.writeBit(1)        // frame_mbs_only_flag
	b.writeBit(1)        // direct_8x8_inference_flag
	b.writeBit(0)        // frame_cropping_flag
	b.writeBit(1)        // vui_parameters_present_flag

	b.writeBit(1)       // aspect_ratio_info_present_flag
	b.writeBits(255, 8) // Extended_SAR
	b.writeBits(4, 16)
	b.writeBits(3, 16)
	b.writeBit(1) // overscan_info_present_flag
	b.writeBit(0)
	b.writeBit(1) // video_signal_type_present_flag
	b.writeBits(5, 3)
	b.writeBit(1)
	b.writeBit(1) // colour_description_present_flag
	b.writeBits(1, 8)
	b.writeBits(1, 8)
	b.writeBits(1, 8)
	b.writeBit(1) // chroma_loc_info_present_flag
	b.writeUE(2)
	b.writeUE(3)
	b.writeBit(1) // timing_info_present_flag
	b.writeBits(numUnitsInTick, 32)
	b.writeBits(timeScale, 32)
	b.writeBit(1)
	b.writeBit(0) // nal_hrd_parameters_present_flag
	b.writeBit(0) // vcl_hrd_parameters_present_flag
	b.writeBit(1) // pic_struct_present_flag
	b.writeBit(1) // bitstream_restriction_flag
	b.writeBit(1) // motion_vectors_over_pic_boundaries_flag
	b.writeUE(0)
	b.writeUE(1)
	b.writeUE(8)
	b.writeUE(9)
	b.writeUE(2)
	b.writeUE(4)

	rbsp := b.rbsp()
	raw := []byte{0x67}
	return append(raw, escapeRBSPForNALPayload(rbsp)...)
}

func decoderSPSNALWithPicStructVUI() []byte {
	return decoderSPSNALWithPicStructVUITiming(1001, 60000)
}

func decoderSPSNALWithPicStructVUITiming(numUnitsInTick uint32, timeScale uint32) []byte {
	var b decoderSEIBitBuilder
	b.writeBits(66, 8)   // profile_idc
	b.writeBits(0xc0, 8) // constraint flags plus reserved bits
	b.writeBits(10, 8)   // level_idc
	b.writeUE(0)         // seq_parameter_set_id
	b.writeUE(0)         // log2_max_frame_num_minus4
	b.writeUE(2)         // pic_order_cnt_type
	b.writeUE(0)         // max_num_ref_frames
	b.writeBit(0)        // gaps_in_frame_num_value_allowed_flag
	b.writeUE(0)         // pic_width_in_mbs_minus1
	b.writeUE(0)         // pic_height_in_map_units_minus1
	b.writeBit(1)        // frame_mbs_only_flag
	b.writeBit(1)        // direct_8x8_inference_flag
	b.writeBit(0)        // frame_cropping_flag
	b.writeBit(1)        // vui_parameters_present_flag
	b.writeBit(0)        // aspect_ratio_info_present_flag
	b.writeBit(0)        // overscan_info_present_flag
	b.writeBit(0)        // video_signal_type_present_flag
	b.writeBit(0)        // chroma_loc_info_present_flag
	b.writeBit(1)        // timing_info_present_flag
	b.writeBits(numUnitsInTick, 32)
	b.writeBits(timeScale, 32)
	b.writeBit(1) // fixed_frame_rate_flag
	b.writeBit(0) // nal_hrd_parameters_present_flag
	b.writeBit(0) // vcl_hrd_parameters_present_flag
	b.writeBit(1) // pic_struct_present_flag
	b.writeBit(0) // bitstream_restriction_flag

	rbsp := b.rbsp()
	raw := []byte{0x67}
	return append(raw, escapeRBSPForNALPayload(rbsp)...)
}

type decoderSEITestMessage struct {
	typ     int
	payload []byte
}

func decoderTestSEINAL(messages ...decoderSEITestMessage) []byte {
	rbsp := decoderTestSEIRBSP(messages...)
	return append([]byte{byte(h264.NALSEI)}, escapeRBSPForNALPayload(rbsp)...)
}

func decoderTestSEIRBSP(messages ...decoderSEITestMessage) []byte {
	var out []byte
	for _, msg := range messages {
		out = appendDecoderSEIHeaderValue(out, msg.typ)
		out = appendDecoderSEIHeaderValue(out, len(msg.payload))
		out = append(out, msg.payload...)
	}
	return append(out, 0x80)
}

func appendDecoderSEIHeaderValue(out []byte, value int) []byte {
	for value >= 255 {
		out = append(out, 255)
		value -= 255
	}
	return append(out, uint8(value))
}

func decoderSEIPictureTimingPayload(picStruct uint32) []byte {
	var b decoderSEIBitBuilder
	b.writeBits(picStruct, 4)
	for i := uint8(0); i < decoderSEINumClockTSTable[picStruct]; i++ {
		b.writeBit(0)
	}
	return b.bytes()
}

func decoderSEIPictureTimingTimecodePayload() []byte {
	return decoderSEIPictureTimingTimecodePayloadWithFrame(true, 0)
}

func decoderSEIPictureTimingTimecodePayloadWithFrame(drop bool, frame uint32) []byte {
	var b decoderSEIBitBuilder
	b.writeBits(decoderSEIPicStructFrame, 4)
	b.writeBit(1)
	b.writeBits(2, 2)
	b.writeBit(0)
	b.writeBits(3, 5)
	b.writeBit(1)
	b.writeBit(0)
	if drop {
		b.writeBit(1)
	} else {
		b.writeBit(0)
	}
	b.writeBits(frame, 8)
	b.writeBits(34, 6)
	b.writeBits(56, 6)
	b.writeBits(7, 5)
	b.writeBits(0, 24)
	return b.bytes()
}

var decoderSEINumClockTSTable = [9]uint8{1, 1, 1, 2, 2, 3, 3, 2, 3}

func decoderSEIRegisteredA53Payload(cc []byte) []byte {
	if len(cc)%3 != 0 {
		panic("A53 test payload must contain whole three-byte CC entries")
	}
	out := []byte{0xb5, 0x00, 0x31, 'G', 'A', '9', '4', 0x03}
	out = append(out, 0x40|uint8(len(cc)/3), 0xff)
	out = append(out, cc...)
	out = append(out, 0xff)
	return out
}

func decoderSEIRegisteredAFDPayload(description uint8) []byte {
	return []byte{0xb5, 0x00, 0x31, 'D', 'T', 'G', '1', 0x40, description}
}

func decoderSEIRegisteredLCEVCPayload(data []byte) []byte {
	out := []byte{0xb4, 0x00, 0x50, 0x01}
	return append(out, data...)
}

func decoderSEIRecoveryPointPayload() []byte {
	return decoderSEIRecoveryPointPayloadWith(4)
}

func decoderSEIRecoveryPointPayloadWith(frameCount uint32) []byte {
	var b decoderSEIBitBuilder
	b.writeUE(frameCount)
	b.writeBit(1)
	b.writeBit(0)
	b.writeBits(2, 2)
	return b.bytes()
}

func decoderSEIDisplayOrientationPayload() []byte {
	return decoderSEIDisplayOrientationPayloadWith(0x4000, true, false)
}

func decoderSEIDisplayOrientationPayloadWith(rotation uint32, hflip bool, vflip bool) []byte {
	var b decoderSEIBitBuilder
	b.writeBit(0)
	if hflip {
		b.writeBit(1)
	} else {
		b.writeBit(0)
	}
	if vflip {
		b.writeBit(1)
	} else {
		b.writeBit(0)
	}
	b.writeBits(rotation, 16)
	return b.bytes()
}

func decoderSEIFramePackingPayload() []byte {
	return decoderSEIFramePackingPayloadWith(3, 2, false, true)
}

func decoderSEIFramePackingPayloadWith(arrangementType uint32, contentInterpretation uint32, quincunx bool, currentFrameIsFrame0 bool) []byte {
	var b decoderSEIBitBuilder
	b.writeUE(2)
	b.writeBit(0)
	b.writeBits(arrangementType, 7)
	if quincunx {
		b.writeBit(1)
	} else {
		b.writeBit(0)
	}
	b.writeBits(contentInterpretation, 6)
	b.writeBits(0, 3)
	if currentFrameIsFrame0 {
		b.writeBit(1)
	} else {
		b.writeBit(0)
	}
	b.writeBits(0, 2)
	b.writeBits(0x1234, 16)
	b.writeBits(0, 8)
	b.writeUE(5)
	b.writeBit(0)
	return b.bytes()
}

func decoderSEIAmbientViewingPayload() []byte {
	return []byte{0x00, 0x00, 0x30, 0x39, 0x61, 0xa8, 0x41, 0x1b}
}

func decoderSEIMasteringDisplayPayload() []byte {
	return []byte{
		0x27, 0x10, 0x4e, 0x20,
		0x3a, 0x98, 0x61, 0xa8,
		0x75, 0x30, 0x88, 0xb8,
		0x3d, 0x13, 0x40, 0x42,
		0x00, 0x98, 0x96, 0x80,
		0x00, 0x00, 0x00, 0x64,
	}
}

func decoderPacketAmbientViewingSideData(illuminance uint32, lightX uint16, lightY uint16) []byte {
	var out []byte
	out = appendDecoderAVRationalLE(out, illuminance, 10000)
	out = appendDecoderAVRationalLE(out, uint32(lightX), 50000)
	out = appendDecoderAVRationalLE(out, uint32(lightY), 50000)
	return out
}

func decoderPacketDisplayMatrixSideData(matrix [9]int32) []byte {
	out := make([]byte, 9*4)
	for i, v := range matrix {
		binary.LittleEndian.PutUint32(out[i*4:i*4+4], uint32(v))
	}
	return out
}

func decoderPacketStereo3DSideData(typ int32, flags int32, view int32, primaryEye int32, baseline uint32, disparity Rational, fieldOfView Rational) []byte {
	out := make([]byte, 36)
	binary.LittleEndian.PutUint32(out[0:4], uint32(typ))
	binary.LittleEndian.PutUint32(out[4:8], uint32(flags))
	binary.LittleEndian.PutUint32(out[8:12], uint32(view))
	binary.LittleEndian.PutUint32(out[12:16], uint32(primaryEye))
	binary.LittleEndian.PutUint32(out[16:20], baseline)
	binary.LittleEndian.PutUint32(out[20:24], uint32(disparity.Num))
	binary.LittleEndian.PutUint32(out[24:28], uint32(disparity.Den))
	binary.LittleEndian.PutUint32(out[28:32], uint32(fieldOfView.Num))
	binary.LittleEndian.PutUint32(out[32:36], uint32(fieldOfView.Den))
	return out
}

func decoderPacketSphericalSideData(projection int32, yaw int32, pitch int32, roll int32, boundLeft uint32, boundTop uint32, boundRight uint32, boundBottom uint32, padding uint32) []byte {
	out := make([]byte, 36)
	binary.LittleEndian.PutUint32(out[0:4], uint32(projection))
	binary.LittleEndian.PutUint32(out[4:8], uint32(yaw))
	binary.LittleEndian.PutUint32(out[8:12], uint32(pitch))
	binary.LittleEndian.PutUint32(out[12:16], uint32(roll))
	binary.LittleEndian.PutUint32(out[16:20], boundLeft)
	binary.LittleEndian.PutUint32(out[20:24], boundTop)
	binary.LittleEndian.PutUint32(out[24:28], boundRight)
	binary.LittleEndian.PutUint32(out[28:32], boundBottom)
	binary.LittleEndian.PutUint32(out[32:36], padding)
	return out
}

func decoderPacketReferenceDisplaysSideData(precWidth uint8, viewingDistance bool, precDistance uint8, displays []ReferenceDisplay) []byte {
	const (
		headerSize = 24
		entrySize  = 12
	)
	out := make([]byte, headerSize+entrySize*len(displays))
	out[0] = precWidth
	if viewingDistance {
		out[1] = 1
	}
	out[2] = precDistance
	out[3] = uint8(len(displays))
	binary.LittleEndian.PutUint64(out[8:16], headerSize)
	binary.LittleEndian.PutUint64(out[16:24], entrySize)
	for i, display := range displays {
		off := headerSize + i*entrySize
		binary.LittleEndian.PutUint16(out[off:off+2], display.LeftViewID)
		binary.LittleEndian.PutUint16(out[off+2:off+4], display.RightViewID)
		out[off+4] = display.ExponentRefDisplayWidth
		out[off+5] = display.MantissaRefDisplayWidth
		out[off+6] = display.ExponentRefViewingDistance
		out[off+7] = display.MantissaRefViewingDistance
		if display.AdditionalShiftPresentFlag {
			out[off+8] = 1
		}
		binary.LittleEndian.PutUint16(out[off+10:off+12], uint16(display.NumSampleShift))
	}
	return out
}

func decoderPacketMasteringDisplaySideData(primaries [3][2]uint16, white [2]uint16, maxLuminance uint32, minLuminance uint32, hasPrimaries bool, hasLuminance bool) []byte {
	var out []byte
	for i := range primaries {
		for j := range primaries[i] {
			out = appendDecoderAVRationalLE(out, uint32(primaries[i][j]), 50000)
		}
	}
	for i := range white {
		out = appendDecoderAVRationalLE(out, uint32(white[i]), 50000)
	}
	out = appendDecoderAVRationalLE(out, minLuminance, 10000)
	out = appendDecoderAVRationalLE(out, maxLuminance, 10000)
	out = appendDecoderBoolInt32LE(out, hasPrimaries)
	out = appendDecoderBoolInt32LE(out, hasLuminance)
	return out
}

func decoderPacketContentLightSideData(maxCLL uint32, maxFALL uint32) []byte {
	out := make([]byte, 8)
	binary.LittleEndian.PutUint32(out[:4], maxCLL)
	binary.LittleEndian.PutUint32(out[4:8], maxFALL)
	return out
}

func appendDecoderAVRationalLE(dst []byte, numerator uint32, denominator uint32) []byte {
	var buf [8]byte
	binary.LittleEndian.PutUint32(buf[:4], numerator)
	binary.LittleEndian.PutUint32(buf[4:8], denominator)
	return append(dst, buf[:]...)
}

func appendDecoderBoolInt32LE(dst []byte, v bool) []byte {
	var buf [4]byte
	if v {
		binary.LittleEndian.PutUint32(buf[:], 1)
	}
	return append(dst, buf[:]...)
}

func decoderSEIFilmGrainPayload() []byte {
	return decoderSEIFilmGrainPayloadWithRepetition(4)
}

func decoderSEIFilmGrainPayloadWithRepetition(repetitionPeriod uint32) []byte {
	var b decoderSEIBitBuilder
	b.writeBit(0)
	b.writeBits(1, 2)
	b.writeBit(1)
	b.writeBits(2, 3)
	b.writeBits(0, 3)
	b.writeBit(1)
	b.writeBits(9, 8)
	b.writeBits(16, 8)
	b.writeBits(9, 8)
	b.writeBits(1, 2)
	b.writeBits(7, 4)
	b.writeBit(1)
	b.writeBit(1)
	b.writeBit(0)
	b.writeBits(0, 8)
	b.writeBits(1, 3)
	b.writeBits(10, 8)
	b.writeBits(20, 8)
	b.writeSE(3)
	b.writeSE(-2)
	b.writeBits(1, 8)
	b.writeBits(0, 3)
	b.writeBits(30, 8)
	b.writeBits(40, 8)
	b.writeSE(-1)
	b.writeBits(41, 8)
	b.writeBits(60, 8)
	b.writeSE(5)
	b.writeUE(repetitionPeriod)
	return b.bytes()
}

type decoderSEIBitBuilder struct {
	bits []byte
}

func (b *decoderSEIBitBuilder) writeBit(v uint32) {
	if v&1 != 0 {
		b.bits = append(b.bits, 1)
	} else {
		b.bits = append(b.bits, 0)
	}
}

func (b *decoderSEIBitBuilder) writeBits(v uint32, n uint32) {
	for i := int(n) - 1; i >= 0; i-- {
		b.writeBit(v >> uint(i))
	}
}

func (b *decoderSEIBitBuilder) writeUE(v uint32) {
	codeNum := v + 1
	bitLen := 32 - bits.LeadingZeros32(codeNum)
	for i := 0; i < bitLen-1; i++ {
		b.writeBit(0)
	}
	b.writeBits(codeNum, uint32(bitLen))
}

func (b *decoderSEIBitBuilder) writeSE(v int32) {
	var ue uint32
	if v <= 0 {
		ue = uint32(-v) * 2
	} else {
		ue = uint32(v)*2 - 1
	}
	b.writeUE(ue)
}

func (b *decoderSEIBitBuilder) bytes() []byte {
	out := make([]byte, (len(b.bits)+7)/8)
	for i, bit := range b.bits {
		if bit != 0 {
			out[i/8] |= 1 << uint(7-i%8)
		}
	}
	return out
}

func (b *decoderSEIBitBuilder) rbsp() []byte {
	b.writeBit(1)
	for len(b.bits)&7 != 0 {
		b.writeBit(0)
	}
	return b.bytes()
}

func escapeRBSPForNALPayload(rbsp []byte) []byte {
	out := make([]byte, 0, len(rbsp))
	zeros := 0
	for _, b := range rbsp {
		if zeros == 2 && b <= 3 {
			out = append(out, 0x03)
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

func prependAnnexBNAL(data []byte, raw []byte) []byte {
	out := appendAnnexBNAL(nil, raw)
	return append(out, data...)
}

func appendAnnexBNAL(dst []byte, raw []byte) []byte {
	dst = append(dst, 0x00, 0x00, 0x00, 0x01)
	return append(dst, raw...)
}

func insertAnnexBNALBeforeVCL(t *testing.T, data []byte, raw []byte, vclIndex int) []byte {
	t.Helper()
	nals, err := h264.SplitAnnexB(data)
	if err != nil {
		t.Fatal(err)
	}
	var out []byte
	seenVCL := 0
	inserted := false
	for _, nal := range nals {
		isVCL := nal.Type == h264.NALSlice || nal.Type == h264.NALIDRSlice
		if isVCL && seenVCL == vclIndex {
			out = appendAnnexBNAL(out, raw)
			inserted = true
		}
		out = appendAnnexBNAL(out, nal.Raw)
		if isVCL {
			seenVCL++
		}
	}
	if !inserted {
		t.Fatalf("VCL index %d not found", vclIndex)
	}
	return out
}

func replaceAnnexBSPS(t *testing.T, data []byte, sps []byte) []byte {
	t.Helper()
	nals, err := h264.SplitAnnexB(data)
	if err != nil {
		t.Fatal(err)
	}
	var out []byte
	replaced := false
	for _, nal := range nals {
		if nal.Type == h264.NALSPS && !replaced {
			out = appendAnnexBNAL(out, sps)
			replaced = true
			continue
		}
		out = appendAnnexBNAL(out, nal.Raw)
	}
	if !replaced {
		t.Fatal("no SPS NAL found")
	}
	return out
}

func annexBParameterSetsAndPacket(t *testing.T, data []byte) ([]byte, []byte) {
	t.Helper()
	nals, err := h264.SplitAnnexB(data)
	if err != nil {
		t.Fatal(err)
	}
	var extradata []byte
	var packet []byte
	for _, nal := range nals {
		switch nal.Type {
		case h264.NALSPS, h264.NALPPS:
			extradata = appendAnnexBNAL(extradata, nal.Raw)
		default:
			packet = appendAnnexBNAL(packet, nal.Raw)
		}
	}
	if len(extradata) == 0 || len(packet) == 0 {
		t.Fatalf("annexb split produced extradata=%d packet=%d", len(extradata), len(packet))
	}
	return extradata, packet
}

func annexBToAVCConfigAndPacket(t *testing.T, data []byte, nalLengthSize int) ([]byte, []byte) {
	t.Helper()
	config, samples := annexBToAVCConfigAndSamples(t, data, nalLengthSize)
	var packet []byte
	for _, sample := range samples {
		packet = append(packet, sample...)
	}
	return config, packet
}

func annexBToAVCConfigAndSamples(t *testing.T, data []byte, nalLengthSize int) ([]byte, [][]byte) {
	t.Helper()
	if nalLengthSize < 1 || nalLengthSize > 4 {
		t.Fatalf("invalid nalLengthSize %d", nalLengthSize)
	}
	nals, err := h264.SplitAnnexB(data)
	if err != nil {
		t.Fatal(err)
	}

	var spsNals [][]byte
	var ppsNals [][]byte
	var spsList [32]*h264.SPS
	var ppsList [256]*h264.PPS
	var samples [][]byte
	var sample []byte
	hasVCL := false
	for _, nal := range nals {
		switch nal.Type {
		case h264.NALSPS:
			sps, err := h264.DecodeSPS(nal.RBSP)
			if err != nil {
				t.Fatal(err)
			}
			spsList[sps.SPSID] = sps
			spsNals = append(spsNals, nal.Raw)
		case h264.NALPPS:
			pps, err := h264.DecodePPS(nal.RBSP, &spsList)
			if err != nil {
				t.Fatal(err)
			}
			ppsList[pps.PPSID] = pps
			ppsNals = append(ppsNals, nal.Raw)
		default:
			isVCL := nal.Type == h264.NALSlice || nal.Type == h264.NALIDRSlice
			if isVCL {
				sh, err := h264.ParseSliceHeader(nal, &ppsList)
				if err != nil {
					t.Fatal(err)
				}
				if hasVCL && sh.FirstMBAddr == 0 {
					samples = append(samples, sample)
					sample = nil
					hasVCL = false
				}
			}
			sample = appendAVCNALUnit(t, sample, nal.Raw, nalLengthSize)
			if isVCL {
				hasVCL = true
			}
		}
	}
	if len(sample) != 0 {
		samples = append(samples, sample)
	}
	if len(spsNals) == 0 || len(spsNals) > 31 || len(ppsNals) == 0 || len(ppsNals) > 255 {
		t.Fatalf("parameter set counts: sps=%d pps=%d", len(spsNals), len(ppsNals))
	}
	if len(spsNals[0]) < 4 {
		t.Fatalf("short SPS NAL: %x", spsNals[0])
	}

	config := []byte{
		1,
		spsNals[0][1],
		spsNals[0][2],
		spsNals[0][3],
		0xfc | byte(nalLengthSize-1),
		0xe0 | byte(len(spsNals)),
	}
	for _, raw := range spsNals {
		config = appendAVCConfigNALUnit(t, config, raw)
	}
	config = append(config, byte(len(ppsNals)))
	for _, raw := range ppsNals {
		config = appendAVCConfigNALUnit(t, config, raw)
	}
	return config, samples
}

func appendAVCNALUnit(t *testing.T, dst []byte, raw []byte, nalLengthSize int) []byte {
	t.Helper()
	maxSize := uint64(1)<<(uint(nalLengthSize)*8) - 1
	size := len(raw)
	if size == 0 || uint64(size) > maxSize {
		t.Fatalf("NAL size %d exceeds %d-byte length field", size, nalLengthSize)
	}
	for shift := (nalLengthSize - 1) * 8; shift >= 0; shift -= 8 {
		dst = append(dst, byte(size>>shift))
	}
	return append(dst, raw...)
}

func appendAVCConfigNALUnit(t *testing.T, dst []byte, raw []byte) []byte {
	t.Helper()
	if len(raw) == 0 || len(raw) > 0xffff {
		t.Fatalf("bad config NAL size %d", len(raw))
	}
	dst = append(dst, byte(len(raw)>>8), byte(len(raw)))
	return append(dst, raw...)
}

func assertFrameMD5Strings(t *testing.T, frames []*Frame, want []string) {
	t.Helper()
	if len(frames) != len(want) {
		t.Fatalf("frames = %d, want %d", len(frames), len(want))
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

func assertHigh422FrameMD5Strings(t *testing.T, frames []*Frame, want []string) {
	t.Helper()
	if len(frames) != len(want) {
		t.Fatalf("frames = %d, want %d", len(frames), len(want))
	}
	for i, frame := range frames {
		if frame.Width != 16 || frame.Height != 16 || frame.ChromaFormatIDC != 2 || frame.BitDepthLuma != 8 || frame.BitDepthChroma != 8 {
			t.Fatalf("frame[%d] metadata = %dx%d chroma %d depth %d/%d", i, frame.Width, frame.Height, frame.ChromaFormatIDC, frame.BitDepthLuma, frame.BitDepthChroma)
		}
		raw, err := frame.AppendRawYUV(nil)
		if err != nil {
			t.Fatalf("frame[%d] raw yuv: %v", i, err)
		}
		if len(raw) != 512 {
			t.Fatalf("frame[%d] raw frame size = %d, want 512", i, len(raw))
		}
		got := md5.Sum(raw)
		if hex.EncodeToString(got[:]) != want[i] {
			t.Fatalf("frame[%d] md5 = %x, want %s", i, got, want[i])
		}
	}
}

func decodeConfiguredIPWithRecoveryPoint(t *testing.T, recoveryFrameCount uint32) []*Frame {
	t.Helper()
	config, samples := annexBToAVCConfigAndSamples(t, decodeHexFixture(t, black16IPAnnexBHex), 4)
	if len(samples) != 2 {
		t.Fatalf("samples = %d, want 2", len(samples))
	}
	sei := decoderTestSEINAL(decoderSEITestMessage{
		typ:     decoderSEITypeRecoveryPoint,
		payload: decoderSEIRecoveryPointPayloadWith(recoveryFrameCount),
	})
	samples[1] = append(appendAVCNALUnit(t, nil, sei, 4), samples[1]...)

	dec := NewDecoder()
	if _, err := dec.ParseAVCDecoderConfigurationRecord(config); err != nil {
		t.Fatal(err)
	}
	first, err := dec.DecodeConfiguredAVC(samples[0])
	if err != nil {
		t.Fatal(err)
	}
	second, err := dec.DecodeConfiguredAVC(samples[1])
	if err != nil {
		t.Fatal(err)
	}
	frames := []*Frame{first, second}
	assertFrameMD5Strings(t, frames, []string{
		"8aaefe0adcea094cfb5161a060bab4e2",
		"8aaefe0adcea094cfb5161a060bab4e2",
	})
	return frames
}

func writeTempH264(t *testing.T, data []byte) string {
	t.Helper()
	path := t.TempDir() + "/fixture.h264"
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
