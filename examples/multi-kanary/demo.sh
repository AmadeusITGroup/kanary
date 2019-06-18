#!/bin/bash

# Prerequisit
# 1- run the kind cluster       --> kind create cluster
# 2- install helm tiller        --> ./hack/install-helm-tiller.sh
# 3- install istio              --> ./hack/install-istio.sh
# 4- build the operator         --> make build && make TAG=latest KINDPUSH=true container
# 5- build the plugin           --> make build-plugin    // ensure that the produced binary is accessible in $PATH


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

desc "Create a dedicated namespace"
run "kubectl create ns multikanary; kubens multikanary"

desc "Install kanary crd"
run "kubectl apply -f deploy/crds/kanary_v1alpha1_kanarydeployment_crd.yaml"
desc "Install kanary operator"
run "for file in {service_account,role,role_binding,operator}; do kubectl apply -f deploy/\${file}.yaml; done"
wait_for_deployment "kanary"

desc "Deploy the application and service"
run "for file in {deployment,service}; do kubectl apply -f examples/multi-kanary/\${file}.yaml; done"
wait_for_deployment "myapp"

desc "Inventory of objects"
run "kubectl get all"

desc "Create a kanary named 'david' on top of 'my-app', with traffic=both"
run "kubectl kanary generate myapp --traffic=both --service=myapp-svc | jq '.metadata.name = \"david\"' | kubectl apply -f -"

desc "Create a kanary named 'cedric' on top of 'my-app', with traffic=both"
run "kubectl kanary generate myapp --traffic=both --service=myapp-svc | jq '.metadata.name = \"cedric\"' | kubectl apply -f -"

desc "Checking kanary deployments"
run "kubectl get kd"

wait_for_deployment "myapp-kanary-david"
wait_for_deployment "myapp-kanary-cedric"

desc "Checking kanary deployments with plugin"
run "kubectl kanary get"

desc "Inventory of objects"
run "kubectl get all"

desc "Checking the endpoints"
run "kubectl get ep"

kubectl proxy --port=8001&
proxyPID=$!
function killProxy {
  echo "Stopping kubeproxy"
  kill -9 ${proxyPID}
  cd "$CURRENT"
}
trap killProxy EXIT

desc "Starting kubctl proxy: pid=${proxyPID}"

desc "Let's check who is behind the myapp-svc"
run "for i in {1..20}; do curl 127.0.0.1:8001/api/v1/namespaces/multikanary/services/http:myapp-svc:80/proxy/host; sleep 0.5; echo; done"

desc "Let's check who is behind the myapp-svc-kanary-david"
run "for i in {1..10}; do curl 127.0.0.1:8001/api/v1/namespaces/multikanary/services/http:myapp-svc-kanary-david:80/proxy/host; sleep 0.5; echo; done"

desc "Let's check who is behind the myapp-svc-kanary-cedric"
run "for i in {1..10}; do curl 127.0.0.1:8001/api/v1/namespaces/multikanary/services/http:myapp-svc-kanary-cedric:80/proxy/host; sleep 0.5; echo; done"

desc "Let's remove the kanary 'david'"
run "kubectl delete kanary david"

desc "Let's check who is behind the myapp-svc"
run "for i in {1..15}; do curl 127.0.0.1:8001/api/v1/namespaces/multikanary/services/http:myapp-svc:80/proxy/host; sleep 0.5; echo; done"

desc "Inventory of objects"
run "kubectl get all"

desc "Let's remove the kanary 'cedric'"
run "kubectl delete kanary cedric"

desc "Inventory of objects"
run "kubectl get all"

desc "Inventory of objects"
run "kubectl get all"

desc "cleaning the system"
run "kubectl delete ns multikanary"
