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
      app: wormhole
  updateStrategy:
    rollingUpdate:
      maxUnavailable: 1
    type: RollingUpdate
  template:
    metadata:
      labels:
        app: wormhole
    spec:
      priorityClassName: system-node-critical
      hostNetwork: true
      containers:
      - name: wormhole
        image: "{{ .Image }}"
        command:
        - sh
        - -c
        - |
          cp ${KUBE_CLIENT_CERT} /tmp/client-cert.livenessprobe
          exec wormhole client --listen={{ .ApiserverIP }}:{{ .ApiserverPort }} --kubeconfig=/var/lib/kubelet/kubeconfig
        volumeMounts:
        - mountPath: /var/lib/kubelet/
          name: kubernetes
          readOnly: true
        - mountPath: /etc/kubernetes/certs
          name: certs
          readOnly: true
        securityContext:
          privileged: true
        livenessProbe:
          exec:
            command:
            - sh
            - -c
            - |
              cmp -s ${KUBE_CLIENT_CERT} /tmp/client-cert.livenessprobe
          initialDelaySeconds: 60
          periodSeconds: 60
        env:
        - name: KUBE_CLIENT_CERT
          value: "/var/lib/kubelet/pki/kubelet-client-current.pem"
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
