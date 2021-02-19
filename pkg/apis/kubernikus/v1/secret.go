package v1

import (
	"encoding/json"
	"strings"

	corev1 "k8s.io/api/core/v1"
)

type Secret struct {
	Openstack

	NodePassword   string `json:"node-password,omitempty"`
	BootstrapToken string `json:"bootstrapToken"`

	DexClientSecret   string `json:"dex-client-secret,omitempty"`
	DexStaticPassword string `json:"dex-static-password,omitempty"`

	Certificates

	ExtraValues string `json:"extra-values,omitempty"`
}

func NewSecret(secret *corev1.Secret) (*Secret, error) {
	var data = make(map[string]string, len(secret.Data))
	for k, v := range secret.Data {
		data[k] = string(v)
	}
	jdata, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	var s = new(Secret)
	if err := json.Unmarshal(jdata, s); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Secret) ToData() (map[string][]byte, error) {
	data, err := s.ToStringData()
	if err != nil {
		return nil, err
	}
	var result = make(map[string][]byte, len(data))
	for k, v := range data {
		result[k] = []byte(v)
	}
	return result, nil
}

func (s *Secret) ToStringData() (map[string]string, error) {
	jdata, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}
	var result map[string]string
	if err := json.Unmarshal(jdata, &result); err != nil {
		return nil, err
	}
	result["apiserver-clients-and-nodes-ca.pem"] = strings.TrimSuffix(s.ApiserverClientsCACertifcate, "\n") + "\n" + s.ApiserverNodesCACertificate
	return result, nil
}

type Openstack struct {
	AuthURL           string `json:"openstack-auth-url"`
	Region            string `json:"openstack-region"`
	Username          string `json:"openstack-username,omitempty"`
	DomainName        string `json:"openstack-domain-name,omitempty"`
	Password          string `json:"openstack-password"`
	ProjectID         string `json:"openstack-project-id"`
	ProjectDomainName string `json:"openstack-project-domain-name,omitempty"`
}

type Certificates struct {
	ApiserverClientsCAPrivateKey                     string `json:"apiserver-clients-ca-key.pem"`
	ApiserverClientsCACertifcate                     string `json:"apiserver-clients-ca.pem"`
	ApiserverClientsClusterAdminPrivateKey           string `json:"apiserver-clients-cluster-admin-key.pem"`
	ApiserverClientsClusterAdminCertificate          string `json:"apiserver-clients-cluster-admin.pem"`
	ApiserverClientsKubeControllerManagerPrivateKey  string `json:"apiserver-clients-system-kube-controller-manager-key.pem"`
	ApiserverClientsKubeControllerManagerCertificate string `json:"apiserver-clients-system-kube-controller-manager.pem"`
	ApiserverClientsKubeProxyPrivateKey              string `json:"apiserver-clients-system-kube-proxy-key.pem"`
	ApiserverClientsKubeProxyCertificate             string `json:"apiserver-clients-system-kube-proxy.pem"`
	ApiserverClientsKubeSchedulerPrivateKey          string `json:"apiserver-clients-system-kube-scheduler-key.pem"`
	ApiserverClientsKubeSchedulerCertificate         string `json:"apiserver-clients-system-kube-scheduler.pem"`
	ApiserverClientsKubernikusWormholePrivateKey     string `json:"apiserver-clients-kubernikus-wormhole-key.pem"`
	ApiserverClientsKubernikusWormholeCertificate    string `json:"apiserver-clients-kubernikus-wormhole.pem"`
	ApiserverClientsCSIControllerPrivateKey          string `json:"apiserver-clients-csi-controller-key.pem"`
	ApiserverClientsCSIControllerCertificate         string `json:"apiserver-clients-csi-controller.pem"`

	ApiserverNodesCAPrivateKey  string `json:"apiserver-nodes-ca-key.pem"`
	ApiserverNodesCACertificate string `json:"apiserver-nodes-ca.pem"`

	EtcdClientsCAPrivateKey         string `json:"etcd-clients-ca-key.pem"`
	EtcdClientsCACertificate        string `json:"etcd-clients-ca.pem"`
	EtcdClientsApiserverPrivateKey  string `json:"etcd-clients-apiserver-key.pem"`
	EtcdClientsApiserverCertificate string `json:"etcd-clients-apiserver.pem"`
	EtcdClientsBackupPrivateKey     string `json:"etcd-clients-backup-key.pem"`
	EtcdClientsBackupCertificate    string `json:"etcd-clients-backup.pem"`
	EtcdClientsDexPrivateKey        string `json:"etcd-clients-dex-key.pem"`
	EtcdClientsDexCertificate       string `json:"etcd-clients-dex.pem"`

	EtcdPeersCAPrivateKey  string `json:"etcd-peers-ca-key.pem"`
	EtcdPeersCACertificate string `json:"etcd-peers-ca.pem"`

	KubeletClientsCAPrivateKey         string `json:"kubelet-clients-ca-key.pem"`
	KubeletClientsCACertificate        string `json:"kubelet-clients-ca.pem"`
	KubeletClientsApiserverPrivateKey  string `json:"kubelet-clients-apiserver-key.pem"`
	KubeletClientsApiserverCertificate string `json:"kubelet-clients-apiserver.pem"`

	TLSCAPrivateKey         string `json:"tls-ca-key.pem"`
	TLSCACertificate        string `json:"tls-ca.pem"`
	TLSApiserverPrivateKey  string `json:"tls-apiserver-key.pem"`
	TLSApiserverCertificate string `json:"tls-apiserver.pem"`
	TLSWormholePrivateKey   string `json:"tls-wormhole-key.pem"`
	TLSWormholeCertificate  string `json:"tls-wormhole.pem"`

	TLSEtcdCAPrivateKey  string `json:"tls-etcd-ca-key.pem"`
	TLSEtcdCACertificate string `json:"tls-etcd-ca.pem"`
	TLSEtcdPrivateKey    string `json:"tls-etcd-key.pem"`
	TLSEtcdCertificate   string `json:"tls-etcd.pem"`

	AggregationCAPrivateKey          string `json:"aggregation-ca-key.pem"`
	AggregationCACertificate         string `json:"aggregation-ca.pem"`
	AggregationAggregatorPrivateKey  string `json:"aggregation-aggregator-key.pem"`
	AggregationAggregatorCertificate string `json:"aggregation-aggregator.pem"`
}

func (s *Certificates) ToStringData() (map[string]string, error) {
	jdata, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}
	var result map[string]string
	if err := json.Unmarshal(jdata, &result); err != nil {
		return nil, err
	}
	return result, nil
}
