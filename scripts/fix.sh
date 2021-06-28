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

./wormhole-fixer.sh $CLUSTER $CLUSTER_CONFIG
./kube-proxy-fixer.sh $PARENT_CONTEXT $CLUSTER $CLUSTER $CLUSTER_CONFIG

echo Check status with:
echo env KUBECONFIG=$CLUSTER_CONFIG kubectl --context=$CLUSTER -n default get pods
KUBECONFIG=$CLUSTER_CONFIG kubectl rollout status -n default daemonset/replace-proxy-certs
KUBECONFIG=$CLUSTER_CONFIG kubectl rollout status -n default daemonset/restart-wormhole

KUBECONFIG=$CLUSTER_CONFIG kubectl delete -n default ds replace-proxy-certs restart-wormhole
KUBECONFIG=$CLUSTER_CONFIG kubectl delete -n default secret new-proxy-certs

rm $CLUSTER_CONFIG
