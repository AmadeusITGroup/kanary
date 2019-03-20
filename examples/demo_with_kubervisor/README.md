# Demo with Kubervisor

Aim of the demo is to use the kubervisor demo app (flight search app), and try to canary it.

## env

Kubernetes 1.9.4

## Init

```shell
export DEMO_PATH=/tmp/demo
mkdir $DEMO_PATH && cd $DEMO_PATH
export KUBERVISOR_PATH=$DEMO_PATH/src/github.com/amadeusitgroup/kubervisor
export KANARY_PATH=$DEMO_PATH/src/github.com/amadeusitgroup/kanary
export GOPATH=$DEMO_PATH
mkdir -p $DEMO_PATH/src/github.com/amadeusitgroup/
git clone https://github.com/amadeusitgroup/kubervisor $KUBERVISOR_PATH
git clone https://rndwww.nce.amadeus.net/git/scm/op1a/kanary.git $KANARY_PATH
minikube start --memory=4096 --kubernetes-version=v1.9.4
eval $(minikube docker-env)
```

start Kubervisor demo initialization

```shell
cd $KUBERVISOR_PATH
make plugin
git tag latest
make container
cd $GOPATH/src/github.com/amadeusitgroup/kubervisor/examples/demo
./scripts/init.sh
```

then deploy the kanary controller

```shell
cd $KANARY_PATH
make TAG=stable container
helm install --wait -n kanary chart/kanary
make build-plugin
export PATH="$(pwd):$PATH"
```

## scenario

- Start the Kubervisor Demo app, with injector
- Start the KubervisorService pricer-kubervisorservice_3.yaml (don't update the service)
- Generate a KanaryDeployment from the Pricer deployment thanks to the kubectl kanary plugin: (scale=static;traffic=service;validation:labelwath-pod)
- Update the Deployment configuration by changing the container parameter: "--km-price"
- Create the KanaryDeployment.
- Wait en see what append.

## Run

```shell
# run in a terminal readprice.sh to generate some traffic
$KUBERVISOR_PATH/examples/demo/scripts/readprice.sh

# open the grafana dashboard http://grafana.demo.mk
open http://grafana.demo.mk

# create the KubervisorService (pause strategy)
kubectl create -f $KUBERVISOR_PATH/examples/demo/scripts/pricer-kubervisorservice_3.yaml

# check the dashboard to see that the Kubervisor manage properly the pricer-1a deployment.

# generate now the KanaryDeployment (dry-run to see the spec)
kubectl kanary generate prod-pricer-1a --service pricer-1a -o yaml --validation-labelwatch-pod "kubervisor/traffic=pause" --validation-period 3m --traffic both


# then create the KanaryDeployment resource
kubectl kanary generate prod-pricer-1a --service pricer-1a -o yaml --validation-labelwatch-pod "kubervisor/traffic=pause" --validation-period 3m --traffic both > kanaryDeployment_1.yaml

# edit the Deployment template (change --km-price=1 to --km-price=0)
code kanaryDeployment_1.yaml

# create the resource
kubectl create -f kanaryDeployment_1.yaml

# watch the status and dashboard
watch -n1 kubectl kanary get

# wait until failure
kubectl delete -f kanaryDeployment_1.yaml

## New KanaryDeployment
kubectl kanary generate prod-pricer-1a -o yaml --validation-labelwatch-pod "kubervisor/traffic=pause" --validation-period 3m > kanaryDeployment_2.yaml

# edit the Deployment template (change --rand-price=30 to --rand-price=25)
code kanaryDeployment_2.yaml

# create the resource
kubectl create -f kanaryDeployment_2.yaml

# watch the status and dashboard
watch -n1 kubectl kanary get

# wait until success

# clear
kubectl delete -f kanaryDeployment_2.yaml

```
