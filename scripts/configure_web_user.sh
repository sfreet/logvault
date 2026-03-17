#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
CONFIG_TOOL="${REPO_ROOT}/bin/configure-web-user"

if [ ! -x "${CONFIG_TOOL}" ]; then
    echo "Error: ${CONFIG_TOOL} not found. Build it with 'make build-config-tool'." >&2
    exit 1
fi

exec "${CONFIG_TOOL}" "$@"
