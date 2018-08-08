package gpu

const (
	NVIDIADevicePlugin_v20180808 = `
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: nvidia-gpu-device-plugin
  namespace: kube-system
  labels:
    k8s-app: nvidia-gpu-device-plugin
    addonmanager.kubernetes.io/mode: Reconcile
spec:
  selector:
    matchLabels:
      k8s-app: nvidia-gpu-device-plugin
  template:
    metadata:
      labels:
        k8s-app: nvidia-gpu-device-plugin
      annotations:
        scheduler.alpha.kubernetes.io/critical-pod: ''
    spec:
      priorityClassName: system-node-critical
      affinity:
        nodeAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            nodeSelectorTerms:
            - matchExpressions:
              - key: ccloud.sap.com/nvidia-gpu
                operator: Exists
      tolerations:
      - operator: "Exists"
        effect: "NoExecute"
      - operator: "Exists"
        effect: "NoSchedule"
      volumes:
      - name: device-plugin
        hostPath:
          path: /var/lib/kubelet/device-plugins
      - name: dev
        hostPath:
          path: /dev
      containers:
      - image: "k8s.gcr.io/nvidia-gpu-device-plugin@sha256:0842734032018be107fa2490c98156992911e3e1f2a21e059ff0105b07dd8e9e"
        command:
          - /usr/bin/nvidia-gpu-device-plugin
          - -logtostderr
          - -host-path=/opt/nvidia/current/lib64
          - -container-path=/usr/local/nvidia/lib64
        name: nvidia-gpu-device-plugin
        resources:
          requests:
            cpu: 50m
            memory: 10Mi
          limits:
            cpu: 50m
            memory: 10Mi
        securityContext:
          privileged: true
        volumeMounts:
        - name: device-plugin
          mountPath: /device-plugin
        - name: dev
          mountPath: /dev
  updateStrategy:
    type: RollingUpdate
`

	NVIDIADriverInstaller_v20180808 = `
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: nvidia-driver-installer
  namespace: kube-system
  labels:
    k8s-app: nvidia-driver-installer
spec:
  selector:
    matchLabels:
      k8s-app: nvidia-driver-installer
  updateStrategy:
    type: RollingUpdate
  template:
    metadata:
      labels:
        name: nvidia-driver-installer
        k8s-app: nvidia-driver-installer
    spec:
      affinity:
        nodeAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            nodeSelectorTerms:
            - matchExpressions:
              - key: ccloud.sap.com/nvidia-gpu
                operator: Exists
      tolerations:
      - key: "nvidia.com/gpu"
        effect: "NoSchedule"
        operator: "Exists"
      hostNetwork: true
      hostPID: true
      volumes:
      - name: rootfs
        hostPath:
          path: /
      initContainers:
      - image: bugroger/coreos-nvidia-driver:stable-396.44-tesla
        name: nvidia-driver-installer
        imagePullPolicy: Always
        resources:
          requests:
            cpu: 0.15
        securityContext:
          privileged: true
        volumeMounts:
        - name: rootfs 
          mountPath: /root
          mountPropagation: Bidirectional
      containers:
      - image: "gcr.io/google-containers/pause:2.0"
        name: pause
`
)
