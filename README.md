# Operator for OSBuilder

This repo has the operator which provides a K8S API for building [OSBuild](https://www.osbuild.org/) images.

## Installation

### Requirements

- `go >= 1.17`

### Test Requirements
- `genisoimage`
- `xorriso`

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

#### Certificate Manager Operator

Please note that the provisioning of the OSBuild Operator will also provision the [cert-manager](https://cert-manager.io/) Operator as it is a prerequisite for Admission Webhooks

### Run

- Apply the project's CRDs on the cluster:

  `make install`

- Run the operator locally (outside the cluster):

  `make run`

## Store rhel image in accessible endpoint
### Deploy nexus
- Deploy nexus operator - use that instruction: https://github.com/RedHatGov/nexus-operator#installation
- Deploy a nexus instance:
  `oc apply -g config/creating_env/deploy_nexus.yaml -n osbuild`
- Get admin password by reading the secret 
  `oc get secret nexus-osbuild-admin-credentials -o yaml -n osbuild`
- Log in the nexus UI (use the route URL) and create a raw-host repository
- Upload the rhel qcow2 image 
  ```
  pip install nexus3-cli
  nexus3 login --url <URL> --username admin --password <PASSWORD>
  nexus3 upload worker-image.qcow2 raw-hosted
  ```

## Create generic S3 service
- Deploy MiniO
  `oc apply -f config/creating_env/deploy_minio.yaml`
- Create a bucket `osbuild-images`
- Create a secret
  `oc create secret generic osbuild-s3-credentials -n osbuild --from-literal=access-key-id=minioadmin --from-literal=secret-access-key=minioadmin --type=kubernetes.io/glusterfs`

## Create OSBuildEnvConfig singleton CR
- Apply postgresssql (please enter a real Password)
  `oc new-app --env-file config/creating_env/psql.env postgresql:13-el8 -n osbuild`
- Create new secret (please enter a real encoded password)
  `oc create -f config/creating_env/postgress_secret.yaml`
- Apply OSBuildEnvConfig
  `oc create -f config/samples/osbuilder_v1alpha1_osbuildenvconfig.yaml`

## SSH into osbuild-workers
- Get the secret that contains the ssh-key 
  `oc get secret osbuild-worker-ssh -n osbuild -o yaml`
- copy into a debug node's pod (for example worker-1-debug)
  `oc cp worker-ssh.key worker-1-debug:/root/`
- ssh into the worker with cloud-user and the VMI's IP
