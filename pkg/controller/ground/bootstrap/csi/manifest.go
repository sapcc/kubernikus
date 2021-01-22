package csi

const (
	CSIServiceAccountController = `
apiVersion: v1
kind: ServiceAccount
metadata:
  name: csi-cinder-controller-sa
  namespace: kube-system
`
	CSIServiceAccountNode = `
apiVersion: v1
kind: ServiceAccount
metadata:
  name: csi-cinder-node-sa
  namespace: kube-system
`
	CSIRole = `
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: external-resizer-cfg
  namespace: kube-system
rules:
- apiGroups:
  - coordination.k8s.io
  resources:
  - leases
  verbs:
  - get
  - watch
  - list
  - delete
  - update
  - create
`
	CSIClusterRoleAttacher = `
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: csi-attacher-role
rules:
- apiGroups:
  - ""
  resources:
  - persistentvolumes
  verbs:
  - get
  - list
  - watch
  - update
  - patch
- apiGroups:
  - ""
  resources:
  - nodes
  verbs:
  - get
  - list
  - watch
- apiGroups:
  - storage.k8s.io
  resources:
  - volumeattachments
  verbs:
  - get
  - list
  - watch
  - update
  - patch
- apiGroups:
  - storage.k8s.io
  resources:
  - csinodes
  verbs:
  - get
  - list
  - watch
`
	CSIClusterRoleNodePlugin = `
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: csi-nodeplugin-role
rules:
- apiGroups:
  - ""
  resources:
  - events
  verbs:
  - get
  - list
  - watch
  - create
  - update
  - patch
`
	CSIClusterRoleProvisioner = `
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: csi-provisioner-role
rules:
- apiGroups:
  - ""
  resources:
  - persistentvolumes
  verbs:
  - get
  - list
  - watch
  - create
  - delete
- apiGroups:
  - ""
  resources:
  - persistentvolumeclaims
  verbs:
  - get
  - list
  - watch
  - update
- apiGroups:
  - storage.k8s.io
  resources:
  - storageclasses
  verbs:
  - get
  - list
  - watch
- apiGroups:
  - ""
  resources:
  - nodes
  verbs:
  - get
  - list
  - watch
- apiGroups:
  - storage.k8s.io
  resources:
  - csinodes
  verbs:
  - get
  - list
  - watch
- apiGroups:
  - ""
  resources:
  - events
  verbs:
  - list
  - watch
  - create
  - update
  - patch
- apiGroups:
  - snapshot.storage.k8s.io
  resources:
  - volumesnapshots
  verbs:
  - get
  - list
- apiGroups:
  - snapshot.storage.k8s.io
  resources:
  - volumesnapshotcontents
  verbs:
  - get
  - list
`
	CSIClusterRoleResizer = `
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: csi-resizer-role
rules:
- apiGroups:
  - ""
  resources:
  - persistentvolumes
  verbs:
  - get
  - list
  - watch
  - update
  - patch
- apiGroups:
  - ""
  resources:
  - persistentvolumeclaims
  verbs:
  - get
  - list
  - watch
- apiGroups:
  - ""
  resources:
  - persistentvolumeclaims/status
  verbs:
  - update
  - patch
- apiGroups:
  - storage.k8s.io
  resources:
  - storageclasses
  verbs:
  - get
  - list
  - watch
- apiGroups:
  - ""
  resources:
  - events
  verbs:
  - list
  - watch
  - create
  - update
  - patch
`
	CSIClusterRoleSnapshotter = `
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: csi-snapshotter-role
rules:
- apiGroups:
  - ""
  resources:
  - events
  verbs:
  - list
  - watch
  - create
  - update
  - patch
- apiGroups:
  - snapshot.storage.k8s.io
  resources:
  - volumesnapshotclasses
  verbs:
  - get
  - list
  - watch
- apiGroups:
  - snapshot.storage.k8s.io
  resources:
  - volumesnapshotcontents
  verbs:
  - create
  - get
  - list
  - watch
  - update
  - delete
- apiGroups:
  - snapshot.storage.k8s.io
  resources:
  - volumesnapshotcontents/status
  verbs:
  - update
`
	CSIRoleBindingResizer = `
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: csi-resizer-role-cfg
  namespace: kube-system
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: external-resizer-cfg
subjects:
- kind: ServiceAccount
  name: csi-cinder-controller-sa
  namespace: kube-system
`
	CSIClusterRoleBindingAttacher = `
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: csi-attacher-binding
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: csi-attacher-role
subjects:
- kind: ServiceAccount
  name: csi-cinder-controller-sa
  namespace: kube-system
`
	CSIClusterRoleBindingNodePlugin = `
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: csi-nodeplugin-binding
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: csi-nodeplugin-role
subjects:
- kind: ServiceAccount
  name: csi-cinder-node-sa
  namespace: kube-system
`
	CSIClusterRoleBindingProvisioner = `
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: csi-provisioner-binding
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: csi-provisioner-role
subjects:
- kind: ServiceAccount
  name: csi-cinder-controller-sa
  namespace: kube-system
`
	CSIClusterRoleBindingResizer = `
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: csi-resizer-binding
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: csi-resizer-role
subjects:
- kind: ServiceAccount
  name: csi-cinder-controller-sa
  namespace: kube-system
`
	CSIClusterRoleBindingSnapshotter = `
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: csi-snapshotter-binding
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: csi-snapshotter-role
subjects:
- kind: ServiceAccount
  name: csi-cinder-controller-sa
  namespace: kube-system
`
	CSIControllerService = `
kind: Service
apiVersion: v1
metadata:
  name: csi-cinder-controller-service
  namespace: kube-system
  labels:
    app: csi-cinder-controllerplugin
spec:
  selector:
    app: csi-cinder-controllerplugin
  ports:
    - name: dummy
      port: 12345
`
	CSIStatefulSet = `
kind: StatefulSet
apiVersion: apps/v1
metadata:
  name: csi-cinder-controllerplugin
  namespace: kube-system
spec:
  serviceName: "csi-cinder-controller-service"
  replicas: 1
  selector:
    matchLabels:
      app: csi-cinder-controllerplugin
  template:
    metadata:
      labels:
        app: csi-cinder-controllerplugin
    spec:
      serviceAccount: csi-cinder-controller-sa
      containers:
        - name: csi-attacher
          image: k8s.gcr.io/sig-storage/csi-attacher:v2.2.1
          args:
            - "--csi-address=$(ADDRESS)"
            - "--timeout=3m"
          env:
            - name: ADDRESS
              value: /var/lib/csi/sockets/pluginproxy/csi.sock
          imagePullPolicy: "IfNotPresent"
          volumeMounts:
            - name: socket-dir
              mountPath: /var/lib/csi/sockets/pluginproxy/
        - name: csi-provisioner
          image: k8s.gcr.io/sig-storage/csi-provisioner:v1.6.1
          args:
            - "--csi-address=$(ADDRESS)"
            - "--timeout=3m"
          env:
            - name: ADDRESS
              value: /var/lib/csi/sockets/pluginproxy/csi.sock
          imagePullPolicy: "IfNotPresent"
          volumeMounts:
            - name: socket-dir
              mountPath: /var/lib/csi/sockets/pluginproxy/
        - name: csi-snapshotter
          image: k8s.gcr.io/sig-storage/csi-snapshotter:v2.1.3
          args:
            - "--csi-address=$(ADDRESS)"
            - "--timeout=3m"
          env:
            - name: ADDRESS
              value: /var/lib/csi/sockets/pluginproxy/csi.sock
          imagePullPolicy: Always
          volumeMounts:
            - mountPath: /var/lib/csi/sockets/pluginproxy/
              name: socket-dir
        - name: csi-resizer
          image: k8s.gcr.io/sig-storage/csi-resizer:v0.5.1
          args:
            - "--csi-address=$(ADDRESS)"
            - "--csiTimeout=3m"
          env:
            - name: ADDRESS
              value: /var/lib/csi/sockets/pluginproxy/csi.sock
          imagePullPolicy: "IfNotPresent"
          volumeMounts:
            - name: socket-dir
              mountPath: /var/lib/csi/sockets/pluginproxy/
        - name: liveness-probe
          image: k8s.gcr.io/sig-storage/livenessprobe:v2.1.0
          args:
            - "--csi-address=$(ADDRESS)"
          env:
            - name: ADDRESS
              value: /var/lib/csi/sockets/pluginproxy/csi.sock
          volumeMounts:
            - mountPath: /var/lib/csi/sockets/pluginproxy/
              name: socket-dir
        - name: cinder-csi-plugin
          image: docker.io/k8scloudprovider/cinder-csi-plugin:latest
          args:
            - /bin/cinder-csi-plugin
            - "--nodeid=$(NODE_ID)"
            - "--endpoint=$(CSI_ENDPOINT)"
            - "--cloud-config=$(CLOUD_CONFIG)"
            - "--cluster=$(CLUSTER_NAME)"
          env:
            - name: NODE_ID
              valueFrom:
                fieldRef:
                  fieldPath: spec.nodeName
            - name: CSI_ENDPOINT
              value: unix://csi/csi.sock
            - name: CLOUD_CONFIG
              value: /etc/config/cloudprovider.conf
            - name: CLUSTER_NAME
              value: kubernetes
          imagePullPolicy: "IfNotPresent"
          ports:
            - containerPort: 9808
              name: healthz
              protocol: TCP
          # The probe
          livenessProbe:
            failureThreshold: 5
            httpGet:
              path: /healthz
              port: healthz
            initialDelaySeconds: 10
            timeoutSeconds: 3
            periodSeconds: 10
          volumeMounts:
            - name: socket-dir
              mountPath: /csi
            - name: secret-cinderplugin
              mountPath: /etc/config
              readOnly: true
      volumes:
        - name: socket-dir
          emptyDir: {}
        - name: secret-cinderplugin
          secret:
            secretName: cloud-config
`
	CSIDaemonsSet = `
kind: DaemonSet
apiVersion: apps/v1
metadata:
  name: csi-cinder-nodeplugin
  namespace: kube-system
spec:
  selector:
    matchLabels:
      app: csi-cinder-nodeplugin
  template:
    metadata:
      labels:
        app: csi-cinder-nodeplugin
    spec:
      tolerations:
        - operator: Exists
      serviceAccount: csi-cinder-node-sa
      hostNetwork: true
      containers:
        - name: node-driver-registrar
          image: k8s.gcr.io/sig-storage/csi-node-driver-registrar:v1.3.0
          args:
            - "--csi-address=$(ADDRESS)"
            - "--kubelet-registration-path=$(DRIVER_REG_SOCK_PATH)"
          lifecycle:
            preStop:
              exec:
                command: ["/bin/sh", "-c", "rm -rf /registration/cinder.csi.openstack.org /registration/cinder.csi.openstack.org-reg.sock"]
          env:
            - name: ADDRESS
              value: /csi/csi.sock
            - name: DRIVER_REG_SOCK_PATH
              value: /var/lib/kubelet/plugins/cinder.csi.openstack.org/csi.sock
            - name: KUBE_NODE_NAME
              valueFrom:
                fieldRef:
                  fieldPath: spec.nodeName
          imagePullPolicy: "IfNotPresent"
          volumeMounts:
            - name: socket-dir
              mountPath: /csi
            - name: registration-dir
              mountPath: /registration
        - name: liveness-probe
          image: k8s.gcr.io/sig-storage/livenessprobe:v2.1.0
          args:
            - --csi-address=/csi/csi.sock
          volumeMounts:
            - name: socket-dir
              mountPath: /csi
        - name: cinder-csi-plugin
          securityContext:
            privileged: true
            capabilities:
              add: ["SYS_ADMIN"]
            allowPrivilegeEscalation: true
          image: docker.io/k8scloudprovider/cinder-csi-plugin:latest
          args:
            - /bin/cinder-csi-plugin
            - "--nodeid=$(NODE_ID)"
            - "--endpoint=$(CSI_ENDPOINT)"
            - "--cloud-config=$(CLOUD_CONFIG)"
          env:
            - name: NODE_ID
              valueFrom:
                fieldRef:
                  fieldPath: spec.nodeName
            - name: CSI_ENDPOINT
              value: unix://csi/csi.sock
            - name: CLOUD_CONFIG
              value: /etc/config/cloudprovider.conf
          imagePullPolicy: "IfNotPresent"
          ports:
            - containerPort: 9808
              name: healthz
              protocol: TCP
          # The probe
          livenessProbe:
            failureThreshold: 5
            httpGet:
              path: /healthz
              port: healthz
            initialDelaySeconds: 10
            timeoutSeconds: 3
            periodSeconds: 10
          volumeMounts:
            - name: socket-dir
              mountPath: /csi
            - name: kubelet-dir
              mountPath: /var/lib/kubelet
              mountPropagation: "Bidirectional"
            - name: pods-probe-dir
              mountPath: /dev
              mountPropagation: "HostToContainer"
            - name: secret-cinderplugin
              mountPath: /etc/config
              readOnly: true
      volumes:
        - name: socket-dir
          hostPath:
            path: /var/lib/kubelet/plugins/cinder.csi.openstack.org
            type: DirectoryOrCreate
        - name: registration-dir
          hostPath:
            path: /var/lib/kubelet/plugins_registry/
            type: Directory
        - name: kubelet-dir
          hostPath:
            path: /var/lib/kubelet
            type: Directory
        - name: pods-probe-dir
          hostPath:
            path: /dev
            type: Directory
        - name: secret-cinderplugin
          secret:
            secretName: cloud-config
`
	CSIDriver = `
apiVersion: storage.k8s.io/v1
kind: CSIDriver
metadata:
  name: cinder.csi.openstack.org
spec:
  attachRequired: true
  podInfoOnMount: true
  volumeLifecycleModes:
  - Persistent
  - Ephemeral  
`
)
