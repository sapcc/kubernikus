{{- define "dex.url" -}}
{{- printf "*.%s.%s.%s" .Values.dex.dns.zone .Values.global.region .Values.global.tld -}}
{{- end -}}
