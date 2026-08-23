#!/usr/bin/env bash
set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
VERSION_VALUE="${1:-$(tr -d '[:space:]' <"${ROOT_DIR}/VERSION")}"
OUTPUT_ROOT="${RELEASE_OUTPUT_DIR:-${ROOT_DIR}/dist/release}"
RELEASE_DIR="${OUTPUT_ROOT}/v${VERSION_VALUE}"
REGISTRY="${RELEASE_IMAGE_REGISTRY:-ghcr.io/alekpopovic/cloudivision}"

fail() { echo "release: $*" >&2; exit 1; }
require() { command -v "$1" >/dev/null 2>&1 || fail "required command not found: $1"; }

[[ "${VERSION_VALUE}" =~ ^[0-9]+\.[0-9]+\.[0-9]+([+-][0-9A-Za-z.-]+)?$ ]] || fail "version must be semantic (got ${VERSION_VALUE})"
require git
require go
require npm
require helm
require tar
require sha256sum

CHART_VERSION="$(awk '$1 == "version:" {print $2}' "${ROOT_DIR}/charts/cloudivision/Chart.yaml")"
APP_VERSION="$(awk '$1 == "appVersion:" {gsub(/\"/, "", $2); print $2}' "${ROOT_DIR}/charts/cloudivision/Chart.yaml")"
[[ "${CHART_VERSION}" == "${VERSION_VALUE}" ]] || fail "Chart.yaml version ${CHART_VERSION} does not match ${VERSION_VALUE}"
[[ "${APP_VERSION}" == "${VERSION_VALUE}" ]] || fail "Chart.yaml appVersion ${APP_VERSION} does not match ${VERSION_VALUE}"

if [[ "${RELEASE_ALLOW_DIRTY:-false}" != "true" ]] && [[ -n "$(git -C "${ROOT_DIR}" status --porcelain)" ]]; then
  fail "worktree is dirty; commit changes or set RELEASE_ALLOW_DIRTY=true for a non-publishable test build"
fi
[[ ! -e "${RELEASE_DIR}" ]] || fail "output already exists: ${RELEASE_DIR}"
mkdir -p "${RELEASE_DIR}"

if [[ "${RELEASE_SKIP_CHECKS:-false}" != "true" ]]; then
  test -z "$(gofmt -l "${ROOT_DIR}/api" "${ROOT_DIR}/cmd" "${ROOT_DIR}/internal")" || fail "gofmt check failed"
  (cd "${ROOT_DIR}" && go test ./... && go vet ./...)
  npm --prefix "${ROOT_DIR}/web" ci
  npm --prefix "${ROOT_DIR}/web" audit --omit=dev --audit-level=high
  npm --prefix "${ROOT_DIR}/web" run build
  npm --prefix "${ROOT_DIR}/web" test -- --watch=false --browsers=ChromeHeadless
  (cd "${ROOT_DIR}" && make security-check && make upgrade-test && make scale-test)
  if [[ "${RELEASE_RUN_CONFORMANCE:-false}" == "true" ]]; then
    (cd "${ROOT_DIR}" && make conformance)
  else
    echo "release: SKIP live conformance (set RELEASE_RUN_CONFORMANCE=true with a disposable cluster)"
  fi
else
  npm --prefix "${ROOT_DIR}/web" run build
fi

GIT_SHA="$(git -C "${ROOT_DIR}" rev-parse --verify HEAD)"
SHORT_SHA="$(git -C "${ROOT_DIR}" rev-parse --short=12 HEAD)"
BUILD_DATE="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
TARGET_GOOS="${RELEASE_GOOS:-linux}"
TARGET_GOARCH="${RELEASE_GOARCH:-amd64}"
BUILD_DIR="${RELEASE_DIR}/build"
mkdir -p "${BUILD_DIR}"

for component in api controller runner; do
  (cd "${ROOT_DIR}" && CGO_ENABLED=0 GOOS="${TARGET_GOOS}" GOARCH="${TARGET_GOARCH}" go build -trimpath -ldflags "-s -w" \
    -o "${BUILD_DIR}/cloudivision-${component}" "./cmd/${component}")
  tar -C "${BUILD_DIR}" -czf "${RELEASE_DIR}/cloudivision-${component}_${VERSION_VALUE}_${TARGET_GOOS}_${TARGET_GOARCH}.tar.gz" "cloudivision-${component}"
done
CLI_PLATFORMS="${RELEASE_CLI_PLATFORMS:-linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64}"
for platform in ${CLI_PLATFORMS}; do
  cli_os="${platform%/*}"
  cli_arch="${platform#*/}"
  cli_binary="cloudivision"
  [[ "${cli_os}" != windows ]] || cli_binary="cloudivision.exe"
  (cd "${ROOT_DIR}" && CGO_ENABLED=0 GOOS="${cli_os}" GOARCH="${cli_arch}" go build -trimpath \
    -ldflags "-s -w -X main.version=${VERSION_VALUE} -X main.commit=${GIT_SHA} -X main.buildDate=${BUILD_DATE}" \
    -o "${BUILD_DIR}/${cli_binary}" "./cmd/cloudivision")
  if [[ "${cli_os}" == windows ]]; then
    require zip
    (cd "${BUILD_DIR}" && zip -q "${RELEASE_DIR}/cloudivision-cli_${VERSION_VALUE}_${cli_os}_${cli_arch}.zip" "${cli_binary}")
  else
    tar -C "${BUILD_DIR}" -czf "${RELEASE_DIR}/cloudivision-cli_${VERSION_VALUE}_${cli_os}_${cli_arch}.tar.gz" "${cli_binary}"
  fi
  rm -f -- "${BUILD_DIR}/${cli_binary}"
done
rm -rf -- "${BUILD_DIR}"
tar -C "${ROOT_DIR}/web/dist/cloudivision-web/browser" -czf \
  "${RELEASE_DIR}/cloudivision-web_${VERSION_VALUE}.tar.gz" .

helm package "${ROOT_DIR}/charts/cloudivision" --destination "${RELEASE_DIR}" >/dev/null
cp "${ROOT_DIR}/charts/cloudivision/crds/cicd.cloudivision.io_crds.yaml" \
  "${RELEASE_DIR}/cloudivision-crds_${VERSION_VALUE}.yaml"

printf '%s\n' \
  "version=${VERSION_VALUE}" \
  "git_sha=${GIT_SHA}" \
  "controller_image=${REGISTRY}/controller:${VERSION_VALUE}" \
  "api_image=${REGISTRY}/api:${VERSION_VALUE}" \
  "runner_image=${REGISTRY}/runner:${VERSION_VALUE}" \
  "web_image=${REGISTRY}/web:${VERSION_VALUE}" \
  "sha_tag=sha-${SHORT_SHA}" \
  >"${RELEASE_DIR}/cloudivision-${VERSION_VALUE}.manifest"

if [[ "${RELEASE_GENERATE_SBOM:-auto}" == "true" ]] || { [[ "${RELEASE_GENERATE_SBOM:-auto}" == "auto" ]] && command -v syft >/dev/null 2>&1; }; then
  require syft
  syft "dir:${ROOT_DIR}" -o "spdx-json=${RELEASE_DIR}/cloudivision-${VERSION_VALUE}.source.spdx.json"
else
  echo "release: SKIP source SBOM (install syft or set RELEASE_GENERATE_SBOM=true)"
fi

(
  cd "${RELEASE_DIR}"
  sha256sum cloudivision-* >SHA256SUMS
)

if [[ -n "${RELEASE_COSIGN_KEY:-}" ]]; then
  require cosign
  while IFS= read -r artifact; do
    cosign sign-blob --yes --key "${RELEASE_COSIGN_KEY}" \
      --bundle "${artifact}.sigstore.json" "${artifact}"
  done < <(find "${RELEASE_DIR}" -maxdepth 1 -type f ! -name '*.sigstore.json' | sort)
else
  echo "release: SKIP local signatures (set RELEASE_COSIGN_KEY; CI signs images keylessly)"
fi

if [[ "${RELEASE_BUILD_IMAGES:-false}" == "true" ]]; then
  require docker
  for component in controller api runner web; do
    image="${REGISTRY}/${component}"
    docker build -f "${ROOT_DIR}/build/${component}.Dockerfile" \
      --label "org.opencontainers.image.version=${VERSION_VALUE}" \
      --label "org.opencontainers.image.revision=${GIT_SHA}" \
      -t "${image}:${VERSION_VALUE}" -t "${image}:sha-${SHORT_SHA}" "${ROOT_DIR}"
    if [[ "${RELEASE_TAG_LATEST:-false}" == "true" ]]; then docker tag "${image}:${VERSION_VALUE}" "${image}:latest"; fi
    if [[ "${RELEASE_PUSH_IMAGES:-false}" == "true" ]]; then
      docker push "${image}:${VERSION_VALUE}"
      docker push "${image}:sha-${SHORT_SHA}"
      if [[ "${RELEASE_TAG_LATEST:-false}" == "true" ]]; then docker push "${image}:latest"; fi
    fi
  done
fi

echo "release: local artifacts ready at ${RELEASE_DIR}"
