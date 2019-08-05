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


**TODO:**  
This chart uses the Kubernetes Prometheus alert- and aggregation rules as deployed in the controlplane via [kube-rules](https://github.com/sapcc/helm-charts/tree/master/system/kube-monitoring/charts/kube-rules) sub-chart.
Re-use the entire [kube-monitoring chart](https://github.com/sapcc/helm-charts/tree/master/system/kube-monitoring) once all openstack and infrastructure parts have been cleansed from it.
