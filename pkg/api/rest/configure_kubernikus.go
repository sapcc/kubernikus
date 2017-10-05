package rest

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"

	errors "github.com/go-openapi/errors"
	runtime "github.com/go-openapi/runtime"
	middleware "github.com/go-openapi/runtime/middleware"
	"github.com/golang/glog"
	gmiddleware "github.com/gorilla/handlers"
	"github.com/rs/cors"
	graceful "github.com/tylerb/graceful"

	apipkg "github.com/sapcc/kubernikus/pkg/api"
	"github.com/sapcc/kubernikus/pkg/api/handlers"
	"github.com/sapcc/kubernikus/pkg/api/rest/operations"
)

// This file is safe to edit. Once it exists it will not be overwritten

//go:generate swagger generate server --target ../pkg/api --name kubernikus --spec ../swagger.yml --server-package rest --principal models.Principal --exclude-main

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
	api.Logger = func(msg string, args ...interface{}) {
		glog.InfoDepth(2, fmt.Sprintf(msg, args...))
	}

	api.JSONConsumer = runtime.JSONConsumer()

	api.JSONProducer = runtime.JSONProducer()

	// Applies when the "x-auth-token" header is set
	api.KeystoneAuth = keystoneAuth()

	rt := &apipkg.Runtime{Namespace: namespace}
	rt.Kubernikus, rt.Kubernetes = NewKubeClients()

	api.ListAPIVersionsHandler = handlers.NewListAPIVersions(rt)
	api.ListClustersHandler = handlers.NewListClusters(rt)
	api.CreateClusterHandler = handlers.NewCreateCluster(rt)
	api.ShowClusterHandler = handlers.NewShowCluster(rt)
	api.TerminateClusterHandler = handlers.NewTerminateCluster(rt)
	api.UpdateClusterHandler = handlers.NewUpdateCluster(rt)
	api.GetClusterCredentialsHandler = handlers.NewGetClusterCredentials(rt)

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
	return middleware.Redoc(middleware.RedocOpts{}, handler)
}

// The middleware configuration happens before anything, this middleware also applies to serving the swagger.json document.
// So this is a good place to plug in a panic handling middleware, logging and metrics
func setupGlobalMiddleware(handler http.Handler) http.Handler {
	c := cors.New(cors.Options{
		AllowedHeaders: []string{"X-Auth-Token", "Content-Type", "Accept"},
		AllowedMethods: []string{"GET", "HEAD", "POST", "DELETE", "PUT"},
		MaxAge: 600,
	})
	return gmiddleware.LoggingHandler(os.Stdout, handlers.RootHandler(c.Handler(handler)))
}
