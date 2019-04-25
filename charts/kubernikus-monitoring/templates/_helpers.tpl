{{/* Generate additional scrape config. */}}
{{- define "scrapeConfig" -}}
{{- include (print $.Template.BasePath  "/_prometheus.yaml.tpl") . -}}
{{- if .Values.extraScrapeConfig -}}
{{- tpl .Values.extraScrapeConfig . -}}
{{- end -}}
{{- end -}}
