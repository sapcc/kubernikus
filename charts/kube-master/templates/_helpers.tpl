{{/* vim: set filetype=gotexttmpl: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
*/}}
{{- define "master.fullname" -}}
{{- .Release.Name | trunc 63 -}}
{{- end -}}

{{- define "etcd.fullname" -}}
{{- $name := default "etcd" .Values.etcd.nameOverride -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Get filenames of certificates referenced in given flags map seperated by space.
*/}}
{{- define "certNamesFromFlags" }}
{{- range . }}
{{- range toString . | splitList ","  }}
{{- if contains "/etc/kubernetes/certs/" . }} {{ trimPrefix "/etc/kubernetes/certs/" . }}{{ end }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Generate secret item selection for certificates referenced in flags
*/}}
{{- define "certItemsFromFlags"}}
{{- range include "certNamesFromFlags" . | trim | splitList " " }}
- key: {{ . }}
  path: {{ . }}
{{- end }}
{{- end }}

{{/*
Space seperate list of all certificates referenced in flags
*/}}
{{- define "allCerts" }}
{{- $apiCerts := include "certNamesFromFlags" .Values.api.flags | trim }}
{{- $controllerManagerCerts := include "certNamesFromFlags" .Values.controllerManager.flags | trim }}
{{- cat $apiCerts $controllerManagerCerts }}
{{- end}}
