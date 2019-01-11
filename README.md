Kanary
======


## How to use it

### Simple strategy

As indicated by its name, this strategy aims to be the simplest strategy with some manual action to move between the different canary deployment steps.

It is also the default strategy if any other strategy has been configured in the ```KanaryDeployment.Spec.Strategy```.

the simple strategy step is configured thank the `mode` field, and can have 4 different values:

- `deactivate`: corresponds to the mode where the canary deployment is deactivated which means the replicas value is set to 0.
- `activate`: corresponds to the mode where the canary deployment is activated with the replicas value present in the strategy. If the value is not set, the replicas value is equal to the default replicas deployment value that is `1`.
- `pause`: corresponds to the mode where any modification in the strategy will not be applied.
- `apply`: corresponds to the last mode where the deployment update is propagated to the `principal` deployment managed by the KanaryDeployment. In addition to the `mode: apply` the 'operator' needs to provide also the hash present in the KanaryDeployment.Status.CurrentHash in order to validate the `apply` of a specific deployment template and so avoid any mistake.

#### Create KanaryDeployment with `simple` strategy

the following yaml file is an example of how you create a KanaryDeployment with the `simple` strategy:

```yaml
apiVersion: kanary.k8s.io/v1alpha1
kind: KanaryDeployment
metadata:
  name: nginx
  labels:
    app: nginx
spec:
  serviceName: nginx
  strategy:
    simple:
      replicas: 1
      mode: deactivate
  template:
    # deployment template
```

This artifact will trigger the creation of 2 deployments:

- `principal` Deployment: its name will be the same than the KanaryDeployment
- `canary` Deployment: its name will be the KanaryDeployment name plus the prefix "-kanary"

Also, a Service will be created in order to target only the Canary pods managed by the `canary` Deployment. This service is created only if the `KanaryDeployment.spec.serviceName` is defined.

If the `deactivate` mode is set, the `canary` deployment replicas value is forced to `0`.

#### Move to `activate` mode

When you are ready to activate the canary testing of a new version of your application, you need to update the KanaryDeployment specification:

- Update the `spec.template` with your new application version definition.
- Update the `spec.strategy.simple.mode` to `activate`.

```yaml
apiVersion: kanary.k8s.io/v1alpha1
kind: KanaryDeployment
metadata:
  name: nginx
  labels:
    app: nginx
spec:
  serviceName: nginx
  strategy:
    simple:
      replicas: 1
      mode: activate
  template:
    # deployment template
```

Those update can be done at the same time, or with 2 differents KanaryDeployment updates. It will trigger the deployment of your new application only on the `canary` deployment instance.
You will have all the time needed to validate this canary deployment. If never you detect an issue with the `canary` pods, you can set back the `spec.strategy.simple.mode` to `deactivate`; this action will
force the `canary` deployment replicas is forced to `0`.

#### Move to `apply` mode

When you have validated properly you canary deployment, the next step is to move the `apply` mode.

In this mode, the Kanary controller will update the `principal` Deployment. with the same Deployment template used for the `canary` deployment.

to activate this mode you need to update two fields in the KanaryDeployment spec simple strategy:

- Update the `spec.strategy.simple.mode` to `activate`.
- Update the `spec.strategy.simple.applyHash` with the value present in the `status.currentHash` field. This is to avoid any unwanted deployment template update done by mistake between the `activate` and the `apply`.

```yaml
apiVersion: kanary.k8s.io/v1alpha1
kind: KanaryDeployment
metadata:
  name: nginx
  labels:
    app: nginx
spec:
  serviceName: nginx
  strategy:
    simple:
      replicas: 1
      mode: apply
      applyHash: sdsfsfjasfmadwdj9ad
  template:
    # deployment template
    # ...
status:
    currentHash: sdsfsfjasfmadwdj9ad
```