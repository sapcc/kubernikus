package requestutil

import (
	"net/http"
	"testing"
)

func TestScheme(t *testing.T) {

	var cases = []struct {
		headers  map[string]string
		expected string
	}{
		{
			map[string]string{},
			"http",
		},
		{
			map[string]string{"X-Forwarded-Ssl": "on"},
			"https",
		},
		{
			map[string]string{"X-Forwarded-Scheme": "https"},
			"https",
		},
		{
			map[string]string{"X-Forwarded-Proto": "https,urks"},
			"https",
		},
	}
	for i, c := range cases {
		header := http.Header{}
		for key, val := range c.headers {
			header[key] = []string{val}
		}
		req := http.Request{
			Header: header,
		}

		if s := Scheme(&req); s != c.expected {
			t.Errorf("Case %d. Expected scheme %#v, got %#v, headers: %v", i+1, c.expected, s, c.headers)
		}
	}

}

func TestHostWithPort(t *testing.T) {
	var cases = []struct {
		headers  map[string]string
		expected string
	}{
		{
			map[string]string{},
			"localhost",
		},
		{
			map[string]string{"Host": "localhost", "X-Forwarded-Host": "blafasel:1, blufasel,blafasel:443"},
			"blafasel:443",
		},
	}

	for i, c := range cases {
		header := http.Header{}
		for key, val := range c.headers {
			header[key] = []string{val}
		}
		req := http.Request{
			Header: header,
			Host:   "localhost",
		}

		if s := HostWithPort(&req); s != c.expected {
			t.Errorf("Case %d. Expected %#v, got %#v, headers: %v", i+1, c.expected, s, c.headers)
		}
	}

}

func TestPort(t *testing.T) {
	var cases = []struct {
		headers  map[string]string
		expected int
	}{
		{
			map[string]string{},
			80,
		},
		{
			map[string]string{},
			80,
		},
		{
			map[string]string{"X-Forwarded-Ssl": "on"},
			443,
		},
		{
			map[string]string{"X-Forwarded-Host": "blafasel:1, blufasel,blafasel:443"},
			443,
		},
	}

	for i, c := range cases {
		header := http.Header{}
		for key, val := range c.headers {
			header[key] = []string{val}
		}
		req := http.Request{
			Header: header,
		}

		if s := Port(&req); s != c.expected {
			t.Errorf("Case %d. Expected %d, got %d, headers: %v", i+1, c.expected, s, c.headers)
		}
	}

}
