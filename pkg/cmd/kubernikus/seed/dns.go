package seed

import (
	"errors"

	"github.com/go-kit/kit/log"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/sapcc/kubernikus/pkg/client/kubernetes"
	"github.com/sapcc/kubernikus/pkg/cmd"
	"github.com/sapcc/kubernikus/pkg/controller/ground/bootstrap/dns"
)

func NewKubeDNSCommand(logger log.Logger) *cobra.Command {
	o := NewKubeDNSOptions(logger)

	c := &cobra.Command{
		Use:   "dns",
		Short: "Seeds the kube-dns addon",
		Run: func(c *cobra.Command, args []string) {
			cmd.CheckError(o.Validate(c, args))
			cmd.CheckError(o.Complete(args))
			cmd.CheckError(o.Run(c))
		},
	}

	o.BindFlags(c.Flags())

	return c
}

type KubeDNSOptions struct {
	kubeConfig string
	context    string
	repository string
	version    string
	domain     string
	clusterIP  string

	Logger log.Logger
}

func NewKubeDNSOptions(logger log.Logger) *KubeDNSOptions {
	return &KubeDNSOptions{
		repository: dns.DEFAULT_REPOSITORY,
		version:    dns.DEFAULT_VERSION,
		domain:     dns.DEFAULT_DOMAIN,
		Logger:     logger,
	}
}

func (o *KubeDNSOptions) BindFlags(flags *pflag.FlagSet) {
	flags.StringVar(&o.kubeConfig, "kubeconfig", o.kubeConfig, "Path to kubeconfig file with authorization information")
	flags.StringVar(&o.context, "context", o.context, "Overwrite the current-context in kubeconfig")
	flags.StringVar(&o.repository, "repository", o.repository, "Docker repository for kube-dns containers")
	flags.StringVar(&o.version, "version", o.version, "Version tag for kube-dns containers")
	flags.StringVar(&o.domain, "domain", o.domain, "Cluster Domain")
	flags.StringVar(&o.clusterIP, "cluster-ip", o.clusterIP, "ClusterIP for kube-dns service")
}

func (o *KubeDNSOptions) Validate(c *cobra.Command, args []string) error {
	if o.clusterIP == "" {
		return errors.New("--cluster-ip is required")
	}
	return nil
}

func (o *KubeDNSOptions) Complete(args []string) error {
	return nil
}

func (o *KubeDNSOptions) Run(c *cobra.Command) error {
	client, err := kubernetes.NewClient(o.kubeConfig, o.context, o.Logger)
	if err != nil {
		return err
	}

	if err = dns.SeedKubeDNS(client, o.repository, o.version, o.domain, o.clusterIP); err != nil {
		return err
	}

	return nil
}
