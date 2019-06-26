### Alerts specific to our baremetal controlplane and not applicable for the Kubernikus controlplane.
### TODO: Generalize.
groups:
- name: controlplane.alerts
  rules:
  ### Node Labels ###
  - alert: KubernetesNodeLabelMissingZone
    expr: sum(kube_node_labels{label_zone!~"farm|petting-zoo"}) by (node,label_zone) > 0
    for: 15m
    labels:
      tier: {{ required ".Values.tier missing" .Values.tier }}
      service: node
      severity: warning
      context: label
      meta: "{{`{{ $labels.node }}`}}"
      playbook: "docs/support/playbook/kubernetes/k8s_label_missing.html"
    annotations:
      description: Node {{`{{ $labels.node }}`}} is missing the correct zone label. It is currently set to zone='{{`{{ $labels.label_zone }}`}}' Possible scheduling issues.
      summary: Node {{`{{ $labels.node }}`}} is missing the correct zone label.

  - alert: KubernetesNodeLabelMissingSpecies
    expr: sum(kube_node_labels{node=~"storage.*", label_species!="swift-storage"}) by (node,label_species) > 0 OR sum(kube_node_labels{node=~"network.*", label_species!="network"}) by (node,label_species) > 0 OR sum(kube_node_labels{node=~"master.*", label_species!="master"}) by (node,label_species) > 0
    for: 15m
    labels:
      tier: {{ required ".Values.tier missing" .Values.tier }}
      service: node
      severity: warning
      context: label
      meta: "{{`{{ $labels.node }}`}}"
      playbook: "docs/support/playbook/kubernetes/k8s_label_missing.html"
    annotations:
      description: Node {{`{{ $labels.node }}`}} is missing the correct species label. It is currently set to species='{{`{{ $labels.label_species }}`}}' Possible scheduling issues.
      summary: Node {{`{{ $labels.node }}`}} is missing the correct species label.

  ### Node Taints ###

  - alert: KubernetesNodeTaintMissing
    expr: sum(kube_node_spec_taint{node=~"storage.*", value!~"swift-storage.*"}) by (node,value) > 0 OR sum(kube_node_spec_taint{node=~"network.*", value!~"network|alien"}) by (node,value) > 0
    for: 15m
    labels:
      tier: {{ required ".Values.tier missing" .Values.tier }}
      service: node
      severity: warning
      context: label
      meta: "{{`{{ $labels.node }}`}}"
      playbook: "docs/support/playbook/kubernetes/k8s_taint_missing.html"
    annotations:
      description: Node {{`{{ $labels.node }}`}} is missing the correct taint. It is currently set to value='{{`{{ $labels.value}}`}}' Possible scheduling issues.
      summary: Node {{`{{ $labels.node }}`}} is missing the correct taint label.


  ### Scheduler alerts ###

  - alert: KubernetesSchedulerDown
    expr: count(up{job="kube-system/scheduler"} == 1) == 0
    for: 5m
    labels:
      tier: {{ required ".Values.tier missing" .Values.tier }}
      service: k8s
      severity: critical
      context: scheduler
      dashboard: kubernetes-health
    annotations:
      summary: Scheduler is down
      description: No scheduler is running. New pods are not being assigned to nodes!

  - alert: KubernetesSchedulerScrapeMissing
    expr: absent(up{job="kube-system/scheduler"})
    for: 1h
    labels:
      tier: {{ required ".Values.tier missing" .Values.tier }}
      service: k8s
      severity: warning
      context: scheduler
      dashboard: kubernetes-health
    annotations:
      description: Scheduler in failed to be scraped
      summary: Scheduler cannot be scraped


  ### ControllerManager alerts ###

  - alert: KubernetesControllerManagerDown
    expr: count(up{job="kube-system/controller-manager"} == 1) == 0
    for: 5m
    labels:
      tier: {{ required ".Values.tier missing" .Values.tier }}
      service: k8s
      severity: critical
      context: controller-manager
      dashboard: kubernetes-health
    annotations:
      description: No controller-manager is running. Deployments and replication controllers are not making progress
      summary: Controller manager is down

  - alert: KubernetesControllerManagerScrapeMissing
    expr: absent(up{job="kube-system/controller-manager"})
    for: 1h
    labels:
      tier: {{ required ".Values.tier missing" .Values.tier }}
      service: k8s
      severity: warning
      context: controller-manager
      dashboard: kubernetes-health
    annotations:
      description: ControllerManager failed to be scraped
      summary: ControllerManager cannot be scraped
