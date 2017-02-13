# Couchbase

Couchbase Server is a NoSQL document database with a distributed architecture for performance, scalability, and availability. It enables developers to build applications easier and
faster by leveraging the power of SQL with the flexibility of JSON.

## Introduction

This chart bootstraps a multi-node Couchbase cluster, with separate index, query and data nodes. It utilises PetSet/StatefulSet to maintain node identity, and supports scale up & scale down actions.

## Prerequisites

* Kubernetes 1.4+ with alpha APIs enabled OR Kubernetes 1.5+ with beta APIs enabled
* __Suggested:__ PV provisioner support in the underlying infrastructure

## Installing the Chart

To install the chart with the release name `my-release` in the namespace `couchbase`:

```bash
$ helm install jetstack/couchbase --name my-release --namespace couchbase
```

This will deploy a simple multi-node Couchbase cluster with the default options.

## Uninstalling the Chart

To uninstall/delete the `my-release` deployment:

```bash
$ helm delete my-release
```

The command removes all the Kubernetes components associated with the chart and deletes the release. __This will also delete all data disks created for this release.__

## Configuration

The following table lists the configurable parameters of the Couchbase chart and their default values:

| Parameter                                   | Description                                                                     | Default                                                                 |
| ------------------------------------------- | ------------------------------------------------------------------------------- | ----------------------------------------------------------------------- |
| `statefulSet.apiVersion`                    | API version to use for StatefulSet. Use to switch to PetSet on pre-1.5 clusters | `apps/v1beta1`                                                          |
| `statefulSet.kind`                          | Kind for the StatefulSet. Use to switch to PetSet on pre-1.5 clusters           | `StatefulSet`                                                           |
| `couchbase.username`                        | Default admin username to use for Couchbase                                     | `admin`                                                                 |
| `couchbase.password`                        | Default admin password to use for Couchbase                                     | `insecure`                                                              |
| `couchbase.image.repository`                | Image repository to use for Couchbase                                           | `couchbase`                                                             |
| `couchbase.image.tag`                       | Image tag to use for Couchbase                                                  | `enterprise`                                                            |
| `couchbase.image.pullPolicy`                | Image pull policy to use for Couchbase components                               | `IfNotPresent`                                                          |
| `sidecar.image.repository`                  | Image repository to use for the Couchbase sidecar                               | `jetstackexperimental/couchbase-sidecar`                                |
| `sidecar.image.tag`                         | Image tag to use for the Couchbase sidecar                                      | `0.0.2`                                                                 |
| `sidecar.image.pullPolicy`                  | Image pull policy to use for the Couchbase sidecar                              | `IfNotPresent`                                                          |
| `sidecar.resources`                         | Sidecar resource config (YAML)                                                  | `limits: {cpu: 100m, memory: 128Mi}, requests: {cpu:10m, memory: 32Mi}` |
| `roles.data.replicaCount`                   | Total number of Couchbase data nodes                                            | `3`                                                                     |
| `roles.data.terminationGracePeriodSeconds`  | Number of seconds to wait before forcefully killing data nodes                  | `86400` (24h)                                                           |
| `roles.data.storage.class`                  | Storage class to use for data node storage                                      | `anything`                                                              |
| `roles.data.storage.size`                   | Amount of storage to request for data nodes                                     | `50Gi`                                                                  |
| `roles.data.resources`                      | Data node resource config (YAML)                                                | `limits: {cpu: 2, memory: 2Gi}, requests: {cpu: 100m, memory: 2Gi}`     |
| `roles.query.replicaCount`                  | Total number of Couchbase query nodes                                           | `3`                                                                     |
| `roles.query.terminationGracePeriodSeconds` | Number of seconds to wait before forcefully killing query nodes                 | `300`                                                                   |
| `roles.query.storage.class`                 | Storage class to use for query nodes storage                                    | `anything`                                                              |
| `roles.query.storage.size`                  | Amount of storage to requery for query nodes                                    | `5Gi`                                                                   |
| `roles.query.resources`                     | Query node resource config (YAML)                                               | `limits: {cpu: 1, memory: 1Gi}, requests: {cpu: 100m, memory: 1Gi}`     |
| `roles.index.replicaCount`                  | Total number of Couchbase index nodes                                           | `3`                                                                     |
| `roles.index.terminationGracePeriodSeconds` | Number of seconds to wait before forcefully killing index nodes                 | `300`                                                                   |
| `roles.index.storage.class`                 | Storage class to use for index nodes storage                                    | `anything`                                                              |
| `roles.index.storage.size`                  | Amount of storage to request for index nodes                                    | `5Gi`                                                                   |
| `roles.index.resources`                     | Index node resource config (YAML)                                               | `limits: {cpu: 1, memory: 1Gi}, requests: {cpu: 100m, memory: 1Gi}`     |

Specify each parameter using the `--set key=value[,key=value]` argument to `helm install`. For example,

```console
$ helm install --name my-release \
  --set couchbase.password=secure \
    jetstack/couchbase
```

Alternatively, a YAML file that specifies the values for the parameters can be provided while installing the chart. For example,

```console
$ helm install --name my-release -f values.yaml jetstack/couchbase
```

> **Tip**: You can use the default [values.yaml](values.yaml)
