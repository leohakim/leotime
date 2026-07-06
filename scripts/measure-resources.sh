#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

BASE_URL="${BASE_URL:-http://127.0.0.1:8080}"
SAMPLE_SECONDS="${SAMPLE_SECONDS:-60}"
SAMPLE_INTERVAL="${SAMPLE_INTERVAL:-5}"
WITH_LOAD="${WITH_LOAD:-0}"
K6_VUS="${K6_VUS:-10}"
K6_DURATION="${K6_DURATION:-30s}"
SERVICE="${SERVICE:-leotime}"

printf '\n📊 leotime resource measurement\n\n'
printf '  service:         %s\n' "$SERVICE"
printf '  base url:        %s\n' "$BASE_URL"
printf '  sample window:   %ss every %ss\n' "$SAMPLE_SECONDS" "$SAMPLE_INTERVAL"
printf '  load during run: %s\n\n' "$WITH_LOAD"

if ! command -v docker >/dev/null 2>&1; then
  printf '❌ docker is required\n' >&2
  exit 1
fi

printf '🐳 Ensuring app is reachable...\n'
if curl -fsS "$BASE_URL/api/health" >/dev/null 2>&1; then
  printf '✅ API already healthy at %s\n' "$BASE_URL"
else
  printf '🐳 Starting Docker stack...\n'
  docker compose up -d "$SERVICE"
  for _ in $(seq 1 30); do
    if curl -fsS "$BASE_URL/api/health" >/dev/null 2>&1; then
      break
    fi
    sleep 1
  done
fi

curl -fsS "$BASE_URL/api/health" >/dev/null
curl -fsS "$BASE_URL/metrics" >/dev/null

container_id="$(docker compose ps -q "$SERVICE" 2>/dev/null || true)"
if [ -z "$container_id" ]; then
  container_id="$(docker ps -q --filter "publish=8080" | head -n 1)"
fi
if [ -z "$container_id" ]; then
  printf '\n⚠️  No Docker container found for %s. API is up, but docker stats are unavailable.\n' "$SERVICE"
  printf '    Stop the local dev server or run: make down && make up\n\n'
  metrics="$(curl -fsS "$BASE_URL/metrics")"
  printf '=== Prometheus process metrics (snapshot) ===\n'
  printf '%s\n' "$metrics" | awk '
    /^process_resident_memory_bytes / {printf "  process_resident_memory_bytes: %s MiB\n", $2/1024/1024}
    /^process_virtual_memory_bytes / {printf "  process_virtual_memory_bytes:  %s MiB\n", $2/1024/1024}
    /^go_goroutines / {printf "  go_goroutines:                 %s\n", $2}
    /^go_memstats_alloc_bytes / {printf "  go_memstats_alloc_bytes:       %s MiB\n", $2/1024/1024}
    /^go_memstats_heap_inuse_bytes / {printf "  go_memstats_heap_inuse_bytes:  %s MiB\n", $2/1024/1024}
  '
  exit 0
fi

load_pid=""
if [ "$WITH_LOAD" = "1" ]; then
  printf '🔥 Starting k6 load (%s VUs, %s)...\n' "$K6_VUS" "$K6_DURATION"
  docker compose --profile tools run --rm \
    -e BASE_URL="http://${SERVICE}:8080" \
    -e K6_VUS="$K6_VUS" \
    -e K6_DURATION="$K6_DURATION" \
    k6 run /scripts/leotime-smoke.js >/tmp/leotime-k6-measure.log 2>&1 &
  load_pid=$!
fi

samples_file="$(mktemp)"
trap 'rm -f "$samples_file"; [ -n "$load_pid" ] && kill "$load_pid" 2>/dev/null || true' EXIT

printf '⏱️ Sampling docker stats...\n'
end=$((SECONDS + SAMPLE_SECONDS))
while [ "$SECONDS" -lt "$end" ]; do
  date -u +"%Y-%m-%dT%H:%M:%SZ" >>"$samples_file"
  docker stats --no-stream --format '{{.CPUPerc}} {{.MemUsage}} {{.MemPerc}} {{.NetIO}} {{.BlockIO}} {{.PIDs}}' "$container_id" >>"$samples_file"
  sleep "$SAMPLE_INTERVAL"
done

if [ -n "$load_pid" ]; then
  wait "$load_pid" || true
fi

mem_mib=()
cpu_values=()
while IFS= read -r line; do
  case "$line" in
    *MiB* | *GiB*)
      mem_value="$(printf '%s\n' "$line" | awk '{print $2}' | sed -E 's/([0-9.]+)(MiB|GiB)/\1 \2/')"
      amount="$(printf '%s\n' "$mem_value" | awk '{print $1}')"
      unit="$(printf '%s\n' "$mem_value" | awk '{print $2}')"
      if [ "$unit" = "GiB" ]; then
        amount="$(awk "BEGIN {printf \"%.2f\", $amount * 1024}")"
      fi
      mem_mib+=("$amount")
      cpu_values+=("$(printf '%s\n' "$line" | awk '{print $1}' | tr -d '%')")
      ;;
  esac
done <"$samples_file"

avg_mem="$(printf '%s\n' "${mem_mib[@]}" | awk '{sum+=$1; n+=1} END {if (n) printf "%.1f", sum/n; else print "0"}')"
peak_mem="$(printf '%s\n' "${mem_mib[@]}" | awk 'BEGIN {max=0} {if ($1>max) max=$1} END {printf "%.1f", max}')"
avg_cpu="$(printf '%s\n' "${cpu_values[@]}" | awk '{sum+=$1; n+=1} END {if (n) printf "%.2f", sum/n; else print "0"}')"
peak_cpu="$(printf '%s\n' "${cpu_values[@]}" | awk 'BEGIN {max=0} {if ($1>max) max=$1} END {printf "%.2f", max}')"

printf '\n=== Docker summary (%s) ===\n' "$SERVICE"
docker stats --no-stream --format 'table {{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.MemPerc}}\t{{.NetIO}}\t{{.BlockIO}}\t{{.PIDs}}' "$container_id"
printf '\nSampled over %ss (%s samples):\n' "$SAMPLE_SECONDS" "${#mem_mib[@]}"
printf '  avg CPU:       %s%%\n' "$avg_cpu"
printf '  peak CPU:      %s%%\n' "$peak_cpu"
printf '  avg memory:    %s MiB\n' "$avg_mem"
printf '  peak memory:   %s MiB\n' "$peak_mem"

if command -v curl >/dev/null 2>&1; then
  printf '\n=== Prometheus process metrics (snapshot) ===\n'
  metrics="$(curl -fsS "$BASE_URL/metrics")"
  printf '%s\n' "$metrics" | awk '
    /^process_resident_memory_bytes / {printf "  process_resident_memory_bytes: %s MiB\n", $2/1024/1024}
    /^process_virtual_memory_bytes / {printf "  process_virtual_memory_bytes:  %s MiB\n", $2/1024/1024}
    /^go_goroutines / {printf "  go_goroutines:                 %s\n", $2}
    /^go_memstats_alloc_bytes / {printf "  go_memstats_alloc_bytes:       %s MiB\n", $2/1024/1024}
    /^go_memstats_heap_inuse_bytes / {printf "  go_memstats_heap_inuse_bytes:  %s MiB\n", $2/1024/1024}
  '
fi

db_path="$(docker compose exec -T "$SERVICE" sh -lc 'printf "%s" "$LEOTIME_DB_PATH"' 2>/dev/null || true)"
if [ -n "$db_path" ]; then
  db_size="$(docker compose exec -T "$SERVICE" sh -lc "wc -c < '$db_path' 2>/dev/null || true")"
  if [ -n "$db_size" ]; then
    printf '\n=== SQLite file ===\n'
    printf '  path: %s\n' "$db_path"
    printf '  size: %.2f MiB\n' "$(awk "BEGIN {printf \"%.2f\", $db_size/1024/1024}")"
  fi
fi

printf '\n=== Solidtime reference (your VPS snapshot) ===\n'
printf '  queue:      ~171 MiB\n'
printf '  scheduler:  ~37 MiB\n'
printf '  app:        ~558 MiB\n'
printf '  database:   ~51 MiB\n'
printf '  total RAM:  ~817 MiB across 4 containers\n'
printf '  leotime idle peak above: %.1f MiB in 1 container\n' "$peak_mem"
printf '\nNotes:\n'
printf '  - leotime ships API + static web + SQLite in one Go process.\n'
printf '  - No queue, scheduler, or mail worker yet (password reset, long-running timer emails).\n'
printf '  - Re-run with load: WITH_LOAD=1 make resources\n'
printf '  - Longer window: SAMPLE_SECONDS=300 make resources\n\n'
