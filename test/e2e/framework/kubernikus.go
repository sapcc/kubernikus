package framework

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/go-openapi/runtime"
	"k8s.io/apimachinery/pkg/util/wait"

	kubernikus "github.com/sapcc/kubernikus/pkg/api/client"
	"github.com/sapcc/kubernikus/pkg/api/client/operations"
	"github.com/sapcc/kubernikus/pkg/api/models"
)

type Kubernikus struct {
	Client   *kubernikus.Kubernikus
	AuthInfo runtime.ClientAuthInfoWriter
}

func NewKubernikusFramework(kubernikusURL *url.URL, authInfo runtime.ClientAuthInfoWriter) *Kubernikus {

	kubernikusClient := kubernikus.NewHTTPClientWithConfig(
		nil,
		&kubernikus.TransportConfig{
			Host:    kubernikusURL.Host,
			Schemes: []string{kubernikusURL.Scheme},
		},
	)

	return &Kubernikus{
		Client:   kubernikusClient,
		AuthInfo: authInfo,
	}
}

func (f *Kubernikus) WaitForKlusterPhase(klusterName string, expectedPhase models.KlusterPhase, timeout time.Duration) (finalPhase models.KlusterPhase, err error) {
	err = wait.PollImmediate(Poll, timeout, //nolint:staticcheck
		func() (bool, error) {
			cluster, err := f.Client.Operations.ShowCluster(
				operations.NewShowClusterParams().WithName(klusterName),
				f.AuthInfo,
			)

			if err != nil {
				return false, fmt.Errorf("Failed to show kluster: %v", err)
			}

			if cluster.Payload == nil {
				return false, fmt.Errorf("API return empty response")
			}

			finalPhase = cluster.Payload.Status.Phase

			return finalPhase == expectedPhase, nil
		})

	return finalPhase, err
}

// WaitForKlusterToHaveEnoughSchedulableNodes waits until the specified number of nodes equals the number of currently running, healthy, schedulable nodes
func (k *Kubernikus) WaitForKlusterToHaveEnoughSchedulableNodes(klusterName string, timeout time.Duration) error {
	return wait.PollImmediate(Poll, timeout, //nolint:staticcheck
		func() (done bool, err error) {
			cluster, err := k.Client.Operations.ShowCluster(
				operations.NewShowClusterParams().WithName(klusterName),
				k.AuthInfo,
			)
			if err != nil {
				return false, err
			}
			return IsAllNodePoolsOfKlusterReady(cluster.Payload), nil
		},
	)
}

func (k *Kubernikus) WaitForKlusterToBeDeleted(klusterName string, timeout time.Duration) error {
	count := 0
	overall := time.Now()
	return wait.PollImmediate(Poll, timeout, //nolint:staticcheck
		func() (done bool, err error) {
			count++
			req_start := time.Now()
			_, err = k.Client.Operations.ShowCluster(
				operations.NewShowClusterParams().WithName(klusterName),
				k.AuthInfo,
			)
			if err != nil {
				switch err.(type) {
				case *operations.ShowClusterDefault:
					result := err.(*operations.ShowClusterDefault)
					return result.Code() == 404, nil
				}
				return false, fmt.Errorf("Polling cluster state failed after %d tries and %s. Failed request took %s: %w", count, time.Now().Sub(overall), time.Now().Sub(req_start), err)
			}
			return false, nil
		},
	)
}

func (k *Kubernikus) WaitForKlusters(prefix string, count int, timeout time.Duration) error {
	return wait.PollImmediate(Poll, timeout, //nolint:staticcheck
		func() (done bool, err error) {
			res, err := k.Client.Operations.ListClusters(
				operations.NewListClustersParams(),
				k.AuthInfo,
			)

			if err != nil {
				return true, err
			}

			k := 0
			for _, kluster := range res.Payload {
				if strings.HasPrefix(kluster.Name, prefix) {
					k++
				}
			}

			if k == count {
				return true, nil
			}

			return false, nil
		},
	)
}
