#!/bin/bash

cd charts/kubernikus-monitoring

echo "Checking prometheus alert and aggregation rules ..."
promtool check rules aggregations/*.rules alerts/*.alerts
if [ $? -ne 0 ]; then
    echo "Checking of prometheus rules failed."
    exit 1
fi
