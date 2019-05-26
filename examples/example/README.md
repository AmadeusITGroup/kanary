# Nginx Kanary Demo

## Prerequisites
- Minikube installation
- Docker to build image locally
- Operator-sdk binary in path

## Setup
```
minikube start
eval $(minikube docker-env)
```

## Build Kanary operator
```
mkdir -p $GOPATH/src/github.com/amadeusitgroup/
cd $GOPATH/src/github.com/amadeusitgroup/
```

TODO: *1A note : don't forget your username@*
TODO: *replace with github ?*
TODO: *check if v0.0.1 is what we want, maybe need to change it in deploy/operator.yaml*
```
git clone https://aorlinski@rndwww.nce.amadeus.net/git/scm/op1a/kanary.git

cd $GOPATH/src/github.com/amadeusitgroup/kanary/

make TAG=v0.0.1 container
```

## Build Kanary kubectl plugin
```
make build-plugin
export PATH="$(pwd):$PATH"
```

## Deploy Kanary Operator
```
kubectl apply -f deploy/crds/kanary_v1alpha1_kanarydeployment_crd.yaml
kubectl apply -f deploy/service_account.yaml
kubectl apply -f deploy/role.yaml
kubectl apply -f deploy/role_binding.yaml
kubectl apply -f deploy/operator.yaml
```

## Demo : Deploy a nginx
```
kubectl apply -f examples/example/nginx_service.yaml
kubectl apply -f examples/example/nginx_deployment.yaml
```

## Demo : Make a canary deployment for nginx
The aim of this kanarydeployment is to update from version 1.15.4 of the image to latest
```
kubectl apply -f examples/example/nginx_kanarydeployment.yaml
```

## Demo : current status
```
>kubectl get pods
NAME                                READY     STATUS    RESTARTS   AGE
kanary-fd5c94d74-v49f8              1/1       Running   0          9m
nginx-dep-75bd58f5c7-gts2j          1/1       Running   0          5m
nginx-dep-75bd58f5c7-m5fct          1/1       Running   0          5m
nginx-dep-75bd58f5c7-n8pbh          1/1       Running   0          5m
nginx-dep-kanary-5476b5d6b9-sklr7   1/1       Running   0          1m

>kubectl kanary get
NAMESPACE  NAME       STATUS                          DEPLOYMENT  SERVICE  SCALE   TRAFFIC  VALIDATION
  default    nginx-dep  Running                         nginx-dep   nginx    static  both     manual
                        (1h24m27.094786s/15m0s)
```

We can see the canary deployment has the latest image instead of 1.15.4
Do whatever manual checks you need on this pod

```
>kubectl get pods -o=custom-columns="NAME:.metadata.name,IMAGE:.spec.containers[0].image"
NAME                                IMAGE
kanary-fd5c94d74-v49f8              kanaryoperator/operator:v0.0.1
nginx-dep-75bd58f5c7-gts2j          nginx:1.15.4
nginx-dep-75bd58f5c7-m5fct          nginx:1.15.4
nginx-dep-75bd58f5c7-n8pbh          nginx:1.15.4
nginx-dep-kanary-5476b5d6b9-sklr7   nginx:latest
```

## Demo : need to take action now to validate / invalidate kanary as this is manual validation

Set the status to valid so all the pods get updated using the deployment strategy defined
```
kubectl patch kanarydeployment nginx-dep --type=merge -p '{"spec":{"validation":{"manual":{"statusAfterDeadline":"valid"}}}}'
```

Check status
```
>kubectl get pods -o=custom-columns="NAME:.metadata.name,IMAGE:.spec.containers[0].image"
NAME                                IMAGE
kanary-fd5c94d74-v49f8              kanary/operator:v0.0.1
nginx-dep-6574bd76c-8ssm5           nginx:latest
nginx-dep-6574bd76c-ld2jm           nginx:latest
nginx-dep-6574bd76c-rvbxc           nginx:latest
nginx-dep-kanary-5476b5d6b9-sklr7   nginx:latest
```

And the kanary pod does not receive traffic anymore, but stays alive for further analysis

```
kubectl delete kanarydeployment nginx-dep
```

will delete kanary deployment and cascade delete the canary pod

## Demo : invalidate the kanary instead
Step back to version 1.15.4
```
kubectl delete deployment nginx-dep
kubectl apply -f examples/example/nginx_deployment.yaml
```

Start kanary deployment
```
kubectl apply -f examples/example/nginx_kanarydeployment.yaml
```

Invalidate kanary so nothing happens on existing pods, and the kanary does not receive traffic anymore, but stays for further investigation
```
kubectl patch kanarydeployment nginx-dep --type=merge -p '{"spec":{"validation":{"manual":{"statusAfterDeadline":"invalid"}}}}'
```

Delete kanarydeployment will also delete kanary pod
```
kubectl delete kanarydeployment nginx-dep
```

Check status
```
>get pods -o=custom-columns="NAME:.metadata.name,IMAGE:.spec.containers[0].image"
NAME                         IMAGE
kanary-fd5c94d74-v49f8       kanaryoperator/operator:v0.0.1
nginx-dep-75bd58f5c7-ddwqn   nginx:1.15.4
nginx-dep-75bd58f5c7-lf75x   nginx:1.15.4
nginx-dep-75bd58f5c7-vmzlf   nginx:1.15.4
```







