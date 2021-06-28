package flatcar

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"sync"
	"time"

	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"

	"github.com/sapcc/kubernikus/pkg/util/version"
)

const (
	stable channel = "stable"
	beta   channel = "beta"
	alpha  channel = "alpha"
)

var (
	now = time.Now

	flatcarVersionBaseURL       = "https://%s.release.flatcar-linux.net/amd64-usr/current/version.txt"
	flatcarVersionRE            = regexp.MustCompile(`FLATCAR_VERSION=(.+)`)
	flatcarVersionIdentifierRE  = regexp.MustCompile(`(\d+\.\d+\.\d+)`)
	flatcarVersionFetchInterval = 1 * time.Hour
)

type channel string

// Version is a helper that fetches and caches flatcar versions
type Version struct {
	Client    *http.Client
	versions  map[channel]*version.Version
	fetchedAt map[channel]time.Time

	mu sync.Mutex
}

// Stable returns version of flatcar stable channel
func (d *Version) Stable() (*version.Version, error) {
	return d.latest(stable)
}

// Beta returns version of flatcar beta channel
func (d *Version) Beta() (*version.Version, error) {
	return d.latest(beta)
}

// Alpha returns version of flatcar alpha channel
func (d *Version) Alpha() (*version.Version, error) {
	return d.latest(alpha)
}

// IsNodeUptodate checkes whether a Kubernetes Node is a flatcar that needs updating
func (d *Version) IsNodeUptodate(node *v1.Node) (bool, error) {
	var availableVersion, nodeVersion *version.Version
	var err error

	availableVersion, err = d.Stable()
	if err != nil {
		return false, errors.Wrap(err, "flatcar version couldn't be retrieved.")
	}

	nodeVersion, err = ExractVersion(node)
	if err != nil {
		return false, err
	}

	return nodeVersion.AtLeast(availableVersion), nil
}

// ExractVersion returns a semantic version of the node
func ExractVersion(node *v1.Node) (*version.Version, error) {
	match := flatcarVersionIdentifierRE.FindSubmatch([]byte(node.Status.NodeInfo.OSImage))
	if len(match) < 2 {
		return nil, fmt.Errorf("Couldn't match flatcar version from NodeInfo.OSImage: %s", node.Status.NodeInfo.OSImage)
	}

	nodeVersion, err := version.ParseSemantic(string(match[1]))
	if err != nil {
		return nil, errors.Wrapf(err, "Node version can't be parsed from %s", match[1])
	}

	return nodeVersion, nil
}

func (d *Version) latest(c channel) (*version.Version, error) {
	var err error
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.Client == nil {
		d.Client = &http.Client{
			Timeout: time.Second * 10,
		}
	}

	if d.versions == nil {
		d.versions = make(map[channel]*version.Version)
	}

	if d.fetchedAt == nil {
		d.fetchedAt = make(map[channel]time.Time)
	}

	if d.versions[c] == nil || now().After(d.fetchedAt[c].Add(flatcarVersionFetchInterval)) {
		d.versions[c], err = d.fetch(c)
		if err != nil {
			return nil, errors.Wrapf(err, "Couldn't fetch latest %s version", c)
		}
	}

	return d.versions[c], nil
}

func (d *Version) fetch(c channel) (*version.Version, error) {
	r, err := d.Client.Get(fmt.Sprintf(flatcarVersionBaseURL, c))
	if err != nil {
		return nil, fmt.Errorf("Couldn't fetch flatcar version: %s", err)
	}

	defer r.Body.Close()

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("Couldn't read flatcar version: %s", err)
	}

	v := flatcarVersionRE.FindSubmatch(body)
	if len(v) < 2 {
		return nil, fmt.Errorf("Couldn't parse flatcar version: %s", err)
	}

	d.fetchedAt[c] = now()

	return version.ParseSemantic(string(v[1]))
}
