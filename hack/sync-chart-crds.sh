#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUTPUT="${ROOT_DIR}/charts/cloudivision/crds/cicd.cloudivision.io_crds.yaml"
TEMP_FILE="$(mktemp "${OUTPUT}.tmp.XXXXXX")"
trap 'rm -f "${TEMP_FILE}"' EXIT

mapfile -t crds < <(find "${ROOT_DIR}/config/crd/bases" -maxdepth 1 -type f \
  -name 'cicd.cloudivision.io_*.yaml' | sort)

if [[ "${#crds[@]}" -ne 6 ]]; then
  echo "expected 6 generated cloudivision CRDs, found ${#crds[@]}" >&2
  exit 1
fi

for crd in "${crds[@]}"; do
  cat "${crd}" >>"${TEMP_FILE}"
done

mv "${TEMP_FILE}" "${OUTPUT}"
trap - EXIT
