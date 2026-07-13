#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PROJECT="${1:-}"

cd "$ROOT"

docker run --platform linux/amd64 --rm \
  -v "$ROOT:/work" \
  -v /work/node_modules \
  -v /work/apps/web/node_modules \
  -w /work \
  -e CI=true \
  golang:1.26-bookworm \
  bash -c '
    set -euo pipefail
    apt-get update -qq
    DEBIAN_FRONTEND=noninteractive apt-get install -y -qq curl ca-certificates >/dev/null
    curl -fsSL https://deb.nodesource.com/setup_25.x | bash - >/dev/null
    DEBIAN_FRONTEND=noninteractive apt-get install -y -qq nodejs >/dev/null
    npm ci
    cd apps/web && npx playwright install --with-deps chromium
    if [ -n "'"${PROJECT}"'" ]; then
      npm --workspace @leotime/web run test:e2e:visual-regression:update -- --project '"${PROJECT}"'
    else
      npm --workspace @leotime/web run test:e2e:visual-regression:update
    fi
  '
