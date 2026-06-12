#!/usr/bin/env bash
set -euo pipefail

CLUSTER_NAME="${CLUSTER_NAME:-cloudivision-dev}"
REGISTRY_NAME="${REGISTRY_NAME:-cloudivision-registry}"
REGISTRY_PORT="${REGISTRY_PORT:-5001}"

if ! command -v kind >/dev/null 2>&1; then
  echo "kind is required. Install it from https://kind.sigs.k8s.io/"
  exit 1
fi

if ! command -v docker >/dev/null 2>&1; then
  echo "docker is required for kind clusters."
  exit 1
fi

if ! docker inspect "${REGISTRY_NAME}" >/dev/null 2>&1; then
  docker run -d --restart=always -p "127.0.0.1:${REGISTRY_PORT}:5000" --name "${REGISTRY_NAME}" registry:2 >/dev/null
fi

if ! kind get clusters | grep -qx "${CLUSTER_NAME}"; then
  config_file="$(mktemp)"
  trap 'rm -f "${config_file}"' EXIT
  cat >"${config_file}" <<EOF
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
containerdConfigPatches:
  - |-
    [plugins."io.containerd.grpc.v1.cri".registry.mirrors."localhost:${REGISTRY_PORT}"]
      endpoint = ["http://${REGISTRY_NAME}:5000"]
nodes:
  - role: control-plane
EOF
  kind create cluster --name "${CLUSTER_NAME}" --config "${config_file}"
else
  echo "kind cluster ${CLUSTER_NAME} already exists"
fi

if ! docker network inspect kind --format '{{json .Containers}}' | grep -q "\"${REGISTRY_NAME}\""; then
  docker network connect kind "${REGISTRY_NAME}" >/dev/null
fi

kubectl apply -f - <<EOF
apiVersion: v1
kind: ConfigMap
metadata:
  name: local-registry-hosting
  namespace: kube-public
data:
  localRegistryHosting.v1: |
    host: "localhost:${REGISTRY_PORT}"
    help: "https://kind.sigs.k8s.io/docs/user/local-registry/"
EOF

echo "kind cluster ${CLUSTER_NAME} is ready"
echo "local registry is available at localhost:${REGISTRY_PORT}"
