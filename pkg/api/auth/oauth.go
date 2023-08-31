package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/aokoli/goutils"
	oidc "github.com/coreos/go-oidc/v3/oidc"
	"github.com/go-kit/log"
	errors "github.com/go-openapi/errors"
	runtime "github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/middleware"
	flag "github.com/spf13/pflag"
	"golang.org/x/oauth2"

	"github.com/sapcc/kubernikus/pkg/api/models"
	"github.com/sapcc/kubernikus/pkg/api/rest/operations"
)

var (
	issuerURL    string
	clientID     string
	clientSecret string
	callbackURL  string
)

func init() {
	flag.StringVar(&issuerURL, "oidc-issuer-url", "", "")
	flag.StringVar(&clientID, "oidc-client-id", "", "")
	flag.StringVar(&clientSecret, "oidc-client-secret", "", "")
	flag.StringVar(&callbackURL, "oidc-callback-url", "", "")
}

func OAuthEnabled() bool {
	return issuerURL != "" && clientID != "" && clientSecret != "" && callbackURL != ""
}

func OAuthConfig(logger log.Logger) (func(token string, scopes []string) (*models.Principal, error), operations.GetAuthLoginHandler, operations.GetAuthCallbackHandler, error) {
	//func OAuthConfig() (*oauth2.Config, *oidc.IDTokenVerifier, error) {

	provider, err := oidc.NewProvider(context.Background(), issuerURL)
	if err != nil {
		return nil, nil, nil, err
	}

	config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint:     provider.Endpoint(),
		RedirectURL:  callbackURL,
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email", "groups"},
	}
	randomPasswordChars := []rune("abcdefghjkmnpqrstuvwxABCDEFGHJKLMNPQRSTUVWX23456789")
	state, err := goutils.Random(12, 0, 0, true, true, randomPasswordChars...)
	if err != nil {
		return nil, nil, nil, err
	}

	verifier := provider.Verifier(&oidc.Config{ClientID: config.ClientID})
	return OAuth(verifier, logger), NewAuthLogin(config, state), NewAuthCallback(config, state), nil
}

func OAuth(verifier *oidc.IDTokenVerifier, logger log.Logger) func(token string, scopes []string) (*models.Principal, error) {
	return func(token string, scopes []string) (*models.Principal, error) {
		id, err := verifier.Verify(context.Background(), token)
		if err != nil {
			return nil, errors.New(401, "invalid token: %s", err)
		}
		var claims struct {
			Name   string
			Email  string
			Groups []string
		}
		if err := id.Claims(&claims); err != nil {
			return nil, errors.New(401, "invalid token 2: %s", err)
		}
		emailParts := strings.Split(claims.Email, "@")
		if len(emailParts) < 2 {
			return nil, errors.New(401, "Malformed email in idToken")
		}
		prin := models.Principal{ID: id.Subject, Name: claims.Name, Account: emailParts[1], Groups: claims.Groups}
		return &prin, nil
	}
}

type authLogin struct {
	config *oauth2.Config
	state  string
}

func NewAuthLogin(c *oauth2.Config, state string) operations.GetAuthLoginHandler {
	return &authLogin{config: c, state: state}
}

func (a *authLogin) Handle(params operations.GetAuthLoginParams) middleware.Responder {
	opts := []oauth2.AuthCodeOption{}
	if params.ConnectorID != nil && len(*params.ConnectorID) > 0 {
		opts = append(opts, oauth2.SetAuthURLParam("connector_id", *params.ConnectorID))
	}

	// implements the login with a redirection
	return middleware.ResponderFunc(
		func(w http.ResponseWriter, pr runtime.Producer) {
			http.Redirect(w, params.HTTPRequest, a.config.AuthCodeURL(a.state, opts...), http.StatusFound)
		})
}

type authCallback struct {
	config *oauth2.Config
	state  string
}

func NewAuthCallback(c *oauth2.Config, state string) operations.GetAuthCallbackHandler {
	return &authCallback{config: c, state: state}
}

func (a *authCallback) Handle(params operations.GetAuthCallbackParams) middleware.Responder {

	token, err := a.verify(params.Code, params.State)
	if err != nil {
		return operations.NewGetAuthCallbackDefault(400).WithPayload(&models.Error{Code: 400, Message: err.Error()})
	}
	return operations.NewGetAuthCallbackOK().WithPayload(&models.GetAuthCallbackOKBody{IDToken: token, Type: "Bearer"})

}

func (a *authCallback) verify(authCode, state string) (string, error) {
	// we expect the redirected client to call us back
	// with 2 query params: state and code.
	// We use directly the Request params here, since we did not
	// bother to document these parameters in the spec.

	if state != a.state {
		return "", fmt.Errorf("state did not match")
	}

	//ctx := oidc.ClientContext(context.Background(), http.DefaultClient)

	// Exchange converts an authorization code into a token.
	// Under the hood, the oauth2 client POST a request to do so
	// at tokenURL, then redirects...
	oauth2Token, err := a.config.Exchange(context.Background(), authCode)
	if err != nil {
		return "", fmt.Errorf("failed to exchange token: %w", err)
	}
	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		return "", fmt.Errorf("No id token in response")
	}

	return rawIDToken, nil
}
