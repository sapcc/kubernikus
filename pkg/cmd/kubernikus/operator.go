package kubernikus

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/sapcc/kubernikus/pkg/cmd"
	"github.com/sapcc/kubernikus/pkg/controller"
	"github.com/sapcc/kubernikus/pkg/controller/metrics"
	logutil "github.com/sapcc/kubernikus/pkg/util/log"
)

func NewOperatorCommand() *cobra.Command {
	o := NewOperatorOptions()

	c := &cobra.Command{
		Use:   "operator",
		Short: "Starts an operator that operates things. Beware of magic!",
		Run: func(c *cobra.Command, args []string) {
			cmd.CheckError(o.Validate(c, args))
			cmd.CheckError(o.Run(c))
		},
	}

	o.BindFlags(c.Flags())

	return c
}

type Options struct {
	controller.KubernikusOperatorOptions
}

func NewOperatorOptions() *Options {
	options := &Options{}
	options.ChartDirectory = "charts/"
	options.AuthURL = "http://keystone.monsoon3:5000/v3"
	options.AuthUsername = "kubernikus"
	options.AuthDomain = "Default"
	options.KubernikusDomain = "kluster.staging.cloud.sap"
	options.Namespace = "kubernikus"
	options.MetricPort = 9091
	options.Controllers = []string{"groundctl", "launchctl", "deorbiter", "routegc", "flight", "migration", "hammertime", "servicing", "certs"}
	options.Region = "eu-de-1"
	return options
}

func (o *Options) BindFlags(flags *pflag.FlagSet) {
	flags.StringVar(&o.KubeConfig, "kubeconfig", o.KubeConfig, "Path to the kubeconfig file to use to talk to the Kubernetes apiserver. If unset, try the environment variable KUBECONFIG, as well as in-cluster configuration")
	flags.StringVar(&o.Context, "context", "", "Override context")
	flags.StringVar(&o.ChartDirectory, "chart-directory", o.ChartDirectory, "Directory containing the kubernikus related charts")
	flags.StringVar(&o.Region, "region", o.Region, "Local region. (used for container image localization)")
	flags.StringVar(&o.AuthURL, "auth-url", o.AuthURL, "Openstack keystone url")
	flags.StringVar(&o.AuthUsername, "auth-username", o.AuthUsername, "Service user for kubernikus")
	flags.StringVar(&o.AuthPassword, "auth-password", "", "Service user password (if unset its read from env var OS_PASSWORD)")
	flags.StringVar(&o.AuthDomain, "auth-domain", o.AuthDomain, "Service user domain")
	flags.StringVar(&o.AuthProject, "auth-project", o.AuthProject, "Scope service user to this project")
	flags.StringVar(&o.AuthProjectDomain, "auth-project-domain", o.AuthProjectDomain, "Domain of the project")

	flags.StringVar(&o.KubernikusDomain, "kubernikus-domain", o.KubernikusDomain, "Regional domain name for all Kubernikus clusters")
	flags.StringVar(&o.KubernikusProjectID, "kubernikus-projectid", o.KubernikusProjectID, "ID of the project the k*s control plane.")
	flags.StringVar(&o.KubernikusNetworkID, "kubernikus-networkid", o.KubernikusNetworkID, "ID of the network the k*s control plane.")
	flags.StringVar(&o.Namespace, "namespace", o.Namespace, "Restrict operator to resources in the given namespace")
	flags.IntVar(&o.MetricPort, "metric-port", o.MetricPort, "Port on which metrics are exposed")
	flags.StringSliceVar(&o.Controllers, "controllers", o.Controllers, fmt.Sprintf("A list of controllers to enable.  Default is to enable all. controllers: %s", strings.Join(o.Controllers, ", ")))
	flags.IntVar(&o.LogLevel, "v", 0, "log level")
}

func (o *Options) Validate(c *cobra.Command, args []string) error {

	if o.AuthPassword == "" {
		o.AuthPassword = os.Getenv("OS_PASSWORD")
	}

	if o.AuthPassword == "" {
		return errors.New("you must specify the auth-password flag")
	}

	return nil
}

func (o *Options) Run(c *cobra.Command) error {

	logger := logutil.NewLogger(o.LogLevel)

	sigs := make(chan os.Signal, 1)
	stop := make(chan struct{})
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM) // Push signals into channel
	wg := &sync.WaitGroup{}                            // Goroutines can add themselves to this to be waited on

	operator, err := controller.NewKubernikusOperator(&o.KubernikusOperatorOptions, logger)
	if err != nil {
		return err
	}

	go operator.Run(stop, wg)
	go metrics.ExposeMetrics("0.0.0.0", o.MetricPort, stop, wg, logger)
	go func() {
		host := "127.0.0.1:7353"
		ln, err := net.Listen("tcp", "127.0.0.1:7353")
		if err != nil {
			logger.Log("msg", "failed to create pprof listener", "err", err)
			return
		}
		logger.Log("msg", "starting pprof listener", "addr", host)
		go http.Serve(ln, http.DefaultServeMux)
		<-stop
		ln.Close()
	}()

	<-sigs // Wait for signals (this hangs until a signal arrives)
	logger.Log("msg", "shutting down", "v", 1)
	close(stop) // Tell goroutines to stop themselves
	wg.Wait()   // Wait for all to be stopped

	return nil
}
