# Operator for OSBuilder

This repo has the operator which provides a K8S API for building [OSBuild](https://www.osbuild.org/) images.

## Installation

### Requirements

- `go >= 1.17`

## Getting started

### Deployment

- Build and push image:

    `IMG=<image repository and tag> make docker-build docker-push`

    for example: `IMG=quay.io/project-flotta/osbuild-operator:latest make docker-build docker-push`

- Deploy the operator:

    `IMG=<image repository and tag> make deploy`

    for example: `IMG=quay.io/project-flotta/osbuild-operator:latest make deploy`

- In the `osbuild` namespace an operator should be running:
  ```
  → oc get pods -n osbuild
  NAME                                                  READY   STATUS    RESTARTS   AGE
  osbuild-operator-controller-manager-54f9fdbff-85hfj   2/2     Running   0          2m47s
  ```

### Run

- Apply the project's CRDs on the cluster:
  
  `make install`

- Run the operator locally (outside the cluster):
  
  `make run`
