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

    echo "Checking API versions ..."
    while IFS= read -r line <&3; do
      if echo "$line" | grep "^---$" > /dev/null; then
        unset api_real kind_real
        continue
      fi
      api_tmp=`echo $line | grep "^apiVersion: .*$" | awk -F': ' '{print $2}' | sed 's/\"//g'`
      kind_tmp=`echo $line | grep "^kind: .*$" | awk -F': ' '{print $2}' | sed 's/\"//g'`
      if [[ ! -z "$api_tmp" ]]; then
        api_real=$api_tmp
      fi
      if [[ ! -z "$kind_tmp" ]]; then
        kind_real=$kind_tmp
      fi
      if [[ ! -z "$api_real" && ! -z "$kind_real" ]]; then
        if ! echo "$resources" | grep "^$kind_real:$api_real$" > /dev/null; then
          echo "kind: $kind_real apiVersion: $api_real not matching minimum version requirements ($k8s_min_version)!"
          exit 1
        fi
        unset api_real kind_real
      fi      
    done 3< "/tmp/chart.yaml"
    echo "Done."
  fi
done
