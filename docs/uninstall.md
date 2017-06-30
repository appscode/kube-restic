> New to Stash? Please start [here](/docs/tutorial.md).

# Uninstall Stash
Please follow the steps below to uninsall Stash:

1. Delete the deployment and service used for Stash operator.
```sh
$ kubectl delete deployment -l app=stash -n <operator-namespace>
$ kubectl delete service -l app=stash -n <operator-namespace>
```

2. Now, wait several seconds for Stash to stop running. To confirm that Stash operator pod(s) have stopped running, run:
```sh
$ kubectl get pods --all-namespaces -l app=stash
```

3. To keep a copy of your existing `Restic` objects, run:
```sh
kubectl get restic.stash.appscode.com --all-namespaces -o yaml > data.yaml
```

4. To delete existing `Restic` objects from all namespaces, run the following command in each namespace one by one.
```
kubectl delete restic.stash.appscode.com --all --cascade=false
```

5. Delete the old TPR-registration.
```sh
kubectl delete thirdpartyresource restic.stash.appscode.com
```
