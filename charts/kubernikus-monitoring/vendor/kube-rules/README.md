Kube rules
----------------

This chart is a collection of Prometheus alerting and aggregation rules for Kubernetes.  

## Configuration

The following table provides an overview of configurable parameters of this chart and their defaults.  
See the [values.yaml](./values.yaml) for more details.  

|       Parameter                        |           Description                                                                                                   |                         Default                     |
|----------------------------------------|-------------------------------------------------------------------------------------------------------------------------|-----------------------------------------------------|
| `prometheusName`                       | Name of the Prometheus to which the rules should be assigned to.                                                        | `""`                                                |
| `prometheusCollectorName`              | Optional name of the Prometheus collector instance. Only required if the collector -> frontend pattern is used.         | `""`                                                |
