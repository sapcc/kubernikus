Prometheus Chart
----------------

This chart shall serve as a template to deploy your own Prometheus.  
It will set up:
- Prometheus server instance (with 2 sidecars: configmap, rules reload)
- Persistent volume claim (if configured; default off)
- Ingress to manage external access for the Prometheus instance. (default off)
- RBAC resources for the Prometheus. Might be required if e.g. when service discoveries are used. (default off)

## Prerequisite

This chart relies on resources brought to you by the [Prometheus Operator](https://github.com/coreos/prometheus-operator).  
It may be installed using the [official helm chart](https://github.com/helm/charts/tree/master/stable/prometheus-operator).

## Scrape configuration

A default scrape configuration ifor service, endpoint and pod service discovery is part of this chart and deployed via [ServiceMonitors](./templates/servicemonitors),
[PodMonitors](./templates/podmonitors) and via [configuration](./examples/prometheus.yaml).
It can be disabled via values.  
See the official specification on [ServiceMonitor](https://github.com/coreos/prometheus-operator/blob/master/Documentation/api.md#servicemonitor) 
and [PodMonitor](https://github.com/coreos/prometheus-operator/blob/master/Documentation/api.md#podmonitor) for additional information.  

### Providing additional scrape configurations

Additional scrape configuration is provided via a secret referenced by the `additionalScrapeConfigs.name` and `additionalScrapeConfigs.key` parameters.  
See the `prometheus.yaml` and `additional-scrape-config.yaml` in the [examples](./examples) folder.

## Aggregation and Alerting rules

Aggregation and alerting rules can be deployed independently of the Prometheus server instance using the `PrometheusRule` CRD.  
An example can be found [here](./examples/kubernetes-health.alerts.yaml).  
Rules are assigned to a Prometheus instance by setting labels on the PrometheusRule as shown below. Refers to the `name` of the Prometheus as describe above.
```
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule

metadata:
  labels:
    prometheus: < name of the prometheus to which the rule should be assigned to >
  ...
```

## Integrate custom Service Discovery (SD)

To integrate a [custom SD](https://prometheus.io/blog/2018/07/05/implementing-custom-sd/#implementing-custom-service-discovery) one needs to provide the scrape configuration
```
scrape_configs:
  - job_name: "custom-sd"
    scrape_interval: "15s"
    file_sd_configs:
    - files:
      - /etc/prometheus/config/custom_sd.json
```

and have the Kubernetes configmap containing the `custom_sd.json` mounted to the Prometheus server by setting
```
configMaps:
  - <name of the configmap containing the custom_sd.json`
```

## Thanos (Long-term storage for metrics)

This chart supports [Thanos](https://github.com/improbable-eng/thanos) out-of-the-box.
See [Thanos docs](./templates/thanos/README.md) for additional details specific to the SAP Converged Cloud Enterprise Edition.

## Configuration

The following table provides an overview of configurable parameters of this chart and their defaults.  
See the [values.yaml](./values.yaml) for more details.  
**TLDR;** Set the `name`, `global.region`, `global.domain` parameters and get started where `name` has to be a unique identifier for your prometheus.

|       Parameter                        |           Description                                                                                                   |                         Default                     |
|----------------------------------------|-------------------------------------------------------------------------------------------------------------------------|-----------------------------------------------------|
| `global.region`                        | The OpenStack region.                                                                                                   | `""`                                                |
| `global.domain`                        | The URL domain.                                                                                                         | `""`                                                |
| `global.clusterType`                   | Optional: The type of the cluster to which the Prometheus is deployed.                                                  | `""`                                                |
| `global.cluster`                       | Optional: The name of the cluster to which the Prometheus is deployed.                                                  | `$global.region`                                    |
| `image.repository`                     | Repository of the Prometheus image.                                                                                     | `prom/prometheus`                                   |
| `image.tag`                            | Tag of the Prometheus image.                                                                                            | `v2.8.0`                                            |
| `name`                                 | Unique name for this Prometheus instance. The name will be used to assign aggregation and alerting rules to Prometheus. | `""`                                                |
| `retentionTime`                        | Defines how long data is stored. Format: `[0-9]+(ms|s|m|h|d|w|y)`.                                                      | `7d`                                                |
| `additionalScrapeConfigs.name`         | Name of the Secret containing the additional scrape configuration.                                                      | `""`                                                |
| `additionalScrapeConfigs.key`          | Key in the Secret containing the additional scrape configuration.                                                       | `""`                                                |
| `additionalScrapeConfigs.optional`     | Whether the Secret or the key must be found.                                                                            | `false`                                             |
| `scrapeInterval`                       | Global interval between consecutive scrapes.                                                                            | `"60s"`                                             |
| `ingress.enabled`                      | If enabled deploy an Ingress for this Prometheus.                                                                       | `false`                                             |
| `ingress.host`                         | Used to generate the external URL and ingress host in the form `<host>.<region>.<domain>`.                              | `""`                                                |
| `ingress.hostNameOverride`             | Used to set the complete hostname and skip above generation.                                                            | `""`                                                |
| `ingress.vice_president`               | Automate certificate management via vice-president (k8s operator).                                                      | `true`                                              |
| `ingress.disco`                        | Automate management of DNS provider via disco (k8s operator).                                                           | `true`                                              |
| `ingress.annotations`                  | Additional annotations for ingress.                                                                                     | `{}`                                                |
| `ingress.authentication`               | Ingress client certificate authentication. See values for details.                                                      | `{}`                                                |
| `persistence.enabled`                  | If enabled a persistent volume is used to store the data. Else data is stored in memory.                                | `false`                                             |
| `persistence.name`                     | Name of the persistent volume claim.                                                                                    | `$name`                                             |
| `persistence.accessMode`               | Access mode for the persistent volume.                                                                                  | `ReadWriteOnce`                                     |
| `persistence.size`                     | The size of the persistent volume claim with unit.                                                                      | `100Gi`                                             |
| `persistence.selector`                 | Label selector to be applied to the PVC.                                                                                | `{}`                                                |
| `logLevel`                             | The log level of the Prometheus server.                                                                                 | `info`                                              |
| `resources.requests.cpu`               | Kubernetes resource requests for CPU.                                                                                   | `4`                                                 |
| `resources.requests.memory`            | Kubernetes resource requests for memory.                                                                                | `8Gi`                                               |
| `rbac.create`                          | Create RBAC resources.                                                                                                  | `false`                                             |
| `serviceAccount.create`                | Create a service account for the Prometheus server.<br> Will not create a service account with name `default`. See values.yaml . | `false`                                             |                                               |
| `serviceAccount.name`                  | Name of the service account to use for the Prometheus server.                                                           | `""`                                                |
| `externalLabels`                       | The labels to add to any time series or alerts when communicating with any external system.                             | `{}`                                                |
| `configMaps`                           | List of configmaps in the same namespace as the Prometheus that should be mounted to `/etc/prometheus/configmaps`.      | `[]`                                                |
| `alertmanagers`                        | List of Alertmanagers to send alerts to.                                                                                | `[]`                                                |
| `serviceDiscoveries.endpoints.enabled` | En-/Disable service discovery of endpoints. See documentation for additional details.                                   | `true`                                              |
| `serviceDiscoveries.pods.enabled`      | En-/Disable service discovery of pods. See documentation for additional details.                                        | `false`                                             |    
| `tolerations`                          | The pods tolerations.                                                                                                   | `[]`                                                |
| `affinity`                             | The pod affinity.                                                                                                       | `{}`                                                |
| `nodeSelector`                         | The pod node selector.                                                                                                  | `{}`                                                |
| `secrets`                              | List of secrets in the same namespace as the Prometheus that should be mounted to `/etc/prometheus/secrets`.            | `[]`                                                |
| `thanos.enabled`                       | Enable Thanos support.                                                                                                  | `false`                                             |
| `thanos.seed.enabled`                  | Enable Openstack seed triggering creation of an OpenStack user and Swift container.                                     | `true`                                              |   
| `thanos.swiftStorageConfig`            | Thanos storage configuration for use with OpenStack Swift. See values for details.                                      | `{}`                                                |
| `thanos.spec`                          | The [Thanos spec](https://github.com/coreos/prometheus-operator/blob/master/Documentation/api.md#thanosspec)            | `{}`                                                |
| `alerts.tier`                          | The `tier` to which the Prometheus alerts are assigned to. <br>`$values.global.tier` takes precedence if set.           | `$values.global.tier` or `k8s`                      |
