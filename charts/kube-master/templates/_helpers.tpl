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
{{- $images := required "imagesForVersion undefined" .Values.imagesForVersion}}
{{- $version := required "version.kubernetes undefined" .Values.version.kubernetes }}
{{- $imagesForVersion := required (printf "unsupported kubernetes version %s" $version) (index $images $version) }}
{{- $hyperkube := required (printf "No hyperkube image found for version %s" $version) (index $imagesForVersion "hyperkube") }}
{{- required (printf "repository for hyperkube missing for version %s" $version) $hyperkube.repository }}:
  {{- required (printf "tag for hyperkube missing for version %s" $version) $hyperkube.tag }}
{{- end -}}

{{- define "cloudControllerManager.image" }}
{{- $images := required "imagesForVersion undefined" .Values.imagesForVersion}}
{{- $version := required "version.kubernetes undefined" .Values.version.kubernetes }}
{{- $imagesForVersion := required (printf "unsupported kubernetes version %s" $version) (index $images $version) }}
{{- $cloudControllerManager := required (printf "No cloudControllerManager image found for version %s" $imagesForVersion) (index $imagesForVersion "cloudControllerManager") }}
{{- required (printf "repository for cloudControllerManager missing for version %s" $version) $cloudControllerManager.repository }}:
  {{- required (printf "tag for cloudControllerManager missing for version %s" $version) $cloudControllerManager.tag }}
{{- end -}}