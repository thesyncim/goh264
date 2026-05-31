# goh264

`goh264` is a source-shaped Go port of FFmpeg `libavcodec`'s H.264 decoder path.

The port is intentionally decoder-only. Encoder, muxer, filter, and unrelated codec code are out of scope unless an H.264 decoder file directly depends on a shared primitive.

Current safe point:

- Upstream source truth is pinned to FFmpeg `n8.0.1`, commit `894da5ca7d742e4429ffb2af534fcda0103ef593`.
- The active Go slice covers Annex B NAL splitting, RBSP emulation-prevention removal, bit reading, unsigned/signed Exp-Golomb reads, and SPS metadata parsing.
- Frame reconstruction, CABAC/CAVLC residual decode, prediction, inverse transform, deblocking, DPB, threading, and public `Decode` output are not yet implemented.

Run the default tests:

```sh
go test ./...
```

Run tests that call the pinned native oracle tools available on this machine:

```sh
GOH264_ORACLE=1 go test ./...
```

Fetch the pinned upstream source snapshot into the ignored local cache:

```sh
scripts/fetch-upstream.sh
```

The file-by-file status lives in [docs/translation-ledger.md](docs/translation-ledger.md).

