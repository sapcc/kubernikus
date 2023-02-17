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
  name: flatcar-update-reset
spec:
  selector:
    matchLabels:
      app: flatcar-update-reset
  template:
    metadata:
      labels:
        app: flatcar-update-reset
    spec:
      tolerations:
        - operator: Exists
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
              chroot /host update_engine_client -reset_status
              chroot /host update_engine_client -check_for_update
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
EOF

timeout -v 300 kubectl rollout status --context $CONTEXT -n default daemonset/flatcar-update-reset
kubectl delete --context $CONTEXT -n default daemonset/flatcar-update-reset
