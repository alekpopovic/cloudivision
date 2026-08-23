#!/usr/bin/env bash
set -Eeuo pipefail

REPOSITORY="${CLOUDIVISION_CLI_REPOSITORY:-alekpopovic/cloudivision}"
VERSION_VALUE="${CLOUDIVISION_CLI_VERSION:-latest}"
INSTALL_DIR="${CLOUDIVISION_CLI_INSTALL_DIR:-/usr/local/bin}"

case "$(uname -s)" in
  Linux) os=linux ;;
  Darwin) os=darwin ;;
  *) echo "install-cli: unsupported operating system; use the Windows binary download" >&2; exit 1 ;;
esac
case "$(uname -m)" in
  x86_64|amd64) arch=amd64 ;;
  arm64|aarch64) arch=arm64 ;;
  *) echo "install-cli: unsupported architecture $(uname -m)" >&2; exit 1 ;;
esac

if [[ "${VERSION_VALUE}" == latest ]]; then
  VERSION_VALUE="$(curl -fsSL "https://api.github.com/repos/${REPOSITORY}/releases/latest" | sed -n 's/.*"tag_name": *"v\{0,1\}\([^"]*\)".*/\1/p' | head -n1)"
fi
[[ -n "${VERSION_VALUE}" ]] || { echo "install-cli: could not resolve release version" >&2; exit 1; }

archive="cloudivision-cli_${VERSION_VALUE}_${os}_${arch}.tar.gz"
base_url="https://github.com/${REPOSITORY}/releases/download/v${VERSION_VALUE}"
temp_dir="$(mktemp -d)"
trap 'rm -rf -- "${temp_dir}"' EXIT

curl -fsSL "${base_url}/${archive}" -o "${temp_dir}/${archive}"
curl -fsSL "${base_url}/SHA256SUMS" -o "${temp_dir}/SHA256SUMS"
(cd "${temp_dir}" && grep " ${archive}$" SHA256SUMS | sha256sum -c -)
tar -xzf "${temp_dir}/${archive}" -C "${temp_dir}"
install -m 0755 "${temp_dir}/cloudivision" "${INSTALL_DIR}/cloudivision"
"${INSTALL_DIR}/cloudivision" version
