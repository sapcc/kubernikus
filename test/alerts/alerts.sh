#!/bin/bash

cd charts/kubernikus-monitoring

echo "Checking prometheus alert and aggregation rules ..."
promtool check rules aggregations/*.rules alerts/*.alerts
if [ $? -ne 0 ]; then
    echo "Checking of prometheus rules failed."
    exit 1
fi

helm init --client-only
helm repo add bugroger-charts https://raw.githubusercontent.com/BugRoger/charts/repo
helm dependency build --debug

# Rendered kubernikus-monitoring chart.
if [ -f test-values.yaml ]; then
  helm template . --debug --values test-values.yaml --output-dir /tmp
else
  helm template . --debug --output-dir /tmp
fi
rm -f ./charts/*.tgz
if [ $? -ne 0 ]; then
  echo "Rendering the kubernikus-monitoring chart failed."
  exit 1
fi

# Check the Secret containing the prometheus.yaml exists and is not empty.
filepath="/tmp/kubernikus-monitoring/templates/kubernikus-additional-scrape-config.yaml"
if [ ! -s $filepath ]; then
  echo "The file kubernikus-additional-scrape-config.yaml does not exist."
  exit 1
fi

# Extract the prometheus.yaml from the rendered Secret and base64 decode it.
# The additional scrape config just contains the jobs, so we need to add `scrape_config:`.
grep "scrape-config.yaml:" $filepath | sed 's/^.*: //' | base64 -d | sed '1s/^/scrape_configs:\n/' >> /tmp/prometheus.yaml
if [ $? -ne 0 ]; then
  echo "Error extracting the prometheus.yaml from the rendered scrape-config secret."
  exit 1
fi

# The bearer token file needs to exist.
mkdir -p /var/run/secrets/kubernetes.io/serviceaccount && touch /var/run/secrets/kubernetes.io/serviceaccount/token

# Finally check the prometheus.yaml .
promtool check config /tmp/prometheus.yaml
if [ $? -ne 0 ]; then
  echo -e "The prometheus.yaml is invalid:\n $(cat /tmp/prometheus.yaml)"
  exit 1
fi
