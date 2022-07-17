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
- Deploy nexus operator by following the instruction here: https://github.com/RedHatGov/nexus-operator#installation
- Deploy a nexus instance:
  ```bash
  oc apply -n osbuild -f config/creating_env/deploy_nexus.yaml
  ```
- Install the Nexus CLI
  ```bash
  pip install nexus3-cli
  ```
- Login to Nexus
  ```bash
  export CLUSTER_DOMAIN='mycluster.example.com'
  oc get secret -n osbuild nexus-osbuild-admin-credentials -o jsonpath={.data.password} | base64 -d | xargs nexus3 login --url https://nexus-osbuild-osbuild.apps.${CLUSTER_DOMAIN} --no-x509_verify --username admin --password
  ```
- Create a Hosted Raw repository named _disk-images_
  ```bash
  nexus3 repository create hosted raw disk-images
  ```
- Upload the rhel qcow2 image
  ```bash
  nexus3 upload rhel-8.6-x86_64-kvm.qcow2 disk-images
  ```

## Create generic S3 service
- Deploy MiniO
  ```bash
  oc apply -n osbuild -f config/creating_env/deploy_minio.yaml
  ```
- Create a bucket name `osbuild-images`. You can use the `aws` cli:
  ```bash
  export CLUSTER_DOMAIN='mycluster.example.com'
  AWS_ACCESS_KEY_ID=minioadmin AWS_SECRET_ACCESS_KEY=minioadmin aws --endpoint-url https://minio-s3-osbuild.apps.${CLUSTER_DOMAIN} --no-verify-ssl s3 mb s3://osbuild-images
  ```
- Create a secret
  ```bash
  oc create secret generic osbuild-s3-credentials -n osbuild --from-literal=access-key-id=minioadmin --from-literal=secret-access-key=minioadmin
  ```

## Create Secret for RedHat Credentials
- Find your RH creds and create a secret:
  ```bash
  oc create secret generic redhat-portal-credentials -n osbuild --from-literal=username=<USERNAME> --from-literal=password=<PASSWORD>
  ```

## Create PSQL Server
Currently the controller does not support creating the PSQL server on its own, making this step mandatory
- Edit the file `config/creating_env/psql.env` and set a real Password
- Create the PSQL server
  ```bash
  oc new-app --env-file config/creating_env/psql.env postgresql:13-el8 -n osbuild
  ```
- Edit the file `config/creating_env/postgress_secret.yaml` with the same Password
- Create new secret (please enter a real encoded password)
  ```bash
  oc create -f config/creating_env/postgress_secret.yaml
  ```

## Create OSBuildEnvConfig singleton CR
- Apply OSBuildEnvConfig
  ```bash
  oc create -f config/samples/osbuilder_v1alpha1_osbuildenvconfig.yaml
  ```

## SSH into osbuild-workers
- Fetch the Private key from the secret and save it to a file
  ```bash
  oc get secret -n osbuild osbuild-worker-ssh -o jsonpath={.data.ssh-privatekey} | base64 -d > /tmp/worker-ssh.key
  ```
- In a separate shell run a debug pod (for example, using node _worker-1_)
  ```bash
  oc debug node/worker-1
  ```
- Copy the SSH key to the debug pod
  ```bash
  oc cp /tmp/worker-ssh.key worker-1-debug:/root/
  ```
- Get the worker VM's IP address (for the worker named _builder-1_ )
  ```bash
  oc get vmi builder-1 -o jsonpath={.status.interfaces[0].ipAddress}
  ```
- Go back to the debug pod and ssh into the worker with cloud-user and the VMI's IP
  ```bash
  ssh -i /root/worker-ssh.key cloud-user@<IP Address>
  ```
