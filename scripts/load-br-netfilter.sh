#!/bin/bash
set -eo pipefail
if [ "$1" == "" ]; then
  echo "usage: $0 CONTEXT [KUBECONFIG]"
  exit 1
fi

CONTEXT=$1

if [ "$2" != "" ]; then
  echo using kubeconfig $2
  export KUBECONFIG=$2
fi

kubectl apply --context $CONTEXT -n default -f - <<EOF
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: load-br-netfilter-on-boot
spec:
  selector:
    matchLabels:
      app: load-br-netfilter-on-boot
  template:
    metadata:
      labels:
        app: load-br-netfilter-on-boot
    spec:
      hostPID: true
      initContainers:
        - name: init
          image: keppel.global.cloud.sap/ccloud-dockerhub-mirror/library/alpine:latest
          securityContext:
            privileged: true
          command:
            - sh
            - -c
          args:
            - |-
              set -xe
              chroot /host modprobe br_netfilter
              chroot /host sh -c 'echo br_netfilter > /etc/modules-load.d/br_netfilter.conf'
          volumeMounts:
            - name: host
              mountPath: "/host"
      containers:
        - name: pause
          image: keppel.global.cloud.sap/ccloud-dockerhub-mirror/sapcc/pause-amd64:3.1
      volumes:
        - name: host
          hostPath:
              path: "/"
      tolerations:
        - operator: Exists
EOF

kubectl rollout status --context $CONTEXT -n default daemonset/load-br-netfilter-on-boot
kubectl delete --context $CONTEXT -n default daemonset/load-br-netfilter-on-boot
