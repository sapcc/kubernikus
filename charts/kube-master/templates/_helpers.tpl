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
{{- if (semverCompare "< 1.19" .Values.version.kubernetes) -}}
  {{- $images := required "imagesForVersion undefined" .Values.imagesForVersion}}
  {{- $version := required "version.kubernetes undefined" .Values.version.kubernetes }}
  {{- $imagesForVersion := required (printf "unsupported kubernetes version %s" $version) (index $images $version) }}
  {{- $hyperkube := required (printf "No hyperkube image found for version %s" $version) (index $imagesForVersion "hyperkube") }}
  {{- required (printf "repository for hyperkube missing for version %s" $version) $hyperkube.repository }}:
    {{- required (printf "tag for hyperkube missing for version %s" $version) $hyperkube.tag }}
{{- end -}}
{{- end -}}

{{- define "cloudControllerManager.image" }}
{{- $images := required "imagesForVersion undefined" .Values.imagesForVersion}}
{{- $version := required "version.kubernetes undefined" .Values.version.kubernetes }}
{{- $imagesForVersion := required (printf "unsupported kubernetes version %s" $version) (index $images $version) }}
{{- $cloudControllerManager := required (printf "No cloudControllerManager image found for version %s" $imagesForVersion) (index $imagesForVersion "cloudControllerManager") }}
{{- required (printf "repository for cloudControllerManager missing for version %s" $version) $cloudControllerManager.repository }}:
  {{- required (printf "tag for cloudControllerManager missing for version %s" $version) $cloudControllerManager.tag }}
{{- end -}}

{{- define "dex.image" }}
{{- $images := required "imagesForVersion undefined" .Values.imagesForVersion}}
{{- $version := required "version.kubernetes undefined" .Values.version.kubernetes }}
{{- $imagesForVersion := required (printf "unsupported kubernetes version %s" $version) (index $images $version) }}
{{- $dex := required (printf "No dex image found for version %s" $imagesForVersion) (index $imagesForVersion "dex") }}
{{- required (printf "repository for dex missing for version %s" $version) $dex.repository }}:
  {{- required (printf "tag for dex missing for version %s" $version) $dex.tag }}
{{- end -}}

{{- define "dashboard.image" }}
{{- $images := required "imagesForVersion undefined" .Values.imagesForVersion}}
{{- $version := required "version.kubernetes undefined" .Values.version.kubernetes }}
{{- $imagesForVersion := required (printf "unsupported kubernetes version %s" $version) (index $images $version) }}
{{- $dashboard := required (printf "No dashboard image found for version %s" $imagesForVersion) (index $imagesForVersion "dashboard") }}
{{- required (printf "repository for dashboard missing for version %s" $version) $dashboard.repository }}:
  {{- required (printf "tag for dashboard missing for version %s" $version) $dashboard.tag }}
{{- end -}}

{{- define "dashboardProxy.image" }}
{{- $images := required "imagesForVersion undefined" .Values.imagesForVersion}}
{{- $version := required "version.kubernetes undefined" .Values.version.kubernetes }}
{{- $imagesForVersion := required (printf "unsupported kubernetes version %s" $version) (index $images $version) }}
{{- $dashboardProxy := required (printf "No dashboardProxy image found for version %s" $imagesForVersion) (index $imagesForVersion "dashboardProxy") }}
{{- required (printf "repository for dashboardProxy missing for version %s" $version) $dashboardProxy.repository }}:
  {{- required (printf "tag for dashboardProxy missing for version %s" $version) $dashboardProxy.tag }}
{{- end -}}

{{- define "apiserver.image" }}
{{- if (semverCompare ">= 1.19" .Values.version.kubernetes) -}}
  {{- $images := required "imagesForVersion undefined" .Values.imagesForVersion}}
  {{- $version := required "version.kubernetes undefined" .Values.version.kubernetes }}
  {{- $imagesForVersion := required (printf "unsupported kubernetes version %s" $version) (index $images $version) }}
  {{- $apiserver := required (printf "No apiserver image found for version %s" $version) (index $imagesForVersion "apiserver") }}
  {{- required (printf "repository for apiserver missing for version %s" $version) $apiserver.repository }}:
    {{- required (printf "tag for apiserver missing for version %s" $version) $apiserver.tag }}
{{- end -}}
{{- end -}}

{{- define "controllerManager.image" }}
{{- if (semverCompare ">= 1.19" .Values.version.kubernetes) -}}
  {{- $images := required "imagesForVersion undefined" .Values.imagesForVersion}}
  {{- $version := required "version.kubernetes undefined" .Values.version.kubernetes }}
  {{- $imagesForVersion := required (printf "unsupported kubernetes version %s" $version) (index $images $version) }}
  {{- $controllerManager := required (printf "No controllerManager image found for version %s" $version) (index $imagesForVersion "controllerManager") }}
  {{- required (printf "repository for controllerManager missing for version %s" $version) $controllerManager.repository }}:
    {{- required (printf "tag for controllerManager missing for version %s" $version) $controllerManager.tag }}
{{- end -}}
{{- end -}}

{{- define "scheduler.image" }}
{{- if (semverCompare ">= 1.19" .Values.version.kubernetes) -}}
  {{- $images := required "imagesForVersion undefined" .Values.imagesForVersion}}
  {{- $version := required "version.kubernetes undefined" .Values.version.kubernetes }}
  {{- $imagesForVersion := required (printf "unsupported kubernetes version %s" $version) (index $images $version) }}
  {{- $scheduler := required (printf "No scheduler image found for version %s" $version) (index $imagesForVersion "scheduler") }}
  {{- required (printf "repository for scheduler missing for version %s" $version) $scheduler.repository }}:
    {{- required (printf "tag for scheduler missing for version %s" $version) $scheduler.tag }}
{{- end -}}
{{- end -}}

{{- define "kubelet.image" }}
{{- if (semverCompare ">= 1.19" .Values.version.kubernetes) -}}
  {{- $images := required "imagesForVersion undefined" .Values.imagesForVersion}}
  {{- $version := required "version.kubernetes undefined" .Values.version.kubernetes }}
  {{- $imagesForVersion := required (printf "unsupported kubernetes version %s" $version) (index $images $version) }}
  {{- $kubelet := required (printf "No kubelet image found for version %s" $version) (index $imagesForVersion "kubelet") }}
  {{- required (printf "repository for kubelet missing for version %s" $version) $kubelet.repository }}:
    {{- required (printf "tag for kubelet missing for version %s" $version) $kubelet.tag }}
{{- end -}}
{{- end -}}

{{- define "dashboard.url" -}}
{{- printf "dashboard-%s" ( .Values.api.apiserverHost | replace (include "master.fullname" .) (printf "%s.ingress" (include "master.fullname" .) ) ) -}}
{{- end -}}

{{- define "dex.url" -}}
{{- printf "auth-%s" ( .Values.api.apiserverHost | replace (include "master.fullname" .) (printf "%s.ingress" (include "master.fullname" .) ) ) -}}
{{- end -}}
