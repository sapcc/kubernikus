#!/bin/bash

cd charts/kubernikus-monitoring

echo "Checking prometheus alert and aggregation rules ..."
promtool check rules aggregations/*.rules alerts/*.alerts
if [ $? -ne 0 ]; then
    echo "Checking of prometheus rules failed."
    exit 1
fi

helm init --client-only
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
