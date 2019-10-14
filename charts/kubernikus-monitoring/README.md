kubernikus-monitoring
---------------------

This chart contains Prometheus alert and aggregation rules and Grafana dashboards and datasources for Kubernikus.  
The underlying infrastructure (Prometheus, exporters, Grafana, etc.) are deployed via [kube-monitoring-kubernikus Helm chart](https://github.com/sapcc/helm-charts/tree/master/system/kube-monitoring-kubernikus).

## Prometheus

Prometheus alert and aggregation rules are provided in the respective subfolder and are deployed as a [PrometheusRule](https://github.com/coreos/prometheus-operator/blob/master/Documentation/api.md#prometheusrule).  
Rules are mapped to the respective Prometheus server instance via labels. Per convention a kubernikus Prometheus rule must have the following label:
```
metadata:
  labels:
    prometheus: kubernikus
```

## Grafana

Grafana dashboards and datasources are deployed via ConfigMaps and considered if they are labeled as shown below:

Dashboards:
```
metadata:
  labels:
    kubernikus-grafana-dashboard: "true"
```

Datasources:
```
metadata:
  labels:
    kubernikus-grafana-datasource: "true"
```
