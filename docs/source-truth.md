# Source Truth

Scope: FFmpeg `n8.0.1` H.264 decoder path only.

Proved today: progressive Annex B/AVC IDR/P/B subsets, selected High10/High12/High14
fixtures including public High10/High422 intra conformance and High10 unweighted 4:2:2/4:4:4 I/P chroma
no-deblock plus weighted 4:2:2/4:4:4 luma-only and luma+chroma P frame deblock modes 0/1 and
slice-boundary mode-2 deblock plus CAVLC/CABAC 4:2:2/4:4:4 implicit and
explicit weighted B frame deblock mode 1 and slice-boundary mode-2 deblock, High12 CAVLC IntraPCM, public CAVLC
Intra16x16 no-residual and luma-DC/luma-AC/luma-DC+AC/chroma-DC/chroma-AC/
chroma-DC+AC/luma+chroma residual plus two-frame CAVLC P-skip/P16x16
no-residual and P16x16 luma-residual/luma+chroma residual plus
P16x8/P8x16/P8x8 luma+chroma residual Annex B/AVC/configuration-record
fixtures, mixed no-residual intra, plus
Intra16x16 luma, chroma, and combined luma/chroma residual plus public
P-skip/P16x16, internal High12 no-deblock weighted P-skip and
weighted P16x16/partitioned P plus
P16x16 residual handoff plus no-deblock/mode-1 I/P deblock including
public CAVLC/CABAC 4:2:2/4:4:4 unweighted I/P no-deblock/mode-1/mode-2
deblock and CAVLC/CABAC luma-only and luma+chroma weighted-P no-deblock/mode-1/mode-2 deblock,
CAVLC/CABAC unweighted B no-deblock/mode-1/mode-2 deblock including
B-skip/direct-sub no-deblock and mode-1/mode-2 deblock plus partitioned
B16x8/B8x16/B8x8 no-deblock and mode-1/mode-2 deblock plus CAVLC/CABAC
direct-sub residual,
4:2:0 implicit weighted B no-deblock/mode-1/mode-2 deblock including
direct-sub and partitioned B16x8/B8x16/B8x8 shapes,
4:2:0 explicit weighted B no-deblock/mode-1/mode-2 deblock including
direct-sub and partitioned B16x8/B8x16/B8x8 shapes,
unweighted 4:2:2/4:4:4 I/P chroma deblock plus 4:2:0 I/P
slice-boundary mode-2 deblock and unweighted 4:2:2/4:4:4 I/P chroma
slice-boundary mode-2 deblock, High14 CAVLC/CABAC IntraPCM plus
CAVLC/CABAC no-residual Intra4x4/Intra16x16 and CAVLC/CABAC
Intra16x16 luma-DC/luma-AC/luma-DC+AC/chroma-DC/chroma-AC/
chroma-DC+AC/luma+chroma residual plus public two-frame CAVLC P-skip/P16x16
no-residual and P16x16 luma-residual/luma+chroma residual plus
P16x8/P8x16/P8x8 luma+chroma residual Annex B/AVC/configuration-record
fixtures, mixed no-residual
intra, separate and combined Intra16x16 luma/chroma residual plus
CAVLC/CABAC unweighted and weighted P-skip/P16x16 plus CAVLC mode-1/mode-2
I/P and weighted-P deblock plus public CAVLC/CABAC 4:2:2/4:4:4 unweighted I/P
no-deblock/mode-1/mode-2 deblock and CAVLC/CABAC luma-only and luma+chroma weighted-P
no-deblock/mode-1/mode-2 deblock
and CAVLC/CABAC unweighted B no-deblock/mode-1/mode-2 deblock
including B-skip/direct-sub no-deblock and mode-1/mode-2 deblock plus partitioned
B16x8/B8x16/B8x8 no-deblock and mode-1/mode-2 deblock plus CAVLC/CABAC
direct-sub residual
plus 4:2:0 implicit weighted B no-deblock/mode-1/mode-2 deblock including
direct-sub and partitioned B16x8/B8x16/B8x8 shapes
plus 4:2:0 explicit weighted B no-deblock/mode-1/mode-2 deblock including
direct-sub and partitioned B16x8/B8x16/B8x8 shapes
plus mode-1/mode-2 I/P and weighted-P deblock, High10 frame-MBAFF field-coded CAVLC IntraPCM
entropy/reconstruct pairing plus public High12/High14 4:2:0 frame-MBAFF CAVLC
IntraPCM, P-skip, and field-coded/frame-coded P16x16/P16x8/P8x16/P8x8 no-residual, luma-residual, and luma+chroma-residual rows plus P-skip and field-coded/frame-coded P16x16/P16x8/P8x16/P8x8 mode-1/mode-2 deblock rows plus explicit weighted-P field-coded/frame-coded P16x16/P16x8/P8x16/P8x8 no-residual, luma-residual, and luma+chroma-residual rows across no-deblock/mode-1/mode-2 plus temporal/spatial direct B-skip no-deblock/mode-1/mode-2 rows plus temporal/spatial direct-sub B8x8 field-coded/frame-coded no-deblock/mode-1/mode-2 rows plus field-coded/frame-coded partitioned B16x8/B8x16/B8x8 no-deblock/mode-1/mode-2 rows plus implicit weighted temporal/spatial direct B-skip, direct-sub B8x8 field-coded/frame-coded, and field-coded/frame-coded partitioned B16x8/B8x16/B8x8 no-deblock/mode-1/mode-2 rows plus explicit weighted temporal/spatial direct B-skip, direct-sub B8x8 field-coded/frame-coded, and field-coded/frame-coded partitioned B16x8/B8x16/B8x8 no-deblock/mode-1/mode-2 rows plus public High10/High422 field-coded
frame-MBAFF deblock rows plus public derived High10 4:2:2/4:4:4 luma-only
and luma+chroma weighted-P slice-boundary mode-2 rows plus internal High10 4:2:2/4:4:4 top/bottom field
explicit/implicit weighted B luma/chroma deblock modes 0/1/2 plus internal High10 4:2:2/4:4:4 top/bottom field
luma-only, luma+chroma, and source-normalized chroma-only weighted P deblock modes 0/1/2 plus
internal High12/High14 4:2:2/4:4:4 top/bottom field explicit/implicit weighted B plus luma-only, luma+chroma,
and source-normalized chroma-only weighted P deblock modes 0/1/2, complete FFmpeg FRext FATE row coverage including
monochrome-to-yuv420p output, selected compact baseline/main conformance rows
including AUD, SVA/Sony/MW/FT/JVC/Sand/Toshiba/VTC baseline/main, CABAC,
no-deblock, reference, field, and slice-boundary rows,
CABAC delayed-output reorder plus weighted/field compact rows, CAVLC
field/QP/crop/weighted/no-deblock compact rows, compact
long-slice/multiple-reference rows, frame-num-gap multiple-reference rows, and
MMCO-reset multiple-reference reorder rows,
non-conformance Annex B FATE rows covering extreme plane prediction,
reference-list-mod overflow recovery, missing-frame recovery, and
large-to-small SPS reinit metadata, with
packetized/configured surfaces proven for reference-list-mod overflow,
missing-frame, timecode, and BBC2 recovery/PAFF,
lossless High444 transform-bypass Annex B plus 4-byte AVC/configuration-record surfaces,
public High9 4:2:0 SPS reinit metadata from 9-bit to 8-bit output,
High444 10-bit SPS reinit metadata, XAVC High422 terminal damaged
first-field recovery, container-backed FATE H.264 vectors extracted to Annex B
with both source and extracted bitstream MD5s, auxiliary API/MOV/MPEG-TS/MKV/CBS H.264 sample rows,
public FATE vector harness,
raw-diff/frame-MD5 diagnostics, and CLI benchmark comparison.

Public vectors: 226 imported public refs, 225 selected decoder-facing manifest
rows, 225 green oracle rows, 0 known-red, and one explicit non-decoder
exclusion. The always-on
`TestH264DecoderTDDContractClassifiesEveryImportedPublicVector` test treats
that inventory as the decoder TDD backlog: every imported ref must be executable
in the manifest, documented as an exclusion, or recorded as a known-red
manifest row. Matrix mode is the safe-point gate. Green oracle rows include
decode-ok, metadata-ok, and expected decode-error rows. The strict script
excludes only rows in
`testdata/h264/realvectors/failures.jsonl`, which is currently empty; the
failure-ledger freshness and matrix gates execute known-red rows when present
and require them to remain current. The red-queue script intentionally exits
non-zero while selected known-red rows remain red.
Use `scripts/h264-real-vector-red-queue.sh <filter>` or
`scripts/h264-red-vector.sh <filter>` to hand agents a failing public lane; the
single-lane script exits at the first divergent raw byte for raw-MD5 rows.
Use `scripts/h264-real-vector-red-each.sh` for per-known-red logs plus a TSV
summary of first-divergence evidence.

Imported public-vector inventory: 224 refs are generated from the pinned FFmpeg
`n8.0.1` FATE/CBS scan and 2 additional public fate-suite H.264/LCEVC auxiliary
container refs are represented in the manifest. The FATE-derived subset has 223
refs represented in the public-vector manifest and one explicitly excluded
non-decoder row. `h264-conformance/FM1_BT_B.h264` is a negative decode-error
manifest row; `mkv/h264_tta_undecodable.mkv` remains excluded because it has no
H.264 video stream.

Still guarded: unselected MBAFF/PIC-AFF/PAFF motion paths, broad public field/MBAFF
high modes, public 12/14-bit
high streams beyond the current FFmpeg FATE 8-bit/10-bit set, broader
damaged-slice error resilience, threading/SIMD, and full libavcodec
delayed-output behavior.

Intentionally unsupported at the pinned FFmpeg parity boundary: FMO, 11/13-bit
luma depths, `chroma_format_idc > 3`, separate color planes, and mixed
chroma/luma bit depths. FFmpeg n8.0.1 has `FMO` compiled out in
`libavcodec/h264dec.h`, and `ff_h264_decode_picture_parameter_set` returns
PATCHWELCOME as soon as `num_slice_groups_minus1 > 0` or `bit_depth_luma` is
11 or 13; local PPS parsing mirrors both with `ErrUnsupported`. FFmpeg
`ff_h264_decode_seq_parameter_set` also fails SPS admission for unsupported
chroma formats, separate color planes, and different chroma/luma bit depths;
local SPS parsing mirrors those with `ErrUnsupported`.

Frame-MBAFF direct-sub B4x4 rows are not listed as missing proof: non-frame-only
SPS syntax must set `direct_8x8_inference_flag`, so the valid generated
frame-MBAFF direct-sub B surface is B8x8-only.

Canonical detail lives in manifests and tests, not Markdown.
