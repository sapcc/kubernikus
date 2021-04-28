#!/bin/bash

# default to kubernikus namespace
KUBENAMESPACE=${KUBENAMESPACE:-kubernikus}

case $KUBECONTEXT in
  k-master)
    BASE_URL=kubernikus-master.eu-nl-1.cloud.sap
    ;;
  v-*)
    BASE_URL=kubernikus-v.${KUBECONTEXT#"v-"}.cloud.sap
    ;;
  k-*)
    BASE_URL=kubernikus.${KUBECONTEXT#"k-"}.cloud.sap
    ;;
  admin)
    BASE_URL=$KUBENAMESPACE.admin.cloud.sap
    ;;
esac

kubectl get secret -n$KUBENAMESPACE $1-secret -ogo-template-file=<(cat<< EOF
{{ \$cluster := index .metadata.ownerReferences 0 "name" -}}
apiVersion: v1
kind: Config
clusters:
  - name: {{ \$cluster }}
    cluster:
       certificate-authority-data: {{ index .data "tls-ca.pem" }}
       server: https://{{ \$cluster }}.${BASE_URL}
contexts:
  - name: {{ \$cluster }}
    context:
      cluster: {{ \$cluster }}
      user: {{ \$cluster }}
current-context: {{ \$cluster }}
users:
  - name: {{ \$cluster }}
    user:
      client-certificate-data: {{ index .data "apiserver-clients-cluster-admin.pem" }}
      client-key-data: {{ index .data "apiserver-clients-cluster-admin-key.pem" }}
EOF
)
