#!/usr/bin/env bash
set -euo pipefail

NAMESPACE="${NAMESPACE:-cloudivision}"
RELEASE_NAME="${RELEASE_NAME:-cloudivision}"
IMAGE_REGISTRY="${IMAGE_REGISTRY:-ghcr.io/cloudivision}"
IMAGE_TAG="${IMAGE_TAG:-dev}"

if ! command -v helm >/dev/null 2>&1; then
  echo "helm is required to install cloudivision."
  exit 1
fi

if ! command -v kubectl >/dev/null 2>&1; then
  echo "kubectl is required to install cloudivision."
  exit 1
fi

helm upgrade --install "${RELEASE_NAME}" charts/cloudivision \
  --namespace "${NAMESPACE}" \
  --create-namespace \
  --set global.imageRegistry="${IMAGE_REGISTRY}" \
  --set controller.image.tag="${IMAGE_TAG}" \
  --set api.image.tag="${IMAGE_TAG}" \
  --set runner.image.tag="${IMAGE_TAG}" \
  --set web.image.tag="${IMAGE_TAG}" \
  --set controller.image.pullPolicy=IfNotPresent \
  --set api.image.pullPolicy=IfNotPresent \
  --set web.image.pullPolicy=IfNotPresent \
  --set api.defaultNamespace="${NAMESPACE}" \
  --set web.config.apiBaseUrl=""

kubectl -n "${NAMESPACE}" rollout status deploy/"${RELEASE_NAME}"-cloudivision-controller --timeout=120s
kubectl -n "${NAMESPACE}" rollout status deploy/"${RELEASE_NAME}"-cloudivision-api --timeout=120s
kubectl -n "${NAMESPACE}" rollout status deploy/"${RELEASE_NAME}"-cloudivision-web --timeout=120s

echo "cloudivision is installed in namespace ${NAMESPACE}"
echo "port-forward API: kubectl -n ${NAMESPACE} port-forward svc/${RELEASE_NAME}-cloudivision-api 8080:8080"
echo "port-forward web: kubectl -n ${NAMESPACE} port-forward svc/${RELEASE_NAME}-cloudivision-web 4200:80"
