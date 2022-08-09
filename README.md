# Operator for OSBuilder

This repo has the operator which provides a K8S API for building [OSBuild](https://www.osbuild.org/) images.

## Installation

### Requirements

- `go >= 1.17`
- To provision Internal Worker VMs, or to follow the sample for the External Worker VM, `kubevirt-hyperconverged` is required

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

## Images for Worker VMs
There are two ways to configure the base images of the Worker VMs:

### Provide a RHEL QCOW2 Image
To Provide a RHEL QCOW2 Image you will need to store it in an accessible endpoint and provide its link in the OSBuildEnvConfig
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
- Set the URL in the vmWorkerConfig field of the OSBuildEnvConfig CR
  ```yaml
  worker:
  - name: WorkerName
    vmWorkerConfig:
      dataVolumeSource:
        http:
          url: "http://nexus-osbuild:8081/repository/disk-images/rhel-8.6-x86_64-kvm.qcow2"
  ```

### Use RHEL golden Images of CNV
To use RHEL golden Images of CNV you will need to create a secret containing your credentials for `registry.redhat.io` provide the secret reference in the OSBuildEnvConfig
- Create a Secret containing your credentials for `registry.redhat.io`
  ```bash
  oc create secret generic osbuild-registry-redhat-io-credentials -n osbuild --from-literal=accessKeyId=<Username> --from-literal=secretKey=<Password>
  ```
- Set the ImageRegistrySecretReference field of the OSBuildEnvConfig CR
  ```yaml
  worker:
  - name: WorkerName
    vmWorkerConfig:
      dataVolumeSource:
        registry:
          url: docker://registry.redhat.io/rhel8/rhel-guest-image:8.6.0
          secretRef: osbuild-registry-redhat-io-credentials
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
- Create a secret for the S3 credentials
  ```bash
  oc create secret generic osbuild-s3-credentials -n osbuild --from-literal=access-key-id=minioadmin --from-literal=secret-access-key=minioadmin
  ```
- Create a secret for the CA Bundle using the OCP route
  ```bash
  oc get secrets -n openshift-ingress-operator router-ca -o "jsonpath={.data.tls\.crt}" | base64 -d > /tmp/ca-bundle
  oc create secret generic osbuild-s3-ca-bundle -n osbuild --from-file=/tmp/ca-bundle
  ```

## Create a Container Registry service
- Create an HTPasswd file
  ```bash
  htpasswd -Bbc /tmp/auth admin <Password>
  ```
- Create a secret for the HTPasswd file
  ```bash
  oc create secret generic container-registry-auth -n osbuild --from-file=/tmp/auth
  ```
- Because docker.io uses rate limiting it is recommanded to use an Image Pull Secret
  - If you don't have an account at docker.io create one
  - Create a docker-registry secret with your credentials
  ```bash
  oc create secret docker-registry docker-io-creds -n osbuild --docker-server=docker.io --docker-username=<Username> --docker-password=<Password>
  ```
  - Uncommnet the imagePullSecret field of the podSpec in `config/creating_env/deploy_container_registry.yaml`
- Deploy the container registry
  ```bash
  oc apply -f config/creating_env/deploy_container_registry.yaml -n osbuild
  ```
- Add the OCP CA certificate to its additionalTrustedCA
  ```bash
  export CLUSTER_DOMAIN='mycluster.example.com'
  oc get secrets -n openshift-ingress-operator router-ca -o "jsonpath={.data.tls\.crt}" | base64 -d > /tmp/ca.crt
  oc create configmap osbuild-registry-config --from-file=container-registry-osbuild.apps.${CLUSTER_DOMAIN}=/tmp/ca.crt -n openshift-config
  oc patch image.config.openshift.io/cluster --patch '{"spec":{"additionalTrustedCA":{"name":"osbuild-registry-config"}}}' --type=merge
  ```
- Create a secret for the Container Registry credentials
  ```bash
  oc create secret docker-registry osbuild-registry-credentials -n osbuild --docker-server=container-registry-osbuild.apps.${CLUSTER_DOMAIN} --docker-username=admin --docker-password=<Password>
  ```
- If you wish to use images from this registry, you will need to create the same secret in your namespace and add the following to your PodSpec:
  ```yaml
  spec:
    imagePullSecrets:
      - name: < Secret Name >
  ```
- Create a secret for the CA Bundle using the OCP route
  ```bash
  oc get secrets -n openshift-ingress-operator router-ca -o "jsonpath={.data.tls\.crt}" | base64 -d > /tmp/ca-bundle
  oc create secret generic osbuild-container-registry-ca-bundle -n osbuild --from-file=/tmp/ca-bundle
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

## External Worker VM using CNV
- Create an SSH key-pair
  ```bash
  ssh-keygen -t rsa -b 4096 -C cloud-user@external-builder -f ~/.ssh/external-builder
  ```
- Create symlinks to the files to facilitate the next step
  ```bash
  ln -s ~/.ssh/external-builder.pub config/creating_env/ssh-publickey
  ln -s ~/.ssh/external-builder config/creating_env/ssh-privatekey
  ```
- Generate the secret
  ```bash
  oc create secret generic external-builder-ssh-pair --from-file=config/creating_env/ssh-privatekey --from-file=config/creating_env/ssh-publickey -n osbuild
  ```
- The example External worker uses a RHEL Golden imageas defined for the [VM Worker](README.md#use-rhel-golden-images-of-cnv)
- Deploy the VM
  ```bash
  oc apply -n osbuild -f config/creating_env/external-worker-vm.yaml
  ```
- Wait for the VM to reach Ready state
  ```bash
  oc wait --for=condition=Ready -n osbuild virtualmachine.kubevirt.io/external-builder --timeout=5m
  ```
- Get VM Address
  ```bash
  oc get vmi -n osbuild external-builder -o jsonpath={.status.interfaces[0].ipAddress}
  ```

## Create OSBuildEnvConfig singleton CR
- Apply OSBuildEnvConfig
  ```bash
  export CLUSTER_DOMAIN='mycluster.example.com'
  export EXTERNAL_WORKER_IP=`oc get vmi external-builder -n osbuild -o jsonpath={.status.interfaces[0].ipAddress}`
  cat config/samples/osbuilder_v1alpha1_osbuildenvconfig.yaml | envsubst | oc apply -f -
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

## Building an edge-container image
- Edit the sample [OSBuildConfig](config/samples/osbuilder_v1alpha1_osbuildconfig.yaml) and create the instance
  ```bash
  oc apply -f config/samples/osbuilder_v1alpha1_osbuildconfig.yaml
  ```
- Look at the OSBuildConfig status to the index of the OSBuild instance that was created
  ```bash
  oc get osbuildconfig osbuildconfig-sample -o jsonpath={.status.lastVersion}
  ```
- Wait for the OSBuild instance to finish successfully by running the command below and waiting for `containerBuildDone`
  ```bash
  oc get osbuild osbuildconfig-sample-1 -o jsonpath={.status.conditions}
  ```
- Get the URL of the Container Image
  ```bash
  oc get osbuild osbuildconfig-sample-1 -o jsonpath={.status.containerUrl}
  ```

## Deploy the Edge Container
- Create a docker registry secret for your Container Image Registry as explained [here](README.md#create-a-container-registry-service)
- Edit the sample Edge Commit [Deployment](config/creating_env/deploy_edge_commit.yaml) with the URL returned by the OSBuild CR's status and the name of the secret you created
- Deploy the Edge Commit and expose it as a Route
  ```bash
  oc apply -f config/creating_env/deploy_edge_commit.yaml
  ```
- Once the deployment is in status Running fetch the Commit
  ```bash
  export CLUSTER_DOMAIN='mycluster.example.com'
  export REF='rhel/8/x86_64/edge'
  curl -k https://edge-commit-default.apps.${CLUSTER_DOMAIN}/repo/refs/heads/${REF}
  ```