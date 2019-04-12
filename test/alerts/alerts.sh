#!/bin/bash

TMP_PROMETHEUS_CONFIG=/tmp/prometheus.yaml
TMP_VALUES=/tmp/values.yaml
TMP_CONFIGMAP=/tmp/config.yaml.bak

helm init --client-only
mkdir -p /var/run/secrets/kubernetes.io/serviceaccount/
touch /var/run/secrets/kubernetes.io/serviceaccount/token
cd charts/kubernikus-system/charts/prometheus
cat values.yaml ../../../../test/alerts/dummy-values.yaml > ${TMP_VALUES}
cp templates/config.yaml ${TMP_CONFIGMAP}
sed -i "s/kubernikus-system\/charts\///g" templates/config.yaml

echo "Checking prometheus rules ..."
promtool check rules *.rules *.alerts
if [ $? -ne 0 ]; then
    echo "Checking of prometheus rules failed."
    exit 1
fi

echo "Checking prometheus alerts ..."
helm template -f ${TMP_VALUES} . | yq r - data[prometheus.yaml] > ${TMP_PROMETHEUS_CONFIG}
if [ ! -s "${TMP_PROMETHEUS_CONFIG}" ]
then
    echo "Prometheus config is empty, exiting."
    exit 1
fi
promtool check config ${TMP_PROMETHEUS_CONFIG}
if [ $? -ne 0 ]; then
    echo "Checking of prometheus config failed."
    exit 1
fi

cp -f ${TMP_CONFIGMAP} templates/config.yaml
rm -f ${TMP_PROMETHEUS_CONFIG} ${TMP_VALUES}
