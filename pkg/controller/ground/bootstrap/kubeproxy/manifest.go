package kubeproxy

const (
	KubeProxyDaemonset = `
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: kube-proxy
  namespace: kube-system
spec:
  selector:
    matchLabels:
      app: kube-proxy
  updateStrategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 1
  template:
    metadata:
      labels:
        app: kube-proxy
    spec:
      priorityClassName: system-node-critical
      hostNetwork: true
      nodeSelector:
        kubernetes.io/os: linux
      tolerations:
      - operator: "Exists"
        effect: "NoExecute"
      - operator: "Exists"
        effect: "NoSchedule"
      containers:
      - name: kube-proxy
        image: {{ .Image }}
        resources:
          requests:
            cpu: 500m
        command:
        - /bin/sh
        - -c
        - kube-proxy --hostname-override=${NODE_NAME} --config=/etc/kubernetes/kube-proxy/config
        env:
        - name: NODE_NAME
          valueFrom:
            fieldRef:
              fieldPath: spec.nodeName
        - name: KUBERNETES_SERVICE_HOST
          value: {{ .ApiserverHost }}
        securityContext:
          privileged: true
        volumeMounts:
        - mountPath: /run/xtables.lock
          name: xtables-lock
          readOnly: false
        - mountPath: /lib/modules
          name: lib-modules
          readOnly: true
        - name: config
          mountPath: /etc/kubernetes/kube-proxy
          readOnly: true
        - name: ca
          mountPath: /etc/ssl/certs/ca.crt
          subPath: ca.crt
          readOnly: true
      volumes:
      - name: xtables-lock
        hostPath:
          path: /run/xtables.lock
          type: FileOrCreate
      - name: lib-modules
        hostPath:
          path: /lib/modules
      - name: config
        configMap:
          name: kube-proxy
      - name: ca
        configMap:
          name: kube-root-ca.crt
      serviceAccountName: kube-proxy
`
	KubeProxyServiceAccount = `
apiVersion: v1
kind: ServiceAccount
metadata:
  name: kube-proxy
  namespace: kube-system
  labels:
    addonmanager.kubernetes.io/mode: Reconcile
`
	KubeProxyClusterRoleBinding = `
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: system:kube-proxy
  labels:
    addonmanager.kubernetes.io/mode: Reconcile
subjects:
  - kind: ServiceAccount
    name: kube-proxy
    namespace: kube-system
roleRef:
  kind: ClusterRole
  name: system:node-proxier
  apiGroup: rbac.authorization.k8s.io
`
	KubeProxyConfigmap = `
kind: ConfigMap
apiVersion: v1
metadata:
  name: kube-proxy
  namespace: kube-system
  labels:
    tier: node
    app: kube-proxy
data:
  config: |
    apiVersion: kubeproxy.config.k8s.io/v1alpha1
    kind: KubeProxyConfiguration
    bindAddress: 0.0.0.0
    clientConnection:
      acceptContentTypes: ""
      burst: 10
      contentType: application/vnd.kubernetes.protobuf
      qps: 5
    clusterCIDR: "{{ .ClusterCIDR }}"
    configSyncPeriod: 15m0s
    conntrack:
      maxPerCore: 32768
      min: 131072
      tcpCloseWaitTimeout: 1h0m0s
      tcpEstablishedTimeout: 24h0m0s
    enableProfiling: false
    featureGates: {}
    healthzBindAddress: 0.0.0.0:10256
    iptables:
      masqueradeAll: false
      masqueradeBit: 14
      minSyncPeriod: 0s
      syncPeriod: 30s
    metricsBindAddress: 127.0.0.1:10249
    mode: "iptables"
    oomScoreAdj: -999
    portRange: ""
    udpIdleTimeout: 250ms
`
)
