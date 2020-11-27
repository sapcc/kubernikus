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

{{- define "hyperkube.image" }}
  {{- required "repository for hyperkube missing" .Values.images.hyperkube.repository }}:
    {{- required "tag for hyperkube missing" .Values.images.hyperkube.tag }}
{{- end -}}

{{- define "cloudControllerManager.image" }}
{{- required "repository for cloudControllerManager missing" .Values.images.cloudControllerManager.repository }}:
  {{- required "tag for cloudControllerManager missing" .Values.images.cloudControllerManager.tag }}
{{- end -}}

{{- define "dex.image" }}
{{- required "repository for dex missing" .Values.images.dex.repository }}:
  {{- required "tag for dex missing" .Values.images.dex.tag }}
{{- end -}}

{{- define "dashboard.image" }}
{{- required "repository for dashboard missing" .Values.images.dashboard.repository }}:
  {{- required "tag for dashboard missing" .Values.images.dashboard.tag }}
{{- end -}}

{{- define "dashboardProxy.image" }}
{{- required "repository for dashboardProxy missing" .Values.images.dashboardProxy.repository }}:
  {{- required "tag for dashboardProxy missing" .Vaules.images.dashboardProxy.tag }}
{{- end -}}

{{- define "apiserver.image" }}
  {{- required "repository for apiserver missing" .Values.images.apiserver.repository }}:
    {{- required "tag for apiserver missing" .Values.images.apiserver.tag }}
{{- end -}}

{{- define "controllerManager.image" }}
  {{- required "repository for controllerManager missing" .Values.images.controllerManager.repository }}:
    {{- required "tag for controllerManager missing" .Values.images.controllerManager.tag }}
{{- end -}}

{{- define "scheduler.image" }}
  {{- required "repository for scheduler missing" .Values.images.scheduler.repository }}:
    {{- required "tag for scheduler missing" .Values.images.scheduler.tag }}
{{- end -}}

{{- define "kubelet.image" }}
  {{- required (printf "repository for kubelet missing" .Values.images.kubelet.repository) }}:
    {{- required (printf "tag for kubelet missing" .Values.images.kubelet.tag) }}
{{- end -}}

{{- define "dashboard.url" -}}
{{- printf "dashboard-%s" ( .Values.api.apiserverHost | replace (include "master.fullname" .) (printf "%s.ingress" (include "master.fullname" .) ) ) -}}
{{- end -}}

{{- define "dex.url" -}}
{{- printf "auth-%s" ( .Values.api.apiserverHost | replace (include "master.fullname" .) (printf "%s.ingress" (include "master.fullname" .) ) ) -}}
{{- end -}}
