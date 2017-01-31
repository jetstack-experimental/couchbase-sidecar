{{/* vim: set filetype=mustache: */}}
{{/*
Couchbase Sidecar Container
*/}}
{{- define "sidecar-container" -}}
- name: couchbase-sidecar
  image: {{ .Values.sidecar.image.repository }}:{{ .Values.sidecar.image.tag }}
  imagePullPolicy: {{ .Values.sidecar.image.imagePullPolicy }}
  env:
  - name: POD_NAME
    valueFrom:
      fieldRef:
        fieldPath: metadata.name
  - name: POD_NAMESPACE
    valueFrom:
      fieldRef:
        fieldPath: metadata.namespace
  - name: POD_IP
    valueFrom:
      fieldRef:
        fieldPath: status.podIP
  readinessProbe:
    httpGet:
      path: "/_status/ready"
      port: 8080
    timeoutSeconds: 3
  resources:
{{ toYaml .Values.sidecar.resources | indent 4 }}
  lifecycle:
    preStop:
      exec:
        command:
        - "/couchbase-sidecar"
        - stop
  ports:
  - containerPort: 8080
    name: sidecar
  volumeMounts:
  - mountPath: "/sidecar"
    name: sidecar
{{- end -}}
