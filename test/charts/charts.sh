#!/bin/ash

#set -o xtrace

# get minimum supported api version with `kubectl explain resource`
# in ash no arrays are supported, using grep on env var instead
k8s_min_version="1.7"
resources=`echo -e "ClusterRole:rbac.authorization.k8s.io/v1alpha1\n\
ClusterRole:rbac.authorization.k8s.io/v1beta1\n\
ClusterRoleBinding:rbac.authorization.k8s.io/v1beta1\n\
ClusterRoleBinding:rbac.authorization.k8s.io/v1alpha1\n\
Config:v1\n\
ConfigMap:v1\n\
DaemonSet:extensions/v1beta1\n\
Deployment:extensions/v1beta1\n\
Ingress:extensions/v1beta1\n\
PersistentVolumeClaim:v1\n\
Role:rbac.authorization.k8s.io/v1beta1\n\
RoleBinding:rbac.authorization.k8s.io/v1beta1\n\
Secret:v1\n\
Service:v1\n\
ServiceAccount:v1"`

helm init --client-only
helm repo add bugroger-charts https://raw.githubusercontent.com/BugRoger/charts/repo
helm repo add sapcc https://charts.global.cloud.sap

pwd=$(pwd)
for chart in $pwd/charts/*; do
  if [ -d "$chart" ]; then
    echo "Rendering chart in $chart ..."
    cd $chart
    # fix cross device move of overlay fs
    if [ -d "./charts" ]; then
      cp -a ./charts ./charts.bak
      rm -rf ./charts
      mv ./charts.bak ./charts
    fi
    helm dependency build --debug
    if [ -f test-values.yaml ]; then
      helm template --debug -f test-values.yaml . > /tmp/chart.yaml
    else
      helm template --debug . > /tmp/chart.yaml
    fi
    retval=$?
    rm -f ./charts/*.tgz
    if [ $retval -ne 0 ]; then
      echo "Rendering of template failed."
      exit $retval
    fi
    cd ..
    echo "Done."
  fi
done
