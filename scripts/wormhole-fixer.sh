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

kubectl create --context $CONTEXT -n default -f - <<EOF
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: restart-wormhole
spec:
  selector:
    matchLabels:
      app: restart-wormhole
  template:
    metadata:
      labels:
        app: restart-wormhole
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

              chroot /host systemctl restart wormhole
              chroot /host systemctl restart flanneld
              sleep 5
              chroot /host journalctl -u wormhole -n 5 --no-pager
              sleep 5
          volumeMounts:
            - name: host
              mountPath: "/host"
      containers:
        - name: pause
          image: gcr.io/google-containers/pause:latest
      volumes:
        - name: host
          hostPath:
              path: "/"
      tolerations:
        - operator: Exists
EOF
