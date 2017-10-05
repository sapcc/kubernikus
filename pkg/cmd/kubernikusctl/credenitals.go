package kubernikusctl

import (
	"fmt"
	"os"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/tokens"
	"github.com/sapcc/kubernikus/pkg/cmd"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func NewCredentialsCommand() *cobra.Command {
	o := NewCredentialsOptions()

	c := &cobra.Command{
		Use:   "credentials",
		Short: "Fetches Kubernikus credentials via API",
		Run: func(c *cobra.Command, args []string) {
			cmd.CheckError(o.Validate(c, args))
			cmd.CheckError(o.Complete(args))
			cmd.CheckError(o.Run(c))
		},
	}

	o.BindFlags(c.Flags())

	return c
}

type CredentialsOptions struct {
	Name string
}

func NewCredentialsOptions() *CredentialsOptions {
	return &CredentialsOptions{}
}

func (o *CredentialsOptions) BindFlags(flags *pflag.FlagSet) {
}

func (o *CredentialsOptions) Validate(c *cobra.Command, args []string) error {
	return nil
}

func (o *CredentialsOptions) Complete(args []string) error {
	return nil
}

func (o *CredentialsOptions) Run(c *cobra.Command) error {
	token, err := getToken()
	if err != nil {
		return err
	}

	fmt.Println(token)
	return nil
}

func getToken() (string, error) {
	authOptions := authOptionsFromENV()

	provider, err := openstack.NewClient(authOptions.IdentityEndpoint)
	if err != nil {
		return "", err
	}

	err = openstack.AuthenticateV3(provider, authOptions, gophercloud.EndpointOpts{})
	if err != nil {
		return "", err
	}

	return provider.TokenID, nil
}

func authOptionsFromENV() *tokens.AuthOptions {
	return &tokens.AuthOptions{
		IdentityEndpoint: os.Getenv("OS_AUTH_URL"),
		Username:         os.Getenv("OS_USERNAME"),
		Password:         os.Getenv("OS_PASSWORD"),
		DomainName:       os.Getenv("OS_USER_DOMAIN_NAME"),
		AllowReauth:      true,
		Scope: tokens.Scope{
			ProjectName: os.Getenv("OS_PROJECT_NAME"),
			DomainName:  os.Getenv("OS_PROJECT_DOMAIN_NAME"),
		},
	}
}


