package coreos

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
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

	coreOSVersionBaseURL       = "https://%s.release.core-os.net/amd64-usr/current/version.txt"
	coreOSVersionRE            = regexp.MustCompile(`COREOS_VERSION=(.+)`)
	coreOSVersionIdentifierRE  = regexp.MustCompile(`(\d+\.\d+\.\d+)`) // Container Linux by CoreOS 1800.6.0 (Rhyolite)
	coreOSVersionFetchInterval = 1 * time.Hour
)

type channel string

// LatestCoreOSVersion is a helper that fetches and caches CoreOS versions
type Version struct {
	Client    *http.Client
	versions  map[channel]*version.Version
	fetchedAt map[channel]time.Time
}

// Stable returns version of CoreOS stable channel
func (d *Version) Stable() (*version.Version, error) {
	return d.latest(stable)
}

// Beta returns version of CoreOS beta channel
func (d *Version) Beta() (*version.Version, error) {
	return d.latest(beta)
}

// Alpha returns version of CoreOS alpha channel
func (d *Version) Alpha() (*version.Version, error) {
	return d.latest(alpha)
}

// IsNodeUptodate checkes whether a Kubernetes Node is a CoreOS that needs updating
func (d *Version) IsNodeUptodate(node *v1.Node) (bool, error) {
	var availableVersion, nodeVersion *version.Version
	var err error

	availableVersion, err = d.Stable()
	if err != nil {
		return false, errors.Wrap(err, "CoreOS version couldn't be retrieved.")
	}

	match := coreOSVersionIdentifierRE.FindSubmatch([]byte(node.Status.NodeInfo.OSImage))
	if len(match) < 2 {
		return false, fmt.Errorf("Couldn't match CoreOS version from NodeInfo.OSImage: %s", node.Status.NodeInfo.OSImage)
	}

	nodeVersion, err = version.ParseSemantic(string(match[1]))
	if err != nil {
		return false, errors.Wrapf(err, "Node version can't be parsed from %s", match[1])
	}

	return nodeVersion.AtLeast(availableVersion), nil
}

func (d *Version) latest(c channel) (*version.Version, error) {
	var err error

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

	if d.versions[c] == nil || now().After(d.fetchedAt[c].Add(coreOSVersionFetchInterval)) {
		d.versions[c], err = d.fetch(c)
		if err != nil {
			return nil, errors.Wrapf(err, "Couldn't fetch latest %s version", c)
		}
	}

	return d.versions[c], nil
}

func (d *Version) fetch(c channel) (*version.Version, error) {
	r, err := d.Client.Get(fmt.Sprintf(coreOSVersionBaseURL, c))
	if err != nil {
		return nil, fmt.Errorf("Couldn't fetch CoreOS version: %s", err)
	}

	defer r.Body.Close()

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("Couldn't read CoreOS version: %s", err)
	}

	v := coreOSVersionRE.FindSubmatch(body)
	if len(v) < 2 {
		return nil, fmt.Errorf("Couldn't parse CoreOS version: %s", err)
	}

	d.fetchedAt[c] = now()

	return version.ParseSemantic(string(v[1]))
}
