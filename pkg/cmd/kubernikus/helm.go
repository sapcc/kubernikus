package kubernikus

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"sigs.k8s.io/yaml"

	"github.com/sapcc/kubernikus/pkg/api/models"
	"github.com/sapcc/kubernikus/pkg/apis/kubernikus"
	v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/cmd"
	"github.com/sapcc/kubernikus/pkg/util"
	"github.com/sapcc/kubernikus/pkg/util/helm"
	"github.com/sapcc/kubernikus/pkg/version"
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
	ImagesFile        string
	Region            string
}

func NewHelmOptions() *HelmOptions {
	return &HelmOptions{
		AuthUsername:      os.Getenv("USER"),
		AuthDomain:        "ccadmin",
		AuthProject:       "cloud_admin",
		AuthProjectDomain: "ccadmin",
		ImagesFile:        "charts/images.yaml",
		Region:            "eu-de-1",
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
	flags.StringVar(&o.ImagesFile, "images-file", o.ImagesFile, "Yaml file for populating the image registry")
	flags.StringVar(&o.Region, "region", o.Region, "The region (used for image replacements)")
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

	return nil
}

func (o *HelmOptions) Complete(args []string) error {
	o.Name = args[0]
	return nil
}

func (o *HelmOptions) Run(c *cobra.Command) error {
	nameA := strings.SplitN(o.Name, ".", 2)
	registry, err := version.NewImageRegistry(o.ImagesFile, o.Region)
	if err != nil {
		return fmt.Errorf("Failed to load images from file: %s", err)
	}

	kluster, err := kubernikus.NewKlusterFactory().KlusterFor(models.KlusterSpec{
		Name: nameA[0],
	})
	if err != nil {
		return err
	}

	var secret v1.Secret
	certFactory := util.NewCertificateFactory(kluster, &secret.Certificates, nameA[1])
	if _, err := certFactory.Ensure(); err != nil {
		return err
	}

	secret.Openstack.AuthURL = o.AuthURL
	secret.Openstack.Username = o.AuthUsername
	secret.Openstack.Password = o.AuthPassword
	secret.Openstack.DomainName = o.AuthDomain
	secret.Openstack.ProjectID = o.ProjectID
	secret.BootstrapToken = util.GenerateBootstrapToken()

	values, err := helm.KlusterToHelmValues(kluster, &secret, kluster.Spec.Version, registry, "")
	if err != nil {
		return err
	}

	result, err := yaml.Marshal(&values)
	if err != nil {
		return err
	}
	fmt.Println(string(result))

	return nil
}
