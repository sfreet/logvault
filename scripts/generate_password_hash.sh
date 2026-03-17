#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
HASH_TOOL="${REPO_ROOT}/bin/generate-password-hash"

if [ ! -x "${HASH_TOOL}" ]; then
    echo "Error: ${HASH_TOOL} not found. Build it with 'make build-hash-tool'." >&2
    exit 1
fi

exec "${HASH_TOOL}" "$@"
