#!/bin/bash
set -x
set -o errexit
set -o nounset
set -o pipefail

ISTIO_VER="1.1.8"
REPO_ROOT=$(git rev-parse --show-toplevel)
helm repo add istio.io https://storage.googleapis.com/istio-release/releases/${ISTIO_VER}/charts
helm upgrade -i istio-init istio.io/istio-init --wait --namespace istio-system
kubectl -n istio-system wait --timeout=120s --for=condition=complete job/istio-init-crd-10
kubectl -n istio-system wait --timeout=120s --for=condition=complete job/istio-init-crd-11
helm upgrade -i istio istio.io/istio --wait --namespace istio-system -f ${REPO_ROOT}/hack/istio/istio-values.yaml || true
#expose grafana and prometheus over the gateway
kubectl apply -f ${REPO_ROOT}/hack/istio/gateway.grafana.yaml
kubectl apply -f ${REPO_ROOT}/hack/istio/gateway.prometheus.yaml
kubectl apply -f ${REPO_ROOT}/hack/istio/virtualservice.grafana.yaml
kubectl apply -f ${REPO_ROOT}/hack/istio/virtualservice.prometheus.yaml
kubectl -n istio-system get all
