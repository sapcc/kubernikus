package helm

import (
	"fmt"

	"github.com/go-kit/log"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/kube"
)

func NewClient3(releaseNamespace, kubeConfig, kubeContext string, logger log.Logger) (*action.Configuration, error) {
	client3 := &action.Configuration{}
	err := client3.Init(kube.GetConfig(kubeConfig, kubeContext, releaseNamespace), releaseNamespace, "secrets", func(format string, v ...interface{}) {
		logger.Log("component", "helm3", "msg", fmt.Sprintf(format, v), "v", 2)
	})
	if err != nil {
		return nil, err
	}
	return client3, nil
}
