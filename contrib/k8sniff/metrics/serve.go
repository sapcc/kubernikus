package metrics

import (
	"log"
	"net/http"

	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Serve the prometheus metrics endpoint
func Serve(addr, path string) {
	glog.V(1).Infof("Metrics exposed on %s%s", addr, path)

	http.Handle(path, promhttp.Handler())
	log.Fatal(http.ListenAndServe(addr, nil))
}
