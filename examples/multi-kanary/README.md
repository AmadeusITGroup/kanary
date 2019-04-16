# Multi-kanary

Demonstrate the ability to run multiple Kanary deployments at the same time on to of the same deployment

Creating a dedicated namespace:
```
kubectl create ns multikanary
kubens multikanary
```

Injecting deployment and service from multi-kanary example:
```
kubectl apply -f deployment.yaml
kubectl apply -f service.yaml
```

The application running is the deployment is an augmented version of httpbin. One of the endpoint /host, return the value of the variable HOSTNAME, in other word the name of the pod. This would be really usefull to check how the traffic is loadbalanced between instances.

Launching the 2 kanary in parallel:
```
kubectl kanary generate myapp --traffic=both --service=myapp-svc | jq '.metadata.name = "david"' | sed 's/Hello from /Hi from/' | kubectl apply -f -
kubectl kanary generate myapp --traffic=both --service=myapp-svc | jq '.metadata.name = "cedric"'   | sed 's/Hello from /Bonjour depuis/' | kubectl apply -f -
```

Checking the kanary crd 
```
> kubectl get kd
NAME     AGE
cedric   6m
david    6m
```

Checking the pods
```
> kubectl get pods -owide
NAME                                   READY   STATUS    RESTARTS   AGE     IP            NODE       NOMINATED NODE   READINESS GATES
kanary-67d5d66856-g6fkb                1/1     Running   0          7m4s    172.17.0.6    minikube   <none>           <none>
myapp-6dcb4464-2pcg5                   1/1     Running   0          20m     172.17.0.7    minikube   <none>           <none>
myapp-6dcb4464-gmblt                   1/1     Running   0          20m     172.17.0.8    minikube   <none>           <none>
myapp-6dcb4464-j9bbn                   1/1     Running   0          20m     172.17.0.9    minikube   <none>           <none>
myapp-kanary-cedric-75747d57b8-ghwnx   1/1     Running   0          6m53s   172.17.0.13   minikube   <none>           <none>
myapp-kanary-david-84958747f6-h6djl    1/1     Running   0          6m54s   172.17.0.12   minikube   <none>           <none>
```

Checking the services and endpoints
```
> kubectl get ep
NAME                      ENDPOINTS                                                       AGE
myapp-svc                 172.17.0.12:8080,172.17.0.13:8080,172.17.0.7:8080 + 2 more...   12m
myapp-svc-kanary-cedric   172.17.0.13:8080                                                7m37s
myapp-svc-kanary-david    172.17.0.12:8080                                                7m37s
```

Let's run a proxy in a dedicated terminal
```
kubectl proxy --port=8001
```

Now let's check what each service returns. First let's target the normal service multiple times:
```
> for i in {1..15}; do curl 127.0.0.1:8001/api/v1/namespaces/multikanary/services/http:myapp-svc:80/proxy/host; sleep 1; echo; done
myapp-kanary-david-d4675655f-x9npq
myapp-685f66fc4-f97gp
myapp-685f66fc4-bs2cg
myapp-kanary-david-d4675655f-x9npq
myapp-685f66fc4-f97gp
myapp-685f66fc4-9hjgc
myapp-kanary-david-d4675655f-x9npq
myapp-685f66fc4-bs2cg
myapp-685f66fc4-bs2cg
myapp-685f66fc4-9hjgc
myapp-685f66fc4-bs2cg
myapp-kanary-cedric-5447b95887-fg7k9
myapp-685f66fc4-bs2cg
myapp-kanary-david-d4675655f-x9npq
myapp-685f66fc4-bs2cg
```

We can see that the traffic is loadbalanced between all the available instance including the Kanary instances, because we used traffic=both

Now let's check that the specific Kanary service for 'david' kanary targets the good pod:
```
> for i in {1..15}; do curl 127.0.0.1:8001/api/v1/namespaces/multikanary/services/http:myapp-svc-kanary-david:80/proxy/host; sleep 1; echo; done
myapp-kanary-david-d4675655f-x9npq
myapp-kanary-david-d4675655f-x9npq
myapp-kanary-david-d4675655f-x9npq
myapp-kanary-david-d4675655f-x9npq
myapp-kanary-david-d4675655f-x9npq
myapp-kanary-david-d4675655f-x9npq
myapp-kanary-david-d4675655f-x9npq
myapp-kanary-david-d4675655f-x9npq
myapp-kanary-david-d4675655f-x9npq
myapp-kanary-david-d4675655f-x9npq
myapp-kanary-david-d4675655f-x9npq
myapp-kanary-david-d4675655f-x9npq
myapp-kanary-david-d4675655f-x9npq
myapp-kanary-david-d4675655f-x9npq
myapp-kanary-david-d4675655f-x9npq

```

Let's do the  same with 'cedric' kanary:
```
> for i in {1..15}; do curl 127.0.0.1:8001/api/v1/namespaces/multikanary/services/http:myapp-svc-kanary-cedric:80/proxy/host; sleep 1; echo; done
myapp-kanary-cedric-5447b95887-fg7k9
myapp-kanary-cedric-5447b95887-fg7k9
myapp-kanary-cedric-5447b95887-fg7k9
myapp-kanary-cedric-5447b95887-fg7k9
myapp-kanary-cedric-5447b95887-fg7k9
myapp-kanary-cedric-5447b95887-fg7k9
myapp-kanary-cedric-5447b95887-fg7k9
myapp-kanary-cedric-5447b95887-fg7k9
myapp-kanary-cedric-5447b95887-fg7k9
myapp-kanary-cedric-5447b95887-fg7k9
myapp-kanary-cedric-5447b95887-fg7k9
myapp-kanary-cedric-5447b95887-fg7k9
myapp-kanary-cedric-5447b95887-fg7k9
myapp-kanary-cedric-5447b95887-fg7k9
myapp-kanary-cedric-5447b95887-fg7k9
```
