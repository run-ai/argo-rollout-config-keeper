# argo-rollout-config-keeper

argo-rollout-config-keeper is a Kubernetes controller for managing Argo Rollout application configurations. It ensures that the specified configurations versions are applied and maintained across the cluster.

It's Also adding the following annotation to the outdated version of managed configMaps and secrets: `argocd.argoproj.io/compare-options: IgnoreExtraneous` to ignore them from the application sync global status. More details can be found in [ArgoCd documentation](https://argo-cd.readthedocs.io/en/stable/user-guide/compare-options/)

There are two main CRDs that the controller is managing:
- ArgoRolloutConfigKeeper.
- ArgoRolloutConfigKeeperClusterScope.

The ArgoRolloutConfigKeeper is managing the configMaps and secrets in the namespace scope, while the ArgoRolloutConfigKeeperClusterScope is managing the configMaps and secrets in the cluster scope.

> [!WARNING]
>
> Please note, both CRDs are having the following optional fields to override the operator defaults values, and it's important for the operator mechanism because it filtered the ReplicaSets by them:
> - `appLabel`, if it not set it will get the following default value: `app.kubernetes.io/name`.
> - `appVersionLabel`, if it not set it will get the following default value: `app.kubernetes.io/version`.

you can see example yamls, [here](../config/samples/): 


## Getting Started

### Prerequisites
- go version v1.20.0+
- docker version 17.03+.
- kubectl version v1.11.3+.
- Access to a Kubernetes v1.11.3+ cluster.

### To Deploy on the cluster
**Clone the repository:**
```sh
git clone https://github.com/yossig-runai/ArgoRolloutConfigKeeper.git
cd ArgoRolloutConfigKeeper
```

**Build and push your image to the location specified by `IMG`:**

```sh
make docker-build docker-push IMG=<some-registry>/argo-rollout-config-keeper:tag
```

**NOTE:** This image ought to be published in the personal registry you specified.
And it is required to have access to pull the image from the working environment.
Make sure you have the proper permission to the registry if the above commands don’t work.

**Install the CRDs into the cluster:**

```sh
make install
```
or
```sh
kubectl apply -f github.com/run-ai/argo-rollout-config-keeper/rleasses/v0.0.3/argo-rollout-config-keeper.yaml
``` 


**Deploy the Manager to the cluster with the image specified by `IMG`:**

```sh
make deploy IMG=<some-registry>/argo-rollout-config-keeper:tag
```

> **NOTE**: If you encounter RBAC errors, you may need to grant yourself cluster-admin
privileges or be logged in as admin.

**Create instances of your solution**
You can apply the samples (examples) from the config/sample:

```sh
kubectl apply -k config/samples/
```

>**NOTE**: Ensure that the samples has default values to test it out.

### To Uninstall
**Delete the instances (CRs) from the cluster:**

```sh
kubectl delete -k config/samples/
```

**Delete the APIs(CRDs) from the cluster:**

```sh
make uninstall
```

**UnDeploy the controller from the cluster:**

```sh
make undeploy
```

## Testing
Run unit tests
 ```sh
 make test
 ```

## Grafana Dashboards

![](./docs/grafana%20dashboards/grafana.png)

Example dashboards can be found in the following links:

[Cluster Scope Dashboard](./docs/grafana%20dashboards/cluster-scope-dashboard.json)

[Namespace Scope Dashboard](./docs/grafana%20dashboards/namespace-scope-dashboard.json)

## Contributing

Please see [CONTRIBUTING.md](CONTRIBUTING.md) for details on how to contribute to this project.

**NOTE:** Run `make help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

## License

Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

