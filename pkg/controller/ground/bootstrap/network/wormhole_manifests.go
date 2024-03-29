package network

const (
	WormholeDaemonSet = `
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: wormhole
  namespace: kube-system
  labels:
    tier: node
    app: wormhole
spec:
  updateStrategy:
    rollingUpdate:
      maxUnavailable: 1
    type: RollingUpdate
  selector:
    matchLabels:
      app: wormhole
  template:
    metadata:
      labels:
        tier: node
        app: wormhole
    spec:
      affinity:
        nodeAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            nodeSelectorTerms:
            - matchExpressions:
              - key: kubernetes.io/os
                operator: In
                values:
                - linux
              - key: kubernikus.cloud.sap/cni
                operator: In
                values:
                - "true"
      hostNetwork: true
      priorityClassName: system-node-critical
      tolerations:
      - operator: Exists
        effect: NoSchedule
      serviceAccountName: flannel
      containers:
      - name: wormhole
        image: "{{.Wormhole}}"
        command:
        - sh
        - -ec
        args:
        - |
          cp ${KUBE_CLIENT_CERT} /tmp/client-cert.livenessprobe
          exec wormhole client --listen {{ .Listen }} --health-check=false --kubeconfig=/var/lib/kubelet/kubeconfig
        env:
          - name: KUBE_CLIENT_CERT
            value: /var/lib/kubelet/pki/kubelet-client-current.pem
        resources:
          requests:
            cpu: "100m"
            memory: "50Mi"
        livenessProbe:
          exec:
            command: [cmp, -s, $(KUBE_CLIENT_CERT), /tmp/client-cert.livenessprobe]
        volumeMounts:
        - name: config
          mountPath: /var/lib/kubelet
          readOnly: true
      volumes:
      - name: config
        hostPath:
          path: /var/lib/kubelet
`
)
