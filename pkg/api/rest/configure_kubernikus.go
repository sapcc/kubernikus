package rest

import (
	"crypto/tls"
	"net/http"

	graceful "github.com/tylerb/graceful"

	"github.com/sapcc/kubernikus/pkg/api/rest/operations"
)

// This file is safe to edit. Once it exists it will not be overwritten

//go:generate swagger generate server --target ../pkg/api --name kubernikus --spec ../swagger.yml --server-package rest --principal models.Principal --exclude-main

func configureFlags(api *operations.KubernikusAPI) {
	// api.CommandLineOptionsGroups = []swag.CommandLineOptionsGroup{ ... }
}

func configureAPI(api *operations.KubernikusAPI) http.Handler {
	return api.Serve(func(handler http.Handler) http.Handler {
		return handler
	})
}

// The TLS configuration before HTTPS server starts.
func configureTLS(tlsConfig *tls.Config) {
	// Make all necessary changes to the TLS configuration here.
}

// As soon as server is initialized but not run yet, this function will be called.
// If you need to modify a config, store server instance to stop it individually later, this is the place.
// This function can be called multiple times, depending on the number of serving schemes.
// scheme value will be set accordingly: "http", "https" or "unix"
func configureServer(s *graceful.Server, scheme, addr string) {
}
