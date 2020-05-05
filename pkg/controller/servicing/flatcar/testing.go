package flatcar

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func NewFakeVersion(t *testing.T, version string) *Version {
	body := fmt.Sprintf("FLATCAR_VERSION=%s", version)
	subject := &Version{}
	subject.Client = NewTestClient(t, "https://stable.release.flatcar-linux.net/amd64-usr/current/version.txt", body, nil)
	return subject
}

func NewFakeRelease(t *testing.T, version string) *Release {
	body := fmt.Sprintf(`
		FLATCAR_BUILD=xxxx
		FLATCAR_BRANCH=x
		FLATCAR_PATCH=x
		FLATCAR_VERSION=%s
		FLATCAR_VERSION_ID=%s
		FLATCAR_BUILD_ID="2020-02-08-0830"
		FLATCAR_SDK_VERSION=%s`, version, version, version)
	subject := &Release{}
	subject.Client = NewTestClient(t, fmt.Sprintf("https://stable.release.flatcar-linux.net/amd64-usr/%s/version.txt", version), body, nil)
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

type RoundTripFunc func(req *http.Request) *http.Response

func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}
