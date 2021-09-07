package common

import "github.com/spf13/pflag"

type OIDCClient struct {
	Token string
}

func (o *OIDCClient) BindOIDCFlags(flags *pflag.FlagSet) {
	flags.StringVar(&o.Token, "oidc-token", "", "Preauthed OIDC token")
}

func NewOIDCClient() *OIDCClient {
	return &OIDCClient{}
}