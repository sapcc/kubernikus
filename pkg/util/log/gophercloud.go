package log

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	kitlog "github.com/go-kit/log"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
)

func NewLoggingProviderClient(endpoint string, logger kitlog.Logger) (*gophercloud.ProviderClient, error) {
	providerClient, err := openstack.NewClient(endpoint)
	if err != nil {
		return nil, err
	}

	providerClient.UserAgent.Prepend("kubernikus")
	providerClient.UseTokenLock()

	transport := providerClient.HTTPClient.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	providerClient.HTTPClient.Transport = &loggingRoundTripper{
		transport,
		kitlog.With(logger, "api", "egress"),
	}

	return providerClient, err
}

type loggingRoundTripper struct {
	rt     http.RoundTripper
	Logger kitlog.Logger
}

func (lrt *loggingRoundTripper) RoundTrip(request *http.Request) (response *http.Response, err error) {
	defer func(begin time.Time) {
		keyvals := make([]interface{}, 0, 6)

		if response != nil {
			keyvals = append(keyvals,
				"status", response.StatusCode,
				"openstack_id", strings.Join(requestIds(response), ","))
		}

		if id := request.Context().Value(KubernikusRequestID); id != nil {
			keyvals = append(keyvals, "id", fmt.Sprintf("%s", id))
		}

		keyvals = append(keyvals,
			"took", time.Since(begin),
			"v", 2,
			"err", err,
		)

		log(lrt.Logger, request, keyvals...)
	}(time.Now())

	return lrt.rt.RoundTrip(request)
}

func requestIds(response *http.Response) []string {
	ids := []string{}

	if id := response.Header.Get("X-Openstack-Request-ID"); id != "" {
		ids = append(ids, id)
	}

	if id := response.Header.Get("X-Compute-Request-ID"); id != "" {
		ids = append(ids, id)
	}

	return ids
}
