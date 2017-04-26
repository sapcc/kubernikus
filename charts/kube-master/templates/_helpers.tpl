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
{{- define "fullname" -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}


{{- define "certItemsFromFlags"}}
{{- range $value := . }}
{{- range $value := splitList "," (toString $value)}}
{{- if contains "/etc/kubernetes/certs/" $value }}
- key: {{ trimPrefix "/etc/kubernetes/certs/" $value }}
  path: {{ trimPrefix "/etc/kubernetes/certs/" $value }}
{{- end }}
{{- end }}
{{- end }}
{{- end}}
