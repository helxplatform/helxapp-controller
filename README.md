# helxapp-controller

A Kubernetes operator that manages application deployments via three CRDs: **HelxApp** (application template), **HelxInst** (instance request), and **HelxUser** (user record). When all three exist and are correlated, the controller synthesizes Deployments, Services, and PersistentVolumeClaims.

See [docs/execution-model.md](docs/execution-model.md) for detailed architecture documentation.

## Deployment

The helm chart supports two installation modes controlled by the `cluster` value (default `false`).

### Cluster install (cluster admin)

A cluster admin performs the one-time setup: installing CRDs and optionally deploying the controller cluster-wide.

1. Install CRDs:

```sh
make install
```

2. Deploy the controller cluster-wide with the helm chart:

```sh
helm install helxapp-controller chart/ --set cluster=true
```

Or equivalently via kustomize:

```sh
make deploy IMG=<registry>/helxapp-controller:tag
```

### Namespace install (developer)

For developers running their own controller instance in a namespace, a cluster admin must first grant the developer’s service account CRD permissions. This is needed because Kubernetes RBAC escalation prevention blocks a service account from creating roles that grant permissions it doesn’t already hold.

**Step 1 — Cluster admin grants access (one-time per namespace):**

```sh
make grant-access SA=<namespace>:<service-account>
```

For example:

```sh
make grant-access SA=jeffw:jeffw
```

This creates a ClusterRole with `helx.renci.org` CRD permissions and a RoleBinding in the target namespace granting those permissions to the service account.

**Step 2 — Developer installs the helm chart:**

```sh
helm install helxapp-controller chart/
```

With `cluster=false` (the default), the chart creates only namespace-scoped Roles and RoleBindings. No cluster-admin privileges are required for the helm install itself.

### Uninstall

Remove the controller:

```sh
helm uninstall helxapp-controller
```

Remove CRDs from the cluster (cluster admin):

```sh
make uninstall
```

## Development

### Running locally

1. Install CRDs into the cluster:

```sh
make install
```

2. Run the controller locally (uses current kubeconfig context):

```sh
make run
```

### Building

```sh
make build                                        # compile to bin/manager
make docker-build docker-push IMG=<registry>:tag  # build and push container image
```

### Testing

```sh
make test
```

### Modifying the API definitions

After editing `api/v1/*_types.go`, regenerate manifests and DeepCopy methods:

```sh
make manifests generate
```

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

## License

Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

