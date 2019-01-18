# List of actions to be discuss and/or coded

## Add Dry-run KanaryDeployment validation (simple)
Allow to run a kanaryDeployment without updating the Deployment in case of success.

## Validation service/deployment (simple)
Validate that the selector of the service in the spec is really selecting the pods created by the deploymentTemplate.

## Extend to more than one service (complex + discussion)
A pod can be addressed by multiple service.
Should we propose to list the servicename, or list the trafficSpec, or just propose to create a different KanaryDeployment for each service? (last proposition already possible today)

## In case of Invalid KanaryDeployment, remove pod from Service (medium)
If a KanaryDeployment is set as invalid, it should not interfer anymore with the service

## HPA (medium)
Introduce HAP in the KanaryDeploymentSpecScale

## SourceFilter based on istio (complex)
Pilot Source and traffic split using istio from the TrafficSpec (not only mirror)

## Validation based on annotation (simple)
Add annotationWatch (just like labelWatch)

## Use Controller Cache
In many places in the reconcile loop we get the deployment... maybe we can reuse the controller cache in some cases?