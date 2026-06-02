#!/usr/bin/env bash
set -euo pipefail

ledger="${GOH264_REAL_VECTOR_FAILURE_LEDGER:-testdata/h264/realvectors/failures.jsonl}"
runner="${GOH264_RED_VECTOR_RUNNER:-scripts/h264-red-vector.sh}"
report="${GOH264_RED_VECTOR_EACH_REPORT:-${TMPDIR:-/tmp}/goh264-red-vector-each.$$.tsv}"
log_dir="${GOH264_RED_VECTOR_EACH_LOG_DIR:-${TMPDIR:-/tmp}/goh264-red-vector-each.$$.logs}"

rows_file="$(mktemp "${TMPDIR:-/tmp}/goh264-red-vector-rows.XXXXXX")"
trap 'rm -f "$rows_file"' EXIT

python3 - "$ledger" >"$rows_file" <<'PY'
import json
import sys

with open(sys.argv[1], encoding="utf-8") as manifest:
    for line_no, line in enumerate(manifest, 1):
        text = line.strip()
        if not text or text.startswith("#"):
            continue
        row = json.loads(text)
        known = row.get("known_failure") or {}
        row_id = row.get("id", "")
        if not row_id:
            raise SystemExit(f"{sys.argv[1]}:{line_no}: missing id")
        print(f"{row_id}\t{known.get('class', '')}\t{known.get('detail_contains', '')}")
PY

mkdir -p "$(dirname "$report")" "$log_dir"
printf 'status\texit\tid\tknown_failure_class\tfirst_divergence\traw_md5\tlog\n' >"$report"

tsv_field() {
    printf '%s' "$1" | tr '\t\r\n' '   '
}

log_name() {
    printf '%s' "$1" | tr -c 'A-Za-z0-9._-' '_'
}

total=0
red=0
stale=0
unexpected_pass=0
failed=0

printf 'known-red per-row runner ledger=%s report=%s log_dir=%s\n' "$ledger" "$report" "$log_dir"

while IFS=$'\t' read -r row_id known_class known_detail; do
    total=$((total + 1))
    safe_name="$(log_name "$row_id")"
    log="$log_dir/$safe_name.log"

    printf '\n[%d] %s class=%s expected_contains=%q\n' "$total" "$row_id" "$known_class" "$known_detail"
    set +e
    "$runner" "$row_id" >"$log" 2>&1
    rc=$?
    set -e

    divergence="$(grep -m 1 -E 'first divergent raw byte frame [0-9]+|first divergent frame [0-9]+ md5 = ' "$log" || true)"
    raw="$(grep -m 1 -E 'rawvideo md5 = [0-9a-f]+, want [0-9a-f]+' "$log" || true)"
    stale_detail="$(grep -m 1 -E 'failure-ledger row now matches oracle|raw frames matched but failure ledger|frame MD5s matched but failure ledger' "$log" || true)"

    if [[ "$rc" -eq 0 ]]; then
        status="unexpected-pass"
        unexpected_pass=$((unexpected_pass + 1))
    elif [[ -n "$stale_detail" ]]; then
        status="stale-ledger"
        stale=$((stale + 1))
    elif [[ -n "$divergence" || -n "$raw" ]]; then
        status="red"
        red=$((red + 1))
    else
        status="failed"
        failed=$((failed + 1))
    fi

    printf '  status=%s exit=%d\n' "$status" "$rc"
    if [[ -n "$divergence" ]]; then
        printf '  divergence: %s\n' "$divergence"
    fi
    if [[ -n "$raw" ]]; then
        printf '  raw:        %s\n' "$raw"
    fi
    if [[ -n "$stale_detail" ]]; then
        printf '  stale:      %s\n' "$stale_detail"
    fi
    printf '  log:        %s\n' "$log"

    printf '%s\t%d\t%s\t%s\t%s\t%s\t%s\n' \
        "$status" \
        "$rc" \
        "$(tsv_field "$row_id")" \
        "$(tsv_field "$known_class")" \
        "$(tsv_field "$divergence")" \
        "$(tsv_field "$raw")" \
        "$(tsv_field "$log")" >>"$report"
done <"$rows_file"

if [[ "$total" -eq 0 ]]; then
    printf 'no known-red rows found in %s\n' "$ledger" >&2
    exit 0
fi

printf '\nknown-red per-row summary total=%d red=%d stale=%d failed=%d unexpected_pass=%d report=%s\n' \
    "$total" "$red" "$stale" "$failed" "$unexpected_pass" "$report"

if [[ "$red" -ne 0 || "$stale" -ne 0 || "$failed" -ne 0 || "$unexpected_pass" -ne 0 ]]; then
    exit 1
fi
