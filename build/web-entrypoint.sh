#!/bin/sh
set -eu

API_BASE_URL="${CLOU_DIVISION_API_BASE_URL:-${API_BASE_URL:-}}"
cat > /usr/share/nginx/html/assets/config.json <<EOF
{"apiBaseUrl":"${API_BASE_URL}"}
EOF
