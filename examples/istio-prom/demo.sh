#!/bin/bash

# Prerequisit
# 1- run the kind cluster       --> kind create cluster
# 2- install helm tiller        --> ./hack/install-helm-tiller.sh
# 3- install istio              --> ./hack/install-istio.sh
# 4- build the operator         --> make build && make TAG=latest KINDPUSH=true container
# 5- build the plugin           --> make build-plugin    // ensure that the produced binary is accessible in $PATH
# 6- build the simple-server    --> make KINDPUSH=true simple-server
# 7- build the reverse-proxy    --> make reverse-proxy

#Local port being used for the demo
GRAFANA_REVERSE_PROXY=8080
LOCAL80=30080

DEMO_DIR="$(cd "$(dirname "${0}")" && pwd)"
CURRENT=$PWD

cd "${DEMO_DIR}"/../..

ROOTDIR=$PWD

. script/demo-utils.sh 

function wait_for_deployment() {
    deployment=${1}
    desc "waiting for deployment: ${deployment}"
    until [ $(kubectl get deployment ${deployment} -ojsonpath="{.status.conditions[?(@.type=='Available')].status}") == "True" ] > /dev/null 2>&1; do sleep 3; kubectl get deployment ${deployment}; done
#    kubectl wait --for=condition=available --timeout=600s deployment/${deployment}
}

function kanaryStatus() {
  echo $(kubectl get kanary "${1}" -ojsonpath="{.status.report.status}")
}

function kanaryCompleted() {
  local status=$(kanaryStatus "${1}")
  if [ "${status}" == "Failed" ] || [ "${status}" == "Succeeded" ] || [ "${status}" == "DeploymentUpdated" ]; then echo "true"; else echo "false"; fi
}

function wait_for_kanary() {
    kanary=${1}
    local kanaryWatch=${1}
    if [ "${2}" == "--getAll" ]; then kanaryWatch=""; fi
    desc "waiting for kanary to complete: ${kanary}"
    until [ $(kanaryCompleted ${kanary}) == "true" ] > /dev/null 2>&1; do sleep 1; kubectl kanary get ${kanaryWatch}; done
    kubectl kanary get ${kanaryWatch}
}

NAMESPACE="prom-istio-example"

#Cleanup script when demo exit
function cleanup() {
  desc "Cleanup"
  desc "kill -9 ${portforwardPID} #Stopping PortForward"
  kill -9 ${portforwardPID}
  desc "kill -9 ${reverseproxyPID} #Stopping ReverseProxy"
  kill -9 ${reverseproxyPID}
  desc "kill -9 ${injectionPID} #Stopping injection"
  kill -9 ${injectionPID} 
  desc "kubectl delete ns $NAMESPACE #cleaning namespace"
  kubectl delete ns $NAMESPACE
  cd "$CURRENT"
}
trap cleanup EXIT

kubectl delete pod -n istio-system -l app=prometheus > /dev/null 2>&1 &
kubectl delete pod -n istio-system -l app=telemetry > /dev/null 2>&1 &

desc "running port forwarder to kind cluster $LOCAL80:80"
kubectl port-forward -n istio-system service/istio-ingressgateway "$LOCAL80:80" > /dev/null 2>&1 &
portforwardPID=$!
desc "running reverse proxy on localhost:$GRAFANA_REVERSE_PROXY to target grafana.example.com"
$ROOTDIR/bin/reverse-proxy -svc grafana.example.com -port "$GRAFANA_REVERSE_PROXY" > /dev/null 2>&1 &
reverseproxyPID=$!

#Starting the demo
desc "Create a dedicated namespace"
DEMO_AUTO_RUN=1 run "kubectl create ns $NAMESPACE; kubectl label namespace $NAMESPACE istio-injection=enabled; kubens $NAMESPACE"

desc "Install kanary crd"
DEMO_AUTO_RUN=1 run "kubectl apply -f deploy/crds/kanary_v1alpha1_kanarydeployment_crd.yaml"
desc "Install kanary operator"
DEMO_AUTO_RUN=1 run "for file in {service_account,role,role_binding,operator}; do kubectl apply -f deploy/\${file}.yaml; done"
wait_for_deployment "kanary"

desc "Deploy the application and service"
DEMO_AUTO_RUN=1 run "for file in {deployment,service}; do kubectl apply -f examples/istio-prom/\${file}.yaml; done"
wait_for_deployment "myapp"

desc "Exposing service over istio ingress"
DEMO_AUTO_RUN=1 run "for file in {gateway,virtualservice}; do kubectl apply -f examples/istio-prom/\${file}.yaml; done"

desc "Inventory of objects"
run "kubectl get all"

function injection() {
  while true; do curl -HHost:myapp.example.com 127.0.0.1:$LOCAL80/host > /dev/null 2>&1; sleep 0.05; done
}

desc "Let's run injection"
injection &
injectionPID=$!

desc "Open Grafana Istio Dashboard" 
DEMO_AUTO_RUN=1 run "# http://127.0.0.1:$GRAFANA_REVERSE_PROXY"

desc "Create a kanary with traffic=both and new version with response time degradation"
desc "Let's look at the command, using kubectl plugin"
run "# kubectl kanary generate myapp --name=batman --traffic=both --service=myapp-svc --validation-period=1m --validation-promql-istio-quantile=\"P99<310\""
desc "Let's look at the create resource"
run "kubectl kanary generate myapp --name=batman --traffic=both --service=myapp-svc --validation-period=1m --validation-promql-istio-quantile=\"P99<310\" --output=yaml | less"
desc "Now let's inject it"
DEMO_AUTO_RUN=1 run "kubectl kanary generate myapp --name=batman --traffic=both --service=myapp-svc --validation-period=1m --validation-promql-istio-quantile=\"P99<310\" | jq '(.spec.template.spec.template.spec.containers[0].args[0]) |= \"--responseTime=10:300,50:100,100:80\"' | kubectl apply -f -"

desc "Monitoring the kanary..."
wait_for_kanary batman

desc "The Kanary failed, but all objects are still there for investigation!!!"
run "kubectl get all"

desc "Let's remove the kanary"
DEMO_AUTO_RUN=1 run "kubectl delete kanary batman"

desc "Inventory of objects"
run "kubectl get all"

desc "Create a kanary with traffic=both and new version with correct response time"
DEMO_AUTO_RUN=1 run "kubectl kanary generate myapp --name=hulk --traffic=both --service=myapp-svc --validation-period=1m --validation-promql-istio-quantile=\"P99<310\" | jq '(.spec.template.spec.template.spec.containers[0].args[0]) |= \"--responseTime=5:90,50:10,100:0\"' | kubectl apply -f -"

desc "Monitoring the kanary deployments..."
wait_for_kanary hulk

desc "Inventory of objects"
run "kubectl get all"

desc "Let's remove the kanary"
DEMO_AUTO_RUN=1 run "kubectl delete kanary hulk"

desc "What about doing multiple (3) kanaries at the same time?"
DEMO_AUTO_RUN=1 run "kubectl kanary generate myapp --name=thor --traffic=both --service=myapp-svc --validation-period=1m --validation-promql-istio-quantile=\"P99<310\" --validation-promql-istio-success=\"0.95\" --dry-run | jq '(.spec.template.spec.template.spec.containers[0].args[0]) |= \"--responseTime=10:100,50:50,100:10\"' | kubectl apply -f -"
DEMO_AUTO_RUN=1 run "kubectl kanary generate myapp --name=superman --traffic=both --service=myapp-svc --validation-period=1m --validation-promql-istio-quantile=\"P99<310\" --validation-promql-istio-success=\"0.95\" --dry-run | jq '(.spec.template.spec.template.spec.containers[0].args[0]) |= \"--responseTime=10:450,100:80\"' | kubectl apply -f -"
DEMO_AUTO_RUN=1 run "kubectl kanary generate myapp --name=wonderwoman --traffic=both --service=myapp-svc --validation-period=1m --validation-promql-istio-quantile=\"P99<310\" --validation-promql-istio-success=\"0.95\" --dry-run | jq '(.spec.template.spec.template.spec.containers[0].args[0]) |= \"--responseCode=10:500\"' | kubectl apply -f -"

desc "Monitoring the kanaries"
wait_for_kanary thor --getAll

desc "Inventory of objects"
run "kubectl get all"
