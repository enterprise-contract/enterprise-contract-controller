# Enterprise Contract controller

A Kubernetes controller that defines a CRD for Conforma (formerly known as Enterprise Contract) resources.

## Description
Currently contains `EnterpriseContractConfiguration` Kubernetes custom resource. See an [example](config/samples/appstudio.redhat.com_v1alpha1_enterprisecontractpolicy.yaml).

> [!NOTE]
> Enterprise Contract is now called Conforma. However, because changing the CRD and controller name would have a large impact, we're not going to rename them at this stage.

## Getting Started
You'll need a Kubernetes cluster to run against. You can use [KIND](https://sigs.k8s.io/kind) to get a local cluster for testing, or run against a remote cluster.
**Note:** Your controller will automatically use the current context in your kubeconfig file (i.e. whatever cluster `kubectl cluster-info` shows).

### Running on the cluster
1. Install Instances of Custom Resources:

```sh
kubectl apply -f config/samples/
```

2. Build and push your image to the location specified by `IMG`:

```sh
make docker-build docker-push IMG=<some-registry>/enterprise-contract-controller:tag
```

3. Deploy the controller to the cluster with the image specified by `IMG`:

```sh
make deploy IMG=<some-registry>/enterprise-contract-controller:tag
```

### Deployment Options

There are two ways to deploy the controller:

1. **Default Deployment**: Deploys the EnterpriseContract CRD and enables the basic reconciler for EnterpriseContract resources.

```sh
# Set your container registry
export KO_DOCKER_REPO=<your-container-registry>

# Deploy using ko and kustomize
kustomize build config/default | ko apply -f -
```

2. **PipelineRun Reconciler Deployment**: In addition to the default deployment, this enables a PipelineRun reconciler that triggers Conforma to verify PipelineRun attestations.

```sh
# Set your container registry
export KO_DOCKER_REPO=<your-container-registry>

# Deploy using ko and kustomize with PipelineRun reconciler
kustomize build config/overlays/pipelinerun/ | ko apply -f -
```

### Local Development
For local development and testing, you can use ko for faster iteration:

```sh
# Set your container registry
export KO_DOCKER_REPO=<your-container-registry>

# Deploy using ko and kustomize
kustomize build config/default | ko apply -f -
```

To run the test suite:

```sh
make test
```

### Uninstall CRDs
To delete the CRDs from the cluster:

```sh
make uninstall
```

### Undeploy controller
UnDeploy the controller to the cluster:

```sh
make undeploy
```

## Contributing
// TODO(user): Add detailed information on how you would like others to contribute to this project

### How it works
This project aims to follow the Kubernetes [Operator pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/)

It uses [Controllers](https://kubernetes.io/docs/concepts/architecture/controller/)
which provides a reconcile function responsible for synchronizing resources until the desired state is reached on the cluster

### Test It Out
1. Install the CRDs into the cluster:

```sh
make install
```

2. Run your controller (this will run in the foreground, so switch to a new terminal if you want to leave it running):

```sh
make run
```

**NOTE:** You can also run this in one step by running: `make install run`

### Modifying the API definitions
If you are editing the API definitions, generate the manifests such as CRs or CRDs using:

```sh
make manifests
```

**NOTE:** Run `make --help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

## License

Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
