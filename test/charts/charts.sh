#!/bin/ash

helm init --client-only --stable-repo-url https://charts.helm.sh/stable

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
