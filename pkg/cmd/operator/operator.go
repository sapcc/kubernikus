package operator

import (
	"errors"
	goflag "flag"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/golang/glog"
	"github.com/sapcc/kubernikus/pkg/cmd"
	"github.com/sapcc/kubernikus/pkg/controller"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func NewCommand() *cobra.Command {
	o := NewOperatorOptions()

	if f := goflag.Lookup("logtostderr"); f != nil {
		f.Value.Set("true") // log to stderr by default
	}

	c := &cobra.Command{
		Use:   "operator",
		Short: "Starts an operator that operates things. Beware of magic!",
		Run: func(c *cobra.Command, args []string) {
			cmd.CheckError(o.Validate(c, args))
			cmd.CheckError(o.Complete(args))
			cmd.CheckError(o.Run(c))
		},
	}

	o.BindFlags(c.Flags())

	return c
}

type Options struct {
	KubeConfig string

	ChartDirectory string

	AuthURL           string
	AuthUsername      string
	AuthPassword      string
	AuthDomain        string
	AuthProject       string
	AuthProjectDomain string

	KubernikusDomain    string
	KubernikusProjectID string
	KubernikusNetworkID string

	Namespace   string
	Controllers []string
}

func NewOperatorOptions() *Options {
	return &Options{
		ChartDirectory:   "charts/",
		AuthURL:          "http://keystone.monsoon3:5000/v3",
		AuthUsername:     "kubernikus",
		AuthDomain:       "Default",
		KubernikusDomain: "kluster.staging.cloud.sap",
		Namespace:        "kubernikus",
		Controllers:      []string{"groundctl", "launchctl"},
	}
}

func (o *Options) BindFlags(flags *pflag.FlagSet) {
	flags.StringVar(&o.KubeConfig, "kubeconfig", o.KubeConfig, "Path to the kubeconfig file to use to talk to the Kubernetes apiserver. If unset, try the environment variable KUBECONFIG, as well as in-cluster configuration")
	flags.StringVar(&o.ChartDirectory, "chart-directory", o.ChartDirectory, "Directory containing the kubernikus related charts")
	flags.StringVar(&o.AuthURL, "auth-url", o.AuthURL, "Openstack keystone url")
	flags.StringVar(&o.AuthUsername, "auth-username", o.AuthUsername, "Service user for kubernikus")
	flags.StringVar(&o.AuthPassword, "auth-password", o.AuthPassword, "Service user password")
	flags.StringVar(&o.AuthDomain, "auth-domain", o.AuthDomain, "Service user domain")
	flags.StringVar(&o.AuthProject, "auth-project", o.AuthProject, "Scope service user to this project")
	flags.StringVar(&o.AuthProjectDomain, "auth-project-domain", o.AuthProjectDomain, "Domain of the project")

	flags.StringVar(&o.KubernikusDomain, "kubernikus-domain", o.KubernikusDomain, "Regional domain name for all Kubernikus clusters")
	flags.StringVar(&o.KubernikusProjectID, "kubernikus-projectid", o.KubernikusProjectID, "ID of the project the k*s control plane.")
	flags.StringVar(&o.KubernikusNetworkID, "kubernikus-networkid", o.KubernikusNetworkID, "ID of the network the k*s control plane.")
	flags.StringVar(&o.Namespace, "namespace", o.Namespace, "Restrict operator to resources in the given namespace")
	flags.StringSliceVar(&o.Controllers, "controllers", o.Controllers, "A list of controllers to enable.  Default is to enable all. controllers: groundctl, launchctl")
}

func (o *Options) Validate(c *cobra.Command, args []string) error {
	if len(o.AuthPassword) == 0 {
		return errors.New("you must specify the auth-password flag")
	}

	return nil
}

func (o *Options) Complete(args []string) error {
	return nil
}

func (o *Options) Run(c *cobra.Command) error {
	sigs := make(chan os.Signal, 1)
	stop := make(chan struct{})
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM) // Push signals into channel
	wg := &sync.WaitGroup{}                            // Goroutines can add themselves to this to be waited on

	opts := &controller.KubernikusOperatorOptions{
		KubeConfig:          o.KubeConfig,
		ChartDirectory:      o.ChartDirectory,
		AuthURL:             o.AuthURL,
		AuthUsername:        o.AuthUsername,
		AuthPassword:        o.AuthPassword,
		AuthDomain:          o.AuthDomain,
		AuthProject:         o.AuthProject,
		AuthProjectDomain:   o.AuthProjectDomain,
		KubernikusDomain:    o.KubernikusDomain,
		KubernikusProjectID: o.KubernikusProjectID,
		KubernikusNetworkID: o.KubernikusNetworkID,
		Namespace:           o.Namespace,
		Controllers:         o.Controllers,
	}

	go controller.NewKubernikusOperator(opts).Run(stop, wg)

	<-sigs // Wait for signals (this hangs until a signal arrives)
	glog.Info("Shutting down...")
	close(stop) // Tell goroutines to stop themselves
	wg.Wait()   // Wait for all to be stopped

	return nil
}
