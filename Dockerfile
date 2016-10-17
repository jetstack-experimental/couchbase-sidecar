FROM busybox
ADD couchbase-sidecar /couchbase-sidecar
CMD ["/couchbase-sidecar"]
