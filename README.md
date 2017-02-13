# Couchbase on Kubernetes

Couchbase Server is a NoSQL document database with a distributed architecture for performance, scalability, and availability. It enables developers to build applications easier and
faster by leveraging the power of SQL with the flexibility of JSON. This project provides a proof-of-concept sidecar container that can be used alongside official couchbase images
to provide a scalable and flexible couchbase deployment.

The sidecar is responsible for registering new nodes into the couchbase cluster, automatically triggering cluster rebalances, and handles migration of data given a scale down or node failure event.

Whilst working and handling the majority of failure cases, this **should not be used in production**. It has not been battle tested, and there are some edge-cases that can result in manual intervention
being needed to bring your cluster back online. Please check the `Limitations/Caveats` section below for more details.

## Requirements

* Support for dynamic volume provisioning, or a set of pre-made PVs
* Helm/tiller (optional - it's possible to modify the helm templates to create manifests)
* Cluster supporting PetSet/StatefulSet

## Getting started

We use Helm to easily package up Couchbase, making it easier to view all available configuration options and easily manage the lifecycle of your deployment. More information on Helm can be found [here](https://github.com/kubernetes/helm).

1) A pre-made Helm chart is available in the [contrib/charts/couchbase](contrib/charts/couchbase) directory. You can review the available configuration options in [values.yaml](contrib/charts/couchbase/values.yaml), or [README.md](contrib/charts/couchbase/README.md).

2) As Couchbase recommends not to use loadbalancers across nodes, we will use `port-forward` to access the Couchbase UI. If you named your release `my-release`, you can port-forward to one of your nodes for example with the following:

```bash
$ kubectl port-forward --namespace=<namespace> my-release-couchbase-data-0 8091:8091
```

You should then be able to access the web UI at `http://localhost:8091/`

3) We can scale up the cluster using a Helm upgrade:

```bash
$ helm upgrade --set roles.data.replicaCount=5 my-release jetstack/couchbase
```

and back down again:

```bash
$ helm upgrade --set roles.data.replicaCount=3 my-release jetstack/couchbase
```

When scaling down, you should scale by no more than your minimum replica count for a bucket (so if your least replicated bucket has 1 replica, scale down one at a time) - if you scale quicker than this, you will most likely cause data loss.

4) We can clean up after ourselves with a `helm delete`:

```bash
$ helm delete my-release
```

__This will remove all cluster data, and is an irreversible action__

## Limitations/Caveats

* This has only been tested for cluster-local access. It should be possible to access externally, however it will require careful configuration of the network fabric, DNS and port-forwarding on pods
* Occasionally, after deleting all pods/restarting the cluster, data nodes can get stuck in the `warmup` state. Kubernetes will stop scaling up as the sidecar will report not-healthy, and a deadlock will occur
(Couchbase needs more nodes in order to function, but Kubernetes won't add more nodes until it is functioning)
* Using emptyDir for persistent disks can result in storage issues. On GKE, the stateful data partition gets remounted RO and the whole Kubernetes cluster stops working
* Currently only supports enterprise edition, as the sidecar attempts to label nodes with zone information, which is unsupported in Couchbase community edition. This will be fixed in a later release
