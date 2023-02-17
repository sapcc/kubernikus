#!/bin/bash
set -eo pipefail

if [ "$2" == "" ]; then
  echo usage $0 PARENT_CONTEXT CLUSTER_FQDN
  exit
fi
unset KUBECONTEXT
unset KUBENAMESPACE

PARENT_CONTEXT=$1
CLUSTER=$2
CLUSTER_NAME=${2%-*}
CLUSTER_CONFIG=$TMPDIR$CLUSTER

KUBECONTEXT=$PARENT_CONTEXT ./kubeconfig.sh $2 >$CLUSTER_CONFIG
echo Check status with:
echo env KUBECONFIG=$CLUSTER_CONFIG kubectl --context=$CLUSTER -n default get pods
./load-br-netfilter.sh $CLUSTER $CLUSTER_CONFIG


rm $CLUSTER_CONFIG
