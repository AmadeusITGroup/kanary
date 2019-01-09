# List of actions to be discuss and/or coded

## Validation service/deployment (simple)
Validate that the selector of the service in the spec is really selecting the pods created by the deploymentTemplate.

## Extend to more than one service (complex + discussion)
A pod can be addressed by multiple service.
Should we propose to list the servicename, or list the trafficSpec, or just propose to create a different KanaryDeployment for each service? (last proposition already possible today)

## Rename Shadow into Mirror (simple)
Since the shadow feature will rely on the implementation of Istio Mirroring feature, the best is to use the same name.
https://istio.io/docs/tasks/traffic-management/mirroring/

## In case of Invalid KanaryDeployment, remove pod from Service
If a KanaryDeployment is set as invalid, it should not interfer anymore with the service

## HPA (medium)
Introduce HAP in the KanaryDeploymentSpecScale

## SourceFilter based on istio (complex)
Pilot Source and traffic split using istio from the TrafficSpec (not only mirror)

## Validation based on annotation (simple)
Add annotationWatch (just like labelWatch)
