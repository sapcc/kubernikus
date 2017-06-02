package rest

import (
	"crypto/tls"
	"net/http"

	errors "github.com/go-openapi/errors"
	runtime "github.com/go-openapi/runtime"
	middleware "github.com/go-openapi/runtime/middleware"
	graceful "github.com/tylerb/graceful"

	"github.com/sapcc/kubernikus/pkg/api/rest/operations"
)

// This file is safe to edit. Once it exists it will not be overwritten

//go:generate swagger generate server --target ../pkg/api --name kubernikus --spec ../swagger.yml --server-package rest --default-scheme https

func configureFlags(api *operations.KubernikusAPI) {
	// api.CommandLineOptionsGroups = []swag.CommandLineOptionsGroup{ ... }
}

func configureAPI(api *operations.KubernikusAPI) http.Handler {
	// configure the api here
	api.ServeError = errors.ServeError

	// Set your custom logger if needed. Default one is log.Printf
	// Expected interface func(string, ...interface{})
	//
	// Example:
	// s.api.Logger = log.Printf

	api.JSONConsumer = runtime.JSONConsumer()

	api.JSONProducer = runtime.JSONProducer()

	// Applies when the "x-auth-token" header is set
	api.KeystoneAuth = func(token string) (interface{}, error) {
		return nil, errors.NotImplemented("api key auth (keystone) x-auth-token from header param [x-auth-token] has not yet been implemented")
	}

	api.GetAPIHandler = operations.GetAPIHandlerFunc(func(params operations.GetAPIParams) middleware.Responder {
		return middleware.NotImplemented("operation .GetAPI has not yet been implemented")
	})
	api.GetAPIV1ClustersHandler = operations.GetAPIV1ClustersHandlerFunc(func(params operations.GetAPIV1ClustersParams, principal interface{}) middleware.Responder {
		return middleware.NotImplemented("operation .GetAPIV1Clusters has not yet been implemented")
	})
	api.GetAPIV1ClustersNameHandler = operations.GetAPIV1ClustersNameHandlerFunc(func(params operations.GetAPIV1ClustersNameParams, principal interface{}) middleware.Responder {
		return middleware.NotImplemented("operation .GetAPIV1ClustersName has not yet been implemented")
	})
	api.PostAPIV1ClustersHandler = operations.PostAPIV1ClustersHandlerFunc(func(params operations.PostAPIV1ClustersParams, principal interface{}) middleware.Responder {
		return middleware.NotImplemented("operation .PostAPIV1Clusters has not yet been implemented")
	})

	api.ServerShutdown = func() {}

	return setupGlobalMiddleware(api.Serve(setupMiddlewares))
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

// The middleware configuration is for the handler executors. These do not apply to the swagger.json document.
// The middleware executes after routing but before authentication, binding and validation
func setupMiddlewares(handler http.Handler) http.Handler {
	return handler
}

// The middleware configuration happens before anything, this middleware also applies to serving the swagger.json document.
// So this is a good place to plug in a panic handling middleware, logging and metrics
func setupGlobalMiddleware(handler http.Handler) http.Handler {
	return handler
}
