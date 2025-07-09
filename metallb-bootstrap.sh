#!/usr/bin/env bash
set -euo pipefail

# ----- config: edit to match your IP range ---------
POOL_CIDR="192.168.8.210-192.168.8.215"
METALLB_VERSION="0.15.2"
NAMESPACE="metallb-system"
# ----------------------------------------------

echo ">>> Adding MetalLB Helm repo"
helm repo add metallb https://metallb.github.io/metallb
helm repo update

echo ">>> Installing MetalLB ${METALLB_VERSION}"
helm upgrade --install metallb metallb/metallb \
  --version "${METALLB_VERSION}" \
  --namespace "${NAMESPACE}" --create-namespace \
  --set crds.create=true

echo ">>> Waiting for MetalLB pods to be Ready"
kubectl wait --namespace "${NAMESPACE}" \
  --for=condition=Ready pod \
  --selector=app.kubernetes.io/instance=metallb \
  --timeout=300s

echo ">>> Creating IPAddressPool and L2Advertisement"
cat <<EOF | kubectl apply -f -
apiVersion: metallb.io/v1beta1
kind: IPAddressPool
metadata:
  name: default-pool
  namespace: ${NAMESPACE}
spec:
  addresses:
    - ${POOL_CIDR}
---
apiVersion: metallb.io/v1beta1
kind: L2Advertisement
metadata:
  name: default
  namespace: ${NAMESPACE}
spec:
  ipAddressPools:
    - default-pool
EOF

echo ">>> Verifying resources"
kubectl get ipaddresspools -n "${NAMESPACE}"
kubectl get l2advertisements -n "${NAMESPACE}"
echo ">>> MetalLB bootstrap complete"