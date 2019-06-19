{{/* Name of the Prometheus. */}}
{{- define "prometheus.name" -}}
{{- required ".Values.name missing" .Values.name -}}
{{- end -}}

{{/* Fullname of the Prometheus. */}}
{{- define "prometheus.fullName" -}}
prometheus-{{- (include "prometheus.name" .) -}}
{{- end -}}

{{/* External URL of this Prometheus. */}}
{{- define "prometheus.externalURL" -}}
{{- if .Values.ingress.hostNameOverride -}}
{{- .Values.ingress.hostNameOverride -}}
{{- else -}}
{{- required ".Values.ingress.host missing" .Values.ingress.host -}}.{{- required ".Values.global.region missing" .Values.global.region -}}.{{- required ".Values.global.domain missing" .Values.global.domain -}}
{{- end -}}
{{- end -}}

{{/* Prometheus image. */}}
{{- define "prometheus.image" -}}
{{- required ".Values.image.repository missing" .Values.image.repository -}}:{{- required ".Values.image.tag missing" .Values.image.tag -}}
{{- end -}}

{{/* Name of the PVC. */}}
{{- define "pvc.name" -}}
{{- default .Values.name .Values.persistence.name | quote -}}
{{- end -}}

{{/* The name of the serviceAccount. */}}
{{- define "serviceAccount.name" -}}
{{- if .Values.serviceAccount.create -}}
{{- default (include "prometheus.fullName" . ) .Values.serviceAccount.name -}}
{{- else -}}
{{- default "default" .Values.serviceAccount.name -}}
{{- end -}}
{{- end -}}

{{/* Alertmanager configuration. */}}
{{- define "alertmanager.config" -}}
- scheme: https
  timeout: 10s
  static_configs:
  - targets:
{{ toYaml .Values.alertmanagers | indent 6 }}
{{- end -}}

{{/* Thanos image. */}}
{{- define "thanos.image" -}}
{{- if .Values.thanos.spec.image -}}
{{- .Values.thanos.spec.image -}}
{{- else -}}
{{- required ".Values.thanos.spec.baseImage missing" .Values.thanos.spec.baseImage -}}:{{- required ".Values.thanos.spec.version missing" .Values.thanos.spec.version -}}
{{- end -}}
{{- end -}}

{{- define "thanos.peers" -}}
{{- if .Values.thanos.spec.peers -}}
{{- .Values.thanos.spec.peers -}}
{{- else -}}
thanos-peers.{{ .Release.Namespace }}.svc:10900
{{- end -}}
{{- end -}}

{{- define "thanos.objectStorageConfig.name" -}}
{{- if and .Values.thanos.spec.objectStorageConfig -}}
{{- required ".Values.thanos.spec.objectStorageConfig.name missing" .Values.thanos.spec.objectStorageConfig.name -}}
{{- else -}}
{{- include "prometheus.fullName" . -}}-{{- required ".Values.thanos.objectStorageConfig.name missing" .Values.thanos.objectStorageConfig.name -}}
{{- end -}}
{{- end -}}

{{- define "thanos.objectStorageConfig.key" -}}
{{- if .Values.thanos.spec.objectStorageConfig -}}
{{- required ".Values.thanos.spec.objectStorageConfig.key missing" .Values.thanos.spec.objectStorageConfig.key -}}
{{- else -}}
{{- required ".Values.thanos.objectStorageConfig.key missing" .Values.thanos.objectStorageConfig.key -}}
{{- end -}}
{{- end -}}

{{- define "thanos.projectName" -}}
{{- if .Values.thanos.swiftStorageConfig.tenantName }}
{{- .Values.thanos.swiftStorageConfig.tenantName | quote -}}
{{- else -}}
{{- required ".Values.thanos.swiftStorageConfig.projectName missing" .Values.thanos.swiftStorageConfig.projectName | quote -}}
{{- end -}}
{{- end -}}

{{- define "thanos.projectDomainName" -}}
{{- if .Values.thanos.swiftStorageConfig.projectDomainName -}}
{{- .Values.thanos.swiftStorageConfig.projectDomainName | quote -}}
{{- else -}}
{{- required ".Values.thanos.swiftStorageConfig.domainName missing" .Values.thanos.swiftStorageConfig.domainName | quote -}}
{{- end -}}
{{- end -}}

{{/* Value for prometheus.io/targets annotation. */}}
{{- define "prometheusTargetsValue" -}}
{{- $value := printf ".*%s.*" (include "prometheus.name" . ) -}}
{{- if .Values.serviceDiscoveries.additionalTargets -}}
{{- $value -}}|.*{{- .Values.serviceDiscoveries.additionalTargets | join ".*|.*" -}}.*
{{- else -}}
{{- $value -}}
{{- end -}}
{{- end -}}
