package servicing

const (
	DisableNodeServicesDaemonset = `
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: disable-node-services
  namespace: kube-system
spec:
  selector:
    matchLabels:
      app: disable-node-services
  template:
    metadata:
      labels:
        app: disable-node-services
    spec:
      tolerations:
      - operator: Exists
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
          set -x
          echo " _____________________________________________________________"
          echo "/                                                             \"
          echo "|  This daemonset disables kube-proxy/wormhole/flannel        |"
          echo "|  services on the nodes. Services were moved to run          |"
          echo "|  in the cluster since Kubernetes version 1.20.              |"
          echo "\_______________________________________________________  __'\"
          echo "                                                        |/   \\"
          echo "                                                         \    \\  ."
          echo "                                                              |\\/|"
          echo "                                                              / " '\"
          echo "                                                              . .   ."
          echo "                                                              /    ) |"
          echo "                                                             '  _.'  |"
          echo "                                                             '-'/    \"
          chroot /host systemctl disable kube-proxy || true
          chroot /host systemctl stop kube-proxy || true
          chroot /host systemctl disable wormhole.path || true
          chroot /host systemctl disable wormhole || true
          chroot /host systemctl stop wormhole.path || true
          chroot /host systemctl stop wormhole || true
          chroot /host systemctl disable flanneld || true
          chroot /host systemctl stop flanneld || true
        volumeMounts:
        - name: host
          mountPath: "/host"
      containers:
      - name: pause
        image: keppel.eu-de-1.cloud.sap/ccloud-dockerhub-mirror/sapcc/pause-amd64:3.1
      volumes:
      - name: host
        hostPath:
          path: "/"
`
)
