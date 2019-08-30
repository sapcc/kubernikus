{{/* Generate additional scrape config. */}}
{{- define "scrapeConfig" -}}
{{- include (print $.Template.BasePath  "/_prometheus.yaml.tpl") . -}}
{{- if .Values.extraScrapeConfig -}}
{{- tpl .Values.extraScrapeConfig . -}}
{{- end -}}
{{- end -}}

{{- define "prometheus.keep-metrics.metric-relabel-config" -}}
- source_labels: [ __name__ ]
  regex: ^({{ . | join "|" }})$
  action: keep
{{- end -}}

{{- define "prometheus.external-labels.relabel-config" -}}
- action: replace
  target_label: region
  replacement: {{ required ".Values.global.region missing" .Values.global.region }}
- action: replace
  target_label: cluster
  replacement: {{ required ".Values.global.cluster missing" .Values.global.cluster }}
{{- end -}}
