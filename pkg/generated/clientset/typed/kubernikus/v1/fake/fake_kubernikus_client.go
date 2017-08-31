package fake

import (
	v1 "github.com/sapcc/kubernikus/pkg/generated/clientset/typed/kubernikus/v1"
	rest "k8s.io/client-go/rest"
	testing "k8s.io/client-go/testing"
)

type FakeKubernikusV1 struct {
	*testing.Fake
}

func (c *FakeKubernikusV1) Klusters(namespace string) v1.KlusterInterface {
	return &FakeKlusters{c, namespace}
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *FakeKubernikusV1) RESTClient() rest.Interface {
	var ret *rest.RESTClient
	return ret
}
