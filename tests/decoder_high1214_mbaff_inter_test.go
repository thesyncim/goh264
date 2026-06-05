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
			if sh.PictureStructure != h264.PictureFrame || sh.DeblockingFilter != 0 ||
				sh.QScale != 26 || sh.SPS.MBAFF != 1 {
				t.Fatalf("slice picture/deblock/qp/mbaff = %d/%d/%d/%d, want frame/disabled/26/1",
					sh.PictureStructure, sh.DeblockingFilter, sh.QScale, sh.SPS.MBAFF)
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
		if disableIDC != 1 {
			t.Fatalf("disable_deblocking_filter_idc = %d, want 1", disableIDC)
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
		if disableIDC != 1 {
			t.Fatalf("disable_deblocking_filter_idc = %d, want 1", disableIDC)
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
