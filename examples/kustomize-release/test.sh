#!/usr/bin/env sh
set -eu
rendered="$(kubectl kustomize overlays/dev)"
printf '%s' "$rendered" | grep -q 'namespace: example-dev'
printf '%s' "$rendered" | grep -q 'ghcr.io/example/example-api:dev'
printf '%s' "$rendered" | grep -q 'allowPrivilegeEscalation: false'
