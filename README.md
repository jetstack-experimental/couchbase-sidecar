# Couchbase on OpenShift/Kubernetes

Couchbase Server is a NoSQL document database with a distributed architecture for performance, scalability, and availability. It enables developers to build applications easier and
faster by leveraging the power of SQL with the flexibility of JSON.

This project provides a proof-of-concept sidecar container that can be used alongside official Couchbase images to provide a scalable and flexible Couchbase deployment.

The sidecar is responsible for registering new nodes into the Couchbase cluster, automatically triggering cluster rebalances, and handles migration of data given a scale-down or node failure event.

Whilst working and handling the majority of failure cases, this **should not be used in production**. It has not been battle tested, and there are some edge-cases that can result in manual intervention
being needed to bring your cluster back online. Please check the `Limitations/Caveats` section below for more details.

## Requirements

* OpenShift cluster supporting StatefulSet (i.e. Origin 3.3+) with full administrator access
* or Kubernetes cluster supporting StatefulSet (i.e. 1.5+)
  * Helm/Tiller (optional - it's possible to modify the Helm templates to create manifests)
* Support for dynamic volume provisioning, or a set of pre-made PVs

## Getting started

### OpenShift

1) Add Couchbase templates to the `openshift` project:

```bash
$ oc apply --namespace=openshift -f openshift/templates/
```

2) Ensure a default `StorageClass` exists if you wish to use dynamic volume provisioning. For example, to use EBS gp2 on AWS as default `fast` storage:

```bash
$ oc create -f - <<EOF
apiVersion: storage.k8s.io/v1beta1
kind: StorageClass
metadata:
  name: fast
  annotations:
    storageclass.beta.kubernetes.io/is-default-class: "true"
provisioner: kubernetes.io/aws-ebs
parameters:
  type: gp2
EOF
```

3) Now create a new OpenShift project and add additional roles to the project serviceaccount required by the sidecar:

```bash
$ oc new-project couchbase
$ oc policy add-role-to-user edit system:serviceaccount:couchbase:default -n couchbase
$ odm policy add-cluster-role-to-user system:node-reader system:serviceaccount:couchbase:default
```

4) The [couchbase/docker](https://hub.docker.com/r/couchbase/server/) container image initially runs as root for init, and then runs Couchbase components as a couchbase user. By default, OpenShift will prevent root execution in a container so it is necessary to relax these restrictions for the `couchbase` project default serviceaccount.

```bash
$ oadm policy add-scc-to-user anyuid system:serviceaccount:couchbase:default
```

5) Use the OpenShift UI to create a new Couchbase cluster ('Add to project' -> 'New datastore').

### Kubernetes

For Kubernetes deployments, we use [Helm](https://github.com/kubernetes/helm) to package up Couchbase, making it easier to view all available configuration options and easily manage the lifecycle of your deployment.

1) A ready-made Helm chart is available in the [contrib/charts/couchbase](contrib/charts/couchbase) directory. You can review the available configuration options in [values.yaml](contrib/charts/couchbase/values.yaml), or [README.md](contrib/charts/couchbase/README.md).

2) As Couchbase recommends not to use load balancers across nodes, we will use `port-forward` to access the Couchbase UI. If you named your release `my-release`, you can port-forward to one of your nodes for example with the following:

```bash
$ kubectl port-forward --namespace=<namespace> my-release-couchbase-data-0 8091:8091
```

You should then be able to access the web UI at `http://localhost:8091/`

3) We can scale-up the cluster using a Helm upgrade:

```bash
$ helm upgrade --set roles.data.replicaCount=5 my-release jetstack/couchbase
```

and back down again:

```bash
$ helm upgrade --set roles.data.replicaCount=3 my-release jetstack/couchbase
```

When scaling down, you should scale by no more than your minimum replica count for a bucket (so if your least replicated bucket has 1 replica, scale down one at a time) - if you scale quicker than this, you will most likely cause data loss.

4) We can clean-up after ourselves with a `helm delete`:

```bash
$ helm delete my-release
```

__Note: this will remove all cluster data, and is an irreversible action__

## Limitations/Caveats

* This has only been tested for local cluster access. It should be possible to access externally, however it will require careful configuration of the network fabric, DNS and port-forwarding on pods.
* Occasionally, after deleting all pods/restarting the cluster, data nodes can get stuck in the `warmup` state. Kubernetes will stop scaling up as the sidecar will report not-healthy, and a deadlock will occur
(Couchbase needs more nodes in order to function, but Kubernetes won't add more nodes until it is functioning).
* Using `emptyDir` for persistent disks can result in storage issues. On GKE, the stateful data partition gets remounted RO and the whole Kubernetes cluster stops working.
* Currently only supports Enterprise Edition, as the sidecar attempts to label nodes with zone information, which is unsupported in Couchbase Community Edition. This will be fixed in a later release.
