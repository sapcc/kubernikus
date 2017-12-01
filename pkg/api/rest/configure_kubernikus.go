package rest

import (
	"crypto/tls"
	"net/http"
	"os"
	"strings"

	"github.com/go-openapi/runtime/middleware"
	gmiddleware "github.com/gorilla/handlers"
	"github.com/justinas/alice"
	"github.com/rs/cors"
	graceful "github.com/tylerb/graceful"

	"github.com/sapcc/kubernikus/pkg/api/handlers"
	"github.com/sapcc/kubernikus/pkg/api/rest/operations"
)

// This file is safe to edit. Once it exists it will not be overwritten

//go:generate swagger generate server --target ../pkg/api --name kubernikus --spec ../swagger.yml --server-package rest --principal models.Principal --exclude-main

func configureFlags(api *operations.KubernikusAPI) {
	// api.CommandLineOptionsGroups = []swag.CommandLineOptionsGroup{ ... }
}

func configureAPI(api *operations.KubernikusAPI) http.Handler {

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
	corsHandler := cors.New(cors.Options{
		AllowedHeaders: []string{"X-Auth-Token", "Content-Type", "Accept"},
		AllowedMethods: []string{"GET", "HEAD", "POST", "DELETE", "PUT"},
		MaxAge:         600,
	}).Handler

	loggingHandler := func(next http.Handler) http.Handler {
		return gmiddleware.LoggingHandler(os.Stdout, next)
	}
	redocHandler := func(next http.Handler) http.Handler {
		return middleware.Redoc(middleware.RedocOpts{Path: "swagger"}, next)
	}

	return alice.New(loggingHandler, handlers.RootHandler, redocHandler, StaticFiles, corsHandler).Then(handler)
}

func StaticFiles(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/docs") {
			http.StripPrefix("/docs", http.FileServer(http.Dir("static/docs"))).ServeHTTP(rw, r)
			return
		}
		next.ServeHTTP(rw, r)
	})
}
