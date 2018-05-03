package handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/sapcc/kubernikus/pkg/api/models"
)

func TestKubernikusctlDownloadLinks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, releaseResponse)
	}))
	handler := getClusterInfo{githubApiURL: server.URL}
	links, err := handler.getLinks()
	assert.NoError(t, err)
	expected := []models.Link{
		{Platform: "darwin", Link: "https://github.com/sapcc/kubernikus/releases/download/v20171115131940/kubernikusctl_darwin_amd64"},
		{Platform: "linux", Link: "https://github.com/sapcc/kubernikus/releases/download/v20171115131940/kubernikusctl_linux_amd64"},
		{Platform: "windows", Link: "https://github.com/sapcc/kubernikus/releases/download/v20171115131940/kubernikusctl_windows_amd64.exe"},
	}
	assert.Equal(t, expected, links)

}

const releaseResponse = `{
  "url": "https://api.github.com/repos/sapcc/kubernikus/releases/8526436",
  "assets_url": "https://api.github.com/repos/sapcc/kubernikus/releases/8526436/assets",
  "upload_url": "https://uploads.github.com/repos/sapcc/kubernikus/releases/8526436/assets{?name,label}",
  "html_url": "https://github.com/sapcc/kubernikus/releases/tag/v20171115131940",
  "id": 8526436,
  "tag_name": "v20171115131940",
  "target_commitish": "10d14aeeec3e0ae063fd7e83ae121b9a10de876f",
  "name": "20171115131940",
  "draft": false,
  "author": {
    "login": "sapcc-bot",
    "id": 23400221,
    "avatar_url": "https://avatars2.githubusercontent.com/u/23400221?v=4",
    "gravatar_id": "",
    "url": "https://api.github.com/users/sapcc-bot",
    "html_url": "https://github.com/sapcc-bot",
    "followers_url": "https://api.github.com/users/sapcc-bot/followers",
    "following_url": "https://api.github.com/users/sapcc-bot/following{/other_user}",
    "gists_url": "https://api.github.com/users/sapcc-bot/gists{/gist_id}",
    "starred_url": "https://api.github.com/users/sapcc-bot/starred{/owner}{/repo}",
    "subscriptions_url": "https://api.github.com/users/sapcc-bot/subscriptions",
    "organizations_url": "https://api.github.com/users/sapcc-bot/orgs",
    "repos_url": "https://api.github.com/users/sapcc-bot/repos",
    "events_url": "https://api.github.com/users/sapcc-bot/events{/privacy}",
    "received_events_url": "https://api.github.com/users/sapcc-bot/received_events",
    "type": "User",
    "site_admin": false
  },
  "prerelease": false,
  "created_at": "2017-11-15T11:43:05Z",
  "published_at": "2017-11-15T13:21:40Z",
  "assets": [
    {
      "url": "https://api.github.com/repos/sapcc/kubernikus/releases/assets/5353345",
      "id": 5353345,
      "name": "kubernikusctl_darwin_amd64",
      "label": "",
      "uploader": {
        "login": "sapcc-bot",
        "id": 23400221,
        "avatar_url": "https://avatars2.githubusercontent.com/u/23400221?v=4",
        "gravatar_id": "",
        "url": "https://api.github.com/users/sapcc-bot",
        "html_url": "https://github.com/sapcc-bot",
        "followers_url": "https://api.github.com/users/sapcc-bot/followers",
        "following_url": "https://api.github.com/users/sapcc-bot/following{/other_user}",
        "gists_url": "https://api.github.com/users/sapcc-bot/gists{/gist_id}",
        "starred_url": "https://api.github.com/users/sapcc-bot/starred{/owner}{/repo}",
        "subscriptions_url": "https://api.github.com/users/sapcc-bot/subscriptions",
        "organizations_url": "https://api.github.com/users/sapcc-bot/orgs",
        "repos_url": "https://api.github.com/users/sapcc-bot/repos",
        "events_url": "https://api.github.com/users/sapcc-bot/events{/privacy}",
        "received_events_url": "https://api.github.com/users/sapcc-bot/received_events",
        "type": "User",
        "site_admin": false
      },
      "content_type": "application/octet-stream",
      "state": "uploaded",
      "size": 6025872,
      "download_count": 2,
      "created_at": "2017-11-15T13:21:41Z",
      "updated_at": "2017-11-15T13:21:52Z",
      "browser_download_url": "https://github.com/sapcc/kubernikus/releases/download/v20171115131940/kubernikusctl_darwin_amd64"
    },
    {
      "url": "https://api.github.com/repos/sapcc/kubernikus/releases/assets/5353346",
      "id": 5353346,
      "name": "kubernikusctl_linux_amd64",
      "label": "",
      "uploader": {
        "login": "sapcc-bot",
        "id": 23400221,
        "avatar_url": "https://avatars2.githubusercontent.com/u/23400221?v=4",
        "gravatar_id": "",
        "url": "https://api.github.com/users/sapcc-bot",
        "html_url": "https://github.com/sapcc-bot",
        "followers_url": "https://api.github.com/users/sapcc-bot/followers",
        "following_url": "https://api.github.com/users/sapcc-bot/following{/other_user}",
        "gists_url": "https://api.github.com/users/sapcc-bot/gists{/gist_id}",
        "starred_url": "https://api.github.com/users/sapcc-bot/starred{/owner}{/repo}",
        "subscriptions_url": "https://api.github.com/users/sapcc-bot/subscriptions",
        "organizations_url": "https://api.github.com/users/sapcc-bot/orgs",
        "repos_url": "https://api.github.com/users/sapcc-bot/repos",
        "events_url": "https://api.github.com/users/sapcc-bot/events{/privacy}",
        "received_events_url": "https://api.github.com/users/sapcc-bot/received_events",
        "type": "User",
        "site_admin": false
      },
      "content_type": "application/octet-stream",
      "state": "uploaded",
      "size": 5648232,
      "download_count": 1,
      "created_at": "2017-11-15T13:21:52Z",
      "updated_at": "2017-11-15T13:22:17Z",
      "browser_download_url": "https://github.com/sapcc/kubernikus/releases/download/v20171115131940/kubernikusctl_linux_amd64"
    },
    {
      "url": "https://api.github.com/repos/sapcc/kubernikus/releases/assets/5353347",
      "id": 5353347,
      "name": "kubernikusctl_windows_amd64.exe",
      "label": "",
      "uploader": {
        "login": "sapcc-bot",
        "id": 23400221,
        "avatar_url": "https://avatars2.githubusercontent.com/u/23400221?v=4",
        "gravatar_id": "",
        "url": "https://api.github.com/users/sapcc-bot",
        "html_url": "https://github.com/sapcc-bot",
        "followers_url": "https://api.github.com/users/sapcc-bot/followers",
        "following_url": "https://api.github.com/users/sapcc-bot/following{/other_user}",
        "gists_url": "https://api.github.com/users/sapcc-bot/gists{/gist_id}",
        "starred_url": "https://api.github.com/users/sapcc-bot/starred{/owner}{/repo}",
        "subscriptions_url": "https://api.github.com/users/sapcc-bot/subscriptions",
        "organizations_url": "https://api.github.com/users/sapcc-bot/orgs",
        "repos_url": "https://api.github.com/users/sapcc-bot/repos",
        "events_url": "https://api.github.com/users/sapcc-bot/events{/privacy}",
        "received_events_url": "https://api.github.com/users/sapcc-bot/received_events",
        "type": "User",
        "site_admin": false
      },
      "content_type": "application/octet-stream",
      "state": "uploaded",
      "size": 5641216,
      "download_count": 1,
      "created_at": "2017-11-15T13:22:17Z",
      "updated_at": "2017-11-15T13:22:37Z",
      "browser_download_url": "https://github.com/sapcc/kubernikus/releases/download/v20171115131940/kubernikusctl_windows_amd64.exe"
    }
  ],
  "tarball_url": "https://api.github.com/repos/sapcc/kubernikus/tarball/v20171115131940",
  "zipball_url": "https://api.github.com/repos/sapcc/kubernikus/zipball/v20171115131940",
  "body": ""
}`
