// SPDX-License-Identifier: LGPL-2.1-or-later

package goh264_test

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/thesyncim/goh264/internal/h264"
)

type highCABACChromaWeightedPCase struct {
	name           string
	bitDepth       int
	sourceFile     string
	chromaFormat   uint32
	chromaWeighted bool
	deblockMode    int32
	mode2Deblock   bool
	pixFmt         string
	frameSize      int
	bitstreamMD5   string
	rawVideoMD5    string
	frameMD5       []string
}

func TestHighCABACChromaWeightedPFixtureSyntax(t *testing.T) {
	for _, tt := range highCABACChromaWeightedPCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highCABACChromaWeightedPFixture(t, tt)
			assertHighCABACChromaWeightedPFixtureSyntax(t, data, tt)
		})
	}
}

func TestDecodeAnnexBHighCABACChromaWeightedPFrames(t *testing.T) {
	for _, tt := range highCABACChromaWeightedPCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highCABACChromaWeightedPFixture(t, tt)
			assertHighCABACChromaWeightedPFixtureSyntax(t, data, tt)
			frames, err := NewDecoder().DecodeAnnexBFrames(data)
			if err != nil {
				t.Fatalf("DecodeAnnexBFrames: %v", err)
			}
			assertHighCABACChromaWeightedPFrames(t, frames, tt)
		})
	}
}

func TestDecodeAVCHighCABACChromaWeightedPFrames(t *testing.T) {
	for _, tt := range highCABACChromaWeightedPCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highCABACChromaWeightedPFixture(t, tt)
			assertHighCABACChromaWeightedPFixtureSyntax(t, data, tt)
			for _, nalLengthSize := range []int{2, 3, 4} {
				frames, err := NewDecoder().DecodeAVCFrames(annexBToAVC(t, data, nalLengthSize), nalLengthSize)
				if err != nil {
					t.Fatalf("nalLengthSize=%d: DecodeAVCFrames: %v", nalLengthSize, err)
				}
				assertHighCABACChromaWeightedPFrames(t, frames, tt)
			}
		})
	}
}

func TestDecodeConfiguredAVCHighCABACChromaWeightedPFramesAcrossSamples(t *testing.T) {
	for _, tt := range highCABACChromaWeightedPCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highCABACChromaWeightedPFixture(t, tt)
			assertHighCABACChromaWeightedPFixtureSyntax(t, data, tt)
			for _, nalLengthSize := range []int{2, 3, 4} {
				config, samples := annexBToAVCConfigAndSamples(t, data, nalLengthSize)
				if len(samples) != len(tt.frameMD5) {
					t.Fatalf("nalLengthSize=%d samples = %d, want %d", nalLengthSize, len(samples), len(tt.frameMD5))
				}
				dec := NewDecoder()
				if _, err := dec.ConfigureAVCDecoderConfigurationRecord(config); err != nil {
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
				assertHighCABACChromaWeightedPFrames(t, frames, tt)
			}
		})
	}
}

func TestFFmpegRawVideoFrameMD5OracleHighCABACChromaWeightedP(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run native ffmpeg oracle")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}
	for _, tt := range highCABACChromaWeightedPCases() {
		t.Run(tt.name, func(t *testing.T) {
			data := highCABACChromaWeightedPFixture(t, tt)
			assertHighCABACChromaWeightedPFixtureSyntax(t, data, tt)
			assertFFmpegHighCABACChromaWeightedPRawVideoOracle(t, data, tt)
		})
	}
}

func highCABACChromaWeightedPCases() []highCABACChromaWeightedPCase {
	var cases []highCABACChromaWeightedPCase
	expected := highCABACChromaWeightedPExpected()
	for _, bitDepth := range []int{12, 14} {
		suffix := "12"
		if bitDepth == 14 {
			suffix = "14"
		}
		for _, tt := range []struct {
			name           string
			sourceFile     string
			chromaFormat   uint32
			chromaWeighted bool
			deblockMode    int32
			mode2Deblock   bool
			pixFmt         string
			frameSize      int
		}{
			{name: "422-luma-chroma-no-deblock", sourceFile: "high10_weighted422_cabac_p.h264", chromaFormat: 2, chromaWeighted: true, pixFmt: "yuv422p" + suffix + "le", frameSize: 16384},
			{name: "422-luma-chroma-frame-deblock", sourceFile: "high10_weighted422_deblock_cabac_p.h264", chromaFormat: 2, chromaWeighted: true, deblockMode: 1, pixFmt: "yuv422p" + suffix + "le", frameSize: 16384},
			{name: "422-luma-chroma-slice-boundary", sourceFile: "high10_weighted422_deblock_cabac_p.h264", chromaFormat: 2, chromaWeighted: true, deblockMode: 2, mode2Deblock: true, pixFmt: "yuv422p" + suffix + "le", frameSize: 16384},
			{name: "444-luma-chroma-no-deblock", sourceFile: "high10_weighted444_cabac_p.h264", chromaFormat: 3, chromaWeighted: true, pixFmt: "yuv444p" + suffix + "le", frameSize: 24576},
			{name: "444-luma-chroma-frame-deblock", sourceFile: "high10_weighted444_deblock_cabac_p.h264", chromaFormat: 3, chromaWeighted: true, deblockMode: 1, pixFmt: "yuv444p" + suffix + "le", frameSize: 24576},
			{name: "444-luma-chroma-slice-boundary", sourceFile: "high10_weighted444_deblock_cabac_p.h264", chromaFormat: 3, chromaWeighted: true, deblockMode: 2, mode2Deblock: true, pixFmt: "yuv444p" + suffix + "le", frameSize: 24576},
			{name: "422-luma-only-no-deblock", sourceFile: "high10_luma_weighted422_cabac_p.h264", chromaFormat: 2, pixFmt: "yuv422p" + suffix + "le", frameSize: 16384},
			{name: "422-luma-only-frame-deblock", sourceFile: "high10_luma_weighted422_deblock_cabac_p.h264", chromaFormat: 2, deblockMode: 1, pixFmt: "yuv422p" + suffix + "le", frameSize: 16384},
			{name: "422-luma-only-slice-boundary", sourceFile: "high10_luma_weighted422_deblock_cabac_p.h264", chromaFormat: 2, deblockMode: 2, mode2Deblock: true, pixFmt: "yuv422p" + suffix + "le", frameSize: 16384},
			{name: "444-luma-only-no-deblock", sourceFile: "high10_luma_weighted444_cabac_p.h264", chromaFormat: 3, pixFmt: "yuv444p" + suffix + "le", frameSize: 24576},
			{name: "444-luma-only-frame-deblock", sourceFile: "high10_luma_weighted444_deblock_cabac_p.h264", chromaFormat: 3, deblockMode: 1, pixFmt: "yuv444p" + suffix + "le", frameSize: 24576},
			{name: "444-luma-only-slice-boundary", sourceFile: "high10_luma_weighted444_deblock_cabac_p.h264", chromaFormat: 3, deblockMode: 2, mode2Deblock: true, pixFmt: "yuv444p" + suffix + "le", frameSize: 24576},
		} {
			name := fmt.Sprintf("high%d-%s", bitDepth, tt.name)
			want, ok := expected[name]
			if !ok {
				panic("missing High CABAC chroma weighted-P expected hashes for " + name)
			}
			cases = append(cases, highCABACChromaWeightedPCase{
				name:           name,
				bitDepth:       bitDepth,
				sourceFile:     tt.sourceFile,
				chromaFormat:   tt.chromaFormat,
				chromaWeighted: tt.chromaWeighted,
				deblockMode:    tt.deblockMode,
				mode2Deblock:   tt.mode2Deblock,
				pixFmt:         tt.pixFmt,
				frameSize:      tt.frameSize,
				bitstreamMD5:   want.bitstreamMD5,
				rawVideoMD5:    want.rawVideoMD5,
				frameMD5:       want.frameMD5,
			})
		}
	}
	return cases
}

func highCABACChromaWeightedPExpected() map[string]struct {
	bitstreamMD5 string
	rawVideoMD5  string
	frameMD5     []string
} {
	return map[string]struct {
		bitstreamMD5 string
		rawVideoMD5  string
		frameMD5     []string
	}{
		"high12-422-luma-chroma-no-deblock": {
			bitstreamMD5: "bb212cf17644731abf0b90c714bf78ac",
			rawVideoMD5:  "8080d0a8b03c0306320f450be0f4038e",
			frameMD5:     []string{"df48611341a11949b5553ca8ae106246", "cc29044648fc89a6e78f4a316f58cf55", "7f3081d86a01f4faf8c4d2be58a45b1f", "1cea7d152f655b4738db8b91d8f38a59", "715def0d896ce1d534fa53b36298a4d7"},
		},
		"high12-422-luma-chroma-frame-deblock": {
			bitstreamMD5: "e2dd68dec0b3aac66096056545be2775",
			rawVideoMD5:  "d624a0110172cef54bb6dd0901980481",
			frameMD5:     []string{"df48611341a11949b5553ca8ae106246", "5fc2bb4eac252c910686aeec269f8569", "6f125c7c28a63cb8f48575adcd669c4f", "a41e62096ded87ee94fe10ac60f240cf", "eac452e310faa5b37b910efc5841c6cd"},
		},
		"high12-422-luma-chroma-slice-boundary": {
			bitstreamMD5: "d6699f30aacf3033ac4396fdf6355c20",
			rawVideoMD5:  "d624a0110172cef54bb6dd0901980481",
			frameMD5:     []string{"df48611341a11949b5553ca8ae106246", "5fc2bb4eac252c910686aeec269f8569", "6f125c7c28a63cb8f48575adcd669c4f", "a41e62096ded87ee94fe10ac60f240cf", "eac452e310faa5b37b910efc5841c6cd"},
		},
		"high12-444-luma-chroma-no-deblock": {
			bitstreamMD5: "055a26f91f8ad0e654ec74e54d5826b5",
			rawVideoMD5:  "4243d46a46c273ecece4f069f8d82f97",
			frameMD5:     []string{"07b0d1f72ca5c3016a760b7983432546", "ba73af7fd2826f04930de6669edb63d2", "6df82d2aab40616f144c4036c0b611d2", "322f4d9cd74c7dda6f18367593c18a72", "8ac0eb9a1e54cf6f22916afb2080bf55"},
		},
		"high12-444-luma-chroma-frame-deblock": {
			bitstreamMD5: "191238b644b4691a41a5aab43a65625d",
			rawVideoMD5:  "34f6fa69194f488a259fcb04f5f63d7f",
			frameMD5:     []string{"07b0d1f72ca5c3016a760b7983432546", "ee160766e3b6a45f9836bd35d766eb4f", "ddb7b2b5a7ff313b80f007105c2474ea", "4b9c2ced43bf2065f2f4e08c88d912b5", "000915d82726f1e3fbcf4740497bde3e"},
		},
		"high12-444-luma-chroma-slice-boundary": {
			bitstreamMD5: "faacf69f665bdd0699ccb035d7580244",
			rawVideoMD5:  "34f6fa69194f488a259fcb04f5f63d7f",
			frameMD5:     []string{"07b0d1f72ca5c3016a760b7983432546", "ee160766e3b6a45f9836bd35d766eb4f", "ddb7b2b5a7ff313b80f007105c2474ea", "4b9c2ced43bf2065f2f4e08c88d912b5", "000915d82726f1e3fbcf4740497bde3e"},
		},
		"high12-422-luma-only-no-deblock": {
			bitstreamMD5: "0b38eb9b841749e5369743296fb83b16",
			rawVideoMD5:  "b5417cef447379cc7b17a20ebf071338",
			frameMD5:     []string{"df48611341a11949b5553ca8ae106246", "0b4f66065fbbbabd52bef3374c9c8eb6", "8db716850a44d6ed4ddb11ac06c4d59c", "5d34f44c36979f57e504d8aafcd0a5d0", "5f3dd4b50fac78540f13b78af2095c55"},
		},
		"high12-422-luma-only-frame-deblock": {
			bitstreamMD5: "093cdc8c856e72576d83126616029408",
			rawVideoMD5:  "7b24bff7aae3a1ac7f87f4ec716b2136",
			frameMD5:     []string{"df48611341a11949b5553ca8ae106246", "ebdd0ebcb20e49594c30e05400909325", "1f9bed5ab9f504ca5066495d5a52ba19", "7d70858f09ed7aa7949d542b998700f7", "bf5211a7ac7350f04d9070a9b0deeb09"},
		},
		"high12-422-luma-only-slice-boundary": {
			bitstreamMD5: "246022b113ccd73f1b6d5d2730da8a0d",
			rawVideoMD5:  "7b24bff7aae3a1ac7f87f4ec716b2136",
			frameMD5:     []string{"df48611341a11949b5553ca8ae106246", "ebdd0ebcb20e49594c30e05400909325", "1f9bed5ab9f504ca5066495d5a52ba19", "7d70858f09ed7aa7949d542b998700f7", "bf5211a7ac7350f04d9070a9b0deeb09"},
		},
		"high12-444-luma-only-no-deblock": {
			bitstreamMD5: "31b1d277cb93b0aba552dbb0ac2fd5ee",
			rawVideoMD5:  "4a87780fa7601583e1947e0da4222bec",
			frameMD5:     []string{"07b0d1f72ca5c3016a760b7983432546", "82703b8bb763de4ef85ddbcf0cd05866", "cf3bee5bb6d5e2e7bb44a2f324264bd4", "a38c052093627e2370f5916f7f090cdf", "822c83b7c0088a186bfcb0a276c6fefd"},
		},
		"high12-444-luma-only-frame-deblock": {
			bitstreamMD5: "52d3c65bbba72d2d7b4ca574e56d2299",
			rawVideoMD5:  "26ff4a393ece064d3f751a8e6aead5a0",
			frameMD5:     []string{"07b0d1f72ca5c3016a760b7983432546", "299a8c1d5ed282ad21579538bcf86c89", "39487b585db229239947fcd51ffd62c0", "ae05f646b8a2b7db87fe1b87fec0ef2e", "6842e5ad06edc58f996acada9aedcabc"},
		},
		"high12-444-luma-only-slice-boundary": {
			bitstreamMD5: "93cacde4b821de0315dbe6e3bb3c97cd",
			rawVideoMD5:  "26ff4a393ece064d3f751a8e6aead5a0",
			frameMD5:     []string{"07b0d1f72ca5c3016a760b7983432546", "299a8c1d5ed282ad21579538bcf86c89", "39487b585db229239947fcd51ffd62c0", "ae05f646b8a2b7db87fe1b87fec0ef2e", "6842e5ad06edc58f996acada9aedcabc"},
		},
		"high14-422-luma-chroma-no-deblock": {
			bitstreamMD5: "68853e3c213e5684532852d3ad6009e4",
			rawVideoMD5:  "216d9a2e0e75ca1112fa35ac515bdff9",
			frameMD5:     []string{"b49a1d54998f9c7b3b8800df815d200d", "ae0f4ff0fa735a18407756e33da771f8", "950d5aed3d175b28420302b73c51959e", "b34654bf524bbf194581c736096af1ad", "4b0fe46ccf1526a5bc296ea1905788e0"},
		},
		"high14-422-luma-chroma-frame-deblock": {
			bitstreamMD5: "fc0d81333e69ee7099b5f32721d101a6",
			rawVideoMD5:  "1ed40e15582930386e36afbf891e95f5",
			frameMD5:     []string{"b49a1d54998f9c7b3b8800df815d200d", "e0fda66deea40ef3cc0053e71531a0a5", "9d22897b5c7d412b4e133794cb16e8b9", "ba7a4867295e468c318672fc4ce67393", "96c35700e310d12afaf9c2b5d3e7d6b5"},
		},
		"high14-422-luma-chroma-slice-boundary": {
			bitstreamMD5: "ab31cb38b4fe43ed86b78828f16b85b6",
			rawVideoMD5:  "1ed40e15582930386e36afbf891e95f5",
			frameMD5:     []string{"b49a1d54998f9c7b3b8800df815d200d", "e0fda66deea40ef3cc0053e71531a0a5", "9d22897b5c7d412b4e133794cb16e8b9", "ba7a4867295e468c318672fc4ce67393", "96c35700e310d12afaf9c2b5d3e7d6b5"},
		},
		"high14-444-luma-chroma-no-deblock": {
			bitstreamMD5: "cc2f69d6477818b146606ec2f4f27f1d",
			rawVideoMD5:  "03a7292256c4b7c50fc2c5e1c9093100",
			frameMD5:     []string{"bd23a22aa4808303967d5c085c331045", "eda458bc1f47d3fad31a273e1cd7d43d", "f95916e2cafdbf5825c1cc3906139651", "2b9ae1ea7dd3f73c6e4349fe91492806", "67d97dff9afe84062a629943f79ab37c"},
		},
		"high14-444-luma-chroma-frame-deblock": {
			bitstreamMD5: "4b7bc8eed5a0375f750f685ca29c7019",
			rawVideoMD5:  "066d0f271fcc52190ae7c9e8c08aff33",
			frameMD5:     []string{"bd23a22aa4808303967d5c085c331045", "18ad071e3423250f61ebdb4f17db7984", "93d138398dc5b5f08eab68d391bd9452", "0c5d8b9f13189c3cc50ce7e8437fcf5a", "45f7e9dd0ecec9ae7da541ab65cfd044"},
		},
		"high14-444-luma-chroma-slice-boundary": {
			bitstreamMD5: "febc94d12aef265b335d7c2d83f727e4",
			rawVideoMD5:  "066d0f271fcc52190ae7c9e8c08aff33",
			frameMD5:     []string{"bd23a22aa4808303967d5c085c331045", "18ad071e3423250f61ebdb4f17db7984", "93d138398dc5b5f08eab68d391bd9452", "0c5d8b9f13189c3cc50ce7e8437fcf5a", "45f7e9dd0ecec9ae7da541ab65cfd044"},
		},
		"high14-422-luma-only-no-deblock": {
			bitstreamMD5: "ef6f3b8a5f39bc7a5735084455bd0321",
			rawVideoMD5:  "24773b921571387b4048c195c37d61db",
			frameMD5:     []string{"b49a1d54998f9c7b3b8800df815d200d", "441886dcc9cd545ca6ccc23f9b78794a", "a0c2228a9303d8fbf3d87002bc6c85c1", "172f540d07cfb2a589e439e38a1a825d", "0a0c491321f54a94d9cfc09ab169736b"},
		},
		"high14-422-luma-only-frame-deblock": {
			bitstreamMD5: "536f801442742a65960b6b3dc40d71be",
			rawVideoMD5:  "21097430807af9bfeaab1588d515c0c1",
			frameMD5:     []string{"b49a1d54998f9c7b3b8800df815d200d", "bb18a94e8499e4bf6ef2ee4e5fc98d58", "2048a472043b4afdf3ddbea2e4309987", "18c8a07fac51e25dfb88abb1d00a2d56", "6295e88916595a9fcfca2fc451988bf1"},
		},
		"high14-422-luma-only-slice-boundary": {
			bitstreamMD5: "7dec6b1cb35df2beb62f5d29d19fc4a9",
			rawVideoMD5:  "21097430807af9bfeaab1588d515c0c1",
			frameMD5:     []string{"b49a1d54998f9c7b3b8800df815d200d", "bb18a94e8499e4bf6ef2ee4e5fc98d58", "2048a472043b4afdf3ddbea2e4309987", "18c8a07fac51e25dfb88abb1d00a2d56", "6295e88916595a9fcfca2fc451988bf1"},
		},
		"high14-444-luma-only-no-deblock": {
			bitstreamMD5: "fb0359eddffeb97eeb88a715d818b1ee",
			rawVideoMD5:  "180d156af94f820638a2015e7d1c2275",
			frameMD5:     []string{"bd23a22aa4808303967d5c085c331045", "5ced50617487f191a62b6c2f8e584a58", "3d73ab6b2bf3d712cb9b1eec67c6e3a5", "fbec07baaa08e017a663fbdc7932ad9d", "672c31617f7272b633bb7012b173a4bf"},
		},
		"high14-444-luma-only-frame-deblock": {
			bitstreamMD5: "50591ab311f877ccc9057335f401906e",
			rawVideoMD5:  "4f0c7a5ce14927b329b62e2b2c54d0a8",
			frameMD5:     []string{"bd23a22aa4808303967d5c085c331045", "ad6ca01afd666562236e7cd30cf6e518", "9cfaaf2704ce08c171d478050d2063b4", "14b099e7cbefc6b026147fcb463bd642", "c6fe40a05e8e95ee0187fdff23daca2c"},
		},
		"high14-444-luma-only-slice-boundary": {
			bitstreamMD5: "b2296e61b4eeedec14839321ffd737ff",
			rawVideoMD5:  "4f0c7a5ce14927b329b62e2b2c54d0a8",
			frameMD5:     []string{"bd23a22aa4808303967d5c085c331045", "ad6ca01afd666562236e7cd30cf6e518", "9cfaaf2704ce08c171d478050d2063b4", "14b099e7cbefc6b026147fcb463bd642", "c6fe40a05e8e95ee0187fdff23daca2c"},
		},
	}
}

func highCABACChromaWeightedPFixture(t *testing.T, tt highCABACChromaWeightedPCase) []byte {
	t.Helper()
	data := highCABACChromaWeightedPRawFixture(t, tt)
	if tt.bitstreamMD5 == "" {
		t.Fatalf("%s missing bitstream MD5", tt.name)
	}
	sum := md5.Sum(data)
	if got := hex.EncodeToString(sum[:]); got != tt.bitstreamMD5 {
		t.Fatalf("%s bitstream md5 = %s, want %s", tt.name, got, tt.bitstreamMD5)
	}
	return data
}

func highCABACChromaWeightedPRawFixture(t *testing.T, tt highCABACChromaWeightedPCase) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "h264", tt.sourceFile))
	if err != nil {
		t.Fatalf("read %s: %v", tt.sourceFile, err)
	}
	return highCABACChromaWeightedPRewriteAnnexB(t, data, tt.bitDepth, tt.mode2Deblock)
}

func highCABACChromaWeightedPRewriteAnnexB(t *testing.T, data []byte, bitDepth int, mode2Deblock bool) []byte {
	t.Helper()
	start, prefixLen, ok := high14CABACBFindStartCode(data, 0)
	if !ok {
		t.Fatal("source fixture has no Annex B start code")
	}
	var out []byte
	var spsList [32]*h264.SPS
	var ppsList [256]*h264.PPS
	for ok {
		nalStart := start + prefixLen
		nextStart, nextPrefixLen, nextOK := high14CABACBFindStartCode(data, nalStart)
		nalEnd := len(data)
		if nextOK {
			nalEnd = nextStart
		}
		if nalEnd > nalStart {
			out = append(out, data[start:nalStart]...)
			raw := append([]byte(nil), data[nalStart:nalEnd]...)
			nalType := h264.NALUnitType(raw[0] & 0x1f)
			rbsp := high14CABACBEBSPToRBSP(raw[1:])
			switch nalType {
			case h264.NALSPS:
				sps, err := h264.DecodeSPS(rbsp)
				if err != nil {
					t.Fatalf("decode source SPS: %v", err)
				}
				spsList[sps.SPSID] = sps
				raw = highCABACChromaWeightedPRewriteSPSRaw(t, raw, bitDepth)
			case h264.NALPPS:
				pps, err := h264.DecodePPS(rbsp, &spsList)
				if err != nil {
					t.Fatalf("decode source PPS: %v", err)
				}
				ppsList[pps.PPSID] = pps
			case h264.NALSlice, h264.NALIDRSlice:
				nal := h264.NALUnit{RefIDC: raw[0] >> 5 & 0x03, Type: nalType, Raw: raw, RBSP: rbsp}
				sh, err := h264.ParseSliceHeader(nal, &ppsList)
				if err != nil {
					t.Fatalf("parse source slice: %v", err)
				}
				if mode2Deblock && nalType == h264.NALSlice && sh.SliceTypeNoS == h264.PictureTypeP && sh.PPS.CABAC != 0 && sh.DeblockingFilter == 1 {
					raw = highCABACBRewriteSliceDeblockMode(t, raw, sh, 2)
					rbsp = high14CABACBEBSPToRBSP(raw[1:])
					nal.RBSP = rbsp
					nal.Raw = raw
					sh, err = h264.ParseSliceHeader(nal, &ppsList)
					if err != nil {
						t.Fatalf("parse rewritten slice: %v", err)
					}
					if sh.DeblockingFilter != 2 {
						t.Fatalf("rewritten slice deblock = %d, want mode-2", sh.DeblockingFilter)
					}
				}
			}
			out = append(out, raw...)
		}
		if !nextOK {
			break
		}
		start, prefixLen, ok = nextStart, nextPrefixLen, true
	}
	return out
}

func highCABACChromaWeightedPRewriteSPSRaw(t *testing.T, raw []byte, bitDepth int) []byte {
	t.Helper()
	rbsp := high14CABACBEBSPToRBSP(raw[1:])
	bits := high14CABACBBits(rbsp)
	stop := strings.LastIndexByte(bits, '1')
	if stop < 0 {
		t.Fatal("SPS has no rbsp stop bit")
	}
	syntax := bits[:stop]
	pos := 0
	profileIDC, _ := high14CABACBReadFixedBits(t, syntax, &pos, 8)
	constraints, _ := high14CABACBReadFixedBits(t, syntax, &pos, 8)
	level, _ := high14CABACBReadFixedBits(t, syntax, &pos, 8)
	_, spsIDBits := high14CABACBReadUEBits(t, syntax, &pos)
	chroma, chromaBits := high14CABACBReadUEBits(t, syntax, &pos)
	separateColourPlaneBits := ""
	if chroma == 3 {
		_, separateColourPlaneBits = high14CABACBReadFixedBits(t, syntax, &pos, 1)
	}
	bitDepthLumaMinus8, _ := high14CABACBReadUEBits(t, syntax, &pos)
	bitDepthChromaMinus8, _ := high14CABACBReadUEBits(t, syntax, &pos)
	if (profileIDC != 122 && profileIDC != 244) || (chroma != 2 && chroma != 3) || bitDepthLumaMinus8 != 2 || bitDepthChromaMinus8 != 2 {
		t.Fatalf("source SPS profile/chroma/depth-minus8 = %d/%d/%d/%d, want High10 4:2:2/4:4:4 2/2",
			profileIDC, chroma, bitDepthLumaMinus8, bitDepthChromaMinus8)
	}
	if bitDepth != 12 && bitDepth != 14 {
		t.Fatalf("unsupported rewritten bit depth %d", bitDepth)
	}

	bitDepthMinus8 := uint32(bitDepth - 8)
	payload := fmt.Sprintf("%08b%08b%08b", 244, constraints, level) +
		spsIDBits +
		chromaBits +
		separateColourPlaneBits +
		high14CABACBUEBits(bitDepthMinus8) +
		high14CABACBUEBits(bitDepthMinus8) +
		syntax[pos:]
	rbsp = high14CABACBPackRBSP(payload)
	return append([]byte{raw[0]}, high14CABACBRBSPToEBSP(rbsp)...)
}

func assertHighCABACChromaWeightedPFixtureSyntax(t *testing.T, data []byte, tt highCABACChromaWeightedPCase) {
	t.Helper()
	nals, err := h264.SplitAnnexB(data)
	if err != nil {
		t.Fatal(err)
	}
	var spsList [32]*h264.SPS
	var ppsList [256]*h264.PPS
	var pSlices int
	var lumaWeightedPSlices int
	var chromaWeightedPSlices int
	var gotVCL []h264.NALUnitType
	for _, nal := range nals {
		switch nal.Type {
		case h264.NALSPS:
			sps, err := h264.DecodeSPS(nal.RBSP)
			if err != nil {
				t.Fatal(err)
			}
			if sps.ProfileIDC != 244 || sps.Width != 64 || sps.Height != 64 ||
				sps.ChromaFormatIDC != tt.chromaFormat || sps.BitDepthLuma != int32(tt.bitDepth) || sps.BitDepthChroma != int32(tt.bitDepth) ||
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
			if pps.CABAC != 1 || pps.Transform8x8Mode != 0 || pps.WeightedPred != 1 ||
				pps.WeightedBipredIDC != 0 || pps.RefCount[0] != 1 || pps.RefCount[1] != 1 ||
				pps.DeblockingFilterParametersPresent == 0 {
				t.Fatalf("PPS cabac/8x8/weights/refs/deblock-present = %d/%d/%d/%d/%d/%d/%d, want CABAC/no-8x8 weighted P ref=1",
					pps.CABAC, pps.Transform8x8Mode, pps.WeightedPred, pps.WeightedBipredIDC,
					pps.RefCount[0], pps.RefCount[1], pps.DeblockingFilterParametersPresent)
			}
			ppsList[pps.PPSID] = pps
		case h264.NALSEI:
		case h264.NALIDRSlice, h264.NALSlice:
			sh, err := h264.ParseSliceHeader(nal, &ppsList)
			if err != nil {
				t.Fatal(err)
			}
			gotVCL = append(gotVCL, nal.Type)
			wantDeblock := tt.deblockMode
			if tt.mode2Deblock && nal.Type == h264.NALIDRSlice {
				wantDeblock = 1
			}
			if sh.PictureStructure != h264.PictureFrame || sh.DeblockingFilter != wantDeblock {
				t.Fatalf("slice picture/deblock = %d/%d, want frame/mode%d", sh.PictureStructure, sh.DeblockingFilter, wantDeblock)
			}
			if sh.SliceTypeNoS == h264.PictureTypeP {
				if sh.ListCount != 1 || sh.RefCount[0] != 1 || sh.PPS == nil || sh.PPS.WeightedPred != 1 {
					t.Fatalf("P slice lists/ref/weighted-p = %d/%d/%v, want one L0 ref with weighted-P PPS",
						sh.ListCount, sh.RefCount[0], sh.PPS)
				}
				pSlices++
				if sh.PredWeightTable.UseWeight != 0 {
					lumaWeightedPSlices++
				}
				if sh.PredWeightTable.UseWeightChroma != 0 {
					chromaWeightedPSlices++
				}
			}
		default:
			t.Fatalf("unexpected NAL type %d in %s", nal.Type, tt.name)
		}
	}
	if len(gotVCL) != 5 || gotVCL[0] != h264.NALIDRSlice {
		t.Fatalf("VCL NALs = %v, want IDR plus four P slices", gotVCL)
	}
	if pSlices != 4 {
		t.Fatalf("P slices = %d, want 4", pSlices)
	}
	if lumaWeightedPSlices == 0 {
		t.Fatal("weighted-P fixture has no luma-weighted P slices")
	}
	if tt.chromaWeighted && chromaWeightedPSlices == 0 {
		t.Fatal("weighted-P fixture has no chroma-weighted P slices")
	}
	if !tt.chromaWeighted && chromaWeightedPSlices != 0 {
		t.Fatalf("luma-only weighted-P fixture has %d chroma-weighted P slices, want 0", chromaWeightedPSlices)
	}
}

func assertHighCABACChromaWeightedPFrames(t *testing.T, frames []*Frame, tt highCABACChromaWeightedPCase) {
	t.Helper()
	if len(tt.frameMD5) == 0 || tt.frameMD5[0] == "" {
		t.Fatalf("%s missing frame MD5s", tt.name)
	}
	if len(frames) != len(tt.frameMD5) {
		t.Fatalf("frames = %d, want %d", len(frames), len(tt.frameMD5))
	}
	rawVideo := make([]byte, 0, len(frames)*tt.frameSize)
	for i, frame := range frames {
		if frame == nil {
			t.Fatalf("frame[%d] is nil", i)
		}
		if frame.Width != 64 || frame.Height != 64 ||
			frame.ChromaFormatIDC != tt.chromaFormat ||
			frame.BitDepthLuma != tt.bitDepth || frame.BitDepthChroma != tt.bitDepth {
			t.Fatalf("frame[%d] metadata = %dx%d chroma %d depth %d/%d, want 64x64 chroma %d High%d",
				i, frame.Width, frame.Height, frame.ChromaFormatIDC, frame.BitDepthLuma, frame.BitDepthChroma,
				tt.chromaFormat, tt.bitDepth)
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

func assertFFmpegHighCABACChromaWeightedPRawVideoOracle(t *testing.T, data []byte, tt highCABACChromaWeightedPCase) {
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
