package framework

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/tokens"
)

type openstackAuthInfo struct {
	token string
}

func NewOpensStackAuth(authOptions *tokens.AuthOptions) (runtime.ClientAuthInfoWriter, error) {
	provider, err := openstack.NewClient(os.Getenv("OS_AUTH_URL"))
	if err != nil {
		return nil, fmt.Errorf("could not initialize openstack client: %v", err)
	}

	if err := openstack.AuthenticateV3(provider, authOptions, gophercloud.EndpointOpts{}); err != nil {
		return nil, fmt.Errorf("could not authenticate with openstack: %v", err)
	}
	return &openstackAuthInfo{token: provider.Token()}, nil
}

func (o openstackAuthInfo) AuthenticateRequest(req runtime.ClientRequest, reg strfmt.Registry) error {
	req.SetHeaderParam("X-Auth-Token", o.token)
	return nil
}

type oidcAuthInfo struct {
	token string
}

func NewOIDCAuth(username, password, connector_id, authUrl string) (runtime.ClientAuthInfoWriter, error) {

	idToken, err := oidcLogin(username, password, connector_id, authUrl)
	if err != nil {
		return nil, fmt.Errorf("oidc logon failed: %w", err)
	}
	return &oidcAuthInfo{token: idToken}, nil
}

func (o oidcAuthInfo) AuthenticateRequest(req runtime.ClientRequest, reg strfmt.Registry) error {
	req.SetHeaderParam("Authorization", "Bearer "+o.token)
	return nil
}

func oidcLogin(username, password, connector_id, authUrl string) (string, error) {

	var redirects = 0
	client := http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			redirects++
			return nil
		},
	}

	req, err := http.NewRequest("GET", authUrl, nil)
	if err != nil {
		return "", fmt.Errorf("Failed to build initial request: %w", err)
	}
	q := url.Values{}
	q.Set("connector_id", connector_id)
	req.URL.RawQuery = q.Encode()

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("Failed to call %s: %w", authUrl, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("Calling %s failed with %s, maybe because of an incorrect connector_id", resp.Request.URL, resp.Status)
	}
	if redirects < 1 {
		return "", fmt.Errorf("Login failed, expected some redirects")
	}

	redirects = 0
	v := url.Values{}
	v.Set("login", username)
	v.Set("password", password)
	//log.Printf("Logging in using %s", resp.Request.URL.String())
	resp2, err := client.PostForm(resp.Request.URL.String(), v)
	if err != nil {
		return "", fmt.Errorf("Failed to call %s: %w", resp.Request.URL.String(), err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode >= 400 {
		return "", fmt.Errorf("Calling %s failed with %s", resp2.Request.URL.String(), resp.Status)
	}
	if redirects < 1 {
		return "", fmt.Errorf("Login failed, probably because of an incorrect username/password")
	}

	p, err := io.ReadAll(resp2.Body)
	if err != nil {
		return "", fmt.Errorf("Failed to read auth response: %w", err)
	}
	var token struct {
		IDToken string `json:"idToken"`
		Type    string `json:"type"`
	}
	if err := json.Unmarshal(p, &token); err != nil {
		return "", fmt.Errorf("Failed")
	}
	return token.IDToken, err

}
