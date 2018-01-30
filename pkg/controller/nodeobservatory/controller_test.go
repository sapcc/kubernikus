package nodeobservatory

import (
	"os"
	"testing"
	"time"

	api_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"

	kitLog "github.com/go-kit/kit/log"
	"github.com/stretchr/testify/assert"

	"github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/client/kubernetes"
	"github.com/sapcc/kubernikus/pkg/controller/config"
	kubernikusfake "github.com/sapcc/kubernikus/pkg/generated/clientset/fake"
	kubernikus_informers "github.com/sapcc/kubernikus/pkg/generated/informers/externalversions"
)

const (
	KlusterName      = "fakeKluster"
	KlusterNamespace = "default"
	NodeName         = "node0"
	Reconciliation   = 1 * time.Minute
)

func createFakeNodeObservatory(kluster *v1.Kluster, node *api_v1.Node) NodeObservatory {
	dummyCertPEM := []byte(`
-----BEGIN CERTIFICATE-----
MIIDZzCCAk+gAwIBAgIJAKR66aipzKWxMA0GCSqGSIb3DQEBBQUAME8xCzAJBgNV
BAYTAkNOMQswCQYDVQQIEwJHRDELMAkGA1UEBxMCU1oxEzARBgNVBAoTCkFjbWUs
IEluYy4xETAPBgNVBAMTCE15Um9vdENBMB4XDTE3MTIxMTIxMTAxOVoXDTE3MTIy
MTIxMTAxOVowVjELMAkGA1UEBhMCQ04xCzAJBgNVBAgTAkdEMQswCQYDVQQHEwJT
WjETMBEGA1UEChMKQWNtZSwgSW5jLjEYMBYGA1UEAxMPd3d3LmV4YW1wbGUuY29t
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAq2X0gM8UyKKJFPFaBCiY
cN43/h4NgvmL/2BQ7JretsD1iDZm7bicYLjJn7wkzhuAz2HfxO6Hr0XAAEllqCTu
tqenSM+sX8hROUBA5KDkdDf1QZddEBwWbTVwfYw4vdSx05OdACzH557eKJvHp071
+V53EoPVHRxk8jl6FX/57+r8P/MbOSPAsm84da79UNKpp+1gh+kY1bK2bTGXMEr7
idm2BqOWmz9nFpR+h94wUhq7Hxdwyg6pwxUcAy0m2w8J73JhZdxbSvCJX8aSEbjU
C7W9Xpm2zxXDNvUh7iHtIIqlXqHLa0ZoaIuRK3U4DTiFJYJqASl8vSUKggapX++s
uwIDAQABoz8wPTA7BgNVHREENDAyggtleGFtcGxlLmNvbYIPd3d3LmV4YW1wbGUu
Y29tghJ3d3cubXktZXhhbXBsZS5jb20wDQYJKoZIhvcNAQEFBQADggEBAI8XdSbw
Dyeff77OAyKGG5w5Dzhy6Czhtb2C7mtRonE5iUlNC4lD2wDyRGMVo1kK/Q/Z8WRy
g4arhwElX1FwrnulU/P/7cQn1mvtFfhXrwXXrlEUJjpA83a9sitsGuTImv+1tobk
QC4xVuBo9IbLApCtISBT9didHigZfHqIZjrfJNi46rQEr+rh2ao5xvqGOVLwwBur
mm7Lfxh5ET2KRZPQ31S1mL0i9G6gUqVw4eybmmvnyyP10VP5NYVuHECQrcUPvEvg
wcNw5Ufl8Idbpz5xKFr4K12Jd9q8dPDskw3LA8l3iUuIigG7mKCdFB9HSCro7OAh
t6cvIMqAVp6arvU=
-----END CERTIFICATE-----
`)

	dummyKeyPEM := []byte(
		`
-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEAq2X0gM8UyKKJFPFaBCiYcN43/h4NgvmL/2BQ7JretsD1iDZm
7bicYLjJn7wkzhuAz2HfxO6Hr0XAAEllqCTutqenSM+sX8hROUBA5KDkdDf1QZdd
EBwWbTVwfYw4vdSx05OdACzH557eKJvHp071+V53EoPVHRxk8jl6FX/57+r8P/Mb
OSPAsm84da79UNKpp+1gh+kY1bK2bTGXMEr7idm2BqOWmz9nFpR+h94wUhq7Hxdw
yg6pwxUcAy0m2w8J73JhZdxbSvCJX8aSEbjUC7W9Xpm2zxXDNvUh7iHtIIqlXqHL
a0ZoaIuRK3U4DTiFJYJqASl8vSUKggapX++suwIDAQABAoIBAC8Ii0/Ng6aK85ML
p+f8O9i9IiBZntuSHxi1FX/X/8WmrbnzM8uIHWFtU+bBelgUtAQ0l3MzAYjXWxi5
C2xYtijpWL5iPqsKDT/ooeYbQJWjxWl6X89L5duSDoxlLizpcOLeXvbtUu38ano6
RU9kG5uSkJpEEvcqE4lkvFuqAqwTHKxrYLMI9yFsJ94wHmfXO1kizU0Y2dYwxiqa
eVs9G5tEC3RJzb0OiyHuIlPoXqPLPZXetTPqm21vr67GxPYZJMDy/rRBX5b+f/6r
73w8jXc+CkyRnXODN87D4HWXJLPMYC0JMDRRaA7Z7eCNx9YOQ3UDxZkdlr3cE89z
C5wrxgECgYEA1ChMq91oxIY9OhZI6sqCv2byS/GSvKDV7VaDZZLG/uZRlSfsZwz1
ggGs9idQPS3k0RsSJPEd0y2RkrGnNFSYVIGBgcegdKSUAxKP9JBS1ieXIH7yZKhE
RZAWerbCScvihJOi1fmVxUKslFR4fVtxAaNsxHDtdYyqUcoOLMswyvUCgYEAztFi
1PYDRHOqHhyNS5QgClzLKNvxl6o8CksOQFaXP7VNRWufeFc5YTP5sVgz8vfiiPmY
udEzIAkOgoHxn6p5+HlyPre95WRWuy6GIQiD6u7jX674tzQEzZEQ/63tYn3dmDtg
BeDQHpbPi/NKgMoQCHjfD74yPIyuZhss8Pn+Ku8CgYEAoWhXjJnST1Hh2wOBTj/r
4TqtNGIBxUiH+R1MskZM5zjK8LODA5O0ZMhpkoyuWx1DbGMwFrLqgfO1QOmvz/xc
OE6e/OGnjZZ4lS3WH7Z9jzhnne129GWgK1xH/ex1PDfFih/YTvqnm3/yVJc/Y//h
peFzqrBPuJLgMYGL70BXStECgYBlwcXjzAs9gb9Aw4GNnxrInnFi8ByFJ8AUvGsN
os0WDmkvb81tk1TrC3yeEiy1Lduq00uemVyTNYGLGs48Zc9PPsnEK/llxSGbRT+/
PwZQ8Cq1KEy9Lv3x+p8nfXbfz9fYj9Yl7j/X3RHO5OxSQ5jx4i61+zmSaxFfsZ1C
D25LxwKBgB+CY9XHW/k+18tuu85aJXVFYaQG3BPYFr1tYyIP7WLO1Ea6daI6J+0N
41ljcA8+cVeUbw7K+lNd/uRiJfgTeTmYva9dFFEkEB+lOK+nDal3ZdIaZFEWVDh4
g1GGPll8zndfDTDecJrHjV4G7Bj+QVGuArVG0dAF6Gk3lEzS9AVb
-----END RSA PRIVATE KEY-----
`)

	secret := &api_v1.Secret{
		ObjectMeta: meta_v1.ObjectMeta{
			Namespace: KlusterNamespace,
			Name:      KlusterName,
		},
		Data: map[string][]byte{
			"apiserver-clients-cluster-admin.pem":     dummyCertPEM,
			"apiserver-clients-cluster-admin-key.pem": dummyKeyPEM,
			"tls-ca.pem":                              dummyCertPEM,
		},
	}

	fakeKubernetesClientset := fake.NewSimpleClientset(node, secret)
	fakeKubernikusClientset := kubernikusfake.NewSimpleClientset(kluster)
	kubernikusInformerFactory := kubernikus_informers.NewSharedInformerFactory(fakeKubernikusClientset, Reconciliation)

	return NodeObservatory{
		Clients: config.Clients{
			Kubernetes: fakeKubernetesClientset,
			Kubernikus: fakeKubernikusClientset,
			Satellites: kubernetes.NewSharedClientFactory(
				fakeKubernetesClientset.Core().Secrets(KlusterNamespace),
				kubernikusInformerFactory.Kubernikus().V1().Klusters().Informer(),
				kitLog.NewLogfmtLogger(kitLog.NewSyncWriter(os.Stdout)),
			),
		},
	}
}

//FIXME: mock list,watch for nodes not working. so the NodeInformer will be empty
/*func TestCreateWatcherForNodeInKluster(t *testing.T) {

	kluster := &v1.Kluster{
		ObjectMeta: meta_v1.ObjectMeta{
			Namespace: KlusterNamespace,
			Name:      KlusterName,
		},
	}

	node := &api_v1.Node{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: NodeName,
		},
		Status: api_v1.NodeStatus{
			Phase: api_v1.NodeRunning,
		},
	}

	no := createFakeNodeObservatory(kluster, node)

	no.AddEventHandlerFuncs(
		func(kluster *v1.Kluster, node *api_v1.Node) {
			no.logger.Log("kluster %s/%s: added node %s", kluster.GetNamespace(), kluster.GetName(), node.GetName())
		},
		func(kluster *v1.Kluster, nodeCur, nodeOld *api_v1.Node) {
			no.logger.Log("updated %s/%s: added node %s", kluster.GetNamespace(), kluster.GetName(), nodeOld.GetName())
		},
		func(kluster *v1.Kluster, node *api_v1.Node) {
			no.logger.Log("kluster %s/%s: added node %s", kluster.GetNamespace(), kluster.GetName(), node.GetName())
		},
	)

	if err := no.createAndWatchNodeInformerForKluster(kluster); err != nil {
		t.Errorf("failed to createAndWatchNodeInformerForKluster: %v", err)
	}

	key, err := cache.MetaNamespaceKeyFunc(kluster)
	if err != nil {
		t.Errorf("couldn't create key for kluster %v: %v", kluster, err)
	}

	i, ok := no.nodeInformerMap.Load(key)
	if !ok {
		t.Errorf("could'nt find any watcher for kluster %s", key)
	}

	informer := i.(*NodeInformer)
	assert.NotNil(t, informer)
}*/

func TestGetStoreForKluster(t *testing.T) {
	kluster := &v1.Kluster{
		ObjectMeta: meta_v1.ObjectMeta{
			Namespace: KlusterNamespace,
			Name:      KlusterName,
		},
	}

	node := &api_v1.Node{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: NodeName,
		},
		Status: api_v1.NodeStatus{
			Phase: api_v1.NodeRunning,
		},
	}

	no := createFakeNodeObservatory(kluster, node)
	stopCh := make(chan struct{})
	ni := NodeInformer{kluster: kluster, stopCh: stopCh}
	ni.SharedIndexInformer = cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options meta_v1.ListOptions) (runtime.Object, error) {
				return no.Clients.Kubernetes.CoreV1().Nodes().List(meta_v1.ListOptions{})
			},
			WatchFunc: func(options meta_v1.ListOptions) (watch.Interface, error) {
				return no.Clients.Kubernetes.CoreV1().Nodes().Watch(meta_v1.ListOptions{})
			},
		},
		&api_v1.Node{},
		NodeResyncPeriod,
		cache.Indexers{},
	)
	if err := ni.SharedIndexInformer.GetStore().Add(node); err != nil {
		t.Error(err)
	}

	key, _ := cache.MetaNamespaceKeyFunc(kluster)
	no.nodeInformerMap.Store(key, &ni)

	store, err := no.GetStoreForKluster(kluster)
	if err != nil {
		t.Error(err)
	}

	assert.Contains(t, store.List(), node)
}
