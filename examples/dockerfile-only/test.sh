#!/usr/bin/env sh
set -eu
grep -q 'Built by cloudivision' index.html
grep -q '^USER nginx$' Dockerfile
! grep -Eq 'docker\.sock|privileged:[[:space:]]*true' Dockerfile
