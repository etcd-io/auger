# augerctl

`augerctl` is a command line client for [Kubernetes][kubernetes] specific [etcd][etcd],
and as close as possible to [kubectl][kubectl].
It can be used in scripts or for administrators to explore an etcd cluster.

## Getting augerctl

The latest release is not yet available as a binary on [Github][github-release],
the next release will be available.

so that it can be built from source.

``` bash
git clone https://github.com/etcd-io/auger
cd auger
go install ./augerctl
```

or

``` bash
go install github.com/etcd-io/auger/augerctl@main
```

and the binary will be available in the path `$GOBIN` or `$GOPATH/bin`

## Configuration

### --endpoints
+ gRPC endpoints of etcd cluster
+ default: `"http://127.0.0.1:2379"`

### --cert
+ path to the etcd client TLS cert file
+ default: none

### --key
+ path to the etcd client TLS key file
+ default: none

### --cacert
+ path to the etcd client TLS CA cert file
+ default: none

### --user
+ username for authentication, provide username[:password]
+ default: none

### --password
+ password for authentication, only available if --user has no password
+ default: none

## Usage

### Setting a resource

TODO

### Retrieving a resource

List a single service with namespace `default` and name `kubernetes`

``` bash
augerctl get services -n default kubernetes

# Nearly equivalent
kubectl get services -n default kubernetes -o yaml
```

List a single resource of type `priorityclasses` and name `system-node-critical` without namespaced

``` bash
augerctl get priorityclasses system-node-critical

# Nearly equivalent
kubectl get priorityclasses system-node-critical -o yaml
```

List all leases with namespace `kube-system`

``` bash
augerctl get leases -n kube-system

# Nearly equivalent
kubectl get leases -n kube-system -o yaml
```

Watch all leases with namespace `kube-system`

``` bash
augerctl get leases -n kube-system -w
# Nearly equivalent
kubectl get leases -n kube-system -w -o yaml
```

List a single resource of type `apiservices.apiregistration.k8s.io` and name `v1.apps`

``` bash
augerctl get apiservices.apiregistration.k8s.io v1.apps

# Nearly equivalent
kubectl get apiservices.apiregistration.k8s.io v1.apps -o yaml
```

List all resources

``` bash
augerctl get

# Nearly equivalent
kubectl get $(kubectl api-resources --verbs=list --output=name | paste -s -d, - ) -A -o yaml
```

### Deleting a resource

TODO

### Watching for changes

TODO

## Endpoint

If the etcd cluster isn't available on `http://127.0.0.1:2379`, specify a `--endpoints` flag.

## Project Details

### Versioning

augerctl uses [semantic versioning][semver].
Releases will follow with the [Kubernetes][kubernetes] release cycle as possible (need API updates),
but the version numbers will be not.

### License

augerctl is under the Apache 2.0 license. See the [LICENSE][license] file for details.

[kubernetes]: https://kubernetes.io/
[kubectl]: https://kubectl.sigs.k8s.io/
[etcd]: https://github.com/etcd-io/etcd
[github-release]: https://github.com/etcd-io/auger/releases/
[license]: ../LICENSE
[semver]: http://semver.org/
