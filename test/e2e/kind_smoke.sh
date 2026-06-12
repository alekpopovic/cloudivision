#!/usr/bin/env bash
set -euo pipefail

missing=()
for tool in kind kubectl helm docker; do
  if ! command -v "${tool}" >/dev/null 2>&1; then
    missing+=("${tool}")
  fi
done

if [ "${#missing[@]}" -gt 0 ]; then
  echo "e2e prerequisites missing: ${missing[*]}"
  echo "Install the missing tools, then run: ./hack/kind-create.sh && ./hack/kind-load-images.sh && ./hack/install-dev.sh"
  exit 2
fi

echo "e2e prerequisites found."
echo "Skeleton only: run the kind quickstart scripts, apply deploy/examples, and assert BuildRun completion in a future full e2e."
