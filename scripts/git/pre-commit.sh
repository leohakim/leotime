#!/usr/bin/env bash
set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
cd "$ROOT"

printf '\n🧪 leotime pre-commit quality gate\n\n'

make pre-commit

printf '\n✅ Pre-commit checks passed\n'
