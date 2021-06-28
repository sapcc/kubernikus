package dns

import (
	"github.com/pkg/errors"
	apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	extensions "k8s.io/api/extensions/v1beta1"
	rbac "k8s.io/api/rbac/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	clientset "k8s.io/client-go/kubernetes"
	clientsetscheme "k8s.io/client-go/kubernetes/scheme"

	"github.com/sapcc/kubernikus/pkg/api/spec"
	"github.com/sapcc/kubernikus/pkg/controller/ground/bootstrap"
)

// SeedCoreDNS creates a deployment and related resources for running CoreDNS in the customer cluster
func SeedCoreDNS(client clientset.Interface, image, domain, clusterIP string) error {
	if err := createCoreDNSServiceAccount(client); err != nil {
		return errors.Wrap(err, "Failed to ensure coredns service account")
	}

	if err := createCoreDNSClusterRole(client); err != nil {
		return errors.Wrap(err, "Failed to ensure coredns cluster role")
	}

	if err := createCoreDNSClusterRoleBinding(client); err != nil {
		return errors.Wrap(err, "Failed to ensure coredns cluster role binding")
	}

	if err := createCoreDNSConfigMap(client, domain); err != nil {
		return errors.Wrap(err, "Failed to ensure coredns config map")
	}

	if err := createCoreDNSService(client, clusterIP); err != nil {
		return errors.Wrap(err, "Failed to ensure coredns service")
	}
	if err := createCoreDNSDeployment(client, image); err != nil {
		return errors.Wrap(err, "Failed to ensure coredns deployment")
	}

	return nil
}

func SeedCoreDNS116(client clientset.Interface, image, domain, clusterIP string) error {
	if err := createCoreDNSServiceAccount(client); err != nil {
		return errors.Wrap(err, "Failed to ensure coredns service account")
	}

	if err := createCoreDNSClusterRole(client); err != nil {
		return errors.Wrap(err, "Failed to ensure coredns cluster role")
	}

	if err := createCoreDNSClusterRoleBinding(client); err != nil {
		return errors.Wrap(err, "Failed to ensure coredns cluster role binding")
	}

	if err := createCoreDNSConfigMap(client, domain); err != nil {
		return errors.Wrap(err, "Failed to ensure coredns config map")
	}

	if err := createCoreDNSService(client, clusterIP); err != nil {
		return errors.Wrap(err, "Failed to ensure coredns service")
	}
	if err := createCoreDNSDeployment116(client, image); err != nil {
		return errors.Wrap(err, "Failed to ensure coredns deployment")
	}

	return nil
}

func createCoreDNSServiceAccount(client clientset.Interface) error {
	return bootstrap.CreateOrUpdateServiceAccount(client, &v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "coredns",
			Namespace: metav1.NamespaceSystem,
			Labels: map[string]string{
				"kubernetes.io/cluster-service":   "true",
				"addonmanager.kubernetes.io/mode": "Reconcile",
			},
		},
	})
}

func createCoreDNSClusterRole(client clientset.Interface) error {
	return bootstrap.CreateOrUpdateClusterRole(client, &rbac.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: "system:coredns",
			Labels: map[string]string{
				"kubernetes.io/bootstrapping":     "rbac-defaults",
				"addonmanager.kubernetes.io/mode": "Reconcile",
			},
		},
		Rules: []rbac.PolicyRule{
			{
				Verbs:     []string{"list", "watch"},
				APIGroups: []string{""},
				Resources: []string{"endpoints", "services", "pods", "namespaces"},
			},
			{
				Verbs:     []string{"get"},
				APIGroups: []string{""},
				Resources: []string{"nodes"},
			},
		},
	})
}

func createCoreDNSClusterRoleBinding(client clientset.Interface) error {
	return bootstrap.CreateOrUpdateClusterRoleBinding(client, &rbac.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "system:coredns",
			Labels: map[string]string{
				"kubernetes.io/bootstrapping":     "rbac-defaults",
				"addonmanager.kubernetes.io/mode": "EnsureExists",
			},
			Annotations: map[string]string{
				"rbac.authorization.kubernetes.io/autoupdate": "true",
			},
		},
		RoleRef: rbac.RoleRef{
			APIGroup: rbac.GroupName,
			Kind:     "ClusterRole",
			Name:     "system:coredns",
		},
		Subjects: []rbac.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "coredns",
				Namespace: metav1.NamespaceSystem,
			},
		},
	})
}

func createCoreDNSConfigMap(client clientset.Interface, domain string) error {
	if domain == "" {
		domain = spec.MustDefaultString("KlusterSpec", "dnsDomain")
	}

	manifest := `
.:53 {
    errors
    health
    kubernetes {{ .Domain }} in-addr.arpa ip6.arpa {
        pods insecure
        fallthrough in-addr.arpa ip6.arpa
        ttl 30
    }
    prometheus :9153
    forward . /etc/resolv.conf
    cache 30
    loop
    reload
    loadbalance
}`

	data := struct{ Domain string }{domain}

	template, err := bootstrap.RenderManifest(manifest, data)
	if err != nil {
		return err
	}

	return bootstrap.CreateOrUpdateConfigMap(client, &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "coredns",
			Namespace: metav1.NamespaceSystem,
			Labels: map[string]string{
				"addonmanager.kubernetes.io/mode": "EnsureExists",
			},
		},
		Data: map[string]string{
			"Corefile": string(template),
		},
	})
}

func createCoreDNSService(client clientset.Interface, clusterIP string) error {
	if clusterIP == "" {
		return errors.New("ClusterIP is missing")
	}

	return bootstrap.CreateOrUpdateService(client, &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kube-dns",
			Namespace: metav1.NamespaceSystem,
			Labels: map[string]string{
				"k8s-app":                         "kube-dns",
				"kubernetes.io/cluster-service":   "true",
				"addonmanager.kubernetes.io/mode": "Reconcile",
				"kubernetes.io/name":              "CoreDNS",
			},
			Annotations: map[string]string{
				"prometheus.io/port":   "9153",
				"prometheus.io/scrape": "true",
			},
			ResourceVersion: "0",
		},
		Spec: v1.ServiceSpec{
			Selector: map[string]string{
				"k8s-app": "kube-dns",
			},
			ClusterIP: clusterIP,
			Ports: []v1.ServicePort{
				{Name: "dns", Port: 53, Protocol: v1.ProtocolUDP},
				{Name: "dns-tcp", Port: 53, Protocol: v1.ProtocolTCP},
				{Name: "metrics", Port: 9153, Protocol: v1.ProtocolTCP},
			},
		},
	})
}

func createCoreDNSDeployment(client clientset.Interface, image string) error {
	if image == "" {
		image = "sapcc/coredns:1.6.2"
	}

	manifest := `
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: coredns
  namespace: kube-system
  labels:
    k8s-app: kube-dns
    kubernetes.io/cluster-service: "true"
    addonmanager.kubernetes.io/mode: Reconcile
    kubernetes.io/name: "CoreDNS"
spec:
  replicas: 2
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 1
  selector:
    matchLabels:
      k8s-app: kube-dns
  template:
    metadata:
      labels:
        k8s-app: kube-dns
      annotations:
        seccomp.security.alpha.kubernetes.io/pod: 'docker/default'
    spec:
      affinity:
        podAntiAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
          - labelSelector:
              matchExpressions:
              - key: k8s-app
                operator: In
                values:
                - kube-dns
            topologyKey: kubernetes.io/hostname
      priorityClassName: system-cluster-critical
      serviceAccountName: coredns
      tolerations:
        - key: "CriticalAddonsOnly"
          operator: "Exists"
      nodeSelector:
        beta.kubernetes.io/os: linux
      containers:
      - name: coredns
        image: {{ .Image }}
        imagePullPolicy: IfNotPresent
        resources:
          limits:
            memory: 170Mi
          requests:
            cpu: 100m
            memory: 70Mi
        args: [ "-conf", "/etc/coredns/Corefile" ]
        volumeMounts:
        - name: config-volume
          mountPath: /etc/coredns
          readOnly: true
        ports:
        - containerPort: 53
          name: dns
          protocol: UDP
        - containerPort: 53
          name: dns-tcp
          protocol: TCP
        - containerPort: 9153
          name: metrics
          protocol: TCP
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
            scheme: HTTP
          initialDelaySeconds: 60
          timeoutSeconds: 5
          successThreshold: 1
          failureThreshold: 5
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
            scheme: HTTP
        securityContext:
          allowPrivilegeEscalation: false
          capabilities:
            add:
            - NET_BIND_SERVICE
            drop:
            - all
          readOnlyRootFilesystem: true
      dnsPolicy: Default
      volumes:
        - name: config-volume
          configMap:
            name: coredns
            items:
            - key: Corefile
              path: Corefile
`

	data := struct {
		Image string
	}{
		image,
	}

	template, err := bootstrap.RenderManifest(manifest, data)
	if err != nil {
		return err
	}

	deployment, _, err := serializer.NewCodecFactory(clientsetscheme.Scheme).UniversalDeserializer().Decode(template, nil, &extensions.Deployment{})
	if err != nil {
		return err
	}

	return bootstrap.CreateOrUpdateDeployment(client, deployment.(*extensions.Deployment))
}

func createCoreDNSDeployment116(client clientset.Interface, image string) error {
	if image == "" {
		image = "sapcc/coredns:1.6.2"
	}

	manifest := `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: coredns
  namespace: kube-system
  labels:
    k8s-app: kube-dns
    kubernetes.io/cluster-service: "true"
    addonmanager.kubernetes.io/mode: Reconcile
    kubernetes.io/name: "CoreDNS"
spec:
  replicas: 2
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 1
  selector:
    matchLabels:
      k8s-app: kube-dns
  template:
    metadata:
      labels:
        k8s-app: kube-dns
      annotations:
        seccomp.security.alpha.kubernetes.io/pod: 'docker/default'
    spec:
      affinity:
        podAntiAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
          - labelSelector:
              matchExpressions:
              - key: k8s-app
                operator: In
                values:
                - kube-dns
            topologyKey: kubernetes.io/hostname
      priorityClassName: system-cluster-critical
      serviceAccountName: coredns
      tolerations:
        - key: "CriticalAddonsOnly"
          operator: "Exists"
      nodeSelector:
        beta.kubernetes.io/os: linux
      containers:
      - name: coredns
        image: {{ .Image }}
        imagePullPolicy: IfNotPresent
        resources:
          limits:
            memory: 170Mi
          requests:
            cpu: 100m
            memory: 70Mi
        args: [ "-conf", "/etc/coredns/Corefile" ]
        volumeMounts:
        - name: config-volume
          mountPath: /etc/coredns
          readOnly: true
        ports:
        - containerPort: 53
          name: dns
          protocol: UDP
        - containerPort: 53
          name: dns-tcp
          protocol: TCP
        - containerPort: 9153
          name: metrics
          protocol: TCP
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
            scheme: HTTP
          initialDelaySeconds: 60
          timeoutSeconds: 5
          successThreshold: 1
          failureThreshold: 5
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
            scheme: HTTP
        securityContext:
          allowPrivilegeEscalation: false
          capabilities:
            add:
            - NET_BIND_SERVICE
            drop:
            - all
          readOnlyRootFilesystem: true
      dnsPolicy: Default
      volumes:
        - name: config-volume
          configMap:
            name: coredns
            items:
            - key: Corefile
              path: Corefile
`

	data := struct {
		Image string
	}{
		image,
	}

	template, err := bootstrap.RenderManifest(manifest, data)
	if err != nil {
		return err
	}

	deployment, _, err := serializer.NewCodecFactory(clientsetscheme.Scheme).UniversalDeserializer().Decode(template, nil, &apps.Deployment{})
	if err != nil {
		return err
	}

	return bootstrap.CreateOrUpdateDeployment116(client, deployment.(*apps.Deployment))
}
