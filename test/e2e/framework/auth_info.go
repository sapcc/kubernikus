package framework

import (
	"crypto/tls"
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
	provider.UserAgent.Prepend("kubernikus-e2e-tests")

	transport := &http.Transport{}
	if os.Getenv("OS_CERT") != "" && os.Getenv("OS_KEY") != "" {
		cert, err := tls.LoadX509KeyPair(os.Getenv("OS_CERT"), os.Getenv("OS_KEY"))
		if err != nil {
			return nil, fmt.Errorf("failed to load x509 keypair: %w", err)
		}
		transport.TLSClientConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS12,
		}
		provider.HTTPClient = http.Client{
			Transport: transport,
		}
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
		return "", fmt.Errorf("failed to build initial request: %w", err)
	}
	q := url.Values{}
	q.Set("connector_id", connector_id)
	req.URL.RawQuery = q.Encode()

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call %s: %w", authUrl, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("calling %s failed with %s, maybe because of an incorrect connector_id", resp.Request.URL, resp.Status)
	}
	if redirects < 1 {
		return "", fmt.Errorf("login failed, expected some redirects")
	}

	redirects = 0
	v := url.Values{}
	v.Set("login", username)
	v.Set("password", password)
	//log.Printf("Logging in using %s", resp.Request.URL.String())
	resp2, err := client.PostForm(resp.Request.URL.String(), v)
	if err != nil {
		return "", fmt.Errorf("failed to call %s: %w", resp.Request.URL.String(), err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode >= 400 {
		return "", fmt.Errorf("calling %s failed with %s", resp2.Request.URL.String(), resp.Status)
	}
	if redirects < 1 {
		return "", fmt.Errorf("login failed, probably because of an incorrect username/password")
	}

	p, err := io.ReadAll(resp2.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read auth response: %w", err)
	}
	var token struct {
		IDToken string `json:"idToken"`
		Type    string `json:"type"`
	}
	if err := json.Unmarshal(p, &token); err != nil {
		return "", fmt.Errorf("failed")
	}
	return token.IDToken, err

}
