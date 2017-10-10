# Configuring RBAC

To use Stash in a cluster with RBAC enabled, [install Stash](/docs/install.md) with RBAC options.

Sidecar container added to workloads makes various calls to Kubernetes api. ServiceAccounts used with workloads should have the following roles:

```yaml
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRole
metadata:
  name: stash-sidecar
rules:
- apiGroups:
  - stash.appscode.com
  resources: ["*"]
  verbs: ["*"]
- apiGroups:
  - apps
  resources:
  - deployments
  verbs: ["get"]
- apiGroups:
  - extensions
  resources:
  - daemonsets
  - replicasets
  verbs: ["get"]
- apiGroups: [""]
  resources:
  - replicationcontrollers
  - secrets
  verbs: ["get"]
- apiGroups: [""]
  resources:
  - events
  verbs: ["create"]
```

Create `stash-sidecar` ClusterRole, if it is not already present.

Now, create a RoleBinding for service account used to a workload.
```yaml
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: RoleBinding
metadata:
  name: workload-sa
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: stash-sidecar
subjects:
- kind: ServiceAccount
  name: workload-sa
  namespace: default
```

You can find full working examples [here](/docs/examples/workloads).


Further discussion is ongoing whether Stash should automatically configure RBAC for workload service accounts. Please give your feedback [here](https://github.com/appscode/stash/issues/123).
