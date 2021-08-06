#!/bin/bash
set -eo pipefail
if [ "$3" == "" ]; then
  echo "usage: $0 PARENT_CONTEXT CONTEXT CLUSTERFQDN [KUBECONFIG]"
  exit 1
fi

PARENT_CONTEXT=$1
CONTEXT=$2
CLUSTER=$3



kubectl get secret --context $PARENT_CONTEXT $CLUSTER-secret --namespace=kubernikus -o go-template='{"kind":"Secret", "apiVersion":"v1", "metadata":{"name":"new-proxy-certs"}, "data":{"apiserver-clients-system-kube-proxy-key.pem": "{{index .data "apiserver-clients-system-kube-proxy-key.pem"}}", "apiserver-clients-system-kube-proxy.pem":"{{index .data "apiserver-clients-system-kube-proxy.pem"}}"}}' | kubectl --kubeconfig=$4 apply -n default --context=$CONTEXT -f -

if [ "$4" != "" ]; then
  echo using kubeconfig $4
  export KUBECONFIG=$4
fi

kubectl apply --context $CONTEXT -n default -f - <<EOF
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: replace-proxy-certs
spec:
  selector:
    matchLabels:
      app: replace-proxy-certs
  template:
    metadata:
      labels:
        app: replace-proxy-certs
    spec:
      hostPID: true
      initContainers:
        - name: init
          image: keppel.eu-de-1.cloud.sap/ccloud-dockerhub-mirror/library/alpine:latest
          securityContext:
            privileged: true
          command:
            - sh
            - -c
          args:
            - |-
              set -xe
              sleep $((RANDOM % 15))s
              cp /certs/apiserver-clients-system-kube-proxy* /host/etc/kubernetes/certs/

              chroot /host systemctl restart kube-proxy
              sleep 5
              chroot /host journalctl -u kube-proxy -n 5 --no-pager
              sleep 5
          volumeMounts:
            - name: host
              mountPath: "/host"
            - name: certs
              mountPath: "/certs"
      containers:
        - name: pause
          image: gcr.io/google-containers/pause:latest
      volumes:
        - name: host
          hostPath:
              path: "/"
        - name: certs
          secret:
            secretName: new-proxy-certs
      tolerations:
        - operator: Exists
EOF

uid=$(kubectl --context=$CONTEXT -n default get daemonset replace-proxy-certs -ojsonpath='{.metadata.uid}')
#kubectl --context=$CONTEXT -n default patch secret new-proxy-certs --type='json' -p='[{"op": "replace", "path":"/metadata/ownerReferences", "value":[{"apiVersion":"apps/v1", "kind":"DaemonSet", "name":"replace-proxy-certs", "uid":"'$uid'"}]}]'
