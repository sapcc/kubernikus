package rest

import (
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/databus23/keystone"
	"github.com/databus23/keystone/cache/memory"
	errors "github.com/go-openapi/errors"
	flag "github.com/spf13/pflag"

	"github.com/sapcc/kubernikus/pkg/api/models"
)

var authURL string

func init() {
	flag.StringVar(&authURL, "auth-url", "", "Openstack identity v3 auth url")
}

func keystoneAuth() func(token string) (*models.Principal, error) {

	if !(strings.HasSuffix("/v3") || strings.HasSuffix("/v3/")) {
		authURL = path.Join(authURL, "/v3")
	}

	auth := keystone.New(authURL)
	auth.TokenCache = memory.New(10 * time.Minute)

	return func(token string) (*models.Principal, error) {
		t, err := auth.Validate(token)
		if err != nil {
			return nil, errors.New(401, fmt.Sprintf("Authentication failed: %s", err))
		}
		if t.Project == nil {
			return nil, errors.New(401, "Auth token isn't project scoped")
		}
		roles := make([]string, 0, len(t.Roles))
		for _, role := range t.Roles {
			roles = append(roles, role.Name)
		}
		return &models.Principal{ID: t.User.ID, Name: t.User.Name, Account: t.Project.ID, Roles: roles}, nil
	}
}
