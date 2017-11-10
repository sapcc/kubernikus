package rest

import (
	"fmt"

	"github.com/go-openapi/errors"
	runtime "github.com/go-openapi/runtime"
	"github.com/golang/glog"

	apipkg "github.com/sapcc/kubernikus/pkg/api"
	"github.com/sapcc/kubernikus/pkg/api/handlers"
	"github.com/sapcc/kubernikus/pkg/api/rest/operations"
)

func Configure(api *operations.KubernikusAPI, rt *apipkg.Runtime) {
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

	// Set your custom authorizer if needed. Default one is security.Authorized()
	// api.APIAuthorizer = security.Authorized()

	api.InfoHandler = handlers.NewInfo(rt)
	api.ListAPIVersionsHandler = handlers.NewListAPIVersions(rt)
	api.ListClustersHandler = handlers.NewListClusters(rt)
	api.CreateClusterHandler = handlers.NewCreateCluster(rt)
	api.ShowClusterHandler = handlers.NewShowCluster(rt)
	api.TerminateClusterHandler = handlers.NewTerminateCluster(rt)
	api.UpdateClusterHandler = handlers.NewUpdateCluster(rt)
	api.GetClusterCredentialsHandler = handlers.NewGetClusterCredentials(rt)
	api.GetClusterInfoHandler = handlers.NewGetClusterInfo(rt)

	api.ServerShutdown = func() {}

}
