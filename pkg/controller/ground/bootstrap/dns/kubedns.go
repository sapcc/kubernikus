package dns

import (
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	extensions "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	clientset "k8s.io/client-go/kubernetes"
	clientsetscheme "k8s.io/client-go/kubernetes/scheme"

	"github.com/sapcc/kubernikus/pkg/api/spec"
	"github.com/sapcc/kubernikus/pkg/controller/ground/bootstrap"
)

const (
	SERVICE_ACCOUNT    = "kube-dns"
	CONFIGMAP          = "kube-dns"
	DEFAULT_REPOSITORY = "sapcc" // Used to be gcr.io/google_containers but that is not working in china

	// If you change this version you need to ensure these images are mirrored:
	//   * k8s-dns-kube-dns-amd64
	//   * k8s-dns-sidecar-amd64
	//   * k8s-dns-dnsmasq-nanny-amd64
	//
	// We have a pipline that should do this automatically
	DEFAULT_VERSION = "1.14.13"
)

var (
	DEFAULT_DOMAIN = spec.MustDefaultString("KlusterSpec", "dnsDomain")
)

type DeploymentOptions struct {
	Repository string
	Version    string
	Domain     string
}

type ServiceOptions struct {
	ClusterIP string
}

func SeedKubeDNS(client clientset.Interface, repository, version, domain, clusterIP string) error {
	if repository == "" {
		repository = DEFAULT_REPOSITORY
	}

	if version == "" {
		version = DEFAULT_VERSION
	}

	if domain == "" {
		domain = DEFAULT_DOMAIN
	}

	if clusterIP == "" {
		return errors.New("Cluster IP for kube-dns service missing.")
	}

	if err := createKubeDNSServiceAccount(client); err != nil {
		return errors.Wrap(err, "Failed to ensure kubedns service account")
	}

	if err := createKubeDNSConfigMap(client); err != nil {
		return errors.Wrap(err, "Failed to ensure kubedns configmap")
	}

	if err := createKubeDNSDeployment(client, repository, version, domain); err != nil {
		return errors.Wrap(err, "Failed to ensure kubedns deployment")
	}

	if err := createKubeDNSService(client, clusterIP); err != nil {
		return errors.Wrap(err, "Failed to ensure kubedns service")
	}

	return nil
}

func createKubeDNSServiceAccount(client clientset.Interface) error {
	return bootstrap.CreateOrUpdateServiceAccount(client, &v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      SERVICE_ACCOUNT,
			Namespace: metav1.NamespaceSystem,
			Labels: map[string]string{
				"kubernetes.io/cluster-service":   "true",
				"addonmanager.kubernetes.io/mode": "Reconcile",
			},
		},
	})
}

func createKubeDNSConfigMap(client clientset.Interface) error {
	return bootstrap.CreateOrUpdateConfigMap(client, &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      CONFIGMAP,
			Namespace: metav1.NamespaceSystem,
			Labels: map[string]string{
				"addonmanager.kubernetes.io/mode": "EnsureExists",
			},
		},
	})
}

func createKubeDNSDeployment(client clientset.Interface, repository, version, domain string) error {
	options := &DeploymentOptions{
		Repository: repository,
		Version:    version,
		Domain:     domain,
	}

	deployment, err := getKubeDNSDeployment(options)
	if err != nil {
		return err
	}

	if err := bootstrap.CreateOrUpdateDeployment(client, deployment); err != nil {
		return err
	}

	return nil
}

func createKubeDNSService(client clientset.Interface, clusterIP string) error {
	options := &ServiceOptions{
		ClusterIP: clusterIP,
	}

	service, err := getKubeDNSService(options)
	if err != nil {
		return err
	}

	if err := bootstrap.CreateOrUpdateService(client, service); err != nil {
		return err
	}

	return nil
}

func getKubeDNSDeployment(options *DeploymentOptions) (*extensions.Deployment, error) {
	manifest := KubeDNSDeployment_v20171016

	template, err := bootstrap.RenderManifest(manifest, options)
	if err != nil {
		return nil, err
	}

	deployment, _, err := serializer.NewCodecFactory(clientsetscheme.Scheme).UniversalDeserializer().Decode(template, nil, &extensions.Deployment{})
	if err != nil {
		return nil, err
	}

	return deployment.(*extensions.Deployment), nil
}

func getKubeDNSService(options *ServiceOptions) (*v1.Service, error) {
	manifest := KubeDNSService_v20171016

	template, err := bootstrap.RenderManifest(manifest, options)
	if err != nil {
		return nil, err
	}

	service, _, err := serializer.NewCodecFactory(clientsetscheme.Scheme).UniversalDeserializer().Decode(template, nil, &v1.Service{})
	if err != nil {
		return nil, err
	}

	return service.(*v1.Service), nil
}

const (
	KubeDNSDeployment_v20171016 = `
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: kube-dns
  namespace: kube-system
  labels:
    k8s-app: kube-dns
    kubernetes.io/cluster-service: "true"
    addonmanager.kubernetes.io/mode: Reconcile
spec:
  replicas: 2
  strategy:
    rollingUpdate:
      maxSurge: 10%
      maxUnavailable: 0
  selector:
    matchLabels:
      k8s-app: kube-dns
  template:
    metadata:
      labels:
        k8s-app: kube-dns
      annotations:
        scheduler.alpha.kubernetes.io/critical-pod: ''
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
      tolerations:
      - key: "CriticalAddonsOnly"
        operator: "Exists"
      volumes:
      - name: kube-dns-config
        configMap:
          name: kube-dns
          optional: true
      containers:
      - name: kubedns
        image: {{ .Repository }}/k8s-dns-kube-dns-amd64:{{ .Version }}
        resources:
          # TODO: Set memory limits when we've profiled the container for large
          # clusters, then set request = limit to keep this container in
          # guaranteed class. Currently, this container falls into the
          # "burstable" category so the kubelet doesn't backoff from restarting it.
          limits:
            memory: 170Mi
          requests:
            cpu: 100m
            memory: 70Mi
        livenessProbe:
          httpGet:
            path: /healthcheck/kubedns
            port: 10054
            scheme: HTTP
          initialDelaySeconds: 60
          timeoutSeconds: 5
          successThreshold: 1
          failureThreshold: 5
        readinessProbe:
          httpGet:
            path: /readiness
            port: 8081
            scheme: HTTP
          # we poll on pod startup for the Kubernetes master service and
          # only setup the /readiness HTTP server once that's available.
          initialDelaySeconds: 3
          timeoutSeconds: 5
        args:
        - --domain={{ .Domain }}.
        - --dns-port=10053
        - --config-dir=/kube-dns-config
        - --v=2
        env:
        - name: PROMETHEUS_PORT
          value: "10055"
        ports:
        - containerPort: 10053
          name: dns-local
          protocol: UDP
        - containerPort: 10053
          name: dns-tcp-local
          protocol: TCP
        - containerPort: 10055
          name: metrics
          protocol: TCP
        volumeMounts:
        - name: kube-dns-config
          mountPath: /kube-dns-config
      - name: dnsmasq
        image: {{ .Repository }}/k8s-dns-dnsmasq-nanny-amd64:{{ .Version }}
        livenessProbe:
          httpGet:
            path: /healthcheck/dnsmasq
            port: 10054
            scheme: HTTP
          initialDelaySeconds: 60
          timeoutSeconds: 5
          successThreshold: 1
          failureThreshold: 5
        args:
        - -v=2
        - -logtostderr
        - -configDir=/etc/k8s/dns/dnsmasq-nanny
        - -restartDnsmasq=true
        - --
        - -k
        - --cache-size=1000
        - --no-negcache
        - --dns-loop-detect
        - --log-facility=-
        - --server=/{{ .Domain }}/127.0.0.1#10053
        - --server=/in-addr.arpa/127.0.0.1#10053
        - --server=/ip6.arpa/127.0.0.1#10053
        ports:
        - containerPort: 53
          name: dns
          protocol: UDP
        - containerPort: 53
          name: dns-tcp
          protocol: TCP
        # see: https://github.com/kubernetes/kubernetes/issues/29055 for details
        resources:
          requests:
            cpu: 150m
            memory: 20Mi
        volumeMounts:
        - name: kube-dns-config
          mountPath: /etc/k8s/dns/dnsmasq-nanny
      - name: sidecar
        image: {{ .Repository }}/k8s-dns-sidecar-amd64:{{ .Version }}
        livenessProbe:
          httpGet:
            path: /metrics
            port: 10054
            scheme: HTTP
          initialDelaySeconds: 60
          timeoutSeconds: 5
          successThreshold: 1
          failureThreshold: 5
        args:
        - --v=2
        - --logtostderr
        - --probe=kubedns,127.0.0.1:10053,kubernetes.default.svc.{{ .Domain }},5,SRV
        - --probe=dnsmasq,127.0.0.1:53,kubernetes.default.svc.{{ .Domain }},5,SRV
        ports:
        - containerPort: 10054
          name: metrics
          protocol: TCP
        resources:
          requests:
            memory: 20Mi
            cpu: 10m
      dnsPolicy: Default  # Don't use cluster DNS.
      serviceAccountName: kube-dns
`

	// KubeDNSService is the kube-dns Service manifest
	KubeDNSService_v20171016 = `
apiVersion: v1
kind: Service
metadata:
  name: kube-dns
  namespace: kube-system
  labels:
    k8s-app: kube-dns
    kubernetes.io/cluster-service: "true"
    addonmanager.kubernetes.io/mode: Reconcile
    kubernetes.io/name: "KubeDNS"
  # Without this resourceVersion value, an update of the Service between versions will yield:
  #   Service "kube-dns" is invalid: metadata.resourceVersion: Invalid value: "": must be specified for an update
  resourceVersion: "0"
spec:
  selector:
    k8s-app: kube-dns
  clusterIP: {{ .ClusterIP }}
  ports:
  - name: dns
    port: 53
    protocol: UDP
  - name: dns-tcp
    port: 53
    protocol: TCP
`
)
