#!/usr/bin/env bash
set -euo pipefail

start_epoch_ns="$(date +%s%N)"

file="tests/file-import-all.csv"
branch_id="1"
ref_code="50"
batch="20000"
x_client_code="cron"

url_base="http://127.0.0.1:9009/"
email="hdepablos@libgot.com"
password="123456"

retry_max="8"
rate_limit_window_seconds="${RATE_LIMIT_WINDOW_SECONDS:-60}"
rate_limit_window_sleep_seconds="$((rate_limit_window_seconds + 5))"
imports_rate_limit_per_minute="${RATE_LIMIT_IMPORTS_PER_MINUTE:-5000}"
per_request_sleep_seconds="0"

if [[ ! -f "$file" ]]; then
  echo "file not found: $file" >&2
  exit 1
fi

if [[ "${url_base: -1}" != "/" ]]; then
  url_base="${url_base}/"
fi

login_url="${url_base}api/v1/auth/login"
imports_base_url="${url_base}api/v1/imports/all"
logout_url="${url_base}api/v1/auth/logout"

key_code="$(date +%y%m%d%H%M%S)$(printf '%04d' $(( (RANDOM % 9999) + 1000 )))"

total_lines="$(wc -l < "$file" | tr -d '[:space:]')"
if [[ -z "$total_lines" || "$total_lines" -lt 2 ]]; then
  echo "file must contain header + at least 1 data row: $file" >&2
  exit 1
fi

data_rows="$((total_lines - 1))"
chunks_expected="$(
  python3 - <<PY
import math
data_rows = int("${data_rows}")
batch = int("${batch}")
print(int(math.ceil(data_rows / batch)) if batch > 0 else 0)
PY
)"

per_request_sleep_seconds="$(
  python3 - <<PY
limit_per_minute = int("${imports_rate_limit_per_minute}")
if limit_per_minute <= 0:
  print("0")
else:
  print(f"{60/limit_per_minute:.6f}")
PY
)"

token="$(
  attempt="1"
  while true; do
    login_body_file="$(mktemp)"
    http_code="$(
      curl -sS -o "$login_body_file" -w "%{http_code}" \
        -X POST "$login_url" \
        -H "Content-Type: application/json" \
        -H "X-Client-Code: ${x_client_code}" \
        -d "{\"email\":\"${email}\",\"password\":\"${password}\"}"
    )"

    if [[ "$http_code" == "200" ]]; then
      token="$(
        python3 - "$login_body_file" <<'PY'
import json, sys
path = sys.argv[1]
with open(path, "r", encoding="utf-8") as f:
  data = json.load(f)
token = (data.get("data") or {}).get("access_token")
if not token:
  raise SystemExit("missing data.access_token")
print(token)
PY
      )"
      rm -f "$login_body_file"
      printf "%s" "$token"
      break
    fi

    if [[ "$http_code" == "429" && "$attempt" -lt "$retry_max" ]]; then
      echo "rate limit (http=429) on login attempt=${attempt}/${retry_max} sleeping=${rate_limit_window_sleep_seconds}s" >&2
      rm -f "$login_body_file"
      sleep "$rate_limit_window_sleep_seconds"
      attempt="$((attempt + 1))"
      continue
    fi

    echo "login failed (http=${http_code})" >&2
    if [[ -s "$login_body_file" ]]; then
      echo "response body:" >&2
      head -c 2000 "$login_body_file" >&2 || true
      echo >&2
    fi
    rm -f "$login_body_file"
    exit 1
  done
)"

tmp_dir="$(mktemp -d)"
cleanup() {
  rm -rf "$tmp_dir"
}
trap cleanup EXIT

awk -v batch="$batch" -v outdir="$tmp_dir" '
  NR==1 { header=$0; next }
  {
    file = int((NR-2)/batch) + 1
    fname = sprintf("%s/chunk_%03d.csv", outdir, file)
    if (!(file in seen)) { print header > fname; seen[file]=1 }
    print $0 >> fname
  }
' "$file"

shopt -s nullglob
chunks=("$tmp_dir"/chunk_*.csv)
if [[ "${#chunks[@]}" -eq 0 ]]; then
  echo "no chunks were generated" >&2
  exit 1
fi

for chunk_file in "${chunks[@]}"; do
  url="${imports_base_url}/${branch_id}/${ref_code}/${total_lines}/${key_code}"
  attempt="1"
  while true; do
    resp_body_file="${tmp_dir}/resp_$(basename "$chunk_file").json"
    http_code="$(
      curl -sS -o "$resp_body_file" -w "%{http_code}" \
        -X POST "$url" \
        -H "Authorization: Bearer ${token}" \
        -H "X-Client-Code: ${x_client_code}" \
        -F "file=@${chunk_file};filename=$(basename "$chunk_file")"
    )"

    if [[ "$http_code" == "200" ]]; then
      break
    fi

    if [[ "$http_code" == "429" && "$attempt" -lt "$retry_max" ]]; then
      echo "rate limit (http=429) on $(basename "$chunk_file") attempt=${attempt}/${retry_max} sleeping=${rate_limit_window_sleep_seconds}s" >&2
      sleep "$rate_limit_window_sleep_seconds"
      attempt="$((attempt + 1))"
      continue
    fi

    echo "upload failed for $(basename "$chunk_file") (http=${http_code})" >&2
    if [[ -s "$resp_body_file" ]]; then
      echo "response body:" >&2
      head -c 2000 "$resp_body_file" >&2 || true
      echo >&2
    fi
    exit 1
  done

  if [[ "$per_request_sleep_seconds" != "0" && "$per_request_sleep_seconds" != "0.000000" ]]; then
    sleep "$per_request_sleep_seconds"
  fi
done

curl -sS -o /dev/null \
  -X POST "$logout_url" \
  -H "Authorization: Bearer ${token}" \
  -H "X-Client-Code: ${x_client_code}" \
  -H "Content-Type: application/json" \
  -d '{}' || true

end_epoch_ns="$(date +%s%N)"
elapsed_ns="$((end_epoch_ns - start_epoch_ns))"
elapsed_ms="$((elapsed_ns / 1000000))"
elapsed_h="$((elapsed_ms / 3600000))"
elapsed_m="$(((elapsed_ms % 3600000) / 60000))"
elapsed_s="$(((elapsed_ms % 60000) / 1000))"
elapsed_rem_ms="$((elapsed_ms % 1000))"

printf "ok key_code=%s chunks=%s total_lines=%s elapsed=%02d:%02d:%02d.%03d\n" \
  "$key_code" "${#chunks[@]}" "$total_lines" "$elapsed_h" "$elapsed_m" "$elapsed_s" "$elapsed_rem_ms"

printf "info imports_rate_limit_per_minute=%s per_request_sleep_seconds=%s chunks_expected=%s\n" \
  "$imports_rate_limit_per_minute" "$per_request_sleep_seconds" "$chunks_expected"
