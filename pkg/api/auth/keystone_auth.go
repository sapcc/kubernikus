package auth

import (
	"fmt"
	"strings"
	"time"

	"github.com/databus23/keystone"
	"github.com/databus23/keystone/cache/memory"
	"github.com/go-kit/log"
	errors "github.com/go-openapi/errors"
	flag "github.com/spf13/pflag"

	"github.com/sapcc/kubernikus/pkg/api/models"
)

var authURL string

func init() {
	flag.StringVar(&authURL, "auth-url", "", "Openstack identity v3 auth url")
}

func OpenStackAuthURL() string {
	if authURL == "" {
		return ""
	}
	if !(strings.HasSuffix(authURL, "/v3") || strings.HasSuffix(authURL, "/v3/")) {
		return strings.TrimRight(authURL, "/") + "/v3"
	}
	return authURL
}

func Keystone(logger log.Logger) func(token string) (*models.Principal, error) {

	keystone.Log = func(format string, a ...interface{}) {
		logger.Log("library", "keystone", "msg", fmt.Sprintf(format, a...))
	}
	auth := keystone.New(OpenStackAuthURL())
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
		return &models.Principal{ID: t.User.ID, Name: t.User.Name, Domain: t.User.Domain.Name, Account: t.Project.ID, AccountName: t.Project.Name, Roles: roles}, nil
	}
}
