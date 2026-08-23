#!/usr/bin/env sh
set -eu

manifest=${1:?usage: no-quadratic-crd-schema.sh RENDERED_MANIFEST}

if grep -Eq '^[[:space:]]+uniqueItems:[[:space:]]+true[[:space:]]*$' "$manifest"; then
  echo "FAIL: CRD schema uses uniqueItems, which modern Kubernetes rejects for quadratic validation cost" >&2
  exit 1
fi

echo "PASS: CRD schemas avoid quadratic uniqueItems validation"
