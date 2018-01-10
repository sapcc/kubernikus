package nanny

import (
	"flag"
	"net"
	"time"

	"github.com/gophercloud/gophercloud/openstack/identity/v3/tokens"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/sapcc/kubernikus/pkg/cmd"
	"github.com/sapcc/kubernikus/pkg/controller/routegc"
	logutil "github.com/sapcc/kubernikus/pkg/util/log"
)

func NewCommand(name string) *cobra.Command {
	o := NewNannyOptions()
	c := &cobra.Command{
		Use:   name,
		Short: "Takes care of changing kubernetes diapers",
		Long:  `A sidecar for cleaning up stuff that gets left behind by kubernetes`,
		RunE: func(c *cobra.Command, args []string) error {
			if err := cmd.Validate(o, c, args); err != nil {
				return err
			}
			return o.Run(c)
		},
	}
	o.BindFlags(c.Flags())

	c.PersistentFlags().AddGoFlagSet(flag.CommandLine)

	return c
}

func NewNannyOptions() *Options {
	return &Options{SyncPeriod: 1 * time.Minute}
}

type Options struct {
	AuthURL           string        `env:"OS_AUTH_URL" valid:"url,required"`
	AuthUsername      string        `env:"OS_USERNAME" valid:"required"`
	AuthPassword      string        `env:"OS_PASSWORD" valid:"required"`
	AuthDomain        string        `env:"OS_USER_DOMAIN_NAME" valid:"required"`
	AuthProject       string        `env:"OS_PROJECT_NAME"`
	AuthProjectDomain string        `env:"OS_PROJECT_DOMAIN_NAME"`
	AuthProjectID     string        `env:"OS_PROJECT_ID"`
	RouterID          string        `env:"ROUTER_ID" valid:"required"`
	ClusterCIDR       string        `env:"CLUSTER_CIDR" valid:"cidr,required"`
	SyncPeriod        time.Duration `env:"SYNC_PERIOD"`
}

func (o *Options) BindFlags(flags *pflag.FlagSet) {
	flags.StringVar(&o.AuthURL, "auth-url", "", "Openstack keystone url")
	flags.StringVar(&o.AuthUsername, "auth-username", "", "Service user for kubernikus")
	flags.StringVar(&o.AuthPassword, "auth-password", "", "Service user password")
	flags.StringVar(&o.AuthDomain, "auth-domain", "", "Service user domain")
	flags.StringVar(&o.AuthProject, "auth-project", "", "Scope service user to this project")
	flags.StringVar(&o.AuthProjectDomain, "auth-project-domain", "", "Domain of the project")
	flags.StringVar(&o.AuthProjectID, "auth-project-id", "", "Domain of the project")
	flags.StringVar(&o.RouterID, "router-id", "", "The OpenStack router used by the kubernetes cluster")
	flags.StringVar(&o.ClusterCIDR, "cluster-cidr", "", "The Pod CIDR used by the kubernetes cluster")
	flags.DurationVar(&o.SyncPeriod, "sync-period", o.SyncPeriod, "How often should the sync handler run.")
}

func (o *Options) Run(c *cobra.Command) error {
	logger := logutil.NewLogger(c.Flags())

	group := cmd.Runner()
	authOpts := tokens.AuthOptions{
		IdentityEndpoint: o.AuthURL,
		Username:         o.AuthUsername,
		DomainName:       o.AuthDomain,
		Password:         o.AuthPassword,
		Scope: tokens.Scope{
			ProjectID:   o.AuthProjectID,
			ProjectName: o.AuthProject,
			DomainName:  o.AuthProjectDomain,
		},
		AllowReauth: true,
	}

	_, cidr, err := net.ParseCIDR(o.ClusterCIDR)
	if err != nil {
		return err //Shouldn't happen as we validate the input before
	}

	routeCleaner := routegc.New(authOpts, o.RouterID, cidr)
	closeCh := make(chan struct{})
	group.Add(
		func() error {
			return routeCleaner.Run(logger, o.SyncPeriod, closeCh)
		},
		func(error) {
			close(closeCh)
		})
	return group.Run()

}
