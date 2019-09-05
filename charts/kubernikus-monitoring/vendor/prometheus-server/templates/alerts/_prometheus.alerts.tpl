groups:
- name: prometheus.alerts
  rules:
  - alert: PrometheusFailedConfigReload
    expr: prometheus_config_last_reload_successful == 0
    for: 5m
    labels:
      context: availability
      service: prometheus
      severity: critical
      tier: {{ include "alerts.tier" . }}
      playbook: 'docs/support/playbook/prometheus/failed_config_reload.html'
      meta: 'Prometheus {{`{{ $externalLabels.region }}`}}/{{`{{ $labels.prometheus }}`}} failed to load it`s configuration.'
    annotations:
      description: 'Prometheus {{`{{ $externalLabels.region }}`}}/{{`{{ $labels.prometheus }}`}} failed to load it`s configuration. Prometheus cannot start with a malformed configuration.'
      summary: Prometheus configuration reload has failed

  - alert: PrometheusRuleEvaluationFailed
    expr: increase(prometheus_rule_evaluation_failures_total[5m]) > 0
    labels:
      context: availability
      service: prometheus
      severity: warning
      tier: {{ include "alerts.tier" . }}
      playbook: 'docs/support/playbook/prometheus/rule_evaluation.html'
      meta: 'Prometheus {{`{{ $externalLabels.region }}`}}/{{`{{ $labels.prometheus }}`}} failed to evaluate rules.'
    annotations:
      description: 'Prometheus {{`{{ $externalLabels.region }}`}}/{{`{{ $labels.prometheus }}`}} failed to evaluate rules. Aggregation or alerting rules may not be loaded or provide false results.'
      summary: Prometheus rule evaluation failed

  - alert: PrometheusRuleEvaluationSlow
    expr: prometheus_rule_evaluation_duration_seconds{quantile="0.9"} > 60
    for: 10m
    labels:
      context: availability
      service: prometheus
      severity: info
      tier: {{ include "alerts.tier" . }}
      playbook: 'docs/support/playbook/prometheus/rule_evaluation.html'
      meta: 'Prometheus {{`{{ $externalLabels.region }}`}}/{{`{{ $labels.prometheus }}`}} rule evaluation is slow.'
    annotations:
      description: 'Prometheus {{`{{ $externalLabels.region }}`}}/{{`{{ $labels.prometheus }}`}} rule evaluation is slow'
      summary: Prometheus rule evaluation is slow

  - alert: PrometheusWALCorruption
    expr: increase(prometheus_tsdb_wal_corruptions_total[5m]) > 0
    labels:
      context: availability
      service: prometheus
      severity: info
      tier: {{ include "alerts.tier" . }}
      playbook: 'docs/support/playbook/prometheus/wal.html'
      meta: 'Prometheus {{`{{ $externalLabels.region }}`}}/{{`{{ $labels.prometheus }}`}} has {{`{{ $value }}`}} WAL corruptions.'
    annotations:
      description: 'Prometheus {{`{{ $externalLabels.region }}`}}/{{`{{ $labels.prometheus }}`}}  has {{`{{ $value }}`}} WAL corruptions.'
      summary: Prometheus has WAL corruptions

  - alert: PrometheusTSDBReloadsFailing
    expr: increase(prometheus_tsdb_reloads_failures_total[2h]) > 0
    for: 12h
    labels:
      context: availability
      service: prometheus
      severity: info
      tier: {{ include "alerts.tier" . }}
      playbook: 'docs/support/playbook/prometheus/failed_tsdb_reload.html'
      meta: 'Prometheus {{`{{ $externalLabels.region }}`}}/{{`{{ $labels.prometheus }}`}} failed to reload TSDB.'
    annotations:
      description: 'Prometheus {{`{{ $externalLabels.region }}`}}/{{`{{ $labels.prometheus }}`}} had {{`{{$value | humanize}}`}} reload failures over the last four hours.'
      summary: Prometheus has issues reloading data blocks from disk

  - alert: PrometheusNotIngestingSamples
    expr: rate(prometheus_tsdb_head_samples_appended_total[5m]) <= 0
    for: 10m
    labels:
      context: availability
      service: prometheus
      severity: info
      tier: {{ include "alerts.tier" . }}
      playbook: 'docs/support/playbook/prometheus/failed_scrapes.html'
      meta: 'Prometheus {{`{{ $externalLabels.region }}`}}/{{`{{ $labels.prometheus }}`}} failed to evaluate rules.'
    annotations:
      description: 'Prometheus {{`{{ $externalLabels.region }}`}}/{{`{{ $labels.prometheus }}`}} failed to evaluate rules. Aggregation or alerting rules may not be loaded or provide false results.'
      summary: Prometheus rule evaluation failed

  - alert: PrometheusTargetScrapesDuplicate
    expr: increase(prometheus_target_scrapes_sample_duplicate_timestamp_total[5m]) > 0
    for: 10m
    labels:
      context: availability
      service: prometheus
      severity: info
      tier: {{ include "alerts.tier" . }}
      playbook: 'docs/support/playbook/prometheus/failed_scrapes.html'
      meta: 'Prometheus {{`{{ $externalLabels.region }}`}}/{{`{{ $labels.prometheus }}`}} rejects many samples'
    annotations:
      description: 'Prometheus {{`{{ $externalLabels.region }}`}}/{{`{{ $labels.prometheus }}`}} has many samples rejected due to duplicate timestamps but different values. This indicates metric duplication.'
      summary: Prometheus rejects many samples

  - alert: PrometheusLargeScrapes
    expr: increase(prometheus_target_scrapes_exceeded_sample_limit_total[30m]) > 60
    labels:
      context: availability
      service: prometheus
      severity: info
      tier: {{ include "alerts.tier" . }}
      playbook: 'docs/support/playbook/prometheus/failed_scrapes.html'
      meta: 'Prometheus {{`{{ $externalLabels.region }}`}}/{{`{{ $labels.prometheus }}`}} fails to scrape targets'
    annotations:
      description: 'Prometheus {{`{{ $externalLabels.region }}`}}/{{`{{ $labels.prometheus }}`}} has many scrapes that exceed the sample limit'
      summary: Prometheus fails to scrape targets.
