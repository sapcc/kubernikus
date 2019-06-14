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
