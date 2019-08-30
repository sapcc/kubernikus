- job_name: 'endpoints'
  kubernetes_sd_configs:
  - role: endpoints
  relabel_configs:
  - action: keep
    source_labels: [__meta_kubernetes_service_annotation_prometheus_io_scrape]
    regex: true
  - action: keep
    source_labels: [__meta_kubernetes_pod_container_port_number, __meta_kubernetes_pod_container_port_name, __meta_kubernetes_service_annotation_prometheus_io_port]
    regex: (9102;.*;.*)|(.*;metrics;.*)|(.*;.*;\d+)
  - source_labels: [__meta_kubernetes_service_annotation_prometheus_io_scheme]
    target_label: __scheme__
    regex: (https?)
  - source_labels: [__meta_kubernetes_service_annotation_prometheus_io_path]
    target_label: __metrics_path__
    regex: (.+)
  - source_labels: [__address__, __meta_kubernetes_service_annotation_prometheus_io_port]
    target_label: __address__
    regex: ([^:]+)(?::\d+);(\d+)
    replacement: $1:$2
  - action: labelmap
    regex: __meta_kubernetes_service_label_(.+)
  - source_labels: [__meta_kubernetes_namespace]
    target_label: kubernetes_namespace
  - source_labels: [__meta_kubernetes_service_name]
    target_label: kubernetes_name
{{ include "prometheus.external-labels.relabel-config" . | indent 2 }}

- job_name: 'pods'
  kubernetes_sd_configs:
  - role: pod
  relabel_configs:
  - action: keep
    source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
    regex: true
  - action: keep
    source_labels: [__meta_kubernetes_pod_container_port_number, __meta_kubernetes_pod_container_port_name, __meta_kubernetes_pod_annotation_prometheus_io_port]
    regex: (9102;.*;.*)|(.*;metrics;.*)|(.*;.*;\d+)
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_path]
    target_label: __metrics_path__
    regex: (.+)
  - source_labels: [__address__, __meta_kubernetes_pod_annotation_prometheus_io_port]
    target_label: __address__
    regex: ([^:]+)(?::\d+)?;(\d+)
    replacement: ${1}:${2}
  - action: labelmap
    regex: __meta_kubernetes_pod_label_(.+)
  - source_labels: [__meta_kubernetes_namespace]
    target_label: kubernetes_namespace
  - source_labels: [__meta_kubernetes_pod_name]
    target_label: kubernetes_pod_name
{{ include "prometheus.external-labels.relabel-config" . | indent 2 }}

- job_name: 'kube-system/dnsmasq'
  kubernetes_sd_configs:
  - role: pod
  relabel_configs:
  - action: keep
    source_labels: [__meta_kubernetes_namespace]
    regex: kube-system
  - action: keep
    source_labels: [__meta_kubernetes_pod_name]
    regex: (kube-dns[^\.]+).+
  - source_labels: [__address__]
    target_label: __address__
    regex: ([^:]+)(:\d+)?
    replacement: ${1}:10054
  - target_label: component
    replacement: dnsmasq
  - action: replace
    source_labels: [__meta_kubernetes_pod_node_name]
    target_label: instance
{{ include "prometheus.external-labels.relabel-config" . | indent 2 }}

- job_name: 'kube-system/dns'
  kubernetes_sd_configs:
  - role: pod
  relabel_configs:
  - action: keep
    source_labels: [__meta_kubernetes_namespace]
    regex: kube-system
  - action: keep
    source_labels: [__meta_kubernetes_pod_name]
    regex: (kube-dns[^\.]+).+
  - source_labels: [__address__]
    target_label: __address__
    regex: ([^:]+)(:\d+)?
    replacement: ${1}:10055
  - target_label: component
    replacement: dns
  - action: replace
    source_labels: [__meta_kubernetes_pod_node_name]
    target_label: instance
{{ include "prometheus.external-labels.relabel-config" . | indent 2 }}

- job_name: 'kubernikus-system/node'
  kubernetes_sd_configs:
  - role: node
  relabel_configs:
  - action: labelmap
    regex: __meta_kubernetes_node_label_(.+)
  - target_label: component
    replacement: node
  - action: replace
    source_labels: [__meta_kubernetes_node_name]
    target_label: instance
  - source_labels: [__address__]
    target_label: __address__
    regex: ([^:]+)(:\d+)?
    replacement: ${1}:9100
  - source_labels: [mountpoint]
    target_label: mountpoint
    regex: '\/host(\/.*)'
    action: replace
    replacement: ${1}
{{ include "prometheus.external-labels.relabel-config" . | indent 2 }}

- job_name: kube-system/kubelet
  scheme: https
  kubernetes_sd_configs:
  - role: node
  bearer_token_file: /var/run/secrets/kubernetes.io/serviceaccount/token
  tls_config:
    ca_file: /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
    insecure_skip_verify: true
  relabel_configs:
  - separator: ;
    regex: __meta_kubernetes_node_label_(.+)
    replacement: $1
    action: labelmap
  - separator: ;
    regex: (.*)
    target_label: component
    replacement: kubelet
    action: replace
  - source_labels: [__meta_kubernetes_node_name]
    separator: ;
    regex: (.*)
    target_label: instance
    replacement: $1
    action: replace
{{ include "prometheus.external-labels.relabel-config" . | indent 2 }}
  metric_relabel_configs:
    - source_labels: [ id ]
      action: replace
      regex: ^/system\.slice/(.+)\.service$
      target_label: systemd_service_name
      replacement: '${1}'
    - source_labels: [ id ]
      action: replace
      regex: ^/system\.slice/(.+)\.service$
      target_label: container_name
      replacement: '${1}'
{{ include "prometheus.keep-metrics.metric-relabel-config" .Values.allowedMetrics.kubelet | indent 4 }}
    - source_labels:
      - container_name
      - __name__
      # The system container POD is used for networking.
      regex: POD;({{ .Values.allowedMetrics.kubelet | join "|" }})
      action: drop

- job_name: 'kubernetes-cadvisors'
  scheme: https
  metrics_path: /metrics/cadvisor
  tls_config:
    ca_file: /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
    insecure_skip_verify: true
  bearer_token_file: /var/run/secrets/kubernetes.io/serviceaccount/token
  kubernetes_sd_configs:
    - role: node
  relabel_configs:
    - action: labelmap
      regex: __meta_kubernetes_node_label_(.+)
{{ include "prometheus.external-labels.relabel-config" . | indent 4 }}
  metric_relabel_configs:
    - source_labels: [ id ]
      action: replace
      regex: ^/system\.slice/(.+)\.service$
      target_label: systemd_service_name
      replacement: '${1}'
    - source_labels: [ id ]
      action: replace
      regex: ^/system\.slice/(.+)\.service$
      target_label: container_name
      replacement: '${1}'
{{ include "prometheus.keep-metrics.metric-relabel-config" .Values.allowedMetrics.cAdvisor | indent 4 }}
    - source_labels:
      - container_name
      - __name__
      # The system container POD is used for networking.
      regex: POD;({{ without .Values.allowedMetrics.cAdvisor "container_network_receive_bytes_total" "container_network_transmit_bytes_total" | join "|" }})
      action: drop
    - source_labels: [ container_name ]
      regex: ^$
      action: drop
    - regex: ^id$
      action: labeldrop

- job_name: 'kube-system/apiserver'
  tls_config:
    ca_file: /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
    insecure_skip_verify: true
  bearer_token_file: /var/run/secrets/kubernetes.io/serviceaccount/token
  scheme: https
  static_configs:
  - targets:
    - $(KUBERNETES_SERVICE_HOST)
  relabel_configs:
    - target_label: component
      replacement: apiserver
{{ include "prometheus.external-labels.relabel-config" . | indent 4 }}
  metric_relabel_configs:
{{ include "prometheus.keep-metrics.metric-relabel-config" .Values.allowedMetrics.kubeAPIServer | indent 4 }}
