package wormhole

const (
	WormholeDaemonset = `
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: wormhole
  namespace: kube-system
spec:
  selector:
    matchLabels:
      name: wormhole
  updateStrategy:
    rollingUpdate:
      maxUnavailable: 1
    type: RollingUpdate
  template:
    metadata:
      labels:
        name: wormhole
    spec:
      priorityClassName: system-node-critical
      hostNetwork: true
      containers:
      - name: wormhole
        image: "{{ .Image }}"
        command:
        - "sh"
        - "-c"
        - "wormhole client --listen={{ .ApiserverIP }}:{{ .ApiserverPort }} --kubeconfig=/var/lib/kubelet/kubeconfig"
        volumeMounts:
        - mountPath: /var/lib/kubelet/
          name: kubernetes
          readOnly: true
        - mountPath: /etc/kubernetes/certs
          name: certs
          readOnly: true
        securityContext:
          privileged: true
      tolerations:
      - operator: Exists
      volumes:
      - name: kubernetes
        hostPath:
          path: /var/lib/kubelet/
      - name: certs
        hostPath:
          path: /etc/kubernetes/certs
`
)
