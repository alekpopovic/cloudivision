#!/bin/sh
set -eu

API_BASE_URL="${CLOU_DIVISION_API_BASE_URL:-${API_BASE_URL:-}}"
config_path="/usr/share/nginx/html/assets/config.json"

if [ -e "${config_path}" ] && [ ! -w "${config_path}" ]; then
  exit 0
fi

cat > "${config_path}" <<EOF
{"apiBaseUrl":"${API_BASE_URL}"}
EOF
