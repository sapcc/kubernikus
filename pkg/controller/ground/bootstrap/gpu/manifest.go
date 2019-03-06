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
              - key: gpu
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
      - image: "k8s.gcr.io/nvidia-gpu-device-plugin@sha256:d18b678437fedc4ec4211c20b3e5469a137a44f989da43dc275e4f2678170db4"
        command:
          - /usr/bin/nvidia-gpu-device-plugin
          - -logtostderr
        name: nvidia-gpu-device-plugin
        resources:
          requests:
            cpu: 50m
            memory: 50Mi
          limits:
            cpu: 50m
            memory: 50Mi
	terminationGracePeriodSeconds: 0
        securityContext:
          privileged: true
        volumeMounts:
        - name: device-plugin
          mountPath: /device-plugin
        - name: dev
          mountPath: /dev
  updateStrategy:
    type: OnDelete
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
              - key: gpu
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
      - name: dev 
        hostPath:
          path: /dev
      initContainers:
      - image: bugroger/coreos-nvidia-driver:stable-396.44-tesla
        name: nvidia-driver-installer
        imagePullPolicy: Always
	terminationGracePeriodSeconds: 0
        securityContext:
          privileged: true
        volumeMounts:
        - name: rootfs 
          mountPath: /root
          mountPropagation: Bidirectional
	- name: dev
	  mountPath: /dev
	  mountPropagation: Bidirectional
      containers:
      - image: "gcr.io/google-containers/pause:2.0"
        name: pause
`
)
