package main

import (
	goflag "flag"
	"fmt"
	"log"
	"os"

	"github.com/golang/glog"
	"github.com/spf13/pflag"

	apipkg "github.com/sapcc/kubernikus/pkg/api"
	"github.com/sapcc/kubernikus/pkg/api/rest"
	"github.com/sapcc/kubernikus/pkg/api/rest/operations"
	"github.com/sapcc/kubernikus/pkg/api/spec"
	"github.com/sapcc/kubernikus/pkg/version"
)

var namespace string

func init() {
	pflag.StringVar(&namespace, "namespace", "kubernikus", "Namespace the apiserver should work in")
}

func main() {

	swaggerSpec, err := spec.Spec()
	if err != nil {
		log.Fatalln(err)
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

	rt := &apipkg.Runtime{Namespace: namespace}
	rt.Kubernikus, rt.Kubernetes = rest.NewKubeClients()
	if err := rest.Configure(api, rt); err != nil {
		glog.Fatalf("Failed to configure apiserver :%s", err)
	}
	glog.Infof("Starting kubernikus apiserver v%v. Using namespace %s", version.GitCommit, namespace)

	// get server with flag values filled out
	server = rest.NewServer(api)

	defer server.Shutdown()

	server.ConfigureAPI()
	if err := server.Serve(); err != nil {
		log.Fatalln(err)
	}

}
