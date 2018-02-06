package rest

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/go-openapi/errors"
	runtime "github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/middleware"
	"github.com/justinas/alice"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/cors"

	apipkg "github.com/sapcc/kubernikus/pkg/api"
	"github.com/sapcc/kubernikus/pkg/api/auth"
	"github.com/sapcc/kubernikus/pkg/api/handlers"
	"github.com/sapcc/kubernikus/pkg/api/rest/operations"
	"github.com/sapcc/kubernikus/pkg/api/spec"
	logutil "github.com/sapcc/kubernikus/pkg/util/log"
)

func Configure(api *operations.KubernikusAPI, rt *apipkg.Runtime) error {
	// configure the api here
	api.ServeError = errors.ServeError

	// Set your custom logger if needed. Default one is log.Printf
	// Expected interface func(string, ...interface{})
	//
	// Example:
	api.Logger = func(msg string, args ...interface{}) {
		rt.Logger.Log("msg", fmt.Sprintf(msg, args...))
	}

	api.JSONConsumer = runtime.JSONConsumer()
	api.JSONProducer = runtime.JSONProducer()

	// Applies when the "x-auth-token" header is set
	api.KeystoneAuth = auth.Keystone(rt.Logger)

	// Set your custom authorizer if needed. Default one is security.Authorized()
	rules, err := auth.LoadPolicy(auth.DefaultPolicyFile)
	if err != nil {
		return err
	}
	document, err := spec.Spec()
	if err != nil {
		return err
	}
	authorizer, err := auth.NewOsloPolicyAuthorizer(document, rules)
	if err != nil {
		return err
	}
	api.APIAuthorizer = authorizer

	api.InfoHandler = handlers.NewInfo(rt)
	api.ListAPIVersionsHandler = handlers.NewListAPIVersions(rt)
	api.ListClustersHandler = handlers.NewListClusters(rt)
	api.CreateClusterHandler = handlers.NewCreateCluster(rt)
	api.ShowClusterHandler = handlers.NewShowCluster(rt)
	api.TerminateClusterHandler = handlers.NewTerminateCluster(rt)
	api.UpdateClusterHandler = handlers.NewUpdateCluster(rt)
	api.GetClusterCredentialsHandler = handlers.NewGetClusterCredentials(rt)
	api.GetClusterInfoHandler = handlers.NewGetClusterInfo(rt)
	api.GetOpenstackMetadataHandler = handlers.NewGetOpenstackMetadata(rt)
	api.GetClusterEventsHandler = handlers.NewGetClusterEvents(rt)

	api.ServerShutdown = func() {}

	api.Middleware = func(builder middleware.Builder) http.Handler {
		return setupGlobalMiddleware(api.Context().APIHandler(builder), rt)
	}
	return nil
}

// The middleware configuration happens before anything, this middleware also applies to serving the swagger.json document.
// So this is a good place to plug in a panic handling middleware, logging and metrics
func setupGlobalMiddleware(handler http.Handler, rt *apipkg.Runtime) http.Handler {
	corsHandler := cors.New(cors.Options{
		AllowedHeaders: []string{"X-Auth-Token", "Content-Type", "Accept"},
		AllowedMethods: []string{"GET", "HEAD", "POST", "DELETE", "PUT"},
		MaxAge:         600,
	}).Handler

	requestIDHandler := func(next http.Handler) http.Handler {
		return logutil.RequestIDHandler(next)
	}

	loggingHandler := func(next http.Handler) http.Handler {
		return logutil.LoggingHandler(rt.Logger, next)
	}

	redocHandler := func(next http.Handler) http.Handler {
		return middleware.Redoc(middleware.RedocOpts{Path: "swagger"}, next)
	}

	staticHandler := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/docs") {
				http.StripPrefix("/docs", http.FileServer(http.Dir("static/docs"))).ServeHTTP(rw, r)
				return
			}
			next.ServeHTTP(rw, r)
		})
	}
	instrumentationHandler := func(next http.Handler) http.Handler {
		return promhttp.InstrumentHandlerCounter(HTTPRequestsTotal, next)
	}

	return alice.New(
		requestIDHandler,
		loggingHandler,
		instrumentationHandler,
		handlers.RootHandler,
		redocHandler,
		staticHandler,
		corsHandler).Then(handler)
}
