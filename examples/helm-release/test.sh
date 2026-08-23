#!/usr/bin/env sh
set -eu
helm lint chart
rendered="$(helm template example chart --set image.repository=example.invalid/api --set image.tag=test)"
printf '%s' "$rendered" | grep -q 'runAsNonRoot: true'
printf '%s' "$rendered" | grep -q 'allowPrivilegeEscalation: false'
