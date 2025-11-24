package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/databus23/requestutil"
	"github.com/go-openapi/runtime/middleware"

	"github.com/sapcc/kubernikus/pkg/api"
	"github.com/sapcc/kubernikus/pkg/api/auth"
	"github.com/sapcc/kubernikus/pkg/api/models"
	"github.com/sapcc/kubernikus/pkg/api/rest/operations"
	"github.com/sapcc/kubernikus/pkg/version"
)

func NewGetClusterInfo(rt *api.Runtime) operations.GetClusterInfoHandler {
	return &getClusterInfo{Runtime: rt, githubApiURL: "https://api.github.com"}
}

type getClusterInfo struct {
	*api.Runtime
	links        []models.Link
	linkMutex    sync.Mutex
	githubApiURL string
}

func (d *getClusterInfo) Handle(params operations.GetClusterInfoParams, principal *models.Principal) middleware.Responder {

	links, err := d.getLinks()
	if err != nil {
		return NewErrorResponse(&operations.GetClusterInfoDefault{}, 500, "%s", err)
	}

	baseURL := fmt.Sprintf("%s://%s", requestutil.Scheme(params.HTTPRequest), requestutil.HostWithPort(params.HTTPRequest))

	command := []string{
		"kubernikusctl", "auth", "init",
		"--username", principal.Name,
		"--user-domain-name", principal.Domain,
		"--project-id", principal.Account,
		"--auth-url", auth.OpenStackAuthURL(),
		"--url", baseURL,
		"--name", params.Name,
	}

	info := &models.KlusterInfo{
		SetupCommand: strings.Join(command, " "),
		Binaries: []models.Binaries{
			{
				Name:  "kubernikusctl",
				Links: links,
			},
		},
	}
	return operations.NewGetClusterInfoOK().WithPayload(info)
}

func (d *getClusterInfo) getLinks() ([]models.Link, error) {
	d.linkMutex.Lock()
	defer d.linkMutex.Unlock()
	if d.links != nil {
		return d.links, nil
	}

	release := "latest"
	if version.GitCommit != "HEAD" {
		release = fmt.Sprintf("v%s+%s", version.VERSION, version.GitCommit)
	}
	resp, err := http.Get(fmt.Sprintf("%s/repos/sapcc/kubernikus/releases/tags/%s", d.githubApiURL, release))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch release %s: %s", release, err)
	}
	//Fall back to latest relese if the specific release is not found
	if resp.StatusCode == 404 {
		resp.Body.Close()
		resp, err = http.Get(fmt.Sprintf("%s/repos/sapcc/kubernikus/releases/latest", d.githubApiURL))
		if err != nil {
			return nil, fmt.Errorf("failed to fetch latest release: %s", err)
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("failed to fetch release %s: %s", release, resp.Status)
	}
	var releaseResponse struct {
		Assets []struct {
			Name        string
			DownloadURL string `json:"browser_download_url"`
		}
	}
	if err := json.NewDecoder(resp.Body).Decode(&releaseResponse); err != nil {
		return nil, err
	}
	links := make([]models.Link, 0, 3)
	for _, asset := range releaseResponse.Assets {
		link := models.Link{Link: asset.DownloadURL}
		switch {
		case strings.Contains(asset.Name, "darwin"):
			link.Platform = "darwin"
		case strings.Contains(asset.Name, "linux"):
			link.Platform = "linux"
		case strings.Contains(asset.Name, "windows"):
			link.Platform = "windows"
		default:
			//skip unknown assets
			continue
		}
		links = append(links, link)
	}
	if len(links) == 0 {
		return nil, fmt.Errorf("no downloads found for release %s", release)
	}
	d.links = links
	return links, nil

}
