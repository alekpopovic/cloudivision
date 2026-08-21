#!/usr/bin/env bash
set -euo pipefail

UPGRADE_BASE_REF="${UPGRADE_BASE_REF:-}"
UPGRADE_TARGET_IMAGE="${UPGRADE_TARGET_IMAGE:-}"

if [[ -z "${UPGRADE_BASE_REF}" || -z "${UPGRADE_TARGET_IMAGE}" ]]; then
  echo "upgrade test skeleton: UPGRADE_BASE_REF and UPGRADE_TARGET_IMAGE are required" >&2
  echo "see test/upgrade/README.md for the staged upgrade contract" >&2
  exit 2
fi

echo "upgrade automation is intentionally not enabled until a released base artifact exists" >&2
echo "base ref: ${UPGRADE_BASE_REF}; target image: ${UPGRADE_TARGET_IMAGE}" >&2
exit 2
