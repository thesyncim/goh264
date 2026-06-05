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

const (
	high12FrameMBAFFPSkipNoResidualBitstreamMD5 = "c72a2e5b5d5bcf216cf092180f5600b2"
	high12FrameMBAFFPSkipNoResidualPFrameMD5    = "5d168280547309a62ad1066a599c4ba5"
	high12FrameMBAFFPSkipNoResidualRawVideoMD5  = "5d1af51e10e3ea2a87b530ca462543c2"

	high12FrameMBAFFP16x16NoResidualBitstreamMD5 = "1bdb8c58f3a8a6a7f2d9802921c74361"
	high12FrameMBAFFP16x16NoResidualPFrameMD5    = "5d168280547309a62ad1066a599c4ba5"
	high12FrameMBAFFP16x16NoResidualRawVideoMD5  = "5d1af51e10e3ea2a87b530ca462543c2"

	high12FrameMBAFFP16x16LumaResidualBitstreamMD5 = "5c1ccfb6647f6008932eb78566bd03a9"
	high12FrameMBAFFP16x16LumaResidualPFrameMD5    = "e4035988ecbfd8504393a3e88f7726f2"
	high12FrameMBAFFP16x16LumaResidualRawVideoMD5  = "61b3f0aeaf64684cd4156ed61a7e7c69"

	high12FrameMBAFFFrameP16x16NoResidualBitstreamMD5 = "fa0dfdbe12142c0267104faedbcdf26a"
	high12FrameMBAFFFrameP16x16NoResidualPFrameMD5    = "5d168280547309a62ad1066a599c4ba5"
	high12FrameMBAFFFrameP16x16NoResidualRawVideoMD5  = "5d1af51e10e3ea2a87b530ca462543c2"

	high12FrameMBAFFFrameP16x16LumaResidualBitstreamMD5 = "cb61ca3a1cc62672a8a81f9e24d125ba"
	high12FrameMBAFFFrameP16x16LumaResidualPFrameMD5    = "9dee5b79f6b5454527202eb9f5f5409a"
	high12FrameMBAFFFrameP16x16LumaResidualRawVideoMD5  = "ac44fa831476a4c76f0e2eb628948293"

	high12FrameMBAFFP16x16LumaChromaResidualBitstreamMD5 = "b6fab0edf7b64eb55e8ae98914d1b1ff"
	high12FrameMBAFFP16x16LumaChromaResidualPFrameMD5    = "c0dbf6a3e55009aee8c9ac804179c384"
	high12FrameMBAFFP16x16LumaChromaResidualRawVideoMD5  = "70b549601284196fd1131bc5f30efd7e"

	high12FrameMBAFFP16x8LumaChromaResidualBitstreamMD5 = "ed926eaab9cbd0f61543dba21262588c"
	high12FrameMBAFFP16x8LumaChromaResidualPFrameMD5    = "c0dbf6a3e55009aee8c9ac804179c384"
	high12FrameMBAFFP16x8LumaChromaResidualRawVideoMD5  = "70b549601284196fd1131bc5f30efd7e"

	high12FrameMBAFFP8x16LumaChromaResidualBitstreamMD5 = "6c7a8d4532c0f21eb50de06985fc5be8"
	high12FrameMBAFFP8x16LumaChromaResidualPFrameMD5    = "c0dbf6a3e55009aee8c9ac804179c384"
	high12FrameMBAFFP8x16LumaChromaResidualRawVideoMD5  = "70b549601284196fd1131bc5f30efd7e"

	high12FrameMBAFFP8x8LumaChromaResidualBitstreamMD5 = "1c741cc12a5a5731979c3418d15b7498"
	high12FrameMBAFFP8x8LumaChromaResidualPFrameMD5    = "c0dbf6a3e55009aee8c9ac804179c384"
	high12FrameMBAFFP8x8LumaChromaResidualRawVideoMD5  = "70b549601284196fd1131bc5f30efd7e"

	high12FrameMBAFFFrameP16x16LumaChromaResidualBitstreamMD5 = "54073b34105bcf336879f86db856478e"
	high12FrameMBAFFFrameP16x16LumaChromaResidualPFrameMD5    = "9fcdad62af62ebe33d99daaf8d3e79ac"
	high12FrameMBAFFFrameP16x16LumaChromaResidualRawVideoMD5  = "97ff9eb67f1ece11b4755aff72d135d6"

	high12FrameMBAFFFrameP16x8LumaChromaResidualBitstreamMD5 = "787db7ce68fc112939ba24cd06ef45c4"
	high12FrameMBAFFFrameP16x8LumaChromaResidualPFrameMD5    = "9fcdad62af62ebe33d99daaf8d3e79ac"
	high12FrameMBAFFFrameP16x8LumaChromaResidualRawVideoMD5  = "97ff9eb67f1ece11b4755aff72d135d6"

	high12FrameMBAFFFrameP8x16LumaChromaResidualBitstreamMD5 = "7ad60733dc3dc8983c10cfbc56a187e0"
	high12FrameMBAFFFrameP8x16LumaChromaResidualPFrameMD5    = "9fcdad62af62ebe33d99daaf8d3e79ac"
	high12FrameMBAFFFrameP8x16LumaChromaResidualRawVideoMD5  = "97ff9eb67f1ece11b4755aff72d135d6"

	high12FrameMBAFFFrameP8x8LumaChromaResidualBitstreamMD5 = "7766d1ed42f4d97c43e3152bec537278"
	high12FrameMBAFFFrameP8x8LumaChromaResidualPFrameMD5    = "9fcdad62af62ebe33d99daaf8d3e79ac"
	high12FrameMBAFFFrameP8x8LumaChromaResidualRawVideoMD5  = "97ff9eb67f1ece11b4755aff72d135d6"

	high14FrameMBAFFPSkipNoResidualBitstreamMD5 = "446edd4f33960f9a60ab97902d081c23"
	high14FrameMBAFFPSkipNoResidualPFrameMD5    = "7da709fea95b8767edeb8e5963b37f2a"
	high14FrameMBAFFPSkipNoResidualRawVideoMD5  = "237561940e2f07d30cac465fcea640bb"

	high14FrameMBAFFP16x16NoResidualBitstreamMD5 = "fb218b8d8ed9f2c46171ceeb150b46c0"
	high14FrameMBAFFP16x16NoResidualPFrameMD5    = "7da709fea95b8767edeb8e5963b37f2a"
	high14FrameMBAFFP16x16NoResidualRawVideoMD5  = "237561940e2f07d30cac465fcea640bb"

	high14FrameMBAFFP16x16LumaResidualBitstreamMD5 = "d1e34562a78fad5892a70cea9f193d9a"
	high14FrameMBAFFP16x16LumaResidualPFrameMD5    = "fd935c9da5bce0db0b3a454ad05a38b3"
	high14FrameMBAFFP16x16LumaResidualRawVideoMD5  = "50655e79e07d0b15d5ab6e237c591069"

	high14FrameMBAFFFrameP16x16NoResidualBitstreamMD5 = "7e55ae8f953d90dddc94059632eafcff"
	high14FrameMBAFFFrameP16x16NoResidualPFrameMD5    = "7da709fea95b8767edeb8e5963b37f2a"
	high14FrameMBAFFFrameP16x16NoResidualRawVideoMD5  = "237561940e2f07d30cac465fcea640bb"

	high14FrameMBAFFFrameP16x16LumaResidualBitstreamMD5 = "dc94ef5a71c666c531e6cac59b01961c"
	high14FrameMBAFFFrameP16x16LumaResidualPFrameMD5    = "a18837da598699cc9374020639b5b96d"
	high14FrameMBAFFFrameP16x16LumaResidualRawVideoMD5  = "d9e96cf61b26ec0321d2aa0e7f12541a"

	high14FrameMBAFFP16x16LumaChromaResidualBitstreamMD5 = "91046f4ce63c6c955b6f1262deb2b15b"
	high14FrameMBAFFP16x16LumaChromaResidualPFrameMD5    = "5474386e384e7353bd7b33c2d44d04c1"
	high14FrameMBAFFP16x16LumaChromaResidualRawVideoMD5  = "c5a9d16416f05bc6db747b0c603cabc5"

	high14FrameMBAFFP16x8LumaChromaResidualBitstreamMD5 = "e923dc7fddb43a8b81e3ca0e085d5352"
	high14FrameMBAFFP16x8LumaChromaResidualPFrameMD5    = "5474386e384e7353bd7b33c2d44d04c1"
	high14FrameMBAFFP16x8LumaChromaResidualRawVideoMD5  = "c5a9d16416f05bc6db747b0c603cabc5"

	high14FrameMBAFFP8x16LumaChromaResidualBitstreamMD5 = "dd57f677d26e57b2f95573c4b8f6aa6b"
	high14FrameMBAFFP8x16LumaChromaResidualPFrameMD5    = "5474386e384e7353bd7b33c2d44d04c1"
	high14FrameMBAFFP8x16LumaChromaResidualRawVideoMD5  = "c5a9d16416f05bc6db747b0c603cabc5"

	high14FrameMBAFFP8x8LumaChromaResidualBitstreamMD5 = "4b6c070bca89c7db60ee26a258a65a64"
	high14FrameMBAFFP8x8LumaChromaResidualPFrameMD5    = "5474386e384e7353bd7b33c2d44d04c1"
	high14FrameMBAFFP8x8LumaChromaResidualRawVideoMD5  = "c5a9d16416f05bc6db747b0c603cabc5"

	high14FrameMBAFFFrameP16x16LumaChromaResidualBitstreamMD5 = "31c1aeb4e6aada6859ca417fe24279f2"
	high14FrameMBAFFFrameP16x16LumaChromaResidualPFrameMD5    = "b85b53afe622571ae0124eab6310392b"
	high14FrameMBAFFFrameP16x16LumaChromaResidualRawVideoMD5  = "2d65809bb1c4417f27154b4f3ec9bf01"

	high14FrameMBAFFFrameP16x8LumaChromaResidualBitstreamMD5 = "44557f7a612b0d4878ee367ecac2c2b8"
	high14FrameMBAFFFrameP16x8LumaChromaResidualPFrameMD5    = "b85b53afe622571ae0124eab6310392b"
	high14FrameMBAFFFrameP16x8LumaChromaResidualRawVideoMD5  = "2d65809bb1c4417f27154b4f3ec9bf01"

	high14FrameMBAFFFrameP8x16LumaChromaResidualBitstreamMD5 = "79e811513606ca25fd5e80a6f969ba66"
	high14FrameMBAFFFrameP8x16LumaChromaResidualPFrameMD5    = "b85b53afe622571ae0124eab6310392b"
	high14FrameMBAFFFrameP8x16LumaChromaResidualRawVideoMD5  = "2d65809bb1c4417f27154b4f3ec9bf01"

	high14FrameMBAFFFrameP8x8LumaChromaResidualBitstreamMD5 = "7fe988f8aa00a0fc3d1d9aa05b7757ca"
	high14FrameMBAFFFrameP8x8LumaChromaResidualPFrameMD5    = "b85b53afe622571ae0124eab6310392b"
	high14FrameMBAFFFrameP8x8LumaChromaResidualRawVideoMD5  = "2d65809bb1c4417f27154b4f3ec9bf01"

	highFrameMBAFFP16x16NoResidualPayloadBits   = "1111111111111"
	highFrameMBAFFP16x16LumaResidualPayloadBits = "11111101110101111" +
		"1111101110101111"
	highFrameMBAFFP16x16LumaResidualTailBits          = "10101111"
	highFrameMBAFFP16x16LumaChromaResidualTailBits    = "10101111010101011111111"
	highFrameMBAFFP16x16LumaChromaResidualPayloadBits = "11" + highInterP16x16LumaChromaResidualPayloadBits +
		"1" + highInterP16x16LumaChromaResidualPayloadBits
	highFrameMBAFFP16x8LumaChromaResidualMacroblockBits = "010" + "11" + "1111" + "000011001" +
		highFrameMBAFFP16x16LumaChromaResidualTailBits
	highFrameMBAFFP8x16LumaChromaResidualMacroblockBits = "011" + "11" + "1111" + "000011001" +
		highFrameMBAFFP16x16LumaChromaResidualTailBits
	highFrameMBAFFP8x8LumaChromaResidualMacroblockBits = "00100" + "1111" + "1111" + "11111111" + "000011001" +
		highFrameMBAFFP16x16LumaChromaResidualTailBits
	highFrameMBAFFP16x8NoResidualMacroblockBits   = "010" + "11" + "1111" + "1"
	highFrameMBAFFP8x16NoResidualMacroblockBits   = "011" + "11" + "1111" + "1"
	highFrameMBAFFP8x8NoResidualMacroblockBits    = "00100" + "1111" + "1111" + "11111111" + "1"
	highFrameMBAFFP16x8LumaResidualMacroblockBits = "010" + "11" + "1111" + "011" +
		highFrameMBAFFP16x16LumaResidualTailBits
	highFrameMBAFFP8x16LumaResidualMacroblockBits = "011" + "11" + "1111" + "011" +
		highFrameMBAFFP16x16LumaResidualTailBits
	highFrameMBAFFP8x8LumaResidualMacroblockBits = "00100" + "1111" + "1111" + "11111111" + "011" +
		highFrameMBAFFP16x16LumaResidualTailBits
	highFrameMBAFFP16x8NoResidualPayloadBits = "11" + highFrameMBAFFP16x8NoResidualMacroblockBits +
		"1" + highFrameMBAFFP16x8NoResidualMacroblockBits
	highFrameMBAFFP8x16NoResidualPayloadBits = "11" + highFrameMBAFFP8x16NoResidualMacroblockBits +
		"1" + highFrameMBAFFP8x16NoResidualMacroblockBits
	highFrameMBAFFP8x8NoResidualPayloadBits = "11" + highFrameMBAFFP8x8NoResidualMacroblockBits +
		"1" + highFrameMBAFFP8x8NoResidualMacroblockBits
	highFrameMBAFFP16x8LumaResidualPayloadBits = "11" + highFrameMBAFFP16x8LumaResidualMacroblockBits +
		"1" + highFrameMBAFFP16x8LumaResidualMacroblockBits
	highFrameMBAFFP8x16LumaResidualPayloadBits = "11" + highFrameMBAFFP8x16LumaResidualMacroblockBits +
		"1" + highFrameMBAFFP8x16LumaResidualMacroblockBits
	highFrameMBAFFP8x8LumaResidualPayloadBits = "11" + highFrameMBAFFP8x8LumaResidualMacroblockBits +
		"1" + highFrameMBAFFP8x8LumaResidualMacroblockBits
	highFrameMBAFFP16x8LumaChromaResidualPayloadBits = "11" + highFrameMBAFFP16x8LumaChromaResidualMacroblockBits +
		"1" + highFrameMBAFFP16x8LumaChromaResidualMacroblockBits
	highFrameMBAFFP8x16LumaChromaResidualPayloadBits = "11" + highFrameMBAFFP8x16LumaChromaResidualMacroblockBits +
		"1" + highFrameMBAFFP8x16LumaChromaResidualMacroblockBits
	highFrameMBAFFP8x8LumaChromaResidualPayloadBits = "11" + highFrameMBAFFP8x8LumaChromaResidualMacroblockBits +
		"1" + highFrameMBAFFP8x8LumaChromaResidualMacroblockBits
	highFrameMBAFFFrameP16x16LumaChromaResidualMacroblockBits = "1" + "11" + "000011001" +
		highFrameMBAFFP16x16LumaChromaResidualTailBits
	highFrameMBAFFFrameP16x8LumaChromaResidualMacroblockBits = "010" + "1111" + "000011001" +
		highFrameMBAFFP16x16LumaChromaResidualTailBits
	highFrameMBAFFFrameP8x16LumaChromaResidualMacroblockBits = "011" + "1111" + "000011001" +
		highFrameMBAFFP16x16LumaChromaResidualTailBits
	highFrameMBAFFFrameP8x8LumaChromaResidualMacroblockBits = "00100" + "1111" + "11111111" + "000011001" +
		highFrameMBAFFP16x16LumaChromaResidualTailBits
	highFrameMBAFFFrameP16x16NoResidualMacroblockBits   = "1" + "11" + "1"
	highFrameMBAFFFrameP16x16LumaResidualMacroblockBits = "1" + "11" + "011" + highFrameMBAFFP16x16LumaResidualTailBits
	highFrameMBAFFFrameP16x8NoResidualMacroblockBits    = "010" + "1111" + "1"
	highFrameMBAFFFrameP8x16NoResidualMacroblockBits    = "011" + "1111" + "1"
	highFrameMBAFFFrameP8x8NoResidualMacroblockBits     = "00100" + "1111" + "11111111" + "1"
	highFrameMBAFFFrameP16x8LumaResidualMacroblockBits  = "010" + "1111" + "011" +
		highFrameMBAFFP16x16LumaResidualTailBits
	highFrameMBAFFFrameP8x16LumaResidualMacroblockBits = "011" + "1111" + "011" +
		highFrameMBAFFP16x16LumaResidualTailBits
	highFrameMBAFFFrameP8x8LumaResidualMacroblockBits = "00100" + "1111" + "11111111" + "011" +
		highFrameMBAFFP16x16LumaResidualTailBits
	highFrameMBAFFFrameP16x16NoResidualPayloadBits = "10" + highFrameMBAFFFrameP16x16NoResidualMacroblockBits +
		"1" + highFrameMBAFFFrameP16x16NoResidualMacroblockBits
	highFrameMBAFFFrameP16x16LumaResidualPayloadBits = "10" + highFrameMBAFFFrameP16x16LumaResidualMacroblockBits +
		"1" + highFrameMBAFFFrameP16x16LumaResidualMacroblockBits
	highFrameMBAFFFrameP16x8NoResidualPayloadBits = "10" + highFrameMBAFFFrameP16x8NoResidualMacroblockBits +
		"1" + highFrameMBAFFFrameP16x8NoResidualMacroblockBits
	highFrameMBAFFFrameP8x16NoResidualPayloadBits = "10" + highFrameMBAFFFrameP8x16NoResidualMacroblockBits +
		"1" + highFrameMBAFFFrameP8x16NoResidualMacroblockBits
	highFrameMBAFFFrameP8x8NoResidualPayloadBits = "10" + highFrameMBAFFFrameP8x8NoResidualMacroblockBits +
		"1" + highFrameMBAFFFrameP8x8NoResidualMacroblockBits
	highFrameMBAFFFrameP16x8LumaResidualPayloadBits = "10" + highFrameMBAFFFrameP16x8LumaResidualMacroblockBits +
		"1" + highFrameMBAFFFrameP16x8LumaResidualMacroblockBits
	highFrameMBAFFFrameP8x16LumaResidualPayloadBits = "10" + highFrameMBAFFFrameP8x16LumaResidualMacroblockBits +
		"1" + highFrameMBAFFFrameP8x16LumaResidualMacroblockBits
	highFrameMBAFFFrameP8x8LumaResidualPayloadBits = "10" + highFrameMBAFFFrameP8x8LumaResidualMacroblockBits +
		"1" + highFrameMBAFFFrameP8x8LumaResidualMacroblockBits
	highFrameMBAFFFrameP16x16LumaChromaResidualPayloadBits = "10" + highFrameMBAFFFrameP16x16LumaChromaResidualMacroblockBits +
		"1" + highFrameMBAFFFrameP16x16LumaChromaResidualMacroblockBits
	highFrameMBAFFFrameP16x8LumaChromaResidualPayloadBits = "10" + highFrameMBAFFFrameP16x8LumaChromaResidualMacroblockBits +
		"1" + highFrameMBAFFFrameP16x8LumaChromaResidualMacroblockBits
	highFrameMBAFFFrameP8x16LumaChromaResidualPayloadBits = "10" + highFrameMBAFFFrameP8x16LumaChromaResidualMacroblockBits +
		"1" + highFrameMBAFFFrameP8x16LumaChromaResidualMacroblockBits
	highFrameMBAFFFrameP8x8LumaChromaResidualPayloadBits = "10" + highFrameMBAFFFrameP8x8LumaChromaResidualMacroblockBits +
		"1" + highFrameMBAFFFrameP8x8LumaChromaResidualMacroblockBits
)

type highFrameMBAFFP16x16NoResidualCase struct {
	name         string
	bitDepth     int
	bitstreamMD5 string
	refFrameMD5  string
	pFrameMD5    string
	rawVideoMD5  string
}

type highFrameMBAFFPSkipNoResidualCase = highFrameMBAFFP16x16NoResidualCase
type highFrameMBAFFP16x16LumaResidualCase = highFrameMBAFFP16x16NoResidualCase
type highFrameMBAFFP16x16LumaChromaResidualCase = highFrameMBAFFP16x16NoResidualCase
type highFrameMBAFFFrameP16x16NoResidualCase = highFrameMBAFFP16x16NoResidualCase
type highFrameMBAFFFrameP16x16LumaResidualCase = highFrameMBAFFP16x16NoResidualCase

type highFrameMBAFFP16x16DeblockCase struct {
	name                       string
	bitDepth                   int
	pskip                      bool
	fieldFlag                  uint32
	cbp                        uint32
	payloadBits                string
	residualTailBits           string
	disableDeblockingFilterIDC uint32
	deblockMode                int32
	bitstreamMD5               string
	refFrameMD5                string
	pFrameMD5                  string
	rawVideoMD5                string
}

type highFrameMBAFFPartitionedPDeblockCase struct {
	name                       string
	bitDepth                   int
	mbType                     uint32
	fieldFlag                  uint32
	cbp                        uint32
	payloadBits                string
	residualTailBits           string
	disableDeblockingFilterIDC uint32
	deblockMode                int32
	bitstreamMD5               string
	refFrameMD5                string
	pFrameMD5                  string
	rawVideoMD5                string
}

type highFrameMBAFFPartitionedPLumaChromaResidualCase struct {
	name         string
	bitDepth     int
	mbType       uint32
	payloadBits  string
	bitstreamMD5 string
	refFrameMD5  string
	pFrameMD5    string
	rawVideoMD5  string
}

type highFrameMBAFFPartitionedPSparseResidualCase struct {
	name             string
	bitDepth         int
	mbType           uint32
	fieldFlag        uint32
	cbp              uint32
	residualTailBits string
	payloadBits      string
	bitstreamMD5     string
	refFrameMD5      string
	pFrameMD5        string
	rawVideoMD5      string
}

type highFrameMBAFFFrameCodedPLumaChromaResidualCase = highFrameMBAFFPartitionedPLumaChromaResidualCase

func TestHigh1214FrameMBAFFPSkipNoResidualFixtureSyntax(t *testing.T) {
	for _, tt := range highFrameMBAFFPSkipNoResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFPSkipNoResidualFixture(tt.bitDepth)
			sum := md5.Sum(data)
			if got := hex.EncodeToString(sum[:]); got != tt.bitstreamMD5 {
				t.Fatalf("%s frame-MBAFF P-skip bitstream md5 = %s, want %s", tt.name, got, tt.bitstreamMD5)
			}
			assertHighFrameMBAFFPSkipNoResidualFixtureSyntax(t, data, tt)
		})
	}
}

func TestHigh1214FrameMBAFFP16x16NoResidualFixtureSyntax(t *testing.T) {
	for _, tt := range highFrameMBAFFP16x16NoResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFP16x16NoResidualFixture(tt.bitDepth)
			sum := md5.Sum(data)
			if got := hex.EncodeToString(sum[:]); got != tt.bitstreamMD5 {
				t.Fatalf("%s frame-MBAFF P16x16 bitstream md5 = %s, want %s", tt.name, got, tt.bitstreamMD5)
			}
			assertHighFrameMBAFFP16x16NoResidualFixtureSyntax(t, data, tt)
		})
	}
}

func TestHigh1214FrameMBAFFP16x16LumaResidualFixtureSyntax(t *testing.T) {
	for _, tt := range highFrameMBAFFP16x16LumaResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFP16x16LumaResidualFixture(tt.bitDepth)
			sum := md5.Sum(data)
			if got := hex.EncodeToString(sum[:]); got != tt.bitstreamMD5 {
				t.Fatalf("%s frame-MBAFF P16x16 luma-residual bitstream md5 = %s, want %s", tt.name, got, tt.bitstreamMD5)
			}
			assertHighFrameMBAFFP16x16LumaResidualFixtureSyntax(t, data, tt)
		})
	}
}

func TestHigh1214FrameMBAFFFrameP16x16NoResidualFixtureSyntax(t *testing.T) {
	for _, tt := range highFrameMBAFFFrameP16x16NoResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFFrameP16x16NoResidualFixture(tt.bitDepth)
			sum := md5.Sum(data)
			if got := hex.EncodeToString(sum[:]); got != tt.bitstreamMD5 {
				t.Fatalf("%s frame-MBAFF frame-coded P16x16 no-residual bitstream md5 = %s, want %s", tt.name, got, tt.bitstreamMD5)
			}
			assertHighFrameMBAFFFrameP16x16NoResidualFixtureSyntax(t, data, tt)
		})
	}
}

func TestHigh1214FrameMBAFFFrameP16x16LumaResidualFixtureSyntax(t *testing.T) {
	for _, tt := range highFrameMBAFFFrameP16x16LumaResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFFrameP16x16LumaResidualFixture(tt.bitDepth)
			sum := md5.Sum(data)
			if got := hex.EncodeToString(sum[:]); got != tt.bitstreamMD5 {
				t.Fatalf("%s frame-MBAFF frame-coded P16x16 luma-residual bitstream md5 = %s, want %s", tt.name, got, tt.bitstreamMD5)
			}
			assertHighFrameMBAFFFrameP16x16LumaResidualFixtureSyntax(t, data, tt)
		})
	}
}

func TestHigh1214FrameMBAFFP16x16DeblockFixtureSyntax(t *testing.T) {
	for _, tt := range highFrameMBAFFP16x16DeblockCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFP16x16DeblockFixture(tt)
			sum := md5.Sum(data)
			if got := hex.EncodeToString(sum[:]); got != tt.bitstreamMD5 {
				t.Fatalf("%s frame-MBAFF P16x16 deblock bitstream md5 = %s, want %s", tt.name, got, tt.bitstreamMD5)
			}
			assertHighFrameMBAFFP16x16DeblockFixtureSyntax(t, data, tt)
		})
	}
}

func TestHigh1214FrameMBAFFPartitionedPDeblockFixtureSyntax(t *testing.T) {
	for _, tt := range highFrameMBAFFPartitionedPDeblockCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFPartitionedPDeblockFixture(tt)
			sum := md5.Sum(data)
			if got := hex.EncodeToString(sum[:]); got != tt.bitstreamMD5 {
				t.Fatalf("%s frame-MBAFF partitioned P deblock bitstream md5 = %s, want %s", tt.name, got, tt.bitstreamMD5)
			}
			assertHighFrameMBAFFPartitionedPDeblockFixtureSyntax(t, data, tt)
		})
	}
}

func TestHigh1214FrameMBAFFPartitionedPSparseResidualFixtureSyntax(t *testing.T) {
	for _, tt := range highFrameMBAFFPartitionedPSparseResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFPartitionedPSparseResidualFixture(tt.bitDepth, tt.payloadBits)
			sum := md5.Sum(data)
			if got := hex.EncodeToString(sum[:]); got != tt.bitstreamMD5 {
				t.Fatalf("%s frame-MBAFF partitioned P sparse-residual bitstream md5 = %s, want %s", tt.name, got, tt.bitstreamMD5)
			}
			assertHighFrameMBAFFPartitionedPSparseResidualFixtureSyntax(t, data, tt)
		})
	}
}

func TestHigh1214FrameMBAFFP16x16LumaChromaResidualFixtureSyntax(t *testing.T) {
	for _, tt := range highFrameMBAFFP16x16LumaChromaResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFP16x16LumaChromaResidualFixture(tt.bitDepth)
			sum := md5.Sum(data)
			if got := hex.EncodeToString(sum[:]); got != tt.bitstreamMD5 {
				t.Fatalf("%s frame-MBAFF P16x16 luma+chroma-residual bitstream md5 = %s, want %s", tt.name, got, tt.bitstreamMD5)
			}
			assertHighFrameMBAFFP16x16LumaChromaResidualFixtureSyntax(t, data, tt)
		})
	}
}

func TestHigh1214FrameMBAFFPartitionedPLumaChromaResidualFixtureSyntax(t *testing.T) {
	for _, tt := range highFrameMBAFFPartitionedPLumaChromaResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFPartitionedPLumaChromaResidualFixture(tt.bitDepth, tt.payloadBits)
			sum := md5.Sum(data)
			if got := hex.EncodeToString(sum[:]); got != tt.bitstreamMD5 {
				t.Fatalf("%s frame-MBAFF partitioned P luma+chroma-residual bitstream md5 = %s, want %s", tt.name, got, tt.bitstreamMD5)
			}
			assertHighFrameMBAFFPartitionedPLumaChromaResidualFixtureSyntax(t, data, tt)
		})
	}
}

func TestHigh1214FrameMBAFFFrameCodedPLumaChromaResidualFixtureSyntax(t *testing.T) {
	for _, tt := range highFrameMBAFFFrameCodedPLumaChromaResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFFrameCodedPLumaChromaResidualFixture(tt.bitDepth, tt.payloadBits)
			sum := md5.Sum(data)
			if got := hex.EncodeToString(sum[:]); got != tt.bitstreamMD5 {
				t.Fatalf("%s frame-MBAFF frame-coded P luma+chroma-residual bitstream md5 = %s, want %s", tt.name, got, tt.bitstreamMD5)
			}
			assertHighFrameMBAFFFrameCodedPLumaChromaResidualFixtureSyntax(t, data, tt)
		})
	}
}

func TestDecodeAnnexBHigh1214FrameMBAFFPSkipNoResidualFrames(t *testing.T) {
	for _, tt := range highFrameMBAFFPSkipNoResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFPSkipNoResidualFixture(tt.bitDepth)
			assertHighFrameMBAFFPSkipNoResidualFixtureSyntax(t, data, tt)

			frames, err := NewDecoder().DecodeAnnexBFrames(data)
			if err != nil {
				t.Fatalf("decode %s frame-MBAFF P-skip Annex B: %v", tt.name, err)
			}
			assertHighFrameMBAFFPSkipNoResidualFrames(t, frames, tt)
		})
	}
}

func TestDecodeAnnexBHigh1214FrameMBAFFP16x16NoResidualFrames(t *testing.T) {
	for _, tt := range highFrameMBAFFP16x16NoResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFP16x16NoResidualFixture(tt.bitDepth)
			assertHighFrameMBAFFP16x16NoResidualFixtureSyntax(t, data, tt)

			frames, err := NewDecoder().DecodeAnnexBFrames(data)
			if err != nil {
				t.Fatalf("decode %s frame-MBAFF P16x16 Annex B: %v", tt.name, err)
			}
			assertHighFrameMBAFFP16x16NoResidualFrames(t, frames, tt)
		})
	}
}

func TestDecodeAnnexBHigh1214FrameMBAFFP16x16LumaResidualFrames(t *testing.T) {
	for _, tt := range highFrameMBAFFP16x16LumaResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFP16x16LumaResidualFixture(tt.bitDepth)
			assertHighFrameMBAFFP16x16LumaResidualFixtureSyntax(t, data, tt)

			frames, err := NewDecoder().DecodeAnnexBFrames(data)
			if err != nil {
				t.Fatalf("decode %s frame-MBAFF P16x16 luma-residual Annex B: %v", tt.name, err)
			}
			assertHighFrameMBAFFP16x16LumaResidualFrames(t, frames, tt)
		})
	}
}

func TestDecodeAnnexBHigh1214FrameMBAFFFrameP16x16NoResidualFrames(t *testing.T) {
	for _, tt := range highFrameMBAFFFrameP16x16NoResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFFrameP16x16NoResidualFixture(tt.bitDepth)
			assertHighFrameMBAFFFrameP16x16NoResidualFixtureSyntax(t, data, tt)

			frames, err := NewDecoder().DecodeAnnexBFrames(data)
			if err != nil {
				t.Fatalf("decode %s frame-MBAFF frame-coded P16x16 no-residual Annex B: %v", tt.name, err)
			}
			assertHighFrameMBAFFFrameP16x16NoResidualFrames(t, frames, tt)
		})
	}
}

func TestDecodeAnnexBHigh1214FrameMBAFFFrameP16x16LumaResidualFrames(t *testing.T) {
	for _, tt := range highFrameMBAFFFrameP16x16LumaResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFFrameP16x16LumaResidualFixture(tt.bitDepth)
			assertHighFrameMBAFFFrameP16x16LumaResidualFixtureSyntax(t, data, tt)

			frames, err := NewDecoder().DecodeAnnexBFrames(data)
			if err != nil {
				t.Fatalf("decode %s frame-MBAFF frame-coded P16x16 luma-residual Annex B: %v", tt.name, err)
			}
			assertHighFrameMBAFFFrameP16x16LumaResidualFrames(t, frames, tt)
		})
	}
}

func TestDecodeAnnexBHigh1214FrameMBAFFP16x16DeblockFrames(t *testing.T) {
	for _, tt := range highFrameMBAFFP16x16DeblockCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFP16x16DeblockFixture(tt)
			assertHighFrameMBAFFP16x16DeblockFixtureSyntax(t, data, tt)

			frames, err := NewDecoder().DecodeAnnexBFrames(data)
			if err != nil {
				t.Fatalf("decode %s frame-MBAFF P16x16 deblock Annex B: %v", tt.name, err)
			}
			assertHighFrameMBAFFP16x16DeblockFrames(t, frames, tt)
		})
	}
}

func TestDecodeAnnexBHigh1214FrameMBAFFPartitionedPDeblockFrames(t *testing.T) {
	for _, tt := range highFrameMBAFFPartitionedPDeblockCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFPartitionedPDeblockFixture(tt)
			assertHighFrameMBAFFPartitionedPDeblockFixtureSyntax(t, data, tt)

			frames, err := NewDecoder().DecodeAnnexBFrames(data)
			if err != nil {
				t.Fatalf("decode %s frame-MBAFF partitioned P deblock Annex B: %v", tt.name, err)
			}
			assertHighFrameMBAFFPartitionedPDeblockFrames(t, frames, tt)
		})
	}
}

func TestDecodeAnnexBHigh1214FrameMBAFFPartitionedPSparseResidualFrames(t *testing.T) {
	for _, tt := range highFrameMBAFFPartitionedPSparseResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFPartitionedPSparseResidualFixture(tt.bitDepth, tt.payloadBits)
			assertHighFrameMBAFFPartitionedPSparseResidualFixtureSyntax(t, data, tt)

			frames, err := NewDecoder().DecodeAnnexBFrames(data)
			if err != nil {
				t.Fatalf("decode %s frame-MBAFF partitioned P sparse-residual Annex B: %v", tt.name, err)
			}
			assertHighFrameMBAFFPartitionedPSparseResidualFrames(t, frames, tt)
		})
	}
}

func TestDecodeAnnexBHigh1214FrameMBAFFP16x16LumaChromaResidualFrames(t *testing.T) {
	for _, tt := range highFrameMBAFFP16x16LumaChromaResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFP16x16LumaChromaResidualFixture(tt.bitDepth)
			assertHighFrameMBAFFP16x16LumaChromaResidualFixtureSyntax(t, data, tt)

			frames, err := NewDecoder().DecodeAnnexBFrames(data)
			if err != nil {
				t.Fatalf("decode %s frame-MBAFF P16x16 luma+chroma-residual Annex B: %v", tt.name, err)
			}
			assertHighFrameMBAFFP16x16LumaChromaResidualFrames(t, frames, tt)
		})
	}
}

func TestDecodeAnnexBHigh1214FrameMBAFFPartitionedPLumaChromaResidualFrames(t *testing.T) {
	for _, tt := range highFrameMBAFFPartitionedPLumaChromaResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFPartitionedPLumaChromaResidualFixture(tt.bitDepth, tt.payloadBits)
			assertHighFrameMBAFFPartitionedPLumaChromaResidualFixtureSyntax(t, data, tt)

			frames, err := NewDecoder().DecodeAnnexBFrames(data)
			if err != nil {
				t.Fatalf("decode %s frame-MBAFF partitioned P luma+chroma-residual Annex B: %v", tt.name, err)
			}
			assertHighFrameMBAFFPartitionedPLumaChromaResidualFrames(t, frames, tt)
		})
	}
}

func TestDecodeAnnexBHigh1214FrameMBAFFFrameCodedPLumaChromaResidualFrames(t *testing.T) {
	for _, tt := range highFrameMBAFFFrameCodedPLumaChromaResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFFrameCodedPLumaChromaResidualFixture(tt.bitDepth, tt.payloadBits)
			assertHighFrameMBAFFFrameCodedPLumaChromaResidualFixtureSyntax(t, data, tt)

			frames, err := NewDecoder().DecodeAnnexBFrames(data)
			if err != nil {
				t.Fatalf("decode %s frame-MBAFF frame-coded P luma+chroma-residual Annex B: %v", tt.name, err)
			}
			assertHighFrameMBAFFFrameCodedPLumaChromaResidualFrames(t, frames, tt)
		})
	}
}

func TestDecodeAVCHigh1214FrameMBAFFPSkipNoResidualFrames(t *testing.T) {
	for _, tt := range highFrameMBAFFPSkipNoResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFPSkipNoResidualFixture(tt.bitDepth)
			assertHighFrameMBAFFPSkipNoResidualFixtureSyntax(t, data, tt)

			for _, nalLengthSize := range []int{2, 3, 4} {
				frames, err := NewDecoder().DecodeAVCFrames(annexBToAVC(t, data, nalLengthSize), nalLengthSize)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: decode %s frame-MBAFF P-skip AVC: %v", nalLengthSize, tt.name, err)
				}
				assertHighFrameMBAFFPSkipNoResidualFrames(t, frames, tt)
			}
		})
	}
}

func TestDecodeAVCHigh1214FrameMBAFFP16x16NoResidualFrames(t *testing.T) {
	for _, tt := range highFrameMBAFFP16x16NoResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFP16x16NoResidualFixture(tt.bitDepth)
			assertHighFrameMBAFFP16x16NoResidualFixtureSyntax(t, data, tt)

			for _, nalLengthSize := range []int{2, 3, 4} {
				frames, err := NewDecoder().DecodeAVCFrames(annexBToAVC(t, data, nalLengthSize), nalLengthSize)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: decode %s frame-MBAFF P16x16 AVC: %v", nalLengthSize, tt.name, err)
				}
				assertHighFrameMBAFFP16x16NoResidualFrames(t, frames, tt)
			}
		})
	}
}

func TestDecodeAVCHigh1214FrameMBAFFP16x16LumaResidualFrames(t *testing.T) {
	for _, tt := range highFrameMBAFFP16x16LumaResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFP16x16LumaResidualFixture(tt.bitDepth)
			assertHighFrameMBAFFP16x16LumaResidualFixtureSyntax(t, data, tt)

			for _, nalLengthSize := range []int{2, 3, 4} {
				frames, err := NewDecoder().DecodeAVCFrames(annexBToAVC(t, data, nalLengthSize), nalLengthSize)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: decode %s frame-MBAFF P16x16 luma-residual AVC: %v", nalLengthSize, tt.name, err)
				}
				assertHighFrameMBAFFP16x16LumaResidualFrames(t, frames, tt)
			}
		})
	}
}

func TestDecodeAVCHigh1214FrameMBAFFFrameP16x16NoResidualFrames(t *testing.T) {
	for _, tt := range highFrameMBAFFFrameP16x16NoResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFFrameP16x16NoResidualFixture(tt.bitDepth)
			assertHighFrameMBAFFFrameP16x16NoResidualFixtureSyntax(t, data, tt)

			for _, nalLengthSize := range []int{2, 3, 4} {
				frames, err := NewDecoder().DecodeAVCFrames(annexBToAVC(t, data, nalLengthSize), nalLengthSize)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: decode %s frame-MBAFF frame-coded P16x16 no-residual AVC: %v", nalLengthSize, tt.name, err)
				}
				assertHighFrameMBAFFFrameP16x16NoResidualFrames(t, frames, tt)
			}
		})
	}
}

func TestDecodeAVCHigh1214FrameMBAFFFrameP16x16LumaResidualFrames(t *testing.T) {
	for _, tt := range highFrameMBAFFFrameP16x16LumaResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFFrameP16x16LumaResidualFixture(tt.bitDepth)
			assertHighFrameMBAFFFrameP16x16LumaResidualFixtureSyntax(t, data, tt)

			for _, nalLengthSize := range []int{2, 3, 4} {
				frames, err := NewDecoder().DecodeAVCFrames(annexBToAVC(t, data, nalLengthSize), nalLengthSize)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: decode %s frame-MBAFF frame-coded P16x16 luma-residual AVC: %v", nalLengthSize, tt.name, err)
				}
				assertHighFrameMBAFFFrameP16x16LumaResidualFrames(t, frames, tt)
			}
		})
	}
}

func TestDecodeAVCHigh1214FrameMBAFFP16x16DeblockFrames(t *testing.T) {
	for _, tt := range highFrameMBAFFP16x16DeblockCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFP16x16DeblockFixture(tt)
			assertHighFrameMBAFFP16x16DeblockFixtureSyntax(t, data, tt)

			for _, nalLengthSize := range []int{2, 3, 4} {
				frames, err := NewDecoder().DecodeAVCFrames(annexBToAVC(t, data, nalLengthSize), nalLengthSize)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: decode %s frame-MBAFF P16x16 deblock AVC: %v", nalLengthSize, tt.name, err)
				}
				assertHighFrameMBAFFP16x16DeblockFrames(t, frames, tt)
			}
		})
	}
}

func TestDecodeAVCHigh1214FrameMBAFFPartitionedPDeblockFrames(t *testing.T) {
	for _, tt := range highFrameMBAFFPartitionedPDeblockCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFPartitionedPDeblockFixture(tt)
			assertHighFrameMBAFFPartitionedPDeblockFixtureSyntax(t, data, tt)

			for _, nalLengthSize := range []int{2, 3, 4} {
				frames, err := NewDecoder().DecodeAVCFrames(annexBToAVC(t, data, nalLengthSize), nalLengthSize)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: decode %s frame-MBAFF partitioned P deblock AVC: %v", nalLengthSize, tt.name, err)
				}
				assertHighFrameMBAFFPartitionedPDeblockFrames(t, frames, tt)
			}
		})
	}
}

func TestDecodeAVCHigh1214FrameMBAFFPartitionedPSparseResidualFrames(t *testing.T) {
	for _, tt := range highFrameMBAFFPartitionedPSparseResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFPartitionedPSparseResidualFixture(tt.bitDepth, tt.payloadBits)
			assertHighFrameMBAFFPartitionedPSparseResidualFixtureSyntax(t, data, tt)

			for _, nalLengthSize := range []int{2, 3, 4} {
				frames, err := NewDecoder().DecodeAVCFrames(annexBToAVC(t, data, nalLengthSize), nalLengthSize)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: decode %s frame-MBAFF partitioned P sparse-residual AVC: %v", nalLengthSize, tt.name, err)
				}
				assertHighFrameMBAFFPartitionedPSparseResidualFrames(t, frames, tt)
			}
		})
	}
}

func TestDecodeAVCHigh1214FrameMBAFFP16x16LumaChromaResidualFrames(t *testing.T) {
	for _, tt := range highFrameMBAFFP16x16LumaChromaResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFP16x16LumaChromaResidualFixture(tt.bitDepth)
			assertHighFrameMBAFFP16x16LumaChromaResidualFixtureSyntax(t, data, tt)

			for _, nalLengthSize := range []int{2, 3, 4} {
				frames, err := NewDecoder().DecodeAVCFrames(annexBToAVC(t, data, nalLengthSize), nalLengthSize)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: decode %s frame-MBAFF P16x16 luma+chroma-residual AVC: %v", nalLengthSize, tt.name, err)
				}
				assertHighFrameMBAFFP16x16LumaChromaResidualFrames(t, frames, tt)
			}
		})
	}
}

func TestDecodeAVCHigh1214FrameMBAFFPartitionedPLumaChromaResidualFrames(t *testing.T) {
	for _, tt := range highFrameMBAFFPartitionedPLumaChromaResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFPartitionedPLumaChromaResidualFixture(tt.bitDepth, tt.payloadBits)
			assertHighFrameMBAFFPartitionedPLumaChromaResidualFixtureSyntax(t, data, tt)

			for _, nalLengthSize := range []int{2, 3, 4} {
				frames, err := NewDecoder().DecodeAVCFrames(annexBToAVC(t, data, nalLengthSize), nalLengthSize)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: decode %s frame-MBAFF partitioned P luma+chroma-residual AVC: %v", nalLengthSize, tt.name, err)
				}
				assertHighFrameMBAFFPartitionedPLumaChromaResidualFrames(t, frames, tt)
			}
		})
	}
}

func TestDecodeAVCHigh1214FrameMBAFFFrameCodedPLumaChromaResidualFrames(t *testing.T) {
	for _, tt := range highFrameMBAFFFrameCodedPLumaChromaResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFFrameCodedPLumaChromaResidualFixture(tt.bitDepth, tt.payloadBits)
			assertHighFrameMBAFFFrameCodedPLumaChromaResidualFixtureSyntax(t, data, tt)

			for _, nalLengthSize := range []int{2, 3, 4} {
				frames, err := NewDecoder().DecodeAVCFrames(annexBToAVC(t, data, nalLengthSize), nalLengthSize)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: decode %s frame-MBAFF frame-coded P luma+chroma-residual AVC: %v", nalLengthSize, tt.name, err)
				}
				assertHighFrameMBAFFFrameCodedPLumaChromaResidualFrames(t, frames, tt)
			}
		})
	}
}

func TestDecodeAVCWithConfigurationRecordHigh1214FrameMBAFFPSkipNoResidualFrames(t *testing.T) {
	for _, tt := range highFrameMBAFFPSkipNoResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFPSkipNoResidualFixture(tt.bitDepth)
			assertHighFrameMBAFFPSkipNoResidualFixtureSyntax(t, data, tt)

			for _, nalLengthSize := range []int{2, 3, 4} {
				config, packet := annexBToAVCConfigAndPacket(t, data, nalLengthSize)
				frames, err := NewDecoder().DecodeAVCFramesWithConfigurationRecord(config, packet)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: decode %s frame-MBAFF P-skip configured AVC: %v", nalLengthSize, tt.name, err)
				}
				assertHighFrameMBAFFPSkipNoResidualFrames(t, frames, tt)
			}
		})
	}
}

func TestDecodeAVCWithConfigurationRecordHigh1214FrameMBAFFP16x16NoResidualFrames(t *testing.T) {
	for _, tt := range highFrameMBAFFP16x16NoResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFP16x16NoResidualFixture(tt.bitDepth)
			assertHighFrameMBAFFP16x16NoResidualFixtureSyntax(t, data, tt)

			for _, nalLengthSize := range []int{2, 3, 4} {
				config, packet := annexBToAVCConfigAndPacket(t, data, nalLengthSize)
				frames, err := NewDecoder().DecodeAVCFramesWithConfigurationRecord(config, packet)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: decode %s frame-MBAFF P16x16 configured AVC: %v", nalLengthSize, tt.name, err)
				}
				assertHighFrameMBAFFP16x16NoResidualFrames(t, frames, tt)
			}
		})
	}
}

func TestDecodeAVCWithConfigurationRecordHigh1214FrameMBAFFP16x16LumaResidualFrames(t *testing.T) {
	for _, tt := range highFrameMBAFFP16x16LumaResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFP16x16LumaResidualFixture(tt.bitDepth)
			assertHighFrameMBAFFP16x16LumaResidualFixtureSyntax(t, data, tt)

			for _, nalLengthSize := range []int{2, 3, 4} {
				config, packet := annexBToAVCConfigAndPacket(t, data, nalLengthSize)
				frames, err := NewDecoder().DecodeAVCFramesWithConfigurationRecord(config, packet)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: decode %s frame-MBAFF P16x16 luma-residual configured AVC: %v", nalLengthSize, tt.name, err)
				}
				assertHighFrameMBAFFP16x16LumaResidualFrames(t, frames, tt)
			}
		})
	}
}

func TestDecodeAVCWithConfigurationRecordHigh1214FrameMBAFFFrameP16x16NoResidualFrames(t *testing.T) {
	for _, tt := range highFrameMBAFFFrameP16x16NoResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFFrameP16x16NoResidualFixture(tt.bitDepth)
			assertHighFrameMBAFFFrameP16x16NoResidualFixtureSyntax(t, data, tt)

			for _, nalLengthSize := range []int{2, 3, 4} {
				config, packet := annexBToAVCConfigAndPacket(t, data, nalLengthSize)
				frames, err := NewDecoder().DecodeAVCFramesWithConfigurationRecord(config, packet)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: decode %s frame-MBAFF frame-coded P16x16 no-residual configured AVC: %v", nalLengthSize, tt.name, err)
				}
				assertHighFrameMBAFFFrameP16x16NoResidualFrames(t, frames, tt)
			}
		})
	}
}

func TestDecodeAVCWithConfigurationRecordHigh1214FrameMBAFFFrameP16x16LumaResidualFrames(t *testing.T) {
	for _, tt := range highFrameMBAFFFrameP16x16LumaResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFFrameP16x16LumaResidualFixture(tt.bitDepth)
			assertHighFrameMBAFFFrameP16x16LumaResidualFixtureSyntax(t, data, tt)

			for _, nalLengthSize := range []int{2, 3, 4} {
				config, packet := annexBToAVCConfigAndPacket(t, data, nalLengthSize)
				frames, err := NewDecoder().DecodeAVCFramesWithConfigurationRecord(config, packet)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: decode %s frame-MBAFF frame-coded P16x16 luma-residual configured AVC: %v", nalLengthSize, tt.name, err)
				}
				assertHighFrameMBAFFFrameP16x16LumaResidualFrames(t, frames, tt)
			}
		})
	}
}

func TestDecodeAVCWithConfigurationRecordHigh1214FrameMBAFFP16x16DeblockFrames(t *testing.T) {
	for _, tt := range highFrameMBAFFP16x16DeblockCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFP16x16DeblockFixture(tt)
			assertHighFrameMBAFFP16x16DeblockFixtureSyntax(t, data, tt)

			for _, nalLengthSize := range []int{2, 3, 4} {
				config, packet := annexBToAVCConfigAndPacket(t, data, nalLengthSize)
				frames, err := NewDecoder().DecodeAVCFramesWithConfigurationRecord(config, packet)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: decode %s frame-MBAFF P16x16 deblock configured AVC: %v", nalLengthSize, tt.name, err)
				}
				assertHighFrameMBAFFP16x16DeblockFrames(t, frames, tt)
			}
		})
	}
}

func TestDecodeAVCWithConfigurationRecordHigh1214FrameMBAFFPartitionedPDeblockFrames(t *testing.T) {
	for _, tt := range highFrameMBAFFPartitionedPDeblockCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFPartitionedPDeblockFixture(tt)
			assertHighFrameMBAFFPartitionedPDeblockFixtureSyntax(t, data, tt)

			for _, nalLengthSize := range []int{2, 3, 4} {
				config, packet := annexBToAVCConfigAndPacket(t, data, nalLengthSize)
				frames, err := NewDecoder().DecodeAVCFramesWithConfigurationRecord(config, packet)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: decode %s frame-MBAFF partitioned P deblock configured AVC: %v", nalLengthSize, tt.name, err)
				}
				assertHighFrameMBAFFPartitionedPDeblockFrames(t, frames, tt)
			}
		})
	}
}

func TestDecodeAVCWithConfigurationRecordHigh1214FrameMBAFFPartitionedPSparseResidualFrames(t *testing.T) {
	for _, tt := range highFrameMBAFFPartitionedPSparseResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFPartitionedPSparseResidualFixture(tt.bitDepth, tt.payloadBits)
			assertHighFrameMBAFFPartitionedPSparseResidualFixtureSyntax(t, data, tt)

			for _, nalLengthSize := range []int{2, 3, 4} {
				config, packet := annexBToAVCConfigAndPacket(t, data, nalLengthSize)
				frames, err := NewDecoder().DecodeAVCFramesWithConfigurationRecord(config, packet)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: decode %s frame-MBAFF partitioned P sparse-residual configured AVC: %v", nalLengthSize, tt.name, err)
				}
				assertHighFrameMBAFFPartitionedPSparseResidualFrames(t, frames, tt)
			}
		})
	}
}

func TestDecodeAVCWithConfigurationRecordHigh1214FrameMBAFFP16x16LumaChromaResidualFrames(t *testing.T) {
	for _, tt := range highFrameMBAFFP16x16LumaChromaResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFP16x16LumaChromaResidualFixture(tt.bitDepth)
			assertHighFrameMBAFFP16x16LumaChromaResidualFixtureSyntax(t, data, tt)

			for _, nalLengthSize := range []int{2, 3, 4} {
				config, packet := annexBToAVCConfigAndPacket(t, data, nalLengthSize)
				frames, err := NewDecoder().DecodeAVCFramesWithConfigurationRecord(config, packet)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: decode %s frame-MBAFF P16x16 luma+chroma-residual configured AVC: %v", nalLengthSize, tt.name, err)
				}
				assertHighFrameMBAFFP16x16LumaChromaResidualFrames(t, frames, tt)
			}
		})
	}
}

func TestDecodeAVCWithConfigurationRecordHigh1214FrameMBAFFPartitionedPLumaChromaResidualFrames(t *testing.T) {
	for _, tt := range highFrameMBAFFPartitionedPLumaChromaResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFPartitionedPLumaChromaResidualFixture(tt.bitDepth, tt.payloadBits)
			assertHighFrameMBAFFPartitionedPLumaChromaResidualFixtureSyntax(t, data, tt)

			for _, nalLengthSize := range []int{2, 3, 4} {
				config, packet := annexBToAVCConfigAndPacket(t, data, nalLengthSize)
				frames, err := NewDecoder().DecodeAVCFramesWithConfigurationRecord(config, packet)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: decode %s frame-MBAFF partitioned P luma+chroma-residual configured AVC: %v", nalLengthSize, tt.name, err)
				}
				assertHighFrameMBAFFPartitionedPLumaChromaResidualFrames(t, frames, tt)
			}
		})
	}
}

func TestDecodeAVCWithConfigurationRecordHigh1214FrameMBAFFFrameCodedPLumaChromaResidualFrames(t *testing.T) {
	for _, tt := range highFrameMBAFFFrameCodedPLumaChromaResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFFrameCodedPLumaChromaResidualFixture(tt.bitDepth, tt.payloadBits)
			assertHighFrameMBAFFFrameCodedPLumaChromaResidualFixtureSyntax(t, data, tt)

			for _, nalLengthSize := range []int{2, 3, 4} {
				config, packet := annexBToAVCConfigAndPacket(t, data, nalLengthSize)
				frames, err := NewDecoder().DecodeAVCFramesWithConfigurationRecord(config, packet)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: decode %s frame-MBAFF frame-coded P luma+chroma-residual configured AVC: %v", nalLengthSize, tt.name, err)
				}
				assertHighFrameMBAFFFrameCodedPLumaChromaResidualFrames(t, frames, tt)
			}
		})
	}
}

func TestFFmpegRawVideoMD5OracleHigh1214FrameMBAFFPSkipNoResidual(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}

	for _, tt := range highFrameMBAFFPSkipNoResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFPSkipNoResidualFixture(tt.bitDepth)
			assertHighFrameMBAFFPSkipNoResidualFixtureSyntax(t, data, tt)
			path := writeTempH264(t, data)
			pixFmt := fmt.Sprintf("yuv420p%dle", tt.bitDepth)

			framemd5 := exec.Command(
				"ffmpeg",
				"-v", "error",
				"-f", "h264",
				"-i", path,
				"-an", "-sn", "-dn",
				"-pix_fmt", pixFmt,
				"-f", "framemd5",
				"-",
			)
			framemd5Out, err := framemd5.Output()
			if err != nil {
				t.Fatalf("ffmpeg framemd5: %v", err)
			}
			for i, want := range []string{tt.refFrameMD5, tt.pFrameMD5} {
				line := []byte(fmt.Sprintf("0, %10d, %10d,        1,     1536, %s", i, i, want))
				if !bytes.Contains(framemd5Out, line) {
					t.Fatalf("missing %q in framemd5:\n%s", line, framemd5Out)
				}
			}

			rawvideo := exec.Command(
				"ffmpeg",
				"-v", "error",
				"-f", "h264",
				"-i", path,
				"-an", "-sn", "-dn",
				"-pix_fmt", pixFmt,
				"-f", "rawvideo",
				"-",
			)
			raw, err := rawvideo.Output()
			if err != nil {
				t.Fatalf("ffmpeg rawvideo: %v", err)
			}
			if len(raw) != 3072 {
				t.Fatalf("rawvideo size = %d, want 3072", len(raw))
			}
			sum := md5.Sum(raw)
			if got := hex.EncodeToString(sum[:]); got != tt.rawVideoMD5 {
				t.Fatalf("rawvideo md5 = %s, want %s", got, tt.rawVideoMD5)
			}
		})
	}
}

func TestFFmpegRawVideoMD5OracleHigh1214FrameMBAFFP16x16NoResidual(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}

	for _, tt := range highFrameMBAFFP16x16NoResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFP16x16NoResidualFixture(tt.bitDepth)
			assertHighFrameMBAFFP16x16NoResidualFixtureSyntax(t, data, tt)
			path := writeTempH264(t, data)
			pixFmt := fmt.Sprintf("yuv420p%dle", tt.bitDepth)

			framemd5 := exec.Command(
				"ffmpeg",
				"-v", "error",
				"-f", "h264",
				"-i", path,
				"-an", "-sn", "-dn",
				"-pix_fmt", pixFmt,
				"-f", "framemd5",
				"-",
			)
			framemd5Out, err := framemd5.Output()
			if err != nil {
				t.Fatalf("ffmpeg framemd5: %v", err)
			}
			for i, want := range []string{tt.refFrameMD5, tt.pFrameMD5} {
				line := []byte(fmt.Sprintf("0, %10d, %10d,        1,     1536, %s", i, i, want))
				if !bytes.Contains(framemd5Out, line) {
					t.Fatalf("missing %q in framemd5:\n%s", line, framemd5Out)
				}
			}

			rawvideo := exec.Command(
				"ffmpeg",
				"-v", "error",
				"-f", "h264",
				"-i", path,
				"-an", "-sn", "-dn",
				"-pix_fmt", pixFmt,
				"-f", "rawvideo",
				"-",
			)
			raw, err := rawvideo.Output()
			if err != nil {
				t.Fatalf("ffmpeg rawvideo: %v", err)
			}
			if len(raw) != 3072 {
				t.Fatalf("rawvideo size = %d, want 3072", len(raw))
			}
			sum := md5.Sum(raw)
			if got := hex.EncodeToString(sum[:]); got != tt.rawVideoMD5 {
				t.Fatalf("rawvideo md5 = %s, want %s", got, tt.rawVideoMD5)
			}
		})
	}
}

func TestFFmpegRawVideoMD5OracleHigh1214FrameMBAFFP16x16LumaResidual(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}

	for _, tt := range highFrameMBAFFP16x16LumaResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFP16x16LumaResidualFixture(tt.bitDepth)
			assertHighFrameMBAFFP16x16LumaResidualFixtureSyntax(t, data, tt)
			path := writeTempH264(t, data)
			pixFmt := fmt.Sprintf("yuv420p%dle", tt.bitDepth)

			framemd5 := exec.Command(
				"ffmpeg",
				"-v", "error",
				"-f", "h264",
				"-i", path,
				"-an", "-sn", "-dn",
				"-pix_fmt", pixFmt,
				"-f", "framemd5",
				"-",
			)
			framemd5Out, err := framemd5.Output()
			if err != nil {
				t.Fatalf("ffmpeg framemd5: %v", err)
			}
			for i, want := range []string{tt.refFrameMD5, tt.pFrameMD5} {
				line := []byte(fmt.Sprintf("0, %10d, %10d,        1,     1536, %s", i, i, want))
				if !bytes.Contains(framemd5Out, line) {
					t.Fatalf("missing %q in framemd5:\n%s", line, framemd5Out)
				}
			}

			rawvideo := exec.Command(
				"ffmpeg",
				"-v", "error",
				"-f", "h264",
				"-i", path,
				"-an", "-sn", "-dn",
				"-pix_fmt", pixFmt,
				"-f", "rawvideo",
				"-",
			)
			raw, err := rawvideo.Output()
			if err != nil {
				t.Fatalf("ffmpeg rawvideo: %v", err)
			}
			if len(raw) != 3072 {
				t.Fatalf("rawvideo size = %d, want 3072", len(raw))
			}
			sum := md5.Sum(raw)
			if got := hex.EncodeToString(sum[:]); got != tt.rawVideoMD5 {
				t.Fatalf("rawvideo md5 = %s, want %s", got, tt.rawVideoMD5)
			}
		})
	}
}

func TestFFmpegRawVideoMD5OracleHigh1214FrameMBAFFFrameP16x16NoResidual(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}

	for _, tt := range highFrameMBAFFFrameP16x16NoResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFFrameP16x16NoResidualFixture(tt.bitDepth)
			assertHighFrameMBAFFFrameP16x16NoResidualFixtureSyntax(t, data, tt)
			path := writeTempH264(t, data)
			pixFmt := fmt.Sprintf("yuv420p%dle", tt.bitDepth)

			framemd5 := exec.Command(
				"ffmpeg",
				"-v", "error",
				"-f", "h264",
				"-i", path,
				"-an", "-sn", "-dn",
				"-pix_fmt", pixFmt,
				"-f", "framemd5",
				"-",
			)
			framemd5Out, err := framemd5.Output()
			if err != nil {
				t.Fatalf("ffmpeg framemd5: %v", err)
			}
			for i, want := range []string{tt.refFrameMD5, tt.pFrameMD5} {
				line := []byte(fmt.Sprintf("0, %10d, %10d,        1,     1536, %s", i, i, want))
				if !bytes.Contains(framemd5Out, line) {
					t.Fatalf("missing %q in framemd5:\n%s", line, framemd5Out)
				}
			}

			rawvideo := exec.Command(
				"ffmpeg",
				"-v", "error",
				"-f", "h264",
				"-i", path,
				"-an", "-sn", "-dn",
				"-pix_fmt", pixFmt,
				"-f", "rawvideo",
				"-",
			)
			raw, err := rawvideo.Output()
			if err != nil {
				t.Fatalf("ffmpeg rawvideo: %v", err)
			}
			if len(raw) != 3072 {
				t.Fatalf("rawvideo size = %d, want 3072", len(raw))
			}
			sum := md5.Sum(raw)
			if got := hex.EncodeToString(sum[:]); got != tt.rawVideoMD5 {
				t.Fatalf("rawvideo md5 = %s, want %s", got, tt.rawVideoMD5)
			}
		})
	}
}

func TestFFmpegRawVideoMD5OracleHigh1214FrameMBAFFFrameP16x16LumaResidual(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}

	for _, tt := range highFrameMBAFFFrameP16x16LumaResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFFrameP16x16LumaResidualFixture(tt.bitDepth)
			assertHighFrameMBAFFFrameP16x16LumaResidualFixtureSyntax(t, data, tt)
			path := writeTempH264(t, data)
			pixFmt := fmt.Sprintf("yuv420p%dle", tt.bitDepth)

			framemd5 := exec.Command(
				"ffmpeg",
				"-v", "error",
				"-f", "h264",
				"-i", path,
				"-an", "-sn", "-dn",
				"-pix_fmt", pixFmt,
				"-f", "framemd5",
				"-",
			)
			framemd5Out, err := framemd5.Output()
			if err != nil {
				t.Fatalf("ffmpeg framemd5: %v", err)
			}
			for i, want := range []string{tt.refFrameMD5, tt.pFrameMD5} {
				line := []byte(fmt.Sprintf("0, %10d, %10d,        1,     1536, %s", i, i, want))
				if !bytes.Contains(framemd5Out, line) {
					t.Fatalf("missing %q in framemd5:\n%s", line, framemd5Out)
				}
			}

			rawvideo := exec.Command(
				"ffmpeg",
				"-v", "error",
				"-f", "h264",
				"-i", path,
				"-an", "-sn", "-dn",
				"-pix_fmt", pixFmt,
				"-f", "rawvideo",
				"-",
			)
			raw, err := rawvideo.Output()
			if err != nil {
				t.Fatalf("ffmpeg rawvideo: %v", err)
			}
			if len(raw) != 3072 {
				t.Fatalf("rawvideo size = %d, want 3072", len(raw))
			}
			sum := md5.Sum(raw)
			if got := hex.EncodeToString(sum[:]); got != tt.rawVideoMD5 {
				t.Fatalf("rawvideo md5 = %s, want %s", got, tt.rawVideoMD5)
			}
		})
	}
}

func TestFFmpegRawVideoMD5OracleHigh1214FrameMBAFFP16x16Deblock(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}

	for _, tt := range highFrameMBAFFP16x16DeblockCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFP16x16DeblockFixture(tt)
			assertHighFrameMBAFFP16x16DeblockFixtureSyntax(t, data, tt)
			path := writeTempH264(t, data)
			pixFmt := fmt.Sprintf("yuv420p%dle", tt.bitDepth)

			framemd5 := exec.Command(
				"ffmpeg",
				"-v", "error",
				"-f", "h264",
				"-i", path,
				"-an", "-sn", "-dn",
				"-pix_fmt", pixFmt,
				"-f", "framemd5",
				"-",
			)
			framemd5Out, err := framemd5.Output()
			if err != nil {
				t.Fatalf("ffmpeg framemd5: %v", err)
			}
			for i, want := range []string{tt.refFrameMD5, tt.pFrameMD5} {
				line := []byte(fmt.Sprintf("0, %10d, %10d,        1,     1536, %s", i, i, want))
				if !bytes.Contains(framemd5Out, line) {
					t.Fatalf("missing %q in framemd5:\n%s", line, framemd5Out)
				}
			}

			rawvideo := exec.Command(
				"ffmpeg",
				"-v", "error",
				"-f", "h264",
				"-i", path,
				"-an", "-sn", "-dn",
				"-pix_fmt", pixFmt,
				"-f", "rawvideo",
				"-",
			)
			raw, err := rawvideo.Output()
			if err != nil {
				t.Fatalf("ffmpeg rawvideo: %v", err)
			}
			if len(raw) != 3072 {
				t.Fatalf("rawvideo size = %d, want 3072", len(raw))
			}
			sum := md5.Sum(raw)
			if got := hex.EncodeToString(sum[:]); got != tt.rawVideoMD5 {
				t.Fatalf("rawvideo md5 = %s, want %s", got, tt.rawVideoMD5)
			}
		})
	}
}

func TestFFmpegRawVideoMD5OracleHigh1214FrameMBAFFPartitionedPDeblock(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}

	for _, tt := range highFrameMBAFFPartitionedPDeblockCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFPartitionedPDeblockFixture(tt)
			assertHighFrameMBAFFPartitionedPDeblockFixtureSyntax(t, data, tt)
			path := writeTempH264(t, data)
			pixFmt := fmt.Sprintf("yuv420p%dle", tt.bitDepth)

			framemd5 := exec.Command(
				"ffmpeg",
				"-v", "error",
				"-f", "h264",
				"-i", path,
				"-an", "-sn", "-dn",
				"-pix_fmt", pixFmt,
				"-f", "framemd5",
				"-",
			)
			framemd5Out, err := framemd5.Output()
			if err != nil {
				t.Fatalf("ffmpeg framemd5: %v", err)
			}
			for i, want := range []string{tt.refFrameMD5, tt.pFrameMD5} {
				line := []byte(fmt.Sprintf("0, %10d, %10d,        1,     1536, %s", i, i, want))
				if !bytes.Contains(framemd5Out, line) {
					t.Fatalf("missing %q in framemd5:\n%s", line, framemd5Out)
				}
			}

			rawvideo := exec.Command(
				"ffmpeg",
				"-v", "error",
				"-f", "h264",
				"-i", path,
				"-an", "-sn", "-dn",
				"-pix_fmt", pixFmt,
				"-f", "rawvideo",
				"-",
			)
			raw, err := rawvideo.Output()
			if err != nil {
				t.Fatalf("ffmpeg rawvideo: %v", err)
			}
			if len(raw) != 3072 {
				t.Fatalf("rawvideo size = %d, want 3072", len(raw))
			}
			sum := md5.Sum(raw)
			if got := hex.EncodeToString(sum[:]); got != tt.rawVideoMD5 {
				t.Fatalf("rawvideo md5 = %s, want %s", got, tt.rawVideoMD5)
			}
		})
	}
}

func TestFFmpegRawVideoMD5OracleHigh1214FrameMBAFFPartitionedPSparseResidual(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}

	for _, tt := range highFrameMBAFFPartitionedPSparseResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFPartitionedPSparseResidualFixture(tt.bitDepth, tt.payloadBits)
			assertHighFrameMBAFFPartitionedPSparseResidualFixtureSyntax(t, data, tt)
			path := writeTempH264(t, data)
			pixFmt := fmt.Sprintf("yuv420p%dle", tt.bitDepth)

			framemd5 := exec.Command(
				"ffmpeg",
				"-v", "error",
				"-f", "h264",
				"-i", path,
				"-an", "-sn", "-dn",
				"-pix_fmt", pixFmt,
				"-f", "framemd5",
				"-",
			)
			framemd5Out, err := framemd5.Output()
			if err != nil {
				t.Fatalf("ffmpeg framemd5: %v", err)
			}
			for i, want := range []string{tt.refFrameMD5, tt.pFrameMD5} {
				line := []byte(fmt.Sprintf("0, %10d, %10d,        1,     1536, %s", i, i, want))
				if !bytes.Contains(framemd5Out, line) {
					t.Fatalf("missing %q in framemd5:\n%s", line, framemd5Out)
				}
			}

			rawvideo := exec.Command(
				"ffmpeg",
				"-v", "error",
				"-f", "h264",
				"-i", path,
				"-an", "-sn", "-dn",
				"-pix_fmt", pixFmt,
				"-f", "rawvideo",
				"-",
			)
			raw, err := rawvideo.Output()
			if err != nil {
				t.Fatalf("ffmpeg rawvideo: %v", err)
			}
			if len(raw) != 3072 {
				t.Fatalf("rawvideo size = %d, want 3072", len(raw))
			}
			sum := md5.Sum(raw)
			if got := hex.EncodeToString(sum[:]); got != tt.rawVideoMD5 {
				t.Fatalf("rawvideo md5 = %s, want %s", got, tt.rawVideoMD5)
			}
		})
	}
}

func TestFFmpegRawVideoMD5OracleHigh1214FrameMBAFFP16x16LumaChromaResidual(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}

	for _, tt := range highFrameMBAFFP16x16LumaChromaResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFP16x16LumaChromaResidualFixture(tt.bitDepth)
			assertHighFrameMBAFFP16x16LumaChromaResidualFixtureSyntax(t, data, tt)
			path := writeTempH264(t, data)
			pixFmt := fmt.Sprintf("yuv420p%dle", tt.bitDepth)

			framemd5 := exec.Command(
				"ffmpeg",
				"-v", "error",
				"-f", "h264",
				"-i", path,
				"-an", "-sn", "-dn",
				"-pix_fmt", pixFmt,
				"-f", "framemd5",
				"-",
			)
			framemd5Out, err := framemd5.Output()
			if err != nil {
				t.Fatalf("ffmpeg framemd5: %v", err)
			}
			for i, want := range []string{tt.refFrameMD5, tt.pFrameMD5} {
				line := []byte(fmt.Sprintf("0, %10d, %10d,        1,     1536, %s", i, i, want))
				if !bytes.Contains(framemd5Out, line) {
					t.Fatalf("missing %q in framemd5:\n%s", line, framemd5Out)
				}
			}

			rawvideo := exec.Command(
				"ffmpeg",
				"-v", "error",
				"-f", "h264",
				"-i", path,
				"-an", "-sn", "-dn",
				"-pix_fmt", pixFmt,
				"-f", "rawvideo",
				"-",
			)
			raw, err := rawvideo.Output()
			if err != nil {
				t.Fatalf("ffmpeg rawvideo: %v", err)
			}
			if len(raw) != 3072 {
				t.Fatalf("rawvideo size = %d, want 3072", len(raw))
			}
			sum := md5.Sum(raw)
			if got := hex.EncodeToString(sum[:]); got != tt.rawVideoMD5 {
				t.Fatalf("rawvideo md5 = %s, want %s", got, tt.rawVideoMD5)
			}
		})
	}
}

func TestFFmpegRawVideoMD5OracleHigh1214FrameMBAFFPartitionedPLumaChromaResidual(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}

	for _, tt := range highFrameMBAFFPartitionedPLumaChromaResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFPartitionedPLumaChromaResidualFixture(tt.bitDepth, tt.payloadBits)
			assertHighFrameMBAFFPartitionedPLumaChromaResidualFixtureSyntax(t, data, tt)
			path := writeTempH264(t, data)
			pixFmt := fmt.Sprintf("yuv420p%dle", tt.bitDepth)

			framemd5 := exec.Command(
				"ffmpeg",
				"-v", "error",
				"-f", "h264",
				"-i", path,
				"-an", "-sn", "-dn",
				"-pix_fmt", pixFmt,
				"-f", "framemd5",
				"-",
			)
			framemd5Out, err := framemd5.Output()
			if err != nil {
				t.Fatalf("ffmpeg framemd5: %v", err)
			}
			for i, want := range []string{tt.refFrameMD5, tt.pFrameMD5} {
				line := []byte(fmt.Sprintf("0, %10d, %10d,        1,     1536, %s", i, i, want))
				if !bytes.Contains(framemd5Out, line) {
					t.Fatalf("missing %q in framemd5:\n%s", line, framemd5Out)
				}
			}

			rawvideo := exec.Command(
				"ffmpeg",
				"-v", "error",
				"-f", "h264",
				"-i", path,
				"-an", "-sn", "-dn",
				"-pix_fmt", pixFmt,
				"-f", "rawvideo",
				"-",
			)
			raw, err := rawvideo.Output()
			if err != nil {
				t.Fatalf("ffmpeg rawvideo: %v", err)
			}
			if len(raw) != 3072 {
				t.Fatalf("rawvideo size = %d, want 3072", len(raw))
			}
			sum := md5.Sum(raw)
			if got := hex.EncodeToString(sum[:]); got != tt.rawVideoMD5 {
				t.Fatalf("rawvideo md5 = %s, want %s", got, tt.rawVideoMD5)
			}
		})
	}
}

func TestFFmpegRawVideoMD5OracleHigh1214FrameMBAFFFrameCodedPLumaChromaResidual(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}

	for _, tt := range highFrameMBAFFFrameCodedPLumaChromaResidualCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highFrameMBAFFFrameCodedPLumaChromaResidualFixture(tt.bitDepth, tt.payloadBits)
			assertHighFrameMBAFFFrameCodedPLumaChromaResidualFixtureSyntax(t, data, tt)
			path := writeTempH264(t, data)
			pixFmt := fmt.Sprintf("yuv420p%dle", tt.bitDepth)

			framemd5 := exec.Command(
				"ffmpeg",
				"-v", "error",
				"-f", "h264",
				"-i", path,
				"-an", "-sn", "-dn",
				"-pix_fmt", pixFmt,
				"-f", "framemd5",
				"-",
			)
			framemd5Out, err := framemd5.Output()
			if err != nil {
				t.Fatalf("ffmpeg framemd5: %v", err)
			}
			for i, want := range []string{tt.refFrameMD5, tt.pFrameMD5} {
				line := []byte(fmt.Sprintf("0, %10d, %10d,        1,     1536, %s", i, i, want))
				if !bytes.Contains(framemd5Out, line) {
					t.Fatalf("missing %q in framemd5:\n%s", line, framemd5Out)
				}
			}

			rawvideo := exec.Command(
				"ffmpeg",
				"-v", "error",
				"-f", "h264",
				"-i", path,
				"-an", "-sn", "-dn",
				"-pix_fmt", pixFmt,
				"-f", "rawvideo",
				"-",
			)
			raw, err := rawvideo.Output()
			if err != nil {
				t.Fatalf("ffmpeg rawvideo: %v", err)
			}
			if len(raw) != 3072 {
				t.Fatalf("rawvideo size = %d, want 3072", len(raw))
			}
			sum := md5.Sum(raw)
			if got := hex.EncodeToString(sum[:]); got != tt.rawVideoMD5 {
				t.Fatalf("rawvideo md5 = %s, want %s", got, tt.rawVideoMD5)
			}
		})
	}
}

func highFrameMBAFFPSkipNoResidualCases() []highFrameMBAFFPSkipNoResidualCase {
	return []highFrameMBAFFPSkipNoResidualCase{
		{
			name:         "High12",
			bitDepth:     12,
			bitstreamMD5: high12FrameMBAFFPSkipNoResidualBitstreamMD5,
			refFrameMD5:  high12FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:    high12FrameMBAFFPSkipNoResidualPFrameMD5,
			rawVideoMD5:  high12FrameMBAFFPSkipNoResidualRawVideoMD5,
		},
		{
			name:         "High14",
			bitDepth:     14,
			bitstreamMD5: high14FrameMBAFFPSkipNoResidualBitstreamMD5,
			refFrameMD5:  high14FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:    high14FrameMBAFFPSkipNoResidualPFrameMD5,
			rawVideoMD5:  high14FrameMBAFFPSkipNoResidualRawVideoMD5,
		},
	}
}

func highFrameMBAFFP16x16NoResidualCases() []highFrameMBAFFP16x16NoResidualCase {
	return []highFrameMBAFFP16x16NoResidualCase{
		{
			name:         "High12",
			bitDepth:     12,
			bitstreamMD5: high12FrameMBAFFP16x16NoResidualBitstreamMD5,
			refFrameMD5:  high12FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:    high12FrameMBAFFP16x16NoResidualPFrameMD5,
			rawVideoMD5:  high12FrameMBAFFP16x16NoResidualRawVideoMD5,
		},
		{
			name:         "High14",
			bitDepth:     14,
			bitstreamMD5: high14FrameMBAFFP16x16NoResidualBitstreamMD5,
			refFrameMD5:  high14FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:    high14FrameMBAFFP16x16NoResidualPFrameMD5,
			rawVideoMD5:  high14FrameMBAFFP16x16NoResidualRawVideoMD5,
		},
	}
}

func highFrameMBAFFP16x16LumaResidualCases() []highFrameMBAFFP16x16LumaResidualCase {
	return []highFrameMBAFFP16x16LumaResidualCase{
		{
			name:         "High12",
			bitDepth:     12,
			bitstreamMD5: high12FrameMBAFFP16x16LumaResidualBitstreamMD5,
			refFrameMD5:  high12FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:    high12FrameMBAFFP16x16LumaResidualPFrameMD5,
			rawVideoMD5:  high12FrameMBAFFP16x16LumaResidualRawVideoMD5,
		},
		{
			name:         "High14",
			bitDepth:     14,
			bitstreamMD5: high14FrameMBAFFP16x16LumaResidualBitstreamMD5,
			refFrameMD5:  high14FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:    high14FrameMBAFFP16x16LumaResidualPFrameMD5,
			rawVideoMD5:  high14FrameMBAFFP16x16LumaResidualRawVideoMD5,
		},
	}
}

func highFrameMBAFFFrameP16x16NoResidualCases() []highFrameMBAFFFrameP16x16NoResidualCase {
	return []highFrameMBAFFFrameP16x16NoResidualCase{
		{
			name:         "High12FrameP16x16",
			bitDepth:     12,
			bitstreamMD5: high12FrameMBAFFFrameP16x16NoResidualBitstreamMD5,
			refFrameMD5:  high12FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:    high12FrameMBAFFFrameP16x16NoResidualPFrameMD5,
			rawVideoMD5:  high12FrameMBAFFFrameP16x16NoResidualRawVideoMD5,
		},
		{
			name:         "High14FrameP16x16",
			bitDepth:     14,
			bitstreamMD5: high14FrameMBAFFFrameP16x16NoResidualBitstreamMD5,
			refFrameMD5:  high14FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:    high14FrameMBAFFFrameP16x16NoResidualPFrameMD5,
			rawVideoMD5:  high14FrameMBAFFFrameP16x16NoResidualRawVideoMD5,
		},
	}
}

func highFrameMBAFFFrameP16x16LumaResidualCases() []highFrameMBAFFFrameP16x16LumaResidualCase {
	return []highFrameMBAFFFrameP16x16LumaResidualCase{
		{
			name:         "High12FrameP16x16",
			bitDepth:     12,
			bitstreamMD5: high12FrameMBAFFFrameP16x16LumaResidualBitstreamMD5,
			refFrameMD5:  high12FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:    high12FrameMBAFFFrameP16x16LumaResidualPFrameMD5,
			rawVideoMD5:  high12FrameMBAFFFrameP16x16LumaResidualRawVideoMD5,
		},
		{
			name:         "High14FrameP16x16",
			bitDepth:     14,
			bitstreamMD5: high14FrameMBAFFFrameP16x16LumaResidualBitstreamMD5,
			refFrameMD5:  high14FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:    high14FrameMBAFFFrameP16x16LumaResidualPFrameMD5,
			rawVideoMD5:  high14FrameMBAFFFrameP16x16LumaResidualRawVideoMD5,
		},
	}
}

func highFrameMBAFFP16x16DeblockCases() []highFrameMBAFFP16x16DeblockCase {
	return []highFrameMBAFFP16x16DeblockCase{
		{
			name:                       "High12PSkipDeblockMode1",
			bitDepth:                   12,
			pskip:                      true,
			disableDeblockingFilterIDC: 0,
			deblockMode:                1,
			bitstreamMD5:               "608b9216c7968c2237653c4e220cde61",
			refFrameMD5:                high12FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:                  high12FrameMBAFFPSkipNoResidualPFrameMD5,
			rawVideoMD5:                high12FrameMBAFFPSkipNoResidualRawVideoMD5,
		},
		{
			name:                       "High12FieldP16x16NoResidualDeblockMode1",
			bitDepth:                   12,
			fieldFlag:                  1,
			cbp:                        0,
			payloadBits:                highFrameMBAFFP16x16NoResidualPayloadBits,
			disableDeblockingFilterIDC: 0,
			deblockMode:                1,
			bitstreamMD5:               "acd28f7457c9749c4b387891665ea1a4",
			refFrameMD5:                high12FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:                  high12FrameMBAFFP16x16NoResidualPFrameMD5,
			rawVideoMD5:                high12FrameMBAFFP16x16NoResidualRawVideoMD5,
		},
		{
			name:                       "High12FieldP16x16LumaResidualDeblockMode1",
			bitDepth:                   12,
			fieldFlag:                  1,
			cbp:                        1,
			payloadBits:                highFrameMBAFFP16x16LumaResidualPayloadBits,
			residualTailBits:           highFrameMBAFFP16x16LumaResidualTailBits,
			disableDeblockingFilterIDC: 0,
			deblockMode:                1,
			bitstreamMD5:               "a803c5425184576a14ef49ddb30be5b3",
			refFrameMD5:                high12FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:                  high12FrameMBAFFP16x16LumaResidualPFrameMD5,
			rawVideoMD5:                high12FrameMBAFFP16x16LumaResidualRawVideoMD5,
		},
		{
			name:                       "High12FieldP16x16LumaChromaResidualDeblockMode1",
			bitDepth:                   12,
			fieldFlag:                  1,
			cbp:                        33,
			payloadBits:                highFrameMBAFFP16x16LumaChromaResidualPayloadBits,
			residualTailBits:           highFrameMBAFFP16x16LumaChromaResidualTailBits,
			disableDeblockingFilterIDC: 0,
			deblockMode:                1,
			bitstreamMD5:               "18cca3d955ee085a69ed0ad4995505e1",
			refFrameMD5:                high12FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:                  high12FrameMBAFFP16x16LumaChromaResidualPFrameMD5,
			rawVideoMD5:                high12FrameMBAFFP16x16LumaChromaResidualRawVideoMD5,
		},
		{
			name:                       "High12FrameP16x16NoResidualDeblockMode1",
			bitDepth:                   12,
			fieldFlag:                  0,
			cbp:                        0,
			payloadBits:                highFrameMBAFFFrameP16x16NoResidualPayloadBits,
			disableDeblockingFilterIDC: 0,
			deblockMode:                1,
			bitstreamMD5:               "38cbd9a0806d3a7d3e37d073fadfed72",
			refFrameMD5:                high12FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:                  high12FrameMBAFFFrameP16x16NoResidualPFrameMD5,
			rawVideoMD5:                high12FrameMBAFFFrameP16x16NoResidualRawVideoMD5,
		},
		{
			name:                       "High12FrameP16x16LumaResidualDeblockMode1",
			bitDepth:                   12,
			fieldFlag:                  0,
			cbp:                        1,
			payloadBits:                highFrameMBAFFFrameP16x16LumaResidualPayloadBits,
			residualTailBits:           highFrameMBAFFP16x16LumaResidualTailBits,
			disableDeblockingFilterIDC: 0,
			deblockMode:                1,
			bitstreamMD5:               "ea7ffd5c2efe2695560f9d3c2a564294",
			refFrameMD5:                high12FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:                  high12FrameMBAFFFrameP16x16LumaResidualPFrameMD5,
			rawVideoMD5:                high12FrameMBAFFFrameP16x16LumaResidualRawVideoMD5,
		},
		{
			name:                       "High12FrameP16x16LumaChromaResidualDeblockMode1",
			bitDepth:                   12,
			fieldFlag:                  0,
			cbp:                        33,
			payloadBits:                highFrameMBAFFFrameP16x16LumaChromaResidualPayloadBits,
			residualTailBits:           highFrameMBAFFP16x16LumaChromaResidualTailBits,
			disableDeblockingFilterIDC: 0,
			deblockMode:                1,
			bitstreamMD5:               "090b99a424826c453ad894e23d6ae501",
			refFrameMD5:                high12FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:                  high12FrameMBAFFFrameP16x16LumaChromaResidualPFrameMD5,
			rawVideoMD5:                high12FrameMBAFFFrameP16x16LumaChromaResidualRawVideoMD5,
		},
		{
			name:                       "High12PSkipDeblockMode2",
			bitDepth:                   12,
			pskip:                      true,
			disableDeblockingFilterIDC: 2,
			deblockMode:                2,
			bitstreamMD5:               "7f66740b1b274e36811dab05f558addc",
			refFrameMD5:                high12FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:                  high12FrameMBAFFPSkipNoResidualPFrameMD5,
			rawVideoMD5:                high12FrameMBAFFPSkipNoResidualRawVideoMD5,
		},
		{
			name:                       "High12FieldP16x16NoResidualDeblockMode2",
			bitDepth:                   12,
			fieldFlag:                  1,
			cbp:                        0,
			payloadBits:                highFrameMBAFFP16x16NoResidualPayloadBits,
			disableDeblockingFilterIDC: 2,
			deblockMode:                2,
			bitstreamMD5:               "0310cff17dd6568f86cc3a670143a00c",
			refFrameMD5:                high12FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:                  high12FrameMBAFFP16x16NoResidualPFrameMD5,
			rawVideoMD5:                high12FrameMBAFFP16x16NoResidualRawVideoMD5,
		},
		{
			name:                       "High12FieldP16x16LumaResidualDeblockMode2",
			bitDepth:                   12,
			fieldFlag:                  1,
			cbp:                        1,
			payloadBits:                highFrameMBAFFP16x16LumaResidualPayloadBits,
			residualTailBits:           highFrameMBAFFP16x16LumaResidualTailBits,
			disableDeblockingFilterIDC: 2,
			deblockMode:                2,
			bitstreamMD5:               "31d665552533173bd6e7b12db68ae9fc",
			refFrameMD5:                high12FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:                  high12FrameMBAFFP16x16LumaResidualPFrameMD5,
			rawVideoMD5:                high12FrameMBAFFP16x16LumaResidualRawVideoMD5,
		},
		{
			name:                       "High12FieldP16x16LumaChromaResidualDeblockMode2",
			bitDepth:                   12,
			fieldFlag:                  1,
			cbp:                        33,
			payloadBits:                highFrameMBAFFP16x16LumaChromaResidualPayloadBits,
			residualTailBits:           highFrameMBAFFP16x16LumaChromaResidualTailBits,
			disableDeblockingFilterIDC: 2,
			deblockMode:                2,
			bitstreamMD5:               "964e6b4135d0ff26a64dfbb6fd6eeade",
			refFrameMD5:                high12FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:                  high12FrameMBAFFP16x16LumaChromaResidualPFrameMD5,
			rawVideoMD5:                high12FrameMBAFFP16x16LumaChromaResidualRawVideoMD5,
		},
		{
			name:                       "High12FrameP16x16NoResidualDeblockMode2",
			bitDepth:                   12,
			fieldFlag:                  0,
			cbp:                        0,
			payloadBits:                highFrameMBAFFFrameP16x16NoResidualPayloadBits,
			disableDeblockingFilterIDC: 2,
			deblockMode:                2,
			bitstreamMD5:               "a954eb2c54b5fb83d8819095a4c60711",
			refFrameMD5:                high12FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:                  high12FrameMBAFFFrameP16x16NoResidualPFrameMD5,
			rawVideoMD5:                high12FrameMBAFFFrameP16x16NoResidualRawVideoMD5,
		},
		{
			name:                       "High12FrameP16x16LumaResidualDeblockMode2",
			bitDepth:                   12,
			fieldFlag:                  0,
			cbp:                        1,
			payloadBits:                highFrameMBAFFFrameP16x16LumaResidualPayloadBits,
			residualTailBits:           highFrameMBAFFP16x16LumaResidualTailBits,
			disableDeblockingFilterIDC: 2,
			deblockMode:                2,
			bitstreamMD5:               "27365539290b8929a69e21b5b63e904f",
			refFrameMD5:                high12FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:                  high12FrameMBAFFFrameP16x16LumaResidualPFrameMD5,
			rawVideoMD5:                high12FrameMBAFFFrameP16x16LumaResidualRawVideoMD5,
		},
		{
			name:                       "High12FrameP16x16LumaChromaResidualDeblockMode2",
			bitDepth:                   12,
			fieldFlag:                  0,
			cbp:                        33,
			payloadBits:                highFrameMBAFFFrameP16x16LumaChromaResidualPayloadBits,
			residualTailBits:           highFrameMBAFFP16x16LumaChromaResidualTailBits,
			disableDeblockingFilterIDC: 2,
			deblockMode:                2,
			bitstreamMD5:               "3b34687c5c2da799bc6de264d75b0afd",
			refFrameMD5:                high12FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:                  high12FrameMBAFFFrameP16x16LumaChromaResidualPFrameMD5,
			rawVideoMD5:                high12FrameMBAFFFrameP16x16LumaChromaResidualRawVideoMD5,
		},
		{
			name:                       "High14PSkipDeblockMode1",
			bitDepth:                   14,
			pskip:                      true,
			disableDeblockingFilterIDC: 0,
			deblockMode:                1,
			bitstreamMD5:               "ec0d3b183eaa775a772fc354dae7161e",
			refFrameMD5:                high14FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:                  high14FrameMBAFFPSkipNoResidualPFrameMD5,
			rawVideoMD5:                high14FrameMBAFFPSkipNoResidualRawVideoMD5,
		},
		{
			name:                       "High14FieldP16x16NoResidualDeblockMode1",
			bitDepth:                   14,
			fieldFlag:                  1,
			cbp:                        0,
			payloadBits:                highFrameMBAFFP16x16NoResidualPayloadBits,
			disableDeblockingFilterIDC: 0,
			deblockMode:                1,
			bitstreamMD5:               "77a0a6e988de2c3cde0678f99a6f9ecb",
			refFrameMD5:                high14FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:                  high14FrameMBAFFP16x16NoResidualPFrameMD5,
			rawVideoMD5:                high14FrameMBAFFP16x16NoResidualRawVideoMD5,
		},
		{
			name:                       "High14FieldP16x16LumaResidualDeblockMode1",
			bitDepth:                   14,
			fieldFlag:                  1,
			cbp:                        1,
			payloadBits:                highFrameMBAFFP16x16LumaResidualPayloadBits,
			residualTailBits:           highFrameMBAFFP16x16LumaResidualTailBits,
			disableDeblockingFilterIDC: 0,
			deblockMode:                1,
			bitstreamMD5:               "b74276c97423f43b0855eee818d3d08f",
			refFrameMD5:                high14FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:                  high14FrameMBAFFP16x16LumaResidualPFrameMD5,
			rawVideoMD5:                high14FrameMBAFFP16x16LumaResidualRawVideoMD5,
		},
		{
			name:                       "High14FieldP16x16LumaChromaResidualDeblockMode1",
			bitDepth:                   14,
			fieldFlag:                  1,
			cbp:                        33,
			payloadBits:                highFrameMBAFFP16x16LumaChromaResidualPayloadBits,
			residualTailBits:           highFrameMBAFFP16x16LumaChromaResidualTailBits,
			disableDeblockingFilterIDC: 0,
			deblockMode:                1,
			bitstreamMD5:               "6fcc2c21bf81d640c9834ed9870efc41",
			refFrameMD5:                high14FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:                  high14FrameMBAFFP16x16LumaChromaResidualPFrameMD5,
			rawVideoMD5:                high14FrameMBAFFP16x16LumaChromaResidualRawVideoMD5,
		},
		{
			name:                       "High14FrameP16x16NoResidualDeblockMode1",
			bitDepth:                   14,
			fieldFlag:                  0,
			cbp:                        0,
			payloadBits:                highFrameMBAFFFrameP16x16NoResidualPayloadBits,
			disableDeblockingFilterIDC: 0,
			deblockMode:                1,
			bitstreamMD5:               "ca9482c6cf08291174b81a7ef9ca3be6",
			refFrameMD5:                high14FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:                  high14FrameMBAFFFrameP16x16NoResidualPFrameMD5,
			rawVideoMD5:                high14FrameMBAFFFrameP16x16NoResidualRawVideoMD5,
		},
		{
			name:                       "High14FrameP16x16LumaResidualDeblockMode1",
			bitDepth:                   14,
			fieldFlag:                  0,
			cbp:                        1,
			payloadBits:                highFrameMBAFFFrameP16x16LumaResidualPayloadBits,
			residualTailBits:           highFrameMBAFFP16x16LumaResidualTailBits,
			disableDeblockingFilterIDC: 0,
			deblockMode:                1,
			bitstreamMD5:               "dffc96eebf983476ad09a0fab2bcb908",
			refFrameMD5:                high14FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:                  high14FrameMBAFFFrameP16x16LumaResidualPFrameMD5,
			rawVideoMD5:                high14FrameMBAFFFrameP16x16LumaResidualRawVideoMD5,
		},
		{
			name:                       "High14FrameP16x16LumaChromaResidualDeblockMode1",
			bitDepth:                   14,
			fieldFlag:                  0,
			cbp:                        33,
			payloadBits:                highFrameMBAFFFrameP16x16LumaChromaResidualPayloadBits,
			residualTailBits:           highFrameMBAFFP16x16LumaChromaResidualTailBits,
			disableDeblockingFilterIDC: 0,
			deblockMode:                1,
			bitstreamMD5:               "64cfd8aa7c0d300da6ec72ffe0bd2f50",
			refFrameMD5:                high14FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:                  high14FrameMBAFFFrameP16x16LumaChromaResidualPFrameMD5,
			rawVideoMD5:                high14FrameMBAFFFrameP16x16LumaChromaResidualRawVideoMD5,
		},
		{
			name:                       "High14PSkipDeblockMode2",
			bitDepth:                   14,
			pskip:                      true,
			disableDeblockingFilterIDC: 2,
			deblockMode:                2,
			bitstreamMD5:               "b358f93b6d1bf810ac7a720ee5bfb420",
			refFrameMD5:                high14FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:                  high14FrameMBAFFPSkipNoResidualPFrameMD5,
			rawVideoMD5:                high14FrameMBAFFPSkipNoResidualRawVideoMD5,
		},
		{
			name:                       "High14FieldP16x16NoResidualDeblockMode2",
			bitDepth:                   14,
			fieldFlag:                  1,
			cbp:                        0,
			payloadBits:                highFrameMBAFFP16x16NoResidualPayloadBits,
			disableDeblockingFilterIDC: 2,
			deblockMode:                2,
			bitstreamMD5:               "35cf1d5a423c4f9bedd5962a6a2986a0",
			refFrameMD5:                high14FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:                  high14FrameMBAFFP16x16NoResidualPFrameMD5,
			rawVideoMD5:                high14FrameMBAFFP16x16NoResidualRawVideoMD5,
		},
		{
			name:                       "High14FieldP16x16LumaResidualDeblockMode2",
			bitDepth:                   14,
			fieldFlag:                  1,
			cbp:                        1,
			payloadBits:                highFrameMBAFFP16x16LumaResidualPayloadBits,
			residualTailBits:           highFrameMBAFFP16x16LumaResidualTailBits,
			disableDeblockingFilterIDC: 2,
			deblockMode:                2,
			bitstreamMD5:               "c5640c7b503dd597aa860d4c6a153037",
			refFrameMD5:                high14FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:                  high14FrameMBAFFP16x16LumaResidualPFrameMD5,
			rawVideoMD5:                high14FrameMBAFFP16x16LumaResidualRawVideoMD5,
		},
		{
			name:                       "High14FieldP16x16LumaChromaResidualDeblockMode2",
			bitDepth:                   14,
			fieldFlag:                  1,
			cbp:                        33,
			payloadBits:                highFrameMBAFFP16x16LumaChromaResidualPayloadBits,
			residualTailBits:           highFrameMBAFFP16x16LumaChromaResidualTailBits,
			disableDeblockingFilterIDC: 2,
			deblockMode:                2,
			bitstreamMD5:               "d14f1036c4c2aecca37b5f28f8075f41",
			refFrameMD5:                high14FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:                  high14FrameMBAFFP16x16LumaChromaResidualPFrameMD5,
			rawVideoMD5:                high14FrameMBAFFP16x16LumaChromaResidualRawVideoMD5,
		},
		{
			name:                       "High14FrameP16x16NoResidualDeblockMode2",
			bitDepth:                   14,
			fieldFlag:                  0,
			cbp:                        0,
			payloadBits:                highFrameMBAFFFrameP16x16NoResidualPayloadBits,
			disableDeblockingFilterIDC: 2,
			deblockMode:                2,
			bitstreamMD5:               "300f11e4238688ca463c2a0c93f7a868",
			refFrameMD5:                high14FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:                  high14FrameMBAFFFrameP16x16NoResidualPFrameMD5,
			rawVideoMD5:                high14FrameMBAFFFrameP16x16NoResidualRawVideoMD5,
		},
		{
			name:                       "High14FrameP16x16LumaResidualDeblockMode2",
			bitDepth:                   14,
			fieldFlag:                  0,
			cbp:                        1,
			payloadBits:                highFrameMBAFFFrameP16x16LumaResidualPayloadBits,
			residualTailBits:           highFrameMBAFFP16x16LumaResidualTailBits,
			disableDeblockingFilterIDC: 2,
			deblockMode:                2,
			bitstreamMD5:               "3c36dcd36227649291fc20835fa0322d",
			refFrameMD5:                high14FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:                  high14FrameMBAFFFrameP16x16LumaResidualPFrameMD5,
			rawVideoMD5:                high14FrameMBAFFFrameP16x16LumaResidualRawVideoMD5,
		},
		{
			name:                       "High14FrameP16x16LumaChromaResidualDeblockMode2",
			bitDepth:                   14,
			fieldFlag:                  0,
			cbp:                        33,
			payloadBits:                highFrameMBAFFFrameP16x16LumaChromaResidualPayloadBits,
			residualTailBits:           highFrameMBAFFP16x16LumaChromaResidualTailBits,
			disableDeblockingFilterIDC: 2,
			deblockMode:                2,
			bitstreamMD5:               "66ac79f719abe37e2ab43108f4f313e6",
			refFrameMD5:                high14FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:                  high14FrameMBAFFFrameP16x16LumaChromaResidualPFrameMD5,
			rawVideoMD5:                high14FrameMBAFFFrameP16x16LumaChromaResidualRawVideoMD5,
		},
	}
}

func highFrameMBAFFPartitionedPDeblockCases() []highFrameMBAFFPartitionedPDeblockCase {
	bitstreamMD5 := highFrameMBAFFPartitionedPDeblockBitstreamMD5()
	var cases []highFrameMBAFFPartitionedPDeblockCase
	for _, base := range highFrameMBAFFPartitionedPDeblockBaseCases() {
		for _, deblock := range []struct {
			suffix     string
			disableIDC uint32
			mode       int32
		}{
			{suffix: "DeblockMode1", disableIDC: 0, mode: 1},
			{suffix: "DeblockMode2", disableIDC: 2, mode: 2},
		} {
			tt := base
			tt.name += deblock.suffix
			tt.disableDeblockingFilterIDC = deblock.disableIDC
			tt.deblockMode = deblock.mode
			tt.bitstreamMD5 = bitstreamMD5[tt.name]
			cases = append(cases, tt)
		}
	}
	return cases
}

func highFrameMBAFFPartitionedPDeblockBaseCases() []highFrameMBAFFPartitionedPDeblockCase {
	type shape struct {
		name        string
		mbType      uint32
		fieldNo     string
		fieldLuma   string
		fieldChroma string
		frameNo     string
		frameLuma   string
		frameChroma string
	}
	shapes := []shape{
		{
			name:        "P16x8",
			mbType:      1,
			fieldNo:     highFrameMBAFFP16x8NoResidualPayloadBits,
			fieldLuma:   highFrameMBAFFP16x8LumaResidualPayloadBits,
			fieldChroma: highFrameMBAFFP16x8LumaChromaResidualPayloadBits,
			frameNo:     highFrameMBAFFFrameP16x8NoResidualPayloadBits,
			frameLuma:   highFrameMBAFFFrameP16x8LumaResidualPayloadBits,
			frameChroma: highFrameMBAFFFrameP16x8LumaChromaResidualPayloadBits,
		},
		{
			name:        "P8x16",
			mbType:      2,
			fieldNo:     highFrameMBAFFP8x16NoResidualPayloadBits,
			fieldLuma:   highFrameMBAFFP8x16LumaResidualPayloadBits,
			fieldChroma: highFrameMBAFFP8x16LumaChromaResidualPayloadBits,
			frameNo:     highFrameMBAFFFrameP8x16NoResidualPayloadBits,
			frameLuma:   highFrameMBAFFFrameP8x16LumaResidualPayloadBits,
			frameChroma: highFrameMBAFFFrameP8x16LumaChromaResidualPayloadBits,
		},
		{
			name:        "P8x8",
			mbType:      3,
			fieldNo:     highFrameMBAFFP8x8NoResidualPayloadBits,
			fieldLuma:   highFrameMBAFFP8x8LumaResidualPayloadBits,
			fieldChroma: highFrameMBAFFP8x8LumaChromaResidualPayloadBits,
			frameNo:     highFrameMBAFFFrameP8x8NoResidualPayloadBits,
			frameLuma:   highFrameMBAFFFrameP8x8LumaResidualPayloadBits,
			frameChroma: highFrameMBAFFFrameP8x8LumaChromaResidualPayloadBits,
		},
	}

	var cases []highFrameMBAFFPartitionedPDeblockCase
	add := func(name string, bitDepth int, mbType uint32, fieldFlag uint32, cbp uint32, payloadBits string, residualTailBits string, pFrameMD5 string, rawVideoMD5 string) {
		refFrameMD5 := high12FrameMBAFFIntraPCMFrameMD5
		if bitDepth == 14 {
			refFrameMD5 = high14FrameMBAFFIntraPCMFrameMD5
		}
		cases = append(cases, highFrameMBAFFPartitionedPDeblockCase{
			name:             name,
			bitDepth:         bitDepth,
			mbType:           mbType,
			fieldFlag:        fieldFlag,
			cbp:              cbp,
			payloadBits:      payloadBits,
			residualTailBits: residualTailBits,
			refFrameMD5:      refFrameMD5,
			pFrameMD5:        pFrameMD5,
			rawVideoMD5:      rawVideoMD5,
		})
	}
	addBitDepth := func(bitDepth int) {
		prefix := fmt.Sprintf("High%d", bitDepth)
		fieldNoPFrameMD5 := high12FrameMBAFFP16x16NoResidualPFrameMD5
		fieldNoRawVideoMD5 := high12FrameMBAFFP16x16NoResidualRawVideoMD5
		fieldLumaPFrameMD5 := high12FrameMBAFFP16x16LumaResidualPFrameMD5
		fieldLumaRawVideoMD5 := high12FrameMBAFFP16x16LumaResidualRawVideoMD5
		fieldChromaPFrameMD5 := high12FrameMBAFFP16x16LumaChromaResidualPFrameMD5
		fieldChromaRawVideoMD5 := high12FrameMBAFFP16x16LumaChromaResidualRawVideoMD5
		frameNoPFrameMD5 := high12FrameMBAFFFrameP16x16NoResidualPFrameMD5
		frameNoRawVideoMD5 := high12FrameMBAFFFrameP16x16NoResidualRawVideoMD5
		frameLumaPFrameMD5 := high12FrameMBAFFFrameP16x16LumaResidualPFrameMD5
		frameLumaRawVideoMD5 := high12FrameMBAFFFrameP16x16LumaResidualRawVideoMD5
		frameChromaPFrameMD5 := high12FrameMBAFFFrameP16x16LumaChromaResidualPFrameMD5
		frameChromaRawVideoMD5 := high12FrameMBAFFFrameP16x16LumaChromaResidualRawVideoMD5
		if bitDepth == 14 {
			fieldNoPFrameMD5 = high14FrameMBAFFP16x16NoResidualPFrameMD5
			fieldNoRawVideoMD5 = high14FrameMBAFFP16x16NoResidualRawVideoMD5
			fieldLumaPFrameMD5 = high14FrameMBAFFP16x16LumaResidualPFrameMD5
			fieldLumaRawVideoMD5 = high14FrameMBAFFP16x16LumaResidualRawVideoMD5
			fieldChromaPFrameMD5 = high14FrameMBAFFP16x16LumaChromaResidualPFrameMD5
			fieldChromaRawVideoMD5 = high14FrameMBAFFP16x16LumaChromaResidualRawVideoMD5
			frameNoPFrameMD5 = high14FrameMBAFFFrameP16x16NoResidualPFrameMD5
			frameNoRawVideoMD5 = high14FrameMBAFFFrameP16x16NoResidualRawVideoMD5
			frameLumaPFrameMD5 = high14FrameMBAFFFrameP16x16LumaResidualPFrameMD5
			frameLumaRawVideoMD5 = high14FrameMBAFFFrameP16x16LumaResidualRawVideoMD5
			frameChromaPFrameMD5 = high14FrameMBAFFFrameP16x16LumaChromaResidualPFrameMD5
			frameChromaRawVideoMD5 = high14FrameMBAFFFrameP16x16LumaChromaResidualRawVideoMD5
		}
		for _, shape := range shapes {
			add(prefix+"Field"+shape.name+"NoResidual", bitDepth, shape.mbType, 1, 0, shape.fieldNo, "", fieldNoPFrameMD5, fieldNoRawVideoMD5)
			add(prefix+"Field"+shape.name+"LumaResidual", bitDepth, shape.mbType, 1, 1, shape.fieldLuma, highFrameMBAFFP16x16LumaResidualTailBits, fieldLumaPFrameMD5, fieldLumaRawVideoMD5)
			add(prefix+"Field"+shape.name+"LumaChromaResidual", bitDepth, shape.mbType, 1, 33, shape.fieldChroma, highFrameMBAFFP16x16LumaChromaResidualTailBits, fieldChromaPFrameMD5, fieldChromaRawVideoMD5)
			add(prefix+"Frame"+shape.name+"NoResidual", bitDepth, shape.mbType, 0, 0, shape.frameNo, "", frameNoPFrameMD5, frameNoRawVideoMD5)
			add(prefix+"Frame"+shape.name+"LumaResidual", bitDepth, shape.mbType, 0, 1, shape.frameLuma, highFrameMBAFFP16x16LumaResidualTailBits, frameLumaPFrameMD5, frameLumaRawVideoMD5)
			add(prefix+"Frame"+shape.name+"LumaChromaResidual", bitDepth, shape.mbType, 0, 33, shape.frameChroma, highFrameMBAFFP16x16LumaChromaResidualTailBits, frameChromaPFrameMD5, frameChromaRawVideoMD5)
		}
	}
	addBitDepth(12)
	addBitDepth(14)
	return cases
}

func highFrameMBAFFPartitionedPDeblockBitstreamMD5() map[string]string {
	return map[string]string{
		"High12FieldP16x8NoResidualDeblockMode1":         "c96d6e5c566a22f3aef2772d41d9b587",
		"High12FieldP16x8NoResidualDeblockMode2":         "44a9835540e01499f018d21509584e5b",
		"High12FieldP16x8LumaResidualDeblockMode1":       "22e05df6fd86e67acfcc0d5d6ea733e4",
		"High12FieldP16x8LumaResidualDeblockMode2":       "eee25807d76406a23144fc2bb7a6485e",
		"High12FieldP16x8LumaChromaResidualDeblockMode1": "195389845645cdd59f084dc6bb5929d2",
		"High12FieldP16x8LumaChromaResidualDeblockMode2": "0a535cb89eb4a54c75d76d2d2ed0a4ce",
		"High12FrameP16x8NoResidualDeblockMode1":         "870197ab9198aafc95a50b9399f33f3d",
		"High12FrameP16x8NoResidualDeblockMode2":         "da5dcabb071ed9a91886d4f2cee4b3a2",
		"High12FrameP16x8LumaResidualDeblockMode1":       "5bb1ef2f76c7f317d428273a45a32eb6",
		"High12FrameP16x8LumaResidualDeblockMode2":       "220f7fa9693dcc6c09c9ffab7a5057c4",
		"High12FrameP16x8LumaChromaResidualDeblockMode1": "dfd1cb6f4d4d35de8886bc322866d2b3",
		"High12FrameP16x8LumaChromaResidualDeblockMode2": "0e6af9fc0c61251e510b69e85e7b6cdf",
		"High12FieldP8x16NoResidualDeblockMode1":         "f8ca7e271741cbd42f33a025a96981f8",
		"High12FieldP8x16NoResidualDeblockMode2":         "3e0a7baf7e3dff69d7193fa293e19b1f",
		"High12FieldP8x16LumaResidualDeblockMode1":       "2bb3a401360084d632351058404b2ba6",
		"High12FieldP8x16LumaResidualDeblockMode2":       "af05c51bad929384cee724954a2eed76",
		"High12FieldP8x16LumaChromaResidualDeblockMode1": "5d7b6979150541ed1cc57dc9f378af62",
		"High12FieldP8x16LumaChromaResidualDeblockMode2": "12f9e9f50efec188fafab3b3f96c1592",
		"High12FrameP8x16NoResidualDeblockMode1":         "3380210c2402b45af9c07a4301d9773c",
		"High12FrameP8x16NoResidualDeblockMode2":         "00205f754a811bce44776e5c492e34ca",
		"High12FrameP8x16LumaResidualDeblockMode1":       "ece1a39d667e1456960c28411bb51fd3",
		"High12FrameP8x16LumaResidualDeblockMode2":       "2e121220e18ad2a1f927e46be51fdb37",
		"High12FrameP8x16LumaChromaResidualDeblockMode1": "110e30dd50d8d8d460ed04e8dae43b54",
		"High12FrameP8x16LumaChromaResidualDeblockMode2": "9f2ee4e87620f439bef1efe73baf480b",
		"High12FieldP8x8NoResidualDeblockMode1":          "1f86184ca5b73d33ff06a3d924fd484a",
		"High12FieldP8x8NoResidualDeblockMode2":          "54438f307b9779ebb04c635e7565c353",
		"High12FieldP8x8LumaResidualDeblockMode1":        "7543ad11487f8af6755e27c54ee0b10d",
		"High12FieldP8x8LumaResidualDeblockMode2":        "4aabd8a224863b0b2e7e245b39c9db4d",
		"High12FieldP8x8LumaChromaResidualDeblockMode1":  "121a9cc470bb79da08fb6aa055a78335",
		"High12FieldP8x8LumaChromaResidualDeblockMode2":  "ce2f3b6df1bd88a4e21c0861a3024bd2",
		"High12FrameP8x8NoResidualDeblockMode1":          "730189059d9164d9e40f282ae740b82d",
		"High12FrameP8x8NoResidualDeblockMode2":          "ebb22470f16a0cf35328cd07befae510",
		"High12FrameP8x8LumaResidualDeblockMode1":        "de655918dbd7578d9a2bd4108e73f1f0",
		"High12FrameP8x8LumaResidualDeblockMode2":        "275465cff89cf1958d708179f4dc2266",
		"High12FrameP8x8LumaChromaResidualDeblockMode1":  "6e6aae6af43d02ff632da9b230530b6c",
		"High12FrameP8x8LumaChromaResidualDeblockMode2":  "b050d45a7fa50c5dfe328f7c67c6fefe",
		"High14FieldP16x8NoResidualDeblockMode1":         "2deea9ab09e9abf105f14ed302a81e77",
		"High14FieldP16x8NoResidualDeblockMode2":         "3a50a3ba5473310012e88fe271e6fd28",
		"High14FieldP16x8LumaResidualDeblockMode1":       "2cd9cbd0575125ef9874a7c1dd4a9e5d",
		"High14FieldP16x8LumaResidualDeblockMode2":       "c872691ad37e0aeff013caa56a8fd1ed",
		"High14FieldP16x8LumaChromaResidualDeblockMode1": "30213e1a323a7e9c3e80466eb43c1740",
		"High14FieldP16x8LumaChromaResidualDeblockMode2": "0dcae12d45da61b91d5221ab694b7214",
		"High14FrameP16x8NoResidualDeblockMode1":         "d1d145dff6dfa51128b36131cdbbfb24",
		"High14FrameP16x8NoResidualDeblockMode2":         "110b7ef427edb0dc489fb7275f2b2d8d",
		"High14FrameP16x8LumaResidualDeblockMode1":       "d8b4db720a5a3e1c22cbd1214c9bf754",
		"High14FrameP16x8LumaResidualDeblockMode2":       "23fe2ea0ed5f1c6906c25c3c79c4af03",
		"High14FrameP16x8LumaChromaResidualDeblockMode1": "2b37e39d8b387b871703553942142a0f",
		"High14FrameP16x8LumaChromaResidualDeblockMode2": "bb62d52e71d0fdbc514f08bf26973fae",
		"High14FieldP8x16NoResidualDeblockMode1":         "7df49636bbef045c89bff22891a4731e",
		"High14FieldP8x16NoResidualDeblockMode2":         "04b421684e9b54cebe91e99c7340608d",
		"High14FieldP8x16LumaResidualDeblockMode1":       "1725dbf2dc58006ca5442cc5dfe90493",
		"High14FieldP8x16LumaResidualDeblockMode2":       "e7230694a1654f9efc383cddb5478570",
		"High14FieldP8x16LumaChromaResidualDeblockMode1": "208e9e307eb6824a8d92f74567f7ce2e",
		"High14FieldP8x16LumaChromaResidualDeblockMode2": "5e2393934dcffc51d9e02c9518db88a3",
		"High14FrameP8x16NoResidualDeblockMode1":         "195ac6cf58f25bf52ddc8973b4d9ba72",
		"High14FrameP8x16NoResidualDeblockMode2":         "60ec1ce87ebfa0d4067b6a3a89eef4dc",
		"High14FrameP8x16LumaResidualDeblockMode1":       "34931c414ea5def9383d18aaf8057c24",
		"High14FrameP8x16LumaResidualDeblockMode2":       "ce588dc185dc86253c471919eea8639e",
		"High14FrameP8x16LumaChromaResidualDeblockMode1": "701b4e71301c5b13a02b1ee17633836d",
		"High14FrameP8x16LumaChromaResidualDeblockMode2": "8cf2cefc40521543fa28e8f6ce44c54a",
		"High14FieldP8x8NoResidualDeblockMode1":          "7f73fffa8276b868f99604307a763bd7",
		"High14FieldP8x8NoResidualDeblockMode2":          "fe790e4b120dcf3e87b626227718dba6",
		"High14FieldP8x8LumaResidualDeblockMode1":        "f05885cbd700ba89096aaaa370e11a8e",
		"High14FieldP8x8LumaResidualDeblockMode2":        "0d5d7f58f72eac58161a4bdbb85b9890",
		"High14FieldP8x8LumaChromaResidualDeblockMode1":  "5a0b64cdc3706871bc2b710de9c80e6d",
		"High14FieldP8x8LumaChromaResidualDeblockMode2":  "16e86485d47304da2c49683db2194ac2",
		"High14FrameP8x8NoResidualDeblockMode1":          "daac5e5e849581836eecc934d93d6dce",
		"High14FrameP8x8NoResidualDeblockMode2":          "e197f369236f17b18aae24852608e13b",
		"High14FrameP8x8LumaResidualDeblockMode1":        "c6d9afa7d78f9cb420f03d78599cc60a",
		"High14FrameP8x8LumaResidualDeblockMode2":        "135d6936c40959d9bf451cf88101cf6e",
		"High14FrameP8x8LumaChromaResidualDeblockMode1":  "4297a73a8a9970a309f58d5d159afbcc",
		"High14FrameP8x8LumaChromaResidualDeblockMode2":  "1ff62527fb866fcf4999fff9cddb28a2",
	}
}

func highFrameMBAFFPartitionedPSparseResidualCases() []highFrameMBAFFPartitionedPSparseResidualCase {
	return []highFrameMBAFFPartitionedPSparseResidualCase{
		{
			name:         "High12FieldP16x8NoResidual",
			bitDepth:     12,
			mbType:       1,
			fieldFlag:    1,
			cbp:          0,
			payloadBits:  highFrameMBAFFP16x8NoResidualPayloadBits,
			bitstreamMD5: "3d3bf6abc41a15cf2cc00ef284371617",
			refFrameMD5:  high12FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:    high12FrameMBAFFP16x16NoResidualPFrameMD5,
			rawVideoMD5:  high12FrameMBAFFP16x16NoResidualRawVideoMD5,
		},
		{
			name:         "High12FieldP8x16NoResidual",
			bitDepth:     12,
			mbType:       2,
			fieldFlag:    1,
			cbp:          0,
			payloadBits:  highFrameMBAFFP8x16NoResidualPayloadBits,
			bitstreamMD5: "9fbedccafb9aaba6ba6b4a6ad7365d71",
			refFrameMD5:  high12FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:    high12FrameMBAFFP16x16NoResidualPFrameMD5,
			rawVideoMD5:  high12FrameMBAFFP16x16NoResidualRawVideoMD5,
		},
		{
			name:         "High12FieldP8x8NoResidual",
			bitDepth:     12,
			mbType:       3,
			fieldFlag:    1,
			cbp:          0,
			payloadBits:  highFrameMBAFFP8x8NoResidualPayloadBits,
			bitstreamMD5: "4f26ad5f1dc3c0689ab197fcced92321",
			refFrameMD5:  high12FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:    high12FrameMBAFFP16x16NoResidualPFrameMD5,
			rawVideoMD5:  high12FrameMBAFFP16x16NoResidualRawVideoMD5,
		},
		{
			name:             "High12FieldP16x8LumaResidual",
			bitDepth:         12,
			mbType:           1,
			fieldFlag:        1,
			cbp:              1,
			residualTailBits: highFrameMBAFFP16x16LumaResidualTailBits,
			payloadBits:      highFrameMBAFFP16x8LumaResidualPayloadBits,
			bitstreamMD5:     "eec9957187e45baed80178b851efcbb1",
			refFrameMD5:      high12FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:        high12FrameMBAFFP16x16LumaResidualPFrameMD5,
			rawVideoMD5:      high12FrameMBAFFP16x16LumaResidualRawVideoMD5,
		},
		{
			name:             "High12FieldP8x16LumaResidual",
			bitDepth:         12,
			mbType:           2,
			fieldFlag:        1,
			cbp:              1,
			residualTailBits: highFrameMBAFFP16x16LumaResidualTailBits,
			payloadBits:      highFrameMBAFFP8x16LumaResidualPayloadBits,
			bitstreamMD5:     "2c7a37599c7d4fa72e74fc778b2f0b32",
			refFrameMD5:      high12FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:        high12FrameMBAFFP16x16LumaResidualPFrameMD5,
			rawVideoMD5:      high12FrameMBAFFP16x16LumaResidualRawVideoMD5,
		},
		{
			name:             "High12FieldP8x8LumaResidual",
			bitDepth:         12,
			mbType:           3,
			fieldFlag:        1,
			cbp:              1,
			residualTailBits: highFrameMBAFFP16x16LumaResidualTailBits,
			payloadBits:      highFrameMBAFFP8x8LumaResidualPayloadBits,
			bitstreamMD5:     "dbc41bd8ff4c86385c79bee56a01f913",
			refFrameMD5:      high12FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:        high12FrameMBAFFP16x16LumaResidualPFrameMD5,
			rawVideoMD5:      high12FrameMBAFFP16x16LumaResidualRawVideoMD5,
		},
		{
			name:         "High12FrameP16x8NoResidual",
			bitDepth:     12,
			mbType:       1,
			fieldFlag:    0,
			cbp:          0,
			payloadBits:  highFrameMBAFFFrameP16x8NoResidualPayloadBits,
			bitstreamMD5: "1b53f836e7518d33429f38bb942bd751",
			refFrameMD5:  high12FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:    high12FrameMBAFFFrameP16x16NoResidualPFrameMD5,
			rawVideoMD5:  high12FrameMBAFFFrameP16x16NoResidualRawVideoMD5,
		},
		{
			name:         "High12FrameP8x16NoResidual",
			bitDepth:     12,
			mbType:       2,
			fieldFlag:    0,
			cbp:          0,
			payloadBits:  highFrameMBAFFFrameP8x16NoResidualPayloadBits,
			bitstreamMD5: "720505d29440fcc42635e2c16a6f04df",
			refFrameMD5:  high12FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:    high12FrameMBAFFFrameP16x16NoResidualPFrameMD5,
			rawVideoMD5:  high12FrameMBAFFFrameP16x16NoResidualRawVideoMD5,
		},
		{
			name:         "High12FrameP8x8NoResidual",
			bitDepth:     12,
			mbType:       3,
			fieldFlag:    0,
			cbp:          0,
			payloadBits:  highFrameMBAFFFrameP8x8NoResidualPayloadBits,
			bitstreamMD5: "fb299371da0a273977146a81ba78c926",
			refFrameMD5:  high12FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:    high12FrameMBAFFFrameP16x16NoResidualPFrameMD5,
			rawVideoMD5:  high12FrameMBAFFFrameP16x16NoResidualRawVideoMD5,
		},
		{
			name:             "High12FrameP16x8LumaResidual",
			bitDepth:         12,
			mbType:           1,
			fieldFlag:        0,
			cbp:              1,
			residualTailBits: highFrameMBAFFP16x16LumaResidualTailBits,
			payloadBits:      highFrameMBAFFFrameP16x8LumaResidualPayloadBits,
			bitstreamMD5:     "ed75a503245c8738f451da3ee63a8dda",
			refFrameMD5:      high12FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:        high12FrameMBAFFFrameP16x16LumaResidualPFrameMD5,
			rawVideoMD5:      high12FrameMBAFFFrameP16x16LumaResidualRawVideoMD5,
		},
		{
			name:             "High12FrameP8x16LumaResidual",
			bitDepth:         12,
			mbType:           2,
			fieldFlag:        0,
			cbp:              1,
			residualTailBits: highFrameMBAFFP16x16LumaResidualTailBits,
			payloadBits:      highFrameMBAFFFrameP8x16LumaResidualPayloadBits,
			bitstreamMD5:     "4e96af0550d590fe7dc12e0114c11ed8",
			refFrameMD5:      high12FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:        high12FrameMBAFFFrameP16x16LumaResidualPFrameMD5,
			rawVideoMD5:      high12FrameMBAFFFrameP16x16LumaResidualRawVideoMD5,
		},
		{
			name:             "High12FrameP8x8LumaResidual",
			bitDepth:         12,
			mbType:           3,
			fieldFlag:        0,
			cbp:              1,
			residualTailBits: highFrameMBAFFP16x16LumaResidualTailBits,
			payloadBits:      highFrameMBAFFFrameP8x8LumaResidualPayloadBits,
			bitstreamMD5:     "0b7a6bf3dc5be20db610b3dacc996693",
			refFrameMD5:      high12FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:        high12FrameMBAFFFrameP16x16LumaResidualPFrameMD5,
			rawVideoMD5:      high12FrameMBAFFFrameP16x16LumaResidualRawVideoMD5,
		},
		{
			name:         "High14FieldP16x8NoResidual",
			bitDepth:     14,
			mbType:       1,
			fieldFlag:    1,
			cbp:          0,
			payloadBits:  highFrameMBAFFP16x8NoResidualPayloadBits,
			bitstreamMD5: "b7ced151f9ab553a33b2a2b4c8cc5094",
			refFrameMD5:  high14FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:    high14FrameMBAFFP16x16NoResidualPFrameMD5,
			rawVideoMD5:  high14FrameMBAFFP16x16NoResidualRawVideoMD5,
		},
		{
			name:         "High14FieldP8x16NoResidual",
			bitDepth:     14,
			mbType:       2,
			fieldFlag:    1,
			cbp:          0,
			payloadBits:  highFrameMBAFFP8x16NoResidualPayloadBits,
			bitstreamMD5: "11f794780a05b3df278e25918be95c89",
			refFrameMD5:  high14FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:    high14FrameMBAFFP16x16NoResidualPFrameMD5,
			rawVideoMD5:  high14FrameMBAFFP16x16NoResidualRawVideoMD5,
		},
		{
			name:         "High14FieldP8x8NoResidual",
			bitDepth:     14,
			mbType:       3,
			fieldFlag:    1,
			cbp:          0,
			payloadBits:  highFrameMBAFFP8x8NoResidualPayloadBits,
			bitstreamMD5: "80f84a6087d755750eb2b394171222db",
			refFrameMD5:  high14FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:    high14FrameMBAFFP16x16NoResidualPFrameMD5,
			rawVideoMD5:  high14FrameMBAFFP16x16NoResidualRawVideoMD5,
		},
		{
			name:             "High14FieldP16x8LumaResidual",
			bitDepth:         14,
			mbType:           1,
			fieldFlag:        1,
			cbp:              1,
			residualTailBits: highFrameMBAFFP16x16LumaResidualTailBits,
			payloadBits:      highFrameMBAFFP16x8LumaResidualPayloadBits,
			bitstreamMD5:     "2f3a635a64165d33b30c6239fe08141f",
			refFrameMD5:      high14FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:        high14FrameMBAFFP16x16LumaResidualPFrameMD5,
			rawVideoMD5:      high14FrameMBAFFP16x16LumaResidualRawVideoMD5,
		},
		{
			name:             "High14FieldP8x16LumaResidual",
			bitDepth:         14,
			mbType:           2,
			fieldFlag:        1,
			cbp:              1,
			residualTailBits: highFrameMBAFFP16x16LumaResidualTailBits,
			payloadBits:      highFrameMBAFFP8x16LumaResidualPayloadBits,
			bitstreamMD5:     "7bcd8368c0feaf7ec8378dd68ea5194e",
			refFrameMD5:      high14FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:        high14FrameMBAFFP16x16LumaResidualPFrameMD5,
			rawVideoMD5:      high14FrameMBAFFP16x16LumaResidualRawVideoMD5,
		},
		{
			name:             "High14FieldP8x8LumaResidual",
			bitDepth:         14,
			mbType:           3,
			fieldFlag:        1,
			cbp:              1,
			residualTailBits: highFrameMBAFFP16x16LumaResidualTailBits,
			payloadBits:      highFrameMBAFFP8x8LumaResidualPayloadBits,
			bitstreamMD5:     "f85202d0c6dc3d19b33ce2a2245f2ac2",
			refFrameMD5:      high14FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:        high14FrameMBAFFP16x16LumaResidualPFrameMD5,
			rawVideoMD5:      high14FrameMBAFFP16x16LumaResidualRawVideoMD5,
		},
		{
			name:         "High14FrameP16x8NoResidual",
			bitDepth:     14,
			mbType:       1,
			fieldFlag:    0,
			cbp:          0,
			payloadBits:  highFrameMBAFFFrameP16x8NoResidualPayloadBits,
			bitstreamMD5: "ce9b5e97a13751eff127aa90e5bb5c3b",
			refFrameMD5:  high14FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:    high14FrameMBAFFFrameP16x16NoResidualPFrameMD5,
			rawVideoMD5:  high14FrameMBAFFFrameP16x16NoResidualRawVideoMD5,
		},
		{
			name:         "High14FrameP8x16NoResidual",
			bitDepth:     14,
			mbType:       2,
			fieldFlag:    0,
			cbp:          0,
			payloadBits:  highFrameMBAFFFrameP8x16NoResidualPayloadBits,
			bitstreamMD5: "fe410c80ae8025f62b937a2ff3c564fb",
			refFrameMD5:  high14FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:    high14FrameMBAFFFrameP16x16NoResidualPFrameMD5,
			rawVideoMD5:  high14FrameMBAFFFrameP16x16NoResidualRawVideoMD5,
		},
		{
			name:         "High14FrameP8x8NoResidual",
			bitDepth:     14,
			mbType:       3,
			fieldFlag:    0,
			cbp:          0,
			payloadBits:  highFrameMBAFFFrameP8x8NoResidualPayloadBits,
			bitstreamMD5: "b854f71f4708f7814ff3450827cadfad",
			refFrameMD5:  high14FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:    high14FrameMBAFFFrameP16x16NoResidualPFrameMD5,
			rawVideoMD5:  high14FrameMBAFFFrameP16x16NoResidualRawVideoMD5,
		},
		{
			name:             "High14FrameP16x8LumaResidual",
			bitDepth:         14,
			mbType:           1,
			fieldFlag:        0,
			cbp:              1,
			residualTailBits: highFrameMBAFFP16x16LumaResidualTailBits,
			payloadBits:      highFrameMBAFFFrameP16x8LumaResidualPayloadBits,
			bitstreamMD5:     "b9714d5e3dc69b9235d4cb0fbd73ea96",
			refFrameMD5:      high14FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:        high14FrameMBAFFFrameP16x16LumaResidualPFrameMD5,
			rawVideoMD5:      high14FrameMBAFFFrameP16x16LumaResidualRawVideoMD5,
		},
		{
			name:             "High14FrameP8x16LumaResidual",
			bitDepth:         14,
			mbType:           2,
			fieldFlag:        0,
			cbp:              1,
			residualTailBits: highFrameMBAFFP16x16LumaResidualTailBits,
			payloadBits:      highFrameMBAFFFrameP8x16LumaResidualPayloadBits,
			bitstreamMD5:     "a4fd3bbc2e1837b5e167f879ffc9c22b",
			refFrameMD5:      high14FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:        high14FrameMBAFFFrameP16x16LumaResidualPFrameMD5,
			rawVideoMD5:      high14FrameMBAFFFrameP16x16LumaResidualRawVideoMD5,
		},
		{
			name:             "High14FrameP8x8LumaResidual",
			bitDepth:         14,
			mbType:           3,
			fieldFlag:        0,
			cbp:              1,
			residualTailBits: highFrameMBAFFP16x16LumaResidualTailBits,
			payloadBits:      highFrameMBAFFFrameP8x8LumaResidualPayloadBits,
			bitstreamMD5:     "331bb94c12da698095657459af8aef8a",
			refFrameMD5:      high14FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:        high14FrameMBAFFFrameP16x16LumaResidualPFrameMD5,
			rawVideoMD5:      high14FrameMBAFFFrameP16x16LumaResidualRawVideoMD5,
		},
	}
}

func highFrameMBAFFP16x16LumaChromaResidualCases() []highFrameMBAFFP16x16LumaChromaResidualCase {
	return []highFrameMBAFFP16x16LumaChromaResidualCase{
		{
			name:         "High12",
			bitDepth:     12,
			bitstreamMD5: high12FrameMBAFFP16x16LumaChromaResidualBitstreamMD5,
			refFrameMD5:  high12FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:    high12FrameMBAFFP16x16LumaChromaResidualPFrameMD5,
			rawVideoMD5:  high12FrameMBAFFP16x16LumaChromaResidualRawVideoMD5,
		},
		{
			name:         "High14",
			bitDepth:     14,
			bitstreamMD5: high14FrameMBAFFP16x16LumaChromaResidualBitstreamMD5,
			refFrameMD5:  high14FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:    high14FrameMBAFFP16x16LumaChromaResidualPFrameMD5,
			rawVideoMD5:  high14FrameMBAFFP16x16LumaChromaResidualRawVideoMD5,
		},
	}
}

func highFrameMBAFFPartitionedPLumaChromaResidualCases() []highFrameMBAFFPartitionedPLumaChromaResidualCase {
	return []highFrameMBAFFPartitionedPLumaChromaResidualCase{
		{
			name:         "High12P16x8",
			bitDepth:     12,
			mbType:       1,
			payloadBits:  highFrameMBAFFP16x8LumaChromaResidualPayloadBits,
			bitstreamMD5: high12FrameMBAFFP16x8LumaChromaResidualBitstreamMD5,
			refFrameMD5:  high12FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:    high12FrameMBAFFP16x8LumaChromaResidualPFrameMD5,
			rawVideoMD5:  high12FrameMBAFFP16x8LumaChromaResidualRawVideoMD5,
		},
		{
			name:         "High12P8x16",
			bitDepth:     12,
			mbType:       2,
			payloadBits:  highFrameMBAFFP8x16LumaChromaResidualPayloadBits,
			bitstreamMD5: high12FrameMBAFFP8x16LumaChromaResidualBitstreamMD5,
			refFrameMD5:  high12FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:    high12FrameMBAFFP8x16LumaChromaResidualPFrameMD5,
			rawVideoMD5:  high12FrameMBAFFP8x16LumaChromaResidualRawVideoMD5,
		},
		{
			name:         "High12P8x8",
			bitDepth:     12,
			mbType:       3,
			payloadBits:  highFrameMBAFFP8x8LumaChromaResidualPayloadBits,
			bitstreamMD5: high12FrameMBAFFP8x8LumaChromaResidualBitstreamMD5,
			refFrameMD5:  high12FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:    high12FrameMBAFFP8x8LumaChromaResidualPFrameMD5,
			rawVideoMD5:  high12FrameMBAFFP8x8LumaChromaResidualRawVideoMD5,
		},
		{
			name:         "High14P16x8",
			bitDepth:     14,
			mbType:       1,
			payloadBits:  highFrameMBAFFP16x8LumaChromaResidualPayloadBits,
			bitstreamMD5: high14FrameMBAFFP16x8LumaChromaResidualBitstreamMD5,
			refFrameMD5:  high14FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:    high14FrameMBAFFP16x8LumaChromaResidualPFrameMD5,
			rawVideoMD5:  high14FrameMBAFFP16x8LumaChromaResidualRawVideoMD5,
		},
		{
			name:         "High14P8x16",
			bitDepth:     14,
			mbType:       2,
			payloadBits:  highFrameMBAFFP8x16LumaChromaResidualPayloadBits,
			bitstreamMD5: high14FrameMBAFFP8x16LumaChromaResidualBitstreamMD5,
			refFrameMD5:  high14FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:    high14FrameMBAFFP8x16LumaChromaResidualPFrameMD5,
			rawVideoMD5:  high14FrameMBAFFP8x16LumaChromaResidualRawVideoMD5,
		},
		{
			name:         "High14P8x8",
			bitDepth:     14,
			mbType:       3,
			payloadBits:  highFrameMBAFFP8x8LumaChromaResidualPayloadBits,
			bitstreamMD5: high14FrameMBAFFP8x8LumaChromaResidualBitstreamMD5,
			refFrameMD5:  high14FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:    high14FrameMBAFFP8x8LumaChromaResidualPFrameMD5,
			rawVideoMD5:  high14FrameMBAFFP8x8LumaChromaResidualRawVideoMD5,
		},
	}
}

func highFrameMBAFFFrameCodedPLumaChromaResidualCases() []highFrameMBAFFFrameCodedPLumaChromaResidualCase {
	return []highFrameMBAFFFrameCodedPLumaChromaResidualCase{
		{
			name:         "High12FrameP16x16",
			bitDepth:     12,
			mbType:       0,
			payloadBits:  highFrameMBAFFFrameP16x16LumaChromaResidualPayloadBits,
			bitstreamMD5: high12FrameMBAFFFrameP16x16LumaChromaResidualBitstreamMD5,
			refFrameMD5:  high12FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:    high12FrameMBAFFFrameP16x16LumaChromaResidualPFrameMD5,
			rawVideoMD5:  high12FrameMBAFFFrameP16x16LumaChromaResidualRawVideoMD5,
		},
		{
			name:         "High12FrameP16x8",
			bitDepth:     12,
			mbType:       1,
			payloadBits:  highFrameMBAFFFrameP16x8LumaChromaResidualPayloadBits,
			bitstreamMD5: high12FrameMBAFFFrameP16x8LumaChromaResidualBitstreamMD5,
			refFrameMD5:  high12FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:    high12FrameMBAFFFrameP16x8LumaChromaResidualPFrameMD5,
			rawVideoMD5:  high12FrameMBAFFFrameP16x8LumaChromaResidualRawVideoMD5,
		},
		{
			name:         "High12FrameP8x16",
			bitDepth:     12,
			mbType:       2,
			payloadBits:  highFrameMBAFFFrameP8x16LumaChromaResidualPayloadBits,
			bitstreamMD5: high12FrameMBAFFFrameP8x16LumaChromaResidualBitstreamMD5,
			refFrameMD5:  high12FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:    high12FrameMBAFFFrameP8x16LumaChromaResidualPFrameMD5,
			rawVideoMD5:  high12FrameMBAFFFrameP8x16LumaChromaResidualRawVideoMD5,
		},
		{
			name:         "High12FrameP8x8",
			bitDepth:     12,
			mbType:       3,
			payloadBits:  highFrameMBAFFFrameP8x8LumaChromaResidualPayloadBits,
			bitstreamMD5: high12FrameMBAFFFrameP8x8LumaChromaResidualBitstreamMD5,
			refFrameMD5:  high12FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:    high12FrameMBAFFFrameP8x8LumaChromaResidualPFrameMD5,
			rawVideoMD5:  high12FrameMBAFFFrameP8x8LumaChromaResidualRawVideoMD5,
		},
		{
			name:         "High14FrameP16x16",
			bitDepth:     14,
			mbType:       0,
			payloadBits:  highFrameMBAFFFrameP16x16LumaChromaResidualPayloadBits,
			bitstreamMD5: high14FrameMBAFFFrameP16x16LumaChromaResidualBitstreamMD5,
			refFrameMD5:  high14FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:    high14FrameMBAFFFrameP16x16LumaChromaResidualPFrameMD5,
			rawVideoMD5:  high14FrameMBAFFFrameP16x16LumaChromaResidualRawVideoMD5,
		},
		{
			name:         "High14FrameP16x8",
			bitDepth:     14,
			mbType:       1,
			payloadBits:  highFrameMBAFFFrameP16x8LumaChromaResidualPayloadBits,
			bitstreamMD5: high14FrameMBAFFFrameP16x8LumaChromaResidualBitstreamMD5,
			refFrameMD5:  high14FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:    high14FrameMBAFFFrameP16x8LumaChromaResidualPFrameMD5,
			rawVideoMD5:  high14FrameMBAFFFrameP16x8LumaChromaResidualRawVideoMD5,
		},
		{
			name:         "High14FrameP8x16",
			bitDepth:     14,
			mbType:       2,
			payloadBits:  highFrameMBAFFFrameP8x16LumaChromaResidualPayloadBits,
			bitstreamMD5: high14FrameMBAFFFrameP8x16LumaChromaResidualBitstreamMD5,
			refFrameMD5:  high14FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:    high14FrameMBAFFFrameP8x16LumaChromaResidualPFrameMD5,
			rawVideoMD5:  high14FrameMBAFFFrameP8x16LumaChromaResidualRawVideoMD5,
		},
		{
			name:         "High14FrameP8x8",
			bitDepth:     14,
			mbType:       3,
			payloadBits:  highFrameMBAFFFrameP8x8LumaChromaResidualPayloadBits,
			bitstreamMD5: high14FrameMBAFFFrameP8x8LumaChromaResidualBitstreamMD5,
			refFrameMD5:  high14FrameMBAFFIntraPCMFrameMD5,
			pFrameMD5:    high14FrameMBAFFFrameP8x8LumaChromaResidualPFrameMD5,
			rawVideoMD5:  high14FrameMBAFFFrameP8x8LumaChromaResidualRawVideoMD5,
		},
	}
}

func highFrameMBAFFPSkipNoResidualFixture(bitDepth int) []byte {
	var data []byte
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALSPS), highFrameMBAFFInterSPSRBSP(bitDepth)))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALPPS), highIntraPCMPPSRBSP(bitDepth)))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALIDRSlice), highFrameMBAFFIntraPCMSliceRBSP(bitDepth)))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALSlice), highFrameMBAFFPSkipNoResidualSliceRBSP()))
	return data
}

func highFrameMBAFFP16x16NoResidualFixture(bitDepth int) []byte {
	var data []byte
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALSPS), highFrameMBAFFInterSPSRBSP(bitDepth)))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALPPS), highIntraPCMPPSRBSP(bitDepth)))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALIDRSlice), highFrameMBAFFIntraPCMSliceRBSP(bitDepth)))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALSlice), highFrameMBAFFP16x16NoResidualSliceRBSP()))
	return data
}

func highFrameMBAFFP16x16LumaResidualFixture(bitDepth int) []byte {
	var data []byte
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALSPS), highFrameMBAFFInterSPSRBSP(bitDepth)))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALPPS), highIntraPCMPPSRBSP(bitDepth)))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALIDRSlice), highFrameMBAFFIntraPCMSliceRBSP(bitDepth)))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALSlice), highFrameMBAFFP16x16LumaResidualSliceRBSP()))
	return data
}

func highFrameMBAFFFrameP16x16NoResidualFixture(bitDepth int) []byte {
	var data []byte
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALSPS), highFrameMBAFFInterSPSRBSP(bitDepth)))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALPPS), highIntraPCMPPSRBSP(bitDepth)))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALIDRSlice), highFrameMBAFFIntraPCMSliceRBSP(bitDepth)))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALSlice), highFrameMBAFFPInterSliceRBSP(highFrameMBAFFFrameP16x16NoResidualPayloadBits)))
	return data
}

func highFrameMBAFFFrameP16x16LumaResidualFixture(bitDepth int) []byte {
	var data []byte
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALSPS), highFrameMBAFFInterSPSRBSP(bitDepth)))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALPPS), highIntraPCMPPSRBSP(bitDepth)))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALIDRSlice), highFrameMBAFFIntraPCMSliceRBSP(bitDepth)))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALSlice), highFrameMBAFFPInterSliceRBSP(highFrameMBAFFFrameP16x16LumaResidualPayloadBits)))
	return data
}

func highFrameMBAFFP16x16DeblockFixture(tt highFrameMBAFFP16x16DeblockCase) []byte {
	var data []byte
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALSPS), highFrameMBAFFInterSPSRBSP(tt.bitDepth)))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALPPS), highIntraPCMPPSRBSP(tt.bitDepth)))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALIDRSlice), highFrameMBAFFIntraPCMSliceRBSP(tt.bitDepth)))
	if tt.pskip {
		data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALSlice), highFrameMBAFFPSkipNoResidualSliceRBSPWithDeblock(tt.disableDeblockingFilterIDC)))
	} else {
		data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALSlice), highFrameMBAFFPInterSliceRBSPWithDeblock(tt.payloadBits, tt.disableDeblockingFilterIDC)))
	}
	return data
}

func highFrameMBAFFPartitionedPDeblockFixture(tt highFrameMBAFFPartitionedPDeblockCase) []byte {
	var data []byte
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALSPS), highFrameMBAFFInterSPSRBSP(tt.bitDepth)))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALPPS), highIntraPCMPPSRBSP(tt.bitDepth)))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALIDRSlice), highFrameMBAFFIntraPCMSliceRBSP(tt.bitDepth)))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALSlice), highFrameMBAFFPInterSliceRBSPWithDeblock(tt.payloadBits, tt.disableDeblockingFilterIDC)))
	return data
}

func highFrameMBAFFPartitionedPSparseResidualFixture(bitDepth int, payloadBits string) []byte {
	var data []byte
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALSPS), highFrameMBAFFInterSPSRBSP(bitDepth)))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALPPS), highIntraPCMPPSRBSP(bitDepth)))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALIDRSlice), highFrameMBAFFIntraPCMSliceRBSP(bitDepth)))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALSlice), highFrameMBAFFPInterSliceRBSP(payloadBits)))
	return data
}

func highFrameMBAFFP16x16LumaChromaResidualFixture(bitDepth int) []byte {
	var data []byte
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALSPS), highFrameMBAFFInterSPSRBSP(bitDepth)))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALPPS), highIntraPCMPPSRBSP(bitDepth)))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALIDRSlice), highFrameMBAFFIntraPCMSliceRBSP(bitDepth)))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALSlice), highFrameMBAFFP16x16LumaChromaResidualSliceRBSP()))
	return data
}

func highFrameMBAFFPartitionedPLumaChromaResidualFixture(bitDepth int, payloadBits string) []byte {
	var data []byte
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALSPS), highFrameMBAFFInterSPSRBSP(bitDepth)))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALPPS), highIntraPCMPPSRBSP(bitDepth)))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALIDRSlice), highFrameMBAFFIntraPCMSliceRBSP(bitDepth)))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALSlice), highFrameMBAFFPartitionedPLumaChromaResidualSliceRBSP(payloadBits)))
	return data
}

func highFrameMBAFFFrameCodedPLumaChromaResidualFixture(bitDepth int, payloadBits string) []byte {
	var data []byte
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALSPS), highFrameMBAFFInterSPSRBSP(bitDepth)))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALPPS), highIntraPCMPPSRBSP(bitDepth)))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALIDRSlice), highFrameMBAFFIntraPCMSliceRBSP(bitDepth)))
	data = appendAnnexBNAL(data, highIntraPCMNAL(byte(0x60|h264.NALSlice), highFrameMBAFFPartitionedPLumaChromaResidualSliceRBSP(payloadBits)))
	return data
}

func highFrameMBAFFInterSPSRBSP(bitDepth int) []byte {
	var b decoderSEIBitBuilder
	b.writeBits(244, 8)
	b.writeBits(0, 8)
	b.writeBits(10, 8)
	b.writeUE(0)
	b.writeUE(1)
	b.writeUE(uint32(bitDepth - 8))
	b.writeUE(uint32(bitDepth - 8))
	b.writeBit(0)
	b.writeBit(0)
	b.writeUE(0)
	b.writeUE(2)
	b.writeUE(1)
	b.writeBit(0)
	b.writeUE(0)
	b.writeUE(0)
	b.writeBit(0)
	b.writeBit(1)
	b.writeBit(1)
	b.writeBit(0)
	b.writeBit(0)
	return b.rbsp()
}

func highFrameMBAFFPSkipNoResidualSliceRBSP() []byte {
	return highFrameMBAFFPSkipNoResidualSliceRBSPWithDeblock(1)
}

func highFrameMBAFFPSkipNoResidualSliceRBSPWithDeblock(disableDeblockingFilterIDC uint32) []byte {
	var b decoderSEIBitBuilder
	b.writeUE(0)
	b.writeUE(0)
	b.writeUE(0)
	b.writeBits(1, 4)
	b.writeBit(0)
	b.writeBit(0)
	b.writeBit(0)
	b.writeBit(0)
	b.writeSE(0)
	writeHighCAVLCDeblockSyntax(&b, disableDeblockingFilterIDC)
	b.writeUE(2)
	return b.rbsp()
}

func highFrameMBAFFP16x16NoResidualSliceRBSP() []byte {
	var b decoderSEIBitBuilder
	b.writeUE(0)
	b.writeUE(0)
	b.writeUE(0)
	b.writeBits(1, 4)
	b.writeBit(0)
	b.writeBit(0)
	b.writeBit(0)
	b.writeBit(0)
	b.writeSE(0)
	b.writeUE(1)
	highIntra16x16WritePayloadBits(&b, highFrameMBAFFP16x16NoResidualPayloadBits)
	return b.rbsp()
}

func highFrameMBAFFP16x16LumaResidualSliceRBSP() []byte {
	var b decoderSEIBitBuilder
	b.writeUE(0)
	b.writeUE(0)
	b.writeUE(0)
	b.writeBits(1, 4)
	b.writeBit(0)
	b.writeBit(0)
	b.writeBit(0)
	b.writeBit(0)
	b.writeSE(0)
	b.writeUE(1)
	highIntra16x16WritePayloadBits(&b, highFrameMBAFFP16x16LumaResidualPayloadBits)
	return b.rbsp()
}

func highFrameMBAFFP16x16LumaChromaResidualSliceRBSP() []byte {
	var b decoderSEIBitBuilder
	b.writeUE(0)
	b.writeUE(0)
	b.writeUE(0)
	b.writeBits(1, 4)
	b.writeBit(0)
	b.writeBit(0)
	b.writeBit(0)
	b.writeBit(0)
	b.writeSE(0)
	b.writeUE(1)
	highIntra16x16WritePayloadBits(&b, highFrameMBAFFP16x16LumaChromaResidualPayloadBits)
	return b.rbsp()
}

func highFrameMBAFFPartitionedPLumaChromaResidualSliceRBSP(payloadBits string) []byte {
	return highFrameMBAFFPInterSliceRBSP(payloadBits)
}

func highFrameMBAFFPInterSliceRBSP(payloadBits string) []byte {
	return highFrameMBAFFPInterSliceRBSPWithDeblock(payloadBits, 1)
}

func highFrameMBAFFPInterSliceRBSPWithDeblock(payloadBits string, disableDeblockingFilterIDC uint32) []byte {
	var b decoderSEIBitBuilder
	b.writeUE(0)
	b.writeUE(0)
	b.writeUE(0)
	b.writeBits(1, 4)
	b.writeBit(0)
	b.writeBit(0)
	b.writeBit(0)
	b.writeBit(0)
	b.writeSE(0)
	writeHighCAVLCDeblockSyntax(&b, disableDeblockingFilterIDC)
	highIntra16x16WritePayloadBits(&b, payloadBits)
	return b.rbsp()
}

func assertHighFrameMBAFFPSkipNoResidualFixtureSyntax(t *testing.T, data []byte, tt highFrameMBAFFPSkipNoResidualCase) {
	t.Helper()
	assertHighFrameMBAFFP16x16NoResidualFixtureSyntax(t, data, tt)
}

func assertHighFrameMBAFFPSkipNoResidualFrames(t *testing.T, frames []*Frame, tt highFrameMBAFFPSkipNoResidualCase) {
	t.Helper()
	assertHighFrameMBAFFP16x16NoResidualFrames(t, frames, tt)
}

func assertHighFrameMBAFFP16x16NoResidualFixtureSyntax(t *testing.T, data []byte, tt highFrameMBAFFP16x16NoResidualCase) {
	t.Helper()
	parseHighFrameMBAFFInterFixtureSyntax(t, data, tt)
}

func assertHighFrameMBAFFP16x16LumaResidualFixtureSyntax(t *testing.T, data []byte, tt highFrameMBAFFP16x16LumaResidualCase) {
	t.Helper()
	assertHighFrameMBAFFP16x16NoResidualFixtureSyntax(t, data, tt)
	nals, spsList, ppsList := parseHighFrameMBAFFInterFixtureSyntax(t, data, tt)
	pair := readHighFrameMBAFFCAVLCP16x16Pair(t, nals[1], spsList[0], ppsList[0], highFrameMBAFFP16x16LumaResidualTailBits)
	if pair.fieldFlag != 1 {
		t.Fatalf("%s frame-MBAFF P pair field flag = %d, want field-coded", tt.name, pair.fieldFlag)
	}
	for i, mb := range []highFrameMBAFFCAVLCP16x16Macroblock{pair.top, pair.bottom} {
		if mb.skipRun != 0 || mb.mbType != 0 || mb.refIdxFlag != 1 || mb.cbp != 1 {
			t.Fatalf("%s P macroblock[%d] skip/mb_type/ref_idx_flag/cbp = %d/%d/%d/%d (code %d), want field-coded P16x16 luma residual",
				tt.name, i, mb.skipRun, mb.mbType, mb.refIdxFlag, mb.cbp, mb.cbpCode)
		}
	}
}

func assertHighFrameMBAFFP16x16LumaChromaResidualFixtureSyntax(t *testing.T, data []byte, tt highFrameMBAFFP16x16LumaChromaResidualCase) {
	t.Helper()
	assertHighFrameMBAFFP16x16NoResidualFixtureSyntax(t, data, tt)
	nals, spsList, ppsList := parseHighFrameMBAFFInterFixtureSyntax(t, data, tt)
	pair := readHighFrameMBAFFCAVLCP16x16Pair(t, nals[1], spsList[0], ppsList[0], highFrameMBAFFP16x16LumaChromaResidualTailBits)
	if pair.fieldFlag != 1 {
		t.Fatalf("%s frame-MBAFF P pair field flag = %d, want field-coded", tt.name, pair.fieldFlag)
	}
	for i, mb := range []highFrameMBAFFCAVLCP16x16Macroblock{pair.top, pair.bottom} {
		if mb.skipRun != 0 || mb.mbType != 0 || mb.refIdxFlag != 1 || mb.cbp != 33 {
			t.Fatalf("%s P macroblock[%d] skip/mb_type/ref_idx_flag/cbp = %d/%d/%d/%d (code %d), want field-coded P16x16 luma+chroma residual",
				tt.name, i, mb.skipRun, mb.mbType, mb.refIdxFlag, mb.cbp, mb.cbpCode)
		}
	}
}

func assertHighFrameMBAFFFrameP16x16NoResidualFixtureSyntax(t *testing.T, data []byte, tt highFrameMBAFFFrameP16x16NoResidualCase) {
	t.Helper()
	nals, spsList, ppsList := parseHighFrameMBAFFInterFixtureSyntax(t, data, tt)
	pair := readHighFrameMBAFFCAVLCP16x16Pair(t, nals[1], spsList[0], ppsList[0], "")
	if pair.fieldFlag != 0 {
		t.Fatalf("%s frame-MBAFF P pair field flag = %d, want frame-coded", tt.name, pair.fieldFlag)
	}
	for i, mb := range []highFrameMBAFFCAVLCP16x16Macroblock{pair.top, pair.bottom} {
		if mb.skipRun != 0 || mb.mbType != 0 || mb.refIdxFlag != 0 || mb.cbp != 0 {
			t.Fatalf("%s frame-coded P macroblock[%d] skip/mb_type/ref_idx_flag/cbp = %d/%d/%d/%d (code %d), want P16x16 no residual",
				tt.name, i, mb.skipRun, mb.mbType, mb.refIdxFlag, mb.cbp, mb.cbpCode)
		}
	}
}

func assertHighFrameMBAFFFrameP16x16LumaResidualFixtureSyntax(t *testing.T, data []byte, tt highFrameMBAFFFrameP16x16LumaResidualCase) {
	t.Helper()
	nals, spsList, ppsList := parseHighFrameMBAFFInterFixtureSyntax(t, data, tt)
	pair := readHighFrameMBAFFCAVLCP16x16Pair(t, nals[1], spsList[0], ppsList[0], highFrameMBAFFP16x16LumaResidualTailBits)
	if pair.fieldFlag != 0 {
		t.Fatalf("%s frame-MBAFF P pair field flag = %d, want frame-coded", tt.name, pair.fieldFlag)
	}
	for i, mb := range []highFrameMBAFFCAVLCP16x16Macroblock{pair.top, pair.bottom} {
		if mb.skipRun != 0 || mb.mbType != 0 || mb.refIdxFlag != 0 || mb.cbp != 1 {
			t.Fatalf("%s frame-coded P macroblock[%d] skip/mb_type/ref_idx_flag/cbp = %d/%d/%d/%d (code %d), want P16x16 luma residual",
				tt.name, i, mb.skipRun, mb.mbType, mb.refIdxFlag, mb.cbp, mb.cbpCode)
		}
	}
}

func assertHighFrameMBAFFP16x16DeblockFixtureSyntax(t *testing.T, data []byte, tt highFrameMBAFFP16x16DeblockCase) {
	t.Helper()
	parseCase := highFrameMBAFFP16x16NoResidualCase{
		name:         tt.name,
		bitDepth:     tt.bitDepth,
		bitstreamMD5: tt.bitstreamMD5,
		refFrameMD5:  tt.refFrameMD5,
		pFrameMD5:    tt.pFrameMD5,
		rawVideoMD5:  tt.rawVideoMD5,
	}
	nals, spsList, ppsList := parseHighFrameMBAFFInterFixtureSyntaxWithDeblock(t, data, parseCase, tt.deblockMode)
	if tt.pskip {
		skipRun := readHighFrameMBAFFPSkipRun(t, nals[1], spsList[0], ppsList[0], tt.disableDeblockingFilterIDC)
		if skipRun != 2 {
			t.Fatalf("%s frame-MBAFF P-skip skip_run = %d, want macroblock-pair skip_run 2", tt.name, skipRun)
		}
		return
	}

	pair := readHighFrameMBAFFCAVLCP16x16PairWithDeblock(t, nals[1], spsList[0], ppsList[0], tt.residualTailBits, tt.disableDeblockingFilterIDC)
	if pair.fieldFlag != tt.fieldFlag {
		t.Fatalf("%s frame-MBAFF P16x16 pair field flag = %d, want %d", tt.name, pair.fieldFlag, tt.fieldFlag)
	}
	for i, mb := range []highFrameMBAFFCAVLCP16x16Macroblock{pair.top, pair.bottom} {
		if mb.skipRun != 0 || mb.mbType != 0 || mb.refIdxFlag != tt.fieldFlag || mb.cbp != tt.cbp {
			t.Fatalf("%s P16x16 deblock macroblock[%d] skip/mb_type/ref_idx_flag/cbp = %d/%d/%d/%d (code %d), want fieldFlag %d cbp %d",
				tt.name, i, mb.skipRun, mb.mbType, mb.refIdxFlag, mb.cbp, mb.cbpCode, tt.fieldFlag, tt.cbp)
		}
	}
}

func assertHighFrameMBAFFPartitionedPDeblockFixtureSyntax(t *testing.T, data []byte, tt highFrameMBAFFPartitionedPDeblockCase) {
	t.Helper()
	parseCase := highFrameMBAFFP16x16NoResidualCase{
		name:         tt.name,
		bitDepth:     tt.bitDepth,
		bitstreamMD5: tt.bitstreamMD5,
		refFrameMD5:  tt.refFrameMD5,
		pFrameMD5:    tt.pFrameMD5,
		rawVideoMD5:  tt.rawVideoMD5,
	}
	nals, spsList, ppsList := parseHighFrameMBAFFInterFixtureSyntaxWithDeblock(t, data, parseCase, tt.deblockMode)
	pair := readHighFrameMBAFFCAVLCPartitionedPPairWithDeblock(t, nals[1], spsList[0], ppsList[0], tt.residualTailBits, tt.disableDeblockingFilterIDC)
	if pair.fieldFlag != tt.fieldFlag {
		t.Fatalf("%s frame-MBAFF partitioned P pair field flag = %d, want %d", tt.name, pair.fieldFlag, tt.fieldFlag)
	}
	wantRefIdxCount := highFrameMBAFFPartitionedPRefIdxCount(t, tt.mbType)
	for i, mb := range []highFrameMBAFFCAVLCPartitionedPMacroblock{pair.top, pair.bottom} {
		if mb.skipRun != 0 || mb.mbType != tt.mbType || mb.cbp != tt.cbp {
			t.Fatalf("%s partitioned P deblock macroblock[%d] skip/mb_type/cbp = %d/%d/%d (code %d), want mb_type %d cbp %d",
				tt.name, i, mb.skipRun, mb.mbType, mb.cbp, mb.cbpCode, tt.mbType, tt.cbp)
		}
		if mb.refIdxCount != wantRefIdxCount {
			t.Fatalf("%s partitioned P deblock macroblock[%d] ref_idx count = %d, want %d", tt.name, i, mb.refIdxCount, wantRefIdxCount)
		}
		wantRefIdxFlag := tt.fieldFlag
		for j := 0; j < mb.refIdxCount; j++ {
			if mb.refIdxFlags[j] != wantRefIdxFlag {
				t.Fatalf("%s partitioned P deblock macroblock[%d] ref_idx_l0[%d] flag = %d, want %d",
					tt.name, i, j, mb.refIdxFlags[j], wantRefIdxFlag)
			}
		}
		if tt.mbType == 3 {
			for j, subType := range mb.subMBType {
				if subType != 0 {
					t.Fatalf("%s partitioned P deblock macroblock[%d] sub_mb_type[%d] = %d, want P_L0_8x8", tt.name, i, j, subType)
				}
			}
		}
	}
}

func assertHighFrameMBAFFPartitionedPSparseResidualFixtureSyntax(t *testing.T, data []byte, tt highFrameMBAFFPartitionedPSparseResidualCase) {
	t.Helper()
	parseCase := highFrameMBAFFP16x16NoResidualCase{
		name:         tt.name,
		bitDepth:     tt.bitDepth,
		bitstreamMD5: tt.bitstreamMD5,
		refFrameMD5:  tt.refFrameMD5,
		pFrameMD5:    tt.pFrameMD5,
		rawVideoMD5:  tt.rawVideoMD5,
	}
	nals, spsList, ppsList := parseHighFrameMBAFFInterFixtureSyntax(t, data, parseCase)
	pair := readHighFrameMBAFFCAVLCPartitionedPPair(t, nals[1], spsList[0], ppsList[0], tt.residualTailBits)
	if pair.fieldFlag != tt.fieldFlag {
		t.Fatalf("%s frame-MBAFF P pair field flag = %d, want %d", tt.name, pair.fieldFlag, tt.fieldFlag)
	}
	wantRefIdxCount := highFrameMBAFFPartitionedPRefIdxCount(t, tt.mbType)
	for i, mb := range []highFrameMBAFFCAVLCPartitionedPMacroblock{pair.top, pair.bottom} {
		if mb.skipRun != 0 || mb.mbType != tt.mbType || mb.cbp != tt.cbp {
			t.Fatalf("%s partitioned P macroblock[%d] skip/mb_type/cbp = %d/%d/%d (code %d), want sparse residual cbp %d",
				tt.name, i, mb.skipRun, mb.mbType, mb.cbp, mb.cbpCode, tt.cbp)
		}
		if mb.refIdxCount != wantRefIdxCount {
			t.Fatalf("%s partitioned P macroblock[%d] ref_idx count = %d, want %d", tt.name, i, mb.refIdxCount, wantRefIdxCount)
		}
		wantRefIdxFlag := tt.fieldFlag
		for j := 0; j < mb.refIdxCount; j++ {
			if mb.refIdxFlags[j] != wantRefIdxFlag {
				t.Fatalf("%s partitioned P macroblock[%d] ref_idx_l0[%d] flag = %d, want %d",
					tt.name, i, j, mb.refIdxFlags[j], wantRefIdxFlag)
			}
		}
		if tt.mbType == 3 {
			for j, subType := range mb.subMBType {
				if subType != 0 {
					t.Fatalf("%s partitioned P macroblock[%d] sub_mb_type[%d] = %d, want P_L0_8x8", tt.name, i, j, subType)
				}
			}
		}
	}
}

func assertHighFrameMBAFFPartitionedPLumaChromaResidualFixtureSyntax(t *testing.T, data []byte, tt highFrameMBAFFPartitionedPLumaChromaResidualCase) {
	t.Helper()
	parseCase := highFrameMBAFFP16x16NoResidualCase{
		name:         tt.name,
		bitDepth:     tt.bitDepth,
		bitstreamMD5: tt.bitstreamMD5,
		refFrameMD5:  tt.refFrameMD5,
		pFrameMD5:    tt.pFrameMD5,
		rawVideoMD5:  tt.rawVideoMD5,
	}
	nals, spsList, ppsList := parseHighFrameMBAFFInterFixtureSyntax(t, data, parseCase)
	pair := readHighFrameMBAFFCAVLCPartitionedPPair(t, nals[1], spsList[0], ppsList[0], highFrameMBAFFP16x16LumaChromaResidualTailBits)
	if pair.fieldFlag != 1 {
		t.Fatalf("%s frame-MBAFF P pair field flag = %d, want field-coded", tt.name, pair.fieldFlag)
	}
	wantRefIdxCount := highFrameMBAFFPartitionedPRefIdxCount(t, tt.mbType)
	for i, mb := range []highFrameMBAFFCAVLCPartitionedPMacroblock{pair.top, pair.bottom} {
		if mb.skipRun != 0 || mb.mbType != tt.mbType || mb.cbp != 33 {
			t.Fatalf("%s P macroblock[%d] skip/mb_type/cbp = %d/%d/%d (code %d), want field-coded partitioned P luma+chroma residual",
				tt.name, i, mb.skipRun, mb.mbType, mb.cbp, mb.cbpCode)
		}
		if mb.refIdxCount != wantRefIdxCount {
			t.Fatalf("%s P macroblock[%d] ref_idx count = %d, want %d", tt.name, i, mb.refIdxCount, wantRefIdxCount)
		}
		for j := 0; j < mb.refIdxCount; j++ {
			if mb.refIdxFlags[j] != 1 {
				t.Fatalf("%s P macroblock[%d] ref_idx_l0[%d] flag = %d, want field ref 1", tt.name, i, j, mb.refIdxFlags[j])
			}
		}
		if tt.mbType == 3 {
			for j, subType := range mb.subMBType {
				if subType != 0 {
					t.Fatalf("%s P macroblock[%d] sub_mb_type[%d] = %d, want P_L0_8x8", tt.name, i, j, subType)
				}
			}
		}
	}
}

func assertHighFrameMBAFFFrameCodedPLumaChromaResidualFixtureSyntax(t *testing.T, data []byte, tt highFrameMBAFFFrameCodedPLumaChromaResidualCase) {
	t.Helper()
	parseCase := highFrameMBAFFP16x16NoResidualCase{
		name:         tt.name,
		bitDepth:     tt.bitDepth,
		bitstreamMD5: tt.bitstreamMD5,
		refFrameMD5:  tt.refFrameMD5,
		pFrameMD5:    tt.pFrameMD5,
		rawVideoMD5:  tt.rawVideoMD5,
	}
	nals, spsList, ppsList := parseHighFrameMBAFFInterFixtureSyntax(t, data, parseCase)
	if tt.mbType == 0 {
		pair := readHighFrameMBAFFCAVLCP16x16Pair(t, nals[1], spsList[0], ppsList[0], highFrameMBAFFP16x16LumaChromaResidualTailBits)
		if pair.fieldFlag != 0 {
			t.Fatalf("%s frame-MBAFF P pair field flag = %d, want frame-coded", tt.name, pair.fieldFlag)
		}
		for i, mb := range []highFrameMBAFFCAVLCP16x16Macroblock{pair.top, pair.bottom} {
			if mb.skipRun != 0 || mb.mbType != 0 || mb.refIdxFlag != 0 || mb.cbp != 33 {
				t.Fatalf("%s frame-coded P macroblock[%d] skip/mb_type/ref_idx_flag/cbp = %d/%d/%d/%d (code %d), want P16x16 luma+chroma residual",
					tt.name, i, mb.skipRun, mb.mbType, mb.refIdxFlag, mb.cbp, mb.cbpCode)
			}
		}
		return
	}

	pair := readHighFrameMBAFFCAVLCPartitionedPPair(t, nals[1], spsList[0], ppsList[0], highFrameMBAFFP16x16LumaChromaResidualTailBits)
	if pair.fieldFlag != 0 {
		t.Fatalf("%s frame-MBAFF P pair field flag = %d, want frame-coded", tt.name, pair.fieldFlag)
	}
	wantRefIdxCount := highFrameMBAFFPartitionedPRefIdxCount(t, tt.mbType)
	for i, mb := range []highFrameMBAFFCAVLCPartitionedPMacroblock{pair.top, pair.bottom} {
		if mb.skipRun != 0 || mb.mbType != tt.mbType || mb.cbp != 33 {
			t.Fatalf("%s frame-coded P macroblock[%d] skip/mb_type/cbp = %d/%d/%d (code %d), want partitioned P luma+chroma residual",
				tt.name, i, mb.skipRun, mb.mbType, mb.cbp, mb.cbpCode)
		}
		if mb.refIdxCount != wantRefIdxCount {
			t.Fatalf("%s frame-coded P macroblock[%d] ref_idx count = %d, want %d", tt.name, i, mb.refIdxCount, wantRefIdxCount)
		}
		for j := 0; j < mb.refIdxCount; j++ {
			if mb.refIdxFlags[j] != 0 {
				t.Fatalf("%s frame-coded P macroblock[%d] ref_idx_l0[%d] flag = %d, want implicit frame ref 0", tt.name, i, j, mb.refIdxFlags[j])
			}
		}
		if tt.mbType == 3 {
			for j, subType := range mb.subMBType {
				if subType != 0 {
					t.Fatalf("%s frame-coded P macroblock[%d] sub_mb_type[%d] = %d, want P_L0_8x8", tt.name, i, j, subType)
				}
			}
		}
	}
}

func parseHighFrameMBAFFInterFixtureSyntax(t *testing.T, data []byte, tt highFrameMBAFFP16x16NoResidualCase) ([]h264.NALUnit, [32]*h264.SPS, [256]*h264.PPS) {
	t.Helper()
	return parseHighFrameMBAFFInterFixtureSyntaxWithDeblock(t, data, tt, 0)
}

func parseHighFrameMBAFFInterFixtureSyntaxWithDeblock(t *testing.T, data []byte, tt highFrameMBAFFP16x16NoResidualCase, wantDeblockMode int32) ([]h264.NALUnit, [32]*h264.SPS, [256]*h264.PPS) {
	t.Helper()
	nals, err := h264.SplitAnnexB(data)
	if err != nil {
		t.Fatal(err)
	}

	var spsList [32]*h264.SPS
	var ppsList [256]*h264.PPS
	var gotVCL []h264.NALUnit
	var gotSliceTypes []int32
	for _, nal := range nals {
		switch nal.Type {
		case h264.NALSPS:
			sps, err := h264.DecodeSPS(nal.RBSP)
			if err != nil {
				t.Fatal(err)
			}
			if sps.ProfileIDC != 244 || sps.Width != 16 || sps.Height != 32 ||
				sps.ChromaFormatIDC != 1 || sps.BitDepthLuma != int32(tt.bitDepth) ||
				sps.BitDepthChroma != int32(tt.bitDepth) || sps.RefFrameCount != 1 ||
				sps.FrameMBSOnlyFlag != 0 || sps.MBAFF != 1 {
				t.Fatalf("SPS profile/format = %d %dx%d chroma %d depth %d/%d refs %d frame_mbs_only:%d mbaff:%d, want High 4:4:4 Predictive-compatible 16x32 yuv420p%dle ref frame-MBAFF",
					sps.ProfileIDC, sps.Width, sps.Height, sps.ChromaFormatIDC,
					sps.BitDepthLuma, sps.BitDepthChroma, sps.RefFrameCount,
					sps.FrameMBSOnlyFlag, sps.MBAFF, tt.bitDepth)
			}
			spsList[sps.SPSID] = sps
		case h264.NALPPS:
			pps, err := h264.DecodePPS(nal.RBSP, &spsList)
			if err != nil {
				t.Fatal(err)
			}
			if pps.CABAC != 0 || pps.Transform8x8Mode != 0 || pps.DeblockingFilterParametersPresent == 0 ||
				pps.RefCount[0] != 1 || pps.RefCount[1] != 1 {
				t.Fatalf("PPS CABAC/8x8/deblock-present/refs = %d/%d/%d/%d/%d, want CAVLC/no-8x8/deblock params/1/1",
					pps.CABAC, pps.Transform8x8Mode, pps.DeblockingFilterParametersPresent, pps.RefCount[0], pps.RefCount[1])
			}
			ppsList[pps.PPSID] = pps
		case h264.NALIDRSlice, h264.NALSlice:
			sh, err := h264.ParseSliceHeader(nal, &ppsList)
			if err != nil {
				t.Fatal(err)
			}
			wantSliceDeblockMode := wantDeblockMode
			if sh.SliceTypeNoS != h264.PictureTypeP {
				wantSliceDeblockMode = 0
			}
			if sh.PictureStructure != h264.PictureFrame || sh.DeblockingFilter != wantSliceDeblockMode ||
				sh.QScale != 26 || sh.SPS.MBAFF != 1 {
				t.Fatalf("slice picture/deblock/qp/mbaff = %d/%d/%d/%d, want frame/mode-%d/26/1",
					sh.PictureStructure, sh.DeblockingFilter, sh.QScale, sh.SPS.MBAFF, wantSliceDeblockMode)
			}
			if sh.SliceTypeNoS == h264.PictureTypeP && (sh.ListCount != 1 || sh.RefCount[0] != 1) {
				t.Fatalf("P slice lists/refs = %d/%v, want one L0 ref", sh.ListCount, sh.RefCount)
			}
			gotVCL = append(gotVCL, nal)
			gotSliceTypes = append(gotSliceTypes, sh.SliceTypeNoS)
		default:
			t.Fatalf("unexpected NAL type %d in %s frame-MBAFF inter fixture", nal.Type, tt.name)
		}
	}
	if len(gotVCL) != 2 || gotVCL[0].Type != h264.NALIDRSlice || gotVCL[1].Type != h264.NALSlice {
		gotTypes := make([]h264.NALUnitType, 0, len(gotVCL))
		for _, nal := range gotVCL {
			gotTypes = append(gotTypes, nal.Type)
		}
		t.Fatalf("VCL NALs = %v, want IDR slice followed by non-IDR slice", gotTypes)
	}
	if len(gotSliceTypes) != 2 || gotSliceTypes[0] != h264.PictureTypeI || gotSliceTypes[1] != h264.PictureTypeP {
		t.Fatalf("slice types = %v, want I then P", gotSliceTypes)
	}
	return gotVCL, spsList, ppsList
}

type highFrameMBAFFCAVLCP16x16Macroblock struct {
	skipRun    uint32
	mbType     uint32
	refIdxFlag uint32
	cbpCode    uint32
	cbp        uint32
}

type highFrameMBAFFCAVLCP16x16Pair struct {
	fieldFlag uint32
	top       highFrameMBAFFCAVLCP16x16Macroblock
	bottom    highFrameMBAFFCAVLCP16x16Macroblock
}

type highFrameMBAFFCAVLCPartitionedPMacroblock struct {
	skipRun     uint32
	mbType      uint32
	refIdxCount int
	refIdxFlags [4]uint32
	subMBType   [4]uint32
	cbpCode     uint32
	cbp         uint32
}

type highFrameMBAFFCAVLCPartitionedPPair struct {
	fieldFlag uint32
	top       highFrameMBAFFCAVLCPartitionedPMacroblock
	bottom    highFrameMBAFFCAVLCPartitionedPMacroblock
}

func readHighFrameMBAFFCAVLCP16x16Pair(t *testing.T, nal h264.NALUnit, sps *h264.SPS, pps *h264.PPS, residualTailBits string) highFrameMBAFFCAVLCP16x16Pair {
	t.Helper()
	return readHighFrameMBAFFCAVLCP16x16PairWithDeblock(t, nal, sps, pps, residualTailBits, 1)
}

func readHighFrameMBAFFCAVLCP16x16PairWithDeblock(t *testing.T, nal h264.NALUnit, sps *h264.SPS, pps *h264.PPS, residualTailBits string, disableDeblockingFilterIDC uint32) highFrameMBAFFCAVLCP16x16Pair {
	t.Helper()
	if sps == nil || pps == nil {
		t.Fatal("missing SPS/PPS for frame-MBAFF P macroblock syntax check")
	}
	br := high10ResidualCAVLCBitReader{data: nal.RBSP}
	firstMB := br.readUE(t)
	rawSliceType := br.readUE(t)
	sliceTypeNoS := high10ResidualCAVLCSliceTypeNoS(t, rawSliceType)
	ppsID := br.readUE(t)
	if firstMB != 0 || sliceTypeNoS != h264.PictureTypeP || ppsID != pps.PPSID {
		t.Fatalf("slice header first_mb/type/pps = %d/%d/%d, want first P slice with PPS %d", firstMB, sliceTypeNoS, ppsID, pps.PPSID)
	}
	br.readBits(t, int(sps.Log2MaxFrameNum))
	fieldPic := br.readBit(t)
	if fieldPic != 0 {
		t.Fatalf("field_pic_flag = %d, want frame picture", fieldPic)
	}
	if sps.PocType == 0 {
		br.readBits(t, int(sps.Log2MaxPocLSB))
		if pps.PicOrderPresent != 0 {
			br.readSE(t)
		}
	}
	refCount0 := pps.RefCount[0]
	if br.readBit(t) != 0 {
		refCount0 = br.readUE(t) + 1
	}
	high10ResidualCAVLCReadRefPicListModifications(t, &br, 1)
	if pps.WeightedPred != 0 {
		t.Fatal("fixture unexpectedly uses weighted P prediction")
	}
	high10ResidualCAVLCReadRefPicMarking(t, &br, nal)
	if pps.CABAC != 0 {
		br.readUE(t)
	}
	br.readSE(t)
	if pps.DeblockingFilterParametersPresent != 0 {
		disableIDC := br.readUE(t)
		if disableIDC != disableDeblockingFilterIDC {
			t.Fatalf("disable_deblocking_filter_idc = %d, want %d", disableIDC, disableDeblockingFilterIDC)
		}
		if disableIDC != 1 {
			if alphaOffset := br.readSE(t); alphaOffset != 0 {
				t.Fatalf("slice_alpha_c0_offset_div2 = %d, want 0", alphaOffset)
			}
			if betaOffset := br.readSE(t); betaOffset != 0 {
				t.Fatalf("slice_beta_offset_div2 = %d, want 0", betaOffset)
			}
		}
	}

	topSkipRun := br.readUE(t)
	fieldFlag := br.readBit(t)
	refCount0 = h264MBAFFRefCountForSyntax(refCount0, fieldFlag)
	top := readHighFrameMBAFFCAVLCP16x16Macroblock(t, &br, topSkipRun, refCount0, residualTailBits)
	bottomSkipRun := br.readUE(t)
	bottom := readHighFrameMBAFFCAVLCP16x16Macroblock(t, &br, bottomSkipRun, refCount0, residualTailBits)
	return highFrameMBAFFCAVLCP16x16Pair{
		fieldFlag: fieldFlag,
		top:       top,
		bottom:    bottom,
	}
}

func readHighFrameMBAFFPSkipRun(t *testing.T, nal h264.NALUnit, sps *h264.SPS, pps *h264.PPS, disableDeblockingFilterIDC uint32) uint32 {
	t.Helper()
	if sps == nil || pps == nil {
		t.Fatal("missing SPS/PPS for frame-MBAFF P-skip syntax check")
	}
	br := high10ResidualCAVLCBitReader{data: nal.RBSP}
	firstMB := br.readUE(t)
	rawSliceType := br.readUE(t)
	sliceTypeNoS := high10ResidualCAVLCSliceTypeNoS(t, rawSliceType)
	ppsID := br.readUE(t)
	if firstMB != 0 || sliceTypeNoS != h264.PictureTypeP || ppsID != pps.PPSID {
		t.Fatalf("slice header first_mb/type/pps = %d/%d/%d, want first P-skip slice with PPS %d", firstMB, sliceTypeNoS, ppsID, pps.PPSID)
	}
	br.readBits(t, int(sps.Log2MaxFrameNum))
	fieldPic := br.readBit(t)
	if fieldPic != 0 {
		t.Fatalf("field_pic_flag = %d, want frame picture", fieldPic)
	}
	if sps.PocType == 0 {
		br.readBits(t, int(sps.Log2MaxPocLSB))
		if pps.PicOrderPresent != 0 {
			br.readSE(t)
		}
	}
	if br.readBit(t) != 0 {
		br.readUE(t)
	}
	high10ResidualCAVLCReadRefPicListModifications(t, &br, 1)
	if pps.WeightedPred != 0 {
		t.Fatal("fixture unexpectedly uses weighted P prediction")
	}
	high10ResidualCAVLCReadRefPicMarking(t, &br, nal)
	if pps.CABAC != 0 {
		br.readUE(t)
	}
	br.readSE(t)
	if pps.DeblockingFilterParametersPresent != 0 {
		disableIDC := br.readUE(t)
		if disableIDC != disableDeblockingFilterIDC {
			t.Fatalf("disable_deblocking_filter_idc = %d, want %d", disableIDC, disableDeblockingFilterIDC)
		}
		if disableIDC != 1 {
			if alphaOffset := br.readSE(t); alphaOffset != 0 {
				t.Fatalf("slice_alpha_c0_offset_div2 = %d, want 0", alphaOffset)
			}
			if betaOffset := br.readSE(t); betaOffset != 0 {
				t.Fatalf("slice_beta_offset_div2 = %d, want 0", betaOffset)
			}
		}
	}
	return br.readUE(t)
}

func h264MBAFFRefCountForSyntax(refCount uint32, fieldFlag uint32) uint32 {
	if fieldFlag == 0 {
		return refCount
	}
	if refCount > 16 {
		return 32
	}
	return refCount * 2
}

func readHighFrameMBAFFCAVLCPartitionedPPair(t *testing.T, nal h264.NALUnit, sps *h264.SPS, pps *h264.PPS, residualTailBits string) highFrameMBAFFCAVLCPartitionedPPair {
	t.Helper()
	return readHighFrameMBAFFCAVLCPartitionedPPairWithDeblock(t, nal, sps, pps, residualTailBits, 1)
}

func readHighFrameMBAFFCAVLCPartitionedPPairWithDeblock(t *testing.T, nal h264.NALUnit, sps *h264.SPS, pps *h264.PPS, residualTailBits string, disableDeblockingFilterIDC uint32) highFrameMBAFFCAVLCPartitionedPPair {
	t.Helper()
	if sps == nil || pps == nil {
		t.Fatal("missing SPS/PPS for frame-MBAFF partitioned P macroblock syntax check")
	}
	br := high10ResidualCAVLCBitReader{data: nal.RBSP}
	firstMB := br.readUE(t)
	rawSliceType := br.readUE(t)
	sliceTypeNoS := high10ResidualCAVLCSliceTypeNoS(t, rawSliceType)
	ppsID := br.readUE(t)
	if firstMB != 0 || sliceTypeNoS != h264.PictureTypeP || ppsID != pps.PPSID {
		t.Fatalf("slice header first_mb/type/pps = %d/%d/%d, want first P slice with PPS %d", firstMB, sliceTypeNoS, ppsID, pps.PPSID)
	}
	br.readBits(t, int(sps.Log2MaxFrameNum))
	fieldPic := br.readBit(t)
	if fieldPic != 0 {
		t.Fatalf("field_pic_flag = %d, want frame picture", fieldPic)
	}
	if sps.PocType == 0 {
		br.readBits(t, int(sps.Log2MaxPocLSB))
		if pps.PicOrderPresent != 0 {
			br.readSE(t)
		}
	}
	refCount0 := pps.RefCount[0]
	if br.readBit(t) != 0 {
		refCount0 = br.readUE(t) + 1
	}
	high10ResidualCAVLCReadRefPicListModifications(t, &br, 1)
	if pps.WeightedPred != 0 {
		t.Fatal("fixture unexpectedly uses weighted P prediction")
	}
	high10ResidualCAVLCReadRefPicMarking(t, &br, nal)
	if pps.CABAC != 0 {
		br.readUE(t)
	}
	br.readSE(t)
	if pps.DeblockingFilterParametersPresent != 0 {
		disableIDC := br.readUE(t)
		if disableIDC != disableDeblockingFilterIDC {
			t.Fatalf("disable_deblocking_filter_idc = %d, want %d", disableIDC, disableDeblockingFilterIDC)
		}
		if disableIDC != 1 {
			if alphaOffset := br.readSE(t); alphaOffset != 0 {
				t.Fatalf("slice_alpha_c0_offset_div2 = %d, want 0", alphaOffset)
			}
			if betaOffset := br.readSE(t); betaOffset != 0 {
				t.Fatalf("slice_beta_offset_div2 = %d, want 0", betaOffset)
			}
		}
	}

	topSkipRun := br.readUE(t)
	fieldFlag := br.readBit(t)
	refCount0 = h264MBAFFRefCountForSyntax(refCount0, fieldFlag)
	top := readHighFrameMBAFFCAVLCPartitionedPMacroblock(t, &br, topSkipRun, refCount0, residualTailBits)
	bottomSkipRun := br.readUE(t)
	bottom := readHighFrameMBAFFCAVLCPartitionedPMacroblock(t, &br, bottomSkipRun, refCount0, residualTailBits)
	return highFrameMBAFFCAVLCPartitionedPPair{
		fieldFlag: fieldFlag,
		top:       top,
		bottom:    bottom,
	}
}

func readHighFrameMBAFFCAVLCP16x16Macroblock(t *testing.T, br *high10ResidualCAVLCBitReader, skipRun uint32, refCount0 uint32, residualTailBits string) highFrameMBAFFCAVLCP16x16Macroblock {
	t.Helper()
	mbType := br.readUE(t)
	if mbType != 0 {
		t.Fatalf("P macroblock type = %d, want P16x16", mbType)
	}
	refIdxFlag := uint32(0)
	if refCount0 > 1 {
		refIdxFlag = br.readBit(t)
	}
	br.readSE(t)
	br.readSE(t)
	cbpCode := br.readUE(t)
	if cbpCode >= uint32(len(high10ResidualCAVLCInterCBP)) {
		t.Fatalf("coded_block_pattern code = %d, want < %d", cbpCode, len(high10ResidualCAVLCInterCBP))
	}
	readHighFrameMBAFFCAVLCBits(t, br, residualTailBits)
	return highFrameMBAFFCAVLCP16x16Macroblock{
		skipRun:    skipRun,
		mbType:     mbType,
		refIdxFlag: refIdxFlag,
		cbpCode:    cbpCode,
		cbp:        uint32(high10ResidualCAVLCInterCBP[cbpCode]),
	}
}

func readHighFrameMBAFFCAVLCBits(t *testing.T, br *high10ResidualCAVLCBitReader, bits string) {
	t.Helper()
	for i, bit := range bits {
		got := br.readBit(t)
		switch bit {
		case '0':
			if got != 0 {
				t.Fatalf("residual bit[%d] = %d, want 0", i, got)
			}
		case '1':
			if got != 1 {
				t.Fatalf("residual bit[%d] = %d, want 1", i, got)
			}
		}
	}
}

func readHighFrameMBAFFCAVLCPartitionedPMacroblock(t *testing.T, br *high10ResidualCAVLCBitReader, skipRun uint32, refCount0 uint32, residualTailBits string) highFrameMBAFFCAVLCPartitionedPMacroblock {
	t.Helper()
	var mb highFrameMBAFFCAVLCPartitionedPMacroblock
	mb.skipRun = skipRun
	mb.mbType = br.readUE(t)

	mvdPairs := 0
	switch mb.mbType {
	case 1, 2:
		mb.refIdxCount = 2
		mvdPairs = 2
	case 3:
		for i := 0; i < 4; i++ {
			subType := br.readUE(t)
			mb.subMBType[i] = subType
			switch subType {
			case 0:
				mvdPairs += 1
			case 1, 2:
				mvdPairs += 2
			case 3:
				mvdPairs += 4
			default:
				t.Fatalf("P sub macroblock type[%d] = %d, want P8x8/P8x4/P4x8/P4x4 syntax", i, subType)
			}
		}
		mb.refIdxCount = 4
	default:
		t.Fatalf("P macroblock type = %d, want P16x8/P8x16/P8x8", mb.mbType)
	}

	if refCount0 > 1 {
		for i := 0; i < mb.refIdxCount; i++ {
			mb.refIdxFlags[i] = br.readBit(t)
		}
	}
	for i := 0; i < mvdPairs; i++ {
		br.readSE(t)
		br.readSE(t)
	}
	cbpCode := br.readUE(t)
	if cbpCode >= uint32(len(high10ResidualCAVLCInterCBP)) {
		t.Fatalf("coded_block_pattern code = %d, want < %d", cbpCode, len(high10ResidualCAVLCInterCBP))
	}
	readHighFrameMBAFFCAVLCBits(t, br, residualTailBits)
	mb.cbpCode = cbpCode
	mb.cbp = uint32(high10ResidualCAVLCInterCBP[cbpCode])
	return mb
}

func highFrameMBAFFPartitionedPRefIdxCount(t *testing.T, mbType uint32) int {
	t.Helper()
	switch mbType {
	case 1, 2:
		return 2
	case 3:
		return 4
	default:
		t.Fatalf("partitioned P mb_type = %d, want P16x8/P8x16/P8x8", mbType)
		return 0
	}
}

func assertHighFrameMBAFFP16x16LumaChromaResidualFrames(t *testing.T, frames []*Frame, tt highFrameMBAFFP16x16LumaChromaResidualCase) {
	t.Helper()
	assertHighFrameMBAFFP16x16NoResidualFrames(t, frames, tt)
}

func assertHighFrameMBAFFPartitionedPLumaChromaResidualFrames(t *testing.T, frames []*Frame, tt highFrameMBAFFPartitionedPLumaChromaResidualCase) {
	t.Helper()
	assertHighFrameMBAFFP16x16NoResidualFrames(t, frames, highFrameMBAFFP16x16NoResidualCase{
		name:         tt.name,
		bitDepth:     tt.bitDepth,
		bitstreamMD5: tt.bitstreamMD5,
		refFrameMD5:  tt.refFrameMD5,
		pFrameMD5:    tt.pFrameMD5,
		rawVideoMD5:  tt.rawVideoMD5,
	})
}

func assertHighFrameMBAFFFrameCodedPLumaChromaResidualFrames(t *testing.T, frames []*Frame, tt highFrameMBAFFFrameCodedPLumaChromaResidualCase) {
	t.Helper()
	assertHighFrameMBAFFP16x16NoResidualFrames(t, frames, highFrameMBAFFP16x16NoResidualCase{
		name:         tt.name,
		bitDepth:     tt.bitDepth,
		bitstreamMD5: tt.bitstreamMD5,
		refFrameMD5:  tt.refFrameMD5,
		pFrameMD5:    tt.pFrameMD5,
		rawVideoMD5:  tt.rawVideoMD5,
	})
}

func assertHighFrameMBAFFPartitionedPSparseResidualFrames(t *testing.T, frames []*Frame, tt highFrameMBAFFPartitionedPSparseResidualCase) {
	t.Helper()
	assertHighFrameMBAFFP16x16NoResidualFrames(t, frames, highFrameMBAFFP16x16NoResidualCase{
		name:         tt.name,
		bitDepth:     tt.bitDepth,
		bitstreamMD5: tt.bitstreamMD5,
		refFrameMD5:  tt.refFrameMD5,
		pFrameMD5:    tt.pFrameMD5,
		rawVideoMD5:  tt.rawVideoMD5,
	})
}

func assertHighFrameMBAFFPartitionedPDeblockFrames(t *testing.T, frames []*Frame, tt highFrameMBAFFPartitionedPDeblockCase) {
	t.Helper()
	assertHighFrameMBAFFP16x16NoResidualFrames(t, frames, highFrameMBAFFP16x16NoResidualCase{
		name:         tt.name,
		bitDepth:     tt.bitDepth,
		bitstreamMD5: tt.bitstreamMD5,
		refFrameMD5:  tt.refFrameMD5,
		pFrameMD5:    tt.pFrameMD5,
		rawVideoMD5:  tt.rawVideoMD5,
	})
}

func assertHighFrameMBAFFP16x16DeblockFrames(t *testing.T, frames []*Frame, tt highFrameMBAFFP16x16DeblockCase) {
	t.Helper()
	assertHighFrameMBAFFP16x16NoResidualFrames(t, frames, highFrameMBAFFP16x16NoResidualCase{
		name:         tt.name,
		bitDepth:     tt.bitDepth,
		bitstreamMD5: tt.bitstreamMD5,
		refFrameMD5:  tt.refFrameMD5,
		pFrameMD5:    tt.pFrameMD5,
		rawVideoMD5:  tt.rawVideoMD5,
	})
}

func assertHighFrameMBAFFFrameP16x16NoResidualFrames(t *testing.T, frames []*Frame, tt highFrameMBAFFFrameP16x16NoResidualCase) {
	t.Helper()
	assertHighFrameMBAFFP16x16NoResidualFrames(t, frames, tt)
}

func assertHighFrameMBAFFFrameP16x16LumaResidualFrames(t *testing.T, frames []*Frame, tt highFrameMBAFFFrameP16x16LumaResidualCase) {
	t.Helper()
	assertHighFrameMBAFFP16x16NoResidualFrames(t, frames, tt)
}

func assertHighFrameMBAFFP16x16LumaResidualFrames(t *testing.T, frames []*Frame, tt highFrameMBAFFP16x16LumaResidualCase) {
	t.Helper()
	assertHighFrameMBAFFP16x16NoResidualFrames(t, frames, tt)
}

func assertHighFrameMBAFFP16x16NoResidualFrames(t *testing.T, frames []*Frame, tt highFrameMBAFFP16x16NoResidualCase) {
	t.Helper()
	if len(frames) != 2 {
		t.Fatalf("frames = %d, want 2", len(frames))
	}
	for i, frame := range frames {
		if frame.Width != 16 || frame.Height != 32 ||
			frame.ChromaFormatIDC != 1 || frame.BitDepthLuma != tt.bitDepth ||
			frame.BitDepthChroma != tt.bitDepth {
			t.Fatalf("frame[%d] format = %dx%d chroma %d depth %d/%d, want 16x32 yuv420p%dle",
				i, frame.Width, frame.Height, frame.ChromaFormatIDC, frame.BitDepthLuma, frame.BitDepthChroma,
				tt.bitDepth)
		}
		if pixFmt, err := frame.RawPixelFormat(); err != nil || pixFmt != fmt.Sprintf("yuv420p%dle", tt.bitDepth) {
			t.Fatalf("frame[%d] RawPixelFormat = %q/%v, want yuv420p%dle/nil", i, pixFmt, err, tt.bitDepth)
		}
		if size, err := frame.RawYUVSize(); err != nil || size != 1536 {
			t.Fatalf("frame[%d] RawYUVSize = %d/%v, want 1536/nil", i, size, err)
		}
		raw, err := frame.AppendRawYUVBytesLE(nil)
		if err != nil {
			t.Fatalf("frame[%d] AppendRawYUVBytesLE: %v", i, err)
		}
		want := tt.refFrameMD5
		if i == 1 {
			want = tt.pFrameMD5
		}
		sum := md5.Sum(raw)
		if got := hex.EncodeToString(sum[:]); got != want {
			t.Fatalf("frame[%d] raw md5 = %s, want %s", i, got, want)
		}
		if _, err := frame.AppendRawYUV(nil); err != ErrUnsupported {
			t.Fatalf("frame[%d] AppendRawYUV high%d error = %v, want ErrUnsupported", i, tt.bitDepth, err)
		}
	}
}
