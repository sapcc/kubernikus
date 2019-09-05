groups:
- name: oomkill.alerts
  rules:
  - alert: PodOOMKilled
    expr: sum(changes(klog_pod_oomkill[24h]) > 0 or ((klog_pod_oomkill == 1) unless (klog_pod_oomkill offset 24h == 1))) by (namespace, pod_name)
    for: 5m
    labels:
      tier: {{ required ".Values.tier missing" .Values.tier }}
      service: resources
      severity: info
      context: memory
      meta: "{{`{{ $labels.namespace }}`}}/{{`{{ $labels.pod_name }}`}}"
    annotations:
      summary: Pod was oomkilled recently
      description: The pod {{`{{ $labels.namespace }}`}}/{{`{{ $labels.pod_name }}`}} was hit at least once by the oom killer within 24h

  - alert: PodConstantlyOOMKilled
    expr: sum(changes(klog_pod_oomkill[30m]) ) by (namespace, pod_name) > 2
    for: 5m
    labels:
      tier: {{ required ".Values.tier missing" .Values.tier }}
      service: resources
      severity: warning
      context: memory
      meta: "{{`{{ $labels.namespace }}`}}/{{`{{ $labels.pod_name }}`}}"
    annotations:
      summary: Pod was oomkilled more than 2 times in 30 minutes
      description: The pod {{`{{ $labels.namespace }}`}}/{{`{{ $labels.pod_name }}`}} killed several times in short succession. This could be due to wrong resource limits.

  - alert: PodOOMExceedingLimits
    expr: max(predict_linear(container_memory_working_set_bytes{pod_name=~".+"}[1h], 8 * 3600)) by (container_name, pod_name, namespace)  >  max(label_replace(label_replace(kube_pod_container_resource_limits_memory_bytes{pod=~".+"}, "pod_name", "$1", "pod", "(.*)"), "container_name", "$1", "container", "(.*)")) by (container_name, pod_name, namespace)
    for: 5m
    labels:
      tier: {{ required ".Values.tier missing" .Values.tier }}
      service: resources
      severity: info
      context: memory
      meta: "{{`{{ $labels.namespace }}`}}/{{`{{ $labels.pod_name }}`}}"
    annotations:
      summary: Pod will likely exceed memory limits in 8h
      description: The pod {{`{{ $labels.namespace }}`}}/{{`{{ $labels.pod_name }}`}} will exceed its memory limit in 8h.
