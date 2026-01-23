#!/usr/bin/env bash
set -euo pipefail

NAMESPACE="${NAMESPACE:-dev}"
IMG="${IMG:-controller:latest}"
KIND_CLUSTER="${KIND_CLUSTER:-kubebuilder}"

for cmd in kubectl helm make docker; do
  if ! command -v "${cmd}" >/dev/null 2>&1; then
    echo "ERROR: missing dependency: ${cmd}"
    exit 1
  fi
done

if command -v kind >/dev/null 2>&1; then
  if ! kind get clusters | grep -q "^${KIND_CLUSTER}$"; then
    echo "==> Creating kind cluster: ${KIND_CLUSTER}"
    kind create cluster --name "${KIND_CLUSTER}" --config kind-config.yaml
  fi
  if kubectl config get-contexts "kind-${KIND_CLUSTER}" >/dev/null 2>&1; then
    kubectl config use-context "kind-${KIND_CLUSTER}" >/dev/null 2>&1 || true
  fi
else
  echo "ERROR: kind not found; this script expects a kind cluster"
  exit 1
fi

echo "==> Namespace: ${NAMESPACE}"
if kubectl get namespace "${NAMESPACE}" >/dev/null 2>&1; then
  phase="$(kubectl get namespace "${NAMESPACE}" -o jsonpath='{.status.phase}')"
  if [ "${phase}" = "Terminating" ]; then
    echo "ERROR: namespace ${NAMESPACE} is Terminating. Finish deletion before deploy."
    exit 1
  fi
else
  kubectl create namespace "${NAMESPACE}"
fi

echo "==> Build and deploy operator"
make -C static-content-operator install
make -C static-content-operator docker-build IMG="${IMG}"

kind load docker-image "${IMG}" --name "${KIND_CLUSTER}"

make -C static-content-operator deploy IMG="${IMG}"
kubectl patch deployment/static-website-controller-manager -n static-website-system \
  --type='json' \
  -p='[{"op":"replace","path":"/spec/template/spec/containers/0/imagePullPolicy","value":"IfNotPresent"}]'
kubectl rollout restart deployment/static-website-controller-manager -n static-website-system
if ! kubectl rollout status deployment/static-website-controller-manager -n static-website-system --timeout=180s; then
  echo "ERROR: controller-manager not ready; describing pod for details"
  kubectl get pods -n static-website-system
  kubectl describe pod -n static-website-system -l control-plane=controller-manager
  exit 1
fi
kubectl apply -f static-content-operator/config/samples/static_v1_staticsite.yaml -n "${NAMESPACE}"

echo "==> Install Prometheus"
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update
helm upgrade --install kps prometheus-community/kube-prometheus-stack -n "${NAMESPACE}" \
  --set prometheus.service.type=NodePort \
  --set prometheus.service.nodePort=30900

echo "==> Apply ServiceMonitor"
kubectl apply -f prometheus/pthConfig.yml -n "${NAMESPACE}"

echo "==> Install Prometheus Adapter"
helm upgrade --install prom-adapter prometheus-community/prometheus-adapter -n "${NAMESPACE}" \
  --set prometheus.url="http://kps-kube-prometheus-stack-prometheus.${NAMESPACE}.svc" \
  --set prometheus.port=9090

echo "==> Apply Adapter config + restart"
kubectl apply -f hpa/adapter-config.yml -n "${NAMESPACE}"
kubectl rollout restart deployment/prom-adapter-prometheus-adapter -n "${NAMESPACE}"

echo "==> Apply HPA"
kubectl apply -f hpa/hpa.yml -n "${NAMESPACE}"

echo "==> Done"
