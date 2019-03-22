package servicing

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	core_v1 "k8s.io/api/core/v1"
)

const (
	Stable2023_4_0 = `
		COREOS_BUILD=2023
		COREOS_BRANCH=4
		COREOS_PATCH=0
		COREOS_VERSION=2023.4.0
		COREOS_VERSION_ID=2023.4.0
		COREOS_BUILD_ID="2019-02-26-0016"
		COREOS_SDK_VERSION=2023.3.0`

	Stable2023_5_0 = `
		COREOS_BUILD=2023
		COREOS_BRANCH=5
		COREOS_PATCH=0
		COREOS_VERSION=2023.5.0
		COREOS_VERSION_ID=2023.5.0
		COREOS_BUILD_ID="2019-02-27-0343"
		COREOS_SDK_VERSION=2023.3.0`

	Beta2051_1_0 = `
		COREOS_BUILD=2051
		COREOS_BRANCH=1
		COREOS_PATCH=0
		COREOS_VERSION=2051.1.0
		COREOS_VERSION_ID=2051.1.0
		COREOS_BUILD_ID="2019-02-26-0031"
		COREOS_SDK_VERSION=2051.0.0`

	Alpha2065_0_0 = ` 
		COREOS_BUILD=2065
		COREOS_BRANCH=0
		COREOS_PATCH=0
		COREOS_VERSION=2065.0.0
		COREOS_VERSION_ID=2065.0.0
		COREOS_BUILD_ID="2019-02-26-0039"
		COREOS_SDK_VERSION=2051.0.0`

	UnexpectedResponse = `404 CoreOS has been bought by IBM`
)

type RoundTripFunc func(req *http.Request) *http.Response

func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

func NewFakeLatestCoreOSVersion(t *testing.T, version string) *LatestCoreOSVersion {
	body := fmt.Sprintf("COREOS_VERSION=%s", version)
	subject := &LatestCoreOSVersion{}
	subject.Client = NewTestClient(t, "https://stable.release.core-os.net/amd64-usr/current/version.txt", body, nil)
	return subject
}

func NewTestClient(t *testing.T, baseURL, body string, count *int) *http.Client {
	fn := func(req *http.Request) *http.Response {
		assert.Equal(t, req.URL.String(), baseURL)
		if count != nil {
			*count = *count + 1
		}
		return &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(bytes.NewBufferString(body)),
			Header:     make(http.Header),
		}
	}

	return &http.Client{
		Transport: RoundTripFunc(fn),
	}
}

func TestServicingCoreOSVersionStable(t *testing.T) {
	now = func() time.Time { return time.Date(2019, 2, 3, 4, 0, 0, 0, time.UTC) }
	count := 0
	subject := &LatestCoreOSVersion{}
	subject.Client = NewTestClient(t, "https://stable.release.core-os.net/amd64-usr/current/version.txt", Stable2023_4_0, &count)

	t.Run("fetches correct version", func(t *testing.T) {
		version, err := subject.Stable()
		assert.NoError(t, err)
		assert.NotNil(t, version)
		assert.Equal(t, "2023.4.0", version.String())
	})

	subject.Client = NewTestClient(t, "https://stable.release.core-os.net/amd64-usr/current/version.txt", Stable2023_5_0, &count)

	t.Run("caches result", func(t *testing.T) {
		version, err := subject.Stable()
		assert.NoError(t, err)
		assert.NotNil(t, version)
		assert.Equal(t, "2023.4.0", version.String())
		assert.Equal(t, 1, count)
	})

	now = func() time.Time {
		return time.Date(2019, 2, 3, 4, 0, 0, 0, time.UTC).Add(coreOSVersionFetchInterval).Add(1 * time.Minute)
	}

	t.Run("invalidates cached result", func(t *testing.T) {
		version, err := subject.Stable()
		assert.NoError(t, err)
		assert.NotNil(t, version)
		assert.Equal(t, "2023.5.0", version.String())
		assert.Equal(t, 2, count)
	})

	subject = &LatestCoreOSVersion{}
	subject.Client = NewTestClient(t, "https://stable.release.core-os.net/amd64-usr/current/version.txt", UnexpectedResponse, &count)

	t.Run("garble from coreos servers", func(t *testing.T) {
		version, err := subject.Stable()
		assert.Error(t, err)
		assert.Nil(t, version)
	})
}

func TestServicingCoreOSVersionBeta(t *testing.T) {
	now = func() time.Time { return time.Date(2019, 2, 3, 4, 0, 0, 0, time.UTC) }
	subject := &LatestCoreOSVersion{}
	subject.Client = NewTestClient(t, "https://beta.release.core-os.net/amd64-usr/current/version.txt", Beta2051_1_0, nil)

	t.Run("fetches correct version", func(t *testing.T) {
		version, err := subject.Beta()
		assert.NoError(t, err)
		assert.NotNil(t, version)
		assert.Equal(t, "2051.1.0", version.String())
	})
}

func TestServicingCoreOSVersionAlpha(t *testing.T) {
	now = func() time.Time { return time.Date(2019, 2, 3, 4, 0, 0, 0, time.UTC) }
	subject := &LatestCoreOSVersion{}
	subject.Client = NewTestClient(t, "https://alpha.release.core-os.net/amd64-usr/current/version.txt", Alpha2065_0_0, nil)

	t.Run("fetches correct version", func(t *testing.T) {
		version, err := subject.Alpha()
		assert.NoError(t, err)
		assert.NotNil(t, version)
		assert.Equal(t, "2065.0.0", version.String())
	})
}

func TestServicingCoreOSVersionIsNodeUptodate(t *testing.T) {
	subject := &LatestCoreOSVersion{}
	subject.Client = NewTestClient(t, "https://stable.release.core-os.net/amd64-usr/current/version.txt", Stable2023_4_0, nil)

	t.Run("outdated node", func(t *testing.T) {
		node := &core_v1.Node{
			Status: core_v1.NodeStatus{
				NodeInfo: core_v1.NodeSystemInfo{
					OSImage: "Container Linux by CoreOS 1800.6.0 (Rhyolite)",
				},
			},
		}

		result, err := subject.IsNodeUptodate(node)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("recent node", func(t *testing.T) {
		node := &core_v1.Node{
			Status: core_v1.NodeStatus{
				NodeInfo: core_v1.NodeSystemInfo{
					OSImage: "Container Linux by CoreOS 2023.4.0 (Rhyolite)",
				},
			},
		}

		result, err := subject.IsNodeUptodate(node)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("unknown OS", func(t *testing.T) {
		node := &core_v1.Node{
			Status: core_v1.NodeStatus{
				NodeInfo: core_v1.NodeSystemInfo{
					OSImage: "SLES11SP3",
				},
			},
		}

		result, err := subject.IsNodeUptodate(node)
		assert.Error(t, err)
		assert.False(t, result)
	})

}
