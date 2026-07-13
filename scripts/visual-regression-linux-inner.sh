#!/usr/bin/env bash
set -euo pipefail

MODE="${1:-run}"
PROJECT="${2:-}"

if ! command -v go >/dev/null 2>&1; then
  curl -fsSL https://go.dev/dl/go1.26.5.linux-amd64.tar.gz -o /tmp/go.tgz
  rm -rf /usr/local/go
  tar -C /usr/local -xzf /tmp/go.tgz
  export PATH=/usr/local/go/bin:$PATH
fi

npm ci

if [ "$MODE" = "update" ]; then
  if [ -n "$PROJECT" ]; then
    npm --workspace @leotime/web run test:e2e:visual-regression:update -- --project "$PROJECT"
  else
    npm --workspace @leotime/web run test:e2e:visual-regression:update
  fi
else
  npm --workspace @leotime/web run test:e2e:visual-regression
fi
