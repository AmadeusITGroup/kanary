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
  desc "Cleanup"
  desc "kill -9 ${portforwardPID} #Stopping PortForward"
  kill -9 ${portforwardPID} 
  desc "kill -9 ${injectionPID} #Stopping injection"
  kill -9 ${injectionPID} 
  desc "kubectl delete ns $NAMESPACE #cleaning namespace"
  kubectl delete ns $NAMESPACE
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

function injection() {
  while true; do curl -HHost:myapp.example.com 127.0.0.1:$LOCAL80/host > /dev/null 2>&1; sleep 0.05; done
}

desc "Let's run injection"
injection &
injectionPID=$!

desc "Open Grafana Istio Dashboard" 
run "# http://127.0.0.1:$LOCAL80  + Host modification plugin to match grafana.example.com"

desc "Create a kanary with traffic=both and new version with response time degradation"
run "kubectl kanary generate myapp --traffic=both --service=myapp-svc --validation-period=1m --validation-promql-istio-quantile=\"P99<310\" | jq '(.spec.template.spec.template.spec.containers[0].args[0]) |= \"--responseTime=5:300,50:100,100:80\"' | kubectl apply -f -"

desc "Monitoring the kanary deployments... till it fails"
run "watch kubectl kanary get"

desc "All object are still there for investigation!!!"
run "kubectl get all"

desc "Let's remove the kanary"
run "kubectl delete kanary myapp"

desc "Inventory of objects"
run "kubectl get all"

desc "Create a kanary with traffic=both and new version with correct response time"
run "kubectl kanary generate myapp --traffic=both --service=myapp-svc --validation-period=1m --validation-promql-istio-quantile=\"P99<310\" | jq '(.spec.template.spec.template.spec.containers[0].args[0]) |= \"--responseTime=5:100,50:50,100:10\"' | kubectl apply -f -"

desc "Monitoring the kanary deployments... till success and rollout"
run "watch kubectl kanary get"

desc "Inventory of objects"
run "kubectl get all"

desc "Let's remove the kanary"
run "kubectl delete kanary myapp"

desc "What about doing 2 kanaries at the same time?"
run "kubectl kanary generate myapp --name=cedric --traffic=both --service=myapp-svc --validation-period=1m --validation-promql-istio-quantile=\"P99<310\" | jq '(.spec.template.spec.template.spec.containers[0].args[0]) |= \"--responseTime=5:300,50:100,100:80\"' | kubectl apply -f -"
run "kubectl kanary generate myapp --name=david --traffic=both --service=myapp-svc --validation-period=1m --validation-promql-istio-quantile=\"P99<310\" | jq '(.spec.template.spec.template.spec.containers[0].args[0]) |= \"--responseTime=5:100,50:50,100:10\"' | kubectl apply -f -"

desc "Monitoring the kanaries"
run "watch kubectl kanary get"



# using version
# histogram_quantile(0.6, sum(irate(istio_request_duration_seconds_bucket{reporter="destination",destination_service=~"myapp-svc.*"}[10s])) by (destination_version, le))
# using workload name
# histogram_quantile(0.6, sum(irate(istio_request_duration_seconds_bucket{reporter="destination",destination_service=~"myapp-svc.*"}[10s])) by (destination_workload, le))
