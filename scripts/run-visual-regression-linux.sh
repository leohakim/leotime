#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"

docker run --platform linux/amd64 --rm \
  -v "$ROOT:/work" \
  -v /work/node_modules \
  -v /work/apps/web/node_modules \
  -w /work \
  -e CI=true \
  mcr.microsoft.com/playwright:v1.61.1-noble \
  bash /work/scripts/visual-regression-linux-inner.sh run
