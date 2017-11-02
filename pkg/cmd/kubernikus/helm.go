package kubernikus

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/sapcc/kubernikus/pkg/apis/kubernikus"
	"github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/cmd"
	"github.com/sapcc/kubernikus/pkg/util"
	"github.com/sapcc/kubernikus/pkg/util/helm"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func NewHelmCommand() *cobra.Command {
	o := NewHelmOptions()

	c := &cobra.Command{
		Use:   "helm NAME",
		Short: "Print Helm values",
		Run: func(c *cobra.Command, args []string) {
			cmd.CheckError(o.Validate(c, args))
			cmd.CheckError(o.Complete(args))
			cmd.CheckError(o.Run(c))
		},
	}

	o.BindFlags(c.Flags())

	return c
}

type HelmOptions struct {
	Name              string
	AuthURL           string
	AuthUsername      string
	AuthPassword      string
	AuthDomain        string
	AuthProject       string
	AuthProjectDomain string
	ProjectID         string
	KubernikusAPI     string
}

func NewHelmOptions() *HelmOptions {
	return &HelmOptions{
		AuthUsername:      os.Getenv("USER"),
		AuthDomain:        "ccadmin",
		AuthProject:       "cloud_admin",
		AuthProjectDomain: "ccadmin",
	}
}
func (o *HelmOptions) BindFlags(flags *pflag.FlagSet) {
	flags.StringVar(&o.AuthURL, "auth-url", o.AuthURL, "Openstack keystone url")
	flags.StringVar(&o.AuthUsername, "auth-username", o.AuthUsername, "Service user for kubernikus")
	flags.StringVar(&o.AuthPassword, "auth-password", o.AuthPassword, "Service user password [OS_PASSWORD] ")
	flags.StringVar(&o.AuthDomain, "auth-domain", o.AuthDomain, "Service user domain")
	flags.StringVar(&o.AuthProject, "auth-project", o.AuthProject, "Scope service user to this project")
	flags.StringVar(&o.AuthProjectDomain, "auth-project-domain", o.AuthProjectDomain, "Domain of the project")
	flags.StringVar(&o.ProjectID, "project-id", o.ProjectID, "Project ID where the kublets will be running")
	flags.StringVar(&o.KubernikusAPI, "api", o.KubernikusAPI, "Kubernikus API URL. e.g. https://kubernikus.eu-nl-1.cloud.sap")
}

func (o *HelmOptions) Validate(c *cobra.Command, args []string) error {
	if len(args) < 1 {
		return errors.New("you must specify the cluster's name")
	}
	if !strings.Contains(args[0], ".") {
		return errors.New("Name must be the fqdn of the apiserver")
	}
	if o.AuthURL != "" {
		if o.ProjectID == "" {
			return errors.New("project-id is required when specifying an auth-url")
		}
		if o.AuthPassword == "" {
			o.AuthPassword = os.Getenv("OS_PASSWORD")
			if o.AuthPassword == "" {
				return errors.New("password is required")
			}
		}
	}

	if o.KubernikusAPI == "" {
		return errors.New("--api is required")
	}

	return nil
}

func (o *HelmOptions) Complete(args []string) error {
	o.Name = args[0]
	return nil
}

func (o *HelmOptions) Run(c *cobra.Command) error {
	nameA := strings.SplitN(o.Name, ".", 2)
	kluster, err := kubernikus.NewKlusterFactory().KlusterFor(v1.KlusterSpec{
		Name: nameA[0],
		Openstack: v1.OpenstackSpec{
			ProjectID: o.ProjectID,
		},
	})
	if err != nil {
		return err
	}

	options := &helm.OpenstackOptions{
		AuthURL:    o.AuthURL,
		Username:   o.AuthUsername,
		Password:   o.AuthPassword,
		DomainName: o.AuthDomain,
	}

	certificates, err := util.CreateCertificates(kluster, o.KubernikusAPI, o.AuthURL, nameA[1])
	if err != nil {
		return err
	}

	token := util.GenerateBootstrapToken()

	result, err := helm.KlusterToHelmValues(kluster, options, certificates, token, "")
	if err != nil {
		return err
	}

	fmt.Println(string(result))

	return nil
}
