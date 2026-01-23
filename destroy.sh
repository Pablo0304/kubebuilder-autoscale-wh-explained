#!/usr/bin/env bash
set -euo pipefail

NAMESPACE="${NAMESPACE:-dev}"
KIND_CLUSTER="${KIND_CLUSTER:-kubebuilder}"

echo "==> Delete HPA and adapter config"
kubectl delete -f hpa/hpa.yml -n "${NAMESPACE}" --ignore-not-found
kubectl delete -f hpa/adapter-config.yml -n "${NAMESPACE}" --ignore-not-found

echo "==> Uninstall Prometheus Adapter"
helm uninstall prom-adapter -n "${NAMESPACE}" || true

echo "==> Delete ServiceMonitor"
kubectl delete -f prometheus/pthConfig.yml -n "${NAMESPACE}" --ignore-not-found

echo "==> Uninstall Prometheus stack"
helm uninstall kps -n "${NAMESPACE}" || true

echo "==> Delete sample StaticSite"
kubectl delete -f static-content-operator/config/samples/static_v1_staticsite.yaml -n "${NAMESPACE}" --ignore-not-found

echo "==> Undeploy operator"
make -C static-content-operator undeploy ignore-not-found=true
make -C static-content-operator uninstall ignore-not-found=true

if command -v kind >/dev/null 2>&1; then
  echo "==> Delete kind cluster: ${KIND_CLUSTER}"
  kind delete cluster --name "${KIND_CLUSTER}" || true
else
  echo "WARN: kind not found; skipping cluster deletion"
fi

echo "==> Done"
