#!/bin/bash
set -eo pipefail

if [ "$1" == "" ]; then
  echo usage $0 CLUSTER_FQDN
  exit
fi

temp_file=$(mktemp)

cat << EOF > $temp_file
api:
  resources:
    limits:
      cpu: 1        #default: 1
      memory: 2Gi   #default: 2Gi
    requests:
      cpu: 200m     #default: 200m
      memory: 512Mi #default: 512Mi
etcd:
  resources:
    limits:
      cpu: 1         #default: 1
      memory: 2560Mi #default 2560Mi
    requests:
      cpu: 200m      #default: 200m
      memory: 500Mi  #default: 500Mi

#controllerManager:
#  resources:
#    requests:
#      cpu: 100m
#      memory: 256Mi
#    limits:
#      cpu: 500m
#      memory: 512Mi
EOF

extra_values=$(kubectl get secret $1-secret -ojsonpath='{.data.extra-values}' | base64 -D)

echo "$extra_values" | yq -i ea '. as $item ireduce ({}; . * $item )' $temp_file -

before=$(shasum -a 256 $temp_file)
# open editor
${EDITOR:-vi} $temp_file
if echo "$before"|shasum -a 256 -s -c -; then
  echo nothing changed, canceling
  exit 0
fi

#strip comments
yq -i '... comments=""' $temp_file

kubectl patch secret $1-secret --type='json' -p='[{"op": "replace", "path":"/data/extra-values", "value":"'$(cat $temp_file | base64)'"}]'
rm $temp_file

