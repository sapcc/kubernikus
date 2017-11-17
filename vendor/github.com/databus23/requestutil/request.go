//Package requestutil provides some helper function for extracting scheme, host
//and port information from http requests.
package requestutil

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

//Scheme returns the protocol (http or https) used by the requesting client
func Scheme(req *http.Request) string {

	if req.Header.Get("X-Forwarded-Ssl") == "on" {
		return "https"
	}
	if len(req.Header["X-Forwarded-Scheme"]) > 0 {
		return req.Header.Get("X-Forwarded-Scheme")
	}
	if len(req.Header["X-Forwarded-Proto"]) > 0 {
		return strings.Split(req.Header.Get("X-Forwarded-Proto"), ",")[0]
	}
	if req.TLS != nil {
		return "https"
	}
	return "http"

}

//HostWithPort returns the HTTP Host: header used by the requesting client
func HostWithPort(req *http.Request) string {
	if len(req.Header["X-Forwarded-Host"]) > 0 {
		forwardedHosts := regexp.MustCompile(`,\s?`).Split(req.Header["X-Forwarded-Host"][0], -1)
		return forwardedHosts[len(forwardedHosts)-1]
	}
	return req.Host
}

//Host returns just the host part of HostWithPort
func Host(req *http.Request) string {
	return strings.Split(HostWithPort(req), ":")[0]
}

//Port returns the port used by the originating client
func Port(req *http.Request) int {
	if parts := strings.Split(HostWithPort(req), ":"); len(parts) > 1 {
		port, _ := strconv.Atoi(parts[1])
		return port
	}
	if len(req.Header["X-Forwarded-Port"]) > 0 {
		port, _ := strconv.Atoi(req.Header["X-Forwarded-Port"][0])
		return port
	}
	return defaultPorts(Scheme(req))
}

func defaultPorts(scheme string) int {
	if scheme == "https" {
		return 443
	}
	return 80
}
