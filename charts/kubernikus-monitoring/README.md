kubernikus-monitoring
---------------------

This is an umbrella chart for monitoring Kubernikus.

Prometheus alert and aggregation rules are provided in the respective subfolder and are deployed as a [PrometheusRule](https://github.com/coreos/prometheus-operator/blob/master/Documentation/api.md#prometheusrule).  
Rules are mapped to the respective Prometheus server instance via labels. Per convention a kubernikus Prometheus rule must have the following label:
```
metadata:
  labels:
    prometheus: kubernikus
```
