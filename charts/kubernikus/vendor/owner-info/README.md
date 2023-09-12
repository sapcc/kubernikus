# owner-info

This chart deploys a ConfigMap that contains owner info about a chart.

This chart **should not** be deployed stand-alone, it is meant to be used as a dependency
by other charts.

**Caveat:** Only use `owner-info` for the top-level chart (i.e. the chart that you're
deploying). If you use it for any dependencies of your top-level chart then you will get
multiple ConfigMaps with name clash.

## Usage

Add `owner-info` as a dependency to your chart's `Chart.yaml` file:

```yaml
dependencies:
  - name: owner-info
    repository: https://charts.eu-de-2.cloud.sap
    version: # use owner-info's current version from Chart.yaml
```

then run:

```sh
$ helm dep update
```

## Configuration

The following table lists the configurable parameters of the `owner-info` chart and their default values.

| Parameter | Default | Description |
| ---       | ---         | ---     |
| `maintainers` | `[]` | List of people that maintain your chart. The list should be ordered by priority, i.e. primary maintainer should be at the top. |
| `helm-chart-url` | `WHERE-TO-FIND-THE-CHART-IN-GITHUB` | URL to your chart in github, e.g. `https://github.com/sapcc/helm-charts/tree/master/common/owner-info` |
