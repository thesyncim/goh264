// SPDX-License-Identifier: LGPL-2.1-or-later

package goh264_test

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"testing"

	goh264 "github.com/thesyncim/goh264"
)

var decoderOnlyFixtureHex = map[string]string{
	"16x16_annexb_headers":  "000000016742c01f95a7a10000030001000003003c8f08042a0000000168ce3c80",
	"16x16_annexb_packet_0": "0000000165b804a0d0030e19242f3a45505b66717c87929da8141f2a35404b56616c77828d98a3aeb925303b46515c67727d88939ea9b4bfca36414c57626d78838e99a4afbac5d0db47525d68737e89949faab5c0cbd6e1ec58636e79848f9aa5b0bbc6d1dce7f2fd69747f8a95a0abb6c1ccd7e2edf8030e7a85909ba6b1bcc7d2dde8f3fe09141f8b96a1acb7c2cdd8e3eef9040f1a25309ca7b2bdc8d3dee9f4ff0a15202b3641adb8c3ced9e4effa05101b26313c4752bec9d4dfeaf5000b16212c37424d5863cfdae5f0fb06111c27323d48535e6974e0ebf6010c17222d38434e59646f7a85f1fc07121d28333e49545f6a75808b96020d18232e39444f5a65707b86919ca7293c4f6275889bae304356697c8fa2b5374a5d708396a9bc3e5164778a9db0c345586b7e91a4b7ca4c5f728598abbed15366798c9fb2c5d85a6d8093a6b9ccdf6d72777c81868b9084898e93989da2a79ba0a5aaafb4b9beb2b7bcc1c6cbd0d5c9ced3d8dde2e7ece0e5eaeff4f9fe03f7fc01060b10151a0e13181d22272c3180",
	"16x16_raw_0":           "030e19242f3a45505b66717c87929da8141f2a35404b56616c77828d98a3aeb925303b46515c67727d88939ea9b4bfca36414c57626d78838e99a4afbac5d0db47525d68737e89949faab5c0cbd6e1ec58636e79848f9aa5b0bbc6d1dce7f2fd69747f8a95a0abb6c1ccd7e2edf8030e7a85909ba6b1bcc7d2dde8f3fe09141f8b96a1acb7c2cdd8e3eef9040f1a25309ca7b2bdc8d3dee9f4ff0a15202b3641adb8c3ced9e4effa05101b26313c4752bec9d4dfeaf5000b16212c37424d5863cfdae5f0fb06111c27323d48535e6974e0ebf6010c17222d38434e59646f7a85f1fc07121d28333e49545f6a75808b96020d18232e39444f5a65707b86919ca7293c4f6275889bae304356697c8fa2b5374a5d708396a9bc3e5164778a9db0c345586b7e91a4b7ca4c5f728598abbed15366798c9fb2c5d85a6d8093a6b9ccdf6d72777c81868b9084898e93989da2a79ba0a5aaafb4b9beb2b7bcc1c6cbd0d5c9ced3d8dde2e7ece0e5eaeff4f9fe03f7fc01060b10151a0e13181d22272c31",
	"16x16_annexb_packet_1": "0000000141e02294",
	"16x16_raw_1":           "030e19242f3a45505b66717c87929da8141f2a35404b56616c77828d98a3aeb925303b46515c67727d88939ea9b4bfca36414c57626d78838e99a4afbac5d0db47525d68737e89949faab5c0cbd6e1ec58636e79848f9aa5b0bbc6d1dce7f2fd69747f8a95a0abb6c1ccd7e2edf8030e7a85909ba6b1bcc7d2dde8f3fe09141f8b96a1acb7c2cdd8e3eef9040f1a25309ca7b2bdc8d3dee9f4ff0a15202b3641adb8c3ced9e4effa05101b26313c4752bec9d4dfeaf5000b16212c37424d5863cfdae5f0fb06111c27323d48535e6974e0ebf6010c17222d38434e59646f7a85f1fc07121d28333e49545f6a75808b96020d18232e39444f5a65707b86919ca7293c4f6275889bae304356697c8fa2b5374a5d708396a9bc3e5164778a9db0c345586b7e91a4b7ca4c5f728598abbed15366798c9fb2c5d85a6d8093a6b9ccdf6d72777c81868b9084898e93989da2a79ba0a5aaafb4b9beb2b7bcc1c6cbd0d5c9ced3d8dde2e7ece0e5eaeff4f9fe03f7fc01060b10151a0e13181d22272c31",
	"32x16_annexb_headers":  "000000016742c01f95a2e840000003004000000f23c2010a800000000168ce3c80",
	"32x16_annexb_packet_0": "0000000165b804a0d0030e19242f3a45505b66717c87929da8141f2a35404b56616c77828d98a3aeb925303b46515c67727d88939ea9b4bfca36414c57626d78838e99a4afbac5d0db47525d68737e89949faab5c0cbd6e1ec58636e79848f9aa5b0bbc6d1dce7f2fd69747f8a95a0abb6c1ccd7e2edf8030e7a85909ba6b1bcc7d2dde8f3fe09141f8b96a1acb7c2cdd8e3eef9040f1a25309ca7b2bdc8d3dee9f4ff0a15202b3641adb8c3ced9e4effa05101b26313c4752bec9d4dfeaf5000b16212c37424d5863cfdae5f0fb06111c27323d48535e6974e0ebf6010c17222d38434e59646f7a85f1fc07121d28333e49545f6a75808b96020d18232e39444f5a65707b86919ca7293c4f6275889bae304356697c8fa2b5374a5d708396a9bc3e5164778a9db0c345586b7e91a4b7ca4c5f728598abbed15366798c9fb2c5d85a6d8093a6b9ccdf6d72777c81868b9084898e93989da2a79ba0a5aaafb4b9beb2b7bcc1c6cbd0d5c9ced3d8dde2e7ece0e5eaeff4f9fe03f7fc01060b10151a0e13181d22272c310d00b3bec9d4dfeaf5000b16212c37424d58c4cfdae5f0fb06111c27323d48535e69d5e0ebf6010c17222d38434e59646f7ae6f1fc07121d28333e49545f6a75808bf7020d18232e39444f5a65707b86919c08131e29343f4a55606b76818c97a2ad19242f3a45505b66717c87929da8b3be2a35404b56616c77828d98a3aeb9c4cf3b46515c67727d88939ea9b4bfcad5e04c57626d78838e99a4afbac5d0dbe6f15d68737e89949faab5c0cbd6e1ecf7026e79848f9aa5b0bbc6d1dce7f2fd08137f8a95a0abb6c1ccd7e2edf8030e1924909ba6b1bcc7d2dde8f3fe09141f2a35a1acb7c2cdd8e3eef9040f1a25303b46b2bdc8d3dee9f4ff0a15202b36414c57c1d4e7fa0d203346c8dbee0114273a4dcfe2f5081b2e4154d6e9fc0f2235485bddf00316293c4f62e4f70a1d30435669ebfe1124374a5d70f205182b3e516477959a9fa4a9aeb3b8acb1b6bbc0c5cacfc3c8cdd2d7dce1e6dadfe4e9eef3f8fdf1f6fb00050a0f14080d12171c21262b1f24292e33383d42363b40454a4f545980",
	"32x16_raw_0":           "030e19242f3a45505b66717c87929da8b3bec9d4dfeaf5000b16212c37424d58141f2a35404b56616c77828d98a3aeb9c4cfdae5f0fb06111c27323d48535e6925303b46515c67727d88939ea9b4bfcad5e0ebf6010c17222d38434e59646f7a36414c57626d78838e99a4afbac5d0dbe6f1fc07121d28333e49545f6a75808b47525d68737e89949faab5c0cbd6e1ecf7020d18232e39444f5a65707b86919c58636e79848f9aa5b0bbc6d1dce7f2fd08131e29343f4a55606b76818c97a2ad69747f8a95a0abb6c1ccd7e2edf8030e19242f3a45505b66717c87929da8b3be7a85909ba6b1bcc7d2dde8f3fe09141f2a35404b56616c77828d98a3aeb9c4cf8b96a1acb7c2cdd8e3eef9040f1a25303b46515c67727d88939ea9b4bfcad5e09ca7b2bdc8d3dee9f4ff0a15202b36414c57626d78838e99a4afbac5d0dbe6f1adb8c3ced9e4effa05101b26313c47525d68737e89949faab5c0cbd6e1ecf702bec9d4dfeaf5000b16212c37424d58636e79848f9aa5b0bbc6d1dce7f2fd0813cfdae5f0fb06111c27323d48535e69747f8a95a0abb6c1ccd7e2edf8030e1924e0ebf6010c17222d38434e59646f7a85909ba6b1bcc7d2dde8f3fe09141f2a35f1fc07121d28333e49545f6a75808b96a1acb7c2cdd8e3eef9040f1a25303b46020d18232e39444f5a65707b86919ca7b2bdc8d3dee9f4ff0a15202b36414c57293c4f6275889baec1d4e7fa0d203346304356697c8fa2b5c8dbee0114273a4d374a5d708396a9bccfe2f5081b2e41543e5164778a9db0c3d6e9fc0f2235485b45586b7e91a4b7caddf00316293c4f624c5f728598abbed1e4f70a1d304356695366798c9fb2c5d8ebfe1124374a5d705a6d8093a6b9ccdff205182b3e5164776d72777c81868b90959a9fa4a9aeb3b884898e93989da2a7acb1b6bbc0c5cacf9ba0a5aaafb4b9bec3c8cdd2d7dce1e6b2b7bcc1c6cbd0d5dadfe4e9eef3f8fdc9ced3d8dde2e7ecf1f6fb00050a0f14e0e5eaeff4f9fe03080d12171c21262bf7fc01060b10151a1f24292e33383d420e13181d22272c31363b40454a4f5459",
	"32x16_annexb_packet_1": "0000000141e0229c",
	"32x16_raw_1":           "030e19242f3a45505b66717c87929da8b3bec9d4dfeaf5000b16212c37424d58141f2a35404b56616c77828d98a3aeb9c4cfdae5f0fb06111c27323d48535e6925303b46515c67727d88939ea9b4bfcad5e0ebf6010c17222d38434e59646f7a36414c57626d78838e99a4afbac5d0dbe6f1fc07121d28333e49545f6a75808b47525d68737e89949faab5c0cbd6e1ecf7020d18232e39444f5a65707b86919c58636e79848f9aa5b0bbc6d1dce7f2fd08131e29343f4a55606b76818c97a2ad69747f8a95a0abb6c1ccd7e2edf8030e19242f3a45505b66717c87929da8b3be7a85909ba6b1bcc7d2dde8f3fe09141f2a35404b56616c77828d98a3aeb9c4cf8b96a1acb7c2cdd8e3eef9040f1a25303b46515c67727d88939ea9b4bfcad5e09ca7b2bdc8d3dee9f4ff0a15202b36414c57626d78838e99a4afbac5d0dbe6f1adb8c3ced9e4effa05101b26313c47525d68737e89949faab5c0cbd6e1ecf702bec9d4dfeaf5000b16212c37424d58636e79848f9aa5b0bbc6d1dce7f2fd0813cfdae5f0fb06111c27323d48535e69747f8a95a0abb6c1ccd7e2edf8030e1924e0ebf6010c17222d38434e59646f7a85909ba6b1bcc7d2dde8f3fe09141f2a35f1fc07121d28333e49545f6a75808b96a1acb7c2cdd8e3eef9040f1a25303b46020d18232e39444f5a65707b86919ca7b2bdc8d3dee9f4ff0a15202b36414c57293c4f6275889baec1d4e7fa0d203346304356697c8fa2b5c8dbee0114273a4d374a5d708396a9bccfe2f5081b2e41543e5164778a9db0c3d6e9fc0f2235485b45586b7e91a4b7caddf00316293c4f624c5f728598abbed1e4f70a1d304356695366798c9fb2c5d8ebfe1124374a5d705a6d8093a6b9ccdff205182b3e5164776d72777c81868b90959a9fa4a9aeb3b884898e93989da2a7acb1b6bbc0c5cacf9ba0a5aaafb4b9bec3c8cdd2d7dce1e6b2b7bcc1c6cbd0d5dadfe4e9eef3f8fdc9ced3d8dde2e7ecf1f6fb00050a0f14e0e5eaeff4f9fe03080d12171c21262bf7fc01060b10151a1f24292e33383d420e13181d22272c31363b40454a4f5459",
}

type decoderI420Frame struct {
	Width    int
	Height   int
	StrideY  int
	StrideCb int
	StrideCr int
	Y        []byte
	Cb       []byte
	Cr       []byte
}

func decoderAVCTestStream(t *testing.T, width int, height int) ([]byte, [][]byte, []decoderI420Frame) {
	t.Helper()
	headers, packets, frames := decoderAnnexBTestStream(t, width, height)
	data := append([]byte(nil), headers...)
	for _, packet := range packets {
		data = append(data, packet...)
	}
	config, samples := annexBToAVCConfigAndSamples(t, data, 4)
	if len(samples) != len(packets) {
		t.Fatalf("%dx%d AVC samples = %d, want %d", width, height, len(samples), len(packets))
	}
	return config, samples, frames
}

func decoderAnnexBTestStream(t *testing.T, width int, height int) ([]byte, [][]byte, []decoderI420Frame) {
	t.Helper()
	prefix := fmt.Sprintf("%dx%d", width, height)
	headers := decoderOnlyFixtureBytes(t, prefix+"_annexb_headers")
	packets := [][]byte{
		decoderOnlyFixtureBytes(t, prefix+"_annexb_packet_0"),
		decoderOnlyFixtureBytes(t, prefix+"_annexb_packet_1"),
	}
	frames := []decoderI420Frame{
		decoderOnlyFixtureFrame(t, width, height, 0),
		decoderOnlyFixtureFrame(t, width, height, 1),
	}
	return headers, packets, frames
}

func decoderOnlyFixtureFrame(t *testing.T, width int, height int, index int) decoderI420Frame {
	t.Helper()
	raw := decoderOnlyFixtureBytes(t, fmt.Sprintf("%dx%d_raw_%d", width, height, index))
	chromaWidth := width / 2
	chromaHeight := height / 2
	ySize := width * height
	chromaSize := chromaWidth * chromaHeight
	if len(raw) != ySize+2*chromaSize {
		t.Fatalf("%dx%d raw fixture %d length = %d", width, height, index, len(raw))
	}
	return decoderI420Frame{
		Width:    width,
		Height:   height,
		StrideY:  width,
		StrideCb: chromaWidth,
		StrideCr: chromaWidth,
		Y:        append([]byte(nil), raw[:ySize]...),
		Cb:       append([]byte(nil), raw[ySize:ySize+chromaSize]...),
		Cr:       append([]byte(nil), raw[ySize+chromaSize:]...),
	}
}

func decoderOnlyFixtureBytes(t *testing.T, key string) []byte {
	t.Helper()
	s, ok := decoderOnlyFixtureHex[key]
	if !ok {
		t.Fatalf("missing decoder fixture %s", key)
	}
	return decodeHexFixture(t, s)
}

func appendI420DecoderFrameBytes(dst []byte, frame decoderI420Frame) []byte {
	for y := 0; y < frame.Height; y++ {
		row := frame.Y[y*frame.StrideY : y*frame.StrideY+frame.Width]
		dst = append(dst, row...)
	}
	chromaWidth := frame.Width / 2
	chromaHeight := frame.Height / 2
	for y := 0; y < chromaHeight; y++ {
		row := frame.Cb[y*frame.StrideCb : y*frame.StrideCb+chromaWidth]
		dst = append(dst, row...)
	}
	for y := 0; y < chromaHeight; y++ {
		row := frame.Cr[y*frame.StrideCr : y*frame.StrideCr+chromaWidth]
		dst = append(dst, row...)
	}
	return dst
}

func assertDecodedFrameBytes(t *testing.T, frames []*goh264.Frame, want []byte) {
	t.Helper()
	if len(frames) != 1 {
		t.Fatalf("decoded frames = %d, want 1", len(frames))
	}
	raw, err := frames[0].AppendRawYUV(nil)
	if err != nil {
		t.Fatalf("AppendRawYUV: %v", err)
	}
	if !bytes.Equal(raw, want) {
		t.Fatalf("decoded raw md5 = %x, want %x", md5.Sum(raw), md5.Sum(want))
	}
}
