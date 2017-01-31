{{/* vim: set filetype=mustache: */}}
{{/*
Couchbase Container
*/}}
{{- define "couchbase-container" -}}
- name: couchbase
  image: {{ .Values.couchbase.image.repository }}:{{ .Values.couchbase.image.tag }}
  imagePullPolicy: {{ .Values.couchbase.image.imagePullPolicy }}
  livenessProbe:
    initialDelaySeconds: 30
    tcpSocket:
      port: 8091
    timeoutSeconds: 1
  lifecycle:
    preStop:
      exec:
        command:
        - "/sidecar/couchbase-sidecar"
        - stop
  ports:
  - containerPort: 8091
    name: cb-admin
  - containerPort: 8092
    name: cb-views
  - containerPort: 8093
    name: cb-queries
  - containerPort: 8094
    name: cb-search
  - containerPort: 9100
    name: cb-int-ind-ad
  - containerPort: 9101
    name: cb-int-ind-sc
  - containerPort: 9102
    name: cb-int-ind-ht
  - containerPort: 9103
    name: cb-int-ind-in
  - containerPort: 9104
    name: cb-int-ind-ca
  - containerPort: 9105
    name: cb-int-ind-ma
  - containerPort: 9998
    name: cb-int-rest
  - containerPort: 9999
    name: cb-int-gsi
  - containerPort: 11207
    name: cb-memc-ssl
  - containerPort: 11209
    name: cb-int-bu
  - containerPort: 11210
    name: cb-moxi
  - containerPort: 11211
    name: cb-memc
  - containerPort: 11214
    name: cb-ssl-xdr1
  - containerPort: 11215
    name: cb-ssl-xdr2
  - containerPort: 18091
    name: cb-admin-ssl
  - containerPort: 18092
    name: cb-views-ssl
  - containerPort: 18093
    name: cb-queries-ssl
  - containerPort: 4369
    name: empd
  resources:
{{ toYaml .Values.sidecar.resources | indent 4 }}
  volumeMounts:
  - mountPath: "/opt/couchbase/var"
    name: data
  - mountPath: "/sidecar"
    name: sidecar
{{- end -}}
