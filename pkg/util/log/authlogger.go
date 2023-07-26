package log

import (
	"fmt"

	kitlog "github.com/go-kit/log"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/tokens"
)

func NewAuthLogger(logger kitlog.Logger, authOptions *tokens.AuthOptions) kitlog.Logger {
	if project := getProject(authOptions); project != "" {
		logger = kitlog.With(logger, "project", project)
	}

	if authMethod := getAuthMethod(authOptions); authMethod != "" {
		logger = kitlog.With(logger, "auth", authMethod)
	}

	if principal := getPrincipal(authOptions); principal != "" {
		logger = kitlog.With(logger, "principal", principal)
	}
	return logger
}

func getProject(authOptions *tokens.AuthOptions) string {
	if authOptions.Scope.ProjectID != "" {
		return authOptions.Scope.ProjectID
	}

	domain := authOptions.Scope.DomainName
	if authOptions.Scope.DomainID != "" {
		domain = authOptions.Scope.DomainID
	}

	return fmt.Sprintf("%s/%s", domain, authOptions.Scope.ProjectName)
}

func getAuthMethod(authOptions *tokens.AuthOptions) string {
	if authOptions.TokenID != "" {
		return "token"
	}

	if authOptions.Password != "" {
		return "password"
	}

	return ""
}

func getPrincipal(authOptions *tokens.AuthOptions) string {
	if authOptions.TokenID != "" {
		return ""
	}

	if authOptions.UserID != "" {
		return authOptions.UserID
	}

	domain := authOptions.DomainName
	if authOptions.DomainID != "" {
		domain = authOptions.DomainID
	}

	return fmt.Sprintf("%s/%s", domain, authOptions.Username)
}
