#!/bin/bash

# Prerequisit
# 1- run the kind cluster       --> kind create cluster
# 2- install helm tiller        --> ./hack/install-helm-tiller.sh
# 3- install istio              --> ./hack/install-istio.sh
# 4- build the operator         --> make build && make TAG=latest KINDPUSH=true container
# 5- build the simple-server    --> make KINDPUSH=true simple-server

DEMO_DIR="$(cd "$(dirname "${0}")" && pwd)"
CURRENT=$PWD

cd ${DEMO_DIR}/../..

ROOTDIR=$PWD

. script/demo-utils.sh 

function wait_for_deployment() {
    deployment=${1}
    desc "waiting for deployment: ${deployment}"
    until [ $(kubectl get deployment ${deployment} -ojsonpath="{.status.conditions[?(@.type=='Available')].status}") == "True" ] > /dev/null 2>&1; do sleep 1; kubectl get deployment ${deployment}; done
#    kubectl wait --for=condition=available --timeout=600s deployment/${deployment}
}

NAMESPACE="prom-istio-example"

#Cleanup script when demo exit
function cleanup() {
  desc "cleaning the namespace"
  run "kubectl delete ns $NAMESPACE"  
  desc "Stopping PortForward"
  run "kill -9 ${portforwardPID}"
  cd "$CURRENT"
}
trap cleanup EXIT

#Starting the demo
desc "Create a dedicated namespace"
run "kubectl create ns $NAMESPACE; kubectl label namespace $NAMESPACE istio-injection=enabled; kubens $NAMESPACE"

desc "Install kanary crd"
run "kubectl apply -f deploy/crds/kanary_v1alpha1_kanarydeployment_crd.yaml"
desc "Install kanary operator"
run "for file in {service_account,role,role_binding,operator}; do kubectl apply -f deploy/\${file}.yaml; done"
wait_for_deployment "kanary"

desc "Deploy the application and service"
run "for file in {deployment,service}; do kubectl apply -f examples/istio-prom/\${file}.yaml; done"
wait_for_deployment "myapp"

desc "Exposing service over istio ingress"
run "for file in {gateway,virtualservice}; do kubectl apply -f examples/istio-prom/\${file}.yaml; done"

LOCAL80=30080
desc "running port forwarder to kind cluster $LOCAL80:80"
kubectl port-forward -n istio-system service/istio-ingressgateway "$LOCAL80:80" > /dev/null 2>&1 &
portforwardPID=$!

desc "Inventory of objects"
run "kubectl get all"

desc "Let's check that service myapp-svc replies"
run "for i in {1..20}; do curl -HHost:myapp.example.com 127.0.0.1:$LOCAL80/host; echo; sleep 0.1; done"

desc "Create a kanary with traffic=both and new version v2"
run "kubectl kanary generate myapp --traffic=both --service=myapp-svc --validation-period=1m | jq '.spec.template.spec.template.metadata.labels.version = \"v2\"' |kubectl apply -f -"

desc "Checking kanary deployments"
run "kubectl get kd"

wait_for_deployment "myapp-kanary-myapp"

desc "Checking kanary deployments with plugin"
run "kubectl kanary get"

desc "Inventory of objects"
run "kubectl get all"

desc "Checking the endpoints"
run "kubectl get ep"

desc "Inventory of objects"
run "kubectl get all"

# using version
# histogram_quantile(0.6, sum(irate(istio_request_duration_seconds_bucket{reporter="destination",destination_service=~"myapp-svc.*"}[10s])) by (destination_version, le))
# using workload name
# histogram_quantile(0.6, sum(irate(istio_request_duration_seconds_bucket{reporter="destination",destination_service=~"myapp-svc.*"}[10s])) by (destination_workload, le))
