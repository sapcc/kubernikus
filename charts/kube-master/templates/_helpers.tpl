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
    {{- required "tag for dashboardProxy missing" .Values.images.dashboardProxy.tag }}
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
  {{- required "repository for kubelet missing" .Values.images.kubelet.repository }}:
    {{- required "tag for kubelet missing" .Values.images.kubelet.tag }}
{{- end -}}

{{- define "wormhole.image" }}
  {{- required "repository for wormhole missing" .Values.images.wormhole.repository }}:
    {{- required "tag for wormhole missing" .Values.version.kubernikus }}
{{- end -}}

{{- define "etcd.image" }}
  {{- required "repository for etcd missing" .Values.images.etcd.repository }}:
    {{- required "tag for etcd missing" .Values.images.etcd.tag }}
{{- end -}}

{{- define "etcdBackup.image" }}
  {{- required "repository for etcdBackup missing" .Values.images.etcdBackup.repository }}:
    {{- required "tag for etcdBackup missing" .Values.images.etcdBackup.tag }}
{{- end -}}

{{- define "cinderCSIPlugin.image" }}
  {{- required "repository for cinderCSIPlugin missing" .Values.images.cinderCSIPlugin.repository }}:
    {{- required "tag for cinderCSIPlugin missing" .Values.images.cinderCSIPlugin.tag }}
{{- end -}}

{{- define "csiProvisioner.image" }}
  {{- required "repository for csiProvisioner missing" .Values.images.csiProvisioner.repository }}:
    {{- required "tag for csiProvisioner missing" .Values.images.csiProvisioner.tag }}
{{- end -}}

{{- define "csiAttacher.image" }}
  {{- required "repository for csiAttacher missing" .Values.images.csiAttacher.repository }}:
    {{- required "tag for csiAttacher missing" .Values.images.csiAttacher.tag }}
{{- end -}}

{{- define "csiSnapshotter.image" }}
  {{- required "repository for csiSnapshotter missing" .Values.images.csiSnapshotter.repository }}:
    {{- required "tag for csiSnapshotter missing" .Values.images.csiSnapshotter.tag }}
{{- end -}}

{{- define "csiResizer.image" }}
  {{- required "repository for csiResizer missing" .Values.images.csiResizer.repository }}:
    {{- required "tag for csiResizer missing" .Values.images.csiResizer.tag }}
{{- end -}}

{{- define "csiLivenessProbe.image" }}
  {{- required "repository for csiLivenessProbe missing" .Values.images.csiLivenessProbe.repository }}:
    {{- required "tag for csiLivenessProbe missing" .Values.images.csiLivenessProbe.tag }}
{{- end -}}

{{- define "csiSnapshotController.image" }}
  {{- required "repository for csiSnapshotController missing" .Values.images.csiSnapshotController.repository }}:
    {{- required "tag for csiSnapshotController missing" .Values.images.csiSnapshotController.tag }}
{{- end -}}

{{- define "dashboard.url" -}}
  {{- printf "dashboard-%s" ( .Values.api.apiserverHost | replace (include "master.fullname" .) (printf "%s.ingress" (include "master.fullname" .) ) ) -}}
{{- end -}}

{{- define "dex.url" -}}
  {{- printf "auth-%s" ( .Values.api.apiserverHost | replace (include "master.fullname" .) (printf "%s.ingress" (include "master.fullname" .) ) ) -}}
{{- end -}}
