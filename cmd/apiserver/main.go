package main

import (
	goflag "flag"
	"fmt"
	"os"

	kitLog "github.com/go-kit/kit/log"
	"github.com/go-stack/stack"
	"github.com/spf13/pflag"

	apipkg "github.com/sapcc/kubernikus/pkg/api"
	"github.com/sapcc/kubernikus/pkg/api/rest"
	"github.com/sapcc/kubernikus/pkg/api/rest/operations"
	"github.com/sapcc/kubernikus/pkg/api/spec"
	logutil "github.com/sapcc/kubernikus/pkg/util/log"
	"github.com/sapcc/kubernikus/pkg/version"
)

var namespace string

func init() {
	pflag.StringVar(&namespace, "namespace", "kubernikus", "Namespace the apiserver should work in")
}

func main() {
	var logger kitLog.Logger
	logger = kitLog.NewLogfmtLogger(kitLog.NewSyncWriter(os.Stderr))
	logger = logutil.NewTrailingNilFilter(logger)
	logger = kitLog.With(logger, "ts", kitLog.DefaultTimestampUTC, "caller", Caller(3))

	swaggerSpec, err := spec.Spec()
	if err != nil {
		logger.Log(
			"msg", "failed to spec swagger spec",
			"err", err)
		os.Exit(1)
	}

	var server *rest.Server // make sure init is called

	pflag.Usage = func() {
		fmt.Fprint(os.Stderr, "Usage:\n")
		fmt.Fprint(os.Stderr, "  kubernikus-apiserver [OPTIONS]\n\n")

		title := "Kubernikus"
		fmt.Fprint(os.Stderr, title+"\n\n")
		desc := swaggerSpec.Spec().Info.Description
		if desc != "" {
			fmt.Fprintf(os.Stderr, desc+"\n\n")
		}
		fmt.Fprintln(os.Stderr, pflag.CommandLine.FlagUsages())
	}
	// parse the CLI flags
	if f := goflag.Lookup("logtostderr"); f != nil {
		f.Value.Set("true") // log to stderr by default
	}
	pflag.CommandLine.AddGoFlagSet(goflag.CommandLine) //slurp in glog flags
	pflag.Parse()
	//goflag.CommandLine.Parse([]string{}) //https://github.com/kubernetes/kubernetes/issues/17162

	api := operations.NewKubernikusAPI(swaggerSpec)

	rt := &apipkg.Runtime{
		Namespace: namespace,
		Logger:    logger,
	}
	rt.Kubernikus, rt.Kubernetes, err = rest.NewKubeClients(logger)
	if err != nil {
		logger.Log(
			"msg", "failed to create kubernetes clients",
			"err", err)
		os.Exit(1)
	}

	if err := rest.Configure(api, rt); err != nil {
		logger.Log(
			"msg", "failed to configure API server",
			"err", err)
		os.Exit(1)
	}
	logger.Log(
		"msg", "starting Kubernikus API",
		"namespace", namespace,
		"version", version.GitCommit)

	// get server with flag values filled out
	server = rest.NewServer(api)

	defer server.Shutdown()

	server.ConfigureAPI()
	if err := server.Serve(); err != nil {
		logger.Log(
			"msg", "failed to start API server",
			"err", err)
		os.Exit(1)
	}

}

func Caller(depth int) kitLog.Valuer {
	return func() interface{} { return fmt.Sprintf("%+v", stack.Caller(depth)) }
}
