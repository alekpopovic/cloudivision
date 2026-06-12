#!/usr/bin/env bash
set -euo pipefail

CLUSTER_NAME="${CLUSTER_NAME:-cloudivision-dev}"
IMAGE_REGISTRY="${IMAGE_REGISTRY:-ghcr.io/cloudivision}"
IMAGE_TAG="${IMAGE_TAG:-dev}"

if ! command -v docker >/dev/null 2>&1; then
  echo "docker is required to build images."
  exit 1
fi

if ! command -v kind >/dev/null 2>&1; then
  echo "kind is required to load images."
  exit 1
fi

make IMAGE_REGISTRY="${IMAGE_REGISTRY}" IMAGE_TAG="${IMAGE_TAG}" docker-build-controller
make IMAGE_REGISTRY="${IMAGE_REGISTRY}" IMAGE_TAG="${IMAGE_TAG}" docker-build-api
make IMAGE_REGISTRY="${IMAGE_REGISTRY}" IMAGE_TAG="${IMAGE_TAG}" docker-build-runner
make IMAGE_REGISTRY="${IMAGE_REGISTRY}" IMAGE_TAG="${IMAGE_TAG}" docker-build-web

kind load docker-image --name "${CLUSTER_NAME}" "${IMAGE_REGISTRY}/controller:${IMAGE_TAG}"
kind load docker-image --name "${CLUSTER_NAME}" "${IMAGE_REGISTRY}/api:${IMAGE_TAG}"
kind load docker-image --name "${CLUSTER_NAME}" "${IMAGE_REGISTRY}/runner:${IMAGE_TAG}"
kind load docker-image --name "${CLUSTER_NAME}" "${IMAGE_REGISTRY}/web:${IMAGE_TAG}"

echo "loaded cloudivision images into kind cluster ${CLUSTER_NAME}"
