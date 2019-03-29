package coreos

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func NewFakeVersion(t *testing.T, version string) *Version {
	body := fmt.Sprintf("COREOS_VERSION=%s", version)
	subject := &Version{}
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

type RoundTripFunc func(req *http.Request) *http.Response

func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}
